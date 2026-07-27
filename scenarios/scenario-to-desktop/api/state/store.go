package state

import (
	"context"
	"errors"
	"path/filepath"
	"scenario-to-desktop-api/shared/store"
	"time"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// ErrInvalidKey is returned when a scenario name is empty.
var ErrInvalidKey = errors.New("invalid key: scenario name is required")

// Store manages scenario state persistence.
type Store struct {
	fileStore *store.JSONFileStore[string, ScenarioState]
	stateDir  string
	roots     *filerouting.RoutedRoots
}

// NewRoutedStore resolves its state root from each request context. The
// test-mode middleware marks BAS requests, and RoutedRoots then selects the
// lease-owned state directory instead of the live one.
func NewRoutedStore(roots *filerouting.RoutedRoots) (*Store, error) {
	if roots == nil {
		return nil, errors.New("routed state roots are required")
	}
	return &Store{roots: roots}, nil
}

func (s *Store) scoped(ctx context.Context) (*Store, error) {
	if s.roots == nil {
		return s, nil
	}
	dir, err := s.roots.Pick(ctx, storage.ClassState)
	if err != nil {
		return nil, err
	}
	return NewStore(dir)
}

// NewStore creates a new state store at the given state directory.
// State files are stored as {stateDir}/{scenario-name}.json.
func NewStore(stateDir string) (*Store, error) {
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
		stateDir:  stateDir,
	}, nil
}

// Get retrieves scenario state by name.
// Returns nil, nil if not found.
func (s *Store) Get(ctx context.Context, scenarioName string) (*ScenarioState, error) {
	scoped, err := s.scoped(ctx)
	if err != nil {
		return nil, err
	}
	state, err := scoped.fileStore.Get(ctx, scenarioName)
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
	scoped, err := s.scoped(ctx)
	if err != nil {
		return err
	}
	if err := scoped.fileStore.Save(ctx, state.ScenarioName, *state); err != nil {
		return err
	}
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return nil
}

// Delete removes scenario state.
func (s *Store) Delete(ctx context.Context, scenarioName string) error {
	scoped, err := s.scoped(ctx)
	if err != nil {
		return err
	}
	if err := scoped.fileStore.Delete(ctx, scenarioName); err != nil {
		return err
	}
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return nil
}

// Exists checks if scenario state exists.
func (s *Store) Exists(ctx context.Context, scenarioName string) (bool, error) {
	scoped, err := s.scoped(ctx)
	if err != nil {
		return false, err
	}
	return scoped.fileStore.Exists(ctx, scenarioName)
}

// List returns all stored scenario states.
func (s *Store) List(ctx context.Context) ([]ScenarioState, error) {
	scoped, err := s.scoped(ctx)
	if err != nil {
		return nil, err
	}
	return scoped.fileStore.List(ctx)
}

// ListScenarios returns all scenario names with stored state.
func (s *Store) ListScenarios(ctx context.Context) ([]string, error) {
	scoped, err := s.scoped(ctx)
	if err != nil {
		return nil, err
	}
	return scoped.fileStore.ListKeys(ctx)
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
	return filepath.Join(s.stateDir, scenarioName+".json")
}

// GetDataDir returns the state storage directory.
func (s *Store) GetDataDir() string {
	return s.stateDir
}

// Close flushes any pending changes.
func (s *Store) Close() error {
	if s.roots != nil {
		return nil
	}
	return s.fileStore.Close()
}
