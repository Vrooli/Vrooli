package main

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type brokerSession struct {
	ID, TokenDigest, Workspace, Actor, RunID, GrantID, ItemID, Origin, Status string
	ResolvedIPs                                                               []string
	OneUse                                                                    bool
	UseCount                                                                  int
	RecoveryEpoch                                                             uint64
	Expires                                                                   time.Time
}

type brokerSessionInput struct {
	GrantID       string `json:"grant_id"`
	ItemID        string `json:"item_id"`
	RunID         string `json:"run_id,omitempty"`
	TargetOrigin  string `json:"target_origin"`
	AllowInternal bool   `json:"allow_internal"`
	DurationSec   int    `json:"duration_seconds"`
	OneUse        bool   `json:"one_use"`
}

type brokerOperationInput struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type brokerOperationResponse struct {
	Status        int               `json:"status"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body,omitempty"`
	BodyTruncated bool              `json:"body_truncated"`
	Projection    string            `json:"projection"`
}

func (p *PasswordManager) createBrokerSession(ctx context.Context, workspace, actor string, input brokerSessionInput) (brokerSession, string, error) {
	grant, err := p.getGrant(ctx, workspace, input.GrantID)
	if err != nil {
		return brokerSession{}, "", err
	}
	if !p.grantAllows(ctx, grant, actor, input.ItemID, "use", time.Now().UTC()) {
		return brokerSession{}, "", errAccessDenied
	}
	origin, resolvedIPs, err := resolveBrokerOrigin(input.TargetOrigin, input.AllowInternal)
	if err != nil {
		return brokerSession{}, "", err
	}
	if strings.TrimSpace(grant.Target) == "" {
		return brokerSession{}, "", errors.New("broker grant must pin a target origin")
	}
	grantOrigin, _, err := resolveBrokerOrigin(grant.Target, input.AllowInternal)
	if err != nil || grantOrigin != origin {
		return brokerSession{}, "", errAccessDenied
	}
	if input.DurationSec <= 0 || input.DurationSec > 300 {
		input.DurationSec = 120
	}
	recoveryEpoch, err := p.currentRecoveryEpoch(ctx, workspace)
	if err != nil {
		return brokerSession{}, "", err
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(cryptorand.Reader, raw); err != nil {
		return brokerSession{}, "", err
	}
	// The session token is returned once as a capability. Only its digest is
	// stored, so a metadata read cannot recreate it.
	token := hex.EncodeToString(raw)
	digest := sha256.Sum256([]byte(token))
	session := brokerSession{ID: uuid.NewString(), TokenDigest: hex.EncodeToString(digest[:]), Workspace: workspace, Actor: actor, RunID: strings.TrimSpace(input.RunID), GrantID: grant.ID, ItemID: input.ItemID, Origin: origin, ResolvedIPs: resolvedIPs, Status: "active", OneUse: input.OneUse, RecoveryEpoch: recoveryEpoch, Expires: time.Now().UTC().Add(time.Duration(input.DurationSec) * time.Second)}
	if p.memory {
		p.mu.Lock()
		p.brokers[session.ID] = session
		p.mu.Unlock()
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "broker.session.create", ItemID: session.ItemID, Outcome: "success", Detail: "origin-bound bounded session"})
		return session, token, nil
	}
	_, err = p.db.ExecContext(ctx, `INSERT INTO pm_broker_sessions(id,token_digest,workspace_id,actor_id,run_id,grant_id,item_id,target_origin,resolved_ips,expires_at,status,one_use,use_count,recovery_epoch) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'active',$11,0,$12)`, session.ID, session.TokenDigest, session.Workspace, session.Actor, session.RunID, session.GrantID, session.ItemID, session.Origin, strings.Join(session.ResolvedIPs, ","), session.Expires, session.OneUse, session.RecoveryEpoch)
	if err == nil {
		p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "broker.session.create", ItemID: session.ItemID, Outcome: "success", Detail: "origin-bound bounded session"})
	}
	return session, token, err
}

func (p *PasswordManager) brokerSessionFor(ctx context.Context, workspace, actor, token string) (brokerSession, error) {
	digest := sha256.Sum256([]byte(token))
	digestText := hex.EncodeToString(digest[:])
	if p.memory {
		p.mu.RLock()
		var found brokerSession
		for _, session := range p.brokers {
			if session.TokenDigest == digestText {
				found = session
				break
			}
		}
		p.mu.RUnlock()
		current, epochErr := p.currentRecoveryEpoch(ctx, workspace)
		if epochErr != nil || found.ID == "" || found.Workspace != workspace || found.Actor != actor || found.Status != "active" || found.RecoveryEpoch != current || found.OneUse && found.UseCount > 0 || time.Now().After(found.Expires) {
			return brokerSession{}, errAccessDenied
		}
		return found, nil
	}
	var session brokerSession
	var resolvedIPs string
	err := p.db.QueryRowContext(ctx, `SELECT id,token_digest,workspace_id,actor_id,run_id,grant_id,item_id,target_origin,resolved_ips,expires_at,status,one_use,use_count,recovery_epoch FROM pm_broker_sessions WHERE token_digest=$1`, digestText).Scan(&session.ID, &session.TokenDigest, &session.Workspace, &session.Actor, &session.RunID, &session.GrantID, &session.ItemID, &session.Origin, &resolvedIPs, &session.Expires, &session.Status, &session.OneUse, &session.UseCount, &session.RecoveryEpoch)
	session.ResolvedIPs = splitResolvedIPs(resolvedIPs)
	current, epochErr := p.currentRecoveryEpoch(ctx, workspace)
	if err != nil || epochErr != nil || session.Workspace != workspace || session.Actor != actor || session.Status != "active" || session.RecoveryEpoch != current || session.OneUse && session.UseCount > 0 || time.Now().After(session.Expires) {
		return brokerSession{}, errAccessDenied
	}
	return session, nil
}

func (p *PasswordManager) brokerSecret(ctx context.Context, workspace, itemID string) (string, error) {
	var record vaultRecord
	if p.memory {
		p.mu.RLock()
		record = p.items[itemID]
		p.mu.RUnlock()
		if record.ID() == "" || record.Workspace != workspace {
			return "", errAccessDenied
		}
	} else {
		var encrypted, vaultID string
		if err := p.db.QueryRowContext(ctx, `SELECT vault_id,encrypted_payload FROM pm_items WHERE id=$1 AND workspace_id=$2 AND trashed=FALSE`, itemID, workspace).Scan(&vaultID, &encrypted); err != nil {
			return "", err
		}
		record = vaultRecord{Meta: VaultItem{ID: itemID, VaultID: vaultID}, Workspace: workspace, Ciphertext: encrypted}
	}
	vault, err := p.ensureVault(ctx, workspace, record.Meta.VaultID)
	if err != nil {
		return "", err
	}
	if vault.Status == "locked" {
		return "", errVaultLocked
	}
	fields, err := p.decrypt(record.Meta.VaultID, itemID, record.Ciphertext)
	if err != nil {
		return "", err
	}
	for _, field := range []string{"token", "api_key", "password", "secret"} {
		if value := fields[field]; value != "" {
			return value, nil
		}
	}
	return "", errors.New("item has no supported broker credential field")
}

// ID gives broker code a metadata-safe existence check without exposing a
// payload or adding a second item representation.
func (r vaultRecord) ID() string { return r.Meta.ID }

func validateBrokerOrigin(raw string, allowInternal bool) (string, error) {
	origin, _, err := resolveBrokerOrigin(raw, allowInternal)
	return origin, err
}

func resolveBrokerOrigin(raw string, allowInternal bool) (string, []string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "http" && parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return "", nil, errors.New("broker target must be an absolute http(s) origin without embedded credentials")
	}
	if parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", nil, errors.New("broker target must contain only an origin")
	}
	host := parsed.Hostname()
	if host == "" {
		return "", nil, errors.New("broker target must contain a hostname")
	}
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		if !allowInternal {
			return "", nil, errors.New("internal broker destinations require explicit policy")
		}
	}
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		ips, err = net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return "", nil, errors.New("broker destination could not be resolved")
		}
	}
	resolved := make([]string, 0, len(ips))
	seen := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		if isPrivateIP(ip) && !allowInternal {
			return "", nil, errors.New("broker destination resolves to a private address")
		}
		value := ip.String()
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			resolved = append(resolved, value)
		}
	}
	return strings.TrimRight(parsed.String(), "/"), resolved, nil
}

func splitResolvedIPs(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" && net.ParseIP(value) != nil {
			result = append(result, value)
		}
	}
	return result
}

func isPrivateIP(ip net.IP) bool {
	privateRanges := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("127.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
	}
	for _, network := range privateRanges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func (p *PasswordManager) brokerOperation(ctx context.Context, workspace, actor, sessionID, token string, input brokerOperationInput) (brokerOperationResponse, error) {
	session, err := p.brokerSessionFor(ctx, workspace, actor, token)
	if err != nil {
		return brokerOperationResponse{}, err
	}
	if session.ID != sessionID {
		return brokerOperationResponse{}, errAccessDenied
	}
	if err := p.claimBrokerSession(ctx, session); err != nil {
		return brokerOperationResponse{}, err
	}
	grant, err := p.getGrant(ctx, workspace, session.GrantID)
	if err != nil || !p.grantAllows(ctx, grant, actor, session.ItemID, "use", time.Now().UTC()) {
		return brokerOperationResponse{}, errAccessDenied
	}
	method := strings.ToUpper(strings.TrimSpace(input.Method))
	switch method {
	case http.MethodGet, http.MethodHead:
	default:
		return brokerOperationResponse{}, errors.New("unsupported broker method: only read-only GET or HEAD requests are permitted")
	}
	if input.Path == "" || !strings.HasPrefix(input.Path, "/") || strings.Contains(input.Path, "//") || strings.Contains(input.Path, "..") {
		return brokerOperationResponse{}, errors.New("broker path must be a normalized relative path")
	}
	parsed, err := url.Parse(session.Origin + input.Path)
	originURL, originErr := url.Parse(session.Origin)
	if err != nil || originErr != nil || parsed.Scheme != originURL.Scheme || parsed.Host != originURL.Host {
		return brokerOperationResponse{}, errors.New("broker path escapes the approved origin")
	}
	secret, err := p.brokerSecret(ctx, workspace, session.ItemID)
	if err != nil {
		return brokerOperationResponse{}, err
	}
	body := strings.NewReader(input.Body)
	if len(input.Body) > 64*1024 {
		return brokerOperationResponse{}, errors.New("broker request body exceeds 64 KiB")
	}
	request, err := http.NewRequestWithContext(ctx, method, parsed.String(), body)
	if err != nil {
		return brokerOperationResponse{}, err
	}
	for name, value := range input.Headers {
		if strings.EqualFold(name, "authorization") || strings.EqualFold(name, "cookie") || strings.EqualFold(name, "host") {
			return brokerOperationResponse{}, errors.New("authorization, cookie, and host headers are broker-controlled")
		}
		if !brokerHeaderAllowed(name) {
			return brokerOperationResponse{}, errors.New("invalid broker header: header is not permitted")
		}
		request.Header.Set(name, value)
	}
	request.Header.Set("Authorization", "Bearer "+secret)
	transport := &http.Transport{
		Proxy:       nil,
		DialContext: pinnedBrokerDialer(session.ResolvedIPs),
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("broker redirects are not allowed") }}
	response, err := client.Do(request)
	if err != nil {
		return brokerOperationResponse{}, fmt.Errorf("broker request failed: %w", err)
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, 128*1024+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return brokerOperationResponse{}, err
	}
	truncated := len(responseBody) > 128*1024
	if truncated {
		responseBody = responseBody[:128*1024]
	}
	projected := redactBrokerResponse(string(responseBody), secret)
	p.recordAudit(ctx, auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "broker.operation", ItemID: session.ItemID, Outcome: "success", Detail: fmt.Sprintf("%s %s -> %d", method, input.Path, response.StatusCode)})
	return brokerOperationResponse{Status: response.StatusCode, Headers: map[string]string{"content-type": response.Header.Get("Content-Type"), "request-id": response.Header.Get("X-Request-ID")}, Body: projected, BodyTruncated: truncated, Projection: "bounded_redacted_response"}, nil
}

func (p *PasswordManager) claimBrokerSession(ctx context.Context, session brokerSession) error {
	if !session.OneUse {
		return nil
	}
	if p.memory {
		p.mu.Lock()
		defer p.mu.Unlock()
		current, ok := p.brokers[session.ID]
		if !ok || current.Status != "active" || current.UseCount != 0 {
			return errAccessDenied
		}
		current.UseCount = 1
		p.brokers[session.ID] = current
		return nil
	}
	result, err := p.db.ExecContext(ctx, `UPDATE pm_broker_sessions SET use_count=use_count+1 WHERE id=$1 AND status='active' AND one_use=TRUE AND use_count=0 AND expires_at > CURRENT_TIMESTAMP`, session.ID)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count != 1 {
		return errAccessDenied
	}
	return nil
}

func brokerHeaderAllowed(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "accept", "content-type", "user-agent", "x-request-id", "if-none-match", "if-match":
		return true
	default:
		return false
	}
}

func redactBrokerResponse(body, secret string) string {
	if secret == "" {
		return body
	}
	values := []string{secret, url.QueryEscape(secret)}
	if encoded := base64.StdEncoding.EncodeToString([]byte(secret)); encoded != secret {
		values = append(values, encoded)
	}
	if encoded := base64.RawStdEncoding.EncodeToString([]byte(secret)); encoded != secret {
		values = append(values, encoded)
	}
	for _, value := range values {
		body = strings.ReplaceAll(body, value, "[REDACTED]")
	}
	return body
}

func pinnedBrokerDialer(ips []string) func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(address)
		if err != nil || port == "" {
			return nil, errors.New("broker destination has no valid port")
		}
		if len(ips) == 0 {
			return nil, errors.New("broker destination has no pinned addresses")
		}
		var lastErr error
		for _, rawIP := range ips {
			ip := net.ParseIP(rawIP)
			if ip == nil {
				continue
			}
			conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if dialErr == nil {
				return conn, nil
			}
			lastErr = dialErr
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, errors.New("broker destination has no valid pinned addresses")
	}
}

func (h *passwordManagerHandlers) brokerRegisterRoutes(r *mux.Router) {
	r.HandleFunc("/broker/sessions", h.createBrokerSession).Methods(http.MethodPost)
	r.HandleFunc("/broker/sessions/{id}/operations", h.runBrokerOperation).Methods(http.MethodPost)
	r.HandleFunc("/broker/sessions/{id}/revoke", h.revokeBrokerSession).Methods(http.MethodPost)
}

func (h *passwordManagerHandlers) createBrokerSession(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input brokerSessionInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	if auth, ok := r.Context().Value(ownerAuthContextKey{}).(ownerAuthContext); ok && auth.Role == "agent" {
		input.RunID = auth.RunID
	}
	session, token, err := h.manager.createBrokerSession(r.Context(), workspace, actor, input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"session_id": session.ID, "session_token": token, "expires_at": session.Expires, "target_origin": session.Origin, "secret_exposure": "brokered"})
}

func (h *passwordManagerHandlers) runBrokerOperation(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	if err := requireCapability(r, capabilitySecretReveal); err != nil {
		writeVaultError(w, err)
		return
	}
	var input brokerOperationInput
	if err := decodeJSON(r, &input); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	response, err := h.manager.brokerOperation(r.Context(), workspace, actor, mux.Vars(r)["id"], r.Header.Get("X-Broker-Session"), input)
	if err != nil {
		writeVaultError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *passwordManagerHandlers) revokeBrokerSession(w http.ResponseWriter, r *http.Request) {
	if err := h.authorize(r); err != nil {
		writeVaultError(w, err)
		return
	}
	workspace, actor := requestIdentity(r)
	id := mux.Vars(r)["id"]
	if err := h.manager.revokeBrokerSession(r.Context(), workspace, actor, id); err != nil {
		writeVaultError(w, err)
		return
	}
	h.manager.recordAudit(r.Context(), auditEvent{WorkspaceID: workspace, ActorID: actor, Action: "broker.session.revoke", Outcome: "success", Detail: "session capability revoked"})
	writeJSON(w, http.StatusOK, map[string]any{"session_id": id, "status": "revoked"})
}

func (p *PasswordManager) revokeBrokerSession(ctx context.Context, workspace, actor, id string) error {
	if p.memory {
		p.mu.Lock()
		session, ok := p.brokers[id]
		if !ok || session.Workspace != workspace || session.Actor != actor {
			p.mu.Unlock()
			return errAccessDenied
		}
		session.Status = "revoked"
		p.brokers[id] = session
		p.mu.Unlock()
		return nil
	}
	result, err := p.db.ExecContext(ctx, `UPDATE pm_broker_sessions SET status='revoked' WHERE id=$1 AND workspace_id=$2 AND actor_id=$3 AND status='active'`, id, workspace, actor)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return errAccessDenied
	}
	return nil
}
