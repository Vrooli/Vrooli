package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/types/known/durationpb"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

const defaultScenarioSmokeTimeout = 12 * time.Minute

// cmdScenarioSmoke performs one bounded, live verification of the Tracking
// sandbox contract: completion, changed-files accounting, and persisted
// workspace-sandbox provenance. It deliberately uses a uniquely named scratch
// file and removes that file only after every assertion has passed.
func (a *App) cmdScenarioSmoke(args []string) error {
	fs := flag.NewFlagSet("scenario-smoke", flag.ContinueOnError)
	profileID := fs.String("profile-id", "", "Profile ID to use; defaults to an available code.cheap profile")
	projectRoot := fs.String("project-root", "", "Project root for the scratch task (defaults to the current directory)")
	workspaceSandboxURL := fs.String("workspace-sandbox-url", os.Getenv("WORKSPACE_SANDBOX_URL"), "Workspace Sandbox API URL")
	timeout := fs.Duration("timeout", defaultScenarioSmokeTimeout, "Maximum time to wait for terminal completion")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *timeout <= 0 {
		return fmt.Errorf("--timeout must be positive")
	}

	root, err := scenarioSmokeProjectRoot(*projectRoot)
	if err != nil {
		return err
	}
	if *workspaceSandboxURL == "" {
		*workspaceSandboxURL = "http://127.0.0.1:15427"
	}

	chosenProfileID, err := a.scenarioSmokeProfileID(*profileID)
	if err != nil {
		return err
	}
	scratchName := fmt.Sprintf(".agent-manager-scenario-smoke-%d.txt", time.Now().UTC().UnixNano())
	scratchPath := filepath.Join(root, scratchName)
	prompt := fmt.Sprintf("Create the file %q at the project root with exactly this one line: agent-manager tracking smoke. Do not modify any other file.", scratchName)
	task := &domainpb.Task{
		Title:       "Agent Manager tracking smoke",
		Description: prompt,
		ScopePath:   ".",
		ProjectRoot: root,
		CreatedBy:   "agent-manager scenario-smoke",
	}
	_, createdTask, err := a.services.Tasks.Create(task)
	if err != nil || createdTask == nil {
		return fmt.Errorf("create smoke task: %w", err)
	}

	mode := domainpb.RunMode_RUN_MODE_SANDBOXED
	autoApply := true
	request := &apipb.CreateRunRequest{
		TaskId:         createdTask.Id,
		AgentProfileId: protoString(chosenProfileID),
		RunMode:        &mode,
		Prompt:         protoString(prompt),
		InlineConfig: &domainpb.RunConfigOverrides{
			MaxTurns:          durationInt32(3),
			Timeout:           durationpb.New(*timeout),
			ClearAllowedTools: true,
			ClearDeniedTools:  true,
			SandboxConfig: &domainpb.SandboxConfig{
				Mode:      domainpb.SandboxMode_SANDBOX_MODE_TRACKING,
				AutoApply: &autoApply,
			},
		},
	}
	body, run, err := a.services.Runs.Create(request)
	if err != nil || run == nil {
		return fmt.Errorf("create smoke run: %w", apiError(body, err))
	}

	terminal, err := a.scenarioSmokeWait(run.Id, *timeout)
	if err != nil {
		return err
	}
	passed := true
	passed = scenarioSmokeAssert("run completed", terminal.Status == domainpb.RunStatus_RUN_STATUS_COMPLETE, "status="+formatEnumValue(terminal.Status, "RUN_STATUS_", "_")) && passed
	passed = scenarioSmokeAssert("changed files recorded", terminal.ChangedFiles > 0, fmt.Sprintf("changed_files=%d", terminal.ChangedFiles)) && passed
	provenance, err := scenarioSmokeHasAppliedProvenance(*workspaceSandboxURL, root, terminal.Id)
	passed = scenarioSmokeAssert("applied provenance recorded", err == nil && provenance, scenarioSmokeProvenanceDetail(err, provenance)) && passed
	if !passed {
		return fmt.Errorf("scenario smoke failed for run %s", terminal.Id)
	}
	if err := os.Remove(scratchPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove smoke scratch file %s: %w", scratchPath, err)
	}
	fmt.Printf("PASS scratch file cleaned: %s\n", scratchName)
	return nil
}

func durationInt32(value int32) *int32 { return &value }

func (a *App) scenarioSmokeProfileID(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}
	_, profiles, err := a.services.Profiles.List(0, 0)
	if err != nil {
		return "", fmt.Errorf("list profiles for smoke: %w", err)
	}
	for _, role := range []string{"code.cheap", "code.default", "code.smart"} {
		for _, profile := range profiles {
			if profile != nil && profile.RoleRef == role && scenarioSmokeProfileEligible(profile) {
				return profile.Id, nil
			}
		}
	}
	return "", fmt.Errorf("no unrestricted code profile is available; pass --profile-id")
}

func scenarioSmokeProfileEligible(profile *domainpb.AgentProfile) bool {
	if len(profile.AllowedTools) != 0 || len(profile.DeniedTools) != 0 {
		return false
	}
	return profile.GetSandboxConfig().GetMode() != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED
}

func scenarioSmokeProjectRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve current directory: %w", err)
		}
		value = cwd
	}
	root, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat project root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory: %s", root)
	}
	return root, nil
}

func (a *App) scenarioSmokeWait(runID string, timeout time.Duration) (*domainpb.Run, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, run, err := a.services.Runs.Get(runID)
		if err != nil {
			return nil, fmt.Errorf("get smoke run: %w", err)
		}
		if run != nil && isTerminalRunStatus(run.Status) {
			return run, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("smoke run %s did not reach a terminal state within %s", runID, timeout)
}

func isTerminalRunStatus(status domainpb.RunStatus) bool {
	return status == domainpb.RunStatus_RUN_STATUS_COMPLETE || status == domainpb.RunStatus_RUN_STATUS_FAILED || status == domainpb.RunStatus_RUN_STATUS_CANCELLED
}

func scenarioSmokeAssert(name string, ok bool, detail string) bool {
	result := "FAIL"
	if ok {
		result = "PASS"
	}
	fmt.Printf("%s %s: %s\n", result, name, detail)
	return ok
}

func scenarioSmokeHasAppliedProvenance(baseURL, projectRoot, runID string) (bool, error) {
	endpoint, err := url.Parse(strings.TrimRight(baseURL, "/") + "/api/v1/provenance/by-run")
	if err != nil {
		return false, err
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		return false, fmt.Errorf("workspace-sandbox URL must use http or https")
	}
	hostname := strings.ToLower(endpoint.Hostname())
	if hostname != "localhost" && hostname != "127.0.0.1" && hostname != "::1" {
		return false, fmt.Errorf("workspace-sandbox URL must target a local lifecycle endpoint, got %q", hostname)
	}
	port, err := strconv.Atoi(endpoint.Port())
	if err != nil || port < 1 || port > 65535 {
		return false, fmt.Errorf("workspace-sandbox URL must include a valid local TCP port")
	}
	// Rebuild the request endpoint from a fixed loopback host. This makes the
	// network boundary explicit: caller input can choose only a validated port,
	// never a destination host, scheme, or path.
	safeEndpoint := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), Path: "/api/v1/provenance/by-run"}
	query := safeEndpoint.Query()
	query.Set("projectRoot", projectRoot)
	safeEndpoint.RawQuery = query.Encode()
	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Get(safeEndpoint.String())
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("workspace-sandbox returned %s", response.Status)
	}
	var payload struct {
		RunGroups []struct {
			RunID string `json:"runId"`
			Files []struct {
				State string `json:"state"`
			} `json:"files"`
		} `json:"runGroups"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return false, err
	}
	for _, group := range payload.RunGroups {
		if group.RunID != runID {
			continue
		}
		for _, file := range group.Files {
			if file.State == "applied" {
				return true, nil
			}
		}
	}
	return false, nil
}

func scenarioSmokeProvenanceDetail(err error, found bool) string {
	if err != nil {
		return err.Error()
	}
	return fmt.Sprintf("applied=%t", found)
}
