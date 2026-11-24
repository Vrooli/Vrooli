package adoptions

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"react-component-library/internal/components"

	"github.com/google/uuid"
)

// BatchApply validates every root and its closure before writing anything.
// Shared dependency files are materialized once, while every root receives
// provenance rows for the files in its effective closure. The production
// repository commits those rows in one SQLite transaction; scenario files are
// restored from snapshots if either the write or transaction fails.
func (s *service) BatchApply(ctx context.Context, in BatchApplyInput) (BatchApplyResult, error) {
	if len(in.Items) == 0 {
		return BatchApplyResult{}, ErrInvalidAdoption{Field: "items", Reason: "at least one batch item is required"}
	}

	type preparedItem struct {
		input          BatchApplyItem
		root           components.Component
		version        components.ComponentVersion
		closure        components.ClosureReport
		plans          []adoptionPlan
		verdict        AdoptionVerdict
		adoptionID     string
		experiencePath string
		entrySnapshot  string
		writtenPath    string
		importSites    []string
	}
	type targetOwner struct {
		itemIndex int
		assetID   string
		version   string
		isRoot    bool
	}
	type uniqueFile struct {
		scenario string
		path     string
		file     adoptionUnitFile
		plan     adoptionPlan
		owner    int
		body     string
		written  string
	}

	prepared := make([]preparedItem, 0, len(in.Items))
	seenDependency := map[string]struct {
		rootID  string
		version string
	}{}
	shared := map[string]struct{}{}
	seenTargets := map[string]targetOwner{}
	targetControls := map[string]struct {
		replaceExisting  bool
		confirmOverwrite bool
	}{}
	uniqueFiles := map[string]*uniqueFile{}
	snapshots := map[string]batchFileSnapshot{}

	for index, raw := range in.Items {
		item := raw
		item.ComponentID = strings.TrimSpace(item.ComponentID)
		item.Scenario = strings.TrimSpace(item.Scenario)
		item.AdoptedPath = strings.TrimSpace(item.AdoptedPath)
		if item.ComponentID == "" || item.Scenario == "" || item.AdoptedPath == "" {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "component_id/scenario/adopted_path", Reason: "all are required"}
		}
		root, err := s.library.Get(ctx, item.ComponentID)
		if err != nil {
			return BatchApplyResult{}, err
		}
		versionName := strings.TrimSpace(item.Version)
		if versionName == "" {
			versionName = firstNonEmpty(root.LatestVersion, root.Version)
		}
		version, err := s.library.GetVersion(ctx, root.ID, versionName)
		if err != nil {
			return BatchApplyResult{}, err
		}
		closure, err := s.resolveAdoptionClosure(ctx, root, version, item.Scenario, item.IncludeSuggestions)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if _, err := s.requireTokenVerdict(ctx, root.ID, item.Scenario, closure, item.OverrideValidation); err != nil {
			return BatchApplyResult{}, err
		}
		verdict, err := s.adoptionVerdict(ctx, root, version, closure, item.Scenario)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if verdict.Blocking() && !item.OverrideValidation {
			return BatchApplyResult{}, readinessBlockedError(root.ID, item.Scenario, verdict)
		}
		plans := adoptionPlansForClosure(closure.Assets, item.AdoptedPath)
		if err := ensureDistinctAdoptionTargets(plans); err != nil {
			return BatchApplyResult{}, err
		}
		prepared = append(prepared, preparedItem{input: item, root: root, version: version, closure: closure, plans: plans, verdict: verdict, adoptionID: uuid.NewString()})
		current := &prepared[len(prepared)-1]
		if version.ExperienceContract != "" {
			current.experiencePath = cmpExperiencePath(root)
		}
		if err := s.captureBatchSnapshots(ctx, item.Scenario, current.experiencePath, plans, snapshots); err != nil {
			return BatchApplyResult{}, err
		}

		for _, plan := range plans {
			key := plan.Asset.ID
			if prior, ok := seenDependency[key]; ok {
				if prior.version != plan.Version.Version {
					return BatchApplyResult{}, ErrBatchDependencyConflict{Dependency: plan.Asset.LibraryID, FirstRoot: prior.rootID, FirstVersion: prior.version, SecondRoot: root.ID, SecondVersion: plan.Version.Version}
				}
				if prior.rootID != root.ID {
					shared[plan.Asset.LibraryID] = struct{}{}
				}
			} else {
				seenDependency[key] = struct {
					rootID  string
					version string
				}{rootID: root.ID, version: plan.Version.Version}
			}
			for _, file := range plan.Files {
				targetKey := batchSnapshotKey(item.Scenario, file.AdoptedPath)
				owner := targetOwner{itemIndex: index, assetID: plan.Asset.ID, version: plan.Version.Version, isRoot: plan.Asset.ID == root.ID}
				if prior, ok := seenTargets[targetKey]; ok {
					sharedAsset := prior.assetID == owner.assetID && prior.version == owner.version && !prior.isRoot && !owner.isRoot
					if !sharedAsset {
						return BatchApplyResult{}, fmt.Errorf("batch target collision at %q between %s and %s", file.AdoptedPath, prepared[prior.itemIndex].root.ID, root.ID)
					}
				} else {
					seenTargets[targetKey] = owner
					targetControls[targetKey] = struct {
						replaceExisting  bool
						confirmOverwrite bool
					}{replaceExisting: item.ReplaceExisting, confirmOverwrite: item.ConfirmOverwrite}
					uniqueFiles[targetKey] = &uniqueFile{scenario: item.Scenario, path: file.AdoptedPath, file: file, plan: plan, owner: index}
				}
				if prior, exists := seenTargets[targetKey]; exists && prior.itemIndex != index {
					controls := targetControls[targetKey]
					controls.replaceExisting = controls.replaceExisting && item.ReplaceExisting
					controls.confirmOverwrite = controls.confirmOverwrite && item.ConfirmOverwrite
					targetControls[targetKey] = controls
				}
			}
		}
	}

	// Validate existing targets once, using the strictest overwrite controls
	// from all roots that reference a shared target.
	for key := range seenTargets {
		file := uniqueFiles[key]
		if file == nil {
			continue
		}
		controls := targetControls[key]
		exists, err := s.files.Exists(ctx, file.scenario, file.path)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if !exists {
			continue
		}
		if !controls.replaceExisting {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "target file already exists; set replace_existing to replace it"}
		}
		existing, err := s.files.Read(ctx, file.scenario, file.path)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if hashBytes([]byte(stripSourceHeader(string(existing)))) != hashBytes([]byte(stripSourceHeader(file.file.Content))) && !controls.confirmOverwrite {
			return BatchApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing target differs from the ingested library source"}
		}
	}

	// Experience contracts are root-owned files, so validate them separately.
	for i := range prepared {
		if prepared[i].experiencePath == "" {
			continue
		}
		exists, err := s.files.Exists(ctx, prepared[i].input.Scenario, prepared[i].experiencePath)
		if err != nil {
			return BatchApplyResult{}, err
		}
		if !exists || prepared[i].input.ReplaceExisting {
			continue
		}
		return BatchApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "component experience contract already exists; set replace_existing to replace it"}
	}

	mappings := map[string]TokenMapping{}
	for i := range prepared {
		if _, ok := mappings[prepared[i].input.Scenario]; ok {
			continue
		}
		mapping, err := s.resolveTokenMapping(ctx, prepared[i].input.Scenario)
		if err != nil {
			return BatchApplyResult{}, err
		}
		mappings[prepared[i].input.Scenario] = mapping
	}

	keys := make([]string, 0, len(uniqueFiles))
	for key := range uniqueFiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		unique := uniqueFiles[key]
		item := &prepared[unique.owner]
		mapping := mappings[unique.scenario]
		translated, translations, err := TranslateDesignTokens(stripSourceHeader(unique.file.Content), mapping.Namespace, mapping)
		if err != nil {
			if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
				return BatchApplyResult{}, fmt.Errorf("batch translation failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return BatchApplyResult{}, err
		}
		now := s.clock.Now().UTC()
		body := formatProvenance(unique.plan.Version, item.adoptionID, now, hashBytes([]byte(translated)), formatTokenTranslations(translations)) + translated
		path, actual, err := s.writeAdoptedSource(ctx, unique.scenario, unique.path, []byte(body))
		if err != nil {
			if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
				return BatchApplyResult{}, fmt.Errorf("batch write failed: %w (rollback failed: %v)", err, rollbackErr)
			}
			return BatchApplyResult{}, err
		}
		unique.written, unique.body = path, actual
	}
	for i := range prepared {
		for _, plan := range prepared[i].plans {
			for _, file := range plan.Files {
				key := batchSnapshotKey(prepared[i].input.Scenario, file.AdoptedPath)
				unique := uniqueFiles[key]
				if unique == nil {
					continue
				}
				if file.IsEntry && plan.Asset.ID == prepared[i].root.ID {
					prepared[i].entrySnapshot = adoptedSnapshotHash(unique.body)
					prepared[i].writtenPath = unique.written
				}
				if finder, ok := s.files.(ScenarioImportSiteFinder); ok {
					sites, err := finder.FindImportSites(ctx, prepared[i].input.Scenario, file.AdoptedPath)
					if err != nil {
						if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
							return BatchApplyResult{}, fmt.Errorf("batch import-site scan failed: %w (rollback failed: %v)", err, rollbackErr)
						}
						return BatchApplyResult{}, err
					}
					prepared[i].importSites = append(prepared[i].importSites, sites...)
				}
			}
		}
		if prepared[i].experiencePath != "" {
			if _, err := s.files.Write(ctx, prepared[i].input.Scenario, prepared[i].experiencePath, []byte(prepared[i].version.ExperienceContract)); err != nil {
				if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
					return BatchApplyResult{}, fmt.Errorf("batch experience write failed: %w (rollback failed: %v)", err, rollbackErr)
				}
				return BatchApplyResult{}, fmt.Errorf("write component experience contract: %w", err)
			}
		}
	}

	inputs := make([]CreateInput, 0, len(prepared))
	for i := range prepared {
		adoptionFiles := make([]AdoptionFile, 0)
		for _, plan := range prepared[i].plans {
			for _, file := range plan.Files {
				unique := uniqueFiles[batchSnapshotKey(prepared[i].input.Scenario, file.AdoptedPath)]
				adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: adoptedSnapshotHash(unique.body), SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
			}
		}
		inputs = append(inputs, CreateInput{ID: prepared[i].adoptionID, ComponentID: prepared[i].root.ID, LibraryID: prepared[i].root.LibraryID, Scenario: prepared[i].input.Scenario, AdoptedPath: prepared[i].input.AdoptedPath, AdoptedVersion: prepared[i].version.Version, SourceSHA256: prepared[i].version.ContentSHA256, AdoptedSnapshotSHA256: prepared[i].entrySnapshot, IncludeSuggestions: append([]string(nil), prepared[i].input.IncludeSuggestions...), ForkReason: prepared[i].input.ForkReason, ExtensionPoints: append([]string(nil), prepared[i].input.ExtensionPoints...), Files: adoptionFiles})
	}
	var created []Adoption
	var err error
	if batchRepo, ok := s.repo.(BatchCreator); ok {
		created, err = batchRepo.CreateBatch(ctx, inputs)
	} else {
		created = make([]Adoption, 0, len(inputs))
		for _, input := range inputs {
			var adoption Adoption
			adoption, err = s.repo.Create(ctx, input)
			if err != nil {
				for _, prior := range created {
					_ = s.repo.Delete(ctx, prior.ID)
				}
				break
			}
			created = append(created, adoption)
		}
	}
	if err != nil {
		if rollbackErr := s.rollbackBatch(ctx, snapshots, nil); rollbackErr != nil {
			return BatchApplyResult{}, fmt.Errorf("batch persistence failed: %w (rollback failed: %v)", err, rollbackErr)
		}
		return BatchApplyResult{}, err
	}

	result := BatchApplyResult{Results: make([]ApplyResult, 0, len(prepared))}
	for i := range prepared {
		copied := make([]string, 0, len(prepared[i].closure.Assets))
		for _, asset := range prepared[i].closure.Assets {
			copied = append(copied, asset.Asset.LibraryID)
		}
		result.Results = append(result.Results, ApplyResult{Adoption: created[i], WrittenPath: prepared[i].writtenPath, ExperiencePath: prepared[i].experiencePath, ImportSites: prepared[i].importSites, StyleFitAffinity: components.DesignAffinity(prepared[i].verdict.StyleFit), StyleFitDetail: prepared[i].verdict.StyleFitDetail, CopiedAssets: copied, SatisfiedPorts: prepared[i].closure.SatisfiedPorts, AvailableSuggestions: prepared[i].closure.AvailableSuggestions})
	}
	for dependency := range shared {
		result.SharedDependencies = append(result.SharedDependencies, dependency)
	}
	sort.Strings(result.SharedDependencies)
	return result, nil
}

type batchFileSnapshot struct {
	scenario string
	path     string
	existed  bool
	content  []byte
}

func batchSnapshotKey(scenario, path string) string { return scenario + "::" + path }

func (s *service) captureBatchSnapshots(ctx context.Context, scenario, experiencePath string, plans []adoptionPlan, snapshots map[string]batchFileSnapshot) error {
	paths := make([]string, 0)
	for _, plan := range plans {
		for _, file := range plan.Files {
			paths = append(paths, file.AdoptedPath)
		}
	}
	if experiencePath != "" {
		paths = append(paths, experiencePath)
	}
	for _, path := range paths {
		key := batchSnapshotKey(scenario, path)
		if _, seen := snapshots[key]; seen {
			continue
		}
		exists, err := s.files.Exists(ctx, scenario, path)
		if err != nil {
			return err
		}
		snapshot := batchFileSnapshot{scenario: scenario, path: path, existed: exists}
		if exists {
			content, err := s.files.Read(ctx, scenario, path)
			if err != nil {
				return err
			}
			snapshot.content = append([]byte(nil), content...)
		}
		snapshots[key] = snapshot
	}
	return nil
}

func (s *service) rollbackBatch(ctx context.Context, snapshots map[string]batchFileSnapshot, results []ApplyResult) error {
	if len(results) == 0 && len(snapshots) == 0 {
		return nil
	}
	var firstErr error
	for _, result := range results {
		if err := s.repo.Delete(ctx, result.Adoption.ID); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	for _, snapshot := range snapshots {
		var err error
		if snapshot.existed {
			if raw, ok := s.files.(interface {
				WriteRaw(context.Context, string, string, []byte) error
			}); ok {
				err = raw.WriteRaw(ctx, snapshot.scenario, snapshot.path, snapshot.content)
			} else {
				_, err = s.files.Write(ctx, snapshot.scenario, snapshot.path, snapshot.content)
			}
		} else if remover, ok := s.files.(ScenarioFileRemover); ok {
			err = remover.Remove(ctx, snapshot.scenario, snapshot.path)
		} else {
			return fmt.Errorf("file writer cannot remove newly created batch files")
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func cmpExperiencePath(root components.Component) string {
	if root.Slug == "" {
		return ""
	}
	return componentExperiencePath(root.Slug)
}
