package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/hostreq"
	"github.com/vrooli/vrooli/internal/operatorstate"
)

// These aliases keep the API projection small while making the control-plane
// operatorstate package the only production writer and evaluator.
type (
	OperatorState  = operatorstate.Document
	ScenarioChoice = operatorstate.ScenarioChoice
	EnabledChoice  = operatorstate.EnabledChoice
	OptInChoice    = operatorstate.OptInChoice

	operatorCompletion              = operatorstate.Completion
	operatorDegradedAcknowledgement = operatorstate.DegradedAcknowledgement
)

var (
	operatorStateNow   = time.Now
	operatorStateRoots *filerouting.RoutedRoots
	// operatorStatePath remains a test seam for existing API fixtures. Normal
	// runtime path resolution is performed by internal/operatorstate.
	operatorStatePath = func() (string, error) {
		roots, err := resolveRoots()
		if err != nil {
			return "", fmt.Errorf("locate operator state: %w", err)
		}
		return filepath.Join(roots.StorageRoot, operatorstate.StateFile), nil
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

func operatorStateService() *operatorstate.Service {
	roots, _ := resolveRoots()
	root, storageRoot := roots.RepoRoot, roots.StorageRoot
	return operatorstate.New(operatorstate.Config{
		RepoRoot: root, StorageRoot: storageRoot, Roots: operatorStateRoots,
		StatePath: func(context.Context) (string, error) { return operatorStatePath() },
		Now:       operatorStateNow,
	})
}

func defaultOperatorState() OperatorState { return operatorstate.Default() }

func loadOperatorState() (OperatorState, error) {
	return loadOperatorStateFor(context.Background())
}

func loadOperatorStateFor(ctx context.Context) (OperatorState, error) {
	return operatorStateService().Load(ctx)
}

// saveOperatorStateFor exists only as a narrow adapter for older internal
// callers and fixtures. New writers submit a merge patch directly.
func saveOperatorStateFor(ctx context.Context, state OperatorState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = operatorStateService().Apply(ctx, data)
	return err
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
	case http.MethodPatch:
		patch, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read merge patch: " + err.Error()})
			return
		}
		state, err := operatorStateService().ApplyValidated(r.Context(), patch, validateOperatorState)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "validation failed") || strings.Contains(err.Error(), "invalid safeguard") || strings.Contains(err.Error(), "merge patch") {
				status = http.StatusBadRequest
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, state)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func validateOperatorState(state OperatorState) error {
	if err := validateOperatorStateSafeguardConfigs(state); err != nil {
		return err
	}
	if state.Core == nil {
		return nil
	}
	authority := apicoreset.Authority{Seed: state.Core.Seed, TrustedBase: state.Core.TrustedBase}
	if err := authority.Validate(); err != nil {
		return fmt.Errorf("core authority validation failed: %w", err)
	}
	return nil
}

func validateOperatorStateSafeguardConfigs(state OperatorState) error {
	for name, choice := range state.HostSafeguards {
		if len(choice.Config) == 0 {
			continue
		}
		if err := hostreq.ValidateSafeguardConfig(name, choice.Config); err != nil {
			return err
		}
	}
	return nil
}
