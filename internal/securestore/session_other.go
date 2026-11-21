//go:build !linux

package securestore

// sessionDiagnosis has no session-level failure mode outside Linux: the macOS
// Keychain and the Windows Credential Manager are reached through the user's
// own security context rather than through a session bus whose ownership can
// disagree with the process uid.
func sessionDiagnosis() string { return "" }

// sessionRepairNote has nothing to report for the same reason: there is no
// session variable to get wrong, so there is never a repair to disclose.
func sessionRepairNote() string { return "" }

// sessionRuntimeDir reports that this platform offers no session-scoped tmpfs
// for an open credential key. macOS and Windows have per-user temporary
// directories, but they are on durable storage and survive a logout, so an
// unlock here lasts exactly one process rather than leaving a data key behind
// on disk after the operator walks away.
func sessionRuntimeDir() (string, bool) { return "", false }
