# OpenRouter Problems and Improvements

## Current state

OpenRouter is a mature `cloud-api` resource with a Go-native CLI, role policy,
credentials, health checks, structured-output handling, image generation, and
typed provider failures.

## Known debt

- The policy engine now delegates to `packages/ai-provider-go/policy`; the
  resource keeps only OpenRouter-specific validation and command wiring.
- The command layer still contains OpenRouter-only image and spend-cap paths.
  Those are intentional: spend caps, `/images`, and provider pinning are
  OpenRouter-specific capabilities, not part of the shared OpenAI-compatible
  contract.
- `policycmd` and `ensure` still carry some resource-local formatting. They
  should converge on the shared handlers when their test fixtures are migrated.
- The initial model catalog is sourced from an operator survey. Re-verify new
  or changed models against the live catalog and update provenance before
  promoting them.

## Not a defect

- `list-models` returns empty output without a real API key. That is expected.
- Usage analytics return no data until the account has usage.

## Follow-up

- Keep concrete model slugs in `model-policy.json`; never add them to consumer
  code.
- Extend `resources/internal/aiproviderapp` only for behavior shared by every
  OpenAI-compatible provider. OpenRouter-only behavior stays here.
