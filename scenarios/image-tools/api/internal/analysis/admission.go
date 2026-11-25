package analysis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"image-tools/internal/jobs"
	"image-tools/internal/models"
)

type admittedOperation struct{}

// admittedThreads carries the whole CPU threads admission reserved, so the run
// starts its process with exactly that many and never more.
type admittedThreads struct{}

// cpuThreadsByOperation is the CPU a model-backed analysis op reserves and then
// uses. PaddleOCR reads each image at several detector scales and the largest
// takes ~8 s on 8 threads versus ~50 s on one, so it reserves four cores; every
// other op is light enough for one. Admission and the sidecar thread cap read this
// one table, so the reservation can never differ from what the process uses.
var cpuThreadsByOperation = map[string]int{OpOCR: 4}

func cpuThreadsFor(op string) int {
	if threads, ok := cpuThreadsByOperation[op]; ok && threads > 0 {
		return threads
	}
	return 1
}

// admittedModel carries the model admission acquired, so the run uses exactly it.
type admittedModel struct{}

// Admit is used by the common job scheduler before analysis starts. Nested
// auto-scans use the same boundary after their parent inference releases CPU.
func (s *Service) Admit(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	var scope *sharedcapacity.Scope
	var releaseUses []func()
	var once sync.Once
	// The use gate releases once and Scope.Release is retry-safe, so a contended
	// ledger call is retried by the scheduler rather than failing finished work.
	admission := jobs.Admission{Context: ctx, ReleaseRetryable: true}
	admission.Release = func(cleanup context.Context) error {
		once.Do(func() {
			for i := len(releaseUses) - 1; i >= 0; i-- {
				releaseUses[i]()
			}
		})
		if scope != nil {
			return scope.Release(cleanup)
		}
		return nil
	}
	ram := int64(256 << 20)
	if job.Operation == OpOCR {
		ram = 512 << 20
	}
	threads := cpuThreadsFor(job.Operation)
	var chosen *models.Model
	if job.Operation == OpNSFW || job.Operation == OpOCR {
		var modelIDs []string
		if job.Operation == OpNSFW {
			modelIDs = s.profile.ClassifierIDs()
			if len(modelIDs) == 0 {
				modelIDs = []string{"falconsai-nsfw"}
			}
		} else {
			selected, err := s.selectOperationModel(ctx, OpOCR)
			if err != nil {
				return admission, err
			}
			modelID := selected.ID
			chosen = &selected
			modelIDs = []string{modelID}
		}
		for _, modelID := range modelIDs {
			if s.acquireModel != nil {
				// Acquire (or revalidate weights already on disk) before the shared
				// use lease: the installer needs the model's exclusive lease and a
				// download or smoke must not hold capacity. An installed model that is
				// not runtime-ready takes the same path the AI engine uses.
				installed := s.modelInstalled(modelID)
				needsAcquire := !installed
				if installed && s.modelReady != nil {
					ready, _ := s.modelReady(ctx, modelID)
					needsAcquire = !ready
				}
				if needsAcquire {
					if err := s.acquireModel(ctx, modelID, job.ID, jobs.AdmissionProgress(ctx)); err != nil {
						if errors.Is(err, jobs.ErrCapacityBusy) {
							return admission, err
						}
						if errors.Is(err, sharedcapacity.ErrAssetInUse) {
							return admission, fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
						}
						if installed {
							return admission, fmt.Errorf("%w: model %q is not runtime-ready and could not be revalidated: %v", ErrBackendUnavailable, modelID, err)
						}
						return admission, fmt.Errorf("%w: model %q is not installed and could not be acquired: %v", ErrBackendUnavailable, modelID, err)
					}
				}
			}
			if s.acquireModelUse != nil {
				releaseUse, err := s.acquireModelUse(modelID)
				if errors.Is(err, sharedcapacity.ErrAssetInUse) {
					err = fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
				}
				if err != nil {
					return admission, err
				}
				if releaseUse == nil {
					return admission, errors.New("analysis: model admission returned no release function")
				}
				releaseUses = append(releaseUses, releaseUse)
			}
			if !s.modelInstalled(modelID) {
				return admission, fmt.Errorf("%w: model %q not installed — run `image-tools models install %s`", ErrBackendUnavailable, modelID, modelID)
			}
			if s.modelReady != nil {
				if ready, reason := s.modelReady(ctx, modelID); !ready {
					return admission, fmt.Errorf("%w: model %s: %s", ErrBackendUnavailable, modelID, reason)
				}
			}
			if s.modelRAMBytes != nil {
				modelRAM, err := s.modelRAMBytes(modelID)
				if err != nil {
					return admission, err
				}
				if job.Operation == OpOCR {
					ram = modelRAM
				} else {
					ram += modelRAM
				}
			}
		}
	}
	if ram <= 0 {
		return admission, errors.New("analysis: a positive model RAM budget is required")
	}
	if s.acquireCapacity != nil {
		var err error
		scope, err = s.acquireCapacity(ctx, sharedcapacity.Request{OwnerID: "image-tools:" + job.ID + ":analysis", RAMBytes: ram, CPUMilli: int64(threads) * 1000, Priority: string(job.Priority), NoReclaim: true})
		if err != nil {
			var unavailable *sharedcapacity.UnavailableError
			if errors.As(err, &unavailable) {
				err = fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
			}
			return admission, err
		}
		if scope == nil {
			return admission, errors.New("analysis: capacity admission returned no lease")
		}
		ctx = scope.Context()
		admission.Meta = map[string]string{"ram_capacity_claim_bytes": strconv.FormatInt(ram, 10), "cpu_capacity_milli": strconv.Itoa(threads * 1000)}
	}
	admission.Context = context.WithValue(context.WithValue(ctx, admittedOperation{}, job.Operation), admittedThreads{}, threads)
	if chosen != nil {
		admission.Context = context.WithValue(admission.Context, admittedModel{}, *chosen)
	}
	return admission, nil
}

func (s *Service) execution(ctx context.Context, op string) (context.Context, func() error, error) {
	if ctx.Value(admittedOperation{}) == op {
		return ctx, func() error { return nil }, nil
	}
	for {
		admission, err := s.Admit(ctx, jobs.Job{ID: "scan", Operation: op, Priority: jobs.PriorityService})
		finish := func() error { return admission.Release(context.WithoutCancel(ctx)) }
		if err == nil {
			return admission.Context, finish, nil
		}
		if cleanupErr := finish(); cleanupErr != nil {
			return ctx, nil, errors.Join(err, cleanupErr)
		}
		if !errors.Is(err, jobs.ErrCapacityBusy) {
			return ctx, nil, err
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx, nil, context.Cause(ctx)
		case <-timer.C:
		}
	}
}
