package handlers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/performance"
)

type frameCaptureHub struct {
	*MockHub
	frames [][]byte
}

func (h *frameCaptureHub) HasRecordingFrameSubscribers(string) bool { return true }
func (h *frameCaptureHub) BroadcastBinaryFrame(_ string, frame []byte) {
	h.frames = append(h.frames, append([]byte(nil), frame...))
}

func ownedFrameFixture(t *testing.T) (*Handler, *MockRecordModeService, *autosession.Session, map[string]string, *frameCaptureHub) {
	t.Helper()
	h, service, dir, hub := createTestHandlerWithRecordMode(t)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	owner := uuid.New()
	source := map[string]string{"session_id": "frame-session", "execution_id": owner.String(), "lease_id": "frame-lease", "page_id": "driver-red"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"session_id": source["session_id"], "lease_id": source["lease_id"]})
	}))
	t.Cleanup(server.Close)
	client, err := driver.NewClientWithURL(server.URL, driver.WithoutCircuitBreaker())
	require.NoError(t, err)
	sess, err := autosession.NewManagerWithClient(client).Create(context.Background(), autosession.Spec{ExecutionID: owner, Mode: autosession.ModeRecording})
	require.NoError(t, err)
	sess.InitializePageTracking("https://red.test")
	sess.Pages().SetInitialPageDriverID(source["page_id"])
	service.OwnedSessions = map[string]*autosession.Session{source["session_id"]: sess}
	frames := &frameCaptureHub{MockHub: hub}
	h.wsHub = frames
	return h, service, sess, source, frames
}

// [REQ:BAS-RH-J05] [REQ:BAS-RH-J22] Source identity survives the API boundary.
func TestDriverFrameSourceAdmission(t *testing.T) {
	for _, change := range []string{"current", "session", "execution", "lease", "page", "anonymous", "version", "retired session"} {
		t.Run(change, func(t *testing.T) {
			h, service, sess, source, hub := ownedFrameFixture(t)
			canonical := sess.Pages().GetActivePageID().String()
			router := chi.NewRouter()
			done := make(chan struct{})
			router.Get("/{sessionId}", func(w http.ResponseWriter, r *http.Request) { defer close(done); h.HandleDriverFrameStream(w, r) })
			server := httptest.NewServer(router)
			defer server.Close()
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/frame-session", nil)
			require.NoError(t, err)
			defer conn.Close()
			version := 1
			switch change {
			case "session", "execution", "lease", "page":
				source[change+"_id"] = "foreign"
			case "version":
				version = 99
			case "retired session":
				service.mu.Lock()
				delete(service.OwnedSessions, "frame-session")
				service.mu.Unlock()
			}
			jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}
			header, err := json.Marshal(map[string]any{"version": version, "source": source, "captured_at": "2026-09-23T00:00:00Z"})
			require.NoError(t, err)
			packet := make([]byte, 4+len(header)+len(jpeg))
			binary.BigEndian.PutUint32(packet, uint32(len(header)))
			copy(packet[4:], header)
			copy(packet[4+len(header):], jpeg)
			if change == "anonymous" {
				packet = jpeg
			}
			require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, packet))
			require.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("frame stream did not finish")
			}
			if change != "current" {
				require.Empty(t, hub.frames)
				return
			}
			require.Len(t, hub.frames, 1)
			published := hub.frames[0]
			require.Greater(t, len(published), 4)
			length := int(binary.BigEndian.Uint32(published))
			require.Less(t, length, len(published)-4)
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(published[4:4+length], &envelope))
			require.Equal(t, "frame-session", envelope["session_id"])
			require.Equal(t, canonical, envelope["page_id"])
			require.NotContains(t, string(published), source["lease_id"], "browser must not receive driver lease credential")
			require.Equal(t, jpeg, published[4+length:])
		})
	}
}

func TestDecodeDriverFrame(t *testing.T) {
	source := &driver.FrameSource{SessionID: "session", ExecutionID: "execution", LeaseID: "lease", PageID: "page"}
	header := driverFrameHeader{Version: 1, Source: source, CapturedAt: time.Now(), Timing: &performance.FrameHeader{FrameID: "frame-1", CaptureMs: 4}}
	encoded, err := json.Marshal(header)
	require.NoError(t, err)
	jpeg := []byte{0xff, 0xd8, 0xff, 0xd9}
	packet := make([]byte, 4+len(encoded)+len(jpeg))
	binary.BigEndian.PutUint32(packet, uint32(len(encoded)))
	copy(packet[4:], encoded)
	copy(packet[4+len(encoded):], jpeg)
	payload, decoded, err := decodeDriverFrame(packet)
	require.NoError(t, err)
	require.Equal(t, jpeg, payload)
	require.Equal(t, source, decoded.Source)
	require.Equal(t, "frame-1", decoded.Timing.FrameID)
	for name, invalid := range map[string][]byte{
		"anonymous": jpeg, "empty": nil, "missing payload": packet[:4+len(encoded)],
		"oversized header": {0, 1, 0, 0, 0, 0}, "invalid JSON": {0, 0, 0, 1, '{', 0xff, 0xd8},
		"truncated": packet[:10],
	} {
		t.Run(name, func(t *testing.T) { _, _, err := decodeDriverFrame(invalid); require.Error(t, err) })
	}
}

func TestDriverFrameStatsExcludeIdleSocketWait(t *testing.T) {
	h, _, _, source, _ := ownedFrameFixture(t)
	collector := h.perfRegistry.GetOrCreate("frame-session")
	done := make(chan struct{})
	r := chi.NewRouter()
	r.Get("/{sessionId}", func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		h.HandleDriverFrameStream(w, r)
	})
	server := httptest.NewServer(r)
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/frame-session", nil)
	require.NoError(t, err)
	defer conn.Close()
	// Deliberate absence of frames is idle time, never frame processing.
	time.Sleep(100 * time.Millisecond)
	header, err := json.Marshal(map[string]any{"version": 1, "source": source, "captured_at": "2026-09-23T00:00:00Z", "timing": map[string]any{"frame_id": "idle-1", "capture_ms": 2}})
	require.NoError(t, err)
	packet := make([]byte, 4+len(header)+4)
	binary.BigEndian.PutUint32(packet, uint32(len(header)))
	copy(packet[4:], header)
	copy(packet[4+len(header):], []byte{0xff, 0xd8, 0xff, 0xd9})
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, packet))
	require.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("frame handler did not finish")
	}
	frames := collector.GetRecentFrames(1)
	require.Len(t, frames, 1)
	require.Zero(t, frames[0].APIReceiveMs, "message availability is not a transit-time measurement")
	require.GreaterOrEqual(t, frames[0].APITotalMs, frames[0].APIBroadcastMs)
}

func TestHTTPFrameSourceAdmission(t *testing.T) {
	for _, change := range []string{"implicit", "explicit", "wrong page request", "page changes during capture", "wrong lease", "retired owner", "missing source", "driver conflict"} {
		t.Run(change, func(t *testing.T) {
			h, service, sess, source, _ := ownedFrameFixture(t)
			canonical := sess.Pages().GetActivePageID()
			receipt := &driver.GetFrameResponse{SessionID: source["session_id"], Source: &driver.FrameSource{SessionID: source["session_id"], ExecutionID: source["execution_id"], LeaseID: source["lease_id"], PageID: source["page_id"]}, Image: "jpeg", ContentHash: "same"}
			query := "?quality=65"
			if change == "explicit" {
				query += "&page_id=" + canonical.String()
			}
			if change == "wrong page request" {
				query += "&page_id=" + uuid.NewString()
			}
			called := false
			service.MockClient().GetFrameFunc = func(_ context.Context, sessionID, params string) (*driver.GetFrameResponse, error) {
				called = true
				values, err := url.ParseQuery(params)
				require.NoError(t, err)
				require.Equal(t, "driver-red", values.Get("page_id"))
				require.Equal(t, "65", values.Get("quality"))
				require.Equal(t, "frame-session", sessionID)
				switch change {
				case "page changes during capture":
					next := sess.Pages().AddPage(&domain.Page{DriverPageID: "driver-blue"})
					require.NoError(t, sess.Pages().SetActivePage(next.ID))
				case "wrong lease":
					receipt.Source.LeaseID = "retired"
				case "retired owner":
					delete(service.OwnedSessions, sessionID)
				case "missing source":
					receipt.Source = nil
				case "driver conflict":
					return nil, &driver.Error{Status: http.StatusConflict}
				}
				return receipt, nil
			}
			router := chi.NewRouter()
			router.Get("/{sessionId}", h.GetRecordingFrame)
			req := httptest.NewRequest(http.MethodGet, "/frame-session"+query, nil)
			if change != "implicit" && change != "explicit" {
				req.Header.Set("If-None-Match", `"same"`)
			}
			out := httptest.NewRecorder()
			router.ServeHTTP(out, req)
			if change == "implicit" || change == "explicit" {
				require.Equal(t, http.StatusOK, out.Code, out.Body.String())
				var frame map[string]any
				require.NoError(t, json.Unmarshal(out.Body.Bytes(), &frame))
				require.Equal(t, canonical.String(), frame["page_id"])
				require.NotContains(t, frame, "source")
				require.NotNil(t, receipt.Source, "projection must not mutate the driver receipt")
			} else {
				require.Equal(t, http.StatusConflict, out.Code, out.Body.String())
			}
			require.Equal(t, change != "wrong page request", called)
		})
	}
}
