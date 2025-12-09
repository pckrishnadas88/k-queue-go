package broker

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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

// SubPayload is the binary payload for the CMD_SUB command.
type SubPayload struct {
	Topic string
}

// AckPayload is the binary payload for the CMD_ACK command.
type AckPayload struct {
	MessageID string
}

// PubPayload is the binary payload for the CMD_PUB command.
// (You can reuse your existing Message struct if you like,
// or define a separate struct for cleaner command handling)
type PubPayload struct {
	Topic   string
	Payload string // The message content
}

// In internal/broker/broker.go
func (b *Broker) HandleConnection(conn net.Conn) {
	defer conn.Close()

	// Note: If you implement the PING/PONG Monitor, add it here:
	// go b.MonitorConnection(conn, monitor)

	for {
		commandType, payload, err := readBinaryCommand(conn)

		if err != nil {
			// ... handle error and break ...
			break
		}

		// Response variable must be initialized
		var response string

		// CRITICAL: Switch on the command type
		switch commandType {
		case CMD_PUB:
			var pubPayload PubPayload
			if err := json.Unmarshal(payload, &pubPayload); err != nil {
				response = fmt.Sprintf("ERR Decoding PUB payload: %v", err)
			} else {
				response = b.Publish(pubPayload.Topic, pubPayload.Payload)
			}

		case CMD_SUB:
			var subPayload SubPayload
			if err := json.Unmarshal(payload, &subPayload); err != nil {
				response = fmt.Sprintf("ERR Decoding SUB payload: %v", err)
			} else {
				response = b.Subscribe(conn, subPayload.Topic)
			}

		case CMD_ACK:
			var ackPayload AckPayload
			if err := json.Unmarshal(payload, &ackPayload); err != nil {
				response = fmt.Sprintf("ERR Decoding ACK payload: %v", err)
			} else {
				// NOTE: The ACK command needs to be updated to take the topic as well
				// For simplicity, let's assume ACK takes only the ID and we search globally
				response = b.Acknowledge(ackPayload.MessageID)
			}

		default:
			response = fmt.Sprintf("ERR Unknown Command Type: %x", commandType)
		}

		// Write the text response back to the client
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
		return "OK Published " + msg.ID
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

// readBinaryCommand reads the full 5-byte header and the subsequent payload.
func readBinaryCommand(conn net.Conn) (CommandType, []byte, error) {
	// 1. Read the fixed-size header (5 bytes: 1 byte Command + 4 bytes Size)
	headerBuf := make([]byte, MessageHeaderSize)

	_, err := io.ReadFull(conn, headerBuf) // Use io.ReadFull for reliability
	if err != nil {
		return 0, nil, err
	}

	// 2. Decode the Command Type (first byte)
	commandType := CommandType(headerBuf[0])

	// 3. Decode the Payload Size (next 4 bytes)
	payloadSize := binary.BigEndian.Uint32(headerBuf[1:MessageHeaderSize])

	// 4. Read the payload of exactly that size
	payloadBuf := make([]byte, payloadSize)
	_, err = io.ReadFull(conn, payloadBuf)
	if err != nil {
		return 0, nil, err
	}

	return commandType, payloadBuf, nil // Return the type and the raw JSON payload
}
