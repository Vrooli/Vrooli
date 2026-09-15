package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	WASIEnvironment = "VROOLI_DOC_PARSE_WASM"
	HashEnvironment = "VROOLI_DOC_PARSE_SHA256"
)

type Artifact struct {
	Path   string
	SHA256 string
}

// Resolver finds the installed artifact first, then the explicit development
// root. It never builds or downloads; acquisition belongs to the control
// plane and the release stager.
type Resolver struct {
	SourceRoot       string
	ExecutablePath   string
	Environment      func(string) string
	InstalledDataDir string
}

func (r Resolver) Resolve() (Artifact, error) {
	getenv := r.Environment
	if getenv == nil {
		getenv = os.Getenv
	}
	candidates := []string{}
	if path := strings.TrimSpace(getenv(WASIEnvironment)); path != "" {
		candidates = append(candidates, path)
	}
	if dataDir := strings.TrimSpace(r.InstalledDataDir); dataDir != "" {
		candidates = append(candidates, filepath.Join(dataDir, "doc-parse.wasm"))
	} else {
		if root := strings.TrimSpace(r.SourceRoot); root != "" {
			candidates = append(candidates,
				filepath.Join(root, "artifacts", "doc-parse.wasm"),
				filepath.Join(root, "target", "wasm32-wasip1", "release", "vrooli-doc-parse-shim.wasm"),
			)
		}
		if exe := strings.TrimSpace(r.ExecutablePath); exe != "" {
			dir := filepath.Dir(exe)
			candidates = append(candidates, filepath.Join(dir, "doc-parse.wasm"), filepath.Join(dir, "vrooli-doc-parse-shim.wasm"))
		}
	}
	var seen []string
	for _, candidate := range candidates {
		candidate, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if contains(seen, candidate) {
			continue
		}
		seen = append(seen, candidate)
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		digest, err := Verify(candidate, getenv(HashEnvironment))
		if err != nil {
			return Artifact{}, err
		}
		return Artifact{Path: candidate, SHA256: digest}, nil
	}
	return Artifact{}, fmt.Errorf("doc-parse WASI artifact not found; checked %s", strings.Join(seen, ", "))
}

// Verify computes the artifact digest and compares it with the explicit
// environment value or a sibling .sha256 file. Requiring one of these sources
// prevents an unverified local file from becoming a readiness result.
func Verify(path, expected string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read doc-parse artifact %s: %w", path, err)
	}
	digestBytes := sha256.Sum256(data)
	digest := hex.EncodeToString(digestBytes[:])
	expected = strings.TrimSpace(expected)
	if expected == "" {
		if sidecar, sidecarErr := os.ReadFile(path + ".sha256"); sidecarErr == nil {
			fields := strings.Fields(string(sidecar))
			if len(fields) > 0 {
				expected = fields[0]
			}
		}
	}
	if expected == "" {
		return "", errors.New("doc-parse artifact checksum is not declared")
	}
	if !strings.EqualFold(expected, digest) {
		return "", fmt.Errorf("doc-parse artifact checksum mismatch: expected %s, got %s", expected, digest)
	}
	return digest, nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
