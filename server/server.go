package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"sync"

	"../cache"
	"../vectorclock"
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

var sCache *cache.Cache
var data = make(map[string]cache.Value)

var vClock vectorclock.VectorClock
var lockInTree sync.Mutex
var bIntree bool
var lockCache sync.Mutex
var listChild []*rpc.Client

/*
	RPC
*/

// ServerService : RPC type for server services
type ServerService int //temporary type

// StabilizePayload : RPC type for transporting cache data in Stabilize
type StabilizePayload struct {
	dummy   string
	IsChild bool
	Data    map[string]cache.Value
	// Cache   cache.Cache
}

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
	debug(id, fmt.Sprintf("Connecting as client to Server[%d]...", *targetID))

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

// Cleanup : Function to cleanup before murder
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
	//ret := make(map[string]string)

	debug(id, "Printing Store now")
	for k, v := range data {
		(*reply)[k] = v.Val
		debug(id, fmt.Sprintf("P %s: %s", k, (*reply)[k]))
	}
	//reply = &ret

	return nil
}

// Put RPC to respond to a Put request from the client
func (ss *ServerService) Put(clientReq *cache.Payload, serverResp *cache.Payload) error {
	debug(id, fmt.Sprintf("Starting put %s:%s ...", (*clientReq).Key, (*clientReq).Val))

	vClock.Update(&clientReq.Clock)
	vClock.Increment(id)
	serverResp.Clock = vClock
	update := 0

	debug(id, fmt.Sprintf("Client Clock: %s", clientReq.Clock.ToString()))
	val, ok := data[clientReq.Key]
	if ok {
		currClock := val.Clock
		cmp := currClock.Compare(&clientReq.Clock)
		debug(id, fmt.Sprintf("Current Clock: %s", currClock.ToString()))
		if cmp == vectorclock.LESS {
			data[clientReq.Key] = cache.Value{Val: clientReq.Val, Clock: clientReq.Clock}
			update = 1
		} else {
			debug(id, "Record not updated")
			serverResp.Key = clientReq.Key
			serverResp.Val = val.Val
			serverResp.ValTime = val.Clock
		}
	} else {
		data[clientReq.Key] = cache.Value{Val: clientReq.Val, Clock: clientReq.Clock}
		update = 1
	}

	if update == 1 {
		sCache.Insert(clientReq)
		debug(id, "Record updated")
		temp := sCache.Data[clientReq.Key].Clock

		debug(id, fmt.Sprintf("sCache Clock: %s", temp.ToString()))
	}

	return nil
}

// Get RPC respond to Get request from the client
func (ss *ServerService) Get(clientReq *cache.Payload, serverResp *cache.Payload) error {
	debug(id, "Starting get...")

	// Clock Update
	vClock.Update(&clientReq.Clock)
	vClock.Increment(id)
	serverResp.Clock = vClock

	// Check if it exists in data. If not return ERR_KEY
	val, ok := data[clientReq.Key]
	if ok {
		serverResp.Key = clientReq.Key
		serverResp.Val = val.Val
		serverResp.ValTime = val.Clock
	} else {
		serverResp.Key = clientReq.Key
		serverResp.Val = "ERR_KEY"
	}

	return nil
}

// Order : update sCache only when updateData is false, otherwise update both sCache and the DataStore
func Order(otherData *map[string]cache.Value, updateData bool) error {
	debug(id, "Ordering ...")
	for k, v := range *otherData {
		debug(id, fmt.Sprintf("Entry is %s: %s, %s", k, v.Val, v.Clock.ToString()))
		if myEntry, ok := sCache.Data[k]; ok {
			debug(id, fmt.Sprintf("Compare myEntry: %s, %s and newEntry: %s, %s", myEntry.Val, myEntry.Clock.ToString(), v.Val, v.Clock.ToString()))
			if myEntry.Clock.Compare(&v.Clock) == vectorclock.GREATER {
				debug(id, "newEntry is Greater and will update")
				sCache.Data[k] = v
				debug(id, fmt.Sprintf("Update Cache on order: %s:%s", k, v.Val))

			}
		} else {
			debug(id, fmt.Sprintf("Insert Cache on order: %s:%s", k, v.Val))
			sCache.Data[k] = v
		}
	}
	if updateData {
		for k, v := range sCache.Data {
			data[k] = v
			debug(id, fmt.Sprintf("Update DataStore on order: %s:%s", k, v.Val))
		}
	}
	return nil
}

// Gather RPC converge cast, form MST, gather cache data to the root node
func (ss *ServerService) Gather(arg *int64, reply *StabilizePayload) error {
	//var err error
	// var lockCounter sync.Mutex

	lockInTree.Lock()
	debug(id, fmt.Sprintf("%d is checking if it is parent", *arg))
	if bIntree == true {
		reply.IsChild = false
		debug(id, fmt.Sprintf("%d is not parent. Returning", *arg))
		lockInTree.Unlock()
		return nil
	}
	bIntree = true
	lockInTree.Unlock()

	var wg sync.WaitGroup

	debug(id, fmt.Sprintf("Gathering... called by Server[%d]", *arg))
	debug(id, fmt.Sprintf("Now call gather on %d servers", len(RPCclients)))
	for server_id, server := range RPCclients {
		if server_id == *arg {
			continue
		}
		wg.Add(1)
		go func(server *rpc.Client, server_id int64, wg *sync.WaitGroup) {

			defer wg.Done()
			var response StabilizePayload
			response.dummy = "DUMMY"
			response.Data = make(map[string]cache.Value)
			response.IsChild = false

			debug(id, fmt.Sprintf("Calling gather from %d on %d", *arg, server_id))
			err := server.Call("ServerService.Gather", &id, &response)

			debug(id, fmt.Sprintf("Returned from Gather on %d", server_id))
			for k, v := range response.Data {
				debug(id, fmt.Sprintf("%s: %s", k, v.Val))
			}
			debug(id, fmt.Sprintf("respond: %t", response.IsChild))

			if err != nil {
				debug(id, fmt.Sprintf("Error: %s", err.Error()))
				return
			}

			if response.IsChild == true {
				lockCache.Lock()
				defer lockCache.Unlock()
				Order(&response.Data, false)
				listChild = append(listChild, server)
				//lockCache.Unlock()
				debug(id, fmt.Sprintf("Append to childList"))
			}

			debug(id, fmt.Sprintf("Leaving goroutine for gather on %d", server_id))

			return
		}(server, server_id, &wg)

	}

	// Wait till all of other connected servers reply
	debug(id, "Waiting to sync threads")
	wg.Wait()

	debug(id, "Copying cache ...")
	reply.IsChild = true
	reply.Data = sCache.Data
	//reply.Data = make(map[string]cache.Val)
	//debug(id, fmt.Sprintf("Finished Copying and size of cache is %d ...", len(sCache.Data)))

	debug(id, "Now printing cache ...")
	for k, v := range sCache.Data {
		debug(id, fmt.Sprintf("Cache %s: %s", k, v.Val))
	}

	debug(id, "Now printing reply ...")
	for k, v := range reply.Data {
		debug(id, fmt.Sprintf("Reply %s: %s", k, v.Val))
	}
	return nil
}

func (ss *ServerService) Scatter(arg *StabilizePayload, reply *int64) error {

	var wg sync.WaitGroup

	for server_id, server := range listChild {
		wg.Add(1)
		go func(server *rpc.Client, server_id int, wg *sync.WaitGroup) {
			defer wg.Done()
			debug(id, fmt.Sprintf("Scatter to %d from %d", server_id, arg))
			var dummyReply int64
			err := server.Call("ServerService.Scatter", arg, &dummyReply)
			if err != nil {
				debug(id, fmt.Sprintf("Scatter failed with %v", err))
				return
			}
		}(server, server_id, &wg)
	}

	wg.Wait()

	Order(&arg.Data, true)
	*reply = 1

	return nil
}

// InitStabilize starts the Stabilize algorithm. This server is the root of the MST
func (ss *ServerService) InitStabilize(arg *int64, reply *int64) error {
	debug(id, "Start stabilizing as root")
	var response StabilizePayload
	response.dummy = "DUMMY"
	response.Data = make(map[string]cache.Value)
	response.IsChild = false
	//bIntree = true
	debug(id, "Beginning gather ...")
	errGather := ss.Gather(&id, &response)
	debug(id, "Gather complete ...")
	if errGather != nil {
		debug(id, fmt.Sprintf("Gather failed with %v", errGather))
		return errGather
	}
	response.Data = sCache.Data
	var dummyReply int64
	debug(id, "Beginning scatter ...")
	errScatter := ss.Scatter(&response, &dummyReply)
	debug(id, "Scatter complete ...")
	if errScatter != nil {
		debug(id, fmt.Sprintf("Scatter failed with %v", errScatter))
		return errScatter
	}

	// Update the datastore
	go func() {
		for k, v := range sCache.Data {
			data[k] = v
		}
		//sCache.Invalidate()
	}()

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

//var lockDebug sync.Mutex

func debug(id int64, msg string) {
	//lockDebug.Lock()
	//defer lockDebug.Unlock()
	logger.Printf("Server[%d]: %s", id, msg)
}

func Init() {

	InitLogger()
	debug(id, "Starting RPC server ...\n")

	// Init vector clock id
	vClock.Id = id

	// Init cache
	sCache = cache.New()

	bIntree = false

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
