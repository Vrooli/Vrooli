// Package development owns revision-bound scenario-development engagements.
// Review is read-only; admission and accounting are explicit service operations.
package development

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

const (
	maxFileBytes = 512 * 1024
	maxArtifacts = 64
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Outcome identities include canonical PRD targets such as OT-P0-001. Keep
// their spelling intact so review and evidence refer to the same target.
var outcomeIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+(?:-[A-Za-z0-9]+)*$`)

type Outcome struct {
	ID, Criterion, EvidenceSource string
}

// PlanReference identifies the immutable Plan Manager work package that owns
// this development strategy. Adaptive development is an execution strategy,
// not a second kind of plan.
type PlanReference struct {
	Provider string `json:"provider"`
	PlanID   string `json:"plan_id"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
}

const (
	PlanManagerProvider = "plan-manager"
	ExecutionSpecRole   = "execution_spec"
	AdaptiveStrategy    = "adaptive-improvement"
)

func normalizePlanReference(ref *PlanReference) *PlanReference {
	if ref == nil {
		return nil
	}
	normalized := &PlanReference{
		Provider: strings.TrimSpace(ref.Provider),
		PlanID:   strings.TrimSpace(ref.PlanID),
		Slug:     strings.TrimSpace(ref.Slug),
		Role:     strings.TrimSpace(ref.Role),
	}
	if normalized.Provider == "" && normalized.PlanID == "" && normalized.Slug == "" && normalized.Role == "" {
		return nil
	}
	return normalized
}

func validatePlanReference(ref *PlanReference) error {
	if ref == nil {
		return fmt.Errorf("canonical plan reference is required")
	}
	if ref.Provider != PlanManagerProvider || ref.PlanID == "" || ref.Slug == "" || ref.Role != ExecutionSpecRole {
		return fmt.Errorf("plan reference must identify a plan-manager execution_spec")
	}
	if len(ref.Provider) > 64 || len(ref.PlanID) > 200 || len(ref.Slug) > 200 || len(ref.Role) > 64 {
		return fmt.Errorf("plan reference exceeds its bounded field length")
	}
	return nil
}

func planReferencesEqual(a, b *PlanReference) bool {
	a, b = normalizePlanReference(a), normalizePlanReference(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

type Proposal struct {
	Scenario, WorkItem, Objective                                  string
	ArtifactPaths, AcceptanceAllow, AcceptanceDeny, AllowedEffects []string
	Outcomes                                                       []Outcome
	MaxTokens, MaxWallSeconds                                      int64
	BudgetPolicy                                                   string
	PlanRef                                                        *PlanReference
	ExecutionStrategy                                              string
	Guidance                                                       Guidance
}

type Artifact struct {
	Path, SHA256 string
	SizeBytes    int64
}

type Finding struct{ Code, Detail string }

type Review struct {
	ProposalDigest, GoalMessage string
	Artifacts                   []Artifact
	Findings                    []Finding
	ReviewComplete              bool
	LaunchBlockers              []string
	// Contents are the exact bytes fingerprinted in this read, never a second
	// filesystem read at approval time. Transport previews omit these bytes.
	Contents []ArtifactContent `json:"-"`
}

type ArtifactContent struct {
	Path  string `json:"path"`
	Bytes []byte `json:"bytes"`
}

// Reviewer reads only beneath an explicit repository root. OpenRoot prevents
// symlinks from escaping that root; candidates never supply a filesystem root.
type Reviewer struct{ RepoRoot string }

func (r Reviewer) Preview(p Proposal) (Review, error) {
	var result Review
	p.BudgetPolicy = defaultBudgetPolicy(p.BudgetPolicy)
	p.PlanRef = normalizePlanReference(p.PlanRef)
	if strings.TrimSpace(p.ExecutionStrategy) == "" {
		p.ExecutionStrategy = AdaptiveStrategy
	}
	if err := validateBudgetPolicy(p.BudgetPolicy); err != nil {
		return result, err
	}
	if err := p.Guidance.Validate(); err != nil {
		return result, err
	}
	if !slugPattern.MatchString(p.Scenario) || len(p.Scenario) > 100 {
		return result, fmt.Errorf("scenario must be a canonical scenario slug")
	}
	if len(p.ArtifactPaths) > maxArtifacts || len(p.Outcomes) > 32 || len(p.Objective) > 4096 {
		return result, fmt.Errorf("proposal exceeds artifact, outcome, or objective limit")
	}
	for _, values := range [][]string{p.AcceptanceAllow, p.AcceptanceDeny, p.AllowedEffects} {
		if len(values) > 64 {
			return result, fmt.Errorf("proposal contains too many scope/effect entries")
		}
		for _, value := range values {
			if len(value) > 1024 {
				return result, fmt.Errorf("scope/effect entry exceeds 1024 bytes")
			}
		}
	}
	root, err := os.OpenRoot(r.RepoRoot)
	if err != nil {
		return result, fmt.Errorf("repository root unavailable")
	}
	defer root.Close()
	base := "scenarios/" + p.Scenario + "/"
	manifestBytes, err := readFile(root, base+".vrooli/service.json")
	if err != nil {
		return result, fmt.Errorf("scenario manifest unavailable")
	}
	var manifest struct {
		Skills struct {
			Usage struct {
				Source   string   `json:"source"`
				Programs []string `json:"programs"`
			} `json:"usage"`
			Improve struct {
				Source   string   `json:"source"`
				Programs []string `json:"programs"`
			} `json:"improve"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return result, fmt.Errorf("scenario manifest is invalid")
	}
	add := func(code, detail string) { result.Findings = append(result.Findings, Finding{code, detail}) }
	if err := validatePlanReference(p.PlanRef); err != nil {
		add("plan_ref_required", "Bind this work package to the accepted Plan Manager execution plan before approval.")
	}
	if p.ExecutionStrategy != AdaptiveStrategy {
		add("execution_strategy_invalid", "Development review requires the adaptive-improvement strategy; phased execution uses the ordinary plan path.")
	}
	paths := append([]string{base + "PRD.md", base + ".vrooli/service.json", "docs/agent-system/SCENARIO_DEVELOPMENT.md"}, p.ArtifactPaths...)
	for role, source := range map[string]string{"usage": manifest.Skills.Usage.Source, "improve": manifest.Skills.Improve.Source} {
		if !strings.HasPrefix(source, "skills/") || !fs.ValidPath(source) || !strings.HasSuffix(source, "/SKILL.md") {
			add("missing_skill", "Declare a scenario-owned "+role+" skill source.")
			continue
		}
		paths = append(paths, base+source)
	}
	programs := append(append([]string{}, manifest.Skills.Usage.Programs...), manifest.Skills.Improve.Programs...)
	if len(programs) > 32 {
		return result, fmt.Errorf("scenario declares too many programs for a bounded review")
	}
	for _, program := range programs {
		name, local := strings.CutPrefix(program, p.Scenario+".")
		if !local || !slugPattern.MatchString(name) {
			add("program_review_required", "Non-local or invalid declared program requires owner review.")
			continue
		}
		paths = append(paths, base+".vrooli/program-runtime/"+name+".json", base+".vrooli/program-runtime/"+name+".py")
	}
	paths = uniqueSorted(paths)
	if len(paths) > maxArtifacts {
		return result, fmt.Errorf("resolved artifact inventory exceeds 64 files")
	}
	for _, name := range paths {
		if !reviewablePath(name) {
			return result, fmt.Errorf("artifact path is not a reviewable project artifact: %q", name)
		}
		data, readErr := readFile(root, name)
		if name == base+".vrooli/service.json" {
			data, readErr = manifestBytes, nil
		}
		if readErr != nil {
			add("artifact_unavailable", "Cannot fingerprint "+name+"; confirm existence, file type and size.")
			continue
		}
		sum := sha256.Sum256(data)
		result.Artifacts = append(result.Artifacts, Artifact{name, hex.EncodeToString(sum[:]), int64(len(data))})
		result.Contents = append(result.Contents, ArtifactContent{Path: name, Bytes: data})
	}
	if strings.TrimSpace(p.WorkItem) == "" {
		add("work_identity_required", "Select the owning work item; preview does not create one.")
	}
	if len(p.WorkItem) > 200 {
		return result, fmt.Errorf("work identity exceeds 200 bytes")
	}
	if strings.TrimSpace(p.Objective) == "" {
		add("objective_required", "State the bounded product outcome.")
	}
	if len(p.AcceptanceAllow) == 0 {
		add("scope_required", "Declare permitted change paths and dependency repairs.")
	}
	for _, boundary := range append(append([]string{}, p.AcceptanceAllow...), p.AcceptanceDeny...) {
		if !fs.ValidPath(boundary) || strings.ContainsAny(boundary, "\\\n\r") || !strings.Contains(boundary, "/") {
			return result, fmt.Errorf("scope must contain repository-relative path patterns")
		}
	}
	if len(p.AllowedEffects) == 0 {
		add("effects_required", "Describe allowed runtime, network, spending and publication effects.")
	} else if err := validateAllowedEffects(p.AllowedEffects); err != nil {
		add("effects_untyped", err.Error())
	}
	if p.MaxTokens <= 0 || p.MaxWallSeconds <= 0 {
		add("budget_required", "Propose positive aggregate token and wall-time limits for review.")
	}
	if len(p.Outcomes) == 0 {
		add("outcomes_required", "Supply protected acceptance criteria and their evidence sources.")
	}
	seen := map[string]bool{}
	for _, outcome := range p.Outcomes {
		if len(outcome.Criterion) > 4096 || len(outcome.EvidenceSource) > 2048 {
			return result, fmt.Errorf("outcome exceeds text limits")
		}
		if len(outcome.ID) > 128 || !outcomeIDPattern.MatchString(outcome.ID) || seen[outcome.ID] {
			return result, fmt.Errorf("outcome IDs must be unique bounded alphanumeric identifiers separated by hyphens")
		}
		seen[outcome.ID] = true
		if strings.TrimSpace(outcome.Criterion) == "" || strings.TrimSpace(outcome.EvidenceSource) == "" {
			add("outcome_incomplete", "Criterion and evidence source required for "+outcome.ID+".")
		} else if _, err := ParseEvidenceContract(outcome.EvidenceSource); err != nil {
			add("evidence_untyped", err.Error())
		}
	}
	sort.Slice(result.Findings, func(i, j int) bool {
		return result.Findings[i].Code+result.Findings[i].Detail < result.Findings[j].Code+result.Findings[j].Detail
	})
	encoded, err := json.Marshal(struct {
		Version   string
		Proposal  Proposal
		Artifacts []Artifact
		// Bind the instruction text as well as its inputs. Render with an empty
		// fingerprint to avoid a self-referential hash. A renderer change must
		// invalidate an earlier review even if all source files stayed the same.
		GoalMessage string
	}{"development-review/v2", p, result.Artifacts, goalMessage(p, result)})
	if err != nil {
		return result, err
	}
	sum := sha256.Sum256(encoded)
	result.ProposalDigest = hex.EncodeToString(sum[:])
	result.ReviewComplete = len(result.Findings) == 0
	result.LaunchBlockers = append([]string{
		"Operator has not approved this proposal; preview grants no authority.",
		"Preview bytes are not retained; approval must match this fingerprint before retaining an immutable snapshot.",
		"Outcome evidence, artifact semantics and declared scope/effects require review; field presence is not qualification.",
	}, RuntimeBlockers()...)
	result.GoalMessage = goalMessage(p, result)
	return result, nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func reviewablePath(name string) bool {
	if !fs.ValidPath(name) || strings.Contains(name, "\\") {
		return false
	}
	if strings.HasPrefix(name, "docs/") {
		return strings.HasSuffix(name, ".md")
	}
	parts := strings.SplitN(name, "/", 3)
	if len(parts) != 3 || parts[0] != "scenarios" || !slugPattern.MatchString(parts[1]) {
		return false
	}
	rel := parts[2]
	if rel == "PRD.md" || rel == "DESIGN.md" || rel == ".vrooli/service.json" {
		return true
	}
	for _, prefix := range []string{"docs/", "requirements/", "experience/", "skills/", ".vrooli/program-runtime/"} {
		if strings.HasPrefix(rel, prefix) {
			switch path.Ext(rel) {
			case ".md", ".json", ".py":
				return true
			}
		}
	}
	return false
}

func readFile(root *os.Root, name string) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() > maxFileBytes {
		return nil, fmt.Errorf("not a bounded regular file")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
		return nil, fmt.Errorf("not a bounded regular file")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("file grew beyond limit")
	}
	return data, err
}

func goalMessage(p Proposal, review Review) string {
	var b strings.Builder
	b.WriteString("DRAFT FOR OPERATOR REVIEW — NOT AUTHORIZATION TO RUN\n")
	fmt.Fprintf(&b, "Scenario: %s\nOwning item (proposed): %s\nProposal fingerprint: %s\nObjective: %s\n", p.Scenario, p.WorkItem, review.ProposalDigest, p.Objective)
	if p.PlanRef != nil {
		fmt.Fprintf(&b, "Canonical Plan Manager plan: %s (%s)\n", p.PlanRef.Slug, p.PlanRef.PlanID)
	}
	fmt.Fprintf(&b, "Execution strategy: %s\n", p.ExecutionStrategy)
	b.WriteString("After explicit admission, read the approved target snapshot and the scenario improvement skill. Perform successive authorized repairs under this same item; do not create a new approval per engineering decision.\n")
	b.WriteString(renderGuidance(p.Guidance))
	if validItem(p.WorkItem) {
		fmt.Fprintf(&b, "Retrieve the retained packet with swarm-manager development get --item %s --json. Verify its approved digest; read retained sources with swarm-manager development artifact --item %s --digest <approved-digest> --path <artifact-path>. Live documentation does not silently replace that target.\n", p.WorkItem, p.WorkItem)
	}
	b.WriteString("Read scenario-improvement-campaign for the execution method and scenario-work-ladder for layer selection.\nTarget artifacts observed during preview:\n")
	for _, a := range review.Artifacts {
		fmt.Fprintf(&b, "- %s (sha256:%s)\n", a.Path, a.SHA256)
	}
	fmt.Fprintf(&b, "Proposed allow paths: %s\nProposed prohibited paths: %s\nProposed effects: %s\nProposed aggregate limits: %d tokens; %d seconds. These are not grants.\n", strings.Join(p.AcceptanceAllow, ", "), strings.Join(p.AcceptanceDeny, ", "), strings.Join(p.AllowedEffects, "; "), p.MaxTokens, p.MaxWallSeconds)
	b.WriteString(renderBudgetPolicy(p.BudgetPolicy))
	b.WriteString("Required outcomes proposed for review:\n")
	for _, o := range p.Outcomes {
		fmt.Fprintf(&b, "- %s: %s Evidence: %s\n", o.ID, o.Criterion, o.EvidenceSource)
	}
	b.WriteString("Checkpoint interventions, evidence references, unmet outcomes, owner waits and remaining limits. Preserve unknown readings. Request an amendment before weakening targets or exceeding authority. No paid-provider call, commit, push, release or deployment is authorized by this preview. Harness completion is not product acceptance; return evidence for operator disposition.\n")
	return b.String()
}
