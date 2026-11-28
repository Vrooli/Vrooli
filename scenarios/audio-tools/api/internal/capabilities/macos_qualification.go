package capabilities

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// MacOSCandidate describes the evidence required before a native speech
// artifact may be declared supported on macOS.
type MacOSCandidate struct {
	Path        string
	Signed      bool
	SHA256      string
	SmokePassed bool
}

// MacOSQualification reports every unmet qualification requirement. It does
// not change a resource declaration; a manifest may claim support only after
// this result is recorded as evidence.
func MacOSQualification(candidate MacOSCandidate) []string {
	var unmet []string
	if candidate.Path == "" {
		unmet = append(unmet, "native server executable path is missing")
	} else if info, err := os.Stat(candidate.Path); err != nil || info.IsDir() {
		unmet = append(unmet, "native server executable is missing")
	}
	if !candidate.Signed {
		unmet = append(unmet, "signed macOS build is missing")
	}
	if candidate.Path != "" {
		if got, err := fileSHA256(candidate.Path); err != nil {
			unmet = append(unmet, "artifact checksum cannot be computed")
		} else if candidate.SHA256 == "" || got != candidate.SHA256 {
			unmet = append(unmet, fmt.Sprintf("pinned SHA-256 checksum is missing or mismatched (got %s)", got))
		}
	} else {
		unmet = append(unmet, "pinned SHA-256 checksum is missing")
	}
	if !candidate.SmokePassed {
		unmet = append(unmet, "macOS smoke run is missing")
	}
	return unmet
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
