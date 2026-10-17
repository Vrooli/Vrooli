//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var logger = log.New(os.Stdout, "", log.LstdFlags)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":   "healthy",
		"service":  "algorithm-library-api",
		"database": db != nil && db.Ping() == nil,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
