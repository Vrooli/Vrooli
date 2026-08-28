package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/nodeclient"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines"
)

func protojsonString(t *testing.T, message proto.Message) string {
	t.Helper()
	wire, err := protojson.Marshal(message)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(wire)
}

func linkedMachineTarget(lastSeen time.Time, scopes []string) targetConnection {
	return targetConnection{
		Target: targetmodel.Target{
			ID: "bridge-node:node-a", DeviceKind: "bridge-node", Label: "minimouse",
			OS: "darwin", Architecture: "amd64", NodeID: "node-a",
			Health:      targetmodel.TargetHealth{Status: "NODE_STATUS_ONLINE"},
			BridgeTrust: &targetmodel.BridgeTrust{Online: true, Registered: true, DispatchAuthorized: true},
			Available:   true, Mode: "dispatchable", LastSeenAt: lastSeen, Scopes: scopes,
			Readiness: readinessFactsForNode(nil),
		},
		BaseURL: "https://bridge.internal", OwnerToken: "LocalSession secret-owner", ReauthToken: "secret-reauth",
	}
}

// A machine list is the surface an operator decides from, so the projection
// must carry identity, reachability and permission — and nothing that would
// let a browser talk to the control plane directly.
func TestMachineListProjectsGrantAndLeaksNoCredential(t *testing.T) {
	remote := linkedMachineTarget(time.Now().Add(-8*time.Second), []string{"*:read"})
	srv := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{remote} }}

	response, err := (&machineRPC{server: srv}).List(context.Background(), connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	machines := response.Msg.GetMachines()
	if len(machines) != 2 {
		t.Fatalf("machine count = %d, want this computer plus one linked machine", len(machines))
	}
	if local := machines[0]; local.GetManageable() {
		t.Error("the computer the console runs on must not be presented as manageable")
	}
	linked := machines[1]
	if linked.GetTarget().GetLabel() != "minimouse" {
		t.Fatalf("linked machine label = %q", linked.GetTarget().GetLabel())
	}
	grant := linked.GetGrant()
	if got := grant.GetSummary(); got != "Read only; changes are not permitted" {
		t.Errorf("grant summary = %q, want the read-only sentence", got)
	}
	if got := grant.GetEffects(); len(got) != 1 || got[0] != "read" {
		t.Errorf("grant effects = %v, want read alone", got)
	}
	if !grant.GetCoversAllApps() {
		t.Error("a wildcard grant must report that it reaches every app, including ones that do not exist yet")
	}
	if !linked.GetManageable() {
		t.Error("a linked machine must be manageable")
	}
	if age := linked.GetHeartbeatAgeSeconds(); age < 5 || age > 60 {
		t.Errorf("heartbeat age = %ds, want the real age in seconds", age)
	}

	// The surface deliberately publishes the control plane's own endpoint so
	// the footer can say where machines are registered. What must never cross
	// is a credential, or the per-machine connection this server holds.
	wire, err := protojson.Marshal(response.Msg)
	if err != nil {
		t.Fatalf("marshal machines: %v", err)
	}
	for _, secret := range []string{remote.OwnerToken, remote.ReauthToken} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("machines surface leaked %q: %s", secret, wire)
		}
	}
	for _, machine := range machines {
		if strings.Contains(protojsonString(t, machine), remote.BaseURL) {
			t.Fatalf("a machine row carried this server's Bridge connection: %s", protojsonString(t, machine))
		}
	}
}

// A machine that stopped answering must still be listed. Dropping it would make
// a machine that is merely asleep indistinguishable from one that was removed.
func TestMachineListKeepsUnreachableMachinesWithTheirAge(t *testing.T) {
	stale := linkedMachineTarget(time.Now().Add(-7*24*time.Hour), nil)
	stale.Available = false
	stale.Mode = "offline"
	stale.Reason = "heartbeat freshness"
	stale.NextAction = "Reconnect the Bridge agent on this node, then refresh the catalog"
	srv := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{stale} }}

	response, err := (&machineRPC{server: srv}).List(context.Background(), connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	machines := response.Msg.GetMachines()
	if len(machines) != 2 {
		t.Fatalf("an unreachable machine must still be listed, got %d entries", len(machines))
	}
	linked := machines[1]
	if linked.GetTarget().GetDispatchable() {
		t.Error("an unreachable machine must not report as dispatchable")
	}
	if age := linked.GetHeartbeatAgeSeconds(); age < 6*24*3600 {
		t.Errorf("heartbeat age = %ds, want roughly seven days", age)
	}
	if linked.GetTarget().GetRecoveryAction() == "" {
		t.Error("an unreachable machine must offer an action the operator can take")
	}
	if got := linked.GetGrant().GetSummary(); got != "No remote actions granted" {
		t.Errorf("grant summary = %q, want the empty-grant sentence", got)
	}
}

// The console names a posture; the control plane owns what it means. Matching
// must be exact, or a machine holding one extra scope would be reported as
// holding the preset the operator chose.
func TestMatchPresetIsExactNotApproximate(t *testing.T) {
	presets := []*machinesv1.PermissionPreset{
		{Name: "read-only", Scopes: []string{"*:read"}},
		{Name: "operate", Scopes: []string{"*:read", "*:write"}},
	}
	if got := matchPreset([]string{"*:read"}, presets); got != "read-only" {
		t.Errorf("matchPreset(read) = %q, want read-only", got)
	}
	if got := matchPreset([]string{"*:write", "*:read"}, presets); got != "operate" {
		t.Errorf("matchPreset is order sensitive: got %q", got)
	}
	if got := matchPreset([]string{"*:read", "*:write", "*:destructive"}, presets); got != "" {
		t.Errorf("a superset of a preset reported as %q, want custom", got)
	}
	if got := matchPreset(nil, presets); got != "" {
		t.Errorf("an empty grant reported as preset %q", got)
	}
}

// The join code path is the one an operator reads aloud, so an expiry that has
// already passed must not render as a negative countdown.
func TestJoinRequestProjectionStatesAgeAndKeyDerivedWords(t *testing.T) {
	requested := time.Now().Add(-12 * time.Second)
	out := joinRequestsToProto([]*pairingv1.PairingRequest{{
		Id: "req-1", Name: "Studio Mac", Os: "darwin", Arch: "arm64", Endpoint: "192.168.1.44",
		ConfirmationWords: []string{"amber", "dolphin", "quartz"},
		KeyFingerprint:    "ed25519:9f3c…a71d",
		CreatedAt:         timestamppb.New(requested),
	}, nil})
	if len(out) != 1 {
		t.Fatalf("projected %d requests, want 1 (nil entries dropped)", len(out))
	}
	request := out[0]
	if len(request.GetConfirmationWords()) != 3 {
		t.Errorf("confirmation words = %v, want three", request.GetConfirmationWords())
	}
	if request.GetKeyFingerprint() == "" {
		t.Error("the key fingerprint must survive projection; it is the field the sender cannot choose")
	}
	if age := request.GetRequestedAgeSeconds(); age < 10 || age > 60 {
		t.Errorf("requested age = %ds, want the real age", age)
	}
}

func TestNodeIDForMachineRejectsAnythingThatIsNotALinkedMachine(t *testing.T) {
	if _, err := nodeIDForMachine("local"); err == nil {
		t.Error("the local machine has no node id and must be refused")
	}
	if _, err := nodeIDForMachine("bridge-node:"); err == nil {
		t.Error("an empty node id must be refused")
	}
	got, err := nodeIDForMachine("bridge-node:node-a")
	if err != nil || got != "node-a" {
		t.Errorf("nodeIDForMachine = (%q, %v), want node-a", got, err)
	}
}

// An unenrolled installation must say so. Returning only this computer with no
// explanation reads exactly like a fleet the operator emptied on purpose.
func TestMachineListStatesWhyAFleetIsUnavailable(t *testing.T) {
	base := targetConnection{Target: targetmodel.Target{
		ID: "bridge-node:", Reason: "Bridge credentials not configured",
		NextAction: "Enroll this machine with Bridge, then refresh the catalog",
	}}
	srv := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{base} }}

	response, err := (&machineRPC{server: srv}).List(context.Background(), connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(response.Msg.GetMachines()) != 1 {
		t.Fatalf("want only this computer, got %d", len(response.Msg.GetMachines()))
	}
	if response.Msg.GetMessage() == "" || response.Msg.GetRecoveryAction() == "" {
		t.Fatalf("an unusable fleet must state a reason and an action: %v", response.Msg)
	}
}

// The control plane expands a preset to one scope per app per effect, so a
// surface that renders the raw list buries the answer. The classification is
// what makes a hundred-and-seventy-entry grant legible.
func TestClassifyScopesSeparatesBreadthFromCount(t *testing.T) {
	effects, apps, wildcard := classifyScopes([]string{"system-monitor:read", "web-console:read", "web-console:write"})
	if len(effects) != 2 || effects[0] != "read" || effects[1] != "write" {
		t.Errorf("effects = %v, want read then write", effects)
	}
	if apps != 2 {
		t.Errorf("app count = %d, want two distinct apps", apps)
	}
	if wildcard {
		t.Error("an enumerated grant must not report as covering every app")
	}

	effects, apps, wildcard = classifyScopes([]string{"*:read", "*:write"})
	if !wildcard {
		t.Error("a wildcard namespace must report as covering every app")
	}
	if apps != 0 {
		t.Errorf("a wildcard grant reported %d apps; a count would understate it", apps)
	}
	if len(effects) != 2 {
		t.Errorf("effects = %v, want read and write", effects)
	}

	if effects, _, _ = classifyScopes([]string{"vrooli:*"}); len(effects) != 3 {
		t.Errorf("an all-effects scope reported %v, want every effect family", effects)
	}
	if effects, apps, wildcard = classifyScopes([]string{"malformed", ""}); len(effects) != 0 || apps != 0 || wildcard {
		t.Errorf("a malformed scope must contribute nothing: %v %d %t", effects, apps, wildcard)
	}
}

func TestPresetTitleReadsAsWords(t *testing.T) {
	for name, want := range map[string]string{
		"read-only":    "Read only",
		"operate":      "Operate",
		"full-control": "Full control",
		"":             "",
	} {
		if got := presetTitle(name); got != want {
			t.Errorf("presetTitle(%q) = %q, want %q", name, got, want)
		}
	}
}

// The words are the only field a joining machine cannot choose for itself, so
// a mismatch must reach the operator as a sentence they can act on rather than
// as the transport's wire message.
func TestControlPlaneErrorExplainsAConfirmationMismatch(t *testing.T) {
	wire := &nodeclient.Error{Kind: nodeclient.ErrInvalidRequest, Err: errors.New("invalid_argument: confirmation_words: do not match the request")}
	err := controlPlaneError("answer this join request", wire)
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeInvalidArgument {
		t.Fatalf("mismatch reported as %v, want invalid_argument", err)
	}
	if strings.Contains(connectErr.Message(), "confirmation_words") {
		t.Errorf("the operator was shown the wire field name: %s", connectErr.Message())
	}
	if !strings.Contains(connectErr.Message(), "deny the request") {
		t.Errorf("the message does not say what to do: %s", connectErr.Message())
	}

	// Every other refusal keeps its original wording; only the safety case is
	// rewritten.
	other := controlPlaneError("issue a join code", &nodeclient.Error{Kind: nodeclient.ErrInvalidRequest, Err: errors.New("a node id is required")})
	if !errors.As(other, &connectErr) || !strings.Contains(connectErr.Message(), "node id is required") {
		t.Errorf("an unrelated refusal was rewritten: %v", other)
	}
}

// The handoff to the control plane's own interface is the one link on this
// surface that leaves the app, and it is the one that was wrong: it carried the
// Bridge API base, which answers a browser with 404 and names loopback on the
// server rather than on whichever machine the person is actually using.
func TestMachineListLinksTheControlPlaneInterfaceNotItsAPI(t *testing.T) {
	var askedFor, askedHost string
	srv := &Server{
		remoteTargetCatalog: func() []targetConnection { return nil },
		resolveConsoleURL: func(_ context.Context, scenarioSlug, requestHost string) (string, error) {
			askedFor, askedHost = scenarioSlug, requestHost
			return "https://vrooli-bridge.example.com", nil
		},
	}
	ctx := discovery.ContextWithExternalHost(context.Background(), "web-console.example.com")

	response, err := (&machineRPC{server: srv}).List(ctx, connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	plane := response.Msg.GetControlPlane()
	if askedFor != "vrooli-bridge" {
		t.Fatalf("resolved scenario %q, want the control plane slug", askedFor)
	}
	// The origin is the caller's, never the server's: a browser on a phone or
	// behind a tunnel must receive an address it can reach.
	if askedHost != "web-console.example.com" {
		t.Fatalf("resolved against host %q, want the host the request arrived on", askedHost)
	}
	if plane.GetConsoleUrl() != "https://vrooli-bridge.example.com" {
		t.Fatalf("console URL = %q, want the resolved interface address", plane.GetConsoleUrl())
	}
	if plane.GetConsoleUrl() == plane.GetEndpoint() {
		t.Fatal("the interface link and the API base must stay separate fields")
	}
}

// A control plane whose interface cannot be located must withhold the link
// rather than fall back to an address that does not serve a browser.
func TestMachineListWithholdsTheLinkWhenTheInterfaceIsUnresolvable(t *testing.T) {
	srv := &Server{
		remoteTargetCatalog: func() []targetConnection { return nil },
		resolveConsoleURL: func(_ context.Context, _, _ string) (string, error) {
			return "", errors.New("vrooli-bridge is not running")
		},
	}

	response, err := (&machineRPC{server: srv}).List(context.Background(), connect.NewRequest(&machinesv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if url := response.Msg.GetControlPlane().GetConsoleUrl(); url != "" {
		t.Fatalf("console URL = %q, want it withheld", url)
	}
}

// The host a browser used reaches the RPC only if the middleware is mounted and
// composes with the Connect handler. Go's server moves the Host header onto
// Request.Host, where connect.Request.Header() cannot see it, so this is the
// hop the whole seam depends on and the one a unit test cannot prove.
func TestExternalHostReachesTheMachineRPCOverHTTP(t *testing.T) {
	var seenHost string
	srv := &Server{
		router:              mux.NewRouter(),
		remoteTargetCatalog: func() []targetConnection { return nil },
		resolveConsoleURL: func(_ context.Context, _, requestHost string) (string, error) {
			seenHost = requestHost
			return "https://vrooli-bridge.example.com", nil
		},
	}
	srv.router.Use(discovery.ExternalHostMiddleware)
	srv.mountMachines()

	httpServer := httptest.NewServer(srv.router)
	defer httpServer.Close()

	request, err := http.NewRequest(http.MethodPost, httpServer.URL+"/vrooli.web_console.v1.machines.MachineService/List", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-Host", "web-console.example.com")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("call List: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if seenHost != "web-console.example.com" {
		t.Fatalf("host seen by the RPC = %q, want the forwarded host", seenHost)
	}
}
