package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	rulespkg "scenario-auditor/rules"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type StandardsViolation struct {
	ID             string `json:"id"`
	ScenarioName   string `json:"scenario_name"`
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	CodeSnippet    string `json:"code_snippet,omitempty"`
	Recommendation string `json:"recommendation"`
	Standard       string `json:"standard"`
	DiscoveredAt   string `json:"discovered_at"`
}

type StandardsCheckResult struct {
	CheckID      string               `json:"check_id"`
	Status       string               `json:"status"`
	ScanType     string               `json:"scan_type"`
	StartedAt    string               `json:"started_at"`
	CompletedAt  string               `json:"completed_at"`
	Duration     float64              `json:"duration_seconds"`
	FilesScanned int                  `json:"files_scanned"`
	Violations   []StandardsViolation `json:"violations"`
	Statistics   map[string]int       `json:"statistics"`
	Message      string               `json:"message"`
	ScenarioName string               `json:"scenario_name,omitempty"`
}

const (
	targetAPI         = "api"
	targetMainGo      = "main_go"
	targetUI          = "ui"
	targetCLI         = "cli"
	targetTest        = "test"
	targetServiceJSON = "service_json"
	targetMakefile    = "makefile"
	targetStructure   = "structure"
)

var (
	errStandardsScanCancelled = errors.New("standards scan cancelled")
	errStandardsScanNotFound  = errors.New("standards scan not found")
	errStandardsScanFinished  = errors.New("standards scan already finished")
)

type standardsScanTarget struct {
	Name string
	Path string
}

type StandardsScanStatus struct {
	ID                 string                `json:"id"`
	Scenario           string                `json:"scenario"`
	ScanType           string                `json:"scan_type"`
	Status             string                `json:"status"`
	StartedAt          time.Time             `json:"started_at"`
	CompletedAt        *time.Time            `json:"completed_at,omitempty"`
	ElapsedSeconds     float64               `json:"elapsed_seconds"`
	TotalScenarios     int                   `json:"total_scenarios"`
	ProcessedScenarios int                   `json:"processed_scenarios"`
	ProcessedFiles     int                   `json:"processed_files"`
	TotalFiles         int                   `json:"total_files"`
	CurrentScenario    string                `json:"current_scenario,omitempty"`
	CurrentFile        string                `json:"current_file,omitempty"`
	Message            string                `json:"message,omitempty"`
	Error              string                `json:"error,omitempty"`
	Result             *StandardsCheckResult `json:"result,omitempty"`
}

type StandardsScanJob struct {
	mu     sync.RWMutex
	status StandardsScanStatus
	cancel context.CancelFunc
}

type StandardsScanManager struct {
	mu   sync.RWMutex
	jobs map[string]*StandardsScanJob
}

var standardsScanManager = newStandardsScanManager()

var allowedExtensions = map[string]struct{}{
	".go":   {},
	".ts":   {},
	".tsx":  {},
	".js":   {},
	".jsx":  {},
	".sh":   {},
	".json": {},
	".yaml": {},
	".yml":  {},
	".html": {},
	".css":  {},
	".py":   {},
	".java": {},
	".md":   {},
}

func newStandardsScanManager() *StandardsScanManager {
	return &StandardsScanManager{
		jobs: make(map[string]*StandardsScanJob),
	}
}

func (m *StandardsScanManager) StartScan(scenarioName, scanType string, standards []string) (StandardsScanStatus, error) {
	targets, err := buildStandardsScanTargets(scenarioName)
	if err != nil {
		return StandardsScanStatus{}, err
	}
	if len(targets) == 0 {
		return StandardsScanStatus{}, fmt.Errorf("no scenarios available to scan")
	}

	ctx, cancel := context.WithCancel(context.Background())
	jobID := fmt.Sprintf("standards-%s", uuid.NewString())
	started := time.Now()

	job := &StandardsScanJob{
		cancel: cancel,
		status: StandardsScanStatus{
			ID:             jobID,
			Scenario:       scenarioName,
			ScanType:       scanType,
			Status:         "running",
			StartedAt:      started,
			ElapsedSeconds: 0,
			TotalScenarios: len(targets),
			Message:        "Standards scan started",
		},
	}

	m.mu.Lock()
	m.jobs[jobID] = job
	m.mu.Unlock()

	go job.run(ctx, targets, scenarioName, scanType, standards)

	return job.snapshot(), nil
}

func (m *StandardsScanManager) Get(jobID string) (*StandardsScanJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[jobID]
	return job, ok
}

func (m *StandardsScanManager) Cancel(jobID string) (StandardsScanStatus, error) {
	job, ok := m.Get(jobID)
	if !ok {
		return StandardsScanStatus{}, errStandardsScanNotFound
	}

	err := job.requestCancel()
	return job.snapshot(), err
}

func (job *StandardsScanJob) update(fn func(*StandardsScanStatus)) {
	job.mu.Lock()
	defer job.mu.Unlock()
	fn(&job.status)
	if !job.status.StartedAt.IsZero() {
		job.status.ElapsedSeconds = time.Since(job.status.StartedAt).Seconds()
	}
}

func (job *StandardsScanJob) snapshot() StandardsScanStatus {
	job.mu.RLock()
	defer job.mu.RUnlock()
	copy := job.status
	if !copy.StartedAt.IsZero() && copy.Status != "pending" {
		copy.ElapsedSeconds = time.Since(copy.StartedAt).Seconds()
	}
	return copy
}

func (job *StandardsScanJob) requestCancel() error {
	job.mu.Lock()
	status := job.status.Status
	if status == "completed" || status == "failed" || status == "cancelled" {
		job.mu.Unlock()
		return errStandardsScanFinished
	}
	if status == "cancelling" {
		job.mu.Unlock()
		return nil
	}
	job.status.Status = "cancelling"
	job.status.Message = "Cancellation requested"
	if !job.status.StartedAt.IsZero() {
		job.status.ElapsedSeconds = time.Since(job.status.StartedAt).Seconds()
	}
	cancel := job.cancel
	job.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (job *StandardsScanJob) markCancelled() {
	completedAt := time.Now()
	job.update(func(status *StandardsScanStatus) {
		status.Status = "cancelled"
		status.CompletedAt = &completedAt
		status.Message = "Standards scan cancelled"
		status.CurrentScenario = ""
		status.CurrentFile = ""
	})
}

func (job *StandardsScanJob) markFailed(err error) {
	completedAt := time.Now()
	job.update(func(status *StandardsScanStatus) {
		status.Status = "failed"
		status.CompletedAt = &completedAt
		status.Error = err.Error()
		status.Message = "Standards scan failed"
		status.CurrentScenario = ""
		status.CurrentFile = ""
	})
}

func (job *StandardsScanJob) markCompleted(result StandardsCheckResult, processedFiles int) {
	completedAt := time.Now()
	job.update(func(status *StandardsScanStatus) {
		status.Status = "completed"
		status.CompletedAt = &completedAt
		status.Message = result.Message
		status.Result = &result
		status.ProcessedFiles = processedFiles
		status.TotalFiles = processedFiles
		status.CurrentScenario = ""
		status.CurrentFile = ""
	})
}

func (job *StandardsScanJob) run(ctx context.Context, targets []standardsScanTarget, scenarioName, scanType string, specificStandards []string) {
	logger := NewLogger()
	jobSnapshot := job.snapshot()
	start := jobSnapshot.StartedAt
	if start.IsZero() {
		start = time.Now()
		job.update(func(status *StandardsScanStatus) {
			status.StartedAt = start
		})
	}

	processedScenarios := 0
	totalFiles := 0
	var allViolations []StandardsViolation

	for _, target := range targets {
		select {
		case <-ctx.Done():
			job.markCancelled()
			logger.Info(fmt.Sprintf("Standards scan %s cancelled", jobSnapshot.ID))
			return
		default:
		}

		job.update(func(status *StandardsScanStatus) {
			status.CurrentScenario = target.Name
		})

		onFile := func(fileScenario, scenarioRelative string) {
			scenarioForStatus := fileScenario
			if scenarioForStatus == "" {
				scenarioForStatus = target.Name
			}
			job.update(func(status *StandardsScanStatus) {
				status.ProcessedFiles++
				status.CurrentScenario = scenarioForStatus
				status.CurrentFile = scenarioRelative
			})
		}

		violations, filesScanned, err := performStandardsCheck(ctx, target.Path, scanType, specificStandards, onFile)
		if err != nil {
			if errors.Is(err, errStandardsScanCancelled) || errors.Is(err, context.Canceled) {
				job.markCancelled()
				logger.Info(fmt.Sprintf("Standards scan %s cancelled during processing", jobSnapshot.ID))
				return
			}
			job.markFailed(err)
			logger.Error(fmt.Sprintf("Standards scan %s failed", jobSnapshot.ID), err)
			return
		}

		totalFiles += filesScanned
		allViolations = append(allViolations, violations...)

		processedScenarios++
		job.update(func(status *StandardsScanStatus) {
			status.ProcessedScenarios = processedScenarios
			status.CurrentFile = ""
		})
	}

	completedAt := time.Now()
	duration := completedAt.Sub(start)

	stats := computeViolationStats(allViolations)
	message := buildScanCompletionMessage(scenarioName, len(targets), len(allViolations))
	result := StandardsCheckResult{
		CheckID:      jobSnapshot.ID,
		Status:       "completed",
		ScanType:     scanType,
		StartedAt:    start.Format(time.RFC3339),
		CompletedAt:  completedAt.Format(time.RFC3339),
		Duration:     duration.Seconds(),
		FilesScanned: totalFiles,
		Violations:   allViolations,
		Statistics:   stats,
		Message:      message,
	}
	if scenarioName != "all" {
		result.ScenarioName = scenarioName
	}

	standardsStore.StoreViolations(scenarioName, allViolations)
	logger.Info(fmt.Sprintf("Stored %d standards violations for %s", len(allViolations), scenarioName))

	job.markCompleted(result, totalFiles)
	logger.Info(fmt.Sprintf("Standards scan %s completed", jobSnapshot.ID))
}

func getScenariosRoot() string {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	return filepath.Join(vrooliRoot, "scenarios")
}

func buildStandardsScanTargets(scenarioName string) ([]standardsScanTarget, error) {
	root := getScenariosRoot()

	if scenarioName == "" {
		scenarioName = "all"
	}

	if strings.EqualFold(scenarioName, "all") {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		var targets []standardsScanTarget
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			targets = append(targets, standardsScanTarget{
				Name: entry.Name(),
				Path: filepath.Join(root, entry.Name()),
			})
		}
		sort.Slice(targets, func(i, j int) bool {
			return targets[i].Name < targets[j].Name
		})
		return targets, nil
	}

	scenarioPath := filepath.Join(root, scenarioName)
	info, err := os.Stat(scenarioPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("scenario %s not found: %w", scenarioName, err)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenario path %s is not a directory", scenarioPath)
	}

	return []standardsScanTarget{{
		Name: scenarioName,
		Path: scenarioPath,
	}}, nil
}

func buildScanCompletionMessage(scenarioName string, scenarioCount, violations int) string {
	if strings.EqualFold(scenarioName, "all") {
		return fmt.Sprintf("Standards compliance check completed across %d scenarios. Found %d violations.", scenarioCount, violations)
	}
	return fmt.Sprintf("Standards compliance check completed for %s. Found %d violations.", scenarioName, violations)
}

func computeViolationStats(violations []StandardsViolation) map[string]int {
	stats := map[string]int{
		"total":    len(violations),
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for _, violation := range violations {
		switch strings.ToLower(violation.Severity) {
		case "critical":
			stats["critical"]++
		case "high":
			stats["high"]++
		case "low":
			stats["low"]++
		default:
			stats["medium"]++
		}
	}

	return stats
}

// Standards compliance check handler
func enhancedStandardsCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := strings.TrimSpace(vars["name"])
	if scenarioName == "" {
		scenarioName = "all"
	} else if strings.EqualFold(scenarioName, "all") {
		scenarioName = "all"
	}

	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	var checkRequest struct {
		Type      string   `json:"type"`
		Standards []string `json:"standards"`
	}
	if err := json.NewDecoder(r.Body).Decode(&checkRequest); err != nil {
		checkRequest.Type = "full"
	}

	if checkRequest.Type != "quick" && checkRequest.Type != "full" && checkRequest.Type != "targeted" {
		checkRequest.Type = "full"
	}

	status, err := standardsScanManager.StartScan(scenarioName, checkRequest.Type, checkRequest.Standards)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, err)
			return
		}
		logger.Error("Failed to start standards compliance scan", err)
		HTTPError(w, "Failed to start standards scan", http.StatusInternalServerError, err)
		return
	}

	logger.Info(fmt.Sprintf("Started %s standards compliance scan %s for %s", checkRequest.Type, status.ID, scenarioName))

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id": status.ID,
		"status": status,
	})
}

func performStandardsCheck(ctx context.Context, scanPath, _ string, specificStandards []string, onFile func(string, string)) ([]StandardsViolation, int, error) {
	logger := NewLogger()

	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		return nil, 0, err
	}

	ruleBuckets, activeRules := buildRuleBuckets(ruleInfos, specificStandards)
	if len(activeRules) == 0 {
		return nil, 0, nil
	}

	type structureScenarioInfo struct {
		files map[string]struct{}
	}

	structureData := make(map[string]*structureScenarioInfo)
	structurePaths := make(map[string]string)
	scenariosRoot := getScenariosRoot()
	ensureStructureScenario := func(name string) *structureScenarioInfo {
		if strings.TrimSpace(name) == "" {
			return nil
		}
		info, ok := structureData[name]
		if !ok {
			info = &structureScenarioInfo{files: make(map[string]struct{})}
			structureData[name] = info
		}
		if _, exists := structurePaths[name]; !exists {
			structurePaths[name] = filepath.Join(scenariosRoot, name)
		}
		return info
	}

	rootScenario := filepath.Base(scanPath)
	if rootScenario != "" {
		ensureStructureScenario(rootScenario)
		structurePaths[rootScenario] = scanPath
	}

	var violations []StandardsViolation
	filesScanned := 0

	if ctx == nil {
		ctx = context.Background()
	}

	err = filepath.Walk(scanPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return errStandardsScanCancelled
		default:
		}

		if info.IsDir() {
			if shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		scenarioName, scenarioRelative, targets := classifyFileTargets(path)
		if scenarioName != "" && scenarioRelative != "" {
			if info := ensureStructureScenario(scenarioName); info != nil {
				relative := filepath.ToSlash(scenarioRelative)
				info.files[relative] = struct{}{}
			}
		}
		if len(targets) == 0 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to read file %s", path), err)
			return nil
		}

		if onFile != nil {
			onFile(scenarioName, scenarioRelative)
		}

		filesScanned++
		rulesForFile := collectRulesForTargets(targets, ruleBuckets)

		for _, rule := range rulesForFile {
			if !rule.Implementation.Valid {
				continue
			}

			select {
			case <-ctx.Done():
				return errStandardsScanCancelled
			default:
			}

			ruleViolations, execErr := rule.Check(string(content), path, scenarioName)
			if execErr != nil {
				logger.Error(fmt.Sprintf("Rule %s execution failed on %s", rule.ID, path), execErr)
				continue
			}

			for _, rv := range ruleViolations {
				violations = append(violations, convertRuleViolationToStandards(rule, rv, scenarioName, scenarioRelative))
			}
		}

		return nil
	})

	if errors.Is(err, errStandardsScanCancelled) || errors.Is(err, context.Canceled) {
		return violations, filesScanned, errStandardsScanCancelled
	}

	if err != nil {
		return violations, filesScanned, err
	}

	structureRules := collectRulesForTargets([]string{targetStructure}, ruleBuckets)
	if len(structureRules) > 0 {
		for scenario, info := range structureData {
			files := make([]string, 0, len(info.files))
			for relative := range info.files {
				files = append(files, relative)
			}
			sort.Strings(files)
			payload := struct {
				Scenario string   `json:"scenario"`
				Files    []string `json:"files"`
			}{
				Scenario: scenario,
				Files:    files,
			}
			encoded, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				logger.Error("Failed to encode structure payload", marshalErr)
				continue
			}
			scenarioPath := structurePaths[scenario]
			for _, rule := range structureRules {
				if !rule.Implementation.Valid {
					continue
				}

				ruleViolations, execErr := rule.Check(string(encoded), scenarioPath, scenario)
				if execErr != nil {
					logger.Error(fmt.Sprintf("Structure rule %s execution failed for %s", rule.ID, scenario), execErr)
					continue
				}

				for _, rv := range ruleViolations {
					violations = append(violations, convertRuleViolationToStandards(rule, rv, scenario, rv.FilePath))
				}
			}
		}
	}

	return violations, filesScanned, nil
}

func buildRuleBuckets(ruleInfos map[string]RuleInfo, specific []string) (map[string][]RuleInfo, map[string]RuleInfo) {
	allowed := make(map[string]struct{})
	for _, item := range specific {
		id := strings.TrimSpace(item)
		if id != "" {
			allowed[id] = struct{}{}
		}
	}

	buckets := make(map[string][]RuleInfo)
	active := make(map[string]RuleInfo)
	states := ruleStateStore.GetAllStates()

	for id, info := range ruleInfos {
		if len(allowed) > 0 {
			if _, ok := allowed[id]; !ok {
				continue
			}
		}

		if enabled, ok := states[id]; ok {
			info.Enabled = enabled
		}
		if !info.Enabled {
			continue
		}

		targets := info.Targets
		if len(targets) == 0 {
			targets = defaultTargetsForRule(info)
		} else {
			targets = normalizeTargets(targets)
		}

		if len(targets) == 0 {
			continue
		}

		info.Targets = targets
		ruleInfos[id] = info
		active[id] = info

		for _, target := range targets {
			buckets[target] = append(buckets[target], info)
		}
	}

	return buckets, active
}

func normalizeTargets(targets []string) []string {
	dedup := make(map[string]struct{})
	for _, t := range targets {
		name := strings.ToLower(strings.TrimSpace(t))
		if name == "" {
			continue
		}
		dedup[name] = struct{}{}
	}

	result := make([]string, 0, len(dedup))
	for name := range dedup {
		result = append(result, name)
	}
	return result
}

func defaultTargetsForRule(rule RuleInfo) []string {
	switch rule.Category {
	case "api":
		return []string{targetAPI}
	case "cli":
		return []string{targetCLI}
	case "ui":
		return []string{targetUI}
	case "test":
		return []string{targetTest}
	case "config":
		return []string{targetAPI, targetCLI}
	case "makefile":
		return []string{targetMakefile}
	case "structure":
		return []string{targetStructure}
	default:
		return nil
	}
}

func collectRulesForTargets(targets []string, buckets map[string][]RuleInfo) map[string]RuleInfo {
	result := make(map[string]RuleInfo)
	for _, target := range targets {
		for _, rule := range buckets[target] {
			result[rule.ID] = rule
		}
	}
	return result
}

func classifyFileTargets(fullPath string) (string, string, []string) {
	scenarioName, scenarioRelative := scenarioNameAndRelative(fullPath)
	if scenarioName == "" {
		return "", "", nil
	}

	scenarioRelative = filepath.ToSlash(scenarioRelative)
	if scenarioRelative == "" {
		return scenarioName, scenarioRelative, nil
	}

	if scenarioRelative == ".vrooli/service.json" {
		return scenarioName, scenarioRelative, []string{targetServiceJSON}
	}

	ext := strings.ToLower(filepath.Ext(scenarioRelative))
	if ext != "" {
		if _, ok := allowedExtensions[ext]; !ok {
			return scenarioName, scenarioRelative, nil
		}
	}

	var targets []string
	if scenarioRelative == "Makefile" {
		targets = append(targets, targetMakefile)
	}
	if strings.HasPrefix(scenarioRelative, "api/") {
		targets = append(targets, targetAPI)
	}
	if strings.HasPrefix(scenarioRelative, "cli/") {
		targets = append(targets, targetCLI)
	}
	if strings.HasPrefix(scenarioRelative, "ui/") {
		targets = append(targets, targetUI)
	}
	if strings.HasPrefix(scenarioRelative, "test/") {
		targets = append(targets, targetTest)
	}
	if strings.HasSuffix(scenarioRelative, "main.go") {
		targets = append(targets, targetMainGo)
	}

	return scenarioName, scenarioRelative, targets
}

func scenarioNameAndRelative(path string) (string, string) {
	marker := string(filepath.Separator) + "scenarios" + string(filepath.Separator)
	index := strings.Index(path, marker)
	if index == -1 {
		return "", ""
	}
	remainder := path[index+len(marker):]
	parts := strings.SplitN(remainder, string(filepath.Separator), 2)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func toScenarioRelative(path string, scenarioName string) string {
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}

	marker := string(filepath.Separator) + "scenarios" + string(filepath.Separator) + scenarioName + string(filepath.Separator)
	index := strings.Index(path, marker)
	if index == -1 {
		return filepath.ToSlash(path)
	}
	relative := path[index+len(marker):]
	return filepath.ToSlash(relative)
}

func convertRuleViolationToStandards(rule RuleInfo, violation rulespkg.Violation, scenarioName, fallbackPath string) StandardsViolation {
	severity := strings.ToLower(firstNonEmpty(violation.Severity, rule.Severity))
	if severity == "" {
		severity = "medium"
	}

	filePath := firstNonEmpty(violation.FilePath, violation.File)
	if filePath == "" {
		filePath = fallbackPath
	} else {
		filePath = toScenarioRelative(filePath, scenarioName)
	}

	recommendation := violation.Recommendation
	if recommendation == "" {
		recommendation = fmt.Sprintf("Review rule %s guidance", rule.Name)
	}

	title := firstNonEmpty(violation.Title, rule.Name)
	description := firstNonEmpty(violation.Message, rule.Description)
	standard := firstNonEmpty(violation.Standard, rule.Standard)

	return StandardsViolation{
		ID:             uuid.New().String(),
		ScenarioName:   scenarioName,
		Type:           firstNonEmpty(violation.RuleID, rule.ID),
		Severity:       severity,
		Title:          title,
		Description:    description,
		FilePath:       filePath,
		LineNumber:     violation.Line,
		CodeSnippet:    violation.CodeSnippet,
		Recommendation: recommendation,
		Standard:       standard,
		DiscoveredAt:   time.Now().Format(time.RFC3339),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func shouldSkipDirectory(path string) bool {
	switch filepath.Base(path) {
	case "vendor", "node_modules", ".git", "dist", "build", ".next", ".nuxt", "target", "bin", "obj":
		return true
	default:
		return false
	}
}

func extractScenarioName(path string) string {
	name, _ := scenarioNameAndRelative(path)
	if name == "" {
		return "unknown"
	}
	return name
}

func getStandardsCheckStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := strings.TrimSpace(vars["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}

	job, ok := standardsScanManager.Get(jobID)
	if !ok {
		HTTPError(w, "Standards scan not found", http.StatusNotFound, errStandardsScanNotFound)
		return
	}

	status := job.snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func cancelStandardsCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := strings.TrimSpace(vars["jobId"])
	if jobID == "" {
		HTTPError(w, "Job ID is required", http.StatusBadRequest, nil)
		return
	}

	status, err := standardsScanManager.Cancel(jobID)
	if err != nil {
		if errors.Is(err, errStandardsScanNotFound) {
			HTTPError(w, "Standards scan not found", http.StatusNotFound, err)
			return
		}
		if errors.Is(err, errStandardsScanFinished) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"status":  status,
				"message": fmt.Sprintf("Scan already %s", status.Status),
			})
			return
		}
		HTTPError(w, "Failed to cancel standards scan", http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  status,
	})
}

// getStandardsViolationsHandler returns cached violations for a scenario
func getStandardsViolationsHandler(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")

	w.Header().Set("Content-Type", "application/json")

	violations := standardsStore.GetViolations(scenario)

	response := map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
		"timestamp":  time.Now().Format(time.RFC3339),
		"scenario":   scenario,
	}

	if len(violations) == 0 {
		response["note"] = "No standards violations found. Run a standards compliance check to detect violations."
	}

	json.NewEncoder(w).Encode(response)
}
