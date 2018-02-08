package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"syscall"
)

var masterPort int64 = 3000

/*
// Args arithmetic argument
type Args struct {
	A, B int
}

// Quotient division argument
type Quotient struct {
	Quo, Rem int
}

// Arith an int type
type Arith int

// Multiply RPC call to do multiplication
func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

// Divide RPC call to do division
func (t *Arith) Divide(args *Args, quo *Quotient) error {
	if args.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quo = args.A / args.B
	quo.Rem = args.A % args.B
	return nil
}
*/

func main() {
	fmt.Printf("Server process %d started\n", 5)
	id, _ := strconv.ParseInt(os.Args[1], 10, 64)
	fmt.Printf("Server process %d started\n", id)

	masterPort := strconv.FormatInt(masterPort, 10)
	masterConn, masterErr := net.Listen("tcp", ":"+masterPort)
	if masterErr != nil {
		panic(masterErr)
	}
	rpc.Accept(masterConn)
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
	syscall.Exit(100)
}
