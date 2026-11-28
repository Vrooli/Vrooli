// Package archetype infers domain archetypes from stable structural cues. The
// inference is signal-based: each structural signal proposes a canonical
// archetype with a confidence reflecting the signal's specificity (a rare,
// specific signal like AST file-rewrite weighs more than a generic "owns an
// api/internal folder"). Corroborating signals for the same archetype raise
// confidence. Results are returned highest-confidence first (the primary), with
// the remainder as secondary archetypes.
package archetype

import (
	"sort"
	"strings"
)

// Input carries the structural signals available about a domain. Name and Paths
// are always present; the code signals are optional (zero value = not
// evaluated) and supplied by callers that have a graph snapshot.
type Input struct {
	Name  string
	Paths []string

	// HasHTTPHandler is true when the domain owns an api/handlers/<domain>
	// transport package (a served capability).
	HasHTTPHandler bool
	// OwnsStorage is true when the domain owns persistent storage (SQLite, a
	// store/repository, migrations).
	OwnsStorage bool
	// WritesFiles is true when the domain performs file writes or AST rewrites
	// (the apply/mutation shape).
	WritesFiles bool
	// HasWorkflow is true when the domain imports workflow/state-machine/temporal
	// coordination machinery.
	HasWorkflow bool
	// CallsExternal is true when the domain imports outbound HTTP/integration
	// clients (an integration service).
	CallsExternal bool

	// SourcePaths identify snapshot files that support the corresponding
	// structural signal. Name/path heuristics remain document-backed and do not
	// populate these fields.
	HTTPHandlerPaths []string
	StoragePaths     []string
	MutationPaths    []string
	WorkflowPaths    []string
	ExternalPaths    []string
}

// Result is one inferred archetype with its confidence and evidence.
type Result struct {
	Name             string
	Confidence       float64
	Evidence         []string
	SourceEvidence   []SourceEvidence
	DocumentEvidence []string
}

type SourceEvidence struct {
	Path string
	Note string
}

type signal struct {
	archetype   string
	confidence  float64 // specificity weight
	evidence    string
	sourcePaths []string
	document    bool
}

// Infer returns the inferred archetypes for a domain, highest confidence first.
// Empty when no signal fires.
func Infer(in Input) []Result {
	name := strings.ToLower(strings.TrimSpace(in.Name))
	paths := lowerPaths(in.Paths)

	var signals []signal

	// Specific code signals (high specificity).
	if in.WritesFiles {
		signals = append(signals, signal{archetype: string(Mutation), confidence: 0.9, evidence: "performs file writes / AST rewrites", sourcePaths: in.MutationPaths})
	}
	if isMutationDomain(name, paths) {
		signals = append(signals, signal{archetype: string(Mutation), confidence: 0.9, evidence: "domain name/path indicates write-side apply workflow", document: true})
	}
	if in.HasWorkflow {
		signals = append(signals, signal{archetype: string(Orchestration), confidence: 0.85, evidence: "imports workflow/state-machine coordination machinery", sourcePaths: in.WorkflowPaths})
	}
	if in.CallsExternal {
		signals = append(signals, signal{archetype: string(Service), confidence: 0.65, evidence: "imports outbound integration clients", sourcePaths: in.ExternalPaths})
	}
	if in.OwnsStorage {
		signals = append(signals, signal{archetype: string(Service), confidence: 0.7, evidence: "owns persistent storage", sourcePaths: in.StoragePaths})
	}

	// Name/path heuristics (medium specificity).
	if isReportingDomain(name, paths) {
		signals = append(signals, signal{archetype: string(Reporting), confidence: 0.85, evidence: "domain name/path indicates read/reporting surface", document: true})
	}

	// Generic structural signals (low specificity).
	if in.HasHTTPHandler {
		signals = append(signals, signal{archetype: string(Service), confidence: 0.6, evidence: "owns an api/handlers transport package (served capability)", sourcePaths: in.HTTPHandlerPaths})
	}
	if hasAPIDomainPath(paths) {
		signals = append(signals, signal{archetype: string(Service), confidence: 0.6, evidence: "domain owns api/internal implementation path", document: true})
	}

	return mergeSignals(signals)
}

// mergeSignals collapses signals by archetype: confidence is the strongest
// contributing signal, bumped for each corroborating signal, and evidence is the
// union. Results are sorted by confidence (desc), then archetype name for
// determinism.
func mergeSignals(signals []signal) []Result {
	byArchetype := map[string]*Result{}
	counts := map[string]int{}
	order := []string{}
	for _, s := range signals {
		r, ok := byArchetype[s.archetype]
		if !ok {
			r = &Result{Name: s.archetype}
			byArchetype[s.archetype] = r
			order = append(order, s.archetype)
		}
		if s.confidence > r.Confidence {
			r.Confidence = s.confidence
		}
		r.Evidence = append(r.Evidence, s.evidence)
		if s.document {
			r.DocumentEvidence = append(r.DocumentEvidence, s.evidence)
		}
		counts[s.archetype]++
		for _, path := range s.sourcePaths {
			path = strings.TrimSpace(path)
			if path == "" || hasSourceEvidence(r.SourceEvidence, path, s.evidence) {
				continue
			}
			r.SourceEvidence = append(r.SourceEvidence, SourceEvidence{Path: path, Note: s.evidence})
		}
	}
	out := make([]Result, 0, len(order))
	for _, a := range order {
		r := byArchetype[a]
		if n := counts[a]; n > 1 {
			r.Confidence = min(0.98, r.Confidence+0.05*float64(n-1)) // corroboration boost
		}
		out = append(out, *r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].Name < out[j].Name
	})
	for i := range out {
		sort.Slice(out[i].SourceEvidence, func(a, b int) bool {
			if out[i].SourceEvidence[a].Path == out[i].SourceEvidence[b].Path {
				return out[i].SourceEvidence[a].Note < out[i].SourceEvidence[b].Note
			}
			return out[i].SourceEvidence[a].Path < out[i].SourceEvidence[b].Path
		})
	}
	return out
}

func hasSourceEvidence(evidence []SourceEvidence, path, note string) bool {
	for _, candidate := range evidence {
		if candidate.Path == path && candidate.Note == note {
			return true
		}
	}
	return false
}

func isMutationDomain(name string, paths []string) bool {
	if name == "apply" {
		return true
	}
	for _, path := range paths {
		if strings.Contains(path, "/apply/") || strings.HasSuffix(path, "/apply") {
			return true
		}
	}
	return false
}

func isReportingDomain(name string, paths []string) bool {
	switch name {
	case "analytics", "health", "slice":
		return true
	}
	for _, path := range paths {
		if strings.Contains(path, "/analytics/") || strings.Contains(path, "/health/") {
			return true
		}
	}
	return false
}

func hasAPIDomainPath(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, "api/internal/") || strings.Contains(path, "/api/internal/") {
			return true
		}
	}
	return false
}

func lowerPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.ToLower(strings.Trim(strings.TrimSpace(path), "`"))
		if path != "" {
			out = append(out, strings.Trim(path, "/"))
		}
	}
	return out
}
