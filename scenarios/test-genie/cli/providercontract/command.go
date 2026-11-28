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
	catalog "test-genie/internal/orchestrator/phases"
	"test-genie/internal/selfhealth"
)

const usage = "usage: provider-contract <check|scan> ...\n  check <phase|provider> <target-scenario> [--no-restart] [--json]\n  scan [<phase-or-provider>] [--json] [--target <fixture-scenario>] [--timeout <dur>] [--restart]"

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
	Phase       string `json:"phase"`
	Provider    string `json:"provider"`
	Restartable bool   `json:"restartable"`
}

type Result struct {
	Phase          string   `json:"phase"`
	Provider       string   `json:"provider"`
	Target         string   `json:"target"`
	Restarted      bool     `json:"restarted"`
	Status         string   `json:"status"`
	Classification string   `json:"classification,omitempty"`
	ReasonCodes    []string `json:"reasonCodes,omitempty"`
	Violations     []string `json:"violations,omitempty"`
	Assessment     struct {
		Scenario                  string               `json:"scenario"`
		Provider                  string               `json:"provider"`
		Phase                     string               `json:"phase"`
		Version                   string               `json:"version"`
		CurrentLevel              string               `json:"currentLevel"`
		NextLevel                 string               `json:"nextLevel,omitempty"`
		Capabilities              []CapabilityResult   `json:"capabilities,omitempty"`
		HighestPriorityCapability *PriorityFocusResult `json:"highestPriorityCapability,omitempty"`
	} `json:"assessment"`
	Presentation *PresentationResult `json:"presentation,omitempty"`

	// presentation retains the provider-owned proto so the human report can
	// render it through assessment.PresentationView; the JSON projection above
	// is the stable CLI summary.
	presentation *commonv1.PhasePresentation
}

// PresentationResult is the JSON summary of the provider-owned canonical
// phase presentation, proving in the check output what the provider returned.
type PresentationResult struct {
	ContractVersion      string   `json:"contractVersion"`
	CurrentLevel         string   `json:"currentLevel,omitempty"`
	CurrentLevelLabel    string   `json:"currentLevelLabel,omitempty"`
	NextLevel            string   `json:"nextLevel,omitempty"`
	CeilingLevel         string   `json:"ceilingLevel,omitempty"`
	AtMaximum            bool     `json:"atMaximum,omitempty"`
	Clean                bool     `json:"clean"`
	NextAction           string   `json:"nextAction,omitempty"`
	NextActionReason     string   `json:"nextActionReason,omitempty"`
	FocusCapabilityID    string   `json:"focusCapabilityId,omitempty"`
	NorthStar            string   `json:"northStar,omitempty"`
	BlockingFindingCodes []string `json:"blockingFindingCodes,omitempty"`
	DocumentationTopics  []string `json:"documentationTopics,omitempty"`
}

type CapabilityResult struct {
	ID                   string `json:"id"`
	Label                string `json:"label,omitempty"`
	CurrentLevel         string `json:"currentLevel"`
	NextLevel            string `json:"nextLevel,omitempty"`
	CurrentSummary       string `json:"currentSummary,omitempty"`
	NextUnlock           string `json:"nextUnlock,omitempty"`
	Clean                bool   `json:"clean"`
	UnknownCount         int32  `json:"unknownCount,omitempty"`
	BlockingFindingCount int    `json:"blockingFindingCount,omitempty"`
	PriorityRank         int32  `json:"priorityRank,omitempty"`
	PriorityReason       string `json:"priorityReason,omitempty"`
}

type PriorityFocusResult struct {
	CapabilityID    string `json:"capabilityId"`
	CapabilityLabel string `json:"capabilityLabel,omitempty"`
	CurrentLevel    string `json:"currentLevel,omitempty"`
	NextLevel       string `json:"nextLevel,omitempty"`
	Reason          string `json:"reason,omitempty"`
}

func Run(args []string) error {
	if len(args) > 0 && args[0] == "scan" {
		return RunScan(args)
	}
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
	if focus := result.Assessment.HighestPriorityCapability; focus != nil && focus.CapabilityID != "" {
		label := focus.CapabilityLabel
		if label == "" {
			label = focus.CapabilityID
		}
		fmt.Printf("  Focus     : %s", label)
		if focus.NextLevel != "" {
			fmt.Printf(" -> %s", focus.NextLevel)
		}
		if focus.Reason != "" {
			fmt.Printf(" (%s)", focus.Reason)
		}
		fmt.Println()
	}
	for _, capability := range result.Assessment.Capabilities {
		label := capability.Label
		if label == "" {
			label = capability.ID
		}
		fmt.Printf("  Capability: %s current=%s", label, capability.CurrentLevel)
		if capability.NextLevel != "" {
			fmt.Printf(" next=%s", capability.NextLevel)
		}
		fmt.Printf(" clean=%t blocking=%d", capability.Clean, capability.BlockingFindingCount)
		if capability.UnknownCount > 0 {
			fmt.Printf(" unknown=%d", capability.UnknownCount)
		}
		fmt.Println()
	}
	view := assessment.PresentationView(result.presentation)
	fmt.Printf("  Presentation: %s\n", view.Summary)
	for _, line := range view.Lines {
		fmt.Printf("    %s\n", line)
	}
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

// fixConformanceProbe is the PreviewFix/ApplyFix seam; tests stub it to avoid
// live fix RPCs. The default is the exact transport the fleet scan uses.
var fixConformanceProbe selfhealth.FixConformanceProbe = selfhealth.DefaultFixConformanceProbe

// Check restarts the provider (unless --no-restart) and scores it through the
// same selfhealth conformance core the fleet scan and the API self-health
// endpoint use. Owning no validation logic of its own is the point: a check
// verdict cannot drift from the run gate or the scan, so "check passed" means
// the provider's response satisfies assessment.RequireProviderContract plus
// the fleet adoption dimensions (descriptor spec, metrics, fix contract).
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
	conformance, resp := selfhealth.CheckProvider(ctx, cliValidationProbe, fixConformanceProbe, scanResolveRepoRoot(), args.Target, probe.Phase, probe.Provider, args.Timeout)
	out.Classification = string(conformance.Classification)
	out.ReasonCodes = append([]string(nil), conformance.ReasonCodes...)
	out.Violations = append([]string(nil), conformance.Violations...)
	if resp != nil {
		fillAssessment(&out, resp.GetAssessment())
	}
	if !conformance.Reachable {
		return out, fmt.Errorf("probe provider %s: %s", probe.Provider, strings.Join(out.Violations, "; "))
	}
	if conformance.HasHardViolation() {
		return out, fmt.Errorf("%s -> %s violates the provider contract [%s]: %s",
			probe.Phase, probe.Provider, strings.Join(out.ReasonCodes, ", "), strings.Join(out.Violations, "; "))
	}
	out.Status = "ok"
	return out, nil
}

func fillAssessment(out *Result, a *commonv1.MaturityAssessment) {
	if a == nil {
		return
	}
	out.Assessment.Scenario = a.GetScenario()
	out.Assessment.Provider = a.GetProvider()
	out.Assessment.Phase = a.GetPhase()
	out.Assessment.Version = a.GetVersion()
	out.Assessment.CurrentLevel = a.GetLocal().GetCurrentLevel()
	out.Assessment.NextLevel = a.GetLocal().GetNextLevel()
	out.Assessment.Capabilities = capabilityResults(a.GetCapabilities())
	if focus := a.GetHighestPriorityCapability(); focus != nil && strings.TrimSpace(focus.GetCapabilityId()) != "" {
		out.Assessment.HighestPriorityCapability = &PriorityFocusResult{
			CapabilityID:    focus.GetCapabilityId(),
			CapabilityLabel: focus.GetCapabilityLabel(),
			CurrentLevel:    focus.GetCurrentLevel(),
			NextLevel:       focus.GetNextLevel(),
			Reason:          focus.GetReason(),
		}
	}
	p := a.GetPresentation()
	if p == nil {
		return
	}
	out.presentation = p
	out.Presentation = &PresentationResult{
		ContractVersion:      p.GetContractVersion(),
		CurrentLevel:         p.GetCurrentLevel(),
		CurrentLevelLabel:    p.GetCurrentLevelLabel(),
		NextLevel:            p.GetNextLevel(),
		CeilingLevel:         p.GetCeilingLevel(),
		AtMaximum:            p.GetAtMaximum(),
		Clean:                p.GetClean(),
		NextAction:           p.GetNextAction(),
		NextActionReason:     p.GetNextActionReason(),
		FocusCapabilityID:    p.GetFocusCapabilityId(),
		NorthStar:            p.GetNorthStar(),
		BlockingFindingCodes: append([]string(nil), p.GetBlockingFindingCodes()...),
		DocumentationTopics:  append([]string(nil), p.GetDocumentationTopics()...),
	}
}

// cliValidationProbe adapts the CLI's provider URL seam to the shared
// conformance probe signature.
func cliValidationProbe(ctx context.Context, provider, target string, timeout time.Duration) (*scenariovalidationv1.ValidateScenarioResponse, error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	baseURL, err := resolveProviderBaseURL(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("resolve provider %s URL: %w", provider, err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("provider %s base URL is empty", provider)
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	client := scenariovalidationconnect.NewScenarioValidationServiceClient(&http.Client{Timeout: timeout}, baseURL)
	resp, err := client.ValidateScenario(runCtx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: target}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func capabilityResults(capabilities []*commonv1.CapabilityMaturityAssessment) []CapabilityResult {
	out := make([]CapabilityResult, 0, len(capabilities))
	for _, capability := range capabilities {
		if capability == nil {
			continue
		}
		out = append(out, CapabilityResult{
			ID:                   capability.GetId(),
			Label:                capability.GetLabel(),
			CurrentLevel:         capability.GetCurrentLevel(),
			NextLevel:            capability.GetNextLevel(),
			CurrentSummary:       capability.GetCurrentSummary(),
			NextUnlock:           capability.GetNextUnlock(),
			Clean:                capability.GetClean(),
			UnknownCount:         capability.GetUnknownCount(),
			BlockingFindingCount: len(capability.GetBlockingFindingCodes()),
			PriorityRank:         capability.GetPriorityRank(),
			PriorityReason:       capability.GetPriorityReason(),
		})
	}
	return out
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
		probes = append(probes, probe)
	}
	return probes
}
