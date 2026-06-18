package providercontract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	catalog "test-genie/internal/orchestrator/phases"
)

const usage = "usage: provider-contract check <phase|provider> <target-scenario> [--no-restart] [--json]"

var (
	commandRunner          = runCommand
	resolveProviderBaseURL = func(ctx context.Context, provider string) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, provider)
	}
)

type Args struct {
	Subject   string
	Target    string
	Restart   bool
	JSON      bool
	Timeout   time.Duration
	Operation string
}

type Probe struct {
	Phase       string   `json:"phase"`
	Provider    string   `json:"provider"`
	Invocation  []string `json:"invocation,omitempty"`
	Restartable bool     `json:"restartable"`
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
	assessmentMsg, _, err := probeAssessment(ctx, args, probe)
	if err != nil {
		return out, err
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

func probeAssessment(ctx context.Context, args Args, probe Probe) (*commonv1.MaturityAssessment, string, error) {
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
				assessmentMsg, parseErr := parseAssessment(raw)
				if parseErr != nil {
					return nil, source, fmt.Errorf("provider maturity contract violation from `%s`: %w", source, parseErr)
				}
				return assessmentMsg, source, nil
			}
			return nil, source, fmt.Errorf("run provider command `%s`: %w", source, err)
		}
		assessmentMsg, parseErr := parseAssessment(raw)
		if parseErr != nil {
			return nil, source, fmt.Errorf("provider maturity contract violation from `%s`: %w", source, parseErr)
		}
		return assessmentMsg, source, nil
	}
	return runSharedRPCProbe(ctx, args, probe)
}

func runSharedRPCProbe(ctx context.Context, args Args, probe Probe) (*commonv1.MaturityAssessment, string, error) {
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
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	source := probe.Provider + " " + scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	resp, err := client.ValidateScenario(runCtx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: args.Target,
	}))
	if runCtx.Err() != nil {
		return nil, source, runCtx.Err()
	}
	if err != nil {
		return nil, source, fmt.Errorf("provider RPC probe `%s`: %w", source, err)
	}
	if resp.Msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		return nil, source, fmt.Errorf("provider maturity contract violation from `%s`: validation status is required", source)
	}
	assessmentMsg := resp.Msg.GetAssessment()
	if assessmentMsg == nil {
		return nil, source, fmt.Errorf("provider maturity contract violation from `%s`: assessment is required", source)
	}
	if err := assessment.ValidateAssessment(assessmentMsg); err != nil {
		return nil, source, fmt.Errorf("provider maturity contract violation from `%s`: %w", source, err)
	}
	return assessmentMsg, source, nil
}

func ResolveProbe(subject string) (Probe, error) {
	key := catalog.NormalizeKey(subject)
	for _, probe := range Probes() {
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

var probeInvocationOverrides = map[catalog.Name][]string{
	catalog.Standards: {"scenario-auditor", "standards", "scan", "{{target}}", "--wait", "--json"},
}

func Probes() []Probe {
	specs := catalog.DefaultCatalog().All()
	probes := make([]Probe, 0, len(specs))
	for _, spec := range specs {
		if spec.Delegated == nil {
			continue
		}
		probe := Probe{
			Phase:       spec.Name.String(),
			Provider:    spec.Delegated.ProviderScenario,
			Restartable: true,
		}
		if invocation, ok := probeInvocationOverrides[spec.Name]; ok {
			probe.Invocation = append([]string(nil), invocation...)
		}
		probes = append(probes, probe)
	}
	return probes
}
