package testsyntax

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type countingRunner struct {
	mu       sync.Mutex
	calls    int
	identity string
}

func (r *countingRunner) Identity() string { return r.identity }
func (r *countingRunner) Run(_ context.Context, inputs []Input) ([]Observation, error) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	rows := make([]Observation, 0, len(inputs))
	for _, in := range inputs {
		rows = append(rows, Observation{Schema: "vitest-lint/v1", File: in.File, Digest: in.Digest, Status: "observed", Checks: []Check{{Rule: "focused-test", Status: "checked_clean"}}})
	}
	return rows, nil
}
func TestObservationReuseIsContentScopedAndCallerIsolated(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.test.ts"), []byte("source one"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "b.test.ts"), []byte("source two"), 0600))
	runner := &countingRunner{identity: "tool-v1"}
	service := Service{Runner: runner}
	first, err := service.Observe(context.Background(), root, []string{"a.test.ts", "b.test.ts"})
	require.NoError(t, err)
	first[0].Checks[0].Status = "caller mutation"
	second, err := service.Observe(context.Background(), root, []string{"b.test.ts", "a.test.ts"})
	require.NoError(t, err)
	require.Equal(t, 1, runner.calls)
	require.Equal(t, "checked_clean", second[0].Checks[0].Status)
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.test.ts"), []byte("new source"), 0600))
	_, err = service.Observe(context.Background(), root, []string{"a.test.ts", "b.test.ts"})
	require.NoError(t, err)
	require.Equal(t, 2, runner.calls)
	runner.identity = "tool-v2"
	_, err = service.Observe(context.Background(), root, []string{"a.test.ts", "b.test.ts"})
	require.NoError(t, err)
	require.Equal(t, 3, runner.calls)
}
func TestConcurrentObservationReuseRunsOwnerOnce(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.test.ts"), []byte("source"), 0600))
	runner := &countingRunner{identity: "v1"}
	service := Service{Runner: runner}
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.Observe(context.Background(), root, []string{"a.test.ts"})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, 1, runner.calls)
}
func TestObservationRejectsMissingEscapingDuplicateAndUnsupportedSources(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.test.ts"), []byte("outside"), 0600))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.test.ts"), filepath.Join(root, "link.test.ts")))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.test.ts"), []byte("source"), 0600))
	runner := &countingRunner{}
	service := Service{Runner: runner}
	for _, files := range [][]string{{"missing.test.ts"}, {"../escape.test.ts"}, {"link.test.ts"}, {"a.test.ts", "a.test.ts"}, {"production.ts"}} {
		_, err := service.Observe(context.Background(), root, files)
		require.Error(t, err)
	}
	require.Zero(t, runner.calls)
}
func TestNativeOwnerRunnerFixture(t *testing.T) {
	script := os.Getenv("QUALITY_HEALTH_SYNTAX_SCRIPT")
	if script == "" {
		t.Skip("provide installed native profile script for conformance")
	}
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "a.test.ts"), []byte(`import {test,expect} from "vitest"; test("bare",()=>{expect(2)})`), 0600))
	service := Service{Runner: NodeRunner{Script: script}}
	rows, err := service.Observe(context.Background(), root, []string{"a.test.ts"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "1.6.9", rows[0].PluginVersion)
	require.Len(t, rows[0].Diagnostics, 1)
	require.Equal(t, "malformed-expectation", rows[0].Diagnostics[0].CanonicalRule)
	require.NotEmpty(t, rows[0].Digest)
}
