# Decisions

## Structural duplication remains visible but carries zero debt

`DUPLICATED_BOILERPLATE` uses the shared maturity contract's
`clean_requirement: uncheckable`. `DebtScore` excludes that class while the
finding remains visible at INFO severity. This is deliberately a producer-side
classification: generated code is the only duplication fully excluded.

## Quality Health Boundary

Tidiness Manager no longer owns lint/type/static-quality policy. Quality Health owns those contracts and Test Genie exposes them through the `quality` phase.

## Active Tidiness Endpoint

`scenario-validation/v1.ScenarioValidationService.ValidateScenario` is the active maintainability validation endpoint used by Test Genie. The response carries the shared validation status and maturity assessment, with the native tidiness scan summary packed into `native_detail`. Legacy light scan endpoints remain for metrics workflows.

## Documentation Contract

The scenario follows the React/Vite scenario documentation layout: concepts, reference, operations, guides, and internal ledgers. Internal ledgers are preserved under `docs/internal/`.

## File-role vocabulary remains producer-owned

The file-role vocabulary intentionally resembles Architecture Cartographer's
zone vocabulary, but has no shared source of truth. Tidiness Manager consumes
role declarations as structural classification priors and validates them against
its own `.vrooli/schemas/file-roles.schema.json`; a future shared vocabulary
must be reconciled by its producer rather than coupled here prematurely.

## Block analysis is the sole duplication signal

The persisted per-file percentage pipeline cannot represent normalized clone
groups, their classifications, or line-weighted debt. It no longer emits
duplication issues or percentage labels. Duplication work is produced only by
`buildTidinessScan`, where raw groups remain diagnostic and ranked root-cause
opportunities provide the actionable queue.

## Future Decisions

Record new detector ownership, campaign behavior, persistence changes, or integration boundary changes here before implementing broad changes.
