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
		// Repo root detection must not depend on a committed `go.work`.
		// Some deployments intentionally omit `go.work` to avoid workspace-mode coupling.
		if dirExists(filepath.Join(dir, ".vrooli")) && dirExists(filepath.Join(dir, "scenarios")) && dirExists(filepath.Join(dir, "resources")) {
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

	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		return BundleArtifact{}, err
	}

	tmp, err := os.CreateTemp(outDir, "mini-vrooli_*.tar.gz")
	if err != nil {
		return BundleArtifact{}, err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(tmp, hasher)

	size, err := writeDeterministicTarGz(mw, repoRoot, spec)
	if err != nil {
		return BundleArtifact{}, err
	}
	if err := tmp.Sync(); err != nil {
		return BundleArtifact{}, err
	}
	if err := tmp.Close(); err != nil {
		return BundleArtifact{}, err
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	outName := fmt.Sprintf("mini-vrooli_%s_%s.tar.gz", safeFilename(manifest.Scenario.ID), sum)
	outPath := filepath.Join(outDir, outName)

	_ = os.Remove(outPath)
	if err := os.Rename(tmpPath, outPath); err != nil {
		return BundleArtifact{}, err
	}

	return BundleArtifact{
		Path:      outPath,
		Sha256:    sum,
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

	excludes := []string{
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
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return MiniBundleSpec{}, err
	}

	goWork, err := buildMiniGoWork(repoRoot, roots, excludes)
	if err != nil {
		return MiniBundleSpec{}, err
	}

	serviceJSON, err := buildMiniServiceJSON(repoRoot, manifest)
	if err != nil {
		return MiniBundleSpec{}, err
	}

	metaBytes, err := json.MarshalIndent(map[string]interface{}{
		"schema_version": "1.0.0",
		"scenario_id":    manifest.Scenario.ID,
		"include_roots":  roots,
		"excludes":       excludes,
		"manifest_path": ".vrooli/cloud/manifest.json",
	}, "", "  ")
	if err != nil {
		return MiniBundleSpec{}, err
	}

	extra := map[string][]byte{
		".vrooli/cloud/manifest.json":        manifestBytes,
		".vrooli/cloud/bundle-metadata.json": metaBytes,
		"go.work":                           []byte(goWork),
	}
	if len(serviceJSON) > 0 {
		extra[".vrooli/service.json"] = serviceJSON
	}

	return MiniBundleSpec{
		IncludeRoots: roots,
		Excludes:     excludes,
		ExtraFiles: extra,
	}, nil
}

func buildMiniGoWork(repoRoot string, includeRoots []string, excludes []string) (string, error) {
	version := "1.24.0"
	if b, err := os.ReadFile(filepath.Join(repoRoot, "go.work")); err == nil {
		if v := parseGoWorkVersion(string(b)); v != "" {
			version = v
		}
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	moduleDirs, err := discoverGoModDirs(repoRoot, includeRoots, excludes)
	if err != nil {
		return "", err
	}

	var out strings.Builder
	out.WriteString("go " + version + "\n\n")
	out.WriteString("use (\n")
	for _, dir := range moduleDirs {
		out.WriteString("\t./" + dir + "\n")
	}
	out.WriteString(")\n")
	return out.String(), nil
}

func parseGoWorkVersion(contents string) string {
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return ""
}

func discoverGoModDirs(repoRoot string, includeRoots []string, excludes []string) ([]string, error) {
	found := map[string]struct{}{}
	add := func(rel string) {
		rel = filepath.ToSlash(filepath.Clean(rel))
		rel = strings.TrimPrefix(rel, "./")
		if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
			return
		}
		found[rel] = struct{}{}
	}

	for _, root := range includeRoots {
		root = filepath.Clean(root)
		absRoot := filepath.Join(repoRoot, root)
		info, err := os.Lstat(absRoot)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
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
			if isExcluded(rel, excludes) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() != "go.mod" {
				return nil
			}
			add(filepath.Dir(rel))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(found))
	for rel := range found {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
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

	extraKeys := make([]string, 0, len(spec.ExtraFiles))
	for rel := range spec.ExtraFiles {
		extraKeys = append(extraKeys, rel)
	}
	sort.Strings(extraKeys)

	for _, rel := range extraKeys {
		contents := spec.ExtraFiles[rel]
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

	extraFiles := map[string]struct{}{}
	for rel := range spec.ExtraFiles {
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		extraFiles[rel] = struct{}{}
	}

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
				if _, ok := extraFiles[filepath.ToSlash(relRoot)]; ok {
					continue
				}
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
			if _, ok := extraFiles[rel]; ok {
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

func buildMiniServiceJSON(repoRoot string, manifest CloudManifest) ([]byte, error) {
	// Best-effort: if the repo doesn't have a root .vrooli/service.json, don't synthesize one.
	path := filepath.Join(repoRoot, ".vrooli", "service.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse .vrooli/service.json: %w", err)
	}

	resourcesAny, ok := doc["resources"]
	if !ok {
		return json.MarshalIndent(doc, "", "  ")
	}
	resources, ok := resourcesAny.(map[string]interface{})
	if !ok {
		return json.MarshalIndent(doc, "", "  ")
	}

	required := map[string]struct{}{}
	for _, id := range stableUniqueStrings(manifest.Bundle.Resources) {
		required[id] = struct{}{}
	}

	for key, val := range resources {
		m, ok := val.(map[string]interface{})
		if !ok {
			continue
		}
		_, keep := required[key]
		m["enabled"] = keep
		resources[key] = m
	}
	for id := range required {
		if _, ok := resources[id]; ok {
			continue
		}
		resources[id] = map[string]interface{}{"enabled": true}
	}
	doc["resources"] = resources

	return json.MarshalIndent(doc, "", "  ")
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
