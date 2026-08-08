package memberflow

type OperatingGraphMode string

const (
	OperatingGraphModeExplanatory OperatingGraphMode = "explanatory"
	OperatingGraphModeCheckable   OperatingGraphMode = "checkable"
	OperatingGraphModeContract    OperatingGraphMode = "contract"
)

type OperatingGraphNodeKind string

const (
	OperatingGraphNodeKindMember   OperatingGraphNodeKind = "member"
	OperatingGraphNodeKindTopic    OperatingGraphNodeKind = "topic"
	OperatingGraphNodeKindTeam     OperatingGraphNodeKind = "team"
	OperatingGraphNodeKindPOR      OperatingGraphNodeKind = "por"
	OperatingGraphNodeKindExternal OperatingGraphNodeKind = "external"
	OperatingGraphNodeKindProcess  OperatingGraphNodeKind = "process"
	OperatingGraphNodeKindFuture   OperatingGraphNodeKind = "future"
)

type OperatingGraphQualifier string

const (
	OperatingGraphQualifierFuture   OperatingGraphQualifier = "future"
	OperatingGraphQualifierOld      OperatingGraphQualifier = "old"
	OperatingGraphQualifierExternal OperatingGraphQualifier = "external"
)

type OperatingActorKind string

const (
	OperatingActorKindMember   OperatingActorKind = "member"
	OperatingActorKindTeam     OperatingActorKind = "team"
	OperatingActorKindExternal OperatingActorKind = "external"
	OperatingActorKindProcess  OperatingActorKind = "process"
	OperatingActorKindGroup    OperatingActorKind = "group"
	OperatingActorKindUnknown  OperatingActorKind = "unknown"
)

type OperatingTopicCatalogStatus string

const (
	OperatingTopicStatusLive              OperatingTopicCatalogStatus = "live"
	OperatingTopicStatusLiveTransitional  OperatingTopicCatalogStatus = "live_transitional"
	OperatingTopicStatusLiveSystem        OperatingTopicCatalogStatus = "live_system"
	OperatingTopicStatusLiveUnderConsumed OperatingTopicCatalogStatus = "live_under_consumed"
	OperatingTopicStatusTarget            OperatingTopicCatalogStatus = "target"
	OperatingTopicStatusOld               OperatingTopicCatalogStatus = "old"
	OperatingTopicStatusExternal          OperatingTopicCatalogStatus = "external"
	OperatingTopicStatusUnknown           OperatingTopicCatalogStatus = "unknown"
)

type OperatingCoverageStatus string

const (
	OperatingCoverageStatusEnforced       OperatingCoverageStatus = "enforced"
	OperatingCoverageStatusReferenceOnly  OperatingCoverageStatus = "reference_only"
	OperatingCoverageStatusNotImplemented OperatingCoverageStatus = "not_implemented"
	OperatingCoverageStatusMissing        OperatingCoverageStatus = "missing"
	OperatingCoverageStatusMismatch       OperatingCoverageStatus = "mismatch"
	OperatingCoverageStatusUnavailable    OperatingCoverageStatus = "unavailable"
)

type OperatingGraphMetadata struct {
	ID     string             `json:"id"`
	Scope  string             `json:"scope"`
	Team   string             `json:"team"`
	Mode   OperatingGraphMode `json:"mode"`
	Status string             `json:"status,omitempty"`
	Extra  map[string]string  `json:"extra,omitempty"`
}

type OperatingGraphSource struct {
	Path      string `json:"path"`
	Line      int    `json:"line"`
	FenceLine int    `json:"fence_line"`
}

type OperatingGraphBlock struct {
	Metadata OperatingGraphMetadata `json:"metadata"`
	Graph    OperatingGraph         `json:"graph"`
	Docs     OperatingGraphDocs     `json:"docs,omitempty"`
	Source   OperatingGraphSource   `json:"source"`
}

type OperatingGraph struct {
	ID        string                `json:"id"`
	Direction string                `json:"direction"`
	Nodes     []OperatingGraphNode  `json:"nodes"`
	Edges     []OperatingGraphEdge  `json:"edges"`
	Groups    []OperatingGraphGroup `json:"groups,omitempty"`
}

type OperatingGraphNode struct {
	ID         string                  `json:"id"`
	Kind       OperatingGraphNodeKind  `json:"kind"`
	Value      string                  `json:"value"`
	Qualifier  OperatingGraphQualifier `json:"qualifier,omitempty"`
	Display    string                  `json:"display,omitempty"`
	RawLabel   string                  `json:"raw_label"`
	Shape      OperatingGraphNodeShape `json:"shape,omitempty"`
	SourceLine int                     `json:"source_line"`
	Implicit   bool                    `json:"implicit,omitempty"`
}

type OperatingGraphNodeShape string

const (
	OperatingGraphNodeShapeRectangle  OperatingGraphNodeShape = "rectangle"
	OperatingGraphNodeShapeCylinder   OperatingGraphNodeShape = "cylinder"
	OperatingGraphNodeShapeDiamond    OperatingGraphNodeShape = "diamond"
	OperatingGraphNodeShapeStadium    OperatingGraphNodeShape = "stadium"
	OperatingGraphNodeShapeSubroutine OperatingGraphNodeShape = "subroutine"
	OperatingGraphNodeShapeDocument   OperatingGraphNodeShape = "document"
)

type OperatingGraphGroup struct {
	ID         string   `json:"id"`
	Display    string   `json:"display"`
	SourceLine int      `json:"source_line"`
	NodeIDs    []string `json:"node_ids,omitempty"`
}

type OperatingGraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Label      string `json:"label,omitempty"`
	SourceLine int    `json:"source_line"`
}

type OperatingGraphDocs struct {
	TopicCatalog OperatingTopicCatalogTable `json:"topic_catalog,omitempty"`
}

type OperatingTopicCatalogTable struct {
	HeaderLine int                        `json:"header_line,omitempty"`
	Rows       []OperatingTopicCatalogRow `json:"rows,omitempty"`
	Present    bool                       `json:"present,omitempty"`
}

type OperatingTopicCatalogRow struct {
	Topic      string                      `json:"topic"`
	Qualifier  string                      `json:"qualifier,omitempty"`
	Status     string                      `json:"status"`
	StatusKind OperatingTopicCatalogStatus `json:"status_kind,omitempty"`
	Writers    []OperatingActorReference   `json:"writers,omitempty"`
	Readers    []OperatingActorReference   `json:"readers,omitempty"`
	Purpose    string                      `json:"purpose"`
	SourceLine int                         `json:"source_line"`
	RawTopic   string                      `json:"raw_topic"`
}

type OperatingActorReference struct {
	Kind  OperatingActorKind `json:"kind"`
	Value string             `json:"value"`
	Raw   string             `json:"raw"`
}

type OperatingGraphValidationResult struct {
	Findings []OperatingGraphFinding `json:"findings"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
}

type OperatingGraphContractDiff struct {
	Kind             string   `json:"kind"`
	Relationship     string   `json:"relationship"`
	Team             string   `json:"team"`
	Member           string   `json:"member,omitempty"`
	Topic            string   `json:"topic,omitempty"`
	Path             string   `json:"path,omitempty"`
	External         string   `json:"external,omitempty"`
	ProducerTeam     string   `json:"producer_team,omitempty"`
	TargetTeam       string   `json:"target_team,omitempty"`
	SourcePath       string   `json:"source_path,omitempty"`
	Line             int      `json:"line,omitempty"`
	RuntimePath      string   `json:"runtime_path,omitempty"`
	AcceptableFields []string `json:"acceptable_fields,omitempty"`
	Suggestions      []string `json:"suggestions,omitempty"`
	Detail           string   `json:"detail"`
}
