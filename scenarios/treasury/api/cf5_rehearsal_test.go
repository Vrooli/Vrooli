//go:build treasury_rehearsal

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	routingpb "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	notificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	notificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	"treasury/internal/testutil/basefork"
	"treasury/internal/walletops"
)

// TestCF5ForkHandoffAndTreasuryLifecycle proves that the disposable Anvil
// endpoint and state identity belong to one UI-enabled, lifecycle-managed
// Treasury variant, and that both owners close their listeners afterward.
func TestCF5ForkHandoffAndTreasuryLifecycle(t *testing.T) {
	if os.Getenv("TREASURY_CF5_REHEARSAL") != "1" {
		t.Skip("set TREASURY_CF5_REHEARSAL=1 and TREASURY_CF5_FIXTURE_DIR for the isolated CF5 fork handoff")
	}
	runRoot := strings.TrimSpace(os.Getenv("TREASURY_CF5_FIXTURE_DIR"))
	require.True(t, filepath.IsAbs(runRoot), "fixture directory must be absolute")
	require.NoError(t, os.MkdirAll(runRoot, 0o700))
	runDir := filepath.Join(runRoot, "cf5-handoff-"+time.Now().UTC().Format("20060102T150405.000000000Z"))
	require.NoError(t, os.MkdirAll(runDir, 0o700))
	t.Logf("CF5 handoff evidence=%s", runDir)

	fork := basefork.Start(t)
	forkIdentity := fork.Identity()
	require.Equal(t, "0x2105", forkIdentity.ChainID)
	require.True(t, strings.HasPrefix(forkIdentity.URL, "http://127.0.0.1:"))
	require.NotEmpty(t, forkIdentity.BlockNumber)
	require.NotEmpty(t, forkIdentity.BlockHash)
	require.NotZero(t, forkIdentity.AnvilPID)
	writeCF5JSON(t, filepath.Join(runDir, "fork-identity.json"), forkIdentity)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	_, _, authenticatorURL, closeIdentity := startServiceIdentityFixture(t)
	t.Cleanup(closeIdentity)
	ledger := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ledger.Close)
	hubInstance := instanceName("hub")
	hubName := "notification-hub@" + hubInstance
	hubEnv := map[string]string{"VROOLI_TEST_MODE_FORCE_ENABLE": "1"}
	hubSource := prepareCF5ScenarioVariant(t, "notification-hub", "../../notification-hub", runDir, hubEnv)
	hubStarted := lifecycleRehearsal(t, ctx, runDir, "notification-hub-start", map[string]string{
		"API_TOKEN": "", "VROOLI_TEST_MODE_FORCE_ENABLE": "1",
	}, "start", hubName, "--path", hubSource, "--json", "--timeout", "300")
	hubAPIURL := cf5APIURL(t, hubStarted)
	hubStopped := false
	stopHub := func() {
		if hubStopped {
			return
		}
		stopCF5Variant(t, runDir, hubName, hubAPIURL)
		hubStopped = true
	}
	t.Cleanup(stopHub)
	requireHTTPStatus(t, hubAPIURL+"/health", http.StatusOK)
	leaseID := "cf5-" + hubInstance
	leaseDB := filepath.Join(runDir, "notification-hub-lease.db")
	hubRouting := routingconnect.NewRoutingServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	_, err := hubRouting.InstallTestPool(ctx, connect.NewRequest(&routingpb.InstallTestPoolRequest{Dsn: leaseDB, LeaseId: leaseID, LeaseTtlMs: 8 * 60 * 1000}))
	require.NoError(t, err, "install run-scoped routed Notification Hub fixture")
	leaseActive := true
	clearLease := func() {
		if !leaseActive {
			return
		}
		_, clearErr := hubRouting.ClearTestPool(context.Background(), connect.NewRequest(&routingpb.ClearTestPoolRequest{LeaseId: leaseID}))
		if clearErr != nil {
			t.Errorf("clear Notification Hub routed lease: %v", clearErr)
		}
		leaseActive = false
	}
	t.Cleanup(clearLease)
	agentIdentity := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(agentIdentity.Close)

	instance := instanceName("treasury")
	name := "treasury@" + instance
	apiEnv := map[string]string{
		"TREASURY_REHEARSAL_DIR":              filepath.Join(runDir, "custody"),
		"TREASURY_BASE_RPC_URL":               forkIdentity.URL,
		"TREASURY_BASE_USDC_MODE":             "local-fork",
		"MONEY_LEDGER_API_URL":                ledger.URL,
		"SCENARIO_AUTHENTICATOR_API_URL":      authenticatorURL,
		"AGENT_MANAGER_API_URL":               agentIdentity.URL,
		"NOTIFICATION_HUB_API_URL":            hubAPIURL,
		"TREASURY_CF5_NOTIFICATION_TEST_MODE": "1",
	}
	source := prepareCF5ScenarioVariant(t, "treasury", "..", runDir, apiEnv)
	writeCF5JSON(t, filepath.Join(runDir, "treasury-binding.json"), map[string]any{
		"instance": name, "source": source, "fork": forkIdentity, "api_environment": apiEnv,
	})
	started := lifecycleRehearsal(t, ctx, runDir, "treasury-start", map[string]string{
		"GOFLAGS": "-tags=treasury_rehearsal", "API_TOKEN": "",
		"TREASURY_OPERATOR_TOKEN": "", "TREASURY_MANDATE_SIGNING_KEY": "",
	}, "start", name, "--path", source, "--json", "--timeout", "300")
	apiURL, uiURL := cf5RuntimeURLs(t, started)
	stopped := false
	stop := func() {
		if stopped {
			return
		}
		stopCF5Treasury(t, runDir, name, apiURL, uiURL)
		stopped = true
	}
	t.Cleanup(stop)
	requireHTTPStatus(t, apiURL+"/health", http.StatusOK)
	requireHTTPStatus(t, uiURL+"/health", http.StatusOK)
	requireHTTPStatus(t, uiURL+"/fund", http.StatusOK)

	// Exercise the same Treasury notification adapter against the live Hub
	// variant. Reads use the leased test owner, and every write remains on the
	// run-scoped disposable pool.
	notifier, err := walletops.NewLeasedTestNotificationHub(hubAPIURL)
	require.NoError(t, err)
	require.NoError(t, notifier.Received(ctx, "fixture-operator", "cf5-budget", "0x0000000000000000000000000000000000000001", "1000000", "cf5-receipt-"+hubInstance))
	listReq := connect.NewRequest(&notificationsv1.ListRequest{Limit: 20})
	listReq.Header().Set("X-Vrooli-Test-Mode", "1")
	readClient := notificationsconnect.NewNotificationsServiceClient(&http.Client{Timeout: 10 * time.Second}, hubAPIURL)
	listed, err := readClient.List(ctx, listReq)
	require.NoError(t, err)
	var notificationID string
	for _, notice := range listed.Msg.GetNotifications() {
		if strings.Contains(notice.GetBody(), "credited to budget cf5-budget") {
			notificationID = notice.GetId()
			break
		}
	}
	require.NotEmpty(t, notificationID, "Treasury receipt notification must be persisted in the Hub")
	getReq := connect.NewRequest(&notificationsv1.GetRequest{Id: notificationID})
	getReq.Header().Set("X-Vrooli-Test-Mode", "1")
	readback, err := readClient.Get(ctx, getReq)
	require.NoError(t, err)
	require.Contains(t, readback.Msg.GetNotification().GetBody(), "Received 1.000000 USDC")
	writeCF5JSON(t, filepath.Join(runDir, "notification-hub-delivery.json"), map[string]any{
		"lease_id": leaseID, "hub_variant": hubName, "notification_id": notificationID,
		"title": readback.Msg.GetNotification().GetTitle(), "body": readback.Msg.GetNotification().GetBody(),
		"readback": true, "loopback_url": hubAPIURL,
	})
	clearLease()

	// Stop in the owner body so the receipt contains explicit lifecycle and
	// socket evidence; registered test cleanup is an idempotent crash fallback.
	stop()
	stopHub()
	stopReceipt := fork.Stop(ctx)
	writeCF5JSON(t, filepath.Join(runDir, "fork-stop.json"), stopReceipt)
	require.True(t, stopReceipt.ProcessExited)
	require.False(t, stopReceipt.KilledAfterTimeout)
	require.False(t, stopReceipt.EndpointReachableAfter, "%+v", stopReceipt)
	require.True(t, stopReceipt.Clean, "%+v", stopReceipt)
	writeCF5JSON(t, filepath.Join(runDir, "handoff-result.json"), map[string]any{
		"status": "passed", "instance": name, "api_url": apiURL, "ui_url": uiURL,
		"fork": forkIdentity, "fork_stop": stopReceipt,
	})
}

func instanceName(prefix string) string {
	return "cf5-" + prefix + "-" + time.Now().UTC().Format("150405") + fmt.Sprintf("-%d", os.Getpid())
}

func cf5APIURL(t *testing.T, raw []byte) string {
	t.Helper()
	var result struct {
		Success  bool `json:"success"`
		Scenario struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenario"`
		Scenarios []struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenarios"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Success)
	ports := result.Scenario.Ports
	if len(result.Scenarios) == 1 {
		ports = result.Scenarios[0].Ports
	}
	api := ports["API_PORT"]
	require.NotZero(t, api)
	return fmt.Sprintf("http://127.0.0.1:%d", api)
}

func prepareCF5ScenarioVariant(t *testing.T, scenarioName, repoScenario, runDir string, apiEnv map[string]string) string {
	t.Helper()
	repo, err := filepath.Abs(filepath.Clean(filepath.Join(".", repoScenario)))
	require.NoError(t, err)
	require.DirExists(t, repo)
	workspace := filepath.Join(runDir, "workspace")
	source := filepath.Join(workspace, "scenarios", scenarioName)
	require.NoError(t, os.MkdirAll(source, 0o700))
	repoRoot := filepath.Clean(filepath.Join(repo, "../.."))
	entries, err := os.ReadDir(repoRoot)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.Name() == "scenarios" {
			continue
		}
		require.NoError(t, os.Symlink(filepath.Join(repoRoot, entry.Name()), filepath.Join(workspace, entry.Name())))
	}
	for _, component := range []string{"api", "cli", "ui", ".vrooli"} {
		original := filepath.Join(repo, component)
		if _, statErr := os.Stat(original); statErr != nil {
			continue
		}
		destRoot := filepath.Join(source, component)
		require.NoError(t, filepath.WalkDir(original, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(original, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return os.MkdirAll(destRoot, 0o700)
			}
			if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "data" || entry.Name() == "tmp") {
				if component == "ui" && entry.Name() == "node_modules" {
					if err := os.Symlink(path, filepath.Join(destRoot, rel)); err != nil {
						return err
					}
					return filepath.SkipDir
				}
				return filepath.SkipDir
			}
			dest := filepath.Join(destRoot, rel)
			if entry.Type()&os.ModeSymlink != 0 {
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				return os.Symlink(target, dest)
			}
			if entry.IsDir() {
				return os.MkdirAll(dest, 0o700)
			}
			if strings.HasSuffix(entry.Name(), "-api") || strings.HasPrefix(entry.Name(), ".vrooli-closure") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
			if err != nil {
				return err
			}
			_, err = io.Copy(out, in)
			closeErr := out.Close()
			if err != nil {
				return err
			}
			return closeErr
		}))
	}
	manifest := filepath.Join(source, ".vrooli", "service.json")
	var config map[string]any
	readRehearsalJSON(t, manifest, &config)
	config["credentials"] = map[string]any{"descriptors": []any{}}
	config["dependencies"] = map[string]any{"scenarios": map[string]any{}, "resources": map[string]any{}}
	config["cli"].(map[string]any)["enabled"] = false
	config["components"].(map[string]any)["api"].(map[string]any)["run"].(map[string]any)["env"] = apiEnv
	raw, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0o600))
	return source
}

func cf5RuntimeURLs(t *testing.T, raw []byte) (string, string) {
	t.Helper()
	var result struct {
		Success   bool `json:"success"`
		Scenarios []struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenarios"`
		Scenario struct {
			Ports map[string]int `json:"ports"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Success)
	if len(result.Scenarios) == 1 {
		result.Scenario.Ports = result.Scenarios[0].Ports
	}
	api, ui := result.Scenario.Ports["API_PORT"], result.Scenario.Ports["UI_PORT"]
	require.NotZero(t, api)
	require.NotZero(t, ui, "CF5 rehearsal requires the real Treasury UI component")
	return fmt.Sprintf("http://127.0.0.1:%d", api), fmt.Sprintf("http://127.0.0.1:%d", ui)
}

func stopCF5Treasury(t *testing.T, dir, name, apiURL, uiURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	output := lifecycleRehearsal(t, ctx, dir, "treasury-stop", map[string]string{}, "stop", name, "--json")
	var status struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(output, &status))
	require.True(t, status.Success)
	statusOutput := lifecycleRehearsal(t, ctx, dir, "treasury-status-after-stop", map[string]string{}, "status", name, "--json")
	var stopped struct {
		Success  bool `json:"success"`
		Scenario struct {
			Name      string `json:"name"`
			Status    string `json:"status"`
			Processes int    `json:"processes"`
			StartOp   struct {
				Variant string `json:"variant"`
			} `json:"start_operation"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(statusOutput, &stopped))
	require.True(t, stopped.Success)
	require.Equal(t, "treasury", stopped.Scenario.Name)
	require.Equal(t, strings.TrimPrefix(name, "treasury@"), stopped.Scenario.StartOp.Variant)
	require.Equal(t, "stopped", stopped.Scenario.Status)
	require.Zero(t, stopped.Scenario.Processes)
	probes := map[string]bool{
		"api": cf5HTTPReachable(apiURL + "/health"),
		"ui":  cf5HTTPReachable(uiURL + "/health"),
	}
	writeCF5JSON(t, filepath.Join(dir, "treasury-listener-probes.json"), probes)
	require.False(t, probes["api"], "Treasury API listener remained after lifecycle stop")
	require.False(t, probes["ui"], "Treasury UI listener remained after lifecycle stop")
}

func stopCF5Variant(t *testing.T, dir, name, apiURL string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	stop := lifecycleRehearsal(t, ctx, dir, "variant-stop", map[string]string{}, "stop", name, "--json")
	var stopped struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(stop, &stopped))
	require.True(t, stopped.Success)
	statusRaw := lifecycleRehearsal(t, ctx, dir, "variant-status-after-stop", map[string]string{}, "status", name, "--json")
	var status struct {
		Success  bool `json:"success"`
		Scenario struct {
			Status    string `json:"status"`
			Processes int    `json:"processes"`
			StartOp   struct {
				Variant string `json:"variant"`
			} `json:"start_operation"`
		} `json:"scenario"`
	}
	require.NoError(t, json.Unmarshal(statusRaw, &status))
	require.True(t, status.Success)
	parts := strings.SplitN(name, "@", 2)
	require.Len(t, parts, 2)
	require.Equal(t, parts[1], status.Scenario.StartOp.Variant)
	require.Equal(t, "stopped", status.Scenario.Status)
	require.Zero(t, status.Scenario.Processes)
	require.False(t, cf5HTTPReachable(apiURL+"/health"), "variant API listener remained after lifecycle stop")
}

func requireHTTPStatus(t *testing.T, url string, want int) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(url)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, want, response.StatusCode, url)
}

func cf5HTTPReachable(url string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	response, err := client.Get(url)
	if err != nil {
		return false
	}
	_ = response.Body.Close()
	return true
}

func writeCF5JSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
}
