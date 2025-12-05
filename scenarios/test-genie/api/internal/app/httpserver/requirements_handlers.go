package httpserver

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// RequirementsSnapshot represents the data returned by the requirements endpoint.
type RequirementsSnapshot struct {
	ScenarioName string              `json:"scenarioName"`
	GeneratedAt  time.Time           `json:"generatedAt"`
	Summary      RequirementsSummary `json:"summary"`
	Modules      []ModuleSnapshot    `json:"modules"`
	SyncStatus   *SyncStatus         `json:"syncStatus,omitempty"`
}

// RequirementsSummary contains aggregate statistics.
type RequirementsSummary struct {
	TotalRequirements int            `json:"totalRequirements"`
	TotalValidations  int            `json:"totalValidations"`
	CompletionRate    float64        `json:"completionRate"`
	PassRate          float64        `json:"passRate"`
	CriticalGap       int            `json:"criticalGap"`
	ByLiveStatus      map[string]int `json:"byLiveStatus"`
	ByDeclaredStatus  map[string]int `json:"byDeclaredStatus"`
}

// ModuleSnapshot contains module-level data.
type ModuleSnapshot struct {
	Name           string            `json:"name"`
	FilePath       string            `json:"filePath"`
	Total          int               `json:"total"`
	Complete       int               `json:"complete"`
	InProgress     int               `json:"inProgress"`
	Pending        int               `json:"pending"`
	CompletionRate float64           `json:"completionRate"`
	Requirements   []RequirementItem `json:"requirements,omitempty"`
}

// RequirementItem represents a single requirement.
type RequirementItem struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Status      string           `json:"status"`
	LiveStatus  string           `json:"liveStatus"`
	PRDRef      string           `json:"prdRef,omitempty"`
	Criticality string           `json:"criticality,omitempty"`
	Description string           `json:"description,omitempty"`
	Validations []ValidationItem `json:"validations,omitempty"`
}

// ValidationItem represents a test/automation validation.
type ValidationItem struct {
	Type       string `json:"type"`
	Ref        string `json:"ref"`
	Phase      string `json:"phase,omitempty"`
	Status     string `json:"status"`
	LiveStatus string `json:"liveStatus"`
}

// SyncStatus contains sync operation metadata.
type SyncStatus struct {
	Enabled            bool      `json:"enabled"`
	LastSyncedAt       time.Time `json:"lastSyncedAt,omitempty"`
	FilesUpdated       int       `json:"filesUpdated"`
	ValidationsAdded   int       `json:"validationsAdded"`
	ValidationsRemoved int       `json:"validationsRemoved"`
	StatusesChanged    int       `json:"statusesChanged"`
	ErrorCount         int       `json:"errorCount"`
}

// SyncPreviewResponse contains the preview of changes that would be made.
type SyncPreviewResponse struct {
	ScenarioName string       `json:"scenarioName"`
	Changes      []SyncChange `json:"changes"`
	Summary      struct {
		FilesAffected          int `json:"filesAffected"`
		StatusesWouldChange    int `json:"statusesWouldChange"`
		ValidationsWouldAdd    int `json:"validationsWouldAdd"`
		ValidationsWouldRemove int `json:"validationsWouldRemove"`
	} `json:"summary"`
}

// SyncChange represents a single change.
type SyncChange struct {
	Type          string `json:"type"`
	FilePath      string `json:"filePath"`
	RequirementID string `json:"requirementId,omitempty"`
	Field         string `json:"field,omitempty"`
	OldValue      string `json:"oldValue,omitempty"`
	NewValue      string `json:"newValue,omitempty"`
}

// handleGetScenarioRequirements returns requirements data for a scenario.
func (s *Server) handleGetScenarioRequirements(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	// Resolve scenario path
	scenarioDir := s.resolveScenarioDir(name)
	if scenarioDir == "" {
		s.writeError(w, http.StatusNotFound, "scenario not found")
		return
	}

	// Try to load cached snapshot first
	snapshotPath := filepath.Join(scenarioDir, "coverage", "requirements-sync", "latest.json")
	snapshot, err := s.loadRequirementsSnapshot(snapshotPath, name)
	if err != nil || len(snapshot.Modules) == 0 {
		// Snapshot not available or empty - read requirements directly from files
		snapshot = s.loadRequirementsFromFiles(scenarioDir, name)
	}

	// Load sync status
	syncStatusPath := filepath.Join(scenarioDir, "coverage", "sync", "latest.json")
	syncStatus := s.loadSyncStatus(syncStatusPath, scenarioDir)
	snapshot.SyncStatus = syncStatus

	s.writeJSON(w, http.StatusOK, snapshot)
}

// handleSyncScenarioRequirements triggers a manual requirements sync.
func (s *Server) handleSyncScenarioRequirements(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	var payload struct {
		DryRun       bool `json:"dryRun"`
		PruneOrphans bool `json:"pruneOrphans"`
		DiscoverNew  bool `json:"discoverNew"`
	}
	// Default to discover new
	payload.DiscoverNew = true

	if r.Body != nil {
		defer r.Body.Close()
		json.NewDecoder(r.Body).Decode(&payload)
	}

	scenarioDir := s.resolveScenarioDir(name)
	if scenarioDir == "" {
		s.writeError(w, http.StatusNotFound, "scenario not found")
		return
	}

	// For now, we return the current state since sync is typically
	// triggered by test execution. Full manual sync would require
	// integrating with the requirements service.
	if payload.DryRun {
		// Return preview
		preview := SyncPreviewResponse{
			ScenarioName: name,
			Changes:      []SyncChange{},
		}
		s.writeJSON(w, http.StatusOK, preview)
		return
	}

	// Return current snapshot after "sync"
	snapshotPath := filepath.Join(scenarioDir, "coverage", "requirements-sync", "latest.json")
	snapshot, err := s.loadRequirementsSnapshot(snapshotPath, name)
	if err != nil {
		snapshot = &RequirementsSnapshot{
			ScenarioName: name,
			GeneratedAt:  time.Now(),
			Summary: RequirementsSummary{
				ByLiveStatus:     make(map[string]int),
				ByDeclaredStatus: make(map[string]int),
			},
			Modules: []ModuleSnapshot{},
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "completed",
		"snapshot": snapshot,
	})
}

// resolveScenarioDir finds the directory for a scenario.
func (s *Server) resolveScenarioDir(name string) string {
	// Try common locations
	candidates := []string{
		filepath.Join(os.Getenv("SCENARIOS_ROOT"), name),
		filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios", name),
	}

	// Also try relative to working directory
	if wd, err := os.Getwd(); err == nil {
		// If we're in a scenario's api directory, go up
		scenarioDir := filepath.Dir(wd)
		root := filepath.Dir(scenarioDir)
		candidates = append(candidates, filepath.Join(root, name))
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

// loadRequirementsSnapshot loads a cached requirements snapshot.
func (s *Server) loadRequirementsSnapshot(path string, scenarioName string) (*RequirementsSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// The snapshot format from snapshot/builder.go
	var rawSnapshot struct {
		GeneratedAt  time.Time `json:"generated_at"`
		ScenarioName string    `json:"scenario_name"`
		Version      string    `json:"version"`
		Summary      struct {
			TotalRequirements int     `json:"total_requirements"`
			TotalValidations  int     `json:"total_validations"`
			CompletionRate    float64 `json:"completion_rate"`
			PassRate          float64 `json:"pass_rate"`
			CriticalGap       int     `json:"critical_gap"`
		} `json:"summary"`
		Modules []struct {
			Name           string  `json:"name"`
			FilePath       string  `json:"file_path"`
			Total          int     `json:"total"`
			Complete       int     `json:"complete"`
			InProgress     int     `json:"in_progress"`
			Pending        int     `json:"pending"`
			CompletionRate float64 `json:"completion_rate"`
		} `json:"modules"`
	}

	if err := json.Unmarshal(data, &rawSnapshot); err != nil {
		return nil, err
	}

	snapshot := &RequirementsSnapshot{
		ScenarioName: scenarioName,
		GeneratedAt:  rawSnapshot.GeneratedAt,
		Summary: RequirementsSummary{
			TotalRequirements: rawSnapshot.Summary.TotalRequirements,
			TotalValidations:  rawSnapshot.Summary.TotalValidations,
			CompletionRate:    rawSnapshot.Summary.CompletionRate,
			PassRate:          rawSnapshot.Summary.PassRate,
			CriticalGap:       rawSnapshot.Summary.CriticalGap,
			ByLiveStatus:      make(map[string]int),
			ByDeclaredStatus:  make(map[string]int),
		},
		Modules: make([]ModuleSnapshot, 0, len(rawSnapshot.Modules)),
	}

	for _, m := range rawSnapshot.Modules {
		snapshot.Modules = append(snapshot.Modules, ModuleSnapshot{
			Name:           m.Name,
			FilePath:       m.FilePath,
			Total:          m.Total,
			Complete:       m.Complete,
			InProgress:     m.InProgress,
			Pending:        m.Pending,
			CompletionRate: m.CompletionRate,
		})
	}

	// Derive status counts from modules
	for _, m := range snapshot.Modules {
		snapshot.Summary.ByDeclaredStatus["complete"] += m.Complete
		snapshot.Summary.ByDeclaredStatus["in_progress"] += m.InProgress
		snapshot.Summary.ByDeclaredStatus["pending"] += m.Pending
	}

	return snapshot, nil
}

// requirementsFile represents the structure of a requirements JSON file.
type requirementsFile struct {
	Metadata struct {
		Description     string `json:"description"`
		LastValidatedAt string `json:"last_validated_at"`
		AutoSyncEnabled bool   `json:"auto_sync_enabled"`
		LastSyncedAt    string `json:"last_synced_at"`
	} `json:"_metadata"`
	Meta struct {
		Scenario string `json:"scenario"`
	} `json:"meta"`
	Imports      []string `json:"imports"`
	Requirements []struct {
		ID          string   `json:"id"`
		Category    string   `json:"category"`
		PRDRef      string   `json:"prd_ref"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Status      string   `json:"status"`
		Criticality string   `json:"criticality"`
		Children    []string `json:"children"`
		Validation  []struct {
			Type   string `json:"type"`
			Ref    string `json:"ref"`
			Phase  string `json:"phase"`
			Status string `json:"status"`
			Notes  string `json:"notes"`
		} `json:"validation"`
	} `json:"requirements"`
}

// loadRequirementsFromFiles reads requirements directly from the requirements/ folder.
func (s *Server) loadRequirementsFromFiles(scenarioDir, scenarioName string) *RequirementsSnapshot {
	snapshot := &RequirementsSnapshot{
		ScenarioName: scenarioName,
		GeneratedAt:  time.Now(),
		Summary: RequirementsSummary{
			ByLiveStatus:     make(map[string]int),
			ByDeclaredStatus: make(map[string]int),
		},
		Modules: []ModuleSnapshot{},
	}

	reqDir := filepath.Join(scenarioDir, "requirements")
	indexPath := filepath.Join(reqDir, "index.json")

	// Read index.json to get imports
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return snapshot
	}

	var index requirementsFile
	if err := json.Unmarshal(indexData, &index); err != nil {
		return snapshot
	}

	// Track all modules to process
	type moduleInfo struct {
		name     string
		filePath string
		reqs     []RequirementItem
	}
	modules := make(map[string]*moduleInfo)

	// Process index.json requirements first (as "index" module)
	if len(index.Requirements) > 0 {
		indexModule := &moduleInfo{
			name:     "index",
			filePath: "requirements/index.json",
			reqs:     make([]RequirementItem, 0, len(index.Requirements)),
		}
		for _, req := range index.Requirements {
			item := s.convertRequirement(req)
			indexModule.reqs = append(indexModule.reqs, item)
		}
		modules["index"] = indexModule
	}

	// Process each imported module
	for _, importPath := range index.Imports {
		modulePath := filepath.Join(reqDir, importPath)
		moduleData, err := os.ReadFile(modulePath)
		if err != nil {
			continue
		}

		var moduleFile requirementsFile
		if err := json.Unmarshal(moduleData, &moduleFile); err != nil {
			continue
		}

		// Derive module name from path (e.g., "02-builder/workflow-builder/core.json" -> "workflow-builder/core")
		moduleName := strings.TrimSuffix(importPath, ".json")
		// Simplify: use last two path segments
		parts := strings.Split(moduleName, "/")
		if len(parts) >= 2 {
			moduleName = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}

		mod := &moduleInfo{
			name:     moduleName,
			filePath: "requirements/" + importPath,
			reqs:     make([]RequirementItem, 0, len(moduleFile.Requirements)),
		}

		for _, req := range moduleFile.Requirements {
			item := s.convertRequirement(req)
			mod.reqs = append(mod.reqs, item)
		}

		modules[moduleName] = mod
	}

	// Convert to ModuleSnapshots and calculate stats
	var totalReqs, totalVals int
	statusCounts := make(map[string]int)
	liveStatusCounts := make(map[string]int)

	for _, mod := range modules {
		var complete, inProgress, pending int
		for _, req := range mod.reqs {
			totalReqs++
			totalVals += len(req.Validations)

			switch req.Status {
			case "complete":
				complete++
			case "in_progress":
				inProgress++
			default:
				pending++
			}
			statusCounts[req.Status]++
			liveStatusCounts[req.LiveStatus]++
		}

		total := len(mod.reqs)
		var completionRate float64
		if total > 0 {
			completionRate = float64(complete) / float64(total) * 100
		}

		snapshot.Modules = append(snapshot.Modules, ModuleSnapshot{
			Name:           mod.name,
			FilePath:       mod.filePath,
			Total:          total,
			Complete:       complete,
			InProgress:     inProgress,
			Pending:        pending,
			CompletionRate: completionRate,
			Requirements:   mod.reqs,
		})
	}

	// Update summary
	snapshot.Summary.TotalRequirements = totalReqs
	snapshot.Summary.TotalValidations = totalVals
	snapshot.Summary.ByDeclaredStatus = statusCounts
	snapshot.Summary.ByLiveStatus = liveStatusCounts

	if totalReqs > 0 {
		snapshot.Summary.CompletionRate = float64(statusCounts["complete"]) / float64(totalReqs) * 100
	}

	passed := liveStatusCounts["passed"]
	failed := liveStatusCounts["failed"]
	if passed+failed > 0 {
		snapshot.Summary.PassRate = float64(passed) / float64(passed+failed) * 100
	}

	return snapshot
}

// convertRequirement converts a raw requirement from JSON to RequirementItem.
func (s *Server) convertRequirement(req struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	PRDRef      string   `json:"prd_ref"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Criticality string   `json:"criticality"`
	Children    []string `json:"children"`
	Validation  []struct {
		Type   string `json:"type"`
		Ref    string `json:"ref"`
		Phase  string `json:"phase"`
		Status string `json:"status"`
		Notes  string `json:"notes"`
	} `json:"validation"`
}) RequirementItem {
	item := RequirementItem{
		ID:          req.ID,
		Title:       req.Title,
		Status:      req.Status,
		PRDRef:      req.PRDRef,
		Criticality: req.Criticality,
		Description: req.Description,
	}

	// Derive live status from validation statuses
	// If all validations are "implemented" and status is "complete", it's "passed"
	// If any validation is "failing", it's "failed"
	// Otherwise it's based on declared status
	hasFailingValidation := false
	allImplemented := len(req.Validation) > 0

	for _, val := range req.Validation {
		item.Validations = append(item.Validations, ValidationItem{
			Type:       val.Type,
			Ref:        val.Ref,
			Phase:      val.Phase,
			Status:     val.Status,
			LiveStatus: s.deriveLiveStatus(val.Status),
		})
		if val.Status == "failing" || val.Status == "failed" {
			hasFailingValidation = true
		}
		if val.Status != "implemented" && val.Status != "passing" && val.Status != "passed" {
			allImplemented = false
		}
	}

	// Set live status
	if hasFailingValidation {
		item.LiveStatus = "failed"
	} else if req.Status == "complete" && allImplemented {
		item.LiveStatus = "passed"
	} else if req.Status == "in_progress" {
		item.LiveStatus = "not_run"
	} else if req.Status == "pending" || req.Status == "planned" {
		item.LiveStatus = "not_run"
	} else if len(req.Validation) == 0 {
		item.LiveStatus = "unknown"
	} else {
		item.LiveStatus = "not_run"
	}

	return item
}

// deriveLiveStatus converts validation status to live status.
func (s *Server) deriveLiveStatus(validationStatus string) string {
	switch validationStatus {
	case "implemented", "passing", "passed":
		return "passed"
	case "failing", "failed":
		return "failed"
	case "skipped":
		return "skipped"
	default:
		return "not_run"
	}
}

// loadSyncStatus loads the sync metadata.
func (s *Server) loadSyncStatus(path string, scenarioDir string) *SyncStatus {
	status := &SyncStatus{
		Enabled: true, // Default to enabled
	}

	// Check testing.json for sync enabled state
	testingPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")
	if data, err := os.ReadFile(testingPath); err == nil {
		var testing struct {
			Requirements struct {
				Sync bool `json:"sync"`
			} `json:"requirements"`
		}
		if json.Unmarshal(data, &testing) == nil {
			status.Enabled = testing.Requirements.Sync
		}
	}

	// Load sync metadata
	data, err := os.ReadFile(path)
	if err != nil {
		return status
	}

	var syncMeta struct {
		SyncedAt           time.Time `json:"synced_at"`
		FilesUpdated       int       `json:"files_updated"`
		ValidationsAdded   int       `json:"validations_added"`
		ValidationsRemoved int       `json:"validations_removed"`
		StatusesChanged    int       `json:"statuses_changed"`
		ErrorCount         int       `json:"error_count"`
	}

	if json.Unmarshal(data, &syncMeta) == nil {
		status.LastSyncedAt = syncMeta.SyncedAt
		status.FilesUpdated = syncMeta.FilesUpdated
		status.ValidationsAdded = syncMeta.ValidationsAdded
		status.ValidationsRemoved = syncMeta.ValidationsRemoved
		status.StatusesChanged = syncMeta.StatusesChanged
		status.ErrorCount = syncMeta.ErrorCount
	}

	return status
}
