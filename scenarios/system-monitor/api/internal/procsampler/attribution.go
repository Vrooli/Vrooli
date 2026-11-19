package procsampler

// DOC: docs/internal/SEAMS.md#process-attribution

import (
	"context"
	"path"
	"regexp"
	"strings"

	capacity "github.com/vrooli/vrooli/internal/capacity"
)

// OwnerUnknown is the first-class bucket for processes without a supported
// scenario, resource, or explicit host identity (for example kernel threads or
// a generic user shell). It is a normal result, never an error.
const OwnerUnknown = "unknown"

// DockerFallback resolves a containerized PID to an owner when the cheap
// host-process heuristics (cwd / binary / PPID walk) find nothing. It mirrors
// the platform internal/capacity DockerAttributor seam; system-monitor injects
// a thin adapter so this package keeps no docker dependency. It returns "" when
// the PID is not containerized.
type DockerFallback interface {
	Attribute(ctx context.Context, pid int) string
}

// Attributor maps host PIDs to the strongest available scenario, managed
// resource, or explicit host identity. The primary path anchors scenario API
// binaries and inherits their owner through ancestry, then uses the canonical
// resource normalizer and explicit local host cgroups before the cwd fallback.
// Containerized leftovers fall through to the optional Docker resolver.
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

// Attribute fills Owner on every sample in place. It runs six cheap passes,
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
//  3. Resource identity: managed host resource workers such as llama-server and
//     the native reranker use the control-plane's canonical process normalizer.
//  4. Explicit host identity: systemd app/unit/session cgroups name desktop and
//     host-service processes, and exact Vrooli agent scopes remain distinct.
//  5. Location: only for processes with no stronger owner signal do we fall
//     back to the cwd / cmdline scenarios/<name>/ path.
//  6. Docker fallback (optional): containerized leftovers; else OwnerUnknown.
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

	// Pass 3: managed host resources (Ollama, Whisper, reranker, and processes
	// living under an artifact path). Delegate aliases to the control plane so
	// resource telemetry does not drift from GPU/capacity attribution.
	for i := range samples {
		if samples[i].Owner != "" {
			continue
		}
		if owner := capacity.NormalizeProcessOwner(samples[i].Cmdline); owner != "" {
			samples[i].Owner = owner
			continue
		}
		if owner := capacity.NormalizeProcessOwner(samples[i].Comm); owner != "" {
			samples[i].Owner = owner
		}
	}

	// Pass 4: explicit local host identity. This is a named owner source, not a
	// scenario/resource guess. In particular, cgroup scope IDs make agent
	// processes individually reconcilable with `vrooli agent list`.
	for i := range samples {
		if samples[i].Owner == "" {
			samples[i].Owner = hostIdentityOwner(samples[i])
		}
	}

	// Pass 5: location fallback (cwd / cmdline path) for anything with no
	// scenario-api ancestor, managed-resource identity, or explicit host owner.
	for i := range samples {
		if samples[i].Owner != "" {
			continue
		}
		samples[i].Owner = locationOwner(samples[i])
	}

	// Pass 6: docker fallback for genuinely containerized PIDs, then normalize
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
