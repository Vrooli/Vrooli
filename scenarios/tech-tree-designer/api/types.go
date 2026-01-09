package main

import (
	"encoding/json"
	"time"
)

type TechTree struct {
	ID          string    `json:"id" db:"id"`
	Slug        string    `json:"slug" db:"slug"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Version     string    `json:"version" db:"version"`
	TreeType    string    `json:"tree_type" db:"tree_type"`
	Status      string    `json:"status" db:"status"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	ParentTree  *string   `json:"parent_tree_id,omitempty" db:"parent_tree_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type TechTreeSummary struct {
	Tree             TechTree `json:"tree"`
	SectorCount      int      `json:"sector_count"`
	StageCount       int      `json:"stage_count"`
	ScenarioMappings int      `json:"scenario_mapping_count"`
}

type Sector struct {
	ID                 string             `json:"id" db:"id"`
	TreeID             string             `json:"tree_id" db:"tree_id"`
	Name               string             `json:"name" db:"name"`
	Category           string             `json:"category" db:"category"`
	Description        string             `json:"description" db:"description"`
	ProgressPercentage float64            `json:"progress_percentage" db:"progress_percentage"`
	PositionX          float64            `json:"position_x" db:"position_x"`
	PositionY          float64            `json:"position_y" db:"position_y"`
	Color              string             `json:"color" db:"color"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
	Stages             []ProgressionStage `json:"stages,omitempty"`
}

type ProgressionStage struct {
	ID                 string            `json:"id" db:"id"`
	SectorID           string            `json:"sector_id" db:"sector_id"`
	ParentStageID      *string           `json:"parent_stage_id,omitempty" db:"parent_stage_id"`
	StageType          string            `json:"stage_type" db:"stage_type"`
	StageOrder         int               `json:"stage_order" db:"stage_order"`
	Name               string            `json:"name" db:"name"`
	Description        string            `json:"description" db:"description"`
	ProgressPercentage float64           `json:"progress_percentage" db:"progress_percentage"`
	Maturity           string            `json:"maturity" db:"maturity"` // planned, building, live, scaled
	Examples           json.RawMessage   `json:"examples" db:"examples"`
	PositionX          float64           `json:"position_x" db:"position_x"`
	PositionY          float64           `json:"position_y" db:"position_y"`
	HasChildren        bool              `json:"has_children" db:"has_children"`
	ChildrenLoaded     bool              `json:"children_loaded" db:"children_loaded"`
	CreatedAt          time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at" db:"updated_at"`
	ScenarioMappings   []ScenarioMapping `json:"scenario_mappings,omitempty"`
	Dependencies       []StageDependency `json:"dependencies,omitempty"`
	Children           []ProgressionStage `json:"children,omitempty"` // Lazy loaded
}

type ScenarioMapping struct {
	ID                 string    `json:"id" db:"id"`
	ScenarioName       string    `json:"scenario_name" db:"scenario_name"`
	StageID            string    `json:"stage_id" db:"stage_id"`
	ContributionWeight float64   `json:"contribution_weight" db:"contribution_weight"`
	CompletionStatus   string    `json:"completion_status" db:"completion_status"`
	Priority           int       `json:"priority" db:"priority"`
	EstimatedImpact    float64   `json:"estimated_impact" db:"estimated_impact"`
	LastStatusCheck    time.Time `json:"last_status_check" db:"last_status_check"`
	Notes              string    `json:"notes" db:"notes"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type StageDependency struct {
	ID                  string    `json:"id" db:"id"`
	DependentStageID    string    `json:"dependent_stage_id" db:"dependent_stage_id"`
	PrerequisiteStageID string    `json:"prerequisite_stage_id" db:"prerequisite_stage_id"`
	DependencyType      string    `json:"dependency_type" db:"dependency_type"`
	DependencyStrength  float64   `json:"dependency_strength" db:"dependency_strength"`
	Description         string    `json:"description" db:"description"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

type StagePositionUpdate struct {
	ID        string  `json:"id"`
	PositionX float64 `json:"position_x"`
	PositionY float64 `json:"position_y"`
}

type GraphDependencyInput struct {
	ID                  string  `json:"id"`
	DependentStageID    string  `json:"dependent_stage_id"`
	PrerequisiteStageID string  `json:"prerequisite_stage_id"`
	DependencyType      string  `json:"dependency_type"`
	DependencyStrength  float64 `json:"dependency_strength"`
	Description         string  `json:"description"`
}

type GraphUpdateRequest struct {
	StagePositions []StagePositionUpdate  `json:"stages"`
	Dependencies   []GraphDependencyInput `json:"dependencies"`
}

type DependencyPayload struct {
	Dependency       StageDependency `json:"dependency"`
	DependentName    string          `json:"dependent_name"`
	PrerequisiteName string          `json:"prerequisite_name"`
}

type StrategicMilestone struct {
	ID                      string          `json:"id" db:"id"`
	TreeID                  string          `json:"tree_id" db:"tree_id"`
	Name                    string          `json:"name" db:"name"`
	Description             string          `json:"description" db:"description"`
	MilestoneType           string          `json:"milestone_type" db:"milestone_type"`
	RequiredSectors         json.RawMessage `json:"required_sectors" db:"required_sectors"`
	RequiredStages          json.RawMessage `json:"required_stages" db:"required_stages"`
	CompletionPercentage    float64         `json:"completion_percentage" db:"completion_percentage"`
	EstimatedCompletionDate *time.Time      `json:"estimated_completion_date" db:"estimated_completion_date"`
	ConfidenceLevel         float64         `json:"confidence_level" db:"confidence_level"`
	BusinessValueEstimate   int64           `json:"business_value_estimate" db:"business_value_estimate"`
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
}

type StrategicRecommendation struct {
	Scenario         string  `json:"scenario"`
	PriorityScore    float64 `json:"priority_score"`
	ImpactMultiplier float64 `json:"impact_multiplier"`
	Reasoning        string  `json:"reasoning"`
}

type AnalysisRequest struct {
	CurrentResources int      `json:"current_resources"`
	TimeHorizon      int      `json:"time_horizon"`
	PrioritySectors  []string `json:"priority_sectors"`
}

type AnalysisResponse struct {
	Recommendations    []StrategicRecommendation `json:"recommendations"`
	ProjectedTimeline  ProjectedTimeline         `json:"projected_timeline"`
	BottleneckAnalysis []string                  `json:"bottleneck_analysis"`
	CrossSectorImpacts []CrossSectorImpact       `json:"cross_sector_impacts"`
}

type ProjectedTimeline struct {
	Milestones []MilestoneProjection `json:"milestones"`
}

type MilestoneProjection struct {
	Name                string    `json:"name"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	Confidence          float64   `json:"confidence"`
}

type CrossSectorImpact struct {
	SourceSector string  `json:"source_sector"`
	TargetSector string  `json:"target_sector"`
	ImpactScore  float64 `json:"impact_score"`
	Description  string  `json:"description"`
}

type StageIdea struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	StageType          string   `json:"stage_type"`
	SuggestedScenarios []string `json:"suggested_scenarios"`
	Confidence         float64  `json:"confidence"`
	StrategicRationale string   `json:"strategic_rationale"`
}

type ideaTemplate struct {
	Name        string
	StageType   string
	Description string
	Scenarios   []string
}
