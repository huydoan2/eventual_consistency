package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"time"

	"../cache"
	"../vectorclock"
)

const masterPort int64 = 3000
const baseClientPort int64 = 5000
const baseServerPort int64 = 5000
const clientPortRange int64 = 10
const LOGDIR = "log"

// global variables and structures
var id int64
var idStr string
var RPCclients = make(map[int64]*rpc.Client) //store client struct for each connection
var RPCserver *rpc.Server

var cCache *cache.Cache            // Client cache
var vClock vectorclock.VectorClock // local vector clock
var versionNumber int64

/*
	RPC
*/

// ClientService : RPC type for client services
type ClientService int //temporary type

type PutData struct {
	Key, Value string
}

// BreakConnection : RPC to break connection between client and server with id
//					: Reply 0 if conn existed and closed, 1 if never existed
func (cs *ClientService) BreakConnection(serverID *int64, reply *int64) error {
	debug(id, fmt.Sprintf("Breaking connection to Server[%d]...", *serverID))

	if client, ok := RPCclients[*serverID]; ok {
		client.Close()
		debug(id, fmt.Sprintf("Connection to server[%d] is broken successfully", *serverID))
		delete(RPCclients, *serverID)
		*reply = 0
	} else {
		debug(id, fmt.Sprintf("Tried to break connection to server[%d] but was already broken", *serverID))
		*reply = 1
	}
	return nil
}

// CreateConnection : RPC to create connection between client and server with id
//					: Reply 0 if conn existed and created, 1 if never existed
func (cs *ClientService) CreateConnection(serverID *int64, reply *int64) error {
	if _, ok := RPCclients[*serverID]; !ok {
		serverPort := strconv.FormatInt(baseServerPort+(*serverID), 10)
		client, err := rpc.Dial("tcp", "localhost:"+serverPort)
		if err == nil {
			RPCclients[*serverID] = client
			debug(id, fmt.Sprintf("Finished joining client[%d] to server[%d]\n", id, *serverID))
		} else {
			debug(id, err.Error())
			panic(err)
		}
		debug(id, fmt.Sprintf("Connection to server[%d] is created successfully", *serverID))
		RPCclients[*serverID] = client
		*reply = 0
	} else {
		debug(id, fmt.Sprintf("Tried to create connection to server[%d] but was already created", *serverID))
		*reply = 1
	}
	return nil
}

// Put: RPC to put key:value to a server
func (cs *ClientService) Put(putData *PutData, reply *int64) error {

	debug(id, fmt.Sprintf("Putting %s:%s ...", putData.Key, putData.Value))

	// Check if the client is connected to any server
	length := len(RPCclients)
	if length == 0 {
		return errors.New("Client does not connect to any servers")
	}

	server := getRandomServer()

	var serverVersion int64
	errVersion := server.Call("ServerService.GetVersionNumber", &id, &serverVersion)
	if errVersion != nil {
		debug(id, fmt.Sprintf("Failed to get version number from server"))
		return errVersion
	}

	if serverVersion > versionNumber {
		cCache.Invalidate()
		versionNumber = serverVersion
	}

	var data cache.Payload
	data.Key = putData.Key
	data.Val = putData.Value
	vClock.Increment(id)
	data.ValTime = vClock
	data.Clock = vClock

	cCache.Insert(&data)

	var serverResp cache.Payload
	// We have a server now, put data to it
	debug(id, "Calling Put RPC from server")
	err := server.Call("ServerService.Put", &data, &serverResp) // TODO: need to support if server fails in the middle

	if err != nil {
		debug(id, err.Error())
		return err
	}

	vClock.Update(&serverResp.Clock)

	if serverResp.Key != "" {
		cCache.Insert(&serverResp)
	}

	return nil
}

// Get: RPC to querry the value of a key
func (cs *ClientService) Get(key *string, reply *string) error {

	// If not, query a server for the key
	server := getRandomServer()

	var serverVersion int64
	errVersion := server.Call("ServerService.GetVersionNumber", &id, &serverVersion)
	if errVersion != nil {
		debug(id, fmt.Sprintf("Failed to get version number from server"))
		return errVersion
	}

	if serverVersion > versionNumber {
		cCache.Invalidate()
		versionNumber = serverVersion
	}

	val, ok := cCache.Find(key)

	// Found the entry in the cache
	if ok {
		*reply = val.Val
		return nil
	}

	var data cache.Payload
	arg := cache.Payload{Key: *key, Clock: vClock}

	err := server.Call("ServerService.Get", &arg, &data)
	if err != nil {
		// Error with RPC call or from the server
		s := fmt.Sprintf("Failed to communicate with server\nError: %v", err)
		debug(id, s)
		return errors.New(s)
	}

	// RPC succeeded, sync time
	vClock.Update(&data.Clock)

	// Check the replied data from the server
	if data.Val == "ERR_KEY" {
		debug(id, fmt.Sprintf("%s:ERR_KEY", *key))
		*reply = "ERR_KEY"
		return nil
	}

	// Server has the value, update the cache
	cCache.Insert(&data)

	//return value to the master
	*reply = data.Val

	return nil
}

// InvalidateCache RPC to invalidate client's cache. Used for testing
func (cs *ClientService) InvalidateCache(arg *int64, reply *int64) error {
	cCache.Invalidate()
	return nil
}

/*******************************************************/

var logger *log.Logger

func InitLogger() {
	f, err := os.OpenFile("../log/client"+idStr, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	//defer f.Close()
	logger = log.New(f, "", 0)
	// log.SetOutput(f)
}

func debug(id int64, msg string) {
	logger.Printf("Client[%d]: %s", id, msg)
}

func Init(serverId int64) {

	InitLogger()
	debug(id, "Starting RPC server ...\n")

	// Init versionNumber
	versionNumber = 0

	// Init VectorClock
	vClock.Id = id

	// Init Cache
	cCache = cache.New()

	// Connect to serverId
	tmp := fmt.Sprintf("Connecting to Server[%d]", serverId)
	debug(id, tmp)

	// Connect to server ID
	serverPort := strconv.FormatInt(baseServerPort+serverId, 10)
	client, err := rpc.Dial("tcp", "localhost:"+serverPort)
	if err == nil {
		RPCclients[serverId] = client
		debug(id, fmt.Sprintf("Finished joining client[%d] to server[%d]\n", id, serverId))
	} else {
		debug(id, err.Error())
		panic(err)
	}

	// Register RPC server
	//RPCserver = rpc.NewServer()
	clientService := new(ClientService)
	rpc.Register(clientService)

	clientPort := strconv.FormatInt(baseClientPort+id, 10)
	RPCclientConn, err := net.Listen("tcp", ":"+clientPort)
	if err != nil {
		debug(id, "Cannot start RPC server\nProcess terminated!\n")
		panic(err)
	}

	// Need to check for correctness of the Accept(). Assume if the client hangs up, Accept() returns
	go rpc.Accept(RPCclientConn)

	debug(id, "Initialization finished!\n")
}

func getRandomServer() *rpc.Client {
	length := len(RPCclients)
	// Get a random position in the server set
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	serverPos := r.Int63n(int64(length))
	var server *rpc.Client
	var i int64
	var serverID int64

	// A bit complex to get a random server
	for serverID, server = range RPCclients {
		if i == serverPos {
			break
		} else {
			i++
		}
	}
	debug(id, fmt.Sprintf("Chosen server is %d", serverID))
	return server
}

func main() {
	fmt.Printf("Client process %s started\n", os.Args[1])
	id, _ = strconv.ParseInt(os.Args[1], 10, 64) // get id from command line
	idStr = os.Args[1]

	serverID, _ := strconv.ParseInt(os.Args[2], 10, 64) // get server id from command line

	Init(serverID)

	for {

	}
}
