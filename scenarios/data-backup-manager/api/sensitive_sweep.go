package main

import (
	"context"
	"log"
	"time"

	coverageH "data-backup-manager/handlers/coverage"
)

const (
	sensitiveSweepFirst    = 2 * time.Minute
	sensitiveSweepInterval = 6 * time.Hour
)

// runSensitiveSweep applies the operator's standing rule for places that hold
// secrets. With no rule it does nothing, so a secret is never added without the
// operator's recorded consent; with one it registers what is new and lets the
// flow send its weekly note.
func runSensitiveSweep(ctx context.Context, flow *coverageH.SensitiveDecisionFlow, logger *log.Logger) {
	timer := time.NewTimer(sensitiveSweepFirst)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if result, err := flow.Sweep(ctx); err != nil {
				logger.Printf("sensitive coverage standing rule sweep: %v", err)
			} else if len(result.Registered) > 0 || result.NoteSent {
				logger.Printf("sensitive coverage standing rule: added %d, %d failed, note sent %t", len(result.Registered), result.Failed, result.NoteSent)
			}
			timer.Reset(sensitiveSweepInterval)
		}
	}
}
