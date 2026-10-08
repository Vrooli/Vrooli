package main

import (
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// prepareSessionEnvironment applies the same validated workspace and tool
// environment used by coding-agent launchers to Web Console shells. This keeps
// direct shortcuts such as `codex` from depending on the API service's launch
// directory or shell profile.
func prepareSessionEnvironment(base []string, workingDir string) ([]string, string, error) {
	context, err := cliutil.ResolveLaunchContext(cliutil.LaunchContextRequest{
		WorkingDir:  workingDir,
		Environment: base,
	})
	if err != nil {
		return nil, "", err
	}
	return cliutil.PrepareLaunchEnvironment(base, context), context.WorkingDir, nil
}

func sessionEnvironmentValue(environment []string, key string) string {
	prefix := key + "="
	for _, entry := range environment {
		if value, ok := strings.CutPrefix(entry, prefix); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
