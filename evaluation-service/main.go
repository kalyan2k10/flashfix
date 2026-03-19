package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
)

var db *sql.DB

type UserEvent struct {
	Username string `json:"username"`
}

type EvaluationResult struct {
	Username string `json:"username"`
	Status   string `json:"status"`
}

func main() {
	// 1. Initialize local evaluation database
	initDB()

	// 2. Setup Kafka Reader (Consumer)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{os.Getenv("KAFKA_BROKER")},
		Topic:       "user-registrations",
		GroupID:     "eval-service-group",
		StartOffset: kafka.FirstOffset,
	})
	defer reader.Close()

	// 3. Setup Kafka Writer (Producer)
	writer := &kafka.Writer{
		Addr:     kafka.TCP(os.Getenv("KAFKA_BROKER")),
		Topic:    "evaluation-results",
		Balancer: &kafka.Hash{},
	}
	defer writer.Close()

	log.Println("Evaluation Worker Started and listening for events...")

	for {
		// This blocks until a new user registration event arrives
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}
		log.Printf("Received message from Kafka: %s", string(m.Value))

		var event UserEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Failed to unmarshal: %v", err)
			continue
		}

		// 4. Run Business Logic (The "Evaluation")
		// Logic: Reject if name contains "bad", otherwise Approve
		status := "ACTIVE"
		reason := "Passed automated checks"
		if strings.Contains(strings.ToLower(event.Username), "bad") {
			status = "REJECTED"
			reason = "Flagged as restricted username"
		}

		// 5. Save result to Evaluation DB (Source of Truth for this service)
		_, err = db.Exec("INSERT INTO evaluations (username, result, reason) VALUES (?, ?, ?)",
			event.Username, status, reason)
		if err != nil {
			log.Printf("Failed to save evaluation record: %v", err)
			// We continue anyway so we don't block the pipeline
		}

		// 6. Produce result back to Kafka
		res := EvaluationResult{
			Username: event.Username,
			Status:   status,
		}
		resBytes, _ := json.Marshal(res)

		err = writer.WriteMessages(context.Background(), kafka.Message{
			Key:   m.Key, // Keeping the same key (Username) for partition ordering
			Value: resBytes,
		})

		if err != nil {
			log.Printf("Failed to publish result to Kafka: %v", err)
		} else {
			log.Printf("Successfully evaluated %s as %s", event.Username, status)
		}
	}
}

func initDB() {
	dbUrl := os.Getenv("DB_URL")
	var err error

	// Retry loop for MySQL availability
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dbUrl)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Evaluation DB not ready... retrying in 3s (Attempt %d)", i+1)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to Evaluation DB:", err)
	}

	// Create table for storing evaluation history
	query := `
	CREATE TABLE IF NOT EXISTS evaluations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		result VARCHAR(50) NOT NULL,
		reason TEXT,
		evaluated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatal("Evaluation table creation failed:", err)
	}
	log.Println("Evaluation Database initialized")
}
