package depsapproved

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

const removalFixture = `module example.com/root

go 1.25.0

toolchain go1.25.11

require (
	example.com/keep v1.2.3
	github.com/ulikunitz/xz v0.5.17 // indirect
)

exclude example.com/excluded v1.0.0

replace example.com/keep => ../keep

retract v0.1.0 // obsolete release

godebug default=go1.24
`

func TestRemoveRootGoRequirementDryRunPreservesSourceAndOtherSemantics(t *testing.T) {
	root := writeRemovalFixture(t, removalFixture)
	goModPath := filepath.Join(root, "go.mod")
	goSumPath := filepath.Join(root, "go.sum")
	beforeGoMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	beforeGoSum, err := os.ReadFile(goSumPath)
	if err != nil {
		t.Fatal(err)
	}

	result, err := removeRootGoRequirement(root, validRemovalRequest(false, ""))
	if err != nil {
		t.Fatal(err)
	}
	if result.Applied {
		t.Fatal("dry run reported an applied change")
	}
	if result.Target != goModPath || result.Module != "github.com/ulikunitz/xz" {
		t.Fatalf("result target/module = %q/%q", result.Target, result.Module)
	}
	proposed, err := removeRequirementForTest(beforeGoMod, "github.com/ulikunitz/xz")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(proposed), "github.com/ulikunitz/xz") {
		t.Fatalf("proposed go.mod still contains the removed module: %s", proposed)
	}
	if result.AfterSHA != sha256Hex(proposed) {
		t.Fatal("preview after digest does not match the exact proposed bytes")
	}
	if err := verifyOnlyRequirementRemoved(beforeGoMod, proposed, "github.com/ulikunitz/xz"); err != nil {
		t.Fatal(err)
	}
	actualGoMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	actualGoSum, err := os.ReadFile(goSumPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(actualGoMod) != string(beforeGoMod) || string(actualGoSum) != string(beforeGoSum) {
		t.Fatal("dry run modified go.mod or go.sum")
	}
}

func TestRemoveRootGoRequirementApplyRequiresDigestAndPreservesGoSum(t *testing.T) {
	root := writeRemovalFixture(t, removalFixture)
	goModPath := filepath.Join(root, "go.mod")
	goSumPath := filepath.Join(root, "go.sum")
	goSumBefore, err := os.ReadFile(goSumPath)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := removeRootGoRequirement(root, validRemovalRequest(false, ""))
	if err != nil {
		t.Fatal(err)
	}
	apply := validRemovalRequest(true, preview.BeforeSHA)
	result, err := removeRootGoRequirement(root, apply)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied {
		t.Fatal("apply result did not report the write")
	}
	actualGoMod, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	if sha256Hex(actualGoMod) != result.AfterSHA || strings.Contains(string(actualGoMod), "github.com/ulikunitz/xz") {
		t.Fatal("applied go.mod differs from previewed removal")
	}
	if err := verifyOnlyRequirementRemoved([]byte(removalFixture), actualGoMod, "github.com/ulikunitz/xz"); err != nil {
		t.Fatal(err)
	}
	goSumAfter, err := os.ReadFile(goSumPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(goSumAfter) != string(goSumBefore) {
		t.Fatal("apply modified or misreported go.sum")
	}
}

func TestRemoveRootGoRequirementRejectsDriftBeforeApply(t *testing.T) {
	root := writeRemovalFixture(t, removalFixture)
	preview, err := removeRootGoRequirement(root, validRemovalRequest(false, ""))
	if err != nil {
		t.Fatal(err)
	}
	goModPath := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goModPath, []byte(removalFixture+"\n// concurrent edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := removeRootGoRequirement(root, validRemovalRequest(true, preview.BeforeSHA)); err == nil || !strings.Contains(err.Error(), "changed since preview") {
		t.Fatalf("apply after concurrent edit error = %v", err)
	}
	data, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "// concurrent edit") {
		t.Fatal("drift refusal overwrote the concurrent edit")
	}
}

func TestRemoveRootGoRequirementRejectsWrongTargetAndMalformedInputs(t *testing.T) {
	root := writeRemovalFixture(t, removalFixture)
	cases := []struct {
		name string
		edit func(*removeGoRequirementRequest)
	}{
		{name: "wrong scenario", edit: func(r *removeGoRequirementRequest) { r.Scenario = "device-studio" }},
		{name: "wrong surface", edit: func(r *removeGoRequirementRequest) { r.Surface = "api" }},
		{name: "wrong ecosystem", edit: func(r *removeGoRequirementRequest) { r.Ecosystem = "npm" }},
		{name: "malformed module", edit: func(r *removeGoRequirementRequest) { r.Module = "../go.mod" }},
		{name: "whitespace module", edit: func(r *removeGoRequirementRequest) { r.Module = " github.com/ulikunitz/xz" }},
		{name: "apply without digest", edit: func(r *removeGoRequirementRequest) { r.Apply = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := validRemovalRequest(false, "")
			tc.edit(&request)
			if _, err := removeRootGoRequirement(root, request); err == nil {
				t.Fatal("invalid target or input was accepted")
			}
		})
	}
}

func TestRemoveRootGoRequirementRejectsMissingAndMalformedRootModule(t *testing.T) {
	t.Run("missing requirement", func(t *testing.T) {
		root := writeRemovalFixture(t, "module example.com/root\ngo 1.25.0\n")
		if _, err := removeRootGoRequirement(root, validRemovalRequest(false, "")); err == nil || !strings.Contains(err.Error(), "found 0") {
			t.Fatalf("missing requirement error = %v", err)
		}
	})
	t.Run("malformed go.mod", func(t *testing.T) {
		root := writeRemovalFixture(t, "module !!!\n")
		if _, err := removeRootGoRequirement(root, validRemovalRequest(false, "")); err == nil {
			t.Fatal("malformed root go.mod was accepted")
		}
	})
}

func TestRemoveRootGoRequirementRejectsSymlinkedRootGoMod(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-go.mod")
	if err := os.WriteFile(outside, []byte(removalFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "go.mod")); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	if _, err := removeRootGoRequirement(root, validRemovalRequest(false, "")); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("symlinked root go.mod error = %v", err)
	}
	data, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != removalFixture {
		t.Fatal("symlink refusal modified its target")
	}
}

func TestRemoveCommandOverridesAPIRequirement(t *testing.T) {
	group := RegisterInstall(nil)
	for _, command := range group.Subcommands {
		if command.Name != "remove" {
			continue
		}
		if command.NeedsAPIOverride == nil || *command.NeedsAPIOverride {
			t.Fatal("local root-module removal must remain available without the scenario API")
		}
		return
	}
	t.Fatal("deps remove command was not registered")
}

func writeRemovalFixture(t *testing.T, contents string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), []byte("example.com/keep v1.2.3 h1:sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func validRemovalRequest(apply bool, expectedSHA256 string) removeGoRequirementRequest {
	return removeGoRequirementRequest{
		Scenario: "vrooli", Surface: "control-plane", Ecosystem: "go",
		Module: "github.com/ulikunitz/xz", Apply: apply, ExpectedSHA256: expectedSHA256,
	}
}

func verifyOnlyRequirementRemoved(before, after []byte, modulePath string) error {
	old, err := modfile.Parse("go.mod", before, nil)
	if err != nil {
		return err
	}
	newFile, err := modfile.Parse("go.mod", after, nil)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(semanticForTest(old, modulePath), semanticForTest(newFile, "")) {
		return fmt.Errorf("semantic go.mod declarations changed beyond %s", modulePath)
	}
	return nil
}

func removeRequirementForTest(before []byte, modulePath string) ([]byte, error) {
	file, err := modfile.Parse("go.mod", before, nil)
	if err != nil {
		return nil, err
	}
	if err := file.DropRequire(modulePath); err != nil {
		return nil, err
	}
	return file.Format()
}

func semanticForTest(file *modfile.File, excludedRequire string) any {
	copy := *file
	copy.Syntax = nil
	copy.Module = cloneModule(file.Module)
	copy.Go = cloneGo(file.Go)
	copy.Toolchain = cloneToolchain(file.Toolchain)
	copy.Require = nil
	for _, item := range file.Require {
		if item.Mod.Path != excludedRequire {
			value := *item
			value.Syntax = nil
			copy.Require = append(copy.Require, &value)
		}
	}
	copy.Exclude = nil
	for _, item := range file.Exclude {
		value := *item
		value.Syntax = nil
		copy.Exclude = append(copy.Exclude, &value)
	}
	copy.Replace = nil
	for _, item := range file.Replace {
		value := *item
		value.Syntax = nil
		copy.Replace = append(copy.Replace, &value)
	}
	copy.Retract = nil
	for _, item := range file.Retract {
		value := *item
		value.Syntax = nil
		copy.Retract = append(copy.Retract, &value)
	}
	copy.Godebug = nil
	for _, item := range file.Godebug {
		value := *item
		value.Syntax = nil
		copy.Godebug = append(copy.Godebug, &value)
	}
	copy.Tool = nil
	for _, item := range file.Tool {
		value := *item
		value.Syntax = nil
		copy.Tool = append(copy.Tool, &value)
	}
	copy.Ignore = nil
	for _, item := range file.Ignore {
		value := *item
		value.Syntax = nil
		copy.Ignore = append(copy.Ignore, &value)
	}
	return copy
}

func cloneModule(value *modfile.Module) *modfile.Module {
	if value == nil {
		return nil
	}
	clone := *value
	clone.Syntax = nil
	return &clone
}
func cloneGo(value *modfile.Go) *modfile.Go {
	if value == nil {
		return nil
	}
	clone := *value
	clone.Syntax = nil
	return &clone
}
func cloneToolchain(value *modfile.Toolchain) *modfile.Toolchain {
	if value == nil {
		return nil
	}
	clone := *value
	clone.Syntax = nil
	return &clone
}
