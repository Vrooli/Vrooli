package supervision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	scopesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	scopesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
)

const supervisionLedgerScope = "agent-manager-supervision"

// SourceLedgerOutcomeSink is an optional projection adapter. Agent Manager's
// SQLite outcome row remains canonical; discovery, scope provisioning, or
// append failures are returned to the caller as degradation evidence.
type SourceLedgerOutcomeSink struct {
	resolve func(context.Context) (string, error)
	client  *http.Client
}

func NewSourceLedgerOutcomeSink() *SourceLedgerOutcomeSink {
	return &SourceLedgerOutcomeSink{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "source-ledger")
		},
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *SourceLedgerOutcomeSink) AppendSupervisionOutcome(ctx context.Context, outcome SupervisionOutcome) (string, error) {
	base, err := s.resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("discover source-ledger: %w", err)
	}
	base = strings.TrimRight(base, "/")
	scopes := scopesconnect.NewScopesServiceClient(s.client, base)
	listed, err := scopes.ListScopes(ctx, connect.NewRequest(&scopesv1.ListScopesRequest{}))
	if err != nil {
		return "", fmt.Errorf("list source-ledger scopes: %w", err)
	}
	found := false
	for _, scope := range listed.Msg.GetScopes() {
		found = found || scope.GetId() == supervisionLedgerScope
	}
	if !found {
		_, err = scopes.CreateScope(ctx, connect.NewRequest(&scopesv1.CreateScopeRequest{Scope: &scopesv1.Scope{
			Id: supervisionLedgerScope, Label: "Agent Manager supervision outcomes", FrontierTarget: 32, WakeBudget: 64, MaxEntryLines: 4,
			Facets: []*scopesv1.FacetSpec{{Id: "supervision-outcomes", Label: "Supervision outcomes", Guidance: "Policy-labelled intervention outcomes and counterexamples", CompactionEligible: true, ResidentBudget: 32}},
		}}))
		if err != nil {
			return "", fmt.Errorf("create source-ledger supervision scope: %w", err)
		}
	}
	body, err := supervisionOutcomeLedgerBody(outcome)
	if err != nil {
		return "", err
	}
	journal := journalconnect.NewJournalServiceClient(s.client, base)
	response, err := journal.AppendEntry(ctx, connect.NewRequest(&journalv1.AppendEntryRequest{Scope: supervisionLedgerScope, FacetId: "supervision-outcomes", Kind: "supervision-outcome", Body: body, RequestKey: "supervision-outcome:" + outcome.ID}))
	if err != nil {
		return "", fmt.Errorf("append source-ledger supervision outcome: %w", err)
	}
	return response.Msg.GetEntry().GetId(), nil
}

func supervisionOutcomeLedgerBody(outcome SupervisionOutcome) (string, error) {
	payload := struct {
		OutcomeID, PolicyVersion, FamilyExecutionID, WatchID, DecisionID, ActionID, ChildRunID string
		EvidenceIDs                                                                            []string
		PredictedClass, ObservedClass                                                          string
		Overridden, Counterexample, SafetyViolation                                            bool
		CompletionImpact                                                                       float64
		CreatedAt                                                                              string
	}{
		OutcomeID: outcome.ID, PolicyVersion: outcome.PolicyVersion, FamilyExecutionID: outcome.FamilyExecutionID,
		WatchID: outcome.WatchID, DecisionID: outcome.DecisionID, ActionID: outcome.ActionID, ChildRunID: outcome.ChildRunID,
		EvidenceIDs: boundedPolicyEvidence(outcome.EvidenceIDs), PredictedClass: outcome.PredictedClass, ObservedClass: outcome.ObservedClass,
		Overridden: outcome.Overridden, Counterexample: outcome.Counterexample, SafetyViolation: outcome.SafetyViolation,
		CompletionImpact: outcome.CompletionImpact, CreatedAt: outcome.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode source-ledger supervision outcome: %w", err)
	}
	if len(raw) > 4096 {
		return "", errors.New("source-ledger supervision outcome exceeds 4096 bytes")
	}
	return string(raw), nil
}
