package planning

import "fmt"

// EnsureLockedOccurrencesRetained rejects a draft that silently drops or
// replaces a locked slot. A user can explicitly unlock a retained slot before
// changing or clearing its meal.
func EnsureLockedOccurrencesRetained(current, proposed Draft) error {
	byKey := make(map[string]Occurrence, len(proposed.Occurrences))
	for _, occurrence := range proposed.Occurrences {
		byKey[occurrenceKey(occurrence)] = occurrence
	}
	for _, before := range current.Occurrences {
		if !before.Locked {
			continue
		}
		after, ok := byKey[occurrenceKey(before)]
		if !ok {
			return fmt.Errorf("locked slot %s %s must remain in the draft; unlock it explicitly before removing it", before.Date, slotName(before.SlotName))
		}
		if after.Locked && (after.RecipeID != before.RecipeID || after.RecipeRevision != before.RecipeRevision) {
			return fmt.Errorf("locked slot %s %s cannot change recipe while it remains locked", before.Date, slotName(before.SlotName))
		}
	}
	return nil
}

func occurrenceKey(occurrence Occurrence) string {
	return occurrence.Date + "\x00" + slotName(occurrence.SlotName)
}

func slotName(name string) string {
	if name == "" {
		return "meal"
	}
	return name
}
