package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// UserRequest matches the JSON structure sent by the User Service
type UserRequest struct {
	Username string `json:"username"`
}

func main() {
	// 1. Define the handler
	http.HandleFunc("/evaluate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req UserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Logic for FlashFix roadside evaluation
		log.Printf("Processing evaluation for user: %s", req.Username)
		
		responseMessage := fmt.Sprintf("evaluated for this %s", req.Username)
		
		// Send back the plain text response
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseMessage))
	})

	// 2. Determine port from environment or default to 8081
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// 3. Start the server
	fmt.Printf("Evaluation Service (FlashFix) starting on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %s", err)
	}
}
