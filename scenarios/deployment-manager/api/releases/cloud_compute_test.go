package releases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deployment-manager/profiles"
	"github.com/gorilla/mux"
)

type cloudComputeAdmissionFixture struct {
	quoteKey, instanceID string
	calls                int
	err                  error
}

func (f *cloudComputeAdmissionFixture) VerifyCloudComputeAdmission(_ context.Context, quoteKey, instanceID string) error {
	f.calls++
	f.quoteKey, f.instanceID = quoteKey, instanceID
	return f.err
}

func TestCloudStartRequiresAndBindsAdmittedQuoteAndInstanceBeforeEffects(t *testing.T) {
	valid := StartRequest{
		CloudManifest:                 json.RawMessage(`{"target":{"type":"vps","vps":{"compute_manager_instance_id":"cm-instance-1"}}}`),
		CloudSpendQuoteIdempotencyKey: "cm-spend-key", CloudComputeManagerInstanceID: "cm-instance-1",
	}
	verifier := &cloudComputeAdmissionFixture{}
	h := NewHandler(nil, nil, nil, nil, nil).WithCloudComputeAdmissionVerifier(verifier)
	if err := h.validateCloudComputeRequest(context.Background(), "web-console/staging", valid); err != nil {
		t.Fatal(err)
	}
	if verifier.calls != 1 || verifier.quoteKey != "cm-spend-key" || verifier.instanceID != "cm-instance-1" {
		t.Fatalf("verifier call = %+v", verifier)
	}

	cases := []struct {
		name    string
		request StartRequest
		want    string
	}{
		{name: "missing quote", request: StartRequest{CloudManifest: valid.CloudManifest, CloudComputeManagerInstanceID: valid.CloudComputeManagerInstanceID}, want: "quote/admission reference is required"},
		{name: "missing instance", request: StartRequest{CloudManifest: valid.CloudManifest, CloudSpendQuoteIdempotencyKey: valid.CloudSpendQuoteIdempotencyKey}, want: "instance reference is required"},
		{name: "mismatched target", request: StartRequest{CloudManifest: json.RawMessage(`{"target":{"vps":{"compute_manager_instance_id":"cm-other"}}}`), CloudSpendQuoteIdempotencyKey: "cm-spend-key", CloudComputeManagerInstanceID: "cm-instance-1"}, want: "does not match"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := verifier.calls
			err := h.validateCloudComputeRequest(context.Background(), "web-console/staging", tc.request)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v, want substring %q", err, tc.want)
			}
			if verifier.calls != before {
				t.Fatal("invalid cloud refs reached Compute Manager verifier")
			}
		})
	}
}

func TestCloudStartFailsClosedWhenVerifierIsUnavailable(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil)
	request := StartRequest{CloudManifest: json.RawMessage(`{"target":{"vps":{"compute_manager_instance_id":"cm-instance-1"}}}`), CloudSpendQuoteIdempotencyKey: "cm-spend-key", CloudComputeManagerInstanceID: "cm-instance-1"}
	err := h.validateCloudComputeRequest(context.Background(), "web-console/staging", request)
	if err == nil || !strings.Contains(err.Error(), "verification is unavailable") {
		t.Fatalf("error=%v", err)
	}
}

func TestOperationReadbackIncludesExactPersistedComputeReferences(t *testing.T) {
	request := DeployRequest{ProfileID: "web-console/staging", ReleaseID: "release-1", CloudManifest: json.RawMessage(`{"target":{"vps":{"compute_manager_instance_id":"cm-instance-exact"}}}`), CloudSpendQuoteIdempotencyKey: "cm-spend-exact", CloudComputeManagerInstanceID: "cm-instance-exact"}
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	operation := Operation{ID: operationIDForRelease(request.ReleaseID), RequestSnapshot: snapshot}
	hydrateOperationCloudComputeRefs(&operation)
	wire := operation.Proto()
	if operation.CloudSpendQuoteIdempotencyKey != "cm-spend-exact" || operation.CloudComputeManagerInstanceID != "cm-instance-exact" || wire.GetCloudSpendQuoteIdempotencyKey() != "cm-spend-exact" || wire.GetCloudComputeManagerInstanceId() != "cm-instance-exact" {
		t.Fatalf("readback operation=%+v wire=%+v", operation, wire)
	}
}

func TestCloudStartRejectsUnconfiguredReleaseProfile(t *testing.T) {
	verifier := &cloudComputeAdmissionFixture{}
	h := NewHandler(nil, nil, nil, nil, nil).WithCloudComputeAdmissionVerifier(verifier)
	request := StartRequest{CloudManifest: json.RawMessage(`{"target":{"vps":{"compute_manager_instance_id":"cm-instance-1"}}}`), CloudSpendQuoteIdempotencyKey: "cm-spend-key", CloudComputeManagerInstanceID: "cm-instance-1"}
	err := h.validateCloudComputeRequest(context.Background(), "other-profile", request)
	if err == nil || !strings.Contains(err.Error(), "no configured quote and spend ceiling policy") {
		t.Fatalf("error=%v", err)
	}
	if verifier.calls != 0 {
		t.Fatal("unconfigured profile reached admission verifier")
	}
}

func TestStartRefusesMissingOrUnadmittedComputeFactsBeforePersistingRelease(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		verifyErr error
	}{
		{name: "missing quote", body: `{"profile_id":"web-console/staging","git_commit_hash":"commit","release_version":"1.0.0","cloud_manifest":{"target":{"vps":{"compute_manager_instance_id":"cm-1"}}},"cloud_compute_manager_instance_id":"cm-1"}`},
		{name: "missing target", body: `{"profile_id":"web-console/staging","git_commit_hash":"commit","release_version":"1.0.0","cloud_manifest":{"target":{"vps":{"compute_manager_instance_id":"cm-1"}}},"cloud_spend_quote_idempotency_key":"quote-1"}`},
		{name: "over cap or stale receipt", body: `{"profile_id":"web-console/staging","git_commit_hash":"commit","release_version":"1.0.0","cloud_manifest":{"target":{"vps":{"compute_manager_instance_id":"cm-1"}}},"cloud_spend_quote_idempotency_key":"quote-1","cloud_compute_manager_instance_id":"cm-1"}`, verifyErr: errors.New("persisted quote exceeds A2 cap")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo, orch := newMockRepo(), &mockOrch{}
			verifier := &cloudComputeAdmissionFixture{err: tc.verifyErr}
			h := NewHandler(repo, &mockLPBSConfig{configs: map[string]*profiles.LPBSReleaseConfig{}}, nil, orch, nil).
				WithAuthorization(func(context.Context) error { return nil }).
				WithCloudComputeAdmissionVerifier(verifier).WithDurableOperations(true)
			req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/profiles/web-console/staging/releases/start", strings.NewReader(tc.body)), map[string]string{"id": "web-console/staging"})
			response := httptest.NewRecorder()
			h.Start(response, req)
			if response.Code != http.StatusPreconditionFailed {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if len(repo.store) != 0 || orch.called != 0 {
				t.Fatalf("invalid compute refs caused effects: releases=%d orchestration=%d", len(repo.store), orch.called)
			}
			if tc.verifyErr != nil && verifier.calls != 1 {
				t.Fatalf("verifier calls=%d, want one", verifier.calls)
			}
		})
	}
}

func TestCloudAcquireRequestIsBoundToApprovedStagingProfile(t *testing.T) {
	manifest := json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"web-console"},"edge":{"domain":"staging.vrooli.com"}}`)
	base := StartRequest{IdempotencyKey: "client-key", CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudManifest: manifest}
	if err := validateCloudAcquireRequest("web-console/staging", base); err != nil {
		t.Fatalf("valid A2 acquire refused: %v", err)
	}
	for name, mutate := range map[string]func(*StartRequest){
		"missing stable release key": func(r *StartRequest) { r.IdempotencyKey = "" },
		"unsupported region":         func(r *StartRequest) { r.CloudAcquireRegion = "ash" },
		"preselected instance":       func(r *StartRequest) { r.CloudComputeManagerInstanceID = "cm-guess" },
		"caller quote":               func(r *StartRequest) { r.CloudSpendQuoteIdempotencyKey = "quote-guess" },
		"wrong scenario": func(r *StartRequest) {
			r.CloudManifest = json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"other"},"edge":{"domain":"staging.vrooli.com"}}`)
		},
		"outside DNS scope": func(r *StartRequest) {
			r.CloudManifest = json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"web-console"},"edge":{"domain":"other.example"}}`)
		},
	} {
		t.Run(name, func(t *testing.T) {
			request := base
			mutate(&request)
			if err := validateCloudAcquireRequest("web-console/staging", request); err == nil {
				t.Fatal("invalid acquire was accepted")
			}
		})
	}
}

func TestCloudAcquireSnapshotRetainsStableKeyAndExactTargetAfterReadback(t *testing.T) {
	request := DeployRequest{ProfileID: "web-console/staging", ReleaseID: "release-1", IdempotencyKey: "start-key", CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudAcquireIdempotencyKey: "dm-acquire:release-op-release-1", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-release-1", CloudManifest: json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"web-console"},"edge":{"domain":"staging.vrooli.com"}}`)}
	boundManifest, err := bindCloudManifestInstance(request.CloudManifest, "cm-instance-1")
	if err != nil {
		t.Fatal(err)
	}
	request.CloudManifest, request.CloudComputeManagerInstanceID = boundManifest, "cm-instance-1"
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := deployRequestFromSnapshot(snapshot, nil)
	if err != nil {
		t.Fatal(err)
	}
	if restored.CloudAcquireIdempotencyKey != request.CloudAcquireIdempotencyKey || restored.CloudSpendQuoteIdempotencyKey != request.CloudSpendQuoteIdempotencyKey || restored.CloudComputeManagerInstanceID != "cm-instance-1" || !restored.CloudAcquireCompute {
		t.Fatalf("restored acquire request=%+v", restored)
	}
	var manifest releaseCloudTarget
	if err := json.Unmarshal(restored.CloudManifest, &manifest); err != nil || manifest.Target.VPS.ComputeManagerInstanceID != "cm-instance-1" {
		t.Fatalf("persisted manifest target=%q err=%v", manifest.Target.VPS.ComputeManagerInstanceID, err)
	}
}

func TestCloudManifestBindingAcceptsExactSavedTargetDuringEnrollmentResume(t *testing.T) {
	raw := json.RawMessage(`{"target":{"type":"vps","vps":{"compute_manager_instance_id":"cm-instance-1"}}}`)
	bound, err := bindCloudManifestInstance(raw, "cm-instance-1")
	if err != nil {
		t.Fatalf("same saved target was rejected on enrollment resume: %v", err)
	}
	var target releaseCloudTarget
	if err := json.Unmarshal(bound, &target); err != nil {
		t.Fatal(err)
	}
	if target.Target.VPS.ComputeManagerInstanceID != "cm-instance-1" {
		t.Fatalf("target=%q", target.Target.VPS.ComputeManagerInstanceID)
	}
}

func TestCloudAcquireIdempotentReplayReusesTheSavedAcquireIntent(t *testing.T) {
	snapshot, err := json.Marshal(durableDeployRequest{CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudAcquireIdempotencyKey: "dm-acquire:release-op-rel-1", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-rel-1", CloudManifest: json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"web-console"},"edge":{"domain":"staging.vrooli.com"}}`)})
	if err != nil {
		t.Fatal(err)
	}
	replay := StartRequest{CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudManifest: json.RawMessage(`{"environment":"staging","target":{"type":"vps","vps":{}},"scenario":{"id":"web-console"},"edge":{"domain":"staging.vrooli.com"}}`)}
	if !sameCloudOperationRequest(snapshot, replay) {
		t.Fatal("same acquire request did not bind to its persisted intent")
	}
	replay.CloudAcquireRegion = "fsn1"
	if sameCloudOperationRequest(snapshot, replay) {
		t.Fatal("changed acquisition region replayed the original operation")
	}
}
