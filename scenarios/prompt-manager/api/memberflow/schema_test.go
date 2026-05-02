package memberflow

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestTopicsValidate_ValidCanonical(t *testing.T) {
	// The canonical worked example from docs/agent-system/drafts/topics-schema.md
	raw := `{
		"intake": [
			{"prefix": "research-inbox/*", "drained_by_skill": "marketing-research-router", "source_team": null}
		],
		"output": [
			{"prefix": "audience-scan/*", "destination_kind": "knowledge", "destination_team": null},
			{"prefix": "monetization-benchmark-adjacent/*", "destination_kind": "knowledge", "destination_team": "monetization"}
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
			entry:   IntakeEntry{DrainedBySkill: "router"},
			wantErr: "prefix is required",
		},
		{
			name:    "whitespace prefix",
			entry:   IntakeEntry{Prefix: "  ", DrainedBySkill: "router"},
			wantErr: "prefix is required",
		},
		{
			name:    "missing drain skill",
			entry:   IntakeEntry{Prefix: "foo/*"},
			wantErr: "drained_by_skill is required",
		},
		{
			name:    "bare star prefix",
			entry:   IntakeEntry{Prefix: "*", DrainedBySkill: "router"},
			wantErr: "malformed",
		},
		{
			name:    "inner star prefix",
			entry:   IntakeEntry{Prefix: "foo/*/bar", DrainedBySkill: "router"},
			wantErr: "malformed",
		},
		{
			name:  "valid wildcard",
			entry: IntakeEntry{Prefix: "research-inbox/*", DrainedBySkill: "marketing-research-router"},
		},
		{
			name:  "valid exact prefix",
			entry: IntakeEntry{Prefix: "research-inbox/audience/foo", DrainedBySkill: "marketing-research-router"},
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
			name: "valid knowledge",
			entry: OutputEntry{
				Prefix:          "audience-scan/*",
				DestinationKind: DestinationKnowledge,
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
			{Prefix: "", DrainedBySkill: "x"},
			{Prefix: "ok/*", DrainedBySkill: ""},
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

func TestRoundTripJSON(t *testing.T) {
	original := Topics{
		Intake: []IntakeEntry{
			{Prefix: "research-inbox/*", DrainedBySkill: "marketing-research-router", SourceTeam: nil},
		},
		Output: []OutputEntry{
			{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge},
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
	if len(roundTrip.Intake) != 1 || roundTrip.Intake[0].DrainedBySkill != "marketing-research-router" {
		t.Errorf("intake round-trip lost data: %+v", roundTrip.Intake)
	}
	if len(roundTrip.Output) != 2 {
		t.Errorf("output round-trip wrong length: %d", len(roundTrip.Output))
	}
}
