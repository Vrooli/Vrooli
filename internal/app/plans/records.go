package plans

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
)

func filterArchived(records []PlanRecord, includeArchived bool) []PlanRecord {
	filtered := make([]PlanRecord, 0, len(records))
	for _, record := range records {
		if record.Archived && !includeArchived {
			continue
		}
		filtered = append(filtered, record)
	}
	sortRecords(filtered)
	return filtered
}

func sortRecords(records []PlanRecord) {
	slices.SortFunc(records, func(a, b PlanRecord) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.After(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func titleFromSlug(slug string) string {
	words := strings.Split(slug, "-")
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}
