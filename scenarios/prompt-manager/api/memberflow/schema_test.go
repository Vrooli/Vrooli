package memberflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestTopicsValidate_ValidCanonical(t *testing.T) {
	// The canonical worked example from docs/agent-system/TOPICS_SCHEMA.md
	raw := `{
		"intake": [
			{"prefix": "research-inbox/*", "taxonomy": "marketing-research", "classifier_skill": "marketing-signal-classifier", "source_team": null}
		],
		"output": [
			{"prefix": "audience-scan/*", "destination_kind": "knowledge", "destination_team": null, "schema": "audience-scan"},
			{"prefix": "monetization-benchmark-adjacent/*", "destination_kind": "knowledge", "destination_team": "monetization", "schema": "monetization-benchmark-adjacent"}
		],
		"decisions_owned": ["audience-update", "channel-strategy-update"],
		"decisions_consumed": ["capability-gap"],
		"raises_capability_gaps": true,
		"external_producers": ["vision-walk", "operator"]
	}`
	var topics Topics
	if err := json.Unmarshal([]byte(raw), &topics); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if err := topics.Validate(); err != nil {
		t.Fatalf("Validate() returned error on canonical example: %v", err)
	}
	if topics.IsEmpty() {
		t.Errorf("IsEmpty() = true on canonical example")
	}
	if topics.Intake[0].Taxonomy != "marketing-research" {
		t.Errorf("taxonomy field did not round-trip: %+v", topics.Intake[0])
	}
	if topics.Output[0].Schema != "audience-scan" {
		t.Errorf("schema field did not round-trip: %+v", topics.Output[0])
	}
}

func TestTopics_LegacyDrainedBySkill_IsIgnored(t *testing.T) {
	// Phase I: drained_by_skill has been removed from the struct. Older
	// topics.json files that still carry the field unmarshal cleanly
	// (the JSON decoder ignores unknown keys); Topics.Validate doesn't
	// look at the field. The validator's missing_taxonomy rule is the
	// signal that surfaces such files for migration.
	raw := `{
		"intake": [
			{"prefix": "x/*", "drained_by_skill": "legacy-router"}
		]
	}`
	var topics Topics
	if err := json.Unmarshal([]byte(raw), &topics); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if err := topics.Validate(); err != nil {
		t.Errorf("shape validation should still pass with extra legacy keys: %v", err)
	}
	if topics.Intake[0].Taxonomy != "" {
		t.Errorf("legacy field must not populate taxonomy: %+v", topics.Intake[0])
	}
}

func TestTopicsValidate_EmptyObjectIsValid(t *testing.T) {
	var topics Topics
	if err := json.Unmarshal([]byte(`{}`), &topics); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if err := topics.Validate(); err != nil {
		t.Errorf("Validate() rejected empty object: %v", err)
	}
	if !topics.IsEmpty() {
		t.Errorf("IsEmpty() = false on empty object")
	}
}

func TestIntakeValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   IntakeEntry
		wantErr string
	}{
		{
			name:    "missing prefix",
			entry:   IntakeEntry{Taxonomy: "tx"},
			wantErr: "prefix is required",
		},
		{
			name:    "whitespace prefix",
			entry:   IntakeEntry{Prefix: "  ", Taxonomy: "tx"},
			wantErr: "prefix is required",
		},
		{
			name:    "bare star prefix",
			entry:   IntakeEntry{Prefix: "*", Taxonomy: "tx"},
			wantErr: "malformed",
		},
		{
			name:    "inner star prefix",
			entry:   IntakeEntry{Prefix: "foo/*/bar", Taxonomy: "tx"},
			wantErr: "malformed",
		},
		{
			name:  "valid wildcard with taxonomy",
			entry: IntakeEntry{Prefix: "research-inbox/*", Taxonomy: "marketing-research", ClassifierSkill: "marketing-signal-classifier"},
		},
		{
			name:  "valid exact prefix without classifier",
			entry: IntakeEntry{Prefix: "research-inbox/audience/foo", Taxonomy: "marketing-research"},
		},
		{
			name:  "valid intake with no taxonomy is shape-clean (transitional)",
			entry: IntakeEntry{Prefix: "x/*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics := Topics{Intake: []IntakeEntry{tt.entry}}
			err := topics.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() returned %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() returned nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestOutputValidation(t *testing.T) {
	tests := []struct {
		name    string
		entry   OutputEntry
		wantErr string
	}{
		{
			name:    "missing prefix",
			entry:   OutputEntry{DestinationKind: DestinationKnowledge},
			wantErr: "prefix is required",
		},
		{
			name:    "missing destination kind",
			entry:   OutputEntry{Prefix: "foo/*"},
			wantErr: "destination_kind",
		},
		{
			name:    "unknown destination kind",
			entry:   OutputEntry{Prefix: "foo/*", DestinationKind: DestinationKind("nope")},
			wantErr: "destination_kind",
		},
		{
			name: "por_file without destination_path",
			entry: OutputEntry{
				Prefix:          "doctrine/*",
				DestinationKind: DestinationPORFile,
			},
			wantErr: "destination_path is required",
		},
		{
			name: "por_file with empty destination_path",
			entry: OutputEntry{
				Prefix:          "doctrine/*",
				DestinationKind: DestinationPORFile,
				DestinationPath: ptr("   "),
			},
			wantErr: "destination_path is required",
		},
		{
			name: "valid por_file",
			entry: OutputEntry{
				Prefix:          "doctrine/*",
				DestinationKind: DestinationPORFile,
				DestinationPath: ptr("docs/agent-system/PRIMITIVES.md"),
			},
		},
		{
			name: "valid knowledge with schema",
			entry: OutputEntry{
				Prefix:          "audience-scan/*",
				DestinationKind: DestinationKnowledge,
				Schema:          "audience-scan",
			},
		},
		{
			name: "valid cross-team knowledge",
			entry: OutputEntry{
				Prefix:          "monetization-benchmark-adjacent/*",
				DestinationKind: DestinationKnowledge,
				DestinationTeam: ptr("monetization"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics := Topics{Output: []OutputEntry{tt.entry}}
			err := topics.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Validate() returned %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate() returned nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate() error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateAllReturnsAllErrors(t *testing.T) {
	topics := Topics{
		Intake: []IntakeEntry{
			{Prefix: "", Taxonomy: "x"},
			{Prefix: "*", Taxonomy: "x"},
		},
		Output: []OutputEntry{
			{Prefix: "out/*", DestinationKind: DestinationKind("bogus")},
		},
	}
	errs := topics.ValidateAll()
	if len(errs) != 3 {
		t.Errorf("ValidateAll() returned %d errors, want 3", len(errs))
		for _, e := range errs {
			t.Logf("  - %v", e)
		}
	}
}

func TestDestinationKindValid(t *testing.T) {
	valid := []DestinationKind{
		DestinationKnowledge,
		DestinationDecision,
		DestinationPORFile,
		DestinationCapabilityGap,
		DestinationSkillProposal,
		DestinationBacklog,
	}
	for _, k := range valid {
		if !k.Valid() {
			t.Errorf("%q.Valid() = false, want true", k)
		}
	}
	invalid := []DestinationKind{"", "knowledge ", "Knowledge", "skill", "doc"}
	for _, k := range invalid {
		if k.Valid() {
			t.Errorf("%q.Valid() = true, want false", k)
		}
	}
}

func TestOverlap(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		// Equal exact prefixes
		{"foo/bar", "foo/bar", true},
		// Equal wildcards
		{"foo/*", "foo/*", true},
		// Wildcard contains exact
		{"foo/*", "foo/bar", true},
		{"foo/bar", "foo/*", true},
		// Nested wildcards
		{"foo/*", "foo/bar/*", true},
		{"foo/bar/*", "foo/*", true},
		// Disjoint exact prefixes
		{"foo/bar", "foo/baz", false},
		// Disjoint wildcards
		{"foo/bar/*", "foo/baz/*", false},
		// Disjoint at root
		{"foo/*", "bar/*", false},
		// Wildcard does not overlap something with shared prefix but no separator
		{"foo/*", "fooBar", false},
		{"foo/*", "foobar/*", false},
	}
	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			if got := Overlap(tt.a, tt.b); got != tt.want {
				t.Errorf("Overlap(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIntakeRoundTrip_SourceTeamWildcard(t *testing.T) {
	// "*" is a first-class source_team value declaring universal-source
	// intake (any team's members may write). Shape validation must accept
	// it; cross-graph validation handles the orphan_input skip and the
	// paired wildcard_source_misuse warning (covered in validation_test.go).
	raw := `{
		"intake": [
			{"prefix": "bug-inbox/*", "taxonomy": "bug-report", "source_team": "*"}
		],
		"external_producers": ["report-bug-skill"]
	}`
	var topics Topics
	if err := json.Unmarshal([]byte(raw), &topics); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if err := topics.Validate(); err != nil {
		t.Errorf("Validate() rejected source_team=\"*\": %v", err)
	}
	if topics.Intake[0].SourceTeam == nil || *topics.Intake[0].SourceTeam != SourceTeamWildcard {
		t.Errorf("source_team=\"*\" did not round-trip: %+v", topics.Intake[0])
	}
}

func TestRoundTripJSON(t *testing.T) {
	original := Topics{
		Intake: []IntakeEntry{
			{Prefix: "research-inbox/*", Taxonomy: "marketing-research", ClassifierSkill: "marketing-signal-classifier", SourceTeam: nil},
		},
		Output: []OutputEntry{
			{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge, Schema: "audience-scan"},
			{Prefix: "doctrine/*", DestinationKind: DestinationPORFile, DestinationPath: ptr("docs/agent-system/PRIMITIVES.md")},
		},
		DecisionsOwned:       []string{"audience-update"},
		RaisesCapabilityGaps: true,
		ExternalProducers:    []string{"operator"},
	}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var roundTrip Topics
	if err := json.Unmarshal(b, &roundTrip); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if err := roundTrip.Validate(); err != nil {
		t.Errorf("round-trip Validate(): %v", err)
	}
	if len(roundTrip.Intake) != 1 || roundTrip.Intake[0].Taxonomy != "marketing-research" {
		t.Errorf("intake round-trip lost data: %+v", roundTrip.Intake)
	}
	if roundTrip.Intake[0].ClassifierSkill != "marketing-signal-classifier" {
		t.Errorf("classifier_skill round-trip lost data: %+v", roundTrip.Intake[0])
	}
	if len(roundTrip.Output) != 2 {
		t.Errorf("output round-trip wrong length: %d", len(roundTrip.Output))
	}
	if roundTrip.Output[0].Schema != "audience-scan" {
		t.Errorf("output[0].schema round-trip lost data: %+v", roundTrip.Output[0])
	}
}
