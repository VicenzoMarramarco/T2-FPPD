package main

import (
	"flag"
	cl "jogo/common/client"
	"log"
)

func main() {
	addr := flag.String("addr", "localhost:12345", "server address (ip:port)")
	name := flag.String("name", "Player", "player name")
	flag.Parse()

	client, err := cl.NewClient(*name, *addr)
	if err != nil {
		log.Fatalf("Failed to connect/register: %v", err)
	}

	client.StartPolling()
	if err := client.StartLocalCommandListener(); err != nil {
		log.Fatalf("Failed to start local command listener: %v", err)
	}

	select {}
}
