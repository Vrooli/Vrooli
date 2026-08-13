package onboard

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
	onboardconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard/onboard_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "vrooli-bridge/cli/internal/testutil"
)

// fakeOnboard is a programmable OnboardService handler for the CLI thin-wrapper
// tests. It records the last request per verb and returns canned responses; the
// Get sequence lets a watch test model progress → terminal without polling.
type fakeOnboard struct {
	machinesconnect.UnimplementedMachineServiceHandler

	createMachineReq *machinesv1.CreateMachineRequest
	preflightReq     *onboardv1.PreflightOnboardingRequest
	preflightResp    *onboardv1.PreflightOnboardingResponse
	startReq         *onboardv1.StartOnboardingRequest
	startResp        *onboardv1.StartOnboardingResponse
	startErr         error

	getResps []*onboardv1.GetOnboardingResponse
	getCalls int

	waitReq   *onboardv1.WaitOnboardingRequest
	waitResp  *onboardv1.WaitOnboardingResponse
	waitCalls int

	listReq  *onboardv1.ListOnboardingsRequest
	listResp *onboardv1.ListOnboardingsResponse

	cancelReq  *onboardv1.CancelOnboardingRequest
	cancelResp *onboardv1.CancelOnboardingResponse
}

func (f *fakeOnboard) CreateMachine(_ context.Context, req *connect.Request[machinesv1.CreateMachineRequest]) (*connect.Response[machinesv1.CreateMachineResponse], error) {
	f.createMachineReq = req.Msg
	return connect.NewResponse(&machinesv1.CreateMachineResponse{Machine: &machinesv1.Machine{Id: "machine-test-1"}}), nil
}

func (f *fakeOnboard) PreflightOnboarding(_ context.Context, req *connect.Request[onboardv1.PreflightOnboardingRequest]) (*connect.Response[onboardv1.PreflightOnboardingResponse], error) {
	f.preflightReq = req.Msg
	if f.preflightResp != nil {
		return connect.NewResponse(f.preflightResp), nil
	}
	machineID := req.Msg.MachineId
	if machineID == "" {
		machineID = "machine-test-1"
	}
	return connect.NewResponse(&onboardv1.PreflightOnboardingResponse{
		Decision:  onboardv1.ConnectDecision_CONNECT_DECISION_FIRST_TOUCH,
		MachineId: machineID, Host: req.Msg.Host, Port: req.Msg.Port, User: req.Msg.User,
		PasswordRequired: true, Message: "test first touch",
	}), nil
}

func (f *fakeOnboard) StartOnboarding(_ context.Context, req *connect.Request[onboardv1.StartOnboardingRequest]) (*connect.Response[onboardv1.StartOnboardingResponse], error) {
	f.startReq = req.Msg
	if f.startErr != nil {
		return nil, f.startErr
	}
	return connect.NewResponse(f.startResp), nil
}

func (f *fakeOnboard) GetOnboarding(_ context.Context, req *connect.Request[onboardv1.GetOnboardingRequest]) (*connect.Response[onboardv1.GetOnboardingResponse], error) {
	idx := f.getCalls
	if idx >= len(f.getResps) {
		idx = len(f.getResps) - 1
	}
	f.getCalls++
	return connect.NewResponse(f.getResps[idx]), nil
}

func (f *fakeOnboard) ListOnboardings(_ context.Context, req *connect.Request[onboardv1.ListOnboardingsRequest]) (*connect.Response[onboardv1.ListOnboardingsResponse], error) {
	f.listReq = req.Msg
	return connect.NewResponse(f.listResp), nil
}

func (f *fakeOnboard) WaitOnboarding(_ context.Context, req *connect.Request[onboardv1.WaitOnboardingRequest]) (*connect.Response[onboardv1.WaitOnboardingResponse], error) {
	f.waitReq = req.Msg
	f.waitCalls++
	return connect.NewResponse(f.waitResp), nil
}

func (f *fakeOnboard) CancelOnboarding(_ context.Context, req *connect.Request[onboardv1.CancelOnboardingRequest]) (*connect.Response[onboardv1.CancelOnboardingResponse], error) {
	f.cancelReq = req.Msg
	return connect.NewResponse(f.cancelResp), nil
}

func (f *fakeOnboard) RemoveFailedOnboarding(_ context.Context, _ *connect.Request[onboardv1.RemoveFailedOnboardingRequest]) (*connect.Response[onboardv1.RemoveFailedOnboardingResponse], error) {
	return connect.NewResponse(&onboardv1.RemoveFailedOnboardingResponse{}), nil
}

func connectAPI(svc *fakeOnboard) http.Handler {
	path, handler := onboardconnect.NewOnboardServiceHandler(svc)
	machinePath, machineHandler := machinesconnect.NewMachineServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	mux.Handle(machinePath, machineHandler)
	return mux
}

// startSchema mirrors the manifest `onboard start` flags so the RunContext
// resolves the same names (and the same @cp default) the dispatcher would.
func startSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "host", Required: true},
		{Name: "machine-id"},
		{Name: "user"},
		{Name: "port"},
		{Name: "name"},
		{Name: "capabilities", Aliases: []string{"scopes"}},
		{Name: "revision", Default: defaultRevision},
		{Name: "repo-url"},
		{Name: "checkout-dir"},
		{Name: "control-plane-url"},
		{Name: "reachability-mode"},
		{Name: "verify-timeout"},
		{Name: "setup-environment"},
		{Name: "setup-resources"},
		{Name: "setup-scenarios"},
		{Name: "include-optional", Bool: true},
		{Name: "skip-setup", Bool: true},
		{Name: "skip-prereqs", Bool: true},
		{Name: "provision-sudo", Bool: true},
		{Name: "no-provision-sudo", Bool: true},
		{Name: "source"},
		{Name: "password-stdin", Bool: true},
		{Name: "setup-passphrase-stdin", Bool: true},
		{Name: "prompt-password", Bool: true},
	}}
}

func TestStart_ExplicitMachineIDReusesDurableIdentity(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{
			Decision: onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT, MachineId: "machine-existing",
			Host: "minimouse.local", Port: 22, User: "matthalloran8", Message: "trusted",
		},
		startResp: &onboardv1.StartOnboardingResponse{
			OpId: "op-existing", Host: "minimouse.local", Port: 22, User: "matthalloran8",
		},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return false },
		readSecret: func() ([]byte, error) { t.Fatal("password prompt must not run"); return nil, nil },
		prompt:     io.Discard,
	}

	ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"host":       "minimouse.local",
			"machine-id": "machine-existing",
			"user":       "matthalloran8",
		},
		RawArgs: []string{"--host", "minimouse.local", "--machine-id", "machine-existing", "--user", "matthalloran8"},
	})

	require.NoError(t, h.start(ctx))
	require.Nil(t, svc.createMachineReq, "an explicit durable Machine must not create a replacement identity")
	require.Equal(t, "machine-existing", svc.startReq.GetMachineId())
}

func terminalConnectResponse() *onboardv1.GetOnboardingResponse {
	return &onboardv1.GetOnboardingResponse{Op: &onboardv1.OnboardingOp{Id: "op-connect", Host: "swarminator", Port: 22, User: "matthalloran8", State: onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED}}
}

func TestConnect_FirstTouchPromptsOnceAndUsesResolvedMachine(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{Decision: onboardv1.ConnectDecision_CONNECT_DECISION_FIRST_TOUCH, MachineId: "machine-first", Host: "swarminator", Port: 22, User: "matthalloran8", PasswordRequired: true, Message: "first touch"},
		startResp:     &onboardv1.StartOnboardingResponse{OpId: "op-connect", Host: "swarminator", Port: 22, User: "matthalloran8"},
		waitResp:      &onboardv1.WaitOnboardingResponse{Op: terminalConnectResponse().Op},
		getResps:      []*onboardv1.GetOnboardingResponse{terminalConnectResponse()},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	prompts := 0
	h.password = passwordSource{lookupEnv: func(string) (string, bool) { return "", false }, isTerminal: func() bool { return true }, readSecret: func() ([]byte, error) { prompts++; return []byte("once-only"), nil }, prompt: io.Discard}
	ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"host": "swarminator", "user": "matthalloran8"}})
	require.NoError(t, h.preflightConnect(ctx))
	require.Equal(t, 1, prompts)
	require.Equal(t, "machine-first", svc.startReq.MachineId)
	require.Equal(t, "once-only", svc.startReq.SshPassword)
}

func TestConnect_TrustedReconnectDoesNotPrompt(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{Decision: onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT, MachineId: "machine-trusted", Host: "swarminator", Port: 22, User: "matthalloran8", ClientKeyFingerprint: "SHA256:client", PasswordRequired: false, Message: "trusted"},
		startResp:     &onboardv1.StartOnboardingResponse{OpId: "op-connect", Host: "swarminator", Port: 22, User: "matthalloran8"},
		waitResp:      &onboardv1.WaitOnboardingResponse{Op: terminalConnectResponse().Op},
		getResps:      []*onboardv1.GetOnboardingResponse{terminalConnectResponse()},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = passwordSource{lookupEnv: func(string) (string, bool) { return "", false }, isTerminal: func() bool { return true }, readSecret: func() ([]byte, error) { t.Fatal("trusted reconnect must not prompt"); return nil, nil }, prompt: io.Discard}
	ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{Flags: map[string]string{"host": "swarminator", "user": "matthalloran8"}})
	require.NoError(t, h.preflightConnect(ctx))
	require.Equal(t, "machine-trusted", svc.startReq.MachineId)
	require.Empty(t, svc.startReq.SshPassword)
}

// envPassword is an injected password source that yields a fixed secret via the
// env path and fails the test if the interactive prompt is ever reached.
func envPassword(t *testing.T, secret string) passwordSource {
	t.Helper()
	return passwordSource{
		lookupEnv:  func(k string) (string, bool) { return secret, k == sshPasswordEnvVar },
		isTerminal: func() bool { return false },
		readSecret: func() ([]byte, error) { t.Fatal("prompt must not run when the env var is set"); return nil, nil },
		prompt:     io.Discard,
	}
}

func TestStart_PasswordStdinFlagPipesTheSecret(t *testing.T) {
	svc := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{
		OpId: "op-stdin", Host: "10.0.0.5", Port: 22, User: "deploy",
	}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = passwordSource{
		lookupEnv: func(string) (string, bool) {
			t.Fatal("env must not be consulted under --password-stdin")
			return "", false
		},
		isTerminal: func() bool { return false },
		readSecret: func() ([]byte, error) { t.Fatal("prompt must not run under --password-stdin"); return nil, nil },
		stdin:      strings.NewReader("piped-pw\n"),
		prompt:     io.Discard,
	}

	ctx, out := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags:     map[string]string{"host": "10.0.0.5", "user": "deploy"},
		BoolFlags: map[string]bool{"password-stdin": true},
	})
	require.NoError(t, h.start(ctx))

	require.Equal(t, "machine-test-1", svc.startReq.MachineId)
	require.Nil(t, svc.createMachineReq, "start must use the server-owned preflight resolver instead of creating a Machine in the CLI")
	// The piped secret reached the request body with the pipe newline stripped...
	require.Equal(t, "piped-pw", svc.startReq.SshPassword)
	// ...and the report names the (non-secret) source without leaking the value.
	require.Contains(t, out.String(), "--password-stdin")
	require.NotContains(t, out.String(), "piped-pw")
}

func TestStart_NoCredentialReportsKeyTrustedAssumption(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{
			Decision: onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT, MachineId: "machine-test-1",
			Host: "10.0.0.5", Port: 22, User: "deploy", PasswordRequired: false, Message: "trusted",
		},
		startResp: &onboardv1.StartOnboardingResponse{
			OpId: "op-none", Host: "10.0.0.5", Port: 22, User: "deploy",
		},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = passwordSource{
		lookupEnv:  func(string) (string, bool) { return "", false },
		isTerminal: func() bool { return true }, // even on a TTY: no prompt, ever
		readSecret: func() ([]byte, error) { t.Fatal("start must never auto-prompt"); return nil, nil },
		prompt:     io.Discard,
	}

	ctx, out := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "10.0.0.5", "user": "deploy"},
	})
	require.NoError(t, h.start(ctx))

	require.Empty(t, svc.startReq.SshPassword)
	// The report says the run is riding on key trust and teaches every intake path.
	require.Contains(t, out.String(), "already trusts the bridge key")
	require.Contains(t, out.String(), "--password-stdin")
}

func TestStart_SendsPasswordInBodyNotArgv(t *testing.T) {
	svc := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{
		OpId: "op-1", Host: "10.0.0.5", Port: 22, User: "deploy",
	}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = envPassword(t, "s3cret-pw")

	// A realistic argv for `onboard start` — note it carries NO password.
	argv := []string{"--host", "10.0.0.5", "--user", "deploy", "--capabilities", "scenario,deploy"}
	ctx, out := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags:   map[string]string{"host": "10.0.0.5", "user": "deploy", "capabilities": "scenario,deploy"},
		RawArgs: argv,
	})

	require.NoError(t, h.start(ctx))

	// The secret reached the server in the request body...
	require.Equal(t, "s3cret-pw", svc.startReq.SshPassword)
	// ...the target + parsed capabilities came through...
	require.Equal(t, "10.0.0.5", svc.startReq.Host)
	require.Equal(t, "deploy", svc.startReq.User)
	require.Equal(t, []string{"scenario", "deploy"}, svc.startReq.Capabilities)
	// ...the revision defaulted to @cp...
	require.Equal(t, defaultRevision, svc.startReq.TargetRevision)
	// ...and it NEVER appeared on argv (the ps-leak boundary).
	for _, arg := range ctx.Args() {
		require.NotContains(t, arg, "s3cret-pw", "password must never be on argv")
	}
	require.NotContains(t, strings.Join(argv, " "), "s3cret-pw")
	// Human output points the operator at watch.
	require.Contains(t, out.String(), "op-1")
	require.Contains(t, out.String(), "onboard watch op-1")
}

func TestStart_ProvisionSudoDefaultsOnAndCanBeDisabled(t *testing.T) {
	newStartCtx := func(bools map[string]bool) (*fakeOnboard, cliapp.RunContext) {
		svc := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{OpId: "op-s", Host: "h", Port: 22, User: "root"}}
		core := clitest.NewTestApp(t, connectAPI(svc))
		h := newHandlers(core)
		h.password = envPassword(t, "pw")
		ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
			Flags:     map[string]string{"host": "h"},
			BoolFlags: bools,
		})
		require.NoError(t, h.start(ctx))
		return svc, ctx
	}

	// Default: no flag → provisioning ON (the operator handed over admin creds).
	svc, _ := newStartCtx(nil)
	require.True(t, svc.startReq.ProvisionSudo, "provision-sudo must default ON")

	// Opt out with --no-provision-sudo.
	svcOff, _ := newStartCtx(map[string]bool{"no-provision-sudo": true})
	require.False(t, svcOff.startReq.ProvisionSudo, "--no-provision-sudo must disable it")

	// Explicit --provision-sudo wins if both are somehow present.
	svcBoth, _ := newStartCtx(map[string]bool{"no-provision-sudo": true, "provision-sudo": true})
	require.True(t, svcBoth.startReq.ProvisionSudo, "explicit --provision-sudo wins")
}

func TestStart_SetupProfileFlagsReachRequest(t *testing.T) {
	svc := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{OpId: "op-p", Host: "h", Port: 22, User: "root"}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = envPassword(t, "pw")

	ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{
			"host":              "h",
			"setup-environment": "production",
			"setup-resources":   "enabled",
			"setup-scenarios":   "none",
		},
		BoolFlags: map[string]bool{"include-optional": true},
	})
	require.NoError(t, h.start(ctx))

	require.Equal(t, "production", svc.startReq.SetupEnvironment)
	require.Equal(t, "enabled", svc.startReq.SetupResources)
	require.Equal(t, "none", svc.startReq.SetupScenarios)
	require.True(t, svc.startReq.IncludeOptional)

	// Omitted profile flags default to empty (the node uses its own setup defaults).
	svcEmpty := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{OpId: "op-e", Host: "h", Port: 22, User: "root"}}
	coreEmpty := clitest.NewTestApp(t, connectAPI(svcEmpty))
	hEmpty := newHandlers(coreEmpty)
	hEmpty.password = envPassword(t, "pw")
	ctxEmpty, _ := cliapptest.NewCapturedRunContext(coreEmpty, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h"},
	})
	require.NoError(t, hEmpty.start(ctxEmpty))
	require.Empty(t, svcEmpty.startReq.SetupEnvironment)
	require.Empty(t, svcEmpty.startReq.SetupResources)
	require.Empty(t, svcEmpty.startReq.SetupScenarios)
	require.False(t, svcEmpty.startReq.IncludeOptional)
}

func TestStart_SourceModeFlag(t *testing.T) {
	// --source working-tree sets the working-tree source mode on the request.
	svc := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{OpId: "op-wt", Host: "h", Port: 22, User: "root"}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = envPassword(t, "pw")
	ctx, _ := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h", "source": "working-tree"},
	})
	require.NoError(t, h.start(ctx))
	require.Equal(t, onboardv1.SourceMode_SOURCE_MODE_WORKING_TREE, svc.startReq.SourceMode)

	// Omitted --source defaults to pinned.
	svcDef := &fakeOnboard{startResp: &onboardv1.StartOnboardingResponse{OpId: "op-p", Host: "h", Port: 22, User: "root"}}
	coreDef := clitest.NewTestApp(t, connectAPI(svcDef))
	hDef := newHandlers(coreDef)
	hDef.password = envPassword(t, "pw")
	ctxDef, _ := cliapptest.NewCapturedRunContext(coreDef, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h"},
	})
	require.NoError(t, hDef.start(ctxDef))
	require.Equal(t, onboardv1.SourceMode_SOURCE_MODE_PINNED_REVISION, svcDef.startReq.SourceMode)
}

func TestStart_JSONShape(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{Decision: onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT, MachineId: "machine-json", Host: "h", Port: 2222, User: "root", Message: "trusted"},
		startResp:     &onboardv1.StartOnboardingResponse{OpId: "op-json", Host: "h", Port: 2222, User: "root"},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = envPassword(t, "")

	ctx, out := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h", "port": "2222"},
		JSON:  true,
	})
	require.NoError(t, h.start(ctx))
	require.Contains(t, out.String(), "op-json")
	require.Contains(t, out.String(), "op_id") // proto-name JSON field
}

func TestStart_DryRunRendersNoOp(t *testing.T) {
	svc := &fakeOnboard{
		preflightResp: &onboardv1.PreflightOnboardingResponse{Decision: onboardv1.ConnectDecision_CONNECT_DECISION_RECONNECT, MachineId: "machine-dry", Host: "h", Port: 22, User: "root", Message: "trusted"},
		startResp:     &onboardv1.StartOnboardingResponse{DryRun: true, Host: "h", Port: 22, User: "root"},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)
	h.password = envPassword(t, "")

	ctx, out := cliapptest.NewCapturedRunContext(core, startSchema(), cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h"},
	})
	require.NoError(t, h.start(ctx))
	require.Contains(t, out.String(), "Dry run")
	require.Contains(t, out.String(), "not touched")
}

func TestStatus_RendersOpAndEvents(t *testing.T) {
	svc := &fakeOnboard{getResps: []*onboardv1.GetOnboardingResponse{{
		Op: &onboardv1.OnboardingOp{
			Id: "op-2", Host: "host-a", Port: 22, User: "root", NodeName: "mac-mini",
			State: onboardv1.OnboardingState_ONBOARDING_STATE_BOOTSTRAPPING, CreatedAt: timestamppb.Now(),
		},
		Events: []*onboardv1.OnboardingStepEvent{
			{Sequence: 1, StepId: "ssh-setup", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK, Detail: "key installed"},
			{Sequence: 2, StepId: "clone", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_STARTED},
		},
	}}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "op-id", Required: true}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"op-id": "op-2"},
	})
	require.NoError(t, h.status(ctx))
	s := out.String()
	require.Contains(t, s, "op-2")
	require.Contains(t, s, "bootstrapping")
	require.Contains(t, s, "ssh-setup")
	require.Contains(t, s, "key installed")
}

func TestList_PassesFiltersAndFormats(t *testing.T) {
	svc := &fakeOnboard{listResp: &onboardv1.ListOnboardingsResponse{Ops: []*onboardv1.OnboardingOp{
		{Id: "op-a", Host: "h1", Port: 22, User: "root", State: onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED, NodeId: "node-1", CreatedAt: timestamppb.Now()},
	}}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "host"}, {Name: "limit"}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"host": "h1", "limit": "5"},
	})
	require.NoError(t, h.list(ctx))
	require.Equal(t, "h1", svc.listReq.Host)
	require.Equal(t, int32(5), svc.listReq.Limit)
	require.Contains(t, out.String(), "op-a")
	require.Contains(t, out.String(), "succeeded")
}

func watchSchema() cliapp.ArgSchema {
	return cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "op-id", Required: true}},
		Flags:       []cliapp.Flag{{Name: "timeout"}},
	}
}

func TestWatch_TerminalSucceededOnFirstGet(t *testing.T) {
	svc := &fakeOnboard{getResps: []*onboardv1.GetOnboardingResponse{{
		Op: &onboardv1.OnboardingOp{
			Id: "op-3", Host: "host-b", User: "root", NodeId: "node-9",
			State: onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED,
		},
		Events: []*onboardv1.OnboardingStepEvent{
			{Sequence: 1, StepId: "verify-online-confirm", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK},
		},
	}}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	ctx, out := cliapptest.NewCapturedRunContext(core, watchSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"op-id": "op-3"},
	})
	require.NoError(t, h.watch(ctx))
	// Terminal on the first Get → no Wait call needed (not a poll loop).
	require.Equal(t, 0, svc.waitCalls)
	s := out.String()
	require.Contains(t, s, "verify-online-confirm")
	require.Contains(t, s, "SUCCEEDED")
	require.Contains(t, s, "node-9")
}

func TestWatch_FailedReturnsDistinctGuidance(t *testing.T) {
	svc := &fakeOnboard{getResps: []*onboardv1.GetOnboardingResponse{{
		Op: &onboardv1.OnboardingOp{
			Id: "op-4", Host: "host-c", User: "deploy", ExitCode: 4,
			State: onboardv1.OnboardingState_ONBOARDING_STATE_FAILED, FailureReason: failPairing,
		},
		Events: []*onboardv1.OnboardingStepEvent{
			{Sequence: 1, StepId: "pair-redeem", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_FAILED, Detail: "code consumed"},
		},
	}}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	ctx, out := cliapptest.NewCapturedRunContext(core, watchSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"op-id": "op-4"},
	})
	err := h.watch(ctx)
	require.Error(t, err) // non-zero exit for automation
	require.Contains(t, err.Error(), "reissue a fresh code")
	s := out.String()
	require.Contains(t, s, "FAILED")
	require.Contains(t, s, "reissue a fresh code") // taxonomy-specific, in stdout too
}

func TestWatch_ReattachesThenTerminatesWithoutDuplicatingEvents(t *testing.T) {
	running := &onboardv1.GetOnboardingResponse{
		Op: &onboardv1.OnboardingOp{Id: "op-5", Host: "h", User: "root", State: onboardv1.OnboardingState_ONBOARDING_STATE_BOOTSTRAPPING},
		Events: []*onboardv1.OnboardingStepEvent{
			{Sequence: 1, StepId: "clone", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK},
		},
	}
	done := &onboardv1.GetOnboardingResponse{
		Op: &onboardv1.OnboardingOp{Id: "op-5", Host: "h", User: "root", NodeId: "node-5", State: onboardv1.OnboardingState_ONBOARDING_STATE_SUCCEEDED},
		Events: []*onboardv1.OnboardingStepEvent{
			{Sequence: 1, StepId: "clone", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK},
			{Sequence: 2, StepId: "verify-online-confirm", Status: onboardv1.OnboardingStepStatus_ONBOARDING_STEP_STATUS_OK},
		},
	}
	svc := &fakeOnboard{
		getResps: []*onboardv1.GetOnboardingResponse{running, done},
		waitResp: &onboardv1.WaitOnboardingResponse{TimedOut: true, Op: running.Op},
	}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	ctx, out := cliapptest.NewCapturedRunContext(core, watchSchema(), cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"op-id": "op-5"},
	})
	require.NoError(t, h.watch(ctx))
	require.Equal(t, 1, svc.waitCalls) // one block-once wait between the two Gets
	s := out.String()
	// The "clone" step is printed exactly once despite appearing in both Gets.
	require.Equal(t, 1, strings.Count(s, "clone"))
	require.Contains(t, s, "verify-online-confirm")
	require.Contains(t, s, "SUCCEEDED")
}

func TestCancel_RendersRequestedState(t *testing.T) {
	svc := &fakeOnboard{cancelResp: &onboardv1.CancelOnboardingResponse{Op: &onboardv1.OnboardingOp{
		Id: "op-6", Host: "h", User: "root", State: onboardv1.OnboardingState_ONBOARDING_STATE_CANCELLED,
	}}}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "op-id", Required: true}}}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"op-id": "op-6"},
	})
	require.NoError(t, h.cancel(ctx))
	require.Equal(t, "op-6", svc.cancelReq.Id)
	require.Contains(t, out.String(), "cancelled")
}
