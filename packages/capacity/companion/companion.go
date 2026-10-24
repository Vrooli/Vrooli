// Package companion owns the one job three resource CLIs were each doing
// separately: telling the capacity broker what this resource currently holds
// and whether it is working.
//
// The observation differs per resource — ollama polls /api/ps, reranker reads
// its model state, whisper brackets each request at the proxy — and everything
// after the observation is identical: claim on the first non-zero footprint,
// resize when it moves, release when it reaches zero, heartbeat otherwise. That
// identical half lived in three hand-rolled copies with three different bugs
// available to it; it lives here instead.
//
// Two properties are load-bearing and are enforced by this package rather than
// left to each caller:
//
//   - Fail open. Ledger errors stop dependent mutations while the resource
//     keeps running. Completed mutations are not rolled back; an outage must
//     never stop inference or transcription.
//   - One claim per resource lifetime. A changed footprint resizes the existing
//     claim; it does not release and re-claim. Release-and-reclaim adds a
//     ledger row per change and discards the observed-usage history the
//     right-sizing advisory reads.
package companion

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
)

// Footprint is one observation of what a resource currently holds.
type Footprint struct {
	// Bytes is the resource's current footprint. Zero means it holds nothing,
	// which is a release.
	Bytes int64
	// Activity is "active" or "idle". Empty leaves the ledger's current
	// activity state alone: idleness is the work-owner's truth to report, and
	// an observer that cannot tell must not guess.
	Activity string
}

// Observer is the resource-specific half: look at this resource and say what it
// holds. It is the only thing a resource CLI implements.
type Observer interface {
	// Observe returns the current footprint. An error means "could not tell",
	// and the loop leaves the ledger untouched rather than assuming zero.
	Observe(ctx context.Context) (Footprint, error)
}

// ObserverFunc adapts a function to Observer.
type ObserverFunc func(ctx context.Context) (Footprint, error)

// Observe calls f.
func (f ObserverFunc) Observe(ctx context.Context) (Footprint, error) { return f(ctx) }

// Exec runs a control-plane command. It is the seam that keeps this package
// testable without a broker.
type Exec func(ctx context.Context, name string, args ...string) ([]byte, error)

// Activity states the broker understands.
const (
	ActivityActive = "active"
	ActivityIdle   = "idle"
)

// Config declares one resource's capacity companion.
type Config struct {
	// Resource is the capacity owner id.
	Resource string
	// Observer supplies the resource-specific observation.
	Observer Observer
	// Exec runs the vrooli CLI. nil is a programming error, not a runtime one.
	Exec Exec
	// Interval is the poll cadence. Zero means DefaultInterval.
	Interval time.Duration
	// ResizeThresholdBytes is how far the footprint must move before the claim
	// is resized. Small jitter in a reported size must not churn the ledger.
	// Zero means DefaultResizeThreshold.
	ResizeThresholdBytes int64
	// GPUIndex is the device the claim is against.
	GPUIndex int
	// Priority is the claim's tier name.
	Priority string
	// PreferredBytes and FloorBytes are the declared reservation ladder used
	// when the claim is first created.
	PreferredBytes int64
	FloorBytes     int64
	// Profile is the declared degrade ladder, as JSON.
	Profile string
	// YieldWhenIdle opts the claim into the idle-yield rule.
	YieldWhenIdle bool
	// IdleGrace is the dwell before an idle claim becomes reclaimable.
	IdleGrace time.Duration
	// LedgerCacheTTL bounds repeated capacity-list probes when a resource
	// performs a short reconciliation burst. Mutations invalidate the cache.
	// Zero uses the conservative two-second default.
	LedgerCacheTTL time.Duration
	// Log receives one line per ledger action. nil discards them.
	Log io.Writer
	// ParentPID makes a companion self-terminating when its owning process is
	// gone. A positive value identifies the resource process that owns it.
	ParentPID int
	// ParentAlive is injectable for deterministic parent-death tests.
	ParentAlive func(int) bool
}

// DefaultInterval is the poll cadence when none is declared.
const DefaultInterval = 20 * time.Second

// DefaultResizeThreshold is the footprint movement that justifies a resize.
// Below it the loop heartbeats instead, so reported-size jitter does not churn.
const DefaultResizeThreshold = 512 << 20

const defaultLedgerCacheTTL = 2 * time.Second

func (c Config) interval() time.Duration {
	if c.Interval > 0 {
		return c.Interval
	}
	return DefaultInterval
}

func (c Config) resizeThreshold() int64 {
	if c.ResizeThresholdBytes > 0 {
		return c.ResizeThresholdBytes
	}
	return DefaultResizeThreshold
}

// Runner drives one resource's capacity reporting.
type Runner struct {
	cfg             Config
	active          *LedgerClaim
	activeFetchedAt time.Time
}

// New builds a Runner. It validates only what would make the loop meaningless,
// because a companion that refuses to start is worse than one that reports
// nothing: the resource itself still needs to run.
func New(cfg Config) (*Runner, error) {
	if strings.TrimSpace(cfg.Resource) == "" {
		return nil, fmt.Errorf("capacity companion: resource name is required")
	}
	if cfg.Observer == nil {
		return nil, fmt.Errorf("capacity companion: an observer is required")
	}
	if cfg.Exec == nil {
		return nil, fmt.Errorf("capacity companion: an exec seam is required")
	}
	return &Runner{cfg: cfg}, nil
}

// Run polls until the context is cancelled. It returns nil on cancellation:
// being asked to stop is not a failure.
func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.interval())
	defer ticker.Stop()
	if r.parentGone() {
		r.logf("parent_gone: exiting")
		return nil
	}
	r.SyncOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if r.parentGone() {
				r.logf("parent_gone: exiting")
				return nil
			}
			r.SyncOnce(ctx)
		}
	}
}

func (r *Runner) parentGone() bool {
	if r.cfg.ParentPID <= 1 {
		return false
	}
	if r.cfg.ParentAlive != nil {
		return !r.cfg.ParentAlive(r.cfg.ParentPID)
	}
	return !platform.IsPIDRunning(r.cfg.ParentPID)
}

// SyncOnce reconciles the ledger against one observation. Errors skip dependent
// ledger actions while resource work continues.
func (r *Runner) SyncOnce(ctx context.Context) {
	footprint, err := r.cfg.Observer.Observe(ctx)
	if err != nil {
		// Could not tell. Leaving the ledger alone is the only safe answer:
		// treating "unknown" as "zero" would release a live reservation.
		r.logf("observation unavailable, leaving the ledger unchanged: %v", err)
		return
	}

	active, err := r.activeClaim(ctx)
	if err != nil {
		r.logf("ledger unavailable, skipping mutations: %v", err)
		return
	}
	if footprint.Bytes <= 0 {
		if active != nil {
			r.release(ctx, active.ClaimID)
		}
		return
	}
	switch {
	case active == nil:
		active = r.claim(ctx, footprint.Bytes)
	case abs64(active.AmountBytes-footprint.Bytes) > r.cfg.resizeThreshold():
		active = r.resize(ctx, active.ClaimID, active.Generation, footprint.Bytes)
	default:
		active = r.heartbeat(ctx, active.ClaimID, active.Generation)
	}
	if footprint.Activity != "" && active != nil {
		r.activity(ctx, active.ClaimID, active.Generation, footprint.Activity)
	}
}

// LedgerClaim is the slice of `vrooli capacity list --json` the loop reads.
type LedgerClaim struct {
	ClaimID      string `json:"claim_id"`
	OwnerID      string `json:"owner_id"`
	OwnerKind    string `json:"owner_kind"`
	ResourceKind string `json:"resource_kind"`
	GPUIndex     *int   `json:"gpu_index"`
	AmountBytes  int64  `json:"amount_bytes"`
	Generation   int64  `json:"generation"`
	Status       string `json:"status"`
}

// activeClaim distinguishes confirmed absence from an unavailable ledger.
// Only successful, well-formed, unambiguous results are cached.
func (r *Runner) activeClaim(ctx context.Context) (*LedgerClaim, error) {
	ttl := r.cfg.LedgerCacheTTL
	if ttl <= 0 {
		ttl = defaultLedgerCacheTTL
	}
	age := time.Since(r.activeFetchedAt)
	if !r.activeFetchedAt.IsZero() && age >= 0 && age < ttl {
		if r.active == nil {
			return nil, nil
		}
		copy := *r.active
		return &copy, nil
	}
	out, err := r.cfg.Exec(ctx, "vrooli", "capacity", "list", "--owner", r.cfg.Resource, "--active", "--json")
	if err != nil {
		r.invalidateActiveClaim()
		return nil, err
	}
	claim, err := selectActiveClaim(out, r.cfg.Resource, r.cfg.GPUIndex)
	if err != nil {
		r.invalidateActiveClaim()
		return nil, err
	}
	r.active = claim
	r.activeFetchedAt = time.Now()
	if claim == nil {
		return nil, nil
	}
	copy := *claim
	return &copy, nil
}

func activeStatus(status string) bool {
	return status == "granted" || status == "reserved" || status == "degraded"
}

func validClaim(c LedgerClaim) bool {
	return c.ClaimID != "" && c.OwnerID != "" && (c.OwnerKind == "resource" || c.OwnerKind == "scenario" || c.OwnerKind == "op") &&
		(c.ResourceKind == "vram" || c.ResourceKind == "ram" || c.ResourceKind == "cpu") && c.Generation > 0 &&
		(activeStatus(c.Status) || c.Status == "released" || c.Status == "expired" || c.Status == "preempted") && (c.ResourceKind != "vram" || c.GPUIndex != nil)
}
func exactResourceClaim(c LedgerClaim, resource string, gpu int) bool {
	return validClaim(c) && c.OwnerID == resource && c.OwnerKind == "resource" && c.ResourceKind == "vram" && c.GPUIndex != nil && *c.GPUIndex == gpu && activeStatus(c.Status)
}
func selectActiveClaim(out []byte, resource string, gpu int) (*LedgerClaim, error) {
	var payload struct {
		Claims *[]LedgerClaim `json:"claims"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, fmt.Errorf("invalid capacity list: %w", err)
	}
	if payload.Claims == nil {
		return nil, fmt.Errorf("capacity list lacks claims array")
	}
	var found *LedgerClaim
	for _, claim := range *payload.Claims {
		if !validClaim(claim) {
			return nil, fmt.Errorf("capacity list contains incomplete claim identity")
		}
		if exactResourceClaim(claim, resource, gpu) {
			if found != nil {
				return nil, fmt.Errorf("capacity list contains ambiguous resource/GPU claims")
			}
			copy := claim
			found = &copy
		}
	}
	return found, nil
}

func (r *Runner) invalidateActiveClaim() {
	r.active = nil
	r.activeFetchedAt = time.Time{}
}

func (r *Runner) claim(ctx context.Context, amount int64) *LedgerClaim {
	preferred := r.cfg.PreferredBytes
	if preferred <= 0 {
		preferred = amount
	}
	floor := r.cfg.FloorBytes
	if floor < 0 {
		floor = 0
	}
	args := []string{
		"capacity", "claim",
		"--owner-kind", "resource", "--owner-id", r.cfg.Resource,
		"--resource-kind", "vram",
		"--gpu-index", strconv.Itoa(r.cfg.GPUIndex),
		"--preferred", strconv.FormatInt(preferred, 10),
		"--floor", strconv.FormatInt(floor, 10),
	}
	if priority := strings.TrimSpace(r.cfg.Priority); priority != "" {
		args = append(args, "--priority", priority)
	}
	if r.cfg.YieldWhenIdle {
		args = append(args, "--yield-when-idle")
	}
	if r.cfg.IdleGrace > 0 {
		args = append(args, "--idle-grace", r.cfg.IdleGrace.String())
	}
	if profile := strings.TrimSpace(r.cfg.Profile); profile != "" {
		args = append(args, "--profile", profile)
	}
	args = append(args, "--json")
	return r.mutateClaim(ctx, "claim", "", 0, args...)
}

func (r *Runner) resize(ctx context.Context, claimID string, generation, amount int64) *LedgerClaim {
	if strings.TrimSpace(claimID) == "" || amount <= 0 {
		return nil
	}
	return r.mutateClaim(ctx, "resize", claimID, generation, "capacity", "resize",
		"--claim-id", claimID,
		"--generation", strconv.FormatInt(generation, 10),
		"--amount", strconv.FormatInt(amount, 10), "--json")
}

func (r *Runner) heartbeat(ctx context.Context, claimID string, generation int64) *LedgerClaim {
	if strings.TrimSpace(claimID) == "" {
		return nil
	}
	return r.mutateClaim(ctx, "heartbeat", claimID, generation, "capacity", "heartbeat",
		"--claim-id", claimID,
		"--generation", strconv.FormatInt(generation, 10), "--json")
}

func (r *Runner) release(ctx context.Context, claimID string) {
	if strings.TrimSpace(claimID) == "" {
		return
	}
	r.run(ctx, "release", "capacity", "release", "--claim-id", claimID, "--json")
	r.invalidateActiveClaim()
}

func (r *Runner) activity(ctx context.Context, claimID string, generation int64, state string) {
	if strings.TrimSpace(claimID) == "" {
		return
	}
	r.run(ctx, "activity", "capacity", "activity",
		"--claim-id", claimID,
		"--generation", strconv.FormatInt(generation, 10),
		"--state", state, "--json")
	r.invalidateActiveClaim()
}

// mutateClaim consumes the real JSON response so subsequent activity uses the
// current identity/generation. Unknown mutation outcomes never authorize another.
func (r *Runner) mutateClaim(ctx context.Context, action, claimID string, generation int64, args ...string) *LedgerClaim {
	defer r.invalidateActiveClaim()
	out, err := r.run(ctx, action, args...)
	if err != nil {
		return nil
	}
	var claim LedgerClaim
	if action == "claim" {
		var payload struct {
			Claim LedgerClaim `json:"claim"`
		}
		err = json.Unmarshal(out, &payload)
		claim = payload.Claim
	} else {
		err = json.Unmarshal(out, &claim)
	}
	if err != nil || !exactResourceClaim(claim, r.cfg.Resource, r.cfg.GPUIndex) || (claimID != "" && claim.ClaimID != claimID) || (action == "resize" && claim.Generation <= generation) || (action == "heartbeat" && claim.Generation != generation) {
		r.logf("%s response has no validated current claim; skipping activity", action)
		return nil
	}
	return &claim
}

// run fails open for resource work, while stopping dependent ledger actions.
func (r *Runner) run(ctx context.Context, action string, args ...string) ([]byte, error) {
	out, err := r.cfg.Exec(ctx, "vrooli", args...)
	if err != nil {
		r.logf("%s failed; skipping subsequent ledger actions: %v", action, err)
	}
	return out, err
}

func (r *Runner) logf(format string, args ...any) {
	if r.cfg.Log == nil {
		return
	}
	fmt.Fprintf(r.cfg.Log, "capacity companion (%s): "+format+"\n", append([]any{r.cfg.Resource}, args...)...)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// Reporter is the companion contract for a resource that is not a poller.
//
// whisper's activity edge is a reverse proxy: it learns that work started when
// a request arrives and that it ended when the response completes, so it has no
// footprint to poll. What it shares with the pollers is the half after the
// observation — find this resource's active claim, then act on it — and that
// half is here rather than copied into the proxy.
type Reporter struct {
	// GPUIndex selects the exact device; zero is GPU0.
	GPUIndex int
	// Resource is the capacity owner id.
	Resource string
	// Exec runs the vrooli CLI.
	Exec Exec
}

// ReportActivity sets the activity state on this resource's active claim.
//
// It is deliberately fire-and-forget: a broker outage must never affect a
// transcription, so every failure is swallowed. Activity auto-resolves the
// generation server-side under last-writer-wins, so no generation is passed.
func (r Reporter) ReportActivity(ctx context.Context, state string) {
	if r.Exec == nil || strings.TrimSpace(r.Resource) == "" {
		return
	}
	claim, ok := r.ActiveClaim(ctx)
	if !ok {
		return
	}
	_, _ = r.Exec(ctx, "vrooli", "capacity", "activity", "--claim-id", claim.ClaimID, "--state", state, "--json")
}

// ActiveClaim returns this resource's current active claim.
func (r Reporter) ActiveClaim(ctx context.Context) (LedgerClaim, bool) {
	if r.Exec == nil || strings.TrimSpace(r.Resource) == "" {
		return LedgerClaim{}, false
	}
	out, err := r.Exec(ctx, "vrooli", "capacity", "list", "--owner", r.Resource, "--active", "--json")
	if err != nil {
		return LedgerClaim{}, false
	}
	claim, err := selectActiveClaim(out, r.Resource, r.GPUIndex)
	if err != nil || claim == nil {
		return LedgerClaim{}, false
	}
	return *claim, true
}

// Heartbeat renews only an existing resource-owned admission. It never creates,
// resizes or changes activity; failures remain advisory to the request path.
func (r Reporter) Heartbeat(ctx context.Context) error {
	claim, ok := r.ActiveClaim(ctx)
	if !ok {
		return fmt.Errorf("no active admission claim for %s", r.Resource)
	}
	if claim.OwnerKind != "resource" || claim.Generation <= 0 {
		return fmt.Errorf("invalid resource admission identity for %s", r.Resource)
	}
	_, err := r.Exec(ctx, "vrooli", "capacity", "heartbeat", "--claim-id", claim.ClaimID, "--generation", strconv.FormatInt(claim.Generation, 10), "--json")
	return err
}
