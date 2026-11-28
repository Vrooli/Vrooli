package androidadb

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestDreamStateParsesAndroidDreamManagerSummary(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		wantState string
	}{
		{"active", "mCurrentDreamToken=Binder{abc}\nmCurrentDreamName=com.example/.Dream\nmCurrentDreamIsTest=false\nmCurrentDreamIsWaking=false", "active"},
		{"preview", "mCurrentDreamToken=Binder{abc}\nmCurrentDreamName=com.example/.Dream\nmCurrentDreamIsTest=true\nmCurrentDreamIsWaking=false", "preview"},
		{"waking", "mCurrentDreamToken=Binder{abc}\nmCurrentDreamName=com.example/.Dream\nmCurrentDreamIsTest=false\nmCurrentDreamIsWaking=true", "waking"},
		{"inactive", "mCurrentDreamToken=null", "inactive"},
		{"unknown missing token", "DreamManagerService state", "unknown"},
		{"unknown missing flags", "mCurrentDreamToken=Binder{abc}", "unknown"},
		{"unknown malformed flag", "mCurrentDreamToken=Binder{abc}\nmCurrentDreamIsTest=maybe\nmCurrentDreamIsWaking=false", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, component := dreamState(tt.output)
			require.Equal(t, tt.wantState, state)
			if tt.wantState == "active" || tt.wantState == "preview" || tt.wantState == "waking" {
				require.Equal(t, "com.example/.Dream", component)
			}
		})
	}
}

func TestReadStateReportsDreamStateAndUnknownWhenUnsupported(t *testing.T) {
	base := map[string][]byte{
		"adb -s tv shell dumpsys activity activities":                []byte("mResumedActivity: ActivityRecord{u0 com.example/.Main t1}"),
		"adb -s tv shell dumpsys power":                              []byte("Display Power: state=ON"),
		"adb -s tv shell dumpsys window":                             []byte("mShowingLockscreen=false"),
		"adb -s tv shell dumpsys input":                              []byte("mCurrentRotation=0"),
		"adb -s tv shell settings get system user_rotation":          []byte("0"),
		"adb -s tv shell settings get system accelerometer_rotation": []byte("1"),
		"adb -s tv shell dumpsys battery":                            []byte("level: 80\nAC powered: true"),
		"adb -s tv shell dumpsys thermalservice":                     []byte("Thermal Status: nominal"),
		"adb -s tv shell wm size":                                    []byte("Physical size: 1920x1080"),
		"adb -s tv shell wm density":                                 []byte("Physical density: 240"),
	}
	active := make(map[string][]byte, len(base)+1)
	for key, value := range base {
		active[key] = value
	}
	active["adb -s tv shell dumpsys dreams"] = []byte("mCurrentDreamToken=Binder{abc}\nmCurrentDreamName=com.dreamx/.Dream\nmCurrentDreamIsTest=false\nmCurrentDreamIsWaking=false")
	state, err := NewWithRunner(&scriptedRunner{responses: active}, "tv").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "active", state.DreamState)
	require.Equal(t, "com.dreamx/.Dream", state.DreamComponent)
	require.NotContains(t, state.Unavailable, "dream_state")

	unsupported := make(map[string][]byte, len(base))
	for key, value := range base {
		unsupported[key] = value
	}
	state, err = NewWithRunner(&scriptedRunner{responses: unsupported}, "tv").ReadState(context.Background())
	require.NoError(t, err)
	require.Equal(t, "unknown", state.DreamState)
	require.Contains(t, state.Unavailable, "dream_state")
}

type blockedDreamRunner struct{}

func (blockedDreamRunner) Run(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDreamStateProbeHasItsOwnDeadline(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	started := time.Now()
	state, component, unavailable := probeDreamState(parent, blockedDreamRunner{}, []string{"-s", "tv", "shell", "dumpsys", "dreams"}, 10*time.Millisecond)
	require.Less(t, time.Since(started), 250*time.Millisecond)
	require.Equal(t, "unknown", state)
	require.Empty(t, component)
	require.True(t, strings.Contains(unavailable, "dumpsys dreams"))
	require.NoError(t, parent.Err(), "probe timeout must not cancel the enclosing state read")
}

func TestDeviceStateDreamFieldsAreSerialized(t *testing.T) {
	payload, err := json.Marshal(strategy.DeviceState{DreamState: "active", DreamComponent: "com.dreamx/.Dream"})
	require.NoError(t, err)
	require.JSONEq(t, `{"dream_state":"active","dream_component":"com.dreamx/.Dream"}`, string(payload))
}

var _ strategy.StateReader = (*Adapter)(nil)
