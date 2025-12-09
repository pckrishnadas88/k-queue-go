package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
)

const BrokerAddr = "localhost:6379"

// MessageHeaderSize must match the broker's definition (1 byte Command + 4 bytes Size)
const MessageHeaderSize = 5

// Command Type Constants (Must match broker's internal definition)
const CMD_PUB byte = 0x01
const CMD_SUB byte = 0x02

// Message struct must match the broker's struct definition
type Message struct {
	ID      string
	Payload string
}

// PubPayload struct represents the binary content sent for the PUBLISH command.
type PubPayload struct {
	Topic   string
	Message Message // Embeds the message struct with ID and Payload
}

// SubPayload struct represents the binary content sent for the SUBSCRIBE command.
type SubPayload struct {
	Topic string
}

// encodeMessage creates the binary payload with the 5-byte header.
// This function simulates the client's binary protocol encoder.
func encodeMessage(topic string, payload string) []byte {

	// 1. Prepare the payload struct (what gets JSON serialized)
	pubPayload := PubPayload{
		Topic: topic,
		Message: Message{
			ID:      uuid.New().String(),
			Payload: payload,
		},
	}

	// 2. Serialize the payload struct into JSON bytes
	payloadBytes, err := json.Marshal(pubPayload)
	if err != nil {
		log.Fatalf("Client: Failed to marshal message: %v", err)
	}

	payloadSize := uint32(len(payloadBytes))

	// 3. Create the final buffer (Header + Payload)
	buffer := make([]byte, MessageHeaderSize+payloadSize)

	// 4. ✅ FIX: Write the 1-byte Command Type to the start of the buffer (Index 0)
	buffer[0] = CMD_PUB

	// 5. ✅ FIX: Write the 4-byte Payload Size starting from the second byte (Index 1)
	binary.BigEndian.PutUint32(buffer[1:MessageHeaderSize], payloadSize)

	// 6. Copy the JSON payload into the buffer
	copy(buffer[MessageHeaderSize:], payloadBytes)

	return buffer
}

// readResponse reads the broker's simple text response (e.g., OK Published)
func readResponse(conn net.Conn) {
	// A simple buffer read is sufficient for reading the short text ACK response.
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Client: Error reading response: %v", err)
		return
	}
	fmt.Printf("Broker Response: %s\n", string(buffer[:n]))
}

// encodeSubCommand creates the binary payload for a SUBSCRIBE command.
func encodeSubCommand(topic string) []byte {

	// 1. Prepare the payload struct (just the topic name)
	subPayload := SubPayload{Topic: topic}

	// 2. Serialize the payload struct into JSON bytes
	payloadBytes, err := json.Marshal(subPayload)
	if err != nil {
		log.Fatalf("Client: Failed to marshal SUB message: %v", err)
	}

	payloadSize := uint32(len(payloadBytes))

	// 3. Create the final buffer (Header + Payload)
	buffer := make([]byte, MessageHeaderSize+payloadSize)

	// 4. Write the 1-byte Command Type (CMD_SUB) to the start of the buffer
	buffer[0] = CMD_SUB

	// 5. Write the 4-byte Payload Size starting from the second byte (Index 1)
	binary.BigEndian.PutUint32(buffer[1:MessageHeaderSize], payloadSize)

	// 6. Copy the JSON payload into the buffer
	copy(buffer[MessageHeaderSize:], payloadBytes)

	return buffer
}

func main() {
	// 1. Connect to the Broker (unchanged)
	conn, err := net.Dial("tcp", BrokerAddr)
	// ...
	defer conn.Close()
	log.Printf("Client: Connected to %s", BrokerAddr)

	testTopic := "test_binary_topic"

	// 2. NEW: Send the Binary SUB command
	subMessage := encodeSubCommand(testTopic)
	log.Printf("Client: Sending SUB command (%d bytes)...", len(subMessage))

	_, err = conn.Write(subMessage)
	if err != nil {
		log.Fatalf("Client: Failed to write SUB message: %v", err)
	}
	readResponse(conn) // Wait for Broker Response: OK Subscribed to test_binary_topic

	// 3. Encode and Send the Binary PUB Message (Existing code)
	testPayload := "This is the final binary test message."
	binaryMessage := encodeMessage(testTopic, testPayload)

	// 4. Send the Raw Binary Data
	log.Printf("Client: Sending PUB command (%d bytes) for topic '%s'...", len(binaryMessage), testTopic)

	_, err = conn.Write(binaryMessage)
	if err != nil {
		log.Fatalf("Client: Failed to write message: %v", err)
	}

	// 5. Read the Broker's text acknowledgment
	readResponse(conn)
}
