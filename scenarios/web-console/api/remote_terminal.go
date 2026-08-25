package main

// remote_terminal.go owns the web-console → vrooli-bridge federation seam.
// Browser clients continue to speak the stable terminal JSON protocol; this
// server-side adapter is the only place that holds the Bridge owner and
// re-authentication credentials and translates to the binary session.Frame
// protocol. A node never receives browser credentials.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"google.golang.org/protobuf/proto"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

type remoteTerminalTarget struct {
	ID              string                `json:"id"`
	Kind            string                `json:"kind"`
	Label           string                `json:"label"`
	OS              string                `json:"os,omitempty"`
	Arch            string                `json:"arch,omitempty"`
	Revision        string                `json:"revision,omitempty"`
	Status          string                `json:"status,omitempty"`
	Online          bool                  `json:"online"`
	LastSeenAt      time.Time             `json:"last_seen_at,omitempty"`
	Available       bool                  `json:"available"`
	Readiness       []string              `json:"readiness,omitempty"`
	FailureRung     string                `json:"failureRung,omitempty"`
	State           string                `json:"state,omitempty"`
	RecoveryAction  string                `json:"recoveryAction,omitempty"`
	SurvivesRestart bool                  `json:"survives_restart"`
	ReadinessFacts  []remoteReadinessFact `json:"-"`
	BaseURL         string                `json:"-"`
	NodeID          string                `json:"-"`
	OwnerToken      string                `json:"-"`
	ReauthToken     string                `json:"-"`
}

type remoteReadinessFact struct {
	Key    string
	Label  string
	Passed bool
	Detail string
}

type remoteTerminalSession struct {
	ID                   string
	Target               remoteTerminalTarget
	Shell                string
	WorkingDir           string
	LaunchCommand        string
	ExecuteLaunchCommand bool
	Cols                 int
	Rows                 int
	CreatedAt            time.Time
	Launched             bool
	NextBridgeSeq        uint64
	cancel               context.CancelFunc
}

type remoteTerminalRegistry struct {
	mu       sync.RWMutex
	sessions map[string]remoteTerminalSession
}

func (r *remoteTerminalRegistry) put(s remoteTerminalSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sessions == nil {
		r.sessions = make(map[string]remoteTerminalSession)
	}
	r.sessions[s.ID] = s
}

func (r *remoteTerminalRegistry) get(id string) (remoteTerminalSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sessions[id]
	return s, ok
}

func (r *remoteTerminalRegistry) delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, id)
}

func (r *remoteTerminalRegistry) setCancel(id string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.sessions[id]; ok {
		current.cancel = cancel
		r.sessions[id] = current
	}
}

func (r *remoteTerminalRegistry) clearCancel(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.sessions[id]; ok {
		current.cancel = nil
		r.sessions[id] = current
	}
}

func (r *remoteTerminalRegistry) claimLaunch(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.sessions[id]
	if !ok || current.Launched {
		return false
	}
	current.Launched = true
	r.sessions[id] = current
	return true
}

func (r *remoteTerminalRegistry) nextBridgeSequence(id string) (uint64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.sessions[id]
	if !ok {
		return 0, false
	}
	sequence := current.NextBridgeSeq
	current.NextBridgeSeq++
	r.sessions[id] = current
	return sequence, true
}

func (s *Server) remoteRegistry() *remoteTerminalRegistry {
	if s.remoteSessions == nil {
		s.remoteSessions = &remoteTerminalRegistry{sessions: make(map[string]remoteTerminalSession)}
	}
	return s.remoteSessions
}

func configuredRemoteTarget() remoteTerminalTarget {
	ownerToken, reauthToken := resolveBridgeOwnerCredentials()
	t := remoteTerminalTarget{
		ID:             "bridge-node:" + strings.TrimSpace(getEnvOrDefault("WEB_CONSOLE_BRIDGE_NODE_ID", "")),
		Kind:           "bridge-node",
		Label:          getEnvOrDefault("WEB_CONSOLE_BRIDGE_LABEL", "Bridge node"),
		State:          "unconfigured",
		RecoveryAction: "Configure Bridge access on the Web Console server",
		BaseURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_URL")), "/"),
		NodeID:         strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_NODE_ID")),
		OwnerToken:     ownerToken,
		ReauthToken:    reauthToken,
	}
	localSession := hasExplicitAuthScheme(t.OwnerToken, "LocalSession")
	if t.BaseURL == "" {
		t.FailureRung = "Bridge URL not configured"
		return t
	}
	if t.OwnerToken == "" || (!localSession && t.ReauthToken == "") {
		t.FailureRung = "bridge credentials not configured"
		return t
	}
	u, err := url.Parse(t.BaseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		t.FailureRung = "bridge URL is invalid"
		return t
	}
	t.Available = true
	t.State = "dispatchable"
	t.RecoveryAction = ""
	if localSession {
		t.Readiness = []string{"Bridge URL configured", "node identity configured", "enrolled local session configured"}
	} else {
		t.Readiness = []string{"Bridge URL configured", "node identity configured", "owner re-authentication configured"}
	}
	return t
}

// resolveBridgeOwnerCredentials prefers the enrolled operator session on the
// Web Console host. The enrollment store contains only a signing key and
// metadata; Resolve mints the short-lived LocalSession token immediately
// before the server talks to Bridge. Static environment credentials remain a
// compatibility fallback for deployments that have not enrolled the host.
func resolveBridgeOwnerCredentials() (string, string) {
	if store, err := sharedsession.DefaultFileStore(); err == nil {
		if resolution, resolveErr := (sharedsession.LocalResolver{Store: store}).Resolve(); resolveErr == nil && strings.TrimSpace(resolution.Token) != "" {
			return sharedsession.LocalSessionScheme + " " + resolution.Token, ""
		}
	}
	return strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_OWNER_TOKEN")), strings.TrimSpace(os.Getenv("WEB_CONSOLE_BRIDGE_REAUTH_TOKEN"))
}

func hasExplicitAuthScheme(value, scheme string) bool {
	prefix := strings.TrimSpace(scheme) + " "
	return strings.HasPrefix(strings.TrimSpace(value), prefix)
}

// bridgeOwnerTransport keeps Bridge credentials on the web-console server.
// They are injected only into the server-to-server registry request and never
// become browser-visible target fields.
type bridgeOwnerTransport struct {
	base   http.RoundTripper
	owner  string
	reauth string
}

func (t bridgeOwnerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", currentOwnerAuthorization(t.owner))
	if strings.TrimSpace(t.reauth) != "" {
		clone.Header.Set("X-Bridge-Owner-Reauth", t.reauth)
	}
	return t.base.RoundTrip(clone)
}

func readinessFacts(node *registryv1.Node) []string {
	return []string{
		fmt.Sprintf("registry record: %t", node.GetRegistryRecordPresent()),
		fmt.Sprintf("heartbeat fresh: %t", node.GetHeartbeatFresh()),
		fmt.Sprintf("channel held: %t", node.GetChannelHeld()),
		fmt.Sprintf("protocol compatible: %t", node.GetProtocolCompatible()),
		fmt.Sprintf("dispatchable: %t", node.GetDispatchable()),
	}
}

func readinessFailure(node *registryv1.Node) string {
	checks := []struct {
		ok   bool
		name string
	}{
		{node.GetRegistryRecordPresent(), "registry record"},
		{node.GetHeartbeatFresh(), "heartbeat freshness"},
		{node.GetChannelHeld(), "live channel"},
		{node.GetProtocolCompatible(), "protocol compatibility"},
		{node.GetDispatchable(), "dispatchability"},
	}
	for _, check := range checks {
		if !check.ok {
			return check.name
		}
	}
	return ""
}

func targetKind(node *registryv1.Node) string {
	switch node.GetKind() {
	case registryv1.NodeKind_NODE_KIND_SSH:
		return "ssh"
	case registryv1.NodeKind_NODE_KIND_ATTACHED:
		return "attached"
	default:
		return "bridge-node"
	}
}

func targetFromRegistryNode(base remoteTerminalTarget, node *registryv1.Node) remoteTerminalTarget {
	target := base
	target.ID = "bridge-node:" + node.GetId()
	target.Kind = targetKind(node)
	target.Label = node.GetName()
	if target.Label == "" {
		target.Label = node.GetId()
	}
	target.NodeID = node.GetId()
	target.OS = node.GetOs()
	target.Arch = node.GetArch()
	target.Revision = node.GetRevision()
	target.Status = node.GetStatus().String()
	target.Online = node.GetOnline()
	if node.GetLastSeenAt() != nil {
		target.LastSeenAt = node.GetLastSeenAt().AsTime()
	}
	target.Readiness = readinessFacts(node)
	target.ReadinessFacts = readinessFactsForNode(node)
	target.Available = node.GetDispatchable() && node.GetKind() == registryv1.NodeKind_NODE_KIND_AGENT
	if !target.Available {
		if failure := readinessFailure(node); failure != "" {
			target.FailureRung = failure
		} else {
			target.FailureRung = "session backend unavailable for " + target.Kind
		}
		target.RecoveryAction = recoveryActionForNode(node, target.FailureRung)
	}
	target.State = targetStateForNode(node, target.Available)
	return target
}

func configuredRemoteTargets() []remoteTerminalTarget {
	base := configuredRemoteTarget()
	if !base.Available {
		return []remoteTerminalTarget{base}
	}
	client := &http.Client{Timeout: 3 * time.Second, Transport: bridgeOwnerTransport{
		base: http.DefaultTransport, owner: base.OwnerToken, reauth: base.ReauthToken,
	}}
	registryClient := registryconnect.NewNodeRegistryServiceClient(client, base.BaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	response, err := registryClient.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	if err != nil || response == nil || response.Msg == nil {
		base.Available = false
		base.FailureRung = "Bridge registry unavailable"
		base.Readiness = nil
		return []remoteTerminalTarget{base}
	}
	targets := make([]remoteTerminalTarget, 0, len(response.Msg.GetNodes()))
	for _, node := range response.Msg.GetNodes() {
		if node != nil {
			targets = append(targets, targetFromRegistryNode(base, node))
		}
	}
	if len(targets) == 0 {
		base.Available = false
		base.FailureRung = "no registered Bridge nodes"
		base.Readiness = nil
		return []remoteTerminalTarget{base}
	}
	return targets
}

func (s *Server) remoteTargets() []remoteTerminalTarget {
	if s.remoteTargetCatalog != nil {
		return s.remoteTargetCatalog()
	}
	return configuredRemoteTargets()
}

type remoteTerminalCreateRequest struct {
	TargetID             string `json:"target_id"`
	Shell                string `json:"shell"`
	WorkingDir           string `json:"working_dir"`
	LaunchCommand        string `json:"launch_command"`
	ExecuteLaunchCommand bool   `json:"execute_launch_command"`
	Cols                 int    `json:"cols"`
	Rows                 int    `json:"rows"`
}

func (s *Server) handleRemoteTerminalTargets(w http.ResponseWriter, _ *http.Request) {
	writeRemoteJSON(w, http.StatusOK, map[string]any{"targets": s.remoteTargets()})
}

func (s *Server) handleRemoteTerminalCreate(w http.ResponseWriter, r *http.Request) {
	var req remoteTerminalCreateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	targets := s.remoteTargets()
	var target remoteTerminalTarget
	for _, candidate := range targets {
		if candidate.ID == req.TargetID {
			target = candidate
			break
		}
	}
	if target.ID == "" || !target.Available {
		if target.FailureRung == "" {
			target.FailureRung = "target is not dispatchable"
		}
		writeCatalogError(w, "remote_target_unavailable", target.FailureRung)
		return
	}
	if req.Cols <= 0 {
		req.Cols = 80
	}
	if req.Rows <= 0 {
		req.Rows = 24
	}
	id := "remote:" + uuid.NewString()
	now := time.Now().UTC()
	s.remoteRegistry().put(remoteTerminalSession{ID: id, Target: target, Shell: req.Shell, WorkingDir: req.WorkingDir, LaunchCommand: req.LaunchCommand, ExecuteLaunchCommand: req.ExecuteLaunchCommand, Cols: req.Cols, Rows: req.Rows, CreatedAt: now})
	writeRemoteJSON(w, http.StatusCreated, map[string]any{
		"id": id, "shell": req.Shell, "created_at": now, "cols": req.Cols, "rows": req.Rows,
		"backend": "standard", "survives_restart": false, "policy": map[string]any{"mode": "never"},
		"origin": "remote", "owner": "target:" + target.ID, "display_label": target.Label, "target": target,
	})
}

func (s *Server) handleRemoteTerminalList(w http.ResponseWriter, _ *http.Request) {
	s.remoteRegistry().mu.RLock()
	items := make([]map[string]any, 0, len(s.remoteRegistry().sessions))
	for _, current := range s.remoteRegistry().sessions {
		items = append(items, map[string]any{
			"id": current.ID, "shell": current.Shell, "created_at": current.CreatedAt, "cols": current.Cols, "rows": current.Rows,
			"backend": "standard", "survives_restart": false, "policy": map[string]any{"mode": "never"},
			"origin": "remote", "owner": "target:" + current.Target.ID, "display_label": current.Target.Label, "target": current.Target,
		})
	}
	s.remoteRegistry().mu.RUnlock()
	writeRemoteJSON(w, http.StatusOK, map[string]any{"sessions": items})
}

func (s *Server) handleRemoteTerminalDelete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	remote, ok := s.remoteRegistry().get(id)
	if !ok {
		writeCatalogError(w, "session_not_found", "Remote session not found")
		return
	}
	if remote.cancel != nil {
		remote.cancel()
	}
	s.remoteRegistry().delete(id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoteTerminalWS(w http.ResponseWriter, r *http.Request, remote remoteTerminalSession) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	bridgeURL, err := url.Parse(remote.Target.BaseURL)
	if err != nil {
		http.Error(w, "bridge URL is invalid", http.StatusBadGateway)
		return
	}
	bridgeURL.Scheme = websocketScheme(bridgeURL.Scheme)
	bridgeURL.Path = strings.TrimRight(bridgeURL.Path, "/") + "/api/v1/channel/session"
	q := bridgeURL.Query()
	q.Set("node", remote.Target.NodeID)
	q.Set("session_id", remote.ID)
	q.Set("scopes", "vrooli-bridge:session")
	if remote.Shell != "" {
		q.Set("shell", remote.Shell)
	}
	if remote.WorkingDir != "" {
		q.Set("working_dir", remote.WorkingDir)
	}
	bridgeURL.RawQuery = q.Encode()
	header := http.Header{}
	header.Set("Authorization", currentOwnerAuthorization(remote.Target.OwnerToken))
	if strings.TrimSpace(remote.Target.ReauthToken) != "" {
		header.Set("X-Bridge-Owner-Reauth", remote.Target.ReauthToken)
	}
	upstream, _, err := websocket.DefaultDialer.DialContext(ctx, bridgeURL.String(), header)
	if err != nil {
		http.Error(w, "bridge session unavailable", http.StatusBadGateway)
		return
	}
	defer upstream.Close()
	s.remoteRegistry().setCancel(remote.ID, cancel)
	defer s.remoteRegistry().clearCancel(remote.ID)

	client, err := (&websocket.Upgrader{ReadBufferSize: 32 * 1024, WriteBufferSize: 32 * 1024, CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer client.Close()

	var writeMu sync.Mutex
	writeClient := func(msg TerminalMessage) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return client.WriteJSON(msg)
	}
	if err := writeClient(TerminalMessage{Type: MsgTypeHistoryEnd}); err != nil {
		return
	}
	if err := writeClient(TerminalMessage{Type: MsgTypeSizeInfo, Cols: remote.Cols, Rows: remote.Rows, HoldsLease: true}); err != nil {
		return
	}
	if err := writeClient(TerminalMessage{Type: MsgTypeSessionReady, Gen: s.nextWSGen.Add(1)}); err != nil {
		return
	}

	// Bridge owns a durable zero-based data sequence; the browser protocol
	// starts at one. The registry preserves the Bridge sequence across browser
	// reconnects so the same PTY session can be reattached without a gap.
	pending := make(map[uint64]int64)
	var pendingMu sync.Mutex
	if remote.ExecuteLaunchCommand && remote.LaunchCommand != "" && s.remoteRegistry().claimLaunch(remote.ID) {
		sequence, ok := s.remoteRegistry().nextBridgeSequence(remote.ID)
		if !ok {
			return
		}
		if err := writeBridgeFrame(upstream, &sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: sequence, Data: []byte(remote.LaunchCommand + "\n")}}}); err != nil {
			return
		}
	}

	go func() {
		for {
			kind, payload, readErr := upstream.ReadMessage()
			if readErr != nil {
				_ = writeClient(TerminalMessage{Type: MsgTypeExit, Data: "bridge_session_closed"})
				return
			}
			if kind != websocket.BinaryMessage {
				continue
			}
			var frame sessionv1.Frame
			if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &frame); err != nil {
				_ = writeClient(TerminalMessage{Type: MsgTypeError, Data: "bridge frame rejected"})
				return
			}
			switch p := frame.Payload.(type) {
			case *sessionv1.Frame_Data:
				if err := writeClient(TerminalMessage{Type: MsgTypeStdout, Data: string(p.Data.GetData())}); err != nil {
					return
				}
			case *sessionv1.Frame_Ack:
				if !p.Ack.GetAccepted() {
					reason := p.Ack.GetCode()
					if reason == "" {
						reason = p.Ack.GetReason()
					}
					if reason == "" {
						reason = "bridge_session_rejected"
					}
					_ = writeClient(TerminalMessage{Type: MsgTypeError, Data: reason})
					return
				}
				pendingMu.Lock()
				browserSeq, ok := pending[p.Ack.GetSequence()]
				if ok {
					delete(pending, p.Ack.GetSequence())
				}
				pendingMu.Unlock()
				if ok {
					_ = writeClient(TerminalMessage{Type: MsgTypeStdinAck, Seq: browserSeq, Ok: p.Ack.GetAccepted(), Reason: p.Ack.GetReason()})
				}
			case *sessionv1.Frame_Close:
				_ = writeClient(TerminalMessage{Type: MsgTypeExit, Data: p.Close.GetReason()})
				return
			}
		}
	}()

	for {
		_, raw, readErr := client.ReadMessage()
		if readErr != nil {
			// Treat a browser/network disconnect as a transport detach. Closing
			// the upstream Bridge session here would kill the remote PTY and make
			// the browser's reconnect/scrollback contract impossible to satisfy.
			return
		}
		var msg TerminalMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			_ = writeClient(TerminalMessage{Type: MsgTypeError, Data: "invalid terminal message"})
			continue
		}
		switch msg.Type {
		case MsgTypeStdin:
			seq, ok := s.remoteRegistry().nextBridgeSequence(remote.ID)
			if !ok {
				return
			}
			pendingMu.Lock()
			pending[seq] = msg.Seq
			pendingMu.Unlock()
			if err := writeBridgeFrame(upstream, &sessionv1.Frame{Payload: &sessionv1.Frame_Data{Data: &sessionv1.Data{Sequence: seq, Data: []byte(msg.Data)}}}); err != nil {
				return
			}
		case MsgTypeResize:
			if err := writeBridgeFrame(upstream, &sessionv1.Frame{Payload: &sessionv1.Frame_Resize{Resize: &sessionv1.Resize{Columns: uint32(msg.Cols), Rows: uint32(msg.Rows)}}}); err != nil {
				return
			}
			_ = writeClient(TerminalMessage{Type: MsgTypeResizeInfo, Cols: msg.Cols, Rows: msg.Rows})
		case MsgTypePing:
			_ = writeClient(TerminalMessage{Type: MsgTypePong})
		case "close":
			_ = writeBridgeFrame(upstream, &sessionv1.Frame{Payload: &sessionv1.Frame_Close{Close: &sessionv1.Close{Code: "client_closed", Reason: msg.Data}}})
			return
		}
	}
}

func ownerAuthorization(value string) string {
	value = strings.TrimSpace(value)
	if hasExplicitAuthScheme(value, "LocalSession") || hasExplicitAuthScheme(value, "Bearer") {
		return value
	}
	return "Bearer " + value
}

// currentOwnerAuthorization refreshes an enrolled local session immediately
// before a Bridge request. LocalSession credentials are intentionally
// short-lived; retaining the token captured at Web Console startup would make
// an otherwise healthy operator surface fail after the 15-minute TTL. The
// fallback preserves explicit test/configuration tokens when this process has
// no local enrollment to mint from.
func currentOwnerAuthorization(value string) string {
	value = strings.TrimSpace(value)
	if !hasExplicitAuthScheme(value, sharedsession.LocalSessionScheme) {
		return ownerAuthorization(value)
	}
	store, err := sharedsession.DefaultFileStore()
	if err != nil {
		return ownerAuthorization(value)
	}
	resolution, err := (sharedsession.LocalResolver{Store: store}).Resolve()
	if err != nil || strings.TrimSpace(resolution.Token) == "" {
		return ownerAuthorization(value)
	}
	return sharedsession.LocalSessionScheme + " " + resolution.Token
}

func writeBridgeFrame(conn *websocket.Conn, frame *sessionv1.Frame) error {
	payload, err := proto.Marshal(frame)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.BinaryMessage, payload)
}

func websocketScheme(scheme string) string {
	if scheme == "https" {
		return "wss"
	}
	return "ws"
}

func writeRemoteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (s *Server) remoteForRequest(r *http.Request) (remoteTerminalSession, bool) {
	id := mux.Vars(r)["id"]
	if !strings.HasPrefix(id, "remote:") {
		return remoteTerminalSession{}, false
	}
	return s.remoteRegistry().get(id)
}
