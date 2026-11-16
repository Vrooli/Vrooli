package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/captures"
	"scenario-to-desktop-api/livedesktop"
	"scenario-to-desktop-api/target"
)

const packagedWebConsoleMonetizationJourneyID = "web-console:sp1/packaged-metered-ai"
const packagedWebConsoleIdentity = "sp1-web-console@example.invalid"

type packagedWebConsoleMonetizationEvidence struct {
	SchemaVersion  int                             `json:"schema_version"`
	Scenario       string                          `json:"scenario"`
	ArtifactPath   string                          `json:"artifact_path"`
	ArtifactDigest string                          `json:"artifact_digest"`
	TargetID       string                          `json:"target_id"`
	Online         packagedWebConsolePhaseEvidence `json:"signed_online"`
	Offline        packagedWebConsolePhaseEvidence `json:"offline_restart"`
}

type packagedWebConsolePhaseEvidence struct {
	SchemaVersion         int      `json:"schema_version"`
	Scenario              string   `json:"scenario"`
	ArtifactPath          string   `json:"artifact_path"`
	ArtifactDigest        string   `json:"artifact_digest"`
	TargetID              string   `json:"target_id"`
	RootAPIOperation      string   `json:"root_api_operation"`
	DisabledSources       []string `json:"disabled_sources"`
	VrooliSourceEnabled   bool     `json:"vrooli_source_enabled"`
	IdentityHint          string   `json:"identity_hint"`
	HTTPStatus            int      `json:"http_status"`
	ErrorType             string   `json:"error_type"`
	UnauthenticatedMeter  bool     `json:"unauthenticated_meter_refused"`
	Phase                 string   `json:"phase"`
	Identity              string   `json:"identity"`
	LeaseStatus           string   `json:"lease_status"`
	SignedSessionAccepted bool     `json:"signed_session_accepted"`
	SessionConfigured     bool     `json:"session_configured"`
	PersistedLeaseValid   bool     `json:"persisted_lease_valid"`
	LPBSUnavailable       bool     `json:"lpbs_unavailable"`
	OfflineSummaryStatus  int      `json:"offline_summary_status"`
	OfflineSummaryError   string   `json:"offline_summary_error"`
}

func (e validationMatrixLocalExecutor) executePackagedWebConsoleMonetizationFixture(ctx context.Context, request validationmatrix.CellRequest, artifactPath string) validationmatrix.CellResult {
	fail := func(reason string, evidence ...*domainv1.LayeredEvidence) validationmatrix.CellResult {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: reason, Evidence: evidence}
	}
	if request.Cell.GetScenarioName() != "web-console" || request.Cell.GetJourneyId() != packagedWebConsoleMonetizationJourneyID {
		return fail("packaged Web Console monetization fixture was selected outside its registered scenario journey")
	}
	if request.DeploymentMode != "bundled" || !hasBundledAPIService(artifactPath) {
		return fail("packaged Web Console monetization fixture requires the exact artifact's bundled root API")
	}
	if request.Target == nil || !request.Target.GetAvailable() || strings.TrimSpace(request.Cell.GetTargetId()) == "" {
		return fail("packaged Web Console monetization fixture requires an available target identity")
	}
	if e.desktop == nil || e.captures == nil {
		return fail("packaged Web Console monetization fixture requires target and durable evidence owners")
	}
	if ctx.Err() != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "packaged Web Console monetization fixture was cancelled before start"}
	}

	session, err := e.desktop.StartSession(ctx, livedesktop.SessionConfig{Width: 1920, Height: 1080, ScenarioName: "web-console", Platform: "linux"})
	if err != nil {
		return fail(fmt.Sprintf("start isolated Web Console desktop session: %v", err))
	}
	defer func() { _ = e.desktop.StopSession(session.ID) }()

	contextID := "web-console-monetization-" + request.Cell.GetCellId()
	offlineRoot, err := os.MkdirTemp("", "s2d-web-console-private-")
	if err != nil {
		return fail(fmt.Sprintf("allocate private Web Console runtime root: %v", err))
	}
	if err := os.Remove(offlineRoot); err != nil {
		return fail(fmt.Sprintf("reserve private Web Console runtime root: %v", err))
	}
	appDataDir := filepath.Join(offlineRoot, "profile", "app-data")
	jwksPath := filepath.Join(appDataDir, "fixture-jwks.json")
	upstreamHostURL, err := freeLoopbackHTTPOrigin()
	if err != nil {
		return fail(fmt.Sprintf("allocate isolated Web Console LPBS fixture endpoint: %v", err))
	}
	upstream, err := target.NewOfflineLoopbackUpstream(upstreamHostURL)
	if err != nil {
		return fail(fmt.Sprintf("prepare isolated Web Console LPBS fixture relay: %v", err))
	}
	session.SetEnvVars(map[string]string{
		"LPBS_URL":     upstream.LocalURL(),
		"APP_DATA_DIR": appDataDir,
	})
	targetInfo, err := e.desktop.LaunchElectronValidation(ctx, session.ID, artifactPath, target.ElectronLaunchOptions{
		ContextID: contextID, ScenarioName: "web-console", ArtifactDigest: request.ArtifactDigest,
		TargetID: request.Cell.GetTargetId(), JourneyID: packagedWebConsoleMonetizationJourneyID,
		ProfileID: profileID(request.Cell.GetEnvironmentProfile()), IsolationLeaseID: request.RunID + ":" + request.Cell.GetCellId(),
		BundledRuntime: true, OfflineNetwork: true, PrivateSecretService: true,
		OfflineDataRoot:          offlineRoot,
		OfflineWritablePaths:     []string{"profile/app-data"},
		OfflineLoopbackUpstreams: []target.OfflineLoopbackUpstream{upstream},
	}, target.RendererExpectation{URLPrefix: "http://127.0.0.1:"})
	if err != nil {
		return fail(fmt.Sprintf("launch exact bundled Web Console artifact: %v", err))
	}
	if err := target.ValidateAppTarget(targetInfo); err != nil {
		return fail(fmt.Sprintf("validate bundled Web Console target identity: %v", err))
	}
	if targetInfo.GetScenarioName() != "web-console" || targetInfo.GetTargetId() != request.Cell.GetTargetId() || targetInfo.GetArtifactDigest() != request.ArtifactDigest || strings.TrimSpace(targetInfo.GetApiUrl()) == "" {
		return fail("bundled Web Console target did not bind the exact scenario, target, artifact, and root API")
	}
	privateRoot := targetInfo.GetTestDatabaseRoot()
	if privateRoot != filepath.Join(offlineRoot, "test-database") {
		return fail("bundled Web Console target did not remain inside its assigned private runtime root")
	}
	evidenceDir, err := os.MkdirTemp("", "s2d-packaged-web-console-monetization-")
	if err != nil {
		return fail(fmt.Sprintf("create isolated Web Console evidence workspace: %v", err))
	}
	defer os.RemoveAll(evidenceDir)
	onlineEvidencePath := filepath.Join(evidenceDir, "web-console-signed-online.json")
	offlineEvidencePath := filepath.Join(evidenceDir, "web-console-offline-restart.json")
	combinedEvidencePath := filepath.Join(evidenceDir, "web-console-monetization.json")
	apiDir := filepath.Join(e.scenarioRoot, "web-console", "api")
	if _, err := os.Stat(filepath.Join(apiDir, "go.mod")); err != nil {
		return fail(fmt.Sprintf("locate Web Console monetization journey module: %v", err))
	}
	journeyCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	appImage, err := filepath.Abs(artifactPath)
	if err != nil {
		return fail(fmt.Sprintf("resolve exact Web Console artifact path: %v", err))
	}
	if err := runPackagedWebConsoleTest(journeyCtx, apiDir, packagedWebConsoleTestInputs{
		Phase: "signed-online", APIURL: targetInfo.GetApiUrl(),
		ArtifactPath: appImage, ArtifactDigest: request.ArtifactDigest,
		TargetID: request.Cell.GetTargetId(), EvidencePath: onlineEvidencePath,
		AppDataDir: appDataDir, JWKSPath: jwksPath, UpstreamListenURL: upstream.HostURL,
	}); err != nil {
		if ctx.Err() != nil {
			return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "packaged Web Console signed-online fixture was cancelled"}
		}
		return fail(fmt.Sprintf("run packaged Web Console signed-online session journey: %v", err))
	}
	online, err := readPackagedWebConsolePhaseEvidence(onlineEvidencePath, appImage, request.ArtifactDigest, request.Cell.GetTargetId(), "signed-online")
	if err != nil {
		return fail(fmt.Sprintf("validate packaged Web Console signed-online evidence: %v", err))
	}
	if !online.SignedSessionAccepted || !online.SessionConfigured || !strings.EqualFold(online.Identity, packagedWebConsoleIdentity) || !strings.EqualFold(online.LeaseStatus, "active") || !online.VrooliSourceEnabled || !containsString(online.DisabledSources, "ollama") || !containsString(online.DisabledSources, "openrouter") {
		return fail("packaged Web Console did not accept an LPBS-signed active session through the bundled root API")
	}

	if session.ElectronValidation == nil {
		return fail("packaged Web Console target lost its Electron process owner before offline restart")
	}
	stopCtx, stopCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := session.ElectronValidation.RequestExit(stopCtx); err != nil {
		stopCancel()
		return fail(fmt.Sprintf("request exact Web Console artifact exit before offline restart: %v", err))
	}
	if err := session.ElectronValidation.WaitForExit(stopCtx); err != nil {
		stopCancel()
		return fail(fmt.Sprintf("wait for exact Web Console artifact exit: %v", err))
	}
	stopCancel()
	restarted, err := session.ElectronValidation.RestartOffline(journeyCtx)
	if err != nil {
		return fail(fmt.Sprintf("restart exact Web Console artifact without LPBS upstream: %v", err))
	}
	session.SetElectronValidation(restarted)
	targetInfo = restarted.Target()
	if err := target.ValidateAppTarget(targetInfo); err != nil {
		return fail(fmt.Sprintf("validate offline-restarted Web Console target identity: %v", err))
	}
	if targetInfo.GetArtifactDigest() != request.ArtifactDigest || targetInfo.GetTargetId() != request.Cell.GetTargetId() || targetInfo.GetApiUrl() == "" {
		return fail("offline restart did not retain the exact artifact and target identity")
	}
	if err := runPackagedWebConsoleTest(journeyCtx, apiDir, packagedWebConsoleTestInputs{
		Phase: "offline-restart", APIURL: targetInfo.GetApiUrl(),
		ArtifactPath: appImage, ArtifactDigest: request.ArtifactDigest,
		TargetID: request.Cell.GetTargetId(), EvidencePath: offlineEvidencePath,
		AppDataDir: appDataDir, JWKSPath: jwksPath, PreviousEvidencePath: onlineEvidencePath,
	}); err != nil {
		if ctx.Err() != nil {
			return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "packaged Web Console offline-restart fixture was cancelled"}
		}
		return fail(fmt.Sprintf("run packaged Web Console offline-restart journey: %v", err))
	}
	offline, err := readPackagedWebConsolePhaseEvidence(offlineEvidencePath, appImage, request.ArtifactDigest, request.Cell.GetTargetId(), "offline-restart")
	if err != nil {
		return fail(fmt.Sprintf("validate packaged Web Console offline-restart evidence: %v", err))
	}
	if !offline.SignedSessionAccepted || !offline.SessionConfigured || !offline.PersistedLeaseValid || !strings.EqualFold(offline.Identity, packagedWebConsoleIdentity) || !strings.EqualFold(offline.LeaseStatus, "active") || !offline.LPBSUnavailable || offline.OfflineSummaryStatus != 401 || offline.OfflineSummaryError != "subscription session is not configured" || offline.HTTPStatus != 402 || offline.ErrorType != "credits_required" || !offline.UnauthenticatedMeter || !offline.VrooliSourceEnabled || !containsString(offline.DisabledSources, "ollama") || !containsString(offline.DisabledSources, "openrouter") {
		return fail("offline-restarted Web Console did not restore the signed lease and reject forged metered-AI use")
	}
	evidence := packagedWebConsoleMonetizationEvidence{
		SchemaVersion: 1, Scenario: "web-console", ArtifactPath: appImage,
		ArtifactDigest: request.ArtifactDigest, TargetID: request.Cell.GetTargetId(),
		Online: online, Offline: offline,
	}
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fail(fmt.Sprintf("encode combined Web Console monetization evidence: %v", err))
	}
	if err := os.WriteFile(combinedEvidencePath, append(data, '\n'), 0o600); err != nil {
		return fail(fmt.Sprintf("write combined Web Console monetization evidence: %v", err))
	}

	onlineCapture, err := e.captures.SaveCapture("web-console", captures.CaptureJourney, request.Cell.GetCellId()+"-signed-online", onlineEvidencePath, 0, 0, 0)
	if err != nil {
		return fail(fmt.Sprintf("persist Web Console signed-online evidence: %v", err))
	}
	offlineCapture, err := e.captures.SaveCapture("web-console", captures.CaptureJourney, request.Cell.GetCellId()+"-offline-restart", offlineEvidencePath, 0, 0, 0)
	if err != nil {
		return fail(fmt.Sprintf("persist Web Console offline-restart evidence: %v", err))
	}
	assertionCapture, err := e.captures.SaveCapture("web-console", captures.CaptureJourney, request.Cell.GetCellId()+"-combined", combinedEvidencePath, 0, 0, 0)
	if err != nil {
		return fail(fmt.Sprintf("persist combined Web Console evidence: %v", err))
	}
	screenshot, err := e.desktop.ExecuteAction(ctx, session.ID, "screenshot", nil)
	if err != nil {
		return fail(fmt.Sprintf("capture packaged Web Console desktop runtime: %v", err))
	}
	screenshotID, _ := screenshot.Data["capture_id"].(string)
	if screenshotID == "" {
		return fail("packaged Web Console desktop screenshot has no durable capture identity")
	}
	catalog, err := e.captures.Store().List("web-console")
	if err != nil {
		return fail(fmt.Sprintf("read packaged Web Console screenshot capture catalog: %v", err))
	}
	var screenshotCapture *captures.Capture
	for index := range catalog {
		if catalog[index].ID == screenshotID {
			copy := catalog[index]
			screenshotCapture = &copy
			break
		}
	}
	if screenshotCapture == nil {
		return fail("packaged Web Console desktop screenshot is not in the durable capture catalog")
	}

	targetText := fmt.Sprintf("scenario=web-console artifact=%s target=%s cell=%s API=%s", request.ArtifactDigest, request.Cell.GetTargetId(), request.Cell.GetCellId(), offline.RootAPIOperation)
	targetHash := sha256.Sum256([]byte(targetText))
	journeyURI := "/api/v1/captures/web-console/" + assertionCapture.ID + "/file"
	screenshotURI := "/api/v1/captures/web-console/" + screenshotCapture.ID + "/file"
	return validationmatrix.CellResult{
		Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS,
		Reason:      "exact bundled Web Console accepted a signed online session, restored the lease after restart with its only LPBS relay removed, observed the unconfigured-session 401, and refused forged metered AI with 402",
		Identity:    validationmatrix.ExecutionIdentity{RunID: request.RunID, ArtifactDigest: request.ArtifactDigest},
		Evidence: []*domainv1.LayeredEvidence{
			{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: onlineCapture.ID, Uri: "/api/v1/captures/web-console/" + onlineCapture.ID + "/file", Sha256: onlineCapture.Checksum, MediaType: stringPointer("application/json"), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: offlineCapture.ID, Uri: "/api/v1/captures/web-console/" + offlineCapture.ID + "/file", Sha256: offlineCapture.Checksum, MediaType: stringPointer("application/json"), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "assertion-" + request.Cell.GetCellId(), Uri: journeyURI, Sha256: assertionCapture.Checksum, MediaType: stringPointer("application/json"), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "target-" + request.Cell.GetTargetId(), Uri: "validation://target/" + request.Cell.GetTargetId(), Sha256: "sha256:" + hex.EncodeToString(targetHash[:]), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: screenshotCapture.ID, Uri: screenshotURI, Sha256: screenshotCapture.Checksum, MediaType: stringPointer("image/png"), Redacted: true},
		},
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

type packagedWebConsoleTestInputs struct {
	Phase, APIURL, ArtifactPath, ArtifactDigest, TargetID, EvidencePath string
	AppDataDir, JWKSPath, UpstreamListenURL, PreviousEvidencePath       string
}

func runPackagedWebConsoleTest(ctx context.Context, apiDir string, inputs packagedWebConsoleTestInputs) error {
	command := exec.CommandContext(ctx, "go", "test", ".", "-run", "^TestWebConsolePackagedMeteringRefusal$", "-count=1")
	command.Dir = apiDir
	command.Env = packagedWebConsoleMonetizationEnvironment(inputs)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go test failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func readPackagedWebConsolePhaseEvidence(path, artifactPath, digest, targetID, phase string) (packagedWebConsolePhaseEvidence, error) {
	var evidence packagedWebConsolePhaseEvidence
	data, err := os.ReadFile(path)
	if err != nil {
		return evidence, err
	}
	if err := json.Unmarshal(data, &evidence); err != nil {
		return evidence, err
	}
	if evidence.SchemaVersion != 1 || evidence.Scenario != "web-console" || evidence.ArtifactPath != artifactPath || evidence.ArtifactDigest != digest || evidence.TargetID != targetID || evidence.Phase != phase || evidence.RootAPIOperation == "" {
		return evidence, fmt.Errorf("evidence binding mismatch for phase %s", phase)
	}
	return evidence, nil
}

func freeLoopbackHTTPOrigin() (string, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", err
	}
	return "http://" + addr, nil
}

func packagedWebConsoleMonetizationEnvironment(inputs packagedWebConsoleTestInputs) []string {
	allowed := map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "GOCACHE": true, "GOMODCACHE": true,
		"GOPATH": true, "GOROOT": true, "GOTOOLCHAIN": true, "GOFLAGS": true, "GOPROXY": true,
		"GOSUMDB": true, "CGO_ENABLED": true,
	}
	env := make([]string, 0, len(allowed)+12)
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && allowed[name] {
			env = append(env, item)
		}
	}
	values := map[string]string{
		"WC_MONETIZATION_PHASE":           inputs.Phase,
		"WC_MONETIZATION_API_URL":         inputs.APIURL,
		"WC_MONETIZATION_APPIMAGE_PATH":   inputs.ArtifactPath,
		"WC_MONETIZATION_ARTIFACT_DIGEST": inputs.ArtifactDigest,
		"WC_MONETIZATION_TARGET_ID":       inputs.TargetID,
		"WC_MONETIZATION_EVIDENCE_PATH":   inputs.EvidencePath,
		"WC_MONETIZATION_APP_DATA_DIR":    inputs.AppDataDir,
		"WC_MONETIZATION_JWKS_PATH":       inputs.JWKSPath,
	}
	if inputs.UpstreamListenURL != "" {
		values["WC_MONETIZATION_UPSTREAM_LISTEN_ADDR"] = strings.TrimPrefix(inputs.UpstreamListenURL, "http://")
	}
	if inputs.PreviousEvidencePath != "" {
		values["WC_MONETIZATION_PREVIOUS_EVIDENCE_PATH"] = inputs.PreviousEvidencePath
	}
	for name, value := range values {
		env = append(env, name+"="+value)
	}
	return env
}
