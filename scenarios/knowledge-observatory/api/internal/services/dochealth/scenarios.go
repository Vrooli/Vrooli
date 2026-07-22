package dochealth

import (
	"context"
	"knowledge-observatory/internal/docvalidation"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScenarioSummary describes documentation status for a scenario.
type ScenarioSummary struct {
	Name         string
	Path         string
	DocCount     int
	HealthScore  float64
	HasManifest  bool
	HasReadme    bool
	LastModified time.Time
}

// ListScenarios returns documentation summary information for each scenario.
func (s *Service) ListScenarios(ctx context.Context) ([]ScenarioSummary, error) {
	_ = ctx
	if s == nil {
		return nil, ErrScenarioRootInvalid
	}
	entries, err := os.ReadDir(s.scenariosRoot)
	if err != nil {
		return nil, err
	}
	var summaries []ScenarioSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(s.scenariosRoot, name)
		stats, err := collectScenarioStats(path)
		if err != nil {
			continue
		}
		healthScore := 0.0
		if validation, err := docvalidation.ValidateScenarioDocumentation(path); err == nil {
			healthScore = validation.HealthScore
		}
		summaries = append(summaries, ScenarioSummary{
			Name:         name,
			Path:         path,
			DocCount:     stats.count,
			HealthScore:  healthScore,
			HasManifest:  stats.hasManifest,
			HasReadme:    stats.hasReadme,
			LastModified: stats.lastModified,
		})
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries, nil
}

func isDocFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".md" || ext == ".json"
}

type docStats struct {
	count        int
	hasManifest  bool
	hasReadme    bool
	lastModified time.Time
}

func collectScenarioStats(scenarioPath string) (docStats, error) {
	stats := docStats{}
	rootFiles := []string{"README.md", "PRD.md"}
	for _, name := range rootFiles {
		path := filepath.Join(scenarioPath, name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		stats.count++
		if name == "README.md" {
			stats.hasReadme = true
		}
		if info.ModTime().After(stats.lastModified) {
			stats.lastModified = info.ModTime()
		}
	}
	docsRoot := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsRoot); err == nil && info.IsDir() {
		_ = filepath.WalkDir(docsRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !isDocFile(path) {
				return nil
			}
			stats.count++
			if filepath.Base(path) == "manifest.json" {
				stats.hasManifest = true
			}
			if info, err := os.Stat(path); err == nil {
				if info.ModTime().After(stats.lastModified) {
					stats.lastModified = info.ModTime()
				}
			}
			return nil
		})
	}
	return stats, nil
}
