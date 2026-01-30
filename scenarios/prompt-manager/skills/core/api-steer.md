## Steer focus: API Steer

Design and maintain scenario APIs as **stable, intuitive, domain-aligned contracts**. The API is the scenario’s entrypoint for core logic and the primary integration surface for other scenarios and the scenario CLI—treat it as a product.

Optional reading:
- `prompt-manager skills read interoperability-steer`

---

### 0. Why This Skill Exists

In Vrooli, the API is not “just an interface”—it is:
- The **entrypoint** into the scenario’s core logic and operational guarantees.
- The primary mechanism for **inter-scenario communication**.
- The foundation for the scenario’s **CLI**, which should be a thin wrapper over the API.
- The most durable surface area: once clients depend on it, changing it becomes expensive.

This skill steers toward APIs that are:
- **Screaming-architecture organized** (domain boundaries are obvious by directory/module layout).
- **Typed and contract-driven** (proto-first for reliability and interoperability).
- **Consistent and predictable** (errors, naming, pagination, idempotency, auth).
- **Thin at the transport edge** (HTTP/gRPC handlers orchestrate; domain logic lives elsewhere).
- **Hard to misuse** (safe defaults; obvious “happy path”; guardrails at boundaries).

---

### 1. Scope Boundaries

**In scope**
- API surface design: endpoints, RPCs, request/response shapes, error models
- API organization: module boundaries, routers/controllers layout, keeping files small
- Consistency conventions: naming, pagination/filtering, idempotency, auth, observability hooks
- Compatibility strategy: versioning, additive change patterns, deprecation approach
- Shared API utilities: validation, auth middleware, pagination helpers, error translation, context

**Out of scope**
- Deep domain modeling (belongs in domain/service architecture skills)
- Transport choice wars (REST vs gRPC) beyond consistency and reliability
- Full “proto governance” details (use `interoperability-steer` for canonical rules)
- UI concerns (except where UI↔API contract reliability is impacted)

---

### 2. The Scenario Development Ladder (Where API Fits)

A useful ordering for development clarity:

```

PRD (what + targets)
↓
Requirements (how targets are met; constraints; failure modes)
↓
Documentation (onboarding + mental model)
↓
API (how other systems and agents use the scenario)
↓
Implementation (services, domain logic, infra)

```

**Steer:** If the API is unclear, the implementation will trend toward being unclear too.
A strong API surface forces clean boundaries and makes downstream development faster and safer.

---

### 3. Scream the Architecture: Organize APIs by Domain, Not by Mechanics

**Goal:** A new engineer (or agent) should be able to infer the scenario’s major capabilities by scanning the API directory tree.

Prefer:
- Grouping by **bounded context / capability / feature area**
- Endpoints that align to “what the scenario does,” not “what tables exist” or “what HTTP verbs we used”

Avoid:
- A single giant `routes.ts` / `handlers.ts`
- A “misc” grab bag
- Grouping solely by HTTP verbs (`get.ts`, `post.ts`) or by pure technical layers without domain shape

#### 3.1 Convergence Pattern: Endpoint Placement Decision Tree

```

New API operation needed
│
▼
Is it clearly part of an existing domain module?
├─ YES → add to that domain module (keep module cohesive)
└─ NO
│
▼
Is it a cross-cutting concern (auth, health, admin, observability)?
├─ YES → place in a dedicated cross-cutting module
└─ NO
│
▼
Does it represent a new bounded context / capability?
├─ YES → create a new domain module with its own routes + handlers
└─ NO  → refactor: your concept boundaries aren’t clear yet

```

#### 3.2 Convergence Pattern: Module Size and Responsibility Rules

**A module stays healthy when:**
- It has a single clear responsibility (one bounded context).
- Its public surface is small and readable.
- Handlers are short (orchestrate, don’t implement).
- Shared helpers are extracted *only when they’re truly shared*.

**Red flags**
- A router/handler file routinely exceeds “human scannability” (hundreds of lines).
- Similar validation/error mapping logic appears in multiple handlers.
- Endpoints in the same module feel unrelated (“we didn’t know where else to put it”).

---

### 4. API Shape: Resources, Commands, and Jobs (Make Usage Obvious)

Scenarios commonly need three kinds of operations. Design the API so each kind is unmistakable.

| Operation kind | What it means | Common patterns | Key property |
|---|---|---|---|
| **Query** | Read state without side effects | list/get endpoints | Safe + idempotent |
| **Command** | Change state / trigger side effects | create/update/execute endpoints | Explicit intent |
| **Job** | Long-running / async work | start + status + result endpoints | Observable progress |

#### 4.1 Resource vs Action: Naming Convergence Table

| If the client is… | Prefer | Example shape |
|---|---|---|
| Managing a domain entity | Resource endpoints | `/projects/{id}`, `/projects:list` |
| Triggering a meaningful operation | Command/action endpoint | `/projects/{id}:archive`, `/runs:execute` |
| Starting work that may outlive the request | Job endpoints | `/imports:start` → `/imports/{jobId}` |

**Steer:** Avoid “verb soup” routes that encode business logic in URLs without a stable conceptual model.
If you must expose actions, make them explicit and consistent.

---

### 5. Proto-First Contracts (Reliability + Interoperability)

**Principle:** Treat the proto schema as the canonical contract. Generated types should flow through:
- API↔API calls (inter-scenario)
- UI↔API calls
- CLI↔API calls

This increases reliability by eliminating “stringly typed” drift and enforcing consistent shapes.

**Guidance (high-level):**
- Requests/responses are **structured messages**, not ad-hoc JSON blobs.
- Changes should be **additive** whenever possible.
- Prefer **explicit fields** over “metadata maps” unless strongly justified.
- Keep stable identifiers and enums stable; avoid reusing enum numeric values.

> For detailed proto conventions (package naming, field numbering, compat rules, generation workflow), defer to `interoperability-steer`.

---

### 6. Consistency Rules That Make APIs Feel “Professional”

Professional APIs are boring in a good way: predictable, uniform, and easy to compose.

#### 6.1 Error Model Consistency

**Steer:** Every endpoint should fail in familiar ways and return errors in a consistent shape.

Minimum expectations:
- A stable **error code** (machine-oriented)
- A human message safe to show in logs/UI
- Structured **details** (field violations, retry hints, dependency failures)
- Correlation/request ID propagation for debugging

Avoid:
- Endpoint-specific error shapes
- Throwing raw exceptions through the boundary
- Hiding dependency failure causes behind generic “Internal error” without any structured details

#### 6.2 Pagination, Filtering, Sorting

Pick one canonical approach and use it everywhere.

**Convergence checklist:**
- Cursor vs offset: choose one default (cursor often scales better).
- Always define: `page_size` (bounded), `page_token`/`cursor`, `next_page_token`.
- Filtering is explicit and typed, not free-form strings.
- Sorting is explicit and validated.

Avoid:
- Endpoint-by-endpoint pagination styles
- Unbounded list endpoints
- “Filter strings” with ad-hoc parsing rules per endpoint

#### 6.3 Idempotency and Retries

Assume clients (and other scenarios) will retry.

Use idempotency patterns for:
- Create/execute commands that may be retried due to timeouts or network failures.
- Any operation that could create duplicate side effects.

Avoid:
- “Double create” issues because retry semantics weren’t considered.
- Non-deterministic behavior on repeated calls.

#### 6.4 Authentication and Authorization Consistency

Steer toward:
- One auth gateway pattern (middleware/interceptor) that sets request context.
- Authorization checks close to domain boundaries, but not duplicated everywhere.

Avoid:
- Some endpoints requiring auth and others “accidentally public.”
- Copy/pasted auth parsing across handlers.

#### 6.5 Observability Hooks (APIs Must Be Debuggable)

Minimum: consistent structured logging fields such as:
- request/correlation ID
- scenario/module name
- endpoint/method name
- latency + outcome
- dependency call outcomes

Avoid:
- Logging raw sensitive payloads
- No way to tie logs across scenario boundaries

---

### 7. Keep the Transport Edge Thin (Handlers Orchestrate)

The API boundary should do boundary work:
- Authentication / request context
- Validation
- Error translation
- Calling the correct domain/service function
- Returning typed responses

It should **not**:
- Contain business rules
- Perform multi-step domain workflows inline
- Mix persistence/integration calls directly into handlers without a service layer

#### 7.1 Convergence Pattern: Layering Diagram

```

API Boundary (routes/controllers/handlers)

* parse + validate + auth context + error mapping
* orchestrate a use-case call
  ↓
  Use-cases / Application services
* coordinates domain operations
* handles transactional boundaries if needed
  ↓
  Domain services
* core rules and invariants
  ↓
  Infra adapters (DB, queues, external APIs)

```

**Dependency rule:** domain must not depend on API transport details.

---

### 8. DRY Without Losing Clarity: Shared Utilities and Singletons

A “professional API” avoids repeated boilerplate while preserving readability.

Examples of good shared primitives:
- Request context builder (auth, request IDs, actor identity)
- Validation helpers (proto → validation; field violation formatting)
- Standard pagination helper
- Standard error translation layer
- Client singletons (HTTP clients, DB clients) with consistent configuration and timeouts

#### 8.1 Convergence Pattern: Extract Helper Decision Tree

```

You wrote helper logic in an endpoint
│
▼
Will this logic be used in 2+ endpoints or 2+ modules?
├─ NO → keep it local (optimize for readability)
└─ YES
│
▼
Is it a domain rule?
├─ YES → move to domain/service layer (not API utils)
└─ NO
│
▼
Is it boundary glue (validation, auth, error mapping, pagination)?
├─ YES → extract to shared API utility (canonical)
└─ NO  → consider a dedicated application service helper

```

**Avoid**
- “utils.ts” dumping grounds
- Singletons that hide global mutable state without need
- Copy/paste divergence for “same” behavior across endpoints

---

### 9. API Evolution: Backward Compatibility Is a Feature

Assume other scenarios and agents depend on you.

#### 9.1 Convergence Pattern: Change Strategy Table

| Desired change | Prefer | Avoid |
|---|---|---|
| Add a new optional field | Additive schema change | Breaking existing consumers |
| Change meaning of a field | New field + deprecate old | Silent semantic change |
| Remove a field/endpoint | Deprecate + migration window | Sudden removal |
| Replace a workflow | New endpoint/version + compat shim | Forcing all clients to update immediately |

**Steer:** Additive changes scale. Breaking changes create coordination tax.

---

### 10. API Documentation and Discoverability

A professional API can be learned quickly.

Minimum expectations:
- API “table of contents” grouped by domain/capability
- For each operation: intent, request/response, key errors, auth, idempotency, examples
- A mapping between **CLI commands** and API operations (since CLI is thin wrapper)

Avoid:
- Docs that only restate types without explaining behavior
- “Read the code” as the primary onboarding mechanism

---

### 11. API Coherence Audit (Brownfield-Friendly)

When inheriting or improving an existing scenario API, first establish the current shape.

**Audit prompts (what to look for):**
- Are endpoints grouped by domain boundaries?
- Do request/response shapes look uniform?
- Are errors consistent and typed?
- Where does domain logic live (handlers vs services)?
- Are there “god routers” or “misc” endpoints?
- Are there duplicate utility patterns?

**Typical red flags:**
- Huge handler files
- Inconsistent pagination
- Multiple error formats
- Handler contains business logic and persistence calls
- Proto types exist but aren’t end-to-end adopted

**Document findings** in a durable place (e.g., `docs/internal/API_NOTES.md` or equivalent):
- Current module map
- Top 5 inconsistencies to fix
- Proposed re-org plan (incremental, non-breaking)

---

### 12. Maintain Scenario Constraints

* Do **not** introduce new product features under the banner of “API cleanup.”
* Do **not** break existing API clients (other scenarios, UI, CLI) without a compatibility plan.
* Prefer incremental refactors that improve organization and consistency without large rewrites.
* Preserve operational targets (latency, reliability, correctness); API polish must not degrade them.

---

### 13. Output Expectations

You may:
- Re-organize endpoint modules to better match bounded contexts
- Extract shared boundary utilities (auth/context/validation/errors/pagination)
- Improve naming consistency and endpoint taxonomy (query/command/job)
- Improve API documentation and CLI↔API mapping documentation
- Add compatibility layers or deprecation markers

You must:
- Keep the API intuitive and domain-aligned (“screams” what the scenario does)
- Keep handlers thin; move business logic into services/use-cases
- Enforce consistent error shapes and response envelopes where applicable
- Ensure proto-first adoption is progressing (or explicitly documented why not)
- Avoid shallow refactors that only rename files without improving structure and reliability