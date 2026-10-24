package versions

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

var (
	reachabilityVersionPattern     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	reachabilitySpecifierPattern   = regexp.MustCompile(`@vrooli/react-component-library/([A-Za-z0-9_-]+)(?:/(\d+(?:\.\d+){0,2}))?`)
	reachabilityImportPattern      = regexp.MustCompile(`(?:from\s*|import\s*|require\s*\(\s*)["']([^"']+)["']`)
	reachabilitySourceExtensions   = map[string]bool{".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true}
	reachabilitySkippedDirectories = map[string]bool{"node_modules": true, "dist": true, ".git": true, "coverage": true, ".retired": true, ".pnpm": true}
)

type reachabilityVersion struct {
	Asset   string `json:"asset"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Path    string `json:"path"`
	Lines   int    `json:"lines"`
	Files   int    `json:"files"`
}

type reachabilityAsset struct {
	Asset    string   `json:"asset"`
	Kind     string   `json:"kind"`
	Versions []string `json:"versions"`
}

// ReachabilityReport is the reproducible corpus decision used by versions
// reachability and versions reap. Newest-per-major entries are included in
// reachable because they are public package selectors even when this
// workspace has no current importer for that selector.
type ReachabilityReport struct {
	Workspace            string                `json:"workspace"`
	LibraryRoot          string                `json:"library_root"`
	OnDisk               []reachabilityVersion `json:"on_disk"`
	Reachable            []reachabilityVersion `json:"reachable"`
	Unreachable          []reachabilityVersion `json:"unreachable"`
	AssetsWithNoImporter []reachabilityAsset   `json:"assets_with_no_importer"`
	ImporterFiles        int                   `json:"importer_files"`
	SpecifierCount       int                   `json:"specifier_count"`
}

type reachabilityIndex struct {
	Report ReachabilityReport
	ByKey  map[string]reachabilityVersion
}

func (h *handlers) reachability(ctx cliapp.RunContext) error {
	workspace, err := workspaceRoot(ctx.Flag("workspace"))
	if err != nil {
		return err
	}
	index, err := buildReachabilityIndex(workspace)
	if err != nil {
		return err
	}
	return renderReachability(ctx, index.Report, "Reachability scan")
}

func (h *handlers) reapReachable(ctx cliapp.RunContext) error {
	workspace, err := workspaceRoot(ctx.Flag("workspace"))
	if err != nil {
		return err
	}
	index, err := buildReachabilityIndex(workspace)
	if err != nil {
		return err
	}
	plan := append([]reachabilityVersion(nil), index.Report.Unreachable...)
	sortReachabilityVersions(plan)
	ledgerBefore, err := releaseProvenanceEntryCount(index.Report.LibraryRoot)
	if err != nil {
		return err
	}
	if !ctx.BoolFlag("confirm") {
		return renderReap(ctx, plan, ledgerBefore, ledgerBefore, false)
	}
	for _, item := range plan {
		if !isSafeVersionPath(index.Report.LibraryRoot, item.Path) {
			return fmt.Errorf("refusing unsafe version path %q", item.Path)
		}
		if err := os.RemoveAll(item.Path); err != nil {
			return fmt.Errorf("remove unreachable version %s@%s: %w", item.Asset, item.Version, err)
		}
	}
	ledgerAfter, err := releaseProvenanceEntryCount(index.Report.LibraryRoot)
	if err != nil {
		return err
	}
	if ledgerAfter != ledgerBefore {
		return fmt.Errorf("release provenance entry count changed during reap: before=%d after=%d", ledgerBefore, ledgerAfter)
	}
	return renderReap(ctx, plan, ledgerBefore, ledgerAfter, true)
}

func renderReachability(ctx cliapp.RunContext, report ReachabilityReport, summary string) error {
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), report)
	}
	return ctx.RenderOperational(cliapp.OperationalReport{
		Status: []string{fmt.Sprintf("%s: on_disk=%d reachable=%d unreachable=%d assets_with_no_importer=%d.", summary, len(report.OnDisk), len(report.Reachable), len(report.Unreachable), len(report.AssetsWithNoImporter))},
		Triage: []cliapp.TriageGroup{{Heading: "Unreachable versions", Items: reachabilityLines(report.Unreachable)}},
	})
}

func renderReap(ctx cliapp.RunContext, plan []reachabilityVersion, ledgerBefore, ledgerAfter int, applied bool) error {
	mode := "dry-run"
	if applied {
		mode = "applied"
	}
	payload := struct {
		Mode                 string                `json:"mode"`
		RemovedOrWouldRemove []reachabilityVersion `json:"removed_or_would_remove"`
		LedgerEntriesBefore  int                   `json:"ledger_entries_before"`
		LedgerEntriesAfter   int                   `json:"ledger_entries_after"`
	}{Mode: mode, RemovedOrWouldRemove: plan, LedgerEntriesBefore: ledgerBefore, LedgerEntriesAfter: ledgerAfter}
	if ctx.JSON() {
		return cliapp.PrintJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderMutation(cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Reachability reap %s: %d version directories; release-provenance entries %d → %d.", mode, len(plan), ledgerBefore, ledgerAfter)},
		Changes: reachabilityLines(plan),
	})
}

func reachabilityLines(items []reachabilityVersion) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s/%s@%s (%d files, %d lines)", item.Kind, item.Asset, item.Version, item.Files, item.Lines))
	}
	return lines
}

func workspaceRoot(requested string) (string, error) {
	if requested != "" {
		absolute, err := filepath.Abs(requested)
		if err != nil {
			return "", fmt.Errorf("resolve workspace %q: %w", requested, err)
		}
		return absolute, nil
	}
	start, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	for current := start; ; current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start, nil
		}
	}
}

func buildReachabilityIndex(workspace string) (reachabilityIndex, error) {
	libraryRoot := filepath.Join(workspace, "scenarios", "react-component-library", "library")
	if _, err := os.Stat(libraryRoot); err != nil {
		return reachabilityIndex{}, fmt.Errorf("library root %q: %w", libraryRoot, err)
	}
	versions, byKey, err := inventoryVersions(libraryRoot)
	if err != nil {
		return reachabilityIndex{}, err
	}
	used, relativeKeys, importedAssets, importerFiles, specifierCount, err := scanSpecifiers(workspace, libraryRoot)
	if err != nil {
		return reachabilityIndex{}, err
	}
	reachableKeys := make(map[string]bool)
	for asset, specifiers := range used {
		assetVersions := versions[asset]
		for specifier := range specifiers {
			if selected := resolveSpecifier(assetVersions, specifier); selected != nil {
				reachableKeys[versionKey(selected.Kind, selected.Asset, selected.Version)] = true
			}
		}
	}
	for key := range relativeKeys {
		reachableKeys[key] = true
	}
	// The package contract promises the newest materialized version in every
	// major line, even if this workspace currently has no importer for it.
	for _, assetVersions := range versions {
		newestByMajor := map[string]*reachabilityVersion{}
		for i := range assetVersions {
			item := &assetVersions[i]
			major := strings.SplitN(item.Version, ".", 2)[0]
			if current := newestByMajor[major]; current == nil || compareVersions(current.Version, item.Version) < 0 {
				newestByMajor[major] = item
			}
		}
		for _, item := range newestByMajor {
			reachableKeys[versionKey(item.Kind, item.Asset, item.Version)] = true
		}
	}
	reachable := make([]reachabilityVersion, 0, len(reachableKeys))
	unreachable := make([]reachabilityVersion, 0)
	for _, item := range byKey {
		if reachableKeys[versionKey(item.Kind, item.Asset, item.Version)] {
			reachable = append(reachable, item)
		} else {
			unreachable = append(unreachable, item)
		}
	}
	sortReachabilityVersions(versionsFlattened(versions))
	sortReachabilityVersions(reachable)
	sortReachabilityVersions(unreachable)
	assetsWithNoImporter := make([]reachabilityAsset, 0)
	for asset, assetVersions := range versions {
		if importedAssets[asset] {
			continue
		}
		if len(assetVersions) == 0 {
			continue
		}
		versionsForAsset := make([]string, 0, len(assetVersions))
		for _, item := range assetVersions {
			versionsForAsset = append(versionsForAsset, item.Version)
		}
		sort.Slice(versionsForAsset, func(i, j int) bool { return compareVersions(versionsForAsset[i], versionsForAsset[j]) < 0 })
		assetsWithNoImporter = append(assetsWithNoImporter, reachabilityAsset{Asset: assetVersions[0].Asset, Kind: assetVersions[0].Kind, Versions: versionsForAsset})
	}
	sort.Slice(assetsWithNoImporter, func(i, j int) bool { return assetsWithNoImporter[i].Asset < assetsWithNoImporter[j].Asset })
	return reachabilityIndex{Report: ReachabilityReport{Workspace: workspace, LibraryRoot: libraryRoot, OnDisk: versionsFlattened(versions), Reachable: reachable, Unreachable: unreachable, AssetsWithNoImporter: assetsWithNoImporter, ImporterFiles: importerFiles, SpecifierCount: specifierCount}, ByKey: byKey}, nil
}

func inventoryVersions(libraryRoot string) (map[string][]reachabilityVersion, map[string]reachabilityVersion, error) {
	byAsset := map[string][]reachabilityVersion{}
	byKey := map[string]reachabilityVersion{}
	// Walk version directories once so metadata counts include every file while
	// avoiding one record per source file.
	for _, kindEntry := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		kindRoot := filepath.Join(libraryRoot, kindEntry)
		assets, readErr := os.ReadDir(kindRoot)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, nil, readErr
		}
		for _, assetEntry := range assets {
			if !assetEntry.IsDir() {
				continue
			}
			versionsRoot := filepath.Join(kindRoot, assetEntry.Name(), "versions")
			versionEntries, readErr := os.ReadDir(versionsRoot)
			if os.IsNotExist(readErr) {
				continue
			}
			if readErr != nil {
				return nil, nil, readErr
			}
			for _, versionEntry := range versionEntries {
				if !versionEntry.IsDir() || !reachabilityVersionPattern.MatchString(versionEntry.Name()) {
					continue
				}
				versionPath := filepath.Join(versionsRoot, versionEntry.Name())
				lines, files, err := countVersionFiles(versionPath)
				if err != nil {
					return nil, nil, err
				}
				item := reachabilityVersion{Asset: assetEntry.Name(), Kind: kindEntry, Version: versionEntry.Name(), Path: versionPath, Lines: lines, Files: files}
				byAsset[item.Asset] = append(byAsset[item.Asset], item)
				byKey[versionKey(item.Kind, item.Asset, item.Version)] = item
			}
		}
	}
	for asset := range byAsset {
		sortReachabilityVersions(byAsset[asset])
	}
	return byAsset, byKey, nil
}

func countVersionFiles(root string) (int, int, error) {
	lines, files := 0, 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		files++
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines += len(strings.Split(string(body), "\n"))
		return nil
	})
	return lines, files, err
}

func scanSpecifiers(workspace, libraryRoot string) (map[string]map[string]bool, map[string]bool, map[string]bool, int, int, error) {
	used := map[string]map[string]bool{}
	relativeKeys := map[string]bool{}
	importedAssets := map[string]bool{}
	importerFiles, specifierCount := 0, 0
	err := filepath.WalkDir(workspace, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != workspace && reachabilitySkippedDirectories[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !reachabilitySourceExtensions[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(body)
		matches := reachabilitySpecifierPattern.FindAllStringSubmatch(source, -1)
		imports := reachabilityImportPattern.FindAllStringSubmatch(source, -1)
		if len(matches) == 0 && len(imports) == 0 {
			return nil
		}
		importerFiles++
		for _, match := range matches {
			asset, selector := match[1], match[2]
			if used[asset] == nil {
				used[asset] = map[string]bool{}
			}
			used[asset][selector] = true
			importedAssets[asset] = true
			specifierCount++
		}
		for _, match := range imports {
			specifier := match[1]
			if !strings.HasPrefix(specifier, ".") {
				continue
			}
			if key, asset, ok := resolveRelativeVersionImport(path, specifier, libraryRoot); ok {
				relativeKeys[key] = true
				importedAssets[asset] = true
				specifierCount++
			}
		}
		return nil
	})
	return used, relativeKeys, importedAssets, importerFiles, specifierCount, err
}

func resolveRelativeVersionImport(importerPath, specifier, libraryRoot string) (string, string, bool) {
	resolved := filepath.Clean(filepath.Join(filepath.Dir(importerPath), filepath.FromSlash(specifier)))
	relative, err := filepath.Rel(libraryRoot, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 4 || parts[2] != "versions" || !reachabilityVersionPattern.MatchString(parts[3]) {
		return "", "", false
	}
	return versionKey(parts[0], parts[1], parts[3]), parts[1], true
}

func resolveSpecifier(versions []reachabilityVersion, selector string) *reachabilityVersion {
	if len(versions) == 0 {
		return nil
	}
	var selected *reachabilityVersion
	for i := range versions {
		candidate := &versions[i]
		matches := selector == "" || candidate.Version == selector || strings.HasPrefix(candidate.Version, selector+".")
		if !matches || (selector != "" && strings.Count(selector, ".") == 1 && !strings.HasPrefix(candidate.Version, selector+".")) {
			continue
		}
		if selected == nil || compareVersions(selected.Version, candidate.Version) < 0 {
			selected = candidate
		}
	}
	return selected
}

func versionKey(kind, asset, version string) string { return kind + "/" + asset + "/" + version }

func versionsFlattened(versions map[string][]reachabilityVersion) []reachabilityVersion {
	all := make([]reachabilityVersion, 0)
	for _, items := range versions {
		all = append(all, items...)
	}
	sortReachabilityVersions(all)
	return all
}

func sortReachabilityVersions(items []reachabilityVersion) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Asset != items[j].Asset {
			return items[i].Asset < items[j].Asset
		}
		return compareVersions(items[i].Version, items[j].Version) < 0
	})
}

func compareVersions(left, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for i := 0; i < 3; i++ {
		leftValue, _ := strconv.Atoi(leftParts[i])
		rightValue, _ := strconv.Atoi(rightParts[i])
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
}

func releaseProvenanceEntryCount(libraryRoot string) (int, error) {
	path := filepath.Join(libraryRoot, "release-provenance.json")
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read release provenance: %w", err)
	}
	var document struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		return 0, fmt.Errorf("decode release provenance: %w", err)
	}
	return len(document.Entries), nil
}

func isSafeVersionPath(libraryRoot, path string) bool {
	relative, err := filepath.Rel(libraryRoot, path)
	if err != nil || strings.HasPrefix(relative, "..") {
		return false
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	return len(parts) == 4 && parts[2] == "versions" && reachabilityVersionPattern.MatchString(parts[3])
}
