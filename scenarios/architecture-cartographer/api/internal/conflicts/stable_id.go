package conflicts

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// StableID returns the deterministic content-hash identity for a
// conflict. Two runs that produce the same underlying drift produce the
// same StableID; an unrelated drift produces a different one.
//
// Inputs to the hash are intentionally minimal: scenario, detector,
// type, subtype, sorted locations, sorted domains. Severity, finding class,
// evidence, suggested fixes, and timestamps are excluded — they may evolve
// while the underlying drift is "the same thing".
//
// The output is "csid:" + first 16 hex chars of sha256(canonical).
// Short enough to read in a CLI; long enough (64 bits) to avoid
// collisions across one repo's lifetime.
func StableID(c Conflict) string {
	locs := append([]string(nil), c.Locations...)
	sort.Strings(locs)
	doms := append([]string(nil), c.Domains...)
	sort.Strings(doms)
	var b strings.Builder
	b.WriteString(c.Scenario)
	b.WriteByte('\x1f')
	b.WriteString(c.Detector)
	b.WriteByte('\x1f')
	b.WriteString(c.Type)
	b.WriteByte('\x1f')
	b.WriteString(c.Subtype)
	b.WriteByte('\x1f')
	b.WriteString(strings.Join(locs, ","))
	b.WriteByte('\x1f')
	b.WriteString(strings.Join(doms, ","))
	sum := sha256.Sum256([]byte(b.String()))
	return "csid:" + hex.EncodeToString(sum[:8])
}
