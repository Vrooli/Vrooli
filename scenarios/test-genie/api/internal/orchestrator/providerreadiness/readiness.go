// Package providerreadiness evaluates execution-time readiness for provider-
// backed phases after applicability/selection and before target runtime startup.
package providerreadiness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
)

type OutcomeStatus string

const (
	OutcomeReady             OutcomeStatus = "ready"
	OutcomeStarted           OutcomeStatus = "started"
	OutcomeRestarted         OutcomeStatus = "restarted"
	OutcomeUnreachable       OutcomeStatus = "unreachable"
	OutcomeStale             OutcomeStatus = "stale"
	OutcomeContractInvalid   OutcomeStatus = "contract_invalid"
	OutcomeIdentityMismatch  OutcomeStatus = "identity_mismatch"
	OutcomeMetricsMissing    OutcomeStatus = "metrics_missing"
	OutcomeSkippedBestEffort OutcomeStatus = "skipped_best_effort"
)

type Input struct {
	Phase            string
	ProviderScenario string
	TargetScenario   string
	TargetPath       string
	Policy           phasepolicy.Policy
	Timeout          time.Duration
}

type Outcome struct {
	Phase            string        `json:"phase"`
	ProviderScenario string        `json:"providerScenario,omitempty"`
	Status           OutcomeStatus `json:"status"`
	Ready            bool          `json:"ready"`
	Started          bool          `json:"started,omitempty"`
	Restarted        bool          `json:"restarted,omitempty"`
	BestEffort       bool          `json:"bestEffort,omitempty"`
	Message          string        `json:"message,omitempty"`
	Err              error         `json:"-"`
}

func (o Outcome) BlocksExecution() bool {
	return !o.Ready && o.Status != ""
}

func (o Outcome) SkipsWithoutFailure() bool {
	return o.Status == OutcomeSkippedBestEffort
}

func (o Outcome) ErrorString() string {
	if o.Err != nil {
		return o.Err.Error()
	}
	return strings.TrimSpace(o.Message)
}

type ProbeResult struct {
	Reachable      bool
	ContractValid  bool
	IdentityMatch  bool
	MetricsPresent bool
	Stale          bool
	Message        string
}

type Probe func(context.Context, Input) (ProbeResult, error)

type Lifecycle interface {
	Start(context.Context, string, io.Writer) error
	Restart(context.Context, string, io.Writer) error
}

type Manager struct {
	Probe     Probe
	Lifecycle Lifecycle
}

func NewManager() *Manager {
	return &Manager{
		Probe:     DefaultProbe,
		Lifecycle: CommandLifecycle{},
	}
}

func (m *Manager) Check(ctx context.Context, in Input, logWriter io.Writer) Outcome {
	in.Phase = strings.TrimSpace(in.Phase)
	in.ProviderScenario = strings.TrimSpace(in.ProviderScenario)
	if in.Policy.IsZero() {
		in.Policy = phasepolicy.RequiredProviderPolicy()
	}
	if in.Policy.ProviderReadiness == phasepolicy.ProviderReadinessNone || in.ProviderScenario == "" {
		return Outcome{Phase: in.Phase, ProviderScenario: in.ProviderScenario, Status: OutcomeReady, Ready: true}
	}
	if m == nil {
		m = NewManager()
	}
	probe := m.Probe
	if probe == nil {
		probe = DefaultProbe
	}

	switch in.Policy.ProviderLifecycle {
	case phasepolicy.ProviderLifecycleNone, phasepolicy.ProviderLifecycleCheckOnly, "":
		return classify(in, probeOnce(ctx, probe, in), false, false)
	case phasepolicy.ProviderLifecycleStartIfNeeded:
		first := classify(in, probeOnce(ctx, probe, in), false, false)
		if first.Ready {
			return first
		}
		lifecycle := m.Lifecycle
		if lifecycle == nil {
			lifecycle = CommandLifecycle{}
		}
		if err := lifecycle.Start(ctx, in.ProviderScenario, logWriter); err != nil {
			return unavailable(in, fmt.Errorf("start provider %s: %w", in.ProviderScenario, err))
		}
		return classify(in, probeOnce(ctx, probe, in), true, false)
	case phasepolicy.ProviderLifecycleRestartBeforeProbe:
		lifecycle := m.Lifecycle
		if lifecycle == nil {
			lifecycle = CommandLifecycle{}
		}
		if err := lifecycle.Restart(ctx, in.ProviderScenario, logWriter); err != nil {
			return unavailable(in, fmt.Errorf("restart provider %s: %w", in.ProviderScenario, err))
		}
		return classify(in, probeOnce(ctx, probe, in), false, true)
	default:
		return unavailable(in, fmt.Errorf("unsupported provider lifecycle policy %q", in.Policy.ProviderLifecycle))
	}
}

type probeOutcome struct {
	result ProbeResult
	err    error
}

func probeOnce(ctx context.Context, probe Probe, in Input) probeOutcome {
	result, err := probe(ctx, in)
	return probeOutcome{result: result, err: err}
}

func classify(in Input, probed probeOutcome, started, restarted bool) Outcome {
	if probed.err != nil {
		return unavailable(in, probed.err)
	}
	result := probed.result
	if !result.Reachable {
		return unavailable(in, errors.New(emptyAs(result.Message, "provider is unreachable")))
	}
	status := OutcomeReady
	if started {
		status = OutcomeStarted
	}
	if restarted {
		status = OutcomeRestarted
	}
	out := Outcome{
		Phase:            in.Phase,
		ProviderScenario: in.ProviderScenario,
		Status:           status,
		Ready:            true,
		Started:          started,
		Restarted:        restarted,
		Message:          result.Message,
	}
	if in.Policy.Freshness == phasepolicy.FreshnessRequireLiveContract && !result.ContractValid {
		return blocking(in, OutcomeContractInvalid, errors.New(emptyAs(result.Message, "provider contract probe failed")))
	}
	if !result.IdentityMatch {
		return blocking(in, OutcomeIdentityMismatch, errors.New(emptyAs(result.Message, "provider maturity identity mismatch")))
	}
	if in.Policy.Freshness == phasepolicy.FreshnessRequireFreshBinary && result.Stale {
		return blocking(in, OutcomeStale, errors.New(emptyAs(result.Message, "provider binary is stale")))
	}
	if in.Policy.Freshness == phasepolicy.FreshnessRequireFreshBinary && !result.MetricsPresent {
		return blocking(in, OutcomeMetricsMissing, errors.New(emptyAs(result.Message, "provider metrics are missing")))
	}
	return out
}

func unavailable(in Input, err error) Outcome {
	return blocking(in, OutcomeUnreachable, err)
}

func blocking(in Input, status OutcomeStatus, err error) Outcome {
	bestEffort := in.Policy.ProviderReadiness == phasepolicy.ProviderReadinessBestEffort ||
		in.Policy.Unavailable == phasepolicy.UnavailableSkipWithoutFailing ||
		in.Policy.Unavailable == phasepolicy.UnavailableAdvisory
	if bestEffort {
		return Outcome{
			Phase:            in.Phase,
			ProviderScenario: in.ProviderScenario,
			Status:           OutcomeSkippedBestEffort,
			BestEffort:       true,
			Message:          err.Error(),
			Err:              err,
		}
	}
	return Outcome{
		Phase:            in.Phase,
		ProviderScenario: in.ProviderScenario,
		Status:           status,
		Message:          err.Error(),
		Err:              err,
	}
}

func DefaultProbe(ctx context.Context, in Input) (ProbeResult, error) {
	if strings.TrimSpace(in.ProviderScenario) == "" {
		return ProbeResult{}, errors.New("provider scenario is required")
	}
	if strings.TrimSpace(in.TargetScenario) == "" {
		return ProbeResult{}, errors.New("target scenario is required")
	}
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = phases.DefaultTimeout
	}
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, in.ProviderScenario)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("resolve %s URL: %w", in.ProviderScenario, err)
	}
	resp, err := scenariovalidationconnect.NewScenarioValidationServiceClient(
		&http.Client{Timeout: timeout},
		baseURL,
	).ValidateScenario(ctx, connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         strings.TrimSpace(in.TargetScenario),
		Path:             strings.TrimSpace(in.TargetPath),
		IncludeExecution: false,
	}))
	if err != nil {
		return ProbeResult{}, fmt.Errorf("%s validation contract probe failed: %w", in.ProviderScenario, err)
	}
	if resp == nil || resp.Msg == nil {
		return ProbeResult{Reachable: true}, errors.New("provider returned an empty validation response")
	}
	msg := resp.Msg
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		return ProbeResult{Reachable: true, ContractValid: false, Message: "provider returned unspecified validation status"}, nil
	}
	identityMatch := true
	if err := assessment.RequireIdentity(in.ProviderScenario, in.Phase, msg.GetAssessment()); err != nil {
		identityMatch = false
		return ProbeResult{
			Reachable:      true,
			ContractValid:  true,
			IdentityMatch:  false,
			MetricsPresent: msg.GetMetrics() != nil,
			Message:        err.Error(),
		}, nil
	}
	return ProbeResult{
		Reachable:      true,
		ContractValid:  true,
		IdentityMatch:  identityMatch,
		MetricsPresent: msg.GetMetrics() != nil,
	}, nil
}

type CommandLifecycle struct{}

func (CommandLifecycle) Start(ctx context.Context, scenario string, logWriter io.Writer) error {
	return run(ctx, logWriter, "scenario", "start", strings.TrimSpace(scenario), "--clean-stale")
}

func (CommandLifecycle) Restart(ctx context.Context, scenario string, logWriter io.Writer) error {
	return run(ctx, logWriter, "scenario", "restart", strings.TrimSpace(scenario), "--clean-stale")
}

func run(ctx context.Context, logWriter io.Writer, args ...string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("provider scenario is required")
	}
	cmd := exec.CommandContext(ctx, "vrooli", args...)
	cmd.Env = os.Environ()
	if logWriter != nil {
		cmd.Stdout = logWriter
		cmd.Stderr = logWriter
	}
	return cmd.Run()
}

func ResultObservation(out Outcome) phases.Observation {
	switch {
	case out.Ready:
		return phases.NewInfoObservation(fmt.Sprintf("provider readiness: %s is %s", out.ProviderScenario, out.Status))
	case out.SkipsWithoutFailure():
		return phases.NewSkipObservation(fmt.Sprintf("provider readiness: %s skipped because %s is unavailable: %s", out.Phase, out.ProviderScenario, out.ErrorString()))
	default:
		return phases.NewErrorObservation(fmt.Sprintf("provider readiness: %s unavailable for %s: %s", out.ProviderScenario, out.Phase, out.ErrorString()))
	}
}

func FailureClass(out Outcome) string {
	switch out.Status {
	case OutcomeContractInvalid, OutcomeIdentityMismatch, OutcomeMetricsMissing:
		return phases.FailureClassMaturityContract
	default:
		return phases.FailureClassMissingDependency
	}
}

func Remediation(out Outcome) string {
	if out.SkipsWithoutFailure() {
		return "Start " + out.ProviderScenario + " through lifecycle if this best-effort phase should run."
	}
	switch out.Status {
	case OutcomeContractInvalid, OutcomeIdentityMismatch, OutcomeMetricsMissing:
		return "Run `test-genie provider-contract check " + out.Phase + " " + out.ProviderScenario + " --json` and fix the provider validation contract."
	default:
		return "Ensure " + out.ProviderScenario + " is running through `vrooli scenario start " + out.ProviderScenario + "` and reachable."
	}
}

func emptyAs(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

var _ Lifecycle = CommandLifecycle{}
