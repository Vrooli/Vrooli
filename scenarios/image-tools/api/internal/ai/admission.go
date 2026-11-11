package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/adapters"
	"image-tools/internal/backends"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

type (
	executionAdmissionKey struct{}
	preparedExecution     struct {
		model                   models.Model
		selection               backends.Selection
		payload                 Payload
		finishInference         func(context.Context) error
		markDiskCleanupVerified func(bool)
	}
)

var ErrDiskOutputCleanupUnverified = errors.New("ai: output temporary-file cleanup is unverified")

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
	if e.deps.Disk == nil {
		return admission, errors.New("ai: shared host disk admission is unavailable; refusing output generation")
	}
	if e.deps.Disk != nil {
		rootProvider, ok := e.deps.Store.(interface {
			ReservationRoot(context.Context) (string, error)
		})
		if !ok {
			return admission, errors.New("ai: blob store does not expose its routed output filesystem for disk admission")
		}
		outputRoot, err := rootProvider.ReservationRoot(ctx)
		if err != nil {
			return admission, fmt.Errorf("ai: resolve output filesystem for disk admission: %w", err)
		}
		outputPeak, tempPeak, err := estimateDiskPeak(pl)
		if err != nil {
			return admission, err
		}
		lease, err := e.deps.Disk.ReserveDiskJob(ctx, sharedcapacity.DiskJobRequest{
			OwnerID: "image-tools:" + job.ID, JobID: job.ID, FloorBytes: 120 << 30,
			Paths: []sharedcapacity.DiskJobPath{
				{Path: outputRoot, Role: "generated-output-and-publication", PeakBytes: outputPeak},
				{Path: os.TempDir(), Role: "worker-temporary-files", PeakBytes: tempPeak},
			},
		})
		if err != nil {
			return admission, fmt.Errorf("ai: reserve complete disk job peak: %w", err)
		}
		if lease == nil {
			return admission, errors.New("ai: disk admission returned no job lease")
		}
		group.renew("disk job", lease)
		reservation := lease.Reservation()
		meta["disk_job_reservation_id"] = reservation.ID
		meta["disk_job_filesystems"] = strconv.Itoa(len(reservation.Filesystems))
		for i, filesystem := range reservation.Filesystems {
			prefix := fmt.Sprintf("disk_job_filesystem_%d_", i)
			meta[prefix+"id"] = filesystem.FilesystemID
			meta[prefix+"peak_bytes"] = strconv.FormatInt(filesystem.PeakBytes, 10)
			meta[prefix+"available_bytes"] = strconv.FormatInt(filesystem.AvailableBytes, 10)
			meta[prefix+"other_reserved_bytes"] = strconv.FormatInt(filesystem.OtherReservedBytes, 10)
		}
		ctx = group.ctx
	}
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
		if adapter.LicenseTermsVersion != "" {
			prefix := "adapter." + adapter.ID + "."
			meta[prefix+"license_terms_version"] = adapter.LicenseTermsVersion
			meta[prefix+"license_terms_accepted"] = strconv.FormatBool(adapter.LicenseTermsAccepted)
		}
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
	if hasPosePreprocessor(pl) && e.deps.Memory != nil {
		// MediaPipe's CPU pose task executes inside the generation job. Keep its
		// bounded worker memory in the same admission lifetime as the generator.
		const poseWorkerRAMBytes = int64(1 << 30)
		lease, err := e.deps.Memory(group.ctx, "image-tools:"+job.ID+":pose-preprocessor-ram", poseWorkerRAMBytes)
		if err != nil {
			return admission, fmt.Errorf("ai: pose preprocessor RAM admission: %w", err)
		}
		if lease == nil {
			return admission, errors.New("ai: pose preprocessor RAM admission returned no lease")
		}
		group.renew("pose preprocessor RAM", lease)
		meta["pose_preprocessor_ram_claim_bytes"] = strconv.FormatInt(poseWorkerRAMBytes, 10)
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
	admission.Context = context.WithValue(ctx, executionAdmissionKey{}, &preparedExecution{model: model, selection: bsel, payload: pl, finishInference: group.releaseInference, markDiskCleanupVerified: group.markDiskCleanupVerified})
	admission.Release = func(cleanup context.Context) error {
		err := group.release(cleanup)
		if err != nil {
			if errors.Is(err, ErrDiskOutputCleanupUnverified) {
				meta["capacity_release"] = "retained"
			} else {
				meta["capacity_release"] = "failed"
			}
			meta["capacity_release_warning"] = err.Error()
		} else {
			meta["capacity_release"] = "released"
		}
		return err
	}
	return admission, nil
}

func hasPosePreprocessor(pl Payload) bool {
	for _, adapter := range pl.Adapters {
		if adapter.Preprocessor == adapters.PreprocessorPose {
			return true
		}
	}
	return false
}

const maxDiskImagePixels int64 = 128_000_000
const imageToolsDiskPlanningCeiling int64 = 25 << 30

// estimateDiskPeak bounds managed outputs and temporary raster materialization
// from the image ingest limits. Unknown input dimensions use the maximum allowed
// pixel area; all arithmetic is checked before admission.
func estimateDiskPeak(pl Payload) (outputBytes, tempBytes int64, err error) {
	variations := int64(pl.Variations)
	if variations < 1 {
		variations = 1
	}
	width, height := int64(1024), int64(1024)
	if raw := pl.Params["width"]; raw != "" {
		width, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || width <= 0 {
			return 0, 0, fmt.Errorf("ai: invalid output width for disk estimate")
		}
	}
	if raw := pl.Params["height"]; raw != "" {
		height, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || height <= 0 {
			return 0, 0, fmt.Errorf("ai: invalid output height for disk estimate")
		}
	}
	if width > math.MaxInt64/height {
		return 0, 0, fmt.Errorf("ai: output dimensions overflow the disk estimate")
	}
	pixels := width * height
	if pl.InputKey != "" && pl.Params["width"] == "" && pl.Params["height"] == "" {
		pixels = maxDiskImagePixels
	}
	if width > maxDiskImagePixels || height > maxDiskImagePixels || pixels <= 0 || pixels > maxDiskImagePixels {
		return 0, 0, fmt.Errorf("ai: output dimensions exceed the bounded disk-admission estimate")
	}
	if pixels > math.MaxInt64/4 || pixels*4 > math.MaxInt64/variations {
		return 0, 0, fmt.Errorf("ai: requested variations exceed the bounded disk-admission estimate")
	}
	outputBytes = pixels * 4 * variations
	// PNG noise can approach its decoded byte size. Reserve 25% encoding and
	// publication overhead in addition to the retained result set.
	if outputBytes > math.MaxInt64/5*4 {
		return 0, 0, fmt.Errorf("ai: output disk estimate overflow")
	}
	outputBytes = outputBytes * 5 / 4
	inputCount := int64(0)
	if pl.InputKey != "" {
		inputCount++
	}
	if pl.MaskKey != "" {
		inputCount++
	}
	for _, adapter := range pl.Adapters {
		if adapter.ConditioningImageKey != "" {
			inputCount++
		}
	}
	if inputCount > math.MaxInt64/(64<<20) {
		return 0, 0, fmt.Errorf("ai: input count exceeds the bounded disk estimate")
	}
	inputBytes := inputCount * (64 << 20)
	if inputCount > 0 {
		if maxDiskImagePixels > math.MaxInt64/4/inputCount {
			return 0, 0, fmt.Errorf("ai: input disk estimate overflow")
		}
		decodedInputBytes := inputCount * maxDiskImagePixels * 4
		if inputBytes > math.MaxInt64-decodedInputBytes {
			return 0, 0, fmt.Errorf("ai: combined input disk estimate overflow")
		}
		inputBytes += decodedInputBytes
	}
	// Hold encoded/decoded inputs, one output under construction, and an extra
	// intermediate image while the runner publishes every variation.
	const tempOverhead = int64(128 << 20)
	if inputBytes > math.MaxInt64-tempOverhead || outputBytes > (math.MaxInt64-tempOverhead-inputBytes)/2 {
		return 0, 0, fmt.Errorf("ai: temporary disk estimate overflow")
	}
	tempBytes = inputBytes + outputBytes*2 + tempOverhead
	if outputBytes > imageToolsDiskPlanningCeiling-tempBytes {
		return 0, 0, fmt.Errorf("ai: generated-output job peak exceeds the %d-byte owner planning ceiling", imageToolsDiskPlanningCeiling)
	}
	return outputBytes, tempBytes, nil
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
	mu                  sync.Mutex
	ctx                 context.Context
	stops               []func()
	finish              []func(context.Context) error
	releases            []func(context.Context) error
	diskRelease         func(context.Context) error
	diskStopRenew       func()
	diskLeaseID         string
	diskCleanupVerified bool
}

func newLeaseGroup(ctx context.Context) *leaseGroup {
	return &leaseGroup{ctx: ctx, diskCleanupVerified: true}
}

func (g *leaseGroup) renew(kind string, lease MemoryLease) {
	scope := sharedcapacity.Maintain(g.ctx, lease)
	g.ctx = scope.Context()
	release := func(ctx context.Context) error {
		if err := scope.Release(ctx); err != nil {
			return fmt.Errorf("ai: release %s capacity: %w", kind, err)
		}
		return nil
	}
	if kind == "disk job" {
		g.diskRelease = release
		g.diskStopRenew = scope.StopRenew
		if disk, ok := lease.(sharedcapacity.DiskJobLease); ok {
			g.diskLeaseID = disk.Reservation().ID
		}
		return
	}
	g.releases = append(g.releases, release)
}

func (g *leaseGroup) release(ctx context.Context) error {
	return g.releaseWithDisk(ctx, true)
}

func (g *leaseGroup) releaseInference(ctx context.Context) error {
	return g.releaseWithDisk(ctx, false)
}

func (g *leaseGroup) markDiskCleanupVerified(verified bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.diskCleanupVerified = verified
}

func (g *leaseGroup) releaseWithDisk(ctx context.Context, includeDisk bool) error {
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
	if includeDisk && g.diskRelease != nil {
		if !g.diskCleanupVerified {
			if g.diskStopRenew != nil {
				g.diskStopRenew()
				g.diskStopRenew = nil
			}
			errs = append(errs, fmt.Errorf("%w: disk job lease %s remains active", ErrDiskOutputCleanupUnverified, g.diskLeaseID))
		} else if err := g.diskRelease(ctx); err != nil {
			errs = append(errs, err)
		} else {
			g.diskRelease = nil
			g.diskStopRenew = nil
		}
	}
	return errors.Join(errs...)
}
