package ontology

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,127}$`)

type Capability struct {
	ID          string
	Slug        string
	Name        string
	Description string
	Kind        ontologyv1.CapabilityKind
	ParentID    string
	SortOrder   int32
	Importance  float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CapabilityEdge struct {
	FromID string
	ToID   string
	Type   ontologyv1.CapabilityEdgeType
}

type Fulfillment struct {
	CapabilityID string
	ScenarioSlug string
	Note         string
	CreatedAt    time.Time
}

type CapabilityRef struct {
	ID   string
	Slug string
}

type CapabilityFilter struct {
	ParentID           string
	Kind               ontologyv1.CapabilityKind
	IncludeDescendants bool
}

type FulfillmentFilter struct {
	CapabilityID string
	ScenarioSlug string
}

type TopologyImport struct {
	Capabilities []Capability
	Edges        []CapabilityEdge
}

type TopologyImportResult struct {
	SectorsImported      int32
	CapabilitiesImported int32
	EdgesImported        int32
	SectorsTotal         int32
	CapabilitiesTotal    int32
	EdgesTotal           int32
}

// seam: ScenarioSource reads the bottom-up implementation graph for cross-layer
// coverage and overlay projection. Production wires an adapter over graph.Service;
// tests wire a fake source.
type ScenarioSource interface {
	ScenarioGraph(ctx context.Context) (*graphv1.TechTreeGraph, error)
}

type CoverageRequest struct {
	IncludeSubtreeRollup bool
}

type CoverageSummary struct {
	BuiltCapabilities          int32
	InflightCapabilities       int32
	GapCapabilities            int32
	UnmappedScenarios          int32
	TotalCapabilities          int32
	TotalScenarios             int32
	OntologyCompleteness       float64
	ImplementationSituatedness float64
	Sectors                    []SectorCoverage
	Classifications            []CoverageClassification
	GraphError                 string
}

type SectorCoverage struct {
	SectorID             string
	SectorSlug           string
	SectorName           string
	BuiltCapabilities    int32
	InflightCapabilities int32
	GapCapabilities      int32
	TotalCapabilities    int32
	OntologyCompleteness float64
}

type CoverageClassification struct {
	CapabilityID      string
	CapabilitySlug    string
	State             ontologyv1.CoverageState
	DirectlyFulfilled bool
	SubtreeCovered    bool
	BuiltScenarios    []string
	PlannedScenarios  []string
}

type FocusItem struct {
	CapabilityID         string
	CapabilitySlug       string
	CapabilityName       string
	Reason               ontologyv1.FocusReason
	Score                float64
	DownstreamDependents int32
	RelatedScenarios     []string
}

type CapabilityScenarios struct {
	CapabilityID     string
	CapabilitySlug   string
	BuiltScenarios   []string
	PlannedScenarios []string
	Fulfillments     []Fulfillment
}

type ScenarioCapabilities struct {
	ScenarioSlug string
	Capabilities []Capability
	Fulfillments []Fulfillment
}

type OverlayGraphRequest struct {
	IncludeImplementation bool
	IncludeOntology       bool
	IncludeFulfillment    bool
}

type ErrInvalidArgument struct {
	Field  string
	Reason string
}

func (e ErrInvalidArgument) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}

type ErrCapabilityNotFound struct {
	Ref CapabilityRef
}

func (e ErrCapabilityNotFound) Error() string {
	if e.Ref.ID != "" {
		return fmt.Sprintf("capability %q not found", e.Ref.ID)
	}
	return fmt.Sprintf("capability %q not found", e.Ref.Slug)
}

type ErrCapabilityCycle struct {
	ID       string
	ParentID string
}

func (e ErrCapabilityCycle) Error() string {
	return fmt.Sprintf("capability %q cannot use descendant %q as parent", e.ID, e.ParentID)
}

func NormalizeID(field, value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", ErrInvalidArgument{Field: field, Reason: "is required"}
	}
	if !idPattern.MatchString(value) {
		return "", ErrInvalidArgument{Field: field, Reason: "must use lowercase letters, numbers, hyphens, and underscores"}
	}
	return value, nil
}

func NormalizeOptionalID(field, value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", nil
	}
	return NormalizeID(field, value)
}

func NormalizeScenarioSlug(value string) (string, error) {
	return NormalizeID("scenario_slug", value)
}

func DefaultName(slug string) string {
	parts := strings.Split(slug, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func NormalizeCapabilityKind(kind ontologyv1.CapabilityKind) (ontologyv1.CapabilityKind, error) {
	if kind == ontologyv1.CapabilityKind_CAPABILITY_KIND_UNSPECIFIED {
		return ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY, nil
	}
	switch kind {
	case ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR,
		ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPABILITY,
		ontologyv1.CapabilityKind_CAPABILITY_KIND_COMPONENT,
		ontologyv1.CapabilityKind_CAPABILITY_KIND_CAPSTONE,
		ontologyv1.CapabilityKind_CAPABILITY_KIND_SIMULATION:
		return kind, nil
	default:
		return 0, ErrInvalidArgument{Field: "kind", Reason: "is not supported"}
	}
}

func NormalizeCapabilityEdgeType(edgeType ontologyv1.CapabilityEdgeType) (ontologyv1.CapabilityEdgeType, error) {
	if edgeType == ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_UNSPECIFIED {
		return 0, ErrInvalidArgument{Field: "edge.type", Reason: "is required"}
	}
	switch edgeType {
	case ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES,
		ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_PROGRESSION,
		ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_REQUIRES:
		return edgeType, nil
	default:
		return 0, ErrInvalidArgument{Field: "edge.type", Reason: "is not supported"}
	}
}

func KindToStorage(kind ontologyv1.CapabilityKind) string {
	return strings.TrimPrefix(kind.String(), "CAPABILITY_KIND_")
}

func KindFromStorage(value string) ontologyv1.CapabilityKind {
	return ontologyv1.CapabilityKind(ontologyv1.CapabilityKind_value["CAPABILITY_KIND_"+value])
}

func EdgeTypeToStorage(edgeType ontologyv1.CapabilityEdgeType) string {
	return strings.TrimPrefix(edgeType.String(), "CAPABILITY_EDGE_TYPE_")
}

func EdgeTypeFromStorage(value string) ontologyv1.CapabilityEdgeType {
	return ontologyv1.CapabilityEdgeType(ontologyv1.CapabilityEdgeType_value["CAPABILITY_EDGE_TYPE_"+value])
}

func parseTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, value)
	return t
}
