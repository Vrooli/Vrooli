package commandref

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	measures "github.com/vrooli/measures-go"

	"cli-health/internal/aisearch"
)

// fakeSchemas is a stub ParamSchemaReader keyed by "Service.Method".
type fakeSchemas struct {
	params map[string][]measures.ParamSchema
}

func (f fakeSchemas) RequestParams(service, method string) ([]measures.ParamSchema, error) {
	if p, ok := f.params[service+"."+method]; ok {
		return p, nil
	}
	return nil, &notFoundErr{}
}

type notFoundErr struct{}

func (*notFoundErr) Error() string { return "not found" }

func i64(n int64) *int64   { return &n }
func u64(n uint64) *uint64 { return &n }
func docsReq(text string) Request {
	return Request{CommandText: text, Policy: "COMMAND_REFERENCE_POLICY_DOCS"}
}

// planManagerService builds a Service whose catalog mirrors the plan-manager
// lighthouse: author skill-pack with a session positional and concepts +
// complexity (values + synonyms) flags.
func planManagerService(schemas ParamSchemaReader) Service {
	return Service{
		Schemas: schemas,
		Discovery: fakeDiscovery{
			scenarios: []string{"plan-manager"},
			records: map[string][]aisearch.CommandRecord{
				"plan-manager": {{
					Origin:   "plan-manager",
					Group:    "author",
					Name:     "skill-pack",
					FullPath: "plan-manager author skill-pack",
					Source:   aisearch.SourceManifest,
					Binding:  "PlanAuthoringService.SkillPack",
					Args: &cliapp.ArgSchema{
						Positionals: []cliapp.Positional{{Name: "session", Required: true}},
						Flags: []cliapp.Flag{
							{Name: "concepts"},
							{
								Name:         "complexity",
								Values:       []string{"minor", "moderate", "major", "architectural"},
								ValueAliases: map[string]string{"low": "minor", "medium": "moderate", "high": "major"},
							},
						},
					},
				}},
			},
		},
	}
}

// TestDocsPolicySkillMdSnippetValidates pins the lighthouse fixture: the exact
// implementation-plan-authoring SKILL.md snippet reaches
// argument_shape_validated with its enum alternation checked.
func TestDocsPolicySkillMdSnippetValidates(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack "<session>" --concepts "<c1>,<c2>" --complexity "<minor|moderate|major|architectural>"`,
	))
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
	if got.Level != LevelArgumentShapeValidated {
		t.Fatalf("level = %s, want %s", got.Level, LevelArgumentShapeValidated)
	}
	if len(got.Issues) != 0 {
		t.Fatalf("issues = %+v, want none", got.Issues)
	}
}

func TestDocsPolicyEnumPlaceholderMismatch(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack "<session>" --complexity "<minor|moderate|major>"`,
	))
	if got.Verdict != VerdictInvalid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictInvalid, got.Issues)
	}
	var found *Issue
	for i := range got.Issues {
		if got.Issues[i].Code == issueEnumPlaceholderMismatch {
			found = &got.Issues[i]
		}
	}
	if found == nil {
		t.Fatalf("issues = %+v, want %s", got.Issues, issueEnumPlaceholderMismatch)
	}
	if !strings.Contains(found.Message, "<architectural|major|minor|moderate>") {
		t.Fatalf("message %q must name the expected vocabulary", found.Message)
	}
	if found.Severity != "error" {
		t.Fatalf("severity = %s, want error", found.Severity)
	}
}

func TestDocsPolicyEnumPlaceholderOrderInsensitive(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack "<session>" --complexity "<architectural|major|moderate|minor>"`,
	))
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
}

func TestDocsPolicyNamedPlaceholderMismatchIsWarning(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack "<execution>"`,
	))
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want %s — name mismatch is a warning (%+v)", got.Verdict, VerdictValid, got.Issues)
	}
	if got.Level != LevelArgumentShapeValidated {
		t.Fatalf("level = %s, want %s", got.Level, LevelArgumentShapeValidated)
	}
	if len(got.Issues) != 1 || got.Issues[0].Code != issuePlaceholderNameMismatch || got.Issues[0].Severity != "warning" {
		t.Fatalf("issues = %+v, want one %s warning", got.Issues, issuePlaceholderNameMismatch)
	}
}

func TestDocsPolicyNamedPlaceholderMatchesFlagValueSlot(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack "<session>" --concepts "<concepts>"`,
	))
	if got.Verdict != VerdictValid || len(got.Issues) != 0 {
		t.Fatalf("verdict = %s issues = %+v, want valid with no issues", got.Verdict, got.Issues)
	}
}

func TestDocsPolicyLiteralVocabularyChecks(t *testing.T) {
	svc := planManagerService(nil)

	t.Run("declared value passes", func(t *testing.T) {
		got := svc.Validate(context.Background(), docsReq(
			`plan-manager author skill-pack "<session>" --complexity moderate`,
		))
		if got.Verdict != VerdictValid {
			t.Fatalf("verdict = %s (%+v)", got.Verdict, got.Issues)
		}
	})

	t.Run("synonym passes", func(t *testing.T) {
		got := svc.Validate(context.Background(), docsReq(
			`plan-manager author skill-pack "<session>" --complexity low`,
		))
		if got.Verdict != VerdictValid {
			t.Fatalf("verdict = %s (%+v)", got.Verdict, got.Issues)
		}
	})

	t.Run("out-of-vocabulary literal fails", func(t *testing.T) {
		got := svc.Validate(context.Background(), docsReq(
			`plan-manager author skill-pack "<session>" --complexity banana`,
		))
		if got.Verdict != VerdictInvalid {
			t.Fatalf("verdict = %s, want invalid (%+v)", got.Verdict, got.Issues)
		}
		if len(got.Issues) == 0 || got.Issues[0].Code != issueInvalidLiteralValue {
			t.Fatalf("issues = %+v, want %s", got.Issues, issueInvalidLiteralValue)
		}
		if !strings.Contains(got.Issues[0].Message, "architectural, major, minor, moderate") {
			t.Fatalf("message %q must list the vocabulary", got.Issues[0].Message)
		}
	})
}

func TestDocsPolicyTypedLiteralChecks(t *testing.T) {
	schemas := fakeSchemas{params: map[string][]measures.ParamSchema{
		"DemoService.Tune": {
			{Name: "limit", Type: "int32", Min: i64(1), Max: i64(100)},
			{Name: "ratio", Type: "double"},
			{Name: "session_id", Type: "string", Format: "uuid"},
			{Name: "note", Type: "string", MinLen: u64(2), MaxLen: u64(5)},
			{Name: "mode", Type: "enum", EnumValues: []string{"MODE_FAST", "MODE_SLOW"}},
		},
	}}
	svc := Service{
		Schemas: schemas,
		Discovery: fakeDiscovery{
			scenarios: []string{"demo"},
			records: map[string][]aisearch.CommandRecord{
				"demo": {{
					Origin:   "demo",
					FullPath: "demo tune",
					Source:   aisearch.SourceManifest,
					Binding:  "DemoService.Tune",
					Args: &cliapp.ArgSchema{Flags: []cliapp.Flag{
						{Name: "limit"},
						{Name: "ratio"},
						{Name: "session-id"},
						{Name: "note"},
						{Name: "mode"},
					}},
				}},
			},
		},
	}

	cases := []struct {
		name    string
		text    string
		valid   bool
		message string
	}{
		{name: "int in bounds", text: "demo tune --limit 50", valid: true},
		{name: "int not a number", text: "demo tune --limit abc", message: "not a valid int32"},
		{name: "int below min", text: "demo tune --limit 0", message: "below the minimum 1"},
		{name: "int above max", text: "demo tune --limit 500", message: "above the maximum 100"},
		{name: "float ok", text: "demo tune --ratio 0.5", valid: true},
		{name: "float bad", text: "demo tune --ratio half", message: "not a valid double"},
		{name: "uuid ok", text: "demo tune --session-id 1aeb7e76-2e15-486d-8742-f8cfa4eefd76", valid: true},
		{name: "uuid bad", text: "demo tune --session-id not-a-uuid", message: "not a valid uuid"},
		{name: "string too short", text: "demo tune --note x", message: "shorter than the minimum length 2"},
		{name: "string too long", text: "demo tune --note abcdef", message: "exceeds the maximum length 5"},
		{name: "proto enum member ok", text: "demo tune --mode MODE_FAST", valid: true},
		{name: "proto enum CLI short form ok", text: "demo tune --mode fast", valid: true},
		{name: "proto enum non-member", text: "demo tune --mode MODE_WARP", message: "not a member of the proto enum"},
		{name: "proto enum short-form non-member", text: "demo tune --mode warp", message: "not a member of the proto enum"},
		{name: "placeholder skips typed checks", text: `demo tune --limit "<limit>"`, valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.Validate(context.Background(), docsReq(tc.text))
			if tc.valid {
				if got.Verdict != VerdictValid {
					t.Fatalf("verdict = %s, want valid (%+v)", got.Verdict, got.Issues)
				}
				return
			}
			if got.Verdict != VerdictInvalid {
				t.Fatalf("verdict = %s, want invalid (%+v)", got.Verdict, got.Issues)
			}
			if len(got.Issues) == 0 || got.Issues[0].Code != issueInvalidLiteralValue {
				t.Fatalf("issues = %+v, want %s", got.Issues, issueInvalidLiteralValue)
			}
			if !strings.Contains(got.Issues[0].Message, tc.message) {
				t.Fatalf("message %q, want substring %q", got.Issues[0].Message, tc.message)
			}
		})
	}
}

// TestDocsPolicyEnumPlaceholderUsesProtoEnum proves the effective vocabulary
// unions manifest values with proto enum values from the binding.
func TestDocsPolicyEnumPlaceholderUsesProtoEnum(t *testing.T) {
	schemas := fakeSchemas{params: map[string][]measures.ParamSchema{
		"DemoService.Tune": {{Name: "mode", Type: "enum", EnumValues: []string{"MODE_FAST", "MODE_SLOW"}}},
	}}
	svc := Service{
		Schemas: schemas,
		Discovery: fakeDiscovery{
			scenarios: []string{"demo"},
			records: map[string][]aisearch.CommandRecord{
				"demo": {{
					Origin:   "demo",
					FullPath: "demo tune",
					Source:   aisearch.SourceManifest,
					Binding:  "DemoService.Tune",
					Args:     &cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "mode"}}},
				}},
			},
		},
	}

	good := svc.Validate(context.Background(), docsReq(`demo tune --mode "<MODE_FAST|MODE_SLOW>"`))
	if good.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want valid (%+v)", good.Verdict, good.Issues)
	}
	bad := svc.Validate(context.Background(), docsReq(`demo tune --mode "<MODE_FAST|MODE_WARP>"`))
	if bad.Verdict != VerdictInvalid || len(bad.Issues) == 0 || bad.Issues[0].Code != issueEnumPlaceholderMismatch {
		t.Fatalf("verdict = %s issues = %+v, want %s", bad.Verdict, bad.Issues, issueEnumPlaceholderMismatch)
	}
}

// TestDocsPolicyStructuralErrorsStillFail pins that DOCS keeps parser-grade
// structural validation: unknown flags and missing required positionals fail.
func TestDocsPolicyStructuralErrorsStillFail(t *testing.T) {
	svc := planManagerService(nil)

	unknown := svc.Validate(context.Background(), docsReq(`plan-manager author skill-pack "<session>" --bogus x`))
	if unknown.Verdict != VerdictInvalid {
		t.Fatalf("unknown flag verdict = %s, want invalid (%+v)", unknown.Verdict, unknown.Issues)
	}

	missing := svc.Validate(context.Background(), docsReq(`plan-manager author skill-pack`))
	if missing.Verdict != VerdictInvalid {
		t.Fatalf("missing positional verdict = %s, want invalid (%+v)", missing.Verdict, missing.Issues)
	}
}

// TestNonDocsPoliciesUnchanged pins the pre-existing behavior for every
// non-DOCS policy: quoted placeholders reach the strict runtime parser, which
// rejects the literal "<...>" string when the flag declares a vocabulary, and
// the invalid_arguments code is used — never the DOCS-only codes.
func TestNonDocsPoliciesUnchanged(t *testing.T) {
	svc := planManagerService(nil)
	for _, policy := range []string{"", "COMMAND_REFERENCE_POLICY_UNSPECIFIED", "COMMAND_REFERENCE_POLICY_SKILL", "COMMAND_REFERENCE_POLICY_PLAN", "COMMAND_REFERENCE_POLICY_ACTION"} {
		t.Run("policy="+policy, func(t *testing.T) {
			got := svc.Validate(context.Background(), Request{
				CommandText: `plan-manager author skill-pack "<session>" --complexity "<minor|moderate|major|architectural>"`,
				Policy:      policy,
			})
			if got.Verdict != VerdictInvalid {
				t.Fatalf("verdict = %s, want invalid under strict parsing (%+v)", got.Verdict, got.Issues)
			}
			if len(got.Issues) == 0 || got.Issues[0].Code != "invalid_arguments" {
				t.Fatalf("issues = %+v, want invalid_arguments", got.Issues)
			}
		})
	}
}
