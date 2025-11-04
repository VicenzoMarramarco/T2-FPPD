package main

import (
	"flag"
	cl "jogo/common/client"
	"log"
)

func main() {
	addr := flag.String("addr", "localhost:12345", "server address (ip:port)")
	name := flag.String("name", "Player", "player name")
	uiAddr := flag.String("ui", "127.0.0.1:4001", "local UI state broadcast address (ip:port)")
	listenAddr := flag.String("listen", "127.0.0.1:4000", "local command listener address for MOVE messages from UI (ip:port)")
	flag.Parse()

	client, err := cl.NewClient(*name, *addr)
	if err != nil {
		log.Fatalf("Failed to connect/register: %v", err)
	}

	client.StartPolling()
	// broadcast state to local UI
	if err := client.StartLocalStateBroadcaster(*uiAddr); err != nil {
		log.Fatalf("Failed to start local state broadcaster: %v", err)
	}
	if err := client.StartLocalCommandListener(*listenAddr); err != nil {
		log.Fatalf("Failed to start local command listener: %v", err)
	}

	select {}
}
