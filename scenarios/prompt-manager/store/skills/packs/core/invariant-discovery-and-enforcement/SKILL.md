## Steer focus: Invariant Discovery & Enforcement

Prioritize **making the rules that must always hold visible, named, and mechanically enforced** in `scenarios/{{TARGET}}/`. Discover implicit invariants first, then move each critical one up the maturity ladder from prose → code reference → test → declared mechanism → registry-gated drift check.

An invariant is a condition that **must always be true** for the system to behave correctly: "this list is always sortable by date", "this route requires an authenticated user", "this account balance is never negative", "every `EndpointDescriptor.Path` is either a Connect procedure or carries a `RESTException`". Invariants are the load-bearing facts that bugs violate.

Do **not** turn current bugs, temporary workarounds, or incomplete behavior into "invariants." Stabilize true rules, do not freeze accidental behavior.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — owns directory shape and per-layer responsibility; this skill assumes domain ownership is already clear.
- `prompt-manager skill read seam-discovery-and-enforcement` — owns substitution points; invariants at a seam (e.g. "fake and production agree on shape") are seam-skill territory.
- `prompt-manager skill read temporal-flow-audit` — owns ordering/lifecycle invariants; this skill defers temporal rules there and only catalogs their existence.
- `prompt-manager skill read interoperability-steer` — owns proto + `protovalidate` as the canonical boundary-validation mechanism this skill points to.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/INVARIANTS.md` — prior invariant registry and enforcement status. **Code is the source of truth; verify claims against code before extending.**
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map; cross-cutting invariants may originate here.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — known enforcement gaps deferred for later.

Optional context:
- `docs/scenario-qa/methods/audit/invariant-discovery-and-enforcement.md` — when this lens applies, when it backfires, what the qa-contrarian challenges.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain as a worked example. It is *not* a real domain every scenario inherits. Examples below use placeholder identifiers (`<domain>`, `<Resource>`, `<Flow>`); substituting them is the moment to check whether a leftover `notes` folder is template residue.

---

### 1. Scope Boundaries

**In scope:**
- discovering implicit invariants embedded in guards, assertions, branches, error paths
- naming invariants (`invariant: <name>`) so they are findable in code and docs
- choosing the right enforcement mechanism per archetype (type / `protovalidate` / runtime guard / DB constraint / test / registry)
- adding or strengthening enforcement tests that fail when an invariant is violated
- reconciling the invariant registry in `INVARIANTS.md` against code (no claim without evidence)
- recording unenforced invariants as explicit gaps

**Out of scope:**
- directory shape and layer ownership (use `boundary-of-responsibility-enforcement`)
- seam shape and test substitution (use `seam-discovery-and-enforcement`)
- transition tables and lifecycle ordering (use `temporal-flow-audit`; this skill only catalogs that the temporal invariant exists)
- proto schema design and codegen workflow (use `interoperability-steer`)
- broad rewrites or new features under the banner of invariant cleanup

---

### 2. Invariant Maturity Ladder

Assess each candidate invariant independently. A scenario may have one L5 invariant and ten L1 invariants. Move only as far as risk and time justify.

| Level | Name | What exists | Gate to next level |
|---|---|---|---|
| L0 | Implicit | Invariant lives only in agents' heads; rediscovered each loop. | Write it down in `INVARIANTS.md` under the 4-section schema. |
| L1 | Documented | `docs/internal/INVARIANTS.md` lists the invariant in Critical / Important / Gaps with prose description and rationale. | Anchor the invariant to a code location. |
| L2 | Code-anchored | Each Critical invariant entry carries a `path:file:line` reference to the guard, type, schema, or check that embodies it. | Add (or identify) a test that fails when the invariant is violated. |
| L3 | Test-enforced | At least one test asserts the violation case fails (not just the happy path). The test name or comment references the invariant name. | Declare the enforcement mechanism class. |
| L4 | Mechanism-declared | Each invariant declares its enforcement class — `type-system` / `protovalidate` / `runtime-guard` / `db-constraint` / `test` / `registry` — and the doc cross-checks the declared mechanism actually exists in code. | Add a registry / drift test. |
| L5 | Drift-gated | Every `invariant: <name>` tag in code appears in `INVARIANTS.md` and vice versa, validated by a registry test analogous to `validateTransport` (`path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`). New invariants cannot be added in one place without the other. | — (steady state) |

The level is not a vanity score. It tells the next agent what kind of drift is still possible.

---

### 3. Invariant Archetype → Enforcement Mechanism

Pick the enforcement mechanism by the *shape* of the invariant, not by what is most convenient. Each row is a decision shortcut; the right answer is usually the leftmost mechanism that can carry the rule.

| Archetype | Example | Preferred mechanism | Fallback | Cross-link |
|---|---|---|---|---|
| Type / shape | "this field is always non-empty"; "this enum has exactly N values" | Type system (non-nullable, discriminated union, sealed enum) | `protovalidate` annotation at boundary; test | `interoperability-steer` |
| Range / value | "value is between 0 and 100"; "string matches regex" | `protovalidate` constraint on proto field | Runtime guard at the boundary entrypoint | `interoperability-steer` |
| Ordering / temporal | "X must happen before Y"; "terminal states cannot be escaped" | State machine + transition function in the owning domain | Trace tests over the workflow | `temporal-flow-audit` |
| Reference / referential integrity | "foreign key always resolves"; "owner of resource still exists" | DB foreign-key constraint + cascade rule | Service-level guard with test | — |
| Identity / stability | "ID is stable across runs"; "IDs are globally unique" | Type (`uuid`, opaque ID) + DB uniqueness constraint | Test asserting stability across reload | — |
| Cross-aggregate / invariant sum | "sum of debits = sum of credits"; "count(active children) ≤ parent.cap" | DB constraint or transactional check | Scheduled reconciliation job + test | — |
| Cross-surface parity | "every `EndpointDescriptor.Path` is a Connect procedure or has `RESTException`" | Registry validator (codegen-time check) | Generated-artifact diff test | `api-steer` / `interoperability-steer` |
| Authorization / ownership | "caller must own resource"; "admin-only operation" | Interceptor / middleware at the boundary | Per-handler guard with test | — |

Decision rule:

```text
Can the type system make the illegal state unrepresentable?
  YES -> encode in types; tests guard the conversion boundary only.
Can a declarative annotation (protovalidate, DB constraint) carry the rule?
  YES -> declare it there; tests assert the annotation is present and enforced.
Does correctness depend on history or ordering?
  YES -> hand off to temporal-flow-audit; record only the existence here.
Does the rule cross artifacts (code <-> registry <-> generated file)?
  YES -> add a drift test at codegen or boot time (L5 pattern).
Otherwise:
  Encode as a named runtime guard + an enforcement test.
```

A real example of this pattern, lit up end-to-end, is the `RESTReason` enum at `path:templates/scenarios/react-vite/api/internal/module/module.go` paired with the `validateTransport` check at `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`. The invariant ("every endpoint is either Connect or a tagged REST exception") is type-encoded (closed enum), registry-enforced (validator iterates all endpoints), and drift-gated (codegen fails on violation). That is what L5 looks like.

---

### 4. Discovery: Finding Implicit Invariants

Implicit invariants leave fingerprints. Search for them.

**Static signals in code:**
- repeated guards or assertions at the start of functions ("this value is never null here")
- branches that assume a particular shape and `panic` / `throw` / return error on others
- comments containing "must", "always", "never", "invariant", "assume", "precondition", "postcondition"
- error types that name a rule violation (`ErrOwnerMismatch`, `ErrAlreadyClaimed`)
- type assertions and unchecked casts that work because the producer guarantees the shape
- `// TODO` / `// HACK` near validation paths

**Domain signals:**
- PRD or operational targets that say "must", "never", "exactly", "at most N"
- existing tests with names like `Test_RejectsNegativeBalance`, `Test_DuplicateIDFails`
- finite enums and discriminated unions
- DB unique / check / foreign-key constraints already in migrations
- protovalidate annotations already present

**Useful greps (run from `scenarios/{{TARGET}}/`):**

```bash
# named invariant tags already in the codebase
rg -n 'invariant:\s*\w' --type-add 'src:*.{go,ts,tsx,py,rs}' -tsrc

# guards and assertions that imply an unwritten rule
rg -n '\b(assert|invariant|require|must|panic\()' -tsrc

# DB constraints declared in migrations vs claims in INVARIANTS.md
rg -n '(UNIQUE|FOREIGN KEY|CHECK \()' --type sql

# hand-rolled validation that should be protovalidate at the proto boundary
rg -n 'len\([^)]+\)\s*==\s*0|strings\.TrimSpace\([^)]+\)\s*==\s*""' api/internal

# mutators that change state without an explicit guard
rg -n 'func.*\b(Update|Set|Adjust|Apply|Increment|Decrement)\w*\b' api/internal

# claims in INVARIANTS.md that reference no code
rg -n '^\s*-\s' docs/internal/INVARIANTS.md
```

Treat each match as a *candidate* invariant. Promote to the registry only if it aligns with PRD intent, real usage, and existing tests. Demote incidental quirks into `PROBLEMS.md`.

---

### 5. Audit Workflow

1. **Read the registry.** Open `docs/internal/INVARIANTS.md` if present. List every claimed invariant; for each, the next step is to verify or refute.
2. **Verify against code.** For each Critical entry, follow it to the guard/type/constraint. If the code does not match the claim, the doc is wrong — fix the doc or fix the code, never let the mismatch stand.
3. **Discover new candidates.** Run the greps in §4. Cross-reference PRD, error types, and existing tests.
4. **Classify each candidate.** Pick the archetype from §3 and the preferred mechanism. Decide if it is Critical (corruption/security/crash) or Important (recoverable).
5. **Choose enforcement.** Prefer making illegal states unrepresentable (types) over runtime guards over tests. If the rule already has a stronger mechanism available, escalate.
6. **Encode the invariant.**
   - Add a `// invariant: <name>` comment on the guard / type / check.
   - Add or strengthen the violation test. The test name should reference the invariant name.
   - For boundary input rules, prefer a `protovalidate` annotation on the proto field over a hand-rolled check in the handler.
   - For cross-artifact rules (registry parity), add a codegen / boot-time validator following the `validateTransport` pattern.
7. **Handle violations gracefully.** Fail fast in trusted internal paths; fail with clear user-facing errors at the boundary. Avoid leaking internals; preserve data integrity.
8. **Update the registry.** Reconcile `INVARIANTS.md` so each Critical entry has a name, prose statement, `path:file:line` anchor, declared mechanism, and a pointer to its enforcing test.
9. **Record gaps.** Invariants that cannot be enforced this loop go in the Gaps section with a concrete next step and risk note.

Do not introduce new noisy or user-hostile failures while encoding invariants. Protect correctness without degrading UX.

---

### 6. Canonical File Shape for `INVARIANTS.md`

The doc is a **registry**, not a deep prose document. Keep it short, structured, and code-anchored. At L3+ it has four sections:

```markdown
# Invariants

## Critical Invariants

| Name | Statement | Mechanism | Code Anchor | Enforcing Test | Notes |
|---|---|---|---|---|---|
| ownerMustMatchCaller | A user can only mutate resources they own. | runtime-guard + test | path:api/internal/<domain>/service.go:142 | path:api/internal/<domain>/service_test.go:Test_RejectsNonOwnerUpdate | Pre-protovalidate; candidate to lift to interceptor. |
| balanceNeverNegative | Account balance must never drop below zero. | db-constraint + test | path:api/migrations/0007_balance_check.sql | path:api/internal/ledger/ledger_test.go:Test_RejectsNegativeBalance | — |
| restPathHasReason | Every EndpointDescriptor.Path is Connect or tagged with RESTException. | registry | path:api/cmd/gen-endpoints/main.go:validateTransport | codegen fails | L5 pattern. |

## Enforcement Mechanisms

| Mechanism | Where it lives | What it catches | What it does NOT catch |
|---|---|---|---|
| type-system | proto messages, Go types, TS discriminated unions | shape / nullability / closed enums | range, cross-field, cross-row |
| protovalidate | `path:packages/proto/schemas/{{SCENARIO_ID}}/v1/.../*.proto` | range, regex, required, value constraints at ingress | runtime-only or cross-aggregate rules |
| runtime-guard | service / handler entrypoints | preconditions that types cannot express | rules better moved to protovalidate |
| db-constraint | migrations | uniqueness, FK, range CHECK, transactional invariants | application-layer rules |
| test | `*_test.go` / `*.test.ts` | regression of any of the above | invariants no one wrote a test for |
| registry | codegen validators (`validateTransport`-style) | cross-artifact parity drift | invariants within a single file |

## Important Invariants

Same table shape as Critical. Violation is recoverable (degraded UX, retriable) rather than corrupting.

## Gaps

| Invariant | Why unenforced | Risk if violated | Next concrete step |
|---|---|---|---|
| uniqueWorkflowIdPerScenario | No DB constraint yet; only service-level check. | Duplicate IDs would silently overwrite. | Add UNIQUE constraint + migration. |
```

Rules:
- Code is the source of truth. If the table claims something the code does not, the table is the bug.
- Anchors use `path:` references (machine-readable).
- Cross-cutting observations that change the mental model belong in `ARCHITECTURE.md`; unresolved drift belongs in `PROBLEMS.md`. Keep `INVARIANTS.md` focused on the registry.

---

### 7. Documentation

Use `knowledge-observatory-tools` to read and update stable docs.

- `docs/internal/INVARIANTS.md` — primary surface. Four-section schema above. Create `path:scenarios/{{TARGET}}/docs/internal/` if needed.
- `docs/concepts/ARCHITECTURE.md` — when the invariant changes the mental model (e.g. "the API is the source of truth for X"), record it there too with a back-link.
- `docs/internal/PROBLEMS.md` — record unresolved enforcement gaps and risky deferred work; the Gaps table in `INVARIANTS.md` is the index, `PROBLEMS.md` holds the narrative.

Do not create one-off `INVARIANT_AUDIT.md` reports. The registry is the durable surface.

---

### 8. Output Expectations

By the end of this loop, the scenario should:
- have a verified `INVARIANTS.md` where every Critical entry is code-anchored and test-enforced (≥ L3)
- have at least one invariant that moved up the maturity ladder
- prefer mechanism strength in this order: type-system > declarative annotation (`protovalidate` / DB constraint) > runtime guard > test
- have named `invariant: <name>` tags in code wherever a guard embodies a Critical invariant
- record unenforced invariants in the Gaps section with a concrete next step
- avoid promoting incidental quirks or current bugs into the registry

Avoid superficial edits. The goal is not a longer document; it is a codebase where the rules that must hold are mechanically protected and a future agent cannot violate one by accident.
