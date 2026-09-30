package fakes

import (
	"context"
	"testing"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestFloorOnlyStrategyHasNoScreenOrInputFloorRequirement(t *testing.T) {
	s := NewFloorOnly("floor-only")
	report := strategy.Verify(context.Background(), s)
	require.Empty(t, report.Failed)
	require.Contains(t, report.Passed, "screenless-floor")
	require.NotContains(t, strategy.StepKinds(s.Declaration), "observe")
	require.NotContains(t, strategy.StepKinds(s.Declaration), "tap")
}

func TestPropertyOnlyValidationStopsOutOfRangeValueBeforeTransport(t *testing.T) {
	minimum, maximum := 0.0, 100.0
	s := NewPropertyOnly("light", strategy.PropertyDescriptor{Name: "brightness", ValueType: "number", Writable: true, Minimum: &minimum, Maximum: &maximum}, 50.0)
	property, ok := any(s).(strategy.PropertyActuator)
	require.True(t, ok)
	err := property.SetProperty(context.Background(), strategy.PropertySet{Name: "brightness", Value: 101.0, CausationID: "cause-1"})
	var validation *strategy.PropertyValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, "brightness", validation.Descriptor)
	require.Empty(t, s.Changes)
	require.NoError(t, property.SetProperty(context.Background(), strategy.PropertySet{Name: "brightness", Value: 75.0, CausationID: "cause-2"}))
}

func TestSensorOnlyDeviceDeclaresActuationUnavailable(t *testing.T) {
	s := NewSensorOnly("motion", strategy.SensorReading{Name: "motion", Value: true})
	report := strategy.Verify(context.Background(), s)
	require.Empty(t, report.Failed)
	require.Equal(t, strategy.StatusAvailable, s.Declaration.Capabilities[strategy.CapSensor].Status)
	require.Equal(t, strategy.StatusUnavailable, s.Declaration.Capabilities[strategy.CapInput].Status)
	require.NotEmpty(t, s.Declaration.Capabilities[strategy.CapInput].Reason)
}

func TestStrategyProvidesReplayableFloor(t *testing.T) { // [REQ:DVC-P0-001]
	s := New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	frame, err := s.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, "image/png", frame.MediaType)
	require.NotEmpty(t, frame.Bytes)
	require.NoError(t, s.Actuate(context.Background(), strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: "ENTER"}}))
	require.Len(t, s.Calls(), 1)
}
