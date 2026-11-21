package androidadb

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

type frameworkLogRunner struct {
	*scriptedRunner
	output []byte
	call   string
	limit  int
	err    error
}

func TestApplicationLogsAreBoundedAndScoped(t *testing.T) {
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{responses: map[string][]byte{"adb -s serial shell pidof com.example.app": []byte("123\n")}}, output: []byte("example log")}
	adapter := NewWithRunner(runner, "serial")
	output, err := adapter.ReadApplicationLogs(context.Background(), "com.example.app")
	require.NoError(t, err)
	require.Equal(t, []byte("example log"), output)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 500 --pid 123", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	_, err = adapter.ReadApplicationLogs(context.Background(), "--all")
	require.Error(t, err)
}

func (r *frameworkLogRunner) RunBounded(_ context.Context, name string, limit int, args ...string) ([]byte, error) {
	r.call = strings.Join(append([]string{name}, args...), " ")
	r.limit = limit
	return r.output, r.err
}

func TestFrameworkLogsFiltersAndSummarizesWithoutRawContent(t *testing.T) {
	component := "com.example/.service.Home"
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte(strings.Join([]string{
		"W ActivityManager: Permission Denial com.example/.service.Home private-secret",
		"W ActivityManager: Unable to start service com.example/com.example.service.Home",
		"W ActivityManager: com.example/.service.HomeExtra private-secret",
		"W ActivityManager: xcom.example/.service.Home private-secret",
		"W ActivityManager: com.other/.service.Home private-secret",
	}, "\n"))}
	var output []byte
	err := NewWithRunner(runner, "serial").Actuate(context.Background(), strategy.Actuation{Action: "framework-logs", Component: component, LogTag: "ActivityManager", Output: &output})
	require.NoError(t, err)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 250 -s ActivityManager:W", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	require.NotContains(t, string(output), "private-secret")
	var summary struct {
		Matching int            `json:"matching_line_count"`
		Priority string         `json:"priority"`
		Classes  map[string]int `json:"diagnostic_classes"`
	}
	require.NoError(t, json.Unmarshal(output, &summary))
	require.Equal(t, 2, summary.Matching)
	require.Equal(t, "W", summary.Priority, "omitted priority keeps the warning-level default")
	require.Equal(t, 1, summary.Classes["permission_denial"])
	require.Equal(t, 1, summary.Classes["service_start_failure"])
}

func TestFrameworkLogsInfoMinimumIncludesHigherLevelsAndStaysRedacted(t *testing.T) {
	component := "com.example/.service.Home"
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte(strings.Join([]string{
		"I ActivityManager: Binding service com.example/.service.Home private-secret",
		"W ActivityManager: Unable to start service com.example/com.example.service.Home private-secret",
		"I ActivityManager: Binding service com.example/.service.HomeExtra private-secret",
	}, "\n"))}
	var output []byte
	err := NewWithRunner(runner, "serial").Actuate(context.Background(), strategy.Actuation{
		Action: "framework-logs", Component: component, LogTag: "ActivityManager", LogPriority: "I", Output: &output,
	})
	require.NoError(t, err)
	require.Equal(t, "adb -s serial logcat -d -v epoch -t 250 -s ActivityManager:I", runner.call)
	require.Equal(t, 256*1024, runner.limit)
	require.NotContains(t, string(output), "private-secret", "raw log text must not enter persisted evidence")
	var summary struct {
		Component string `json:"component"`
		Tag       string `json:"tag"`
		Priority  string `json:"priority"`
		Matching  int    `json:"matching_line_count"`
	}
	require.NoError(t, json.Unmarshal(output, &summary))
	require.Equal(t, component, summary.Component)
	require.Equal(t, "ActivityManager", summary.Tag)
	require.Equal(t, "I", summary.Priority)
	require.Equal(t, 2, summary.Matching, "the INFO minimum also includes higher-priority warnings")
}

func TestFrameworkLogsRejectsInvalidFiltersAndReportsNoMatch(t *testing.T) {
	runner := &frameworkLogRunner{scriptedRunner: &scriptedRunner{}, output: []byte("unrelated line")}
	adapter := NewWithRunner(runner, "serial")
	for _, event := range []strategy.Actuation{
		{Action: "framework-logs", Component: "com.example/.Home;evil", LogTag: "ActivityManager"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "OtherTag"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "ActivityManager", LogPriority: "D"},
		{Action: "framework-logs", Component: "com.example/.Home", LogTag: "ActivityManager", LogPriority: "i"},
	} {
		require.Error(t, adapter.Actuate(context.Background(), event))
		require.Empty(t, runner.call)
	}
	var output []byte
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "framework-logs", Component: "com.example/.Home", LogTag: "AccessibilityManagerService", Output: &output}))
	require.Contains(t, string(output), "inconclusive_no_matching_component_lines")
}
