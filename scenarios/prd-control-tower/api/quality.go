package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type ScenarioQualityReport struct {
	EntityType                 string                       `json:"entity_type"`
	EntityName                 string                       `json:"entity_name"`
	HasPRD                     bool                         `json:"has_prd"`
	HasRequirements            bool                         `json:"has_requirements"`
	PRDPath                    string                       `json:"prd_path,omitempty"`
	RequirementsPath           string                       `json:"requirements_path,omitempty"`
	ValidatedAt                time.Time                    `json:"validated_at"`
	CacheUsed                  bool                         `json:"cache_used"`
	Status                     string                       `json:"status"`
	Message                    string                       `json:"message,omitempty"`
	Error                      string                       `json:"error,omitempty"`
	IssueCounts                QualityIssueCounts           `json:"issue_counts"`
	TemplateCompliance         *PRDTemplateValidationResult `json:"template_compliance,omitempty"`
	TemplateComplianceV2       *PRDValidationResultV2       `json:"template_compliance_v2,omitempty"`
	TargetLinkageIssues        []TargetLinkageIssue         `json:"target_linkage_issues,omitempty"`
	RequirementsWithoutTargets []RequirementSummary         `json:"requirements_without_targets,omitempty"`
	PRDRefIssues               []PRDValidationIssue         `json:"prd_ref_issues,omitempty"`
	RequirementCount           int                          `json:"requirement_count"`
	TargetCount                int                          `json:"target_count"`
}

type RequirementSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	PRDRef      string `json:"prd_ref"`
	Criticality string `json:"criticality"`
	Status      string `json:"status"`
	Category    string `json:"category"`
	FilePath    string `json:"file_path,omitempty"`
	Issue       string `json:"issue"`
}

type QualityIssueCounts struct {
	MissingPRD              int `json:"missing_prd"`
	MissingTemplateSections int `json:"missing_template_sections"`
	TargetCoverage          int `json:"target_coverage"`
	RequirementCoverage     int `json:"requirement_coverage"`
	PRDRef                  int `json:"prd_ref"`
	Total                   int `json:"total"`
	Blocking                int `json:"blocking"`
}

type QualityScanEntity struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type QualityScanRequest struct {
	Entities []QualityScanEntity `json:"entities"`
	UseCache bool                `json:"use_cache"`
}

type QualityScanResponse struct {
	Reports     []ScenarioQualityReport `json:"reports"`
	GeneratedAt time.Time               `json:"generated_at"`
	DurationMs  int64                   `json:"duration_ms"`
}

type QualitySummary struct {
	TotalEntities int                    `json:"total_entities"`
	Scanned       int                    `json:"scanned"`
	WithIssues    int                    `json:"with_issues"`
	MissingPRD    int                    `json:"missing_prd"`
	LastGenerated time.Time              `json:"last_generated"`
	DurationMs    int64                  `json:"duration_ms"`
	CacheUsed     bool                   `json:"cache_used"`
	Samples       []QualitySummaryEntity `json:"samples,omitempty"`
}

type QualitySummaryEntity struct {
	EntityType  string             `json:"entity_type"`
	EntityName  string             `json:"entity_name"`
	Status      string             `json:"status"`
	IssueCounts QualityIssueCounts `json:"issue_counts"`
}

// StandardsViolation describes a standards-compatible violation payload that other
// scenarios (like scenario-auditor) can consume directly.
type StandardsViolation struct {
	RuleID         string         `json:"rule_id"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	FilePath       string         `json:"file_path,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// QualityStandardsResponse is served on /quality/{type}/{name}/standards.
type QualityStandardsResponse struct {
	EntityType  string               `json:"entity_type"`
	EntityName  string               `json:"entity_name"`
	Status      string               `json:"status"`
	Message     string               `json:"message,omitempty"`
	GeneratedAt time.Time            `json:"generated_at"`
	Violations  []StandardsViolation `json:"violations"`
}

var (
	qualityCacheTTL = time.Minute
	qualityCache    sync.Map

	summaryCacheTTL     = 2 * time.Minute
	qualitySummaryMu    sync.RWMutex
	qualitySummaryEntry struct {
		summary  QualitySummary
		cachedAt time.Time
	}
)

func handleGetQualityReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	useCache := parseBoolQuery(r, "use_cache", true)
	report, err := buildQualityReport(entityType, entityName, useCache)
	if err != nil {
		slog.Warn("quality report generated with warnings", "entity", entityName, "error", err)
	}

	respondJSON(w, http.StatusOK, report)
}

func handleGetQualityStandards(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	useCache := parseBoolQuery(r, "use_cache", false)
	report, err := buildQualityReport(entityType, entityName, useCache)
	if err != nil {
		slog.Warn("quality standards report generated with warnings", "entity", entityName, "error", err)
	}

	response := QualityStandardsResponse{
		EntityType:  entityType,
		EntityName:  entityName,
		Status:      report.Status,
		Message:     firstNonEmpty(report.Message, report.Error),
		GeneratedAt: time.Now(),
		Violations:  buildStandardsViolationsFromReport(report),
	}

	respondJSON(w, http.StatusOK, response)
}

func handleQualityScan(w http.ResponseWriter, r *http.Request) {
	var req QualityScanRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		if !errors.Is(err, io.EOF) {
			respondInvalidJSON(w, err)
			return
		}
	}

	entities := dedupeEntities(req.Entities)
	if len(entities) == 0 {
		catalogEntries, err := listCatalogEntries()
		if err != nil {
			respondInternalError(w, "Failed to enumerate catalog for scan", err)
			return
		}
		for _, entry := range catalogEntries {
			if !entry.HasPRD {
				continue
			}
			entities = append(entities, QualityScanEntity{Type: entry.Type, Name: entry.Name})
		}
	}

	if len(entities) == 0 {
		respondJSON(w, http.StatusOK, QualityScanResponse{Reports: []ScenarioQualityReport{}, GeneratedAt: time.Now()})
		return
	}

	start := time.Now()
	reports := make([]ScenarioQualityReport, 0, len(entities))
	for _, entity := range entities {
		if !isValidEntityType(entity.Type) {
			slog.Warn("Skipping invalid entity type in scan", "entity", entity.Name, "type", entity.Type)
			continue
		}
		report, err := buildQualityReport(entity.Type, entity.Name, req.UseCache)
		if err != nil {
			slog.Warn("quality scan entry failed", "entity", entity.Name, "error", err)
		}
		reports = append(reports, report)
	}

	response := QualityScanResponse{
		Reports:     reports,
		GeneratedAt: time.Now(),
		DurationMs:  time.Since(start).Milliseconds(),
	}

	respondJSON(w, http.StatusOK, response)
}

func handleQualitySummary(w http.ResponseWriter, r *http.Request) {
	useCache := parseBoolQuery(r, "use_cache", true)
	summary, err := buildQualitySummary(useCache)
	if err != nil {
		respondInternalError(w, "Failed to build quality summary", err)
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

func buildQualityReport(entityType, entityName string, useCache bool) (ScenarioQualityReport, error) {
	key := fmt.Sprintf("%s:%s", entityType, entityName)
	if useCache {
		if entry, ok := qualityCache.Load(key); ok {
			cached := entry.(qualityCacheEntry)
			if time.Since(cached.cachedAt) < qualityCacheTTL {
				report := cached.report
				report.CacheUsed = true
				return report, cached.err
			}
		}
	} else {
		qualityCache.Delete(key)
	}

	report, err := computeQualityReport(entityType, entityName)
	qualityCache.Store(key, qualityCacheEntry{report: report, err: err, cachedAt: time.Now()})
	return report, err
}

type qualityCacheEntry struct {
	report   ScenarioQualityReport
	err      error
	cachedAt time.Time
}

func computeQualityReport(entityType, entityName string) (ScenarioQualityReport, error) {
	now := time.Now()
	report := ScenarioQualityReport{
		EntityType:  entityType,
		EntityName:  entityName,
		ValidatedAt: now,
		Status:      "healthy",
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		report.Status = "error"
		report.Error = err.Error()
		return report, err
	}

	var entityDir string
	if entityType == EntityTypeScenario {
		entityDir = "scenarios"
	} else {
		entityDir = "resources"
	}

	prdPath := filepath.Join(vrooliRoot, entityDir, entityName, "PRD.md")
	report.PRDPath = prdPath

	if _, err := os.Stat(prdPath); errors.Is(err, os.ErrNotExist) {
		report.HasPRD = false
		report.Status = "missing_prd"
		report.Message = "PRD.md not found"
		report.IssueCounts.MissingPRD = 1
		report.IssueCounts.Total = 1
		report.IssueCounts.Blocking = 1
		return report, nil
	} else if err != nil {
		report.Status = "error"
		report.Error = fmt.Sprintf("failed to access PRD: %v", err)
		return report, err
	}
	report.HasPRD = true

	content, err := os.ReadFile(prdPath)
	if err != nil {
		report.Status = "error"
		report.Error = fmt.Sprintf("failed to read PRD: %v", err)
		return report, err
	}

	tmpl := ValidatePRDTemplate(string(content))
	tmplV2 := ValidatePRDTemplateV2(string(content))
	report.TemplateCompliance = &tmpl
	report.TemplateComplianceV2 = &tmplV2

	missingSections := len(tmplV2.MissingSections)
	for _, subsections := range tmplV2.MissingSubsections {
		missingSections += len(subsections)
	}
	unexpectedSections := len(tmplV2.UnexpectedSections)
	structureIssues := missingSections + unexpectedSections

	requirementsPath := filepath.Join(vrooliRoot, entityDir, entityName, "requirements", "index.json")
	if _, err := os.Stat(requirementsPath); err == nil {
		report.HasRequirements = true
		report.RequirementsPath = requirementsPath
	}

	groups, err := loadRequirementsForEntity(entityType, entityName)
	if err != nil {
		report.Status = "error"
		report.Error = fmt.Sprintf("failed to load requirements: %v", err)
		return report, err
	}

	flattened := flattenRequirements(groups)
	report.RequirementCount = len(flattened)

	targets, err := extractOperationalTargets(entityType, entityName)
	if err != nil {
		report.Status = "error"
		report.Error = fmt.Sprintf("failed to parse operational targets: %v", err)
		return report, err
	}
	report.TargetCount = len(targets)

	linkedTargets, unmatched := linkTargetsAndRequirements(targets, groups)
	targetIssues := findTargetCoverageIssues(linkedTargets)

	prdIssues := validatePRDReferences(entityType, entityName, flattened)

	report.TargetLinkageIssues = targetIssues
	report.RequirementsWithoutTargets = summarizeRequirements(unmatched, "missing_target_linkage")
	report.PRDRefIssues = prdIssues

	report.IssueCounts.MissingTemplateSections = structureIssues
	report.IssueCounts.TargetCoverage = len(targetIssues)
	report.IssueCounts.RequirementCoverage = len(unmatched)
	report.IssueCounts.PRDRef = len(prdIssues)
	report.IssueCounts.Total = report.IssueCounts.MissingTemplateSections + report.IssueCounts.TargetCoverage + report.IssueCounts.RequirementCoverage + report.IssueCounts.PRDRef

	report.IssueCounts.Blocking = report.IssueCounts.MissingPRD
	if structureIssues > 0 {
		report.IssueCounts.Blocking += 1
	}
	report.IssueCounts.Blocking += countCriticalTargetIssues(targetIssues, "P0")

	if report.IssueCounts.MissingPRD > 0 {
		report.Status = "missing_prd"
	} else if report.IssueCounts.Blocking > 0 {
		report.Status = "blocked"
	} else if report.IssueCounts.Total > 0 {
		report.Status = "needs_attention"
	} else {
		report.Status = "healthy"
		report.Message = "No structural or linkage issues detected"
	}

	return report, nil
}

func summarizeRequirements(reqs []RequirementRecord, issue string) []RequirementSummary {
	if len(reqs) == 0 {
		return nil
	}
	summaries := make([]RequirementSummary, 0, len(reqs))
	for _, req := range reqs {
		summaries = append(summaries, RequirementSummary{
			ID:          req.ID,
			Title:       req.Title,
			PRDRef:      req.PRDRef,
			Criticality: req.Criticality,
			Status:      req.Status,
			Category:    req.Category,
			FilePath:    req.FilePath,
			Issue:       issue,
		})
	}
	return summaries
}

func findTargetCoverageIssues(targets []OperationalTarget) []TargetLinkageIssue {
	var issues []TargetLinkageIssue
	for _, target := range targets {
		if len(target.LinkedRequirements) > 0 {
			continue
		}
		if target.Criticality == "P0" || target.Criticality == "P1" {
			issues = append(issues, TargetLinkageIssue{
				Title:       target.Title,
				Criticality: target.Criticality,
				Message:     fmt.Sprintf("%s target '%s' must be linked to requirements", target.Criticality, target.Title),
			})
		}
	}
	return issues
}

func countCriticalTargetIssues(issues []TargetLinkageIssue, criticality string) int {
	count := 0
	for _, issue := range issues {
		if issue.Criticality == criticality {
			count++
		}
	}
	return count
}

func dedupeEntities(entities []QualityScanEntity) []QualityScanEntity {
	if len(entities) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(entities))
	deduped := make([]QualityScanEntity, 0, len(entities))
	for _, entity := range entities {
		key := fmt.Sprintf("%s:%s", entity.Type, entity.Name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, entity)
	}
	return deduped
}

func parseBoolQuery(r *http.Request, key string, defaultValue bool) bool {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultValue
	}
	switch raw {
	case "0", "false", "FALSE", "False":
		return false
	case "1", "true", "TRUE", "True":
		return true
	}
	return defaultValue
}

func buildQualitySummary(useCache bool) (QualitySummary, error) {
	qualitySummaryMu.RLock()
	if useCache && time.Since(qualitySummaryEntry.cachedAt) < summaryCacheTTL {
		cached := qualitySummaryEntry.summary
		cached.CacheUsed = true
		qualitySummaryMu.RUnlock()
		return cached, nil
	}
	qualitySummaryMu.RUnlock()

	entries, err := listCatalogEntries()
	if err != nil {
		return QualitySummary{}, err
	}

	summary := QualitySummary{
		TotalEntities: len(entries),
	}

	start := time.Now()
	var samples []QualitySummaryEntity

	for _, entry := range entries {
		if !entry.HasPRD {
			summary.MissingPRD++
			summary.WithIssues++
			if len(samples) < 5 {
				samples = append(samples, QualitySummaryEntity{
					EntityType: entry.Type,
					EntityName: entry.Name,
					Status:     "missing_prd",
					IssueCounts: QualityIssueCounts{
						MissingPRD: 1,
						Total:      1,
						Blocking:   1,
					},
				})
			}
			continue
		}

		summary.Scanned++
		report, err := buildQualityReport(entry.Type, entry.Name, true)
		if err != nil {
			slog.Warn("quality summary entry failed", "entity", entry.Name, "error", err)
			continue
		}

		if report.IssueCounts.Total > 0 {
			summary.WithIssues++
			if len(samples) < 5 {
				samples = append(samples, QualitySummaryEntity{
					EntityType:  report.EntityType,
					EntityName:  report.EntityName,
					Status:      report.Status,
					IssueCounts: report.IssueCounts,
				})
			}
		}
	}

	summary.Samples = samples
	summary.LastGenerated = time.Now()
	summary.DurationMs = time.Since(start).Milliseconds()

	qualitySummaryMu.Lock()
	qualitySummaryEntry.summary = summary
	qualitySummaryEntry.cachedAt = time.Now()
	qualitySummaryMu.Unlock()

	return summary, nil
}

func buildStandardsViolationsFromReport(report ScenarioQualityReport) []StandardsViolation {
	var violations []StandardsViolation
	appendViolation := func(v StandardsViolation) {
		violations = append(violations, v)
	}

	if !report.HasPRD {
		appendViolation(StandardsViolation{
			RuleID:         "prd_missing_prd",
			Severity:       "critical",
			Title:          "PRD.md missing",
			Description:    "Every scenario must include a PRD.md that follows the standard template.",
			FilePath:       report.PRDPath,
			Recommendation: "Add PRD.md using the canonical template before shipping the scenario.",
		})
		return violations
	}

	if !report.HasRequirements {
		appendViolation(StandardsViolation{
			RuleID:         "prd_missing_requirements",
			Severity:       "high",
			Title:          "requirements/index.json missing",
			Description:    "Operational targets must be backed by requirements definitions.",
			FilePath:       report.RequirementsPath,
			Recommendation: "Create requirements/index.json with P0/P1/P2 groupings and link each requirement to the PRD.",
		})
	}

	if tmpl := report.TemplateComplianceV2; tmpl != nil {
		for _, section := range tmpl.MissingSections {
			appendViolation(StandardsViolation{
				RuleID:         "prd_template_sections",
				Severity:       "high",
				Title:          fmt.Sprintf("Missing PRD section: %s", section),
				Description:    fmt.Sprintf("Section '%s' is required by the PRD template.", section),
				FilePath:       report.PRDPath,
				Recommendation: fmt.Sprintf("Add section '%s' to PRD.md", section),
				Metadata: map[string]any{
					"section": section,
					"issue":   "missing_section",
				},
			})
		}
		for parent, subsections := range tmpl.MissingSubsections {
			for _, subsection := range subsections {
				appendViolation(StandardsViolation{
					RuleID:         "prd_template_sections",
					Severity:       "high",
					Title:          fmt.Sprintf("Missing PRD subsection: %s", subsection),
					Description:    fmt.Sprintf("Subsection '%s' is required under '%s'.", subsection, parent),
					FilePath:       report.PRDPath,
					Recommendation: fmt.Sprintf("Add subsection '%s' beneath '%s' using checklist formatting.", subsection, parent),
					Metadata: map[string]any{
						"section":    parent,
						"subsection": subsection,
						"issue":      "missing_subsection",
					},
				})
			}
		}
		for _, unexpected := range tmpl.UnexpectedSections {
			appendViolation(StandardsViolation{
				RuleID:      "prd_template_unexpected_sections",
				Severity:    "low",
				Title:       fmt.Sprintf("Unexpected PRD section: %s", unexpected),
				Description: fmt.Sprintf("Section '%s' is not part of the official PRD template.", unexpected),
				FilePath:    report.PRDPath,
				Metadata: map[string]any{
					"section": unexpected,
				},
			})
		}
		for _, issue := range tmpl.ContentIssues {
			severity := strings.ToLower(strings.TrimSpace(issue.Severity))
			if severity == "" {
				severity = "medium"
			}
			appendViolation(StandardsViolation{
				RuleID:         "prd_template_content",
				Severity:       severity,
				Title:          fmt.Sprintf("Content issue in %s", issue.Section),
				Description:    issue.Message,
				FilePath:       report.PRDPath,
				Recommendation: issue.Suggestion,
				Metadata: map[string]any{
					"section": issue.Section,
					"issue":   issue.IssueType,
				},
			})
		}
	}

	for _, issue := range report.TargetLinkageIssues {
		severity := "high"
		if strings.EqualFold(issue.Criticality, "P0") {
			severity = "critical"
		}
		appendViolation(StandardsViolation{
			RuleID:         "prd_operational_target_linkage",
			Severity:       severity,
			Title:          fmt.Sprintf("%s target missing requirements", issue.Criticality),
			Description:    issue.Message,
			FilePath:       report.RequirementsPath,
			Recommendation: "Link each P0/P1 operational target to at least one requirement before publishing.",
			Metadata: map[string]any{
				"criticality": issue.Criticality,
			},
		})
	}

	for _, summary := range report.RequirementsWithoutTargets {
		appendViolation(StandardsViolation{
			RuleID:         "prd_requirements_without_targets",
			Severity:       "medium",
			Title:          fmt.Sprintf("Requirement %s missing operational target linkage", summary.ID),
			Description:    "Requirements must reference at least one operational target to document coverage.",
			FilePath:       summary.FilePath,
			Recommendation: "Update operational targets to reference this requirement or adjust criticality.",
			Metadata: map[string]any{
				"requirement_id": summary.ID,
				"criticality":    summary.Criticality,
			},
		})
	}

	for _, issue := range report.PRDRefIssues {
		severity := "medium"
		if issue.IssueType == "missing_section" {
			severity = "high"
		}
		appendViolation(StandardsViolation{
			RuleID:         "prd_prd_ref_integrity",
			Severity:       severity,
			Title:          fmt.Sprintf("Requirement %s references missing PRD content", issue.RequirementID),
			Description:    issue.Message,
			FilePath:       report.RequirementsPath,
			Recommendation: "Update prd_ref to point to an existing PRD section or checklist item.",
			Metadata: map[string]any{
				"requirement_id": issue.RequirementID,
				"issue_type":     issue.IssueType,
				"suggestions":    issue.Suggestions,
			},
		})
	}

	return violations
}
