package attached

import (
	"context"
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

type Service struct{ repo Repository }

func NewService() *Service { return &Service{repo: newMemoryRepository()} }

func NewServiceWithRepository(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Pair(ctx context.Context, in PairInput) (Device, error) {
	if strings.TrimSpace(in.HostNodeID) == "" || strings.TrimSpace(in.Kind) == "" {
		return Device{}, fmt.Errorf("host_node_id and kind are required")
	}
	now := time.Now().UTC()
	d := Device{ID: uuid.NewString(), Name: strings.TrimSpace(in.Name), HostNodeID: strings.TrimSpace(in.HostNodeID), Kind: strings.TrimSpace(in.Kind), Transport: strings.TrimSpace(in.Transport), Serial: strings.TrimSpace(in.Serial), OSVersion: strings.TrimSpace(in.OSVersion), TrustState: "trusted", Reachability: "reachable", CreatedAt: now}
	if !in.HostNodeOnline {
		d.Reachability = "unreachable"
		d.HealthReason = "host node " + d.HostNodeID + " is offline"
	}
	return s.repo.Create(ctx, d)
}

func (s *Service) List(ctx context.Context) []Device {
	items, err := s.repo.List(ctx)
	if err != nil {
		return nil
	}
	return items
}

func (s *Service) Revoke(ctx context.Context, id string) (Device, error) {
	if strings.TrimSpace(id) == "" {
		return Device{}, fmt.Errorf("attached device id is required")
	}
	return s.repo.Revoke(ctx, id, time.Now().UTC())
}
