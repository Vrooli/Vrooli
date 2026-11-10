package androidadb

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

func (a *Adapter) ResolveSemantic(ctx context.Context, target string) (strategy.SemanticResult, error) {
	tree, err := a.runner.Run(ctx, "adb", a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	values, err := findSemanticNode(tree, target)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	left, top, right, bottom, err := semanticBoundsRect(values, target)
	if err != nil {
		return strategy.SemanticResult{}, err
	}
	frame, err := a.Observe(ctx)
	if err != nil || frame.Width <= 0 || frame.Height <= 0 {
		if err == nil {
			err = fmt.Errorf("semantic resolution requires positive display dimensions")
		}
		return strategy.SemanticResult{}, err
	}
	return strategy.SemanticResult{
		Bounds:     []float64{float64(left) / float64(frame.Width), float64(top) / float64(frame.Height), float64(right) / float64(frame.Width), float64(bottom) / float64(frame.Height)},
		Confidence: 1,
	}, nil
}

var boundsPattern = regexp.MustCompile(`^\[(\d+),(\d+)\]\[(\d+),(\d+)\]$`)

const maxSemanticSnapshotNodes = 200

const maxSemanticLabelRunes = 160

// parseSemanticSnapshot returns a compact, bounded view without retaining raw
// accessibility XML. Editable values and password descriptions are omitted.
func parseSemanticSnapshot(tree []byte) (strategy.SemanticSnapshot, error) {
	result := strategy.SemanticSnapshot{Elements: []strategy.SemanticElement{}}
	label := func(value string) string {
		runes := []rune(strings.TrimSpace(value))
		if len(runes) > maxSemanticLabelRunes {
			result.Truncated = true
			runes = runes[:maxSemanticLabelRunes]
		}
		return string(runes)
	}
	optionalBool := func(value string) *bool {
		switch value {
		case "true":
			v := true
			return &v
		case "false":
			v := false
			return &v
		default:
			return nil
		}
	}
	decoder := xml.NewDecoder(bytes.NewReader(tree))
	depth, root := 0, false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, fmt.Errorf("parse accessibility hierarchy: %w", err)
		}
		switch element := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root || element.Name.Local != "hierarchy" {
					return result, errors.New("accessibility hierarchy root unavailable")
				}
				root = true
			}
			depth++
			if element.Name.Local != "node" {
				continue
			}
			attrs := make(map[string]string, len(element.Attr))
			for _, attr := range element.Attr {
				attrs[attr.Name.Local] = attr.Value
			}
			x0, y0, x1, y1, err := semanticBoundsRect(attrs, "node")
			if err != nil || x1 <= x0 || y1 <= y0 {
				continue
			}
			editable := attrs["password"] == "true" || attrs["editable"] == "true" || strings.HasSuffix(attrs["class"], ".EditText")
			node := strategy.SemanticElement{ResourceID: label(attrs["resource-id"]), Class: label(attrs["class"]), Bounds: [4]int{x0, y0, x1, y1}, Clickable: attrs["clickable"] == "true", Focusable: attrs["focusable"] == "true", Focused: attrs["focused"] == "true", Scrollable: attrs["scrollable"] == "true", Enabled: attrs["enabled"] == "true", Editable: editable}
			node.Selected = optionalBool(attrs["selected"])
			if attrs["checkable"] == "true" {
				node.Checked = optionalBool(attrs["checked"])
			}
			if editable {
				node.TextRedacted = attrs["text"] != "" || attrs["content-desc"] != ""
			} else {
				node.Text, node.Description = label(attrs["text"]), label(attrs["content-desc"])
			}
			if node.ResourceID == "" && node.Text == "" && node.Description == "" && !node.Clickable && !node.Focusable && !node.Scrollable && !node.Editable {
				continue
			}
			if len(result.Elements) >= maxSemanticSnapshotNodes {
				result.Truncated = true
				continue
			}
			result.Elements = append(result.Elements, node)
		case xml.EndElement:
			depth--
		}
	}
	if !root || depth != 0 {
		return result, errors.New("accessibility hierarchy is unavailable or incomplete")
	}
	result.ObservedAt = time.Now().UTC()
	return result, nil
}

func (a *Adapter) ObserveSemantics(ctx context.Context) (strategy.SemanticSnapshot, error) {
	bounded, ok := a.runner.(boundedRunner)
	if !ok {
		return strategy.SemanticSnapshot{}, errors.New("bounded accessibility observation is unavailable")
	}
	data, err := bounded.RunBounded(ctx, "adb", 2*1024*1024, a.args("exec-out", "uiautomator", "dump", "/dev/stdout")...)
	if err != nil {
		return strategy.SemanticSnapshot{}, fmt.Errorf("capture accessibility hierarchy: %w", err)
	}
	return parseSemanticSnapshot(data)
}

// findSemanticNode resolves a stable Android accessibility identifier. The XML
// is device-owned input; only matched attributes and numeric bounds are used.
func findSemanticNode(tree []byte, target string) (map[string]string, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("accessibility target is required")
	}
	decoder := xml.NewDecoder(bytes.NewReader(tree))
	var match map[string]string
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse accessibility hierarchy: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "node" {
			continue
		}
		values := make(map[string]string, len(start.Attr))
		for _, attr := range start.Attr {
			values[attr.Name.Local] = attr.Value
		}
		if !semanticValueMatches(values, target) {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("accessibility target %q is ambiguous in the view hierarchy", target)
		}
		match = values
	}
	if match != nil {
		return match, nil
	}
	return nil, fmt.Errorf("accessibility target %q was not found in the view hierarchy", target)
}

func semanticBounds(values map[string]string, target string) (int, int, error) {
	x0, y0, x1, y1, err := semanticBoundsRect(values, target)
	if err != nil {
		return 0, 0, err
	}
	return (x0 + x1) / 2, (y0 + y1) / 2, nil
}

func semanticBoundsRect(values map[string]string, target string) (int, int, int, int, error) {
	match := boundsPattern.FindStringSubmatch(values["bounds"])
	if len(match) != 5 {
		return 0, 0, 0, 0, fmt.Errorf("accessibility target %q has invalid bounds", target)
	}
	x0, _ := strconv.Atoi(match[1])
	y0, _ := strconv.Atoi(match[2])
	x1, _ := strconv.Atoi(match[3])
	y1, _ := strconv.Atoi(match[4])
	return x0, y0, x1, y1, nil
}

func semanticValueMatches(values map[string]string, target string) bool {
	for _, key := range []string{"resource-id", "text", "content-desc", "hint-text"} {
		value := values[key]
		if value == target || strings.HasSuffix(value, "/"+target) || strings.HasSuffix(value, ":"+target) {
			return true
		}
	}
	return false
}
