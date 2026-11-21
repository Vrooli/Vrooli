package inventory

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo   Repository
	source DeviceDiscoverySource
	now    func() time.Time
}

type Config struct {
	Repo   Repository
	Source DeviceDiscoverySource
	Now    func() time.Time
}

func NewService(cfg Config) *Service {
	s := &Service{repo: cfg.Repo, source: cfg.Source, now: cfg.Now}
	if s.source == nil {
		s.source = ConservativeResolverSource{}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}
	return s
}

func (s *Service) Refresh(ctx context.Context, dryRun bool) ([]Device, []string, error) {
	observations, findings, err := s.source.Discover(ctx)
	if err != nil && err != ErrUnsupported {
		return nil, nil, err
	}
	if err == ErrUnsupported {
		devices, listErr := s.repo.ListDevices(ctx, "")
		if listErr != nil {
			return nil, nil, listErr
		}
		return devices, findings, nil
	}
	if dryRun {
		devices := make([]Device, 0, len(observations))
		for _, obs := range observations {
			devices = append(devices, s.deviceFromObservation(ctx, obs))
		}
		return devices, append(findings, "Dry run only; discovered devices were not persisted."), nil
	}
	for _, obs := range observations {
		device := s.deviceFromObservation(ctx, obs)
		if _, err := s.repo.SaveDevice(ctx, device); err != nil {
			return nil, nil, err
		}
	}
	devices, err := s.repo.ListDevices(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	return devices, findings, nil
}

func (s *Service) List(ctx context.Context, group string) ([]Device, error) {
	return s.repo.ListDevices(ctx, normalizeGroup(group))
}

func (s *Service) UpdateGroup(ctx context.Context, id, group string) (Device, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Device{}, fmt.Errorf("id is required")
	}
	return s.repo.UpdateGroup(ctx, id, normalizeGroup(group))
}

func (s *Service) Explain(ctx context.Context, id string) (Device, []string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Device{}, nil, fmt.Errorf("id is required")
	}
	device, err := s.repo.GetDevice(ctx, id)
	if err != nil {
		return Device{}, nil, err
	}
	evidence := []string{
		fmt.Sprintf("identity_confidence=%s", device.IdentityConfidence),
	}
	if device.StableID != "" {
		evidence = append(evidence, "stable identifier observed")
	}
	if device.ResolverClientID != "" {
		evidence = append(evidence, "resolver client identifier observed")
	}
	if device.MACAddress != "" {
		evidence = append(evidence, "MAC address observed")
	}
	if device.IPAddress != "" {
		evidence = append(evidence, "IP address observed")
	}
	evidence = append(evidence, device.Notes...)
	return device, evidence, nil
}

func (s *Service) deviceFromObservation(ctx context.Context, obs Observation) Device {
	now := s.now().UTC()
	obs = normalizeObservation(obs, now)
	device := Device{
		ID:               uuid.NewString(),
		Hostname:         obs.Hostname,
		IPAddress:        obs.IPAddress,
		MACAddress:       obs.MACAddress,
		StableID:         obs.StableID,
		ResolverClientID: obs.ResolverClientID,
		Group:            "unassigned",
		LastSeen:         obs.LastSeen,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	existing, matched, ambiguous := s.findExisting(ctx, obs)
	if matched {
		device.ID = existing.ID
		device.Group = existing.Group
		device.CreatedAt = existing.CreatedAt
		if device.Hostname == "" {
			device.Hostname = existing.Hostname
		} else if existing.Hostname != "" && !strings.EqualFold(existing.Hostname, device.Hostname) {
			device.Notes = append(device.Notes, fmt.Sprintf("hostname changed from %q to %q; previous hostname may be stale", existing.Hostname, device.Hostname))
		}
	}

	device.IdentityConfidence, device.Notes = identityAssessment(obs, matched, ambiguous, device.Notes)
	return device
}

func (s *Service) findExisting(ctx context.Context, obs Observation) (Device, bool, bool) {
	devices, err := s.repo.ListDevices(ctx, "")
	if err != nil {
		return Device{}, false, false
	}
	var ipMatches int
	var ipMatch Device
	for _, device := range devices {
		switch {
		case obs.StableID != "" && device.StableID == obs.StableID:
			return device, true, false
		case obs.ResolverClientID != "" && device.ResolverClientID == obs.ResolverClientID:
			return device, true, false
		case obs.MACAddress != "" && device.MACAddress == obs.MACAddress:
			return device, true, false
		}
		if obs.IPAddress != "" && device.IPAddress == obs.IPAddress {
			ipMatches++
			ipMatch = device
		}
	}
	if ipMatches == 1 && obs.StableID == "" && obs.ResolverClientID == "" && obs.MACAddress == "" {
		return ipMatch, true, true
	}
	return Device{}, false, ipMatches > 1 || weakObservation(obs)
}

func identityAssessment(obs Observation, matched, ambiguous bool, notes []string) (string, []string) {
	confidence := "low"
	if obs.StableID != "" || obs.ResolverClientID != "" || obs.MACAddress != "" {
		confidence = "medium"
	}
	if matched && (obs.StableID != "" || obs.ResolverClientID != "" || obs.MACAddress != "") {
		confidence = "high"
		notes = append(notes, "matched existing device by stable identity evidence")
	}
	if isRandomizedMAC(obs.MACAddress) {
		confidence = "low"
		notes = append(notes, "MAC address appears locally administered/randomized; identity may change across networks")
	}
	if ambiguous {
		confidence = "low"
		notes = append(notes, "identity is ambiguous because only weak or reused network evidence was available")
	}
	if obs.Hostname == "" {
		notes = append(notes, "hostname unavailable; keeping confidence conservative")
	}
	return confidence, dedupe(notes)
}

func normalizeObservation(obs Observation, now time.Time) Observation {
	obs.Source = strings.TrimSpace(obs.Source)
	obs.Hostname = strings.TrimSpace(obs.Hostname)
	obs.IPAddress = strings.TrimSpace(obs.IPAddress)
	obs.MACAddress = strings.ToLower(strings.TrimSpace(obs.MACAddress))
	obs.StableID = strings.TrimSpace(obs.StableID)
	obs.ResolverClientID = strings.TrimSpace(obs.ResolverClientID)
	if obs.LastSeen.IsZero() {
		obs.LastSeen = now
	}
	return obs
}

func normalizeGroup(group string) string {
	group = strings.TrimSpace(group)
	if group == "" {
		return "unassigned"
	}
	return group
}

func weakObservation(obs Observation) bool {
	return obs.StableID == "" && obs.ResolverClientID == "" && obs.MACAddress == ""
}

func isRandomizedMAC(mac string) bool {
	hw, err := net.ParseMAC(mac)
	if err != nil || len(hw) == 0 {
		return false
	}
	return hw[0]&0x02 == 0x02
}

func dedupe(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
