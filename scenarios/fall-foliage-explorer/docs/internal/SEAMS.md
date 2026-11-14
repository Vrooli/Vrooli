# Seams

## API To UI

The UI depends on the API envelope and endpoint paths documented in [DOC: docs/reference/api-endpoints.md]. Preserve the `status`, `data`, and `error` envelope unless every consumer is migrated.

## API To CLI

The CLI decodes API responses through [CODE: cli/internal/support/types.go] and [CODE: cli/internal/support/support.go]. API response changes must be additive or coordinated with CLI type updates.

## Lifecycle To Runtime

Runtime process ownership belongs to [CODE: .vrooli/service.json] and [CODE: Makefile]. Direct binary or Node starts are outside the supported seam.

## Prediction Integration

Ollama failures must remain non-fatal for prediction requests. The fallback path in [CODE: api/main.go#predictHandler] preserves scenario usefulness when the AI resource is unavailable.
