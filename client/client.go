package main

import (
	"fmt"
	"log"
	"net/rpc"
)

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

func main() {
	args := &Args{27, 3}
	var reply Quotient
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	err = client.Call("Arith.Divide", args, &reply)
	if err != nil {
		log.Fatal("arith error:", err)
	}
	fmt.Printf("Arith: %d/%d=%d and %d\n", args.A, args.B, reply.Quo, reply.Rem)
}
