package main

import "time"

type RequirementValidation struct {
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	Phase  string `json:"phase"`
	Status string `json:"status"`
	Notes  string `json:"notes"`
}

type RequirementRecord struct {
	ID            string                  `json:"id"`
	Category      string                  `json:"category"`
	PRDRef        string                  `json:"prd_ref"`
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	Status        string                  `json:"status"`
	Criticality   string                  `json:"criticality"`
	FilePath      string                  `json:"file_path"`
	Validations   []RequirementValidation `json:"validation"`
	LinkedTargets []string                `json:"linked_operational_target_ids"`
	TestFiles     []TestFileReference     `json:"test_files,omitempty"`    // Test files that reference this requirement
	PRDRefIssue   *PRDValidationIssue     `json:"prd_ref_issue,omitempty"` // Issue with prd_ref if any
}

type RequirementGroup struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	FilePath     string              `json:"file_path"`
	IsModule     bool                `json:"is_module"` // True if this represents a module.json file
	Requirements []RequirementRecord `json:"requirements"`
	Children     []RequirementGroup  `json:"children"`
}

type RequirementsResponse struct {
	EntityType string             `json:"entity_type"`
	EntityName string             `json:"entity_name"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Groups     []RequirementGroup `json:"groups"`
}

type OperationalTarget struct {
	ID                 string   `json:"id"`
	EntityType         string   `json:"entity_type"`
	EntityName         string   `json:"entity_name"`
	Category           string   `json:"category"`
	Criticality        string   `json:"criticality"`
	Title              string   `json:"title"`
	Notes              string   `json:"notes"`
	Status             string   `json:"status"`
	Path               string   `json:"path"`
	LinkedRequirements []string `json:"linked_requirement_ids"`
}

type OperationalTargetsResponse struct {
	EntityType            string              `json:"entity_type"`
	EntityName            string              `json:"entity_name"`
	Targets               []OperationalTarget `json:"targets"`
	UnmatchedRequirements []RequirementRecord `json:"unmatched_requirements"`
}

type requirementsFile struct {
	Metadata     map[string]any           `json:"_metadata"`
	ModuleID     string                   `json:"module_id"`     // For module.json files
	Title        string                   `json:"title"`         // For module.json files
	Priority     string                   `json:"priority"`      // For module.json files
	Description  string                   `json:"description"`   // For module.json files
	Imports      []string                 `json:"imports"`
	Requirements []RequirementRecordInput `json:"requirements"`
}

type RequirementRecordInput struct {
	ID          string                  `json:"id"`
	Category    string                  `json:"category"`
	PRDRef      string                  `json:"prd_ref"`
	Title       string                  `json:"title"`
	Description string                  `json:"description"`
	Status      string                  `json:"status"`
	Criticality string                  `json:"criticality"`
	Validations []RequirementValidation `json:"validation"`
}
