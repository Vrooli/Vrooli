package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clitest "agent-manager/cli/internal/testutil"

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
	home := copyBASGoalHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		s = strings.Replace(s, "3. **RD-EXP", "3. [ready] **RD-EXP", 1)
		s = strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices; SUP-AUDIT-003 is tracked under Waiting on others.\n- The census notes", 1)
		return strings.Replace(s, "## Waiting on others", "### Orchestrator handoff — 2026-10-06T16:43Z\n\nNothing changed; parking for 12 hours.\n\n## Waiting on others", 1)
	}})
	output, code := runEffort(t, &App{}, "lint", home)
	if code != exitRefused || !strings.Contains(output, "FAIL handoff-count QUEUE.md") || !strings.Contains(output, "### Orchestrator handoff — 2026-10-06T16:43Z") || !strings.Contains(output, "goal home fails 1 rules: handoff-count") {
		t.Fatalf("lint must fail and name the duplicate handoff (exit %d):\n%s", code, output)
	}
	home = copyBASGoalHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		s = strings.Replace(s, "3. **RD-EXP", "3. [ready] **RD-EXP", 1)
		return strings.Replace(s, "- The census notes", "- FB-063 is translated into the RD slices; SUP-AUDIT-003 is tracked under Waiting on others.\n- The census notes", 1)
	}})
	if output, code = runEffort(t, &App{}, "lint", home, "--json"); code != 0 || !strings.Contains(output, `"blocking": false`) {
		t.Fatalf("a goal home that meets the rules passes (exit %d):\n%s", code, output)
	}
}

func TestEffortHandoffSetReplacesOnceAndRefusesOversizedText(t *testing.T) {
	home := copyBASGoalHome(t, nil)
	queue := filepath.Join(home, "QUEUE.md")
	text := writeFile(t, filepath.Join(t.TempDir(), "handoff.md"), "- 2026-10-07T15:00Z: E26 accepted; admit JX-2 as E27 next.\n")
	output, code := runEffort(t, &App{}, "handoff", "set", home, "--file", text)
	if code != 0 || !strings.Contains(output, "Handoff replaced") {
		t.Fatalf("handoff set failed (exit %d): %s", code, output)
	}
	written, _ := os.ReadFile(queue)
	if strings.Count(string(written), "## Handoff") != 1 || !strings.Contains(string(written), "## Handoff\n\n- 2026-10-07T15:00Z: E26 accepted") || strings.Contains(string(written), "act on BAS-FB-064") {
		t.Fatalf("QUEUE.md does not hold exactly the new handoff:\n%s", written)
	}
	if output, code = runEffort(t, &App{}, "handoff", "set", home, "--file", text); code != 0 || !strings.Contains(output, "Handoff unchanged") {
		t.Fatalf("an unchanged handoff must write nothing (exit %d): %s", code, output)
	}
	large := writeFile(t, filepath.Join(t.TempDir(), "large.md"), strings.Repeat("census prose ", 400))
	if output, code = runEffort(t, &App{}, "handoff", "set", home, "--file", large); code != exitRefused || !strings.Contains(output, "REFUSED handoff-size") {
		t.Fatalf("an oversized handoff must be refused (exit %d): %s", code, output)
	}
	if after, _ := os.ReadFile(queue); string(after) != string(written) {
		t.Fatal("a refused handoff must leave QUEUE.md untouched")
	}
}

// parkServer answers the children list with the given runs and records the park.
func parkServer(t *testing.T, children string) (*App, *clitest.RecordingServer) {
	t.Helper()
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		if request.Path == "/api/v1/runs" {
			return `{"runs":[` + children + `]}`
		}
		return `{"message":"parked"}`
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions {
		return cliutil.APIBaseOptions{DefaultBase: server.URL()}
	}, nil)
	t.Setenv(cliutil.EnvIdentityToken, "orchestrator-token-fixture")
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

func TestEffortParkRefusesWithoutChildOrDecision(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	app, server := parkServer(t, `{"id":"c33912b5-1b51-49b3-8ddb-aad93600b001","status":"RUN_STATUS_COMPLETE"}`)
	output, code := runEffort(t, app, "park", copyBASGoalHome(t, nil), "--run", run, "--timeout", "12h")
	if code != exitRefused || !strings.Contains(output, "PARK_REFUSED admissible-slice") || !strings.Contains(output, "Admissible-slice rule") {
		t.Fatalf("park must be refused with the admissible-slice rule (exit %d): %s", code, output)
	}
	if parkRequests(server) != 0 {
		t.Fatalf("a refused park must not reach the server: %+v", server.Requests())
	}
	decision := copyBASGoalHome(t, map[string]func(string) string{"QUEUE.md": func(s string) string {
		return strings.Replace(s, "(none; the operator answered BAS-FB-063 on 2026-10-06)", "- Retire Record mode? Options: keep, retire. Recommendation: keep.", 1)
	}})
	if output, code = runEffort(t, app, "park", decision, "--run", run); code != 0 || !strings.Contains(output, "for 72h0m0s (needs-operator)") || parkRequests(server) != 1 {
		t.Fatalf("a Needs operator item allows a 72h park (exit %d): %s", code, output)
	}
}

func TestEffortParkLiveChildUsesChildrenBackstop(t *testing.T) {
	const run = "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"
	app, server := parkServer(t, `{"id":"e8c322ca-72e7-434c-9d37-1d2cf853961f","status":"RUN_STATUS_RUNNING"}`)
	home := copyBASGoalHome(t, nil)
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
