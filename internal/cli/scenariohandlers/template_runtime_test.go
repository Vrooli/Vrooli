package scenariohandlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	rootcli "github.com/vrooli/vrooli/internal/cli/rootcli"
	scenariocli "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioexec"
)

func TestBuildTemplateValuesAndCopyTemplateRenderGeneratedGoModPaths(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath() error = %v", err)
	}
	info, err := loadTemplate(repoRoot, "react-vite")
	if err != nil {
		t.Fatalf("loadTemplate() error = %v", err)
	}

	destination := filepath.Join(t.TempDir(), "scenarios", "alpha")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID":           "alpha",
		"SCENARIO_DISPLAY_NAME": "Alpha",
		"SCENARIO_DESCRIPTION":  "Alpha scenario",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues() error = %v", err)
	}
	wantPackagesRel, err := filepath.Rel(filepath.Join(destination, "api"), filepath.Join(repoRoot, "packages"))
	if err != nil {
		t.Fatalf("filepath.Rel(packages) error = %v", err)
	}
	if got := values["PACKAGES_REL_FROM_API"]; got != filepath.ToSlash(wantPackagesRel) {
		t.Fatalf("PACKAGES_REL_FROM_API = %q, want %q", got, filepath.ToSlash(wantPackagesRel))
	}
	wantRepoRootRel, err := filepath.Rel(filepath.Join(destination, "cli"), repoRoot)
	if err != nil {
		t.Fatalf("filepath.Rel(repo root) error = %v", err)
	}
	if got := values["REPO_ROOT_REL_FROM_CLI"]; got != filepath.ToSlash(wantRepoRootRel) {
		t.Fatalf("REPO_ROOT_REL_FROM_CLI = %q, want %q", got, filepath.ToSlash(wantRepoRootRel))
	}

	if err := copyTemplate(info.Path, destination, values, info.Manifest.Relocations); err != nil {
		t.Fatalf("copyTemplate() error = %v", err)
	}
	if err := verifyTemplate(destination); err != nil {
		t.Fatalf("verifyTemplate() error = %v", err)
	}

	apiGoMod, err := os.ReadFile(filepath.Join(destination, "api", "go.mod"))
	if err != nil {
		t.Fatalf("read api/go.mod: %v", err)
	}
	expectedReplace := "replace github.com/vrooli/api-core => " + filepath.ToSlash(filepath.Join(wantPackagesRel, "api-core"))
	if !strings.Contains(string(apiGoMod), expectedReplace) {
		t.Fatalf("api/go.mod = %s", string(apiGoMod))
	}

	issues := validateGeneratedScenario(destination, false, nil, info.Name)
	if len(issues) != 0 {
		t.Fatalf("validateGeneratedScenario() issues = %#v", issues)
	}
}

func TestValidateGeneratedScenarioFlagsBrokenLocalReplaceTarget(t *testing.T) {
	destination := t.TempDir()
	moduleDir := filepath.Join(destination, "api")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	goMod := `module example.com/demo

go 1.22

replace github.com/vrooli/api-core => ../../../packages/api-core
`
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	issues := validateGeneratedScenario(destination, false, nil, "demo")
	if len(issues) == 0 {
		t.Fatal("expected validation issues for broken replace target")
	}
	if !strings.Contains(issues[0].Message, "does not resolve") {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestValidateTemplateSourceFlagsHardcodedLocalReplaceTargets(t *testing.T) {
	templateDir := t.TempDir()
	apiDir := filepath.Join(templateDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "go.mod"), []byte("module example.com/demo\n\nreplace github.com/vrooli/api-core => ../../../../packages/api-core\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	issues := validateTemplateSource(scenariocli.TemplateInfo{Name: "demo", Path: templateDir, Manifest: scenariocli.TemplateManifest{}})
	if len(issues) == 0 {
		t.Fatal("expected validateTemplateSource() to flag hardcoded local replace target")
	}
	if !strings.Contains(issues[0].Message, "generator-computed placeholders") {
		t.Fatalf("issues = %#v", issues)
	}
}

// --------------------------------------------------------------------------
// Relocation tests
// --------------------------------------------------------------------------

// captureSubprocess records every SubprocessSpec passed to RunSubprocess so
// post-command tests can assert cwd, command, and call count without
// actually executing anything.
type capturedSubprocess struct {
	calls []scenarioexec.SubprocessSpec
}

func (c *capturedSubprocess) Run(_ struct{}, spec scenarioexec.SubprocessSpec) error {
	c.calls = append(c.calls, spec)
	return nil
}

// newRelocationTestDeps builds a HandlerDeps that captures subprocess
// invocations, so post-command tests can assert against cwd / command.
// Stdout/Stderr go to the provided buffers; CommandEnv is empty.
func newRelocationTestDeps(repoRoot string, stdout, stderr io.Writer, capture *capturedSubprocess) HandlerDeps[struct{}] {
	return HandlerDeps[struct{}]{
		Stdout:        func(struct{}) io.Writer { return stdout },
		Stderr:        func(struct{}) io.Writer { return stderr },
		Root:          func(struct{}) string { return repoRoot },
		Globals:       func(struct{}) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat:  func(struct{}) (cliout.Format, error) { return cliout.FormatHuman, nil },
		HomeDir:       func(struct{}) (string, error) { return "", nil },
		RunSubprocess: capture.Run,
		CommandEnv:    func(struct{}) []string { return nil },
	}
}

// seedRepoContract copies the canonical .vrooli/repo-contract.json from
// the real repo into the test repoRoot so buildTemplateValues can resolve
// {{PACKAGES_REL_FROM_API}} et al. The contract validator is strict
// (resource.manifest, scenario.required_files, etc. all required), so
// reusing the real file is simpler and safer than maintaining a
// hand-trimmed copy.
func seedRepoContract(t *testing.T, repoRoot string) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed in seedRepoContract")
	}
	realRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath() error = %v", err)
	}
	contract, err := os.ReadFile(filepath.Join(realRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read canonical repo-contract.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
	// The contract loader checks that root.markers.required_dirs exist, so
	// pre-create them as empty dirs.
	for _, dir := range []string{"packages", "templates", "scenarios", "resources", "cmd", "internal"} {
		_ = os.MkdirAll(filepath.Join(repoRoot, dir), 0o755)
	}
}

// writeRelocationTemplate creates a tiny template tree at templatesDir/<name>/
// with a manifest that declares one proto-shaped relocation. Returns the
// repo root for use as deps.Root. The template's `proto/` source contains
// a `{{SCENARIO_ID}}.proto` file whose body references {{SCENARIO_ID_SNAKE}}
// so substitution is exercised in both path and content.
func writeRelocationTemplate(t *testing.T, templateName string, extraManifest map[string]any) (repoRoot string, info scenariocli.TemplateInfo) {
	t.Helper()
	repoRoot = t.TempDir()
	seedRepoContract(t, repoRoot)
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	protoSrc := filepath.Join(templatesDir, "proto", "{{SCENARIO_ID}}", "v1")
	if err := os.MkdirAll(protoSrc, 0o755); err != nil {
		t.Fatalf("mkdir proto src: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(protoSrc, "health.proto"),
		[]byte("syntax = \"proto3\";\npackage vrooli.{{SCENARIO_ID_SNAKE}}.v1.health;\n"),
		0o644,
	); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	// Also include a non-proto file inside the scenario tree (so we can
	// confirm copyTemplate's skip-list excludes proto/ from the in-tree
	// copy without affecting unrelated files).
	apiDir := filepath.Join(templatesDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	manifest := map[string]any{
		"name":         templateName,
		"requiredVars": map[string]any{"SCENARIO_ID": map[string]any{"flag": "id"}},
		"relocations": []map[string]any{
			{
				"description": "Relocate proto schemas",
				"from":        "proto/",
				"to":          "packages/proto/schemas/{{SCENARIO_ID}}/",
				"post": []map[string]any{
					{"description": "regen", "cmd": "make generate", "cwd": "packages/proto"},
				},
			},
		},
	}
	for k, v := range extraManifest {
		manifest[k] = v
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}
	info, err = loadTemplate(repoRoot, templateName)
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}
	return repoRoot, info
}

func TestRelocations_CopiesAndSubstitutes(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-copy", nil)
	destination := filepath.Join(repoRoot, "scenarios", "my-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "my-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	if got := values["SCENARIO_ID_SNAKE"]; got != "my_app" {
		t.Fatalf("SCENARIO_ID_SNAKE = %q, want %q", got, "my_app")
	}

	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved = %d, want 1", len(resolved))
	}
	wantTo := filepath.Join(repoRoot, "packages", "proto", "schemas", "my-app")
	if resolved[0].To != wantTo {
		t.Fatalf("resolved.To = %q, want %q", resolved[0].To, wantTo)
	}

	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	if err := runRelocations(deps, struct{}{}, info.Path, resolved, values); err != nil {
		t.Fatalf("runRelocations: %v", err)
	}

	// Path component substitution: {{SCENARIO_ID}} -> my-app
	out := filepath.Join(wantTo, "my-app", "v1", "health.proto")
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read relocated proto: %v", err)
	}
	// Content substitution: {{SCENARIO_ID_SNAKE}} -> my_app
	if !strings.Contains(string(body), "package vrooli.my_app.v1.health;") {
		t.Fatalf("substitution did not run; body=%s", body)
	}
}

func TestRelocations_VerifyDetectsUnresolvedPlaceholders(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-verify", nil)
	// Append a deliberately unresolved placeholder to the proto body so
	// renderTemplateString can't substitute it (the value never appears
	// in the values map).
	protoFile := filepath.Join(info.Path, "proto", "{{SCENARIO_ID}}", "v1", "health.proto")
	body, err := os.ReadFile(protoFile)
	if err != nil {
		t.Fatalf("read proto: %v", err)
	}
	if err := os.WriteFile(protoFile, append(body, []byte("\n// {{UNKNOWN_PLACEHOLDER}}\n")...), 0o644); err != nil {
		t.Fatalf("write proto with placeholder: %v", err)
	}

	destination := filepath.Join(repoRoot, "scenarios", "verify")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "verify",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	err = runRelocations(deps, struct{}{}, info.Path, resolved, values)
	if err == nil {
		t.Fatal("expected runRelocations to fail on unresolved placeholder")
	}
	if !strings.Contains(err.Error(), "unresolved placeholders") {
		t.Fatalf("err = %v", err)
	}
}

func TestRelocations_PostCommandsRunAtRepoRoot(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-post", nil)
	destination := filepath.Join(repoRoot, "scenarios", "post-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "post-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	resolved, err := resolveRelocations(repoRoot, info, values)
	if err != nil {
		t.Fatalf("resolveRelocations: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	if err := runRelocations(deps, struct{}{}, info.Path, resolved, values); err != nil {
		t.Fatalf("runRelocations: %v", err)
	}
	if len(capture.calls) != 1 {
		t.Fatalf("captured %d calls, want 1: %#v", len(capture.calls), capture.calls)
	}
	got := capture.calls[0]
	wantCwd := filepath.Join(repoRoot, "packages", "proto")
	if got.Dir != wantCwd {
		t.Fatalf("post.Dir = %q, want %q (must run at repo root + Cwd, NOT scenario destination)", got.Dir, wantCwd)
	}
	if !strings.Contains(strings.Join(got.Args, " "), "make generate") {
		t.Fatalf("post.Args = %#v, want to contain 'make generate'", got.Args)
	}
}

func TestRunGenerate_RelocationIdempotencyGuard(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-idem", map[string]any{
		"requiredVars": map[string]any{
			"SCENARIO_ID":           map[string]any{"flag": "id"},
			"SCENARIO_DISPLAY_NAME": map[string]any{"flag": "display-name"},
			"SCENARIO_DESCRIPTION":  map[string]any{"flag": "description"},
		},
	})

	// Pre-create the relocation target so the idempotency guard fires.
	preExisting := filepath.Join(repoRoot, "packages", "proto", "schemas", "idem-app")
	if err := os.MkdirAll(preExisting, 0o755); err != nil {
		t.Fatalf("pre-create relocation target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(preExisting, "leftover.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}

	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	req := scenariocli.GenerateRequest{
		TemplateInfo: info,
		Options: scenariocli.GenerateOptions{
			Force: false,
			Values: map[string]string{
				"SCENARIO_ID":           "idem-app",
				"SCENARIO_DISPLAY_NAME": "Idem App",
				"SCENARIO_DESCRIPTION":  "Idempotency test",
			},
		},
	}
	_, _, err := runGenerate(deps, struct{}{}, req)
	if err == nil {
		t.Fatal("expected runGenerate to error when relocation target exists without --force")
	}
	if !strings.Contains(err.Error(), "relocation target already exists") {
		t.Fatalf("err = %v", err)
	}

	// With --force the existing target is removed.
	req.Options.Force = true
	_, _, err = runGenerate(deps, struct{}{}, req)
	if err != nil {
		t.Fatalf("runGenerate with --force: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(preExisting, "leftover.txt")); statErr == nil {
		t.Fatal("leftover.txt should have been removed by --force")
	}
}

func TestRunGenerate_DryRunDoesNotWriteRelocations(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-dry", map[string]any{
		"requiredVars": map[string]any{
			"SCENARIO_ID":           map[string]any{"flag": "id"},
			"SCENARIO_DISPLAY_NAME": map[string]any{"flag": "display-name"},
			"SCENARIO_DESCRIPTION":  map[string]any{"flag": "description"},
		},
	})
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	req := scenariocli.GenerateRequest{
		TemplateInfo: info,
		Options: scenariocli.GenerateOptions{
			DryRun: true,
			Values: map[string]string{
				"SCENARIO_ID":           "dry-app",
				"SCENARIO_DISPLAY_NAME": "Dry App",
				"SCENARIO_DESCRIPTION":  "Dry run",
			},
		},
	}
	_, result, err := runGenerate(deps, struct{}{}, req)
	if err != nil {
		t.Fatalf("dry runGenerate: %v", err)
	}
	if !result.DryRun {
		t.Fatal("result.DryRun should be true")
	}
	if len(result.Relocations) != 1 {
		t.Fatalf("dry-run result.Relocations len = %d, want 1", len(result.Relocations))
	}
	wantTo := filepath.Join(repoRoot, "packages", "proto", "schemas", "dry-app")
	if result.Relocations[0].To != wantTo {
		t.Fatalf("dry-run reloc.To = %q, want %q", result.Relocations[0].To, wantTo)
	}
	// Nothing should have been written.
	if _, err := os.Stat(filepath.Join(repoRoot, "scenarios", "dry-app")); err == nil {
		t.Fatal("dry-run wrote scenario destination")
	}
	if _, err := os.Stat(wantTo); err == nil {
		t.Fatal("dry-run wrote relocation target")
	}
	if len(capture.calls) != 0 {
		t.Fatalf("dry-run invoked %d subprocess calls, want 0", len(capture.calls))
	}
}

func TestCopyTemplate_SkipsRelocationFromDirs(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-skip", nil)
	destination := filepath.Join(repoRoot, "scenarios", "skip-app")
	values, err := buildTemplateValues(repoRoot, destination, info.Name, info.Manifest, map[string]string{
		"SCENARIO_ID": "skip-app",
	})
	if err != nil {
		t.Fatalf("buildTemplateValues: %v", err)
	}
	if err := copyTemplate(info.Path, destination, values, info.Manifest.Relocations); err != nil {
		t.Fatalf("copyTemplate: %v", err)
	}
	// proto/ is in the manifest's relocations so it must NOT appear
	// inside the scenario destination — the in-tree skip-list pruned it.
	if _, err := os.Stat(filepath.Join(destination, "proto")); err == nil {
		t.Fatal("copyTemplate should skip relocation From dirs (proto/ leaked into scenario destination)")
	}
	// But unrelated files should still land.
	if _, err := os.Stat(filepath.Join(destination, "api", "main.go")); err != nil {
		t.Fatalf("non-relocation file missing from destination: %v", err)
	}
}

func TestValidateRelocationProtoSources_DetectsProtoDir(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-lint", nil)
	// The validator copies template protos into a temp dir under
	// packages/proto/schemas/ before running `buf lint`, so the schemas
	// directory must exist for the lint path to be exercised.
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "proto", "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir packages/proto/schemas: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	// We don't assert on issue presence (the capture stub always returns
	// nil from RunSubprocess so no failure is reported); we assert the
	// lint call was made with the right shape.
	if len(capture.calls) != 1 {
		t.Fatalf("captured %d calls, want 1 buf lint invocation", len(capture.calls))
	}
	args := strings.Join(capture.calls[0].Args, " ")
	if !strings.Contains(args, "buf lint") {
		t.Fatalf("call args = %q, want to contain 'buf lint'", args)
	}
	// The path is a temp subdirectory under schemas/ — it should be a
	// relative path beginning with `schemas/.tmp-validate-reloc-lint-`.
	if !strings.Contains(args, "schemas/.tmp-validate-reloc-lint-") {
		t.Fatalf("call args = %q, want to lint a temp dir under schemas/", args)
	}
	if capture.calls[0].Dir != filepath.Join(repoRoot, "packages", "proto") {
		t.Fatalf("call dir = %q, want packages/proto", capture.calls[0].Dir)
	}
	_ = issues
}

// TestValidateRelocationProtoSources_SurfacesStdoutOnLintFailure pins the
// stdout-capture fix: `buf lint` writes diagnostics to stdout, not stderr,
// so the validator must surface stdout in the issue Message. Pre-fix,
// failures collapsed to a useless "buf lint failed: exit status 100" string.
func TestValidateRelocationProtoSources_SurfacesStdoutOnLintFailure(t *testing.T) {
	repoRoot, info := writeRelocationTemplate(t, "reloc-lint-fail", nil)
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "proto", "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir packages/proto/schemas: %v", err)
	}

	const lintDiagnostic = `Service name "Notes" should be suffixed with "Service".`
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, &capturedSubprocess{})
	deps.RunSubprocess = func(_ struct{}, spec scenarioexec.SubprocessSpec) error {
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, lintDiagnostic+"\n")
		}
		return errors.New("exit status 100")
	}

	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 1 {
		t.Fatalf("issues = %#v, want exactly 1", issues)
	}
	msg := issues[0].Message
	if !strings.Contains(msg, lintDiagnostic) {
		t.Fatalf("issue message %q must surface the stdout diagnostic %q", msg, lintDiagnostic)
	}
	if strings.Contains(msg, ".tmp-validate-") {
		t.Fatalf("issue message %q must rewrite the temp-dir prefix back to the template path; got raw temp dir", msg)
	}
}

func TestValidateRelocationProtoSources_SkipsWhenSchemasDirAbsent(t *testing.T) {
	// When packages/proto/schemas/ doesn't exist (test repo without a
	// real proto module), the validator returns no issues and runs no
	// subprocess — schema-level mistakes will surface at make-generate
	// time during an actual scenario generation, which is a separate
	// failure mode.
	repoRoot, info := writeRelocationTemplate(t, "reloc-noschemas", nil)
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 0 {
		t.Fatalf("issues = %#v, want none when schemas dir absent", issues)
	}
	if len(capture.calls) != 0 {
		t.Fatalf("invoked %d subprocess calls; want 0 when schemas dir absent", len(capture.calls))
	}
}

// TestValidateRelocationProtoSources_RunsOfflineWithBSRUnreachable proves
// that template proto validation does not contact BSR — the change that
// CD-1 (local plugins) and CD-2 (vendored modules) were meant to deliver.
// See docs/plans/proto-codegen-local-and-bsr-login-implementation-plan.md.
//
// Mechanism: invoke validateRelocationProtoSources end-to-end against the
// real packages/proto/ module with HTTPS_PROXY=http://127.0.0.1:9 forced
// into the subprocess env. Any BSR call would hard-fail; the test asserts
// the lint still succeeds. The fixture proto imports
// `buf/validate/validate.proto` so transitive resolution through the
// vendored protovalidate workspace module is exercised — pre-CD-2 this
// import would have triggered a BSR fetch.
//
// Skipped when buf isn't on PATH or when packages/proto/schemas/ is
// missing (e.g. minimal CI environments without `vrooli setup` complete).
func TestValidateRelocationProtoSources_RunsOfflineWithBSRUnreachable(t *testing.T) {
	if _, err := exec.LookPath("buf"); err != nil {
		t.Skipf("buf not on PATH: %v", err)
	}

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	repoRoot, err := repocontract.FindRepoRootFromPath(thisFile)
	if err != nil {
		t.Fatalf("FindRepoRootFromPath: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "packages", "proto", "schemas")); err != nil {
		t.Skipf("packages/proto/schemas not present in this repo layout: %v", err)
	}

	// Pollute templates/scenarios/ with a uniquely named fixture and
	// clean it up on completion. The validator copies into
	// packages/proto/schemas/.tmp-validate-<name>-<rand>/ and removes that
	// itself, so we only own the templates side.
	templateName := "offline-bsr-validation-test"
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", templateName)
	if _, err := os.Stat(templatesDir); err == nil {
		t.Fatalf("fixture %s already exists; another run may be in progress", templatesDir)
	}
	t.Cleanup(func() { _ = os.RemoveAll(templatesDir) })

	protoSrc := filepath.Join(templatesDir, "proto", "{{SCENARIO_ID}}", "v1")
	if err := os.MkdirAll(protoSrc, 0o755); err != nil {
		t.Fatalf("mkdir proto src: %v", err)
	}
	// The import on `buf/validate/validate.proto` is the load-bearing
	// detail: it forces buf lint to resolve a symbol from the vendored
	// protovalidate module. Pre-CD-2, buf would have fetched it from BSR.
	protoBody := strings.Join([]string{
		`syntax = "proto3";`,
		`package vrooli.{{SCENARIO_ID_SNAKE}}.v1.health;`,
		`import "buf/validate/validate.proto";`,
		`message Probe {`,
		`  string name = 1 [(buf.validate.field).string.min_len = 1];`,
		`}`,
		``,
	}, "\n")
	if err := os.WriteFile(filepath.Join(protoSrc, "health.proto"), []byte(protoBody), 0o644); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	manifest := map[string]any{
		"name":         templateName,
		"requiredVars": map[string]any{"SCENARIO_ID": map[string]any{"flag": "id"}},
		"relocations": []map[string]any{{
			"description": "Relocate proto schemas",
			"from":        "proto/",
			"to":          "packages/proto/schemas/{{SCENARIO_ID}}/",
		}},
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}

	info, err := loadTemplate(repoRoot, templateName)
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}

	// Real subprocess runner; CommandEnv injects deliberately unreachable
	// proxies so any BSR HTTP would hard-fail. The test asserts the lint
	// still succeeds — proving template validation is offline-clean.
	deps := HandlerDeps[struct{}]{
		Stdout:       func(struct{}) io.Writer { return io.Discard },
		Stderr:       func(struct{}) io.Writer { return io.Discard },
		Root:         func(struct{}) string { return repoRoot },
		Globals:      func(struct{}) rootcli.GlobalOptions { return rootcli.GlobalOptions{} },
		OutputFormat: func(struct{}) (cliout.Format, error) { return cliout.FormatHuman, nil },
		HomeDir:      func(struct{}) (string, error) { return os.UserHomeDir() },
		RunSubprocess: func(_ struct{}, spec scenarioexec.SubprocessSpec) error {
			return scenarioexec.RunSubprocess(spec)
		},
		CommandEnv: func(struct{}) []string {
			// Inherit PATH/HOME so `buf` and ~/.netrc are reachable, then
			// add proxies pointing at a port that nothing listens on.
			env := os.Environ()
			env = append(env,
				"HTTPS_PROXY=http://127.0.0.1:9",
				"HTTP_PROXY=http://127.0.0.1:9",
				"https_proxy=http://127.0.0.1:9",
				"http_proxy=http://127.0.0.1:9",
				"NO_PROXY=",
				"no_proxy=",
			)
			return env
		},
	}

	issues := validateRelocationProtoSources(deps, struct{}{}, info)

	if len(issues) != 0 {
		var msg strings.Builder
		for _, iss := range issues {
			fmt.Fprintf(&msg, "  - [%s] %s: %s\n", iss.Template, iss.Path, iss.Message)
		}
		t.Fatalf(
			"validateRelocationProtoSources surfaced %d issue(s) with HTTPS_PROXY=http://127.0.0.1:9 — codegen is reaching BSR (CD-1/CD-2 regressed):\n%s",
			len(issues), msg.String(),
		)
	}
}

func TestValidateRelocationProtoSources_SkipsNonProtoRelocations(t *testing.T) {
	repoRoot := t.TempDir()
	seedRepoContract(t, repoRoot)
	templatesDir := filepath.Join(repoRoot, "templates", "scenarios", "reloc-nonproto")
	scriptsDir := filepath.Join(templatesDir, "scripts", "{{SCENARIO_ID}}")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scriptsDir, "deploy.sh"), []byte("#!/bin/bash\n"), 0o644); err != nil {
		t.Fatalf("write deploy.sh: %v", err)
	}
	manifest := map[string]any{
		"name": "reloc-nonproto",
		"relocations": []map[string]any{
			{"from": "scripts/", "to": "scripts/{{SCENARIO_ID}}/"},
		},
	}
	manifestBytes, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(templatesDir, "template.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write template.json: %v", err)
	}
	info, err := loadTemplate(repoRoot, "reloc-nonproto")
	if err != nil {
		t.Fatalf("loadTemplate: %v", err)
	}
	capture := &capturedSubprocess{}
	deps := newRelocationTestDeps(repoRoot, io.Discard, io.Discard, capture)
	issues := validateRelocationProtoSources(deps, struct{}{}, info)
	if len(issues) != 0 {
		t.Fatalf("non-proto relocation produced issues: %#v", issues)
	}
	if len(capture.calls) != 0 {
		t.Fatalf("non-proto relocation invoked %d subprocess calls; should not run buf lint", len(capture.calls))
	}
}

