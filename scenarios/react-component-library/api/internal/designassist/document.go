package designassist

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Node mirrors the client's TemplateNode. Only what a proposal may inspect is
// retained; unknown client fields are ignored rather than rejected so the
// studio can add a rendering prop without breaking assistance.
type Node struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Role     string         `json:"role,omitempty"`
	Accepts  []string       `json:"accepts,omitempty"`
	Props    map[string]any `json:"props"`
	Children []Node         `json:"children,omitempty"`
}

// Frame is one device composition inside a design.
type Frame struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Device string `json:"device"`
	Spec   Node   `json:"spec"`
}

// Document is the part of a design a proposal can see.
type Document struct {
	Name   string            `json:"name"`
	Theme  map[string]string `json:"theme"`
	Frames []Frame           `json:"frames"`
}

// ParseDocument reads the opaque API payloads into the assistant's view.
// A malformed frame or theme is an error: proposing against a document the
// studio cannot render would produce an unusable patch.
func ParseDocument(framesRaw, themeRaw json.RawMessage) (Document, error) {
	doc := Document{Theme: map[string]string{}}
	if len(framesRaw) > 0 && string(framesRaw) != "null" {
		if err := json.Unmarshal(framesRaw, &doc.Frames); err != nil {
			return Document{}, fmt.Errorf("frames are not a design array: %w", err)
		}
	}
	if len(themeRaw) > 0 && string(themeRaw) != "null" {
		if err := json.Unmarshal(themeRaw, &doc.Theme); err != nil {
			return Document{}, fmt.Errorf("theme is not a string map: %w", err)
		}
	}
	if doc.Theme == nil {
		doc.Theme = map[string]string{}
	}
	seen := map[string]string{}
	for frameIndex := range doc.Frames {
		frame := &doc.Frames[frameIndex]
		if strings.TrimSpace(frame.ID) == "" {
			return Document{}, fmt.Errorf("frame %d has no id", frameIndex)
		}
		if err := walkValue(frame.Spec, func(node Node) error {
			if strings.TrimSpace(node.ID) == "" {
				return fmt.Errorf("frame %s has a layer without an id", frame.ID)
			}
			if node.Type == "" {
				return fmt.Errorf("layer %s has no type", node.ID)
			}
			if previous, ok := seen[node.ID]; ok {
				return fmt.Errorf("layer id %s is reused (frames %s and %s)", node.ID, previous, frame.ID)
			}
			seen[node.ID] = frame.ID
			return nil
		}); err != nil {
			return Document{}, err
		}
	}
	return doc, nil
}

func walkValue(node Node, visit func(Node) error) error {
	if err := visit(node); err != nil {
		return err
	}
	for _, child := range node.Children {
		if err := walkValue(child, visit); err != nil {
			return err
		}
	}
	return nil
}

// treeNode is the mutable form validation simulates operations on.
type treeNode struct {
	ID       string
	Type     string
	Props    map[string]any
	Children []*treeNode
	Parent   *treeNode
}

type tree struct {
	root   *treeNode
	frame  *Frame
	byID   map[string]*treeNode
	frames []*tree
}

func newTree(frame *Frame) (*tree, error) {
	index := map[string]*treeNode{}
	root, err := buildTreeNode(frame.Spec, nil, index)
	if err != nil {
		return nil, err
	}
	return &tree{root: root, frame: frame, byID: index}, nil
}

func buildTreeNode(node Node, parent *treeNode, index map[string]*treeNode) (*treeNode, error) {
	current := &treeNode{ID: node.ID, Type: node.Type, Props: map[string]any{}, Parent: parent}
	for key, value := range node.Props {
		current.Props[key] = value
	}
	index[current.ID] = current
	for _, child := range node.Children {
		next, err := buildTreeNode(child, current, index)
		if err != nil {
			return nil, err
		}
		current.Children = append(current.Children, next)
	}
	return current, nil
}

// index is the whole document as trees, used by semantic validation.
type documentIndex struct {
	trees []*tree
	byID  map[string]*treeNode
	owner map[string]*tree
}

func indexDocument(doc Document) (*documentIndex, error) {
	index := &documentIndex{byID: map[string]*treeNode{}, owner: map[string]*tree{}}
	for frameIndex := range doc.Frames {
		frame := &doc.Frames[frameIndex]
		current, err := newTree(frame)
		if err != nil {
			return nil, err
		}
		index.trees = append(index.trees, current)
		for id, node := range current.byID {
			index.byID[id] = node
			index.owner[id] = current
		}
	}
	return index, nil
}

// BuildContext returns the JSON the model sees and the ids a proposal may
// target. The allowed set is deliberately the context, not the whole design:
// a layer-scoped request must not silently rewrite another frame.
func BuildContext(doc Document, scope Scope) (json.RawMessage, map[string]struct{}, error) {
	allowed := map[string]struct{}{}
	allow := func(node Node) {
		_ = walkValue(node, func(current Node) error {
			allowed[current.ID] = struct{}{}
			return nil
		})
	}

	var payload any
	switch scope.Kind {
	case ScopeDesign:
		for _, frame := range doc.Frames {
			allow(frame.Spec)
		}
		payload = struct {
			Design Document `json:"design"`
		}{doc}

	case ScopeFrame:
		frame, err := findFrame(doc, scope.FrameID)
		if err != nil {
			return nil, nil, err
		}
		allow(frame.Spec)
		payload = struct {
			Design struct {
				Name  string            `json:"name"`
				Theme map[string]string `json:"theme"`
			} `json:"design"`
			Frame Frame `json:"frame"`
		}{
			Design: struct {
				Name  string            `json:"name"`
				Theme map[string]string `json:"theme"`
			}{doc.Name, doc.Theme},
			Frame: *frame,
		}

	case ScopeLayer:
		frame, err := findFrame(doc, scope.FrameID)
		if err != nil {
			return nil, nil, err
		}
		selected, err := findNode(frame, scope.NodeID)
		if err != nil {
			return nil, nil, err
		}
		frameTree, err := newTree(frame)
		if err != nil {
			return nil, nil, err
		}
		target := frameTree.byID[scope.NodeID]
		if target == nil {
			return nil, nil, fmt.Errorf("%w: layer %s is not in frame %s", ErrInvalid, scope.NodeID, frame.ID)
		}
		allow(*selected)
		ancestors := []map[string]string{}
		parent := target.Parent
		if parent != nil {
			for _, node := range parent.Children {
				allowed[node.ID] = struct{}{}
			}
			for cursor := parent; cursor != nil; cursor = cursor.Parent {
				allowed[cursor.ID] = struct{}{}
				ancestors = append(ancestors, map[string]string{"id": cursor.ID, "type": cursor.Type})
			}
		}
		siblings := []map[string]string{}
		if parent != nil {
			for _, sibling := range parent.Children {
				siblings = append(siblings, map[string]string{"id": sibling.ID, "type": sibling.Type})
			}
		}
		var parentPayload any
		if parent != nil {
			parentPayload = map[string]string{"id": parent.ID, "type": parent.Type}
		}
		payload = struct {
			Design struct {
				Name  string            `json:"name"`
				Theme map[string]string `json:"theme"`
			} `json:"design"`
			Frame struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Device string `json:"device"`
			} `json:"frame"`
			Selection Node                `json:"selection"`
			Parent    any                 `json:"parent"`
			Siblings  []map[string]string `json:"siblings"`
			Ancestors []map[string]string `json:"ancestors"`
			Guide     []string            `json:"guide"`
		}{
			Design: struct {
				Name  string            `json:"name"`
				Theme map[string]string `json:"theme"`
			}{doc.Name, doc.Theme},
			Frame: struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Device string `json:"device"`
			}{frame.ID, frame.Name, frame.Device},
			Selection: *selected,
			Parent:    parentPayload,
			Siblings:  siblings,
			Ancestors: ancestors,
			Guide:     contextGuide,
		}

	default:
		return nil, nil, fmt.Errorf("%w: scope kind must be design, frame or layer", ErrInvalid)
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	if len(raw) > 400*1024 {
		return nil, nil, fmt.Errorf("%w: the selected context is too large; choose a frame or a layer", ErrInvalid)
	}
	return raw, allowed, nil
}

var contextGuide = []string{
	"Only node ids in this JSON may be referenced.",
	"An addLayer node may omit ids; the studio assigns them.",
	"Containers that may receive children: " + strings.Join(ContainerTypes, ", ") + ".",
	"Addable layer types: " + strings.Join(AddableTypes, ", ") + ".",
}

func findFrame(doc Document, id string) (*Frame, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: a frame id is required for this scope", ErrInvalid)
	}
	for frameIndex := range doc.Frames {
		if doc.Frames[frameIndex].ID == id {
			return &doc.Frames[frameIndex], nil
		}
	}
	return nil, fmt.Errorf("%w: frame %s is not in this design", ErrInvalid, id)
}

// findNode returns a copy of the selected subtree so callers cannot mutate the
// document view while building context.
func findNode(frame *Frame, id string) (*Node, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("%w: a node id is required for layer scope", ErrInvalid)
	}
	var found *Node
	var visit func(Node) bool
	visit = func(node Node) bool {
		if node.ID == id {
			copy := node
			found = &copy
			return true
		}
		for _, child := range node.Children {
			if visit(child) {
				return true
			}
		}
		return false
	}
	if !visit(frame.Spec) || found == nil {
		return nil, fmt.Errorf("%w: layer %s is not in frame %s", ErrInvalid, id, frame.ID)
	}
	return found, nil
}
