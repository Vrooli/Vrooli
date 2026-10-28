package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "agent-manager/cli/internal/testutil"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/cli-core/cliutil"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestEffortCLIUsesSharedBoardAndExplicitMutationRequests(t *testing.T) {
	services, server := newContractServices(t)
	app := &App{services: services}
	if err := app.cmdEffort([]string{"board", "--effort-ref", "effort:any", "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := app.cmdEffort([]string{"enroll"}); err == nil {
		t.Fatal("mutation accepted without a request")
	}
	path := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(path, []byte(`{"enrollment":{"effort_ref":"effort:any","work_shape":"investigation"},"idempotency_key":"one"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.cmdEffort([]string{"enroll", "--request-file", path, "--json"}); err != nil {
		t.Fatal(err)
	}
	requests := server.Requests()
	if len(requests) != 2 || requests[0].Path != "/agent_manager.v1.AgentManagerService/GetEffortBoard" || requests[1].Path != "/agent_manager.v1.AgentManagerService/EnrollEffort" {
		t.Fatal("CLI bypassed canonical RPC projection", requests)
	}
	// A partially populated wire row remains a useful human response, not a panic.
	if err := app.cmdEffort([]string{"board"}); err != nil {
		t.Fatal(err)
	}
}

func TestEffortCLICompactBoardPrintsJoinedOwnerFacts(t *testing.T) {
	board := &pb.EffortBoard{ActiveCount: 1, QuotaObservations: []*pb.EffortQuotaObservation{{Provider: "openai", Pool: "primary", Window: "daily", Standing: "reported", EvidenceRef: "quota:1"}}, Rows: []*pb.EffortBoardRow{{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:arbitrary", DisplayName: "Arbitrary effort", TargetRevision: "accepted"}, RuntimeState: "active", Freshness: pb.EffortFreshness_EFFORT_FRESHNESS_FRESH, OutcomeStanding: &pb.EffortOutcomeStanding{State: "unknown"}, EvidenceRefs: []string{"checkpoint:changed"}, PendingOperations: []string{"owner:wait"}, Usage: &pb.EffortUsage{Partial: true}}}}
	output := captureStdout(t, func() error { printCompactEffortBoard(board); return nil })
	for _, want := range []string{"checkpoint:changed", "Named waits: owner:wait", "agent-manager:GetEffortBoard:effort:arbitrary", "Usage: tokens=unknown", "Quota observations (shared owner cut):", "openai/primary/daily", "quota:1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("compact owner projection omitted %q: %s", want, output)
		}
	}
}

func TestEffortLocalOwnerDoesNotElevateAgentRequests(t *testing.T) {
	services, recorder := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "request.json")
	if err := os.WriteFile(file, []byte(`{"authority":"WATCH_AUTHORITY_FAMILY_PARENT"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "identified-agent-fixture")
	if err := app.cmdEffort([]string{"reconcile-metadata", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "identified agent") {
		t.Fatalf("agent must not invoke owner exchange: %v", err)
	}
	t.Setenv(cliutil.EnvIdentityToken, "")
	if err := app.cmdEffort([]string{"reconcile-metadata", "--request-file", file, "--local-owner"}); err == nil || !strings.Contains(err.Error(), "operator authority") {
		t.Fatalf("agent authority must not be upgraded: %v", err)
	}
	if err := app.cmdEffort([]string{"board", "--local-owner"}); err == nil {
		t.Fatal("read must not exchange unnecessary credentials")
	}
	if len(recorder.Requests()) != 0 {
		t.Fatal("rejected override made an API request")
	}
}

// copyBASGoalHome copies the trimmed real BAS goal home into a temporary
// directory and applies edits to named files.
func copyBASGoalHome(t *testing.T, edits map[string]func(string) string) string {
	t.Helper()
	source, home := filepath.Join(goalhomeTestdata, "bas"), t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name, _ := filepath.Rel(source, path)
		data, err := os.ReadFile(path)
		text := string(data)
		if edit := edits[name]; edit != nil {
			text = edit(text)
		}
		writeFile(t, filepath.Join(home, name), text)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func copyRuntimeBASGoalHome(t *testing.T, edits map[string]func(string) string) string {
	t.Helper()
	home := copyBASGoalHome(t, edits)
	v1, err := goalhome.LoadLegacy(os.DirFS(home))
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte { b, _ := os.ReadFile(filepath.Join(home, name)); return b }
	conv := goalhome.ConvertHome(v1, read(goalhome.GoalFile), read(goalhome.QueueFile), read(goalhome.WorkaroundsFile))
	if !conv.Equivalent {
		t.Fatal(conv.Divergence)
	}
	if err := applyMigration(home, conv); err != nil {
		t.Fatal(err)
	}
	return home
}

// copyV2BASGoalHome copies the Phase-2 v2 BAS home (CHARTER/STATUS/LOG/FEEDBACK)
// into a temp dir, so the CLI writers sniff it as v2 and target STATUS.md.
func copyV2BASGoalHome(t *testing.T, edits map[string]func(string) string) string {
	t.Helper()
	source, home := filepath.Join(goalhomeTestdata, "v2-bas"), t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name, _ := filepath.Rel(source, path)
		data, err := os.ReadFile(path)
		text := string(data)
		if edit := edits[name]; edit != nil {
			text = edit(text)
		}
		writeFile(t, filepath.Join(home, name), text)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return home
}

func runEffort(t *testing.T, app *App, args ...string) (string, int) {
	t.Helper()
	code := 0
	output := captureStdout(t, func() error {
		code = exitCode(app.cmdEffort(args))
		return nil
	})
	return output, code
}

func TestEffortLintNamesDuplicateHandoffAndExitsFour(t *testing.T) {
	home := copyRuntimeBASGoalHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		s = strings.Replace(s, "3. **RD-EXP", "3. [ready] **RD-EXP", 1)
		s = strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices; SUP-AUDIT-003 is tracked under Waiting on others.\n- The census notes", 1)
		return strings.Replace(s, "## Waiting on others", "### Orchestrator handoff — 2026-10-06T16:43Z\n\nNothing changed; parking for 12 hours.\n\n## Waiting on others", 1)
	}})
	output, code := runEffort(t, &App{}, "lint", home)
	if code != exitRefused || !strings.Contains(output, "FAIL handoff-count STATUS.md") || !strings.Contains(output, "### Orchestrator handoff — 2026-10-06T16:43Z") || !strings.Contains(output, "goal home fails 1 rules: handoff-count") {
		t.Fatalf("lint must fail and name the duplicate handoff (exit %d):\n%s", code, output)
	}
	home = copyRuntimeBASGoalHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		s = strings.Replace(s, "3. **RD-EXP", "3. [ready] **RD-EXP", 1)
		return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices; SUP-AUDIT-003 is tracked under Waiting on others.\n- The census notes", 1)
	}})
	if output, code = runEffort(t, &App{}, "lint", home, "--json"); code != 0 || !strings.Contains(output, `"blocking": false`) {
		t.Fatalf("a goal home that meets the rules passes (exit %d):\n%s", code, output)
	}
}

func TestEffortHandoffSetReplacesOnceAndRefusesOversizedText(t *testing.T) {
	home := copyRuntimeBASGoalHome(t, nil)
	queue := filepath.Join(home, "STATUS.md")
	text := writeFile(t, filepath.Join(t.TempDir(), "handoff.md"), "- 2026-10-07T15:00Z: E26 accepted; admit JX-2 as E27 next.\n")
	output, code := runEffort(t, &App{}, "handoff", "set", home, "--file", text)
	if code != 0 || !strings.Contains(output, "Handoff replaced") {
		t.Fatalf("handoff set failed (exit %d): %s", code, output)
	}
	written, _ := os.ReadFile(queue)
	if strings.Count(string(written), "## Handoff") != 1 || !strings.Contains(string(written), "## Handoff\n\n- 2026-10-07T15:00Z: E26 accepted") || strings.Contains(string(written), "act on BAS-FB-064") {
		t.Fatalf("STATUS.md does not hold exactly the new handoff:\n%s", written)
	}
	if output, code = runEffort(t, &App{}, "handoff", "set", home, "--file", text); code != 0 || !strings.Contains(output, "Handoff unchanged") {
		t.Fatalf("an unchanged handoff must write nothing (exit %d): %s", code, output)
	}
	large := writeFile(t, filepath.Join(t.TempDir(), "large.md"), strings.Repeat("census prose ", 400))
	if output, code = runEffort(t, &App{}, "handoff", "set", home, "--file", large); code != exitRefused || !strings.Contains(output, "REFUSED handoff-size") {
		t.Fatalf("an oversized handoff must be refused (exit %d): %s", code, output)
	}
	if after, _ := os.ReadFile(queue); string(after) != string(written) {
		t.Fatal("a refused handoff must leave STATUS.md untouched")
	}
}

// TestEffortHandoffSetTargetsStatusForV2Home proves the handoff writer targets
// STATUS.md (not STATUS.md) when the home is a Phase-2 v2 home.
func TestEffortHandoffSetTargetsStatusForV2Home(t *testing.T) {
	home := copyV2BASGoalHome(t, nil)
	status := filepath.Join(home, goalhome.StatusFile)
	text := writeFile(t, filepath.Join(t.TempDir(), "handoff.md"), "- 2026-10-08T09:00Z: migrated; admit the next journey slice.\n")
	output, code := runEffort(t, &App{}, "handoff", "set", home, "--file", text)
	if code != 0 || !strings.Contains(output, "Handoff replaced") || !strings.Contains(output, goalhome.StatusFile) {
		t.Fatalf("handoff set must replace the STATUS.md handoff (exit %d): %s", code, output)
	}
	written, _ := os.ReadFile(status)
	if strings.Count(string(written), "## Handoff") != 1 || !strings.Contains(string(written), "- 2026-10-08T09:00Z: migrated") || strings.Contains(string(written), "act on BAS-FB-064") {
		t.Fatalf("STATUS.md must hold exactly the new handoff:\n%s", written)
	}
	if _, err := os.Stat(filepath.Join(home, goalhome.QueueFile)); !os.IsNotExist(err) {
		t.Fatal("the v2 handoff writer must not create a QUEUE.md")
	}
}

// TestEffortLintV2HomeUsesStatusAnchors proves every lint anchor points at the
// v2 file set (STATUS.md, not GOAL.md/QUEUE.md) and that the absent v1 files are
// never flagged missing. The unmodified v2-bas home already fails idle-queue,
// feedback-unacknowledged and diminishing-returns, so a refusal is expected.
func TestEffortLintV2HomeUsesStatusAnchors(t *testing.T) {
	home := copyV2BASGoalHome(t, nil)
	output, code := runEffort(t, &App{}, "lint", home)
	if code != exitRefused {
		t.Fatalf("the v2-bas home must fail lint on its open items (exit %d):\n%s", code, output)
	}
	// idle-queue anchors ## Next/## Needs operator, now in STATUS.md.
	if !strings.Contains(output, "FAIL idle-queue "+goalhome.StatusFile) {
		t.Fatalf("a queue-section finding must name STATUS.md for a v2 home:\n%s", output)
	}
	// No finding may name the absent v1 files, nor flag them missing.
	if strings.Contains(output, "missing-file") || strings.Contains(output, goalhome.QueueFile) || strings.Contains(output, goalhome.GoalFile) {
		t.Fatalf("no lint finding may reference the absent v1 GOAL.md/QUEUE.md for a v2 home:\n%s", output)
	}
}

func TestEffortMigrationResumesPublicationAndArchivesOnce(t *testing.T) {
	home := copyBASGoalHome(t, nil)
	v1, err := goalhome.LoadLegacy(os.DirFS(home))
	if err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	conv := goalhome.ConvertHome(v1, read(goalhome.GoalFile), read(goalhome.QueueFile), read(goalhome.WorkaroundsFile))
	if !conv.Equivalent {
		t.Fatal(conv.Divergence)
	}
	// An archive collision interrupts publication after the new files exist.
	writeFile(t, filepath.Join(home, "archive/v1/GOAL.md"), "unrelated retained data")
	if err := applyMigration(home, conv); err == nil {
		t.Fatal("archive collision must preserve existing data")
	}
	// Retry refuses to clobber an operator edit to a partially published file.
	published := read(goalhome.StatusFile)
	writeFile(t, filepath.Join(home, goalhome.StatusFile), "operator edit")
	if err := finishMigration(home); err == nil {
		t.Fatal("retry overwrote an operator edit")
	}
	if string(read(goalhome.StatusFile)) != "operator edit" {
		t.Fatal("operator edit was changed")
	}
	writeFile(t, filepath.Join(home, goalhome.StatusFile), string(published))
	if err := os.Remove(filepath.Join(home, "archive/v1/GOAL.md")); err != nil {
		t.Fatal(err)
	}
	app := &App{}
	if _, code := runEffort(t, app, "migrate", home, "--apply", "--json"); code != 0 {
		t.Fatalf("recover interrupted migration: exit %d", code)
	}
	before := read(goalhome.LogFile)
	if _, code := runEffort(t, app, "migrate", home, "--apply", "--json"); code != 0 {
		t.Fatalf("repeat migration: exit %d", code)
	}
	if string(read(goalhome.LogFile)) != string(before) {
		t.Fatal("repeat changed the accepted ledger")
	}
	if _, err := os.Stat(filepath.Join(home, goalhome.GoalFile)); !os.IsNotExist(err) {
		t.Fatal("recovery did not finish archiving GOAL")
	}
}

func TestEffortMigrationRejectsIncompleteV2AndPreservesEdits(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, goalhome.CharterFile), "# Partial publication\n")
	if _, code := runEffort(t, &App{}, "migrate", home, "--apply", "--json"); code == 0 {
		t.Fatal("a lone charter is not a completed migration")
	}
}

// parkServer answers the children list with the given runs and records the park.
const parkTestRunID = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"

func parkServer(t *testing.T, children string) (*App, *clitest.RecordingServer) {
	return parkServerWithRun(t, children, func(runID string) string {
		return `{"run":{"id":"` + runID + `","status":"RUN_STATUS_RUNNING"}}`
	})
}

func parkServerWithRun(t *testing.T, children string, runResponse func(runID string) string) (*App, *clitest.RecordingServer) {
	return parkServerWithIdentity(t, children, `{"valid":true,"claims":{"run_id":"`+parkTestRunID+`"}}`, runResponse)
}

func parkServerWithIdentity(t *testing.T, children, identityResponse string, runResponse func(runID string) string) (*App, *clitest.RecordingServer) {
	t.Helper()
	clitest.IsolateCredentialBroker(t)
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Method == "POST" && request.Path == "/api/v1/identity/verify" {
			return identityResponse
		}
		const runPrefix = "/api/v1/runs/"
		if request.Method == "GET" && strings.HasPrefix(request.Path, runPrefix) {
			return runResponse(strings.TrimPrefix(request.Path, runPrefix))
		}
		if request.Path == "/api/v1/runs" {
			return `{"runs":[` + children + `]}`
		}
		if request.Method == "POST" && strings.HasSuffix(request.Path, "/park") {
			runID := strings.TrimSuffix(strings.TrimPrefix(request.Path, runPrefix), "/park")
			return `{"success":true,"message":"parked","run":{"id":"` + runID + `","status":"RUN_STATUS_PARKED","await_handle":{"producer":"children","key":"` + runID + `"}}}`
		}
		return `{}`
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	t.Setenv(cliutil.EnvIdentityToken, "orchestrator-token-fixture")
	t.Setenv("AGENT_MANAGER_API_BASE", server.URL())
	return &App{services: NewServices(api)}, server
}

func parkRequests(server *clitest.RecordingServer) int {
	parks := 0
	for _, request := range server.Requests() {
		if request.Method == "POST" && strings.HasSuffix(request.Path, "/park") {
			parks++
		}
	}
	return parks
}

func TestEffortParkValidatesExactTargetBeforeChildPolicy(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	for _, tc := range []struct {
		name        string
		runResponse func(string) string
		wantError   string
	}{
		{
			name:        "missing run",
			runResponse: func(string) string { return `{"run":null}` },
			wantError:   "returned no run",
		},
		{
			name:        "mismatched run identity",
			runResponse: func(string) string { return `{"run":{"id":"another-run"}}` },
			wantError:   "returned mismatched run ID",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, server := parkServerWithRun(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`, tc.runResponse)
			err := app.effortPark([]string{copyAcknowledgedParkHome(t, nil), "--run", run})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("invalid park target should fail before policy evaluation, got %v", err)
			}
			requests := server.Requests()
			if len(requests) != 2 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/identity/verify" || requests[1].Method != "GET" || requests[1].Path != "/api/v1/runs/"+run {
				t.Fatalf("invalid target must stop after exact Get, got requests: %+v", requests)
			}
			if parkRequests(server) != 0 {
				t.Fatalf("invalid target must not be parked: %+v", requests)
			}
		})
	}
}

func TestEffortParkDefaultsToVerifiedCallerAndPreservesJSONReceipt(t *testing.T) {
	app, server := parkServer(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`)
	output, code := runEffort(t, app, "park", copyAcknowledgedParkHome(t, nil), "--json")
	var receipt struct {
		Park struct {
			Success bool `json:"success"`
			Run     struct {
				ID          string `json:"id"`
				Status      string `json:"status"`
				AwaitHandle struct {
					Producer string `json:"producer"`
					Key      string `json:"key"`
				} `json:"await_handle"`
			} `json:"run"`
		} `json:"park"`
	}
	if code != 0 || json.Unmarshal([]byte(output), &receipt) != nil || !receipt.Park.Success ||
		receipt.Park.Run.ID != parkTestRunID || receipt.Park.Run.Status != "RUN_STATUS_PARKED" ||
		receipt.Park.Run.AwaitHandle.Producer != "children" || receipt.Park.Run.AwaitHandle.Key != parkTestRunID {
		t.Fatalf("self-defaulted JSON park should preserve the owner receipt (exit %d): %s", code, output)
	}
	var targetRead, childRead, parkWrite bool
	for _, request := range server.Requests() {
		switch {
		case request.Method == "GET" && request.Path == "/api/v1/runs/"+parkTestRunID:
			targetRead = true
		case request.Method == "GET" && request.Path == "/api/v1/runs" && strings.Contains(request.Query, "parentRunId="+parkTestRunID):
			childRead = true
		case request.Method == "POST" && request.Path == "/api/v1/runs/"+parkTestRunID+"/park":
			parkWrite = true
		}
	}
	if !targetRead || !childRead || !parkWrite {
		t.Fatalf("verified caller ID must flow through target, child, and park calls: %+v", server.Requests())
	}
}

func TestEffortParkRefusesForeignExplicitRunBeforeTargetReads(t *testing.T) {
	app, server := parkServer(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`)
	err := app.effortPark([]string{copyAcknowledgedParkHome(t, nil), "--run", "another-run"})
	if err == nil || !strings.Contains(err.Error(), "does not match verified caller run") {
		t.Fatalf("foreign explicit run should be refused against verified identity, got %v", err)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/identity/verify" {
		t.Fatalf("run mismatch must refuse before target/child reads, got %+v", requests)
	}
}

func TestEffortParkUsesConfiguredAPIBaseForVerificationAndRunCalls(t *testing.T) {
	app, target := parkServer(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`)
	ambient := clitest.NewRecordingServerForRequests(t, func(clitest.Request) string { return `{}` })
	t.Setenv("AGENT_MANAGER_API_BASE", ambient.URL())
	targetAPI := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: target.URL(), EnvVars: []string{"AGENT_MANAGER_API_BASE"}}
	}, nil)
	app.services = NewServices(targetAPI)

	output, code := runEffort(t, app, "park", copyAcknowledgedParkHome(t, nil), "--json")
	if code != 0 || !strings.Contains(output, `"RUN_STATUS_PARKED"`) {
		t.Fatalf("configured target should complete the park flow (exit %d): %s", code, output)
	}
	if requests := ambient.Requests(); len(requests) != 0 {
		t.Fatalf("ambient Agent Manager must receive no verification or run requests: %+v", requests)
	}
	requests := target.Requests()
	if len(requests) != 4 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/identity/verify" ||
		requests[1].Method != "GET" || requests[1].Path != "/api/v1/runs/"+parkTestRunID ||
		requests[2].Method != "GET" || requests[2].Path != "/api/v1/runs" ||
		requests[3].Method != "POST" || requests[3].Path != "/api/v1/runs/"+parkTestRunID+"/park" {
		t.Fatalf("verification and park operations must share the configured target: %+v", requests)
	}
}

func TestEffortParkInvalidIdentityUsesConfiguredAPIBaseBeforeRunReads(t *testing.T) {
	app, target := parkServerWithIdentity(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`, `{"valid":false,"error":"target rejected token"}`, func(runID string) string {
		return `{"run":{"id":"` + runID + `","status":"RUN_STATUS_RUNNING"}}`
	})
	ambient := clitest.NewRecordingServerForRequests(t, func(clitest.Request) string { return `{"valid":true,"claims":{"run_id":"` + parkTestRunID + `"}}` })
	t.Setenv("AGENT_MANAGER_API_BASE", ambient.URL())
	targetAPI := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{Override: target.URL(), EnvVars: []string{"AGENT_MANAGER_API_BASE"}}
	}, nil)
	app.services = NewServices(targetAPI)

	err := app.effortPark([]string{copyAcknowledgedParkHome(t, nil)})
	if err == nil || !strings.Contains(err.Error(), "target rejected token") {
		t.Fatalf("configured target identity refusal should be returned, got %v", err)
	}
	if requests := target.Requests(); len(requests) != 1 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/identity/verify" {
		t.Fatalf("invalid configured identity must stop before target run reads: %+v", requests)
	}
	if requests := ambient.Requests(); len(requests) != 0 {
		t.Fatalf("ambient Agent Manager must receive no identity or run requests: %+v", requests)
	}
}

func TestEffortParkRefusesInvalidIdentityBeforeTargetReads(t *testing.T) {
	app, server := parkServerWithIdentity(t, `{"id":"child","status":"RUN_STATUS_RUNNING"}`, `{"valid":false,"error":"token revoked"}`, func(runID string) string {
		return `{"run":{"id":"` + runID + `","status":"RUN_STATUS_RUNNING"}}`
	})
	err := app.effortPark([]string{copyAcknowledgedParkHome(t, nil)})
	if err == nil || !strings.Contains(err.Error(), "token revoked") {
		t.Fatalf("invalid identity should be refused explicitly, got %v", err)
	}
	requests := server.Requests()
	if len(requests) != 1 || requests[0].Method != "POST" || requests[0].Path != "/api/v1/identity/verify" {
		t.Fatalf("invalid identity must refuse before target/child reads, got %+v", requests)
	}
}

func TestEffortParkRefusesWithoutChildOrDecision(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	app, server := parkServer(t, `{"id":"c33912b5-1b51-49b3-8ddb-aad93600b001","status":"RUN_STATUS_COMPLETE"}`)
	output, code := runEffort(t, app, "park", copyAcknowledgedParkHome(t, nil), "--run", run, "--timeout", "12h")
	if code != exitRefused || !strings.Contains(output, "PARK_REFUSED admissible-slice") || !strings.Contains(output, "Admissible-slice rule") {
		t.Fatalf("park must be refused with the admissible-slice rule (exit %d): %s", code, output)
	}
	if parkRequests(server) != 0 {
		t.Fatalf("a refused park must not reach the server: %+v", server.Requests())
	}
	readyAndDecision := copyAcknowledgedParkHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		s = strings.Replace(s, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- [decision X-1] Wait for an owner? Options: wait, continue.\n", 1)
		return strings.Replace(s, "1. **RD-WF", "1. [ready] **RD-WF", 1)
	}})
	if output, code = runEffort(t, app, "park", readyAndDecision, "--run", run); code != exitRefused || !strings.Contains(output, "PARK_REFUSED admissible-slice") || !strings.Contains(output, "ready] slice(s) remain") {
		t.Fatalf("independent ready work must take precedence over a Needs operator wait (exit %d): %s", code, output)
	}
	if parkRequests(server) != 0 {
		t.Fatalf("a refused park with ready work must not reach the server: %+v", server.Requests())
	}
	decision := copyAcknowledgedParkHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		return strings.Replace(s, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- Retire Record mode? Options: keep, retire. Recommendation: keep.", 1)
	}})
	if output, code = runEffort(t, app, "park", decision, "--run", run); code != 0 || !strings.Contains(output, "for 72h0m0s (needs-operator)") || parkRequests(server) != 1 {
		t.Fatalf("a Needs operator item allows a 72h park (exit %d): %s", code, output)
	}
}

func TestEffortParkLiveChildUsesChildrenBackstop(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	app, server := parkServer(t, `{"id":"e8c322ca-72e7-434c-9d37-1d2cf853961f","status":"RUN_STATUS_RUNNING"}`)
	home := copyAcknowledgedParkHome(t, nil)
	if output, code := runEffort(t, app, "park", home, "--run", run, "--timeout", "12h"); code != exitRefused || !strings.Contains(output, "exceeds the 1h0m0s maximum (live-child)") || parkRequests(server) != 0 {
		t.Fatalf("a 12h park with a live child must be refused (exit %d): %s", code, output)
	}
	output, code := runEffort(t, app, "park", home, "--run", run)
	if code != 0 || !strings.Contains(output, "for 1h0m0s (live-child)") || !strings.Contains(output, "parked") || parkRequests(server) != 1 {
		t.Fatalf("a live child allows the 1h backstop park (exit %d): %s", code, output)
	}
	for _, request := range server.Requests() {
		if request.Path == "/api/v1/runs" && !strings.Contains(request.Query, "parentRunId="+run) {
			t.Fatalf("children must be listed for the parked run: %+v", request)
		}
	}
}

func TestEffortParkExcludesReviewChildrenButKeepsExecutionChildren(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	home := copyAcknowledgedParkHome(t, nil)

	reviewOnly, reviewServer := parkServer(t, `{"id":"e8c322ca-72e7-434c-9d37-1d2cf853961f","status":"RUN_STATUS_NEEDS_REVIEW"}`)
	output, code := runEffort(t, reviewOnly, "park", home, "--run", run, "--timeout", "12h")
	if code != exitRefused || !strings.Contains(output, "admissible-slice") || strings.Contains(output, "live-child") || parkRequests(reviewServer) != 0 {
		t.Fatalf("a review-only child must not trigger the live-child backstop (exit %d): %s", code, output)
	}

	for _, tc := range []struct {
		name   string
		status string
	}{
		{name: "pending", status: "RUN_STATUS_PENDING"},
		{name: "starting", status: "RUN_STATUS_STARTING"},
		{name: "running", status: "RUN_STATUS_RUNNING"},
		{name: "parked", status: "RUN_STATUS_PARKED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, server := parkServer(t, `{"id":"a69af561-e4c7-4852-b53a-19f87a880041","status":"`+tc.status+`"}`)
			output, code := runEffort(t, app, "park", home, "--run", run, "--timeout", "12h")
			if code != exitRefused || !strings.Contains(output, "exceeds the 1h0m0s maximum (live-child)") || parkRequests(server) != 0 {
				t.Fatalf("%s child must retain the live-child backstop (exit %d): %s", tc.name, code, output)
			}
		})
	}
}

func TestEffortBoardPrintsGoalHomeFactsAndUnknowns(t *testing.T) {
	target, weighted := 24.0, 450000.0
	board := &pb.EffortBoard{Rows: []*pb.EffortBoardRow{{Enrollment: &pb.EffortEnrollment{EffortRef: "effort:bas"}, GoalHome: &pb.EffortGoalHome{
		Path: "scenarios/browser-automation-studio/docs/internal/goal", State: "available", DestinationMetric: "journeys", DestinationOp: ">=", DestinationTarget: &target,
		ActiveUnit: "E34", ActiveStatus: "in-progress", NeedsOperator: []string{"[decision BAS-D-070] Close or re-aim?"}, CanaryStatus: "unknown",
		CapUsage:      []*pb.EffortCapUsage{{RunId: "run-1", Role: "worker", WeightedTokens: &weighted, Mode: "report"}},
		OpenDecisions: []*pb.EffortDecision{{DecisionId: "BAS-D-070", GoalHome: "g", State: "pending", Recommended: "re-aim", Question: "Close or re-aim?"}},
	}}}}
	for _, print := range []func(*pb.EffortBoard){printEffortBoard, printCompactEffortBoard} {
		output := captureStdout(t, func() error { print(board); return nil })
		for _, want := range []string{"Destination: journeys >= 24; current: unknown", "Active unit: E34 in-progress", "workarounds=unknown", "spend per accepted unit=unknown", "Needs operator: [decision BAS-D-070]", "Decision: BAS-D-070  g  pending recommended=re-aim unpublished", "Cap: worker run=run-1 weighted=450000 cap=none mode=report"} {
			if !strings.Contains(output, want) {
				t.Fatalf("board omitted %q:\n%s", want, output)
			}
		}
	}
}

// Park-policy fixtures have already consumed prior steering. New-feedback tests
// add a fresh entry explicitly; unrelated child/identity cases retain their focus.
func copyAcknowledgedParkHome(t *testing.T, edits map[string]func(string) string) string {
	t.Helper()
	home := copyRuntimeBASGoalHome(t, edits)
	writeFile(t, filepath.Join(home, goalhome.FeedbackFile), "# Feedback\n\n## Resolved\n\nPrior fixture steering was resolved before this park.\n")
	return home
}

func TestEffortParkDoesNotIgnoreFreshFeedbackWhileChildRuns(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	app, server := parkServer(t, `{"id":"e8c322ca-72e7-434c-9d37-1d2cf853961f","status":"RUN_STATUS_RUNNING"}`)
	home := copyAcknowledgedParkHome(t, nil)
	writeFile(t, filepath.Join(home, goalhome.FeedbackFile), "# Feedback\n\n### NEW-REPAIR — supervisor correction\n\nAct on the corrected outcome.\n\nStatus: open\n")
	output, code := runEffort(t, app, "park", home, "--run", run)
	if code != exitRefused || !strings.Contains(output, "feedback-unacknowledged") || !strings.Contains(output, "NEW-REPAIR") || parkRequests(server) != 0 {
		t.Fatalf("new steering was ignored before park (exit %d): %s", code, output)
	}
}
