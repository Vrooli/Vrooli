package hostreqkit

import "fmt"

// DryRunComparison contains the two outcomes produced by one handler for the
// same host facts and status. Callers must stub mutation seams before invoking
// this helper; it deliberately exercises the real Apply method twice so gate
// ordering is tested rather than inferred from source shape.
type DryRunComparison struct {
	DryRun ItemStatus
	Apply  ItemStatus
}

// CompareDryRunAndApply verifies the invariant that a preview and a real run
// make the same gate decision. A handler may change the execution state only
// after all gates have passed; otherwise a preview can claim would_apply while
// the real run refuses to proceed.
func CompareDryRunAndApply(handler Handler, host Host, status ItemStatus, opts EnsureOptions) (DryRunComparison, error) {
	dryStatus, err := handler.Apply(host, status, EnsureOptions{
		Environment:       opts.Environment,
		SudoMode:          opts.SudoMode,
		AutoInstall:       opts.AutoInstall,
		IncludeOptional:   opts.IncludeOptional,
		MaintenanceWindow: opts.MaintenanceWindow,
		Stdout:            opts.Stdout,
		Stderr:            opts.Stderr,
		DryRun:            true,
	})
	if err != nil {
		return DryRunComparison{}, fmt.Errorf("dry-run: %w", err)
	}
	applyStatus, err := handler.Apply(host, status, EnsureOptions{
		Environment:       opts.Environment,
		SudoMode:          opts.SudoMode,
		AutoInstall:       opts.AutoInstall,
		IncludeOptional:   opts.IncludeOptional,
		MaintenanceWindow: opts.MaintenanceWindow,
		Stdout:            opts.Stdout,
		Stderr:            opts.Stderr,
	})
	if err != nil {
		return DryRunComparison{DryRun: dryStatus, Apply: applyStatus}, fmt.Errorf("apply: %w", err)
	}
	comparison := DryRunComparison{DryRun: dryStatus, Apply: applyStatus}
	if dryStatus.BlockingReason != applyStatus.BlockingReason {
		return comparison, fmt.Errorf("blocking reason mismatch: dry-run=%q apply=%q", dryStatus.BlockingReason, applyStatus.BlockingReason)
	}
	if dryStatus.ExecutionState == ExecutionWouldApply && applyStatus.ExecutionState != ExecutionWouldApply && applyStatus.BlockingReason == BlockingNone {
		return comparison, fmt.Errorf("dry-run reported would_apply while apply reached %q without a blocker", applyStatus.ExecutionState)
	}
	return comparison, nil
}
