// Package rootrebaselinedecision defines the exact terms a human reviews before
// a root persistent-state rebaseline. Both producer and Hub hash the same data.
package rootrebaselinedecision

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Terms is serialized in this field order for the v1 canonical digest. Changing
// fields, order, JSON names, or serialization requires a new protocol version.
type Terms struct {
	RequesterRef      string   `json:"requester_ref"`
	Deadline          string   `json:"deadline"`
	CandidateSHA      string   `json:"candidate_sha256"`
	ServingSHA        string   `json:"serving_sha256"`
	WatchdogSHA       string   `json:"watchdog_sha256"`
	CandidateContract string   `json:"candidate_contract_sha256"`
	ServingContract   string   `json:"serving_contract_sha256"`
	Changes           []string `json:"changes"`
}

// ValidateChanges preserves the complete explanation within the Hub's display
// limit. It never silently truncates information the operator must review.
func ValidateChanges(changes []string) error {
	if len(changes) == 0 || len(changes) > 128 {
		return fmt.Errorf("expected 1 to 128 state changes")
	}
	total := 0
	for _, change := range changes {
		if !utf8.ValidString(change) || change == "" || strings.TrimSpace(change) != change {
			return fmt.Errorf("state change is empty or invalid")
		}
		for _, r := range change {
			if unicode.IsControl(r) {
				return fmt.Errorf("state change contains a control character")
			}
		}
		total += utf8.RuneCountInString(change) + 3
	}
	if total > 3500 {
		return fmt.Errorf("state changes exceed the Hub display limit")
	}
	return nil
}

func Digest(terms Terms) (string, error) {
	encoded, err := json.Marshal(terms)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
