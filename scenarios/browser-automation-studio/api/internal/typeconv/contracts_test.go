package typeconv

import (
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestToAssertionOutcome(t *testing.T) {
	t.Run("pointer to AssertionOutcome", func(t *testing.T) {
		input := &contracts.AssertionOutcome{
			Mode:     "exists",
			Selector: ".test",
			Success:  true,
		}
		result := ToAssertionOutcome(input)
		if result != input {
			t.Errorf("expected same pointer")
		}
	})

	t.Run("AssertionOutcome value", func(t *testing.T) {
		input := contracts.AssertionOutcome{
			Mode:    "contains",
			Message: "test message",
			Success: true,
		}
		result := ToAssertionOutcome(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Mode != input.Mode || result.Message != input.Message {
			t.Errorf("values not preserved")
		}
	})

	t.Run("map conversion", func(t *testing.T) {
		input := map[string]any{
			"mode":          "equals",
			"selector":      "#element",
			"message":       "test",
			"success":       true,
			"negated":       false,
			"caseSensitive": true,
			"expected":      "value",
			"actual":        "value",
		}
		result := ToAssertionOutcome(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Mode != "equals" || result.Selector != "#element" {
			t.Errorf("conversion failed")
		}
		if !result.Success || result.Negated || !result.CaseSensitive {
			t.Errorf("boolean fields incorrect")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := ToAssertionOutcome("invalid")
		if result != nil {
			t.Errorf("expected nil for invalid type")
		}
	})
}

func TestToHighlightRegions(t *testing.T) {
	t.Run("[]HighlightRegion", func(t *testing.T) {
		input := []contracts.HighlightRegion{
			{Selector: ".test", Color: "red", Padding: 5},
		}
		result := ToHighlightRegions(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 region, got %d", len(result))
		}
		if result[0].Selector != ".test" {
			t.Errorf("selector not preserved")
		}
	})

	t.Run("[]map[string]any", func(t *testing.T) {
		input := []map[string]any{
			{"selector": ".elem1", "color": "blue", "padding": 10},
			{"selector": ".elem2", "color": "green", "padding": 5},
		}
		result := ToHighlightRegions(input)
		if len(result) != 2 {
			t.Fatalf("expected 2 regions, got %d", len(result))
		}
	})

	t.Run("[]any", func(t *testing.T) {
		input := []any{
			map[string]any{"selector": ".test", "color": "red", "padding": 3},
		}
		result := ToHighlightRegions(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 region, got %d", len(result))
		}
	})

	t.Run("invalid entries filtered", func(t *testing.T) {
		input := []any{
			map[string]any{"selector": ".valid", "color": "red"},
			map[string]any{"color": "blue"}, // no selector
			"invalid",
		}
		result := ToHighlightRegions(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 valid region, got %d", len(result))
		}
	})
}

func TestToHighlightRegion(t *testing.T) {
	t.Run("valid with selector", func(t *testing.T) {
		input := map[string]any{
			"selector": ".test",
			"color":    "red",
			"padding":  5,
		}
		result := ToHighlightRegion(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Selector != ".test" || result.Color != "red" || result.Padding != 5 {
			t.Errorf("field values incorrect")
		}
	})

	t.Run("valid with bounding box", func(t *testing.T) {
		input := map[string]any{
			"boundingBox": map[string]any{
				"x": 10.0, "y": 20.0, "width": 100.0, "height": 50.0,
			},
			"color": "blue",
		}
		result := ToHighlightRegion(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BoundingBox == nil {
			t.Error("expected bounding box")
		}
	})

	t.Run("missing selector and bbox", func(t *testing.T) {
		input := map[string]any{"color": "red"}
		result := ToHighlightRegion(input)
		if result != nil {
			t.Errorf("expected nil for missing selector and bbox")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := ToHighlightRegion("invalid")
		if result != nil {
			t.Errorf("expected nil for invalid type")
		}
	})
}

func TestToMaskRegions(t *testing.T) {
	t.Run("[]MaskRegion", func(t *testing.T) {
		input := []contracts.MaskRegion{
			{Selector: ".mask", Opacity: 0.5},
		}
		result := ToMaskRegions(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 region, got %d", len(result))
		}
	})

	t.Run("[]map[string]any", func(t *testing.T) {
		input := []map[string]any{
			{"selector": ".mask1", "opacity": 0.3},
			{"selector": ".mask2", "opacity": 0.7},
		}
		result := ToMaskRegions(input)
		if len(result) != 2 {
			t.Fatalf("expected 2 regions, got %d", len(result))
		}
	})
}

func TestToMaskRegion(t *testing.T) {
	t.Run("valid with selector", func(t *testing.T) {
		input := map[string]any{
			"selector": ".mask",
			"opacity":  0.5,
		}
		result := ToMaskRegion(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Selector != ".mask" || result.Opacity != 0.5 {
			t.Errorf("field values incorrect")
		}
	})

	t.Run("valid with bounding box", func(t *testing.T) {
		input := map[string]any{
			"boundingBox": map[string]any{
				"x": 5.0, "y": 10.0, "width": 50.0, "height": 25.0,
			},
			"opacity": 0.8,
		}
		result := ToMaskRegion(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BoundingBox == nil {
			t.Error("expected bounding box")
		}
	})

	t.Run("missing selector and bbox", func(t *testing.T) {
		input := map[string]any{"opacity": 0.5}
		result := ToMaskRegion(input)
		if result != nil {
			t.Errorf("expected nil for missing selector and bbox")
		}
	})
}

func TestToElementFocus(t *testing.T) {
	t.Run("valid with selector", func(t *testing.T) {
		input := map[string]any{"selector": ".focus"}
		result := ToElementFocus(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Selector != ".focus" {
			t.Errorf("selector incorrect")
		}
	})

	t.Run("valid with bounding box", func(t *testing.T) {
		input := map[string]any{
			"boundingBox": map[string]any{
				"x": 15.0, "y": 25.0, "width": 200.0, "height": 100.0,
			},
		}
		result := ToElementFocus(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.BoundingBox == nil {
			t.Error("expected bounding box")
		}
	})

	t.Run("missing selector and bbox", func(t *testing.T) {
		input := map[string]any{}
		result := ToElementFocus(input)
		if result != nil {
			t.Errorf("expected nil for empty map")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := ToElementFocus("invalid")
		if result != nil {
			t.Errorf("expected nil for invalid type")
		}
	})
}

func TestToBoundingBox(t *testing.T) {
	t.Run("valid bounding box", func(t *testing.T) {
		input := map[string]any{
			"x":      10.5,
			"y":      20.3,
			"width":  100.0,
			"height": 50.0,
		}
		result := ToBoundingBox(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.X != 10.5 || result.Y != 20.3 {
			t.Errorf("coordinates incorrect")
		}
		if result.Width != 100.0 || result.Height != 50.0 {
			t.Errorf("dimensions incorrect")
		}
	})

	t.Run("empty map", func(t *testing.T) {
		input := map[string]any{}
		result := ToBoundingBox(input)
		if result != nil {
			t.Errorf("expected nil for empty map")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := ToBoundingBox("invalid")
		if result != nil {
			t.Errorf("expected nil for invalid type")
		}
	})
}

func TestToPoint(t *testing.T) {
	t.Run("valid point", func(t *testing.T) {
		input := map[string]any{"x": 15.0, "y": 25.0}
		result := ToPoint(input)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.X != 15.0 || result.Y != 25.0 {
			t.Errorf("coordinates incorrect")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		result := ToPoint("invalid")
		if result != nil {
			t.Errorf("expected nil for invalid type")
		}
	})
}

func TestToPointSlice(t *testing.T) {
	t.Run("[]Point", func(t *testing.T) {
		input := []contracts.Point{{X: 1, Y: 2}, {X: 3, Y: 4}}
		result := ToPointSlice(input)
		if len(result) != 2 {
			t.Fatalf("expected 2 points, got %d", len(result))
		}
	})

	t.Run("[]*Point", func(t *testing.T) {
		p1, p2 := &contracts.Point{X: 1, Y: 2}, &contracts.Point{X: 3, Y: 4}
		input := []*contracts.Point{p1, p2, nil}
		result := ToPointSlice(input)
		if len(result) != 2 {
			t.Fatalf("expected 2 points (nil filtered), got %d", len(result))
		}
	})

	t.Run("[]map[string]any", func(t *testing.T) {
		input := []map[string]any{
			{"x": 10.0, "y": 20.0},
			{"x": 30.0, "y": 40.0},
		}
		result := ToPointSlice(input)
		if len(result) != 2 {
			t.Fatalf("expected 2 points, got %d", len(result))
		}
	})

	t.Run("[]any", func(t *testing.T) {
		input := []any{
			map[string]any{"x": 5.0, "y": 10.0},
			"invalid",
		}
		result := ToPointSlice(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 valid point, got %d", len(result))
		}
	})

	t.Run("single map", func(t *testing.T) {
		input := map[string]any{"x": 7.0, "y": 14.0}
		result := ToPointSlice(input)
		if len(result) != 1 {
			t.Fatalf("expected 1 point, got %d", len(result))
		}
	})
}
