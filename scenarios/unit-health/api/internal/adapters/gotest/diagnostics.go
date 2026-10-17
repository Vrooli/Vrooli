package gotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"unit-health/internal/adapters"
)

const (
	maxDiagnosticLineBytes     = 64 << 10
	maxDiagnosticFailures      = 32
	maxDiagnosticSummaryBytes  = 16 << 10
	maxDiagnosticIdentityBytes = 2048
	// Reserve space for the fixed heading and bounded counters/status below.
	maxDiagnosticWitnessBytes = maxDiagnosticSummaryBytes - 512
)

func (Analyzer) NewExecutionDiagnostics() adapters.ExecutionDiagnostics {
	return &failureDiagnostics{}
}

// failureDiagnostics keeps observed identities, not full output or execution
// proof. It is command-local: Write has one stdout owner and Summary is called
// after that owner finishes. Carry and witness budgets do not depend on total
// command output, package count, or concurrent test count.
type failureDiagnostics struct {
	carry                []byte
	discardLine          bool
	witnesses            []string
	witnessBytes         int
	validEvents          uint64
	malformedLines       uint64
	oversizedLines       uint64
	omittedFailureEvents uint64
}

func (d *failureDiagnostics) Write(p []byte) (int, error) {
	n := len(p)
	for len(p) > 0 {
		end := bytes.IndexByte(p, '\n')
		part := p
		if end >= 0 {
			part = p[:end]
		}
		if !d.discardLine {
			if len(part) > maxDiagnosticLineBytes-len(d.carry) {
				d.oversizedLines++
				d.carry = d.carry[:0]
				d.discardLine = true
			} else {
				// Allocate a fixed cap, avoiding append's growth beyond the limit.
				if d.carry == nil {
					d.carry = make([]byte, 0, maxDiagnosticLineBytes)
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
	// Optional diagnostics cannot turn runner output into an execution error.
	return n, nil
}

func (d *failureDiagnostics) observe(line []byte) {
	if len(bytes.TrimSpace(line)) == 0 {
		return
	}
	var event nativeEvent
	if !utf8.Valid(line) || json.Unmarshal(line, &event) != nil || event.Action == "" || event.Package == "" {
		d.malformedLines++
		return
	}
	d.validEvents++
	if event.Action != "fail" {
		return
	}
	if len(event.Package) > maxDiagnosticIdentityBytes || len(event.Test) > maxDiagnosticIdentityBytes || len(d.witnesses) >= maxDiagnosticFailures {
		d.omittedFailureEvents++
		return
	}
	witness := fmt.Sprintf("package=%q", event.Package)
	if event.Test != "" {
		witness += fmt.Sprintf(" test=%q", event.Test)
	} else {
		witness += " (package failure; no test identity supplied)"
	}
	if len(witness)+1 > maxDiagnosticWitnessBytes-d.witnessBytes {
		d.omittedFailureEvents++
		return
	}
	d.witnesses = append(d.witnesses, witness)
	d.witnessBytes += len(witness) + 1
}

func (d *failureDiagnostics) Summary() string {
	var out strings.Builder
	out.WriteString("--- observed Go failure events (bounded diagnostic; not complete execution proof) ---\n")
	for _, witness := range d.witnesses {
		out.WriteString(witness)
		out.WriteByte('\n')
	}
	if len(d.witnesses) == 0 {
		out.WriteString("No Go failure identity retained; this does not establish that tests passed.\n")
	}
	unterminated := d.discardLine || len(bytes.TrimSpace(d.carry)) != 0
	fmt.Fprintf(&out, "retained=%d omitted_failure_events=%d malformed_lines=%d oversized_lines=%d unterminated_line=%t valid_events=%d", len(d.witnesses), d.omittedFailureEvents, d.malformedLines, d.oversizedLines, unterminated, d.validEvents)
	if d.omittedFailureEvents > 0 || d.malformedLines > 0 || d.oversizedLines > 0 || unterminated {
		out.WriteString("; diagnostics partial")
	}
	return out.String()
}
