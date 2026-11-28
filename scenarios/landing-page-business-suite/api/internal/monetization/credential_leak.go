package monetization

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"landing-page-business-suite-api/internal/credentialleak"
)

const credentialLeakPhase = "credential-leak-proof"

type CredentialLeakSuiteResult struct {
	Name               string                   `json:"name"`
	Status             string                   `json:"status"`
	Command            []string                 `json:"command"`
	WorkingDir         string                   `json:"working_directory"`
	DurationMS         int64                    `json:"duration_ms"`
	ExitCode           int                      `json:"exit_code"`
	OutputScannedBytes int                      `json:"output_scanned_bytes"`
	Findings           []credentialleak.Finding `json:"findings,omitempty"`
	EvidenceRefs       []string                 `json:"evidence_refs"`
}

type CredentialLeakReport struct {
	Status           string                      `json:"status"`
	LocalContracts   string                      `json:"local_contracts"`
	InventoryStatus  string                      `json:"inventory_status"`
	Families         []credentialleak.Family     `json:"families"`
	NegativeControl  string                      `json:"negative_control"`
	TestGenArtifacts string                      `json:"test_gen_artifacts"`
	Suites           []CredentialLeakSuiteResult `json:"suites"`
	Limitations      []string                    `json:"limitations"`
}

type CredentialLeakRunner func(context.Context, string) (*CredentialLeakReport, error)

// RunCredentialLeakProof executes fixed local owner suites with a secret-free
// environment, then scans every captured output before retaining metadata.
// Output contents are never put in the provider response.
func RunCredentialLeakProof(ctx context.Context, scenarioPath string) (*CredentialLeakReport, error) {
	root, err := lpbsRepositoryRoot(scenarioPath)
	if err != nil {
		return nil, err
	}
	families, err := credentialleak.ReadManifestFamilies(filepath.Join(root, "scenarios", "landing-page-business-suite", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	if err := credentialleak.ValidateFamilies(families); err != nil {
		return nil, err
	}
	scanner, err := credentialleak.NewScanner(credentialleak.SyntheticCanaries(families))
	if err != nil {
		return nil, err
	}
	negative, err := scanner.Scan(credentialleak.Observation{Channel: "negative-control.synthetic-fault", Data: []byte("injected=" + credentialleak.SyntheticCanaries(families)[0].Value)})
	if err != nil || len(negative) != 1 {
		return nil, fmt.Errorf("synthetic leak negative control did not fire")
	}
	suites := []struct {
		name, dir string
		args      []string
		refs      []string
	}{
		{"scanner-and-artifact-negative-control", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "./internal/credentialleak", "./cmd/credential-leak-scan"}, []string{"internal/credentialleak/scanner_test.go::TestScannerFindsEncodedWrappedAndTruncatedCanaries", "internal/credentialleak/scanner_test.go::TestScannerNegativeControlFailsForEveryFamilyAndAllowsUnrelatedText", "internal/credentialleak/scanner_test.go::TestArtifactDirectoryScannerFindsLeaksWithoutRetainingValuesOrPaths", "cmd/credential-leak-scan/main_test.go::TestScannerCommandReportsSyntheticArtifactLeakAsMetadataOnly"}},
		{"stripe-mail-resolution-and-redaction", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "-run", "^(TestCredentialLeakProof|TestStripeCLIRedactsAPIAndWebhookSecrets|TestStripeSettingsConnectUpdateRedactsSecretsAndRefreshesRuntime|TestStripeSettingsConnectGetAndRevealKeepSecretsSeparated|TestSendViaSendGrid_APIError|TestPaymentSettingsServicePropagatesCredentialProviderFailure|TestPaymentSettingsServiceTreatsUnconfiguredCredentialAsOptional|TestPaymentSettingsServiceUsesStrictTestCredentialNamespace|TestPaymentSettingsServiceWithUnsetModeDoesNotResolveStripeCredentials)$", "."}, []string{"stripe_test_listener_test.go::TestStripeCLIRedactsAPIAndWebhookSecrets", "payment_settings_connect_test.go::TestStripeSettingsConnectUpdateRedactsSecretsAndRefreshesRuntime", "payment_settings_connect_test.go::TestStripeSettingsConnectGetAndRevealKeepSecretsSeparated", "payment_settings_service_test.go::TestPaymentSettingsServicePropagatesCredentialProviderFailure", "payment_settings_service_test.go::TestPaymentSettingsServiceTreatsUnconfiguredCredentialAsOptional", "payment_settings_service_test.go::TestPaymentSettingsServiceUsesStrictTestCredentialNamespace", "payment_settings_service_test.go::TestPaymentSettingsServiceWithUnsetModeDoesNotResolveStripeCredentials", "email_service_test.go::TestSendViaSendGrid_APIError"}},
		{"authority-outage-and-s3-redaction", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "./internal/administration", "./internal/delivery"}, []string{"internal/administration/credential_resolver_test.go", "internal/delivery/s3_readiness_test.go::TestS3CredentialResolutionPreservesProviderOutageAndDoesNotUseAmbientEnv", "internal/delivery/s3_readiness_test.go::TestClassifyS3ErrorNeverLeaksSecrets"}},
		{"subprocess-environment-and-truncation-guards", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "-run", "^(TestCredentialLeakSafeTestEnvironmentDropsPaidCredentials|TestCredentialLeakOutputBufferMarksExactBoundaryTruncation)$", "./internal/monetization"}, []string{"internal/monetization/credential_leak_test.go::TestCredentialLeakSafeTestEnvironmentDropsPaidCredentials", "internal/monetization/credential_leak_test.go::TestCredentialLeakOutputBufferMarksExactBoundaryTruncation"}},
		{"admin-cli-secret-receipts", "scenarios/landing-page-business-suite/cli", []string{"go", "test", "-count=1", "./internal/support"}, []string{"internal/support/admin_credentials_test.go"}},
		{"lpbs-scoped-route-identity", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "-run", "^(TestAcceptedReceiptGateAcceptsOnlyItsScopedServicePrincipal|TestDeploymentAndAnalyticsRoutesRequireTheirOwnServiceRole)$", "."}, []string{"crypto_receipt_funding_test.go::TestAcceptedReceiptGateAcceptsOnlyItsScopedServicePrincipal", "crypto_receipt_funding_test.go::TestDeploymentAndAnalyticsRoutesRequireTheirOwnServiceRole"}},
		{"lpbs-versioned-rotation-restart-recovery", "scenarios/landing-page-business-suite/api", []string{"go", "test", "-count=1", "-run", "^(TestConsumerSigningSubprocessReloadAndExternalStandardsJWKSVerifier|TestConsumerSigningAuthorityReloadUpdatesPeerReplicaAndExternalVerifier|TestConsumerSigningAuthorityReloadFailsClosedOnProviderOutage|TestConsumerSigningRotationActivatesVersionedKeyWithJWKSOverlap|TestConsumerSigningRotationFailsClosedOnAuthorityOutage|TestSyntheticCredentialAuthorityFamiliesDoNotEnterMetadataArtifacts)$", "./internal/administration"}, []string{"internal/administration/consumer_signing_process_test.go::TestConsumerSigningSubprocessReloadAndExternalStandardsJWKSVerifier", "internal/administration/consumer_signing_reload_test.go::TestConsumerSigningAuthorityReloadUpdatesPeerReplicaAndExternalVerifier", "internal/administration/consumer_signing_reload_test.go::TestConsumerSigningAuthorityReloadFailsClosedOnProviderOutage", "internal/administration/credential_witness_test.go::TestConsumerSigningRotationActivatesVersionedKeyWithJWKSOverlap", "internal/administration/credential_witness_test.go::TestConsumerSigningRotationFailsClosedOnAuthorityOutage", "internal/administration/credential_leak_synthetic_test.go::TestSyntheticCredentialAuthorityFamiliesDoNotEnterMetadataArtifacts"}},
		{"scenario-to-cloud-scoped-handoff", "scenarios/scenario-to-cloud/api", []string{"go", "test", "-count=1", "-run", "^(TestValidateHandoffSeparatesScopedAuthorizationFromLegacyBridge|TestLPBSServiceAuthorizationPrefersScopedAndOnlyFallsBackWhenUnissued|TestLPBSManifestWiresScopedAuthToDistinctTargetBoundRemoteSecretPlan)$", "./deployment"}, []string{"deployment/local_credential_handoff_test.go::TestValidateHandoffSeparatesScopedAuthorizationFromLegacyBridge", "deployment/local_credential_handoff_test.go::TestLPBSServiceAuthorizationPrefersScopedAndOnlyFallsBackWhenUnissued", "deployment/local_credential_handoff_test.go::TestLPBSManifestWiresScopedAuthToDistinctTargetBoundRemoteSecretPlan"}},
		{"scenario-to-cloud-custody-canary-surfaces", "scenarios/scenario-to-cloud/api", []string{"go", "test", "-count=1", "-run", "^(TestCanaryValuesNeverReachRetainedSurfaces|TestScannerFindsPlantedCanary)$", "./credentials"}, []string{"credentials/canary_test.go::TestCanaryValuesNeverReachRetainedSurfaces", "credentials/canary_test.go::TestScannerFindsPlantedCanary"}},
		{"scenario-to-desktop-scoped-fallback", "scenarios/scenario-to-desktop/api", []string{"go", "test", "-count=1", "-run", "^TestScenarioToDesktopLPBSAuthorizationLimitsLegacyFallbackToUnissuedPrincipal$", "./deploy"}, []string{"deploy/lpbs_client_test.go::TestScenarioToDesktopLPBSAuthorizationLimitsLegacyFallbackToUnissuedPrincipal"}},
		{"deployment-manager-scoped-fallback", "scenarios/deployment-manager/api", []string{"go", "test", "-count=1", "-run", "^(TestHTTPLPBSReleaseClientPrefersScopedTokenAndFailsClosedOnExchangeFailure|TestLegacyServiceSecretResolvedOnlyForConfirmedUnissuedPrincipal|TestLegacyServiceSecretEnvironmentIsIgnored)$", "./deployments"}, []string{"deployments/lpbs_release_client_test.go::TestHTTPLPBSReleaseClientPrefersScopedTokenAndFailsClosedOnExchangeFailure", "deployments/lpbs_release_client_test.go::TestLegacyServiceSecretResolvedOnlyForConfirmedUnissuedPrincipal", "deployments/lpbs_release_client_test.go::TestLegacyServiceSecretEnvironmentIsIgnored"}},
		{"treasury-receipt-funding-identity-and-restart", "scenarios/treasury/api", []string{"go", "test", "-count=1", "-run", "^(TestLPBSCreditLotClientSendsAcceptedReceiptAndReturnsActualOwnerID|TestLPBSCreditLotClientUsesLegacyCredentialOnlyWhenScopedIdentityUnavailable|TestLPBSCreditLotClientDoesNotUseLegacyAfterScopedPrincipalIsConfigured|TestLPBSCreditLotClientDoesNotDowngradeAfterScopedExchangeFailure|TestFundingRecoversLostAcceptanceAndRestartWithoutDuplicateCredit)$", "./internal/receiptfunding"}, []string{"internal/receiptfunding/lpbs_credit_lot_test.go::TestLPBSCreditLotClientSendsAcceptedReceiptAndReturnsActualOwnerID", "internal/receiptfunding/lpbs_credit_lot_test.go::TestLPBSCreditLotClientUsesLegacyCredentialOnlyWhenScopedIdentityUnavailable", "internal/receiptfunding/lpbs_credit_lot_test.go::TestLPBSCreditLotClientDoesNotUseLegacyAfterScopedPrincipalIsConfigured", "internal/receiptfunding/lpbs_credit_lot_test.go::TestLPBSCreditLotClientDoesNotDowngradeAfterScopedExchangeFailure", "internal/receiptfunding/service_test.go::TestFundingRecoversLostAcceptanceAndRestartWithoutDuplicateCredit"}},
	}
	results := make([]CredentialLeakSuiteResult, 0, len(suites))
	allPassed := true
	for _, suite := range suites {
		result := runCredentialLeakSuite(ctx, root, scanner, suite.name, suite.dir, suite.args, suite.refs)
		results = append(results, result)
		if result.Status != "passed" {
			allPassed = false
		}
	}
	inventoryStatus := "complete"
	for _, family := range families {
		if family.Scope == "unscoped" {
			inventoryStatus = "incomplete"
		}
	}
	status, local := "failed", "failed"
	if allPassed {
		status, local = "partial", "passed"
	}
	if inventoryStatus != "complete" {
		status = "failed"
	}
	return &CredentialLeakReport{
		Status: status, LocalContracts: local, InventoryStatus: inventoryStatus, Families: families, NegativeControl: "detected_all_families", TestGenArtifacts: "unverified",
		Suites: results,
		Limitations: []string{
			"This owner phase proves synthetic local credential paths and scans its own subprocess output; it does not prove a live provider, external verifier, production or staging behavior.",
			"Test Genie run artifacts are external to the LPBS process and must be downloaded and scanned after a supported phase run; they remain unverified until that receipt is retained.",
			"Release-authority signing credentials and foreign consumers in scenario-to-desktop, deployment-manager, Treasury, Command Center, scenario-to-cloud and Prompt Manager are outside this LPBS-owned inventory.",
		},
	}, nil
}

func runCredentialLeakSuite(ctx context.Context, root string, scanner *credentialleak.Scanner, name, relDir string, args, refs []string) CredentialLeakSuiteResult {
	result := CredentialLeakSuiteResult{Name: name, Status: "failed", Command: append([]string(nil), args...), WorkingDir: relDir, EvidenceRefs: append([]string(nil), refs...)}
	started := time.Now()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = filepath.Join(root, relDir)
	cmd.Env = cl3SafeTestEnv(nil)
	var output credentialLeakBoundedBuffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	result.DurationMS = time.Since(started).Milliseconds()
	result.OutputScannedBytes = output.Len()
	findings, scanErr := scanner.Scan(credentialleak.Observation{Channel: "owner-subprocess.stdout-stderr", Data: output.Bytes()})
	if scanErr != nil {
		result.Findings = []credentialleak.Finding{{Family: "scanner-error", Channel: "owner-subprocess.stdout-stderr", Representation: "oversized-or-invalid"}}
	}
	result.Findings = append(result.Findings, findings...)
	if err == nil && scanErr == nil && len(result.Findings) == 0 && !output.truncated {
		result.Status, result.ExitCode = "passed", 0
	} else if exit, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exit.ExitCode()
	} else if err != nil {
		result.ExitCode = -1
	} else if output.truncated {
		result.ExitCode = -2
	} else {
		result.ExitCode = -3
	}
	return result
}

type credentialLeakBoundedBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *credentialLeakBoundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	const max = 32 << 10
	remaining := max - b.Len()
	if remaining < 0 {
		remaining = 0
	}
	if len(p) > remaining {
		b.truncated = true
		p = p[:remaining]
	}
	_, _ = b.Buffer.Write(p)
	return n, nil
}

func lpbsRepositoryRoot(scenarioPath string) (string, error) {
	if strings.TrimSpace(scenarioPath) == "" {
		return "", fmt.Errorf("landing-page-business-suite path is required")
	}
	path, err := filepath.Abs(scenarioPath)
	if err != nil {
		return "", fmt.Errorf("resolve LPBS path")
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve LPBS path")
	}
	if filepath.Base(path) != "landing-page-business-suite" || filepath.Base(filepath.Dir(path)) != "scenarios" {
		return "", fmt.Errorf("path must resolve to scenarios/landing-page-business-suite")
	}
	root := filepath.Dir(filepath.Dir(path))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return "", fmt.Errorf("repository root is unavailable")
	}
	return root, nil
}

func credentialLeakPhaseResponse(report *CredentialLeakReport) *scenariovalidationv1.ValidateScenarioResponse {
	if report == nil {
		report = &CredentialLeakReport{Status: "failed", LocalContracts: "failed", NegativeControl: "unknown", TestGenArtifacts: "unverified"}
	}
	status := scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED
	if report.Status == "not_evaluated" {
		status = scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_NOT_EVALUATED
	}
	findings := make([]*commonv1.AssessmentFinding, 0, 3)
	if report.LocalContracts != "passed" {
		findings = append(findings, credentialLeakFinding("credential.local_contracts_failed", "Credential leak owner contracts did not pass", "Repair the LPBS credential consumer suites and rerun the phase.", "L0"))
	}
	if report.InventoryStatus != "complete" && report.Status != "not_evaluated" {
		findings = append(findings, credentialLeakFinding("credential.inventory_scope_incomplete", "A paid credential family lacks explicit consumer scope metadata", "Migrate each declared credential consumer to an exact identity, field, and scope binding.", "L1"))
	}
	if report.TestGenArtifacts != "verified" {
		findings = append(findings, credentialLeakFinding("credential.test_genie_artifacts_unverified", "Test Genie artifacts have not been scanned", "Download the native Test Genie run artifacts and scan them with the same synthetic sentinel inventory.", "L1"))
	}
	clean := len(findings) == 0
	level, next := "L0", "L1"
	if report.LocalContracts == "passed" {
		level, next = "L1", "L2"
	}
	codes := make([]string, 0, len(findings))
	for _, item := range findings {
		codes = append(codes, item.Code)
	}
	severity := map[string]int32{}
	if len(findings) > 0 {
		severity["FINDING_SEVERITY_ERROR"] = int32(len(findings))
	}
	assessment := &commonv1.MaturityAssessment{Scenario: "landing-page-business-suite", Provider: "landing-page-business-suite", Phase: credentialLeakPhase, Version: "0.1.0", Findings: findings, FindingsBySeverity: severity,
		Local: &commonv1.LocalMaturityAssessment{CurrentLevel: level, NextLevel: next, BlockingFindingCodes: codes, Clean: clean}}
	data, _ := json.Marshal(report)
	var reportValue map[string]any
	_ = json.Unmarshal(data, &reportValue)
	detail, _ := structpb.NewStruct(map[string]any{"credential_leak_proof": reportValue})
	native, _ := anypb.New(detail)
	return &scenariovalidationv1.ValidateScenarioResponse{Scenario: "landing-page-business-suite", Status: status, Assessment: assessment, NativeDetail: native, FailureClassification: "credential_leak_proof_incomplete"}
}

func credentialLeakFinding(code, title, remediation, level string) *commonv1.AssessmentFinding {
	return &commonv1.AssessmentFinding{Code: code, Severity: "SEVERITY_ERROR", Title: title, Message: remediation, Remediation: remediation, FixClass: "manual",
		Maturity: &commonv1.FindingMaturity{CapabilityId: "credential_leak_proof", LocalLevel: level, GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP, Dimension: "security", CleanRequirement: commonv1.CleanRequirement_CLEAN_REQUIREMENT_REQUIRED}}
}
