package coverage

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/spacedoc"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry/registry_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// numeratorDeadline bounds a single owner's live numerator read. The previous
// substrate spawned the owner's CLI with a 30s timeout, so one slow/hung owner
// (e.g. test-genie health aggregating its fleet ledger) stalled the whole
// scoreboard for ~30s. Reads are now a single typed Connect-RPC call resolved
// through api-core/discovery; they should return sub-second. The short deadline
// turns a slow/unreachable owner into a fast, honest per-projection UNAVAILABLE
// instead of a board-wide hang.
const numeratorDeadline = 3 * time.Second

// scenarioResolver resolves an owner scenario slug to its API base URL. It seams
// over discovery.Resolver so tests can point reads at an httptest server (or
// simulate a not-running owner) without shelling out to the vrooli CLI.
type scenarioResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// apiNumeratorJoiner is the production NumeratorJoiner: it resolves each owner's
// API base URL, calls the owner's typed Connect-RPC numerator read under a short
// deadline, and maps the typed response onto the projection's per-cell statuses.
// Any resolve/RPC/deadline failure yields a graceful Available=false JoinResult
// with an honest reason — never a fabricated number, never a board-wide hang.
type apiNumeratorJoiner struct {
	resolver scenarioResolver
	http     connect.HTTPClient
	deadline time.Duration
}

var _ NumeratorJoiner = (*apiNumeratorJoiner)(nil)

// NewNumeratorJoiner returns the production NumeratorJoiner wired to the CLI-
// backed discovery resolver and an HTTP client bounded by the read deadline.
func NewNumeratorJoiner() NumeratorJoiner {
	return newAPINumeratorJoiner(
		discovery.NewResolver(discovery.ResolverConfig{}),
		&http.Client{Timeout: numeratorDeadline},
		numeratorDeadline,
	)
}

// newAPINumeratorJoiner builds an apiNumeratorJoiner over an injected resolver,
// HTTP client, and deadline (tests).
func newAPINumeratorJoiner(r scenarioResolver, h connect.HTTPClient, deadline time.Duration) *apiNumeratorJoiner {
	if deadline <= 0 {
		deadline = numeratorDeadline
	}
	return &apiNumeratorJoiner{resolver: r, http: h, deadline: deadline}
}

// Join resolves the projection's owner, reads its live numerator over a typed
// client under the per-owner deadline, and returns the joined per-cell statuses.
func (j *apiNumeratorJoiner) Join(ctx context.Context, p Projection, cells []spacedoc.Cell) JoinResult {
	owner := OwnerFor(p)
	if owner == "" {
		return JoinResult{Available: false, Reason: "unknown coverage projection: " + string(p)}
	}

	ctx, cancel := context.WithTimeout(ctx, j.deadline)
	defer cancel()

	base, err := j.resolver.ResolveScenarioURLDefault(ctx, owner)
	if err != nil {
		return JoinResult{Available: false, Reason: resolveReason(owner, err)}
	}

	switch p {
	case ProjectionAnswer:
		client := registryconnect.NewRegistryServiceClient(j.http, base)
		resp, err := client.ListProviders(ctx, connect.NewRequest(&registryv1.ListProvidersRequest{}))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "ListProviders", err)}
		}
		return JoinResult{Available: true, Statuses: recomputeAnswer(cells, providersToLive(resp.Msg))}

	case ProjectionValidate:
		client := runsconnect.NewRunsServiceClient(j.http, base)
		resp, err := client.GetSelfHealth(ctx, connect.NewRequest(&runsv1.GetSelfHealthRequest{}))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "GetSelfHealth", err)}
		}
		return JoinResult{Available: true, Statuses: recomputeValidate(cells, selfHealthToValidateIndex(resp.Msg.GetSelfHealth()))}

	case ProjectionGuide:
		client := graphconnect.NewGraphServiceClient(j.http, base)
		resp, err := client.GetHealthScores(ctx, connect.NewRequest(&graphv1.GetHealthScoresRequest{}))
		if err != nil {
			return JoinResult{Available: false, Reason: rpcReason(owner, "GetHealthScores", err)}
		}
		return JoinResult{Available: true, Statuses: recomputeGuide(cells, scoresToGuideIndex(resp.Msg.GetScores()))}

	default:
		return JoinResult{Available: false, Reason: "unknown coverage projection: " + string(p)}
	}
}

// resolveReason formats an honest reason for a failed owner URL resolution,
// distinguishing the not-running case from other discovery failures.
func resolveReason(owner string, err error) string {
	if discovery.IsScenarioNotRunning(err) {
		return owner + " not running"
	}
	return owner + " registry unreachable: " + err.Error()
}

// rpcReason formats an honest reason for a failed owner RPC. A deadline trip
// reads as a timeout; any other failure carries the underlying error.
func rpcReason(owner, method string, err error) string {
	if isDeadline(err) {
		return fmt.Sprintf("%s %s timed out", owner, method)
	}
	return fmt.Sprintf("%s %s failed: %v", owner, method, err)
}

func isDeadline(err error) bool {
	if connect.CodeOf(err) == connect.CodeDeadlineExceeded {
		return true
	}
	return strings.Contains(err.Error(), context.DeadlineExceeded.Error())
}

// providersToLive distills a search-hub ListProviders response into the live
// provider key set the Answer join matches against. Each provider contributes
// its id, group, and type (lower-cased, non-empty), mirroring the keys the
// previous JSON-walking joiner collected.
func providersToLive(resp *registryv1.ListProvidersResponse) map[string]bool {
	out := map[string]bool{}
	if resp == nil {
		return out
	}
	add := func(s string) {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out[s] = true
		}
	}
	for _, p := range resp.GetProviders() {
		add(p.GetProviderId())
		add(p.GetProviderGroup())
		add(p.GetType())
	}
	return out
}

// selfHealthToValidateIndex distills a test-genie SelfHealth message into the
// per-provider Validate signal: catalog phases seed the provider set; a ledger
// phase with a positive failure rate marks the provider failing; a conformance
// entry with pending autofix work marks it pending. Mirrors the shape the
// previous JSON-walking joiner derived.
func selfHealthToValidateIndex(h *runsv1.SelfHealth) map[string]validateProviderStatus {
	out := map[string]validateProviderStatus{}
	if h == nil {
		return out
	}
	if cat := h.GetCatalog(); cat != nil {
		for _, ph := range cat.GetPhases() {
			if prov := strings.ToLower(ph.GetProvider()); prov != "" {
				if _, ok := out[prov]; !ok {
					out[prov] = validateProviderStatus{}
				}
			}
		}
	}
	if led := h.GetLedger(); led != nil {
		for _, ph := range led.GetPhases() {
			prov := strings.ToLower(ph.GetProvider())
			if prov == "" {
				continue
			}
			st := out[prov]
			if ph.GetFailureRate() > 0 {
				st.failing = true
			}
			out[prov] = st
		}
	}
	for _, cf := range h.GetConformance() {
		prov := strings.ToLower(cf.GetProvider())
		if prov == "" {
			continue
		}
		st := out[prov]
		if af := cf.GetAutofix(); af != nil && af.GetPending() > 0 {
			st.autofixPending = true
		}
		out[prov] = st
	}
	return out
}

// scoresToGuideIndex distills prompt-manager graph health scores into the
// node-id → score index the Guide join resolves skills against.
func scoresToGuideIndex(scores []*graphv1.HealthScore) map[string]float64 {
	out := make(map[string]float64, len(scores))
	for _, s := range scores {
		if id := strings.ToLower(s.GetNodeId()); id != "" {
			out[id] = s.GetScore()
		}
	}
	return out
}
