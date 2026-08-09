package hostreq

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const operatorStateFileName = ".vrooli/operator-state.json"

type operatorStateEntry struct {
	OptedIn *bool          `json:"opted_in"`
	Config  map[string]any `json:"config"`
}

type operatorStateDocument struct {
	Schema         string                        `json:"$schema"`
	Version        string                        `json:"version"`
	UpdatedAt      string                        `json:"updated_at"`
	HostTools      map[string]operatorStateEntry `json:"host_tools"`
	HostSafeguards map[string]operatorStateEntry `json:"host_safeguards"`
}

// OperatorState is the validated, read-only view of the mutable operator
// choice document. A missing file is a valid empty state; malformed state is
// an error so setup cannot silently fall back to a different policy.
type OperatorState struct {
	hostTools      map[string]operatorStateEntry
	hostSafeguards map[string]operatorStateEntry
	trustPosture   trustposture.State
}

// LoadOperatorState loads the project's durable operator choices. The file is
// intentionally read at the resolver boundary so every caller sees the same
// typed distinction between opted in, declined, and not recorded.
func LoadOperatorState(root string) (OperatorState, error) {
	path := filepath.Join(root, operatorStateFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return OperatorState{trustPosture: trustposture.State{Posture: trustposture.Personal, Source: "default"}}, nil
	}
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator state %s: %w", path, err)
	}

	var document operatorStateDocument
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err := decoder.Decode(&document); err != nil {
		return OperatorState{}, fmt.Errorf("decode operator state %s: %w", path, err)
	}
	if strings.TrimSpace(document.Version) == "" {
		return OperatorState{}, fmt.Errorf("operator state %s: version is required", path)
	}
	if _, err := time.Parse(time.RFC3339, document.UpdatedAt); err != nil {
		return OperatorState{}, fmt.Errorf("operator state %s: updated_at must be RFC3339: %w", path, err)
	}
	posture, err := trustposture.Parse(data, path)
	if err != nil {
		return OperatorState{}, err
	}

	return OperatorState{
		hostTools:      document.HostTools,
		hostSafeguards: document.HostSafeguards,
		trustPosture:   posture,
	}, nil
}

// TrustPosture returns the immutable, typed posture and the source selected by
// the operator-state reader. Callers cannot mutate the durable state through
// this value.
func (s OperatorState) TrustPosture() trustposture.State {
	return s.trustPosture
}

func (s OperatorState) choice(kind hostreqspec.Kind, name string) hostreqspec.OperatorChoice {
	var entries map[string]operatorStateEntry
	if kind == hostreqspec.KindSafeguard {
		entries = s.hostSafeguards
	} else {
		entries = s.hostTools
	}
	entry, ok := entries[strings.TrimSpace(name)]
	if !ok || entry.OptedIn == nil {
		return hostreqspec.OperatorChoiceNotRecorded
	}
	if *entry.OptedIn {
		return hostreqspec.OperatorChoiceOptedIn
	}
	return hostreqspec.OperatorChoiceDeclined
}

func (s OperatorState) config(kind hostreqspec.Kind, name string) map[string]any {
	var entries map[string]operatorStateEntry
	if kind == hostreqspec.KindSafeguard {
		entries = s.hostSafeguards
	} else {
		entries = s.hostTools
	}
	entry, ok := entries[strings.TrimSpace(name)]
	if !ok || len(entry.Config) == 0 {
		return nil
	}
	return cloneConfig(entry.Config)
}

func cloneConfig(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
