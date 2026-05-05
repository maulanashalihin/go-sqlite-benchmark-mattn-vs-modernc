# SQLite Go Driver Benchmark

Benchmark comparison between [`mattn/go-sqlite3`](https://github.com/mattn/go-sqlite3) and [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) for CRUD operations.

## Environment

- **OS:** macOS (darwin)
- **Arch:** arm64 (Apple M4)
- **Go:** 1.26.1
- **Libraries:**
  - `github.com/mattn/go-sqlite3` v1.14.22
  - `modernc.org/sqlite` v1.34.5

## Results

| Operation | mattn/go-sqlite3 | modernc.org/sqlite | Winner | Diff |
|---|---|---|---|---|
| Insert single | 2,531 ns/op | 3,411 ns/op | **mattn** | ~26% faster |
| Bulk Insert (100 rows/tx) | 186,398 ns/op | 269,902 ns/op | **mattn** | ~31% faster |
| Select by ID | 2,685 ns/op | 3,311 ns/op | **mattn** | ~19% faster |
| Select All (1000 rows) | 495,871 ns/op | **443,785 ns/op** | **modernc** | ~10% faster |
| Update single | 1,329 ns/op | 1,867 ns/op | **mattn** | ~29% faster |
| Delete single | 5,543 ns/op | 6,886 ns/op | **mattn** | ~19% faster |
| Prepared Insert | 1,544 ns/op | 3,395 ns/op | **mattn** | ~54% faster |
| Insert (file + WAL) | 11,753 ns/op | 14,156 ns/op | **mattn** | ~17% faster |

## Conclusion

**mattn/go-sqlite3 is faster for almost all operations**, especially:

- **Write operations** (Insert, Update, Delete): 17–31% faster
- **Prepared statements**: ~2x faster
- **File-based DB**: 17% faster

**modernc.org/sqlite** only wins on:

- **Select All** (scanning many rows at once): ~10% faster

## Trade-off

| mattn/go-sqlite3 | modernc.org/sqlite |
|---|---|
| Faster (CGO) | Slower but no CGO |
| Harder to cross-compile | Easy cross-compile (pure Go) |
| 24K+ importers | 3.5K+ importers |
| Requires gcc/Clang | No C toolchain needed |

## Run the Benchmark

```bash
cd /Volumes/data/Project/go-db-benchmark
GONOSUMDB=* GOFLAGS="-mod=mod" GOPROXY=off go test -bench=. -run=^$ -benchtime=1s
```

Or with a different bench time:

```bash
go test -bench=. -run=^$ -benchtime=3s
```

## Concurrent Benchmark (Real-World Parallel Load)

Single-threaded benchmarks above don't reflect production reality. Below are results from `b.RunParallel` simulating concurrent goroutines (10-core Apple M4, WAL mode, 100 max open connections):

| Scenario | mattn/go-sqlite3 | modernc.org/sqlite | Notes |
|---|---|---|---|
| **Select (in-memory)** | ~123K RPS | ~204K RPS | `cache=shared` required |
| **Write (in-memory)** | ~130K RPS | ~198K RPS | Single-writer lock applies |
| **Mixed 80/20 (file + WAL)** | ~155K RPS | ~130K RPS | File-backed more realistic |
| **Select (file + WAL)** | **~624K RPS** | ~188K RPS | Mattn excels on file reads |
| **Write (file + WAL)** | ~58K RPS | ~47K RPS | Both bottleneck on write lock |

### Key Takeaway from Concurrent Tests

At **10,000 RPS production target**, both drivers handle the load. The raw speed gap shrinks significantly under concurrent load because SQLite's single-writer lock becomes the bottleneck, not the driver.

---

## Production Recommendation: Use `modernc.org/sqlite`

Despite `mattn/go-sqlite3` winning most raw benchmarks, **we recommend `modernc.org/sqlite` for production Go servers** — especially with frameworks like **Go Fiber**.

### Why? HTTP Overhead Dominates

A real-world Fiber endpoint spends time on many things before touching SQLite:

```
Fiber HTTP parse + routing     ~50-100 µs
Auth (JWT verify)              ~100-300 µs
JSON marshal/unmarshal         ~30-80 µs
Business logic                 ~20-50 µs
SQLite query (modernc)         ~5 µs
SQLite query (mattn)           ~1.5 µs
───────────────────────────────────────
Total (modernc)                ~205-535 µs
Total (mattn)                  ~202-532 µs
```

**The driver difference is only ~0.5-1.5% of total request time.** You hit network, JSON, and auth limits long before the SQLite driver.

### Deployment Reality

| | `mattn/go-sqlite3` | `modernc.org/sqlite` |
|---|---|---|
| **Docker build** | Needs `gcc`, `musl-dev`, larger image | Alpine or `FROM scratch` works |
| **Cross-compile** | Requires `CC=` toolchain setup | `GOOS=linux GOARCH=amd64` trivial |
| **Static binary** | Hard (CGO) | `CGO_ENABLED=0` single binary |
| **Debug production** | CGO stack traces are painful | Full Go, easy profiling |
| **CI/CD** | Extra setup | Works everywhere |

### When to Use `mattn/go-sqlite3`

Only choose mattn if:
- Workload is **batch/ETL** where DB is 90%+ of CPU time
- You need **absolute lowest per-query latency** (microservices mesh)
- Cross-compilation toolchain is already set up and maintained

### Bottom Line for 10K RPS + Fiber

Pick **`modernc.org/sqlite`**. Sleep better with pure Go deployments. If you ever outgrow SQLite's single-writer limit at 50K+ RPS, migrate to PostgreSQL — but the driver choice won't be why you hit that wall.

---

## Run the Concurrent Benchmark

```bash
cd /Volumes/data/Project/go-db-benchmark
go test -bench=BenchmarkConcurrent -run=^$ -benchtime=2s
```

## License

MIT
