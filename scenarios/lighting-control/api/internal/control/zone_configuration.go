package control

import (
	"context"
	"errors"
	"fmt"
	"strings"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/types/known/emptypb"
)

var ErrZoneSetupNeeded = errors.New("zone setup needed: configure the connected LED count before rendering")

type configuredZoneBackend interface {
	ApplyConfiguredStableZone(context.Context, string, string, uint32, map[string]uint32, *controlv1.ZoneState) (SDKCommandResult, error)
}

// SDK v6 normal size flag is 1<<1. Effects-only sizes do not establish a
// controllable per-LED buffer and therefore do not enable this operation.
func configurableZone(z SDKZone) bool {
	return z.Flags&2 != 0 && z.LEDMin < z.LEDMax && z.LEDMax <= 65535
}
func configurationMin(z SDKZone) uint32 {
	if configurableZone(z) {
		return z.LEDMin
	}
	return 0
}
func configurationMax(z SDKZone) uint32 {
	if configurableZone(z) {
		return z.LEDMax
	}
	return 0
}
func validateZoneLength(z SDKZone, count uint32) error {
	if !configurableZone(z) {
		return ErrUnsupported
	}
	if count == 0 || count < z.LEDMin || count > z.LEDMax {
		return fmt.Errorf("%w: LED count must be between %d and %d (and greater than zero)", ErrInvalidState, z.LEDMin, z.LEDMax)
	}
	return nil
}
func mappingCapability(z ZoneMapping) *controlv1.ZoneCapability {
	configurable := z.LEDMax > 0
	return &controlv1.ZoneCapability{ZoneId: z.ZoneID, Name: z.Name, LedCount: z.LEDCount, SupportsStaticColor: z.Static, SupportsBrightness: z.Static, SupportedEffects: z.Effects,
		SupportsLedCountConfiguration: configurable, MinLedCount: z.LEDMin, MaxLedCount: z.LEDMax, ConfiguredLedCount: z.ConfiguredLEDCount,
		SetupNeeded: configurable && (z.ConfiguredLEDCount == 0 || z.ConfiguredLEDCount < z.LEDMin || z.ConfiguredLEDCount > z.LEDMax || z.LEDCount != z.ConfiguredLEDCount) || z.LEDCount == 0}
}

// ConfigureZone stores physical wiring intent before attempting output. The
// normal reconciler remains the only writer and preserves active manual Off.
func (s *Service) ConfigureZone(ctx context.Context, controllerID, zoneID string, count uint32) (*controlv1.ZoneCapability, []*controlv1.ReconcileZoneResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if strings.TrimSpace(controllerID) == "" || strings.TrimSpace(zoneID) == "" {
		return nil, nil, ErrInvalidState
	}
	status, _, err := s.ListControllers(ctx)
	if err != nil {
		return nil, nil, err
	}
	if status.State != controlv1.BackendState_BACKEND_STATE_READY {
		return nil, nil, ErrBackendUnavailable
	}
	c, err := s.findController(ctx, controllerID)
	if err != nil {
		return nil, nil, err
	}
	if !c.Enrolled {
		return nil, nil, ErrNotEnrolled
	}
	if !stableIdentitySource(c.IdentitySource) {
		return nil, nil, ErrUnstableIdentity
	}
	mappings, err := s.repo.ListZones(ctx, controllerID)
	if err != nil {
		return nil, nil, err
	}
	var selected *ZoneMapping
	for i := range mappings {
		if mappings[i].ZoneID == zoneID {
			selected = &mappings[i]
		}
	}
	if selected == nil {
		return nil, nil, ErrControllerNotFound
	}
	if err := validateZoneLength(SDKZone{LEDMin: selected.LEDMin, LEDMax: selected.LEDMax, Flags: 2}, count); err != nil {
		return nil, nil, err
	}
	var desiredTotal uint64
	for _, zone := range mappings {
		length := zone.LEDCount
		if zone.ConfiguredLEDCount > 0 {
			length = zone.ConfiguredLEDCount
		}
		if zone.ZoneID == zoneID {
			length = count
		}
		desiredTotal += uint64(length)
	}
	if desiredTotal > 65535 {
		return nil, nil, fmt.Errorf("%w: controller LED total exceeds SDK capacity", ErrInvalidState)
	}
	// A new length without prior intent starts safely Off. Existing manual or
	// automatic state and ownership are preserved.
	source, err := s.repo.DesiredSource(ctx, controllerID, zoneID)
	if err != nil {
		return nil, nil, err
	}
	if source == "" {
		desired := &controlv1.ZoneState{ControllerId: controllerID, ZoneId: zoneID, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
		if err := s.repo.SaveDesired(ctx, desired, nil); err != nil {
			return nil, nil, err
		}
	}
	// Establish safe initial intent before publishing a usable length. If either
	// write fails, no physical mutation occurs; a retry preserves existing intent.
	if err := s.repo.SaveZoneConfiguration(ctx, controllerID, zoneID, count); err != nil {
		return nil, nil, err
	}
	results, err := s.reconcile(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("LED count saved; reconciliation pending: %w", err)
	}
	s.invalidateDiscovery()
	_, _, err = s.ListControllers(ctx)
	if err != nil {
		return nil, nil, err
	}
	mappings, err = s.repo.ListZones(ctx, controllerID)
	if err != nil {
		return nil, nil, err
	}
	for _, z := range mappings {
		if z.ZoneID == zoneID {
			return mappingCapability(z), results, nil
		}
	}
	return nil, results, ErrControllerNotFound
}

// Match the complete physical roster on the mutation connection. Missing,
// newly appeared or ambiguous peers cannot authorize a controller-wide mode.
func completeControllerOffScope(c SDKController, expected map[string]uint32) bool {
	if len(expected) == 0 || len(expected) != len(c.Zones) {
		return false
	}
	seen := map[string]bool{}
	for _, z := range c.Zones {
		fp := zoneFingerprint(z)
		if z.LEDCount == 0 || expected[fp] != z.LEDCount || seen[fp] {
			return false
		}
		seen[fp] = true
	}
	return len(seen) == len(expected)
}
