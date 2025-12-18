package export

import (
	basexport "github.com/vrooli/browser-automation-studio/services/export"
)

// Apply applies client-provided overrides to a movie spec, applying presets first,
// then explicit overrides, and finally synchronizing cursor-related fields across
// the spec's Cursor, Decor, and CursorMotion structures.
//
// This is a thin wrapper that delegates to services/export.Apply.
func Apply(spec *basexport.ReplayMovieSpec, overrides *Overrides) {
	basexport.Apply(spec, overrides)
}
