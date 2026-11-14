package androidadb

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"strings"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestSemanticSnapshotCapturesControlsAndRedactsEditableValues(t *testing.T) {
	tree := []byte(`<hierarchy><node class="android.widget.Button" resource-id="com.example:id/save" text="Save" clickable="true" enabled="true" bounds="[10,20][90,70]"/><node class="android.widget.EditText" text="secret" content-desc="private" password="true" enabled="true" bounds="[1,2][80,40]"/></hierarchy>`)
	snapshot, err := parseSemanticSnapshot(tree)
	require.NoError(t, err)
	require.Len(t, snapshot.Elements, 2)
	require.Equal(t, "Save", snapshot.Elements[0].Text)
	require.Equal(t, [4]int{10, 20, 90, 70}, snapshot.Elements[0].Bounds)
	require.True(t, snapshot.Elements[0].Clickable)
	require.True(t, snapshot.Elements[1].TextRedacted)
	require.Empty(t, snapshot.Elements[1].Text)
	require.Empty(t, snapshot.Elements[1].Description)
}

func TestSemanticSnapshotRetainsReportedSelectionAndCheckState(t *testing.T) {
	tree := []byte(`<hierarchy><node text="Selected" selected="true" checkable="true" checked="false" bounds="[0,0][10,10]"/><node text="Unknown" bounds="[0,10][10,20]"/><node text="Not checkable" checkable="false" checked="false" bounds="[0,20][10,30]"/></hierarchy>`)
	snapshot, err := parseSemanticSnapshot(tree)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Elements[0].Selected)
	require.True(t, *snapshot.Elements[0].Selected)
	require.NotNil(t, snapshot.Elements[0].Checked)
	require.False(t, *snapshot.Elements[0].Checked)
	require.Nil(t, snapshot.Elements[1].Selected)
	require.Nil(t, snapshot.Elements[1].Checked)
	require.Nil(t, snapshot.Elements[2].Checked)
}

func TestSemanticSnapshotBoundsOutputAndMarksTruncation(t *testing.T) {
	var tree strings.Builder
	tree.WriteString("<hierarchy>")
	for i := 0; i < maxSemanticSnapshotNodes+1; i++ {
		fmt.Fprintf(&tree, `<node text="%s" enabled="true" bounds="[0,0][10,10]"/>`, strings.Repeat("x", maxSemanticLabelRunes+1))
	}
	tree.WriteString("</hierarchy>")
	snapshot, err := parseSemanticSnapshot([]byte(tree.String()))
	require.NoError(t, err)
	require.True(t, snapshot.Truncated)
	require.Len(t, snapshot.Elements, maxSemanticSnapshotNodes)
	require.Len(t, []rune(snapshot.Elements[0].Text), maxSemanticLabelRunes)
	single := fmt.Sprintf(`<hierarchy><node text="%s" enabled="true" bounds="[0,0][10,10]"/></hierarchy>`, strings.Repeat("x", maxSemanticLabelRunes+1))
	snapshot, err = parseSemanticSnapshot([]byte(single))
	require.NoError(t, err)
	require.True(t, snapshot.Truncated, "truncated labels must be disclosed even if no nodes are omitted")
}

func TestSemanticSnapshotRefusesMissingOrMalformedHierarchy(t *testing.T) {
	for _, raw := range []string{"ERROR: could not get idle state.", `<hierarchy><node bounds="[0,0][1,1]">`} {
		_, err := parseSemanticSnapshot([]byte(raw))
		require.Error(t, err)
	}
}

func TestSemanticTargetRejectsAmbiguousMatches(t *testing.T) {
	tree := []byte(`<hierarchy><node text="Open" bounds="[0,0][20,20]"/><node text="Open" bounds="[0,30][20,50]"/></hierarchy>`)
	_, err := findSemanticNode(tree, "Open")
	require.ErrorContains(t, err, "ambiguous")
}

func TestSemanticTargetResolvesAccessibilityBoundsAndTapsCenter(t *testing.T) {
	var screenshot bytes.Buffer
	require.NoError(t, png.Encode(&screenshot, image.NewRGBA(image.Rect(0, 0, 100, 200))))
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out uiautomator dump /dev/stdout": []byte(`<hierarchy><node resource-id="com.example:id/hello-mobile-input" bounds="[10,20][90,180]" /></hierarchy>`),
		"adb -s serial-1 exec-out screencap -p":                 screenshot.Bytes(),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	match, err := adapter.ResolveSemantic(context.Background(), "hello-mobile-input")
	require.NoError(t, err)
	require.Equal(t, []float64{.1, .1, .9, .9}, match.Bounds)
	require.Equal(t, 1.0, match.Confidence)
	require.NotContains(t, strings.Join(runner.calls, "\n"), "/sdcard/")
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-target", Value: "hello-mobile-input"}), "resolver-scoped")
}

func TestSemanticAssertVerifiesAccessibilityText(t *testing.T) {
	runner := &scriptedRunner{responses: map[string][]byte{
		"adb -s serial-1 exec-out uiautomator dump /dev/stdout": []byte(`<hierarchy><node resource-id="com.example:id/state" text="Connectivity: offline" bounds="[0,0][100,40]" /></hierarchy>`),
	}}
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: offline"}))
	require.ErrorContains(t, adapter.Actuate(context.Background(), strategy.Actuation{Action: "semantic-assert", Value: "state", Expected: "Connectivity: online"}), "expected text")
}
