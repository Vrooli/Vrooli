package adoptions

import (
	"context"
	"fmt"
	"path/filepath"
	"react-component-library/internal/components"
	"regexp"
	"strings"
)

type adoptionPlan struct {
	Asset       components.Component
	Version     components.ComponentVersion
	EntryTarget string
	Files       []adoptionUnitFile
}

func (s *service) resolveAdoptionClosure(ctx context.Context, root components.Component, version components.ComponentVersion, scenario string, includeSuggestions []string) (components.ClosureReport, error) {
	if len(root.Dependencies) == 0 {
		return components.ClosureReport{Assets: []components.ResolvedAsset{{Asset: root, Version: version}}}, nil
	}
	reader, ok := any(s.library).(components.DependencyReader)
	if !ok {
		return components.ClosureReport{}, fmt.Errorf("asset dependency reader is not configured")
	}
	var provides []string
	if s.manifestLoader != nil {
		manifest, err := s.manifestLoader.Load(scenario)
		if err != nil {
			return components.ClosureReport{}, fmt.Errorf("load target UI manifest: %w", err)
		}
		provides = manifest.Provides
	}
	return components.ResolveDependencyClosureReport(ctx, reader, root.ID, version.Version, includeSuggestions, provides, nil)
}

func adoptionPlansForClosure(closure []components.ResolvedAsset, rootTarget string) []adoptionPlan {
	targets := make(map[string]string, len(closure)*2)
	for _, resolved := range closure {
		target := rootTarget
		if resolved.Asset.ID != closure[len(closure)-1].Asset.ID {
			target = dependencyEntryTarget(rootTarget, resolved.Asset, resolved.Version)
		}
		targets[moduleKey(resolved.Version.SourcePath)] = target
		targets[moduleKey(resolved.Asset.DisplayName)] = target
	}
	plans := make([]adoptionPlan, 0, len(closure))
	for _, resolved := range closure {
		target := rootTarget
		if resolved.Asset.ID != closure[len(closure)-1].Asset.ID {
			target = dependencyEntryTarget(rootTarget, resolved.Asset, resolved.Version)
		}
		files := adoptionUnitFiles(resolved.Version, target)
		for i := range files {
			files[i].Content = rewriteUnitImports(files[i].Content, files[i].AdoptedPath, targets)
		}
		plans = append(plans, adoptionPlan{Asset: resolved.Asset, Version: resolved.Version, EntryTarget: target, Files: files})
	}
	return plans
}

func dependencyEntryTarget(rootTarget string, asset components.Component, version components.ComponentVersion) string {
	dir := filepath.ToSlash(filepath.Dir(rootTarget))
	if asset.AssetKind == components.AssetKindHook {
		if strings.HasSuffix(dir, "/components") || dir == "components" {
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
		dir = filepath.ToSlash(filepath.Join(dir, "hooks"))
	} else if asset.AssetKind == components.AssetKindFoundation {
		if strings.HasSuffix(dir, "/components") || dir == "components" {
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
		dir = filepath.ToSlash(filepath.Join(dir, "foundations"))
	} else if strings.HasSuffix(dir, "/components") || dir == "components" {
		dir = filepath.ToSlash(filepath.Dir(dir))
		dir = filepath.ToSlash(filepath.Join(dir, "components"))
	}
	return filepath.ToSlash(filepath.Join(dir, filepath.Base(version.SourcePath)))
}

func ensureDistinctAdoptionTargets(plans []adoptionPlan) error {
	seen := map[string]string{}
	for _, plan := range plans {
		for _, file := range plan.Files {
			if owner, exists := seen[file.AdoptedPath]; exists && owner != plan.Asset.LibraryID {
				return fmt.Errorf("asset dependency target collision at %q between %s and %s", file.AdoptedPath, owner, plan.Asset.LibraryID)
			}
			seen[file.AdoptedPath] = plan.Asset.LibraryID
		}
	}
	return nil
}

type adoptionUnitFile struct {
	components.ComponentVersionFile
	AdoptedPath string
}

func adoptionUnitFiles(v components.ComponentVersion, entryTarget string) []adoptionUnitFile {
	files := append([]components.ComponentVersionFile(nil), v.Files...)
	if len(files) == 0 {
		files = []components.ComponentVersionFile{{Path: filepath.Base(v.SourcePath), Content: v.Content, ContentSHA256: v.ContentSHA256, IsEntry: true}}
	}
	if files[0].Path == "" {
		files[0].Path = filepath.Base(entryTarget)
	}
	out := make([]adoptionUnitFile, 0, len(files))
	targets := make(map[string]string, len(files))
	for _, file := range files {
		// Story metadata belongs to the library catalog, not the adopted
		// runtime closure. Dependency stories otherwise collide at the shared
		// target and make a valid Button adoption impossible.
		if file.Path == "story.json" || file.Path == "story.tsx" || file.Path == "experience-contract.json" {
			continue
		}
		target := filepath.ToSlash(filepath.Join(filepath.Dir(entryTarget), file.Path))
		if file.IsEntry {
			target = entryTarget
		} else if isHookCompanion(file.Path, entryTarget) {
			target = filepath.ToSlash(filepath.Join(filepath.Dir(filepath.Dir(entryTarget)), "hooks", file.Path))
		}
		targets[moduleKey(file.Path)] = target
		out = append(out, adoptionUnitFile{ComponentVersionFile: file, AdoptedPath: target})
	}
	for i := range out {
		out[i].Content = rewriteUnitImports(out[i].Content, out[i].AdoptedPath, targets)
	}
	return out
}

func isHookCompanion(path, entryTarget string) bool {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	entryDir := filepath.ToSlash(filepath.Dir(entryTarget))
	return strings.HasPrefix(name, "use") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' &&
		(strings.HasSuffix(entryDir, "/components") || entryDir == "components")
}

func moduleKey(path string) string {
	base := filepath.ToSlash(filepath.Base(path))
	return strings.TrimSuffix(base, filepath.Ext(base))
}

var unitImportRE = regexp.MustCompile(`["']\.[^"']+["']`)

func rewriteUnitImports(body, adoptedPath string, targets map[string]string) string {
	return unitImportRE.ReplaceAllStringFunc(body, func(match string) string {
		quote, specifier := match[:1], match[1:len(match)-1]
		target, ok := targets[moduleKey(specifier)]
		if !ok {
			return match
		}
		rel, err := filepath.Rel(filepath.Dir(adoptedPath), target)
		if err != nil {
			return match
		}
		module := strings.TrimSuffix(filepath.ToSlash(rel), filepath.Ext(rel))
		if !strings.HasPrefix(module, ".") {
			module = "./" + module
		}
		return quote + module + quote
	})
}

func (s *service) Reapply(ctx context.Context, in ReapplyInput) (Adoption, string, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return Adoption{}, "", ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return Adoption{}, "", err
	}
	_, localStatus, _ := s.computeStatus(ctx, row)
	if localStatus == LocalStatusModified && !in.ConfirmLocalOverwrite && !in.DryRun {
		return Adoption{}, "", ErrInvalidAdoption{Field: "confirm_local_overwrite", Reason: "adopted file has local modifications"}
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		cmp, err := s.library.Get(ctx, row.ComponentID)
		if err != nil {
			return Adoption{}, "", err
		}
		version = firstNonEmpty(cmp.LatestVersion, cmp.Version, row.AdoptedVersion)
	}
	if err := s.ensureVersionMaterialized(ctx, row.ComponentID, version); err != nil {
		return Adoption{}, "", err
	}
	root, err := s.library.Get(ctx, row.ComponentID)
	if err != nil {
		return Adoption{}, "", err
	}
	v, err := s.library.GetVersion(ctx, row.ComponentID, version)
	if err != nil {
		return Adoption{}, "", err
	}
	now := s.clock.Now().UTC()
	closure, err := s.resolveAdoptionClosure(ctx, root, v, row.Scenario, row.IncludeSuggestions)
	if err != nil {
		return Adoption{}, "", err
	}
	if _, err := s.requireTokenVerdict(ctx, row.ComponentID, row.Scenario, closure, in.OverrideValidation); err != nil {
		return Adoption{}, "", err
	}
	readiness, err := s.adoptionVerdict(ctx, root, v, closure, row.Scenario)
	if err != nil {
		return Adoption{}, "", err
	}
	if readiness.Blocking() && !in.OverrideValidation {
		return Adoption{}, "", readinessBlockedError(row.ComponentID, row.Scenario, readiness)
	}
	plans := adoptionPlansForClosure(closure.Assets, row.AdoptedPath)
	if in.DryRun {
		if _, err := s.resolveTokenMapping(ctx, row.Scenario); err != nil {
			return Adoption{}, "", err
		}
		return row, "", nil
	}
	newPaths := make(map[string]struct{})
	for _, plan := range plans {
		for _, file := range plan.Files {
			newPaths[file.AdoptedPath] = struct{}{}
		}
	}
	adoptionFiles := make([]AdoptionFile, 0)
	written, entrySnapshot := "", ""
	tokenMapping, err := s.resolveTokenMapping(ctx, row.Scenario)
	if err != nil {
		return Adoption{}, "", err
	}
	for _, plan := range plans {
		for _, file := range plan.Files {
			fv := plan.Version
			fv.Content, fv.ContentSHA256 = file.Content, file.ContentSHA256
			translated, translations, err := TranslateDesignTokens(stripSourceHeader(file.Content), tokenMapping.Namespace, tokenMapping)
			if err != nil {
				return Adoption{}, "", err
			}
			body := formatProvenance(fv, row.ID, now, hashBytes([]byte(translated)), formatTokenTranslations(translations)) + translated
			path, formattedBody, err := s.writeAdoptedSource(ctx, row.Scenario, file.AdoptedPath, []byte(body))
			if err != nil {
				return Adoption{}, "", err
			}
			body = formattedBody
			snapshot := adoptedSnapshotHash(body)
			if file.IsEntry && plan.Asset.ID == row.ComponentID {
				written, entrySnapshot = path, snapshot
			}
			adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: snapshot, SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
		}
	}
	if strings.TrimSpace(v.ExperienceContract) != "" {
		experiencePath := componentExperiencePath(root.Slug)
		if _, err := s.files.Write(ctx, row.Scenario, experiencePath, []byte(v.ExperienceContract)); err != nil {
			return Adoption{}, "", fmt.Errorf("write component experience contract: %w", err)
		}
	}
	updated, err := s.repo.UpdateAppliedUnit(ctx, AppliedUnitUpdate{AppliedSnapshotUpdate: AppliedSnapshotUpdate{
		ID:                    row.ID,
		AdoptedVersion:        version,
		SourceSHA256:          v.ContentSHA256,
		AdoptedSnapshotSHA256: entrySnapshot,
		AppliedAt:             now,
	}, Files: adoptionFiles})
	if err != nil {
		return Adoption{}, "", err
	}
	// Remove orphaned files only after the replacement unit and its registry
	// row have been written successfully. Other live adoptions retain shared
	// files; failed writes never delete the previous unit.
	if remover, ok := s.files.(ScenarioFileRemover); ok {
		ownedByOther := make(map[string]struct{})
		if rows, listErr := s.repo.List(ctx, ListQuery{Scenario: row.Scenario, Limit: 100000}); listErr == nil {
			for _, other := range rows {
				if other.ID == row.ID {
					continue
				}
				for _, file := range other.Files {
					ownedByOther[file.AdoptedPath] = struct{}{}
				}
			}
		}
		for _, oldFile := range row.Files {
			if _, stillPresent := newPaths[oldFile.AdoptedPath]; stillPresent {
				continue
			}
			if _, shared := ownedByOther[oldFile.AdoptedPath]; shared {
				continue
			}
			if err := remover.Remove(ctx, row.Scenario, oldFile.AdoptedPath); err != nil {
				return updated, written, fmt.Errorf("remove orphaned adopted file %q: %w", oldFile.AdoptedPath, err)
			}
		}
	}
	return updated, written, nil
}

// componentExperiencePath is the single canonical location for a copied
// component-scope experience contract. Catalog slugs retain their display
// casing for library source paths, while experience ids and document names
// are stable kebab-case identifiers.
func componentExperiencePath(slug string) string {
	return filepath.ToSlash(filepath.Join("experience", "components", toKebab(slug)+".json"))
}

// writeAdoptedSource returns the exact bytes that the target scenario now
// owns. The production filesystem writer formats through the target UI's
// local toolchain, so snapshots must be derived from the post-format bytes;
// otherwise a successful reapply immediately reports itself as modified.
func (s *service) writeAdoptedSource(ctx context.Context, scenario, adoptedPath string, content []byte) (string, string, error) {
	path, err := s.files.Write(ctx, scenario, adoptedPath, content)
	if err != nil {
		return "", "", err
	}
	actual, err := s.files.Read(ctx, scenario, adoptedPath)
	if err != nil {
		return "", "", fmt.Errorf("read formatted adopted file %q: %w", adoptedPath, err)
	}
	return path, string(actual), nil
}

func (s *service) resolveTokenMapping(ctx context.Context, scenario string) (TokenMapping, error) {
	if s.mappings != nil {
		mapping, err := s.mappings.TokenMapping(ctx, scenario)
		if err != nil {
			return TokenMapping{}, err
		}
		return mapping, nil
	}
	// Unit-test services intentionally omit filesystem wiring. The production
	// handler installs FSScenarioFileReader through SetTokenNamespaceReader,
	// where a missing scenario-owned file is a hard error.
	return TokenMapping{Namespace: "app"}, nil
}
