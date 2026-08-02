//go:build windows

package hostpaths

// trashRoots reports no trash root on Windows, deliberately.
//
// The Recycle Bin is not an ordinary directory that may be emptied by deleting
// paths. It is a per-volume, SID-scoped store ($Recycle.Bin\<SID>) whose entries
// are split into a metadata record and a payload, and whose canonical lifecycle
// runs through the shell API (SHEmptyRecycleBin / IFileOperation). Removing
// those files by path corrupts the bin's bookkeeping and can leave the user
// unable to restore or empty it through Explorer.
//
// Returning nothing is therefore the correct behaviour rather than a gap: the
// trash provider simply reports no reclaimable items on Windows, while the
// temp and cache providers — which are ordinary directories on every platform —
// work normally. Emptying the Recycle Bin safely would require the shell API
// and a genuine owner-approved provider, not a filesystem walk.
func trashRoots() []string { return nil }
