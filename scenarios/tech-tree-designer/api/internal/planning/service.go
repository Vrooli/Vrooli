package planning

import (
	"context"
	"fmt"
	"sort"
	"strings"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	planningv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning"
)

type Service struct {
	repo         Repository
	validator    ProtoValidator
	materializer Materializer
}

func NewService(repo Repository, validator ProtoValidator, materializer Materializer) *Service {
	return &Service{repo: repo, validator: validator, materializer: materializer}
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Scenario, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.TargetStability = firstNonEmpty(in.TargetStability, DefaultTargetStability)
	return s.repo.CreateScenario(ctx, in)
}

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Scenario, error) {
	return s.repo.ListScenarios(ctx, filter)
}

func (s *Service) Get(ctx context.Context, slug string) (Scenario, error) {
	return s.repo.GetScenario(ctx, slug)
}

func (s *Service) PutFile(ctx context.Context, in PutFileInput) (ProtoFile, error) {
	return s.repo.PutFile(ctx, in)
}

func (s *Service) DeleteFile(ctx context.Context, slug, path string) (bool, error) {
	return s.repo.DeleteFile(ctx, slug, path)
}

func (s *Service) Validate(ctx context.Context, slug string) (bool, []PlanFinding, error) {
	if s.validator == nil {
		return false, nil, fmt.Errorf("proto validator is not configured")
	}
	scenario, err := s.repo.GetScenario(ctx, slug)
	if err != nil {
		return false, nil, err
	}
	findings, err := s.validator.Validate(ctx, scenario)
	if err != nil {
		return false, nil, err
	}
	return !hasErrorFinding(findings), findings, nil
}

func (s *Service) Materialize(ctx context.Context, slug string) (MaterializeResult, error) {
	if s.materializer == nil {
		return MaterializeResult{}, fmt.Errorf("materializer is not configured")
	}
	scenario, err := s.repo.GetScenario(ctx, slug)
	if err != nil {
		return MaterializeResult{}, err
	}
	findings, err := s.validator.Validate(ctx, scenario)
	if err != nil {
		return MaterializeResult{}, err
	}
	if hasErrorFinding(findings) {
		return MaterializeResult{}, ErrInvalidArgument{Field: "scenario", Reason: "validation failed"}
	}
	return s.materializer.Materialize(ctx, scenario)
}

func (s *Service) PlannedGraph(ctx context.Context) (*graphv1.TechTreeGraph, error) {
	scenarios, err := s.repo.ListScenarios(ctx, ListFilter{})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graphv1.TechNode, 0, len(scenarios))
	var edges []*graphv1.TechEdge
	for _, scenario := range scenarios {
		nodes = append(nodes, &graphv1.TechNode{
			Scenario:       scenario.Slug,
			Kind:           graphv1.NodeKind_NODE_KIND_PLANNED,
			DisplayName:    firstNonEmpty(scenario.DisplayName, DefaultDisplayName(scenario.Slug)),
			TransportWorld: TransportWorldPlanned,
			Stability:      []string{DefaultTargetStability},
			Sector:         scenario.Sector,
			Tier:           scenario.Tier,
		})
		for _, file := range scenario.Files {
			for _, imp := range extractImports(file.Text) {
				to := scenarioFromProtoPath(imp)
				if to == "" || to == scenario.Slug {
					continue
				}
				edges = append(edges, &graphv1.TechEdge{
					FromScenario: scenario.Slug,
					ToScenario:   to,
					Evidence: []*graphv1.GraphEvidence{{
						Source:     graphv1.EvidenceSource_EVIDENCE_SOURCE_PLANNED_PROTO_IMPORT,
						ImportPath: imp,
						FromFile:   file.Path,
					}},
					TransportWorld: TransportWorldPlanned,
					Stability:      []string{DefaultTargetStability},
				})
			}
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].GetScenario() < nodes[j].GetScenario() })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].GetFromScenario() != edges[j].GetFromScenario() {
			return edges[i].GetFromScenario() < edges[j].GetFromScenario()
		}
		return edges[i].GetToScenario() < edges[j].GetToScenario()
	})
	return &graphv1.TechTreeGraph{Nodes: nodes, Edges: edges}, nil
}

func hasErrorFinding(findings []PlanFinding) bool {
	for _, finding := range findings {
		if finding.Severity == planningv1.PlanFindingSeverity_PLAN_FINDING_SEVERITY_ERROR {
			return true
		}
	}
	return false
}

func scenarioFromProtoPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	return strings.Split(path, "/")[0]
}
