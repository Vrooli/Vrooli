package profilevalidation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/vrooli/browser-automation-studio/internal/cancellationqualification"
	"github.com/vrooli/browser-automation-studio/internal/evidencecompletenessqualification"
	"github.com/vrooli/browser-automation-studio/internal/motionqualification"
	"github.com/vrooli/browser-automation-studio/internal/resourcebudgetqualification"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
)

func TestValidateAcceptsCurrentOwnerReceiptsAndBuild(t *testing.T) {
	const build = "sha256:test-current"
	provider := newProviderEvidenceFixture(t, build)
	if findings := provider.validate(context.Background()); len(findings) != 0 {
		t.Fatalf("current rehabilitation owner evidence rejected: %+v", findings)
	}
}

func TestPhaseDescriptorMatchesTestGenieSchema(t *testing.T) {
	repoRoot := "../../../../../"
	scenarioRoot := "../../../"
	schemaBytes, err := os.ReadFile(filepath.Join(repoRoot, "scenarios/test-genie/schemas/test-genie-phase-descriptor.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	descriptorBytes, err := os.ReadFile(filepath.Join(scenarioRoot, ".vrooli/test-genie.json"))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const schemaURL = "https://vrooli.dev/schemas/test-genie-phase-descriptor.schema.json"
	if err := compiler.AddResource(schemaURL, bytes.NewReader(schemaBytes)); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(descriptorBytes, &value); err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(value); err != nil {
		t.Fatalf("phase descriptor rejected: %v", err)
	}
}

func TestValidateRejectsBuildMismatch(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:recorded")
	provider.deps.BuildIdentity = func(context.Context) (string, error) { return "sha256:stale", nil }
	findings := provider.validate(context.Background())
	if len(findings) != 6 {
		t.Fatalf("findings = %+v, want all six owner capabilities rejected on build mismatch", findings)
	}
}

func TestValidateReportsEvidenceCompletenessFailureSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, evidencecompletenessqualification.EvidenceDir, "evidence-completeness-test.json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "EVIDENCE_COMPLETENESS_INVALID" {
		t.Fatalf("findings = %+v, want only evidence-completeness evidence finding", findings)
	}
}

func TestValidateReportsResourceBudgetFailureSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, ".vrooli/runtime/rehabilitation-evidence/resource-budget-w189-test.json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "RESOURCE_BUDGET_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want only resource-budget evidence finding", findings)
	}
}

func TestValidateRejectsResourceBudgetOutOfBandSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, ".vrooli/runtime/rehabilitation-evidence/resource-budget-w189-test.json")
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	idle := receipt["idleSampling"].(map[string]any)
	idle["maxCombinedPssKiB"] = float64(resourcebudgetqualification.MaxIdlePSSKiB + 1)
	testutil.WriteJSONFile(t, receiptPath, receipt)
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "RESOURCE_BUDGET_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want only resource-budget out-of-band finding", findings)
	}
}

func TestValidateReportsPassiveFidelityFailureSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, ".vrooli/runtime/rehabilitation-evidence/passive-fidelity-semantics-w188-test.json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "PASSIVE_FIDELITY_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want only passive-fidelity evidence finding", findings)
	}
}

func TestValidateReportsCancellationReceiptFailureSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, ".vrooli/runtime/rehabilitation-evidence/cancellation-recovery-test-receipt.json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "CANCELLATION_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want only cancellation evidence finding", findings)
	}
}

func TestValidateReportsMotionReceiptFailureSeparately(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, filepath.FromSlash(motionqualification.EvidenceDir), "motion-receipt-test.json")
	if err := os.Remove(receiptPath); err != nil {
		t.Fatal(err)
	}
	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "MOTION_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want only motion evidence finding", findings)
	}
}

func TestValidateRejectsCancellationReceiptOutsideCleanupBand(t *testing.T) {
	provider := newProviderEvidenceFixture(t, "sha256:test-current")
	receiptPath := filepath.Join(provider.deps.ScenarioDir, ".vrooli/runtime/rehabilitation-evidence/cancellation-recovery-test-receipt.json")
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]any
	if err := json.Unmarshal(raw, &receipt); err != nil {
		t.Fatal(err)
	}
	cases := receipt["cases"].(map[string]any)
	cancel := cases["cancellation"].(map[string]any)
	cancel["cleanupMs"] = 5001
	testutil.WriteJSONFile(t, receiptPath, receipt)

	findings := provider.validate(context.Background())
	if len(findings) != 1 || findings[0].Code != "CANCELLATION_EVIDENCE_INVALID" {
		t.Fatalf("findings = %+v, want cleanup-band cancellation finding", findings)
	}
}

func newProviderEvidenceFixture(t *testing.T, build string) *provider {
	t.Helper()
	const scenarioSource = "../../../"
	root := t.TempDir()
	evidenceDir := filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fixtureSources := append([]string{
		"docs/internal/REFRACTOR_CONTRACT.json",
		"api/cmd/profile-durability-cohort/qualification.mjs",
		"playwright-driver/src/session/manager.ts",
	}, cancellationqualification.RequiredSourceFiles...)
	destinations := testutil.CopyFiles(t, scenarioSource, root, fixtureSources)
	contract := destinations["docs/internal/REFRACTOR_CONTRACT.json"]
	harness := destinations["api/cmd/profile-durability-cohort/qualification.mjs"]
	driverManager := destinations["playwright-driver/src/session/manager.ts"]
	contractSHA := fileSHAForTest(t, contract)
	harnessSHA := fileSHAForTest(t, harness)
	buildBefore := build
	seedPath := ".vrooli/runtime/rehabilitation-evidence/seed-owner.json"
	restartPath := ".vrooli/runtime/rehabilitation-evidence/restart-owner.json"

	for _, receipt := range []struct {
		path  string
		stage string
	}{{seedPath, "seed"}, {restartPath, "verify"}} {
		testutil.WriteJSONFile(t, filepath.Join(root, filepath.FromSlash(receipt.path)), map[string]any{
			"stage":          receipt.stage,
			"contractSha256": contractSHA,
			"harnessSha256":  harnessSHA,
			"beforeHealth":   map[string]any{"build_identity": buildBefore},
			"results":        []map[string]any{{"passed": true}},
		})
	}

	cohort := map[string]any{
		"contractRow":    "profile-durability",
		"contractSha256": contractSHA,
		"sourceFiles": map[string]string{
			"api/cmd/profile-durability-cohort/qualification.mjs": harnessSHA,
			"playwright-driver/src/session/manager.ts":            fileSHAForTest(t, driverManager),
		},
		"runtime": map[string]any{"managedBuildIdentityBeforeSeedAndAfterRestart": build},
		"cohort": map[string]any{
			"allChecksPassed":                          true,
			"seedChecks":                               5,
			"restartChecks":                            2,
			"checkpointVisibleAfterMs":                 1000,
			"alphaBetaIsolation":                       "passed",
			"alphaAndBetaAfterManagedApiDriverRestart": "passed",
			"syntheticProfilesDeleted":                 2,
			"seedOwnerReceipt":                         map[string]any{"path": seedPath, "sha256": fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(seedPath)))},
			"restartOwnerReceipt":                      map[string]any{"path": restartPath, "sha256": fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(restartPath)))},
		},
	}
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "profile-durability-test-receipt.json"), cohort)

	cancellationSources := map[string]string{}
	for _, relative := range cancellationqualification.RequiredSourceFiles {
		cancellationSources[relative] = fileSHAForTest(t, destinations[relative])
	}
	cancellationCases := map[string]any{}
	for _, name := range []string{"cancellation", "timeout", "driverDeath", "apiRestart", "retriedStart"} {
		terminalStatus := "failed"
		if name == "cancellation" {
			terminalStatus = "cancelled"
		}
		ownerTest := map[string]string{
			"cancellation": "TestStopExecutionRetainsUncertainOutcomeAndJoinsLeasedDriverClose",
			"timeout":      "TestExecuteTimeoutDuringLiveInstructionClosesSessionWithoutReplay",
			"driverDeath":  "TestExecuteDriverDeathRetainsUncertainEffectWithoutReplay",
			"apiRestart":   "cancellation-restart-cohort/qualification.mjs",
			"retriedStart": "close keeps a completed click uncertain while denying retry admission",
		}[name]
		cancellationCases[name] = map[string]any{
			"ownerTest": ownerTest,
			"observed":  true, "passed": true, "externalEffects": 1,
			"terminalStatus":           terminalStatus,
			"liveResourcesBeforeClose": 1, "liveResourcesAfterClose": 0,
			"inputStoppedMs": 100, "cleanupMs": 300, "recoveryMs": 500,
			"uncertainEffect": true, "retryAdmitted": false,
		}
	}
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "cancellation-recovery-test-receipt.json"), map[string]any{
		"schemaVersion":  1,
		"contractRow":    "cancellation-recovery",
		"contractSha256": contractSHA,
		"sourceFiles":    cancellationSources,
		"runtime":        map[string]any{"buildIdentity": build},
		"cases":          cancellationCases,
	})

	writePassiveFidelityFixture(t, root, build)
	writeResourceBudgetFixture(t, root, build)
	writeMotionFixture(t, scenarioSource, root, build)
	writeEvidenceCompletenessFixture(t, root, build)

	return &provider{deps: deps{
		ScenarioDir: root,
		BuildIdentity: func(context.Context) (string, error) {
			return build, nil
		},
	}}
}

func writeMotionFixture(t *testing.T, scenarioSource, root, build string) {
	t.Helper()
	sources := make(map[string]string, len(motionqualification.RequiredSources))
	destinations := testutil.CopyFiles(t, scenarioSource, root, motionqualification.RequiredSources)
	for _, relative := range motionqualification.RequiredSources {
		sources[relative] = fileSHAForTest(t, destinations[relative])
	}
	evidenceDir := filepath.Join(root, filepath.FromSlash(motionqualification.EvidenceDir))
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	artifacts := []map[string]string{}
	for _, name := range []string{"motion-owner-live.json", "motion-owner-tests.log"} {
		path := filepath.Join(evidenceDir, name)
		if err := os.WriteFile(path, []byte("motion fixture artifact:"+name), 0o600); err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, map[string]string{
			"path":   filepath.ToSlash(filepath.Join(motionqualification.EvidenceDir, name)),
			"sha256": fileSHAForTest(t, path),
		})
	}
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "motion-receipt-test.json"), map[string]any{
		"schemaVersion": 1, "contractRow": "motion", "result": "passed", "managedBuildIdentity": build,
		"contractSha256": sources["docs/internal/REFRACTOR_CONTRACT.json"], "sourceSha256": sources,
		"baseline": map[string]any{
			"durationMs": 300000, "renderedFrames": 9200, "uniqueFixtureFrames": 9200, "renderedFps": 30.67,
			"p95FrameAgeMs": 80, "maxFrameAgeMs": 99, "maxFrameBytes": 120000, "p95DecodeMs": 5, "maxDecodeMs": 20,
		},
		"slowReader": map[string]any{
			"durationMs": 18000, "stallCount": 3, "receivedFrames": 540, "decodedFrames": 220, "renderedFrames": 200,
			"maxConcurrentDecodes": 1, "maxFrameAgeMs": 530, "maxFrameBytes": 120000,
			"maxApiQueueBytes":    motionqualification.MaxFrameBytes,
			"apiQueueBudgetBytes": motionqualification.MaxFrameBytes,
		},
		"artifacts": artifacts,
	})
}

func writeEvidenceCompletenessFixture(t *testing.T, root, build string) {
	t.Helper()
	sources := make(map[string]string, len(evidencecompletenessqualification.RequiredSources))
	destinations := testutil.CopyFiles(t, "../../../", root, evidencecompletenessqualification.RequiredSources)
	for _, relative := range evidencecompletenessqualification.RequiredSources {
		sources[relative] = fileSHAForTest(t, destinations[relative])
	}
	evidenceDir := filepath.Join(root, filepath.FromSlash(evidencecompletenessqualification.EvidenceDir))
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	receipt := evidencecompletenessqualification.Receipt{
		SchemaVersion: 1,
		ContractRow:   evidencecompletenessqualification.ContractRow,
		Result:        "passed",
		BuildIdentity: build,
		SourceSHA256:  sources,
	}
	for i, tests := range [][]string{evidencecompletenessqualification.RequiredTests[:3], evidencecompletenessqualification.RequiredTests[3:]} {
		content := ""
		for _, name := range tests {
			line, err := json.Marshal(map[string]string{"Action": "pass", "Test": name})
			if err != nil {
				t.Fatal(err)
			}
			content += string(line) + "\n"
		}
		name := []string{"evidence-completeness-owner-writer-test.jsonl", "evidence-completeness-owner-retention-test.jsonl"}[i]
		path := filepath.Join(evidenceDir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		receipt.Artifacts = append(receipt.Artifacts, evidencecompletenessqualification.Artifact{
			Path: ".vrooli/runtime/rehabilitation-evidence/" + name, SHA256: fileSHAForTest(t, path), Tests: tests,
		})
	}
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "evidence-completeness-test.json"), receipt)
}

func writeResourceBudgetFixture(t *testing.T, root, build string) {
	t.Helper()
	sources := map[string]string{}
	destinations := testutil.CopyFiles(t, "../../../", root, resourcebudgetqualification.RequiredSources)
	for _, relative := range resourcebudgetqualification.RequiredSources {
		sources[relative] = fileSHAForTest(t, destinations[relative])
	}
	artifactPath := filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence/resource-budget-test.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	artifactBytes := []byte("independent resource-budget owner artifact\n")
	if err := os.WriteFile(artifactPath, artifactBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	artifactSHA := sha256.Sum256(artifactBytes)
	samples := make([]map[string]any, 61)
	for index := range samples {
		cpu := any(nil)
		if index > 0 {
			cpu = map[string]float64{"api": 0.05, "driver": 0.05}
		}
		samples[index] = map[string]any{
			"pssKiB":           map[string]int{"api": 40000, "driver": 45000},
			"combinedPssKiB":   85000,
			"cpuPercent":       cpu,
			"sessions":         0,
			"activeRecordings": 0,
		}
	}
	testutil.WriteJSONFile(t, filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence/resource-budget-w189-test.json"), map[string]any{
		"schemaVersion": 1, "contractRow": "resource-budget", "result": "passed",
		"managedBuildIdentityBefore": build, "managedBuildIdentityAfter": build,
		"platform": map[string]any{"os": "linux", "windowsPrivateMemory": map[string]string{"status": "not_measured", "reason": "fixture"}},
		"idleSampling": map[string]any{
			"startedAt": "2026-09-24T00:00:00Z", "endedAt": "2026-09-24T00:01:01Z", "durationMs": 61000,
			"sampleIntervalMs": 1000, "sampleCount": 61, "sessionsThroughout": 0, "recordingsThroughout": 0,
			"maxCombinedPssKiB": 85000, "averageCombinedCPUPercent": 0.1, "p95CombinedCPUPercent": 0.1, "samples": samples,
		},
		"fixtureBrowserAndShell": map[string]any{
			"browserProcessCount": 1, "browserPids": []map[string]int{{"pid": 10, "pssKiB": 3000}},
			"browserPssKiB": 3000, "shellPssKiB": 1000, "combinedPssKiB": 4000,
		},
		"cleanup":        map[string]bool{"fixtureSessionClosed": true, "apiHealthy": true},
		"source_sha256":  sources,
		"owner_artifact": map[string]string{"path": ".vrooli/runtime/rehabilitation-evidence/resource-budget-test.json", "sha256": hex.EncodeToString(artifactSHA[:])},
	})
}

func writePassiveFidelityFixture(t *testing.T, root, build string) {
	t.Helper()
	managedSources := []string{
		"docs/internal/REFRACTOR_CONTRACT.json",
		"playwright-driver/tests/integration/saved-workflow-fresh-context.test.ts",
		"playwright-driver/src/recording/capture/browser-scripts/recording-script.js",
		"playwright-driver/src/recording/orchestration/pipeline-manager.ts",
		"api/handlers/record_mode.go",
		"api/services/recording/service.go",
		"api/services/recording/persistence/sqlite.go",
	}
	crashSources := []string{
		"docs/internal/REFRACTOR_CONTRACT.json",
		"api/services/recording/service.go",
		"api/services/recording/service_test.go",
		"api/services/recording/persistence/sqlite.go",
	}
	semanticsSources := []string{
		"docs/internal/REFRACTOR_CONTRACT.json",
		"playwright-driver/tests/integration/pipeline-e2e.test.ts",
		"playwright-driver/src/recording/capture/browser-scripts/recording-script.js",
		"playwright-driver/src/recording/orchestration/pipeline-manager.ts",
		"playwright-driver/src/proto/recording.ts",
	}
	const scenarioSource = "../../../"
	copySet := make(map[string]bool)
	copyPaths := make([]string, 0, len(managedSources)+len(crashSources)+len(semanticsSources))
	for _, set := range [][]string{managedSources, crashSources, semanticsSources} {
		for _, relative := range set {
			if !copySet[relative] {
				copySet[relative] = true
				copyPaths = append(copyPaths, relative)
			}
		}
	}
	testutil.CopyFiles(t, scenarioSource, root, copyPaths)
	hashes := func(paths []string) map[string]string {
		result := make(map[string]string, len(paths))
		for _, relative := range paths {
			result[relative] = fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(relative)))
		}
		return result
	}
	evidenceDir := filepath.Join(root, ".vrooli/runtime/rehabilitation-evidence")
	managedOwner := map[string]any{
		"schemaVersion": 1, "contractRow": "passive-fidelity",
		"managedBuildIdentityBefore": build, "managedBuildIdentityAfter": build,
		"actions": 10000, "fixtureEffects": 10000, "uniqueJournalIds": 10000,
		"strictlyIncreasingJournalSequence": true, "appliedInputReceipts": 10000,
		"strictlyIncreasingAppliedSequence": true,
		"storageIsolation":                  map[string]any{"routedTestPool": true, "testPoolRequests": 10, "primaryRequestsDuringTestMode": 0, "temporaryDatabase": true},
	}
	managedArtifact := ".vrooli/runtime/rehabilitation-evidence/managed-owner-test.json"
	testutil.WriteJSONFile(t, filepath.Join(root, filepath.FromSlash(managedArtifact)), managedOwner)
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "passive-fidelity-managed-test.json"), map[string]any{
		"contractRow": "passive-fidelity", "result": "passed", "source_sha256": hashes(managedSources),
		"owner_receipt":  managedOwner,
		"owner_artifact": map[string]any{"path": managedArtifact, "sha256": fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(managedArtifact)))},
	})
	crashOwner := map[string]any{
		"case": "recording-service-process-death", "actionsBeforeCrash": 10000,
		"committedBeforeAcknowledgment": true, "childTerminatedAbruptly": true,
		"reopenedTotal": 10001, "retriedSameEventId": true, "totalAfterRetry": 10001,
		"expectedPrefixIntactAndOrdered": true,
	}
	crashArtifact := ".vrooli/runtime/rehabilitation-evidence/crash-owner-test.json"
	testutil.WriteJSONFile(t, filepath.Join(root, filepath.FromSlash(crashArtifact)), crashOwner)
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "passive-fidelity-process-crash-w188-test.json"), map[string]any{
		"contractRow": "passive-fidelity", "result": "passed", "tests": "1/1",
		"ownerTest": "TestJournalSameIDRetryRecoversAcrossServiceProcessDeath", "source_sha256": hashes(crashSources),
		"owner":          crashOwner,
		"owner_artifact": map[string]any{"path": crashArtifact, "sha256": fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(crashArtifact)))},
	})
	semanticsCases := []map[string]any{
		{"case": "core-events", "actionTypes": []string{"click", "type", "scroll"}, "assertions": []string{"one independent click effect", "typed value and input selector retained", "positive scroll delta retained", "unique increasing sequence numbers"}},
		{"case": "navigation", "actionTypes": []string{"click", "navigate"}, "assertions": []string{"click triggering navigation retained", "navigation entry identifies /page-2"}},
		{"case": "capture-after-navigation", "actionTypes": []string{"click", "navigate"}, "assertions": []string{"pre-navigation click retained", "navigation completed", "post-navigation click retained"}},
	}
	semanticsArtifact := "playwright-driver/.vrooli/runtime/rehabilitation-evidence/semantics-owner-test.jsonl"
	var jsonLines string
	for _, entry := range semanticsCases {
		line, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		jsonLines += string(line) + "\n"
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(semanticsArtifact))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(semanticsArtifact)), []byte(jsonLines), 0o600); err != nil {
		t.Fatal(err)
	}
	testutil.WriteJSONFile(t, filepath.Join(evidenceDir, "passive-fidelity-semantics-w188-test.json"), map[string]any{
		"contractRow": "passive-fidelity", "result": "passed", "tests": "3/3",
		"ownerTests": []string{"[CRITICAL] should capture all core event types in single session", "[CRITICAL] should capture navigation events", "[CRITICAL] should continue capturing events after navigation"},
		"cases":      semanticsCases, "source_sha256": hashes(semanticsSources),
		"owner_artifact": map[string]any{"path": semanticsArtifact, "sha256": fileSHAForTest(t, filepath.Join(root, filepath.FromSlash(semanticsArtifact)))},
	})
}

func fileSHAForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
