//go:build linux

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	cf1TreasuryRunSecret = "local-fork-only-agent-manager-identity-secret"
	cf1TreasuryOperator  = "cf1-local-test-operator"
	cf1RunKeyPassphrase  = "local-fork-only-encrypted-authority-passphrase"
	cf1RunKeyField       = "treasury-agent-session-key"
)

func TestTreasuryCLIFromOrdinaryRun(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-004] [REQ:TRS-P0-006]
	coord := strings.TrimSpace(os.Getenv("TREASURY_CF1_COORD_DIR"))
	cf7Only := os.Getenv("TREASURY_CF7_ONLY") == "1"
	cf2AgentManagerOnly := os.Getenv("TREASURY_CF2_AGENT_MANAGER_ONLY") == "1"
	if cf2AgentManagerOnly && cf7Only {
		t.Fatal("CF2 Agent Manager order mode and CF7 budget mode are mutually exclusive")
	}
	if cf2AgentManagerOnly {
		coord = strings.TrimSpace(os.Getenv("TREASURY_CF2_AGENT_MANAGER_COORD_DIR"))
	}
	if cf7Only {
		coord = strings.TrimSpace(os.Getenv("TREASURY_CF7_COORD_DIR"))
	}
	if coord == "" {
		if cf2AgentManagerOnly {
			t.Skip("local-fork harness sets TREASURY_CF2_AGENT_MANAGER_COORD_DIR")
		}
		t.Skip("local-fork harness sets TREASURY_CF1_COORD_DIR")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := os.MkdirAll(coord, 0o700); err != nil {
		t.Fatal(err)
	}
	tmp, err := os.MkdirTemp("/tmp", "cf1-am-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmp) })
	if err := os.MkdirAll(filepath.Join(tmp, "vrooli-agent-manager-"+strconv.Itoa(os.Getuid())), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmp)
	t.Setenv("VROOLI_VARIANT", "live")

	db, dbCleanup := testutil.SetupTestDB(t)
	defer dbCleanup()
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	authorityPath := os.Getenv("TREASURY_CF1_AUTHORITY_PATH")
	if cf2AgentManagerOnly {
		authorityPath = filepath.Join(coord, "agent-manager-run-keys.enc")
		if _, err := credentialauthority.OpenEncryptedFile(authorityPath, cf1RunKeyPassphrase); err != nil {
			t.Fatalf("initialize isolated CF2 Agent Manager run-key authority: %v", err)
		}
	}
	if !filepath.IsAbs(authorityPath) {
		t.Fatal("isolated encrypted Credential Authority path must be absolute")
	}
	fixtureKeys := []string{fmt.Sprintf("%064x", 1), fmt.Sprintf("%064x", 2), fmt.Sprintf("%064x", 3), fmt.Sprintf("%064x", 4)}
	fixtureKeyIndex := 0
	keyStore := orchestration.NewCredentialAuthorityTreasurySessionKeysWithGenerator(func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(authorityPath, cf1RunKeyPassphrase)
	}, func() (string, error) {
		if fixtureKeyIndex >= len(fixtureKeys) {
			return "", errors.New("CF1 stale-key probe exhausted deterministic key fixtures")
		}
		key := fixtureKeys[fixtureKeyIndex]
		fixtureKeyIndex++
		return key, nil
	})
	keys := &cf1RunSessionKeys{store: keyStore, path: authorityPath}
	runResultPath := filepath.Join(coord, "run-result.json")
	var service *orchestration.Orchestrator
	mockRunner := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mockRunner.ExecuteFunc = func(execCtx context.Context, request runner.ExecuteRequest) (*runner.ExecuteResult, error) {
		if request.Transcript != nil && request.Transcript.OnProcessStart != nil {
			if err := request.Transcript.OnProcessStart(os.Getpid(), os.Getpid()); err != nil {
				return nil, fmt.Errorf("record ordinary run process identity: %w", err)
			}
		}
		write := func(name string, value any) error {
			data, err := json.MarshalIndent(value, "", "  ")
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(coord, name), data, 0o600)
		}
		if err := write("run-id.json", map[string]any{
			"run_id": request.RunID.String(), "runner": string(request.ResolvedConfig.RunnerType),
			"requested_scopes": []string{"treasury:read", "treasury:write"}, "ordinary_create_run": true,
		}); err != nil {
			return nil, err
		}
		configPath := filepath.Join(coord, "treasury-config.json")
		if cf2AgentManagerOnly {
			var config cf2AgentManagerOrderConfig
			if err := waitForFile(execCtx, configPath, &config); err != nil {
				return nil, fmt.Errorf("wait for CF2 Agent Manager order config: %w", err)
			}
			order, err := executeCF2AgentManagerOrder(execCtx, request.RunID.String(), config, coord, write)
			if err != nil {
				return nil, err
			}
			if request.EventSink == nil {
				return nil, errors.New("Agent Manager run event sink is unavailable")
			}
			if err := request.EventSink.Emit(domain.NewMessageEvent(request.RunID, "assistant", "Completed the canonical LPBS company-market-scan service order through the local-fork x402 path.")); err != nil {
				return nil, fmt.Errorf("record CF2 Agent Manager order completion message: %w", err)
			}
			if err := write("agent-manager-lpbs-result.json", map[string]any{
				"run_id": request.RunID.String(), "order_id": order.OrderID, "job_id": order.JobID,
				"agent_manager_run_id": order.AgentManagerRunID,
				"authorization_id":     order.AuthorizationID, "capture_id": order.CaptureID,
				"posting_id": order.LedgerPostingID, "receipt_id": order.LedgerReceiptID,
				"state": order.State, "offer_id": order.OfferID, "offer_version": order.OfferVersion,
				"amount_minor": order.AmountMinor, "currency": order.Currency,
			}); err != nil {
				return nil, err
			}
			return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "canonical LPBS x402 service order completed by an ordinary Agent Manager run"}}, nil
		}
		var config cf1TreasuryRunConfig
		if err := waitForFile(execCtx, configPath, &config); err != nil {
			return nil, err
		}
		if cf7Only {
			if err := executeCF7X402Run(execCtx, db, service, request, config, coord, write); err != nil {
				return nil, err
			}
			if request.EventSink == nil {
				return nil, errors.New("Agent Manager run event sink is unavailable")
			}
			if err := request.EventSink.Emit(domain.NewMessageEvent(request.RunID, "assistant", "Settled three priced local-fork research calls, then settled the fourth over-cap call after its separate Treasury approval and exact retry.")); err != nil {
				return nil, fmt.Errorf("record CF7 run completion message: %w", err)
			}
			if err := write("run-result.json", map[string]any{
				"run_id": request.RunID.String(), "profile_id": request.Profile.ID.String(), "task_id": request.Task.ID.String(),
				"ordinary_create_run": true, "settled_calls": 4, "fourth_call_dispatch": "after_exact_treasury_approval",
				"x402_calls": config.X402Calls,
			}); err != nil {
				return nil, err
			}
			return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "CF7 local-fork x402 sequence completed after a separately approved over-cap call and same-receipt retry"}}, nil
		}
		if config.APIBaseURL == "" || config.CLIBinary == "" || config.MandateID == "" || config.InCapInstrumentID == "" || config.InCapAmountMinor <= 0 || config.OverCapInstrumentID == "" || config.OverCapAmountMinor <= 0 || config.ExpiryMandateID == "" || config.ExpiryAuthorization == "" || config.ExpirySettlement == "" || config.ExpiryInstrumentID == "" {
			return nil, errors.New("Treasury run config is incomplete")
		}
		if err := validateCF1TreasuryCounterpartIdentity(config, request.RunID.String()); err != nil {
			return nil, fmt.Errorf("validate paired Treasury counterpart before ordinary-run authorization: %w", err)
		}
		if os.Getenv("TREASURY_CF1_WAIT_FOR_FUNDING") == "1" {
			var funding struct {
				BudgetID        string `json:"budget_id"`
				WalletID        string `json:"wallet_id"`
				Address         string `json:"address"`
				TransactionHash string `json:"transaction_hash"`
				AtomicAmount    string `json:"atomic_amount"`
				Accepted        bool   `json:"accepted"`
			}
			if err := waitForFile(execCtx, filepath.Join(coord, "funding-accepted.json"), &funding); err != nil {
				return nil, fmt.Errorf("wait for the run-bound local-fork deposit to be accepted: %w", err)
			}
			expectedAtoms := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPECTED_FUNDING_ATOMS"))
			if funding.BudgetID != config.BudgetID || funding.WalletID == "" || funding.Address == "" || funding.TransactionHash == "" || !funding.Accepted || expectedAtoms == "" || funding.AtomicAmount != expectedAtoms {
				return nil, fmt.Errorf("run-bound deposit evidence does not match the configured budget and exact accepted amount: %+v", funding)
			}
		}
		if request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"] == "" {
			return nil, errors.New("ordinary run did not receive its verified identity")
		}
		if request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] != "" {
			return nil, errors.New("run-owned Treasury session key was exposed before the approved launch boundary")
		}
		if request.PrepareLaunch == nil {
			return nil, errors.New("ordinary run is missing its launch-scoped Treasury credential provider")
		}
		// The local Treasury harness can publish its run config before this
		// integration test has approved the first exact operation grant. Keep
		// the ordinary runner live until that approval is durably recorded; the
		// CLI must resolve the current grant from Agent Manager before its first
		// Treasury read, and must never race ahead on the baseline identity.
		if !cf7Only {
			if err := waitForFile(execCtx, filepath.Join(coord, "am-operation-approved.json"), nil); err != nil {
				return nil, fmt.Errorf("wait for initial Treasury operation grant approval: %w", err)
			}
		}
		env := make([]string, 0, len(os.Environ())+len(request.Environment)+2)
		env = append(env, os.Environ()...)
		for key, value := range request.Environment {
			env = append(env, key+"="+value)
		}
		env = append(env, "API_BASE_URL="+strings.TrimRight(config.APIBaseURL, "/"))
		setEnv := func(key, value string) {
			prefix := key + "="
			for i := range env {
				if strings.HasPrefix(env[i], prefix) {
					env[i] = prefix + value
					return
				}
			}
			env = append(env, prefix+value)
		}
		initialSessionKey := ""
		runCommand := func(cmd *exec.Cmd) ([]byte, error) {
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			launch := &runner.LaunchRequest{Env: append([]string(nil), cmd.Env...)}
			release, launchErr := request.PrepareLaunch(execCtx, launch)
			if launchErr != nil {
				return nil, fmt.Errorf("prepare run-owned Treasury CLI launch: %w", launchErr)
			}
			if initialSessionKey == "" {
				const sessionKeyPrefix = "VROOLI_TREASURY_SESSION_PRIVATE_KEY="
				for _, entry := range launch.Env {
					if strings.HasPrefix(entry, sessionKeyPrefix) {
						initialSessionKey = strings.TrimPrefix(entry, sessionKeyPrefix)
						break
					}
				}
			}
			cmd.Env = launch.Env
			startErr := cmd.Start()
			if release != nil {
				release()
			}
			if startErr != nil {
				return output.Bytes(), startErr
			}
			waitErr := cmd.Wait()
			return output.Bytes(), waitErr
		}
		runCLI := func(label string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(execCtx, config.CLIBinary, args...)
			cmd.Env = append([]string(nil), env...)
			output, runErr := runCommand(cmd)
			if writeErr := os.WriteFile(filepath.Join(coord, label+".log"), output, 0o600); writeErr != nil {
				return output, writeErr
			}
			if runErr != nil {
				return output, fmt.Errorf("Treasury CLI %s failed: %w: %s", label, runErr, strings.TrimSpace(string(output)))
			}
			return output, nil
		}
		agentWalletOutput, err := runCLI("agent-wallet", "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", config.BudgetID, "--mandate-id", config.MandateID, "--json")
		if err != nil {
			return nil, err
		}
		agentWallet, err := parseCF1AgentWalletOutput(agentWalletOutput, config.BudgetID, config.MandateID, request.RunID.String())
		if err != nil {
			return nil, fmt.Errorf("validate ordinary-run Treasury wallet binding: %w", err)
		}
		foreignWalletProbeStatus, foreignMandateProbeStatus := "", ""
		foreignBudgetID, foreignMandateID, probeIDsErr := config.foreignProbeIDs()
		if probeIDsErr != nil {
			return nil, probeIDsErr
		}
		{
			probeForeignTuple := func(label, budgetID, mandateID string) (string, error) {
				probeCmd := exec.CommandContext(execCtx, config.CLIBinary, "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", budgetID, "--mandate-id", mandateID, "--json")
				probeCmd.Env = append([]string(nil), env...)
				probeOutput, probeErr := runCommand(probeCmd)
				if writeErr := os.WriteFile(filepath.Join(coord, label+".log"), probeOutput, 0o600); writeErr != nil {
					return "", writeErr
				}
				if probeErr == nil {
					return "", fmt.Errorf("ordinary Agent Manager run read another run's budget wallet through %s", label)
				}
				probeText := strings.ToLower(string(probeOutput))
				if !strings.Contains(probeText, "permission_denied") || !strings.Contains(probeText, "mandate wallet session is inactive") {
					return "", fmt.Errorf("%s failed for an unexpected reason: %s", label, strings.TrimSpace(string(probeOutput)))
				}
				return "permission_denied", nil
			}
			foreignWalletProbeStatus, err = probeForeignTuple("cross-budget-wallet-probe", foreignBudgetID, config.MandateID)
			if err != nil {
				return nil, err
			}
			foreignMandateProbeStatus, err = probeForeignTuple("cross-mandate-wallet-probe", config.BudgetID, foreignMandateID)
			if err != nil {
				return nil, err
			}
			if _, err := probeForeignTuple("cross-budget-and-mandate-wallet-probe", foreignBudgetID, foreignMandateID); err != nil {
				return nil, err
			}
			if err := write("cross-budget-wallet-denied.json", map[string]any{
				"requesting_run": request.RunID.String(), "target_budget_id": foreignBudgetID,
				"target_mandate_id": foreignMandateID, "result": "denied", "address_returned": false,
				"foreign_wallet_probe_status": foreignWalletProbeStatus, "foreign_mandate_probe_status": foreignMandateProbeStatus,
				"authorization_scope_includes_target": true, "reason": "agent subject/run mismatch prevents session-key binding",
			}); err != nil {
				return nil, err
			}
		}
		if _, err := runCLI("budget-headroom-before", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
		if config.PairedProbeBarrierDir != "" {
			probeMarker := cf1PairProbeMarker{
				RunID: request.RunID.String(), CounterpartRunID: config.CounterpartIdentity.RunID,
				TargetBudgetID: foreignBudgetID, TargetMandateID: foreignMandateID,
				CounterpartMandateStatus: config.CounterpartIdentity.MandateStatus,
				CounterpartGeneration:    config.CounterpartIdentity.SessionGeneration,
				ForeignWalletProbeStatus: foreignWalletProbeStatus, ForeignMandateProbeStatus: foreignMandateProbeStatus,
			}
			if err := writeCF1PairBarrierMarker(config.PairedProbeBarrierDir, "probe-complete", request.RunID.String(), probeMarker); err != nil {
				return nil, fmt.Errorf("record paired probe completion: %w", err)
			}
			var probeRelease cf1PairProbeRelease
			if err := waitForFile(execCtx, cf1PairBarrierPath(config.PairedProbeBarrierDir, "probe-release", request.RunID.String()), &probeRelease); err != nil {
				return nil, fmt.Errorf("wait for Treasury paired-probe active-mandate check: %w", err)
			}
			if err := validateCF1PairProbeRelease(probeRelease, request.RunID.String(), config.CounterpartIdentity.RunID); err != nil {
				return nil, err
			}
		}
		ownHeadroomMinor := int64(0)
		readHeadroom := func(label string, expectedUsed, expectedRemaining int64) error {
			data, readErr := os.ReadFile(filepath.Join(coord, label+".log"))
			if readErr != nil {
				return readErr
			}
			var response struct {
				Headroom struct {
					BudgetID            string `json:"budget_id"`
					TotalCapMinor       string `json:"total_cap_minor"`
					TotalRemainingMinor string `json:"total_remaining_minor"`
				} `json:"headroom"`
			}
			if err := json.Unmarshal(data, &response); err != nil {
				return fmt.Errorf("decode run-owned headroom response: %w", err)
			}
			capMinor, capErr := strconv.ParseInt(response.Headroom.TotalCapMinor, 10, 64)
			remainingMinor, remainingErr := strconv.ParseInt(response.Headroom.TotalRemainingMinor, 10, 64)
			usedMinor := capMinor - remainingMinor
			if response.Headroom.BudgetID != config.BudgetID || capErr != nil || remainingErr != nil || capMinor != 1000 || usedMinor != expectedUsed || remainingMinor != expectedRemaining {
				return fmt.Errorf("run-owned headroom %s did not match authorized spend ledger: %+v", label, response.Headroom)
			}
			if label == "budget-headroom-after-in-cap" {
				ownHeadroomMinor = remainingMinor
			}
			return nil
		}
		if err := readHeadroom("budget-headroom-before", 0, 1000); err != nil {
			return nil, err
		}
		if _, err := runCLI("authorization-propose", "--api-base", config.APIBaseURL, "authorization", "propose",
			"--id", config.InCapAuthorization, "--idempotency-key", config.InCapIdempotency,
			"--mandate-id", config.MandateID, "--amount-minor", strconv.FormatInt(config.InCapAmountMinor, 10),
			"--currency", "USD", "--counterparty", config.Counterparty,
			"--settlement-id", config.InCapSettlement, "--instrument-id", config.InCapInstrumentID,
			"--settlement-idempotency-key", config.InCapIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
		}
		if err := write("authorization-ready.json", map[string]any{"authorization_id": config.InCapAuthorization, "cli_command": "authorization propose"}); err != nil {
			return nil, err
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "settle-release"), nil); err != nil {
			return nil, err
		}
		// The harness writes the ordinary-run config before the run starts, then
		// adds the Agent Manager operation-grant ID after approving the exact
		// spend intent. Reload only after the release handshake so this runner
		// observes the grant authorized for this spend.
		if err := waitForFile(execCtx, configPath, &config); err != nil {
			return nil, err
		}
		if config.InCapGrantID == "" {
			return nil, errors.New("in-cap spend is missing its distinct Agent Manager operation grant")
		}
		if err := write("settle-release-observed.json", map[string]any{"observed": true}); err != nil {
			return nil, err
		}
		if _, err := runCLI("settlement-report-altered-instrument", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.InCapAuthorization, "--settlement-id", config.InCapSettlement,
			"--instrument-id", config.InCapInstrumentID+"-altered", "--idempotency-key", config.InCapIdempotency+"-settlement", "--json"); err == nil {
			return nil, errors.New("Treasury CLI accepted a settlement instrument that was not committed by the approved intent")
		}
		if err := write("altered-instrument-command-completed.json", map[string]any{"completed": true}); err != nil {
			return nil, err
		}
		if err := write("altered-instrument-denied.json", map[string]any{
			"authorization_id": config.InCapAuthorization, "expected_instrument_id": config.InCapInstrumentID,
			"attempted_instrument_id": config.InCapInstrumentID + "-altered", "signature_created": false, "dispatch": "none",
		}); err != nil {
			return nil, err
		}
		inCapSettlementOutput, err := runCLI("settlement-report-in-cap", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.InCapAuthorization, "--settlement-id", config.InCapSettlement,
			"--instrument-id", config.InCapInstrumentID, "--idempotency-key", config.InCapIdempotency+"-settlement", "--json")
		if err != nil {
			return nil, err
		}
		var inCapSettlement struct {
			Settlement struct {
				ID          string `json:"id"`
				Outcome     string `json:"outcome"`
				ExternalID  string `json:"external_id"`
				AmountMinor string `json:"amount_minor"`
			} `json:"settlement"`
		}
		if err := json.Unmarshal(inCapSettlementOutput, &inCapSettlement); err != nil {
			return nil, fmt.Errorf("decode ordinary-run in-cap settlement: %w", err)
		}
		amountMinor, amountErr := strconv.ParseInt(inCapSettlement.Settlement.AmountMinor, 10, 64)
		if amountErr != nil || inCapSettlement.Settlement.ID != config.InCapSettlement || inCapSettlement.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || amountMinor != config.InCapAmountMinor || !cf1HexIdentity(inCapSettlement.Settlement.ExternalID, 32) {
			return nil, fmt.Errorf("ordinary-run in-cap spend lacks a settled chain transaction identity: settlement=%+v amount_decode=%v", inCapSettlement.Settlement, amountErr)
		}
		if err := write("in-cap-settled.json", map[string]any{"authorization_id": config.InCapAuthorization, "settlement_id": config.InCapSettlement}); err != nil {
			return nil, err
		}
		var rotationReceipt struct {
			Address       string `json:"address"`
			NewGeneration string `json:"new_generation"`
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "mandate-session-rotated.json"), &rotationReceipt); err != nil {
			return nil, err
		}
		// A stable request ID makes the Agent Manager broker rotation safe to
		// retry if the CLI read or Treasury rebind response is lost.
		const rotationRequestID = "cf1-cross-budget-rotation-v1"
		setEnv("VROOLI_TREASURY_SESSION_ROTATION_REQUEST", rotationRequestID)
		rotatedWalletOutput, err := runCLI("agent-wallet-after-rotation", "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", config.BudgetID, "--mandate-id", config.MandateID, "--json")
		if err != nil {
			return nil, err
		}
		var rotatedWallet struct {
			Wallet struct {
				Address           string `json:"address"`
				SessionGeneration string `json:"session_generation"`
			} `json:"wallet"`
		}
		if err := json.Unmarshal(rotatedWalletOutput, &rotatedWallet); err != nil || rotatedWallet.Wallet.Address == "" || rotatedWallet.Wallet.SessionGeneration == "" {
			return nil, fmt.Errorf("decode mandate wallet after in-run rotation: %w", err)
		}
		if rotatedWallet.Wallet.Address != rotationReceipt.Address || rotatedWallet.Wallet.SessionGeneration != rotationReceipt.NewGeneration {
			return nil, fmt.Errorf("ordinary run did not rebind its key to the rotated mandate generation: wallet=%+v receipt=%+v", rotatedWallet.Wallet, rotationReceipt)
		}
		// The key is captured from the first process-specific LaunchRequest in
		// memory only; it is no longer present in the runner's pre-launch map.
		oldKey := initialSessionKey
		newKey, err := keyStore.ResolveOrMint(execCtx, request.RunID)
		if err != nil {
			return nil, fmt.Errorf("read current run-owned key after broker rotation: %w", err)
		}
		oldPublicKeyByFixture := map[string]string{
			fmt.Sprintf("%064x", 1): "0x0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8",
			fmt.Sprintf("%064x", 3): "0x04f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9388f7b0f632de8140fe337e62a37f3566500a99934c2231b6cb9fd7584b8e672",
		}
		oldPublicKey := oldPublicKeyByFixture[oldKey]
		if oldPublicKey == "" {
			return nil, errors.New("initial run key is not one of the deterministic stale-key fixtures")
		}
		staleKeyStatus, err := cf1LiveStaleSessionKeyProbe(execCtx, config.APIBaseURL, request.Environment[cliutil.EnvIdentityToken], config.BudgetID, config.MandateID, oldPublicKey)
		if err != nil {
			return nil, err
		}
		oldKeyDigest, newKeyDigest := sha256.Sum256([]byte(oldKey)), sha256.Sum256([]byte(newKey))
		if oldKey == "" || newKey == "" || oldKey == newKey {
			return nil, errors.New("Agent Manager rotation did not replace the run-owned session key")
		}
		if err := write("ordinary-run-key-rotation.json", map[string]any{
			"run_id": request.RunID.String(), "rotation_request_id": rotationRequestID,
			"old_key_refused_for_new_generation": true, "old_key_sha256": fmt.Sprintf("%x", oldKeyDigest),
			"new_key_sha256": fmt.Sprintf("%x", newKeyDigest), "new_public_key_rebound": true,
			"old_key_refusal_status": staleKeyStatus, "live_stale_key_probe": true,
			"mandate_generation": rotationReceipt.NewGeneration,
		}); err != nil {
			return nil, err
		}
		if _, err := runCLI("budget-headroom-after-in-cap", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
		if err := readHeadroom("budget-headroom-after-in-cap", config.InCapAmountMinor, 1000-config.InCapAmountMinor); err != nil {
			return nil, err
		}
		crossBudgetEvidencePath := strings.TrimSpace(os.Getenv(treasuryCF1CrossBudgetEvidencePathEnv))
		if crossBudgetEvidencePath == "" {
			crossBudgetEvidencePath = filepath.Join(coord, "agent-manager-cross-budget-identities.json")
		}
		agentIdentity := cf1AgentCrossBudgetIdentity{
			RunID: request.RunID.String(), ProfileID: request.Profile.ID.String(), BudgetID: config.BudgetID,
			MandateID: config.MandateID, WalletID: agentWallet.WalletID, WalletAddress: agentWallet.Address,
			OwnAuthorizationID: config.InCapAuthorization, OwnSettlementID: config.InCapSettlement,
			OwnTxHash: inCapSettlement.Settlement.ExternalID, OwnHeadroomMinor: ownHeadroomMinor,
			ForeignWalletProbeStatus: foreignWalletProbeStatus, ForeignMandateProbeStatus: foreignMandateProbeStatus,
		}
		if err := upsertCF1CrossBudgetIdentityEvidence(crossBudgetEvidencePath, agentIdentity); err != nil {
			return nil, fmt.Errorf("record ordinary-run cross-budget identity evidence: %w", err)
		}
		if config.PairedProbeBarrierDir != "" {
			identityMarker := cf1PairIdentityReady{
				RunID: request.RunID.String(), CounterpartRunID: config.CounterpartIdentity.RunID,
				BudgetID: agentIdentity.BudgetID, MandateID: agentIdentity.MandateID,
				WalletID: agentIdentity.WalletID, WalletAddress: agentIdentity.WalletAddress,
			}
			if err := writeCF1PairBarrierMarker(config.PairedProbeBarrierDir, "identity-ready", request.RunID.String(), identityMarker); err != nil {
				return nil, fmt.Errorf("record paired identity readiness: %w", err)
			}
			var identityRelease cf1PairIdentityRelease
			if err := waitForFile(execCtx, cf1PairBarrierPath(config.PairedProbeBarrierDir, "identity-release", request.RunID.String()), &identityRelease); err != nil {
				return nil, fmt.Errorf("wait for Treasury paired identity verification: %w", err)
			}
			if err := validateCF1PairIdentityRelease(identityRelease, request.RunID.String(), config.CounterpartIdentity.RunID); err != nil {
				return nil, err
			}
			if err := validateCF1PairedIdentityEvidence(crossBudgetEvidencePath, agentIdentity, *config.CounterpartIdentity); err != nil {
				return nil, fmt.Errorf("validate two-run identity artifact before paired release: %w", err)
			}
		}
		overCapTarget, overCapSettlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: config.OverCapAuthorization, MandateID: config.MandateID, AmountMinor: config.OverCapAmountMinor,
			Currency: "USD", Counterparty: config.Counterparty, AuthorizationIdempotencyKey: config.OverCapIdempotency,
			SettlementID: config.OverCapSettlement, InstrumentID: config.OverCapInstrumentID,
			SettlementIdempotencyKey: config.OverCapIdempotency + "-settlement",
		}).Targets()
		if err := write("over-cap-operation-grant-requested.json", map[string]any{
			"authorization_id": config.OverCapAuthorization, "amount_minor": config.OverCapAmountMinor,
			"required_target": overCapTarget, "required_settlement_target": overCapSettlementTarget,
		}); err != nil {
			return nil, err
		}
		var overCapGrantReceipt struct {
			GrantID      string   `json:"grant_id"`
			SpendTargets []string `json:"spend_targets"`
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "over-cap-operation-approved.json"), &overCapGrantReceipt); err != nil {
			return nil, err
		}
		if overCapGrantReceipt.GrantID == "" || overCapGrantReceipt.GrantID == config.InCapGrantID || len(overCapGrantReceipt.SpendTargets) != 2 || overCapGrantReceipt.SpendTargets[0] != overCapTarget || overCapGrantReceipt.SpendTargets[1] != overCapSettlementTarget {
			return nil, errors.New("over-cap spend is missing its distinct Agent Manager operation grant")
		}
		if _, err := runCLI("authorization-propose-over-cap", "--api-base", config.APIBaseURL, "authorization", "propose",
			"--id", config.OverCapAuthorization, "--idempotency-key", config.OverCapIdempotency,
			"--mandate-id", config.MandateID, "--amount-minor", strconv.FormatInt(config.OverCapAmountMinor, 10),
			"--currency", "USD", "--counterparty", config.Counterparty,
			"--settlement-id", config.OverCapSettlement, "--instrument-id", config.OverCapInstrumentID,
			"--settlement-idempotency-key", config.OverCapIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
		}
		if err := write("over-cap-authorization-ready.json", map[string]any{"authorization_id": config.OverCapAuthorization, "cli_command": "authorization propose"}); err != nil {
			return nil, err
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "over-cap-settle-release"), nil); err != nil {
			return nil, err
		}
		overCapFirstOutput, err := runCLI("settlement-report-over-cap-first", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.OverCapAuthorization, "--settlement-id", config.OverCapSettlement,
			"--instrument-id", config.OverCapInstrumentID, "--idempotency-key", config.OverCapIdempotency+"-settlement", "--json")
		if err != nil {
			return nil, err
		}
		overCapFirst, err := parseCF1SettledRetryCharge(overCapFirstOutput, config.OverCapAuthorization, config.OverCapSettlement, config.OverCapInstrumentID, config.OverCapAmountMinor)
		if err != nil {
			return nil, fmt.Errorf("validate approved over-cap settlement before Treasury restart: %w", err)
		}
		if err := write("over-cap-settled.json", map[string]any{"authorization_id": config.OverCapAuthorization, "settlement_id": config.OverCapSettlement}); err != nil {
			return nil, err
		}
		if _, err := runCLI("budget-headroom-after-over-cap", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
		if err := readHeadroom("budget-headroom-after-over-cap", config.InCapAmountMinor+config.OverCapAmountMinor, 1000-config.InCapAmountMinor-config.OverCapAmountMinor); err != nil {
			return nil, err
		}
		if err := write("over-cap-headroom-before-restart.json", map[string]any{"authorization_id": config.OverCapAuthorization, "validated": true}); err != nil {
			return nil, err
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "over-cap-retry-release"), nil); err != nil {
			return nil, err
		}
		overCapRetryOutput, err := runCLI("settlement-report-over-cap-after-restart", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.OverCapAuthorization, "--settlement-id", config.OverCapSettlement,
			"--instrument-id", config.OverCapInstrumentID, "--idempotency-key", config.OverCapIdempotency+"-settlement", "--json")
		if err != nil {
			return nil, err
		}
		overCapRetry, err := parseCF1SettledRetryCharge(overCapRetryOutput, config.OverCapAuthorization, config.OverCapSettlement, config.OverCapInstrumentID, config.OverCapAmountMinor)
		if err != nil {
			return nil, fmt.Errorf("validate approved over-cap settlement retry after Treasury restart: %w", err)
		}
		receiptDigest, receiptBytes, err := compareCF1SettledRetryCharges(overCapFirst, overCapRetry)
		if err != nil {
			return nil, fmt.Errorf("approved over-cap retry changed transaction, exact amount, or raw receipt across Treasury restart: %w", err)
		}
		if err := write("over-cap-retry-after-restart.json", map[string]any{"authorization_id": config.OverCapAuthorization, "settlement_id": config.OverCapSettlement}); err != nil {
			return nil, err
		}
		if _, err := runCLI("budget-headroom-after-over-cap-retry", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
		if err := readHeadroom("budget-headroom-after-over-cap-retry", config.InCapAmountMinor+config.OverCapAmountMinor, 1000-config.InCapAmountMinor-config.OverCapAmountMinor); err != nil {
			return nil, err
		}
		if err := write("over-cap-restart-retry-continuity.json", map[string]any{
			"run_id": request.RunID.String(), "authorization_id": config.OverCapAuthorization,
			"settlement_id": config.OverCapSettlement, "instrument_id": config.OverCapInstrumentID,
			"status_before_restart": overCapFirst.Outcome, "status_after_restart": overCapRetry.Outcome,
			"transaction_hash_before_restart": overCapFirst.ExternalID, "transaction_hash_after_restart": overCapRetry.ExternalID,
			"same_transaction_hash": true, "amount_minor_before_restart": overCapFirst.AmountMinor,
			"amount_minor_after_restart": overCapRetry.AmountMinor, "currency": overCapRetry.Currency,
			"receipt_reference_bytes": receiptBytes, "receipt_reference_sha256": receiptDigest,
			"same_receipt_reference_bytes": true, "ledger_debit_count_before_retry": 1,
			"ledger_debit_count_after_retry": 1, "headroom_minor_before_retry": 1000 - config.InCapAmountMinor - config.OverCapAmountMinor,
			"headroom_minor_after_retry": 1000 - config.InCapAmountMinor - config.OverCapAmountMinor,
		}); err != nil {
			return nil, fmt.Errorf("write approved over-cap retry continuity evidence: %w", err)
		}
		var x402GrantID string
		var x402MeasuredGrantID string
		if config.X402CatalogURL != "" {
			if config.X402PriceID != "cf4-research-inference" || config.X402CallID != "cf4-research-inference-call" || config.X402Counterparty == "" || config.X402InstrumentID == "" || config.X402AmountMinor != 1 || config.X402AuthorizationID == "" || config.X402SettlementID == "" || config.X402IdempotencyKey == "" {
				return nil, errors.New("CF4 x402 ordinary-run config is incomplete or does not use the one-minor-unit per-call price")
			}
			catalogURL, parseErr := url.ParseRequestURI(config.X402CatalogURL)
			if parseErr != nil || !strings.EqualFold(catalogURL.Hostname(), config.X402Counterparty) {
				return nil, errors.New("CF4 x402 counterparty must match the priced Offer Desk URL hostname")
			}
			x402SettlementIdempotency := config.X402IdempotencyKey + "-settlement"
			x402ProposalTarget, x402SettlementTarget := (cliutil.TreasurySpendIntent{
				AuthorizationID: config.X402AuthorizationID, MandateID: config.MandateID, AmountMinor: config.X402AmountMinor, Currency: "USD",
				Counterparty: config.X402Counterparty, AuthorizationIdempotencyKey: config.X402IdempotencyKey,
				SettlementID: config.X402SettlementID, InstrumentID: config.X402InstrumentID,
				SettlementIdempotencyKey: x402SettlementIdempotency,
			}).Targets()
			if err := write("x402-operation-grant-requested.json", map[string]any{
				"run_id": request.RunID.String(), "authorization_id": config.X402AuthorizationID, "settlement_id": config.X402SettlementID,
				"amount_minor": config.X402AmountMinor, "price_id": config.X402PriceID, "call_id": config.X402CallID,
				"catalog_url": config.X402CatalogURL, "instrument_id": config.X402InstrumentID,
				"required_target": x402ProposalTarget, "required_settlement_target": x402SettlementTarget,
			}); err != nil {
				return nil, err
			}
			var x402GrantReceipt struct {
				GrantID      string   `json:"grant_id"`
				SpendTargets []string `json:"spend_targets"`
			}
			if err := waitForFile(execCtx, filepath.Join(coord, "x402-operation-approved.json"), &x402GrantReceipt); err != nil {
				return nil, err
			}
			if x402GrantReceipt.GrantID == "" || x402GrantReceipt.GrantID == config.InCapGrantID || x402GrantReceipt.GrantID == overCapGrantReceipt.GrantID || len(x402GrantReceipt.SpendTargets) != 2 || x402GrantReceipt.SpendTargets[0] != x402ProposalTarget || x402GrantReceipt.SpendTargets[1] != x402SettlementTarget {
				return nil, errors.New("CF4 priced call is missing its own exact Agent Manager operation grant")
			}
			x402GrantID = x402GrantReceipt.GrantID
			if _, err := runCLI("authorization-propose-x402", "--api-base", config.APIBaseURL, "authorization", "propose",
				"--id", config.X402AuthorizationID, "--idempotency-key", config.X402IdempotencyKey,
				"--mandate-id", config.MandateID, "--amount-minor", strconv.FormatInt(config.X402AmountMinor, 10),
				"--currency", "USD", "--counterparty", config.X402Counterparty,
				"--settlement-id", config.X402SettlementID, "--instrument-id", config.X402InstrumentID,
				"--settlement-idempotency-key", x402SettlementIdempotency, "--json"); err != nil {
				return nil, err
			}
			if err := write("x402-authorization-ready.json", map[string]any{"authorization_id": config.X402AuthorizationID, "grant_id": x402GrantID, "price_id": config.X402PriceID, "call_id": config.X402CallID}); err != nil {
				return nil, err
			}
			if err := waitForFile(execCtx, filepath.Join(coord, "x402-report-release"), nil); err != nil {
				return nil, err
			}
			if _, err := runCLI("settlement-report-x402-first", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
				"--authorization-id", config.X402AuthorizationID, "--settlement-id", config.X402SettlementID,
				"--instrument-id", config.X402InstrumentID, "--idempotency-key", x402SettlementIdempotency, "--json"); err != nil {
				return nil, fmt.Errorf("Treasury did not retain the ambiguous CF4 x402 outcome: %w", err)
			}
			firstOutcomeData, err := os.ReadFile(filepath.Join(coord, "settlement-report-x402-first.log"))
			if err != nil {
				return nil, err
			}
			var firstOutcome struct {
				Settlement struct {
					Outcome          string `json:"outcome"`
					ExternalID       string `json:"external_id"`
					ReceiptReference string `json:"receipt_reference"`
				} `json:"settlement"`
			}
			if err := json.Unmarshal(firstOutcomeData, &firstOutcome); err != nil || firstOutcome.Settlement.Outcome != "CHARGE_OUTCOME_UNKNOWN" || firstOutcome.Settlement.ReceiptReference == "" {
				return nil, fmt.Errorf("first CF4 x402 outcome was not durably reported unknown (outcome=%q, decode=%v)", firstOutcome.Settlement.Outcome, err)
			}
			receiptDigestHex, receiptLength, err := hashCF1X402ReceiptReference(firstOutcome.Settlement.ReceiptReference)
			if err != nil {
				return nil, err
			}
			const receiptArtifact = "x402-raw-receipt-reference.bin"
			receiptFile, err := os.OpenFile(filepath.Join(coord, receiptArtifact), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
			if err != nil {
				return nil, fmt.Errorf("create private raw x402 receipt evidence: %w", err)
			}
			if _, err := receiptFile.Write([]byte(firstOutcome.Settlement.ReceiptReference)); err != nil {
				_ = receiptFile.Close()
				return nil, fmt.Errorf("write raw x402 receipt evidence: %w", err)
			}
			if err := receiptFile.Sync(); err != nil {
				_ = receiptFile.Close()
				return nil, fmt.Errorf("sync raw x402 receipt evidence: %w", err)
			}
			if err := receiptFile.Close(); err != nil {
				return nil, fmt.Errorf("close raw x402 receipt evidence: %w", err)
			}
			if err := write("x402-outcome-unknown.json", map[string]any{
				"authorization_id": config.X402AuthorizationID, "settlement_id": config.X402SettlementID,
				"operation_grant_id": x402GrantID, "retry_idempotency_key": x402SettlementIdempotency,
				"catalog_url": config.X402CatalogURL, "price_id": config.X402PriceID, "call_id": config.X402CallID,
				"retained_external_id":       firstOutcome.Settlement.ExternalID,
				"receipt_reference_artifact": receiptArtifact, "receipt_reference_bytes": receiptLength,
				"receipt_reference_sha256": receiptDigestHex,
			}); err != nil {
				return nil, err
			}
			if err := waitForFile(execCtx, filepath.Join(coord, "x402-retry-release"), nil); err != nil {
				return nil, err
			}
			if _, err := runCLI("settlement-report-x402-after-restart", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
				"--authorization-id", config.X402AuthorizationID, "--settlement-id", config.X402SettlementID,
				"--instrument-id", config.X402InstrumentID, "--idempotency-key", x402SettlementIdempotency, "--json"); err != nil {
				return nil, fmt.Errorf("exact CF4 x402 retry after Treasury restart did not settle: %w", err)
			}
			retryOutcomeData, err := os.ReadFile(filepath.Join(coord, "settlement-report-x402-after-restart.log"))
			if err != nil {
				return nil, err
			}
			var retryOutcome struct {
				Settlement struct {
					Outcome          string `json:"outcome"`
					ExternalID       string `json:"external_id"`
					ReceiptReference string `json:"receipt_reference"`
				} `json:"settlement"`
			}
			if err := json.Unmarshal(retryOutcomeData, &retryOutcome); err != nil || retryOutcome.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || strings.TrimSpace(retryOutcome.Settlement.ExternalID) == "" {
				return nil, fmt.Errorf("exact CF4 x402 retry did not return a settled transaction hash (outcome=%q, external_id=%q, decode=%v)", retryOutcome.Settlement.Outcome, retryOutcome.Settlement.ExternalID, err)
			}
			retryReceiptDigestHex, retryReceiptLength, err := compareCF1X402ReceiptReferences(firstOutcome.Settlement.ReceiptReference, retryOutcome.Settlement.ReceiptReference)
			if err != nil || retryReceiptDigestHex != receiptDigestHex || retryReceiptLength != receiptLength {
				return nil, fmt.Errorf("exact CF4 x402 retry changed the raw receipt-reference bytes after Treasury restart: %w", err)
			}
			if err := write("x402-settled-after-restart.json", map[string]any{
				"authorization_id": config.X402AuthorizationID, "settlement_id": config.X402SettlementID,
				"operation_grant_id": x402GrantID, "same_authorization": true,
				"same_settlement_id": true, "same_idempotency_key": true,
				"transaction_hash":             retryOutcome.Settlement.ExternalID,
				"same_receipt_reference_bytes": true, "receipt_reference_artifact": receiptArtifact,
				"receipt_reference_bytes": retryReceiptLength, "receipt_reference_sha256": retryReceiptDigestHex,
			}); err != nil {
				return nil, err
			}
			if _, err := runCLI("budget-headroom-after-x402", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
				return nil, err
			}
			expectedX402Used := config.InCapAmountMinor + config.OverCapAmountMinor + config.X402AmountMinor
			if err := readHeadroom("budget-headroom-after-x402", expectedX402Used, 1000-expectedX402Used); err != nil {
				return nil, err
			}
			if _, err := runCLI("budget-headroom-after-x402-retry", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
				return nil, err
			}
			if err := readHeadroom("budget-headroom-after-x402-retry", expectedX402Used, 1000-expectedX402Used); err != nil {
				return nil, err
			}
			if treasuryCF1MeasuredX402Enabled() {
				if err := executeCF1MeasuredX402Call(execCtx, request, config, coord, runCLI, write); err != nil {
					return nil, fmt.Errorf("settle the separate measured CF1 x402 call: %w", err)
				}
				var measuredApproval struct {
					GrantID string `json:"grant_id"`
				}
				if err := waitForFile(execCtx, filepath.Join(coord, "x402-measured-operation-approved.json"), &measuredApproval); err != nil || measuredApproval.GrantID == "" {
					return nil, fmt.Errorf("read the exact grant used by the measured CF1 x402 call: grant=%q error=%v", measuredApproval.GrantID, err)
				}
				x402MeasuredGrantID = measuredApproval.GrantID
				if _, err := runCLI("budget-headroom-after-x402-measured", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
					return nil, err
				}
				expectedMeasuredUsed := expectedX402Used + 1
				if err := readHeadroom("budget-headroom-after-x402-measured", expectedMeasuredUsed, 1000-expectedMeasuredUsed); err != nil {
					return nil, err
				}
			}
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "freeze-next-auth-release"), nil); err != nil {
			return nil, err
		}
		frozenAuthorizationID := config.InCapAuthorization + "-frozen"
		frozenTarget, frozenSettlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: frozenAuthorizationID, MandateID: config.MandateID, AmountMinor: 10, Currency: "USD",
			Counterparty: config.Counterparty, AuthorizationIdempotencyKey: config.InCapIdempotency + "-frozen",
			SettlementID: config.InCapSettlement + "-frozen", InstrumentID: config.InCapInstrumentID,
			SettlementIdempotencyKey: config.InCapIdempotency + "-frozen-settlement",
		}).Targets()
		if err := write("frozen-operation-grant-requested.json", map[string]any{
			"authorization_id": frozenAuthorizationID, "amount_minor": 10,
			"required_target": frozenTarget, "required_settlement_target": frozenSettlementTarget,
		}); err != nil {
			return nil, err
		}
		var frozenGrantReceipt struct {
			GrantID      string   `json:"grant_id"`
			SpendTargets []string `json:"spend_targets"`
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "frozen-operation-approved.json"), &frozenGrantReceipt); err != nil {
			return nil, err
		}
		if frozenGrantReceipt.GrantID == "" || frozenGrantReceipt.GrantID == config.InCapGrantID || frozenGrantReceipt.GrantID == overCapGrantReceipt.GrantID || len(frozenGrantReceipt.SpendTargets) != 2 || frozenGrantReceipt.SpendTargets[0] != frozenTarget || frozenGrantReceipt.SpendTargets[1] != frozenSettlementTarget {
			return nil, errors.New("frozen authorization is missing its own exact Agent Manager operation grant")
		}
		if _, err := runCLI("authorization-propose-frozen", "--api-base", config.APIBaseURL, "authorization", "propose",
			"--id", config.InCapAuthorization+"-frozen", "--idempotency-key", config.InCapIdempotency+"-frozen",
			"--mandate-id", config.MandateID, "--amount-minor", "10",
			"--currency", "USD", "--counterparty", config.Counterparty,
			"--settlement-id", config.InCapSettlement+"-frozen", "--instrument-id", config.InCapInstrumentID,
			"--settlement-idempotency-key", config.InCapIdempotency+"-frozen-settlement", "--json"); err != nil {
			return nil, err
		}
		if err := write("frozen-next-authorization.json", map[string]any{"authorization_id": config.InCapAuthorization + "-frozen"}); err != nil {
			return nil, err
		}
		expiryTarget, expirySettlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: config.ExpiryAuthorization, MandateID: config.ExpiryMandateID, AmountMinor: 10, Currency: "USD",
			Counterparty: config.Counterparty, AuthorizationIdempotencyKey: config.ExpiryIdempotency,
			SettlementID: config.ExpirySettlement, InstrumentID: config.ExpiryInstrumentID,
			SettlementIdempotencyKey: config.ExpiryIdempotency + "-settlement",
		}).Targets()
		if err := write("expired-operation-grant-requested.json", map[string]any{
			"authorization_id": config.ExpiryAuthorization, "amount_minor": 10,
			"required_target": expiryTarget, "required_settlement_target": expirySettlementTarget,
		}); err != nil {
			return nil, err
		}
		var expiryGrantReceipt struct {
			GrantID      string   `json:"grant_id"`
			SpendTargets []string `json:"spend_targets"`
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "expired-operation-approved.json"), &expiryGrantReceipt); err != nil {
			return nil, err
		}
		if expiryGrantReceipt.GrantID == "" || expiryGrantReceipt.GrantID == config.InCapGrantID || expiryGrantReceipt.GrantID == overCapGrantReceipt.GrantID || expiryGrantReceipt.GrantID == frozenGrantReceipt.GrantID || len(expiryGrantReceipt.SpendTargets) != 2 || expiryGrantReceipt.SpendTargets[0] != expiryTarget || expiryGrantReceipt.SpendTargets[1] != expirySettlementTarget {
			return nil, errors.New("expired-mandate authorization is missing its own exact Agent Manager operation grant")
		}
		var expiredMandate struct {
			MandateID string    `json:"mandate_id"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		if err := waitForFile(execCtx, filepath.Join(coord, "expired-mandate-created.json"), &expiredMandate); err != nil {
			return nil, err
		}
		if expiredMandate.MandateID != config.ExpiryMandateID || expiredMandate.ExpiresAt.IsZero() {
			return nil, errors.New("expired-mandate receipt did not bind the expected mandate and expiry")
		}
		if delay := time.Until(expiredMandate.ExpiresAt.Add(50 * time.Millisecond)); delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-execCtx.Done():
				timer.Stop()
				return nil, execCtx.Err()
			case <-timer.C:
			}
		}
		if _, err := runCLI("authorization-propose-expired", "--api-base", config.APIBaseURL, "authorization", "propose",
			"--id", config.ExpiryAuthorization, "--idempotency-key", config.ExpiryIdempotency,
			"--mandate-id", config.ExpiryMandateID, "--amount-minor", "10",
			"--currency", "USD", "--counterparty", config.Counterparty,
			"--settlement-id", config.ExpirySettlement, "--instrument-id", config.ExpiryInstrumentID,
			"--settlement-idempotency-key", config.ExpiryIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
		}
		if err := write("expired-mandate-authorization-ready.json", map[string]any{"authorization_id": config.ExpiryAuthorization, "mandate_id": config.ExpiryMandateID}); err != nil {
			return nil, err
		}
		if request.EventSink == nil {
			return nil, errors.New("Agent Manager run event sink is unavailable")
		}
		if err := request.EventSink.Emit(domain.NewMessageEvent(request.RunID, "assistant", "Settled the authorized local-fork test payment through Treasury CLI.")); err != nil {
			return nil, fmt.Errorf("record ordinary run completion message: %w", err)
		}
		cliCommands := []string{"budget agent-wallet", "budget headroom before and after settlements", "authorization propose in-cap", "settlement report-outcome in-cap", "budget agent-wallet after mandate rotation", "authorization propose over-cap", "settlement report-outcome over-cap", "same settlement retry after restart", "headroom unchanged after retry", "authorization propose while frozen", "authorization propose after mandate expiry"}
		if x402GrantID != "" {
			cliCommands = append(cliCommands, "Offer Desk x402 priced call under distinct per-spend grant", "same x402 settlement retry after Treasury restart", "headroom unchanged after x402 retry")
			if treasuryCF1MeasuredX402Enabled() {
				cliCommands = append(cliCommands, "separate no-drop Offer Desk x402 call with measured result")
			}
		}
		result := map[string]any{
			"run_id": request.RunID.String(), "profile_id": request.Profile.ID.String(), "task_id": request.Task.ID.String(),
			"runner": string(request.ResolvedConfig.RunnerType), "cli_commands": cliCommands,
			"session_private_key_injected": true, "identity_token_injected": true,
			"per_spend_operation_grant_ids": func() []string {
				grants := []string{config.InCapGrantID, overCapGrantReceipt.GrantID}
				if x402GrantID != "" {
					grants = append(grants, x402GrantID)
					if treasuryCF1MeasuredX402Enabled() {
						grants = append(grants, x402MeasuredGrantID)
					}
				}
				return append(grants, frozenGrantReceipt.GrantID, expiryGrantReceipt.GrantID)
			}(),
			"private_values_recorded": false,
		}
		if err := write("run-result.json", result); err != nil {
			return nil, err
		}
		return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "local Base Treasury CLI flow completed from an ordinary Agent Manager run"}}, nil
	}

	registry := runner.NewRegistry()
	if err := registry.Register(mockRunner); err != nil {
		t.Fatal(err)
	}
	service = orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: 4 * time.Minute, MaxConcurrentRuns: 1, RequireSandboxByDefault: false}),
		orchestration.WithEvents(events), orchestration.WithRunners(registry), orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency), orchestration.WithRunStateRoot(filepath.Join(coord, "run-state")),
		orchestration.WithIdentitySecret([]byte(cf1TreasuryRunSecret)),
		orchestration.WithTreasurySessionKeyStore(keys),
		orchestration.WithAuthorizations(database.NewAuthorizationRepository(db)),
		orchestration.WithPendingAuthorizations(database.NewPendingAuthorizationRepository(db)),
		orchestration.WithStandingGrants(database.NewStandingGrantRepository(db)),
		orchestration.WithSandbox(mocks.NewFakeSandboxProvider()),
		orchestration.WithOwnerIdentity(cf1TreasuryOperatorVerifier{}), newTestRolePolicyOption(t),
	)
	stopBroker, err := service.StartAuthorizationBroker()
	if err != nil {
		t.Fatalf("start local Agent Manager credential broker: %v", err)
	}
	t.Cleanup(func() { _ = stopBroker() })
	profileName, taskTitle, taskDescription := "cf1-local-treasury-cli", "CF1 local Treasury CLI", "exercise a run-owned budget session against the local fork"
	runPrompt := "Use only the run-scoped Treasury CLI session to settle one local-fork test payment."
	if cf2AgentManagerOnly {
		profileName, taskTitle, taskDescription = "cf2-lpbs-x402-order", "CF2 LPBS x402 service order", "buy the canonical company-market-scan service through LPBS from an ordinary Agent Manager run"
		runPrompt = "Place one test-mode order for Offer Desk company-market-scan v1 through the LPBS Agent Manager x402 door. Verify the delivered report before settlement, then reconcile the accepted seller receipt."
	}
	if cf7Only {
		profileName, taskTitle, taskDescription = "cf7-research-budget-x402", "CF7 local research budget x402 calls", "buy three priced local-fork research calls and hold a fourth over-cap quote for decision"
	}
	profile, err := service.CreateProfile(ctx, &domain.AgentProfile{
		ID: uuid.New(), Name: profileName, RoleRef: "code.default",
		DeclaredScopes: []string{"treasury:read", "treasury:write"},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	projectRoot := t.TempDir()
	task, err := service.CreateTask(ctx, &domain.Task{ID: uuid.New(), Title: taskTitle, Description: taskDescription, ScopePath: projectRoot, ProjectRoot: projectRoot, Status: domain.TaskStatusQueued})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	run, err := service.CreateRun(ctx, orchestration.CreateRunRequest{
		OwnerToken: cf1TreasuryOperator, ExpectedOwnerSubject: "cf1-local-operator", TaskID: task.ID, AgentProfileID: &profile.ID,
		Prompt: runPrompt, RequestedScopes: []string{"treasury:read", "treasury:write"},
	})
	if err != nil {
		t.Fatalf("create ordinary Treasury-scoped run: %v", err)
	}
	if err := waitForRunReceipt(ctx, service, run.ID, filepath.Join(coord, "run-id.json")); err != nil {
		t.Fatalf("ordinary run did not reach the runner receipt boundary: %v", err)
	}
	policy, _ := authn.InteractivePolicy("treasury-spending")
	readPolicy, _ := authn.InteractivePolicy("treasury-wallet-reading")
	ownerScopes := append([]string{"agent-manager:write"}, policy.Scopes...)
	ownerScopes = append(ownerScopes, readPolicy.Scopes...)
	owner := coreidentity.Principal{Kind: coreidentity.ActorHuman, Source: coreidentity.SourceCloudflareAccess, Verified: true, Subject: "cf1-local-operator", Scopes: ownerScopes, ExpiresAt: time.Now().Add(time.Hour)}
	ownerCtx := coreidentity.WithPrincipal(ctx, owner)
	if cf7Only {
		config, err := readCF1RunConfig(coord)
		if err != nil {
			t.Fatalf("wait for CF7 local-fork setup: %v", err)
		}
		coordinateCF7X402Run(ctx, db, service, ownerCtx, run, coord, config, t)
		return
	}
	if cf2AgentManagerOnly {
		finalRun := waitForCompletedRun(t, ctx, service, run.ID, 3*time.Minute)
		if err := waitForCF1SessionRevocation(ctx, keys, run.ID); err != nil {
			t.Fatal(err)
		}
		var result struct {
			RunID             string `json:"run_id"`
			AgentManagerRunID string `json:"agent_manager_run_id"`
			OrderID           string `json:"order_id"`
			JobID             string `json:"job_id"`
			AuthorizationID   string `json:"authorization_id"`
			CaptureID         string `json:"capture_id"`
			PostingID         string `json:"posting_id"`
			ReceiptID         string `json:"receipt_id"`
			State             string `json:"state"`
		}
		if err := waitForFile(ctx, filepath.Join(coord, "agent-manager-lpbs-result.json"), &result); err != nil {
			t.Fatalf("read Agent Manager LPBS order evidence: %v", err)
		}
		if result.RunID != run.ID.String() || result.AgentManagerRunID != run.ID.String() || result.OrderID == "" || result.JobID == "" || result.AuthorizationID == "" || result.CaptureID == "" || result.PostingID == "" || result.ReceiptID == "" || result.State != "reconciled" {
			t.Fatalf("incomplete durable Agent Manager-to-LPBS-to-Ledger join: %+v", result)
		}
		if err := writeJSON(filepath.Join(coord, "agent-manager-lpbs-run-join.json"), map[string]any{
			"run_id": run.ID.String(), "agent_manager_run_id": result.AgentManagerRunID, "run_status": finalRun.Status, "order_id": result.OrderID,
			"job_id": result.JobID, "authorization_id": result.AuthorizationID, "capture_id": result.CaptureID,
			"posting_id": result.PostingID, "receipt_id": result.ReceiptID, "order_state": result.State,
			"terminal_session_revoked": keys.WasRevoked(run.ID), "private_values_recorded": false,
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	config, err := readCF1RunConfig(coord)
	if err != nil {
		t.Fatalf("wait for Treasury local-fork setup: %v", err)
	}
	budgetTarget := "treasury:budget/" + os.Getenv("TREASURY_CF1_BUDGET_ID")
	mandateID := strings.TrimSpace(os.Getenv("TREASURY_CF1_MANDATE_ID"))
	counterparty := strings.TrimSpace(os.Getenv("TREASURY_CF1_COUNTERPARTY"))
	inCapAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_IN_CAP_AUTHORIZATION_ID"))
	overCapAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_OVER_CAP_AUTHORIZATION_ID"))
	isolationBudgetID := strings.TrimSpace(os.Getenv("TREASURY_CF1_ISOLATION_BUDGET_ID"))
	readTargets := []string{budgetTarget, "treasury:authorization/" + inCapAuthorizationID}
	if isolationBudgetID != "" {
		readTargets = append(readTargets, "treasury:budget/"+isolationBudgetID)
	}
	if foreignBudgetID := strings.TrimSpace(config.ForeignBudgetID); foreignBudgetID != "" && foreignBudgetID != isolationBudgetID {
		readTargets = append(readTargets, "treasury:budget/"+foreignBudgetID)
	}
	inCapTarget, inCapSettlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: inCapAuthorizationID, MandateID: mandateID, AmountMinor: 30, Currency: "USD", Counterparty: counterparty,
		AuthorizationIdempotencyKey: os.Getenv("TREASURY_CF1_IN_CAP_IDEMPOTENCY_KEY"),
		SettlementID:                os.Getenv("TREASURY_CF1_IN_CAP_SETTLEMENT_ID"), InstrumentID: os.Getenv("TREASURY_CF1_IN_CAP_INSTRUMENT_ID"),
		SettlementIdempotencyKey: os.Getenv("TREASURY_CF1_IN_CAP_IDEMPOTENCY_KEY") + "-settlement",
	}).Targets()
	overCapTarget, overCapSettlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: overCapAuthorizationID, MandateID: mandateID, AmountMinor: 110, Currency: "USD", Counterparty: counterparty,
		AuthorizationIdempotencyKey: os.Getenv("TREASURY_CF1_OVER_CAP_IDEMPOTENCY_KEY"),
		SettlementID:                os.Getenv("TREASURY_CF1_OVER_CAP_SETTLEMENT_ID"), InstrumentID: os.Getenv("TREASURY_CF1_OVER_CAP_INSTRUMENT_ID"),
		SettlementIdempotencyKey: os.Getenv("TREASURY_CF1_OVER_CAP_IDEMPOTENCY_KEY") + "-settlement",
	}).Targets()
	grant, err := service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
		RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf1-local-fork", HarnessSession: "commissioned-local-test",
		Rules: []domain.AuthorizationRule{
			{Policy: "treasury-spending", Targets: []string{inCapTarget, inCapSettlementTarget}},
			{Policy: "treasury-wallet-reading", Targets: readTargets},
		}, DurationSeconds: 3600,
	})
	if err != nil {
		t.Fatalf("request mandate-scoped Treasury capability: %v", err)
	}
	approved, err := service.ApproveAuthorization(ownerCtx, grant.ID)
	if err != nil {
		t.Fatalf("approve local Treasury capability: %v", err)
	}
	_ = approved // Raw operation tokens are intentionally never written to evidence.
	if err := writeJSON(filepath.Join(coord, "am-operation-approved.json"), map[string]any{
		"run_id": run.ID.String(), "grant_id": grant.ID.String(), "spend": "in-cap", "policy": "treasury-spending",
		"spend_targets": []string{inCapTarget, inCapSettlementTarget}, "read_targets": readTargets, "approved_at": approved.ApprovedAt,
	}); err != nil {
		t.Fatal(err)
	}
	config.InCapGrantID = grant.ID.String()
	configPath := filepath.Join(coord, "treasury-config.json")
	if err := writeJSON(configPath, config); err != nil {
		t.Fatal(err)
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "authorization-ready.json"), nil); err != nil {
		t.Fatalf("Treasury CLI did not propose spend: %v", err)
	}
	if err := os.WriteFile(filepath.Join(coord, "settle-release"), []byte("approved by local Treasury fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	var overCapGrantRequest struct {
		AuthorizationID          string `json:"authorization_id"`
		AmountMinor              int64  `json:"amount_minor"`
		RequiredTarget           string `json:"required_target"`
		RequiredSettlementTarget string `json:"required_settlement_target"`
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "over-cap-operation-grant-requested.json"), &overCapGrantRequest); err != nil {
		t.Fatalf("Treasury CLI did not request its second exact intent grant: %v", err)
	}
	if overCapGrantRequest.AuthorizationID != overCapAuthorizationID || overCapGrantRequest.AmountMinor != 110 || overCapGrantRequest.RequiredTarget != overCapTarget || overCapGrantRequest.RequiredSettlementTarget != overCapSettlementTarget {
		t.Fatalf("over-cap operation grant request did not bind the expected immutable intent: %+v", overCapGrantRequest)
	}
	secondSpendGrant, err := service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
		RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf1-local-fork", HarnessSession: "commissioned-local-test",
		Rules: []domain.AuthorizationRule{
			{Policy: "treasury-spending", Targets: []string{overCapTarget, overCapSettlementTarget}},
			{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + overCapAuthorizationID}},
		}, DurationSeconds: 3600,
		BigPicture:       "Authorize this run to request its over-cap local Treasury test spend.",
		TechnicalDetails: "One spend only: 110 USD minor units, separately approved by Treasury, on the isolated Base fork; run " + run.ID.String(),
	})
	if err != nil {
		t.Fatalf("request distinct over-cap operation grant: %v", err)
	}
	secondSpendApproval, err := service.ApproveAuthorization(ownerCtx, secondSpendGrant.ID)
	if err != nil {
		t.Fatalf("approve distinct over-cap operation grant: %v", err)
	}
	if secondSpendGrant.ID == grant.ID {
		t.Fatal("both spends received the same Agent Manager operation grant")
	}
	if err := writeJSON(filepath.Join(coord, "over-cap-operation-approved.json"), map[string]any{
		"run_id": run.ID.String(), "grant_id": secondSpendGrant.ID.String(), "spend": "over-cap", "policy": "treasury-spending",
		"spend_targets": []string{overCapTarget, overCapSettlementTarget}, "read_targets": []string{budgetTarget, "treasury:authorization/" + overCapAuthorizationID}, "approved_at": secondSpendApproval.ApprovedAt,
	}); err != nil {
		t.Fatal(err)
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "over-cap-authorization-ready.json"), nil); err != nil {
		t.Fatalf("Treasury CLI did not propose its separately granted spend: %v", err)
	}
	var x402Grant *domain.AuthorizationGrant
	var x402MeasuredGrant *domain.AuthorizationGrant
	if config.X402CatalogURL != "" {
		var x402GrantRequest struct {
			RunID                    string `json:"run_id"`
			AuthorizationID          string `json:"authorization_id"`
			SettlementID             string `json:"settlement_id"`
			AmountMinor              int64  `json:"amount_minor"`
			PriceID                  string `json:"price_id"`
			CallID                   string `json:"call_id"`
			CatalogURL               string `json:"catalog_url"`
			InstrumentID             string `json:"instrument_id"`
			RequiredTarget           string `json:"required_target"`
			RequiredSettlementTarget string `json:"required_settlement_target"`
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-operation-grant-requested.json"), &x402GrantRequest); err != nil {
			t.Fatalf("ordinary run did not request its per-call CF4 x402 grant: %v", err)
		}
		x402Target, x402SettlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: config.X402AuthorizationID, MandateID: mandateID, AmountMinor: config.X402AmountMinor, Currency: "USD",
			Counterparty: config.X402Counterparty, AuthorizationIdempotencyKey: config.X402IdempotencyKey,
			SettlementID: config.X402SettlementID, InstrumentID: config.X402InstrumentID,
			SettlementIdempotencyKey: config.X402IdempotencyKey + "-settlement",
		}).Targets()
		if config.X402AmountMinor != 1 || x402GrantRequest.RunID != run.ID.String() || x402GrantRequest.AuthorizationID != config.X402AuthorizationID || x402GrantRequest.SettlementID != config.X402SettlementID || x402GrantRequest.AmountMinor != config.X402AmountMinor || x402GrantRequest.PriceID != config.X402PriceID || x402GrantRequest.CallID != config.X402CallID || x402GrantRequest.CatalogURL != config.X402CatalogURL || x402GrantRequest.InstrumentID != config.X402InstrumentID || x402GrantRequest.RequiredTarget != x402Target || x402GrantRequest.RequiredSettlementTarget != x402SettlementTarget {
			t.Fatalf("CF4 operation grant request was not bound to the run's one-call Offer Desk intent: %+v", x402GrantRequest)
		}
		x402Grant, err = service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
			RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf4-local-fork", HarnessSession: "commissioned-local-test",
			Rules: []domain.AuthorizationRule{
				{Policy: "treasury-spending", Targets: []string{x402Target, x402SettlementTarget}},
				{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + config.X402AuthorizationID}},
			}, DurationSeconds: 3600,
			BigPicture:       "Authorize this ordinary research run to buy one per-call-capped Offer Desk x402 call on the isolated Base fork.",
			TechnicalDetails: "One USD minor unit only, exact Offer Desk catalog target, and one settled retry for run " + run.ID.String(),
		})
		if err != nil {
			t.Fatalf("request distinct CF4 x402 operation grant: %v", err)
		}
		x402Grant, err = service.ApproveAuthorization(ownerCtx, x402Grant.ID)
		if err != nil {
			t.Fatalf("approve CF4 x402 operation grant: %v", err)
		}
		if x402Grant.ID == grant.ID || x402Grant.ID == secondSpendGrant.ID {
			t.Fatal("CF4 x402 spend reused an earlier Agent Manager operation grant")
		}
		if err := writeJSON(filepath.Join(coord, "x402-operation-approved.json"), map[string]any{
			"run_id": run.ID.String(), "grant_id": x402Grant.ID.String(), "spend": "cf4-x402-priced-call",
			"spend_targets": []string{x402Target, x402SettlementTarget}, "read_targets": []string{budgetTarget, "treasury:authorization/" + config.X402AuthorizationID},
			"approved_at": x402Grant.ApprovedAt,
		}); err != nil {
			t.Fatal(err)
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-authorization-ready.json"), nil); err != nil {
			t.Fatalf("Treasury CLI did not propose the per-call CF4 x402 spend: %v", err)
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-call-release"), nil); err != nil {
			t.Fatalf("Treasury harness did not validate the mandate-wallet payer before the CF4 call: %v", err)
		}
		if err := os.WriteFile(filepath.Join(coord, "x402-report-release"), []byte("report one approved per-call x402 spend"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-outcome-unknown.json"), nil); err != nil {
			t.Fatalf("CF4 x402 did not retain its first ambiguous result: %v", err)
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-settled-after-restart.json"), nil); err != nil {
			t.Fatalf("CF4 x402 exact retry did not settle after Treasury restart: %v", err)
		}
		if treasuryCF1MeasuredX402Enabled() {
			var measuredRequest struct {
				RunID                    string `json:"run_id"`
				AuthorizationID          string `json:"authorization_id"`
				SettlementID             string `json:"settlement_id"`
				AmountMinor              int64  `json:"amount_minor"`
				PriceID                  string `json:"price_id"`
				CallID                   string `json:"call_id"`
				InstrumentID             string `json:"instrument_id"`
				RequiredTarget           string `json:"required_target"`
				RequiredSettlementTarget string `json:"required_settlement_target"`
			}
			if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-measured-operation-grant-requested.json"), &measuredRequest); err != nil {
				t.Fatalf("ordinary run did not request the separate measured x402 grant: %v", err)
			}
			measuredAuthorizationID := config.X402AuthorizationID + "-measured"
			measuredSettlementID := config.X402SettlementID + "-measured"
			measuredIdempotency := config.X402IdempotencyKey + "-measured"
			measuredTarget, measuredSettlementTarget := (cliutil.TreasurySpendIntent{
				AuthorizationID: measuredAuthorizationID, MandateID: mandateID, AmountMinor: 1, Currency: "USD",
				Counterparty: config.X402Counterparty, AuthorizationIdempotencyKey: measuredIdempotency,
				SettlementID: measuredSettlementID, InstrumentID: config.X402InstrumentID,
				SettlementIdempotencyKey: measuredIdempotency + "-settlement",
			}).Targets()
			if measuredRequest.RunID != run.ID.String() || measuredRequest.AuthorizationID != measuredAuthorizationID || measuredRequest.SettlementID != measuredSettlementID || measuredRequest.AmountMinor != 1 || measuredRequest.PriceID != config.X402PriceID || measuredRequest.CallID != config.X402CallID || measuredRequest.InstrumentID != config.X402InstrumentID || measuredRequest.RequiredTarget != measuredTarget || measuredRequest.RequiredSettlementTarget != measuredSettlementTarget {
				t.Fatalf("measured x402 operation was not bound to a separate one-minor-unit run spend: %+v", measuredRequest)
			}
			x402MeasuredGrant, err = service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
				RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf5-local-fork", HarnessSession: "commissioned-local-test",
				Rules: []domain.AuthorizationRule{
					{Policy: "treasury-spending", Targets: []string{measuredTarget, measuredSettlementTarget}},
					{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + measuredAuthorizationID}},
				}, DurationSeconds: 3600,
				BigPicture:       "Authorize a second, no-drop priced Offer Desk call in the same local-fork research run and record its measured benefit.",
				TechnicalDetails: "A separate USD minor-unit charge with exact authorization and settlement idempotency IDs for run " + run.ID.String(),
			})
			if err != nil {
				t.Fatalf("request separate measured x402 Agent Manager grant: %v", err)
			}
			x402MeasuredGrant, err = service.ApproveAuthorization(ownerCtx, x402MeasuredGrant.ID)
			if err != nil {
				t.Fatalf("approve separate measured x402 Agent Manager grant: %v", err)
			}
			if x402MeasuredGrant.ID == x402Grant.ID || x402MeasuredGrant.ID == grant.ID || x402MeasuredGrant.ID == secondSpendGrant.ID {
				t.Fatal("measured x402 call reused an earlier Agent Manager operation grant")
			}
			if err := writeJSON(filepath.Join(coord, "x402-measured-operation-approved.json"), map[string]any{
				"run_id": run.ID.String(), "grant_id": x402MeasuredGrant.ID.String(), "spend": "cf5-no-drop-measured-x402-call",
				"spend_targets": []string{measuredTarget, measuredSettlementTarget},
				"read_targets":  []string{budgetTarget, "treasury:authorization/" + measuredAuthorizationID}, "approved_at": x402MeasuredGrant.ApprovedAt,
			}); err != nil {
				t.Fatal(err)
			}
			if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-measured-authorization-ready.json"), nil); err != nil {
				t.Fatalf("Treasury CLI did not propose the separately granted measured x402 spend: %v", err)
			}
			if err := writeJSON(filepath.Join(coord, "x402-measured-call-release"), map[string]any{"released": true}); err != nil {
				t.Fatal(err)
			}
			var measuredResult struct {
				Status           string   `json:"status"`
				RunID            string   `json:"run_id"`
				SettlementID     string   `json:"settlement_id"`
				ReceiptID        string   `json:"receipt_id"`
				LedgerEmissionID string   `json:"ledger_emission_id"`
				AmountMinor      int64    `json:"amount_minor"`
				ResponseBytes    int      `json:"response_bytes"`
				ResponseSHA256   string   `json:"response_sha256"`
				BodyPath         string   `json:"body_path"`
				PriceID          string   `json:"price_id"`
				CallID           string   `json:"call_id"`
				BenefitState     string   `json:"benefit_state"`
				ReturnedNodeIDs  []string `json:"returned_node_ids"`
			}
			if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-measured-result.json"), &measuredResult); err != nil {
				t.Fatalf("measured x402 call did not produce its exact response manifest: %v", err)
			}
			if measuredResult.Status != "settled" || measuredResult.RunID != run.ID.String() || measuredResult.SettlementID != measuredSettlementID || measuredResult.ReceiptID == "" || measuredResult.LedgerEmissionID != measuredSettlementID+":ledger" || measuredResult.AmountMinor != 1 || measuredResult.ResponseBytes <= 0 || len(measuredResult.ResponseSHA256) != 64 || measuredResult.BodyPath == "" || measuredResult.PriceID != config.X402PriceID || measuredResult.CallID != config.X402CallID || measuredResult.BenefitState != "measured" || len(measuredResult.ReturnedNodeIDs) == 0 {
				t.Fatalf("measured x402 result did not match its settled run-bound benefit: %+v", measuredResult)
			}
		}
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "frozen-operation-grant-requested.json"), nil); err != nil {
		t.Fatalf("run did not request an exact frozen-spend grant: %v", err)
	}
	frozenAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_IN_CAP_AUTHORIZATION_ID")) + "-frozen"
	frozenTarget, frozenSettlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: frozenAuthorizationID, MandateID: mandateID, AmountMinor: 10, Currency: "USD", Counterparty: counterparty,
		AuthorizationIdempotencyKey: os.Getenv("TREASURY_CF1_IN_CAP_IDEMPOTENCY_KEY") + "-frozen",
		SettlementID:                strings.TrimSpace(os.Getenv("TREASURY_CF1_IN_CAP_SETTLEMENT_ID")) + "-frozen", InstrumentID: strings.TrimSpace(os.Getenv("TREASURY_CF1_IN_CAP_INSTRUMENT_ID")),
		SettlementIdempotencyKey: os.Getenv("TREASURY_CF1_IN_CAP_IDEMPOTENCY_KEY") + "-frozen-settlement",
	}).Targets()
	frozenGrant, err := service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
		RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf1-local-fork", HarnessSession: "commissioned-local-test",
		Rules: []domain.AuthorizationRule{
			{Policy: "treasury-spending", Targets: []string{frozenTarget, frozenSettlementTarget}},
			{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + frozenAuthorizationID}},
		}, DurationSeconds: 3600,
		BigPicture:       "Authorize an exact frozen-budget spend request so Treasury's freeze can refuse it.",
		TechnicalDetails: "One spend only: 10 USD minor units for authorization " + frozenAuthorizationID + "; expected budget_frozen refusal on the isolated Base fork; no dispatch.",
	})
	if err != nil {
		t.Fatalf("request exact frozen-spend operation grant: %v", err)
	}
	frozenGrantApproval, err := service.ApproveAuthorization(ownerCtx, frozenGrant.ID)
	if err != nil {
		t.Fatalf("approve exact frozen-spend operation grant: %v", err)
	}
	if frozenGrant.ID == grant.ID || frozenGrant.ID == secondSpendGrant.ID {
		t.Fatal("frozen spend reused an earlier Agent Manager operation grant")
	}
	if err := writeJSON(filepath.Join(coord, "frozen-operation-approved.json"), map[string]any{
		"run_id": run.ID.String(), "grant_id": frozenGrant.ID.String(), "spend": "frozen-denial",
		"spend_targets": []string{frozenTarget, frozenSettlementTarget}, "read_targets": []string{budgetTarget, "treasury:authorization/" + frozenAuthorizationID}, "approved_at": frozenGrantApproval.ApprovedAt,
	}); err != nil {
		t.Fatal(err)
	}
	expiryMandateID := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPIRY_MANDATE_ID"))
	expiryAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPIRY_AUTHORIZATION_ID"))
	expirySettlementID := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPIRY_SETTLEMENT_ID"))
	expiryInstrumentID := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPIRY_INSTRUMENT_ID"))
	expiryIdempotency := strings.TrimSpace(os.Getenv("TREASURY_CF1_EXPIRY_IDEMPOTENCY_KEY"))
	if config.ExpiryMandateID != expiryMandateID || expiryAuthorizationID != config.ExpiryAuthorization || expirySettlementID != config.ExpirySettlement || expiryInstrumentID != config.ExpiryInstrumentID || expiryIdempotency != config.ExpiryIdempotency {
		t.Fatal("expired-mandate identity was not preserved from the local-fork harness")
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "expired-operation-grant-requested.json"), nil); err != nil {
		t.Fatalf("run did not request the exact expired-mandate operation grant: %v", err)
	}
	expiryTarget, expirySettlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: expiryAuthorizationID, MandateID: expiryMandateID, AmountMinor: 10, Currency: "USD", Counterparty: counterparty,
		AuthorizationIdempotencyKey: expiryIdempotency, SettlementID: expirySettlementID,
		InstrumentID: expiryInstrumentID, SettlementIdempotencyKey: expiryIdempotency + "-settlement",
	}).Targets()
	expiryGrant, err := service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
		RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf1-local-fork", HarnessSession: "commissioned-local-test",
		Rules: []domain.AuthorizationRule{
			{Policy: "treasury-spending", Targets: []string{expiryTarget, expirySettlementTarget}},
			{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + expiryAuthorizationID}},
		}, DurationSeconds: 3600,
		BigPicture:       "Authorize one exact Treasury attempt after its short-lived test mandate expires.",
		TechnicalDetails: "One 10 USD minor-unit proposal under " + expiryMandateID + "; expected mandate_live refusal and no dispatch in the local fork.",
	})
	if err != nil {
		t.Fatalf("request exact expired-mandate operation grant: %v", err)
	}
	expiryGrantApproval, err := service.ApproveAuthorization(ownerCtx, expiryGrant.ID)
	if err != nil {
		t.Fatalf("approve exact expired-mandate operation grant: %v", err)
	}
	var expiryMandate struct {
		MandateID string    `json:"mandate_id"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	if err := waitForFile(ctx, filepath.Join(coord, "expired-mandate-created.json"), &expiryMandate); err != nil || expiryMandate.MandateID != expiryMandateID || expiryMandate.ExpiresAt.IsZero() {
		t.Fatalf("read the short-lived Treasury mandate receipt: mandate=%+v err=%v", expiryMandate, err)
	}
	if err := writeJSON(filepath.Join(coord, "expired-operation-approved.json"), map[string]any{
		"run_id": run.ID.String(), "grant_id": expiryGrant.ID.String(), "spend": "expired-mandate-denial",
		"spend_targets": []string{expiryTarget, expirySettlementTarget}, "approved_at": expiryGrantApproval.ApprovedAt,
	}); err != nil {
		t.Fatal(err)
	}
	if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "expired-mandate-authorization-ready.json"), nil); err != nil {
		t.Fatalf("expired-mandate attempt did not finish: %v", err)
	}
	var expiredDecision struct {
		Authorization struct {
			Verdict            string `json:"verdict"`
			ViolatedConstraint string `json:"violated_constraint"`
		} `json:"authorization"`
	}
	decisionBytes, err := os.ReadFile(filepath.Join(coord, "authorization-propose-expired.log"))
	if err != nil {
		t.Fatalf("decode expired-mandate decision: %v", err)
	}
	if err := json.Unmarshal(decisionBytes, &expiredDecision); err != nil {
		t.Fatalf("decode expired-mandate decision: %v", err)
	}
	if expiredDecision.Authorization.Verdict != "AUTHORIZATION_VERDICT_REFUSED" || expiredDecision.Authorization.ViolatedConstraint != "mandate_live" {
		t.Fatalf("expired mandate did not fail closed: %+v", expiredDecision.Authorization)
	}
	if err := writeJSON(filepath.Join(coord, "expired-mandate-decision.json"), map[string]any{
		"mandate_id": expiryMandateID, "authorization_id": expiryAuthorizationID,
		"verdict": expiredDecision.Authorization.Verdict, "violated_constraint": expiredDecision.Authorization.ViolatedConstraint,
	}); err != nil {
		t.Fatal(err)
	}
	finalRun := waitForCompletedRun(t, ctx, service, run.ID, 3*time.Minute)
	if err := waitForCF1SessionRevocation(ctx, keys, run.ID); err != nil {
		t.Fatal(err)
	}
	var outcome struct {
		RunID       string   `json:"run_id"`
		CLICommands []string `json:"cli_commands"`
		SessionKey  bool     `json:"session_private_key_injected"`
		Identity    bool     `json:"identity_token_injected"`
		SpendGrants []string `json:"per_spend_operation_grant_ids"`
	}
	data, err := os.ReadFile(runResultPath)
	if err != nil || json.Unmarshal(data, &outcome) != nil {
		t.Fatalf("read run result evidence: %v", err)
	}
	expectedCommands := 11
	expectedGrants := []string{grant.ID.String(), secondSpendGrant.ID.String()}
	if x402Grant != nil {
		expectedCommands += 3
		expectedGrants = append(expectedGrants, x402Grant.ID.String())
	}
	if x402MeasuredGrant != nil {
		expectedCommands++
		expectedGrants = append(expectedGrants, x402MeasuredGrant.ID.String())
	}
	expectedGrants = append(expectedGrants, frozenGrant.ID.String(), expiryGrant.ID.String())
	if outcome.RunID != run.ID.String() || !outcome.SessionKey || !outcome.Identity || len(outcome.CLICommands) != expectedCommands || len(outcome.SpendGrants) != len(expectedGrants) {
		t.Fatalf("incomplete Agent Manager CLI proof: %+v", outcome)
	}
	for index, expected := range expectedGrants {
		if outcome.SpendGrants[index] != expected {
			t.Fatalf("per-spend operation grant %d = %q, want %q", index, outcome.SpendGrants[index], expected)
		}
	}
	var overCapRetryContinuity struct {
		RunID                        string `json:"run_id"`
		AuthorizationID              string `json:"authorization_id"`
		SettlementID                 string `json:"settlement_id"`
		InstrumentID                 string `json:"instrument_id"`
		StatusBeforeRestart          string `json:"status_before_restart"`
		StatusAfterRestart           string `json:"status_after_restart"`
		TransactionHashBeforeRestart string `json:"transaction_hash_before_restart"`
		TransactionHashAfterRestart  string `json:"transaction_hash_after_restart"`
		SameTransactionHash          bool   `json:"same_transaction_hash"`
		AmountMinorBeforeRestart     int64  `json:"amount_minor_before_restart"`
		AmountMinorAfterRestart      int64  `json:"amount_minor_after_restart"`
		Currency                     string `json:"currency"`
		ReceiptReferenceBytes        int    `json:"receipt_reference_bytes"`
		ReceiptReferenceSHA256       string `json:"receipt_reference_sha256"`
		SameReceiptReferenceBytes    bool   `json:"same_receipt_reference_bytes"`
		LedgerDebitCountBeforeRetry  int    `json:"ledger_debit_count_before_retry"`
		LedgerDebitCountAfterRetry   int    `json:"ledger_debit_count_after_retry"`
		HeadroomMinorBeforeRetry     int64  `json:"headroom_minor_before_retry"`
		HeadroomMinorAfterRetry      int64  `json:"headroom_minor_after_retry"`
	}
	if err := waitForFile(ctx, filepath.Join(coord, "over-cap-restart-retry-continuity.json"), &overCapRetryContinuity); err != nil {
		t.Fatalf("read ordinary-run over-cap retry continuity artifact: %v", err)
	}
	expectedInCapMinor, expectedOverCapMinor := int64(30), int64(110)
	expectedRetryHeadroom := int64(1000) - expectedInCapMinor - expectedOverCapMinor
	expectedOverCapSettlementID := strings.TrimSpace(os.Getenv("TREASURY_CF1_OVER_CAP_SETTLEMENT_ID"))
	expectedOverCapInstrumentID := strings.TrimSpace(os.Getenv("TREASURY_CF1_OVER_CAP_INSTRUMENT_ID"))
	if overCapRetryContinuity.RunID != run.ID.String() || overCapRetryContinuity.AuthorizationID != overCapAuthorizationID || overCapRetryContinuity.SettlementID != expectedOverCapSettlementID || overCapRetryContinuity.InstrumentID != expectedOverCapInstrumentID || overCapRetryContinuity.StatusBeforeRestart != "CHARGE_OUTCOME_SETTLED" || overCapRetryContinuity.StatusAfterRestart != overCapRetryContinuity.StatusBeforeRestart || !cf1HexIdentity(overCapRetryContinuity.TransactionHashBeforeRestart, 32) || overCapRetryContinuity.TransactionHashAfterRestart != overCapRetryContinuity.TransactionHashBeforeRestart || !overCapRetryContinuity.SameTransactionHash || overCapRetryContinuity.AmountMinorBeforeRestart != expectedOverCapMinor || overCapRetryContinuity.AmountMinorAfterRestart != expectedOverCapMinor || overCapRetryContinuity.Currency != "USD" || overCapRetryContinuity.ReceiptReferenceBytes <= 0 || len(overCapRetryContinuity.ReceiptReferenceSHA256) != 64 || !cf1HexIdentity(overCapRetryContinuity.ReceiptReferenceSHA256, 32) || !overCapRetryContinuity.SameReceiptReferenceBytes || overCapRetryContinuity.LedgerDebitCountBeforeRetry != 1 || overCapRetryContinuity.LedgerDebitCountAfterRetry != 1 || overCapRetryContinuity.HeadroomMinorBeforeRetry != expectedRetryHeadroom || overCapRetryContinuity.HeadroomMinorAfterRetry != expectedRetryHeadroom {
		t.Fatalf("ordinary-run over-cap retry continuity artifact does not prove one exact debit across restart: %+v", overCapRetryContinuity)
	}
	if x402Grant != nil {
		var settled struct {
			AuthorizationID string `json:"authorization_id"`
			SettlementID    string `json:"settlement_id"`
			OperationGrant  string `json:"operation_grant_id"`
			SameAuth        bool   `json:"same_authorization"`
			SameSettlement  bool   `json:"same_settlement_id"`
			SameIdempotency bool   `json:"same_idempotency_key"`
			TransactionHash string `json:"transaction_hash"`
		}
		if err := waitForFile(ctx, filepath.Join(coord, "x402-settled-after-restart.json"), &settled); err != nil {
			t.Fatalf("read CF4 x402 settled retry evidence: %v", err)
		}
		if settled.AuthorizationID != config.X402AuthorizationID || settled.SettlementID != config.X402SettlementID || settled.OperationGrant != x402Grant.ID.String() || !settled.SameAuth || !settled.SameSettlement || !settled.SameIdempotency || !strings.HasPrefix(strings.ToLower(settled.TransactionHash), "0x") {
			t.Fatalf("CF4 x402 retry did not preserve the run's exact spend and chain receipt: %+v", settled)
		}
	}
	if err := writeJSON(filepath.Join(coord, "agent-manager-result.json"), map[string]any{
		"run_id": run.ID.String(), "profile_id": profile.ID.String(), "task_id": task.ID.String(),
		"run_status": finalRun.Status, "terminal_session_revoked": keys.WasRevoked(run.ID),
		"terminal_session_key_absent": keys.WasRevoked(run.ID), "session_key_authority_backend": "isolated-encrypted-file",
		"cli_commands": outcome.CLICommands, "per_spend_operation_grant_ids": outcome.SpendGrants, "private_values_recorded": false,
	}); err != nil {
		t.Fatal(err)
	}
}
