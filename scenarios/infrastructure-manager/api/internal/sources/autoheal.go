package sources

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions/actions_v1connect"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
	healingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/healing"
	healingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/healing/healing_v1connect"
)

// AutohealReader reads only persisted, typed autoheal observations. It does
// not assign trust or band verdicts; the condition domain owns those rules.
type AutohealReader struct {
	Resolver    *discovery.Resolver
	HTTP        *http.Client
	Projection  string
	WindowHours int32
	// Qualifiers is an optional cache shared by every projection reader. The
	// qualifier facts are registry-wide, not projection-specific, so reading
	// them once per request cycle instead of once per projection keeps a
	// multi-projection read from paying the core-set subprocess repeatedly.
	Qualifiers *QualifierCache
}

// QualifierCache holds one registry-wide qualifier read for a short window.
// It is safe for concurrent use and collapses concurrent misses into a single
// upstream fetch.
type QualifierCache struct {
	TTL time.Duration

	mu        sync.Mutex
	value     checkQualifiers
	fetchedAt time.Time
	loaded    bool
}

func NewQualifierCache(ttl time.Duration) *QualifierCache {
	if ttl <= 0 {
		ttl = 5 * time.Second
	}
	return &QualifierCache{TTL: ttl}
}

func (c *QualifierCache) get(now time.Time, fetch func() checkQualifiers) checkQualifiers {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loaded && now.Sub(c.fetchedAt) < c.TTL {
		return c.value
	}
	c.value, c.fetchedAt, c.loaded = fetch(), now, true
	return c.value
}

// substrateSensors is the numerator join for the substrate projection: which
// autoheal checks answer each authored cell. The denominator — which cells
// exist — is owned by vrooli-autoheal in docs/spaces/substrate-space.md; this
// table only says how to read them, and a cell with no entry here is not
// fabricated as healthy.
//
// Six of the thirteen authored cells are deliberately absent, for two different
// reasons, and neither absence means the cell is well:
//
//   - SB6 (userspace crash accounting) is MISSING. No core-dump sensor exists
//     anywhere in the platform, so there is nothing to name here.
//   - SB9-SB13 (device identity, storage predictive health, thermal, memory
//     errors, network interface health) are IN-REACH. Their sensor is shipped
//     and emitting, but it is system-monitor's device graph, which grades on a
//     rung ladder rather than the check status this table maps to a severity.
//     Joining them means a second substrate source with its own unit mapping,
//     not a new line in this map — adding one here would compare a rung against
//     a severity bar.
//
// Do not renumber: SB7 and SB8 are cited by cell_ref in the operator setpoint.
var substrateSensors = map[string][]string{
	"substrate/SB1": {"host-kernel-error-signals"},
	"substrate/SB2": {"system-panic-evidence", "system-pstore-evidence"},
	"substrate/SB3": {"system-mce-recent"},
	"substrate/SB4": {"system-gpu", "resource-gpu-access"},
	"substrate/SB5": {"host-kernel-module-drift", "host-device-driver-binding", "host-runtime-integrity"},
	"substrate/SB7": {"system-boot-history"},
	"substrate/SB8": {"os-watchdog", "host-capability-drift"},
}

// severityOf projects a check status onto an ordered severity so a substrate
// reading can be graded against a numeric bar. Availability is a percentage;
// a device fault is not, and flattening one into the other is what made a GPU
// fault readable only as a dip in an uptime figure.
func severityOf(status checksv1.CheckStatus) (float64, bool) {
	switch status {
	case checksv1.CheckStatus_CHECK_STATUS_OK:
		return 0, true
	case checksv1.CheckStatus_CHECK_STATUS_WARNING:
		return 1, true
	case checksv1.CheckStatus_CHECK_STATUS_CRITICAL:
		return 2, true
	default:
		// UNSPECIFIED and NOT_APPLICABLE carry no severity. They must not be
		// read as OK.
		return 0, false
	}
}

func (r AutohealReader) clients(ctx context.Context) (actionsconnect.ActionsServiceClient, checksconnect.ChecksServiceClient, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
	if err != nil {
		return nil, nil, err
	}
	return actionsconnect.NewActionsServiceClient(httpClient, base),
		checksconnect.NewChecksServiceClient(httpClient, base), nil
}

func (r AutohealReader) readiness(ctx context.Context) (*healingv1.GetReadinessResponse, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
	if err != nil {
		return nil, err
	}
	client := healingconnect.NewHealingServiceClient(httpClient, base)
	response, err := client.GetReadiness(ctx, connect.NewRequest(&healingv1.GetReadinessRequest{Limit: 200}))
	if err != nil {
		return nil, err
	}
	if response == nil || response.Msg == nil || !response.Msg.GetAvailable() {
		reason := "readiness source unavailable"
		if response != nil && response.Msg != nil && response.Msg.GetUnavailableReason() != "" {
			reason = response.Msg.GetUnavailableReason()
		}
		return nil, fmt.Errorf("%s", reason)
	}
	return response.Msg, nil
}

func (r AutohealReader) window() int32 {
	if r.WindowHours > 0 {
		return r.WindowHours
	}
	return 24
}

// checkQualifiers holds the per-check facts that decide trust. Each is fetched
// with one request for the whole registry, so the number of round trips does
// not grow with the number of checks.
//
// Failures are tracked per qualifier rather than as one flag. A qualifier that
// could not be read makes only the readings it actually governs untrusted:
// collapsing them into a single flag meant one slow or wedged qualifier
// invalidated the entire surface, which is the same failure — a transport
// problem reported as a verdict about the plant — one layer up.
type checkQualifiers struct {
	ghosts     map[string]bool
	outOfScope map[string]bool
	shelved    map[string]bool
	saturated  map[string]bool
	// reconcileReason is set when ghost classification is unavailable. It only
	// affects scenario checks, because reconcile classifies nothing else.
	reconcileReason string
	// universalReason is set when a qualifier that governs every check could
	// not be read.
	universalReason string
}

// qualifiers fetches the three qualifier reads concurrently. They are
// independent, and one of them — reconcile — derives the core set through a
// subprocess that routinely takes several seconds. Run in sequence it consumed
// most of the caller's per-source budget and pushed the cheap reads past their
// deadline, so a four-second subprocess decided the trust verdict of every
// check on the host.
func (r AutohealReader) qualifiers(ctx context.Context, checks checksconnect.ChecksServiceClient) checkQualifiers {
	if r.Qualifiers != nil {
		return r.Qualifiers.get(time.Now(), func() checkQualifiers { return r.fetchQualifiers(ctx, checks) })
	}
	return r.fetchQualifiers(ctx, checks)
}

func (r AutohealReader) fetchQualifiers(ctx context.Context, checks checksconnect.ChecksServiceClient) checkQualifiers {
	result := checkQualifiers{
		ghosts:     map[string]bool{},
		outOfScope: map[string]bool{},
		shelved:    map[string]bool{},
		saturated:  map[string]bool{},
	}
	var (
		mu   sync.Mutex
		wait sync.WaitGroup
	)
	wait.Add(3)

	go func() {
		defer wait.Done()
		reconcile, reconcileErr := checks.GetReconcile(ctx, connect.NewRequest(&checksv1.GetReconcileRequest{}))
		mu.Lock()
		defer mu.Unlock()
		switch {
		case reconcileErr != nil:
			result.reconcileReason = fmt.Sprintf("check reconcile is unreadable: %v", reconcileErr)
		case !reconcile.Msg.GetReconcile().GetAvailable():
			result.reconcileReason = "check reconcile is unavailable: " + reconcile.Msg.GetReconcile().GetUnavailableReason()
		default:
			// Only honour ghosts when the owner could actually determine
			// existence. Otherwise a check is left unclassified rather than
			// being dropped from every aggregate on a guess.
			if reconcile.Msg.GetReconcile().GetGhostDetectionAvailable() {
				for _, id := range reconcile.Msg.GetReconcile().GetGhostCheckIds() {
					result.ghosts[id] = true
				}
			} else {
				result.reconcileReason = "ghost detection is unavailable: " + reconcile.Msg.GetReconcile().GetGhostUnavailableReason()
			}
			// Out-of-scope checks target something real. They are recorded so
			// the supervision projection can report scope, and deliberately do
			// NOT affect trust.
			for _, id := range reconcile.Msg.GetReconcile().GetOutOfScopeCheckIds() {
				result.outOfScope[id] = true
			}
		}
	}()

	go func() {
		defer wait.Done()
		shelves, shelvesErr := checks.ListShelves(ctx, connect.NewRequest(&checksv1.ListShelvesRequest{}))
		mu.Lock()
		defer mu.Unlock()
		if shelvesErr != nil {
			result.universalReason = fmt.Sprintf("check shelves are unreadable: %v", shelvesErr)
			return
		}
		for _, shelf := range shelves.Msg.GetShelves() {
			result.shelved[shelf.GetCheckId()] = true
		}
	}()

	go func() {
		defer wait.Done()
		saturation, saturationErr := checks.ListSaturation(ctx, connect.NewRequest(&checksv1.ListSaturationRequest{WindowHours: r.window()}))
		mu.Lock()
		defer mu.Unlock()
		switch {
		case saturationErr != nil:
			result.universalReason = fmt.Sprintf("check saturation is unreadable: %v", saturationErr)
		case saturation.Msg.GetTruncated():
			result.universalReason = "check saturation tally was truncated; a capped window cannot distinguish no-transition from not-read"
		default:
			// Use the owner's `saturated` verdict, not `!transitioned`. A check
			// steady at OK has also not transitioned and is simply healthy.
			for _, item := range saturation.Msg.GetSaturations() {
				if item.GetSaturated() {
					result.saturated[item.GetCheckId()] = true
				}
			}
		}
	}()

	wait.Wait()
	return result
}

func (q checkQualifiers) hintsFor(checkID string) TrustHints {
	hints := TrustHints{
		Ghost:       q.ghosts[checkID],
		Shelved:     q.shelved[checkID],
		Saturated:   q.saturated[checkID],
		UnitMatches: true,
		OutOfScope:  q.outOfScope[checkID],
	}
	switch {
	case q.universalReason != "":
		hints.Untrusted, hints.UntrustedReason = true, q.universalReason
	// Reconcile classifies scenario checks and nothing else, so an unreadable
	// reconcile says nothing about a host, system, infra or resource check.
	// Marking those untrusted too would let a flaky core-set derivation blank
	// out the substrate readings, which are the innermost cascade tier.
	case q.reconcileReason != "" && strings.HasPrefix(checkID, "scenario-"):
		hints.Untrusted, hints.UntrustedReason = true, q.reconcileReason
	}
	return hints
}

func (r AutohealReader) Read(ctx context.Context) ([]Observation, error) {
	switch r.Projection {
	case "availability":
		return r.readAvailability(ctx)
	case "recovery":
		return r.readRecovery(ctx)
	case "substrate":
		return r.readSubstrate(ctx)
	default:
		return nil, nil
	}
}

func (r AutohealReader) readAvailability(ctx context.Context) ([]Observation, error) {
	actions, checksClient, err := r.clients(ctx)
	if err != nil {
		return nil, err
	}
	trends, err := actions.GetPerCheckTrends(ctx, connect.NewRequest(&actionsv1.GetPerCheckTrendsRequest{WindowHours: r.window()}))
	if err != nil {
		return nil, err
	}
	qualifiers := r.qualifiers(ctx, checksClient)
	checkedAt := time.Now().UTC()
	out := make([]Observation, 0, len(trends.Msg.GetTrends()))
	for _, trend := range trends.Msg.GetTrends() {
		observedAt := checkedAt
		if trend.GetLastChecked() != nil {
			observedAt = trend.GetLastChecked().AsTime()
		}
		out = append(out, Observation{
			ID: trend.GetCheckId(), CellRef: "availability/A1", Value: trend.GetUptimePercent(), Unit: "percent",
			Source: "vrooli-autoheal/actions.GetPerCheckTrends", ObservedAt: observedAt,
			TrustHints: qualifiers.hintsFor(trend.GetCheckId()),
		})
	}
	readiness, err := r.readiness(ctx)
	if err != nil {
		return nil, err
	}
	for _, element := range readiness.GetElements() {
		observedAt := checkedAt
		if element.GetReadyAt() != nil {
			observedAt = element.GetReadyAt().AsTime()
		}
		hints := qualifiers.hintsFor(element.GetCheckId())
		if element.GetLatencyMs() < 0 {
			hints.Untrusted = true
			hints.UntrustedReason = element.GetEvidence()
		}
		out = append(out, Observation{ID: element.GetCheckId(), CellRef: "availability/A3", Value: float64(element.GetLatencyMs()), Unit: "ms", Source: "vrooli-autoheal/healing.GetReadiness", ObservedAt: observedAt, TrustHints: hints})
	}
	return out, nil
}

// readSubstrate reads host, kernel and device condition as an ordered severity
// per authored cell. A cell backed by several checks takes the worst severity
// among them, because a subsystem is only as healthy as its unhealthiest
// sensor; the contributing check is named on the reading so the verdict points
// at a signal rather than at a projection.
func (r AutohealReader) readSubstrate(ctx context.Context) ([]Observation, error) {
	actions, checksClient, err := r.clients(ctx)
	if err != nil {
		return nil, err
	}
	trends, err := actions.GetPerCheckTrends(ctx, connect.NewRequest(&actionsv1.GetPerCheckTrendsRequest{WindowHours: r.window()}))
	if err != nil {
		return nil, err
	}
	byCheck := make(map[string]*actionsv1.PerCheckTrend, len(trends.Msg.GetTrends()))
	for _, trend := range trends.Msg.GetTrends() {
		byCheck[trend.GetCheckId()] = trend
	}
	qualifiers := r.qualifiers(ctx, checksClient)
	checkedAt := time.Now().UTC()

	cellRefs := make([]string, 0, len(substrateSensors))
	for cellRef := range substrateSensors {
		cellRefs = append(cellRefs, cellRef)
	}
	sort.Strings(cellRefs)

	out := make([]Observation, 0, len(cellRefs))
	for _, cellRef := range cellRefs {
		worst, worstCheck, observedAt := -1.0, "", time.Time{}
		var hints TrustHints
		missing := make([]string, 0)
		for _, checkID := range substrateSensors[cellRef] {
			trend, ok := byCheck[checkID]
			if !ok {
				missing = append(missing, checkID)
				continue
			}
			severity, readable := severityOf(trend.GetCurrentStatus())
			if !readable {
				missing = append(missing, checkID)
				continue
			}
			if severity > worst {
				worst, worstCheck, hints = severity, checkID, qualifiers.hintsFor(checkID)
				if trend.GetLastChecked() != nil {
					observedAt = trend.GetLastChecked().AsTime()
				}
			}
		}
		if worstCheck == "" {
			// No contributing check produced a readable status. Report the
			// cell as unreadable rather than omitting it, so a substrate blind
			// spot is visible instead of silently absent.
			out = append(out, Observation{
				ID: cellRef, CellRef: cellRef, Unit: "severity",
				Source: "vrooli-autoheal/actions.GetPerCheckTrends", ObservedAt: checkedAt,
				TrustHints: TrustHints{
					UnitMatches:     true,
					Untrusted:       true,
					UntrustedReason: fmt.Sprintf("no readable status for %v", substrateSensors[cellRef]),
				},
			})
			continue
		}
		if len(missing) > 0 && hints.UntrustedReason == "" {
			hints.Untrusted = true
			hints.UntrustedReason = fmt.Sprintf("partial substrate coverage; unreadable checks: %v", missing)
		}
		if observedAt.IsZero() {
			observedAt = checkedAt
		}
		out = append(out, Observation{
			ID: worstCheck, CellRef: cellRef, Value: worst, Unit: "severity",
			Source: "vrooli-autoheal/actions.GetPerCheckTrends", ObservedAt: observedAt, TrustHints: hints,
		})
	}
	return out, nil
}

func (r AutohealReader) readRecovery(ctx context.Context) ([]Observation, error) {
	readiness, err := r.readiness(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Observation, 0, len(readiness.GetEpisodes()))
	for _, episode := range readiness.GetEpisodes() {
		started, completed := episode.GetStartedAt(), episode.GetCompletedAt()
		if started == nil || completed == nil {
			continue
		}
		out = append(out, Observation{ID: episode.GetId(), CellRef: "recovery/R3", Value: float64(completed.AsTime().Sub(started.AsTime()).Milliseconds()), Unit: "ms", Source: "vrooli-autoheal/healing.GetReadiness", ObservedAt: completed.AsTime(), TrustHints: TrustHints{UnitMatches: true}})
	}
	return out, nil
}
