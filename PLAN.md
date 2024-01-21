# Mem DB

## Project Overview
Multithreaded in-memory database engine in Go supporting concurrent inserts, point lookups, and range scans using a B+ tree index. Lightweight query execution layer with goroutines and sync.RWMutex for read-write synchronization. Benchmarked with pprof for throughput and latency under concurrent workloads.

## Tech Stack
- **Language:** Go 1.22+
- **Data Structure:** B+ tree (custom implementation)
- **Concurrency:** sync.RWMutex, goroutines
- **API:** gRPC
- **Profiling:** pprof
- **Testing:** Go benchmarks

## Architecture Overview
```
Client (gRPC)
     |
     v
[Query Executor]
     |
     v
[B+ Tree Index]  <-- sync.RWMutex -->  [Concurrent Readers/Writers]
     |
[In-Memory Pages]
```
