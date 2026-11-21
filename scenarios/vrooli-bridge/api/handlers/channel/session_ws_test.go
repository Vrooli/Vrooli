package channel

import (
	_ "embed"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"vrooli-bridge/internal/audit"
	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/registry"
	registrymocks "vrooli-bridge/internal/registry/mocks"
	"vrooli-bridge/internal/session"

	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

// Keep the fixture-shape check independent of the directory from which `go test`
// invokes this package. The test binary may be built with -trimpath, which makes
// runtime.Caller return the module-relative source name instead of a filesystem path.
//
//go:embed session_ws_test.go
var sessionWSTestSource string

func sessionRegistry(scopes ...string) registry.Service {
	return &registrymocks.FakeService{GetOut: registry.Node{ID: "n1", Scopes: scopes}}
}

func sessionHTTPHandler(h *sessionWSHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.WithIdentity(r.Context(), auth.Identity{OwnerID: "owner-1", Scopes: []string{session.TransportScope}})
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
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), audit: sink, auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}, registry: sessionRegistry(session.TransportScope)}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes=" + session.TransportScope
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
		require.Equal(t, "in:sha256=2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824:bytes=5", records[0].Detail)
		require.Equal(t, audit.ActionSessionDataOut, records[1].Action)
		require.Equal(t, "out:sha256=2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824:bytes=5", records[1].Detail)
	}
}

func TestSessionWebSocketRejectsTerminalFrameOnDesktopSession(t *testing.T) { // REM-02
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), audit: &auditmocks.FakeSink{}, auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}, registry: sessionRegistry(session.TransportScope)}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s-protocol&scopes=" + session.TransportScope
	conn, resp, err := (&websocket.Dialer{}).Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()
	require.Equal(t, "s-protocol", readSessionFrame(t, conn).GetOpen().GetSessionId())
	openFrame, err := proto.Marshal(&sessionv1.Frame{Payload: &sessionv1.Frame_Open{Open: &sessionv1.Open{SessionId: "spoof"}}})
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, openFrame))
	reject := readSessionFrame(t, conn).GetAck()
	require.Equal(t, "unsupported_frame", reject.GetCode())
	require.Contains(t, reject.GetReason(), "server-owned")
}

func TestSessionWebSocketAcceptsEnrolledLocalSessionWithoutSecondJWT(t *testing.T) {
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), audit: &auditmocks.FakeSink{}, registry: sessionRegistry(session.TransportScope)}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := auth.WithIdentity(r.Context(), auth.Identity{OwnerID: "owner-1", AuthMethod: auth.AuthMethodEnrolled, Scopes: []string{session.TransportScope}})
		h.handle(w, r.WithContext(ctx))
	}))
	defer srv.Close()

	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=enrolled-s1&scopes=" + session.TransportScope
	conn, resp, err := (&websocket.Dialer{}).Dial(url, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	defer conn.Close()
	require.Equal(t, "enrolled-s1", readSessionFrame(t, conn).GetOpen().GetSessionId())
}

func TestSessionWebSocketRefusesNodeWithoutScopeBeforeUpgrade(t *testing.T) {
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}, registry: sessionRegistry()}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes="
	_, resp, err := (&websocket.Dialer{}).Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}})
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSessionWebSocketRejectsCrossOrigin(t *testing.T) {
	h := &sessionWSHandler{manager: session.NewManager(nil, nil), auth: &auth.FakeValidator{Identity: auth.Identity{OwnerID: "owner-1"}}, registry: sessionRegistry(session.TransportScope)}
	srv := httptest.NewServer(sessionHTTPHandler(h))
	defer srv.Close()
	url := "ws" + srv.URL[4:] + "/api/v1/channel/session?node=n1&session_id=s1&scopes=" + session.TransportScope
	_, resp, err := (&websocket.Dialer{}).Dial(url, http.Header{"X-Bridge-Owner-Reauth": []string{"fresh-proof"}, "Origin": []string{"https://attacker.invalid"}})
	require.Error(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSessionBindingQueryRequiresLeaseAndPolicyIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/channel/session?node=n1&surface_id=desktop&transport=websocket&lease_id=lease-1&lease_epoch=7&policy_revision=policy-1", nil)
	binding, err := sessionBindingFromQuery(req, "n1", "owner-1")
	require.NoError(t, err)
	require.Equal(t, "desktop", binding.Surface.SurfaceID)
	require.Equal(t, "lease-1", binding.LeaseID)
	require.Equal(t, uint64(7), binding.LeaseEpoch)
	require.Equal(t, "owner-1", binding.OwnerID)

	bad := httptest.NewRequest(http.MethodGet, "/api/v1/channel/session?node=n1&surface_id=desktop&transport=websocket&lease_id=lease-1&lease_epoch=0&policy_revision=policy-1", nil)
	_, err = sessionBindingFromQuery(bad, "n1", "owner-1")
	require.Error(t, err)
}

func TestNoSessionHandlerTestConstructsWithoutRegistry(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	f, err := parser.ParseFile(token.NewFileSet(), file, []byte(sessionWSTestSource), 0)
	require.NoError(t, err)
	violations := 0
	ast.Inspect(f, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		ident, ok := literal.Type.(*ast.Ident)
		if !ok || ident.Name != "sessionWSHandler" {
			return true
		}
		for _, elt := range literal.Elts {
			field, ok := elt.(*ast.KeyValueExpr)
			if ok {
				if key, ok := field.Key.(*ast.Ident); ok && key.Name == "registry" {
					return true
				}
			}
		}
		violations++
		return true
	})
	require.Zero(t, violations, "every sessionWSHandler test fixture must exercise the registry-backed production path")
}
