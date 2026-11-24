//go:build !linux

package accel

// processDescendantPIDs has no portable implementation off Linux. Placement
// still matches the supervised pid and the artifact-prefix fallback; only
// child-process attribution is unavailable.
func processDescendantPIDs(int) map[int]struct{} { return nil }
