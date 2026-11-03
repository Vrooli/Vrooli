package routing_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/ai-provider-go/policy"

	"ai-gateway/internal/routing"
)

func TestPrivatePlanAndExecuteShareEligibilityAndReceiptSurvivesDisconnect(t *testing.T) {
	db := newSchemaDB(t)
	feature := privateFeatureFixture()
	eligibilityCalls := 0
	service := routing.NewPrivatePolicyService([]routing.PrivateFeature{feature}, func(context.Context, string) (routing.PrivatePolicySnapshot, error) {
		eligibilityCalls++
		return routing.PrivatePolicySnapshot{}, nil
	}, routing.NewSQLPrivatePlanRepository(db), routing.NewSQLPrivateReceiptRepository(db))

	plan, err := service.Plan(context.Background(), routing.PrivatePlanRequest{OperationID: "op-denied", FeatureID: feature.ID, Model: "model-request", BudgetMicros: 100_000})
	require.NoError(t, err)
	require.False(t, plan.Eligible, "zero approved records and routes must deny")
	require.Equal(t, "no_eligible_route", plan.Reason)
	require.Equal(t, 1, eligibilityCalls)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	receipt, err := service.Execute(ctx, routing.PrivateExecuteRequest{
		OperationID: "op-denied", Plan: plan,
		Feature: routing.PrivateFeatureInput{Instruction: "published instruction", Turn: privateTurnFixture(), Schema: json.RawMessage(`{"type":"object"}`)},
	})
	require.NoError(t, err)
	require.Equal(t, "failed", receipt.State)
	require.Equal(t, "no_eligible_route", receipt.ErrorCode)
	require.Equal(t, 2, eligibilityCalls, "execute must re-evaluate the same eligibility path")

	stored, err := service.Receipt(context.Background(), "op-denied")
	require.NoError(t, err)
	require.Equal(t, receipt, stored, "cancelled caller must not erase the durable terminal receipt")
}

func TestPrivateExecuteRejectsCallerMutatedPlanAndDoesNotChooseFallback(t *testing.T) {
	db := newSchemaDB(t)
	feature := privateFeatureFixture()
	terms := policy.ProviderTerms{
		PlanRules:     []policy.PlanRule{{Plan: "basic", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseBoundedFeature: policy.PermissionAllowed}}},
		Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "review-ref"},
		EffectiveFrom: "2026-01-01", OnExpiry: "deny", Downstream: policy.DownstreamRules{DataCollection: "deny"},
	}
	record, err := policy.NewVerifiedRuleRecord("supplier-a", terms, nil, func(reference string) bool { return reference == "review-ref" })
	require.NoError(t, err)
	routeCheap := policy.Route{ID: "route-cheap", Supplier: "supplier-a", Plan: "basic", Model: "model-cheap", Endpoint: feature.Role, Profile: feature.Profile, QuoteRef: "quote-1", QuotedCredits: 1}
	routeExpensive := policy.Route{ID: "route-expensive", Supplier: "supplier-a", Plan: "basic", Model: "model-expensive", Endpoint: feature.Role, Profile: feature.Profile, QuoteRef: "quote-2", QuotedCredits: 2}
	service := routing.NewPrivatePolicyService([]routing.PrivateFeature{feature}, func(context.Context, string) (routing.PrivatePolicySnapshot, error) {
		return routing.PrivatePolicySnapshot{Account: policy.Account{BalanceStanding: true}, Records: []policy.RuleRecord{record}, Routes: []policy.Route{routeExpensive, routeCheap}}, nil
	}, routing.NewSQLPrivatePlanRepository(db), routing.NewSQLPrivateReceiptRepository(db))

	plan, err := service.Plan(context.Background(), routing.PrivatePlanRequest{OperationID: "op-pinned", FeatureID: feature.ID, Model: "model-cheap", BudgetMicros: 100_000})
	require.NoError(t, err)
	require.True(t, plan.Eligible)
	require.Equal(t, "route-cheap", plan.RouteID)
	mutated := plan
	mutated.RouteID, mutated.Model, mutated.Supplier = routeExpensive.ID, routeExpensive.Model, routeExpensive.Supplier
	_, err = service.Execute(context.Background(), routing.PrivateExecuteRequest{OperationID: "op-pinned", Plan: mutated})
	require.ErrorIs(t, err, routing.ErrPrivatePlanConflict)

	receipt, err := service.Execute(context.Background(), routing.PrivateExecuteRequest{
		OperationID: "op-pinned", Plan: plan,
		Feature: routing.PrivateFeatureInput{Instruction: "published instruction", Turn: privateTurnFixture(), Schema: json.RawMessage(`{"type":"object"}`)},
	})
	require.NoError(t, err)
	require.Equal(t, "route_executor_unavailable", receipt.ErrorCode, "execution remains pinned to the selected route and never tries a second route")
}

func TestPrivatePlanIdempotencyRejectsDifferentRequestForSameOperation(t *testing.T) {
	db := newSchemaDB(t)
	feature := privateFeatureFixture()
	service := routing.NewPrivatePolicyService([]routing.PrivateFeature{feature}, nil,
		routing.NewSQLPrivatePlanRepository(db), routing.NewSQLPrivateReceiptRepository(db))
	first, err := service.Plan(context.Background(), routing.PrivatePlanRequest{OperationID: "same-operation", FeatureID: feature.ID, Model: "model-a"})
	require.NoError(t, err)
	second, err := service.Plan(context.Background(), routing.PrivatePlanRequest{OperationID: "same-operation", FeatureID: feature.ID, Model: "model-b"})
	require.ErrorIs(t, err, routing.ErrPrivatePlanConflict)
	require.NotEqual(t, "model-b", second.RequestedModel)
	require.Equal(t, "model-a", first.RequestedModel)
}

func privateFeatureFixture() routing.PrivateFeature {
	digest := sha256.Sum256([]byte("published instruction"))
	return routing.PrivateFeature{
		ID: "fixture.classify", Role: "chat.default", Profile: "remote-only", MaxInputBytes: 1024,
		MaxOutputBytes: 512, MaxUserTurns: 1, Schema: json.RawMessage(`{"type":"object"}`),
		PinnedInstructionDigest: hex.EncodeToString(digest[:]),
	}
}

func privateTurnFixture() routing.PrivateTurn {
	return routing.PrivateTurn{Role: "user", Content: `{"question":"classify"}`}
}

func TestPrivateReceiptConflictIsNotSilentlyReplayed(t *testing.T) {
	db := newSchemaDB(t)
	repository := routing.NewSQLPrivateReceiptRepository(db)
	first := routing.PrivateReceipt{OperationID: "op", State: "failed", ErrorCode: "denied", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	_, err := repository.Persist(context.Background(), "op", "digest-a", first)
	require.NoError(t, err)
	_, err = repository.Persist(context.Background(), "op", "digest-b", first)
	require.ErrorIs(t, err, routing.ErrPrivateReceiptConflict)
}

// Listing observes one authority snapshot; a later call must observe revocation.
func TestPrivateModelsLoadsOnceAndRechecksChangedApproval(t *testing.T) {
	feature := privateFeatureFixture()
	terms := policy.ProviderTerms{
		PlanRules:     []policy.PlanRule{{Plan: "basic", AccessTier: policy.AccessTierGrant, UseClasses: map[policy.UseClass]policy.Permission{policy.UseBoundedFeature: policy.PermissionAllowed}}},
		Review:        policy.Review{Status: policy.ReviewApproved, DecisionRef: "review-ref"},
		EffectiveFrom: "2026-01-01", OnExpiry: "deny", Downstream: policy.DownstreamRules{DataCollection: "deny"},
	}
	record, err := policy.NewVerifiedRuleRecord("supplier-a", terms, nil, func(string) bool { return true })
	require.NoError(t, err)
	snapshot := routing.PrivatePolicySnapshot{Account: policy.Account{BalanceStanding: true}, Records: []policy.RuleRecord{record}}
	for _, model := range []string{"model-a", "model-b"} {
		snapshot.Routes = append(snapshot.Routes, policy.Route{ID: model, Supplier: "supplier-a", Plan: "basic", Model: model, Endpoint: feature.Role, Profile: feature.Profile, QuoteRef: "quote", QuotedCredits: 1})
	}
	calls := 0
	db := newSchemaDB(t)
	service := routing.NewPrivatePolicyService([]routing.PrivateFeature{feature}, func(context.Context, string) (routing.PrivatePolicySnapshot, error) { calls++; return snapshot, nil }, routing.NewSQLPrivatePlanRepository(db), routing.NewSQLPrivateReceiptRepository(db))
	models, reason, err := service.Models(context.Background(), feature.ID, "")
	require.NoError(t, err)
	require.Equal(t, []string{"model-a", "model-b"}, models)
	require.Empty(t, reason)
	require.Equal(t, 1, calls, "listing must read and verify authority once regardless of model count")
	snapshot.Records = nil
	plan, err := service.Plan(context.Background(), routing.PrivatePlanRequest{OperationID: "after-revoke", FeatureID: feature.ID, Model: "model-a"})
	require.NoError(t, err)
	require.False(t, plan.Eligible, "listing cannot authorize a later execution")
	require.Equal(t, 2, calls)
	models, _, err = service.Models(context.Background(), feature.ID, "")
	require.NoError(t, err)
	require.Empty(t, models, "a later picker must observe changed authority")
	require.Equal(t, 3, calls)
}
