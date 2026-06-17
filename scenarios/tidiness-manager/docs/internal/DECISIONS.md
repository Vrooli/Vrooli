# Decisions

## Quality Health Boundary

Tidiness Manager no longer owns lint/type/static-quality policy. Quality Health owns those contracts and Test Genie exposes them through the `quality` phase.

## Active Tidiness Endpoint

`scenario-validation/v1.ScenarioValidationService.ValidateScenario` is the active maintainability validation endpoint used by Test Genie. The response carries the shared validation status and maturity assessment, with the native tidiness scan summary packed into `native_detail`. Legacy light scan endpoints remain for metrics workflows.

## Documentation Contract

The scenario follows the React/Vite scenario documentation layout: concepts, reference, operations, guides, and internal ledgers. Internal ledgers are preserved under `docs/internal/`.

## Future Decisions

Record new detector ownership, campaign behavior, persistence changes, or integration boundary changes here before implementing broad changes.
