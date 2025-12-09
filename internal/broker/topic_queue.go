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

// listenAndDispatch blocks on the message channel and sends messages to the client.
func (q *TopicQueue) listenAndDispatch(conn net.Conn, topic string) {
	defer q.removeSubscriber(conn, topic)

	// This goroutine waits for messages from the q.msgChan (the queue)
	for msg := range q.msgChan {

		// 1. Format output to include the unique ID
		// Protocol: MSG <topic> <message_id> <payload>
		output := fmt.Sprintf("MSG %s %s %s\n", topic, msg.ID, msg.Payload)

		// 2. Mark as Unacknowledged (CRITICAL STEP)
		// This is safe because only ONE dispatch goroutine is writing to the map
		// on behalf of this one connection, but we still use the mutex
		// because the 'ACK' command logic (ExecuteCommand) will access this map concurrently.
		q.mu.Lock()
		q.unacked[msg.ID] = msg
		q.mu.Unlock()

		log.Printf("[DISPATCH] Sending ID:%s to %s", msg.ID, conn.RemoteAddr())

		_, err := conn.Write([]byte(output))
		if err != nil {
			// 3. Handle Write Failure (Connection Dead)
			// If writing fails, the connection is considered dead. We break the loop
			// and the 'defer' call to removeSubscriber cleans up.
			log.Printf("Failed to dispatch to subscriber on %s: %v. Client closing.", topic, err)
			// NOTE: In a real broker, we would need to check if the message was actually sent.
			// If not, it needs to be immediately re-queued or moved to a failed list.
			return
		}
		// NOTE: The message remains in the unacked map until the client sends the ACK command.
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
