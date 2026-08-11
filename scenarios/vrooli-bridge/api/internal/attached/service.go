package attached

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID, Name, HostNodeID, Kind, Transport, Serial, OSVersion, TrustState, Reachability, HealthReason string
	CreatedAt, RevokedAt                                                                             time.Time
}
type PairInput struct {
	Name, HostNodeID, Kind, Transport, Serial, OSVersion string
	HostNodeOnline                                       bool
}
type Service struct {
	mu      sync.Mutex
	devices map[string]Device
	db      *sql.DB
}

func NewService() *Service { return &Service{devices: map[string]Device{}} }
func NewServiceWithDB(db *sql.DB) (*Service, error) {
	s := NewService()
	s.db = db
	if db == nil {
		return s, nil
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS bridge_attached_devices (id TEXT PRIMARY KEY, name TEXT NOT NULL, host_node_id TEXT NOT NULL, kind TEXT NOT NULL, transport TEXT NOT NULL, serial TEXT NOT NULL, os_version TEXT NOT NULL, trust_state TEXT NOT NULL, reachability TEXT NOT NULL, health_reason TEXT NOT NULL, created_at TEXT NOT NULL, revoked_at TEXT NOT NULL DEFAULT '')`); err != nil {
		return nil, fmt.Errorf("initialize attached-device registry: %w", err)
	}
	return s, nil
}

func (s *Service) Pair(_ context.Context, in PairInput) (Device, error) {
	if strings.TrimSpace(in.HostNodeID) == "" || strings.TrimSpace(in.Kind) == "" {
		return Device{}, fmt.Errorf("host_node_id and kind are required")
	}
	now := time.Now().UTC()
	d := Device{ID: uuid.NewString(), Name: strings.TrimSpace(in.Name), HostNodeID: strings.TrimSpace(in.HostNodeID), Kind: strings.TrimSpace(in.Kind), Transport: strings.TrimSpace(in.Transport), Serial: strings.TrimSpace(in.Serial), OSVersion: strings.TrimSpace(in.OSVersion), TrustState: "trusted", Reachability: "reachable", CreatedAt: now}
	if !in.HostNodeOnline {
		d.Reachability = "unreachable"
		d.HealthReason = "host node " + d.HostNodeID + " is offline"
	}
	s.mu.Lock()
	if s.db != nil {
		if _, err := s.db.Exec(`INSERT INTO bridge_attached_devices (id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, d.Name, d.HostNodeID, d.Kind, d.Transport, d.Serial, d.OSVersion, d.TrustState, d.Reachability, d.HealthReason, d.CreatedAt.Format(time.RFC3339Nano), ""); err != nil {
			s.mu.Unlock()
			return Device{}, fmt.Errorf("persist attached device: %w", err)
		}
	}
	s.devices[d.ID] = d
	s.mu.Unlock()
	return d, nil
}

func (s *Service) List(_ context.Context) []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.Query(`SELECT id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at FROM bridge_attached_devices WHERE revoked_at = '' ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Device, 0)
			for rows.Next() {
				var d Device
				var created, revoked string
				if rows.Scan(&d.ID, &d.Name, &d.HostNodeID, &d.Kind, &d.Transport, &d.Serial, &d.OSVersion, &d.TrustState, &d.Reachability, &d.HealthReason, &created, &revoked) == nil {
					d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
					if revoked != "" {
						d.RevokedAt, _ = time.Parse(time.RFC3339Nano, revoked)
					}
					out = append(out, d)
				}
			}
			return out
		}
	}
	out := make([]Device, 0, len(s.devices))
	for _, d := range s.devices {
		if d.RevokedAt.IsZero() {
			out = append(out, d)
		}
	}
	return out
}

func (s *Service) Revoke(_ context.Context, id string) (Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[id]
	if !ok && s.db != nil {
		var created, revoked string
		row := s.db.QueryRow(`SELECT id,name,host_node_id,kind,transport,serial,os_version,trust_state,reachability,health_reason,created_at,revoked_at FROM bridge_attached_devices WHERE id = ?`, id)
		if err := row.Scan(&d.ID, &d.Name, &d.HostNodeID, &d.Kind, &d.Transport, &d.Serial, &d.OSVersion, &d.TrustState, &d.Reachability, &d.HealthReason, &created, &revoked); err == nil {
			d.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
			if revoked != "" {
				d.RevokedAt, _ = time.Parse(time.RFC3339Nano, revoked)
			}
			ok = true
		}
	}
	if !ok {
		return Device{}, fmt.Errorf("attached device %q not found", id)
	}
	d.RevokedAt = time.Now().UTC()
	d.TrustState = "revoked"
	if s.db != nil {
		if _, err := s.db.Exec(`UPDATE bridge_attached_devices SET trust_state = ?, revoked_at = ? WHERE id = ?`, d.TrustState, d.RevokedAt.Format(time.RFC3339Nano), d.ID); err != nil {
			return Device{}, fmt.Errorf("persist attached device revocation: %w", err)
		}
	}
	s.devices[id] = d
	return d, nil
}
