package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestShadowBoardSettingsAreNamespacedAndDoNotTouchSourceCatalog(t *testing.T) {
	root := t.TempDir()
	t.Setenv("COMMAND_CENTER_CONFIG_DIR", "")
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "command-center_shadow")
	t.Setenv("VROOLI_VARIANT", "shadow")

	sourcePath := filepath.Join(catalogDir(), "board.json")
	sourceBefore, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read checked-in board defaults: %v", err)
	}

	path, err := boardSettingsPath()
	if err != nil {
		t.Fatalf("boardSettingsPath: %v", err)
	}
	wantPath := filepath.Join(root, "config", "vrooli", "command-center_shadow", "board.json")
	if path != wantPath {
		t.Fatalf("board settings path = %q, want isolated variant path %q", path, wantPath)
	}

	initial, err := readBoardSettings()
	if err != nil {
		t.Fatalf("read initial shadow settings: %v", err)
	}
	if initial.CycleSeconds != 60 || initial.Transition != "crossfade" || len(initial.Rooms) != 0 {
		t.Fatalf("initial shadow settings = %+v, want synthetic defaults", initial)
	}

	custom := BoardSettings{
		CycleSeconds: 45,
		Transition:   "cut",
		Rooms:        []BoardRoomMode{{ID: "mission-control", Enabled: true}},
	}
	if err := writeBoardSettings(custom); err != nil {
		t.Fatalf("write shadow settings: %v", err)
	}
	reloaded, err := readBoardSettings()
	if err != nil {
		t.Fatalf("reload shadow settings: %v", err)
	}
	if !reflect.DeepEqual(reloaded, custom) {
		t.Fatalf("reloaded settings = %+v, want %+v", reloaded, custom)
	}
	sourceAfter, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("re-read checked-in board defaults: %v", err)
	}
	if !reflect.DeepEqual(sourceAfter, sourceBefore) {
		t.Fatal("saving shadow settings modified checked-in board defaults")
	}
}
