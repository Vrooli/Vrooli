package optimization

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDesktopStartupAdapterMapsPinnedRequestAcquisitionAndProofs(t *testing.T) {
	adapter := NewDesktopStartupAdapter()
	request := desktopEngineRequest()
	evidence := desktopEngineEvidence(request)
	mapped, err := adapter.mapRequest(request, evidence, true)
	require.NoError(t, err)
	require.Equal(t, "desktop-startup.optimize-request.v1", mapped.Schema)
	require.Equal(t, request.RunID, mapped.RunID)
	require.Equal(t, request.Target, mapped.Target)
	require.Equal(t, "suite-v1", mapped.BaselineRef)
	require.Equal(t, "calculator", mapped.Subject.Scenario)
	require.Equal(t, "sha256:predecessor", mapped.Subject.ArtifactDigest)
	require.Equal(t, "host-1", mapped.Subject.HostFingerprint)
	require.Equal(t, "xvfb-0", mapped.Subject.Display)
	require.Equal(t, "local", mapped.Subject.DeploymentMode)
	require.Equal(t, "startup", mapped.Subject.ProfilerMode)
	require.Equal(t, map[string]string{"mode": "warm"}, mapped.Candidate.StartupProfile)
	require.Equal(t, "candidate-1", mapped.Candidate.ID)
	require.Equal(t, "sha256:candidate", mapped.Candidate.ArtifactDigest)
	require.Equal(t, "sha256:predecessor", mapped.Candidate.PredecessorDigest)
	require.Equal(t, "acquire-owner-receipt", mapped.Acquisition.ReceiptID)
	require.Equal(t, mapped.Candidate.ArtifactDigest, mapped.Acquisition.ArtifactDigest)
	require.Equal(t, SubjectDigest(request), mapped.PreflightRights.SubjectDigest)
	require.Equal(t, SubjectDigest(request), mapped.PreflightDisk.SubjectDigest)
	require.Equal(t, "rights-pre", mapped.PreflightRights.ReceiptID)
	require.Equal(t, "disk-pre", mapped.PreflightDisk.LeaseID)
	require.Equal(t, AcquiredSubjectDigest(request, evidence), mapped.Rights.SubjectDigest)
	require.Equal(t, AcquiredSubjectDigest(request, evidence), mapped.Disk.SubjectDigest)
	require.Equal(t, "rights-fresh", mapped.Rights.ReceiptID)
	require.Equal(t, "disk-fresh", mapped.Disk.LeaseID)
	require.Equal(t, desktopMinPairs, mapped.MinimumPairs)
	require.Equal(t, .05, mapped.MaxRegressionFraction)
}

func TestDesktopStartupAdapterFailsClosedWhileAcquisitionAndOtherOwnersAreAbsent(t *testing.T) {
	adapter := NewDesktopStartupAdapter()
	request := desktopEngineRequest()
	require.Equal(t, desktopStartupTarget, adapter.Target())
	require.NoError(t, adapter.Validate(request))

	_, err := adapter.Acquire(context.Background(), request.RunID, request)
	require.ErrorContains(t, err, "artifact acquisition owner is unavailable")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
	_, err = adapter.Evaluate(context.Background(), request.RunID, request, desktopEngineEvidence(request))
	require.ErrorContains(t, err, "DP1 measurement will not be called")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
	_, err = adapter.CheckRights(context.Background(), request.RunID, StageRights, request, nil)
	require.ErrorContains(t, err, "rights proof owner is unavailable")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
	_, err = adapter.CheckDisk(context.Background(), request.RunID, StageDisk, request, nil)
	require.ErrorContains(t, err, "disk admission owner is unavailable")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
	_, err = adapter.Promote(context.Background(), request.RunID, request, nil)
	require.ErrorContains(t, err, "promotion owners are unavailable")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
	_, err = adapter.Cleanup(context.Background(), request.RunID, request, nil)
	require.ErrorContains(t, err, "predecessor cleanup owners are unavailable")
	require.True(t, errors.Is(err, ErrOwnerUnavailable))
}

func TestDesktopStartupRequestMappingRejectsMismatchedAcquisitionOrProof(t *testing.T) {
	adapter := NewDesktopStartupAdapter()
	request := desktopEngineRequest()
	evidence := desktopEngineEvidence(request)
	evidence[2].Details["candidate_artifact_digests"] = map[string]string{"candidate-1": "sha256:other"}
	_, err := adapter.mapRequest(request, evidence, true)
	require.ErrorContains(t, err, "acquired digest differs")

	evidence = desktopEngineEvidence(request)
	evidence[3].Proof.SubjectDigest = "sha256:stale"
	_, err = adapter.mapRequest(request, evidence, true)
	require.ErrorContains(t, err, "refreshed rights proof")

	evidence = desktopEngineEvidence(request)
	evidence[1].Proof.Decision = "deny"
	_, err = adapter.mapRequest(request, evidence, false)
	require.ErrorContains(t, err, "preflight disk proof")
}

func desktopEngineRequest() Request {
	request := Request{
		RunID: "run-dp1", Target: desktopStartupTarget,
		Subject: map[string]any{
			"scenario": "calculator", "artifact_digest": "sha256:predecessor", "host_fingerprint": "host-1",
			"display": "xvfb-0", "deployment_mode": "local", "profiler_mode": "startup", "startup_profile": map[string]any{"mode": "warm"},
		},
		Candidates:    []Candidate{{ID: "candidate-1", ArtifactDigest: "sha256:candidate", PredecessorDigest: "sha256:predecessor"}},
		PredecessorID: "predecessor-1", SuiteID: "suite-v1", MinimumCases: 30,
	}
	for i := 0; i < 30; i++ {
		request.CaseIDs = append(request.CaseIDs, "case-"+time.Unix(int64(i), 0).UTC().Format("150405"))
	}
	return request
}

func desktopEngineEvidence(request Request) []Evidence {
	preDigest := SubjectDigest(request)
	acquisition := Evidence{Stage: StageAcquire, ReceiptID: "acquire-owner-receipt", Details: map[string]any{"candidate_artifact_digests": map[string]string{"candidate-1": "sha256:candidate"}}}
	evidence := []Evidence{
		{Stage: StagePreflightRights, Proof: &Proof{Decision: "allow", ReceiptID: "rights-pre", SubjectDigest: preDigest, ExpiresAt: time.Now().Add(time.Hour)}},
		{Stage: StagePreflightDisk, Proof: &Proof{Decision: "allow", ReceiptID: "disk-pre", SubjectDigest: preDigest, RequiredBytes: 1, AvailableBytes: 2, ExpiresAt: time.Now().Add(time.Hour)}},
		acquisition,
	}
	fullDigest := AcquiredSubjectDigest(request, evidence)
	evidence = append(evidence,
		Evidence{Stage: StageRights, Proof: &Proof{Decision: "allow", ReceiptID: "rights-fresh", SubjectDigest: fullDigest, ExpiresAt: time.Now().Add(time.Hour)}},
		Evidence{Stage: StageDisk, Proof: &Proof{Decision: "allow", ReceiptID: "disk-fresh", SubjectDigest: fullDigest, RequiredBytes: 1, AvailableBytes: 2, ExpiresAt: time.Now().Add(time.Hour)}},
	)
	return evidence
}
