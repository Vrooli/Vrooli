# Capability Catalog

<!-- AUTO-GENERATED — do not edit by hand. -->
<!-- Source: config/profile-catalog.json, config/inference-role-catalog.json, api/internal/requestvocab, api/internal/providers. -->
<!-- Regenerate: cd scenarios/ai-gateway/api && GOWORK=off go run ./cmd/gen-gateway-catalog -->
<!-- Verify: cd scenarios/ai-gateway/api && GOWORK=off go run ./cmd/gen-gateway-catalog -check -->

The tables below are the current inventory of the closed profile vocabulary, the gateway-owned inference and decision roles, the registered provider resources, and every dispatchable role candidate. They are generated from data and code, not edited by hand. Qualification candidates that are not dispatched yet are excluded from the role candidate counts and matrix. When a reason is shared by several candidate rows, the matrix adds the role and provider so each row stays self-describing.

## Profiles

| Key | Proto | Label key | Description key |
| --- | --- | --- | --- |
| `local-only` | `PROFILE_LOCAL_ONLY` | `profiles.localOnly` | `profiles.localOnlyDesc` |
| `local-first` | `PROFILE_LOCAL_FIRST` | `profiles.localFirst` | `profiles.localFirstDesc` |
| `remote-only` | `PROFILE_REMOTE_ONLY` | `profiles.remoteOnly` | `profiles.remoteOnlyDesc` |
| `quality-first` | `PROFILE_QUALITY_FIRST` | `profiles.qualityFirst` | `profiles.qualityFirstDesc` |
| `cheap-first` | `PROFILE_CHEAP_FIRST` | `profiles.cheapFirst` | `profiles.cheapFirstDesc` |
| `privacy-sensitive` | `PROFILE_PRIVACY_SENSITIVE` | `profiles.privacySensitive` | `profiles.privacySensitiveDesc` |

## Privacy Classes

Privacy classes are closed by the shared proto enum and are declared here for operator reference. `UNSPECIFIED` is a wire default, not a selectable class.

| Value | Proto |
| --- | --- |
| `public` | `PRIVACY_CLASS_PUBLIC` |
| `internal` | `PRIVACY_CLASS_INTERNAL` |
| `confidential` | `PRIVACY_CLASS_CONFIDENTIAL` |
| `secret` | `PRIVACY_CLASS_SECRET` |

## Request Kinds and Resource Capabilities

Each selectable proto request kind declares its required input modalities, output kind, and resource-owned capability aliases. Resources can also declare additional resource-specific capabilities; those do not change gateway request semantics.

| Request kind | Required input modalities | Output kind | Accepted resource capabilities |
| --- | --- | --- | --- |
| `REQUEST_KIND_TEXT_GENERATION` | `text` | `text` | `generate`, `chat`, `completion` |
| `REQUEST_KIND_TEXT_EMBEDDING` | `text` | `vector` | `embedding`, `embeddings` |
| `REQUEST_KIND_STRUCTURED_EXTRACTION` | `text` | `json` | `extract`, `structured`, `json`, `tools`, `generate`, `chat` |
| `REQUEST_KIND_IMAGE_GENERATION` | `text` | `image` | `image_generation`, `image_output` |
| `REQUEST_KIND_VIDEO_GENERATION` | `text` | `video` | `video_generation`, `video_output` |
| `REQUEST_KIND_DETECT_PII` | `text` | `json` | `detect_pii`, `detect`, `pii` |
| `REQUEST_KIND_DECISION` | `text` | `json` | `decide` |

## Roles

| Role | Kind | Timeout (ms) | Candidates |
| --- | --- | --- | --- |
| `author.generator` | typed_value | 300000 | 3 |
| `browser.decide` | decision | 30000 | 5 |
| `browser.extract` | typed_value | 120000 | 4 |
| `browser.locate` | typed_value | 90000 | 4 |
| `browser.plan` | typed_value | 90000 | 4 |
| `chat.uncensored` | typed_value | 240000 | 1 |
| `classify.fast` | typed_value | 45000 | 4 |
| `decide.accurate` | decision | 120000 | 6 |
| `decide.candidates` | decision | 120000 | 15 |
| `decide.fast` | decision | 30000 | 5 |
| `decide.short` | decision | 30000 | 2 |
| `decide.short.accurate` | decision | 120000 | 2 |
| `detect.pii.default` | typed_value | 45000 | 1 |
| `device-control.plan` | typed_value | 90000 | 4 |
| `device.geometry.byok` | typed_value | 120000 | 1 |
| `device.geometry.vrooli_credits` | typed_value | 120000 | 1 |
| `embedding.default` | typed_value | 120000 | 1 |
| `extract.structured` | typed_value | 120000 | 4 |
| `judge.default` | typed_value | 120000 | 4 |
| `locate.visual` | typed_value | 90000 | 4 |
| `research.unrestricted` | typed_value | 240000 | 1 |
| `research.unrestricted.flash` | typed_value | 240000 | 1 |
| `research.unrestricted.large` | typed_value | 240000 | 1 |
| `summarize.audio_tools` | typed_value | 120000 | 1 |
| `summarize.audio_tools.quality` | typed_value | 120000 | 1 |
| `summarize.faithful` | typed_value | 120000 | 1 |
| `ux.visual-critique` | typed_value | 90000 | 1 |
| `write.default` | typed_value | 240000 | 4 |
| `write.diverse` | typed_value | 300000 | 4 |

## Providers

| ID | Family | Locality | Text | Spend |
| --- | --- | --- | --- | --- |
| `abliteration` | openai-compatible | remote | yes | no |
| `ollama` | ollama | local | yes | no |
| `ollaya` | decision | local | no | no |
| `openrouter` | openai-compatible | remote | yes | yes |
| `orcarouter` | openai-compatible | remote | yes | no |
| `pii-detector` | pii-detector | local | no | no |

## Role to Candidate Matrix

| Role | Provider | Resource role | Reason |
| --- | --- | --- | --- |
| `author.generator` | `openrouter` | `code.default` | A generator is a program: it must stay internally consistent across a few hundred lines. code.default is the provider's exposed code-authoring role, so the concrete model stays a policy decision rather than a slug in a runtime surface. |
| `author.generator` | `orcarouter` | `code.default` | Hosted OrcaRouter code-authoring fallback. |
| `author.generator` | `ollama` | `code.local` | Local fallback for a host with no hosted credential. Every authored generator is validated before storage — determinism, declared parameters, no active content, composition at both small and large sizes — so a weaker candidate costs a rejection rather than a bad artifact. |
| `browser.decide` | `ollaya` | `decide.local` | Local fast decision route. |
| `browser.decide` | `ollaya` | `decide.jev.edenai` | Hosted TypeSafe decision route when provisioned. (role `browser.decide`, provider `ollaya`) |
| `browser.decide` | `ollaya` | `decide.jev.typesafe` | Hosted TypeSafe decision route when provisioned. (role `browser.decide`, provider `ollaya`) |
| `browser.decide` | `ollaya` | `decide.clef.cloudflare` | Hosted decision route when provisioned. |
| `browser.decide` | `ai-gateway` | `classify.fast` | Enum-constrained Gateway fallback for choice-only candidate ranking. |
| `browser.extract` | `ollama` | `chat.default` | Local structured extraction keeps page content on the device; ollama has no extract.structured role, so the local generation role carries the schema. |
| `browser.extract` | `openrouter` | `extract.structured` | Hosted extraction fallback under selected source and consent. |
| `browser.extract` | `orcarouter` | `extract.structured` | Hosted extraction fallback under Gateway policy. |
| `browser.extract` | `lpbs` | `extract.structured` | LPBS metered extraction; LPBS owns reserve and settlement. |
| `browser.locate` | `ollama` | `vision.default` | Local-first grounding keeps captured browser frames on the device. |
| `browser.locate` | `openrouter` | `vision.default` | Hosted multimodal fallback under the caller-selected source and consent. |
| `browser.locate` | `orcarouter` | `vision.default` | Hosted grounding fallback under Gateway policy. |
| `browser.locate` | `lpbs` | `vision.default` | Subscription-backed metered grounding; LPBS owns reserve and settlement. |
| `browser.plan` | `ollama` | `chat.default` | Use existing local generation with Gateway schema validation. |
| `browser.plan` | `openrouter` | `extract.structured` | Hosted structured planning fallback under selected source. |
| `browser.plan` | `orcarouter` | `extract.structured` | Hosted structured planning fallback under Gateway policy. |
| `browser.plan` | `lpbs` | `extract.structured` | LPBS metered structured planning; LPBS owns reserve and settlement. |
| `chat.uncensored` | `ollama` | `chat.uncensored` | Local-only by design: unrestricted generation stays on the host and is reachable only through explicit role selection. |
| `classify.fast` | `ollama` | `classify.routing` | Small local classification role; the enum grammar carries most of the constraint. |
| `classify.fast` | `openrouter` | `classify.routing` | Hosted classification fallback when the local resource is unavailable. |
| `classify.fast` | `orcarouter` | `classify.routing` | Hosted OrcaRouter classification fallback when the local resource is unavailable and OpenRouter is not preferred. |
| `classify.fast` | `lpbs` | `classify.fast` | Subscription-backed metered fallback when local and operator BYOK routes are unavailable. (role `classify.fast`, provider `lpbs`) |
| `decide.accurate` | `ollaya` | `decide.local.accurate` | Winnow-E4B is the most accurate open decision model; skipped until it is pulled (8 GB, ~9 GB VRAM for GPU). |
| `decide.accurate` | `ollaya` | `decide.jev.edenai` | Hosted TypeSafe Jev through Eden AI. |
| `decide.accurate` | `ollaya` | `decide.jev.typesafe` | Hosted TypeSafe Jev direct. |
| `decide.accurate` | `ollaya` | `decide.clef.cloudflare` | Hosted Cloudflare Clef-flash. |
| `decide.accurate` | `ollaya` | `decide.local` | Local Laya keeps decisions available when no accurate candidate can serve. |
| `decide.accurate` | `ai-gateway` | `classify.fast` | Last resort for choice-only questions through the enum-constrained LLM path. |
| `decide.candidates` | `ollaya` | `decide.local` | Laya router, the current decide.fast model: the baseline every candidate is measured against. |
| `decide.candidates` | `ollaya` | `decide.local.accurate` | Winnow-E4B, the current decide.accurate model: the accuracy baseline. |
| `decide.candidates` | `ollaya` | `decide.local.typed` | Laya fine-tuned on the typed-decisions workflows (421M, 1024-token context). |
| `decide.candidates` | `ollaya` | `decide.local.coba-400m` | Coba 400M, Ollaya's own Laya-network model distilled from pplx-decider. |
| `decide.candidates` | `ollaya` | `decide.local.coba-1b` | Coba 1B, the most accurate Coba size. |
| `decide.candidates` | `ollaya` | `decide.local.largha` | Largha 150M, the smallest multilingual decision encoder under evaluation. |
| `decide.candidates` | `ollaya` | `decide.local.decima-clean` | Decima-base-clean, a late-interaction scorer trained without non-commercial data. |
| `decide.candidates` | `ollaya` | `decide.local.decima-agent` | Decima-agent, tuned for coding-agent decisions with an 8192-token state; rights unknown until the operator decides. |
| `decide.candidates` | `ollaya` | `decide.local.decima-small` | Decima-small, the fastest CPU model under evaluation; rights unknown until the operator decides. |
| `decide.candidates` | `ollaya` | `decide.local.nli-modernbert` | ModernBERT-large zero-shot NLI baseline; rights unknown until the operator decides. |
| `decide.candidates` | `ollaya` | `decide.local.nli-deberta` | DeBERTa-v3-large zero-shot NLI baseline; rights unknown until the operator decides. |
| `decide.candidates` | `ollaya` | `decide.local.pplx` | Perplexity pplx-decider-flash (Qwen3.5-4B), the strongest open decider under evaluation; GPU only. |
| `decide.candidates` | `ollaya` | `decide.local.kev` | Kev 4B (LoRA plus pointer head on Qwen3.5-4B-Base); GPU only. |
| `decide.candidates` | `ollaya` | `decide.local.credence` | Credence v1 E4B calibrated (Winnow refinement with validation-fitted temperature); GPU only, rights unknown until the operator decides. |
| `decide.candidates` | `ai-gateway` | `classify.fast` | The enum-constrained LLM path, as a no-probability baseline for choice questions. |
| `decide.fast` | `ollaya` | `decide.local` | Local Laya answers in milliseconds on the GPU and keeps the state on the host. |
| `decide.fast` | `ollaya` | `decide.jev.edenai` | Hosted TypeSafe Jev through Eden AI; skipped until vrooli/edenai is provisioned. |
| `decide.fast` | `ollaya` | `decide.jev.typesafe` | Hosted TypeSafe Jev direct; skipped until vrooli/typesafe is provisioned. |
| `decide.fast` | `ollaya` | `decide.clef.cloudflare` | Hosted Cloudflare Clef-flash; skipped until vrooli/cloudflare-workers-ai is provisioned. |
| `decide.fast` | `ai-gateway` | `classify.fast` | Last resort for choice-only questions: the enum-constrained LLM path, which returns labels without calibrated probabilities. |
| `decide.short` | `ollaya` | `decide.local.coba-400m` | Measured bake-off 2026-11-20 on operator-mood gold: best of the cleared-rights models up to 1.9 GiB on all three mood labels at the same warm latency as Laya (about 0.24 s for three questions). Clears no gate by itself. |
| `decide.short` | `ollaya` | `decide.local` | Local Laya answers when Coba 400M cannot be admitted; measured below it on every mood label. |
| `decide.short.accurate` | `ollaya` | `decide.local.coba-1b` | Measured bake-off 2026-11-20: the only measured cleared-rights model larger than Coba 400M (3.5 GiB, about 6 s warm for three questions); its gain over the 400M is within a few gold rows. |
| `decide.short.accurate` | `ollaya` | `decide.local` | Local Laya keeps escalation available when Coba 1B cannot be admitted. |
| `detect.pii.default` | `pii-detector` | `pii.small` | The pii-detector resource owns the tiered PII model policy (pii.small default -> pii.default -> pii.large) and always runs the deterministic regex/checksum pass. |
| `device-control.plan` | `ollama` | `chat.default` | Use Ollama's existing local generation role with gateway schema validation; the dedicated resource extraction role is not yet available. |
| `device-control.plan` | `openrouter` | `extract.structured` | Hosted structured-output fallback when the local planning resource is unavailable. |
| `device-control.plan` | `orcarouter` | `extract.structured` | Hosted OrcaRouter structured planning fallback. |
| `device-control.plan` | `lpbs` | `extract.structured` | Subscription-backed structured planning fallback when local and operator BYOK routes are unavailable. |
| `device.geometry.byok` | `openrouter` | `extract.structured` | The caller explicitly selected its owner-managed OpenRouter credential. |
| `device.geometry.vrooli_credits` | `lpbs` | `extract.structured` | The caller explicitly selected LPBS metered inference and credit accounting. |
| `embedding.default` | `ollama` | `embedding.default` | The resource owns the embedding model and its declared vector dimension. |
| `extract.structured` | `ollama` | `chat.default` | Mid-size local generation role; the gateway validates every returned value locally. |
| `extract.structured` | `openrouter` | `extract.structured` | Hosted structured-output fallback for schemas the local resource cannot reliably produce. |
| `extract.structured` | `orcarouter` | `extract.structured` | Hosted OrcaRouter structured-output fallback. |
| `extract.structured` | `lpbs` | `extract.structured` | Subscription-backed metered structured-output fallback. |
| `judge.default` | `ollama` | `classify.routing` | Local judge keeps rubric decisions available without a network dependency. |
| `judge.default` | `openrouter` | `extract.structured` | Hosted structured-output fallback when the local judge is unavailable. |
| `judge.default` | `orcarouter` | `extract.structured` | Hosted OrcaRouter structured-output judge fallback. |
| `judge.default` | `lpbs` | `judge.default` | Subscription-backed metered judge fallback. |
| `locate.visual` | `ollama` | `vision.default` | Local-first grounding keeps captured frames on the device and uses the measured vision model ladder. |
| `locate.visual` | `openrouter` | `vision.default` | Hosted multimodal grounding fallback for hosts without an available local vision rung. |
| `locate.visual` | `orcarouter` | `vision.default` | Hosted OrcaRouter visual grounding fallback. |
| `locate.visual` | `lpbs` | `vision.default` | Subscription-backed metered visual grounding fallback. |
| `research.unrestricted` | `abliteration` | `research.unrestricted.default` | Abliteration.ai is an explicit research/red-team provider, never a fleet default. Selecting this role is the authorization signal; the credential still gates the call. |
| `research.unrestricted.flash` | `abliteration` | `research.unrestricted.flash` | Explicit research/red-team route. Never selected unless the caller names this role. (role `research.unrestricted.flash`, provider `abliteration`) |
| `research.unrestricted.large` | `abliteration` | `research.unrestricted.large` | Explicit research/red-team route. Never selected unless the caller names this role. (role `research.unrestricted.large`, provider `abliteration`) |
| `summarize.audio_tools` | `ollama` | `summarize.audio_tools` | Audio Tools owns this compact summary role; generic summarize.faithful remains governed by summarize.default. |
| `summarize.audio_tools.quality` | `ollama` | `summarize.audio_tools.quality` | Audio Tools quality alternative. The compact 1.7B default retained 14/22 facts on one 300-word report; preserve this option for fact-dense input. |
| `summarize.faithful` | `ollama` | `summarize.default` | Resource-owned summarization policy; concrete model and fallback remain governed by resource policy. |
| `ux.visual-critique` | `ollama` | `vision.default` | Keep UX screenshots on the local, resource-governed vision model; preserve the caller's finding schema. |
| `write.default` | `ollama` | `write.default` | Local prose keeps drafts on the host and is the BYOK path a self-hoster gets with no credential. |
| `write.default` | `openrouter` | `write.default` | Hosted fallback where prose quality is the binding constraint; the model stays a resource policy decision. |
| `write.default` | `orcarouter` | `write.default` | Hosted OrcaRouter prose fallback. |
| `write.default` | `lpbs` | `write.default` | Subscription-backed metered fallback when local and operator BYOK routes are unavailable. (role `write.default`, provider `lpbs`) |
| `write.diverse` | `openrouter` | `write.diverse` | A k-candidate response is neither short nor cheap, and diversity gain scales with model capability; this is the rung the technique is designed for. |
| `write.diverse` | `orcarouter` | `write.default` | Hosted OrcaRouter diversity fallback at a different gateway. |
| `write.diverse` | `lpbs` | `write.diverse` | Subscription-backed metered fallback, ordered ahead of the local rung because capability matters more than locality for this role specifically. |
| `write.diverse` | `ollama` | `write.default` | Local last resort so a host with no credential still gets candidates rather than a failure. Expect measurably lower diversity here; the role's own description says so. |

## Semantic Escalation Candidates (Caller Opt-In)

These candidates are considered only after the ordinary route and schema-repair stages fail and the caller has opted in. They are intentionally separate from ordinary availability fallbacks.

| Role | Provider | Resource role | Reason |
| --- | --- | --- | --- |

## Qualification Candidates (Not Dispatchable)

Qualification candidates are staged for evidence collection and are never selected by runtime routing. Move one into the dispatchable candidate list only after its qualification evidence is accepted.

| Role | Provider | Resource role | Status | Evidence | Reason |
| --- | --- | --- | --- | --- | --- |
| `extract.structured` | `ollama` | `chat.small` | pending |  | LM4 compact extraction candidate; not dispatched until a real-caller comparison qualifies it. |
| `summarize.faithful` | `ollama` | `chat.small` | pending |  | LM4 compact faithful-summary candidate; not dispatched until a real caller validates factual fidelity. |
