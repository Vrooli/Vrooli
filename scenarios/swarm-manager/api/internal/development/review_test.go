package development

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) (Reviewer, Proposal) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"scenarios/example/.vrooli/service.json":                       `{"skills":{"usage":{"source":"skills/example/SKILL.md"},"improve":{"source":"skills/example-improve/SKILL.md","programs":["example.setpoint-read"]}}}`,
		"scenarios/example/PRD.md":                                     "Protected product outcome",
		"scenarios/example/skills/example/SKILL.md":                    "Usage",
		"scenarios/example/skills/example-improve/SKILL.md":            "Improvement judgment",
		"scenarios/example/.vrooli/program-runtime/setpoint-read.json": "{}",
		"scenarios/example/.vrooli/program-runtime/setpoint-read.py":   "# read-only program",
		"docs/agent-system/SCENARIO_DEVELOPMENT.md":                    "Approval and completion policy",
	}
	for name, data := range files {
		writeFixture(t, root, name, data)
	}
	return Reviewer{RepoRoot: root}, Proposal{Scenario: "example", WorkItem: "execute/example-development", Objective: "Repair local dictation and prove final-tail preservation", PlanRef: &PlanReference{Provider: PlanManagerProvider, PlanID: "plan-example-development", Slug: "example-development", Role: ExecutionSpecRole}, AcceptanceAllow: []string{"scenarios/example/**"}, AcceptanceDeny: []string{"scenarios/example/private/**"}, AllowedEffects: []string{"filesystem.write[paths=scenarios/example/**]", "process.test[scope=scenarios/example]"}, MaxTokens: 10000, MaxWallSeconds: 600, Outcomes: []Outcome{{ID: "tail", Criterion: "No silently lost captured interval", EvidenceSource: "testgenie.audio@v1/linux-amd64-chromium"}}}
}

func writeFixture(t *testing.T, root, name, data string) {
	t.Helper()
	filename := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPreviewFingerprintsProposalWithoutGrantingAuthority(t *testing.T) {
	r, p := fixture(t)
	got, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ReviewComplete || len(got.Artifacts) != 7 || len(got.ProposalDigest) != 64 || len(got.LaunchBlockers) == 0 {
		t.Fatalf("unexpected review: %#v", got)
	}
	for _, want := range []string{"NOT AUTHORIZATION TO RUN", p.Objective, "same item", p.Outcomes[0].Criterion, "operator disposition"} {
		if !strings.Contains(got.GoalMessage, want) {
			t.Errorf("goal lacks %q", want)
		}
	}
	again, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if again.ProposalDigest != got.ProposalDigest {
		t.Fatal("same input has unstable fingerprint")
	}
	data, err := os.ReadFile(filepath.Join(r.RepoRoot, "scenarios/example/PRD.md"))
	if err != nil || string(data) != "Protected product outcome" {
		t.Fatal("preview changed target")
	}
}

func TestPreviewRequiresCanonicalPlanReference(t *testing.T) {
	r, p := fixture(t)
	p.PlanRef = nil
	review, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if review.ReviewComplete {
		t.Fatal("planless development proposal was marked review-complete")
	}
	for _, finding := range review.Findings {
		if finding.Code == "plan_ref_required" {
			return
		}
	}
	t.Fatalf("missing plan_ref_required finding: %+v", review.Findings)
}

func TestPreviewFingerprintsDesignWithoutGrantingAuthority(t *testing.T) {
	r, p := fixture(t)
	p.Outcomes[0].ID = "OT-P0-001"
	name := "scenarios/example/DESIGN.md"
	writeFixture(t, r.RepoRoot, name, "Target architecture")
	p.ArtifactPaths = []string{name}
	before, err := r.Preview(p)
	if err != nil || !before.ReviewComplete || len(before.Artifacts) != 8 || len(before.LaunchBlockers) == 0 {
		t.Fatalf("design review: %+v, %v", before, err)
	}
	if !strings.Contains(before.GoalMessage, "OT-P0-001") {
		t.Fatal("canonical outcome identity was lost")
	}
	writeFixture(t, r.RepoRoot, name, "Changed target architecture")
	after, err := r.Preview(p)
	if err != nil || before.ProposalDigest == after.ProposalDigest {
		t.Fatalf("design change did not invalidate review: %v", err)
	}
	for _, unsafe := range []string{"scenarios/example/DESIGN.md/secret", "scenarios/example/DESIGN.json", "scenarios/example/private.md"} {
		if reviewablePath(unsafe) {
			t.Errorf("accepted non-contract path %q", unsafe)
		}
	}
}

func TestPreviewRejectsUntypedEffectProse(t *testing.T) {
	r, p := fixture(t)
	p.AllowedEffects = []string{"local edits and deterministic tests"}
	got, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewComplete {
		t.Fatal("untyped effect prose was admitted")
	}
	for _, finding := range got.Findings {
		if finding.Code == "effects_untyped" {
			return
		}
	}
	t.Fatalf("missing effects_untyped finding: %+v", got.Findings)
}

func TestPreviewRejectsMalformedOutcomeIdentities(t *testing.T) {
	for _, id := range []string{"", "../OT-P0-001", "OT P0 001", "OT-P0-001\n", "OT--P0-001", strings.Repeat("a", 129)} {
		r, p := fixture(t)
		p.Outcomes[0].ID = id
		if _, err := r.Preview(p); err == nil {
			t.Errorf("malformed outcome identity accepted: %q", id)
		}
	}
}

func TestRelevantSourceAndAuthorityChangesInvalidateFingerprint(t *testing.T) {
	r, p := fixture(t)
	before, _ := r.Preview(p)
	writeFixture(t, r.RepoRoot, "unrelated.txt", "concurrent user change")
	unrelated, _ := r.Preview(p)
	if unrelated.ProposalDigest != before.ProposalDigest {
		t.Fatal("unrelated edit invalidated proposal")
	}
	writeFixture(t, r.RepoRoot, "scenarios/example/PRD.md", "Different protected outcome")
	changed, _ := r.Preview(p)
	if changed.ProposalDigest == before.ProposalDigest {
		t.Fatal("target change retained fingerprint")
	}
	p.MaxTokens++
	budget, _ := r.Preview(p)
	if budget.ProposalDigest == changed.ProposalDigest {
		t.Fatal("budget change retained fingerprint")
	}
	p.Outcomes[0].Criterion = "weaker outcome"
	weaker, _ := r.Preview(p)
	if weaker.ProposalDigest == budget.ProposalDigest {
		t.Fatal("changed criterion retained fingerprint")
	}
}

func TestMissingArtifactsAndLimitsRemainReviewFindings(t *testing.T) {
	r, p := fixture(t)
	p.MaxTokens = 0
	p.MaxWallSeconds = 0
	p.Outcomes = nil
	p.AllowedEffects = nil
	p.ArtifactPaths = []string{"scenarios/example/docs/internal/MISSING.md"}
	got, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewComplete {
		t.Fatal("incomplete proposal declared complete")
	}
	codes := map[string]bool{}
	for _, f := range got.Findings {
		codes[f.Code] = true
	}
	for _, code := range []string{"artifact_unavailable", "budget_required", "outcomes_required", "effects_required"} {
		if !codes[code] {
			t.Errorf("missing %s", code)
		}
	}
}

func TestRejectUnsafeAndOversizedProposals(t *testing.T) {
	for _, name := range []string{"../secret", "/etc/passwd", "scenarios/example/.env", "scenarios/example/docs/../../private.json", "docs\\private.md"} {
		t.Run(name, func(t *testing.T) {
			r, p := fixture(t)
			p.ArtifactPaths = []string{name}
			if _, err := r.Preview(p); err == nil {
				t.Fatal("unsafe artifact accepted")
			}
		})
	}
	r, p := fixture(t)
	p.Scenario = "../example"
	if _, err := r.Preview(p); err == nil {
		t.Fatal("unsafe scenario accepted")
	}
	r, p = fixture(t)
	p.ArtifactPaths = make([]string, 65)
	if _, err := r.Preview(p); err == nil {
		t.Fatal("oversized inventory accepted")
	}
	r, p = fixture(t)
	p.Outcomes = append(p.Outcomes, p.Outcomes[0])
	if _, err := r.Preview(p); err == nil {
		t.Fatal("duplicate outcome accepted")
	}
	r, p = fixture(t)
	p.AcceptanceAllow = []string{"/**"}
	if _, err := r.Preview(p); err == nil {
		t.Fatal("absolute scope accepted")
	}
}

func TestUnreadableOrEscapingArtifactDoesNotBecomeEvidence(t *testing.T) {
	r, p := fixture(t)
	outside := filepath.Join(t.TempDir(), "private.md")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := "scenarios/example/docs/escape.md"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(r.RepoRoot, link)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(r.RepoRoot, link)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	p.ArtifactPaths = []string{link}
	got, err := r.Preview(p)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReviewComplete {
		t.Fatal("escaping artifact satisfied review")
	}
	for _, a := range got.Artifacts {
		if a.Path == link {
			t.Fatal("outside artifact fingerprinted")
		}
	}
	writeFixture(t, r.RepoRoot, "scenarios/example/docs/large.md", strings.Repeat("x", maxFileBytes+1))
	p.ArtifactPaths = []string{"scenarios/example/docs/large.md"}
	got, err = r.Preview(p)
	if err != nil || got.ReviewComplete {
		t.Fatalf("oversized artifact review: %v %#v", err, got)
	}
}
