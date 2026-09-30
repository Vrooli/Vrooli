package procsampler

// DOC: docs/internal/SEAMS.md#process-attribution

import (
	"context"
	"path"
	"regexp"
	"strings"
)

// OwnerUnknown is the first-class bucket for processes that belong to no
// scenario (kernel threads, the user's shell, unrelated host daemons). It is a
// normal result, never an error.
const OwnerUnknown = "unknown"

// DockerFallback resolves a containerized PID to an owner when the cheap
// host-process heuristics (cwd / binary / PPID walk) find nothing. It mirrors
// the platform internal/capacity DockerAttributor seam; system-monitor injects
// a thin adapter so this package keeps no docker dependency. It returns "" when
// the PID is not containerized.
type DockerFallback interface {
	Attribute(ctx context.Context, pid int) string
}

// Attributor maps host PIDs to the scenario that owns them. The primary path is
// the bare-host model (this deployment is not containerized): match a process's
// working directory against ".../scenarios/<name>/", or parse a "<scenario>-api"
// binary name, then walk the parent chain so children inherit their launcher's
// owner (e.g. an osv-scanner spawned by security-health attributes to
// security-health). Containerized PIDs fall through to the optional docker
// fallback. Everything else is OwnerUnknown.
type Attributor struct {
	docker DockerFallback
}

// NewAttributor builds an attributor. A nil fallback disables docker resolution
// (correct for a pure bare-host deployment).
func NewAttributor(fallback DockerFallback) *Attributor {
	return &Attributor{docker: fallback}
}

// scenarioDirRe extracts "<name>" from any path containing
// ".../scenarios/<name>/...". Names are restricted to the scenario-id charset
// so a stray "scenarios" segment deeper in a path cannot mis-match.
var scenarioDirRe = regexp.MustCompile(`(?:^|/)scenarios/([a-z0-9][a-z0-9-]*)(?:/|$)`)

// apiBinaryRe matches a "<scenario>-api" executable name (the convention every
// scenario API binary follows, e.g. "system-monitor-api").
var apiBinaryRe = regexp.MustCompile(`^([a-z0-9][a-z0-9-]*)-api$`)

// Attribute fills Owner on every sample in place. It runs four cheap passes,
// ordered so a tool is attributed to the scenario that SPAWNED it rather than to
// whatever directory it happens to run in:
//
//  1. Anchor: a "<scenario>-api" process owns itself (binary-name identity).
//  2. Ancestry: any other process inherits its nearest scenario-api ancestor —
//     this is the osv-scanner→security-health link the plan targets. A scan
//     subprocess runs with cwd inside the scenario it is SCANNING (e.g.
//     scenarios/agent-manager), so a cwd-first rule would misattribute the
//     reconcile's CPU to the innocent scanned scenarios; ancestry attributes it
//     to the security-health process that launched it.
//  3. Location: only for processes with no scenario-api ancestor (e.g. a command
//     run by hand inside a scenario dir) do we fall back to the cwd / cmdline
//     scenarios/<name>/ path.
//  4. Docker fallback (optional): containerized leftovers; else OwnerUnknown.
//
// The PID->owner map is built once per cycle and reused across all passes.
func (a *Attributor) Attribute(ctx context.Context, samples []ProcessSample) {
	byPID := make(map[int]*ProcessSample, len(samples))
	for i := range samples {
		byPID[samples[i].PID] = &samples[i]
	}

	// Pass 1: anchor scenario-api processes by their binary identity.
	anchor := make(map[int]string, len(samples))
	for i := range samples {
		if owner := ownerFromCommand(samples[i].Comm, samples[i].Cmdline); owner != "" {
			samples[i].Owner = owner
			anchor[samples[i].PID] = owner
		}
	}

	// Pass 2: a non-anchored process inherits its nearest scenario-api ancestor.
	for i := range samples {
		if samples[i].Owner != "" {
			continue
		}
		samples[i].Owner = walkToAnchor(samples[i].PID, byPID, anchor)
	}

	// Pass 3: location fallback (cwd / cmdline path) for anything with no
	// scenario-api ancestor.
	for i := range samples {
		if samples[i].Owner != "" {
			continue
		}
		samples[i].Owner = locationOwner(samples[i])
	}

	// Pass 4: docker fallback for genuinely containerized PIDs, then normalize
	// the unknown bucket so every sample has a non-empty owner.
	for i := range samples {
		if samples[i].Owner != "" {
			continue
		}
		if a.docker != nil {
			if owner := strings.TrimSpace(a.docker.Attribute(ctx, samples[i].PID)); owner != "" {
				samples[i].Owner = owner
				continue
			}
		}
		samples[i].Owner = OwnerUnknown
	}
}

// locationOwner derives an owner from a process's cwd or a scenarios/<name>/
// path embedded in its command line. Returns "" when nothing matches. This is a
// LOCATION signal (where the process runs), used only as a fallback after
// ancestry, because a spawned tool's cwd reflects what it is operating on, not
// who owns it.
func locationOwner(s ProcessSample) string {
	if m := scenarioDirRe.FindStringSubmatch(s.Cwd); m != nil {
		return m[1]
	}
	if m := scenarioDirRe.FindStringSubmatch(s.Cmdline); m != nil {
		return m[1]
	}
	return ""
}

// ownerFromCommand parses a "<scenario>-api" binary name out of the comm or the
// first cmdline token.
func ownerFromCommand(comm, cmdline string) string {
	if owner := apiBinaryOwner(comm); owner != "" {
		return owner
	}
	first := cmdline
	if i := strings.IndexByte(cmdline, ' '); i >= 0 {
		first = cmdline[:i]
	}
	return apiBinaryOwner(path.Base(first))
}

func apiBinaryOwner(name string) string {
	if m := apiBinaryRe.FindStringSubmatch(name); m != nil {
		return m[1]
	}
	return ""
}

// walkToAnchor climbs the parent chain until it reaches a process anchored to a
// scenario (a "<scenario>-api" binary), guarding against cycles and missing
// parents (PID 1 / reaped parents / processes outside the sample). Returns ""
// (not OwnerUnknown) when no scenario-api ancestor exists, so the caller can
// fall back to the location heuristic.
func walkToAnchor(pid int, byPID map[int]*ProcessSample, anchor map[int]string) string {
	seen := map[int]bool{}
	cur := pid
	for {
		if seen[cur] {
			return "" // cycle guard (should not happen on a real tree)
		}
		seen[cur] = true

		s, ok := byPID[cur]
		if !ok {
			return "" // parent exited or lives outside the sample
		}
		if owner, ok := anchor[cur]; ok && owner != "" {
			return owner
		}
		if s.PPID <= 0 || s.PPID == cur {
			return ""
		}
		cur = s.PPID
	}
}
