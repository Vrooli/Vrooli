package android

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "android", Description: "Operate the Android delivery ramp", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "targets", Description: "Probe local, emulator, physical, and bridge Android targets", NeedsAPI: true, RunCtx: get(core, "/android/targets")},
		{Name: "build", Description: "Build a debug APK and AAB from a scenario web bundle", NeedsAPI: true, RunCtx: build(core)},
		{Name: "conformance-plan", Description: "Show the generated-app Android conformance chapters", NeedsAPI: true, RunCtx: get(core, "/android/conformance-plan")},
		{Name: "readiness", Description: "Show the six-rung Google release readiness ladder", NeedsAPI: true, RunCtx: get(core, "/android/readiness")},
		{Name: "distribution", Description: "Show independent Play, sideload, and ADB channels", NeedsAPI: true, RunCtx: get(core, "/android/distribution")},
		{Name: "matrix-catalog", Description: "Show the durable Android validation catalog", NeedsAPI: true, RunCtx: get(core, "/validation/catalog?scenario=hello-mobile")},
		{Name: "matrix-list", Description: "List durable Android validation runs", NeedsAPI: true, RunCtx: get(core, "/validation/matrices?scenario=hello-mobile")},
		{Name: "matrix-create", Description: "Create a normal-profile Android validation matrix", NeedsAPI: true, RunCtx: matrixCreate(core)},
		{Name: "matrix-start", Description: "Start a server-owned Android validation matrix", NeedsAPI: true, RunCtx: matrixAction(core, "start")},
		{Name: "matrix-wait", Description: "Wait for a server-owned Android validation matrix", NeedsAPI: true, RunCtx: matrixWait(core)},
		{Name: "matrix-rerun-failed", Description: "Rerun failed Android validation cells", NeedsAPI: true, RunCtx: matrixRerun(core)},
		{Name: "evidence-paths", Description: "Print the absolute physical and emulator evidence paths", NeedsAPI: false, RunCtx: evidencePaths},
	}}
}

func build(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		source := strings.TrimSpace(getenv("ANDROID_SOURCE_REF"))
		if source == "" {
			return fmt.Errorf("ANDROID_SOURCE_REF is required")
		}
		body, err := core.Request("POST", "/android/build", nil, map[string]string{
			"source_ref":   source,
			"package_name": strings.TrimSpace(getenv("ANDROID_ARTIFACT_PACKAGE")),
			"app_name":     strings.TrimSpace(getenv("ANDROID_ARTIFACT_APP")),
			"version_name": strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION")),
			"version_code": strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION_CODE")),
			"target_sdk":   strings.TrimSpace(getenv("ANDROID_TARGET_SDK")),
		})
		if err != nil {
			return fmt.Errorf("build Android artifact: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Use the returned APK checksum when creating an Android validation matrix."}})
	}
}

func get(core *cliapp.ScenarioApp, path string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		body, err := core.Get(path, url.Values{})
		if err != nil {
			return fmt.Errorf("get %s: %w", path, err)
		}
		text := strings.TrimSpace(string(body))
		if text == "" {
			text = "{}"
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{text}, NextSteps: []string{"Use the next action reported by the Android ramp before attempting a release."}})
	}
}

func matrixCreate(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		digest := strings.TrimSpace(getenv("ANDROID_ARTIFACT_DIGEST"))
		if digest == "" {
			return fmt.Errorf("ANDROID_ARTIFACT_DIGEST is required")
		}
		body, err := core.Request("POST", "/validation/matrices", nil, map[string]any{"scenario_name": "hello-mobile", "artifact_digest": digest, "environment_profiles": []int{1}, "max_concurrency": 1})
		if err != nil {
			return fmt.Errorf("create Android matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Run vrooli scenario-to-android android matrix-start with ANDROID_MATRIX_RUN_ID from this response."}})
	}
}

func matrixAction(core *cliapp.ScenarioApp, action string) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID is required")
		}
		body, err := core.Request("POST", "/validation/matrices/"+url.PathEscape(id)+"/"+action, nil, nil)
		if err != nil {
			return fmt.Errorf("matrix %s: %w", action, err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Use matrix-wait after the server-owned run completes."}})
	}
}

func matrixWait(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID is required")
		}
		body, err := core.Get("/validation/matrices/"+url.PathEscape(id)+"/wait", nil)
		if err != nil {
			return fmt.Errorf("wait for Android matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Inspect cell evidence and the fail-closed gate in the run review surface."}})
	}
}

func matrixRerun(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID is required")
		}
		body, err := core.Request("POST", "/validation/matrices/"+url.PathEscape(id)+"/rerun", nil, map[string]string{"kind": "failed"})
		if err != nil {
			return fmt.Errorf("rerun Android matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Start the returned server-owned run."}})
	}
}

func evidencePaths(ctx cliapp.RunContext) error {
	physical := strings.TrimSpace(getenv("ANDROID_PHYSICAL_EVIDENCE_PATH"))
	emulator := strings.TrimSpace(getenv("ANDROID_EMULATOR_EVIDENCE_PATH"))
	if physical == "" || emulator == "" || !filepath.IsAbs(physical) || !filepath.IsAbs(emulator) {
		return fmt.Errorf("ANDROID_PHYSICAL_EVIDENCE_PATH and ANDROID_EMULATOR_EVIDENCE_PATH must be absolute paths")
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{"physical=" + physical, "emulator=" + emulator}, NextSteps: []string{"Run the five evidence-bar checks against both files."}})
}

var getenv = os.Getenv
