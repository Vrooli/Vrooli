package sessioncontext

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"swarm-manager/internal/agentsessions"
)

// ReferenceCandidate is a single typed entity mention extracted from agent
// output (e.g. from an `initiative:ship-cockpit` code span). Type uses the
// marker vocabulary the session skills emit (see markerContextType); Name is
// the raw reference (`<name>`, `<kind>/<name>`, or `<id>`).
type ReferenceCandidate struct {
	Type string
	Name string
}

// ResolvedReference is the verdict for one candidate. Exists is the only
// thing a consumer should trust: it is set solely by the resolver finding the
// backing record, never by the model's claim. DetailPath is populated only
// when Exists is true and the entity maps to a navigable UI route.
type ResolvedReference struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Exists     bool   `json:"exists"`
	DetailPath string `json:"detail_path,omitempty"`
	Title      string `json:"title,omitempty"`
	NodeID     string `json:"node_id,omitempty"`
}

// markerContextType maps the typed-reference marker vocabulary emitted by
// session skills to the internal ContextType the resolver understands. The
// markers are deliberately short and hyphenated (`operating-mode`) to read
// naturally in prose; the ContextType values are the storage enum.
var markerContextType = map[string]agentsessions.ContextType{
	"initiative":     agentsessions.ContextInitiative,
	"backlog":        agentsessions.ContextBacklogItem,
	"execution":      agentsessions.ContextExecution,
	"capture":        agentsessions.ContextCapture,
	"session":        agentsessions.ContextSession,
	"operating-mode": agentsessions.ContextOperatingMode,
	"scenario":       agentsessions.ContextScenario,
}

// IsKnownReferenceMarker reports whether the given marker type is one the
// resolver can resolve. Callers extracting candidates use this to discard
// unknown markers before calling ResolveBulk.
func IsKnownReferenceMarker(marker string) bool {
	_, ok := markerContextType[marker]
	return ok
}

// ResolveBulk resolves a batch of typed reference candidates to ground-truth
// verdicts. Each candidate is resolved independently via the same per-ref
// resolve() the single-context path uses, so existence is authoritative: a
// candidate naming a record that does not exist comes back Exists=false with
// no DetailPath. The method makes no network call — it is the in-process seam
// the message-append enrichment path and the EntityReferenceService Connect
// handler both delegate to.
func (r *Resolver) ResolveBulk(ctx context.Context, candidates []ReferenceCandidate, limits agentsessions.ContextLimits) []ResolvedReference {
	out := make([]ResolvedReference, 0, len(candidates))
	for _, cand := range candidates {
		out = append(out, r.resolveOne(ctx, cand, limits))
	}
	return out
}

func (r *Resolver) resolveOne(ctx context.Context, cand ReferenceCandidate, limits agentsessions.ContextLimits) ResolvedReference {
	res := ResolvedReference{Type: cand.Type, Name: cand.Name}

	contextType, known := markerContextType[cand.Type]
	if !known || strings.TrimSpace(cand.Name) == "" {
		return res // unknown marker or empty ref: Exists stays false
	}
	if contextType == agentsessions.ContextOperatingMode {
		return res // retired runtime; operating-mode references are historical only
	}

	item, err := r.resolve(ctx, agentsessions.ContextRef{Type: contextType, Ref: cand.Name}, limits)
	if err != nil {
		return res
	}

	res.Exists = true
	res.Title = item.Title
	res.NodeID = item.NodeID
	res.DetailPath = detailPathFromNodeID(item.NodeID)
	return res
}

// detailPathFromNodeID mirrors the UI's detailPathFromNodeId
// (ui/src/app/routes/route-paths.ts) so server-resolved references navigate
// to the same routes the Artifacts tab uses. Node IDs already expressed as a
// path (session, operations) pass through unchanged; unknown or
// non-navigable prefixes (agent activity) return "".
func detailPathFromNodeID(nodeID string) string {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return ""
	}
	// Resolver already emits a path for session/operations/startup briefs.
	if strings.HasPrefix(nodeID, "/") {
		return nodeID
	}
	if rest, ok := strings.CutPrefix(nodeID, "operatingMode/"); ok {
		return joinDetailPath("/operating-modes", rest)
	}
	if rest, ok := strings.CutPrefix(nodeID, "backlog-item/"); ok {
		idx := strings.Index(rest, "/")
		if idx <= 0 || idx == len(rest)-1 {
			return ""
		}
		kind, name := rest[:idx], rest[idx+1:]
		return "/backlog/" + url.PathEscape(kind) + "/" + url.PathEscape(name)
	}
	if rest, ok := strings.CutPrefix(nodeID, "scenario/"); ok {
		return joinDetailPath("/scenarios", rest)
	}
	if rest, ok := strings.CutPrefix(nodeID, "execution-record/"); ok {
		return joinDetailPath("/executions", rest)
	}
	if rest, ok := strings.CutPrefix(nodeID, "initiative/"); ok {
		return joinDetailPath("/initiatives", rest)
	}
	if rest, ok := strings.CutPrefix(nodeID, "capture/"); ok {
		return joinDetailPath("/captures", rest)
	}
	// agent-activity and any unrecognized prefix have no detail route.
	return ""
}

func joinDetailPath(base, identifier string) string {
	if identifier == "" {
		return ""
	}
	return base + "/" + url.PathEscape(identifier)
}

// inlineCodeSpanRe matches single-backtick inline code spans on one line. The
// typed-reference convention is carried exclusively inside code spans so that
// resolution never scans free prose (which is where false positives live).
var inlineCodeSpanRe = regexp.MustCompile("`([^`\n]+)`")

// referenceEnrichmentLimits bounds the per-reference resolve during message
// enrichment. Enriched ContextItems keep only Title + NodeID, so the summary
// truncation budget is irrelevant; a tiny value avoids loading large bodies.
var referenceEnrichmentLimits = agentsessions.ContextLimits{MaxSummaryRunes: 1}

// extractReferenceCandidates pulls typed `type:name` references out of inline
// code spans. The contract is deliberately strict — only marked, known-marker,
// whitespace-free spans qualify — because we never trust the model to be
// reliable: an explicit prefix is the price of navigability. Unmarked prose
// mentions and CLI commands (`initiatives list` — has a space, no marker) are
// ignored. Duplicates collapse to one candidate.
func extractReferenceCandidates(content string) []ReferenceCandidate {
	seen := make(map[string]bool)
	var out []ReferenceCandidate
	for _, match := range inlineCodeSpanRe.FindAllStringSubmatch(content, -1) {
		inner := strings.TrimSpace(match[1])
		colon := strings.Index(inner, ":")
		if colon <= 0 || colon == len(inner)-1 {
			continue
		}
		marker := inner[:colon]
		name := strings.TrimSpace(inner[colon+1:])
		if name == "" || !IsKnownReferenceMarker(marker) {
			continue
		}
		// A real entity id never contains whitespace; a span that does is
		// prose or a command, not a reference.
		if strings.ContainsAny(name, " \t\r\n") {
			continue
		}
		key := marker + ":" + name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ReferenceCandidate{Type: marker, Name: name})
	}
	return out
}

// EnrichMessageReferences extracts typed references from assistant content,
// resolves them in-process, and returns ContextItems for the survivors (the
// references that actually exist). The returned items carry the resolver's
// NodeID so the chat UI linkifies them with the same node-id→path rule the
// Artifacts tab uses. Returns nil when nothing resolves. Satisfies
// agentsessions.MessageReferenceEnricher.
func (r *Resolver) EnrichMessageReferences(ctx context.Context, content string) []agentsessions.ContextItem {
	candidates := extractReferenceCandidates(content)
	if len(candidates) == 0 {
		return nil
	}
	resolved := r.ResolveBulk(ctx, candidates, referenceEnrichmentLimits)
	now := time.Now().UTC().Format(time.RFC3339)
	var items []agentsessions.ContextItem
	for _, res := range resolved {
		if !res.Exists {
			continue
		}
		items = append(items, agentsessions.ContextItem{
			Type:       markerContextType[res.Type],
			Ref:        res.Name,
			Title:      res.Title,
			NodeID:     res.NodeID,
			SelectedAt: now,
		})
	}
	return items
}
