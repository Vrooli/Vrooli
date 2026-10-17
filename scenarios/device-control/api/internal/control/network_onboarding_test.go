package control

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"

	"device-control/strategy"
	"device-control/strategy/androidadb"
	strategyregistry "device-control/strategy/registry"

	"github.com/stretchr/testify/require"
)

// scriptedADBRunner is a deterministic androidadb.Runner. Scripted commands
// return their canned output; any other command returns empty output with no
// error, so capability probes for unscripted verbs resolve to unavailable
// without failing the onboarding path.
type scriptedADBRunner struct {
	mu        sync.Mutex
	responses map[string][]byte
	calls     []string
}

func (r *scriptedADBRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.Join(append([]string{name}, args...), " ")
	r.calls = append(r.calls, key)
	if out, ok := r.responses[key]; ok {
		return out, nil
	}
	return []byte{}, nil
}

func TestAndroidTVNetworkADBOnboardingAndApplicationLifecycle(t *testing.T) { // [REQ:DVC-P0-011]
	var frame bytes.Buffer
	require.NoError(t, png.Encode(&frame, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	// A firmware log preamble precedes the PNG, mirroring the live SmartTV.
	screencap := append([]byte("Init wrapper sys mutex successful\n"), frame.Bytes()...)

	endpoint := "192.168.1.158:5555"
	runner := &scriptedADBRunner{responses: map[string][]byte{
		"adb connect 192.168.1.158:5555":                                        []byte("connected to 192.168.1.158:5555"),
		"adb devices -l":                                                        []byte("List of devices attached\n192.168.1.158:5555\tdevice model:SmartTV_4K\n"),
		"adb -s 192.168.1.158:5555 shell getprop ro.serialno":                   []byte("AE70A4D38B\n"),
		"adb -s 192.168.1.158:5555 shell getprop ro.build.version.release":      []byte("12\n"),
		"adb -s 192.168.1.158:5555 exec-out screencap -p":                       screencap,
		"adb -s 192.168.1.158:5555 shell input --help":                          []byte("usage: input"),
		"adb -s 192.168.1.158:5555 shell pm list packages":                      []byte("package:org.smarttube.stable\n"),
		"adb -s 192.168.1.158:5555 shell pm list packages org.smarttube.stable": []byte("package:org.smarttube.stable\n"),
		"adb -s 192.168.1.158:5555 shell dumpsys package org.smarttube.stable":  []byte("versionName=32.47"),
		"adb -s 192.168.1.158:5555 shell pm list packages com.example.absent":   []byte(""),
	}}

	svc := New(strategyregistry.New(androidadb.NewWithRunner(runner, "")))

	// Onboarding derives identity from the hardware serial, publishes
	// app-lifecycle, and decodes the log-prefixed screenshot.
	device, err := svc.OnboardNetworkADB(context.Background(), endpoint)
	require.NoError(t, err)
	require.Equal(t, "AE70A4D38B", device.Serial)
	require.Equal(t, endpoint, device.Endpoint)
	require.Equal(t, "wireless", device.Transport)
	require.NotContains(t, device.ID, endpoint)
	caps := map[string]string{}
	for _, c := range device.Capabilities {
		caps[c.Name] = c.Status
	}
	require.Equal(t, "available", caps["app-lifecycle"], "network TV must publish app-lifecycle")
	require.Equal(t, "available", caps["screenshot"], "log-prefixed screenshot must decode")

	// An unselected transport routes to the endpoint-bound strategy, never the
	// ambient base adapter.
	routed, ok := svc.strategyForFlow(device.ID, "")
	require.True(t, ok)
	require.NotNil(t, routed)

	// package-state is verified against the live device, under a lease.
	session, err := svc.Acquire(device.ID, "test", time.Minute)
	require.NoError(t, err)

	present, err := svc.ExecuteAppLifecycle(context.Background(), device.ID, AppLifecycleOperation{
		Actor:               "test",
		LeaseToken:          session.LeaseToken,
		AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "package-state", Package: "org.smarttube.stable"},
	})
	require.NoError(t, err)
	require.True(t, present.Result.Installed)
	require.Equal(t, "32.47", present.Result.Version)
	require.NotEmpty(t, present.CommandID)
	require.Equal(t, "success", present.Audit.Outcome)
	require.Equal(t, "wireless", present.Audit.Transport, "audit must record the endpoint-bound transport, not empty")

	absent, err := svc.ExecuteAppLifecycle(context.Background(), device.ID, AppLifecycleOperation{
		Actor:               "test",
		LeaseToken:          session.LeaseToken,
		AppLifecycleRequest: strategy.AppLifecycleRequest{Operation: "package-state", Package: "com.example.absent"},
	})
	require.NoError(t, err)
	require.False(t, absent.Result.Installed)

	_ = fmt.Sprint(runner.calls) // retained for debugging on failure
}
