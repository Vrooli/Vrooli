package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var secretPattern = regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[:=]\s*["']?[A-Za-z0-9_\-]{16,}`)

type PolicyDecision struct {
	Allowed    bool     `json:"allowed"`
	Excluded   []string `json:"excluded"`
	Violations []string `json:"violations"`
}

type ScanLimits struct {
	MaxFiles int   `json:"maxFiles"`
	MaxBytes int64 `json:"maxBytes"`
}
type PublishPolicy struct {
	Version             int        `json:"version"`
	Limits              ScanLimits `json:"limits"`
	RequireAssetLicense bool       `json:"requireAssetLicense"`
}

func DefaultPublishPolicy() PublishPolicy {
	return PublishPolicy{Version: 1, Limits: ScanLimits{MaxFiles: 10000, MaxBytes: 512 << 20}, RequireAssetLicense: true}
}
func (p PublishPolicy) Validate() error {
	if p.Version != 1 {
		return fmt.Errorf("unsupported publish policy version %d", p.Version)
	}
	if p.Limits.MaxFiles <= 0 || p.Limits.MaxBytes <= 0 {
		return fmt.Errorf("publish policy limits must be finite and positive")
	}
	return nil
}
func (p PublishPolicy) Digest() string {
	data, _ := json.Marshal(p)
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func CheckPublishability(root string, closure Closure) (PolicyDecision, error) {
	return CheckPublishabilityWithPolicy(root, closure, DefaultPublishPolicy())
}

func CheckPublishabilityWithPolicy(root string, closure Closure, policy PublishPolicy) (PolicyDecision, error) {
	if err := policy.Validate(); err != nil {
		return PolicyDecision{}, err
	}
	decision := PolicyDecision{Allowed: true}
	if len(closure.Files) > policy.Limits.MaxFiles {
		decision.Violations = append(decision.Violations, "file count exceeds scan budget")
	}
	var totalBytes int64
	for _, file := range closure.Files {
		totalBytes += file.SizeBytes
		if privatePath(file.SourcePath) {
			decision.Excluded = append(decision.Excluded, file.SourcePath)
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.SourcePath)))
		if err != nil {
			return PolicyDecision{}, fmt.Errorf("read %s: %w", file.SourcePath, err)
		}
		if secretPattern.Match(data) {
			decision.Violations = append(decision.Violations, "secret-like content in "+file.SourcePath)
		}
		if policy.RequireAssetLicense && assetPath(file.SourcePath) && !hasLicense(root, file.SourcePath) {
			decision.Violations = append(decision.Violations, "unknown asset license: "+file.SourcePath)
		}
		if strings.HasSuffix(strings.ToLower(file.SourcePath), ".zip") || strings.HasSuffix(strings.ToLower(file.SourcePath), ".tar.gz") {
			decision.Violations = append(decision.Violations, "nested archive requires bounded scan: "+file.SourcePath)
		}
	}
	if totalBytes > policy.Limits.MaxBytes {
		decision.Violations = append(decision.Violations, "source bytes exceed scan budget")
	}
	decision.Allowed = len(decision.Violations) == 0 && len(closure.Unresolved) == 0
	return decision, nil
}

func assetPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	return (strings.HasPrefix(lower, "assets/") || strings.Contains(lower, "/assets/")) && (strings.HasSuffix(lower, ".png") || strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".gif") || strings.HasSuffix(lower, ".webp"))
}
func hasLicense(root, path string) bool {
	dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(path)))
	for _, name := range []string{"LICENSE", "LICENSE.txt", "NOTICE", "NOTICE.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
