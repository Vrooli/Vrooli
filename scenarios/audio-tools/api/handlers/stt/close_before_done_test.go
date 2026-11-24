package stt

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/logx"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/session"
	"audio-tools/internal/sttengine"
	"github.com/vrooli/api-core/schedule"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

// Closing the request half after audio input is complete must not strand a
// batch provider that is still transcribing. The client cancellation must
// reach the provider and let the Connect handler unwind without waiting for
// an eventual Done event.
func TestTranscribeStream_CloseBeforeDoneCancelsBatchProvider(t *testing.T) {
	entered := make(chan struct{})
	providerCanceled := make(chan struct{})
	sessions := session.NewRegistry(0)
	executor := &sttmocks.FakeBatchExecutor{ExecuteFn: func(ctx context.Context, _ sttchain.Request) (*sttchain.Result, error) {
		close(entered)
		<-ctx.Done()
		close(providerCanceled)
		return nil, ctx.Err()
	}}
	deps := Deps{
		Chain:        sttchain.NewChain(sttchain.Options{}),
		Selector:     sttpkg.NewSelector(executor),
		Sessions:     sessions,
		StreamConfig: staticStreamConfig{raw: `{"streaming_mode":"off"}`},
		Logger:       logx.Std{},
		Clock:        schedule.System(),
	}
	connectPath, handler := sttconnect.NewSTTServiceHandler(NewConnectHandler(deps))
	handlerExited := make(chan struct{})
	mux := http.NewServeMux()
	mux.Handle(connectPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerExited)
		handler.ServeHTTP(w, r)
	}))
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	client := sttconnect.NewSTTServiceClient(srv.Client(), srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := client.TranscribeStream(ctx)
	require.NoError(t, sendStream(t, stream, startMsg("close-before-done", "close-before-done-token"), chunkMsg(0, []byte{0, 0})))
	require.NoError(t, stream.CloseRequest())

	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("batch provider did not enter its in-flight state")
	}
	cancel()

	select {
	case <-providerCanceled:
	case <-time.After(3 * time.Second):
		t.Fatal("client cancellation did not reach the in-flight batch provider")
	}

	select {
	case <-handlerExited:
	case <-time.After(3 * time.Second):
		t.Fatal("Connect handler did not unwind after provider cancellation")
	}
	ledger, resumed, err := sessions.Open("close-before-done", "close-before-done-token")
	require.NoError(t, err)
	require.True(t, resumed)
	require.Equal(t, session.TerminalNone, ledger.Snapshot().TerminalReason, "request half-close without StreamEnd must not complete or fail the turn")
}

// A pipeline failure before StreamEnd must terminate the response even when
// the client is still waiting for server events and has not half-closed.
func TestTranscribeStream_PipelineFailureBeforeInputEndDoesNotHang(t *testing.T) {
	sessions := session.NewRegistry(0)
	resolver := sttengine.NewLiveResolver(func(context.Context, ...string) ([]byte, error) {
		return nil, errors.New("inventory unavailable")
	}, time.Minute)
	deps := Deps{
		Chain:          sttchain.NewChain(sttchain.Options{}),
		Selector:       sttpkg.NewSelector(&sttmocks.FakeBatchExecutor{}),
		Registry:       sttengine.Default(),
		EngineResolver: resolver,
		Sessions:       sessions,
		StreamConfig:   staticStreamConfig{raw: `{"streaming_mode":"off"}`},
		Logger:         logx.Std{},
		Clock:          schedule.System(),
	}
	connectPath, handler := sttconnect.NewSTTServiceHandler(NewConnectHandler(deps))
	handlerExited := make(chan struct{})
	mux := http.NewServeMux()
	mux.Handle(connectPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(handlerExited)
		handler.ServeHTTP(w, r)
	}))
	srv := httptest.NewUnstartedServer(mux)
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)
	client := sttconnect.NewSTTServiceClient(srv.Client(), srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream := client.TranscribeStream(ctx)
	require.NoError(t, sendStream(t, stream, startMsg("pipeline-fails-early", "pipeline-fails-token")))

	events, err := receiveEvents(t, stream)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, "accepted", events[0].GetSessionStatus().GetState())
	var done *sttv1.StreamDone
	for _, event := range events {
		if event.GetDone() != nil {
			done = event.GetDone()
		}
	}
	require.NotNil(t, done, "the failed pipeline should deliver a terminal event without a client half-close")
	require.Equal(t, "pipeline_failed", done.GetTerminalReason())

	select {
	case <-handlerExited:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Connect handler remained blocked in Receive after the pipeline failed")
	}
	ledger, resumed, err := sessions.Open("pipeline-fails-early", "pipeline-fails-token")
	require.NoError(t, err)
	require.True(t, resumed)
	require.Equal(t, session.TerminalReason("pipeline_failed"), ledger.Snapshot().TerminalReason)
}
