//go:build linux

package hostpaths

import (
	"os"
	"path/filepath"
	"strings"
)

// trashRoots resolves the freedesktop.org trash directories.
//
// The spec puts the home trash at $XDG_DATA_HOME/Trash, defaulting to
// ~/.local/share/Trash, and splits it in two: the payloads live under files/
// and one .trashinfo metadata record per payload lives under info/.
//
// Both subdirectories are returned rather than the Trash directory itself, and
// the distinction matters because the trash provider treats each immediate
// child of a root as one cleanup unit. Rooted at Trash/, that yields exactly
// two units — all of files/ and all of info/ — so a single recently-trashed
// item would hold the entire trash above the age threshold. This host's trash
// is 99 GB across 5,064,705 files; collapsing it into two all-or-nothing units
// would make it effectively unreclaimable. Rooted at the subdirectories, each
// trashed item is its own unit and ages on its own.
//
// The payload and its metadata record are still removed separately. In practice
// they share a modification time, since both are created when the item is
// trashed, so any age threshold selects the pair together. A stale .trashinfo
// is a few hundred bytes and is ignored by file managers whose payload is gone,
// so the worst case of them diverging is cosmetic rather than corrupting.
//
// Per-volume trash directories (<mount>/.Trash-$uid) are deliberately not
// resolved here. Enumerating them means enumerating mount points, a materially
// larger surface than the home trash, and they were not implicated in the
// incident this cleanup path exists for.
func trashRoots() []string {
	base := ""
	if dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dataHome != "" {
		base = filepath.Join(dataHome, "Trash")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		base = filepath.Join(home, ".local", "share", "Trash")
	}
	return []string{
		filepath.Join(base, "files"),
		filepath.Join(base, "info"),
	}
}
