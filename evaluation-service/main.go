package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		dsn = "root:admin@tcp(evaluation-db:3306)/evaluationdb"
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS evaluations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		evaluated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req UserRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Persist to Evaluation DB
		_, err := db.Exec("INSERT INTO evaluations (username) VALUES (?)", req.Username)
		if err != nil {
			log.Printf("DB Insert Error: %v", err)
		}

		w.Write([]byte(fmt.Sprintf("evaluated for this %s", req.Username)))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("Evaluation Service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
