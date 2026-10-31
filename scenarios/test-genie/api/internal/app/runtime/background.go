package runtime

import (
	"context"
	"sync"
)

// BackgroundCoordinator owns advisory process work. Its jobs are deliberately
// started only by the serving transport, after the listener has been bound.
// A cancelled serving context is the one cancellation authority for every job.
type BackgroundCoordinator struct {
	jobs []func(context.Context)
	once sync.Once
}

func NewBackgroundCoordinator(jobs ...func(context.Context)) *BackgroundCoordinator {
	return &BackgroundCoordinator{jobs: jobs}
}

func (c *BackgroundCoordinator) Start(ctx context.Context) {
	if c == nil || ctx == nil {
		return
	}
	c.once.Do(func() {
		for _, job := range c.jobs {
			if job != nil {
				go job(ctx)
			}
		}
	})
}
