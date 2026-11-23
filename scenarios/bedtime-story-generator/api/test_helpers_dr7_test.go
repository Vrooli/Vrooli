package main

// Test-only helpers moved from production files by the private-function debt lens.
func getAgeDescription(ageGroup string) string {
	switch ageGroup {
	case "3-5":
		return "ages 3-5 (very simple vocabulary, short sentences, repetition)"
	case "6-8":
		return "ages 6-8 (simple vocabulary, clear plot, relatable characters)"
	case "9-12":
		return "ages 9-12 (richer vocabulary, complex plot, character development)"
	default:
		return "ages 6-8"
	}
}
func getLengthDescription(length string) string {
	switch length {
	case "short":
		return "3-5 minute read (about 300-500 words)"
	case "medium":
		return "8-10 minute read (about 800-1200 words)"
	case "long":
		return "12-15 minute read (about 1500-2000 words)"
	default:
		return "8-10 minute read"
	}
}
