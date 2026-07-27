package session

import "errors"

// LeaseReason records why the terminal-size authority last moved.
type LeaseReason string

const (
	LeaseReasonFirstClient      LeaseReason = "first_client"
	LeaseReasonInput            LeaseReason = "input"
	LeaseReasonExplicit         LeaseReason = "explicit"
	LeaseReasonLeaderDisconnect LeaseReason = "leader_disconnect"
)

var ErrLeaseNotHeld = errors.New("terminal size lease is not held by this connection")

// DeclareSize records a connection's preferred grid without changing the PTY.
func (s *Session) DeclareSize(client chan []byte, cols, rows uint16) {
	if client == nil || cols == 0 || rows == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if info := s.clients[client]; info != nil {
		info.DeclaredCols = cols
		info.DeclaredRows = rows
	}
}

// SetClientDevice records display-only client identity. It is never an
// authorization signal.
func (s *Session) SetClientDevice(client chan []byte, id, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info := s.clients[client]; info != nil {
		info.DeviceID, info.DeviceLabel = id, label
	}
}

// AcquireLease transfers size authority then applies the new owner's last
// declaration through the single locked resize path.
func (s *Session) AcquireLease(client chan []byte, reason LeaseReason) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clients[client] == nil {
		return ErrLeaseNotHeld
	}
	s.leaseOwner, s.leaseReason = client, reason
	s.applyDeclaredSizeLocked(client)
	s.publishAllSizesLocked()
	return nil
}

func (s *Session) HoldsLease(client chan []byte) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return client != nil && client == s.leaseOwner
}

func (s *Session) applyDeclaredSizeLocked(client chan []byte) {
	info := s.clients[client]
	if info == nil || info.DeclaredCols == 0 || info.DeclaredRows == 0 {
		return
	}
	s.applyResizeLocked(info.DeclaredCols, info.DeclaredRows)
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

// SizeLeaseState returns the current authoritative grid and display-only
// leader identity for one connection.
func (s *Session) SizeLeaseState(client chan []byte) (cols, rows uint16, leader, leaderDevice string, holds bool, viewerCount int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info := s.clients[s.leaseOwner]; info != nil {
		leader, leaderDevice = info.DeviceID, info.DeviceLabel
	}
	return s.Cols, s.Rows, leader, leaderDevice, client != nil && client == s.leaseOwner, len(s.clients)
}

// Viewers returns non-empty display labels for the connected viewers.
func (s *Session) Viewers() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	viewers := make([]string, 0, len(s.clients))
	for _, info := range s.clients {
		if info.DeviceLabel != "" {
			viewers = append(viewers, info.DeviceLabel)
		}
	}
	return viewers
}
