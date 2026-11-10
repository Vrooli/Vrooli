package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/backends"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

type (
	executionAdmissionKey struct{}
	preparedExecution     struct {
		model           models.Model
		selection       backends.Selection
		payload         Payload
		finishInference func(context.Context) error
	}
)

// Admit is the single model-work admission boundary. The jobs scheduler owns
// the returned lease through execution and cleanup; direct engine consumers
// use this same boundary. Failed acquisition returns its partial ownership so
// every acquired resource is released before retry or terminal completion.
func (e *Engine) Admit(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	ctx = context.WithValue(ctx, capacityPriorityKey{}, string(job.Priority))
	group := newLeaseGroup(ctx)
	meta := map[string]string{}
	admission := jobs.Admission{Context: ctx, Release: group.release, Meta: meta}
	var pl Payload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return admission, fmt.Errorf("ai: decode payload: %w", err)
	}
	if pl.Backend == "" {
		return admission, errors.New("ai: job payload has no resolved provider; the jobs recovery boundary must fail legacy nonterminal jobs before admission")
	}
	model, ok := e.effectiveRegistry(ctx).ByID(pl.ModelID)
	if !ok {
		return admission, fmt.Errorf("ai: model %q not in registry", pl.ModelID)
	}
	if !e.deps.ModelInstalled(model.ID) {
		return admission, fmt.Errorf("%w: %q", ErrModelNotInstalled, model.ID)
	}
	if e.deps.ModelRuntimeReadiness != nil {
		if ready, state := e.deps.ModelRuntimeReadiness(ctx, model.ID); !ready {
			return admission, fmt.Errorf("%w: %q (%s)", ErrModelNotRuntimeReady, model.ID, state)
		}
	}
	host, err := e.deps.Probe.Inventory(ctx)
	if err != nil {
		return admission, fmt.Errorf("ai: recheck host support for %q: %w", model.ID, err)
	}
	if !model.SupportsHost(host) {
		return admission, fmt.Errorf("%w: %q (%s/%s)", ErrModelUnsupportedHost, model.ID, host.OS, host.Arch)
	}
	if pl.RequireGPU && e.deps.Capacity == nil {
		return admission, fmt.Errorf("%w: capacity manager is unavailable", ErrGPURequired)
	}

	bsel, err := e.deps.Backends.RevalidateProvider(ctx, backends.SelectRequest{
		Operation:       job.Operation,
		GPUViable:       pl.GPU,
		AllowBYOK:       pl.AllowBYOK,
		RequireAdapters: len(pl.Adapters) > 0,
	}, pl.Backend)
	if err != nil {
		return admission, err
	}
	if pl.RequireGPU && bsel.Tier != backends.TierLocalGPU {
		return admission, fmt.Errorf("%w for operation %q with model %q", ErrGPURequired, job.Operation, model.ID)
	}
	if bsel.Tier == backends.TierLocalCPU && !model.Hardware.CPUCapable {
		return admission, fmt.Errorf("ai: model %q has no CPU execution path on the current backend", model.ID)
	}
	if bsel.Tier == backends.TierLocalGPU && modelVRAMClaimBytes(model) <= 0 {
		if pl.RequireGPU {
			return admission, fmt.Errorf("%w: model %q has no declared GPU memory budget", ErrGPURequired, model.ID)
		}
		if !model.Hardware.CPUCapable {
			return admission, fmt.Errorf("ai: model %q has no declared GPU memory budget or CPU execution path", model.ID)
		}
		e.deps.Logger.Printf("ai: model %q has no declared GPU memory budget; using CPU", model.ID)
		bsel, err = e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
			Operation: job.Operation, ModelBackend: model.Backend, GPUViable: false,
			AllowBYOK: pl.AllowBYOK, RequireAdapters: len(pl.Adapters) > 0,
		})
		if err != nil {
			return admission, err
		}
	}

	local := bsel.Tier == backends.TierLocalCPU || bsel.Tier == backends.TierLocalGPU
	threads := min(4, runtime.NumCPU())
	if local && e.deps.CPU != nil {
		lease, err := e.deps.CPU(ctx, "image-tools:"+job.ID+":cpu", int64(threads)*1000)
		if err != nil {
			return admission, fmt.Errorf("ai: CPU capacity: %w", err)
		}
		if lease == nil {
			return admission, errors.New("ai: CPU admission returned no lease")
		}
		group.renew("CPU", lease)
		meta["cpu_capacity_milli"] = strconv.Itoa(threads * 1000)
	}
	ctx = context.WithValue(group.ctx, cpuThreadsContextKey{}, threads)
	group.ctx = ctx

	// A warm worker owns RAM and model-file leases for its entire residency,
	// independently of the per-job CPU lease. A model switch reaps it first.
	var warm *warmSidecarRunner
	if provider, ok := bsel.Provider.(*execProvider); ok && bsel.Tier == backends.TierLocalCPU {
		warm, _ = provider.warm.(*warmSidecarRunner)
	}
	if warm != nil && e.deps.Memory != nil {
		warmAdmission, err := warm.reserve(ctx, model.ID, job.ID, func(residentCtx context.Context) (*leaseGroup, error) {
			resident := newLeaseGroup(residentCtx)
			if err := e.admitModelMemory(resident, model, "image-tools:resident:"+model.ID, true); err != nil {
				return resident, err
			}
			return resident, nil
		}, meta)
		if warmAdmission.Release != nil {
			group.finish = append(group.finish, warmAdmission.Release)
		}
		if err != nil {
			return admission, err
		}
		ctx, group.ctx = warmAdmission.Context, warmAdmission.Context
		for key, value := range warmAdmission.Meta {
			meta[key] = value
		}
	} else {
		if err := e.admitModelMemory(group, model, "image-tools:"+job.ID+":ram", local); err != nil {
			return admission, err
		}
		ctx = group.ctx
		if local && e.deps.Memory != nil {
			ctx = context.WithValue(ctx, operationMemoryContextKey{}, true)
			meta["ram_capacity_claim_bytes"] = strconv.FormatInt(int64(model.Hardware.MinRAMGB)<<30, 10)
			meta["ram_capacity_policy"] = "shared-broker-no-reclaim"
		}
	}
	if !e.deps.ModelInstalled(model.ID) {
		return admission, fmt.Errorf("%w: %q", ErrModelNotInstalled, model.ID)
	}
	if e.deps.ModelRuntimeReadiness != nil {
		if ready, state := e.deps.ModelRuntimeReadiness(ctx, model.ID); !ready {
			return admission, fmt.Errorf("%w: %q (%s)", ErrModelNotRuntimeReady, model.ID, state)
		}
	}
	seenAdapters := map[string]bool{}
	for _, adapter := range pl.Adapters {
		if seenAdapters[adapter.ID] {
			continue
		}
		seenAdapters[adapter.ID] = true
		if e.deps.AcquireAdapterUse != nil {
			release, err := e.deps.AcquireAdapterUse(adapter.ID)
			if errors.Is(err, sharedcapacity.ErrAssetInUse) {
				err = fmt.Errorf("%w: %w", jobs.ErrCapacityBusy, err)
			}
			if err != nil {
				return admission, fmt.Errorf("ai: acquire adapter %q: %w", adapter.ID, err)
			}
			if release == nil {
				return admission, fmt.Errorf("ai: adapter %q admission returned no release function", adapter.ID)
			}
			group.releases = append(group.releases, func(context.Context) error { release(); return nil })
		}
		if e.deps.AdapterInstalled == nil || !e.deps.AdapterInstalled(adapter.ID) {
			return admission, fmt.Errorf("ai: adapter %q is no longer installed", adapter.ID)
		}
	}
	if e.deps.Capacity != nil && bsel.Tier == backends.TierLocalGPU {
		gpu, err := capacityGPU(host, modelVRAMClaimBytes(model), bsel.Provider)
		if err != nil {
			return admission, err
		}
		lease, err := e.deps.Capacity.Claim(ctx, "image-tools:"+job.ID, modelVRAMClaimBytes(model), CapacityClaimOptions{
			Priority: pl.CapacityPriority, AllowReclaim: pl.AllowReclaim, GPUUUID: gpu.UUID,
		})
		if err != nil {
			return admission, fmt.Errorf("ai: GPU capacity admission failed: %w", err)
		}
		group.releases = append(group.releases, func(cleanup context.Context) error { return e.deps.Capacity.Release(cleanup, lease) })
		if lease.DegradeToCPU {
			if pl.RequireGPU {
				return admission, fmt.Errorf("%w: capacity admission denied or deferred GPU execution for model %q", jobs.ErrCapacityBusy, model.ID)
			}
			if !model.Hardware.CPUCapable {
				return admission, fmt.Errorf("%w: GPU capacity unavailable; model %q has no CPU execution path", jobs.ErrCapacityBusy, model.ID)
			}
			bsel, err = e.deps.Backends.SelectProvider(ctx, backends.SelectRequest{
				Operation: job.Operation, ModelBackend: model.Backend, GPUViable: false,
				AllowBYOK: pl.AllowBYOK, RequireAdapters: len(pl.Adapters) > 0,
			})
			if err != nil {
				return admission, err
			}
			meta["capacity_fallback"] = "cpu"
		} else {
			pl.GPUUUID = gpu.UUID
			var stop func()
			ctx, stop = keepCapacityAlive(ctx, e.deps.Capacity, lease, capacityClaimTTL/3)
			group.stops = append(group.stops, stop)
			meta["gpu_capacity_claim_id"] = lease.ClaimID
			meta["gpu_uuid"] = gpu.UUID
		}
	}
	admission.Context = context.WithValue(ctx, executionAdmissionKey{}, &preparedExecution{model: model, selection: bsel, payload: pl, finishInference: group.release})
	admission.Release = func(cleanup context.Context) error {
		err := group.release(cleanup)
		if err != nil {
			meta["capacity_release"] = "failed"
			meta["capacity_release_warning"] = err.Error()
		} else {
			meta["capacity_release"] = "released"
		}
		return err
	}
	return admission, nil
}

func (e *Engine) admitModelMemory(group *leaseGroup, model models.Model, owner string, local bool) error {
	if e.deps.AcquireModelUse != nil {
		release, err := e.deps.AcquireModelUse(model.ID)
		if err != nil {
			if errors.Is(err, models.ErrModelInUse) {
				return fmt.Errorf("%w: %w", jobs.ErrCapacityBusy, err)
			}
			return fmt.Errorf("ai: acquire model use %q: %w", model.ID, err)
		}
		if release == nil {
			return fmt.Errorf("ai: acquire model use %q returned no release function", model.ID)
		}
		group.releases = append(group.releases, func(context.Context) error { release(); return nil })
	}
	if local && e.deps.Memory != nil {
		if model.Hardware.MinRAMGB <= 0 || model.Hardware.MinRAMGB > 1024 {
			return fmt.Errorf("ai: model %q requires a bounded host RAM budget before local execution", model.ID)
		}
		lease, err := e.deps.Memory(group.ctx, owner, int64(model.Hardware.MinRAMGB)<<30)
		if err != nil {
			return fmt.Errorf("ai: local RAM capacity: %w", err)
		}
		if lease == nil {
			return errors.New("ai: local RAM admission returned no lease")
		}
		group.renew("local RAM", lease)
	}
	return nil
}

// leaseGroup retains failed releases for a later retry. Renewal is stopped and
// joined once before any lease is released, preventing late heartbeats.
type leaseGroup struct {
	mu       sync.Mutex
	ctx      context.Context
	stops    []func()
	finish   []func(context.Context) error
	releases []func(context.Context) error
}

func newLeaseGroup(ctx context.Context) *leaseGroup { return &leaseGroup{ctx: ctx} }

func (g *leaseGroup) renew(kind string, lease MemoryLease) {
	scope := sharedcapacity.Maintain(g.ctx, lease)
	g.ctx = scope.Context()
	g.releases = append(g.releases, func(ctx context.Context) error {
		if err := scope.Release(ctx); err != nil {
			return fmt.Errorf("ai: release %s capacity: %w", kind, err)
		}
		return nil
	})
}

func (g *leaseGroup) release(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	var errs []error
	for i := len(g.finish) - 1; i >= 0; i-- {
		if g.finish[i] == nil {
			continue
		}
		if err := g.finish[i](ctx); err != nil {
			errs = append(errs, err)
		} else {
			g.finish[i] = nil
		}
	}
	for i := len(g.stops) - 1; i >= 0; i-- {
		g.stops[i]()
	}
	g.stops = nil
	for i := len(g.releases) - 1; i >= 0; i-- {
		if g.releases[i] == nil {
			continue
		}
		if err := g.releases[i](ctx); err != nil {
			errs = append(errs, err)
		} else {
			g.releases[i] = nil
		}
	}
	return errors.Join(errs...)
}
