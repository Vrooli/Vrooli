package channel

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"vrooli-bridge/internal/audit"
	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/session"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

func sessionHTTPHandler(h *sessionWSHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.WithIdentity(r.Context(), auth.Identity{OwnerID: "owner-1"})
		h.handle(w, r.WithContext(ctx))
	})
}

func readSessionFrame(t *testing.T, c *websocket.Conn) *sessionv1.Frame {
	t.Helper()
	_, b, err := c.ReadMessage()
	require.NoError(t, err)
	var frame sessionv1.Frame
	require.NoError(t, (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, &frame))
	return &frame
}

func TestSessionWebSocketRelaysBytesAndAuditsLifecycle(t *testing.T) {
	sink := &auditmocks.FakeSink{}
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), audit: sink, auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes=" + session.Scope
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()
	open := readSessionFrame(t, conn).GetOpen()
	require.Equal(t, "s1", open.GetSessionId())
	b, err := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: 0, Data: []byte("hello")}}})
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, b))
	require.True(t, readSessionFrame(t, conn).GetAck().GetAccepted())
	require.Equal(t, []byte("hello"), readSessionFrame(t, conn).GetData().GetData())
	closeFrame, _ := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Reason: "done"}}})
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, closeFrame))
	records := sink.Appended()
	require.Len(t, records, 2)
	if len(records) == 2 {
		require.Equal(t, audit.ActionSessionDataIn, records[0].Action)
		require.Equal(t, "in:aGVsbG8=", records[0].Detail)
		require.Equal(t, audit.ActionSessionDataOut, records[1].Action)
		require.Equal(t, "out:aGVsbG8=", records[1].Detail)
	}
}

func TestSessionWebSocketRefusesNodeWithoutScopeBeforeUpgrade(t *testing.T) {
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes="
	_, resp, err := (&websocket.Dialer{}).Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}})
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSessionWebSocketRejectsCrossOrigin(t *testing.T) {
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes=" + session.Scope
	_, resp, err := (&websocket.Dialer{}).Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}, "Origin": []string{"https://attacker.invalid"}})
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}
