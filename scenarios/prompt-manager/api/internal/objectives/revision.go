package objectives

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// ComputeMeaningRevision digests the parts of an objective that change what a
// serving team is obliged to do. Global order is deliberately excluded so that
// reordering never invalidates an acknowledgement.
func ComputeMeaningRevision(o Objective) string {
	return digest([]string{
		strings.ToUpper(strings.TrimSpace(o.ID)),
		normalizeText(o.Title),
		strings.ToLower(strings.TrimSpace(string(o.Class))),
		normalizeText(o.EvidenceSource),
		strings.ToLower(strings.TrimSpace(o.GapMarker)),
	})
}

// ComputeAttachmentRevision digests a team's ordered objective links. Role,
// coverage and note are part of the attachment meaning; priority is part of the
// order, so a pure reorder changes this revision but never a meaning revision.
func ComputeAttachmentRevision(atts []Attachment) string {
	ordered := append([]Attachment(nil), atts...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		return ordered[i].ObjectiveID < ordered[j].ObjectiveID
	})
	parts := make([]string, 0, len(ordered))
	for _, a := range ordered {
		parts = append(parts, strings.Join([]string{
			strings.ToUpper(strings.TrimSpace(a.ObjectiveID)),
			strings.ToLower(strings.TrimSpace(a.Role)),
			strings.ToLower(strings.TrimSpace(a.Coverage)),
			normalizeText(a.Note),
		}, "/"))
	}
	return digest(parts)
}

func digest(parts []string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:12]
}

func normalizeText(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
