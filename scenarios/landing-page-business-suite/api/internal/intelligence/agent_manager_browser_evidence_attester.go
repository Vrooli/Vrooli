package intelligence

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
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
)

const (
	agentManagerBrowserEvidencePath       = "/api/v1/browser/settlement-attestation/"
	LPBSBrowserEvidencePrincipal          = "landing-page-business-suite-browser-evidence-attester"
	LPBSBrowserEvidenceAudience           = "scenario-authenticator:agent-manager"
	LPBSBrowserEvidenceScope              = "agent-manager:browser-evidence.attest"
	lpbsBrowserEvidenceCredentialIdentity = "scenario-authenticator/service-principal/landing-page-business-suite/landing-page-business-suite-browser-evidence-attester/agent-manager"
)

type browserEvidenceServiceTokenSource interface {
	Token(context.Context) (string, error)
}

type browserEvidenceEndpointResolver func(context.Context) (string, error)

// AgentManagerBrowserEvidenceAttester uses LPBS's dedicated exact-scope
// service identity to ask Agent Manager to resolve a BAS proof and consume it
// through AM's existing BAS verifier. It does not call BAS directly and never
// accepts client-measured usage. Reservation ownership and replay protection
// remain in LPBS's existing account ledger.
type AgentManagerBrowserEvidenceAttester struct {
	tokens     browserEvidenceServiceTokenSource
	client     *http.Client
	resolveURL browserEvidenceEndpointResolver
}

// NewDiscoveredAgentManagerBrowserEvidenceAttester is safe before the operator
// issues the principal: token lookup is lazy, and absence or mismatch causes
// every attestation to fail closed.
func NewDiscoveredAgentManagerBrowserEvidenceAttester() *AgentManagerBrowserEvidenceAttester {
	client := &http.Client{Timeout: 8 * time.Second,
		Transport: &http.Transport{
			DialContext:  (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
			MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	tokens := servicetoken.New(servicetoken.Config{
		Name: "LPBS browser evidence settler",
		Credential: func(context.Context) (string, error) {
			identity, err := credentialauthority.ParseIdentity(lpbsBrowserEvidenceCredentialIdentity)
			if err != nil {
				return "", err
			}
			binding, err := administration.ResolveBoundCredential(identity, "credential", LPBSBrowserEvidenceScope)
			return binding.Value, err
		},
		Expect:    servicetoken.ExpectIdentity(LPBSBrowserEvidencePrincipal, LPBSBrowserEvidenceAudience, LPBSBrowserEvidenceScope),
		IssueHint: "the operator configures and issues the LPBS browser evidence attester principal",
	})
	return &AgentManagerBrowserEvidenceAttester{
		tokens: tokens, client: client,
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
		},
	}
}

// Ready checks the exact service credential and safe AM origin without
// calling the attestation endpoint. LPBS runs it before reserving credits or
// dispatching inference so an absent operator-issued credential cannot create
// an uncertain hold or incur an unfunded provider call.
func (a *AgentManagerBrowserEvidenceAttester) Ready(ctx context.Context) error {
	if a == nil || a.tokens == nil || a.client == nil || a.resolveURL == nil {
		return errors.New("Agent Manager browser evidence attester is unavailable")
	}
	if _, err := a.origin(ctx); err != nil {
		return err
	}
	token, err := a.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return errors.New("LPBS browser evidence service identity is unavailable")
	}
	return nil
}

type browserSettlementAttestationRequest struct {
	GrantReceipt   string `json:"grant_receipt"`
	ProofReference string `json:"proof_reference"`
	ReservationID  string `json:"reservation_id"`
}

type browserSettlementAttestationResponse struct {
	RunID                   string                     `json:"run_id"`
	ReservationID           string                     `json:"reservation_id"`
	GrantReceiptFingerprint string                     `json:"grant_receipt_fingerprint"`
	ProofFingerprint        string                     `json:"proof_fingerprint"`
	Observation             browserEvidenceObservation `json:"observation"`
}

type browserEvidenceObservation struct {
	Kind           string `json:"kind"`
	SessionID      string `json:"session_id"`
	State          string `json:"state"`
	SessionMinutes int64  `json:"session_minutes"`
	Actions        int64  `json:"actions"`
	Bytes          int64  `json:"bytes"`
}

func (a *AgentManagerBrowserEvidenceAttester) VerifyAndConsumeBrowserEvidence(ctx context.Context, proof BrowserEvidenceProof) (AttestedBrowserUsage, error) {
	if a == nil || a.tokens == nil || a.client == nil || a.resolveURL == nil {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser evidence attester is unavailable")
	}
	if err := proof.validate(); err != nil {
		return AttestedBrowserUsage{}, err
	}
	runID, err := uuid.Parse(strings.TrimSpace(proof.RunID))
	if err != nil || runID == uuid.Nil || runID.String() != proof.RunID {
		return AttestedBrowserUsage{}, errors.New("browser evidence run id must be a canonical UUID")
	}
	reservationID, err := uuid.Parse(strings.TrimSpace(proof.ReservationID))
	if err != nil || reservationID == uuid.Nil || reservationID.String() != proof.ReservationID {
		return AttestedBrowserUsage{}, errors.New("browser evidence requires the canonical LPBS reservation id")
	}
	parsed, err := a.origin(ctx)
	if err != nil {
		return AttestedBrowserUsage{}, err
	}
	token, err := a.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return AttestedBrowserUsage{}, errors.New("LPBS browser evidence service identity is unavailable")
	}
	body, err := json.Marshal(browserSettlementAttestationRequest{
		GrantReceipt: proof.GrantReceipt, ProofReference: proof.Reference, ReservationID: reservationID.String(),
	})
	if err != nil {
		return AttestedBrowserUsage{}, errors.New("browser settlement attestation request could not be encoded")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(parsed.String(), "/")+agentManagerBrowserEvidencePath+runID.String(), bytes.NewReader(body))
	if err != nil {
		return AttestedBrowserUsage{}, errors.New("browser settlement attestation request could not be created")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser settlement attestation failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return AttestedBrowserUsage{}, fmt.Errorf("Agent Manager browser settlement attestation refused the request (HTTP %d)", response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(responseBody) > 64<<10 {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser settlement attestation response exceeded its size limit")
	}
	var attested browserSettlementAttestationResponse
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&attested); err != nil {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser settlement attestation response was invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser settlement attestation response had trailing data")
	}
	if attested.RunID != runID.String() || attested.ReservationID != reservationID.String() || attested.GrantReceiptFingerprint != fingerprint(proof.GrantReceipt) || attested.ProofFingerprint != fingerprint(proof.Reference) {
		return AttestedBrowserUsage{}, errors.New("Agent Manager browser settlement attestation binding did not match the request")
	}
	if attested.Observation.Kind != "session" || strings.TrimSpace(attested.Observation.SessionID) == "" || attested.Observation.State != "closed" || attested.Observation.SessionMinutes < 0 || attested.Observation.Actions < 0 || attested.Observation.Bytes < 0 {
		return AttestedBrowserUsage{}, errors.New("Agent Manager returned no complete closed broker observation")
	}
	return AttestedBrowserUsage{
		RunID: attested.RunID, ReservationID: attested.ReservationID, SessionID: attested.Observation.SessionID,
		State: attested.Observation.State, SessionMinutes: attested.Observation.SessionMinutes,
		Actions: attested.Observation.Actions, Bytes: attested.Observation.Bytes,
	}, nil
}

func (a *AgentManagerBrowserEvidenceAttester) origin(ctx context.Context) (*url.URL, error) {
	base, err := a.resolveURL(ctx)
	if err != nil {
		return nil, errors.New("Agent Manager browser evidence endpoint is unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || !safeBrowserEvidenceOrigin(parsed) {
		return nil, errors.New("Agent Manager browser evidence endpoint must be HTTPS or loopback HTTP origin")
	}
	return parsed, nil
}

func fingerprint(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func safeBrowserEvidenceOrigin(parsed *url.URL) bool {
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
