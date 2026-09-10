package preflight

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/reachtest"
)

func postJSON(t *testing.T, handler http.HandlerFunc, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload)))
	return rec
}

// [REQ:STC-P0-028] Disk cleanup runs only owner operations: every action is
// a privilege-broker action through `cloud-target host repair`, with
// per-action results, and the free-space delta comes from df observations.
func TestHandleDiskCleanupRunsOwnerActionsAndReportsPerAction(t *testing.T) {
	dfCalls := 0
	host := &reachtest.Scripted{Strict: true, Prefixes: map[string]reachtest.Answer{
		"vrooli cloud-target host repair --action journald.vacuum --subject b64:":            {Result: reach.Result{ExitCode: 0, Stdout: `{"ok":true}`}},
		"vrooli cloud-target host repair --action docker.prune.unused-images --subject b64:": {Result: reach.Result{ExitCode: 2, Stdout: `{"error":{"code":"broker_unavailable","message":"docker is not installed"}}`}},
	}}
	reachFor := func(target identity.TargetRef) reach.Reach {
		if target.Locator.Host != "203.0.113.10" || target.Transport != identity.TransportSSH {
			t.Fatalf("unexpected target %+v", target)
		}
		return &dfCounting{Scripted: host, calls: &dfCalls}
	}
	rec := postJSON(t, HandleDiskCleanup(reachFor), "/api/v1/preflight/disk/cleanup", map[string]any{"host": "203.0.113.10", "actions": []string{"journal_vacuum", "docker_prune"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp DiskCleanupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK || len(resp.ActionsRun) != 1 || resp.ActionsRun[0] != "journal_vacuum" || len(resp.ActionsFailed) != 1 || resp.ActionsFailed[0] != "docker_prune" {
		t.Fatalf("unexpected response %+v", resp)
	}
	if resp.SpaceFreedKB != 200 {
		t.Fatalf("space freed = %d, want 200 (df before 1000, after 1200)", resp.SpaceFreedKB)
	}
	for _, r := range resp.ActionResults {
		if r.Action == "docker_prune" && (r.OK || r.ExitCode != 2 || !strings.Contains(r.Summary, "broker_unavailable") || r.Hint == "") {
			t.Fatalf("docker_prune result = %+v", r)
		}
	}
	for _, call := range host.Calls {
		if call.IsObservation() {
			continue
		}
		if call.Verb != "cloud-target host repair" || !call.Effectful {
			t.Fatalf("cleanup must run through host repair only, got %+v", call)
		}
	}
}

// dfCounting answers df with a growing free space so the handler's delta is
// observable, and delegates everything else to the scripted host.
type dfCounting struct {
	*reachtest.Scripted
	calls *int
}

func (d *dfCounting) Exec(ctx context.Context, target identity.TargetRef, cmd reach.Command) (reach.Result, error) {
	if cmd.Program == "df" {
		*d.calls++
		free := "1000"
		if *d.calls > 1 {
			free = "1200"
		}
		return reach.Result{Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 5000 4000 " + free + " 80% /"}, nil
	}
	return d.Scripted.Exec(ctx, target, cmd)
}

// [REQ:STC-P0-028] An action without a privilege-broker owner is refused
// before anything runs, naming the missing owner.
func TestHandleDiskCleanupRefusesUnownedActionsTyped(t *testing.T) {
	host := &reachtest.Scripted{Strict: true}
	for _, action := range []string{"apt_clean", "tmp_clean", "rm_everything"} {
		rec := postJSON(t, HandleDiskCleanup(func(identity.TargetRef) reach.Reach { return host }), "/api/v1/preflight/disk/cleanup", map[string]any{"host": "203.0.113.10", "actions": []string{"journal_vacuum", action}})
		if rec.Code != http.StatusNotImplemented && rec.Code != http.StatusUnprocessableEntity && rec.Code != http.StatusBadRequest && rec.Code != http.StatusConflict {
			t.Fatalf("%s: status=%d body=%s", action, rec.Code, rec.Body.String())
		}
		var body struct {
			Error struct {
				Code       string `json:"code"`
				NextAction struct {
					Owner string `json:"owner"`
				} `json:"next_action"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code != "unsupported_capability" || body.Error.NextAction.Owner != "internal/privilegebroker" {
			t.Fatalf("%s: body=%s err=%v", action, rec.Body.String(), err)
		}
	}
	if len(host.Calls) != 0 {
		t.Fatalf("a refused request must not touch the target: %v", host.CallKeys())
	}
}

// Disk usage is df plus du over a fixed root set; no caller path reaches the
// target and the largest entries are ranked.
func TestHandleDiskUsageObservesFixedRoots(t *testing.T) {
	host := &reachtest.Scripted{Answers: map[string]reachtest.Answer{
		"df -Pk /":                    {Result: reach.Result{Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/vda1 40000000 30000000 10000000 75% /"}},
		"ls -A1 -- /var":              {Result: reach.Result{Stdout: "log\nlib"}},
		"ls -A1 -- /tmp":              {Result: reach.Result{Stdout: "evil; rm -rf /\nsafe"}},
		"du -sk -- /var/log /var/lib": {Result: reach.Result{Stdout: "500 /var/log\n9000 /var/lib"}},
		"du -sk -- /tmp/safe":         {Result: reach.Result{Stdout: "7 /tmp/safe"}},
	}}
	rec := postJSON(t, HandleDiskUsage(func(identity.TargetRef) reach.Reach { return host }), "/api/v1/preflight/disk/usage", map[string]any{"host": "203.0.113.10"})
	var resp DiskUsageResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || !resp.OK {
		t.Fatalf("body=%s err=%v", rec.Body.String(), err)
	}
	if resp.FreeBytes != 10000000*1024 || resp.UsedPercent != 75 {
		t.Fatalf("usage = %+v", resp)
	}
	if len(resp.LargestDirs) != 3 || resp.LargestDirs[0].Path != "/var/lib" {
		t.Fatalf("largest dirs = %+v", resp.LargestDirs)
	}
	for _, key := range host.CallKeys() {
		if strings.Contains(key, "evil") {
			t.Fatalf("an unsafe directory name reached the target: %s", key)
		}
	}
}
