package server

import "regexp"

func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[mGKHf]`)
	return ansiRegex.ReplaceAllString(str, "")
}
