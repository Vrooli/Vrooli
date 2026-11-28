package main

import (
	"context"
	"fmt"

	internaljobs "image-tools/internal/jobs"

	"github.com/vrooli/api-core/lifecycle"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// jobDrainHooks connects the durable job manager to the shared maintenance
// fence. Quiesce stops new work and returns every job that is only waiting
// (queued, or inside admission for capacity or model weights) to the queue, so
// the next process resumes it. Snapshot counts only the jobs that are actually
// computing and reports their progress, so a replacement waits for exactly
// that work and never for a job that is merely waiting.
func jobDrainHooks(manager *internaljobs.Manager) lifecycle.Hooks {
	return lifecycle.Hooks{
		Quiesce: manager.Quiesce,
		Resume:  manager.Resume,
		Snapshot: func(context.Context) (lifecycle.Inventory, error) {
			return jobInventory(manager.Active()), nil
		},
	}
}

func jobInventory(active []internaljobs.ActiveJob) lifecycle.Inventory {
	inv := lifecycle.Inventory{Remaining: int32(len(active)), Complete: true}
	for _, job := range active {
		status := fmt.Sprintf("%s %d%%", job.State, job.Progress)
		if job.Message != "" {
			status += ": " + job.Message
		}
		inv.Work = append(inv.Work, &commonv1.LifecycleWorkItem{
			Id:                      job.ID + " (" + job.Operation + ")",
			Kind:                    "image_tools_job",
			Status:                  status,
			PhysicalExecutorPresent: true,
		})
	}
	return inv
}
