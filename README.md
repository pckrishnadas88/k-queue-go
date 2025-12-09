# 🚀 KQueue-Go: A High-Performance Message Broker

KQueue-Go is an educational, low-latency message broker built entirely in Go. It is designed to master Go's concurrency model (Goroutines and Channels) and the fundamentals of distributed systems, providing a practical alternative to complex external brokers for internal microservices communication.

## 🌟 Key Features

  * **Concurrency First:** Leverages **Goroutines** to handle every client connection (producer or consumer) and **Channels** to manage the message queue buffer, ensuring minimal overhead and high throughput.
  * **Raw TCP Protocol:** Implements a custom, text-based TCP protocol (`PUB`, `SUB`) for low-latency communication, bypassing HTTP overhead.
  * **Built-in Backpressure:** Uses buffered channels to automatically apply backpressure. If the message queue is full, the publisher is immediately rejected, preventing broker memory overload.
  * **Scalable Architecture:** Designed with clear separation of concerns, making it ready for integration into a multi-node cluster (see Roadmap).

-----

## 🛠️ Getting Started

### Prerequisites

  * Go (1.20 or newer)
  * A terminal for client testing (e.g., `nc` or `telnet`).

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

### 2\. Basic Usage (Terminal Test)

While the server is running, open two separate terminal windows.

**Terminal 1 (Consumer):**

```bash
nc localhost 6379
SUB orders
# Output: OK Subscribed to orders
```

**Terminal 2 (Producer):**

```bash
nc localhost 6379
PUB orders new order received at 12:00
# Output: OK Published
```

The message will instantly appear in **Terminal 1** (`MSG orders new order received at 12:00`).

-----

Yes, you absolutely **should include a prominent warning** in your README stating that **KQueue-Go is not ready for production use**.

This is crucial for two main reasons: **Honesty** (managing expectations) and **Professionalism** (protecting your reputation).

Here is the recommended addition, along with an explanation of where to place it.

---


### ⚠️ WARNING: Educational and Experimental Status

**KQueue-Go is an educational project and is currently NOT suitable for production use.**

This broker is a work-in-progress designed to explore and master high-performance Go concurrency, distributed systems fundamentals (Partitioning, Replication), and custom protocol design.

* **Known Limitations:** The current version lacks essential features required for durability and reliability, including:
    * No Leader Election or Replication (Single Point of Failure).
    * No Consumer Heartbeats or Message Redelivery on failure.
    * No guaranteed strong consistency.
* **Use Case:** This repository is intended for learning, benchmarking, and demonstrating advanced Go systems engineering skills.

---

## 🗺️ Future Roadmap: Distributed Systems Mastery

This roadmap outlines the journey from a single-process broker to a production-ready, fault-tolerant distributed system.

### Phase I: Reliability and Performance (Months 1-2)

| Status | Milestone | Description |
| :--- | :--- | :--- |
| **✅** | TCP & Protocol | Stabilize PUB, SUB, PING; clean parser; subscriber cleanup. |
| **✅** | Core Queue | Topic channels, fanout, backpressure (via channel buffer size). |
| **⬜** | **ACK / Message IDs** | Assign unique IDs to messages, implement client ACK command, and remove messages after confirmation. |
| **⬜** | **Zero-Copy Serialization** | Implement binary serialization (e.g., using Go's `encoding/binary`) to handle message payload as raw bytes, eliminating string conversion overhead. |
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