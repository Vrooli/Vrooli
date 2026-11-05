package envresolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConformanceDerivesUndeclaredResourceFindingAndOwner(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "resources", "postgres"),
		filepath.Join(root, "scenarios", "consumer", ".vrooli"),
		filepath.Join(root, "scenarios", "consumer", "api"),
		filepath.Join(root, "packages", "api-core", ".vrooli"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"environment_exports":{"static":{"POSTGRES_HOST":"localhost"}}}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", ".vrooli", "service.json"), `{}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", "api", "main.go"), "package main\nimport \"os\"\nvar _ = os.Getenv(\"POSTGRES_HOST\")\n")
	writeFile(t, filepath.Join(root, "packages", "api-core", ".vrooli", "package.json"), `{"package":{"name":"api-core","adoption":{"owns_resource_environment":["postgres"]}}}`)
	findings, err := ConformanceScan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "env.package_bypassed" || findings[0].Package != "api-core" {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestConformanceFlagsRawResourceReadEvenWhenResourceIsDeclared(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "resources", "postgres"),
		filepath.Join(root, "scenarios", "consumer", ".vrooli"),
		filepath.Join(root, "scenarios", "consumer", "api"),
		filepath.Join(root, "packages", "api-core", ".vrooli"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"environment_exports":{"static":{"POSTGRES_HOST":"localhost"}}}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", ".vrooli", "service.json"), `{"dependencies":{"resources":{"postgres":{}}}}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", "api", "main.go"), "package main\nimport \"os\"\nvar _ = os.Getenv(\"POSTGRES_HOST\")\n")
	writeFile(t, filepath.Join(root, "packages", "api-core", ".vrooli", "package.json"), `{"package":{"name":"api-core","adoption":{"owns_resource_environment":["postgres"]}}}`)
	findings, err := ConformanceScan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "env.package_bypassed" {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestConformanceReportsAddressAndAbsentFindings(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "consumer", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "peer", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "service.json"), `{}`)
	writeFile(t, filepath.Join(root, "scenarios", "peer", ".vrooli", "service.json"), `{}`)
	writeFile(t, filepath.Join(root, "scenarios", "consumer", "api", "main.go"), "package main\nimport \"os\"\nvar _ = os.Getenv(\"PEER_API_URL\")\nvar _ = os.Getenv(\"UNOWNED_SETTING\")\n")
	findings, err := ConformanceScan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestConformanceReportsDeadResourceProducer(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "consumer")
	if err := os.MkdirAll(filepath.Join(scenario, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(scenario, ".vrooli", "service.json"), `{}`)
	writeFile(t, filepath.Join(scenario, "main.go"), "package main\nimport \"os\"\nfunc getResourcePort(string) string { return \"\" }\nfunc main() { _ = getResourcePort(\"n8n\"); _ = os.Getenv(\"N8N_BASE_URL\") }\n")
	findings, err := ConformanceScan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "env.dead_producer" || findings[0].Producer != "n8n" {
		t.Fatalf("findings = %#v", findings)
	}
}
