package journal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func newReaderWithMock(t *testing.T) (*Reader, *checks.MockExecutor) {
	t.Helper()
	mock := checks.NewMockExecutor()
	return NewReader(mock), mock
}

func TestAvailableTrueWhenJournalctlPresent(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte("systemd 254\n")}
	if !r.Available(context.Background()) {
		t.Fatal("Available = false, want true")
	}
}

func TestAvailableFalseWhenCommandFails(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Error: errors.New("exec: \"journalctl\": executable file not found in $PATH")}
	if r.Available(context.Background()) {
		t.Fatal("Available = true, want false on missing journalctl")
	}
}

func TestAvailableFalseWhenReaderNil(t *testing.T) {
	var r *Reader
	if r.Available(context.Background()) {
		t.Fatal("Available = true on nil Reader")
	}
}

func TestBuildArgsDeterministicOrder(t *testing.T) {
	args := buildArgs(QueryOpts{
		Unit:     []string{"docker", "rasdaemon"},
		Kernel:   true,
		Since:    "1 hour ago",
		Priority: "err",
		Grep:     "panic",
		Tail:     50,
		Reverse:  true,
		Boot:     "-1",
	}, true)

	want := []string{
		"--no-pager", "-o", "json",
		"-u", "docker", "-u", "rasdaemon",
		"-k",
		"-b", "-1",
		"--since", "1 hour ago",
		"-p", "err",
		"-g", "panic",
		"-n", "50",
		"-r",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("buildArgs argv mismatch:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildArgsTextMode(t *testing.T) {
	args := buildArgs(QueryOpts{Unit: []string{"docker"}, Tail: 100}, false)
	want := []string{"--no-pager", "-o", "short-iso", "-u", "docker", "-n", "100"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("text-mode buildArgs:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildArgsEmptyOpts(t *testing.T) {
	args := buildArgs(QueryOpts{}, true)
	want := []string{"--no-pager", "-o", "json"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("empty opts: got %v want %v", args, want)
	}
}

func TestBuildArgsEmitsCursorFlags(t *testing.T) {
	args := buildArgs(QueryOpts{
		Kernel:      true,
		Boot:        "0",
		Grep:        "panic",
		AfterCursor: "s=abc;i=1",
		ShowCursor:  true,
	}, true)

	want := []string{
		"--no-pager", "-o", "json",
		"-k",
		"-b", "0",
		"--after-cursor=s=abc;i=1",
		"--show-cursor",
		"-g", "panic",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("buildArgs with cursor flags:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildArgsOmitsCursorFlagsWhenUnset(t *testing.T) {
	args := buildArgs(QueryOpts{Kernel: true}, true)
	for _, a := range args {
		if strings.HasPrefix(a, "--after-cursor") || a == "--show-cursor" {
			t.Fatalf("cursor flags should be absent when unset: %v", args)
		}
	}
}

func TestQueryLogsParsesCursor(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte(strings.Join([]string{
		`{"__CURSOR":"s=aaa;i=1","__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"first"}`,
		`{"__CURSOR":"s=aaa;i=2","__REALTIME_TIMESTAMP":"1714900001000000","MESSAGE":"second"}`,
	}, "\n"))}

	entries, err := r.QueryLogs(context.Background(), QueryOpts{ShowCursor: true})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Cursor != "s=aaa;i=1" || entries[1].Cursor != "s=aaa;i=2" {
		t.Errorf("cursors not parsed: %q, %q", entries[0].Cursor, entries[1].Cursor)
	}
}

func TestQueryLogsParsesJSON(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte(strings.Join([]string{
		`{"__REALTIME_TIMESTAMP":"1714900000000000","_HOSTNAME":"vrooli","_SYSTEMD_UNIT":"docker.service","SYSLOG_IDENTIFIER":"dockerd","_PID":"1234","PRIORITY":"3","_BOOT_ID":"abc","MESSAGE":"failed to pull"}`,
		`{"__REALTIME_TIMESTAMP":"1714900001000000","_HOSTNAME":"vrooli","_SYSTEMD_UNIT":"docker.service","PRIORITY":"6","MESSAGE":"started"}`,
		``,
	}, "\n"))}

	entries, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"docker"}, Tail: 2})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Unit != "docker.service" || entries[0].PID != 1234 || entries[0].Priority != 3 {
		t.Errorf("entry[0]: %+v", entries[0])
	}
	if entries[0].Message != "failed to pull" {
		t.Errorf("message: %q", entries[0].Message)
	}
	if entries[0].Timestamp.IsZero() {
		t.Error("timestamp not parsed")
	}
}

func TestQueryLogsBinaryMessageDecodes(t *testing.T) {
	r, mock := newReaderWithMock(t)
	// "hi" as a byte array — journald's encoding for binary log messages.
	mock.DefaultResponse = checks.MockResponse{Output: []byte(
		`{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":[104,105]}` + "\n",
	)}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 1 || entries[0].Message != "hi" {
		t.Errorf("decoded message: %+v", entries)
	}
}

func TestQueryLogsErrorPropagated(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte("oops"), Error: errors.New("exit status 1")}
	_, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"x"}})
	if err == nil {
		t.Fatal("expected error")
	}
	// Argv must be in the error so debugging is non-mysterious.
	if !strings.Contains(err.Error(), "-u x") {
		t.Errorf("error should include argv: %v", err)
	}
}

func TestQueryLogsEmptyExitOneMeansNoMatches(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Error: errors.New("exit status 1")}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{Kernel: true, Grep: "pm_runtime_work .* hogged CPU"})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestQueryLogsFallsBackToTextOnJSONFailure(t *testing.T) {
	r, mock := newReaderWithMock(t)
	jsonKey := "journalctl --no-pager -o json -u docker"
	textKey := "journalctl --no-pager -o short-iso -u docker"
	mock.Responses = map[string]checks.MockResponse{
		jsonKey: {Output: []byte("not json at all\nstill not json\n")},
		textKey: {Output: []byte("2026-05-07T10:15:23+0000 vrooli dockerd[1234]: hello world\n")},
	}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"docker"}})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Message != "hello world" || entries[0].Identifier != "dockerd" || entries[0].PID != 1234 {
		t.Errorf("text fallback parse: %+v", entries[0])
	}
}

func TestQueryLogsMalformedLineKeptAsRaw(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte(strings.Join([]string{
		`{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"good"}`,
		`{not valid json`,
		`{"__REALTIME_TIMESTAMP":"1714900001000000","MESSAGE":"also good"}`,
	}, "\n"))}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3 (incl raw)", len(entries))
	}
	if entries[1].Raw == "" {
		t.Errorf("malformed line should be preserved in Raw: %+v", entries[1])
	}
}

func TestQueryLogsEmptyOutput(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte("")}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("empty output: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestTailPassesArgvUnchanged(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte("raw bytes")}
	out, err := r.Tail(context.Background(), QueryOpts{Unit: []string{"cloudflared"}, Tail: 100})
	if err != nil {
		t.Fatalf("Tail: %v", err)
	}
	if string(out) != "raw bytes" {
		t.Errorf("Tail output: %q", out)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(mock.Calls))
	}
	want := []string{"--no-pager", "-o", "short-iso", "-u", "cloudflared", "-n", "100"}
	if !reflect.DeepEqual(mock.Calls[0].Args, want) {
		t.Errorf("argv:\n got: %v\nwant: %v", mock.Calls[0].Args, want)
	}
}

func TestListBootsParsesJSON(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Output: []byte(
		`[{"index":-1,"boot_id":"abc","first_entry":1714900000000000,"last_entry":1714903600000000},` +
			`{"index":0,"boot_id":"def","first_entry":1714903700000000,"last_entry":1714907300000000}]`,
	)}
	boots, err := r.ListBoots(context.Background())
	if err != nil {
		t.Fatalf("ListBoots: %v", err)
	}
	if len(boots) != 2 {
		t.Fatalf("got %d boots, want 2", len(boots))
	}
	if boots[0].Index != -1 || boots[0].BootID != "abc" {
		t.Errorf("boot[0]: %+v", boots[0])
	}
	if boots[0].FirstEntry.IsZero() || boots[0].LastEntry.IsZero() {
		t.Errorf("timestamps not parsed: %+v", boots[0])
	}
}

func TestListBootsFallsBackToText(t *testing.T) {
	r, mock := newReaderWithMock(t)
	jsonKey := "journalctl --list-boots --no-pager -o json"
	textKey := "journalctl --list-boots --no-pager"
	mock.Responses = map[string]checks.MockResponse{
		jsonKey: {Output: []byte("the table is not json")},
		textKey: {Output: []byte(strings.Join([]string{
			"-2 1111aaaa Mon 2026-05-05 09:00:00 UTC—Mon 2026-05-05 11:00:00 UTC",
			"-1 2222bbbb Mon 2026-05-05 11:30:00 UTC—Mon 2026-05-05 22:00:00 UTC",
			" 0 3333cccc Mon 2026-05-06 08:00:00 UTC—Mon 2026-05-06 09:00:00 UTC",
		}, "\n"))},
	}
	boots, err := r.ListBoots(context.Background())
	if err != nil {
		t.Fatalf("ListBoots: %v", err)
	}
	if len(boots) != 3 {
		t.Fatalf("got %d, want 3", len(boots))
	}
	if boots[0].Index != -2 || boots[2].Index != 0 {
		t.Errorf("indexes: %+v", boots)
	}
	if boots[1].BootID != "2222bbbb" {
		t.Errorf("bootid: %+v", boots[1])
	}
}

func TestListBootsErrorWhenBothFormatsFail(t *testing.T) {
	r, mock := newReaderWithMock(t)
	mock.DefaultResponse = checks.MockResponse{Error: errors.New("permission denied")}
	if _, err := r.ListBoots(context.Background()); err == nil {
		t.Fatal("expected error when both formats fail")
	}
}

func TestRenderText(t *testing.T) {
	entries := []LogEntry{
		{Timestamp: time.Date(2026, 5, 7, 10, 15, 23, 0, time.UTC), Hostname: "vrooli", Unit: "docker.service", Message: "started"},
		{Raw: "raw line as-is"},
		{Timestamp: time.Date(2026, 5, 7, 10, 15, 24, 0, time.UTC), Hostname: "vrooli", Identifier: "kernel", Message: "oops"},
	}
	got := RenderText(entries)
	if !strings.Contains(got, "2026-05-07T10:15:23Z vrooli docker.service: started") {
		t.Errorf("missing structured line:\n%s", got)
	}
	if !strings.Contains(got, "raw line as-is") {
		t.Errorf("raw line missing: %s", got)
	}
	if !strings.Contains(got, "kernel: oops") {
		t.Errorf("identifier fallback missing: %s", got)
	}
}

func TestQueryLogsNilReader(t *testing.T) {
	var r *Reader
	if _, err := r.QueryLogs(context.Background(), QueryOpts{}); err == nil {
		t.Fatal("nil reader should error")
	}
}
