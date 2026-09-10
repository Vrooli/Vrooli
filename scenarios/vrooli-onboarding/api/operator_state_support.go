package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	apicoreset "github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/clock"
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
	operatorStateNow   = clock.Real{}.Now
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

func validateOperatorState(state OperatorState) error {
	if err := validateOperatorStateSafeguardConfigs(state); err != nil {
		return err
	}
	if err := validateOperatorStateCapacity(state); err != nil {
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

func validateOperatorStateCapacity(state OperatorState) error {
	roots, err := resolveRoots()
	if err != nil {
		return err
	}
	return operatorstate.ValidateCapacityChoices(roots.RepoRoot, state)
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
