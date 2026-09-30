package cloudrelease

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestComputeReleaseDigestIsCanonical [REQ:STC-P0-026] pins the digest to the
// documented canonical bytes so the cloud builder and the target verifier
// cannot drift apart.
func TestComputeReleaseDigestIsCanonical(t *testing.T) {
	m := Manifest{
		SchemaVersion:       ManifestSchemaVersion,
		BundleSHA256:        strings.Repeat("a", 64),
		NativeCLI:           NativeCLI{SHA256: strings.Repeat("b", 64), GOOS: "linux", GOARCH: "arm64"},
		ClosureDigest:       "sha256:" + strings.Repeat("c", 64),
		ConfigurationDigest: strings.Repeat("d", 64),
		Provenance:          Provenance{Builder: "ignored", Policy: "ignored"},
		Limits:              Limits{MaxEntries: 1},
	}
	canonical, err := CanonicalJSON(m.Identity())
	if err != nil {
		t.Fatal(err)
	}
	want := `{"bundle_sha256":"` + m.BundleSHA256 + `","closure_digest":"` + m.ClosureDigest + `","configuration_digest":"` + m.ConfigurationDigest +
		`","native_cli":{"goarch":"arm64","goos":"linux","sha256":"` + m.NativeCLI.SHA256 + `"}}`
	if string(canonical) != want {
		t.Fatalf("canonical bytes\n got %s\nwant %s", canonical, want)
	}
	sum := sha256.Sum256([]byte(want))
	got, err := ComputeReleaseDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	if got != hex.EncodeToString(sum[:]) {
		t.Fatalf("digest %s != sha256 of canonical bytes", got)
	}
	m.Provenance.Builder = "someone else"
	m.Limits.MaxEntries = 99
	again, _ := ComputeReleaseDigest(m)
	if again != got {
		t.Fatal("provenance and limits must not be part of the identity")
	}
}

func TestParseManifestRejectsShapeFaults(t *testing.T) {
	good := `{"schema_version":1,"release_digest":"` + strings.Repeat("1", 64) + `","bundle_sha256":"` + strings.Repeat("2", 64) +
		`","native_cli":{"sha256":"` + strings.Repeat("3", 64) + `","goos":"linux","goarch":"amd64"},"closure_digest":"x","configuration_digest":"y","provenance":{},"limits":{}}`
	if _, err := ParseManifest([]byte(good)); err != nil {
		t.Fatalf("good manifest rejected: %v", err)
	}
	for name, raw := range map[string]string{
		"schema":      strings.Replace(good, `"schema_version":1`, `"schema_version":2`, 1),
		"digest case": strings.Replace(good, strings.Repeat("1", 64), strings.ToUpper(strings.Repeat("a", 64)), 1),
		"no goarch":   strings.Replace(good, `"goarch":"amd64"`, `"goarch":""`, 1),
		"no closure":  strings.Replace(good, `"closure_digest":"x"`, `"closure_digest":""`, 1),
		"not json":    "{",
	} {
		if _, err := ParseManifest([]byte(raw)); !errors.Is(err, ErrManifestInvalid) {
			t.Errorf("%s: want ErrManifestInvalid, got %v", name, err)
		}
	}
}

func TestExtractOptionsFillDefaults(t *testing.T) {
	deadline := time.Now().Add(time.Minute)
	opts := Manifest{Limits: Limits{MaxEntries: 5}}.ExtractOptions(deadline)
	if opts.MaxEntries != 5 || opts.MaxExpandedBytes != DefaultMaxExpandedBytes || opts.MaxEntryBytes != DefaultMaxEntryBytes || !opts.Deadline.Equal(deadline) {
		t.Fatalf("unexpected options %+v", opts)
	}
	if NativeCLIFileName("linux", "arm64") != "vrooli-linux-arm64" {
		t.Fatal("native cli file name")
	}
}
