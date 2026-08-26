package session

import (
	"errors"

	"web-console/internal/pty"
)

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
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	if info := s.clients[client]; info != nil {
		info.DeclaredCols = cols
		info.DeclaredRows = rows
	}
}

// SetClientDevice records display-only client identity. It is never an
// authorization signal.
func (s *Session) SetClientDevice(client chan []byte, id, label string) {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if info := s.clients[client]; info != nil {
		info.DeviceID, info.DeviceLabel = id, label
		for existing, other := range s.clients {
			if existing != client {
				s.publishPresenceLocked(other)
			}
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

// SizeLeaseState returns the current authoritative grid and display-only
// leader identity for one connection.
func (s *Session) SizeLeaseState(client chan []byte) (cols, rows uint16, leader, leaderDevice string, holds bool, viewerCount int) {
	s.clientsMu.Lock()
	s.emuMu.Lock()
	if info := s.clients[s.leaseOwner]; info != nil {
		leader, leaderDevice = info.DeviceID, info.DeviceLabel
	}
	cols, rows = s.Cols, s.Rows
	holds, viewerCount = client != nil && client == s.leaseOwner, len(s.clients)
	s.emuMu.Unlock()
	s.clientsMu.Unlock()
	return cols, rows, leader, leaderDevice, holds, viewerCount
}
