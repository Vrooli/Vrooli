package vroolicli

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestOrphansDecodesProcesses(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"orphans":[{"pid":4242,"ppid":1,"command":"vrooli scenario start x"}]}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.Orphans(context.Background())
	if err != nil {
		t.Fatalf("Orphans returned error: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetOrphans()) != 1 {
		t.Fatalf("unexpected orphans payload: %+v", resp)
	}
	if p := resp.GetOrphans()[0]; p.GetPid() != 4242 || p.GetPpid() != 1 || p.GetCommand() == "" {
		t.Fatalf("orphan process not decoded: %+v", p)
	}
	if want := []string{"--no-stale-check", "orphans", "--json"}; !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("argv = %v, want %v", runner.calls[0].args, want)
	}
}

func TestDiagnosePortBuildsArgvAndDecodes(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"diagnostic":{"port":16542,"scenario":"ecosystem-manager","in_use":true,"host_orphan_count":0}}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.DiagnosePort(context.Background(), 16542, "ecosystem-manager")
	if err != nil {
		t.Fatalf("DiagnosePort returned error: %v", err)
	}
	diag := resp.GetDiagnostic()
	if diag.GetPort() != 16542 || diag.GetScenario() != "ecosystem-manager" || !diag.GetInUse() {
		t.Fatalf("diagnostic not decoded: %+v", diag)
	}
	if want := []string{"--no-stale-check", "diagnose-port", "16542", "ecosystem-manager", "--json"}; !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("argv = %v, want %v", runner.calls[0].args, want)
	}
}

func TestDiagnosePortOmitsEmptyScenario(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{{output: []byte(`{"success":true,"diagnostic":{"port":8080}}`)}},
	}
	client := New(WithRunner(runner))

	if _, err := client.DiagnosePort(context.Background(), 8080, "  "); err != nil {
		t.Fatalf("DiagnosePort returned error: %v", err)
	}
	if want := []string{"--no-stale-check", "diagnose-port", "8080", "--json"}; !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("argv = %v, want %v (blank scenario should be omitted)", runner.calls[0].args, want)
	}
}

func TestDiagnosePortRejectsBadPort(t *testing.T) {
	client := New(WithRunner(&stubRunner{}))
	if _, err := client.DiagnosePort(context.Background(), 0, ""); err == nil {
		t.Fatal("expected out-of-range port to error before exec")
	}
}

func TestCleanupOrphansAndLocksDecodeStopReport(t *testing.T) {
	cases := []struct {
		name    string
		call    func(*Client) (interface{ GetSuccess() bool }, error)
		wantArg string
	}{
		{"orphans", func(c *Client) (interface{ GetSuccess() bool }, error) { return c.CleanupOrphans(context.Background()) }, "orphans"},
		{"locks", func(c *Client) (interface{ GetSuccess() bool }, error) { return c.CleanupLocks(context.Background()) }, "locks"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := &stubRunner{
				responses: []stubResponse{
					{output: []byte(`{"success":true,"data":{"stopped":[{"name":"a","message":"ok"}],"failed":[],"message":"done"}}`)},
				},
			}
			client := New(WithRunner(runner))

			resp, err := tc.call(client)
			if err != nil {
				t.Fatalf("cleanup %s returned error: %v", tc.name, err)
			}
			if !resp.GetSuccess() {
				t.Fatalf("cleanup %s: success=false", tc.name)
			}
			if want := []string{"--no-stale-check", "cleanup", tc.wantArg, "--json"}; !reflect.DeepEqual(runner.calls[0].args, want) {
				t.Fatalf("argv = %v, want %v", runner.calls[0].args, want)
			}
		})
	}
}

func TestCleanupPropagatesError(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{err: errors.New("boom")}}}
	client := New(WithRunner(runner))
	if _, err := client.CleanupLocks(context.Background()); err == nil {
		t.Fatal("expected error to propagate, never an empty success")
	}
}
