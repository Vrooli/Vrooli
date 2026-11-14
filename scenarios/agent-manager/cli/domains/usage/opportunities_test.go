package usage

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// body is the registry's response for rows built the way the measure builds them.
func body(t *testing.T, rows ...map[string]string) []byte {
	t.Helper()
	out, err := json.Marshal(map[string]any{"fields": rows})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func runOpportunities(t *testing.T, response []byte, args ...string) (string, map[string]string) {
	t.Helper()
	var sent map[string]string
	cmd := opportunitiesCommand(func(_ cliapp.RunContext, params map[string]string) ([]byte, error) {
		sent = params
		return response, nil
	})
	var stdout bytes.Buffer
	ctx, err := cliapp.NewTestRunContextFromArgs(cmd.Args, args, nil, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunCtx(ctx); err != nil {
		t.Fatalf("usage opportunities %v: %v", args, err)
	}
	return stdout.String(), sent
}

var coverageRow = map[string]string{"kind": "coverage", "sentence": "1383 projected command calls in 40 runs"}

// Opportunities are grouped by owner in the measure's rank order (an owner
// where its best-ranked opportunity is; the CLI sums nothing), each with its
// evidence and the check that verifies it; the window start is sent to the
// measure.
func TestOpportunitiesGroupByOwnerRankedWithEvidence(t *testing.T) {
	out, sent := runOpportunities(t, body(t, coverageRow,
		map[string]string{"kind": "bypass", "owner": "agent-manager", "subject": "agent-manager via sqlite3 on its database", "action": "add a CLI command", "calls_saved": "139", "saved_label": "calls replaced", "saved_basis": "direct reads", "sessions": "12", "occurrences": "139", "details": "table runs ×80\ntable run_events ×40", "verify": "v"},
		map[string]string{"kind": "sequence", "verdict": "adopt", "owner": "test-genie", "subject": "vrooli scenario test → test-genie runs wait → vrooli-memory journal note", "action": "adopt existing program `test-genie.iterate`", "calls_saved": "30", "saved_label": "calls saved", "saved_basis": "each full occurrence's steps less the one program call that would run them", "tokens_saved": "12000", "sessions": "7", "occurrences": "10", "program": "test-genie.iterate", "program_basis": "owner", "program_runs": "2", "examples": "vrooli scenario test x", "run_ids": "r1\nr2", "verify": "usage opportunities --since D"},
		map[string]string{"kind": "skill-reload", "owner": "prompt-manager", "subject": "adaptive-research", "action": "split it", "calls_saved": "5", "saved_label": "reads saved", "saved_basis": "reads of a skill still in context", "sessions": "3", "occurrences": "5", "verify": "v"},
	), "--since", "2026-10-30T00:00:00Z")
	if sent["since"] != "2026-10-30T00:00:00Z" {
		t.Fatalf("sent %v", sent)
	}
	for _, want := range []string{
		"Coverage: 1383 projected command calls in 40 runs",

		"[sequence · adopt] vrooli scenario test → test-genie runs wait → vrooli-memory journal note",
		"139 calls replaced (direct reads) · 12 sessions · 139 occurrences",
		"30 calls saved (each full occurrence's steps less the one program call that would run them), up to 12k result tokens · 7 sessions · 10 occurrences",
		"program test-genie.iterate (owner match): 2 program runs vs 10 manual",
		"touches table run_events ×40",
		"e.g. vrooli scenario test x",
		"runs r1, r2",
		"verify: usage opportunities --since D",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	var owners []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line == "agent-manager" || line == "test-genie" || line == "prompt-manager" {
			owners = append(owners, line)
		}
	}
	if strings.Join(owners, ",") != "agent-manager,test-genie,prompt-manager" || strings.Contains(out, "opportunities ·") {
		t.Fatalf("owners %v not in the measure's order, or re-summed:\n%s", owners, out)
	}
}

// An empty list is printed beside its coverage and says it is bounded.
func TestEmptyOpportunitiesPrintCoverage(t *testing.T) {
	out, sent := runOpportunities(t, body(t, coverageRow))
	if len(sent) != 0 || !strings.Contains(out, "Coverage: 1383 projected command calls") || !strings.Contains(out, "No pattern met its threshold") {
		t.Fatalf("sent %v, output:\n%s", sent, out)
	}
}

func TestOpportunitiesJSONIsTheMeasureBody(t *testing.T) {
	response := body(t, coverageRow)
	out, _ := runOpportunities(t, response, "--json")
	if strings.TrimSpace(out) != string(response) {
		t.Fatalf("--json = %s, want the registry body %s", out, response)
	}
}

// A mandated ritual keeps its work: the list states round-trips saved, never
// calls removed.
func TestMandatedRitualStatesRoundTrips(t *testing.T) {
	out, _ := runOpportunities(t, body(t, coverageRow,
		map[string]string{"kind": "sequence", "verdict": "candidate", "mandated": "true", "owner": "prompt-manager", "subject": "journal → query → read", "action": "write a program", "calls_saved": "40", "tokens_saved": "9000", "sessions": "9", "occurrences": "20", "verify": "v"},
	))
	if !strings.Contains(out, "40 round-trips saved (the mandated ritual's work stays)") || strings.Contains(out, "40 calls saved,") {
		t.Fatalf("output:\n%s", out)
	}
}

// A sequence the API found independent or unmeasured shows its verdict and
// action as given and claims no savings, whatever the numbers beside it.
func TestIndependentSequenceClaimsNoSavings(t *testing.T) {
	out, _ := runOpportunities(t, body(t, coverageRow,
		map[string]string{"kind": "sequence", "verdict": "independent", "owner": "prompt-manager", "subject": "query → read", "action": "agents choose each step independently (0% of 5 later steps used an earlier step's result)", "calls_saved": "0", "tokens_saved": "0", "sessions": "5", "occurrences": "5", "verify": "v"},
	))
	if !strings.Contains(out, "[sequence · independent] query → read") || !strings.Contains(out, "choose each step independently") || !strings.Contains(out, "no program savings claimed") || strings.Contains(out, "0 calls saved") {
		t.Fatalf("output:\n%s", out)
	}
}
