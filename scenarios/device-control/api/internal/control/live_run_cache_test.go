package control

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestForgetLiveRunRemovesOnlyVolatileIndexes(t *testing.T) {
	s, _ := testService(t)
	ctx := context.Background()
	flow := Flow{Steps: []Step{{ID: "verify", Kind: "semantic-assert", Target: "Ready"}}}
	result := RunResult{RunID: "ordinary", Disposition: "passed", Binding: RunBinding{DeviceID: "tv", LeaseID: "lease-1"}, Chapters: []Chapter{{ID: "verify", Disposition: "passed", EvidenceIDs: []string{"evidence-1"}}}}
	s.runs[result.RunID] = result
	s.flowRuns[result.RunID] = flow
	s.runDevices[result.RunID] = "tv"
	s.artifacts["evidence-1"] = "/tmp/evidence-1.png"
	s.artifactKinds["evidence-1"] = "screenshot"
	s.audits = append(s.audits, Audit{ID: "audit-1"})

	require.NoError(t, s.RetainRun(ctx, result.RunID, time.Now().UTC().Add(time.Hour)))
	saved, err := s.SaveValidatedFlow(ctx, result.RunID, "tv", "living-room", "", 0)
	require.NoError(t, err)
	require.NoError(t, s.ForgetLiveRun(result.RunID))
	require.NotContains(t, s.runs, result.RunID)
	require.NotContains(t, s.flowRuns, result.RunID)
	require.NotContains(t, s.runDevices, result.RunID)
	ids, active := s.LiveRunInventory()
	require.Empty(t, ids)
	require.Zero(t, active)
	require.Equal(t, "/tmp/evidence-1.png", s.artifacts["evidence-1"])
	require.Equal(t, "screenshot", s.artifactKinds["evidence-1"])
	require.Equal(t, "audit-1", s.audits[0].ID)

	retained, err := s.RetainedRun(ctx, result.RunID)
	require.NoError(t, err)
	require.Equal(t, result, retained.Result)
	export, err := s.ExportFlow(result.RunID)
	require.NoError(t, err)
	require.Equal(t, flow.Steps, export.Flow.Steps)
	reopenedSaved, err := s.GetSavedFlow(ctx, saved.ID, saved.Version)
	require.NoError(t, err)
	require.Equal(t, saved, reopenedSaved)
}

func TestForgetLiveRunRefusesActiveCleanupWithoutCancelling(t *testing.T) {
	s, _ := testService(t)
	called := false
	result := RunResult{RunID: "active", Binding: RunBinding{LeaseID: "lease-active"}}
	s.runs[result.RunID] = result
	s.flowRuns[result.RunID] = Flow{}
	s.runDevices[result.RunID] = "tv"
	s.activeCancels[result.Binding.LeaseID] = func() { called = true }

	err := s.ForgetLiveRun(result.RunID)
	require.ErrorIs(t, err, ErrLiveRunInProgress)
	require.False(t, called)
	require.Contains(t, s.runs, result.RunID)
	require.Contains(t, s.flowRuns, result.RunID)
	require.Contains(t, s.runDevices, result.RunID)
}

func TestForgetLiveRunExcludesPromotedAgentEntries(t *testing.T) {
	s, _ := testService(t)
	s.agents["agent-1"] = AgentRun{ID: "agent-1", State: "completed", Result: RunResult{Disposition: "passed"}, PlannedSteps: []Step{{ID: "launch", Kind: "launch", Target: "com.example.tv"}}}
	promoted, err := s.PromoteAgent("agent-1")
	require.NoError(t, err)
	flowID := promoted.PromotedFlowID

	err = s.ForgetLiveRun(flowID)
	require.ErrorIs(t, err, ErrLiveRunNotEvictable)
	require.Contains(t, s.runs, flowID)
	require.Contains(t, s.flowRuns, flowID)
	require.NotContains(t, s.runDevices, flowID)
	require.Equal(t, flowID, s.agents["agent-1"].PromotedFlowID)
	require.Len(t, s.agents["agent-1"].PlannedSteps, 1)
}

func TestForgetLiveRunIsAtomicForMissingIndexesAndRepeatedRequests(t *testing.T) {
	s, _ := testService(t)
	s.runs["partial"] = RunResult{RunID: "partial"}
	s.flowRuns["partial"] = Flow{}
	err := s.ForgetLiveRun("partial")
	require.ErrorIs(t, err, ErrLiveRunInconsistent)
	require.Contains(t, s.runs, "partial")
	require.Contains(t, s.flowRuns, "partial")
	require.NotContains(t, s.runDevices, "partial")

	require.ErrorIs(t, s.ForgetLiveRun("missing"), ErrLiveRunNotFound)
}
