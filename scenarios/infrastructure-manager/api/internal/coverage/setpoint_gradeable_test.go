package coverage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/spacedoc"
	repocontract "github.com/vrooli/repo-contract-go"
)

func writeSetpoint(t *testing.T, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "infrastructure-manager", "setpoint")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reliability-setpoint.json"), []byte(body), 0o644))
	return root
}

// A bar that grades nothing and does not say so reads as a target while
// silently letting every reading fall through to "not evaluated".
func TestLoadSetpointRejectsASilentlyUngradeableBar(t *testing.T) {
	root := writeSetpoint(t, `{
  "schema_version": "1",
  "bars": [{"id":"quiet","cell_ref":"availability/A1","projection":"availability","decision_ref":"d","sustain":"24h"}]
}`)
	_, err := LoadSetpoint(root)
	require.ErrorContains(t, err, "not gradeable")
}

// Claiming gradeability without a threshold or a unit is the same failure
// wearing a flag.
func TestLoadSetpointRejectsGradeableBarsWithoutThresholdOrUnit(t *testing.T) {
	root := writeSetpoint(t, `{
  "schema_version": "1",
  "bars": [{"id":"no-threshold","cell_ref":"availability/A1","projection":"availability","decision_ref":"d","gradeable":true,"unit":"percent"}]
}`)
	_, err := LoadSetpoint(root)
	require.ErrorContains(t, err, "authors no min or max")

	root = writeSetpoint(t, `{
  "schema_version": "1",
  "bars": [{"id":"no-unit","cell_ref":"availability/A1","projection":"availability","decision_ref":"d","gradeable":true,"min":99.5}]
}`)
	_, err = LoadSetpoint(root)
	require.ErrorContains(t, err, "names no unit")
}

// An explicitly non-gradeable bar with a stated reason is legitimate: some
// deadbands are genuinely prose, and saying so is the honest form.
func TestLoadSetpointAcceptsAnExplainedNonGradeableBar(t *testing.T) {
	root := writeSetpoint(t, `{
  "schema_version": "1",
  "bars": [{"id":"prose","cell_ref":"availability/A3","projection":"availability","decision_ref":"d","gradeable":false,"not_gradeable_reason":"open-loop cell; no sensor exists"}]
}`)
	setpoint, err := LoadSetpoint(root)
	require.NoError(t, err)
	require.Len(t, setpoint.Bars, 1)
	require.False(t, setpoint.Bars[0].Gradeable)
}

// The checked-in setpoint is the one this scenario actually grades against, so
// it has to satisfy its own contract.
func TestCheckedInSetpointIsLoadable(t *testing.T) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	require.NoError(t, err)
	setpoint, err := LoadSetpoint(root)
	require.NoError(t, err)
	require.NotEmpty(t, setpoint.Bars)

	gradeable := 0
	for _, bar := range setpoint.Bars {
		if bar.Gradeable {
			gradeable++
		}
	}
	// The instrument exists to grade. If almost nothing is gradeable, every
	// condition reading returns NOT_EVALUATED and the board reports no state
	// at all — which is the failure this contract was added to catch.
	require.Greater(t, gradeable, len(setpoint.Bars)/2,
		"most bars must be gradeable; %d of %d are", gradeable, len(setpoint.Bars))
}

// Every bar must grade a cell that some owner actually declared. A bar whose
// cell_ref resolves to nothing is worse than a missing bar: it reports as a
// target, contributes to the "bars authored" count, and grades nothing, so the
// board looks more instrumented than it is. This is the check SETPOINT-MODEL.md
// § Setpoint Integrity names first.
func TestEveryBarResolvesToADeclaredCell(t *testing.T) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	require.NoError(t, err)
	setpoint, err := LoadSetpoint(root)
	require.NoError(t, err)

	declared := map[string]spacedoc.CellStatus{}
	reader := FileSpaceReader{Root: root}
	for projection := range projectionOwners {
		def, readErr := reader.Read(context.Background(), projection)
		require.NoError(t, readErr, "projection %s must be readable", projection)
		for _, cell := range def.Cells {
			declared[string(projection)+"/"+cell.ID] = cell.Status
		}
	}

	for _, bar := range setpoint.Bars {
		status, ok := declared[bar.CellRef]
		require.True(t, ok, "bar %q grades %q, which no owner declares", bar.ID, bar.CellRef)
		if status != spacedoc.StatusMissing {
			continue
		}
		// A bar on an open-loop cell is legitimate — it pre-declares the target
		// so the gap is visible with its intent attached. What it may not do is
		// route the gap to the plant: an absent sensor is instrument work, and
		// filing it as a runtime finding sends someone to fix a machine that is
		// not broken.
		require.Equal(t, "instrumentation-gap", bar.Actuator,
			"bar %q covers the open-loop cell %q, so its actuator must be instrumentation-gap, not %q", bar.ID, bar.CellRef, bar.Actuator)
	}

	// The converse, and the rule SETPOINT-MODEL.md states outright: a cell whose
	// sensor exists and whose join is built must have a bar. Readings flowing
	// into nothing is the quietest failure on the board — the cell counts as
	// instrumented in every coverage ratio and is never evaluated.
	graded := map[string]bool{}
	for _, bar := range setpoint.Bars {
		graded[bar.CellRef] = true
	}
	for ref, status := range declared {
		if status == spacedoc.StatusNow {
			require.True(t, graded[ref], "cell %q is instrumented (NOW) and has no bar; its readings are never evaluated", ref)
		}
	}
}

// A provisional bar that does not say what is wrong with it is the state this
// field exists to prevent: it looks examined and is not. A ratified bar must
// likewise carry the evidence that cleared it, or ratification is a flag flip.
func TestProvisionalAndRatifiedBarsBothCarryTheirReason(t *testing.T) {
	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	require.NoError(t, err)
	setpoint, err := LoadSetpoint(root)
	require.NoError(t, err)

	reasons := map[string]bool{}
	for _, bar := range setpoint.Bars {
		if bar.Provisional {
			require.NotEmpty(t, bar.ProvisionalReason, "provisional bar %q states no reason", bar.ID)
			require.Empty(t, bar.RatificationNote, "bar %q is both provisional and ratified", bar.ID)
			// One boilerplate reason repeated across every bar is
			// indistinguishable from nobody having examined them.
			require.False(t, reasons[bar.ProvisionalReason],
				"bar %q repeats another bar's provisional reason verbatim; a shared reason is not a per-bar judgment", bar.ID)
			reasons[bar.ProvisionalReason] = true
			continue
		}
		if bar.RatificationNote != "" {
			require.NotEmpty(t, bar.DecisionRef, "ratified bar %q carries no decision_ref", bar.ID)
		}
	}
}
