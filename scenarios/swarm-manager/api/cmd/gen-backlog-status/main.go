// Command gen-backlog-status renders the UI's mirror of the backlog status
// vocabulary from the Go SSOT in internal/backlogstatus.
//
// The status table lives in Go because the server is what validates and writes
// statuses; the UI needs the same vocabulary for its union type and its
// lifecycle-ordered lists. Generating the mirror keeps them from drifting the
// way the hand-maintained proto allowlists did.
//
// Deliberately NOT generated: the status→color maps. Those are design
// decisions that belong to the UI, and leaving them hand-written keeps
// TypeScript's exhaustiveness checking on `Record<BacklogStatus, string>`
// doing real work — it is what forces a new status to be given a color.
//
// Usage:
//
//	go run ./cmd/gen-backlog-status -out ../ui/src/types/backlog-status.generated.ts
//	go run ./cmd/gen-backlog-status -check   # verify the committed file is current
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"swarm-manager/internal/backlogstatus"
)

const defaultOut = "../ui/src/types/backlog-status.generated.ts"

func main() {
	out := flag.String("out", defaultOut, "path to the generated TypeScript file")
	check := flag.String("check", "", "verify this file matches generated output instead of writing")
	flag.Parse()

	rendered := render()

	if *check != "" {
		existing, err := os.ReadFile(*check)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gen-backlog-status: read %s: %v\n", *check, err)
			os.Exit(1)
		}
		if !bytes.Equal(existing, rendered) {
			fmt.Fprintf(os.Stderr,
				"gen-backlog-status: %s is stale.\nRun `make gen-status` after editing the status table.\n", *check)
			os.Exit(1)
		}
		return
	}

	if err := os.WriteFile(*out, rendered, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "gen-backlog-status: write %s: %v\n", *out, err)
		os.Exit(1)
	}
}

func render() []byte {
	defs := backlogstatus.Definitions()

	var b strings.Builder
	b.WriteString(`/**
 * GENERATED FILE — DO NOT EDIT.
 *
 * Mirror of the backlog status vocabulary. The source of truth is
 * api/internal/backlogstatus/statuses.go; regenerate with ` + "`make gen-status`" + `
 * from the scenario root after changing the status table.
 *
 * Status colors are intentionally NOT generated — they live in
 * types/constants.ts so TypeScript's exhaustiveness check on
 * Record<BacklogStatus, string> forces a new status to be given one.
 */

`)

	// The union, with each member's doc comment carried across so the meaning
	// travels with the type rather than living only on the server.
	b.WriteString("/** Valid lifecycle states for a backlog item, in lifecycle order. */\nexport type BacklogStatus =\n")
	for i, d := range defs {
		b.WriteString(fmt.Sprintf("  /** %s */\n", wrapDoc(d.Doc, "   * ")))
		terminator := ""
		if i == len(defs)-1 {
			terminator = ";"
		}
		b.WriteString(fmt.Sprintf("  | %q%s\n", d.Value, terminator))
	}

	b.WriteString("\n/** Every status, in lifecycle order. */\nexport const BACKLOG_STATUSES: readonly BacklogStatus[] = [\n")
	for _, d := range defs {
		b.WriteString(fmt.Sprintf("  %q,\n", d.Value))
	}
	b.WriteString("] as const;\n")

	b.WriteString("\n/** Human-readable label for each status. */\nexport const BACKLOG_STATUS_LABELS: Record<BacklogStatus, string> = {\n")
	for _, d := range defs {
		b.WriteString(fmt.Sprintf("  %s: %q,\n", tsKey(d.Value), d.Label))
	}
	b.WriteString("};\n")

	writeSubset(&b, defs, "USER_SETTABLE_STATUSES",
		"Statuses an operator may set directly via the generic status patch.\n * Execution-owned and review-gated statuses are excluded: the former belong to\n * the execution system, the latter must exit through review-decide so the\n * decision carries an audit trail.",
		func(d backlogstatus.Definition) bool { return d.UserSettable })

	writeSubset(&b, defs, "TERMINAL_STATUSES",
		"Settled statuses — the item is not coming back without an explicit revival.",
		func(d backlogstatus.Definition) bool { return d.Phase == backlogstatus.PhaseTerminal })

	writeSubset(&b, defs, "RESOLVED_STATUSES",
		"Statuses meaning nothing depending on the item is still waiting.\n * Note this is NOT the same as completed: dropped work resolves a dependency\n * without having achieved anything, and failed work does not resolve at all.",
		func(d backlogstatus.Definition) bool { return d.Resolved })

	writeSubset(&b, defs, "QUEUEABLE_BACKLOG_STATUSES",
		"Statuses from which an item can be queued for execution.",
		func(d backlogstatus.Definition) bool { return d.Phase == backlogstatus.PhasePlanning })

	writeSubset(&b, defs, "IN_FLIGHT_STATUSES",
		"Statuses owned by the execution system while a run is live.",
		func(d backlogstatus.Definition) bool { return d.Phase == backlogstatus.PhaseInFlight })

	writeSubset(&b, defs, "REVIEW_STATUSES",
		"Statuses where a review round is gathering evidence or awaiting a verdict.",
		func(d backlogstatus.Definition) bool { return d.Phase == backlogstatus.PhaseReview })

	return []byte(b.String())
}

func writeSubset(b *strings.Builder, defs []backlogstatus.Definition, name, doc string, keep func(backlogstatus.Definition) bool) {
	b.WriteString(fmt.Sprintf("\n/**\n * %s\n */\nexport const %s: readonly BacklogStatus[] = [\n", doc, name))
	for _, d := range defs {
		if keep(d) {
			b.WriteString(fmt.Sprintf("  %q,\n", d.Value))
		}
	}
	b.WriteString("] as const;\n")
}

// tsKey renders an object key, quoting only when the value is not a valid bare
// identifier. Every current status is snake_case and safe bare, but a future
// value with a hyphen would otherwise emit invalid TypeScript.
func tsKey(value string) string {
	for i, r := range value {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLower || r == '_' || (isDigit && i > 0) {
			continue
		}
		return fmt.Sprintf("%q", value)
	}
	return value
}

// wrapDoc reflows a doc string into a JSDoc-safe single block, keeping it
// readable in the generated file without importing a wrapping dependency.
func wrapDoc(doc, indent string) string {
	const width = 76
	words := strings.Fields(doc)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	current := words[0]
	for _, w := range words[1:] {
		if len(current)+1+len(w) > width {
			lines = append(lines, current)
			current = w
			continue
		}
		current += " " + w
	}
	lines = append(lines, current)
	return strings.Join(lines, "\n"+indent)
}
