package walletops

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/vrooli/cli-core/cliutil"
	"treasury/internal/authorization"
	"treasury/internal/identity"
)

const (
	HeaderAgentSessionPublicKey = "X-Vrooli-Treasury-Session-Public-Key"
	HeaderAgentSessionProof     = "X-Vrooli-Treasury-Session-Proof"
)

var ErrAgentSessionProof = errors.New("mandate-bound agent session proof is invalid")

// AgentSettlementIntent is the canonical, mandate-specific action an agent
// approves with its run-held session key. Treasury's derived budget wallet
// remains the transaction signer; the agent's key is a separate proof of the
// run's authority to request that exact spend.
type AgentSettlementIntent struct {
	AuthorizationID string `json:"authorization_id"`
	MandateID       string `json:"mandate_id"`
	BudgetID        string `json:"budget_id"`
	WalletID        string `json:"wallet_id"`
	WalletAddress   string `json:"wallet_address"`
	Network         string `json:"network"`
	AgentSubject    string `json:"agent_subject"`
	AgentRunID      string `json:"agent_run_id"`
	SettlementID    string `json:"settlement_id"`
	InstrumentID    string `json:"instrument_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	Counterparty    string `json:"counterparty"`
	Generation      int64  `json:"generation"`
	ExpiresAt       string `json:"expires_at"`
}

// AgentSettlementIntentMessage is exported so an agent-side signer can build
// exactly the same domain-separated message without receiving a Treasury key.
func AgentSettlementIntentMessage(intent AgentSettlementIntent) (string, error) {
	if intent.AuthorizationID == "" || intent.MandateID == "" || intent.BudgetID == "" || intent.WalletID == "" || intent.WalletAddress == "" || intent.Network == "" || intent.AgentSubject == "" || intent.AgentRunID == "" || intent.SettlementID == "" || intent.InstrumentID == "" || intent.IdempotencyKey == "" || intent.AmountMinor <= 0 || intent.Currency == "" || intent.Counterparty == "" || intent.Generation <= 0 || intent.ExpiresAt == "" {
		return "", ErrInvalid
	}
	encoded, err := json.Marshal(intent)
	if err != nil {
		return "", err
	}
	return "Vrooli Treasury mandate session settlement v1\n" + string(encoded), nil
}

func AgentSettlementIntentDigest(intent AgentSettlementIntent) ([]byte, error) {
	message, err := AgentSettlementIntentMessage(intent)
	if err != nil {
		return nil, err
	}
	return gethcrypto.Keccak256([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len([]byte(message))) + message)), nil
}

// BindAgentSessionPublicKey binds the agent-generated public key to the live
// mandate generation. The matching private key never enters Treasury or
// Credential Authority. Rebinding to a different key requires operator
// rotation, so a lost agent key cannot silently regain authority.
func (s *Service) BindAgentSessionPublicKey(ctx context.Context, budgetID, mandateID string, claims identity.Claims, publicKey string) error {
	if s == nil || s.DB == nil || s.Clock == nil || s.Mandates == nil || budgetID == "" || mandateID == "" {
		return ErrInvalid
	}
	if err := identity.Authorize(claims, "treasury.read", "treasury:budget/"+budgetID, s.Clock.Now()); err != nil {
		return err
	}
	key, err := parseSessionPublicKey(publicKey)
	if err != nil {
		return err
	}
	publicKey = "0x" + hex.EncodeToString(gethcrypto.FromECDSAPub(key))
	record, err := s.sessionByMandate(ctx, mandateID, "active")
	if err != nil || record.budgetID != budgetID || record.agent != claims.Subject || record.run != claims.RunID || !s.Clock.Now().Before(record.expiresAt) {
		return ErrSessionInactive
	}
	if _, err := s.Mandates.RequireLive(ctx, mandateID); err != nil {
		return ErrSessionInactive
	}
	now := s.Clock.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.DB.ExecContext(ctx, `INSERT INTO mandate_agent_session_keys(mandate_id,generation,budget_id,agent_subject,agent_run_id,public_key,registered_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT(mandate_id,generation) DO NOTHING`, mandateID, record.generation, budgetID, claims.Subject, claims.RunID, publicKey, now)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		var existing string
		if err = s.DB.QueryRowContext(ctx, `SELECT public_key FROM mandate_agent_session_keys WHERE mandate_id=? AND generation=? AND budget_id=? AND agent_subject=? AND agent_run_id=?`, mandateID, record.generation, budgetID, claims.Subject, claims.RunID).Scan(&existing); err != nil || !strings.EqualFold(existing, publicKey) {
			return ErrConflict
		}
	}
	return nil
}

func parseSessionPublicKey(value string) (*ecdsa.PublicKey, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	raw, err := hex.DecodeString(value)
	if err != nil || len(raw) != 65 || raw[0] != 4 {
		return nil, ErrInvalid
	}
	key, err := gethcrypto.UnmarshalPubkey(raw)
	if err != nil {
		return nil, ErrInvalid
	}
	return key, nil
}

// AuthorizeAgentSettlement verifies proof-of-possession from the agent-held
// session key before settlement can enter the rail. It rechecks the active
// mandate/session/authorization and freeze state, and makes identical retries
// idempotent while rejecting an authorization rebound to another request.
func (s *Service) AuthorizeAgentSettlement(ctx context.Context, claims identity.Claims, intent AgentSettlementIntent, signatureHex string) error {
	if s == nil || s.DB == nil || s.Clock == nil || s.Mandates == nil || s.Authorizations == nil || s.Budgets == nil {
		return ErrInvalid
	}
	if err := identity.Authorize(claims, "treasury.spend", cliutil.TreasurySettlementTarget(intent.AuthorizationID, intent.MandateID, intent.AmountMinor, intent.Currency, intent.Counterparty, intent.SettlementID, intent.InstrumentID, intent.IdempotencyKey), s.Clock.Now()); err != nil {
		return err
	}
	record, err := s.sessionByMandate(ctx, intent.MandateID, "active")
	if err != nil || record.budgetID != intent.BudgetID || record.agent != claims.Subject || record.run != claims.RunID || record.agent != intent.AgentSubject || record.run != intent.AgentRunID || !s.Clock.Now().Before(record.expiresAt) {
		return ErrSessionInactive
	}
	if _, err := s.Mandates.RequireLive(ctx, intent.MandateID); err != nil {
		return ErrSessionInactive
	}
	var publicKey string
	if err = s.DB.QueryRowContext(ctx, `SELECT public_key FROM mandate_agent_session_keys WHERE mandate_id=? AND generation=? AND budget_id=? AND agent_subject=? AND agent_run_id=?`, intent.MandateID, record.generation, record.budgetID, record.agent, record.run).Scan(&publicKey); err != nil {
		return ErrSessionInactive
	}
	b, err := s.load(ctx, record.budgetID)
	if err != nil || b.agent != claims.Subject || b.run != claims.RunID {
		return ErrSessionInactive
	}
	w, err := s.Wallets.Get(ctx, b.walletID, record.network, record.operator)
	if err != nil {
		return err
	}
	intent.WalletID, intent.WalletAddress, intent.Network = b.walletID, w.Address, record.network
	intent.Generation, intent.ExpiresAt = record.generation, record.expiresAt.UTC().Format(time.RFC3339Nano)
	auth, err := s.Authorizations.Get(ctx, intent.AuthorizationID)
	if err != nil {
		return err
	}
	if auth.Verdict != authorization.VerdictApproved && auth.Verdict != authorization.VerdictSettled || auth.MandateID != intent.MandateID || auth.BudgetID != intent.BudgetID || auth.RequestingAgent != claims.Subject || auth.AmountMinor != intent.AmountMinor || !strings.EqualFold(auth.Currency, intent.Currency) || !strings.EqualFold(auth.Counterparty, intent.Counterparty) || auth.ID == "" || !s.Clock.Now().Before(auth.ExpiresAt) {
		return ErrAgentSessionProof
	}
	grant, err := s.Mandates.RequireLive(ctx, intent.MandateID)
	if err != nil || grant.AgentSubject != claims.Subject || grant.AgentRunID != claims.RunID || grant.BudgetID != intent.BudgetID || !grant.AllowsCounterparty(intent.Counterparty) {
		return ErrAgentSessionProof
	}
	policy, err := s.Budgets.Get(ctx, intent.BudgetID)
	if err != nil || policy.Currency != strings.ToUpper(intent.Currency) || !policy.AllowsCounterparty(intent.Counterparty) {
		return ErrAgentSessionProof
	}
	frozen, _, err := s.Budgets.IsFrozen(ctx, grant.BookID, intent.BudgetID)
	if err != nil || frozen {
		return ErrAgentSessionProof
	}
	message, err := AgentSettlementIntentMessage(intent)
	if err != nil {
		return err
	}
	signature, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(signatureHex), "0x"))
	if err != nil || len(signature) != 65 || signature[64] > 1 {
		return ErrAgentSessionProof
	}
	digest, err := AgentSettlementIntentDigest(intent)
	if err != nil {
		return err
	}
	pub, err := parseSessionPublicKey(publicKey)
	if err != nil {
		return ErrAgentSessionProof
	}
	recovered, err := gethcrypto.SigToPub(digest, signature)
	if err != nil || !equalSessionKeys(pub, recovered) {
		return ErrAgentSessionProof
	}
	intentHash := sha256.Sum256([]byte(message))
	signatureHash := sha256.Sum256(signature)
	return s.consumeAgentSessionProof(ctx, intent, record.generation, "sha256:"+hex.EncodeToString(intentHash[:]), "sha256:"+hex.EncodeToString(signatureHash[:]))
}

func equalSessionKeys(a, b *ecdsa.PublicKey) bool {
	return a != nil && b != nil && strings.EqualFold(hex.EncodeToString(gethcrypto.FromECDSAPub(a)), hex.EncodeToString(gethcrypto.FromECDSAPub(b)))
}

func (s *Service) consumeAgentSessionProof(ctx context.Context, intent AgentSettlementIntent, generation int64, intentDigest, signatureDigest string) error {
	db, ok := s.DB.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return ErrInvalid
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `UPDATE authorization_admission_gate SET generation=generation+1 WHERE id=1`); err != nil {
		return err
	}
	var frozen bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM budgets WHERE id=? AND frozen=1
		UNION ALL SELECT 1 FROM freeze_controls WHERE frozen=1 AND (
			(scope='scenario' AND scope_id='*') OR
			(scope='book' AND scope_id=(SELECT book_id FROM budgets WHERE id=?))
		)
	)`, intent.BudgetID, intent.BudgetID).Scan(&frozen); err != nil {
		return err
	}
	if frozen {
		return ErrAgentSessionProof
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO mandate_agent_session_proofs(authorization_id,mandate_id,generation,intent_digest,signature_digest,settlement_id,instrument_id,idempotency_key,consumed_at) VALUES(?,?,?,?,?,?,?,?,?)`, intent.AuthorizationID, intent.MandateID, generation, intentDigest, signatureDigest, intent.SettlementID, intent.InstrumentID, intent.IdempotencyKey, s.Clock.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		var priorIntent, priorSignature, priorSettlement, priorInstrument, priorIdempotency string
		readErr := tx.QueryRowContext(ctx, `SELECT intent_digest,signature_digest,settlement_id,instrument_id,idempotency_key FROM mandate_agent_session_proofs WHERE authorization_id=? AND mandate_id=? AND generation=?`, intent.AuthorizationID, intent.MandateID, generation).Scan(&priorIntent, &priorSignature, &priorSettlement, &priorInstrument, &priorIdempotency)
		if readErr == nil && priorIntent == intentDigest && priorSettlement == intent.SettlementID && priorInstrument == intent.InstrumentID && priorIdempotency == intent.IdempotencyKey {
			return tx.Commit()
		}
		return ErrSessionReplay
	}
	return tx.Commit()
}

func (s *Service) HasAgentSessionProof(ctx context.Context, authorizationID, mandateID string, generation int64) bool {
	if s == nil || s.DB == nil || authorizationID == "" || mandateID == "" || generation <= 0 {
		return false
	}
	var count int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM mandate_agent_session_proofs WHERE authorization_id=? AND mandate_id=? AND generation=?`, authorizationID, mandateID, generation).Scan(&count)
	return err == nil && count == 1
}

func signAgentIntent(privateKeyHex string, intent AgentSettlementIntent) (string, error) {
	key, err := gethcrypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", err
	}
	digest, err := AgentSettlementIntentDigest(intent)
	if err != nil {
		return "", err
	}
	signature, err := gethcrypto.Sign(digest, key)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(signature), nil
}
