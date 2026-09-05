package journal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// mockExec implements CommandExecutor for tests.
type mockExec struct {
	Responses       map[string]mockResp
	DefaultResponse mockResp
	Calls           []mockCall
}

type mockResp struct {
	Output []byte
	Error  error
}

type mockCall struct {
	Name string
	Args []string
}

func newMockExec() *mockExec {
	return &mockExec{Responses: map[string]mockResp{}}
}

func (m *mockExec) CombinedOutput(_ context.Context, name string, args ...string) ([]byte, error) {
	m.Calls = append(m.Calls, mockCall{Name: name, Args: append([]string(nil), args...)})
	key := name
	for _, a := range args {
		key += " " + a
	}
	if r, ok := m.Responses[key]; ok {
		return r.Output, r.Error
	}
	return m.DefaultResponse.Output, m.DefaultResponse.Error
}

func newReaderWithMock() (*Reader, *mockExec) {
	m := newMockExec()
	return NewReader(m), m
}

func TestAvailableTrue(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte("systemd 254\n")}
	if !r.Available(context.Background()) {
		t.Fatal("Available = false, want true")
	}
}

func TestAvailableFalseOnError(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Error: errors.New("not found")}
	if r.Available(context.Background()) {
		t.Fatal("Available = true, want false")
	}
}

func TestAvailableNilReader(t *testing.T) {
	var r *Reader
	if r.Available(context.Background()) {
		t.Fatal("nil reader should not be available")
	}
}

func TestBuildArgs(t *testing.T) {
	args := buildArgs(QueryOpts{
		Unit:   []string{"docker"},
		Kernel: true, Since: "1h ago", Priority: "err", Grep: "panic",
		Tail: 50, Reverse: true, Boot: "-1", AfterCursor: "abc",
	}, true)
	want := []string{
		"--no-pager", "-o", "json",
		"-u", "docker",
		"-k",
		"-b", "-1",
		"--since", "1h ago",
		"-p", "err",
		"-g", "panic",
		"--after-cursor", "abc",
		"-n", "50",
		"-r",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("buildArgs:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildArgsTextMode(t *testing.T) {
	args := buildArgs(QueryOpts{Unit: []string{"docker"}}, false)
	want := []string{"--no-pager", "-o", "short-iso", "-u", "docker"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("text mode:\n got: %v\nwant: %v", args, want)
	}
}

func TestQueryLogsJSON(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte(strings.Join([]string{
		`{"__REALTIME_TIMESTAMP":"1714900000000000","__CURSOR":"s=1","_HOSTNAME":"vrooli","_SYSTEMD_UNIT":"docker.service","PRIORITY":"3","MESSAGE":"failed"}`,
		`{"__REALTIME_TIMESTAMP":"1714900001000000","_SYSTEMD_UNIT":"docker.service","PRIORITY":"6","MESSAGE":"started"}`,
	}, "\n"))}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"docker"}})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Cursor != "s=1" {
		t.Errorf("cursor not parsed: %+v", entries[0])
	}
	if entries[0].Unit != "docker.service" || entries[0].Priority != 3 {
		t.Errorf("entry[0]: %+v", entries[0])
	}
}

func TestQueryLogsBinaryMessage(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte(`{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":[104,105]}` + "\n")}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 1 || entries[0].Message != "hi" {
		t.Errorf("decoded: %+v", entries)
	}
}

func TestQueryLogsErrorIncludesArgv(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte("oops"), Error: errors.New("exit 1")}
	_, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"x"}})
	if err == nil || !strings.Contains(err.Error(), "-u x") {
		t.Errorf("error should include argv: %v", err)
	}
}

func TestQueryLogsTextFallback(t *testing.T) {
	r, m := newReaderWithMock()
	m.Responses["journalctl --no-pager -o json -u docker"] = mockResp{Output: []byte("not json\n")}
	m.Responses["journalctl --no-pager -o short-iso -u docker"] = mockResp{
		Output: []byte("2026-05-07T10:15:23+0000 vrooli dockerd[1234]: hello\n"),
	}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{Unit: []string{"docker"}})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 1 || entries[0].Message != "hello" || entries[0].PID != 1234 {
		t.Errorf("text fallback: %+v", entries)
	}
}

func TestQueryLogsMalformedKeptAsRaw(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte(strings.Join([]string{
		`{"__REALTIME_TIMESTAMP":"1714900000000000","MESSAGE":"good"}`,
		`{not valid`,
		`{"__REALTIME_TIMESTAMP":"1714900001000000","MESSAGE":"good2"}`,
	}, "\n"))}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("QueryLogs: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d, want 3", len(entries))
	}
	if entries[1].Raw == "" {
		t.Errorf("malformed line should be in Raw: %+v", entries[1])
	}
}

func TestQueryLogsEmpty(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte("")}
	entries, err := r.QueryLogs(context.Background(), QueryOpts{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %d, want 0", len(entries))
	}
}

func TestListBootsJSON(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Output: []byte(
		`[{"index":-1,"boot_id":"abc","first_entry":1714900000000000,"last_entry":1714903600000000},` +
			`{"index":0,"boot_id":"def","first_entry":1714903700000000,"last_entry":1714907300000000}]`,
	)}
	boots, err := r.ListBoots(context.Background())
	if err != nil {
		t.Fatalf("ListBoots: %v", err)
	}
	if len(boots) != 2 || boots[0].Index != -1 || boots[1].BootID != "def" {
		t.Errorf("boots: %+v", boots)
	}
}

func TestListBootsTextFallback(t *testing.T) {
	r, m := newReaderWithMock()
	m.Responses["journalctl --list-boots --no-pager -o json"] = mockResp{Output: []byte("not json")}
	m.Responses["journalctl --list-boots --no-pager"] = mockResp{Output: []byte(strings.Join([]string{
		"-1 aaaa Mon 2026-05-05 09:00:00 UTC—Mon 2026-05-05 11:00:00 UTC",
		" 0 bbbb Mon 2026-05-06 08:00:00 UTC—Mon 2026-05-06 09:00:00 UTC",
	}, "\n"))}
	boots, err := r.ListBoots(context.Background())
	if err != nil {
		t.Fatalf("ListBoots: %v", err)
	}
	if len(boots) != 2 || boots[0].Index != -1 || boots[1].BootID != "bbbb" {
		t.Errorf("boots: %+v", boots)
	}
}

func TestListBootsBothFail(t *testing.T) {
	r, m := newReaderWithMock()
	m.DefaultResponse = mockResp{Error: errors.New("denied")}
	if _, err := r.ListBoots(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestQueryLogsNilReader(t *testing.T) {
	var r *Reader
	if _, err := r.QueryLogs(context.Background(), QueryOpts{}); err == nil {
		t.Fatal("nil reader should error")
	}
}
