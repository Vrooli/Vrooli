package promptcatalog

import (
	"strconv"
	"strings"
)

func IsKnownSkillID(skillID string) bool {
	_, ok := skillEntry(skillID)
	return ok
}

func VariableKeysForSkill(skillID string) []string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return nil
	}
	return append([]string{}, entry.VariableKeys...)
}

func SkillUsageType(skillID string) UsageType {
	entry, ok := skillEntry(skillID)
	if !ok {
		return ""
	}
	return entry.UsageType
}

func SkillGroups(skillID string) []string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return nil
	}
	return []string{string(entry.Group)}
}

func SkillUsageCount(skillID string) int {
	entry, ok := skillEntry(skillID)
	if !ok {
		return 0
	}
	switch entry.UsageType {
	case UsageSupportReference:
		count := 0
		for _, candidate := range entries {
			if candidate.UsageType != UsageDirectRuntime {
				continue
			}
			if contains(candidate.ReferenceSkillIDs, entry.SkillID) {
				count++
			}
		}
		return count
	default:
		count := 0
		for _, candidate := range entries {
			if candidate.SourceType != SourceSkill || candidate.UsageType != UsageDirectRuntime {
				continue
			}
			if candidate.SkillID == entry.SkillID {
				count++
			}
		}
		return count
	}
}

func SkillImpactSummary(skillID string) string {
	entry, ok := skillEntry(skillID)
	if !ok {
		return "Not referenced by the prompt catalog."
	}
	count := SkillUsageCount(skillID)
	switch entry.UsageType {
	case UsageSupportReference:
		if count == 1 {
			return "Referenced by 1 runtime prompt path."
		}
		return "Referenced by " + itoa(count) + " runtime prompt paths."
	default:
		if count == 1 {
			return "Used directly by 1 runtime prompt path."
		}
		return "Used directly by " + itoa(count) + " runtime prompt paths."
	}
}

func skillEntry(skillID string) (Entry, bool) {
	normalized := strings.TrimSpace(skillID)
	for _, entry := range entries {
		if entry.SourceType != SourceSkill {
			continue
		}
		if entry.SkillID == normalized {
			return cloneEntry(entry), true
		}
	}
	return Entry{}, false
}

func cloneEntry(entry Entry) Entry {
	entry.BacklogKinds = append([]string{}, entry.BacklogKinds...)
	entry.Modes = append([]string{}, entry.Modes...)
	entry.Operations = append([]string{}, entry.Operations...)
	entry.OutputPaths = append([]string{}, entry.OutputPaths...)
	entry.VariableKeys = append([]string{}, entry.VariableKeys...)
	entry.ReferenceSkillIDs = append([]string{}, entry.ReferenceSkillIDs...)
	return entry
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
