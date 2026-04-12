package runtime

func detectFirstAvailable(candidates []string) string {
	for _, candidate := range candidates {
		if commandAvailable(candidate) {
			return candidate
		}
	}
	return ""
}
