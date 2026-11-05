package resources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/binaryfetch"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
)

const severityError = "error"

func composedManifest(lockfile, indexURL string) ResourceManifest {
	return ResourceManifest{
		Name:   "test-resource",
		Driver: "managed-service",
		ManagedService: &manifestpkg.ResourceManagedService{
			Acquisition: &binaryfetch.Acquisition{
				Kind: "composed",
				Targets: []binaryfetch.AcquisitionTarget{{
					Compose: []binaryfetch.ComposeStep{
						{Role: "python-wheels", Kind: "python-wheels", Dest: "wheels", Lockfile: lockfile, IndexURL: indexURL},
					},
				}},
			},
		},
	}
}

func writeLock(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A composed artifact pins its OUTPUT tree, so its inputs must be reproducible
// or the pin becomes unreachable. kyutai-stt shipped a "lockfile" of eight
// unhashed direct requirements while the installer ran --no-deps; the staged
// tree was neither the declared closure nor rebuildable, and the resource
// failed verification with no way back.
func TestComposedAcquisitionRejectsUnpinnedWheelLock(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "requirements.lock", "torch==2.5.1\nnumpy==1.26.4\n")

	issues := composedAcquisitionIssues(dir, composedManifest("requirements.lock", "https://example.test/simple"))
	var found bool
	for _, issue := range issues {
		if issue.Severity == severityError && strings.Contains(issue.Message, "unpinned_wheel_lockfile") {
			found = true
		}
	}
	if !found {
		t.Fatalf("composedAcquisitionIssues() = %+v, want an unpinned_wheel_lockfile error", issues)
	}
}

func TestComposedAcquisitionAcceptsHashPinnedLock(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "requirements.lock", "torch==2.5.1+cu124 \\\n    --hash=sha256:"+strings.Repeat("a", 64)+"\n")

	for _, issue := range composedAcquisitionIssues(dir, composedManifest("requirements.lock", "https://example.test/simple")) {
		if issue.Severity == severityError {
			t.Fatalf("composedAcquisitionIssues() reported %q for a hash-pinned lock", issue.Message)
		}
	}
}

// The index warning is scoped to resources that actually have an accelerated
// build to lose. This is the kyutai-stt shape: cuda declared, index implicit.
func TestComposedAcquisitionWarnsOnImplicitIndexForAcceleratedResource(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "requirements.lock", "torch==2.5.1 \\\n    --hash=sha256:"+strings.Repeat("a", 64)+"\n")

	manifest := composedManifest("requirements.lock", "")
	manifest.Acceleration = &manifestpkg.AccelerationSpec{Backends: []string{"cuda", "cpu"}}
	var warned bool
	for _, issue := range composedAcquisitionIssues(dir, manifest) {
		if issue.Severity == "warning" && strings.Contains(issue.Message, "implicit_wheel_index") {
			warned = true
		}
	}
	if !warned {
		t.Fatal("composedAcquisitionIssues() accepted an implicit index; an accelerated wheel can silently become a CPU one")
	}
}

func TestComposedAcquisitionReportsMissingLockfile(t *testing.T) {
	dir := t.TempDir()
	issues := composedAcquisitionIssues(dir, composedManifest("requirements.lock", "https://example.test/simple"))
	var found bool
	for _, issue := range issues {
		if issue.Severity == severityError && strings.Contains(issue.Message, "unreadable_wheel_lockfile") {
			found = true
		}
	}
	if !found {
		t.Fatalf("composedAcquisitionIssues() = %+v, want an unreadable_wheel_lockfile error", issues)
	}
}

// Non-composed resources must not be dragged into this rule.
func TestComposedAcquisitionIgnoresOtherKinds(t *testing.T) {
	dir := t.TempDir()
	manifest := composedManifest("requirements.lock", "")
	manifest.ManagedService.Acquisition.Kind = "url"
	if issues := composedAcquisitionIssues(dir, manifest); len(issues) != 0 {
		t.Fatalf("composedAcquisitionIssues() = %+v, want none for a url acquisition", issues)
	}
}

// A resource with no accelerated variant has nothing to silently lose, so the
// index warning must stay quiet rather than become background noise.
func TestComposedAcquisitionQuietOnImplicitIndexWithoutAcceleration(t *testing.T) {
	dir := t.TempDir()
	writeLock(t, dir, "requirements.lock", "flask==3.1.3 \\\n    --hash=sha256:"+strings.Repeat("a", 64)+"\n")

	if issues := composedAcquisitionIssues(dir, composedManifest("requirements.lock", "")); len(issues) != 0 {
		t.Fatalf("composedAcquisitionIssues() = %+v, want none for an unaccelerated resource", issues)
	}
}

func digestManifest(platform, platformDigest string, targets ...binaryfetch.AcquisitionTarget) ResourceManifest {
	m := ResourceManifest{Name: "test-resource", Driver: "managed-service", ManagedService: &manifestpkg.ResourceManagedService{
		Acquisition: &binaryfetch.Acquisition{Kind: "composed", Targets: targets},
	}}
	m.ManagedService.Artifact.SHA256ByPlatform = map[string]string{platform: platformDigest}
	return m
}

func target(when map[string]string, digest string) binaryfetch.AcquisitionTarget {
	return binaryfetch.AcquisitionTarget{When: when, ArtifactSHA256: digest}
}

// The trap: a manifest states its digest twice and only the target copy is read
// at launch, so editing one and not the other leaves a stale claim that
// silently does nothing. Finding this cost an hour of chasing the wrong number.
func TestArtifactDigestAgreementRejectsStalePlatformCopy(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	issues := artifactDigestAgreementIssues(digestManifest("linux-amd64", a,
		target(map[string]string{"os": "linux", "arch": "amd64"}, b)))
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "artifact_digest_disagreement") {
		t.Fatalf("artifactDigestAgreementIssues() = %+v, want a disagreement error", issues)
	}
}

// A platform may legitimately carry several fact-predicated targets, and the
// platform map holds only one digest. Matching ANY of them is correct.
func TestArtifactDigestAgreementAllowsFactPredicatedSiblings(t *testing.T) {
	cuda, cpu := strings.Repeat("c", 64), strings.Repeat("d", 64)
	issues := artifactDigestAgreementIssues(digestManifest("linux-amd64", cuda,
		target(map[string]string{"os": "linux", "arch": "amd64", "gpu.cuda_compute": ">=8.9"}, cuda),
		target(map[string]string{"os": "linux", "arch": "amd64"}, cpu)))
	if len(issues) != 0 {
		t.Fatalf("artifactDigestAgreementIssues() = %+v, want none", issues)
	}
}

// A platform with no digest-bearing target has nothing to disagree with.
func TestArtifactDigestAgreementIgnoresPlatformsWithoutTargetDigests(t *testing.T) {
	issues := artifactDigestAgreementIssues(digestManifest("linux-amd64", strings.Repeat("e", 64),
		target(map[string]string{"os": "macos", "arch": "arm64"}, strings.Repeat("f", 64))))
	if len(issues) != 0 {
		t.Fatalf("artifactDigestAgreementIssues() = %+v, want none", issues)
	}
}
