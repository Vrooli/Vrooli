package main

// Credential use is deliberately separate from the general HTTP broker. A
// browser session has an origin and document binding; an SSH signer has a
// destination and principal binding. Neither operation returns a stored
// credential value to the caller.

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	gossh "golang.org/x/crypto/ssh"
)

type browserExposureClass string

const (
	browserExposureProtected    browserExposureClass = "protected_browser"
	browserExposureUnrestricted browserExposureClass = "unrestricted_session"
)

type browserSession struct {
	ID, Workspace, Actor, RunID, GrantID, ItemID, Origin, Account, DocumentID, Status string
	Exposure                                                                          browserExposureClass
	RecoveryEpoch                                                                     uint64
	Expires                                                                           time.Time
	Filled                                                                            []string
}

type browserUseInput struct {
	GrantID         string   `json:"grant_id"`
	ItemID          string   `json:"item_id"`
	RunID           string   `json:"run_id,omitempty"`
	Origin          string   `json:"origin"`
	Account         string   `json:"account"`
	DocumentID      string   `json:"document_id"`
	Exposure        string   `json:"exposure_class"`
	Allowed         []string `json:"allowed_operations"`
	DurationSeconds int      `json:"duration_seconds"`
}

type browserActionInput struct {
	Action        string `json:"action"`
	Origin        string `json:"origin"`
	DocumentID    string `json:"document_id"`
	NavigationURL string `json:"navigation_url"`
	Success       bool   `json:"success"`
}

type browserActionResult struct {
	SessionID      string               `json:"session_id"`
	Status         string               `json:"status"`
	Origin         string               `json:"origin"`
	DocumentID     string               `json:"document_id"`
	Exposure       browserExposureClass `json:"exposure_class"`
	ClearedFields  []string             `json:"cleared_fields,omitempty"`
	OperationClass string               `json:"operation_class"`
}

// trustedBrowserExecutor receives plaintext only inside the trusted executor
// boundary. Its methods return metadata, never field values, DOM, cookies, or
// evaluation results. A nil executor is an explicit unsupported state.
type trustedBrowserExecutor interface {
	Fill(context.Context, browserSession, map[string]string) ([]string, error)
	Clear(context.Context, browserSession, []string) error
	Close(context.Context, browserSession) error
}

func (p *PasswordManager) SetTrustedBrowserExecutor(executor trustedBrowserExecutor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.browserExec = executor
}

func normalizeCredentialOrigin(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("credential origin must be an absolute http(s) origin")
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), nil
}

func (p *PasswordManager) createBrowserSession(ctx context.Context, workspace, actor string, input browserUseInput) (browserSession, error) {
	grant, err := p.getGrant(ctx, workspace, input.GrantID)
	if err != nil || !p.grantAllows(ctx, grant, actor, input.ItemID, "inject", time.Now().UTC()) {
		return browserSession{}, errAccessDenied
	}
	origin, err := normalizeCredentialOrigin(input.Origin)
	if err != nil {
		return browserSession{}, err
	}
	grantOrigin, err := normalizeCredentialOrigin(grant.Target)
	if err != nil || grantOrigin != origin {
		return browserSession{}, errAccessDenied
	}
	if strings.TrimSpace(input.Account) == "" || strings.TrimSpace(input.DocumentID) == "" {
		return browserSession{}, errors.New("browser account and document_id are required")
	}
	exposure := browserExposureClass(strings.TrimSpace(input.Exposure))
	if exposure == "" {
		exposure = browserExposureProtected
	}
	if exposure != browserExposureProtected && exposure != browserExposureUnrestricted {
		return browserSession{}, fmt.Errorf("unsupported browser exposure class: %s", exposure)
	}
	if exposure == browserExposureUnrestricted {
		return browserSession{}, errors.New("unrestricted browser sessions require an explicit broader exposure class")
	}
	if !contains(input.Allowed, "login") {
		return browserSession{}, errors.New("browser login operation must be explicitly allowed")
	}
	item, err := p.loadItemRecord(ctx, workspace, grant.VaultID, input.ItemID)
	if err != nil {
		return browserSession{}, err
	}
	if item.Meta.Username != input.Account {
		return browserSession{}, errAccessDenied
	}
	if input.DurationSeconds <= 0 || input.DurationSeconds > 300 {
		input.DurationSeconds = 120
	}
	recoveryEpoch, err := p.currentRecoveryEpoch(ctx, workspace)
	if err != nil {
		return browserSession{}, err
	}
	session := browserSession{
		ID: uuid.NewString(), Workspace: workspace, Actor: actor, RunID: strings.TrimSpace(input.RunID), GrantID: grant.ID,
		ItemID: input.ItemID, Origin: origin, Account: input.Account, DocumentID: input.DocumentID,
		Exposure: exposure, Status: "active", RecoveryEpoch: recoveryEpoch, Expires: time.Now().UTC().Add(time.Duration(input.DurationSeconds) * time.Second),
	}
	p.mu.Lock()
	p.browsers[session.ID] = session
	p.mu.Unlock()
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "browser.session.create", ItemID: input.ItemID, Outcome: "success", Detail: "origin and document bound trusted executor"})
	return session, nil
}

func (p *PasswordManager) browserSessionFor(workspace, actor, id string) (browserSession, error) {
	current, epochErr := p.currentRecoveryEpoch(context.Background(), workspace)
	p.mu.Lock()
	session, ok := p.browsers[id]
	if !ok || session.Workspace != workspace || session.Actor != actor || session.Status != "active" {
		p.mu.Unlock()
		return browserSession{}, errAccessDenied
	}
	if epochErr != nil || session.RecoveryEpoch != current {
		session.Status = "recovery_invalidated"
		p.browsers[id] = session
		executor := p.browserExec
		p.mu.Unlock()
		if executor != nil {
			_ = executor.Close(context.Background(), session)
		}
		return browserSession{}, errAccessDenied
	}
	if time.Now().UTC().After(session.Expires) {
		session.Status = "expired"
		p.browsers[id] = session
		executor := p.browserExec
		p.mu.Unlock()
		if executor != nil {
			_ = executor.Close(context.Background(), session)
		}
		return browserSession{}, errAccessDenied
	}
	p.mu.Unlock()
	return session, nil
}

func (p *PasswordManager) browserAction(ctx context.Context, workspace, actor, id string, input browserActionInput) (browserActionResult, error) {
	p.credentialUseMu.Lock()
	defer p.credentialUseMu.Unlock()
	return p.browserActionLocked(ctx, workspace, actor, id, input)
}

func (p *PasswordManager) browserActionLocked(ctx context.Context, workspace, actor, id string, input browserActionInput) (browserActionResult, error) {
	session, err := p.browserSessionFor(workspace, actor, id)
	if err != nil {
		return browserActionResult{}, err
	}
	switch input.Action {
	case "cookies", "evaluate", "secret_field", "dom":
		return browserActionResult{}, fmt.Errorf("unsupported browser operation %q: protected executor denies cookies, DOM, and secret-field evaluation", input.Action)
	}
	currentOrigin, err := normalizeCredentialOrigin(input.Origin)
	if err != nil || currentOrigin != session.Origin {
		return browserActionResult{}, errBrowserAuthorizationRequired
	}
	if strings.TrimSpace(input.DocumentID) == "" || input.DocumentID != session.DocumentID {
		return browserActionResult{}, errBrowserDocumentMismatch
	}
	p.mu.Lock()
	executor := p.browserExec
	p.mu.Unlock()
	if executor == nil {
		return browserActionResult{}, errCredentialExecutorUnsupported
	}

	switch input.Action {
	case "navigate":
		navigationOrigin, navErr := normalizeCredentialOrigin(input.NavigationURL)
		if navErr != nil || navigationOrigin != session.Origin {
			return browserActionResult{}, errBrowserAuthorizationRequired
		}
		return p.browserMetadata(session, "navigation_allowed", nil), nil
	case "fill":
		fields, fieldErr := p.browserSecretFields(ctx, session)
		if fieldErr != nil {
			return browserActionResult{}, fieldErr
		}
		filled, fillErr := executor.Fill(ctx, session, fields)
		for name := range fields {
			fields[name] = ""
		}
		if fillErr != nil {
			return browserActionResult{}, fillErr
		}
		p.mu.Lock()
		current := p.browsers[session.ID]
		current.Filled = append([]string(nil), filled...)
		p.browsers[session.ID] = current
		p.mu.Unlock()
		return p.browserMetadata(session, "trusted_fill", nil), nil
	case "submit":
		if input.Success {
			return p.browserMetadata(session, "login_submitted", nil), nil
		}
		p.mu.Lock()
		current := p.browsers[session.ID]
		filled := append([]string(nil), current.Filled...)
		current.Filled = nil
		p.browsers[session.ID] = current
		p.mu.Unlock()
		if clearErr := executor.Clear(ctx, session, filled); clearErr != nil {
			return browserActionResult{}, clearErr
		}
		return p.browserMetadata(session, "login_failed_fields_cleared", filled), nil
	case "complete":
		return p.closeBrowserSessionLocked(ctx, workspace, actor, id, "run_complete")
	default:
		return browserActionResult{}, fmt.Errorf("unsupported browser operation: %s", input.Action)
	}
}

func (p *PasswordManager) browserSecretFields(ctx context.Context, session browserSession) (map[string]string, error) {
	grant, err := p.getGrant(ctx, session.Workspace, session.GrantID)
	if err != nil || !p.grantAllows(ctx, grant, session.Actor, session.ItemID, "inject", time.Now().UTC()) {
		return nil, errAccessDenied
	}
	item, err := p.loadItemRecord(ctx, session.Workspace, grant.VaultID, session.ItemID)
	if err != nil {
		return nil, err
	}
	vault, err := p.ensureVault(ctx, session.Workspace, grant.VaultID)
	if err != nil {
		return nil, err
	}
	if vault.Status == "locked" {
		return nil, errVaultLocked
	}
	fields, err := p.decrypt(grant.VaultID, session.ItemID, item.Ciphertext)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for _, name := range []string{"username", "password"} {
		if value := fields[name]; value != "" {
			result[name] = value
		}
	}
	if value := fields["totp"]; value != "" {
		code, codeErr := generateTOTPCode(value, time.Now().UTC(), 30, 6, "SHA1")
		if codeErr != nil {
			return nil, codeErr
		}
		result["totp"] = code
	}
	for name := range fields {
		fields[name] = ""
	}
	return result, nil
}

func (p *PasswordManager) loadItemRecord(ctx context.Context, workspace, vaultID, itemID string) (vaultRecord, error) {
	if p.memory {
		p.mu.RLock()
		item, ok := p.items[itemID]
		p.mu.RUnlock()
		if !ok || item.Workspace != workspace || item.Meta.VaultID != vaultID {
			return vaultRecord{}, sql.ErrNoRows
		}
		return item, nil
	}
	var item vaultRecord
	var tags, created, updated string
	if err := p.db.QueryRowContext(ctx, `SELECT id,vault_id,item_type,name,username,uri,folder,tags,favorite,trashed,version,encrypted_payload,created_at,updated_at FROM pm_items WHERE id=$1 AND workspace_id=$2 AND vault_id=$3`, itemID, workspace, vaultID).Scan(&item.Meta.ID, &item.Meta.VaultID, &item.Meta.Type, &item.Meta.Name, &item.Meta.Username, &item.Meta.URI, &item.Meta.Folder, &tags, &item.Meta.Favorite, &item.Meta.Trashed, &item.Meta.Revision, &item.Ciphertext, &created, &updated); err != nil {
		return vaultRecord{}, err
	}
	_ = json.Unmarshal([]byte(tags), &item.Meta.Tags)
	item.Meta.CreatedAt, item.Meta.UpdatedAt, item.Workspace = parseTime(created), parseTime(updated), workspace
	return item, nil
}

func (p *PasswordManager) browserMetadata(session browserSession, operation string, cleared []string) browserActionResult {
	return browserActionResult{SessionID: session.ID, Status: "active", Origin: session.Origin, DocumentID: session.DocumentID, Exposure: session.Exposure, ClearedFields: append([]string(nil), cleared...), OperationClass: operation}
}

func (p *PasswordManager) closeBrowserSession(ctx context.Context, workspace, actor, id, reason string) (browserActionResult, error) {
	p.credentialUseMu.Lock()
	defer p.credentialUseMu.Unlock()
	return p.closeBrowserSessionLocked(ctx, workspace, actor, id, reason)
}

func (p *PasswordManager) closeBrowserSessionLocked(ctx context.Context, workspace, actor, id, reason string) (browserActionResult, error) {
	session, err := p.browserSessionFor(workspace, actor, id)
	if err != nil {
		return browserActionResult{}, err
	}
	p.mu.Lock()
	executor := p.browserExec
	session.Status = "closed"
	session.Filled = nil
	p.browsers[id] = session
	p.mu.Unlock()
	if executor != nil {
		if closeErr := executor.Close(ctx, session); closeErr != nil {
			return browserActionResult{}, closeErr
		}
	}
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "browser.session.close", ItemID: session.ItemID, Outcome: "success", Detail: reason})
	return p.browserMetadata(session, "closed", nil), nil
}

// terminateCredentialUse is called by grant and vault lifecycle transitions.
// Handles are invalidated before the trusted executor is asked to close its
// browser session, so a concurrent agent request cannot win the race with
// revocation.
func (p *PasswordManager) terminateCredentialUse(ctx context.Context, workspace, grantID, vaultID, reason string) {
	p.credentialUseMu.Lock()
	defer p.credentialUseMu.Unlock()
	p.mu.Lock()
	executor := p.browserExec
	toClose := make([]browserSession, 0)
	for id, session := range p.browsers {
		matchesGrant := grantID != "" && session.GrantID == grantID
		matchesVault := false
		if vaultID != "" {
			if grant, ok := p.grants[session.GrantID]; ok {
				matchesVault = grant.VaultID == vaultID
			}
		}
		if session.Workspace == workspace && session.Status == "active" && (matchesGrant || matchesVault) {
			session.Status = reason
			session.Filled = nil
			p.browsers[id] = session
			toClose = append(toClose, session)
		}
	}
	for id, session := range p.sshSigners {
		if session.Workspace == workspace && session.Status == "active" && ((grantID != "" && session.GrantID == grantID) || (vaultID != "" && p.grants[session.GrantID].VaultID == vaultID)) {
			session.Status = reason
			p.sshSigners[id] = session
		}
	}
	p.mu.Unlock()
	if executor != nil {
		for _, session := range toClose {
			_ = executor.Close(ctx, session)
		}
	}
}

// terminateCredentialUseForRun invalidates every capability issued to one
// verified Agent Manager run. The handle state is changed before any trusted
// executor callback, so a terminal run cannot race a late fill or signature.
func (p *PasswordManager) terminateCredentialUseForRun(ctx context.Context, workspace, runID, reason string) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}
	p.credentialUseMu.Lock()
	defer p.credentialUseMu.Unlock()
	p.mu.Lock()
	executor := p.browserExec
	toClose := make([]browserSession, 0)
	for id, session := range p.browsers {
		if session.Workspace == workspace && session.RunID == runID && session.Status == "active" {
			session.Status = reason
			session.Filled = nil
			p.browsers[id] = session
			toClose = append(toClose, session)
		}
	}
	for id, session := range p.sshSigners {
		if session.Workspace == workspace && session.RunID == runID && session.Status == "active" {
			session.Status = reason
			p.sshSigners[id] = session
		}
	}
	for id, session := range p.brokers {
		if session.Workspace == workspace && session.RunID == runID && session.Status == "active" {
			session.Status = reason
			p.brokers[id] = session
		}
	}
	p.mu.Unlock()
	if p.db != nil && !p.memory {
		_, _ = p.db.ExecContext(ctx, `UPDATE pm_broker_sessions SET status=$1 WHERE workspace_id=$2 AND run_id=$3 AND status='active'`, reason, workspace, runID)
	}
	if executor != nil {
		for _, session := range toClose {
			_ = executor.Close(ctx, session)
		}
	}
}

type sshSignerSession struct {
	ID, Workspace, Actor, RunID, GrantID, ItemID, Destination, Principal, Status string
	RecoveryEpoch                                                                uint64
	Expires                                                                      time.Time
}

type sshSignerInput struct {
	GrantID         string `json:"grant_id"`
	ItemID          string `json:"item_id"`
	RunID           string `json:"run_id,omitempty"`
	Destination     string `json:"destination"`
	Principal       string `json:"principal"`
	DurationSeconds int    `json:"duration_seconds"`
}

type sshSignInput struct {
	Challenge string `json:"challenge"`
}

type sshSignResult struct {
	SignerID  string    `json:"signer_id"`
	Algorithm string    `json:"algorithm"`
	Signature string    `json:"signature"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (p *PasswordManager) createSSHSigner(ctx context.Context, workspace, actor string, input sshSignerInput) (sshSignerSession, error) {
	grant, err := p.getGrant(ctx, workspace, input.GrantID)
	if err != nil || !p.grantAllows(ctx, grant, actor, input.ItemID, "sign", time.Now().UTC()) {
		return sshSignerSession{}, errAccessDenied
	}
	destination, principal := strings.TrimSpace(input.Destination), strings.TrimSpace(input.Principal)
	if destination == "" || principal == "" || strings.ContainsAny(destination, "\r\n") || strings.ContainsAny(principal, "\r\n") || grant.Target == "" || !strings.EqualFold(strings.TrimSpace(grant.Target), destination) {
		return sshSignerSession{}, errSSHDestinationDenied
	}
	if input.DurationSeconds <= 0 || input.DurationSeconds > 300 {
		input.DurationSeconds = 120
	}
	recoveryEpoch, err := p.currentRecoveryEpoch(ctx, workspace)
	if err != nil {
		return sshSignerSession{}, err
	}
	item, err := p.loadItemRecord(ctx, workspace, grant.VaultID, input.ItemID)
	if err != nil {
		return sshSignerSession{}, err
	}
	vault, err := p.ensureVault(ctx, workspace, grant.VaultID)
	if err != nil {
		return sshSignerSession{}, err
	}
	if vault.Status == "locked" {
		return sshSignerSession{}, errVaultLocked
	}
	fields, err := p.decrypt(grant.VaultID, input.ItemID, item.Ciphertext)
	if err != nil {
		return sshSignerSession{}, errAccessDenied
	}
	boundPrincipal := strings.TrimSpace(fields["principal"])
	if boundPrincipal == "" {
		boundPrincipal = strings.TrimSpace(fields["username"])
	}
	for key := range fields {
		fields[key] = ""
	}
	if boundPrincipal != "" && boundPrincipal != principal {
		return sshSignerSession{}, errSSHDestinationDenied
	}
	session := sshSignerSession{ID: uuid.NewString(), Workspace: workspace, Actor: actor, RunID: strings.TrimSpace(input.RunID), GrantID: grant.ID, ItemID: input.ItemID, Destination: destination, Principal: principal, Status: "active", RecoveryEpoch: recoveryEpoch, Expires: time.Now().UTC().Add(time.Duration(input.DurationSeconds) * time.Second)}
	p.mu.Lock()
	p.sshSigners[session.ID] = session
	p.mu.Unlock()
	return session, nil
}

func (p *PasswordManager) signSSH(ctx context.Context, workspace, actor, id string, input sshSignInput) (sshSignResult, error) {
	p.credentialUseMu.Lock()
	defer p.credentialUseMu.Unlock()
	p.mu.RLock()
	session, ok := p.sshSigners[id]
	p.mu.RUnlock()
	currentEpoch, epochErr := p.currentRecoveryEpoch(ctx, workspace)
	if epochErr != nil || !ok || session.Workspace != workspace || session.Actor != actor || session.Status != "active" || session.RecoveryEpoch != currentEpoch || time.Now().UTC().After(session.Expires) {
		return sshSignResult{}, errAccessDenied
	}
	challenge, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(input.Challenge))
	if err != nil || len(challenge) == 0 || len(challenge) > 64*1024 {
		return sshSignResult{}, errors.New("ssh challenge must be base64url and between 1 byte and 64 KiB")
	}
	grant, err := p.getGrant(ctx, workspace, session.GrantID)
	if err != nil || !p.grantAllows(ctx, grant, actor, session.ItemID, "sign", time.Now().UTC()) {
		return sshSignResult{}, errAccessDenied
	}
	item, err := p.loadItemRecord(ctx, workspace, grant.VaultID, session.ItemID)
	if err != nil {
		return sshSignResult{}, errAccessDenied
	}
	vault, err := p.ensureVault(ctx, workspace, grant.VaultID)
	if err != nil {
		return sshSignResult{}, err
	}
	if vault.Status == "locked" {
		return sshSignResult{}, errVaultLocked
	}
	fields, err := p.decrypt(grant.VaultID, session.ItemID, item.Ciphertext)
	if err != nil {
		return sshSignResult{}, errAccessDenied
	}
	privateKey := []byte(fields["private_key"])
	if len(privateKey) == 0 {
		privateKey = []byte(fields["key"])
	}
	var signer gossh.Signer
	if passphrase := fields["passphrase"]; passphrase != "" {
		signer, err = gossh.ParsePrivateKeyWithPassphrase(privateKey, []byte(passphrase))
	} else {
		signer, err = gossh.ParsePrivateKey(privateKey)
	}
	for i := range privateKey {
		privateKey[i] = 0
	}
	for key := range fields {
		fields[key] = ""
	}
	if err != nil {
		return sshSignResult{}, errAccessDenied
	}
	signature, err := signer.Sign(rand.Reader, challenge)
	if err != nil {
		return sshSignResult{}, errAccessDenied
	}
	encoded := base64.RawStdEncoding.EncodeToString(append([]byte(signature.Format), signature.Blob...))
	return sshSignResult{SignerID: session.ID, Algorithm: signer.PublicKey().Type(), Signature: encoded, ExpiresAt: session.Expires}, nil
}

func (h *passwordManagerHandlers) credentialUseRegisterRoutes(r *mux.Router) {
	r.HandleFunc("/credential-use/runs/{run_id}/revoke", h.revokeRunCredentialUse).Methods(http.MethodPost)
	r.HandleFunc("/credential-use/browser/sessions", h.createBrowserSession).Methods(http.MethodPost)
	r.HandleFunc("/credential-use/browser/sessions/{id}/actions", h.browserAction).Methods(http.MethodPost)
	r.HandleFunc("/credential-use/browser/sessions/{id}/close", h.closeBrowserSession).Methods(http.MethodPost)
	r.HandleFunc("/credential-use/ssh/signers", h.createSSHSigner).Methods(http.MethodPost)
	r.HandleFunc("/credential-use/ssh/signers/{id}/sign", h.signSSH).Methods(http.MethodPost)
}

func (h *passwordManagerHandlers) revokeRunCredentialUse(w http.ResponseWriter, r *http.Request) {
	auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext)
	if !ok || auth.Role != "agent" || auth.RunID == "" || auth.RunID != strings.TrimSpace(mux.Vars(r)["run_id"]) {
		writeVaultError(w, errAccessDenied)
		return
	}
	h.manager.terminateCredentialUseForRun(r.Context(), auth.WorkspaceID, auth.RunID, "run_terminal")
	writeJSON(w, http.StatusOK, map[string]any{"run_id": auth.RunID, "status": "revoked"})
}

func (h *passwordManagerHandlers) createBrowserSession(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input browserUseInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.Role == "agent" {
		input.RunID = auth.RunID
	}
	session, err := h.manager.createBrowserSession(r.Context(), workspace, actor, input)
	if err != nil {
		h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.browser.session", input.ItemID, "failure", err.Error()))
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.browser.session", input.ItemID, "success", "session issued"))
	writeJSON(w, http.StatusCreated, map[string]any{"session_id": session.ID, "origin": session.Origin, "account": session.Account, "document_id": session.DocumentID, "exposure_class": session.Exposure, "expires_at": session.Expires})
}

func (h *passwordManagerHandlers) browserAction(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input browserActionInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	result, err := h.manager.browserAction(r.Context(), workspace, actor, mux.Vars(r)["id"], input)
	if err != nil {
		h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.browser."+input.Action, "", "failure", err.Error()))
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.browser."+input.Action, "", "success", result.OperationClass))
	writeJSON(w, http.StatusOK, result)
}

func (h *passwordManagerHandlers) closeBrowserSession(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	result, err := h.manager.closeBrowserSession(r.Context(), workspace, actor, mux.Vars(r)["id"], "explicit_close")
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *passwordManagerHandlers) createSSHSigner(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input sshSignerInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.Role == "agent" {
		input.RunID = auth.RunID
	}
	session, err := h.manager.createSSHSigner(r.Context(), workspace, actor, input)
	if err != nil {
		h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.ssh.signer", input.ItemID, "failure", err.Error()))
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.ssh.signer", input.ItemID, "success", "signer issued"))
	writeJSON(w, http.StatusCreated, map[string]any{"signer_id": session.ID, "destination": session.Destination, "principal": session.Principal, "expires_at": session.Expires, "secret_exposure": "sign_only"})
}

func (h *passwordManagerHandlers) signSSH(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	var input sshSignInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	result, err := h.manager.signSSH(r.Context(), workspace, actor, mux.Vars(r)["id"], input)
	if err != nil {
		h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.ssh.sign", "", "failure", err.Error()))
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), credentialUseAuditEvent(r, workspace, actor, "credential_use.ssh.sign", "", "success", "signature issued"))
	writeJSON(w, http.StatusOK, result)
}

func credentialUseAuditEvent(r *http.Request, workspace, actor, action, itemID, outcome, detail string) auditEvent {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("X-Request-ID"))
	}
	if key == "" {
		key = uuid.NewString()
	}
	return auditEvent{WorkspaceID: workspace, ActorID: actor, Action: action, ItemID: itemID, RequestID: key, CorrelationID: key, EventKey: action + ":" + key, Outcome: outcome, Detail: detail}
}
