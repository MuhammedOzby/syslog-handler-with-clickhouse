# Syslog Handler with ClickHouse – README

## 📊 Logic Flow (Mermaid)

```mermaid
flowchart TD
  "UDP Listener" --> "Read Message"
  "Read Message" --> "Parse Log (ParseLog)"
  "Parse Log (ParseLog)" --> "Buffered Channel (messageCache)"
  "Buffered Channel (messageCache)" --> "CacheFlush Goroutine"
  "CacheFlush Goroutine" --> "Buffer Size Check"
  "Buffer Size Check" -->|Buffer Full| "Flush to ClickHouse (flushLogs)"
  "Buffer Size Check" -->|Timeout| "Flush to ClickHouse (flushLogs)"
  "Flush to ClickHouse (flushLogs)" --> "Insert into ClickHouse"
  "Insert into ClickHouse" --> "Database (ClickHouse)"
```

## 📁 About

This project is a lightweight **syslog collector** written in Go.  
It listens for UDP syslog messages, parses each message into a structured format, and stores the logs in a **ClickHouse** database in batches.  
The goal is to handle high‑volume syslog traffic efficiently while keeping database write load low.

## 🔎 Entrance

- **Why this code?**  
  Network operations centers (NOCs) and security teams need a fast, reliable way to collect and analyze syslog data.  
  Existing solutions may be heavyweight or lack batching, leading to dropped messages or slow inserts.

- **Where to use?**  
  Deploy it in:
  - NOCs for real‑time monitoring
  - SIEMs for log aggregation
  - Anywhere you need a central, scalable syslog repository

## 📚 Explain

### 1️⃣ Main components

| File | Responsibility |
|------|----------------|
| `main.go` | Application entry point: parses flags, sets up ClickHouse connection, starts UDP listener, launches cache goroutine. |
| `lib/listener.go` | Simple struct to hold listener address and port. |
| `lib/lopParse.go` | Parses RFC 5424 syslog messages into a `LogData` struct. Handles severity extraction and categorization. |
| `lib/cacheManage.go` | Manages an in‑memory buffer (`cacheBuffer`) that accumulates logs. When buffer size or timeout triggers, it calls `flushLogs` to write a batch to ClickHouse. |
| `go.mod` / `go.sum` | Dependencies, including the ClickHouse driver and environment loader. |

### 2️⃣ Data flow

1. **UDP Listener** (`net.ListenUDP`) waits on the configured port (default 514).  
2. When a packet arrives, it is read into a byte slice and converted to a string.  
3. `ParseLog` is invoked with the raw message and the sender’s UDP address.  
4. The resulting `LogData` struct is sent through a **buffered channel** (`messageCache`).  
5. A dedicated goroutine runs `CacheFlush`, which:
   - Collects logs into an in‑memory slice.
   - Checks if either **buffer size** (`CacheSize`) or **timeout** (`CacheTimeout`) has been reached.
   - Calls `flushLogs`, preparing a ClickHouse batch and sending it with a 10‑second context timeout.  
6. Successful inserts log the number of rows written; errors are reported to `log`.

### 3️⃣ Key constants

| Constant | Meaning |
|----------|---------|
| `CacheSize` | Number of logs to buffer before a write (1000). |
| `CacheTimeout` | Maximum time to wait before flushing buffered logs (2 s). |
| `BufferLimit` | Channel capacity to avoid bursts (10 000). |

### 4️⃣ Error handling

- Connection issues to ClickHouse terminate the program.
- UDP read errors set the program state to `1` and exit the loop.
- Batch preparation or send errors are logged but do not crash the goroutine.

### 5️⃣ Environment variables

| Variable | Purpose |
|----------|---------|
| `DB_HOST` | ClickHouse server address |
| `DB_NAME` | Target database |
| `DB_USER` | Username |
| `DB_PASS` | Password |

These are loaded by `godotenv` at startup.

## 🚀 Usage Examples

```bash
# 1️⃣ Build and run (default port 514)
go run main.go

# 2️⃣ Run on a custom port
go run main.go -port 11514

# 3️⃣ Using a .env file
echo "DB_HOST=localhost:9000" >> .env
echo "DB_NAME=logs" >> .env
echo "DB_USER=default" >> .env
echo "DB_PASS=" >> .env
go run main.go
```

**Send a test syslog message**

```bash
echo "<166>1 2023-10-27T10:00:00+00:00 MyDevice this is a test" | nc -u -w1 127.0.0.1 514
```

You should see console output like:

```
>> 1 adet log ClickHouse'a yazıldı.
```

## 📌 Conclusion

- **What does it do?**  
  It collects syslog UDP packets, parses them, buffers them, and writes them in bulk to ClickHouse.  
- **Why is it useful?**  
  Provides a high‑performance, low‑overhead way to ingest logs for analysis or alerting.  
- **When to deploy?**  
  In environments where log volume is high and you need quick, query‑friendly storage.

> That explanation created from AI.

## AI Context & Memory

This Go project implements a UDP‑based syslog collector that parses RFC 5424 messages into a `LogData` struct and buffers them before bulk inserting into ClickHouse. Core logic:

- `main.go` parses command‑line flags, loads environment, opens ClickHouse connection, starts a UDP listener, and spawns a goroutine for `CacheFlush`.
- `lib/lopParse.go` splits raw syslog string, extracts severity (0‑7 enum), device IP, categories, and message body.
- `lib/cacheManage.go` accumulates logs into an in‑memory slice, flushes on buffer size or timeout, and uses a prepared batch for ClickHouse insertion with a 10 s context.
- Buffered channel (`messageCache`) prevents backpressure; `BufferLimit` caps burst capacity.
- Environment variables (`DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASS`) configure ClickHouse connectivity via `clickhouse-go/v2`.
- The system logs insert counts or errors, enabling monitoring of data flow.

Suitable for AI agents to understand message parsing, caching, and database write patterns in a high‑throughput logging pipeline.
