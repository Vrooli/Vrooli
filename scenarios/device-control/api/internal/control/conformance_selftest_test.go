package control

import (
	"context"
	"testing"

	devicedomain "device-control/internal/devices"
	strategyregistry "device-control/strategy/registry"
	"github.com/stretchr/testify/require"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestAndroidCapabilitySelfTestUnavailableResultEmitsPhysicalTargetVerdict(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "fake", Serial: "serial-1", HostNodeID: "host-1", Kind: "physical"})
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "failed", result.Disposition)
	require.NotNil(t, result.Verdict)
	require.Equal(t, commonv1.DeviceKind_DEVICE_KIND_PHYSICAL, result.Verdict.Target.DeviceKind)
	require.Contains(t, result.Verdict.Detail, "device_id=fake")
	require.Contains(t, result.Verdict.Detail, "serial=serial-1")
	require.Contains(t, result.Verdict.Detail, "host_node_id=host-1")
}

func TestAndroidCapabilitySelfTestRefreshesBridgeInventoryBeforeLease(t *testing.T) {
	reader := staticAttachedReader{devices: []AttachedDevice{{
		ID: "bridge-android-1", Name: "Galaxy", HostNodeID: "host-1", Kind: "android",
		Transport: "usb", Serial: "serial-1", OSVersion: "13", TrustState: "trusted", Reachability: "reachable",
	}}}
	svc := NewWithAttached(strategyregistry.New(), reader)
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "bridge-android-1", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Equal(t, "serial-1", result.Serial)
	require.Equal(t, "host-1", result.HostNodeID)
}

func TestAndroidCapabilitySelfTestAcceptsEmulatorKind(t *testing.T) {
	svc, _ := testService(t)
	svc.devices.Upsert(devicedomain.Record{ID: "emulator-1", Kind: "emulator", Serial: "emulator-1", HostNodeID: "host-1"})
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "emulator-1", "operator", "")
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestAndroidCapabilitySelfTestReportsCapabilityGap(t *testing.T) {
	svc, _ := testService(t)
	result, err := svc.RunAndroidCapabilitySelfTest(context.Background(), "fake", "operator", "")
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.Disposition)
	require.Contains(t, result.Reason, "not present in device-control inventory")
	require.Empty(t, result.Chapters)
}
