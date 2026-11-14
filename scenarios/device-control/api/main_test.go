package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"device-control/internal/desktophelper"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestDesktopHelperLifecycleInventorySurfacesPendingReceiptAndStillBlocksExit(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, "state")
	require.NoError(t, os.Mkdir(stateDir, 0700))
	require.NoError(t, os.Chmod(dir, 0700))
	surface := targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}
	helper := desktophelper.Config{Version: 1, Surface: surface, SessionID: "os-session", StateDirectory: stateDir, XAuthorityFile: filepath.Join(dir, "authority"), GrantStatusFile: filepath.Join(dir, "grants"), PublicKey: base64.StdEncoding.EncodeToString(make([]byte, 32))}
	helperConfigPath := filepath.Join(dir, "helper.json")
	writeJSON := func(path string, value any) {
		data, err := json.Marshal(value)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0600))
	}
	writeJSON(helperConfigPath, helper)
	ownerBootstrapPath := filepath.Join(dir, "owner.json")
	owner := desktophelper.OwnerBootstrap{DeviceID: "host-desktop", HelperConfigPath: helperConfigPath, Helper: helper}
	owner.Helper.PublicKey = ""
	writeJSON(ownerBootstrapPath, owner)
	writeJSON(filepath.Join(stateDir, "registration.json"), desktophelper.Registration{HelperID: "helper-generation-4", Epoch: 4, SessionID: helper.SessionID, Surface: surface, SocketPath: filepath.Join(stateDir, "helper.sock")})
	receipt := desktophelper.LifecycleReceipt{HelperID: "helper-generation-4", Epoch: 4, SessionID: helper.SessionID, Surface: surface, Status: "stopping", ActiveRequests: 2, ErrorCode: "rpc_drain_timeout", ObservedAt: time.Now().UTC()}
	require.NoError(t, desktophelper.WriteLifecycleReceipt(stateDir, receipt))

	work, blockers, complete := desktopHelperLifecycleInventory(context.Background(), ownerBootstrapPath)
	require.True(t, complete, "a valid receipt completes enumeration even while shutdown is pending")
	require.Contains(t, lifecycleWorkStatus(work, "desktop-helper-shutdown"), "stopping")
	require.Contains(t, lifecycleWorkStatus(work, "desktop-helper-process-exit"), "unknown")
	require.Contains(t, lifecycleBlockerCode(blockers, "desktop_helper_shutdown_pending"), "desktop_helper_shutdown_pending")
	require.Contains(t, lifecycleBlockerCode(blockers, "desktop_helper_process_exit_unobserved"), "desktop_helper_process_exit_unobserved")
	components, _, _, enumerated := desktopHelperManagedInventory(context.Background(), ownerBootstrapPath, nil)
	require.False(t, enumerated, "owner work can be enumerated while exact component generation is unavailable")
	require.Empty(t, components)

	// Launch binding makes enumeration complete while exit remains a blocker.
	receipt.ComponentGeneration = "component-generation"
	require.NoError(t, desktophelper.WriteLifecycleReceipt(stateDir, receipt))
	writeJSON(filepath.Join(stateDir, "registration.json"), desktophelper.Registration{HelperID: receipt.HelperID, Epoch: receipt.Epoch, SessionID: helper.SessionID, Surface: surface, SocketPath: filepath.Join(stateDir, "helper.sock"), ComponentGeneration: receipt.ComponentGeneration})
	components, managedWork, managedBlockers, enumerated := desktopHelperManagedInventory(context.Background(), ownerBootstrapPath, nil)
	require.True(t, enumerated)
	require.Len(t, components, 1)
	require.Equal(t, "unknown", lifecycleWorkStatus(managedWork, "desktop-helper-process-exit"))
	require.NotEmpty(t, managedBlockers)
	_, managedWork, managedBlockers, enumerated = desktopHelperManagedInventory(context.Background(), ownerBootstrapPath, map[string]string{"desktop-helper": "other-generation"})
	require.True(t, enumerated)
	require.Equal(t, "unknown", lifecycleWorkStatus(managedWork, "desktop-helper-process-exit"))
	require.NotEmpty(t, managedBlockers)
	_, managedWork, managedBlockers, enumerated = desktopHelperManagedInventory(context.Background(), ownerBootstrapPath, map[string]string{"desktop-helper": receipt.ComponentGeneration})
	require.True(t, enumerated)
	require.Equal(t, "completed", lifecycleWorkStatus(managedWork, "desktop-helper-process-exit"))
	require.NotEmpty(t, managedBlockers, "process exit cannot hide pending controller cleanup")

	receipt.Status = "running"
	receipt.ActiveRequests = 0
	receipt.ErrorCode = ""
	receipt.ObservedAt = time.Now().UTC()
	require.NoError(t, desktophelper.WriteLifecycleReceipt(stateDir, receipt))
	work, blockers, complete = desktopHelperLifecycleInventory(context.Background(), ownerBootstrapPath)
	require.True(t, complete, "a last-known running receipt still enumerates its owner work")
	require.Equal(t, "unknown", lifecycleWorkStatus(work, "desktop-helper-shutdown"))
	require.Contains(t, lifecycleBlockerCode(blockers, "desktop_helper_liveness_unconfirmed"), "desktop_helper_liveness_unconfirmed")

	receipt.Status = "controller-cleanup-completed"
	receipt.ActiveRequests = 0
	receipt.ErrorCode = ""
	receipt.ObservedAt = time.Now().UTC()
	require.NoError(t, desktophelper.WriteLifecycleReceipt(stateDir, receipt))
	work, blockers, complete = desktopHelperLifecycleInventory(context.Background(), ownerBootstrapPath)
	require.True(t, complete, "cleanup receipt plus registration enumerates the process-exit work item")
	require.Equal(t, "completed", lifecycleWorkStatus(work, "desktop-helper-shutdown"))
	require.Equal(t, "unknown", lifecycleWorkStatus(work, "desktop-helper-process-exit"))
	require.Contains(t, lifecycleBlockerCode(blockers, "desktop_helper_process_exit_unobserved"), "desktop_helper_process_exit_unobserved")
	_, managedWork, managedBlockers, enumerated = desktopHelperManagedInventory(context.Background(), ownerBootstrapPath, map[string]string{"desktop-helper": receipt.ComponentGeneration})
	require.True(t, enumerated)
	require.Equal(t, "completed", lifecycleWorkStatus(managedWork, "desktop-helper-process-exit"))
	require.Empty(t, managedBlockers, "matching owner wait plus successful cleanup must release the blocker")
}

func lifecycleWorkStatus(work []*commonv1.LifecycleWorkItem, kind string) string {
	for _, item := range work {
		if item.GetKind() == kind {
			return item.GetStatus()
		}
	}
	return ""
}

func lifecycleBlockerCode(blockers []*commonv1.LifecycleBlocker, code string) string {
	for _, blocker := range blockers {
		if blocker.GetCode() == code {
			return blocker.GetCode()
		}
	}
	return ""
}
