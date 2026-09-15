package execution

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"swarm-manager/internal/review"
)

const criterionCheckTimeout = 30 * time.Second

type criterionCommandRunner interface {
	Run(context.Context, []string) (string, error)
}

type defaultCriterionCommandRunner struct{}

func (defaultCriterionCommandRunner) Run(ctx context.Context, argv []string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("empty command")
	}
	out, err := exec.CommandContext(ctx, argv[0], argv[1:]...).CombinedOutput()
	return string(out), err
}

// resolveCriterionChecks converts deterministic checks into observed evidence
// before the review workflow starts. It never starts another Test Genie run:
// phase checks are only settled when finalization has recorded their result.
// This execution bridge currently has no phase-result record, so it emits an
// explicit unavailable observation rather than guessing or rerunning a suite.
func resolveCriterionChecks(ctx context.Context, criteria []backlogCriterion, runner criterionCommandRunner) []review.EvidenceItem {
	if runner == nil {
		runner = defaultCriterionCommandRunner{}
	}
	result := make([]review.EvidenceItem, 0, len(criteria))
	for _, criterion := range criteria {
		if criterion.Check == nil || strings.TrimSpace(criterion.ID) == "" {
			continue
		}
		check := criterion.Check
		evidence := review.EvidenceItem{
			ID:          "machine-" + criterion.ID,
			CriterionID: criterion.ID,
			Title:       "Machine check: " + criterion.ID,
			Description: criterion.Gherkin,
			Trust:       "observed",
		}
		switch check.Kind {
		case "command":
			evidence.Type = review.EvidenceTypeCLIOutput
			checkCtx, cancel := context.WithTimeout(ctx, criterionCheckTimeout)
			output, err := runner.Run(checkCtx, check.Argv)
			timedOut := checkCtx.Err() == context.DeadlineExceeded
			cancel()
			evidence.Producer = "command"
			exitCode := commandExitCode(err)
			passed := !timedOut && exitCode == check.ExpectExit
			evidence.TestResults = []review.EvidenceTestResult{{Name: strings.Join(check.Argv, " "), Passed: passed, OutputSummary: strings.TrimSpace(output)}}
			switch {
			case timedOut:
				evidence.Settlement = "unavailable"
				evidence.UnavailableReason = "command check timed out"
			case !passed:
				evidence.Settlement = "refuted"
			default:
				evidence.Settlement = "settled"
			}
		case "test_genie_phase":
			evidence.Type = review.EvidenceTypeAPITest
			evidence.Producer = "test-genie"
			evidence.Settlement = "unavailable"
			evidence.UnavailableReason = "Test Genie phase result was not recorded by finalization"
		default:
			evidence.Type = review.EvidenceTypeCustom
			evidence.Producer = "criterion-check"
			evidence.Settlement = "unavailable"
			evidence.UnavailableReason = "unsupported criterion check kind"
		}
		result = append(result, evidence)
	}
	return result
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
