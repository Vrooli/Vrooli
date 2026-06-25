package plans

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// SourceReader is the filesystem seam for the plan-source resolver. It reads
// markdown plan sources from the hygiene-blessed fallback read locations
// (~/.vrooli/plans, <repo>/docs/plans, <repo>/plans). Production wires the
// os-backed reader; tests inject a fake. Kept narrow so the domain never imports
// path/filesystem concerns beyond a byte read.
type SourceReader interface {
	ReadFile(path string) ([]byte, error)
}

// OSSourceReader is the production SourceReader (reads from disk).
type OSSourceReader struct{}

// ReadFile reads the named file from disk.
func (OSSourceReader) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

var _ SourceReader = OSSourceReader{}

// FallbackReadLocations are the ordered fallback read/import locations the
// resolver treats as valid plan sources (the canonical write location is the
// ~/.vrooli home store owned by the repository). These are relative hints; the
// import path accepts an explicit source path. The order encodes precedence.
var FallbackReadLocations = []string{
	"~/.vrooli/plans",
	"docs/plans",
	"plans",
}

// Import adopts a markdown plan (from one of the fallback read locations) into
// the structured model and persists it through Create. When markdown is empty,
// the source is read from sourcePath via the SourceReader seam. References are
// parsed from the [CODE:]/[REQ:]/[DOC:] grammar. The fallback source is NOT
// mutated (non-destructive import — see docs/concepts/DATA.md).
func (s *service) Import(ctx context.Context, sourcePath, markdown string) (Plan, error) {
	if strings.TrimSpace(markdown) == "" {
		if strings.TrimSpace(sourcePath) == "" {
			return Plan{}, ErrInvalidPlan{Reason: "import requires markdown or a source path"}
		}
		if s.reader == nil {
			return Plan{}, ErrInvalidPlan{Reason: "no source reader configured; pass inline markdown"}
		}
		raw, err := s.reader.ReadFile(sourcePath)
		if err != nil {
			return Plan{}, fmt.Errorf("read plan source %q: %w", sourcePath, err)
		}
		markdown = string(raw)
	}
	parsed, err := ParsePlanMarkdown(markdown)
	if err != nil {
		return Plan{}, err
	}
	return s.Create(ctx, parsed)
}

// Migrate ensures a plan resolved from a fallback location is present in the
// canonical home store (idempotent re-save). Since the repository IS the
// canonical store, a plan already known to the store round-trips through a touch
// that re-affirms its canonical residence; an unknown plan is a not-found (Import
// it first). The fallback source is never destructively removed here.
func (s *service) Migrate(ctx context.Context, idOrSlug string) (Plan, error) {
	p, err := s.Get(ctx, idOrSlug)
	if err != nil {
		return Plan{}, err
	}
	p.UpdatedAt = s.now()
	if err := s.repo.Save(ctx, p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

var (
	titleRe     = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
	sectionRe   = regexp.MustCompile(`(?m)^##\s+(.+?)\s*$`)
	phaseRe     = regexp.MustCompile(`(?m)^###\s+Phase\s+(\d+)\s*[—:-]\s*(.+?)\s*$`)
	referenceRe = regexp.MustCompile(`\[(CODE|REQ|DOC):\s*([^\]]+?)\]`)
	bulletKVRe  = regexp.MustCompile(`(?m)^-\s*([A-Za-z ]+):\s*(.+?)\s*$`)
)

// ParsePlanMarkdown parses a markdown plan into the structured model. It is a
// pure, deterministic function (no I/O) so it is directly unit-testable. The
// parse is intentionally forgiving: it extracts the title, the recognized prose
// sections, the machine-readable references, and the first-class phases; unknown
// sections are ignored. The markdown view is a projection, so a parse round-trip
// is best-effort adoption, not a lossless inverse of RenderMarkdown.
func ParsePlanMarkdown(markdown string) (Plan, error) {
	if strings.TrimSpace(markdown) == "" {
		return Plan{}, ErrInvalidPlan{Reason: "empty markdown"}
	}
	var p Plan
	if m := titleRe.FindStringSubmatch(markdown); m != nil {
		p.Title = strings.TrimSpace(m[1])
	}
	if p.Title == "" {
		return Plan{}, ErrInvalidPlan{Reason: "markdown has no title heading"}
	}

	// Split out the phases region (everything from the first "## Phases" or the
	// first "### Phase" heading) so prose-section extraction doesn't swallow it.
	sections := extractSections(markdown)
	p.Purpose = sections["purpose"]
	p.Scope = sections["scope"]
	p.Constraints = sections["constraints"]
	p.NonGoals = firstNonEmpty(sections["non-goals"], sections["non goals"])
	p.DefinitionOfDone = firstNonEmpty(sections["definition of done"], sections["definition-of-done"])

	p.References = parseReferences(markdown)
	p.Phases = parsePhases(markdown)
	return p, nil
}

// extractSections maps each "## Heading" (lowercased) to the prose body up to
// the next heading of the same-or-higher level.
func extractSections(markdown string) map[string]string {
	out := map[string]string{}
	locs := sectionRe.FindAllStringSubmatchIndex(markdown, -1)
	for i, loc := range locs {
		heading := strings.ToLower(strings.TrimSpace(markdown[loc[2]:loc[3]]))
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		// Stop a section body before a phases block if one starts inside it.
		body := markdown[bodyStart:bodyEnd]
		if idx := phaseRe.FindStringIndex(body); idx != nil {
			body = body[:idx[0]]
		}
		out[heading] = strings.TrimSpace(body)
	}
	return out
}

func parseReferences(markdown string) []Reference {
	matches := referenceRe.FindAllStringSubmatch(markdown, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]Reference, 0, len(matches))
	for _, m := range matches {
		kind := referenceKindFromMarker(m[1])
		target := strings.TrimSpace(m[2])
		key := string(kind) + "|" + target
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Reference{Kind: kind, Target: target})
	}
	return out
}

func parsePhases(markdown string) []Phase {
	locs := phaseRe.FindAllStringSubmatchIndex(markdown, -1)
	if len(locs) == 0 {
		return nil
	}
	out := make([]Phase, 0, len(locs))
	for i, loc := range locs {
		order, _ := strconv.Atoi(markdown[loc[2]:loc[3]])
		title := strings.TrimSpace(markdown[loc[4]:loc[5]])
		bodyStart := loc[1]
		bodyEnd := len(markdown)
		if i+1 < len(locs) {
			bodyEnd = locs[i+1][0]
		}
		body := markdown[bodyStart:bodyEnd]
		ph := Phase{Order: order, Title: title, Status: PhaseStatusTodo}
		for _, kv := range bulletKVRe.FindAllStringSubmatch(body, -1) {
			key := strings.ToLower(strings.TrimSpace(kv[1]))
			val := strings.TrimSpace(kv[2])
			switch key {
			case "intent":
				ph.Intent = val
			case "acceptance":
				ph.Acceptance = val
			case "status":
				ph.Status = phaseStatusFromLabel(val)
			}
		}
		ph.References = parseReferences(body)
		out = append(out, ph)
	}
	return out
}

func referenceKindFromMarker(marker string) ReferenceKind {
	switch strings.ToUpper(strings.TrimSpace(marker)) {
	case "REQ":
		return ReferenceReq
	case "DOC":
		return ReferenceDoc
	default:
		return ReferenceCode
	}
}

func phaseStatusFromLabel(s string) PhaseStatus {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), "*")) {
	case "active":
		return PhaseStatusActive
	case "done":
		return PhaseStatusDone
	case "blocked":
		return PhaseStatusBlocked
	default:
		return PhaseStatusTodo
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
