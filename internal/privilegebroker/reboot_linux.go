//go:build linux

package privilegebroker

import (
	"context"
	"fmt"
	"time"
)

const (
	// scheduledRebootUnit names the one transient timer that holds a pending
	// reboot. Naming it is what makes scheduling idempotent and cancelling
	// unambiguous: there is at most one pending host reboot.
	scheduledRebootUnit  = "vrooli-scheduled-reboot"
	rebootCommandTimeout = 30 * time.Second
	systemctlBinary      = "/usr/bin/systemctl"
	systemdRunBinary     = "/usr/bin/systemd-run"
)

// executeSystemReboot asks systemd to reboot, immediately or after a bounded
// delay. It deliberately uses the systemd D-Bus API rather than `shutdown`:
// the broker unit runs with ProtectSystem=strict, and a scheduled power action
// must not depend on writing a nologin or utmp file.
func executeSystemReboot(ctx context.Context, executor Executor, req Request) Result {
	ctx, cancel := context.WithTimeout(ctx, rebootCommandTimeout)
	defer cancel()
	delay := req.Reboot.DelayMinutes
	if delay == 0 {
		if _, err := executor.Run(ctx, systemctlBinary, "reboot", "--no-block"); err != nil {
			return NewFailure(req.RequestID, req.Action, "reboot_failed")
		}
		return Result{
			Status:  "changed",
			Changed: true,
			Evidence: Evidence{Available: true, Active: true,
				Detail: "reboot requested now" + rebootDecisionDetail(req)},
		}
	}
	// A transient timer survives the request but not a reboot, which is exactly
	// the lifetime a pending reboot should have.
	_, scheduleErr := executor.Run(ctx, systemdRunBinary,
		"--quiet", "--collect", fmt.Sprintf("--on-active=%dm", delay),
		"--unit", scheduledRebootUnit,
		systemctlBinary, "reboot", "--no-block")
	if scheduleErr != nil {
		// One pending reboot per host: an already-active timer is a successful
		// schedule, not a reason to report failure.
		if _, err := executor.Run(ctx, systemctlBinary, "is-active", "--quiet", scheduledRebootUnit+".timer"); err == nil {
			return Result{
				Status:   "already_present",
				Evidence: Evidence{Available: true, Active: true, Detail: scheduledRebootUnit},
			}
		}
		return NewFailure(req.RequestID, req.Action, "reboot_schedule_failed")
	}
	if _, err := executor.Run(ctx, systemctlBinary, "is-active", "--quiet", scheduledRebootUnit+".timer"); err != nil {
		return NewFailure(req.RequestID, req.Action, "reboot_schedule_unverified")
	}
	return Result{
		Status:  "changed",
		Changed: true,
		Evidence: Evidence{Available: true, Active: true,
			Detail: fmt.Sprintf("%s in %dm", scheduledRebootUnit, delay) + rebootDecisionDetail(req)},
	}
}

// executeSystemRebootCancel removes a pending scheduled reboot. Cancelling when
// nothing is scheduled is a successful no-op: the caller's intent is already
// satisfied and reporting failure would invite a retry loop.
func executeSystemRebootCancel(ctx context.Context, executor Executor, req Request) Result {
	ctx, cancel := context.WithTimeout(ctx, rebootCommandTimeout)
	defer cancel()
	timer := scheduledRebootUnit + ".timer"
	if _, err := executor.Run(ctx, systemctlBinary, "is-active", "--quiet", timer); err != nil {
		return Result{
			Status:   "unchanged",
			Evidence: Evidence{Available: true, Active: false, Detail: "no scheduled reboot"},
		}
	}
	if _, err := executor.Run(ctx, systemctlBinary, "stop", timer); err != nil {
		return NewFailure(req.RequestID, req.Action, "reboot_cancel_failed")
	}
	if _, err := executor.Run(ctx, systemctlBinary, "is-active", "--quiet", timer); err == nil {
		return NewFailure(req.RequestID, req.Action, "reboot_cancel_unverified")
	}
	return Result{
		Status:   "changed",
		Changed:  true,
		Evidence: Evidence{Available: true, Active: false, Detail: "scheduled reboot cancelled"},
	}
}

// rebootDecisionDetail appends the operator decision that authorised the
// action to the evidence line. The broker never verifies the value; it exists
// so an audit reader can join the host action to the human decision.
func rebootDecisionDetail(req Request) string {
	if req.Reboot == nil || req.Reboot.DecisionID == "" {
		return ""
	}
	return " (decision " + req.Reboot.DecisionID + ")"
}
