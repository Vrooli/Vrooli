package clipboard

import "testing"

func TestToolNameReportsKnownValue(t *testing.T) {
	switch got := ToolName(); got {
	case "pbcopy", "xclip", "xsel", "wl-copy", "clip", "none":
	default:
		t.Fatalf("unexpected clipboard tool name: %q", got)
	}
}

func TestIsAvailableMatchesToolName(t *testing.T) {
	if ToolName() == "none" && IsAvailable() {
		t.Fatal("clipboard cannot be available when no tool is selected")
	}
}
