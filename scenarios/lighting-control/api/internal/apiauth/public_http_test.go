package apiauth_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scheduletest"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control/controlconnect"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy/policyconnect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	controlH "lighting-control/handlers/control"
	policyH "lighting-control/handlers/policy"
	internalcontrol "lighting-control/internal/control"
	internalpolicy "lighting-control/internal/policy"
)

type observedBackend struct {
	calls        atomic.Int64
	writeCalls   atomic.Int64
	discoverErr  error
	configurable bool
	count        uint32
	emittedOff   bool
}

func (b *observedBackend) ProtocolVersion(context.Context) (uint32, error) {
	b.calls.Add(1)
	return 6, nil
}
func (b *observedBackend) Discover(context.Context) ([]internalcontrol.SDKController, error) {
	b.calls.Add(1)
	if b.discoverErr != nil {
		return nil, b.discoverErr
	}
	zone := internalcontrol.SDKZone{Name: "Fixture zone", LEDCount: 2, Index: 0, Static: true}
	if b.configurable {
		zone.LEDCount = b.count
		zone.LEDMin = 0
		zone.LEDMax = 500
		zone.Flags = 2
	}
	return []internalcontrol.SDKController{{ID: 7, Name: "Fixture board", Vendor: "Fixture", Serial: "private-http-fixture", Zones: []internalcontrol.SDKZone{zone}}}, nil
}
func (b *observedBackend) SendZoneColors(context.Context, uint32, uint32, []internalcontrol.RGB) (internalcontrol.SDKCommandResult, error) {
	b.calls.Add(1)
	b.writeCalls.Add(1)
	return internalcontrol.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

type fixtureInputs struct{ now time.Time }

func (f fixtureInputs) Read(context.Context) internalpolicy.Inputs {
	return internalpolicy.Inputs{Now: f.now}
}

type credentialProvider struct {
	principal identity.Principal
	invalid   bool
}

func (credentialProvider) Source() identity.AuthSource { return identity.SourceScenarioAuthenticator }
func (p credentialProvider) VerifyRequest(_ context.Context, r *http.Request) (identity.Principal, error) {
	if r.Header.Get("Authorization") == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	if p.invalid {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	return p.principal, nil
}

type fixture struct {
	db                   *sql.DB
	backend              *observedBackend
	control              controlconnect.ControlServiceClient
	policy               policyconnect.PolicyServiceClient
	controllerID, zoneID string
	credential           bool
	headers              http.Header
	controlDomain        *internalcontrol.Service
}

func newFixture(t *testing.T, principal identity.Principal, credential, invalid bool, providers ...authn.Provider) *fixture {
	return newFixtureWrapped(t, principal, credential, invalid, func(h http.Handler) http.Handler { return h }, providers...)
}

func newFixtureWrapped(t *testing.T, principal identity.Principal, credential, invalid bool, wrap func(http.Handler) http.Handler, providers ...authn.Provider) *fixture {
	t.Helper()
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(internalcontrol.Schema), database.SchemaProviderFunc(internalpolicy.Schema)); err != nil {
		t.Fatal(err)
	}
	if err := internalcontrol.EnsureIdentitySource(ctx, db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	clock := scheduletest.New(now)
	backend := &observedBackend{}
	controlService := internalcontrol.NewService(internalcontrol.NewSQLiteRepository(db, clock), backend, clock)
	_, controllers, err := controlService.ListControllers(ctx)
	if err != nil || len(controllers) != 1 {
		t.Fatalf("seed discovery: %+v %v", controllers, err)
	}
	id, zone := controllers[0].ControllerId, controllers[0].Zones[0].ZoneId
	if _, err := controlService.Enroll(ctx, id, true); err != nil {
		t.Fatal(err)
	}
	if _, err := controlService.SetZoneState(ctx, zoneState(id, zone)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := controlService.SetManualOff(ctx, true, nil, nil); err != nil {
		t.Fatal(err)
	}
	policyService := policyH.NewService(internalpolicy.NewSQLiteStore(db), fixtureInputs{now}, controlService)
	if _, err := policyService.SetConfig(ctx, internalpolicy.DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := policyService.SetManual(ctx, policyv1.PresetKind_PRESET_KIND_OFF, time.Time{}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	cp, ch := controlconnect.NewControlServiceHandler(controlH.NewConnectHandler(controlService, nil))
	pp, ph := policyconnect.NewPolicyServiceHandler(policyH.NewConnectHandler(policyService, nil))
	mux.Handle(cp, ch)
	mux.Handle(pp, ph)
	// Use production's passive attribution middleware: missing or invalid proof
	// reaches the public handlers and must be refused there before domain calls.
	config := authn.Config{}
	if credential {
		config.Providers = []authn.Provider{credentialProvider{principal, invalid}}
	}
	if len(providers) > 0 {
		config.Providers = providers
	}
	server := httptest.NewServer(wrap(authn.Middleware(config)(mux)))
	t.Cleanup(server.Close)
	backend.calls.Store(0)
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Transport: transport}
	return &fixture{db, backend, controlconnect.NewControlServiceClient(client, server.URL), policyconnect.NewPolicyServiceClient(client, server.URL), id, zone, credential, nil, controlService}
}
func zoneState(id, zone string) *controlv1.ZoneState {
	return &controlv1.ZoneState{ControllerId: id, ZoneId: zone, DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 11, Green: 22, Blue: 33}, BrightnessPercent: 80}}}
}
func request[T any](f *fixture, message *T) *connect.Request[T] {
	r := connect.NewRequest(message)
	for k, values := range f.headers {
		for _, value := range values {
			r.Header().Add(k, value)
		}
	}
	if f.credential {
		r.Header().Set("Authorization", "Bearer private-fixture")
	}
	return r
}

type mutation struct {
	service, method string
	call            func(context.Context, *fixture) error
}

var publicMutations = []mutation{
	{"control", "ConfigureZone", func(ctx context.Context, f *fixture) error {
		_, err := f.control.ConfigureZone(ctx, request(f, &controlv1.ConfigureZoneRequest{ControllerId: f.controllerID, ZoneId: f.zoneID, LedCount: 1}))
		// The fixed-zone fixture has no size operation. Reaching its domain
		// refusal proves admission; permission failures remain errors.
		if connect.CodeOf(err) == connect.CodeUnimplemented {
			return nil
		}
		return err
	}},
	{"control", "EnrollController", func(ctx context.Context, f *fixture) error {
		_, err := f.control.EnrollController(ctx, request(f, &controlv1.EnrollControllerRequest{ControllerId: f.controllerID, Enrolled: false}))
		return err
	}},
	{"control", "SetZoneState", func(ctx context.Context, f *fixture) error {
		_, err := f.control.SetZoneState(ctx, request(f, &controlv1.SetZoneStateRequest{Desired: zoneState(f.controllerID, f.zoneID)}))
		return err
	}},
	{"control", "SetManualOff", func(ctx context.Context, f *fixture) error {
		_, err := f.control.SetManualOff(ctx, request(f, &controlv1.SetManualOffRequest{AllEnrolled: true}))
		return err
	}},
	{"control", "ClearManualOverride", func(ctx context.Context, f *fixture) error {
		_, err := f.control.ClearManualOverride(ctx, request(f, &controlv1.ClearManualOverrideRequest{}))
		return err
	}},
	{"control", "Reconcile", func(ctx context.Context, f *fixture) error {
		_, err := f.control.Reconcile(ctx, request(f, &controlv1.ReconcileRequest{}))
		return err
	}},
	{"policy", "SetPolicy", func(ctx context.Context, f *fixture) error {
		config := internalpolicy.DefaultConfig()
		config.Presets[0].Label = "Authorized configuration"
		_, err := f.policy.SetPolicy(ctx, request(f, &policyv1.SetPolicyRequest{Config: config}))
		return err
	}},
	{"policy", "SetManualPreset", func(ctx context.Context, f *fixture) error {
		_, err := f.policy.SetManualPreset(ctx, request(f, &policyv1.SetManualPresetRequest{Preset: policyv1.PresetKind_PRESET_KIND_OFF}))
		return err
	}},
	{"policy", "ClearManualPreset", func(ctx context.Context, f *fixture) error {
		_, err := f.policy.ClearManualPreset(ctx, request(f, &policyv1.ClearManualPresetRequest{}))
		return err
	}},
	{"policy", "EvaluatePolicy", func(ctx context.Context, f *fixture) error {
		_, err := f.policy.EvaluatePolicy(ctx, request(f, &policyv1.EvaluatePolicyRequest{}))
		return err
	}},
}

func databaseSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	var snapshot strings.Builder
	for _, name := range names {
		rows, err := db.Query(`SELECT * FROM "` + strings.ReplaceAll(name, `"`, `""`) + `" ORDER BY rowid`)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintln(&snapshot, name)
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			snapshot.Write(encoded)
			snapshot.WriteByte('\n')
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if err := rows.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return snapshot.String()
}

func TestEveryPublicMutationDeniesUnqualifiedHTTPIdentityBeforeEffects(t *testing.T) {
	now := time.Now()
	base := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"lighting-control:write"}, ExpiresAt: now.Add(time.Hour)}
	for _, tc := range []struct {
		name                string
		credential, invalid bool
		change              func(*identity.Principal)
	}{
		{"anonymous", false, false, nil}, {"invalid credential", true, true, nil},
		{"unverified service", true, false, func(p *identity.Principal) { p.Kind = identity.ActorService; p.Verified = false }},
		{"expired", true, false, func(p *identity.Principal) { p.ExpiresAt = now.Add(-time.Hour) }},
		{"read only", true, false, func(p *identity.Principal) { p.Scopes = []string{"lighting-control:read"} }},
		{"other scenario", true, false, func(p *identity.Principal) { p.Scopes = []string{"other-scenario:write"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			if tc.change != nil {
				tc.change(&p)
			}
			f := newFixture(t, p, tc.credential, tc.invalid)
			for _, operation := range publicMutations {
				t.Run(operation.method, func(t *testing.T) {
					before := databaseSnapshot(t, f.db)
					calls := f.backend.calls.Load()
					err := operation.call(context.Background(), f)
					if err == nil || connect.CodeOf(err) != connect.CodePermissionDenied {
						t.Fatalf("unqualified mutation reached domain: %v", err)
					}
					if after := databaseSnapshot(t, f.db); before != after {
						t.Fatal("denied HTTP request changed durable state")
					}
					if f.backend.calls.Load() != calls {
						t.Fatal("denied HTTP request called lighting backend")
					}
				})
			}
		})
	}
}

func TestEveryPublicMutationAcceptsExistingScopedAndWildcardOwnerAuthority(t *testing.T) {
	for _, actor := range []struct {
		name   string
		kind   identity.ActorKind
		scopes []string
	}{
		{"human", identity.ActorHuman, []string{"lighting-control:write"}},
		{"service", identity.ActorService, []string{"lighting-control:write"}},
		{"canonical wildcard", identity.ActorHuman, []string{"lighting-control:*"}},
	} {
		for _, operation := range publicMutations {
			t.Run(actor.name+"/"+operation.method, func(t *testing.T) {
				p := identity.Principal{Kind: actor.kind, Subject: "owner", Owner: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: actor.scopes, ExpiresAt: time.Now().Add(time.Hour)}
				f := newFixture(t, p, true, false)
				if err := operation.call(context.Background(), f); err != nil {
					t.Fatalf("existing authorized %s denied: %v", actor.kind, err)
				}
			})
		}
	}
}

func TestMutationHTTPRegressionsCoverAllNonReadRPCs(t *testing.T) {
	got := map[string][]string{}
	for _, operation := range publicMutations {
		got[operation.service] = append(got[operation.service], operation.method)
	}
	for _, service := range []struct {
		name  string
		file  protoreflect.FileDescriptor
		reads map[string]bool
	}{
		{"control", controlv1.File_lighting_control_v1_control_control_proto, map[string]bool{"GetBackendStatus": true, "ListControllers": true, "GetDesiredState": true}},
		{"policy", policyv1.File_lighting_control_v1_policy_policy_proto, map[string]bool{"GetPolicy": true}},
	} {
		var want []string
		methods := service.file.Services().Get(0).Methods()
		for i := 0; i < methods.Len(); i++ {
			name := string(methods.Get(i).Name())
			if !service.reads[name] {
				want = append(want, name)
			}
		}
		slices.Sort(want)
		observed := slices.Clone(got[service.name])
		slices.Sort(observed)
		if !slices.Equal(observed, want) {
			t.Fatalf("%s public mutation coverage changed: got%v want%v", service.name, got[service.name], want)
		}
	}
}

func TestAnonymousManualOffDoesNotCreateDurableOverride(t *testing.T) {
	f := newFixture(t, identity.Principal{}, false, false)
	if _, err := f.db.Exec(`DELETE FROM lighting_manual_override`); err != nil {
		t.Fatal(err)
	}
	before := databaseSnapshot(t, f.db)
	_, err := f.control.SetManualOff(context.Background(), request(f, &controlv1.SetManualOffRequest{AllEnrolled: true}))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("anonymous Manual Off accepted: %v", err)
	}
	var rows int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM lighting_manual_override`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || databaseSnapshot(t, f.db) != before || f.backend.calls.Load() != 0 {
		t.Fatalf("anonymous Manual Off left effects: overrides=%d backend=%d", rows, f.backend.calls.Load())
	}
}

func (b *observedBackend) ApplyStableZone(ctx context.Context, _, _ string, d *controlv1.ZoneState) (internalcontrol.SDKCommandResult, error) {
	return b.ApplyConfiguredStableZone(ctx, "", "", 0, nil, d)
}
func (b *observedBackend) ApplyConfiguredStableZone(_ context.Context, _, _ string, count uint32, _ map[string]uint32, d *controlv1.ZoneState) (internalcontrol.SDKCommandResult, error) {
	b.calls.Add(1)
	if b.discoverErr != nil {
		return internalcontrol.SDKCommandResult{}, b.discoverErr
	}
	b.writeCalls.Add(1)
	if b.configurable {
		b.count = count
	}
	b.emittedOff = d.GetOff() != nil
	return internalcontrol.SDKCommandResult{LocalWriteCompleted: true, BackendAcknowledged: true}, nil
}

func TestPublicZoneConfigurationPersistsLengthAndPreservesManualOff(t *testing.T) {
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Owner: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"lighting-control:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	f := newFixture(t, p, true, false)
	f.backend.configurable = true
	ctx := context.Background()
	response, err := f.control.ConfigureZone(ctx, request(f, &controlv1.ConfigureZoneRequest{ControllerId: f.controllerID, ZoneId: f.zoneID, LedCount: 24}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.Zone.ConfiguredLedCount != 24 || response.Msg.Zone.SetupNeeded || len(response.Msg.Results) != 1 {
		t.Fatalf("configuration result: %+v", response.Msg)
	}
	_, err = f.control.SetManualOff(ctx, request(f, &controlv1.SetManualOffRequest{AllEnrolled: true}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.control.ConfigureZone(ctx, request(f, &controlv1.ConfigureZoneRequest{ControllerId: f.controllerID, ZoneId: f.zoneID, LedCount: 32}))
	if err != nil {
		t.Fatal(err)
	}
	if !f.backend.emittedOff || f.backend.count != 32 {
		t.Fatalf("length update lost manual Off: %+v", f.backend)
	}
	inventory, err := f.control.ListControllers(ctx, request(f, &controlv1.ListControllersRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if inventory.Msg.Controllers[0].Zones[0].ZoneId != f.zoneID {
		t.Fatal("configuration rebound legacy zone identity")
	}
}

func TestPublicOfflineInventoryKeepsSavedWiringAndOffThroughReconnect(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Owner: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"lighting-control:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	f := newFixture(t, principal, true, false)
	ctx := context.Background()
	before, err := f.control.ListControllers(ctx, request(f, &controlv1.ListControllersRequest{IncludeUnavailable: true}))
	if err != nil {
		t.Fatal(err)
	}
	saved := before.Msg.Controllers[0]
	writes := f.backend.writeCalls.Load()
	f.backend.discoverErr = internalcontrol.ErrTransport
	// A failed probe must not publish a new inventory observation.
	if _, err := f.db.Exec(`CREATE TRIGGER reject_offline_discovery BEFORE INSERT ON lighting_controllers BEGIN SELECT RAISE(ABORT,'offline discovery cannot sync'); END`); err != nil {
		t.Fatal(err)
	}
	current, err := f.control.ListControllers(ctx, request(f, &controlv1.ListControllersRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(current.Msg.Controllers) != 0 {
		t.Fatal("failed discovery presented current controllers")
	}
	offline, err := f.control.ListControllers(ctx, request(f, &controlv1.ListControllersRequest{IncludeUnavailable: true}))
	if err != nil {
		t.Fatal(err)
	}
	if offline.Msg.Backend.State == controlv1.BackendState_BACKEND_STATE_READY || offline.Msg.Backend.Detail != internalcontrol.ErrTransport.Error() {
		t.Fatalf("discovery error hidden: %+v", offline.Msg.Backend)
	}
	if len(offline.Msg.Controllers) != 1 || !proto.Equal(saved, offline.Msg.Controllers[0]) {
		t.Fatalf("saved roster changed during failure: %+v", offline.Msg.Controllers)
	}
	probe, err := f.control.GetBackendStatus(ctx, request(f, &controlv1.GetBackendStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if probe.Msg.Backend.State != controlv1.BackendState_BACKEND_STATE_READY {
		t.Fatal("fixture must distinguish protocol readiness from failed inventory")
	}
	off, err := f.control.SetManualOff(ctx, request(f, &controlv1.SetManualOffRequest{AllEnrolled: true}))
	if err != nil {
		t.Fatal(err)
	}
	if !off.Msg.Override.Active {
		t.Fatal("offline Off intent was not retained")
	}
	changed, err := f.control.SetZoneState(ctx, request(f, &controlv1.SetZoneStateRequest{Desired: zoneState(f.controllerID, f.zoneID)}))
	if err != nil {
		t.Fatal(err)
	}
	if changed.Msg.Result.Operation != controlv1.OperationState_OPERATION_STATE_DEFERRED || !strings.Contains(changed.Msg.Result.Reason, "higher priority") {
		t.Fatal("saved Off lost manual priority")
	}
	if f.backend.writeCalls.Load() != writes {
		t.Fatal("offline inspection or requests wrote stale capabilities")
	}
	if _, err := f.db.Exec(`DROP TRIGGER reject_offline_discovery`); err != nil {
		t.Fatal(err)
	}
	f.backend.discoverErr = nil
	if _, err := f.control.Reconcile(ctx, request(f, &controlv1.ReconcileRequest{})); err != nil {
		t.Fatal(err)
	}
	if f.backend.writeCalls.Load() <= writes || !f.backend.emittedOff {
		t.Fatal("reconnect did not reconcile saved Off")
	}
	desired, err := f.control.GetDesiredState(ctx, request(f, &controlv1.GetDesiredStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if !desired.Msg.ManualOverride.Active {
		t.Fatal("reconnect cleared durable Off")
	}
}
