package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"strings"
	"testing"
	"time"
)

type observationFixtureEffects struct {
	clock                                                         time.Time
	expireAfterMarker, expireAfterChild                           bool
	markerErr, childErr, reportErr, cleanupErr                    error
	markerCalls, childCalls, reportCalls, waitCalls, cleanupCalls int
	reportBytes                                                   []byte
}

func (f *observationFixtureEffects) markers(probeprotocol.Observation) error {
	f.markerCalls++
	if f.expireAfterMarker {
		f.clock = f.clock.Add(5 * time.Minute)
	}
	return f.markerErr
}
func (f *observationFixtureEffects) child(context.Context, probeprotocol.Observation) (int, func() error, error) {
	f.childCalls++
	if f.expireAfterChild {
		f.clock = f.clock.Add(5 * time.Minute)
	}
	return 200, func() error { f.cleanupCalls++; return f.cleanupErr }, f.childErr
}
func (f *observationFixtureEffects) report(raw []byte) error {
	f.reportCalls++
	f.reportBytes = append([]byte{}, raw...)
	return f.reportErr
}
func (f *observationFixtureEffects) parentPID() int                  { return 100 }
func (f *observationFixtureEffects) wait(context.Context, time.Time) { f.waitCalls++ }
func observationFixture() (probeprotocol.Observation, time.Time) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	return probeprotocol.Observation{Repository: "/approved/repository", CanaryID: strings.Repeat("a", 32), WorkspaceSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "workspace"), DeniedSHA256: probeprotocol.MarkerDigest(strings.Repeat("a", 32), "denied"), Deadline: now.Add(5 * time.Minute)}, now
}
func TestFixedFirstObservationUsesOnlyPrivateMarkerProcessAbstractions(t *testing.T) {
	o, now := observationFixture()
	f := &observationFixtureEffects{clock: now}
	if e := observe(context.Background(), o, now, f); e != nil {
		t.Fatal(e)
	}
	if f.markerCalls != 1 || f.childCalls != 1 || f.cleanupCalls != 1 || f.waitCalls != 1 || f.reportCalls != 1 || len(f.reportBytes) > probeprotocol.MaxReportBytes {
		t.Fatal("fixed bounded effect flow changed", f)
	}
	var r probeprotocol.Report
	if json.Unmarshal(f.reportBytes, &r) != nil || r.ParentPID != 100 || r.ChildPID != 200 || !r.WorkspaceReadable || !r.MarkerReadDenied || r.NetworkObservation != "not-observed" || !r.Deadline.Equal(o.Deadline) {
		t.Fatal("observation report changed", r)
	}
}
func TestFixedFirstObservationFailuresPreserveCleanupAndNoLaterEffects(t *testing.T) {
	for _, name := range []string{"expired", "marker", "child", "report", "cleanup", "marker-expiry", "child-expiry"} {
		t.Run(name, func(t *testing.T) {
			o, now := observationFixture()
			f := &observationFixtureEffects{clock: now}
			switch name {
			case "expired":
				o.Deadline = now
			case "marker":
				f.markerErr = errors.New("canary unreadable")
			case "child":
				f.childErr = errors.New("child unavailable")
			case "report":
				f.reportErr = errors.New("report unavailable")
			case "marker-expiry":
				f.expireAfterMarker = true
			case "child-expiry":
				f.expireAfterChild = true
			case "cleanup":
				f.cleanupErr = errors.New("cleanup unavailable")
			}
			if observe(context.Background(), o, now, f) == nil {
				t.Fatal("failure hidden")
			}
			if name == "expired" && (f.markerCalls+f.childCalls+f.reportCalls) != 0 {
				t.Fatal("expired input had effects")
			}
			if (name == "marker" || name == "marker-expiry") && (f.childCalls+f.reportCalls) != 0 {
				t.Fatal("marker refusal started child")
			}
			if name == "child" && f.reportCalls != 0 {
				t.Fatal("child refusal reported success")
			}
			if (name == "report" || name == "cleanup" || name == "child-expiry") && f.cleanupCalls != 1 {
				t.Fatal("descendant cleanup omitted")
			}
		})
	}
}

func (f *observationFixtureEffects) now() time.Time { return f.clock }
