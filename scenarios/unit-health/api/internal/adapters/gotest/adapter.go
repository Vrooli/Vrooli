package gotest

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"unit-health/internal/adapters"
	"unit-health/internal/testquality"
)

type Analyzer struct{}

func (Analyzer) Identity() adapters.Identity       { return adapters.Identity{ID: "go", Version: "1.0.0"} }
func (Analyzer) Matches(match adapters.Match) bool { return strings.EqualFold(match.Language, "go") }
func (Analyzer) AnalyzeSource(ctx context.Context, input adapters.QualityInput) ([]testquality.Result, testquality.Reason) {
	rows, _, reason := (Analyzer{}).AnalyzeSourceEvidence(ctx, input)
	return rows, reason
}
func (Analyzer) AnalyzeSourceEvidence(ctx context.Context, input adapters.QualityInput) ([]testquality.Result, []testquality.TestLinks, testquality.Reason) {
	var files []string
	err := filepath.WalkDir(input.Root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "testdata", "coverage":
				if path != input.Root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			rel, err := filepath.Rel(input.Root, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
			if len(files) > 1000 {
				return fs.ErrInvalid
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, testquality.MissingInput
	}
	rows, links, err := AnalyzeEvidence(Input{Root: input.Root, Workspace: input.Workspace, Files: files, DefaultTestKind: input.TestKind})
	if err != nil {
		return nil, nil, testquality.BuildContextUnavailable
	}
	return rows, links, testquality.ReasonNone
}
