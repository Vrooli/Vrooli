package main

// stripANSI removes ANSI escape sequences from data.
// Handles CSI sequences (\x1b[...X), OSC sequences (\x1b]...ST),
// and simple two-byte escapes (\x1bX).
// Preserves all non-escape content including UTF-8.
func stripANSI(data []byte) []byte {
	if len(data) == 0 {
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

		// ESC byte found — determine sequence type.
		if i+1 >= len(data) {
			// Trailing ESC with no follower — skip it.
			i++
			continue
		}

		switch data[i+1] {
		case '[':
			// CSI sequence: \x1b[ followed by parameter bytes (0x30-0x3F),
			// intermediate bytes (0x20-0x2F), and a final byte (0x40-0x7E).
			j := i + 2
			// Skip parameter bytes (0x30-0x3F).
			for j < len(data) && data[j] >= 0x30 && data[j] <= 0x3F {
				j++
			}
			// Skip intermediate bytes (0x20-0x2F).
			for j < len(data) && data[j] >= 0x20 && data[j] <= 0x2F {
				j++
			}
			// Skip final byte (0x40-0x7E).
			if j < len(data) && data[j] >= 0x40 && data[j] <= 0x7E {
				j++
			}
			i = j

		case ']':
			// OSC sequence: \x1b] terminated by ST (\x1b\\) or BEL (\x07).
			j := i + 2
			for j < len(data) {
				if data[j] == 0x07 {
					// BEL terminator.
					j++
					break
				}
				if data[j] == 0x1b && j+1 < len(data) && data[j+1] == '\\' {
					// ST terminator (\x1b\\).
					j += 2
					break
				}
				j++
			}
			i = j

		default:
			// Simple two-byte escape: \x1b + single byte.
			i += 2
		}
	}

	return out
}
