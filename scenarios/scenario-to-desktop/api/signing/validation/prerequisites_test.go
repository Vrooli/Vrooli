package validation

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"scenario-to-desktop-api/signing/types"
	"testing"
	"time"
)

type prerequisiteTestFS struct{}

func (prerequisiteTestFS) Exists(string) bool              { return false }
func (prerequisiteTestFS) ReadFile(string) ([]byte, error) { return nil, errors.New("not found") }

type mappedPrerequisiteFS map[string][]byte

func (f mappedPrerequisiteFS) Exists(path string) bool { _, ok := f[path]; return ok }
func (f mappedPrerequisiteFS) ReadFile(path string) ([]byte, error) {
	value, ok := f[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return value, nil
}

type prerequisiteTestEnv map[string]string

func (e prerequisiteTestEnv) GetEnv(key string) string { return e[key] }
func (e prerequisiteTestEnv) LookupEnv(key string) (string, bool) {
	v, ok := e[key]
	return v, ok
}

func TestCertificateSafetyHelpersAndMetadata(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	checker := NewPrerequisiteChecker(WithTimeProvider(fixedPrerequisiteTime{now: now}))
	cert := &x509.Certificate{
		Subject:      pkix.Name{CommonName: "Desktop Signing"},
		Issuer:       pkix.Name{CommonName: "Test CA"},
		SerialNumber: big.NewInt(42),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(0, 0, 10),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning, x509.ExtKeyUsageTimeStamping},
	}
	info := checker.extractCertificateInfo(cert)
	if info.IsExpired || !info.IsCodeSign || info.DaysToExpiry != 10 {
		t.Fatalf("certificate info = %#v", info)
	}
	if !isCertificateExpiringCritical(cert.NotAfter, now) || !isCertificateExpiringWarning(cert.NotAfter, now) {
		t.Fatal("10-day certificate must be both critical and warning horizon")
	}
	if isCertificateExpired(cert.NotAfter, now) || !isCertificateExpired(now.Add(-time.Second), now) {
		t.Fatal("expiration boundaries are incorrect")
	}
	if got := extractSigntoolVersion("SignTool Version: 10.0.22621.1\n"); got != "10.0.22621.1" {
		t.Fatalf("extractSigntoolVersion() = %q", got)
	}
	if got := extractSigntoolVersion("unrelated output"); got != "" {
		t.Fatalf("unexpected version %q", got)
	}
}

type fixedPrerequisiteTime struct{ now time.Time }

func (t fixedPrerequisiteTime) Now() time.Time { return t.now }

type prerequisiteTestRunner struct {
	lookPathErr error
	runErr      error
}

func (r prerequisiteTestRunner) LookPath(string) (string, error) {
	return "/usr/bin/gpg", r.lookPathErr
}

func (r prerequisiteTestRunner) Run(context.Context, string, ...string) ([]byte, []byte, error) {
	return []byte("gpg (GnuPG) 2.4"), nil, r.runErr
}

func hasPrerequisiteCode(result *types.ValidationResult, code string) bool {
	for _, issue := range result.Errors {
		if issue.Code == code {
			return true
		}
	}
	for _, issue := range result.Warnings {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestPrerequisiteCheckerLinuxReportsMissingToolAndPassphrase(t *testing.T) {
	checker := NewPrerequisiteChecker(
		WithFileSystem(prerequisiteTestFS{}),
		WithCommandRunner(prerequisiteTestRunner{lookPathErr: errors.New("missing")}),
		WithEnvironmentReader(prerequisiteTestEnv{}),
	)
	result := checker.CheckPlatformPrerequisites(context.Background(), &types.SigningConfig{
		Enabled: true,
		Linux:   &types.LinuxSigningConfig{GPGPassphraseEnv: "SIGNING_PASSPHRASE"},
	}, types.PlatformLinux)
	if result.Valid || !hasPrerequisiteCode(result, "LINUX_GPG_NOT_FOUND") || !hasPrerequisiteCode(result, "LINUX_GPG_PASSPHRASE_ENV_NOT_SET") {
		t.Fatalf("unexpected prerequisite result: %#v", result)
	}
}

func TestPrerequisiteCheckerLinuxRejectsMissingConfiguredKey(t *testing.T) {
	checker := NewPrerequisiteChecker(
		WithFileSystem(prerequisiteTestFS{}),
		WithCommandRunner(prerequisiteTestRunner{runErr: errors.New("key missing")}),
		WithEnvironmentReader(prerequisiteTestEnv{"SIGNING_PASSPHRASE": "present"}),
	)
	result := checker.CheckPrerequisites(context.Background(), &types.SigningConfig{
		Enabled: true,
		Linux:   &types.LinuxSigningConfig{GPGKeyID: "ABC123DEF456", GPGPassphraseEnv: "SIGNING_PASSPHRASE"},
	})
	if result.Valid || !hasPrerequisiteCode(result, "LINUX_KEY_NOT_FOUND") {
		t.Fatalf("expected missing key evidence, got %#v", result)
	}
}

func TestMacOSPrerequisitesReportMissingFilesAndEnvironmentWithoutHostTools(t *testing.T) {
	checker := NewPrerequisiteChecker(
		WithFileSystem(mappedPrerequisiteFS{}),
		WithCommandRunner(prerequisiteTestRunner{}),
		WithEnvironmentReader(prerequisiteTestEnv{}),
	)
	result := NewValidationResult()
	checker.checkMacOSPrerequisites(context.Background(), &types.MacOSSigningConfig{
		AppleAPIKeyFile: "missing-key.p8", AppleIDEnv: "APPLE_ID", AppleIDPasswordEnv: "APPLE_PASSWORD", EntitlementsFile: "missing.plist",
	}, result)
	for _, code := range []string{"MACOS_API_KEY_NOT_FOUND", "MACOS_ENTITLEMENTS_NOT_FOUND", "MACOS_APPLE_ID_ENV_NOT_SET", "MACOS_APPLE_PASSWORD_ENV_NOT_SET"} {
		if !hasPrerequisiteCode(result, code) {
			t.Fatalf("expected %s in %#v", code, result)
		}
	}
	platform := result.Platforms[types.PlatformMacOS]
	if !platform.Configured || len(platform.Errors) != 2 {
		t.Fatalf("macOS platform result = %#v", platform)
	}
}

func TestMacOSIdentityDetectionAndCertificateExpirySignals(t *testing.T) {
	checker := NewPrerequisiteChecker(
		WithCommandRunner(prerequisiteTestRunner{}),
		WithTimeProvider(fixedPrerequisiteTime{now: time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)}),
	)
	result := NewValidationResult()
	pv := &types.PlatformValidation{}
	checker.checkMacOSIdentity(context.Background(), "Developer ID Application: Example (TEAM123456)", pv, result)
	if !hasPrerequisiteCode(result, "MACOS_IDENTITY_NOT_FOUND") || len(pv.Errors) != 1 {
		t.Fatalf("missing identity result = %#v %#v", pv, result)
	}

	expired := &types.CertificateInfo{NotAfter: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC), IsExpired: true}
	checker.checkCertificateExpiration(expired, types.PlatformMacOS, pv, result)
	if !hasPrerequisiteCode(result, "mac_CERT_EXPIRED") {
		t.Fatalf("expired certificate result = %#v", result)
	}
	soon := &types.CertificateInfo{NotAfter: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), DaysToExpiry: 5}
	checker.checkCertificateExpiration(soon, types.PlatformMacOS, pv, result)
	if !hasPrerequisiteCode(result, "mac_CERT_EXPIRING_SOON") {
		t.Fatalf("critical certificate warning missing: %#v", result)
	}
}
