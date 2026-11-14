package lifecycle

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// stopManagedComponents is the process owner's narrow pre-replacement path.
// It never discovers or signals an unrecorded process and never treats absent
// numeric PIDs or old exited statuses as completion. All handles are pinned
// before any signal, and every completion follows a wait on that handle.
func (r *Runner) stopManagedComponents(ctx context.Context, item scenario.Scenario, components []*commonv1.LifecycleManagedComponent) error {
	var original []process.Record
	read := func() ([]process.Record, error) {
		records, err := process.ReadScenarioRecords(r.Home, recordSlug(item))
		if original == nil && err == nil {
			original = records
		}
		return records, err
	}
	if err := stopComponentGenerations(ctx, item, components, read, process.PinRecord); err != nil {
		return err
	}
	for _, c := range components {
		for _, record := range original {
			if record.Step == "start-"+c.GetName() && record.Generation == c.GetGeneration() {
				if err := r.retireComponentGeneration(ctx, item, record); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// retireComponentGeneration is called only after this operation's pinned wait.
// Its tombstone prevents later cleanup from signaling a recycled PID. Neither
// the tombstone nor the registry's exited status is accepted as a new exit proof.
func (r *Runner) retireComponentGeneration(ctx context.Context, item scenario.Scenario, record process.Record) error {
	// Publish the filesystem suppression first. If registry retirement fails,
	// immediate cleanup must not reselect the old PID from a running record.
	fresh, err := process.ReadScenarioRecords(r.Home, recordSlug(item))
	if err != nil {
		return err
	}
	found := false
	for _, actual := range fresh {
		if actual.Step != record.Step {
			continue
		}
		if actual != record {
			return fmt.Errorf("component launch changed before verified exit retirement")
		}
		found = true
	}
	if !found {
		return fmt.Errorf("component launch disappeared before verified exit retirement")
	}
	record.Status = process.RecordStatusExitedVerified
	if err := process.WriteScenarioRecord(r.Home, recordSlug(item), record.Step, record); err != nil {
		return err
	}
	verified, err := process.ReadScenarioRecords(r.Home, recordSlug(item))
	if err != nil {
		return err
	}
	found = false
	for _, actual := range verified {
		if actual == record {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("verified exit tombstone changed before registry retirement")
	}
	if record.RuntimeInstanceID != "" {
		store, err := r.runtimeDeps().runtimeRegistry(ctx, r.Home)
		if err != nil {
			return err
		}
		defer store.Close()
		refs, err := store.ListProcessRefs(ctx, record.RuntimeInstanceID)
		if err != nil {
			return err
		}
		matched := false
		for _, ref := range refs {
			if ref.RefID != process.LaunchRefID(record.Generation) {
				continue
			}
			if ref.InstanceID != record.RuntimeInstanceID || ref.PID == nil || *ref.PID != record.PID || ref.Step != record.Step || !ref.StartedAt.Equal(record.StartedAt) {
				return fmt.Errorf("runtime process reference does not match verified launch")
			}
			ended := r.runtimeDeps().now().UTC()
			if _, err = store.UpdateProcessRefStatus(ctx, ref.RefID, scenarioruntime.ProcessRefStatusExited, &ended); err != nil {
				return err
			}
			matched = true
		}
		if !matched {
			return fmt.Errorf("verified launch has no exact runtime process reference")
		}
	}
	r.releaseContainment(record.PID)
	return nil
}

func stopComponentGenerations(ctx context.Context, item scenario.Scenario, components []*commonv1.LifecycleManagedComponent, read func() ([]process.Record, error), pin func(process.Record) (process.Handle, error)) error {
	records, err := read()
	if err != nil {
		return fmt.Errorf("read component launch records: %w", err)
	}
	var handles []process.Handle
	defer func() {
		for _, h := range handles {
			_ = h.Close()
		}
	}()
	selected := map[string]process.Record{}
	for _, c := range components {
		component, declared := item.Manifest.Components[c.GetName()]
		if !declared || component.Role != "sidecar" || c.GetGeneration() == "" || selected[c.GetName()].PID != 0 {
			return fmt.Errorf("component stop request is undeclared, duplicate, or missing generation: %q", c.GetName())
		}
		var matches []process.Record
		for _, record := range records {
			if record.Step == "start-"+c.GetName() {
				matches = append(matches, record)
			}
		}
		if len(matches) != 1 {
			return fmt.Errorf("component %q has no unique launch record", c.GetName())
		}
		record := matches[0]
		if record.Scenario != item.Slug || record.Generation != c.GetGeneration() || record.Status != "running" || record.Phase == "" || record.ProcessID != fmt.Sprintf("vrooli.%s.%s.%s", record.Phase, recordSlug(item), record.Step) {
			return fmt.Errorf("component %q launch record does not match requested generation", c.GetName())
		}
		h, err := pin(record)
		if err != nil {
			return fmt.Errorf("pin component %q: %w", c.GetName(), err)
		}
		handles = append(handles, h)
		selected[c.GetName()] = record
	}
	// Re-read after pinning so a launch record changed during pin cannot be
	// used to stop the superseded component. The signal still targets only the
	// pinned incarnation, even if the numeric PID changes after this check.
	fresh, err := read()
	if err != nil {
		return fmt.Errorf("recheck component launch records: %w", err)
	}
	for name, expected := range selected {
		matches := 0
		for _, actual := range fresh {
			if actual.Step == expected.Step {
				matches++
				if actual != expected {
					return fmt.Errorf("component %q launch record changed before stop", name)
				}
			}
		}
		if matches != 1 {
			return fmt.Errorf("component %q launch record disappeared before stop", name)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, h := range handles {
		if err := h.Signal(false); err != nil {
			return fmt.Errorf("signal pinned component: %w", err)
		}
	}
	for _, h := range handles {
		if err := process.AwaitExit(ctx, h); err != nil {
			return fmt.Errorf("wait pinned component: %w", err)
		}
	}
	return nil
}
