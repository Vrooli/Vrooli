package config

import (
	"context"
	"sort"
	"time"

	"tunnel-manager/internal/manifest"
)

// OwnershipState classifies one ingress hostname after reconciling the
// desired manifest (scenario + external routes), the live tunnel (Cloudflare
// in remote mode, config.yml in local mode), and the ownership ledger. It is
// the heart of the drift-aware control plane: every hostname gets exactly one
// state, and removal is only ever proposed for states the operator can act on
// explicitly — never applied implicitly.
type OwnershipState string

const (
	// StateUnspecified is the zero value (never emitted by reconcile).
	StateUnspecified OwnershipState = ""
	// StateManaged: a scenario-backed desired route that is live (healthy).
	StateManaged OwnershipState = "managed"
	// StateMissing: a desired route (scenario or external) that is not yet
	// live — an add candidate the next additive sync will publish.
	StateMissing OwnershipState = "missing"
	// StateExternalOK: an external desired route that is live, or a
	// ledger-EXTERNAL hostname that is live — tracked, TM owns its ingress.
	StateExternalOK OwnershipState = "external_ok"
	// StateOrphaned: a ledger-MANAGED hostname whose route no longer exists
	// in the desired set — a prune candidate (TM created it, route gone).
	StateOrphaned OwnershipState = "orphaned"
	// StateIgnored: a ledger-IGNORED hostname that is live — acknowledged as
	// external, never pushed and never pruned by reconcile.
	StateIgnored OwnershipState = "ignored"
	// StateUnmanaged: a live hostname with no desired route and no ledger
	// record — genuine drift. Surfaced; requires an explicit decision
	// (adopt / ignore / prune). Never auto-removed.
	StateUnmanaged OwnershipState = "unmanaged"
)

// RouteSource distinguishes scenario-backed desired entries from external
// ones. It mirrors the routes-domain RouteSource (Phase 3) at the reconcile
// boundary so the config domain never imports routes' wire types.
type RouteSource string

const (
	// SourceUnspecified is the zero value.
	SourceUnspecified RouteSource = ""
	// SourceScenario marks a desired entry backed by a scenario route.
	SourceScenario RouteSource = "scenario"
	// SourceExternal marks a desired entry pointing at an arbitrary target.
	SourceExternal RouteSource = "external"
)

// Owner is the ownership recorded in the ingress_ownership ledger.
type Owner string

const (
	// OwnerManaged: TM created this ingress for a managed route. When the
	// route disappears the hostname becomes ORPHANED (prune candidate).
	OwnerManaged Owner = "MANAGED"
	// OwnerExternal: an external ingress TM tracks and owns (adopted as
	// external). It reconciles as EXTERNAL_OK while live.
	OwnerExternal Owner = "EXTERNAL"
	// OwnerIgnored: an external ingress the operator acknowledged but TM must
	// never touch — never pushed, never pruned.
	OwnerIgnored Owner = "IGNORED"
)

// DesiredEntry is one entry in the desired ingress set, carrying its
// provenance so reconcile can tell scenario-backed entries (→ MANAGED) from
// external ones (→ EXTERNAL_OK).
type DesiredEntry struct {
	Hostname string
	Service  string
	Source   RouteSource
	Scenario string
	LeaseID  string
	// PublicExposure is the route's per-route override for the /public
	// Access-bypass convention (inherit|enabled|disabled). Empty = inherit.
	PublicExposure manifest.PublicExposure
}

// LedgerEntry is one persisted ownership record, keyed by full hostname.
type LedgerEntry struct {
	Hostname  string
	Owner     Owner
	Scenario  string
	Note      string
	AdoptedAt time.Time
}

// IngressEntry is a fully-classified ingress hostname for the drift view.
// It is the read-model row the GetDrift RPC, the CLI, and the UI render.
type IngressEntry struct {
	Hostname      string
	ServiceTarget string
	State         OwnershipState
	Source        RouteSource
	Scenario      string
	LeaseID       string
	Note          string
}

// DriftReport is the reconcile read model: every live/desired/tracked
// hostname classified by ownership state, plus per-state counts. It is pure
// — produced from (desired, live, ledger) with no writes.
type DriftReport struct {
	Mode    Mode
	Entries []IngressEntry
	Counts  map[OwnershipState]int
}

// OwnershipLedger is the persistence seam over the ingress_ownership table.
// Declared at the consumer per seam-discovery: production wires the
// sqlite-backed implementation from ledger_sqlite.go; service unit tests wire
// a fake. Absence of a row means UNMANAGED — the safe default.
type OwnershipLedger interface {
	// List returns every ledger record.
	List(ctx context.Context) ([]LedgerEntry, error)
	// Get returns the record for a hostname; found is false when absent.
	Get(ctx context.Context, hostname string) (entry LedgerEntry, found bool, err error)
	// Put inserts or replaces the record for a hostname (idempotent).
	Put(ctx context.Context, entry LedgerEntry) error
	// Delete removes the record for a hostname; returns false when absent.
	Delete(ctx context.Context, hostname string) (bool, error)
}

// reconcile is the pure classification function at the centre of the
// drift-aware design. It takes the desired ingress set, the live ingress, and
// the ownership ledger, and returns every distinct hostname classified by
// ownership state. It performs no I/O and no writes.
//
// The state table (see plan §5):
//
//	in desired & live                  -> MANAGED (scenario) / EXTERNAL_OK (external)
//	in desired & !live                 -> MISSING        (add candidate)
//	ledger.MANAGED & !desired          -> ORPHANED       (prune candidate)
//	ledger.EXTERNAL & live & !desired  -> EXTERNAL_OK     (tracked external)
//	ledger.IGNORED & live              -> IGNORED         (acknowledged, untouched)
//	live & !desired & !ledger          -> UNMANAGED       (drift; requires decision)
func reconcile(mode Mode, desired []DesiredEntry, live []IngressRule, ledger []LedgerEntry) DriftReport {
	desiredByHost := make(map[string]DesiredEntry, len(desired))
	for _, d := range desired {
		if d.Hostname == "" {
			continue // catch-all has no hostname
		}
		desiredByHost[d.Hostname] = d
	}
	liveByHost := make(map[string]IngressRule, len(live))
	for _, r := range live {
		if r.Hostname == "" {
			continue // catch-all has no hostname
		}
		liveByHost[r.Hostname] = r
	}
	ledgerByHost := make(map[string]LedgerEntry, len(ledger))
	for _, l := range ledger {
		if l.Hostname == "" {
			continue
		}
		ledgerByHost[l.Hostname] = l
	}

	hosts := make(map[string]struct{}, len(desiredByHost)+len(liveByHost)+len(ledgerByHost))
	for h := range desiredByHost {
		hosts[h] = struct{}{}
	}
	for h := range liveByHost {
		hosts[h] = struct{}{}
	}
	for h := range ledgerByHost {
		hosts[h] = struct{}{}
	}

	entries := make([]IngressEntry, 0, len(hosts))
	counts := make(map[OwnershipState]int, len(hosts))
	for h := range hosts {
		d, inDesired := desiredByHost[h]
		lv, inLive := liveByHost[h]
		l, inLedger := ledgerByHost[h]
		e := classifyHostname(h, d, inDesired, lv, inLive, l, inLedger)
		entries = append(entries, e)
		counts[e.State]++
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Hostname < entries[j].Hostname })
	return DriftReport{Mode: mode, Entries: entries, Counts: counts}
}

// classifyHostname assigns one hostname its ownership state and provenance from
// its presence in (desired, live, ledger). Split out of reconcile so the
// classification matrix is isolated and individually testable.
func classifyHostname(h string, d DesiredEntry, inDesired bool, lv IngressRule, inLive bool, l LedgerEntry, inLedger bool) IngressEntry {
	e := IngressEntry{Hostname: h}
	if inDesired {
		e.Source = d.Source
		e.Scenario = d.Scenario
		e.LeaseID = d.LeaseID
		e.ServiceTarget = d.Service
	}
	if e.ServiceTarget == "" && inLive {
		e.ServiceTarget = lv.Service
	}
	if inLedger {
		e.Note = l.Note
		if e.Scenario == "" {
			e.Scenario = l.Scenario
		}
	}

	switch {
	case inLedger && l.Owner == OwnerIgnored && inLive:
		e.State = StateIgnored
		e.Source = SourceExternal
	case inDesired && inLive:
		e.State = StateManaged
		if d.Source == SourceExternal {
			e.State = StateExternalOK
		}
	case inDesired && !inLive:
		e.State = StateMissing
	case inLedger && l.Owner == OwnerExternal && inLive:
		e.State = StateExternalOK
		e.Source = SourceExternal
	case inLedger && l.Owner == OwnerManaged:
		// Route gone from the desired set; TM created the ingress.
		e.State = StateOrphaned
	case inLive:
		e.State = StateUnmanaged
	default:
		// Stale ledger record (EXTERNAL/IGNORED, no longer live and not
		// desired) — a cleanup candidate, treated as orphaned.
		e.State = StateOrphaned
	}
	return e
}
