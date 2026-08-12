package attached

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
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

type Repository interface {
	Create(context.Context, Device) (Device, error)
	List(context.Context) ([]Device, error)
	Get(context.Context, string) (Device, error)
	Revoke(context.Context, string, time.Time) (Device, error)
}

// Presence is the live node-channel view. Durable attached-device records keep
// trust and identity; reachability is projected from the owning node's current
// channel so a dead agent cannot appear usable merely because it was paired.
type Presence interface {
	IsOnline(nodeID string) bool
}

type Service struct {
	repo     Repository
	presence Presence
}

func NewService() *Service { return &Service{repo: newMemoryRepository()} }

func NewServiceWithRepository(repo Repository) *Service { return &Service{repo: repo} }

func NewServiceWithRepositoryAndPresence(repo Repository, live Presence) *Service {
	return &Service{repo: repo, presence: live}
}

func (s *Service) Pair(ctx context.Context, in PairInput) (Device, error) {
	if strings.TrimSpace(in.HostNodeID) == "" || strings.TrimSpace(in.Kind) == "" {
		return Device{}, fmt.Errorf("host_node_id and kind are required")
	}
	now := time.Now().UTC()
	d := Device{ID: stableID(in.Kind, in.Serial), Name: strings.TrimSpace(in.Name), HostNodeID: strings.TrimSpace(in.HostNodeID), Kind: strings.TrimSpace(in.Kind), Transport: strings.TrimSpace(in.Transport), Serial: strings.TrimSpace(in.Serial), OSVersion: strings.TrimSpace(in.OSVersion), TrustState: "trusted", Reachability: "reachable", CreatedAt: now}
	if !in.HostNodeOnline {
		d.Reachability = "unreachable"
		d.HealthReason = "host node " + d.HostNodeID + " is offline"
	}
	return s.repo.Create(ctx, d)
}

func stableID(kind, serial string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	serial = strings.TrimSpace(serial)
	if kind == "android" && serial != "" {
		digest := sha256.Sum256([]byte(serial))
		return "android-" + hex.EncodeToString(digest[:8])
	}
	return uuid.NewString()
}

func (s *Service) List(ctx context.Context) []Device {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil
	}
	if s.presence == nil {
		return items
	}
	for i := range items {
		if strings.TrimSpace(items[i].HostNodeID) == "" {
			continue
		}
		if s.presence.IsOnline(items[i].HostNodeID) {
			items[i].Reachability = "reachable"
			items[i].HealthReason = ""
			continue
		}
		items[i].Reachability = "unreachable"
		items[i].HealthReason = "host node " + items[i].HostNodeID + " is offline"
	}
	return items
}

func (s *Service) Revoke(ctx context.Context, id string) (Device, error) {
	if strings.TrimSpace(id) == "" {
		return Device{}, fmt.Errorf("attached device id is required")
	}
	return s.repo.Revoke(ctx, id, time.Now().UTC())
}
