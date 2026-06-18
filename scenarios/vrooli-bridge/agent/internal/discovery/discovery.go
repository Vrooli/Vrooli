// Package discovery resolves the control-plane URL the node-agent dials out to
// (OT-P1-006). The agent is a DIAL-OUT node (DECISIONS.md: nodes open the
// channel to the control plane, never the reverse), so the control-plane URL is
// the one piece of configuration the bootstrap must learn. Two paths supply it:
//
//   - The MANUAL URL+code path is the cross-network DEFAULT — an operator pastes
//     the control-plane URL (plus a pairing code, handled by Phase 2) once. It
//     works across any network, including off-LAN, and depends on nothing but the
//     value the operator typed.
//   - mDNS LAN auto-discovery is a CONVENIENCE on a trusted LAN: the control
//     plane advertises `_vrooli-bridge._tcp.local` and the agent can browse for
//     it so the operator need not paste a URL at all.
//
// Resolve enforces the resolution order: an explicit/manual URL always wins (so
// off-LAN bootstrap never depends on mDNS and the Browser is never even
// invoked); otherwise the agent tries an mDNS browse; if mDNS finds nothing it
// returns a clean "not found, supply a manual URL" outcome rather than an error
// spam, leaving the manual path as the fallback.
//
// The Browser interface is the seam: production wires the dependency-free mDNS
// querier (mdns.go); tests substitute a fake instance source.
package discovery

import (
	"context"
	"strings"
	"time"
)

// defaultServiceType is the DNS-SD service the control plane advertises on a
// trusted LAN. The agent browses for PTR records under this name.
const defaultServiceType = "_vrooli-bridge._tcp.local"

// defaultBrowseTimeout bounds a single mDNS browse. Discovery is a best-effort
// convenience, so the agent waits only briefly before falling back to the
// manual URL path.
const defaultBrowseTimeout = 2 * time.Second

// ServiceInstance is one advertised control-plane endpoint discovered on the
// LAN. URL is the dial-out base URL the agent would use; Host and Port carry the
// resolved A/SRV record for diagnostics.
type ServiceInstance struct {
	Host string
	Port int
	URL  string
}

// Browser browses the LAN for advertised instances of a DNS-SD service within
// timeout. It is the discovery seam: production wires the mDNS querier
// (MDNSBrowser); tests substitute a fake that returns synthetic instances (or an
// empty result to exercise the manual-URL fallback). A browse that finds nothing
// returns an empty slice and a nil error — "no advertised control plane" is a
// normal outcome, not a failure.
type Browser interface {
	Browse(ctx context.Context, service string, timeout time.Duration) ([]ServiceInstance, error)
}

// Source distinguishes how Resolve obtained the control-plane URL, so the caller
// can log the path taken (manual vs. discovered) and decide whether to prompt.
type Source int

const (
	// SourceNone means no URL was resolved: mDNS found nothing and no manual URL
	// was supplied. The caller should fall back to prompting for a manual URL.
	SourceNone Source = iota
	// SourceManual means the explicit/manual URL was used (the Browser was never
	// invoked).
	SourceManual
	// SourceDiscovered means the URL came from an mDNS LAN browse.
	SourceDiscovered
)

// String renders the source for human-facing logs.
func (s Source) String() string {
	switch s {
	case SourceManual:
		return "manual"
	case SourceDiscovered:
		return "discovered"
	default:
		return "none"
	}
}

// Result is the outcome of Resolve. URL is empty exactly when Source is
// SourceNone (mDNS found nothing and no manual URL was given), which the caller
// treats as "use the manual URL path".
type Result struct {
	URL    string
	Source Source
	// Instance is the full advertised record when Source is SourceDiscovered;
	// the zero value otherwise.
	Instance ServiceInstance
}

// Found reports whether a control-plane URL was resolved (manual or discovered).
func (r Result) Found() bool { return r.Source != SourceNone }

// Resolve determines the control-plane URL to dial.
//
// Order:
//  1. A non-empty manualURL ALWAYS wins — it is returned immediately and the
//     Browser is never invoked, so off-LAN bootstrap never depends on mDNS.
//  2. Otherwise, if browser is non-nil, the agent browses the LAN for the
//     advertised control plane and returns the first instance carrying a usable
//     URL.
//  3. If mDNS finds nothing (or the browse errors at the transport level), it
//     returns a clean SourceNone result — the caller falls back to the manual
//     URL path. A transport error is returned to the caller for optional logging
//     but is NOT treated as fatal: the manual path remains available.
func Resolve(ctx context.Context, manualURL string, browser Browser, timeout time.Duration) (Result, error) {
	if u := strings.TrimSpace(manualURL); u != "" {
		return Result{URL: u, Source: SourceManual}, nil
	}
	if browser == nil {
		return Result{Source: SourceNone}, nil
	}
	if timeout <= 0 {
		timeout = defaultBrowseTimeout
	}

	instances, err := browser.Browse(ctx, defaultServiceType, timeout)
	if err != nil {
		// A browse that fails at the transport level (e.g. no network, no
		// multicast route) is not fatal: report it for optional logging but fall
		// back cleanly to the manual URL path.
		return Result{Source: SourceNone}, err
	}

	for _, inst := range instances {
		if strings.TrimSpace(inst.URL) != "" {
			return Result{URL: inst.URL, Source: SourceDiscovered, Instance: inst}, nil
		}
	}
	return Result{Source: SourceNone}, nil
}
