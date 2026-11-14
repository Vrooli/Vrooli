package androidadb

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPackageJavaCrashSelectionDoesNotLeakOtherProcessOrPIDReuse(t *testing.T) {
	data := "E/AndroidRuntime( 10): FATAL EXCEPTION: main\nE/AndroidRuntime( 10): Process: com.example, PID: 10\nE/AndroidRuntime( 10): java.lang.IllegalStateException: wanted\nE/AndroidRuntime( 20): FATAL EXCEPTION: main\nE/AndroidRuntime( 20): Process: com.examples, PID: 20\nE/AndroidRuntime( 20): private foreign data\nE/AndroidRuntime( 10): FATAL EXCEPTION: main\nE/AndroidRuntime( 10): Process: other.app, PID: 10\nE/AndroidRuntime( 10): reused pid secret\n"
	out := string(packageJavaCrashes(data, "com.example"))
	require.Contains(t, out, "wanted")
	require.NotContains(t, out, "foreign")
	require.NotContains(t, out, "secret")
	require.NotContains(t, out, "com.examples")
	require.Empty(t, packageJavaCrashes(data, "missing.app"))
}

func TestAccessibilityRuntimeCrashedAndBindingAreScoped(t *testing.T) {
	data := "User state[\nattributes:{id=0,}\nBinding services:{ComponentInfo{com.example/.Svc}}\nCrashed services:{ComponentInfo{other.app/.Svc}}\n}]"
	require.True(t, *accessibilityServiceSet(data, "0", "com.example/com.example.Svc", "Binding"))
	require.False(t, *accessibilityServiceSet(data, "0", "com.example/com.example.Svc", "Crashed"))
	require.Nil(t, accessibilityServiceSet(data, "10", "com.example/com.example.Svc", "Binding"))
}
