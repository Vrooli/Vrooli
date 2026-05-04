package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkMember(team, member string, t Topics) MemberTopics {
	return MemberTopics{Ref: MemberRef{Team: team, Member: member}, Topics: t, Exists: true}
}

func TestRule_ConflictingDrain_OverlappingPrefixes(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "tx-a"}},
		}),
		mkMember("team-b", "bob", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/audience/*", Taxonomy: "tx-b"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	if r.Errors == 0 {
		t.Fatalf("expected conflicting_drain finding, got %v", r.Findings)
	}
	found := false
	for _, f := range r.Findings {
		if f.Rule == "conflicting_drain" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no conflicting_drain finding; findings=%v", r.Findings)
	}
}

func TestRule_ConflictingDrain_DisjointPrefixes(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "alice", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/audience/*", Taxonomy: "tx-a"}},
		}),
		mkMember("team-b", "bob", Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/competitor/*", Taxonomy: "tx-b"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "conflicting_drain" {
			t.Errorf("disjoint prefixes should not conflict; got %v", f)
		}
	}
}

func TestRule_OrphanOutput_NoConsumer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "isolated-output/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	found := false
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan_output, got %v", r.Findings)
	}
}

func TestRule_OrphanOutput_KnowledgeWithConsumer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared-knowledge/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			Intake:            []IntakeEntry{{Prefix: "shared-knowledge/*", Taxonomy: "tx"}},
			ExternalProducers: []string{"team-a"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("output with consumer should not be orphan; got %v", f)
		}
	}
}

func TestRule_OrphanOutput_NonKnowledgeIsNeverOrphan(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{
				{Prefix: "decisions/*", DestinationKind: DestinationDecision},
				{Prefix: "doctrine/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/PRIMITIVES.md")},
				{Prefix: "gaps/*", DestinationKind: DestinationCapabilityGap},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("non-knowledge destination should never be orphan; got %v", f)
		}
	}
}

func TestRule_OrphanInput_NoProducer(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake: []IntakeEntry{{Prefix: "lonely-input/*", Taxonomy: "tx"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	found := false
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan_input, got %v", r.Findings)
	}
}

func TestRule_OrphanInput_ExternalProducerSatisfies(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "external-input/*", Taxonomy: "tx"}},
			ExternalProducers: []string{"vision-walk"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			t.Errorf("external producer should satisfy intake; got %v", f)
		}
	}
}

func TestRule_OrphanInput_WildcardSourceTeamSkipsCheck(t *testing.T) {
	// source_team == "*" declares a universal-source intake: any team's
	// members may write. The orphan_input check should be skipped because
	// no specific peer producer is required by the topology. The paired
	// wildcard_source_misuse warning still fires when external_producers
	// is empty (covered by TestRule_WildcardSourceMisuse_*).
	wildcard := SourceTeamWildcard
	members := []MemberTopics{
		mkMember("scenario-qa", "bug-investigator", Topics{
			Intake: []IntakeEntry{{
				Prefix:     "bug-inbox/*",
				Taxonomy:   "bug-report",
				SourceTeam: &wildcard,
			}},
			ExternalProducers: []string{"report-bug-skill"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			t.Errorf("source_team=\"*\" should suppress orphan_input; got %v", f)
		}
	}
}

func TestRule_WildcardSourceMisuse_FiresWhenExternalProducersEmpty(t *testing.T) {
	// source_team=="*" without any external_producers is misuse: "I made
	// it universal but forgot to document who actually writes." Warning,
	// not error, because the topology still works — operators just lose
	// the audit trail.
	wildcard := SourceTeamWildcard
	members := []MemberTopics{
		mkMember("scenario-qa", "bug-investigator", Topics{
			Intake: []IntakeEntry{{
				Prefix:     "bug-inbox/*",
				Taxonomy:   "bug-report",
				SourceTeam: &wildcard,
			}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "wildcard_source_misuse" {
			hits++
			if f.Severity != SeverityWarning {
				t.Errorf("wildcard_source_misuse should be warning; got %s", f.Severity)
			}
		}
	}
	if hits != 1 {
		t.Errorf("expected 1 wildcard_source_misuse finding, got %d (findings=%v)", hits, r.Findings)
	}
}

func TestRule_WildcardSourceMisuse_QuietWhenExternalProducersDocumented(t *testing.T) {
	wildcard := SourceTeamWildcard
	members := []MemberTopics{
		mkMember("scenario-qa", "bug-investigator", Topics{
			Intake: []IntakeEntry{{
				Prefix:     "bug-inbox/*",
				Taxonomy:   "bug-report",
				SourceTeam: &wildcard,
			}},
			ExternalProducers: []string{"report-bug-skill"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "wildcard_source_misuse" {
			t.Errorf("documented external_producers should suppress misuse warning; got %v", f)
		}
	}
}

func TestRule_WildcardSourceMisuse_QuietForNonWildcardSources(t *testing.T) {
	specific := "marketing-crew"
	members := []MemberTopics{
		mkMember("monetization", "market-validator", Topics{
			Intake: []IntakeEntry{{
				Prefix:     "monetization-benchmark-adjacent-record/*",
				Taxonomy:   "monetization-validation",
				SourceTeam: &specific,
			}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "wildcard_source_misuse" {
			t.Errorf("specific source_team should not trip wildcard_source_misuse; got %v", f)
		}
	}
}

func TestRule_OrphanInput_PeerProducerSatisfies(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			Intake: []IntakeEntry{{Prefix: "shared/*", Taxonomy: "tx"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_input" {
			t.Errorf("peer producer should satisfy intake; got %v", f)
		}
	}
}

func TestRule_UnknownTaxonomy_FiresWhenSetButUnresolved(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "x/*", Taxonomy: "no-such-taxonomy"}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{Taxonomies: TaxonomyRegistry{
		"real-taxonomy": &Taxonomy{ID: "real-taxonomy"},
	}}
	r := Validate(members, opts)
	found := false
	for _, f := range r.Findings {
		if f.Rule == "unknown_taxonomy" && f.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown_taxonomy error; findings=%v", r.Findings)
	}
}

func TestRule_MissingTaxonomy_IsHardError(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "x/*"}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{Taxonomies: TaxonomyRegistry{}}
	r := Validate(members, opts)
	found := false
	for _, f := range r.Findings {
		if f.Rule == "missing_taxonomy" && f.Severity == SeverityError {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing_taxonomy error; findings=%v", r.Findings)
	}
	if r.Errors == 0 {
		t.Errorf("missing taxonomy should produce a hard error post-Phase-I")
	}
}

func TestRule_UnknownTaxonomy_SkippedWhenNoRegistryAndNoRepoRoot(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake:            []IntakeEntry{{Prefix: "x/*", Taxonomy: "anything"}},
			ExternalProducers: []string{"operator"},
		}),
	}
	r := Validate(members, ValidationOptions{}) // no taxonomies, no repo root
	for _, f := range r.Findings {
		if f.Rule == "unknown_taxonomy" || f.Rule == "missing_taxonomy" {
			t.Errorf("rule should be skipped without taxonomies+repo; got %v", f)
		}
	}
}

func TestRule_NonPortableClassifier_DetectsForbiddenContent(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "SKILL.md")
	body := `## Tools focus

This classifier is great. Look at research-inbox/foo for examples.
Run prompt-manager team knowledge-update to retag.
`
	if err := os.WriteFile(skillPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake: []IntakeEntry{{
				Prefix:          "y/*",
				Taxonomy:        "tx",
				ClassifierSkill: "leaky-classifier",
			}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{
		Taxonomies: TaxonomyRegistry{"tx": &Taxonomy{ID: "tx"}},
		SkillPaths: map[string]string{"leaky-classifier": skillPath},
	}
	r := Validate(members, opts)
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "non_portable_classifier" && f.Severity == SeverityError {
			hits++
		}
	}
	if hits == 0 {
		t.Errorf("expected non_portable_classifier error; findings=%v", r.Findings)
	}
}

func TestRule_NonPortableClassifier_CleanSkill(t *testing.T) {
	dir := t.TempDir()
	skillPath := filepath.Join(dir, "SKILL.md")
	body := `## Tools focus: Marketing Signal Classifier

Pure judgment. Read the taxonomy, score evidence, return a recommendation.
`
	if err := os.WriteFile(skillPath, []byte(body), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake: []IntakeEntry{{
				Prefix:          "y/*",
				Taxonomy:        "tx",
				ClassifierSkill: "clean-classifier",
			}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{
		Taxonomies: TaxonomyRegistry{"tx": &Taxonomy{ID: "tx"}},
		SkillPaths: map[string]string{"clean-classifier": skillPath},
	}
	r := Validate(members, opts)
	for _, f := range r.Findings {
		if f.Rule == "non_portable_classifier" {
			t.Errorf("clean classifier should not trip rule; got %v", f)
		}
	}
}

func TestRule_NonPortableClassifier_MissingFromRegistry(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "consumer", Topics{
			Intake: []IntakeEntry{{
				Prefix:          "y/*",
				Taxonomy:        "tx",
				ClassifierSkill: "ghost-classifier",
			}},
			ExternalProducers: []string{"operator"},
		}),
	}
	opts := ValidationOptions{
		Taxonomies: TaxonomyRegistry{"tx": &Taxonomy{ID: "tx"}},
		SkillPaths: map[string]string{"some-other-classifier": "/dev/null"},
	}
	r := Validate(members, opts)
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "non_portable_classifier" && f.Severity == SeverityError {
			hits++
		}
	}
	if hits == 0 {
		t.Errorf("expected non_portable_classifier error for missing skill; findings=%v", r.Findings)
	}
}

func TestRule_MissingDestinationSchema(t *testing.T) {
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{
				{Prefix: "good/*", DestinationKind: DestinationKnowledge, Schema: "audience-scan"},
				{Prefix: "bad/*", DestinationKind: DestinationKnowledge, Schema: "no-such-schema"},
			},
		}),
	}
	opts := ValidationOptions{
		Taxonomies: TaxonomyRegistry{
			"tx": &Taxonomy{
				ID:      "tx",
				Schemas: map[string]TaxonomySchema{"audience-scan": {}},
			},
		},
	}
	r := Validate(members, opts)
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "missing_destination_schema" {
			hits++
			if f.Severity != SeverityWarning {
				t.Errorf("missing_destination_schema should be warning; got %s", f.Severity)
			}
		}
	}
	if hits != 1 {
		t.Errorf("expected exactly 1 missing_destination_schema finding, got %d", hits)
	}
}

func TestRule_DanglingPORSink(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "docs", "agent-system"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "docs", "agent-system", "EXISTING.md"), []byte("real"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{
				{Prefix: "good/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/EXISTING.md")},
				{Prefix: "bad/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/MISSING.md")},
			},
		}),
	}
	r := Validate(members, ValidationOptions{RepoRoot: repoRoot})
	missingFound := 0
	for _, f := range r.Findings {
		if f.Rule == "dangling_por_sink" {
			missingFound++
			if !strings.Contains(f.Detail, "MISSING") {
				t.Errorf("dangling rule fired on existing file: %v", f)
			}
		}
	}
	if missingFound != 1 {
		t.Errorf("expected exactly 1 dangling_por_sink finding, got %d (findings=%v)", missingFound, r.Findings)
	}
}

func TestValidate_RealStoreCanary(t *testing.T) {
	// The canary backfill (marketing-crew + monetization + meta-opt + ...) on
	// the real store should validate clean for orphan rules and the new
	// taxonomy/classifier rules. dangling_por_sink will fire only if a
	// member declares a por_file destination with a missing path.
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	skillPaths, err := LoadSkillPaths(storeDir)
	if err != nil {
		t.Fatalf("LoadSkillPaths: %v", err)
	}
	repoRoot := filepath.Join(storeDir, "..", "..", "..")
	repoRoot, _ = filepath.Abs(repoRoot)
	r := Validate(members, ValidationOptions{
		RepoRoot:   repoRoot,
		SkillPaths: skillPaths,
	})
	if r.Errors > 0 {
		for _, f := range r.Findings {
			t.Logf("[%s] %s %s %s", f.Severity, f.Rule, f.Member, f.Detail)
		}
		t.Errorf("real-store validation produced %d errors and %d warnings", r.Errors, r.Warnings)
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name   string
		result ValidationResult
		want   int
	}{
		{"clean", ValidationResult{}, 0},
		{"warnings only", ValidationResult{Warnings: 5}, 0},
		{"any error", ValidationResult{Errors: 1, Warnings: 2}, 1},
	}
	for _, tt := range tests {
		if got := tt.result.ExitCode(); got != tt.want {
			t.Errorf("%s: ExitCode = %d, want %d", tt.name, got, tt.want)
		}
	}
}
