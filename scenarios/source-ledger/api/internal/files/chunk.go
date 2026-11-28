package files

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// Chunk is one citable span of a file.
type Chunk struct {
	LineStart, LineEnd int
	Heading            string
	Text               string
}

const (
	windowLines    = 40
	maxSectionLine = 60
	recordsPer     = 10
	rowsPer        = 20
	maxChunkChars  = 4000
)

// readCapped reads at most limit bytes: the head, or the tail for kinds whose
// newest content is last. It returns the 1-based line number of the first line
// returned so citations stay exact.
func readCapped(abs string, kind Kind, size, limit int64) ([]byte, int, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	if limit <= 0 || size <= limit {
		b, err := io.ReadAll(f)
		return b, 1, err
	}
	if !kind.tail() {
		b, err := io.ReadAll(io.LimitReader(f, limit))
		if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
			b = b[:i+1]
		}
		return b, 1, err
	}
	// Count the lines being skipped so the tail's citations are real line numbers.
	skipped, err := countLines(f, size-limit)
	if err != nil {
		return nil, 0, err
	}
	skipped++
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, 0, err
	}
	// Drop the partial first line.
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[i+1:]
		skipped++
	}
	return b, skipped, nil
}

// countLines reads n bytes from r and counts their newlines.
func countLines(r io.Reader, n int64) (int, error) {
	lines := 0
	buf := make([]byte, 64*1024)
	for n > 0 {
		read, err := io.ReadFull(r, buf[:min(n, int64(len(buf)))])
		lines += bytes.Count(buf[:read], []byte{'\n'})
		n -= int64(read)
		if err != nil {
			return 0, err
		}
	}
	return lines, nil
}

// chunkFile splits content into citable chunks by kind.
func chunkFile(kind Kind, content []byte, firstLine int) []Chunk {
	// Files in other encodings keep their text; invalid bytes become U+FFFD so
	// every chunk can travel through protobuf and JSON.
	text := strings.ToValidUTF8(strings.ReplaceAll(string(content), "\r\n", "\n"), "\uFFFD")
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
		return nil
	}
	switch kind {
	case KindMarkdown:
		return chunkMarkdown(lines, firstLine)
	case KindCSV, KindTSV:
		return chunkTable(lines, firstLine, firstLine == 1)
	case KindJSONL:
		return chunkWindows(lines, firstLine, recordsPer, "records")
	default:
		return chunkWindows(lines, firstLine, windowLines, "")
	}
}

// chunkMarkdown makes one chunk per section, keeping the heading trail
// ("Plan › Risks") so a hit says where it sits; long sections are windowed.
func chunkMarkdown(lines []string, firstLine int) []Chunk {
	type heading struct {
		level int
		title string
	}
	var chunks []Chunk
	var trail []heading
	start := 0
	fence := false
	flush := func(end int) {
		titles := make([]string, len(trail))
		for i, h := range trail {
			titles[i] = h.title
		}
		chunks = append(chunks, sectionChunks(lines[start:end], firstLine+start, strings.Join(titles, " › "))...)
	}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = !fence
		}
		if fence {
			continue
		}
		level := headingLevel(trimmed)
		if level == 0 {
			continue
		}
		flush(i)
		start = i
		for len(trail) > 0 && trail[len(trail)-1].level >= level {
			trail = trail[:len(trail)-1]
		}
		trail = append(trail, heading{level, strings.TrimSpace(trimmed[level:])})
	}
	flush(len(lines))
	return chunks
}

// sectionChunks windows one section into chunks of at most maxSectionLine
// lines, dropping blank windows.
func sectionChunks(lines []string, firstLine int, label string) []Chunk {
	var chunks []Chunk
	for s := 0; s < len(lines); s += maxSectionLine {
		e := min(s+maxSectionLine, len(lines))
		if body := strings.TrimSpace(strings.Join(lines[s:e], "\n")); body != "" {
			chunks = append(chunks, Chunk{LineStart: firstLine + s, LineEnd: firstLine + e - 1, Heading: label, Text: clip(body)})
		}
	}
	return chunks
}

// headingLevel returns an ATX heading's level (1-6), or 0 for any other line.
func headingLevel(trimmed string) int {
	level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
	if level == 0 || level > 6 || len(trimmed) == level || trimmed[level] != ' ' {
		return 0
	}
	return level
}

// chunkTable keeps the header with every group of rows so a row hit is
// self-describing.
func chunkTable(lines []string, firstLine int, hasHeader bool) []Chunk {
	header := ""
	body := lines
	offset := 0
	if hasHeader && len(lines) > 0 {
		header, body, offset = lines[0], lines[1:], 1
	}
	var chunks []Chunk
	for s := 0; s < len(body); s += rowsPer {
		e := s + rowsPer
		if e > len(body) {
			e = len(body)
		}
		text := strings.Join(body[s:e], "\n")
		if header != "" {
			text = header + "\n" + text
		}
		start := firstLine + offset + s
		chunks = append(chunks, Chunk{LineStart: start, LineEnd: start + (e - s) - 1, Heading: fmt.Sprintf("rows %d-%d", start, start+(e-s)-1), Text: clip(text)})
	}
	return chunks
}

func chunkWindows(lines []string, firstLine, size int, label string) []Chunk {
	var chunks []Chunk
	for s := 0; s < len(lines); s += size {
		e := s + size
		if e > len(lines) {
			e = len(lines)
		}
		body := strings.TrimSpace(strings.Join(lines[s:e], "\n"))
		if body == "" {
			continue
		}
		heading := ""
		if label != "" {
			heading = fmt.Sprintf("%s %d-%d", label, firstLine+s, firstLine+e-1)
		}
		chunks = append(chunks, Chunk{LineStart: firstLine + s, LineEnd: firstLine + e - 1, Heading: heading, Text: clip(body)})
	}
	return chunks
}

func clip(text string) string {
	if len(text) <= maxChunkChars {
		return text
	}
	cut := maxChunkChars
	for cut > 0 && text[cut]&0xC0 == 0x80 {
		cut--
	}
	return text[:cut]
}

// firstHeading is the file's title for feed listings.
func firstHeading(chunks []Chunk, rel string) string {
	for _, c := range chunks {
		if c.Heading != "" {
			if i := strings.Index(c.Heading, " › "); i > 0 {
				return c.Heading[:i]
			}
			return c.Heading
		}
	}
	return rel[strings.LastIndex(rel, "/")+1:]
}
