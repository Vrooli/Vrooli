package catalogcoverage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
)

// MergedEvidence combines fresh persisted browser outcomes with recomputed catalog gates.
func MergedEvidence(ctx context.Context, root string, store *EvidenceStore) ([]GateEvidence, error) {
	var runtimeDB interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	var persisted []GateEvidence
	// One revision index serves the whole pass. Every per-asset freshness
	// decision below is a map lookup against it rather than its own corpus
	// walk, which is what makes per-asset attribution affordable at all.
	revisions, err := BuildRevisionIndex(root)
	if err != nil {
		return nil, err
	}
	var stale map[string]map[string]bool
	if store != nil {
		runtimeDB = store.Database()
		persisted, err = store.List(ctx)
		if err != nil {
			return nil, err
		}
		definitions, definitionErr := LoadGateDefinitions(filepath.Join(resolveScenarioRoot(root), "catalog", "config.json"))
		if definitionErr != nil {
			return nil, definitionErr
		}
		stale = staleAssetsByGate(root, persisted, definitions, revisions)
	}
	computed, err := recomputeEvidenceWithSkip(root, runtimeDB, stale, revisions)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return computed, nil
	}
	if len(computed) > 0 {
		var attributableRows []GateEvidence
		for _, item := range computed {
			if item.AssetID != "__corpus__" {
				attributableRows = append(attributableRows, item)
			}
		}
		if err := store.Save(ctx, attributableRows); err != nil {
			return nil, err
		}
		persisted, err = store.List(ctx)
		if err != nil {
			return nil, err
		}
	}
	for _, item := range persisted {
		if item.AssetID == "__corpus__" && item.SourceRevision == "corpus" {
			computed = append(computed, item)
			continue
		}
		// Older evidence producers could persist pseudo-assets such as
		// workbench.conformance. They are corpus observations, not catalog
		// assets, and must never enter the attributable revision join.
		if _, known := revisions[item.AssetID]; !known {
			continue
		}
		if revisions[item.AssetID] != item.SourceRevision {
			continue
		}
		computed = append(computed, item)
	}
	return computed, nil
}

// staleAssetsByGate returns, for every attributable gate, the assets whose
// persisted verdict can no longer be trusted: no evidence recorded, or
// evidence recorded against a different revision. An entry with an empty set
// means every applicable asset is current and the runner can be skipped
// outright.
//
// This used to be an all-or-nothing question — a gate counted as fresh only if
// every applicable asset was current, so one edited component re-ran every
// runner across the whole corpus. Corpus gates are absent from the result
// because their verdict is not attributable to an individual asset and must
// always be recomputed.
func staleAssetsByGate(root string, persisted []GateEvidence, definitions []GateDefinition, revisions map[string]string) map[string]map[string]bool {
	catalogRoot := filepath.Join(resolveScenarioRoot(root), "catalog")
	assets, err := LoadCatalog(catalogRoot)
	if err != nil {
		return nil
	}
	// Track exactly the assets the recompute loop will visit, which is the
	// built ones: that loop skips any catalog asset with no implementation, so
	// tracking more here would mark a never-emitted row permanently stale and
	// keep every runner warm-proof. Tracking fewer would drop a row on warm
	// passes and silently move the reported rungs. The two sets have to be the
	// same set, and TestWarmPassProducesIdenticalEvidenceToColdPass is what
	// holds them together.
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil
	}
	built := make(map[string]bool, len(impls))
	for _, impl := range impls {
		if impl.CatalogID != "" {
			built[impl.CatalogID] = true
		}
	}
	bindingsByAsset := make(map[string]map[string]RuleBinding, len(assets))
	for _, asset := range assets {
		if !built[asset.ID] {
			continue
		}
		bindings, resolveErr := ResolveRuleSetFor(catalogRoot, asset, definitions)
		if resolveErr != nil {
			// Small isolated fixtures used by the evidence tests often provide
			// catalog assets without the production config.json. Preserve the
			// legacy definition-driven applicability in that case; production
			// catalogs still use the resolver above as the source of truth.
			for _, definition := range definitions {
				if definition.Attribution == "corpus" || !slices.Contains(definition.AppliesTo, asset.Kind) {
					continue
				}
				bindings = append(bindings, RuleBinding{GateID: definition.ID})
			}
		}
		bindingsByAsset[asset.ID] = make(map[string]RuleBinding, len(bindings))
		for _, binding := range bindings {
			bindingsByAsset[asset.ID][binding.GateID] = binding
		}
	}
	byGateAsset := map[string]map[string]GateEvidence{}
	for _, item := range persisted {
		if item.AssetID == "__corpus__" {
			continue
		}
		if byGateAsset[item.Gate] == nil {
			byGateAsset[item.Gate] = map[string]GateEvidence{}
		}
		if current, ok := byGateAsset[item.Gate][item.AssetID]; !ok || item.RecordedAt > current.RecordedAt {
			byGateAsset[item.Gate][item.AssetID] = item
		}
	}
	out := map[string]map[string]bool{}
	for _, definition := range definitions {
		if definition.Attribution == "corpus" {
			continue
		}
		gateStale := map[string]bool{}
		for assetID := range bindingsByAsset {
			if _, applicable := bindingsByAsset[assetID][definition.ID]; !applicable {
				continue
			}
			revision, known := revisions[assetID]
			if !known {
				// An asset with no computable revision can never be shown to be
				// current, so it is always recomputed rather than assumed good.
				gateStale[assetID] = true
				continue
			}
			item, ok := byGateAsset[definition.ID][assetID]
			digest, digestErr := RuleSetDigest(root, assetID)
			digestStale := digestErr == nil && (item.RuleSetDigest == "" || digest != item.RuleSetDigest)
			if !ok || revision != item.SourceRevision || digestStale {
				gateStale[assetID] = true
			}
		}
		out[definition.ID] = gateStale
	}
	return out
}

// MergeExperienceEvidence adds only evidence that is both declared by a
// catalog gate and observed by Experience Manager. A manager outage is
// intentionally non-fatal: it leaves the gate absent (and therefore below
// target) instead of converting an unavailable capture into a pass.
func MergeExperienceEvidence(ctx context.Context, root string, store *EvidenceStore, fetch ExperienceCaptureFetcher) ([]GateEvidence, error) {
	evidence, err := MergedEvidence(ctx, root, store)
	if err != nil || fetch == nil {
		return evidence, err
	}
	assets, err := LoadCatalog(filepath.Join(resolveScenarioRoot(root), "catalog"))
	if err != nil {
		return nil, err
	}
	impls, err := LoadImplementations(filepath.Join(resolveScenarioRoot(root), "library"))
	if err != nil {
		return nil, err
	}
	gates, err := LoadGateDefinitions(filepath.Join(resolveScenarioRoot(root), "catalog", "config.json"))
	if err != nil {
		return nil, err
	}
	targets := map[string]string{}
	for _, asset := range assets {
		target := "react-vite"
		if len(asset.Targets) > 0 && strings.TrimSpace(asset.Targets[0]) != "" {
			target = asset.Targets[0]
		}
		targets[asset.ID] = target
	}
	// Experience Manager evidence is an optional enrichment, but the catalog
	// contains hundreds of exact-version implementations. Fetching them in a
	// single serial loop can consume the validation RPC deadline before the
	// component gate even starts. Keep the fan-out bounded so the manager and
	// its database are protected while the validation path remains responsive.
	type fetchJob struct {
		libraryID string
		assetID   string
		target    string
		version   string
	}
	type fetchResult struct{ captures []ExperienceCapture }
	jobs := make(chan fetchJob)
	results := make(chan fetchResult)
	const workers = 8
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					return
				}
				fetched, fetchErr := fetch(ctx, job.libraryID, job.version)
				if fetchErr != nil {
					continue
				}
				for index := range fetched {
					fetched[index].AssetID = job.assetID
					fetched[index].Target = job.target
					fetched[index].Version = job.version
				}
				results <- fetchResult{captures: fetched}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, impl := range impls {
			if impl.CatalogID == "" || impl.Latest == "" || targets[impl.CatalogID] == "" || ctx.Err() != nil {
				continue
			}
			job := fetchJob{libraryID: "react-component-library:" + impl.Name, assetID: impl.CatalogID, target: targets[impl.CatalogID], version: impl.Latest}
			select {
			case jobs <- job:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wait.Wait()
		close(results)
	}()
	var captures []ExperienceCapture
	for result := range results {
		captures = append(captures, result.captures...)
	}
	fresh, err := deriveExperienceEvidence(root, captures, gates)
	if err != nil {
		return nil, err
	}
	if len(fresh) == 0 {
		return evidence, nil
	}
	if store != nil {
		if err := store.Save(ctx, fresh); err != nil {
			return nil, err
		}
	}
	return append(evidence, fresh...), nil
}

func deriveExperienceEvidence(root string, captures []ExperienceCapture, definitions []GateDefinition) ([]GateEvidence, error) {
	revisions, revisionErr := BuildRevisionIndex(root)
	if revisionErr != nil {
		return nil, revisionErr
	}
	// Experience Manager retains an audit history. Reduce it to the newest
	// observation for each declared claim/state/example/viewport before a gate
	// is evaluated; otherwise one old skipped capture can poison a current pass.
	latest := map[string]ExperienceCapture{}
	for _, capture := range captures {
		// Experience Manager may retain observations for workbench or
		// supplemental surfaces that are not catalog assets. They cannot be
		// scored by this catalog and must not make the evidence join fail.
		if _, known := revisions[capture.AssetID]; !known {
			continue
		}
		claim := capture.ClaimID
		if claim == "" {
			claim = capture.ClaimType
		}
		key := strings.Join([]string{capture.AssetID, capture.Target, claim, capture.StateID, capture.ExampleName, capture.Viewport}, "\x00")
		current, exists := latest[key]
		if !exists || capture.CheckedAt > current.CheckedAt {
			latest[key] = capture
		}
	}
	type aggregate struct {
		captures []ExperienceCapture
	}
	groups := map[string]*aggregate{}
	for _, capture := range latest {
		for _, definition := range definitions {
			if !contains(definition.ExperienceClaimTypes, capture.ClaimType) {
				continue
			}
			key := capture.AssetID + "\x00" + capture.Target + "\x00" + definition.ID
			if groups[key] == nil {
				groups[key] = &aggregate{}
			}
			groups[key].captures = append(groups[key].captures, capture)
		}
	}
	var out []GateEvidence
	for key, group := range groups {
		parts := strings.Split(key, "\x00")
		if len(parts) != 3 || len(group.captures) == 0 {
			continue
		}
		assetID, target, gateID := parts[0], parts[1], parts[2]
		var definition GateDefinition
		for _, candidate := range definitions {
			if candidate.ID == gateID {
				definition = candidate
				break
			}
		}
		result := "pass"
		viewports := map[string]struct{}{}
		for _, capture := range group.captures {
			if capture.Viewport != "" {
				viewports[capture.Viewport] = struct{}{}
			}
			switch strings.ToLower(strings.TrimSpace(capture.Verdict)) {
			case "failed", "fail", "error":
				result = "fail"
			case "blocked", "skipped", "not-run":
				if result == "pass" {
					result = "skipped"
				}
			}
			if definition.requiresCapture() && strings.TrimSpace(capture.CaptureRef) == "" && result == "pass" {
				result = "skipped"
			}
		}
		if definition.minimumViewports() > 0 && len(viewports) < definition.minimumViewports() && result == "pass" {
			result = "skipped"
		}
		revision, known := revisions[assetID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q not found", assetID)
		}
		out = append(out, GateEvidence{AssetID: assetID, Target: target, Version: group.captures[0].Version, Gate: gateID, Result: result, SourceRevision: revision})
	}
	// A completed, passing capture matrix is also direct evidence for the
	// capture-backed visual and responsive gates. These gates used to depend on
	// a claim type being present in a particular contract, which meant a
	// perfectly captured component with only element-presence claims remained
	// permanently below verified. Keep the proof conservative: every latest
	// observation must pass, every observation must have a screenshot reference,
	// and responsive evidence must span the configured viewport count.
	type matrix struct {
		allPassed bool
		refs      int
		viewports map[string]struct{}
	}
	matrixByAsset := map[string]*matrix{}
	for _, capture := range latest {
		if _, known := revisions[capture.AssetID]; !known {
			continue
		}
		key := capture.AssetID + "\x00" + capture.Target
		entry := matrixByAsset[key]
		if entry == nil {
			entry = &matrix{allPassed: true, viewports: map[string]struct{}{}}
			matrixByAsset[key] = entry
		}
		if !strings.EqualFold(strings.TrimSpace(capture.Verdict), "passed") {
			entry.allPassed = false
		}
		if strings.TrimSpace(capture.CaptureRef) == "" {
			entry.allPassed = false
		} else {
			entry.refs++
		}
		if strings.TrimSpace(capture.Viewport) != "" {
			entry.viewports[capture.Viewport] = struct{}{}
		}
	}
	for key, entry := range matrixByAsset {
		if !entry.allPassed || entry.refs == 0 {
			continue
		}
		parts := strings.Split(key, "\x00")
		if len(parts) != 2 {
			continue
		}
		assetID, target := parts[0], parts[1]
		revision, known := revisions[assetID]
		if !known {
			return nil, fmt.Errorf("catalog asset %q not found", assetID)
		}
		for _, definition := range definitions {
			if definition.ID != "visual" && definition.ID != "responsive" {
				continue
			}
			if definition.ID == "responsive" && len(entry.viewports) < definition.minimumViewports() {
				continue
			}
			out = append(out, GateEvidence{AssetID: assetID, Target: target, Version: latestCaptureVersion(latest, assetID, target), Gate: definition.ID, Result: "pass", SourceRevision: revision})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AssetID != out[j].AssetID {
			return out[i].AssetID < out[j].AssetID
		}
		return out[i].Gate < out[j].Gate
	})
	return out, nil
}

func latestCaptureVersion(captures map[string]ExperienceCapture, assetID, target string) string {
	for _, capture := range captures {
		if capture.AssetID == assetID && capture.Target == target && capture.Version != "" {
			return capture.Version
		}
	}
	return "legacy"
}
