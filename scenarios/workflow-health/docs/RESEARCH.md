# Research — Workflow Health

## Purpose

Record facts and references that shaped the initial Workflow Health contract.

## Capability Fit

Workflow Health is distinct from Browser Automation Studio and Test Genie:

- Browser Automation Studio owns browser execution primitives, schema/linting of workflow JSON, and artifact production.
- Test Genie owns suite orchestration and phase reports.
- Workflow Health owns BAS asset discovery across scenarios, validation policy, maturity assessment, safe execution orchestration, deterministic fixes, Search Hub leaves, and the shared validation-provider contract for the workflow phase.

This split avoids making Test Genie a workflow search engine and avoids bloating BAS with scenario maturity, requirements traceability, or Test Genie provider semantics.

## References

- `scenarios/test-genie/docs/phases/README.md` documents the current native Playbooks phase.
- `scenarios/test-genie/api/internal/orchestrator/phases/phase_validationprovider.go` is the provider delegation model to follow.
- `scenarios/test-genie/api/internal/playbooks/registry/builder.go` is source material for catalog compatibility.
- `scenarios/browser-automation-studio/api/services/workflow/executions.go` is source material for BAS-backed execution.
- `docs/reference/health-maturity-assessments.md` is the shared maturity response guidance.
- `docs/reference/ai-search-routing.md` is the expected Search Hub routing context for typed workflow leaves.
