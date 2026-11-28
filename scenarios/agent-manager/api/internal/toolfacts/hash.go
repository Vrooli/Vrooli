package toolfacts

import (
	"strconv"
	"unicode/utf8"
)

const (
	fnvOffset uint64 = 14695981039346656037
	fnvPrime  uint64 = 1099511628211
)

// hashString is the FNV-1a hash of text with runs of whitespace collapsed to
// one space and the ends trimmed, so formatting never makes two texts differ.
// Identities are compared, never stored, so 64 bits are plenty.
func hashString(text string) uint64 { return hashFields("", text) }

// hashFields hashes a short prefix (an outcome marker) and a text, separated
// so no prefix and text pair collides with another.
func hashFields(prefix, text string) uint64 {
	hash := fnvOffset
	for i := 0; i < len(prefix); i++ {
		hash = (hash ^ uint64(prefix[i])) * fnvPrime
	}
	hash = (hash ^ 0x1f) * fnvPrime
	space := false
	started := false
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f' {
			space = started
			continue
		}
		if space {
			hash = (hash ^ ' ') * fnvPrime
			space = false
		}
		started = true
		hash = (hash ^ uint64(c)) * fnvPrime
	}
	return hash
}

// sampleParts joins texts into one string of at most head+tail bytes: the whole
// text when it fits, else its first head bytes, a marker carrying the full
// length, and its last tail bytes. A change anywhere in a huge output changes
// the length or one of the ends far more often than not, at a fixed cost.
func sampleParts(parts []string, head, tail int) string {
	total := 0
	for _, part := range parts {
		total += len(part) + 1
	}
	if total <= head+tail {
		joined := make([]byte, 0, total)
		for _, part := range parts {
			joined = append(joined, part...)
			joined = append(joined, '\n')
		}
		return string(joined)
	}
	first := make([]byte, 0, head)
	for _, part := range parts {
		if len(first) >= head {
			break
		}
		room := head - len(first)
		if len(part) < room {
			room = len(part)
		}
		first = append(first, part[:room]...)
		first = append(first, '\n')
	}
	last := make([]byte, 0, tail)
	for i := len(parts) - 1; i >= 0 && len(last) < tail; i-- {
		part := parts[i]
		room := tail - len(last)
		if len(part) < room {
			room = len(part)
		}
		last = append(append([]byte(part[len(part)-room:]), '\n'), last...)
	}
	return string(first) + "\n[" + strconv.Itoa(total) + " bytes]\n" + string(last)
}

// boundText keeps at most limit bytes of text, cut at a rune boundary.
func boundText(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}
