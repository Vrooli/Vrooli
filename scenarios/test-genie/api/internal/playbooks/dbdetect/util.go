package dbdetect

import "sort"

func sortStrings(s []string) {
	sort.Strings(s)
}

func sortStringsCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
