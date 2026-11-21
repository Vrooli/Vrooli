package control

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	devicedomain "device-control/internal/devices"
	"device-control/internal/identity"
	"device-control/strategy"
)

const defaultInventoryTimeout = 10 * time.Second

func (s *Service) Strategies(ctx context.Context) []strategy.Declaration { return s.registry.List(ctx) }

// CachedDevices returns the most recently observed inventory without probing
// every transport. The dashboard uses this snapshot for its first paint so a
// slow or unavailable device adapter cannot hold the operator shell hostage;
// explicit operator refreshes still call Devices and update the snapshot.
func (s *Service) CachedDevices() []Device {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Device, 0)
	for _, item := range s.devices.ListActionable(time.Now().UTC(), 15*time.Minute) {
		// Restored rows are intentionally marked unreachable until a fresh
		// transport probe completes. Do not present those historical rows as
		// live inventory during the dashboard's first paint.
		if item.Status != strategy.StatusAvailable && item.Health != strategy.StatusAvailable {
			continue
		}
		out = append(out, deviceFromRecord(item))
	}
	return out
}

func (s *Service) Verify(ctx context.Context, id string) (strategy.ConformanceReport, error) {
	return s.registry.Verify(ctx, id)
}

// Devices holds the inventory gate throughout a refresh. Release only the general
// service lock around external calls, restoring it even if a provider panics.
func (s *Service) inventoryProbe(probe func()) {
	s.mu.Unlock()
	defer s.mu.Lock()
	probe()
}

type baseEnumeration struct {
	done    chan struct{}
	devices []strategy.Device
	err     error
}

type deviceDescription struct {
	done        chan struct{}
	declaration strategy.Declaration
	err         error
}

func unavailableDeviceDeclaration(base strategy.Declaration) strategy.Declaration {
	caps := make([]strategy.Capability, 0, len(base.Capabilities))
	for name, capability := range base.Capabilities {
		caps = append(caps, strategy.Capability{Name: name, Status: strategy.StatusUnavailable, Reason: "device capability probe failed", NextAction: "Retry the device capability probe", StateClass: capability.StateClass})
	}
	return strategy.UnavailableDeclaration(base.StrategyID, "device capability probe failed", caps, "Retry the device capability probe")
}

func unavailableCapabilities(capabilities []strategy.Capability, reason string) []strategy.Capability {
	result := make([]strategy.Capability, len(capabilities))
	for index, capability := range capabilities {
		capability.Status = strategy.StatusUnavailable
		capability.Reason = reason
		capability.NextAction = "Refresh Device Control transport inventory"
		result[index] = capability
	}
	return result
}

func routeGapTransports(transports []strategy.DeviceTransport, reason string) []strategy.DeviceTransport {
	result := make([]strategy.DeviceTransport, len(transports))
	for index, transport := range transports {
		transport.Health = strategy.StatusUnavailable
		transport.HealthReason = reason
		transport.Endpoint = ""
		transport.Endpoints = nil
		for name, capability := range transport.Capabilities {
			capability.Status = strategy.StatusUnavailable
			capability.Reason = reason
			capability.NextAction = "Refresh Device Control transport inventory"
			transport.Capabilities[name] = capability
		}
		result[index] = transport
	}
	return result
}

// Each discovered serial has an independent capability profile. Start probes
// together while retaining the original device merge order.
func startDeviceDescriptions(ctx context.Context, scoped strategy.DeviceScoped, devices []strategy.Device, promotedEndpoints map[string]string, limit chan struct{}) []*deviceDescription {
	results := make([]*deviceDescription, len(devices))
	for index, device := range devices {
		if strings.TrimSpace(device.Serial) == "" {
			continue
		}
		if _, duplicate := promotedEndpoints[strings.TrimSpace(device.Serial)]; duplicate {
			continue
		}
		result := &deviceDescription{done: make(chan struct{})}
		results[index] = result
		go func(result *deviceDescription, serial string) {
			defer close(result.done)
			defer func() {
				if recover() != nil {
					result.err = fmt.Errorf("device capability probe panicked")
				}
			}()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				result.err = ctx.Err()
				return
			}
			if err := ctx.Err(); err != nil {
				result.err = err
				return
			}
			probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			defer cancel()
			result.declaration, result.err = scoped.ForDevice(serial).Describe(probeCtx)
		}(result, device.Serial)
	}
	return results
}

// Probe independent base transports concurrently, then merge their results in
// declaration order. Endpoint-bound promoted transports are probed separately.
func (s *Service) enumerateBaseStrategies(ctx context.Context, declarations []strategy.Declaration, limit chan struct{}) []*baseEnumeration {
	results := make([]*baseEnumeration, len(declarations))
	for index, declaration := range declarations {
		base, ok := s.registry.Get(declaration.StrategyID)
		if !ok {
			continue
		}
		enumerator, ok := base.(strategy.Enumerator)
		if !ok {
			continue
		}
		result := &baseEnumeration{done: make(chan struct{})}
		results[index] = result
		go func(result *baseEnumeration, enumerator strategy.Enumerator) {
			defer close(result.done)
			defer func() {
				if recover() != nil {
					result.err = fmt.Errorf("strategy enumeration panicked")
				}
			}()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				result.err = ctx.Err()
				return
			}
			if err := ctx.Err(); err != nil {
				result.err = err
				return
			}
			result.devices, result.err = enumerator.Enumerate(ctx)
		}(result, enumerator)
	}
	return results
}

// Endpoint-bound adapters have independent probes; starting them together avoids
// spending the entire inventory deadline on whichever device happens to come first.
type promotedDiscovery struct {
	done         chan struct{}
	declaration  strategy.Declaration
	describeErr  error
	devices      []strategy.Device
	enumerateErr error
}

// startPromotedDiscoveries probes every endpoint-bound adapter concurrently.
// An adapter whose endpoint is absent from the adb server (the TV slept, adb was
// restarted) is reconnected once when gate allows, so a durable network
// identity heals instead of lingering as "not present" until an operator
// reruns onboarding.
func startPromotedDiscoveries(ctx context.Context, adapters map[string]strategy.Strategy, limit chan struct{}, gate *reconnectGate) map[string]*promotedDiscovery {
	results := make(map[string]*promotedDiscovery, len(adapters))
	for id, adapter := range adapters {
		result := &promotedDiscovery{done: make(chan struct{})}
		results[id] = result
		go func(id string, adapter strategy.Strategy, result *promotedDiscovery) {
			defer close(result.done)
			defer func() {
				if recover() != nil {
					result.describeErr = fmt.Errorf("promoted device probe panicked")
					result.enumerateErr = result.describeErr
				}
			}()
			select {
			case limit <- struct{}{}:
				defer func() { <-limit }()
			case <-ctx.Done():
				result.describeErr, result.enumerateErr = ctx.Err(), ctx.Err()
				return
			}
			if err := ctx.Err(); err != nil {
				result.describeErr, result.enumerateErr = err, err
				return
			}
			if enumerator, ok := adapter.(strategy.Enumerator); ok {
				result.devices, result.enumerateErr = enumerator.Enumerate(ctx)
				if result.enumerateErr == nil && len(result.devices) == 0 {
					if reconnector, ok := adapter.(strategy.WirelessReconnector); ok && gate.allow(id, time.Now()) {
						reconnectCtx, cancelReconnect := context.WithTimeout(ctx, 15*time.Second)
						reconnectErr := reconnector.ReconnectWireless(reconnectCtx)
						cancelReconnect()
						gate.record(id, time.Now(), reconnectErr == nil)
						if reconnectErr == nil {
							result.devices, result.enumerateErr = enumerator.Enumerate(ctx)
						}
					}
				}
			}
			probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			result.declaration, result.describeErr = adapter.Describe(probeCtx)
			cancel()
		}(id, adapter, result)
	}
	return results
}

func (s *Service) Devices(ctx context.Context) []Device {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := s.inventoryTimeout
	if timeout <= 0 {
		timeout = defaultInventoryTimeout
	}
	inventoryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	unlockInventory, err := s.lockInventory(inventoryCtx)
	if err != nil {
		return nil
	}
	defer unlockInventory()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restoreTransportStrategies()
	var declarations []strategy.Declaration
	s.inventoryProbe(func() { declarations = s.registry.List(inventoryCtx) })
	limit := make(chan struct{}, 8)
	adapters := make(map[string]strategy.Strategy, len(s.transportStrategies))
	for id, adapter := range s.transportStrategies {
		if _, exists := s.devices.Get(id); exists && adapter != nil {
			adapters[id] = adapter
		}
	}
	var baseResults []*baseEnumeration
	var promotedResults map[string]*promotedDiscovery
	s.inventoryProbe(func() {
		baseResults = s.enumerateBaseStrategies(inventoryCtx, declarations, limit)
		promotedResults = startPromotedDiscoveries(inventoryCtx, adapters, limit, s.reconnects)
	})
	declarationByID := make(map[string]strategy.Declaration, len(declarations))
	for _, declaration := range declarations {
		declarationByID[declaration.StrategyID] = declaration
	}
	hostNodeID := s.hostNodeID
	seen := map[string]bool{}
	promotedEndpoints := make(map[string]string, len(s.transportStates))
	for id, state := range s.transportStates {
		if endpoint := strings.TrimSpace(state.Endpoint); endpoint != "" {
			promotedEndpoints[endpoint] = id
		}
	}
	for declarationIndex, d := range declarations {
		enumerating := false
		if base, ok := s.registry.Get(d.StrategyID); ok {
			// Enumerate the unscoped adapter as well as any promoted transport
			// adapters. A promoted wireless phone must stay endpoint-bound for
			// its verbs, but it must not hide a concurrently running emulator
			// from the host inventory.
			type inventoryStrategy struct {
				strategy strategy.Strategy
				promoted bool
				deviceID string
			}
			items := []inventoryStrategy{{strategy: base}}
			for deviceID, promoted := range s.transportStrategies {
				if device, exists := s.devices.Get(deviceID); exists && device.StrategyID == d.StrategyID && promoted != nil {
					items = append(items, inventoryStrategy{strategy: promoted, promoted: true, deviceID: deviceID})
				}
			}
			for _, candidate := range items {
				item := candidate.strategy
				// A promoted wireless adapter is endpoint-bound. Its capability
				// declaration can differ from the base adapter, so use the
				// effective declaration for the devices it enumerates.
				declaration := d
				// The registry already described the base adapter. Endpoint-bound
				// promoted adapters need their own independent capability probe.
				if candidate.promoted {
					result := promotedResults[candidate.deviceID]
					s.inventoryProbe(func() { <-result.done })
					if result.describeErr == nil {
						declaration = result.declaration
					} else {
						declaration = unavailableDeviceDeclaration(d)
					}
				}
				if _, ok := item.(strategy.Enumerator); ok {
					enumerating = true
					var discovered []strategy.Device
					var enumerateErr error
					if candidate.promoted {
						result := promotedResults[candidate.deviceID]
						discovered, enumerateErr = result.devices, result.enumerateErr
					} else {
						result := baseResults[declarationIndex]
						s.inventoryProbe(func() { <-result.done })
						discovered, enumerateErr = result.devices, result.err
					}
					if enumerateErr == nil {
						var deviceResults []*deviceDescription
						if scoped, scopedOK := item.(strategy.DeviceScoped); scopedOK && !candidate.promoted {
							s.inventoryProbe(func() {
								deviceResults = startDeviceDescriptions(inventoryCtx, scoped, discovered, promotedEndpoints, limit)
							})
						}
						for deviceIndex, discoveredDevice := range discovered {
							if !candidate.promoted {
								if promotedID, duplicate := promotedEndpoints[strings.TrimSpace(discoveredDevice.Serial)]; duplicate {
									seen[promotedID] = true
									continue
								}
							}
							deviceDeclaration := declaration
							// An ADB strategy can enumerate several devices, but its
							// declaration probes whichever serial the adapter owns. Use
							// a device-scoped adapter for the unpromoted inventory so
							// emulator capabilities are not inherited from a phone (or
							// vice versa). Promoted transports retain their endpoint-
							// bound declaration above.
							if deviceResults != nil && deviceResults[deviceIndex] != nil {
								result := deviceResults[deviceIndex]
								s.inventoryProbe(func() { <-result.done })
								if result.err == nil {
									deviceDeclaration = result.declaration
								} else {
									deviceDeclaration = unavailableDeviceDeclaration(declaration)
								}
							}
							identityKey := discoveredDevice.IdentityKey
							if identityKey == "" && discoveredDevice.StrategyID == "android-adb" {
								identityKey = discoveredDevice.Serial
							}
							// Locally enumerated devices are owned by this device-control
							// process. HostNodeID is reserved for Bridge-attached
							// peripherals; populating it here makes the lease fence ask
							// Bridge to authorize a local TV that Bridge never registered.
							record := devicedomain.Record{ID: discoveredDevice.ID, IdentityKey: identityKey, IdentityKind: discoveredDevice.IdentityKind, Claims: claimsForDevice(discoveredDevice), Name: discoveredDevice.Model, Kind: deviceKind(discoveredDevice), Serial: discoveredDevice.Serial, Endpoint: discoveredDevice.Endpoint, Model: discoveredDevice.Model, OSVersion: discoveredDevice.OSVersion, StrategyID: discoveredDevice.StrategyID, Status: discoveredDevice.Health, Health: discoveredDevice.Health, HealthReason: discoveredDevice.HealthReason, Transport: discoveredDevice.Transport, ObservedAt: discoveredDevice.ObservedAt, Capabilities: mapCaps(deviceDeclaration), Operations: append([]string(nil), deviceDeclaration.Operations...), Properties: append([]strategy.PropertyDescriptor(nil), deviceDeclaration.Properties...)}
							if record.Name == "" {
								record.Name = record.Serial
							}
							merged := s.devices.UpsertIdentity(record)
							_ = s.persistObservedTransportProfiles(inventoryCtx, merged)
							_ = s.persistIdentityClaims(inventoryCtx, merged)
							s.startObserverLocked(merged)
							seen[merged.ID] = true
						}
					} else {
						// Enumeration is part of the device contract. Preserve a
						// diagnostic row instead of silently dropping the transport.
						// The synthetic id is transport-scoped and is replaced by a
						// hardware identity as soon as a browse succeeds.
						reason := fmt.Sprintf("%s enumeration failed: %v", d.StrategyID, enumerateErr)
						id := "unreachable:" + d.StrategyID
						merged := s.devices.Upsert(devicedomain.Record{ID: id, IdentityKey: "", Name: d.Description, Kind: "physical", StrategyID: d.StrategyID, Transport: d.StrategyID, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: reason, HostNodeID: hostNodeID, Capabilities: mapCaps(declaration), ObservedAt: time.Now().UTC()})
						seen[merged.ID] = true
					}
				}
			}
		}
		if enumerating {
			continue
		}
		// Strategies are exposed by /strategies. The device inventory contains
		// physical identities (and bridge-attached identities) only; a strategy
		// row is not a device and must not be presented as one.
		_ = d
	}
	if len(seen) == 0 {
		seen = map[string]bool{}
	}
	if inventoryCtx.Err() == nil {
		s.devices.MarkAbsentExcept(time.Now().UTC(), seen, func(item devicedomain.Record) string {
			if item.HostNodeID != "" {
				return "device not present on host node " + item.HostNodeID
			}
			return "device not present on host node local"
		})
	}
	out := make([]Device, 0)
	for _, item := range s.devices.ListActionable(time.Now().UTC(), 15*time.Minute) {
		if seen[item.ID] || item.Kind == "physical" || item.Kind == "emulator" || item.Kind == "desktop" {
			out = append(out, deviceFromRecord(item))
		}
	}
	if s.attached != nil {
		reader := s.attached
		var attached []AttachedDevice
		var err error
		s.inventoryProbe(func() { attached, err = reader.List(inventoryCtx) })
		if err != nil {
			s.bridgeInventoryErr = fmt.Sprintf("list Bridge attached devices: %v", err)
			// A failed bridge lookup must not manufacture a pseudo-device beside
			// locally enumerated physical devices. Consumers expect one row per
			// real device. Keep the failure in the diagnostics projection so a
			// healthy local device cannot hide a broken federation edge.
			if len(out) == 0 {
				out = append(out, Device{ID: "bridge", Name: "Bridge attached-device registry", Kind: "bridge", Status: strategy.StatusUnavailable, Health: "unreachable", HealthReason: "bridge host node is unavailable", Capabilities: make([]strategy.Capability, 0), ObservedAt: time.Now().UTC()})
			}
		} else {
			s.bridgeInventoryErr = ""
			byID := make(map[string]int, len(out))
			bySerial := make(map[string]int, len(out))
			for i, device := range out {
				if device.Kind != "physical" {
					continue
				}
				byID[device.ID] = i
				if device.Serial != "" {
					bySerial[device.Serial] = i
				}
			}
			for _, d := range attached {
				const routeGap = currentRouteObservationGap + "; attached-device route unavailable"
				index, found := byID[d.ID]
				if !found && d.Serial != "" {
					index, found = bySerial[d.Serial]
				}
				if found {
					device := &out[index]
					if device.StrategyID == "" && strings.EqualFold(d.Kind, "android") {
						device.StrategyID = "android-adb"
					}
					if d.Name != "" {
						device.Name = d.Name
						if device.Model == "" {
							device.Model = d.Name
						}
					}
					if d.OSVersion != "" {
						device.OSVersion = d.OSVersion
					}
					if d.Transport != "" {
						device.Transport = d.Transport
					}
					if d.HostNodeID != "" {
						device.HostNodeID = d.HostNodeID
					}
					if len(device.Capabilities) == 0 {
						if declaration, ok := declarationByID[device.StrategyID]; ok {
							device.Capabilities = mapCaps(declaration)
						}
					}
					// Bridge trust and host-channel presence establish an attached identity,
					// not a route this Device Control process can operate. Only an exact
					// identity seen by the current local enumeration owns readiness.
					if !seen[device.ID] {
						device.Status, device.Health = strategy.StatusUnavailable, "degraded"
						device.HealthReason = routeGap
						device.Capabilities = unavailableCapabilities(device.Capabilities, routeGap)
						device.Endpoint = ""
						device.Transports = routeGapTransports(device.Transports, routeGap)
					}
					merged := s.devices.UpsertIdentity(recordFromDevice(*device))
					_ = s.persistObservedTransportProfiles(inventoryCtx, merged)
					_ = s.persistIdentityClaims(inventoryCtx, merged)
					s.startObserverLocked(merged)
					continue
				}
				byID[d.ID] = len(out)
				if d.Serial != "" {
					bySerial[d.Serial] = len(out)
				}
				strategyID := ""
				if strings.EqualFold(d.Kind, "android") {
					strategyID = "android-adb"
				}
				capabilities := make([]strategy.Capability, 0)
				if declaration, ok := declarationByID[strategyID]; ok {
					capabilities = mapCaps(declaration)
				}
				// Bridge's attached-device contract carries the platform kind
				// (currently "android"), while device-control's inventory kind
				// answers whether the row is a physical target. Keep those
				// concepts separate: every bridge peripheral is physical, and
				// the strategy id carries the platform-specific driver.
				merged := s.devices.UpsertIdentity(devicedomain.Record{ID: d.ID, IdentityKey: d.Serial, Name: d.Name, Kind: "physical", Serial: d.Serial, Model: d.Name, OSVersion: d.OSVersion, StrategyID: strategyID, Transport: d.Transport, HostNodeID: d.HostNodeID, Status: strategy.StatusUnavailable, Health: "degraded", HealthReason: routeGap, Capabilities: unavailableCapabilities(capabilities, routeGap), ObservedAt: time.Now().UTC()})
				_ = s.persistObservedTransportProfiles(inventoryCtx, merged)
				_ = s.persistIdentityClaims(inventoryCtx, merged)
				s.startObserverLocked(merged)
				out = append(out, deviceFromRecord(merged))
			}
		}
	}
	// ADB and the bridge can both report the USB endpoint during the handoff.
	// Promotion owns the identity transition, so restore its selected
	// transport after all merges rather than allowing a USB observation to
	// overwrite the stable record on the next refresh.
	for i := range out {
		if _, promoted := s.transportStrategies[out[i].ID]; !promoted {
			continue
		}
		out[i].Transport = "wireless"
		// Do not turn a retained, unreachable identity into a healthy
		// device merely because its wireless adapter was restored. The
		// enumerator owns the fresh reachability result; this pass only
		// preserves the selected transport across USB/bridge merges.
		if out[i].Health == strategy.StatusAvailable {
			out[i].HealthReason = "wireless ADB endpoint reachable and identity verified"
		}
		s.devices.UpsertIdentity(recordFromDevice(out[i]))
	}
	return out
}

func claimsForDevice(device strategy.Device) []identity.IdentityClaim {
	kind := strings.TrimSpace(device.IdentityKind)
	if kind == "" && device.StrategyID == "android-adb" {
		kind = string(identity.ADBSerial)
	}
	value := strings.TrimSpace(device.IdentityKey)
	if value == "" && kind == string(identity.ADBSerial) {
		value = strings.TrimSpace(device.Serial)
	}
	if kind == "" || value == "" {
		return nil
	}
	claim, err := identity.NewClaim(kind, value, device.StrategyID, "observed")
	if err != nil {
		return nil
	}
	return []identity.IdentityClaim{claim}
}

// DescribeDevice returns a known identity without requiring discovery of every
// other device. Historical observations remain historical; this read does not
// manufacture a new reachability or capability timestamp.
func (s *Service) DescribeDevice(ctx context.Context, id string) (Device, error) {
	id = strings.TrimSpace(id)
	if record, found := s.devices.Get(id); found {
		return deviceFromRecord(record), nil
	}
	// Discover only when the identity has never been observed locally.
	for _, device := range s.Devices(ctx) {
		if device.ID == id {
			return device, nil
		}
	}
	return Device{}, fmt.Errorf("unknown device %q", id)
}

// CorrelationCandidates is a diagnostic-only projection. Candidates are
// deliberately not folded into durable identity until an owner confirms a
// hardware claim through MergeDevices.
func (s *Service) CorrelationCandidates(ctx context.Context) []devicedomain.CorrelationCandidate {
	_ = s.Devices(ctx)
	return s.devices.CorrelationCandidates(time.Now().UTC())
}

func (s *Service) DiagnosticDevices(ctx context.Context) []Device {
	_ = s.Devices(ctx)
	records := s.devices.Diagnostics(time.Now().UTC(), 15*time.Minute)
	out := make([]Device, 0, len(records))
	for _, record := range records {
		out = append(out, deviceFromRecord(record))
	}
	s.mu.Lock()
	bridgeErr := s.bridgeInventoryErr
	s.mu.Unlock()
	if bridgeErr != "" {
		out = append(out, Device{ID: "bridge-inventory", Name: "Bridge attached-device registry", Kind: "bridge", Status: strategy.StatusUnavailable, Health: strategy.HealthUnreachable, HealthReason: bridgeErr, Capabilities: make([]strategy.Capability, 0), ObservedAt: time.Now().UTC()})
	}
	return out
}

func adbDeviceKind(serial string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(serial)), "emulator-") {
		return "emulator"
	}
	return "physical"
}

func deviceKind(device strategy.Device) string {
	if strings.EqualFold(strings.TrimSpace(device.StrategyID), "host-desktop") {
		return "desktop"
	}
	return adbDeviceKind(device.Serial)
}

// ForgetDevice removes a retained identity only when the owner explicitly
// requests it. Device discovery never calls this path, preserving stable ids
// across disconnects and adb-server restarts.
func (s *Service) ForgetDevice(id string) bool {
	return s.ForgetDeviceContext(context.Background(), id)
}

func (s *Service) ForgetDeviceContext(ctx context.Context, id string) bool {
	unlockInventory, err := s.lockInventory(ctx)
	if err != nil {
		return false
	}
	defer unlockInventory()
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	forgotten := s.devices.Forget(id)
	delete(s.transportStrategies, id)
	if job := s.observerCancels[id]; job != nil {
		job.stopping = true
		job.cancel()
	}
	delete(s.transportStates, id)
	for key, state := range s.transportProfiles {
		if state.DeviceID == id {
			delete(s.transportProfiles, key)
		}
	}
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transports WHERE device_id = ?`, id)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_transport_profiles WHERE device_id = ?`, id)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_claims WHERE device_id = ?`, id)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_merges WHERE canonical_id = ? OR member_id = ?`, id, id)
		_, _ = s.db.ExecContext(ctx, `DELETE FROM device_control_identity_aliases WHERE canonical_id = ? OR alias_id = ?`, id, id)
	}
	delete(s.auditAliases, id)
	for canonical, aliases := range s.auditAliases {
		delete(aliases, id)
		if len(aliases) == 0 {
			delete(s.auditAliases, canonical)
		}
	}
	return forgotten
}

func mapCaps(d strategy.Declaration) []strategy.Capability {
	names := make([]string, 0, len(d.Capabilities))
	for n := range d.Capabilities {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]strategy.Capability, 0, len(names))
	for _, n := range names {
		capability := d.Capabilities[n]
		capability.Name = n
		out = append(out, capability)
	}
	return out
}

func recordFromDevice(device Device) devicedomain.Record {
	capabilities := make([]strategy.Capability, len(device.Capabilities))
	copy(capabilities, device.Capabilities)
	return devicedomain.Record{
		ID: device.ID, IdentityKey: device.IdentityKey, Claims: append([]identity.IdentityClaim(nil), device.Claims...), IdentityReason: device.IdentityReason, Name: device.Name, Kind: device.Kind, Serial: device.Serial,
		Model: device.Model, OSVersion: device.OSVersion, StrategyID: device.StrategyID,
		Status: device.Status, Health: device.Health, HealthReason: device.HealthReason,
		HostNodeID: device.HostNodeID, Transport: device.Transport, Capabilities: capabilities,
		Transports: append([]strategy.DeviceTransport(nil), device.Transports...),
		Endpoint:   device.Endpoint,
		ObservedAt: device.ObservedAt, FirstSeenAt: device.FirstSeenAt, LastSeenAt: device.LastSeenAt,
	}
}
