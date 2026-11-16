//go:build linux

package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
	"agent-manager/internal/researchbudget"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

func executeCF7X402Run(ctx context.Context, db *database.DB, service *orchestration.Orchestrator, request runner.ExecuteRequest, config cf1TreasuryRunConfig, coord string, write func(string, any) error) error {
	if config.APIBaseURL == "" || config.CLIBinary == "" || config.BudgetID == "" || config.MandateID == "" || len(config.X402Calls) != 4 {
		return errors.New("CF7 requires a run-bound budget, mandate, Treasury CLI and four priced call intents")
	}
	if err := validateCF7X402Calls(config.X402Calls); err != nil {
		return err
	}
	if service == nil || request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"] == "" || request.Environment["VROOLI_TREASURY_SESSION_PRIVATE_KEY"] != "" || request.PrepareLaunch == nil {
		return errors.New("ordinary CF7 run requires its verified identity and launch-scoped Treasury session-key provider")
	}
	env := make([]string, 0, len(os.Environ())+len(request.Environment)+2)
	env = append(env, os.Environ()...)
	for key, value := range request.Environment {
		env = append(env, key+"="+value)
	}
	env = append(env, "API_BASE_URL="+strings.TrimRight(config.APIBaseURL, "/"))
	runCLI := func(label string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, config.CLIBinary, args...)
		cmd.Env = append([]string(nil), env...)
		var output bytes.Buffer
		cmd.Stdout, cmd.Stderr = &output, &output
		launch := &runner.LaunchRequest{Env: append([]string(nil), cmd.Env...)}
		release, err := request.PrepareLaunch(ctx, launch)
		if err != nil {
			return nil, fmt.Errorf("prepare run-owned CF7 Treasury CLI launch: %w", err)
		}
		cmd.Env = launch.Env
		startErr := cmd.Start()
		if release != nil {
			release()
		}
		if startErr != nil {
			err = startErr
		} else {
			err = cmd.Wait()
		}
		if writeErr := os.WriteFile(filepath.Join(coord, label+".log"), output.Bytes(), 0o600); writeErr != nil {
			return output.Bytes(), writeErr
		}
		if err != nil {
			return output.Bytes(), fmt.Errorf("CF7 Treasury CLI %s failed: %w: %s", label, err, strings.TrimSpace(output.String()))
		}
		return output.Bytes(), nil
	}
	if strings.TrimSpace(request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"]) == "" {
		return errors.New("CF7 run identity token is missing for report evidence")
	}
	for index, call := range config.X402Calls {
		if index == 3 {
			if err := executeCF7OverCapProposal(ctx, request.RunID.String(), coord, config, call, runCLI, write); err != nil {
				return err
			}
			if err := waitForFile(ctx, filepath.Join(coord, "x402-call-4-settle-release"), nil); err != nil {
				return err
			}
			settlementKey := call.IdempotencyKey + "-settlement"
			reportArgs := []string{"--api-base", config.APIBaseURL, "settlement", "report-outcome", "--authorization-id", call.AuthorizationID, "--settlement-id", call.SettlementID, "--instrument-id", call.InstrumentID, "--idempotency-key", settlementKey, "--json"}
			first, reportErr := runCLI("cf7-call-4-settlement-report-first", reportArgs...)
			settlementReportAttempts := 1
			if reportErr != nil {
				if _, approvalErr := resolveCF7ResearchBudgetApproval(ctx, db, service, request.RunID); approvalErr != nil {
					return fmt.Errorf("first approved CF7 over-cap settlement report failed without an exact report-grant decision: report_error=%v approval_error=%w", reportErr, approvalErr)
				}
				first, reportErr = runCLI("cf7-call-4-settlement-report-after-report-approval", reportArgs...)
				settlementReportAttempts++
			}
			if reportErr != nil {
				return fmt.Errorf("settle the approved exact CF7 over-cap call: %w", reportErr)
			}
			var firstReceipt cf7X402SettlementReceipt
			if err := json.Unmarshal(first, &firstReceipt); err != nil || firstReceipt.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || firstReceipt.Settlement.Authorization != call.AuthorizationID || firstReceipt.Settlement.Id != call.SettlementID || firstReceipt.Settlement.AmountMinor != call.AmountMinor || firstReceipt.Settlement.Currency != "USD" || !strings.HasPrefix(strings.ToLower(firstReceipt.Settlement.ExternalID), "0x") {
				return fmt.Errorf("approved CF7 over-cap call did not settle with its exact Treasury receipt: receipt=%+v decode=%v", firstReceipt, err)
			}
			retried, retryErr := runCLI("cf7-call-4-settlement-report-retry", reportArgs...)
			if retryErr != nil {
				return fmt.Errorf("retry approved CF7 over-cap settlement with its original idempotency key: %w", retryErr)
			}
			var retryReceipt cf7X402SettlementReceipt
			if err := json.Unmarshal(retried, &retryReceipt); err != nil || retryReceipt.Settlement.Id != firstReceipt.Settlement.Id || retryReceipt.Settlement.ExternalID != firstReceipt.Settlement.ExternalID || retryReceipt.Settlement.AmountMinor != firstReceipt.Settlement.AmountMinor || retryReceipt.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" {
				return fmt.Errorf("CF7 over-cap retry changed the settled receipt: first=%+v retry=%+v decode=%v", firstReceipt.Settlement, retryReceipt.Settlement, err)
			}
			settlementReportAttempts++
			catalogResult, err := observeCF7Catalog(ctx, call.CatalogURL)
			if err != nil {
				return fmt.Errorf("CF7 approved over-cap call has no measured Offer Desk result: %w", err)
			}
			if !strings.Contains(firstReceipt.Settlement.Detail, fmt.Sprintf("response_bytes=%d", catalogResult.BodyBytes)) || !strings.Contains(firstReceipt.Settlement.Detail, "response_sha256="+strings.TrimPrefix(catalogResult.Digest, "sha256:")) {
				return fmt.Errorf("CF7 over-cap Treasury receipt lacks matching paid-response evidence: detail=%q expected_bytes=%d expected_digest=%s", firstReceipt.Settlement.Detail, catalogResult.BodyBytes, catalogResult.Digest)
			}
			freeAlternative, err := observeCF7FreeAlternative(ctx, call.CatalogURL)
			if err != nil {
				return fmt.Errorf("CF7 over-cap call could not measure its free alternative: %w", err)
			}
			if err := publishCF7FreeAlternative(ctx, config.APIBaseURL, request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], request.RunID.String(), call.SettlementID, freeAlternative); err != nil {
				return fmt.Errorf("persist the CF7 over-cap free comparison against its exact run settlement: %w", err)
			}
			resultRef := fmt.Sprintf("run:%s/x402-call-4/%s", request.RunID.String(), catalogResult.Digest)
			benefitSummary := fmt.Sprintf("Paid Offer Desk response: %d bytes, %d graph records, digest %s. Free alternative: %s returned %d bytes, digest %s; it is a no-signup example only and does not provide the paid endpoint's current full catalog response.", catalogResult.BodyBytes, catalogResult.NodeCount, catalogResult.Digest, freeAlternative.URL, freeAlternative.BodyBytes, freeAlternative.Digest)
			if err := write("cf7-call-4-benefit.json", map[string]any{
				"run_id": request.RunID.String(), "settlement_id": firstReceipt.Settlement.Id,
				"result_ref": resultRef, "benefit_state": "measured", "benefit_summary": benefitSummary,
				"catalog_node_count": catalogResult.NodeCount, "catalog_node_names": catalogResult.Names,
				"catalog_body_bytes": catalogResult.BodyBytes, "catalog_digest": catalogResult.Digest,
				"free_alternative": freeAlternative,
			}); err != nil {
				return err
			}
			approval, err := readCF7TreasuryApprovalReceipt(coord, request.RunID.String(), call)
			if err != nil {
				return err
			}
			if err := write("x402-call-4-settled.json", map[string]any{
				"run_id": request.RunID.String(), "authorization_id": firstReceipt.Settlement.Authorization,
				"settlement_id": firstReceipt.Settlement.Id, "treasury_receipt_id": firstReceipt.Settlement.ExternalID,
				"amount_minor": firstReceipt.Settlement.AmountMinor, "currency": firstReceipt.Settlement.Currency,
				"approval_id": approval.ApprovalID, "approval_source": approval.Source,
				"settlement_report_attempts": settlementReportAttempts, "idempotent_retry_returned_same_receipt": true,
			}); err != nil {
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
		reportArgs := []string{
			"--api-base", config.APIBaseURL, "settlement", "report-outcome",
			"--authorization-id", call.AuthorizationID, "--settlement-id", call.SettlementID,
			"--instrument-id", call.InstrumentID, "--idempotency-key", settlementKey, "--json",
		}
		reportOutput, reportErr := runCLI(fmt.Sprintf("cf7-call-%d-settlement-report-first", index+1), reportArgs...)
		var reportGrant *domain.StandingGrant
		retriedAfterApproval := false
		if reportErr != nil {
			var approvalErr error
			reportGrant, approvalErr = resolveCF7ResearchBudgetApproval(ctx, db, service, request.RunID)
			if approvalErr != nil {
				return fmt.Errorf("Treasury report failed and no exact persisted-run approval could be applied: first_error=%v approval_error=%w", reportErr, approvalErr)
			}
			if err := write("research-budget-ingest-approved.json", map[string]any{
				"run_id": request.RunID.String(), "grant_id": reportGrant.ID, "subject": reportGrant.Subject,
				"operation": "research-budget.ingest", "target": "agent-manager:run/" + request.RunID.String(),
				"approved_by": reportGrant.ApprovedBy, "expires_at": reportGrant.ExpiresAt,
				"resolved_after_native_treasury_publish_was_refused": true,
			}); err != nil {
				return err
			}
			retriedOutput, retryErr := runCLI(fmt.Sprintf("cf7-call-%d-settlement-report", index+1), reportArgs...)
			if retryErr != nil {
				return fmt.Errorf("retry Treasury's exact settlement report after the run-bound approval: %w", retryErr)
			}
			reportOutput = retriedOutput
			retriedAfterApproval = true
		}
		var receipt cf7X402SettlementReceipt
		if err := json.Unmarshal(reportOutput, &receipt); err != nil || receipt.Settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || receipt.Settlement.Authorization != call.AuthorizationID || receipt.Settlement.Id != call.SettlementID || receipt.Settlement.AmountMinor != call.AmountMinor || !strings.HasPrefix(strings.ToLower(receipt.Settlement.ExternalID), "0x") || receipt.Settlement.Currency != "USD" || receipt.Settlement.InstrumentID != call.InstrumentID {
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
		freeAlternative, err := observeCF7FreeAlternative(ctx, call.CatalogURL)
		if err != nil {
			return fmt.Errorf("CF7 call %d could not measure its Offer Desk free alternative: %w", index+1, err)
		}
		if err := publishCF7FreeAlternative(ctx, config.APIBaseURL, request.Environment["VROOLI_AGENT_IDENTITY_TOKEN"], request.RunID.String(), call.SettlementID, freeAlternative); err != nil {
			return fmt.Errorf("persist CF7 call %d free comparison against its exact run settlement: %w", index+1, err)
		}
		resultRef := fmt.Sprintf("run:%s/x402-call-%d/%s", request.RunID.String(), index+1, catalogResult.Digest)
		benefitSummary := fmt.Sprintf("Paid Offer Desk response: %d bytes, %d graph records, digest %s. Free alternative: %s returned %d bytes, digest %s; it is a no-signup example only and does not provide the paid endpoint's current full catalog response.", catalogResult.BodyBytes, catalogResult.NodeCount, catalogResult.Digest, freeAlternative.URL, freeAlternative.BodyBytes, freeAlternative.Digest)
		// Treasury's authenticated AgentSpend.ReportOutcome owns settlement and
		// publishes the receipt-backed record. This run-owned route adds only the
		// immutable free comparator to that exact settlement; it cannot create a
		// second Treasury settlement projection.
		budget, err := (researchbudget.Ledger{DB: db.Routed}).Read(ctx, request.RunID.String())
		if err != nil {
			return fmt.Errorf("read Treasury-published CF7 budget evidence: %w", err)
		}
		matched := false
		for _, spend := range budget.Spend {
			if spend.SettlementID == receipt.Settlement.Id && spend.TreasuryReceiptID == receipt.Settlement.ExternalID && spend.AmountMinor == receipt.Settlement.AmountMinor && spend.BenefitState == "measured" {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("Treasury AgentSpend did not retain the exact CF7 settlement and paid response evidence: summary=%+v", budget)
		}
		matched = false
		for _, spend := range budget.Spend {
			if spend.SettlementID == receipt.Settlement.Id && spend.FreeAlternative != nil && spend.FreeAlternative.SHA256 == strings.TrimPrefix(freeAlternative.Digest, "sha256:") && spend.FreeAlternative.ResponseBytes == int64(freeAlternative.BodyBytes) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("Agent Manager did not read back the exact durable CF7 free comparator for settlement %s", receipt.Settlement.Id)
		}
		if err := write(fmt.Sprintf("cf7-call-%d-benefit.json", index+1), map[string]any{
			"run_id": request.RunID.String(), "settlement_id": receipt.Settlement.Id,
			"result_ref": resultRef, "benefit_state": "measured", "benefit_summary": benefitSummary,
			"catalog_node_count": catalogResult.NodeCount, "catalog_node_names": catalogResult.Names, "catalog_body_bytes": catalogResult.BodyBytes, "catalog_digest": catalogResult.Digest,
			"free_alternative": freeAlternative,
		}); err != nil {
			return err
		}
		if err := write(fmt.Sprintf("x402-call-%d-settled.json", index+1), map[string]any{
			"run_id": request.RunID.String(), "authorization_id": receipt.Settlement.Authorization,
			"settlement_id": receipt.Settlement.Id, "treasury_receipt_id": receipt.Settlement.ExternalID,
			"amount_minor": receipt.Settlement.AmountMinor, "currency": receipt.Settlement.Currency,
			"instrument_id": receipt.Settlement.InstrumentID, "settled_at": settledAt,
			"price_id": call.PriceID, "call_id": call.CallID, "operation_grant_id": approval.GrantID,
			"report_retried_after_approval": retriedAfterApproval,
		}); err != nil {
			return err
		}
	}
	budget, err := (researchbudget.Ledger{DB: db.Routed}).Read(ctx, request.RunID.String())
	if err != nil {
		return fmt.Errorf("read authoritative Treasury budget for CF7 commission checkpoint: %w", err)
	}
	checkpoint, markdown, err := buildCF7CommissionCheckpoint(request.RunID.String(), budget, coord)
	if err != nil {
		return err
	}
	if err := write("commission-checkpoint.json", checkpoint); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(coord, "commission-checkpoint.md"), []byte(markdown), 0o600); err != nil {
		return fmt.Errorf("write CF7 commission checkpoint: %w", err)
	}
	return nil
}

type cf7TreasuryApprovalReceipt struct {
	RunID           string `json:"run_id"`
	AuthorizationID string `json:"authorization_id"`
	SettlementID    string `json:"settlement_id"`
	ApprovalID      string `json:"approval_id"`
	ApprovalEventID string `json:"approval_event_id"`
	AmountMinor     int64  `json:"amount_minor"`
	Decision        string `json:"decision"`
	Source          string `json:"source"`
}

func readCF7TreasuryApprovalReceipt(coord, runID string, call cf1TreasuryX402Call) (cf7TreasuryApprovalReceipt, error) {
	var approval cf7TreasuryApprovalReceipt
	data, err := os.ReadFile(filepath.Join(coord, "x402-call-4-treasury-approved.json"))
	if err != nil {
		return approval, fmt.Errorf("read Treasury's separate CF7 over-cap approval receipt: %w", err)
	}
	if err := json.Unmarshal(data, &approval); err != nil {
		return approval, fmt.Errorf("decode Treasury's separate CF7 over-cap approval receipt: %w", err)
	}
	if approval.RunID != runID || approval.AuthorizationID != call.AuthorizationID || approval.SettlementID != call.SettlementID || approval.ApprovalID == "" || approval.AmountMinor != call.AmountMinor || approval.Decision != "approved" || (approval.Source != "notification-hub" && approval.Source != "treasury-admin") || (approval.Source == "notification-hub" && approval.ApprovalEventID == "") {
		return approval, fmt.Errorf("CF7 over-cap Treasury approval receipt does not match the exact run and priced call: %+v", approval)
	}
	return approval, nil
}

func awaitCF7TreasuryOverCapApproval(ctx context.Context, service *orchestration.Orchestrator, runID uuid.UUID, coord string, call cf1TreasuryX402Call) (cf7TreasuryApprovalReceipt, error) {
	var approval cf7TreasuryApprovalReceipt
	path := filepath.Join(coord, "x402-call-4-treasury-approved.json")
	if err := waitForFileOrRunTerminal(ctx, service, runID, path, &approval); err != nil {
		return approval, err
	}
	validated, err := readCF7TreasuryApprovalReceipt(coord, runID.String(), call)
	if err != nil {
		return approval, err
	}
	return validated, nil
}

type cf7CommissionCheckpoint struct {
	RunID              string                     `json:"run_id"`
	TreasuryBudgetID   string                     `json:"treasury_budget_id"`
	MandateID          string                     `json:"mandate_id"`
	Currency           string                     `json:"currency"`
	UnitCode           string                     `json:"unit_code"`
	CapMinor           int64                      `json:"cap_minor"`
	SettledMinor       int64                      `json:"settled_minor"`
	RemainingMinor     int64                      `json:"remaining_minor"`
	Settlements        []researchbudget.Spend     `json:"settlements"`
	FreeComparisons    []cf7CommissionComparison  `json:"free_comparisons"`
	OverCapDecision    cf7OverCapDecisionEvidence `json:"over_cap_decision"`
	OverCapSettlement  cf7SettledCallEvidence     `json:"over_cap_settlement"`
	ComputeProcurement string                     `json:"compute_procurement"`
	DatasetProcurement string                     `json:"dataset_procurement"`
}

type cf7CommissionComparison struct {
	SettlementID    string             `json:"settlement_id"`
	ReceiptID       string             `json:"treasury_receipt_id"`
	PaidAmountMinor int64              `json:"paid_amount_minor"`
	Currency        string             `json:"currency"`
	PaidResultRef   string             `json:"paid_result_ref"`
	PaidBenefit     string             `json:"paid_benefit_summary"`
	FreeAlternative cf7FreeAlternative `json:"free_alternative"`
}

type cf7OverCapDecisionEvidence struct {
	AuthorizationID string `json:"authorization_id"`
	SettlementID    string `json:"settlement_id"`
	PriceID         string `json:"price_id"`
	CallID          string `json:"call_id"`
	AmountMinor     int64  `json:"amount_minor"`
	Verdict         string `json:"verdict"`
	Dispatch        string `json:"dispatch"`
	CallRelease     bool   `json:"call_release_written"`
	OperationGrant  string `json:"operation_grant_id"`
}

type cf7SettledCallEvidence struct {
	RunID                    string `json:"run_id"`
	AuthorizationID          string `json:"authorization_id"`
	SettlementID             string `json:"settlement_id"`
	TreasuryReceiptID        string `json:"treasury_receipt_id"`
	AmountMinor              int64  `json:"amount_minor"`
	Currency                 string `json:"currency"`
	ApprovalID               string `json:"approval_id"`
	ApprovalSource           string `json:"approval_source"`
	SettlementReportAttempts int    `json:"settlement_report_attempts"`
	RetrySameReceipt         bool   `json:"idempotent_retry_returned_same_receipt"`
}

// buildCF7CommissionCheckpoint joins Treasury's immutable run-budget readback
// with the bounded free-option observations captured for each completed call.
// Non-integrated suppliers remain explicit instead of disappearing as zero cost.
func buildCF7CommissionCheckpoint(runID string, budget researchbudget.Summary, coord string) (cf7CommissionCheckpoint, string, error) {
	var result cf7CommissionCheckpoint
	if runID == "" || budget.RunID != runID || budget.TreasuryBudgetID == "" || budget.MandateID == "" || budget.SpendMinor < 0 || budget.RemainingMinor != budget.CapMinor-budget.SpendMinor {
		return result, "", errors.New("CF7 commission checkpoint requires a matching, internally consistent Treasury budget readback")
	}
	result = cf7CommissionCheckpoint{
		RunID: runID, TreasuryBudgetID: budget.TreasuryBudgetID, MandateID: budget.MandateID,
		Currency: budget.Currency, UnitCode: budget.UnitCode, CapMinor: budget.CapMinor,
		SettledMinor: budget.SpendMinor, RemainingMinor: budget.RemainingMinor,
		Settlements:        append([]researchbudget.Spend(nil), budget.Spend...),
		ComputeProcurement: "not_run: CI4/CI9 run-bound supplier contract is not connected",
		DatasetProcurement: "not_run: no run-bound dataset procurement owner/API is connected",
	}
	comparators := make(map[string]cf7FreeAlternative, 4)
	for i := 1; i <= 4; i++ {
		var evidence struct {
			RunID           string             `json:"run_id"`
			SettlementID    string             `json:"settlement_id"`
			FreeAlternative cf7FreeAlternative `json:"free_alternative"`
		}
		path := filepath.Join(coord, fmt.Sprintf("cf7-call-%d-benefit.json", i))
		data, err := os.ReadFile(path)
		if err != nil {
			return result, "", fmt.Errorf("read CF7 free comparator artifact %d: %w", i, err)
		}
		if err := json.Unmarshal(data, &evidence); err != nil {
			return result, "", fmt.Errorf("decode CF7 free comparator artifact %d: %w", i, err)
		}
		if evidence.RunID != runID || evidence.SettlementID == "" || evidence.FreeAlternative.Digest == "" || evidence.FreeAlternative.BodyBytes <= 0 || evidence.FreeAlternative.SourceID == "" || evidence.FreeAlternative.Label == "" || evidence.FreeAlternative.ResultRef == "" || evidence.FreeAlternative.ObservedAt.IsZero() {
			return result, "", fmt.Errorf("CF7 free comparator artifact %d is incomplete or belongs to another run: %+v", i, evidence)
		}
		if _, duplicate := comparators[evidence.SettlementID]; duplicate {
			return result, "", fmt.Errorf("CF7 comparator artifacts reuse settlement %s", evidence.SettlementID)
		}
		comparators[evidence.SettlementID] = evidence.FreeAlternative
	}
	for _, spend := range budget.Spend {
		if spend.Category != "x402_call" {
			return result, "", fmt.Errorf("CF7 settlement %s has category %q but this checkpoint has no owner-bound comparator", spend.SettlementID, spend.Category)
		}
		artifactComparison, ok := comparators[spend.SettlementID]
		if !ok {
			return result, "", fmt.Errorf("CF7 free comparator is not bound to Treasury settlement %s", spend.SettlementID)
		}
		comparison := spend.FreeAlternative
		if comparison == nil || comparison.RunID != runID || comparison.SettlementID != spend.SettlementID || comparison.SourceID != artifactComparison.SourceID || comparison.Label != artifactComparison.Label || comparison.ResultRef != artifactComparison.ResultRef || comparison.SHA256 != strings.TrimPrefix(artifactComparison.Digest, "sha256:") || comparison.ResponseBytes != int64(artifactComparison.BodyBytes) || comparison.Limitation != artifactComparison.Comparison || !comparison.ObservedAt.Equal(artifactComparison.ObservedAt) {
			return result, "", fmt.Errorf("CF7 settlement %s is missing its exact durable free-alternative observation", spend.SettlementID)
		}
		result.FreeComparisons = append(result.FreeComparisons, cf7CommissionComparison{
			SettlementID: spend.SettlementID, ReceiptID: spend.TreasuryReceiptID,
			PaidAmountMinor: spend.AmountMinor, Currency: spend.Currency,
			PaidResultRef: spend.ResultRef, PaidBenefit: spend.BenefitSummary,
			FreeAlternative: cf7FreeAlternative{URL: artifactComparison.URL, SourceID: comparison.SourceID, Label: comparison.Label, ResultRef: comparison.ResultRef, ObservedAt: comparison.ObservedAt, BodyBytes: int(comparison.ResponseBytes), Digest: "sha256:" + comparison.SHA256, Comparison: comparison.Limitation},
		})
	}
	if result.FreeComparisons == nil {
		result.FreeComparisons = []cf7CommissionComparison{}
	}
	decisionData, err := os.ReadFile(filepath.Join(coord, "x402-call-4-decision-raised.json"))
	if err != nil {
		return result, "", fmt.Errorf("read CF7 pending over-cap decision evidence: %w", err)
	}
	if err := json.Unmarshal(decisionData, &result.OverCapDecision); err != nil {
		return result, "", fmt.Errorf("read CF7 pending over-cap decision evidence: %w", err)
	}
	if result.OverCapDecision.AuthorizationID == "" || result.OverCapDecision.SettlementID == "" || result.OverCapDecision.AmountMinor != 110 || result.OverCapDecision.Verdict != "AUTHORIZATION_VERDICT_PENDING" || result.OverCapDecision.Dispatch != "none" || result.OverCapDecision.CallRelease || result.OverCapDecision.OperationGrant == "" {
		return result, "", fmt.Errorf("CF7 fourth-call decision evidence is not the exact undispatched pending quote: %+v", result.OverCapDecision)
	}
	settledData, err := os.ReadFile(filepath.Join(coord, "x402-call-4-settled.json"))
	if err != nil {
		return result, "", fmt.Errorf("read CF7 approved over-cap settlement evidence: %w", err)
	}
	if err := json.Unmarshal(settledData, &result.OverCapSettlement); err != nil {
		return result, "", fmt.Errorf("decode CF7 over-cap settlement evidence: %w", err)
	}
	if result.OverCapSettlement.RunID != runID || result.OverCapSettlement.AuthorizationID != result.OverCapDecision.AuthorizationID || result.OverCapSettlement.SettlementID != result.OverCapDecision.SettlementID || result.OverCapSettlement.TreasuryReceiptID == "" || result.OverCapSettlement.AmountMinor != result.OverCapDecision.AmountMinor || result.OverCapSettlement.Currency != budget.Currency || result.OverCapSettlement.ApprovalID == "" || result.OverCapSettlement.ApprovalSource == "" || result.OverCapSettlement.SettlementReportAttempts < 2 || !result.OverCapSettlement.RetrySameReceipt {
		return result, "", fmt.Errorf("CF7 over-cap settlement evidence is incomplete or not idempotent: %+v", result.OverCapSettlement)
	}
	approval, err := readCF7TreasuryApprovalReceipt(coord, runID, cf1TreasuryX402Call{AuthorizationID: result.OverCapDecision.AuthorizationID, SettlementID: result.OverCapDecision.SettlementID, AmountMinor: result.OverCapDecision.AmountMinor})
	if err != nil {
		return result, "", fmt.Errorf("read the exact Treasury approval receipt for the CF7 settlement: %w", err)
	}
	if result.OverCapSettlement.ApprovalID != approval.ApprovalID || result.OverCapSettlement.ApprovalSource != approval.Source {
		return result, "", fmt.Errorf("CF7 settlement does not retain its exact Treasury approval receipt: approval=%+v settlement=%+v", approval, result.OverCapSettlement)
	}
	matchedOverCapSettlement := false
	for _, spend := range budget.Spend {
		if spend.RunID == runID && spend.SettlementID == result.OverCapSettlement.SettlementID && spend.TreasuryReceiptID == result.OverCapSettlement.TreasuryReceiptID && spend.AmountMinor == result.OverCapSettlement.AmountMinor && spend.Currency == result.OverCapSettlement.Currency {
			matchedOverCapSettlement = true
			break
		}
	}
	if !matchedOverCapSettlement {
		return result, "", fmt.Errorf("approved CF7 over-cap receipt does not appear in Treasury readback: %+v", result.OverCapSettlement)
	}
	var markdown strings.Builder
	fmt.Fprintf(&markdown, "# CF7 commission checkpoint\n\nRun: `%s`\nTreasury budget: `%s` · mandate: `%s`\n\n", runID, budget.TreasuryBudgetID, budget.MandateID)
	fmt.Fprintf(&markdown, "Treasury readback records %d settled item(s), %d %s spent against a cap of %d %s; %d %s remains.\n\n", budget.SpendCount, budget.SpendMinor, budget.Currency, budget.CapMinor, budget.Currency, budget.RemainingMinor, budget.Currency)
	for _, spend := range budget.Spend {
		fmt.Fprintf(&markdown, "- `%s` / `%s`: %d %s, receipt `%s`, result `%s`; benefit evidence: %s\n", spend.Category, spend.IntegrationID, spend.AmountMinor, spend.Currency, spend.TreasuryReceiptID, spend.ResultRef, spend.BenefitSummary)
	}
	fmt.Fprintf(&markdown, "\n## Paid results and free alternatives\n\n")
	for _, comparison := range result.FreeComparisons {
		fmt.Fprintf(&markdown, "- Treasury settlement `%s` / receipt `%s`: paid %d %s, result `%s` (%s). Free alternative: %d bytes, `%s` (%s).\n",
			comparison.SettlementID, comparison.ReceiptID, comparison.PaidAmountMinor, comparison.Currency,
			comparison.PaidResultRef, comparison.PaidBenefit, comparison.FreeAlternative.BodyBytes,
			comparison.FreeAlternative.Digest, comparison.FreeAlternative.Comparison)
	}
	decisionJSON, err := json.MarshalIndent(result.OverCapDecision, "", "  ")
	if err != nil {
		return result, "", fmt.Errorf("encode CF7 pending over-cap decision for the checkpoint: %w", err)
	}
	settlementJSON, err := json.MarshalIndent(result.OverCapSettlement, "", "  ")
	if err != nil {
		return result, "", fmt.Errorf("encode CF7 approved over-cap settlement for the checkpoint: %w", err)
	}
	fmt.Fprintf(&markdown, "\n## Decision and unconnected procurement\n\nOver-cap request (pending before approval, no dispatch):\n\n```json\n%s\n```\n\nApproved settlement and retry:\n\n```json\n%s\n```\n\n- Compute: %s\n- Dataset: %s\n", decisionJSON, settlementJSON, result.ComputeProcurement, result.DatasetProcurement)
	return result, markdown.String(), nil
}

type cf7FreeAlternative struct {
	URL        string    `json:"url"`
	SourceID   string    `json:"source_id"`
	Label      string    `json:"label"`
	ResultRef  string    `json:"result_ref"`
	ObservedAt time.Time `json:"observed_at"`
	BodyBytes  int       `json:"body_bytes"`
	Digest     string    `json:"sha256"`
	Comparison string    `json:"comparison"`
}

// observeCF7FreeAlternative measures Offer Desk's checked-in no-signup sample
// on the same local service host as the paid catalog endpoint. It is a usable
// preview of the offer's output shape, not an equivalent source of current
// catalog data; the distinction is retained in the report.
func observeCF7FreeAlternative(ctx context.Context, paidURL string) (cf7FreeAlternative, error) {
	var result cf7FreeAlternative
	parsed, err := url.ParseRequestURI(paidURL)
	if err != nil || parsed.Scheme != "http" || parsed.Hostname() == "" || parsed.User != nil {
		return result, errors.New("paid endpoint must be an HTTP URL on the local Offer Desk service")
	}
	ip := net.ParseIP(parsed.Hostname())
	if parsed.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return result, errors.New("free alternative fetch is restricted to loopback Offer Desk endpoints")
	}
	parsed.Path = "/api/v1/proposal-preview/service-offer/sample"
	parsed.RawQuery, parsed.Fragment = "", ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return result, err
	}
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return result, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("Offer Desk free sample returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil {
		return result, err
	}
	if len(data) == 0 || len(data) > 64<<10 {
		return result, errors.New("Offer Desk free sample is empty or exceeds 64 KiB")
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	return cf7FreeAlternative{URL: parsed.String(), SourceID: "offer-desk:free-sample", Label: "no-signup sample endpoint", ResultRef: "sha256:" + digest, ObservedAt: time.Now().UTC(), BodyBytes: len(data), Digest: "sha256:" + digest, Comparison: "no-signup sample; demonstrates output shape but does not include the paid endpoint's current full catalog response"}, nil
}

func publishCF7FreeAlternative(ctx context.Context, apiBaseURL, identityToken, runID, settlementID string, alternative cf7FreeAlternative) error {
	parsedBase, err := url.ParseRequestURI(strings.TrimSpace(apiBaseURL))
	if err != nil || (parsedBase.Scheme != "http" && parsedBase.Scheme != "https") || parsedBase.Host == "" || parsedBase.User != nil {
		return errors.New("Agent Manager API base URL must be an HTTP(S) origin")
	}
	parsedRunID, err := uuid.Parse(strings.TrimSpace(runID))
	if err != nil || strings.TrimSpace(settlementID) == "" {
		return errors.New("valid run and settlement identities are required")
	}
	if strings.TrimSpace(identityToken) == "" || alternative.SourceID == "" || alternative.Label == "" || alternative.ResultRef == "" || alternative.BodyBytes <= 0 || alternative.BodyBytes > 64<<10 || alternative.ObservedAt.IsZero() || alternative.Comparison == "" {
		return errors.New("verified run identity and bounded free-alternative observation are required")
	}
	digest := strings.TrimPrefix(alternative.Digest, "sha256:")
	if len(digest) != 64 {
		return errors.New("free-alternative digest must be a SHA-256 value")
	}
	body, err := json.Marshal(map[string]any{
		"source_id": alternative.SourceID, "label": alternative.Label, "result_ref": alternative.ResultRef,
		"sha256": digest, "response_bytes": alternative.BodyBytes, "limitation": alternative.Comparison,
		"observed_at": alternative.ObservedAt,
	})
	if err != nil {
		return fmt.Errorf("encode free-alternative evidence: %w", err)
	}
	path := "/api/v1/research-budgets/" + parsedRunID.String() + "/settlements/" + url.PathEscape(strings.TrimSpace(settlementID)) + "/free-alternative"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsedBase.String(), "/")+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Agent Manager free-alternative request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(identityToken))
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("post free-alternative evidence to Agent Manager: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("Agent Manager free-alternative route returned %d: %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	return nil
}

func TestObserveCF7FreeAlternativeMeasuresOnlyLocalOfferDeskSample(t *testing.T) {
	const sample = "public sample with limitations"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/proposal-preview/service-offer/sample" {
			t.Errorf("unexpected free sample request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, sample)
	}))
	defer server.Close()
	got, err := observeCF7FreeAlternative(context.Background(), server.URL+"/api/v1/paid/catalog?price=local")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(sample))
	if got.BodyBytes != len(sample) || got.Digest != "sha256:"+hex.EncodeToString(sum[:]) || !strings.Contains(got.Comparison, "does not include") {
		t.Fatalf("free sample evidence = %+v", got)
	}
	if _, err := observeCF7FreeAlternative(context.Background(), "http://example.invalid/api/v1/paid/catalog"); err == nil {
		t.Fatal("free alternative must not fetch outside the local Offer Desk host")
	}
}

func TestPublishCF7FreeAlternativeUsesVerifiedRunAndExactSettlementRoute(t *testing.T) {
	runID := uuid.New()
	observedAt := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	var path, authorization string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, authorization = r.URL.Path, r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	alternative := cf7FreeAlternative{SourceID: "offer-desk:free-sample", Label: "no-signup sample", ResultRef: "free-result", ObservedAt: observedAt, BodyBytes: 81, Digest: "sha256:" + strings.Repeat("a", 64), Comparison: "shape sample only"}
	if err := publishCF7FreeAlternative(context.Background(), server.URL, "run-token", runID.String(), "settlement-1", alternative); err != nil {
		t.Fatal(err)
	}
	if path != "/api/v1/research-budgets/"+runID.String()+"/settlements/settlement-1/free-alternative" || authorization != "Bearer run-token" {
		t.Fatalf("request was not bound to the exact run settlement: path=%s authorization=%s", path, authorization)
	}
	if body["source_id"] != alternative.SourceID || body["result_ref"] != alternative.ResultRef || body["sha256"] != strings.Repeat("a", 64) || body["response_bytes"] != float64(81) || body["limitation"] != alternative.Comparison {
		t.Fatalf("free evidence payload changed: %+v", body)
	}
}

func TestCF7CommissionCheckpointUsesTreasuryReadbackAndMarksMissingSuppliers(t *testing.T) {
	dir := t.TempDir()
	spendBySettlement := make(map[string]researchbudget.Spend, 4)
	for i := 1; i <= 4; i++ {
		settlementID := fmt.Sprintf("settle-%d", i)
		digestChar := fmt.Sprintf("%x", i+1)
		free := cf7FreeAlternative{URL: "http://127.0.0.1:1234/sample", SourceID: "offer-desk:free-sample", Label: "no-signup sample", ResultRef: fmt.Sprintf("free-result-%d", i), ObservedAt: time.Date(2026, 11, 16, 12, i, 0, 0, time.UTC), BodyBytes: 24 + i, Digest: "sha256:" + strings.Repeat(digestChar, 64), Comparison: "sample only; not equivalent"}
		benefit, err := json.Marshal(map[string]any{"run_id": "run-1", "settlement_id": settlementID, "free_alternative": free})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("cf7-call-%d-benefit.json", i)), benefit, 0o600); err != nil {
			t.Fatal(err)
		}
		amountMinor := int64(10)
		if i == 4 {
			amountMinor = 110
		}
		spendBySettlement[settlementID] = researchbudget.Spend{RunID: "run-1", SettlementID: settlementID, TreasuryReceiptID: fmt.Sprintf("receipt-%d", i), Category: "x402_call", IntegrationID: "x402:offer-desk", AmountMinor: amountMinor, Currency: "USD", UnitCode: "USD_MINOR", BenefitState: "measured", BenefitSummary: "paid result", FreeAlternative: &researchbudget.FreeAlternative{RunID: "run-1", SettlementID: settlementID, SourceID: free.SourceID, Label: free.Label, ResultRef: free.ResultRef, SHA256: strings.TrimPrefix(free.Digest, "sha256:"), ResponseBytes: int64(free.BodyBytes), Limitation: free.Comparison, ObservedAt: free.ObservedAt}}
	}
	if err := os.WriteFile(filepath.Join(dir, "x402-call-4-decision-raised.json"), []byte(`{"authorization_id":"auth-4","settlement_id":"settle-4","amount_minor":110,"verdict":"AUTHORIZATION_VERDICT_PENDING","dispatch":"none","call_release_written":false,"operation_grant_id":"grant-4"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x402-call-4-treasury-approved.json"), []byte(`{"run_id":"run-1","authorization_id":"auth-4","settlement_id":"settle-4","approval_id":"approval-4","approval_event_id":"event-4","amount_minor":110,"decision":"approved","source":"notification-hub"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x402-call-4-settled.json"), []byte(`{"run_id":"run-1","authorization_id":"auth-4","settlement_id":"settle-4","treasury_receipt_id":"receipt-4","amount_minor":110,"currency":"USD","approval_id":"approval-4","approval_source":"notification-hub","settlement_report_attempts":2,"idempotent_retry_returned_same_receipt":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	budget := researchbudget.Summary{
		RunID: "run-1", TreasuryBudgetID: "budget-1", MandateID: "mandate-1", Currency: "USD", UnitCode: "USD_MINOR",
		CapMinor: 200, SpendMinor: 140, RemainingMinor: 60, SpendCount: 4,
		Spend: []researchbudget.Spend{spendBySettlement["settle-4"], spendBySettlement["settle-2"], spendBySettlement["settle-1"], spendBySettlement["settle-3"]},
	}
	checkpoint, markdown, err := buildCF7CommissionCheckpoint("run-1", budget, dir)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.SettledMinor != 140 || checkpoint.RemainingMinor != 60 || len(checkpoint.Settlements) != 4 || len(checkpoint.FreeComparisons) != 4 || checkpoint.FreeComparisons[0].FreeAlternative.Digest != "sha256:"+strings.Repeat("5", 64) || checkpoint.FreeComparisons[0].SettlementID != "settle-4" || checkpoint.FreeComparisons[1].FreeAlternative.Digest != "sha256:"+strings.Repeat("3", 64) || checkpoint.OverCapSettlement.TreasuryReceiptID != "receipt-4" {
		t.Fatalf("checkpoint omitted Treasury spend or free comparator: %+v", checkpoint)
	}
	if !strings.Contains(checkpoint.ComputeProcurement, "not_run") || !strings.Contains(checkpoint.DatasetProcurement, "not_run") || !strings.Contains(markdown, `"dispatch": "none"`) || !strings.Contains(markdown, "not equivalent") || !strings.Contains(markdown, "idempotent_retry_returned_same_receipt") || !strings.Contains(markdown, "settle-4` / receipt `receipt-4`: paid 110 USD") {
		t.Fatalf("checkpoint overclaimed missing work or omitted comparison/decision: %+v\n%s", checkpoint, markdown)
	}
	withoutDurableAlternative := budget
	withoutDurableAlternative.Spend = append([]researchbudget.Spend(nil), budget.Spend...)
	withoutDurableAlternative.Spend[0].FreeAlternative = nil
	if _, _, err := buildCF7CommissionCheckpoint("run-1", withoutDurableAlternative, dir); err == nil {
		t.Fatal("commission report must not substitute a harness artifact for missing durable run-bound comparator evidence")
	}
}

func TestCF7TreasuryOverCapApprovalReceiptMustBindExactCallAndOwnerSource(t *testing.T) {
	dir := t.TempDir()
	call := cf1TreasuryX402Call{AuthorizationID: "auth-4", SettlementID: "settle-4", AmountMinor: 110}
	valid := `{"run_id":"run-1","authorization_id":"auth-4","settlement_id":"settle-4","approval_id":"approval-1","approval_event_id":"event-1","amount_minor":110,"decision":"approved","source":"notification-hub"}`
	path := filepath.Join(dir, "x402-call-4-treasury-approved.json")
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readCF7TreasuryApprovalReceipt(dir, "run-1", call)
	if err != nil || got.ApprovalID != "approval-1" {
		t.Fatalf("valid exact Treasury decision receipt = %+v, %v", got, err)
	}
	adminReceipt := strings.Replace(strings.Replace(valid, `,"approval_event_id":"event-1"`, "", 1), `"notification-hub"`, `"treasury-admin"`, 1)
	if err := os.WriteFile(path, []byte(adminReceipt), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := readCF7TreasuryApprovalReceipt(dir, "run-1", call); err != nil || got.Source != "treasury-admin" || got.ApprovalEventID != "" {
		t.Fatalf("valid explicit Treasury Admin receipt = %+v, %v", got, err)
	}
	for _, invalid := range []string{
		strings.Replace(valid, `"run-1"`, `"foreign-run"`, 1),
		strings.Replace(valid, `"notification-hub"`, `"test-helper"`, 1),
		strings.Replace(valid, `"approved"`, `"pending"`, 1),
	} {
		if err := os.WriteFile(path, []byte(invalid), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readCF7TreasuryApprovalReceipt(dir, "run-1", call); err == nil {
			t.Fatalf("accepted an unbound or non-owner over-cap decision: %s", invalid)
		}
	}
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

// resolveCF7ResearchBudgetApproval applies the local test operator's explicit
// answer to the exact pending decision raised by Treasury's authenticated
// AgentSpend.ReportOutcome call. Treasury then retries the same settlement and
// publishes its own snapshot/receipt; this helper never writes report data.
func resolveCF7ResearchBudgetApproval(ctx context.Context, db *database.DB, service *orchestration.Orchestrator, runID uuid.UUID) (*domain.StandingGrant, error) {
	if db == nil || service == nil || runID == uuid.Nil {
		return nil, errors.New("persisted Agent Manager run and authorization stores are required")
	}
	const subject, operation = orchestration.TreasuryServiceSubject, "research-budget.ingest"
	target := "agent-manager:run/" + runID.String()
	sum := sha256.Sum256([]byte(subject + "\x00" + operation + "\x00" + target))
	pendingID := "auth-" + hex.EncodeToString(sum[:16])
	pending, err := database.NewPendingAuthorizationRepository(db).Get(ctx, pendingID)
	if err != nil || pending == nil || pending.Subject != subject || pending.Operation != operation || pending.Target != target || pending.State != domain.PendingAuthorizationPending || pending.EventID == "" {
		return nil, fmt.Errorf("Treasury did not raise the exact pending research-budget decision after report refusal: pending=%+v err=%v", pending, err)
	}
	handled, err := service.ResolveAuthorizationFromHub(ctx, supervision.DecisionResolution{
		AskID: pending.ID, State: "answered", Answer: "approve", AnsweredBy: "cf7-local-test-operator",
		SourceEventType: orchestration.AuthorizationDecisionEventType, RequesterRef: pending.EventID,
		Correlation: map[string]string{"authorization_request_id": pending.ID},
	})
	if err != nil || !handled {
		return nil, fmt.Errorf("apply explicit run-bound report approval: handled=%t err=%v", handled, err)
	}
	resolved, err := database.NewPendingAuthorizationRepository(db).Get(ctx, pending.ID)
	if err != nil || resolved == nil || resolved.State != domain.PendingAuthorizationApproved || resolved.GrantID == "" || resolved.ResolvedBy != "cf7-local-test-operator" {
		return nil, fmt.Errorf("exact run-bound report grant was not retained after approval: request=%+v err=%v", resolved, err)
	}
	active, err := database.NewStandingGrantRepository(db).ActiveForSubject(ctx, subject, time.Now())
	if err != nil {
		return nil, fmt.Errorf("read approved Treasury report grants: %w", err)
	}
	var granted *domain.StandingGrant
	for _, candidate := range active {
		if candidate != nil && candidate.ID == resolved.GrantID {
			granted = candidate
			break
		}
	}
	if granted == nil || granted.Grant == nil || len(granted.Grant.Operations) != 1 || granted.Grant.Operations[0] != operation || len(granted.Grant.Targets) != 1 || granted.Grant.Targets[0] != target || granted.Grant.Approver != "cf7-local-test-operator" || !granted.ExpiresAt.After(time.Now()) || granted.ExpiresAt.After(time.Now().Add(time.Hour)) {
		return nil, fmt.Errorf("issued research-budget grant exceeds the one-run report scope: %+v", granted)
	}
	return granted, nil
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
			approval, err := awaitCF7TreasuryOverCapApproval(ctx, service, run.ID, coord, call)
			if err != nil {
				t.Fatalf("CF7 over-cap call did not receive its separate Treasury decision: %v", err)
			}
			if err := writeJSON(filepath.Join(coord, "x402-call-4-agent-manager-approval.json"), map[string]any{
				"run_id": run.ID.String(), "grant_id": grant.ID.String(), "treasury_approval_id": approval.ApprovalID,
				"decision": "approved", "dispatch_authorized_after_treasury_decision": true,
			}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(coord, "x402-call-4-settle-release"), []byte("released only after exact Treasury approval receipt"), 0o600); err != nil {
				t.Fatal(err)
			}
			var settled struct {
				RunID                    string `json:"run_id"`
				AuthorizationID          string `json:"authorization_id"`
				SettlementID             string `json:"settlement_id"`
				TreasuryReceiptID        string `json:"treasury_receipt_id"`
				AmountMinor              int64  `json:"amount_minor"`
				Currency                 string `json:"currency"`
				ApprovalID               string `json:"approval_id"`
				ApprovalSource           string `json:"approval_source"`
				SettlementReportAttempts int    `json:"settlement_report_attempts"`
				RetrySameReceipt         bool   `json:"idempotent_retry_returned_same_receipt"`
			}
			if err := waitForFileOrRunTerminal(ctx, service, run.ID, filepath.Join(coord, "x402-call-4-settled.json"), &settled); err != nil {
				t.Fatalf("CF7 over-cap call did not settle after the Treasury decision: %v", err)
			}
			if settled.RunID != run.ID.String() || settled.AuthorizationID != call.AuthorizationID || settled.SettlementID != call.SettlementID || settled.TreasuryReceiptID == "" || settled.AmountMinor != call.AmountMinor || settled.Currency != "USD" || settled.ApprovalID != approval.ApprovalID || settled.ApprovalSource != approval.Source || settled.SettlementReportAttempts < 2 || !settled.RetrySameReceipt {
				t.Fatalf("CF7 over-cap settlement did not bind the exact decision and stable retry: %+v", settled)
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
