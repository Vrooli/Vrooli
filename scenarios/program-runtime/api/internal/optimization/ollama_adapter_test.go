package optimization

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

func TestOllamaRoleAdapterAcquirePinsOnlyOwnerConfirmedDigests(t *testing.T) {
	var got *cliv1.InspectOllamaRoleCandidatesRequest
	path, handler := cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(req *cliv1.InspectOllamaRoleCandidatesRequest) string {
		got = req
		return `{"schema_version":"ollama.optimize.receipt.v1","receipt_id":"run-1:acquire:resource-ollama","request_id":"run-1:acquire","target":"ollama-role","role":"chat.default","status":"refused","candidates":[{"model":"m1","installed":true,"digest":"sha256:one","policy_eligible":true,"rights":"allowed"},{"model":"m2","installed":true,"digest":"sha256:two","policy_eligible":true,"rights":"allowed"},{"model":"m3","installed":true,"digest":"sha256:three","policy_eligible":true,"rights":"allowed"}],"refusals":["evaluation receipt is not verified by an owner API"]}`
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	adapter := NewOllamaRoleAdapterWithClient(server.Client(), func(_ context.Context, target string) (string, error) {
		if target != "vrooli" {
			t.Fatalf("root owner target = %q", target)
		}
		return server.URL, nil
	})
	record := ollamaRoleRequest()
	evidence, err := adapter.Acquire(context.Background(), record.RunID, record)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if got == nil || got.GetRequestId() != "run-1:acquire" || got.GetEvaluationReceiptId() != "" {
		t.Fatalf("owner request = %#v; evaluation receipt must remain absent until Evaluate", got)
	}
	digests, ok := evidence.Details["candidate_artifact_digests"].(map[string]string)
	if !ok || len(digests) != 3 || digests["m1"] != "sha256:one" || digests["m2"] != "sha256:two" || digests["m3"] != "sha256:three" {
		t.Fatalf("acquisition digests = %#v", evidence.Details["candidate_artifact_digests"])
	}
}

func TestOllamaRoleAdapterAcquireRefusesUnconfirmedDigest(t *testing.T) {
	path, handler := cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(*cliv1.InspectOllamaRoleCandidatesRequest) string {
		return `{"schema_version":"ollama.optimize.receipt.v1","receipt_id":"run-1:acquire:resource-ollama","request_id":"run-1:acquire","target":"ollama-role","role":"chat.default","status":"refused","candidates":[{"model":"m1","installed":true,"digest":"sha256:one","policy_eligible":true,"rights":"allowed"},{"model":"m2","installed":true,"digest":"sha256:two","policy_eligible":true,"rights":"allowed"},{"model":"m3","installed":true,"policy_eligible":true,"rights":"allowed"}]}`
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	adapter := NewOllamaRoleAdapterWithClient(server.Client(), func(context.Context, string) (string, error) { return server.URL, nil })
	if _, err := adapter.Acquire(context.Background(), "run-1", ollamaRoleRequest()); err == nil {
		t.Fatal("Acquire() succeeded without an owner-confirmed digest for every candidate")
	}
}

func TestOllamaRoleAdapterDoesNotInventEvaluationEvidence(t *testing.T) {
	adapter := NewOllamaRoleAdapterWithClient(http.DefaultClient, func(context.Context, string) (string, error) { return "", nil })
	if _, err := adapter.Evaluate(context.Background(), "run-1", ollamaRoleRequest(), nil); !errors.Is(err, ErrOwnerUnavailable) {
		t.Fatal("Evaluate() must remain unavailable until an evaluation owner exists")
	}
	for name, operation := range map[string]func() error{
		"disk": func() error {
			_, err := adapter.CheckDisk(context.Background(), "run-1", StageDisk, ollamaRoleRequest(), nil)
			return err
		},
		"promote": func() error {
			_, err := adapter.Promote(context.Background(), "run-1", ollamaRoleRequest(), nil)
			return err
		},
		"cleanup": func() error {
			_, err := adapter.Cleanup(context.Background(), "run-1", ollamaRoleRequest(), nil)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := operation(); !errors.Is(err, ErrOwnerUnavailable) {
				t.Fatalf("%s() error = %v; want ErrOwnerUnavailable", name, err)
			}
		})
	}
	missingInspectOwner := NewOllamaRoleAdapterWithClient(nil, nil)
	if _, err := missingInspectOwner.Acquire(context.Background(), "run-1", ollamaRoleRequest()); !errors.Is(err, ErrOwnerUnavailable) {
		t.Fatalf("Acquire() error = %v; want ErrOwnerUnavailable when no inspection owner is configured", err)
	}
}

func TestOllamaRoleAdapterRightsUsesFreshOwnerEvidenceAndRestores(t *testing.T) {
	request := ollamaRoleRequest()
	policySHA := strings.Repeat("a", 64)
	statuses := map[string]string{}
	decisionDigests := map[string]string{}
	for _, candidate := range request.Candidates {
		statuses[candidate.ID] = "allowed"
		decisionDigests[candidate.ID] = strings.Repeat("b", 64)
	}
	path, handler := cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(req *cliv1.InspectOllamaRoleCandidatesRequest) string {
		return rightsOwnerReceipt(t, req.GetRequestId(), request, statuses, decisionDigests, policySHA, nil)
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	adapter := NewOllamaRoleAdapterWithClient(server.Client(), func(context.Context, string) (string, error) { return server.URL, nil })

	allowed, err := adapter.CheckRights(context.Background(), request.RunID, StagePreflightRights, request, nil)
	if err != nil {
		t.Fatalf("preflight CheckRights() error = %v", err)
	}
	if allowed.Proof == nil || allowed.Proof.Decision != "allow" || allowed.Proof.SubjectDigest != SubjectDigest(request) {
		t.Fatalf("preflight rights proof = %+v; want allow bound to pinned request", allowed.Proof)
	}
	firstDigest := allowed.Details["candidate_rights_evidence"].(map[string]ollamaRoleRightsEvidence)[request.Candidates[0].ID].DecisionSHA256

	statuses[request.Candidates[0].ID] = "denied"
	decisionDigests[request.Candidates[0].ID] = strings.Repeat("c", 64)
	refused, err := adapter.CheckRights(context.Background(), request.RunID, StagePreflightRights, request, nil)
	if err != nil {
		t.Fatalf("crossed-context CheckRights() error = %v", err)
	}
	if refused.Proof == nil || refused.Proof.Decision != "deny" || refused.Details["refusals"] == nil {
		t.Fatalf("crossed-context rights proof = %+v; want explicit denial with reason", refused)
	}

	statuses[request.Candidates[0].ID] = "allowed"
	decisionDigests[request.Candidates[0].ID] = strings.Repeat("b", 64)
	restored, err := adapter.CheckRights(context.Background(), request.RunID, StagePreflightRights, request, nil)
	if err != nil {
		t.Fatalf("restored-context CheckRights() error = %v", err)
	}
	if restored.Proof == nil || restored.Proof.Decision != "allow" {
		t.Fatalf("restored rights proof = %+v; want allow", restored.Proof)
	}
	restoredDigest := restored.Details["candidate_rights_evidence"].(map[string]ollamaRoleRightsEvidence)[request.Candidates[0].ID].DecisionSHA256
	if restoredDigest != firstDigest {
		t.Fatalf("restored decision digest=%s, initial=%s; want exact restoration", restoredDigest, firstDigest)
	}
}

func TestOllamaRoleAdapterPostEvaluationRightsBindsAcquiredDigests(t *testing.T) {
	request := ollamaRoleRequest()
	request.Candidates[0].ArtifactDigest = "sha256:expected-one"
	request.Candidates[1].ArtifactDigest = "sha256:expected-two"
	request.Candidates[2].ArtifactDigest = "sha256:expected-three"
	actual := map[string]string{
		request.Candidates[0].ID: "sha256:actual-one",
		request.Candidates[1].ID: "sha256:actual-two",
		request.Candidates[2].ID: "sha256:actual-three",
	}
	policySHA := strings.Repeat("a", 64)
	prior := []Evidence{
		{Stage: StagePreflightRights, ReceiptID: "rights-preflight", Details: map[string]any{"policy_sha256": policySHA}},
		{Stage: StageAcquire, ReceiptID: "acquire-owner", Details: map[string]any{"candidate_artifact_digests": actual}},
	}
	statuses := map[string]string{}
	decisionDigests := map[string]string{}
	for _, candidate := range request.Candidates {
		statuses[candidate.ID] = "allowed"
		decisionDigests[candidate.ID] = strings.Repeat("b", 64)
	}
	path, handler := cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(req *cliv1.InspectOllamaRoleCandidatesRequest) string {
		return rightsOwnerReceipt(t, req.GetRequestId(), request, statuses, decisionDigests, policySHA, actual)
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	adapter := NewOllamaRoleAdapterWithClient(server.Client(), func(context.Context, string) (string, error) { return server.URL, nil })
	checked, err := adapter.CheckRights(context.Background(), request.RunID, StageRights, request, prior)
	if err != nil {
		t.Fatalf("post-evaluation CheckRights() error = %v", err)
	}
	if checked.Proof == nil || checked.Proof.Decision != "allow" || checked.Proof.SubjectDigest != AcquiredSubjectDigest(request, prior) {
		t.Fatalf("post-evaluation rights proof = %+v; want allow bound to acquired artifact digests", checked.Proof)
	}

	wrongActual := map[string]string{}
	for key, digest := range actual {
		wrongActual[key] = digest
	}
	wrongActual[request.Candidates[1].ID] = "sha256:changed-after-acquisition"
	path, handler = cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(req *cliv1.InspectOllamaRoleCandidatesRequest) string {
		return rightsOwnerReceipt(t, req.GetRequestId(), request, statuses, decisionDigests, policySHA, wrongActual)
	}})
	mux = http.NewServeMux()
	mux.Handle(path, handler)
	server = httptest.NewServer(mux)
	defer server.Close()
	adapter = NewOllamaRoleAdapterWithClient(server.Client(), func(context.Context, string) (string, error) { return server.URL, nil })
	changed, err := adapter.CheckRights(context.Background(), request.RunID, StageRights, request, prior)
	if err != nil {
		t.Fatalf("changed-digest CheckRights() error = %v", err)
	}
	if changed.Proof == nil || changed.Proof.Decision != "deny" {
		t.Fatalf("changed-digest proof=%+v; want denial", changed.Proof)
	}
}

func TestEnginePersistsOwnerRightsProofBeforeUnavailableDiskGate(t *testing.T) {
	request := ollamaRoleRequest()
	statuses := map[string]string{}
	decisionDigests := map[string]string{}
	for _, candidate := range request.Candidates {
		statuses[candidate.ID] = "allowed"
		decisionDigests[candidate.ID] = strings.Repeat("b", 64)
	}
	policySHA := strings.Repeat("a", 64)
	path, handler := cliv1connect.NewOllamaOptimizationServiceHandler(ollamaOwnerFixture{onInspect: func(req *cliv1.InspectOllamaRoleCandidatesRequest) string {
		return rightsOwnerReceipt(t, req.GetRequestId(), request, statuses, decisionDigests, policySHA, nil)
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	adapter := NewOllamaRoleAdapterWithClient(server.Client(), func(context.Context, string) (string, error) { return server.URL, nil })
	engine := Engine{Store: newMemoryStore(), Adapters: map[string]Adapter{"ollama-role": adapter}}
	if _, err := engine.Start(context.Background(), request); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	record, err := engine.Advance(context.Background(), request.RunID)
	if err != nil {
		t.Fatalf("owner-backed preflight rights Advance() error = %v", err)
	}
	if record.Stage != StagePreflightDisk || len(record.Evidence) != 1 || record.Evidence[0].Stage != StagePreflightRights || record.Evidence[0].Proof == nil || record.Evidence[0].Proof.Decision != "allow" {
		t.Fatalf("record after preflight rights = %+v; want durable allow receipt before disk gate", record)
	}
	if _, err := engine.Advance(context.Background(), request.RunID); !errors.Is(err, ErrOwnerUnavailable) {
		t.Fatalf("disk gate Advance() error = %v; want fail-closed unavailable disk owner", err)
	}
	record, err = engine.Store.Load(context.Background(), request.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Stage != StagePreflightDisk || len(record.Evidence) != 1 || !strings.Contains(record.LastError, "optimizer disk admission owner is unavailable") {
		t.Fatalf("record after absent disk owner = %+v; want no acquisition and visible disk wait", record)
	}
}

func rightsOwnerReceipt(t *testing.T, requestID string, request Request, statuses, decisionDigests map[string]string, policySHA string, digests map[string]string) string {
	t.Helper()
	role, _ := request.Subject["role"].(string)
	receipt := ollamaRoleOwnerReceipt{
		SchemaVersion: "ollama.optimize.receipt.v1", ReceiptID: requestID + ":resource-ollama",
		RequestID: requestID, Target: request.Target, Role: role, Status: "refused", PolicySHA256: policySHA,
		Candidates: make([]ollamaRoleOwnerCandidate, 0, len(request.Candidates)),
	}
	evaluatedAt := time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)
	for _, candidate := range request.Candidates {
		digest := candidate.ArtifactDigest
		if digests != nil {
			digest = digests[candidate.ID]
		}
		status := statuses[candidate.ID]
		decisionDigest := decisionDigests[candidate.ID]
		receipt.Candidates = append(receipt.Candidates, ollamaRoleOwnerCandidate{
			Model: candidate.ID, Installed: digests != nil, Digest: digest,
			PolicyEligible: true, Rights: status,
			RightsEvidence: ollamaRoleRightsEvidence{
				Status: status, DecisionSHA256: decisionDigest, EvaluatedAt: evaluatedAt,
				ContextID: "operator-stated-business-context", ContextVersion: "test-context-v1",
				ContextSHA256: strings.Repeat("d", 64), ArtifactID: "registry.ollama.ai/library/" + candidate.ID,
				ArtifactSHA256: strings.Repeat("e", 64), PolicySHA256: policySHA,
			},
		})
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal rights owner receipt: %v", err)
	}
	return string(encoded)
}

func TestOllamaRoleAdapterRequiresExpectedCandidateDigests(t *testing.T) {
	adapter := NewOllamaRoleAdapterWithClient(http.DefaultClient, func(context.Context, string) (string, error) { return "", nil })
	request := ollamaRoleRequest()
	request.Candidates[2].ArtifactDigest = ""
	if err := adapter.Validate(request); err == nil {
		t.Fatal("Validate() accepted an Ollama candidate without a pinned expected digest")
	}
}

func TestOllamaRoleAdapterRejectsAliasesOfTheSameArtifact(t *testing.T) {
	adapter := NewOllamaRoleAdapterWithClient(http.DefaultClient, func(context.Context, string) (string, error) { return "", nil })
	request := ollamaRoleRequest()
	request.Candidates[2].ArtifactDigest = request.Candidates[1].ArtifactDigest
	if err := adapter.Validate(request); err == nil {
		t.Fatal("Validate() accepted three candidate names that collapse to only two artifacts")
	}
}

func ollamaRoleRequest() Request {
	cases := make([]string, 20)
	for i := range cases {
		cases[i] = "case-" + string(rune('a'+i))
	}
	return Request{
		RunID: "run-1", Target: "ollama-role", Subject: map[string]any{"role": "chat.default"},
		Candidates: []Candidate{{ID: "m1", ArtifactDigest: "sha256:one"}, {ID: "m2", ArtifactDigest: "sha256:two"}, {ID: "m3", ArtifactDigest: "sha256:three"}},
		SuiteID:    "fixed-suite", CaseIDs: cases, MinimumCases: 20,
	}
}

type ollamaOwnerFixture struct {
	onInspect func(*cliv1.InspectOllamaRoleCandidatesRequest) string
}

func (f ollamaOwnerFixture) InspectRoleCandidates(_ context.Context, req *connect.Request[cliv1.InspectOllamaRoleCandidatesRequest]) (*connect.Response[cliv1.InspectOllamaRoleCandidatesResponse], error) {
	return connect.NewResponse(&cliv1.InspectOllamaRoleCandidatesResponse{ReceiptJson: f.onInspect(req.Msg)}), nil
}

var _ cliv1connect.OllamaOptimizationServiceHandler = ollamaOwnerFixture{}
