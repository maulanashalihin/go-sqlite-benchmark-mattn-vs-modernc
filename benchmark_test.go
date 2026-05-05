package benchmark

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

const (
	createTableSQL = `CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		age INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`
	insertSQL = `INSERT INTO users (name, email, age) VALUES (?, ?, ?)`
	selectSQL = `SELECT id, name, email, age FROM users WHERE id = ?`
	updateSQL = `UPDATE users SET age = ? WHERE id = ?`
	deleteSQL = `DELETE FROM users WHERE id = ?`
)

func setupDB(b *testing.B, driverName, dataSourceName string) *sql.DB {
	b.Helper()
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		b.Fatalf("create table: %v", err)
	}
	return db
}

func insertTestData(b *testing.B, db *sql.DB, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatalf("insert test data: %v", err)
		}
	}
}

func BenchmarkInsert_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsert_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBulkInsert_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 100; j++ {
			if _, err := tx.Exec(insertSQL, fmt.Sprintf("user%d", j), fmt.Sprintf("user%d@test.com", j), j%100); err != nil {
				b.Fatal(err)
			}
		}
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBulkInsert_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		for j := 0; j < 100; j++ {
			if _, err := tx.Exec(insertSQL, fmt.Sprintf("user%d", j), fmt.Sprintf("user%d@test.com", j), j%100); err != nil {
				b.Fatal(err)
			}
		}
		if err := tx.Commit(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelect_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id, age int
		var name, email string
		if err := db.QueryRow(selectSQL, (i%1000)+1).Scan(&id, &name, &email, &age); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelect_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id, age int
		var name, email string
		if err := db.QueryRow(selectSQL, (i%1000)+1).Scan(&id, &name, &email, &age); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSelectAll_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.Query(`SELECT id, name, email, age FROM users`)
		if err != nil {
			b.Fatal(err)
		}
		for rows.Next() {
			var id, age int
			var name, email string
			if err := rows.Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
		}
		rows.Close()
	}
}

func BenchmarkSelectAll_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.Query(`SELECT id, name, email, age FROM users`)
		if err != nil {
			b.Fatal(err)
		}
		for rows.Next() {
			var id, age int
			var name, email string
			if err := rows.Scan(&id, &name, &email, &age); err != nil {
				b.Fatal(err)
			}
		}
		rows.Close()
	}
}

func BenchmarkUpdate_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(updateSQL, (i%100)+1, (i%1000)+1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdate_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()
	insertTestData(b, db, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(updateSQL, (i%100)+1, (i%1000)+1); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelete_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, "test", "test@test.com", 25); err != nil {
			b.Fatal(err)
		}
		var id int
		if err := db.QueryRow(`SELECT last_insert_rowid()`).Scan(&id); err != nil {
			b.Fatal(err)
		}
		if _, err := db.Exec(deleteSQL, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelete_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, "test", "test@test.com", 25); err != nil {
			b.Fatal(err)
		}
		var id int
		if err := db.QueryRow(`SELECT last_insert_rowid()`).Scan(&id); err != nil {
			b.Fatal(err)
		}
		if _, err := db.Exec(deleteSQL, id); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPreparedInsert_Mattn(b *testing.B) {
	db := setupDB(b, "sqlite3", ":memory:")
	defer db.Close()

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stmt.Exec(fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPreparedInsert_Modernc(b *testing.B) {
	db := setupDB(b, "sqlite", ":memory:")
	defer db.Close()

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		b.Fatal(err)
	}
	defer stmt.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := stmt.Exec(fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertFile_Mattn(b *testing.B) {
	os.Remove("/tmp/benchmark_mattn.db")
	db := setupDB(b, "sqlite3", "/tmp/benchmark_mattn.db?_journal_mode=WAL&_synchronous=NORMAL")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_mattn.db")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertFile_Modernc(b *testing.B) {
	os.Remove("/tmp/benchmark_modernc.db")
	db := setupDB(b, "sqlite", "file:/tmp/benchmark_modernc.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_modernc.db")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatal(err)
		}
	}
}
