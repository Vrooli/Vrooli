package validation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const evidenceFingerprintVersion = "security-health-evidence-v2"

type toolDigestCacheEntry struct {
	identity string
	digest   string
}

var toolDigestCache sync.Map

func scannerEvidencePlan(ctx context.Context, cmd Commander, scanner, binary, scenarioDir string, sub Substrate, now time.Time) (ScannerEvidencePlan, error) {
	files, advisory, weight, err := scannerEvidenceFiles(ctx, cmd, scanner, scenarioDir, sub)
	if err != nil {
		return ScannerEvidencePlan{}, err
	}
	tool, err := scannerToolIdentity(cmd, binary)
	if err != nil {
		return ScannerEvidencePlan{}, err
	}
	h := sha256.New()
	writeFingerprintPart(h, evidenceFingerprintVersion)
	writeFingerprintPart(h, scanner)
	writeFingerprintPart(h, scannerPolicyVersion(scanner))
	writeFingerprintPart(h, tool)
	if advisory {
		writeFingerprintPart(h, now.UTC().Format("2006-01-02"))
	}
	for _, path := range files {
		select {
		case <-ctx.Done():
			return ScannerEvidencePlan{}, ctx.Err()
		default:
		}
		rel, err := filepath.Rel(scenarioDir, path)
		if err != nil {
			return ScannerEvidencePlan{}, fmt.Errorf("relativize scanner input %q: %w", path, err)
		}
		writeFingerprintPart(h, filepath.ToSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return ScannerEvidencePlan{}, fmt.Errorf("stat scanner input %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return ScannerEvidencePlan{}, fmt.Errorf("read scanner symlink %q: %w", path, err)
			}
			writeFingerprintPart(h, "symlink:"+target)
			continue
		}
		writeFingerprintPart(h, fmt.Sprintf("bytes:%d", info.Size()))
		file, err := os.Open(path)
		if err != nil {
			return ScannerEvidencePlan{}, fmt.Errorf("open scanner input %q: %w", path, err)
		}
		_, copyErr := io.Copy(h, file)
		closeErr := file.Close()
		if copyErr != nil {
			return ScannerEvidencePlan{}, fmt.Errorf("hash scanner input %q: %w", path, copyErr)
		}
		if closeErr != nil {
			return ScannerEvidencePlan{}, fmt.Errorf("close scanner input %q: %w", path, closeErr)
		}
	}
	ttl := 30 * 24 * time.Hour
	if advisory {
		// The UTC-day identity is the freshness boundary. A slightly longer TTL
		// permits reuse for the rest of the day without making yesterday's row
		// reachable after the fingerprint changes.
		ttl = 25 * time.Hour
	}
	return ScannerEvidencePlan{Fingerprint: hex.EncodeToString(h.Sum(nil)), Weight: weight, FreshFor: ttl}, nil
}

func writeFingerprintPart(w io.Writer, value string) {
	_, _ = io.WriteString(w, fmt.Sprintf("%d:%s\n", len(value), value))
}

func scannerPolicyVersion(scanner string) string {
	switch scanner {
	case "gitleaks":
		return "normalize-v2-ignored-info-redacted"
	case "gosec":
		return "normalize-v2-high-gates-nolint-aware"
	case "govulncheck":
		return "normalize-v1-reachable-only"
	case "pnpm-audit":
		return "normalize-v2-production-gates-rsc-aware"
	case "osv-scanner":
		return "normalize-v2-advisory-only"
	default:
		return "unknown-policy"
	}
}

func scannerEvidenceFiles(ctx context.Context, cmd Commander, scanner, scenarioDir string, sub Substrate) ([]string, bool, int64, error) {
	switch scanner {
	case "gitleaks":
		files, err := commitEligibleFiles(ctx, cmd, scenarioDir)
		return files, false, 1, err
	case "gosec":
		files, err := walkModuleEvidenceFiles(scenarioDir, sub.GoModDirs)
		return files, false, 2, err
	case "govulncheck":
		files, err := walkModuleEvidenceFiles(scenarioDir, sub.GoModDirs)
		return files, true, 3, err
	case "pnpm-audit":
		files, err := walkModuleEvidenceFiles(scenarioDir, sub.PnpmLockDirs)
		return files, true, 2, err
	case "osv-scanner":
		files, err := osvEvidenceFiles(scenarioDir)
		return files, true, 2, err
	default:
		return nil, false, 1, fmt.Errorf("scanner %q has no evidence-input policy", scanner)
	}
}

// OSVEvidenceFingerprint is the canonical cache identity for the raw OSV
// report. Dependency annotation and validation intentionally share it so the
// two consumers cannot drift into subtly different invalidation policies.
func OSVEvidenceFingerprint(ctx context.Context, cmd Commander, scenarioDir string, now time.Time) (string, error) {
	plan, err := scannerEvidencePlan(ctx, cmd, "osv-scanner", "osv-scanner", scenarioDir, Substrate{}, now)
	if err != nil {
		return "", err
	}
	return plan.Fingerprint, nil
}

func osvEvidenceFiles(scenarioDir string) ([]string, error) {
	lockfiles, err := discoverOSVLockfiles(scenarioDir)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(lockfiles))
	for _, path := range lockfiles {
		set[path] = struct{}{}
		// go.sum participates in Go module resolution even though it is not a
		// valid standalone --lockfile argument.
		if filepath.Base(path) == "go.mod" {
			sum := filepath.Join(filepath.Dir(path), "go.sum")
			if info, statErr := os.Stat(sum); statErr == nil && info.Mode().IsRegular() {
				set[sum] = struct{}{}
			}
		}
	}
	files := make([]string, 0, len(set))
	for path := range set {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func walkModuleEvidenceFiles(root string, dirs []string) ([]string, error) {
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	set := make(map[string]struct{})
	for _, rel := range dirs {
		files, err := walkEvidenceFiles(filepath.Join(root, rel), true)
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			set[path] = struct{}{}
		}
	}
	files := make([]string, 0, len(set))
	for path := range set {
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func walkEvidenceFiles(root string, excludeGenerated bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && excludeGenerated {
				if _, skip := skipDirs[entry.Name()]; skip {
					return filepath.SkipDir
				}
				if strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func scannerToolIdentity(cmd Commander, binary string) (string, error) {
	path, err := cmd.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("resolve scanner binary %q: %w", binary, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		// Test commanders and remote runners may return a logical path. It is
		// still a stable identity; production binaries take the content-digest
		// path below.
		return "logical:" + path, nil
	}
	identity := fmt.Sprintf("%s:%d:%d", path, info.Size(), info.ModTime().UnixNano())
	if cached, ok := toolDigestCache.Load(binary); ok {
		entry := cached.(toolDigestCacheEntry)
		if entry.identity == identity {
			return entry.digest, nil
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open scanner binary %q: %w", path, err)
	}
	h := sha256.New()
	_, copyErr := io.Copy(h, file)
	closeErr := file.Close()
	if copyErr != nil {
		return "", fmt.Errorf("hash scanner binary %q: %w", path, copyErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close scanner binary %q: %w", path, closeErr)
	}
	digest := identity + ":sha256:" + hex.EncodeToString(h.Sum(nil))
	toolDigestCache.Store(binary, toolDigestCacheEntry{identity: identity, digest: digest})
	return digest, nil
}
