package suppressions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Format renders a marker as a source comment line (no newline). The
// comment style is chosen from the file extension.
func Format(file string, m Marker) string {
	body := fmt.Sprintf("%s %s reason=%q", MarkerDirective, m.ID, m.Reason)
	if strings.TrimSpace(m.Expires) != "" {
		body += fmt.Sprintf(" expires=%q", m.Expires)
	}
	switch strings.ToLower(filepath.Ext(file)) {
	case ".py", ".sh":
		return "# " + body
	default:
		return "// " + body
	}
}

// Writer inserts a suppression marker into a source file. It is the safe,
// non-destructive write path the apply domain uses to sanction a finding as
// intentional — distinct from the deferred destructive file-moving
// execution.
//
// seam: Writer lets the apply service write markers without the test
// touching the real filesystem.
type Writer interface {
	// WriteMarker inserts a comment for m into absPath. When line is a valid
	// 1-based line it is inserted immediately above that line; otherwise the
	// marker is appended at end of file. It is a no-op (returns nil) if an
	// equivalent active marker already covers the file (idempotent).
	WriteMarker(absPath string, line int, m Marker) error
}

// FileWriter is the production filesystem writer.
type FileWriter struct{}

// NewFileWriter returns the production writer.
func NewFileWriter() *FileWriter { return &FileWriter{} }

var _ Writer = (*FileWriter)(nil)

// WriteMarker implements Writer with an atomic temp-file + rename.
func (w *FileWriter) WriteMarker(absPath string, line int, m Marker) error {
	if !m.Validate() {
		return fmt.Errorf("refusing to write malformed marker (id=%q reason=%q)", m.ID, m.Reason)
	}
	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", absPath, err)
	}
	rel := filepath.Base(absPath)
	comment := Format(rel, m)

	// Idempotency: if the file already carries an active marker with the
	// same id, do nothing.
	for _, existing := range parseAll(string(data)) {
		if existing.ID == m.ID && existing.Validate() {
			return nil
		}
	}

	lines := strings.Split(string(data), "\n")
	insertAt := len(lines)
	if line >= 1 && line <= len(lines) {
		insertAt = line - 1
	}
	indent := leadingIndent(lines, insertAt)
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:insertAt]...)
	out = append(out, indent+comment)
	out = append(out, lines[insertAt:]...)

	tmp := absPath + ".arch-tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return fmt.Errorf("write temp for %s: %w", absPath, err)
	}
	if err := os.Rename(tmp, absPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp for %s: %w", absPath, err)
	}
	return nil
}

func parseAll(content string) []Marker {
	var out []Marker
	for _, l := range strings.Split(content, "\n") {
		if m, ok := ParseMarker(l); ok {
			out = append(out, m)
		}
	}
	return out
}

func leadingIndent(lines []string, idx int) string {
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	l := lines[idx]
	n := 0
	for n < len(l) && (l[n] == ' ' || l[n] == '\t') {
		n++
	}
	return l[:n]
}
