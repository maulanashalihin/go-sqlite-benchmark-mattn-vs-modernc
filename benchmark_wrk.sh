#!/bin/bash
set -e

DURATION="30s"
THREADS=10
CONNECTIONS=400

echo "========================================"
echo "SQLite Driver HTTP Benchmark with wrk"
echo "Threads: $THREADS | Connections: $CONNECTIONS | Duration: $DURATION"
echo "========================================"

# Clean up old DBs
rm -f benchmark_mattn.db benchmark_mattn.db-*
rm -f benchmark_modernc.db benchmark_modernc.db-*

# Build server once
echo "Building server..."
go build -o benchmark_server ./cmd/benchmark_server

echo ""
echo "----------------------------------------"
echo "1) mattn/go-sqlite3 on port 3001"
echo "----------------------------------------"
./benchmark_server -driver=mattn -port=3001 -db=benchmark_mattn.db &
SERVER_PID=$!
sleep 2  # Wait for server startup

echo "Warmup..."
wrk -t2 -c20 -d3s -s wrk_script.lua http://localhost:3001 >/dev/null 2>&1 || true

echo "Running wrk..."
wrk -t$THREADS -c$CONNECTIONS -d$DURATION -s wrk_script.lua http://localhost:3001 | tee /tmp/wrk_mattn.txt

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "----------------------------------------"
echo "2) modernc.org/sqlite on port 3002"
echo "----------------------------------------"
./benchmark_server -driver=modernc -port=3002 -db=benchmark_modernc.db &
SERVER_PID=$!
sleep 2

echo "Warmup..."
wrk -t2 -c20 -d3s -s wrk_script.lua http://localhost:3002 >/dev/null 2>&1 || true

echo "Running wrk..."
wrk -t$THREADS -c$CONNECTIONS -d$DURATION -s wrk_script.lua http://localhost:3002 | tee /tmp/wrk_modernc.txt

kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "========================================"
echo "Summary"
echo "========================================"
echo ""
echo "mattn/go-sqlite3:"
grep -E "Requests/sec|Latency|Transfer/sec" /tmp/wrk_mattn.txt || true
echo ""
echo "modernc.org/sqlite:"
grep -E "Requests/sec|Latency|Transfer/sec" /tmp/wrk_modernc.txt || true

# Cleanup
rm -f benchmark_server
