package explorer

// DOC: docs/reference/api-endpoints.md#scenario-documentation-tree

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"knowledge-observatory/internal/docschema"
)

type treeNode struct {
	DocTreeNode
	children map[string]*treeNode
}

func newTreeNode(name, path, nodeType string) *treeNode {
	return &treeNode{
		DocTreeNode: DocTreeNode{
			Name: name,
			Path: path,
			Type: nodeType,
		},
		children: map[string]*treeNode{},
	}
}

// GetDocTree builds a documentation file tree for a scenario.
func (s *Service) GetDocTree(ctx context.Context, scenarioName string) (*DocTreeNode, error) {
	if s == nil {
		return nil, fmt.Errorf("explorer service is nil")
	}
	scenarioPath, err := s.scenarioPath(scenarioName)
	if err != nil {
		return nil, err
	}
	validation, err := docschema.ValidateScenarioDocumentation(scenarioPath)
	if err != nil {
		return nil, err
	}
	warnings := buildWarningIndex(validation)

	root := newTreeNode(scenarioName, s.repoRelative(scenarioPath), "directory")

	// Root-level docs
	for _, name := range []string{"README.md", "PRD.md"} {
		abs := filepath.Join(scenarioPath, name)
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			addFileNode(root, s, scenarioPath, abs, info, warnings)
		}
	}

	docsRoot := filepath.Join(scenarioPath, "docs")
	if info, err := os.Stat(docsRoot); err == nil && info.IsDir() {
		ensureDirNode(root, s, scenarioPath, "docs")
		if err := filepath.WalkDir(docsRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if d.IsDir() {
				if shouldSkipDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isDocFile(d.Name()) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			addFileNode(root, s, scenarioPath, path, info, warnings)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	node := finalizeTree(root)
	return &node, nil
}

type warningIndex struct {
	misplaced map[string]docschema.MisplacedDoc
	extra     map[string]struct{}
}

func buildWarningIndex(validation *docschema.ValidationResult) warningIndex {
	index := warningIndex{
		misplaced: map[string]docschema.MisplacedDoc{},
		extra:     map[string]struct{}{},
	}
	if validation == nil {
		return index
	}
	for _, misplaced := range validation.MisplacedDocs {
		index.misplaced[filepath.ToSlash(misplaced.ActualPath)] = misplaced
	}
	for _, extra := range validation.ExtraDocs {
		index.extra[filepath.ToSlash(extra)] = struct{}{}
	}
	return index
}

func warningFor(relPath string, warnings warningIndex) *DocWarning {
	relPath = filepath.ToSlash(relPath)
	if misplaced, ok := warnings.misplaced[relPath]; ok {
		return &DocWarning{
			Type:         "misplaced",
			Message:      "Documentation file is in the wrong location",
			ExpectedPath: misplaced.ExpectedPath,
			Severity:     misplaced.Severity,
		}
	}
	if _, ok := warnings.extra[relPath]; ok {
		return &DocWarning{
			Type:     "extra",
			Message:  fmt.Sprintf("Documentation file is outside the standard layout: %s", relPath),
			Severity: "info",
		}
	}
	return nil
}

func addFileNode(root *treeNode, svc *Service, scenarioPath, absPath string, info fs.FileInfo, warnings warningIndex) {
	rel, err := filepath.Rel(scenarioPath, absPath)
	if err != nil {
		return
	}
	rel = filepath.ToSlash(rel)
	segments := splitSegments(rel)
	if len(segments) == 0 {
		return
	}
	current := root
	for i, segment := range segments {
		isFile := i == len(segments)-1
		if isFile {
			if existing, ok := current.children[segment]; ok && existing.Type == "file" {
				return
			}
			docType := ""
			if dt, ok := docschema.DocTypeForPath(segment); ok {
				docType = string(dt)
			}
			current.children[segment] = &treeNode{
				DocTreeNode: DocTreeNode{
					Name:       segment,
					Path:       svc.repoRelative(absPath),
					Type:       "file",
					DocType:    docType,
					Size:       info.Size(),
					ModifiedAt: info.ModTime(),
					Warning:    warningFor(rel, warnings),
				},
				children: nil,
			}
			return
		}
		child, ok := current.children[segment]
		if !ok {
			child = &treeNode{
				DocTreeNode: DocTreeNode{
					Name: segment,
					Path: svc.repoRelative(filepath.Join(scenarioPath, filepath.FromSlash(strings.Join(segments[:i+1], "/")))),
					Type: "directory",
				},
				children: map[string]*treeNode{},
			}
			current.children[segment] = child
		}
		current = child
	}
}

func ensureDirNode(root *treeNode, svc *Service, scenarioPath, relDir string) {
	segments := splitSegments(relDir)
	if len(segments) == 0 {
		return
	}
	current := root
	for i, segment := range segments {
		child, ok := current.children[segment]
		if !ok {
			child = &treeNode{
				DocTreeNode: DocTreeNode{
					Name: segment,
					Path: svc.repoRelative(filepath.Join(scenarioPath, filepath.FromSlash(strings.Join(segments[:i+1], "/")))),
					Type: "directory",
				},
				children: map[string]*treeNode{},
			}
			current.children[segment] = child
		}
		current = child
	}
}

func splitSegments(rel string) []string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func finalizeTree(root *treeNode) DocTreeNode {
	if root == nil {
		return DocTreeNode{}
	}
	if len(root.children) == 0 {
		return root.DocTreeNode
	}
	children := make([]DocTreeNode, 0, len(root.children))
	for _, child := range root.children {
		children = append(children, finalizeTree(child))
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].Type != children[j].Type {
			return children[i].Type < children[j].Type
		}
		return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
	})
	node := root.DocTreeNode
	node.Children = children
	return node
}

func isDocFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".md" || ext == ".json"
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".")
}
