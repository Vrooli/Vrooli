package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// MigrationReport records a legacy import without ever exposing credential
// values. Source deletion is opt-in and occurs only after native persistence
// succeeds.
type MigrationReport struct {
	Imported      []string `json:"imported"`
	SourceDeleted bool     `json:"source_deleted"`
}

// MigrateLegacyFile imports the retired desktop secrets document into this
// manager's native authority. It is deliberately separate from Load: runtime
// startup never reads a legacy file and callers must explicitly consent to its
// deletion after a successful import.
func (sm *Manager) MigrateLegacyFile(path string, deleteSource bool) (MigrationReport, error) {
	if sm == nil || sm.authority == nil {
		return MigrationReport{}, fmt.Errorf("legacy migration requires a native credential authority")
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return MigrationReport{}, fmt.Errorf("legacy credential file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return MigrationReport{}, fmt.Errorf("read legacy credential file: %w", err)
	}
	legacy, err := decodeLegacySecrets(data)
	if err != nil {
		return MigrationReport{}, err
	}
	allowed := map[string]struct{}{}
	for _, definition := range sm.manifest.Secrets {
		allowed[definition.ID] = struct{}{}
	}
	imported := make(map[string]string)
	for key, value := range legacy {
		if _, ok := allowed[key]; ok && strings.TrimSpace(value) != "" {
			imported[key] = value
		}
	}
	existing, err := sm.Load()
	if err != nil {
		return MigrationReport{}, fmt.Errorf("read existing native credentials: %w", err)
	}
	for key, value := range imported {
		existing[key] = value
	}
	if err := sm.Persist(existing); err != nil {
		return MigrationReport{}, fmt.Errorf("persist imported credentials: %w", err)
	}
	report := MigrationReport{Imported: make([]string, 0, len(imported))}
	for key := range imported {
		report.Imported = append(report.Imported, key)
	}
	sort.Strings(report.Imported)
	if deleteSource {
		if err := os.Remove(path); err != nil {
			return MigrationReport{}, fmt.Errorf("delete migrated legacy credential file: %w", err)
		}
		report.SourceDeleted = true
	}
	return report, nil
}

func decodeLegacySecrets(data []byte) (map[string]string, error) {
	var wrapped struct {
		Secrets map[string]string `json:"secrets"`
	}
	if err := json.Unmarshal(data, &wrapped); err == nil && wrapped.Secrets != nil {
		return wrapped.Secrets, nil
	}
	var flat map[string]string
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, fmt.Errorf("decode legacy credential file: %w", err)
	}
	return flat, nil
}
