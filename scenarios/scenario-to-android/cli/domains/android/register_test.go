package android

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

func TestMatrixAbortWaitsForOwnerTerminalState(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/abort"):
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"state":"running"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/wait"):
			_, _ = w.Write([]byte(`{"state":"cancelled","completed_at":"2026-11-03T12:00:00Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "scenario-to-android", APIPrefix: "/api/v1", DefaultAPIBase: server.URL, AllowAnonymous: true})
	require.NoError(t, err)
	core.APIOverride = server.URL
	t.Setenv("ANDROID_MATRIX_RUN_ID", "run-123")
	var stdout bytes.Buffer
	runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &stdout})

	err = matrixAbort(core)(runCtx)
	require.NoError(t, err)
	require.Equal(t, []string{
		"POST /api/v1/validation/matrices/run-123/abort",
		"GET /api/v1/validation/matrices/run-123/wait",
	}, paths)
	require.Contains(t, stdout.String(), "cancelled")
	require.Contains(t, stdout.String(), "Owner persisted the terminal matrix state.")
}

func TestRegisterExposesMatrixAbortCommand(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	for _, command := range group.Subcommands {
		if command.Name == "matrix-abort" {
			require.Contains(t, command.Description, "terminal state")
			require.NotNil(t, command.RunCtx)
			return
		}
	}
	t.Fatal("matrix-abort is not discoverable in the Android CLI")
}

func TestRegisterExposesAppUpdateCommand(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	for _, command := range group.Subcommands {
		if command.Name == "app-update" {
			require.Contains(t, command.Description, "higher-version")
			require.NotNil(t, command.RunCtx)
			return
		}
	}
	t.Fatal("app-update is not discoverable in the Android CLI")
}

func TestRegisterExposesInstalledAppVerificationCommand(t *testing.T) {
	group := Register(&cliapp.ScenarioApp{})
	for _, command := range group.Subcommands {
		if command.Name == "app-verify" {
			require.Contains(t, command.Description, "without reinstalling")
			require.NotNil(t, command.RunCtx)
			return
		}
	}
	t.Fatal("app-verify is not discoverable in the Android CLI")
}

func TestAppVerifyBuildsAndSelectsTheInstalledAppVerificationJourney(t *testing.T) {
	digest := strings.Repeat("a", 64)
	var matrixSelection map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/android/build":
			_, _ = w.Write([]byte(`{"immutable_ref":"android-debug:` + digest + `","local_path":"/artifacts/app.apk","checksum":"sha256:` + digest + `","metadata":{"package_name":"com.example.generated","version_name":"1.2.3","version_code":"12","signing":"debug-only"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/catalog":
			_, _ = w.Write([]byte(`{"targets":[{"descriptor":{"target_id":"android-physical-1","available":true},"kind":"local"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices":
			if err := json.NewDecoder(r.Body).Decode(&matrixSelection); err != nil {
				t.Fatalf("decode matrix selection: %v", err)
			}
			_, _ = w.Write([]byte(`{"run_id":"run-verify"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices/run-verify/start":
			_, _ = w.Write([]byte(`{"state":"running"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/matrices/run-verify/wait":
			_, _ = w.Write([]byte(`{"state":"passed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "scenario-to-android", APIPrefix: "/api/v1", DefaultAPIBase: server.URL, AllowAnonymous: true})
	require.NoError(t, err)
	core.APIOverride = server.URL
	t.Setenv("ANDROID_SOURCE_REF", "/scenario/dist")
	t.Setenv("ANDROID_ARTIFACT_VERSION_CODE", "12")
	t.Setenv("ANDROID_ARTIFACT_VERSION", "1.2.3")
	t.Setenv("ANDROID_ARTIFACT_SCENARIO", "hello-mobile")
	t.Setenv("ANDROID_TARGET_ID", "android-physical-1")
	var stdout bytes.Buffer
	runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &stdout})
	require.NoError(t, appVerify(core)(runCtx))
	journeys, ok := matrixSelection["journeys"].([]any)
	require.True(t, ok)
	require.Len(t, journeys, 1)
	require.Equal(t, "android-scenario-app-verify-v1", journeys[0].(map[string]any)["journey_id"])
	metadata := matrixSelection["metadata"].(map[string]any)
	require.Equal(t, "verify", metadata["install_mode"])
	require.Equal(t, "com.example.generated", metadata["package_name"])
	require.Contains(t, stdout.String(), "passed")
}

func TestAppUpdateBuildsAndSelectsTheHigherVersionUpdateJourney(t *testing.T) {
	digest := strings.Repeat("a", 64)
	var matrixSelection map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/android/build":
			_, _ = w.Write([]byte(`{"immutable_ref":"android-debug:` + digest + `","local_path":"/artifacts/app.apk","checksum":"sha256:` + digest + `","metadata":{"package_name":"com.example.generated","version_name":"1.2.3","version_code":"12","signing":"debug-only"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/catalog":
			_, _ = w.Write([]byte(`{"targets":[{"descriptor":{"target_id":"android-physical-1","available":true},"kind":"local"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&matrixSelection))
			_, _ = w.Write([]byte(`{"run_id":"run-update-1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices/run-update-1/start":
			_, _ = w.Write([]byte(`{"state":"running"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/matrices/run-update-1/wait":
			_, _ = w.Write([]byte(`{"state":"passed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "scenario-to-android", APIPrefix: "/api/v1", DefaultAPIBase: server.URL, AllowAnonymous: true})
	require.NoError(t, err)
	core.APIOverride = server.URL
	t.Setenv("ANDROID_SOURCE_REF", "/scenario/dist")
	t.Setenv("ANDROID_ARTIFACT_VERSION_CODE", "12")
	t.Setenv("ANDROID_ARTIFACT_VERSION", "1.2.3")
	t.Setenv("ANDROID_ARTIFACT_SCENARIO", "hello-mobile")
	t.Setenv("ANDROID_TARGET_ID", "android-physical-1")
	var stdout bytes.Buffer
	runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &stdout})

	require.NoError(t, appUpdate(core)(runCtx))
	journeys, ok := matrixSelection["journeys"].([]any)
	require.True(t, ok)
	require.Len(t, journeys, 1)
	require.Equal(t, "android-scenario-app-update-v1", journeys[0].(map[string]any)["journey_id"])
	metadata := matrixSelection["metadata"].(map[string]any)
	require.Equal(t, "update", metadata["install_mode"])
	require.Equal(t, "12", metadata["version_code"])
	require.Equal(t, "android-debug:"+digest, metadata["immutable_ref"])
	require.Contains(t, stdout.String(), "passed")
}

func TestConformanceRunCarriesImmutableInstallIdentity(t *testing.T) {
	digest := strings.Repeat("c", 64)
	var matrixSelection map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/android/build":
			_, _ = w.Write([]byte(`{"immutable_ref":"android-debug:` + digest + `","local_path":"/artifacts/app.apk","checksum":"sha256:` + digest + `","metadata":{"package_name":"com.example.conformance","version_name":"1.0.0","version_code":"1","signing":"debug-only"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/catalog":
			_, _ = w.Write([]byte(`{"targets":[{"descriptor":{"target_id":"android-physical-1","available":true},"kind":"local"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&matrixSelection))
			_, _ = w.Write([]byte(`{"run_id":"run-conformance-1"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/validation/matrices/run-conformance-1/start":
			_, _ = w.Write([]byte(`{"state":"running"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/validation/matrices/run-conformance-1/wait":
			_, _ = w.Write([]byte(`{"state":"passed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "scenario-to-android", APIPrefix: "/api/v1", DefaultAPIBase: server.URL, AllowAnonymous: true})
	require.NoError(t, err)
	core.APIOverride = server.URL
	t.Setenv("ANDROID_SOURCE_REF", "/scenario/dist")
	t.Setenv("ANDROID_ARTIFACT_SCENARIO", "hello-mobile")
	t.Setenv("ANDROID_ARTIFACT_PACKAGE", "com.example.conformance")
	t.Setenv("ANDROID_TARGET_ID", "android-physical-1")
	var stdout bytes.Buffer
	runCtx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, JSON: true, Stdout: &stdout})

	require.NoError(t, run(core)(runCtx))
	journeys := matrixSelection["journeys"].([]any)
	require.Equal(t, "android-generated-app-conformance-v1", journeys[0].(map[string]any)["journey_id"])
	metadata := matrixSelection["metadata"].(map[string]any)
	require.Equal(t, "android-debug:"+digest, metadata["immutable_ref"])
	require.Equal(t, "1.0.0", metadata["version_name"])
	require.Equal(t, "1", metadata["version_code"])
	require.Equal(t, "debug-only", metadata["signing"])
	require.NotContains(t, metadata, "install_mode")
	require.Contains(t, stdout.String(), "passed")
}

func TestBuildRequestCarriesNativeProfileAndSigningIdentity(t *testing.T) {
	t.Setenv("ANDROID_SOURCE_REF", "/native/source")
	t.Setenv("ANDROID_PROJECT_KIND", "native")
	t.Setenv("ANDROID_NATIVE_PROFILE", "daily")
	t.Setenv("ANDROID_SIGNING_IDENTITY", "vrooli/portal/tv-home-signing-key")
	request, err := buildRequest()
	require.NoError(t, err)
	require.Equal(t, "daily", request["native_profile"])
	require.Equal(t, "native", request["project_kind"])
	require.Equal(t, "vrooli/portal/tv-home-signing-key", request["signing_identity"])
}

func TestSelectPhysicalTargetPrefersAvailablePhysicalTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": true}, Kind: "local"},
		{Descriptor: map[string]any{"target_id": "android-phone", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	target, err := selectPhysicalTarget(body, "")
	require.NoError(t, err)
	require.Equal(t, "android-phone", target["descriptor"].(map[string]any)["target_id"])
}

func TestSelectPhysicalTargetPreservesExplicitUnavailableTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android-phone", "available": false, "reason": "locked"}, Kind: "local"},
	}})
	require.NoError(t, err)

	target, err := selectPhysicalTarget(body, "android-phone")
	require.NoError(t, err)
	require.Equal(t, false, target["descriptor"].(map[string]any)["available"])
}

func TestSelectPhysicalTargetRejectsExplicitEmulator(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "android:emulator:local")
	require.ErrorContains(t, err, "requires a physical target")
}

func TestSelectPhysicalTargetFailsWithoutPhysicalTarget(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android:emulator:local", "available": false}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "")
	require.ErrorContains(t, err, "no available physical Android target")
}

func TestSelectPhysicalTargetRequiresExplicitSelectionWhenAmbiguous(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android-phone-1", "available": true}, Kind: "local"},
		{Descriptor: map[string]any{"target_id": "android-phone-2", "available": true}, Kind: "local"},
	}})
	require.NoError(t, err)

	_, err = selectPhysicalTarget(body, "")
	require.ErrorContains(t, err, "multiple physical Android targets")
}

func TestMatrixSelectionCarriesOwnerProducedArtifactMetadata(t *testing.T) {
	metadata := map[string]string{
		"immutable_ref": "android-debug:" + strings.Repeat("a", 64),
		"package_name":  "com.example.custom", "scenario_name": "hello-mobile",
		"version_name": "1.0.0", "version_code": "1", "signing": "debug-only",
	}
	request := matrixSelectionRequest("sha256:"+strings.Repeat("a", 64), "/tmp/app-debug.apk", metadata, nil)
	metadata, ok := request["metadata"].(map[string]string)
	require.True(t, ok)
	require.Equal(t, "com.example.custom", metadata["package_name"])
	require.Equal(t, "android-debug:"+strings.Repeat("a", 64), metadata["immutable_ref"])
	require.Equal(t, "1", metadata["version_code"])
	require.Equal(t, "hello-mobile", request["scenario_name"])
}

func TestSelectedJourneyIDsParsesFocusedSelectionsAndRejectsAmbiguousInput(t *testing.T) {
	journeys, err := selectedJourneyIDs(" android-scenario-app-offline-transition-v1, android-scenario-app-verify-v1 ")
	require.NoError(t, err)
	require.Equal(t, []map[string]string{
		{"journey_id": "android-scenario-app-offline-transition-v1"},
		{"journey_id": "android-scenario-app-verify-v1"},
	}, journeys)
	for _, invalid := range []string{",offline", "offline,", "offline,,verify", "offline,offline", "offline/verify"} {
		_, err := selectedJourneyIDs(invalid)
		require.Error(t, err, invalid)
	}
	journeys, err = selectedJourneyIDs(" ")
	require.NoError(t, err)
	require.Nil(t, journeys)
}

func TestMatrixSelectionCanNameOneExactTargetWithoutSubstitution(t *testing.T) {
	body, err := json.Marshal(androidCatalog{Targets: []catalogTarget{
		{Descriptor: map[string]any{"target_id": "android-phone", "available": true}, Kind: "local"},
		{Descriptor: map[string]any{"target_id": "android:emulator:api36", "available": false, "reason": "booting"}, Kind: "local"},
	}})
	require.NoError(t, err)

	target, err := selectCatalogTarget(body, "android:emulator:api36")
	require.NoError(t, err)
	require.Equal(t, "android:emulator:api36", target["descriptor"].(map[string]any)["target_id"])
	require.Equal(t, false, target["descriptor"].(map[string]any)["available"])

	selection := matrixSelectionRequest("sha256:"+strings.Repeat("a", 64), "/tmp/app-debug.apk", map[string]string{"scenario_name": "hello-mobile"}, target)
	selectedTargets := selection["targets"].([]map[string]any)
	require.Len(t, selectedTargets, 1)
	require.Equal(t, "android:emulator:api36", selectedTargets[0]["descriptor"].(map[string]any)["target_id"])
	_, err = selectCatalogTarget(body, "android:missing")
	require.ErrorContains(t, err, "was not found in the live catalog")
}

func TestWithAuthProfileAddsOnlyConfiguredReference(t *testing.T) {
	metadata := map[string]string{"package_name": "com.example.app"}
	require.Equal(t, metadata, withAuthProfile(metadata, " auth-moto "))
	require.Equal(t, "auth-moto", metadata["auth_profile_id"])
	withoutProfile := withAuthProfile(map[string]string{"package_name": "com.example.app"}, " ")
	require.NotContains(t, withoutProfile, "auth_profile_id")
}

func TestMatrixArtifactMetadataFailsClosedOnMismatchedIdentity(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	artifact := artifactDescriptor{
		ImmutableRef: "android-debug:" + strings.Repeat("a", 64), Checksum: digest,
		PackageName: "com.example.generated", VersionName: "1.0.0", VersionCode: 1,
	}
	metadata, err := matrixArtifactMetadata(digest, artifact.ImmutableRef, "com.example.generated", "", artifact)
	require.NoError(t, err)
	require.Equal(t, defaultAndroidScenarioName, metadata["scenario_name"])
	require.Equal(t, "debug-only", metadata["signing"])
	require.ErrorContains(t, func() error {
		_, err := matrixArtifactMetadata(digest, artifact.ImmutableRef, "com.example.other", "", artifact)
		return err
	}(), "does not match owner-produced package")

	artifact.Checksum = "sha256:" + strings.Repeat("b", 64)
	_, err = matrixArtifactMetadata(digest, artifact.ImmutableRef, "", "", artifact)
	require.ErrorContains(t, err, "does not match the requested immutable ref and digest")
}

func TestMatrixArtifactMetadataRequiresReleaseSigningIdentity(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	artifact := artifactDescriptor{
		ImmutableRef: "android-release:" + strings.Repeat("a", 64), Checksum: digest,
		PackageName: "com.example.generated", VersionName: "1.0.0", VersionCode: 1, Signing: "secrets-manager",
	}
	_, err := matrixArtifactMetadata(digest, artifact.ImmutableRef, "", "", artifact)
	require.ErrorContains(t, err, "missing its signing identity")
}

func TestInstallJourneyRequiresImmutableRefAndBuildIdentity(t *testing.T) {
	digest := strings.Repeat("a", 64)
	artifact := builtArtifact{
		ImmutableRef: "android-debug:" + digest,
		Checksum:     "sha256:" + digest,
		Metadata: map[string]string{
			"package_name": "com.example.generated", "version_name": "1.2.3", "version_code": "12", "signing": "debug-only",
		},
	}
	if !immutableArtifactMetadataComplete(artifact) {
		t.Fatal("complete debug artifact identity was rejected")
	}
	artifact.ImmutableRef = "android-debug:" + strings.Repeat("b", 64)
	if immutableArtifactMetadataComplete(artifact) {
		t.Fatal("mismatched artifact ref was accepted")
	}
	artifact.ImmutableRef = "android-release:" + digest
	if immutableArtifactMetadataComplete(artifact) {
		t.Fatal("release artifact without signing identity was accepted")
	}
}
