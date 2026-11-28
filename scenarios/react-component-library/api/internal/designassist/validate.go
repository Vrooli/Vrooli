package designassist

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	layerIDPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$`)
	propKeyPattern  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]{0,39}$`)
	hexColour       = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)
	namedColour     = regexp.MustCompile(`^[a-zA-Z]{3,20}$`)
	maxSummaryRunes = 600
	maxTextRunes    = 2000
	maxOperations   = 8
	maxDocumentNode = 500
	maxDepth        = 12
)

// Validate checks a proposal against the document it will be applied to.
//
// It simulates the patch in order on a private copy, so a later operation that
// depends on an earlier removal is rejected here rather than breaking the
// client halfway through an apply. The client re-validates against its live
// tree before mutating anything.
func Validate(doc Document, allowed map[string]struct{}, proposal Proposal) error {
	if strings.TrimSpace(proposal.Summary) == "" || utf8.RuneCountInString(proposal.Summary) > maxSummaryRunes {
		return fmt.Errorf("%w: summary must be 1 to %d characters", ErrInvalid, maxSummaryRunes)
	}
	if len(proposal.Operations) < 1 || len(proposal.Operations) > maxOperations {
		return fmt.Errorf("%w: a proposal needs 1 to %d operations", ErrInvalid, maxOperations)
	}
	index, err := indexDocument(doc)
	if err != nil {
		return err
	}
	totalNodes := len(index.byID)
	if totalNodes > maxDocumentNode {
		return fmt.Errorf("%w: this design is too large to patch safely", ErrInvalid)
	}

	kindSet := map[string]bool{}
	for _, kind := range OperationKinds {
		kindSet[kind] = true
	}
	addable := map[string]bool{}
	for _, kind := range AddableTypes {
		addable[kind] = true
	}
	pending := 0

	for position, operation := range proposal.Operations {
		fail := func(format string, args ...any) error {
			return fmt.Errorf("%w: operation %d (%s): %s", ErrInvalid, position+1, operation.Kind, fmt.Sprintf(format, args...))
		}
		if !kindSet[operation.Kind] {
			return fail("unknown kind")
		}
		switch operation.Kind {
		case "setText":
			node, err := resolve(index, allowed, operation.NodeID)
			if err != nil {
				return fail("%s", err)
			}
			if operation.Text == nil {
				return fail("text is required")
			}
			if !isTextLayer(node.Type) {
				return fail("layer type %s does not render text; use setProp", node.Type)
			}
			if err := validateCopy(*operation.Text); err != nil {
				return fail("%s", err)
			}
			node.Props["text"] = *operation.Text

		case "setProp":
			node, err := resolve(index, allowed, operation.NodeID)
			if err != nil {
				return fail("%s", err)
			}
			if err := validateProp(node, operation.Key, operation.Value); err != nil {
				return fail("%s", err)
			}
			node.Props[operation.Key] = operation.Value

		case "setTheme":
			if !contains(PaletteKeys, operation.Key) {
				return fail("theme key %q is not a palette token", operation.Key)
			}
			value, ok := operation.Value.(string)
			if !ok || !(hexColour.MatchString(value) || namedColour.MatchString(value)) {
				return fail("theme value must be a hex colour or a colour name")
			}

		case "addLayer":
			parent, err := resolve(index, allowed, operation.ParentID)
			if err != nil {
				return fail("%s", err)
			}
			if !isContainer(parent.Type) {
				return fail("layer type %s cannot own children", parent.Type)
			}
			if operation.Node == nil {
				return fail("node is required")
			}
			if err := validateNewNode(*operation.Node, addable); err != nil {
				return fail("%s", err)
			}
			added := countNewNodes(*operation.Node)
			if totalNodes+added > maxDocumentNode {
				return fail("the design would exceed %d layers", maxDocumentNode)
			}
			if nodeDepth(parent)+1+newNodeDepth(*operation.Node) > maxDepth {
				return fail("the layer would nest deeper than %d levels", maxDepth)
			}
			totalNodes += added
			pending++
			built := buildNewTreeNode(*operation.Node, parent, pending)
			parent.Children = append(parent.Children, built)

		case "removeLayer":
			node, err := resolve(index, allowed, operation.NodeID)
			if err != nil {
				return fail("%s", err)
			}
			if err := removeNode(index, node); err != nil {
				return fail("%s", err)
			}
			totalNodes--

		case "moveLayer":
			node, err := resolve(index, allowed, operation.NodeID)
			if err != nil {
				return fail("%s", err)
			}
			if node.Parent == nil {
				return fail("the page root cannot be moved")
			}
			parent, err := resolve(index, allowed, operation.ParentID)
			if err != nil {
				return fail("%s", err)
			}
			if !isContainer(parent.Type) {
				return fail("layer type %s cannot own children", parent.Type)
			}
			if parent == node || isDescendant(parent, node) {
				return fail("a layer cannot move into itself")
			}
			if err := moveNode(index, node, parent, operation.Index); err != nil {
				return fail("%s", err)
			}
		}
	}
	return nil
}

func resolve(index *documentIndex, allowed map[string]struct{}, id string) (*treeNode, error) {
	if !layerIDPattern.MatchString(id) {
		return nil, fmt.Errorf("layer id %q is not a valid id", id)
	}
	if _, ok := allowed[id]; !ok {
		return nil, fmt.Errorf("layer %s is outside the selected context", id)
	}
	node := index.byID[id]
	if node == nil {
		return nil, fmt.Errorf("layer %s does not exist", id)
	}
	return node, nil
}

func validateProp(node *treeNode, key string, value any) error {
	if !propKeyPattern.MatchString(key) {
		return fmt.Errorf("prop key %q is not allowed", key)
	}
	switch key {
	case "id", "children", "src", "href", "role", "accepts":
		return fmt.Errorf("prop %q cannot be set by a proposal", key)
	}
	if strings.HasPrefix(key, "on") || strings.HasPrefix(key, "_") {
		return fmt.Errorf("prop %q cannot be set by a proposal", key)
	}
	if key == "x" || key == "y" || key == "w" || key == "h" {
		if node.Type != "slot" {
			return fmt.Errorf("placement prop %q is only valid on a slot", key)
		}
	}
	return validateScalar(key, value)
}

func validateScalar(key string, value any) error {
	switch typed := value.(type) {
	case nil:
		return fmt.Errorf("prop %q needs a value", key)
	case string:
		if err := validateCopy(typed); err != nil {
			return fmt.Errorf("prop %q: %s", key, err)
		}
	case bool:
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Abs(typed) > 100_000 {
			return fmt.Errorf("prop %q is out of range", key)
		}
	case int:
	case int64:
	default:
		return fmt.Errorf("prop %q must be a string, number or boolean", key)
	}
	return nil
}

func validateCopy(text string) error {
	if utf8.RuneCountInString(text) > maxTextRunes {
		return fmt.Errorf("copy exceeds %d characters", maxTextRunes)
	}
	for _, r := range text {
		if r == '\n' || r == '\t' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("copy contains a control character")
		}
	}
	return nil
}

func validateNewNode(node NewNode, addable map[string]bool) error {
	if !addable[node.Type] {
		return fmt.Errorf("layer type %q cannot be added by a proposal", node.Type)
	}
	if len(node.Props) > 16 {
		return fmt.Errorf("a new layer may carry at most 16 props")
	}
	for key, value := range node.Props {
		if !propKeyPattern.MatchString(key) {
			return fmt.Errorf("prop key %q is not allowed", key)
		}
		switch key {
		case "id", "children", "src", "href", "role", "accepts":
			return fmt.Errorf("prop %q cannot be set by a proposal", key)
		}
		if strings.HasPrefix(key, "on") || strings.HasPrefix(key, "_") {
			return fmt.Errorf("prop %q cannot be set by a proposal", key)
		}
		if key == "x" || key == "y" || key == "w" || key == "h" {
			return fmt.Errorf("placement props are only valid on a slot")
		}
		if err := validateScalar(key, value); err != nil {
			return err
		}
	}
	if len(node.Children) > 6 {
		return fmt.Errorf("a new layer may carry at most 6 children")
	}
	for _, child := range node.Children {
		if len(child.Children) > 0 {
			return fmt.Errorf("a new layer may only nest one level")
		}
		if err := validateNewNode(child, addable); err != nil {
			return err
		}
	}
	return nil
}

func buildNewTreeNode(node NewNode, parent *treeNode, sequence int) *treeNode {
	built := &treeNode{Type: node.Type, Props: map[string]any{}, Parent: parent}
	built.ID = fmt.Sprintf("pending-%d", sequence)
	for key, value := range node.Props {
		built.Props[key] = value
	}
	for childIndex, child := range node.Children {
		next := &treeNode{Type: child.Type, Props: map[string]any{}, Parent: built}
		next.ID = fmt.Sprintf("pending-%d-%d", sequence, childIndex)
		for key, value := range child.Props {
			next.Props[key] = value
		}
		built.Children = append(built.Children, next)
	}
	return built
}

func removeNode(index *documentIndex, node *treeNode) error {
	if node.Parent == nil {
		return fmt.Errorf("the page root cannot be removed")
	}
	if node.Type == "slot" {
		return fmt.Errorf("component slots are structural and cannot be removed")
	}
	parent := node.Parent
	for position, child := range parent.Children {
		if child == node {
			parent.Children = append(parent.Children[:position], parent.Children[position+1:]...)
			delete(index.byID, node.ID)
			return nil
		}
	}
	return fmt.Errorf("layer %s is not attached to its parent", node.ID)
}

func moveNode(index *documentIndex, node, parent *treeNode, position *int) error {
	oldParent := node.Parent
	for at, child := range oldParent.Children {
		if child == node {
			oldParent.Children = append(oldParent.Children[:at], oldParent.Children[at+1:]...)
			break
		}
	}
	at := len(parent.Children)
	if position != nil {
		at = *position
		if at < 0 || at > len(parent.Children) {
			return fmt.Errorf("index %d is outside the destination", at)
		}
	}
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[at+1:], parent.Children[at:])
	parent.Children[at] = node
	node.Parent = parent
	return nil
}

func isDescendant(candidate, ancestor *treeNode) bool {
	for cursor := candidate.Parent; cursor != nil; cursor = cursor.Parent {
		if cursor == ancestor {
			return true
		}
	}
	return false
}

func nodeDepth(node *treeNode) int {
	depth := 0
	for cursor := node; cursor != nil; cursor = cursor.Parent {
		depth++
	}
	return depth
}

func newNodeDepth(node NewNode) int {
	if len(node.Children) > 0 {
		return 2
	}
	return 1
}

func countNewNodes(node NewNode) int {
	total := 1
	for _, child := range node.Children {
		total += countNewNodes(child)
	}
	return total
}

func isTextLayer(kind string) bool {
	return kind == "text" || kind == "button" || kind == "chip"
}

func isContainer(kind string) bool {
	return contains(ContainerTypes, kind)
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
