# Research & References

## Uniqueness Check
No existing Vrooli scenario handles centralized branding lifecycle management. The deleted Brand Manager and App Personalizer scenarios are being replaced with this clean rewrite.

## Related Scenarios
- **LPBS**: Has `.vrooli/branding.json` with site_name, tagline, theme colors, logo URLs, SEO — Brand Manager's data model is a superset and can import this format
- **Scenario Auditor**: Will consume Brand Manager's HTTP provider for branding compliance rules
- **agent-manager**: Used for agent-assisted application of brands to complex scenarios
- **web-console**: Reference implementation for AIProviderChain pattern (`ai_provider.go`)
- **stream-of-consciousness-analyzer**: Reference for two-tier Ollama/OpenRouter priority system

## Key Reference Files
- `scenarios/web-console/api/ai_provider.go` — AIProvider interface + chain pattern
- `scenarios/scenario-auditor/api/external_rules.go` — Provider registration pattern
- `scenarios/scenario-auditor/api/external_rules_tidiness_manager.go` — Example external provider
- `scenarios/swarm-manager/api/internal/storage/json_store.go` — Atomic file write pattern
- `scenarios/landing-page-business-suite/.vrooli/branding.json` — Existing branding data format
- `scenarios/landing-page-business-suite/.vrooli/schemas/branding.schema.json` — Existing branding schema

## Workshop Summary
4 rounds of workshop produced decisions on:
- AI provider strategy: Ollama-first + OpenRouter fallback
- Validation approach: Inline code markers as ground truth
- SQLite schema: Single DB with brands, brand_versions, assignments, assets tables
- Version storage: Explicit snapshots with JSON column
- Discovery: Import everything with confidence scores
- Application: Structured prompt + API poll + post-validation for agent-assisted tier
- Testing: Unit + integration + interface-based test doubles
- WCAG: Contrast validation at generation time + Lighthouse for runtime a11y
