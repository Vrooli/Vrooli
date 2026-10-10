package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"time"
)

// Effect seams remain private. Unit tests use only disposable marker/process
// abstractions; the production helper has no arbitrary executable or target.
type observationEffects interface {
	markers(probeprotocol.Observation) error
	child(context.Context, probeprotocol.Observation) (int, func() error, error)
	report([]byte) error
	parentPID() int
	now() time.Time
	wait(context.Context, time.Time)
}

func observe(ctx context.Context, o probeprotocol.Observation, now time.Time, effects observationEffects) (result error) {
	if effects == nil || ctx.Err() != nil || o.Validate(now) != nil {
		return errors.New("fixed observation refused")
	}
	if effects.markers(o) != nil {
		return errors.New("fixed observation marker check refused")
	}
	if ctx.Err() != nil || o.Validate(effects.now()) != nil {
		return errors.New("fixed observation expired before descendant")
	}
	child, cleanup, e := effects.child(ctx, o)
	if e != nil || child <= 0 || cleanup == nil {
		return errors.New("fixed observation descendant refused")
	}
	defer func() {
		if cleanup() != nil {
			result = errors.New("fixed descendant cleanup unconfirmed")
		}
	}()
	if ctx.Err() != nil || o.Validate(effects.now()) != nil {
		return errors.New("fixed observation expired before report")
	}
	report := probeprotocol.Report{Event: "observed", Deadline: o.Deadline, ParentPID: effects.parentPID(), ChildPID: child, WorkspaceReadable: true, MarkerReadDenied: true, NetworkObservation: "not-observed"}
	raw, e := json.Marshal(report)
	if e != nil || len(raw) > probeprotocol.MaxReportBytes || effects.report(append(raw, '\n')) != nil {
		return errors.New("fixed observation report refused")
	}
	effects.wait(ctx, o.Deadline)
	return nil
}
