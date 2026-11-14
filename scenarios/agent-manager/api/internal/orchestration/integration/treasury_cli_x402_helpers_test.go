//go:build linux

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	agenthandlers "agent-manager/internal/handlers"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/researchbudget"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func executeCF7X402Run(ctx context.Context, db *database.DB, request runner.ExecuteRequest, config cf1TreasuryRunConfig, coord string, write func(string, any) error) error {
	if config.APIBaseURL == "" || config.CLIBinary == "" || config.BudgetID == "" || config.MandateID == "" || len(config.X402Calls) != 4 {
		return errors.New("CF7 requires a run-bound budget, mandate, Treasury CLI and four priced call intents")
	}
	if err := validateCF7X402Calls(config.X402Calls); err != nil {
		return err
	}
	if request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] == "" {
		return errors.New("ordinary CF7 run did not receive its verified identity and run-owned session key")
	}
	env := make([]string, 0, len(os.Environ())+len(request.Environment)+2)
	env = append(env, os.Environ()...)
	for key, value := range request.Environment {
		env = append(env, key+"="+value)
	}
	env = append(env, "API_BASE_URL="+strings.TrimRight(config.APIBaseURL, "/"))
	runCLI := func(label string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, config.CLIBinary, args...)
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if writeErr := os.WriteFile(filepath.Join(coord, label+".log"), output, 0o600); writeErr != nil {
			return output, writeErr
		}
		if err != nil {
			return output, fmt.Errorf("CF7 Treasury CLI %s failed: %w: %s", label, err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	if strings.TrimSpace(request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]) == "" {
		return errors.New("CF7 run identity token is missing for report evidence")
	}
	var budgetRecorded bool
	for index, call := range config.X402Calls {
		if index == 3 {
			if err := executeCF7OverCapProposal(ctx, request.RunID.String(), coord, config, call, runCLI, write); err != nil {
				return err
			}
			continue
		}
		settlementKey := call.IdempotencyKey + "-settlement"
		target, settlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: call.AuthorizationID, MandateID: config.MandateID, AmountMinor: call.AmountMinor, Currency: "USD",
			Counterparty: call.Counterparty, AuthorizationIdempotencyKey: call.IdempotencyKey,
			SettlementID: call.SettlementID, InstrumentID: call.InstrumentID, SettlementIdempotencyKey: settlementKey,
		}).Targets()
		if err := write(fmt.Sprintf("x402-call-%d-operation-grant-requested.json", index+1), map[string]any{
			"run_id": request.RunID.String(), "authorization_id": call.AuthorizationID, "settlement_id": call.SettlementID,
			"amount_minor": call.AmountMinor, "price_id": call.PriceID, "call_id": call.CallID,
			"required_target": target, "required_settlement_target": settlementTarget,
		}); err != nil {
			return err
		}
		var approval struct {
			RunID        string   `json:"run_id"`
			GrantID      string   `json:"grant_id"`
			SpendTargets []string `json:"spend_targets"`
		}
		if err := waitForFile(ctx, filepath.Join(coord, fmt.Sprintf("x402-call-%d-operation-approved.json", index+1)), &approval); err != nil {
			return err
		}
		if approval.RunID != request.RunID.String() || approval.GrantID == "" || len(approval.SpendTargets) != 2 || approval.SpendTargets[0] != target || approval.SpendTargets[1] != settlementTarget {
			return fmt.Errorf("CF7 call %d is missing its own exact run-bound Agent Manager grant", index+1)
		}
		if !budgetRecorded {
			// Treasury's own headroom response is the cap snapshot; no amount
			// supplied by the Agent Manager task is used as settlement evidence.
			headroom, err := runCLI("cf7-budget-headroom-before", "--api-base", config.APIBaseURL, "budget", "headroom", config.BudgetID, "--json")
			if err != nil {
				return err
			}
			var headroomResult struct {
				Headroom struct {
					BudgetID            string `json:"budget_id"`
					TotalCapMinor       string `json:"total_cap_minor"`
					TotalRemainingMinor string `json:"total_remaining_minor"`
				} `json:"headroom"`
			}
			if err := json.Unmarshal(headroom, &headroomResult); err != nil {
				return fmt.Errorf("decode Treasury headroom snapshot: %w", err)
			}
			capMinor, err := strconv.ParseInt(headroomResult.Headroom.TotalCapMinor, 10, 64)
			if err != nil || headroomResult.Headroom.BudgetID != config.BudgetID || capMinor != 1000 {
				return fmt.Errorf("CF7 run budget snapshot must be the Treasury-reported 1000-minor-unit cap: %+v", headroomResult.Headroom)
			}
			if err := postCF7ResearchBudgetEvidence(ctx, db, request.RunID, map[string]any{
				"treasury_budget_id": config.BudgetID, "mandate_id": config.MandateID,
				"agent_subject": "agent:agent-manager-runner", "cap_minor": capMinor, "currency": "USD",
				"unit_code": "USD_MINOR", "policy_revision": "treasury-local-fork-per-transaction-v1", "created_at": time.Now().UTC(),
			}, "snapshot"); err != nil {
				return fmt.Errorf("submit Treasury-reported CF7 budget snapshot through the Treasury-only route: %w", err)
			}
			budgetRecorded = true
		}
		if _, err := runCLI(fmt.Sprintf("cf7-call-%d-authorization-propose", index+1), "--api-base", config.APIBaseURL, "authorization", "propose",
			"--id", call.AuthorizationID, "--idempotency-key", call.IdempotencyKey,
			"--mandate-id", config.MandateID, "--amount-minor", strconv.FormatInt(call.AmountMinor, 10),
			"--currency", "USD", "--counterparty", call.Counterparty,
			"--settlement-id", call.SettlementID, "--instrument-id", call.InstrumentID,
			"--settlement-idempotency-key", settlementKey, "--json"); err != nil {
			return err
		}
		if err := write(fmt.Sprintf("x402-call-%d-authorization-ready.json", index+1), map[string]any{
			"run_id": request.RunID.String(), "authorization_id": call.AuthorizationID, "settlement_id": call.SettlementID,
			"price_id": call.PriceID, "call_id": call.CallID, "amount_minor": call.AmountMinor,
		}); err != nil {
			return err
		}
		if err := waitForFile(ctx, filepath.Join(coord, fmt.Sprintf("x402-call-%d-release", index+1)), nil); err != nil {
			return err
		}
		if _, err := runCLI(fmt.Sprintf("cf7-call-%d-settlement-report", index+1), "--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", call.AuthorizationID, "--settlement-id", call.SettlementID,
			"--instrument-id", call.InstrumentID, "--idempotency-key", settlementKey, "--json"); err != nil {
			return err
		}
		output, err := os.ReadFile(filepath.Join(coord, fmt.Sprintf("cf7-call-%d-settlement-report.log", index+1)))
		if err != nil {
			return err
		}
		var receipt cf7X402SettlementReceipt
		if err := json.Unmarshal(output, &receipt); err != nil || receipt.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || receipt.Settlement.Authorization != call.AuthorizationID || receipt.Settlement.Id != call.SettlementID || receipt.Settlement.AmountMinor != call.AmountMinor || !strings.HasPrefix(strings.ToLower(receipt.Settlement.ExternalID), "0x") || receipt.Settlement.Currency != "USD" || receipt.Settlement.InstrumentID != call.InstrumentID {
			return fmt.Errorf("CF7 call %d settled response did not retain the exact Treasury receipt: receipt=%+v decode=%v", index+1, receipt, err)
		}
		settledAt, err := time.Parse(time.RFC3339Nano, receipt.Settlement.SettledAt)
		if err != nil {
			return fmt.Errorf("CF7 call %d Treasury settlement has no authoritative settled-at timestamp: %w", index+1, err)
		}
		catalogResult, err := observeCF7Catalog(ctx, call.CatalogURL)
		if err != nil {
			return fmt.Errorf("CF7 call %d has no measured Offer Desk result after settlement: %w", index+1, err)
		}
		if !strings.Contains(receipt.Settlement.Detail, fmt.Sprintf("response_bytes=%d", catalogResult.BodyBytes)) || !strings.Contains(receipt.Settlement.Detail, "response_sha256="+strings.TrimPrefix(catalogResult.Digest, "sha256:")) {
			return fmt.Errorf("CF7 call %d Treasury settlement lacks matching bounded evidence for the paid Offer Desk response: detail=%q expected_bytes=%d expected_digest=%s", index+1, receipt.Settlement.Detail, catalogResult.BodyBytes, catalogResult.Digest)
		}
		resultRef := fmt.Sprintf("run:%s/x402-call-%d/%s", request.RunID.String(), index+1, catalogResult.Digest)
		benefitSummary := fmt.Sprintf("Offer Desk paid response was %d bytes and returned %d graph records; Treasury recorded response digest %s.", catalogResult.BodyBytes, catalogResult.NodeCount, catalogResult.Digest)
		if err := postCF7ResearchBudgetEvidence(ctx, db, request.RunID, map[string]any{
			"id": fmt.Sprintf("cf7-%s", call.SettlementID), "settlement_id": receipt.Settlement.Id,
			"treasury_receipt_id": receipt.Settlement.ExternalID, "integration_id": "offer-desk:" + call.PriceID,
			"category": "x402_call", "amount_minor": receipt.Settlement.AmountMinor, "currency": receipt.Settlement.Currency,
			"unit_code": "USD_MINOR", "result_ref": resultRef,
			"benefit_state": "measured", "benefit_summary": benefitSummary, "settled_at": settledAt,
		}, "settlements"); err != nil {
			return fmt.Errorf("submit receipt-backed CF7 settlement %d through the Treasury-only route: %w", index+1, err)
		}
		if err := write(fmt.Sprintf("cf7-call-%d-benefit.json", index+1), map[string]any{
			"result_ref": resultRef, "benefit_state": "measured", "benefit_summary": benefitSummary,
			"catalog_node_count": catalogResult.NodeCount, "catalog_node_names": catalogResult.Names, "catalog_body_bytes": catalogResult.BodyBytes, "catalog_digest": catalogResult.Digest,
		}); err != nil {
			return err
		}
		if err := write(fmt.Sprintf("x402-call-%d-settled.json", index+1), map[string]any{
			"run_id": request.RunID.String(), "authorization_id": receipt.Settlement.Authorization,
			"settlement_id": receipt.Settlement.Id, "treasury_receipt_id": receipt.Settlement.ExternalID,
			"amount_minor": receipt.Settlement.AmountMinor, "currency": receipt.Settlement.Currency,
			"instrument_id": receipt.Settlement.InstrumentID, "settled_at": receipt.Settlement.SettledAt,
			"price_id": call.PriceID, "call_id": call.CallID, "operation_grant_id": approval.GrantID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func executeCF7OverCapProposal(ctx context.Context, runID, coord string, config cf1TreasuryRunConfig, call cf1TreasuryX402Call, runCLI func(string, ...string) ([]byte, error), write func(string, any) error) error {
	if call.AmountMinor != 110 || call.PriceID == config.X402Calls[0].PriceID || call.CallID == config.X402Calls[0].CallID {
		return errors.New("CF7 fourth priced call must be a distinct 110-minor-unit Offer Desk quote")
	}
	settlementKey := call.IdempotencyKey + "-settlement"
	target, settlementTarget := (cliutil.TreasurySpendIntent{
		AuthorizationID: call.AuthorizationID, MandateID: config.MandateID, AmountMinor: call.AmountMinor, Currency: "USD",
		Counterparty: call.Counterparty, AuthorizationIdempotencyKey: call.IdempotencyKey,
		SettlementID: call.SettlementID, InstrumentID: call.InstrumentID, SettlementIdempotencyKey: settlementKey,
	}).Targets()
	if err := write("x402-call-4-operation-grant-requested.json", map[string]any{
		"run_id": runID, "authorization_id": call.AuthorizationID, "settlement_id": call.SettlementID, "amount_minor": call.AmountMinor,
		"price_id": call.PriceID, "call_id": call.CallID, "required_target": target, "required_settlement_target": settlementTarget,
	}); err != nil {
		return err
	}
	var approved struct {
		RunID        string   `json:"run_id"`
		GrantID      string   `json:"grant_id"`
		SpendTargets []string `json:"spend_targets"`
	}
	if err := waitForFile(ctx, filepath.Join(coord, "x402-call-4-operation-approved.json"), &approved); err != nil {
		return err
	}
	if approved.RunID != runID || approved.GrantID == "" || len(approved.SpendTargets) != 2 || approved.SpendTargets[0] != target || approved.SpendTargets[1] != settlementTarget {
		return errors.New("CF7 fourth call is missing its exact Agent Manager operation grant")
	}
	output, err := runCLI("cf7-call-4-authorization-propose", "--api-base", config.APIBaseURL, "authorization", "propose",
		"--id", call.AuthorizationID, "--idempotency-key", call.IdempotencyKey,
		"--mandate-id", config.MandateID, "--amount-minor", strconv.FormatInt(call.AmountMinor, 10),
		"--currency", "USD", "--counterparty", call.Counterparty,
		"--settlement-id", call.SettlementID, "--instrument-id", call.InstrumentID,
		"--settlement-idempotency-key", settlementKey, "--json")
	if err != nil {
		return err
	}
	var proposal struct {
		Authorization struct {
			ID          string `json:"id"`
			AmountMinor int64  `json:"amount_minor"`
			Verdict     string `json:"verdict"`
		} `json:"authorization"`
	}
	if err := json.Unmarshal(output, &proposal); err != nil || proposal.Authorization.ID != call.AuthorizationID || proposal.Authorization.AmountMinor != 110 || proposal.Authorization.Verdict != "AUTHORIZATION_VERDICT_PENDING" {
		return fmt.Errorf("CF7 fourth call did not stop at a pending Treasury decision: authorization=%+v decode=%v", proposal.Authorization, err)
	}
	return write("x402-call-4-decision-raised.json", map[string]any{
		"authorization_id": proposal.Authorization.ID, "settlement_id": call.SettlementID,
		"price_id": call.PriceID, "call_id": call.CallID, "amount_minor": proposal.Authorization.AmountMinor,
		"verdict": proposal.Authorization.Verdict, "dispatch": "none", "call_release_written": false, "operation_grant_id": approved.GrantID,
	})
}

func validateCF7X402Calls(calls []cf1TreasuryX402Call) error {
	if len(calls) != 4 {
		return errors.New("CF7 requires exactly three priced calls and one over-cap quote")
	}
	seen := map[string]bool{}
	for i, call := range calls {
		if call.PriceID == "" || call.CallID == "" || call.CatalogURL == "" || call.Counterparty == "" || call.InstrumentID == "" || call.AuthorizationID == "" || call.SettlementID == "" || call.IdempotencyKey == "" {
			return fmt.Errorf("CF7 call %d has incomplete Offer Desk or Treasury intent identity", i+1)
		}
		amount := int64(1)
		if i == 3 {
			amount = 110
		}
		if call.AmountMinor != amount {
			return fmt.Errorf("CF7 call %d amount is %d; expected %d minor units", i+1, call.AmountMinor, amount)
		}
		for _, value := range []string{call.AuthorizationID, call.SettlementID, call.IdempotencyKey} {
			if seen[value] {
				return fmt.Errorf("CF7 call identity %q is reused", value)
			}
			seen[value] = true
		}
		if i < 3 && (call.PriceID != calls[0].PriceID || call.CatalogURL != calls[0].CatalogURL || !strings.EqualFold(call.Counterparty, calls[0].Counterparty)) {
			return errors.New("the first three CF7 calls must use the same Offer Desk price and endpoint")
		}
	}
	if calls[3].PriceID == calls[0].PriceID || calls[3].CallID == calls[0].CallID || calls[3].CatalogURL == calls[0].CatalogURL {
		return errors.New("CF7 fourth call must use a distinct Offer Desk price, call and endpoint")
	}
	return nil
}

type cf7ResearchBudgetAuthority struct{ runID uuid.UUID }

func (a cf7ResearchBudgetAuthority) AuthorizeOperation(_ context.Context, principal coreidentity.Principal, capability, operation, target string) error {
	if principal.Kind != coreidentity.ActorService || principal.Subject != "treasury" || !principal.IsVerified() || capability != "agent-manager:write" || operation != "research-budget.ingest" || target != "agent-manager:run/"+a.runID.String() {
		return errors.New("CF7 test Treasury grant does not cover the exact run-scoped evidence operation")
	}
	return nil
}

// postCF7ResearchBudgetEvidence composes the new Treasury-only API boundary
// with a verified local service identity and an exact run-scoped operation
// grant. Callers pass only cap data from Treasury headroom or settlement data
// parsed from Treasury's settled response.
func postCF7ResearchBudgetEvidence(ctx context.Context, db *database.DB, runID uuid.UUID, body any, resource string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	router := mux.NewRouter()
	agenthandlers.RegisterResearchBudgetIngestion(router, researchbudget.Ledger{DB: db.DB})
	service := coreidentity.Principal{
		Kind: coreidentity.ActorService, Subject: "treasury", Verified: true,
		Scopes: []string{"agent-manager:write"},
		Grant: &coreidentity.OperationGrant{
			ID: "cf7-local-treasury-ingest-" + runID.String(), Approver: "cf7-local-operator",
			Operations: []string{"research-budget.ingest"}, Targets: []string{"agent-manager:run/" + runID.String()},
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://agent-manager/api/v1/research-budgets/"+runID.String()+"/"+resource, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	requestCtx := coreidentity.WithPrincipal(req.Context(), service)
	requestCtx = authn.WithRequestOperationAuthority(requestCtx, cf7ResearchBudgetAuthority{runID: runID})
	req = req.WithContext(requestCtx)
	recorder := httptest.NewRecorder()
	agenthandlers.AgentOperationMiddleware(router).ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNoContent {
		return fmt.Errorf("Treasury-only research budget route returned %d: %s", recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	return nil
}

func coordinateCF7X402Run(ctx context.Context, db *database.DB, service *orchestration.Orchestrator, ownerCtx context.Context, run *domain.Run, coord string, config cf1TreasuryRunConfig, t *testing.T) {
	t.Helper()
	if err := validateCF7X402Calls(config.X402Calls); err != nil {
		t.Fatal(err)
	}
	if config.BudgetID == "" || config.MandateID == "" || config.X402Counterparty == "" {
		t.Fatal("CF7 config must bind the new ordinary run to its Treasury budget, mandate and counterparty")
	}
	budgetTarget := "treasury:budget/" + config.BudgetID
	grantIDs := make(map[string]bool, len(config.X402Calls))
	grantByCall := make(map[string]string, len(config.X402Calls))
	for i, call := range config.X402Calls {
		index := i + 1
		var requested struct {
			RunID                    string `json:"run_id"`
			AuthorizationID          string `json:"authorization_id"`
			SettlementID             string `json:"settlement_id"`
			AmountMinor              int64  `json:"amount_minor"`
			PriceID                  string `json:"price_id"`
			CallID                   string `json:"call_id"`
			RequiredTarget           string `json:"required_target"`
			RequiredSettlementTarget string `json:"required_settlement_target"`
		}
		requestPath := filepath.Join(coord, fmt.Sprintf("x402-call-%d-operation-grant-requested.json", index))
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, requestPath, &requested); err != nil {
			t.Fatalf("CF7 call %d did not request its per-call grant: %v", index, err)
		}
		settlementKey := call.IdempotencyKey + "-settlement"
		target, settlementTarget := (cliutil.TreasurySpendIntent{
			AuthorizationID: call.AuthorizationID, MandateID: config.MandateID, AmountMinor: call.AmountMinor, Currency: "USD",
			Counterparty: call.Counterparty, AuthorizationIdempotencyKey: call.IdempotencyKey,
			SettlementID: call.SettlementID, InstrumentID: call.InstrumentID, SettlementIdempotencyKey: settlementKey,
		}).Targets()
		if requested.RunID != run.ID.String() || requested.AuthorizationID != call.AuthorizationID || requested.SettlementID != call.SettlementID || requested.AmountMinor != call.AmountMinor || requested.PriceID != call.PriceID || requested.CallID != call.CallID || requested.RequiredTarget != target || requested.RequiredSettlementTarget != settlementTarget {
			t.Fatalf("CF7 call %d grant request is not bound to its exact run and priced intent: %+v", index, requested)
		}
		grant, err := service.RequestAuthorization(ctx, orchestration.AuthorizationRequest{
			RunID: run.ID, PID: os.Getpid(), HarnessKind: "cf7-local-fork", HarnessSession: "research-budget-showcase",
			Rules: []domain.AuthorizationRule{
				{Policy: "treasury-spending", Targets: []string{target, settlementTarget}},
				{Policy: "treasury-wallet-reading", Targets: []string{budgetTarget, "treasury:authorization/" + call.AuthorizationID}},
			}, DurationSeconds: 3600,
			BigPicture:       "Authorize one exact priced x402 research call from this Agent Manager run.",
			TechnicalDetails: fmt.Sprintf("CF7 call %d: %d USD minor units, exact Offer Desk price %s, run %s", index, call.AmountMinor, call.PriceID, run.ID.String()),
		})
		if err != nil {
			t.Fatalf("request CF7 call %d exact operation grant: %v", index, err)
		}
		approved, err := service.ApproveAuthorization(ownerCtx, grant.ID)
		if err != nil {
			t.Fatalf("approve CF7 call %d operation grant: %v", index, err)
		}
		if grantIDs[grant.ID.String()] {
			t.Fatalf("CF7 call %d reused an operation grant", index)
		}
		grantIDs[grant.ID.String()] = true
		grantByCall[call.AuthorizationID] = grant.ID.String()
		if err := writeJSON(filepath.Join(coord, fmt.Sprintf("x402-call-%d-operation-approved.json", index)), map[string]any{
			"run_id": run.ID.String(), "grant_id": grant.ID.String(), "authorization_id": call.AuthorizationID,
			"spend_targets": []string{target, settlementTarget}, "approved_at": approved.ApprovedAt,
		}); err != nil {
			t.Fatal(err)
		}
		if index == 4 {
			var decision map[string]any
			if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-call-4-decision-raised.json"), &decision); err != nil {
				t.Fatalf("fourth CF7 call did not raise the expected Treasury decision: %v", err)
			}
			if decision["dispatch"] != "none" || decision["call_release_written"] != false || decision["amount_minor"] != float64(110) || decision["operation_grant_id"] != grantByCall[call.AuthorizationID] {
				t.Fatalf("fourth CF7 call was dispatched or lost its over-cap decision: %+v", decision)
			}
			continue
		}
		var ready map[string]any
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, fmt.Sprintf("x402-call-%d-authorization-ready.json", index)), &ready); err != nil {
			t.Fatalf("CF7 call %d Treasury authorization was not proposed: %v", index, err)
		}
		if ready["authorization_id"] != call.AuthorizationID || ready["settlement_id"] != call.SettlementID {
			t.Fatalf("CF7 call %d authorization identity changed: %+v", index, ready)
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, fmt.Sprintf("x402-call-%d-release", index)), nil); err != nil {
			t.Fatalf("CF7 call %d was not released by the Treasury fork harness after payer validation: %v", index, err)
		}
		var settled struct {
			RunID             string `json:"run_id"`
			AuthorizationID   string `json:"authorization_id"`
			SettlementID      string `json:"settlement_id"`
			TreasuryReceiptID string `json:"treasury_receipt_id"`
			AmountMinor       int64  `json:"amount_minor"`
			Currency          string `json:"currency"`
			OperationGrantID  string `json:"operation_grant_id"`
		}
		if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, fmt.Sprintf("x402-call-%d-settled.json", index)), &settled); err != nil {
			t.Fatalf("CF7 call %d did not return a settled Treasury receipt: %v", index, err)
		}
		if settled.RunID != run.ID.String() || settled.AuthorizationID != call.AuthorizationID || settled.SettlementID != call.SettlementID || settled.TreasuryReceiptID == "" || settled.AmountMinor != call.AmountMinor || settled.Currency != "USD" || settled.OperationGrantID != grantByCall[call.AuthorizationID] {
			t.Fatalf("CF7 call %d settlement evidence is not tied to its exact approved intent: %+v", index, settled)
		}
	}
	result, err := service.GetRun(ctx, run.ID)
	if err != nil || result.Status.IsTerminal() == false {
		t.Fatalf("CF7 ordinary run did not complete after its three settled calls and held decision: run=%+v err=%v", result, err)
	}
	if err := waitForFile(ctx, filepath.Join(coord, "run-result.json"), nil); err != nil {
		t.Fatalf("CF7 ordinary-run receipt missing: %v", err)
	}
	budget, err := (researchbudget.Ledger{DB: db.DB}).Read(ctx, run.ID.String())
	if err != nil {
		t.Fatalf("read CF7 receipt-backed research budget report: %v", err)
	}
	if budget.TreasuryBudgetID != config.BudgetID || budget.MandateID != config.MandateID || budget.CapMinor != 1000 || budget.SpendMinor != 3 || budget.RemainingMinor != 997 || budget.SpendCount != 3 {
		t.Fatalf("CF7 research budget report does not reconcile three settled calls against Treasury budget: %+v", budget)
	}
	for _, call := range config.X402Calls[:3] {
		found := false
		for _, spend := range budget.Spend {
			if spend.SettlementID == call.SettlementID {
				found = true
				if spend.AmountMinor != 1 || spend.TreasuryReceiptID == "" || spend.ResultRef == "" || spend.BenefitState != "measured" || !strings.Contains(spend.BenefitSummary, "response digest") {
					t.Fatalf("CF7 spend row is not tied to an authoritative Treasury receipt and result record: %+v", spend)
				}
			}
		}
		if !found {
			t.Fatalf("CF7 research budget report omitted settlement %s", call.SettlementID)
		}
	}
}
