package log

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"
	logconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log/log_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
)

// logRecorder is a fake LogService that captures the evidence slice each
// handler built. Only the *-add verbs are implemented; the rest satisfy the
// interface and are never called by these tests.
type logRecorder struct {
	logconnect.UnimplementedLogServiceHandler
	mu       sync.Mutex
	evidence []string
}

func (r *logRecorder) record(ev []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evidence = append([]string(nil), ev...)
}

func (r *logRecorder) captured() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.evidence...)
}

func ok() *connect.Response[logv1.AddEntryResponse] {
	return connect.NewResponse(&logv1.AddEntryResponse{})
}

func (r *logRecorder) AddDecision(_ context.Context, req *connect.Request[logv1.AddDecisionRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	r.record(req.Msg.GetEvidence())
	return ok(), nil
}

func (r *logRecorder) AddFinding(_ context.Context, req *connect.Request[logv1.AddFindingRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	r.record(req.Msg.GetEvidence())
	return ok(), nil
}

func (r *logRecorder) AddNote(_ context.Context, req *connect.Request[logv1.AddNoteRequest]) (*connect.Response[logv1.AddEntryResponse], error) {
	r.record(req.Msg.GetEvidence())
	return ok(), nil
}

func newLogFixture(t *testing.T, rec *logRecorder) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := logconnect.NewLogServiceHandler(rec)
	mux.Handle(path, handler)
	app := clitest.NewTestApp(t, mux)
	group, err := Register(app, clitest.ReadManifest(t))
	require.NoError(t, err, "register log group against manifest")
	return app, []cliapp.SubcommandGroup{group}
}

// TestEvidenceFlagsPreserveCommasViaRepeatableFlag is the reason the repeatable
// variant exists. --evidence splits on every comma with no quoting, so a
// locator containing one was silently torn into two entries; --evidence-item
// keeps it whole. Both flags must still compose.
func TestEvidenceFlagsPreserveCommasViaRepeatableFlag(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		argv []string
		want []string
	}{
		{
			name: "csv flag splits on commas as before",
			cmd:  "decision-add",
			argv: []string{"plan-1", "--title", "t", "--evidence", "run:a,run:b"},
			want: []string{"run:a", "run:b"},
		},
		{
			name: "repeatable flag preserves a comma inside one locator",
			cmd:  "decision-add",
			argv: []string{"plan-1", "--title", "t", "--evidence-item", "Makefile:122,139"},
			want: []string{"Makefile:122,139"},
		},
		{
			name: "repeatable flag accumulates",
			cmd:  "finding-add",
			argv: []string{"plan-1", "--title", "t", "--evidence-item", "run:a", "--evidence-item", "run:b"},
			want: []string{"run:a", "run:b"},
		},
		{
			name: "both flags compose, repeatable first",
			cmd:  "note-add",
			argv: []string{"plan-1", "--title", "t", "--evidence-item", "x,y", "--evidence", "run:c"},
			want: []string{"x,y", "run:c"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &logRecorder{}
			app, groups := newLogFixture(t, rec)
			_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "log", tc.cmd), app, tc.argv...)
			require.NoError(t, err)
			require.Equal(t, tc.want, rec.captured())
		})
	}
}

// TestEvidenceFlagsDropEmptyEntries keeps the trailing-comma and whitespace
// behaviour the CSV path always had.
func TestEvidenceFlagsDropEmptyEntries(t *testing.T) {
	rec := &logRecorder{}
	app, groups := newLogFixture(t, rec)
	_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "log", "decision-add"), app,
		"plan-1", "--title", "t", "--evidence", " run:a , , run:b ")
	require.NoError(t, err)
	require.Equal(t, []string{"run:a", "run:b"}, rec.captured())
}
