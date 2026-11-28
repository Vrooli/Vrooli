// Package ios exposes the complete operator control surface for the iOS ramp.
package ios

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns commands for every supported operator operation. The API
// remains the owner of business decisions; these handlers only translate HTTP
// requests and render report-shaped responses.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "ios", Description: "Operate the iOS delivery ramp", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "targets", Description: "Probe iOS targets and macOS bridge readiness", NeedsAPI: true, RunCtx: get(core, "/ios/targets")},
		{Name: "generate", Description: "Generate a deterministic Capacitor iOS project", NeedsAPI: true, RunCtx: post(core, "/ios/generate", "IOS_SOURCE_REF")},
		{Name: "build", Description: "Build an iOS artifact on a macOS bridge host", NeedsAPI: true, RunCtx: post(core, "/ios/build", "IOS_SOURCE_REF")},
		{Name: "conformance-plan", Description: "Show the twelve iOS conformance chapters", NeedsAPI: true, RunCtx: get(core, "/ios/conformance-plan")},
		{Name: "readiness", Description: "Show the probed six-rung Apple readiness ladder", NeedsAPI: true, RunCtx: get(core, "/ios/readiness")},
		{Name: "distribution", Description: "Show independent TestFlight, App Store, and ad-hoc channels", NeedsAPI: true, RunCtx: get(core, "/ios/distribution")},
		{Name: "matrix", Description: "Inspect iOS validation matrix readiness", NeedsAPI: true, RunCtx: get(core, "/ios/matrix")},
		{Name: "matrix-catalog", Description: "Show the durable iOS validation catalog", NeedsAPI: true, RunCtx: get(core, "/validation/catalog?scenario=hello-mobile")},
		{Name: "matrix-list", Description: "List durable iOS validation runs", NeedsAPI: true, RunCtx: get(core, "/validation/matrices?scenario=hello-mobile")},
		{Name: "matrix-create", Description: "Create a normal-profile iOS validation matrix", NeedsAPI: true, RunCtx: matrixCreate(core)},
		{Name: "matrix-start", Description: "Start a server-owned iOS validation matrix", NeedsAPI: true, RunCtx: matrixAction(core, "start")},
		{Name: "matrix-wait", Description: "Wait for a server-owned iOS validation matrix", NeedsAPI: true, RunCtx: matrixWait(core)},
		{Name: "compare", Description: "Compare two durable iOS validation matrix runs", NeedsAPI: true, RunCtx: matrixCompare(core)},
		{Name: "matrix-rerun-failed", Description: "Rerun failed iOS validation cells", NeedsAPI: true, RunCtx: matrixRerun(core)},
	}}
}

func get(core *cliapp.ScenarioApp, path string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		body, err := core.Get(path, nil)
		if err != nil {
			return fmt.Errorf("get %s: %w", path, err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Follow the backend disposition and next_action before retrying."}})
	}
}

func post(core *cliapp.ScenarioApp, path, env string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		source := strings.TrimSpace(os.Getenv(env))
		if source == "" {
			return fmt.Errorf("%s is required", env)
		}
		body, err := core.Request("POST", path, nil, map[string]string{"source_ref": source})
		if err != nil {
			return fmt.Errorf("post %s: %w", path, err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Inspect the report-shaped response before proceeding."}})
	}
}

func matrixSelectionRequest(digest, artifactPath string) map[string]any {
	return map[string]any{
		"scenario_name":        "hello-mobile",
		"artifact_digest":      digest,
		"artifact_path":        artifactPath,
		"environment_profiles": []int{1},
		"max_concurrency":      1,
		"metadata": map[string]string{
			"platform": "ios",
		},
	}
}

func matrixCreate(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		digest := strings.TrimSpace(os.Getenv("IOS_ARTIFACT_DIGEST"))
		if digest == "" {
			return fmt.Errorf("IOS_ARTIFACT_DIGEST is required")
		}
		body, err := core.Request("POST", "/validation/matrices", nil, matrixSelectionRequest(digest, strings.TrimSpace(os.Getenv("IOS_ARTIFACT_PATH"))))
		if err != nil {
			return fmt.Errorf("create iOS matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Run vrooli scenario-to-ios ios matrix-start with IOS_MATRIX_RUN_ID from this response."}})
	}
}

func matrixAction(core *cliapp.ScenarioApp, action string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(os.Getenv("IOS_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("IOS_MATRIX_RUN_ID is required")
		}
		body, err := core.Request("POST", "/validation/matrices/"+url.PathEscape(id)+"/"+action, nil, nil)
		if err != nil {
			return fmt.Errorf("iOS matrix %s: %w", action, err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Use ios matrix-wait after the server-owned run completes."}})
	}
}

func matrixWait(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(os.Getenv("IOS_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("IOS_MATRIX_RUN_ID is required")
		}
		body, err := core.Get("/validation/matrices/"+url.PathEscape(id)+"/wait", nil)
		if err != nil {
			return fmt.Errorf("wait for iOS matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Inspect iOS cell evidence and the fail-closed gate in the run review surface."}})
	}
}

func matrixCompare(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		current := strings.TrimSpace(os.Getenv("IOS_MATRIX_RUN_ID"))
		prior := strings.TrimSpace(os.Getenv("IOS_PRIOR_MATRIX_RUN_ID"))
		if current == "" || prior == "" {
			return fmt.Errorf("IOS_MATRIX_RUN_ID and IOS_PRIOR_MATRIX_RUN_ID are required")
		}
		body, err := core.Get("/validation/matrices/"+url.PathEscape(current)+"/compare/"+url.PathEscape(prior), nil)
		if err != nil {
			return fmt.Errorf("compare iOS matrices: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Inspect changed cells and evidence counts before promoting an iOS release gate."}})
	}
}

func matrixRerun(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(os.Getenv("IOS_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("IOS_MATRIX_RUN_ID is required")
		}
		body, err := core.Request("POST", "/validation/matrices/"+url.PathEscape(id)+"/rerun", nil, map[string]string{"kind": "failed"})
		if err != nil {
			return fmt.Errorf("rerun iOS matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Start the returned server-owned iOS run."}})
	}
}
