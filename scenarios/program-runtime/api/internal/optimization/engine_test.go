package optimization

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type memoryStore struct {
	mu   sync.Mutex
	rows map[string]Record
}

func newMemoryStore() *memoryStore { return &memoryStore{rows: map[string]Record{}} }
func (s *memoryStore) Create(_ context.Context, r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rows[r.Request.RunID]; ok {
		return Record{}, errors.New("duplicate")
	}
	s.rows[r.Request.RunID] = cloneRecordForTest(r)
	return cloneRecordForTest(r), nil
}

func (s *memoryStore) Load(_ context.Context, id string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.rows[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	return cloneRecordForTest(r), nil
}

func (s *memoryStore) CompareAndSwap(_ context.Context, rev uint64, r Record) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.rows[r.Request.RunID]
	if !ok {
		return Record{}, ErrNotFound
	}
	if old.Revision != rev {
		return Record{}, ErrRevision
	}
	s.rows[r.Request.RunID] = cloneRecordForTest(r)
	return cloneRecordForTest(r), nil
}

func (s *memoryStore) ListActive(_ context.Context, limit int) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Record, 0)
	for _, r := range s.rows {
		if r.Stage != StageComplete && r.Stage != StageRefused {
			out = append(out, cloneRecordForTest(r))
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func cloneRecordForTest(r Record) Record {
	copied, err := cloneRequest(r.Request)
	if err == nil {
		r.Request = copied
	}
	r.Evidence = cloneEvidence(r.Evidence)
	if r.StageAttempts != nil {
		m := make(map[Stage]int, len(r.StageAttempts))
		for k, v := range r.StageAttempts {
			m[k] = v
		}
		r.StageAttempts = m
	}
	return r
}

type fakeAdapter struct {
	rights, disk string
	calls        []Stage
	winner       string
	target       string
	evalFailures int
}

type rejectingValidationAdapter struct{ Adapter }

func (rejectingValidationAdapter) Validate(Request) error {
	return errors.New("target-specific shape is invalid")
}

func (a *fakeAdapter) Target() string {
	if a.target != "" {
		return a.target
	}
	return "ollama-role"
}

func (a *fakeAdapter) Validate(r Request) error {
	switch a.Target() {
	case "ollama-role":
		role, _ := r.Subject["role"].(string)
		if len(r.Candidates) != 3 || r.MinimumCases < 20 || strings.TrimSpace(role) == "" {
			return ErrInvalidRequest
		}
		for _, candidate := range r.Candidates {
			if strings.TrimSpace(candidate.ArtifactDigest) == "" {
				return ErrInvalidRequest
			}
		}
	case "digest-discovery-test":
		if len(r.Candidates) != 3 || r.MinimumCases < 20 {
			return ErrInvalidRequest
		}
	case "desktop-startup":
		if len(r.Candidates) != 1 || r.Candidates[0].ArtifactDigest == "" || r.PredecessorID == "" || r.MinimumCases < 30 {
			return ErrInvalidRequest
		}
	default:
		return ErrInvalidRequest
	}
	return nil
}

func (a *fakeAdapter) Acquire(_ context.Context, id string, r Request) (Evidence, error) {
	a.calls = append(a.calls, StageAcquire)
	digests := map[string]string{}
	for _, c := range r.Candidates {
		digests[c.ID] = c.ArtifactDigest
		if digests[c.ID] == "" {
			digests[c.ID] = "acquired:" + c.ID
		}
	}
	return Evidence{Stage: StageAcquire, ReceiptID: id + "/acquire", Details: map[string]any{"candidate_artifact_digests": digests}}, nil
}

func (a *fakeAdapter) Evaluate(_ context.Context, id string, r Request, _ []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StageEvaluate)
	if a.evalFailures > 0 {
		a.evalFailures--
		return Evidence{}, errors.New("temporary evaluation owner failure")
	}
	results := map[string]any{}
	for _, c := range r.Candidates {
		results[c.ID] = map[string]any{"case_count": len(r.CaseIDs), "case_digest": digestCases(r.CaseIDs), "score": 1.0}
	}
	return Evidence{Stage: StageEvaluate, ReceiptID: id + "/eval", Details: map[string]any{"suite_id": r.SuiteID, "case_count": len(r.CaseIDs), "case_digest": digestCases(r.CaseIDs), "candidate_results": results}}, nil
}

func (a *fakeAdapter) CheckRights(_ context.Context, id string, stage Stage, r Request, evidence []Evidence) (Evidence, error) {
	a.calls = append(a.calls, stage)
	digest := SubjectDigest(r)
	if stage == StageRights {
		digest = AcquiredSubjectDigest(r, evidence)
	}
	return Evidence{Stage: stage, ReceiptID: id + "/" + string(stage), Proof: &Proof{Decision: a.rights, ReceiptID: id + "/" + string(stage) + "-proof", SubjectDigest: digest, ExpiresAt: time.Now().Add(time.Hour)}}, nil
}

func (a *fakeAdapter) CheckDisk(_ context.Context, id string, stage Stage, r Request, evidence []Evidence) (Evidence, error) {
	a.calls = append(a.calls, stage)
	digest := SubjectDigest(r)
	if stage == StageDisk {
		digest = AcquiredSubjectDigest(r, evidence)
	}
	return Evidence{Stage: stage, ReceiptID: id + "/" + string(stage), Proof: &Proof{Decision: a.disk, ReceiptID: id + "/" + string(stage) + "-proof", SubjectDigest: digest, ExpiresAt: time.Now().Add(time.Hour), RequiredBytes: 10, AvailableBytes: 100}}, nil
}

func (a *fakeAdapter) Promote(_ context.Context, id string, r Request, evidence []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StagePromote)
	digest := acquiredDigests(evidence)[a.winner]
	return Evidence{Stage: StagePromote, ReceiptID: id + "/promote", Disposition: "promoted", Details: map[string]any{"winner_id": a.winner, "artifact_digest": digest}}, nil
}

func (a *fakeAdapter) Cleanup(_ context.Context, id string, r Request, _ []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StageCleanup)
	removed := make([]string, 0, len(r.Candidates))
	for _, c := range r.Candidates {
		if c.ID != a.winner {
			removed = append(removed, c.ID)
		}
	}
	if r.PredecessorID != "" && r.PredecessorID != a.winner {
		removed = append(removed, r.PredecessorID)
	}
	receipts := make(map[string]string, len(removed))
	for _, candidate := range removed {
		receipts[candidate] = "del-" + candidate
	}
	return Evidence{Stage: StageCleanup, ReceiptID: id + "/cleanup", Disposition: "cleanup_complete", Details: map[string]any{"removal_receipts": receipts}}, nil
}

func sampleRequest() Request {
	cases := make([]string, 20)
	for i := range cases {
		cases[i] = string(rune('a' + i))
	}
	return Request{RunID: "r-1", Target: "ollama-role", Subject: map[string]any{"role": "chat.default"}, Candidates: []Candidate{{ID: "m1", ArtifactDigest: "sha256:1"}, {ID: "m2", ArtifactDigest: "sha256:2"}, {ID: "m3", ArtifactDigest: "sha256:3"}}, SuiteID: "suite-1", CaseIDs: cases, MinimumCases: 20}
}

func TestRunPersistsEachOwnerReceiptAndCanResumeAfterRestart(t *testing.T) {
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	now := time.Now().UTC()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}, Now: func() time.Time { return now }}
	record, err := engine.Start(context.Background(), sampleRequest())
	require.NoError(t, err)
	for range 7 {
		record, err = engine.Advance(context.Background(), record.Request.RunID)
		require.NoError(t, err)
	}
	require.Equal(t, StageCleanup, record.Stage, "promotion receipt is durable before loser cleanup")
	// New engine instance models a process restart. It resumes from persisted
	// stage and does not repeat the earlier owner operations.
	restarted := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}, Now: func() time.Time { return now }}
	record, err = restarted.Run(context.Background(), sampleRequest())
	require.NoError(t, err)
	require.Equal(t, StageComplete, record.Stage)
	require.Equal(t, []Stage{StagePreflightRights, StagePreflightDisk, StageAcquire, StageEvaluate, StageRights, StageDisk, StagePromote, StageCleanup}, adapter.calls)
	require.Equal(t, "m2", record.WinnerID)
	require.Len(t, record.Evidence, 8)
}

func TestAcquiredDigestSubjectBindsFinalGateAndPromotion(t *testing.T) {
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	request := sampleRequest()
	request.Target = "digest-discovery-test"
	adapter.target = request.Target
	for i := range request.Candidates {
		request.Candidates[i].ArtifactDigest = ""
	}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{adapter.Target(): adapter}}
	record, err := engine.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, StageComplete, record.Stage)
	acquired := acquiredDigests(record.Evidence)
	require.Equal(t, map[string]string{"m1": "acquired:m1", "m2": "acquired:m2", "m3": "acquired:m3"}, acquired)
	require.NotEqual(t, SubjectDigest(request), AcquiredSubjectDigest(request, record.Evidence), "proof subject must bind to acquired content identities, not only model names")
	require.Equal(t, AcquiredSubjectDigest(request, record.Evidence), record.Evidence[4].Proof.SubjectDigest)
	require.Equal(t, acquired["m2"], record.Evidence[6].Details["artifact_digest"], "promotion must identify the acquired winner digest")
}

func TestAcquisitionArtifactMismatchRefusesBeforeEvaluation(t *testing.T) {
	store := newMemoryStore()
	base := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	adapter := &badAcquireAdapter{fakeAdapter: base}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
	record, err := engine.Run(context.Background(), sampleRequest())
	require.NoError(t, err)
	require.Equal(t, StageRefused, record.Stage)
	require.Equal(t, []Stage{StagePreflightRights, StagePreflightDisk, StageAcquire}, base.calls)
}

func TestAcquisitionRejectsDistinctCandidatesThatResolveToSameArtifact(t *testing.T) {
	base := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2", target: "digest-discovery-test"}
	adapter := &duplicateAcquiredDigestAdapter{fakeAdapter: base}
	request := sampleRequest()
	request.Target = adapter.Target()
	for i := range request.Candidates {
		request.Candidates[i].ArtifactDigest = ""
	}
	engine := &Engine{Store: newMemoryStore(), Adapters: map[string]Adapter{adapter.Target(): adapter}}
	record, err := engine.Run(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, StageRefused, record.Stage)
	require.Contains(t, record.Failure, "acquisition receipt does not match")
	require.NotContains(t, adapter.calls, StageEvaluate, "aliases must be rejected before evaluation")
}

type badAcquireAdapter struct{ *fakeAdapter }

func (a *badAcquireAdapter) Acquire(_ context.Context, id string, r Request) (Evidence, error) {
	a.calls = append(a.calls, StageAcquire)
	return Evidence{Stage: StageAcquire, ReceiptID: id + "/acquire", Details: map[string]any{"candidate_artifact_digests": map[string]string{"m1": "wrong", "m2": "sha256:2", "m3": "sha256:3"}}}, nil
}

type duplicateAcquiredDigestAdapter struct{ *fakeAdapter }

func (a *duplicateAcquiredDigestAdapter) Acquire(_ context.Context, id string, r Request) (Evidence, error) {
	a.calls = append(a.calls, StageAcquire)
	digests := make(map[string]string, len(r.Candidates))
	for _, candidate := range r.Candidates {
		digests[candidate.ID] = "sha256:same-content"
	}
	return Evidence{Stage: StageAcquire, ReceiptID: id + "/acquire", Details: map[string]any{"candidate_artifact_digests": digests}}, nil
}

type badPromotionAdapter struct{ *fakeAdapter }

func (a *badPromotionAdapter) Promote(_ context.Context, id string, r Request, _ []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StagePromote)
	return Evidence{Stage: StagePromote, ReceiptID: id + "/promote", Disposition: "promoted", Details: map[string]any{"winner_id": "m2", "artifact_digest": "wrong"}}, nil
}

type badCleanupAdapter struct{ *fakeAdapter }

func (a *badCleanupAdapter) Cleanup(_ context.Context, id string, _ Request, _ []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StageCleanup)
	return Evidence{Stage: StageCleanup, ReceiptID: id + "/cleanup", Disposition: "cleanup_complete", Details: map[string]any{"removal_receipts": map[string]string{"m1": "del-m1"}}}, nil
}

func TestPromotionAndCleanupRequireExactArtifactReceipts(t *testing.T) {
	for _, tc := range []struct {
		name        string
		adapter     Adapter
		wantCleanup bool
	}{{"wrong-promotion-digest", &badPromotionAdapter{fakeAdapter: &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}, false}, {"missing-loser-receipt", &badCleanupAdapter{fakeAdapter: &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}}, true}} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": tc.adapter}}
			record, err := engine.Run(context.Background(), sampleRequest())
			require.NoError(t, err)
			require.Equal(t, StageRefused, record.Stage)
			base := adapterBase(tc.adapter)
			if tc.wantCleanup {
				require.Contains(t, base.calls, StageCleanup)
			} else {
				require.NotContains(t, base.calls, StageCleanup)
			}
		})
	}
}

func adapterBase(adapter Adapter) *fakeAdapter {
	switch a := adapter.(type) {
	case *badPromotionAdapter:
		return a.fakeAdapter
	case *badCleanupAdapter:
		return a.fakeAdapter
	default:
		return nil
	}
}

func TestRightsOrDiskRefusalNeverPromotesOrCleans(t *testing.T) {
	for _, tc := range []struct{ name, rights, disk string }{{"rights-denied", "deny", "allow"}, {"rights-unknown", "unknown", "allow"}, {"disk-denied", "allow", "deny"}} {
		t.Run(tc.name, func(t *testing.T) {
			store := newMemoryStore()
			adapter := &fakeAdapter{rights: tc.rights, disk: tc.disk, winner: "m2"}
			engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
			record, err := engine.Run(context.Background(), sampleRequest())
			require.NoError(t, err)
			require.Equal(t, StageRefused, record.Stage)
			require.NotContains(t, adapter.calls, StagePromote)
			require.NotContains(t, adapter.calls, StageCleanup)
		})
	}
}

func TestDiskProofMustDeclarePositiveRequiredBytesAtBothGateStages(t *testing.T) {
	for _, stage := range []Stage{StagePreflightDisk, StageDisk} {
		t.Run(string(stage), func(t *testing.T) {
			base := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
			adapter := &incompleteDiskProofAdapter{fakeAdapter: base, invalidStage: stage}
			engine := &Engine{Store: newMemoryStore(), Adapters: map[string]Adapter{"ollama-role": adapter}}
			record, err := engine.Run(context.Background(), sampleRequest())
			require.NoError(t, err)
			require.Equal(t, StageRefused, record.Stage)
			require.Contains(t, record.Failure, "capacity proof")
			if stage == StagePreflightDisk {
				require.NotContains(t, base.calls, StageAcquire, "invalid preflight capacity proof must not allow acquisition")
			} else {
				require.NotContains(t, base.calls, StagePromote, "invalid refreshed capacity proof must not allow promotion")
			}
		})
	}
}

func TestInvalidEvaluationDenominatorRefusesBeforeFinalRightsOrMutation(t *testing.T) {
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	// A wrapper deliberately returns evidence for a different denominator.
	broken := &badEvalAdapter{fakeAdapter: adapter}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": broken}}
	record, err := engine.Run(context.Background(), sampleRequest())
	require.NoError(t, err)
	require.Equal(t, StageRefused, record.Stage)
	require.Equal(t, []Stage{StagePreflightRights, StagePreflightDisk, StageAcquire, StageEvaluate}, adapter.calls)
}

type badEvalAdapter struct{ *fakeAdapter }

func (a *badEvalAdapter) Evaluate(_ context.Context, id string, r Request, _ []Evidence) (Evidence, error) {
	a.calls = append(a.calls, StageEvaluate)
	return Evidence{Stage: StageEvaluate, ReceiptID: id + "/eval", Details: map[string]any{"suite_id": r.SuiteID, "case_count": len(r.CaseIDs), "case_digest": "other", "candidate_results": map[string]any{}}}, nil
}

type incompleteDiskProofAdapter struct {
	*fakeAdapter
	invalidStage Stage
}

func (a *incompleteDiskProofAdapter) CheckDisk(ctx context.Context, id string, stage Stage, request Request, evidence []Evidence) (Evidence, error) {
	if stage != a.invalidStage {
		return a.fakeAdapter.CheckDisk(ctx, id, stage, request, evidence)
	}
	a.calls = append(a.calls, stage)
	digest := SubjectDigest(request)
	if stage == StageDisk {
		digest = AcquiredSubjectDigest(request, evidence)
	}
	return Evidence{Stage: stage, ReceiptID: id + "/" + string(stage), Proof: &Proof{Decision: "allow", ReceiptID: id + "/disk-proof", SubjectDigest: digest, ExpiresAt: time.Now().Add(time.Hour), AvailableBytes: 100}}, nil
}

func TestStartPinsRunIDToImmutableInputs(t *testing.T) {
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
	req := sampleRequest()
	_, err := engine.Start(context.Background(), req)
	require.NoError(t, err)
	req.Candidates[0].ArtifactDigest = "changed"
	_, err = engine.Start(context.Background(), req)
	require.ErrorIs(t, err, ErrInvalidRequest)
}

func TestRequestRejectsNonCanonicalRunAndEvaluationArtifactIDs(t *testing.T) {
	for _, mutate := range []func(*Request){
		func(r *Request) { r.RunID = " " + r.RunID },
		func(r *Request) { r.Target = " " + r.Target },
		func(r *Request) { r.SuiteID = r.SuiteID + " " },
		func(r *Request) { r.Candidates[0].ID = " " + r.Candidates[0].ID },
		func(r *Request) { r.Candidates[0].ArtifactDigest = " " + r.Candidates[0].ArtifactDigest },
		func(r *Request) { r.Candidates[0].PredecessorDigest = " predecessor-digest " },
		func(r *Request) { r.PredecessorID = " predecessor " },
		func(r *Request) { r.CaseIDs[0] = r.CaseIDs[0] + " " },
	} {
		request := sampleRequest()
		mutate(&request)
		if err := validateRequest(request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("validateRequest(%+v) error = %v, want invalid request", request, err)
		}
	}
}

func TestTargetAdapterValidationErrorsRemainCallerInput(t *testing.T) {
	store := newMemoryStore()
	adapter := rejectingValidationAdapter{Adapter: &fakeAdapter{target: "generic-test"}}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"generic-test": adapter}}
	request := sampleRequest()
	request.Target = "generic-test"
	_, err := engine.Start(context.Background(), request)
	require.ErrorIs(t, err, ErrInvalidRequest)
	require.Contains(t, err.Error(), "target-specific shape is invalid")
	_, err = store.Load(context.Background(), request.RunID)
	require.ErrorIs(t, err, ErrNotFound, "invalid target-specific request is rejected before journaling")
}

func TestSingleCandidateOptimizationRequiresPredecessorForCleanup(t *testing.T) {
	request := Request{
		RunID: "single-without-predecessor", Target: "generic-test", Subject: map[string]any{},
		Candidates: []Candidate{{ID: "candidate", ArtifactDigest: "sha256:candidate"}},
		SuiteID:    "suite", CaseIDs: []string{"case"}, MinimumCases: 1,
	}
	store := newMemoryStore()
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"generic-test": &fakeAdapter{target: "generic-test"}}}
	if _, err := engine.Start(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Start() error = %v, want invalid request before any owner side effect", err)
	}
	if _, err := store.Load(context.Background(), request.RunID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid run was persisted: %v", err)
	}
}

func TestFailDoesNotRewriteTerminalRun(t *testing.T) {
	for _, stage := range []Stage{StageComplete, StageRefused} {
		t.Run(string(stage), func(t *testing.T) {
			store := newMemoryStore()
			request, err := cloneRequest(sampleRequest())
			require.NoError(t, err)
			initial := Record{Request: request, Stage: stage, Revision: 3, Failure: "original outcome", UpdatedAt: time.Now().UTC()}
			_, err = store.Create(context.Background(), initial)
			require.NoError(t, err)
			engine := &Engine{Store: store}
			got, err := engine.Fail(context.Background(), request.RunID, "late failure")
			require.NoError(t, err)
			require.Equal(t, stage, got.Stage)
			require.Equal(t, uint64(3), got.Revision)
			require.Equal(t, "original outcome", got.Failure)
		})
	}
}

func TestSameEngineAcceptsDesktopAdapterSingleCandidatePredecessorPlan(t *testing.T) {
	cases := make([]string, 30)
	for i := range cases {
		cases[i] = "pair-phase-" + strconv.Itoa(i)
	}
	req := Request{RunID: "dp1-1", Target: "desktop-startup", Subject: map[string]any{"scenario": "app", "artifact_digest": "sha256:new"}, Candidates: []Candidate{{ID: "new-artifact", ArtifactDigest: "sha256:new", PredecessorDigest: "sha256:old"}}, PredecessorID: "old-artifact", SuiteID: "dp1-paired-v1", CaseIDs: cases, MinimumCases: 30}
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "new-artifact", target: "desktop-startup"}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"desktop-startup": adapter}}
	record, err := engine.Run(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, StageComplete, record.Stage)
	require.Equal(t, "new-artifact", record.WinnerID)
	cleanup := record.Evidence[len(record.Evidence)-1]
	require.Equal(t, map[string]string{"old-artifact": "del-old-artifact"}, cleanup.Details["removal_receipts"])
}

func TestManagerRecoversWorkAndStopsUnchangedRetriesAfterThreeAttempts(t *testing.T) {
	store := newMemoryStore()
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2", evalFailures: 5}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}, RetryDelay: func(int) time.Duration { return 0 }}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop()
	_, err = manager.Submit(ctx, sampleRequest())
	require.NoError(t, err)
	terminal, done, err := manager.Wait(ctx, "r-1", 5*time.Second)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, StageRefused, terminal.Stage)
	evalCount := 0
	for _, stage := range adapter.calls {
		if stage == StageEvaluate {
			evalCount++
		}
	}
	require.Equal(t, maxAttemptsPerStage, evalCount)
	require.Contains(t, terminal.Failure, "after 3 attempts")
}

func TestManagerRestartRecoversDurableSubmissionBeforeFirstStage(t *testing.T) {
	store := newMemoryStore()
	request := sampleRequest()
	pinned, err := cloneRequest(request)
	require.NoError(t, err)
	_, err = store.Create(context.Background(), Record{Request: pinned, Stage: StagePreflightRights, Evidence: []Evidence{}, StageAttempts: map[Stage]int{}, UpdatedAt: time.Now().UTC()})
	require.NoError(t, err)
	adapter := &fakeAdapter{rights: "allow", disk: "allow", winner: "m2"}
	engine := &Engine{Store: store, Adapters: map[string]Adapter{"ollama-role": adapter}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop()
	record, terminal, err := manager.Wait(ctx, request.RunID, 5*time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	require.Equal(t, StageComplete, record.Stage)
	require.Equal(t, []Stage{StagePreflightRights, StagePreflightDisk, StageAcquire, StageEvaluate, StageRights, StageDisk, StagePromote, StageCleanup}, adapter.calls)
}

func TestRecoveredRunRefusesOnceWhenTargetAdapterDisappears(t *testing.T) {
	store := newMemoryStore()
	req := sampleRequest()
	pinned, err := cloneRequest(req)
	require.NoError(t, err)
	_, err = store.Create(context.Background(), Record{Request: pinned, Stage: StagePreflightRights, Evidence: []Evidence{}, StageAttempts: map[Stage]int{}, UpdatedAt: time.Now().UTC()})
	require.NoError(t, err)
	engine := &Engine{Store: store, Adapters: map[string]Adapter{}}
	manager, err := NewManager(engine)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, manager.Start(ctx))
	defer manager.Stop()
	record, terminal, err := manager.Wait(ctx, req.RunID, 5*time.Second)
	require.NoError(t, err)
	require.True(t, terminal)
	require.Equal(t, StageRefused, record.Stage)
	require.Contains(t, record.Failure, "adapter became unavailable")
}
