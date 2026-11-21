package control

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
	"device-control/strategy/fakes"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

func TestFlowHTTPWriteTimeoutOutlivesBoundedFlowExecution(t *testing.T) {
	flowBudget := time.Duration(maxFlowDurationMS) * time.Millisecond
	require.Greater(t, FlowHTTPWriteTimeout(), flowBudget)
	require.Equal(t, time.Minute, FlowHTTPWriteTimeout()-flowBudget)
}

func TestFlowLifecycleUsesTypedAdapterInsteadOfGenericInput(t *testing.T) {
	fake := fakes.New("flow-lifecycle", strategy.StatusAvailable, strategy.CapAppLifecycle)
	svc := New(strategyregistry.New(fake))
	svc.devices.Upsert(devicedomain.Record{ID: "flow-lifecycle", StrategyID: "flow-lifecycle", Transport: "usb", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})
	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "launch", Kind: "launch", Target: "com.example.editor", RequiredCapabilities: []string{strategy.CapAppLifecycle}}}}, "flow-lifecycle", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, fake.LifecycleCalls(), 1)
	require.Equal(t, "launch", fake.LifecycleCalls()[0].Operation)
	require.Empty(t, fake.Calls())
}

func TestFlowReportsWhetherStepOrRunDeadlineStoppedLifecycle(t *testing.T) {
	tests := []struct {
		name            string
		flowTimeoutMS   int64
		stepTimeoutMS   int64
		wantDisposition string
		wantMessage     string
	}{
		{name: "step deadline first", flowTimeoutMS: 1000, stepTimeoutMS: 200, wantDisposition: "failed", wantMessage: "command failed (signal: killed) after context ended: context deadline exceeded"},
		{name: "run deadline first", flowTimeoutMS: 200, stepTimeoutMS: 1000, wantDisposition: "cancelled", wantMessage: "flow stopped: context deadline exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := fakes.New("deadline-lifecycle", strategy.StatusAvailable, strategy.CapAppLifecycle)
			fake := &blockingLifecycleFake{Strategy: base, deadlines: make(chan time.Time, 1)}
			svc := New(strategyregistry.New(fake))
			svc.devices.Upsert(devicedomain.Record{ID: "deadline-lifecycle", StrategyID: fake.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable, Capabilities: []strategy.Capability{{Name: strategy.CapAppLifecycle, Status: strategy.StatusAvailable}}})

			result, err := svc.Run(context.Background(), Flow{
				MaxDurationMS: tt.flowTimeoutMS,
				Steps:         []Step{{ID: "launch", Kind: "launch", Target: "com.example.editor", TimeoutMS: tt.stepTimeoutMS, RequiredCapabilities: []string{strategy.CapAppLifecycle}}},
			}, "deadline-lifecycle", "operator")
			require.NoError(t, err)
			require.Equal(t, tt.wantDisposition, result.Disposition)
			require.Len(t, result.Chapters, 1)
			require.Contains(t, result.Chapters[0].Message, tt.wantMessage)
			select {
			case deadline := <-fake.deadlines:
				require.False(t, deadline.IsZero())
			case <-time.After(time.Second):
				t.Fatal("lifecycle operation did not receive its effective context deadline")
			}
		})
	}
}

func TestRequestStorageUsesRoutedDatabaseAndFileRoots(t *testing.T) { // [REQ:DVC-P0-011]
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	routed, err := coredb.Open(context.Background(), coredb.Config{
		Driver:       coredb.DriverSQLite,
		DSN:          primaryPath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })

	primaryRoots := storage.Paths{DataDir: t.TempDir()}
	roots := filerouting.New(primaryRoots)
	fake := fakes.New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	svc, err := NewWithDB(strategyregistry.New(fake), routed, roots)
	require.NoError(t, err)

	leaseID := "device-control-routing-test"
	testCtx := coredb.WithTestMode(context.Background())
	testDBPath := filepath.Join(t.TempDir(), "test.db")
	require.NoError(t, routed.InstallTestPool(context.Background(), testDBPath, leaseID, time.Minute))
	testRoots := storage.Paths{DataDir: t.TempDir()}
	require.NoError(t, roots.InstallTestRoots(testRoots, leaseID, time.Minute))
	t.Cleanup(func() { _ = routed.ClearTestPool(leaseID) })
	t.Cleanup(func() { _ = roots.ClearTestRoots(leaseID) })

	_, err = routed.ExecContext(testCtx, `CREATE TABLE device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
 )`)
	require.NoError(t, err)

	session, err := svc.AcquireContext(testCtx, "fake", "routing-test", time.Minute)
	require.NoError(t, err)
	require.Len(t, svc.ListSessionsContext(testCtx), 1)
	require.Empty(t, svc.ListSessions(), "test-mode session must not leak into the primary database")

	require.NoError(t, svc.persistArtifact(testCtx, "routed-artifact", []byte("redacted"), "log"))
	artifact, kind, err := svc.ArtifactContext(testCtx, "routed-artifact")
	require.NoError(t, err)
	require.Equal(t, "log", kind)
	require.Equal(t, []byte("redacted"), artifact)
	require.Equal(t, int64(1), roots.LeaseStats().TestRootWrites)
	_, err = os.Stat(filepath.Join(testRoots.DataDir, "evidence", "routed-artifact.bin"))
	require.NoError(t, err)

	_, _ = svc.ReleaseContext(testCtx, session.ID)
}

func TestRunRetainsRedactedCaptureAndTapCoordinates(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{ID: "flow-1", Steps: []Step{
		{ID: "capture", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "tap", Kind: "tap", Target: "12,34", RequiredCapabilities: []string{strategy.CapInput}},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, result.Evidence, 1)
	require.NotEmpty(t, result.Evidence[0].SHA256)
	require.Greater(t, result.Evidence[0].SizeBytes, int64(0))
	artifactPath := svc.artifacts[result.Evidence[0].ID]
	contents, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.Len(t, contents, int(result.Evidence[0].SizeBytes))
	fake, ok := svc.registry.Get("fake")
	require.True(t, ok)
	actuator := fake.(*fakes.Strategy)
	calls := actuator.Calls()
	require.Len(t, calls, 1)
	require.InDelta(t, 12, calls[0].Pointer.X, 0)
	require.InDelta(t, 34, calls[0].Pointer.Y, 0)
}

func TestPropertyActuationPublishesLocalStateChangeWithSharedCausation(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	property := fakes.NewPropertyOnly("property-device", strategy.PropertyDescriptor{
		Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum,
	}, 20.0)
	db, err := sql.Open("sqlite", "file:control-test-property-events-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	svc, err := NewWithDB(strategyregistry.New(property), db)
	require.NoError(t, err)
	subscription := svc.SubscribeStateChanges(1)
	t.Cleanup(subscription.Cancel)

	result, err := svc.Run(context.Background(), Flow{Steps: []Step{{
		ID: "set-brightness", Kind: "property-set", RequiredCapabilities: []string{strategy.CapProperty},
		Arguments: map[string]any{"name": "brightness", "value": 40.0},
	}}}, "property-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)

	select {
	case event := <-subscription.Events:
		require.Equal(t, "property-device", event.DeviceID)
		require.Equal(t, "brightness", event.Attribute)
		require.Equal(t, 20.0, event.OldValue)
		require.Equal(t, 40.0, event.NewValue)
		require.Equal(t, strategy.StateBearing, event.StateClass)
		require.NotEmpty(t, event.CausationID)
		require.NotEmpty(t, event.ObservedAt)
		require.NotEmpty(t, svc.Audit())
		require.Equal(t, event.CausationID, svc.Audit()[0].CausationID)
	case <-time.After(time.Second):
		t.Fatal("property actuation did not publish a local state-change event")
	}
}

func TestSensorReadPublishesOnlyObservedTransitions(t *testing.T) {
	sensor := fakes.NewSensorOnly("sensor-rest", strategy.SensorReading{Name: "temperature", Value: 21.0, StateClass: strategy.StateBearing})
	svc := New(strategyregistry.New(sensor))
	svc.devices.UpsertIdentity(devicedomain.Record{ID: "sensor-device", Kind: "physical", Serial: "sensor-serial", StrategyID: sensor.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	subscription := svc.SubscribeStateChanges(2)
	t.Cleanup(subscription.Cancel)
	flow := Flow{Transport: "rest", Steps: []Step{{ID: "read", Kind: "sensor-read", RequiredCapabilities: []string{strategy.CapSensor}}}}

	result, err := svc.Run(context.Background(), flow, "sensor-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	select {
	case event := <-subscription.Events:
		t.Fatalf("first sensor observation unexpectedly emitted event: %+v", event)
	default:
	}

	sensor.Readings[0].Value = 22.5
	result, err = svc.Run(context.Background(), flow, "sensor-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	select {
	case event := <-subscription.Events:
		require.Equal(t, "sensor-device", event.DeviceID)
		require.Equal(t, "rest", event.Transport)
		require.Equal(t, "temperature", event.Attribute)
		require.Equal(t, 21.0, event.OldValue)
		require.Equal(t, 22.5, event.NewValue)
		require.Equal(t, strategy.StateBearing, event.StateClass)
		require.NotEmpty(t, event.CausationID)
	case <-time.After(time.Second):
		t.Fatal("sensor transition did not publish a local state-change event")
	}
}

func TestFlowContractBindsRunAndEnforcesConditionsAndIdempotency(t *testing.T) {
	property := fakes.NewPropertyOnly("contract-property", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Writable: true, StateClass: strategy.StateBearing}, 20.0)
	svc := New(strategyregistry.New(property))
	svc.devices.Upsert(devicedomain.Record{ID: "contract-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow := Flow{
		Transport:           "rest",
		ApplicationID:       "editor",
		ApplicationRevision: "editor-v1",
		MaxDurationMS:       5000,
		Steps: []Step{
			{ID: "set-volume", Kind: "property-set", Arguments: map[string]any{"name": "volume", "value": 30.0}, IdempotencyKey: "volume-change", Postconditions: []Condition{{Kind: "property-equals", Target: "volume", Expected: 30.0}}},
			{ID: "duplicate-volume", Kind: "property-set", Arguments: map[string]any{"name": "volume", "value": 40.0}, IdempotencyKey: "volume-change"},
		},
	}
	result, err := svc.Run(context.Background(), flow, "contract-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, "contract-device", result.Binding.DeviceID)
	require.Equal(t, "rest", result.Binding.Transport)
	require.Equal(t, "editor", result.Binding.ApplicationID)
	require.Equal(t, "editor-v1", result.Binding.ApplicationRevision)
	require.NotEmpty(t, result.Binding.LeaseID)
	require.Len(t, property.Changes, 1)
	require.Equal(t, 30.0, property.Values["volume"])
	require.Contains(t, result.Chapters[1].Message, "deduplicated")
}

func TestFlowIdempotencyReceiptSurvivesServiceReconstruction(t *testing.T) {
	db, err := sql.Open("sqlite", "file:idempotency-restart-"+t.Name()+"?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	property := fakes.NewPropertyOnly("durable-retry-property", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Writable: true}, 20.0)
	registry := strategyregistry.New(property)
	svc, err := NewWithDB(registry, db)
	require.NoError(t, err)
	svc.devices.Upsert(devicedomain.Record{ID: "durable-retry-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow := Flow{Transport: "rest", Steps: []Step{{ID: "set-volume", Kind: "property-set", IdempotencyKey: "durable-volume-change", Arguments: map[string]any{"name": "volume", "value": 30.0}}}}
	first, err := svc.Run(context.Background(), flow, "durable-retry-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", first.Disposition)

	restarted, err := NewWithDB(strategyregistry.New(property), db)
	require.NoError(t, err)
	restarted.devices.Upsert(devicedomain.Record{ID: "durable-retry-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	flow.Steps[0].Arguments["value"] = 40.0
	second, err := restarted.Run(context.Background(), flow, "durable-retry-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", second.Disposition)
	require.Contains(t, second.Chapters[0].Message, "prior command receipt")
	require.Empty(t, property.Changes[1:])
	require.Equal(t, 30.0, property.Values["volume"])
}

func TestFlowContractRejectsFailedPreconditionBeforeActuation(t *testing.T) {
	property := fakes.NewPropertyOnly("precondition-property", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Writable: true, StateClass: strategy.StateBearing}, 20.0)
	svc := New(strategyregistry.New(property))
	svc.devices.Upsert(devicedomain.Record{ID: "precondition-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "set-volume", Kind: "property-set", Arguments: map[string]any{"name": "volume", "value": 30.0}, Preconditions: []Condition{{Kind: "property-equals", Target: "volume", Expected: 10.0}}}}}, "precondition-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Empty(t, property.Changes)
	require.Contains(t, result.Chapters[0].Message, "precondition")
}

func TestFlowContractRequiresObservationWhenRequested(t *testing.T) {
	property := fakes.NewPropertyOnly("observation-property", strategy.PropertyDescriptor{Name: "volume", ValueType: "number", Writable: true}, 20.0)
	svc := New(strategyregistry.New(property))
	svc.devices.Upsert(devicedomain.Record{ID: "observation-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "set-volume", Kind: "property-set", ObservationRequired: true, Arguments: map[string]any{"name": "volume", "value": 30.0}}}}, "observation-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Empty(t, property.Changes)
	require.Contains(t, result.Chapters[0].Message, "prior observation")
}

func TestFlowContractRetriesOnlyRepeatableObservationSteps(t *testing.T) {
	property := &transientPropertyStrategy{PropertyOnly: fakes.NewPropertyOnly("retry-property", strategy.PropertyDescriptor{Name: "volume", ValueType: "number"}, 20.0), failures: 1}
	svc := New(strategyregistry.New(property))
	svc.devices.Upsert(devicedomain.Record{ID: "retry-device", StrategyID: property.ID(), Transport: "rest", Status: strategy.StatusAvailable, Health: strategy.StatusAvailable})
	result, err := svc.Run(context.Background(), Flow{Transport: "rest", Steps: []Step{{ID: "verify-volume", Kind: "property-assert", RetryBudget: 1, Arguments: map[string]any{"name": "volume", "equals": 20.0}}}}, "retry-device", "operator")
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Equal(t, 0, property.failures)
}

func TestRunRejectsUnredactedCaptureWithoutActor(t *testing.T) {
	svc, _ := testService(t)
	_, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "")
	require.ErrorContains(t, err, "requires an actor")
}

func TestUnredactedCaptureIsAuditedWithActor(t *testing.T) {
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{AllowUnredactedCapture: true, Steps: []Step{{ID: "capture", Kind: "observe"}}}, "fake", "owner-1")
	require.NoError(t, err)
	require.Len(t, result.Evidence, 1)
	require.True(t, result.Evidence[0].OptedOut)
	records := svc.Audit()
	require.NotEmpty(t, records)
	require.Equal(t, "owner-1", records[0].Actor)
	require.True(t, records[0].RedactionOptedOut)
}

func TestLocalCaptureDefaultsUnredactedAndExplicitRedactionIsRecorded(t *testing.T) {
	svc, _ := testService(t)
	defaultResult, err := svc.Run(context.Background(), Flow{Steps: []Step{{ID: "default-screen", Kind: "observe"}}}, "fake", "owner-1")
	require.NoError(t, err)
	require.Len(t, defaultResult.Evidence, 1)
	defaultRef := defaultResult.Evidence[0]
	require.False(t, defaultRef.RedactionVerified)
	require.Contains(t, defaultRef.AppliedRules, "redaction_not_requested")
	stored, kind, err := svc.Artifact(defaultRef.ID)
	require.NoError(t, err)
	require.Equal(t, "image", kind)
	require.Equal(t, defaultRef.SizeBytes, int64(len(stored)))

	redactedResult, err := svc.Run(context.Background(), Flow{RedactSensitiveContent: true, Steps: []Step{{ID: "redacted-screen", Kind: "observe", Arguments: map[string]any{"sensitive_regions": []any{[]any{float64(0), float64(0), float64(1), float64(1)}}}}}}, "fake", "owner-1")
	require.NoError(t, err)
	require.Len(t, redactedResult.Evidence, 1)
	redactedRef := redactedResult.Evidence[0]
	require.True(t, redactedRef.RedactionVerified)
	require.Contains(t, redactedRef.AppliedRules, "content_classification_not_performed")
	require.NotContains(t, redactedRef.AppliedRules, "full_frame_private_content")
}

func TestRunReusesHeldLeaseTokenWithoutConflict(t *testing.T) { // [REQ:DVC-P0-004]
	svc, _ := testService(t)
	session, err := svc.Acquire("fake", "operator", time.Minute)
	require.NoError(t, err)
	result, err := svc.RunWithLease(context.Background(), Flow{Steps: []Step{{ID: "wait", Kind: "wait"}}}, "fake", "operator", session.LeaseToken)
	require.NoError(t, err)
	require.Equal(t, "passed", result.Disposition)
	require.Len(t, svc.ListSessions(), 1)
	require.Equal(t, "held", svc.ListSessions()[0].State)
	_, err = svc.Release(session.ID)
	require.NoError(t, err)
}

func TestFrameDifferenceFailsWhenActuationIsSuppressed(t *testing.T) { // [REQ:DVC-P0-008]
	svc, _ := testService(t)
	result, err := svc.Run(context.Background(), Flow{SuppressActuation: true, Steps: []Step{
		{ID: "before", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "actuate", Kind: "key", Target: "KEYCODE_APP_SWITCH", RequiredCapabilities: []string{strategy.CapInput}},
		{ID: "after", Kind: "observe", RequiredCapabilities: []string{strategy.CapScreenshot}},
		{ID: "changed", Kind: "assert-frame-different"},
	}}, "fake", "operator")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.Contains(t, result.Chapters[len(result.Chapters)-1].Message, "identical")
}
