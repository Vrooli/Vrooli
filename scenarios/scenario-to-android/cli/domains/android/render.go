package android

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// renderAPIBody renders an Android ramp API response. With --json the
// response is printed as the JSON document it is, so callers can pipe it to
// jq; human output pretty-prints it under the report's status and next steps.
func renderAPIBody(ctx cliapp.RunContext, body []byte, extraStatus []string, nextSteps ...string) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		trimmed = []byte("{}")
	}
	if ctx.JSON() {
		if json.Valid(trimmed) {
			return cliapp.PrintJSON(ctx.Stdout(), json.RawMessage(trimmed))
		}
		return cliapp.PrintJSON(ctx.Stdout(), map[string]string{"raw": string(trimmed)})
	}
	text := string(trimmed)
	var pretty bytes.Buffer
	if json.Indent(&pretty, trimmed, "", "  ") == nil {
		text = pretty.String()
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: append([]string{text}, extraStatus...), NextSteps: nextSteps})
}

// matrixRunArgs lets every matrix command take the run id as an argument;
// ANDROID_MATRIX_RUN_ID remains the fallback for existing scripts.
var matrixRunArgs = cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "run_id", Description: "Matrix run id (default: ANDROID_MATRIX_RUN_ID)"}}}

var matrixCompareArgs = cliapp.ArgSchema{Positionals: []cliapp.Positional{
	{Name: "run_id", Description: "Current matrix run id (default: ANDROID_MATRIX_RUN_ID)"},
	{Name: "prior_run_id", Description: "Prior matrix run id (default: ANDROID_PRIOR_MATRIX_RUN_ID)"},
}}

var scenarioArgs = cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "scenario", Default: "hello-mobile", Description: "Scenario whose validation records to read"}}}

func matrixRunID(ctx cliapp.RunContext) (string, error) {
	if id := firstNonEmpty(ctx.Positional("run_id"), getenv("ANDROID_MATRIX_RUN_ID")); id != "" {
		return id, nil
	}
	return "", fmt.Errorf("a matrix run id is required: pass <run_id> (from matrix-list) or set ANDROID_MATRIX_RUN_ID")
}

func scenarioGet(core *cliapp.ScenarioApp, path string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		query := url.Values{"scenario": []string{strings.TrimSpace(ctx.Flag("scenario"))}}
		body, err := core.Get(path, query)
		if err != nil {
			return fmt.Errorf("get %s: %w", path, err)
		}
		return renderAPIBody(ctx, body, nil, "Read one run with `scenario-to-android android matrix-get <run_id>`; block on one with `matrix-wait <run_id>`.")
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
