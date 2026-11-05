package attached

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrIdentityConflict is returned when a stable device identity is presented
// with a different host or after its prior trust grant was revoked. Pairing
// must never silently move an identity or resurrect authority held by an old
// operation.
var ErrIdentityConflict = errors.New("attached device identity conflicts with existing trust")

type Device struct {
	ID, Name, HostNodeID, Kind, Transport, Serial, OSVersion, TrustState, Reachability, HealthReason string
	Transports                                                                                       []string
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
	hostNodeID := strings.TrimSpace(in.HostNodeID)
	kind := strings.TrimSpace(in.Kind)
	if hostNodeID == "" || kind == "" {
		return Device{}, fmt.Errorf("host_node_id and kind are required")
	}
	now := time.Now().UTC()
	serial := strings.TrimSpace(in.Serial)
	id := stableID(kind, serial)
	if existing, err := s.repo.Get(ctx, id); err == nil {
		if existing.HostNodeID != hostNodeID {
			return Device{}, fmt.Errorf("%w: identity %q is bound to host %q, not %q", ErrIdentityConflict, id, existing.HostNodeID, hostNodeID)
		}
		if !existing.RevokedAt.IsZero() || strings.EqualFold(strings.TrimSpace(existing.TrustState), "revoked") {
			return Device{}, fmt.Errorf("%w: identity %q was revoked and must not be reactivated", ErrIdentityConflict, id)
		}
	}
	transport := strings.TrimSpace(in.Transport)
	d := Device{ID: id, Name: strings.TrimSpace(in.Name), HostNodeID: hostNodeID, Kind: kind, Transport: transport, Transports: normalizeTransports([]string{transport}), Serial: serial, OSVersion: strings.TrimSpace(in.OSVersion), TrustState: "trusted", Reachability: "reachable", CreatedAt: now}
	if !in.HostNodeOnline {
		d.Reachability = "unreachable"
		d.HealthReason = "host node " + d.HostNodeID + " is offline"
	}
	return s.repo.Create(ctx, d)
}

func normalizeTransports(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
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

// Get returns the durable trust state, including revoked records. List is
// intentionally active-only for inventory, while control paths need this
// exact read to fence an already-held device lease after revocation.
func (s *Service) Get(ctx context.Context, id string) (Device, error) {
	if strings.TrimSpace(id) == "" {
		return Device{}, fmt.Errorf("attached device id is required")
	}
	return s.repo.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) Revoke(ctx context.Context, id string) (Device, error) {
	if strings.TrimSpace(id) == "" {
		return Device{}, fmt.Errorf("attached device id is required")
	}
	return s.repo.Revoke(ctx, id, time.Now().UTC())
}
