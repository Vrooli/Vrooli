package assets

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestScenarioToCloudA2PlanPinsExactScope(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, found, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil || !found {
		t.Fatalf("LoadScenarioPlan = found %v, err %v", found, err)
	}
	if plan.MinimumFreeBytes != 120_000_000_000 || plan.MaximumGrowthBytes != 25_000_000_000 {
		t.Fatalf("capacity bounds = reserve %d growth %d", plan.MinimumFreeBytes, plan.MaximumGrowthBytes)
	}
	if len(plan.Images) != 2 || len(plan.HostTools) != 3 {
		t.Fatalf("plan images/tools = %d/%d; want exactly 2/3", len(plan.Images), len(plan.HostTools))
	}
}

func TestPlanRejectsWidenedA2Scope(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*Plan)
	}{
		{"reserve", func(p *Plan) { p.MinimumFreeBytes = 119_000_000_000 }},
		{"growth", func(p *Plan) { p.MaximumGrowthBytes++ }},
		{"digest", func(p *Plan) { p.Images[0].SHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }},
		{"package", func(p *Plan) { p.HostTools[0].Package = "qemu" }},
		{"destination", func(p *Plan) { p.OutputDirectory = "/tmp/vm" }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			candidate := plan
			candidate.Images = append([]Image(nil), plan.Images...)
			candidate.HostTools = append([]ToolPlan(nil), plan.HostTools...)
			mutation.edit(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatal("widened or changed plan unexpectedly validated")
			}
		})
	}
}

func TestValidateToolPreviewRequiresExactThreeOwnersAndAptPackages(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	plan, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud")
	if err != nil {
		t.Fatal(err)
	}
	report := hostreqkit.Report{Host: hostreqkit.Host{OS: "linux", PackageManager: "apt"}, Tools: []hostreqkit.ToolStatus{
		{Name: "qemu", PackageName: "qemu-system-x86"},
		{Name: "qemu-arm", PackageName: "qemu-system-arm"},
		{Name: "cloud-localds", PackageName: "cloud-image-utils"},
	}}
	if err := ValidateToolPreview(plan, report); err != nil {
		t.Fatalf("exact preview rejected: %v", err)
	}
	report.Tools[0].PackageName = "qemu"
	if err := ValidateToolPreview(plan, report); err == nil {
		t.Fatal("changed package plan unexpectedly passed")
	}
}

func TestLoadScenarioPlanDoesNotMutateManifest(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	planPath := filepath.Join(root, "scenarios", "scenario-to-cloud", ".vrooli", planName)
	before, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := LoadScenarioPlan(root, "scenario-to-cloud"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("loading setup plan mutated its source manifest")
	}
	var plan Plan
	if err := json.Unmarshal(after, &plan); err != nil {
		t.Fatal(err)
	}
}
