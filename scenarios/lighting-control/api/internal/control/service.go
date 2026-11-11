package control

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	internalpolicy "lighting-control/internal/policy"
)

var (
	ErrControllerNotFound   = errors.New("controller not found")
	ErrNotEnrolled          = errors.New("controller is not enrolled")
	ErrAmbiguousZone        = errors.New("zone identity is ambiguous")
	ErrUnstableIdentity     = errors.New("controller has no stable serial or location identity")
	ErrUnsupported          = errors.New("operation is unsupported by this zone")
	ErrBrightnessLimit      = errors.New("native effect has no usable brightness control for the requested limit")
	ErrModeReadbackMismatch = errors.New("OpenRGB ACK was received, but cached mode readback did not match the requested mode")
	ErrInvalidState         = errors.New("invalid desired zone state")
)

type Backend interface {
	ProtocolVersion(context.Context) (uint32, error)
	Discover(context.Context) ([]SDKController, error)
	SendZoneColors(context.Context, uint32, uint32, []RGB) (SDKCommandResult, error)
}

type modeBackend interface {
	SendMode(context.Context, uint32, SDKModeTarget, SDKMode) (SDKCommandResult, error)
}

type stableZoneBackend interface {
	ApplyStableZone(context.Context, string, string, *controlv1.ZoneState) (SDKCommandResult, error)
}

type Service struct {
	brightnessPolicy internalpolicy.Store
	repo             Repository
	backend          Backend
	clock            schedule.Clock
	// writeMu serializes all persisted desired-state and reconciliation writes,
	// as well as backend replacement/quiesce against those writes.
	writeMu     sync.Mutex
	cacheMu     sync.Mutex
	cache       []SDKController
	cacheAt     time.Time
	cacheLoaded bool
}

func NewService(repo Repository, backend Backend, clock schedule.Clock) *Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &Service{repo: repo, backend: backend, clock: clock}
}

// BindBrightnessPolicy composes the authoritative policy store without acquiring
// the policy service mutex. Lock order is policy opMu, control writeMu, store.
func (s *Service) BindBrightnessPolicy(store internalpolicy.Store) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.brightnessPolicy = store
}

// UpdateBrightnessPolicy commits a cap change in the same causal order as all
// hardware writes, then reapplies manual intent without adopting policy ownership.
func (s *Service) UpdateBrightnessPolicy(ctx context.Context, save func() error) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := save(); err != nil {
		return err
	}
	if _, err := s.reconcile(ctx); err != nil {
		return fmt.Errorf("brightness policy saved; desired-state reconciliation pending: %w", err)
	}
	return nil
}

func (s *Service) emissionState(ctx context.Context, desired *controlv1.ZoneState) (*controlv1.ZoneState, error) {
	if _, off := desired.DesiredMode.(*controlv1.ZoneState_Off); off {
		return desired, nil
	}
	cap := internalpolicy.DefaultConfig().BrightnessCapPercent
	if s.brightnessPolicy != nil {
		config, _, err := internalpolicy.LoadForRead(ctx, s.brightnessPolicy)
		if err != nil {
			return nil, err
		}
		cap = config.BrightnessCapPercent
	}
	if cap > 100 {
		return nil, errors.New("stored maximum brightness exceeds 100 percent")
	}
	out := proto.Clone(desired).(*controlv1.ZoneState)
	if cap == 0 || requestedBrightness(out) == 0 {
		out.DesiredMode = &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}
		return out, nil
	}
	switch mode := out.DesiredMode.(type) {
	case *controlv1.ZoneState_StaticColor:
		mode.StaticColor.BrightnessPercent = min(mode.StaticColor.BrightnessPercent, cap)
	case *controlv1.ZoneState_Effect:
		mode.Effect.BrightnessPercent = min(mode.Effect.BrightnessPercent, cap)
	}
	return out, nil
}

func (s *Service) BackendStatus(ctx context.Context) *controlv1.BackendStatus {
	now := timestamppb.New(s.clock.Now().UTC())
	status := &controlv1.BackendStatus{CheckedAt: now.AsTime().Format(time.RFC3339Nano)}
	if s.backend == nil {
		status.State = controlv1.BackendState_BACKEND_STATE_DISABLED
		status.Reason = controlv1.BackendReason_BACKEND_REASON_DISABLED
		status.Detail = "OpenRGB SDK endpoint is not configured; no connection was attempted"
		return status
	}
	version, err := s.backend.ProtocolVersion(ctx)
	if err != nil {
		status.State = controlv1.BackendState_BACKEND_STATE_UNAVAILABLE
		status.Reason = backendReason(err)
		status.Detail = err.Error()
		if errors.Is(err, ErrProtocolMismatch) {
			status.State = controlv1.BackendState_BACKEND_STATE_PROTOCOL_MISMATCH
		}
		return status
	}
	status.State = controlv1.BackendState_BACKEND_STATE_READY
	status.Reason = controlv1.BackendReason_BACKEND_REASON_READY
	status.SdkProtocolVersion = version
	return status
}

func backendReason(err error) controlv1.BackendReason {
	switch {
	case errors.Is(err, ErrProtocolMismatch):
		return controlv1.BackendReason_BACKEND_REASON_PROTOCOL_INCOMPATIBLE
	case errors.Is(err, ErrPermission):
		return controlv1.BackendReason_BACKEND_REASON_PERMISSION_DENIED
	case errors.Is(err, ErrBackendUnavailable):
		return controlv1.BackendReason_BACKEND_REASON_STOPPED
	case errors.Is(err, ErrNoDevice):
		return controlv1.BackendReason_BACKEND_REASON_PROBE_FAILED
	default:
		return controlv1.BackendReason_BACKEND_REASON_PROBE_FAILED
	}
}

func (s *Service) ListControllers(ctx context.Context, includeUnavailable ...bool) (*controlv1.BackendStatus, []*controlv1.Controller, error) {
	noDevice := false
	availableIDs := make(map[string]bool)
	if s.backend != nil {
		found, err := s.discover(ctx, true)
		if err != nil && !errors.Is(err, ErrNoDevice) {
			return statusFromError(s.clock.Now(), err), nil, nil
		}
		noDevice = errors.Is(err, ErrNoDevice)
		controllers, mappings, ambiguous := projectControllers(found, s.clock.Now())
		for _, controller := range controllers {
			availableIDs[controller.ControllerId] = true
		}
		if err := s.repo.SyncControllers(ctx, controllers, mappings); err != nil {
			return nil, nil, fmt.Errorf("persist controller discovery: %w", err)
		}
		if len(ambiguous) > 0 {
			// Keep discovery visible but fail closed when an exact fingerprint
			// collision prevents mapping a durable zone identity to one SDK zone.
		}
	}
	controllers, err := s.repo.ListControllers(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("list persisted controllers: %w", err)
	}
	status := s.BackendStatus(ctx)
	if noDevice {
		status.Reason = controlv1.BackendReason_BACKEND_REASON_PROBE_FAILED
		status.Detail = ErrNoDevice.Error()
	}
	include := len(includeUnavailable) > 0 && includeUnavailable[0]
	if !include {
		current := controllers[:0]
		for _, c := range controllers {
			if availableIDs[c.ControllerId] {
				current = append(current, c)
			}
		}
		controllers = current
	}
	return status, controllers, nil
}

func statusFromError(now time.Time, err error) *controlv1.BackendStatus {
	state := controlv1.BackendState_BACKEND_STATE_UNAVAILABLE
	if errors.Is(err, ErrProtocolMismatch) {
		state = controlv1.BackendState_BACKEND_STATE_PROTOCOL_MISMATCH
	}
	return &controlv1.BackendStatus{State: state, Reason: backendReason(err), Detail: err.Error(), CheckedAt: now.UTC().Format(time.RFC3339Nano)}
}

func zoneFingerprint(z SDKZone) string {
	name := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(z.Name)), " "))
	return fmt.Sprintf("%s|%d", name, z.LEDCount)
}

func stableID(prefix, material string) string {
	sum := sha256.Sum256([]byte(material))
	return prefix + hex.EncodeToString(sum[:12])
}

func projectControllers(found []SDKController, lastSeen time.Time) ([]*controlv1.Controller, []ZoneMapping, map[string]bool) {
	controllers := make([]*controlv1.Controller, 0, len(found))
	mappings := make([]ZoneMapping, 0)
	ambiguous := map[string]bool{}
	identityCounts := make(map[string]int, len(found))
	for _, raw := range found {
		id, source := controllerIdentity(raw)
		if stableIdentitySource(source) {
			identityCounts[id]++
		}
	}
	for _, raw := range found {
		id, identitySource := controllerIdentity(raw)
		if identityCounts[id] > 1 {
			id = stableID("ambiguous-controller-", id+"|sdk-handle|"+strconv.FormatUint(uint64(raw.ID), 10))
			identitySource = "openrgb_identity_ambiguous"
		}
		c := &controlv1.Controller{BackendId: "openrgb", ControllerId: id, Name: raw.Name, Vendor: raw.Vendor, Description: raw.Description, IdentitySource: identitySource, LastSeenAt: timestamppb.New(lastSeen.UTC())}
		counts := map[string]int{}
		for _, z := range raw.Zones {
			counts[zoneFingerprint(z)]++
		}
		for _, z := range raw.Zones {
			fingerprint := zoneFingerprint(z)
			zoneID := stableID("zone-", id+"|"+fingerprint)
			capability := &controlv1.ZoneCapability{ZoneId: zoneID, Name: z.Name, LedCount: z.LEDCount, SupportsStaticColor: z.Static, SupportsBrightness: z.Static}
			for _, mode := range z.Modes {
				if !strings.EqualFold(strings.TrimSpace(mode.Name), "static") && strings.TrimSpace(mode.Name) != "" {
					capability.SupportedEffects = append(capability.SupportedEffects, mode.Name)
				}
			}
			if sdkControllerModeCoversZone(raw.LEDCount, zoneLEDCounts(raw.Zones), z.Index) {
				for _, mode := range raw.Modes {
					if strings.EqualFold(strings.TrimSpace(mode.Name), "static") {
						capability.SupportsStaticColor = true
						capability.SupportsBrightness = true
						continue
					}
					if strings.TrimSpace(mode.Name) != "" {
						capability.SupportedEffects = append(capability.SupportedEffects, mode.Name)
					}
				}
			}
			c.Zones = append(c.Zones, capability)
			if counts[fingerprint] > 1 {
				ambiguous[id+"/"+zoneID] = true
				continue
			}
			mappings = append(mappings, ZoneMapping{ControllerID: id, ZoneID: zoneID, Fingerprint: fingerprint, Name: z.Name, LEDCount: z.LEDCount, Static: z.Static})
		}
		controllers = append(controllers, c)
	}
	return controllers, mappings, ambiguous
}

func controllerIdentity(raw SDKController) (string, string) {
	if id, source := stableControllerIdentity(SDKHardwareIdentity{
		Type: raw.Type, Name: raw.Name, Vendor: raw.Vendor,
		Description: raw.Description, Serial: raw.Serial, Location: raw.Location,
	}); id != "" {
		return id, source
	}
	// Keep transient SDK handles visible for diagnostics, but never enroll or
	// write to them because they may change when devices reorder or the daemon restarts.
	return "transient-openrgb-" + strconv.FormatUint(uint64(raw.ID), 10), "openrgb_sdk_transient_handle"
}

func stableIdentitySource(source string) bool {
	return source == "openrgb_sdk_serial" || source == "openrgb_linux_pci_i2c_location"
}

func (s *Service) Enroll(ctx context.Context, id string, enrolled bool) (*controlv1.Controller, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrControllerNotFound
	}
	if enrolled {
		controller, err := s.findController(ctx, id)
		if err != nil {
			return nil, err
		}
		if !stableIdentitySource(controller.IdentitySource) {
			return nil, ErrUnstableIdentity
		}
	}
	c, err := s.repo.SetEnrollment(ctx, id, enrolled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrControllerNotFound
	}
	return c, err
}

func (s *Service) SetZoneState(ctx context.Context, desired *controlv1.ZoneState) (*controlv1.ZoneApplyResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := validateDesired(desired); err != nil {
		return nil, ErrInvalidState
	}
	c, err := s.findController(ctx, desired.ControllerId)
	if err != nil {
		return nil, err
	}
	if !c.Enrolled {
		return nil, ErrNotEnrolled
	}
	if !stableIdentitySource(c.IdentitySource) {
		return nil, ErrUnstableIdentity
	}
	if err := s.repo.SaveDesired(ctx, desired, nil); err != nil {
		return nil, err
	}
	override, err := s.activeOverride(ctx)
	if err != nil {
		return nil, err
	}
	if covers(override, desired.ControllerId) {
		result := &controlv1.ZoneApplyResult{Desired: desired, Operation: controlv1.OperationState_OPERATION_STATE_DEFERRED, Reason: "manual Off override has higher priority", CompletedAt: timestamppb.New(s.clock.Now().UTC())}
		if err := s.repo.SaveDesired(ctx, desired, result); err != nil {
			return nil, err
		}
		return result, nil
	}
	result := s.apply(ctx, desired)
	if err := s.repo.SaveDesired(ctx, desired, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ApplyPolicyState projects fresh policy output through the serialized writer.
// Explicit manual zone settings take precedence. Automatic pending state is
// persisted for inspection, but only the policy owner may reevaluate and retry it.
func (s *Service) ApplyPolicyState(ctx context.Context, desired *controlv1.ZoneState) (*controlv1.ZoneApplyResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := validateDesired(desired); err != nil {
		return nil, err
	}
	controller, err := s.findController(ctx, desired.ControllerId)
	if err != nil {
		return nil, err
	}
	if !controller.Enrolled {
		return nil, ErrNotEnrolled
	}
	if !stableIdentitySource(controller.IdentitySource) {
		return nil, ErrUnstableIdentity
	}
	zones, err := s.repo.ListZones(ctx, desired.ControllerId)
	if err != nil {
		return nil, err
	}
	found := false
	for _, zone := range zones {
		if zone.ZoneID == desired.ZoneId {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrControllerNotFound
	}
	source, err := s.repo.DesiredSource(ctx, desired.ControllerId, desired.ZoneId)
	if err != nil {
		return nil, err
	}
	if source == "legacy" {
		return &controlv1.ZoneApplyResult{Desired: desired, Operation: controlv1.OperationState_OPERATION_STATE_DEFERRED, Reason: "Saved state predates request ownership. Review it and use Save & apply to adopt manual settings, or explicitly select a policy preset."}, nil
	}
	rows, err := s.repo.ListManualDesired(ctx)
	if err != nil {
		return nil, err
	}
	var base *controlv1.ZoneState
	for _, row := range rows {
		if row != nil && row.Desired != nil && row.Desired.ControllerId == desired.ControllerId && row.Desired.ZoneId == desired.ZoneId {
			base = row.Desired
			break
		}
	}
	if base != nil {
		desired = base
	} else if err := s.repo.SavePolicyDesired(ctx, desired, nil); err != nil {
		return nil, err
	}
	override, err := s.activeOverride(ctx)
	if err != nil {
		return nil, err
	}
	if covers(override, desired.ControllerId) {
		result := &controlv1.ZoneApplyResult{Desired: &controlv1.ZoneState{ControllerId: desired.ControllerId, ZoneId: desired.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}, Operation: controlv1.OperationState_OPERATION_STATE_DEFERRED, Reason: "manual Off override has higher priority", CompletedAt: timestamppb.New(s.clock.Now().UTC())}
		if err := s.repo.SaveApply(ctx, desired, result); err != nil {
			return nil, err
		}
		return result, nil
	}
	result := s.apply(ctx, desired)
	err = s.repo.SaveApply(ctx, desired, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PolicySelection identifies one durable operator selection across retries.
// Policy owns the identity and original request time; control owns its receipt.
type PolicySelection struct {
	ID           string
	BaseRevision *int64
	RequestedAt  time.Time
	Off          bool
	ExpiresAt    *timestamppb.Timestamp
}

// PolicyRevision captures causal override order before policy persists a request.
func (s *Service) PolicyRevision(ctx context.Context) (int64, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.repo.PolicyRevision(ctx)
}

// SelectPolicy serializes precedence and the atomic archive/ownership/Off/receipt
// commit with direct commands. False means a newer active Off superseded it.
func (s *Service) SelectPolicy(ctx context.Context, selection PolicySelection) (bool, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if selection.ID == "" {
		return false, ErrInvalidState
	}
	if selection.ExpiresAt != nil {
		if !selection.ExpiresAt.IsValid() {
			return false, ErrInvalidState
		}
		if !selection.ExpiresAt.AsTime().After(s.clock.Now()) {
			return false, nil
		}
	}
	var off *controlv1.ManualOverride
	var err error
	if selection.Off {
		requestedAt := selection.RequestedAt
		if requestedAt.IsZero() {
			requestedAt = s.clock.Now()
		}
		off, err = s.newManualOff(ctx, true, nil, selection.ExpiresAt, requestedAt)
		if err != nil {
			return false, err
		}
	}
	result, err := s.repo.SelectPolicy(ctx, selection, off)
	if err != nil {
		return false, err
	}
	if result == SelectionSuperseded {
		return false, nil
	}
	if result == SelectionCommitted && off != nil {
		// The durable transition is complete before backend effects. A lost
		// policy acknowledgement never repeats ownership adoption or Off publication.
		if _, err := s.applyManualOff(ctx, off); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Service) SetManualOff(ctx context.Context, all bool, ids []string, expires *timestamppb.Timestamp) (*controlv1.ManualOverride, []*controlv1.ZoneApplyResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	override, err := s.newManualOff(ctx, all, ids, expires, s.clock.Now())
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.SetManualOverride(ctx, override); err != nil {
		return nil, nil, err
	}
	results, err := s.applyManualOff(ctx, override)
	return override, results, err
}

func (s *Service) newManualOff(ctx context.Context, all bool, ids []string, expires *timestamppb.Timestamp, requestedAt time.Time) (*controlv1.ManualOverride, error) {
	if all == (len(ids) > 0) {
		return nil, ErrInvalidState
	}
	if expires != nil && (!expires.IsValid() || !expires.AsTime().After(s.clock.Now())) {
		return nil, ErrInvalidState
	}
	controllers, err := s.repo.ListControllers(ctx)
	if err != nil {
		return nil, err
	}
	selected := map[string]bool{}
	if all {
		for _, c := range controllers {
			if c.Enrolled {
				selected[c.ControllerId] = true
			}
		}
	} else {
		for _, id := range ids {
			found := false
			for _, c := range controllers {
				if c.ControllerId == id && c.Enrolled {
					found = true
					break
				}
			}
			if !found {
				return nil, ErrControllerNotFound
			}
			selected[id] = true
		}
	}
	ordered := make([]string, 0, len(selected))
	for id := range selected {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	return &controlv1.ManualOverride{Active: true, AllEnrolled: all, ControllerIds: ordered, CreatedAt: timestamppb.New(requestedAt.UTC()), ExpiresAt: expires}, nil
}

func (s *Service) applyManualOff(ctx context.Context, override *controlv1.ManualOverride) ([]*controlv1.ZoneApplyResult, error) {
	desiredRows, err := s.repo.ListDesired(ctx)
	if err != nil {
		return nil, err
	}
	knownDesired := make(map[string]*controlv1.ZoneState, len(desiredRows))
	for _, row := range desiredRows {
		if row != nil && row.Desired != nil {
			knownDesired[row.Desired.ControllerId+"/"+row.Desired.ZoneId] = row.Desired
		}
	}
	results := make([]*controlv1.ZoneApplyResult, 0)
	for _, id := range override.ControllerIds {
		zones, err := s.repo.ListZones(ctx, id)
		if err != nil {
			return nil, err
		}
		for _, z := range zones {
			desired := &controlv1.ZoneState{ControllerId: id, ZoneId: z.ZoneID, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
			result := s.apply(ctx, desired)
			if base, exists := knownDesired[id+"/"+z.ZoneID]; exists {
				if err := s.repo.SaveApply(ctx, base, result); err != nil {
					return nil, err
				}
			} else if err := s.repo.SavePolicyDesired(ctx, desired, result); err != nil {
				return nil, err
			}
			results = append(results, result)
		}
	}
	return results, nil
}

func (s *Service) DesiredState(ctx context.Context) ([]*controlv1.ZoneStatus, *controlv1.ManualOverride, error) {
	zones, err := s.repo.ListDesired(ctx)
	if err != nil {
		return nil, nil, err
	}
	override, err := s.repo.GetManualOverride(ctx)
	if err != nil {
		return nil, nil, err
	}
	if override.ExpiresAt != nil && !override.ExpiresAt.AsTime().After(s.clock.Now()) {
		override.Active = false
	}
	return zones, override, nil
}

func (s *Service) ClearManualOverride(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := s.repo.ClearManualOverride(ctx); err != nil {
		return err
	}
	_, err := s.reconcile(ctx)
	return err
}

func (s *Service) Reconcile(ctx context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.reconcile(ctx)
}

func (s *Service) reconcile(ctx context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	status, controllers, err := s.ListControllers(ctx, true)
	if err != nil {
		return nil, err
	}
	if status == nil || status.State != controlv1.BackendState_BACKEND_STATE_READY {
		return nil, fmt.Errorf("OpenRGB discovery is unavailable for policy reconciliation: %w", ErrBackendUnavailable)
	}
	zones, err := s.repo.ListManualDesired(ctx)
	if err != nil {
		return nil, err
	}
	override, err := s.activeOverride(ctx)
	if err != nil {
		return nil, err
	}
	// Off covers every enrolled zone, including zones with automatic or legacy
	// pending state. Those states never become independently replayable bases.
	byZone := make(map[string]*controlv1.ZoneStatus, len(zones))
	for _, zone := range zones {
		byZone[zone.Desired.ControllerId+"/"+zone.Desired.ZoneId] = zone
	}
	for _, controller := range controllers {
		if !controller.Enrolled || !covers(override, controller.ControllerId) {
			continue
		}
		for _, capability := range controller.Zones {
			key := controller.ControllerId + "/" + capability.ZoneId
			if _, exists := byZone[key]; !exists {
				zone := &controlv1.ZoneStatus{Desired: &controlv1.ZoneState{ControllerId: controller.ControllerId, ZoneId: capability.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}}
				source, err := s.repo.DesiredSource(ctx, controller.ControllerId, capability.ZoneId)
				if err != nil {
					return nil, err
				}
				if source == "" {
					if err := s.repo.SavePolicyDesired(ctx, zone.Desired, nil); err != nil {
						return nil, err
					}
				}
				zones = append(zones, zone)
				byZone[key] = zone
			}
		}
	}
	results := make([]*controlv1.ReconcileZoneResult, 0, len(zones))
	for _, z := range zones {
		base := z.Desired
		desired := base
		if override.Active && covers(override, desired.ControllerId) {
			desired = &controlv1.ZoneState{ControllerId: desired.ControllerId, ZoneId: desired.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
		}
		apply := s.apply(ctx, desired)
		err = s.repo.SaveApply(ctx, base, apply)
		if err != nil {
			return nil, err
		}
		results = append(results, &controlv1.ReconcileZoneResult{ControllerId: base.ControllerId, ZoneId: base.ZoneId, Result: apply})
	}
	return results, nil
}

// BindBackend replaces the ephemeral SDK endpoint, waiting for prior writes
// to finish. Only a verified incarnation supplied by the owner is accepted.
func (s *Service) BindBackend(ctx context.Context, address, incarnation string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	binding, ok := s.backend.(*BackendBinding)
	if !ok {
		return errors.New("lighting API was not composed with a dynamic backend binding")
	}
	if err := binding.Bind(ctx, address, incarnation); err != nil {
		return err
	}
	s.invalidateDiscovery()
	return nil
}

// Quiesce drains the desired-state writer before the owner stops OpenRGB.
func (s *Service) Quiesce(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	binding, ok := s.backend.(*BackendBinding)
	if !ok {
		return errors.New("lighting API was not composed with a dynamic backend binding")
	}
	if err := binding.Quiesce(ctx); err != nil {
		return err
	}
	s.invalidateDiscovery()
	return nil
}

func (s *Service) invalidateDiscovery() {
	s.cacheMu.Lock()
	s.cache = nil
	s.cacheAt = time.Time{}
	s.cacheLoaded = false
	s.cacheMu.Unlock()
}

func (s *Service) activeOverride(ctx context.Context) (*controlv1.ManualOverride, error) {
	o, err := s.repo.GetManualOverride(ctx)
	if err != nil {
		return nil, err
	}
	if o.ExpiresAt != nil && !o.ExpiresAt.AsTime().After(s.clock.Now()) {
		o.Active = false
	}
	return o, nil
}

func validateDesired(d *controlv1.ZoneState) error {
	if d == nil || strings.TrimSpace(d.ControllerId) == "" || strings.TrimSpace(d.ZoneId) == "" || d.DesiredMode == nil {
		return ErrInvalidState
	}
	validateColor := func(c *controlv1.Color) bool { return c != nil && c.Red <= 255 && c.Green <= 255 && c.Blue <= 255 }
	switch mode := d.DesiredMode.(type) {
	case *controlv1.ZoneState_Off:
		if mode.Off == nil {
			return ErrInvalidState
		}
	case *controlv1.ZoneState_StaticColor:
		if mode.StaticColor == nil || !validateColor(mode.StaticColor.Color) || mode.StaticColor.BrightnessPercent > 100 {
			return ErrInvalidState
		}
	case *controlv1.ZoneState_Effect:
		if mode.Effect == nil || strings.TrimSpace(mode.Effect.EffectId) == "" || !validateColor(mode.Effect.Color) || mode.Effect.BrightnessPercent > 100 {
			return ErrInvalidState
		}
	default:
		return ErrInvalidState
	}
	return nil
}

func covers(o *controlv1.ManualOverride, id string) bool {
	if o == nil || !o.Active {
		return false
	}
	if o.AllEnrolled {
		return true
	}
	for _, x := range o.ControllerIds {
		if x == id {
			return true
		}
	}
	return false
}

func (s *Service) findController(ctx context.Context, id string) (*controlv1.Controller, error) {
	controllers, err := s.repo.ListControllers(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range controllers {
		if c.ControllerId == id {
			return c, nil
		}
	}
	return nil, ErrControllerNotFound
}

const discoveryCacheTTL = 5 * time.Second

func (s *Service) discover(ctx context.Context, force bool) ([]SDKController, error) {
	if s.backend == nil {
		return nil, ErrBackendUnavailable
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if !force && s.cacheLoaded && s.clock.Now().Sub(s.cacheAt) < discoveryCacheTTL {
		return append([]SDKController(nil), s.cache...), nil
	}
	found, err := s.backend.Discover(ctx)
	if err != nil && !errors.Is(err, ErrNoDevice) {
		return nil, err
	}
	s.cache = append([]SDKController(nil), found...)
	s.cacheAt = s.clock.Now()
	s.cacheLoaded = true
	return found, err
}

func (s *Service) apply(ctx context.Context, d *controlv1.ZoneState) *controlv1.ZoneApplyResult {
	emitted, capErr := s.emissionState(ctx, d)
	if capErr != nil {
		return &controlv1.ZoneApplyResult{Desired: d, Operation: controlv1.OperationState_OPERATION_STATE_DEFERRED, Reason: "maximum brightness policy unavailable: " + capErr.Error(), CompletedAt: timestamppb.New(s.clock.Now().UTC())}
	}
	capped := !proto.Equal(d, emitted)
	d = emitted
	result := &controlv1.ZoneApplyResult{Desired: d, CompletedAt: timestamppb.New(s.clock.Now().UTC()), CachedModeReadback: controlv1.CachedModeReadback_CACHED_MODE_READBACK_NOT_CHECKED}
	if capped {
		defer func() { result.Reason = "maximum brightness cap applied; " + result.Reason }()
	}
	if s.backend == nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = "OpenRGB SDK endpoint is not configured"
		return result
	}
	c, err := s.findController(ctx, d.ControllerId)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = err.Error()
		return result
	}
	if !c.Enrolled {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = ErrNotEnrolled.Error()
		return result
	}
	if !stableIdentitySource(c.IdentitySource) {
		result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
		result.Reason = ErrUnstableIdentity.Error()
		return result
	}
	mappings, err := s.repo.ListZones(ctx, d.ControllerId)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = err.Error()
		return result
	}
	var mapping *ZoneMapping
	for i := range mappings {
		if mappings[i].ZoneID == d.ZoneId {
			mapping = &mappings[i]
			break
		}
	}
	if mapping == nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = ErrControllerNotFound.Error()
		return result
	}
	if stable, ok := s.backend.(stableZoneBackend); ok {
		command, commandErr := stable.ApplyStableZone(ctx, d.ControllerId, mapping.Fingerprint, d)
		result.LocalWriteCompleted = command.LocalWriteCompleted
		result.BackendAcknowledged = command.BackendAcknowledged
		result.BackendAckStatus = command.AckStatus
		if command.CachedModeReadback != controlv1.CachedModeReadback_CACHED_MODE_READBACK_UNSPECIFIED {
			result.CachedModeReadback = command.CachedModeReadback
		}
		if commandErr != nil {
			if d.GetEffect() != nil && errors.Is(commandErr, ErrBrightnessLimit) {
				result = s.applyBrightnessFallback(ctx, d)
				return result
			}
			switch {
			case errors.Is(commandErr, ErrUnsupported), errors.Is(commandErr, ErrUnstableIdentity):
				result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
			case errors.Is(commandErr, ErrControllerNotFound):
				result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
			case command.LocalWriteCompleted && !command.BackendAcknowledged:
				result.Operation = controlv1.OperationState_OPERATION_STATE_COMMAND_SENT
			default:
				result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
			}
			result.Reason = commandErr.Error()
			return result
		}
		if command.BackendAcknowledged {
			result.Operation = controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED
			switch result.CachedModeReadback {
			case controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES:
				result.Reason = "OpenRGB v6 acknowledged the requested operation; fresh cached mode readback matches; physical output remains unverified"
			case controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH:
				result.Reason = "OpenRGB v6 acknowledged the requested operation; cached mode readback mismatched; physical output remains unverified"
			default:
				result.Reason = "OpenRGB v6 acknowledged the requested operation; cached mode readback was not checked; physical output remains unverified"
			}
		} else if command.LocalWriteCompleted {
			result.Operation = controlv1.OperationState_OPERATION_STATE_COMMAND_SENT
			result.Reason = "the command was written locally without a backend acknowledgement; cached mode readback and physical output are unverified"
		} else {
			result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
			result.Reason = "no command was sent"
		}
		return result
	}
	// A cached controller handle is useful for routine inventory reads, but may
	// never authorize a mutation after OpenRGB has reordered its bus devices.
	found, err := s.discover(ctx, true)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = err.Error()
		return result
	}
	var raw *SDKController
	for i := range found {
		id, source := controllerIdentity(found[i])
		if id == d.ControllerId && stableIdentitySource(source) {
			if raw != nil {
				result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
				result.Reason = ErrUnstableIdentity.Error() + ": duplicate live controller identity"
				return result
			}
			raw = &found[i]
		}
	}
	if raw == nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = ErrControllerNotFound.Error()
		return result
	}
	var target *SDKZone
	matches := 0
	for i := range raw.Zones {
		if zoneFingerprint(raw.Zones[i]) == mapping.Fingerprint {
			matches++
			target = &raw.Zones[i]
		}
	}
	if matches > 1 {
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = ErrAmbiguousZone.Error()
		return result
	}
	if matches == 0 {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = "zone is not currently present"
		return result
	}
	if _, nativeModes := s.backend.(modeBackend); !mapping.Static && !nativeModes {
		result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
		result.Reason = ErrUnsupported.Error()
		return result
	}
	if target.LEDCount == 0 {
		result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
		result.Reason = "zone reports no controllable LEDs"
		return result
	}
	color, ok := requestedColor(d)
	if !ok {
		result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
		result.Reason = ErrUnsupported.Error()
		return result
	}
	if _, effect := d.DesiredMode.(*controlv1.ZoneState_Effect); effect {
		if _, supported := s.backend.(modeBackend); !supported {
			result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
			result.Reason = ErrUnsupported.Error() + ": backend adapter cannot apply native mode updates"
			return result
		}
	}
	if modes, ok := s.backend.(modeBackend); ok {
		mode, targetMode, modeOK := requestedMode(d, *raw, *target)
		if !modeOK {
			result.Operation = controlv1.OperationState_OPERATION_STATE_UNSUPPORTED
			result.Reason = "the exact requested zone mode is not advertised at a safe scope"
			return result
		}
		_, wantsEffect := d.DesiredMode.(*controlv1.ZoneState_Effect)
		if wantsEffect && !modeCanHonorBrightness(mode, d) {
			result = s.applyBrightnessFallback(ctx, d)
			return result
		}
		if wantsEffect || !modeAlreadyActive(targetMode, *raw, *target) {
			mode = configureMode(mode, d)
			modeResult, modeErr := modes.SendMode(ctx, raw.ID, targetMode, mode)
			result.LocalWriteCompleted = modeResult.LocalWriteCompleted
			result.BackendAcknowledged = modeResult.BackendAcknowledged
			result.BackendAckStatus = modeResult.AckStatus
			if modeErr != nil {
				result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
				result.Reason = "mode change was not confirmed: " + modeErr.Error()
				return result
			}
		}
		if wantsEffect {
			result.Operation = controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED
			result.Reason = "OpenRGB v6 acknowledged the selected mode; cached readback and visible output are unverified"
			return result
		}
	}
	brightness := requestedBrightness(d)
	color.R = uint8(math.Round(float64(color.R) * brightness / 100))
	color.G = uint8(math.Round(float64(color.G) * brightness / 100))
	color.B = uint8(math.Round(float64(color.B) * brightness / 100))
	colors := make([]RGB, target.LEDCount)
	for i := range colors {
		colors[i] = color
	}
	command, err := s.backend.SendZoneColors(ctx, raw.ID, target.Index, colors)
	result.LocalWriteCompleted = command.LocalWriteCompleted
	result.BackendAcknowledged = command.BackendAcknowledged
	result.BackendAckStatus = command.AckStatus
	if err != nil {
		var ackErr *SDKAckError
		if command.LocalWriteCompleted && !errors.As(err, &ackErr) {
			result.Operation = controlv1.OperationState_OPERATION_STATE_COMMAND_SENT
			result.Reason = "complete local UpdateZoneLEDs write; backend ACK was not received, so backend processing, cached state, device application, and visible output are unknown: " + err.Error()
			return result
		}
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = err.Error()
		return result
	}
	if command.BackendAcknowledged {
		result.Operation = controlv1.OperationState_OPERATION_STATE_BACKEND_ACKNOWLEDGED
		result.Reason = "OpenRGB v6 acknowledged UpdateZoneLEDs with status OK; cached readback and visible output are unverified"
		return result
	}
	result.Operation = controlv1.OperationState_OPERATION_STATE_COMMAND_SENT
	result.Reason = "UpdateZoneLEDs bytes were written locally without a backend acknowledgement; device state and visible output are unverified"
	return result
}

// applyBrightnessFallback does not change durable intent or publish an Off
// override. It only emits a safely scoped Off when an effect cannot honor a cap.
func (s *Service) applyBrightnessFallback(ctx context.Context, desired *controlv1.ZoneState) *controlv1.ZoneApplyResult {
	off := &controlv1.ZoneState{ControllerId: desired.ControllerId, ZoneId: desired.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
	result := s.apply(ctx, off)
	result.Reason = "native effect cannot honor maximum brightness; static Off fallback: " + result.Reason
	if !result.LocalWriteCompleted {
		result.Reason += "; no Off command was completed, so prior output may remain unchanged"
	}
	return result
}

func zoneLEDCounts(zones []SDKZone) []uint32 {
	counts := make([]uint32, len(zones))
	for i := range zones {
		counts[i] = zones[i].LEDCount
	}
	return counts
}

func modeCanHonorBrightness(mode SDKMode, desired *controlv1.ZoneState) bool {
	brightness := uint32(requestedBrightness(desired))
	return brightness == 100 || (mode.Flags&sdkModeHasBrightness != 0 && mode.BrightnessMax > mode.BrightnessMin && uint64(mode.BrightnessMax)*uint64(brightness)/100 >= uint64(mode.BrightnessMin))
}

func configureMode(mode SDKMode, desired *controlv1.ZoneState) SDKMode {
	brightness := uint32(requestedBrightness(desired))
	if mode.Flags&sdkModeHasBrightness != 0 {
		low, high := mode.BrightnessMin, mode.BrightnessMax
		if low > high {
			low, high = high, low
		}
		mode.Brightness = max(low, uint32(uint64(high)*uint64(brightness)/100))
	}
	color, ok := requestedColor(desired)
	if ok && mode.Flags&sdkModeHasModeSpecificColor != 0 && mode.ColorMode == sdkModeColorsModeSpecific {
		packed := uint32(color.R) | uint32(color.G)<<8 | uint32(color.B)<<16
		count := mode.ColorsMin
		if count == 0 {
			count = 1
		}
		if mode.ColorsMax > 0 && count > mode.ColorsMax {
			count = mode.ColorsMax
		}
		if count <= maxSDKModeColors {
			mode.Colors = make([]uint32, count)
			for i := range mode.Colors {
				mode.Colors[i] = packed
			}
		}
	}
	return mode
}

func requestedMode(desired *controlv1.ZoneState, controller SDKController, zone SDKZone) (SDKMode, SDKModeTarget, bool) {
	want := "Static"
	switch mode := desired.DesiredMode.(type) {
	case *controlv1.ZoneState_Off, *controlv1.ZoneState_StaticColor:
	case *controlv1.ZoneState_Effect:
		if mode.Effect == nil {
			return SDKMode{}, SDKModeTarget{}, false
		}
		want = mode.Effect.EffectId
	default:
		return SDKMode{}, SDKModeTarget{}, false
	}
	for i, mode := range zone.Modes {
		if strings.EqualFold(strings.TrimSpace(mode.Name), strings.TrimSpace(want)) && mode.Flags&sdkModeRequiresEntireDevice == 0 {
			idx := uint32(zone.Index)
			return mode, SDKModeTarget{ModeIndex: uint32(i), ZoneIndex: &idx}, true
		}
	}
	for i, mode := range controller.Modes {
		if !strings.EqualFold(strings.TrimSpace(mode.Name), strings.TrimSpace(want)) {
			continue
		}
		counts := make([]uint32, len(controller.Zones))
		for i := range controller.Zones {
			counts[i] = controller.Zones[i].LEDCount
		}
		if sdkControllerModeCoversZone(controller.LEDCount, counts, zone.Index) {
			return mode, SDKModeTarget{ModeIndex: uint32(i)}, true
		}
	}
	return SDKMode{}, SDKModeTarget{}, false
}

func modeAlreadyActive(target SDKModeTarget, controller SDKController, zone SDKZone) bool {
	if target.ZoneIndex != nil {
		return zone.ActiveMode >= 0 && uint32(zone.ActiveMode) == target.ModeIndex
	}
	return controller.ActiveMode >= 0 && uint32(controller.ActiveMode) == target.ModeIndex
}

func modeReadbackMatches(target SDKModeTarget, controller SDKController, zone SDKZone, requested SDKMode) bool {
	if target.ZoneIndex != nil {
		return int(zone.ActiveMode) >= 0 && int(zone.ActiveMode) < len(zone.Modes) &&
			sdkModeMatchesReadback(zone.ActiveMode, zone.Modes[zone.ActiveMode], target.ModeIndex, requested)
	}
	return int(controller.ActiveMode) >= 0 && int(controller.ActiveMode) < len(controller.Modes) &&
		sdkModeMatchesReadback(controller.ActiveMode, controller.Modes[controller.ActiveMode], target.ModeIndex, requested)
}

func requestedColor(d *controlv1.ZoneState) (RGB, bool) {
	switch v := d.DesiredMode.(type) {
	case *controlv1.ZoneState_Off:
		return RGB{}, true
	case *controlv1.ZoneState_StaticColor:
		if v.StaticColor == nil || v.StaticColor.Color == nil {
			return RGB{}, false
		}
		return RGB{uint8(v.StaticColor.Color.Red), uint8(v.StaticColor.Color.Green), uint8(v.StaticColor.Color.Blue)}, true
	case *controlv1.ZoneState_Effect:
		if v.Effect == nil || v.Effect.Color == nil {
			return RGB{}, false
		}
		return RGB{uint8(v.Effect.Color.Red), uint8(v.Effect.Color.Green), uint8(v.Effect.Color.Blue)}, true
	default:
		return RGB{}, false
	}
}

func requestedBrightness(d *controlv1.ZoneState) float64 {
	switch v := d.DesiredMode.(type) {
	case *controlv1.ZoneState_StaticColor:
		if v.StaticColor != nil {
			return float64(v.StaticColor.BrightnessPercent)
		}
	case *controlv1.ZoneState_Effect:
		if v.Effect != nil {
			return float64(v.Effect.BrightnessPercent)
		}
	}
	return 100
}
