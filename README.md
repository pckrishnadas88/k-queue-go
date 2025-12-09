# 🚀 KQueue-Go: A High-Performance Message Broker

KQueue-Go is an educational, low-latency message broker built entirely in Go. It is designed to master Go's concurrency model (Goroutines and Channels) and the fundamentals of distributed systems, providing a practical alternative to complex external brokers for internal microservices communication.

### ⚠️ WARNING: Educational and Experimental Status

**KQueue-Go is an educational project and is currently NOT suitable for production use.**

This broker is a work-in-progress designed to explore and master high-performance Go concurrency, distributed systems fundamentals (Partitioning, Replication), and custom protocol design.

  * **Known Limitations:** The current version lacks essential features required for durability and reliability, including:
      * No Leader Election or Replication (Single Point of Failure).
      * No Consumer Heartbeats or Message Redelivery on failure.
      * No guaranteed strong consistency.
  * **Use Case:** This repository is intended for learning, benchmarking, and demonstrating advanced Go systems engineering skills.

-----

## 🌟 Key Features

  * **Concurrency First:** Leverages **Goroutines** to handle every client connection and **Channels** to manage the message queue buffer, ensuring minimal overhead and high throughput.
  * **Binary Protocol (Zero-Copy Ready):** Implements a **custom binary TCP protocol** using a fixed-size header for maximum performance, eliminating the overhead of text parsing and delimiters.
  * **Reliability Contract:** Implements **ACK/Message ID tracking** to guarantee message delivery and prevent data loss from unacknowledged messages.
  * **Built-in Backpressure:** Uses buffered channels to automatically apply backpressure. If the message queue is full, the publisher is immediately rejected, preventing broker memory overload.
  * **Scalable Architecture:** Designed with clear separation of concerns, making it ready for integration into a multi-node cluster (see Roadmap).

-----

## 🛠️ Getting Started

### Prerequisites

  * Go (1.20 or newer)
  * The project's dedicated Go test client (`publisher_client.go`).

### 1\. Installation and Run

Clone the repository and run the main executable:

```bash
# Clone the repository
git clone https://github.com/pckrishnadas88/k-queue-go
cd kqueue-go

# Compile and install the server executable
go install ./cmd/kqueue-server

# Run the broker
kqueue-server
```

### 2\. Usage (Binary Protocol Test)

**NOTE:** Because KQueue-Go now uses a binary protocol, you **must use the dedicated Go client** for testing. `nc` and `telnet` are no longer compatible.

Run the client from your project root to perform a full `SUB` -\> `PUB` test:

```bash
go run pkg/client/publisher_client.go
```

**Expected Client Output:**

```
2025/... Client: Sending SUB command (34 bytes)...
Broker Response: OK Subscribed to test_binary_topic

2025/... Client: Sending PUB command (141 bytes) for topic 'test_binary_topic'...
Broker Response: OK Published [UUID] 
```

*A successful test confirms that the binary header and the ACK/ID tracking are working.*

-----

## 🗺️ Future Roadmap: Distributed Systems Mastery

This roadmap outlines the journey from a single-process broker to a production-ready, fault-tolerant distributed system.

### Phase I: Reliability and Performance (Months 1-2)

| Status | Milestone | Description |
| :--- | :--- | :--- |
| **✅** | TCP & Protocol | Stabilize PUB, SUB, PING; clean parser; subscriber cleanup. |
| **✅** | Core Queue | Topic channels, fanout, backpressure (via channel buffer size). |
| **✅** | **ACK / Message IDs** | Assign unique IDs to messages, implement client ACK command, and remove messages after confirmation. |
| **✅** | **Zero-Copy Serialization** | Implemented binary serialization using a 5-byte fixed header to eliminate text parsing overhead. |
| **⬜** | **Persistence V1 (WAL)** | Implement a basic **Write-Ahead Log (WAL)** using Go's `os` package to write committed messages to disk, allowing the broker to survive restarts. |
| **⬜** | **Performance** | Establish baseline benchmarks (throughput, latency) using `go test -bench` and external tools. |

### Phase II: Distributed Cluster (Months 3-4)

This phase requires mastering consensus and cluster membership.

| Status | Milestone | Description |
| :--- | :--- | :--- |
| **⬜** | **Clustering** | Implement cluster member discovery (e.g., using `net/http` heartbeats or a simple list) and inter-broker communication via **gRPC**. |
| **⬜** | **Partitioning** | Implement a **Consistent Hashing** scheme to deterministically map topics to their owning broker instance, distributing the load. |
| **⬜** | **Leader Election** | Integrate an external library (like `hashicorp/raft`) or implement a simple failure detector to elect a topic leader. |
| **⬜** | **Replicated Log** | The topic leader must replicate new messages to follower nodes before committing, ensuring fault tolerance. |

### Phase III: Production Polish (Months 5-6)

| Status | Milestone | Description |
| :--- | :--- | :--- |
| **⬜** | **Reliability** | Implement consumer heartbeats, message redelivery on failure, and a Dead Letter Queue (DLQ) for failed messages. |
| **⬜** | **Monitoring** | Expose internal runtime metrics (Goroutine count, queue latency, memory usage) via a Prometheus-compatible endpoint. |
| **⬜** | **Client SDKs** | Build simple client libraries (e.g., in Python or TypeScript) to prove cross-language compatibility. |

-----

## 📚 Core Architecture

KQueue-Go's architecture is built on the following principles:

1.  **Goroutine per Connection:** Every producer and consumer gets its own dedicated goroutine, ensuring optimal use of CPU cores.
2.  **Channel as Queue:** The core message buffer is a **buffered Go Channel**, which inherently manages concurrency and backpressure.
3.  **Synchronization:** The shared `topics` map is protected by a **`sync.RWMutex`** to prevent race conditions during concurrent access by many goroutines.