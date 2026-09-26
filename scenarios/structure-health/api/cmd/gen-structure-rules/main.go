// Command gen-structure-rules regenerates docs/reference/structure-rules.md
// from the rule catalog.
//
// The document has carried a "GENERATED FILE" marker since it was written, but
// no generator existed, so it was hand-maintained and drifted: catalog entries
// were added without the corresponding rows, and only the catalog test noticed.
// This command closes that gap. Run it after changing the catalog.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"structure-health/internal/rules"
)

func main() {
	out := flag.String("out", filepath.Join("..", "docs", "reference", "structure-rules.md"), "path to the generated document")
	check := flag.Bool("check", false, "exit non-zero if the file on disk differs from the generated content")
	flag.Parse()

	rendered := render()

	if *check {
		current, err := os.ReadFile(*out) // #nosec G304 -- operator-supplied output path.
		if err != nil {
			fmt.Fprintf(os.Stderr, "read %s: %v\n", *out, err)
			os.Exit(1)
		}
		if !bytes.Equal(current, rendered) {
			fmt.Fprintf(os.Stderr, "%s is stale; run: go run ./cmd/gen-structure-rules\n", *out)
			os.Exit(1)
		}
		return
	}

	if err := os.WriteFile(*out, rendered, 0o644); err != nil { // #nosec G306 -- documentation artifact.
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *out, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s (%d rules)\n", *out, len(rules.Catalog()))
}

func render() []byte {
	var b bytes.Buffer
	b.WriteString("<!-- GENERATED FILE: structure-health rules docs. DO NOT EDIT. -->\n\n")
	b.WriteString("# Structural Rule Catalog\n\n")
	b.WriteString("This page is generated from the Structure Health rule catalog.\n")
	b.WriteString("Regenerate with `go run ./cmd/gen-structure-rules` from `api/`.\n\n")
	b.WriteString("| Code | Target kind | Severity | Enforcement | Claim | What it checks | Remediation |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")

	catalog := rules.Catalog()
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].Code < catalog[j].Code })
	for _, e := range catalog {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			e.Code, e.TargetKind, e.Severity, e.Enforcement, e.Claim, e.WhatItChecks, e.Remediation)
	}

	b.WriteString("\n## Coverage Matrix\n\n")
	b.WriteString("| Target kind | Rules | Enforced | Advisory | None | Reachable | Callers |\n")
	b.WriteString("|---|---:|---:|---:|---:|---|---:|\n")
	for _, row := range rules.Coverage() {
		reachable := "no"
		if row.Reachable {
			reachable = "yes"
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %s | %d |\n",
			row.TargetKind, row.RuleCount, row.Enforced, row.Advisory, row.Unenforced, reachable, row.CallerCount)
	}
	return b.Bytes()
}
