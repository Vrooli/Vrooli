package skillscripts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"research-desk/internal/testutil"
)

// The desk reads packet.py's result as the script prints it, over
// internal/testutil's excerpts.

func packets(t *testing.T) Packets {
	t.Helper()
	return Packets{Paths: Paths{Skills: skillsDir}, Runner: NewPython(), ResearchDir: testutil.NewHome(t).Research}
}

func TestAFailedGateIsAResultNotAnError(t *testing.T) {
	p := packets(t)
	spec := `{"topics":["niche-book-production"],"allowance_hours":12,"stance_quotes":["` + strings.Repeat("Make it big. ", 400) + `"],"slug":"programme-five","window":"fifth"}`
	result, err := p.Preview(context.Background(), []byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	failed := ""
	for _, g := range result.Gates {
		if !g.OK {
			failed = g.Name
		}
	}
	if result.OK || result.ExitCode != 1 || failed != "stance_chars" {
		t.Fatalf("ok %v, exit %d, failed gate %q", result.OK, result.ExitCode, failed)
	}
}

func TestAnInvalidSpecIsRefusedWithACode(t *testing.T) {
	result, err := packets(t).Preview(context.Background(), []byte(`{"topics":["no-such-topic"],"allowance_hours":12}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.OK || result.ExitCode != 2 || len(result.Errors) == 0 || result.Errors[0].Code == "" {
		t.Fatalf("result = %+v", result)
	}
}

// [REQ:RD-P0-001]
func TestCheckGatesARealPreparedPacket(t *testing.T) {
	result, err := packets(t).Check(context.Background(), "local-models-w1-20261111")
	if err != nil {
		t.Fatal(err)
	}
	if result.CommissionID != "local-models-w1-20261111" || len(result.Gates) == 0 || result.Written {
		t.Fatalf("result = %+v", result)
	}
}

type badRunner struct{ out Result }

func (b badRunner) Run(context.Context, string, []string, []byte) (Result, error) { return b.out, nil }

func TestAScriptThatPrintsNoResultSaysSo(t *testing.T) {
	for name, out := range map[string]Result{
		"crash":      {ExitCode: 120, Stderr: []byte("Traceback (most recent call last)\n")},
		"no json":    {ExitCode: 0, Stdout: []byte("hello")},
		"old schema": {ExitCode: 0, Stdout: []byte(`{"schema":"research-packet-result/v0"}`)},
	} {
		_, err := Packets{Paths: Paths{Skills: skillsDir}, Runner: badRunner{out}}.Check(context.Background(), "x")
		if err == nil || !strings.Contains(err.Error(), "packet.py") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// The research skills live in this scenario and the charter the desk renders
// from them carries the supervisor's route discipline: a failing optional
// route is recorded and the direct path taken, and decisions are written as
// needs-you items the desk delivers.
// [REQ:RD-P0-009]
func TestTheScenarioOwnedSkillsRenderRouteDiscipline(t *testing.T) {
	for _, skill := range []string{"research-coordination", "adaptive-research", "local-experimentation", "research-review", "research-desk"} {
		if _, err := os.Stat(filepath.Join(skillsDir, skill, "SKILL.md")); err != nil {
			t.Errorf("skill %s is not owned by this scenario: %v", skill, err)
		}
	}
	spec := `{"topics":["niche-book-production"],"allowance_hours":12,"stance_quotes":["Make it big."],"slug":"programme-five","window":"fifth"}`
	result, err := packets(t).Preview(context.Background(), []byte(spec))
	if err != nil || !result.OK || result.Texts == nil {
		t.Fatalf("preview: %v %+v", err, result)
	}
	for _, line := range []string{
		"## Route discipline",
		"`Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`",
		"Write each decision as a needs-you item in `STATE.md`",
	} {
		if !strings.Contains(result.Texts.Charter, line) {
			t.Errorf("the charter lacks %q", line)
		}
	}
}
