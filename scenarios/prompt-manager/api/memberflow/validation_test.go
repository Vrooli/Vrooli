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

func TestRule_OrphanOutput_RequiredReadSatisfies(t *testing.T) {
	// A required_read[] entry on any member that overlaps a knowledge output
	// prefix counts as a consumer (consumerSet includes RequiredRead). The
	// previously-orphaned prefix is no longer flagged.
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "shared/*"}},
			Output:       []OutputEntry{{Prefix: "shared/*", DestinationKind: DestinationKnowledge}}, // also produced peer-side so unread_required stays quiet
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("required_read overlap should satisfy orphan_output; got %v", f)
		}
	}
}

func TestRule_OrphanOutput_EvidenceConsumedSatisfies(t *testing.T) {
	// An evidence_consumed[] entry on any member that overlaps a knowledge
	// output prefix counts as a consumer. The for_decisions field stays
	// validator-shape-only at this layer; the cross-check on decision-context
	// existence is ruleDanglingEvidenceDecision (Phase 1.2), which needs a
	// team-contract registry — irrelevant to consumer-set membership.
	members := []MemberTopics{
		mkMember("monetization", "opportunity-scout", Topics{
			Output: []OutputEntry{{Prefix: "candidate-sku-record/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("monetization", "catalog-strategist", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "candidate-sku-record/*",
				ForDecisions: []string{"catalog-promotion"},
			}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "orphan_output" {
			t.Errorf("evidence_consumed overlap should satisfy orphan_output; got %v", f)
		}
	}
}

func TestRule_UnreadRequired_NoProducer(t *testing.T) {
	// A required_read prefix with no overlapping output[] across any member
	// fires unread_required at warning severity.
	members := []MemberTopics{
		mkMember("team-a", "reader", Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "missing-context/YYYY-MM-DD"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	hits := 0
	for _, f := range r.Findings {
		if f.Rule != "unread_required" {
			continue
		}
		hits++
		if f.Severity != SeverityWarning {
			t.Errorf("unread_required must be warning severity; got %s", f.Severity)
		}
		if f.Member.Member != "reader" {
			t.Errorf("unread_required should attribute to declaring member; got %v", f.Member)
		}
		if f.Prefix != "missing-context/YYYY-MM-DD" {
			t.Errorf("unread_required prefix should match the unmatched required_read; got %q", f.Prefix)
		}
	}
	if hits != 1 {
		t.Errorf("expected one unread_required finding; got %d (findings=%v)", hits, r.Findings)
	}
}

func TestRule_UnreadRequired_PeerProducerSatisfies(t *testing.T) {
	// Any member's output[] overlap suppresses the warning, even if the
	// producer is on a different team.
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "shared-context/*", DestinationKind: DestinationKnowledge}},
		}),
		mkMember("team-b", "reader", Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "shared-context/2026-04-23"}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "unread_required" {
			t.Errorf("peer producer should satisfy required_read; got %v", f)
		}
	}
}

func TestRule_UnreadRequired_OwnOutputSatisfies(t *testing.T) {
	// A member is allowed to read its own past outputs as required-context.
	// The consumer-search walks every member's output[], including the
	// declaring member's own.
	members := []MemberTopics{
		mkMember("marketing-crew", "researcher", Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "audience-scan/2026-04-23"}},
			Output:       []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "unread_required" {
			t.Errorf("own-output should satisfy own required_read; got %v", f)
		}
	}
}

func TestRule_UnreadRequired_WildcardSourceTeamSkipsCheck(t *testing.T) {
	// source_team == "*" on a required_read entry declares a universal-source
	// read: any team may write. The check is skipped for the same reason
	// orphan_input skips wildcard intakes — the topology is intentionally
	// open. ruleWildcardSourceMisuse covers the missing-anchor case
	// independently for intake, but not (today) for required_read; tightening
	// that is out-of-scope for P1.6 and tracked as future work.
	wildcard := SourceTeamWildcard
	members := []MemberTopics{
		mkMember("meta-optimization", "skill-optimizer", Topics{
			RequiredRead: []RequiredReadEntry{{
				Prefix:     "skill-visited/<skill-id>",
				SourceTeam: &wildcard,
			}},
		}),
	}
	r := Validate(members, ValidationOptions{})
	for _, f := range r.Findings {
		if f.Rule == "unread_required" {
			t.Errorf("source_team=\"*\" should suppress unread_required; got %v", f)
		}
	}
}

func TestRule_UnreadRequired_ExternalProducersDoesNotSuppress(t *testing.T) {
	// Member-level external_producers is intentionally NOT a satisfying
	// anchor for unread_required. The rule's purpose is to surface drift
	// between declared read prefixes and declared write prefixes; the
	// loose external_producers skip used by ruleOrphanInput would mask
	// exactly the drift cases the plan calls out (vision-walk-prep,
	// outcome-snapshot, portfolio-snapshot, deep-audit). Severity is
	// warning so the operator can either rename to match a producer
	// (Phase 1.7) or accept the warning as documenting an external-only
	// write.
	members := []MemberTopics{
		mkMember("director-swarm", "vision-walk-prep", Topics{
			RequiredRead:      []RequiredReadEntry{{Prefix: "vision-walk/<date>/<slug>"}},
			ExternalProducers: []string{"operator", "morning-vision-walk"},
		}),
	}
	r := Validate(members, ValidationOptions{})
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "unread_required" {
			hits++
		}
	}
	if hits != 1 {
		t.Errorf("external_producers must NOT suppress unread_required; expected 1 finding, got %d (findings=%v)", hits, r.Findings)
	}
}

func TestRule_UnreadRequired_MultipleEntriesEachReported(t *testing.T) {
	// Each unmatched required_read[] entry produces its own finding so the
	// operator can reconcile per-prefix rather than chasing a single
	// aggregated message.
	members := []MemberTopics{
		mkMember("team-a", "reader", Topics{
			RequiredRead: []RequiredReadEntry{
				{Prefix: "missing-a/*"},
				{Prefix: "missing-b/*"},
			},
		}),
	}
	r := Validate(members, ValidationOptions{})
	prefixes := map[string]bool{}
	for _, f := range r.Findings {
		if f.Rule == "unread_required" {
			prefixes[f.Prefix] = true
		}
	}
	if !prefixes["missing-a/*"] || !prefixes["missing-b/*"] {
		t.Errorf("each unmatched required_read should fire its own finding; got %v", prefixes)
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

// Pre-P4.0 this file held three TestRule_NonPortableClassifier_* tests
// that exercised a substring-coupling rule against a synthetic
// classifier skill. P4.0 retired ruleNonPortableClassifier in favor of
// ruleProseTopicLeak's broader, file-walking coverage; the subsumption
// proof lives in non_portable_classifier_subsumption_test.go and the
// belt-and-suspenders live-store check in classifier_purity_test.go.

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
	// taxonomy rules. dangling_por_sink will fire only if a member declares
	// a por_file destination with a missing path. dangling_evidence_decision
	// relies on the team-contract registry, which the canary loads via
	// StoreDir lazy-load. Pillar 2 prose-scan coverage is exercised
	// separately by TestClassifierPurity_RegisteredClassifiers_NoProseTopicLeak.
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	repoRoot := filepath.Join(storeDir, "..", "..", "..")
	repoRoot, _ = filepath.Abs(repoRoot)
	r := Validate(members, ValidationOptions{
		RepoRoot: repoRoot,
		StoreDir: storeDir,
	})
	if r.Errors > 0 {
		for _, f := range r.Findings {
			t.Logf("[%s] %s %s %s", f.Severity, f.Rule, f.Member, f.Detail)
		}
		t.Errorf("real-store validation produced %d errors and %d warnings", r.Errors, r.Warnings)
	}
}

func TestRule_DanglingEvidenceDecision_Resolves(t *testing.T) {
	// A correctly-declared evidence_consumed entry whose for_decisions ids
	// resolve in the registry must NOT trip the rule.
	members := []MemberTopics{
		mkMember("monetization", "catalog-strategist", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "candidate-sku-record/*",
				ForDecisions: []string{"catalog-promotion", "sku-retirement"},
			}},
		}),
	}
	opts := ValidationOptions{
		TeamContracts: TeamContractRegistry{
			"monetization": {TeamID: "monetization", Contract: stubContract("catalog-promotion", "sku-retirement")},
		},
	}
	r := Validate(members, opts)
	for _, f := range r.Findings {
		if f.Rule == "dangling_evidence_decision" {
			t.Errorf("declared decision-context should not trip rule; got %v", f)
		}
	}
}

func TestRule_DanglingEvidenceDecision_Dangles(t *testing.T) {
	// A typo or removed decision-context id surfaces as a structural error.
	members := []MemberTopics{
		mkMember("monetization", "catalog-strategist", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "candidate-sku-record/*",
				ForDecisions: []string{"catalog-promotion", "ghost-decision"},
			}},
		}),
	}
	opts := ValidationOptions{
		TeamContracts: TeamContractRegistry{
			"monetization": {TeamID: "monetization", Contract: stubContract("catalog-promotion")},
		},
	}
	r := Validate(members, opts)
	hits := 0
	for _, f := range r.Findings {
		if f.Rule != "dangling_evidence_decision" {
			continue
		}
		hits++
		if f.Severity != SeverityError {
			t.Errorf("dangling_evidence_decision must be error severity; got %s", f.Severity)
		}
		if !strings.Contains(f.Detail, "ghost-decision") {
			t.Errorf("finding detail should name the dangling id; got %q", f.Detail)
		}
		if f.Member.Member != "catalog-strategist" {
			t.Errorf("finding should attribute to declaring member; got %v", f.Member)
		}
	}
	if hits != 1 {
		t.Errorf("expected exactly one dangling_evidence_decision finding; got %d", hits)
	}
}

func TestRule_DanglingEvidenceDecision_ResolvesAcrossTeams(t *testing.T) {
	// The rule's resolution semantics (per plan): "exists somewhere in the
	// team graph." A member referencing a decision-context owned by
	// another team is allowed — the registry is a global pool.
	members := []MemberTopics{
		mkMember("scenario-qa", "qa-contrarian", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "challenge-report/*",
				ForDecisions: []string{"audience-update"}, // owned by marketing-crew
			}},
		}),
	}
	opts := ValidationOptions{
		TeamContracts: TeamContractRegistry{
			"marketing-crew": {TeamID: "marketing-crew", Contract: stubContract("audience-update")},
		},
	}
	r := Validate(members, opts)
	for _, f := range r.Findings {
		if f.Rule == "dangling_evidence_decision" {
			t.Errorf("cross-team decision-context resolution should be allowed; got %v", f)
		}
	}
}

func TestRule_DanglingEvidenceDecision_DeduplicatesPerEntry(t *testing.T) {
	// If the same dangling id appears twice on one entry's for_decisions,
	// emit only one finding so the operator's log isn't flooded.
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "ledger/*",
				ForDecisions: []string{"ghost", "ghost"},
			}},
		}),
	}
	opts := ValidationOptions{
		TeamContracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: stubContract()},
		},
	}
	r := Validate(members, opts)
	hits := 0
	for _, f := range r.Findings {
		if f.Rule == "dangling_evidence_decision" {
			hits++
		}
	}
	if hits != 1 {
		t.Errorf("expected one finding even with duplicate ids; got %d", hits)
	}
}

func TestRule_DanglingEvidenceDecision_SkipsWhenRegistryEmpty(t *testing.T) {
	// Without any team-contract registry, the rule has no ground truth to
	// cross-check against — silently skip rather than flag everything.
	// This keeps unit tests that don't fixture contracts honest.
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "ledger/*",
				ForDecisions: []string{"any-id"},
			}},
		}),
	}
	r := Validate(members, ValidationOptions{}) // no registry, no StoreDir
	for _, f := range r.Findings {
		if f.Rule == "dangling_evidence_decision" {
			t.Errorf("rule should be skipped without registry; got %v", f)
		}
	}
}

func TestRule_DanglingEvidenceDecision_LazyLoadFromStoreDir(t *testing.T) {
	// When TeamContracts is nil but StoreDir is set, Validate lazy-loads.
	// This is the canonical configuration the CLI uses.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "team-a"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "teams", "team-a", "team.json"),
		[]byte(`{"id":"team-a","operatingContract":{"schemaVersion":1,"decisionContexts":{"declared":{"description":"x"}}}}`),
		0o644,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			EvidenceConsumed: []EvidenceConsumedEntry{
				{Prefix: "good/*", ForDecisions: []string{"declared"}},
				{Prefix: "bad/*", ForDecisions: []string{"undeclared"}},
			},
		}),
	}
	r := Validate(members, ValidationOptions{StoreDir: root})

	hits := 0
	for _, f := range r.Findings {
		if f.Rule != "dangling_evidence_decision" {
			continue
		}
		hits++
		if !strings.Contains(f.Detail, "undeclared") {
			t.Errorf("rule fired on declared id: %v", f)
		}
	}
	if hits != 1 {
		t.Errorf("expected exactly one dangling finding; got %d", hits)
	}
}

func TestRule_DanglingEvidenceDecision_IgnoresMembersWithoutEvidence(t *testing.T) {
	// Members whose topics.json has no evidence_consumed[] entries
	// must not cause findings even when the registry is empty.
	members := []MemberTopics{
		mkMember("team-a", "writer", Topics{
			Output: []OutputEntry{{Prefix: "x/*", DestinationKind: DestinationKnowledge}},
		}),
	}
	opts := ValidationOptions{
		TeamContracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: stubContract()},
		},
	}
	r := Validate(members, opts)
	for _, f := range r.Findings {
		if f.Rule == "dangling_evidence_decision" {
			t.Errorf("member without evidence_consumed should not trigger rule; got %v", f)
		}
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
