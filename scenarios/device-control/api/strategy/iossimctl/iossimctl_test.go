package iossimctl

import (
	"context"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestNewWithUDIDKeepsResolvedConfiguration(t *testing.T) {
	adapter := NewWithUDID(" simulator-123 ")
	require.Equal(t, "simulator-123", adapter.udid)
}

func TestObserveWithoutSelectedUDIDFailsClosed(t *testing.T) {
	frame, err := New().Observe(context.Background())
	require.Empty(t, frame)
	var unavailable *strategy.AvailabilityError
	require.ErrorAs(t, err, &unavailable)
	require.Contains(t, unavailable.Reason, "no iOS simulator UDID is selected")
}

func TestSelectBootedDeviceRequiresExactPresentBootedUDID(t *testing.T) {
	listed := simctlDeviceList{Devices: map[string][]simctlDeviceInfo{
		"iOS 18.0": {
			{UDID: "booted-1", Name: "iPhone Test", State: "Booted"},
			{UDID: "shutdown-1", Name: "iPhone Other", State: "Shutdown"},
		},
	}}

	selected, err := selectBootedDevice(listed, "booted-1")
	require.NoError(t, err)
	require.Equal(t, "booted-1", selected)

	_, err = selectBootedDevice(listed, "shutdown-1")
	var unavailable *strategy.AvailabilityError
	require.ErrorAs(t, err, &unavailable)
	require.Contains(t, unavailable.Reason, "not booted")

	_, err = selectBootedDevice(listed, "missing-1")
	require.ErrorAs(t, err, &unavailable)
	require.Contains(t, unavailable.Reason, "not present")
}
