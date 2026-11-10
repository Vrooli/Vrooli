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
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrControllerNotFound = errors.New("controller not found")
	ErrNotEnrolled        = errors.New("controller is not enrolled")
	ErrAmbiguousZone      = errors.New("zone identity is ambiguous")
	ErrUnsupported        = errors.New("operation is unsupported by this zone")
	ErrInvalidState       = errors.New("invalid desired zone state")
)

type Backend interface {
	ProtocolVersion(context.Context) (uint32, error)
	Discover(context.Context) ([]SDKController, error)
	SendZoneColors(context.Context, uint32, uint32, []RGB) (SDKCommandResult, error)
}

type Service struct {
	repo        Repository
	backend     Backend
	clock       schedule.Clock
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
		for _, controller := range found {
			availableIDs[strconv.FormatUint(uint64(controller.ID), 10)] = true
		}
		controllers, mappings, ambiguous := projectControllers(found, s.clock.Now())
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
	for _, raw := range found {
		id := strconv.FormatUint(uint64(raw.ID), 10)
		c := &controlv1.Controller{BackendId: "openrgb", ControllerId: id, Name: raw.Name, Vendor: raw.Vendor, Description: raw.Description, IdentitySource: "openrgb_sdk_v6_unique_id", LastSeenAt: timestamppb.New(lastSeen.UTC())}
		counts := map[string]int{}
		for _, z := range raw.Zones {
			counts[zoneFingerprint(z)]++
		}
		for _, z := range raw.Zones {
			fingerprint := zoneFingerprint(z)
			zoneID := stableID("zone-", id+"|"+fingerprint)
			c.Zones = append(c.Zones, &controlv1.ZoneCapability{ZoneId: zoneID, Name: z.Name, LedCount: z.LEDCount, SupportsStaticColor: z.Static, SupportsBrightness: z.Static})
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

func (s *Service) Enroll(ctx context.Context, id string, enrolled bool) (*controlv1.Controller, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrControllerNotFound
	}
	c, err := s.repo.SetEnrollment(ctx, id, enrolled)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrControllerNotFound
	}
	return c, err
}

func (s *Service) SetZoneState(ctx context.Context, desired *controlv1.ZoneState) (*controlv1.ZoneApplyResult, error) {
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

func (s *Service) SetManualOff(ctx context.Context, all bool, ids []string, expires *timestamppb.Timestamp) (*controlv1.ManualOverride, []*controlv1.ZoneApplyResult, error) {
	if all == (len(ids) > 0) {
		return nil, nil, ErrInvalidState
	}
	if expires != nil && !expires.IsValid() {
		return nil, nil, ErrInvalidState
	}
	if expires != nil && !expires.AsTime().After(s.clock.Now()) {
		return nil, nil, ErrInvalidState
	}
	controllers, err := s.repo.ListControllers(ctx)
	if err != nil {
		return nil, nil, err
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
				return nil, nil, ErrControllerNotFound
			}
			selected[id] = true
		}
	}
	ordered := make([]string, 0, len(selected))
	for id := range selected {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	override := &controlv1.ManualOverride{Active: true, AllEnrolled: all, ControllerIds: ordered, CreatedAt: timestamppb.New(s.clock.Now().UTC()), ExpiresAt: expires}
	if err := s.repo.SetManualOverride(ctx, override); err != nil {
		return nil, nil, err
	}
	results := make([]*controlv1.ZoneApplyResult, 0)
	for _, id := range ordered {
		zones, e := s.repo.ListZones(ctx, id)
		if e != nil {
			return nil, nil, e
		}
		for _, z := range zones {
			desired := &controlv1.ZoneState{ControllerId: id, ZoneId: z.ZoneID, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
			result := s.apply(ctx, desired)
			if err := s.repo.SaveDesired(ctx, desired, result); err != nil {
				return nil, nil, err
			}
			results = append(results, result)
		}
	}
	return override, results, nil
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
	if override.Active {
		for _, z := range zones {
			if covers(override, z.Desired.ControllerId) {
				z.Desired = &controlv1.ZoneState{ControllerId: z.Desired.ControllerId, ZoneId: z.Desired.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
			}
		}
	}
	return zones, override, nil
}

func (s *Service) ClearManualOverride(ctx context.Context) error {
	if err := s.repo.ClearManualOverride(ctx); err != nil {
		return err
	}
	_, err := s.Reconcile(ctx)
	return err
}

func (s *Service) Reconcile(ctx context.Context) ([]*controlv1.ReconcileZoneResult, error) {
	_, _, err := s.ListControllers(ctx)
	if err != nil {
		return nil, err
	}
	zones, override, err := s.DesiredState(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*controlv1.ReconcileZoneResult, 0, len(zones))
	for _, z := range zones {
		desired := z.Desired
		if override.Active && covers(override, desired.ControllerId) {
			desired = &controlv1.ZoneState{ControllerId: desired.ControllerId, ZoneId: desired.ZoneId, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
		}
		apply := s.apply(ctx, desired)
		if err := s.repo.SaveDesired(ctx, desired, apply); err != nil {
			return nil, err
		}
		results = append(results, &controlv1.ReconcileZoneResult{ControllerId: desired.ControllerId, ZoneId: desired.ZoneId, Result: apply})
	}
	return results, nil
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
	result := &controlv1.ZoneApplyResult{Desired: d, CompletedAt: timestamppb.New(s.clock.Now().UTC())}
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
	mappings, err := s.repo.ListZones(ctx, d.ControllerId)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = err.Error()
		return result
	}
	controllerID, err := strconv.ParseUint(d.ControllerId, 10, 32)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_FAILED
		result.Reason = ErrControllerNotFound.Error()
		return result
	}
	found, err := s.discover(ctx, false)
	if err != nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = err.Error()
		return result
	}
	var raw *SDKController
	for i := range found {
		if strconv.FormatUint(uint64(found[i].ID), 10) == d.ControllerId {
			raw = &found[i]
			break
		}
	}
	if raw == nil {
		result.Operation = controlv1.OperationState_OPERATION_STATE_DEFERRED
		result.Reason = ErrControllerNotFound.Error()
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
	if !mapping.Static {
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
	brightness := requestedBrightness(d)
	color.R = uint8(math.Round(float64(color.R) * brightness / 100))
	color.G = uint8(math.Round(float64(color.G) * brightness / 100))
	color.B = uint8(math.Round(float64(color.B) * brightness / 100))
	colors := make([]RGB, target.LEDCount)
	for i := range colors {
		colors[i] = color
	}
	command, err := s.backend.SendZoneColors(ctx, uint32(controllerID), target.Index, colors)
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
		result.Reason = "OpenRGB v6 acknowledged and processed UpdateZoneLEDs with status OK; this does not prove cached controller state or visible output"
		return result
	}
	result.Operation = controlv1.OperationState_OPERATION_STATE_COMMAND_SENT
	result.Reason = "UpdateZoneLEDs bytes were written locally without a backend acknowledgement; device state and visible output are unverified"
	return result
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
		return RGB{}, false
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
