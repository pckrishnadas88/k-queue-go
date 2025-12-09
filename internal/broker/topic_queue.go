package broker

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"net"
	"sync"
)

const ChannelBufferSize = 100 // Buffer size controls backpressure

// TopicQueue manages the state and subscribers for a single topic.
type TopicQueue struct {
	msgChan chan Message      // Channel where publishers send messages (the queue buffer)
	subs    map[net.Conn]bool // Map to track active subscribers
	mu      sync.Mutex        // Protects the subs map
	unacked map[string]Message
}

// Message holds the payload and its unique ID for tracking.
type Message struct {
	ID      string
	Payload string
	// Status  string // Future: for tracking PENDING, ACKED, FAILED
}

// MessageHeader is the fixed-size structure sent before the payload.
// Total size is 5 bytes.
type MessageHeader struct {
	Command     CommandType // 1 byte: What operation to perform
	PayloadSize uint32      // 4 bytes: Length of the JSON payload that follows
}

// CommandType defines the type of operation requested by the client.
type CommandType byte

const (
	// CMD_PUB is the command code for publishing a message.
	CMD_PUB CommandType = 0x01
	// CMD_SUB is the command code for subscribing to a topic.
	CMD_SUB CommandType = 0x02
	// CMD_ACK is the command code for acknowledging a message ID.
	CMD_ACK CommandType = 0x03
)

// MessageHeaderSize will increase to 5 bytes: 4 bytes for size, 1 byte for command type.
const MessageHeaderSize = 5

// NewTopicQueue initializes the buffered channel.
func NewTopicQueue() *TopicQueue {
	return &TopicQueue{
		// Buffered channel: This is the queue. When full (100), publish blocks.
		msgChan: make(chan Message, ChannelBufferSize),
		subs:    make(map[net.Conn]bool),
		unacked: make(map[string]Message),
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

// listenAndDispatch blocks on the message channel and sends messages to the client using the binary protocol.
func (q *TopicQueue) listenAndDispatch(conn net.Conn, topic string) {
	// The defer call ensures cleanup runs regardless of how the goroutine exits.
	defer q.removeSubscriber(conn, topic)

	// This goroutine waits for messages from the q.msgChan (the queue)
	for msg := range q.msgChan {

		// 1. Get the final binary wire format.
		// This includes the 4-byte header and the JSON-encoded payload.
		wireData := msg.Bytes()

		// 2. Mark as Unacknowledged (CRITICAL: Tracking the delivery contract)
		// We lock because the ACK command in the broker handles this map concurrently.
		q.mu.Lock()
		q.unacked[msg.ID] = msg
		q.mu.Unlock()

		log.Printf("[DISPATCH] Sending binary ID:%s (%d bytes total) to %s", msg.ID, len(wireData), conn.RemoteAddr())

		// 3. Write the raw binary data to the connection.
		_, err := conn.Write(wireData)
		if err != nil {
			// Handle Write Failure (Connection Dead)
			log.Printf("Failed to dispatch to subscriber on %s: %v. Client closing.", topic, err)
			return
		}
	}
}

// removeSubscriber cleans up the connection when it fails.
func (q *TopicQueue) removeSubscriber(conn net.Conn, topic string) {
	q.mu.Lock()
	delete(q.subs, conn)
	// NOTE: In a real system, a crashing/disconnecting subscriber means
	// all messages still in the q.unacked map for this connection must be
	// re-queued or sent to a Dead Letter Queue (DLQ).
	// For now, we only clean up the subscriber reference.
	q.mu.Unlock()
	log.Printf("Subscriber removed from topic: %s", topic)
}

// Bytes converts the Message struct into a binary []byte ready for the wire.
func (m *Message) Bytes() []byte {
	// We are simplifying: marshaling the entire struct (ID + Payload) into JSON/Gob first,
	// then prepending the fixed-size header.
	// NOTE: For true zero-copy, you'd use a more specialized library like Protobuf or Cap'n Proto.

	// For this educational example, let's use JSON as the internal serialization format
	// because binary libraries are complex. The key is separating header from payload.

	// Step 1: Serialize the entire Message (ID + Payload)
	payload, err := json.Marshal(m)
	if err != nil {
		log.Fatalf("JSON marshal error: %v", err) // Handle this better in production
	}

	payloadSize := uint32(len(payload))

	// Step 2: Create the final buffer (Header + Payload)
	buffer := make([]byte, MessageHeaderSize+payloadSize)

	// Step 3: Write the fixed-size header (PayloadSize)
	// We use BigEndian (network byte order) which is standard.
	binary.BigEndian.PutUint32(buffer[:MessageHeaderSize], payloadSize)

	// Step 4: Copy the JSON payload into the buffer
	copy(buffer[MessageHeaderSize:], payload)

	return buffer
}
