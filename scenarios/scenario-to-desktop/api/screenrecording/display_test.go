package screenrecording

import (
	"os/exec"
	"testing"
)

func TestParseDisplayNumber(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{":99", 99, false},
		{":0", 0, false},
		{":1", 1, false},
		{"abc", 0, true},
		{":", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDisplayNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseDisplayNumber(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ParseDisplayNumber(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNewDisplayManager(t *testing.T) {
	dm := NewDisplayManager()
	if dm == nil {
		t.Fatal("NewDisplayManager returned nil")
	}
}

func TestStartWindowManager_NoWMInstalled(t *testing.T) {
	// Save and override wmCandidates with commands that definitely don't exist.
	origCandidates := wmCandidates
	wmCandidates = []struct {
		name string
		args []string
	}{
		{"__nonexistent_wm_a__", nil},
		{"__nonexistent_wm_b__", nil},
	}
	defer func() { wmCandidates = origCandidates }()

	// Should return nil without error — best-effort.
	proc := startWindowManager(":99")
	if proc != nil {
		t.Fatal("expected nil process when no WM is installed")
	}
}

func TestStartWindowManager_WithAvailableWM(t *testing.T) {
	// Use "sleep" as a fake WM to verify process lifecycle.
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	origCandidates := wmCandidates
	wmCandidates = []struct {
		name string
		args []string
	}{
		{"sleep", []string{"60"}},
	}
	defer func() { wmCandidates = origCandidates }()

	proc := startWindowManager(":99")
	if proc == nil {
		t.Fatal("expected non-nil process")
	}
	// Clean up the sleep process.
	_ = proc.Kill()
	_, _ = proc.Wait()
}

func TestWmCandidates_DefaultsConfigured(t *testing.T) {
	// Verify the default WM candidates list is non-empty and has expected entries.
	if len(wmCandidates) < 2 {
		t.Fatalf("expected at least 2 WM candidates, got %d", len(wmCandidates))
	}
	if wmCandidates[0].name != "openbox" {
		t.Errorf("first WM candidate should be openbox, got %q", wmCandidates[0].name)
	}
	if wmCandidates[1].name != "matchbox-window-manager" {
		t.Errorf("second WM candidate should be matchbox-window-manager, got %q", wmCandidates[1].name)
	}
}
