package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"scenario-to-desktop-api/shared/store"
)

// ErrInvalidKey is returned when a scenario name is empty.
var ErrInvalidKey = errors.New("invalid key: scenario name is required")

// Store manages scenario state persistence.
type Store struct {
	fileStore *store.JSONFileStore[string, ScenarioState]
	dataDir   string
}

// NewStore creates a new state store at the given data directory.
// State files are stored as {dataDir}/{scenario-name}.json.
func NewStore(dataDir string) (*Store, error) {
	stateDir := filepath.Join(dataDir, "state")

	fileStore, err := store.NewJSONFileStoreString[ScenarioState](
		stateDir,
		store.PerItem,
		store.WithFileStoreOptions[string, ScenarioState](store.StoreOptions[string, ScenarioState]{
			BeforeSave: func(s ScenarioState) ScenarioState {
				s.UpdatedAt = time.Now()
				if s.CreatedAt.IsZero() {
					s.CreatedAt = s.UpdatedAt
				}
				s.SchemaVersion = SchemaVersion
				// Sanitize secrets before persisting
				if s.FormState.PreflightSecrets != nil {
					sanitized := make(map[string]string, len(s.FormState.PreflightSecrets))
					for k := range s.FormState.PreflightSecrets {
						sanitized[k] = "" // Clear values, keep keys
					}
					s.FormState.PreflightSecrets = sanitized
				}
				return s
			},
		}),
	)
	if err != nil {
		return nil, err
	}

	return &Store{
		fileStore: fileStore,
		dataDir:   stateDir,
	}, nil
}

// Get retrieves scenario state by name.
// Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, scenarioName string) (*ScenarioState, error) {
	state, err := s.fileStore.Get(ctx, scenarioName)
	if err == store.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// Save stores or updates scenario state.
func (s *Store) Save(ctx context.Context, state *ScenarioState) error {
	if state.ScenarioName == "" {
		return ErrInvalidKey
	}
	return s.fileStore.Save(ctx, state.ScenarioName, *state)
}

// Delete removes scenario state.
func (s *Store) Delete(ctx context.Context, scenarioName string) error {
	return s.fileStore.Delete(ctx, scenarioName)
}

// Exists checks if scenario state exists.
func (s *Store) Exists(ctx context.Context, scenarioName string) (bool, error) {
	return s.fileStore.Exists(ctx, scenarioName)
}

// List returns all stored scenario states.
func (s *Store) List(ctx context.Context) ([]ScenarioState, error) {
	return s.fileStore.List(ctx)
}

// ListScenarios returns all scenario names with stored state.
func (s *Store) ListScenarios(ctx context.Context) ([]string, error) {
	return s.fileStore.ListKeys(ctx)
}

// Update atomically updates a scenario state using a modifier function.
func (s *Store) Update(ctx context.Context, scenarioName string, modifier func(*ScenarioState)) error {
	existing, err := s.Get(ctx, scenarioName)
	if err != nil {
		return err
	}

	state := existing
	if state == nil {
		state = &ScenarioState{
			ScenarioName:  scenarioName,
			SchemaVersion: SchemaVersion,
			CreatedAt:     time.Now(),
			Stages:        make(map[string]StageState),
		}
	}

	modifier(state)
	return s.Save(ctx, state)
}

// GetStatePath returns the file path for a scenario's state file.
func (s *Store) GetStatePath(scenarioName string) string {
	return filepath.Join(s.dataDir, scenarioName+".json")
}

// GetDataDir returns the state storage directory.
func (s *Store) GetDataDir() string {
	return s.dataDir
}

// Close flushes any pending changes.
func (s *Store) Close() error {
	return s.fileStore.Close()
}

// DefaultDataDir returns the default state storage location.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".vrooli", "scenario-to-desktop")
}
