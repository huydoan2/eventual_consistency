package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"

	"github.com/huydoan2/eventual_consistency/vectorclock"
)

const masterPort int64 = 3000
const baseClientPort int64 = 5000
const baseServerPort int64 = 5000
const serverPortRange int64 = 10
const LOGDIR = "log"

// global variables and structures
var id int64
var idStr string
var RPCclients = make(map[int64]*rpc.Client) //store client struct for each connection
var RPCserver *rpc.Server

// key-value store
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
	c.data[p.key] = Value{p.val, p.clock}
}

var cache Cache
var data = make(map[string]Value)

var vClock vectorclock.VectorClock

/*
	RPC
*/

// ServerService : RPC type for server services
type ServerService int //temporary type

// BreakConnection : RPC to break connection between servers
//					: Reply 0 if conn existed and closed, 1 if never existed
func (serverService *ServerService) BreakConnection(serverID *int64, reply *int64) error {
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
func (serverService *ServerService) CreateConnection(serverID *int64, reply *int64) error {
	debug(id, fmt.Sprintf("Creating connection to Server[%d]...", *serverID))

	if _, ok := RPCclients[*serverID]; !ok {
		serverPort := strconv.FormatInt(baseServerPort+(*serverID), 10)
		client, err := rpc.Dial("tcp", "localhost:"+serverPort)
		if err == nil {
			RPCclients[*serverID] = client
			debug(id, fmt.Sprintf("Finished joining server[%d] to server[%d]\n", id, *serverID))
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

// ConnectAsClient : RPC call to connect to the target server as a client
func (ss *ServerService) ConnectAsClient(targetID *int64, reply *int64) error {
	debug(id, fmt.Sprint("Connecting as client to Server[%d]...", *targetID))

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

// Cleanup: Function to cleanup before murder
func (ss *ServerService) Cleanup(targetID *int64, reply *int64) error {
	debug(id, "Cleaning up before being terminated...")

	for k, v := range RPCclients {
		v.Close()
		delete(RPCclients, k)
	}
	debug(id, "Cleanup complete. Prepare to die")
	return nil
}

// PrintStore: RPC returns to "client" the key-value store without the time information
// reply: the memory will be allocated by the function. User only needs to provide pointer
func (ss *ServerService) PrintStore(notUse *int64, reply *map[string]string) error {
	ret := make(map[string]string)

	for k, v := range data {
		ret[k] = v.val
	}

	reply = &ret

	return nil
}

func (ss *ServerService) Put(clientReq *Payload, serverResp *Payload) error {

	vClock.Update(&clientReq.clock)
	vClock.Increment(id)
	serverResp.clock = vClock

	val, ok := data[clientReq.key]
	if ok {
		currClock := val.clock
		cmp := currClock.Compare(&clientReq.clock)
		if cmp == vectorclock.LESS {
			data[clientReq.key] = Value{clientReq.val, clientReq.clock}
		} else {
			serverResp.key = clientReq.key
			serverResp.val = val.val
			serverResp.valTime = val.clock
		}
	} else {
		data[clientReq.key] = Value{clientReq.val, clientReq.clock}
	}

	return nil
}

/*******************************************************/

func connectToServers() {
	debug(id, "Connecting to other available servers ...")

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

func CreateLogDir(dir string) {
	if _, err := os.Stat("dir"); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func InitLogger() {
	//CreateLogDir("../log")

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

	vClock.Id = id

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
	fmt.Printf("Server process %s started\n", os.Args[1])
	id, _ = strconv.ParseInt(os.Args[1], 10, 64) // get id from command line
	idStr = os.Args[1]

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
