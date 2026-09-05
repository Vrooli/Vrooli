package diagnostics_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	diagH "audio-tools/handlers/diagnostics"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	diagcore "audio-tools/internal/diagnostics"
	"audio-tools/internal/testutil/mocks"

	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	diagconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics/diagnostics_v1connect"
)

type sttExec struct {
	res *sttchain.Result
	err error
}

func (s *sttExec) Execute(_ context.Context, _ sttchain.Request) (*sttchain.Result, error) {
	return s.res, s.err
}

type ttsExec struct{ res *ttschain.Result }

func (s *ttsExec) Execute(_ context.Context, _ ttschain.Request) (*ttschain.Result, error) {
	return s.res, nil
}

type summExec struct{ res *summarizechain.Result }

func (s *summExec) Execute(_ context.Context, _ summarizechain.Request) (*summarizechain.Result, error) {
	return s.res, nil
}

type tcExec struct{ out []byte }

func (s *tcExec) Transcode(_ context.Context, _ []byte, _ string) ([]byte, error) {
	return s.out, nil
}

func newSuiteClient(t *testing.T, orch *diagcore.Orchestrator) diagconnect.DiagnosticsServiceClient {
	t.Helper()
	mod := diagH.Module(orch, &mocks.FakeLogger{})
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return diagconnect.NewDiagnosticsServiceClient(http.DefaultClient, srv.URL)
}

func passOrchestrator() *diagcore.Orchestrator {
	return diagcore.New(diagcore.Deps{
		STT:       &sttExec{res: &sttchain.Result{Text: "hi", Tier: sttchain.TierLocal, ProviderID: "whisper", Latency: 5 * time.Millisecond}},
		TTS:       &ttsExec{res: &ttschain.Result{Audio: []byte("x"), ContentType: "audio/wav", Tier: ttschain.TierLocal, ProviderID: "kokoro", Latency: 6 * time.Millisecond}},
		Summarize: &summExec{res: &summarizechain.Result{Text: "tldr", Tier: summarizechain.TierLocal, ProviderID: "ollama", Latency: 7 * time.Millisecond}},
		Transcode: &tcExec{out: []byte("RIFF...")},
		NewRunID:  func() string { return "run-id" },
	})
}

func TestRunSuite_HappyPath(t *testing.T) {
	c := newSuiteClient(t, passOrchestrator())
	resp, err := c.RunSuite(context.Background(), connect.NewRequest(&diagv1.RunSuiteRequest{}))
	require.NoError(t, err)
	run := resp.Msg.GetRun()
	require.Equal(t, "run-id", run.GetRunId())
	require.Equal(t, diagv1.SuiteOverall_STATUS_PASS, run.GetOverall().GetStatus())
	require.Len(t, run.GetSteps(), 4)
}

func TestRunSuite_CapabilityFilter(t *testing.T) {
	c := newSuiteClient(t, passOrchestrator())
	resp, err := c.RunSuite(context.Background(), connect.NewRequest(&diagv1.RunSuiteRequest{
		Capabilities: []diagv1.Capability{diagv1.Capability_CAPABILITY_SUMMARIZE},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetRun().GetSteps(), 1)
	require.Equal(t, diagv1.Capability_CAPABILITY_SUMMARIZE, resp.Msg.GetRun().GetSteps()[0].GetCapability())
}

func TestGetLastRun_NeverThenPopulated(t *testing.T) {
	orch := passOrchestrator()
	c := newSuiteClient(t, orch)
	resp, err := c.GetLastRun(context.Background(), connect.NewRequest(&diagv1.GetLastRunRequest{}))
	require.NoError(t, err)
	require.Equal(t, diagv1.SuiteOverall_STATUS_NEVER, resp.Msg.GetRun().GetOverall().GetStatus())

	_, err = c.RunSuite(context.Background(), connect.NewRequest(&diagv1.RunSuiteRequest{}))
	require.NoError(t, err)

	resp, err = c.GetLastRun(context.Background(), connect.NewRequest(&diagv1.GetLastRunRequest{}))
	require.NoError(t, err)
	require.Equal(t, "run-id", resp.Msg.GetRun().GetRunId())
}

func TestListFixtures_ReportsBundled(t *testing.T) {
	c := newSuiteClient(t, passOrchestrator())
	resp, err := c.ListFixtures(context.Background(), connect.NewRequest(&diagv1.ListFixturesRequest{}))
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, f := range resp.Msg.GetFixtures() {
		ids[f.GetId()] = true
		require.Greater(t, f.GetSizeBytes(), int64(0))
	}
	require.True(t, ids["smoke.wav"], "smoke.wav fixture missing")
	require.True(t, ids["smoke_text.txt"], "smoke_text.txt fixture missing")
}

func TestRunSuite_RejectsUnknownCapability(t *testing.T) {
	c := newSuiteClient(t, passOrchestrator())
	_, err := c.RunSuite(context.Background(), connect.NewRequest(&diagv1.RunSuiteRequest{
		Capabilities: []diagv1.Capability{diagv1.Capability_CAPABILITY_UNSPECIFIED},
	}))
	require.Error(t, err)
	var connErr *connect.Error
	require.True(t, errors.As(err, &connErr))
	require.Equal(t, connect.CodeInvalidArgument, connErr.Code())
}
