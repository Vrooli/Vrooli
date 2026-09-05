package control

import (
	"context"
	"testing"

	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
)

func TestValidatedFlowPersistsAndRepairPreservesChecks(t *testing.T) { // [REQ:DVC-AGENT-REUSE]
	s, db := testService(t)
	ctx := context.Background()
	flow := Flow{ID: "candidate", Name: "TV setting", Steps: []Step{{ID: "verify", Kind: "semantic-assert", Target: "Settings"}}}
	s.flowRuns["run1"] = flow
	s.runs["run1"] = RunResult{RunID: "run1", Disposition: "passed", Chapters: []Chapter{{ID: "verify", Disposition: "passed"}}}
	s.runDevices["run1"] = "tv"
	saved, err := s.SaveValidatedFlow(ctx, "run1", "tv", "settings:v1", "", 0)
	require.NoError(t, err)
	restarted, err := internalflows.NewSQLiteLibrary(db)
	require.NoError(t, err)
	restored, err := restarted.Get(ctx, saved.ID, 1)
	require.NoError(t, err)
	require.Equal(t, flow, restored.Flow)
	require.Equal(t, []string{"verify"}, restored.Receipt.StepIDs)
	retry, err := s.SaveValidatedFlow(ctx, "run1", "tv", "settings:v1", "", 0)
	require.NoError(t, err)
	require.Equal(t, saved.ID, retry.ID)
	_, err = s.SaveValidatedFlow(ctx, "run1", "other", "settings:v1", "", 0)
	require.ErrorContains(t, err, "exact device")
	_, err = s.RunSavedFlow(ctx, saved.ID, 1, "other", "settings:v1", "test")
	require.ErrorContains(t, err, "mismatch")
	s.flowRuns["run2"] = Flow{Steps: []Step{{ID: "verify", Kind: "semantic-assert", Target: "Anything"}}}
	s.runs["run2"] = s.runs["run1"]
	s.runDevices["run2"] = "tv"
	_, err = s.SaveValidatedFlow(ctx, "run2", "tv", "settings:v1", saved.ID, 1)
	require.ErrorContains(t, err, "preserve assertion")
	s.flowRuns["run2"] = flow
	v2, err := s.SaveValidatedFlow(ctx, "run2", "tv", "settings:v1", saved.ID, 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, v2.Version)
	s.flowRuns["run3"] = flow
	s.runs["run3"] = s.runs["run1"]
	s.runDevices["run3"] = "tv"
	_, err = s.SaveValidatedFlow(ctx, "run3", "tv", "settings:v1", saved.ID, 1)
	require.ErrorContains(t, err, "version conflict")
	old, err := restarted.Get(ctx, saved.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "run1", old.SourceRunID)
}

func TestFlowPromotionRejectsUnverifiedAndInferredRuns(t *testing.T) { // [REQ:DVC-AGENT-REUSE]
	for _, kind := range []string{"failed", "incomplete", "vision", "no-assert", "dry-run"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := testService(t)
			f := Flow{Steps: []Step{{ID: "verify", Kind: "semantic-assert"}}}
			r := RunResult{Disposition: "passed"}
			switch kind {
			case "failed":
				r.Disposition = "failed"
			case "incomplete":
				r.Incomplete = true
			case "vision":
				r.Resolutions = []Resolution{{Rung: "vision"}}
			case "no-assert":
				f.Steps[0].Kind = "key"
			case "dry-run":
				f.SuppressActuation = true
			}
			s.flowRuns["run"] = f
			s.runs["run"] = r
			s.runDevices["run"] = "tv"
			_, err := s.SaveValidatedFlow(context.Background(), "run", "tv", "cohort", "", 0)
			require.Error(t, err)
		})
	}
}

func TestPropertyOutcomeReplayAndRestart(t *testing.T) { // [REQ:DVC-AGENT-REUSE]
	s, db := testService(t)
	adapter := fakes.NewPropertyOnly("property-fixture", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Writable: true, StateClass: strategy.StateBearing}, 20.0)
	s.registry = strategyregistry.New(adapter)
	s.devices.UpsertIdentity(devicedomain.Record{ID: "fixture-tv", StrategyID: adapter.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	ctx := context.Background()
	candidate := Flow{Transport: "rest", Steps: []Step{{ID: "verify-volume", Kind: "property-assert", Arguments: map[string]any{"name": "volume", "equals": 20.0}}}}
	result, err := s.Run(ctx, candidate, "fixture-tv", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	saved, err := s.SaveValidatedFlow(ctx, result.RunID, "fixture-tv", "fixture:volume:v1", "", 0)
	require.NoError(t, err)
	s.library, err = internalflows.NewSQLiteLibrary(db)
	require.NoError(t, err)
	replay, err := s.RunSavedFlow(ctx, saved.ID, saved.Version, "fixture-tv", "fixture:volume:v1", "test")
	require.NoError(t, err)
	require.Equal(t, "passed", replay.Disposition)
	candidate.Steps[0].Arguments["equals"] = 25.0
	failed, err := s.Run(ctx, candidate, "fixture-tv", "test")
	require.NoError(t, err)
	require.Equal(t, "failed", failed.Disposition)
	_, err = s.SaveValidatedFlow(ctx, failed.RunID, "fixture-tv", "fixture:volume:v1", saved.ID, 1)
	require.Error(t, err)
}
