package androidadb

import (
	"context"
	"strings"
	"time"
)

const dreamStateProbeTimeout = 2 * time.Second

func probeDreamState(ctx context.Context, runner Runner, args []string, timeout time.Duration) (state, component, unavailable string) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	output, err := runner.Run(probeCtx, "adb", args...)
	if err != nil {
		return "unknown", "", "adb " + strings.Join(args, " ")
	}
	state, component = dreamState(string(output))
	if state == "unknown" {
		if len(output) == 0 {
			return state, component, "adb " + strings.Join(args, " ")
		}
		return state, component, "unrecognized adb dumpsys dreams output"
	}
	return state, component, ""
}

// dreamState parses the stable summary fields emitted by Android's
// DreamManagerService dump. It deliberately does not infer a running dream
// from the foreground package.
func dreamState(output string) (state, component string) {
	values := make(map[string]string, 4)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "mCurrentDreamToken", "mCurrentDreamName", "mCurrentDreamIsTest", "mCurrentDreamIsWaking":
			values[key] = strings.TrimSpace(value)
		}
	}
	component = values["mCurrentDreamName"]
	token, hasToken := values["mCurrentDreamToken"]
	if !hasToken || token == "" {
		return "unknown", component
	}
	if token == "null" {
		return "inactive", component
	}
	isTest, testOK := parseDumpBool(values["mCurrentDreamIsTest"])
	isWaking, wakingOK := parseDumpBool(values["mCurrentDreamIsWaking"])
	if !testOK || !wakingOK {
		return "unknown", component
	}
	if isWaking {
		return "waking", component
	}
	if isTest {
		return "preview", component
	}
	return "active", component
}

func parseDumpBool(value string) (bool, bool) {
	switch value {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}
