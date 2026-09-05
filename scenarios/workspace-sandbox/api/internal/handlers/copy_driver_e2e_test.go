// Package handlers_test also carries the copy-driver end-to-end proof.
//
// copy_driver_e2e_test.go exercises the exact code path a macOS host takes:
// the copy driver (ContainmentNone, no path illusion, host-path workspace)
// driven through the production HTTP handler stack over a real sqlite repo.
// Copy-driver semantics are OS-independent — a Linux run with the driver
// FORCED to copy is representative of what a darwin host selects — so this
// test is a permanent, always-on suite member rather than a darwin-gated one.
//
// The arc: force copy via the durable driver preference, create a sandbox
// (real CopyDriver.Mount copies the scope), assert the negotiated workspace
// contract (workspacePath == MergedDir, pathIllusion == false, containment
// backend "none"), mutate the workspace through /exec (direct exec in
// MergedDir) and /processes (background process + logs + structured exit),
// read the diff (CopyStrategy: add + modify + a delete detected via the
// lower-tree walk), approve/apply, then confirm the applied files landed in
// the canonical repo and a provenance record was written.
package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/driverid"
	"workspace-sandbox/internal/driverpref"
	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"

	db "github.com/vrooli/api-core/databasetest"
	httpx "github.com/vrooli/api-core/servertest"

	"github.com/vrooli/api-core/schedule"
)

// forceCopyDriver writes the durable driver preference into baseDir and
// resolves it through the real boot-time selection path, proving the copy
// driver is what a host with this preference actually gets. Returns the
// selected driver (already configured with baseDir) for reuse in the
// service and the handler's driver slot.
func forceCopyDriver(t *testing.T, baseDir string, starter process.Starter) driver.Driver {
	t.Helper()

	if err := driverpref.Save(baseDir, driverid.Copy); err != nil {
		t.Fatalf("driverpref.Save(copy): %v", err)
	}

	deps := driver.Deps{
		Clock:   schedule.System(),
		Mounter: fsmount.NewSystemMounter(starter),
		Starter: starter,
	}
	drv, report, err := driver.SelectDriverWithPreference(
		context.Background(),
		driver.Config{BaseDir: baseDir},
		deps,
	)
	if err != nil {
		t.Fatalf("SelectDriverWithPreference: %v", err)
	}
	if !report.PreferenceUsed {
		t.Fatalf("selection did not honor the saved preference: %+v", report)
	}
	if report.Selected != driverid.Copy {
		t.Fatalf("selected driver = %q, want %q", report.Selected, driverid.Copy)
	}
	if drv.ID() != driver.DriverCopy {
		t.Fatalf("resolved driver ID = %q, want %q", drv.ID(), driver.DriverCopy)
	}
	if drv.RequiredContainment() != driver.ContainmentNone {
		t.Fatalf("copy driver RequiredContainment = %v, want ContainmentNone", drv.RequiredContainment())
	}
	return drv
}

// TestCopyDriverSelectionForcesCopy pins that the driver-preference
// mechanism a darwin host relies on (write preference → boot selection)
// resolves to the copy driver with ContainmentNone. This is the seam the
// end-to-end test builds on, asserted in isolation so a selection
// regression is diagnosed here rather than deep in the arc.
func TestCopyDriverSelectionForcesCopy(t *testing.T) {
	drv := forceCopyDriver(t, t.TempDir(), process.NewOSExecStarter())

	// The copy driver reports no path illusion and an identity workspace:
	// the agent sees the host merged dir, and no enforcements are attributed.
	const merged = "/tmp/ws-copy/merged"
	wsPath, illusion, cont := driver.DeriveWorkspaceLayout(drv.RequiredContainment(), nil, merged)
	if wsPath != merged {
		t.Errorf("workspacePath = %q, want %q (identity layout)", wsPath, merged)
	}
	if illusion {
		t.Error("pathIllusion = true, want false for the copy driver")
	}
	if cont.Backend != "none" {
		t.Errorf("containment backend = %q, want none", cont.Backend)
	}
	if len(cont.Enforcements) != 0 {
		t.Errorf("enforcements = %v, want none", cont.Enforcements)
	}
}

// copyE2E is the fully-wired live server for the copy-driver arc: a real
// CopyDriver, a real sqlite-backed Service, a real process tracker/logger,
// and the production handler routes behind the production middleware.
type copyE2E struct {
	live        *httpx.LiveServer
	tracker     *process.Tracker
	repo        *repository.SandboxRepository
	projectRoot string
	scopePath   string
}

// newCopyE2E builds the harness. The scope IS the project root (whole-repo
// mount) so change paths, applied paths, and provenance paths all agree with
// no scope-prefix bookkeeping — the same shape agent-manager uses.
func newCopyE2E(t *testing.T) *copyE2E {
	t.Helper()

	tmp := t.TempDir()
	// Pin storage-class roots (blobstore, profiles) under the test temp dir;
	// the user-profile default resolves under the operator runtime home.
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)

	baseDir := filepath.Join(tmp, "sandbox-base")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		t.Fatalf("mkdir baseDir: %v", err)
	}

	// Canonical repo seeded with the files the arc mutates.
	projectRoot := filepath.Join(tmp, "project")
	if err := os.MkdirAll(projectRoot, 0o755); err != nil {
		t.Fatalf("mkdir projectRoot: %v", err)
	}
	seed := map[string]string{
		"mod.txt":  "v1\n",
		"drop.txt": "delete me\n",
		"keep.txt": "unchanged\n",
	}
	for name, content := range seed {
		if err := os.WriteFile(filepath.Join(projectRoot, name), []byte(content), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	clk := schedule.System()
	starter := process.NewOSExecStarter()
	drv := forceCopyDriver(t, baseDir, starter)

	sqliteDB := db.NewSQLite(t)
	repo := repository.NewSandboxRepository(sqliteDB, clk)
	archiveRepo := repository.NewArchiveRepository(sqliteDB, clk)

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli-copy-e2e",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		t.Fatalf("storage.NewResolver: %v", err)
	}
	blobs, err := blobstore.New(resolver)
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}

	svc := sandbox.NewService(
		repo, drv,
		sandbox.ServiceConfig{DefaultProjectRoot: projectRoot, MaxSandboxes: 100},
		clk,
		audit.NewRepoEmitter(repo.LogAuditEvent, clk),
		starter,
		sandbox.WithGitOps(mocks.NewFakeGitOps()),
		sandbox.WithArchive(archiveRepo, blobs),
	)

	tracker := process.NewTrackerWithConfig(process.TrackerConfig{
		GracePeriod: 2 * time.Second,
		KillWait:    2 * time.Second,
	}, clk)
	logger := process.NewLogger(process.DefaultLogConfig(baseDir), clk)

	snapshot := map[string]config.IsolationProfile{}
	for _, p := range config.DefaultProfiles() {
		snapshot[p.ID] = p
	}

	h := &handlers.Handlers{
		Service:        svc,
		DriverSlot:     driver.NewSlot(drv),
		DB:             mocks.NewFakePinger(),
		Config:         config.Config{},
		Clock:          clk,
		Mounter:        fsmount.NewSystemMounter(starter),
		Starter:        starter,
		ProcessTracker: tracker,
		ProcessLogger:  logger,
	}
	h.SetProfileSnapshot(snapshot)

	// Fail loud if the builtin profile registry ever drops "full" — the
	// exec path resolves to it by default and would otherwise 409.
	if _, err := (&runtime.ProfileResolver{Profiles: snapshot}).Resolve(""); err != nil {
		t.Fatalf("default profile resolution broken: %v", err)
	}

	return &copyE2E{
		live:        httpx.NewLiveServer(t, h),
		tracker:     tracker,
		repo:        repo,
		projectRoot: projectRoot,
		scopePath:   projectRoot,
	}
}

// assertIdentityLayout checks the phase-3 negotiated workspace contract for
// the copy driver: agent-visible path == host merged dir, no path illusion,
// containment backend "none" with no enforcements.
func assertIdentityLayout(t *testing.T, where string, sb *types.Sandbox) {
	t.Helper()
	if sb.MergedDir == "" {
		t.Fatalf("%s: mergedDir empty", where)
	}
	if sb.WorkspacePath != sb.MergedDir {
		t.Errorf("%s: workspacePath = %q, want mergedDir %q", where, sb.WorkspacePath, sb.MergedDir)
	}
	if sb.PathIllusion {
		t.Errorf("%s: pathIllusion = true, want false", where)
	}
	if sb.Containment == nil {
		t.Fatalf("%s: containment missing", where)
	}
	if sb.Containment.Level != "none" {
		t.Errorf("%s: containment level = %q, want none", where, sb.Containment.Level)
	}
	if sb.Containment.Backend != "none" {
		t.Errorf("%s: containment backend = %q, want none", where, sb.Containment.Backend)
	}
	if len(sb.Containment.Enforcements) != 0 {
		t.Errorf("%s: enforcements = %v, want none", where, sb.Containment.Enforcements)
	}
}

// TestCopyDriverEndToEnd walks the full copy-driver arc over the live HTTP
// stack and asserts the identity-layout contract at every sandbox-returning
// hop, direct (backend "none") exec/process containment, a CopyStrategy diff
// including a lower-tree-walk deletion, and post-apply provenance.
func TestCopyDriverEndToEnd(t *testing.T) {
	e := newCopyE2E(t)
	ctx := context.Background()

	// --- Create: real CopyDriver.Mount copies the scope. ---
	createBody := `{"scopePath":"` + e.scopePath + `","projectRoot":"` + e.projectRoot + `","owner":"e2e","noLock":true}`
	resp, body := e.live.DoJSON(t, "POST", "/api/v1/sandboxes", createBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", resp.StatusCode, body)
	}
	var created types.Sandbox
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.DriverID != string(driver.DriverCopy) {
		t.Errorf("driverId = %q, want %q", created.DriverID, driver.DriverCopy)
	}
	if created.Status != types.StatusActive {
		t.Errorf("status = %q, want active", created.Status)
	}
	assertIdentityLayout(t, "create", &created)
	id := created.ID
	merged := created.MergedDir

	// --- Get: the same negotiated contract, re-derived server-side. ---
	resp, body = e.live.Do(t, "GET", "/api/v1/sandboxes/"+id.String(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var fetched types.Sandbox
	if err := json.Unmarshal(body, &fetched); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	assertIdentityLayout(t, "get", &fetched)

	// --- Exec: direct execution in MergedDir mutates the workspace. ---
	// Modify mod.txt, delete drop.txt, add added.txt — all in the merged view.
	execScript := "printf 'v2\\n' > mod.txt && rm drop.txt && printf 'new\\n' > added.txt"
	execBody := `{"command":"sh","args":["-c",` + jsonString(execScript) + `]}`
	resp, body = e.live.DoJSON(t, "POST", "/api/v1/sandboxes/"+id.String()+"/exec", execBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("exec status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var execResp handlers.ExecResponse
	if err := json.Unmarshal(body, &execResp); err != nil {
		t.Fatalf("decode exec: %v", err)
	}
	if execResp.ExitCode != 0 {
		t.Fatalf("exec exit = %d, stderr=%q", execResp.ExitCode, execResp.Stderr)
	}
	assertLaunchContainmentNone(t, "exec", execResp.Containment)

	// The mutation must have landed in the merged view (== workspace/upper),
	// not the canonical repo (approve applies there later).
	if got := readFile(t, filepath.Join(merged, "mod.txt")); got != "v2\n" {
		t.Errorf("workspace mod.txt = %q, want v2", got)
	}
	if _, err := os.Stat(filepath.Join(merged, "drop.txt")); !os.IsNotExist(err) {
		t.Errorf("workspace drop.txt still present (stat err=%v)", err)
	}

	// --- Process: background process, structured exit, captured logs. ---
	// The process prints to stdout, then blocks reading stdin. Blocking makes
	// the exit deterministic: the handler only returns 201 after the process
	// is tracked, so closing stdin afterwards guarantees the exit reaper finds
	// the tracked process and records ExitInfo (no instant-exit race).
	procScript := "printf 'proc-out\\n'; cat; exit 3"
	procBody := `{"command":"sh","args":["-c",` + jsonString(procScript) + `],"name":"probe","withStdin":true}`
	resp, body = e.live.DoJSON(t, "POST", "/api/v1/sandboxes/"+id.String()+"/processes", procBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("start process status = %d, want 201; body=%s", resp.StatusCode, body)
	}
	var procResp struct {
		PID         int                       `json:"pid"`
		Containment *types.SandboxContainment `json:"containment"`
	}
	if err := json.Unmarshal(body, &procResp); err != nil {
		t.Fatalf("decode process: %v", err)
	}
	if procResp.PID <= 0 {
		t.Fatalf("process pid = %d, want > 0", procResp.PID)
	}
	assertLaunchContainmentNone(t, "process", procResp.Containment)

	// Close stdin (EOF) so the blocked `cat` returns and the process exits 3.
	resp, body = e.live.Do(t, "POST",
		"/api/v1/sandboxes/"+id.String()+"/processes/"+strconv.Itoa(procResp.PID)+"/stdin?close=true", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stdin close status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	// Block once on the structured exit (never poll): the tracker closes the
	// exit channel when the reaper records ExitInfo.
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	exit, err := e.tracker.WaitForExit(waitCtx, id, procResp.PID)
	if err != nil {
		t.Fatalf("WaitForExit: %v", err)
	}
	if exit.ExitCode != 3 {
		t.Errorf("process exit code = %d, want 3", exit.ExitCode)
	}

	// Structured stdout log is retrievable via the logs endpoint.
	resp, body = e.live.Do(t, "GET",
		"/api/v1/sandboxes/"+id.String()+"/processes/"+strconv.Itoa(procResp.PID)+"/logs?stream=stdout", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logs status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var logResp struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &logResp); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if !strings.Contains(logResp.Content, "proc-out\n") {
		t.Errorf("stdout log = %q, want it to contain %q", logResp.Content, "proc-out\n")
	}
	// The log footer records the structured exit (code 3).
	if !strings.Contains(logResp.Content, "code 3") {
		t.Errorf("stdout log = %q, want it to record exit code 3", logResp.Content)
	}

	// --- Diff: CopyStrategy classifies add + modify + a lower-tree-walk delete. ---
	resp, body = e.live.Do(t, "GET", "/api/v1/sandboxes/"+id.String()+"/diff", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("diff status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var diffResult types.DiffResult
	if err := json.Unmarshal(body, &diffResult); err != nil {
		t.Fatalf("decode diff: %v", err)
	}
	if diffResult.Stats.FilesAdded != 1 {
		t.Errorf("filesAdded = %d, want 1", diffResult.Stats.FilesAdded)
	}
	if diffResult.Stats.FilesModified != 1 {
		t.Errorf("filesModified = %d, want 1", diffResult.Stats.FilesModified)
	}
	if diffResult.Stats.FilesDeleted != 1 {
		t.Errorf("filesDeleted = %d, want 1 (drop.txt via lower-tree walk)", diffResult.Stats.FilesDeleted)
	}
	byPath := map[string]types.ChangeType{}
	for _, f := range diffResult.Files {
		byPath[f.FilePath] = f.ChangeType
	}
	if byPath["added.txt"] != types.ChangeTypeAdded {
		t.Errorf("added.txt change = %q, want added", byPath["added.txt"])
	}
	if byPath["mod.txt"] != types.ChangeTypeModified {
		t.Errorf("mod.txt change = %q, want modified", byPath["mod.txt"])
	}
	if byPath["drop.txt"] != types.ChangeTypeDeleted {
		t.Errorf("drop.txt change = %q, want deleted", byPath["drop.txt"])
	}

	// --- Approve/apply: changes land in the canonical repo. ---
	approveBody := `{"mode":"all","actor":"e2e","agentManagerRunId":"run-e2e","createCommit":false}`
	resp, body = e.live.DoJSON(t, "POST", "/api/v1/sandboxes/"+id.String()+"/approve", approveBody)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var approval types.ApprovalResult
	if err := json.Unmarshal(body, &approval); err != nil {
		t.Fatalf("decode approve: %v", err)
	}
	if !approval.Success {
		t.Fatalf("approve success = false, msg=%q", approval.ErrorMsg)
	}
	if approval.Applied != 3 {
		t.Errorf("applied = %d, want 3", approval.Applied)
	}

	// Canonical repo now reflects the applied changes.
	if got := readFile(t, filepath.Join(e.projectRoot, "mod.txt")); got != "v2\n" {
		t.Errorf("canonical mod.txt = %q, want v2", got)
	}
	if got := readFile(t, filepath.Join(e.projectRoot, "added.txt")); got != "new\n" {
		t.Errorf("canonical added.txt = %q, want new", got)
	}
	if _, err := os.Stat(filepath.Join(e.projectRoot, "drop.txt")); !os.IsNotExist(err) {
		t.Errorf("canonical drop.txt still present after apply (stat err=%v)", err)
	}

	// --- Provenance: an AppliedChange row attributes the run. ---
	prov, err := e.repo.GetFileProvenance(ctx, filepath.Join(e.projectRoot, "added.txt"), e.projectRoot, 10)
	if err != nil {
		t.Fatalf("GetFileProvenance: %v", err)
	}
	if len(prov) == 0 {
		t.Fatal("no provenance recorded for applied file")
	}
	rec := prov[0]
	if rec.AgentManagerRunID != "run-e2e" {
		t.Errorf("provenance runID = %q, want run-e2e", rec.AgentManagerRunID)
	}
	if rec.SandboxID != id {
		t.Errorf("provenance sandboxID = %q, want %q", rec.SandboxID, id)
	}
	if rec.ChangeType != string(types.ChangeTypeAdded) {
		t.Errorf("provenance changeType = %q, want added", rec.ChangeType)
	}

	// Sandbox is terminal after full approval.
	resp, body = e.live.Do(t, "GET", "/api/v1/sandboxes/"+id.String(), nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post-approve get status = %d; body=%s", resp.StatusCode, body)
	}
	var final types.Sandbox
	if err := json.Unmarshal(body, &final); err != nil {
		t.Fatalf("decode final get: %v", err)
	}
	if final.Status != types.StatusApproved {
		t.Errorf("final status = %q, want approved", final.Status)
	}
}

// assertLaunchContainmentNone checks the per-launch containment stamped on
// an exec/process response: backend "none" (direct path), level "none", no
// enforcements — the honest report for the copy driver.
func assertLaunchContainmentNone(t *testing.T, where string, c *types.SandboxContainment) {
	t.Helper()
	if c == nil {
		t.Fatalf("%s: containment missing on launch response", where)
	}
	if c.Backend != "none" {
		t.Errorf("%s: launch backend = %q, want none", where, c.Backend)
	}
	if c.Level != "none" {
		t.Errorf("%s: launch level = %q, want none", where, c.Level)
	}
	if len(c.Enforcements) != 0 {
		t.Errorf("%s: launch enforcements = %v, want none", where, c.Enforcements)
	}
}

// jsonString returns the JSON-quoted form of s for inlining into a request
// body literal, so shell scripts with quotes/backslashes stay well-formed.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// readFile reads a file and fails the test on error.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
