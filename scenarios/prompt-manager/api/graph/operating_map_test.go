package graph

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/memberflow"
)

func TestBuildOperatingMap_ComposesTeamTopicEdgesWithoutMembers(t *testing.T) {
	models := []memberflow.OperatingModelDocument{{
		Team: "team-a",
		Graphs: []memberflow.OperatingGraphBlock{{Graph: memberflow.OperatingGraph{
			Nodes: []memberflow.OperatingGraphNode{
				{ID: "team", Kind: memberflow.OperatingGraphNodeKindTeam, Value: "team-a"},
				{ID: "topic", Kind: memberflow.OperatingGraphNodeKindTopic, Value: "a/output"},
				{ID: "member", Kind: memberflow.OperatingGraphNodeKindMember, Value: "writer"},
			},
			Edges: []memberflow.OperatingGraphEdge{{From: "team", To: "topic"}, {From: "member", To: "topic"}},
		}}},
	}}
	result := BuildOperatingMap(models, memberflow.OperatingGraphValidationResult{}, map[string]string{"team-a": "primary: The Forge"})
	if len(result.Teams) != 1 || result.Teams[0].GoalLinkage != "primary: The Forge" || !result.Teams[0].Valid {
		t.Fatalf("unexpected teams: %#v", result.Teams)
	}
	if len(result.Topics) != 1 || result.Topics[0].ID != "a/output" {
		t.Fatalf("unexpected topics: %#v", result.Topics)
	}
	if len(result.Edges) != 1 || result.Edges[0] != (OperatingMapEdge{From: "team-a", To: "a/output"}) {
		t.Fatalf("unexpected edges: %#v", result.Edges)
	}
}

func TestOperatingMapStore_CachesAndInvalidates(t *testing.T) {
	source := &stubOperatingMapSource{models: memberflow.OperatingModelListResponse{Models: []memberflow.OperatingModelDocument{{Team: "team-a"}}}}
	store := &OperatingMapStore{source: source, goalLinkages: map[string]string{}}
	if _, err := store.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.listCalls != 1 || source.validateCalls != 1 {
		t.Fatalf("expected one cached build, got list=%d validate=%d", source.listCalls, source.validateCalls)
	}
	store.Invalidate()
	if _, err := store.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source.listCalls != 2 || source.validateCalls != 2 {
		t.Fatalf("expected rebuild after invalidation, got list=%d validate=%d", source.listCalls, source.validateCalls)
	}
}

func TestLoadTeamGoalLinkages_UsesContributionMap(t *testing.T) {
	repoRoot, err := repoRootForOperatingMapTest()
	if err != nil {
		t.Fatal(err)
	}
	linkages, err := LoadTeamGoalLinkages(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got := linkages["monetization"]; got != "primary: Ledger — revenue & subscriptions; supporting: Broadcast — marketing & growth" {
		t.Fatalf("unexpected monetization linkage: %q", got)
	}
}

func repoRootForOperatingMapTest() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "docs", "director-swarm", "evidence", "OUTCOMES_CHARTER.md")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

type stubOperatingMapSource struct {
	models                   memberflow.OperatingModelListResponse
	validation               memberflow.OperatingModelValidationResponse
	listCalls, validateCalls int
}

func (s *stubOperatingMapSource) List(_ context.Context, _ memberflow.OperatingModelFilter) (memberflow.OperatingModelListResponse, error) {
	s.listCalls++
	return s.models, nil
}
func (s *stubOperatingMapSource) Validate(_ context.Context, _ memberflow.OperatingModelFilter) (memberflow.OperatingModelValidationResponse, error) {
	s.validateCalls++
	return s.validation, nil
}
