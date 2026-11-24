package adoptions

import (
	"context"
	"errors"
	"fmt"
	"react-component-library/internal/components"
	"strings"

	"github.com/google/uuid"
)

func (s *service) Create(ctx context.Context, in CreateInput) (Adoption, error) {
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.AdoptedPath = strings.TrimSpace(in.AdoptedPath)
	if in.ComponentID == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if in.Scenario == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if in.AdoptedPath == "" {
		return Adoption{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	// Validate the component exists and echo its libraryId. Snapshot
	// hash is best-effort: if the adopted file is unreadable at create
	// time we still record the row (operator can refresh later).
	cmp, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return Adoption{}, ErrInvalidAdoption{Field: "component_id", Reason: "no component with that id"}
		}
		return Adoption{}, fmt.Errorf("validate component %q: %w", in.ComponentID, err)
	}
	if in.LibraryID == "" {
		in.LibraryID = cmp.LibraryID
	}
	if in.AdoptedSnapshotSHA256 == "" {
		if bytes, err := s.files.Read(ctx, in.Scenario, in.AdoptedPath); err == nil {
			in.AdoptedSnapshotSHA256 = hashBytes(bytes)
		}
	}
	return s.repo.Create(ctx, in)
}

func (s *service) Apply(ctx context.Context, in ApplyInput) (ApplyResult, error) {
	explicitVersion := strings.TrimSpace(in.Version) != ""
	in.ComponentID = strings.TrimSpace(in.ComponentID)
	in.Scenario = strings.TrimSpace(in.Scenario)
	in.AdoptedPath = strings.TrimSpace(in.AdoptedPath)
	if in.ComponentID == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if in.Scenario == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if in.AdoptedPath == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	cmp, err := s.library.Get(ctx, in.ComponentID)
	if err != nil {
		if errors.As(err, &components.ErrComponentNotFound{}) {
			return ApplyResult{}, ErrInvalidAdoption{Field: "component_id", Reason: "no component with that id"}
		}
		return ApplyResult{}, err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = cmp.LatestVersion
	}
	if version == "" {
		version = cmp.Version
	}
	if version == "" {
		return ApplyResult{}, ErrInvalidAdoption{Field: "version", Reason: "component has no latest version"}
	}
	if err := s.ensureVersionMaterialized(ctx, cmp.ID, version); err != nil {
		return ApplyResult{}, err
	}
	exists, err := s.files.Exists(ctx, in.Scenario, in.AdoptedPath)
	if err != nil {
		return ApplyResult{}, err
	}
	v, err := s.library.GetVersion(ctx, in.ComponentID, version)
	if err != nil {
		return ApplyResult{}, err
	}
	_ = exists // entry existence is checked together with every unit target below.
	closure, err := s.resolveAdoptionClosure(ctx, cmp, v, in.Scenario, in.IncludeSuggestions)
	if err != nil {
		return ApplyResult{}, err
	}
	if _, err := s.requireTokenVerdict(ctx, in.ComponentID, in.Scenario, closure, in.OverrideValidation); err != nil {
		return ApplyResult{}, err
	}
	verdict, err := s.adoptionVerdict(ctx, cmp, v, closure, in.Scenario)
	if err != nil {
		return ApplyResult{}, err
	}
	if verdict.Blocking() && (!in.OverrideValidation || (v.Status == components.VersionStatusDraft && !explicitVersion)) {
		return ApplyResult{}, readinessBlockedError(in.ComponentID, in.Scenario, verdict)
	}
	plans := adoptionPlansForClosure(closure.Assets, in.AdoptedPath)
	if err := ensureDistinctAdoptionTargets(plans); err != nil {
		return ApplyResult{}, err
	}
	var importSites []string
	now := s.clock.Now().UTC()
	for _, plan := range plans {
		for _, file := range plan.Files {
			exists, err := s.files.Exists(ctx, in.Scenario, file.AdoptedPath)
			if err != nil {
				return ApplyResult{}, err
			}
			if !exists {
				continue
			}
			if !in.ReplaceExisting {
				return ApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "target file already exists; set replace_existing to replace it"}
			}
			existing, err := s.files.Read(ctx, in.Scenario, file.AdoptedPath)
			if err != nil {
				return ApplyResult{}, err
			}
			if hashBytes([]byte(stripSourceHeader(string(existing)))) != hashBytes([]byte(stripSourceHeader(file.Content))) && !in.ConfirmOverwrite {
				return ApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing target differs from the ingested library source"}
			}
		}
	}

	// One adoption represents the operator's root apply. Dependency assets are
	// materialized as files attributed to that root, not independent direct
	// adoptions. This preserves the parent relationship required to distinguish
	// direct component usage from mediated hook usage.
	adoptionID := uuid.NewString()
	adoptionFiles := make([]AdoptionFile, 0)
	written := ""
	entrySnapshot := ""
	tokenMapping, err := s.resolveTokenMapping(ctx, in.Scenario)
	if err != nil {
		return ApplyResult{}, err
	}
	for _, plan := range plans {
		for _, file := range plan.Files {
			fv := plan.Version
			fv.Content, fv.ContentSHA256 = file.Content, file.ContentSHA256
			translated, translations, err := TranslateDesignTokens(stripSourceHeader(file.Content), tokenMapping.Namespace, tokenMapping)
			if err != nil {
				return ApplyResult{}, err
			}
			translationNote := formatTokenTranslations(translations)
			body := formatProvenance(fv, adoptionID, now, hashBytes([]byte(translated)), translationNote) + translated
			path, formattedBody, err := s.writeAdoptedSource(ctx, in.Scenario, file.AdoptedPath, []byte(body))
			if err != nil {
				return ApplyResult{}, err
			}
			body = formattedBody
			if file.IsEntry && plan.Asset.ID == cmp.ID {
				entrySnapshot = adoptedSnapshotHash(body)
				if plan.Asset.ID == cmp.ID {
					written = path
				}
			}
			adoptionFiles = append(adoptionFiles, AdoptionFile{LibraryPath: file.Path, AdoptedPath: file.AdoptedPath, SourceSHA256: file.ContentSHA256, AdoptedSnapshotSHA256: adoptedSnapshotHash(body), SourceAssetID: plan.Asset.ID, SourceLibraryID: plan.Asset.LibraryID, SourceVersion: plan.Version.Version})
			if finder, ok := s.files.(ScenarioImportSiteFinder); ok {
				sites, err := finder.FindImportSites(ctx, in.Scenario, file.AdoptedPath)
				if err != nil {
					return ApplyResult{}, err
				}
				importSites = append(importSites, sites...)
			}
		}
	}
	experiencePath := ""
	if strings.TrimSpace(v.ExperienceContract) != "" {
		experiencePath = componentExperiencePath(cmp.Slug)
		experienceExists, err := s.files.Exists(ctx, in.Scenario, experiencePath)
		if err != nil {
			return ApplyResult{}, err
		}
		if experienceExists {
			if !in.ReplaceExisting {
				return ApplyResult{}, ErrInvalidAdoption{Field: "replace_existing", Reason: "component experience contract already exists; set replace_existing to replace it"}
			}
			existing, err := s.files.Read(ctx, in.Scenario, experiencePath)
			if err != nil {
				return ApplyResult{}, err
			}
			if string(existing) != v.ExperienceContract && !in.ConfirmOverwrite {
				return ApplyResult{}, ErrInvalidAdoption{Field: "confirm_overwrite", Reason: "existing component experience contract differs from the catalog source"}
			}
		}
		if _, err := s.files.Write(ctx, in.Scenario, experiencePath, []byte(v.ExperienceContract)); err != nil {
			return ApplyResult{}, fmt.Errorf("write component experience contract: %w", err)
		}
	}
	root, err := s.repo.Create(ctx, CreateInput{ID: adoptionID, ComponentID: cmp.ID, LibraryID: cmp.LibraryID, Scenario: in.Scenario, AdoptedPath: in.AdoptedPath, AdoptedVersion: version, SourceSHA256: v.ContentSHA256, AdoptedSnapshotSHA256: entrySnapshot, IncludeSuggestions: append([]string(nil), in.IncludeSuggestions...), ForkReason: in.ForkReason, ExtensionPoints: append([]string(nil), in.ExtensionPoints...), Files: adoptionFiles})
	if err != nil {
		return ApplyResult{}, err
	}
	copied := make([]string, 0, len(closure.Assets))
	for _, asset := range closure.Assets {
		copied = append(copied, asset.Asset.LibraryID)
	}
	result := ApplyResult{Adoption: root, WrittenPath: written, ExperiencePath: experiencePath, ImportSites: importSites, CopiedAssets: copied, SatisfiedPorts: closure.SatisfiedPorts, AvailableSuggestions: closure.AvailableSuggestions}
	result.StyleFitAffinity = components.DesignAffinity(verdict.StyleFit)
	result.StyleFitDetail = verdict.StyleFitDetail
	return result, nil
}

func (s *service) Preflight(ctx context.Context, in PreflightInput) (PreflightResult, error) {
	componentID := strings.TrimSpace(in.ComponentID)
	scenario := strings.TrimSpace(in.Scenario)
	if componentID == "" || scenario == "" {
		return PreflightResult{}, ErrInvalidAdoption{Field: "component_id/scenario", Reason: "required"}
	}
	root, err := s.library.Get(ctx, componentID)
	if err != nil {
		return PreflightResult{}, err
	}
	version := strings.TrimSpace(in.Version)
	if version == "" {
		version = firstNonEmpty(root.LatestVersion, root.Version)
	}
	if err := s.ensureVersionMaterialized(ctx, componentID, version); err != nil {
		return PreflightResult{}, err
	}
	v, err := s.library.GetVersion(ctx, componentID, version)
	if err != nil {
		return PreflightResult{}, err
	}
	closure, err := s.resolveAdoptionClosure(ctx, root, v, scenario, nil)
	if err != nil {
		return PreflightResult{}, err
	}
	tokens, err := s.resolveTokenVerdict(ctx, closure, scenario)
	if err != nil {
		return PreflightResult{}, err
	}
	readiness, err := s.adoptionVerdict(ctx, root, v, closure, scenario)
	if err != nil {
		return PreflightResult{}, err
	}
	result := PreflightResult{ComponentID: componentID, CatalogAssetID: root.CatalogID, CatalogLatestVersion: root.LatestVersion, Scenario: scenario, Version: version, Verdict: readiness, Tokens: tokens, Dependency: readiness.Dependency, StyleFit: readiness.StyleFit, I18n: readiness.I18n, Selectors: readiness.Selectors, Blocking: readiness.Blocking()}
	return result, nil
}

func (s *service) adoptionVerdict(ctx context.Context, root components.Component, version components.ComponentVersion, closure components.ClosureReport, scenario string) (AdoptionVerdict, error) {
	// A nil coverage reader is a deliberately reduced unit-test seam. Production
	// wiring always installs CatalogGateReader, which returns not-measured when
	// evidence is absent and therefore keeps the adoption blocking contract.
	result := AdoptionVerdict{Version: version.Status, I18n: "pass", Selectors: "pass"}
	if s.coverage != nil {
		result.I18n = "not-measured"
		result.Selectors = "not-measured"
	}
	if s.deps != nil {
		verdict, err := s.deps.ValidateAdoption(ctx, root.ID, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Dependency = string(verdict.Kind)
	}
	if s.styles != nil {
		verdict, err := s.styles.ValidateStyleFit(ctx, root.ID, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.StyleFit = string(verdict.Affinity)
		result.StyleFitDetail = verdict.Detail
	}
	if s.maturity != nil {
		maturity, err := s.maturity.Maturity(ctx, root, version.Version, scenario)
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Maturity = maturity
	}
	var err error
	if s.coverage != nil {
		result.I18n, err = s.coverage.GateVerdict(ctx, root, version.Version, scenario, "i18n")
		if err != nil {
			return AdoptionVerdict{}, err
		}
		result.Selectors, err = s.coverage.GateVerdict(ctx, root, version.Version, scenario, "selector-coverage")
		if err != nil {
			return AdoptionVerdict{}, err
		}
	}
	result.Tokens, err = s.resolveTokenVerdict(ctx, closure, scenario)
	if err != nil {
		return AdoptionVerdict{}, err
	}
	if version.Status == components.VersionStatusDeprecated {
		result.Warnings = []string{"selected version is deprecated"}
	}
	return result, nil
}
