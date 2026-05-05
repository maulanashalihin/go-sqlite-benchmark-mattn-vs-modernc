package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
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
)

func main() {
	driver := flag.String("driver", "mattn", "sqlite driver: mattn or modernc")
	port := flag.String("port", "3000", "server port")
	dbPath := flag.String("db", "benchmark.db", "database file path")
	populate := flag.Int("populate", 10000, "number of test users to insert")
	flag.Parse()

	var dsn string
	if *driver == "mattn" {
		dsn = fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", *dbPath)
	} else {
		dsn = fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)", *dbPath)
	}

	driverName := "sqlite3"
	if *driver == "modernc" {
		driverName = "sqlite"
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("create table: %v", err)
	}

	// Check if table already has data
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		log.Fatalf("count users: %v", err)
	}

	if count < *populate {
		log.Printf("Populating %d users...", *populate)
		for i := 0; i < *populate; i++ {
			if _, err := db.Exec(insertSQL, fmt.Sprintf("user%d", i), fmt.Sprintf("user%d@test.com", i), i%100); err != nil {
				log.Fatalf("insert: %v", err)
			}
		}
		log.Println("Done populating.")
	} else {
		log.Printf("Table already has %d users, skipping populate.", count)
	}

	// Prepared statement for best performance
	stmt, err := db.Prepare("SELECT id, name, email, age FROM users WHERE id = ?")
	if err != nil {
		log.Fatalf("prepare: %v", err)
	}
	defer stmt.Close()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Prefork:               false,
	})

	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")
		var user struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   int    `json:"age"`
		}
		if err := stmt.QueryRow(id).Scan(&user.ID, &user.Name, &user.Email, &user.Age); err != nil {
			if err == sql.ErrNoRows {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(user)
	})

	log.Printf("Server starting on :%s with driver=%s db=%s", *port, *driver, *dbPath)
	log.Fatal(app.Listen(":" + *port))
}
