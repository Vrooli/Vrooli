package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DuplicateResult contains code duplication analysis results
type DuplicateResult struct {
	TotalDuplicates int              `json:"total_duplicates"`
	DuplicateBlocks []DuplicateBlock `json:"duplicate_blocks"`
	TotalLines      int              `json:"total_lines"`
	Skipped         bool             `json:"skipped"`
	SkipReason      string           `json:"skip_reason,omitempty"`
	Tool            string           `json:"tool,omitempty"`
}

// DuplicateBlock represents a block of duplicated code
type DuplicateBlock struct {
	Files  []DuplicateLocation `json:"files"`
	Lines  int                 `json:"lines"`
	Tokens int                 `json:"tokens,omitempty"`
}

// DuplicateLocation represents a location where duplicate code appears
type DuplicateLocation struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// DuplicationDetector detects code duplication using external tools
type DuplicationDetector struct {
	scenarioPath string
	timeout      time.Duration
}

// NewDuplicationDetector creates a detector for the specified scenario
func NewDuplicationDetector(scenarioPath string, timeout time.Duration) *DuplicationDetector {
	if timeout == 0 {
		timeout = 90 * time.Second
	}
	return &DuplicationDetector{
		scenarioPath: scenarioPath,
		timeout:      timeout,
	}
}

// DetectDuplication runs duplication detection based on language
func (dd *DuplicationDetector) DetectDuplication(ctx context.Context, lang Language, files []string) (*DuplicateResult, error) {
	switch lang {
	case LanguageGo:
		return dd.detectGoDuplication(ctx, files)
	case LanguageTypeScript, LanguageJavaScript:
		return dd.detectJSDuplication(ctx, files)
	default:
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: fmt.Sprintf("duplication detection not implemented for %s", lang),
		}, nil
	}
}

// detectGoDuplication uses dupl to detect Go code duplication
func (dd *DuplicationDetector) detectGoDuplication(ctx context.Context, files []string) (*DuplicateResult, error) {
	// Check if dupl is installed
	if !commandExists("dupl") {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: "dupl not installed (install with: go install github.com/mibk/dupl@latest)",
		}, nil
	}

	if len(files) == 0 {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: "no files to analyze",
		}, nil
	}

	// Convert relative paths to absolute paths
	absPaths := make([]string, len(files))
	for i, relPath := range files {
		absPaths[i] = filepath.Join(dd.scenarioPath, relPath)
	}

	// Run dupl with threshold (15 tokens minimum for significant duplication)
	cmdCtx, cancel := context.WithTimeout(ctx, dd.timeout)
	defer cancel()

	// dupl -t 15 <files...>
	args := append([]string{"-t", "15"}, absPaths...)
	cmd := exec.CommandContext(cmdCtx, "dupl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// dupl returns non-zero if duplicates found
	_ = cmd.Run()

	// Parse output
	output := stdout.String()
	blocks := dd.parseDuplOutput(output)

	totalLines := 0
	for _, block := range blocks {
		totalLines += block.Lines
	}

	return &DuplicateResult{
		TotalDuplicates: len(blocks),
		DuplicateBlocks: blocks,
		TotalLines:      totalLines,
		Skipped:         false,
		Tool:            "dupl",
	}, nil
}

// parseDuplOutput parses dupl output into structured blocks
// Format:
// <file>:<start>-<end>
// <file>:<start>-<end>
// found <N> clones
func (dd *DuplicationDetector) parseDuplOutput(output string) []DuplicateBlock {
	var blocks []DuplicateBlock
	var currentBlock DuplicateBlock
	var currentLocations []DuplicateLocation

	// Regex for line like: /path/to/file.go:10-25
	locationRe := regexp.MustCompile(`^(.+?):(\d+)-(\d+)`)
	// Regex for summary like: found 2 clones
	summaryRe := regexp.MustCompile(`^found (\d+) clones`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line separates duplicate blocks
			if len(currentLocations) > 0 {
				currentBlock.Files = currentLocations
				blocks = append(blocks, currentBlock)
				currentBlock = DuplicateBlock{}
				currentLocations = []DuplicateLocation{}
			}
			continue
		}

		// Check for location line
		if matches := locationRe.FindStringSubmatch(line); len(matches) == 4 {
			filePath := matches[1]
			startLine, _ := strconv.Atoi(matches[2])
			endLine, _ := strconv.Atoi(matches[3])

			// Convert to relative path
			relPath, err := filepath.Rel(dd.scenarioPath, filePath)
			if err != nil {
				relPath = filePath
			}

			currentLocations = append(currentLocations, DuplicateLocation{
				Path:      relPath,
				StartLine: startLine,
				EndLine:   endLine,
			})

			// Calculate lines (if first location in block)
			if currentBlock.Lines == 0 {
				currentBlock.Lines = endLine - startLine + 1
			}
			continue
		}

		// Check for summary line (ignore for now, we count blocks ourselves)
		if summaryRe.MatchString(line) {
			continue
		}
	}

	// Add final block if exists
	if len(currentLocations) > 0 {
		currentBlock.Files = currentLocations
		blocks = append(blocks, currentBlock)
	}

	return blocks
}

// detectJSDuplication uses jscpd to detect JavaScript/TypeScript duplication
func (dd *DuplicationDetector) detectJSDuplication(ctx context.Context, files []string) (*DuplicateResult, error) {
	// Check if npx is available (for running jscpd)
	if !commandExists("npx") {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: "npx not installed (required for jscpd)",
		}, nil
	}

	if len(files) == 0 {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: "no files to analyze",
		}, nil
	}

	// Run jscpd with JSON output
	cmdCtx, cancel := context.WithTimeout(ctx, dd.timeout)
	defer cancel()

	// jscpd --reporters json --output /tmp <dir>
	// For simplicity, analyze the entire ui/src directory if TypeScript files present
	uiSrcPath := filepath.Join(dd.scenarioPath, "ui", "src")

	cmd := exec.CommandContext(cmdCtx, "npx", "jscpd", "--reporters", "json", "--min-lines", "5", "--min-tokens", "50", "--silent", uiSrcPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// jscpd may exit with non-zero if duplicates found or if it's not installed
	// Check stderr for "command not found" type errors
	stderrStr := stderr.String()
	if strings.Contains(stderrStr, "not found") || strings.Contains(stderrStr, "command not found") {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: "jscpd not available (install with: npm install -g jscpd)",
		}, nil
	}

	// If no output and error, skip
	if err != nil && stdout.Len() == 0 {
		return &DuplicateResult{
			Skipped:    true,
			SkipReason: fmt.Sprintf("jscpd failed: %v", err),
		}, nil
	}

	// Parse JSON output
	output := stdout.String()
	blocks := dd.parseJscpdOutput(output)

	totalLines := 0
	for _, block := range blocks {
		totalLines += block.Lines
	}

	return &DuplicateResult{
		TotalDuplicates: len(blocks),
		DuplicateBlocks: blocks,
		TotalLines:      totalLines,
		Skipped:         false,
		Tool:            "jscpd",
	}, nil
}

// parseJscpdOutput parses jscpd JSON output
func (dd *DuplicationDetector) parseJscpdOutput(output string) []DuplicateBlock {
	// jscpd outputs complex JSON - for now, return empty if we can't parse
	// A full implementation would parse the jscpd JSON structure
	var result struct {
		Duplicates []struct {
			FirstFile struct {
				Name  string `json:"name"`
				Start int    `json:"start"`
				End   int    `json:"end"`
			} `json:"firstFile"`
			SecondFile struct {
				Name  string `json:"name"`
				Start int    `json:"start"`
				End   int    `json:"end"`
			} `json:"secondFile"`
			Lines  int `json:"lines"`
			Tokens int `json:"tokens"`
		} `json:"duplicates"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// If JSON parsing fails, return empty
		return []DuplicateBlock{}
	}

	var blocks []DuplicateBlock
	for _, dup := range result.Duplicates {
		relPath1, _ := filepath.Rel(dd.scenarioPath, dup.FirstFile.Name)
		relPath2, _ := filepath.Rel(dd.scenarioPath, dup.SecondFile.Name)

		block := DuplicateBlock{
			Files: []DuplicateLocation{
				{
					Path:      relPath1,
					StartLine: dup.FirstFile.Start,
					EndLine:   dup.FirstFile.End,
				},
				{
					Path:      relPath2,
					StartLine: dup.SecondFile.Start,
					EndLine:   dup.SecondFile.End,
				},
			},
			Lines:  dup.Lines,
			Tokens: dup.Tokens,
		}
		blocks = append(blocks, block)
	}

	return blocks
}
