package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// PlanWriter is the write seam onto the plans SSOT. Finalize maps the session's
// sections into a plans.Plan and persists it through this seam. Production wraps
// the plans domain Service (a CreatePlan adapter in the handler module, mirroring
// validation's planAdapter); tests inject a fake. authoring never owns the plan
// record — it only composes and writes it.
type PlanWriter interface {
	CreatePlan(ctx context.Context, p planmodel.Plan) (planmodel.Plan, error)
}

// PlanRenderer renders an in-progress plan to its markdown review artifact so the
// wizard can offer a render-preview before finalize. Production wires
// plans.RenderMarkdown; a nil renderer disables preview (the service returns a
// typed "preview unavailable" error). The renderer is pure — no persistence.
type PlanRenderer interface {
	Render(p planmodel.Plan) string
}

// AnchorAutofiller captures the regression anchor for a plan-in-progress. The
// production impl shells git-control-tower (LookPath-guarded, timeout-bounded);
// a nil seam or an error degrades the regression-anchor section to "left for the
// author" (AutofillResult.Degraded=true) — never a fabricated anchor.
type AnchorAutofiller interface {
	// Anchor returns the prose to fill the regression-anchor section for the
	// given plan title/slug. An error degrades that section honestly.
	Anchor(ctx context.Context, title, slug string) (string, error)
}

// RequiredReadingSource is retained for explicit legacy migration input. New
// setup guidance is discovered through ContextDiscoverer and accepted as typed
// relevant context.
type RequiredReadingSource interface {
	// RequiredReading returns legacy lines that can be migrated into structured
	// relevant context when explicitly requested.
	RequiredReading(ctx context.Context, title string) (string, error)
}

// ReferenceExtractor extracts code references via code-facts. A nil seam or an
// error degrades the references section honestly.
type ReferenceExtractor interface {
	// References returns the prose to fill the references section for the given
	// plan title/scope. An error degrades that section honestly.
	References(ctx context.Context, title, scope string) (string, error)
}

// ContextDiscoverer proposes relevant-context setup items from decomposed
// concepts. It discovers candidates only; acceptance remains an authoring
// decision in the Service.
type ContextDiscoverer interface {
	DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) ([]ContextCandidate, error)
}

// CommandRunner is the exec seam shared by the production autofill sources (the
// live dispatch path to git-control-tower / prompt-manager / code-facts).
// Production wires execRunner (LookPath-guarded, timeout-bounded); tests inject a
// fake. A nil runner means "no live dispatch" — the source degrades honestly.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunner returns the production CommandRunner (LookPath-guarded,
// timeout-bounded). Wired in the handler module; tests inject a fake instead.
func DefaultRunner() CommandRunner { return execRunner }

// autofillTimeout bounds a single autofill dispatch. Generous (a discovery or
// anchor capture can shell out) but finite so a hung command cannot block the
// wizard forever.
const autofillTimeout = 2 * time.Minute

// execRunner is the production CommandRunner. It guards against running an
// arbitrary binary by requiring the command to be on PATH, bounds the call with
// a timeout, and returns combined output. Never fabricates results: a failure is
// a real error the caller turns into a degraded autofill honestly.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("command %q not found on PATH: %w", name, err)
	}
	ctx, cancel := context.WithTimeout(ctx, autofillTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

// --- production autofill sources (CommandRunner-backed, all degradable) ---

// cmdAnchorAutofiller captures the regression anchor by shelling git-control-tower
// through the CommandRunner seam. A nil runner or a command failure degrades.
type cmdAnchorAutofiller struct{ run CommandRunner }

// NewCommandAnchorAutofiller wires the production AnchorAutofiller over the given
// CommandRunner (git-control-tower). A nil runner yields a source that always
// degrades (honest, never a false fill).
func NewCommandAnchorAutofiller(run CommandRunner) AnchorAutofiller {
	return cmdAnchorAutofiller{run: run}
}

func (a cmdAnchorAutofiller) Anchor(ctx context.Context, title, slug string) (string, error) {
	if a.run == nil {
		return "", fmt.Errorf("git-control-tower unavailable")
	}
	scenario, name := anchorScenarioAndName(title, slug)
	if scenario == "" || name == "" {
		return "", fmt.Errorf("scenario and baseline name are required")
	}
	out, err := a.run(ctx, "git-control-tower", "baseline", "snapshot", "status", "--scenario", scenario, "--name", name, "--json")
	if err != nil {
		return "", err
	}
	captured, err := parseSnapshotStatusBaselineName(out)
	if err != nil {
		return "", err
	}
	return captured, nil
}

func parseSnapshotStatusBaselineName(out []byte) (string, error) {
	var status struct {
		Status   string `json:"status"`
		Name     string `json:"name"`
		Baseline struct {
			Name string `json:"name"`
		} `json:"baseline"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(out, &status); err != nil {
		return "", fmt.Errorf("parse git-control-tower snapshot status: %w", err)
	}
	if strings.TrimSpace(status.Status) != "ready" {
		if msg := strings.TrimSpace(status.Error); msg != "" {
			return "", fmt.Errorf("baseline snapshot not ready: status=%s: %s", status.Status, msg)
		}
		return "", fmt.Errorf("baseline snapshot not ready: status=%s", status.Status)
	}
	name := strings.TrimSpace(status.Baseline.Name)
	if name == "" {
		name = strings.TrimSpace(status.Name)
	}
	if name == "" {
		return "", fmt.Errorf("git-control-tower snapshot status returned no baseline name")
	}
	return name, nil
}

// cmdRequiredReadingSource discovers legacy migration input via the live
// prompt-manager discover surface through the CommandRunner seam.
type cmdRequiredReadingSource struct{ run CommandRunner }

// NewCommandRequiredReadingSource wires the legacy RequiredReadingSource over the
// given CommandRunner (prompt-manager). A nil runner always degrades.
func NewCommandRequiredReadingSource(run CommandRunner) RequiredReadingSource {
	return cmdRequiredReadingSource{run: run}
}

func (s cmdRequiredReadingSource) RequiredReading(ctx context.Context, title string) (string, error) {
	if s.run == nil {
		return "", fmt.Errorf("prompt-manager unavailable")
	}
	out, err := s.run(ctx, "prompt-manager", "discover", title, "--type", "skill,doc")
	if err != nil {
		return "", err
	}
	discovered := strings.TrimSpace(string(out))
	if discovered == "" {
		return "", fmt.Errorf("prompt-manager returned no required reading")
	}
	return discovered, nil
}

// cmdReferenceExtractor extracts code references via code-facts through the
// CommandRunner seam.
type cmdReferenceExtractor struct{ run CommandRunner }

// NewCommandReferenceExtractor wires the production ReferenceExtractor over the
// given CommandRunner (code-facts). A nil runner always degrades.
func NewCommandReferenceExtractor(run CommandRunner) ReferenceExtractor {
	return cmdReferenceExtractor{run: run}
}

func (e cmdReferenceExtractor) References(ctx context.Context, title, scope string) (string, error) {
	if e.run == nil {
		return "", fmt.Errorf("code-facts unavailable")
	}
	refs := extractCodeReferenceTargets(title + " " + scope)
	if len(refs) == 0 {
		return "", fmt.Errorf("no code reference targets found")
	}
	target := codeFactsDescribeTarget(refs[0])
	if _, err := e.run(ctx, "code-facts", "facts", "describe", target, "--include", "surfaces,parse_units", "--json"); err != nil {
		return "", err
	}
	lines := make([]string, 0, len(refs))
	for _, ref := range refs {
		lines = append(lines, "[CODE: "+ref+"]")
	}
	return strings.Join(lines, "\n"), nil
}

type cmdContextDiscoverer struct{ run CommandRunner }

func NewCommandContextDiscoverer(run CommandRunner) ContextDiscoverer {
	return cmdContextDiscoverer{run: run}
}

func (d cmdContextDiscoverer) DiscoverContext(ctx context.Context, title string, concepts []string, complexity string) ([]ContextCandidate, error) {
	if len(concepts) == 0 {
		concepts = []string{title}
	}
	var out []ContextCandidate
	for _, concept := range concepts {
		concept = strings.TrimSpace(concept)
		if concept == "" {
			continue
		}
		out = append(out,
			d.commandCandidate(ctx, concept, "prompt-manager-skill-discovery", planmodel.RelevantContextSkill, promptManagerSkillDiscoveryArgv(concept, complexity), "Discover and load relevant prompt-manager skills before implementation."),
			d.commandCandidate(ctx, concept, "prompt-manager-actions", planmodel.RelevantContextCommand, []string{"prompt-manager", "discover", concept, "--type", "all"}, "Find executable actions and operational tools before hand-rolling steps."),
			d.commandCandidate(ctx, concept, "search-hub-recall", planmodel.RelevantContextSearch, []string{"search-hub", "query", concept, "--type", "record,skill,doc"}, "Recall prior records, skills, and docs connected to this concept."),
			d.commandCandidate(ctx, concept, "cli-health-search", planmodel.RelevantContextSearch, []string{"cli-health", "search", concept}, "Find current CLI commands related to this concept."),
		)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no non-empty context concepts supplied")
	}
	return out, nil
}

func (d cmdContextDiscoverer) commandCandidate(ctx context.Context, concept, source string, kind planmodel.RelevantContextKind, argv []string, reason string) ContextCandidate {
	item := planmodel.RelevantContextItem{
		ID:           uuid.NewString(),
		Kind:         kind,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Label:        source + ": " + concept,
		Reason:       reason,
		Instruction:  "Run this during context setup; inspect the output and keep only relevant findings.",
		Command:      shellJoin(argv),
		Argv:         append([]string(nil), argv...),
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
		Source:       planmodel.RelevantContextSourceDiscovered,
		Status:       planmodel.RelevantContextStatusReady,
	}
	candidate := ContextCandidate{
		ID:      uuid.NewString(),
		Item:    item,
		Concept: concept,
		Source:  source,
		Status:  ContextCandidatePending,
	}
	if d.run == nil {
		candidate.Degraded = true
		candidate.Detail = source + " unavailable"
		candidate.Item.Status = planmodel.RelevantContextStatusDegraded
		return candidate
	}
	if _, err := d.run(ctx, argv[0], argv[1:]...); err != nil {
		candidate.Degraded = true
		candidate.Detail = err.Error()
		candidate.Item.Status = planmodel.RelevantContextStatusDegraded
	}
	return candidate
}

func promptManagerSkillDiscoveryArgv(concept, complexity string) []string {
	argv := []string{"prompt-manager", "discover", concept}
	if strings.TrimSpace(complexity) != "" {
		argv = append(argv, "--complexity", strings.TrimSpace(complexity))
	}
	return argv
}

func shellJoin(argv []string) string {
	parts := make([]string, 0, len(argv))
	for _, arg := range argv {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if strings.IndexFunc(arg, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' || r == '<' || r == '>' || r == '[' || r == ']' || r == ':' || r == ';' || r == '|' || r == '&' || r == ','
	}) < 0 {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

func anchorScenarioAndName(title, slug string) (string, string) {
	if s := sanitizeIdentifier(slug); s != "" {
		return s, s
	}
	s := sanitizeIdentifier(title)
	return s, s
}

func sanitizeIdentifier(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "_", "-")
	fields := strings.FieldsFunc(v, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-')
	})
	return strings.Trim(strings.Join(fields, "-"), "-")
}

var scenarioPathPattern = regexp.MustCompile(`(?:^|[\s\[\]()"'])((?:\./)?scenarios/[A-Za-z0-9._-]+/[^\s\[\]()"',]+)`)

func extractCodeReferenceTargets(text string) []string {
	matches := scenarioPathPattern.FindAllStringSubmatch(text, -1)
	seen := map[string]bool{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		target := strings.TrimPrefix(strings.TrimSpace(m[1]), "./")
		target = strings.TrimRight(target, ".,;:")
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		out = append(out, target)
	}
	sort.Strings(out)
	return out
}

func codeFactsDescribeTarget(path string) string {
	if name := scenarioFromPath(path); name != "" {
		return "scenario:" + name
	}
	return path
}

func scenarioFromPath(path string) string {
	path = strings.TrimPrefix(path, "./")
	const prefix = "scenarios/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if i := strings.IndexByte(rest, '/'); i > 0 {
		return rest[:i]
	}
	return rest
}
