package focus

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/api-core/discovery"
	agentmeasures "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	sharedmeasures "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

const agentManagerScenario = "agent-manager"

// agentManagerReadDeadline mirrors coverage's numeratorDeadline so all
// cross-scenario read lanes have the same bounded per-owner budget. Revisit
// together with coverage when owner latency evidence changes.
const agentManagerReadDeadline = 3 * time.Second

// agentFindingLookback is the tool-friction window read per refresh: long
// enough for a problem to recur across runs, short enough for agent-manager to
// profile it inside the read deadline.
const agentFindingLookback = 24 * time.Hour

// AgentFindingObservation is one evidence run of an owner-level agent-manager
// tool finding. Run and finding locators keep every emitted gap independently
// auditable.
type AgentFindingObservation struct {
	RunID           string
	Fingerprint     string
	Kind            string
	OwnerScenario   string
	Description     string
	EvidenceLocator string
}

type AgentFindingReport struct {
	Findings []AgentFindingObservation
}

// AgentFindingReader is the read-only seam for agent-manager's observed tool
// friction. Tests fake this report; no test reaches a live agent-manager.
type AgentFindingReader interface {
	ReadFindings(ctx context.Context) (AgentFindingReport, error)
}

type agentManagerGapSource struct {
	reader AgentFindingReader
}

func NewAgentManagerGapSource(reader AgentFindingReader) GapSource {
	return &agentManagerGapSource{reader: reader}
}

var _ GapSource = (*agentManagerGapSource)(nil)

func (*agentManagerGapSource) Axis() Axis { return AxisEmpirical }

// agentMinimumRecurrence prevents a one-off observation from entering the
// recurring-friction lane: a finding must have evidence in two runs.
const agentMinimumRecurrence = 2

func (s *agentManagerGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s.reader == nil {
		return nil, fmt.Errorf("agent-manager findings reader is not configured")
	}
	report, err := s.reader.ReadFindings(ctx)
	if err != nil {
		return nil, fmt.Errorf("read agent-manager findings: %w", err)
	}
	clusters := map[string]map[string]AgentFindingObservation{}
	for _, finding := range report.Findings {
		fingerprint := strings.TrimSpace(finding.Fingerprint)
		if fingerprint == "" {
			continue
		}
		if clusters[fingerprint] == nil {
			clusters[fingerprint] = map[string]AgentFindingObservation{}
		}
		// Recurrence is independent runs, so one run counts once.
		if _, exists := clusters[fingerprint][finding.RunID]; !exists {
			clusters[fingerprint][finding.RunID] = finding
		}
	}
	fingerprints := make([]string, 0, len(clusters))
	for fingerprint := range clusters {
		fingerprints = append(fingerprints, fingerprint)
	}
	sort.Strings(fingerprints)
	out := make([]Gap, 0, len(fingerprints))
	for _, fingerprint := range fingerprints {
		runs := clusters[fingerprint]
		if len(runs) < agentMinimumRecurrence {
			continue
		}
		var any AgentFindingObservation
		for _, observation := range runs {
			any = observation
			break
		}
		out = append(out, Gap{
			ID:              "empirical/agent-manager/" + fingerprint,
			Axis:            AxisEmpirical,
			Title:           fmt.Sprintf("agent-manager tool finding %s recurs", fingerprint),
			Global:          true,
			EvidenceSource:  "agent-manager",
			EvidenceLocator: any.EvidenceLocator,
			Recurrence:      len(runs),
			Notes:           []string{fmt.Sprintf("kind=%s; owner=%s; %s", any.Kind, any.OwnerScenario, any.Description)},
		})
	}
	return out, nil
}

type agentManagerFrictionClient interface {
	ToolFriction(context.Context, *connect.Request[agentmeasures.ToolFrictionRequest]) (*connect.Response[agentmeasures.ToolFrictionResponse], error)
}

type agentManagerClientFactory interface {
	New(httpClient connect.HTTPClient, baseURL string) agentManagerFrictionClient
}

type productionAgentManagerClientFactory struct{}

func (productionAgentManagerClientFactory) New(httpClient connect.HTTPClient, baseURL string) agentManagerFrictionClient {
	return measuresconnect.NewMeasuresServiceClient(httpClient, baseURL)
}

type agentManagerFindingReader struct {
	resolver scenarioURLResolver
	http     connect.HTTPClient
	factory  agentManagerClientFactory
	deadline time.Duration
	now      func() time.Time
}

type scenarioURLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// NewAgentManagerFindingReader constructs the production typed Connect reader.
// The resolver is intentionally discovery-backed and the deadline is bounded
// so a missing agent-manager never blocks coverage or trial evidence.
func NewAgentManagerFindingReader() AgentFindingReader {
	return newAgentManagerFindingReader(
		discovery.NewResolver(discovery.ResolverConfig{}),
		&http.Client{Timeout: agentManagerReadDeadline},
		productionAgentManagerClientFactory{},
		agentManagerReadDeadline,
	)
}

func newAgentManagerFindingReader(resolver scenarioURLResolver, httpClient connect.HTTPClient, factory agentManagerClientFactory, deadline time.Duration) *agentManagerFindingReader {
	if deadline <= 0 {
		deadline = agentManagerReadDeadline
	}
	return &agentManagerFindingReader{resolver: resolver, http: httpClient, factory: factory, deadline: deadline, now: time.Now}
}

var _ AgentFindingReader = (*agentManagerFindingReader)(nil)

// ReadFindings reads agent-manager's owner-level tool findings of the last
// day: every open or reported finding contributes one observation per
// evidence run. Dismissed findings are the operator's decision to set aside.
func (r *agentManagerFindingReader) ReadFindings(ctx context.Context) (AgentFindingReport, error) {
	ctx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, agentManagerScenario)
	if err != nil {
		return AgentFindingReport{}, fmt.Errorf("resolve agent-manager: %w", err)
	}
	to := r.now().UTC()
	window := &sharedmeasures.TimeWindow{Window: &sharedmeasures.TimeWindow_Custom{Custom: &sharedmeasures.CustomRange{From: timestamppb.New(to.Add(-agentFindingLookback)), To: timestamppb.New(to)}}}
	resp, err := r.factory.New(r.http, base).ToolFriction(ctx, connect.NewRequest(&agentmeasures.ToolFrictionRequest{Window: window}))
	if err != nil {
		return AgentFindingReport{}, fmt.Errorf("read agent-manager tool friction: %w", err)
	}
	report := AgentFindingReport{}
	for _, finding := range resp.Msg.GetFindings() {
		if finding.GetStatus() == "dismissed" || strings.TrimSpace(finding.GetOwner()) == "" {
			continue
		}
		for _, ref := range finding.GetEvidenceRefs() {
			if ref.GetKind() != "run" || ref.GetId() == "" {
				continue
			}
			report.Findings = append(report.Findings, AgentFindingObservation{
				RunID: ref.GetId(), Fingerprint: finding.GetFingerprint(), Kind: finding.GetKind(), OwnerScenario: finding.GetOwner(), Description: finding.GetDescription(),
				EvidenceLocator: "agent-manager://usage/friction/" + finding.GetFingerprint(),
			})
		}
	}
	return report, nil
}
