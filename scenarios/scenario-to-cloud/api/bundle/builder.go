package bundle

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
	"scenario-to-cloud/internal/stringutil"

	repocontract "github.com/vrooli/repo-contract-go"
)

// MiniBundleSpec describes what to include in a mini-Vrooli bundle.
type MiniBundleSpec struct {
	IncludeRoots []string
	Excludes     []string
	ExtraFiles   map[string][]byte // relative path -> contents
}

// BuildMiniVrooliBundle creates a deterministic tar.gz bundle from a manifest.
func BuildMiniVrooliBundle(repoRoot, outDir string, manifest domain.CloudManifest) (domain.BundleArtifact, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return domain.BundleArtifact{}, err
	}

	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		return domain.BundleArtifact{}, err
	}

	tmp, err := os.CreateTemp(outDir, "mini-vrooli_*.tar.gz")
	if err != nil {
		return domain.BundleArtifact{}, err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	hasher := sha256.New()
	mw := io.MultiWriter(tmp, hasher)

	size, err := WriteDeterministicTarGz(mw, repoRoot, spec)
	if err != nil {
		return domain.BundleArtifact{}, err
	}
	if err := tmp.Sync(); err != nil {
		return domain.BundleArtifact{}, err
	}
	if err := tmp.Close(); err != nil {
		return domain.BundleArtifact{}, err
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	outName := fmt.Sprintf("mini-vrooli_%s_%s.tar.gz", SafeFilename(manifest.Scenario.ID), sum)
	outPath := filepath.Join(outDir, outName)

	_ = os.Remove(outPath)
	if err := os.Rename(tmpPath, outPath); err != nil {
		return domain.BundleArtifact{}, err
	}

	return domain.BundleArtifact{
		Path:      outPath,
		Sha256:    sum,
		SizeBytes: size,
	}, nil
}

// CalculateBundleSHA computes the deterministic bundle SHA256 and size without writing a bundle file.
// This is useful for freshness checks where we only need a fingerprint comparison.
func CalculateBundleSHA(repoRoot string, manifest domain.CloudManifest) (string, int64, error) {
	spec, err := MiniVrooliBundleSpec(repoRoot, manifest)
	if err != nil {
		return "", 0, err
	}

	hasher := sha256.New()
	size, err := WriteDeterministicTarGz(io.MultiWriter(io.Discard, hasher), repoRoot, spec)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}

// MiniVrooliBundleSpec builds the specification for a mini-Vrooli bundle using
// the contract-defined mini_vrooli_bundle profile plus manifest-specific
// augmentations kept in code.
func MiniVrooliBundleSpec(repoRoot string, manifest domain.CloudManifest) (MiniBundleSpec, error) {
	scenarioIDs := selectedMiniBundleScenarios(manifest)
	contract, profile, err := resolveMiniBundleProfile(repoRoot, manifest)
	if err != nil {
		return MiniBundleSpec{}, err
	}
	roots, err := miniBundleRoots(repoRoot, contract, profile, scenarioIDs)
	if err != nil {
		return MiniBundleSpec{}, err
	}
	excludes := miniBundleExcludes(manifest, scenarioIDs, profile.Exclude)
	extra, err := miniBundleExtraFiles(repoRoot, manifest, scenarioIDs, roots, excludes)
	if err != nil {
		return MiniBundleSpec{}, err
	}
	return MiniBundleSpec{IncludeRoots: roots, Excludes: excludes, ExtraFiles: extra}, nil
}

func selectedMiniBundleScenarios(manifest domain.CloudManifest) []string {
	scenarioIDs := stringutil.SortedUnique(manifest.Bundle.Scenarios)
	if manifest.Bundle.IncludeAutoheal && !stringutil.Contains(scenarioIDs, "vrooli-autoheal") {
		scenarioIDs = append(scenarioIDs, "vrooli-autoheal")
		sort.Strings(scenarioIDs)
	}
	return scenarioIDs
}

func resolveMiniBundleProfile(repoRoot string, manifest domain.CloudManifest) (*repocontract.Contract, repocontract.ResolvedProfile, error) {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return nil, repocontract.ResolvedProfile{}, fmt.Errorf("load repo contract: %w", err)
	}
	profile, err := contract.ResolveProfile("mini_vrooli_bundle", repocontract.ResolveParams{
		Values: map[string]string{"scenario": manifest.Scenario.ID},
		Lists:  map[string][]string{"resources": stringutil.SortedUnique(manifest.Bundle.Resources)},
	})
	if err != nil {
		return nil, repocontract.ResolvedProfile{}, fmt.Errorf("resolve mini bundle profile: %w", err)
	}
	return contract, profile, nil
}

func miniBundleRoots(repoRoot string, contract *repocontract.Contract, profile repocontract.ResolvedProfile, scenarioIDs []string) ([]string, error) {
	roots := existingProfileRoots(repoRoot, profile.Include, profile.OptionalInclude)
	for _, scenarioID := range scenarioIDs {
		if err := appendSelectedScenarioRoot(repoRoot, contract, scenarioID, &roots); err != nil {
			return nil, err
		}
	}
	sort.Strings(roots)
	return roots, nil
}

func appendSelectedScenarioRoot(repoRoot string, contract *repocontract.Contract, scenarioID string, roots *[]string) error {
	scenarioRoot, err := contract.ScenarioRoot(repoRoot, scenarioID)
	if err != nil {
		return fmt.Errorf("resolve selected scenario root %s: %w", scenarioID, err)
	}
	rel, err := filepath.Rel(repoRoot, scenarioRoot)
	if err != nil {
		return fmt.Errorf("make selected scenario root relative %s: %w", scenarioID, err)
	}
	rel = filepath.ToSlash(rel)
	if dirExists(filepath.Join(repoRoot, filepath.FromSlash(rel))) && !stringutil.Contains(*roots, rel) {
		*roots = append(*roots, rel)
	}
	return nil
}

func miniBundleExcludes(manifest domain.CloudManifest, scenarioIDs, profileExcludes []string) []string {
	excludes := filterProfileExcludes(profileExcludes)
	for _, scenarioID := range scenarioIDs {
		if scenarioID != manifest.Scenario.ID && !stringutil.Contains(manifest.Bundle.UIScenarios, scenarioID) {
			excludes = append(excludes, "scenarios/"+scenarioID+"/ui/**")
		}
	}
	if manifest.Bundle.IncludeAutoheal {
		excludes = withoutCLIExclude(excludes)
	}
	return excludes
}

func filterProfileExcludes(profileExcludes []string) []string {
	excludes := make([]string, 0, len(profileExcludes))
	for _, pattern := range profileExcludes {
		// Resource owners need cli/** directories; remove generic UI exclusions
		// so the selected scenario can keep its primary user experience.
		if pattern == "cli/**" || strings.Contains(pattern, "ui/**") {
			continue
		}
		excludes = append(excludes, pattern)
	}
	return excludes
}

func withoutCLIExclude(excludes []string) []string {
	filtered := make([]string, 0, len(excludes))
	for _, pattern := range excludes {
		if pattern != "cli/**" {
			filtered = append(filtered, pattern)
		}
	}
	return filtered
}

func miniBundleExtraFiles(repoRoot string, manifest domain.CloudManifest, scenarioIDs, roots, excludes []string) (map[string][]byte, error) {
	extra, err := miniBundleBaseFiles(repoRoot, manifest, roots, excludes)
	if err != nil {
		return nil, err
	}
	if err := addTargetScenarioServiceFile(repoRoot, manifest, extra); err != nil {
		return nil, err
	}
	if err := addScenarioPlansFile(repoRoot, manifest, extra); err != nil {
		return nil, err
	}
	if err := addDependencyServiceFiles(repoRoot, manifest, scenarioIDs, extra); err != nil {
		return nil, err
	}
	return extra, nil
}

func miniBundleBaseFiles(repoRoot string, manifest domain.CloudManifest, roots, excludes []string) (map[string][]byte, error) {
	manifestForBundle := manifest
	// Secrets are provisioned during execution; omitting them keeps bundle
	// fingerprinting reproducible for stored deployment manifests.
	manifestForBundle.Secrets = nil
	manifestBytes, err := json.MarshalIndent(manifestForBundle, "", "  ")
	if err != nil {
		return nil, err
	}
	goWork, err := buildMiniGoWork(repoRoot, roots, excludes)
	if err != nil {
		return nil, err
	}
	goWorkSum, err := buildMiniGoWorkSum(repoRoot, roots, excludes)
	if err != nil {
		return nil, err
	}
	serviceJSON, err := buildMiniServiceJSON(repoRoot, manifest)
	if err != nil {
		return nil, err
	}
	metaBytes, err := json.MarshalIndent(map[string]interface{}{
		"schema_version": "1.0.0",
		"scenario_id":    manifest.Scenario.ID,
		"include_roots":  roots,
		"excludes":       excludes,
		"manifest_path":  ".vrooli/cloud/manifest.json",
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	extra := map[string][]byte{
		".vrooli/cloud/manifest.json":        manifestBytes,
		".vrooli/cloud/bundle-metadata.json": metaBytes,
		// The contract requires the templates root marker, but mini bundles
		// intentionally omit its developer template payload.
		"templates/.keep": {},
		"go.work":         []byte(goWork),
		"go.work.sum":     []byte(goWorkSum),
	}
	if len(serviceJSON) > 0 {
		extra[".vrooli/service.json"] = serviceJSON
	}
	return extra, nil
}

func addTargetScenarioServiceFile(repoRoot string, manifest domain.CloudManifest, extra map[string][]byte) error {
	// Fixed ports from the deployment manifest must override dynamically
	// allocated ports in the scenario's service.json.
	serviceJSON, err := buildScenarioServiceJSONWithFixedPorts(repoRoot, manifest)
	if err != nil {
		return fmt.Errorf("build scenario service.json with fixed ports: %w", err)
	}
	if len(serviceJSON) == 0 {
		return nil
	}
	path, err := ResolveScenarioFileRelative(repoRoot, manifest.Scenario.ID, "service")
	if err != nil {
		return fmt.Errorf("resolve scenario service.json path: %w", err)
	}
	extra[path] = serviceJSON
	return nil
}

func addScenarioPlansFile(repoRoot string, manifest domain.CloudManifest, extra map[string][]byte) error {
	// Include the plans catalog at its canonical and runtime lookup paths.
	plansPath := filepath.Join(repoRoot, "scenarios", manifest.Scenario.ID, ".vrooli", "plans.json")
	plansJSON, err := os.ReadFile(plansPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read scenario plans.json: %w", err)
	}
	canonicalPath := filepath.ToSlash(filepath.Join("scenarios", manifest.Scenario.ID, ".vrooli", "plans.json"))
	extra[canonicalPath] = plansJSON
	extra[".vrooli/plans.json"] = plansJSON
	return nil
}

func addDependencyServiceFiles(repoRoot string, manifest domain.CloudManifest, scenarioIDs []string, extra map[string][]byte) error {
	for _, scenarioID := range scenarioIDs {
		if scenarioID == manifest.Scenario.ID || stringutil.Contains(manifest.Bundle.UIScenarios, scenarioID) {
			continue
		}
		if err := addDependencyServiceFile(repoRoot, scenarioID, extra); err != nil {
			return err
		}
	}
	return nil
}

func addDependencyServiceFile(repoRoot, scenarioID string, extra map[string][]byte) error {
	path, err := ResolveScenarioFileRelative(repoRoot, scenarioID, "service")
	if err != nil {
		return fmt.Errorf("resolve dependency service.json for %s: %w", scenarioID, err)
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
	if err != nil {
		return fmt.Errorf("read dependency service.json for %s: %w", scenarioID, err)
	}
	filtered, err := withoutUIComponent(data)
	if err != nil {
		return fmt.Errorf("filter dependency UI for %s: %w", scenarioID, err)
	}
	extra[path] = filtered
	return nil
}

func withoutUIComponent(data []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if components, ok := doc["components"].(map[string]any); ok {
		delete(components, "ui")
	}
	if deployment, ok := doc["deployment"].(map[string]any); ok {
		if listeners, ok := deployment["listeners"].([]any); ok {
			filtered := listeners[:0]
			for _, item := range listeners {
				listener, ok := item.(map[string]any)
				if ok && (listener["id"] == "ui" || listener["port"] == "ui") {
					continue
				}
				filtered = append(filtered, item)
			}
			deployment["listeners"] = filtered
		}
	}
	return json.MarshalIndent(doc, "", "  ")
}

func existingProfileRoots(repoRoot string, required, optional []string) []string {
	var roots []string
	addIfExists := func(rel string) {
		rel = filepath.ToSlash(filepath.Clean(rel))
		rel = strings.TrimPrefix(rel, "./")
		if rel == "." || rel == "" {
			return
		}
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if dirExists(fullPath) || fileExists(fullPath) {
			roots = append(roots, rel)
		}
	}
	for _, rel := range required {
		addIfExists(rel)
	}
	for _, rel := range optional {
		addIfExists(rel)
	}
	return roots
}

func buildMiniGoWork(repoRoot string, includeRoots, excludes []string) (string, error) {
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
	for _, dir := range moduleDirs {
		if candidate, ok := goDirective(filepath.Join(repoRoot, filepath.FromSlash(dir), "go.mod")); ok && newerGoVersion(candidate, version) {
			version = candidate
		}
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

// buildMiniGoWorkSum merges the checksums owned by the modules in the mini
// workspace. A target build runs with the generated go.work file active, so
// its workspace checksum file must cover dependencies that are intentionally
// absent from the repository root module's go.sum.
func buildMiniGoWorkSum(repoRoot string, includeRoots, excludes []string) (string, error) {
	moduleDirs, err := discoverGoModDirs(repoRoot, includeRoots, excludes)
	if err != nil {
		return "", err
	}
	lines := map[string]struct{}{}
	for _, dir := range moduleDirs {
		path := filepath.Join(repoRoot, filepath.FromSlash(dir), "go.sum")
		contents, readErr := os.ReadFile(path)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return "", readErr
		}
		for _, line := range strings.Split(string(contents), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				lines[line] = struct{}{}
			}
		}
	}
	merged := make([]string, 0, len(lines))
	for line := range lines {
		merged = append(merged, line)
	}
	sort.Strings(merged)
	if len(merged) == 0 {
		return "", nil
	}
	return strings.Join(merged, "\n") + "\n", nil
}

func goDirective(path string) (string, bool) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "go "))
			if value != "" {
				return value, true
			}
		}
	}
	return "", false
}

func newerGoVersion(candidate, current string) bool {
	cMajor, cMinor, cPatch, cOK := parseGoVersion(candidate)
	vMajor, vMinor, vPatch, vOK := parseGoVersion(current)
	if !cOK || !vOK {
		return false
	}
	return compareGoVersion(cMajor, cMinor, cPatch, vMajor, vMinor, vPatch) > 0
}

func parseGoVersion(value string) (major, minor, patch int, ok bool) {
	parts := strings.Split(strings.TrimSpace(value), ".")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return 0, 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return 0, 0, 0, false
	}
	if len(parts) == 3 {
		if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
			return 0, 0, 0, false
		}
	}
	return major, minor, patch, true
}

func compareGoVersion(leftMajor, leftMinor, leftPatch, rightMajor, rightMinor, rightPatch int) int {
	left := []int{leftMajor, leftMinor, leftPatch}
	right := []int{rightMajor, rightMinor, rightPatch}
	for i := range left {
		if left[i] > right[i] {
			return 1
		}
		if left[i] < right[i] {
			return -1
		}
	}
	return 0
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

func discoverGoModDirs(repoRoot string, includeRoots, excludes []string) ([]string, error) {
	found := map[string]struct{}{}
	// The repository root is a module too. It owns root-level maintenance
	// commands such as cmd/vrooli-watchdog; omitting it from the generated
	// workspace makes those commands look like packages outside the workspace
	// when setup rebuilds them on a target.
	if info, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil && !info.IsDir() {
		addGoModDir(found, ".")
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	for _, root := range includeRoots {
		modules, err := discoverGoModDirsUnderRoot(repoRoot, root, excludes)
		if err != nil {
			return nil, err
		}
		for _, module := range modules {
			addGoModDir(found, module)
		}
	}

	out := make([]string, 0, len(found))
	for rel := range found {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func addGoModDir(found map[string]struct{}, rel string) {
	rel = filepath.ToSlash(filepath.Clean(rel))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || strings.HasPrefix(rel, "..") {
		return
	}
	found[rel] = struct{}{}
}

func discoverGoModDirsUnderRoot(repoRoot, root string, excludes []string) ([]string, error) {
	absoluteRoot := filepath.Join(repoRoot, filepath.Clean(root))
	info, err := os.Lstat(absoluteRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	var modules []string
	err = filepath.WalkDir(absoluteRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		return visitGoModPath(repoRoot, path, entry, walkErr, excludes, &modules)
	})
	return modules, err
}

func visitGoModPath(repoRoot, path string, entry fs.DirEntry, walkErr error, excludes []string, modules *[]string) error {
	if walkErr != nil {
		return walkErr
	}
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return nil
	}
	if IsExcluded(rel, excludes) {
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if entry.IsDir() || entry.Name() != "go.mod" {
		return nil
	}
	*modules = append(*modules, filepath.Dir(rel))
	return nil
}

// WriteDeterministicTarGz creates a deterministic tar.gz file from a bundle spec.
// Exported for testing purposes.
func WriteDeterministicTarGz(w io.Writer, repoRoot string, spec MiniBundleSpec) (int64, error) {
	var size int64
	tw, gw, err := newDeterministicTarWriter(w, &size)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = tw.Close()
		_ = gw.Close()
	}()

	paths, err := collectIncludedPaths(repoRoot, spec)
	if err != nil {
		return 0, err
	}

	for _, rel := range paths {
		if err := writeFilesystemEntry(tw, repoRoot, rel); err != nil {
			return 0, err
		}
	}
	if err := writeExtraFileEntries(tw, spec.ExtraFiles); err != nil {
		return 0, err
	}

	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gw.Close(); err != nil {
		return 0, err
	}
	return size, nil
}

func newDeterministicTarWriter(w io.Writer, size *int64) (*tar.Writer, *gzip.Writer, error) {
	compressed, err := gzip.NewWriterLevel(countingWriter{w: w, n: size}, gzip.BestCompression)
	if err != nil {
		return nil, nil, err
	}
	compressed.Header.ModTime = time.Unix(0, 0).UTC()
	compressed.Header.OS = 255
	compressed.Header.Name = "mini-vrooli.tar"
	return tar.NewWriter(compressed), compressed, nil
}

func writeFilesystemEntry(writer *tar.Writer, repoRoot, rel string) error {
	absolute := filepath.Join(repoRoot, rel)
	info, err := os.Lstat(absolute)
	if err != nil || info.IsDir() {
		return err
	}
	link, err := filesystemEntryLink(absolute, info)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}
	prepareTarHeader(header, rel)
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	if header.Typeflag != tar.TypeReg || header.Size <= 0 {
		return nil
	}
	return copyFilesystemEntry(writer, absolute)
}

func filesystemEntryLink(path string, info os.FileInfo) (string, error) {
	if info.Mode()&os.ModeSymlink == 0 {
		return "", nil
	}
	return os.Readlink(path)
}

func prepareTarHeader(header *tar.Header, rel string) {
	header.Name = toTarPath(rel)
	header.ModTime = time.Unix(0, 0).UTC()
	header.AccessTime = time.Unix(0, 0).UTC()
	header.ChangeTime = time.Unix(0, 0).UTC()
	header.Uid = 0
	header.Gid = 0
	header.Uname = ""
	header.Gname = ""
}

func copyFilesystemEntry(writer *tar.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(writer, file)
	_ = file.Close()
	return copyErr
}

func writeExtraFileEntries(writer *tar.Writer, extraFiles map[string][]byte) error {
	keys := make([]string, 0, len(extraFiles))
	for path := range extraFiles {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	for _, path := range keys {
		if err := writeExtraFileEntry(writer, path, extraFiles[path]); err != nil {
			return err
		}
	}
	return nil
}

func writeExtraFileEntry(writer *tar.Writer, rel string, contents []byte) error {
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("extra file path must be relative: %q", rel)
	}
	header := &tar.Header{Name: toTarPath(rel), Size: int64(len(contents)), Mode: 0o644, Typeflag: tar.TypeReg,
		ModTime: time.Unix(0, 0).UTC(), AccessTime: time.Unix(0, 0).UTC(), ChangeTime: time.Unix(0, 0).UTC(), Uid: 0, Gid: 0, Uname: "", Gname: ""}
	if err := writer.WriteHeader(header); err != nil {
		return err
	}
	_, err := io.Copy(writer, bytes.NewReader(contents))
	return err
}

func collectIncludedPaths(repoRoot string, spec MiniBundleSpec) ([]string, error) {
	collector := newIncludedPathCollector(repoRoot, spec)
	for _, root := range spec.IncludeRoots {
		if err := collector.visitRoot(root); err != nil {
			return nil, err
		}
	}
	sort.Strings(collector.paths)
	return collector.paths, nil
}

type includedPathCollector struct {
	repoRoot string
	excludes []string
	extra    map[string]struct{}
	seen     map[string]struct{}
	paths    []string
}

func newIncludedPathCollector(repoRoot string, spec MiniBundleSpec) *includedPathCollector {
	extra := make(map[string]struct{}, len(spec.ExtraFiles))
	for rel := range spec.ExtraFiles {
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		extra[rel] = struct{}{}
	}
	return &includedPathCollector{repoRoot: repoRoot, excludes: spec.Excludes, extra: extra, seen: map[string]struct{}{}}
}

func (c *includedPathCollector) visitRoot(root string) error {
	root = filepath.Clean(root)
	absolute := filepath.Join(c.repoRoot, root)
	info, err := os.Lstat(absolute)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return c.addSingleFileRoot(root)
	}
	return filepath.WalkDir(absolute, c.visitPath)
}

func (c *includedPathCollector) addSingleFileRoot(root string) error {
	if _, isExtra := c.extra[filepath.ToSlash(root)]; isExtra || IsExcluded(root, c.excludes) {
		return nil
	}
	c.add(root)
	return nil
}

func (c *includedPathCollector) visitPath(path string, entry fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	rel, err := filepath.Rel(c.repoRoot, path)
	if err != nil {
		return err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return nil
	}
	if IsExcluded(rel, c.excludes) {
		if entry.IsDir() {
			return filepath.SkipDir
		}
		return nil
	}
	if entry.IsDir() {
		return nil
	}
	if _, isExtra := c.extra[rel]; isExtra {
		return nil
	}
	c.add(rel)
	return nil
}

func (c *includedPathCollector) add(rel string) {
	rel = filepath.Clean(rel)
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) || rel == "." {
		return
	}
	if _, exists := c.seen[rel]; exists {
		return
	}
	c.seen[rel] = struct{}{}
	c.paths = append(c.paths, rel)
}

func buildMiniServiceJSON(repoRoot string, manifest domain.CloudManifest) ([]byte, error) {
	path := filepath.Join(repoRoot, ".vrooli", "service.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return buildGeneratedMiniServiceJSON("", manifest)
		}
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse .vrooli/service.json: %w", err)
	}

	rootVersion := ""
	if rawVersion, ok := doc["version"].(string); ok {
		rootVersion = strings.TrimSpace(rawVersion)
	}
	generated, err := buildGeneratedMiniServiceJSON(rootVersion, manifest)
	if err != nil {
		return nil, err
	}
	return mergeScenarioEnvironment(repoRoot, manifest.Scenario.ID, generated)
}

func mergeScenarioEnvironment(repoRoot, scenarioID string, generated []byte) ([]byte, error) {
	if strings.TrimSpace(scenarioID) == "" {
		return generated, nil
	}
	scenarioServicePath, err := ResolveScenarioFile(repoRoot, scenarioID, "service")
	if err != nil {
		return nil, err
	}
	scenarioBytes, err := os.ReadFile(scenarioServicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return generated, nil
		}
		return nil, err
	}
	var scenarioDoc map[string]interface{}
	if err := json.Unmarshal(scenarioBytes, &scenarioDoc); err != nil {
		return nil, fmt.Errorf("parse scenario service.json: %w", err)
	}
	environment, ok := scenarioDoc["environment"].(map[string]interface{})
	if !ok || len(environment) == 0 {
		return generated, nil
	}
	var generatedDoc map[string]interface{}
	if err := json.Unmarshal(generated, &generatedDoc); err != nil {
		return nil, fmt.Errorf("parse generated service.json: %w", err)
	}
	generatedDoc["environment"] = environment
	return json.MarshalIndent(generatedDoc, "", "  ")
}

func buildGeneratedMiniServiceJSON(version string, manifest domain.CloudManifest) ([]byte, error) {
	if strings.TrimSpace(version) == "" {
		version = "2.0.0"
	}

	requiredResources := stringutil.SortedUnique(manifest.Bundle.Resources)
	requiredScenarios := stringutil.SortedUnique(manifest.Bundle.Scenarios)
	if manifest.Scenario.ID != "" && !stringutil.Contains(requiredScenarios, manifest.Scenario.ID) {
		requiredScenarios = append(requiredScenarios, manifest.Scenario.ID)
		sort.Strings(requiredScenarios)
	}

	resourceDeps := make(map[string]map[string]interface{}, len(requiredResources))
	for _, name := range requiredResources {
		resourceDeps[name] = map[string]interface{}{
			"enabled":  true,
			"required": true,
		}
	}

	scenarioDeps := make(map[string]map[string]interface{}, len(requiredScenarios))
	for _, name := range requiredScenarios {
		required := name == manifest.Scenario.ID
		scenarioDeps[name] = map[string]interface{}{
			"enabled":  true,
			"required": required,
		}
	}

	doc := map[string]interface{}{
		"version": version,
		"service": map[string]interface{}{
			"name":        "mini-vrooli",
			"displayName": "Mini Vrooli",
			"description": fmt.Sprintf("Generated native VPS bundle for %s", manifest.Scenario.ID),
			"type":        "project",
		},
		"dependencies": map[string]interface{}{
			"resources": resourceDeps,
			"scenarios": scenarioDeps,
		},
		"lifecycle": map[string]interface{}{
			"version": "2.0.0",
			"setup": map[string]interface{}{
				"description": "Generated native VPS setup",
				"steps":       []map[string]interface{}{},
			},
		},
	}

	return json.MarshalIndent(doc, "", "  ")
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

// buildScenarioServiceJSONWithFixedPorts reads the target scenario's service.json
// and replaces port ranges with fixed port values from the deployment manifest.
// This ensures VPS deployments use exact ports rather than dynamic allocation.
func buildScenarioServiceJSONWithFixedPorts(repoRoot string, manifest domain.CloudManifest) ([]byte, error) {
	if manifest.Scenario.ID == "" {
		return nil, nil
	}

	// Read the scenario's service.json
	path, err := ResolveScenarioFile(repoRoot, manifest.Scenario.ID, "service")
	if err != nil {
		return nil, err
	}
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

// SafeFilename converts a string to a safe filename.
func SafeFilename(s string) string {
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
