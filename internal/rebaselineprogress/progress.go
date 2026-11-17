package rebaselineprogress

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/vrooli/vrooli/internal/config"
)

// Progress records effects already completed; it is never operator
// authority. Incomplete or unreadable evidence cannot establish completion.
type Progress struct {
	Version    int    `json:"version"`
	Candidate  string `json:"candidate_sha256"`
	Watchdog   string `json:"watchdog_sha256"`
	Bundle     string `json:"bundle_sha256,omitempty"`
	SourceRoot string `json:"source_root,omitempty"`
	Mode       string `json:"mode,omitempty"`
	Complete   bool   `json:"complete"`
}

func Read(path string) (*Progress, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var p Progress
	if len(data) > 4096 || json.Unmarshal(data, &p) != nil || p.Version != 1 || len(p.Candidate) != 64 || len(p.Watchdog) != 64 || (p.Mode != "" && p.Mode != "prepared" && p.Mode != "hub") {
		return nil, fmt.Errorf("invalid rebaseline progress at %s; inspect recovery state before continuing", path)
	}
	return &p, nil
}

func Write(path string, p Progress) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(path, data, 0600)
}
