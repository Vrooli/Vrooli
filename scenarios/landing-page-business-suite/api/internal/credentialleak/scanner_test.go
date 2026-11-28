package credentialleak

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testFamilies() []Family {
	return []Family{
		{Name: "stripe_test", Identity: "vrooli/landing-page-business-suite", Field: "stripe-test-secret-key", Scope: "stripe:test:payments"},
		{Name: "s3", Identity: "vrooli/landing-page-business-suite", Field: "delivery-s3-secret-access-key", Scope: "s3:delivery:object-read-write"},
		{Name: "mail", Identity: "vrooli/landing-page-business-suite", Field: "sendgrid-api-key", Scope: "mail:transactional:send"},
	}
}

func TestManifestFamiliesCoverEveryPrivateDescriptorWithExplicitScope(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal("LPBS credential declaration file is unavailable")
	}
	families, err := ManifestFamilies(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateFamilies(families); err != nil {
		t.Fatal(err)
	}
	byField := map[string]Family{}
	for _, family := range families {
		byField[family.Identity+"/"+family.Field] = family
		if strings.TrimSpace(family.Scope) == "" || family.Scope == "unscoped" {
			t.Errorf("family %q has no scope", family.Name)
		}
	}
	for _, field := range []string{"stripe-test-secret-key", "stripe-test-webhook-secret", "stripe-live-secret-key", "stripe-live-webhook-secret", "delivery-s3-access-key-id", "delivery-s3-secret-access-key", "delivery-s3-session-token", "sendgrid-api-key", "mailgun-api-key", "smtp-password", "session-secret", "consumer-auth-private-key", "service-secret", "admin-default-password", "google-play-oauth-token"} {
		if _, ok := byField[lpbsIdentity+"/"+field]; !ok {
			t.Errorf("paid credential descriptor %q is missing from leak inventory", field)
		}
	}
	for _, field := range []string{"stripe-test-publishable-key", "stripe-live-publishable-key", "public-base-url", "sendgrid-webhook-public-key"} {
		if _, ok := byField[lpbsIdentity+"/"+field]; ok {
			t.Errorf("public descriptor %q was classified as a secret family", field)
		}
	}
	if external, ok := byField["vrooli/openrouter/api-key"]; !ok || external.Scope != "external-owner:scope-not-exposed" {
		t.Error("LPBS metered OpenRouter consumer is missing or its external scope was overclaimed")
	}
}

func TestSyntheticLeakChannelsCoverCredentialAndArtifactSurfaces(t *testing.T) {
	families := testFamilies()
	canaries := SyntheticCanaries(families)
	scanner, err := NewScanner(canaries)
	if err != nil {
		t.Fatal(err)
	}
	channels := []string{"authority-error", "request-response-log", "child-env-args-stdout-stderr", "database-row-json", "test-genie-result", "artifact-manifest", "operator-receipt"}
	for _, canary := range canaries {
		for _, channel := range channels {
			findings, err := scanner.Scan(Observation{Channel: channel, Data: []byte(`{"error":"` + canary.Value + `"}`)})
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) == 0 {
				t.Errorf("scanner missed synthetic control in channel %q", channel)
			}
			redacted, _, err := scanner.RedactForEvidence(Observation{Channel: channel, Data: []byte(`{"error":"` + canary.Value + `"}`)})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(redacted), canary.Value) {
				t.Errorf("evidence sanitizer retained a synthetic control in channel %q", channel)
			}
		}
	}
}

func TestScannerFindsEncodedWrappedAndTruncatedCanaries(t *testing.T) {
	canaries := SyntheticCanaries(testFamilies())
	scanner, err := NewScanner(canaries)
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range canaries {
		t.Run(canary.Family, func(t *testing.T) {
			tests := []struct{ name, content string }{
				{"raw", "stdout: " + canary.Value},
				{"base64", base64.StdEncoding.EncodeToString([]byte(canary.Value))},
				{"wrapped", strings.Join([]string{canary.Value[:18], canary.Value[18:42], canary.Value[42:]}, "\n")},
				{"mixed-case", strings.ToLower(canary.Value)},
				{"punctuation-injected", strings.Join(strings.Split(canary.Value, ""), "*")},
				{"truncated-prefix", canary.Value[:20]},
				{"truncated-middle", canary.Value[31:51]},
				{"truncated-suffix", canary.Value[len(canary.Value)-20:]},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					findings, err := scanner.Scan(Observation{Channel: "synthetic-test", Data: []byte(test.content)})
					if err != nil {
						t.Fatal(err)
					}
					if !hasFamily(findings, canary.Family) {
						t.Fatalf("scanner missed injected canary: %+v", findings)
					}
					if test.name == "base64" && len(findings) != 1 {
						t.Fatalf("encoded canary triggered cross-family false positives: %d findings", len(findings))
					}
					encoded, _, err := scanner.RedactForEvidence(Observation{Channel: "phase-output", Data: []byte(test.content)})
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(encoded), canary.Value) || strings.Contains(string(encoded), "SYNTHETIC-") {
						t.Fatal("redacted evidence retained a sentinel")
					}
				})
			}
		})
	}
}

func hasFamily(findings []Finding, name string) bool {
	for _, finding := range findings {
		if finding.Family == name {
			return true
		}
	}
	return false
}

func TestScannerNegativeControlFailsForEveryFamilyAndAllowsUnrelatedText(t *testing.T) {
	canaries := SyntheticCanaries(testFamilies())
	scanner, err := NewScanner(canaries)
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range canaries {
		findings, err := scanner.Scan(Observation{Channel: "negative-control.stderr", Data: []byte("fault detail=" + canary.Value)})
		if err != nil {
			t.Fatal(err)
		}
		if len(findings) != 1 || findings[0].Family != canary.Family || findings[0].Channel != "negative-control.stderr" {
			t.Fatalf("negative control did not identify only family/channel metadata: %+v", findings)
		}
		if strings.Contains(findings[0].Family+findings[0].Channel+findings[0].Representation, canary.Value) {
			t.Fatal("finding contains sentinel value")
		}
	}
	findings, err := scanner.Scan(Observation{Channel: "benign", Data: []byte("stripe settings updated; request_id=synthetic-123; no key material")})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("unrelated value triggered a false positive: %+v", findings)
	}
}

func TestScannerRefusesOversizedObservationInsteadOfTruncating(t *testing.T) {
	scanner, err := NewScanner(SyntheticCanaries(testFamilies()))
	if err != nil {
		t.Fatal(err)
	}
	_, err = scanner.Scan(Observation{Channel: "artifact", Data: make([]byte, maxObservationBytes+1)})
	if err != ErrObservationTooLarge {
		t.Fatalf("oversized scan error = %v", err)
	}
}

func TestArtifactDirectoryScannerFindsLeaksWithoutRetainingValuesOrPaths(t *testing.T) {
	families := testFamilies()
	canaries := SyntheticCanaries(families)
	scanner, err := NewScanner(canaries)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "phase"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "phase", "clean.json"), []byte(`{"status":"failed"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "phase", "detail.log"), []byte("synthetic-error="+canaries[0].Value), 0o600); err != nil {
		t.Fatal(err)
	}
	result := scanner.ScanArtifactDirectory(root)
	if result.Status != "leak_detected" || result.FilesScanned != 2 || len(result.Findings) == 0 {
		t.Fatalf("artifact scan status=%q files=%d findings=%d", result.Status, result.FilesScanned, len(result.Findings))
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), canaries[0].Value) || strings.Contains(string(encoded), "detail.log") {
		t.Fatal("leak report retained a canary or exposed the leaking artifact path")
	}
	for _, finding := range result.Findings {
		if finding.Family != canaries[0].Family || finding.Channel != "test-genie-artifact" {
			t.Fatalf("artifact finding is not metadata-only or family-scoped: %+v", finding)
		}
	}
}

func TestArtifactDirectoryScannerFailsClosedOnOversizeOrSymlink(t *testing.T) {
	scanner, err := NewScanner(SyntheticCanaries(testFamilies()))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "large.bin"), make([]byte, maxObservationBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	result := scanner.ScanArtifactDirectory(root)
	if result.Status != "failed" || result.FailureReason != "artifact_scan_bound_exceeded" {
		t.Fatalf("oversized artifact result = %+v", result)
	}
	linkRoot := t.TempDir()
	if err := os.Symlink(root, filepath.Join(linkRoot, "linked")); err != nil {
		t.Skip("symlinks unavailable")
	}
	result = scanner.ScanArtifactDirectory(linkRoot)
	if result.Status != "failed" || result.FailureReason != "artifact_symlink_refused" {
		t.Fatalf("symlink artifact result = %+v", result)
	}
	empty := t.TempDir()
	if err := os.Mkdir(filepath.Join(empty, "empty-subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	result = scanner.ScanTestGenieArtifactDirectory(empty)
	if result.Status != "failed" || result.FailureReason != "artifact_set_empty" {
		t.Fatalf("empty artifact set result = %+v", result)
	}
}

func TestArtifactDirectoryScannerBoundsLeakingFileCount(t *testing.T) {
	scanner, err := NewScanner(SyntheticCanaries(testFamilies()))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for i := 0; i <= maxArtifactFiles; i++ {
		name := filepath.Join(root, fmt.Sprintf("artifact-%04d.log", i))
		if err := os.WriteFile(name, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := scanner.ScanArtifactDirectory(root)
	if result.Status != "failed" || result.FailureReason != "artifact_file_count_exceeded" {
		t.Fatalf("file count bound result = status %q reason %q", result.Status, result.FailureReason)
	}
}

func TestArtifactOpenRejectsSymlinkSwapBetweenWalkAndOpen(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "artifact.log")
	replacement := filepath.Join(root, "replacement.log")
	if err := os.WriteFile(path, []byte("walked file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("replacement file"), 0o600); err != nil {
		t.Fatal(err)
	}
	walkedInfo, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	openAfterSwap := func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Symlink(replacement, name); err != nil {
			return nil, err
		}
		return os.Open(name)
	}
	file, _, err := openVerifiedArtifact(path, walkedInfo, openAfterSwap)
	if file != nil {
		_ = file.Close()
		t.Fatal("swapped artifact unexpectedly remained open")
	}
	if err == nil || err.Error() != "artifact_changed_during_scan" {
		t.Fatalf("symlink swap result = %v, want artifact_changed_during_scan", err)
	}
}
