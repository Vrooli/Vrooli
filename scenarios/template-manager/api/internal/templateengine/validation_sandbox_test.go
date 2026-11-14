package templateengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/packages/proto/protogen"
	scenariocli "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templatecontracts"
)

// TestShallowValidationNeverWritesSharedProtoTree pins the template
// validation race fix: relocations, their regeneration hook, and proto lint
// run inside a private sandbox that cannot publish to the shared Proto
// artifact store; the only checkout-visible output is the relocated owner's
// generated Go, lent under the Proto generation lock for module convergence
// and removed before the lock is released.
func TestShallowValidationNeverWritesSharedProtoTree(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-sandbox", nil)
	writeTestFile(t, filepath.Join(info.Path, "api", "go.mod"), "module {{SCENARIO_ID}}/api\n\ngo 1.25.0\n")
	owner := "template-validation-reloc-sandbox"
	sharedProto := filepath.Join(repoRoot, "packages", "proto")
	writeTestFile(t, filepath.Join(sharedProto, "schemas", "common", "v1", "common.proto"), "syntax = \"proto3\";\n")
	writeTestFile(t, filepath.Join(sharedProto, "gen", "go", "common", "v1", "common.pb.go"), "package v1\n")
	writeTestFile(t, filepath.Join(repoRoot, "packages", "api-core", "go.mod"), "module github.com/vrooli/api-core\n")
	// A staging tree left in the shared tree must not be copied into the sandbox.
	writeTestFile(t, filepath.Join(sharedProto, ".proto-compatibility-1", "gen", "x"), "stale")
	before := treeDigest(t, filepath.Join(repoRoot, "packages"))

	locked := false
	restore := protoGenerationLock
	protoGenerationLock = func(context.Context) (func(), error) {
		locked = true
		return func() { locked = false }, nil
	}
	defer func() { protoGenerationLock = restore }()

	var hook shell.Spec
	var sandboxSchema string
	moduleChecks := 0
	capture := &capturedSubprocess{onRun: func(spec shell.Spec) error {
		switch spec.Name {
		case "make":
			hook = spec
			sandboxSchema = filepath.Join(spec.Dir, "schemas", owner, owner, "v1", "health.proto")
			if _, err := os.Stat(sandboxSchema); err != nil {
				t.Errorf("relocated schema not present in the sandbox when the hook ran: %v", err)
			}
			for _, path := range []string{
				filepath.Join(spec.Dir, "gen", "go", "common", "v1", "common.pb.go"),
				filepath.Join(spec.Dir, "..", "api-core", "go.mod"),
			} {
				if _, err := os.Stat(path); err != nil {
					t.Errorf("sandbox is missing %s: %v", path, err)
				}
			}
			if _, err := os.Stat(filepath.Join(spec.Dir, ".proto-compatibility-1")); !os.IsNotExist(err) {
				t.Errorf("protogen work tree was copied into the sandbox: %v", err)
			}
			// Simulate the hook publishing generated output and a read-only
			// artifact snapshot inside the sandbox.
			writeTestFile(t, filepath.Join(spec.Dir, "gen", "go", owner, "v1", "health.pb.go"), "package v1\n")
			snapshot := filepath.Join(envValue(spec.Env, protogen.ArtifactRootEnv), "snapshots", "id")
			writeTestFile(t, filepath.Join(snapshot, "gen", "a.go"), "package a\n")
			return os.Chmod(filepath.Join(snapshot, "gen"), 0o555)
		case "scenario-dependency-analyzer":
			moduleChecks++
			if !locked {
				t.Error("module convergence ran without the Proto generation lock")
			}
			if _, err := os.Stat(filepath.Join(sharedProto, "gen", "go", owner, "v1", "health.pb.go")); err != nil {
				t.Errorf("relocated owner's generated Go was not lent to module convergence: %v", err)
			}
		}
		return nil
	}}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)

	issues := validateTemplateShallow(deps, struct{}{}, info)
	if len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
	if hook.Name == "" || moduleChecks == 0 {
		t.Fatalf("hook ran=%t module checks=%d, want both", hook.Name != "", moduleChecks)
	}
	if strings.HasPrefix(hook.Dir, repoRoot) {
		t.Fatalf("post hook ran in the shared checkout: %s", hook.Dir)
	}
	sandboxRoot := filepath.Dir(filepath.Dir(hook.Dir))
	if artifactRoot := envValue(hook.Env, protogen.ArtifactRootEnv); !strings.HasPrefix(artifactRoot, sandboxRoot+string(filepath.Separator)) {
		t.Fatalf("post hook artifact root = %q, want one inside the sandbox %s", artifactRoot, sandboxRoot)
	}
	for _, call := range capture.calls {
		if call.Name == "buf" && strings.HasPrefix(call.Dir, repoRoot) {
			t.Fatalf("proto lint ran in the shared checkout: %s", call.Dir)
		}
	}
	if locked {
		t.Fatal("Proto generation lock was not released")
	}
	if after := treeDigest(t, filepath.Join(repoRoot, "packages")); after != before {
		t.Fatal("shared packages tree changed during validation")
	}
	if _, err := os.Stat(sandboxRoot); !os.IsNotExist(err) {
		t.Fatalf("validation sandbox survived validation: %v", err)
	}
}

func TestLendGeneratedGoRefusesCheckoutResidue(t *testing.T) {
	repoRoot := t.TempDir()
	sandbox := &validationSandbox{Root: t.TempDir()}
	writeTestFile(t, filepath.Join(sandbox.Root, "packages", "proto", "gen", "go", "demo", "a.go"), "package a\n")
	residue := filepath.Join(repoRoot, "packages", "proto", "gen", "go", "demo", "old.go")
	writeTestFile(t, residue, "package old\n")
	restore := protoGenerationLock
	protoGenerationLock = func(context.Context) (func(), error) { return func() {}, nil }
	defer func() { protoGenerationLock = restore }()

	_, err := sandbox.lendGeneratedGo(context.Background(), repoRoot, []string{"demo"}, func() []scenariocli.TemplateValidationIssue {
		t.Fatal("check ran despite residue in the checkout")
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "lifecycle destroy demo --proto-only") {
		t.Fatalf("err = %v, want residue refusal naming the reap command", err)
	}
	if _, err := os.Stat(residue); err != nil {
		t.Fatalf("pre-existing checkout output must be left for its owner command: %v", err)
	}
}

func TestShallowValidationWithoutRelocationsUsesTheCheckout(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "no-reloc", nil)
	info.Manifest.Relocations = nil
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	if issues := validateTemplateShallow(deps, struct{}{}, info); len(issues) != 0 {
		t.Fatalf("issues = %#v", issues)
	}
	for _, call := range capture.calls {
		if call.Name == "make" || call.Name == "buf" {
			t.Fatalf("template without relocations ran %s", call.Name)
		}
	}
}

func envValue(env []string, key string) string {
	for _, entry := range env {
		if name, value, ok := strings.Cut(entry, "="); ok && name == key {
			return value
		}
	}
	return ""
}

// treeDigest fingerprints every path, mode, link target and file body under
// root without following links.
func treeDigest(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		line := rel + " " + info.Mode().String()
		switch {
		case entry.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			line += " -> " + link
		case entry.Type().IsRegular():
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(body)
			line += " " + hex.EncodeToString(sum[:])
		}
		lines = append(lines, line)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}
