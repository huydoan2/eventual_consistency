package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
)

const masterPort int64 = 3000
const baseServerPort int64 = 5000
const serverPortRange int64 = 10
const LOGDIR = "log"

// global variables and structures
var id int64
var idStr string
var RPCclients = make(map[int64]*rpc.Client) //store client struct for each connection
var RPCserver *rpc.Server

/*
	RPC
*/

// ServerService : RPC type for server services
type ServerService int //temporary type

// ConnectAsClient : RPC call to connect to the target server as a client
func (ss *ServerService) ConnectAsClient(targetID *int64, reply *int64) error {
	if *targetID < 0 || *targetID >= serverPortRange {
		return errors.New("Server id is out of range")
	}

	targetPort := strconv.FormatInt(*targetID+baseServerPort, 10)
	client, err := rpc.Dial("tcp", "localhost:"+targetPort)
	if err != nil {
		// Cannot connect to the target server
		errorMsg := fmt.Sprintf("Cannot connect to server %d", *targetID)
		return errors.New(errorMsg)
	}

	// Sucessfully connected to the target server
	RPCclients[*targetID] = client // store the client handler
	*reply = 1
	return nil
}

/*******************************************************/

func connectToServers() {
	var count int64
	for i := int64(0); i < serverPortRange; i++ {
		if i == id {
			continue
		}
		targetPort := strconv.FormatInt(i+baseServerPort, 10)
		client, err := rpc.Dial("tcp", "localhost:"+targetPort)
		if err == nil {
			// Succesffuly connected
			debug(id, "Connected to "+targetPort)
			RPCclients[i] = client // store the client handler
			// now call the rpc of the target server to connect to me
			var reply int64
			err = client.Call("ServerService.ConnectAsClient", &id, &reply)
			if err == nil {
				count++
			} else {
				debug(id, err.Error())
			}
		}
	}
	temp := fmt.Sprintf("connected with %d other server(s)\n", count)
	debug(id, temp)
}

var logger *log.Logger

func InitLogger() {
	f, err := os.OpenFile("../log/server"+idStr, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	//defer f.Close()
	logger = log.New(f, "", 0)
	// log.SetOutput(f)
}

func debug(id int64, msg string) {
	logger.Printf("Server[%d]: %s", id, msg)
}

func Init() {

	InitLogger()
	debug(id, "Starting RPC server ...\n")

	// Register RPC server
	//RPCserver = rpc.NewServer()
	serverService := new(ServerService)
	rpc.Register(serverService)

	serverPort := strconv.FormatInt(baseServerPort+id, 10)
	RPCserverConn, err := net.Listen("tcp", ":"+serverPort)
	if err != nil {
		debug(id, "Cannot start RPC server\nProcess terminated!\n")
		panic(err)
	}

	// Need to check for correctness of the Accept(). Assume if the client hangs up, Accept() returns
	go rpc.Accept(RPCserverConn)

	// Connect to other servers and ask them to connect to me
	debug(id, "Connecting to other servers ...\n")
	connectToServers()

	debug(id, "Initialization finished!\n")

	// fmt.Printf("Connect to the master\n")
	// masterPort := strconv.FormatInt(masterPort, 10)
	// masterConn, masterErr := net.Listen("tcp", ":"+masterPort)
	// if masterErr != nil {
	// 	log(id, "Cannot connect to the master\nProcess terminated!\n")
	// 	panic(masterErr)
	// }

	// Seperate goroutine to handle request from master
	//go rpc.Accept(masterConn)
}

func main() {
	id, _ = strconv.ParseInt(os.Args[1], 10, 64) // get id from command line
	idStr = os.Args[1]
	fmt.Printf("Server process %d started\n", id)

	Init()

	for {

	}
	/*
		var stop = make(chan bool)
		arith := new(Arith)
		rpc.Register(arith)
		rpc.HandleHTTP()
		l, e := net.Listen("tcp", "localhost:1234")
		if e != nil {
			log.Fatal("listen error:", e)
		}
		go http.Serve(l, nil)
		<-stop
	*/

}