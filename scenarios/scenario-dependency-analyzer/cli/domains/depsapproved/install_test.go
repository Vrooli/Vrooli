package depsapproved

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestParseInstallDependencySpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		spec            string
		explicitVersion string
		wantEcosystem   string
		wantPackage     string
		wantVersion     string
	}{
		{
			name:          "unscoped package with inline version",
			spec:          "npm/react@^18.3.1",
			wantEcosystem: "npm",
			wantPackage:   "react",
			wantVersion:   "^18.3.1",
		},
		{
			name:            "scoped package with explicit version flag",
			spec:            "npm/@dnd-kit/core",
			explicitVersion: "^6.3.1",
			wantEcosystem:   "npm",
			wantPackage:     "@dnd-kit/core",
			wantVersion:     "^6.3.1",
		},
		{
			name:          "tarball URL inline version",
			spec:          "npm/xlsx@https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz",
			wantEcosystem: "npm",
			wantPackage:   "xlsx",
			wantVersion:   "https://cdn.sheetjs.com/xlsx-0.20.3/xlsx-0.20.3.tgz",
		},
		{
			name:          "scoped package with inline local file version",
			spec:          "npm/@vrooli/file-viewer@file:../../../packages/file-viewer",
			wantEcosystem: "npm",
			wantPackage:   "@vrooli/file-viewer",
			wantVersion:   "file:../../../packages/file-viewer",
		},
		{
			name:          "scoped package with inline version",
			spec:          "npm/@dnd-kit/sortable@^10.0.0",
			wantEcosystem: "npm",
			wantPackage:   "@dnd-kit/sortable",
			wantVersion:   "^10.0.0",
		},
		{
			name:            "explicit version overrides inline version",
			spec:            "npm/@dnd-kit/utilities@^3.2.1",
			explicitVersion: "^3.2.2",
			wantEcosystem:   "npm",
			wantPackage:     "@dnd-kit/utilities",
			wantVersion:     "^3.2.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ecosystem, packageName, version, err := parseInstallDependencySpec(tt.spec, tt.explicitVersion)
			if err != nil {
				t.Fatalf("parseInstallDependencySpec() error = %v", err)
			}
			if ecosystem != tt.wantEcosystem || packageName != tt.wantPackage || version != tt.wantVersion {
				t.Fatalf(
					"parseInstallDependencySpec(%q, %q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.spec,
					tt.explicitVersion,
					ecosystem,
					packageName,
					version,
					tt.wantEcosystem,
					tt.wantPackage,
					tt.wantVersion,
				)
			}
		})
	}
}

func TestDepsCommandsExplainTheirPreviewDefaultWhenGivenGlobalDryRun(t *testing.T) {
	core, err := cliapp.NewScenarioApp(cliapp.ScenarioOptions{Name: "scenario-dependency-analyzer", DefaultAPIBase: "http://127.0.0.1:1", AllowAnonymous: true})
	if err != nil {
		t.Fatal(err)
	}
	group := RegisterInstall(core)
	for _, command := range group.Subcommands {
		if command.DryRunAlternative == "" {
			t.Errorf("deps %s rejects global --dry-run without saying what to do instead", command.Name)
		}
	}
}
