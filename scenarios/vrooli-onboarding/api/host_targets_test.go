package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/scopecatalog"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/permissions"
)

type targetProbeReacher struct {
	request nodereach.ScenarioRequest
	err     error
}

func (f *targetProbeReacher) CallScenario(_ context.Context, request nodereach.ScenarioRequest) ([]byte, error) {
	f.request = request
	return nil, f.err
}

func TestBridgeClientConfigUsesTheEnrolledLocalOperatorSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", dir)
	private, err := operatorsession.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	store, err := operatorsession.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(private, operatorsession.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: "scenario-authenticator",
		Mode:             operatorsession.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Now().UTC(),
		ScopeCeiling:     []string{"vrooli-bridge:read"},
	}); err != nil {
		t.Fatal(err)
	}

	config := bridgeClientConfig()
	if config.TokenProvider == nil {
		t.Fatal("bridge target discovery must provide the enrolled local operator session")
	}
	token, err := config.TokenProvider(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, operatorsession.LocalSessionScheme+" ") {
		t.Fatalf("token scheme = %q, want %q", token, operatorsession.LocalSessionScheme)
	}
}

func TestNodeToTargetOnlyProvidesRecoveryActionWhenUnavailable(t *testing.T) {
	online := nodeToTarget(&registryv1.Node{
		Id:                    "node-online",
		Name:                  "minimouse",
		Os:                    "darwin",
		Arch:                  "amd64",
		Status:                registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Online:                true,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
	}, []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}, nil, []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}, nil)
	if !online.Available || online.NextAction != "" {
		t.Fatalf("online target = %+v, want available without next action", online)
	}

	offline := nodeToTarget(&registryv1.Node{
		Id:     "node-offline",
		Name:   "offline-node",
		Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE,
	}, nil, nil, []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}, nil)
	if offline.NextAction == "" {
		t.Fatal("offline target should include a recovery action")
	}
}

func TestNodeToTargetRejectsConnectedNodeWithoutOnboardingScopes(t *testing.T) {
	target := nodeToTarget(&registryv1.Node{
		Id:                    "node-read-only",
		Name:                  "read-only-node",
		Status:                registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Online:                true,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
	}, []string{"system-monitor:read"}, nil, []string{"vrooli-onboarding:read", "vrooli-onboarding:write"}, nil)
	if target.Available {
		t.Fatal("node without onboarding scopes must not be advertised as available")
	}
	if !strings.Contains(target.Reason, "vrooli-onboarding:read") || target.NextAction == "" {
		t.Fatalf("target = %+v, want missing-scope reason and remediation", target)
	}
}

func TestRequiredOnboardingScopesFollowOwnerAuthenticationProfile(t *testing.T) {
	catalog := scopecatalog.Catalog{AuthenticationProfiles: []scopecatalog.AuthenticationProfile{{
		Scenario: "vrooli-onboarding", Declared: true,
		Capabilities: []scopecatalog.Capability{
			{ID: "onboarding:read", Effect: scopecatalog.EffectRead},
			{ID: "onboarding:write", Effect: scopecatalog.EffectWrite},
			{ID: "onboarding:destructive", Effect: scopecatalog.EffectDestructive},
		},
	}}}
	required, err := requiredOnboardingScopesFromCatalog(catalog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(required, ",") != "onboarding:read,onboarding:write" {
		t.Fatalf("required scopes = %v", required)
	}
	if missing := missingRequiredScopes([]string{"onboarding:*"}, required); len(missing) != 0 {
		t.Fatalf("catalog wildcard should satisfy requirements, missing=%v", missing)
	}
}

func TestRequiredOnboardingScopesFailClosedForInvalidOwnerProfile(t *testing.T) {
	for _, profile := range []scopecatalog.AuthenticationProfile{
		{Scenario: "other", Declared: true, Capabilities: []scopecatalog.Capability{{ID: "other:read", Effect: scopecatalog.EffectRead}, {ID: "other:write", Effect: scopecatalog.EffectWrite}}},
		{Scenario: "vrooli-onboarding", Declared: false, Capabilities: []scopecatalog.Capability{{ID: "onboarding:read", Effect: scopecatalog.EffectRead}, {ID: "onboarding:write", Effect: scopecatalog.EffectWrite}}},
		{Scenario: "vrooli-onboarding", Declared: true, MissingDeclarations: []string{"profile mismatch"}, Capabilities: []scopecatalog.Capability{{ID: "onboarding:read", Effect: scopecatalog.EffectRead}, {ID: "onboarding:write", Effect: scopecatalog.EffectWrite}}},
		{Scenario: "vrooli-onboarding", Declared: true, Capabilities: []scopecatalog.Capability{{ID: "onboarding:read", Effect: scopecatalog.EffectRead}}},
	} {
		if _, err := requiredOnboardingScopesFromCatalog(scopecatalog.Catalog{AuthenticationProfiles: []scopecatalog.AuthenticationProfile{profile}}); err == nil {
			t.Fatalf("invalid profile accepted: %+v", profile)
		}
	}
}

func TestRequiredOnboardingScopesComeFromCurrentScenarioManifest(t *testing.T) {
	required, err := requiredOnboardingTargetScopes()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(required, ",") != "vrooli-onboarding:read,vrooli-onboarding:write" {
		t.Fatalf("source-declared requirements = %v", required)
	}
}

func TestNodeToTargetFailsClosedWhenTypedScopeReadOrCatalogFails(t *testing.T) {
	node := &registryv1.Node{Id: "node-1", Name: "target", Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, Online: true, Dispatchable: true}
	for _, tc := range []struct {
		name           string
		scopeErr       error
		requirementErr error
	}{
		{name: "scope read", scopeErr: errors.New("stale bridge source")},
		{name: "scope catalog", requirementErr: errors.New("invalid manifest")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := nodeToTarget(node, nil, tc.scopeErr, []string{"onboarding:read", "onboarding:write"}, tc.requirementErr)
			if got.Available || got.NextAction == "" {
				t.Fatalf("target = %+v, want unavailable with remediation", got)
			}
		})
	}
}

func TestReadBridgeNodeScopesUsesExactRegistryTargetAndRevision(t *testing.T) {
	target := permissions.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7}
	descriptor := permissions.Descriptor{
		Version: permissions.ContractVersion, ID: "bridge/node-scopes/machine-1/node-1", Owner: "vrooli-bridge",
		PrincipalKind: string(identity.ActorHuman), PrincipalID: "operator-1", Scope: "vrooli-bridge:write",
		SubjectKind: "node-scopes", SubjectID: "node-1", TargetBound: true, Target: target,
		Source: "Bridge NodeRegistryService + MachineService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator",
	}
	value, _ := json.Marshal(map[string]any{"scopes": []string{"onboarding:read", "onboarding:write"}})
	provider := &targetScopeRegistryFixture{descriptor: descriptor, value: value}
	registry, err := permissions.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true})
	node := &registryv1.Node{Id: "node-1", EnrollmentGeneration: 7}
	got, err := readBridgeNodeScopes(ctx, registry, []permissions.Descriptor{descriptor}, node)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "onboarding:read,onboarding:write" || provider.reads != 1 {
		t.Fatalf("scopes=%v registry reads=%d", got, provider.reads)
	}
}

func TestReadBridgeNodeScopesRefusesStaleGenerationBeforeOwnerRead(t *testing.T) {
	descriptor := permissions.Descriptor{
		Version: permissions.ContractVersion, ID: "bridge/node-scopes/machine-1/node-1", Owner: "vrooli-bridge",
		PrincipalKind: string(identity.ActorHuman), PrincipalID: "operator-1", Scope: "vrooli-bridge:write",
		SubjectKind: "node-scopes", SubjectID: "node-1", TargetBound: true,
		Target: permissions.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7},
		Source: "Bridge NodeRegistryService + MachineService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator",
	}
	provider := &targetScopeRegistryFixture{descriptor: descriptor, value: []byte(`{"scopes":["onboarding:read","onboarding:write"]}`)}
	registry, err := permissions.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true})
	_, err = readBridgeNodeScopes(ctx, registry, []permissions.Descriptor{descriptor}, &registryv1.Node{Id: "node-1", EnrollmentGeneration: 8})
	if err == nil || provider.reads != 0 {
		t.Fatalf("stale generation error=%v registry reads=%d", err, provider.reads)
	}
}

func TestReadBridgeNodeScopesRefusesCrossPrincipalReceipt(t *testing.T) {
	descriptor := permissions.Descriptor{
		Version: permissions.ContractVersion, ID: "bridge/node-scopes/machine-1/node-1", Owner: "vrooli-bridge",
		PrincipalKind: string(identity.ActorHuman), PrincipalID: "operator-1", Scope: "vrooli-bridge:write",
		SubjectKind: "node-scopes", SubjectID: "node-1", TargetBound: true,
		Target: permissions.TargetBinding{MachineID: "machine-1", NodeID: "node-1", EnrollmentGeneration: 7},
		Source: "Bridge NodeRegistryService + MachineService", SchemaVersion: "registry/v1", CanRead: true, CanWrite: true, Sensitivity: "operator",
	}
	provider := &targetScopeRegistryFixture{descriptor: descriptor, value: []byte(`{"scopes":["onboarding:read","onboarding:write"]}`)}
	registry, err := permissions.NewRegistry(provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator-2", Verified: true})
	_, err = readBridgeNodeScopes(ctx, registry, []permissions.Descriptor{descriptor}, &registryv1.Node{Id: "node-1", EnrollmentGeneration: 7})
	if err == nil || provider.reads != 0 {
		t.Fatalf("cross-principal error=%v registry reads=%d", err, provider.reads)
	}
}

type targetScopeRegistryFixture struct {
	descriptor permissions.Descriptor
	value      []byte
	reads      int
}

func (f *targetScopeRegistryFixture) ProviderID() string { return "bridge.scope-fixture" }
func (f *targetScopeRegistryFixture) Owner() string      { return "vrooli-bridge" }
func (f *targetScopeRegistryFixture) List(context.Context) ([]permissions.Descriptor, error) {
	return []permissions.Descriptor{f.descriptor}, nil
}
func (f *targetScopeRegistryFixture) AuthorizeRead(_ context.Context, d permissions.Descriptor, req permissions.ReadRequest) (permissions.Decision, error) {
	if d.ID != f.descriptor.ID || req.ActorID != f.descriptor.PrincipalID || req.Owner != f.descriptor.Owner || req.Target != f.descriptor.Target {
		return permissions.Decision{}, errors.New("scope read binding mismatch")
	}
	return permissions.Decision{Allowed: true, DecisionRef: "fixture-read"}, nil
}
func (f *targetScopeRegistryFixture) Read(_ context.Context, d permissions.Descriptor, permit permissions.Permit, req permissions.ReadRequest) (permissions.ReadResult, error) {
	f.reads++
	if !permit.ValidForRead(d.ID, req) {
		return permissions.ReadResult{}, errors.New("scope read permit mismatch")
	}
	return permissions.ReadResult{PermissionID: d.ID, Owner: d.Owner, Target: d.Target, Revision: "r7", Value: append([]byte(nil), f.value...)}, nil
}
func (*targetScopeRegistryFixture) Authorize(context.Context, permissions.Descriptor, permissions.WriteRequest) (permissions.Decision, error) {
	return permissions.Decision{}, connect.NewError(connect.CodeUnimplemented, errors.New("fixture is read-only"))
}
func (*targetScopeRegistryFixture) Write(context.Context, permissions.Descriptor, permissions.Permit, permissions.WriteRequest) (permissions.Receipt, error) {
	return permissions.Receipt{}, connect.NewError(connect.CodeUnimplemented, errors.New("fixture is read-only"))
}

func TestProbeOnboardingTargetUsesBoundedReadOnlySessionRequest(t *testing.T) {
	reacher := &targetProbeReacher{}
	if err := probeOnboardingTarget(context.Background(), reacher, "node-1"); err != nil {
		t.Fatal(err)
	}
	if reacher.request.NodeID != "node-1" || reacher.request.Scenario != "vrooli-onboarding" || reacher.request.Procedure != onboardingTargetProbeProcedure || reacher.request.Timeout != 1200*time.Millisecond || reacher.request.MaxResponse != 64<<10 {
		t.Fatalf("probe request = %+v", reacher.request)
	}
	if len(reacher.request.Body) == 0 {
		t.Fatal("probe request did not contain a serialized session request")
	}
}

func TestProbeOnboardingTargetPreservesReachFailure(t *testing.T) {
	want := errors.New("stale deployment")
	reacher := &targetProbeReacher{err: want}
	if err := probeOnboardingTarget(context.Background(), reacher, "node-1"); !errors.Is(err, want) {
		t.Fatalf("probe error = %v, want %v", err, want)
	}
}
