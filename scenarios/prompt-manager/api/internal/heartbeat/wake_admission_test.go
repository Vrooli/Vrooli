package heartbeat

import (
	"context"
	"errors"
	"testing"

	"prompt-manager/internal/store"
	"prompt-manager/internal/teamconfig"
)

type fakeWakeSignals struct {
	signal WakeSignal
	err    error
}

func (f fakeWakeSignals) Snapshot(context.Context, string, string, []string) (WakeSignal, error) {
	return f.signal, f.err
}

type fakeAdmissionState struct {
	state *store.HeartbeatAdmissionState
	err   error
}

func (f *fakeAdmissionState) GetHeartbeatAdmissionState(context.Context, string, string) (*store.HeartbeatAdmissionState, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.state == nil {
		return &store.HeartbeatAdmissionState{Version: 1}, nil
	}
	copy := *f.state
	return &copy, nil
}

func (f *fakeAdmissionState) SetHeartbeatAdmissionState(_ context.Context, _, _ string, state *store.HeartbeatAdmissionState) error {
	if f.err != nil {
		return f.err
	}
	copy := *state
	f.state = &copy
	return nil
}

func TestEvaluateWakeAdmissionIsDeterministic(t *testing.T) {
	policy := &teamconfig.WakeAdmission{
		Mode:          teamconfig.WakeAdmissionOnChange,
		ChangeSources: []string{"team"},
	}
	signal := WakeSignal{Key: "signal-a"}

	tests := []struct {
		name     string
		previous *store.HeartbeatAdmissionState
		want     wakeDecision
	}{
		{name: "first observation", want: wakeDecision{wakeDecisionAdmit, "first-observation"}},
		{name: "unchanged", previous: &store.HeartbeatAdmissionState{LastSignal: "signal-a"}, want: wakeDecision{wakeDecisionQuiet, "signal-unchanged"}},
		{name: "changed", previous: &store.HeartbeatAdmissionState{LastSignal: "signal-old"}, want: wakeDecision{wakeDecisionAdmit, "signal-changed"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := evaluateWakeAdmission(policy, tc.previous, signal); got != tc.want {
				t.Fatalf("decision = %#v, want %#v", got, tc.want)
			}
		})
	}

	if got := evaluateWakeAdmission(nil, &store.HeartbeatAdmissionState{LastSignal: signal.Key}, signal); got.Decision != wakeDecisionAdmit {
		t.Fatalf("legacy policy decision = %#v, want admit", got)
	}
}

func TestRunOrdinaryWakeSkipsUnchangedSignalAndAdmitsChange(t *testing.T) {
	state := &fakeAdmissionState{}
	scheduler := &Scheduler{
		wakeSignals:    fakeWakeSignals{signal: WakeSignal{Key: "signal-a"}},
		admissionState: state,
	}
	config := &store.HeartbeatConfig{WakeAdmission: &teamconfig.WakeAdmission{
		Mode:          teamconfig.WakeAdmissionOnChange,
		ChangeSources: []string{"team"},
	}}
	calls := 0
	execute := func() error { calls++; return nil }

	scheduler.runOrdinaryWake(context.Background(), "team", "agent", config, execute)
	scheduler.runOrdinaryWake(context.Background(), "team", "agent", config, execute)
	if calls != 1 {
		t.Fatalf("unchanged signal caused %d executions, want 1", calls)
	}
	if state.state == nil || state.state.AdmittedCount != 1 || state.state.QuietCount != 1 {
		t.Fatalf("admission counters = %+v, want one admit and one quiet", state.state)
	}

	scheduler.wakeSignals = fakeWakeSignals{signal: WakeSignal{Key: "signal-b"}}
	scheduler.runOrdinaryWake(context.Background(), "team", "agent", config, execute)
	if calls != 2 {
		t.Fatalf("changed signal caused %d executions, want 2", calls)
	}
}

func TestRunOrdinaryWakeDoesNotConsumeFailedExecution(t *testing.T) {
	state := &fakeAdmissionState{}
	scheduler := &Scheduler{
		wakeSignals:    fakeWakeSignals{signal: WakeSignal{Key: "signal-a"}},
		admissionState: state,
	}
	config := &store.HeartbeatConfig{WakeAdmission: &teamconfig.WakeAdmission{
		Mode:          teamconfig.WakeAdmissionOnChange,
		ChangeSources: []string{"team"},
	}}

	if state.state != nil {
		t.Fatal("state unexpectedly initialized")
	}
	scheduler.runOrdinaryWake(context.Background(), "team", "agent", config, func() error {
		return errors.New("queue unavailable")
	})
	if state.state != nil {
		t.Fatal("failed execution consumed the signal")
	}
}

func TestRunOrdinaryWakeFailsOpenWhenEvidenceUnavailable(t *testing.T) {
	scheduler := &Scheduler{
		wakeSignals:    fakeWakeSignals{err: errors.New("read failed")},
		admissionState: &fakeAdmissionState{},
	}
	config := &store.HeartbeatConfig{WakeAdmission: &teamconfig.WakeAdmission{
		Mode:          teamconfig.WakeAdmissionOnChange,
		ChangeSources: []string{"team"},
	}}
	calls := 0
	scheduler.runOrdinaryWake(context.Background(), "team", "agent", config, func() error {
		calls++
		return nil
	})
	if calls != 1 {
		t.Fatalf("evidence failure caused %d executions, want 1", calls)
	}
	if state := scheduler.admissionState.(*fakeAdmissionState).state; state == nil || state.FailOpenCount != 1 || state.LastReason != "fail-open:signal-read" {
		t.Fatalf("fail-open telemetry = %+v", state)
	}
}
