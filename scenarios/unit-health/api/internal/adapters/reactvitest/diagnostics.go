package reactvitest

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"unit-health/internal/adapters"
)

const (
	maxVitestDiagnosticLineBytes = 16 << 10
	maxVitestFailureHeaders      = 32
	maxVitestDiagnosticBytes     = 12 << 10
)

type vitestDiagnosticStream uint8

const (
	vitestStdout vitestDiagnosticStream = iota
	vitestStderr
)

var ansiControl = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)

// vitestStreamDiagnostics retains only suite failure headers or the final
// Vitest summary. DOM snapshots and stack bodies are deliberately discarded.
type vitestStreamDiagnostics struct {
	stream         vitestDiagnosticStream
	carry          []byte
	discardLine    bool
	lines          []string
	retainedBytes  int
	omittedEntries uint64
	omittedBytes   uint64
	oversizedLines uint64
	finalized      bool
}

func newVitestStreamDiagnostics(stream vitestDiagnosticStream) *vitestStreamDiagnostics {
	return &vitestStreamDiagnostics{stream: stream}
}

func (d *vitestStreamDiagnostics) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		end := bytes.IndexByte(p, '\n')
		part := p
		if end >= 0 {
			part = p[:end]
		}
		if !d.discardLine {
			if len(part) > maxVitestDiagnosticLineBytes-len(d.carry) {
				d.oversizedLines++
				d.carry = d.carry[:0]
				d.discardLine = true
			} else {
				if d.carry == nil {
					d.carry = make([]byte, 0, maxVitestDiagnosticLineBytes)
				}
				d.carry = append(d.carry, part...)
			}
		}
		if end < 0 {
			break
		}
		if !d.discardLine {
			d.observe(d.carry)
		}
		d.carry = d.carry[:0]
		d.discardLine = false
		p = p[end+1:]
	}
	return n, nil
}

func (d *vitestStreamDiagnostics) observe(line []byte) {
	if !utf8.Valid(line) {
		d.omittedBytes += uint64(len(line))
		return
	}
	clean := strings.TrimSpace(ansiControl.ReplaceAllString(string(line), ""))
	if clean == "" {
		return
	}
	keep := false
	if d.stream == vitestStdout {
		keep = strings.HasPrefix(clean, "Test Files ") || strings.HasPrefix(clean, "Tests ")
	} else {
		keep = strings.HasPrefix(clean, "FAIL ")
	}
	if !keep {
		return
	}
	if d.stream == vitestStderr && len(d.lines) >= maxVitestFailureHeaders {
		d.omittedEntries++
		d.omittedBytes += uint64(len(clean))
		return
	}
	if len(clean) > maxVitestDiagnosticBytes-d.retainedBytes {
		d.omittedEntries++
		d.omittedBytes += uint64(len(clean))
		return
	}
	d.lines = append(d.lines, clean)
	d.retainedBytes += len(clean)
}

func (d *vitestStreamDiagnostics) finalize() {
	if d.finalized {
		return
	}
	d.finalized = true
	if len(d.carry) > 0 && !d.discardLine {
		d.observe(d.carry)
	}
	d.carry = nil
}

func (d *vitestStreamDiagnostics) Summary() string {
	if d == nil {
		return ""
	}
	d.finalize()
	if len(d.lines) == 0 && d.omittedEntries == 0 && d.omittedBytes == 0 && d.oversizedLines == 0 {
		return ""
	}
	var out strings.Builder
	if d.stream == vitestStdout {
		out.WriteString("--- Vitest summary lines (bounded diagnostic; not execution proof) ---\n")
	} else {
		out.WriteString("--- Vitest failure headers (bounded diagnostic; not execution proof) ---\n")
	}
	for _, line := range d.lines {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	fmt.Fprintf(&out, "retained=%d omitted_entries=%d omitted_bytes=%d oversized_lines=%d", len(d.lines), d.omittedEntries, d.omittedBytes, d.oversizedLines)
	if d.omittedEntries > 0 || d.omittedBytes > 0 || d.oversizedLines > 0 {
		out.WriteString("; diagnostics partial")
	}
	return out.String()
}

var _ adapters.ExecutionDiagnostics = (*vitestStreamDiagnostics)(nil)
