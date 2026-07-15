package adoptions

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"react-component-library/internal/components"
)

// Reconcile backfills records from provenance already on disk. It deliberately
// bypasses Refresh's DriftReporter: initial inventory must not create backlog
// noise. A subsequent normal Refresh owns all future first-transition reports.
func (s *service) Reconcile(ctx context.Context, in ReconcileInput) (ReconcileResult, error) {
	scanner, ok := s.files.(ScenarioProvenanceScanner)
	if !ok {
		return ReconcileResult{}, fmt.Errorf("adoptions reconcile: provenance scanner not configured")
	}
	files, err := scanner.ScanProvenance(ctx)
	if err != nil {
		return ReconcileResult{}, err
	}
	return s.reconcileFiles(ctx, files, in.Apply)
}

func (s *service) reconcileFiles(ctx context.Context, files []ProvenanceFile, apply bool) (ReconcileResult, error) {
	result := ReconcileResult{Scanned: len(files)}
	componentsList, err := s.library.List(ctx, components.SearchQuery{Limit: 1000})
	if err != nil {
		return result, err
	}
	byLibraryID := make(map[string]components.Component, len(componentsList))
	for _, component := range componentsList {
		byLibraryID[component.LibraryID] = component
	}
	existing, err := s.repo.List(ctx, ListQuery{Limit: 100000})
	if err != nil {
		return result, err
	}
	recorded := map[string]bool{}
	recordedIDs := map[string]bool{}
	for _, row := range existing {
		recordedIDs[row.ID] = true
		recorded[provenancePathKey(row.Scenario, row.AdoptedPath)] = true
		for _, file := range row.Files {
			recorded[provenancePathKey(row.Scenario, file.AdoptedPath)] = true
		}
	}
	groups := map[string][]ProvenanceFile{}
	for _, file := range files {
		if recorded[provenancePathKey(file.Scenario, file.AdoptedPath)] {
			result.AlreadyRecorded++
			continue
		}
		key := file.Scenario + "\x00" + file.LibraryID + "\x00" + file.Version + "\x00" + file.AdoptionID
		if file.AdoptionID == "" {
			key += "\x00" + file.AdoptedPath
		}
		groups[key] = append(groups[key], file)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		group := groups[key]
		component, ok := byLibraryID[group[0].LibraryID]
		if !ok {
			result.Findings = append(result.Findings, reconcileFinding(group[0], "library component is not indexed"))
			continue
		}
		version, err := s.library.GetVersion(ctx, component.ID, group[0].Version)
		if err != nil {
			result.Findings = append(result.Findings, reconcileFinding(group[0], "library version is not indexed"))
			continue
		}
		files, entry, err := reconcileAdoptionFiles(group, version)
		if err != nil {
			result.Findings = append(result.Findings, reconcileFinding(group[0], err.Error()))
			continue
		}
		if !apply {
			result.Created++
			continue
		}
		id := strings.TrimSpace(group[0].AdoptionID)
		// Historical copies can preserve the same provenance UUID in more
		// than one scenario. Preserve it only when unused; the fresh database
		// ID still gives the reconciled record a stable unique identity.
		if recordedIDs[id] {
			id = ""
		}
		created, err := s.repo.Create(ctx, CreateInput{ID: id, ComponentID: component.ID, LibraryID: component.LibraryID, Scenario: group[0].Scenario, AdoptedPath: entry.AdoptedPath, AdoptedVersion: group[0].Version, SourceSHA256: entry.SourceSHA256, AdoptedSnapshotSHA256: entry.AdoptedSnapshotSHA256, Files: files})
		if err != nil {
			return result, err
		}
		libraryStatus, localStatus, detail := s.computeStatus(ctx, created)
		if _, err := s.repo.ApplyRefresh(ctx, []RefreshUpdate{{ID: created.ID, LibraryVersionStatus: libraryStatus, LocalStatus: localStatus, StatusDetail: detail, RefreshedAt: s.clock.Now().UTC()}}); err != nil {
			return result, err
		}
		result.Created++
		recordedIDs[created.ID] = true
	}
	return result, nil
}

func reconcileAdoptionFiles(group []ProvenanceFile, version components.ComponentVersion) ([]AdoptionFile, AdoptionFile, error) {
	files := make([]AdoptionFile, 0, len(group))
	var entry AdoptionFile
	for _, scanned := range group {
		libraryFile, ok := matchReconciledLibraryFile(version, scanned.AdoptedPath)
		if !ok {
			return nil, AdoptionFile{}, fmt.Errorf("no library file matches %s", scanned.AdoptedPath)
		}
		file := AdoptionFile{LibraryPath: libraryFile.Path, AdoptedPath: scanned.AdoptedPath, SourceSHA256: libraryFile.ContentSHA256, AdoptedSnapshotSHA256: hashBytes(scanned.Content)}
		files = append(files, file)
		if libraryFile.IsEntry || entry.AdoptedPath == "" {
			entry = file
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].AdoptedPath < files[j].AdoptedPath })
	return files, entry, nil
}

func matchReconciledLibraryFile(version components.ComponentVersion, adoptedPath string) (components.ComponentVersionFile, bool) {
	name := filepath.Base(adoptedPath)
	for _, file := range version.Files {
		if file.Path == name {
			return file, true
		}
	}
	if strings.HasSuffix(name, ".tsx") {
		for _, file := range version.Files {
			if file.IsEntry {
				return file, true
			}
		}
	}
	// A single-file component may be resolved to a template-specific filename
	// (for example Button.tsx → ui/src/components/ui/button.tsx). Its entry is
	// still unambiguous even though the adopter's basename differs.
	if len(version.Files) == 1 {
		return version.Files[0], true
	}
	if filepath.Base(version.SourcePath) == name {
		return components.ComponentVersionFile{Path: name, ContentSHA256: version.ContentSHA256, IsEntry: true}, true
	}
	return components.ComponentVersionFile{}, false
}

func reconcileFinding(file ProvenanceFile, detail string) ReconcileFinding {
	return ReconcileFinding{Scenario: file.Scenario, AdoptedPath: file.AdoptedPath, LibraryID: file.LibraryID, Version: file.Version, Detail: detail}
}

func provenancePathKey(scenario, path string) string {
	return scenario + "\x00" + filepath.ToSlash(path)
}
