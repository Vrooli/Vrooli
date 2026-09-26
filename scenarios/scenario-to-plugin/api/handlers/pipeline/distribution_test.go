package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateSkillDistributionFailsWithNamedMissingCommand(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "fixture")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: fixture\ndescription: fixture\nmetadata:\n  requires:\n    commands:\n      - fixture missing\n---\n\n# fixture\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(target, []byte(`{"name":"fixture","groups":[{"name":"fixture","commands":[{"name":"present"}]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := evaluateSkillDistribution(filepath.Join(skillDir, "SKILL.md"), target)
	if err != nil {
		t.Fatal(err)
	}
	if result.Distributable || len(result.MissingCommands) != 1 || result.MissingCommands[0] != "fixture missing" {
		t.Fatalf("unexpected distribution result: %+v", result)
	}
}

func TestEvaluateSkillDistributionAcceptsQualifiedCommands(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skills", "fixture")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: fixture\ndescription: fixture\nmetadata:\n  requires:\n    commands:\n      - fixture present\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(target, []byte(`{"name":"fixture","groups":[{"name":"fixture","commands":[{"name":"present"}]}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := evaluateSkillDistribution(filepath.Join(skillDir, "SKILL.md"), target)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Distributable || len(result.MissingCommands) != 0 {
		t.Fatalf("unexpected distribution result: %+v", result)
	}
}

func TestScenarioDistributabilityReportUsesDeclaredRequirements(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	report, err := distributabilityForScenario(root, "hello-plugin", "scenarios/hello-plugin/cli/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Skills) != 1 || !report.Skills[0].Distributable {
		t.Fatalf("hello-plugin should be distributable to its CLI: %+v", report)
	}
}
