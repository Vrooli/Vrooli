package execution

import (
	"context"
	"errors"
	"testing"
)

type fakeCriterionCommandRunner struct {
	output string
	err    error
}

func (r fakeCriterionCommandRunner) Run(context.Context, []string) (string, error) {
	return r.output, r.err
}

func TestResolveCriterionChecks_CommandSettlesAndRefutes(t *testing.T) {
	criteria := []backlogCriterion{{ID: "criterion-1", Gherkin: "Given a check When it runs Then it settles.", Check: &backlogCriterionCheck{Kind: "command", Argv: []string{"check"}}}}
	settled := resolveCriterionChecks(context.Background(), criteria, fakeCriterionCommandRunner{output: "ok"})
	if len(settled) != 1 || settled[0].Settlement != "settled" || settled[0].Trust != "observed" {
		t.Fatalf("settled evidence = %#v", settled)
	}
	refuted := resolveCriterionChecks(context.Background(), criteria, fakeCriterionCommandRunner{err: errors.New("exit status 1")})
	if len(refuted) != 1 || refuted[0].Settlement != "refuted" {
		t.Fatalf("refuted evidence = %#v", refuted)
	}
}

func TestResolveCriterionChecks_CommandHonorsExpectedExit(t *testing.T) {
	criteria := []backlogCriterion{{ID: "criterion-1", Gherkin: "Given a check When it exits Then its declared exit code settles it.", Check: &backlogCriterionCheck{Kind: "command", Argv: []string{"check"}, ExpectExit: 1}}}
	got := resolveCriterionChecks(context.Background(), criteria, fakeCriterionCommandRunner{err: errors.New("exit status 1")})
	if len(got) != 1 || got[0].Settlement != "refuted" {
		t.Fatalf("non-ExitError is not a trustworthy exit code: %#v", got)
	}
}

func TestResolveCriterionChecks_TestGenieIsUnavailableWithoutRecordedResult(t *testing.T) {
	criteria := []backlogCriterion{{ID: "criterion-1", Gherkin: "Given a phase When finalization runs Then it is read.", Check: &backlogCriterionCheck{Kind: "test_genie_phase", Scenario: "swarm-manager", Phase: "unit"}}}
	got := resolveCriterionChecks(context.Background(), criteria, nil)
	if len(got) != 1 || got[0].Settlement != "unavailable" || got[0].Producer != "test-genie" {
		t.Fatalf("evidence = %#v", got)
	}
}
