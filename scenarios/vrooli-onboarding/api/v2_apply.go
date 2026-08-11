package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type applyExecutor interface {
	InstallTool(context.Context, string) error
	ApplySafeguard(context.Context, string) error
	EnableResource(context.Context, string) error
	StartScenario(context.Context, string) error
}

type controlPlaneExecutor struct{}

var controlPlaneCommand = exec.CommandContext

func (controlPlaneExecutor) run(ctx context.Context, args ...string) error {
	command := controlPlaneCommand(ctx, "vrooli", args...)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("control plane %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
func (e controlPlaneExecutor) InstallTool(ctx context.Context, name string) error {
	return e.run(ctx, "host", "install", name, "--json")
}
func (e controlPlaneExecutor) ApplySafeguard(ctx context.Context, name string) error {
	return e.run(ctx, "host", "safeguard", name, "--json")
}
func (e controlPlaneExecutor) EnableResource(ctx context.Context, name string) error {
	return e.run(ctx, "resource", "enable", name, "--json")
}
func (e controlPlaneExecutor) StartScenario(ctx context.Context, name string) error {
	return e.run(ctx, "scenario", "start", name, "--json")
}

var onboardingApplyExecutor applyExecutor = controlPlaneExecutor{}

type applyItem struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Dependencies []string `json:"dependencies,omitempty"`
	Required     bool     `json:"required"`
}

type applyItemResult struct {
	applyItem
	Outcome     string `json:"outcome"`
	Error       string `json:"error,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	BlockedBy   string `json:"blocked_by,omitempty"`
}

type applyRun struct {
	ID              string            `json:"run_id"`
	Status          string            `json:"status"`
	SelectionDigest string            `json:"selection_digest"`
	StartedAt       string            `json:"started_at"`
	CompletedAt     string            `json:"completed_at,omitempty"`
	Items           []applyItemResult `json:"items"`
}

var applyRuns = struct {
	sync.RWMutex
	items map[string]applyRun
}{items: map[string]applyRun{}}

func selectionDigest(closure closureResult) string {
	h := sha256.New()
	for _, member := range closure.Scenarios {
		_, _ = h.Write([]byte("scenario:" + member.Name + "\n"))
	}
	for _, member := range closure.Resources {
		_, _ = h.Write([]byte("resource:" + member.Name + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildApplyItems(closure closureResult, requirements hostRequirementsResponse) []applyItem {
	items := make([]applyItem, 0, len(requirements.Tools)+len(requirements.Safeguards)+len(closure.Resources)+len(closure.Scenarios))
	for _, item := range requirements.Tools {
		items = append(items, applyItem{ID: "tool:" + item.Name, Kind: "tool", Name: item.Name, Required: item.Required})
	}
	for _, item := range requirements.Safeguards {
		items = append(items, applyItem{ID: "safeguard:" + item.Name, Kind: "safeguard", Name: item.Name, Required: item.Required})
	}
	for _, member := range closure.Resources {
		items = append(items, applyItem{ID: "resource:" + member.Name, Kind: "resource", Name: member.Name, Required: member.Required})
	}
	for _, member := range closure.Scenarios {
		dependencies := make([]string, 0)
		for _, resource := range closure.Resources {
			for _, provenance := range resource.Provenance {
				if provenance.From == member.Name {
					dependencies = append(dependencies, "resource:"+resource.Name)
					break
				}
			}
		}
		items = append(items, applyItem{ID: "scenario:" + member.Name, Kind: "scenario", Name: member.Name, Dependencies: dependencies, Required: member.Required || member.Direct})
	}
	return items
}

func (s *Server) createApplyRun(ctx context.Context) (applyRun, error) {
	root, err := manifestRoot()
	if err != nil {
		return applyRun{}, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return applyRun{}, err
	}
	closure, err := resolveClosure(root, models)
	if err != nil {
		return applyRun{}, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return applyRun{}, err
	}
	requirements, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		return applyRun{}, err
	}
	now := operatorStateNow()
	run := applyRun{ID: fmt.Sprintf("apply-%d", now.UnixNano()), Status: "applying", SelectionDigest: selectionDigest(closure), StartedAt: now.UTC().Format(time.RFC3339), Items: make([]applyItemResult, 0)}
	items := buildApplyItems(closure, requirements)
	if state.Completion != nil && state.Completion.SelectionDigest == run.SelectionDigest {
		run.Status = "already_satisfied"
		for _, item := range items {
			run.Items = append(run.Items, applyItemResult{applyItem: item, Outcome: "already_satisfied"})
		}
		run.CompletedAt = operatorStateNow().UTC().Format(time.RFC3339)
		return run, nil
	}
	failed := map[string]error{}
	for _, item := range items {
		result := applyItemResult{applyItem: item}
		for _, dependency := range item.Dependencies {
			if dependencyErr, ok := failed[dependency]; ok {
				result.Outcome = "blocked"
				result.BlockedBy = dependency
				result.Error = dependencyErr.Error()
				result.Remediation = "resolve the blocking dependency, then re-run setup"
				failed[item.ID] = fmt.Errorf("blocked by %s", dependency)
				break
			}
		}
		if result.Outcome == "blocked" {
			run.Items = append(run.Items, result)
			continue
		}
		var actionErr error
		switch item.Kind {
		case "tool":
			actionErr = onboardingApplyExecutor.InstallTool(ctx, item.Name)
		case "safeguard":
			actionErr = onboardingApplyExecutor.ApplySafeguard(ctx, item.Name)
		case "resource":
			actionErr = onboardingApplyExecutor.EnableResource(ctx, item.Name)
		case "scenario":
			actionErr = onboardingApplyExecutor.StartScenario(ctx, item.Name)
		}
		if actionErr != nil {
			result.Outcome = "failed"
			result.Error = actionErr.Error()
			result.Remediation = "inspect the control-plane error, correct the host, then re-run setup"
			failed[item.ID] = actionErr
		} else {
			result.Outcome = "applied"
		}
		run.Items = append(run.Items, result)
	}
	if len(failed) > 0 {
		run.Status = "partially_applied"
	} else {
		run.Status = "applied"
		if _, err := operatorStateService().MarkApplied(ctx, run.SelectionDigest, operatorStateNow()); err != nil {
			return applyRun{}, err
		}
	}
	run.CompletedAt = operatorStateNow().UTC().Format(time.RFC3339)
	return run, nil
}

func (s *Server) handleV2Apply(w http.ResponseWriter, r *http.Request) {
	run, err := s.createApplyRun(r.Context())
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	applyRuns.Lock()
	applyRuns.items[run.ID] = run
	applyRuns.Unlock()
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleV2ApplyStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v2/apply/"))
	applyRuns.RLock()
	run, ok := applyRuns.items[id]
	applyRuns.RUnlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "apply run not found"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (r applyRun) MarshalJSON() ([]byte, error) {
	type plain applyRun
	return json.Marshal(plain(r))
}
