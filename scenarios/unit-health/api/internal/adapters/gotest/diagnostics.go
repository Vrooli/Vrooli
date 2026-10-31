package gotest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"unit-health/internal/adapters"
)

const (
	maxDiagnosticLineBytes               = 64 << 10
	maxDiagnosticFailures                = 32
	maxDiagnosticSummaryBytes            = 16 << 10
	maxDiagnosticIdentityBytes           = 2048
	maxPendingDiagnosticTests            = 16
	maxPendingOutputKeyBytes             = 512
	maxPendingOutputPerTestBytes         = 2 << 10
	maxPendingOutputTotalBytes           = 8 << 10
	maxFailureOutputExcerptBytes         = 768
	maxDiagnosticFailureOutputTotalBytes = 8 << 10
	// Reserve space for the fixed heading and bounded counters/status below.
	maxDiagnosticWitnessBytes = maxDiagnosticSummaryBytes - 512
)

func (Analyzer) NewExecutionDiagnostics() adapters.ExecutionDiagnostics {
	return &failureDiagnostics{}
}

type outputKey struct {
	packageName string
	testName    string
}

type pendingOutput struct {
	text     string
	lastSeen uint64
}

// failureDiagnostics keeps bounded failure identities and nearby Go test
// output, not full command output or execution proof. It is command-local: Write
// has one stdout owner and Summary is called after that owner finishes.
type failureDiagnostics struct {
	carry                []byte
	discardLine          bool
	witnesses            []string
	witnessBytes         int
	validEvents          uint64
	malformedLines       uint64
	oversizedLines       uint64
	omittedFailureEvents uint64
	omittedOutputBytes   uint64
	failureOutputBytes   int
	pendingOutputs       map[outputKey]pendingOutput
	pendingOutputBytes   int
	outputSequence       uint64
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
	if !utf8.Valid(line) || json.Unmarshal(line, &event) != nil || event.Action == "" {
		d.malformedLines++
		return
	}
	if event.Action == "build-output" || event.Action == "build-fail" {
		if event.ImportPath == "" || len(event.ImportPath) > maxDiagnosticIdentityBytes || len(event.ImportPath) > maxPendingOutputKeyBytes {
			d.malformedLines++
			return
		}
		d.validEvents++
		if event.Action == "build-output" && event.Output != "" {
			d.rememberOutput(outputKey{packageName: event.ImportPath}, event.Output)
		}
		return
	}
	if event.Package == "" {
		d.malformedLines++
		return
	}
	d.validEvents++
	key, keyOK := diagnosticOutputKey(event)
	switch event.Action {
	case "run":
		if keyOK {
			d.clearPendingOutput(key)
		}
	case "output":
		if event.Output != "" {
			if keyOK {
				d.rememberOutput(key, event.Output)
			} else {
				d.omittedOutputBytes += uint64(len(event.Output))
			}
		}
	case "pass", "skip":
		if keyOK {
			d.clearPendingOutput(key)
		}
	case "fail":
		d.recordFailure(event, key, keyOK)
	}
}

func diagnosticOutputKey(event nativeEvent) (outputKey, bool) {
	packageName := event.Package
	// A package-level build failure may be reported by the importing package
	// while the compiler output belongs to a dependency package. Join output to
	// the explicit failed-build identity when the event provides one.
	if event.Action == "fail" && event.Test == "" && event.FailedBuild != "" && len(event.FailedBuild) <= maxDiagnosticIdentityBytes {
		packageName = event.FailedBuild
	}
	key := outputKey{packageName: packageName, testName: event.Test}
	if len(key.packageName) > maxDiagnosticIdentityBytes || len(key.testName) > maxDiagnosticIdentityBytes || len(key.packageName)+len(key.testName) > maxPendingOutputKeyBytes {
		return outputKey{}, false
	}
	return key, true
}

func (d *failureDiagnostics) rememberOutput(key outputKey, output string) {
	if d.pendingOutputs == nil {
		d.pendingOutputs = make(map[outputKey]pendingOutput)
	}
	d.outputSequence++
	current, exists := d.pendingOutputs[key]
	if !exists {
		for len(d.pendingOutputs) >= maxPendingDiagnosticTests {
			d.evictOldestOutput(outputKey{}, false)
		}
		current = pendingOutput{}
	}
	combined := current.text + output
	if len(combined) > maxPendingOutputPerTestBytes {
		trimmed := utf8Tail(combined, maxPendingOutputPerTestBytes)
		d.omittedOutputBytes += uint64(len(combined) - len(trimmed))
		combined = trimmed
	}
	d.pendingOutputBytes += len(combined) - len(current.text)
	current.text = combined
	current.lastSeen = d.outputSequence
	d.pendingOutputs[key] = current
	for d.pendingOutputBytes > maxPendingOutputTotalBytes {
		if !d.evictOldestOutput(key, true) {
			break
		}
	}
}

func (d *failureDiagnostics) evictOldestOutput(except outputKey, hasExcept bool) bool {
	var oldestKey outputKey
	var oldest pendingOutput
	found := false
	for key, value := range d.pendingOutputs {
		if hasExcept && key == except {
			continue
		}
		if !found || value.lastSeen < oldest.lastSeen {
			oldestKey, oldest, found = key, value, true
		}
	}
	if !found {
		return false
	}
	delete(d.pendingOutputs, oldestKey)
	d.pendingOutputBytes -= len(oldest.text)
	d.omittedOutputBytes += uint64(len(oldest.text))
	return true
}

func (d *failureDiagnostics) clearPendingOutput(key outputKey) string {
	pending, ok := d.pendingOutputs[key]
	if !ok {
		return ""
	}
	delete(d.pendingOutputs, key)
	d.pendingOutputBytes -= len(pending.text)
	return pending.text
}

func (d *failureDiagnostics) recordFailure(event nativeEvent, key outputKey, keyOK bool) {
	if len(event.Package) > maxDiagnosticIdentityBytes || len(event.Test) > maxDiagnosticIdentityBytes || len(d.witnesses) >= maxDiagnosticFailures {
		d.omittedFailureEvents++
		if keyOK {
			d.omittedOutputBytes += uint64(len(d.clearPendingOutput(key)))
		}
		return
	}
	witness := fmt.Sprintf("package=%q", event.Package)
	if event.FailedBuild != "" && len(event.FailedBuild) <= maxDiagnosticIdentityBytes {
		witness += fmt.Sprintf(" failed_build=%q", event.FailedBuild)
	}
	if event.Test != "" {
		witness += fmt.Sprintf(" test=%q", event.Test)
	} else {
		witness += " (package failure; no test identity supplied)"
	}
	if len(witness)+1 > maxDiagnosticWitnessBytes-d.witnessBytes {
		d.omittedFailureEvents++
		if keyOK {
			d.omittedOutputBytes += uint64(len(d.clearPendingOutput(key)))
		}
		return
	}

	if keyOK {
		output := d.clearPendingOutput(key)
		if output != "" {
			available := maxDiagnosticFailureOutputTotalBytes - d.failureOutputBytes
			if witnessRoom := maxDiagnosticWitnessBytes - d.witnessBytes - len(witness) - len(" output=") - 1; available > witnessRoom {
				available = witnessRoom
			}
			if available > maxFailureOutputExcerptBytes {
				available = maxFailureOutputExcerptBytes
			}
			quoted, retainedBytes := quoteOutputTail(output, available)
			if quoted != "" {
				witness += " output=" + quoted
				d.failureOutputBytes += len(" output=") + len(quoted)
				d.omittedOutputBytes += uint64(len(output) - retainedBytes)
			} else {
				d.omittedOutputBytes += uint64(len(output))
			}
		}
	}
	d.witnesses = append(d.witnesses, witness)
	d.witnessBytes += len(witness) + 1
}

func quoteOutputTail(output string, maxQuotedBytes int) (string, int) {
	if maxQuotedBytes < 2 {
		return "", 0
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return "", 0
	}
	runes := []rune(output)
	truncated := false
	for len(runes) > 0 {
		candidate := string(runes)
		if truncated {
			candidate = "…" + candidate
		}
		quoted := strconv.Quote(candidate)
		if len(quoted) <= maxQuotedBytes {
			return quoted, len(string(runes))
		}
		// Keep the tail: test diagnostics commonly put the assertion at the end.
		runes = runes[1:]
		truncated = true
	}
	return "", 0
}

func utf8Tail(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	value = value[len(value)-maxBytes:]
	for value != "" && !utf8.ValidString(value) {
		value = value[1:]
	}
	return value
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
	fmt.Fprintf(&out, "retained=%d omitted_failure_events=%d omitted_output_bytes=%d malformed_lines=%d oversized_lines=%d unterminated_line=%t valid_events=%d", len(d.witnesses), d.omittedFailureEvents, d.omittedOutputBytes, d.malformedLines, d.oversizedLines, unterminated, d.validEvents)
	if d.omittedFailureEvents > 0 || d.omittedOutputBytes > 0 || d.malformedLines > 0 || d.oversizedLines > 0 || unterminated {
		out.WriteString("; diagnostics partial")
	}
	return out.String()
}
