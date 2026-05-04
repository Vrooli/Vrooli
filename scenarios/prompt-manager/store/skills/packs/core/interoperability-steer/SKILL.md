## Steer focus: Interoperability Steer

Prioritize **hardening end-to-end interoperability** — both UI↔API within `scenarios/{{TARGET}}/` and API↔API across scenario boundaries — across contracts, serialization, discovery, dependency declarations, and runtime recovery.

Your goal is to ensure the target scenario communicates reliably — its own UI with its API, and its API with other scenarios — under normal operation and failure conditions (restart, port drift, partial availability), while preserving proto-first type safety.

Do **not** break functionality, regress tests, or introduce unrelated features. All changes must maintain or improve the scenario's interoperability posture.

Required reading:
- `prompt-manager skill read visited-tracker-tools`

Optional reading:
- `prompt-manager skill read api-steer`
- `prompt-manager skill read cli-steer`

---

### 0. Why This Skill Exists

Inter-scenario communication is the connective tissue of Vrooli. When `{{TARGET}}` has weak boundaries, the system becomes fragile in ways that often appear only at runtime:

- **Silent data loss:** fields renamed/removed without synchronized consumers.
- **Type drift:** mixed casing or payload shapes silently dropped by parsers.
- **Unsafe casts:** `as SomeType` hides invalid payloads until production failures.
- **Validation gaps:** schemas exist but are not enforced at boundaries.
- **Addressing drift:** hardcoded or stale URLs/ports after scenario restarts.
- **Status drift:** `complete` vs `completed`, top-level `status` vs `run.status`.
- **Lifecycle drift:** code depends on another scenario that service manifest does not declare.
- **Operational ambiguity:** no clear fail-fast vs graceful-degrade policy.
- **UI casing mismatch:** UI hand-reads `camelCase` while API sends proto-name JSON (`snake_case`), causing fields to disappear outside the generated parser.
- **UI type divergence:** hand-written TypeScript interfaces drift from proto definitions, hiding missing/renamed fields until runtime.

Proto-first contracts are necessary but not sufficient. This skill ensures interoperability is robust across **schema + serialization + UI consumption + discovery + lifecycle + runtime behavior**.

---

### 1. Scope Boundaries

**In scope**
- Proto schema structure, naming, validation annotations.
- Proto generation and breakage detection workflow.
- Type-safe contract consumption through generated Connect-RPC clients and,
  for REST exceptions, generated descriptor parsing (`fromJson`,
  `toJsonString`, `protojson`).
- UI↔API contract safety: generated type usage, casing options, and validation in frontend code.
- Runtime boundary validation (protovalidate).
- Inter-scenario URL resolution and addressing patterns.
- Dependency declaration parity in `.vrooli/service.json`.
- Envelope/status normalization rules for cross-scenario calls.
- Runtime resilience rules (retry, re-resolve, fail-fast/degrade).
- Interoperability audit workflow and completion gates.

**Out of scope**
- API product design decisions (resource model, endpoint UX) -> `api-steer`.
- Domain/business logic implementation.
- Transport preference debates for proto-owned payloads; Connect-RPC is the
  baseline substrate.
- Language-specific build tooling internals beyond interoperability impact.

---

### 2. Interop Contract Stack (Convergence Pattern)

Treat interoperability as a layered contract. Reliability requires all layers.

```
INTEROP CONTRACT STACK
┌─────────────────────────────────────────────────────────┐
│ L5 Runtime Recovery                                    │
│ Retry/backoff, re-resolution, degrade/fail-fast policy │
├─────────────────────────────────────────────────────────┤
│ L4 Lifecycle Dependency Contract                       │
│ .vrooli/service.json dependencies.scenarios parity     │
├─────────────────────────────────────────────────────────┤
│ L3 Discovery & Addressing                              │
│ api-core/discovery, no hardcoded scenario ports/URLs   │
├─────────────────────────────────────────────────────────┤
│ L2 Envelope & Status Semantics                         │
│ run.status path, terminal values, error/result fields  │
├─────────────────────────────────────────────────────────┤
│ L1 Transport + Serialization Contract                  │
│ Connect-RPC for proto payloads; proto JSON for REST    │
├─────────────────────────────────────────────────────────┤
│ L0 Schema Contract                                     │
│ packages/proto/schemas + protovalidate constraints     │
└─────────────────────────────────────────────────────────┘
```

Applicability:
- L0–L2 apply to ALL boundaries (UI↔API and API↔API).
- L3–L4 apply to inter-scenario boundaries.
- L5 applies to both: inter-scenario (retry/re-resolve) and UI↔API (error states, loading, retry UX).

**Steer:** "Using protos" only addresses L0-L1. Most production interop failures — whether UI↔API or API↔API — happen at L2-L5.

Connect-RPC is now the substrate for proto-owned wire calls. Generated
transport code moves method paths, request/response typing, and error codes
out of hand-written UI/API/CLI glue.

**Wire format decision tree:**
- Inter-scenario API↔API: **Connect-RPC**.
- UI↔API for proto-typed payloads: **Connect-RPC**.
- CLI↔API for proto-typed payloads: **Connect-RPC**.
- File uploads / opaque binary edges: **REST with multipart**; proto
  describes metadata only.
- Webhook receivers / third-party API consumption: **REST/JSON** with the
  shape the third party dictates.
- Non-proto-typed internal endpoints: there should be none. Add a proto.

---

### 3. Proto Contract Hierarchy

```
packages/proto/schemas/      <- source of truth
          |
          v make generate
packages/proto/gen/          <- generated artifacts (committed)
          |
          v import
scenarios/{{TARGET}}/        <- consumers (UI/API/CLI)
```

Rules:
- Never edit `gen/` directly.
- Never maintain parallel hand-written types when generated types exist.
- Flow changes downward from schema to generated code to consumers.

---

### 4. Proto Schema Design Principles

#### 4.1 Layer Architecture

Higher conceptual layers may import from lower layers, never the reverse.

| Layer | Purpose | Examples |
|---|---|---|
| 0 | primitives | shared enums/basic types |
| 1 | domain concepts | business models |
| 2 | definitions/templates | workflows/action schemas |
| 3 | historical/audit | logs, recordings |
| 4 | runtime state | active execution/session state |
| 5 | API surface | service contracts/top-level aggregates |

Decision flow:
```
Primitive? -> L0
Domain model independent of execution? -> L1
Template/definition? -> L2
History/audit? -> L3
Live runtime state? -> L4
Service/API aggregate? -> L5
```

#### 4.2 Prefer Enums for Finite State

- First enum value: `*_UNSPECIFIED = 0`.
- Avoid string status fields for finite state machines.
- Document valid transitions for lifecycle enums.

#### 4.3 Use Optional for Semantic Presence

Use `optional` when "unset" must be distinguished from default value.

#### 4.4 Prefer Typed Metadata

Prefer typed message fields to raw `Struct`/map escape hatches when structure is known.

#### 4.5 Protovalidate in Schemas

Add constraints in proto definitions for boundary-enforceable validation.

---

### 5. Proto Generation Workflow

After any schema change:
```bash
cd packages/proto
make generate
make lint
make breaking
make check
```

Breaking-change guidance:
- Prefer additive changes.
- Deprecate before removal.
- Never reuse field numbers.
- Never renumber enums/fields.

---

### 6. Type-Safe Consumption Patterns

#### 6.1 Golden Rule

Do not bypass generated types using `as SomeType`/`any` where schemas are available. This applies equally to UI components consuming API responses and to backend code calling other scenarios.

#### 6.2 JSON Casing Compatibility

Connect clients handle proto JSON casing for Connect-RPC calls. The manual
proto JSON rules below apply only to REST exceptions, external APIs, stored
fixtures, and other places where generated Connect clients are not the
transport:
- TypeScript read: `fromJson(MessageSchema, payload, { ignoreUnknownFields: true })`
- TypeScript write: `toJsonString(MessageSchema, message, { useProtoFieldName: true })`
- Go write: `protojson.MarshalOptions{UseProtoNames: true}`
- Go ingress: `protojson.UnmarshalOptions{DiscardUnknown: false}` when rejecting unknown request fields; use `DiscardUnknown: true` only for forward-compatible response reads.

TypeScript `fromJson` accepts both proto field names (`created_at`) and JSON/lowerCamel names (`createdAt`). Do not add manual casing transforms between UI and API.

Multipart/form-data and raw file uploads are transport exceptions. Send file bytes as multipart parts or binary streams, and keep structured metadata in a proto JSON part parsed through the generated descriptor.

#### 6.3 Type Consumption Decision Tree

```
Incoming API/WS payload? -> schema parse (fromJson/protojson)
Outgoing payload? -> schema serialization (toJsonString/protojson)
Proto-owned UI/API/CLI call? -> generated Connect-RPC client
Optional field presence check needed? -> explicit presence logic
UI receiving REST exception response? -> schema parse in src/api with fromJson
UI sending REST exception metadata? -> schema serialization in src/api with toJsonString + useProtoFieldName
UI props/types? -> import generated types directly, never hand-written interfaces
```

#### 6.4 Optional Field Handling

Do not collapse optional numeric fields with truthy fallbacks when `0` is valid.

---

### 7. Runtime Validation at Boundaries

Validate at boundaries, trust internals.

| Boundary | Validation |
|---|---|
| API ingress | required |
| Inter-scenario ingress/egress | required |
| UI submit | required |
| Internal service-to-service (same scenario) | context-dependent |

Use protovalidate where external or cross-scenario payloads enter.

---

### 8. Inter-Scenario Adapter Convergence Pattern

Use a scannable structure to reduce drift.

| Slot | Canonical Path | Responsibility | Marker |
|---|---|---|---|
| [A] | `scenarios/{{TARGET}}/.vrooli/service.json` | scenario dependency declarations | `dependencies.scenarios` |
| [B] | `scenarios/{{TARGET}}/api/integrations/` | outbound scenario clients | generated Connect clients behind per-dependency adapters |
| [C] | `scenarios/{{TARGET}}/api/integrations/contracts/` | envelope/status normalization | centralized extractors/constants |
| [D] | `scenarios/{{TARGET}}/api/integrations/discovery.go` (or equivalent) | URL resolution | `discovery.ResolveScenarioURLDefault` |
| [E] | `scenarios/{{TARGET}}/api/integrations/policy.go` | retry/degrade policy | explicit required/optional behavior |
| [F] | `scenarios/{{TARGET}}/api/integrations/*_test.go` | contract and recovery tests | envelope/status/restart tests |
| [G] | `scenarios/{{TARGET}}/ui/src/api/` | UI↔API contract clients | generated Connect clients, REST-exception helpers |

Flexibility:
- Keep one canonical location per concern. New React scenarios use `ui/src/api/` for UI API clients; `ui/src/lib/` remains for pure utilities.

---

### 9. Discovery and Addressing Rules

1. Resolve scenario URLs via `api-core/discovery`.
2. Do not hardcode `localhost:<port>` for scenario-to-scenario calls in production paths.
3. Do not rely on one-time startup URL capture for long-lived clients.
4. Re-resolve URL on connection/refused/transport failures (bounded attempts).
5. Every outbound call must have explicit timeout.
6. Manual fallback logic must remain deterministic and documented.

---

### 10. Dependency Declaration Parity Rules

1. If code calls scenario `X`, declare `X` in:
   `scenarios/{{TARGET}}/.vrooli/service.json` -> `dependencies.scenarios.X`.
2. Required dependency:
`required: true`
fail fast with actionable error when unavailable.
3. Optional dependency:
`required: false`
graceful degradation with explicit logs/status.
4. Code and manifest disagreement is an interop defect.

---

### 11. Envelope and Status Normalization Rules

1. Parse actual response envelope shape (for example `run.status` vs top-level `status`).
2. Maintain centralized mapping of terminal/pending/failure statuses per dependency.
3. Normalize known variants (`complete`, `completed`, enum strings) to canonical local states.
4. Keep parser/status constants out of handlers/UI; place them in adapter contract layer.
5. Add regression tests for:
- envelope path extraction.
- terminal-status classification.
- error field extraction.

---

### 12. Runtime Recovery Policy

For each dependency adapter, define:

- dependency criticality (`required`/`optional`)
- timeout budget
- retry/backoff policy (bounded)
- re-resolution trigger
- terminal fallback behavior

Decision flow:
```
Call fails
├─ transport/discovery failure -> re-resolve + bounded retry
├─ 5xx/transient -> bounded retry
├─ 4xx/schema/validation -> no blind retry; surface contract mismatch
└─ retries exhausted
   ├─ required -> fail fast, explicit operator signal
   └─ optional -> degrade gracefully, explicit telemetry/logs
```

---

### 13. Anti-Patterns to Detect and Eliminate

- Unsafe type assertions near external payloads.
- Hand-written duplicate contract types where generated types exist.
- Missing proto JSON serialization options causing casing drift.
- Stringly status comparisons scattered across files.
- Hardcoded scenario ports/URLs in integration code.
- Startup-only client wiring with no reconnect strategy.
- Undeclared runtime scenario dependencies.
- Silent degrade behavior with no observable signal.
- Hand-written TypeScript interfaces duplicating proto message shapes in UI code.
- UI fetch/axios calls for proto-owned operations instead of generated
  Connect clients.
- UI REST-exception helpers parsing responses without `fromJson` in
  `ui/src/api/`.
- UI REST-exception submissions sending raw metadata objects instead of
  proto-serialized payloads.

---

### 14. Interoperability Audit

Before changes, assess `{{TARGET}}` posture.

#### 14.1 Audit Commands

```bash
# Proto usage and unsafe casts
rg "from '@vrooli/proto-types|@vrooli/proto-types|fromJson|toJsonString|protojson|protovalidate" scenarios/{{TARGET}}/
rg "as [A-Z][a-zA-Z]+|as any|: any" scenarios/{{TARGET}}/ --glob '!**/*_test.*'

# Discovery/addressing and hardcoded ports
rg "ResolveScenarioURLDefault|ResolveScenarioURL|scenario port|localhost:[0-9]+" scenarios/{{TARGET}}/api --glob '!**/*_test.go'

# Envelope/status parsing hotspots
rg "status|run\\.status|completed|complete|needs_review|failed|cancelled" scenarios/{{TARGET}}/api --glob '!**/*_test.go'

# Startup-only client capture risk
rg "New.*Client\\(|baseURL\\s*=|PromptManagerURL|AgentManagerURL" scenarios/{{TARGET}}/api --glob '!**/*_test.go'

# Dependency declarations
jq '.dependencies.scenarios // {}' scenarios/{{TARGET}}/.vrooli/service.json

# UI: hand-written interfaces that may duplicate proto types
rg "interface\s+[A-Z][a-zA-Z]+\s*\{" scenarios/{{TARGET}}/ui/src --type ts

# UI: unsafe casts near API consumption
rg "as [A-Z][a-zA-Z]+|as any|: any" scenarios/{{TARGET}}/ui/src --type ts --glob '!**/*.test.*'

# UI: fromJson/fetch usage outside the API boundary
rg "fromJson|fetch|axios|useQuery|useMutation" scenarios/{{TARGET}}/ui/src --type ts

# Proto generation sync
cd packages/proto && make check
```

#### 14.2 Red Flags Checklist

- [ ] hardcoded scenario `localhost:port` usage.
- [ ] no discovery usage for outbound scenario calls.
- [ ] URL/client captured once at startup and never refreshed.
- [ ] missing `dependencies.scenarios` declaration for actual outbound call target.
- [ ] envelope mismatch in parser logic.
- [ ] terminal status mismatch or drift.
- [ ] missing required/optional behavior definition.
- [ ] missing recovery tests for restart/re-resolution.
- [ ] unsafe casts bypassing schema validation.
- [ ] hand-written UI interfaces duplicating proto message shapes.
- [ ] UI calling proto-owned operations without generated Connect clients.
- [ ] UI parsing REST-exception responses without `fromJson` in `ui/src/api/`.
- [ ] UI sending REST-exception metadata without proto serialization.

#### 14.3 Findings Template

Write findings to:
`scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md`

```markdown
# {{TARGET}} Interoperability Audit

## Last Updated
YYYY-MM-DD

## Dependency Inventory
| Dependency | Declared | Used in Code | Required/Optional | Status |
|---|---|---|---|---|

## Contract Findings
1. [file:line] ...

## UI↔API Findings
1. [file:line] ...

## Discovery/Lifecycle Findings
1. [file:line] ...

## Priority Fixes
1. ...
2. ...

## Proper/Complete Gates
- [ ] Contract safety
- [ ] Discovery/addressing safety
- [ ] Dependency parity
- [ ] Envelope/status normalization
- [ ] UI↔API contract safety (generated types, casing, validation)
- [ ] Runtime recovery tests
```

---

### 15. Proper/Complete Criteria

A scenario's interop setup is considered proper/complete when:

1. No critical/high interoperability defects remain.
2. UI code consumes API responses via generated types with proper serialization options (no hand-written duplicates, no unsafe casts).
3. Outbound scenario dependencies are declared and aligned with runtime behavior.
4. No hardcoded scenario ports/URLs in production integration paths.
5. Envelope parsing and status normalization are centralized and tested.
6. URL re-resolution and bounded retry behavior are implemented.
7. Required dependencies fail fast with actionable diagnostics.
8. Optional dependencies degrade gracefully with explicit observability.
9. Interop audit document reflects verified current state.

---

### 16. Troubleshooting & Edge Cases

| Symptom | Likely Cause | First Check | Fix |
|---|---|---|---|
| connection refused on old port | stale captured URL | adapter init + URL lifecycle | re-resolve on failure/per-request |
| invalid JSON request body | payload not proto-compatible | compare payload with schema | use generated types/protojson-compatible structure |
| async run never reaches terminal | status vocabulary drift | terminal mapping constants | normalize status variants centrally |
| successful run marked cancelled/unknown | envelope parsing mismatch | parser path (`status` vs `run.status`) | parse canonical path in adapter |
| works only if dependency starts first | startup-only client wiring | startup logs + nil client path | lazy init/retry/recover strategy |
| feature silently no-ops | optional dependency degrade not surfaced | logs/health/status route | add explicit degrade telemetry and user-visible state |
| UI shows stale/missing fields after API change | hand-written UI interface not updated | search for duplicated interfaces in `ui/src` | replace with generated proto type imports |
| API returns data but UI renders blank/wrong values | manual casing mismatch, bypassed Connect client, or missing proto parse on a REST exception | check `ui/src/api/` for generated Connect clients or REST helper parsing | use generated Connect clients for proto-owned calls; centralize REST exceptions in `ui/src/api/` |

---

### 17. Memory Management with Visited Tracker

Use the `visited-tracker-tools` skill with:
- LOCATION: `scenarios/{{TARGET}}`
- TAG: `interoperability`

---

### 18. Documentation and Memory Loop

At session start, read:
- `packages/proto/README.md`
- `packages/proto/STYLE_GUIDE.md`
- `scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md` (if present)

At session end, update:
- `scenarios/{{TARGET}}/docs/internal/INTEROP_AUDIT.md`
- verify old claims against current code.
- record remaining risks and un-audited areas.

---

### 19. Output Expectations

You may update in `scenarios/{{TARGET}}/`:
- proto schemas and generated-type consumers.
- integration adapters/discovery/policy layers.
- `.vrooli/service.json` dependency declarations.
- UI fetch/API consumption layers to use generated types and proper serialization.
- interop-specific tests and audit docs.

You must:
- keep `{{TARGET}}` functional and non-regressed.
- preserve/add contract safety at boundaries.
- use discovery for scenario URL resolution.
- ensure dependency declaration parity.
- centralize envelope/status normalization.
- add tests for envelope/status/recovery seams.
- ensure UI code uses generated proto types and proto JSON at API boundaries.
- document migration paths for breaking changes.

You must NOT:
- hardcode scenario API ports/URLs.
- scatter inter-scenario parsing logic across handlers.
- bypass generated types with unsafe casts.
- leave undeclared runtime scenario dependencies.
- define hand-written UI interfaces that duplicate generated proto types.
- make superficial refactors without interoperability impact.

**Avoid superficial changes that rename/restructure code without materially improving interoperability reliability.**

Last updated: 2026-05-04 (Connect-RPC adoption)
