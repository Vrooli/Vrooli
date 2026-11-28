package components

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type manifestFile struct {
	CatalogID   string `json:"catalogId"`
	LibraryID   string `json:"libraryId"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Slot        string `json:"slot"`
	// Category is catalog metadata, not source-header metadata. Keeping it in
	// the manifest makes it stable across versions and available to list views.
	Category string `json:"category"`
	// AssetKind is optional for backwards compatibility: the authored root is
	// authoritative for legacy manifests. New manifests may state it explicitly
	// as a guard against moving a manifest into the wrong root.
	AssetKind string `json:"assetKind"`
	Entry     string `json:"entry"`
	// FileSlots pins an explicit placement slot per version-unit basename
	// (e.g. {"useFocusTrap.ts": "hook"}). Authoritative over the resolver's
	// extension heuristic. Applies across all versions of the component.
	FileSlots          map[string]string        `json:"fileSlots"`
	Tags               []string                 `json:"tags"`
	DesignStyles       []manifestDesignAffinity `json:"designStyles"`
	Latest             string                   `json:"latest"`
	Draft              string                   `json:"draft"`
	DeprecatedVersions []string                 `json:"deprecatedVersions"`
	EvictedVersions    []string                 `json:"evictedVersions"`
}

type manifestDesignAffinity struct {
	StyleID  string `json:"styleId"`
	Affinity string `json:"affinity"`
	Reason   string `json:"reason"`
}

func (idx *Indexer) buildManifestInput(path string) (IndexManifestInput, map[string]string, error) {
	raw, err := fs.ReadFile(idx.fs, path)
	if err != nil {
		return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", path, err)
	}
	var mf manifestFile
	if err := json.Unmarshal(raw, &mf); err != nil {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "component.json", Reason: err.Error()}
	}
	slug := filepath.Base(filepath.Dir(path))
	assetKind, err := assetKindForManifestPath(path, mf.AssetKind)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	if metadata, metadataErr := catalogAssetMetadataFor(idx.root, mf.CatalogID); metadataErr != nil {
		return IndexManifestInput{}, nil, metadataErr
	} else if metadata != nil {
		if mf.DisplayName == "" {
			mf.DisplayName = metadata.Name
		}
		if mf.Description == "" {
			mf.Description = metadata.Description
		}
		if mf.Slot == "" && assetKind == AssetKindComponent {
			mf.Slot = catalogComponentSlot(metadata.Slot)
		}
		if mf.Category == "" {
			mf.Category = metadata.Domain
		}
	}
	slot, err := normalizeComponentSlot(mf.Slot)
	if err != nil && assetKind == AssetKindComponent {
		return IndexManifestInput{}, nil, errInvalidHeader(path, "slot", err.Error())
	}
	if assetKind == AssetKindHook && strings.TrimSpace(mf.Slot) != "" {
		return IndexManifestInput{}, nil, errInvalidHeader(path, "slot", "hooks are non-renderable and must not declare a UI slot")
	}
	manifest := ComponentManifest{
		CatalogID:          strings.TrimSpace(mf.CatalogID),
		LibraryID:          strings.TrimSpace(mf.LibraryID),
		Slug:               slug,
		DisplayName:        strings.TrimSpace(mf.DisplayName),
		Description:        strings.TrimSpace(mf.Description),
		Slot:               string(slot),
		Category:           strings.TrimSpace(mf.Category),
		ManifestPath:       filepath.ToSlash(path),
		LatestVersion:      strings.TrimSpace(mf.Latest),
		DraftVersion:       strings.TrimSpace(mf.Draft),
		DeprecatedVersions: append([]string(nil), mf.DeprecatedVersions...),
		EvictedVersions:    append([]string(nil), mf.EvictedVersions...),
		Tags:               append([]string(nil), mf.Tags...),
		AssetKind:          assetKind,
	}
	designStyles, err := parseManifestDesignStyles(path, mf.DesignStyles)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	staleFindings, err := staleDesignStyleFindings(path, designStyles)
	if err != nil {
		return IndexManifestInput{}, nil, err
	}
	manifest.DesignStyles = designStyles
	if manifest.LibraryID == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "libraryId", Reason: "required"}
	}
	if manifest.DisplayName == "" {
		manifest.DisplayName = slug
	}
	// An asset created as a draft has no release until that draft is
	// published; its draft stands in for the latest version until then.
	unreleased := manifest.LatestVersion == ""
	if unreleased && manifest.DraftVersion == "" {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "required unless an initial draft is active"}
	}
	currentVersion := manifest.LatestVersion
	if unreleased {
		currentVersion = manifest.DraftVersion
	}
	versionRoot := filepath.ToSlash(filepath.Join(filepath.Dir(path), "versions"))
	entries, err := fs.ReadDir(idx.fs, versionRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && len(manifest.EvictedVersions) > 0 {
			entries = nil
		} else {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "versions", Reason: err.Error()}
		}
	}
	evicted := map[string]struct{}{}
	for _, rawVersion := range manifest.EvictedVersions {
		version := strings.TrimSpace(rawVersion)
		if version == "" {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "evictedVersions", Reason: "versions must not be empty"}
		}
		if _, exists := evicted[version]; exists {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "evictedVersions", Reason: "duplicate version " + version}
		}
		evicted[version] = struct{}{}
	}
	deprecated := map[string]bool{}
	for _, v := range manifest.DeprecatedVersions {
		deprecated[strings.TrimSpace(v)] = true
	}
	var versions []ComponentVersion
	var stories []ComponentStory
	findings := append([]IndexFinding(nil), staleFindings...)
	var warnings []string
	// A promoted component with no declared affinities is catalog-incomplete:
	// its detail view reads "No design affinities declared" while authored
	// peers carry 2-3. Surface it as a soft conformance finding (never a hard
	// error) so the gap is visible on every reindex without blocking the walk.
	if len(manifest.DesignStyles) == 0 {
		findings = append(findings, missingDesignAffinityFinding(path, manifest.LibraryID))
	}
	var latestFound bool
	var draftFound bool
	latestHeaders := map[string]string{"libraryId": manifest.LibraryID}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := strings.TrimSpace(entry.Name())
		versionPath := filepath.ToSlash(filepath.Join(versionRoot, version))
		if _, declaredEvicted := evicted[version]; declaredEvicted {
			// Reconciliation needs a component-version row to repair a stale
			// presence tier. Rejecting the asset here made the state self-locking:
			// the reconciler could never observe the materialized bytes that prove
			// the evicted declaration stale.
			findings = append(findings, headerDisagreementFinding(
				versionPath,
				"evictedVersions",
				"version absent from the authored tree",
				version,
				"materialized source is indexed so presence reconciliation can repair the stale tier",
			))
		}
		files, err := fs.ReadDir(idx.fs, versionPath)
		if err != nil {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: err.Error()}
		}
		var sourceFiles []fs.DirEntry
		for _, f := range files {
			if f.IsDir() {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: "subdirectories are not supported"}
			}
			// story.tsx is a preview-only artifact. It is validated with the
			// story contract but is never an asset companion or adoption input.
			if f.Name() == "story.tsx" {
				continue
			}
			if strings.HasSuffix(f.Name(), ".tsx") || strings.HasSuffix(f.Name(), ".ts") || strings.HasSuffix(f.Name(), ".css") {
				sourceFiles = append(sourceFiles, f)
			}
		}
		// Evicted versions may retain an empty placeholder directory after
		// their source has moved to cold storage. It is not materialized source
		// and must not invalidate the live manifest; materialized evicted
		// versions still proceed through the normal reconciliation path above.
		if _, declaredEvicted := evicted[version]; declaredEvicted && len(sourceFiles) == 0 {
			continue
		}
		entryName := strings.TrimSpace(mf.Entry)
		if entryName != "" && (filepath.Base(entryName) != entryName || !validEntryExtension(entryName, assetKind)) {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "entry", Reason: entryFileRequirement(assetKind)}
		}
		if entryName == "" {
			var entries []string
			for _, f := range sourceFiles {
				if validEntryExtension(f.Name(), assetKind) && !isTestArtifact(f.Name()) {
					entries = append(entries, f.Name())
				}
			}
			if len(entries) != 1 {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "version", Reason: "expected exactly one " + entryFileRequirement(assetKind) + " or component.json entry"}
			}
			entryName = entries[0]
		}
		var entryFound bool
		var versionFiles []ComponentVersionFile
		for _, f := range sourceFiles {
			filePath := filepath.ToSlash(filepath.Join(versionPath, f.Name()))
			body, err := fs.ReadFile(idx.fs, filePath)
			if err != nil {
				return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", filePath, err)
			}
			isEntry := f.Name() == entryName
			if isEntry {
				entryFound = true
			}
			versionFiles = append(versionFiles, ComponentVersionFile{Path: f.Name(), Content: string(body), ContentSHA256: digestBytes(body), IsEntry: isEntry, Slot: strings.TrimSpace(mf.FileSlots[f.Name()])})
		}
		if !entryFound {
			return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: versionPath, Field: "entry", Reason: "entry file not found"}
		}
		dependencies, lockFile, err := idx.readVersionDependencyLock(versionPath, manifest.LibraryID, version)
		if err != nil {
			return IndexManifestInput{}, nil, err
		}
		if lockFile != nil {
			versionFiles = append(versionFiles, *lockFile)
		}
		experienceContract := ""
		contractPath := filepath.ToSlash(filepath.Join(versionPath, "experience-contract.json"))
		if rawContract, contractErr := fs.ReadFile(idx.fs, contractPath); contractErr == nil {
			experienceContract = string(rawContract)
			versionFiles = append(versionFiles, ComponentVersionFile{Path: "experience-contract.json", Content: experienceContract, ContentSHA256: digestBytes([]byte(experienceContract))})
		} else if !errors.Is(contractErr, fs.ErrNotExist) {
			return IndexManifestInput{}, nil, fmt.Errorf("read experience contract %s: %w", contractPath, contractErr)
		}
		if experienceContract == "" {
			if repoRoot := repositoryRootFromLibrary(idx.root); repoRoot != "" {
				canonical := filepath.Join(repoRoot, "scenarios", "react-component-library", "experience", "components", kebabAssetName(slug)+".json")
				if rawContract, contractErr := os.ReadFile(canonical); contractErr == nil {
					experienceContract = string(rawContract)
				}
			}
		}
		// story.json is the declarative preview/story contract. Keep it in the
		// version file projection so the Files tab can inspect the same artifact
		// that powers the Preview tab. story.tsx remains preview-harness-only.
		storyPath := filepath.ToSlash(filepath.Join(versionPath, "story.json"))
		if rawStory, storyErr := fs.ReadFile(idx.fs, storyPath); storyErr == nil {
			versionFiles = append(versionFiles, ComponentVersionFile{Path: "story.json", Content: string(rawStory), ContentSHA256: digestBytes(rawStory)})
		} else if !errors.Is(storyErr, fs.ErrNotExist) {
			return IndexManifestInput{}, nil, fmt.Errorf("read story contract %s: %w", storyPath, storyErr)
		}
		// story.tsx is the executable Preview harness source. Keep it in the
		// version file projection as a read-only artifact so the Files tab can
		// explain how each declarative story is rendered.
		harnessPath := filepath.ToSlash(filepath.Join(versionPath, "story.tsx"))
		if rawHarness, harnessErr := fs.ReadFile(idx.fs, harnessPath); harnessErr == nil {
			versionFiles = append(versionFiles, ComponentVersionFile{Path: "story.tsx", Content: string(rawHarness), ContentSHA256: digestBytes(rawHarness)})
		} else if !errors.Is(harnessErr, fs.ErrNotExist) {
			return IndexManifestInput{}, nil, fmt.Errorf("read story harness %s: %w", harnessPath, harnessErr)
		}
		sourcePath := filepath.ToSlash(filepath.Join(versionPath, entryName))
		src, err := fs.ReadFile(idx.fs, sourcePath)
		if err != nil {
			return IndexManifestInput{}, nil, fmt.Errorf("read %s: %w", sourcePath, err)
		}
		headers := map[string]string{}
		if header, ok := extractHeaderBlock(string(src)); ok {
			headers, err = parseHeader(sourcePath, header)
			if err != nil {
				return IndexManifestInput{}, nil, err
			}
			if hv := strings.TrimSpace(headers["version"]); hv != "" && hv != version {
				findings = append(findings, headerDisagreementFinding(
					sourcePath,
					"version",
					version,
					hv,
					"source header @version does not match the version folder",
				))
			}
			if hid := strings.TrimSpace(headers["libraryId"]); hid != "" && hid != manifest.LibraryID {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: sourcePath, Field: "libraryId", Reason: "does not match manifest"}
			}
			if rawDeps := strings.TrimSpace(headers["deps"]); rawDeps != "" {
				if err := validateDepsHeaderJSON(rawDeps); err != nil {
					findings = append(findings, headerDisagreementFinding(
						sourcePath,
						"deps",
						"JSON object or array",
						rawDeps,
						err.Error(),
					))
				}
			}
		}
		status := VersionStatusReleased
		if strings.Contains(version, "-") {
			status = VersionStatusDraft
		}
		if deprecated[version] {
			status = VersionStatusDeprecated
		}
		if rawStatus := strings.TrimSpace(headers["status"]); rawStatus != "" {
			headerStatus := ComponentVersionStatus(strings.ToLower(rawStatus))
			if !isValidVersionStatus(headerStatus) || headerStatus != status {
				findings = append(findings, headerDisagreementFinding(
					sourcePath,
					"status",
					string(status),
					rawStatus,
					"source header @status does not match manifest-derived version status",
				))
			}
		}
		if version == manifest.LatestVersion {
			latestFound = true
			if status != VersionStatusReleased {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "must point to a released non-deprecated version"}
			}
			for k, v := range headers {
				latestHeaders[k] = v
			}
		}
		if unreleased && version == manifest.DraftVersion {
			for k, v := range headers {
				latestHeaders[k] = v
			}
		}
		if version == manifest.DraftVersion {
			draftFound = true
			if status != VersionStatusDraft {
				return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "must point to a draft/pre-release version"}
			}
		}
		parity, err := idx.readParityReport(filepath.ToSlash(filepath.Join(versionPath, "parity.json")))
		if err != nil {
			return IndexManifestInput{}, nil, err
		}
		requiredTokens := ExtractRequiredTokens(versionFiles)
		versions = append(versions, ComponentVersion{
			LibraryID:             manifest.LibraryID,
			Version:               version,
			Status:                status,
			SourcePath:            sourcePath,
			Content:               string(src),
			ContentSHA256:         digestBytes(src),
			Headers:               headers,
			Files:                 versionFiles,
			Dependencies:          dependencies,
			DependencyLockPresent: lockFile != nil,
			RequiredTokens:        requiredTokens,
			RequiredTokenPatterns: ExtractRequiredTokenPatterns(versionFiles),
			KitCompatibility:      DeriveKitCompatibility(requiredTokens, idx.kitVocabularies),
			ExperienceContract:    experienceContract,
			ParityReport:          parity,
			Presence:              "materialized",
		})
		story, storyFindings, _ := idx.readVersionStory(filepath.ToSlash(filepath.Join(versionPath, "story.json")), manifest.LibraryID, version, manifest.AssetKind, string(src))
		if story != nil {
			stories = append(stories, *story)
		}
		findings = append(findings, storyFindings...)
	}
	// Evicted versions are represented by their durable database mirror. They
	// are deliberately not reconstructed from the working tree: their absence
	// is the storage placement being indexed.
	// A missing mirror row used to fail the whole manifest. That is a deadlock
	// on first index: the component has no ledger row precisely because it has
	// never indexed, and refusing keeps it out forever. Eight assets sat in
	// that state, and because an unindexed asset cannot appear as a referrer,
	// retention stopped seeing every dependency they declared.
	//
	// An evicted version that has no mirror is unrecoverable either way, so it
	// is reported and omitted. The versions on disk — the ones composition and
	// retention actually depend on — index normally.
	if len(evicted) > 0 {
		component, err := idx.repo.GetByLibraryID(context.Background(), manifest.LibraryID)
		if err != nil {
			findings = append(findings, unrecoverableEvictedFinding(path, manifest.LibraryID, sortedVersions(evicted),
				"component has no ledger row, so declared evicted versions cannot be restored from a mirror"))
		} else {
			for _, version := range sortedVersions(evicted) {
				// The filesystem walk above is authoritative whenever the version
				// directory exists, including after cold-source restoration.
				// Checking the directory explicitly also covers a stale repository
				// projection whose version metadata is incomplete.
				if _, statErr := fs.Stat(idx.fs, filepath.ToSlash(filepath.Join(versionRoot, version))); statErr == nil {
					continue
				}
				// A restored cold version is already represented by the authored
				// directory above. Do not append its durable projection a second
				// time; doing so reuses the same version ID and duplicates files.
				materialized := false
				for _, indexed := range versions {
					if indexed.Version == version {
						materialized = true
						break
					}
				}
				if materialized {
					continue
				}
				stored, err := idx.repo.GetVersion(context.Background(), component.ID, version)
				if err != nil {
					findings = append(findings, unrecoverableEvictedFinding(path, manifest.LibraryID, []string{version},
						"no ledger row for this evicted version, so its mirror cannot be restored"))
					continue
				}
				stored.Presence = "evicted"
				versions = append(versions, stored)
			}
		}
	}
	if !latestFound && !unreleased {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "latest", Reason: "version folder not found"}
	}
	for _, indexedVersion := range versions {
		if indexedVersion.Version == currentVersion {
			manifest.Dependencies = append([]AssetDependency(nil), indexedVersion.Dependencies...)
			break
		}
	}
	if manifest.DraftVersion != "" && !draftFound {
		return IndexManifestInput{}, nil, ErrInvalidHeader{SourcePath: path, Field: "draft", Reason: "version folder not found"}
	}
	// Older catalog entries stored this in the entry header. Keep that as an
	// import-time fallback only; new manifests own the canonical value.
	if manifest.Category == "" {
		manifest.Category = strings.TrimSpace(latestHeaders["category"])
	}
	if manifest.CatalogID != "" {
		latestHeaders["catalogId"] = manifest.CatalogID
		if expects, satisfies, ok := idx.catalogPorts(manifest.CatalogID); ok {
			if raw, err := json.Marshal(expects); err == nil {
				latestHeaders["catalogExpects"] = string(raw)
			}
			if raw, err := json.Marshal(satisfies); err == nil {
				latestHeaders["catalogSatisfies"] = string(raw)
			}
			manifest.Expects = expects
			manifest.Satisfies = satisfies
		}
	}
	return IndexManifestInput{Manifest: manifest, Versions: versions, Stories: stories, Headers: latestHeaders, Findings: findings, Warnings: warnings}, latestHeaders, nil
}
