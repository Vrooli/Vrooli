# Fact Families

| Family | Meaning |
|---|---|
| `surfaces` | Implemented in Phase 7. Scenario API/CLI/UI/sidecar metadata and generic target status, with evidence from directories and manifests. |
| `parse_units` | Implemented in Phase 7. Go modules, TypeScript projects, explicit `tsconfig.json` configs, unsupported Node packages, and unknown units. |

Each parse unit also carries a neutral `ToolchainObservation` envelope. It
records observed manifest and lockfile paths, build-system markers,
package-manager/toolchain identity, and runner indicators such as declared
scripts or dependencies. These are observations with provenance; Code Facts
does not decide whether a framework is canonical or whether a declared adapter
is policy-compliant. Unit Health reconciles them with declared intent and host
capabilities.
| `imports` | Implemented in Phase 8. Import specs/bindings from provider graphs, preserving provider attributes such as Go import paths and TypeScript source modules. |
| `symbols` | Implemented in Phase 8. Declaration-like graph nodes normalized from provider output. |
| `references` | Implemented in Phase 8. Resolved symbol references and Go type-usage facts where graph providers emit them. |
| `calls` | Implemented in Phase 8. Call expressions and JSX usage facts, including provider metadata such as callee, argument summaries, props summaries, and enclosing declarations when present. |
| `proto_adoption` | Implemented in Phase 10. Generated proto artifact adoption evidence synthesized from generic imports, scoped by API/CLI/UI surface and classified as scenario-owned or shared generated proto usage. |
| `endpoint_proofs` | Implemented. Static REST-exception implementation evidence synthesized by comparing `.vrooli/endpoints.json` declarations with normalized endpoint implementation facts from Code Facts framework adapters. |
| `cli_proofs` | Planned CLI implementation evidence. |
| `ui_widget_proofs` | Planned UI widget evidence. |
| `file_domain` | Architecture Cartographer's per-file domain verdicts. Code Facts delegates verdict production to cartographer, normalizes returned verdicts into this family, and returns typed unsupported evidence when cartographer is not configured or reachable. |
| `all` | All supported families for the target. |

Generic facts include provider-specific metadata in `attributes`. Code Facts treats those attributes as evidence payload, not policy conclusions. Proof families interpret a small, documented subset of those attributes and emit `unknown` instead of `proven` when the provider graph lacks enough detail.

`file_domain` facts use the existing `GenericFact` envelope: `subject` is the
repo-relative file path, `kind` is `file_domain`, and attributes carry
`path`, `top_domain`, `top_value`, `tier`, `runner_up_domain`,
`runner_up_value`, `tied`, and `authority_confidence`. The producing policy
belongs to Architecture Cartographer so Code Facts remains a broker/cache
surface rather than a second domain classifier.

## Endpoint Proof Support Matrix

| Language | Framework | Route Proof | Response Proto Proof | Error Proto Proof | Status |
|---|---|---|---|---|---|
| Go | `net/http` / gorilla mux-style static registrations | Supported from route registration facts with literal path and method attributes. | Supported through generated Go proto type usage and recognized `httpx.WriteProto`-style helpers. | Supported through generated ErrorEnvelope usage and recognized `httpx.WriteError`-style helpers. | Active |
| TypeScript | Express | Supported for literal `app.METHOD(...)` and `router.METHOD(...)` registrations emitted by `typescript-code-graph`. | Supported for static generated TypeScript proto imports plus recognized response calls or typed arguments. Import-only evidence remains `unknown`. | Partial and conservative; generated error payload usage must be visible in handler-local facts. | Active, partial |
| TypeScript | Fastify, Hono, and other frameworks | Not evaluated until an adapter exists. | Unsupported. | Unsupported. | Planned |

Endpoint-level status is an aggregate over route and required proto payload
roles. `contradicted` outranks all other statuses, `missing` outranks
`unknown` and `unsupported`, and `unknown` prevents a proven endpoint unless it
belongs to a role that is explicitly non-proto or non-required.
Route-registration facts for frameworks without an endpoint adapter emit
`code-facts.endpoint_proof.framework_unsupported` warnings with `unsupported`
status; they are not treated as proof.
