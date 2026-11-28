// Package buildidentity exposes the immutable source identity embedded in the
// Program Runtime API binary by the scenario lifecycle build owner.
package buildidentity

// SourceIdentity is populated by the scenario build driver. Test and manually
// built binaries retain an explicit unknown value.
var SourceIdentity = "unknown"
