package binaryfetch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ProvenanceResult records whether an optional upstream signature check ran.
// A skipped result is safe when the acquisition target already carries a
// reviewed digest; it is not equivalent to a successful signature check.
type ProvenanceResult struct {
	Verified bool   `json:"verified"`
	Skipped  bool   `json:"skipped"`
	Reason   string `json:"reason,omitempty"`
}

var (
	provenanceLookPath = exec.LookPath
	provenanceRun      = func(ctx context.Context, executable string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, executable, args...).CombinedOutput()
	}
)

// VerifyProvenance verifies a gpg-checksums declaration against already
// downloaded key, checksum-manifest, and signature files. Missing GPG tooling
// returns Skipped rather than failing a digest-pinned acquisition. A present
// toolchain and a wrong fingerprint or signature remain hard failures.
func VerifyProvenance(ctx context.Context, declaration *Provenance, keyPath, manifestPath, signaturePath string) (ProvenanceResult, error) {
	if declaration == nil || strings.TrimSpace(declaration.Kind) == "" || declaration.Kind == "none" {
		return ProvenanceResult{}, nil
	}
	if declaration.Kind != "gpg-checksums" {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: unsupported provenance kind %q", declaration.Kind)
	}
	if strings.TrimSpace(keyPath) == "" || strings.TrimSpace(manifestPath) == "" || strings.TrimSpace(signaturePath) == "" {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: gpg-checksums requires key, manifest, and signature paths")
	}
	gpg, gpgErr := provenanceLookPath("gpg")
	gpgv, gpgvErr := provenanceLookPath("gpgv")
	if gpgErr != nil || gpgvErr != nil {
		return ProvenanceResult{Skipped: true, Reason: "gpg and/or gpgv is unavailable; pinned digest remains the active trust check"}, nil
	}
	output, err := provenanceRun(ctx, gpg, "--show-keys", "--with-colons", keyPath)
	if err != nil {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: inspect provenance key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if fingerprint := strings.TrimSpace(declaration.Fingerprint); fingerprint != "" && !strings.Contains(strings.ToUpper(string(output)), strings.ToUpper(fingerprint)) {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: provenance key fingerprint does not match pinned fingerprint %q", fingerprint)
	}
	// Release publishers normally provide an ASCII-armored public key. gpgv
	// accepts a binary keyring, not the armored file directly, so materialize a
	// private temporary keyring before verification. The key is still checked
	// against the pinned fingerprint above; this conversion does not broaden
	// trust.
	keyringPath := filepath.Join(filepath.Dir(keyPath), "release-keyring.gpg")
	if output, err = provenanceRun(ctx, gpg, "--batch", "--yes", "--dearmor", "--output", keyringPath, keyPath); err != nil {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: convert provenance keyring: %w: %s", err, strings.TrimSpace(string(output)))
	}
	defer os.Remove(keyringPath)
	output, err = provenanceRun(ctx, gpgv, "--keyring", keyringPath, signaturePath, manifestPath)
	if err != nil {
		return ProvenanceResult{}, fmt.Errorf("binaryfetch: verify checksum signature: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return ProvenanceResult{Verified: true}, nil
}
