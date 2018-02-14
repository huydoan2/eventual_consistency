package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
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
var cache = make(map[string]string)

/*
	RPC
*/

// ClientService : RPC type for client services
type ClientService int //temporary type

// BreakConnection : RPC to break connection between client and server with id
//					: Reply 0 if conn existed and closed, 1 if never existed
func (clientService *ClientService) BreakConnection(serverID *int64, reply *int64) error {
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
