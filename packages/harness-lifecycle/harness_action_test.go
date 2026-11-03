//go:build !windows

package agentinstall

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	harnesscatalog "github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// harnessFixture is a temp HOME with real files, real symlinks, a real PATH and
// fake binaries that report a version. Only the network and the Dependency
// Analyzer are replaced; neither has a local fixture.
type harnessFixture struct {
	t       *testing.T
	home    string
	harness Harness
	deps    harnessDeps
	gate    gateResult
	rootOwn map[string]bool
	latest  string
	// installed records what the gateway was asked to install.
	installed []string
	approved  []string
	// record stands in for the Dependency Analyzer's approval record.
	record   approvalRecord
	readErr  error
	writeErr error
}

func newHarnessFixture(t *testing.T, resource, binary string) *harnessFixture {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")
	binDir := filepath.Join(home, ".local", "bin")
	nodeDir := filepath.Join(home, "tools")
	for _, dir := range []string{binDir, nodeDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeScript(t, filepath.Join(nodeDir, "node"), "echo v20.19.4")
	t.Setenv("PATH", binDir+":"+nodeDir+":/usr/bin:/bin")

	f := &harnessFixture{t: t, home: home, latest: "2.0.0", rootOwn: map[string]bool{}, gate: gateResult{Reachable: true, Verdict: "approved"}, record: originalApproval()}
	f.harness = Harness{
		Resource: resource, Binary: binary, Home: home, BinDir: binDir, ReviewedVersion: "1.0.0",
		Package: "@fixture/" + binary, Executable: "node_modules/.bin/" + binary,
	}
	f.deps = harnessDeps{
		latest: func(context.Context, Harness) (HarnessSource, string, error) {
			return HarnessSource{Kind: "npm", ID: f.harness.Package}, f.latest, nil
		},
		gate:         func(context.Context, Harness, string, platformDeps) gateResult { return f.gate },
		readApproval: func(context.Context, Harness, platformDeps) (approvalRecord, error) { return f.record, f.readErr },
		writeApproval: func(_ context.Context, _ Harness, record approvalRecord, _ platformDeps) error {
			if f.writeErr != nil {
				return f.writeErr
			}
			f.record = record
			f.approved = append(f.approved, fmt.Sprint(record["versionRange"]))
			return nil
		},
		actor: "agent claude-code session s1",
		// The gateway's effect: a new verified prefix, then the selected link.
		install: func(_ context.Context, _ Spec, ref string) error {
			f.installed = append(f.installed, ref)
			version := ref[strings.LastIndex(ref, "@")+1:]
			return LinkNPMBinary(f.prefix(version), f.harness.Binary, f.harness.BinDir)
		},
		lookPath: cliutil.ResolveAgentBinary,
		version:  InstalledVersion,
		rootOwn: func(info os.FileInfo) bool {
			return f.rootOwn[info.Name()] || ownedByRoot(info)
		},
		now: func() time.Time { return time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC) },
	}
	return f
}

func originalApproval() approvalRecord {
	return approvalRecord{"ecosystem": "npm", "packageName": "@fixture/codex", "versionRange": "1.5.0", "state": "approved", "rangePolicy": "exact", "rationale": "reviewed by a person", "approvedBy": "the reviewer", "approvedDate": "2026-10-27", "lastReviewed": "2026-10-27", "securityNotes": "no scripts", "allowedScenarios": []any{"codex"}}
}

func writeScript(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// prefix lays out one complete managed version and returns its executable.
func (f *harnessFixture) prefix(version string) string {
	f.t.Helper()
	prefix, err := NPMInstallPrefix(f.home, f.harness.Resource, version)
	if err != nil {
		f.t.Fatal(err)
	}
	executable := filepath.Join(prefix, filepath.FromSlash(f.harness.Executable))
	writeScript(f.t, executable, fmt.Sprintf("echo %q", f.harness.Binary+" "+version))
	digest := treeDigest(f.t, filepath.Join(prefix, "node_modules"))
	data, _ := json.Marshal(installationRecord{Resource: f.harness.Resource, Package: f.harness.Package, Version: version, TreeSHA256: digest})
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), data, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return executable
}

func (f *harnessFixture) link(version string) {
	f.t.Helper()
	if err := LinkNPMBinary(f.prefix(version), f.harness.Binary, f.harness.BinDir); err != nil {
		f.t.Fatal(err)
	}
}

func (f *harnessFixture) inspect() HarnessReport {
	f.t.Helper()
	return inspectHarness(context.Background(), f.harness, f.deps)
}

func (f *harnessFixture) run(action string) HarnessActionResult {
	f.t.Helper()
	return runHarnessAction(context.Background(), f.harness, action, f.deps)
}

func wantState(t *testing.T, report HarnessReport, state, action string) {
	t.Helper()
	if report.State != state || report.PrimaryAction != action {
		t.Fatalf("state=%q action=%q, want %q %q: %s\n%+v", report.State, report.PrimaryAction, state, action, report.Reason, report.Checks)
	}
}

func TestStateMissingOffersInstall(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	t.Setenv("PATH", f.harness.BinDir)
	wantState(t, f.inspect(), StateMissing, ActionInstall)
}

func TestMissingLauncherOffersExistingVerifiedCopyWithoutReinstall(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	if err := os.Remove(f.harness.linkPath()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", f.harness.BinDir)
	wantState(t, f.inspect(), StateUnlinked, ActionLinkManagedCopy)
	result := f.run(ActionLinkManagedCopy)
	if !result.Success {
		t.Fatal(result.Message)
	}
	if result.Report.Installed != "1.5.0" {
		t.Fatalf("linked version = %q", result.Report.Installed)
	}
}

func TestStateUpdateReadyOnlyWhenManagedAndBehind(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	report := f.inspect()
	wantState(t, report, StateUpdateReady, ActionUpdate)
	if report.Installed != "1.5.0" || report.ManagedVersion != "1.5.0" || report.Latest != "2.0.0" || report.InstallKind != InstallKindManaged {
		t.Fatalf("report = %+v", report)
	}
	if !strings.Contains(report.Reason, "Running sessions keep") {
		t.Fatalf("reason must state the effect on running sessions: %q", report.Reason)
	}
	f.latest = "1.5.0"
	wantState(t, f.inspect(), StateUpToDate, "")
}

func TestNewerThanReviewedComparesRunningAgainstThePin(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	if !f.inspect().NewerThanReviewed {
		t.Fatal("running 1.5.0 is newer than the reviewed 1.0.0")
	}
	f.harness.ReviewedVersion = "1.5.0"
	if f.inspect().NewerThanReviewed {
		t.Fatal("running the reviewed version is not newer")
	}
	f.harness.ReviewedVersion = "9.0.0"
	if f.inspect().NewerThanReviewed {
		t.Fatal("running older than reviewed is not newer")
	}
}

// A root-owned file inside a user-writable directory is what the old directory
// mode check missed. An unprivileged test cannot create one, so the ownership
// answer is injected for this file; the real path below covers real root files.
func TestRootOwnedFileInWritableDirectoryNeedsSetup(t *testing.T) {
	f := newHarnessFixture(t, "antigravity", "agy")
	writeScript(t, f.harness.linkPath(), `echo "1.1.22"`)
	f.rootOwn["agy"] = true
	report := f.inspect()
	wantState(t, report, StateNeedsSetup, "")
	if report.OperatorCommand != NeedsSetupCommand || report.InstallKind != InstallKindRootOwned {
		t.Fatalf("report = %+v", report)
	}
	before, _ := os.ReadFile(f.harness.linkPath())
	result := f.run(ActionLinkManagedCopy)
	if result.Success {
		t.Fatalf("no action may succeed on a root-owned file: %+v", result)
	}
	after, _ := os.ReadFile(f.harness.linkPath())
	if !bytes.Equal(before, after) {
		t.Fatal("refused action changed the file")
	}
}

func TestMoveAsideRefusesARealRootOwnedFile(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	for _, candidate := range []string{"/bin/sh", "/usr/bin/env"} {
		info, err := os.Stat(candidate)
		if err != nil || !ownedByRoot(info) {
			continue
		}
		if _, err := moveAside(candidate, f.deps); err == nil || !strings.Contains(err.Error(), NeedsSetupCommand) {
			t.Fatalf("moveAside(%s) = %v", candidate, err)
		}
		return
	}
	t.Skip("no root-owned binary on this host")
}

func TestSystemCopyFirstOnPathIsFoundWithNoAction(t *testing.T) {
	info, err := os.Stat("/usr/bin/env")
	if err != nil || !ownedByRoot(info) {
		t.Skip("needs a root-owned /usr/bin/env")
	}
	f := newHarnessFixture(t, "codex", "env")
	f.link("1.5.0")
	t.Setenv("PATH", "/usr/bin:"+f.harness.BinDir)
	report := f.inspect()
	wantState(t, report, StateFound, "")
	if report.CommandPath != "/usr/bin/env" || report.OperatorCommand != "" || report.InstallKind != InstallKindRootOwned {
		t.Fatalf("report = %+v", report)
	}
}

func TestShadowingCopyOnPathReportsWhatRunsAndWhatIsManaged(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	f.link("1.19.4")
	shadow := filepath.Join(f.home, ".opencode", "bin", "opencode")
	writeScript(t, shadow, `echo "1.18.35"`)
	t.Setenv("PATH", filepath.Dir(shadow)+":"+os.Getenv("PATH"))
	report := f.inspect()
	wantState(t, report, StateFound, "")
	if report.InstallState != "found" {
		t.Fatalf("install state = %q", report.InstallState)
	}
	if report.Installed != "1.18.35" || report.ManagedVersion != "1.19.4" || report.CommandPath != shadow {
		t.Fatalf("report = %+v", report)
	}
}

// The API's PATH can order the managed link first while every shell a session
// starts reads a startup file that puts the upstream copy ahead of it.
func TestShellStartupShadowCountsEvenWhenThisProcessPathDisagrees(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	f.link("1.19.4")
	shadow := filepath.Join(f.home, ".opencode", "bin", "opencode")
	writeScript(t, shadow, `echo "1.18.35"`)
	f.harness.InstallerPaths = []string{shadow}
	wantState(t, f.inspect(), StateUpdateReady, ActionUpdate) // no startup line: the copy is inert

	if err := os.WriteFile(filepath.Join(f.home, ".bashrc"), []byte("# export PATH=$HOME/.opencode/bin:$PATH\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantState(t, f.inspect(), StateUpdateReady, ActionUpdate) // a comment changes nothing

	line := "export PATH=" + filepath.Dir(shadow) + ":$PATH\n"
	if err := os.WriteFile(filepath.Join(f.home, ".bashrc"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	report := f.inspect()
	wantState(t, report, StateFound, "")
	if report.CommandPath != shadow || !strings.Contains(report.Reason, "shell startup") {
		t.Fatalf("report = %+v", report)
	}
}

func TestUseManagedCopyLeavesFoundCopyAndStartupAlone(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	f.link("1.19.4")
	shadow := filepath.Join(f.home, ".opencode", "bin", "opencode")
	writeScript(t, shadow, `echo "1.18.35"`)
	f.harness.InstallerPaths = []string{shadow}
	startup := "export PATH=" + filepath.Dir(shadow) + ":$PATH\n"
	bashrc := filepath.Join(f.home, ".bashrc")
	if err := os.WriteFile(bashrc, []byte(startup), 0o644); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(shadow)

	result := f.run(ActionUseManagedCopy)
	if !result.Success {
		t.Fatalf("action failed: %+v", result)
	}
	after, err := os.ReadFile(shadow)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatalf("found copy changed: %v", err)
	}
	if data, _ := os.ReadFile(bashrc); string(data) != startup {
		t.Fatal("the operator's shell startup was edited")
	}
	wantState(t, result.Report, StateFound, "")
}

func TestUseManagedCopyDoesNotMutateFoundCopyWhenNoManagedCopyExists(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	shadow := filepath.Join(f.home, ".opencode", "bin", "opencode")
	writeScript(t, shadow, `echo "1.18.35"`)
	t.Setenv("PATH", filepath.Dir(shadow)+":"+os.Getenv("PATH"))
	f.deps.install = func(context.Context, Spec, string) error {
		t.Fatal("found-copy action must not install or mutate files")
		return nil
	}
	result := f.run(ActionUseManagedCopy)
	if !result.Success {
		t.Fatalf("expected a safe no-op: %+v", result)
	}
	if raw, err := InstalledVersion(context.Background(), shadow); err != nil || !strings.Contains(raw, "1.18.35") {
		t.Fatalf("the original must be restored, got %q %v", raw, err)
	}
}

func TestHarnessReleaseInstallUsesPerCallPlatformDependencies(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	f.link("1.0.0")
	platformErr := errors.New("injected dependency analyzer failure")
	called := false
	f.deps.platform = platformDeps{
		homeDir: func() (string, error) { return f.home, nil },
		commandOutput: func(_ context.Context, name string, _ ...string) ([]byte, error) {
			if name == "node" {
				return []byte("v20.19.4"), nil
			}
			called = true
			return nil, platformErr
		},
	}
	f.deps.installWithDeps = installNPMWithDeps
	f.deps.install = func(context.Context, Spec, string) error {
		t.Fatal("legacy installer unexpectedly used")
		return nil
	}

	result := f.run(ActionUpdate)
	if result.Success || !strings.Contains(result.Message, platformErr.Error()) {
		t.Fatalf("expected the injected installer failure, got %+v", result)
	}
	if !called {
		t.Fatal("release installer did not use the injected platform command runner")
	}
}

func TestStandaloneFileIsFoundAndLinkActionDoesNotRenameIt(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	f.latest = "1.0.70"
	path := f.harness.linkPath()
	writeScript(t, path, `echo "grok 0.2.72"`)
	f.prefix("1.0.70")
	original, _ := os.ReadFile(path)

	report := f.inspect()
	wantState(t, report, StateFound, "")
	if report.InstallKind != InstallKindStandalone {
		t.Fatalf("kind = %s", report.InstallKind)
	}
	result := f.run(ActionLinkManagedCopy)
	if result.Success {
		t.Fatalf("found copy was unexpectedly linked: %+v", result)
	}
	unchanged, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(unchanged, original) {
		t.Fatalf("standalone file changed: %v", err)
	}
	wantState(t, result.Report, StateFound, "")
	if len(f.installed) != 0 {
		t.Fatalf("an existing managed copy needs no install, ran %v", f.installed)
	}
}

func TestLinkDoesNotRenameStandaloneFileWhenManagedCopyDoesNotRun(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	path := f.harness.linkPath()
	writeScript(t, path, `echo "grok 0.2.72"`)
	executable := f.prefix("1.0.70")
	writeScript(t, executable, `exit 7`)
	result := f.run(ActionLinkManagedCopy)
	if result.Success {
		t.Fatalf("expected failure: %+v", result)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("the standalone file must be untouched: %v %v", info, err)
	}
	if raw, _ := InstalledVersion(context.Background(), path); !strings.Contains(raw, "0.2.72") {
		t.Fatalf("restored file reports %q", raw)
	}
}

func TestLinkActionLeavesANewerStandaloneCopyUntouched(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	path := f.harness.linkPath()
	writeScript(t, path, `echo "grok 1.9.0"`)
	f.prefix("1.0.70")
	f.deps.install = func(_ context.Context, _ Spec, ref string) error {
		f.installed = append(f.installed, ref)
		return LinkNPMBinary(f.prefix("2.0.0"), "grok", f.harness.BinDir)
	}
	result := f.run(ActionLinkManagedCopy)
	if result.Success || len(f.installed) != 0 {
		t.Fatalf("result=%+v installed=%v", result, f.installed)
	}
	if raw, err := InstalledVersion(context.Background(), path); err != nil || !strings.Contains(raw, "1.9.0") {
		t.Fatalf("standalone copy changed: %q %v", raw, err)
	}
}

// grokWrapper behaves like @xai-official/grok's bin/grok: it runs the file in its
// home when that exists and otherwise builds it from the package it belongs to.
func grokWrapper(delegate, version string) string {
	return fmt.Sprintf(`delegate=%q
if [ ! -e "$delegate" ]; then
  mkdir -p "$(dirname "$delegate")"
  printf '#!/bin/sh\necho "grok %s"\n' > "$delegate-%s"
  chmod +x "$delegate-%s"
  ln -s "$(basename "$delegate")-%s" "$delegate"
fi
exec "$delegate" "$@"`, delegate, version, version, version, version)
}

func (f *harnessFixture) grokPrefix(version string) string {
	f.t.Helper()
	executable := f.prefix(version)
	writeScript(f.t, executable, grokWrapper(f.harness.DelegatePath, version))
	data, _ := json.Marshal(installationRecord{Resource: "grok", Package: f.harness.Package, Version: version, TreeSHA256: treeDigest(f.t, filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(executable))), "node_modules"))})
	if err := os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(executable))), "installation.json"), data, 0o600); err != nil {
		f.t.Fatal(err)
	}
	return executable
}

func newGrokFixture(t *testing.T) *harnessFixture {
	t.Helper()
	f := newHarnessFixture(t, "grok", "grok")
	f.harness.DelegatePath = filepath.Join(f.home, ".grok", "bin", "grok")
	f.latest = "1.0.70"
	stale := f.harness.DelegatePath + "-0.2.72"
	writeScript(t, stale, `echo "grok 0.2.72 [stable]"`)
	if err := os.Symlink("grok-0.2.72", f.harness.DelegatePath); err != nil {
		t.Fatal(err)
	}
	return f
}

// A managed launcher may delegate to the vendor's own found copy. Report it,
// but never rewrite that found path without an explicit owner action.
func TestGrokStaleDelegateIsFoundAndLeftUnchanged(t *testing.T) {
	f := newGrokFixture(t)
	if err := LinkNPMBinary(f.grokPrefix("1.0.70"), "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	report := f.inspect()
	wantState(t, report, StateFound, "")
	if report.Installed != "0.2.72" || report.ManagedVersion != "1.0.70" || !strings.Contains(report.Reason, f.harness.DelegatePath) {
		t.Fatalf("report = %+v", report)
	}
	result := f.run(ActionUseManagedCopy)
	if !result.Success {
		t.Fatalf("safe keep-yours action failed: %+v", result)
	}
	wantState(t, result.Report, StateFound, "")
	if result.Report.Installed != "0.2.72" {
		t.Fatalf("found copy reports %s", result.Report.Installed)
	}
	if _, err := os.Stat(f.harness.DelegatePath + "-0.2.72"); err != nil {
		t.Fatalf("the older versioned file must stay: %v", err)
	}
}

func TestNativeGrokSelectionPreservesIndependentDelegate(t *testing.T) {
	f := newGrokFixture(t)
	f.link("1.0.70")
	before, err := os.Readlink(f.harness.DelegatePath)
	if err != nil {
		t.Fatal(err)
	}
	change := &harnessChange{}
	if err := f.harness.refreshDelegate(context.Background(), "1.0.70", f.deps, change); err != nil {
		t.Fatal(err)
	}
	change.commit()
	after, err := os.Readlink(f.harness.DelegatePath)
	if err != nil || after != before {
		t.Fatalf("native selection changed the independent delegate: %q, %v", after, err)
	}
}

func TestGrokRollbackRetargetsTheDelegateToo(t *testing.T) {
	f := newGrokFixture(t)
	f.latest = "1.1.0"
	f.grokPrefix("1.0.70")
	if err := LinkNPMBinary(f.grokPrefix("1.1.0"), "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	if res := f.run(ActionUseManagedCopy); !res.Success {
		t.Fatalf("repair: %+v", res)
	}
	result := f.run(ActionRollback)
	if !result.Success || result.Report.Installed != "1.0.70" {
		t.Fatalf("rollback = %+v", result)
	}
}

func TestDelegateRestoredWhenTheLauncherCannotRebuildIt(t *testing.T) {
	f := newGrokFixture(t)
	executable := f.grokPrefix("1.0.70")
	writeScript(t, executable, `echo "broken"`)
	if err := LinkNPMBinary(executable, "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	result := f.run(ActionUseManagedCopy)
	if result.Success {
		t.Fatalf("expected failure: %+v", result)
	}
	if target, err := os.Readlink(f.harness.DelegatePath); err != nil || target != "grok-0.2.72" {
		t.Fatalf("delegate = %q, %v", target, err)
	}
}

func TestUpdateRecordsExactApprovalThenInstallsAndKeepsThePreviousVersion(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	f.gate = gateResult{Reachable: true, Blocked: true, Verdict: "out_of_range", Message: "outside approved range"}
	wantState(t, f.inspect(), StateUpdateReady, ActionUpdate)

	result := f.run(ActionUpdate)
	if !result.Success {
		t.Fatalf("update failed: %+v", result)
	}
	if fmt.Sprint(f.approved) != "[2.0.0]" || fmt.Sprint(f.installed) != "["+f.harness.Package+"@2.0.0]" {
		t.Fatalf("approved=%v installed=%v", f.approved, f.installed)
	}
	wantState(t, result.Report, StateUpToDate, "")
	if result.Report.RollbackVersion != "1.5.0" {
		t.Fatalf("rollback version = %q", result.Report.RollbackVersion)
	}
}

func TestPolicyRefusalAndGatewayOutageAreNeverUpdateReady(t *testing.T) {
	for name, gate := range map[string]gateResult{
		"denied":     {Reachable: true, Blocked: true, Verdict: "denied", Message: "denied"},
		"deprecated": {Reachable: true, Blocked: true, Verdict: "deprecated", Message: "deprecated"},
		"outage":     {Message: "did not answer"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newHarnessFixture(t, "codex", "codex")
			f.link("1.5.0")
			f.gate = gate
			report := f.inspect()
			wantState(t, report, StateCheckFailed, ActionRetry)
			if name == "outage" && report.OperatorCommand == "" {
				t.Fatal("an outage must say what to start")
			}
			if res := f.run(ActionUpdate); res.Success || len(f.installed) != 0 {
				t.Fatalf("update must refuse: %+v, installed %v", res, f.installed)
			}
		})
	}
}

func TestNodeBlocksOnlyWhereNodeRunsThePackage(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	prefix, _ := NPMInstallPrefix(f.home, "codex", "1.5.0")
	manifest := filepath.Join(prefix, "node_modules", filepath.FromSlash(f.harness.Package), "package.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"engines":{"node":">=22"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A native entry point: npm only warns about engines, nothing fails.
	wantState(t, f.inspect(), StateUpdateReady, ActionUpdate)

	executable := filepath.Join(prefix, filepath.FromSlash(f.harness.Executable))
	if err := os.WriteFile(executable, []byte("#!/usr/bin/env node\necho codex 1.5.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if h := f.harness.requiredNodeMajor(); h != 22 {
		t.Fatalf("required node major = %d", h)
	}
	var checks []HarnessCheck
	reason := f.harness.nodeBlocker(context.Background(), func(id string, passed bool, message string) {
		checks = append(checks, HarnessCheck{id, passed, message})
	}, defaultPlatformDeps())
	if !strings.Contains(reason, "22") || len(checks) != 1 || checks[0].Passed {
		t.Fatalf("reason=%q checks=%+v", reason, checks)
	}
}

func TestMissingNodeIsNeverUpdateReady(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	t.Setenv("PATH", f.harness.BinDir)
	wantState(t, f.inspect(), StateCheckFailed, ActionRetry)
}

func TestPackageReportingAnotherVersionWithoutADelegateIsACheckFailure(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	writeScript(t, f.prefix("1.5.0"), `echo "codex 1.4.0"`)
	prefix, _ := NPMInstallPrefix(f.home, "codex", "1.5.0")
	if err := LinkNPMBinary(filepath.Join(prefix, filepath.FromSlash(f.harness.Executable)), "codex", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	wantState(t, f.inspect(), StateCheckFailed, ActionRetry)
}

func TestUnknownUpstreamIsACheckFailureWithoutAnArrow(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	f.deps.latest = func(context.Context, Harness) (HarnessSource, string, error) {
		return HarnessSource{}, "", errors.New("offline")
	}
	report := f.inspect()
	wantState(t, report, StateCheckFailed, ActionRetry)
	if report.Status != upstreamcheck.StatusUnknown {
		t.Fatalf("status = %s", report.Status)
	}
}

func TestFailedUpdateVerificationReselectsThePreviousVersion(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	f.deps.install = func(context.Context, Spec, string) error {
		executable := f.prefix("2.0.0")
		writeScript(t, executable, `echo "codex 1.9.9"`)
		return LinkNPMBinary(executable, "codex", f.harness.BinDir)
	}
	result := f.run(ActionUpdate)
	if result.Success {
		t.Fatalf("expected failure: %+v", result)
	}
	if result.Report.ManagedVersion != "1.5.0" {
		t.Fatalf("selected = %s, want the previous version", result.Report.ManagedVersion)
	}
}

func TestRollbackRelinksThePreviousDirectoryAndUpdateCanReturn(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.prefix("1.4.0")
	f.prefix("1.5.0")
	f.link("1.6.0")
	f.latest = "1.6.0"
	report := f.inspect()
	if report.RollbackVersion != "1.5.0" {
		t.Fatalf("rollback version = %q, want the newest older", report.RollbackVersion)
	}
	result := f.run(ActionRollback)
	if !result.Success || result.Report.Installed != "1.5.0" || result.Report.ManagedVersion != "1.5.0" {
		t.Fatalf("rollback = %+v", result)
	}
	wantState(t, result.Report, StateUpdateReady, ActionUpdate)
	if result.Report.RollbackVersion != "1.4.0" {
		t.Fatalf("next rollback = %q", result.Report.RollbackVersion)
	}
	if len(f.installed) != 0 {
		t.Fatalf("rollback must not install: %v", f.installed)
	}
}

func TestRollbackRefusesAPrefixThatNoLongerMatchesItsReceipt(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	old := f.prefix("1.5.0")
	f.link("1.6.0")
	writeScript(t, old, `echo "tampered 1.5.0"`)
	result := f.run(ActionRollback)
	if result.Success || result.Report.Installed != "1.6.0" {
		t.Fatalf("rollback = %+v", result)
	}
}

func TestIncompletePrefixIsNeverARollbackTarget(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	prefix, _ := NPMInstallPrefix(f.home, "codex", "1.5.0")
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	f.link("1.6.0")
	report := f.inspect()
	if report.RollbackVersion != "" {
		t.Fatalf("rollback version = %q", report.RollbackVersion)
	}
	if res := f.run(ActionRollback); res.Success {
		t.Fatalf("rollback with nothing to return to: %+v", res)
	}
}

func TestRepeatingARepairAfterItWorkedIsANoOp(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	for _, action := range []string{ActionUseManagedCopy, ActionLinkManagedCopy} {
		if res := f.run(action); !res.Success || res.PreservedPath != "" {
			t.Fatalf("%s = %+v", action, res)
		}
	}
	f.latest = "1.5.0"
	if res := f.run(ActionUpdate); !res.Success || len(f.installed) != 0 {
		t.Fatalf("update when current = %+v, installed %v", res, f.installed)
	}
}

func releaseArchive(t *testing.T, entry, script string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(gz)
	if err := archive.WriteHeader(&tar.Header{Name: entry, Mode: 0o755, Size: int64(len(script))}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func newManifestFixture(t *testing.T, version, script string) (*harnessFixture, *[]byte) {
	t.Helper()
	f := newHarnessFixture(t, "antigravity", "agy")
	f.harness.Package, f.harness.Executable, f.harness.ArchiveEntry = "", "agy", "antigravity"
	archive := releaseArchive(t, "antigravity", script)
	digest := sha512.Sum512(archive)
	checksum := hex.EncodeToString(digest[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifests/" + runtime.GOOS + "_" + runtime.GOARCH + ".json":
			_ = json.NewEncoder(w).Encode(releaseManifest{Version: version, URL: "http://" + r.Host + "/agy.tar.gz", SHA512: checksum})
		case "/agy.tar.gz":
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	f.harness.ManifestBase = server.URL
	f.latest = version
	f.deps.latest = func(ctx context.Context, h Harness) (HarnessSource, string, error) {
		release, err := fetchReleaseManifest(ctx, h.ManifestBase, releasePlatform())
		return HarnessSource{Kind: "release-manifest", ID: releasePlatform()}, release.Version, err
	}
	return f, &archive
}

func TestManifestHarnessUpdatesIntoAVersionPrefixAndRollsBack(t *testing.T) {
	f, _ := newManifestFixture(t, "1.4.12", "#!/bin/sh\necho 1.4.12\n")
	// The current install is itself a managed prefix.
	prefix, _ := NPMInstallPrefix(f.home, "antigravity", "1.1.22")
	executable := filepath.Join(prefix, "agy")
	writeScript(t, executable, "echo 1.1.22")
	digest, err := fileDigest(executable)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(installationRecord{Resource: "antigravity", Version: "1.1.22", FileSHA256: digest})
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LinkNPMBinary(executable, "agy", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	report := f.inspect()
	wantState(t, report, StateUpdateReady, ActionUpdate)
	if !hasCheck(report, "updater", true) || hasCheck(report, "dependency_gate", true) {
		t.Fatalf("checks = %+v", report.Checks)
	}

	result := f.run(ActionUpdate)
	if !result.Success {
		t.Fatalf("update: %+v", result)
	}
	wantState(t, result.Report, StateUpToDate, "")
	if result.Report.Installed != "1.4.12" || result.Report.RollbackVersion != "1.1.22" {
		t.Fatalf("report = %+v", result.Report)
	}
	back := f.run(ActionRollback)
	if !back.Success || back.Report.Installed != "1.1.22" {
		t.Fatalf("rollback = %+v", back)
	}
}

func TestManifestLinkInstallsWhenNoPrefixExistsAndKeepsTheStandaloneFile(t *testing.T) {
	f, _ := newManifestFixture(t, "1.4.12", "#!/bin/sh\necho 1.4.12\n")
	path := f.harness.linkPath()
	writeScript(t, path, `echo 1.1.22`)
	wantState(t, f.inspect(), StateFound, "")
	result := f.run(ActionLinkManagedCopy)
	if result.Success {
		t.Fatalf("standalone found copy was unexpectedly linked: %+v", result)
	}
	if raw, _ := InstalledVersion(context.Background(), path); !strings.Contains(raw, "1.1.22") {
		t.Fatalf("found file reports %q", raw)
	}
	wantState(t, result.Report, StateFound, "")
}

func TestManifestChecksumMismatchLeavesNothingSelected(t *testing.T) {
	f := newHarnessFixture(t, "antigravity", "agy")
	f.harness.Package, f.harness.Executable, f.harness.ArchiveEntry = "", "agy", "antigravity"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/manifests/") {
			_ = json.NewEncoder(w).Encode(releaseManifest{Version: "1.4.12", URL: "http://" + r.Host + "/agy.tar.gz", SHA512: strings.Repeat("0", 128)})
			return
		}
		_, _ = w.Write(releaseArchive(t, "antigravity", "#!/bin/sh\necho 1.4.12\n"))
	}))
	defer server.Close()
	f.harness.ManifestBase = server.URL
	err := f.harness.installFromManifest(context.Background(), f.deps)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("err = %v", err)
	}
	if installs, _ := f.harness.managedInstalls(); len(installs) != 0 {
		t.Fatalf("prefix committed despite the mismatch: %+v", installs)
	}
	if _, err := os.Lstat(f.harness.linkPath()); !os.IsNotExist(err) {
		t.Fatalf("link created: %v", err)
	}
}

func TestManifestCandidateReportingAnotherVersionIsRefused(t *testing.T) {
	f, _ := newManifestFixture(t, "1.4.12", "#!/bin/sh\necho 1.0.0\n")
	err := f.harness.installFromManifest(context.Background(), f.deps)
	if err == nil || !strings.Contains(err.Error(), "manifest says") {
		t.Fatalf("err = %v", err)
	}
}

func TestInsecureReleaseURLIsRefused(t *testing.T) {
	if err := requireTransportSecurity("http://example.com/x.tar.gz"); err == nil {
		t.Fatal("plain http to a remote host must be refused")
	}
	if err := requireTransportSecurity("https://example.com/x.tar.gz"); err != nil {
		t.Fatal(err)
	}
}

func TestRootOwnedHarnessPathsSeesOnlyRootOwnedHarnessFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeScript(t, filepath.Join(home, ".local", "bin", "agy"), "echo 1")
	owned, err := RootOwnedHarnessPaths(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 0 {
		t.Fatalf("a user-owned home has nothing to hand back: %v", owned)
	}
}

func TestUninstallRemovesOnlyReceiptOwnedCopies(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	f.harness.Package = "@openai/codex"
	managed := f.prefix("1.5.0")
	f.link("1.5.0")
	found := filepath.Join(f.home, ".codex", "bin", "codex")
	writeScript(t, found, "echo codex 9.0.0")
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if err := Uninstall(context.Background(), root, "codex"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(f.harness.linkPath()); !os.IsNotExist(err) {
		t.Fatalf("managed link remains: %v", err)
	}
	if _, err := os.Stat(managed); !os.IsNotExist(err) {
		t.Fatalf("managed prefix remains: %v", err)
	}
	if _, err := os.Stat(found); err != nil {
		t.Fatalf("found upstream copy was removed: %v", err)
	}
}

func TestUninstallRejectsForgedReceipt(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	executable := f.prefix("1.5.0")
	prefix := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
	data, err := os.ReadFile(filepath.Join(prefix, "installation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var receipt installationRecord
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt.Package = "@attacker/other"
	receipt.TreeSHA256 = strings.Repeat("0", 64)
	data, _ = json.Marshal(receipt)
	if err := os.WriteFile(filepath.Join(prefix, "installation.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHarness(context.Background(), f.harness); err == nil {
		t.Fatal("forged receipt was accepted")
	}
	if _, err := os.Stat(prefix); err != nil {
		t.Fatalf("forged install was removed: %v", err)
	}
}

func TestUninstallDoesNotFollowSymlinkedVersionsRoot(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	executable := f.prefix("1.5.0")
	prefix := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
	versionsRoot := filepath.Dir(prefix)
	outside := filepath.Join(t.TempDir(), "versions")
	if err := os.Rename(versionsRoot, outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, versionsRoot); err != nil {
		t.Fatal(err)
	}
	if err := uninstallHarness(context.Background(), f.harness); err == nil {
		t.Fatal("symlinked versions root was accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, filepath.Base(prefix))); err != nil {
		t.Fatalf("outside install was removed: %v", err)
	}
}

func TestUninstallDoesNotFollowInHomeSymlinkedVersionsAncestor(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	versions, err := f.harness.versionsRoot()
	if err != nil {
		t.Fatal(err)
	}
	ancestor := filepath.Dir(versions)
	if err := os.RemoveAll(ancestor); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(ancestor), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(f.home, "Documents")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, ancestor); err != nil {
		t.Fatal(err)
	}
	executable := f.prefix("1.5.0")
	prefix := filepath.Dir(filepath.Dir(filepath.Dir(executable)))
	if err := uninstallHarness(context.Background(), f.harness); err == nil {
		t.Fatal("in-home symlinked versions ancestor was accepted")
	}
	if _, err := os.Stat(filepath.Join(target, filepath.Base(versions), filepath.Base(prefix))); err != nil {
		t.Fatalf("in-home target install was removed: %v", err)
	}
}

func TestUninstallLeavesStandaloneManagedBinCopy(t *testing.T) {
	f := newHarnessFixture(t, "codex", "codex")
	managed := f.prefix("1.5.0")
	standalone := filepath.Join(f.harness.BinDir, f.harness.Binary)
	writeScript(t, standalone, "echo codex 9.0.0")
	if err := uninstallHarness(context.Background(), f.harness); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(standalone); err != nil {
		t.Fatalf("standalone copy was removed: %v", err)
	}
	if _, err := os.Stat(managed); !os.IsNotExist(err) {
		t.Fatalf("managed prefix remains: %v", err)
	}
}

func TestLoadHarnessReadsTheManifests(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	cases := map[string]struct{ binary, pkg, reviewed string }{
		"claude-code": {"claude", "@anthropic-ai/claude-code", "2.1.185"},
		"grok":        {"grok", "@xai-official/grok", "0.2.72"},
		"antigravity": {"agy", "", "1.0.13"},
	}
	for resource, want := range cases {
		h, err := LoadHarness(root, resource)
		if err != nil {
			t.Fatalf("%s: %v", resource, err)
		}
		if h.Binary != want.binary || h.Package != want.pkg || h.ReviewedVersion != want.reviewed {
			t.Fatalf("%s = %+v", resource, h)
		}
	}
	antigravity, _ := LoadHarness(root, "antigravity")
	if antigravity.ManifestBase != AntigravityManifestBase || antigravity.Executable != "agy" || antigravity.ArchiveEntry != "antigravity" {
		t.Fatalf("antigravity = %+v", antigravity)
	}
	if _, err := LoadHarness(root, "postgres"); err == nil {
		t.Fatal("a non-harness resource must be refused")
	}
}

func TestLoadHarnessUsesInjectedManifestAndHome(t *testing.T) {
	manifestPath := ""
	manifestCalls, homeCalls := 0, 0
	root := t.TempDir()
	h, err := loadHarness(root, "claude-code", harnessLoadDeps{
		loadManifest: func(path string) (harnessManifest, error) {
			manifestPath = path
			manifestCalls++
			return harnessManifest{Name: "claude-code"}, nil
		},
		homeDir: func() (string, error) {
			homeCalls++
			return "/injected/home", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifestCalls != 1 || homeCalls != 1 || manifestPath != filepath.Join(root, "resources", "claude-code", "resource.json") {
		t.Fatalf("manifest calls=%d home calls=%d path=%q", manifestCalls, homeCalls, manifestPath)
	}
	if h.Home != "/injected/home" || h.BinDir != filepath.Join("/injected/home", ".local", "bin") {
		t.Fatalf("harness did not use injected home: %+v", h)
	}
}

func outOfRangeFixture(t *testing.T) *harnessFixture {
	t.Helper()
	f := newHarnessFixture(t, "codex", "codex")
	f.link("1.5.0")
	f.gate = gateResult{Reachable: true, Blocked: true, Verdict: "out_of_range", Message: "outside approved range"}
	return f
}

// The audit trail names who did it and claims no review that did not happen.
func TestApprovalRecordNamesTheRealActorAndClaimsNoReview(t *testing.T) {
	f := outOfRangeFixture(t)
	if result := f.run(ActionUpdate); !result.Success {
		t.Fatalf("update: %+v", result)
	}
	record := f.record
	if record["versionRange"] != "2.0.0" || record["rangePolicy"] != "exact" {
		t.Fatalf("record = %v", record)
	}
	if record["approvedBy"] != "agent claude-code session s1" {
		t.Fatalf("approvedBy = %v", record["approvedBy"])
	}
	rationale := fmt.Sprint(record["rationale"])
	if !strings.Contains(rationale, "harness update to 2.0.0 by agent claude-code session s1") || !strings.Contains(rationale, "No dependency review") {
		t.Fatalf("rationale = %q", rationale)
	}
	if _, present := record["approvedDate"]; present {
		t.Fatalf("approvedDate claims an approval date: %v", record["approvedDate"])
	}
	if record["lastReviewed"] != "2026-10-27" {
		t.Fatalf("lastReviewed moved: %v", record["lastReviewed"])
	}
	if record["securityNotes"] != "no scripts" || record["state"] != "approved" {
		t.Fatalf("notes or state lost: %v", record)
	}
}

func TestFailureAfterApprovalRestoresThePreviousRecord(t *testing.T) {
	t.Run("install fails", func(t *testing.T) {
		f := outOfRangeFixture(t)
		f.deps.install = func(context.Context, Spec, string) error { return errors.New("registry is down") }
		if result := f.run(ActionUpdate); result.Success {
			t.Fatalf("expected failure: %+v", result)
		}
		if !reflect.DeepEqual(f.record, originalApproval()) {
			t.Fatalf("record = %v", f.record)
		}
	})
	t.Run("verification fails", func(t *testing.T) {
		f := outOfRangeFixture(t)
		f.deps.install = func(context.Context, Spec, string) error {
			executable := f.prefix("2.0.0")
			writeScript(t, executable, `echo "codex 1.9.9"`)
			return LinkNPMBinary(executable, "codex", f.harness.BinDir)
		}
		if result := f.run(ActionUpdate); result.Success {
			t.Fatalf("expected failure: %+v", result)
		}
		if !reflect.DeepEqual(f.record, originalApproval()) {
			t.Fatalf("record = %v", f.record)
		}
	})
	t.Run("linking fails after an install", func(t *testing.T) {
		f := newHarnessFixture(t, "codex", "codex")
		f.gate = gateResult{Reachable: true, Blocked: true, Verdict: "out_of_range"}
		writeScript(t, f.harness.linkPath(), `echo "codex 1.0.0"`)
		f.deps.install = func(context.Context, Spec, string) error { return errors.New("registry is down") }
		if result := f.run(ActionLinkManagedCopy); result.Success {
			t.Fatalf("expected failure: %+v", result)
		}
		if !reflect.DeepEqual(f.record, originalApproval()) {
			t.Fatalf("record = %v", f.record)
		}
	})
	t.Run("a failed restoration is reported", func(t *testing.T) {
		f := outOfRangeFixture(t)
		write := f.deps.writeApproval
		f.deps.writeApproval = func(ctx context.Context, h Harness, record approvalRecord, platform platformDeps) error {
			if record["versionRange"] == "1.5.0" {
				return errors.New("gateway down")
			}
			return write(ctx, h, record, platform)
		}
		f.deps.install = func(context.Context, Spec, string) error { return errors.New("registry is down") }
		result := f.run(ActionUpdate)
		if result.Success || !strings.Contains(result.Message, "gateway down") {
			t.Fatalf("result = %+v", result)
		}
	})
}

func TestRollbackRestoresThePriorApproval(t *testing.T) {
	f := outOfRangeFixture(t)
	if result := f.run(ActionUpdate); !result.Success {
		t.Fatalf("update: %+v", result)
	}
	result := f.run(ActionRollback)
	if !result.Success || result.Report.Installed != "1.5.0" {
		t.Fatalf("rollback: %+v", result)
	}
	if !reflect.DeepEqual(f.record, originalApproval()) {
		t.Fatalf("record = %v", f.record)
	}
	if !strings.Contains(result.Message, "approval is back at 1.5.0") {
		t.Fatalf("message = %q", result.Message)
	}
}

func TestRollbackStillWorksWhenTheGatewayIsDownAndSaysSo(t *testing.T) {
	f := outOfRangeFixture(t)
	if result := f.run(ActionUpdate); !result.Success {
		t.Fatalf("update: %+v", result)
	}
	f.readErr = errors.New("gateway down")
	result := f.run(ActionRollback)
	if !result.Success || result.Report.Installed != "1.5.0" || !strings.Contains(result.Message, "still names 2.0.0") {
		t.Fatalf("rollback: %+v", result)
	}
}

func TestRollbackLeavesALaterApprovalDecisionAlone(t *testing.T) {
	f := outOfRangeFixture(t)
	if result := f.run(ActionUpdate); !result.Success {
		t.Fatalf("update: %+v", result)
	}
	later := approvalRecord{"versionRange": "9.9.9"}
	f.record = later
	result := f.run(ActionRollback)
	if !result.Success || !reflect.DeepEqual(f.record, later) || !strings.Contains(result.Message, "9.9.9") {
		t.Fatalf("result=%+v record=%v", result, f.record)
	}
}

// A rename replaces in one step. With a remove first, a failure between the two
// leaves no command at all: the vanished backup here stands in for that moment.
func TestLinkRestoreNeverRemovesTheCommandBeforeReplacingIt(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	path := f.harness.linkPath()
	writeScript(t, path, `echo "grok 0.2.72"`)
	writeScript(t, f.prefix("1.0.70"), `exit 7`)
	version := f.deps.version
	f.deps.version = func(ctx context.Context, target string) (string, error) {
		raw, err := version(ctx, target)
		if err != nil {
			backups, _ := filepath.Glob(path + ".vrooli-replaced-*")
			for _, backup := range backups {
				_ = os.Remove(backup)
			}
		}
		return raw, err
	}
	if result := f.run(ActionLinkManagedCopy); result.Success {
		t.Fatalf("expected failure: %+v", result)
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("a failed restore left no command at %s: %v", path, err)
	}
}

func TestFoundGrokDelegateBlocksUpdateAndStaysUntouched(t *testing.T) {
	f := newGrokFixture(t)
	f.latest = "1.1.0"
	if err := LinkNPMBinary(f.grokPrefix("1.0.70"), "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	before, err := os.Readlink(f.harness.DelegatePath)
	if err != nil {
		t.Fatal(err)
	}
	result := f.run(ActionUpdate)
	if result.Success {
		t.Fatalf("update must not replace the found delegate: %+v", result)
	}
	if target, _ := os.Readlink(f.harness.DelegatePath); target != before {
		t.Fatalf("found delegate changed from %q to %q", before, target)
	}
	if len(f.installed) != 0 {
		t.Fatalf("unexpected install attempts: %v", f.installed)
	}
	if result.Report.State != StateFound || result.Report.Installed != "0.2.72" {
		t.Fatalf("report=%+v", result.Report)
	}
}

func TestRegularFileDelegateFoundRemainsUnchanged(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	f.harness.DelegatePath = filepath.Join(f.home, ".grok", "bin", "grok")
	f.latest = "1.0.70"
	writeScript(t, f.harness.DelegatePath, `echo "grok 0.2.72"`)
	original, _ := os.ReadFile(f.harness.DelegatePath)
	if err := LinkNPMBinary(f.grokPrefix("1.0.70"), "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	if result := f.run(ActionUseManagedCopy); !result.Success {
		t.Fatalf("safe keep-yours result: %+v", result)
	}
	unchanged, err := os.ReadFile(f.harness.DelegatePath)
	if err != nil || !bytes.Equal(unchanged, original) {
		t.Fatalf("found delegate changed: %v", err)
	}
	if left, _ := filepath.Glob(f.harness.DelegatePath + ".vrooli-replaced-*"); len(left) != 0 {
		t.Fatalf("moved-aside delegate kept: %v", left)
	}
}

func TestRegularFileDelegateComesBackWhenTheRebuildFails(t *testing.T) {
	f := newHarnessFixture(t, "grok", "grok")
	f.harness.DelegatePath = filepath.Join(f.home, ".grok", "bin", "grok")
	writeScript(t, f.harness.DelegatePath, `echo "grok 0.2.72"`)
	executable := f.prefix("1.0.70")
	writeScript(t, executable, `echo "broken"`)
	if err := LinkNPMBinary(executable, "grok", f.harness.BinDir); err != nil {
		t.Fatal(err)
	}
	if result := f.run(ActionUseManagedCopy); result.Success {
		t.Fatalf("expected failure: %+v", result)
	}
	if raw, _ := InstalledVersion(context.Background(), f.harness.DelegatePath); !strings.Contains(raw, "0.2.72") {
		t.Fatalf("delegate reports %q", raw)
	}
}

func TestReleaseDownloadMustStayOnTheManifestHostOrAllowlist(t *testing.T) {
	const base = AntigravityManifestBase
	allowed := []string{
		base + "/artifact.tar.gz",
		"https://storage.googleapis.com/antigravity-public/antigravity-cli/1.4.12-1/linux-x64/cli.tar.gz",
	}
	refused := []string{
		"https://evil.example/cli.tar.gz",
		"https://storage.googleapis.com/someone-elses-bucket/cli.tar.gz",
		"https://storage.googleapis.com/antigravity-public/../someone-elses-bucket/cli.tar.gz",
		"https://storage.googleapis.com/antigravity-public/%2e%2e/someone-elses-bucket/cli.tar.gz",
		"http://storage.googleapis.com/antigravity-public/cli.tar.gz",
	}
	for _, target := range allowed {
		if err := requireArtifactLocation(base, target); err != nil {
			t.Errorf("%s: %v", target, err)
		}
	}
	for _, target := range refused {
		if err := requireArtifactLocation(base, target); err == nil {
			t.Errorf("%s was allowed", target)
		}
	}
}

func TestManifestPointingAtAnotherHostInstallsNothing(t *testing.T) {
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(releaseArchive(t, "antigravity", "#!/bin/sh\necho 1.4.12\n"))
	}))
	defer foreign.Close()
	archive := releaseArchive(t, "antigravity", "#!/bin/sh\necho 1.4.12\n")
	digest := sha512.Sum512(archive)
	f := newHarnessFixture(t, "antigravity", "agy")
	f.harness.Package, f.harness.Executable, f.harness.ArchiveEntry = "", "agy", "antigravity"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(releaseManifest{Version: "1.4.12", URL: foreign.URL + "/agy.tar.gz", SHA512: hex.EncodeToString(digest[:])})
	}))
	defer server.Close()
	f.harness.ManifestBase = server.URL
	err := f.harness.installFromManifest(context.Background(), f.deps)
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("err = %v", err)
	}
	if installs, _ := f.harness.managedInstalls(); len(installs) != 0 {
		t.Fatalf("installed from a foreign host: %+v", installs)
	}
}

func TestReleaseRedirectToAnotherHostIsRefused(t *testing.T) {
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	defer foreign.Close()
	origin := httptest.NewServer(http.RedirectHandler(foreign.URL+"/x", http.StatusFound))
	defer origin.Close()
	destination := filepath.Join(t.TempDir(), "download")
	err := downloadVerifiedSHA512(context.Background(), origin.URL, origin.URL+"/agy.tar.gz", destination, strings.Repeat("0", 128))
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("err = %v", err)
	}
}

func TestReleaseDownloadIsCappedWithOrWithoutAContentLength(t *testing.T) {
	previous := releaseDownloadLimit
	releaseDownloadLimit = 16
	t.Cleanup(func() { releaseDownloadLimit = previous })
	body := bytes.Repeat([]byte("x"), 64)
	digest := sha512.Sum512(body)
	want := hex.EncodeToString(digest[:])
	for name, handler := range map[string]http.HandlerFunc{
		"declared": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) },
		"streamed": func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			_, _ = w.Write(body)
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()
			err := downloadVerifiedSHA512(context.Background(), server.URL, server.URL+"/x", filepath.Join(t.TempDir(), "download"), want)
			if err == nil || !strings.Contains(err.Error(), "limit") {
				t.Fatalf("a correct checksum must not excuse the size: %v", err)
			}
		})
	}
}

func TestArchiveAcceptsOnlyARegularFileAsTheExecutable(t *testing.T) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(gz)
	script := "#!/bin/sh\necho real\n"
	for _, header := range []*tar.Header{
		{Name: "docs/antigravity", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777},
		{Name: "antigravity", Typeflag: tar.TypeDir, Mode: 0o755},
	} {
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.WriteHeader(&tar.Header{Name: "bin/antigravity", Typeflag: tar.TypeReg, Mode: 0o755, Size: int64(len(script))}); err != nil {
		t.Fatal(err)
	}
	if _, err := archive.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	_ = archive.Close()
	_ = gz.Close()
	dir := t.TempDir()
	source := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(source, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "agy")
	if err := extractTarGz(source, target, "antigravity"); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(target); string(data) != script {
		t.Fatalf("extracted %q", data)
	}
}

func TestSafeguardScopeIgnoresGrokHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	shared := t.TempDir()
	t.Setenv("GROK_HOME", shared)
	candidates, err := harnessFileCandidates(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, shared) {
			t.Fatalf("GROK_HOME reached the elevated scope: %s", candidate)
		}
	}
	want := filepath.Join(home, ".grok", "bin", "grok")
	if !slices.Contains(candidates, want) {
		t.Fatalf("%s missing from %v", want, candidates)
	}
}

// Real ownership: ~/.local resolves to /usr, so ~/.local/bin is the real root-owned
// /usr/bin. It reads as under home, and handing it back would be a disaster.
func TestSafeguardScopeRefusesAParentThatResolvesOutsideHome(t *testing.T) {
	info, err := os.Stat("/usr/bin")
	if err != nil || !ownedByRoot(info) {
		t.Skip("needs a root-owned /usr/bin")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.Symlink("/usr", filepath.Join(home, ".local")); err != nil {
		t.Fatal(err)
	}
	owned, err := RootOwnedHarnessPaths(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 0 {
		t.Fatalf("listed paths outside home: %v", owned)
	}
}

func TestRootOwnedDelegateOutsideHomeDoesNotPromiseSetup(t *testing.T) {
	info, err := os.Lstat("/usr/bin/env")
	if err != nil || !ownedByRoot(info) {
		t.Skip("needs a root-owned /usr/bin/env")
	}
	f := newHarnessFixture(t, "grok", "grok")
	f.harness.DelegatePath = "/usr/bin/env"
	f.link("1.0.70")
	report := f.inspect()
	wantState(t, report, StateCheckFailed, ActionRetry)
	if report.OperatorCommand != "" {
		t.Fatalf("setup will not touch this path: %+v", report)
	}
}

func hasCheck(report HarnessReport, id string, passed bool) bool {
	for _, check := range report.Checks {
		if check.ID == id && check.Passed == passed {
			return true
		}
	}
	return false
}

func treeDigest(t *testing.T, root string) string {
	t.Helper()
	digest, err := binaryfetch.TreeDigest(root)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func fileDigest(path string) (string, error) {
	return binaryfetch.FileDigestContext(context.Background(), path)
}

// TestHarnessLifecycleStateEventMatrix is SM7's observation/action contract.
// States are observations of files, selection, upstream and policy, not durable
// workflow phases. "shadowed" is retained wire vocabulary: a PATH shadow now
// observes "found". Updating/rolled-back/removed are action outcomes, followed by
// a fresh observation. Install dispatch is exercised at installRelease's exact-
// version owner seam; network/gateway effects use the existing harness fixture.
// The observation is not the complete input: managed rows include a verified
// rollback candidate; check_failed here means policy denial; needs_setup injects
// ownership detection and an install permission failure. Other failure partitions
// remain covered by the neighboring receipt, upstream, manifest and approval tests.
func TestHarnessLifecycleStateEventMatrix(t *testing.T) {
	// Each cell independently declares success and the observation after the
	// event. A denied action must not change selection, receipts or approval.
	type outcome struct {
		success bool
		state   string
	}
	y := func(state string) outcome { return outcome{true, state} }
	n := func(state string) outcome { return outcome{false, state} }
	events := []string{"check", "install", ActionUpdate, ActionUseManagedCopy, ActionLinkManagedCopy, ActionRollback, "preview_cleanup", "uninstall", "unknown", "retry", "doctor"}
	rows := []struct {
		state, observed string
		outcomes        []outcome
	}{
		{StateMissing, StateMissing, []outcome{y(StateMissing), y(StateUpToDate), n(StateMissing), n(StateMissing), n(StateMissing), n(StateMissing), y(StateMissing), y(StateMissing), n(StateMissing)}},
		{StateFound, StateFound, []outcome{y(StateFound), n(StateFound), n(StateFound), y(StateFound), n(StateFound), n(StateFound), y(StateFound), y(StateFound), n(StateFound)}},
		{StateShadowed, StateFound, []outcome{y(StateFound), n(StateFound), n(StateFound), y(StateFound), n(StateFound), n(StateFound), y(StateFound), y(StateFound), n(StateFound)}},
		{StateUnlinked, StateUnlinked, []outcome{y(StateUnlinked), y(StateUpToDate), n(StateUnlinked), n(StateUnlinked), y(StateUpdateReady), n(StateUnlinked), y(StateUnlinked), y(StateMissing), n(StateUnlinked)}},
		{StateNeedsSetup, StateNeedsSetup, []outcome{y(StateNeedsSetup), n(StateNeedsSetup), n(StateNeedsSetup), n(StateNeedsSetup), n(StateNeedsSetup), n(StateNeedsSetup), y(StateNeedsSetup), y(StateMissing), n(StateNeedsSetup)}},
		{StateUpdateReady, StateUpdateReady, []outcome{y(StateUpdateReady), y(StateUpToDate), y(StateUpToDate), y(StateUpdateReady), y(StateUpdateReady), y(StateUpdateReady), y(StateUpdateReady), y(StateMissing), n(StateUpdateReady)}},
		{StateUpToDate, StateUpToDate, []outcome{y(StateUpToDate), y(StateUpToDate), y(StateUpToDate), y(StateUpToDate), y(StateUpToDate), y(StateUpdateReady), y(StateUpToDate), y(StateMissing), n(StateUpToDate)}},
		{StateCheckFailed, StateCheckFailed, []outcome{n(StateCheckFailed), n(StateCheckFailed), n(StateCheckFailed), n(StateCheckFailed), n(StateCheckFailed), y(StateCheckFailed), y(StateCheckFailed), y(StateMissing), n(StateCheckFailed)}},
	}
	seen := map[string]bool{}
	for _, row := range rows {
		// Retry and doctor re-observe the same immutable inputs as check.
		row.outcomes = append(row.outcomes, row.outcomes[0], row.outcomes[0])
		if seen[row.state] || len(row.outcomes) != len(events) {
			t.Fatalf("duplicate or incomplete row: %s", row.state)
		}
		seen[row.state] = true
		for i, event := range events {
			t.Run(row.state+"/"+event, func(t *testing.T) {
				f := harnessMatrixFixture(t, row.state)
				before := f.inspect()
				if before.State != row.observed {
					t.Fatalf("fixture: got %s, want %s: %+v", before.State, row.observed, before)
				}
				beforeLink, _ := os.Readlink(f.harness.linkPath())
				beforeInstalls, err := f.harness.managedInstalls()
				if err != nil {
					t.Fatal(err)
				}
				beforeApproval, _ := json.Marshal(f.record)
				found := filepath.Join(f.home, "found", f.harness.Binary)
				foundInfo, _ := os.Stat(found)
				foundBytes, _ := os.ReadFile(found)
				result := harnessMatrixEvent(t, f, event)
				want := row.outcomes[i]
				if result.Success != want.success || result.Report.State != want.state {
					t.Fatalf("got success=%v state=%s; want success=%v state=%s: %s", result.Success, result.Report.State, want.success, want.state, result.Message)
				}
				harnessObservationInvariants(t, result.Report)
				if foundInfo != nil {
					afterInfo, err := os.Stat(found)
					afterBytes, readErr := os.ReadFile(found)
					if err != nil || readErr != nil || !os.SameFile(foundInfo, afterInfo) || !bytes.Equal(foundBytes, afterBytes) || foundInfo.Mode() != afterInfo.Mode() || !foundInfo.ModTime().Equal(afterInfo.ModTime()) {
						t.Fatal("independent found copy was altered")
					}
					if backups, _ := filepath.Glob(found + ".vrooli-replaced-*"); len(backups) != 0 {
						t.Fatal("found copy was moved aside")
					}
				}
				// Installation can stage a verified managed copy before discovering
				// an independent PATH selection; that does not rename the found copy.
				if !result.Success && event != "install" {
					afterLink, _ := os.Readlink(f.harness.linkPath())
					afterInstalls, err := f.harness.managedInstalls()
					afterApproval, _ := json.Marshal(f.record)
					if err != nil || beforeLink != afterLink || !reflect.DeepEqual(beforeInstalls, afterInstalls) || !bytes.Equal(beforeApproval, afterApproval) {
						t.Fatal("denied action changed managed state")
					}
				}
			})
		}
	}
	harnessMatrixInventory(t, seen, events)
}

func harnessMatrixFixture(t *testing.T, state string) *harnessFixture {
	t.Helper()
	f := newHarnessFixture(t, "codex", "codex")
	t.Setenv("PATH", f.harness.BinDir+":"+filepath.Join(f.home, "tools"))
	// Empty managed inventory is a valid removal no-op; do not accidentally
	// conflate it with an unavailable/symlinked versions root.
	root, err := f.harness.versionsRoot()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if state != StateMissing && state != StateFound {
		f.prefix("1.0.0")
		f.link("1.5.0")
	}
	switch state {
	case StateFound, StateShadowed:
		found := filepath.Join(f.home, "found", "codex")
		writeScript(t, found, `echo "codex 1.2.0"`)
		t.Setenv("PATH", filepath.Dir(found)+":"+os.Getenv("PATH"))
	case StateUnlinked:
		if err := os.Remove(f.harness.linkPath()); err != nil {
			t.Fatal(err)
		}
	case StateNeedsSetup:
		f.rootOwn["codex"] = true
		// The ownership seam models a real non-writable install effect without
		// requiring privileged chown in a test.
		f.deps.install = func(context.Context, Spec, string) error { return os.ErrPermission }
	case StateUpToDate:
		f.link("2.0.0")
	case StateCheckFailed:
		f.gate = gateResult{Reachable: true, Blocked: true, Verdict: "denied", Message: "package refused"}
	}
	return f
}

// Event adapter calls the existing production owners. Retry and doctor are
// observation aliases; refresh_catalog changes model data, outside this machine.
func harnessMatrixEvent(t *testing.T, f *harnessFixture, event string) HarnessActionResult {
	t.Helper()
	result := HarnessActionResult{Action: event}
	switch event {
	case "check", "retry", "doctor":
		result.Report = f.inspect()
		result.Success = result.Report.State != StateCheckFailed
		return result
	case "install":
		change := &harnessChange{}
		err := f.harness.installRelease(context.Background(), f.latest, f.deps, change)
		if err != nil {
			result.Message = err.Error()
			_ = change.revert(context.Background())
		} else {
			result.Success = true
			change.commit()
		}
	case "preview_cleanup":
		plan, err := previewCleanup(f.harness)
		result.CleanupPlan, result.Success = &plan, err == nil
		if err != nil {
			result.Message = err.Error()
		}
	case "uninstall":
		err := uninstallHarness(context.Background(), f.harness)
		result.Success = err == nil
		if err != nil {
			result.Message = err.Error()
		}
	default:
		return f.run(event)
	}
	result.Report = f.inspect()
	return result
}

func harnessObservationInvariants(t *testing.T, report HarnessReport) {
	t.Helper()
	if report.SignInState != "unknown" {
		t.Fatalf("installation alone invented sign-in: %+v", report)
	}
	if report.State == StateUpdateReady && (report.InstallState != "managed" || report.UpdateState != "available" || report.PrimaryAction != ActionUpdate || report.Installed != report.ManagedVersion) {
		t.Fatalf("contradictory update offer: %+v", report)
	}
	if report.State == StateMissing && (report.InstallState != "absent" || report.PrimaryAction != ActionInstall) {
		t.Fatalf("contradictory missing observation: %+v", report)
	}
	if report.State == StateFound && report.PrimaryAction != "" {
		t.Fatalf("found copy offers a managed mutation: %+v", report)
	}
	if report.State == StateNeedsSetup && (report.PrimaryAction != "" || report.OperatorCommand != NeedsSetupCommand) {
		t.Fatalf("ownership repair offered an unsupported action: %+v", report)
	}
}

func TestHarnessLifecycleFailureRecoveryTrace(t *testing.T) {
	f := outOfRangeFixture(t)
	t.Setenv("PATH", f.harness.BinDir+":"+filepath.Join(f.home, "tools"))
	install := f.deps.install
	f.deps.install = func(context.Context, Spec, string) error { return errors.New("download interrupted") }
	failed := f.run(ActionUpdate)
	if failed.Success || failed.Report.State != StateUpdateReady || failed.Report.ManagedVersion != "1.5.0" || !reflect.DeepEqual(f.record, originalApproval()) {
		t.Fatalf("failed update did not restore selection and approval: %+v", failed)
	}
	f.deps.install = install
	for _, step := range []struct{ event, state, version string }{
		{"retry", StateUpdateReady, "1.5.0"},
		{ActionUpdate, StateUpToDate, "2.0.0"},
		{ActionUpdate, StateUpToDate, "2.0.0"}, // duplicate submit is a no-op
		{ActionRollback, StateUpdateReady, "1.5.0"},
		{ActionUpdate, StateUpToDate, "2.0.0"},
		{"uninstall", StateMissing, ""},
		{"uninstall", StateMissing, ""},
	} {
		result := harnessMatrixEvent(t, f, step.event)
		if !result.Success || result.Report.State != step.state || result.Report.ManagedVersion != step.version {
			t.Fatalf("%s: %+v", step.event, result)
		}
		harnessObservationInvariants(t, result.Report)
	}
}

// This is a vocabulary freshness check, not a substitute for behavioral replay.
// A newly declared state/action must get an explicit matrix row/column.
func harnessMatrixInventory(t *testing.T, states map[string]bool, events []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "eligibility.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	ast.Inspect(file, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			if !strings.HasPrefix(name.Name, "State") && !strings.HasPrefix(name.Name, "Action") {
				continue
			}
			literal, ok := spec.Values[i].(*ast.BasicLit)
			if !ok {
				t.Fatalf("inventory %s is no longer a literal; update the vocabulary check", name.Name)
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.HasPrefix(name.Name, "State") {
				count++
				if !states[value] {
					t.Errorf("state %s has no matrix row", value)
				}
			} else if !slices.Contains(events, value) {
				t.Errorf("action %s has no matrix column", value)
			}
		}
		return true
	})
	if count != len(states) {
		t.Fatalf("matrix inventory=%d, production=%d", len(states), count)
	}
}

func TestHarnessLifecycleRecordedJ8TraceReplay(t *testing.T) {
	var fixture struct {
		Schema int `json:"schema"`
		Traces []struct {
			Name, Agent, Source, SHA256 string
			Events                      []struct {
				Action  string
				Success bool
				Line    int
			}
		}
	}
	data, err := os.ReadFile("testdata/j8-lifecycle-traces.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Schema != 1 || len(fixture.Traces) != 20 {
		t.Fatal("expected both real J8 modes, five harnesses and two viewports")
	}
	seen := map[string]bool{}
	for _, trace := range fixture.Traces {
		if seen[trace.Name] || len(trace.SHA256) != 64 || trace.Source == "" || len(trace.Events) != 7 {
			t.Fatalf("invalid trace provenance: %+v", trace)
		}
		seen[trace.Name] = true
		t.Run(trace.Name, func(t *testing.T) {
			resource := trace.Agent
			if resource == "claude" {
				resource = "claude-code"
			}
			if resource == "agy" {
				resource = "antigravity"
			}
			f := newHarnessFixture(t, resource, trace.Agent)
			t.Setenv("PATH", f.harness.BinDir+":"+filepath.Join(f.home, "tools"))
			if report := f.inspect(); report.State != StateMissing {
				t.Fatalf("initial observation: %+v", report)
			}
			var preview *CleanupPlan
			for i, event := range trace.Events {
				if !event.Success || event.Line <= 0 {
					t.Fatalf("unexpected historical outcome: %+v", event)
				}
				// The real browser journey explicitly resets an older receipt-owned
				// release between removal and update. This is input staging, not a
				// claimed production transition from missing directly to update_ready.
				if event.Action == ActionUpdate {
					f.link("1.5.0")
					if report := f.inspect(); report.State != StateUpdateReady {
						t.Fatalf("staged older release: %+v", report)
					}
				}
				result := harnessMatrixEvent(t, f, event.Action)
				wantState := StateUpToDate
				if event.Action == "uninstall" {
					wantState = StateMissing
				}
				if result.Success != event.Success || result.Report.State != wantState {
					t.Fatalf("source %s:%d, step %d %s: %+v", trace.Source, event.Line, i, event.Action, result)
				}
				harnessObservationInvariants(t, result.Report)
				if event.Action == "preview_cleanup" {
					preview = result.CleanupPlan
				}
				if event.Action == "uninstall" {
					if preview == nil || len(preview.Remove) == 0 {
						t.Fatal("recorded removal has no prior managed preview")
					}
					for _, path := range preview.Remove {
						if _, err := os.Lstat(path); !os.IsNotExist(err) {
							t.Fatalf("previewed removal still present: %s (%v)", path, err)
						}
					}
					for _, path := range preview.Preserved {
						if _, err := os.Stat(path); err != nil {
							t.Fatalf("preserved path disappeared: %s (%v)", path, err)
						}
					}
					preview = nil
				}
			}
		})
	}
}

func TestHarnessThreeFactsRemainIndependentAcrossObservationStates(t *testing.T) {
	states := []string{StateMissing, StateFound, StateShadowed, StateUnlinked, StateNeedsSetup, StateUpdateReady, StateUpToDate, StateCheckFailed}
	for _, state := range states {
		for _, signin := range []string{"signed_in", "signed_out", "unknown"} {
			t.Run(state+"/"+signin, func(t *testing.T) {
				f := harnessMatrixFixture(t, state)
				before := f.inspect()
				f.harness.SignInProbe = &harnesscatalog.SignInProbe{Args: []string{"status"}, SignedInPrefixes: []string{"Logged in using "}, SignedOutLines: []string{"Not logged in"}}
				status := map[string]string{"signed_in": "Logged in using test account", "signed_out": "Not logged in", "unknown": "status unavailable"}[signin]
				installs, err := f.harness.managedInstalls()
				if err != nil {
					t.Fatal(err)
				}
				paths := []string{filepath.Join(f.home, "found", f.harness.Binary)}
				for _, install := range installs {
					paths = append(paths, install.Executable)
				}
				for _, path := range paths {
					body, err := os.ReadFile(path)
					if os.IsNotExist(err) {
						continue
					}
					if err != nil {
						t.Fatal(err)
					}
					writeScript(t, path, "if [ \"$1\" = status ]; then echo "+strconv.Quote(status)+"; exit 0; fi\n"+strings.TrimPrefix(string(body), "#!/bin/sh\n"))
				}
				// Receipts describe the test binaries actually placed on disk.
				for _, install := range installs {
					receipt, err := readInstallationRecord(install.Prefix)
					if err != nil {
						t.Fatal(err)
					}
					receipt.TreeSHA256 = treeDigest(t, filepath.Join(install.Prefix, "node_modules"))
					data, err := json.Marshal(receipt)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(install.Prefix, "installation.json"), data, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				after := f.inspect()
				want := signin
				if state == StateMissing || state == StateUnlinked {
					want = "unknown"
				}
				if after.SignInState != want || after.State != before.State || after.InstallState != before.InstallState || after.UpdateState != before.UpdateState || after.PrimaryAction != before.PrimaryAction {
					t.Fatalf("sign-in changed independent install/update facts: before=%+v after=%+v", before, after)
				}
			})
		}
	}
}

func TestHarnessLifecycleUpdatingRejectsConcurrentMutation(t *testing.T) {
	f := harnessMatrixFixture(t, StateUpdateReady)
	install := f.deps.install
	f.deps.install = func(ctx context.Context, spec Spec, ref string) error {
		// The updater holds the real per-resource lock, with the prior runnable
		// version selected until publication. There is no fabricated "updating"
		// observation that could erase installed/sign-in/update facts.
		if during := f.inspect(); during.State != StateUpdateReady || during.ManagedVersion != "1.5.0" {
			t.Fatalf("in-progress observation: %+v", during)
		}
		contender, cancel := context.WithCancel(ctx)
		cancel()
		blocked := runHarnessAction(contender, f.harness, ActionRollback, f.deps)
		if blocked.Success || !strings.Contains(blocked.Message, "Another action") || blocked.Report.ManagedVersion != "1.5.0" {
			t.Fatalf("concurrent cancelled mutation escaped the lock: %+v", blocked)
		}
		return install(ctx, spec, ref)
	}
	result := f.run(ActionUpdate)
	if !result.Success || result.Report.State != StateUpToDate || len(f.installed) != 1 {
		t.Fatalf("active update did not complete exactly once: %+v (%v)", result, f.installed)
	}
}
