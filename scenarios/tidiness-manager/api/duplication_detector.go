package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	debt "tidiness-manager/internal/discovery"
)

// DuplicateResult adapts the shared body-identity detector to scan responses.
type DuplicateResult struct {
	TotalDuplicates int              `json:"total_duplicates"`
	DuplicateBlocks []DuplicateBlock `json:"duplicate_blocks"`
	TotalLines      int              `json:"total_lines"`
	Skipped         bool             `json:"skipped"`
	SkipReason      string           `json:"skip_reason,omitempty"`
	Tool            string           `json:"tool,omitempty"`
}

// DuplicateMinimumLines is the physical-line floor used by both scan responses
// and ranked refactor recommendations.
const DuplicateMinimumLines = 5

type DuplicateBlock struct {
	Files []DuplicateLocation `json:"files"`
	Lines int                 `json:"lines"`
}

type DuplicateLocation struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// DuplicationDetector projects Code Facts evidence into the legacy scan shape;
// source parsing and duplicate grouping have one owner in Code Facts and the
// discovery registry.
type DuplicationDetector struct {
	scenarioPath string
	timeout      time.Duration
	source       ReductionFacts
}

func NewDuplicationDetector(scenarioPath string, timeout time.Duration) *DuplicationDetector {
	if timeout == 0 {
		timeout = 90 * time.Second
	}
	return &DuplicationDetector{scenarioPath: scenarioPath, timeout: timeout}
}

func (dd *DuplicationDetector) DetectDuplication(ctx context.Context, lang Language, files []string) (*DuplicateResult, error) {
	switch lang {
	case LanguageGo:
		return dd.detectGraphDuplication(ctx, files, "go", []string{"go"})
	case LanguageTypeScript:
		return dd.detectGraphDuplication(ctx, files, "typescript", []string{"typescript"})
	case LanguageJavaScript:
		// JavaScript in an established TypeScript project is emitted by that
		// project graph; standalone JavaScript packages have their own Code Facts
		// parse unit routed to the same TypeScript AST provider.
		return dd.detectGraphDuplication(ctx, files, "javascript", []string{"javascript", "typescript"})
	default:
		return &DuplicateResult{Skipped: true, SkipReason: fmt.Sprintf("duplication detection not implemented for %s", lang)}, nil
	}
}

func (dd *DuplicationDetector) detectGraphDuplication(ctx context.Context, files []string, language string, languageFilter []string) (*DuplicateResult, error) {
	if len(files) == 0 {
		return &DuplicateResult{Skipped: true, SkipReason: "no files to analyze"}, nil
	}
	if _, err := resolveScenarioFiles(dd.scenarioPath, files); err != nil {
		return &DuplicateResult{Skipped: true, SkipReason: err.Error()}, nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, dd.timeout)
	defer cancel()
	report, err := currentCodeFacts(cmdCtx, dd.source, &facts.CodeTarget{Kind: facts.TargetKind_TARGET_KIND_PATH, Path: dd.scenarioPath, LanguageFilter: languageFilter})
	if err != nil {
		return &DuplicateResult{Skipped: true, SkipReason: err.Error(), Tool: "code-facts"}, nil
	}
	return dd.DetectDuplicationFromFacts(files, language, report)
}

func (dd *DuplicationDetector) DetectDuplicationFromFacts(files []string, language string, report *facts.CodeFactsReport) (*DuplicateResult, error) {
	return dd.DetectDuplicationForLanguagesFromFacts(map[string][]string{language: files}, report)[language], nil
}

// DetectDuplicationForLanguagesFromFacts projects all language groups from one
// graph traversal, allowing multi-language scans to reuse shared symbols.
func (dd *DuplicationDetector) DetectDuplicationForLanguagesFromFacts(filesByLanguage map[string][]string, report *facts.CodeFactsReport) map[string]*DuplicateResult {
	groupsByLanguage, warnings, evidence := debt.DuplicateFunctionBodiesByLanguageWithEvidence(report, DuplicateMinimumLines)
	results := make(map[string]*DuplicateResult, len(filesByLanguage))
	for language, files := range filesByLanguage {
		if len(files) == 0 {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: "no files to analyze"}
			continue
		}
		if _, err := resolveScenarioFiles(dd.scenarioPath, files); err != nil {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: err.Error()}
			continue
		}

		allowed := make(map[string]bool, len(files))
		for _, file := range files {
			allowed[filepath.ToSlash(filepath.Clean(file))] = true
		}
		hasEvidence := false
		for path := range allowed {
			if evidence[path][language] {
				hasEvidence = true
				break
			}
		}
		if !hasEvidence {
			results[language] = &DuplicateResult{Skipped: true, SkipReason: "Code Facts returned no " + language + " source evidence for the requested files", Tool: "code-facts"}
			continue
		}
		groups := groupsByLanguage[language]
		blocks := make([]DuplicateBlock, 0, len(groups))
		totalLines := 0
		for _, group := range groups {
			block := DuplicateBlock{Lines: group.Lines}
			for _, location := range group.Locations {
				if allowed[filepath.ToSlash(filepath.Clean(location.Path))] {
					block.Files = append(block.Files, DuplicateLocation{Path: location.Path, StartLine: location.StartLine, EndLine: location.EndLine})
				}
			}
			if len(block.Files) > 1 {
				blocks = append(blocks, block)
				totalLines += block.Lines
			}
		}

		results[language] = &DuplicateResult{
			TotalDuplicates: len(blocks),
			DuplicateBlocks: blocks,
			TotalLines:      totalLines,
			Skipped:         len(warnings) > 0,
			SkipReason:      strings.Join(warnings, "; "),
			Tool:            "code-facts",
		}
	}
	return results
}
