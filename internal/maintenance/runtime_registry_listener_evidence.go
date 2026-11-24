package maintenance

import (
	"context"
	"os"

	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func runtimeListenerEvidence(snapshot network.TCPListenerSnapshot, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef) map[int]scenarioruntime.ListenerEvidence {
	return scenarioruntime.ListenerEvidenceFromClaims(claims, refs, func(port int) scenarioruntime.ListenerEvidence {
		state := snapshot.Listening(port)
		if !state.Known {
			return scenarioruntime.ListenerEvidence{Known: false}
		}
		return scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
	})
}

func listRuntimeProcessRefs(ctx context.Context, store runtimeMaintenanceStore, claims []RuntimeClaimInfo) ([]RuntimeProcessRefInfo, error) {
	seen := make(map[string]struct{})
	pidRunning := newPIDLivenessMemo(processIsRunning)
	instanceIDs := make([]string, 0, len(claims))
	for _, claim := range claims {
		instanceIDs = append(instanceIDs, claim.InstanceID)
	}
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	out := make([]RuntimeProcessRefInfo, 0)
	for _, claim := range claims {
		if _, ok := seen[claim.InstanceID]; ok {
			continue
		}
		seen[claim.InstanceID] = struct{}{}
		for _, ref := range refsByInstance[claim.InstanceID] {
			item := RuntimeProcessRefInfo{
				RefID:          ref.RefID,
				InstanceID:     ref.InstanceID,
				Scenario:       claim.Scenario,
				InstanceStatus: claim.InstanceStatus,
				PID:            ref.PID,
				PGID:           ref.PGID,
				ProcessID:      ref.ProcessID,
				Step:           ref.Step,
				Command:        ref.Command,
				Status:         ref.Status,
			}
			if ref.PID != nil {
				running := pidRunning(*ref.PID)
				item.PIDRunning = &running
			}
			out = append(out, item)
		}
	}
	return out, nil
}

func runtimeTrackedProcessRefs(home string) ([]trackedProcessRef, error) {
	store, closeStore, err := openRuntimeRegistryFn(home)
	if err != nil || store == nil {
		return nil, err
	}
	defer closeStore()

	ctx := context.Background()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	// One batched query instead of one ListProcessRefs round trip per instance.
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	out := make([]trackedProcessRef, 0)
	for _, instance := range instances {
		for _, ref := range refsByInstance[instance.InstanceID] {
			item := trackedProcessRef{}
			if ref.PID != nil {
				item.PID = *ref.PID
			}
			if ref.PGID != nil {
				item.PGID = *ref.PGID
			}
			if item.PID > 0 || item.PGID > 0 {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func processIsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	if os.Getpid() == pid {
		return true
	}
	return pidIsRunningFn(pid)
}

// newPIDLivenessMemo wraps a liveness function with a per-run cache so each
// distinct PID is probed at most once per registry scan, regardless of how
// many claims or process refs reference it.
func newPIDLivenessMemo(isRunning func(int) bool) func(int) bool {
	cache := make(map[int]bool)
	return func(pid int) bool {
		if alive, ok := cache[pid]; ok {
			return alive
		}
		alive := isRunning(pid)
		cache[pid] = alive
		return alive
	}
}
