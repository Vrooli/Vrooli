package gatewaycatalog

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/requestvocab"
	"ai-gateway/internal/rolecatalog"
)

func writeReferenceHeader(out *bytes.Buffer) {
	out.WriteString("# Capability Catalog\n\n")
	out.WriteString("<!-- AUTO-GENERATED — do not edit by hand. -->\n")
	out.WriteString("<!-- Source: config/profile-catalog.json, config/inference-role-catalog.json, api/internal/requestvocab, api/internal/providers. -->\n")
	out.WriteString("<!-- Regenerate: cd scenarios/ai-gateway/api && GOWORK=off go run ./cmd/gen-gateway-catalog -->\n")
	out.WriteString("<!-- Verify: cd scenarios/ai-gateway/api && GOWORK=off go run ./cmd/gen-gateway-catalog -check -->\n\n")
	out.WriteString("The tables below are the current inventory of the closed profile vocabulary, the gateway-owned inference and decision roles, the registered provider resources, and every dispatchable role candidate. They are generated from data and code, not edited by hand. Qualification candidates that are not dispatched yet are excluded from the role candidate counts and matrix. When a reason is shared by several candidate rows, the matrix adds the role and provider so each row stays self-describing.\n\n")
}

func writeProfileTable(out *bytes.Buffer, catalog *ProfileCatalog) {
	out.WriteString("## Profiles\n\n")
	out.WriteString("| Key | Proto | Label key | Description key |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, row := range catalog.Sorted() {
		fmt.Fprintf(out, "| `%s` | `%s` | `%s` | `%s` |\n",
			mdCell(row.Key), mdCell(row.Proto), mdCell(row.LabelKey), mdCell(row.DescriptionKey))
	}
}

func writePrivacyClassTable(out *bytes.Buffer) {
	out.WriteString("\n## Privacy Classes\n\n")
	out.WriteString("Privacy classes are closed by the shared proto enum and are declared here for operator reference. `UNSPECIFIED` is a wire default, not a selectable class.\n\n")
	out.WriteString("| Value | Proto |\n")
	out.WriteString("| --- | --- |\n")
	for _, entry := range privacyClassCatalog() {
		fmt.Fprintf(out, "| `%s` | `%s` |\n", entry.value, entry.proto)
	}
}

func writeRequestKindTable(out *bytes.Buffer) {
	out.WriteString("\n## Request Kinds and Resource Capabilities\n\n")
	out.WriteString("Each selectable proto request kind declares its required input modalities, output kind, and resource-owned capability aliases. Resources can also declare additional resource-specific capabilities; those do not change gateway request semantics.\n\n")
	out.WriteString("| Request kind | Required input modalities | Output kind | Accepted resource capabilities |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, definition := range requestvocab.RequestKindCatalog() {
		fmt.Fprintf(out, "| `%s` | %s | `%s` | %s |\n",
			definition.Kind.String(), codeList(modalityNames(definition.RequiredInputs)), mdCell(definition.OutputKind), codeList(definition.RequiredCapabilities))
	}
}

func writeRoleTable(out *bytes.Buffer, roles *rolecatalog.RoleCatalog) {
	out.WriteString("\n## Roles\n\n")
	out.WriteString("| Role | Kind | Timeout (ms) | Candidates |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, name := range sortedRoleNames(roles) {
		role := roles.Roles[name]
		kind, _ := requestvocab.NormalizeRoleKind(role.Kind)
		timeout := "—"
		if role.TimeoutMS > 0 {
			timeout = fmt.Sprintf("%d", role.TimeoutMS)
		}
		fmt.Fprintf(out, "| `%s` | %s | %s | %d |\n", mdCell(name), mdCell(kind), timeout, len(role.Candidates))
	}
}

func writeProviderTable(out *bytes.Buffer, specs []providers.ProviderSpec) error {
	sortedSpecs := append([]providers.ProviderSpec(nil), specs...)
	sort.Slice(sortedSpecs, func(i, j int) bool { return sortedSpecs[i].ID < sortedSpecs[j].ID })
	if err := validateProviderRows(sortedSpecs); err != nil {
		return err
	}
	out.WriteString("\n## Providers\n\n")
	out.WriteString("| ID | Family | Locality | Text | Spend |\n")
	out.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, spec := range sortedSpecs {
		fmt.Fprintf(out, "| `%s` | %s | %s | %s | %s |\n",
			mdCell(spec.ID), mdCell(spec.Family), mdCell(spec.Locality), yesNo(spec.Text), yesNo(spec.SpendCapable))
	}
	return nil
}

func validateProviderRows(specs []providers.ProviderSpec) error {
	seen := map[string]bool{}
	for _, spec := range specs {
		if strings.TrimSpace(spec.ID) == "" {
			return fmt.Errorf("provider spec has an empty id")
		}
		if seen[spec.ID] {
			return fmt.Errorf("provider %q is declared more than once", spec.ID)
		}
		seen[spec.ID] = true
	}
	return nil
}

func writeCandidateMatrix(out *bytes.Buffer, roles *rolecatalog.RoleCatalog) {
	out.WriteString("\n## Role to Candidate Matrix\n\n")
	out.WriteString("| Role | Provider | Resource role | Reason |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	reasonCounts := map[string]int{}
	for _, name := range sortedRoleNames(roles) {
		for _, candidate := range roles.Roles[name].Candidates {
			reasonCounts[candidate.Reason]++
		}
	}
	for _, name := range sortedRoleNames(roles) {
		for _, candidate := range roles.Roles[name].Candidates {
			fmt.Fprintf(out, "| `%s` | `%s` | `%s` | %s |\n",
				mdCell(name), mdCell(candidate.Provider), mdCell(candidate.ResourceRole), candidateReason(name, candidate, reasonCounts))
		}
	}
}

func candidateReason(role string, candidate rolecatalog.Candidate, reasonCounts map[string]int) string {
	reason := candidate.Reason
	if reasonCounts[candidate.Reason] <= 1 {
		return mdCell(reason)
	}
	reason = strings.TrimSpace(reason)
	if !strings.HasSuffix(reason, ".") {
		reason += "."
	}
	return mdCell(reason + fmt.Sprintf(" (role `%s`, provider `%s`)", role, candidate.Provider))
}

func writeEscalationCandidates(out *bytes.Buffer, roles *rolecatalog.RoleCatalog) {
	out.WriteString("\n## Semantic Escalation Candidates (Caller Opt-In)\n\n")
	out.WriteString("These candidates are considered only after the ordinary route and schema-repair stages fail and the caller has opted in. They are intentionally separate from ordinary availability fallbacks.\n\n")
	out.WriteString("| Role | Provider | Resource role | Reason |\n")
	out.WriteString("| --- | --- | --- | --- |\n")
	for _, name := range sortedRoleNames(roles) {
		for _, candidate := range roles.Roles[name].SemanticEscalationCandidates {
			fmt.Fprintf(out, "| `%s` | `%s` | `%s` | %s |\n",
				mdCell(name), mdCell(candidate.Provider), mdCell(candidate.ResourceRole), mdCell(candidate.Reason))
		}
	}
}

func writeQualificationCandidates(out *bytes.Buffer, roles *rolecatalog.RoleCatalog) {
	out.WriteString("\n## Qualification Candidates (Not Dispatchable)\n\n")
	out.WriteString("Qualification candidates are staged for evidence collection and are never selected by runtime routing. Move one into the dispatchable candidate list only after its qualification evidence is accepted.\n\n")
	out.WriteString("| Role | Provider | Resource role | Status | Evidence | Reason |\n")
	out.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, name := range sortedRoleNames(roles) {
		for _, qualified := range roles.Roles[name].QualificationCandidates {
			candidate := qualified.Candidate
			fmt.Fprintf(out, "| `%s` | `%s` | `%s` | %s | %s | %s |\n",
				mdCell(name), mdCell(candidate.Provider), mdCell(candidate.ResourceRole), mdCell(qualified.Status), mdCell(qualified.Evidence), mdCell(candidate.Reason))
		}
	}
}
