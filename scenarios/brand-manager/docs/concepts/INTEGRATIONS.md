# Integrations — Brand Manager

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes | API, persistence-backed domains | resolved by `api-core/storage` from the scenario id | API reports unhealthy if unreachable. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario should be started through lifecycle commands. |

## Vrooli Resources

`.vrooli/service.json` declares two optional text-generation resources;
both use `try_start`. The provider chain tries Ollama first and appends
OpenRouter only when `OPENROUTER_API_KEY` is configured.

| Resource | Required? | Startup | Used By | Degradation |
|---|---|---|---|
| ollama | no | `try_start` | First text provider; routed through `resource-ollama gateway generate` using the configured role. | If unavailable or generation fails, the chain tries OpenRouter when configured. |
| openrouter | no | `try_start` | Text fallback; resource policy resolves the `chat.default` role before chat completions. | It is omitted when no API key is configured. If it also fails, text generation returns an all-providers-failed error. |

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| image-tools | optional (`try_start`) | Owns image generation and image-processing infrastructure. | Brand Manager uses its HTTP/Connect API for text-to-image, edits, object removal, vectorization, rasterization, and icon-container operations. Image actions report backend/job errors when unavailable; text generation remains independent. |

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| OpenRouter API | indirect, optional | Used for text fallback when `OPENROUTER_API_KEY` is set. | The shared `ai-provider-go/openai` client calls chat completions; `resource-openrouter policy resolve` supplies the default `chat.default` model. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| ollama | unavailable or generation error | The text chain tries the configured OpenRouter fallback; if no provider succeeds, generation returns an all-providers-failed error. | `api/internal/generation/providers_test.go` |
| openrouter | no configured key, role resolution failure, or chat error | The provider is omitted without a key; otherwise the chain reports its error and fails only if no provider succeeds. | `api/internal/generation/providers_test.go` |
| image-tools | scenario discovery, job, or result retrieval error | The image operation returns a typed backend/job error; text generation and existing brand data are unaffected. | `api/internal/generation/service_test.go` |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DATA.md`](DATA.md) — storage ownership
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
