package httpserver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/packages/artifactpaths"
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
	Enabled            bool       `json:"enabled"`
	LastSyncedAt       *time.Time `json:"lastSyncedAt,omitempty"`
	FilesUpdated       int        `json:"filesUpdated"`
	ValidationsAdded   int        `json:"validationsAdded"`
	ValidationsRemoved int        `json:"validationsRemoved"`
	StatusesChanged    int        `json:"statusesChanged"`
	ErrorCount         int        `json:"errorCount"`
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

type requirementsPaths struct {
	requirementsDir string
	indexPath       string
	snapshotPath    string
	syncStatusPath  string
}

type requirementsModuleData struct {
	name         string
	filePath     string
	requirements []RequirementItem
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

	s.writeJSON(w, http.StatusOK, s.loadScenarioRequirementsView(scenarioDir, name))
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
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && err != io.EOF {
			s.writeError(w, http.StatusBadRequest, "invalid sync payload")
			return
		}
	}

	scenarioDir := s.resolveScenarioDir(name)
	if scenarioDir == "" {
		s.writeError(w, http.StatusNotFound, "scenario not found")
		return
	}

	// Dry run just returns a preview (not yet implemented)
	if payload.DryRun {
		preview := SyncPreviewResponse{
			ScenarioName: name,
			Changes:      []SyncChange{},
		}
		s.writeJSON(w, http.StatusOK, preview)
		return
	}

	// Perform actual sync if service is available
	if s.requirementsSyncer != nil {
		if err := s.requirementsSyncer.Sync(r.Context(), scenarioDir); err != nil {
			log.Printf("requirements sync error: %v", err)
			// Continue anyway - we'll return current state
		}
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "completed",
		"snapshot": s.loadScenarioRequirementsView(scenarioDir, name),
	})
}

// resolveScenarioDir finds the directory for a scenario.
func (s *Server) resolveScenarioDir(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	if root := s.resolveRepoRoot(); root != "" {
		if path, err := repocontract.ResolveScenarioPath(root, name); err == nil {
			if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
				return path
			}
		}
	}

	if scenariosRoot := s.resolveScenariosRoot(); scenariosRoot != "" {
		candidate := filepath.Join(scenariosRoot, name)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return ""
}

func (s *Server) loadScenarioRequirementsView(scenarioDir, scenarioName string) *RequirementsSnapshot {
	paths := newRequirementsPaths(scenarioDir)
	snapshot, err := s.loadRequirementsSnapshot(paths.snapshotPath, scenarioName)
	if err != nil || len(snapshot.Modules) == 0 {
		snapshot = s.loadRequirementsFromFiles(scenarioDir, scenarioName)
	} else {
		s.enrichSnapshotWithRequirements(snapshot, scenarioDir)
	}
	snapshot.SyncStatus = s.loadSyncStatus(paths.syncStatusPath, scenarioDir)
	return snapshot
}

func newRequirementsPaths(scenarioDir string) requirementsPaths {
	requirementsDir := filepath.Join(scenarioDir, "requirements")
	artifactRoot, _ := artifactpaths.ScenarioRootForDir(scenarioDir)
	return requirementsPaths{
		requirementsDir: requirementsDir,
		indexPath:       filepath.Join(requirementsDir, "index.json"),
		snapshotPath:    artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot, "requirements-sync", "latest.json"),
		syncStatusPath:  artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot, "sync", "latest.json"),
	}
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

// enrichSnapshotWithRequirements adds requirement details to a cached snapshot by loading from files.
// This allows us to use the cached summary stats while still providing full requirement data to the UI.
func (s *Server) enrichSnapshotWithRequirements(snapshot *RequirementsSnapshot, scenarioDir string) {
	modules, requirementsByKey, liveStatusCounts, err := s.loadRequirementModules(scenarioDir)
	if err != nil {
		return
	}
	for i := range snapshot.Modules {
		mod := &snapshot.Modules[i]

		if reqs, ok := requirementsByKey[mod.FilePath]; ok {
			mod.Requirements = reqs
			continue
		}
		if reqs, ok := requirementsByKey[mod.Name]; ok {
			mod.Requirements = reqs
			continue
		}
		if reqs, ok := requirementsByKey["requirements/"+mod.Name]; ok {
			mod.Requirements = reqs
		}
	}

	if len(snapshot.Modules) == 0 && len(modules) > 0 {
		snapshot.Modules = buildModuleSnapshots(modules)
	}
	snapshot.Summary.ByLiveStatus = liveStatusCounts
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
	modules, _, liveStatusCounts, err := s.loadRequirementModules(scenarioDir)
	if err != nil {
		return newEmptyRequirementsSnapshot(scenarioName)
	}

	snapshot := newEmptyRequirementsSnapshot(scenarioName)
	snapshot.Modules = buildModuleSnapshots(modules)
	snapshot.Summary = summarizeRequirementsSnapshot(snapshot.Modules, liveStatusCounts)
	return snapshot
}

func (s *Server) loadRequirementModules(scenarioDir string) ([]requirementsModuleData, map[string][]RequirementItem, map[string]int, error) {
	paths := newRequirementsPaths(scenarioDir)
	index, err := s.loadRequirementsIndex(paths.indexPath)
	if err != nil {
		return nil, nil, nil, err
	}

	modules := make([]requirementsModuleData, 0, len(index.Imports)+1)
	requirementsByKey := make(map[string][]RequirementItem, len(index.Imports)*3+3)
	liveStatusCounts := make(map[string]int)

	if len(index.Requirements) > 0 {
		module := requirementsModuleData{
			name:         "index",
			filePath:     "requirements/index.json",
			requirements: s.convertRequirements(index.Requirements, liveStatusCounts),
		}
		modules = append(modules, module)
		s.registerRequirementsKeys(requirementsByKey, paths.indexPath, module)
	}

	for _, importPath := range index.Imports {
		modulePath := filepath.Join(paths.requirementsDir, importPath)
		moduleFile, err := s.loadRequirementsIndex(modulePath)
		if err != nil {
			continue
		}

		module := requirementsModuleData{
			name:         moduleNameFromImport(importPath),
			filePath:     "requirements/" + importPath,
			requirements: s.convertRequirements(moduleFile.Requirements, liveStatusCounts),
		}
		modules = append(modules, module)
		s.registerRequirementsKeys(requirementsByKey, modulePath, module)
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].filePath < modules[j].filePath
	})
	return modules, requirementsByKey, liveStatusCounts, nil
}

func (s *Server) loadRequirementsIndex(path string) (requirementsFile, error) {
	var file requirementsFile
	data, err := os.ReadFile(path)
	if err != nil {
		return file, err
	}
	return file, json.Unmarshal(data, &file)
}

func (s *Server) convertRequirements(
	requirements []struct {
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
	},
	liveStatusCounts map[string]int,
) []RequirementItem {
	items := make([]RequirementItem, 0, len(requirements))
	for _, req := range requirements {
		item := s.convertRequirement(req)
		items = append(items, item)
		liveStatusCounts[item.LiveStatus]++
	}
	return items
}

func (s *Server) registerRequirementsKeys(target map[string][]RequirementItem, absolutePath string, module requirementsModuleData) {
	target[absolutePath] = module.requirements
	target[module.filePath] = module.requirements
	target[module.name] = module.requirements
	target[filepath.Base(module.filePath)] = module.requirements
}

func moduleNameFromImport(importPath string) string {
	moduleName := strings.TrimSuffix(importPath, ".json")
	parts := strings.Split(moduleName, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return moduleName
}

func newEmptyRequirementsSnapshot(scenarioName string) *RequirementsSnapshot {
	return &RequirementsSnapshot{
		ScenarioName: scenarioName,
		GeneratedAt:  time.Now(),
		Summary: RequirementsSummary{
			ByLiveStatus:     make(map[string]int),
			ByDeclaredStatus: make(map[string]int),
		},
		Modules: []ModuleSnapshot{},
	}
}

func buildModuleSnapshots(modules []requirementsModuleData) []ModuleSnapshot {
	snapshots := make([]ModuleSnapshot, 0, len(modules))
	for _, mod := range modules {
		complete, inProgress, pending := requirementStatusBreakdown(mod.requirements)
		total := len(mod.requirements)
		completionRate := 0.0
		if total > 0 {
			completionRate = float64(complete) / float64(total) * 100
		}
		snapshots = append(snapshots, ModuleSnapshot{
			Name:           mod.name,
			FilePath:       mod.filePath,
			Total:          total,
			Complete:       complete,
			InProgress:     inProgress,
			Pending:        pending,
			CompletionRate: completionRate,
			Requirements:   mod.requirements,
		})
	}
	return snapshots
}

func summarizeRequirementsSnapshot(modules []ModuleSnapshot, liveStatusCounts map[string]int) RequirementsSummary {
	summary := RequirementsSummary{
		ByLiveStatus:     liveStatusCounts,
		ByDeclaredStatus: make(map[string]int),
	}
	for _, mod := range modules {
		summary.TotalRequirements += mod.Total
		summary.TotalValidations += countValidations(mod.Requirements)
		summary.ByDeclaredStatus["complete"] += mod.Complete
		summary.ByDeclaredStatus["in_progress"] += mod.InProgress
		summary.ByDeclaredStatus["pending"] += mod.Pending
	}
	if summary.TotalRequirements > 0 {
		summary.CompletionRate = float64(summary.ByDeclaredStatus["complete"]) / float64(summary.TotalRequirements) * 100
	}
	passed := summary.ByLiveStatus["passed"]
	failed := summary.ByLiveStatus["failed"]
	if passed+failed > 0 {
		summary.PassRate = float64(passed) / float64(passed+failed) * 100
	}
	return summary
}

func requirementStatusBreakdown(requirements []RequirementItem) (complete, inProgress, pending int) {
	for _, req := range requirements {
		switch req.Status {
		case "complete":
			complete++
		case "in_progress":
			inProgress++
		default:
			pending++
		}
	}
	return complete, inProgress, pending
}

func countValidations(requirements []RequirementItem) int {
	total := 0
	for _, req := range requirements {
		total += len(req.Validations)
	}
	return total
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
},
) RequirementItem {
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
		if !syncMeta.SyncedAt.IsZero() {
			status.LastSyncedAt = &syncMeta.SyncedAt
		}
		status.FilesUpdated = syncMeta.FilesUpdated
		status.ValidationsAdded = syncMeta.ValidationsAdded
		status.ValidationsRemoved = syncMeta.ValidationsRemoved
		status.StatusesChanged = syncMeta.StatusesChanged
		status.ErrorCount = syncMeta.ErrorCount
	}

	return status
}
