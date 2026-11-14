package cliutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// TreasurySpendIntent is the reviewable immutable intent an Agent Manager
// consent covers. The proposal target authorizes admission; the settlement
// target commits the exact rail instrument and settlement id before Treasury
// admits the proposal. Both targets are derived from this one value so their
// authorization, amount, mandate, currency, and counterparty cannot drift.
type TreasurySpendIntent struct {
	AuthorizationID             string
	MandateID                   string
	AmountMinor                 int64
	Currency                    string
	Counterparty                string
	AuthorizationIdempotencyKey string
	SettlementID                string
	InstrumentID                string
	SettlementIdempotencyKey    string
}

func (i TreasurySpendIntent) Targets() (proposal, settlement string) {
	proposal = TreasurySpendIntentAuthorizationTarget(i)
	settlement = TreasurySettlementTarget(i.AuthorizationID, i.MandateID, i.AmountMinor, i.Currency, i.Counterparty, i.SettlementID, i.InstrumentID, i.SettlementIdempotencyKey)
	return proposal, settlement
}

// TreasurySpendIntentAuthorizationTarget binds every term known before
// authorization, including the future settlement ID, instrument, and retry
// key. Treasury stores these terms with the authorization and requires the
// later settlement request to match them exactly.
func TreasurySpendIntentAuthorizationTarget(i TreasurySpendIntent) string {
	intent := struct {
		AuthorizationID             string `json:"authorization_id"`
		MandateID                   string `json:"mandate_id"`
		AmountMinor                 int64  `json:"amount_minor"`
		Currency                    string `json:"currency"`
		Counterparty                string `json:"counterparty"`
		AuthorizationIdempotencyKey string `json:"authorization_idempotency_key"`
		SettlementID                string `json:"settlement_id"`
		InstrumentID                string `json:"instrument_id"`
		SettlementIdempotencyKey    string `json:"settlement_idempotency_key"`
	}{
		AuthorizationID: strings.TrimSpace(i.AuthorizationID), MandateID: strings.TrimSpace(i.MandateID),
		AmountMinor: i.AmountMinor, Currency: strings.ToUpper(strings.TrimSpace(i.Currency)),
		Counterparty: strings.TrimSpace(i.Counterparty), AuthorizationIdempotencyKey: strings.TrimSpace(i.AuthorizationIdempotencyKey),
		SettlementID: strings.TrimSpace(i.SettlementID), InstrumentID: strings.TrimSpace(i.InstrumentID),
		SettlementIdempotencyKey: strings.TrimSpace(i.SettlementIdempotencyKey),
	}
	encoded, _ := json.Marshal(intent)
	digest := sha256.Sum256(append([]byte("vrooli treasury immutable spend intent v1\n"), encoded...))
	return "treasury:authorization/" + intent.AuthorizationID + "/intent/" + hex.EncodeToString(digest[:])
}

// TreasuryAuthorizationTarget binds an Agent Manager grant to the immutable
// proposal fields that Treasury evaluates. Treasury repeats this derivation
// from the submitted request or retained authorization before admitting spend.
func TreasuryAuthorizationTarget(id, mandateID string, amountMinor int64, currency, counterparty, idempotencyKey string) string {
	intent := struct {
		ID             string `json:"id"`
		MandateID      string `json:"mandate_id"`
		AmountMinor    int64  `json:"amount_minor"`
		Currency       string `json:"currency"`
		Counterparty   string `json:"counterparty"`
		IdempotencyKey string `json:"idempotency_key"`
	}{
		ID: strings.TrimSpace(id), MandateID: strings.TrimSpace(mandateID), AmountMinor: amountMinor,
		Currency: strings.ToUpper(strings.TrimSpace(currency)), Counterparty: strings.TrimSpace(counterparty), IdempotencyKey: strings.TrimSpace(idempotencyKey),
	}
	encoded, _ := json.Marshal(intent)
	digest := sha256.Sum256(append([]byte("vrooli treasury spend intent v1\n"), encoded...))
	return "treasury:authorization/" + intent.ID + "/intent/" + hex.EncodeToString(digest[:])
}

// TreasurySettlementTarget adds the fixed rail instrument, settlement ID and
// idempotency key to the immutable authorization fields. A retry of the same
// settlement retains this target; changing any settlement field changes it.
func TreasurySettlementTarget(id, mandateID string, amountMinor int64, currency, counterparty, settlementID, instrumentID, idempotencyKey string) string {
	intent := struct {
		AuthorizationID string `json:"authorization_id"`
		MandateID       string `json:"mandate_id"`
		AmountMinor     int64  `json:"amount_minor"`
		Currency        string `json:"currency"`
		Counterparty    string `json:"counterparty"`
		SettlementID    string `json:"settlement_id"`
		InstrumentID    string `json:"instrument_id"`
		IdempotencyKey  string `json:"idempotency_key"`
	}{
		AuthorizationID: strings.TrimSpace(id), MandateID: strings.TrimSpace(mandateID), AmountMinor: amountMinor,
		Currency: strings.ToUpper(strings.TrimSpace(currency)), Counterparty: strings.TrimSpace(counterparty),
		SettlementID: strings.TrimSpace(settlementID), InstrumentID: strings.TrimSpace(instrumentID),
		IdempotencyKey: strings.TrimSpace(idempotencyKey),
	}
	encoded, _ := json.Marshal(intent)
	digest := sha256.Sum256(append([]byte("vrooli treasury settlement intent v1\n"), encoded...))
	return "treasury:settlement/" + intent.AuthorizationID + "/intent/" + hex.EncodeToString(digest[:])
}
