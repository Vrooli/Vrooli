package rules

import (
	"strings"

	apiRules "structure-health/internal/rules"
)

type (
	CatalogEntry = apiRules.CatalogEntry
	CoverageRow  = apiRules.CoverageRow
)

func Catalog() []CatalogEntry { return apiRules.Catalog() }
func Coverage() []CoverageRow { return apiRules.Coverage() }

// GeneratedMarkdown intentionally reads the linked API catalog at build time;
// CLI freshness therefore includes api/internal/rules in service.json.
func GeneratedMarkdown() string {
	entries := Catalog()
	rows := Coverage()
	var b strings.Builder
	b.WriteString("<!-- GENERATED FILE: structure-health rules docs. DO NOT EDIT. -->\n\n")
	b.WriteString("# Structural Rule Catalog\n\n")
	b.WriteString("This page is generated from the Structure Health rule catalog.\n\n")
	b.WriteString("| Code | Target kind | Severity | Enforcement | Claim | What it checks | Remediation |\n|---|---|---|---|---|---|---|\n")
	for _, entry := range entries {
		b.WriteString("| ")
		b.WriteString(strings.Join([]string{entry.Code, entry.TargetKind, entry.Severity, string(entry.Enforcement), entry.Claim, entry.WhatItChecks, entry.Remediation}, " | "))
		b.WriteString(" |\n")
	}
	b.WriteString("\n## Coverage Matrix\n\n| Target kind | Rules | Enforced | Advisory | None | Reachable | Callers |\n|---|---:|---:|---:|---:|---|---:|\n")
	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(strings.Join([]string{row.TargetKind, itoa(row.RuleCount), itoa(row.Enforced), itoa(row.Advisory), itoa(row.Unenforced), boolString(row.Reachable), itoa(row.CallerCount)}, " | "))
		b.WriteString(" |\n")
	}
	return b.String()
}

func boolString(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func itoa(v int) string {
	// Avoid fmt in this tiny generated renderer and keep its output stable.
	const digits = "0123456789"
	if v == 0 {
		return "0"
	}
	if v < 0 {
		return "-" + itoa(-v)
	}
	var out []byte
	for v > 0 {
		out = append(out, digits[v%10])
		v /= 10
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
