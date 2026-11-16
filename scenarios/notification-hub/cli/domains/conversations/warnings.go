package conversations

import (
	"fmt"
	"io"
	"strings"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// Packet check severities from the Hub. A blocking finding refuses an agent's
// ask: the dry run lists it first so the agent fixes it before it sends.
const (
	severityDropped  = "dropped"
	severityBlocking = "blocking"
)

// hasBlocking reports a finding that would refuse the ask.
func hasBlocking(warnings []*v1.PacketWarning) bool {
	for _, warning := range warnings {
		if warning.GetSeverity() == severityBlocking && !warning.GetCarriedOver() {
			return true
		}
	}
	return false
}

// writePacketWarnings prints the Hub's packet check as one readable block:
// what would refuse the ask first, then what it left out, then advice; sent says whether the ask went out (a dry
// run did not). It writes to stderr so --json output stays clean, and prints
// nothing when the packet was clean.
//
// A revision repeats every finding that was already true of the open ask. Those
// the Hub marks carried over are folded into one line, so what is news is what
// the reader sees; --json lists every finding.
func writePacketWarnings(w io.Writer, warnings []*v1.PacketWarning, sent bool) {
	var blocking, dropped, advice []string
	carried := 0
	for _, warning := range warnings {
		if warning.GetCarriedOver() {
			carried++
			continue
		}
		line := packetWarningLine(warning)
		switch warning.GetSeverity() {
		case severityBlocking:
			blocking = append(blocking, line)
		case severityDropped:
			dropped = append(dropped, line)
		default:
			advice = append(advice, line)
		}
	}
	if len(blocking)+len(dropped)+len(advice)+carried == 0 {
		return
	}
	var b strings.Builder
	var counts []string
	if len(blocking) > 0 {
		counts = append(counts, fmt.Sprintf("%d would refuse the ask", len(blocking)))
	}
	if len(dropped) > 0 {
		counts = append(counts, fmt.Sprintf("%d left out", len(dropped)))
	}
	if len(advice) > 0 {
		counts = append(counts, fmt.Sprintf("%d advice", len(advice)))
	}
	if len(counts) == 0 {
		counts = append(counts, "no new findings")
	}
	outcome := "The ask was sent; nothing here blocked it."
	switch {
	case !sent && len(blocking) > 0:
		outcome = "Nothing was sent (dry run). Sending now would be refused: fix every finding under \"Would refuse the ask\", then run --dry-run again."
	case !sent:
		outcome = "Nothing was sent (dry run); the ask would go out as it is, so fix these first if they matter."
	}
	fmt.Fprintf(&b, "Packet check: %s. %s\n", strings.Join(counts, ", "), outcome)
	writeWarningSection(&b, "Would refuse the ask (fix before sending):", blocking)
	writeWarningSection(&b, "Left out of the packet:", dropped)
	writeWarningSection(&b, "Advice:", advice)
	if carried > 0 {
		verb := "still apply"
		if carried == 1 {
			verb = "still applies"
		}
		fmt.Fprintf(&b, "  %s %s (same as before; --json lists them).\n", plural(carried, "earlier warning"), verb)
	}
	_, _ = io.WriteString(w, b.String())
}

func writeWarningSection(b *strings.Builder, heading string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "  %s\n", heading)
	for _, line := range lines {
		fmt.Fprintf(b, "    - %s\n", line)
	}
}

func packetWarningLine(w *v1.PacketWarning) string {
	var b strings.Builder
	if field := strings.TrimSpace(w.GetField()); field != "" {
		b.WriteString(field + ": ")
	}
	b.WriteString(strings.TrimSpace(w.GetMessage()))
	if code := strings.TrimSpace(w.GetCode()); code != "" {
		fmt.Fprintf(&b, " (%s)", code)
	}
	return b.String()
}
