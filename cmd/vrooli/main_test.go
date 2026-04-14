package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=5 | LAST: 2026-04-11

func TestParseArgsRecognizesLeadingGlobalFlags(t *testing.T) {
	parsed, err := rootcli.ParseArgs([]string{"--json", "--verbose", "--no-color", "scenario", "list"})
	if err != nil {
		t.Fatalf("ParseArgs: %v", err)
	}
	if parsed.Command != "scenario" {
		t.Fatalf("command = %q", parsed.Command)
	}
	if strings.Join(parsed.Args, ",") != "list" {
		t.Fatalf("args = %v", parsed.Args)
	}
	if !parsed.Globals.JSON || !parsed.Globals.Verbose || !parsed.Globals.NoColor {
		t.Fatalf("globals = %+v", parsed.Globals)
	}
}

func TestConsumeInlineGlobalFlagsPromotesCommandScopedGlobals(t *testing.T) {
	globals, args := rootcli.ConsumeInlineGlobalFlags(globalOptions{}, []string{"--scenarios", "--json", "--verbose", "--no-color"})
	if !globals.JSON || !globals.Verbose || !globals.NoColor {
		t.Fatalf("globals = %+v", globals)
	}
	if got := strings.Join(args, ","); got != "--scenarios" {
		t.Fatalf("args = %q", got)
	}
}

func TestCanonicalSetupInstallContract(t *testing.T) {
	repoRoot := repoRootFromCaller(t)

	serviceData, err := os.ReadFile(repocontractmeta.ProjectServiceManifestPath(repoRoot))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}

	var service struct {
		Lifecycle struct {
			Setup struct {
				Steps []struct {
					Name string `json:"name"`
					Run  string `json:"run"`
				} `json:"steps"`
			} `json:"setup"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(serviceData, &service); err != nil {
		t.Fatalf("unmarshal service.json: %v", err)
	}

	var installStep string
	for _, step := range service.Lifecycle.Setup.Steps {
		if step.Name == "install-cli" {
			installStep = step.Run
			break
		}
	}
	if installStep == "" {
		t.Fatalf("expected lifecycle.setup.steps to include install-cli")
	}
	if !strings.Contains(installStep, "make install") {
		t.Fatalf("install-cli step does not invoke make install: %q", installStep)
	}

	makefileData, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileContents := string(makefileData)
	if !strings.Contains(makefileContents, "install: build") {
		t.Fatalf("Makefile no longer defines the install target contract")
	}
	if !strings.Contains(makefileContents, "INSTALL_DIR = $(HOME)/.vrooli/bin") {
		t.Fatalf("Makefile no longer targets ~/.vrooli/bin")
	}
}

func TestInferPortEnvVarUsesStepPrefixesAndManifestNames(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
			"frontend": {
				EnvVar: "UI_PORT",
			},
		},
	}

	tests := []struct {
		step string
		want string
	}{
		{step: "start-api", want: "API_PORT"},
		{step: "launch-frontend", want: "UI_PORT"},
		{step: "serve-ui", want: "UI_PORT"},
		{step: "unknown", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.step, func(t *testing.T) {
			if got := scenario.InferPortEnvVar(manifest, tc.step); got != tc.want {
				t.Fatalf("inferPortEnvVar(%q) = %q, want %q", tc.step, got, tc.want)
			}
		})
	}
}

func TestBuildListPortsSortsAndMapsRecords(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
			"ui": {
				EnvVar: "UI_PORT",
			},
		},
	}

	listPorts, ports := scenariocli.BuildListPorts(manifest, []process.Record{
		{Step: "start-ui", Port: 38080},
		{Step: "start-api", Port: 18080},
	})

	if len(listPorts) != 2 {
		t.Fatalf("list port count = %d, want 2", len(listPorts))
	}
	if listPorts[0].Key != "API_PORT" || listPorts[1].Key != "UI_PORT" {
		t.Fatalf("list port order = %#v", listPorts)
	}
	if ports["API_PORT"] != 18080 || ports["UI_PORT"] != 38080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestBuildListPortsKeepsFirstExplicitRecordPerPort(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
		},
	}

	listPorts, ports := scenariocli.BuildListPorts(manifest, []process.Record{
		{Step: "start-api", Port: 18080},
		{Step: "run-api", Port: 19090},
	})

	if len(listPorts) != 1 {
		t.Fatalf("list port count = %d, want 1", len(listPorts))
	}
	if listPorts[0].Port != 18080 {
		t.Fatalf("explicit list port = %d, want 18080", listPorts[0].Port)
	}
	if ports["API_PORT"] != 18080 {
		t.Fatalf("summary port = %d, want first explicit port 18080", ports["API_PORT"])
	}
}
