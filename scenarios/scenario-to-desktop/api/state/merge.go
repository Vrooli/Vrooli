package state

func mergeBy[T any, K comparable](existing, updates []T, key func(T) K) []T {
	if len(updates) == 0 {
		return existing
	}

	merged := make(map[K]T, len(existing)+len(updates))
	for _, item := range existing {
		merged[key(item)] = item
	}
	for _, item := range updates {
		merged[key(item)] = item
	}

	result := make([]T, 0, len(merged))
	for _, item := range merged {
		result = append(result, item)
	}
	return result
}
