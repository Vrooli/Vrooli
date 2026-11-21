package handlers

import (
	"bytes"
	"context"
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
)

const (
	browserEvidenceVerifierPrincipal = "agent-manager-browser-evidence-verifier"
	browserEvidenceVerifierResource  = "browser-automation-studio"
	browserEvidenceVerifierAudience  = "scenario-authenticator:browser-automation-studio"
	browserEvidenceVerifierScope     = "browser-automation-studio:browser-evidence.verify"
	browserEvidenceVerifierIdentity  = "scenario-authenticator/service-principal/agent-manager-browser-evidence-verifier/browser-automation-studio"
)

// DiscoveredBrowserEvidenceProofVerifier calls the BAS owner callback using
// Agent Manager's dedicated, operator-issued service principal. Its credential
// is read from credential authority on exchange; this client never forwards
// inbound run identity, browser grants, cookies, or bearer headers.
type DiscoveredBrowserEvidenceProofVerifier struct {
	Tokens     *servicetoken.Tokens
	Client     *http.Client
	ResolveURL func(context.Context) (string, error)
}

// NewDiscoveredBrowserEvidenceProofVerifier builds an unconfigured-safe client.
// Until the operator issues the declared principal credential, calls fail
// closed at token exchange and the BAS evidence route cannot append events.
func NewDiscoveredBrowserEvidenceProofVerifier() *DiscoveredBrowserEvidenceProofVerifier {
	transport := &http.Transport{
		DialContext:  (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second,
	}
	tokens := servicetoken.New(servicetoken.Config{
		Name: "agent-manager browser evidence verifier",
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity(browserEvidenceVerifierIdentity)
			if err != nil {
				return "", err
			}
			return authority.Require(identity, "credential")
		},
		Expect: servicetoken.ExpectIdentity(browserEvidenceVerifierPrincipal,
			browserEvidenceVerifierAudience, browserEvidenceVerifierScope),
		RefreshMargin: time.Minute,
	})
	client := &http.Client{Timeout: 8 * time.Second, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &DiscoveredBrowserEvidenceProofVerifier{
		Tokens: tokens, Client: client,
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "browser-automation-studio")
		},
	}
}

func (c *DiscoveredBrowserEvidenceProofVerifier) VerifyBrowserEvidenceProof(ctx context.Context, runID uuid.UUID, grantReceipt, proofReference string) (BrowserEvidenceObservation, error) {
	var response struct {
		Observation BrowserEvidenceObservation `json:"observation"`
	}
	err := c.post(ctx, "/api/v1/browser/internal/evidence-proof/verify", map[string]string{
		"run_id": runID.String(), "grant_receipt": grantReceipt, "proof_reference": proofReference,
	}, &response)
	if err != nil {
		return BrowserEvidenceObservation{}, err
	}
	if strings.TrimSpace(response.Observation.Kind) == "" || strings.TrimSpace(response.Observation.SessionID) == "" || response.Observation.State != "closed" {
		return BrowserEvidenceObservation{}, errors.New("BAS evidence verifier returned an incomplete or non-closed observation")
	}
	if response.Observation.SessionMinutes < 0 || response.Observation.Actions < 0 || response.Observation.Bytes < 0 {
		return BrowserEvidenceObservation{}, errors.New("BAS evidence verifier returned negative measurements")
	}
	return response.Observation, nil
}

func (c *DiscoveredBrowserEvidenceProofVerifier) ConsumeBrowserEvidenceProof(ctx context.Context, runID uuid.UUID, proofReference string) error {
	var response struct {
		Consumed bool `json:"consumed"`
	}
	if err := c.post(ctx, "/api/v1/browser/internal/evidence-proof/consume", map[string]string{
		"run_id": runID.String(), "proof_reference": proofReference,
	}, &response); err != nil {
		return err
	}
	if !response.Consumed {
		return errors.New("BAS evidence verifier did not consume the proof reference")
	}
	return nil
}

func (c *DiscoveredBrowserEvidenceProofVerifier) post(ctx context.Context, path string, body any, output any) error {
	if c == nil || c.Tokens == nil || c.Client == nil || c.ResolveURL == nil {
		return errors.New("BAS evidence verifier is not configured")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return errors.New("BAS evidence verifier endpoint is unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("BAS evidence verifier endpoint is invalid")
	}
	token, err := c.Tokens.Token(ctx)
	if err != nil {
		return errors.New("BAS evidence verifier service identity unavailable")
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return errors.New("BAS evidence verifier request could not be encoded")
	}
	endpoint := strings.TrimRight(parsed.String(), "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.New("BAS evidence verifier request could not be created")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.Client.Do(request)
	if err != nil {
		return errors.New("BAS evidence verifier request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("BAS evidence verifier refused the request (HTTP %d)", response.StatusCode)
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, (64<<10)+1))
	if err != nil || len(responseBody) > 64<<10 {
		return errors.New("BAS evidence verifier response exceeded its size limit")
	}
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(output); err != nil {
		return errors.New("BAS evidence verifier response was invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return errors.New("BAS evidence verifier response had trailing data")
	}
	return nil
}
