package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"image-tools/internal/capacity"
)

// yieldPacing is how long and how often a GPU claim is retried after an idle
// resident was unloaded. The broker decides from a host snapshot that trails a
// release (measured 6 to 20 s on this host; `capacity sweep` does not shorten
// it), so a retry made the instant the worker exits is refused for memory that
// is already free.
type yieldPacing struct{ settle, poll time.Duration }

var defaultYieldPacing = yieldPacing{settle: 25 * time.Second, poll: time.Second}

// evictedResidentsMetadataKey lists the idle warm models image-tools unloaded
// so this job's GPU claim could be granted.
const evictedResidentsMetadataKey = "evicted_residents"

// claimGPU claims VRAM for owner. The broker counts every claim, including the
// ones idle warm workers of this service keep for cached weights, so image-tools
// must not refuse itself for memory it can give back. When the claim is denied,
// idle warm residents on the same GPU are unloaded least-recently-used first,
// retrying admission after each, until it is granted or none is left.
//
// A resident that a job holds is never touched, and no other owner's claim is.
// meta names the models that were unloaded.
func (e *Engine) claimGPU(ctx context.Context, owner string, bytes int64, opts capacity.ClaimOptions, meta map[string]string) (capacity.Lease, error) {
	lease, err := e.deps.Capacity.Claim(ctx, owner, bytes, opts)
	if err != nil || !lease.DegradeToCPU {
		return lease, err
	}
	var evicted []string
	for _, idle := range e.idleGPUResidents(opts.GPUUUID) {
		model, yielded, yieldErr := idle.runner.yieldIdleGPU(idle)
		if yieldErr != nil {
			return lease, fmt.Errorf("ai: yield idle resident %q: %w", idle.model, yieldErr)
		}
		if !yielded {
			continue
		}
		evicted = append(evicted, model)
		meta[evictedResidentsMetadataKey] = strings.Join(evicted, ",")
		if lease, err = e.reclaim(ctx, owner, bytes, opts); err != nil || !lease.DegradeToCPU {
			return lease, err
		}
	}
	return lease, nil
}

// reclaim retries a denied claim after a resident was unloaded, polling until
// the broker's snapshot reflects the release or the settle window ends.
func (e *Engine) reclaim(ctx context.Context, owner string, bytes int64, opts capacity.ClaimOptions) (capacity.Lease, error) {
	deadline := time.Now().Add(e.yield.settle)
	for {
		lease, err := e.deps.Capacity.Claim(ctx, owner, bytes, opts)
		if err != nil || !lease.DegradeToCPU || e.yield.poll <= 0 || !time.Now().Add(e.yield.poll).Before(deadline) {
			return lease, err
		}
		select {
		case <-ctx.Done():
			return lease, ctx.Err()
		case <-time.After(e.yield.poll):
		}
	}
}

// idleGPUResidents lists this service's idle warm workers holding VRAM on the
// GPU, least recently used first.
func (e *Engine) idleGPUResidents(gpuUUID string) []idleGPUResident {
	if gpuUUID == "" {
		return nil
	}
	var idle []idleGPUResident
	for _, provider := range e.deps.Backends.All() {
		exec, ok := provider.(*execProvider)
		if !ok {
			continue
		}
		warm, ok := exec.warm.(*warmSidecarRunner)
		if !ok {
			continue
		}
		if resident, ok := warm.idleGPUResident(gpuUUID); ok {
			idle = append(idle, resident)
		}
	}
	sort.SliceStable(idle, func(i, j int) bool { return idle[i].lastUsed.Before(idle[j].lastUsed) })
	return idle
}
