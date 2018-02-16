package main

import (
	"errors"
	"fmt"
	"net/rpc"
	"os/exec"
	"strconv"
	"time"
)

var baseServerPort int64 = 5000
var baseClientPort int64 = 5000
var servers = make(map[int64]*rpc.Client)     // map[server id][server rpc handler]
var clients = make(map[int64]*rpc.Client)     // map[server id][client rpc handler]
var serverProcess = make(map[int64]*exec.Cmd) // map[server id][server procees]
var clientProcess = make(map[int64]*exec.Cmd) // map[client id][client process]

type PutData struct {
	key, value string
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
	fmt.Printf("Pringting store of Server[%d]\n", id)
	var server *rpc.Client
	var ok bool
	if server, ok = servers[id]; !ok {
		fmt.Printf("Server[%d] does not exist\n", id)
		return
	}

	var store map[string]string

	err := server.Call("ServerService.PrintStore", nil, &store)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for k, v := range store {
		fmt.Printf("%s:%s\n", k, v)
	}

}

func put(clientId int64, key, value string) {
	fmt.Printf("Client[%d] putting %s:%s", clientId, key, value)
	client, ok := clients[clientId]

	if !ok {
		fmt.Printf("Client[%d] does not exist\n", clientId)
		return
	}

	var arg PutData
	arg.key = key
	arg.value = value
	var reply int64
	err := client.Call("ServerService.Put", &PutData{key, value}, &reply)

	if err != nil {
		fmt.Printf("Error putting\n")
	} else {
		fmt.Printf("Successfully put %s:%s", key, value)
	}

}

func main() {
	joinServer(1)
	joinServer(2)
	joinServer(3)
	//joinServer(4)
	joinServer(5)

	joinClient(4, 5)

	err := breakConnection(11, 5)
	if err != nil {
		fmt.Println(err.Error())
	}

	breakConnection(2, 3)

	err = createConnection(5, 11)
	if err != nil {
		fmt.Println(err.Error())
	}
	createConnection(3, 2)
	createConnection(3, 4)
	for {

	}

}
