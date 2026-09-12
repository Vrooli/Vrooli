package validationmatrix

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
)

type fakeRegistry struct {
	nodes []*registryv1.Node
}

func (f fakeRegistry) ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	return connect.NewResponse(&registryv1.ListNodesResponse{Nodes: f.nodes}), nil
}

type fakeDispatcher struct {
	request *dispatchv1.DispatchJobRequest
}

func (f *fakeDispatcher) DispatchJob(_ context.Context, request *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&dispatchv1.DispatchJobResponse{RunId: "run-1", NodeId: request.Msg.NodeId, Scenario: request.Msg.Scenario, Verb: request.Msg.Verb}), nil
}

type fakeRuns struct {
	status runsv1.RunStatus
}

func (f fakeRuns) WaitRun(_ context.Context, request *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	return connect.NewResponse(&runsv1.WaitRunResponse{Run: &runsv1.Run{Id: request.Msg.Id, Status: f.status}}), nil
}

func (f fakeRuns) GetRun(_ context.Context, request *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	return connect.NewResponse(&runsv1.GetRunResponse{Run: &runsv1.Run{Id: request.Msg.Id, Status: f.status}}), nil
}

type blockingDispatcher struct{}

func (blockingDispatcher) DispatchJob(ctx context.Context, _ *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type timeoutRuns struct{}

func (timeoutRuns) WaitRun(context.Context, *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	return nil, context.DeadlineExceeded
}

func (timeoutRuns) GetRun(context.Context, *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	return nil, context.DeadlineExceeded
}

// stubHostProber returns fixed host facts so discovery tests classify nodes
// without dispatching a probe job.
type stubHostProber struct {
	facts map[string]HostFacts
	err   error
}

func (s stubHostProber) ProbeHost(_ context.Context, nodeID string) (HostFacts, error) {
	if s.err != nil {
		return HostFacts{}, s.err
	}
	facts, ok := s.facts[nodeID]
	if !ok {
		return HostFacts{}, errors.New("no facts for node")
	}
	return facts, nil
}

func darwinFacts(xcodeVersion, runtimes string) HostFacts {
	tools := map[string]HostTool{"xcodebuild": {Present: xcodeVersion != "", Version: xcodeVersion}}
	tools["simctl"] = HostTool{Present: runtimes != "" || xcodeVersion != "", Version: runtimes}
	return HostFacts{OS: "darwin", Arch: "amd64", RuntimeTools: tools}
}

func androidFacts(os string, emulator, kvm bool) HostFacts {
	tools := map[string]HostTool{"adb": {Present: true, Path: "/opt/adb"}}
	tools["emulator"] = HostTool{Present: emulator}
	if os == "linux" {
		tools["kvm"] = HostTool{Present: kvm}
	}
	return HostFacts{OS: os, Arch: "amd64", RuntimeTools: tools}
}

// Platform capability is derived from probed host facts, never from
// node.Capabilities: that field carries the node's allowlisted dispatch verbs,
// which share no vocabulary with platform names.
func TestDiscoverDerivesIOSCapabilityFromProbedHostFacts(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Name: "minimouse", Os: "darwin", Arch: "amd64", Online: true,
		Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		// Real nodes advertise verb patterns here, not platform names.
		Capabilities: []string{"host inventory*", "setup*", "scenario status*"},
		Scopes:       []string{"vrooli-bridge:read", "vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"mac-1": darwinFacts("26.1", "iOS 26.1")},
	}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	target := targets[0]
	if !target.Available {
		t.Fatalf("a darwin node with Xcode and a simulator runtime must be available: %#v", target)
	}
	if target.Platform != "ios" || target.NodeID != "mac-1" || target.ID != "bridge:mac-1" {
		t.Fatalf("iOS target identity = %#v", target)
	}
	if !target.Supports(CapabilityXcode) || !target.Supports(CapabilitySimctl) || !target.Supports(CapabilityIOSSimulator) {
		t.Fatalf("iOS capabilities = %#v", target.Capabilities)
	}
	if target.BridgeTrust == nil || !target.BridgeTrust.DispatchAuthorized {
		t.Fatalf("dispatch authorization = %#v", target.BridgeTrust)
	}
}

// A dispatch scope is "vrooli-bridge:write". Matching a "scenario test" string
// rejected every real node, because no node has ever carried such a scope.
func TestDiscoverAcceptsRealBridgeWriteScope(t *testing.T) {
	for _, scopes := range [][]string{{"vrooli-bridge:write"}, {"vrooli-bridge:read", "vrooli-bridge:write"}} {
		client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
			Id: "mac-1", Os: "darwin", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, Scopes: scopes,
		}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
			facts: map[string]HostFacts{"mac-1": darwinFacts("26.1", "iOS 26.1")},
		}))
		targets, err := client.Discover(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !targets[0].Available {
			t.Fatalf("scopes %v must authorize dispatch: %#v", scopes, targets[0])
		}
	}
}

func TestDiscoverRejectsNodeWithoutDispatchScope(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Os: "darwin", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Scopes: []string{"vrooli-bridge:read"},
	}}}, nil, nil, WithPlatform("ios"))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Available || targets[0].MissingCapability == "" || targets[0].NextAction == "" {
		t.Fatalf("read-only node must be unavailable with a next action: %#v", targets[0])
	}
}

// A darwin node without a usable toolchain is not an iOS target. Reporting one
// would let a release gate select a node that cannot produce Apple evidence.
func TestDiscoverReportsDarwinNodeWithoutXcodeAsUnavailable(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Os: "darwin", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"mac-1": darwinFacts("", "")},
	}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Available {
		t.Fatalf("a darwin node without Xcode must not be an available iOS target: %#v", targets[0])
	}
	if targets[0].MissingCapability != CapabilityXcode {
		t.Fatalf("missing capability = %q, want %q", targets[0].MissingCapability, CapabilityXcode)
	}
}

// Xcode present but no installed runtime is a real, actionable state: on Intel
// hosts the universal runtime variant must be fetched explicitly.
func TestDiscoverReportsMissingSimulatorRuntimeDistinctly(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Os: "darwin", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"mac-1": darwinFacts("26.1", "")},
	}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Available || targets[0].MissingCapability != CapabilityIOSSimulator {
		t.Fatalf("missing runtime must be distinct and unavailable: %#v", targets[0])
	}
	if !strings.Contains(targets[0].NextAction, "universal") {
		t.Fatalf("next action must name the universal variant: %q", targets[0].NextAction)
	}
}

// A failed probe is not the same as a node lacking capability, and must not be
// silently reported as one.
func TestDiscoverReportsProbeFailureDistinctFromMissingCapability(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Os: "darwin", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{err: errors.New("probe failed")}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Available || targets[0].MissingCapability != "host toolchain probe" {
		t.Fatalf("probe failure = %#v", targets[0])
	}
}

// A Linux node is never an iOS target, however healthy it is.
func TestDiscoverNeverReportsLinuxNodeAsIOSTarget(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "linux-1", Os: "linux", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"linux-1": androidFacts("linux", true, true)},
	}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if targets[0].Available {
		t.Fatalf("a linux node must never be an available iOS target: %#v", targets[0])
	}
}

func TestDiscoverDerivesAndroidCapabilityAndEmulatorAcceleration(t *testing.T) {
	tests := []struct {
		name       string
		facts      HostFacts
		deviceKind string
	}{
		{name: "accelerated emulator", facts: androidFacts("linux", true, true), deviceKind: "emulator"},
		// Without KVM an emulator renders too slowly to produce video evidence,
		// so the node is a physical-device target only.
		{name: "no kvm", facts: androidFacts("linux", true, false), deviceKind: "physical"},
		{name: "no emulator", facts: androidFacts("linux", false, true), deviceKind: "physical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
				Id: "android-1", Os: "linux", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE,
				Scopes: []string{"vrooli-bridge:write"},
			}}}, nil, nil, WithPlatform("android"), WithHostProber(stubHostProber{
				facts: map[string]HostFacts{"android-1": tt.facts},
			}))
			targets, err := client.Discover(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if !targets[0].Available || targets[0].Platform != "android" {
				t.Fatalf("android target = %#v", targets[0])
			}
			if targets[0].DeviceKind != tt.deviceKind {
				t.Fatalf("device kind = %q, want %q", targets[0].DeviceKind, tt.deviceKind)
			}
		})
	}
}

func TestDiscoverReportsOfflineAndRevokedNodesWithoutProbing(t *testing.T) {
	// Probing an offline node would spend a dispatch to learn what its status
	// already says, so the prober must not be consulted at all.
	prober := stubHostProber{err: errors.New("prober must not be called")}
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{
		{Id: "off-1", Name: "Offline", Online: false, Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE},
		{Id: "rev-1", Name: "Revoked", Online: false, Status: registryv1.NodeStatus_NODE_STATUS_REVOKED},
	}}, nil, nil, WithPlatform("ios"), WithHostProber(prober))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
	if targets[0].Available || targets[0].Health.Status != "offline" {
		t.Fatalf("offline node = %#v", targets[0])
	}
	if targets[1].Available || targets[1].Health.Status != "revoked" {
		t.Fatalf("revoked node = %#v", targets[1])
	}
	for _, target := range targets {
		if target.MissingCapability == "" || target.NextAction == "" {
			t.Fatalf("unreachable node must stay actionable: %#v", target)
		}
	}
}

func TestDiscoverReturnsUnavailableTargetWhenFleetIsEmpty(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: nil}, nil, nil, WithPlatform("ios"))
	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Available || targets[0].Reason == "" || targets[0].NextAction == "" {
		t.Fatalf("empty bridge inventory = %#v", targets)
	}
}

func TestExecuteNeverClaimsDesktopPassWithoutTargetEvidence(t *testing.T) {
	dispatcher := &fakeDispatcher{}
	client := NewClientForTesting(nil, dispatcher, fakeRuns{status: runsv1.RunStatus_RUN_STATUS_PASSED})
	result := client.Execute(context.Background(), CellRequest{RunID: "matrix-run", ArtifactDigest: "sha256:artifact", Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})

	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED {
		t.Fatalf("disposition = %s, want degraded", result.Disposition)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].GetKind() != domainv1.LayeredEvidence_KIND_TARGET {
		t.Fatalf("bridge evidence = %#v", result.Evidence)
	}
	if result.Identity.NodeID != "node-1" || result.Identity.JobID != "run-1" || result.Identity.RunID != "run-1" || result.Identity.ArtifactDigest != "sha256:artifact" {
		t.Fatalf("bridge identity = %+v", result.Identity)
	}
	if !strings.Contains(result.Evidence[0].GetUri(), "artifact=sha256%3Aartifact") {
		t.Fatalf("bridge evidence lost artifact identity: %q", result.Evidence[0].GetUri())
	}
	if dispatcher.request.GetVerb() != "scenario test" || dispatcher.request.GetScenario() != "demo" {
		t.Fatalf("typed dispatch request = %#v", dispatcher.request)
	}
}

func TestExecuteCarriesAddressedCommandAndArguments(t *testing.T) {
	dispatcher := &fakeDispatcher{}
	client := NewClientForTesting(nil, dispatcher, fakeRuns{status: runsv1.RunStatus_RUN_STATUS_FAILED})
	result := client.Execute(context.Background(), CellRequest{
		Command: "scenario status", Args: []string{"--json"},
		Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"},
	})
	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED {
		t.Fatalf("disposition = %s, want failed", result.Disposition)
	}
	if dispatcher.request.GetVerb() != "scenario status" || dispatcher.request.GetScenario() != "demo" {
		t.Fatalf("typed dispatch request = %#v", dispatcher.request)
	}
	if len(dispatcher.request.GetArgs()) != 1 || dispatcher.request.GetArgs()[0] != "--json" {
		t.Fatalf("typed dispatch args = %v, want [--json]", dispatcher.request.GetArgs())
	}
}

func TestExecuteMapsFailedAndAbortedRuns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status runsv1.RunStatus
		want   domainv1.ValidationDisposition
	}{
		{name: "failed", status: runsv1.RunStatus_RUN_STATUS_FAILED, want: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED},
		{name: "aborted", status: runsv1.RunStatus_RUN_STATUS_ABORTED, want: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClientForTesting(nil, &fakeDispatcher{}, fakeRuns{status: tc.status})
			result := client.Execute(context.Background(), CellRequest{Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})
			if result.Disposition != tc.want {
				t.Fatalf("disposition = %s, want %s", result.Disposition, tc.want)
			}
		})
	}
}

func TestExecuteConvertsDispatchCancellationToUnavailable(t *testing.T) {
	client := NewClientForTesting(nil, blockingDispatcher{}, fakeRuns{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := client.Execute(ctx, CellRequest{Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})
	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE || !result.Retryable {
		t.Fatalf("cancelled dispatch result = %+v", result)
	}
}

func TestExecuteConvertsBridgeWaitTimeoutToDegraded(t *testing.T) {
	client := NewClientForTesting(nil, &fakeDispatcher{}, timeoutRuns{})
	result := client.Execute(context.Background(), CellRequest{Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})
	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED || !result.Retryable {
		t.Fatalf("timed out bridge result = %+v", result)
	}
}

// A node running a control plane that predates the toolchain probe answers
// host inventory successfully but reports no Apple tools at all. That is not
// the same as "this Mac has no Xcode", and the next action must say so —
// otherwise an operator would go install Xcode on a machine that already has it.
func TestDiscoverDistinguishesUnprobedToolchainFromAbsentToolchain(t *testing.T) {
	legacyFacts := HostFacts{OS: "darwin", Arch: "amd64", RuntimeTools: map[string]HostTool{
		"docker":          {Present: true},
		"system_profiler": {Present: true},
	}}
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-1", Name: "minimouse", Os: "darwin", Online: true,
		Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"mac-1": legacyFacts},
	}))

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	target := targets[0]
	if target.Available {
		t.Fatalf("an unprobed toolchain must not be reported available: %#v", target)
	}
	if target.MissingCapability != "host toolchain probe" {
		t.Fatalf("missing capability = %q, want the probe itself", target.MissingCapability)
	}
	if !strings.Contains(target.NextAction, "control plane") {
		t.Fatalf("next action must point at the node's control plane, got %q", target.NextAction)
	}
	// The distinction matters: an absent Xcode names Xcode instead.
	absent := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{{
		Id: "mac-2", Os: "darwin", Online: true,
		Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, Scopes: []string{"vrooli-bridge:write"},
	}}}, nil, nil, WithPlatform("ios"), WithHostProber(stubHostProber{
		facts: map[string]HostFacts{"mac-2": darwinFacts("", "")},
	}))
	absentTargets, err := absent.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if absentTargets[0].MissingCapability == target.MissingCapability {
		t.Fatal("an absent toolchain and an unprobed toolchain must not report the same missing capability")
	}
}
