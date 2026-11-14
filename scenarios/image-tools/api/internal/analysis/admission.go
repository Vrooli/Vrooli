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
)

type admittedOperation struct{}

// Admit is used by the common job scheduler before analysis starts. Nested
// auto-scans use the same boundary after their parent inference releases CPU.
func (s *Service) Admit(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	var scope *sharedcapacity.Scope
	var releaseUse func()
	var once sync.Once
	admission := jobs.Admission{Context: ctx}
	admission.Release = func(cleanup context.Context) error {
		once.Do(func() {
			if releaseUse != nil {
				releaseUse()
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
	if job.Operation == OpNSFW || job.Operation == OpOCR {
		modelID := "adamcodd-vit-nsfw"
		if job.Operation == OpOCR {
			modelID = "tesseract"
		}
		if s.acquireModelUse != nil {
			var err error
			releaseUse, err = s.acquireModelUse(modelID)
			if errors.Is(err, sharedcapacity.ErrAssetInUse) {
				err = fmt.Errorf("%w: %v", jobs.ErrCapacityBusy, err)
			}
			if err != nil {
				return admission, err
			}
			if releaseUse == nil {
				return admission, errors.New("analysis: model admission returned no release function")
			}
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
			var err error
			ram, err = s.modelRAMBytes(modelID)
			if err != nil {
				return admission, err
			}
		}
	}
	if ram <= 0 {
		return admission, errors.New("analysis: a positive model RAM budget is required")
	}
	if s.acquireCapacity != nil {
		var err error
		scope, err = s.acquireCapacity(ctx, sharedcapacity.Request{OwnerID: "image-tools:" + job.ID + ":analysis", RAMBytes: ram, CPUMilli: 1000, Priority: string(job.Priority), NoReclaim: true})
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
		admission.Meta = map[string]string{"ram_capacity_claim_bytes": strconv.FormatInt(ram, 10), "cpu_capacity_milli": "1000"}
	}
	admission.Context = context.WithValue(ctx, admittedOperation{}, job.Operation)
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
