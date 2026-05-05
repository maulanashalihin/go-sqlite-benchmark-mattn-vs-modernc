package benchmark

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

// setupHTTPDB creates a DB for HTTP benchmarks with connection pooling
func setupHTTPDB(b *testing.B, driverName, dataSourceName string) *sql.DB {
	b.Helper()
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		b.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	if _, err := db.Exec(createTableSQL); err != nil {
		b.Fatalf("create table: %v", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
		b.Fatalf("set busy_timeout: %v", err)
	}
	return db
}

// insertHTTPTestData populates DB with test users
func insertHTTPTestData(b *testing.B, db *sql.DB, count int) {
	b.Helper()
	for i := 0; i < count; i++ {
		if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
			b.Fatalf("insert test data: %v", err)
		}
	}
}

// buildFiberApp creates a Fiber app with a GET /users/:id endpoint
func buildFiberApp(b *testing.B, db *sql.DB) *fiber.App {
	b.Helper()
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var user struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   int    `json:"age"`
		}
		err := db.QueryRow("SELECT id, name, email, age FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email, &user.Age)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	})

	return app
}

// BenchmarkHTTPGet_Mattn_Memory benchmarks full HTTP GET with in-memory DB
func BenchmarkHTTPGet_Mattn_Memory(b *testing.B) {
	db := setupHTTPDB(b, "sqlite3", "file::memory:?cache=shared&_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	defer db.Close()
	insertHTTPTestData(b, db, 10000)

	app := buildFiberApp(b, db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}

func BenchmarkHTTPGet_Modernc_Memory(b *testing.B) {
	db := setupHTTPDB(b, "sqlite", "file::memory:?cache=shared&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)")
	defer db.Close()
	insertHTTPTestData(b, db, 10000)

	app := buildFiberApp(b, db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}

// BenchmarkHTTPGet_Mattn_File benchmarks full HTTP GET with file-backed DB (most realistic)
func BenchmarkHTTPGet_Mattn_File(b *testing.B) {
	os.Remove("/tmp/benchmark_http_mattn.db")
	db := setupHTTPDB(b, "sqlite3", "/tmp/benchmark_http_mattn.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_http_mattn.db")
	insertHTTPTestData(b, db, 10000)

	app := buildFiberApp(b, db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}

func BenchmarkHTTPGet_Modernc_File(b *testing.B) {
	os.Remove("/tmp/benchmark_http_modernc.db")
	db := setupHTTPDB(b, "sqlite", "file:/tmp/benchmark_http_modernc.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_http_modernc.db")
	insertHTTPTestData(b, db, 10000)

	app := buildFiberApp(b, db)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}

// BenchmarkHTTPGetWithMiddleware_Mattn_File adds a fake auth middleware for realism
func BenchmarkHTTPGetWithMiddleware_Mattn_File(b *testing.B) {
	os.Remove("/tmp/benchmark_http_auth_mattn.db")
	db := setupHTTPDB(b, "sqlite3", "/tmp/benchmark_http_auth_mattn.db?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_http_auth_mattn.db")
	insertHTTPTestData(b, db, 10000)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Fake auth middleware
	app.Use(func(c *fiber.Ctx) error {
		// Simulate JWT verification overhead
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		// Simulate 100µs auth check
		// In real world this would be JWT parse + signature verify
		c.Locals("user_id", "12345")
		return c.Next()
	})

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var user struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   int    `json:"age"`
		}
		err := db.QueryRow("SELECT id, name, email, age FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email, &user.Age)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			req.Header.Set("Authorization", "Bearer fake-jwt-token-12345")
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}

func BenchmarkHTTPGetWithMiddleware_Modernc_File(b *testing.B) {
	os.Remove("/tmp/benchmark_http_auth_modernc.db")
	db := setupHTTPDB(b, "sqlite", "file:/tmp/benchmark_http_auth_modernc.db?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)")
	defer db.Close()
	defer os.Remove("/tmp/benchmark_http_auth_modernc.db")
	insertHTTPTestData(b, db, 10000)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Fake auth middleware
	app.Use(func(c *fiber.Ctx) error {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals("user_id", "12345")
		return c.Next()
	})

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var user struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   int    `json:"age"`
		}
		err := db.QueryRow("SELECT id, name, email, age FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Email, &user.Age)
		if err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/users/%d", (i%10000)+1), nil)
			req.Header.Set("Authorization", "Bearer fake-jwt-token-12345")
			resp, err := app.Test(req, -1)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode != fiber.StatusOK {
				b.Fatalf("unexpected status: %d", resp.StatusCode)
			}
			i++
		}
	})
}
