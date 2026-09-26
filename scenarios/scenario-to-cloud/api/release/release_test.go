package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/vrooli/vrooli/packages/artifactlease"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

const canary = "canary-secret-9f3a7c1e-never-emitted"

func writeFile(t *testing.T, root, rel string, contents []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), contents, 0o644); err != nil {
		t.Fatal(err)
	}
}

func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if data, err := os.ReadFile(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("live .vrooli/repo-contract.json not found above the test package")
		}
		dir = parent
	}
}

// fixtureRepo lays out a tiny repository with one scenario and one resource
// so the real bundle profile resolves against it.
func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, ".vrooli/repo-contract.json", liveRepoContract(t))
	writeFile(t, root, ".vrooli/service.json", []byte(`{"version":"2.0.0","resources":{"store":{"enabled":true}}}`))
	writeFile(t, root, "go.mod", []byte("module fixture\n\ngo 1.24.0\n"))
	writeFile(t, root, "go.work", []byte("go 1.24.0\n"))
	writeFile(t, root, "scenarios/app/README.md", []byte("app\n"))
	writeFile(t, root, "scenarios/app/.vrooli/service.json", []byte(`{"version":"2.0.0","ports":{"api":{"range":[3000,3100]}}}`))
	writeFile(t, root, "scenarios/app/api/go.mod", []byte("module example.com/app\n\ngo 1.24\n"))
	writeFile(t, root, "resources/store/README.md", []byte("store\n"))
	writeFile(t, root, "resources/store/resource.json", []byte(`{"name":"store","deployment":{"artifacts":[{"license":"MIT"},{"license":"Apache-2.0"}]}}`))
	writeFile(t, root, "packages/pkg/README.md", []byte("pkg\n"))
	return root
}

func fixtureManifest() domain.CloudManifest {
	m := domain.CloudManifest{
		Version:     "1.0.0",
		Environment: "production",
		Target:      domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli", PreservePaths: []string{"data"}}},
		Scenario:    domain.ManifestScenario{ID: "app"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"app"}, Resources: []string{"store"},
			ClosureDigest: "sha256:" + strings.Repeat("c", 64),
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, Scenarios: []string{"app"}, Resources: []string{"store"}},
		Ports:  domain.ManifestPorts{"api": 3001},
		Edge:   domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
		Secrets: &domain.ManifestSecrets{BundleSecrets: []domain.BundleSecretPlan{{
			ID: "db-password", Class: "per_install_generated", Required: true, Description: canary,
			Format: canary, Target: domain.BundleSecretTarget{Type: "env", Name: "DB_PASSWORD"},
			Descriptor: &domain.DescriptorAddress{LogicalID: "store", Field: "password"},
			Prompt:     &domain.SecretPromptMetadata{Label: canary, Description: canary},
			Generator:  map[string]any{"value": canary, "seed": canary},
		}}},
	}
	m.Dependencies.Analyzer.Tool = "scenario-dependency-analyzer"
	m.Dependencies.Analyzer.Fingerprint = "fp-1"
	return m
}

func fixtureNativeOptions(t *testing.T, verify bool) NativeCLIOptions {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "nativecli"))
	if err != nil {
		t.Fatal(err)
	}
	return NativeCLIOptions{ModuleDir: dir, Package: ".", VerifyReproducible: verify}
}

func fixtureClosure() *domain.Closure {
	return &domain.Closure{
		Digest: "sha256:" + strings.Repeat("c", 64),
		Components: []domain.ClosureComponent{
			{ID: "store", Kind: domain.ClosureKindNativeArtifact, Artifact: &domain.ClosureArtifact{Platform: "linux/amd64", Name: "store-server", Digest: "sha256:" + strings.Repeat("e", 64), Eligibility: domain.ClosureArtifactEligible}},
			{ID: "app"},
		},
	}
}

func buildFixture(t *testing.T, store string, mutate func(*BuildInputs)) Release {
	t.Helper()
	in := BuildInputs{
		RepoRoot:  fixtureRepo(t),
		StoreDir:  store,
		Manifest:  fixtureManifest(),
		Closure:   fixtureClosure(),
		Platform:  Platform{GOOS: "linux", GOARCH: "amd64"},
		NativeCLI: fixtureNativeOptions(t, false),
		TrustMode: TrustDevelopmentLocal,
		Now:       func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) },
	}
	if mutate != nil {
		mutate(&in)
	}
	rel, err := Build(context.Background(), in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return rel
}

// TestBuildIsReproducibleForEqualPinnedInputs [REQ:STC-P0-026] proves two
// builds from the same pinned inputs yield one release identity (P10-A01)
// and that the native binary's reproducibility is recorded, never assumed.
func TestBuildIsReproducibleForEqualPinnedInputs(t *testing.T) {
	first := buildFixture(t, t.TempDir(), func(in *BuildInputs) { in.NativeCLI = fixtureNativeOptions(t, true) })
	second := buildFixture(t, t.TempDir(), func(in *BuildInputs) { in.NativeCLI = fixtureNativeOptions(t, true) })
	if first.Digest != second.Digest {
		t.Fatalf("release digests differ: %s vs %s\nfirst inputs: %+v", first.Digest, second.Digest, first.Inputs.Reproducibility)
	}
	if first.Manifest.BundleSHA256 != second.Manifest.BundleSHA256 || first.Manifest.NativeCLI != second.Manifest.NativeCLI {
		t.Fatal("bundle or native cli identity differs between equal builds")
	}
	expected, err := cloudrelease.ComputeReleaseDigest(first.Manifest)
	if err != nil || expected != first.Digest {
		t.Fatalf("release digest %s is not the canonical digest %s (%v)", first.Digest, expected, err)
	}
	if first.Manifest.ClosureDigest != fixtureClosure().Digest {
		t.Fatalf("closure digest not bound: %s", first.Manifest.ClosureDigest)
	}
	repro := first.Inputs.Reproducibility
	t.Logf("release_digest=%s native_cli reproducibility=%s reason=%q go=%s", first.Digest, repro.NativeCLI, repro.Reason, first.Inputs.NativeCLI.GoVersion)
	switch repro.NativeCLI {
	case domain.ReproducibilityReproducible:
	case domain.ReproducibilityNotReproducible:
		if repro.Reason == "" {
			t.Fatal("not_reproducible must carry a reason")
		}
		found := false
		for _, limit := range first.Inputs.Limitations {
			found = found || strings.Contains(limit, domain.ReproducibilityNotReproducible)
		}
		if !found {
			t.Fatal("not_reproducible must be listed in limitations")
		}
	default:
		t.Fatalf("reproducibility must be verified when requested, got %q", repro.NativeCLI)
	}
	if repro.Bundle != domain.ReproducibilityDeterministic {
		t.Fatalf("bundle reproducibility = %q", repro.Bundle)
	}

	// Building the same inputs into a store that already holds the release
	// returns that release; no second directory appears.
	store := Store{Dir: filepath.Dir(filepath.Dir(first.Dir))}
	before, _ := store.List()
	same := buildFixture(t, store.Dir, nil)
	after, _ := store.List()
	if same.Digest != first.Digest || len(after) != len(before) {
		t.Fatalf("rebuild into the same store must be idempotent (%d -> %d releases)", len(before), len(after))
	}

	for _, name := range []string{cloudrelease.BundleFileName, "vrooli-linux-amd64", cloudrelease.ManifestFileName, cloudrelease.InputsFileName, cloudrelease.CompleteMarker} {
		if _, err := os.Stat(filepath.Join(first.Dir, name)); err != nil {
			t.Fatalf("release is missing %s: %v", name, err)
		}
	}
	if !store.Complete(first.Digest) {
		t.Fatal("release not complete")
	}
	if _, found, err := artifactlease.Load(first.Dir); err != nil || !found {
		t.Fatalf("release lease missing: %v", err)
	}
	inputs := first.Inputs
	if inputs.Source.Snapshot != domain.ReleaseSnapshotWorkingTree || inputs.Source.ContentManifestSHA256 == "" || inputs.Source.FileCount == 0 {
		t.Fatalf("source record incomplete: %+v", inputs.Source)
	}
	if inputs.NativeCLI.GoVersion == "" || len(inputs.NativeCLI.Args) == 0 || !strings.Contains(strings.Join(inputs.NativeCLI.Args, " "), "-trimpath") || !strings.Contains(strings.Join(inputs.NativeCLI.Env, " "), "CGO_ENABLED=0") {
		t.Fatalf("native cli provenance incomplete: %+v", inputs.NativeCLI)
	}
	if len(inputs.ResourceArtifacts) != 1 || inputs.ResourceArtifacts[0].Component != "store" || strings.Join(inputs.ResourceArtifacts[0].LicenseRefs, ",") != "Apache-2.0,MIT" {
		t.Fatalf("resource artifacts: %+v", inputs.ResourceArtifacts)
	}
	if len(inputs.Credentials) != 1 || inputs.Credentials[0].Descriptor == nil || inputs.Credentials[0].Descriptor.LogicalID != "store" {
		t.Fatalf("credential refs: %+v", inputs.Credentials)
	}
	if inputs.Provenance.Policy != cloudrelease.PolicyDevelopmentLocalUnsigned || inputs.Provenance.TrustMode != string(TrustDevelopmentLocal) {
		t.Fatalf("unsigned development policy must be explicit: %+v", inputs.Provenance)
	}
	if inputs.Configuration.Digest != first.Manifest.ConfigurationDigest || inputs.Configuration.Rule == "" {
		t.Fatalf("configuration record: %+v", inputs.Configuration)
	}
}

// TestConfigurationDigestIgnoresLocatorAndSecrets [REQ:STC-P0-026] pins the
// configuration digest to nonsecret, location-independent content.
func TestConfigurationDigestIgnoresLocatorAndSecrets(t *testing.T) {
	base := fixtureManifest()
	baseDigest, err := ConfigurationDigest(base)
	if err != nil {
		t.Fatal(err)
	}
	moved := fixtureManifest()
	moved.Target.VPS.Host = "198.51.100.7"
	moved.Target.VPS.Port = 2222
	moved.Target.VPS.User = "deploy"
	moved.Target.VPS.Workdir = "/elsewhere"
	moved.Target.VPS.Workdir = "/srv/vrooli"
	moved.Secrets = nil
	movedDigest, _ := ConfigurationDigest(moved)
	if movedDigest != baseDigest {
		t.Fatal("locator or secrets changed the configuration digest")
	}
	changed := fixtureManifest()
	changed.Ports["api"] = 4000
	changedDigest, _ := ConfigurationDigest(changed)
	if changedDigest == baseDigest {
		t.Fatal("a port change must change the configuration digest")
	}
	if !cloudrelease.IsDigest(baseDigest) {
		t.Fatalf("configuration digest is not lowercase sha256 hex: %s", baseDigest)
	}
}

func copyRelease(t *testing.T, rel Release) (Store, string) {
	t.Helper()
	store := Store{Dir: t.TempDir()}
	dst := filepath.Join(store.ReleasesDir(), rel.Digest)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := filepath.Walk(rel.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(rel.Dir, path)
		target := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	}); err != nil {
		t.Fatal(err)
	}
	return store, dst
}

func verifyErr(t *testing.T, dir string, opts VerifyOptions) *apierrors.Error {
	t.Helper()
	_, err := Verify(context.Background(), dir, opts)
	if err == nil {
		t.Fatal("expected verification to fail")
	}
	typed := apierrors.As(err)
	if typed == nil {
		t.Fatalf("expected typed error, got %v", err)
	}
	return typed
}

func rewriteManifest(t *testing.T, dir string, mutate func(*cloudrelease.Manifest), recompute bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, cloudrelease.ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	var m cloudrelease.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	mutate(&m)
	if recompute {
		m.ReleaseDigest, _ = cloudrelease.ComputeReleaseDigest(m)
	}
	out, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, cloudrelease.ManifestFileName), out, 0o644); err != nil {
		t.Fatal(err)
	}
	if m.ReleaseDigest != filepath.Base(dir) {
		if err := os.Rename(dir, filepath.Join(filepath.Dir(dir), m.ReleaseDigest)); err != nil {
			t.Fatal(err)
		}
	}
}

// TestVerifyRefusesTamperedAndMismatchedReleases [REQ:STC-P0-026] covers
// P10-A02 (mismatched native CLI), P10-A03 (tampered content), P10-A05
// (incomplete stage) and the wrong-architecture refusal.
func TestVerifyRefusesTamperedAndMismatchedReleases(t *testing.T) {
	good := buildFixture(t, t.TempDir(), nil)
	dev := VerifyOptions{TrustMode: TrustDevelopmentLocal}
	if report, err := Verify(context.Background(), good.Dir, dev); err != nil || !report.Verified {
		t.Fatalf("good release must verify: %v %+v", err, report)
	}

	t.Run("tampered bundle byte", func(t *testing.T) {
		_, dir := copyRelease(t, good)
		path := filepath.Join(dir, cloudrelease.BundleFileName)
		data, _ := os.ReadFile(path)
		data[len(data)/2] ^= 0xff
		_ = os.WriteFile(path, data, 0o644)
		err := verifyErr(t, dir, dev)
		if err.Code != apierrors.CodeReleaseVerificationFailed || err.Details["check"] != "bundle_sha256" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("native cli sha mismatch", func(t *testing.T) {
		_, dir := copyRelease(t, good)
		_ = os.WriteFile(filepath.Join(dir, "vrooli-linux-amd64"), []byte("not the binary"), 0o755)
		err := verifyErr(t, dir, dev)
		if err.Code != apierrors.CodeReleaseVerificationFailed || err.Details["check"] != "native_cli" || err.Details["artifact"] != "vrooli-linux-amd64" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("wrong architecture for target", func(t *testing.T) {
		_, dir := copyRelease(t, good)
		err := verifyErr(t, dir, VerifyOptions{TrustMode: TrustDevelopmentLocal, Platform: &Platform{GOOS: "linux", GOARCH: "arm64"}})
		if err.Code != apierrors.CodeUnsupportedCapability || err.Details["artifact"] != "vrooli-linux-amd64" || err.Details["required"] != "linux/arm64" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("identity edited without digest", func(t *testing.T) {
		_, dir := copyRelease(t, good)
		rewriteManifest(t, dir, func(m *cloudrelease.Manifest) { m.ClosureDigest = "sha256:" + strings.Repeat("f", 64) }, false)
		err := verifyErr(t, dir, dev)
		if err.Details["check"] != "release_digest" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("incomplete stage", func(t *testing.T) {
		store, dir := copyRelease(t, good)
		_ = os.Remove(filepath.Join(dir, cloudrelease.CompleteMarker))
		err := verifyErr(t, dir, dev)
		if err.Details["check"] != "complete" || err.Details["reason"] != ReasonIncomplete {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
		if _, err := store.Load(good.Digest); !apierrors.Is(err, apierrors.CodeReleaseVerificationFailed) {
			t.Fatalf("store must refuse an incomplete release: %v", err)
		}
		if list, _ := store.List(); len(list) != 1 || list[0].Complete {
			t.Fatalf("list must report the release as incomplete: %+v", list)
		}
	})
	t.Run("undeclared provenance policy", func(t *testing.T) {
		_, dir := copyRelease(t, good)
		rewriteManifest(t, dir, func(m *cloudrelease.Manifest) { m.Provenance.Policy = "" }, true)
		err := verifyErr(t, dir, dev)
		if err.Details["check"] != "provenance" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("missing release", func(t *testing.T) {
		store := Store{Dir: t.TempDir()}
		if _, err := store.Load(strings.Repeat("0", 64)); apierrors.As(err) == nil || apierrors.As(err).Details["reason"] != ReasonNotFound {
			t.Fatalf("got %v", err)
		}
		if _, err := store.Load("../escape"); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
			t.Fatalf("digest traversal must be refused: %v", err)
		}
	})
}

func writeTarGz(t *testing.T, path string, entries []*tar.Header, bodies map[string][]byte) {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for _, header := range entries {
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if body, ok := bodies[header.Name]; ok {
			if _, err := tw.Write(body); err != nil {
				t.Fatal(err)
			}
		}
	}
	_ = tw.Close()
	_ = gw.Close()
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snapshotTree(t *testing.T, root string) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			out[path] = info.Size()
		}
		return nil
	})
	return out
}

// TestVerifyRefusesMaliciousArchivesBeforeAnyWrite [REQ:STC-P0-027] proves
// traversal, escaping symlinks, oversized expansion and entry bombs are
// refused by the release's own declared limits and that verification writes
// nothing (P10-A04).
func TestVerifyRefusesMaliciousArchivesBeforeAnyWrite(t *testing.T) {
	good := buildFixture(t, t.TempDir(), nil)
	dev := VerifyOptions{TrustMode: TrustDevelopmentLocal}
	regular := func(name string, size int64) *tar.Header {
		return &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: size}
	}
	cases := []struct {
		name    string
		entries []*tar.Header
		bodies  map[string][]byte
		limits  func(*cloudrelease.Limits)
		reason  string
	}{
		{name: "traversal", entries: []*tar.Header{regular("../escape.txt", 2)}, bodies: map[string][]byte{"../escape.txt": []byte("hi")}, reason: "archive_traversal"},
		{name: "absolute path", entries: []*tar.Header{regular("/etc/passwd", 2)}, bodies: map[string][]byte{"/etc/passwd": []byte("hi")}, reason: "archive_traversal"},
		{name: "escaping symlink", entries: []*tar.Header{{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "../../outside", Mode: 0o777}, regular("link/file", 2)}, bodies: map[string][]byte{"link/file": []byte("hi")}, reason: "archive_symlink_escape"},
		{name: "device entry", entries: []*tar.Header{{Name: "dev", Typeflag: tar.TypeChar, Mode: 0o600}}, reason: "archive_unsupported_entry"},
		{name: "entry bomb", entries: []*tar.Header{regular("a", 0), regular("b", 0), regular("c", 0)}, limits: func(l *cloudrelease.Limits) { l.MaxEntries = 2 }, reason: "archive_entry_limit"},
		{name: "oversized entry", entries: []*tar.Header{regular("big", 64)}, bodies: map[string][]byte{"big": bytes.Repeat([]byte("x"), 64)}, limits: func(l *cloudrelease.Limits) { l.MaxEntryBytes = 16 }, reason: "archive_too_large"},
		{name: "oversized expansion", entries: []*tar.Header{regular("one", 40), regular("two", 40)}, bodies: map[string][]byte{"one": bytes.Repeat([]byte("x"), 40), "two": bytes.Repeat([]byte("y"), 40)}, limits: func(l *cloudrelease.Limits) { l.MaxExpandedBytes = 50 }, reason: "archive_too_large"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, dir := copyRelease(t, good)
			bundlePath := filepath.Join(dir, cloudrelease.BundleFileName)
			writeTarGz(t, bundlePath, tc.entries, tc.bodies)
			sum, _, _ := fileSHA256(bundlePath)
			rewriteManifest(t, dir, func(m *cloudrelease.Manifest) {
				m.BundleSHA256 = sum
				if tc.limits != nil {
					tc.limits(&m.Limits)
				}
			}, true)
			releases, _ := store.List()
			dir = releases[0].Dir
			before := snapshotTree(t, store.Dir)
			err := verifyErr(t, dir, dev)
			if err.Code != apierrors.CodeReleaseVerificationFailed || err.Details["check"] != "archive_bounds" || err.Details["reason"] != tc.reason {
				t.Fatalf("got %s %v", err.Code, err.Details)
			}
			after := snapshotTree(t, store.Dir)
			if len(before) != len(after) {
				t.Fatalf("verification wrote into the store: %d -> %d entries", len(before), len(after))
			}
			for path, size := range before {
				if after[path] != size {
					t.Fatalf("verification modified %s", path)
				}
			}
		})
	}
}

// keySigner signs a provenance stage with an in-memory RSA key using the
// exact envelope the release authority writes. Tests only.
type keySigner struct {
	key  *rsa.PrivateKey
	fail bool
}

func (s keySigner) Sign(_ context.Context, stage string) (resourcedeployment.ReleaseSignature, error) {
	if s.fail {
		return resourcedeployment.ReleaseSignature{}, errors.New("authority refused")
	}
	manifest, err := resourcedeployment.LoadReleaseManifest(stage)
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	canonical, err := manifest.CanonicalBytes()
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	digest := sha256.Sum256(canonical)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.key, crypto.SHA256, digest[:])
	if err != nil {
		return resourcedeployment.ReleaseSignature{}, err
	}
	envelope := resourcedeployment.ReleaseSignature{SchemaVersion: "v1", KeyID: "test-key", Algorithm: "rsa-pkcs1v15-sha256", Signature: base64.StdEncoding.EncodeToString(signature)}
	raw, _ := json.Marshal(envelope)
	return envelope, os.WriteFile(filepath.Join(stage, provenanceSignatureFile), raw, 0o644)
}

func writePublicKey(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "vrooli-release.pub")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestProductionProvenanceIsSignedAndVerified [REQ:STC-P0-026] proves the
// production policy signs through the authority seam, verifies against the
// trust anchor, and refuses untrusted signers, tampered copies and unsigned
// development releases (P10-A03).
func TestProductionProvenanceIsSignedAndVerified(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	trusted := writePublicKey(t, key)
	signed := buildFixture(t, t.TempDir(), func(in *BuildInputs) {
		in.TrustMode = TrustProduction
		in.Signer = keySigner{key: key}
		in.PublicKeyPath = trusted
	})
	if signed.Manifest.Provenance.Policy != cloudrelease.PolicyProductionSigned || signed.Inputs.Provenance.SignerKeyID != "test-key" {
		t.Fatalf("provenance not recorded: %+v", signed.Inputs.Provenance)
	}
	prod := VerifyOptions{TrustMode: TrustProduction, PublicKeyPath: trusted}
	report, err := Verify(context.Background(), signed.Dir, prod)
	if err != nil || !report.Verified || report.SignerKeyID != "test-key" {
		t.Fatalf("signed release must verify in production: %v %+v", err, report)
	}
	if report, err := Verify(context.Background(), signed.Dir, VerifyOptions{TrustMode: TrustDevelopmentLocal, PublicKeyPath: trusted}); err != nil || !report.Verified {
		t.Fatalf("signed release must verify in development too: %v", err)
	}

	t.Run("untrusted signer", func(t *testing.T) {
		other, _ := rsa.GenerateKey(rand.Reader, 2048)
		err := verifyErr(t, signed.Dir, VerifyOptions{TrustMode: TrustProduction, PublicKeyPath: writePublicKey(t, other)})
		if err.Code != apierrors.CodeReleaseVerificationFailed || err.Details["check"] != "provenance" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("tampered signed copy", func(t *testing.T) {
		_, dir := copyRelease(t, signed)
		copyPath := filepath.Join(dir, cloudrelease.ProvenanceDirName, cloudrelease.ProvenanceManifestCopy)
		data, _ := os.ReadFile(copyPath)
		_ = os.WriteFile(copyPath, append(data, ' '), 0o644)
		err := verifyErr(t, dir, prod)
		if err.Details["check"] != "provenance" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("manifest edited after signing", func(t *testing.T) {
		_, dir := copyRelease(t, signed)
		rewriteManifest(t, dir, func(m *cloudrelease.Manifest) { m.Provenance.Builder = "impostor" }, true)
		err := verifyErr(t, dir, prod)
		if err.Details["check"] != "provenance" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("unsigned development release refused in production", func(t *testing.T) {
		unsigned := buildFixture(t, t.TempDir(), nil)
		err := verifyErr(t, unsigned.Dir, prod)
		if err.Details["check"] != "provenance" {
			t.Fatalf("got %s %v", err.Code, err.Details)
		}
	})
	t.Run("production build without signer is refused", func(t *testing.T) {
		store := t.TempDir()
		_, err := Build(context.Background(), BuildInputs{RepoRoot: fixtureRepo(t), StoreDir: store, Manifest: fixtureManifest(), Platform: Platform{GOOS: "linux", GOARCH: "amd64"}, NativeCLI: fixtureNativeOptions(t, false), TrustMode: TrustProduction})
		if !apierrors.Is(err, apierrors.CodeReleaseVerificationFailed) {
			t.Fatalf("got %v", err)
		}
		if entries, _ := os.ReadDir(filepath.Join(store, "releases")); len(entries) != 0 {
			t.Fatalf("a refused build must leave no release: %v", entries)
		}
	})
	t.Run("authority failure leaves no partial release", func(t *testing.T) {
		store := t.TempDir()
		_, err := Build(context.Background(), BuildInputs{RepoRoot: fixtureRepo(t), StoreDir: store, Manifest: fixtureManifest(), Platform: Platform{GOOS: "linux", GOARCH: "amd64"}, NativeCLI: fixtureNativeOptions(t, false), TrustMode: TrustProduction, Signer: keySigner{fail: true}, PublicKeyPath: trusted})
		if !apierrors.Is(err, apierrors.CodeReleaseVerificationFailed) {
			t.Fatalf("got %v", err)
		}
		if entries, _ := os.ReadDir(filepath.Join(store, "releases")); len(entries) != 0 {
			t.Fatalf("a failed signing must leave no release: %v", entries)
		}
	})
}

// TestReleaseCarriesNoCredentialValue [REQ:STC-P0-026] is the canary test
// for P10-A06: a secret planted in the manifest never reaches any emitted
// file, the bundle's embedded manifest included.
func TestReleaseCarriesNoCredentialValue(t *testing.T) {
	rel := buildFixture(t, t.TempDir(), nil)
	err := filepath.Walk(rel.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte(canary)) {
			t.Errorf("%s contains the credential canary", path)
		}
		if strings.HasSuffix(path, ".tar.gz") {
			scanArchive(t, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(filepath.Join(filepath.Dir(rel.Dir), rel.Digest+".lease.json")); bytes.Contains(raw, []byte(canary)) {
		t.Fatal("lease contains the canary")
	}
}

func scanArchive(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(tr)
		if bytes.Contains(data, []byte(canary)) {
			t.Errorf("bundle entry %s contains the credential canary", header.Name)
		}
	}
}

func TestBuildRefusesUnsupportedPlatformBeforeAnyWork(t *testing.T) {
	store := t.TempDir()
	_, err := Build(context.Background(), BuildInputs{RepoRoot: fixtureRepo(t), StoreDir: store, Manifest: fixtureManifest(), Platform: Platform{GOOS: runtime.GOOS, GOARCH: "ppc64le"}, TrustMode: TrustDevelopmentLocal})
	typed := apierrors.As(err)
	if typed == nil || typed.Code != apierrors.CodeUnsupportedCapability || typed.Details["artifact"] != "native_cli" {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(store, "releases")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unsupported platform must not touch the store")
	}
	if _, err := Build(context.Background(), BuildInputs{RepoRoot: fixtureRepo(t), StoreDir: store, Manifest: domain.CloudManifest{Scenario: domain.ManifestScenario{ID: "app"}}, Platform: Platform{GOOS: "linux", GOARCH: "amd64"}, TrustMode: TrustDevelopmentLocal}); !apierrors.Is(err, apierrors.CodeClosureUnavailable) {
		t.Fatalf("missing closure digest must be closure_unavailable: %v", err)
	}
}

func TestResolveTrustModeFromConfiguration(t *testing.T) {
	env := map[string]string{}
	lookup := func(key string) (string, bool) { v, ok := env[key]; return v, ok }
	if mode, err := ResolveTrustMode(lookup); err != nil || mode != TrustDevelopmentLocal {
		t.Fatalf("unset: %v %v", mode, err)
	}
	env[TrustModeEnv] = "production"
	if mode, err := ResolveTrustMode(lookup); err != nil || mode != TrustProduction {
		t.Fatalf("production: %v %v", mode, err)
	}
	env[TrustModeEnv] = "trust-me"
	if _, err := ResolveTrustMode(lookup); !apierrors.Is(err, apierrors.CodeInvalidRequest) {
		t.Fatalf("unknown mode must be refused: %v", err)
	}
}
