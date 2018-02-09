package main

import (
	"fmt"
	"net/rpc"
	"os/exec"
	"strconv"
	"time"
)

var baseServerPort int64 = 5000
var bestClientPort = 4000
var ready bool = false

func ExecServer(id int64) {
	server := exec.Command("../server/server", strconv.FormatInt(id, 10))
	serverErr := server.Start()
	//fmt.Printf("%s\n", serverOut)
	if serverErr != nil {
		panic(serverErr)
	}
	exitCode := server.Wait()
	fmt.Printf("Server %d finished with %v\n", id, exitCode)
}

// ServerReady : notify when a server is ready
// func (s *ServerRPC) ServerReady() {
// 	ready = true
// }

func joinServer(id int64) {
	const maxCount = 100
	count := 0
	fmt.Println("Join Server")
	go ExecServer(id)
	serverPort := strconv.FormatInt(baseServerPort+id, 10)

	_, err := rpc.Dial("tcp", "localhost:"+serverPort)
	for err != nil && count < maxCount {
		time.Sleep(time.Millisecond * 100)
		count++
		_, err = rpc.Dial("tcp", "localhost:"+serverPort)
	}

	if err != nil {
		fmt.Printf("Connection failed at %d", id)
	} else {
		fmt.Println("Connection established!")
	}

}

func main() {
	joinServer(1)
	joinServer(2)
	for {

	}

}
