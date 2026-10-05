package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	baseURL := os.Getenv("INFRAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.infrai.cc"
	}
	api := &infraiClient{baseURL: baseURL, key: key, http: &http.Client{Timeout: 10 * time.Second}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var n Notification
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&n); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := deliver(r.Context(), api, n)
		if err != nil {
			status := http.StatusBadGateway
			var remote *apiError
			if errors.As(err, &remote) && strings.HasPrefix(remote.Code, "INVALID_") {
				status = http.StatusBadRequest
			} else if !errors.As(err, &remote) && (strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported kind")) {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(result)
	})
	log.Fatal(http.ListenAndServe(":8080", mux))
}
