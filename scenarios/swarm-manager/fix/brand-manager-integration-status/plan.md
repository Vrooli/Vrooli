# Fix GenerateOptions Endpoint to Report Actual Integration Availability

## Required Reading

```bash
prompt-manager skill read api-steer test
```

## 1. Purpose

Align the `GET /api/v1/brands/generate/options` endpoint with the actual provider configuration so the UI accurately reports which AI integrations (Ollama, OpenRouter) are available.

## 2. Problem Statement

The `GenerateOptions` handler in `scenarios/brand-manager/api/handlers/scanner_plugin.go` (lines 304-339) hardcodes `available: false` for Ollama and OpenRouter providers. This contradicts the actual runtime configuration — the generation endpoints (`POST /api/v1/brands/{id}/generate`) correctly detect providers via `h.aiChain()` using config checks (`h.cfg.OllamaURL != ""`, `h.cfg.OpenRouterAPIKey != ""`). The options endpoint needs the same logic.

**Impact:** Users see "Not configured" in the UI even when providers are properly set up, leading to confusion and support burden.

## 3. Scope

**In scope:**
- Update `GenerateOptions` handler to check `h.cfg.OllamaURL` and `h.cfg.OpenRouterAPIKey`
- Optional lightweight health check for Ollama reachability
- Tests covering all config permutations
- No UI changes needed (frontend already handles `available: true/false` correctly)

**Out of scope:**
- Refactoring the `aiChain()` method
- Adding new providers
- Changing the response shape or adding new fields
- Modifying the actual generation endpoints

## 4. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/brand-manager/api/handlers/scanner_plugin.go` (L304-339) | `GenerateOptions` handler — hardcoded provider list |
| `scenarios/brand-manager/api/handlers/generate.go` (L51-74) | `aiChain()` — correct config-based provider detection |
| `scenarios/brand-manager/api/config/config.go` (L94-114) | Config fields: `OllamaURL`, `OpenRouterAPIKey` |
| `scenarios/brand-manager/api/aigen/provider.go` | Provider interface with `Available(ctx)` method |
| `scenarios/brand-manager/api/handlers/scanner_plugin_test.go` (L295+) | Existing `TestGenerateOptions` — structure-only, no availability checks |
| `scenarios/brand-manager/ui/src/components/generate-options.tsx` | UI rendering — already handles dynamic availability |

### Existing Pattern
`aiChain()` builds a provider list by checking config strings:
```go
if h.cfg.OllamaURL != "" {
    providers = append(providers, aigen.NewOllamaProvider(...))
}
if h.cfg.OpenRouterAPIKey != "" {
    providers = append(providers, aigen.NewOpenRouterProvider(...))
}
```

## 5. Target End State

- `GenerateOptions` returns `available: true` for providers whose config is present
- Optionally, Ollama availability includes a lightweight reachability check with short timeout
- Tests verify all permutations: both configured, only Ollama, only OpenRouter, neither
- UI accurately reflects provider status without any frontend changes

## 6. Implementation Strategy

### Phase 1: Update GenerateOptions Handler
1. In `scanner_plugin.go`, replace the hardcoded `available: false` values with config checks:
   - Ollama: `h.cfg.OllamaURL != ""`
   - OpenRouter: `h.cfg.OpenRouterAPIKey != ""`
2. If health check decision is YES: add a goroutine or short-timeout HTTP GET to Ollama's `/api/tags` endpoint to verify reachability

### Phase 2: Update Tests
1. Expand `TestGenerateOptions` in `scanner_plugin_test.go` to cover:
   - Default config (neither configured) → both `available: false`
   - Ollama URL set → Ollama `available: true`, OpenRouter `available: false`
   - OpenRouter key set → OpenRouter `available: true`, Ollama `available: false`
   - Both set → both `available: true`
2. Use `setupMockServerWithConfig(t, cfg)` with modified config values

### Phase 3: Verify
1. `go build ./...` — compiles cleanly
2. `go test ./... -timeout 300s` — all tests pass
3. Manual spot-check via running scenario (optional)

## 7. Testing Plan

| Test Case | Config | Expected Ollama | Expected OpenRouter |
|-----------|--------|----------------|-------------------|
| Neither configured | Default | `false` | `false` |
| Ollama only | `OllamaURL: "http://localhost:11434"` | `true` | `false` |
| OpenRouter only | `OpenRouterAPIKey: "sk-test"` | `false` | `true` |
| Both configured | Both set | `true` | `true` |

## 8. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Health check adds latency to options endpoint | Medium | Use short timeout (1-2s), or skip health check entirely and rely on config presence |
| Health check fails intermittently (Ollama cold start) | Low | Config-presence check is sufficient; health check is optional enhancement |

## 9. Non-goals / Prohibited Patterns
- Do not restructure the response format
- Do not add provider-specific error details to the options response
- Do not modify `aiChain()` or the generation endpoints

## 10. Definition of Done
- [ ] `GenerateOptions` returns dynamic `available` based on config
- [ ] 4 test cases covering all config permutations pass
- [ ] `go build ./...` and `go test ./...` succeed
- [ ] No UI changes required (frontend already handles it)
