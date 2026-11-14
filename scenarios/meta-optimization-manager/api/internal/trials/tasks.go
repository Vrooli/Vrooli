package trials

import (
	"context"
	"sort"
	"strings"

	"github.com/vrooli/api-core/spacedoc"
)

// SpaceReader reads the Guide projection's denominator. Declared at the consumer
// (seam-discovery); production wires the coverage domain's exec+file-fallback
// reader (identical method set), tests fake it.
type SpaceReader interface {
	Read(ctx context.Context, p spacedoc.Projection) (*spacedoc.SpaceDefinition, error)
}

// TaskGenerator builds the trial suite from the Guide space: one positive task
// per Guide row (suite derived from the row's category) plus one negative /
// honesty task per suite family. The suite is generated, never stored — it
// tracks the Guide denominator automatically.
type TaskGenerator interface {
	Generate(ctx context.Context, suite string) ([]TrialTask, error)
}

type spaceTaskGenerator struct {
	reader SpaceReader
}

// NewTaskGenerator constructs the production TaskGenerator over a SpaceReader.
func NewTaskGenerator(r SpaceReader) TaskGenerator { return &spaceTaskGenerator{reader: r} }

var _ TaskGenerator = (*spaceTaskGenerator)(nil)

func (g *spaceTaskGenerator) Generate(ctx context.Context, suite string) ([]TrialTask, error) {
	def, err := g.reader.Read(ctx, spacedoc.ProjectionGuide)
	if err != nil {
		return nil, err
	}
	var tasks []TrialTask
	suitesSeen := map[string]bool{}
	for _, c := range def.Cells {
		s := suiteFor(c.Group, c.Question)
		suitesSeen[s] = true
		tasks = append(tasks, TrialTask{
			ID:          "trial/" + c.ID,
			Suite:       s,
			GuideTaskID: c.ID,
			Description: c.Question,
		})
	}
	// One negative / honesty case per suite family (the agent should distrust a
	// low-confidence answer rather than hallucinate).
	negSuites := make([]string, 0, len(suitesSeen))
	for s := range suitesSeen {
		negSuites = append(negSuites, s)
	}
	sort.Strings(negSuites) // deterministic order
	for _, s := range negSuites {
		tasks = append(tasks, TrialTask{
			ID:          "trial/negative/" + s,
			Suite:       SuiteNegative,
			GuideTaskID: "negative/" + s,
			Description: "Honesty case for the " + s + " family: correctly distrust a low-confidence or absent answer rather than fabricate.",
			Negative:    true,
		})
	}
	if suite != "" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.Suite == suite {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}
	return tasks, nil
}

// suiteFor maps a Guide row's category/question to a trial suite family.
func suiteFor(group, question string) string {
	hay := strings.ToLower(group + " " + question)
	switch {
	case strings.Contains(hay, "research") || strings.Contains(hay, "investigat") || strings.Contains(hay, "explore"):
		return SuiteResearch
	case strings.Contains(hay, "comprehend") || strings.Contains(hay, "understand") || strings.Contains(hay, "explain") || strings.Contains(hay, "read"):
		return SuiteComprehend
	case strings.Contains(hay, "bug") || strings.Contains(hay, "fix") || strings.Contains(hay, "debug") || strings.Contains(hay, "repair"):
		return SuiteBugfix
	default:
		return SuiteAddFeature
	}
}
