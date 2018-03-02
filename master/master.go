package main

import (
	"bufio"
	"errors"
	"fmt"
	"math/rand"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	server := exec.Command("./server", strconv.FormatInt(id, 10))

	for serverID, _ := range serverProcess {
		server.Args = append(server.Args, strconv.FormatInt(serverID, 10))
	}

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
	client := exec.Command("./client", strconv.FormatInt(clientId, 10), strconv.FormatInt(serverId, 10))
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
	fmt.Printf("Join Server[%d]\n", id)
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
	fmt.Printf("Join Client[%d]-Server[%d]\n", clientId, serverID)

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

	store := make(map[string]string)
	var dummy int64

	//store["1"] = "a"
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
		return
	}

	var reply string
	err := client.Call("ClientService.Get", &key, &reply)

	if err != nil {
		fmt.Println(err.Error())
		return
	} else {
		fmt.Printf("Client[%d]\t%s:%s\n", clientId, key, reply)
	}
}

func stabilize() {
	fmt.Printf("Stablizing ...\n")

	server := getRandomServer()

	//server := servers[0]
	if server == nil {
		fmt.Printf("No connected servers to stabilize\n")
		return
	}

	serverList := make(map[int64]bool)
	for serverID := range serverProcess {
		serverList[serverID] = true
	}

	for len(serverList) != 0 {
		var arg int64
		reply := make(map[int64]bool)
		err := server.Call("ServerService.InitStabilize", &arg, &reply)

		if err != nil {
			fmt.Println("Server RPC for Stabilize failed")
			fmt.Println(err.Error())
		}

		fmt.Println("List of servers in this MST: ")
		for k := range reply {
			fmt.Printf("%d\t", k)
			delete(serverList, k)
		}
		fmt.Println()

		for k, v := range servers {
			if _, ok := serverList[k]; ok {
				server = v
			}
		}
	}
	fmt.Println("Succeeded stabilizing")

}

/* *******************Helper Functions******************/
// getRandomServer : get an rpc.Client handler of a random existing server
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
	fmt.Printf("Chosen server is %d\n", serverPos)
	return server
}

// InvalidateClientCache : a test function to invalidate clients' caches from the master
// side. Used for testing before version number is introduced. Not being used in current code
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

// Cleanup : send SIGKILL to all of the client/server processes to kill them. This make sure
// that after the master exits, those processes aren't hanging around
func Cleanup() {
	for _, s := range serverProcess {
		s.Process.Kill()
	}

	for _, c := range clientProcess {
		c.Process.Kill()
	}
}

// PrintUsage : print the usage of the master program. We are keeping it minimal here
func PrintUsage() {
	fmt.Println("Invalid command. Please refer to the APIs")
}

// AutomaticTest1 :
func AutomaticTest() {

	// Join servers
	joinServer(0)
	joinServer(1)
	joinServer(2)
	joinServer(3)
	joinServer(4)

	// Partition as simple test
	// breakConnection(0, 2)
	// breakConnection(0, 3)
	// breakConnection(0, 4)
	// breakConnection(1, 2)
	// breakConnection(1, 3)
	// breakConnection(1, 4)

	// Join clients
	joinClient(5, 0)
	joinClient(6, 1)
	joinClient(7, 2)
	joinClient(8, 3)
	joinClient(9, 4)

	createConnection(5, 1)
	createConnection(5, 2)
	createConnection(5, 3)
	createConnection(5, 4)
	createConnection(6, 0)
	createConnection(6, 2)
	createConnection(6, 3)
	createConnection(6, 4)
	createConnection(7, 0)
	createConnection(7, 1)
	createConnection(7, 3)
	createConnection(7, 4)
	createConnection(8, 0)
	createConnection(8, 1)
	createConnection(8, 2)
	createConnection(8, 4)
	createConnection(9, 0)
	createConnection(9, 1)
	createConnection(9, 2)
	createConnection(9, 3)

	// Random puts
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

	// put(5, "1", "a")
	// put(5, "2", "b")
	// put(6, "1", "c")
	// put(6, "3", "d")

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

}

func SimplePartition1() {

	// Create 5 servers
	joinServer(0)
	joinServer(1)
	joinServer(2)
	joinServer(3)
	joinServer(4)

	// Create partition of [0,1] and [2,3,4]
	breakConnection(0, 2)
	breakConnection(0, 3)
	breakConnection(0, 4)
	breakConnection(1, 2)
	breakConnection(1, 3)
	breakConnection(1, 4)

	// Connect clients to each partition (no intersection)
	joinClient(5, 0)
	joinClient(6, 1)
	joinClient(7, 2)
	joinClient(8, 2)
	createConnection(8, 3)
	joinClient(9, 4)

	// Put
	put(5, "1", "a")
	put(6, "1", "c")
	put(7, "1", "b")
	put(8, "2", "d")
	put(9, "1", "c")

	// Gets provide 2 session guarantees but no consistency/total order
	get(5, "1") // a
	get(6, "1") // c
	get(7, "1") // c
	get(8, "1") // b or ERR_KEY
	get(9, "1") // c

	printStore(0)
	printStore(1)
	printStore(2)
	printStore(3)
	printStore(4)

	// Stabilizes 2 partitions separately
	stabilize()

	printStore(0)
	printStore(1)
	printStore(2)
	printStore(3)
	printStore(4)

	// Correct value ordered in each partition
	get(5, "1") // c
	get(6, "1") // c
	get(7, "1") // c
	get(8, "1") // c
	get(9, "1") // c

}

func SimplePartition2() {

	// Create 4 servers
	joinServer(0)
	joinServer(1)
	joinServer(2)
	joinServer(3)

	// Create partition of [0,1] and [2,3]
	breakConnection(0, 2)
	breakConnection(0, 3)
	breakConnection(1, 2)
	breakConnection(1, 3)

	// Connect single client to first partitions
	joinClient(5, 0)
	createConnection(5, 1)

	// Put to each server
	put(5, "1", "a")

	// Get the key
	get(5, "1") // a

	// Connect same client to second partition
	createConnection(5, 2)
	createConnection(5, 3)

	// Break connection to first partition. This guarantees
	// that the new writes only go to second partition
	breakConnection(5, 0)
	breakConnection(5, 1)

	// Put to second partition
	put(5, "1", "b")

	// Get 1 from client
	get(5, "1") // b

	// Stabilize each partition
	stabilize()

	// Break connection to second partition
	breakConnection(5, 2)
	breakConnection(5, 3)

	// Connect back to first partition
	createConnection(5, 0)
	createConnection(5, 1)

	// Now get value for 1. Check if it is serviced
	// from cache (monotonic reads) or from server (wrong answer)
	get(5, "1") // b
}

func TestPartition() {
	joinServer(0)
	joinServer(1)
	breakConnection(0, 1)

	joinClient(2, 0)
	joinClient(3, 1)
	//createConnection(2, 1)
	//createConnection(3, 0)

	put(2, "1", "a")
	//get(3, "1")
	put(3, "2", "b")
	//get(2, "1")

	stabilize()
	createConnection(2, 1)
	//breakConnection(2, 0)
	get(2, "1")
}

func main() {
	//SimplePartition1()
	// SimplePartition2()
	AutomaticTest()
	// TestPartition()

	defer Cleanup()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Enter commands:\n> ")

	for scanner.Scan() {
		line := scanner.Text()
		elements := strings.Split(line, " ")

		// Skip empty line
		if len(elements) == 0 {
			continue
		}

		var id1, id2 int64
		var err error

		switch elements[0] {
		case "joinServer":
			if len(elements) < 2 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			joinServer(id1)

		case "killServer":
			if len(elements) < 2 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			killServer(id1)

		case "joinClient":
			if len(elements) < 3 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			id2, err = strconv.ParseInt(elements[2], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			joinClient(id1, id2)

		case "breakConnection":
			if len(elements) < 3 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			id2, err = strconv.ParseInt(elements[2], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			breakConnection(id1, id2)

		case "createConnection":
			if len(elements) < 3 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			id2, err = strconv.ParseInt(elements[2], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			createConnection(id1, id2)

		case "stabilize":
			stabilize()

		case "printStore":
			if len(elements) < 2 {
				goto InvalidInput
			}
			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			printStore(id1)

		case "put":
			if len(elements) < 4 {
				goto InvalidInput
			}

			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			put(id1, elements[2], elements[3])

		case "get":
			if len(elements) < 3 {
				goto InvalidInput
			}

			id1, err = strconv.ParseInt(elements[1], 10, 64)

			if err != nil {
				fmt.Printf("Can't parse %s to integer\n", elements[1])
				goto InvalidInput
			}

			get(id1, elements[2])

		case "exit":
			return

		default:
			goto InvalidInput
		}
		fmt.Println("################################################")
		fmt.Println()
		fmt.Print("> ")
		continue

	InvalidInput:
		PrintUsage()
		fmt.Printf("> ")

	}

	return

}
