package catalogcoverage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"react-component-library/internal/gates"
)

// CountMatrixFailures evaluates the corpus matrix in-process. It is used by
// corpus reporting so measurement does not compile or launch a second copy of
// the gate executable.
func CountMatrixFailures(ctx context.Context, root string) (int, error) {
	definitions, err := LoadGateDefinitions(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		return 0, err
	}
	// Corpus reporting runs the same DB-backed gates as the production matrix.
	// Supplying the routed live database here prevents evidence-dependent gates
	// from being classified as zero-input failures merely because this probe is
	// in-process.
	database, err := openLiveDatabase()
	if err != nil {
		return 0, fmt.Errorf("open live catalog database for matrix measurement: %w", err)
	}
	defer database.Close()
	results := make([]gates.Result, len(definitions))
	measured := make([]bool, len(definitions))
	var firstErr error
	var mu sync.Mutex
	var wait sync.WaitGroup
	workers := make(chan struct{}, 8)
	for index, definition := range definitions {
		index, definition := index, definition
		wait.Add(1)
		go func() {
			defer wait.Done()
			select {
			case workers <- struct{}{}:
				defer func() { <-workers }()
			case <-ctx.Done():
				return
			}
			registered, ok := gates.Lookup(definition.ID)
			if !ok || registered.Run == nil {
				return
			}
			result, runErr := gates.RunDefinition(registered, gates.Scope{Context: ctx, Root: root, DB: database})
			if runErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("run gate %s: %w", definition.ID, runErr)
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			results[index] = gates.NormalizeResult(root, result)
			measured[index] = true
			mu.Unlock()
		}()
	}
	wait.Wait()
	if firstErr != nil {
		return 0, firstErr
	}
	failures := 0
	for index, definition := range definitions {
		if definition.Attribution == "corpus" {
			continue
		}
		if !measured[index] {
			continue
		}
		// RunDefinition normalizes findings with no resolvable asset identity
		// into RunnerError. Those are the only matrix failures that lack an
		// attributable finding; a clean result must not be counted once per
		// inspected asset.
		failures += len(results[index].RunnerError)
	}
	return failures, nil
}
