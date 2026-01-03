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

type BundleInfo struct {
	Path       string `json:"path"`
	Filename   string `json:"filename"`
	ScenarioID string `json:"scenario_id"`
	Sha256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
	CreatedAt  string `json:"created_at"`
}

func ListBundles(bundlesDir string) ([]BundleInfo, error) {
	if !dirExists(bundlesDir) {
		return []BundleInfo{}, nil
	}

	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		return nil, fmt.Errorf("read bundles directory: %w", err)
	}

	var bundles []BundleInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "mini-vrooli_") || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		scenarioID, sha256Hash := parseBundleFilename(name)
		bundles = append(bundles, BundleInfo{
			Path:       filepath.Join(bundlesDir, name),
			Filename:   name,
			ScenarioID: scenarioID,
			Sha256:     sha256Hash,
			SizeBytes:  info.Size(),
			CreatedAt:  info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	// Sort by creation time, newest first
	sort.Slice(bundles, func(i, j int) bool {
		return bundles[i].CreatedAt > bundles[j].CreatedAt
	})

	return bundles, nil
}

func parseBundleFilename(name string) (scenarioID, sha256Hash string) {
	// Format: mini-vrooli_<scenario-id>_<sha256>.tar.gz
	name = strings.TrimPrefix(name, "mini-vrooli_")
	name = strings.TrimSuffix(name, ".tar.gz")

	// Find the last underscore followed by a 64-char hex string (SHA256)
	lastUnderscore := strings.LastIndex(name, "_")
	if lastUnderscore == -1 || len(name)-lastUnderscore-1 != 64 {
		return name, ""
	}

	return name[:lastUnderscore], name[lastUnderscore+1:]
}

// ScenarioStats holds per-scenario bundle statistics.
type ScenarioStats struct {
	Count     int   `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

// BundleStats holds aggregate bundle storage statistics.
type BundleStats struct {
	TotalCount      int                      `json:"total_count"`
	TotalSizeBytes  int64                    `json:"total_size_bytes"`
	OldestCreatedAt string                   `json:"oldest_created_at,omitempty"`
	NewestCreatedAt string                   `json:"newest_created_at,omitempty"`
	ByScenario      map[string]ScenarioStats `json:"by_scenario"`
}

// GetBundleStats returns aggregate storage statistics for all bundles.
func GetBundleStats(bundlesDir string) (BundleStats, error) {
	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return BundleStats{}, err
	}

	stats := BundleStats{
		ByScenario: make(map[string]ScenarioStats),
	}

	for _, b := range bundles {
		stats.TotalCount++
		stats.TotalSizeBytes += b.SizeBytes

		// Track per-scenario stats
		scenStats := stats.ByScenario[b.ScenarioID]
		scenStats.Count++
		scenStats.SizeBytes += b.SizeBytes
		stats.ByScenario[b.ScenarioID] = scenStats

		// Track oldest/newest (bundles are sorted newest first)
		if stats.NewestCreatedAt == "" {
			stats.NewestCreatedAt = b.CreatedAt
		}
		stats.OldestCreatedAt = b.CreatedAt
	}

	return stats, nil
}

// DeleteBundle removes a single bundle by SHA256 hash.
// Returns the size of the deleted file in bytes, or 0 if not found.
func DeleteBundle(bundlesDir, sha256Hash string) (int64, error) {
	if sha256Hash == "" {
		return 0, fmt.Errorf("sha256 hash is required")
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return 0, err
	}

	for _, b := range bundles {
		if b.Sha256 == sha256Hash {
			if err := os.Remove(b.Path); err != nil {
				if os.IsNotExist(err) {
					return 0, nil // Already deleted, idempotent
				}
				return 0, fmt.Errorf("delete bundle %s: %w", b.Filename, err)
			}
			return b.SizeBytes, nil
		}
	}

	return 0, nil // Not found, idempotent
}

// DeleteBundlesForScenario removes bundles for a specific scenario,
// keeping the N most recent bundles (by creation time).
// Returns the list of deleted bundles and total freed bytes.
func DeleteBundlesForScenario(bundlesDir, scenarioID string, keepLatest int) ([]BundleInfo, int64, error) {
	if scenarioID == "" {
		return nil, 0, fmt.Errorf("scenario ID is required")
	}
	if keepLatest < 0 {
		keepLatest = 0
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return nil, 0, err
	}

	// Filter bundles for this scenario (already sorted newest first)
	var scenarioBundles []BundleInfo
	for _, b := range bundles {
		if b.ScenarioID == scenarioID {
			scenarioBundles = append(scenarioBundles, b)
		}
	}

	// Keep the newest N bundles, delete the rest
	var deleted []BundleInfo
	var freedBytes int64

	for i, b := range scenarioBundles {
		if i < keepLatest {
			continue // Keep this one
		}
		if err := os.Remove(b.Path); err != nil {
			if os.IsNotExist(err) {
				continue // Already deleted
			}
			return deleted, freedBytes, fmt.Errorf("delete bundle %s: %w", b.Filename, err)
		}
		deleted = append(deleted, b)
		freedBytes += b.SizeBytes
	}

	return deleted, freedBytes, nil
}

// DeleteAllOldBundles removes bundles across all scenarios, keeping N newest per scenario.
// Returns the list of deleted bundles and total freed bytes.
func DeleteAllOldBundles(bundlesDir string, keepLatestPerScenario int) ([]BundleInfo, int64, error) {
	if keepLatestPerScenario < 0 {
		keepLatestPerScenario = 0
	}

	bundles, err := ListBundles(bundlesDir)
	if err != nil {
		return nil, 0, err
	}

	// Group bundles by scenario
	byScenario := make(map[string][]BundleInfo)
	for _, b := range bundles {
		byScenario[b.ScenarioID] = append(byScenario[b.ScenarioID], b)
	}

	var deleted []BundleInfo
	var freedBytes int64

	// For each scenario, keep N newest and delete the rest
	for _, scenarioBundles := range byScenario {
		for i, b := range scenarioBundles {
			if i < keepLatestPerScenario {
				continue // Keep this one
			}
			if err := os.Remove(b.Path); err != nil {
				if os.IsNotExist(err) {
					continue // Already deleted
				}
				// Log but continue with other deletions
				continue
			}
			deleted = append(deleted, b)
			freedBytes += b.SizeBytes
		}
	}

	return deleted, freedBytes, nil
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
		// Exclude dist folders EXCEPT packages/*/dist (pre-built shared libraries needed at runtime)
		"scenarios/**/dist/**",
		"resources/**/dist/**",
		"dist/**",
		"**/.next/**",
		// NEVER bundle mothership secrets - these are generated on the target VPS
		".vrooli/secrets.json",
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
		"manifest_path":  ".vrooli/cloud/manifest.json",
	}, "", "  ")
	if err != nil {
		return MiniBundleSpec{}, err
	}

	extra := map[string][]byte{
		".vrooli/cloud/manifest.json":        manifestBytes,
		".vrooli/cloud/bundle-metadata.json": metaBytes,
		"go.work":                            []byte(goWork),
	}
	if len(serviceJSON) > 0 {
		extra[".vrooli/service.json"] = serviceJSON
	}

	// Override the target scenario's service.json with fixed ports from the manifest
	// This ensures VPS deployments use the exact ports specified in the deployment manifest
	// rather than dynamically allocating from ranges
	scenarioServiceJSON, err := buildScenarioServiceJSONWithFixedPorts(repoRoot, manifest)
	if err != nil {
		return MiniBundleSpec{}, fmt.Errorf("build scenario service.json with fixed ports: %w", err)
	}
	if len(scenarioServiceJSON) > 0 {
		scenarioServicePath := filepath.Join("scenarios", manifest.Scenario.ID, ".vrooli", "service.json")
		extra[scenarioServicePath] = scenarioServiceJSON
	}

	return MiniBundleSpec{
		IncludeRoots: roots,
		Excludes:     excludes,
		ExtraFiles:   extra,
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

	// Build set of required resources from manifest
	requiredResources := map[string]struct{}{}
	for _, id := range stableUniqueStrings(manifest.Bundle.Resources) {
		requiredResources[id] = struct{}{}
	}

	// Build set of required scenarios from manifest
	requiredScenarios := map[string]struct{}{}
	for _, id := range stableUniqueStrings(manifest.Bundle.Scenarios) {
		requiredScenarios[id] = struct{}{}
	}
	// Also include the main scenario being deployed
	if manifest.Scenario.ID != "" {
		requiredScenarios[manifest.Scenario.ID] = struct{}{}
	}

	// Ensure dependencies section exists
	dependenciesAny, hasDeps := doc["dependencies"]
	var dependencies map[string]interface{}
	if hasDeps {
		dependencies, _ = dependenciesAny.(map[string]interface{})
	}
	if dependencies == nil {
		dependencies = make(map[string]interface{})
	}

	// Handle resources
	if len(requiredResources) > 0 {
		existingResources := make(map[string]interface{})
		if resourcesAny, hasResources := dependencies["resources"]; hasResources {
			if r, ok := resourcesAny.(map[string]interface{}); ok {
				existingResources = r
			}
		}

		newResources := make(map[string]interface{})
		for id := range requiredResources {
			if existing, ok := existingResources[id]; ok {
				// Keep existing config but ensure enabled=true
				if m, ok := existing.(map[string]interface{}); ok {
					m["enabled"] = true
					newResources[id] = m
				} else {
					newResources[id] = map[string]interface{}{"enabled": true}
				}
			} else {
				// Resource not in original config, add minimal entry
				newResources[id] = map[string]interface{}{"enabled": true}
			}
		}
		dependencies["resources"] = newResources
	}

	// Handle scenarios - ensure we have at least the deployed scenario
	if len(requiredScenarios) > 0 {
		existingScenarios := make(map[string]interface{})
		if scenariosAny, hasScenarios := dependencies["scenarios"]; hasScenarios {
			if s, ok := scenariosAny.(map[string]interface{}); ok {
				existingScenarios = s
			}
		}

		newScenarios := make(map[string]interface{})
		for id := range requiredScenarios {
			if existing, ok := existingScenarios[id]; ok {
				newScenarios[id] = existing
			} else {
				// Scenario not in original config, add minimal entry
				newScenarios[id] = map[string]interface{}{"enabled": true}
			}
		}
		dependencies["scenarios"] = newScenarios
	}

	doc["dependencies"] = dependencies

	// Also handle legacy top-level "resources" key if present
	if resourcesAny, ok := doc["resources"]; ok {
		if resources, ok := resourcesAny.(map[string]interface{}); ok {
			newResources := make(map[string]interface{})
			for id := range requiredResources {
				if existing, ok := resources[id]; ok {
					if m, ok := existing.(map[string]interface{}); ok {
						m["enabled"] = true
						newResources[id] = m
					} else {
						newResources[id] = map[string]interface{}{"enabled": true}
					}
				} else {
					newResources[id] = map[string]interface{}{"enabled": true}
				}
			}
			doc["resources"] = newResources
		}
	}

	return json.MarshalIndent(doc, "", "  ")
}

// buildScenarioServiceJSONWithFixedPorts reads the target scenario's service.json
// and replaces port ranges with fixed port values from the deployment manifest.
// This ensures VPS deployments use exact ports rather than dynamic allocation.
func buildScenarioServiceJSONWithFixedPorts(repoRoot string, manifest CloudManifest) ([]byte, error) {
	if manifest.Scenario.ID == "" {
		return nil, nil
	}

	// Read the scenario's service.json
	path := filepath.Join(repoRoot, "scenarios", manifest.Scenario.ID, ".vrooli", "service.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No service.json, nothing to modify
		}
		return nil, err
	}

	// Parse the service.json
	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse scenario service.json: %w", err)
	}

	// Get the ports section
	portsAny, hasPorts := doc["ports"]
	if !hasPorts {
		// No ports section, return as-is
		return json.MarshalIndent(doc, "", "  ")
	}

	ports, ok := portsAny.(map[string]interface{})
	if !ok {
		return json.MarshalIndent(doc, "", "  ")
	}

	// For each port in manifest.Ports, replace the range with a fixed port
	for portName, portValue := range manifest.Ports {
		portConfig, exists := ports[portName]
		if !exists {
			// Port doesn't exist in service.json, add it
			ports[portName] = map[string]interface{}{
				"port":    portValue,
				"env_var": strings.ToUpper(portName) + "_PORT",
			}
			continue
		}

		// Port exists, modify it
		configMap, ok := portConfig.(map[string]interface{})
		if !ok {
			// Not a map, replace with fixed port
			ports[portName] = map[string]interface{}{
				"port":    portValue,
				"env_var": strings.ToUpper(portName) + "_PORT",
			}
			continue
		}

		// Remove the range field and add fixed port
		delete(configMap, "range")
		configMap["port"] = portValue

		// Ensure env_var is set
		if _, hasEnvVar := configMap["env_var"]; !hasEnvVar {
			configMap["env_var"] = strings.ToUpper(portName) + "_PORT"
		}
	}

	doc["ports"] = ports
	return json.MarshalIndent(doc, "", "  ")
}

func isExcluded(path string, patterns []string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	for _, pat := range patterns {
		pat = filepath.ToSlash(filepath.Clean(pat))

		// Case 1: **/segment/** - match if segment appears anywhere as a path component
		if strings.HasPrefix(pat, "**/") && strings.HasSuffix(pat, "/**") {
			segment := strings.TrimSuffix(strings.TrimPrefix(pat, "**/"), "/**")
			if segment != "" && containsPathSegment(p, segment) {
				return true
			}
			continue
		}

		// Case 2: **/filename - match filename anywhere in path
		// Example: **/.DS_Store matches any/.DS_Store or .DS_Store
		if strings.HasPrefix(pat, "**/") && !strings.HasSuffix(pat, "/**") {
			suffix := strings.TrimPrefix(pat, "**/")
			if p == suffix || strings.HasSuffix(p, "/"+suffix) {
				return true
			}
			continue
		}

		// Case 3: prefix/** (no ** in middle) - match prefix exactly
		if strings.HasSuffix(pat, "/**") && !strings.Contains(pat[:len(pat)-3], "**") {
			prefix := strings.TrimSuffix(pat, "/**")
			if p == prefix || strings.HasPrefix(p, prefix+"/") {
				return true
			}
			continue
		}

		// Case 4: prefix/**/segment/** - match prefix, then segment anywhere after
		// Example: scenarios/**/dist/** matches scenarios/foo/bar/dist/file.js
		if strings.Contains(pat, "/**/") {
			parts := strings.Split(pat, "/**/")
			if len(parts) >= 2 {
				prefix := parts[0]
				// Path must start with prefix
				if !strings.HasPrefix(p, prefix+"/") && p != prefix {
					continue
				}

				// Check remaining parts - each must appear as a path segment after the previous
				remainder := strings.TrimPrefix(p, prefix+"/")
				matched := true
				for i := 1; i < len(parts); i++ {
					segment := parts[i]
					// Handle trailing /** on last segment
					if strings.HasSuffix(segment, "/**") {
						segment = strings.TrimSuffix(segment, "/**")
					}
					if segment == "" {
						continue
					}
					// Check if segment appears as a path component in remainder
					if !containsPathSegment(remainder, segment) {
						matched = false
						break
					}
					// Move remainder past this segment for next iteration
					idx := strings.Index("/"+remainder+"/", "/"+segment+"/")
					if idx >= 0 {
						afterSegment := idx + len(segment) + 1
						if afterSegment < len(remainder)+1 {
							remainder = remainder[afterSegment:]
						} else {
							remainder = ""
						}
					}
				}
				if matched {
					return true
				}
			}
			continue
		}

		// Case 5: Simple glob patterns (no **) - use filepath.Match
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
