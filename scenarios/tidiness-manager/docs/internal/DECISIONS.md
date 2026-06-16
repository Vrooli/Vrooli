# Decisions

## Quality Health Boundary

Tidiness Manager no longer owns lint/type/static-quality policy. Quality Health owns those contracts and Test Genie exposes them through the `quality` phase.

## Active Tidiness Endpoint

`POST /api/v1/scan/tidiness` is the active maintainability endpoint used by Test Genie. Legacy light scan endpoints remain for compatibility and metrics workflows.

## Documentation Contract

The scenario follows the React/Vite scenario documentation layout: concepts, reference, operations, guides, and internal ledgers. Internal ledgers are preserved under `docs/internal/`.

## Future Decisions

Record new detector ownership, campaign behavior, persistence changes, or integration boundary changes here before implementing broad changes.
