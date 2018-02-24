package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/rpc"
	"os/exec"
	"strconv"
	"sync/atomic"
	"time"
)

var baseServerPort int64 = 5000
var baseClientPort int64 = 5000
var servers = make(map[int64]*rpc.Client)     // map[server id][server rpc handler]
var clients = make(map[int64]*rpc.Client)     // map[server id][client rpc handler]
var serverProcess = make(map[int64]*exec.Cmd) // map[server id][server procees]
var clientProcess = make(map[int64]*exec.Cmd) // map[client id][client process]

type PutData struct {
	Key, Value string
}

func ExecServer(id int64) {
	server := exec.Command("../server/server", strconv.FormatInt(id, 10))
	serverProcess[id] = server
	serverErr := server.Start()
	//fmt.Printf("%s\n", serverOut)
	if serverErr != nil {
		panic(serverErr)
	}
	exitCode := server.Wait()
	fmt.Printf("Server %d finished with %v\n", id, exitCode)
}

func ExecClient(clientId, serverId int64) {
	client := exec.Command("../client/client", strconv.FormatInt(clientId, 10), strconv.FormatInt(serverId, 10))
	clientProcess[clientId] = client
	clientErr := client.Start()
	//fmt.Printf("%s\n", serverOut)
	if clientErr != nil {
		panic(clientErr)
	}
	exitCode := client.Wait()
	fmt.Printf("Client %d finished with %v\n", clientId, exitCode)
}

func joinServer(id int64) {
	const maxCount = 100
	count := 0
	fmt.Println("Join Server")
	// 1. Check if server or client already, then print error and exit
	// 2. Else continue

	_, okServer := servers[id]
	_, okClient := clients[id]
	if okClient || okServer {
		fmt.Printf("%d is already used\n", id)
		return
	}

	go ExecServer(id)
	serverPort := strconv.FormatInt(baseServerPort+id, 10)

	client, err := rpc.Dial("tcp", "localhost:"+serverPort)
	for err != nil && count < maxCount {
		time.Sleep(time.Millisecond * 100)
		count++
		client, err = rpc.Dial("tcp", "localhost:"+serverPort)
	}

	if err != nil {
		fmt.Printf("Connection with Server[%d] failed\n", id)
	} else {
		servers[id] = client
		fmt.Printf("Connection with Server[%d] established!\n", id)
	}

}

func joinClient(clientId, serverID int64) {
	const maxCount = 100
	count := 0
	fmt.Println("Join Client")

	_, okServer := servers[clientId]
	_, okClient := clients[clientId]
	if okClient || okServer {
		fmt.Printf("%d is already used\n", clientId)
		return
	}

	go ExecClient(clientId, serverID)
	clientPort := strconv.FormatInt(baseClientPort+clientId, 10)

	client, err := rpc.Dial("tcp", "localhost:"+clientPort)
	for err != nil && count < maxCount {
		time.Sleep(time.Millisecond * 100)
		count++
		client, err = rpc.Dial("tcp", "localhost:"+clientPort)
	}
	if err != nil {
		fmt.Printf("Connection with Client[%d] failed\n", clientId)
	} else {
		clients[clientId] = client
		fmt.Printf("Connection with Client[%d] established!\n", clientId)
	}

}

func killServer(id int64) error {
	if client, ok := servers[id]; ok {
		if _, exist := serverProcess[id]; exist {
			fmt.Printf("Server[%d] exists\n", id)
		} else {
			fmt.Printf("Server[%d] does not exist\n", id)
		}
		// Ask the target server to clean up
		var temp = 0
		err := client.Call("ServerService.Cleanup", &temp, &temp)
		if err != nil {
			fmt.Println(err.Error())
		}
		// Close connection to target server
		client.Close()
		//  Remove the entry from registry
		delete(servers, id)
		// Murder
		serverProcess[id].Process.Kill()
	} else {
		errorString := fmt.Sprintf("Server[%d] does not exist", id)
		return errors.New(errorString)
	}

	return nil
}

func breakConnection(id1 int64, id2 int64) error {
	var reply1, reply2 int64
	_, id1Client := clients[id1]
	_, id1Server := servers[id1]
	_, id2Client := clients[id2]
	_, id2Server := servers[id2]

	if id1Client {
		if id2Client {
			return errors.New("can't break connection between 2 clients")
		} else if id2Server {
			clients[id1].Call("ClientService.BreakConnection", &id2, &reply1)
		} else {
			return errors.New("id2 out of range")
		}
	} else if id1Server {
		if id2Client {
			clients[id2].Call("ClientService.BreakConnection", &id1, &reply2)
		} else if id2Server {
			servers[id1].Call("ServerService.BreakConnection", &id2, &reply1)
			servers[id2].Call("ServerService.BreakConnection", &id1, &reply2)
		} else {
			return errors.New("id2 out of range")
		}
	} else {
		return errors.New("id1 out of range")
	}
	if reply1 == 1 {
		fmt.Printf("Connection from %d to %d was already broken", id1, id2)
	}
	if reply2 == 1 {
		fmt.Printf("Connection from %d to %d was already broken", id2, id1)
	}
	return nil
}

func createConnection(id1 int64, id2 int64) error {
	fmt.Printf("Creating connection between [%d]-[%d]\n", id1, id2)
	var reply1, reply2 int64

	_, id1Client := clients[id1]
	_, id1Server := servers[id1]
	_, id2Client := clients[id2]
	_, id2Server := servers[id2]

	if id1Client {
		if id2Client {
			return errors.New("can't create connection between 2 clients")
		} else if id2Server {
			clients[id1].Call("ClientService.CreateConnection", &id2, &reply1)
		} else {
			return errors.New("id2 out of range")
		}
	} else if id1Server {
		if id2Client {
			clients[id2].Call("ClientService.CreateConnection", &id1, &reply2)
		} else if id2Server {
			servers[id1].Call("ServerService.CreateConnection", &id2, &reply1)
			servers[id2].Call("ServerService.CreateConnection", &id1, &reply2)
		} else {
			return errors.New("id2 out of range")
		}
	} else {
		return errors.New("id1 out of range")
	}
	if reply1 == 1 {
		fmt.Printf("Connection from %d to %d was already established\n", id1, id2)
	}
	if reply2 == 1 {
		fmt.Printf("Connection from %d to %d was already established\n", id2, id1)
	}
	return nil
}

func printStore(id int64) {
	fmt.Printf("Printing store of Server[%d]\n", id)
	var server *rpc.Client
	var ok bool
	if server, ok = servers[id]; !ok {
		fmt.Printf("Server[%d] does not exist\n", id)
		return
	}

	var store map[string]string
	var dummy int64

	err := server.Call("ServerService.PrintStore", &dummy, &store)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for k, v := range store {
		fmt.Printf("%s:%s\n", k, v)
	}

}

func put(clientId int64, key, value string) {
	fmt.Printf("Client[%d] putting %s:%s\n", clientId, key, value)
	client, ok := clients[clientId]

	if !ok {
		fmt.Printf("Client[%d] does not exist\n", clientId)
		return
	}

	var arg PutData
	arg.Key = key
	arg.Value = value
	var reply int64
	err := client.Call("ClientService.Put", &arg, &reply)

	if err != nil {
		fmt.Printf("Error putting\t%v\n", err)
	} else {
		fmt.Printf("Successfully put %s:%s\n", key, value)
	}

}

func get(clientId int64, key string) {
	fmt.Printf("Getting key %s from Client[%d]\n", key, clientId)

	client, ok := clients[clientId]
	if !ok {
		fmt.Printf("Client[%d] does not exist\n", clientId)
	}

	var reply string
	err := client.Call("ClientService.Get", &key, &reply)

	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("Client[%d]\t%s:%s\n", clientId, key, reply)
	}
}

func stabilize() {
	fmt.Printf("Stablizing ...\n")

	server := getRandomServer()

	if server == nil {
		fmt.Printf("No connected servers to stabilize\n")
		return
	}

	var arg, reply int64
	err := server.Call("ServerService.InitStabilize", &arg, &reply)

	if err != nil {
		fmt.Println("Server RPC for Stabilize failed")
		fmt.Println(err.Error())
	} else {
		fmt.Println("Succeeded stabilizing")
	}

	// Invalidate clients' caches
	InvalidateClientCache()

}

/* *******************Helper Functions******************/
func getRandomServer() *rpc.Client {
	length := len(servers)

	if length == 0 {
		return nil
	}

	// Get a random position in the server set
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	serverPos := r.Int63n(int64(length))
	var server *rpc.Client
	var i int64
	// A bit complex to get a random server
	for _, server = range servers {
		if i == serverPos {
			break
		} else {
			i++
		}
	}
	fmt.Printf("Chosen server is %d", serverPos)
	return server
}

func InvalidateClientCache() {
	var count uint64
	for _, client := range clients {
		go func(client *rpc.Client) {
			var arg, reply int64
			client.Call("ClientService.InvalidateCache", &arg, &reply)
		}(client)
		atomic.AddUint64(&count, 1)
	}
	for count < uint64(len(clients)) {

	}
}

func main() {
	joinServer(0)
	joinServer(1)
	joinServer(2)
	joinServer(3)
	joinServer(4)
	// defer killServer(0)
	// defer killServer(1)
	// defer killServer(2)
	// defer killServer(3)
	// defer killServer(4)

	joinClient(5, 0)
	joinClient(6, 1)
	joinClient(7, 2)
	joinClient(8, 3)
	joinClient(9, 4)

	const NUMKEYS int = 20
	const NUMVALS int = 52

	cId := []int{5, 6, 7, 8, 9}
	keys := make([]string, NUMKEYS)
	values := make([]string, NUMVALS)

	// Initialize the test keys and values
	for i := 0; i < NUMKEYS/2; i++ {
		keys[i] = string('0' + i)
		keys[i+10] = "1" + keys[i]
	}

	for i := 0; i < NUMVALS; i++ {
		if i < 26 {
			values[i] = string('a' + i)
		} else {
			values[i] = string('A' + i - 26)
		}
	}

	// Each client in parallel puts random key:value pair
	// var wg sync.WaitGroup
	// wg.Add(5)

	for _, id := range cId {
		//go func(id int64) {
		// defer wg.Done()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < 5; i++ {
			keyPos := r.Intn(NUMKEYS)
			valPos := r.Intn(NUMVALS)
			put(int64(id), keys[keyPos], values[valPos])
		}
		//}(int64(i))
	}

	// Synchronize all "put" threads
	// wg.Wait()
	printStore(0)
	printStore(1)
	printStore(2)
	printStore(3)
	printStore(4)

	stabilize()

	printStore(0)
	printStore(1)
	printStore(2)
	printStore(3)
	printStore(4)

	for {

	}

}
