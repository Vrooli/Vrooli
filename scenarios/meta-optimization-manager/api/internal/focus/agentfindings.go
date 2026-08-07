package focus

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/api-core/discovery"
	agentapi "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	agentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	agentdomain "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	episodeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain/domainconnect"
)

const agentManagerScenario = "agent-manager"

// agentManagerReadDeadline mirrors coverage's numeratorDeadline so all
// cross-scenario read lanes have the same bounded per-owner budget. Revisit
// together with coverage when owner latency evidence changes.
const agentManagerReadDeadline = 3 * time.Second

// AgentFindingObservation is the bounded, attributed friction signal consumed
// by the focus domain. Run and episode locators keep every emitted gap
// independently auditable.
type AgentFindingObservation struct {
	RunID           string
	EpisodeID       string
	Fingerprint     string
	Pattern         string
	OwnerScenario   string
	OwnerConfidence string
	EvidenceLocator string
	At              time.Time
}

type AgentFindingReport struct {
	Findings            []AgentFindingObservation
	DroppedUnattributed int
}

// AgentFindingReader is the read-only seam for agent-manager's observed run
// evidence. Tests fake this report; no test reaches a live agent-manager.
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
// recurring-friction lane. Revisit after attribution coverage improves and the
// run corpus is large enough to calibrate recurrence against interventions.
const agentMinimumRecurrence = 2

func (s *agentManagerGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s.reader == nil {
		return nil, fmt.Errorf("agent-manager findings reader is not configured")
	}
	report, err := s.reader.ReadFindings(ctx)
	if err != nil {
		return nil, fmt.Errorf("read agent-manager findings: %w", err)
	}
	type cluster struct {
		observations map[string]AgentFindingObservation
	}
	clusters := make(map[string]*cluster)
	for _, finding := range report.Findings {
		fingerprint := strings.TrimSpace(finding.Fingerprint)
		if fingerprint == "" {
			continue
		}
		item := clusters[fingerprint]
		if item == nil {
			item = &cluster{observations: map[string]AgentFindingObservation{}}
			clusters[fingerprint] = item
		}
		// One run may produce multiple episodes for one fingerprint; recurrence
		// is independent observations, so retain only one observation per run.
		if _, exists := item.observations[finding.RunID]; !exists {
			item.observations[finding.RunID] = finding
		}
	}

	fingerprints := make([]string, 0, len(clusters))
	for fingerprint := range clusters {
		fingerprints = append(fingerprints, fingerprint)
	}
	sort.Strings(fingerprints)
	out := make([]Gap, 0, len(fingerprints)+1)
	for _, fingerprint := range fingerprints {
		observations := make([]AgentFindingObservation, 0, len(clusters[fingerprint].observations))
		for _, finding := range clusters[fingerprint].observations {
			observations = append(observations, finding)
		}
		sort.SliceStable(observations, func(i, j int) bool {
			if !observations[i].At.Equal(observations[j].At) {
				return observations[i].At.After(observations[j].At)
			}
			return observations[i].RunID > observations[j].RunID
		})
		if len(observations) < agentMinimumRecurrence {
			continue
		}
		newest := observations[0]
		out = append(out, Gap{
			ID:              "empirical/agent-manager/" + fingerprint,
			Axis:            AxisEmpirical,
			Title:           fmt.Sprintf("agent-manager finding %s recurs", fingerprint),
			Global:          true,
			EvidenceSource:  "agent-manager",
			EvidenceLocator: newest.EvidenceLocator,
			Recurrence:      len(observations),
			Notes:           []string{fmt.Sprintf("pattern=%s; owner=%s (%s)", newest.Pattern, newest.OwnerScenario, newest.OwnerConfidence)},
		})
	}
	if report.DroppedUnattributed > 0 {
		out = append(out, Gap{
			ID:                 "source/agent-manager/unattributed",
			Axis:               AxisEmpirical,
			Title:              "agent-manager dropped unattributed friction evidence",
			Global:             true,
			EvidenceSource:     "agent-manager",
			AvailabilityReason: fmt.Sprintf("dropped %d finding observation(s) because run attribution was unknown", report.DroppedUnattributed),
		})
	}
	return out, nil
}

type agentManagerRunClient interface {
	ListRuns(context.Context, *connect.Request[agentapi.ListRunsRequest]) (*connect.Response[agentapi.ListRunsResponse], error)
}

type agentManagerEpisodeClient interface {
	GetEpisodes(context.Context, *connect.Request[agentdomain.GetEpisodesRequest]) (*connect.Response[agentdomain.GetEpisodesResponse], error)
}

type agentManagerClientFactory interface {
	New(httpClient connect.HTTPClient, baseURL string) (agentManagerRunClient, agentManagerEpisodeClient)
}

type productionAgentManagerClientFactory struct{}

func (productionAgentManagerClientFactory) New(httpClient connect.HTTPClient, baseURL string) (agentManagerRunClient, agentManagerEpisodeClient) {
	return agentconnect.NewAgentManagerServiceClient(httpClient, baseURL), episodeconnect.NewEpisodesServiceClient(httpClient, baseURL)
}

type agentManagerFindingReader struct {
	resolver scenarioURLResolver
	http     connect.HTTPClient
	factory  agentManagerClientFactory
	deadline time.Duration
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

func newAgentManagerFindingReader(resolver scenarioURLResolver, httpClient connect.HTTPClient, factory agentManagerClientFactory, deadline time.Duration) AgentFindingReader {
	if deadline <= 0 {
		deadline = agentManagerReadDeadline
	}
	return &agentManagerFindingReader{resolver: resolver, http: httpClient, factory: factory, deadline: deadline}
}

var _ AgentFindingReader = (*agentManagerFindingReader)(nil)

func (r *agentManagerFindingReader) ReadFindings(ctx context.Context) (AgentFindingReport, error) {
	ctx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, agentManagerScenario)
	if err != nil {
		return AgentFindingReport{}, fmt.Errorf("resolve agent-manager: %w", err)
	}
	runsClient, episodesClient := r.factory.New(r.http, base)
	runs, err := listAgentRuns(ctx, runsClient)
	if err != nil {
		return AgentFindingReport{}, err
	}
	report := AgentFindingReport{}
	for _, run := range runs {
		runID := strings.TrimSpace(run.GetId())
		if runID == "" {
			continue
		}
		resp, err := episodesClient.GetEpisodes(ctx, connect.NewRequest(&agentdomain.GetEpisodesRequest{RunId: runID}))
		if err != nil {
			return AgentFindingReport{}, fmt.Errorf("get episodes for run %s: %w", runID, err)
		}
		if resp == nil || resp.Msg == nil {
			return AgentFindingReport{}, fmt.Errorf("agent-manager returned no episodes response for run %s", runID)
		}
		at := time.Time{}
		if createdAt := run.GetCreatedAt(); createdAt != nil {
			at = createdAt.AsTime()
		}
		for _, episode := range resp.Msg.GetEpisodes() {
			if !knownAttribution(episode) {
				report.DroppedUnattributed++
				continue
			}
			fingerprint := strings.TrimSpace(episode.GetFingerprint())
			if fingerprint == "" {
				fingerprint = strings.TrimSpace(episode.GetPattern()) + ":" + strings.TrimSpace(episode.GetCauseScope())
			}
			if fingerprint == ":" {
				continue
			}
			report.Findings = append(report.Findings, AgentFindingObservation{
				RunID:           runID,
				EpisodeID:       episode.GetEpisodeId(),
				Fingerprint:     fingerprint,
				Pattern:         episode.GetPattern(),
				OwnerScenario:   episode.GetSuspectedOwnerScenario(),
				OwnerConfidence: episode.GetOwnerConfidence(),
				EvidenceLocator: fmt.Sprintf("agent-manager://runs/%s/episodes/%s", runID, episode.GetEpisodeId()),
				At:              at,
			})
		}
	}
	return report, nil
}

func listAgentRuns(ctx context.Context, client agentManagerRunClient) ([]*agentdomain.Run, error) {
	const pageSize int32 = 5000
	var out []*agentdomain.Run
	var offset int32
	for {
		resp, err := client.ListRuns(ctx, connect.NewRequest(&agentapi.ListRunsRequest{Limit: proto.Int32(pageSize), Offset: proto.Int32(offset)}))
		if err != nil {
			return nil, fmt.Errorf("list agent-manager runs: %w", err)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("agent-manager returned no runs response")
		}
		page := resp.Msg.GetRuns()
		out = append(out, page...)
		if !resp.Msg.GetHasMore() || len(page) == 0 {
			return out, nil
		}
		offset += int32(len(page))
	}
}

func knownAttribution(episode *agentdomain.FrictionEpisode) bool {
	if episode == nil || strings.TrimSpace(episode.GetSuspectedOwnerScenario()) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(episode.GetOwnerConfidence())) {
	case "", "unknown", "unattributed", "conflicting":
		return false
	default:
		return true
	}
}
