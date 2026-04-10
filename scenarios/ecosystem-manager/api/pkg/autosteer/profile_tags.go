package autosteer

func hasAnyTag(profileTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(filterTags))
	for _, tag := range filterTags {
		tagSet[tag] = struct{}{}
	}

	for _, tag := range profileTags {
		if _, ok := tagSet[tag]; ok {
			return true
		}
	}

	return false
}
