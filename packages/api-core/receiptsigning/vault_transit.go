package receiptsigning

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CredentialSource obtains a short-lived Vault credential from the lifecycle
// identity layer. It deliberately returns only a request credential, never key
// material, and keeps Prompt Manager independent of an auth mechanism.
type CredentialSource interface {
	Credential(context.Context) (string, error)
}

// VaultTransitConfig contains endpoint and identity wiring, not a signing key.
type VaultTransitConfig struct {
	Address         string
	KeyName         string
	Client          *http.Client
	Credentials     CredentialSource
	AllowedPurposes []Purpose
}

// VaultTransitSigner uses Vault Transit so a caller can sign evidence without
// reading the underlying key.
type VaultTransitSigner struct {
	address, keyName string
	client           *http.Client
	credentials      CredentialSource
	allowed          map[Purpose]struct{}
}

func NewVaultTransitSigner(config VaultTransitConfig) (*VaultTransitSigner, error) {
	address := strings.TrimRight(strings.TrimSpace(config.Address), "/")
	if _, err := url.ParseRequestURI(address); err != nil || !strings.HasPrefix(address, "https://") {
		return nil, fmt.Errorf("Vault Transit requires an HTTPS address")
	}
	if strings.TrimSpace(config.KeyName) == "" || config.Credentials == nil {
		return nil, fmt.Errorf("Vault Transit requires a key name and lifecycle credential source")
	}
	allowed := make(map[Purpose]struct{}, len(config.AllowedPurposes))
	for _, p := range config.AllowedPurposes {
		if !p.Valid() {
			return nil, fmt.Errorf("Vault Transit unsupported allowed purpose %q", p)
		}
		allowed[p] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("Vault Transit requires at least one allowed purpose")
	}
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	return &VaultTransitSigner{address: address, keyName: config.KeyName, client: config.Client, credentials: config.Credentials, allowed: allowed}, nil
}

func (s *VaultTransitSigner) Sign(ctx context.Context, purpose Purpose, canonical []byte) (SignatureEnvelope, error) {
	if _, ok := s.allowed[purpose]; !ok {
		return SignatureEnvelope{}, fmt.Errorf("Vault Transit signing purpose %q is not authorized", purpose)
	}
	digest := Digest(canonical)
	body := struct {
		Input   string `json:"input"`
		Context string `json:"context"`
	}{Input: base64.StdEncoding.EncodeToString(mustDigestBytes(digest)), Context: base64.StdEncoding.EncodeToString([]byte(purpose))}
	var response struct {
		Data struct {
			Signature  string `json:"signature"`
			KeyVersion int    `json:"key_version"`
		} `json:"data"`
	}
	if err := s.do(ctx, http.MethodPost, "/v1/transit/sign/"+url.PathEscape(s.keyName), body, &response); err != nil {
		return SignatureEnvelope{}, err
	}
	if response.Data.Signature == "" || response.Data.KeyVersion < 1 {
		return SignatureEnvelope{}, fmt.Errorf("Vault Transit returned an incomplete signature")
	}
	return SignatureEnvelope{Version: EnvelopeVersionV1, Purpose: purpose, Algorithm: AlgorithmVaultTransit, KeyID: fmt.Sprintf("%s:v%d", s.keyName, response.Data.KeyVersion), Digest: digest, Signature: response.Data.Signature}, nil
}

func (s *VaultTransitSigner) Verify(ctx context.Context, envelope SignatureEnvelope, canonical []byte) error {
	if err := envelope.Validate(); err != nil {
		return err
	}
	if envelope.Algorithm != AlgorithmVaultTransit {
		return fmt.Errorf("Vault Transit cannot verify algorithm %q", envelope.Algorithm)
	}
	keyPrefix := s.keyName + ":v"
	if !strings.HasPrefix(envelope.KeyID, keyPrefix) {
		return fmt.Errorf("Vault Transit envelope key ID %q does not belong to configured key", envelope.KeyID)
	}
	if version, err := strconv.Atoi(strings.TrimPrefix(envelope.KeyID, keyPrefix)); err != nil || version < 1 {
		return fmt.Errorf("Vault Transit envelope key ID %q has no valid version", envelope.KeyID)
	}
	if _, ok := s.allowed[envelope.Purpose]; !ok {
		return fmt.Errorf("Vault Transit verification purpose %q is not authorized", envelope.Purpose)
	}
	if envelope.Digest != Digest(canonical) {
		return fmt.Errorf("receipt digest does not match canonical content")
	}
	body := struct {
		Input     string `json:"input"`
		Signature string `json:"signature"`
		Context   string `json:"context"`
	}{Input: base64.StdEncoding.EncodeToString(mustDigestBytes(envelope.Digest)), Signature: envelope.Signature, Context: base64.StdEncoding.EncodeToString([]byte(envelope.Purpose))}
	var response struct {
		Data struct {
			Valid bool `json:"valid"`
		} `json:"data"`
	}
	if err := s.do(ctx, http.MethodPost, "/v1/transit/verify/"+url.PathEscape(s.keyName), body, &response); err != nil {
		return err
	}
	if !response.Data.Valid {
		return fmt.Errorf("Vault Transit reports receipt signature invalid")
	}
	return nil
}

func (s *VaultTransitSigner) Health(ctx context.Context) (Health, error) {
	var response struct {
		Data struct {
			LatestVersion   int  `json:"latest_version"`
			DeletionAllowed bool `json:"deletion_allowed"`
		} `json:"data"`
	}
	if err := s.do(ctx, http.MethodGet, "/v1/transit/keys/"+url.PathEscape(s.keyName), nil, &response); err != nil {
		return Health{Provider: "vault-transit", Production: true}, err
	}
	if response.Data.LatestVersion < 1 {
		return Health{Provider: "vault-transit", Production: true}, fmt.Errorf("Vault Transit key has no active version")
	}
	return Health{Ready: true, Provider: "vault-transit", KeyID: fmt.Sprintf("%s:v%d", s.keyName, response.Data.LatestVersion), Production: true, RotationOK: !response.Data.DeletionAllowed, Description: "Vault Transit key is reachable through lifecycle identity"}, nil
}

// Rotate requests a new Transit key version. It is intentionally absent from
// ReceiptSigner: application workloads receive only sign/verify authority;
// an operator control plane may use this concrete capability after its own
// authorization check.
func (s *VaultTransitSigner) Rotate(ctx context.Context) (Health, error) {
	if err := s.do(ctx, http.MethodPost, "/v1/transit/keys/"+url.PathEscape(s.keyName)+"/rotate", struct{}{}, nil); err != nil {
		return Health{Provider: "vault-transit", Production: true}, err
	}
	return s.Health(ctx)
}

func (s *VaultTransitSigner) do(ctx context.Context, method, path string, in, out any) error {
	credential, err := s.credentials.Credential(ctx)
	if err != nil {
		return fmt.Errorf("Vault lifecycle credential: %w", err)
	}
	if strings.TrimSpace(credential) == "" {
		return fmt.Errorf("Vault lifecycle credential is empty")
	}
	var body *bytes.Reader
	if in == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, marshalErr := json.Marshal(in)
		if marshalErr != nil {
			return marshalErr
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.address+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Vault-Token", credential)
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("Vault Transit request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Vault Transit request failed: %s", resp.Status)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode Vault Transit response: %w", err)
		}
	}
	return nil
}

func mustDigestBytes(digest string) []byte { decoded, _ := ParseDigest(digest); return decoded }
