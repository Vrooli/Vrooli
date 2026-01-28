// Package idgen provides small, shared ID generation utilities.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

var randRead = rand.Read

// Generate creates a short random ID. Falls back to time-based value if entropy fails.
func Generate() string {
	bytes := make([]byte, 8)
	if _, err := randRead(bytes); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(bytes)
}
