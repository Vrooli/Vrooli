package backlog

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	CheckKindTestGeniePhase = "test_genie_phase"
	CheckKindCommand        = "command"
)

// NormalizeCriteria validates replacement criteria, retaining a previous ID
// when its Gherkin text is unchanged and allocating monotonic criterion-N IDs
// for new conditions. A removed ID is never allocated again.
func NormalizeCriteria(previous, requested []Criterion) []Criterion {
	byText := make(map[string]string, len(previous))
	max := 0
	for _, criterion := range previous {
		byText[strings.TrimSpace(criterion.Gherkin)] = criterion.ID
		if n, ok := criterionNumber(criterion.ID); ok && n > max {
			max = n
		}
	}
	out := make([]Criterion, 0, len(requested))
	seen := make(map[string]struct{}, len(requested))
	for _, criterion := range requested {
		criterion.Gherkin = strings.TrimSpace(criterion.Gherkin)
		if criterion.Gherkin == "" || !strings.HasPrefix(strings.ToLower(criterion.Gherkin), "given ") {
			continue
		}
		if _, duplicate := seen[criterion.Gherkin]; duplicate {
			continue
		}
		seen[criterion.Gherkin] = struct{}{}
		if existing := byText[criterion.Gherkin]; existing != "" {
			criterion.ID = existing
		} else {
			max++
			criterion.ID = fmt.Sprintf("criterion-%d", max)
		}
		if !validCheck(criterion.Check) {
			criterion.Check = nil
		}
		out = append(out, criterion)
	}
	return out
}

func criterionNumber(id string) (int, bool) {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "criterion-"))
	return n, err == nil && n > 0 && fmt.Sprintf("criterion-%d", n) == id
}

func validCheck(check *Check) bool {
	if check == nil {
		return true
	}
	switch check.Kind {
	case CheckKindTestGeniePhase:
		return strings.TrimSpace(check.Scenario) != "" && strings.TrimSpace(check.Phase) != ""
	case CheckKindCommand:
		return len(check.Argv) > 0 && strings.TrimSpace(check.Argv[0]) != ""
	default:
		return false
	}
}

func cloneCriteria(in []Criterion) []Criterion {
	out := make([]Criterion, len(in))
	for i, criterion := range in {
		out[i] = criterion
		if criterion.Check != nil {
			check := *criterion.Check
			check.Argv = append([]string(nil), criterion.Check.Argv...)
			out[i].Check = &check
		}
	}
	return out
}
