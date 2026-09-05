package ontology

import (
	"context"
	"errors"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	apicoredb "github.com/vrooli/api-core/database"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

type fakeScenarioSource struct {
	graph *graphv1.TechTreeGraph
	err   error
}

func (f fakeScenarioSource) ScenarioGraph(context.Context) (*graphv1.TechTreeGraph, error) {
	return f.graph, f.err
}

func TestServiceRejectsParentCycles(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewService(NewSQLiteRepository(database))

	if _, err := service.UpsertCapability(ctx, Capability{Slug: "root", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR}); err != nil {
		t.Fatalf("UpsertCapability(root) error = %v", err)
	}
	if _, err := service.UpsertCapability(ctx, Capability{Slug: "child", ParentID: "root"}); err != nil {
		t.Fatalf("UpsertCapability(child) error = %v", err)
	}
	if _, err := service.UpsertCapability(ctx, Capability{Slug: "root", ParentID: "child"}); err == nil {
		t.Fatal("UpsertCapability(cycle) error = nil, want ErrCapabilityCycle")
	} else {
		var cycle ErrCapabilityCycle
		if !errors.As(err, &cycle) {
			t.Fatalf("UpsertCapability(cycle) error = %T %v, want ErrCapabilityCycle", err, err)
		}
	}
}

func TestServiceNormalizesFulfillment(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewService(NewSQLiteRepository(database))

	if _, err := service.UpsertCapability(ctx, Capability{Slug: "proto-foundation"}); err != nil {
		t.Fatalf("UpsertCapability() error = %v", err)
	}
	fulfillment, err := service.LinkFulfillment(ctx, Fulfillment{
		CapabilityID: "PROTO-FOUNDATION",
		ScenarioSlug: "PROTO-HEALTH",
		Note:         "  validates proto graph ",
	})
	if err != nil {
		t.Fatalf("LinkFulfillment() error = %v", err)
	}
	if fulfillment.CapabilityID != "proto-foundation" || fulfillment.ScenarioSlug != "proto-health" || fulfillment.Note != "validates proto graph" {
		t.Fatalf("fulfillment = %+v", fulfillment)
	}
}

func TestServiceCoverageClassifiesBuiltInflightGapAndUnmapped(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewServiceWithScenarioSource(NewSQLiteRepository(database), fakeScenarioSource{graph: &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{
			{Scenario: "proto-health", Kind: graphv1.NodeKind_NODE_KIND_LIVE},
			{Scenario: "planner", Kind: graphv1.NodeKind_NODE_KIND_PLANNED},
			{Scenario: "unmapped-live", Kind: graphv1.NodeKind_NODE_KIND_LIVE},
		},
	}})

	mustCapability(t, service, ctx, Capability{Slug: "engineering", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR})
	mustCapability(t, service, ctx, Capability{Slug: "proto-foundation", ParentID: "engineering"})
	mustCapability(t, service, ctx, Capability{Slug: "planning-foundation", ParentID: "engineering"})
	mustCapability(t, service, ctx, Capability{Slug: "empty-gap", ParentID: "engineering"})
	mustFulfillment(t, service, ctx, Fulfillment{CapabilityID: "proto-foundation", ScenarioSlug: "proto-health"})
	mustFulfillment(t, service, ctx, Fulfillment{CapabilityID: "planning-foundation", ScenarioSlug: "planner"})

	coverage, err := service.GetCoverage(ctx, CoverageRequest{IncludeSubtreeRollup: true})
	if err != nil {
		t.Fatalf("GetCoverage() error = %v", err)
	}
	if coverage.BuiltCapabilities != 2 || coverage.InflightCapabilities != 1 || coverage.GapCapabilities != 1 {
		t.Fatalf("coverage counts = built %d inflight %d gap %d, want 2/1/1", coverage.BuiltCapabilities, coverage.InflightCapabilities, coverage.GapCapabilities)
	}
	if coverage.UnmappedScenarios != 1 || coverage.TotalScenarios != 3 {
		t.Fatalf("scenario counts = unmapped %d total %d, want 1/3", coverage.UnmappedScenarios, coverage.TotalScenarios)
	}
	if coverage.OntologyCompleteness != 0.75 {
		t.Fatalf("ontology completeness = %v, want 0.75", coverage.OntologyCompleteness)
	}
	if coverage.ImplementationSituatedness != float64(2)/float64(3) {
		t.Fatalf("implementation situatedness = %v, want 2/3", coverage.ImplementationSituatedness)
	}
	engineering := findClassification(t, coverage.Classifications, "engineering")
	if engineering.State != ontologyv1.CoverageState_COVERAGE_STATE_BUILT || engineering.DirectlyFulfilled || !engineering.SubtreeCovered {
		t.Fatalf("engineering classification = %+v", engineering)
	}
}

func TestServicePointQueriesUseSubtreeWhenRequested(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewServiceWithScenarioSource(NewSQLiteRepository(database), fakeScenarioSource{graph: &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{
			{Scenario: "proto-health", Kind: graphv1.NodeKind_NODE_KIND_LIVE},
			{Scenario: "planner", Kind: graphv1.NodeKind_NODE_KIND_PLANNED},
		},
	}})
	mustCapability(t, service, ctx, Capability{Slug: "engineering", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR})
	mustCapability(t, service, ctx, Capability{Slug: "proto-foundation", ParentID: "engineering"})
	mustCapability(t, service, ctx, Capability{Slug: "planning-foundation", ParentID: "engineering"})
	mustFulfillment(t, service, ctx, Fulfillment{CapabilityID: "proto-foundation", ScenarioSlug: "proto-health"})
	mustFulfillment(t, service, ctx, Fulfillment{CapabilityID: "planning-foundation", ScenarioSlug: "planner"})

	scenarios, err := service.GetCapabilityScenarios(ctx, CapabilityRef{Slug: "engineering"}, true)
	if err != nil {
		t.Fatalf("GetCapabilityScenarios() error = %v", err)
	}
	if len(scenarios.BuiltScenarios) != 1 || scenarios.BuiltScenarios[0] != "proto-health" {
		t.Fatalf("built scenarios = %v, want [proto-health]", scenarios.BuiltScenarios)
	}
	if len(scenarios.PlannedScenarios) != 1 || scenarios.PlannedScenarios[0] != "planner" {
		t.Fatalf("planned scenarios = %v, want [planner]", scenarios.PlannedScenarios)
	}
	capabilities, err := service.GetScenarioCapabilities(ctx, "PROTO-HEALTH")
	if err != nil {
		t.Fatalf("GetScenarioCapabilities() error = %v", err)
	}
	if len(capabilities.Capabilities) != 1 || capabilities.Capabilities[0].Slug != "proto-foundation" {
		t.Fatalf("scenario capabilities = %+v, want proto-foundation", capabilities.Capabilities)
	}
}

func TestServiceFocusRanksDownstreamGapsBeforeUnmappedScenarios(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewServiceWithScenarioSource(NewSQLiteRepository(database), fakeScenarioSource{graph: &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{{Scenario: "unmapped-live", Kind: graphv1.NodeKind_NODE_KIND_LIVE}},
	}})
	mustCapability(t, service, ctx, Capability{Slug: "root", Importance: 1})
	mustCapability(t, service, ctx, Capability{Slug: "unlock-one", Importance: 1})
	mustCapability(t, service, ctx, Capability{Slug: "unlock-two", Importance: 1})
	if _, err := service.UpsertCapabilityEdge(ctx, CapabilityEdge{FromID: "root", ToID: "unlock-one", Type: ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION}); err != nil {
		t.Fatalf("UpsertCapabilityEdge(root->unlock-one) error = %v", err)
	}
	if _, err := service.UpsertCapabilityEdge(ctx, CapabilityEdge{FromID: "unlock-one", ToID: "unlock-two", Type: ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION}); err != nil {
		t.Fatalf("UpsertCapabilityEdge(unlock-one->unlock-two) error = %v", err)
	}

	items, err := service.ListFocus(ctx, 2)
	if err != nil {
		t.Fatalf("ListFocus() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].CapabilitySlug != "root" || items[0].DownstreamDependents != 2 {
		t.Fatalf("top focus = %+v, want root with two downstream dependents", items[0])
	}
}

func TestServiceDescribeOverlayGraphProjectsOntologyAndFulfillment(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewServiceWithScenarioSource(NewSQLiteRepository(database), fakeScenarioSource{graph: &graphv1.TechTreeGraph{
		Nodes: []*graphv1.TechNode{{Scenario: "proto-health", Kind: graphv1.NodeKind_NODE_KIND_LIVE}},
	}})
	mustCapability(t, service, ctx, Capability{Slug: "engineering", Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR})
	mustCapability(t, service, ctx, Capability{Slug: "proto-foundation", ParentID: "engineering"})
	mustFulfillment(t, service, ctx, Fulfillment{CapabilityID: "proto-foundation", ScenarioSlug: "proto-health"})

	graph, err := service.DescribeOverlayGraph(ctx, OverlayGraphRequest{})
	if err != nil {
		t.Fatalf("DescribeOverlayGraph() error = %v", err)
	}
	if findNode(graph.GetNodes(), "proto-foundation").GetKind() != graphv1.NodeKind_NODE_KIND_CAPABILITY {
		t.Fatalf("proto-foundation node missing or wrong kind")
	}
	if findNode(graph.GetNodes(), "proto-foundation").GetParent() != "engineering" {
		t.Fatalf("proto-foundation parent = %q, want engineering", findNode(graph.GetNodes(), "proto-foundation").GetParent())
	}
	if !hasEdgeSource(graph.GetEdges(), "engineering", "proto-foundation", graphv1.EvidenceSource_EVIDENCE_SOURCE_DECOMPOSES) {
		t.Fatalf("missing decomposes edge engineering -> proto-foundation")
	}
	if !hasEdgeSource(graph.GetEdges(), "proto-foundation", "proto-health", graphv1.EvidenceSource_EVIDENCE_SOURCE_FULFILLS) {
		t.Fatalf("missing fulfills edge proto-foundation -> proto-health")
	}
}

func mustCapability(t *testing.T, service *Service, ctx context.Context, capability Capability) {
	t.Helper()
	if _, err := service.UpsertCapability(ctx, capability); err != nil {
		t.Fatalf("UpsertCapability(%s) error = %v", capability.Slug, err)
	}
}

func mustFulfillment(t *testing.T, service *Service, ctx context.Context, fulfillment Fulfillment) {
	t.Helper()
	if _, err := service.LinkFulfillment(ctx, fulfillment); err != nil {
		t.Fatalf("LinkFulfillment(%s, %s) error = %v", fulfillment.CapabilityID, fulfillment.ScenarioSlug, err)
	}
}

func findClassification(t *testing.T, classifications []CoverageClassification, slug string) CoverageClassification {
	t.Helper()
	for _, classification := range classifications {
		if classification.CapabilitySlug == slug {
			return classification
		}
	}
	t.Fatalf("classification for %q not found", slug)
	return CoverageClassification{}
}

func findNode(nodes []*graphv1.TechNode, scenario string) *graphv1.TechNode {
	for _, node := range nodes {
		if node.GetScenario() == scenario {
			return node
		}
	}
	return &graphv1.TechNode{}
}

func hasEdgeSource(edges []*graphv1.TechEdge, from, to string, source graphv1.EvidenceSource) bool {
	for _, edge := range edges {
		if edge.GetFromScenario() != from || edge.GetToScenario() != to {
			continue
		}
		for _, evidence := range edge.GetEvidence() {
			if evidence.GetSource() == source {
				return true
			}
		}
	}
	return false
}
