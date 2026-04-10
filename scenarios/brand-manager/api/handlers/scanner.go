// Package handlers - inline validation scanner for CSS markers and JSON _brand keys.
// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]
package handlers

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"brand-manager/domain"

	"github.com/gorilla/mux"
)

// cssMarkerRe matches CSS comments like /* brand-manager:primary */ or /* brand-manager:logo */
var cssMarkerRe = regexp.MustCompile(`/\*\s*brand-manager:(\S+)\s*\*/`)

// jsonBrandKeyRe matches JSON keys containing "_brand" prefix
var jsonBrandKeyRe = regexp.MustCompile(`"(_brand[^"]*)"`)

// skipDirs contains directory names to ignore during scenario walks.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true, "vendor": true,
}

// walkScenarioDir walks a scenario directory, calling fn for each regular file
// while skipping common non-source directories.
func walkScenarioDir(scenarioDir string, fn func(path, relPath, ext string)) {
	filepath.Walk(scenarioDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, _ := filepath.Rel(scenarioDir, path)
		fn(path, relPath, strings.ToLower(filepath.Ext(path)))
		return nil
	})
}

// scanFileWithRegex scans a file line-by-line, returning matches for the given regex and result type.
// The regex must have at least one capture group: m[0] is used as Marker, m[1] as Element.
func scanFileWithRegex(path, relPath, resultType string, re *regexp.Regexp) []domain.ScanResult {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var results []domain.ScanResult
	s := bufio.NewScanner(f)
	lineNum := 0
	for s.Scan() {
		lineNum++
		for _, m := range re.FindAllStringSubmatch(s.Text(), -1) {
			results = append(results, domain.ScanResult{
				File:    relPath,
				Line:    lineNum,
				Type:    resultType,
				Marker:  m[0],
				Element: m[1],
			})
		}
	}
	return results
}

// Convenience wrappers used by both the basic scanner and the plugin system.
func scanFileForCSS(path, relPath string) []domain.ScanResult {
	return scanFileWithRegex(path, relPath, "css", cssMarkerRe)
}

func scanFileForJSON(path, relPath string) []domain.ScanResult {
	return scanFileWithRegex(path, relPath, "json", jsonBrandKeyRe)
}

// ScanScenario handles GET /api/v1/scan/{scenario}. [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]
func (h *Handlers) ScanScenario(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario"]

	scenarioDir, done := h.resolveScenarioDir(w, scenario)
	if done {
		return
	}

	report := domain.ScanReport{Scenario: scenario}

	walkScenarioDir(scenarioDir, func(path, relPath, ext string) {
		var results []domain.ScanResult
		switch ext {
		case ".css", ".scss", ".less":
			results = scanFileForCSS(path, relPath)
			report.CSSMarkers += len(results)
		case ".json":
			results = scanFileForJSON(path, relPath)
			report.JSONKeys += len(results)
		}
		report.Results = append(report.Results, results...)
	})

	report.Total = report.CSSMarkers + report.JSONKeys
	writeJSON(w, http.StatusOK, report)
}
