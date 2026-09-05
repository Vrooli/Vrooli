//go:build windows

package hostfs

import "io/fs"

// ownedByCurrentUser always reports true on Windows.
//
// The unix check exists because /tmp is a single world-writable directory
// shared by every user and system service on the host. Windows has no such
// shared location: os.TempDir resolves %TMP%/%TEMP%, which is per-user under
// the profile directory, and os.UserCacheDir resolves %LocalAppData%, likewise
// per-user. The entries a walk can reach are already the current user's, so
// there is nothing for an ownership filter to exclude.
//
// Implementing a real check would mean reading the security descriptor for
// every entry and comparing its owner SID against the process token — a
// meaningful per-entry cost, plus a new syscall surface, to answer a question
// whose answer is structurally always yes. If a future provider is ever pointed
// at a genuinely shared Windows path, this is the function to implement rather
// than a new check bolted on elsewhere.
func ownedByCurrentUser(fs.FileInfo) bool { return true }
