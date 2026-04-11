package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/resources"
)

func TestRunResourceBlueprintListCommandJSON(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{json: true}, []string{"list"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(list): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"blueprints":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintInfoCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"info", "terraform"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(info): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Terraform") || !strings.Contains(output, "external-cli") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintValidateCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"validate"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(validate): %v", err)
	}
	if !strings.Contains(stdout.String(), "Validated") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestRunResourceBlueprintSearchCommandHuman(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{}, []string{"search", "network"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(search): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "networking") && !strings.Contains(output, "Network") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunResourceBlueprintSearchCommandJSON(t *testing.T) {
	controller := resources.NewController(projectRootForCLI(t), t.TempDir())
	var stdout bytes.Buffer

	if err := runResourceBlueprintCommand(controller, globalOptions{json: true}, []string{"search", "network"}, &stdout); err != nil {
		t.Fatalf("runResourceBlueprintCommand(search json): %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"success": true`) || !strings.Contains(output, `"query": "network"`) || !strings.Contains(output, `"blueprints":`) {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestShowResourceHelpIncludesBlueprintCommands(t *testing.T) {
	var stdout bytes.Buffer
	showResourceHelp(&stdout)

	if !strings.Contains(stdout.String(), "resource blueprint") {
		t.Fatalf("help missing blueprint usage: %q", stdout.String())
	}
}

func projectRootForCLI(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	return root
}
