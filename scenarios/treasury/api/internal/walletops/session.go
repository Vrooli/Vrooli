package walletops

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"treasury/internal/authorization"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	"treasury/internal/wallet"
)

var (
	ErrSessionInactive = errors.New("mandate wallet session is inactive")
	ErrSessionReplay   = errors.New("mandate wallet signing authorization was already consumed")
	sha256Digest       = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

const mandateSessionTTL = 15 * time.Minute

// MandateSessionWallet is safe to return to the agent. The signing key stays
// in Credential Authority and every signature still crosses the canonical
// authorization and freeze checks.
type MandateSessionWallet struct {
	MandateID, BudgetID, WalletID, OperatorIdentity string
	AgentSubject, AgentRunID, Network, Address      string
	Generation                                      int64
	ExpiresAt                                       time.Time
}

type sessionKeyRecord struct {
	id, mandateID, budgetID, walletID, operator, agent, run, network, state string
	generation                                                              int64
	expiresAt                                                               time.Time
}

func (s *Service) budgetWalletForSigning(ctx context.Context, walletID string) (sessionKeyRecord, error) {
	var record sessionKeyRecord
	err := s.DB.QueryRowContext(ctx, `SELECT budget_id,wallet_id,operator_identity,agent_subject,agent_run_id,network FROM budget_wallets WHERE wallet_id=?`, walletID).Scan(
		&record.budgetID, &record.walletID, &record.operator, &record.agent, &record.run, &record.network)
	return record, err
}

func (s *Service) ProvisionMandateSession(ctx context.Context, operator, mandateID string) (MandateSessionWallet, error) {
	if s == nil || s.DB == nil || s.Wallets == nil || s.Mandates == nil || s.Clock == nil || operator == "" || mandateID == "" {
		return MandateSessionWallet{}, ErrInvalid
	}
	grant, err := s.Mandates.RequireLive(ctx, mandateID)
	if err != nil {
		return MandateSessionWallet{}, fmt.Errorf("require live mandate for session wallet: %w", err)
	}
	b, err := s.load(ctx, grant.BudgetID)
	if err != nil {
		return MandateSessionWallet{}, err
	}
	if b.operator != operator || b.agent == "" || b.run == "" || grant.AgentSubject != b.agent || grant.AgentRunID != b.run || grant.BookID == "" || grant.BudgetID != b.budgetID {
		return MandateSessionWallet{}, ErrInvalid
	}
	if active, getErr := s.sessionByMandate(ctx, mandateID, "active"); getErr == nil {
		if !sameSessionBinding(active, b, grant) || !s.Clock.Now().Before(active.expiresAt) {
			return MandateSessionWallet{}, ErrConflict
		}
		return s.publicSession(ctx, active)
	} else if !errors.Is(getErr, sql.ErrNoRows) {
		return MandateSessionWallet{}, getErr
	}
	var generation int64
	if err = s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(generation),0) FROM mandate_session_keys WHERE mandate_id=?`, mandateID).Scan(&generation); err != nil {
		return MandateSessionWallet{}, err
	}
	generation++
	walletID := stableID("mandate-session", mandateID, fmt.Sprint(generation))
	// This short-lived generation marker is never the funded budget identity.
	// The agent-held key binds its exact settlement intent to the derived budget
	// address; Treasury keeps its separate budget signer behind the policy gate.
	marker, err := s.Wallets.Create(ctx, wallet.CreateInput{
		ID: walletID, BookID: grant.BookID, Network: b.network, OperatorIdentity: operator,
	})
	if err != nil {
		return MandateSessionWallet{}, err
	}
	now := s.Clock.Now().UTC()
	expiresAt := now.Add(mandateSessionTTL)
	if grant.ExpiresAt.Before(expiresAt) {
		expiresAt = grant.ExpiresAt.UTC()
	}
	if !now.Before(expiresAt) {
		return MandateSessionWallet{}, ErrSessionInactive
	}
	id := stableID("session", mandateID, fmt.Sprint(generation))
	_, err = s.DB.ExecContext(ctx, `INSERT INTO mandate_session_keys(id,mandate_id,budget_id,wallet_id,operator_identity,agent_subject,agent_run_id,network,generation,state,expires_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,'active',?,?,?)`,
		id, mandateID, b.budgetID, marker.ID, operator, grant.AgentSubject, grant.AgentRunID, b.network, generation,
		expiresAt.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		// Concurrent provisioning may have won the unique active-session race.
		if active, readErr := s.sessionByMandate(ctx, mandateID, "active"); readErr == nil && sameSessionBinding(active, b, grant) {
			return s.publicSession(ctx, active)
		}
		return MandateSessionWallet{}, fmt.Errorf("persist mandate session key: %w", err)
	}
	budgetWallet, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return MandateSessionWallet{}, err
	}
	return MandateSessionWallet{MandateID: mandateID, BudgetID: b.budgetID, WalletID: budgetWallet.ID, OperatorIdentity: operator, AgentSubject: grant.AgentSubject, AgentRunID: grant.AgentRunID, Network: budgetWallet.Network, Address: budgetWallet.Address, Generation: generation, ExpiresAt: expiresAt}, nil
}

func (s *Service) GetMandateSessionForAgent(ctx context.Context, budgetID, mandateID string, claims identity.Claims) (MandateSessionWallet, error) {
	if s == nil || s.Clock == nil || s.Mandates == nil || budgetID == "" || mandateID == "" {
		return MandateSessionWallet{}, ErrInvalid
	}
	if err := identity.Authorize(claims, "treasury.read", "treasury:budget/"+budgetID, s.Clock.Now()); err != nil {
		return MandateSessionWallet{}, err
	}
	record, err := s.sessionByMandate(ctx, mandateID, "active")
	if err != nil {
		return MandateSessionWallet{}, ErrNotFound
	}
	if record.budgetID != budgetID || record.agent != claims.Subject || record.run != claims.RunID || !s.Clock.Now().Before(record.expiresAt) {
		return MandateSessionWallet{}, ErrNotFound
	}
	if _, err = s.Mandates.RequireLive(ctx, mandateID); err != nil {
		return MandateSessionWallet{}, ErrSessionInactive
	}
	var publicKey string
	if err = s.DB.QueryRowContext(ctx, `SELECT public_key FROM mandate_agent_session_keys WHERE mandate_id=? AND generation=? AND budget_id=? AND agent_subject=? AND agent_run_id=?`, mandateID, record.generation, budgetID, claims.Subject, claims.RunID).Scan(&publicKey); err != nil || publicKey == "" {
		return MandateSessionWallet{}, ErrSessionInactive
	}
	return s.publicSession(ctx, record)
}

// RevokeMandateSession is an operator-only lifecycle action. A revoked key is
// never reactivated; a later ProvisionMandateSession creates the next
// deterministic generation under the same live mandate.
func (s *Service) RevokeMandateSession(ctx context.Context, operator, mandateID string) error {
	return s.retireMandateSession(ctx, operator, mandateID, "revoked")
}

func (s *Service) retireMandateSession(ctx context.Context, operator, mandateID, state string) error {
	if s == nil || s.DB == nil || s.Clock == nil || operator == "" || mandateID == "" {
		return ErrInvalid
	}
	if state != "revoked" && state != "rotated" {
		return ErrInvalid
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE mandate_session_keys SET state=?,updated_at=? WHERE mandate_id=? AND operator_identity=? AND state='active'`, state, s.Clock.Now().UTC().Format(time.RFC3339Nano), mandateID, operator)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 1 {
		return nil
	}
	current, err := s.sessionByMandate(ctx, mandateID, state)
	if err == nil && current.operator == operator {
		return nil
	}
	return ErrNotFound
}

// RotateMandateSession revokes the old generation before deriving the next.
// If provisioning then fails, the mandate fails closed until the operator
// retries; the former key cannot silently regain signing authority.
func (s *Service) RotateMandateSession(ctx context.Context, operator, mandateID string) (MandateSessionWallet, error) {
	if err := s.retireMandateSession(ctx, operator, mandateID, "rotated"); err != nil {
		if !errors.Is(err, ErrNotFound) {
			return MandateSessionWallet{}, err
		}
	}
	return s.ProvisionMandateSession(ctx, operator, mandateID)
}

func (s *Service) AuthorizeAndConsumeSessionSign(ctx context.Context, request wallet.SessionSignAuthorization) error {
	if s == nil || s.DB == nil || s.Mandates == nil || s.Authorizations == nil || s.Budgets == nil || s.Clock == nil || !sha256Digest.MatchString(request.TransactionDigest) {
		return ErrInvalid
	}
	if request.SigningPurpose != "" && request.SigningPurpose != "mandate-session" && request.SigningPurpose != "operator-withdrawal" {
		return ErrInvalid
	}
	expectedAtoms, err := wallet.AtomicAmountForUSDCMinor(request.ValueMinor, request.Currency)
	if err != nil || expectedAtoms != request.AtomicAmount {
		return fmt.Errorf("%w: Base USDC atom amount does not match exact USD minor valuation", ErrInvalid)
	}
	// A durable identical signing intent may be replayed to recover a crash
	// after admission. It can never authorize a changed transaction and lets the
	// deterministic signer reconstruct the same raw transaction after restart.
	var priorMandate, priorWallet, priorDigest string
	err = s.DB.QueryRowContext(ctx, `SELECT mandate_id,wallet_id,transaction_digest FROM mandate_session_signatures WHERE authorization_id=?`, request.AuthorizationID).Scan(&priorMandate, &priorWallet, &priorDigest)
	if err == nil {
		if priorMandate != request.MandateID || priorWallet != request.WalletID || priorDigest != request.TransactionDigest {
			return ErrSessionReplay
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	now := s.Clock.Now().UTC()
	operatorWithdrawal := request.SigningPurpose == "operator-withdrawal"
	var record sessionKeyRecord
	if operatorWithdrawal {
		record, err = s.budgetWalletForSigning(ctx, request.WalletID)
	} else {
		record, err = s.sessionByMandate(ctx, request.MandateID, "active")
	}
	if err != nil {
		return fmt.Errorf("%w: signing wallet is not bound to a mandate or budget", ErrSessionInactive)
	}
	if !operatorWithdrawal {
		bound, bindErr := s.budgetWalletForSigning(ctx, request.WalletID)
		if bindErr != nil || bound.budgetID != record.budgetID || bound.operator != record.operator || bound.agent != record.agent || bound.run != record.run || bound.network != record.network {
			return fmt.Errorf("%w: mandate session does not control the requested derived budget wallet", ErrSessionInactive)
		}
		record.walletID = bound.walletID
	}
	if record.walletID != request.WalletID || record.network != request.Network || record.operator != request.OperatorIdentity || !operatorWithdrawal && !now.Before(record.expiresAt) {
		return fmt.Errorf("%w: wallet, network, operator or session expiry does not match signing request", ErrSessionInactive)
	}
	grant, err := s.Mandates.RequireLive(ctx, request.MandateID)
	if err != nil || grant.ID != request.MandateID || grant.BudgetID != record.budgetID || grant.AgentSubject != record.agent || grant.AgentRunID != record.run || !grant.AllowsCounterparty(request.Recipient) {
		return fmt.Errorf("%w: live mandate does not cover the bound wallet and recipient: %v", ErrSessionInactive, err)
	}
	authorizationRecord, err := s.Authorizations.Get(ctx, request.AuthorizationID)
	if err != nil {
		return fmt.Errorf("load settlement authorization before session signing: %w", err)
	}
	authorizationValue := request.AuthorizationValueMinor
	if authorizationValue == 0 {
		authorizationValue = request.ValueMinor
	}
	if authorizationRecord.Verdict != authorization.VerdictApproved || !now.Before(authorizationRecord.ExpiresAt) || authorizationRecord.MandateID != request.MandateID || authorizationRecord.BudgetID != record.budgetID || authorizationRecord.RequestingAgent != record.agent || authorizationRecord.Counterparty != request.Recipient || authorizationRecord.AmountMinor != authorizationValue || authorizationRecord.Currency != request.Currency {
		return ErrInvalid
	}
	policy, err := s.Budgets.Get(ctx, record.budgetID)
	if err != nil || policy.Currency != request.Currency || !policy.AllowsCounterparty(request.Recipient) {
		return ErrInvalid
	}
	frozen, _, err := s.Budgets.IsFrozen(ctx, grant.BookID, record.budgetID)
	if err != nil {
		return fmt.Errorf("recheck budget freeze before mandate signing: %w", err)
	}
	if frozen {
		return fmt.Errorf("%w: budget freeze blocks mandate session signing", ErrInvalid)
	}
	// Even an identical crash-recovery replay crosses the current admission
	// gate below. Reconstructing a deterministic signature after expiry,
	// revocation or freeze would otherwise turn an old one-use intent into a
	// way around the kill switch. Existing identical intents are idempotent only
	// while their mandate, authorization, session and budget policy remain live.
	return s.consumeSessionSign(ctx, request, record, grant.BookID, authorizationValue, operatorWithdrawal, now)
}

type transactionDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func (s *Service) consumeSessionSign(ctx context.Context, request wallet.SessionSignAuthorization, record sessionKeyRecord, bookID string, authorizationValue int64, operatorWithdrawal bool, now time.Time) error {
	db, ok := s.DB.(transactionDB)
	if !ok {
		return fmt.Errorf("%w: transactional freeze gate is unavailable", ErrInvalid)
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	// Acquire the same SQLite writer barrier used by authorization admission;
	// freeze and signing admission now have a single durable ordering point.
	if _, err = tx.ExecContext(ctx, `UPDATE authorization_admission_gate SET generation=generation+1 WHERE id=1`); err != nil {
		return fmt.Errorf("acquire session signing freeze gate: %w", err)
	}
	var frozen bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM budgets WHERE id=? AND (frozen=1 OR currency<>?)
		UNION ALL SELECT 1 FROM freeze_controls WHERE frozen=1 AND (
			(scope='scenario' AND scope_id='*') OR (scope='book' AND scope_id=?))
		UNION ALL SELECT 1 FROM budget_scope_entries WHERE budget_id=? AND effect='deny' AND counterparty=?
		UNION ALL SELECT 1 FROM budget_scope_entries WHERE budget_id=? AND effect='allow'
			AND NOT EXISTS (SELECT 1 FROM budget_scope_entries WHERE budget_id=? AND effect='allow' AND counterparty=?)
		UNION ALL SELECT 1 FROM mandates WHERE id=? AND (status<>'live' OR expires_at<=? OR budget_id<>? OR book_id<>?)
		UNION ALL SELECT 1 FROM authorizations WHERE id=? AND (
			verdict<>'approved' OR expires_at<=? OR mandate_id<>? OR budget_id<>? OR requesting_agent<>? OR
			amount_minor<>? OR currency<>? OR lower(counterparty)<>?)
	)`,
		record.budgetID, request.Currency, bookID,
		record.budgetID, strings.ToLower(request.Recipient), record.budgetID, record.budgetID, strings.ToLower(request.Recipient),
		request.MandateID, now.Format(time.RFC3339Nano), record.budgetID, bookID,
		request.AuthorizationID, now.Format(time.RFC3339Nano), request.MandateID, record.budgetID, record.agent, authorizationValue, request.Currency, strings.ToLower(request.Recipient)).Scan(&frozen); err != nil {
		return fmt.Errorf("recheck signing scope under freeze gate: %w", err)
	}
	if frozen {
		return fmt.Errorf("%w: current freeze or signing scope blocks authorization", ErrInvalid)
	}
	var relationshipInvalid bool
	if operatorWithdrawal {
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM budget_wallets WHERE budget_id=? AND wallet_id=? AND operator_identity=? AND network=? AND agent_subject=? AND agent_run_id=?)
			UNION ALL SELECT 1 WHERE NOT EXISTS(SELECT 1 FROM withdrawals WHERE authorization_id=? AND operator_identity=? AND state IN ('authorized','in_progress'))
		)`, record.budgetID, request.WalletID, request.OperatorIdentity, request.Network, record.agent, record.run, request.AuthorizationID, request.OperatorIdentity).Scan(&relationshipInvalid); err != nil {
			return fmt.Errorf("recheck operator withdrawal authorization under freeze gate: %w", err)
		}
	} else if err = tx.QueryRowContext(ctx, `SELECT NOT EXISTS(
		SELECT 1 FROM mandate_session_keys s
		JOIN mandate_agent_session_keys k ON k.mandate_id=s.mandate_id AND k.generation=s.generation
		JOIN mandate_agent_session_proofs p ON p.mandate_id=s.mandate_id AND p.generation=s.generation AND p.authorization_id=?
		JOIN budget_wallets b ON b.budget_id=s.budget_id AND b.wallet_id=?
		WHERE s.mandate_id=? AND s.state='active' AND s.expires_at>?
	)`, request.AuthorizationID, request.WalletID, request.MandateID, now.Format(time.RFC3339Nano)).Scan(&relationshipInvalid); err != nil {
		return fmt.Errorf("recheck mandate session under freeze gate: %w", err)
	}
	if relationshipInvalid {
		return fmt.Errorf("%w: signing wallet lacks its active mandate session or operator withdrawal authorization row", ErrSessionInactive)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO mandate_session_signatures(authorization_id,mandate_id,wallet_id,transaction_digest,consumed_at) VALUES(?,?,?,?,?)`,
		request.AuthorizationID, request.MandateID, request.WalletID, request.TransactionDigest, now.Format(time.RFC3339Nano))
	if err != nil {
		var priorMandate, priorWallet, priorDigest string
		if readErr := tx.QueryRowContext(ctx, `SELECT mandate_id,wallet_id,transaction_digest FROM mandate_session_signatures WHERE authorization_id=?`, request.AuthorizationID).Scan(&priorMandate, &priorWallet, &priorDigest); readErr == nil {
			if priorMandate == request.MandateID && priorWallet == request.WalletID && priorDigest == request.TransactionDigest {
				return tx.Commit()
			}
			return ErrSessionReplay
		}
		return fmt.Errorf("consume mandate signing authorization: %w", err)
	}
	return tx.Commit()
}

func (s *Service) sessionByMandate(ctx context.Context, mandateID, state string) (sessionKeyRecord, error) {
	if s == nil || s.DB == nil {
		return sessionKeyRecord{}, ErrInvalid
	}
	var record sessionKeyRecord
	var expiresAt string
	err := s.DB.QueryRowContext(ctx, `SELECT id,mandate_id,budget_id,wallet_id,operator_identity,agent_subject,agent_run_id,network,generation,state,expires_at FROM mandate_session_keys WHERE mandate_id=? AND state=?`, mandateID, state).Scan(
		&record.id, &record.mandateID, &record.budgetID, &record.walletID, &record.operator, &record.agent, &record.run, &record.network, &record.generation, &record.state, &expiresAt)
	if err != nil {
		return sessionKeyRecord{}, err
	}
	record.expiresAt, err = time.Parse(time.RFC3339Nano, expiresAt)
	return record, err
}

func (s *Service) publicSession(ctx context.Context, record sessionKeyRecord) (MandateSessionWallet, error) {
	b, err := s.load(ctx, record.budgetID)
	if err != nil {
		return MandateSessionWallet{}, err
	}
	w, err := s.Wallets.Get(ctx, b.walletID, record.network, record.operator)
	if err != nil {
		return MandateSessionWallet{}, err
	}
	return MandateSessionWallet{MandateID: record.mandateID, BudgetID: record.budgetID, WalletID: b.walletID, OperatorIdentity: record.operator, AgentSubject: record.agent, AgentRunID: record.run, Network: record.network, Address: w.Address, Generation: record.generation, ExpiresAt: record.expiresAt}, nil
}

func sameSessionBinding(record sessionKeyRecord, b binding, grant mandate.Mandate) bool {
	return record.budgetID == b.budgetID && record.operator == b.operator && record.agent == grant.AgentSubject && record.run == grant.AgentRunID && record.network == b.network && !record.expiresAt.After(grant.ExpiresAt)
}
