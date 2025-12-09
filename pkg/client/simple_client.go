package main

import (
	"log"

	"github.com/pckrishnadas88/k-queue-go/pkg/kqueueclient"
)

func main() {
	client := kqueueclient.NewClient()
	if err := client.Connect("localhost:6379"); err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	log.Println("Client: Connected successfully.")

	// 1. Subscribe (Simple Call)
	subResponse, err := client.Subscribe("new_test_topic")
	if err != nil {
		log.Fatalf("Subscription failed: %v", err)
	}
	log.Printf("Broker: %s", subResponse)

	// 2. Publish (Simple Call)
	pubResponse, err := client.Publish("new_test_topic", "Hello from the clean SDK!")
	if err != nil {
		log.Fatalf("Publish failed: %v", err)
	}
	log.Printf("Broker: %s", pubResponse)
}
