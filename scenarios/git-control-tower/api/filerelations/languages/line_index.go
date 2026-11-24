package languages

import "sort"

// BuildLineIndex returns the byte offset of each line's first character.
func BuildLineIndex(content string) []int {
	lineStarts := []int{0}
	for i := range content {
		if content[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}
	return lineStarts
}

// FindLineNumber returns the 1-indexed line containing pos.
func FindLineNumber(lineStarts []int, pos int) int {
	line := sort.Search(len(lineStarts), func(i int) bool { return lineStarts[i] > pos })
	if line == 0 {
		return 1
	}
	return line
}
