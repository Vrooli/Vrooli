package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	sessionH "audio-tools/handlers/session"
	intsession "audio-tools/internal/session"

	sessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
	sessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session/session_v1connect"
)

func newServer(t *testing.T) (sessconnect.SessionServiceClient, *intsession.Registry) {
	t.Helper()
	reg := intsession.NewRegistry()
	mod := sessionH.Module(reg, nil)
	r := mux.NewRouter()
	mod.Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return sessconnect.NewSessionServiceClient(http.DefaultClient, srv.URL), reg
}

func TestSession_OpenSendTextClose(t *testing.T) {
	c, _ := newServer(t)
	ctx := context.Background()
	open, err := c.OpenSession(ctx, connect.NewRequest(&sessv1.OpenSessionRequest{Transport: "fake"}))
	require.NoError(t, err)
	sid := open.Msg.GetSessionId()
	require.NotEmpty(t, sid)

	_, err = c.SendText(ctx, connect.NewRequest(&sessv1.SendTextRequest{SessionId: sid, Text: "hello"}))
	require.NoError(t, err)

	_, err = c.CloseSession(ctx, connect.NewRequest(&sessv1.CloseSessionRequest{SessionId: sid}))
	require.NoError(t, err)
}

func TestSession_SendTextEmptyTextRejected(t *testing.T) {
	c, _ := newServer(t)
	ctx := context.Background()
	open, err := c.OpenSession(ctx, connect.NewRequest(&sessv1.OpenSessionRequest{}))
	require.NoError(t, err)
	_, err = c.SendText(ctx, connect.NewRequest(&sessv1.SendTextRequest{SessionId: open.Msg.GetSessionId(), Text: ""}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestSession_SendTextUnknownSession(t *testing.T) {
	c, _ := newServer(t)
	_, err := c.SendText(context.Background(), connect.NewRequest(&sessv1.SendTextRequest{SessionId: "nope", Text: "hi"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// TestSession_RegistryFanOut covers the multi-observer broadcast
// contract at the registry/session layer. The connect-go Subscribe
// server-stream RPC is exercised separately by the e2e suite; here we
// hold the same invariants (3 observers, each sees the broadcast) at
// the seam the handler delegates to. This avoids needing HTTP/2 in
// httptest.
//
// TODO: replace with end-to-end Subscribe stream coverage once the
// httpx.NewLiveServer harness gains an h2c option.
func TestSession_RegistryFanOut(t *testing.T) {
	_, reg := newServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	s := intsession.New(intsession.Options{Transport: "fake"})
	reg.Add(s)
	t.Cleanup(func() { s.Close("test"); reg.Remove(s.ID()) })

	const observers = 3
	type result struct{ count int }
	results := make([]result, observers)
	var wg sync.WaitGroup
	chans := make([]<-chan intsession.SessionEvent, observers)
	for i := 0; i < observers; i++ {
		_, ch, err := s.Subscribe(ctx, 8)
		require.NoError(t, err)
		chans[i] = ch
	}
	for i := 0; i < observers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-chans[i]:
					if !ok {
						return
					}
					results[i].count++
					if ev.AssistantFinal != nil {
						return
					}
				}
			}
		}()
	}
	// Give subscribers a moment to enter their loop, then fan out.
	time.Sleep(10 * time.Millisecond)
	s.EmitEvent(intsession.SessionEvent{Type: intsession.EventAssistantDelta, AssistantDelta: &intsession.AssistantDelta{Text: "x"}})
	s.EmitEvent(intsession.SessionEvent{Type: intsession.EventAssistantFinal, AssistantFinal: &intsession.AssistantFinal{Text: "x"}})
	wg.Wait()
	for i, r := range results {
		require.GreaterOrEqual(t, r.count, 1, "observer %d saw no events", i)
	}
}
