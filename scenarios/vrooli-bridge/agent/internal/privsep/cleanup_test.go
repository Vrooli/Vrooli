package privsep_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"os"
	"strings"
	"sync"
	"testing"

	"vrooli-bridge/agent/internal/privsep"

	"github.com/stretchr/testify/require"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/cleanup"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
	"github.com/vrooli/vrooli/packages/proto/sealing"
)

type cleanupRunner struct {
	mu             sync.Mutex
	argvs          [][]string
	inputs         [][]byte
	envs           [][]string
	dirs           []string
	statusComplete bool
}

func (r *cleanupRunner) Run(_ context.Context, argv []string, _ string, onLog func(string)) (int, error) {
	r.mu.Lock()
	r.argvs = append(r.argvs, append([]string(nil), argv...))
	r.mu.Unlock()
	if len(argv) >= 4 && argv[1] == "break-glass" && argv[2] == "status" {
		if r.statusComplete {
			onLog(`{"complete":true}`)
		} else {
			onLog(`{"complete":false}`)
		}
	} else {
		onLog(`{"plan_hash":"frozen"}`)
	}
	return 0, nil
}

func TestCleanupHelperTreatsExistingBreakGlassAsProtectedWithoutOverwrite(t *testing.T) {
	nodePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	seedPath := t.TempDir() + "/node_credential.key"
	require.NoError(t, os.WriteFile(seedPath, nodePrivate.Bytes(), 0o600))
	runner := &cleanupRunner{statusComplete: true}
	h := privsep.NewHelper("vrooli", "/work", nil, privsep.WithStepRunner(runner), privsep.WithSealingSeedPath(seedPath))
	var events []*cleanupv1.CleanupEvent
	err = h.Cleanup(context.Background(), &channelv1.CleanupCommand{
		Operation: channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS,
		OpId:      "already-protected", MachineId: "machine-1", NodeId: "node-1", Target: "mini.local", Scope: "all",
	}, func(event *cleanupv1.CleanupEvent) error { events = append(events, event); return nil })
	require.NoError(t, err)
	require.Len(t, runner.argvs, 1, "only the local status check should run")
	require.Contains(t, events[len(events)-2].GetReason(), "already present; unchanged")
	require.Equal(t, int32(0), events[len(events)-1].GetExitCode())
}

func (r *cleanupRunner) RunWithInput(_ context.Context, argv []string, _ string, input []byte, onLog func(string)) (int, error) {
	r.mu.Lock()
	r.argvs = append(r.argvs, append([]string(nil), argv...))
	r.inputs = append(r.inputs, append([]byte(nil), input...))
	r.mu.Unlock()
	onLog(`{"ok":true}`)
	return 0, nil
}

func (r *cleanupRunner) RunWithEnvironment(ctx context.Context, argv []string, dir string, env []string, onLog func(string)) (int, error) {
	r.mu.Lock()
	r.envs = append(r.envs, append([]string(nil), env...))
	r.dirs = append(r.dirs, dir)
	r.mu.Unlock()
	return r.Run(ctx, argv, dir, onLog)
}

func (r *cleanupRunner) RunWithInputEnvironment(ctx context.Context, argv []string, dir string, input []byte, env []string, onLog func(string)) (int, error) {
	r.mu.Lock()
	r.envs = append(r.envs, append([]string(nil), env...))
	r.dirs = append(r.dirs, dir)
	r.mu.Unlock()
	return r.RunWithInput(ctx, argv, dir, input, onLog)
}

func TestCleanupHelperBindsRunnerHomeForPrivilegedCLI(t *testing.T) {
	runner := &cleanupRunner{}
	h := privsep.NewHelper("vrooli", "/work", nil,
		privsep.WithStepRunner(runner),
		privsep.WithClientHome("/Users/runner"),
	)
	var events []*cleanupv1.CleanupEvent
	err := h.Cleanup(context.Background(), &channelv1.CleanupCommand{
		Operation: channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION,
		OpId:      "inventory-home",
		PlanId:    "inventory-home",
		Target:    "mini",
		Scope:     "all",
	}, func(event *cleanupv1.CleanupEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, []string{
		"VROOLI_ROOT=/work",
		"VROOLI_SOURCE_ROOT=/work",
		"HOME=/Users/runner",
		"VROOLI_BREAK_GLASS_DIR=/Users/runner/.vrooli/identity/break-glass",
	}, runner.envs[0])
	require.Equal(t, os.TempDir(), runner.dirs[0])
}

func TestCleanupHelperCarriesDeferredBridgeServicesForSelfCleanup(t *testing.T) {
	runner := &cleanupRunner{}
	h := privsep.NewHelper("vrooli", "/work", nil,
		privsep.WithStepRunner(runner),
		privsep.WithDeferredServiceNames("vrooli-bridge-agent", "vrooli-bridge-provisioner"),
	)
	err := h.Cleanup(context.Background(), &channelv1.CleanupCommand{
		Operation: channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION,
		OpId:      "inventory-deferred-services",
		PlanId:    "inventory-deferred-services",
		Target:    "mini",
		Scope:     "all",
	}, func(*cleanupv1.CleanupEvent) error { return nil })
	require.NoError(t, err)
	require.Len(t, runner.envs, 1)
	require.Equal(t, []string{"VROOLI_ROOT=/work", "VROOLI_SOURCE_ROOT=/work", "VROOLI_BRIDGE_DEFER_SERVICE_STOPS=vrooli-bridge-agent,vrooli-bridge-provisioner"}, runner.envs[0])
}

func TestCleanupHelperOpensSealedPassphraseOnlyAtNode(t *testing.T) {
	nodePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	public := nodePrivate.PublicKey().Bytes()
	seedPath := t.TempDir() + "/node_credential.key"
	require.NoError(t, os.WriteFile(seedPath, nodePrivate.Bytes(), 0o600))
	cmd := &channelv1.CleanupCommand{
		Operation:         channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS,
		OpId:              "operation-1",
		MachineId:         "machine-1",
		NodeId:            "node-1",
		Target:            "mini.local",
		Scope:             "all",
		PlanId:            "operation-1",
		OperatorId:        "operator-1",
		OperatorConfirmed: true,
	}
	aad := sealing.Context(cmd.GetMachineId(), cmd.GetNodeId(), cmd.GetTarget(), cmd.GetScope(), cmd.GetPlanHash(), cmd.GetPlanId(), cmd.GetOperatorId())
	cmd.SealedPassphrase, err = sealing.Seal(public, []byte("correct horse"), aad)
	require.NoError(t, err)
	runner := &cleanupRunner{}
	h := privsep.NewHelper("vrooli", "/work", nil, privsep.WithStepRunner(runner), privsep.WithSealingSeedPath(seedPath))
	var events []*cleanupv1.CleanupEvent
	err = h.Cleanup(context.Background(), cmd, func(event *cleanupv1.CleanupEvent) error {
		events = append(events, event)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, "completed", events[len(events)-2].GetStatus())
	require.Len(t, runner.inputs, 1)
	require.Equal(t, "correct horse", string(runner.inputs[0]))
	for _, argv := range runner.argvs {
		require.NotContains(t, strings.Join(argv, " "), "correct horse")
	}
}

func TestCleanupHelperRefusesUnknownOperationByName(t *testing.T) {
	h := privsep.NewHelper("vrooli", "/work", nil)
	err := h.Cleanup(context.Background(), &channelv1.CleanupCommand{Operation: channelv1.PrivilegedOperation(99)}, func(*cleanupv1.CleanupEvent) error { return nil })
	require.Error(t, err)
	require.Contains(t, err.Error(), "enum:99")
	_, ok := privilegedops.Parse("remove-everything")
	require.False(t, ok)
}

func TestCleanupHelperRefusesApplyWithoutOperatorConfirmation(t *testing.T) {
	h := privsep.NewHelper("vrooli", "/work", nil)
	var events []*cleanupv1.CleanupEvent
	err := h.Cleanup(context.Background(), &channelv1.CleanupCommand{
		Operation: channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN,
		OpId:      "operation-unconfirmed",
		PlanId:    "plan-1",
	}, func(event *cleanupv1.CleanupEvent) error {
		events = append(events, event)
		return nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator_confirmed")
	require.Len(t, events, 2)
	require.Equal(t, "blocked", events[0].GetStatus())
	require.Equal(t, int32(2), events[1].GetExitCode())
}

func TestCleanupHelperRoutesEveryNamedCleanupOperation(t *testing.T) {
	nodePrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	require.NoError(t, err)
	public := nodePrivate.PublicKey().Bytes()
	seedPath := t.TempDir() + "/node_credential.key"
	require.NoError(t, os.WriteFile(seedPath, nodePrivate.Bytes(), 0o600))
	runner := &cleanupRunner{}
	h := privsep.NewHelper("vrooli", "/work", nil, privsep.WithStepRunner(runner), privsep.WithSealingSeedPath(seedPath))

	operations := []channelv1.PrivilegedOperation{
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_INVENTORY_INSTALLATION,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PLAN_UNINSTALL,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_PROVISION_BREAK_GLASS,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ISSUE_CLEANUP_CAPABILITY,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_VERIFY_RESULT,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_ROTATE_BREAK_GLASS,
		channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_RESET_BREAK_GLASS,
	}
	for _, operation := range operations {
		t.Run(privilegedops.Name(operation), func(t *testing.T) {
			cmd := &channelv1.CleanupCommand{
				Operation:         operation,
				OpId:              "operation-" + privilegedops.Name(operation),
				PlanId:            "plan-1",
				MachineId:         "machine-1",
				NodeId:            "node-1",
				Target:            "mini.local",
				Scope:             "all",
				PlanHash:          "hash-1",
				OperatorId:        "operator-1",
				OperatorConfirmed: true,
			}
			if operation == channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_APPLY_FROZEN_PLAN {
				cmd.Capability = []byte("opaque-capability")
			} else if operation != channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_VERIFY_RESULT && operation != channelv1.PrivilegedOperation_PRIVILEGED_OPERATION_RESET_BREAK_GLASS {
				cmd.SealedPassphrase, err = sealing.Seal(public, []byte("correct horse"), sealing.Context(cmd.MachineId, cmd.NodeId, cmd.Target, cmd.Scope, cmd.PlanHash, cmd.PlanId, cmd.OperatorId))
				require.NoError(t, err)
			}
			before := len(runner.argvs)
			var events []*cleanupv1.CleanupEvent
			require.NoError(t, h.Cleanup(context.Background(), cmd, func(event *cleanupv1.CleanupEvent) error {
				events = append(events, event)
				return nil
			}))
			require.NotEmpty(t, events)
			require.Equal(t, int32(0), events[len(events)-1].GetExitCode())
			require.Greater(t, len(runner.argvs), before)
		})
	}
}
