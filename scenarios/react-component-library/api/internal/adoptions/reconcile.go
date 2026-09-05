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
	// Foundations are excluded from the ordinary component listing because
	// they are not direct visual catalog entries, but adopted copies can still
	// carry their provenance. Include the explicit foundation projection so
	// closure reconciliation does not misclassify valid token/icon assets as
	// missing.
	foundations, err := s.library.List(ctx, components.SearchQuery{AssetKind: components.AssetKindFoundation, Limit: 1000})
	if err != nil {
		return result, err
	}
	componentsList = append(componentsList, foundations...)
	byLibraryID := make(map[string]components.Component, len(componentsList))
	byCatalogID := make(map[string]components.Component, len(componentsList))
	for _, component := range componentsList {
		byLibraryID[component.LibraryID] = component
		if strings.TrimSpace(component.CatalogID) != "" {
			byCatalogID[component.CatalogID] = component
		}
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
			// Older adopted copies used the catalog asset id in
			// @vrooliComponentSource. Treat that value as a compatibility
			// alias, but persist the canonical library id in the new record.
			component, ok = byCatalogID[group[0].LibraryID]
		}
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
	if apply {
		healed, err := s.healPoisonedSnapshots(ctx, existing)
		if err != nil {
			return result, err
		}
		result.Healed = healed
	}
	return result, nil
}

// healPoisonedSnapshots re-derives honest baselines for already-recorded rows
// whose snapshot was captured from a locally-modified copy — the backfill
// hash-masking defect, where a modified file reads CLEAN because its snapshot is
// its own (already-edited) bytes. Only rows that currently read CLEAN yet are not
// provably clean against the adopted library version are touched, so honestly
// clean and honestly modified rows are never churned. Corrected rows re-read
// MODIFIED, which makes reconverge treat them as untouchable.
func (s *service) healPoisonedSnapshots(ctx context.Context, rows []Adoption) (int, error) {
	healed := 0
	for _, row := range rows {
		if len(row.Files) == 0 {
			// Parent-only rows are guarded at reconverge time; there is no
			// per-file baseline to re-derive here.
			continue
		}
		_, localStatus, _ := s.computeStatus(ctx, row)
		if localStatus != LocalStatusClean && !s.hasLegacyRawSnapshot(ctx, row) {
			continue
		}
		corrected, entrySnapshot, changed := s.honestSnapshots(ctx, row)
		if !changed {
			continue
		}
		rebaselined, err := s.repo.Rebaseline(ctx, RebaselineInput{ID: row.ID, AdoptedSnapshotSHA256: entrySnapshot, Files: corrected})
		if err != nil {
			return healed, err
		}
		libStatus, ls, detail := s.computeStatus(ctx, rebaselined)
		if _, err := s.repo.ApplyRefresh(ctx, []RefreshUpdate{{ID: row.ID, LibraryVersionStatus: libStatus, LocalStatus: ls, StatusDetail: detail, RefreshedAt: s.clock.Now().UTC()}}); err != nil {
			return healed, err
		}
		healed++
	}
	return healed, nil
}

// hasLegacyRawSnapshot identifies rows written before adoption snapshots were
// normalized to exclude generated provenance. Such rows appear MODIFIED under
// the current comparison even when their stored snapshot is simply the exact
// header-bearing bytes on disk; they need one safe rebaseline pass.
func (s *service) hasLegacyRawSnapshot(ctx context.Context, row Adoption) bool {
	for _, file := range row.Files {
		body, err := s.files.Read(ctx, row.Scenario, file.AdoptedPath)
		if err != nil || file.AdoptedSnapshotSHA256 == "" {
			continue
		}
		if hashBytes(body) == file.AdoptedSnapshotSHA256 && adoptedSnapshotHash(string(body)) != file.AdoptedSnapshotSHA256 {
			return true
		}
	}
	return false
}

// honestSnapshots recomputes each file's pristine snapshot from the adopted
// library version, mirroring reconciledSnapshot but reading the file from disk.
// It returns the corrected files, the entry snapshot, and whether any value
// changed. A genuinely clean file keeps hashBytes(on-disk); a file that no longer
// matches the library body (or whose body cannot be loaded) gets the pristine
// library body hash, which forces MODIFIED.
func (s *service) honestSnapshots(ctx context.Context, row Adoption) ([]AdoptionFile, string, bool) {
	v, err := s.library.GetVersion(ctx, row.ComponentID, row.AdoptedVersion)
	if err != nil {
		return nil, "", false
	}
	targets := make(map[string]string, len(row.Files))
	for _, f := range row.Files {
		targets[moduleKey(firstNonEmpty(f.LibraryPath, filepath.Base(f.AdoptedPath)))] = f.AdoptedPath
	}
	out := make([]AdoptionFile, 0, len(row.Files))
	entrySnapshot := ""
	changed := false
	for _, f := range row.Files {
		snapshot := f.AdoptedSnapshotSHA256
		libFile, ok := libraryFileByPath(v, f.LibraryPath)
		onDisk, readErr := s.files.Read(ctx, row.Scenario, f.AdoptedPath)
		if ok && readErr == nil {
			expected := expectedAppliedBody(libFile, f.AdoptedPath, targets)
			if strings.TrimSpace(libFile.Content) != "" && stripSourceHeader(string(onDisk)) == expected {
				snapshot = adoptedSnapshotHash(string(onDisk))
			} else {
				snapshot = hashBytes([]byte(expected))
			}
		}
		if snapshot != f.AdoptedSnapshotSHA256 {
			changed = true
		}
		cf := f
		cf.AdoptedSnapshotSHA256 = snapshot
		out = append(out, cf)
		if f.AdoptedPath == row.AdoptedPath {
			entrySnapshot = snapshot
		}
	}
	if entrySnapshot == "" && len(out) > 0 {
		entrySnapshot = out[0].AdoptedSnapshotSHA256
	}
	return out, entrySnapshot, changed
}

func reconcileAdoptionFiles(group []ProvenanceFile, version components.ComponentVersion) ([]AdoptionFile, AdoptionFile, error) {
	// First pass: resolve each scanned file to its library file and build the
	// module→adopted-path map the import rewriter needs to reconstruct the exact
	// bytes an apply of this version would have written to the on-disk layout.
	type resolvedFile struct {
		scanned ProvenanceFile
		library components.ComponentVersionFile
	}
	resolved := make([]resolvedFile, 0, len(group))
	targets := make(map[string]string, len(group))
	for _, scanned := range group {
		libraryFile, ok := matchReconciledLibraryFile(version, scanned.AdoptedPath)
		if !ok {
			return nil, AdoptionFile{}, fmt.Errorf("no library file matches %s", scanned.AdoptedPath)
		}
		resolved = append(resolved, resolvedFile{scanned: scanned, library: libraryFile})
		targets[moduleKey(libraryFile.Path)] = scanned.AdoptedPath
	}

	files := make([]AdoptionFile, 0, len(group))
	var entry AdoptionFile
	for _, r := range resolved {
		snapshot, _ := reconciledSnapshot(r.scanned, r.library, targets)
		file := AdoptionFile{LibraryPath: r.library.Path, AdoptedPath: r.scanned.AdoptedPath, SourceSHA256: r.library.ContentSHA256, AdoptedSnapshotSHA256: snapshot}
		files = append(files, file)
		if r.library.IsEntry || entry.AdoptedPath == "" {
			entry = file
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].AdoptedPath < files[j].AdoptedPath })
	return files, entry, nil
}

// reconciledSnapshot computes the honest AdoptedSnapshotSHA256 for a pre-existing
// vendored file discovered by a provenance scan, and reports whether the copy is
// provably a clean copy of the referenced library version.
//
// The recorded snapshot is the baseline computeStatus later compares the on-disk
// file against. It MUST be derived from the library version — never blindly from
// the local file — or a copy that already carried local edits at backfill time
// would masquerade as CLEAN (the knw-1784076189438860926 incident: a modified
// data-table.tsx read CLEAN and was silently overwritten by reconverge).
//
// Drift is decided by comparing the local body against the exact bytes an apply
// of this version would have written (JSDoc/provenance header ignored, relative
// import specifiers rewritten to the unit's on-disk layout):
//   - a genuine clean copy → snapshot is the file's current on-disk bytes, so a
//     later edit is detected as drift from this confirmed-pristine baseline and
//     reconverge may safely fast-forward it;
//   - a locally modified copy, or one whose library body cannot be loaded, →
//     snapshot is the pristine library body hash, which never equals the on-disk
//     bytes (they carry a header and/or divergent content). computeStatus reports
//     MODIFIED immediately and reconverge refuses to overwrite it.
func reconciledSnapshot(scanned ProvenanceFile, libraryFile components.ComponentVersionFile, targets map[string]string) (string, bool) {
	expected := expectedAppliedBody(libraryFile, scanned.AdoptedPath, targets)
	local := stripSourceHeader(string(scanned.Content))
	if strings.TrimSpace(libraryFile.Content) != "" && local == expected {
		return adoptedSnapshotHash(string(scanned.Content)), true
	}
	return hashBytes([]byte(expected)), false
}

// expectedAppliedBody reconstructs, header-stripped, the exact body an apply of
// libraryFile would have written to adoptedPath: the library body with its
// JSDoc/provenance header removed and relative import specifiers rewritten to the
// unit's on-disk layout (targets maps a module basename to its adopted path).
func expectedAppliedBody(libraryFile components.ComponentVersionFile, adoptedPath string, targets map[string]string) string {
	return rewriteUnitImports(stripSourceHeader(libraryFile.Content), adoptedPath, targets)
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
		return components.ComponentVersionFile{Path: name, Content: version.Content, ContentSHA256: version.ContentSHA256, IsEntry: true}, true
	}
	return components.ComponentVersionFile{}, false
}

func reconcileFinding(file ProvenanceFile, detail string) ReconcileFinding {
	return ReconcileFinding{Scenario: file.Scenario, AdoptedPath: file.AdoptedPath, LibraryID: file.LibraryID, Version: file.Version, Detail: detail}
}

func provenancePathKey(scenario, path string) string {
	return scenario + "\x00" + filepath.ToSlash(path)
}
