package session

import (
	"errors"
	"sort"
	"time"

	"web-console/internal/pty"
)

// DeviceConnection is one live socket belonging to a device.
type DeviceConnection struct {
	ConnID       string
	SubscribedAt time.Time
	HoldsLease   bool
}

// DeviceView is a session-local projection of live connections grouped by
// browser-local device identity.
type DeviceView struct {
	DeviceID    string
	DeviceLabel string
	DeviceClass string
	KbOpen      bool
	HoldsLease  bool
	Connections []DeviceConnection
}

// leaseHolderForDeviceLocked returns the current lease holder for a device,
// excluding the arriving connection. The caller must hold clientsMu.
func (s *Session) leaseHolderForDeviceLocked(deviceID string, exclude chan []byte) chan []byte {
	if deviceID == "" {
		return nil
	}
	if holder := s.clients[s.leaseOwner]; holder != nil && s.leaseOwner != exclude && holder.DeviceID == deviceID {
		return s.leaseOwner
	}
	return nil
}

// ConnectedDevices projects this session's live clients without retaining a
// second roster state. Connections without an identity remain distinct.
func (s *Session) ConnectedDevices() []DeviceView {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	groups := make(map[string]*DeviceView)
	for ch, info := range s.clients {
		key := info.DeviceID
		if key == "" {
			key = "\x00" + info.ConnID
		}
		view := groups[key]
		if view == nil {
			view = &DeviceView{DeviceID: info.DeviceID, DeviceLabel: info.DeviceLabel, DeviceClass: info.DeviceClass, KbOpen: info.KbOpen, HoldsLease: ch == s.leaseOwner}
			groups[key] = view
		}
		view.Connections = append(view.Connections, DeviceConnection{ConnID: info.ConnID, SubscribedAt: info.SubscribedAt, HoldsLease: ch == s.leaseOwner})
	}
	result := make([]DeviceView, 0, len(groups))
	for _, view := range groups {
		sort.Slice(view.Connections, func(i, j int) bool { return view.Connections[i].SubscribedAt.Before(view.Connections[j].SubscribedAt) })
		result = append(result, *view)
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].Connections) == 0 || len(result[j].Connections) == 0 {
			return len(result[i].Connections) > len(result[j].Connections)
		}
		return result[i].Connections[0].SubscribedAt.Before(result[j].Connections[0].SubscribedAt)
	})
	return result
}

// LeaseReason records why the terminal-size authority last moved.
type LeaseReason string

const (
	LeaseReasonFirstClient      LeaseReason = "first_client"
	LeaseReasonInput            LeaseReason = "input"
	LeaseReasonExplicit         LeaseReason = "explicit"
	LeaseReasonLeaderDisconnect LeaseReason = "leader_disconnect"
	LeaseReasonDeviceReclaim    LeaseReason = "device_reclaim"
)

const deviceReclaimGrace = 2 * time.Second

var ErrLeaseNotHeld = errors.New("terminal size lease is not held by this connection")

// DeclareSize records a connection's preferred grid without changing the PTY.
func (s *Session) DeclareSize(client chan []byte, cols, rows uint16) {
	if client == nil || cols == 0 || rows == 0 {
		return
	}
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if info := s.clients[client]; info != nil {
		info.DeclaredCols = cols
		info.DeclaredRows = rows
	}
}

// SetClientDevice records display-only client identity. It is never an
// authorization signal.
func (s *Session) SetClientDevice(client chan []byte, id, label, class string) {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if info := s.clients[client]; info != nil {
		info.DeviceID, info.DeviceLabel, info.DeviceClass = id, label, class
		for existing, other := range s.clients {
			if existing != client {
				s.publishPresenceLocked(other)
			}
		}
	}
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
}

// SetClientProbe installs the transport-owned liveness probe for a client.
func (s *Session) SetClientProbe(client chan []byte, probe func()) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if info := s.clients[client]; info != nil {
		info.probe = probe
	}
}

// MarkClientPong records a successful WebSocket liveness response.
func (s *Session) MarkClientPong(client chan []byte) {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if info := s.clients[client]; info != nil && info.pongCh != nil {
		select {
		case info.pongCh <- struct{}{}:
		default:
		}
	}
}

// ProbeClient asks the transport to probe a connection and waits briefly for
// its pong. The transport callback is invoked without the session lock held.
func (s *Session) ProbeClient(client chan []byte) bool {
	s.clientsMu.Lock()
	info := s.clients[client]
	if info == nil || info.probe == nil || info.pongCh == nil {
		s.clientsMu.Unlock()
		return false
	}
	probe, pongCh := info.probe, info.pongCh
	for {
		select {
		case <-pongCh:
		default:
			goto drained
		}
	}
drained:
	s.clientsMu.Unlock()
	probe()
	select {
	case <-pongCh:
		return true
	case <-time.After(deviceReclaimGrace):
		return false
	}
}

// SetClientKeyboard records whether this connection's virtual keyboard covers
// part of its viewport. Only the lease owner's value reaches followers, but it
// is stored for every connection so a lease handover carries the current state
// rather than waiting for the new leader's next keyboard event.
func (s *Session) SetClientKeyboard(client chan []byte, open bool) {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if info := s.clients[client]; info != nil && info.KbOpen != open {
		info.KbOpen = open
		// Only the leader's keyboard is presented, so a follower toggling its
		// own keyboard must not wake every other viewer.
		if info == s.clients[s.leaseOwner] {
			s.publishAllPresenceLocked()
		}
	}
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
}

// AcquireLease transfers size authority then applies the new owner's last
// declaration through the single locked resize path.
func (s *Session) AcquireLease(client chan []byte, reason LeaseReason) error {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if s.clients[client] == nil {
		s.clientsMu.Unlock()
		s.emuMu.Unlock()
		return ErrLeaseNotHeld
	}
	s.leaseOwner, s.leaseReason = client, reason
	p, cols, rows := s.applyDeclaredSizeLocked(client)
	s.publishAllSizesLocked()
	s.publishAllPresenceLocked()
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
	if p != nil {
		_ = p.SetSize(cols, rows)
	}
	return nil
}

func (s *Session) HoldsLease(client chan []byte) bool {
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	return client != nil && client == s.leaseOwner
}

func (s *Session) applyDeclaredSizeLocked(client chan []byte) (pty.PTY, uint16, uint16) {
	info := s.clients[client]
	if info == nil || info.DeclaredCols == 0 || info.DeclaredRows == 0 {
		return nil, 0, 0
	}
	return s.applyResizeLocked(info.DeclaredCols, info.DeclaredRows), info.DeclaredCols, info.DeclaredRows
}

func (s *Session) oldestClientLocked() chan []byte {
	var oldest chan []byte
	var order uint64
	for ch, info := range s.clients {
		if oldest == nil || info.SubscribedOrder < order {
			oldest, order = ch, info.SubscribedOrder
		}
	}
	return oldest
}

func (s *Session) publishSizeLocked(info *ClientInfo) {
	if info == nil || info.SizeCh == nil {
		return
	}
	next := [2]uint16{s.Cols, s.Rows}
	select {
	case info.SizeCh <- next:
	default:
		select {
		case <-info.SizeCh:
		default:
		}
		select {
		case info.SizeCh <- next:
		default:
		}
	}
}

func (s *Session) publishAllSizesLocked() {
	for _, info := range s.clients {
		s.publishSizeLocked(info)
	}
}

func (s *Session) publishPresenceLocked(info *ClientInfo) {
	if info == nil || info.PresenceCh == nil {
		return
	}
	state := PresenceState{ViewerCount: len(s.clients), HoldsLease: info == s.clients[s.leaseOwner]}
	if leader := s.clients[s.leaseOwner]; leader != nil {
		state.Leader, state.LeaderDevice = leader.DeviceID, leader.DeviceLabel
		state.LeaderClass, state.LeaderKbOpen = leader.DeviceClass, leader.KbOpen
	}
	select {
	case info.PresenceCh <- state:
	default:
		select {
		case <-info.PresenceCh:
		default:
		}
		select {
		case info.PresenceCh <- state:
		default:
		}
	}
}

func (s *Session) publishAllPresenceLocked() {
	for _, info := range s.clients {
		s.publishPresenceLocked(info)
	}
}

// SizeLeaseSnapshot is one connection's view of the authoritative grid and the
// display-only leader identity, captured under a single lock acquisition.
//
// It exists so callers share one construction of the size_info payload. The
// previous six-value return had to be widened, and every call site rewritten,
// each time a presentational field was added.
type SizeLeaseSnapshot struct {
	Cols         uint16
	Rows         uint16
	Leader       string
	LeaderDevice string
	LeaderClass  string
	LeaderKbOpen bool
	HoldsLease   bool
	ViewerCount  int
}

// SizeLeaseState returns the current authoritative grid and display-only
// leader identity for one connection.
func (s *Session) SizeLeaseState(client chan []byte) SizeLeaseSnapshot {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	snapshot := SizeLeaseSnapshot{
		Cols:        s.Cols,
		Rows:        s.Rows,
		HoldsLease:  client != nil && client == s.leaseOwner,
		ViewerCount: len(s.clients),
	}
	if info := s.clients[s.leaseOwner]; info != nil {
		snapshot.Leader, snapshot.LeaderDevice = info.DeviceID, info.DeviceLabel
		snapshot.LeaderClass, snapshot.LeaderKbOpen = info.DeviceClass, info.KbOpen
	}
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
	return snapshot
}
