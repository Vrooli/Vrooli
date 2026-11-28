//go:build linux

package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type cf1SettledRetryCharge struct {
	ID               string `json:"id"`
	AuthorizationID  string `json:"authorization_id"`
	InstrumentID     string `json:"instrument_id"`
	AmountMinor      int64  `json:"amount_minor"`
	Currency         string `json:"currency"`
	Outcome          string `json:"outcome"`
	ExternalID       string `json:"external_id"`
	ReceiptReference string `json:"receipt_reference"`
}

type cf1OverCapRetryContinuityEvidence struct {
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

func parseCF1SettledRetryCharge(data []byte, authorizationID, settlementID, instrumentID string, amountMinor int64) (cf1SettledRetryCharge, error) {
	var response struct {
		Settlement struct {
			ID               string          `json:"id"`
			AuthorizationID  string          `json:"authorization_id"`
			InstrumentID     string          `json:"instrument_id"`
			AmountMinor      json.RawMessage `json:"amount_minor"`
			Currency         string          `json:"currency"`
			Outcome          string          `json:"outcome"`
			ExternalID       string          `json:"external_id"`
			ReceiptReference string          `json:"receipt_reference"`
		} `json:"settlement"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return cf1SettledRetryCharge{}, fmt.Errorf("decode settled Treasury charge: %w", err)
	}
	parsedAmount := int64(0)
	amountText := strings.Trim(string(response.Settlement.AmountMinor), `"`)
	parsedAmount, amountErr := strconv.ParseInt(amountText, 10, 64)
	if amountErr != nil {
		return cf1SettledRetryCharge{}, fmt.Errorf("decode settled Treasury amount_minor: %w", amountErr)
	}
	settlement := cf1SettledRetryCharge{
		ID: response.Settlement.ID, AuthorizationID: response.Settlement.AuthorizationID,
		InstrumentID: response.Settlement.InstrumentID, AmountMinor: parsedAmount,
		Currency: response.Settlement.Currency, Outcome: response.Settlement.Outcome,
		ExternalID: response.Settlement.ExternalID, ReceiptReference: response.Settlement.ReceiptReference,
	}
	if settlement.ID != settlementID || settlement.AuthorizationID != authorizationID || settlement.InstrumentID != instrumentID {
		return cf1SettledRetryCharge{}, errors.New("settled Treasury charge did not preserve authorization, settlement, and instrument identity")
	}
	if settlement.Outcome != "CHARGE_OUTCOME_SETTLED" || parsedAmount != amountMinor || amountMinor <= 0 || settlement.Currency != "USD" {
		return cf1SettledRetryCharge{}, errors.New("Treasury charge is not settled for the exact expected USD amount")
	}
	if !cf1HexIdentity(settlement.ExternalID, 32) || strings.TrimSpace(settlement.ReceiptReference) == "" {
		return cf1SettledRetryCharge{}, errors.New("settled Treasury charge omitted a valid transaction hash or raw receipt reference")
	}
	return settlement, nil
}

func compareCF1SettledRetryCharges(before, after cf1SettledRetryCharge) (string, int, error) {
	if before.ID != after.ID || before.AuthorizationID != after.AuthorizationID || before.InstrumentID != after.InstrumentID {
		return "", 0, errors.New("Treasury retry changed settlement, authorization, or instrument identity")
	}
	if before.Outcome != "CHARGE_OUTCOME_SETTLED" || after.Outcome != before.Outcome || before.AmountMinor <= 0 || after.AmountMinor != before.AmountMinor || before.Currency != "USD" || after.Currency != before.Currency {
		return "", 0, errors.New("Treasury retry changed settled outcome, exact amount, or currency")
	}
	if before.ExternalID != after.ExternalID {
		return "", 0, errors.New("Treasury retry changed the settled transaction hash")
	}
	return compareCF1X402ReceiptReferences(before.ReceiptReference, after.ReceiptReference)
}

func validateCF1OverCapRetryContinuityEvidence(evidence cf1OverCapRetryContinuityEvidence, runID, authorizationID, settlementID, instrumentID string, inCapMinor, overCapMinor int64) error {
	expectedHeadroom := int64(cf1CrossBudgetIdentityCapMinor) - inCapMinor - overCapMinor
	if evidence.RunID != runID || evidence.AuthorizationID != authorizationID || evidence.SettlementID != settlementID || evidence.InstrumentID != instrumentID {
		return errors.New("over-cap retry evidence is not bound to the expected ordinary run spend")
	}
	if evidence.StatusBeforeRestart != "CHARGE_OUTCOME_SETTLED" || evidence.StatusAfterRestart != evidence.StatusBeforeRestart {
		return errors.New("over-cap retry evidence does not show settled outcomes on both sides of restart")
	}
	if !cf1HexIdentity(evidence.TransactionHashBeforeRestart, 32) || evidence.TransactionHashAfterRestart != evidence.TransactionHashBeforeRestart || !evidence.SameTransactionHash {
		return errors.New("over-cap retry evidence changed its transaction identity across restart")
	}
	if overCapMinor <= 0 || inCapMinor <= 0 || evidence.AmountMinorBeforeRestart != overCapMinor || evidence.AmountMinorAfterRestart != overCapMinor || evidence.Currency != "USD" {
		return errors.New("over-cap retry evidence changed its exact USD amount across restart")
	}
	if evidence.ReceiptReferenceBytes <= 0 || !cf1HexIdentity(evidence.ReceiptReferenceSHA256, 32) || !evidence.SameReceiptReferenceBytes {
		return errors.New("over-cap retry evidence does not prove exact receipt-reference continuity")
	}
	if expectedHeadroom < 0 || evidence.LedgerDebitCountBeforeRetry != 1 || evidence.LedgerDebitCountAfterRetry != 1 || evidence.HeadroomMinorBeforeRetry != expectedHeadroom || evidence.HeadroomMinorAfterRetry != expectedHeadroom {
		return errors.New("over-cap retry evidence does not show exactly one ledger debit and unchanged headroom")
	}
	return nil
}

func TestCF1SettledRetryChargeRequiresExactIdentityAndAmount(t *testing.T) {
	const response = `{"settlement":{"id":"settlement-a","authorization_id":"authorization-a","instrument_id":"instrument-a","amount_minor":110,"currency":"USD","outcome":"CHARGE_OUTCOME_SETTLED","external_id":"0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","receipt_reference":"0xrawreceipt"}}`
	charge, err := parseCF1SettledRetryCharge([]byte(response), "authorization-a", "settlement-a", "instrument-a", 110)
	if err != nil {
		t.Fatal(err)
	}
	if charge.AmountMinor != 110 || charge.ReceiptReference != "0xrawreceipt" {
		t.Fatalf("decoded charge = %+v", charge)
	}
	digest, length, err := compareCF1SettledRetryCharges(charge, charge)
	if err != nil || length != len(charge.ReceiptReference) || digest == "" {
		t.Fatalf("exact retry changed receipt continuity: digest=%q length=%d err=%v", digest, length, err)
	}
	for name, mutate := range map[string]func(*cf1SettledRetryCharge){
		"transaction": func(retry *cf1SettledRetryCharge) {
			retry.ExternalID = "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		},
		"amount":  func(retry *cf1SettledRetryCharge) { retry.AmountMinor-- },
		"receipt": func(retry *cf1SettledRetryCharge) { retry.ReceiptReference += "changed" },
	} {
		t.Run("retry changed "+name, func(t *testing.T) {
			retry := charge
			mutate(&retry)
			if _, _, err := compareCF1SettledRetryCharges(charge, retry); err == nil {
				t.Fatal("changed retry result was accepted")
			}
		})
	}
	for _, tc := range []struct {
		name          string
		authorization string
		settlement    string
		instrument    string
		amount        int64
	}{
		{name: "wrong authorization", authorization: "other", settlement: "settlement-a", instrument: "instrument-a", amount: 110},
		{name: "wrong settlement", authorization: "authorization-a", settlement: "other", instrument: "instrument-a", amount: 110},
		{name: "wrong instrument", authorization: "authorization-a", settlement: "settlement-a", instrument: "other", amount: 110},
		{name: "wrong exact amount", authorization: "authorization-a", settlement: "settlement-a", instrument: "instrument-a", amount: 109},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseCF1SettledRetryCharge([]byte(response), tc.authorization, tc.settlement, tc.instrument, tc.amount); err == nil {
				t.Fatal("mismatched settled charge was accepted")
			}
		})
	}
	for _, malformed := range []string{
		strings.Replace(response, `"outcome":"CHARGE_OUTCOME_SETTLED"`, `"outcome":"CHARGE_OUTCOME_UNKNOWN"`, 1),
		strings.Replace(response, `"receipt_reference":"0xrawreceipt"`, `"receipt_reference":""`, 1),
		strings.Replace(response, "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "0xbad", 1),
	} {
		if _, err := parseCF1SettledRetryCharge([]byte(malformed), "authorization-a", "settlement-a", "instrument-a", 110); err == nil {
			t.Fatalf("malformed settled result was accepted: %s", malformed)
		}
	}
}

func TestCF1OverCapRetryContinuityEvidenceRequiresOneExactDebit(t *testing.T) {
	evidence := cf1OverCapRetryContinuityEvidence{
		RunID: "run-1", AuthorizationID: "authorization-1", SettlementID: "settlement-1", InstrumentID: "instrument-1",
		StatusBeforeRestart: "CHARGE_OUTCOME_SETTLED", StatusAfterRestart: "CHARGE_OUTCOME_SETTLED",
		TransactionHashBeforeRestart: "0x" + strings.Repeat("a", 64), TransactionHashAfterRestart: "0x" + strings.Repeat("a", 64), SameTransactionHash: true,
		AmountMinorBeforeRestart: 110, AmountMinorAfterRestart: 110, Currency: "USD", ReceiptReferenceBytes: 30,
		ReceiptReferenceSHA256: strings.Repeat("b", 64), SameReceiptReferenceBytes: true,
		LedgerDebitCountBeforeRetry: 1, LedgerDebitCountAfterRetry: 1, HeadroomMinorBeforeRetry: 860, HeadroomMinorAfterRetry: 860,
	}
	if err := validateCF1OverCapRetryContinuityEvidence(evidence, "run-1", "authorization-1", "settlement-1", "instrument-1", 30, 110); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*cf1OverCapRetryContinuityEvidence){
		"second debit":     func(value *cf1OverCapRetryContinuityEvidence) { value.LedgerDebitCountAfterRetry = 2 },
		"changed headroom": func(value *cf1OverCapRetryContinuityEvidence) { value.HeadroomMinorAfterRetry++ },
		"changed amount":   func(value *cf1OverCapRetryContinuityEvidence) { value.AmountMinorAfterRestart-- },
		"changed transaction": func(value *cf1OverCapRetryContinuityEvidence) {
			value.TransactionHashAfterRestart = "0x" + strings.Repeat("c", 64)
		},
		"changed raw receipt": func(value *cf1OverCapRetryContinuityEvidence) { value.SameReceiptReferenceBytes = false },
	} {
		t.Run(name, func(t *testing.T) {
			changed := evidence
			mutate(&changed)
			if err := validateCF1OverCapRetryContinuityEvidence(changed, "run-1", "authorization-1", "settlement-1", "instrument-1", 30, 110); err == nil {
				t.Fatal("mutated retry evidence was accepted")
			}
		})
	}
}
