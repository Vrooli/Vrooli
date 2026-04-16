package support

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

type TreeSelector struct {
	id   string
	slug string
}

func NewTreeSelector() *TreeSelector {
	return &TreeSelector{}
}

func (s *TreeSelector) Set(id, slug string) {
	s.id = strings.TrimSpace(id)
	s.slug = strings.TrimSpace(slug)
}

func (s *TreeSelector) Append(values url.Values) url.Values {
	if values == nil {
		values = url.Values{}
	}
	if s == nil {
		return values
	}
	if s.id != "" {
		values.Set("tree_id", s.id)
	} else if s.slug != "" {
		values.Set("tree_slug", s.slug)
	}
	return values
}

func (s *TreeSelector) ScopeLine() string {
	if s == nil {
		return "Tree scope: active default tree"
	}
	if s.id != "" {
		return "Tree scope: explicit tree_id=" + s.id
	}
	if s.slug != "" {
		return "Tree scope: explicit tree_slug=" + s.slug
	}
	return "Tree scope: active default tree"
}

type TechTree struct {
	ID          string    `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version"`
	TreeType    string    `json:"tree_type"`
	Status      string    `json:"status"`
	IsActive    bool      `json:"is_active"`
	ParentTree  *string   `json:"parent_tree_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TreeStats struct {
	Sectors          int `json:"sectors"`
	Stages           int `json:"stages"`
	ScenarioMappings int `json:"scenario_mappings"`
}

type TreeEnvelope struct {
	Tree  TechTree  `json:"tree"`
	Stats TreeStats `json:"stats"`
}

type TreeSummary struct {
	Tree             TechTree `json:"tree"`
	SectorCount      int      `json:"sector_count"`
	StageCount       int      `json:"stage_count"`
	ScenarioMappings int      `json:"scenario_mapping_count"`
}

type TreeListResponse struct {
	Trees []TreeSummary `json:"trees"`
}

type Sector struct {
	ID                 string  `json:"id"`
	TreeID             string  `json:"tree_id"`
	Name               string  `json:"name"`
	Category           string  `json:"category"`
	Description        string  `json:"description"`
	ProgressPercentage float64 `json:"progress_percentage"`
	Color              string  `json:"color"`
	Stages             []Stage `json:"stages,omitempty"`
}

type Stage struct {
	ID                 string          `json:"id"`
	SectorID           string          `json:"sector_id"`
	ParentStageID      *string         `json:"parent_stage_id,omitempty"`
	StageType          string          `json:"stage_type"`
	StageOrder         int             `json:"stage_order"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	ProgressPercentage float64         `json:"progress_percentage"`
	Maturity           string          `json:"maturity"`
	Examples           json.RawMessage `json:"examples"`
	HasChildren        bool            `json:"has_children"`
	ChildrenLoaded     bool            `json:"children_loaded"`
	ScenarioMappings   []ScenarioMap   `json:"scenario_mappings,omitempty"`
}

type SectorListResponse struct {
	Sectors []Sector `json:"sectors"`
	Tree    TechTree `json:"tree"`
}

type SectorResponse struct {
	Sector Sector   `json:"sector"`
	Tree   TechTree `json:"tree"`
}

type StageResponse struct {
	Stage Stage    `json:"stage"`
	Tree  TechTree `json:"tree"`
}

type StageChildrenResponse struct {
	Children []Stage `json:"children"`
	Count    int     `json:"count"`
}

type ScenarioMap struct {
	ID                 string    `json:"id"`
	ScenarioName       string    `json:"scenario_name"`
	StageID            string    `json:"stage_id"`
	ContributionWeight float64   `json:"contribution_weight"`
	CompletionStatus   string    `json:"completion_status"`
	Priority           int       `json:"priority"`
	EstimatedImpact    float64   `json:"estimated_impact"`
	LastStatusCheck    time.Time `json:"last_status_check"`
	Notes              string    `json:"notes"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ScenarioMappingEntry struct {
	Mapping    ScenarioMap `json:"mapping"`
	StageName  string      `json:"stage_name"`
	SectorName string      `json:"sector_name"`
}

type ScenarioMappingsResponse struct {
	ScenarioMappings []ScenarioMappingEntry `json:"scenario_mappings"`
	Tree             TechTree               `json:"tree"`
}

type ScenarioMappingMutationResponse struct {
	Message string      `json:"message"`
	Mapping ScenarioMap `json:"mapping"`
	Tree    TechTree    `json:"tree"`
}

type ScenarioStatusMutationResponse struct {
	Message  string   `json:"message"`
	Scenario string   `json:"scenario"`
	Status   string   `json:"status"`
	Tree     TechTree `json:"tree"`
	ID       string   `json:"id"`
}

type StrategicRecommendation struct {
	Scenario         string  `json:"scenario"`
	PriorityScore    float64 `json:"priority_score"`
	ImpactMultiplier float64 `json:"impact_multiplier"`
	Reasoning        string  `json:"reasoning"`
}

type MilestoneProjection struct {
	Name                string    `json:"name"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	Confidence          float64   `json:"confidence"`
}

type ProjectedTimeline struct {
	Milestones []MilestoneProjection `json:"milestones"`
}

type CrossSectorImpact struct {
	SourceSector string  `json:"source_sector"`
	TargetSector string  `json:"target_sector"`
	ImpactScore  float64 `json:"impact_score"`
	Description  string  `json:"description"`
}

type AnalysisResponse struct {
	Recommendations    []StrategicRecommendation `json:"recommendations"`
	ProjectedTimeline  ProjectedTimeline         `json:"projected_timeline"`
	BottleneckAnalysis []string                  `json:"bottleneck_analysis"`
	CrossSectorImpacts []CrossSectorImpact       `json:"cross_sector_impacts"`
	Tree               *TechTree                 `json:"tree,omitempty"`
}

type RecommendationsResponse struct {
	Recommendations []StrategicRecommendation `json:"recommendations"`
	Tree            *TechTree                 `json:"tree,omitempty"`
}

type StrategicMilestone struct {
	ID                      string          `json:"id"`
	TreeID                  string          `json:"tree_id"`
	Name                    string          `json:"name"`
	Description             string          `json:"description"`
	MilestoneType           string          `json:"milestone_type"`
	RequiredSectors         json.RawMessage `json:"required_sectors"`
	RequiredStages          json.RawMessage `json:"required_stages"`
	CompletionPercentage    float64         `json:"completion_percentage"`
	EstimatedCompletionDate *time.Time      `json:"estimated_completion_date"`
	ConfidenceLevel         float64         `json:"confidence_level"`
	BusinessValueEstimate   int64           `json:"business_value_estimate"`
	CreatedAt               time.Time       `json:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at"`
}

type MilestonesResponse struct {
	Milestones []StrategicMilestone `json:"milestones"`
	Tree       TechTree             `json:"tree"`
}

type MilestoneMutationResponse struct {
	Milestone StrategicMilestone `json:"milestone"`
}

type StageDependency struct {
	ID                  string  `json:"id"`
	DependentStageID    string  `json:"dependent_stage_id"`
	PrerequisiteStageID string  `json:"prerequisite_stage_id"`
	DependencyType      string  `json:"dependency_type"`
	DependencyStrength  float64 `json:"dependency_strength"`
	Description         string  `json:"description"`
}

type DependencyEntry struct {
	Dependency       StageDependency `json:"dependency"`
	DependentName    string          `json:"dependent_name"`
	PrerequisiteName string          `json:"prerequisite_name"`
}

type DependenciesResponse struct {
	Dependencies []DependencyEntry `json:"dependencies"`
	Tree         TechTree          `json:"tree"`
}

type SectorConnection struct {
	ID             string          `json:"id"`
	SourceSectorID string          `json:"source_sector_id"`
	TargetSectorID string          `json:"target_sector_id"`
	ConnectionType string          `json:"connection_type"`
	Strength       float64         `json:"strength"`
	Description    string          `json:"description"`
	Examples       json.RawMessage `json:"examples"`
}

type ConnectionEntry struct {
	Connection SectorConnection `json:"connection"`
	SourceName string           `json:"source_name"`
	TargetName string           `json:"target_name"`
}

type ConnectionsResponse struct {
	Connections []ConnectionEntry `json:"connections"`
	Tree        TechTree          `json:"tree"`
}

type ScenarioCatalogEntry struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

type ScenarioCatalogEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

type ScenarioCatalogSnapshot struct {
	Scenarios  []ScenarioCatalogEntry `json:"scenarios"`
	Edges      []ScenarioCatalogEdge  `json:"edges"`
	Hidden     []string               `json:"hidden"`
	LastSynced time.Time              `json:"last_synced"`
	Message    string                 `json:"message"`
}
