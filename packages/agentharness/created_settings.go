package agentharness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// The created-paths loosening has one switch and one audit trail, both beside
// the snapshot bundle so the offline hook can read them.

const (
	creationSettingsFile = "created-paths.json"
	creationAuditFile    = "created-deletions.jsonl"
)

// CreationSettings is the loosening switch. It is on unless an operator turned
// it off, so a missing or unreadable file never removes the rule silently the
// other way: an unreadable file reads as on, a deliberate off is explicit.
type CreationSettings struct {
	Enabled bool      `json:"enabled"`
	Changed time.Time `json:"changed,omitempty"`
	By      string    `json:"by,omitempty"`
}

// LoadCreationSettings reads the switch from dir.
func LoadCreationSettings(dir string) CreationSettings {
	settings := CreationSettings{Enabled: true}
	if data, err := os.ReadFile(filepath.Join(dir, creationSettingsFile)); err == nil {
		var stored CreationSettings
		if json.Unmarshal(data, &stored) == nil {
			settings = stored
		}
	}
	return settings
}

// SaveCreationSettings writes the switch to dir.
func SaveCreationSettings(dir string, settings CreationSettings) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, creationSettingsFile+".tmp")
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, creationSettingsFile))
}

// CreationAudit is one automatic allowance: the guard let an agent delete a
// path because the run created it.
type CreationAudit struct {
	At        time.Time `json:"at"`
	Path      string    `json:"path"`
	Recursive bool      `json:"recursive"`
	Source    string    `json:"source"`
	Certainty string    `json:"certainty"`
	Detail    string    `json:"detail,omitempty"`
	Session   string    `json:"session,omitempty"`
}

// AppendCreationAudit records one allowance. Best effort: the decision stands
// even when the trail cannot be written.
func AppendCreationAudit(dir string, entry CreationAudit) {
	if dir == "" || os.MkdirAll(dir, 0o700) != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, creationAuditFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	if line, err := json.Marshal(entry); err == nil {
		_, _ = file.Write(append(line, '\n'))
	}
}

// ReadCreationAudit returns the allowances recorded since the given time.
func ReadCreationAudit(dir string, since time.Time) []CreationAudit {
	data, err := os.ReadFile(filepath.Join(dir, creationAuditFile))
	if err != nil {
		return nil
	}
	var out []CreationAudit
	for _, line := range splitLines(data) {
		var entry CreationAudit
		if json.Unmarshal(line, &entry) == nil && !entry.At.Before(since) {
			out = append(out, entry)
		}
	}
	return out
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			if i > start {
				lines = append(lines, data[start:i])
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
