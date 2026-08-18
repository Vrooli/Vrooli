package harness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// surfaces are the two documents that teach this runtime to a reader, as
// opposed to the brief, which teaches it to a model. All three must agree.
func surfaces(t *testing.T) map[string]string {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	paths := map[string]string{
		"construction guide": filepath.Join(repoRoot, "scenarios", "program-runtime", "docs", "guides", "program-construction.md"),
		"skill":              filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "skills", "packs", "core", "program-runtime", "SKILL.md"),
	}
	loaded := make(map[string]string, len(paths))
	for name, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s at %s: %v", name, path, err)
		}
		// Probes are matched against whitespace-collapsed text so a prose
		// reflow cannot fail the check. The rule is about whether a surface
		// covers a concept, not about where its lines wrap.
		loaded[name] = strings.Join(strings.Fields(string(data)), " ")
	}
	return loaded
}

// TestNoRuleIsMissingFromAnySurface is the drift check.
//
// It is the mechanical form of "a rule must not exist in one surface and be
// absent from the others" — the failure that let the brief omit `gather`'s
// callable contract while the guide and skill both documented it, costing two
// of twelve authoring-eval cases.
func TestNoRuleIsMissingFromAnySurface(t *testing.T) {
	contract := Load()
	documents := surfaces(t)
	for _, rule := range contract.Rules {
		t.Run(rule.ID, func(t *testing.T) {
			if strings.TrimSpace(rule.Brief) == "" {
				t.Fatal("rule has no brief text; the model would never be told")
			}
			if strings.TrimSpace(rule.DocProbe) == "" || strings.TrimSpace(rule.SkillProbe) == "" {
				t.Fatal("rule declares no probe, so drift in the docs cannot be detected")
			}
			if !strings.Contains(documents["construction guide"], rule.DocProbe) {
				t.Fatalf("construction guide does not cover this rule: no match for %q.\n"+
					"The brief teaches a model something the guide does not teach a reader.", rule.DocProbe)
			}
			if !strings.Contains(documents["skill"], rule.SkillProbe) {
				t.Fatalf("skill does not cover this rule: no match for %q.\n"+
					"The brief teaches a model something the skill does not teach an agent.", rule.SkillProbe)
			}
		})
	}
}

func TestRuleIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, rule := range Load().Rules {
		if seen[rule.ID] {
			t.Fatalf("duplicate rule id %q", rule.ID)
		}
		seen[rule.ID] = true
	}
}

// TestInstructionCarriesEveryRule: the brief is generated, so a rule cannot be
// declared and then silently left out of what the model actually reads.
func TestInstructionCarriesEveryRule(t *testing.T) {
	contract := Load()
	instruction := contract.Instruction()
	for _, rule := range contract.Rules {
		if !strings.Contains(instruction, rule.Brief) {
			t.Fatalf("rule %q is declared but absent from the rendered brief", rule.ID)
		}
	}
	if !strings.Contains(instruction, contract.Preamble) {
		t.Fatal("rendered brief lost its preamble")
	}
	if !strings.Contains(instruction, contract.Closing) {
		t.Fatal("rendered brief lost its closing instruction")
	}
}

// TestBriefCoversTheMeasuredFailureModes pins the specific gaps that the
// 1-of-12 baseline exposed. Each of these was a real authoring failure whose
// cause was an omission in the brief rather than a limit of the model.
func TestBriefCoversTheMeasuredFailureModes(t *testing.T) {
	instruction := Load().Instruction()
	for _, expectation := range []struct {
		needle string
		why    string
	}{
		{"import vrooli", "two cases wrote `import vrooli`; the brief never said there is no module"},
		{"zero-argument callables", "two cases called gather with strings or a list"},
		{"`return` at the top level", "one case used a bare top-level return"},
		{"never a runtime verb", "two cases prefixed a verb with vrooli."},
		{"--async", "the only escape from the synchronous bound was undocumented"},
	} {
		if !strings.Contains(instruction, expectation.needle) {
			t.Fatalf("brief does not mention %q: %s", expectation.needle, expectation.why)
		}
	}
}

func TestStampIdentifiesTheHarnessVersion(t *testing.T) {
	stamp := Load().Stamp()
	if !strings.HasPrefix(stamp, "authoring-brief@") {
		t.Fatalf("stamp %q does not identify the brief version", stamp)
	}
}
