package maintenance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/supervision"
)

// A service's inherited agent labels are not execution ownership. The exception
// requires the canonical lifecycle registration AND the registered process's
// live instance, native service scope and stable birth. Environment strings or
// a service-looking scope alone never grant this exception.
func (c *Controller) inheritedServiceTags(ctx context.Context, executors []ExecutorScopeRef, entries map[int]processTableEntry, identities map[int]scopeIdentity, before, after map[int]supervision.ProcessInfo) (map[int]map[string]bool, error) {
	wanted := map[string]bool{}
	for _, executor := range executors {
		if executor.EndedAt != nil {
			wanted[executor.Tag] = true
			wanted[executor.LegacyTag] = true
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}
	ids := make([]string, 0)
	seen := map[string]bool{}
	for _, identity := range identities {
		if identity.RunID == "" && identity.InstanceID != "" && !seen[identity.InstanceID] && slices.ContainsFunc(identity.Tags, func(tag string) bool { return wanted[tag] }) {
			seen[identity.InstanceID] = true
			ids = append(ids, identity.InstanceID)
		}
	}
	if len(ids) == 0 || hostsession.CurrentBootID() == "" {
		return nil, nil
	}
	path, err := scenarioruntime.DefaultDBPath(c.Home)
	if err != nil {
		return nil, fmt.Errorf("service ownership registry unavailable")
	}
	if _, err = os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("service ownership registry unavailable")
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{DBPath: path, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("service ownership registry unavailable")
	}
	defer store.Close()
	instances, err := store.GetInstances(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("service ownership registrations unavailable")
	}
	refs, err := store.ListProcessRefsForInstances(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("service ownership registrations unavailable")
	}
	bootID := hostsession.CurrentBootID()
	roots := map[int]scopeIdentity{}
	for instanceID, instance := range instances {
		if instance.Status != scenarioruntime.StatusRunning || instance.StoppedAt != nil || instance.HostBootID != bootID ||
			(instance.OwnerKind != scenarioruntime.OwnerKindLifecycle && instance.OwnerKind != scenarioruntime.OwnerKindSupervisor) {
			continue
		}
		for _, ref := range refs[instanceID] {
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("service ownership observation deadline or cancellation")
			}
			if ref.PID == nil || ref.PGID == nil || *ref.PID <= 0 || ref.Status != "running" || ref.EndedAt != nil || ref.HostBootID != bootID || ref.Step == "" || ref.StartedAt.IsZero() {
				continue
			}
			pid := *ref.PID
			entry, present := entries[pid]
			identity := identities[pid]
			birth, stable := stableScopeBirth(pid, before, after)
			if !present || strings.HasPrefix(entry.State, "Z") || identity.RunID != "" || !sameServiceInstance(identity, instance) || !stable ||
				entry.PGID != *ref.PGID || birth.Before(ref.StartedAt.Add(-2*time.Second)) || birth.After(ref.StartedAt.Add(2*time.Second)) {
				continue
			}
			unit := scenarioruntime.ServiceScopeName((scenarioruntime.InstanceKey{Scenario: instance.Scenario, Variant: instance.Variant}).Slug(), ref.Step) + ".scope"
			if !filepath.IsAbs(entry.Cgroup) || filepath.Base(entry.Cgroup) != unit || filepath.Base(filepath.Dir(entry.Cgroup)) != platform.ServicesSlice {
				continue
			}
			roots[pid] = identity
		}
	}
	// Memoized parent traversal keeps deep service trees linear. Crossing a
	// run-labelled boundary, another instance, another scope, a missing parent or
	// a changed process incarnation cannot transfer service ownership.
	owners := map[int]int{}
	visiting := map[int]bool{}
	var owner func(int) int
	owner = func(pid int) int {
		if root, ok := owners[pid]; ok {
			return root
		}
		if _, root := roots[pid]; root {
			owners[pid] = pid
			return pid
		}
		entry, exists := entries[pid]
		identity := identities[pid]
		if !exists || visiting[pid] || identity.RunID != "" || ctx.Err() != nil {
			return 0
		}
		birth, stable := stableScopeBirth(pid, before, after)
		if !stable {
			return 0
		}
		visiting[pid] = true
		root := owner(entry.PPID)
		delete(visiting, pid)
		if root != 0 {
			rootIdentity := roots[root]
			if birth.Before(before[root].StartedAt.Add(-2*time.Second)) || entry.Cgroup != entries[root].Cgroup || identity.InstanceID != rootIdentity.InstanceID || identity.Scenario != rootIdentity.Scenario || identity.Variant != rootIdentity.Variant {
				root = 0
			}
		}
		owners[pid] = root
		return root
	}
	out := map[int]map[string]bool{}
	for pid, identity := range identities {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("service ownership observation deadline or cancellation")
		}
		root := owner(pid)
		if root == 0 {
			continue
		}
		rootIdentity := roots[root]
		inherited := map[string]bool{}
		for key, tag := range identity.TagLabels {
			if rootIdentity.TagLabels[key] == tag {
				inherited[tag] = true
			}
		}
		// A value used by an independent label is still positive evidence,
		// even when a different environment key inherited the same value.
		for key, tag := range identity.TagLabels {
			if rootIdentity.TagLabels[key] != tag {
				delete(inherited, tag)
			}
		}
		out[pid] = inherited
	}
	return out, nil
}

func sameServiceInstance(identity scopeIdentity, instance scenarioruntime.Instance) bool {
	return identity.InstanceID == instance.InstanceID && identity.Scenario == instance.Scenario && identity.Variant != "" && identity.Variant == instance.Variant
}

func stableScopeBirth(pid int, before, after map[int]supervision.ProcessInfo) (time.Time, bool) {
	first, foundBefore := before[pid]
	last, foundAfter := after[pid]
	return first.StartedAt, foundBefore && foundAfter && !first.StartedAt.IsZero() && first.StartedAt.Equal(last.StartedAt)
}
