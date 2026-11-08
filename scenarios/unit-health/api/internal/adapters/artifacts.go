package adapters

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CoverageMetric is the normalized, adapter-produced coverage unit. The
// validation kernel consumes this shape without knowing which runner emitted
// the artifact.
type CoverageMetric struct {
	Covered int64
	Total   int64
}

// Percent reports covered/total. An empty denominator is fully covered, the
// Istanbul convention: there is nothing in that metric left to exercise.
func (m CoverageMetric) Percent() float64 {
	if m.Total == 0 {
		return 100
	}
	return float64(m.Covered) / float64(m.Total) * 100
}

// Coverage metric names shared by artifact readers and coverage commitments in
// `unit.policy_profile`. Not every artifact reports every metric: Go cover
// profiles measure statements, LCOV and Cobertura measure lines.
const (
	MetricLines      = "lines"
	MetricStatements = "statements"
	MetricFunctions  = "functions"
	MetricBranches   = "branches"
)

// CoverageMetricNames lists the metrics a coverage commitment may name, in
// reporting order.
var CoverageMetricNames = []string{MetricLines, MetricStatements, MetricFunctions, MetricBranches}

// CoverageReport is one parsed coverage artifact: per-file units (lines, or
// statements for Go) for file-level targets, and workspace totals per metric
// for committed-coverage gates.
type CoverageReport struct {
	Files  map[string]CoverageMetric
	Totals map[string]CoverageMetric
}

func DefaultCoverageArtifacts(language string) []Artifact {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go":
		return []Artifact{{Kind: "go-cover-profile", Path: "coverage.out"}}
	case "python":
		return []Artifact{{Kind: "cobertura", Path: "coverage.xml"}}
	case "rust":
		return []Artifact{{Kind: "lcov", Path: filepath.Join("coverage", "lcov.info")}}
	case "typescript", "javascript", "node":
		return []Artifact{
			{Kind: "istanbul-summary", Path: filepath.Join("coverage", "coverage-summary.json")},
			{Kind: "lcov", Path: filepath.Join("coverage", "lcov.info")},
		}
	default:
		return nil
	}
}

const (
	maxCoverageArtifactBytes = 32 * 1024 * 1024
	maxCoverageFiles         = 100_000
	maxCoverageRecords       = 1_000_000
)

// ReadCoverage reads the first complete declared coverage artifact. Artifact
// kinds are owned by adapters; an unknown kind is ignored rather than treated
// as a different framework's format.
func ReadCoverage(root string, artifacts []Artifact) (CoverageReport, bool) {
	for _, artifact := range artifacts {
		path := artifact.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		var report CoverageReport
		var ok bool
		switch strings.ToLower(strings.TrimSpace(artifact.Kind)) {
		case "go-cover-profile":
			report.Files, ok = readGoCoverProfile(path)
			report.Totals = aggregateTotals(MetricStatements, report.Files)
		case "istanbul-summary":
			report, ok = readIstanbulSummary(path)
		case "lcov":
			report.Files, ok = readLCOV(path)
			report.Totals = aggregateTotals(MetricLines, report.Files)
		case "cobertura":
			report.Files, ok = readCobertura(path)
			report.Totals = aggregateTotals(MetricLines, report.Files)
		}
		if ok {
			return report, true
		}
	}
	return CoverageReport{}, false
}

// aggregateTotals sums per-file units into the one metric the artifact kind
// measures.
func aggregateTotals(metric string, files map[string]CoverageMetric) map[string]CoverageMetric {
	var total CoverageMetric
	for _, file := range files {
		total.Covered += file.Covered
		total.Total += file.Total
	}
	return map[string]CoverageMetric{metric: total}
}

func boundedFile(path string) ([]byte, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxCoverageArtifactBytes+1))
	if err != nil || len(data) > maxCoverageArtifactBytes {
		return nil, false
	}
	return data, true
}

func readGoCoverProfile(path string) (map[string]CoverageMetric, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	coverage := map[string]CoverageMetric{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 4*1024*1024)
	records := 0
	sawMode := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "mode:") {
			if sawMode || strings.TrimSpace(strings.TrimPrefix(line, "mode:")) == "" {
				return nil, false
			}
			sawMode = true
			continue
		}
		if records >= maxCoverageRecords {
			return nil, false
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, false
		}
		separator := strings.LastIndex(fields[0], ":")
		if separator <= 0 {
			return nil, false
		}
		statements, errStatements := strconv.ParseInt(fields[1], 10, 64)
		count, errCount := strconv.ParseInt(fields[2], 10, 64)
		if errStatements != nil || errCount != nil || statements < 0 || count < 0 {
			return nil, false
		}
		name := fields[0][:separator]
		metric := coverage[name]
		metric.Total += statements
		if count > 0 {
			metric.Covered += statements
		}
		coverage[name] = metric
		records++
	}
	if scanner.Err() != nil || !sawMode || records == 0 {
		return nil, false
	}
	return coverage, true
}

type istanbulCount struct {
	Total   int64 `json:"total"`
	Covered int64 `json:"covered"`
}

func (c istanbulCount) valid() bool { return c.Total >= 0 && c.Covered >= 0 && c.Covered <= c.Total }

type istanbulSummaryEntry struct {
	Lines      istanbulCount  `json:"lines"`
	Statements *istanbulCount `json:"statements"`
	Functions  *istanbulCount `json:"functions"`
	Branches   *istanbulCount `json:"branches"`
}

// metrics returns only the metrics the summary actually reports, so an absent
// metric is never mistaken for an empty (fully covered) one.
func (e istanbulSummaryEntry) metrics() map[string]istanbulCount {
	out := map[string]istanbulCount{MetricLines: e.Lines}
	for name, count := range map[string]*istanbulCount{MetricStatements: e.Statements, MetricFunctions: e.Functions, MetricBranches: e.Branches} {
		if count != nil {
			out[name] = *count
		}
	}
	return out
}

// readIstanbulSummary reads Istanbul's json-summary: per-file line units and,
// from its `total` entry, all four workspace metrics. A summary without a
// `total` entry falls back to aggregated lines.
func readIstanbulSummary(path string) (CoverageReport, bool) {
	data, ok := boundedFile(path)
	if !ok {
		return CoverageReport{}, false
	}
	var summary map[string]istanbulSummaryEntry
	if json.Unmarshal(data, &summary) != nil || len(summary) > maxCoverageFiles {
		return CoverageReport{}, false
	}
	report := CoverageReport{Files: make(map[string]CoverageMetric, len(summary))}
	var total *istanbulSummaryEntry
	for name, entry := range summary {
		if strings.EqualFold(name, "total") {
			entry := entry
			total = &entry
			continue
		}
		if !entry.Lines.valid() {
			return CoverageReport{}, false
		}
		report.Files[name] = CoverageMetric{Covered: entry.Lines.Covered, Total: entry.Lines.Total}
	}
	if len(report.Files) == 0 {
		return CoverageReport{}, false
	}
	if total == nil {
		report.Totals = aggregateTotals(MetricLines, report.Files)
		return report, true
	}
	report.Totals = make(map[string]CoverageMetric, len(CoverageMetricNames))
	for metric, count := range total.metrics() {
		if !count.valid() {
			return CoverageReport{}, false
		}
		report.Totals[metric] = CoverageMetric{Covered: count.Covered, Total: count.Total}
	}
	return report, true
}

func readLCOV(path string) (map[string]CoverageMetric, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	coverage := map[string]CoverageMetric{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 4*1024*1024)
	var name string
	var metric CoverageMetric
	var sawTotal, sawCovered bool
	records := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case strings.HasPrefix(line, "SF:"):
			if name != "" || strings.TrimSpace(strings.TrimPrefix(line, "SF:")) == "" {
				return nil, false
			}
			name = strings.TrimPrefix(line, "SF:")
			metric = CoverageMetric{}
			sawTotal, sawCovered = false, false
		case strings.HasPrefix(line, "LF:"):
			if name == "" || sawTotal {
				return nil, false
			}
			parsed, ok := parseCoverageInt(strings.TrimPrefix(line, "LF:"))
			if !ok {
				return nil, false
			}
			metric.Total, sawTotal = parsed, true
		case strings.HasPrefix(line, "LH:"):
			if name == "" || sawCovered {
				return nil, false
			}
			parsed, ok := parseCoverageInt(strings.TrimPrefix(line, "LH:"))
			if !ok {
				return nil, false
			}
			metric.Covered, sawCovered = parsed, true
		case line == "end_of_record":
			if name == "" || !sawTotal || !sawCovered || metric.Covered > metric.Total {
				return nil, false
			}
			if records >= maxCoverageFiles {
				return nil, false
			}
			coverage[name] = metric
			records++
			name = ""
			sawTotal, sawCovered = false, false
		}
	}
	if scanner.Err() != nil || name != "" || len(coverage) == 0 {
		return nil, false
	}
	return coverage, true
}

type coberturaReport struct {
	XMLName  xml.Name `xml:"coverage"`
	Packages []struct {
		Classes []struct {
			Filename string `xml:"filename,attr"`
			Lines    []struct {
				Number int64 `xml:"number,attr"`
				Hits   int64 `xml:"hits,attr"`
			} `xml:"lines>line"`
		} `xml:"classes>class"`
	} `xml:"packages>package"`
}

func readCobertura(path string) (map[string]CoverageMetric, bool) {
	data, ok := boundedFile(path)
	if !ok {
		return nil, false
	}
	var report coberturaReport
	if xml.Unmarshal(data, &report) != nil {
		return nil, false
	}
	coverage := map[string]CoverageMetric{}
	records := 0
	for _, pkg := range report.Packages {
		for _, class := range pkg.Classes {
			if strings.TrimSpace(class.Filename) == "" {
				return nil, false
			}
			if records >= maxCoverageFiles {
				return nil, false
			}
			metric := coverage[class.Filename]
			for _, line := range class.Lines {
				if line.Number < 0 || line.Hits < 0 {
					return nil, false
				}
				metric.Total++
				if line.Hits > 0 {
					metric.Covered++
				}
			}
			if metric.Total > 0 {
				coverage[class.Filename] = metric
				records++
			}
		}
	}
	if len(coverage) == 0 {
		return nil, false
	}
	return coverage, true
}

func parseCoverageInt(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed, true
}
