package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// StartAPI starts the API server on the specified port.
// It exposes the latest state of the parsed supply data
func startAPI(port int, s *State) error {
	handleSupplyRequest := func(w http.ResponseWriter, r *http.Request) {
		s.RLock()
		defer s.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
	}

	http.HandleFunc("/", handleSupplyRequest)
	log.Printf("Starting server on :%d\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
