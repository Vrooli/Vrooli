package cliutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AgentCredentialSocket is shared by the local authority and CLI clients.
// The directory is private to the host account; peer credentials, rather than
// harness environment variables or a caller-supplied PID, bind each request.
func AgentCredentialSocket() string {
	return filepath.Join(os.TempDir(), "vrooli-agent-manager-"+strconv.Itoa(os.Getuid()), "identity.sock")
}

// AgentAuthorizationError is the local authority's refusal to issue this
// session's credential. The API may be healthy; retrying or starting the
// scenario cannot help, so callers report it as an authorization problem.
type AgentAuthorizationError struct{ Code string }

func (e *AgentAuthorizationError) Error() string {
	if e.Code == "identity_refresh_unavailable" {
		return "agent identity " + e.Code + "; check run/session binding and owner-proof; if the session owner is restarting or unhealthy, retry after it is healthy; otherwise inspect the attached session"
	}
	return "agent authorization " + e.Code + "; inspect with vrooli agent authorization <session-or-run>"
}

// ResolveAgentCredential consults the live authority for every request. An
// unavailable broker preserves a structurally ordinary inherited run token;
// grant-only credentials remain unavailable until the broker can perform its
// current-grant selection. A responding broker's refusal never falls back.
func ResolveAgentCredential(ctx context.Context) (string, error) {
	credentials, err := ResolveAgentCredentials(ctx)
	if err != nil {
		return "", err
	}
	if credentials.BaselineToken != "" {
		return credentials.BaselineToken, nil
	}
	return credentials.LegacyToken, nil
}

// AgentCredentials keeps ordinary run identity separate from a currently
// approved operation grant. Legacy grant-only callers use LegacyToken and
// remain bounded by the existing identity-token policy.
type AgentCredentials struct {
	BaselineToken    string
	OperationToken   string
	LegacyToken      string
	OperationRefusal string
}

// ResolveAgentCredentials consults the live authority for each request. An
// unavailable broker preserves only a structurally ordinary inherited run
// token; a responding broker is authoritative and never falls back to an
// older grant.
func ResolveAgentCredentials(ctx context.Context) (AgentCredentials, error) {
	return resolveAgentCredentials(ctx, AgentCredentialSocket(), os.Getenv(EnvIdentityToken))
}

// ConfirmLocalOwnerEligible succeeds only when the live broker proves that this
// peer is outside every current Agent Manager runner process tree. This is a
// cooperative CLI preflight; the owner exchange remains responsible for proof.
func ConfirmLocalOwnerEligible(ctx context.Context) error {
	return confirmLocalOwnerEligible(ctx, AgentCredentialSocket(), os.Getenv(EnvIdentityToken))
}

func confirmLocalOwnerEligible(ctx context.Context, socket, inherited string) error {
	if strings.TrimSpace(inherited) != "" {
		return errors.New("agent identity is present")
	}
	_, confirmedAbsent, err := resolveAgentCredentialsWithNamedPurpose(ctx, socket, inherited, "local_owner_eligibility")
	if err != nil {
		return err
	}
	if !confirmedAbsent {
		return errors.New("local owner eligibility was not confirmed")
	}
	return nil
}

// ResolveOrdinaryAgentIdentity asks the local broker only for the ordinary
// inherited run identity. It is used by identity reporting and self-parking
// callers that must verify and submit the same current baseline. Normal API and
// child-launch selection remains unchanged; this does not select operation or
// legacy grant credentials.
func ResolveOrdinaryAgentIdentity(ctx context.Context) (string, error) {
	credentials, err := resolveAgentCredentialsWithPurpose(ctx, AgentCredentialSocket(), os.Getenv(EnvIdentityToken), true)
	if err != nil {
		return "", err
	}
	if credentials.BaselineToken == "" || credentials.LegacyToken != "" || credentials.OperationToken != "" {
		return "", errors.New("ordinary agent-manager run identity is unavailable")
	}
	return credentials.BaselineToken, nil
}

func resolveAgentCredentials(ctx context.Context, socket, inherited string) (AgentCredentials, error) {
	return resolveAgentCredentialsWithPurpose(ctx, socket, inherited, false)
}

func resolveAgentCredentialsWithPurpose(ctx context.Context, socket, inherited string, ordinaryIdentity bool) (AgentCredentials, error) {
	purpose := ""
	if ordinaryIdentity {
		purpose = "ordinary_identity"
	}
	credentials, _, err := resolveAgentCredentialsWithNamedPurpose(ctx, socket, inherited, purpose)
	return credentials, err
}

func resolveAgentCredentialsWithNamedPurpose(ctx context.Context, socket, inherited, purpose string) (AgentCredentials, bool, error) {
	inherited = strings.TrimSpace(inherited)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	defer transport.CloseIdleConnections()
	endpoint := "http://local-agent-manager/credential"
	if purpose != "" {
		endpoint += "?purpose=" + url.QueryEscape(purpose)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return AgentCredentials{}, false, err
	}
	if inherited != "" {
		req.Header.Set(HeaderAgentIdentityToken, inherited)
	}
	var resp *http.Response
	for _, timeout := range agentCredentialBrokerTimeouts {
		resp, err = (&http.Client{Transport: transport, Timeout: timeout}).Do(req)
		var netErr net.Error
		if err == nil || !errors.As(err, &netErr) || !netErr.Timeout() {
			break
		}
	}
	if err != nil {
		if purpose == "local_owner_eligibility" {
			return AgentCredentials{}, false, errors.New("local owner identity broker unavailable")
		}
		if _, directoryErr := os.Stat(filepath.Dir(socket)); directoryErr == nil && inherited != "" {
			// The broker exists but did not answer: the request goes out with the
			// bare run identity, so any approved operation grant is missing from it.
			// Say so, or a grant-backed refusal reads as "ask for approval".
			fmt.Fprintf(os.Stderr, "Warning: the Agent Manager identity broker did not answer (%v); continuing without this session's approved operation grant, so grant-backed operations will be refused. Retry when Agent Manager is less busy.\n", err)
		}
		credentials, fallbackErr := inheritedBaselineOnUnavailableBroker(inherited)
		return credentials, false, fallbackErr
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return AgentCredentials{BaselineToken: inherited}, purpose == "local_owner_eligibility" && inherited == "", nil
	}
	if purpose == "local_owner_eligibility" {
		return AgentCredentials{}, false, fmt.Errorf("local owner eligibility not confirmed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		var refusal struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&refusal)
		if refusal.Code == "" {
			refusal.Code = fmt.Sprintf("refused (HTTP %d)", resp.StatusCode)
		}
		return AgentCredentials{}, false, &AgentAuthorizationError{Code: refusal.Code}
	}
	var body struct {
		Token            string `json:"token"`
		BaselineToken    string `json:"baseline_token"`
		OperationToken   string `json:"operation_token"`
		OperationRefusal string `json:"operation_refusal"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 256*1024)).Decode(&body); err != nil || body.Token == "" {
		if err != nil {
			return AgentCredentials{}, false, errors.New("invalid agent credential authority response")
		}
	}
	credentials := AgentCredentials{
		BaselineToken: strings.TrimSpace(body.BaselineToken), OperationToken: strings.TrimSpace(body.OperationToken),
		LegacyToken: strings.TrimSpace(body.Token), OperationRefusal: strings.TrimSpace(body.OperationRefusal),
	}
	if (credentials.BaselineToken == "") == (credentials.LegacyToken == "") ||
		(credentials.LegacyToken != "" && credentials.OperationToken != "") ||
		(credentials.BaselineToken != "" && inherited == "") ||
		(credentials.BaselineToken != "" && !isSameRunBaselineRenewal(inherited, credentials.BaselineToken)) {
		return AgentCredentials{}, false, errors.New("invalid agent credential authority response")
	}
	return credentials, false, nil
}

type ordinaryIdentityClaims struct {
	Purpose     string            `json:"purpose,omitempty"`
	RunID       string            `json:"run_id"`
	TaskID      string            `json:"task_id"`
	Subject     string            `json:"subject,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	Scopes      []string          `json:"scopes"`
	ProfileKey  string            `json:"profile_key"`
	ScopePath   string            `json:"scope_path"`
	IssuedAt    int64             `json:"iat"`
	ExpiresAt   int64             `json:"exp"`
	Meta        map[string]string `json:"meta"`
}

func isSameRunBaselineRenewal(previous, replacement string) bool {
	if previous == replacement {
		return true
	}
	oldClaims, oldOK := parseOrdinaryIdentityClaims(previous)
	newClaims, newOK := parseOrdinaryIdentityClaims(replacement)
	now := time.Now().Unix()
	if !oldOK || !newOK || oldClaims.Purpose != "" || newClaims.Purpose != "" ||
		oldClaims.Meta["authorization_id"] != "" || newClaims.Meta["authorization_id"] != "" ||
		oldClaims.ExpiresAt > now || oldClaims.ExpiresAt > oldClaims.IssuedAt+int64(defaultAgentIdentityTTL/time.Second) ||
		oldClaims.RunID == "" || oldClaims.RunID != newClaims.RunID ||
		oldClaims.TaskID != newClaims.TaskID || oldClaims.Subject != newClaims.Subject ||
		oldClaims.WorkspaceID != newClaims.WorkspaceID || oldClaims.ProfileKey != newClaims.ProfileKey ||
		oldClaims.ScopePath != newClaims.ScopePath || len(newClaims.Scopes) != 0 ||
		!sameStringMap(oldClaims.Meta, newClaims.Meta) || newClaims.IssuedAt <= oldClaims.IssuedAt || newClaims.IssuedAt > now ||
		newClaims.ExpiresAt <= newClaims.IssuedAt || newClaims.ExpiresAt > newClaims.IssuedAt+int64(defaultAgentIdentityTTL/time.Second) ||
		newClaims.ExpiresAt <= now {
		return false
	}
	return true
}

const defaultAgentIdentityTTL = 24 * time.Hour

func parseOrdinaryIdentityClaims(token string) (ordinaryIdentityClaims, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 || parts[0] == "" {
		return ordinaryIdentityClaims{}, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != 32 {
		return ordinaryIdentityClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ordinaryIdentityClaims{}, false
	}
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var claims ordinaryIdentityClaims
	if err := dec.Decode(&claims); err != nil || claims.IssuedAt <= 0 || claims.ExpiresAt <= claims.IssuedAt {
		return ordinaryIdentityClaims{}, false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return ordinaryIdentityClaims{}, false
	}
	if claims.Scopes == nil {
		claims.Scopes = []string{}
	}
	return claims, true
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

// inheritedBaselineOnUnavailableBroker preserves only a well-formed ordinary
// run token when the local authority cannot be reached. This is a restrictive
// role classifier, not token verification: the receiving API still verifies
// the signature, expiry and retained run. A grant token must not bypass the
// broker's current-grant selection or revocation checks.
// agentCredentialBrokerTimeouts bounds each broker attempt. A busy Agent
// Manager (many concurrent writes) can take longer than the first bound; one
// longer retry keeps the operation grant instead of silently dropping it.
var agentCredentialBrokerTimeouts = []time.Duration{5 * time.Second, 15 * time.Second}

func inheritedBaselineOnUnavailableBroker(inherited string) (AgentCredentials, error) {
	if inherited == "" {
		return AgentCredentials{}, nil
	}
	parts := strings.Split(inherited, ".")
	if len(parts) != 2 {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(signature) != 32 {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	var claims struct {
		Purpose   string            `json:"purpose,omitempty"`
		RunID     string            `json:"run_id"`
		IssuedAt  int64             `json:"iat"`
		ExpiresAt int64             `json:"exp"`
		Meta      map[string]string `json:"meta"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Purpose != "" || strings.TrimSpace(claims.RunID) == "" || claims.IssuedAt <= 0 || claims.ExpiresAt <= 0 || claims.Meta["authorization_id"] != "" {
		return AgentCredentials{}, errors.New("agent credential authority unavailable")
	}
	return AgentCredentials{BaselineToken: inherited, OperationRefusal: "authority_unavailable"}, nil
}

// resolveAgentCredential preserves the legacy single-token test/caller seam.
func resolveAgentCredential(ctx context.Context, socket, inherited string) (string, error) {
	credentials, err := resolveAgentCredentials(ctx, socket, inherited)
	if err != nil {
		return "", err
	}
	if credentials.BaselineToken != "" {
		return credentials.BaselineToken, nil
	}
	return credentials.LegacyToken, nil
}
