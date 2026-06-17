package providercontract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

const usage = "usage: provider-contract check <phase|provider> <target-scenario> [--no-restart] [--json]"

var (
	commandRunner          = runCommand
	resolveProviderBaseURL = func(ctx context.Context, provider string) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, provider)
	}
)
var httpClient = &http.Client{Timeout: 2 * time.Minute}

type Args struct {
	Subject   string
	Target    string
	Restart   bool
	JSON      bool
	Timeout   time.Duration
	Operation string
}

type Probe struct {
	Phase       string     `json:"phase"`
	Provider    string     `json:"provider"`
	Invocation  []string   `json:"invocation"`
	HTTP        *HTTPProbe `json:"http,omitempty"`
	Restartable bool       `json:"restartable"`
}

type HTTPProbe struct {
	Method string         `json:"method"`
	Path   string         `json:"path"`
	Body   map[string]any `json:"body,omitempty"`
}

type Result struct {
	Phase      string `json:"phase"`
	Provider   string `json:"provider"`
	Target     string `json:"target"`
	Restarted  bool   `json:"restarted"`
	Status     string `json:"status"`
	Assessment struct {
		Scenario     string `json:"scenario"`
		Provider     string `json:"provider"`
		Phase        string `json:"phase"`
		Version      string `json:"version"`
		CurrentLevel string `json:"currentLevel"`
		NextLevel    string `json:"nextLevel,omitempty"`
	} `json:"assessment"`
}

func Run(args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}
	probe, err := ResolveProbe(parsed.Subject)
	if err != nil {
		return err
	}
	result, err := Check(context.Background(), parsed, probe)
	if err != nil {
		return err
	}
	if parsed.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}
	fmt.Printf("Provider contract ok: %s -> %s\n", result.Phase, result.Provider)
	if result.Restarted {
		fmt.Printf("  Restarted : %s\n", result.Provider)
	}
	fmt.Printf("  Target    : %s\n", result.Target)
	fmt.Printf("  Version   : %s\n", result.Assessment.Version)
	fmt.Printf("  Local     : %s", result.Assessment.CurrentLevel)
	if result.Assessment.NextLevel != "" {
		fmt.Printf(" -> %s", result.Assessment.NextLevel)
	}
	fmt.Println()
	return nil
}

func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 || args[0] != "check" {
		return Args{}, errors.New(usage)
	}
	fs := flag.NewFlagSet("provider-contract check", flag.ContinueOnError)
	fs.SetOutput(flag.CommandLine.Output())
	out := Args{Restart: true, Timeout: 2 * time.Minute, Operation: args[0]}
	fs.BoolVar(&out.JSON, "json", false, "Output JSON")
	fs.BoolVar(&out.Restart, "restart", true, "Restart provider scenario through Vrooli lifecycle before probing")
	noRestart := fs.Bool("no-restart", false, "Skip provider lifecycle restart")
	timeout := fs.Duration("timeout", out.Timeout, "Provider command timeout")
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return Args{}, err
	}
	if *noRestart {
		out.Restart = false
	}
	out.Timeout = *timeout
	rest := fs.Args()
	if len(rest) != 2 {
		return Args{}, errors.New(usage)
	}
	out.Subject = strings.TrimSpace(rest[0])
	out.Target = strings.TrimSpace(rest[1])
	if out.Subject == "" || out.Target == "" {
		return Args{}, errors.New(usage)
	}
	return out, nil
}

func Check(ctx context.Context, args Args, probe Probe) (Result, error) {
	var out Result
	out.Phase = probe.Phase
	out.Provider = probe.Provider
	out.Target = args.Target
	if !probe.Restartable && args.Restart {
		return out, fmt.Errorf("%s uses %s; add a provider-specific RPC freshness probe before hard contract validation", probe.Phase, probe.Provider)
	}
	if args.Restart {
		if _, err := commandRunner(ctx, args.Timeout, "", "vrooli", "scenario", "restart", probe.Provider); err != nil {
			return out, fmt.Errorf("restart provider %s via lifecycle: %w", probe.Provider, err)
		}
		out.Restarted = true
	}
	raw, source, err := runProbe(ctx, args, probe)
	if err != nil {
		return out, err
	}
	assessmentMsg, err := parseAssessment(raw)
	if err != nil {
		return out, fmt.Errorf("provider maturity contract violation from `%s`: %w", source, err)
	}
	if got, want := strings.TrimSpace(assessmentMsg.GetProvider()), probe.Provider; got != "" && got != want {
		return out, fmt.Errorf("provider maturity contract violation: assessment.provider=%q, want %q", got, want)
	}
	if got, want := strings.TrimSpace(assessmentMsg.GetPhase()), probe.Phase; got != "" && got != want {
		return out, fmt.Errorf("provider maturity contract violation: assessment.phase=%q, want %q", got, want)
	}
	out.Status = "ok"
	out.Assessment.Scenario = assessmentMsg.GetScenario()
	out.Assessment.Provider = assessmentMsg.GetProvider()
	out.Assessment.Phase = assessmentMsg.GetPhase()
	out.Assessment.Version = assessmentMsg.GetVersion()
	out.Assessment.CurrentLevel = assessmentMsg.GetLocal().GetCurrentLevel()
	out.Assessment.NextLevel = assessmentMsg.GetLocal().GetNextLevel()
	return out, nil
}

func runProbe(ctx context.Context, args Args, probe Probe) ([]byte, string, error) {
	invocation := append([]string(nil), probe.Invocation...)
	for i, part := range invocation {
		if part == "{{target}}" {
			invocation[i] = args.Target
		}
	}
	if len(invocation) > 0 {
		raw, err := commandRunner(ctx, args.Timeout, "", invocation[0], invocation[1:]...)
		source := strings.Join(invocation, " ")
		if err != nil {
			if len(bytes.TrimSpace(raw)) > 0 {
				return raw, source, nil
			}
			return nil, source, fmt.Errorf("run provider command `%s`: %w", source, err)
		}
		return raw, source, nil
	}
	if probe.HTTP != nil {
		raw, source, err := runHTTPProbe(ctx, args, probe)
		if err != nil {
			return nil, source, err
		}
		return raw, source, nil
	}
	return nil, "", fmt.Errorf("%s has no provider contract probe configured", probe.Phase)
}

func runHTTPProbe(ctx context.Context, args Args, probe Probe) ([]byte, string, error) {
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	baseURL, err := resolveProviderBaseURL(ctx, probe.Provider)
	if err != nil {
		return nil, probe.Provider, fmt.Errorf("resolve provider %s URL: %w", probe.Provider, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, probe.Provider, fmt.Errorf("provider %s base URL is empty", probe.Provider)
	}
	body := replaceProbeBodyTarget(probe.HTTP.Body, args.Target)
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, baseURL + probe.HTTP.Path, fmt.Errorf("encode HTTP probe body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	method := strings.TrimSpace(probe.HTTP.Method)
	if method == "" {
		method = http.MethodGet
	}
	source := method + " " + baseURL + probe.HTTP.Path
	req, err := http.NewRequestWithContext(runCtx, method, baseURL+probe.HTTP.Path, reader)
	if err != nil {
		return nil, source, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := httpClient.Do(req)
	if runCtx.Err() != nil {
		return nil, source, runCtx.Err()
	}
	if err != nil {
		return nil, source, fmt.Errorf("run provider HTTP probe `%s`: %w", source, err)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, source, fmt.Errorf("read provider HTTP probe response: %w", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, source, fmt.Errorf("provider HTTP probe `%s` returned %d: %s", source, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, source, nil
}

func replaceProbeBodyTarget(body map[string]any, target string) map[string]any {
	if body == nil {
		return nil
	}
	out := make(map[string]any, len(body))
	for k, v := range body {
		if s, ok := v.(string); ok && s == "{{target}}" {
			out[k] = target
			continue
		}
		out[k] = v
	}
	return out
}

func ResolveProbe(subject string) (Probe, error) {
	key := strings.ToLower(strings.TrimSpace(subject))
	for _, probe := range probes {
		if key == probe.Phase || key == probe.Provider {
			return probe, nil
		}
	}
	return Probe{}, fmt.Errorf("unknown provider-backed phase or provider %q", subject)
}

func parseAssessment(raw []byte) (*commonv1.MaturityAssessment, error) {
	var payload json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse provider JSON: %w", err)
	}
	assessmentJSON := findAssessmentJSON(payload)
	if len(assessmentJSON) == 0 || bytes.Equal(assessmentJSON, []byte("null")) {
		return nil, fmt.Errorf("assessment is required")
	}
	var msg commonv1.MaturityAssessment
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(assessmentJSON, &msg); err != nil {
		return nil, fmt.Errorf("parse assessment: %w", err)
	}
	if err := assessment.ValidateAssessment(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func findAssessmentJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err == nil {
		if assessmentJSON, ok := obj["assessment"]; ok {
			return assessmentJSON
		}
		for _, child := range obj {
			if assessmentJSON := findAssessmentJSON(child); len(assessmentJSON) > 0 {
				return assessmentJSON
			}
		}
		return nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	for _, child := range arr {
		if assessmentJSON := findAssessmentJSON(child); len(assessmentJSON) > 0 {
			return assessmentJSON
		}
	}
	return nil
}

func runCommand(ctx context.Context, timeout time.Duration, dir string, name string, args ...string) ([]byte, error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if runCtx.Err() != nil {
		return out, runCtx.Err()
	}
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return out, fmt.Errorf("%w: %s", err, msg)
		}
		return out, err
	}
	return out, nil
}

var probes = []Probe{
	{Phase: "contracts", Provider: "cli-health", Invocation: []string{"cli-health", "validate", "scenario", "{{target}}", "--json"}, Restartable: true},
	{Phase: "ui-health", Provider: "ui-health", Invocation: []string{"ui-health", "validate", "scenario", "{{target}}", "--json"}, Restartable: true},
	{Phase: "quality", Provider: "quality-health", Invocation: []string{"quality-health", "audit", "run", "{{target}}", "--json"}, Restartable: true},
	{Phase: "dependencies", Provider: "scenario-dependency-analyzer", Invocation: []string{"scenario-dependency-analyzer", "health", "{{target}}", "--json"}, Restartable: true},
	{Phase: "security", Provider: "security-health", Invocation: []string{"security-health", "validate", "scenario", "{{target}}", "--json"}, Restartable: true},
	{Phase: "measures", Provider: "measures-health", Invocation: []string{"measures-health", "validate", "scenario", "{{target}}", "--json"}, Restartable: true},
	{Phase: "proto", Provider: "proto-health", Invocation: []string{"proto-health", "validate", "scenario", "{{target}}", "--json"}, Restartable: true},
	{Phase: "unit", Provider: "unit-health", Invocation: []string{"unit-health", "validate", "scenario", "{{target}}", "--execution", "--json"}, Restartable: true},
	{Phase: "standards", Provider: "scenario-auditor", Invocation: []string{"scenario-auditor", "standards", "scan", "{{target}}", "--wait", "--json"}, Restartable: true},
	{Phase: "architecture", Provider: "architecture-cartographer", HTTP: &HTTPProbe{Method: http.MethodPost, Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Body: map[string]any{"scenario": "{{target}}"}}, Restartable: true},
	{Phase: "docs", Provider: "knowledge-observatory", HTTP: &HTTPProbe{Method: http.MethodPost, Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Body: map[string]any{"scenario": "{{target}}"}}, Restartable: true},
	{Phase: "tidiness", Provider: "tidiness-manager", HTTP: &HTTPProbe{Method: http.MethodPost, Path: scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure, Body: map[string]any{"scenario": "{{target}}"}}, Restartable: true},
}
