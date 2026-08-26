// Package graphreconcile compares the three dependency declarations used by
// the library. It is deliberately report-only: reconciliation never edits a
// component manifest.
package graphreconcile

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

type Verdict string

const (
	Reconciled           Verdict = "reconciled"
	NotImplemented       Verdict = "not-implemented"
	ManifestEmpty        Verdict = "manifest-empty"
	UndeclaredInManifest Verdict = "undeclared-in-manifest"
	UnpinnedImport       Verdict = "unpinned-import"
	PhantomEdge          Verdict = "phantom-edge"
	ImportsUnavailable   Verdict = "imports-unavailable"
)

type Asset struct {
	AssetID       string
	Verdict       Verdict
	Cause         string
	CatalogEdges  []string
	ManifestEdges []string
	ImportEdges   []string
}

type Report struct {
	Assets       []Asset
	Distribution map[Verdict]int
}

// Compare assigns exactly one verdict to every catalog asset. The precedence
// is fixed so a corpus produces a stable backlog even when several kinds of
// drift affect the same row.
//
// implemented marks the catalog assets that have a library implementation.
// It is checked first because it is knowable from manifests alone: the catalog
// is desired state, so an asset nobody has built yet is a construction gap,
// not dependency drift, and telling a reader to reconcile three views when
// only one exists would bury the assets that really disagree.
func Compare(assetIDs []string, catalogEdges, manifestEdges, importEdges map[string][]string, implemented map[string]bool, importsAvailable bool) Report {
	ids := append([]string(nil), assetIDs...)
	sort.Strings(ids)
	report := Report{Distribution: map[Verdict]int{}}
	for _, id := range ids {
		catalog := sortedSet(catalogEdges[id])
		manifest := sortedSet(manifestEdges[id])
		imports := sortedSet(importEdges[id])
		verdict := Reconciled
		cause := "all three dependency views agree"
		switch {
		case !implemented[id]:
			verdict, cause = NotImplemented, "the catalog declares this asset but no library implementation exists yet"
		case !importsAvailable:
			verdict, cause = ImportsUnavailable, "typescript-code-graph could not provide an import graph"
		case len(manifest) == 0 && (len(catalog) > 0 || len(imports) > 0):
			verdict, cause = ManifestEmpty, "the implementation manifest declares no dependency pins"
		case difference(imports, manifest) != nil:
			verdict, cause = UnpinnedImport, "an implementation import is absent from manifest pins"
		case difference(catalog, manifest) != nil:
			verdict, cause = UndeclaredInManifest, "the catalog requires edge is absent from manifest pins"
		case difference(catalog, imports) != nil:
			verdict, cause = PhantomEdge, "the catalog requires edge has no matching implementation import"
		}
		row := Asset{AssetID: id, Verdict: verdict, Cause: cause, CatalogEdges: catalog, ManifestEdges: manifest, ImportEdges: imports}
		report.Assets = append(report.Assets, row)
		report.Distribution[verdict]++
	}
	return report
}

func sortedSet(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func difference(left, right []string) []string {
	set := map[string]struct{}{}
	for _, value := range right {
		set[value] = struct{}{}
	}
	var out []string
	for _, value := range left {
		if _, ok := set[value]; !ok {
			out = append(out, value)
		}
	}
	return out
}

// Reconcile loads authored catalog and manifest edges and asks the platform
// TypeScript graph extractor for actual imports. Failure to reach that
// capability is represented in the report rather than guessed around.
func Reconcile(ctx context.Context, repoRoot string) (Report, error) {
	catalogDir := filepath.Join(repoRoot, "scenarios", "react-component-library", "catalog")
	libraryDir := filepath.Join(repoRoot, "scenarios", "react-component-library", "library")
	assets, err := loadCatalog(catalogDir)
	if err != nil {
		return Report{}, err
	}
	impls, err := loadImplementations(libraryDir)
	if err != nil {
		return Report{}, err
	}
	assetIDs := make([]string, 0, len(assets))
	catalogEdges := make(map[string][]string, len(assets))
	for _, asset := range assets {
		assetIDs = append(assetIDs, asset.ID)
		catalogEdges[asset.ID] = append([]string(nil), asset.Requires...)
	}
	byLibrary := map[string]string{}
	manifestEdges := map[string][]string{}
	implemented := map[string]bool{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			implemented[impl.CatalogID] = true
		}
		if impl.LibraryID == "" || impl.CatalogID == "" {
			continue
		}
		byLibrary[impl.LibraryID] = impl.CatalogID
	}
	for _, impl := range impls {
		if impl.CatalogID == "" {
			continue
		}
		for _, dependency := range impl.Dependencies {
			if target := byLibrary[dependency]; target != "" {
				manifestEdges[impl.CatalogID] = append(manifestEdges[impl.CatalogID], target)
			}
		}
	}
	imports, available, cause, err := extractImports(ctx, repoRoot, impls)
	if err != nil {
		return Report{}, err
	}
	report := Compare(assetIDs, catalogEdges, manifestEdges, imports, implemented, available)
	if !available && cause != "" {
		for index := range report.Assets {
			if report.Assets[index].Verdict == ImportsUnavailable {
				report.Assets[index].Cause = cause
			}
		}
	}
	return report, nil
}

type (
	catalogAsset struct {
		ID       string
		Requires []string
	}
	implementation struct {
		Name, Root, CatalogID, LibraryID string
		Latest, Draft                    string
		Dependencies                     []string
	}
)

func loadCatalog(dir string) ([]catalogAsset, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "assets", "*", "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []catalogAsset
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var raw struct {
			Kind  string `json:"kind"`
			Asset struct {
				ID string `json:"id"`
			} `json:"asset"`
			Dependencies struct {
				Requires []struct {
					Asset string `json:"asset"`
				} `json:"requires"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, err
		}
		if raw.Kind != "catalog-asset" {
			continue
		}
		asset := catalogAsset{ID: raw.Asset.ID}
		for _, edge := range raw.Dependencies.Requires {
			asset.Requires = append(asset.Requires, edge.Asset)
		}
		out = append(out, asset)
	}
	return out, nil
}

func loadImplementations(dir string) ([]implementation, error) {
	var out []implementation
	for _, root := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		paths, err := filepath.Glob(filepath.Join(dir, root, "*", "component.json"))
		if err != nil {
			return nil, err
		}
		sort.Strings(paths)
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			var raw struct {
				LibraryID    string `json:"libraryId"`
				CatalogID    string `json:"catalogId"`
				Latest       string `json:"latest"`
				Draft        string `json:"draft"`
				Dependencies []struct {
					LibraryID string `json:"libraryId"`
				} `json:"dependencies"`
			}
			if err := json.Unmarshal(data, &raw); err != nil {
				return nil, err
			}
			impl := implementation{Name: filepath.Base(filepath.Dir(path)), Root: root, CatalogID: raw.CatalogID, LibraryID: raw.LibraryID, Latest: raw.Latest, Draft: raw.Draft}
			for _, dep := range raw.Dependencies {
				impl.Dependencies = append(impl.Dependencies, dep.LibraryID)
			}
			out = append(out, impl)
		}
	}
	return out, nil
}

// graphSourceFiles lists the released (and draft) version sources of every
// implementation, relative to the UI directory that owns the TypeScript
// project. Only the versions an asset actually publishes are included:
// assetForPath collapses every version of an asset onto one catalog ID, so
// compiling retired versions would attribute their stale imports to the
// current asset and manufacture drift that no shipped source contains.
func graphSourceFiles(libraryDir, uiDir string, impls []implementation) ([]string, error) {
	var out []string
	for _, impl := range impls {
		seen := map[string]bool{}
		for _, version := range []string{impl.Latest, impl.Draft} {
			version = strings.TrimSpace(version)
			if version == "" || seen[version] {
				continue
			}
			seen[version] = true
			versionDir := filepath.Join(libraryDir, impl.Root, impl.Name, "versions", version)
			entries, err := os.ReadDir(versionDir)
			if err != nil {
				// A manifest pointing at a missing version directory is the
				// conformance gate's finding to report, not this one's to fail on.
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || !(strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".tsx")) {
					continue
				}
				relative, err := filepath.Rel(uiDir, filepath.Join(versionDir, name))
				if err != nil {
					return nil, err
				}
				out = append(out, filepath.ToSlash(relative))
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

// writeGraphProject emits a throwaway TypeScript project whose only purpose is
// import extraction. It extends tsconfig.catalog.json for the module
// resolution the library sources need, and clears the inherited include so the
// UI application's own sources do not enter the graph. `include` must be reset
// explicitly: extends inherits include, files, and exclude.
//
// The project lives in its own directory under ui/ because the extractor only
// accepts a project path named tsconfig.json. Staying under ui/ keeps the
// inherited baseUrl and paths mappings — which resolve against the config that
// declares them — pointing at the UI's node_modules.
func writeGraphProject(uiDir string, files []string) (string, error) {
	dir, err := os.MkdirTemp(uiDir, ".graph-project-*")
	if err != nil {
		return "", err
	}
	relative := make([]string, 0, len(files))
	for _, file := range files {
		relative = append(relative, path.Join("..", file))
	}
	document := map[string]any{
		"extends": "../tsconfig.catalog.json",
		"files":   relative,
		"include": []string{},
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), append(encoded, '\n'), 0o644); err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func extractImports(parent context.Context, repoRoot string, impls []implementation) (map[string][]string, bool, string, error) {
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	scenarioDir := filepath.Join(repoRoot, "scenarios", "react-component-library")
	uiDir := filepath.Join(scenarioDir, "ui")
	files, err := graphSourceFiles(filepath.Join(scenarioDir, "library"), uiDir, impls)
	if err != nil {
		return nil, false, "", err
	}
	if len(files) == 0 {
		return map[string][]string{}, false, "no published implementation sources resolved for extraction", nil
	}
	project, err := writeGraphProject(uiDir, files)
	if err != nil {
		return nil, false, "", err
	}
	defer os.RemoveAll(project)
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	baseURL, err := resolver.ResolveScenarioURLDefault(ctx, "typescript-code-graph")
	if err != nil {
		return map[string][]string{}, false, "typescript-code-graph could not provide an import graph: scenario discovery failed: " + err.Error(), nil
	}
	client := graphconnect.NewTypeScriptCodeGraphServiceClient(&http.Client{Timeout: 90 * time.Second}, baseURL)
	response, err := client.Extract(ctx, connect.NewRequest(&graphv1.ExtractRequest{ProjectPath: filepath.Join(project, "tsconfig.json")}))
	if err != nil {
		return map[string][]string{}, false, extractionCause(err, ""), nil
	}
	document := response.Msg.GetGraph()
	if document == nil {
		return map[string][]string{}, false, "typescript-code-graph returned an empty graph", nil
	}
	nodePath := map[string]string{}
	for _, node := range document.GetNodes() {
		nodePath[node.GetId()] = filepath.ToSlash(node.GetPath())
	}
	// Extractor paths are relative to the generated project directory, so they
	// carry one ascent per level between it and the scenario root. Strip every
	// leading ascent rather than a fixed number: the depth is a property of
	// where the throwaway project lives, not of the asset being matched.
	assetForPath := func(nodePath string) string {
		nodePath = filepath.ToSlash(nodePath)
		for strings.HasPrefix(nodePath, "../") {
			nodePath = strings.TrimPrefix(nodePath, "../")
		}
		for _, impl := range impls {
			dir := path.Join("library", impl.Root, impl.Name)
			if strings.HasPrefix(nodePath, dir+"/") || nodePath == dir {
				return impl.CatalogID
			}
		}
		return ""
	}
	imports := map[string][]string{}
	for _, edge := range document.GetEdges() {
		from := assetForPath(nodePath[edge.GetFromNodeId()])
		to := assetForPath(nodePath[edge.GetToNodeId()])
		if from != "" && to != "" && from != to {
			imports[from] = append(imports[from], to)
		}
	}
	return imports, true, "", nil
}

// extractionCause keeps the extractor's own diagnosis in the report. Without
// it every asset reports imports-unavailable with no way to tell a stopped
// sidecar from an unresolvable project.
func extractionCause(err error, stderr string) string {
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		detail = err.Error()
	}
	if index := strings.IndexByte(detail, '\n'); index >= 0 {
		detail = strings.TrimSpace(detail[:index])
	}
	const limit = 300
	if len(detail) > limit {
		detail = detail[:limit] + "…"
	}
	return "typescript-code-graph could not provide an import graph: " + detail
}
