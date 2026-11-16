package certification

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
)

const (
	certificationAssertionArtifactKind = "scenario-to-cloud.certification.assertion"
	maxCertificationArtifactBytes      = 32 << 20
)

// TestGenieArtifactOwner reads only through Test Genie's typed catalog,
// metadata and run-scoped artifact transfer APIs. Artifact IDs are opaque.
type TestGenieArtifactOwner interface {
	ListRunArtifacts(context.Context, string, string) (*runspb.ListRunArtifactsResponse, error)
	GetRunArtifact(context.Context, string, string, string) (*runspb.ArtifactRef, error)
	DownloadRunArtifact(context.Context, string, string, string, int64) ([]byte, error)
}

// DiscoveredTestGenieArtifactOwner resolves the current Test Genie endpoint on
// every operation. Its HTTP client is used for both Connect calls and the
// owning service's opaque-ID transfer route.
type DiscoveredTestGenieArtifactOwner struct {
	HTTPClient *http.Client
	resolveURL func(context.Context) (string, error)
}

func NewDiscoveredTestGenieArtifactOwner(timeout time.Duration) *DiscoveredTestGenieArtifactOwner {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	return &DiscoveredTestGenieArtifactOwner{
		HTTPClient: client,
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
	}
}

func (o *DiscoveredTestGenieArtifactOwner) client(ctx context.Context) (runsconnect.RunsServiceClient, string, error) {
	if o == nil || o.resolveURL == nil {
		return nil, "", fmt.Errorf("Test Genie artifact owner discovery is unavailable")
	}
	baseURL, err := o.resolveURL(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("resolve Test Genie artifact owner: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, "", fmt.Errorf("Test Genie artifact owner URL is empty")
	}
	httpClient := o.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return runsconnect.NewRunsServiceClient(httpClient, baseURL), baseURL, nil
}

func (o *DiscoveredTestGenieArtifactOwner) ListRunArtifacts(ctx context.Context, scenario, runID string) (*runspb.ListRunArtifactsResponse, error) {
	client, _, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ListRunArtifacts(ctx, connect.NewRequest(&runspb.ListRunArtifactsRequest{Target: scenario, RunId: runID}))
	if err != nil {
		return nil, fmt.Errorf("discover Test Genie artifacts for %s/%s: %w", scenario, runID, err)
	}
	if response == nil || response.Msg == nil {
		return nil, fmt.Errorf("Test Genie returned no artifact catalog for %s/%s", scenario, runID)
	}
	return response.Msg, nil
}

func (o *DiscoveredTestGenieArtifactOwner) GetRunArtifact(ctx context.Context, scenario, runID, artifactID string) (*runspb.ArtifactRef, error) {
	client, _, err := o.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.GetRunArtifact(ctx, connect.NewRequest(&runspb.GetRunArtifactRequest{Target: scenario, RunId: runID, ArtifactId: artifactID}))
	if err != nil {
		return nil, fmt.Errorf("read Test Genie artifact metadata %s/%s/%s: %w", scenario, runID, artifactID, err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetArtifact() == nil {
		return nil, fmt.Errorf("Test Genie returned no metadata for artifact %s", artifactID)
	}
	return response.Msg.GetArtifact(), nil
}

func (o *DiscoveredTestGenieArtifactOwner) DownloadRunArtifact(ctx context.Context, scenario, runID, artifactID string, expectedSize int64) ([]byte, error) {
	if err := validateTestGenieArtifactSize(expectedSize); err != nil {
		return nil, err
	}
	baseURL, accessPath, err := testGenieArtifactDownloadPath(ctx, o, scenario, runID, artifactID, expectedSize)
	if err != nil {
		return nil, err
	}
	return downloadTestGenieArtifact(ctx, o.HTTPClient, baseURL, accessPath, artifactID, expectedSize)
}

func validateTestGenieArtifactSize(expectedSize int64) error {
	if expectedSize < 0 || expectedSize > maxCertificationArtifactBytes {
		return fmt.Errorf("Test Genie artifact size %d is outside the accepted range", expectedSize)
	}
	return nil
}

func testGenieArtifactDownloadPath(ctx context.Context, owner *DiscoveredTestGenieArtifactOwner, scenario, runID, artifactID string, expectedSize int64) (string, string, error) {
	_, baseURL, err := owner.client(ctx)
	if err != nil {
		return "", "", err
	}
	metadata, err := owner.GetRunArtifact(ctx, scenario, runID, artifactID)
	if err != nil {
		return "", "", err
	}
	accessPath := testGenieArtifactAccessPath(scenario, runID, artifactID)
	if metadata.GetId() != artifactID || metadata.GetAccessPath() != accessPath {
		return "", "", fmt.Errorf("Test Genie artifact metadata does not match the requested run-scoped artifact")
	}
	if metadata.GetAccessCapability() != runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM {
		return "", "", fmt.Errorf("Test Genie artifact %s does not grant stream access", artifactID)
	}
	if metadata.GetSizeBytes() != expectedSize {
		return "", "", fmt.Errorf("Test Genie artifact size changed: catalog=%d metadata=%d", expectedSize, metadata.GetSizeBytes())
	}
	if !validTestGenieArtifactBaseURL(baseURL) {
		return "", "", fmt.Errorf("Test Genie artifact owner URL is invalid")
	}
	return baseURL, accessPath, nil
}

func validTestGenieArtifactBaseURL(baseURL string) bool {
	parsed, err := url.Parse(baseURL)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

func downloadTestGenieArtifact(ctx context.Context, client *http.Client, baseURL, accessPath, artifactID string, expectedSize int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+accessPath, nil)
	if err != nil {
		return nil, fmt.Errorf("create Test Genie artifact request: %w", err)
	}
	response, err := testGenieNoRedirectClient(client).Do(request)
	if err != nil {
		return nil, fmt.Errorf("download Test Genie artifact %s: %w", artifactID, err)
	}
	defer response.Body.Close()
	return readTestGenieArtifactResponse(response, artifactID, expectedSize)
}

func testGenieNoRedirectClient(client *http.Client) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}
	noRedirectClient := *client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &noRedirectClient
}

func readTestGenieArtifactResponse(response *http.Response, artifactID string, expectedSize int64) ([]byte, error) {
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download Test Genie artifact %s returned HTTP %d", artifactID, response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength != expectedSize {
		return nil, fmt.Errorf("Test Genie artifact transfer length %d does not match metadata size %d", response.ContentLength, expectedSize)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxCertificationArtifactBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Test Genie artifact %s: %w", artifactID, err)
	}
	if len(content) > maxCertificationArtifactBytes || int64(len(content)) != expectedSize {
		return nil, fmt.Errorf("downloaded Test Genie artifact size %d does not match metadata size %d", len(content), expectedSize)
	}
	return content, nil
}

func testGenieArtifactAccessPath(scenario, runID, artifactID string) string {
	return "/api/v1/scenarios/" + url.PathEscape(scenario) + "/runs/" + url.PathEscape(runID) + "/artifacts/" + url.PathEscape(artifactID)
}

// TestGenieAssertionArtifact is the exact per-cell envelope expected from a
// Test Genie assertion artifact. Ground is carried for cross-owner binding;
// Test Genie itself proves that the bytes came from this run, while the
// independent target-owner receipt must prove the physical target identity.
type TestGenieAssertionArtifact struct {
	SchemaVersion       int       `json:"schema_version"`
	ValidationReceiptID string    `json:"validation_receipt_id"`
	RunID               string    `json:"run_id"`
	Scenario            string    `json:"scenario"`
	Cell                CellID    `json:"cell"`
	DriverRef           string    `json:"driver_ref"`
	AssertionOwner      string    `json:"assertion_owner"`
	OwnerReceiptRef     string    `json:"owner_receipt_ref"`
	Verdict             Verdict   `json:"verdict"`
	Assertion           string    `json:"assertion"`
	Candidate           Candidate `json:"candidate"`
	Target              Target    `json:"target"`
}

// TestGenieAssertionProof binds a Test Genie-owned artifact's authenticated
// bytes to the validation receipt, run, matrix cell and assertion driver.
// This proof is necessary but not sufficient to persist a passed certification
// receipt: callers must also validate the target-owner receipt.
type TestGenieAssertionProof struct {
	Owner                string                     `json:"owner"`
	ValidationReceiptID  string                     `json:"validation_receipt_id"`
	Scenario             string                     `json:"scenario"`
	RunID                string                     `json:"run_id"`
	ArtifactID           string                     `json:"artifact_id"`
	CatalogDigest        string                     `json:"catalog_digest"`
	SHA256               string                     `json:"sha256"`
	ArtifactRef          string                     `json:"artifact_ref"`
	AssertionArtifactRef string                     `json:"assertion_artifact_ref"`
	Envelope             TestGenieAssertionArtifact `json:"envelope"`
}

func validateTestGenieAssertionProof(proof *TestGenieAssertionProof, cell CellID, driverRef, receiptRef string, candidate Candidate, target Target) error {
	if proof == nil {
		return fmt.Errorf("passed receipt requires its authenticated Test Genie assertion proof")
	}
	receiptID := strings.TrimPrefix(strings.TrimSpace(receiptRef), "test-genie:")
	if err := validateTestGenieProofIdentity(proof, receiptID); err != nil {
		return err
	}
	if err := validateTestGenieProofArtifactRefs(proof, driverRef); err != nil {
		return err
	}
	matrix, err := LoadEmbedded()
	if err != nil {
		return err
	}
	caseRow, ok := matrix.Case(cell.CaseID)
	if !ok {
		return fmt.Errorf("Test Genie assertion proof references unknown cell %s", cell)
	}
	return validateTestGenieProofEnvelope(proof, cell, driverRef, receiptID, receiptRef, caseRow, candidate, target)
}

func validateTestGenieProofIdentity(proof *TestGenieAssertionProof, receiptID string) error {
	if proof.Owner != "test-genie" || receiptID == "" || proof.ValidationReceiptID != receiptID || proof.Scenario == "" || proof.RunID == "" || proof.ArtifactID == "" || !isTestGenieCatalogDigest(proof.CatalogDigest) {
		return fmt.Errorf("Test Genie assertion proof has incomplete or mismatched owner identity")
	}
	if len(proof.SHA256) != sha256.Size*2 {
		return fmt.Errorf("Test Genie assertion proof digest is malformed")
	}
	if _, err := hex.DecodeString(proof.SHA256); err != nil {
		return fmt.Errorf("Test Genie assertion proof digest is malformed")
	}
	return nil
}

func validateTestGenieProofArtifactRefs(proof *TestGenieAssertionProof, driverRef string) error {
	wantArtifactRef := fmt.Sprintf("test-genie-artifact://%s/%s/%s?sha256=%s", url.PathEscape(proof.Scenario), url.PathEscape(proof.RunID), url.PathEscape(proof.ArtifactID), proof.SHA256)
	if proof.ArtifactRef != wantArtifactRef || proof.AssertionArtifactRef != "assertion-artifact:"+driverRef+":sha256:"+proof.SHA256 {
		return fmt.Errorf("Test Genie artifact refs do not match the owner proof")
	}
	return nil
}

func validateTestGenieProofEnvelope(proof *TestGenieAssertionProof, cell CellID, driverRef, receiptID, receiptRef string, caseRow Case, candidate Candidate, target Target) error {
	envelope := proof.Envelope
	wantOwner, _ := laneEvidenceOwner(cell.Lane)
	if schemeErr := validateOwnerReceiptScheme(cell.Lane, envelope.OwnerReceiptRef); schemeErr != nil {
		return schemeErr
	}
	if scenarioOwnsAssertion(cell.Lane) && envelope.OwnerReceiptRef != receiptRef {
		return fmt.Errorf("scenario-to-cloud assertion owner receipt must equal the Test Genie validation receipt")
	}
	if !testGenieProofEnvelopeMatches(proof, cell, driverRef, receiptID, wantOwner, caseRow, candidate, target) {
		return fmt.Errorf("Test Genie assertion proof envelope does not match receipt, cell, driver, candidate and target")
	}
	return nil
}

func testGenieProofEnvelopeMatches(proof *TestGenieAssertionProof, cell CellID, driverRef, receiptID, owner string, caseRow Case, candidate Candidate, target Target) bool {
	envelope := proof.Envelope
	return testGenieProofEnvelopeIdentity(envelope, proof, cell, driverRef, receiptID, owner) && testGenieProofEnvelopeExpectation(envelope, caseRow, candidate, target)
}

func testGenieProofEnvelopeIdentity(envelope TestGenieAssertionArtifact, proof *TestGenieAssertionProof, cell CellID, driverRef, receiptID, owner string) bool {
	return envelope.SchemaVersion == 1 && envelope.ValidationReceiptID == receiptID && envelope.RunID == proof.RunID && envelope.Scenario == proof.Scenario && envelope.Cell == cell && envelope.DriverRef == driverRef && envelope.AssertionOwner == owner
}

func testGenieProofEnvelopeExpectation(envelope TestGenieAssertionArtifact, caseRow Case, candidate Candidate, target Target) bool {
	return envelope.Verdict == VerdictPassed && envelope.Assertion == caseRow.Assertion && envelope.Candidate == candidate && envelope.Target == target
}

// VerifyTestGenieAssertionArtifact verifies the actual owner receipt, resolves
// its exact successful suite run, discovers one declared cell-specific
// assertion artifact, fetches its opaque-ID bytes from Test Genie, and checks
// those bytes against the immutable matrix and supplied result.
func VerifyTestGenieAssertionArtifact(ctx context.Context, receipts ValidationReceiptReader, artifacts TestGenieArtifactOwner, result DriverResult) (TestGenieAssertionProof, error) {
	if receipts == nil || artifacts == nil {
		return TestGenieAssertionProof{}, fmt.Errorf("Test Genie receipt and artifact owners are required")
	}
	if result.Verdict != VerdictPassed {
		return TestGenieAssertionProof{}, fmt.Errorf("Test Genie assertion artifact proof only supports passed results")
	}
	ownerContext, err := resolveTestGenieAssertionContext(ctx, receipts, result)
	if err != nil {
		return TestGenieAssertionProof{}, err
	}
	catalog, artifact, err := selectTestGenieAssertionArtifact(ctx, artifacts, ownerContext, result)
	if err != nil {
		return TestGenieAssertionProof{}, err
	}
	metadata, content, err := readTestGenieAssertionArtifact(ctx, artifacts, ownerContext, artifact, result)
	if err != nil {
		return TestGenieAssertionProof{}, err
	}
	envelope, err := decodeTestGenieAssertionArtifact(content)
	if err != nil {
		return TestGenieAssertionProof{}, err
	}
	if err := validateTestGenieAssertionEnvelope(envelope, ownerContext, result, metadata.GetMetadata()["owner_receipt_ref"]); err != nil {
		return TestGenieAssertionProof{}, err
	}
	return buildTestGenieAssertionProof(catalog, artifact, ownerContext, content, envelope), nil
}

type testGenieAssertionContext struct {
	receiptID string
	scenario  string
	runID     string
	caseRow   Case
	driverRef string
	owner     string
}

func resolveTestGenieAssertionContext(ctx context.Context, receipts ValidationReceiptReader, result DriverResult) (testGenieAssertionContext, error) {
	receiptID, receipt, err := readStableAssertionReceipt(ctx, receipts, result)
	if err != nil {
		return testGenieAssertionContext{}, err
	}
	scenario, runID, err := testGenieRunForReceipt(receipt)
	if err != nil {
		return testGenieAssertionContext{}, err
	}
	validated, err := validateAssertionMatrixCell(result)
	if err != nil {
		return testGenieAssertionContext{}, err
	}
	wantDriver, err := assertionDriverForCell(validated.matrix, result)
	if err != nil {
		return testGenieAssertionContext{}, err
	}
	wantOwner, _ := laneEvidenceOwner(result.Cell.Lane)
	return testGenieAssertionContext{receiptID: receiptID, scenario: scenario, runID: runID, caseRow: validated.caseRow, driverRef: wantDriver, owner: wantOwner}, nil
}

func readStableAssertionReceipt(ctx context.Context, receipts ValidationReceiptReader, result DriverResult) (string, *validationv1.ValidationReceipt, error) {
	if err := VerifyTestGenieReceipt(ctx, receipts, result.ValidationReceiptRef, VerdictPassed); err != nil {
		return "", nil, err
	}
	receiptID := strings.TrimPrefix(strings.TrimSpace(result.ValidationReceiptRef), "test-genie:")
	receipt, err := receipts.GetValidation(ctx, receiptID)
	if err != nil {
		return "", nil, fmt.Errorf("reread Test Genie receipt %s: %w", receiptID, err)
	}
	if receipt == nil || receipt.GetReceiptId() != receiptID || receipt.GetState() != validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED {
		return "", nil, fmt.Errorf("Test Genie receipt %s changed while evidence was being read", receiptID)
	}
	if err := verifyStableTestGenieIdentity(receipt); err != nil {
		return "", nil, err
	}
	return receiptID, receipt, nil
}

func assertionDriverForCell(matrix *Matrix, result DriverResult) (string, error) {
	wantDriver, err := driverRefForMatrix(matrix, result.Cell)
	if err != nil {
		return "", err
	}
	if result.DriverRef != wantDriver {
		return "", fmt.Errorf("driver_ref must be %q for %s", wantDriver, result.Cell)
	}
	return wantDriver, nil
}

type validatedAssertionMatrixCell struct {
	matrix  *Matrix
	caseRow Case
}

func validateAssertionMatrixCell(result DriverResult) (validatedAssertionMatrixCell, error) {
	matrix, err := LoadEmbedded()
	if err != nil {
		return validatedAssertionMatrixCell{}, err
	}
	caseRow, ok := matrix.Case(result.Cell.CaseID)
	if !ok || !caseRow.Required || !containsLane(caseRow.Lanes, result.Cell.Lane) || !containsGround(matrix.TargetGroundsFor(result.Cell.CaseID, result.Cell.Lane), result.Cell.TargetGround) {
		return validatedAssertionMatrixCell{}, fmt.Errorf("cell %s is not a required target-ground matrix cell", result.Cell)
	}
	return validatedAssertionMatrixCell{matrix: matrix, caseRow: caseRow}, nil
}

func selectTestGenieAssertionArtifact(ctx context.Context, artifacts TestGenieArtifactOwner, owner testGenieAssertionContext, result DriverResult) (*runspb.ListRunArtifactsResponse, *runspb.ArtifactRef, error) {
	catalog, err := artifacts.ListRunArtifacts(ctx, owner.scenario, owner.runID)
	if err != nil {
		return nil, nil, err
	}
	if err := validateTestGenieArtifactCatalog(catalog); err != nil {
		return nil, nil, err
	}
	matches := matchingAssertionArtifacts(catalog.GetArtifacts(), owner, result.Cell)
	if len(matches) != 1 {
		return nil, nil, fmt.Errorf("Test Genie run %s has %d declared assertion artifacts for %s; exactly one is required", owner.runID, len(matches), result.Cell)
	}
	artifact := matches[0]
	if strings.TrimSpace(artifact.GetId()) == "" || artifact.GetSizeBytes() < 0 || artifact.GetSizeBytes() > maxCertificationArtifactBytes {
		return nil, nil, fmt.Errorf("Test Genie assertion artifact metadata is incomplete or too large")
	}
	return catalog, artifact, nil
}

func validateTestGenieArtifactCatalog(catalog *runspb.ListRunArtifactsResponse) error {
	if catalog == nil || catalog.GetSchemaVersion() == 0 || !isTestGenieCatalogDigest(catalog.GetDigest()) {
		return fmt.Errorf("Test Genie artifact catalog is missing its schema version or digest")
	}
	if catalog.GetLegacyDiscovered() || len(catalog.GetDegradedReasons()) != 0 {
		return fmt.Errorf("Test Genie artifact catalog is degraded or legacy-discovered")
	}
	return nil
}

func matchingAssertionArtifacts(artifacts []*runspb.ArtifactRef, owner testGenieAssertionContext, cell CellID) []*runspb.ArtifactRef {
	var matches []*runspb.ArtifactRef
	for _, artifact := range artifacts {
		if artifact == nil || artifact.GetKind() != certificationAssertionArtifactKind || artifact.GetProvenance() != runspb.ArtifactProvenance_ARTIFACT_PROVENANCE_CATALOG {
			continue
		}
		if assertionArtifactMetadataMatches(artifact.GetMetadata(), owner, cell) {
			matches = append(matches, artifact)
		}
	}
	return matches
}

func assertionArtifactMetadataMatches(metadata map[string]string, owner testGenieAssertionContext, cell CellID) bool {
	return metadata["validation_receipt_id"] == owner.receiptID && metadata["case_id"] == cell.CaseID && metadata["lane"] == string(cell.Lane) && metadata["target_ground"] == string(cell.TargetGround) && metadata["driver_ref"] == owner.driverRef && metadata["assertion_owner"] == owner.owner && validateOwnerReceiptScheme(cell.Lane, metadata["owner_receipt_ref"]) == nil
}

func readTestGenieAssertionArtifact(ctx context.Context, artifacts TestGenieArtifactOwner, owner testGenieAssertionContext, artifact *runspb.ArtifactRef, result DriverResult) (*runspb.ArtifactRef, []byte, error) {
	metadata, err := artifacts.GetRunArtifact(ctx, owner.scenario, owner.runID, artifact.GetId())
	if err != nil {
		return nil, nil, err
	}
	if !testGenieAssertionMetadataMatches(metadata, artifact, owner, result.Cell) {
		return nil, nil, fmt.Errorf("Test Genie assertion artifact metadata changed or does not bind the exact cell")
	}
	content, err := artifacts.DownloadRunArtifact(ctx, owner.scenario, owner.runID, artifact.GetId(), artifact.GetSizeBytes())
	if err != nil {
		return nil, nil, err
	}
	if int64(len(content)) != artifact.GetSizeBytes() {
		return nil, nil, fmt.Errorf("downloaded Test Genie assertion artifact size %d does not match catalog size %d", len(content), artifact.GetSizeBytes())
	}
	return metadata, content, nil
}

func testGenieAssertionMetadataMatches(metadata, artifact *runspb.ArtifactRef, owner testGenieAssertionContext, cell CellID) bool {
	return metadata != nil && metadata.GetId() == artifact.GetId() && metadata.GetKind() == certificationAssertionArtifactKind && metadata.GetProvenance() == runspb.ArtifactProvenance_ARTIFACT_PROVENANCE_CATALOG && metadata.GetSizeBytes() == artifact.GetSizeBytes() && metadata.GetAccessCapability() == runspb.ArtifactAccessCapability_ARTIFACT_ACCESS_CAPABILITY_STREAM && metadata.GetAccessPath() == testGenieArtifactAccessPath(owner.scenario, owner.runID, artifact.GetId()) && assertionArtifactMetadataMatches(metadata.GetMetadata(), owner, cell)
}

func decodeTestGenieAssertionArtifact(content []byte) (TestGenieAssertionArtifact, error) {
	var envelope TestGenieAssertionArtifact
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return TestGenieAssertionArtifact{}, fmt.Errorf("decode Test Genie assertion artifact bytes: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return TestGenieAssertionArtifact{}, fmt.Errorf("Test Genie assertion artifact contains trailing JSON data")
	}
	return envelope, nil
}

func validateTestGenieAssertionEnvelope(envelope TestGenieAssertionArtifact, owner testGenieAssertionContext, result DriverResult, metadataOwnerReceipt string) error {
	if err := validateTestGenieAssertionOwnerReceipt(envelope, result); err != nil {
		return err
	}
	if !testGenieAssertionEnvelopeMatches(envelope, owner, result, metadataOwnerReceipt) {
		return fmt.Errorf("Test Genie assertion artifact bytes do not match receipt, run, cell, driver, assertion, candidate and target")
	}
	return nil
}

func validateTestGenieAssertionOwnerReceipt(envelope TestGenieAssertionArtifact, result DriverResult) error {
	if err := validateOwnerReceiptScheme(result.Cell.Lane, envelope.OwnerReceiptRef); err != nil {
		return err
	}
	if scenarioOwnsAssertion(result.Cell.Lane) && envelope.OwnerReceiptRef != result.ValidationReceiptRef {
		return fmt.Errorf("scenario-to-cloud assertion owner receipt must equal the Test Genie validation receipt")
	}
	return nil
}

func testGenieAssertionEnvelopeMatches(envelope TestGenieAssertionArtifact, owner testGenieAssertionContext, result DriverResult, metadataOwnerReceipt string) bool {
	return testGenieEnvelopeOwnerMatches(envelope, owner, result, metadataOwnerReceipt) && envelope.Assertion == owner.caseRow.Assertion && envelope.Candidate == result.Candidate && envelope.Target == result.Target
}

func testGenieEnvelopeOwnerMatches(envelope TestGenieAssertionArtifact, owner testGenieAssertionContext, result DriverResult, metadataOwnerReceipt string) bool {
	return envelope.SchemaVersion == 1 && envelope.ValidationReceiptID == owner.receiptID && envelope.RunID == owner.runID && envelope.Scenario == owner.scenario && envelope.Cell == result.Cell && envelope.DriverRef == owner.driverRef && envelope.AssertionOwner == owner.owner && envelope.OwnerReceiptRef == metadataOwnerReceipt && envelope.Verdict == VerdictPassed
}

func scenarioOwnsAssertion(lane Lane) bool {
	return lane == LanePackage || lane == LaneAPI || lane == LaneWorkflow
}

func buildTestGenieAssertionProof(catalog *runspb.ListRunArtifactsResponse, artifact *runspb.ArtifactRef, owner testGenieAssertionContext, content []byte, envelope TestGenieAssertionArtifact) TestGenieAssertionProof {
	digest := sha256.Sum256(content)
	sha := hex.EncodeToString(digest[:])
	return TestGenieAssertionProof{
		Owner: "test-genie", ValidationReceiptID: owner.receiptID, Scenario: owner.scenario, RunID: owner.runID, ArtifactID: artifact.GetId(),
		CatalogDigest: catalog.GetDigest(), SHA256: sha,
		ArtifactRef:          fmt.Sprintf("test-genie-artifact://%s/%s/%s?sha256=%s", url.PathEscape(owner.scenario), url.PathEscape(owner.runID), url.PathEscape(artifact.GetId()), sha),
		AssertionArtifactRef: "assertion-artifact:" + owner.driverRef + ":sha256:" + sha,
		Envelope:             envelope,
	}
}

func isTestGenieCatalogDigest(value string) bool {
	const prefix = "ac:sha256:"
	if !strings.HasPrefix(value, prefix) {
		return false
	}
	encoded := strings.TrimPrefix(value, prefix)
	if len(encoded) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(encoded)
	return err == nil
}

func verifyStableTestGenieIdentity(receipt *validationv1.ValidationReceipt) error {
	admitted, observed := receipt.GetAdmittedIdentity(), receipt.GetObservedIdentity()
	if admitted == nil || observed == nil || strings.TrimSpace(admitted.GetIdentity()) == "" || admitted.GetIdentity() != observed.GetIdentity() {
		return fmt.Errorf("Test Genie receipt %s lacks one stable admitted/observed source identity", receipt.GetReceiptId())
	}
	return nil
}

func testGenieRunForReceipt(receipt *validationv1.ValidationReceipt) (string, string, error) {
	var scenario, runID string
	count := 0
	for _, evidence := range receipt.GetEvidence() {
		if evidence == nil || evidence.GetKind() != "test-genie-run" || evidence.GetOwner() != "test-genie" {
			continue
		}
		parsedScenario, parsedRunID, err := parseTestGenieRunURI(evidence.GetUri())
		if err != nil || evidence.GetEvidenceId() != parsedRunID || evidence.GetSubjectId() != parsedScenario {
			return "", "", fmt.Errorf("Test Genie receipt %s has a malformed run evidence reference", receipt.GetReceiptId())
		}
		count++
		scenario, runID = parsedScenario, parsedRunID
	}
	if count != 1 {
		return "", "", fmt.Errorf("Test Genie receipt %s has %d owner run references; exactly one is required", receipt.GetReceiptId(), count)
	}
	return scenario, runID, nil
}

func parseTestGenieRunURI(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if !validTestGenieRunURI(parsed, err) {
		return "", "", fmt.Errorf("invalid Test Genie run URI")
	}
	parts := strings.Split(strings.Trim(parsed.EscapedPath(), "/"), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid Test Genie run URI")
	}
	scenario, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid Test Genie run scenario")
	}
	runID, err := url.PathUnescape(parts[1])
	if !validTestGenieRunIdentity(scenario, runID, err) {
		return "", "", fmt.Errorf("invalid Test Genie run identity")
	}
	return scenario, runID, nil
}

func validTestGenieRunURI(parsed *url.URL, err error) bool {
	return err == nil && parsed.Scheme == "test-genie" && parsed.Host == "runs" && parsed.RawQuery == "" && parsed.Fragment == ""
}

func validTestGenieRunIdentity(scenario, runID string, err error) bool {
	return err == nil && strings.TrimSpace(scenario) != "" && strings.TrimSpace(runID) != "" && !strings.ContainsAny(runID, "/?#") && !strings.ContainsAny(scenario, "/?#")
}
