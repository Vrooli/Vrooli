package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	authdomain "device-control/internal/auth"
	devicedomain "device-control/internal/devices"
	internalflows "device-control/internal/flows"
	identitydomain "device-control/internal/identity"
	"device-control/strategy"
	"device-control/strategy/androidtvremote"
	strategyregistry "device-control/strategy/registry"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/filerouting"
)

type routedDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Service struct {
	registry            *strategyregistry.Registry
	db                  routedDB
	fileRoots           *filerouting.RoutedRoots
	attached            AttachedReader
	mu                  sync.Mutex
	sessions            map[string]Session
	audits              []Audit
	agents              map[string]AgentRun
	devices             *devicedomain.Store
	artifacts           map[string]string
	artifactKinds       map[string]string
	evidenceDir         string
	activeCancels       map[string]context.CancelFunc
	transportStrategies map[string]strategy.Strategy
	transportStates     map[string]transportState
	transportProfiles   map[string]transportState
	anchors             *internalflows.AnchorStore
	runs                map[string]RunResult
	flowRuns            map[string]Flow
	runDevices          map[string]string
	library             internalflows.Library
	externalRecordings  map[string]externalRecording
	auth                *authdomain.Store
	stateEvents         *strategy.EventBus
	observedStates      map[string]any
	observerCancels     map[string]context.CancelFunc
	actuationCauses     map[string]actuationCause
	auditAliases        map[string]map[string]struct{}
	agentPlanner        internalflows.AgentPlanner
	inventoryTimeout    time.Duration
	sessionQueryTimeout time.Duration
	pendingPairings     map[string]pendingPairing
}

type externalRecording struct {
	DeviceID string
	Actor    string
	Recorder strategy.SessionRecorder
	Handle   strategy.RecordingHandle
}

type interactivePairer interface {
	BeginPairing(context.Context) (androidtvremote.PairingSession, error)
	CompletePairing(context.Context, androidtvremote.PairingSession, []byte) (strategy.PairResult, error)
}

type pendingPairing struct {
	deviceID  string
	pairer    interactivePairer
	session   androidtvremote.PairingSession
	expiresAt time.Time
}

// ActuationCorrelationWindow links a transport event to the operator command
// that just caused it without misattributing a later physical-remote change.
const ActuationCorrelationWindow = 5 * time.Second

type actuationCause struct {
	id        string
	createdAt time.Time
}

type scopedStateSink struct {
	service   *Service
	deviceID  string
	transport string
}

func (s scopedStateSink) Publish(event strategy.StateChangeEvent) {
	event.DeviceID = s.deviceID
	if strings.TrimSpace(event.Transport) == "" {
		event.Transport = s.transport
	}
	s.service.EmitStateChange(event)
}

type transportState struct {
	DeviceID   string
	Serial     string
	StrategyID string
	Transport  string
	Endpoint   string
	UpdatedAt  time.Time
}

func New(registry *strategyregistry.Registry) *Service {
	dir := filepath.Join(os.TempDir(), "device-control-evidence")
	if configDir, err := os.UserConfigDir(); err == nil {
		dir = filepath.Join(configDir, "vrooli", "device-control", "evidence")
	}
	if configured := strings.TrimSpace(os.Getenv("DEVICE_CONTROL_EVIDENCE_DIR")); configured != "" {
		dir = configured
	}
	authStore, _ := authdomain.NewStore(nil, nil)
	return &Service{registry: registry, sessions: map[string]Session{}, audits: []Audit{}, agents: map[string]AgentRun{}, devices: devicedomain.NewStore(), artifacts: map[string]string{}, artifactKinds: map[string]string{}, evidenceDir: dir, activeCancels: map[string]context.CancelFunc{}, observerCancels: map[string]context.CancelFunc{}, actuationCauses: map[string]actuationCause{}, auditAliases: map[string]map[string]struct{}{}, transportStrategies: map[string]strategy.Strategy{}, transportStates: map[string]transportState{}, transportProfiles: map[string]transportState{}, anchors: internalflows.NewAnchorStore(), runs: map[string]RunResult{}, flowRuns: map[string]Flow{}, runDevices: map[string]string{}, externalRecordings: map[string]externalRecording{}, auth: authStore, stateEvents: strategy.NewEventBus(), observedStates: map[string]any{}, inventoryTimeout: defaultInventoryTimeout, sessionQueryTimeout: 750 * time.Millisecond, pendingPairings: map[string]pendingPairing{}}
}

func (s *Service) startObserverLocked(record devicedomain.Record) {
	if s.observerCancels[record.ID] != nil {
		return
	}
	type candidate struct {
		strategyID string
		endpoint   string
	}
	candidates := make([]candidate, 0, len(record.Transports)+1)
	for _, profile := range record.Transports {
		strategyID := strings.TrimSpace(profile.StrategyID)
		if strategyID == "" {
			strategyID = strings.TrimSpace(profile.Name)
		}
		if strategyID != "" {
			candidates = append(candidates, candidate{strategyID: strategyID, endpoint: strings.TrimSpace(profile.Endpoint)})
		}
	}
	if record.StrategyID != "" {
		candidates = append(candidates, candidate{strategyID: strings.TrimSpace(record.StrategyID), endpoint: strings.TrimSpace(record.Endpoint)})
	}
	for _, selected := range candidates {
		base, ok := s.registry.Get(selected.strategyID)
		if !ok {
			continue
		}
		if scoped, scopedOK := base.(strategy.DeviceScoped); scopedOK && record.Serial != "" {
			base = scoped.ForDevice(record.Serial)
		}
		if endpointScoped, endpointOK := base.(interface {
			ForEndpoint(string) strategy.Strategy
		}); endpointOK && selected.endpoint != "" {
			base = endpointScoped.ForEndpoint(selected.endpoint)
		}
		observer, hasObserver := base.(strategy.StateObserver)
		declaration, describeErr := base.Describe(context.Background())
		if describeErr != nil {
			continue
		}
		mode := declaration.StateObservation.Mode
		interval := declaration.StateObservation.Interval
		if mode == "" {
			mode = declaration.ObservationMode
		}
		if interval <= 0 {
			interval = declaration.ObservationInterval
		}
		if !hasObserver && mode != "poll" {
			continue
		}
		if !hasObserver {
			reader, readerOK := base.(strategy.StateReader)
			if !readerOK {
				continue
			}
			if interval <= 0 {
				interval = time.Second
			}
			ctx, cancel := context.WithCancel(context.Background())
			s.observerCancels[record.ID] = cancel
			go s.runPollObserver(ctx, record.ID, selected.strategyID, reader, interval)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.observerCancels[record.ID] = cancel
		go func() {
			backoff := time.Second
			for {
				err := observer.ObserveState(ctx, scopedStateSink{service: s, deviceID: record.ID, transport: selected.strategyID})
				if ctx.Err() != nil {
					return
				}
				if err != nil {
					s.EmitStateChange(strategy.StateChangeEvent{DeviceID: record.ID, Transport: selected.strategyID, Attribute: "transport_health", OldValue: "available", NewValue: "unreachable", StateClass: strategy.EventBearing, CausationID: uuid.NewString()})
				} else {
					s.EmitStateChange(strategy.StateChangeEvent{DeviceID: record.ID, Transport: selected.strategyID, Attribute: "transport_health", OldValue: "unreachable", NewValue: "available", StateClass: strategy.EventBearing, CausationID: uuid.NewString()})
				}
				timer := time.NewTimer(backoff)
				select {
				case <-ctx.Done():
					if !timer.Stop() {
						<-timer.C
					}
					return
				case <-timer.C:
				}
				if backoff < 30*time.Second {
					backoff *= 2
					if backoff > 30*time.Second {
						backoff = 30 * time.Second
					}
				}
			}
		}()
		return
	}
}

func (s *Service) runPollObserver(ctx context.Context, deviceID, transport string, reader strategy.StateReader, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var previous map[string]strategy.PropertyValue
	for {
		state, err := reader.ReadState(ctx)
		if err != nil {
			s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: transport, Attribute: "transport_health", NewValue: "unreachable", StateClass: strategy.EventBearing})
		} else {
			for name, value := range state.Properties {
				old, existed := previous[name]
				if !existed || !reflect.DeepEqual(old.Value, value.Value) {
					var oldValue any
					if existed {
						oldValue = old.Value
					}
					s.EmitStateChange(strategy.StateChangeEvent{DeviceID: deviceID, Transport: transport, Attribute: name, OldValue: oldValue, NewValue: value.Value, StateClass: strategy.StateBearing})
				}
			}
			previous = state.Properties
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func NewWithAttached(registry *strategyregistry.Registry, reader AttachedReader) *Service {
	s := New(registry)
	s.attached = reader
	return s
}

func (s *Service) SetAgentPlanner(planner internalflows.AgentPlanner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentPlanner = planner
}

// SubscribeStateChanges exposes the local fast-path seam for a future rule
// engine. It intentionally has no vrooli-events integration.
func (s *Service) SubscribeStateChanges(buffer int) strategy.StateSubscription {
	s.mu.Lock()
	if s.stateEvents == nil {
		s.stateEvents = strategy.NewEventBus()
	}
	bus := s.stateEvents
	s.mu.Unlock()
	return bus.Subscribe(buffer)
}

func (s *Service) EmitStateChange(event strategy.StateChangeEvent) {
	if event.ObservedAt.IsZero() {
		event.ObservedAt = time.Now().UTC()
	}
	s.mu.Lock()
	if strings.TrimSpace(event.CausationID) == "" {
		if cause, ok := s.actuationCauses[event.DeviceID]; ok && time.Since(cause.createdAt) <= ActuationCorrelationWindow {
			event.CausationID = cause.id
		} else {
			event.CausationID = uuid.NewString()
		}
	}
	if s.stateEvents == nil {
		s.stateEvents = strategy.NewEventBus()
	}
	bus := s.stateEvents
	s.mu.Unlock()
	bus.Publish(event)
}

// NewWithDB keeps the in-memory registry fast while making operator state
// durable. The API already owns the SQLite connection; passing it here avoids
// a second connection and keeps test-mode routing under api-core's control.
func NewWithDB(registry *strategyregistry.Registry, db routedDB, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s := New(registry)
	s.db = db
	if len(roots) > 0 {
		s.fileRoots = roots[0]
	}
	if db == nil {
		return s, nil
	}
	anchors, err := internalflows.NewAnchorStoreWithDB(db)
	if err != nil {
		return nil, err
	}
	s.anchors = anchors
	library, err := internalflows.NewSQLiteLibrary(db)
	if err != nil {
		return nil, err
	}
	s.library = library
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS device_control_sessions (
 id TEXT PRIMARY KEY, device_id TEXT NOT NULL, actor TEXT NOT NULL,
 state TEXT NOT NULL, lease_token TEXT NOT NULL DEFAULT '', kill_reason TEXT NOT NULL DEFAULT '',
 expires_at TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS device_control_sessions_device ON device_control_sessions(device_id, state);
CREATE TABLE IF NOT EXISTS device_control_audits (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, device_id TEXT NOT NULL, transport TEXT NOT NULL DEFAULT '', causation_id TEXT NOT NULL DEFAULT '', lease_id TEXT NOT NULL,
 verb TEXT NOT NULL, outcome TEXT NOT NULL, created_at TEXT NOT NULL, redaction_verified INTEGER NOT NULL,
	redaction_opted_out INTEGER NOT NULL DEFAULT 0, profile_id TEXT NOT NULL DEFAULT '',
	interactive INTEGER NOT NULL DEFAULT 0, evidence_backed INTEGER NOT NULL DEFAULT 1,
 method TEXT NOT NULL DEFAULT '', attempts INTEGER NOT NULL DEFAULT 0,
 provider_state TEXT NOT NULL DEFAULT '', before_lock_state TEXT NOT NULL DEFAULT '',
 after_lock_state TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS device_control_transports (
 device_id TEXT PRIMARY KEY, serial TEXT NOT NULL, strategy_id TEXT NOT NULL,
 transport TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at TEXT NOT NULL

);
CREATE TABLE IF NOT EXISTS device_control_transport_profiles (
 device_id TEXT NOT NULL, serial TEXT NOT NULL, strategy_id TEXT NOT NULL,
 transport TEXT NOT NULL, endpoint TEXT NOT NULL, updated_at TEXT NOT NULL,
 PRIMARY KEY (device_id, strategy_id, transport)

);
CREATE TABLE IF NOT EXISTS device_control_identity_claims (
 device_id TEXT NOT NULL, kind TEXT NOT NULL, value TEXT NOT NULL,
 strategy_id TEXT NOT NULL, evidence TEXT NOT NULL,
 PRIMARY KEY (device_id, kind, value)
);
CREATE TABLE IF NOT EXISTS device_control_identity_merges (
 canonical_id TEXT NOT NULL, member_id TEXT NOT NULL,
 claim_kind TEXT NOT NULL, claim_value TEXT NOT NULL,
 claim_strategy_id TEXT NOT NULL, claim_evidence TEXT NOT NULL,
 canonical_snapshot TEXT NOT NULL, member_snapshot TEXT NOT NULL,
 merged_at TEXT NOT NULL,
 PRIMARY KEY (canonical_id, member_id)

);
CREATE TABLE IF NOT EXISTS device_control_identity_aliases (
 canonical_id TEXT NOT NULL, alias_id TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY (canonical_id, alias_id)
);`); err != nil {
		return nil, fmt.Errorf("initialize device-control state: %w", err)
	}
	// Existing local databases predate the opt-out audit field. This additive
	// migration is intentionally best-effort because SQLite reports a duplicate
	// column when the database has already been upgraded.
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN redaction_opted_out INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN interactive INTEGER NOT NULL DEFAULT 0`)
	_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN evidence_backed INTEGER NOT NULL DEFAULT 1`)
	for _, column := range []string{
		`transport TEXT NOT NULL DEFAULT ''`,
		`causation_id TEXT NOT NULL DEFAULT ''`,
		`profile_id TEXT NOT NULL DEFAULT ''`,
		`method TEXT NOT NULL DEFAULT ''`,
		`attempts INTEGER NOT NULL DEFAULT 0`,
		`provider_state TEXT NOT NULL DEFAULT ''`,
		`before_lock_state TEXT NOT NULL DEFAULT ''`,
		`after_lock_state TEXT NOT NULL DEFAULT ''`,
	} {
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE device_control_audits ADD COLUMN `+column)
	}
	authStore, err := authdomain.NewStore(db, nil)
	if err != nil {
		return nil, err
	}
	s.auth = authStore
	if remote, ok := s.registry.Get("android-tv-remote"); ok {
		if configurable, ok := remote.(interface {
			SetCertificateStore(androidtvremote.CertificateStore)
		}); ok {
			configurable.SetCertificateStore(authStore)
		}
	}
	if err := s.loadSessions(); err != nil {
		return nil, err
	}
	if err := s.loadTransportStates(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityClaims(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityMerges(); err != nil {
		return nil, err
	}
	if err := s.loadIdentityAliases(); err != nil {
		return nil, err
	}
	s.restoreTransportStrategies()
	return s, nil
}

func (s *Service) loadSessions() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions`)
	if err != nil {
		return fmt.Errorf("load device sessions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var session Session
		var expires, created string
		if err := rows.Scan(&session.ID, &session.DeviceID, &session.Actor, &session.State, &session.LeaseToken, &session.KillReason, &expires, &created); err != nil {
			return fmt.Errorf("read device session: %w", err)
		}
		session.ExpiresAt, err = time.Parse(time.RFC3339Nano, expires)
		if err != nil {
			return fmt.Errorf("parse device session expiry: %w", err)
		}
		session.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return fmt.Errorf("parse device session creation: %w", err)
		}
		s.sessions[session.ID] = session
	}
	return rows.Err()
}

func (s *Service) Anchors() *internalflows.AnchorStore { return s.anchors }

func (s *Service) ReadDeviceState(ctx context.Context, deviceID string) (strategy.DeviceState, error) {
	record, found := s.devices.Get(deviceID)
	if !found {
		return strategy.DeviceState{}, fmt.Errorf("unknown or unavailable device %q", deviceID)
	}
	profiles := append([]strategy.DeviceTransport(nil), record.Transports...)
	if len(profiles) == 0 {
		name := record.Transport
		if name == "" {
			name = "usb"
		}
		profiles = append(profiles, strategy.DeviceTransport{StrategyID: record.StrategyID, Name: name, Endpoint: record.Endpoint, Health: record.Health, HealthReason: record.HealthReason, Properties: append([]strategy.PropertyDescriptor(nil), record.Properties...)})
	}
	combined := strategy.DeviceState{Properties: map[string]strategy.PropertyValue{}, Unavailable: map[string]string{}}
	var readErrors []string
	readCount := 0
	propertySources := map[string]strategy.PropertyDescriptor{}
	for _, profile := range profiles {
		transport := profile.Name
		if transport == "" {
			transport = profile.StrategyID
		}
		candidate, ok := s.strategyForFlow(deviceID, transport)
		if !ok {
			readErrors = append(readErrors, fmt.Sprintf("%s: strategy unavailable", transport))
			continue
		}
		reader, ok := candidate.(strategy.StateReader)
		if !ok {
			readErrors = append(readErrors, fmt.Sprintf("%s: state reader unavailable", transport))
			for _, descriptor := range profile.Properties {
				combined.Unavailable[descriptor.Name] = fmt.Sprintf("transport %s does not expose a state reader", transport)
			}
			continue
		}
		state, err := reader.ReadState(ctx)
		if err != nil {
			readErrors = append(readErrors, fmt.Sprintf("%s: %v", transport, err))
			for _, descriptor := range profile.Properties {
				combined.Unavailable[descriptor.Name] = fmt.Sprintf("transport %s failed: %v", transport, err)
			}
			continue
		}
		readCount++
		if readCount == 1 {
			combined.ForegroundPackage = state.ForegroundPackage
			combined.ScreenState = state.ScreenState
			combined.LockState = state.LockState
			combined.Orientation = state.Orientation
			combined.AutoRotate = state.AutoRotate
			combined.BatteryLevel = state.BatteryLevel
			combined.Charging = state.Charging
			combined.ThermalStatus = state.ThermalStatus
			combined.DisplayWidth = state.DisplayWidth
			combined.DisplayHeight = state.DisplayHeight
			combined.DisplayDensity = state.DisplayDensity
		}
		for name, value := range state.Properties {
			candidateDescriptor := descriptorFor(profile.Properties, name)
			currentDescriptor, already := propertySources[name]
			if already && currentDescriptor.StateClass == strategy.StateBearing && candidateDescriptor.StateClass != strategy.StateBearing {
				continue
			}
			if !already || value.Status == strategy.StatusAvailable || combined.Properties[name].Status != strategy.StatusAvailable {
				value.Transport = transport
				combined.Properties[name] = value
				propertySources[name] = candidateDescriptor
			}
		}
		for name, reason := range state.Unavailable {
			if _, available := combined.Properties[name]; !available {
				combined.Unavailable[name] = reason
			}
		}
	}
	if readCount == 0 {
		return combined, &strategy.AvailabilityError{Reason: fmt.Sprintf("no transport could read state for %q: %s", deviceID, strings.Join(readErrors, "; ")), NextAction: "Use a reachable transport that declares state-bearing properties."}
	}
	if len(combined.Unavailable) == 0 {
		combined.Unavailable = nil
	}
	return combined, nil
}

func descriptorFor(descriptors []strategy.PropertyDescriptor, name string) strategy.PropertyDescriptor {
	for _, descriptor := range descriptors {
		if descriptor.Name == name {
			return descriptor
		}
	}
	return strategy.PropertyDescriptor{Name: name, StateClass: strategy.StateBearing}
}

func (s *Service) loadTransportStates() error {
	if s.db == nil {
		return nil
	}
	profileRows, err := s.db.QueryContext(context.Background(), `SELECT device_id, serial, strategy_id, transport, endpoint, updated_at FROM device_control_transport_profiles`)
	if err != nil {
		return fmt.Errorf("load device transport profiles: %w", err)
	}
	for profileRows.Next() {
		var state transportState
		var updated string
		if err := profileRows.Scan(&state.DeviceID, &state.Serial, &state.StrategyID, &state.Transport, &state.Endpoint, &updated); err != nil {
			profileRows.Close()
			return fmt.Errorf("read device transport profile: %w", err)
		}
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.transportProfiles[transportProfileKey(state)] = state
		if state.DeviceID != "" {
			s.devices.UpsertIdentity(devicedomain.Record{ID: state.DeviceID, IdentityKey: persistedIdentityKey(state), IdentityKind: persistedIdentityKind(state), Kind: "physical", Serial: state.Serial, StrategyID: state.StrategyID, Transport: state.Transport, Endpoint: state.Endpoint, Transports: []strategy.DeviceTransport{{StrategyID: state.StrategyID, Name: state.Transport, Endpoint: state.Endpoint, Health: strategy.HealthUnreachable, HealthReason: "restored transport has not been probed yet", ObservedAt: state.UpdatedAt}}, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "restored transport has not been probed yet"})
		}
	}
	profileErr := profileRows.Err()
	profileRows.Close()
	if profileErr != nil {
		return profileErr
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT device_id, serial, strategy_id, transport, endpoint, updated_at FROM device_control_transports`)
	if err != nil {
		return fmt.Errorf("load device transports: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var state transportState
		var updated string
		if err := rows.Scan(&state.DeviceID, &state.Serial, &state.StrategyID, &state.Transport, &state.Endpoint, &updated); err != nil {
			return fmt.Errorf("read device transport: %w", err)
		}
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		s.transportStates[state.DeviceID] = state
		s.transportProfiles[transportProfileKey(state)] = state
		if state.Transport == "wireless" {
			s.devices.UpsertIdentity(devicedomain.Record{ID: state.DeviceID, IdentityKey: persistedIdentityKey(state), IdentityKind: persistedIdentityKind(state), Kind: "physical", Serial: state.Serial, StrategyID: state.StrategyID, Transport: state.Transport, Endpoint: state.Endpoint, Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "restored wireless transport has not been probed yet"})
		}
	}
	return rows.Err()
}

func persistedIdentityKey(state transportState) string {
	if state.StrategyID == "android-tv-remote" || state.StrategyID == "google-cast" {
		return ""
	}
	return state.Serial
}

func persistedIdentityKind(state transportState) string {
	switch state.StrategyID {
	case "android-adb":
		return string(identitydomain.ADBSerial)
	case "android-tv-remote":
		return string(identitydomain.BluetoothMAC)
	case "google-cast":
		return string(identitydomain.CastID)
	default:
		return ""
	}
}

func (s *Service) loadIdentityClaims() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT device_id, kind, value, strategy_id, evidence FROM device_control_identity_claims`)
	if err != nil {
		return fmt.Errorf("load device identity claims: %w", err)
	}
	defer rows.Close()
	grouped := map[string][]identitydomain.IdentityClaim{}
	for rows.Next() {
		var deviceID, kind, value, strategyID, evidence string
		if err := rows.Scan(&deviceID, &kind, &value, &strategyID, &evidence); err != nil {
			return fmt.Errorf("read device identity claim: %w", err)
		}
		claim, err := identitydomain.NewClaim(kind, value, strategyID, evidence)
		if err != nil {
			continue
		}
		grouped[deviceID] = append(grouped[deviceID], claim)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for deviceID, claims := range grouped {
		record, ok := s.devices.Get(deviceID)
		if !ok {
			record = devicedomain.Record{ID: deviceID, Kind: "physical", Status: strategy.HealthUnreachable, Health: strategy.HealthUnreachable, HealthReason: "identity claims restored before transport probe"}
		}
		record.Claims = claims
		if record.IdentityKey == "" {
			record.IdentityKey = claims[0].Value
			record.IdentityKind = string(claims[0].Kind)
		}
		s.devices.Upsert(record)
	}
	return nil
}

func (s *Service) loadIdentityMerges() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT canonical_id, member_id, claim_kind, claim_value, claim_strategy_id, claim_evidence, canonical_snapshot, member_snapshot FROM device_control_identity_merges`)
	if err != nil {
		return fmt.Errorf("load device identity merges: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var canonicalID, memberID, kind, value, strategyID, evidence, canonicalJSON, memberJSON string
		if err := rows.Scan(&canonicalID, &memberID, &kind, &value, &strategyID, &evidence, &canonicalJSON, &memberJSON); err != nil {
			return fmt.Errorf("read device identity merge: %w", err)
		}
		var canonical, member devicedomain.Record
		if err := json.Unmarshal([]byte(canonicalJSON), &canonical); err != nil {
			return fmt.Errorf("decode canonical identity snapshot: %w", err)
		}
		if err := json.Unmarshal([]byte(memberJSON), &member); err != nil {
			return fmt.Errorf("decode member identity snapshot: %w", err)
		}
		claim, err := identitydomain.NewClaim(kind, value, strategyID, evidence)
		if err != nil {
			return err
		}
		s.devices.RestoreMerge(canonicalID, devicedomain.MergeSnapshot{CanonicalBefore: canonical, Members: map[string]devicedomain.Record{memberID: member}, Claim: claim})
	}
	return rows.Err()
}

func (s *Service) loadIdentityAliases() error {
	if s.db == nil {
		return nil
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT canonical_id, alias_id FROM device_control_identity_aliases`)
	if err != nil {
		return fmt.Errorf("load device identity aliases: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var canonicalID, aliasID string
		if err := rows.Scan(&canonicalID, &aliasID); err != nil {
			return err
		}
		s.addAuditAlias(canonicalID, aliasID)
	}
	return rows.Err()
}

func (s *Service) persistIdentityClaims(ctx context.Context, record devicedomain.Record) error {
	if s.db == nil {
		return nil
	}
	for _, claim := range record.Claims {
		if err := identitydomain.ValidateClaim(claim); err != nil {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_claims (device_id, kind, value, strategy_id, evidence) VALUES (?, ?, ?, ?, ?) ON CONFLICT(device_id, kind, value) DO UPDATE SET strategy_id=excluded.strategy_id, evidence=excluded.evidence`, record.ID, claim.Kind, claim.Value, claim.StrategyID, claim.Evidence); err != nil {
			return fmt.Errorf("persist device identity claim: %w", err)
		}
	}
	return nil
}

func (s *Service) persistIdentityMerge(ctx context.Context, canonicalID string, snapshot devicedomain.MergeSnapshot) error {
	if s.db == nil {
		return nil
	}
	canonicalJSON, err := json.Marshal(snapshot.CanonicalBefore)
	if err != nil {
		return err
	}
	for memberID, member := range snapshot.Members {
		memberJSON, marshalErr := json.Marshal(member)
		if marshalErr != nil {
			return marshalErr
		}
		claim := snapshot.Claim
		if _, execErr := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_merges (canonical_id, member_id, claim_kind, claim_value, claim_strategy_id, claim_evidence, canonical_snapshot, member_snapshot, merged_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(canonical_id, member_id) DO UPDATE SET claim_kind=excluded.claim_kind, claim_value=excluded.claim_value, claim_strategy_id=excluded.claim_strategy_id, claim_evidence=excluded.claim_evidence, canonical_snapshot=excluded.canonical_snapshot, member_snapshot=excluded.member_snapshot, merged_at=excluded.merged_at`, canonicalID, memberID, claim.Kind, claim.Value, claim.StrategyID, claim.Evidence, string(canonicalJSON), string(memberJSON), time.Now().UTC().Format(time.RFC3339Nano)); execErr != nil {
			return execErr
		}
	}
	return nil
}

func (s *Service) persistIdentityAlias(ctx context.Context, canonicalID, aliasID string) error {
	s.addAuditAlias(canonicalID, aliasID)
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_control_identity_aliases (canonical_id, alias_id, created_at) VALUES (?, ?, ?) ON CONFLICT(canonical_id, alias_id) DO NOTHING`, canonicalID, aliasID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Service) addAuditAlias(canonicalID, aliasID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.auditAliases == nil {
		s.auditAliases = map[string]map[string]struct{}{}
	}
	if s.auditAliases[canonicalID] == nil {
		s.auditAliases[canonicalID] = map[string]struct{}{}
	}
	s.auditAliases[canonicalID][canonicalID] = struct{}{}
	s.auditAliases[canonicalID][aliasID] = struct{}{}
}

func (s *Service) restoreTransportStrategies() {
	for id, state := range s.transportStates {
		if state.Transport != "wireless" || state.Endpoint == "" {
			continue
		}
		base, ok := s.registry.Get(state.StrategyID)
		if !ok {
			continue
		}
		if scoped, ok := base.(strategy.DeviceScoped); ok && state.Serial != "" {
			base = scoped.ForDevice(state.Serial)
		}
		restorer, ok := base.(interface {
			RestoreWireless(string) strategy.Strategy
		})
		if !ok {
			continue
		}
		s.transportStrategies[id] = restorer.RestoreWireless(state.Endpoint)
	}
}

func (s *Service) persistTransportState(ctx context.Context, state transportState) error {
	if s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO device_control_transport_profiles (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id, strategy_id, transport) DO UPDATE SET serial=excluded.serial, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist device transport: %w", err)
	}
	// Keep the legacy selected-transport row for older local databases and
	// wireless strategy restoration; the profile table above is the durable
	// many-transport source of truth.
	_, err = s.db.ExecContext(ctx, `INSERT INTO device_control_transports (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id) DO UPDATE SET serial=excluded.serial, strategy_id=excluded.strategy_id, transport=excluded.transport, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist selected device transport: %w", err)
	}
	s.transportProfiles[transportProfileKey(state)] = state
	return nil
}

// persistObservedTransportProfiles records every transport discovered during
// inventory without changing the selected-transport row. A device can expose
// several independently reachable strategies at once, so writing only the
// selected transport would make the identity merge disappear on restart.
func (s *Service) persistObservedTransportProfiles(ctx context.Context, record devicedomain.Record) error {
	if s.db == nil {
		return nil
	}
	for _, profile := range record.Transports {
		strategyID := strings.TrimSpace(profile.StrategyID)
		name := strings.TrimSpace(profile.Name)
		if strategyID == "" && name == "" {
			continue
		}
		if strategyID == "" {
			strategyID = strings.TrimSpace(record.StrategyID)
		}
		if name == "" {
			name = strategyID
		}
		observedAt := profile.ObservedAt
		if observedAt.IsZero() {
			observedAt = record.ObservedAt
		}
		if observedAt.IsZero() {
			observedAt = time.Now().UTC()
		}
		state := transportState{
			DeviceID:   record.ID,
			Serial:     record.Serial,
			StrategyID: strategyID,
			Transport:  name,
			Endpoint:   profile.Endpoint,
			UpdatedAt:  observedAt,
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_transport_profiles (device_id, serial, strategy_id, transport, endpoint, updated_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(device_id, strategy_id, transport) DO UPDATE SET serial=excluded.serial, endpoint=excluded.endpoint, updated_at=excluded.updated_at`, state.DeviceID, state.Serial, state.StrategyID, state.Transport, state.Endpoint, state.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
			return fmt.Errorf("persist observed device transport: %w", err)
		}
		s.transportProfiles[transportProfileKey(state)] = state
	}
	return nil
}

func transportProfileKey(state transportState) string {
	return state.DeviceID + "\x00" + state.StrategyID + "\x00" + state.Transport
}

func NewWithDBAndAttached(registry *strategyregistry.Registry, db routedDB, reader AttachedReader, roots ...*filerouting.RoutedRoots) (*Service, error) {
	s, err := NewWithDB(registry, db, roots...)
	if err != nil {
		return nil, err
	}
	s.attached = reader
	return s, nil
}

func (s *Service) Acquire(deviceID, actor string, ttl time.Duration) (Session, error) {
	return s.AcquireContext(context.Background(), deviceID, actor, ttl)
}

func (s *Service) AcquireContext(ctx context.Context, deviceID, actor string, ttl time.Duration) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, old := range s.sessions {
		if old.State == "held" && now.After(old.ExpiresAt) {
			old.State = "expired"
			s.sessions[id] = old
		}
	}
	for _, old := range s.sessions {
		if old.DeviceID == deviceID && old.State == "held" {
			return Session{}, fmt.Errorf("device %s already has a held lease owned by %q until %s", deviceID, old.Actor, old.ExpiresAt.Format(time.RFC3339))
		}
	}
	if _, ok := s.strategyForDevice(deviceID); !ok {
		return Session{}, fmt.Errorf("unknown device %q", deviceID)
	}
	if ttl <= 0 || ttl > 30*time.Minute {
		ttl = 10 * time.Minute
	}
	sess := Session{ID: uuid.NewString(), DeviceID: deviceID, Actor: actor, State: "held", LeaseToken: uuid.NewString(), ExpiresAt: now.Add(ttl), CreatedAt: now}
	if s.db != nil {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO device_control_sessions (id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.DeviceID, sess.Actor, sess.State, sess.LeaseToken, sess.KillReason, sess.ExpiresAt.Format(time.RFC3339Nano), sess.CreatedAt.Format(time.RFC3339Nano)); err != nil {
			return Session{}, fmt.Errorf("persist lease: %w", err)
		}
	}
	s.sessions[sess.ID] = sess
	return sess, nil
}

func (s *Service) ListSessions() []Session {
	return s.ListSessionsContext(context.Background())
}

func (s *Service) ListSessionsContext(ctx context.Context) []Session {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		timeout := s.sessionQueryTimeout
		if timeout <= 0 {
			timeout = 750 * time.Millisecond
		}
		queryCtx, cancel := context.WithTimeout(ctx, timeout)
		rows, err := s.db.QueryContext(queryCtx, `SELECT id, device_id, actor, state, lease_token, kill_reason, expires_at, created_at FROM device_control_sessions ORDER BY created_at DESC`)
		if err == nil {
			defer cancel()
			defer rows.Close()
			out := make([]Session, 0)
			for rows.Next() {
				var v Session
				var expires, created string
				if err := rows.Scan(&v.ID, &v.DeviceID, &v.Actor, &v.State, &v.LeaseToken, &v.KillReason, &expires, &created); err != nil {
					continue
				}
				v.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				out = append(out, v)
			}
			if err := rows.Err(); err == nil {
				return out
			}
		} else {
			cancel()
		}
	}
	out := make([]Session, 0, len(s.sessions))
	for _, v := range s.sessions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// ListLiveSessions returns only leases that can currently authorize device
// work. Historical released, killed, and expired sessions remain available to
// ListSessions for audit/reconstruction, but must not appear in operator live
// lease surfaces.
func (s *Service) ListLiveSessions() []Session {
	return s.ListLiveSessionsContext(context.Background())
}

func (s *Service) ListLiveSessionsContext(ctx context.Context) []Session {
	now := time.Now()
	live := make([]Session, 0)
	for _, session := range s.ListSessionsContext(ctx) {
		if session.State == "held" && now.Before(session.ExpiresAt) {
			live = append(live, session)
		}
	}
	return live
}

func (s *Service) Kill(id, reason string) (Session, error) {
	return s.KillContext(context.Background(), id, reason)
}

func (s *Service) KillContext(ctx context.Context, id, reason string) (Session, error) {
	return s.finishContext(ctx, id, "killed", reason)
}

func (s *Service) Release(id string) (Session, error) {
	return s.ReleaseContext(context.Background(), id)
}

func (s *Service) ReleaseContext(ctx context.Context, id string) (Session, error) {
	return s.finishContext(ctx, id, "released", "")
}

func (s *Service) sessionForLease(_ context.Context, deviceID, token string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, session := range s.sessions {
		if session.LeaseToken == token {
			if session.DeviceID != deviceID {
				return Session{}, fmt.Errorf("lease token is bound to device %q", session.DeviceID)
			}
			if session.State != "held" {
				return Session{}, fmt.Errorf("lease is %s", session.State)
			}
			if now.After(session.ExpiresAt) {
				return Session{}, fmt.Errorf("lease expired at %s", session.ExpiresAt.Format(time.RFC3339))
			}
			return session, nil
		}
	}
	return Session{}, fmt.Errorf("lease token is invalid or no longer held")
}

// ValidateLease is the narrow public validation seam for long-running
// delivery ramps. It deliberately returns no session or token material.
func (s *Service) ValidateLease(ctx context.Context, deviceID, token string) error {
	_, err := s.sessionForLease(ctx, deviceID, token)
	return err
}

func (s *Service) finishContext(ctx context.Context, id, state, reason string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[id]
	if !ok {
		return Session{}, fmt.Errorf("session %q not found", id)
	}
	v.State = state
	v.KillReason = reason
	v.LeaseToken = ""
	if s.db != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE device_control_sessions SET state = ?, kill_reason = ?, lease_token = ? WHERE id = ?`, v.State, v.KillReason, v.LeaseToken, v.ID); err != nil {
			return Session{}, fmt.Errorf("persist session state: %w", err)
		}
	}
	s.sessions[id] = v
	if cancel := s.activeCancels[id]; cancel != nil {
		cancel()
		delete(s.activeCancels, id)
	}
	return v, nil
}

func (s *Service) Audit() []Audit {
	return s.AuditContext(context.Background())
}

func (s *Service) AuditContext(ctx context.Context) []Audit {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		rows, err := s.db.QueryContext(ctx, `SELECT id, actor, device_id, transport, causation_id, lease_id, verb, outcome, profile_id, method, attempts, provider_state, before_lock_state, after_lock_state, created_at, redaction_verified, redaction_opted_out, interactive, evidence_backed FROM device_control_audits ORDER BY created_at DESC`)
		if err == nil {
			defer rows.Close()
			out := make([]Audit, 0)
			for rows.Next() {
				var v Audit
				var created string
				var verified, optedOut, interactive, evidenceBacked int
				if err := rows.Scan(&v.ID, &v.Actor, &v.DeviceID, &v.Transport, &v.CausationID, &v.LeaseID, &v.Verb, &v.Outcome, &v.ProfileID, &v.Method, &v.Attempts, &v.ProviderState, &v.BeforeLockState, &v.AfterLockState, &created, &verified, &optedOut, &interactive, &evidenceBacked); err != nil {
					continue
				}
				v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
				v.RedactionVerified = verified != 0
				v.RedactionOptedOut = optedOut != 0
				v.Interactive = interactive != 0
				v.EvidenceBacked = evidenceBacked != 0
				out = append(out, v)
			}
			return out
		}
	}
	return append([]Audit{}, s.audits...)
}
