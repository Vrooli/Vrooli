package control

import (
	"context"
	"sync"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	strategyregistry "device-control/strategy/registry"
)

type volumeCastFixture struct {
	mu    sync.Mutex
	value float64
}

func (f *volumeCastFixture) ID() string { return "google-cast" }
func (f *volumeCastFixture) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{StrategyID: f.ID(), Status: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapProperty: {Name: strategy.CapProperty, Status: strategy.StatusAvailable}}, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", Writable: true}}}, nil
}
func (f *volumeCastFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{"volume": {Value: f.value, Status: strategy.StatusAvailable, SourceTransport: f.ID(), StateDomain: "receiver_volume", ObservedAt: time.Now().UTC(), Confidence: .75}}}, nil
}
func (f *volumeCastFixture) GetProperty(ctx context.Context, name string) (any, error) {
	state, err := f.ReadState(ctx)
	return state.Properties[name].Value, err
}
func (f *volumeCastFixture) SetProperty(_ context.Context, set strategy.PropertySet) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if value, ok := set.Value.(float64); ok {
		f.value = value
	}
	return nil
}

type volumeRemoteFixture struct {
	calls    []strategy.MediaCommand
	failOnce bool
}

type volumeStateFixture struct {
	id         string
	value      float64
	confidence float64
}

func (f *volumeStateFixture) ID() string { return f.id }
func (f *volumeStateFixture) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{StrategyID: f.id, Status: strategy.StatusAvailable, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", StateClass: strategy.StateBearing, StateDomain: "physical_output"}}}, nil
}
func (f *volumeStateFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Properties: map[string]strategy.PropertyValue{"volume": {Value: f.value, Status: strategy.StatusAvailable, SourceTransport: f.id, StateDomain: "physical_output", Confidence: f.confidence}}}, nil
}

func (f *volumeRemoteFixture) ID() string { return "android-tv-remote" }
func (f *volumeRemoteFixture) Describe(context.Context) (strategy.Declaration, error) {
	return strategy.Declaration{StrategyID: f.ID(), Status: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapMedia: {Name: strategy.CapMedia, Status: strategy.StatusAvailable}}}, nil
}
func (f *volumeRemoteFixture) ReadState(context.Context) (strategy.DeviceState, error) {
	return strategy.DeviceState{Unavailable: map[string]string{"state": "remote does not expose state"}}, nil
}
func (f *volumeRemoteFixture) ControlMedia(_ context.Context, command strategy.MediaCommand) error {
	f.calls = append(f.calls, command)
	if f.failOnce {
		f.failOnce = false
		return ErrVolumeSessionLost
	}
	return nil
}

func TestParseVolumeIntentDistinguishesRelativePercentage(t *testing.T) {
	intent, err := ParseVolumeIntent(VolumeRequest{Goal: "Turn down the TV volume by 50 percent"})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Kind != "relative" || intent.ChangeKind != "fraction_of_current" || intent.Value != -0.5 {
		t.Fatalf("unexpected relative intent: %#v", intent)
	}
	if intent.Repeat != 5 {
		t.Fatalf("expected bounded five-key approximation, got %d", intent.Repeat)
	}
}

func TestParseVolumeIntentDistinguishesAbsolutePercentage(t *testing.T) {
	intent, err := ParseVolumeIntent(VolumeRequest{Goal: "set the TV volume to 50%"})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Kind != "absolute" || intent.ChangeKind != "absolute_normalized" || intent.Value != 0.5 {
		t.Fatalf("unexpected absolute intent: %#v", intent)
	}
}

func TestParseVolumeIntentRejectsAmbiguousPercentAndOutOfRange(t *testing.T) {
	for _, request := range []VolumeRequest{{Goal: "volume 50 percent"}, {Goal: "lower volume by 120%"}, {Goal: "set volume to 2"}} {
		if _, err := ParseVolumeIntent(request); err == nil {
			t.Fatalf("expected rejection for %#v", request)
		}
	}
}

func TestPlanVolumeUsesRemoteForRelativeAndCastForState(t *testing.T) {
	zero, one := 0.0, 1.0
	state := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"volume": {Value: 0.42, Status: strategy.StatusAvailable, SourceTransport: "cast", StateDomain: "receiver_volume", ObservedAt: time.Now().UTC(), Confidence: 0.75},
	}}
	intent := VolumeIntent{Kind: "relative", ChangeKind: "fraction_of_current", Value: -0.5, Direction: "down", Repeat: 5, Target: "physical_output"}
	record := devicedomain.Record{ID: "tv", Kind: "physical", Transports: []strategy.DeviceTransport{
		{StrategyID: "google-cast", Name: "cast", Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapProperty: {Name: strategy.CapProperty, Status: strategy.StatusAvailable}}, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", Writable: true, Minimum: &zero, Maximum: &one}}},
		{StrategyID: "android-tv-remote", Name: "remote", Role: "relative-key-actuation", Operations: []string{"volume-up", "volume-down", "mute"}, Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapMedia: {Name: strategy.CapMedia, Status: strategy.StatusAvailable}}},
	}}
	plan, err := PlanVolume(record, state, intent)
	if err != nil {
		t.Fatal(err)
	}
	if plan.StateTransport != "cast" || plan.ActionTransport != "remote" || plan.Repeat != 5 {
		t.Fatalf("unexpected transport plan: %#v", plan)
	}
	if plan.AbsoluteSetpoint != nil || plan.Intent.Value != -0.5 {
		t.Fatalf("relative plan was converted to an absolute request: %#v", plan)
	}
}

func TestPlanVolumeUsesMasterSetterForObservedArcEarcOutput(t *testing.T) {
	zero, one := 0.0, 1.0
	state := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"volume": {Value: 0.08, Status: strategy.StatusAvailable, Reason: "Android TV Remote volume source: arc-earc-amplifier", SourceTransport: "android-tv-remote", StateDomain: "physical_output", ObservedAt: time.Now().UTC(), Confidence: 0.8},
	}}
	intent := VolumeIntent{Kind: "relative", ChangeKind: "fraction_of_current", Value: -0.5, Direction: "down", Repeat: 5, Target: "physical_output"}
	record := devicedomain.Record{ID: "tv", Kind: "physical", Transports: []strategy.DeviceTransport{
		{StrategyID: "google-cast", Name: "cast", Health: strategy.StatusAvailable, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", Writable: true, Minimum: &zero, Maximum: &one}}},
		{StrategyID: "android-tv-remote", Name: "remote", Role: "relative-key-actuation", Operations: []string{"volume-up", "volume-down"}, Health: strategy.StatusAvailable},
	}}
	plan, err := PlanVolume(record, state, intent)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ActionTransport != "cast" || plan.Action != "property:volume" || plan.AbsoluteSetpoint == nil {
		t.Fatalf("unexpected ARC/eARC plan: %#v", plan)
	}
	if *plan.AbsoluteSetpoint != 0.04 {
		t.Fatalf("expected 4%% setpoint, got %v", *plan.AbsoluteSetpoint)
	}
	if plan.Repeat != 0 {
		t.Fatalf("setter plan should not retain remote repeat count: %#v", plan)
	}
}

func TestPlanVolumeRejectsAbsoluteRemoteOnly(t *testing.T) {
	intent := VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: 0.5, Target: "physical_output"}
	record := devicedomain.Record{ID: "tv", Kind: "physical", Transports: []strategy.DeviceTransport{{StrategyID: "android-tv-remote", Name: "remote", Role: "relative-key-actuation", Operations: []string{"volume-up", "volume-down"}, Health: strategy.StatusAvailable}}}
	if _, err := PlanVolume(record, strategy.DeviceState{}, intent); err == nil {
		t.Fatal("numeric absolute volume must be rejected without Cast")
	}
}

func TestPlanVolumeRejectsReceiverVolumeForPhysicalOutput(t *testing.T) {
	zero, one := 0.0, 1.0
	record := devicedomain.Record{ID: "device-1", Kind: "physical", Transports: []strategy.DeviceTransport{{
		StrategyID: "receiver-provider", Name: "receiver", Health: strategy.StatusAvailable,
		Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", Writable: true, Minimum: &zero, Maximum: &one, StateDomain: "receiver_volume"}},
	}}}
	intent := VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: .5, Target: "physical_output"}
	if _, err := PlanVolume(record, strategy.DeviceState{}, intent); err == nil {
		t.Fatal("receiver volume must not be presented as physical output")
	}
}

func TestPlanVolumeUsesDeclaredOperationsForArbitraryTransports(t *testing.T) {
	zero, one := 0.0, 1.0
	record := devicedomain.Record{ID: "device-1", Kind: "physical", Transports: []strategy.DeviceTransport{
		{StrategyID: "state-provider", Name: "property-path", Health: strategy.StatusAvailable, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", Writable: true, Minimum: &zero, Maximum: &one}}},
		{StrategyID: "step-provider", Name: "step-path", Health: strategy.StatusAvailable, Operations: []string{"volume-up", "volume-down"}},
	}}
	intent := VolumeIntent{Kind: "relative", ChangeKind: "direction", Direction: "down", Repeat: 1, Target: "receiver_volume"}
	plan, err := PlanVolume(record, strategy.DeviceState{}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ActionTransport != "step-path" {
		t.Fatalf("planner selected a strategy-specific or wrong transport: %#v", plan)
	}

	absolute := VolumeIntent{Kind: "absolute", ChangeKind: "absolute_normalized", Value: .5, Target: "receiver_volume"}
	plan, err = PlanVolume(record, strategy.DeviceState{}, absolute)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ActionTransport != "property-path" {
		t.Fatalf("planner ignored the declared writable property: %#v", plan)
	}
}

func TestVolumeVerificationDoesNotClaimPhysicalSuccessFromReceiverState(t *testing.T) {
	setpoint := 0.21
	plan := VolumePlan{AbsoluteSetpoint: &setpoint, StateDomain: "receiver_volume"}
	after := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{"volume": {Value: 0.21, Status: strategy.StatusAvailable, StateDomain: "receiver_volume"}}}
	got := volumeVerification(VolumeIntent{Kind: "absolute", Target: "physical_output"}, plan, strategy.DeviceState{}, after, nil)
	if got != "conflicted" {
		t.Fatalf("receiver-only state must not verify physical output, got %s", got)
	}
}

func TestVolumeVerificationAcceptsAbsolutePhysicalOutputReadback(t *testing.T) {
	intent := VolumeIntent{Kind: "absolute", Target: "physical_output"}
	plan := VolumePlan{Intent: intent, StateDomain: "physical_output", AbsoluteSetpoint: floatPtr(0.04)}
	after := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"volume": {Value: 0.04, Status: strategy.StatusAvailable, StateDomain: "physical_output"},
	}}
	if got := volumeVerification(intent, plan, strategy.DeviceState{}, after, nil); got != "verified" {
		t.Fatalf("expected physical absolute readback to verify, got %q", got)
	}
}

func TestVolumeVerificationChecksRelativeDirectionAgainstFreshState(t *testing.T) {
	intent := VolumeIntent{Kind: "relative", Direction: "down", Target: "physical_output"}
	plan := VolumePlan{Intent: intent, StateDomain: "physical_output"}
	before := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"volume": {Value: 0.08, Status: strategy.StatusAvailable, StateDomain: "physical_output"},
	}}
	after := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{
		"volume": {Value: 0.02, Status: strategy.StatusAvailable, StateDomain: "physical_output"},
	}}
	if got := volumeVerification(intent, plan, before, after, nil); got != "verified" {
		t.Fatalf("expected a witnessed relative decrease to verify, got %s", got)
	}
	after.Properties["volume"] = strategy.PropertyValue{Value: 0.08, Status: strategy.StatusAvailable, StateDomain: "physical_output"}
	if got := volumeVerification(intent, plan, before, after, nil); got != "conflicted" {
		t.Fatalf("expected an unchanged relative volume to conflict, got %s", got)
	}
}

func TestStatePropertyConflictReportsDisagreementAcrossVolumeTransports(t *testing.T) {
	reason, conflict := statePropertyConflict("volume", strategy.PropertyValue{
		Value: 0.65, Status: strategy.StatusAvailable, SourceTransport: "android-tv-remote", StateDomain: "physical_output",
	}, strategy.PropertyValue{
		Value: 0, Status: strategy.StatusAvailable, SourceTransport: "google-cast", StateDomain: "physical_output",
	})
	if !conflict {
		t.Fatal("expected conflicting physical volume observations")
	}
	if reason != "volume observations disagree: android-tv-remote=0.650, google-cast=0.000" {
		t.Fatalf("unexpected conflict reason: %q", reason)
	}
}

func TestReadDeviceStateRetainsHigherConfidenceVolumeAndReportsConflict(t *testing.T) {
	remote := &volumeStateFixture{id: "remote-state", value: 0.08, confidence: 0.8}
	cast := &volumeStateFixture{id: "cast-state", value: 0, confidence: 0.75}
	zero, one := 0.0, 1.0
	svc := New(strategyregistry.New(remote, cast))
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "tv", Kind: "physical", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{
		{StrategyID: remote.ID(), Name: "remote", Health: strategy.StatusAvailable, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", StateClass: strategy.StateBearing, StateDomain: "physical_output", Minimum: &zero, Maximum: &one}}},
		{StrategyID: cast.ID(), Name: "cast", Health: strategy.StatusAvailable, Properties: []strategy.PropertyDescriptor{{Name: "volume", ValueType: "number", StateClass: strategy.StateBearing, StateDomain: "physical_output", Minimum: &zero, Maximum: &one}}},
	}})

	state, err := svc.ReadDeviceState(context.Background(), "tv")
	if err != nil {
		t.Fatal(err)
	}
	if state.Properties["volume"].Value != 0.08 || state.Properties["volume"].SourceTransport != remote.ID() {
		t.Fatalf("higher-confidence remote observation was not retained: %#v", state.Properties["volume"])
	}
	if state.Conflicts["volume"] != "volume observations disagree: remote-state=0.080, cast-state=0.000" && state.Conflicts["volume"] != "volume observations disagree: cast-state=0.000, remote-state=0.080" {
		t.Fatalf("unexpected aggregate conflict: %#v", state.Conflicts)
	}
}

func TestExecuteVolumeOwnsLeaseSelectsRemoteAndReturnsHonestVerification(t *testing.T) {
	cast := &volumeCastFixture{value: .42}
	remote := &volumeRemoteFixture{}
	svc := New(strategyregistry.New(cast, remote))
	svc.devices.Upsert(devicedomain.Record{ID: "tv", Name: "Living TV", Kind: "physical", StrategyID: "google-cast", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{
		{StrategyID: "google-cast", Name: "cast", Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapProperty: {Name: strategy.CapProperty, Status: strategy.StatusAvailable}}},
		{StrategyID: "android-tv-remote", Name: "remote", Role: "relative-key-actuation", Operations: []string{"volume-up", "volume-down", "mute"}, Health: strategy.StatusAvailable, Capabilities: map[string]strategy.Capability{strategy.CapMedia: {Name: strategy.CapMedia, Status: strategy.StatusAvailable}}},
	}})
	result, err := svc.ExecuteVolume(context.Background(), VolumeRequest{Device: "tv", Actor: "test", Goal: "turn down by 50 percent", OperationID: "op-fixed"})
	if err != nil {
		t.Fatal(err)
	}
	if result.OperationID != "op-fixed" || result.Plan.ActionTransport != "remote" || len(remote.calls) != 5 {
		t.Fatalf("unexpected operation: %#v calls=%d", result, len(remote.calls))
	}
	if result.VerificationClass != "conflicted" || result.Status != "ok" {
		t.Fatalf("unchanged relative volume must be reported as conflicted: %#v", result)
	}
	if len(svc.ListLiveSessions()) != 0 {
		t.Fatal("semantic operation leaked its internal lease")
	}
	audits := svc.Audit()
	if len(audits) != 1 || audits[0].OperationID != "op-fixed" || audits[0].Verb != "semantic-volume" {
		t.Fatalf("missing operation audit: %#v", audits)
	}
}

func TestExecuteVolumeRetriesOnceAfterTypedSessionLoss(t *testing.T) {
	cast := &volumeCastFixture{value: .42}
	remote := &volumeRemoteFixture{failOnce: true}
	svc := New(strategyregistry.New(cast, remote))
	svc.devices.Upsert(devicedomain.Record{ID: "tv", Name: "Living TV", Kind: "physical", StrategyID: "google-cast", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Transports: []strategy.DeviceTransport{
		{StrategyID: "google-cast", Name: "cast", Health: strategy.StatusAvailable},
		{StrategyID: "android-tv-remote", Name: "remote", Role: "relative-key-actuation", Operations: []string{"volume-up", "volume-down", "mute"}, Health: strategy.StatusAvailable},
	}})
	result, err := svc.ExecuteVolume(context.Background(), VolumeRequest{Device: "tv", Actor: "test", Goal: "down", OperationID: "op-retry"})
	if err != nil {
		t.Fatal(err)
	}
	if result.RecoveryAttempts != 1 || len(remote.calls) != 2 || result.OperationID != "op-retry" {
		t.Fatalf("expected one recovery retry with stable identity: %#v calls=%d", result, len(remote.calls))
	}
	if len(svc.Audit()) != 1 || svc.Audit()[0].OperationID != "op-retry" {
		t.Fatalf("expected one terminal semantic audit, got %#v", svc.Audit())
	}
}
