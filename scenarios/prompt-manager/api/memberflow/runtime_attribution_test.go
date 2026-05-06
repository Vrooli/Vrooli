package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"prompt-manager/store"
)

// runtimeFixture is a small DSL for laying down a synthetic store under
// t.TempDir() and exercising ruleActualWriterUndeclared end-to-end. The
// alternative — golden files in testdata/runtime_drift/ — was considered
// and rejected: the assertions are about Findings shape, not file
// contents, so an in-test DSL is more readable and refactor-friendly
// than scattered fixture files. Future fixtures (e.g. a real-store
// snapshot) can sit alongside this DSL without conflict.
type runtimeFixture struct {
	storeDir string
}

func newRuntimeFixture(t *testing.T) *runtimeFixture {
	t.Helper()
	return &runtimeFixture{storeDir: t.TempDir()}
}

// writeTeam lays down store/teams/<id>/team.json with the supplied
// fields. Pass an empty cutoff to simulate "team has not adopted Pillar
// 3"; pass threshold==0 to omit the policy block (matches the on-disk
// shape for teams without an opt-in policy).
func (rf *runtimeFixture) writeTeam(t *testing.T, teamID, cutoff string, threshold int) {
	t.Helper()
	dir := filepath.Join(rf.storeDir, "teams", teamID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	body := map[string]any{
		"id": teamID,
		"operatingContract": map[string]any{
			"schemaVersion":    1,
			"decisionContexts": map[string]any{},
		},
	}
	if cutoff != "" {
		body["attributionValidFrom"] = cutoff
	}
	if threshold > 0 {
		body["policy"] = map[string]any{"flagExternalWritesPerWeek": threshold}
	}
	b, _ := json.MarshalIndent(body, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "team.json"), b, 0o644); err != nil {
		t.Fatalf("write team.json: %v", err)
	}
}

// writeMember lays down store/teams/<team>/members/<id>/topics.json with
// the supplied output prefixes, all marked DestinationKnowledge.
func (rf *runtimeFixture) writeMember(t *testing.T, teamID, memberID string, outputs ...string) {
	t.Helper()
	dir := filepath.Join(rf.storeDir, "teams", teamID, "members", memberID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	topics := Topics{}
	for _, p := range outputs {
		topics.Output = append(topics.Output, OutputEntry{
			Prefix:          p,
			DestinationKind: DestinationKnowledge,
		})
	}
	b, err := encodeTopicsJSON(topics)
	if err != nil {
		t.Fatalf("encode topics: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "topics.json"), b, 0o644); err != nil {
		t.Fatalf("write topics.json: %v", err)
	}
}

// appendKnowledge writes one knowledge entry to
// store/teams/<team>/shared/knowledge.jsonl. Caller supplies the full
// row so each test case names exactly the shape under test (kind,
// member_id, topic, at, etc.).
func (rf *runtimeFixture) appendKnowledge(t *testing.T, teamID string, row knowledgeEntryRow) {
	t.Helper()
	dir := filepath.Join(rf.storeDir, "teams", teamID, "shared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "knowledge.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	b, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("marshal entry: %v", err)
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		t.Fatalf("write entry: %v", err)
	}
}

// appendRawKnowledge writes a raw line to knowledge.jsonl without
// JSON-marshalling. Used to construct the malformed-line test case.
func (rf *runtimeFixture) appendRawKnowledge(t *testing.T, teamID, line string) {
	t.Helper()
	dir := filepath.Join(rf.storeDir, "teams", teamID, "shared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "knowledge.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line + "\n"); err != nil {
		t.Fatalf("write raw: %v", err)
	}
}

// run loads everything from the synthetic store and invokes the rule
// directly (rather than the full Validate pipeline) so the test is
// scoped to ruleActualWriterUndeclared's output without findings from
// other rules entering the assertions.
func (rf *runtimeFixture) run(t *testing.T) []Finding {
	t.Helper()
	members, err := LoadAll(rf.storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	contracts, err := LoadAllTeamContracts(rf.storeDir)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	opts := ValidationOptions{
		StoreDir:      rf.storeDir,
		TeamContracts: contracts,
	}
	return ruleActualWriterUndeclared(members, opts)
}

// strPtr is a tiny helper for AttributionInfo's pointer fields so test
// rows are readable.
func strPtr(s string) *string { return &s }

// agentMemberRow constructs a canonical post-cutoff agent-member row
// with the optional run_id (heartbeat-spawned writes have null run_id
// by contract; see docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge).
func agentMemberRow(id, at, topic, teamID, memberID, runID string) knowledgeEntryRow {
	row := knowledgeEntryRow{
		ID:    id,
		At:    at,
		Topic: topic,
		Attribution: store.AttributionInfo{
			Kind:        store.KnowledgeKindAgentMember,
			MemberID:    strPtr(memberID),
			TeamID:      strPtr(teamID),
			SpawnOrigin: store.SpawnOriginHeartbeat,
		},
	}
	if runID != "" {
		row.Attribution.RunID = strPtr(runID)
	}
	return row
}

// ---------------------------------------------------------------------
// Happy-path declared writer
// ---------------------------------------------------------------------

func TestRuntimeAttribution_AgentMemberDeclared_NoFindings(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-1", "2026-05-04T10:00:00Z", "audience-scan/2026-05-04/q2", "alpha", "researcher", ""))

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings on declared write; got %d:\n%v", len(findings), formatFindings(findings))
	}
}

// TestRuntimeAttribution_HeartbeatNullRunID_NoFindings pins the
// heartbeat-spawn handoff: heartbeat-spawned attribution carries null
// run_id by design (RUNTIME_ATTRIBUTION.md § Env-var bridge / Run-id
// resolution). The runtime scanner must accept null run_id on
// agent-member entries — it is the canonical heartbeat-spawned shape.
// If this test ever starts firing, a validator change has broken the
// contract.
func TestRuntimeAttribution_HeartbeatNullRunID_NoFindings(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")

	row := agentMemberRow("knw-null-run",
		"2026-05-04T10:00:00Z", "audience-scan/2026-05-04/null-run",
		"alpha", "researcher", "")
	if row.Attribution.RunID != nil {
		t.Fatalf("test setup error: expected nil RunID, got %v", row.Attribution.RunID)
	}
	rf.appendKnowledge(t, "alpha", row)

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("null run_id heartbeat write must not fire; got:\n%v", formatFindings(findings))
	}
}

// ---------------------------------------------------------------------
// Undeclared agent-member writer
// ---------------------------------------------------------------------

func TestRuntimeAttribution_AgentMemberUndeclared_FiresError(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	// Researcher writes to a prefix it does NOT declare on output[].
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-drift", "2026-05-04T10:00:00Z", "rogue-prefix/x", "alpha", "researcher", ""))

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding; got %d:\n%v", len(findings), formatFindings(findings))
	}
	got := findings[0]
	if got.Rule != "actual_writer_undeclared" {
		t.Errorf("Rule = %q, want %q", got.Rule, "actual_writer_undeclared")
	}
	if got.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q (agent-member output drift is concrete, not advisory)", got.Severity, SeverityError)
	}
	if got.Member != (MemberRef{Team: "alpha", Member: "researcher"}) {
		t.Errorf("Member = %v, want alpha/researcher", got.Member)
	}
	if got.Prefix != "rogue-prefix/x" {
		t.Errorf("Prefix = %q, want %q", got.Prefix, "rogue-prefix/x")
	}
	if !strings.Contains(got.Detail, "knw-drift") {
		t.Errorf("Detail should reference entry id knw-drift; got %q", got.Detail)
	}
}

func TestRuntimeAttribution_AgentMemberUnknownMember_FiresWarning(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	// Entry claims a member id that does not exist on alpha.
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-ghost", "2026-05-04T10:00:00Z", "audience-scan/x", "alpha", "ghost", ""))

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding; got %d:\n%v", len(findings), formatFindings(findings))
	}
	if findings[0].Rule != "actual_writer_undeclared" {
		t.Errorf("Rule = %q, want actual_writer_undeclared", findings[0].Rule)
	}
	if findings[0].Member.Member != "ghost" {
		t.Errorf("Member = %v, want member_id=ghost", findings[0].Member)
	}
	if !strings.Contains(findings[0].Detail, "no team member of that id exists") {
		t.Errorf("Detail should explain the unknown-member shape; got %q", findings[0].Detail)
	}
}

// ---------------------------------------------------------------------
// Pre-cutoff and legacy handling
// ---------------------------------------------------------------------

func TestRuntimeAttribution_PreCutoffEntries_Skipped(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")

	// A legacy-marked pre-cutoff entry — the canonical post-migration shape.
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:    "knw-legacy",
		At:    "2026-04-30T10:00:00Z",
		Topic: "anywhere/x",
		Attribution: store.AttributionInfo{
			Kind:        store.KnowledgeKindLegacy,
			SpawnOrigin: store.SpawnOriginLegacy,
		},
	})
	// A live-shape entry that happens to be dated pre-cutoff — should
	// also be skipped purely on date even though kind ≠ legacy.
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-pre", "2026-04-15T10:00:00Z", "rogue-prefix/x", "alpha", "researcher", ""))

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("pre-cutoff entries must be skipped; got:\n%v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_LegacyPostCutoff_FiresAttributionMalformed(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	// kind="legacy" should never appear post-cutoff (the migration tool
	// only stamps pre-existing rows). If we see one, it is a contract
	// violation worth surfacing as an error.
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:    "knw-bad-legacy",
		At:    "2026-05-10T10:00:00Z",
		Topic: "audience-scan/x",
		Attribution: store.AttributionInfo{
			Kind:        store.KnowledgeKindLegacy,
			SpawnOrigin: store.SpawnOriginLegacy,
		},
	})

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("expected 1 attribution_malformed finding; got %d:\n%v", len(findings), formatFindings(findings))
	}
	if findings[0].Rule != "attribution_malformed" {
		t.Errorf("Rule = %q, want attribution_malformed", findings[0].Rule)
	}
	if findings[0].Severity != SeverityError {
		t.Errorf("Severity = %q, want error", findings[0].Severity)
	}
}

// ---------------------------------------------------------------------
// Operator-direct, writer-skill, investigation: silently skipped (this rule's scope)
// ---------------------------------------------------------------------

func TestRuntimeAttribution_NonAgentKinds_Silent(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")

	cases := []store.AttributionInfo{
		{Kind: store.KnowledgeKindOperatorDirect, SpawnOrigin: store.SpawnOriginOperatorCLI},
		{Kind: store.KnowledgeKindWriterSkill, SourceSkillID: strPtr("report-bug"), TeamID: strPtr("alpha"), SpawnOrigin: store.SpawnOriginHeartbeat},
		{Kind: store.KnowledgeKindInvestigation, RunID: strPtr("00000000-0000-0000-0000-000000000001"), SpawnOrigin: store.SpawnOriginInvestigation},
	}
	for i, attr := range cases {
		rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
			ID:          fmt.Sprintf("knw-non-agent-%d", i),
			At:          "2026-05-10T10:00:00Z",
			Topic:       "anywhere/x",
			Attribution: attr,
		})
	}

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("non-agent-member kinds should be silent in this rule's scope; got:\n%v", formatFindings(findings))
	}
}

// ---------------------------------------------------------------------
// External-writer threshold mechanic
// ---------------------------------------------------------------------

func TestRuntimeAttribution_ExternalBelowThreshold_NoFindings(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 3) // threshold=3
	rf.writeMember(t, "alpha", "researcher")

	// Two external entries in the same ISO week (≤ threshold of 3).
	for i, at := range []string{
		"2026-05-04T08:00:00Z",
		"2026-05-05T08:00:00Z",
	} {
		rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
			ID:          fmt.Sprintf("knw-ext-%d", i),
			At:          at,
			Topic:       "external/x",
			Attribution: store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
		})
	}

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("external below threshold must not fire; got:\n%v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_ExternalAboveThreshold_FiresPerExcessEntry(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 2) // threshold=2
	rf.writeMember(t, "alpha", "researcher")

	// 4 external entries in the same ISO week (2026-W19); expect 2
	// findings (entries past index threshold).
	for i, at := range []string{
		"2026-05-04T08:00:00Z",
		"2026-05-05T08:00:00Z",
		"2026-05-06T08:00:00Z",
		"2026-05-07T08:00:00Z",
	} {
		rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
			ID:          fmt.Sprintf("knw-ext-%d", i),
			At:          at,
			Topic:       "external/x",
			Attribution: store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
		})
	}

	findings := rf.run(t)
	if len(findings) != 2 {
		t.Fatalf("expected 2 over-threshold findings; got %d:\n%v", len(findings), formatFindings(findings))
	}
	// Findings should be the third and fourth entries by `at` order.
	if !strings.Contains(findings[0].Detail, "knw-ext-2") {
		t.Errorf("findings[0] should reference knw-ext-2; got %q", findings[0].Detail)
	}
	if !strings.Contains(findings[1].Detail, "knw-ext-3") {
		t.Errorf("findings[1] should reference knw-ext-3; got %q", findings[1].Detail)
	}
	for i, f := range findings {
		if f.Rule != "actual_writer_undeclared" {
			t.Errorf("findings[%d].Rule = %q, want actual_writer_undeclared", i, f.Rule)
		}
		if f.Severity != SeverityWarning {
			t.Errorf("findings[%d].Severity = %q, want warning", i, f.Severity)
		}
		if !strings.Contains(f.Detail, "policy.flagExternalWritesPerWeek=2") {
			t.Errorf("findings[%d].Detail should cite the threshold; got %q", i, f.Detail)
		}
	}
}

func TestRuntimeAttribution_ExternalSpansWeeks_GroupsIndependently(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-04-27", 2) // threshold=2
	rf.writeMember(t, "alpha", "researcher")

	// Three entries in week W18, two in week W19. With threshold=2,
	// week W18 has one over-threshold entry; W19 has none.
	// (Verify the helper bucketing is per-week, not aggregate.)
	weekW18 := []string{"2026-04-27T08:00:00Z", "2026-04-28T08:00:00Z", "2026-04-29T08:00:00Z"}
	weekW19 := []string{"2026-05-05T08:00:00Z", "2026-05-06T08:00:00Z"}
	id := 0
	for _, at := range append(append([]string{}, weekW18...), weekW19...) {
		rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
			ID:          fmt.Sprintf("knw-ext-%d", id),
			At:          at,
			Topic:       "external/x",
			Attribution: store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
		})
		id++
	}

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("expected 1 over-threshold finding (W18 third entry); got %d:\n%v", len(findings), formatFindings(findings))
	}
	if !strings.Contains(findings[0].Detail, "ISO week 2026-W18") {
		t.Errorf("Detail should name 2026-W18; got %q", findings[0].Detail)
	}
}

func TestRuntimeAttribution_ExternalNoPolicy_TrackedNotFlagged(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0) // threshold=0 (default)
	rf.writeMember(t, "alpha", "researcher")

	for i := 0; i < 5; i++ {
		rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
			ID:          fmt.Sprintf("knw-ext-%d", i),
			At:          "2026-05-04T08:00:00Z",
			Topic:       "external/x",
			Attribution: store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
		})
	}

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("threshold=0 must never flag externals; got:\n%v", formatFindings(findings))
	}
}

// ---------------------------------------------------------------------
// Team-state edge cases
// ---------------------------------------------------------------------

func TestRuntimeAttribution_NoCutoff_TeamSkipped(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "", 0) // not yet adopted
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-x", "2026-05-04T10:00:00Z", "rogue-prefix/x", "alpha", "researcher", ""))

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("teams without attributionValidFrom must be skipped; got:\n%v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_MissingKnowledgeJSONL_NoFindings(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	// No knowledge.jsonl on disk — fresh team baseline.

	findings := rf.run(t)
	if len(findings) != 0 {
		t.Fatalf("missing knowledge.jsonl must be silent (fresh-team baseline); got:\n%v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_MalformedJSONL_FiresReadError(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendRawKnowledge(t, "alpha", `{ this is not json `)

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("malformed line should produce 1 read-error finding; got %d:\n%v", len(findings), formatFindings(findings))
	}
	if findings[0].Rule != "actual_writer_undeclared" || findings[0].Severity != SeverityWarning {
		t.Errorf("read-error finding should be a warning under actual_writer_undeclared; got rule=%q severity=%q", findings[0].Rule, findings[0].Severity)
	}
	if !strings.Contains(findings[0].Detail, "could not read team knowledge.jsonl") {
		t.Errorf("Detail should explain the read failure; got %q", findings[0].Detail)
	}
}

func TestRuntimeAttribution_MissingStoreDir_RuleSilent(t *testing.T) {
	// Calling the rule directly with empty StoreDir should produce no
	// findings (the unit-test scenario where memberflow is exercised
	// without a backing on-disk store).
	got := ruleActualWriterUndeclared(nil, ValidationOptions{})
	if got != nil {
		t.Errorf("empty options should yield nil findings; got %v", got)
	}
}

func TestRuntimeAttribution_NoTeamContracts_RuleSilent(t *testing.T) {
	// A populated StoreDir but empty TeamContracts (e.g. because every
	// team.json failed to load) should also yield no findings — the
	// rule is gated on the registry, not the store path.
	dir := t.TempDir()
	got := ruleActualWriterUndeclared(nil, ValidationOptions{StoreDir: dir})
	if got != nil {
		t.Errorf("no contracts should yield nil findings; got %v", got)
	}
}

// ---------------------------------------------------------------------
// Malformed attribution shapes (defense-in-depth; the API rejects these
// at write time, but a hand-edited file or a buggy migration could leak
// one through)
// ---------------------------------------------------------------------

func TestRuntimeAttribution_UnknownKind_FiresAttributionMalformed(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:    "knw-unknown",
		At:    "2026-05-10T10:00:00Z",
		Topic: "anywhere/x",
		Attribution: store.AttributionInfo{
			Kind:        "robot",
			SpawnOrigin: store.SpawnOriginUnknown,
		},
	})

	findings := rf.run(t)
	if len(findings) != 1 {
		t.Fatalf("unknown kind should produce 1 finding; got %d:\n%v", len(findings), formatFindings(findings))
	}
	if findings[0].Rule != "attribution_malformed" || findings[0].Severity != SeverityError {
		t.Errorf("Rule/Severity = %q/%q, want attribution_malformed/error", findings[0].Rule, findings[0].Severity)
	}
	if !strings.Contains(findings[0].Detail, "robot") {
		t.Errorf("Detail should name the unknown kind; got %q", findings[0].Detail)
	}
}

func TestRuntimeAttribution_EmptyKind_FiresAttributionMalformed(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:    "knw-empty-kind",
		At:    "2026-05-10T10:00:00Z",
		Topic: "anywhere/x",
		Attribution: store.AttributionInfo{
			Kind:        "",
			SpawnOrigin: store.SpawnOriginUnknown,
		},
	})

	findings := rf.run(t)
	if len(findings) != 1 || findings[0].Rule != "attribution_malformed" {
		t.Fatalf("empty kind should produce 1 attribution_malformed finding; got %v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_AgentMemberMissingMemberID_FiresAttributionMalformed(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	// kind=agent-member but no member_id on attribution. The API
	// rejects this at write time; on disk it would only appear via a
	// hand-edit. Defensive error.
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:    "knw-no-member",
		At:    "2026-05-10T10:00:00Z",
		Topic: "audience-scan/x",
		Attribution: store.AttributionInfo{
			Kind:        store.KnowledgeKindAgentMember,
			TeamID:      strPtr("alpha"),
			SpawnOrigin: store.SpawnOriginHeartbeat,
		},
	})

	findings := rf.run(t)
	if len(findings) != 1 || findings[0].Rule != "attribution_malformed" || findings[0].Severity != SeverityError {
		t.Fatalf("missing member_id should fire attribution_malformed/error; got %v", formatFindings(findings))
	}
}

func TestRuntimeAttribution_MalformedAtTimestampOnExternal_FiresAttributionMalformed(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 1) // threshold>0 to engage isoWeek
	rf.writeMember(t, "alpha", "researcher")
	rf.appendKnowledge(t, "alpha", knowledgeEntryRow{
		ID:          "knw-bad-ts",
		At:          "2026-05-10T10:00:00Z", // valid date prefix → not skipped pre-cutoff
		Topic:       "external/x",
		Attribution: store.AttributionInfo{Kind: store.KnowledgeKindExternal, SpawnOrigin: store.SpawnOriginUnknown},
	})
	// Now overwrite with a structurally-broken timestamp so isoWeekKey
	// fails. We rebuild the file with the broken row using raw line
	// append so the JSON itself is well-formed but the semantic ts
	// is unparseable.
	path := filepath.Join(rf.storeDir, "teams", "alpha", "shared", "knowledge.jsonl")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove %s: %v", path, err)
	}
	rf.appendRawKnowledge(t, "alpha", `{"id":"knw-bad-ts","at":"not-a-timestamp-but-passes-prefix-skip","topic":"external/x","attribution":{"kind":"external","member_id":null,"team_id":null,"run_id":null,"spawn_origin":"unknown","source_skill_id":null}}`)

	findings := rf.run(t)
	if len(findings) != 1 || findings[0].Rule != "attribution_malformed" {
		t.Fatalf("broken timestamp on bucketed entry should fire attribution_malformed; got %v", formatFindings(findings))
	}
	if !strings.Contains(findings[0].Detail, "knw-bad-ts") {
		t.Errorf("Detail should reference the entry id; got %q", findings[0].Detail)
	}
}

// ---------------------------------------------------------------------
// Validate() pipeline integration
// ---------------------------------------------------------------------

// TestRuntimeAttribution_ValidatePipeline_RuleRunsAndAggregates exercises
// the full Validate() entry point, asserting that ruleActualWriterUndeclared
// is in the rule list and its findings appear in the aggregated result.
func TestRuntimeAttribution_ValidatePipeline_RuleRunsAndAggregates(t *testing.T) {
	rf := newRuntimeFixture(t)
	rf.writeTeam(t, "alpha", "2026-05-04", 0)
	rf.writeMember(t, "alpha", "researcher", "audience-scan/*")
	rf.appendKnowledge(t, "alpha", agentMemberRow(
		"knw-drift", "2026-05-04T10:00:00Z", "rogue-prefix/x", "alpha", "researcher", ""))

	members, err := LoadAll(rf.storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	contracts, err := LoadAllTeamContracts(rf.storeDir)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	res := Validate(members, ValidationOptions{
		StoreDir:      rf.storeDir,
		TeamContracts: contracts,
	})
	if res.Warnings == 0 {
		t.Fatalf("expected at least one warning from runtime-attribution rule; got %d warnings\n%v", res.Warnings, formatFindings(res.Findings))
	}
	found := false
	for _, f := range res.Findings {
		if f.Rule == "actual_writer_undeclared" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Validate() did not include actual_writer_undeclared findings:\n%v", formatFindings(res.Findings))
	}
}

// ---------------------------------------------------------------------
// isoWeekKey direct unit tests (boundary correctness)
// ---------------------------------------------------------------------

func TestIsoWeekKey_ParsesRFC3339AndDateOnly(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-05-04T15:32:11Z", "2026-W19"}, // Monday
		{"2026-05-04", "2026-W19"},
		{"2026-01-01T00:00:00Z", "2026-W01"}, // Thursday → ISO week of its own year
		// ISO-week edges:
		// 2026-12-31 is a Thursday → ISO week 53 of 2026.
		{"2026-12-31T00:00:00Z", "2026-W53"},
		// 2027-01-01 is a Friday → ISO week 53 of 2026.
		{"2027-01-01T00:00:00Z", "2026-W53"},
		// 2027-01-04 is a Monday → ISO week 1 of 2027.
		{"2027-01-04T00:00:00Z", "2027-W01"},
	}
	for _, c := range cases {
		got, err := isoWeekKey(c.in)
		if err != nil {
			t.Errorf("isoWeekKey(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("isoWeekKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsoWeekKey_RejectsUnparseable(t *testing.T) {
	if _, err := isoWeekKey(""); err == nil {
		t.Error("empty timestamp should error")
	}
	if _, err := isoWeekKey("not-a-time"); err == nil {
		t.Error("garbage timestamp should error")
	}
}

// ---------------------------------------------------------------------
// entryDateBeforeCutoff direct unit tests
// ---------------------------------------------------------------------

func TestEntryDateBeforeCutoff(t *testing.T) {
	cases := []struct {
		at, cutoff string
		want       bool
	}{
		{"2026-05-04T10:00:00Z", "2026-05-04", false}, // same day → not before
		{"2026-05-03T23:59:59Z", "2026-05-04", true},  // day earlier
		{"2026-05-04", "2026-05-04", false},           // date-only same day
		{"2026-05-05", "2026-05-04", false},           // day later
		{"", "2026-05-04", false},                     // empty `at` → defensive: not skipped
		{"2026-05-04T10:00:00Z", "", false},           // empty cutoff → not skipped
		{"abc", "2026-05-04", false},                  // short `at` (<10) → not skipped
	}
	for _, c := range cases {
		got := entryDateBeforeCutoff(c.at, c.cutoff)
		if got != c.want {
			t.Errorf("entryDateBeforeCutoff(%q, %q) = %v, want %v", c.at, c.cutoff, got, c.want)
		}
	}
}

// ---------------------------------------------------------------------
// teamFilePolicy ↔ store.TeamPolicy drift detector
// ---------------------------------------------------------------------

// TestTeamFilePolicy_MirrorsStoreTeamPolicy is a structural drift
// detector. memberflow keeps a narrow JSON-only view of team.json's
// policy block; if api/store/models.go::TeamPolicy gains a field that
// memberflow needs to read but the JSON-only mirror is not updated, the
// validator will silently miss the new opt-in. The test compares the
// JSON-tagged fields by name to flag that case.
//
// Fields the validator has chosen NOT to consume can be excluded via
// the validatorIgnored set; today every TeamPolicy field is consumed.
func TestTeamFilePolicy_MirrorsStoreTeamPolicy(t *testing.T) {
	storePolicy := jsonFieldNames(reflect.TypeOf(store.TeamPolicy{}))
	memberflowPolicy := jsonFieldNames(reflect.TypeOf(teamFilePolicy{}))

	sort.Strings(storePolicy)
	sort.Strings(memberflowPolicy)

	if !reflect.DeepEqual(storePolicy, memberflowPolicy) {
		t.Errorf("store.TeamPolicy and memberflow.teamFilePolicy JSON fields drifted.\n  store:      %v\n  memberflow: %v\nUpdate memberflow.teamFilePolicy (and the validator that reads it) when adding a new TeamPolicy field.", storePolicy, memberflowPolicy)
	}
}

// jsonFieldNames returns the JSON field names declared on a struct
// type (taking the tag prefix before the first comma). Anonymous and
// non-tagged fields are ignored.
func jsonFieldNames(rt reflect.Type) []string {
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := tag
		if comma := strings.IndexByte(tag, ','); comma >= 0 {
			name = tag[:comma]
		}
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	return out
}

// formatFindings renders findings for failure messages so test output
// is greppable when an unexpected finding fires.
func formatFindings(fs []Finding) string {
	if len(fs) == 0 {
		return "  (no findings)"
	}
	var b strings.Builder
	for _, f := range fs {
		fmt.Fprintf(&b, "  - rule=%s severity=%s member=%s prefix=%q detail=%s\n",
			f.Rule, f.Severity, f.Member, f.Prefix, f.Detail)
	}
	return b.String()
}

// TestRuntimeAttribution_RealStoreCanary pins the runtime-attribution
// rule's invariant: it must produce zero findings on the live store
// post-cutoff. The canary is the durable signal that no future change
// has accidentally introduced drift the rule was supposed to catch —
// or, if the rule's scope changed, that the store has been updated to
// match.
//
// Scope of assertion: zero `actual_writer_undeclared` findings AND zero
// `attribution_malformed` findings. Other rules' findings (orphan_*,
// unread_required, etc.) are deliberately unconstrained here — those
// canaries live on TestValidate_RealStoreCanary in validation_test.go.
//
// Skipped when the real store is not on disk (e.g. a clean checkout in
// CI without the live fixture); the synthetic in-test cases above
// continue to cover the rule's behavior in that environment.
func TestRuntimeAttribution_RealStoreCanary(t *testing.T) {
	storeDir, _ := realPromptManagerStore(t)
	members, err := LoadAll(storeDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	contracts, err := LoadAllTeamContracts(storeDir)
	if err != nil {
		t.Fatalf("LoadAllTeamContracts: %v", err)
	}
	findings := ruleActualWriterUndeclared(members, ValidationOptions{
		StoreDir:      storeDir,
		TeamContracts: contracts,
	})
	for _, f := range findings {
		if f.Rule == "actual_writer_undeclared" || f.Rule == "attribution_malformed" {
			t.Errorf("real-store runtime-attribution finding: %s [%s] %s prefix=%q detail=%s",
				f.Rule, f.Severity, f.Member, f.Prefix, f.Detail)
		}
	}
}
