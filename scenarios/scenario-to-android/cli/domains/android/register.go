package android

import (
	"encoding/json"
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
		{Name: "generate", Description: "Render a generated Capacitor Android project without building artifacts", NeedsAPI: true, RunCtx: generate(core)},
		{Name: "build", Description: "Build a debug APK and AAB from a scenario web bundle", NeedsAPI: true, RunCtx: build(core)},
		{Name: "signing-provision", Description: "Generate and store the Android upload key through secrets-manager", NeedsAPI: true, RunCtx: signingProvision(core)},
		{Name: "run", Description: "Build, install, drive, and record one server-owned Android validation run", NeedsAPI: true, RunCtx: run(core)},
		{Name: "conformance-plan", Description: "Show the generated-app Android conformance chapters", NeedsAPI: true, RunCtx: get(core, "/android/conformance-plan")},
		{Name: "readiness", Description: "Show the six-rung Google release readiness ladder", NeedsAPI: true, RunCtx: get(core, "/android/readiness")},
		{Name: "distribution", Description: "Show independent Play, sideload, and ADB channels", NeedsAPI: true, RunCtx: get(core, "/android/distribution")},
		{Name: "matrix-catalog", Description: "Show the durable Android validation catalog", NeedsAPI: true, RunCtx: get(core, "/validation/catalog?scenario=hello-mobile")},
		{Name: "matrix-list", Description: "List durable Android validation runs", NeedsAPI: true, RunCtx: get(core, "/validation/matrices?scenario=hello-mobile")},
		{Name: "matrix-create", Description: "Create a normal-profile Android validation matrix", NeedsAPI: true, RunCtx: matrixCreate(core)},
		{Name: "matrix-start", Description: "Start a server-owned Android validation matrix", NeedsAPI: true, RunCtx: matrixAction(core, "start")},
		{Name: "matrix-wait", Description: "Wait for a server-owned Android validation matrix", NeedsAPI: true, RunCtx: matrixWait(core)},
		{Name: "compare", Description: "Compare two durable Android validation matrix runs", NeedsAPI: true, RunCtx: matrixCompare(core)},
		{Name: "matrix-rerun-failed", Description: "Rerun failed Android validation cells", NeedsAPI: true, RunCtx: matrixRerun(core)},
		{Name: "evidence-paths", Description: "Print the absolute physical and emulator evidence paths", NeedsAPI: false, RunCtx: evidencePaths},
	}}
}

func buildRequest() (map[string]string, error) {
	source := strings.TrimSpace(getenv("ANDROID_SOURCE_REF"))
	if source == "" {
		return nil, fmt.Errorf("ANDROID_SOURCE_REF is required")
	}
	return map[string]string{
		"source_ref":       source,
		"scenario_name":    strings.TrimSpace(getenv("ANDROID_ARTIFACT_SCENARIO")),
		"package_name":     strings.TrimSpace(getenv("ANDROID_ARTIFACT_PACKAGE")),
		"app_name":         strings.TrimSpace(getenv("ANDROID_ARTIFACT_APP")),
		"version_name":     strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION")),
		"version_code":     strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION_CODE")),
		"target_sdk":       strings.TrimSpace(getenv("ANDROID_TARGET_SDK")),
		"signing":          strings.TrimSpace(getenv("ANDROID_SIGNING")),
		"signing_identity": strings.TrimSpace(getenv("ANDROID_SIGNING_IDENTITY")),
	}, nil
}

func signingProvision(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		identity := strings.TrimSpace(getenv("ANDROID_SIGNING_IDENTITY"))
		body := map[string]string{}
		if identity != "" {
			body["identity"] = identity
		}
		response, err := core.Request("POST", "/android/signing/provision", nil, body)
		if err != nil {
			return fmt.Errorf("provision Android signing identity: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(response))}, NextSteps: []string{"Use ANDROID_SIGNING=required with android build after provisioning."}})
	}
}

func generate(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		body, err := buildRequest()
		if err != nil {
			return err
		}
		response, err := core.Request("POST", "/android/generate", nil, body)
		if err != nil {
			return fmt.Errorf("generate Android project: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(response))}, NextSteps: []string{"Use the reported project_path as the input to local inspection or android build."}})
	}
}

func build(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		body, err := buildRequest()
		if err != nil {
			return err
		}
		response, err := core.Request("POST", "/android/build", nil, body)
		if err != nil {
			return fmt.Errorf("build Android artifact: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(response))}, NextSteps: []string{"Use the returned APK checksum when creating an Android validation matrix."}})
	}
}

type builtArtifact struct {
	LocalPath string `json:"local_path"`
	Checksum  string `json:"checksum"`
}

type catalogTarget struct {
	Descriptor map[string]any `json:"descriptor"`
	Kind       string         `json:"kind"`
}

type androidCatalog struct {
	Targets []catalogTarget `json:"targets"`
}

// selectPhysicalTarget keeps the end-to-end operator command focused on the
// physical target it is intended to prove. The general matrix-create command
// remains the way to run the full catalog, including an emulator when one is
// available. If a target id is supplied, an unavailable target is preserved so
// the server-owned matrix can record a typed fail-closed result rather than
// silently substituting another device.
func selectPhysicalTarget(body []byte, requestedID string) (map[string]any, error) {
	var catalog androidCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("decode Android target catalog: %w", err)
	}
	requestedID = strings.TrimSpace(requestedID)
	if requestedID != "" {
		if strings.HasPrefix(requestedID, "android:emulator:") {
			return nil, fmt.Errorf("android run requires a physical target; use matrix-create for emulator validation")
		}
		for _, target := range catalog.Targets {
			if target.Descriptor["target_id"] == requestedID {
				return map[string]any{"descriptor": target.Descriptor, "kind": target.Kind}, nil
			}
		}
		return nil, fmt.Errorf("Android target %q was not found in the live catalog", requestedID)
	}

	var candidates []catalogTarget
	for _, target := range catalog.Targets {
		targetID, _ := target.Descriptor["target_id"].(string)
		available, _ := target.Descriptor["available"].(bool)
		if available && !strings.HasPrefix(targetID, "android:emulator:") {
			candidates = append(candidates, target)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available physical Android target is present in the live catalog; set ANDROID_TARGET_ID to request a specific target")
	}
	if len(candidates) > 1 {
		ids := make([]string, 0, len(candidates))
		for _, target := range candidates {
			if targetID, ok := target.Descriptor["target_id"].(string); ok {
				ids = append(ids, targetID)
			}
		}
		return nil, fmt.Errorf("multiple physical Android targets are available (%s); set ANDROID_TARGET_ID", strings.Join(ids, ", "))
	}
	return map[string]any{"descriptor": candidates[0].Descriptor, "kind": candidates[0].Kind}, nil
}

func run(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		body, err := buildRequest()
		if err != nil {
			return err
		}
		if strings.TrimSpace(body["scenario_name"]) == "" {
			body["scenario_name"] = "hello-mobile"
		}

		artifactBody, err := core.Request("POST", "/android/build", nil, body)
		if err != nil {
			return fmt.Errorf("build Android artifact for validation run: %w", err)
		}
		var artifact builtArtifact
		if err := json.Unmarshal(artifactBody, &artifact); err != nil {
			return fmt.Errorf("decode Android build artifact: %w", err)
		}
		if strings.TrimSpace(artifact.LocalPath) == "" || strings.TrimSpace(artifact.Checksum) == "" {
			return fmt.Errorf("Android build returned no usable artifact path and checksum")
		}
		catalogBody, err := core.Get("/validation/catalog?scenario="+url.QueryEscape(body["scenario_name"]), nil)
		if err != nil {
			return fmt.Errorf("resolve physical Android target catalog: %w", err)
		}
		target, err := selectPhysicalTarget(catalogBody, getenv("ANDROID_TARGET_ID"))
		if err != nil {
			return err
		}

		selection := map[string]any{
			"scenario_name":   body["scenario_name"],
			"artifact_digest": artifact.Checksum,
			"artifact_path":   artifact.LocalPath,
			"metadata": map[string]string{
				"package_name":    body["package_name"],
				"scenario_name":   body["scenario_name"],
				"auth_profile_id": strings.TrimSpace(getenv("ANDROID_AUTH_PROFILE_ID")),
			},
			"targets":              []map[string]any{target},
			"environment_profiles": []int{1},
			"max_concurrency":      1,
		}
		matrixBody, err := core.Request("POST", "/validation/matrices", nil, selection)
		if err != nil {
			return fmt.Errorf("create Android validation matrix: %w", err)
		}
		var matrix struct {
			RunID string `json:"run_id"`
		}
		if err := json.Unmarshal(matrixBody, &matrix); err != nil || strings.TrimSpace(matrix.RunID) == "" {
			if err == nil {
				err = fmt.Errorf("response omitted run_id")
			}
			return fmt.Errorf("decode Android validation matrix: %w", err)
		}

		path := "/validation/matrices/" + url.PathEscape(matrix.RunID)
		if _, err := core.Request("POST", path+"/start", nil, nil); err != nil {
			return fmt.Errorf("start Android validation matrix %s: %w", matrix.RunID, err)
		}
		result, err := core.Get(path+"/wait", nil)
		if err != nil {
			return fmt.Errorf("wait for Android validation matrix %s: %w", matrix.RunID, err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{
			Status: []string{strings.TrimSpace(string(result))},
			NextSteps: []string{
				"Review the returned physical cell, correlated timeline, and review_recording_path before promoting the verdict.",
			},
		})
	}
}

func matrixCompare(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		current := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		prior := strings.TrimSpace(getenv("ANDROID_PRIOR_MATRIX_RUN_ID"))
		if current == "" || prior == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID and ANDROID_PRIOR_MATRIX_RUN_ID are required")
		}
		path := "/validation/matrices/" + url.PathEscape(current) + "/compare/" + url.PathEscape(prior)
		body, err := core.Get(path, nil)
		if err != nil {
			return fmt.Errorf("compare Android matrices: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body))}, NextSteps: []string{"Inspect changed cells and evidence counts before promoting a release gate."}})
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
		body, err := core.Request("POST", "/validation/matrices", nil, map[string]any{"scenario_name": "hello-mobile", "artifact_digest": digest, "artifact_path": strings.TrimSpace(getenv("ANDROID_ARTIFACT_PATH")), "environment_profiles": []int{1}, "max_concurrency": 1})
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
