package backlog

import (
	"errors"
	"fmt"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/proto"
	"reflect"
	"swarm-manager/internal/identity"
	"testing"
)

func TestDevelopmentCanonicalPublicationGuardRefusalPreservesExactState(t *testing.T) {
	store := NewFileEffortControlStore(t.TempDir())
	control := testEffortControlRequest("guarded-fixture", 1)
	control.PolicyBinding = identity.PolicyBinding{Source: "fixture", Digest: developmentDigest("policy")}
	control.CandidatePolicy = testEffortCandidate("fixture/model")
	if err := store.Save(control); err != nil {
		t.Fatal(err)
	}
	before, err := store.Load(control.EffortID)
	if err != nil {
		t.Fatal(err)
	}
	next := before
	next.Completion.EvidenceComplete = true
	refusal := errors.New("fixture publication lease expired")
	calls := 0
	if err := store.withFiniteLock(control.EffortID, func() error { return store.saveUnlockedGuarded(next, func() error { calls++; return refusal }) }); !errors.Is(err, refusal) {
		t.Fatal(err)
	}
	after, _ := store.Load(control.EffortID)
	if calls != 1 || !reflect.DeepEqual(before, after) {
		t.Fatal("guard refusal changed canonical record")
	}
}

func TestDevelopmentCanonicalAcceptanceFailureAndRetainedReplay(t *testing.T) {
	for _, stage := range []string{"before-rename", "after-rename-directory-flush"} {
		t.Run(stage, func(t *testing.T) {
			owner, ctx, store, contract := developmentFixture(t)
			view, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: "approve-failure"}); err != nil {
				t.Fatal(err)
			}
			raw := []byte("exact fixture producer receipt")
			product := developmentDigest("fixture product")
			f := &developmentPersistenceEvidenceFixture{snapshot: DevelopmentEvidenceSnapshot{ProductDigest: product, EvidenceSetDigest: developmentDigest("set"), Version: "fixture", Receipts: []DevelopmentRetainedEvidence{{ID: "receipt", Producer: "test-genie", Digest: developmentBytesDigest(raw), Binding: []byte("binding"), Bytes: raw}}, Criteria: []*api.DevelopmentCriterionEvidence{{CriterionId: contract.RequiredCriteria[0], ReceiptIds: []string{"receipt"}, ReceiptSetDigest: developmentDigest("set"), TestedProductDigest: product, Satisfied: true}}}}
			owner.evidence = f
			before, _ := store.Load(contract.Subject.Effort)
			failure := errors.New("fixture controlled persistence failure")
			if stage == "before-rename" {
				f.guardError = failure
			} else {
				store.syncDirectory = func(string) error { return failure }
			}
			request := &api.AcceptDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(1), ExpectedDispositionVersion: developmentGeneration(0), RequestId: "accept-failure", TestedProductDigest: product}
			if _, err = owner.AcceptDevelopment(ctx, request); !errors.Is(err, failure) {
				t.Fatal("failure was lost", err)
			}
			after, _ := store.Load(contract.Subject.Effort)
			if stage == "before-rename" {
				if !reflect.DeepEqual(before, after) {
					t.Fatal("pre-rename refusal changed canonical standing")
				}
				return
			}
			if after.Development.DispositionVersion != 1 || len(after.Development.Decisions) != 2 {
				t.Fatal("post-rename failure lost exact published receipt")
			}
			// Replay must return the committed decision even though the flush reported
			// unknown durability; it must not read changed fixture evidence or save again.
			f.snapshot = DevelopmentEvidenceSnapshot{}
			first, err := owner.AcceptDevelopment(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			second, err := owner.AcceptDevelopment(ctx, request)
			if err != nil || !proto.Equal(first, second) {
				t.Fatal("exact retry duplicated or changed receipt", err)
			}
			replayed, _ := store.Load(contract.Subject.Effort)
			if !reflect.DeepEqual(after, replayed) {
				t.Fatal("replay changed published standing")
			}
		})
	}
}

func TestDevelopmentCanonicalConcurrentApprovalsCASAndExactReplay(t *testing.T) {
	for _, sameKey := range []bool{false, true} {
		t.Run(fmt.Sprintf("same-key-%v", sameKey), func(t *testing.T) {
			owner, ctx, store, contract := developmentFixture(t)
			view, err := owner.GetDevelopment(ctx, contract.Subject.Effort)
			if err != nil {
				t.Fatal(err)
			}
			start := make(chan struct{})
			type result struct {
				receipt *api.DevelopmentDecisionResponse
				err     error
			}
			results := make(chan result, 2)
			for i := 0; i < 2; i++ {
				go func(index int) {
					<-start
					key := fmt.Sprintf("concurrent-%d", index)
					if sameKey {
						key = "concurrent-exact"
					}
					receipt, e := owner.ApproveDevelopment(ctx, &api.ApproveDevelopmentRequest{Reference: view.Reference, ExpectedGeneration: developmentGeneration(0), RequestId: key})
					results <- result{receipt, e}
				}(i)
			}
			close(start)
			first, second := <-results, <-results
			success := 0
			for _, r := range []result{first, second} {
				if r.err == nil {
					success++
				}
			}
			expected := 1
			if sameKey {
				expected = 2
			}
			if success != expected {
				t.Fatalf("success count=%d expected=%d", success, expected)
			}
			if sameKey && !proto.Equal(first.receipt, second.receipt) {
				t.Fatal("same key concurrent replay changed exact receipt")
			}
			current, _ := store.Load(contract.Subject.Effort)
			if current.FiniteCommission.Generation != 1 || len(current.Development.Decisions) != 1 {
				t.Fatal("concurrent owners duplicated approval")
			}
		})
	}
}
