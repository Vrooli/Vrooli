package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	bridgeartifacts "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	bridgeartifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
	bridgeruns "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	bridgerunsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

const bridgeAssertionArtifactMaxBytes = 2 << 20

// BridgeAssertionOwner is the owner read seam for relay-lane evidence.
type BridgeAssertionOwner interface {
	GetRun(context.Context, string) (*bridgeruns.Run, error)
	GetRunArtifact(context.Context, string, string) (*bridgeartifacts.GetRunArtifactResponse, error)
}

// DiscoveredBridgeAssertionOwner resolves Bridge for each call and uses the
// configured owner token. It never caches a host or silently falls back to SSH.
type DiscoveredBridgeAssertionOwner struct {
	HTTPClient    *http.Client
	ResolveURL    func(context.Context) (string, error)
	TokenProvider func(context.Context) (string, error)
}

func NewDiscoveredBridgeAssertionOwner(timeout time.Duration) *DiscoveredBridgeAssertionOwner {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &DiscoveredBridgeAssertionOwner{
		HTTPClient: &http.Client{Timeout: timeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "vrooli-bridge")
		},
		TokenProvider: bridgeAssertionOwnerToken,
	}
}

func bridgeAssertionOwnerToken(context.Context) (string, error) {
	for _, name := range []string{"VROOLI_BRIDGE_API_TOKEN", "VROOLI_BRIDGE_TOKEN", "VROOLI_API_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("no Bridge owner token is configured")
}

type ownerBearerRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (t ownerBearerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.Header = request.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

func (o *DiscoveredBridgeAssertionOwner) clients(ctx context.Context) (bridgerunsconnect.RunsServiceClient, bridgeartifactsconnect.ArtifactsServiceClient, error) {
	if !hasBridgeOwnerProviders(o) {
		return nil, nil, fmt.Errorf("Bridge owner discovery and token provider are required")
	}
	baseURL, err := o.ResolveURL(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve Bridge owner: %w", err)
	}
	if !validBridgeOwnerBaseURL(baseURL) {
		return nil, nil, fmt.Errorf("Bridge owner URL is invalid")
	}
	token, err := o.TokenProvider(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve Bridge owner token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return nil, nil, fmt.Errorf("resolve Bridge owner token: empty token")
	}
	baseClient := o.HTTPClient
	if baseClient == nil {
		baseClient = http.DefaultClient
	}
	securedClient := *baseClient
	transport := baseClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	securedClient.Transport = ownerBearerRoundTripper{base: transport, token: token}
	return bridgerunsconnect.NewRunsServiceClient(&securedClient, strings.TrimRight(baseURL, "/")), bridgeartifactsconnect.NewArtifactsServiceClient(&securedClient, strings.TrimRight(baseURL, "/")), nil
}

func hasBridgeOwnerProviders(owner *DiscoveredBridgeAssertionOwner) bool {
	return owner != nil && owner.ResolveURL != nil && owner.TokenProvider != nil
}

func validBridgeOwnerBaseURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func (o *DiscoveredBridgeAssertionOwner) GetRun(ctx context.Context, id string) (*bridgeruns.Run, error) {
	runs, _, err := o.clients(ctx)
	if err != nil {
		return nil, err
	}
	response, err := runs.GetRun(ctx, connect.NewRequest(&bridgeruns.GetRunRequest{Id: id}))
	if err != nil {
		return nil, fmt.Errorf("read Bridge run %q: %w", id, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetRun() == nil {
		return nil, fmt.Errorf("Bridge returned no run for %q", id)
	}
	return response.Msg.GetRun(), nil
}

func (o *DiscoveredBridgeAssertionOwner) GetRunArtifact(ctx context.Context, runID, name string) (*bridgeartifacts.GetRunArtifactResponse, error) {
	_, artifacts, err := o.clients(ctx)
	if err != nil {
		return nil, err
	}
	response, err := artifacts.GetRunArtifact(ctx, connect.NewRequest(&bridgeartifacts.GetRunArtifactRequest{RunId: runID, Name: name}))
	if err != nil {
		return nil, fmt.Errorf("read Bridge run artifact %q/%q: %w", runID, name, err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("Bridge returned no artifact for %q/%q", runID, name)
	}
	return response.Msg, nil
}

// BridgeAssertionOwnerProof binds the Relay lane to Bridge's durable run and
// artifact store, and proves those bytes match the Test Genie assertion bytes.
type BridgeAssertionOwnerProof struct {
	Owner            string `json:"owner"`
	RunID            string `json:"run_id"`
	NodeID           string `json:"node_id"`
	Scenario         string `json:"scenario"`
	Verb             string `json:"verb"`
	Status           string `json:"status"`
	ExitCode         int32  `json:"exit_code"`
	Artifact         string `json:"artifact"`
	ArtifactRef      string `json:"artifact_ref"`
	OwnerArtifactRef string `json:"owner_artifact_ref"`
	SHA256           string `json:"sha256"`
}

func validateBridgeAssertionOwnerProof(proof *BridgeAssertionOwnerProof, target Target, assertion *TestGenieAssertionProof) error {
	if proof == nil || assertion == nil {
		return fmt.Errorf("relay passed receipt requires Bridge and Test Genie owner proofs")
	}
	if !hasBridgeProofIdentity(proof, target) {
		return fmt.Errorf("Bridge assertion proof has incomplete or mismatched run identity")
	}
	if !matchesBridgeAssertionBytes(proof, assertion) {
		return fmt.Errorf("Bridge assertion proof digest does not match Test Genie bytes")
	}
	if !matchesBridgeOwnerReceipt(proof, assertion) {
		return fmt.Errorf("Bridge assertion proof does not match the lane owner receipt reference")
	}
	if !matchesBridgeArtifactRef(proof) {
		return fmt.Errorf("Bridge artifact ref does not bind the owner run and digest")
	}
	return nil
}

func hasBridgeProofIdentity(proof *BridgeAssertionOwnerProof, target Target) bool {
	return hasBridgeRunIdentity(proof, target) && hasBridgeArtifactIdentity(proof)
}

func hasBridgeRunIdentity(proof *BridgeAssertionOwnerProof, target Target) bool {
	return proof.Owner == "Vrooli Bridge" && proof.RunID != "" && proof.NodeID != "" && proof.NodeID == target.NodeID && proof.Status == bridgeruns.RunStatus_RUN_STATUS_PASSED.String() && proof.ExitCode == 0
}

func hasBridgeArtifactIdentity(proof *BridgeAssertionOwnerProof) bool {
	return proof.Scenario != "" && proof.Verb != "" && proof.Artifact != "" && proof.ArtifactRef != "" && proof.OwnerArtifactRef != ""
}

func matchesBridgeAssertionBytes(proof *BridgeAssertionOwnerProof, assertion *TestGenieAssertionProof) bool {
	return proof.SHA256 == assertion.SHA256 && isHexSHA256(proof.SHA256)
}

func matchesBridgeOwnerReceipt(proof *BridgeAssertionOwnerProof, assertion *TestGenieAssertionProof) bool {
	wantOwnerRef := "bridge-run://" + proof.RunID + "/" + url.PathEscape(proof.Artifact)
	return assertion.Envelope.AssertionOwner == proof.Owner && assertion.Envelope.OwnerReceiptRef == wantOwnerRef
}

func matchesBridgeArtifactRef(proof *BridgeAssertionOwnerProof) bool {
	wantRef := "bridge-run-artifact://" + proof.RunID + "/" + url.PathEscape(proof.Artifact) + "?sha256=" + proof.SHA256
	return proof.ArtifactRef == wantRef
}

func isHexSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

// VerifyBridgeAssertionOwnerReceipt reads Bridge's durable run and artifact.
// The artifact bytes must exactly match the declared Test Genie assertion bytes.
func VerifyBridgeAssertionOwnerReceipt(ctx context.Context, owner BridgeAssertionOwner, result DriverResult, assertion TestGenieAssertionProof) (BridgeAssertionOwnerProof, error) {
	if err := validateBridgeOwnerRequest(owner, result, assertion); err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	runID, name, err := parseBridgeOwnerReceipt(assertion.Envelope.OwnerReceiptRef)
	if err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	run, err := owner.GetRun(ctx, runID)
	if err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	if err := validateBridgeRun(run, runID, result.Target.NodeID); err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	artifact, err := owner.GetRunArtifact(ctx, runID, name)
	if err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	if err := validateBridgeArtifact(artifact, runID, name); err != nil {
		return BridgeAssertionOwnerProof{}, err
	}
	sha := bridgeArtifactDigest(artifact.GetData())
	if assertion.SHA256 != sha {
		return BridgeAssertionOwnerProof{}, fmt.Errorf("Bridge assertion artifact bytes do not match the Test Genie assertion artifact")
	}
	if !bridgeRunDeclaresArtifact(run, artifact.GetArtifactRef()) {
		return BridgeAssertionOwnerProof{}, fmt.Errorf("Bridge run does not declare the retrieved assertion artifact")
	}
	return buildBridgeAssertionOwnerProof(run, artifact, runID, name, sha), nil
}

func validateBridgeOwnerRequest(owner BridgeAssertionOwner, result DriverResult, assertion TestGenieAssertionProof) error {
	if result.Cell.Lane != LaneRelay {
		return fmt.Errorf("Bridge assertion owner only supports relay lane cells")
	}
	if owner == nil {
		return fmt.Errorf("Bridge assertion owner is required for relay lane")
	}
	if assertion.Envelope.AssertionOwner != "Vrooli Bridge" {
		return fmt.Errorf("relay assertion artifact is not owned by Vrooli Bridge")
	}
	return nil
}

func parseBridgeOwnerReceipt(receiptRef string) (string, string, error) {
	u, err := parseBridgeOwnerURL(receiptRef)
	if err != nil {
		return "", "", err
	}
	name, err := parseBridgeArtifactName(u.EscapedPath())
	if err != nil {
		return "", "", err
	}
	return u.Host, name, nil
}

func parseBridgeOwnerURL(receiptRef string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(receiptRef))
	if err != nil || u.Scheme != "bridge-run" || strings.TrimSpace(u.Host) == "" || u.User != nil || strings.ContainsAny(u.Host, ":@") || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("relay owner receipt must use bridge-run://<run-id>/<artifact-name>")
	}
	return u, nil
}

func parseBridgeArtifactName(escapedPath string) (string, error) {
	parts := strings.Split(strings.Trim(escapedPath, "/"), "/")
	if len(parts) != 1 {
		return "", fmt.Errorf("relay owner receipt must identify one Bridge artifact")
	}
	name, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(name) == "" || strings.ContainsAny(name, "/\\\x00") {
		return "", fmt.Errorf("relay owner receipt artifact name is invalid")
	}
	return name, nil
}

func validateBridgeRun(run *bridgeruns.Run, runID, targetNode string) error {
	if run == nil || run.GetId() != runID || run.GetNodeId() != targetNode || run.GetStatus() != bridgeruns.RunStatus_RUN_STATUS_PASSED || run.GetExitCode() != 0 || run.GetFinishedAt() == nil {
		return fmt.Errorf("Bridge run %q is not a successful terminal run for target node %q", runID, targetNode)
	}
	return nil
}

func validateBridgeArtifact(artifact *bridgeartifacts.GetRunArtifactResponse, runID, name string) error {
	if artifact.GetRunId() != runID || artifact.GetName() != name || strings.TrimSpace(artifact.GetArtifactRef()) == "" || len(artifact.GetData()) == 0 || len(artifact.GetData()) > bridgeAssertionArtifactMaxBytes {
		return fmt.Errorf("Bridge returned incomplete or oversized assertion artifact")
	}
	return nil
}

func bridgeArtifactDigest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func bridgeRunDeclaresArtifact(run *bridgeruns.Run, artifactRef string) bool {
	for _, ref := range run.GetArtifactRefs() {
		if ref == artifactRef {
			return true
		}
	}
	return false
}

func buildBridgeAssertionOwnerProof(run *bridgeruns.Run, artifact *bridgeartifacts.GetRunArtifactResponse, runID, name, sha string) BridgeAssertionOwnerProof {
	ownerRef := "bridge-run-artifact://" + runID + "/" + url.PathEscape(name) + "?sha256=" + sha
	return BridgeAssertionOwnerProof{Owner: "Vrooli Bridge", RunID: runID, NodeID: run.GetNodeId(), Scenario: run.GetScenario(), Verb: run.GetVerb(), Status: run.GetStatus().String(), ExitCode: run.GetExitCode(), Artifact: name, ArtifactRef: ownerRef, OwnerArtifactRef: artifact.GetArtifactRef(), SHA256: sha}
}
