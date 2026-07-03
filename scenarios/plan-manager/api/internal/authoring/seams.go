package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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

// PlanReader is the read seam back into the plans SSOT. Finalize uses it as a
// read-after-write guard: a successful finalize response must refer to a plan
// that resolves through the same plans-domain paths callers use.
type PlanReader interface {
	GetPlan(ctx context.Context, idOrSlug string) (planmodel.Plan, error)
	RenderPlan(ctx context.Context, idOrSlug string) (string, error)
}

// PlanRenderer renders an in-progress plan to its markdown review artifact so the
// wizard can offer a render-preview before finalize. Production wires
// plans.RenderMarkdown; a nil renderer disables preview (the service returns a
// typed "preview unavailable" error). The renderer is pure — no persistence.
type PlanRenderer interface {
	Render(p planmodel.Plan) string
}

type DraftPlanRenderer interface {
	RenderDraft(p planmodel.Plan, sessionID string) string
}

// AnchorIntentDeriver derives the typed regression-anchor INTENT for a
// plan-in-progress — the "before" the executor will snapshot. It is pure and
// deterministic (title/slug → typed intent fields), needs no git-control-tower,
// and never goes stale: the actual baseline snapshot is captured fresh at
// execution start (see the execution InputFreshener seam), however many days
// later. Production wires the default deriver; tests inject a fake.
type AnchorIntentDeriver interface {
	// DeriveAnchorIntent returns the boundary-native regression-anchor intent
	// block for the given plan title/slug and change boundary. Affected scenarios
	// and the tiered baseline/diff commands are derived from the boundary; it
	// always succeeds — intent is cheap, deterministic, and never stale (no
	// snapshot, no dependency).
	DeriveAnchorIntent(ctx context.Context, title, slug string, boundary planmodel.ChangeBoundary) string
}

// SkillPackDiscoverer bootstraps plan-wide prompt-manager skills from
// decomposed concepts. It returns ready-to-store context items; no candidate
// queue or search-hub result mirroring lives in plan-manager.
type SkillPackDiscoverer interface {
	DiscoverSkillPack(ctx context.Context, title string, concepts []string, complexity string) (SkillPackResult, error)
}

// CommandRunner is the exec seam shared by production discovery sources.
// Production wires execRunner (LookPath-guarded, timeout-bounded); tests inject
// a fake. A nil runner means "no live dispatch" — the source degrades honestly.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// DefaultRunner returns the production CommandRunner (LookPath-guarded,
// timeout-bounded). Wired in the handler module; tests inject a fake instead.
func DefaultRunner() CommandRunner { return execRunner }

// autofillTimeout bounds a single external dispatch. Generous (skill discovery
// can shell out) but finite so a hung command cannot block the wizard forever.
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

// --- production discovery sources ---

// defaultAnchorIntentDeriver is the production AnchorIntentDeriver: a pure,
// deterministic derivation of the typed regression-anchor intent block (no
// git-control-tower, no snapshot). It promotes the structured intent template to
// the primary authoring output.
type defaultAnchorIntentDeriver struct{}

// DefaultAnchorIntentDeriver returns the production AnchorIntentDeriver. Wired in
// the handler module; tests inject a fake instead.
func DefaultAnchorIntentDeriver() AnchorIntentDeriver { return defaultAnchorIntentDeriver{} }

func (defaultAnchorIntentDeriver) DeriveAnchorIntent(_ context.Context, title, slug string, boundary planmodel.ChangeBoundary) string {
	return RegressionAnchorIntentTemplate(title, slug, boundary)
}

type cmdSkillPackDiscoverer struct {
	run CommandRunner
}

func NewCommandSkillPackDiscoverer(run CommandRunner) SkillPackDiscoverer {
	return cmdSkillPackDiscoverer{run: run}
}

func (d cmdSkillPackDiscoverer) DiscoverSkillPack(ctx context.Context, title string, concepts []string, complexity string) (SkillPackResult, error) {
	if d.run == nil {
		return SkillPackResult{}, fmt.Errorf("prompt-manager unavailable")
	}
	cleaned := make([]string, 0, len(concepts))
	for _, concept := range concepts {
		if concept = strings.TrimSpace(concept); concept != "" {
			cleaned = append(cleaned, concept)
		}
	}
	if len(cleaned) == 0 {
		if title = strings.TrimSpace(title); title == "" {
			return SkillPackResult{}, fmt.Errorf("no non-empty skill-pack concepts supplied")
		}
		cleaned = []string{title}
	}
	args := append([]string{"discover"}, cleaned...)
	args = append(args, "--type", "skill", "--json")
	if strings.TrimSpace(complexity) != "" {
		args = append(args, "--complexity", strings.TrimSpace(complexity))
	}
	out, err := d.run(ctx, "prompt-manager", args...)
	if err != nil {
		return SkillPackResult{}, err
	}
	return parseSkillPackDiscovery(out)
}

type promptManagerDiscoverResponse struct {
	Results []promptManagerSkillResult `json:"results"`
	Items   []promptManagerSkillResult `json:"items"`

	ReadCommand            string `json:"readCommand"`
	RecommendedReadCommand string `json:"recommendedReadCommand"`
	BudgetStatus           string `json:"budgetStatus"`
	TotalContentChars      int    `json:"totalContentChars"`
	Complexity             string `json:"complexity"`
}

type promptManagerSkillResult struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Score        float64  `json:"score"`
	Source       string   `json:"source"`
	ContentChars int      `json:"contentChars"`
	Tags         []string `json:"tags"`
}

func parseSkillPackDiscovery(out []byte) (SkillPackResult, error) {
	var resp promptManagerDiscoverResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return SkillPackResult{}, err
	}
	results := resp.Results
	if len(results) == 0 {
		results = resp.Items
	}
	deduped := make([]planmodel.RelevantContextItem, 0, len(results))
	seen := map[string]bool{}
	for _, result := range results {
		slug := strings.TrimSpace(result.ID)
		if slug == "" {
			slug = strings.TrimSpace(result.Name)
		}
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		label := firstNonEmpty(strings.TrimSpace(result.Name), slug)
		reason := strings.TrimSpace(result.Description)
		if reason == "" {
			reason = "Recommended by prompt-manager skill discovery."
		}
		deduped = append(deduped, planmodel.RelevantContextItem{
			ID:           uuid.NewString(),
			Kind:         planmodel.RelevantContextSkill,
			Scope:        planmodel.RelevantContextScopeGlobal,
			Label:        label,
			Reason:       reason,
			Instruction:  "Load this internal skill before implementation unless it is clearly irrelevant.",
			Target:       slug,
			Required:     true,
			RepeatPolicy: planmodel.RelevantContextOncePerExecution,
			Source:       planmodel.RelevantContextSourceDiscovered,
			Status:       planmodel.RelevantContextStatusReady,
		})
	}
	summary := fmt.Sprintf("prompt-manager returned %d skill(s)", len(deduped))
	if resp.TotalContentChars > 0 {
		summary += fmt.Sprintf(" (%d chars)", resp.TotalContentChars)
	}
	if strings.TrimSpace(resp.Complexity) != "" {
		summary += "; complexity=" + strings.TrimSpace(resp.Complexity)
	}
	return SkillPackResult{
		Items:                  deduped,
		ReadCommand:            strings.TrimSpace(resp.ReadCommand),
		RecommendedReadCommand: strings.TrimSpace(resp.RecommendedReadCommand),
		BudgetStatus:           strings.TrimSpace(resp.BudgetStatus),
		Summary:                summary,
	}, nil
}

// SkillSuggestion is one steered skill recommendation from the applicability
// resolver: a bare skill slug plus the concrete reason it applies.
type SkillSuggestion struct {
	Slug   string
	Reason string
}

// seam: SkillApplicabilityResolver proposes skills that APPLY to a plan's
// change boundary (its blast radius) rather than merely matching its text.
//
// Production currently wires NoopSkillApplicabilityResolver: the boundary→skill
// applicability sources this seam is designed for are still landing elsewhere.
// Future wiring (contract decision D7):
//   - health scenarios' self-declared applicability rules (each health scenario
//     declares which surfaces its steer skill applies to), and
//   - the maturity system's phase→skill links.
//
// Whatever the source, it must stay DATA-DRIVEN — a hard-coded path→skill table
// in this repo is prohibited; that is why the seam ships no-op instead of a
// heuristic.
type SkillApplicabilityResolver interface {
	// SuggestSkills returns steered skill suggestions for the plan's change
	// boundary. Failures degrade to no suggestions; never block the wizard.
	SuggestSkills(ctx context.Context, boundary planmodel.ChangeBoundary) []SkillSuggestion
}

// NoopSkillApplicabilityResolver is the production placeholder for the D7 seam.
type NoopSkillApplicabilityResolver struct{}

func (NoopSkillApplicabilityResolver) SuggestSkills(context.Context, planmodel.ChangeBoundary) []SkillSuggestion {
	return nil
}

// resolverOrNoop keeps the service's resolver non-nil so call sites never
// nil-check the seam.
func resolverOrNoop(r SkillApplicabilityResolver) SkillApplicabilityResolver {
	if r == nil {
		return NoopSkillApplicabilityResolver{}
	}
	return r
}
