package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultSummaryLimit = 20
	maxSummaryBuffer    = 200
	scanArtifactTTL     = 14 * 24 * time.Hour
	scanArtifactRootDir = "logs/scenario-auditor"
)

var severityWeights = map[string]int{
	"critical": 5,
	"high":     4,
	"medium":   3,
	"low":      2,
	"info":     1,
}

var normalizedSeverities = []string{"critical", "high", "medium", "low", "info"}

type RuleCount struct {
	RuleID         string `json:"rule_id"`
	Count          int    `json:"count"`
	Title          string `json:"title,omitempty"`
	Severity       string `json:"severity,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type ViolationExcerpt struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	RuleID         string `json:"rule_id,omitempty"`
	Title          string `json:"title,omitempty"`
	FilePath       string `json:"file_path,omitempty"`
	LineNumber     int    `json:"line_number,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	Source         string `json:"source,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type ScanArtifactRef struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type ViolationSummary struct {
	Total            int                `json:"total"`
	BySeverity       map[string]int     `json:"by_severity"`
	ByRule           []RuleCount        `json:"by_rule"`
	HighestSeverity  string             `json:"highest_severity"`
	TopViolations    []ViolationExcerpt `json:"top_violations"`
	Artifact         *ScanArtifactRef   `json:"artifact,omitempty"`
	RecommendedSteps []string           `json:"recommended_steps,omitempty"`
	GeneratedAt      string             `json:"generated_at"`
}

type violationRecord struct {
	ID             string
	Severity       string
	RuleID         string
	Title          string
	FilePath       string
	LineNumber     int
	Scenario       string
	Source         string
	Recommendation string
}

func buildViolationSummary(records []violationRecord, limit int) ViolationSummary {
	summary := ViolationSummary{
		Total:       len(records),
		BySeverity:  make(map[string]int),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if limit <= 0 {
		limit = defaultSummaryLimit
	}

	ruleAggregation := make(map[string]*RuleCount)
	highestRank := -1

	for _, rec := range records {
		sev := normalizeSeverity(rec.Severity)
		summary.BySeverity[sev]++

		if rec.RuleID != "" {
			rc, ok := ruleAggregation[rec.RuleID]
			if !ok {
				rc = &RuleCount{RuleID: rec.RuleID, Title: rec.Title, Recommendation: rec.Recommendation, Severity: sev}
				ruleAggregation[rec.RuleID] = rc
			}
			rc.Count++
			if severityWeights[sev] > severityWeights[rc.Severity] {
				rc.Severity = sev
			}
		}

		if rank := severityWeights[sev]; rank > highestRank {
			highestRank = rank
			summary.HighestSeverity = sev
		}
	}

	if len(ruleAggregation) > 0 {
		summary.ByRule = make([]RuleCount, 0, len(ruleAggregation))
		for _, rc := range ruleAggregation {
			summary.ByRule = append(summary.ByRule, *rc)
		}
		sort.Slice(summary.ByRule, func(i, j int) bool {
			if summary.ByRule[i].Count == summary.ByRule[j].Count {
				return summary.ByRule[i].RuleID < summary.ByRule[j].RuleID
			}
			return summary.ByRule[i].Count > summary.ByRule[j].Count
		})
	}

	summary.TopViolations = selectTopViolations(records, limit)
	summary.RecommendedSteps = buildRecommendedSteps(&summary)

	return summary
}

func selectTopViolations(records []violationRecord, limit int) []ViolationExcerpt {
	if len(records) == 0 || limit <= 0 {
		return nil
	}

	cappedLimit := limit
	if cappedLimit > maxSummaryBuffer {
		cappedLimit = maxSummaryBuffer
	}

	snapshot := make([]violationRecord, len(records))
	copy(snapshot, records)

	sort.Slice(snapshot, func(i, j int) bool {
		si := severityWeights[normalizeSeverity(snapshot[i].Severity)]
		sj := severityWeights[normalizeSeverity(snapshot[j].Severity)]
		if si != sj {
			return si > sj
		}
		if snapshot[i].Scenario != snapshot[j].Scenario {
			return snapshot[i].Scenario < snapshot[j].Scenario
		}
		if snapshot[i].FilePath != snapshot[j].FilePath {
			return snapshot[i].FilePath < snapshot[j].FilePath
		}
		if snapshot[i].LineNumber != snapshot[j].LineNumber {
			return snapshot[i].LineNumber < snapshot[j].LineNumber
		}
		return snapshot[i].ID < snapshot[j].ID
	})

	limitIndex := cappedLimit
	if len(snapshot) < cappedLimit {
		limitIndex = len(snapshot)
	}

	excerpts := make([]ViolationExcerpt, 0, limitIndex)
	for _, rec := range snapshot[:limitIndex] {
		excerpts = append(excerpts, ViolationExcerpt{
			ID:             rec.ID,
			Severity:       normalizeSeverity(rec.Severity),
			RuleID:         rec.RuleID,
			Title:          rec.Title,
			FilePath:       rec.FilePath,
			LineNumber:     rec.LineNumber,
			Scenario:       rec.Scenario,
			Source:         rec.Source,
			Recommendation: rec.Recommendation,
		})
	}

	return excerpts
}

func buildRecommendedSteps(summary *ViolationSummary) []string {
	if summary == nil {
		return nil
	}

	if summary.Total == 0 {
		return []string{"No violations detected. Document the clean run and rerun scenario-auditor after future changes."}
	}

	steps := []string{}
	if summary.HighestSeverity != "" {
		steps = append(steps, fmt.Sprintf("Address %s findings first (see Top Violations).", titleCase(summary.HighestSeverity)))
	}
	if summary.Artifact != nil {
		steps = append(steps, fmt.Sprintf("Review %s for the full list of %d violations before re-running the audit.", summary.Artifact.Path, summary.Total))
	}
	steps = append(steps, "Re-run scenario-auditor audit after applying fixes to validate remediation.")
	return steps
}

func normalizeSeverity(sev string) string {
	s := strings.ToLower(strings.TrimSpace(sev))
	if _, ok := severityWeights[s]; ok {
		return s
	}
	return "info"
}

func titleCase(value string) string {
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

func cloneSummary(summary *ViolationSummary, limit int, minSeverity string) *ViolationSummary {
	if summary == nil {
		return nil
	}

	clone := *summary
	if summary.Artifact != nil {
		artifact := *summary.Artifact
		clone.Artifact = &artifact
	}
	if summary.ByRule != nil {
		clone.ByRule = append([]RuleCount(nil), summary.ByRule...)
	}
	if summary.BySeverity != nil {
		clone.BySeverity = make(map[string]int, len(summary.BySeverity))
		for key, val := range summary.BySeverity {
			clone.BySeverity[key] = val
		}
	}
	if summary.RecommendedSteps != nil {
		clone.RecommendedSteps = append([]string(nil), summary.RecommendedSteps...)
	}

	clone.TopViolations = filterSummaryViolations(summary.TopViolations, limit, minSeverity)
	return &clone
}

func filterSummaryViolations(violations []ViolationExcerpt, limit int, minSeverity string) []ViolationExcerpt {
	if len(violations) == 0 {
		return nil
	}

	threshold := severityWeights[normalizeSeverity(minSeverity)]
	if threshold == 0 {
		threshold = 1
	}

	filtered := make([]ViolationExcerpt, 0, len(violations))
	for _, v := range violations {
		if severityWeights[normalizeSeverity(v.Severity)] < threshold {
			continue
		}
		filtered = append(filtered, v)
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}

	return filtered
}

func persistScanArtifact(scanType, scenarioName, jobID string, payload any) (*ScanArtifactRef, error) {
	root := getVrooliRoot()
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		root = cwd
	}

	dir := filepath.Join(root, scanArtifactRootDir, scanType, sanitizePathComponent(scenarioName))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%s_job-%s.json", time.Now().UTC().Format("20060102-150405"), jobID)
	fullPath := filepath.Join(dir, filename)

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, err
	}

	checksum := sha256.Sum256(data)
	relPath, err := filepath.Rel(root, fullPath)
	if err != nil {
		relPath = fullPath
	}

	cleanupOldArtifacts(dir)

	return &ScanArtifactRef{
		Path:      filepath.ToSlash(relPath),
		Checksum:  hex.EncodeToString(checksum[:]),
		SizeBytes: int64(len(data)),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func resolveArtifactAbsolutePath(relPath string) (string, error) {
	if relPath == "" {
		return "", fmt.Errorf("artifact path missing")
	}

	root := getVrooliRoot()
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		root = cwd
	}

	cleanRoot := filepath.Clean(root)
	var full string
	if filepath.IsAbs(relPath) {
		full = filepath.Clean(relPath)
	} else {
		full = filepath.Clean(filepath.Join(cleanRoot, relPath))
		if rel, err := filepath.Rel(cleanRoot, full); err != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("artifact path escapes Vrooli root")
		}
	}

	return full, nil
}

func cleanupOldArtifacts(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-scanArtifactTTL)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.IsDir() {
			cleanupOldArtifacts(filepath.Join(dir, entry.Name()))
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

func sanitizePathComponent(value string) string {
	if value == "" {
		return "unknown"
	}
	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r + 32
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-' // replace unsupported chars
		}
	}, value)
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "unknown"
	}
	return sanitized
}
