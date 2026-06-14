// Package findingid computes the deterministic content-hash identity for
// an ArchitectureFinding. It is the single source of truth for the
// `afid:` stable-ID algorithm, imported by BOTH producers (test-genie's
// phases) and the consumer (architecture-cartographer's migration
// tracker). Reconciliation across re-audits matches purely on this ID, so
// the algorithm MUST be byte-identical on both sides — that is why this is
// a shared helper, not duplicated code.
//
// It lives under packages/proto (not gen/, which `make clean` wipes) so it
// ships with the generated ArchitectureFinding type it operates on.
//
// Identity is (scenario, source, code, file-level locations). Line and
// column suffixes are stripped from extension-bearing file paths before
// hashing, so identity is line-shift invariant BY DESIGN: fixing one
// finding (which shifts later findings' line numbers in the same file)
// never churns those findings' IDs. Display locations elsewhere keep their
// line numbers — only the hash input is line-stripped.
package findingid

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// Prefix marks an architecture-finding stable ID.
const Prefix = "afid:"

// Inputs are the frozen hash inputs. Severity, message, suggestion, and
// domains are EXCLUDED so cosmetic changes never manufacture a false
// regression (mirrors the cartographer's `csid:` design).
type Inputs struct {
	Scenario  string
	Source    architecturev1.FindingSource
	Code      string
	Locations []string
}

// SourceToken maps a FindingSource to a short, stable lower-case token.
// Using an explicit map (rather than the generated String()) keeps the
// hash input reviewable and independent of proto-formatting quirks; adding
// a source here is a deliberate, frozen contract change. This is the single
// source of truth for the source-token vocabulary shared by the hash input,
// the campaign tracker's Finding.Source, and reaudit covered-sources.
func SourceToken(s architecturev1.FindingSource) string {
	switch s {
	case architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE:
		return "structure"
	case architecturev1.FindingSource_FINDING_SOURCE_CLI:
		return "cli"
	case architecturev1.FindingSource_FINDING_SOURCE_UI:
		return "ui"
	case architecturev1.FindingSource_FINDING_SOURCE_DOCS:
		return "docs"
	case architecturev1.FindingSource_FINDING_SOURCE_STANDARDS:
		return "standards"
	case architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE:
		return "architecture"
	case architecturev1.FindingSource_FINDING_SOURCE_TIDINESS:
		return "tidiness"
	case architecturev1.FindingSource_FINDING_SOURCE_COVERAGE:
		return "coverage"
	case architecturev1.FindingSource_FINDING_SOURCE_SECURITY:
		return "security"
	case architecturev1.FindingSource_FINDING_SOURCE_MEASURES:
		return "measures"
	case architecturev1.FindingSource_FINDING_SOURCE_BUSINESS:
		return "business"
	case architecturev1.FindingSource_FINDING_SOURCE_PROTO:
		return "proto"
	case architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY:
		return "dependency"
	default:
		return "unspecified"
	}
}

// fileLineSuffix matches an extension-bearing file path with a trailing
// `:line` or `:line:col` suffix. The extension anchor (`.go:75` yes,
// `localhost:8080` no, `/health` no, bare domain names no) is what keeps
// line-stripping from mangling URLs, host:port pairs, or route locations.
// Group 1 is the file path without the line/column suffix.
var fileLineSuffix = regexp.MustCompile(`^(.+\.[A-Za-z0-9_]+):\d+(:\d+)?$`)

// normalizeLocations returns repo-relative, forward-slashed, trimmed,
// sorted, de-duplicated locations with line/column suffixes stripped from
// extension-bearing file paths. This is the canonical form fed into the
// hash; callers need not pre-normalize. Line numbers are deliberately NOT
// part of identity: fixing one finding shifts the line numbers of later
// findings in the same file, and a stable ID must survive that drift.
// Display locations elsewhere keep their line numbers; only the hash input
// is line-stripped. A consequence: multiple findings with the same
// (source, code, file) collapse to ONE hash-input entry — i.e. one stable
// ID, one campaign work-item per file — which is the correct work-unit
// granularity (reaudit validates only when ALL instances in that file are
// gone). Dedup runs AFTER stripping so two same-file lines collapse.
func normalizeLocations(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, l := range in {
		l = strings.TrimSpace(strings.ReplaceAll(l, "\\", "/"))
		if l == "" {
			continue
		}
		if m := fileLineSuffix.FindStringSubmatch(l); m != nil {
			l = m[1]
		}
		if _, ok := seen[l]; ok {
			continue
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// Compute returns the deterministic stable ID for the given inputs:
// "afid:" + first 8 bytes (16 hex chars) of
// sha256(scenario \x1f sourceToken \x1f code \x1f sorted(locations)).
func Compute(in Inputs) string {
	locs := normalizeLocations(in.Locations)
	var b strings.Builder
	b.WriteString(in.Scenario)
	b.WriteByte('\x1f')
	b.WriteString(SourceToken(in.Source))
	b.WriteByte('\x1f')
	b.WriteString(in.Code)
	b.WriteByte('\x1f')
	b.WriteString(strings.Join(locs, ","))
	sum := sha256.Sum256([]byte(b.String()))
	return Prefix + hex.EncodeToString(sum[:8])
}

// For computes the stable ID for an ArchitectureFinding from its frozen
// hash fields. It does NOT mutate the finding.
func For(f *architecturev1.ArchitectureFinding) string {
	if f == nil {
		return ""
	}
	return Compute(Inputs{
		Scenario:  f.GetScenario(),
		Source:    f.GetSource(),
		Code:      f.GetCode(),
		Locations: f.GetLocations(),
	})
}

// Stamp computes the stable ID for the finding and writes it into the
// StableId field, returning the finding for chaining. Producers call this
// after populating scenario/source/code/locations.
func Stamp(f *architecturev1.ArchitectureFinding) *architecturev1.ArchitectureFinding {
	if f == nil {
		return nil
	}
	f.StableId = For(f)
	return f
}
