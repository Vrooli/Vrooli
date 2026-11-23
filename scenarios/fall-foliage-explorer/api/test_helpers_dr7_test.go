package main

import "os"

// Test-only helpers moved from production files by the private-function debt lens.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
