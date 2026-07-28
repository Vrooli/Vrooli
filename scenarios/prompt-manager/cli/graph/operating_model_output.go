package graph

import (
	"fmt"
	"sort"
	"strings"
)

func printOperatingModelValidation(resp operatingModelValidationResponse) {
	fmt.Println("Status")
	fmt.Printf("Validated %d operating model(s): %d error(s), %d warning(s).\n\n", len(resp.Models), resp.Validation.Errors, resp.Validation.Warnings)
	fmt.Println("Triage")
	if len(resp.Validation.Findings) == 0 {
		fmt.Println("- clean")
	} else {
		findings := append([]operatingGraphFinding(nil), resp.Validation.Findings...)
		sort.Slice(findings, func(i, j int) bool {
			if findings[i].Severity != findings[j].Severity {
				return findings[i].Severity < findings[j].Severity
			}
			return findings[i].Rule < findings[j].Rule
		})
		for _, f := range findings {
			loc := ""
			if f.SourcePath != "" {
				loc = fmt.Sprintf(" (%s", f.SourcePath)
				if f.Line > 0 {
					loc += fmt.Sprintf(":%d", f.Line)
				}
				loc += ")"
			}
			fmt.Printf("- [%s] %s: %s%s\n", strings.ToUpper(f.Severity), f.Rule, f.Detail, loc)
		}
	}
	fmt.Println("\nNext Steps")
	if resp.Validation.Errors > 0 {
		fmt.Println("Fix error findings before treating the operating model as an enforceable contract.")
	} else if resp.Validation.Warnings > 0 {
		fmt.Println("Review warning findings and decide whether they are accepted target-state gaps.")
	} else {
		fmt.Println("No action required.")
	}
}

func printOperatingModelDiffGroup(title string, diffs []operatingGraphDiff, kind string) {
	fmt.Println(title)
	var printed bool
	for _, d := range diffs {
		if d.Kind != kind {
			continue
		}
		printed = true
		loc := d.SourcePath
		if loc != "" && d.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, d.Line)
		}
		if loc == "" {
			loc = "unknown source"
		}
		fmt.Printf("- [%s] %s\n", d.Relationship, loc)
		fmt.Printf("  %s\n", d.Detail)
		if d.ProducerTeam != "" {
			fmt.Printf("  Producer team: %s\n", d.ProducerTeam)
		}
		if d.RuntimePath != "" {
			fmt.Printf("  Runtime file: %s\n", d.RuntimePath)
		}
		if len(d.AcceptableFields) > 0 {
			fmt.Printf("  Acceptable runtime fields: %s\n", strings.Join(d.AcceptableFields, ", "))
		}
		for _, suggestion := range d.Suggestions {
			fmt.Printf("  Suggested fix: %s\n", suggestion)
		}
	}
	if !printed {
		fmt.Println("- clean")
	}
}

func printOperatingModelCoverage(resp operatingModelCoverageResponse) {
	fmt.Println("Status")
	fmt.Printf("Analyzed %d operating model(s).\n\n", len(resp.Coverage))
	for _, cov := range resp.Coverage {
		fmt.Printf("Model: %s", cov.GraphID)
		if cov.Team != "" {
			fmt.Printf(" team=%s", cov.Team)
		}
		if cov.Source.Path != "" {
			fmt.Printf(" source=%s:%d", cov.Source.Path, cov.Source.Line)
		}
		fmt.Println()

		fmt.Println("\nRelationship Coverage")
		if len(cov.Relationships) == 0 {
			fmt.Println("- none")
		} else {
			for _, rel := range cov.Relationships {
				fmt.Printf("- %s: runtime declared %d, graph shown %d, matched %d, graph-only %d, runtime-only %d",
					rel.Relationship, rel.RuntimeDeclared, rel.GraphShown, rel.Matched, rel.GraphOnly, rel.RuntimeOnly)
				if rel.ValidationSeverity != "" {
					fmt.Printf(" (%s)", rel.ValidationSeverity)
				}
				fmt.Println()
				for _, subtype := range rel.RuntimeSubtypes {
					fmt.Printf("  - %s: runtime declared %d, covered %d, runtime-only %d\n",
						subtype.Relationship, subtype.RuntimeDeclared, subtype.Covered, subtype.RuntimeOnly)
				}
			}
		}

		fmt.Println("\nPrompt Coverage")
		fmt.Printf("- topic-contract section present: %d/%d graph members\n", cov.Prompts.TopicContractPresent, cov.Prompts.GraphMembers)
		fmt.Printf("- topic-contract source path: %d/%d graph members\n", cov.Prompts.TopicContractSourceMatched, cov.Prompts.GraphMembers)
		if cov.Prompts.TopicContractSourceKind != "" {
			fmt.Printf("- topic-contract source kind: %s\n", cov.Prompts.TopicContractSourceKind)
		}
		fmt.Printf("- content parity: %s\n", cov.Prompts.TopicContractContentParity)

		fmt.Println("\nDocs Coverage")
		fmt.Printf("- Mermaid graph: %s\n", cov.Docs.MermaidGraph)
		if cov.Docs.RequiredSectionsTotal > 0 {
			fmt.Printf("- required sections: %d/%d present\n", cov.Docs.RequiredSectionsPresent, cov.Docs.RequiredSectionsTotal)
		}
		fmt.Printf("- Topic Catalog table: %s (rows %d, matched %d, graph-only %d, docs-only %d, invalid %d)\n",
			cov.Docs.TopicCatalogTable,
			cov.Docs.TopicCatalogRows,
			cov.Docs.TopicCatalogMatched,
			cov.Docs.TopicCatalogGraphOnly,
			cov.Docs.TopicCatalogDocsOnly,
			cov.Docs.TopicCatalogInvalid,
		)
		fmt.Printf("- Topic Catalog purpose parity: matched %d, mismatch %d, missing-runtime %d\n",
			cov.Docs.TopicCatalogPurposeMatched,
			cov.Docs.TopicCatalogPurposeMismatch,
			cov.Docs.TopicCatalogPurposeMissingRuntime,
		)
		fmt.Printf("- Decisions table: %s (rows %d, matched %d, graph-only %d, docs-only %d, invalid %d)\n",
			cov.Docs.DecisionsTable,
			cov.Docs.DecisionsRows,
			cov.Docs.DecisionsMatched,
			cov.Docs.DecisionsGraphOnly,
			cov.Docs.DecisionsDocsOnly,
			cov.Docs.DecisionsInvalid,
		)
		if cov.Docs.DecisionsRows > 0 {
			fmt.Printf("- Decisions metadata: complete %d, incomplete %d, weak accepted effects %d\n",
				cov.Docs.DecisionsMetadataComplete,
				cov.Docs.DecisionsMetadataIncomplete,
				cov.Docs.DecisionsAcceptedEffectWeak,
			)
		}
		if cov.Docs.ExternalInputsTable != "" {
			fmt.Printf("- External Inputs / Triggers table: %s (rows %d, backed %d, unbacked %d)\n",
				cov.Docs.ExternalInputsTable,
				cov.Docs.ExternalInputsRows,
				cov.Docs.ExternalInputsBackedRows,
				cov.Docs.ExternalInputsUnbackedRows,
			)
		}
		if cov.Docs.OutputsTable != "" {
			fmt.Printf("- Outputs / Downstream Consumers table: %s (rows %d, backed %d, unbacked %d)\n",
				cov.Docs.OutputsTable,
				cov.Docs.OutputsRows,
				cov.Docs.OutputsBackedRows,
				cov.Docs.OutputsUnbackedRows,
			)
		}
		if cov.Docs.FeedbackSteps > 0 {
			fmt.Printf("- Feedback loop: anchored steps %d/%d, unbacked references %d\n",
				cov.Docs.FeedbackAnchoredSteps,
				cov.Docs.FeedbackSteps,
				cov.Docs.FeedbackUnbackedReferences,
			)
		}
		if cov.Docs.GapsItems > 0 {
			fmt.Printf("- Current Implementation Gaps: anchored items %d/%d, target-state dispositions %d/%d\n",
				cov.Docs.GapsAnchoredItems,
				cov.Docs.GapsItems,
				cov.Docs.GapsTargetStateItems,
				cov.Docs.GapsItems,
			)
		}
		if cov.Docs.AdoptionValidationCommands > 0 {
			fmt.Printf("- Adoption / Validation commands: %d/3 present\n", cov.Docs.AdoptionValidationCommands)
		}
		if cov.Docs.PlanOfRecordRegistration != "" {
			fmt.Printf("- plan-of-record registration: %s\n", cov.Docs.PlanOfRecordRegistration)
		}
		if cov.Docs.ReadmeDiscoverability != "" {
			fmt.Printf("- README discoverability: %s\n", cov.Docs.ReadmeDiscoverability)
		}

		fmt.Println("\nExcluded")
		if len(cov.Exclusions) == 0 {
			fmt.Println("- none")
		} else {
			for _, exclusion := range cov.Exclusions {
				fmt.Printf("- %s: %d", exclusion.Kind, exclusion.Count)
				if exclusion.Detail != "" {
					fmt.Printf(" (%s)", exclusion.Detail)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}
	if len(resp.Coverage) == 0 {
		fmt.Println("No checkable operating model matched the filters.")
	}
}

func operatingModelPrimaryGraph(model operatingModelDocument) operatingGraphBlock {
	if len(model.Graphs) > 0 {
		return model.Graphs[0]
	}
	return model.Sections.Graph.operatingGraphBlock
}
