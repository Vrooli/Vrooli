//go:build linux

package integration

import (
	"context"
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

type cf1TreasuryOperatorVerifier struct{}

func (cf1TreasuryOperatorVerifier) Verify(_ context.Context, token string) (coreidentity.Principal, error) {
	if token != cf1TreasuryOperator {
		return coreidentity.Principal{}, errors.New("invalid local CF1 operator fixture")
	}
	policy, ok := authn.InteractivePolicy("treasury-spending")
	if !ok {
		return coreidentity.Principal{}, errors.New("Treasury spending policy unavailable")
	}
	scopes := []string{"agent-manager:write", "agent-manager:read", "agent-manager:supervise"}
	for _, scope := range policy.Scopes {
		found := false
		for _, existing := range scopes {
			found = found || existing == scope
		}
		if !found {
			scopes = append(scopes, scope)
		}
	}
	readPolicy, ok := authn.InteractivePolicy("treasury-wallet-reading")
	if !ok {
		return coreidentity.Principal{}, errors.New("Treasury wallet-reading policy unavailable")
	}
	for _, scope := range readPolicy.Scopes {
		found := false
		for _, existing := range scopes {
			found = found || existing == scope
		}
		if !found {
			scopes = append(scopes, scope)
		}
	}
	return coreidentity.Principal{Kind: coreidentity.ActorHuman, Verified: true, Subject: "cf1-local-operator", Scopes: scopes, ExpiresAt: time.Now().Add(time.Hour)}, nil
}

type cf1RunSessionKeys struct {
	store *orchestration.CredentialAuthorityTreasurySessionKeys
	path  string
}

func (s *cf1RunSessionKeys) ResolveOrMint(ctx context.Context, id uuid.UUID) (string, error) {
	return s.store.ResolveOrMint(ctx, id)
}

func (s *cf1RunSessionKeys) RevokeRunCredentialUse(ctx context.Context, id uuid.UUID, reason string) error {
	return s.store.RevokeRunCredentialUse(ctx, id, reason)
}

func (s *cf1RunSessionKeys) WasRevoked(id uuid.UUID) bool {
	identity, err := credentialauthority.ParseIdentity("vrooli/agent-manager-run-" + id.String())
	if err != nil {
		return false
	}
	authority, err := credentialauthority.OpenEncryptedFile(s.path, cf1RunKeyPassphrase)
	if err != nil {
		return false
	}
	_, err = authority.Resolve(identity, cf1RunKeyField)
	return errors.Is(err, credentialauthority.ErrUnconfigured)
}

type cf1TreasuryRunConfig struct {
	APIBaseURL           string `json:"api_base_url"`
	CLIBinary            string `json:"cli_binary"`
	BudgetID             string `json:"budget_id"`
	MandateID            string `json:"mandate_id"`
	IsolationBudgetID    string `json:"isolation_budget_id"`
	IsolationMandateID   string `json:"isolation_mandate_id"`
	Counterparty         string `json:"counterparty"`
	InCapInstrumentID    string `json:"in_cap_instrument_id"`
	InCapAmountMinor     int64  `json:"in_cap_amount_minor"`
	InCapAuthorization   string `json:"in_cap_authorization_id"`
	InCapSettlement      string `json:"in_cap_settlement_id"`
	InCapIdempotency     string `json:"in_cap_idempotency_key"`
	OverCapInstrumentID  string `json:"over_cap_instrument_id"`
	OverCapAmountMinor   int64  `json:"over_cap_amount_minor"`
	OverCapAuthorization string `json:"over_cap_authorization_id"`
	OverCapSettlement    string `json:"over_cap_settlement_id"`
	OverCapIdempotency   string `json:"over_cap_idempotency_key"`
	ExpiryMandateID      string `json:"expiry_mandate_id"`
	ExpiryAuthorization  string `json:"expiry_authorization_id"`
	ExpirySettlement     string `json:"expiry_settlement_id"`
	ExpiryInstrumentID   string `json:"expiry_instrument_id"`
	ExpiryIdempotency    string `json:"expiry_idempotency_key"`
	InCapGrantID         string `json:"in_cap_operation_grant_id"`
	OverCapGrantID       string `json:"over_cap_operation_grant_id"`
	// CF4 is an additive extension: older CF1 local-fork rehearsals omit these
	// fields and retain their existing flow unchanged.
	X402CatalogURL      string                `json:"x402_catalog_url"`
	X402PriceID         string                `json:"x402_price_id"`
	X402CallID          string                `json:"x402_call_id"`
	X402Counterparty    string                `json:"x402_counterparty"`
	X402InstrumentID    string                `json:"x402_instrument_id"`
	X402AmountMinor     int64                 `json:"x402_expected_amount_minor"`
	X402AuthorizationID string                `json:"x402_authorization_id"`
	X402SettlementID    string                `json:"x402_settlement_id"`
	X402IdempotencyKey  string                `json:"x402_idempotency_key"`
	X402Calls           []cf1TreasuryX402Call `json:"x402_calls"`
}

// cf1TreasuryX402Call is one exact priced call inside the same ordinary run.
// Every row has its own Treasury authorization, settlement, idempotency key,
// and Agent Manager grant; no call inherits another call's authority.
type cf1TreasuryX402Call struct {
	PriceID         string `json:"price_id"`
	CallID          string `json:"call_id"`
	CatalogURL      string `json:"catalog_url"`
	Counterparty    string `json:"counterparty"`
	InstrumentID    string `json:"instrument_id"`
	AuthorizationID string `json:"authorization_id"`
	SettlementID    string `json:"settlement_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AmountMinor     int64  `json:"amount_minor"`
}

type cf7X402SettlementReceipt struct {
	Settlement struct {
		Id            string `json:"id"`
		Authorization string `json:"authorization_id"`
		AmountMinor   int64  `json:"amount_minor"`
		Currency      string `json:"currency"`
		Outcome       string `json:"outcome"`
		ExternalID    string `json:"external_id"`
		InstrumentID  string `json:"instrument_id"`
		SettledAt     string `json:"settled_at"`
		Receipt       string `json:"receipt_reference"`
		Detail        string `json:"detail"`
	} `json:"settlement"`
}

// TestTreasuryCLIFromOrdinaryRun is invoked by Treasury's opt-in local-fork
// rehearsal. The fixture waits for the Treasury owner to create a run-bound
// mandate and instrument, then exercises the real CLI through the normal
// CreateProfile -> CreateTask -> CreateRun -> RunExecutor path.
func TestTreasuryCLIFromOrdinaryRun(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-004] [REQ:TRS-P0-006]
	coord := strings.TrimSpace(os.Getenv("TREASURY_CF1_COORD_DIR"))
	cf7Only := os.Getenv("TREASURY_CF7_ONLY") == "1"
	if cf7Only {
		coord = strings.TrimSpace(os.Getenv("TREASURY_CF7_COORD_DIR"))
	}
	if coord == "" {
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
	if !filepath.IsAbs(authorityPath) {
		t.Fatal("isolated encrypted Credential Authority path must be absolute")
	}
	keyStore := orchestration.NewCredentialAuthorityTreasurySessionKeysWithOpener(func() (*credentialauthority.Authority, error) {
		return credentialauthority.OpenEncryptedFile(authorityPath, cf1RunKeyPassphrase)
	})
	keys := &cf1RunSessionKeys{store: keyStore, path: authorityPath}
	runResultPath := filepath.Join(coord, "run-result.json")
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
		var config cf1TreasuryRunConfig
		if err := waitForFile(execCtx, configPath, &config); err != nil {
			return nil, err
		}
		if cf7Only {
			if err := executeCF7X402Run(execCtx, db, request, config, coord, write); err != nil {
				return nil, err
			}
			if request.EventSink == nil {
				return nil, errors.New("Agent Manager run event sink is unavailable")
			}
			if err := request.EventSink.Emit(domain.NewMessageEvent(request.RunID, "assistant", "Settled three priced local-fork research calls; fourth call is awaiting separate authorization.")); err != nil {
				return nil, fmt.Errorf("record CF7 run completion message: %w", err)
			}
			if err := write("run-result.json", map[string]any{
				"run_id": request.RunID.String(), "profile_id": request.Profile.ID.String(), "task_id": request.Task.ID.String(),
				"ordinary_create_run": true, "settled_calls": 3, "fourth_call_dispatch": "none",
				"x402_calls": config.X402Calls,
			}); err != nil {
				return nil, err
			}
			return &runner.ExecuteResult{Success: true, ExitCode: 0, Summary: &domain.RunSummary{Description: "CF7 local-fork x402 sequence completed with fourth call held for decision"}}, nil
		}
		if config.APIBaseURL == "" || config.CLIBinary == "" || config.MandateID == "" || config.InCapInstrumentID == "" || config.InCapAmountMinor <= 0 || config.OverCapInstrumentID == "" || config.OverCapAmountMinor <= 0 || config.ExpiryMandateID == "" || config.ExpiryAuthorization == "" || config.ExpirySettlement == "" || config.ExpiryInstrumentID == "" {
			return nil, errors.New("Treasury run config is incomplete")
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
		if request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] == "" {
			return nil, errors.New("ordinary run did not receive its verified identity and run-owned session key")
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
		runCLI := func(label string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(execCtx, config.CLIBinary, args...)
			cmd.Env = env
			output, runErr := cmd.CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(coord, label+".log"), output, 0o600); writeErr != nil {
				return output, writeErr
			}
			if runErr != nil {
				return output, fmt.Errorf("Treasury CLI %s failed: %w: %s", label, runErr, strings.TrimSpace(string(output)))
			}
			return output, nil
		}
		if _, err := runCLI("agent-wallet", "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", config.BudgetID, "--mandate-id", config.MandateID, "--json"); err != nil {
			return nil, err
		}
		if config.IsolationBudgetID != "" || config.IsolationMandateID != "" {
			if config.IsolationBudgetID == "" || config.IsolationMandateID == "" {
				return nil, errors.New("cross-budget isolation probe requires both budget and mandate IDs")
			}
			probeCmd := exec.CommandContext(execCtx, config.CLIBinary, "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", config.IsolationBudgetID, "--mandate-id", config.IsolationMandateID, "--json")
			probeCmd.Env = env
			probeOutput, probeErr := probeCmd.CombinedOutput()
			if writeErr := os.WriteFile(filepath.Join(coord, "cross-budget-wallet-probe.log"), probeOutput, 0o600); writeErr != nil {
				return nil, writeErr
			}
			if probeErr == nil {
				return nil, errors.New("ordinary Agent Manager run read another run's budget wallet")
			}
			probeText := strings.ToLower(string(probeOutput))
			if !strings.Contains(probeText, "permission_denied") || !strings.Contains(probeText, "mandate wallet session is inactive") {
				return nil, fmt.Errorf("cross-budget wallet probe failed for an unexpected reason: %s", strings.TrimSpace(string(probeOutput)))
			}
			if err := write("cross-budget-wallet-denied.json", map[string]any{
				"requesting_run": request.RunID.String(), "target_budget_id": config.IsolationBudgetID,
				"target_mandate_id": config.IsolationMandateID, "result": "denied", "address_returned": false,
				"authorization_scope_includes_target": true, "reason": "agent subject/run mismatch prevents session-key binding",
				"error_message": strings.TrimSpace(string(probeOutput)),
			}); err != nil {
				return nil, err
			}
		}
		if _, err := runCLI("budget-headroom-before", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
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
		if _, err := runCLI("settlement-report-in-cap", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.InCapAuthorization, "--settlement-id", config.InCapSettlement,
			"--instrument-id", config.InCapInstrumentID, "--idempotency-key", config.InCapIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
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
		if _, err := runCLI("budget-headroom-after-in-cap", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json"); err != nil {
			return nil, err
		}
		if err := readHeadroom("budget-headroom-after-in-cap", config.InCapAmountMinor, 1000-config.InCapAmountMinor); err != nil {
			return nil, err
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
		if _, err := runCLI("settlement-report-over-cap-first", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.OverCapAuthorization, "--settlement-id", config.OverCapSettlement,
			"--instrument-id", config.OverCapInstrumentID, "--idempotency-key", config.OverCapIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
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
		if err := waitForFile(execCtx, filepath.Join(coord, "over-cap-retry-release"), nil); err != nil {
			return nil, err
		}
		if _, err := runCLI("settlement-report-over-cap-after-restart", "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", config.OverCapAuthorization, "--settlement-id", config.OverCapSettlement,
			"--instrument-id", config.OverCapInstrumentID, "--idempotency-key", config.OverCapIdempotency+"-settlement", "--json"); err != nil {
			return nil, err
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
		var x402GrantID string
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
					Outcome    string `json:"outcome"`
					ExternalID string `json:"external_id"`
				} `json:"settlement"`
			}
			if err := json.Unmarshal(firstOutcomeData, &firstOutcome); err != nil || firstOutcome.Settlement.Outcome != "CHARGE_OUTCOME_UNKNOWN" {
				return nil, fmt.Errorf("first CF4 x402 outcome was not durably reported unknown (outcome=%q, decode=%v)", firstOutcome.Settlement.Outcome, err)
			}
			if err := write("x402-outcome-unknown.json", map[string]any{
				"authorization_id": config.X402AuthorizationID, "settlement_id": config.X402SettlementID,
				"operation_grant_id": x402GrantID, "retry_idempotency_key": x402SettlementIdempotency,
				"catalog_url": config.X402CatalogURL, "price_id": config.X402PriceID, "call_id": config.X402CallID,
				"retained_external_id": firstOutcome.Settlement.ExternalID,
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
					Outcome    string `json:"outcome"`
					ExternalID string `json:"external_id"`
				} `json:"settlement"`
			}
			if err := json.Unmarshal(retryOutcomeData, &retryOutcome); err != nil || retryOutcome.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || strings.TrimSpace(retryOutcome.Settlement.ExternalID) == "" {
				return nil, fmt.Errorf("exact CF4 x402 retry did not return a settled transaction hash (outcome=%q, external_id=%q, decode=%v)", retryOutcome.Settlement.Outcome, retryOutcome.Settlement.ExternalID, err)
			}
			if err := write("x402-settled-after-restart.json", map[string]any{
				"authorization_id": config.X402AuthorizationID, "settlement_id": config.X402SettlementID,
				"operation_grant_id": x402GrantID, "same_authorization": true,
				"same_settlement_id": true, "same_idempotency_key": true,
				"transaction_hash": retryOutcome.Settlement.ExternalID,
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
		}
		result := map[string]any{
			"run_id": request.RunID.String(), "profile_id": request.Profile.ID.String(), "task_id": request.Task.ID.String(),
			"runner": string(request.ResolvedConfig.RunnerType), "cli_commands": cliCommands,
			"session_private_key_injected": true, "identity_token_injected": true,
			"per_spend_operation_grant_ids": func() []string {
				grants := []string{config.InCapGrantID, overCapGrantReceipt.GrantID}
				if x402GrantID != "" {
					grants = append(grants, x402GrantID)
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
	service := orchestration.New(
		repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: 4 * time.Minute, MaxConcurrentRuns: 1, RequireSandboxByDefault: false}),
		orchestration.WithEvents(events), orchestration.WithRunners(registry), orchestration.WithCheckpoints(repos.Checkpoints),
		orchestration.WithIdempotency(repos.Idempotency), orchestration.WithRunStateRoot(filepath.Join(coord, "run-state")),
		orchestration.WithIdentitySecret([]byte(cf1TreasuryRunSecret)),
		orchestration.WithTreasurySessionKeyStore(keys),
		orchestration.WithAuthorizations(database.NewAuthorizationRepository(db)),
		orchestration.WithSandbox(mocks.NewFakeSandboxProvider()),
		orchestration.WithOwnerIdentity(cf1TreasuryOperatorVerifier{}), newTestRolePolicyOption(t),
	)
	stopBroker, err := service.StartAuthorizationBroker()
	if err != nil {
		t.Fatalf("start local Agent Manager credential broker: %v", err)
	}
	t.Cleanup(func() { _ = stopBroker() })
	profileName, taskTitle, taskDescription := "cf1-local-treasury-cli", "CF1 local Treasury CLI", "exercise a run-owned budget session against the local fork"
	runTag := ""
	if cf7Only {
		profileName, taskTitle, taskDescription = "cf7-research-budget-x402", "CF7 local research budget x402 calls", "buy three priced local-fork research calls and hold a fourth over-cap quote for decision"
		runTag = orchestration.CF7ResearchBudgetTestRunTag
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
		Prompt: "Use only the run-scoped Treasury CLI session to settle one local-fork test payment.", RequestedScopes: []string{"treasury:read", "treasury:write"}, Tag: runTag,
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
	config, err := readCF1RunConfig(coord)
	if err != nil {
		t.Fatalf("wait for Treasury local-fork setup: %v", err)
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
	finalRun, err := waitForTerminal(t, ctx, service, run.ID, 3*time.Minute)
	if err != nil {
		t.Fatalf("ordinary Treasury run did not reach terminal state: %v", err)
	}
	if finalRun.Status != domain.RunStatusComplete {
		t.Fatalf("ordinary Treasury run failed: status=%s error=%s", finalRun.Status, finalRun.ErrorMsg)
	}
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
	expectedGrants = append(expectedGrants, frozenGrant.ID.String(), expiryGrant.ID.String())
	if outcome.RunID != run.ID.String() || !outcome.SessionKey || !outcome.Identity || len(outcome.CLICommands) != expectedCommands || len(outcome.SpendGrants) != len(expectedGrants) {
		t.Fatalf("incomplete Agent Manager CLI proof: %+v", outcome)
	}
	for index, expected := range expectedGrants {
		if outcome.SpendGrants[index] != expected {
			t.Fatalf("per-spend operation grant %d = %q, want %q", index, outcome.SpendGrants[index], expected)
		}
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

func readCF1RunConfig(coord string) (cf1TreasuryRunConfig, error) {
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		path := filepath.Join(coord, "treasury-config.json")
		if data, err := os.ReadFile(path); err == nil {
			var config cf1TreasuryRunConfig
			if err := json.Unmarshal(data, &config); err != nil {
				return cf1TreasuryRunConfig{}, err
			}
			return config, nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cf1TreasuryRunConfig{}, errors.New("timed out waiting for Treasury fixture configuration")
}

func waitForFile(ctx context.Context, path string, output any) error {
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if output != nil {
				if err := json.Unmarshal(data, output); err != nil {
					return err
				}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitForFileOrRunTerminal(ctx context.Context, service *orchestration.Orchestrator, runID uuid.UUID, path string, output any) error {
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			if output != nil {
				if err := json.Unmarshal(data, output); err != nil {
					return err
				}
			}
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		run, runErr := service.GetStoredRun(ctx, runID)
		if runErr == nil && run.Status.IsTerminal() {
			return fmt.Errorf("Agent Manager run became terminal before writing %s: status=%s error=%s", filepath.Base(path), run.Status, run.ErrorMsg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func waitForCF1SessionRevocation(ctx context.Context, keys *cf1RunSessionKeys, runID uuid.UUID) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if keys.WasRevoked(runID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for run-owned session revocation: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func waitForRunReceipt(ctx context.Context, service *orchestration.Orchestrator, runID uuid.UUID, path string) error {
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		run, err := service.GetRun(ctx, runID)
		if err != nil {
			return fmt.Errorf("read Agent Manager run while waiting for runner receipt: %w", err)
		}
		if run.Status.IsTerminal() {
			return fmt.Errorf("Agent Manager run terminated before the runner wrote its receipt: status=%s error=%s", run.Status, run.ErrorMsg)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
