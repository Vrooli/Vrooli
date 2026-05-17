package audio

import "fmt"

// contentTypeFor maps the canonical audio format short-code to its
// matching HTTP content-type. The empty string and "wav" both map to
// "audio/wav"; unknown formats fall back to "application/octet-stream".
func contentTypeFor(format string) string {
	switch format {
	case "wav", "":
		return "audio/wav"
	case "mp3":
		return "audio/mpeg"
	case "flac":
		return "audio/flac"
	case "aac":
		return "audio/aac"
	case "ogg":
		return "audio/ogg"
	}
	return "application/octet-stream"
}

// atoiOr parses s as a decimal integer; on empty input or parse error
// it returns def.
func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}
