package main

import (
	"bytes"
	"fmt"
	"io"
	"log" // Changed to log for timestamps
	"net/http"
	"os"
)

func main() {
	// Use environment variable with a local fallback
	evalURL := os.Getenv("EVALUATION_SERVICE_URL")
	if evalURL == "" {
		evalURL = "http://localhost:8081" // Local fallback
	}

	http.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Printf("Calling Evaluation Service at: %s/evaluate", evalURL)

		// Call Evaluation Service using the variable
		resp, err := http.Post(evalURL+"/evaluate", "application/json", bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, "Evaluation service unreachable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		evalResult, _ := io.ReadAll(resp.Body)
		w.Write(evalResult)
	})

	fmt.Println("User Service (FlashFix) listening on :8080...")
	http.ListenAndServe(":8080", nil)
}
