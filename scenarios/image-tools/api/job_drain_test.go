package main

import (
	"strings"
	"testing"

	internaljobs "image-tools/internal/jobs"
)

func TestDrainInventoryCountsOnlyJobsThatHoldTheDrainAndReportsProgress(t *testing.T) {
	empty := jobInventory(nil)
	if empty.Remaining != 0 || !empty.Complete || len(empty.Work) != 0 {
		t.Fatalf("no active jobs must drain at once: %+v", empty)
	}
	inv := jobInventory([]internaljobs.ActiveJob{
		{ID: "j1", Operation: "edit_instruct", State: internaljobs.StateRunning, Progress: 64, Message: "denoising step 13/20"},
		{ID: "j2", Operation: "denoise", State: internaljobs.StateCanceling},
	})
	if inv.Remaining != 2 || !inv.Complete || len(inv.Work) != 2 {
		t.Fatalf("inventory = %+v", inv)
	}
	first := inv.Work[0]
	if !strings.Contains(first.Id, "j1") || !strings.Contains(first.Id, "edit_instruct") || first.Status != "running 64%: denoising step 13/20" || !first.PhysicalExecutorPresent {
		t.Fatalf("work item must name the job and its progress: %+v", first)
	}
}

func TestJobDrainHooksSupplyQuiesceAndResumeTogether(t *testing.T) {
	hooks := jobDrainHooks(nil)
	if hooks.Quiesce == nil || hooks.Resume == nil || hooks.Snapshot == nil {
		t.Fatalf("the fence requires quiesce and resume as a pair plus an inventory: %+v", hooks)
	}
}
