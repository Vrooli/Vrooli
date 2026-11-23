package deployments

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"deployment-manager/profiles"
	"deployment-manager/shared"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	"github.com/vrooli/vrooli/packages/credential-authority-go"
)

const androidTemporaryJourney = "android-scenario-app-temporary-test-v1"

var fullGitCommitPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

type AndroidRamp interface {
	Start(context.Context, AndroidValidationRequest) (*AndroidValidationResult, error)
	Status(context.Context, string) (*AndroidValidationResult, error)
}

type AndroidValidationRequest struct {
	ProfileID     string `json:"profile_id"`
	GitCommitHash string `json:"git_commit_hash"`
	ArtifactRef   string `json:"artifact_ref"`
	TargetID      string `json:"target_id,omitempty"`
	Scenario      string `json:"-"`
}

type AndroidValidationResult struct {
	RunID          string         `json:"run_id"`
	State          string         `json:"state"`
	ProfileID      string         `json:"profile_id"`
	GitCommitHash  string         `json:"git_commit_hash"`
	MatrixID       string         `json:"matrix_id"`
	Scenario       string         `json:"scenario"`
	ArtifactRef    string         `json:"artifact_ref"`
	ArtifactDigest string         `json:"artifact_digest"`
	TargetID       string         `json:"target_id"`
	TargetName     string         `json:"target_name"`
	GatePassed     bool           `json:"gate_passed"`
	ReleaseReport  map[string]any `json:"release_report"`
	Run            map[string]any `json:"validation_run"`
}

// AndroidValidationHandler is a deployment-manager operation that delegates
// artifact verification, device selection, installation, journey execution,
// and evidence reporting to scenario-to-android.
type AndroidValidationHandler struct {
	profiles      profiles.Repository
	ramp          AndroidRamp
	authorize     func(context.Context) error
	authorizeRead func(context.Context) error
}

func NewAndroidValidationHandler(repository profiles.Repository, ramp AndroidRamp, authorize, authorizeRead func(context.Context) error) *AndroidValidationHandler {
	return &AndroidValidationHandler{profiles: repository, ramp: ramp, authorize: authorize, authorizeRead: authorizeRead}
}

func (h *AndroidValidationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.authorize != nil {
		if err := h.authorize(r.Context()); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
	}
	var request AndroidValidationRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid Android validation request: " + err.Error()})
		return
	}
	request.ProfileID = strings.TrimSpace(request.ProfileID)
	request.GitCommitHash = strings.TrimSpace(request.GitCommitHash)
	request.ArtifactRef = strings.TrimSpace(request.ArtifactRef)
	request.TargetID = strings.TrimSpace(request.TargetID)
	if request.ProfileID == "" || request.GitCommitHash == "" || request.ArtifactRef == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "profile_id, git_commit_hash, and artifact_ref are required"})
		return
	}
	if !fullGitCommitPattern.MatchString(request.GitCommitHash) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "git_commit_hash must be a full 40- or 64-character hexadecimal commit"})
		return
	}
	if h.profiles == nil || h.ramp == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Android validation owner is unavailable"})
		return
	}
	profile, err := h.profiles.Get(r.Context(), request.ProfileID)
	if err != nil || profile == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "deployment profile not found"})
		return
	}
	request.Scenario = strings.TrimSpace(profile.Scenario)
	result, err := h.ramp.Start(r.Context(), request)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if result == nil || result.RunID == "" || result.ArtifactRef != request.ArtifactRef || result.Scenario != profile.Scenario || result.ProfileID != request.ProfileID || result.GitCommitHash != request.GitCommitHash {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "scenario-to-android returned an incomplete or mismatched validation identity"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusAccepted, result)
}

func (h *AndroidValidationHandler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if h.authorizeRead != nil {
		if err := h.authorizeRead(r.Context()); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
	}
	if h.ramp == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Android validation owner is unavailable"})
		return
	}
	runID := strings.TrimSpace(mux.Vars(r)["run_id"])
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id is required"})
		return
	}
	result, err := h.ramp.Status(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if result == nil || result.RunID != runID || result.ProfileID == "" || result.GitCommitHash == "" || result.ArtifactRef == "" {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "scenario-to-android returned incomplete run identity"})
		return
	}
	if result.State != "completed" && result.State != "failed" && result.State != "cancelled" {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusAccepted, result)
		return
	}
	reportError, _ := result.ReleaseReport["error"].(string)
	if strings.TrimSpace(reportError) != "" {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "scenario-to-android completed without a confirmed deployment-manager evidence receipt", "validation": result})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if strings.TrimSpace(fmt.Sprint(result.ReleaseReport["reported_at"])) == "" {
		writeJSON(w, http.StatusAccepted, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HTTPAndroidRamp uses only producer-owned immutable refs and matrix routes.
// It never sends local paths to scenario-to-android.
type HTTPAndroidRamp struct {
	BaseURL   string
	Client    *http.Client
	Tokens    *servicetoken.Tokens
	Authorize func(context.Context, *http.Request) error
}

func NewHTTPAndroidRamp() (*HTTPAndroidRamp, error) {
	base := strings.TrimSpace(os.Getenv("SCENARIO_TO_ANDROID_URL"))
	if base != "" {
		var err error
		base, err = shared.ValidateServiceURL(base)
		if err != nil {
			return nil, fmt.Errorf("invalid scenario-to-android URL: %w", err)
		}
	}
	return &HTTPAndroidRamp{BaseURL: base, Client: &http.Client{Timeout: 25 * time.Minute}, Tokens: deploymentManagerAndroidTokens()}, nil
}

const deploymentManagerAndroidPrincipal = "deployment-manager-scenario-to-android"

func deploymentManagerAndroidTokens() *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Name:   "deployment-manager scenario-to-android validation",
		Expect: servicetoken.ExpectIdentity("deployment-manager-scenario-to-android", "scenario-authenticator:scenario-to-android", "scenario-to-android:read", "scenario-to-android:write"),
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity("scenario-authenticator/service-principal/" + deploymentManagerAndroidPrincipal + "/scenario-to-android")
			if err != nil {
				return "", err
			}
			return authority.Require(identity, "credential")
		},
	}, servicetoken.WithExchanger(servicetoken.AuthenticatorExchanger(&http.Client{Timeout: 5 * time.Second}, func(ctx context.Context) (string, error) {
		if configured := strings.TrimSpace(os.Getenv("SCENARIO_AUTHENTICATOR_API_URL")); configured != "" {
			return strings.TrimRight(configured, "/"), nil
		}
		return discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
	})))
}

func (c *HTTPAndroidRamp) authorize(ctx context.Context, req *http.Request) error {
	if c.Authorize != nil {
		return c.Authorize(ctx, req)
	}
	if c.Tokens == nil {
		return errors.New("scenario-to-android service identity is not configured")
	}
	value, err := c.Tokens.Authorization(ctx)
	if err != nil {
		return fmt.Errorf("scenario-to-android service identity: %w", err)
	}
	req.Header.Set("Authorization", value)
	return nil
}

type androidArtifactMetadata struct {
	ImmutableRef    string `json:"immutable_ref"`
	Checksum        string `json:"apk_checksum"`
	PackageName     string `json:"package_name"`
	VersionName     string `json:"version_name"`
	VersionCode     int64  `json:"version_code"`
	Signing         string `json:"signing"`
	SigningIdentity string `json:"signing_identity"`
}

type androidRunView struct {
	RunID  string `json:"run_id"`
	Matrix struct {
		MatrixID       string `json:"matrix_id"`
		ScenarioName   string `json:"scenario_name"`
		ArtifactDigest string `json:"artifact_digest"`
		Targets        []struct {
			TargetID    string `json:"target_id"`
			DisplayName string `json:"display_name"`
		} `json:"targets"`
	} `json:"matrix"`
	State     string `json:"state"`
	Selection struct {
		Metadata            map[string]string `json:"metadata"`
		DeploymentProfileID string            `json:"deployment_profile_id"`
		DeploymentGitCommit string            `json:"deployment_git_commit"`
	} `json:"selection"`
	Gate struct {
		Passed bool `json:"passed"`
	} `json:"gate"`
	ReleaseReport map[string]any `json:"release_report"`
}

func (c *HTTPAndroidRamp) Start(ctx context.Context, request AndroidValidationRequest) (*AndroidValidationResult, error) {
	if c == nil || strings.TrimSpace(c.BaseURL) == "" {
		if c == nil {
			return nil, errors.New("scenario-to-android client is not configured")
		}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		var err error
		baseURL, err = discovery.ResolveScenarioURLDefault(ctx, "scenario-to-android")
		if err != nil {
			return nil, fmt.Errorf("resolve scenario-to-android: %w", err)
		}
		baseURL, err = shared.ValidateServiceURL(baseURL)
		if err != nil {
			return nil, err
		}
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Minute}
	}
	call := func(method, path string, payload any) (map[string]any, error) {
		var body io.Reader
		if payload != nil {
			encoded, err := json.Marshal(payload)
			if err != nil {
				return nil, err
			}
			body = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if err := c.authorize(ctx, req); err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%s %s: %w", method, path, err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("scenario-to-android %s %s returned %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(data)))
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, fmt.Errorf("decode scenario-to-android response: %w", err)
		}
		return decoded, nil
	}

	query := url.Values{"immutable_ref": []string{request.ArtifactRef}}
	metadataBody, err := call(http.MethodGet, "/api/v1/android/artifacts/metadata?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	metadataBytes, _ := json.Marshal(metadataBody)
	var metadata androidArtifactMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, err
	}
	if metadata.ImmutableRef != request.ArtifactRef || !strings.HasPrefix(metadata.Checksum, "sha256:") || metadata.PackageName == "" || metadata.VersionName == "" || metadata.VersionCode <= 0 || metadata.Signing == "" {
		return nil, errors.New("scenario-to-android returned incomplete or mismatched immutable artifact metadata")
	}

	if request.Scenario == "" {
		return nil, errors.New("deployment profile has no scenario")
	}
	catalogBody, err := call(http.MethodGet, "/api/v1/validation/catalog?scenario="+url.QueryEscape(request.Scenario), nil)
	if err != nil {
		return nil, err
	}
	entries, ok := catalogBody["targets"].([]any)
	if !ok {
		return nil, errors.New("scenario-to-android catalog returned no target list")
	}
	var selected map[string]any
	var physicalCandidates []map[string]any
	for _, entry := range entries {
		candidate, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		descriptor, ok := candidate["descriptor"].(map[string]any)
		if !ok {
			continue
		}
		targetID, _ := descriptor["target_id"].(string)
		if targetID == "" || strings.HasPrefix(targetID, "android:emulator:") {
			continue
		}
		if request.TargetID != "" {
			if targetID == request.TargetID {
				selected = candidate
				break
			}
		} else if available, _ := descriptor["available"].(bool); available {
			physicalCandidates = append(physicalCandidates, candidate)
		}
	}
	if request.TargetID != "" && selected == nil {
		return nil, fmt.Errorf("requested Android target %q is not in the scenario-to-android catalog", request.TargetID)
	}
	if request.TargetID != "" {
		descriptor, _ := selected["descriptor"].(map[string]any)
		available, _ := descriptor["available"].(bool)
		if !available {
			reason, _ := descriptor["reason"].(string)
			if reason == "" {
				reason = "scenario-to-android reports the target unavailable"
			}
			return nil, fmt.Errorf("requested Android target %q is unavailable: %s", request.TargetID, reason)
		}
	}
	if request.TargetID == "" {
		if len(physicalCandidates) == 0 {
			return nil, errors.New("no available physical Android target; specify target_id to request an unavailable exact target")
		}
		if len(physicalCandidates) > 1 {
			return nil, fmt.Errorf("multiple physical Android targets are available; specify target_id to choose one: %s", androidTargetIDs(physicalCandidates))
		}
		selected = physicalCandidates[0]
	}
	descriptor := selected["descriptor"].(map[string]any)
	targetID, _ := descriptor["target_id"].(string)
	targetName, _ := descriptor["display_name"].(string)
	selection := map[string]any{
		"scenario_name": request.Scenario, "artifact_digest": metadata.Checksum,
		"deployment_profile_id": request.ProfileID, "deployment_git_commit": request.GitCommitHash,
		"idempotency_key":      androidIdempotencyKey(request, targetID),
		"environment_profiles": []int{1}, "max_concurrency": 1,
		"journeys": []map[string]string{{"journey_id": androidTemporaryJourney}},
		"targets":  []map[string]any{selected},
		"metadata": map[string]string{
			"immutable_ref": metadata.ImmutableRef, "scenario_name": request.Scenario, "install_mode": "temporary",
		},
	}
	matrixBody, err := call(http.MethodPost, "/api/v1/android/matrices", map[string]any{"artifact_ref": request.ArtifactRef, "selection": selection})
	if err != nil {
		return nil, err
	}
	runID, _ := matrixBody["run_id"].(string)
	if runID == "" {
		return nil, errors.New("scenario-to-android matrix creation returned no run_id")
	}
	if _, err = call(http.MethodPost, "/api/v1/validation/matrices/"+url.PathEscape(runID)+"/start", nil); err != nil {
		return nil, err
	}
	runBody, err := call(http.MethodGet, "/api/v1/validation/matrices/"+url.PathEscape(runID), nil)
	if err != nil {
		return nil, err
	}
	return androidValidationResult(runBody, request.ArtifactRef, metadata.Checksum, targetID, targetName)
}

func androidIdempotencyKey(request AndroidValidationRequest, targetID string) string {
	identity := strings.Join([]string{request.ProfileID, request.Scenario, request.GitCommitHash, request.ArtifactRef, targetID, androidTemporaryJourney}, "\x00")
	digest := sha256.Sum256([]byte(identity))
	return "deployment-manager-android:" + hex.EncodeToString(digest[:])
}

func (c *HTTPAndroidRamp) Status(ctx context.Context, runID string) (*AndroidValidationResult, error) {
	if c == nil {
		return nil, errors.New("scenario-to-android client is not configured")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		var err error
		baseURL, err = discovery.ResolveScenarioURLDefault(ctx, "scenario-to-android")
		if err != nil {
			return nil, fmt.Errorf("resolve scenario-to-android: %w", err)
		}
		baseURL, err = shared.ValidateServiceURL(baseURL)
		if err != nil {
			return nil, err
		}
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 25 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/v1/validation/matrices/"+url.PathEscape(runID), nil)
	if err != nil {
		return nil, err
	}
	if err := c.authorize(ctx, req); err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("read scenario-to-android matrix %s: %w", runID, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("scenario-to-android matrix lookup returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("decode scenario-to-android matrix: %w", err)
	}
	return androidValidationResult(body, "", "", "", "")
}

func androidValidationResult(runBody map[string]any, expectedRef, expectedDigest, expectedTargetID, expectedTargetName string) (*AndroidValidationResult, error) {
	viewBytes, _ := json.Marshal(runBody)
	var view androidRunView
	if err := json.Unmarshal(viewBytes, &view); err != nil {
		return nil, fmt.Errorf("decode Android validation result: %w", err)
	}
	if view.RunID == "" || view.Matrix.MatrixID == "" || view.Matrix.ScenarioName == "" || view.Matrix.ArtifactDigest == "" {
		return nil, errors.New("scenario-to-android returned a validation result with mismatched run or artifact identity")
	}
	if expectedRef != "" && view.Matrix.ArtifactDigest != expectedDigest {
		return nil, errors.New("scenario-to-android returned a validation result with mismatched artifact digest")
	}
	if len(view.Matrix.Targets) != 1 || (expectedTargetID != "" && view.Matrix.Targets[0].TargetID != expectedTargetID) {
		return nil, errors.New("scenario-to-android validation result does not preserve the exact selected Android target")
	}
	artifactRef := view.Selection.Metadata["immutable_ref"]
	if expectedRef != "" && artifactRef != expectedRef {
		return nil, errors.New("scenario-to-android validation result does not preserve the requested immutable artifact ref")
	}
	view.ReleaseReport = stringMapAny(view.ReleaseReport)
	profileID := view.Selection.DeploymentProfileID
	commit := view.Selection.DeploymentGitCommit
	if profileID == "" || commit == "" {
		return nil, errors.New("scenario-to-android validation run is not bound to a Deployment Manager profile and commit")
	}
	targetID := view.Matrix.Targets[0].TargetID
	targetName := view.Matrix.Targets[0].DisplayName
	if expectedTargetID != "" && (targetID != expectedTargetID || targetName != expectedTargetName) {
		return nil, errors.New("scenario-to-android changed the selected target identity")
	}
	return &AndroidValidationResult{RunID: view.RunID, State: view.State, ProfileID: profileID, GitCommitHash: commit, MatrixID: view.Matrix.MatrixID, Scenario: view.Matrix.ScenarioName, ArtifactRef: artifactRef, ArtifactDigest: view.Matrix.ArtifactDigest, TargetID: targetID, TargetName: targetName, GatePassed: view.Gate.Passed, ReleaseReport: view.ReleaseReport, Run: runBody}, nil
}

func androidTargetIDs(values []map[string]any) string {
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if d, ok := value["descriptor"].(map[string]any); ok {
			if id, ok := d["target_id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}
	return strings.Join(ids, ", ")
}

func stringMapAny(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}
