package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const browserStreamAttachLimit = 4096

type browserStreamProxy struct {
	resolve  desktopURLResolver
	receipts browserPaneReceiptResolver
	client   *http.Client
	dialer   *websocket.Dialer
	upgrader websocket.Upgrader
}

type browserStreamAttach struct {
	Type       string `json:"type"`
	Role       string `json:"role"`
	PaneHandle string `json:"pane_handle"`
}

func newBrowserStreamProxy(resolve desktopURLResolver, receipts ...browserPaneReceiptResolver) *browserStreamProxy {
	var receiptResolver browserPaneReceiptResolver
	if len(receipts) > 0 {
		receiptResolver = receipts[0]
	}
	return &browserStreamProxy{
		resolve: resolve, receipts: receiptResolver,
		client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		dialer: &websocket.Dialer{HandshakeTimeout: 10 * time.Second},
		upgrader: websocket.Upgrader{ReadBufferSize: 16 * 1024, WriteBufferSize: 16 * 1024, CheckOrigin: func(r *http.Request) bool {
			return r.Header.Get("Origin") != "" && webSocketSameOrigin(r)
		}},
	}
}

func webSocketSameOrigin(r *http.Request) bool {
	origin, err := url.Parse(strings.TrimSpace(r.Header.Get("Origin")))
	if err != nil || origin.User != nil || origin.Host == "" || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	authority := strings.TrimSpace(r.Host)
	peer := strings.TrimSpace(r.RemoteAddr)
	if host, _, splitErr := net.SplitHostPort(peer); splitErr == nil {
		peer = host
	} else {
		peer = strings.Trim(peer, "[]")
	}
	if ip := net.ParseIP(peer); ip != nil && ip.IsLoopback() {
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" && !strings.ContainsAny(forwarded, ",/\\") {
			authority = forwarded
		}
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
	}
	return (origin.Scheme == "http" || origin.Scheme == "https") && strings.EqualFold(origin.Scheme, scheme) && strings.EqualFold(origin.Host, authority)
}

func (p *browserStreamProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || p.resolve == nil || r.Method != http.MethodGet {
		http.Error(w, "browser stream unavailable", http.StatusServiceUnavailable)
		return
	}
	// Keep the opaque selector out of the WebSocket URL. URLs commonly reach
	// access logs and browser diagnostics; the first same-origin frame is not
	// persisted in browser history and remains owner-checked below.
	sessionID := ""
	base, err := p.resolve(r.Context())
	if err != nil {
		http.Error(w, "browser session owner unavailable", http.StatusServiceUnavailable)
		return
	}
	baseURL, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil || (baseURL.Scheme != "http" && baseURL.Scheme != "https") || baseURL.Host == "" || baseURL.User != nil {
		http.Error(w, "browser session owner unavailable", http.StatusServiceUnavailable)
		return
	}
	downstream, err := p.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer downstream.Close()
	downstream.SetReadLimit(browserStreamAttachLimit)
	messageType, payload, err := downstream.ReadMessage()
	if err != nil || messageType != websocket.TextMessage {
		writeBrowserProxyError(downstream, "attach_required")
		return
	}
	var attach browserStreamAttach
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&attach); err != nil || attach.Type != "attach" || (attach.Role != "observe" && attach.Role != "control") || !validBrowserPaneHandle(attach.PaneHandle) {
		writeBrowserProxyError(downstream, "attach_invalid")
		return
	}
	if decoder.Decode(new(any)) != io.EOF {
		writeBrowserProxyError(downstream, "attach_invalid")
		return
	}
	if p.receipts == nil {
		writeBrowserProxyError(downstream, "owner_denied")
		return
	}
	sessionID = attach.PaneHandle
	grant, err := p.receipts.IssueHandle(r.Context(), sessionID, attach.Role)
	if err != nil {
		writeBrowserProxyError(downstream, "owner_denied")
		return
	}
	ownerSessionID := grant.SessionID
	ownerCookie, err := p.issueOwnerCookie(r.Context(), baseURL, ownerSessionID, grant.Receipt, attach.Role)
	if err != nil {
		writeBrowserProxyError(downstream, "owner_denied")
		return
	}
	streamURL := *baseURL
	if streamURL.Scheme == "https" {
		streamURL.Scheme = "wss"
	} else {
		streamURL.Scheme = "ws"
	}
	streamURL.Path = strings.TrimRight(streamURL.Path, "/") + "/ws/sessions/" + url.PathEscape(ownerSessionID)
	streamURL.RawQuery = ""
	upstream, response, err := p.dialer.DialContext(r.Context(), streamURL.String(), http.Header{"Cookie": []string{ownerCookie}})
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		writeBrowserProxyError(downstream, "owner_stream_unavailable")
		return
	}
	defer upstream.Close()
	downstream.SetReadLimit(1 << 20)
	upstream.SetReadLimit(9 << 20)
	proxyBrowserWebSockets(downstream, upstream, ownerSessionID, sessionID)
}

func validBrowserPaneHandle(value string) bool {
	if len(value) != 48 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func validBrowserSessionSelector(value string) bool {
	if value == "" || len(value) > 160 || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (p *browserStreamProxy) issueOwnerCookie(ctx context.Context, base *url.URL, sessionID, receipt, role string) (string, error) {
	requestURL := *base
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + "/api/v1/browser/sessions/" + url.PathEscape(sessionID) + "/stream-cookie"
	body, err := json.Marshal(map[string]string{"receipt": receipt, "role": role})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if base.Scheme == "https" {
		request.Header.Set("Origin", "https://"+base.Host)
	} else {
		request.Header.Set("Origin", "http://"+base.Host)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("owner refused stream cookie: %s", response.Status)
	}
	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, 16*1024)); err != nil {
		return "", err
	}
	wantName, wantPath := "bas_ws_"+sessionID, "/ws/sessions/"+sessionID
	for _, cookie := range response.Cookies() {
		secureRequired := base.Scheme == "https"
		if cookie.Name == wantName && cookie.Value != "" && cookie.Path == wantPath && cookie.HttpOnly && cookie.SameSite == http.SameSiteStrictMode && (!secureRequired || cookie.Secure) {
			return cookie.Name + "=" + cookie.Value, nil
		}
	}
	return "", errors.New("owner did not issue the expected path-scoped stream cookie")
}

func writeBrowserProxyError(conn *websocket.Conn, reason string) {
	_ = conn.WriteJSON(map[string]string{"type": "browser_stream_error", "reason": reason})
}

func proxyBrowserWebSockets(downstream, upstream *websocket.Conn, sessionID, paneHandle string) {
	results := make(chan error, 2)
	copyMessages := func(destination, source *websocket.Conn, inbound bool) {
		for {
			messageType, payload, err := source.ReadMessage()
			if err != nil {
				results <- err
				return
			}
			if inbound {
				mapped, mapErr := rewriteBrowserHubSessionID(messageType, payload, paneHandle, sessionID)
				if mapErr != nil || messageType != websocket.TextMessage || !allowedBrowserHubInput(mapped, sessionID) {
					_ = downstream.WriteJSON(map[string]string{"type": "browser_stream_error", "reason": "message_not_allowed"})
					results <- errors.New("browser stream message outside session allowlist")
					return
				}
				payload = mapped
			} else {
				if !allowedBrowserHubOutput(messageType, payload, sessionID) {
					continue
				}
				if messageType == websocket.BinaryMessage {
					payload, err = sanitizeBrowserBinaryFrame(payload, sessionID)
					if err != nil {
						continue
					}
					payload, err = rewriteBinaryBrowserSessionID(payload, sessionID, paneHandle)
					if err != nil {
						continue
					}
				} else {
					payload, err = rewriteBrowserHubSessionID(messageType, payload, sessionID, paneHandle)
					if err != nil {
						continue
					}
				}
			}
			_ = destination.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := destination.WriteMessage(messageType, payload); err != nil {
				results <- err
				return
			}
		}
	}
	go copyMessages(upstream, downstream, true)
	go copyMessages(downstream, upstream, false)
	<-results
	_ = downstream.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	_ = upstream.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
}

func rewriteBrowserHubSessionID(messageType int, payload []byte, from, to string) ([]byte, error) {
	if messageType != websocket.TextMessage {
		return payload, nil
	}
	var message map[string]json.RawMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return nil, err
	}
	raw, ok := message["session_id"]
	if !ok {
		return nil, errors.New("session id missing")
	}
	var selected string
	if json.Unmarshal(raw, &selected) != nil || selected != from {
		return nil, errors.New("session id mismatch")
	}
	value, _ := json.Marshal(to)
	message["session_id"] = value
	return json.Marshal(message)
}

func rewriteBinaryBrowserSessionID(payload []byte, from, to string) ([]byte, error) {
	if len(payload) < 4 {
		return nil, errors.New("frame too short")
	}
	n := int(binary.BigEndian.Uint32(payload[:4]))
	if n <= 0 || 4+n > len(payload) {
		return nil, errors.New("frame header invalid")
	}
	var h map[string]json.RawMessage
	if err := json.Unmarshal(payload[4:4+n], &h); err != nil {
		return nil, err
	}
	raw := h["session_id"]
	var selected string
	if json.Unmarshal(raw, &selected) != nil || selected != from {
		return nil, errors.New("frame session mismatch")
	}
	value, _ := json.Marshal(to)
	h["session_id"] = value
	header, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 4+len(header)+len(payload)-4-n)
	binary.BigEndian.PutUint32(out[:4], uint32(len(header)))
	copy(out[4:], header)
	copy(out[4+len(header):], payload[4+n:])
	return out, nil
}

func allowedBrowserHubInput(payload []byte, sessionID string) bool {
	var message map[string]json.RawMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return false
	}
	var messageType string
	if json.Unmarshal(message["type"], &messageType) != nil {
		return false
	}
	switch messageType {
	case "subscribe_recording":
		var selected string
		if json.Unmarshal(message["session_id"], &selected) != nil || selected != sessionID || len(message) < 2 || len(message) > 3 {
			return false
		}
		if rawFrames, exists := message["frames"]; exists {
			var frames bool
			if json.Unmarshal(rawFrames, &frames) != nil {
				return false
			}
		}
		return len(message) == 2 && message["type"] != nil && message["session_id"] != nil || len(message) == 3 && message["type"] != nil && message["session_id"] != nil && message["frames"] != nil
	case "unsubscribe_recording":
		return len(message) == 1 && message["type"] != nil
	case "recording_input":
		var selected string
		return json.Unmarshal(message["session_id"], &selected) == nil && selected == sessionID && len(message) == 3 && message["input"] != nil
	default:
		return false
	}
}

func allowedBrowserHubOutput(messageType int, payload []byte, sessionID string) bool {
	if messageType == websocket.BinaryMessage {
		_, err := sanitizeBrowserBinaryFrame(payload, sessionID)
		return err == nil
	}
	if messageType != websocket.TextMessage || len(payload) > 8<<20 {
		return false
	}
	var message struct {
		Type      string `json:"type"`
		SessionID string `json:"session_id"`
		Data      string `json:"data"`
		MediaType string `json:"media_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if json.Unmarshal(payload, &message) != nil || message.SessionID != sessionID {
		return false
	}
	switch message.Type {
	case "recording_subscribed", "recording_input_applied":
		return true
	case "session_access_revoked":
		var fields map[string]json.RawMessage
		return json.Unmarshal(payload, &fields) == nil && len(fields) == 2 && fields["type"] != nil && fields["session_id"] != nil && message.SessionID == sessionID
	case "session_frame":
		return message.Data != "" && len(message.Data) <= 8<<20 && (message.MediaType == "image/jpeg" || message.MediaType == "image/png" || message.MediaType == "image/webp") && message.Width > 0 && message.Height > 0
	default:
		return false
	}
}

// sanitizeBrowserBinaryFrame verifies BAS's source-bearing JPEG envelope and
// removes page URL/title/page ID metadata before bytes enter browser state.
func sanitizeBrowserBinaryFrame(payload []byte, sessionID string) ([]byte, error) {
	if len(payload) < 4 || len(payload) > 8<<20 {
		return nil, errors.New("binary frame size is outside the bounded envelope")
	}
	headerLength := int(binary.BigEndian.Uint32(payload[:4]))
	if headerLength <= 0 || headerLength > 16<<10 || 4+headerLength+2 > len(payload) {
		return nil, errors.New("binary frame header is invalid")
	}
	var header struct {
		Version    int    `json:"version"`
		SessionID  string `json:"session_id"`
		CapturedAt string `json:"captured_at"`
	}
	if err := json.Unmarshal(payload[4:4+headerLength], &header); err != nil || header.Version != 1 || header.SessionID != sessionID {
		return nil, errors.New("binary frame does not belong to the selected session")
	}
	jpeg := payload[4+headerLength:]
	if jpeg[0] != 0xff || jpeg[1] != 0xd8 {
		return nil, errors.New("binary frame is not a JPEG image")
	}
	safeHeader, err := json.Marshal(struct {
		Version    int    `json:"version"`
		SessionID  string `json:"session_id"`
		CapturedAt string `json:"captured_at"`
	}{header.Version, header.SessionID, header.CapturedAt})
	if err != nil || len(safeHeader) > 16<<10 {
		return nil, errors.New("binary frame safe header could not be encoded")
	}
	result := make([]byte, 4+len(safeHeader)+len(jpeg))
	binary.BigEndian.PutUint32(result[:4], uint32(len(safeHeader)))
	copy(result[4:], safeHeader)
	copy(result[4+len(safeHeader):], jpeg)
	return result, nil
}
