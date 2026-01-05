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

	"scenario-to-cloud/domain"
)

// Type aliases for backward compatibility and shorter references within main package.
type (
	BundleArtifact = domain.BundleArtifact
	BundleInfo     = domain.BundleInfo
	ScenarioStats  = domain.ScenarioStats
	BundleStats    = domain.BundleStats
)

// GetLocalBundlesDir returns the path to the local bundles directory.
// This consolidates the repeated pattern of finding repo root and appending the bundles path.
func GetLocalBundlesDir() (string, error) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, "scenarios", "scenario-to-cloud", "coverage", "bundles"), nil
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
		// Exclude scenario templates - they have placeholder go.mod files that break go.work
		"scripts/scenarios/templates/**",
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
	requiredResources := toStringSet(stableUniqueStrings(manifest.Bundle.Resources))

	// Build set of required scenarios from manifest (including main scenario)
	requiredScenarios := toStringSet(stableUniqueStrings(manifest.Bundle.Scenarios))
	if manifest.Scenario.ID != "" {
		requiredScenarios[manifest.Scenario.ID] = struct{}{}
	}

	// Ensure dependencies section exists
	dependencies := getOrCreateMapField(doc, "dependencies")

	// Handle resources and scenarios in dependencies section
	if len(requiredResources) > 0 {
		dependencies["resources"] = mergeRequiredEntries(dependencies, "resources", requiredResources, true)
	}
	if len(requiredScenarios) > 0 {
		dependencies["scenarios"] = mergeRequiredEntries(dependencies, "scenarios", requiredScenarios, false)
	}

	doc["dependencies"] = dependencies

	// Also handle legacy top-level "resources" key if present
	if _, hasLegacy := doc["resources"]; hasLegacy && len(requiredResources) > 0 {
		doc["resources"] = mergeRequiredEntries(doc, "resources", requiredResources, true)
	}

	return json.MarshalIndent(doc, "", "  ")
}

// toStringSet converts a slice to a map[string]struct{} for efficient lookup.
func toStringSet(slice []string) map[string]struct{} {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}
	return set
}

// getOrCreateMapField returns a map field from doc, creating it if needed.
func getOrCreateMapField(doc map[string]interface{}, key string) map[string]interface{} {
	if v, ok := doc[key]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			return m
		}
	}
	return make(map[string]interface{})
}

// mergeRequiredEntries merges required IDs with existing config, returning a new map.
// If setEnabled is true, entries in requiredIDs get "enabled": true, and entries in
// existing but not in requiredIDs get "enabled": false.
func mergeRequiredEntries(parent map[string]interface{}, key string, requiredIDs map[string]struct{}, setEnabled bool) map[string]interface{} {
	existing := getOrCreateMapField(parent, key)
	merged := make(map[string]interface{}, len(existing)+len(requiredIDs))

	// First, process all existing entries - set enabled: false for non-required ones
	for id, entry := range existing {
		_, isRequired := requiredIDs[id]
		if m, ok := entry.(map[string]interface{}); ok && setEnabled {
			// Clone the map to avoid mutating the original
			cloned := make(map[string]interface{}, len(m))
			for k, v := range m {
				cloned[k] = v
			}
			cloned["enabled"] = isRequired
			merged[id] = cloned
		} else {
			merged[id] = entry
		}
	}

	// Then add any required entries that weren't in existing
	for id := range requiredIDs {
		if _, ok := existing[id]; !ok {
			merged[id] = map[string]interface{}{"enabled": true}
		}
	}

	return merged
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

	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse scenario service.json: %w", err)
	}

	// Get the ports section (return as-is if missing or invalid type)
	ports := getOrCreateMapField(doc, "ports")
	if len(ports) == 0 && doc["ports"] == nil {
		return json.MarshalIndent(doc, "", "  ")
	}

	// For each port in manifest.Ports, replace the range with a fixed port
	for portName, portValue := range manifest.Ports {
		ports[portName] = setFixedPort(ports[portName], portName, portValue)
	}

	doc["ports"] = ports
	return json.MarshalIndent(doc, "", "  ")
}

// setFixedPort updates or creates a port config with a fixed port value.
// Preserves existing config fields except "range", ensuring "port" and "env_var" are set.
func setFixedPort(existing interface{}, portName string, portValue int) map[string]interface{} {
	envVar := strings.ToUpper(portName) + "_PORT"

	// If existing is a map, preserve other fields
	if configMap, ok := existing.(map[string]interface{}); ok {
		delete(configMap, "range")
		configMap["port"] = portValue
		if _, hasEnvVar := configMap["env_var"]; !hasEnvVar {
			configMap["env_var"] = envVar
		}
		return configMap
	}

	// Create new config
	return map[string]interface{}{
		"port":    portValue,
		"env_var": envVar,
	}
}

// isExcluded checks if a path matches any exclusion pattern.
// Supports five glob pattern styles (see individual matchers for details).
func isExcluded(path string, patterns []string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	for _, pat := range patterns {
		pat = filepath.ToSlash(filepath.Clean(pat))
		if matchesExclusionPattern(p, pat) {
			return true
		}
	}
	return false
}

// matchesExclusionPattern checks if path matches a single exclusion pattern.
// Pattern types (in order of precedence):
//   - **/segment/**  : segment appears anywhere as path component
//   - **/filename    : filename at any depth
//   - prefix/**      : prefix directory and all contents
//   - prefix/**/seg  : prefix + segment anywhere after
//   - simple glob    : standard glob (no **)
func matchesExclusionPattern(path, pattern string) bool {
	switch {
	case matchesAnywhereSegment(path, pattern):
		return true
	case matchesAnywhereFile(path, pattern):
		return true
	case matchesPrefixGlob(path, pattern):
		return true
	case matchesNestedSegmentGlob(path, pattern):
		return true
	default:
		return matchesSimpleGlob(path, pattern)
	}
}

// matchesAnywhereSegment handles **/segment/** patterns.
// Returns true if segment appears anywhere as a complete path component.
// Example: **/node_modules/** matches "foo/node_modules/bar"
func matchesAnywhereSegment(path, pattern string) bool {
	if !strings.HasPrefix(pattern, "**/") || !strings.HasSuffix(pattern, "/**") {
		return false
	}
	segment := strings.TrimSuffix(strings.TrimPrefix(pattern, "**/"), "/**")
	return segment != "" && containsPathSegment(path, segment)
}

// matchesAnywhereFile handles **/filename patterns (no trailing /**).
// Returns true if filename appears at any depth.
// Example: **/.DS_Store matches "any/path/.DS_Store"
func matchesAnywhereFile(path, pattern string) bool {
	if !strings.HasPrefix(pattern, "**/") || strings.HasSuffix(pattern, "/**") {
		return false
	}
	suffix := strings.TrimPrefix(pattern, "**/")
	return path == suffix || strings.HasSuffix(path, "/"+suffix)
}

// matchesPrefixGlob handles prefix/** patterns (no ** in middle).
// Returns true if path starts with prefix or is exactly prefix.
// Example: coverage/** matches "coverage/reports/test.xml"
func matchesPrefixGlob(path, pattern string) bool {
	if !strings.HasSuffix(pattern, "/**") {
		return false
	}
	prefixPart := pattern[:len(pattern)-3]
	if strings.Contains(prefixPart, "**") {
		return false // Has ** in middle - not a simple prefix glob
	}
	prefix := strings.TrimSuffix(pattern, "/**")
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// matchesNestedSegmentGlob handles prefix/**/segment/** patterns.
// Returns true if path starts with prefix and contains segment(s) anywhere after.
// Example: scenarios/**/dist/** matches "scenarios/foo/bar/dist/file.js"
func matchesNestedSegmentGlob(path, pattern string) bool {
	if !strings.Contains(pattern, "/**/") {
		return false
	}

	parts := strings.Split(pattern, "/**/")
	if len(parts) < 2 {
		return false
	}

	prefix := parts[0]
	if !strings.HasPrefix(path, prefix+"/") && path != prefix {
		return false
	}

	remainder := strings.TrimPrefix(path, prefix+"/")
	for i := 1; i < len(parts); i++ {
		segment := strings.TrimSuffix(parts[i], "/**")
		if segment == "" {
			continue
		}
		if !containsPathSegment(remainder, segment) {
			return false
		}
		// Advance remainder past this segment
		remainder = advancePastSegment(remainder, segment)
	}
	return true
}

// advancePastSegment returns the portion of path after the first occurrence of segment.
func advancePastSegment(path, segment string) string {
	idx := strings.Index("/"+path+"/", "/"+segment+"/")
	if idx < 0 {
		return path
	}
	afterSegment := idx + len(segment) + 1
	if afterSegment < len(path)+1 {
		return path[afterSegment:]
	}
	return ""
}

// matchesSimpleGlob handles patterns without ** using filepath.Match.
// Example: *.log matches "error.log"
func matchesSimpleGlob(path, pattern string) bool {
	ok, err := filepath.Match(pattern, path)
	return err == nil && ok
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
