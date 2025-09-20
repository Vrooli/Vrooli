package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
)

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
}

// Standards compliance check handler
func enhancedStandardsCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
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

	startTime := time.Now()

	var scanPath string
	scenarioCount := 1

	if scenarioName == "all" {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios")
		scenarios, _ := getVrooliScenarios()
		if scenarios != nil && scenarios.Scenarios != nil {
			scenarioCount = len(scenarios.Scenarios)
		}
	} else {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	}

	logger.Info(fmt.Sprintf("Starting %s standards compliance check on %s", checkRequest.Type, scenarioName))

	violations, filesScanned, err := performStandardsCheck(scanPath, checkRequest.Type, checkRequest.Standards)
	if err != nil {
		logger.Error("Standards compliance check failed", err)
		HTTPError(w, "Standards check failed", http.StatusInternalServerError, err)
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	stats := map[string]int{
		"total":    len(violations),
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for _, violation := range violations {
		stats[violation.Severity]++
	}

	response := StandardsCheckResult{
		CheckID:      fmt.Sprintf("standards-%d", time.Now().Unix()),
		Status:       "completed",
		ScanType:     checkRequest.Type,
		StartedAt:    startTime.Format(time.RFC3339),
		CompletedAt:  endTime.Format(time.RFC3339),
		Duration:     duration.Seconds(),
		FilesScanned: filesScanned,
		Violations:   violations,
		Statistics:   stats,
	}

	if scenarioName == "all" {
		response.Message = fmt.Sprintf("Standards compliance check completed across %d scenarios. Found %d violations.", scenarioCount, len(violations))
	} else {
		response.ScenarioName = scenarioName
		response.Message = fmt.Sprintf("Standards compliance check completed for %s. Found %d violations.", scenarioName, len(violations))
	}

	standardsStore.StoreViolations(scenarioName, violations)
	logger.Info(fmt.Sprintf("Stored %d standards violations in memory for %s", len(violations), scenarioName))

	logger.Info(fmt.Sprintf("Standards compliance check completed: %d violations found", len(violations)))
	json.NewEncoder(w).Encode(response)
}

func performStandardsCheck(scanPath, _ string, specificStandards []string) ([]StandardsViolation, int, error) {
	logger := NewLogger()

	ruleInfos, err := LoadRulesFromFiles()
	if err != nil {
		return nil, 0, err
	}

	ruleBuckets, activeRules := buildRuleBuckets(ruleInfos, specificStandards)
	if len(activeRules) == 0 {
		return nil, 0, nil
	}

	var violations []StandardsViolation
	filesScanned := 0

	err = filepath.Walk(scanPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return nil
		}

		if info.IsDir() {
			if shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		scenarioName, scenarioRelative, targets := classifyFileTargets(path)
		if len(targets) == 0 {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to read file %s", path), err)
			return nil
		}

		filesScanned++
		rulesForFile := collectRulesForTargets(targets, ruleBuckets)

		for _, rule := range rulesForFile {
			if rule.executor == nil || !rule.Implementation.Valid {
				continue
			}

			ruleViolations, execErr := rule.Check(string(content), path)
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

	return violations, filesScanned, err
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

	for id, info := range ruleInfos {
		if len(allowed) > 0 {
			if _, ok := allowed[id]; !ok {
				continue
			}
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

func convertRuleViolationToStandards(rule RuleInfo, violation Violation, scenarioName, fallbackPath string) StandardsViolation {
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
