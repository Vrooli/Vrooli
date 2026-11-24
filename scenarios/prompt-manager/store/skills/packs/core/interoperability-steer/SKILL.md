---
name: "interoperability-steer"
description: "Ensure scenarios communicate reliably through type-safe, proto-first contracts between UI↔API and API↔API boundaries"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","architecture"]
  tags: ["proto","types","validation","contracts"]
  status: "active"
  revision: 46
  createdAt: "2026-01-30T15:30:00-05:00"
  updatedAt: "2026-02-05T01:01:21Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "vrooli"]
  origin:
    kind: "authored"
---
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

Weak boundaries fail at runtime: silent data loss (fields renamed without synchronized consumers), type drift and unsafe `as SomeType` casts, schemas that exist but are not enforced, hardcoded or stale URLs after restarts, status drift (`complete` vs `completed`, `status` vs `run.status`), undeclared scenario dependencies, no fail-fast or degrade policy, and UI casing mismatches or hand-written TypeScript interfaces that drift from the protos. Proto-first contracts are necessary but not sufficient: interoperability needs schema, serialization, UI consumption, discovery, lifecycle and runtime behavior together.

### 1. Scope Boundaries

In scope: proto schema structure, naming and validation; generation and breakage detection; type-safe consumption through generated Connect-RPC clients (and generated descriptor parsing for REST exceptions); UI to API contract safety; runtime boundary validation (protovalidate); inter-scenario URL resolution; dependency declaration parity in `.vrooli/service.json`; envelope and status normalization; retry, re-resolve and degrade policy; the audit and completion gates. Out of scope: API product design (`api-steer`), domain logic, transport debates for proto-owned payloads (Connect-RPC is the baseline), language build tooling.

---

### 2. Interop Contract Stack

Reliability needs every layer. L0 schema (`packages/proto/schemas` plus protovalidate); L1 transport and serialization (Connect-RPC for proto payloads, proto JSON for REST); L2 envelope and status semantics (`run.status` path, terminal values, error and result fields); L3 discovery and addressing (`api-core/discovery`, no hardcoded ports); L4 lifecycle dependency contract (`dependencies.scenarios` parity); L5 runtime recovery (retry, backoff, re-resolution, degrade or fail fast, and for UI error, loading and retry states). L0 to L2 apply to every boundary, L3 and L4 to inter-scenario boundaries, L5 to both. "Using protos" only addresses L0 and L1; most production failures happen at L2 to L5.

**Wire format.** Inter-scenario API to API, UI to API and CLI to API for proto-typed payloads all use **Connect-RPC**. REST is allowed only for four reasons: `multipart_upload` (file bytes as multipart, proto describes metadata), `webhook_receiver` (third-party-dictated shape), `third_party_shape` (OAuth callbacks, OpenAPI passthrough) and `ops_probe` (`GET /health` for lifecycle, load balancers and curl). A non-proto internal endpoint should not exist: add a proto. The reasons are the `RESTReason*` constants in `templates/scenarios/react-vite/api/internal/module/module.go`, enforced by `validateTransport` in `templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`; a fifth reason is an architectural decision made by editing the template.

---

### 3. Proto contracts

`packages/proto/schemas/` is the source of truth; `make generate` writes `packages/proto/gen/` (committed); scenarios consume. Never edit `gen/`, never keep parallel hand-written types, and flow changes downward.

**Schema design.** Higher layers import lower, never the reverse: 0 primitives, 1 domain concepts, 2 definitions and templates, 3 history and audit, 4 runtime state, 5 API surface. Use enums for finite state (first value `*_UNSPECIFIED = 0`, document transitions, no string status fields); `optional` when unset must differ from the default; typed messages over `Struct` or maps when structure is known; protovalidate constraints for boundary-enforceable rules.

**Generation.** After any schema change run, in `packages/proto`, `make generate`, `make lint`, `make breaking` and `make verify-committed-gen`. Prefer additive changes, deprecate before removal, never reuse or renumber fields or enum values.

### 4. Type-safe consumption

Never bypass generated types with `as SomeType` or `any`, in UI or in backend calls to other scenarios. Connect clients handle JSON casing; the manual proto JSON rules apply only to REST exceptions, external APIs and stored fixtures: TypeScript `fromJson(MessageSchema, payload, { ignoreUnknownFields: true })` and `toJsonString(MessageSchema, message, { useProtoFieldName: true })`; Go `protojson.MarshalOptions{UseProtoNames: true}`, and `UnmarshalOptions{DiscardUnknown: false}` for request ingress (`true` only for forward-compatible response reads). `fromJson` accepts both `created_at` and `createdAt`, so add no manual casing transforms. Default every call to a generated Connect client; branch to REST only for a `RESTReason*` (multipart carries file bytes with a proto JSON metadata part; webhooks accept the third party's JSON and translate to proto domain events at once; `ops_probe` is a plain GET). UI props use generated types, never hand-written interfaces. Do not collapse optional numerics with truthy fallbacks when `0` or `""` is valid.

**Validation.** Validate (protovalidate) at API ingress, inter-scenario ingress and egress, and UI submit; same-scenario internals are context-dependent.

---

### 5. Inter-scenario structure

One canonical location per concern: [A] `.vrooli/service.json` `dependencies.scenarios`; [B] `api/integrations/` outbound clients (generated Connect clients behind per-dependency adapters); [C] `api/integrations/contracts/` envelope and status normalization (centralized extractors and constants); [D] `api/integrations/discovery.go` using `discovery.ResolveScenarioURLDefault`; [E] `api/integrations/policy.go` retry and degrade policy with explicit required or optional behavior; [F] `api/integrations/*_test.go` envelope, status and restart tests; [G] `ui/src/api/` UI clients (generated Connect clients and REST-exception helpers; `ui/src/lib/` stays for pure utilities).

**Discovery.** Resolve through `package:api-core/discovery`; no `localhost:<port>` for scenario to scenario production calls; no one-time URL capture in long-lived clients; re-resolve on connection, refused and transport failures with bounded attempts; explicit timeout on every outbound call; deterministic, documented manual fallback.

**Dependency parity.** Code that calls scenario `X` declares `dependencies.scenarios.X`. Required means fail fast with an actionable error; optional (`required: false`) means degrade with explicit logs and status. Code and manifest disagreement is an interop defect.

**Envelope and status.** Parse the actual envelope (`run.status` vs top-level `status`), keep a centralized per-dependency map of terminal, pending and failure statuses, normalize variants (`complete`, `completed`, enum strings) to canonical local states, keep parsers and constants out of handlers and UI, and test envelope path extraction, terminal classification and error field extraction.

**Recovery policy.** Per adapter define criticality, timeout budget, bounded retry and backoff, re-resolution trigger and terminal fallback. Transport or discovery failure re-resolves and retries within bounds; 5xx retries within bounds; 4xx, schema and validation failures surface the contract mismatch with no blind retry; on exhaustion required fails fast with an operator signal and optional degrades with telemetry.

---

### 6. Anti-patterns and audit

Detect and eliminate: unsafe assertions near external payloads, hand-written types duplicating generated ones (including UI interfaces), missing proto JSON options, stringly status comparisons scattered across files, hardcoded scenario ports or URLs, startup-only client wiring, undeclared runtime dependencies, silent degrade with no signal, UI fetch or axios for proto-owned operations, REST-exception helpers in `ui/src/api/` that skip `fromJson` or send raw metadata objects.

Audit with `rg` and `jq` before changes: proto usage and casts (`fromJson|toJsonString|protojson|protovalidate`, `as [A-Z]\w+|as any|: any`); UI hitting proto-owned REST paths (`fetch(...api/v1`, `axios.*api/v1`); CLI raw `APIClient.*` outside REST exceptions; literal `Path: "/api/v1/` in EndpointDescriptors; REST EndpointDescriptors missing `RESTException`; discovery and hardcoded ports (`ResolveScenarioURL`, `localhost:[0-9]+` in `api`); status parsing hotspots; startup-only client capture (`New.*Client(`, `baseURL =`); `jq '.dependencies.scenarios // {}' .vrooli/service.json`; UI hand-written `interface` declarations; and `cd packages/proto && make verify-committed-gen`.

Record durable findings in the scenario's `docs/internal/SEAMS.md`, `ARCHITECTURE.md` or `PROBLEMS.md` (whichever owns the boundary or debt; no standalone `INTEROP_AUDIT.md`): dependency inventory (declared, used, required or optional), contract, UI to API and discovery or lifecycle findings with `file:line`, priority fixes, and the gates below. At session start read `packages/proto/README.md`, `packages/proto/STYLE_GUIDE.md` and those documents; at session end verify old claims against code and record remaining risks and un-audited areas. Track visited files with `visited-tracker-tools` (location `scenarios/{{TARGET}}`, tag `interoperability`).

### 7. Proper and complete

No critical or high defects remain; UI consumes responses through generated types with proper serialization (no hand-written duplicates or unsafe casts); outbound dependencies are declared and match runtime behavior; no hardcoded scenario ports or URLs in production paths; envelope and status normalization is centralized and tested; re-resolution and bounded retry exist; required dependencies fail fast with diagnostics and optional ones degrade observably; the audit document reflects verified state.

**Troubleshooting.** Connection refused on an old port: stale captured URL, re-resolve per request. Invalid JSON body: payload not proto-compatible, use generated types. Async run never terminal, or success marked cancelled: status vocabulary drift or envelope path mismatch, normalize in the adapter. Works only if the dependency starts first: startup-only wiring, add lazy init and retry. Feature silently no-ops: optional degrade not surfaced, add telemetry and visible state. UI stale or blank after an API change: hand-written interface, manual casing or bypassed Connect client; use generated types and centralize REST exceptions in `ui/src/api/`.

**Maturity ladder** (score UI to API and API to API separately): 0 untyped JSON; 1 proto schemas exist; 2 proto JSON at every boundary, no manual casing; 3 Connect-RPC end to end, REST only with a valid `RESTReason*`; 4 protovalidate at ingress, discovery-based URLs, bounded retry and re-resolution; 5 a codegen drift gate (`validateTransport` or equivalent) fails the build on hand-authored REST paths, parity tests assert UI and CLI to RPC mapping, and envelope normalization is centralized and tested.

---

### **8. Output Expectations**

You may update proto schemas and generated-type consumers, integration adapters, discovery and policy layers, `.vrooli/service.json` dependency declarations, UI API consumption layers, and interop tests and audit docs. You must keep `{{TARGET}}` functional, preserve contract safety at boundaries, use discovery for scenario URLs, keep dependency declarations in parity, centralize envelope and status normalization with tests for those seams and for recovery, use generated proto types and proto JSON at API boundaries, and document migration paths for breaking changes. You must not hardcode scenario ports or URLs, scatter inter-scenario parsing across handlers, bypass generated types with unsafe casts, leave undeclared dependencies, define UI interfaces duplicating generated types, or make superficial refactors.
