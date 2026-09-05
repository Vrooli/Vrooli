package orchestration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/findings"
	"agent-manager/internal/promptmanager"

	"github.com/google/uuid"
)

type findingRepositorySpy struct {
	items []findings.Finding
}

func (r *findingRepositorySpy) Create(context.Context, *findings.Finding) error { return nil }
func (r *findingRepositorySpy) List(context.Context, findings.Filter) ([]findings.Finding, error) {
	return append([]findings.Finding(nil), r.items...), nil
}
func (r *findingRepositorySpy) SetDecision(context.Context, uuid.UUID, string) error { return nil }
func (r *findingRepositorySpy) RecurrenceCount(context.Context, string) (int, error) { return 0, nil }
func (r *findingRepositorySpy) SetEffectiveness(_ context.Context, id uuid.UUID, before, after *float64, effectiveness, topic string) error {
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].BeforeValue, r.items[i].AfterValue, r.items[i].Effectiveness, r.items[i].FrictionTopic = before, after, effectiveness, topic
		}
	}
	return nil
}

type frictionPublisherSpy struct {
	reports []promptmanager.FrictionReport
}

func (p *frictionPublisherSpy) ReadSkill(context.Context, string, map[string]string, bool) (string, error) {
	return "", nil
}

func (p *frictionPublisherSpy) PublishFriction(_ context.Context, report promptmanager.FrictionReport) (string, error) {
	p.reports = append(p.reports, report)
	return "friction-inbox/toolchain/agent-manager-finding-abc", nil
}

func TestRouteInvestigationFindingsPublishesOnceAndPersistsTopic(t *testing.T) {
	investigationID := uuid.New()
	repo := &findingRepositorySpy{items: []findings.Finding{{ID: uuid.New(), InvestigationRunID: investigationID, Fingerprint: "abc", Category: "Tooling", Recommendation: "Fix command", CreatedAt: time.Now()}}}
	publisher := &frictionPublisherSpy{}
	o := New(nil, nil, nil, WithFindings(repo), WithPromptClient(publisher))
	o.routeInvestigationFindings(context.Background(), investigationID)
	o.routeInvestigationFindings(context.Background(), investigationID)
	if len(publisher.reports) != 1 {
		t.Fatalf("published %d reports, want 1", len(publisher.reports))
	}
	if repo.items[0].FrictionTopic == "" {
		t.Fatal("friction topic was not persisted")
	}
}

func TestInvestigationSourceRunIDsReadsDurableWorkflowInput(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	input, err := json.Marshal(map[string]any{"runIds": []string{first.String(), "not-a-uuid", second.String()}})
	if err != nil {
		t.Fatal(err)
	}
	got := investigationSourceRunIDs(input)
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("source IDs = %v, want [%s %s]", got, first, second)
	}
}
