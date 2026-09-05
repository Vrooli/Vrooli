// Package slice derives a domain's vertical implementation slice from graph
// evidence and the manifest-backed zone classifier. Each rung (proto, handler,
// internal, cli, ui) carries its files, exported symbols, and typed evidence;
// the slice also carries package-level layer edges between rungs. Symbol-level
// edge resolution is deferred to the call/reference-edge phase (Q17), so the
// slice attestation is reported PARTIAL.
package slice

import (
	"sort"
	"strings"
)

const (
	RungProto    = "proto"
	RungHandler  = "handler"
	RungInternal = "internal"
	RungCLI      = "cli"
	RungUI       = "ui"
)

const (
	ZoneTransport = "transport"
	ZoneDomain    = "domain"
	ZoneCLI       = "cli"
	ZoneUI        = "ui"
)

type Snapshot struct {
	ID       string
	Scenario string
	Packages []PackageNode
	Imports  []ImportEdge
	Files    []FileNode
	Symbols  []SymbolNode
}

type PackageNode struct {
	ID         string
	ImportPath string
	RepoPath   string
}

type ImportEdge struct {
	FromPackageID string
	ToPackageID   string
	TestOnly      bool
}

type FileNode struct {
	Path      string
	PackageID string
	Lines     int
	IsTest    bool
}

type SymbolNode struct {
	Name      string
	Kind      string
	PackageID string
	FilePath  string
	Exported  bool
}

type DomainMap struct {
	Scenario string
	Domains  []Domain
}

type Domain struct {
	Name  string
	Paths []string
}

type ZoneInfo struct {
	Zone   string
	Domain string
}

type Classifier interface {
	Classify(repoPath string) ZoneInfo
}

// Evidence is one typed justification for a rung's presence. It folds directly
// into a common.v1.Citation at the handler boundary ({kind,value,source}).
type Evidence struct {
	Kind   string // declared_path | package | proto_import | proto_file
	Value  string // the locator (path / import path)
	Source string // doc | code
}

// File is one source file assigned to a rung.
type File struct {
	Path   string
	Lines  int
	IsTest bool
}

// Symbol is one exported identifier in a rung's packages.
type Symbol struct {
	Name string
	Kind string
	File string
}

// Edge is a package-level dependency from one rung to another.
type Edge struct {
	FromRung string
	ToRung   string
	Kind     string // import
}

type Rung struct {
	Name     string
	Present  bool
	Evidence []Evidence
	Files    []File
	Symbols  []Symbol
}

type DomainSlice struct {
	Scenario   string
	Domain     string
	SnapshotID string
	Rungs      []Rung
	Surfaces   []string
	LayerEdges []Edge
}

func Build(snap Snapshot, domainMap DomainMap, classifier Classifier, domain string) DomainSlice {
	domain = strings.TrimSpace(domain)
	out := DomainSlice{
		Scenario:   firstNonEmpty(snap.Scenario, domainMap.Scenario),
		Domain:     domain,
		SnapshotID: snap.ID,
	}

	// pkgRung assigns each relevant package id to a rung. Domain-owned packages
	// are classified by zone; the domain's proto package is detected via imports.
	pkgRung := map[string]string{}
	ev := newEvidenceSets()

	addDeclaredPathEvidence(ev, domainMap, out.Scenario, domain)

	for _, pkg := range snap.Packages {
		path := strings.Trim(strings.TrimSpace(pkg.RepoPath), "/")
		if path == "" {
			continue
		}
		if rung := rungForZone(classifier.Classify(path), domain); rung != "" {
			pkgRung[pkg.ID] = rung
			ev[rung] = appendEvidence(ev[rung], Evidence{Kind: "package", Value: packageEvidence(pkg), Source: "code"})
		}
	}

	protoPkgIDs := addProtoEvidence(ev, snap, out.Scenario, domain)

	// Files + exported symbols per rung (by package membership).
	rungFiles := map[string][]File{}
	rungSymbols := map[string][]Symbol{}
	for _, f := range snap.Files {
		rung := pkgRung[f.PackageID]
		if rung == "" {
			rung = protoFileRung(f.Path, out.Scenario, domain)
		}
		if rung == "" {
			continue
		}
		rungFiles[rung] = append(rungFiles[rung], File{Path: f.Path, Lines: f.Lines, IsTest: f.IsTest})
	}
	for _, s := range snap.Symbols {
		if !s.Exported {
			continue
		}
		rung := pkgRung[s.PackageID]
		if rung == "" {
			continue
		}
		rungSymbols[rung] = append(rungSymbols[rung], Symbol{Name: s.Name, Kind: s.Kind, File: s.FilePath})
	}

	for _, name := range []string{RungProto, RungHandler, RungInternal, RungCLI, RungUI} {
		evs := ev[name]
		files := sortFiles(rungFiles[name])
		symbols := sortSymbols(rungSymbols[name])
		out.Rungs = append(out.Rungs, Rung{
			Name:     name,
			Present:  len(evs) > 0,
			Evidence: evs,
			Files:    files,
			Symbols:  symbols,
		})
		if len(evs) == 0 {
			continue
		}
		switch name {
		case RungHandler:
			out.Surfaces = append(out.Surfaces, "api")
		case RungCLI:
			out.Surfaces = append(out.Surfaces, "cli")
		case RungUI:
			out.Surfaces = append(out.Surfaces, "ui")
		}
	}
	out.Surfaces = sortedUnique(out.Surfaces)
	out.LayerEdges = layerEdges(snap.Imports, pkgRung, protoPkgIDs)
	return out
}

// layerEdges derives package-level rung-to-rung edges from import edges between
// the domain's packages (and into its proto package). Deduped, sorted.
func layerEdges(imports []ImportEdge, pkgRung map[string]string, protoPkgIDs map[string]struct{}) []Edge {
	seen := map[string]struct{}{}
	var out []Edge
	for _, e := range imports {
		if e.TestOnly {
			continue
		}
		from := pkgRung[e.FromPackageID]
		if from == "" {
			continue
		}
		to := pkgRung[e.ToPackageID]
		if to == "" {
			if _, ok := protoPkgIDs[e.ToPackageID]; ok {
				to = RungProto
			}
		}
		if to == "" || from == to {
			continue
		}
		key := from + "->" + to
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Edge{FromRung: from, ToRung: to, Kind: "import"})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromRung != out[j].FromRung {
			return out[i].FromRung < out[j].FromRung
		}
		return out[i].ToRung < out[j].ToRung
	})
	return out
}

func newEvidenceSets() map[string][]Evidence {
	return map[string][]Evidence{
		RungProto:    {},
		RungHandler:  {},
		RungInternal: {},
		RungCLI:      {},
		RungUI:       {},
	}
}

// appendEvidence dedupes by (kind,value).
func appendEvidence(in []Evidence, e Evidence) []Evidence {
	if strings.TrimSpace(e.Value) == "" {
		return in
	}
	for _, existing := range in {
		if existing.Kind == e.Kind && existing.Value == e.Value {
			return in
		}
	}
	return append(in, e)
}

func addDeclaredPathEvidence(ev map[string][]Evidence, domainMap DomainMap, scenario, domain string) {
	for _, declared := range domainMap.Domains {
		if declared.Name != domain {
			continue
		}
		for _, path := range declared.Paths {
			if rung := rungForDeclaredPath(path, scenario, domain); rung != "" {
				ev[rung] = appendEvidence(ev[rung], Evidence{Kind: "declared_path", Value: declaredPathEvidence(path), Source: "doc"})
			}
		}
	}
}

func rungForZone(info ZoneInfo, domain string) string {
	if info.Domain != domain {
		return ""
	}
	switch info.Zone {
	case ZoneTransport:
		return RungHandler
	case ZoneDomain:
		return RungInternal
	case ZoneCLI:
		return RungCLI
	case ZoneUI:
		return RungUI
	default:
		return ""
	}
}

// addProtoEvidence detects the domain's proto package via imports and proto
// files on disk, recording typed evidence and returning the set of proto
// package ids (used for layer edges).
func addProtoEvidence(ev map[string][]Evidence, snap Snapshot, scenario, domain string) map[string]struct{} {
	packagesByID := map[string]PackageNode{}
	for _, pkg := range snap.Packages {
		packagesByID[pkg.ID] = pkg
	}
	protoIDs := map[string]struct{}{}
	for _, edge := range snap.Imports {
		if edge.TestOnly {
			continue
		}
		target := packagesByID[edge.ToPackageID]
		if importsDomainProto(target.ImportPath, scenario, domain) {
			ev[RungProto] = appendEvidence(ev[RungProto], Evidence{Kind: "proto_import", Value: protoEvidence(target), Source: "code"})
			protoIDs[edge.ToPackageID] = struct{}{}
		}
	}
	for _, file := range snap.Files {
		if !file.IsTest && fileMatchesDomainProto(file.Path, scenario, domain) {
			ev[RungProto] = appendEvidence(ev[RungProto], Evidence{Kind: "proto_file", Value: file.Path, Source: "code"})
		}
	}
	return protoIDs
}

func protoFileRung(path, scenario, domain string) string {
	if fileMatchesDomainProto(path, scenario, domain) {
		return RungProto
	}
	return ""
}

func packageEvidence(pkg PackageNode) string {
	if strings.TrimSpace(pkg.RepoPath) != "" {
		return strings.Trim(pkg.RepoPath, "/")
	}
	return pkg.ImportPath
}

func protoEvidence(pkg PackageNode) string {
	if pkg.ImportPath != "" {
		return pkg.ImportPath
	}
	return pkg.ID
}

func importsDomainProto(importPath, scenario, domain string) bool {
	importPath = strings.ToLower(strings.TrimSpace(importPath))
	if importPath == "" || domain == "" {
		return false
	}
	scenario = strings.ToLower(strings.TrimSpace(scenario))
	domain = strings.ToLower(strings.TrimSpace(domain))
	return strings.Contains(importPath, "/"+scenario+"/v1/"+domain) ||
		strings.Contains(importPath, "/"+scenario+"/v1/"+domain+";") ||
		strings.Contains(importPath, "/"+scenario+"/v1/"+domain+"_v1") ||
		strings.Contains(importPath, "/"+scenario+"/v1/"+domain+"/") ||
		strings.Contains(importPath, scenario+"/v1/"+domain)
}

func fileMatchesDomainProto(path, scenario, domain string) bool {
	path = strings.ToLower(strings.Trim(path, "/"))
	scenario = strings.ToLower(strings.TrimSpace(scenario))
	domain = strings.ToLower(strings.TrimSpace(domain))
	if path == "" || scenario == "" || domain == "" {
		return false
	}
	return strings.Contains(path, "proto/schemas/"+scenario+"/v1/"+domain+"/"+domain+".proto")
}

func sortFiles(in []File) []File {
	sort.Slice(in, func(i, j int) bool { return in[i].Path < in[j].Path })
	return in
}

func sortSymbols(in []Symbol) []Symbol {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Name != in[j].Name {
			return in[i].Name < in[j].Name
		}
		return in[i].File < in[j].File
	})
	return in
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func rungForDeclaredPath(path, scenario, domain string) string {
	path = strings.ToLower(strings.Trim(path, "/"))
	path = strings.TrimSuffix(path, "/**")
	scenario = strings.ToLower(strings.TrimSpace(scenario))
	domain = strings.ToLower(strings.TrimSpace(domain))
	if path == "" || domain == "" {
		return ""
	}
	switch {
	case path == "api/handlers/"+domain || strings.HasPrefix(path, "api/handlers/"+domain+"/"):
		return RungHandler
	case path == "api/internal/"+domain || strings.HasPrefix(path, "api/internal/"+domain+"/"):
		return RungInternal
	case path == "cli/domains/"+domain || strings.HasPrefix(path, "cli/domains/"+domain+"/"):
		return RungCLI
	case path == "ui/src/features/"+domain || strings.HasPrefix(path, "ui/src/features/"+domain+"/"):
		return RungUI
	case scenario != "" && (path == "packages/proto/schemas/"+scenario+"/v1/"+domain ||
		strings.HasPrefix(path, "packages/proto/schemas/"+scenario+"/v1/"+domain+"/")):
		return RungProto
	default:
		return ""
	}
}

func declaredPathEvidence(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, "/**")
	return path
}
