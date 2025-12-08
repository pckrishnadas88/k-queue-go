package broker

import (
	"fmt"
	"log"
	"net"
	"sync"
)

const ChannelBufferSize = 100 // Buffer size controls backpressure

// TopicQueue manages the state and subscribers for a single topic.
type TopicQueue struct {
	msgChan chan string       // Channel where publishers send messages (the queue buffer)
	subs    map[net.Conn]bool // Map to track active subscribers
	mu      sync.Mutex        // Protects the subs map
}

// NewTopicQueue initializes the buffered channel.
func NewTopicQueue() *TopicQueue {
	return &TopicQueue{
		// Buffered channel: This is the queue. When full (100), publish blocks.
		msgChan: make(chan string, ChannelBufferSize),
		subs:    make(map[net.Conn]bool),
	}
}

// AddSubscriber adds a connection and starts a new goroutine to listen for messages.
func (q *TopicQueue) AddSubscriber(conn net.Conn, topic string) {
	// Add to the map (must be mutex-protected)
	q.mu.Lock()
	q.subs[conn] = true
	q.mu.Unlock()

	log.Printf("New subscriber added to topic: %s", topic)

	// CRITICAL: Launch the listener goroutine
	go q.listenAndDispatch(conn, topic)
}

// listenAndDispatch blocks on the message channel and sends messages to the client.
func (q *TopicQueue) listenAndDispatch(conn net.Conn, topic string) {
	defer q.removeSubscriber(conn, topic)

	// This goroutine waits for messages from the q.msgChan (the queue)
	for msg := range q.msgChan {
		_, err := conn.Write([]byte(fmt.Sprintf("MSG %s %s\n", topic, msg)))
		if err != nil {
			// If writing fails (e.g., client closed connection), break the loop
			log.Printf("Failed to dispatch to subscriber on %s: %v", topic, err)
			return
		}
	}
}

// removeSubscriber cleans up the connection when it fails.
func (q *TopicQueue) removeSubscriber(conn net.Conn, topic string) {
	q.mu.Lock()
	delete(q.subs, conn)
	q.mu.Unlock()
	log.Printf("Subscriber removed from topic: %s", topic)
}
