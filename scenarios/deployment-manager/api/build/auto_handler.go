package build

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// AutoBuildRequest builds binaries by scanning scenario folders.
type AutoBuildRequest struct {
	Scenario  string   `json:"scenario"`
	Platforms []string `json:"platforms,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	DryRun    bool     `json:"dry_run,omitempty"`
}

type buildTarget struct {
	Folder string
	Config BuildConfig
	ID     string
}

// AutoBuild handles POST /api/v1/build/auto requests.
func (h *Handler) AutoBuild(w http.ResponseWriter, r *http.Request) {
	var req AutoBuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}

	scenarioDir := filepath.Join(h.vrooli, "scenarios", req.Scenario)
	if _, err := os.Stat(scenarioDir); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"scenario not found: %v"}`, err), http.StatusBadRequest)
		return
	}

	targets := detectGoTargets(scenarioDir, req.Scenario, req.Targets)
	buildID := generateAutoBuildID()
	statusURL := buildAutoStatusURL(r, buildID)
	targetStatuses := buildAutoTargetPlan(scenarioDir, targets, req.Platforms)

	job := &AutoBuildStatus{
		BuildID:      buildID,
		Scenario:     req.Scenario,
		Status:       "queued",
		Message:      "Auto build queued",
		CreatedAt:    time.Now(),
		Targets:      targetStatuses,
		StatusURL:    statusURL,
		CheckCommand: buildAutoCheckCommand(statusURL),
		PollAfterMs:  2000,
	}

	if len(targets) == 0 {
		job.Status = "skipped"
		job.Message = "No Go build targets detected"
		h.autoBuilds.Save(job)
		h.writeJSON(w, http.StatusOK, job)
		return
	}

	h.autoBuilds.Save(job)

	if req.DryRun {
		h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
			status.Status = "dry_run"
			status.Message = fmt.Sprintf("Would build %d target(s)", len(targets))
		})
		if status, ok := h.autoBuilds.Get(buildID); ok {
			h.writeJSON(w, http.StatusOK, status)
			return
		}
		h.writeJSON(w, http.StatusOK, job)
		return
	}

	go h.runAutoBuild(buildID, scenarioDir, targets, req.Platforms)
	h.writeJSON(w, http.StatusAccepted, job)
}

// AutoBuildStatus handles GET /api/v1/build/auto/{build_id}.
func (h *Handler) AutoBuildStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	buildID := vars["build_id"]
	if buildID == "" {
		http.Error(w, `{"error":"build_id is required"}`, http.StatusBadRequest)
		return
	}

	status, ok := h.autoBuilds.Get(buildID)
	if !ok {
		http.Error(w, `{"error":"build not found"}`, http.StatusNotFound)
		return
	}

	h.writeJSON(w, http.StatusOK, status)
}

func (h *Handler) runAutoBuild(buildID, scenarioDir string, targets []buildTarget, platforms []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	startedAt := time.Now()
	h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
		status.Status = "building"
		status.Message = "Auto build started"
		status.StartedAt = &startedAt
		status.BuildLog = append(status.BuildLog, fmt.Sprintf("Starting auto build (%d target(s))", len(targets)))
	})

	builder := NewBuilder(scenarioDir, h.log)
	allSucceeded := true
	anySucceeded := false

	targetPlatforms := platforms
	if len(targetPlatforms) == 0 {
		for _, p := range SupportedPlatforms {
			targetPlatforms = append(targetPlatforms, p.Name)
		}
	}

	for _, target := range targets {
		h.appendAutoBuildLog(buildID, fmt.Sprintf("Target %s (%s)", target.ID, target.Folder))
		for _, platform := range filterPlatforms(targetPlatforms) {
			started := time.Now()
			h.updateAutoBuildPlatform(buildID, target.ID, platform.Name, func(p *AutoBuildPlatformStatus) {
				p.Status = "building"
				p.StartedAt = &started
			})
			h.appendAutoBuildLog(buildID, fmt.Sprintf("Building %s for %s", target.ID, platform.Name))

			result := builder.buildForPlatform(ctx, target.ID, &target.Config, platform)
			completed := time.Now()
			if result.Success {
				anySucceeded = true
			} else {
				allSucceeded = false
			}

			h.updateAutoBuildPlatform(buildID, target.ID, platform.Name, func(p *AutoBuildPlatformStatus) {
				if result.Success {
					p.Status = "success"
				} else {
					p.Status = "failed"
					p.Error = result.Error
				}
				p.OutputPath = result.OutputPath
				p.CompletedAt = &completed
			})

			if result.Success {
				h.appendAutoBuildLog(buildID, fmt.Sprintf("Built %s (%s)", target.ID, platform.Name))
			} else {
				h.appendAutoBuildError(buildID, fmt.Sprintf("Failed %s (%s): %s", target.ID, platform.Name, result.Error))
			}
		}
	}

	completedAt := time.Now()
	finalStatus := "success"
	if !allSucceeded && anySucceeded {
		finalStatus = "partial"
	}
	if !anySucceeded {
		finalStatus = "failed"
	}

	h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
		status.Status = finalStatus
		status.CompletedAt = &completedAt
		if finalStatus == "success" {
			status.Message = "Auto build complete"
		} else {
			status.Message = "Auto build finished with errors"
		}
	})
}

func buildAutoTargetPlan(scenarioDir string, targets []buildTarget, platforms []string) []AutoBuildTargetStatus {
	platformList := platforms
	if len(platformList) == 0 {
		for _, p := range SupportedPlatforms {
			platformList = append(platformList, p.Name)
		}
	}

	plan := make([]AutoBuildTargetStatus, 0, len(targets))
	for _, target := range targets {
		targetStatus := AutoBuildTargetStatus{
			ID:     target.ID,
			Folder: target.Folder,
		}
		for _, platform := range filterPlatforms(platformList) {
			outputPath := ResolveOutputPath(scenarioDir, &target.Config, platform)
			targetStatus.Platforms = append(targetStatus.Platforms, AutoBuildPlatformStatus{
				Name:       platform.Name,
				Status:     "pending",
				OutputPath: outputPath,
			})
		}
		plan = append(plan, targetStatus)
	}
	return plan
}

func (h *Handler) appendAutoBuildLog(buildID, message string) {
	h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
		status.BuildLog = append(status.BuildLog, message)
	})
}

func (h *Handler) appendAutoBuildError(buildID, message string) {
	h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
		status.ErrorLog = append(status.ErrorLog, message)
	})
}

func (h *Handler) updateAutoBuildPlatform(buildID, targetID, platformName string, fn func(p *AutoBuildPlatformStatus)) {
	h.autoBuilds.Update(buildID, func(status *AutoBuildStatus) {
		for ti := range status.Targets {
			if status.Targets[ti].ID != targetID {
				continue
			}
			for pi := range status.Targets[ti].Platforms {
				if status.Targets[ti].Platforms[pi].Name == platformName {
					fn(&status.Targets[ti].Platforms[pi])
					return
				}
			}
		}
	})
}

func buildAutoStatusURL(r *http.Request, buildID string) string {
	if r == nil {
		return ""
	}
	host := r.Host
	if host == "" {
		return fmt.Sprintf("/api/v1/build/auto/%s", buildID)
	}
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return fmt.Sprintf("%s://%s/api/v1/build/auto/%s", scheme, host, buildID)
}

func buildAutoCheckCommand(statusURL string) string {
	if statusURL == "" {
		return ""
	}
	return fmt.Sprintf("curl -s %s", statusURL)
}

func generateAutoBuildID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err == nil {
		return hex.EncodeToString(buf)
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func detectGoTargets(scenarioDir, scenarioName string, requested []string) []buildTarget {
	requestedSet := map[string]bool{}
	for _, name := range requested {
		name = strings.TrimSpace(name)
		if name != "" {
			requestedSet[name] = true
		}
	}

	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		return nil
	}

	var targets []buildTarget
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		folder := entry.Name()
		if len(requestedSet) > 0 && !requestedSet[folder] {
			continue
		}
		if !isGoModule(filepath.Join(scenarioDir, folder)) {
			continue
		}
		baseName := binaryBaseName(scenarioName, folder)
		targets = append(targets, buildTarget{
			Folder: folder,
			ID:     serviceIDForTarget(scenarioName, folder),
			Config: BuildConfig{
				Type:          "go",
				SourceDir:     folder,
				EntryPoint:    ".",
				OutputPattern: filepath.ToSlash(filepath.Join("bin", folder, "{{platform}}", baseName+"{{ext}}")),
				Env: map[string]string{
					"CGO_ENABLED": "0",
				},
			},
		})
	}
	return targets
}

func isGoModule(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	return len(matches) > 0
}

func serviceIDForTarget(scenarioName, folder string) string {
	return fmt.Sprintf("%s-%s", scenarioName, folder)
}

func binaryBaseName(scenarioName, folder string) string {
	switch folder {
	case "api":
		return fmt.Sprintf("%s-api", scenarioName)
	case "cli":
		return scenarioName
	default:
		return fmt.Sprintf("%s-%s", scenarioName, folder)
	}
}
