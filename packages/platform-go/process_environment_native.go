package platform

import (
	"errors"
	"strings"
	"unicode/utf16"
)

// The PEB exposes a bounded, double-NUL-terminated UTF-16 block. Reject every
// ambiguity before map conversion, including case aliases and hidden drive
// variables; hidden variables are retained rather than silently discarded.
func parseWindowsProcessEnvironment(buf []uint16) (map[string]string, error) {
	malformed := errors.New("platform: malformed or ambiguous Windows process environment")
	if len(buf) < 2 || len(buf) > 1<<19 || buf[len(buf)-1] != 0 || buf[len(buf)-2] != 0 {
		return nil, malformed
	}
	values := make(map[string]string)
	keys := make([]string, 0)
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] != 0 {
			continue
		}
		if i == start {
			for _, tail := range buf[i:] {
				if tail != 0 {
					return nil, malformed
				}
			}
			return values, nil
		}
		words := buf[start:i]
		for j := 0; j < len(words); j++ {
			ch := words[j]
			if ch >= 0xd800 && ch <= 0xdbff {
				if j+1 >= len(words) || words[j+1] < 0xdc00 || words[j+1] > 0xdfff {
					return nil, malformed
				}
				j++
			} else if ch >= 0xdc00 && ch <= 0xdfff {
				return nil, malformed
			}
		}
		entry := string(utf16.Decode(words))
		start = i + 1
		key, value, ok := strings.Cut(entry, "=")
		if strings.HasPrefix(entry, "=") {
			key, value, ok = strings.Cut(entry[1:], "=")
			key = "=" + key
		}
		if !ok || key == "" || key == "=" || len(keys) >= 4096 {
			return nil, malformed
		}
		for _, prior := range keys {
			if strings.EqualFold(prior, key) {
				return nil, malformed
			}
		}
		keys = append(keys, key)
		values[key] = value
	}
	return nil, malformed
}
