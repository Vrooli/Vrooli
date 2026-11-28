package androidadb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLockStateParsesVendorKeyguardFormats(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{name: "explicit legacy field", output: "mShowingLockscreen=true", want: "locked"},
		{name: "spaced explicit field", output: "isKeyguardShowing: false", want: "unlocked"},
		{name: "samsung showing state", output: "mShowingState=SHOWING", want: "locked"},
		{name: "samsung not showing state", output: "mShowingState: NOT_SHOWING", want: "unlocked"},
		{name: "delegate showing", output: "KeyguardDelegate: showing=false, occluded=false", want: "unlocked"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, lockState(test.output))
		})
	}
}

type blockedStateRunner struct{}

func (blockedStateRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestAndroidStateObservationHonorsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	_, err := NewWithRunner(blockedStateRunner{}, "serial-1").ReadState(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestReadStateParsesRecordedAndroidProbes(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 shell getprop ro.build.version.sdk":               []byte("29"),
		"adb -s serial-1 shell getprop service.adb.tcp.port":               []byte("-1"),
		"adb -s serial-1 shell dumpsys activity activities":                []byte("mResumedActivity: ActivityRecord{u0 com.example.hello/.MainActivity t42}"),
		"adb -s serial-1 shell dumpsys power":                              []byte("mWakefulness=Awake\nDisplay Power: state=ON"),
		"adb -s serial-1 shell dumpsys window":                             []byte("mShowingLockscreen=false isStatusBarKeyguard=false"),
		"adb -s serial-1 shell dumpsys dreams":                             []byte("mCurrentDreamToken=null"),
		"adb -s serial-1 shell dumpsys input":                              []byte("mCurrentRotation=3"),
		"adb -s serial-1 shell settings get system user_rotation":          []byte("3"),
		"adb -s serial-1 shell settings get system accelerometer_rotation": []byte("1"),
		"adb -s serial-1 shell dumpsys battery":                            []byte("level: 87\nAC powered: true"),
		"adb -s serial-1 shell dumpsys thermalservice":                     []byte("Thermal Status: nominal"),
		"adb -s serial-1 shell wm size":                                    []byte("Physical size: 1080x2400"),
		"adb -s serial-1 shell wm density":                                 []byte("Physical density: 420"),
	}}
	state, err := NewWithRunner(runner, "serial-1").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "com.example.hello", state.ForegroundPackage)
	require.Equal(t, "on", state.ScreenState)
	require.Equal(t, "unlocked", state.LockState)
	require.Equal(t, "reverse-landscape", state.Orientation)
	require.Equal(t, 87, state.BatteryLevel)
	require.True(t, state.Charging)
	require.Equal(t, "nominal", state.ThermalStatus)
	require.Equal(t, 1080, state.DisplayWidth)
	require.Equal(t, 2400, state.DisplayHeight)
	require.Equal(t, 420, state.DisplayDensity)
	require.Empty(t, state.Unavailable)
}
