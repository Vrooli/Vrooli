//go:build linux

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScopeProcessesContextVerifiesCollectedScope(t *testing.T) {
	for _, load := range []string{"not-found", "loaded"} {
		t.Run(load, func(t *testing.T) {
			ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-collected.scope", nil)
			ref.Name = "vrooli-test-collected"
			if err := os.RemoveAll(filepath.Join(cgroupMount, ref.Path)); err != nil {
				t.Fatal(err)
			}
			old := queryScopeStatesBatch
			t.Cleanup(func() { queryScopeStatesBatch = old })
			called := false
			queryScopeStatesBatch = func(_ context.Context, units []string) (map[string]scopeUnitState, error) {
				called = true
				return map[string]scopeUnitState{units[0]: {load: load, active: "inactive"}}, nil
			}
			pids, err := ScopeProcessesContext(context.Background(), ref)
			if err != nil || len(pids) != 0 || !called {
				t.Fatalf("gone scope rejected %v %v queried=%t", pids, err, called)
			}
		})
	}
}

func TestScopeProcessesContextRefusesInconsistentOrUnknownGoneState(t *testing.T) {
	for _, kind := range []string{"membership-only", "active-unit", "query-error", "wrong-name"} {
		t.Run(kind, func(t *testing.T) {
			ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-collected.scope", nil)
			ref.Name = "vrooli-test-collected"
			if kind != "membership-only" {
				_ = os.RemoveAll(filepath.Join(cgroupMount, ref.Path))
			}
			if kind == "wrong-name" {
				ref.Name = "other"
			}
			old := queryScopeStatesBatch
			t.Cleanup(func() { queryScopeStatesBatch = old })
			queryScopeStatesBatch = func(_ context.Context, units []string) (map[string]scopeUnitState, error) {
				if kind == "query-error" {
					return nil, errors.New("bus unavailable")
				}
				if kind == "active-unit" {
					return map[string]scopeUnitState{units[0]: {load: "loaded", active: "active"}}, nil
				}
				return map[string]scopeUnitState{units[0]: {load: "not-found", active: "inactive"}}, nil
			}
			if _, err := ScopeProcessesContext(context.Background(), ref); err == nil {
				t.Fatal("unverified empty state accepted")
			}
		})
	}
}

func TestScopeProcessesBatchContextBatchesExactGoneUnitProofs(t *testing.T) {
	refs := make([]ScopeRef, 3)
	for i := range refs {
		name := "vrooli-test-batch-" + string(rune('a'+i))
		refs[i] = fixtureCgroup(t, "user.slice/vrooli-test.slice/"+name+".scope", nil)
		refs[i].Name = name
		if err := os.RemoveAll(filepath.Join(cgroupMount, refs[i].Path)); err != nil {
			t.Fatal(err)
		}
	}
	old := queryScopeStatesBatch
	t.Cleanup(func() { queryScopeStatesBatch = old })
	calls := 0
	queryScopeStatesBatch = func(_ context.Context, units []string) (map[string]scopeUnitState, error) {
		calls++
		if len(units) != len(refs) {
			t.Fatalf("batched units=%d want %d", len(units), len(refs))
		}
		states := make(map[string]scopeUnitState, len(units))
		for _, unit := range units {
			states[unit] = scopeUnitState{load: "not-found", active: "inactive"}
		}
		return states, nil
	}
	observations := ScopeProcessesBatchContext(context.Background(), refs)
	if calls != 1 || len(observations) != len(refs) {
		t.Fatalf("batch calls=%d observations=%d", calls, len(observations))
	}
	for i, observation := range observations {
		if observation.Err != nil || len(observation.PIDs) != 0 {
			t.Fatalf("gone observation %d=%+v", i, observation)
		}
	}
}

func TestScopeProcessesBatchContextRejectsActiveGoneUnit(t *testing.T) {
	ref := fixtureCgroup(t, "user.slice/vrooli-test.slice/vrooli-test-active-collected.scope", nil)
	ref.Name = "vrooli-test-active-collected"
	if err := os.RemoveAll(filepath.Join(cgroupMount, ref.Path)); err != nil {
		t.Fatal(err)
	}
	old := queryScopeStatesBatch
	t.Cleanup(func() { queryScopeStatesBatch = old })
	for _, active := range []string{"active", "activating"} {
		t.Run(active, func(t *testing.T) {
			queryScopeStatesBatch = func(context.Context, []string) (map[string]scopeUnitState, error) {
				return map[string]scopeUnitState{ref.Name + ".scope": {load: "loaded", active: active}}, nil
			}
			observations := ScopeProcessesBatchContext(context.Background(), []ScopeRef{ref})
			if len(observations) != 1 || observations[0].Err == nil {
				t.Fatalf("%s unit was accepted as gone: %+v", active, observations)
			}
		})
	}
}

func TestScopeProcessesBatchContextUsesOneSnapshotForLargeGoneUnitSet(t *testing.T) {
	count := 12017
	refs := make([]ScopeRef, count)
	for i := range refs {
		unit := fmt.Sprintf("vrooli-agent-batch-%04d.scope", i)
		refs[i] = ScopeRef{
			Name: strings.TrimSuffix(unit, ".scope"), Kind: ScopeKindCgroup,
			Path: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/" + unit,
		}
	}
	old := queryScopeStatesBatch
	t.Cleanup(func() { queryScopeStatesBatch = old })
	var batchSizes []int
	queryScopeStatesBatch = func(_ context.Context, units []string) (map[string]scopeUnitState, error) {
		batchSizes = append(batchSizes, len(units))
		states := make(map[string]scopeUnitState, len(units))
		for _, unit := range units {
			states[unit] = scopeUnitState{load: "not-found", active: "inactive"}
		}
		return states, nil
	}
	observations := ScopeProcessesBatchContext(context.Background(), refs)
	if len(batchSizes) != 1 || batchSizes[0] != count {
		t.Fatalf("unit snapshots=%v want one complete request for %d units", batchSizes, count)
	}
	if len(observations) != count {
		t.Fatalf("observations=%d want %d", len(observations), count)
	}
	for i, observation := range observations {
		if observation.Err != nil || len(observation.PIDs) != 0 {
			t.Fatalf("gone observation %d=%+v", i, observation)
		}
	}
}

func TestScopeProcessesBatchCommandFailureStaysUnknown(t *testing.T) {
	unitA := "vrooli-agent-failed-a.scope"
	unitB := "vrooli-agent-failed-b.scope"
	binDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%s loaded active running demo\\n%s not-found inactive dead demo\\n'\nexit 1\n", unitA, unitB)
	if err := os.WriteFile(filepath.Join(binDir, "systemctl"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := queryScopeStatesBatch
	t.Cleanup(func() { queryScopeStatesBatch = old })
	queryScopeStatesBatch = queryScopeStatesBatchCommand
	refs := []ScopeRef{
		{Name: strings.TrimSuffix(unitA, ".scope"), Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/" + unitA},
		{Name: strings.TrimSuffix(unitB, ".scope"), Kind: ScopeKindCgroup, Path: "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/" + unitB},
	}
	observations := ScopeProcessesBatchContext(context.Background(), refs)
	if len(observations) != len(refs) {
		t.Fatalf("observations=%d want %d", len(observations), len(refs))
	}
	for i, observation := range observations {
		if observation.Err == nil {
			t.Fatalf("command failure produced a gone result at %d: %+v", i, observation)
		}
	}
}

func TestScopeProcessesBatchCommandRejectsFailedSnapshot(t *testing.T) {
	unitA := "vrooli-agent-not-found-a.scope"
	unitB := "vrooli-agent-not-found-b.scope"
	binDir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nprintf '%s not-found inactive dead demo\\n%s not-found inactive dead demo\\n'\nexit 1\n", unitA, unitB)
	if err := os.WriteFile(filepath.Join(binDir, "systemctl"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	states, err := queryScopeStatesBatchCommand(context.Background(), []string{unitA, unitB})
	if err == nil || states != nil {
		t.Fatalf("failed scope snapshot was accepted: states=%v err=%v", states, err)
	}
}

func TestParseScopeUnitListRequiresExactInactiveLoadedOrExactMissing(t *testing.T) {
	units := []string{"vrooli-agent-loaded.scope", "vrooli-agent-active.scope", "vrooli-agent-activating.scope", "vrooli-agent-missing.scope"}
	output := "vrooli-agent-loaded.scope loaded inactive dead old session\n" +
		"vrooli-agent-active.scope loaded active running active session\n" +
		"vrooli-agent-activating.scope loaded activating start starting session\n" +
		"vrooli-agent-loaded-extra.scope loaded inactive dead similar name\n"
	states, err := parseScopeUnitList(output, units)
	if err != nil {
		t.Fatal(err)
	}
	if states[units[0]] != (scopeUnitState{load: "loaded", active: "inactive"}) {
		t.Fatalf("loaded inactive state=%+v", states[units[0]])
	}
	if states[units[1]] != (scopeUnitState{load: "loaded", active: "active"}) {
		t.Fatalf("active state=%+v", states[units[1]])
	}
	if states[units[2]] != (scopeUnitState{load: "loaded", active: "activating"}) {
		t.Fatalf("activating state=%+v", states[units[2]])
	}
	if states[units[3]] != (scopeUnitState{load: "not-found", active: "inactive"}) {
		t.Fatalf("missing exact unit state=%+v", states[units[3]])
	}
}

func TestParseScopeUnitListRejectsMalformedAndDuplicateExactEntries(t *testing.T) {
	unit := "vrooli-agent-exact.scope"
	for _, tc := range []struct {
		name   string
		output string
	}{
		{name: "malformed", output: "systemd failed to list units\n"},
		{name: "duplicate_exact", output: unit + " loaded inactive dead first\n" + unit + " loaded inactive dead second\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseScopeUnitList(tc.output, []string{unit}); err == nil {
				t.Fatal("malformed or duplicate unit snapshot was accepted")
			}
		})
	}
}
