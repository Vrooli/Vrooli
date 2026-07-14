package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// quotingValidator mimics cli-health's DOCS-policy behavior: any command
// containing an unquoted <...> group yields an unquoted_placeholder issue
// whose Fix is the command with every group wrapped in double quotes.
type quotingValidator struct{}

func (quotingValidator) ValidateCommandReference(_ context.Context, req CommandReferenceRequest) (CommandReferenceResult, error) {
	cmd := strings.TrimSpace(req.CommandText)
	fixed, groups := quoteUnquotedGroups(cmd)
	result := CommandReferenceResult{CommandText: cmd, Verdict: "valid", ValidationLevel: "argument_shape_validated"}
	for range groups {
		result.Issues = append(result.Issues, CommandReferenceIssue{
			Code:     "unquoted_placeholder",
			Message:  "placeholder is unquoted",
			Severity: "warning",
			Fix:      fixed,
		})
	}
	return result, nil
}

// quoteUnquotedGroups wraps every <...> group that is not already inside
// double quotes.
func quoteUnquotedGroups(cmd string) (string, []string) {
	var b strings.Builder
	var groups []string
	inQuote := false
	runes := []rune(cmd)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '"' {
			inQuote = !inQuote
			b.WriteRune(r)
			continue
		}
		if r == '<' && !inQuote {
			end := -1
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == '>' {
					end = j
					break
				}
			}
			if end == -1 {
				b.WriteRune(r)
				continue
			}
			group := string(runes[i : end+1])
			groups = append(groups, group)
			b.WriteByte('"')
			b.WriteString(group)
			b.WriteByte('"')
			i = end
			continue
		}
		b.WriteRune(r)
	}
	return b.String(), groups
}

func writeFixtureScenario(t *testing.T, markdown string) (string, string, *Service) {
	t.Helper()
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo-scenario")
	docsDir := filepath.Join(scenarioDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(docsDir, "guide.md")
	if err := os.WriteFile(docPath, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(root, WithCommandReferenceValidator(quotingValidator{}))
	if err != nil {
		t.Fatal(err)
	}
	return scenarioDir, docPath, svc
}

const fixtureMarkdown = "# Guide\n" +
	"\n" +
	"```bash\n" +
	"demo-scenario items get <id>\n" +
	"demo-scenario tag --concepts <c1>,<c2> --complexity <minor|moderate>\n" +
	"demo-scenario clean \"<already>\" --quoted ok\n" +
	"```\n" +
	"\n" +
	"Prose mentioning <not-a-command> stays untouched.\n"

func TestPlaceholderFixDryRunAndApplyParity(t *testing.T) {
	_, docPath, svc := writeFixtureScenario(t, fixtureMarkdown)

	dry, err := svc.PlaceholderFix(context.Background(), "demo-scenario", true)
	if err != nil {
		t.Fatalf("PlaceholderFix dry-run: %v", err)
	}
	if len(dry.Files) != 1 {
		t.Fatalf("dry-run files = %+v, want exactly one", dry.Files)
	}
	if got, _ := os.ReadFile(docPath); string(got) != fixtureMarkdown {
		t.Fatal("dry-run must not write files")
	}

	applied, err := svc.PlaceholderFix(context.Background(), "demo-scenario", false)
	if err != nil {
		t.Fatalf("PlaceholderFix apply: %v", err)
	}
	if len(applied.Files) != 1 {
		t.Fatalf("apply files = %+v, want exactly one", applied.Files)
	}

	// Identical file+line selection between dry-run and apply.
	if dry.Files[0].Path != applied.Files[0].Path {
		t.Fatalf("path mismatch: dry %q vs apply %q", dry.Files[0].Path, applied.Files[0].Path)
	}
	if strings.Join(intsToStrings(dry.Files[0].Lines), ",") != strings.Join(intsToStrings(applied.Files[0].Lines), ",") {
		t.Fatalf("line selection mismatch: dry %v vs apply %v", dry.Files[0].Lines, applied.Files[0].Lines)
	}
	if dry.Files[0].After != applied.Files[0].After {
		t.Fatal("dry-run and apply must produce identical content")
	}

	got, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	want := "# Guide\n" +
		"\n" +
		"```bash\n" +
		"demo-scenario items get \"<id>\"\n" +
		"demo-scenario tag --concepts \"<c1>\",\"<c2>\" --complexity \"<minor|moderate>\"\n" +
		"demo-scenario clean \"<already>\" --quoted ok\n" +
		"```\n" +
		"\n" +
		"Prose mentioning <not-a-command> stays untouched.\n"
	if string(got) != want {
		t.Fatalf("applied content:\n%s\nwant:\n%s", got, want)
	}

	// Fenced-block line-number fidelity: lines 4 and 5 (1-based) were touched.
	if strings.Join(intsToStrings(applied.Files[0].Lines), ",") != "4,5" {
		t.Fatalf("touched lines = %v, want [4 5]", applied.Files[0].Lines)
	}
}

func TestPlaceholderFixIdempotent(t *testing.T) {
	_, docPath, svc := writeFixtureScenario(t, fixtureMarkdown)

	if _, err := svc.PlaceholderFix(context.Background(), "demo-scenario", false); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	first, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}

	second, err := svc.PlaceholderFix(context.Background(), "demo-scenario", false)
	if err != nil {
		t.Fatalf("second apply: %v", err)
	}
	if len(second.Files) != 0 {
		t.Fatalf("second apply files = %+v, want none (idempotent)", second.Files)
	}
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(first) {
		t.Fatal("second apply changed the file — fix is not idempotent")
	}
}

func TestCommandIssueFindingMapping(t *testing.T) {
	snippet := commandSnippet{File: "docs/guide.md", Line: 7, Command: "demo get <id>"}
	result := CommandReferenceResult{
		Verdict: "valid",
		Issues: []CommandReferenceIssue{
			{Code: "unquoted_placeholder", Message: "unquoted", Severity: "warning", Fix: `demo get "<id>"`},
			{Code: "enum_placeholder_mismatch", Message: "drift", Severity: "error"},
			{Code: "invalid_literal_value", Message: "bad literal", Severity: "error"},
		},
	}
	findings := commandIssueFindings(snippet, result)
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want exactly one (only unquoted_placeholder has a dedicated code)", findings)
	}
	f := findings[0]
	if f.Code != "placeholder_style" || f.Severity != SeverityWarning {
		t.Fatalf("finding = %+v, want placeholder_style warning", f)
	}
	if f.Fix != `demo get "<id>"` {
		t.Fatalf("fix = %q, want the byte-exact payload", f.Fix)
	}
	if f.Line != 7 || f.Target != "demo get <id>" {
		t.Fatalf("finding position = %+v, want line 7 target preserved", f)
	}
}

func intsToStrings(nums []int) []string {
	out := make([]string, len(nums))
	for i, n := range nums {
		out[i] = strconv.Itoa(n)
	}
	return out
}
