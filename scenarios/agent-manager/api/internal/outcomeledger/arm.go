package outcomeledger

import (
	"strings"
	"time"

	"github.com/vrooli/cli-core/usagemode"
)

// Arm bases: how a run's arm in a Learn comparison is known.
const (
	ArmBasisCanary   = "canary_arm"
	ArmBasisMarker   = "launcher_marker"
	ArmBasisCalendar = "day_block_calendar"
	// markerWindow is how far a launcher marker may sit from the session start
	// it describes.
	markerWindow = 10 * time.Minute
)

// ArmAssignment is a run's place in a Learn comparison.
type ArmAssignment struct {
	Arm, ModeID, Basis, Exception string
}

// learnReason returns the Learn mode id a managed run recorded in its
// selection reason.
func learnReason(reason string) string {
	for _, part := range strings.Split(reason, ";") {
		if id, ok := strings.CutPrefix(part, "usage_mode_learn:"); ok {
			return id
		}
	}
	return ""
}

// ResolveArm names a run's arm. A managed run is exact: the policy snapshot's
// canary arm under the learn mode its selection reason names. An operator
// session is inferred: the launcher's marker when one sits at the session's
// start, otherwise the day-block calendar of an opted-in learn period. The
// calendar is the assignment; a marker only confirms it and records any
// exception, because analysis is by assigned arm whatever the session ran on.
func ResolveArm(run RunFacts, periods []ModePeriod, markers []usagemode.LaunchMarker) ArmAssignment {
	if id := learnReason(run.SelectionReason); id != "" {
		if run.CanaryArm == usagemode.ArmIncumbent || run.CanaryArm == usagemode.ArmChallenger {
			return ArmAssignment{Arm: run.CanaryArm, ModeID: id, Basis: ArmBasisCanary}
		}
		return ArmAssignment{}
	}
	if run.Role != "" || run.Path == PathHeadless || run.Path == PathInteractive || run.Path == PathAttached {
		return ArmAssignment{}
	}
	start := run.SessionStart
	if start.IsZero() || (!run.FirstActivity.IsZero() && run.FirstActivity.Before(start)) {
		start = run.FirstActivity
	}
	if start.IsZero() {
		return ArmAssignment{}
	}
	var best *ModePeriod
	for i := range periods {
		period := &periods[i]
		if period.Kind != usagemode.KindLearn || !period.SessionOptIn || !period.matchesRunner(run.Harness) || !period.Contains(start) {
			continue
		}
		if best == nil || period.StartedAt.After(best.StartedAt) {
			best = period
		}
	}
	if best == nil {
		return ArmAssignment{}
	}
	assignment := ArmAssignment{
		Arm:    usagemode.SessionArm(best.ID, start, best.Split, best.BlockDays),
		ModeID: best.ID,
		Basis:  ArmBasisCalendar,
	}
	for _, marker := range markers {
		if marker.ModeID != best.ID || !best.matchesRunner(marker.Runner) {
			continue
		}
		gap := marker.At.Sub(start)
		if gap < 0 {
			gap = -gap
		}
		if gap <= markerWindow {
			assignment.Basis, assignment.Exception = ArmBasisMarker, marker.Exception
			break
		}
	}
	return assignment
}

// ArmCounts tallies a learn mode's assigned work, by arm, for exposure checks.
type ArmCounts struct {
	Incumbent, Challenger, Exceptions int
}

// DayBlock is the index of the day block holding at, the block
// usagemode.SessionArm hashes. A session arm is assigned to the whole block, so
// the block, not the session, is what was randomized.
func DayBlock(at time.Time, blockDays int) int64 {
	if blockDays <= 0 {
		blockDays = usagemode.DefaultBlockDays
	}
	return at.UTC().Unix() / 86400 / int64(blockDays)
}

// sessionBasis reports an arm assigned by the day-block calendar.
func sessionBasis(basis string) bool { return basis == ArmBasisCalendar || basis == ArmBasisMarker }

// retryLink reports a link that makes a run another attempt at the same work: it
// stays in the arm its unit started in, whatever arm its own run id drew.
func retryLink(kind string) bool {
	return kind == LinkResume || kind == LinkSuccessor || kind == LinkTask || kind == LinkWorkloadRetry
}

// ArmWork returns every run of the units holding work assigned to modeID (any
// mode when empty): a unit is analysed whole, so a retry or child with no arm of
// its own still counts, and a unit started outside the comparison shows as spanning.
func ArmWork(runs []RunFacts, modeID string) []RunFacts {
	units := map[string]bool{}
	for _, run := range runs {
		if run.WorkUnitID != "" && run.Arm != "" && (modeID == "" || run.ArmMode == modeID) {
			units[run.WorkUnitID] = true
		}
	}
	var out []RunFacts
	for _, run := range runs {
		if units[run.WorkUnitID] {
			out = append(out, run)
		}
	}
	return out
}

// DropSpanning removes units that span arms, and their runs, returning how many went.
func DropSpanning(units []Unit, runs []Run) ([]Unit, []Run, int) {
	dropped := map[string]bool{}
	kept := units[:0:0]
	for _, unit := range units {
		if unit.SpansArms {
			dropped[unit.ID] = true
		} else {
			kept = append(kept, unit)
		}
	}
	samples := runs[:0:0]
	for _, run := range runs {
		if !dropped[run.UnitID] {
			samples = append(samples, run)
		}
	}
	return kept, samples, len(dropped)
}

// CountArms counts the runs assigned to a mode.
func CountArms(runs []RunFacts, modeID string) ArmCounts {
	var out ArmCounts
	for _, run := range runs {
		if run.ArmMode != modeID {
			continue
		}
		switch run.Arm {
		case usagemode.ArmIncumbent:
			out.Incumbent++
		case usagemode.ArmChallenger:
			out.Challenger++
		}
		if run.ArmException != "" {
			out.Exceptions++
		}
	}
	return out
}
