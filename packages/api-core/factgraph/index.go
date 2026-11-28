// Package factgraph projects Code Facts graph responses into one normalized,
// reusable file index for scanners and graph consumers.
package factgraph

import (
	"path/filepath"
	"sort"
	"strings"

	facts "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type Node struct {
	unitID     string
	id         string
	name       string
	path       string
	targetPath bool
	language   string
	kind       string
	attributes map[string]string
}

type Edge struct {
	kind       string
	from, to   string
	attributes map[string]string
}

type Unit struct {
	language, root, profile string
	extractionMS            int64
	omissions               []Omission
	nodes                   []Node
	edges                   []Edge
}

type Omission struct{ Capability, Reason string }

func (n Node) Kind() string                { return n.kind }
func (e Edge) Kind() string                { return e.kind }
func (e Edge) From() string                { return e.from }
func (e Edge) To() string                  { return e.to }
func (e Edge) Attribute(key string) string { return e.attributes[key] }
func (u Unit) Language() string            { return u.language }
func (u Unit) Root() string                { return u.root }
func (u Unit) Profile() string             { return u.profile }
func (u Unit) ExtractionMS() int64         { return u.extractionMS }
func (u Unit) Omissions() []Omission       { return append([]Omission(nil), u.omissions...) }

// Nodes preserves provider-relative node paths; check HasTargetPath before
// treating a path as target-contained.
func (u Unit) Nodes() []Node { return append([]Node(nil), u.nodes...) }
func (u Unit) Edges() []Edge { return append([]Edge(nil), u.edges...) }

type File struct {
	path     string
	language string
	imports  int
	nodes    []Node
}

func (n Node) UnitID() string              { return n.unitID }
func (n Node) ID() string                  { return n.id }
func (n Node) Name() string                { return n.name }
func (n Node) Path() string                { return n.path }
func (n Node) HasTargetPath() bool         { return n.targetPath }
func (n Node) Language() string            { return n.language }
func (n Node) Attribute(key string) string { return n.attributes[key] }

func (f File) Path() string     { return f.path }
func (f File) Language() string { return f.language }
func (f File) Imports() int     { return f.imports }

// Nodes returns graph nodes with target-relative file paths.
func (f File) Nodes() []Node { return append([]Node(nil), f.nodes...) }

// Index is an immutable path/language projection of a Code Facts graph report.
// File indexes include only target-relative source paths. Unit projections
// preserve every graph node, including external references, and mark whether
// its path is inside the declared target. Accessors return copies, so
// concurrent consumers can safely share one index.
type Index struct {
	files    map[string]map[string]File
	warnings []string
	units    []Unit
}

func NewIndex(report *facts.CodeFactsReport) *Index {
	index := &Index{files: map[string]map[string]File{}}
	if report == nil {
		return index
	}
	root := report.GetTarget().GetRootPath()
	if root == "" {
		index.warnings = append(index.warnings, "missing target root")
		return index
	}
	root = filepath.Clean(root)
	for _, envelope := range report.GetGraphs() {
		unit := envelope.GetParseUnit()
		if unit == nil || unit.GetRootPath() == "" {
			index.warnings = append(index.warnings, "parse unit missing root path")
			continue
		}
		prefix, err := filepath.Rel(root, filepath.Clean(unit.GetRootPath()))
		if err != nil || filepath.IsAbs(prefix) {
			index.warnings = append(index.warnings, "parse unit outside target: "+unit.GetRootPath())
			continue
		}
		// A parse unit normally sits at or below the target. It can also be an
		// ancestor: a caller asking about <module>/internal receives the module's
		// graph because the module is the unit that owns go.mod. In that case the
		// unit is not outside the target, it contains it, so project only the
		// nodes under the target and make their paths target-relative.
		unitPrefix, targetPrefix, unitRoot := prefix, ".", prefix
		if prefix == ".." || strings.HasPrefix(prefix, ".."+string(filepath.Separator)) {
			inverse, inverseErr := filepath.Rel(filepath.Clean(unit.GetRootPath()), root)
			if inverseErr != nil || !filepath.IsLocal(inverse) {
				index.warnings = append(index.warnings, "parse unit outside target: "+unit.GetRootPath())
				continue
			}
			unitPrefix, targetPrefix, unitRoot = ".", filepath.ToSlash(inverse), "."
		}
		graph := envelope.GetGraph()
		projected := Unit{language: unit.GetLanguage(), root: unitRoot, profile: envelope.GetExtractionProfile(), extractionMS: envelope.GetExtractionMs()}
		for _, omission := range envelope.GetOmittedInformation() {
			if omission != nil {
				projected.omissions = append(projected.omissions, Omission{Capability: omission.GetCapability(), Reason: omission.GetReason()})
			}
		}
		modulePaths := map[string]string{}
		moduleLanguages := map[string]string{}
		for _, source := range graph.GetNodes() {
			if source == nil {
				continue
			}
			attrs := cloneAttributes(source.GetAttributes())
			language := attrs["language"]
			if language == "" {
				language = unit.GetLanguage()
			}
			node := Node{unitID: unit.GetId(), id: source.GetId(), name: source.GetName(), path: source.GetPath(), language: language, kind: source.GetKind().String(), attributes: attrs}
			projected.nodes = append(projected.nodes, node)
			path, ok := graphPath(unitPrefix, source.GetPath())
			if ok {
				path, ok = scopeToTargetPath(targetPrefix, path)
			}
			node.targetPath = ok
			projected.nodes[len(projected.nodes)-1] = node
			if !ok {
				// Nodes outside the requested subdirectory are expected when the
				// parse unit is an ancestor; only a source escaping its own unit is
				// worth reporting.
				if source.GetPath() != "" && targetPrefix == "." {
					index.warnings = append(index.warnings, "symbol outside target: "+source.GetPath())
				}
				continue
			}
			if language == "" {
				continue
			}
			file := index.files[path][language]
			if file.path == "" {
				file.path, file.language = path, language
			}
			fileNode := node
			fileNode.path = path
			fileNode.targetPath = true
			file.nodes = append(file.nodes, fileNode)
			if language == "go" && attrs["kind"] == "go_import_spec" {
				file.imports++
			}
			index.put(file)
			if attrs["kind"] == "TS_NODE_KIND_MODULE" {
				modulePaths[source.GetId()] = path
				moduleLanguages[source.GetId()] = language
			}
		}
		for _, edge := range graph.GetEdges() {
			if edge != nil {
				projected.edges = append(projected.edges, Edge{kind: edge.GetKind().String(), from: edge.GetFromNodeId(), to: edge.GetToNodeId(), attributes: cloneAttributes(edge.GetAttributes())})
			}
			if edge.GetKind() != commonv1.EdgeKind_EDGE_KIND_IMPORT {
				continue
			}
			path := modulePaths[edge.GetFromNodeId()]
			language := moduleLanguages[edge.GetFromNodeId()]
			if path == "" || language == "" {
				continue
			}
			file := index.files[path][language]
			file.imports++
			index.put(file)
		}
		index.units = append(index.units, projected)
	}
	return index
}

func (i *Index) Units() []Unit {
	if i == nil {
		return nil
	}
	return append([]Unit(nil), i.units...)
}

func (i *Index) File(path, language string) (File, bool) {
	if i == nil {
		return File{}, false
	}
	file, ok := i.files[filepath.ToSlash(filepath.Clean(path))][language]
	if !ok {
		return File{}, false
	}
	return file, true
}

func (i *Index) Files() []File {
	if i == nil {
		return nil
	}
	var files []File
	for _, byLanguage := range i.files {
		for _, file := range byLanguage {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(a, b int) bool {
		if files[a].path != files[b].path {
			return files[a].path < files[b].path
		}
		return files[a].language < files[b].language
	})
	return files
}

func (i *Index) Warnings() []string {
	if i == nil {
		return nil
	}
	return append([]string(nil), i.warnings...)
}

func (i *Index) put(file File) {
	byLanguage := i.files[file.path]
	if byLanguage == nil {
		byLanguage = map[string]File{}
		i.files[file.path] = byLanguage
	}
	byLanguage[file.language] = file
}

func graphPath(prefix, path string) (string, bool) {
	if path == "" || filepath.IsAbs(path) {
		return "", false
	}
	combined := filepath.ToSlash(filepath.Clean(filepath.Join(prefix, path)))
	return combined, combined != ".." && !strings.HasPrefix(combined, "../") && !filepath.IsAbs(combined)
}

// scopeToTargetPath keeps only the part of a unit-relative path that belongs to
// the requested target directory. targetPrefix is the target's path relative to
// the unit root; "." means the whole unit is in scope.
func scopeToTargetPath(targetPrefix, path string) (string, bool) {
	targetPrefix = strings.Trim(filepath.ToSlash(targetPrefix), "/")
	if targetPrefix == "" || targetPrefix == "." {
		return path, true
	}
	if path == targetPrefix {
		return ".", true
	}
	if strings.HasPrefix(path, targetPrefix+"/") {
		return strings.TrimPrefix(path, targetPrefix+"/"), true
	}
	return "", false
}

func cloneAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(attributes))
	for key, value := range attributes {
		cloned[key] = value
	}
	return cloned
}
