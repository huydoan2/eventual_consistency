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

// key-value store cache
type Value struct {
	val   string
	clock vectorclock.VectorClock
}

type Payload struct {
	key     string
	val     string
	valTime vectorclock.VectorClock
	clock   vectorclock.VectorClock // current clock of the process
}

// Cache class
type Cache struct {
	data map[string]Value
}

func (c *Cache) Invalidate() {

}

func (c *Cache) Insert(p *Payload) {
	c.data[p.key] = Value{p.val, p.valTime}
}

//var cache = make(map[string]Value)
var cache Cache
var vClock vectorclock.VectorClock

/*
	RPC
*/

// ClientService : RPC type for client services
type ClientService int //temporary type

type PutData struct {
	key, value string
}

// BreakConnection : RPC to break connection between client and server with id
//					: Reply 0 if conn existed and closed, 1 if never existed
func (clientService *ClientService) BreakConnection(serverID *int64, reply *int64) error {
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
func (clientService *ClientService) CreateConnection(serverID *int64, reply *int64) error {
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
func (clientServerce *ClientService) Put(putData *PutData, reply *int64) error {

	debug(id, fmt.Sprintf("Putting %s:%s ...", putData.key, putData.value))

	// Check if the client is connected to any server
	length := len(RPCclients)
	if length == 0 {
		return errors.New("Client does not connect to any servers")
	}

	// Get a random position in the server set
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	serverPos := r.Int63n(int64(length))
	var server *rpc.Client
	var i int64
	// A bit complex to get a random server
	for _, server = range RPCclients {
		if i == serverPos {
			break
		} else {
			i++
		}
	}

	var data Payload
	data.key = putData.key
	data.val = putData.value
	vClock.Increment(id)
	data.clock = vClock

	// cache the put request
	cache.Insert(&data)

	var serverResp Payload
	// We have a server now, put data to it
	err := server.Call("ServerService.Put", &data, &serverResp) // TODO: need to support if server fails in the middle

	if err != nil {
		debug(id, err.Error())
		return err
	}

	vClock.Update(&serverResp.clock)

	if serverResp.key != "" {
		cache.Insert(&serverResp)
	}

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

	// Init VectorClock
	vClock.Id = id

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

func main() {
	fmt.Printf("Client process %s started\n", os.Args[1])
	id, _ = strconv.ParseInt(os.Args[1], 10, 64) // get id from command line
	idStr = os.Args[1]

	serverID, _ := strconv.ParseInt(os.Args[2], 10, 64) // get server id from command line

	Init(serverID)

	for {

	}
}
