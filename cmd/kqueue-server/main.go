package main

import (
	"log"
	"net"

	"github.com/pckrishnadas88/k-queue-go/internal/broker" // Import your internal package
)

const ListenAddr = "localhost:6379"

func main() {
	// Create a new instance of the core broker service
	broker := broker.NewBroker()

	listener, err := net.Listen("tcp", ListenAddr)
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()
	log.Printf("🚀 K-Queue-Go running on %s", ListenAddr)

	// Accept and delegate connections to goroutines
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		// CRITICAL: Handle each client in a new Goroutine for concurrency
		go broker.HandleConnection(conn)
	}
}
