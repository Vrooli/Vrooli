package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type BundleArtifact struct {
	Path      string `json:"path"`
	Sha256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

func FindRepoRootFromCWD() (string, error) {
	if override := strings.TrimSpace(os.Getenv("SCENARIO_TO_CLOUD_REPO_ROOT")); override != "" {
		return filepath.Clean(override), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return FindRepoRoot(cwd)
}

func FindRepoRoot(start string) (string, error) {
	dir := filepath.Clean(start)
	for i := 0; i < 20; i++ {
		if fileExists(filepath.Join(dir, "go.work")) && dirExists(filepath.Join(dir, "scenarios")) && dirExists(filepath.Join(dir, "resources")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("repo root not found from %q", start)
}

func BuildMiniVrooliBundle(repoRoot, outDir string, manifest CloudManifest) (BundleArtifact, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return BundleArtifact{}, err
	}

	outName := fmt.Sprintf("mini-vrooli_%s_%s.tar.gz", safeFilename(manifest.Scenario.ID), time.Now().UTC().Format("20060102-150405"))
	outPath := filepath.Join(outDir, outName)

	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		return BundleArtifact{}, err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return BundleArtifact{}, err
	}
	defer func() { _ = f.Close() }()

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	size, err := writeDeterministicTarGz(mw, repoRoot, spec)
	if err != nil {
		return BundleArtifact{}, err
	}
	if err := f.Sync(); err != nil {
		return BundleArtifact{}, err
	}

	return BundleArtifact{
		Path:      outPath,
		Sha256:    hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: size,
	}, nil
}

type MiniBundleSpec struct {
	IncludeRoots []string
	Excludes     []string
	ExtraFiles   map[string][]byte // relative path -> contents
}

func MiniVrooliBundleSpec(repoRoot string, manifest CloudManifest) (MiniBundleSpec, error) {
	scenarioIDs := stableUniqueStrings(manifest.Bundle.Scenarios)
	resourceIDs := stableUniqueStrings(manifest.Bundle.Resources)

	if manifest.Bundle.IncludeAutoheal && !contains(scenarioIDs, "vrooli-autoheal") {
		scenarioIDs = append(scenarioIDs, "vrooli-autoheal")
		sort.Strings(scenarioIDs)
	}

	var roots []string
	addDirIfExists := func(rel string) {
		if dirExists(filepath.Join(repoRoot, rel)) {
			roots = append(roots, rel)
		}
	}
	addFileIfExists := func(rel string) {
		if fileExists(filepath.Join(repoRoot, rel)) {
			roots = append(roots, rel)
		}
	}

	addDirIfExists(".vrooli")
	addDirIfExists("api")
	addDirIfExists("cli")
	addDirIfExists("src")
	addDirIfExists("scripts")
	addDirIfExists("platforms")
	addDirIfExists("assets")
	if manifest.Bundle.IncludePackages {
		addDirIfExists("packages")
	}

	for _, id := range scenarioIDs {
		addDirIfExists(filepath.Join("scenarios", id))
	}
	for _, id := range resourceIDs {
		addDirIfExists(filepath.Join("resources", id))
	}

	addFileIfExists("go.work")
	addFileIfExists("go.work.sum")
	addFileIfExists("package.json")
	addFileIfExists("pnpm-lock.yaml")
	addFileIfExists("pnpm-workspace.yaml")
	addFileIfExists(".npmrc")
	addFileIfExists(".env-example")
	addFileIfExists("Makefile")
	addFileIfExists("README.md")
	addFileIfExists("LICENSE")

	sort.Strings(roots)

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return MiniBundleSpec{}, err
	}

	return MiniBundleSpec{
		IncludeRoots: roots,
		Excludes: []string{
			".git/**",
			"**/.git/**",
			"node_modules/**",
			"**/node_modules/**",
			".pnpm-store/**",
			"**/.pnpm-store/**",
			"coverage/**",
			"**/coverage/**",
			"logs/**",
			"**/logs/**",
			"data/**",
			"**/data/**",
			"projects/**",
			"**/projects/**",
			"**/.DS_Store",
			"**/dist/**",
			"**/.next/**",
		},
		ExtraFiles: map[string][]byte{
			".vrooli/cloud/manifest.json": manifestBytes,
		},
	}, nil
}

func writeDeterministicTarGz(w io.Writer, repoRoot string, spec MiniBundleSpec) (int64, error) {
	var size int64

	gw, err := gzip.NewWriterLevel(countingWriter{w: w, n: &size}, gzip.BestCompression)
	if err != nil {
		return 0, err
	}
	gw.Header.ModTime = time.Unix(0, 0).UTC()
	gw.Header.OS = 255
	gw.Header.Name = "mini-vrooli.tar"

	tw := tar.NewWriter(gw)
	defer func() {
		_ = tw.Close()
		_ = gw.Close()
	}()

	paths, err := collectIncludedPaths(repoRoot, spec)
	if err != nil {
		return 0, err
	}

	for _, rel := range paths {
		abs := filepath.Join(repoRoot, rel)
		info, err := os.Lstat(abs)
		if err != nil {
			return 0, err
		}
		if info.IsDir() {
			continue
		}

		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			link, err = os.Readlink(abs)
			if err != nil {
				return 0, err
			}
		}

		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return 0, err
		}
		header.Name = toTarPath(rel)
		header.ModTime = time.Unix(0, 0).UTC()
		header.AccessTime = time.Unix(0, 0).UTC()
		header.ChangeTime = time.Unix(0, 0).UTC()
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""

		if err := tw.WriteHeader(header); err != nil {
			return 0, err
		}

		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			f, err := os.Open(abs)
			if err != nil {
				return 0, err
			}
			_, copyErr := io.Copy(tw, f)
			_ = f.Close()
			if copyErr != nil {
				return 0, copyErr
			}
		}
	}

	for rel, contents := range spec.ExtraFiles {
		rel = filepath.Clean(rel)
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return 0, fmt.Errorf("extra file path must be relative: %q", rel)
		}

		b := contents
		hdr := &tar.Header{
			Name:       toTarPath(rel),
			Size:       int64(len(b)),
			Mode:       0o644,
			Typeflag:   tar.TypeReg,
			ModTime:    time.Unix(0, 0).UTC(),
			AccessTime: time.Unix(0, 0).UTC(),
			ChangeTime: time.Unix(0, 0).UTC(),
			Uid:        0,
			Gid:        0,
			Uname:      "",
			Gname:      "",
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return 0, err
		}
		if _, err := io.Copy(tw, bytes.NewReader(b)); err != nil {
			return 0, err
		}
	}

	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gw.Close(); err != nil {
		return 0, err
	}
	return size, nil
}

func collectIncludedPaths(repoRoot string, spec MiniBundleSpec) ([]string, error) {
	var out []string
	seen := map[string]struct{}{}

	addPath := func(rel string) {
		rel = filepath.Clean(rel)
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) || rel == "." {
			return
		}
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}

	for _, relRoot := range spec.IncludeRoots {
		relRoot = filepath.Clean(relRoot)
		absRoot := filepath.Join(repoRoot, relRoot)

		info, err := os.Lstat(absRoot)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			if !isExcluded(relRoot, spec.Excludes) {
				addPath(relRoot)
			}
			continue
		}

		err = filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, err := filepath.Rel(repoRoot, p)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if rel == "." {
				return nil
			}
			if isExcluded(rel, spec.Excludes) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			addPath(rel)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(out)
	return out, nil
}

func isExcluded(path string, patterns []string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	for _, pat := range patterns {
		pat = filepath.ToSlash(filepath.Clean(pat))
		if strings.HasPrefix(pat, "**/") && strings.HasSuffix(pat, "/**") {
			segment := strings.TrimSuffix(strings.TrimPrefix(pat, "**/"), "/**")
			if segment != "" && containsPathSegment(p, segment) {
				return true
			}
			continue
		}
		if strings.HasSuffix(pat, "/**") && !strings.Contains(pat, "**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if p == prefix || strings.HasPrefix(p, prefix+"/") {
				return true
			}
			continue
		}
		ok, err := filepath.Match(pat, p)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func containsPathSegment(path, segment string) bool {
	path = "/" + strings.Trim(path, "/") + "/"
	segment = "/" + strings.Trim(segment, "/") + "/"
	return strings.Contains(path, segment)
}

func safeFilename(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "unknown"
	}
	var out []rune
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out = append(out, r)
			continue
		}
		out = append(out, '_')
	}
	return string(out)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func toTarPath(rel string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(rel)), "./")
}

type countingWriter struct {
	w io.Writer
	n *int64
}

func (cw countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	*cw.n += int64(n)
	return n, err
}
