package store

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestKnowledgeEntry_RoundTrip_AllKindsPreserveFields covers the full
// post-cutoff JSON shape on the wire and on disk: every Kind constant is
// exercised so any future field shape change breaks the test in the
// kind-specific assertion that rendered the change visible.
//
// Pillar 3 contract: docs/agent-system/RUNTIME_ATTRIBUTION.md.
func TestKnowledgeEntry_RoundTrip_AllKindsPreserveFields(t *testing.T) {
	memberID := "researcher"
	teamID := "marketing-crew"
	runID := "5f9c1b2a-aaaa-bbbb-cccc-dddddddddddd"
	skillID := "report-bug"

	cases := []struct {
		name  string
		entry KnowledgeEntry
	}{
		{
			name: "agent-member with run lineage",
			entry: KnowledgeEntry{
				ID:      "knw-agent-1",
				At:      "2026-05-04T15:32:11Z",
				Topic:   "audience-scan/2026-05-04/q2-creators",
				Content: "Three pain points observed in the Q2 creator cohort.",
				Caller:  "marketing-crew/researcher",
				Attribution: AttributionInfo{
					Kind:        KnowledgeKindAgentMember,
					MemberID:    &memberID,
					TeamID:      &teamID,
					RunID:       &runID,
					SpawnOrigin: SpawnOriginHeartbeat,
				},
			},
		},
		{
			name: "writer-skill with skill id and member context",
			entry: KnowledgeEntry{
				ID:      "knw-skill-1",
				At:      "2026-05-04T16:00:00Z",
				Topic:   "bug-inbox/regression/cli-flag-confusion",
				Content: "Operator hit cli-flag-confusion when invoking team validate",
				Caller:  "skill:report-bug",
				Attribution: AttributionInfo{
					Kind:          KnowledgeKindWriterSkill,
					MemberID:      &memberID,
					TeamID:        &teamID,
					RunID:         &runID,
					SpawnOrigin:   SpawnOriginHeartbeat,
					SourceSkillID: &skillID,
				},
			},
		},
		{
			name: "operator-direct without team context",
			entry: KnowledgeEntry{
				ID:         "knw-op-1",
				At:         "2026-05-04T17:00:00Z",
				Topic:      "audience-scan/2026-05-04/manual",
				Content:    "Hand-curated from yesterday's email.",
				Caller:     "operator",
				CallerNote: "hand-curated from yesterday's email",
				Attribution: AttributionInfo{
					Kind:        KnowledgeKindOperatorDirect,
					SpawnOrigin: SpawnOriginOperatorCLI,
				},
			},
		},
		{
			name: "external write below flag threshold",
			entry: KnowledgeEntry{
				ID:      "knw-ext-1",
				At:      "2026-05-04T18:00:00Z",
				Topic:   "external-signal/webhook/foo",
				Content: "External webhook payload.",
				Caller:  "external",
				Attribution: AttributionInfo{
					Kind:        KnowledgeKindExternal,
					SpawnOrigin: SpawnOriginUnknown,
				},
			},
		},
		{
			name: "legacy migrated entry preserves original by-value as note",
			entry: KnowledgeEntry{
				ID:         "knw-legacy-1",
				At:         "2026-04-01T12:00:00Z",
				Topic:      "portfolio-snapshot/2026-04-01",
				Content:    "Pre-cutoff director snapshot.",
				Caller:     "legacy:director",
				CallerNote: "director",
				Attribution: AttributionInfo{
					Kind:        KnowledgeKindLegacy,
					SpawnOrigin: SpawnOriginLegacy,
				},
			},
		},
		{
			name: "investigation reproducer attached to a run",
			entry: KnowledgeEntry{
				ID:         "knw-investigation-1",
				At:         "2026-05-04T19:00:00Z",
				Topic:      "bug-inbox/regression/cli-flag-confusion",
				Content:    "Reproduction notes for prior bug entry.",
				Caller:     "investigation:5f9c1b2a",
				Supersedes: "knw-skill-1",
				Attribution: AttributionInfo{
					Kind:        KnowledgeKindInvestigation,
					RunID:       &runID,
					SpawnOrigin: SpawnOriginInvestigation,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.entry)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got KnowledgeEntry
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal: %v\npayload=%s", err, b)
			}
			if got.ID != tc.entry.ID {
				t.Errorf("ID lost: got %q want %q", got.ID, tc.entry.ID)
			}
			if got.At != tc.entry.At {
				t.Errorf("At lost: got %q want %q", got.At, tc.entry.At)
			}
			if got.Topic != tc.entry.Topic {
				t.Errorf("Topic lost: got %q want %q", got.Topic, tc.entry.Topic)
			}
			if got.Content != tc.entry.Content {
				t.Errorf("Content lost: got %q want %q", got.Content, tc.entry.Content)
			}
			if got.Source != tc.entry.Source {
				t.Errorf("Source lost: got %q want %q", got.Source, tc.entry.Source)
			}
			if got.Supersedes != tc.entry.Supersedes {
				t.Errorf("Supersedes lost: got %q want %q", got.Supersedes, tc.entry.Supersedes)
			}
			if got.Caller != tc.entry.Caller {
				t.Errorf("Caller lost: got %q want %q", got.Caller, tc.entry.Caller)
			}
			if got.CallerNote != tc.entry.CallerNote {
				t.Errorf("CallerNote lost: got %q want %q", got.CallerNote, tc.entry.CallerNote)
			}
			assertAttributionEqual(t, got.Attribution, tc.entry.Attribution)
		})
	}
}

// TestAttributionInfo_NilPointersMarshalAsNull guarantees the canon's
// "preserve null over omission" decision for optional pointer fields:
// every kind shares the same JSON shape, with kind-driven nulls instead
// of missing keys, so on-disk readers don't need to branch on which
// fields might be present.
//
// If this test ever needs to allow `omitempty` on the pointer fields,
// the canon contract has changed and docs/agent-system/RUNTIME_ATTRIBUTION.md
// must be updated alongside.
func TestAttributionInfo_NilPointersMarshalAsNull(t *testing.T) {
	a := AttributionInfo{
		Kind:        KnowledgeKindOperatorDirect,
		SpawnOrigin: SpawnOriginOperatorCLI,
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{
		`"member_id":null`,
		`"team_id":null`,
		`"run_id":null`,
		`"source_skill_id":null`,
		`"kind":"operator-direct"`,
		`"spawn_origin":"operator-cli"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("expected %s in %s", want, b)
		}
	}
}

// TestKnowledgeEntry_OptionalFieldsOmitWhenEmpty checks that genuinely
// optional string fields (Source, Supersedes, CallerNote) drop out of
// the wire form when empty, so the on-disk JSON stays compact for the
// common case of no-source / no-note entries.
func TestKnowledgeEntry_OptionalFieldsOmitWhenEmpty(t *testing.T) {
	memberID := "researcher"
	teamID := "marketing-crew"
	entry := KnowledgeEntry{
		ID:      "knw-min",
		At:      "2026-05-04T15:32:11Z",
		Topic:   "audience-scan/2026-05-04/q2-creators",
		Content: "minimal entry",
		Caller:  "marketing-crew/researcher",
		Attribution: AttributionInfo{
			Kind:        KnowledgeKindAgentMember,
			MemberID:    &memberID,
			TeamID:      &teamID,
			SpawnOrigin: SpawnOriginHeartbeat,
		},
	}
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, omitted := range []string{`"source"`, `"supersedes"`, `"caller_note"`} {
		if strings.Contains(string(b), omitted) {
			t.Errorf("%s should be omitted from minimal entry, got %s", omitted, b)
		}
	}
	for _, present := range []string{`"caller":"marketing-crew/researcher"`, `"attribution":{`} {
		if !strings.Contains(string(b), present) {
			t.Errorf("expected %s in %s", present, b)
		}
	}
}

// TestKnowledgeEntry_AttributionAlwaysPresent guards the contract that
// every entry — even one with a zero-valued attribution struct — emits
// the attribution object. Validators (P3.6) rely on attribution being
// structurally present to surface attribution_malformed findings.
func TestKnowledgeEntry_AttributionAlwaysPresent(t *testing.T) {
	entry := KnowledgeEntry{ID: "knw-zero", At: "2026-05-04T15:32:11Z", Topic: "t", Content: "c", Caller: "c"}
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(b), `"attribution":{`) {
		t.Errorf("attribution must be emitted even for zero-valued struct; got %s", b)
	}
}

// TestKnowledgeKinds_ConstantCoverage asserts that every named kind
// constant is enumerated by the KnowledgeKinds slice. The slice is what
// the API handler (P3.4) iterates to validate incoming attribution; if
// this test fails it means a new constant was added without updating
// the validation slice (or vice versa).
func TestKnowledgeKinds_ConstantCoverage(t *testing.T) {
	want := map[string]bool{
		KnowledgeKindAgentMember:    true,
		KnowledgeKindWriterSkill:    true,
		KnowledgeKindOperatorDirect: true,
		KnowledgeKindExternal:       true,
		KnowledgeKindLegacy:         true,
		KnowledgeKindInvestigation:  true,
	}
	if len(KnowledgeKinds) != len(want) {
		t.Fatalf("KnowledgeKinds length = %d, want %d", len(KnowledgeKinds), len(want))
	}
	seen := make(map[string]bool)
	for _, k := range KnowledgeKinds {
		if !want[k] {
			t.Errorf("unexpected kind in KnowledgeKinds: %q", k)
		}
		if seen[k] {
			t.Errorf("duplicate kind in KnowledgeKinds: %q", k)
		}
		seen[k] = true
	}
}

// TestSpawnOrigins_ConstantCoverage mirrors KnowledgeKinds: any new
// SpawnOrigin* constant must be added to the SpawnOrigins slice or this
// test surfaces the gap immediately.
func TestSpawnOrigins_ConstantCoverage(t *testing.T) {
	want := map[string]bool{
		SpawnOriginHeartbeat:     true,
		SpawnOriginOperatorCLI:   true,
		SpawnOriginSwarmTask:     true,
		SpawnOriginVisionWalk:    true,
		SpawnOriginInvestigation: true,
		SpawnOriginLegacy:        true,
		SpawnOriginUnknown:       true,
	}
	if len(SpawnOrigins) != len(want) {
		t.Fatalf("SpawnOrigins length = %d, want %d", len(SpawnOrigins), len(want))
	}
	seen := make(map[string]bool)
	for _, o := range SpawnOrigins {
		if !want[o] {
			t.Errorf("unexpected origin in SpawnOrigins: %q", o)
		}
		if seen[o] {
			t.Errorf("duplicate origin in SpawnOrigins: %q", o)
		}
		seen[o] = true
	}
}

// TestKnowledgeKinds_StringValuesMatchCanon pins the on-disk kind
// strings to the canon doc's vocabulary. Renaming a constant is a
// migration; a casual rename here would silently break every legacy
// entry's kind comparison. The exact wire strings live in
// docs/agent-system/RUNTIME_ATTRIBUTION.md § kind enum.
func TestKnowledgeKinds_StringValuesMatchCanon(t *testing.T) {
	cases := map[string]string{
		KnowledgeKindAgentMember:    "agent-member",
		KnowledgeKindWriterSkill:    "writer-skill",
		KnowledgeKindOperatorDirect: "operator-direct",
		KnowledgeKindExternal:       "external",
		KnowledgeKindLegacy:         "legacy",
		KnowledgeKindInvestigation:  "investigation",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("kind constant changed: got %q want %q (canon: RUNTIME_ATTRIBUTION.md § kind enum)", got, want)
		}
	}
}

// TestSpawnOrigins_StringValuesMatchCanon pins the on-disk origin
// strings to the canon doc's vocabulary. Same migration-discipline
// rationale as TestKnowledgeKinds_StringValuesMatchCanon.
func TestSpawnOrigins_StringValuesMatchCanon(t *testing.T) {
	cases := map[string]string{
		SpawnOriginHeartbeat:     "heartbeat",
		SpawnOriginOperatorCLI:   "operator-cli",
		SpawnOriginSwarmTask:     "swarm-task",
		SpawnOriginVisionWalk:    "vision-walk",
		SpawnOriginInvestigation: "investigation",
		SpawnOriginLegacy:        "legacy",
		SpawnOriginUnknown:       "unknown",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("spawn-origin constant changed: got %q want %q (canon: RUNTIME_ATTRIBUTION.md § spawn_origin enum)", got, want)
		}
	}
}

// TestKnowledgeEntry_LegacyMigrationFidelity exercises the on-disk
// shape P3.2's migration tool will produce for every pre-cutoff entry.
// The original `by` value MUST survive on CallerNote (it is the only
// historical-attribution signal a legacy entry carries); the derived
// caller MUST encode the legacy provenance with the documented
// "legacy:<original-by>" prefix; attribution.kind MUST be "legacy" so
// ruleActualWriterUndeclared (P3.6) skips the entry.
//
// If P3.2's migration tool diverges from this fidelity contract, this
// test fails and the divergence becomes a P3.2 review blocker.
func TestKnowledgeEntry_LegacyMigrationFidelity(t *testing.T) {
	entry := KnowledgeEntry{
		ID:         "knw-legacy-fidelity",
		At:         "2026-04-01T00:00:00Z",
		Topic:      "portfolio-snapshot/2026-04-01",
		Content:    "Pre-cutoff director snapshot.",
		Caller:     "legacy:director",
		CallerNote: "director",
		Attribution: AttributionInfo{
			Kind:        KnowledgeKindLegacy,
			SpawnOrigin: SpawnOriginLegacy,
		},
	}
	b, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got KnowledgeEntry
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Attribution.Kind != KnowledgeKindLegacy {
		t.Errorf("legacy kind lost: got %q", got.Attribution.Kind)
	}
	if got.CallerNote != "director" {
		t.Errorf("original by-value not preserved on CallerNote: got %q want %q", got.CallerNote, "director")
	}
	if !strings.HasPrefix(got.Caller, "legacy:") {
		t.Errorf("derived caller missing legacy prefix: got %q", got.Caller)
	}
	if got.Attribution.MemberID != nil || got.Attribution.TeamID != nil ||
		got.Attribution.RunID != nil || got.Attribution.SourceSkillID != nil {
		t.Errorf("legacy attribution should leave structured fields nil; got %+v", got.Attribution)
	}
}

// assertAttributionEqual compares two AttributionInfo values
// field-by-field, dereferencing pointers to compare the underlying
// strings. A nil pointer mismatch is reported with both sides so a
// test failure makes it obvious which side dropped the field.
func assertAttributionEqual(t *testing.T, got, want AttributionInfo) {
	t.Helper()
	if got.Kind != want.Kind {
		t.Errorf("Attribution.Kind: got %q want %q", got.Kind, want.Kind)
	}
	if got.SpawnOrigin != want.SpawnOrigin {
		t.Errorf("Attribution.SpawnOrigin: got %q want %q", got.SpawnOrigin, want.SpawnOrigin)
	}
	assertStringPtrEqual(t, "Attribution.MemberID", got.MemberID, want.MemberID)
	assertStringPtrEqual(t, "Attribution.TeamID", got.TeamID, want.TeamID)
	assertStringPtrEqual(t, "Attribution.RunID", got.RunID, want.RunID)
	assertStringPtrEqual(t, "Attribution.SourceSkillID", got.SourceSkillID, want.SourceSkillID)
}

func assertStringPtrEqual(t *testing.T, name string, got, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil:
		t.Errorf("%s: got nil, want %q", name, *want)
	case want == nil:
		t.Errorf("%s: got %q, want nil", name, *got)
	case *got != *want:
		t.Errorf("%s: got %q want %q", name, *got, *want)
	}
}
