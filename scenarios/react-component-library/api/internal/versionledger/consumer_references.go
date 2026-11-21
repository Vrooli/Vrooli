package versionledger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"react-component-library/internal/librarywalk"
)

// Why this file exists.
//
// The package build and retention used to disagree about who a consumer is.
// The build (packages/react-component-library/tooling/version-import-scan.mjs)
// fails when any file under scenarios/ or templates/ imports a specifier the
// package no longer exports. Retention counted only the workbench, adoption
// records and library sources, so a scenario that imported
// `SettingsList/0` without an adoption record watched SettingsList@0.1.6 get
// evicted, and the next package build failed on twelve broken imports
// (bug knw-1788324810742982302).
//
// consumerSourceReferences reads the same tree the build reads and records
// every package import there as an external root. Exact specifiers pin their
// release; a major selector pins the release the build resolves it to; a bare
// specifier follows `latest`, which is already a root.

// consumerSpecifierRE finds the package specifier in any quoted position:
// static and dynamic imports, re-exports, `vi.mock(...)`, and type imports.
// Retention errs toward keeping, so matching a specifier that is only
// mentioned costs a little disk; missing a real import costs a broken build.
var consumerSpecifierRE = regexp.MustCompile("[\"'`](" + regexp.QuoteMeta(packageSpecifierPrefix) + "[^\"'`\\s]+)[\"'`]")

// consumerSourceExtensions matches the extensions the package build scans.
var consumerSourceExtensions = map[string]bool{".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".mjs": true, ".cjs": true}

// consumerFixtureDirs hold specifiers that exist to exercise the resolver,
// including deliberately absent ones. They describe no consumer.
var consumerFixtureDirs = []string{
	string(filepath.Separator) + filepath.Join("catalog", "calibration") + string(filepath.Separator),
	string(filepath.Separator) + filepath.Join("tools", "testdata") + string(filepath.Separator),
}

// consumerSourceReferences walks every consumer of the package. Inside this
// scenario it also resolves relative imports into version folders, which is
// how the workbench and the library's own tests reach historical sources.
//
// Unlike the build scan it keeps test files: a test that imports a release
// fails the moment that release is evicted, so it is as much a consumer as
// the code it tests. The exception is a release's own test folder, whose
// imports belong to that release (newReleaseTestOwners).
func (r *Repository) consumerSourceReferences(ctx context.Context, byVersion map[string][]VersionReference, versions []sourceVersion) error {
	if strings.TrimSpace(r.sourceRoot) == "" {
		// Without a library root there is no tree to place consumers in;
		// resolving "" would walk the process working directory instead.
		return nil
	}
	libraryRoot := filepath.Clean(r.sourceRoot)
	scenarioRoot := filepath.Dir(libraryRoot)
	scenariosRoot := filepath.Dir(scenarioRoot)
	// Each root is a directory whose immediate children are projects. A
	// project's top-level docs tree holds retained evidence and history, not
	// live consumers, exactly as the build scan treats it.
	roots := []string{scenarioRoot}
	projectDepth := 0
	if filepath.Base(scenariosRoot) == "scenarios" {
		repoRoot := filepath.Dir(scenariosRoot)
		roots = []string{scenariosRoot, filepath.Join(repoRoot, "templates")}
		projectDepth = 1
	}
	resolve := newMajorResolver(versions)
	releaseTest := newReleaseTestOwners(libraryRoot, versions)
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("inspect consumer root %s: %w", root, err)
		}
		err := librarywalk.WalkContext(ctx, root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				if os.IsNotExist(walkErr) {
					return filepath.SkipDir
				}
				return walkErr
			}
			if entry.IsDir() {
				if path == root {
					return nil
				}
				if skipConsumerDir(root, path, entry.Name(), projectDepth, libraryRoot) {
					return filepath.SkipDir
				}
				return nil
			}
			if !consumerSourceExtensions[strings.ToLower(filepath.Ext(path))] {
				return nil
			}
			for _, fixture := range consumerFixtureDirs {
				if strings.Contains(path, fixture) {
					return nil
				}
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read consumer source %s: %w", path, err)
			}
			internal := pathWithin(scenarioRoot, path)
			hasPackage := bytes.Contains(body, []byte(packageSpecifierPrefix))
			// A relative import can only land in a version folder when its
			// path names one, so files without that segment skip resolution.
			hasVersionPath := internal && bytes.Contains(body, []byte("versions/"))
			if !hasPackage && !hasVersionPath {
				return nil
			}
			rel := filepath.ToSlash(path)
			if relative, relErr := filepath.Rel(filepath.Dir(root), path); relErr == nil {
				rel = filepath.ToSlash(relative)
			}
			// A consumer is an external root. A release's own test folder is
			// not: its imports are that release's edges, live while the
			// release is materialized and gone with it when it is evicted —
			// exactly when the test runner stops running the folder.
			template := VersionReference{Kind: "consumer-import", OwnerPath: rel, OwnerScenario: consumerOwner(root, path, projectDepth)}
			packageEvidence := "consumer source imports this release through the package export map"
			relativeEvidence := "consumer source import resolves into a surviving version folder"
			if ownerID, ownerVersion, isReleaseTest, live := releaseTest(path); isReleaseTest {
				if !live {
					return nil
				}
				template = VersionReference{Kind: "release-test-import", OwnerLibraryID: ownerID, OwnerVersion: ownerVersion, OwnerPath: rel}
				packageEvidence = "the owning release's test imports this release"
				relativeEvidence = packageEvidence
			}
			record := func(libraryID, version, specifier, evidence string) {
				if libraryID == template.OwnerLibraryID && version == template.OwnerVersion {
					return
				}
				ref := template
				ref.ImportSpecifier, ref.Evidence = specifier, evidence
				key := sourceReferenceKey(libraryID, version)
				byVersion[key] = appendUniqueReference(byVersion[key], ref)
			}
			if hasPackage {
				for _, match := range consumerSpecifierRE.FindAllStringSubmatch(string(body), -1) {
					if libraryID, version, ok := resolve(match[1]); ok {
						record(libraryID, version, match[1], packageEvidence)
					}
				}
			}
			if hasVersionPath {
				for _, match := range sourceImportRE.FindAllStringSubmatch(string(body), -1) {
					specifier := match[1]
					if !strings.HasPrefix(specifier, ".") {
						continue
					}
					target := resolveSourceImport(filepath.Dir(path), specifier)
					for _, targetVersion := range versions {
						if pathWithin(targetVersion.directory, target) {
							record(targetVersion.libraryID, targetVersion.version, specifier, relativeEvidence)
							break
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan consumer sources under %s: %w", root, err)
		}
	}
	return nil
}

// newReleaseTestOwners recognises `library/tests/<kind>/<asset>/<version>/`,
// the folder that holds one release's own tests, and reports which release
// owns a file there and whether that release is materialized (its version
// folder is on disk under any kind). The UI test config
// (ui/vite.config.ts, coldReleaseTestFolders) skips the folders of releases
// that are not, so the two agree on which tests can run.
func newReleaseTestOwners(libraryRoot string, versions []sourceVersion) func(string) (string, string, bool, bool) {
	testsRoot := filepath.Join(libraryRoot, "tests")
	indexed := map[string]string{}
	for _, v := range versions {
		indexed[filepath.Base(filepath.Dir(filepath.Dir(v.directory)))+"@"+v.version] = v.libraryID
	}
	return func(path string) (string, string, bool, bool) {
		rel, err := filepath.Rel(testsRoot, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", "", false, false
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 4 || !isExactVersion(parts[2]) {
			return "", "", false, false
		}
		asset, version := parts[1], parts[2]
		matches, _ := filepath.Glob(filepath.Join(libraryRoot, "*", asset, "versions", version))
		if len(matches) == 0 {
			return "", "", true, false
		}
		libraryID := indexed[asset+"@"+version]
		if libraryID == "" {
			// On disk but unknown to the index: name it the way the package
			// does, and the graph treats the unindexed owner as a root.
			libraryID = "react-component-library:" + asset
		}
		return libraryID, version, true, true
	}
}

// skipConsumerDir prunes trees that hold no live consumer. Library version
// trees are skipped because their sources are owned edges read by the
// version scanners; the library's tests and support files are consumers.
func skipConsumerDir(root, path, name string, projectDepth int, libraryRoot string) bool {
	switch name {
	case ".git", ".vite", "coverage", retiredDirName, "node_modules", "dist":
		return true
	}
	if strings.HasPrefix(name, ".vrooli-artifact-stage-") {
		return true
	}
	if name == "docs" && consumerDepth(root, path) == projectDepth+1 {
		return true
	}
	// <library>/<kind>/<asset>/versions holds every version's source.
	if name == "versions" && filepath.Dir(filepath.Dir(filepath.Dir(path))) == libraryRoot {
		return true
	}
	return false
}

func consumerDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(filepath.ToSlash(rel), "/") + 1
}

// consumerOwner names the project a consumer file belongs to: the scenario or
// template directory directly under the scanned root.
func consumerOwner(root, path string, projectDepth int) string {
	if projectDepth == 0 {
		return filepath.Base(root)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	project, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
	return project
}

// newMajorResolver maps a consumer specifier to the release it pins, using the
// package export rule (tooling/export-resolution.mjs): an exact specifier
// names itself, and `Name/<major>` resolves to the newest non-deprecated
// release in that major, or the newest release when every one is deprecated.
// Retired and draft versions are never selector targets.
func newMajorResolver(versions []sourceVersion) func(string) (string, string, bool) {
	type candidate struct {
		version    string
		deprecated bool
	}
	byMajor := map[string][]candidate{}
	for _, v := range versions {
		status := strings.ToLower(v.status)
		if status == "retired" || strings.Contains(status, "draft") || !isExactVersion(v.version) {
			continue
		}
		major, _, _ := strings.Cut(v.version, ".")
		key := v.libraryID + "@" + major
		byMajor[key] = append(byMajor[key], candidate{version: v.version, deprecated: status == "deprecated"})
	}
	return func(specifier string) (string, string, bool) {
		libraryID, selector, ok := packageSelector(specifier)
		if !ok || selector == "" {
			return "", "", false
		}
		if isExactVersion(selector) {
			return libraryID, selector, true
		}
		var best, bestDeprecated string
		for _, item := range byMajor[libraryID+"@"+selector] {
			if item.deprecated {
				if bestDeprecated == "" || compareRelease(item.version, bestDeprecated) > 0 {
					bestDeprecated = item.version
				}
				continue
			}
			if best == "" || compareRelease(item.version, best) > 0 {
				best = item.version
			}
		}
		if best == "" {
			best = bestDeprecated
		}
		return libraryID, best, best != ""
	}
}

// compareRelease orders two `major.minor.patch` strings numerically.
func compareRelease(left, right string) int {
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	for index := 0; index < 3 && index < len(leftParts) && index < len(rightParts); index++ {
		l, _ := strconv.Atoi(leftParts[index])
		r, _ := strconv.Atoi(rightParts[index])
		if l != r {
			if l < r {
				return -1
			}
			return 1
		}
	}
	return 0
}
