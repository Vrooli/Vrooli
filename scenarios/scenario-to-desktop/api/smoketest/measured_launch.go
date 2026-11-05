package smoketest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MeasuredLaunchRequest struct {
	Scenario            string                      `json:"scenario"`
	Artifact            string                      `json:"artifact_path"`
	Condition           string                      `json:"condition"`
	Strategy            string                      `json:"strategy"`
	OptimizationToggles PackagedOptimizationToggles `json:"optimization_toggles"`
	Experiment          string                      `json:"experiment_id,omitempty"`
	Journey             string                      `json:"journey,omitempty"`
}

// MeasuredLaunchHandler exposes one measured run through the existing smoke
// test launcher. It owns a fresh artifact install directory and delegates both
// protocol and demo launches to the existing isolated/Xvfb path.
type MeasuredLaunchHandler struct {
	service Service
	store   Store
}

func NewMeasuredLaunchHandler(service Service, store Store) http.Handler {
	return &MeasuredLaunchHandler{service: service, store: store}
}

func (h *MeasuredLaunchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request MeasuredLaunchRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&request); err != nil {
		http.Error(w, "invalid measured-launch request: "+err.Error(), http.StatusBadRequest)
		return
	}
	request.Scenario = strings.TrimSpace(request.Scenario)
	request.Artifact = strings.TrimSpace(request.Artifact)
	request.Strategy = strings.TrimSpace(request.Strategy)
	if request.Scenario == "" || request.Artifact == "" {
		http.Error(w, "scenario and artifact_path are required", http.StatusBadRequest)
		return
	}
	if request.Condition != "cold" && request.Condition != "warm" {
		http.Error(w, "condition must be cold or warm", http.StatusBadRequest)
		return
	}
	if request.Strategy == "" {
		request.Strategy = "electron-sidecar"
	}
	if request.Strategy != "electron-sidecar" && request.Strategy != "electron-optimized" {
		http.Error(w, "strategy must be electron-sidecar or electron-optimized", http.StatusBadRequest)
		return
	}
	if request.Strategy == "electron-sidecar" && request.OptimizationToggles != (PackagedOptimizationToggles{}) {
		http.Error(w, "electron-sidecar requires every optimization toggle disabled", http.StatusBadRequest)
		return
	}
	if request.Strategy == "electron-optimized" && request.OptimizationToggles == (PackagedOptimizationToggles{}) {
		http.Error(w, "electron-optimized requires an enabled optimization toggle", http.StatusBadRequest)
		return
	}
	if h == nil || h.service == nil || h.store == nil {
		http.Error(w, "measured-launch service is unavailable", http.StatusServiceUnavailable)
		return
	}
	installedArtifact, artifactDigest, cleanup, err := copyMeasuredArtifact(request.Artifact)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer cleanup()
	id := uuid.NewString()
	if request.Experiment == "" {
		request.Experiment = uuid.NewString()
	}
	started := time.Now().UTC()
	status := &Status{
		SmokeTestID: id, ScenarioName: request.Scenario, Platform: h.service.CurrentPlatform(),
		Status: "running", ArtifactPath: installedArtifact, StartedAt: started,
		DeploymentMode: "bundled", CurrentState: StateInitializing,
		ScreenContentSource: "bundled_runtime", Logs: []string{"packaged-launch.v1 measured run"},
		RecordingConfig: &ScreenRecordingConfig{Enabled: true, DisplayWidth: 1920, DisplayHeight: 1080, FPS: 15},
		Measured:        true, Condition: request.Condition, Strategy: request.Strategy,
		ExperimentID: request.Experiment, ArtifactDigest: artifactDigest,
	}
	h.store.Save(status)
	RunSmokeTestRequest(r.Context(), h.service, SmokeTestRequest{
		SmokeTestID: id, ScenarioName: request.Scenario, ArtifactPath: installedArtifact,
		Platform: status.Platform, DeploymentMode: "bundled", JourneyID: request.Journey,
		Measured: true, Condition: request.Condition, Strategy: request.Strategy,
		ExperimentID: request.Experiment, ArtifactDigest: artifactDigest,
	})
	status, ok := h.store.Get(id)
	if !ok {
		http.Error(w, "measured launch status was not retained", http.StatusInternalServerError)
		return
	}
	sample, err := ReadPackagedLaunch(packagedLaunchPath(id, "demo"))
	if err != nil {
		http.Error(w, fmt.Sprintf("measured launch did not emit packaged-launch.v1: %v", err), http.StatusBadGateway)
		return
	}
	if sample.Strategy != request.Strategy || sample.OptimizationToggles != request.OptimizationToggles {
		http.Error(w, "artifact compiled strategy or optimization toggles do not match the requested cell", http.StatusConflict)
		return
	}
	attachMeasuredJourneyEvidence(&sample, request.Journey, status)
	sample.SmokeTestStatus = status.Status
	sample.SmokeTestError = status.Error
	if request.Condition == "cold" && sample.Platform == "linux" && (sample.PlatformDetail == nil || sample.PlatformDetail.LinuxDesktop == nil || sample.PlatformDetail.LinuxDesktop.ArtifactPageCacheResidencyPct > 5) {
		http.Error(w, "cold launch artifact page-cache residency exceeded 5% or was unavailable", http.StatusBadGateway)
		return
	}
	if len(status.ProcessLifetimeChecks) < 2 {
		http.Error(w, "measured launch lacks protocol and demo process-lifetime evidence", http.StatusBadGateway)
		return
	}
	stages := make(map[string]bool)
	for _, check := range status.ProcessLifetimeChecks {
		if !check.Passed || check.SessionID == "" || len(check.Remaining) > 0 {
			http.Error(w, "measured launch failed post-launch descendant assertion", http.StatusBadGateway)
			return
		}
		stages[check.Stage] = true
		sample.ProcessLifetime = append(sample.ProcessLifetime, PackagedProcessLifetimeCheck{
			Stage: check.Stage, SessionID: check.SessionID, Passed: check.Passed,
			Remaining: append([]int(nil), check.Remaining...), Assertion: check.Assertion,
		})
	}
	if !stages["protocol"] || !stages["demo"] {
		http.Error(w, "measured launch lacks protocol or demo process-lifetime evidence", http.StatusBadGateway)
		return
	}
	if err := sample.Validate(); err != nil {
		http.Error(w, "measured launch evidence is invalid: "+err.Error(), http.StatusBadGateway)
		return
	}
	encoded, err := json.MarshalIndent(sample, "", "  ")
	if err != nil {
		http.Error(w, "encode measured launch evidence: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(packagedLaunchPath(id, "demo"), encoded, 0o600); err != nil {
		http.Error(w, "retain measured process-lifetime evidence: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(sample)
}

func attachMeasuredJourneyEvidence(sample *PackagedLaunchSample, journeyID string, status *Status) {
	if sample == nil || status == nil || strings.TrimSpace(journeyID) == "" {
		return
	}
	sample.JourneyEvidence = &PackagedJourneyEvidence{
		JourneyID: journeyID, CaptureID: status.JourneyCaptureID,
		Disposition: status.JourneyDisposition, DegradedReason: status.JourneyDegradedReason,
	}
}

func copyMeasuredArtifact(source string) (destination, digest string, cleanup func(), err error) {
	input, err := os.Open(source)
	if err != nil {
		return "", "", nil, fmt.Errorf("open desktop artifact: %w", err)
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", "", nil, fmt.Errorf("desktop artifact must be a non-empty regular file")
	}
	directory, err := os.MkdirTemp("", "scenario-to-desktop-measured-install-")
	if err != nil {
		return "", "", nil, fmt.Errorf("create fresh install directory: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(directory) }
	destination = filepath.Join(directory, filepath.Base(source))
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm()|0o100)
	if err != nil {
		cleanup()
		return "", "", nil, fmt.Errorf("create measured artifact copy: %w", err)
	}
	hasher := sha256.New()
	if _, err = io.Copy(io.MultiWriter(output, hasher), input); err == nil {
		err = output.Sync()
	}
	closeErr := output.Close()
	if err != nil || closeErr != nil {
		cleanup()
		if err == nil {
			err = closeErr
		}
		return "", "", nil, fmt.Errorf("copy measured artifact: %w", err)
	}
	return destination, "sha256:" + hex.EncodeToString(hasher.Sum(nil)), cleanup, nil
}
