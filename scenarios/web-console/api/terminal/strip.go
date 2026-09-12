// strip.go: in-memory ANSI escape removal.
//
// StripEscapes is a stateless helper for callers that have a byte stream
// they want to render as plain text without spinning up an emulator and
// scrollback ring (e.g. text-normalization for conversation log storage,
// dedup key computation). For grid-level reads — visible screen cells,
// cursor position, scrollback — use Emulator.View() / PlainText() instead.
//
// The emulator-based path remains the source of truth for "what is on
// screen": StripEscapes does not handle cursor positioning, alt-buffer
// transitions, or scroll behavior. It only erases escape bytes from a
// linear stream so downstream substring/regex matching works.

package terminal

// StripEscapes returns data with CSI (\x1b[...X), OSC (\x1b]...ST/BEL),
// and simple two-byte ESC sequences removed. Non-escape bytes — including
// UTF-8 multi-byte sequences — are preserved verbatim.
//
// Returns the input slice unchanged when it contains no ESC byte, to
// avoid an allocation on the common path.
func StripEscapes(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	// Fast path: no ESC byte present means nothing to strip.
	hasEsc := false
	for _, b := range data {
		if b == 0x1b {
			hasEsc = true
			break
		}
	}
	if !hasEsc {
		return data
	}

	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if data[i] != 0x1b {
			out = append(out, data[i])
			i++
			continue
		}
		if i+1 >= len(data) {
			// Trailing ESC with no follower — drop it.
			i++
			continue
		}
		switch data[i+1] {
		case '[':
			// CSI: ESC '[' params (0x30-0x3F) intermediates (0x20-0x2F) final (0x40-0x7E).
			j := i + 2
			for j < len(data) && data[j] >= 0x30 && data[j] <= 0x3F {
				j++
			}
			for j < len(data) && data[j] >= 0x20 && data[j] <= 0x2F {
				j++
			}
			if j < len(data) && data[j] >= 0x40 && data[j] <= 0x7E {
				j++
			}
			i = j
		case ']':
			// OSC: terminated by BEL (0x07) or ST (ESC '\\').
			j := i + 2
			for j < len(data) {
				if data[j] == 0x07 {
					j++
					break
				}
				if data[j] == 0x1b && j+1 < len(data) && data[j+1] == '\\' {
					j += 2
					break
				}
				j++
			}
			i = j
		default:
			// Two-byte ESC.
			i += 2
		}
	}
	return out
}
