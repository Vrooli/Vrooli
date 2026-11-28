package main

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"web-console/backends/codex"
	"web-console/internal/backend"
	"web-console/internal/sessionstore"
)

const (
	maintenanceInventoryTimeout     = 10 * time.Second
	maintenanceInventoryConcurrency = 8
)

// maintenanceInventory describes the work under Web Console ownership. The
// common fence drains ordinary managed turns before same-host replacement.
// Host coordination must also account for the tmux recovery requirement.
func (s *Server) maintenanceInventory(ctx context.Context) (lifecycle.Inventory, error) {
	ctx, cancel := context.WithTimeout(ctx, maintenanceInventoryTimeout)
	defer cancel()
	report := lifecycle.Inventory{Complete: true}
	rows, err := s.sessionStore.List(ctx)
	if err != nil {
		return report, err
	}
	registry := s.managedCodex
	registry.lifecycleMu.Lock()
	recoveryDone := registry.recoveryDone
	registry.lifecycleMu.Unlock()
	if recoveryDone != nil {
		select {
		case <-recoveryDone:
		default:
			report.Complete = false
		}
	}
	codexRows := make([]sessionstore.Metadata, 0, len(rows))
	managedRows := make([]sessionstore.Metadata, 0, len(rows))
	staticStates := make(map[string]string)
	for _, row := range rows {
		if row.Status != sessionstore.StatusLive || row.Retired() || row.RecoveredInto != "" {
			continue
		}
		if row.LaunchMode != sessionstore.LaunchModeCodexAppServer {
			if row.Backend == backend.Persistent && row.Detached {
				runtime, present := s.sessions.Get(row.ID)
				report.Work = append(report.Work, &commonv1.LifecycleWorkItem{
					Id: row.ID, Kind: "tmux", Status: "explicit_host_recovery_required",
					PhysicalExecutorPresent: present && !runtime.IsDead(),
				})
			}
			continue
		}
		codexRows = append(codexRows, row)
		switch codexMaintenanceReadMode(row.ControlMode) {
		case "recovery_only":
			staticStates[row.ID] = "recovery_only"
			continue
		case "native_state_unverified":
			staticStates[row.ID] = "native_state_unverified"
			continue
		}
		descriptor, descriptorErr := sessionstore.ParseLaunchDescriptor(row.LaunchDescriptorJSON)
		switch {
		case descriptorErr != nil:
			staticStates[row.ID] = "launch_descriptor_unverified"
		case managedCodexNeedsIdentityHandoff(row.Owner, descriptor):
			staticStates[row.ID] = "authorized_identity_handoff_required"
		default:
			managedRows = append(managedRows, row)
		}
	}
	states := readManagedThreadStates(ctx, managedRows, registry.ReadThreadStates)
	for _, row := range codexRows {
		item := &commonv1.LifecycleWorkItem{Id: row.ID, Kind: "codex_app_server"}
		_, item.PhysicalExecutorPresent = registry.owner(row.ID)
		item.Status = staticStates[row.ID]
		if item.Status == "" {
			item.Status = states[row.ID].status
		}
		if item.Status == "" {
			item.Status = "native_state_unverified"
		}
		switch item.Status {
		case "ordinary_turn_running", "waiting_for_native_input":
			report.Remaining++
		case "idle", "native_goal_continuation", "recovery_only":
		default:
			report.Complete = false
		}
		report.Work = append(report.Work, item)
	}
	if mode, _ := ctx.Value(hostSnapshotKey{}).(hostSnapshotMode); mode == hostInventory {
		if s.sessions.RecoveryProgress().InProgress {
			report.Complete = false
		}
		discovered, err := discoverHostTmuxSessions(ctx)
		if err != nil {
			return report, err
		}
		listed := make(map[string]bool)
		for _, item := range report.Work {
			if item.Kind == "tmux" {
				listed[item.Id] = true
			}
		}
		for _, id := range discovered {
			if !listed[id] {
				report.Work = append(report.Work, &commonv1.LifecycleWorkItem{
					Id: id, Kind: "tmux", Status: "explicit_host_recovery_required", PhysicalExecutorPresent: true,
				})
			}
		}
	}
	for _, local := range s.sessions.List() {
		if local.Backend == backend.Standard && !local.IsDead() {
			report.Work = append(report.Work, &commonv1.LifecycleWorkItem{
				Id: local.ID, Kind: "pty", Status: "explicit_host_recovery_required", PhysicalExecutorPresent: true,
			})
		}
	}
	return report, ctx.Err()
}

type managedThreadStateObservation struct {
	status string
}

// readManagedThreadStates bounds concurrency so a host snapshot does not
// serialize one provider round trip per live session. The whole inventory has
// a deadline; any unread session remains explicitly unverified.
func readManagedThreadStates(
	ctx context.Context,
	rows []sessionstore.Metadata,
	read func(context.Context, sessionstore.Metadata, []string) ([]codex.NativeThreadState, error),
) map[string]managedThreadStateObservation {
	observations := make(map[string]managedThreadStateObservation, len(rows))
	if len(rows) == 0 {
		return observations
	}
	workers := min(maintenanceInventoryConcurrency, len(rows))
	jobs := make(chan sessionstore.Metadata)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for row := range jobs {
				status := "native_state_unverified"
				states, err := read(ctx, row, []string{row.NativeThreadID})
				if err == nil && len(states) == 1 {
					status = codexMaintenanceStatus(states[0])
				}
				mu.Lock()
				observations[row.ID] = managedThreadStateObservation{status: status}
				mu.Unlock()
			}
		}()
	}

sendRows:
	for _, row := range rows {
		select {
		case jobs <- row:
		case <-ctx.Done():
			break sendRows
		}
	}
	close(jobs)
	wg.Wait()
	return observations
}

func codexMaintenanceStatus(state codex.NativeThreadState) string {
	switch state.Status {
	case "idle", "active":
	default:
		return "native_state_unverified"
	}
	if len(state.ActiveFlags) > 0 {
		return "waiting_for_native_input"
	}
	if state.GoalAvailable && state.Goal != nil && state.Goal.Status == "active" {
		if state.Status == "active" {
			if state.ConsoleTurn {
				return "ordinary_turn_running"
			}
			if state.ActiveTurnID == "" {
				return "native_state_unverified"
			}
		}
		return "native_goal_continuation"
	}
	if state.Status == "active" {
		return "ordinary_turn_running"
	}
	if !state.GoalAvailable {
		return "native_state_unverified"
	}
	return "idle"
}

func codexMaintenanceReadMode(mode sessionstore.ControlMode) string {
	switch mode {
	case sessionstore.ControlModeRecoveryOnly:
		// No managed owner is allowed to issue work for this thread. Preserve
		// its durable session record and leave it inactive across maintenance.
		return "recovery_only"
	case sessionstore.ControlModeNativeCapable:
		return "read"
	default:
		return "native_state_unverified"
	}
}
