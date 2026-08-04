package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// OperatorState mirrors .vrooli/schemas/operator-state.schema.json. It owns
// mutable operator decisions only; manifests remain the declarative source.
type OperatorState struct {
	Schema         string                    `json:"$schema,omitempty"`
	Version        string                    `json:"version"`
	UpdatedAt      string                    `json:"updated_at"`
	ActiveProfile  *string                   `json:"active_profile,omitempty"`
	Scenarios      map[string]ScenarioChoice `json:"scenarios,omitempty"`
	Resources      map[string]EnabledChoice  `json:"resources,omitempty"`
	HostTools      map[string]OptInChoice    `json:"host_tools,omitempty"`
	HostSafeguards map[string]OptInChoice    `json:"host_safeguards,omitempty"`
}

type ScenarioChoice struct {
	Enabled     *bool `json:"enabled,omitempty"`
	AutoRestart *bool `json:"auto_restart,omitempty"`
}
type EnabledChoice struct {
	Enabled *bool `json:"enabled,omitempty"`
}
type OptInChoice struct {
	OptedIn *bool `json:"opted_in,omitempty"`
}

var (
	operatorStateNow   = time.Now
	operatorStateRoots *filerouting.RoutedRoots
	operatorStatePath  = func() (string, error) {
		root := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
		if root != "" {
			return filepath.Join(root, ".vrooli", "operator-state.json"), nil
		}
		storageRoot := strings.TrimSpace(os.Getenv("VROOLI_STORAGE_ROOT"))
		if storageRoot == "" {
			return "", fmt.Errorf("VROOLI_ROOT or VROOLI_STORAGE_ROOT is required to locate operator state")
		}
		return filepath.Join(storageRoot, "operator-state.json"), nil
	}
)

func configureOperatorStateRoots() error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	root := filepath.Dir(path)
	operatorStateRoots = filerouting.New(storage.Paths{ConfigDir: root, DataDir: root, CacheDir: root, LogsDir: root, StateDir: root})
	return nil
}

func operatorStatePathFor(ctx context.Context) (string, error) {
	if operatorStateRoots == nil {
		return operatorStatePath()
	}
	root, err := operatorStateRoots.Pick(ctx, storage.ClassConfig)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "operator-state.json"), nil
}

func defaultOperatorState() OperatorState {
	return OperatorState{Schema: ".vrooli/schemas/operator-state.schema.json", Version: "1.0.0", Scenarios: map[string]ScenarioChoice{}, Resources: map[string]EnabledChoice{}, HostTools: map[string]OptInChoice{}, HostSafeguards: map[string]OptInChoice{}}
}

func loadOperatorState() (OperatorState, error) {
	return loadOperatorStateFor(context.Background())
}

func loadOperatorStateFor(ctx context.Context) (OperatorState, error) {
	path, err := operatorStatePathFor(ctx)
	if err != nil {
		return OperatorState{}, err
	}
	state := defaultOperatorState()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return OperatorState{}, fmt.Errorf("read operator state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return OperatorState{}, fmt.Errorf("decode operator state: %w", err)
	}
	if state.Version == "" {
		return OperatorState{}, fmt.Errorf("operator state version is required")
	}
	return state, nil
}

func saveOperatorStateFor(ctx context.Context, state OperatorState) error {
	path, err := operatorStatePathFor(ctx)
	if err != nil {
		return err
	}
	state.Schema = ".vrooli/schemas/operator-state.schema.json"
	state.Version = "1.0.0"
	state.UpdatedAt = operatorStateNow().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := storage.WriteFileAtomic(path, append(data, '\n'), storage.SecretFilePerm); err != nil {
		return err
	}
	if operatorStateRoots != nil {
		operatorStateRoots.RecordWrite(ctx)
	}
	return nil
}

func (s *Server) handleOperatorState(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		state, err := loadOperatorStateFor(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, state)
	case http.MethodPut:
		var state OperatorState
		if !decodeJSONBody(w, r, &state) {
			return
		}
		if err := saveOperatorStateFor(r.Context(), state); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, state)
	}
}
