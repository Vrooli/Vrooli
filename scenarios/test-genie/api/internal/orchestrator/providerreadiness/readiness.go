// Package providerreadiness evaluates execution-time readiness for provider-
// backed phases after applicability/selection and before target runtime startup.
package providerreadiness

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	OutcomeSkippedBestEffort OutcomeStatus = "skipped_best_effort"
)

type Input struct {
	Phase            string
	ProviderScenario string
	TargetScenario   string
	TargetPath       string
	Policy           phasepolicy.Policy
	Timeout          time.Duration
	// ExpectedSpecVersion is the maturity spec version the provider's descriptor
	// declares on disk. When set, FreshnessRequireFreshBinary compares it against
	// the version the running binary reports and refuses a mismatch. Empty means
	// the caller has no expectation to enforce.
	ExpectedSpecVersion string
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
	SpecVersion      string        `json:"specVersion,omitempty"`
	BuildRevision    string        `json:"buildRevision,omitempty"`
	FreshnessDigest  string        `json:"freshnessDigest,omitempty"`
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
	Reachable     bool
	ContractValid bool
	IdentityMatch bool
	// SpecVersion is the maturity-spec version the *running* provider binary
	// loaded at startup. Comparing it against the version its descriptor
	// currently declares on disk is an exact staleness test: a mismatch means
	// the descriptor moved and the binary was never rebuilt, which is the
	// failure mode where a provider serves a maturity ladder that no longer
	// exists. Empty means the provider did not report one.
	SpecVersion string
	// BuildRevision is best-effort build provenance for diagnostics. It is
	// never used to gate: an unset value means unknown, not stale.
	BuildRevision string
	// FreshnessDigest is the build-input digest the running provider reported.
	// Empty means the provider could not stamp one, which is treated as
	// "unknown" and never as "stale".
	FreshnessDigest string
	Message         string
}

type Probe func(context.Context, Input) (ProbeResult, error)

type Lifecycle interface {
	Start(context.Context, string, io.Writer) error
	Restart(context.Context, string, io.Writer) error
}

// DefaultMaxStaleRestarts bounds how many providers one run may restart because
// their binaries no longer match source.
//
// The bound exists because on an active branch "stale" is the normal state, not
// the exception: one edit to a widely shared package legitimately stales most of
// the fleet at once. Restarting all of them would cost far more than the stale
// results do. Past the cap, remaining providers are reported rather than
// restarted, so the run still finishes and the operator still learns about it.
const DefaultMaxStaleRestarts = 4

// Discovery is shared host/control-plane work, not provider work. Multiple
// durable Test Genie runs can enter readiness at once (for example, a Git
// Control Tower baseline collection), so a per-run worker limit is not enough
// to protect api-core discovery from a burst of identical lookups.
const defaultProviderDiscoveryConcurrency = 4

var providerDiscoverySlots = make(chan struct{}, defaultProviderDiscoveryConcurrency)

type Manager struct {
	Probe     Probe
	Lifecycle Lifecycle
	// RepoRoot enables the staleness check. Empty disables it entirely — the
	// fail-open default, so a misconfigured root can never cause restarts.
	RepoRoot string
	// MaxStaleRestarts bounds stale-triggered restarts per run. Zero uses
	// DefaultMaxStaleRestarts; negative disables restarting while still
	// reporting staleness.
	MaxStaleRestarts int
	// RestartCooldown suppresses repeat restarts of a provider that was
	// restarted recently and has only been staled again by churn outside its own
	// tree. Zero uses DefaultRestartCooldown; negative disables the cooldown.
	RestartCooldown time.Duration
	// Ledger persists restart times across runs. Nil disables the cooldown,
	// since a window that spans runs cannot be enforced from memory.
	Ledger *restartLedger
	// Now is the clock seam for tests.
	Now func() time.Time

	// restartMu serializes lifecycle mutations and the per-run restart budget.
	// Readiness probes may run concurrently, but restarting a provider is a
	// process mutation and must never overlap another restart.
	restartMu     sync.Mutex
	staleRestarts int
	staleSkipped  []string
	// providerLocks serialize the complete readiness lifecycle for one
	// provider across durable runs. Different providers remain concurrent, but
	// two runs can never restart/start/probe the same provider simultaneously.
	providerLocks sync.Map // map[string]*sync.Mutex
}

// StaleReport summarizes what the staleness rails did during a run, so the cost
// is visible in run output instead of being absorbed silently.
type StaleReport struct {
	Restarted int
	Skipped   []string
}

// StaleReport returns the rails' outcome for this run.
func (m *Manager) StaleReport() StaleReport {
	if m == nil {
		return StaleReport{}
	}
	return StaleReport{Restarted: m.staleRestarts, Skipped: append([]string(nil), m.staleSkipped...)}
}

// restartIfStale enforces the freshness gate after a provider probes ready.
//
// It never cascades: only this provider is restarted, and its own preflight
// decides whether its dependencies need rebuilding. A restart is attempted at
// most once per provider per run.
func (m *Manager) restartIfStale(ctx context.Context, in Input, result ProbeResult, logWriter io.Writer) (restarted bool, note string) {
	if m == nil || strings.TrimSpace(m.RepoRoot) == "" {
		return false, ""
	}
	verdict := EvaluateProviderStaleness(m.RepoRoot, in.ProviderScenario, result.FreshnessDigest)
	if !verdict.Stale {
		return false, ""
	}
	m.restartMu.Lock()
	defer m.restartMu.Unlock()
	limit := m.MaxStaleRestarts
	if limit == 0 {
		limit = DefaultMaxStaleRestarts
	}
	detail := verdict.Describe()

	// Damp repeat restarts driven by churn outside this provider's own tree.
	// While someone is actively editing a shared package, rebuilding the same
	// providers every run costs a great deal and changes their verdicts little.
	window := m.RestartCooldown
	if window == 0 {
		window = DefaultRestartCooldown
	}
	if cooling, remaining := m.Ledger.cooling(in.ProviderScenario, verdict.Class, window, m.now()); cooling {
		m.staleSkipped = append(m.staleSkipped, in.ProviderScenario)
		return false, fmt.Sprintf("%s is stale — %s — but was restarted within the last %s; holding off for %s rather than rebuilding on every run while shared code churns",
			in.ProviderScenario, detail, window.Round(time.Minute), remaining.Round(time.Minute))
	}
	if limit < 0 || m.staleRestarts >= limit {
		m.staleSkipped = append(m.staleSkipped, in.ProviderScenario)
		return false, fmt.Sprintf("%s is stale — %s — but the per-run restart budget is spent; this phase's findings come from a binary that predates that change", in.ProviderScenario, detail)
	}
	lifecycle := m.Lifecycle
	if lifecycle == nil {
		lifecycle = CommandLifecycle{}
	}
	if err := lifecycle.Restart(ctx, in.ProviderScenario, logWriter); err != nil {
		// A failed restart must not fail the phase: the provider is still
		// answering, just with older code. Report and continue.
		m.staleSkipped = append(m.staleSkipped, in.ProviderScenario)
		return false, fmt.Sprintf("%s is stale — %s — and could not be restarted: %v", in.ProviderScenario, detail, err)
	}
	m.staleRestarts++
	m.Ledger.record(in.ProviderScenario, verdict.Class, m.now())
	return true, fmt.Sprintf("restarted %s — %s", in.ProviderScenario, detail)
}

func NewManager() *Manager {
	return &Manager{
		Probe:     DefaultProbe,
		Lifecycle: CommandLifecycle{},
	}
}

func (m *Manager) Check(ctx context.Context, in Input, logWriter io.Writer) Outcome {
	if m == nil {
		m = NewManager()
	}
	in.Phase = strings.TrimSpace(in.Phase)
	in.ProviderScenario = strings.TrimSpace(in.ProviderScenario)
	if in.ProviderScenario != "" {
		lockValue, _ := m.providerLocks.LoadOrStore(in.ProviderScenario, &sync.Mutex{})
		lockValue.(*sync.Mutex).Lock()
		defer lockValue.(*sync.Mutex).Unlock()
	}
	if in.Policy.IsZero() {
		in.Policy = phasepolicy.RequiredProviderPolicy()
	}
	if in.Policy.ProviderReadiness == phasepolicy.ProviderReadinessNone || in.ProviderScenario == "" {
		return Outcome{Phase: in.Phase, ProviderScenario: in.ProviderScenario, Status: OutcomeReady, Ready: true}
	}
	probe := m.Probe
	if probe == nil {
		probe = DefaultProbe
	}

	switch in.Policy.ProviderLifecycle {
	case phasepolicy.ProviderLifecycleNone, phasepolicy.ProviderLifecycleCheckOnly, "":
		return classify(in, probeOnce(ctx, probe, in), false, false)
	case phasepolicy.ProviderLifecycleStartIfNeeded:
		probed := probeOnce(ctx, probe, in)
		first := classify(in, probed, false, false)
		if first.Ready {
			// A ready provider can still be serving superseded code. Restarting
			// makes its own preflight rebuild and re-exec, so the phase scores
			// against current source rather than whatever was built last week.
			if restarted, note := m.restartIfStale(ctx, in, probed.result, logWriter); note != "" {
				if restarted {
					reprobed := classify(in, probeOnce(ctx, probe, in), true, false)
					reprobed.Message = joinMessages(reprobed.Message, note)
					return reprobed
				}
				first.Message = joinMessages(first.Message, note)
			}
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
		SpecVersion:      result.SpecVersion,
		BuildRevision:    result.BuildRevision,
		FreshnessDigest:  result.FreshnessDigest,
	}
	if in.Policy.Freshness == phasepolicy.FreshnessRequireLiveContract && !result.ContractValid {
		return blocking(in, OutcomeContractInvalid, errors.New(emptyAs(result.Message, "provider contract probe failed")))
	}
	if !result.IdentityMatch {
		return blocking(in, OutcomeIdentityMismatch, errors.New(emptyAs(result.Message, "provider maturity identity mismatch")))
	}
	if in.Policy.Freshness == phasepolicy.FreshnessRequireFreshBinary {
		if err := checkFreshBinary(in, result); err != nil {
			return blocking(in, OutcomeStale, err)
		}
	}
	return out
}

// checkFreshBinary enforces FreshnessRequireFreshBinary against the spec version
// the running provider binary actually loaded.
//
// This replaces two gates that could never fire: the previous probe never set a
// Stale flag, and it gated on the presence of ExecutionMetrics — a property of a
// validation *result*, not of provider readiness, so a healthy provider could be
// refused for a reason unrelated to whether it was ready.
//
// The comparison here is exact rather than heuristic. Binary timestamps are not
// consulted, because a git checkout rewrites them and a wrong staleness verdict
// would block a phase for no reason.
func checkFreshBinary(in Input, result ProbeResult) error {
	if strings.TrimSpace(result.SpecVersion) == "" {
		return errors.New(emptyAs(result.Message,
			"provider reported no maturity spec version, so binary freshness cannot be proven"))
	}
	want := strings.TrimSpace(in.ExpectedSpecVersion)
	if want == "" {
		return nil
	}
	if got := strings.TrimSpace(result.SpecVersion); got != want {
		return fmt.Errorf(
			"provider binary serves maturity spec %q but its descriptor declares %q; rebuild and restart %s",
			got, want, in.ProviderScenario)
	}
	return nil
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

// DescribeTimeout bounds the DescribeProvider probe. That RPC answers from
// facts the provider resolved at startup and touches no target, so it is O(1)
// no matter how large the scenario under test is. A provider that cannot answer
// it within this budget is not healthy, and saying so quickly is the point:
// the legacy probe took over 100s against a large target purely because it ran
// the provider's full analysis to read two identity strings.
const DescribeTimeout = 30 * time.Second

// DefaultProbe establishes provider readiness: is this provider live, does it
// speak the current contract, and is it the provider its descriptor claims.
//
// It prefers DescribeProvider, which answers all three from provider-owned
// facts. Providers that have not adopted that RPC return Unimplemented, and
// only then does the probe fall back to the legacy ValidateScenario call —
// which answers the same questions but pays a full target analysis to do it.
func DefaultProbe(ctx context.Context, in Input) (ProbeResult, error) {
	if strings.TrimSpace(in.ProviderScenario) == "" {
		return ProbeResult{}, errors.New("provider scenario is required")
	}
	if strings.TrimSpace(in.TargetScenario) == "" {
		return ProbeResult{}, errors.New("target scenario is required")
	}
	if err := acquireProviderDiscoverySlot(ctx); err != nil {
		return ProbeResult{}, fmt.Errorf("wait for provider discovery capacity: %w", err)
	}
	defer releaseProviderDiscoverySlot()
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, in.ProviderScenario)
	if err != nil {
		return ProbeResult{}, fmt.Errorf("resolve %s URL: %w", in.ProviderScenario, err)
	}
	return probeAt(ctx, baseURL, in)
}

func acquireProviderDiscoverySlot(ctx context.Context) error {
	select {
	case providerDiscoverySlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseProviderDiscoverySlot() {
	<-providerDiscoverySlots
}

// probeAt is DefaultProbe with the provider URL already resolved. It owns the
// fast-path/fallback decision, which is the behavior worth testing directly:
// only Unimplemented may fall back, because every other failure means the
// provider is unhealthy and running its full analysis would not fix that.
func probeAt(ctx context.Context, baseURL string, in Input) (ProbeResult, error) {
	result, describeErr := describeProbe(ctx, baseURL, in)
	if describeErr == nil {
		return result, nil
	}
	if connect.CodeOf(describeErr) != connect.CodeUnimplemented {
		return ProbeResult{}, describeErr
	}
	return legacyValidateProbe(ctx, baseURL, in)
}

// describeProbe is the fast path: one target-independent RPC.
func describeProbe(ctx context.Context, baseURL string, in Input) (ProbeResult, error) {
	resp, err := scenariovalidationconnect.NewScenarioValidationServiceClient(
		&http.Client{Timeout: DescribeTimeout},
		baseURL,
	).DescribeProvider(ctx, connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{}))
	if err != nil {
		// Unimplemented is propagated verbatim so DefaultProbe can distinguish
		// "provider has not adopted this yet" from "provider is unhealthy".
		if connect.CodeOf(err) == connect.CodeUnimplemented {
			return ProbeResult{}, err
		}
		return ProbeResult{}, fmt.Errorf("%s provider description probe failed: %w", in.ProviderScenario, err)
	}
	if resp == nil || resp.Msg == nil {
		return ProbeResult{Reachable: true}, errors.New("provider returned an empty description")
	}
	msg := resp.Msg

	// A provider that names no contract is not speaking a contract we can gate
	// on. This is the DescribeProvider analogue of an unspecified status.
	contract := strings.TrimSpace(msg.GetContract())
	if contract == "" {
		return ProbeResult{Reachable: true, ContractValid: false, Message: "provider reported no validation contract"}, nil
	}

	if got := strings.TrimSpace(msg.GetProvider()); got != strings.TrimSpace(in.ProviderScenario) {
		return ProbeResult{
			Reachable:     true,
			ContractValid: true,
			IdentityMatch: false,
			Message:       fmt.Sprintf("description.provider=%q, want %q", got, in.ProviderScenario),
		}, nil
	}
	if want := strings.TrimSpace(in.Phase); want != "" {
		if got := strings.TrimSpace(msg.GetPhase()); got != want {
			return ProbeResult{
				Reachable:     true,
				ContractValid: true,
				IdentityMatch: false,
				Message:       fmt.Sprintf("description.phase=%q, want %q", got, want),
			}, nil
		}
	}

	return ProbeResult{
		Reachable:       true,
		ContractValid:   true,
		IdentityMatch:   true,
		SpecVersion:     strings.TrimSpace(msg.GetSpecVersion()),
		BuildRevision:   strings.TrimSpace(msg.GetBuild().GetRevision()),
		FreshnessDigest: strings.TrimSpace(msg.GetBuild().GetFreshnessDigest()),
	}, nil
}

// legacyValidateProbe is the pre-DescribeProvider readiness path, kept so a
// provider can adopt the fast RPC on its own schedule. It answers the same
// three questions, but pays a full target analysis to do it — for an
// inspection-only provider that is the provider's entire phase workload, run
// twice per suite.
func legacyValidateProbe(ctx context.Context, baseURL string, in Input) (ProbeResult, error) {
	timeout := in.Timeout
	if timeout <= 0 {
		timeout = phases.DefaultTimeout
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
	if err := assessment.RequireIdentity(in.ProviderScenario, in.Phase, msg.GetAssessment()); err != nil {
		return ProbeResult{
			Reachable:     true,
			ContractValid: true,
			IdentityMatch: false,
			Message:       err.Error(),
		}, nil
	}
	return ProbeResult{
		Reachable:     true,
		ContractValid: true,
		IdentityMatch: true,
		SpecVersion:   strings.TrimSpace(msg.GetAssessment().GetVersion()),
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
	var output bytes.Buffer
	var writer io.Writer = &output
	if logWriter != nil {
		writer = io.MultiWriter(&output, logWriter)
	}
	cmd.Stdout = writer
	cmd.Stderr = writer
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(output.String())
		if len(detail) > 2000 {
			detail = detail[:2000] + "...(truncated)"
		}
		if detail == "" {
			return fmt.Errorf("vrooli %s failed: %w (no command output captured)", strings.Join(args, " "), err)
		}
		return fmt.Errorf("vrooli %s failed: %w; command output: %s", strings.Join(args, " "), err, detail)
	}
	return nil
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
	case OutcomeContractInvalid, OutcomeIdentityMismatch, OutcomeStale:
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
	case OutcomeContractInvalid, OutcomeIdentityMismatch, OutcomeStale:
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

// joinMessages appends a staleness note without discarding an existing message.
func joinMessages(existing, note string) string {
	existing = strings.TrimSpace(existing)
	note = strings.TrimSpace(note)
	switch {
	case existing == "":
		return note
	case note == "":
		return existing
	default:
		return existing + "; " + note
	}
}

func (m *Manager) now() time.Time {
	if m != nil && m.Now != nil {
		return m.Now()
	}
	return time.Now()
}
