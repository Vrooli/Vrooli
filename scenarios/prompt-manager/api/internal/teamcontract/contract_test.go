package teamcontract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRejectsMissingOperatingContract(t *testing.T) {
	err := Validate(nil, ValidationInput{TeamID: "team-1"})
	if err == nil || !strings.Contains(err.Error(), "operatingContract is required") {
		t.Fatalf("expected missing contract error, got %v", err)
	}
}

func TestValidateRejectsMissingActiveMemberContract(t *testing.T) {
	contract := Minimal("", "agent-1")
	err := Validate(contract, ValidationInput{
		TeamID:    "team-1",
		MemberIDs: []string{"agent-1", "agent-2"},
	})
	if err == nil || !strings.Contains(err.Error(), `missing active member "agent-2"`) {
		t.Fatalf("expected missing active member error, got %v", err)
	}
}

func TestValidateFindingsCollectsIndependentContractDefects(t *testing.T) {
	contract := Minimal("", "agent-1")
	contract.SchemaVersion = 0
	findings := ValidateFindings(contract, ValidationInput{TeamID: "team-1", MemberIDs: []string{"agent-2"}})
	if len(findings) < 2 {
		t.Fatalf("ValidateFindings returned %d findings, want independent defects: %+v", len(findings), findings)
	}
	for _, field := range []string{
		"operatingContract.schemaVersion",
		"operatingContract.members",
	} {
		found := false
		for _, finding := range findings {
			if finding.Path == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing finding for %s: %+v", field, findings)
		}
	}
}

func TestNormalizePathRendersRepoRootRelative(t *testing.T) {
	got, err := NormalizePath(PathRef{Base: BaseTeamShared, Path: "RUN_LESSONS.md"}, ValidationInput{TeamID: "meta-optimization"}, "run-introspector")
	if err != nil {
		t.Fatalf("NormalizePath: %v", err)
	}
	want := "scenarios/prompt-manager/store/teams/meta-optimization/shared/RUN_LESSONS.md"
	if got != want {
		t.Fatalf("NormalizePath = %q, want %q", got, want)
	}
}

func TestNormalizePathRejectsTraversal(t *testing.T) {
	_, err := NormalizePath(PathRef{Base: BaseTeamShared, Path: "../TEAM.md"}, ValidationInput{TeamID: "meta-optimization"}, "agent-1")
	if err == nil || !strings.Contains(err.Error(), "must not escape") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestRenderMemberPolicyIncludesMemberPolicy(t *testing.T) {
	contract := Minimal("", "agent-1")
	rendered, err := RenderMemberPolicy(contract, RenderInput{
		TeamID:   "team-1",
		TeamName: "Team One",
		MemberID: "agent-1",
	})
	if err != nil {
		t.Fatalf("RenderMemberPolicy: %v", err)
	}
	for _, want := range []string{
		"## Your Member Contract",
		"Agent ID: agent-1",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered contract missing %q:\n%s", want, rendered)
		}
	}
	for _, forbidden := range []string{
		"# Resolved Operating Contract",
		"Team: team-1",
		// Required reads render into the active task brief's "## Required
		// Memory" section; the operating policy body must not duplicate them.
		"Required knowledge topics:",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("rendered member policy contains retired standalone contract text %q:\n%s", forbidden, rendered)
		}
	}
}

// Document surfaces have exactly one renderer. RenderMemberPolicy used to
// print plan-of-record hubs and shared-state paths under "## Document
// Authority", which RenderTeamStorage already prints with each surface's
// kind, owner, and purpose. Two renderings of one contract meant the agent
// reconciled them at read time, so the subset was removed. This test fails if
// it comes back.
func TestRenderMemberPolicyDoesNotRestateDocumentSurfaces(t *testing.T) {
	contract := Minimal("", "agent-1")
	contract.Documents.PlanOfRecord = []PlanOfRecordDocument{{
		ID:  "canon",
		Hub: &PathRef{Base: BaseRepoRoot, Path: "docs/monetization/README.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		Paths: []PathRef{
			{Base: BaseRepoRoot, Path: "docs/monetization/README.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			{Base: BaseRepoRoot, Path: "docs/monetization/strategy/PRICING.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		},
		WritePolicy:    "operator-curated-via-swarm-manager",
		Consumers:      []string{"monetization"},
		UseFor:         "monetization strategy, catalog, pricing, and channels",
		Required:       boolPtr(false),
		OptionalReason: "test fixture",
	}}
	contract.Documents.SharedState = []SharedStateDocument{{
		ID:             "state",
		Path:           PathRef{Base: BaseTeamShared, Path: "STATE.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		Kind:           "rolling-snapshot",
		Required:       false,
		OptionalReason: "test fixture",
	}}
	member := contract.Members["agent-1"]
	member.AllowedWrites = append(member.AllowedWrites, WriteRef{Base: BaseTeamShared, Path: "STATE.md"})
	member.ForbiddenWrites = []WriteRef{{Base: BaseRepoRoot, Path: "docs/monetization/"}}
	member.SafetyCriticalRules = []string{"Do not edit canon directly."}
	contract.Members["agent-1"] = member

	rendered, err := RenderMemberPolicy(contract, RenderInput{
		TeamID:   "team-1",
		TeamName: "Team One",
		MemberID: "agent-1",
	})
	if err != nil {
		t.Fatalf("RenderMemberPolicy: %v", err)
	}

	for _, restated := range []string{
		"## Document Authority",
		"Plan of record authorities:",
		"Team working state:",
		"## Write Rules",
		"Allowed writes:",
		"Forbidden writes:",
		"docs/monetization/README.md",
		"STATE.md",
	} {
		if strings.Contains(rendered, restated) {
			t.Fatalf("member policy restates document surface %q owned by RenderTeamStorage / the active task brief:\n%s", restated, rendered)
		}
	}

	// What is unique to this renderer must survive the removal.
	for _, want := range []string{
		"## Operating Constraints",
		"Safety-critical rules:",
		"Do not edit canon directly.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("member policy dropped member-only content %q:\n%s", want, rendered)
		}
	}
}

func TestValidateAcceptsFinalTeamWorkingStateKinds(t *testing.T) {
	for _, kind := range TeamWorkingStateKindIDs() {
		contract := Minimal("", "agent-1")
		contract.Documents.SharedState = []SharedStateDocument{{
			ID:             "state",
			Path:           PathRef{Base: BaseTeamShared, Path: "state.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			Kind:           kind,
			Required:       false,
			OptionalReason: "test fixture",
		}}

		if err := Validate(contract, ValidationInput{TeamID: "team-1"}); err != nil {
			t.Fatalf("Validate kind %q: %v", kind, err)
		}
	}
}

func TestValidateRejectsLegacyTeamWorkingStateKinds(t *testing.T) {
	for _, kind := range []string{"rolling-artifact", "append-only-log", "operator-state", "decision-stream", "handoff-history", "unknown"} {
		contract := Minimal("", "agent-1")
		contract.Documents.SharedState = []SharedStateDocument{{
			ID:             "state",
			Path:           PathRef{Base: BaseTeamShared, Path: "state.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			Kind:           kind,
			Required:       false,
			OptionalReason: "test fixture",
		}}

		err := Validate(contract, ValidationInput{TeamID: "team-1"})
		if err == nil || !strings.Contains(err.Error(), "team working state kind") {
			t.Fatalf("expected unsupported kind error for %q, got %v", kind, err)
		}
	}
}

func TestRenderTeamStorageIncludesDocumentSemantics(t *testing.T) {
	contract := Minimal("", "agent-1")
	contract.Documents.PlanOfRecord = []PlanOfRecordDocument{{
		ID:          "strategy",
		Paths:       []PathRef{{Base: BaseRepoRoot, Path: "docs/strategy.md", Required: boolPtr(false), OptionalReason: "test fixture"}},
		WritePolicy: "operator-curated-via-swarm-manager",
		Consumers:   []string{"team-1", "team-2"},
		UseFor:      "durable strategy canon",
	}}
	contract.Documents.SharedState = []SharedStateDocument{{
		ID:             "events",
		Path:           PathRef{Base: BaseTeamShared, Path: "events.jsonl", Required: boolPtr(false), OptionalReason: "test fixture"},
		OwnerMemberID:  "agent-1",
		Kind:           TeamWorkingStateKindAppendOnlyEventLog,
		Required:       false,
		OptionalReason: "test fixture",
	}}

	rendered, err := RenderTeamStorage(contract, RenderInput{TeamID: "team-1", MemberID: "agent-1"})
	if err != nil {
		t.Fatalf("RenderTeamStorage: %v", err)
	}
	for _, want := range []string{
		"## Your Team Storage",
		"Plan of record, read/propose only:",
		"- `docs/strategy.md`",
		"Policy: `operator-curated-via-swarm-manager`",
		"Consumers: `team-1, team-2`",
		"Use for: durable strategy canon",
		"Navigation: start at the hub and follow its file map to the relevant spoke.",
		"Team working state:",
		"Kind: `append-only-event-log`",
		"Use for: structured historical events or observations owned by the team",
		"Primitive availability for this member:",
		"- unified work filing: file findings and requests once into swarm-manager",
		"- knowledge: `write-allowed`",
		"- task board: `review-only`",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered storage missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "Shared state") {
		t.Fatalf("rendered storage must not use legacy shared state wording:\n%s", rendered)
	}
}

func TestValidateRequiresHubForMultiPathPlanOfRecord(t *testing.T) {
	contract := Minimal("", "agent-1")
	contract.Documents.PlanOfRecord = []PlanOfRecordDocument{{
		ID: "canon",
		Paths: []PathRef{
			{Base: BaseRepoRoot, Path: "docs/canon/README.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			{Base: BaseRepoRoot, Path: "docs/canon/spoke.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		},
		WritePolicy: "operator-curated-via-swarm-manager",
		Consumers:   []string{"team-1"},
		Required:    boolPtr(false),
	}}

	err := Validate(contract, ValidationInput{TeamID: "team-1"})
	if err == nil || !strings.Contains(err.Error(), "hub is required") {
		t.Fatalf("expected missing hub error, got %v", err)
	}
}

func TestRenderTeamStorageUsesPlanOfRecordHubs(t *testing.T) {
	contract := Minimal("", "agent-1")
	contract.Documents.PlanOfRecord = []PlanOfRecordDocument{{
		ID:  "canon",
		Hub: &PathRef{Base: BaseRepoRoot, Path: "docs/monetization/README.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		Paths: []PathRef{
			{Base: BaseRepoRoot, Path: "docs/monetization/README.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			{Base: BaseRepoRoot, Path: "docs/monetization/catalogs/CATALOG.md", Required: boolPtr(false), OptionalReason: "test fixture"},
			{Base: BaseRepoRoot, Path: "docs/monetization/strategy/PRICING.md", Required: boolPtr(false), OptionalReason: "test fixture"},
		},
		WritePolicy:    "operator-curated-via-swarm-manager",
		Consumers:      []string{"monetization"},
		UseFor:         "monetization strategy, catalog, pricing, and channels",
		Required:       boolPtr(false),
		OptionalReason: "test fixture",
	}}

	rendered, err := RenderTeamStorage(contract, RenderInput{TeamID: "team-1", MemberID: "agent-1"})
	if err != nil {
		t.Fatalf("RenderTeamStorage: %v", err)
	}
	for _, want := range []string{
		"- `docs/monetization/README.md`",
		"Policy: `operator-curated-via-swarm-manager`",
		"Consumers: `monetization`",
		"Use for: monetization strategy, catalog, pricing, and channels",
		// The file count folded in from the removed "## Document Authority"
		// rendering; this is now its only home.
		"Navigation: 3 declared files; start at the hub and follow its file map to the relevant spoke.",
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered storage missing %q:\n%s", want, rendered)
		}
	}
	for _, legacy := range []string{"docs under", "Exact paths: see `## Document Authority` above.", "`docs/monetization/catalogs/CATALOG.md`"} {
		if strings.Contains(rendered, legacy) {
			t.Fatalf("rendered storage contains legacy path-list wording %q:\n%s", legacy, rendered)
		}
	}
}

func TestBundledMetaOptimizationContractValidatesAndRendersRepoRootPaths(t *testing.T) {
	type teamFile struct {
		OperatingContract *OperatingContract `json:"operatingContract"`
	}

	storeDir := filepath.Clean("../../../store")
	repoRoot := filepath.Clean("../../../../..")
	data, err := os.ReadFile(filepath.Join(storeDir, "teams", "meta-optimization", "team.json"))
	if err != nil {
		t.Fatalf("read bundled team: %v", err)
	}
	var team teamFile
	if err := json.Unmarshal(data, &team); err != nil {
		t.Fatalf("unmarshal bundled team: %v", err)
	}

	err = Validate(team.OperatingContract, ValidationInput{
		TeamID:   "meta-optimization",
		StoreDir: storeDir,
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatalf("Validate bundled contract: %v", err)
	}

	// Shared-state paths render in team storage, which is their single home
	// since "## Document Authority" was removed from the member policy.
	rendered, err := RenderTeamStorage(team.OperatingContract, RenderInput{
		TeamID:   "meta-optimization",
		TeamName: "Meta Optimization",
		MemberID: "team-agent-optimizer",
		StoreDir: storeDir,
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatalf("RenderTeamStorage bundled contract: %v", err)
	}
	want := "scenarios/prompt-manager/store/teams/meta-optimization/shared/TEAM_AUDIT.md"
	if !strings.Contains(rendered, want) {
		t.Fatalf("rendered storage missing repo-root TEAM_AUDIT path %q:\n%s", want, rendered)
	}
	if strings.Contains(rendered, "\n- shared/TEAM_AUDIT.md") {
		t.Fatalf("rendered storage contains ambiguous TEAM_AUDIT path:\n%s", rendered)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func TestBundledTeamContractsValidate(t *testing.T) {
	type teamFile struct {
		OperatingContract *OperatingContract `json:"operatingContract"`
	}

	storeDir := filepath.Clean("../../../store")
	repoRoot := filepath.Clean("../../../../..")
	teamsDir := filepath.Join(storeDir, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		t.Fatalf("read teams dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		teamID := entry.Name()
		data, err := os.ReadFile(filepath.Join(teamsDir, teamID, "team.json"))
		if err != nil {
			t.Fatalf("read %s team: %v", teamID, err)
		}
		var team teamFile
		if err := json.Unmarshal(data, &team); err != nil {
			t.Fatalf("unmarshal %s team: %v", teamID, err)
		}
		if err := Validate(team.OperatingContract, ValidationInput{
			TeamID:   teamID,
			StoreDir: storeDir,
			RepoRoot: repoRoot,
		}); err != nil {
			t.Fatalf("Validate bundled %s contract: %v", teamID, err)
		}
	}
}

func TestBundledPromptProseDoesNotRestateContractPolicy(t *testing.T) {
	storeDir := filepath.Clean("../../../store")
	files := appendGlob(t, nil, filepath.Join(storeDir, "teams", "*", "shared", "TEAM.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "teams", "*", "members", "*", "RESPONSIBILITIES.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "teams", "*", "members", "*", "HEARTBEAT.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "SOUL.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "AGENTS.md"))
	files = appendGlob(t, files, filepath.Join(storeDir, "agents", "*", "TOOLS.md"))

	forbidden := []string{
		"Team-ceiling",
		"Team ceiling",
		"Own-context cap",
		"Work Item Types",
		"Decision Queue Discipline",
		"Shared State",
		"shared state",
		"shared-state",
		"Durable State",
		"Knowledge supersession",
		"14 heartbeats",
		"4+ decisions",
		"3+ decisions",
		"max new decisions",
		"Team queue at capacity",
		"Apply the resolved operating contract",
		"The resolved operating contract is authoritative",
		"Use the resolved operating contract",
		"resolved operating contract",
		"source documents, shared state, write rules",
		"source-document paths",
		"writable surfaces",
		"Write the required knowledge",
		"write required knowledge",
		"End with HANDOFF",
		"Raise decisions only",
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read prompt prose %s: %v", file, err)
		}
		text := string(data)
		for _, phrase := range forbidden {
			if strings.Contains(text, phrase) {
				t.Fatalf("%s restates contract-owned policy phrase %q", file, phrase)
			}
		}
	}
}

func appendGlob(t *testing.T, files []string, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	return append(files, matches...)
}
