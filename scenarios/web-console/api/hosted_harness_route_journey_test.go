//go:build journey

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

// ac6OfflineHostedRelay is a local Bridge-shaped test node. It exercises the
// production Web Console remote-target client and action runner over Connect,
// while keeping the remote host, token and lifecycle state inside this test.
// It does not deploy a hosted service or contact a real Bridge node.
type ac6OfflineHostedRelay struct {
	mu           sync.Mutex
	requests     []*relayv1.RelayCallRequest
	states       map[string]string
	cleanupPlans map[string]string
	responses    map[string]*relayv1.RelayCallResponse
	effects      map[string]int
}

func (f *ac6OfflineHostedRelay) Call(_ context.Context, req *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req.Msg)
	if req.Msg.GetNodeId() != "ac6-offline-node" || req.Msg.GetScenario() != "vrooli" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("offline node or scenario mismatch"))
	}
	args := req.Msg.GetArgs()
	if len(args) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("resource argument missing"))
	}
	if req.Msg.GetCommand() == "resource update" {
		if req.Msg.GetCommandId() == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("remote harness actions require a durable command ID"))
		}
		if prior := f.responses[req.Msg.GetCommandId()]; prior != nil {
			copy := *prior
			copy.Data = append([]byte(nil), prior.Data...)
			return connect.NewResponse(&copy), nil
		}
	} else if req.Msg.GetCommandId() != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("read-only check must not carry a lifecycle command ID"))
	}
	resource := args[0]
	if req.Msg.GetCommand() == "resource update-check" {
		return ac6RelayResponse(f.checkReport(resource)), nil
	}
	if req.Msg.GetCommand() != "resource update" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("only harness lifecycle commands are admitted"))
	}
	action := "update"
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--action" {
			action = args[i+1]
		}
	}
	if action == "uninstall" {
		cleanupPlan := ""
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "--cleanup-plan" {
				cleanupPlan = args[i+1]
			}
		}
		if cleanupPlan == "" || cleanupPlan != f.cleanupPlans[resource] {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("uninstall requires the prior preview plan"))
		}
	}
	result := harnesslifecycle.HarnessActionResult{Action: action, Success: true, Message: "offline hosted relay completed " + action}
	if action == "preview-cleanup" {
		plan := harnesslifecycle.CleanupPlan{Resource: resource, ID: "ac6-offline-plan-" + resource}
		f.cleanupPlans[resource] = plan.ID
		result.CleanupPlan = &plan
	}
	switch action {
	case "install", "update", "use-managed-copy":
		f.states[resource] = agentStateUpToDate
		f.effects[req.Msg.GetCommandId()]++
	case "uninstall":
		f.states[resource] = agentStateMissing
		f.effects[req.Msg.GetCommandId()]++
	case "doctor", "preview-cleanup":
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("unsupported offline lifecycle action"))
	}
	result.Report = f.report(resource)
	data, err := json.Marshal(struct {
		Success bool                                 `json:"success"`
		Update  harnesslifecycle.HarnessActionResult `json:"harness_update"`
	}{Success: result.Success, Update: result})
	if err != nil {
		return nil, err
	}
	response := ac6RelayResponse(data)
	response.Msg.CorrelationId = req.Msg.GetCommandId()
	f.responses[req.Msg.GetCommandId()] = response.Msg
	return response, nil
}

func (f *ac6OfflineHostedRelay) Reconcile(context.Context, *connect.Request[relayv1.RelayReconcileRequest]) (*connect.Response[relayv1.RelayReconcileResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reconciliation is outside this lifecycle route"))
}

func (f *ac6OfflineHostedRelay) report(resource string) harnesslifecycle.HarnessReport {
	state := f.states[resource]
	if state == "" {
		state = agentStateMissing
	}
	installState, updateState, installed := "managed", "current", "1.0.0"
	if state == agentStateMissing {
		installState, updateState, installed = "missing", "unknown", ""
	}
	return harnesslifecycle.HarnessReport{
		Name: resource, Binary: resource, State: state, InstallKind: installState, InstallState: installState,
		SignInState: "unknown", UpdateState: updateState, Installed: installed, ManagedVersion: installed,
		Latest: "1.0.0", Status: upstreamcheck.StatusUpToDate, CommandPath: filepath.Join("/offline-host", ".local", "bin", resource),
		OS: runtime.GOOS, Arch: runtime.GOARCH, CheckedAt: time.Now().UTC(),
		Source: harnesslifecycle.HarnessSource{Kind: "offline-fixture", ID: resource},
	}
}

func (f *ac6OfflineHostedRelay) checkReport(resource string) []byte {
	report := f.report(resource)
	result := harnesslifecycle.HarnessActionResult{Action: "check", Success: true, Report: report}
	data, _ := json.Marshal(struct {
		Success       bool                                 `json:"success"`
		HarnessUpdate harnesslifecycle.HarnessActionResult `json:"harness_update"`
	}{Success: true, HarnessUpdate: result})
	return data
}

func ac6RelayResponse(data []byte) *connect.Response[relayv1.RelayCallResponse] {
	return connect.NewResponse(&relayv1.RelayCallResponse{Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, ExitCode: 0, Data: data})
}

func TestJ8_HostedOfflineRelayLifecycle(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := harness.Load(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	relay := &ac6OfflineHostedRelay{states: map[string]string{}, cleanupPlans: map[string]string{}, responses: map[string]*relayv1.RelayCallResponse{}, effects: map[string]int{}}
	_, handler := relayconnect.NewRelayServiceHandler(relay)
	bridge := httptest.NewServer(handler)
	defer bridge.Close()
	targetID := "bridge-node:ac6-offline-node"
	capabilities := make([]capabilityprobe.Observation, 0, len(catalog.All()))
	probedAt := time.Now().UTC()
	for _, descriptor := range catalog.All() {
		resource, ok := agentResource(descriptor.ID)
		if !ok {
			continue
		}
		path := filepath.Join("/offline-host", ".local", "bin", resource)
		capabilities = append(capabilities, capabilityprobe.Observation{
			Capability: "ai-cli", ID: descriptor.ID, State: capabilityprobe.Ready,
			Path: path, Version: "1.0.0", ProbedAt: probedAt,
		})
	}
	target := targetConnection{Target: targetmodel.Target{
		ID: targetID, NodeID: "ac6-offline-node", Available: true, OS: runtime.GOOS, Architecture: runtime.GOARCH,
	}, Capabilities: capabilities, BaseURL: bridge.URL, BaseURLExplicit: true, OwnerToken: "Bearer ac6-offline-test-token"}
	server := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{target} }}
	// `use_managed_copy` is the remote adoption path for a harness first found
	// outside the managed prefix. Keep it in this route matrix so the hosted
	// command contract cannot silently lose adoption while lifecycle verbs evolve.
	actions := []string{"use_managed_copy", "install", "doctor", "preview_cleanup", "update", "uninstall"}
	harnesses := 0
	for _, descriptor := range catalog.All() {
		_, ok := agentResource(descriptor.ID)
		if !ok {
			continue
		}
		harnesses++
		planID := ""
		for _, action := range actions {
			idempotencyKey := uuid.NewString()
			result, err := server.runAgentUpdateAction(operatorContext(), targetID, descriptor.ID, action, planID, idempotencyKey)
			if err != nil {
				t.Fatalf("%s %s via offline hosted relay: %v", descriptor.ID, action, err)
			}
			if !result.Success || result.Update.TargetID != targetID {
				t.Fatalf("%s %s returned an unsuccessful or local result: %+v", descriptor.ID, action, result)
			}
			if action == "preview_cleanup" {
				if result.CleanupPlan == nil || result.CleanupPlan.ID == "" {
					t.Fatalf("%s cleanup preview returned no plan: %+v", descriptor.ID, result)
				}
				planID = result.CleanupPlan.ID
			}
			if descriptor.ID == "codex" && action == "update" {
				// Rebuild the Web Console server after the successful hosted command,
				// then replay the same operator intent. Bridge's durable receipt must
				// return the first result without applying a second effect.
				restarted := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{target} }}
				replayed, replayErr := restarted.runAgentUpdateAction(operatorContext(), targetID, descriptor.ID, action, "", idempotencyKey)
				if replayErr != nil || !replayed.Success || replayed.Update.State != agentStateUpToDate {
					t.Fatalf("replayed codex update after Web Console restart = %+v, %v", replayed, replayErr)
				}
				if relay.effects[idempotencyKey] != 1 {
					t.Fatalf("idempotent hosted update effects = %d, want 1", relay.effects[idempotencyKey])
				}
			}
		}
	}
	if harnesses != 5 {
		t.Fatalf("exercised %d supported harnesses, want 5", harnesses)
	}
	if got, want := len(relay.requests), harnesses*len(actions)*2+2; got != want {
		t.Fatalf("relay calls = %d, want %d action+observation calls", got, want)
	}
	for _, request := range relay.requests {
		if request.GetCommand() == "resource update" && request.GetCommandId() == "" {
			t.Fatalf("remote lifecycle request omitted command ID: %+v", request)
		}
	}
	for _, request := range relay.requests {
		if request.GetNodeId() != "ac6-offline-node" || request.GetScenario() != "vrooli" || !strings.HasPrefix(request.GetCommand(), "resource update") {
			t.Fatalf("unexpected hosted relay request: %+v", request)
		}
	}
}
