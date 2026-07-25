package related

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/records"
)

type testBacklog struct{ items []backlog.BacklogItem }

func (s testBacklog) LoadAll([]backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return s.items, nil
}

func (s testBacklog) LoadItem(k backlog.BacklogKind, n string) (backlog.BacklogItem, error) {
	for _, i := range s.items {
		if i.Kind == k && i.Name == n {
			return i, nil
		}
	}
	return backlog.BacklogItem{}, records.ErrNotFound
}

type testGoals struct{ items []goals.GoalWithScope }

func (s testGoals) List() ([]goals.GoalWithScope, error) { return s.items, nil }

type testRecords struct{ items []records.Record }

func (s testRecords) Create(records.Record) error { return nil }
func (s testRecords) Get(id string) (records.Record, error) {
	for _, record := range s.items {
		if record.ID == id {
			return record, nil
		}
	}
	return records.Record{}, records.ErrNotFound
}

type testSimilarity struct{ entities []Entity }

func (s testSimilarity) Similar(context.Context, TargetRef, int) ([]Entity, bool, error) {
	return s.entities, false, nil
}

func (s testRecords) List(f records.ListFilter) ([]records.Record, error) {
	var out []records.Record
	for _, r := range s.items {
		if f.BacklogRef != "" && r.BacklogRef != f.BacklogRef {
			continue
		}
		if f.MilestoneID != "" && r.MilestoneID != f.MilestoneID {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (s testRecords) FindByCaptureKey(string) (records.Record, bool, error) {
	return records.Record{}, false, nil
}

func (s testRecords) UpdateNarrative(string, records.Narrative, time.Time) (records.Record, error) {
	return records.Record{}, nil
}

func (s testRecords) UpdateDraft(string, records.Record) (records.Record, error) {
	return records.Record{}, nil
}

func (s testRecords) SetSupersededBy(string, string) (records.Record, error) {
	return records.Record{}, nil
}

func TestEngineGroupsLinkedScopeAndRecords(t *testing.T) {
	items := []backlog.BacklogItem{
		{Kind: backlog.KindIdea, Name: "source", Title: "Source", DependsOn: []string{"fix/dependency"}, Milestone: "one", AcceptanceAllow: []string{"scenarios/swarm-manager/api/**"}},
		{Kind: backlog.KindFix, Name: "dependency", Title: "Dependency", Milestone: "one"},
		{Kind: backlog.KindChore, Name: "dependent", Title: "Dependent", DependsOn: []string{"idea/source"}},
		{Kind: backlog.KindExecute, Name: "same", Title: "Same scope", Creates: []string{"scenarios/swarm-manager/api/new/**"}},
	}
	engine := NewEngine(testBacklog{items}, testGoals{[]goals.GoalWithScope{{Goal: goals.Goal{Name: "one", Title: "One"}, Scope: goals.Scope{Closure: []string{"idea/source", "fix/dependency"}}}}}, testRecords{[]records.Record{{ID: "r1", Trigger: "Record", BacklogRef: "idea/source"}}}, nil)
	report, err := engine.Compute(context.Background(), TargetRef{Kind: TargetBacklog, BacklogKind: backlog.KindIdea, Name: "source"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Linked.Entities) != 3 {
		t.Fatalf("linked=%d, want dependency, dependent, record", len(report.Linked.Entities))
	}
	var scoped *Entity
	for i := range report.SameScope.Entities {
		if report.SameScope.Entities[i].Key == "execute/same" {
			scoped = &report.SameScope.Entities[i]
		}
	}
	if scoped == nil {
		t.Fatalf("scope=%+v", report.SameScope.Entities)
	}
	if got := scoped.Reasons; len(got) < 2 {
		t.Fatalf("scope reasons=%v, want slug and prefix", got)
	}
}

func TestDedupeKeepsStrongestGroupAndMergesReasons(t *testing.T) {
	a, b, c := dedupe([]Entity{{Kind: EntityBacklog, Key: "idea/x", Reasons: []string{"linked"}}}, []Entity{{Kind: EntityBacklog, Key: "idea/x", Reasons: []string{"scope"}}}, []Entity{{Kind: EntityBacklog, Key: "idea/x", Reasons: []string{"similar"}}})
	if len(a) != 1 || len(b) != 0 || len(c) != 0 || len(a[0].Reasons) != 3 {
		t.Fatalf("dedupe=%+v/%+v/%+v", a, b, c)
	}
}

func TestEngineHydratesSimilarRecordTitleFromCanonicalStore(t *testing.T) {
	engine := NewEngine(
		testBacklog{[]backlog.BacklogItem{{Kind: backlog.KindIdea, Name: "source", Title: "Source"}}},
		testGoals{},
		testRecords{items: []records.Record{{ID: "rec-1", Trigger: "Readable record title", Outcome: records.OutcomeShipped}}},
		testSimilarity{entities: []Entity{{Kind: EntityRecord, Key: "rec-1", Title: "legacy-vector-point-id", ScorePercent: 80}}},
	)
	report, err := engine.Compute(context.Background(), TargetRef{Kind: TargetBacklog, BacklogKind: backlog.KindIdea, Name: "source"}, 20)
	if err != nil {
		t.Fatal(err)
	}
	if got := report.Similar.Entities[0]; got.Title != "Readable record title" || got.Status != string(records.OutcomeShipped) {
		t.Fatalf("similar record = %+v", got)
	}
}

func TestConnectServiceHonorsFiltersAndAlwaysReturnsGroups(t *testing.T) {
	engine := NewEngine(
		testBacklog{[]backlog.BacklogItem{{Kind: backlog.KindIdea, Name: "source", Title: "Source", DependsOn: []string{"fix/linked"}}, {Kind: backlog.KindFix, Name: "linked", Title: "Linked"}}},
		testGoals{}, testRecords{items: []records.Record{{ID: "record", Trigger: "Historical", BacklogRef: "idea/source"}}}, nil,
	)
	service := NewConnectService(engine)
	response, err := service.GetRelated(context.Background(), connect.NewRequest(&api.GetRelatedRequest{
		Target:            &api.GetRelatedRequest_Backlog{Backlog: &api.RelatedBacklogTarget{Kind: "idea", Name: "source"}},
		ExcludeHistorical: true,
		EntityKinds:       []string{"backlog"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Msg.Groups) != 3 {
		t.Fatalf("groups=%d", len(response.Msg.Groups))
	}
	if got := len(response.Msg.Groups[0].Entities); got != 1 || response.Msg.Groups[0].Entities[0].GetKey() != "fix/linked" {
		t.Fatalf("linked=%+v", response.Msg.Groups[0].Entities)
	}
}
