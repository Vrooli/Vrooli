package dependencygovernance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// firstPartyAuditFile records, one JSON line per applied install, every
// dependency admitted as first-party. Such an install has no registry record,
// so this file is the only trail of what was admitted without one.
const firstPartyAuditFile = ".vrooli/dependencies/first-party-installs.jsonl"

type firstPartyAuditEntry struct {
	Time      string `json:"time"`
	Scenario  string `json:"scenario"`
	Surface   string `json:"surface"`
	Ecosystem string `json:"ecosystem"`
	Package   string `json:"package"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path"`
}

// appendFirstPartyAudit appends one entry. Lines are written in a single
// O_APPEND write, so concurrent applies never interleave.
func appendFirstPartyAudit(repoRoot string, entry firstPartyAuditEntry) error {
	if repoRoot == "" {
		return fmt.Errorf("repository root unknown")
	}
	entry.Time = time.Now().UTC().Format(time.RFC3339)
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(repoRoot, filepath.FromSlash(firstPartyAuditFile))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(line, '\n'))
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	return writeErr
}
