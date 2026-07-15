package main

import (
	"fmt"
	"sort"
	"strings"
)

// renderSummary produces a deterministic Markdown summary (no timestamps) that
// mirrors the JSON payload for human review.
func renderSummary(inv *Inventory) string {
	var b strings.Builder
	b.WriteString("# swarm-manager persisted-state inventory (Phase 1)\n\n")
	b.WriteString("Read-only snapshot for the declarative-operations state migration (Phase 8).\n")
	b.WriteString("Deterministic and byte-stable: no timestamps; two runs over unchanged state match.\n\n")

	b.WriteString("## Roots\n\n")
	fmt.Fprintf(&b, "- resolved from: `%s`\n", inv.Roots.ResolvedFrom)
	fmt.Fprintf(&b, "- data:  `%s` (exists=%t)\n", inv.Roots.Data.Path, inv.Roots.Data.Exists)
	fmt.Fprintf(&b, "- state: `%s` (exists=%t)\n", inv.Roots.State.Path, inv.Roots.State.Exists)
	fmt.Fprintf(&b, "- cache: `%s` (exists=%t)\n", inv.Roots.Cache.Path, inv.Roots.Cache.Exists)
	fmt.Fprintf(&b, "- config file: `%s` (exists=%t)\n", inv.Roots.ConfigFile.Path, inv.Roots.ConfigFile.Exists)
	if len(inv.Roots.ShadowNamespacesPresent) > 0 {
		fmt.Fprintf(&b, "- **shadow namespaces present**: %s\n", strings.Join(inv.Roots.ShadowNamespacesPresent, ", "))
	}
	b.WriteString("\n")

	b.WriteString("## Totals\n\n")
	fmt.Fprintf(&b, "- files scanned: %d\n", inv.Totals.FilesScanned)
	fmt.Fprintf(&b, "- bytes: %d\n", inv.Totals.Bytes)
	fmt.Fprintf(&b, "- primary objects: %d\n", inv.Totals.ObjectCount)
	fmt.Fprintf(&b, "- anomalies: %d\n", inv.Totals.AnomalyCount)
	fmt.Fprintf(&b, "- referential findings: %d\n", inv.Totals.FindingCount)
	fmt.Fprintf(&b, "- master content hash: `%s`\n\n", inv.Totals.ContentHash)

	b.WriteString("## Object classes\n\n")
	b.WriteString("| class | kind | count | bytes | content hash |\n|---|---|---:|---:|---|\n")
	for _, c := range inv.Classes {
		fmt.Fprintf(&b, "| %s | %s | %d | %d | `%s` |\n", c.Class, c.Kind, c.Count, c.Bytes, c.ContentHash)
	}
	b.WriteString("\n")

	// Status distributions for primary/state classes that have them.
	b.WriteString("## Status distributions\n\n")
	for _, c := range inv.Classes {
		if len(c.ByStatus) == 0 {
			continue
		}
		fmt.Fprintf(&b, "- **%s**: %s\n", c.Class, kvString(c.ByStatus))
		if len(c.ByKind) > 0 {
			fmt.Fprintf(&b, "  - by kind: %s\n", kvString(c.ByKind))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Plan-ref usage\n\n")
	fmt.Fprintf(&b, "- total: %d, managed: %d, **unmanaged: %d**\n", inv.PlanRefs.Total, inv.PlanRefs.Managed, inv.PlanRefs.Unmanaged)
	for _, d := range inv.PlanRefs.Details {
		fmt.Fprintf(&b, "  - `%s` — %s (provider=%q plan_id=%q role=%q)\n", d.Owner, d.Reason, d.Provider, d.PlanID, d.Role)
	}
	b.WriteString("\n")

	b.WriteString("## Ownership\n\n")
	fmt.Fprintf(&b, "- global run-owner index present: %t\n", inv.Ownership.GlobalRunOwnerIndexPresent)
	fmt.Fprintf(&b, "- scope run-owner indexes: %d\n", inv.Ownership.ScopeRunOwnerIndexes)
	fmt.Fprintf(&b, "- engagement-owners present: %t\n", inv.Ownership.EngagementOwnersPresent)
	fmt.Fprintf(&b, "- ambiguous run owners: %d\n", len(inv.Ownership.AmbiguousRunOwners))
	for _, a := range inv.Ownership.AmbiguousRunOwners {
		fmt.Fprintf(&b, "  - `%s` → %s (%s)\n", a.RunID, strings.Join(a.Owners, ", "), a.Source)
	}
	b.WriteString("\n")

	b.WriteString("## Expected-but-absent state\n\n")
	if len(inv.ExpectedAbsent) == 0 {
		b.WriteString("- none\n")
	}
	for _, e := range inv.ExpectedAbsent {
		fmt.Fprintf(&b, "- `%s/%s` — %s\n", e.Root, e.RelPath, e.Note)
	}
	b.WriteString("\n")

	b.WriteString("## Referential findings\n\n")
	if len(inv.ReferentialFindings) == 0 {
		b.WriteString("- none\n")
	}
	byType := map[string]int{}
	for _, f := range inv.ReferentialFindings {
		byType[f.Type]++
	}
	for _, t := range sortedKeys(byType) {
		fmt.Fprintf(&b, "- %s: %d\n", t, byType[t])
	}
	if len(inv.ReferentialFindings) > 0 {
		b.WriteString("\n<details><summary>all findings</summary>\n\n")
		for _, f := range inv.ReferentialFindings {
			fmt.Fprintf(&b, "- [%s] `%s` → `%s` %s\n", f.Type, f.From, f.To, f.Detail)
		}
		b.WriteString("\n</details>\n")
	}
	b.WriteString("\n")

	b.WriteString("## Anomalies (unreadable / invalid state — reported, never dropped)\n\n")
	if len(inv.Anomalies) == 0 {
		b.WriteString("- none\n")
	}
	byAType := map[string]int{}
	for _, a := range inv.Anomalies {
		byAType[a.Type]++
	}
	for _, t := range sortedKeys(byAType) {
		fmt.Fprintf(&b, "- %s: %d\n", t, byAType[t])
	}
	if len(inv.Anomalies) > 0 {
		b.WriteString("\n<details><summary>all anomalies</summary>\n\n")
		for _, a := range inv.Anomalies {
			fmt.Fprintf(&b, "- [%s] `%s` — %s\n", a.Type, a.RelPath, a.Detail)
		}
		b.WriteString("\n</details>\n")
	}
	return b.String()
}

func kvString(m map[string]int) string {
	keys := sortedKeys(m)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		label := k
		if label == "" {
			label = "<empty>"
		}
		parts = append(parts, fmt.Sprintf("%s=%d", label, m[k]))
	}
	return strings.Join(parts, ", ")
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
