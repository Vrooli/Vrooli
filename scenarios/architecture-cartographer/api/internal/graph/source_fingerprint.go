package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SourceFingerprinter computes a cheap source-tree fingerprint before
// language graph adapters run.
type SourceFingerprinter interface {
	Fingerprint(ctx context.Context, in ExtractGraphInput) (string, error)
}

// FileSystemFingerprinter hashes graph-affecting source files under a scenario.
type FileSystemFingerprinter struct {
	RepoRoot string
}

// NewFileSystemFingerprinter constructs the production source fingerprinter.
func NewFileSystemFingerprinter(repoRoot string) *FileSystemFingerprinter {
	return &FileSystemFingerprinter{RepoRoot: repoRoot}
}

var _ SourceFingerprinter = (*FileSystemFingerprinter)(nil)

func (f *FileSystemFingerprinter) Fingerprint(ctx context.Context, in ExtractGraphInput) (string, error) {
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return "", ErrInvalidExtractRequest{Field: "scenario", Reason: "required"}
	}
	root := filepath.Join(f.RepoRoot, "scenarios", scenario)
	if _, err := os.Stat(root); err != nil {
		return "", fmt.Errorf("stat scenario source root: %w", err)
	}

	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := d.Name()
		if d.IsDir() {
			if shouldSkipFingerprintDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !shouldFingerprintFile(name) {
			return nil
		}
		files = append(files, path)
		return nil
	}); err != nil {
		return "", fmt.Errorf("walk scenario source root: %w", err)
	}
	sort.Strings(files)

	h := sha256.New()
	fmt.Fprintf(h, "scenario=%s\nskip_ts=%t\n", scenario, in.SkipTS)
	for _, lang := range in.Languages {
		fmt.Fprintf(h, "lang=%s\n", lang)
	}
	for _, path := range files {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return "", err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read source %s: %w", rel, err)
		}
		fmt.Fprintf(h, "file=%s\nsize=%d\n", filepath.ToSlash(rel), len(b))
		_, _ = h.Write(b)
		_, _ = h.Write([]byte{'\n'})
	}
	return "src:" + hex.EncodeToString(h.Sum(nil)), nil
}

func shouldSkipFingerprintDir(name string) bool {
	switch name {
	case ".git", ".turbo", ".vite", "coverage", "data", "dist", "build", "node_modules", "tmp", "vendor":
		return true
	default:
		return false
	}
}

func shouldFingerprintFile(name string) bool {
	switch name {
	case "go.mod", "go.sum", "package.json", "pnpm-lock.yaml", "tsconfig.json", "vite.config.ts", "vite.config.js":
		return true
	}
	switch filepath.Ext(name) {
	case ".go", ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx":
		return true
	default:
		return false
	}
}
