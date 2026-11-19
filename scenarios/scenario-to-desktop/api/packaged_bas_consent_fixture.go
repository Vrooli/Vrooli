package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"scenario-to-desktop-api/captures"

	"github.com/vrooli/envkit-go"
)

const (
	packagedBASConsentJourneyID = "browser-automation-studio:a15/packaged-bas-consent"
	packagedBASConsentDigest    = "sha256:fab3e3715788ab92333b9d03219c2c3ccd97fba56499b9d374c64e6135ff61ac"
)

type packagedBASConsentEvidence struct {
	ArtifactDigest string `json:"artifact_digest"`
	ArtifactPath   string `json:"artifact_path"`
	Auth           struct {
		PKCEVerified bool `json:"pkce_verified"`
		TokenFound   bool `json:"token_found"`
	} `json:"auth"`
	Entitlement struct {
		AppKey              string `json:"app_key"`
		Feature             string `json:"feature"`
		SignedCanUseAI      bool   `json:"signed_can_use_ai"`
		LegacyGlobalFeature []any  `json:"legacy_global_features"`
	} `json:"entitlement"`
	ConsentSource                string `json:"consent_source"`
	ConsentGrantID               string `json:"consent_grant_id"`
	ConsentPersistedAcrossReload bool   `json:"consent_persisted_across_reload"`
	AIResponses                  struct {
		Grant   int `json:"grant"`
		Revoked int `json:"revoked"`
		Forged  int `json:"forged"`
	} `json:"ai_responses"`
	GrantDispatches   int  `json:"grant_dispatches"`
	RevokedDispatches int  `json:"revoked_dispatches"`
	ForgedDispatches  int  `json:"forged_dispatches"`
	RevokedRefused    bool `json:"revoked_refused"`
	ForgedRefused     bool `json:"forged_refused"`
}

func (e validationMatrixLocalExecutor) executePackagedBASConsentFixture(ctx context.Context, request validationmatrix.CellRequest, artifactPath string) validationmatrix.CellResult {
	fail := func(reason string) validationmatrix.CellResult {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: reason}
	}
	if request.Cell.GetScenarioName() != "browser-automation-studio" || request.Cell.GetJourneyId() != packagedBASConsentJourneyID {
		return fail("packaged BAS consent fixture was selected outside its registered scenario journey")
	}
	if request.ArtifactDigest != packagedBASConsentDigest {
		return fail(fmt.Sprintf("packaged BAS consent fixture requires artifact %s; got %s", packagedBASConsentDigest, request.ArtifactDigest))
	}
	if e.captures == nil {
		return fail("packaged BAS consent fixture has no durable capture store")
	}
	if ctx.Err() != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "packaged BAS consent fixture was cancelled before start"}
	}

	gatewayDir := filepath.Join(e.scenarioRoot, "ai-gateway", "api")
	if _, err := os.Stat(filepath.Join(gatewayDir, "go.mod")); err != nil {
		return fail(fmt.Sprintf("locate AI Gateway consent fixture module: %v", err))
	}
	evidenceDir, err := os.MkdirTemp("", "s2d-packaged-bas-consent-")
	if err != nil {
		return fail(fmt.Sprintf("create isolated packaged journey workspace: %v", err))
	}
	defer os.RemoveAll(evidenceDir)
	evidencePath := filepath.Join(evidenceDir, "packaged-bas-consent.json")
	screenshotPath := filepath.Join(evidenceDir, "packaged-bas-consent.png")

	command := exec.CommandContext(ctx, "go", "test", "./internal/identity", "-run", "^TestGatewayConsentBrowserJourney$", "-count=1", "-v")
	command.Dir = gatewayDir
	command.Env = envkit.Toolchain(envkit.Env(packagedConsentEnvironment(artifactPath, request.ArtifactDigest, evidencePath, screenshotPath)), envkit.ToolchainOptions{})
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN, Reason: "packaged BAS consent fixture was cancelled"}
		}
		return fail(fmt.Sprintf("run packaged BAS consent fixture: %v: %s", err, boundedFixtureOutput(string(output))))
	}

	data, err := os.ReadFile(evidencePath)
	if err != nil {
		return fail(fmt.Sprintf("read packaged BAS consent evidence: %v", err))
	}
	var evidence packagedBASConsentEvidence
	if err := json.Unmarshal(data, &evidence); err != nil {
		return fail(fmt.Sprintf("decode packaged BAS consent evidence: %v", err))
	}
	if evidence.ArtifactDigest != request.ArtifactDigest || filepath.Clean(evidence.ArtifactPath) != filepath.Clean(artifactPath) {
		return fail("packaged BAS consent evidence is bound to a different artifact")
	}
	if !evidence.Auth.PKCEVerified || !evidence.Auth.TokenFound || evidence.Entitlement.AppKey != "browser-automation-studio" || evidence.Entitlement.Feature != "ai" || !evidence.Entitlement.SignedCanUseAI || len(evidence.Entitlement.LegacyGlobalFeature) != 0 {
		return fail("packaged BAS consent fixture omitted native PKCE or signed app-scoped AI entitlement proof")
	}
	targetID := strings.TrimSpace(request.Cell.GetTargetId())
	if targetID == "" {
		return fail("packaged BAS consent fixture has no selected desktop target identity")
	}
	if evidence.ConsentSource != "own-key" || evidence.ConsentGrantID == "" || !evidence.ConsentPersistedAcrossReload || evidence.AIResponses.Grant != 200 || evidence.AIResponses.Revoked < 400 || evidence.AIResponses.Forged < 400 {
		return fail("packaged BAS consent fixture omitted grant, reload, revoke, or Gateway refusal assertions")
	}
	if evidence.GrantDispatches != 1 || evidence.RevokedDispatches != evidence.GrantDispatches || evidence.ForgedDispatches != evidence.RevokedDispatches || !evidence.RevokedRefused || !evidence.ForgedRefused {
		return fail("packaged BAS consent fixture did not prove one grant dispatch and revoked/forged refusal")
	}

	capture, err := e.captures.SaveCapture(request.Cell.GetScenarioName(), captures.CaptureJourney, request.Cell.GetCellId(), evidencePath, 0, 0, 0)
	if err != nil {
		return fail(fmt.Sprintf("persist packaged BAS consent evidence: %v", err))
	}
	uri := "/api/v1/captures/" + request.Cell.GetScenarioName() + "/" + capture.ID + "/file"
	screenshot, err := e.captures.SaveCapture(request.Cell.GetScenarioName(), captures.CaptureScreenshot, request.Cell.GetCellId(), screenshotPath, 1920, 1080, 0)
	if err != nil {
		return fail(fmt.Sprintf("persist packaged BAS consent screenshot: %v", err))
	}
	screenshotURI := "/api/v1/captures/" + request.Cell.GetScenarioName() + "/" + screenshot.ID + "/file"
	selectionEvidence := fmt.Sprintf("scenario=%s artifact=%s target=%s cell=%s", request.Cell.GetScenarioName(), request.ArtifactDigest, targetID, request.Cell.GetCellId())
	selectionHash := sha256.Sum256([]byte(selectionEvidence))
	selectionChecksum := "sha256:" + hex.EncodeToString(selectionHash[:])
	return validationmatrix.CellResult{
		Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS,
		Reason:      "native PKCE, BAS-scoped signed AI entitlement, consent reload/revoke, and fake Gateway dispatch/refusal passed",
		Identity:    validationmatrix.ExecutionIdentity{RunID: request.RunID, ArtifactDigest: request.ArtifactDigest},
		Evidence: []*domainv1.LayeredEvidence{
			{Kind: domainv1.LayeredEvidence_KIND_BAS_WORKFLOW, EvidenceId: capture.ID, Uri: uri, Sha256: capture.Checksum, MediaType: stringPointer("application/json"), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "assertion-" + request.Cell.GetCellId(), Uri: uri, Sha256: capture.Checksum, MediaType: stringPointer("application/json"), Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_TARGET, EvidenceId: "target-" + targetID, Uri: "validation://target/" + targetID, Sha256: selectionChecksum, Redacted: true},
			{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: screenshot.ID, Uri: screenshotURI, Sha256: screenshot.Checksum, MediaType: stringPointer("image/png"), Redacted: true},
		},
	}
}

func packagedConsentEnvironment(artifactPath, digest, evidencePath, screenshotPath string) []string {
	allowed := map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "GOCACHE": true, "GOMODCACHE": true,
		"GOPATH": true, "GOROOT": true, "GOTOOLCHAIN": true, "GOFLAGS": true, "GOPROXY": true,
		"GOSUMDB": true, "CGO_ENABLED": true, "DISPLAY": true, "XAUTHORITY": true,
		"DBUS_SESSION_BUS_ADDRESS": true, "XDG_RUNTIME_DIR": true,
	}
	env := make([]string, 0, len(allowed)+8)
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if ok && allowed[name] {
			env = append(env, item)
		}
	}
	env = append(env,
		"A15_CONSENT_BROWSER=1",
		"A15_CONSENT_ELECTRON=1",
		"A15_CONSENT_SIDECAR_ONLY=1",
		"A15_APPIMAGE_PATH="+artifactPath,
		"A15_ARTIFACT_DIGEST="+digest,
		"A15_EVIDENCE_PATH="+evidencePath,
		"A15_SCREENSHOT_PATH="+screenshotPath,
	)
	return env
}

func boundedFixtureOutput(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 4000 {
		return value[len(value)-4000:]
	}
	return value
}

func stringPointer(value string) *string {
	return &value
}
