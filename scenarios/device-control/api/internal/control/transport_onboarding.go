package control

import (
	"context"
	"fmt"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/strategy"
)

// PromoteWireless changes transport on an existing USB-onboarded identity
// only after the strategy verifies the same serial over its wireless endpoint.
func (s *Service) PromoteWireless(ctx context.Context, id string) (Device, error) {
	unlockInventory, err := s.lockInventory(ctx)
	if err != nil {
		return Device{}, err
	}
	defer unlockInventory()
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.devices.Get(strings.TrimSpace(id))
	if !ok {
		return Device{}, fmt.Errorf("unknown device %q", id)
	}
	if record.Transport != "usb" {
		hasKnownUSB := false
		for _, profile := range record.Transports {
			if profile.Name == "usb" && profile.StrategyID == record.StrategyID && record.Serial != "" {
				hasKnownUSB = true
				break
			}
		}
		if !hasKnownUSB {
			return Device{}, fmt.Errorf("device %q must have a known serial-bound USB transport before wireless promotion", id)
		}
	}
	base, ok := s.registry.Get(record.StrategyID)
	if !ok {
		return Device{}, fmt.Errorf("strategy %q is unavailable", record.StrategyID)
	}
	if scoped, ok := base.(strategy.DeviceScoped); ok && record.Serial != "" {
		base = scoped.ForDevice(record.Serial)
	} else if record.Transport != "usb" {
		return Device{}, fmt.Errorf("device %q cannot re-promote without a serial-scoped USB strategy", id)
	}
	promoter, ok := base.(interface{ PromoteWireless(context.Context) error })
	if !ok {
		return Device{}, fmt.Errorf("strategy %q does not support wireless transport", record.StrategyID)
	}
	if err := promoter.PromoteWireless(ctx); err != nil {
		return Device{}, err
	}
	endpointProvider, ok := base.(interface{ WirelessEndpoint() string })
	if !ok || strings.TrimSpace(endpointProvider.WirelessEndpoint()) == "" {
		return Device{}, fmt.Errorf("strategy %q did not expose a verified wireless endpoint", record.StrategyID)
	}
	state := transportState{DeviceID: id, Serial: record.Serial, StrategyID: record.StrategyID, Transport: "wireless", Endpoint: strings.TrimSpace(endpointProvider.WirelessEndpoint()), UpdatedAt: time.Now().UTC()}
	if err := s.persistTransportState(ctx, state); err != nil {
		return Device{}, err
	}
	s.transportStrategies[id] = base
	s.transportStates[id] = state
	record.Transport = "wireless"
	record.Endpoint = state.Endpoint
	record.ObservedAt = state.UpdatedAt
	record.Status = strategy.StatusAvailable
	record.Health = strategy.StatusAvailable
	record.HealthReason = "wireless ADB transport verified against the onboarded serial"
	return deviceFromRecord(s.devices.UpsertIdentity(record)), nil
}

// OnboardNetworkADB adopts a directly-addressable Android ADB endpoint (classic
// TCP ADB on port 5555 for an Android TV or TV box, or Wireless Debugging/TLS)
// as a first-class governed device. It runs the strategy's verified connect,
// derives the stable identity from the hardware serial, probes live
// capabilities, and persists an endpoint-bound transport so subsequent
// state/app-lifecycle/actuation verbs route through the same profile. The
// network I/O runs before the inventory lock is taken.
func (s *Service) OnboardNetworkADB(ctx context.Context, endpoint string) (Device, error) {
	endpoint = strings.TrimSpace(endpoint)
	base, ok := s.registry.Get("android-adb")
	if !ok {
		return Device{}, fmt.Errorf("android-adb strategy is unavailable")
	}
	factory, ok := base.(interface {
		NewNetworkTransport() strategy.Strategy
	})
	if !ok {
		return Device{}, fmt.Errorf("android-adb strategy does not support network onboarding")
	}
	adapter := factory.NewNetworkTransport()
	onboarder, ok := adapter.(interface {
		ConnectNetwork(context.Context, string) (strategy.Device, error)
	})
	if !ok {
		return Device{}, fmt.Errorf("android-adb strategy does not support network onboarding")
	}
	discovered, err := onboarder.ConnectNetwork(ctx, endpoint)
	if err != nil {
		return Device{}, err
	}
	declaration, err := adapter.Describe(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("probe onboarded endpoint capabilities: %w", err)
	}

	unlockInventory, err := s.lockInventory(ctx)
	if err != nil {
		return Device{}, err
	}
	defer unlockInventory()
	s.mu.Lock()
	defer s.mu.Unlock()
	identityKey := discovered.IdentityKey
	if identityKey == "" {
		identityKey = discovered.Serial
	}
	record := devicedomain.Record{
		ID: discovered.ID, IdentityKey: identityKey, IdentityKind: discovered.IdentityKind,
		Claims: claimsForDevice(discovered), Name: discovered.Model, Kind: deviceKind(discovered),
		Serial: discovered.Serial, Endpoint: discovered.Endpoint, Model: discovered.Model,
		OSVersion: discovered.OSVersion, StrategyID: discovered.StrategyID,
		Status: strategy.StatusAvailable, Health: strategy.StatusAvailable,
		HealthReason: "network ADB endpoint verified against the device serial",
		Transport:    discovered.Transport, ObservedAt: time.Now().UTC(),
		Capabilities: mapCaps(declaration),
		Operations:   append([]string(nil), declaration.Operations...),
		Properties:   append([]strategy.PropertyDescriptor(nil), declaration.Properties...),
	}
	if record.Name == "" {
		record.Name = record.Serial
	}
	merged := s.devices.UpsertIdentity(record)
	state := transportState{DeviceID: merged.ID, Serial: merged.Serial, StrategyID: merged.StrategyID, Transport: discovered.Transport, Endpoint: discovered.Endpoint, UpdatedAt: time.Now().UTC()}
	if err := s.persistTransportState(ctx, state); err != nil {
		return Device{}, err
	}
	s.transportStrategies[merged.ID] = adapter
	s.transportStates[merged.ID] = state
	_ = s.persistObservedTransportProfiles(ctx, merged)
	_ = s.persistIdentityClaims(ctx, merged)
	s.startObserverLocked(merged)
	return deviceFromRecord(merged), nil
}

// ReconnectWireless revalidates a promoted wireless transport and allows the
// Android adapter to replace a stale endpoint through authenticated mDNS
// discovery. The stable device identity remains bound to the onboarded serial.
func (s *Service) ReconnectWireless(ctx context.Context, id string) (Device, error) {
	id = strings.TrimSpace(id)
	unlockInventory, err := s.lockInventory(ctx)
	if err != nil {
		return Device{}, err
	}
	defer unlockInventory()
	s.mu.Lock()
	s.restoreTransportStrategies()
	record, ok := s.devices.Get(strings.TrimSpace(id))
	adapter, hasAdapter := s.transportStrategies[id]
	s.mu.Unlock()
	if !ok {
		return Device{}, fmt.Errorf("unknown device %q", id)
	}
	if record.Transport != "wireless" || !hasAdapter {
		return Device{}, fmt.Errorf("device %q has no promoted wireless transport", id)
	}
	reconnector, ok := adapter.(strategy.WirelessReconnector)
	if !ok {
		return Device{}, fmt.Errorf("strategy %q does not support wireless reconnect", record.StrategyID)
	}
	if err := reconnector.ReconnectWireless(ctx); err != nil {
		return Device{}, err
	}
	endpointProvider, ok := adapter.(interface{ WirelessEndpoint() string })
	if !ok || strings.TrimSpace(endpointProvider.WirelessEndpoint()) == "" {
		return Device{}, fmt.Errorf("strategy %q did not expose a verified wireless endpoint", record.StrategyID)
	}
	// Reconnect proves identity and reachability. Probe capabilities on that
	// verified endpoint instead of carrying forward a stale capability set.
	declaration, err := adapter.Describe(ctx)
	if err != nil {
		return Device{}, fmt.Errorf("probe reconnected endpoint: %w", err)
	}
	if declaration.Status != strategy.StatusAvailable {
		return Device{}, fmt.Errorf("reconnected endpoint is not available: %s", declaration.Status)
	}
	state := transportState{DeviceID: id, Serial: record.Serial, StrategyID: record.StrategyID, Transport: "wireless", Endpoint: strings.TrimSpace(endpointProvider.WirelessEndpoint()), UpdatedAt: time.Now().UTC()}
	if err := s.persistTransportState(ctx, state); err != nil {
		return Device{}, err
	}
	s.mu.Lock()
	s.transportStates[id] = state
	record.Status = strategy.StatusAvailable
	record.Health = strategy.StatusAvailable
	record.HealthReason = "wireless ADB endpoint reconnected and identity verified"
	record.Endpoint = state.Endpoint
	record.ObservedAt = state.UpdatedAt
	record.Capabilities = mapCaps(declaration)
	record.Operations = append([]string(nil), declaration.Operations...)
	record.Properties = append([]strategy.PropertyDescriptor(nil), declaration.Properties...)
	updated := s.devices.UpsertIdentity(record)
	s.mu.Unlock()
	return deviceFromRecord(updated), nil
}
