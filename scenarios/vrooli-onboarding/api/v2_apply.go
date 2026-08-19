package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/projectstate"
)

const (
	toolInstallTimeout    = 2 * time.Minute
	safeguardApplyTimeout = 2 * time.Minute
	resourceEnableTimeout = 3 * time.Minute
	// Starting a scenario may need to wait for a resource health transition, so
	// it deliberately has a longer bound than installation verbs.
	scenarioStartTimeout = 5 * time.Minute
)

type applyExecutor interface {
	InstallTool(context.Context, string) error
	ApplySafeguard(context.Context, string) error
	EnableResource(context.Context, string) error
	StartScenario(context.Context, string) error
}

type privilegedApplyExecutor interface {
	InstallToolPrivileged(context.Context, string) error
	ApplySafeguardPrivileged(context.Context, string) error
}

type needsElevationError struct{ Command string }

func (e *needsElevationError) Error() string {
	return fmt.Sprintf("needs_elevation: the setup-provisioned grant is unavailable; run `sudo vrooli setup`, then retry `%s`", e.Command)
}

type controlPlaneExecutor struct{}

var controlPlaneCommand = exec.CommandContext
var controlPlaneExecutable = exec.LookPath

func (e controlPlaneExecutor) run(ctx context.Context, args ...string) error {
	return e.runNamed(ctx, "vrooli", args...)
}

func (e controlPlaneExecutor) applyCapability(ctx context.Context, request operatorcapability.ActionRequest) (operatorcapability.Result, error) {
	output, runErr := e.runCapabilityJSON(ctx, "apply", request)
	var result operatorcapability.Result
	if decodeErr := json.Unmarshal(output, &result); decodeErr != nil {
		if runErr != nil {
			return operatorcapability.Result{}, runErr
		}
		return operatorcapability.Result{}, fmt.Errorf("decode capability result: %w", decodeErr)
	}
	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

func (e controlPlaneExecutor) previewCapability(ctx context.Context, request operatorcapability.ActionRequest) (operatorcapability.Preview, error) {
	output, runErr := e.runCapabilityJSON(ctx, "preview", request)
	var preview operatorcapability.Preview
	if decodeErr := json.Unmarshal(output, &preview); decodeErr != nil {
		if runErr != nil {
			return operatorcapability.Preview{}, runErr
		}
		return operatorcapability.Preview{}, fmt.Errorf("decode capability preview: %w", decodeErr)
	}
	if runErr != nil {
		return preview, runErr
	}
	return preview, nil
}

func (e controlPlaneExecutor) runCapabilityJSON(ctx context.Context, action string, request operatorcapability.ActionRequest) ([]byte, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encode capability action: %w", err)
	}
	return e.runNamedWithInput(ctx, []byte(append(payload, '\n')), "capability", action, "--json")
}

func (controlPlaneExecutor) runNamed(ctx context.Context, name string, args ...string) error {
	_, err := (controlPlaneExecutor{}).runNamedWithInput(ctx, nil, name, args...)
	return err
}

func (controlPlaneExecutor) runNamedWithInput(ctx context.Context, input []byte, commandName string, args ...string) ([]byte, error) {
	command := controlPlaneCommand(ctx, commandName, args...)
	if input != nil {
		command.Stdin = bytes.NewReader(input)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("control plane %s stdout pipe: %w", strings.Join(args, " "), err)
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("control plane %s stderr pipe: %w", strings.Join(args, " "), err)
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("control plane %s start: %w", strings.Join(args, " "), err)
	}
	var outputMu sync.Mutex
	var output []string
	readPipe := func(pipe io.ReadCloser) {
		defer pipe.Close()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			outputMu.Lock()
			output = append(output, scanner.Text())
			outputMu.Unlock()
		}
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); readPipe(stdout) }()
	go func() { defer wg.Done(); readPipe(stderr) }()
	waitErr := command.Wait()
	wg.Wait()
	if waitErr != nil {
		outputMu.Lock()
		joined := strings.TrimSpace(strings.Join(output, "\n"))
		outputMu.Unlock()
		return []byte(joined), fmt.Errorf("control plane %s failed: %w: %s", strings.Join(args, " "), waitErr, joined)
	}
	outputMu.Lock()
	joined := strings.TrimSpace(strings.Join(output, "\n"))
	outputMu.Unlock()
	return []byte(joined), nil
}

func (e controlPlaneExecutor) InstallTool(ctx context.Context, name string) error {
	return e.run(ctx, "host", "install", name, "--json", "--sudo-mode", "error")
}

func (e controlPlaneExecutor) ApplySafeguard(ctx context.Context, name string) error {
	return e.run(ctx, "host", "safeguard", name, "--json", "--sudo-mode", "error")
}

func (e controlPlaneExecutor) InstallToolPrivileged(ctx context.Context, name string) error {
	return e.runGranted(ctx, "host", "install", name, "--json", "--sudo-mode", "error")
}

func (e controlPlaneExecutor) ApplySafeguardPrivileged(ctx context.Context, name string) error {
	return e.runGranted(ctx, "host", "safeguard", name, "--json", "--sudo-mode", "error")
}

func (e controlPlaneExecutor) runGranted(ctx context.Context, args ...string) error {
	executable, err := controlPlaneExecutable("vrooli")
	if err != nil {
		return &needsElevationError{Command: "vrooli " + strings.Join(args, " ")}
	}
	audit := privilegedApplyAudit{Executable: executable, Args: append([]string(nil), args...)}
	defer func() { _ = audit.write() }()
	if err := e.runNamed(ctx, "sudo", append([]string{"-n", executable}, args...)...); err != nil {
		if isElevationDenied(err) {
			audit.Outcome = "needs_elevation"
			return &needsElevationError{Command: executable + " " + strings.Join(args, " ")}
		}
		audit.Outcome = "failed"
		return err
	}
	audit.Outcome = "applied"
	return nil
}

func isElevationDenied(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "a password is required") || strings.Contains(message, "not allowed to execute") || strings.Contains(message, "may not run sudo") || strings.Contains(message, "no tty present")
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
	Privileged   bool     `json:"privileged,omitempty"`
}

type applyItemResult struct {
	applyItem
	Outcome     string `json:"outcome"`
	Error       string `json:"error,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	BlockedBy   string `json:"blocked_by,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
}

type applyRun struct {
	ID              string            `json:"run_id"`
	Status          string            `json:"status"`
	SelectionDigest string            `json:"selection_digest"`
	StartedAt       string            `json:"started_at"`
	CompletedAt     string            `json:"completed_at,omitempty"`
	Error           string            `json:"error,omitempty"`
	Items           []applyItemResult `json:"items"`
}

var applyRuns = struct {
	sync.RWMutex
	items map[string]applyRun
}{items: map[string]applyRun{}}

var privilegedApplyAuditMu sync.Mutex

type privilegedApplyAudit struct {
	Executable string
	Args       []string
	Outcome    string
}

func (a privilegedApplyAudit) write() error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	path = filepath.Join(filepath.Dir(path), "audit", "onboarding-privileged.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	privilegedApplyAuditMu.Lock()
	defer privilegedApplyAuditMu.Unlock()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(struct {
		Executable string   `json:"executable"`
		Args       []string `json:"args"`
		Outcome    string   `json:"outcome"`
	}{a.Executable, a.Args, a.Outcome})
}

func selectionDigest(items []applyItem) string {
	h := sha256.New()
	for _, item := range items {
		_, _ = h.Write([]byte(item.ID + "|" + item.Kind + "|" + item.Name + "|" + fmt.Sprint(item.Required) + "|" + fmt.Sprint(item.Privileged) + "|" + strings.Join(item.Dependencies, ",") + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func applyRunPath(id string) (string, error) {
	statePath, err := operatorStatePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(statePath), "apply-runs", id+".json"), nil
}

func persistApplyRun(run applyRun) error {
	path, err := applyRunPath(run.ID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, append(data, '\n'), storage.SecretFilePerm)
}

func loadPersistedApplyRun(id string) (applyRun, error) {
	path, err := applyRunPath(id)
	if err != nil {
		return applyRun{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return applyRun{}, err
	}
	var run applyRun
	if err := json.Unmarshal(data, &run); err != nil {
		return applyRun{}, fmt.Errorf("decode apply run %s: %w", id, err)
	}
	return run, nil
}

func storeApplyRun(run applyRun) {
	applyRuns.Lock()
	applyRuns.items[run.ID] = run
	applyRuns.Unlock()
	_ = persistApplyRun(run)
}

func updateApplyRun(run applyRun) {
	applyRuns.Lock()
	applyRuns.items[run.ID] = run
	applyRuns.Unlock()
	_ = persistApplyRun(run)
}

func applyRunSnapshot(id string) (applyRun, bool) {
	applyRuns.RLock()
	run, ok := applyRuns.items[id]
	applyRuns.RUnlock()
	if ok {
		return run, true
	}
	run, err := loadPersistedApplyRun(id)
	return run, err == nil
}

func (s *Server) buildApplyRun(ctx context.Context) (applyRun, error) {
	root, err := manifestRoot()
	if err != nil {
		return applyRun{}, err
	}
	models, err := loadScenarioReadModels()
	if err != nil {
		return applyRun{}, err
	}
	state, err := loadOperatorStateFor(ctx)
	if err != nil {
		return applyRun{}, err
	}
	closure, err := resolveClosureForState(root, models, state)
	if err != nil {
		return applyRun{}, err
	}
	requirements, err := deriveV2HostRequirements(root, state, models)
	if err != nil {
		return applyRun{}, err
	}
	items := buildApplyPlan(applyPlanInput{Closure: closure, Requirements: requirements, State: state})
	now := operatorStateNow()
	run := applyRun{ID: fmt.Sprintf("apply-%d", now.UnixNano()), Status: "pending", SelectionDigest: selectionDigest(items), StartedAt: now.UTC().Format(time.RFC3339), Items: make([]applyItemResult, 0, len(items))}
	for _, item := range items {
		run.Items = append(run.Items, applyItemResult{applyItem: item, Outcome: "pending"})
	}
	if state.Completion != nil && state.Completion.SelectionDigest == run.SelectionDigest {
		run.Status = "already_satisfied"
		for i := range run.Items {
			run.Items[i].Outcome = "already_satisfied"
		}
		run.CompletedAt = operatorStateNow().UTC().Format(time.RFC3339)
	}
	return run, nil
}

func executeApplyItem(ctx context.Context, item applyItem) error {
	var timeout time.Duration
	switch item.Kind {
	case "tool":
		timeout = toolInstallTimeout
	case "safeguard":
		timeout = safeguardApplyTimeout
	case "resource":
		timeout = resourceEnableTimeout
	case "scenario":
		timeout = scenarioStartTimeout
	default:
		return fmt.Errorf("unsupported apply item kind %q", item.Kind)
	}
	itemCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var err error
	switch item.Kind {
	case "tool":
		if item.Privileged {
			if privileged, ok := onboardingApplyExecutor.(privilegedApplyExecutor); ok {
				err = privileged.InstallToolPrivileged(itemCtx, item.Name)
			} else {
				err = onboardingApplyExecutor.InstallTool(itemCtx, item.Name)
			}
		} else {
			err = onboardingApplyExecutor.InstallTool(itemCtx, item.Name)
		}
	case "safeguard":
		if item.Privileged {
			if privileged, ok := onboardingApplyExecutor.(privilegedApplyExecutor); ok {
				err = privileged.ApplySafeguardPrivileged(itemCtx, item.Name)
			} else {
				err = onboardingApplyExecutor.ApplySafeguard(itemCtx, item.Name)
			}
		} else {
			err = onboardingApplyExecutor.ApplySafeguard(itemCtx, item.Name)
		}
	case "resource":
		err = onboardingApplyExecutor.EnableResource(itemCtx, item.Name)
	case "scenario":
		err = onboardingApplyExecutor.StartScenario(itemCtx, item.Name)
	}
	if errors.Is(itemCtx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out after %s", item.Kind, timeout)
	}
	return err
}

func executeApplyRun(ctx context.Context, run applyRun) {
	if run.Status == "already_satisfied" {
		updateApplyRun(run)
		return
	}
	run.Status = "applying"
	updateApplyRun(run)
	failed := map[string]error{}
	for i := range run.Items {
		item := run.Items[i].applyItem
		for _, dependency := range item.Dependencies {
			if dependencyErr, ok := failed[dependency]; ok {
				run.Items[i].Outcome = "blocked"
				run.Items[i].BlockedBy = dependency
				run.Items[i].Error = dependencyErr.Error()
				run.Items[i].Remediation = "resolve the blocking dependency, then re-run setup"
				failed[item.ID] = fmt.Errorf("blocked by %s", dependency)
				updateApplyRun(run)
				continue
			}
		}
		if run.Items[i].Outcome == "blocked" {
			continue
		}
		if err := executeApplyItem(ctx, item); err != nil {
			run.Items[i].Outcome = "timed_out"
			if !strings.Contains(err.Error(), "timed out") {
				run.Items[i].Outcome = "failed"
			}
			var elevationErr *needsElevationError
			if errors.As(err, &elevationErr) {
				run.Items[i].Outcome = "needs_elevation"
				run.Items[i].ErrorCode = "needs_elevation"
				run.Items[i].Remediation = "run `sudo vrooli setup` to provision the exact grant, then re-run onboarding apply"
			}
			run.Items[i].Error = err.Error()
			if run.Items[i].Remediation == "" {
				run.Items[i].Remediation = "inspect the control-plane error, correct the host, then re-run setup"
			}
			failed[item.ID] = err
		} else {
			run.Items[i].Outcome = "applied"
		}
		updateApplyRun(run)
	}
	if len(failed) > 0 {
		run.Status = "partially_applied"
	} else {
		run.Status = "applied"
		if _, err := operatorStateService().MarkApplied(ctx, run.SelectionDigest, operatorStateNow()); err != nil {
			run.Status = "partially_applied"
			run.Error = err.Error()
		} else if err := markConfigurationComplete(run.SelectionDigest); err != nil {
			run.Status = "partially_applied"
			run.Error = err.Error()
		}
	}
	run.CompletedAt = operatorStateNow().UTC().Format(time.RFC3339)
	updateApplyRun(run)
}

func markConfigurationComplete(selectionDigest string) error {
	root, err := manifestRoot()
	if err != nil {
		return err
	}
	home, err := config.HomeDir()
	if err != nil {
		return err
	}
	return projectstate.MarkConfigurationComplete(home, root, selectionDigest)
}

func (s *Server) handleV2Apply(w http.ResponseWriter, r *http.Request) {
	run, err := s.buildApplyRun(r.Context())
	if err != nil {
		if writeCatalogDegraded(w, err) {
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	storeApplyRun(run)
	if run.Status == "pending" {
		// The HTTP request ends as soon as the durable run is accepted. Preserve
		// request-scoped values for the worker while intentionally removing the
		// response cancellation that would otherwise abort a valid apply.
		go executeApplyRun(context.WithoutCancel(r.Context()), run)
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleV2ApplyStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/v2/apply/"))
	run, ok := applyRunSnapshot(id)
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
