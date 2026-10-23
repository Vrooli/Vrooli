package androidadb

import (
	"context"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestAccessibilityStateIsExactReadOnlyConfigurationNotBindingProof(t *testing.T) {
	for _, tc := range []struct {
		name, master, list  string
		configured, enabled bool
	}{
		{"enabled", "1", "com.other/.Service:com.example/.HomeService", true, true},
		{"global-off", "0", "com.example/com.example.HomeService", true, false},
		{"prefix-collision", "1", "com.example/com.example.HomeServiceOther", false, false},
		{"none", "0", "null", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &scriptedRunner{responses: map[string][]byte{
				"adb -s tv shell am get-current-user":                                          []byte("10"),
				"adb -s tv shell settings --user 10 get secure accessibility_enabled":          []byte(tc.master),
				"adb -s tv shell settings --user 10 get secure enabled_accessibility_services": []byte(tc.list),
			}}
			result, err := NewWithRunner(r, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "accessibility-state", Package: "com.example", Value: ".HomeService"})
			require.NoError(t, err)
			require.Equal(t, tc.configured, *result.ServiceConfigured)
			require.Equal(t, tc.enabled, *result.ServiceEnabled)
			require.Equal(t, "10", result.AndroidUser)
			require.Contains(t, result.Reason, "runtime binding")
			for _, forbidden := range []string{" put ", "keyevent", "start", "screencap", "uiautomator", "getevent"} {
				require.NotContains(t, strings.Join(r.calls, "\n"), forbidden)
			}
			require.Nil(t, result.ServiceBound, "an absent runtime dump must preserve unknown binding")
			require.Empty(t, result.Diagnostic, "private dump and other service identities must not be returned")
		})
	}
}

func TestAccessibilityStateRejectsUnknownConfigurationAndUnsafeClass(t *testing.T) {
	r := &scriptedRunner{responses: map[string][]byte{"adb -s tv shell am get-current-user": []byte("0"), "adb -s tv shell settings --user 0 get secure accessibility_enabled": []byte("unknown")}}
	result, err := NewWithRunner(r, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "accessibility-state", Package: "com.example", Value: ".HomeService"})
	require.Error(t, err)
	require.Nil(t, result.ServiceEnabled)
	r.calls = nil
	_, err = NewWithRunner(r, "tv").AppLifecycle(context.Background(), strategy.AppLifecycleRequest{Operation: "accessibility-state", Package: "com.example", Value: ".HomeService;reboot"})
	require.Error(t, err)
	require.Empty(t, r.calls)
}
