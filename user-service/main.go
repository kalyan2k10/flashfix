package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

type UserRequest struct {
	Username string `json:"username"`
}

var db *sql.DB

func initDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		dsn = "root:admin@tcp(user-db:3306)/userdb"
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Invalid DSN:", err)
	}

	// NEW: Force a connection check
	err = db.Ping()
	if err != nil {
		log.Printf("Database not ready yet, retrying... (%v)", err)
		// In a real app, you'd loop here or use a backoff
	}

	query := `CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("Table creation failed: %s", err) // This will now show you WHY it fails
	}
	log.Println("Database initialized successfully")
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req UserRequest
		json.Unmarshal(body, &req)

		// Persist to local DB
		_, err := db.Exec("INSERT INTO users (username) VALUES (?)", req.Username)
		if err != nil {
			log.Printf("DB Insert Error: %v", err)
		}

		// Proxy to Evaluation Service
		evalURL := os.Getenv("EVALUATION_SERVICE_URL")
		resp, err := http.Post(evalURL+"/evaluate", "application/json", bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, "Eval service down", 500)
			return
		}
		defer resp.Body.Close()

		w.WriteHeader(http.StatusOK)
		io.Copy(w, resp.Body)
	})

	log.Println("User Service running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
