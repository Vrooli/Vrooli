# Fact Families

| Family | Meaning |
|---|---|
| `surfaces` | Implemented in Phase 7. Scenario API/CLI/UI/sidecar metadata and generic target status, with evidence from directories and manifests. |
| `parse_units` | Implemented in Phase 7. Go modules, TypeScript projects, explicit `tsconfig.json` configs, unsupported Node packages, and unknown units. |
| `imports` | Implemented in Phase 8. Import specs/bindings from provider graphs, preserving provider attributes such as Go import paths and TypeScript source modules. |
| `symbols` | Implemented in Phase 8. Declaration-like graph nodes normalized from provider output. |
| `references` | Implemented in Phase 8. Resolved symbol references and Go type-usage facts where graph providers emit them. |
| `calls` | Implemented in Phase 8. Call expressions and JSX usage facts, including provider metadata such as callee, argument summaries, props summaries, and enclosing declarations when present. |
| `proto_adoption` | Implemented in Phase 10. Generated proto artifact adoption evidence synthesized from generic imports, scoped by API/CLI/UI surface and classified as scenario-owned or shared generated proto usage. |
| `endpoint_proofs` | Implemented in Phase 10. Static REST-exception implementation evidence synthesized from `.vrooli/endpoints.json`, route metadata when graph facts provide it, and recognized typed helper usage such as `WriteProto`/`WriteError`. |
| `cli_proofs` | Planned CLI implementation evidence. |
| `ui_widget_proofs` | Planned UI widget evidence. |
| `all` | All supported families for the target. |

Generic facts include provider-specific metadata in `attributes`. Code Facts treats those attributes as evidence payload, not policy conclusions. Proof families interpret a small, documented subset of those attributes and emit `unknown` instead of `proven` when the provider graph lacks enough detail.
