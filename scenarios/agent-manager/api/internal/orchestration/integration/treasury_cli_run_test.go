//go:build linux

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
)

const (
	cf1TreasuryRunSecret = "local-fork-only-agent-manager-identity-secret"
	cf1TreasuryOperator  = "cf1-local-test-operator"
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
	mu      sync.Mutex
	keys    map[uuid.UUID]string
	revoked map[uuid.UUID]bool
}

func (s *cf1RunSessionKeys) ResolveOrMint(_ context.Context, id uuid.UUID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id == uuid.Nil {
		return "", errors.New("run ID is required")
	}
	if s.keys == nil {
		s.keys = make(map[uuid.UUID]string)
	}
	if s.keys[id] == "" {
		// Deterministic test-only scalar. It is never written to the run artifacts.
		s.keys[id] = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}
	return s.keys[id], nil
}

func (s *cf1RunSessionKeys) RevokeRunCredentialUse(_ context.Context, id uuid.UUID, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked == nil {
		s.revoked = make(map[uuid.UUID]bool)
	}
	s.revoked[id] = true
	delete(s.keys, id)
	return nil
}

func (s *cf1RunSessionKeys) WasRevoked(id uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revoked[id]
}

type cf1TreasuryRunConfig struct {
	APIBaseURL           string `json:"api_base_url"`
	CLIBinary            string `json:"cli_binary"`
	BudgetID             string `json:"budget_id"`
	MandateID            string `json:"mandate_id"`
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
}

// TestTreasuryCLIFromOrdinaryRun is invoked by Treasury's opt-in local-fork
// rehearsal. The fixture waits for the Treasury owner to create a run-bound
// mandate and instrument, then exercises the real CLI through the normal
// CreateProfile -> CreateTask -> CreateRun -> RunExecutor path.
func TestTreasuryCLIFromOrdinaryRun(t *testing.T) { // [REQ:TRS-P0-001] [REQ:TRS-P0-004] [REQ:TRS-P0-006]
	coord := strings.TrimSpace(os.Getenv("TREASURY_CF1_COORD_DIR"))
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
	keys := &cf1RunSessionKeys{}
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
		if config.APIBaseURL == "" || config.CLIBinary == "" || config.MandateID == "" || config.InCapInstrumentID == "" || config.InCapAmountMinor <= 0 || config.OverCapInstrumentID == "" || config.OverCapAmountMinor <= 0 || config.ExpiryMandateID == "" || config.ExpiryAuthorization == "" || config.ExpirySettlement == "" || config.ExpiryInstrumentID == "" {
			return nil, errors.New("Treasury run config is incomplete")
		}
		if request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] == "" {
			return nil, errors.New("ordinary run did not receive its verified identity and run-owned session key")
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
		if _, err := runCLI("agent-wallet", "--api-base", config.APIBaseURL, "budget", "agent-wallet", "--budget-id", config.BudgetID, "--json"); err != nil {
			return nil, err
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
		result := map[string]any{
			"run_id": request.RunID.String(), "profile_id": request.Profile.ID.String(), "task_id": request.Task.ID.String(),
			"runner": string(request.ResolvedConfig.RunnerType), "cli_commands": []string{"budget agent-wallet", "budget headroom before and after settlements", "authorization propose in-cap", "settlement report-outcome in-cap", "budget agent-wallet after mandate rotation", "authorization propose over-cap", "settlement report-outcome over-cap", "same settlement retry after restart", "headroom unchanged after retry", "authorization propose while frozen", "authorization propose after mandate expiry"},
			"session_private_key_injected": true, "identity_token_injected": true,
			"per_spend_operation_grant_ids": []string{config.InCapGrantID, overCapGrantReceipt.GrantID, frozenGrantReceipt.GrantID, expiryGrantReceipt.GrantID},
			"private_values_recorded":       false,
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
	profile, err := service.CreateProfile(ctx, &domain.AgentProfile{
		ID: uuid.New(), Name: "cf1-local-treasury-cli", RoleRef: "code.default",
		DeclaredScopes: []string{"treasury:read", "treasury:write"},
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	projectRoot := t.TempDir()
	task, err := service.CreateTask(ctx, &domain.Task{ID: uuid.New(), Title: "CF1 local Treasury CLI", Description: "exercise a run-owned budget session against the local fork", ScopePath: projectRoot, ProjectRoot: projectRoot, Status: domain.TaskStatusQueued})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	run, err := service.CreateRun(ctx, orchestration.CreateRunRequest{
		OwnerToken: cf1TreasuryOperator, ExpectedOwnerSubject: "cf1-local-operator", TaskID: task.ID, AgentProfileID: &profile.ID,
		Prompt: "Use only the run-scoped Treasury CLI session to settle one local-fork test payment.", RequestedScopes: []string{"treasury:read", "treasury:write"},
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
	budgetTarget := "treasury:budget/" + os.Getenv("TREASURY_CF1_BUDGET_ID")
	mandateID := strings.TrimSpace(os.Getenv("TREASURY_CF1_MANDATE_ID"))
	counterparty := strings.TrimSpace(os.Getenv("TREASURY_CF1_COUNTERPARTY"))
	inCapAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_IN_CAP_AUTHORIZATION_ID"))
	overCapAuthorizationID := strings.TrimSpace(os.Getenv("TREASURY_CF1_OVER_CAP_AUTHORIZATION_ID"))
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
			{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + inCapAuthorizationID}},
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
		"spend_targets": []string{inCapTarget, inCapSettlementTarget}, "read_targets": []string{budgetTarget, "treasury:authorization/" + inCapAuthorizationID}, "approved_at": approved.ApprovedAt,
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
	if outcome.RunID != run.ID.String() || !outcome.SessionKey || !outcome.Identity || len(outcome.CLICommands) != 11 || len(outcome.SpendGrants) != 4 || outcome.SpendGrants[0] != grant.ID.String() || outcome.SpendGrants[1] != secondSpendGrant.ID.String() || outcome.SpendGrants[2] != frozenGrant.ID.String() || outcome.SpendGrants[3] != expiryGrant.ID.String() {
		t.Fatalf("incomplete Agent Manager CLI proof: %+v", outcome)
	}
	if err := writeJSON(filepath.Join(coord, "agent-manager-result.json"), map[string]any{
		"run_id": run.ID.String(), "profile_id": profile.ID.String(), "task_id": task.ID.String(),
		"run_status": finalRun.Status, "terminal_session_revoked": keys.WasRevoked(run.ID),
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
