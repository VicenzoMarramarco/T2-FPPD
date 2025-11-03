package main

import (
	"flag"
	sv "jogo/common/server"
	"log"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:12345", "server listen address")
	flag.Parse()

	_, err := sv.StartRPCServer(*addr)
	if err != nil {
		log.Fatalf("failed to start RPC server: %v", err)
	}

	// block forever
	select {}
}
