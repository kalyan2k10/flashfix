package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
)

var db *sql.DB

type UserRequest struct {
	Username string `json:"username"`
}

type EvaluationResult struct {
	Username string `json:"username"`
	Status   string `json:"status"`
}

func main() {
	// 1. Initialize Database
	initDB()

	// 2. Setup Kafka Writer (Producer)
	writer := &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_BROKER")),
		Topic:    "user-registrations",
		Balancer: &kafka.Hash{}, // Use Hash to ensure same Username goes to same Partition
	}

	// 3. Start Background Consumer for Results
	go listenForResults()

	// 4. Endpoint to Create User
	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", 405)
			return
		}

		var req UserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Save to DB as PENDING
		_, err := db.Exec("INSERT INTO users (username, status) VALUES (?, 'PENDING')", req.Username)
		if err != nil {
			http.Error(w, "User already exists or DB error", 500)
			return
		}

		// Send to Kafka
		msg, _ := json.Marshal(req)
		err = writer.WriteMessages(context.Background(), kafka.Message{
			Key:   []byte(req.Username),
			Value: msg,
		})

		if err != nil {
			log.Printf("Kafka Write Error: %v", err)
			w.Write([]byte("User saved to DB, but Kafka notification failed"))
			return
		}

		w.Write([]byte("Registration submitted! Current Status: PENDING"))
	})

	// 5. Endpoint to Check Status (Handy for testing!)
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		username := r.URL.Query().Get("username")
		var status string
		err := db.QueryRow("SELECT status FROM users WHERE username = ?", username).Scan(&status)
		if err != nil {
			http.Error(w, "User not found", 404)
			return
		}
		fmt.Fprintf(w, "User: %s | Status: %s", username, status)
	})

	log.Println("User Service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func initDB() {
	dbUrl := os.Getenv("DB_URL")
	var err error

	// Retry logic because MySQL takes a few seconds to boot
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dbUrl)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Waiting for database... attempt %d", i+1)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to DB:", err)
	}

	// Create table with Status column
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		status VARCHAR(50) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Table creation failed:", err)
	}
	log.Println("Database initialized successfully")
}

func listenForResults() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{os.Getenv("KAFKA_BROKER")},
		Topic:   "evaluation-results",
		GroupID: "user-service-group",
		// Add this line to read all existing messages in the topic
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Reader error: %v", err)
			continue
		}

		var res EvaluationResult
		json.Unmarshal(m.Value, &res)

		_, err = db.Exec("UPDATE users SET status = ? WHERE username = ?", res.Status, res.Username)
		if err != nil {
			log.Printf("Update error: %v", err)
		} else {
			log.Printf("Finalized user %s as %s", res.Username, res.Status)
		}
	}
}
