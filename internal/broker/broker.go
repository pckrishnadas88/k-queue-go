package broker

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/google/uuid" // Use a robust external library for UUIDs
)

// Broker is the core service struct.
type Broker struct {
	mu sync.RWMutex // Protects the topics map
	// Map of topic names to their associated TopicQueue structs
	topics map[string]*TopicQueue
}

// NewBroker initializes and returns a new Broker instance.
func NewBroker() *Broker {
	return &Broker{
		topics: make(map[string]*TopicQueue),
	}
}

// HandleConnection manages the lifecycle and message parsing for one client connection.
func (b *Broker) HandleConnection(conn net.Conn) {
	defer conn.Close()
	// Get a reader for efficient line-by-line reading
	reader := bufio.NewReader(conn)

	log.Printf("Client connected: %s", conn.RemoteAddr())

	// Loop to process incoming commands
	for {
		// ReadString blocks this goroutine until a newline is found or an error occurs
		command, err := reader.ReadString('\n')
		if err != nil {
			// This handles disconnects (io.EOF) and read errors gracefully
			log.Printf("Client disconnected or read error: %v", err)
			break
		}

		// Execute the command logic
		response := b.ExecuteCommand(conn, command)

		// Write response back to the client
		if _, err := conn.Write([]byte(response + "\n")); err != nil {
			log.Printf("Error writing response: %v", err)
			break
		}
	}
}

// ExecuteCommand parses and executes the logic for PUB and SUB.
func (b *Broker) ExecuteCommand(conn net.Conn, command string) string {
	parts := strings.Fields(strings.TrimSpace(command))
	if len(parts) == 0 {
		return "ERR Missing command"
	}

	cmd := strings.ToUpper(parts[0])

	switch cmd {
	case "PUB":
		if len(parts) < 3 {
			return "ERR Usage: PUB <topic> <message>"
		}
		topic := parts[1]
		message := strings.Join(parts[2:], " ")
		return b.Publish(topic, message)

	case "SUB":
		if len(parts) < 2 {
			return "ERR Usage: SUB <topic>"
		}
		topic := parts[1]
		return b.Subscribe(conn, topic)

	case "ACK":
		if len(parts) < 2 {
			return "ERR Usage: ACK <message_id>"
		}
		messageID := parts[1]
		return b.Acknowledge(messageID)

	default:
		return "ERR Unknown command"
	}
}

// --- Topic Queue Management ---

// Publish sends a message to the specified topic.
func (b *Broker) Publish(topic, message string) string {
	b.mu.RLock()
	q, ok := b.topics[topic]
	b.mu.RUnlock()
	messageID := uuid.New().String()
	msg := Message{
		ID:      messageID,
		Payload: message,
	}

	if !ok {
		// If topic doesn't exist, create it (optional: depends on policy)
		return "ERR Topic not found. Try subscribing first."
	}

	// CRITICAL: Send message to the TopicQueue channel
	select {
	case q.msgChan <- msg:
		// Message was successfully sent to the channel buffer
		return "OK Published " + messageID
	default:
		// Backpressure in action: Channel buffer is full, reject the message
		return "ERR Queue full, backpressure applied"
	}
}

// Subscribe registers a client connection to a topic.
func (b *Broker) Subscribe(conn net.Conn, topic string) string {
	b.mu.Lock()
	q, ok := b.topics[topic]
	if !ok {
		// If topic doesn't exist, create a new TopicQueue (with a buffered channel)
		q = NewTopicQueue()
		b.topics[topic] = q
	}
	b.mu.Unlock()

	// CRITICAL: Launch a new Goroutine to handle the subscription
	// This goroutine blocks waiting for messages ONLY on this topic.
	q.AddSubscriber(conn, topic)
	return fmt.Sprintf("OK Subscribed to %s", topic)
}

// NEW: Acknowledge function to safely remove the message.
func (b *Broker) Acknowledge(messageID string) string {
	// You'll need to figure out which topic the message belongs to,
	// but for simplicity now, we'll iterate or require the topic in the command:
	// ACK <topic> <id>

	// Assuming a simple system where we just check one global or inferred unacked map:
	// (A real broker would check the specific topic's unacked map)

	b.mu.Lock()
	defer b.mu.Unlock()

	// CRITICAL: Find the message in the appropriate TopicQueue and remove it.
	// For now, this logic is pseudocode:
	for _, q := range b.topics {
		q.mu.Lock()
		if _, found := q.unacked[messageID]; found {
			delete(q.unacked, messageID)
			q.mu.Unlock()
			log.Printf("Message ID %s acknowledged and removed.", messageID)
			return "OK ACK"
		}
		q.mu.Unlock()
	}

	return "ERR Message ID not found or already acknowledged."
}
