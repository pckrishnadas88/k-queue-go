package kqueueclient

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
)

// --- Constants (Must match Broker definitions) ---
const MessageHeaderSize = 5
const CMD_PUB byte = 0x01
const CMD_SUB byte = 0x02
const CMD_ACK byte = 0x03

// Client holds the persistent connection to the broker.
type Client struct {
	conn net.Conn
}

// Message struct used for internal passing (same as broker)
type Message struct {
	ID      string
	Payload string
}

// SubPayload is the binary payload for CMD_SUB
type SubPayload struct {
	Topic string
}

// PubPayload is the binary payload for CMD_PUB
type PubPayload struct {
	Topic   string
	Message Message
}

// NewClient returns a new Client instance.
func NewClient() *Client {
	return &Client{}
}

// Connect establishes the TCP connection and sets it on the Client struct.
func (c *Client) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	c.conn = conn
	return nil
}

// Close gracefully closes the connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// sendCommand sends the encoded binary command (header + payload) to the broker.
func (c *Client) sendCommand(commandType byte, payload interface{}) (string, error) {
	// 1. Serialize the payload struct (e.g., SubPayload) into JSON bytes
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	payloadSize := uint32(len(payloadBytes))

	// 2. Create the final buffer (5-byte Header + Payload)
	buffer := make([]byte, MessageHeaderSize+payloadSize)

	// 3. Write the 1-byte Command Type
	buffer[0] = commandType

	// 4. Write the 4-byte Payload Size
	binary.BigEndian.PutUint32(buffer[1:MessageHeaderSize], payloadSize)

	// 5. Copy the JSON payload
	copy(buffer[MessageHeaderSize:], payloadBytes)

	// 6. Write the raw binary data to the connection
	if _, err := c.conn.Write(buffer); err != nil {
		return "", fmt.Errorf("failed to write command: %w", err)
	}

	// 7. Read the broker's text response (e.g., "OK Subscribed...")
	return c.readBrokerResponse()
}

// readBrokerResponse reads the broker's simple text response (e.g., OK Published ID:...)
func (c *Client) readBrokerResponse() (string, error) {
	// Use a deadline to prevent hanging forever
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{}) // Clear deadline after read

	buffer := make([]byte, 1024)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}
	return string(buffer[:n]), nil
}

// Public SDK methods

// Subscribe sends the CMD_SUB command to the broker.
func (c *Client) Subscribe(topic string) (string, error) {
	payload := SubPayload{Topic: topic}
	return c.sendCommand(CMD_SUB, payload)
}

// Publish sends the CMD_PUB command to the broker.
func (c *Client) Publish(topic string, message string) (string, error) {
	payload := PubPayload{
		Topic: topic,
		Message: Message{
			ID:      uuid.New().String(),
			Payload: message,
		},
	}
	return c.sendCommand(CMD_PUB, payload)
}
