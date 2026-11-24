package policybridge

import (
	"context"
	"log"
	"time"
)

const PublishInterval = 30 * time.Minute

// Start refreshes the typed dependency-governance capability while the API is
// live. Publication is opt-in and RunnerSink refuses candidates when the
// installed evaluator cannot prove identities or when actions would change
// without the separate reviewed-rollout switch.
func Start(ctx context.Context, logger *log.Logger) func() {
	if logger == nil {
		logger = log.Default()
	}
	if !PublicationEnabled() {
		logger.Printf("agent-policy SDA snapshot publication withheld: %s is not true", PublishEnabledEnv)
		return func() {}
	}
	workerCtx, cancel := context.WithCancel(ctx)
	go func() {
		publish := func() {
			if err := Publish(workerCtx, RunnerSink{}, time.Now().UTC()); err != nil {
				logger.Printf("agent-policy SDA snapshot not published: %v", err)
				return
			}
			logger.Printf("agent-policy SDA snapshot published after identity and live-action preflight")
		}
		publish()
		ticker := time.NewTicker(PublishInterval)
		defer ticker.Stop()
		for {
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
				publish()
			}
		}
	}()
	return cancel
}
