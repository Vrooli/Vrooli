package supervision

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

type recoveryRefusalController struct {
	*fakeActionController
	failure      error
	calls        int
	receiptReads int
	runID        uuid.UUID
	message, key string
	accepted     bool
}

func (c *recoveryRefusalController) ContinueRun(_ context.Context, id uuid.UUID, message, key string) error {
	c.calls++
	c.runID, c.message, c.key = id, message, key
	return c.failure
}

func (c *recoveryRefusalController) ContinuationAccepted(_ context.Context, id uuid.UUID, message, key string) (bool, error) {
	c.receiptReads++
	if id != c.runID || message != c.message || key != c.key {
		return false, errors.New("reconciliation changed the original continuation identity or payload")
	}
	return c.accepted, nil
}

func TestEffortRecoveryRefusalRequiresOwnerPreEffectEvidence(t *testing.T) {
	for _, tc := range []struct {
		name      string
		preEffect bool
		wrap      bool
	}{
		{name: "owner-pre-effect-refusal", preEffect: true},
		{name: "wrapped-owner-pre-effect-refusal", preEffect: true, wrap: true},
		{name: "same-code-without-owner-refusal"},
		{name: "wrapped-same-code-without-owner-refusal", wrap: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, repo, base := effortFixture(t)
			ctx := t.Context()
			e := grantFixture(t, s, base, domain.RunStatusNeedsReview)
			request := directiveFixture(s, e)
			request.Directive.RecoveryExpectation = &pb.EffortRecoveryExpectation{
				ProgressCondition:    "the assigned regression passes with an owner receipt",
				BaselineEvidenceRefs: []string{"test-genie:regression-before"},
			}
			providerError := domain.NewRunnerSessionExpiredError(domain.RunnerTypeOpenCode, errors.New("session unavailable"))
			var failure error = providerError
			if tc.preEffect {
				failure = domain.RefuseBeforeEffects(failure)
			}
			if tc.wrap {
				failure = fmt.Errorf("continuation owner: %w", failure)
			}
			controller := &recoveryRefusalController{fakeActionController: base, failure: failure}
			s.controller = controller
			actor := EffortActor{ID: e.SupervisorRunId}

			directive, err := s.RequestDirective(ctx, request, actor)
			want := pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_UNCERTAIN
			if tc.preEffect {
				want = pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_REFUSED
				if err != nil {
					t.Fatal("owner-confirmed refusal was left uncertain", err)
				}
			} else if !errors.Is(err, providerError) {
				t.Fatal("unconfirmed provider failure lost its uncertainty/error", err)
			}
			if directive == nil || directive.Delivery != want || directive.DeliveredAt != nil || controller.calls != 1 {
				t.Fatalf("incorrect delivery standing: directive=%v calls=%d", directive, controller.calls)
			}
			persisted, err := repo.GetEffortDirective(ctx, directive.DirectiveId)
			if err != nil || !proto.Equal(persisted, directive) {
				t.Fatal("delivery classification was not durable", persisted, err)
			}

			// Request replay and owner restart may inspect the original attempt,
			// but neither may issue another continuation under this directive.
			replayed, err := s.RequestDirective(ctx, request, actor)
			if err != nil || !proto.Equal(replayed, directive) || controller.calls != 1 {
				t.Fatal("request replay changed the original delivery or resent it", replayed, err)
			}
			restarted := NewEffortService(repo, controller, s.policies, s.config)
			restarted.now = s.now
			for i := 0; i < 2; i++ {
				got, err := restarted.deliverDirective(ctx, directive)
				if err != nil || !proto.Equal(got, directive) || controller.calls != 1 {
					t.Fatal("restart retried or rewrote unresolved/refused delivery", got, err)
				}
			}
			if tc.preEffect {
				if !directive.RefusalBeforeEffects {
					t.Fatal("owner's no-effect proof was not retained")
				}
				_, reconcileErr := restarted.UpdateDirective(ctx, &pb.UpdateEffortDirectiveRequest{DirectiveId: directive.DirectiveId, ExpectedRevision: directive.Revision, IdempotencyKey: "cannot-reopen-refusal", ReconcileDelivery: true}, actor)
				if reconcileErr == nil {
					t.Fatal("explicit reconciliation reopened a proven no-effect refusal")
				}
				if controller.receiptReads != 0 {
					t.Fatal("definitive refusal was treated as uncertain dispatch")
				}
				return
			}
			if controller.receiptReads != 2 {
				t.Fatal("uncertain delivery did not inspect the original owner receipt")
			}
			controller.accepted = true
			got, err := restarted.deliverDirective(ctx, directive)
			if err != nil || got.Delivery != pb.EffortDirectiveDelivery_EFFORT_DIRECTIVE_DELIVERY_DELIVERED || controller.calls != 1 || controller.receiptReads != 3 {
				t.Fatal("later admission receipt did not reconcile without resending", got, err)
			}
			if got.GetRecoveryVerification().GetState() != "pending" || !proto.Equal(got.SourceSnapshot, directive.SourceSnapshot) {
				t.Fatal("receipt reconciliation invented progress or changed the baseline", got)
			}
		})
	}
}
