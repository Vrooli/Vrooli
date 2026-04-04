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
- Add async Ollama health check (GET `/api/tags`) with 2s timeout to distinguish "configured" from "configured and reachable"
- OpenRouter uses config-presence only (no health check)
- Table-driven tests covering all config permutations
- No UI changes needed (frontend already handles `available: true/false` correctly)

**Out of scope:**
- Refactoring the `aiChain()` method
- Adding new providers
- Changing the response shape or adding new fields
- Modifying the actual generation endpoints
- OpenRouter key validation or health checking

## 4. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/brand-manager/api/handlers/scanner_plugin.go` (L304-339) | `GenerateOptions` handler — hardcoded provider list |
| `scenarios/brand-manager/api/handlers/generate.go` (L51-74) | `aiChain()` — correct config-based provider detection |
| `scenarios/brand-manager/api/config/config.go` (L94-114) | Config fields: `OllamaURL`, `OpenRouterAPIKey` |
| `scenarios/brand-manager/api/aigen/provider.go` (L15-28) | Provider interface with `Available(ctx)` method |
| `scenarios/brand-manager/api/handlers/scanner_plugin_test.go` (L299+) | Existing `TestGenerateOptions` — structure-only, no availability checks |
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

- `GenerateOptions` returns `available: true` for Ollama when both configured AND reachable (health check passes within 2s)
- `GenerateOptions` returns `available: true` for OpenRouter when API key is configured (config-presence only)
- Table-driven tests verify all permutations: both configured, only Ollama, only OpenRouter, neither, plus Ollama configured-but-unreachable
- UI accurately reflects provider status without any frontend changes

## 6. Implementation Strategy

### Phase 1: Update GenerateOptions Handler
1. In `scanner_plugin.go`, replace the hardcoded `available: false` values with dynamic checks:
   - **OpenRouter**: `h.cfg.OpenRouterAPIKey != ""` — simple config-presence check
   - **Ollama**: `h.cfg.OllamaURL != ""` AND passes a lightweight HTTP GET to `{OllamaURL}/api/tags` with a 2-second timeout
2. For the Ollama health check:
   - Create a `context.WithTimeout(ctx, 2*time.Second)` 
   - Fire an `http.Get` to `{OllamaURL}/api/tags`
   - If status 200, `available: true`; otherwise `available: false`
   - Run the health check in a goroutine so it doesn't block OpenRouter's config check (both providers can be evaluated concurrently)
3. Manual Entry remains always `available: true` (no change)

### Phase 2: Update Tests
1. Add a new `TestGenerateOptions_Availability` function in `scanner_plugin_test.go` using table-driven subtests with `t.Run()`:
   - `"neither configured"` — default config → both `available: false`
   - `"ollama only"` — OllamaURL set, start a local httptest server to simulate Ollama → Ollama `true`, OpenRouter `false`
   - `"openrouter only"` — OpenRouterAPIKey set → OpenRouter `true`, Ollama `false`
   - `"both configured"` — both set, httptest for Ollama → both `true`
   - `"ollama configured but unreachable"` — OllamaURL points to non-listening address → Ollama `false`, OpenRouter `false`
2. Use `httptest.NewServer` to simulate a reachable Ollama for the health-check cases
3. Existing `TestGenerateOptions` (structural test) remains unchanged

### Phase 3: Verify
1. `go build ./...` — compiles cleanly
2. `go test ./... -timeout 300s` — all tests pass
3. `gofumpt -w .` — code formatted

## 7. Testing Plan

| Test Case | Config | Mock Ollama | Expected Ollama | Expected OpenRouter |
|-----------|--------|-------------|----------------|-------------------|
| Neither configured | Default | N/A | `false` | `false` |
| Ollama only | `OllamaURL` set | httptest 200 | `true` | `false` |
| OpenRouter only | `OpenRouterAPIKey` set | N/A | `false` | `true` |
| Both configured | Both set | httptest 200 | `true` | `true` |
| Ollama unreachable | `OllamaURL` set (bad addr) | None | `false` | `false` |

## 8. Risks + Mitigations

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Ollama health check adds up to 2s latency | Medium | Use goroutine + `context.WithTimeout` so it runs concurrently with other checks; 2s is the worst case (timeout), healthy Ollama responds in <50ms |
| Ollama cold start causes intermittent false-negatives | Low | 2s timeout accommodates typical cold start; if it times out, user can retry — the endpoint is informational |
| httptest server in tests adds complexity | Low | Standard Go pattern; minimal setup. Each subtest can share a single httptest server |

## 9. Non-goals / Prohibited Patterns
- Do not restructure the response format
- Do not add provider-specific error details to the options response
- Do not modify `aiChain()` or the generation endpoints
- Do not validate OpenRouter key format — presence is sufficient
- Do not add background/cached health checks — direct check per request is sufficient for this S-effort fix

## 10. Definition of Done
- [ ] `GenerateOptions` returns dynamic `available` based on config presence (OpenRouter) and config + health check (Ollama)
- [ ] Ollama health check uses 2s timeout, runs async
- [ ] 5 table-driven test cases covering all config permutations + unreachable case pass
- [ ] `go build ./...` and `go test ./...` succeed
- [ ] Code formatted with `gofumpt`
- [ ] No UI changes required (frontend already handles it)
