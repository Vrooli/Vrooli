package fleet

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"experience-manager/internal/spec"
)

// Scenario summarizes one scenario's experience depth and parser debt.
type Scenario struct {
	Scenario      string
	HasExperience bool
	MaxDepth      int
	PageCount     int
	FindingCount  int
	DebtScore     int
	Status        string
}

// Summary is the compute-on-read fleet view.
type Summary struct {
	Scenarios           []Scenario
	ScenarioCount       int
	WithExperienceCount int
	TotalPages          int
}

// Sweep computes fleet coverage from the filesystem without persisted state.
func Sweep(_ context.Context, repoRoot string) (Summary, error) {
	root := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(root)
	if err != nil {
		return Summary{}, err
	}
	var out Summary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		scenarioDir := filepath.Join(root, name)
		if _, err := os.Stat(filepath.Join(scenarioDir, ".vrooli", "service.json")); err != nil {
			continue
		}
		out.ScenarioCount++
		row := Scenario{Scenario: name, Status: "no experience"}
		if info, err := os.Stat(filepath.Join(scenarioDir, "experience")); err == nil && info.IsDir() {
			row.HasExperience = true
			out.WithExperienceCount++
			report, parseErr := spec.ParseScenario(scenarioDir)
			if parseErr != nil {
				row.FindingCount = 1
				row.DebtScore = 100
				row.Status = "parse error"
			} else {
				row.PageCount = len(report.PageDepths)
				row.FindingCount = len(report.Findings)
				row.MaxDepth = maxDepth(report.PageDepths)
				row.DebtScore = debtScore(row.MaxDepth, row.FindingCount)
				row.Status = status(row)
			}
		} else {
			row.DebtScore = 50
		}
		out.TotalPages += row.PageCount
		out.Scenarios = append(out.Scenarios, row)
	}
	sort.Slice(out.Scenarios, func(i, j int) bool {
		if out.Scenarios[i].DebtScore == out.Scenarios[j].DebtScore {
			return out.Scenarios[i].Scenario < out.Scenarios[j].Scenario
		}
		return out.Scenarios[i].DebtScore > out.Scenarios[j].DebtScore
	})
	return out, nil
}

func maxDepth(depths map[string]int) int {
	max := 0
	for _, depth := range depths {
		if depth > max {
			max = depth
		}
	}
	return max
}

func debtScore(depth, findings int) int {
	score := (4 - depth) * 10
	if score < 0 {
		score = 0
	}
	return score + findings*5
}

func status(row Scenario) string {
	if row.FindingCount > 0 {
		return "findings"
	}
	if row.PageCount == 0 {
		return "empty"
	}
	return "green"
}
