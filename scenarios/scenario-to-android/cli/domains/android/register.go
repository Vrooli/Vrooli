package android

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "android", Description: "Operate the Android delivery ramp", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "targets", Description: "Probe local, emulator, physical, and bridge Android targets", NeedsAPI: true, RunCtx: get(core, "/android/targets")},
		{Name: "generate", Description: "Prepare a native project or generated Capacitor project without building artifacts", NeedsAPI: true, RunCtx: generate(core)},
		{Name: "build", Description: "Build an APK and AAB from a native project or scenario web bundle", NeedsAPI: true, RunCtx: build(core)},
		{Name: "signing-provision", Description: "Generate and store the Android upload key through secrets-manager", NeedsAPI: true, RunCtx: signingProvision(core)},
		{Name: "run", Description: "Build, install, drive, and record one server-owned Android validation run", NeedsAPI: true, RunCtx: run(core)},
		{Name: "app-test", Description: "Build a scenario APK, test cold and warm BAS behavior, and safely remove it", NeedsAPI: true, RunCtx: appTest(core)},
		{Name: "app-install", Description: "Build a scenario APK, test it, and leave a fresh install on the exact device", NeedsAPI: true, RunCtx: appInstall(core)},
		{Name: "app-update", Description: "Build a higher-version scenario APK, preserve app data, and verify cold and warm BAS behavior", NeedsAPI: true, RunCtx: appUpdate(core)},
		{Name: "app-verify", Description: "Verify an installed scenario app with cold and warm BAS without reinstalling it", NeedsAPI: true, RunCtx: appVerify(core)},
		{Name: "conformance-plan", Description: "Show the generated-app Android conformance chapters", NeedsAPI: true, RunCtx: get(core, "/android/conformance-plan")},
		{Name: "readiness", Description: "Show the six-rung Google release readiness ladder", NeedsAPI: true, RunCtx: get(core, "/android/readiness")},
		{Name: "distribution", Description: "Show independent Play, sideload, and ADB channels", NeedsAPI: true, RunCtx: get(core, "/android/distribution")},
		{Name: "matrix-catalog", Description: "Show the durable Android validation catalog", NeedsAPI: true, RunCtx: get(core, "/validation/catalog?scenario=hello-mobile")},
		{Name: "matrix-list", Description: "List durable Android validation runs", NeedsAPI: true, RunCtx: get(core, "/validation/matrices?scenario=hello-mobile")},
		{Name: "matrix-create", Description: "Create a normal-profile Android validation matrix", NeedsAPI: true, RunCtx: matrixCreate(core)},
		{Name: "matrix-start", Description: "Start a server-owned Android validation matrix", NeedsAPI: true, RunCtx: matrixAction(core, "start")},
		{Name: "matrix-abort", Description: "Cancel a matrix and wait for its owner to persist a terminal state", NeedsAPI: true, RunCtx: matrixAbort(core)},
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
		"source_ref":             source,
		"project_kind":           strings.TrimSpace(getenv("ANDROID_PROJECT_KIND")),
		"native_profile":         strings.TrimSpace(getenv("ANDROID_NATIVE_PROFILE")),
		"preview_cleartext_host": strings.TrimSpace(getenv("ANDROID_PREVIEW_CLEARTEXT_HOST")),
		"scenario_name":          strings.TrimSpace(getenv("ANDROID_ARTIFACT_SCENARIO")),
		"package_name":           strings.TrimSpace(getenv("ANDROID_ARTIFACT_PACKAGE")),
		"app_name":               strings.TrimSpace(getenv("ANDROID_ARTIFACT_APP")),
		"version_name":           strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION")),
		"version_code":           strings.TrimSpace(getenv("ANDROID_ARTIFACT_VERSION_CODE")),
		"target_sdk":             strings.TrimSpace(getenv("ANDROID_TARGET_SDK")),
		"signing":                strings.TrimSpace(getenv("ANDROID_SIGNING")),
		"signing_identity":       strings.TrimSpace(getenv("ANDROID_SIGNING_IDENTITY")),
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
	ImmutableRef string            `json:"immutable_ref"`
	LocalPath    string            `json:"local_path"`
	Kind         string            `json:"kind"`
	Checksum     string            `json:"checksum"`
	Metadata     map[string]string `json:"metadata"`
}

type catalogTarget struct {
	Descriptor map[string]any `json:"descriptor"`
	Kind       string         `json:"kind"`
}

type androidCatalog struct {
	Targets []catalogTarget `json:"targets"`
}

type artifactDescriptor struct {
	ImmutableRef    string `json:"immutable_ref"`
	Checksum        string `json:"checksum"`
	PackageName     string `json:"package_name"`
	VersionName     string `json:"version_name"`
	VersionCode     int64  `json:"version_code"`
	Signing         string `json:"signing"`
	SigningIdentity string `json:"signing_identity"`
}

const (
	defaultAndroidPackageName  = "com.vrooli.generated.app"
	defaultAndroidScenarioName = "hello-mobile"
)

func matrixSelectionRequest(digest, artifactPath string, metadata map[string]string, target map[string]any) map[string]any {
	request := map[string]any{
		"scenario_name":        metadata["scenario_name"],
		"artifact_digest":      digest,
		"artifact_path":        artifactPath,
		"environment_profiles": []int{1},
		"max_concurrency":      1,
		"metadata":             metadata,
	}
	if target != nil {
		request["targets"] = []map[string]any{target}
	}
	return request
}

func selectedJourneyIDs(raw string) ([]map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	seen := make(map[string]struct{})
	journeys := make([]map[string]string, 0)
	for _, value := range strings.Split(raw, ",") {
		id := strings.TrimSpace(value)
		if id == "" {
			return nil, fmt.Errorf("ANDROID_JOURNEY_IDS must contain comma-separated journey IDs without empty entries")
		}
		for _, r := range id {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return nil, fmt.Errorf("invalid journey ID %q in ANDROID_JOURNEY_IDS", id)
			}
		}
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("duplicate journey ID %q in ANDROID_JOURNEY_IDS", id)
		}
		seen[id] = struct{}{}
		journeys = append(journeys, map[string]string{"journey_id": id})
	}
	return journeys, nil
}

func withAuthProfile(metadata map[string]string, profileID string) map[string]string {
	if metadata == nil {
		metadata = make(map[string]string)
	}
	if profileID = strings.TrimSpace(profileID); profileID != "" {
		metadata["auth_profile_id"] = profileID
	}
	return metadata
}

func matrixArtifactMetadata(digest, immutableRef, packageOverride, scenarioName string, artifact artifactDescriptor) (map[string]string, error) {
	digest = strings.TrimSpace(digest)
	checksum := "sha256:" + strings.TrimPrefix(digest, "sha256:")
	immutableRef = strings.TrimSpace(immutableRef)
	if !validImmutableRefForDigest(immutableRef, checksum) || artifact.ImmutableRef != immutableRef || artifact.Checksum != checksum {
		return nil, fmt.Errorf("Android artifact metadata does not match the requested immutable ref and digest")
	}
	packageName := strings.TrimSpace(artifact.PackageName)
	if override := strings.TrimSpace(packageOverride); override != "" && override != packageName {
		return nil, fmt.Errorf("ANDROID_ARTIFACT_PACKAGE %q does not match owner-produced package %q", override, packageName)
	}
	signing := strings.TrimSpace(artifact.Signing)
	if signing == "" && strings.HasPrefix(immutableRef, "android-debug:") {
		// The immutable ref identifies the builder-owned debug keystore path;
		// the build owner contract assigns this identity the fixed debug-only label.
		signing = "debug-only"
	}
	if packageName == "" || strings.TrimSpace(artifact.VersionName) == "" || artifact.VersionCode <= 0 || signing == "" {
		return nil, fmt.Errorf("Android artifact owner metadata is incomplete")
	}
	if strings.HasPrefix(immutableRef, "android-release:") && strings.TrimSpace(artifact.SigningIdentity) == "" {
		return nil, fmt.Errorf("Android release artifact metadata is missing its signing identity")
	}
	scenarioName = strings.TrimSpace(scenarioName)
	if scenarioName == "" {
		scenarioName = defaultAndroidScenarioName
	}
	return map[string]string{
		"immutable_ref":    immutableRef,
		"package_name":     packageName,
		"scenario_name":    scenarioName,
		"version_name":     artifact.VersionName,
		"version_code":     strconv.FormatInt(artifact.VersionCode, 10),
		"signing":          signing,
		"signing_identity": artifact.SigningIdentity,
	}, nil
}

func resolveMatrixArtifactMetadata(core *cliapp.ScenarioApp, digest, immutableRef, packageOverride, scenarioName string) (map[string]string, error) {
	immutableRef = strings.TrimSpace(immutableRef)
	if immutableRef == "" {
		return nil, fmt.Errorf("ANDROID_ARTIFACT_REF is required; use the immutable_ref returned by android build")
	}
	query := url.Values{"immutable_ref": []string{immutableRef}}
	body, err := core.Get("/android/artifacts/metadata", query)
	if err != nil {
		return nil, fmt.Errorf("resolve Android artifact metadata: %w", err)
	}
	var artifact artifactDescriptor
	if err := json.Unmarshal(body, &artifact); err != nil {
		return nil, fmt.Errorf("decode owner-produced Android artifact metadata: %w", err)
	}
	return matrixArtifactMetadata(digest, immutableRef, packageOverride, scenarioName, artifact)
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

// selectCatalogTarget narrows a full-catalog matrix to one exact target when
// requested. Unavailable rows are preserved so the server records the typed
// capability failure instead of silently substituting another device.
func selectCatalogTarget(body []byte, requestedID string) (map[string]any, error) {
	requestedID = strings.TrimSpace(requestedID)
	if requestedID == "" {
		return nil, nil
	}
	var catalog androidCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("decode Android target catalog: %w", err)
	}
	for _, target := range catalog.Targets {
		if target.Descriptor["target_id"] == requestedID {
			return map[string]any{"descriptor": target.Descriptor, "kind": target.Kind}, nil
		}
	}
	return nil, fmt.Errorf("Android target %q was not found in the live catalog", requestedID)
}

func run(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return runJourney(core, "conformance")
}

func appTest(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return runJourney(core, "temporary")
}

func appInstall(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return runJourney(core, "durable")
}

func appUpdate(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return runJourney(core, "update")
}

func appVerify(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return runJourney(core, "verify")
}

func runJourney(core *cliapp.ScenarioApp, mode string) func(cliapp.RunContext) error {
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
			return fmt.Errorf("Android build returned no usable artifact path and digest")
		}
		if !immutableArtifactMetadataComplete(artifact) {
			return fmt.Errorf("Android build omitted immutable ref, package, version, or signing metadata required for safe installation")
		}
		catalogBody, err := core.Get("/validation/catalog?scenario="+url.QueryEscape(body["scenario_name"]), nil)
		if err != nil {
			return fmt.Errorf("resolve physical Android target catalog: %w", err)
		}
		target, err := selectPhysicalTarget(catalogBody, getenv("ANDROID_TARGET_ID"))
		if err != nil {
			return err
		}

		packageName := strings.TrimSpace(artifact.Metadata["package_name"])
		if packageName == "" {
			packageName = strings.TrimSpace(body["package_name"])
		}
		if packageName == "" {
			packageName = defaultAndroidPackageName
		}
		metadata := journeyArtifactMetadata(artifact, body["scenario_name"], packageName, getenv("ANDROID_AUTH_PROFILE_ID"), mode)
		if mode != "conformance" {
			metadata["install_mode"] = mode
		}
		journeyID := "android-generated-app-conformance-v1"
		switch mode {
		case "temporary":
			journeyID = "android-scenario-app-temporary-test-v1"
		case "durable":
			journeyID = "android-scenario-app-durable-v1"
		case "update":
			journeyID = "android-scenario-app-update-v1"
		case "verify":
			journeyID = "android-scenario-app-verify-v1"
		}
		selection := map[string]any{
			"scenario_name":        body["scenario_name"],
			"artifact_digest":      artifact.Checksum,
			"artifact_path":        artifact.LocalPath,
			"metadata":             metadata,
			"journeys":             []map[string]string{{"journey_id": journeyID}},
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
		result, err := core.GetWithoutTimeout(path+"/wait", nil)
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

func journeyArtifactMetadata(artifact builtArtifact, scenarioName, packageName, authProfileID, mode string) map[string]string {
	metadata := map[string]string{
		"package_name":    strings.TrimSpace(packageName),
		"scenario_name":   strings.TrimSpace(scenarioName),
		"auth_profile_id": strings.TrimSpace(authProfileID),
		"immutable_ref":   strings.TrimSpace(artifact.ImmutableRef),
	}
	for _, key := range []string{"version_name", "version_code", "signing_identity", "signing"} {
		if value := strings.TrimSpace(artifact.Metadata[key]); value != "" {
			metadata[key] = value
		}
	}
	if strings.TrimSpace(mode) != "conformance" {
		metadata["install_mode"] = strings.TrimSpace(mode)
	}
	return metadata
}

func immutableArtifactMetadataComplete(artifact builtArtifact) bool {
	if !strings.HasPrefix(artifact.ImmutableRef, "android-debug:") && !strings.HasPrefix(artifact.ImmutableRef, "android-release:") {
		return false
	}
	if !validImmutableRefForDigest(artifact.ImmutableRef, artifact.Checksum) {
		return false
	}
	metadata := artifact.Metadata
	if strings.TrimSpace(metadata["package_name"]) == "" || strings.TrimSpace(metadata["version_name"]) == "" || strings.TrimSpace(metadata["version_code"]) == "" || strings.TrimSpace(metadata["signing"]) == "" {
		return false
	}
	return !strings.HasPrefix(artifact.ImmutableRef, "android-release:") || strings.TrimSpace(metadata["signing_identity"]) != ""
}

func validImmutableRefForDigest(ref, checksum string) bool {
	parts := strings.Split(strings.TrimSpace(ref), ":")
	if len(parts) != 2 || (parts[0] != "android-debug" && parts[0] != "android-release") || len(parts[1]) != 64 {
		return false
	}
	for _, char := range parts[1] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return strings.TrimPrefix(strings.TrimSpace(checksum), "sha256:") == parts[1]
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
		artifactPath := strings.TrimSpace(getenv("ANDROID_ARTIFACT_PATH"))
		if artifactPath == "" {
			return fmt.Errorf("ANDROID_ARTIFACT_PATH is required for an Android matrix run")
		}
		metadata, err := resolveMatrixArtifactMetadata(core, digest, getenv("ANDROID_ARTIFACT_REF"), getenv("ANDROID_ARTIFACT_PACKAGE"), getenv("ANDROID_ARTIFACT_SCENARIO"))
		if err != nil {
			return err
		}
		metadata = withAuthProfile(metadata, getenv("ANDROID_AUTH_PROFILE_ID"))
		journeys, err := selectedJourneyIDs(getenv("ANDROID_JOURNEY_IDS"))
		if err != nil {
			return err
		}
		var target map[string]any
		if targetID := strings.TrimSpace(getenv("ANDROID_TARGET_ID")); targetID != "" {
			catalogBody, err := core.Get("/validation/catalog?scenario="+url.QueryEscape(metadata["scenario_name"]), nil)
			if err != nil {
				return fmt.Errorf("resolve exact Android target %q: %w", targetID, err)
			}
			selected, err := selectCatalogTarget(catalogBody, targetID)
			if err != nil {
				return err
			}
			target = selected
		}
		selection := matrixSelectionRequest(digest, artifactPath, metadata, target)
		if len(journeys) > 0 {
			selection["journeys"] = journeys
		}
		body, err := core.Request("POST", "/validation/matrices", nil, selection)
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

func matrixAbort(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID is required")
		}
		path := "/validation/matrices/" + url.PathEscape(id)
		if _, err := core.Request("POST", path+"/abort", nil, nil); err != nil {
			return fmt.Errorf("abort Android matrix: %w", err)
		}
		body, err := core.GetWithoutTimeout(path+"/wait", nil)
		if err != nil {
			return fmt.Errorf("wait for aborted Android matrix: %w", err)
		}
		return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{strings.TrimSpace(string(body)), "Owner persisted the terminal matrix state."}, NextSteps: []string{"Inspect cell evidence and the fail-closed gate in the run review surface."}})
	}
}

func matrixWait(core *cliapp.ScenarioApp) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		id := strings.TrimSpace(getenv("ANDROID_MATRIX_RUN_ID"))
		if id == "" {
			return fmt.Errorf("ANDROID_MATRIX_RUN_ID is required")
		}
		body, err := core.GetWithoutTimeout("/validation/matrices/"+url.PathEscape(id)+"/wait", nil)
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
