package graph

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"prompt-manager/internal/memberflow"
)

// OperatingMap is the swarm-scope Flow projection of the operating contracts.
// It deliberately contains teams and topics only; member detail belongs to the
// team-scope Topics projection.
type OperatingMap struct {
	Teams  []OperatingMapTeam  `json:"teams"`
	Topics []OperatingMapTopic `json:"topics"`
	Edges  []OperatingMapEdge  `json:"edges"`
}

type OperatingMapTeam struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	GoalLinkage string `json:"goal_linkage"`
	Valid       bool   `json:"valid"`
}

type OperatingMapTopic struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type OperatingMapEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// operatingMapSource is intentionally narrow so composition and cache tests
// can use deterministic inputs without parsing authored documents.
type operatingMapSource interface {
	List(context.Context, memberflow.OperatingModelFilter) (memberflow.OperatingModelListResponse, error)
	Validate(context.Context, memberflow.OperatingModelFilter) (memberflow.OperatingModelValidationResponse, error)
}

// OperatingMapStore builds the map once and shares it between API consumers.
// GraphIndexStore invalidates this store whenever authored graph inputs change.
type OperatingMapStore struct {
	source       operatingMapSource
	goalLinkages map[string]string
	mu           sync.RWMutex
	cached       *OperatingMap
}

func NewOperatingMapStore(source operatingMapSource, repoRoot string) (*OperatingMapStore, error) {
	linkages, err := LoadTeamGoalLinkages(repoRoot)
	if err != nil {
		return nil, err
	}
	return &OperatingMapStore{source: source, goalLinkages: linkages}, nil
}

func (s *OperatingMapStore) Get(ctx context.Context) (OperatingMap, error) {
	s.mu.RLock()
	if s.cached != nil {
		result := *s.cached
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cached != nil {
		return *s.cached, nil
	}
	models, err := s.source.List(ctx, memberflow.OperatingModelFilter{})
	if err != nil {
		return OperatingMap{}, err
	}
	validation, err := s.source.Validate(ctx, memberflow.OperatingModelFilter{})
	if err != nil {
		return OperatingMap{}, err
	}
	result := BuildOperatingMap(models.Models, validation.Validation, s.goalLinkages)
	s.cached = &result
	return result, nil
}

func (s *OperatingMapStore) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
}

// BuildOperatingMap composes parsed operating-model documents. It never
// reparses Mermaid: the contract parser remains the sole source of graph data.
func BuildOperatingMap(models []memberflow.OperatingModelDocument, validation memberflow.OperatingGraphValidationResult, goalLinkages map[string]string) OperatingMap {
	invalid := make(map[string]bool)
	for _, finding := range validation.Findings {
		if finding.Team != "" {
			invalid[finding.Team] = true
		}
	}
	result := OperatingMap{}
	topics := map[string]bool{}
	edges := map[string]OperatingMapEdge{}
	for _, model := range models {
		if model.Team == "" {
			continue
		}
		result.Teams = append(result.Teams, OperatingMapTeam{
			ID: model.Team, Label: model.Team, GoalLinkage: goalLinkages[model.Team], Valid: !invalid[model.Team],
		})
		for _, block := range model.Graphs {
			nodes := make(map[string]memberflow.OperatingGraphNode, len(block.Graph.Nodes))
			for _, node := range block.Graph.Nodes {
				nodes[node.ID] = node
			}
			for _, edge := range block.Graph.Edges {
				from, fromOK := nodes[edge.From]
				to, toOK := nodes[edge.To]
				if !fromOK || !toOK {
					continue
				}
				if from.Kind == memberflow.OperatingGraphNodeKindTeam && to.Kind == memberflow.OperatingGraphNodeKindTopic {
					topics[to.Value] = true
					edges[from.Value+"\x00"+to.Value] = OperatingMapEdge{From: from.Value, To: to.Value}
				}
				if from.Kind == memberflow.OperatingGraphNodeKindTopic && to.Kind == memberflow.OperatingGraphNodeKindTeam {
					topics[from.Value] = true
					edges[from.Value+"\x00"+to.Value] = OperatingMapEdge{From: from.Value, To: to.Value}
				}
			}
		}
	}
	for topic := range topics {
		result.Topics = append(result.Topics, OperatingMapTopic{ID: topic, Label: topic})
	}
	for _, edge := range edges {
		result.Edges = append(result.Edges, edge)
	}
	sort.Slice(result.Teams, func(i, j int) bool { return result.Teams[i].ID < result.Teams[j].ID })
	sort.Slice(result.Topics, func(i, j int) bool { return result.Topics[i].ID < result.Topics[j].ID })
	sort.Slice(result.Edges, func(i, j int) bool {
		return result.Edges[i].From+"\x00"+result.Edges[i].To < result.Edges[j].From+"\x00"+result.Edges[j].To
	})
	return result
}

// LoadTeamGoalLinkages derives concise per-team labels from the authoritative
// swarm contribution map, rather than duplicating outcome ownership in code.
func LoadTeamGoalLinkages(repoRoot string) (map[string]string, error) {
	data, err := os.ReadFile(repoRoot + "/docs/director-swarm/evidence/OUTCOMES_CHARTER.md")
	if err != nil {
		return nil, fmt.Errorf("read team contribution map: %w", err)
	}
	linkages := map[string][]string{}
	inTable := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## Team contribution map") {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(line, "## ") {
			break
		}
		if !inTable || !strings.HasPrefix(line, "| ") || strings.Contains(line, "Outcome category") || strings.Contains(line, "---") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 5 {
			continue
		}
		outcome := strings.TrimSpace(parts[1])
		for _, entry := range strings.Split(strings.TrimSpace(parts[2]), ",") {
			addGoalLinkage(linkages, entry, "primary: "+outcome)
		}
		for _, entry := range strings.Split(strings.TrimSpace(parts[3]), ",") {
			addGoalLinkage(linkages, entry, "supporting: "+outcome)
		}
	}
	result := make(map[string]string, len(linkages))
	for team, labels := range linkages {
		result[team] = strings.Join(labels, "; ")
	}
	return result, nil
}

func addGoalLinkage(linkages map[string][]string, raw, label string) {
	team := strings.Trim(strings.TrimSpace(raw), "`")
	team = strings.TrimPrefix(team, "team:")
	if team == "" || team == "all" {
		return
	}
	linkages[team] = append(linkages[team], label)
}
