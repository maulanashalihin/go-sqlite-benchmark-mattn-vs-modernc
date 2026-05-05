package benchmark

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

// setupDBConcurrent creates a DB with connection limits suitable for concurrent testing
func setupDBConcurrent(b *testing.B, driverName, dataSourceName string, maxOpen int) *sql.DB {
	b.Helper()
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	if _, err := db.Exec(createTableSQL); err != nil {
		b.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		b.Fatalf("set busy_timeout: %v", err)
	}
	return db
}

// insertTestDataConcurrent inserts data safely for concurrent setup
func insertTestDataConcurrent(b *testing.B, db *sql.DB, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatalf("insert test data: %v", err)
		}
	}
}

// BenchmarkConcurrentSelect_Mattn simulates many goroutines reading simultaneously
func BenchmarkConcurrentSelect_Mattn(b *testing.B) {
	db := setupDBConcurrent(b, "sqlite3", "file::memory:?cache=shared&_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", 100)
	defer db.Close()
	insertTestDataConcurrent(b, db, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkConcurrentSelect_Modernc(b *testing.B) {
	db := setupDBConcurrent(b, "sqlite", "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", 100)
	defer db.Close()
	insertTestDataConcurrent(b, db, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// BenchmarkConcurrentWrite_Mattn tests write contention (single writer lock)
func BenchmarkConcurrentWrite_Mattn(b *testing.B) {
	db := setupDBConcurrent(b, "sqlite3", "file::memory:?cache=shared&_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", 100)
	defer db.Close()

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			c := counter
			counter++
			mu.Unlock()
			if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", c), fmt.Sprintf("user%d@test.com", c), int(c)%100); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConcurrentWrite_Modernc(b *testing.B) {
	db := setupDBConcurrent(b, "sqlite", "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", 100)
	defer db.Close()

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			c := counter
			counter++
			mu.Unlock()
			if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", c), fmt.Sprintf("user%d@test.com", c), int(c)%100); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkConcurrentReadWrite_Mattn simulates mixed workload (80% read, 20% write)
func BenchmarkConcurrentReadWrite_Mattn(b *testing.B) {
	os.Remove("/tmp/benchmark_mattn_rw.db")
	db := setupDBConcurrent(b, "sqlite3", "/tmp/benchmark_mattn_rw.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_mattn_rw.db")
	insertTestDataConcurrent(b, db, 10000)

	var writeCounter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if i%5 == 0 { // 20% writes
				mu.Lock()
				c := writeCounter
				writeCounter++
				mu.Unlock()
				if _, err := db.Exec(insertSQL, fmt.Sprintf("rwuser%d", c), fmt.Sprintf("rw%d@test.com", c), int(c)%100); err != nil {
					b.Fatal(err)
				}
			} else { // 80% reads
				if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
					b.Fatal(err)
				}
			}
			i++
		}
	})
}

func BenchmarkConcurrentReadWrite_Modernc(b *testing.B) {
	os.Remove("/tmp/benchmark_modernc_rw.db")
	db := setupDBConcurrent(b, "sqlite", "file:/tmp/benchmark_modernc_rw.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_modernc_rw.db")
	insertTestDataConcurrent(b, db, 10000)

	var writeCounter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if i%5 == 0 { // 20% writes
				mu.Lock()
				c := writeCounter
				writeCounter++
				mu.Unlock()
				if _, err := db.Exec(insertSQL, fmt.Sprintf("rwuser%d", c), fmt.Sprintf("rw%d@test.com", c), int(c)%100); err != nil {
					b.Fatal(err)
				}
			} else { // 80% reads
				if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
					b.Fatal(err)
				}
			}
			i++
		}
	})
}

// BenchmarkConcurrentSelectFile_Mattn tests concurrent reads from file-backed DB
func BenchmarkConcurrentSelectFile_Mattn(b *testing.B) {
	os.Remove("/tmp/benchmark_mattn_concurrent.db")
	db := setupDBConcurrent(b, "sqlite3", "/tmp/benchmark_mattn_concurrent.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_mattn_concurrent.db")
	insertTestDataConcurrent(b, db, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func BenchmarkConcurrentSelectFile_Modernc(b *testing.B) {
	os.Remove("/tmp/benchmark_modernc_concurrent.db")
	db := setupDBConcurrent(b, "sqlite", "file:/tmp/benchmark_modernc_concurrent.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_modernc_concurrent.db")
	insertTestDataConcurrent(b, db, 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var id, age int
		var name, email string
		i := 0
		for pb.Next() {
			if err := db.QueryRow(selectSQL, (i%10000)+1).Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// BenchmarkConcurrentWriteFile_Mattn tests write contention on file-backed DB
func BenchmarkConcurrentWriteFile_Mattn(b *testing.B) {
	os.Remove("/tmp/benchmark_mattn_write.db")
	db := setupDBConcurrent(b, "sqlite3", "/tmp/benchmark_mattn_write.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_mattn_write.db")

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			c := counter
			counter++
			mu.Unlock()
			if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", c), fmt.Sprintf("user%d@test.com", c), int(c)%100); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkConcurrentWriteFile_Modernc(b *testing.B) {
	os.Remove("/tmp/benchmark_modernc_write.db")
	db := setupDBConcurrent(b, "sqlite", "file:/tmp/benchmark_modernc_write.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", 100)
	defer db.Close()
	defer os.Remove("/tmp/benchmark_modernc_write.db")

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			c := counter
			counter++
			mu.Unlock()
			if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", c), fmt.Sprintf("user%d@test.com", c), int(c)%100); err != nil {
				b.Fatal(err)
			}
		}
	})
}
