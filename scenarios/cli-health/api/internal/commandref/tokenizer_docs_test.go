package commandref

import (
	"context"
	"errors"
	"testing"
)

func TestTokenizeCommandDocsLenientPlaceholders(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		wantTokens []string
		wantGroups []string
		wantFixed  string
	}{
		{
			name:       "single unquoted group",
			in:         "demo items get <id>",
			wantTokens: []string{"demo", "items", "get", "<id>"},
			wantGroups: []string{"<id>"},
			wantFixed:  `demo items get "<id>"`,
		},
		{
			name:       "enum alternation group keeps pipes",
			in:         "plan-manager author skill-pack <session> --complexity <minor|moderate|major|architectural>",
			wantTokens: []string{"plan-manager", "author", "skill-pack", "<session>", "--complexity", "<minor|moderate|major|architectural>"},
			wantGroups: []string{"<session>", "<minor|moderate|major|architectural>"},
			wantFixed:  `plan-manager author skill-pack "<session>" --complexity "<minor|moderate|major|architectural>"`,
		},
		{
			name:       "inline flag value group",
			in:         "demo tune --limit=<limit>",
			wantTokens: []string{"demo", "tune", "--limit=<limit>"},
			wantGroups: []string{"<limit>"},
			wantFixed:  `demo tune --limit="<limit>"`,
		},
		{
			name:       "multiple groups in one token",
			in:         "demo tag --concepts <c1>,<c2>",
			wantTokens: []string{"demo", "tag", "--concepts", "<c1>,<c2>"},
			wantGroups: []string{"<c1>", "<c2>"},
			wantFixed:  `demo tag --concepts "<c1>","<c2>"`,
		},
		{
			name:       "mixed quoted and unquoted",
			in:         `demo get "<id>" --mode <fast|slow>`,
			wantTokens: []string{"demo", "get", "<id>", "--mode", "<fast|slow>"},
			wantGroups: []string{"<fast|slow>"},
			wantFixed:  `demo get "<id>" --mode "<fast|slow>"`,
		},
		{
			name:       "quoted only emits no groups",
			in:         `demo get "<id>"`,
			wantTokens: []string{"demo", "get", "<id>"},
			wantGroups: nil,
			wantFixed:  "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens, groups, fixed, err := tokenizeCommandDocs(tc.in)
			if err != nil {
				t.Fatalf("tokenizeCommandDocs: %v", err)
			}
			if got, want := len(tokens), len(tc.wantTokens); got != want {
				t.Fatalf("tokens = %q, want %q", tokens, tc.wantTokens)
			}
			for i := range tokens {
				if tokens[i] != tc.wantTokens[i] {
					t.Fatalf("tokens = %q, want %q", tokens, tc.wantTokens)
				}
			}
			if got, want := len(groups), len(tc.wantGroups); got != want {
				t.Fatalf("groups = %+v, want %v", groups, tc.wantGroups)
			}
			for i := range groups {
				if groups[i].Text != tc.wantGroups[i] {
					t.Fatalf("groups = %+v, want %v", groups, tc.wantGroups)
				}
			}
			if fixed != tc.wantFixed {
				t.Fatalf("fixed = %q, want %q (byte-exact)", fixed, tc.wantFixed)
			}
		})
	}
}

func TestTokenizeCommandDocsHardErrors(t *testing.T) {
	cases := []string{
		"demo get <a <b>>",      // nested
		"demo get <unclosed",    // unterminated
		"demo get <>",           // empty group
		"demo list > out.txt",   // bare redirect
		"demo list | jq .",      // pipe outside group
		"demo list $(pwd)",      // substitution
		"demo list `pwd`",       // backtick
		"demo list && demo two", // chaining
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, _, _, err := tokenizeCommandDocs(in)
			if err == nil {
				t.Fatalf("tokenizeCommandDocs(%q) succeeded, want hard error", in)
			}
			var syntaxErr shellSyntaxError
			if in != "demo get <unclosed" && !errors.As(err, &syntaxErr) {
				t.Fatalf("err = %v, want shellSyntaxError", err)
			}
		})
	}
}

// TestDocsPolicyUnquotedPlaceholderFinding proves an unquoted snippet under
// DOCS policy parses, reaches the same semantic checks as its quoted form, and
// yields exactly one unquoted_placeholder finding per group carrying the
// byte-exact quoted fix.
func TestDocsPolicyUnquotedPlaceholderFinding(t *testing.T) {
	svc := planManagerService(nil)
	in := `plan-manager author skill-pack <session> --complexity <minor|moderate|major|architectural>`
	got := svc.Validate(context.Background(), docsReq(in))
	if got.Verdict != VerdictValid {
		t.Fatalf("verdict = %s, want valid (%+v)", got.Verdict, got.Issues)
	}
	if got.Level != LevelArgumentShapeValidated {
		t.Fatalf("level = %s, want %s", got.Level, LevelArgumentShapeValidated)
	}
	var style []Issue
	for _, issue := range got.Issues {
		if issue.Code == "unquoted_placeholder" {
			style = append(style, issue)
		}
	}
	if len(style) != 2 {
		t.Fatalf("issues = %+v, want exactly 2 unquoted_placeholder findings", got.Issues)
	}
	wantFix := `plan-manager author skill-pack "<session>" --complexity "<minor|moderate|major|architectural>"`
	for _, issue := range style {
		if issue.Severity != "warning" {
			t.Fatalf("severity = %s, want warning", issue.Severity)
		}
		if issue.Fix != wantFix {
			t.Fatalf("fix = %q, want %q (byte-exact)", issue.Fix, wantFix)
		}
	}
}

// TestDocsPolicyUnquotedEnumDriftStillFails proves lenient parsing does not
// weaken semantic checks: a drifted unquoted alternation is both a style
// warning and an enum mismatch error.
func TestDocsPolicyUnquotedEnumDriftStillFails(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), docsReq(
		`plan-manager author skill-pack <session> --complexity <minor|huge>`,
	))
	if got.Verdict != VerdictInvalid {
		t.Fatalf("verdict = %s, want invalid (%+v)", got.Verdict, got.Issues)
	}
	haveStyle, haveEnum := false, false
	for _, issue := range got.Issues {
		switch issue.Code {
		case "unquoted_placeholder":
			haveStyle = true
		case issueEnumPlaceholderMismatch:
			haveEnum = true
		}
	}
	if !haveStyle || !haveEnum {
		t.Fatalf("issues = %+v, want both unquoted_placeholder and %s", got.Issues, issueEnumPlaceholderMismatch)
	}
}

// TestNonDocsPolicyStaysStrictOnUnquotedPlaceholders pins that the lenient
// path is DOCS-only.
func TestNonDocsPolicyStaysStrictOnUnquotedPlaceholders(t *testing.T) {
	svc := planManagerService(nil)
	got := svc.Validate(context.Background(), Request{
		CommandText: "plan-manager author skill-pack <session>",
		Policy:      "COMMAND_REFERENCE_POLICY_SKILL",
	})
	if got.Verdict != VerdictUnsupported {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictUnsupported)
	}
	if got.Level != LevelUnsupportedSyntax {
		t.Fatalf("level = %s, want %s", got.Level, LevelUnsupportedSyntax)
	}
}
