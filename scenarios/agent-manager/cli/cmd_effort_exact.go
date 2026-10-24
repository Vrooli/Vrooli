package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// exact-reconcile emits one nonsecret challenge, remains alive while a private
// signer supplies its proof on stdin, then submits using the original protected
// credential. It never attaches, persists or prints a bearer, or retries.
func (a *App) effortExactReconcile(args []string) error {
	fs := flag.NewFlagSet("effort exact-reconcile", flag.ContinueOnError)
	file := fs.String("request-file", "", "Disposable thin typed metadata request")
	grant := fs.String("grant", "", "Existing owner-installed grant ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" || *grant == "" || fs.NArg() != 0 {
		return fmt.Errorf("request-file and existing grant required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	if len(raw) > 16000 {
		return fmt.Errorf("request exceeds bound")
	}
	req := &pb.ReconcileEffortMetadataRequest{}
	if err = protojson.Unmarshal(raw, req); err != nil {
		return err
	}
	return a.exactReconcile(req, *grant, os.Stdin, os.Stdout, cliutil.ResolveAgentCredentials)
}

func (a *App) exactReconcile(req *pb.ReconcileEffortMetadataRequest, grant string, input io.Reader, output io.Writer, resolve func(context.Context) (cliutil.AgentCredentials, error)) error {
	payload, err := marshalProtoRequest(req)
	if err != nil {
		return err
	}
	return protectedExact(a.services.Watches.api, "keyring.exact-action.v1", "/api/v1/keyring/exact-challenge", grant, payload, input, output, resolve, func(api *cliutil.APIClient) ([]byte, error) {
		return (&WatchService{api: api}).call(apiconnect.AgentManagerServiceReconcileEffortMetadataProcedure, req, &pb.EffortEnrollment{})
	})
}

// This typed value mirrors the native receipt-read v1 contract, preserving the
// original executor/action separately from the freshly verified read executor.
type exactReceiptRequest struct {
	OperationKey     string `json:"operationKey"`
	EffortRef        string `json:"effortRef"`
	OriginalExecutor string `json:"originalExecutor"`
	OriginalPayload  string `json:"originalPayload"`
}

func (a *App) effortExactReceipt(args []string) error {
	fs := flag.NewFlagSet("effort exact-receipt", flag.ContinueOnError)
	file := fs.String("request-file", "", "Original immutable receipt request")
	grant := fs.String("grant", "", "Original standing grant with explicit receipt.read eligibility")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" || *grant == "" || fs.NArg() != 0 {
		return fmt.Errorf("request-file and original standing grant required")
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	if len(raw) > 16000 {
		return fmt.Errorf("request exceeds bound")
	}
	var req exactReceiptRequest
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || d.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("typed receipt request required")
	}
	return a.exactReceipt(req, *grant, os.Stdin, os.Stdout, cliutil.ResolveAgentCredentials)
}

func (a *App) exactReceipt(req exactReceiptRequest, grant string, input io.Reader, output io.Writer, resolve func(context.Context) (cliutil.AgentCredentials, error)) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return protectedExact(a.services.Watches.api, "keyring.receipt-read.v1", "/api/v1/keyring/exact-read-challenge", grant, payload, input, output, resolve, func(api *cliutil.APIClient) ([]byte, error) {
		return api.Request("POST", "/api/v1/keyring/exact-receipt", nil, json.RawMessage(payload))
	})
}

func protectedExact(apiBase *cliutil.APIClient, protocol, challengePath, grant string, payload json.RawMessage, input io.Reader, output io.Writer, resolve func(context.Context) (cliutil.AgentCredentials, error), submit func(*cliutil.APIClient) ([]byte, error)) error {
	credentials, err := resolve(context.Background())
	if err != nil {
		return err
	}
	token := credentials.BaselineToken
	if token == "" {
		token = credentials.LegacyToken
	}
	if token == "" {
		return fmt.Errorf("existing verified native credential required")
	}
	var api *cliutil.APIClient
	if credentials.BaselineToken != "" && credentials.OperationToken != "" {
		api = apiBase.WithPinnedAgentCredentials(credentials, resolve)
	} else {
		// Keep the pre-pair exact flow for legacy grant-only credentials. A
		// later credential-mode or operation change is refused by the resolver.
		resolveLegacy := func(ctx context.Context) (string, error) {
			current, resolveErr := resolve(ctx)
			if resolveErr != nil {
				return "", resolveErr
			}
			if !sameExactCredentialMode(credentials, current) {
				return "", fmt.Errorf("native credential changed during exact request")
			}
			if current.BaselineToken != "" {
				return current.BaselineToken, nil
			}
			return current.LegacyToken, nil
		}
		api = apiBase.WithPinnedAgentCredential(token, resolveLegacy)
	}
	challenge, err := api.Request("POST", challengePath, url.Values{"grant": {grant}}, payload)
	if err != nil {
		return fmt.Errorf("exact challenge refused")
	}
	if len(challenge) > 16000 {
		return fmt.Errorf("challenge exceeds bound")
	}
	if _, err = fmt.Fprintln(output, string(challenge)); err != nil {
		return err
	}
	var proof struct {
		Protocol  string `json:"protocol"`
		Bytes     string `json:"bytes"`
		Signature string `json:"signature"`
	}
	b, err := io.ReadAll(io.LimitReader(input, 22001))
	if err != nil || len(b) > 22000 {
		return fmt.Errorf("bounded proof required")
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&proof) != nil || proof.Protocol != protocol || len(proof.Bytes) > 21336 || len(proof.Signature) > 128 {
		return fmt.Errorf("invalid proof")
	}
	envelope, err := base64.RawURLEncoding.DecodeString(proof.Bytes)
	if err != nil || len(envelope) == 0 || len(envelope) > 16000 {
		return fmt.Errorf("invalid proof")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("invalid proof")
	}
	signature, err := base64.RawURLEncoding.DecodeString(proof.Signature)
	if err != nil || len(signature) != 64 {
		return fmt.Errorf("invalid proof")
	}
	// Reconsult the authority; a refusal or changed custody cannot fall back to
	// the original leaf. Even a renewed leaf requires a new explicit flow.
	current, err := resolve(context.Background())
	if err != nil {
		return err
	}
	if !sameExactCredentialMode(credentials, current) {
		return fmt.Errorf("native credential changed during signing; submission refused")
	}
	api = api.WithHeaders(map[string]string{"X-Keyring-Exact-Proof": proof.Bytes, "X-Keyring-Exact-Signature": proof.Signature})
	body, err := submit(api)
	if err != nil {
		var apiErr *cliutil.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 503 {
			return fmt.Errorf("exact response unavailable; outcome is unknown; recover the original receipt or retry only with fresh proof, the same executor and original request identity")
		}
		return fmt.Errorf("exact request did not return a confirmed result; recover the original receipt and retain the same executor and original request identity")
	}
	_, err = fmt.Fprintln(output, string(body))
	return err
}

func sameExactCredentialMode(left, right cliutil.AgentCredentials) bool {
	return left.BaselineToken == right.BaselineToken &&
		left.OperationToken == right.OperationToken &&
		left.LegacyToken == right.LegacyToken
}

// These value-only wire types mirror supervision.ExactSetupRequest/ExactPolicy.
// Keep native authorization and policy decisions on the server; importing the
// repository-owning implementation here would expand the CLI dependency graph.
type exactSetupRequest struct {
	Action        string            `json:"action"`
	Policy        *exactSetupPolicy `json:"policy,omitempty"`
	Grant         string            `json:"grant,omitempty"`
	ExpectedEpoch int               `json:"expectedEpoch"`
	PolicyDigest  string            `json:"policyDigest,omitempty"`
	KeyProof      string            `json:"keyProof,omitempty"`
}
type exactSetupPolicy struct {
	ID              string                   `json:"id"`
	Owner           string                   `json:"owner"`
	Instance        string                   `json:"instance"`
	Issuer          string                   `json:"issuer"`
	KeyVersion      int                      `json:"keyVersion"`
	Epoch           int                      `json:"epoch"`
	Deadline        int64                    `json:"deadline"`
	EffortRef       string                   `json:"effortRef"`
	AuthorizedBy    string                   `json:"authorizedBy"`
	AuthorityRef    string                   `json:"authorityRef"`
	Profile         string                   `json:"profile"`
	PublicKey       []byte                   `json:"publicKey"`
	Operations      []string                 `json:"operations"`
	ReadEligibility string                   `json:"readEligibility"`
	Reader          string                   `json:"reader"`
	ReceiptSource   *exactSetupReceiptSource `json:"receiptSource"`
}
type exactSetupReceiptSource struct {
	Grant      string `json:"grant"`
	Epoch      int    `json:"epoch"`
	Issuer     string `json:"issuer"`
	KeyVersion int    `json:"keyVersion"`
}

// Setup uses existing independent operator proof, or an explicitly requested
// local-owner exchange. Agent attribution and native human gates remain intact.
func (a *App) effortExactSetup(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("stage, activate, revoke or status required")
	}
	action := args[0]
	fs := flag.NewFlagSet("effort exact-setup "+action, flag.ContinueOnError)
	file := fs.String("request-file", "", "Typed public owner setup request")
	grant := fs.String("grant", "", "Existing grant for nonsecret owner status")
	localOwner := fs.Bool("local-owner", false, "Explicit local human authentication; unavailable inside an identified agent run")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected setup arguments")
	}
	method := "GET"
	var query url.Values
	var payload json.RawMessage
	if action == "status" {
		if *grant == "" || *file != "" {
			return fmt.Errorf("existing grant required for status")
		}
		query = url.Values{"grant": {*grant}}
	} else {
		if (action != "stage" && action != "activate" && action != "revoke") || *file == "" || *grant != "" {
			return fmt.Errorf("typed request-file required for stage/activate/revoke")
		}
		raw, e := os.ReadFile(*file)
		if e != nil {
			return e
		}
		if len(raw) > 16000 {
			return fmt.Errorf("setup request exceeds bound")
		}
		var req exactSetupRequest
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&req) != nil || d.Decode(&struct{}{}) != io.EOF || req.Action != action {
			return fmt.Errorf("typed matching owner setup action required")
		}
		// Decode the native public type before obtaining a credential. Native
		// authorization, binding and policy validation still own acceptance.
		if req.ExpectedEpoch < 0 || action == "stage" && (req.Policy == nil || req.Grant != "" || req.PolicyDigest != "" || req.KeyProof != "") || action != "stage" && (req.Policy != nil || req.Grant == "" || req.ExpectedEpoch < 1 || len(req.PolicyDigest) != 64) {
			return fmt.Errorf("typed owner setup fields required")
		}
		if action == "activate" {
			proof, err := base64.RawURLEncoding.DecodeString(req.KeyProof)
			if err != nil || len(proof) != 64 {
				return fmt.Errorf("typed activation key proof required")
			}
		} else if req.KeyProof != "" {
			return fmt.Errorf("key proof is only for activation")
		}
		encoded, e := json.Marshal(req)
		if e != nil {
			return e
		}
		method, payload = "POST", json.RawMessage(encoded)
	}
	// Validate all CLI input before any exchange. The request-local copy never
	// persists credentials, replaces provenance or falls back on refusal.
	api := a.services.Watches.api
	if *localOwner {
		owner, err := localOwnerAPI(api)
		if err != nil {
			return err
		}
		api = owner
	}
	body, err := api.WithoutRedirects().Request(method, "/api/v1/keyring/authority/setup", query, payload)
	if err != nil {
		if method == "POST" {
			return fmt.Errorf("owner setup did not return a confirmed result; original mutation may have committed; reconcile original grant status without changing approval identity")
		}
		return fmt.Errorf("owner status did not return a confirmed result; reconcile original grant status without changing approval identity")
	}
	_, err = fmt.Fprintln(os.Stdout, string(body))
	return err
}
