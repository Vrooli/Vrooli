## Steer focus: Invariant Discovery & Enforcement (Behavior Definition & Verification)

Make every rule the code relies on **formally declared, anchored to code, and mechanically verified**. Invariants are the load-bearing facts of a scenario; this skill ensures they are stated where they belong (local to code, or additionally in the cross-cutting registry), backed by a real enforcement mechanism, and exercised by tests. The skill succeeds to the extent that intended behaviors and enforced behaviors are the same set.

An invariant is a condition that **must always be true** for the system to behave correctly: "this list is always sortable by date", "this route requires an authenticated user", "this account balance is never negative", "every `EndpointDescriptor.Path` is either a Connect procedure or carries a `RESTException`". Bugs are violations of invariants.

This skill replaces the retired `assumption-mapping-and-hardening` skill. An "assumption" is just an invariant that isn't yet enforced — both states are tracked here.

Do **not** turn current bugs, temporary workarounds, or incomplete behavior into "invariants." Stabilize true rules, do not freeze accidental behavior.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read boundary-of-responsibility-enforcement` — owns directory shape and per-layer responsibility; this skill assumes domain ownership is already clear.
- `prompt-manager skill read seam-discovery-and-enforcement` — owns substitution points; invariants at a seam ("fake and production agree on shape") are seam-skill territory.
- `prompt-manager skill read temporal-flow-audit` — owns ordering/lifecycle invariants; this skill defers temporal rules there and only catalogs their existence.
- `prompt-manager skill read interoperability-steer` — owns proto + `protovalidate` as the canonical boundary-validation mechanism this skill points to.
- `prompt-manager skill read error-semantics-recovery-path-design` — owns failure-mode design when an invariant's violation must degrade gracefully rather than fail.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/INVARIANTS.md` — prior invariant registry and enforcement status. **Code is the source of truth; verify claims against code before extending.**
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — domain map; cross-cutting invariants may originate here.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — known enforcement gaps deferred for later; soften-resolution decisions also land here.

Optional context:
- `docs/scenario-qa/methods/audit/invariant-discovery-and-enforcement.md` — when this lens applies, when it backfires, what the qa-contrarian challenges.

---

> **Template example domain — delete on generation.** The react-vite template ships a `notes` domain as a worked example. It is *not* a real domain every scenario inherits. Examples below use placeholder identifiers (`<domain>`, `<Resource>`, `<Flow>`); substituting them is the moment to check whether a leftover `notes` folder is template residue.

---

### 1. Scope Boundaries

**In scope:**
- discovering rules the code relies on (whether currently enforced or not)
- triaging each discovered rule into Enforce / Soften / Accept
- choosing the right enforcement mechanism per archetype (type / `protovalidate` / runtime guard / DB constraint / test / registry)
- declaring rules in code with the `// INVARIANT: <name>` tag at the enforcement site
- adding or strengthening violation-exercising tests
- maintaining `INVARIANTS.md` for cross-cutting (Tier 2) invariants only
- recording unenforced invariants as explicit gaps and softened-out rules as accepted-resolutions in `PROBLEMS.md`

**Out of scope (hand off):**
- directory shape and layer ownership → `boundary-of-responsibility-enforcement`
- seam shape and test substitution → `seam-discovery-and-enforcement`
- transition tables and lifecycle ordering → `temporal-flow-audit` (this skill catalogs that the temporal invariant exists)
- proto schema design and codegen workflow → `interoperability-steer`
- failure-mode design for soft-failure paths → `error-semantics-recovery-path-design`
- broad rewrites or new features under the banner of invariant cleanup
- creating standalone `INVARIANT_AUDIT.md`, `ASSUMPTIONS.md`, or per-domain registry files — the single INVARIANTS.md registry is the durable surface

---

### 2. Two-Tier Model: Local vs. Registry

Every invariant lives at one of two tiers. The tag convention and enforcement mechanism are the same at both tiers; the difference is whether the rule is *additionally* indexed in `INVARIANTS.md`.

**Tier 1 — Local invariant.** Scope is one function, one file, or one tightly-bounded module. Declaration and enforcement both live with the code:

- `// INVARIANT: <name>` tag on the guard, type, annotation, constraint, or validator that embodies the rule (this tag is the universal marker)
- Mechanism is one of: type encoding, `protovalidate` annotation, runtime guard, DB constraint, codegen check, or a violation-exercising test
- **No `INVARIANTS.md` entry required.** Adding one would create drift risk with no readership benefit.

**Tier 2 — Registry-worthy invariant.** Scope spans domains, layers, or has system-wide consequences. Declaration in code (same tag, same mechanism) AND a row in `INVARIANTS.md` so cross-cutting agents can discover the rule without reading every file.

**Tiering criterion — register in INVARIANTS.md when one or more is true:**
- Multiple domains depend on the rule (e.g., "every entity ID is a UUIDv7")
- Layers must agree (UI assumes shape X, API enforces shape X, DB constrains shape X — drift here is silent corruption)
- Violation has system-wide impact: security boundary, billing integrity, data corruption, irreversible state change
- Enforcement is itself cross-cutting (interceptor, codegen validator, registry drift check) — the mechanism touches code that doesn't otherwise have a reason to know about the rule

Otherwise the rule is local. Local rules can graduate to Tier 2 when their scope expands; graduation is just adding the row to `INVARIANTS.md` and ensuring the code tag and doc row agree on the name.

---

### 3. Triage Taxonomy

Before choosing a mechanism, classify every discovered rule into exactly one resolution:

| Resolution | When to choose | Where it lands |
|---|---|---|
| **Enforce** | The rule should always hold and the code already relies on it (or should). Pick a mechanism from §6 and encode it. | Code tag at the enforcement site; test exercising violation; Tier-2 rules additionally appear in `INVARIANTS.md`. |
| **Soften** | The rule is fragile or unrealistic; the right move is to change the code so it no longer *depends* on the rule (guard, default, fallback, graceful degradation). | Code change (no tag, since there's no longer an invariant). Record the decision in `PROBLEMS.md` under "Softened-out rules" with rationale. |
| **Accept** | The rule should hold, enforcement would be costly, and the residual risk is judged acceptable (cost-of-enforcement > impact × likelihood). | `INVARIANTS.md` Gaps section with rationale, impact, and trigger condition for revisiting. |

Rules:
- Every triaged rule has exactly one resolution. "TBD" entries are a code smell — either you haven't decided, or the rule isn't ready to be triaged.
- `Accept` is not a synonym for `Defer`. Deferring is just Accept with a written trigger ("revisit when concurrency model changes"). Both live in the same Gaps section.
- A rule resolved as `Soften` does **not** appear in `INVARIANTS.md` (it's no longer an invariant). The decision lives in `PROBLEMS.md` so future agents can see why the dependency was removed.

---

### 4. The `// INVARIANT:` Tag Convention

The tag is the universal marker that declares an invariant in code, at both tiers.

**Format:**

```
// INVARIANT: <camelCaseName>
```

- **`INVARIANT:` in uppercase** — matches the house convention (`KNOWLEDGE-OBSERVATORY` doc-anchor style).
- **`<camelCaseName>`** — scope-prefixed by the owning domain noun when local (e.g., `noteIDIsImmutable`), or no prefix when truly cross-cutting (e.g., `everyEndpointIsConnectOrTaggedRESTException`).
- The name **reads as an assertion of what is true**, not a wish ("should be") and not a mechanism ("hasNoSetter"). It describes the rule, not how it's enforced.

**Placement: on the line that carries the rule.** The tag sits *at the enforcement site* — the type definition, the proto field annotation, the guard `if` statement, the SQL `ALTER TABLE`, the codegen validator function. The rule of thumb: "the line you'd have to bypass to violate this rule."

Right placement (tag on the guard line itself):

```go
func (s *Service) Update(ctx context.Context, id NoteID, in UpdateInput) (*Note, error) {
    // INVARIANT: noteUpdateRequiresOwnership
    if !s.auth.CallerOwns(ctx, id) {
        return nil, ErrNotOwner
    }
    // ...
}
```

Wrong placement (vague tag above the function — a verifier cannot tell what line is "the rule"):

```go
// INVARIANT: noteUpdateRequiresOwnership   ← do not do this
func (s *Service) Update(...) (*Note, error) { ... }
```

**Test linkage:** for each tag, a sibling test file contains a test whose name or comment references the same invariant name. The common pattern:

```go
// enforces invariant: noteUpdateRequiresOwnership
func Test_Update_RejectsNonOwner(t *testing.T) { /* ... */ }
```

For invariants whose enforcement *is* the build (codegen validators, type-system encodings), the build step itself is the evidence — no separate test required.

---

### 5. Invariant Maturity Ladder (Per Invariant)

The ladder applies **per invariant**, not per scenario. A scenario may have one L5 invariant and ten L1 invariants. Move only as far as risk and time justify. L5 differs between tiers.

| Level | Name | Tier 1 (Local) | Tier 2 (Registry-worthy) |
|---|---|---|---|
| L0 | Implicit | Rule lives only in someone's head; rediscovered each loop. | Same. |
| L1 | Tagged | `// INVARIANT: <name>` on the enforcement site with a one-line prose rationale near it. | Tagged in code AND entered in `INVARIANTS.md` with prose statement. |
| L2 | Code-anchored | Tag sits on the guard/type/check that actually embodies the rule (not floating). | Same, plus the doc row carries the `path:file:line` anchor. |
| L3 | Test-enforced | A violation test exists; its name or comment references the invariant name. | Same. |
| L4 | Mechanism-declared | The enforcement mechanism is named (`type-system` / `protovalidate` / `runtime-guard` / `db-constraint` / `test` / `registry`) and that mechanism actually exists in code. | Same; the doc row's `Mechanism` column matches code reality. |
| L5 | Drift-gated | Local convention (lint rule, grep target, or convention-check) catches new code that violates the tag's contract. | Registry drift gate: every `INVARIANT:` tag at Tier-2 scope appears in `INVARIANTS.md` and vice versa, validated by a registry test analogous to `validateTransport` (`path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`). |

The level is not a vanity score. It tells the next agent what kind of drift is still possible for this specific invariant.

---

### 6. Invariant Archetype → Enforcement Mechanism

Pick the mechanism by the *shape* of the rule, not by what is most convenient. Each row is a decision shortcut; the right answer is usually the leftmost mechanism that can carry the rule.

| Archetype | Example | Preferred mechanism | Fallback | Cross-link |
|---|---|---|---|---|
| Type / shape | "this field is non-empty"; "this enum has exactly N values" | Type system (non-nullable, discriminated union, sealed enum) | `protovalidate` annotation at boundary; test | `interoperability-steer` |
| Range / value | "value is between 0 and 100"; "string matches regex" | `protovalidate` constraint on proto field | Runtime guard at the boundary entrypoint | `interoperability-steer` |
| Ordering / temporal | "X must happen before Y"; "terminal states cannot be escaped" | State machine + transition function in the owning domain | Trace tests over the workflow | `temporal-flow-audit` |
| Reference / referential integrity | "foreign key always resolves"; "owner of resource still exists" | DB foreign-key constraint + cascade rule | Service-level guard with test | — |
| Identity / stability | "ID is stable across runs"; "IDs are globally unique" | Type (opaque ID) + DB uniqueness constraint | Test asserting stability across reload | — |
| Cross-aggregate / invariant sum | "sum of debits = sum of credits"; "count(active children) ≤ parent.cap" | DB constraint or transactional check | Scheduled reconciliation job + test | — |
| Cross-surface parity | "every `EndpointDescriptor.Path` is Connect or has `RESTException`" | Registry validator (codegen-time check) | Generated-artifact diff test | `api-steer` / `interoperability-steer` |
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

A real example of this pattern, lit up end-to-end, is the `RESTReason` enum at `path:templates/scenarios/react-vite/api/internal/module/module.go` paired with the `validateTransport` check at `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go`. The invariant ("every endpoint is either Connect or a tagged REST exception") is type-encoded (closed enum), registry-enforced (validator iterates all endpoints), and drift-gated (codegen fails on violation). That is what L5 Tier-2 looks like.

---

### 7. Discovery: Finding Rules the Code Already Relies On

Discovery is **archaeology, not invention.** An invariant is a rule the code *currently relies on*. If you can't find code that relies on it, it's not an invariant — it's a PRD requirement (belongs in the PRD) or speculation (don't record).

**Earn-the-entry rule:** include only rules whose violation would produce a bug class the team would want to prevent forever. Micro-preconditions that "just need a fix and a regression test" stay in the test suite, not the registry.

**Per-domain density signal** (sanity check, not a target):

| Domain weight | Expected order of magnitude (Critical invariants) |
|---|---|
| Pure CRUD / display | 2–4 |
| Temporal / orchestration | 6–12 (state-machine ones hand off to TEMPORAL-FLOWS; a one-line existence entry remains) |
| Security / billing / auth | 10+ |
| Pure config / static | 0–2 (sometimes legitimately none) |

Zero in a high-risk domain is a discovery gap. Thirty in a CRUD domain is conflation with implementation details — promote the meta-rules, demote the specifics.

**Static signals in code:**
- repeated guards or assertions at the start of functions ("this value is never null here")
- branches that assume a particular shape and `panic` / `throw` / return error on others
- comments containing "must", "always", "never", "invariant", "assume", "precondition", "postcondition"
- error types that name a rule violation (`ErrOwnerMismatch`, `ErrAlreadyClaimed`)
- type assertions and unchecked casts that work because the producer guarantees the shape
- `// TODO` / `// HACK` near validation paths

**Discovery fingerprints absorbed from the retired `assumption-mapping-and-hardening` skill:**
- **Data shape & nullability assumptions**: properties accessed without prior null checks; lists iterated without empty-check where empty would be ambiguous
- **External-system assumptions**: response fields assumed always present; assumed latency, availability, or error behaviors; webhook ordering assumptions
- **Timing / environment assumptions**: functions assuming call order; components assuming mount order; assumed env vars or runtime modes
- **User-behavior assumptions**: assumed input quality ("they won't paste huge blobs"); assumed permissions; assumed flows
- **Test-fixture assumptions**: tests relying on narrow happy-path fixtures that encode an assumption not enforced in production logic

Each fingerprint is a *candidate* invariant. Triage per §3 before encoding.

**Domain signals:**
- PRD or operational targets that say "must", "never", "exactly", "at most N"
- existing tests with names like `Test_RejectsNegativeBalance`, `Test_DuplicateIDFails`
- finite enums and discriminated unions
- DB unique / check / foreign-key constraints already in migrations
- `protovalidate` annotations already present

**Useful greps (run from `scenarios/{{TARGET}}/`):**

```bash
# named invariant tags already in the codebase
rg -n 'INVARIANT:\s*\w' --type-add 'src:*.{go,ts,tsx,py,rs,sql,proto}' -tsrc

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

Treat each match as a *candidate*. Promote only if the rule aligns with PRD intent, real usage, and existing tests. Demote incidental quirks into `PROBLEMS.md`.

---

### 8. Programmatic Verification: What Is and Isn't Checkable

This skill produces tag conventions and registry shape designed to support increasingly programmatic verification over time (tidiness-manager smart-scan or a dedicated CLI as the substrate).

**Can be programmatically checked (day-one floor):**
1. **Tag format** — `// INVARIANT: <camelCaseName>` on a comment line. Catches typos like `// Invariant:`, missing colon, inconsistent case.
2. **Name uniqueness** — no two unrelated tags share a name.
3. **Test linkage** — every invariant name appears in at least one test name or test comment. For codegen-validator invariants, the build step substitutes for the test.
4. **Mechanism evidence** — the tagged line is structurally a guard / type / annotation / constraint / validator (not a random comment).
5. **Registry parity (Tier 2 only)** — every Tier-2 tag appears in `INVARIANTS.md` and vice versa, with the doc's `path:` resolving near the tag.

**Cannot be programmatically checked (stays human/AI judgment):**
1. Whether the prose statement is true (matches actual behavior).
2. Whether the mechanism is sufficient (a runtime guard might be in place when a DB constraint is actually needed).
3. Whether the test exercises the *violation* path, not just the happy path.
4. Whether the rule is correctly classified as Tier 1 vs Tier 2.
5. Whether the right mechanism was chosen from §6.

Day-one programmatic value: structural drift, name uniqueness, test linkage, registry parity. Smart-scan can grow heuristics for the harder checks over time. Every recurring manual finding is a candidate for promotion into the programmatic surface.

---

### 9. Audit Workflow

1. **Read the registry.** Open `docs/internal/INVARIANTS.md` if present. List every claimed Tier-2 invariant; for each, the next step is to verify or refute.
2. **Verify against code.** For each entry, follow it to the guard/type/constraint. If the code does not match the claim, the doc is wrong — fix the doc or fix the code, never let the mismatch stand.
3. **Walk discovery per domain.** Use §7 signals and greps. For each domain: existing tags → guards/errors → tests → declarative annotations → PRD cross-reference. Stop when each domain has been walked at the current pass's risk threshold (Critical first, Important next if time permits).
4. **Triage each candidate (§3).** Pick exactly one resolution: Enforce, Soften, or Accept.
5. **For Enforce candidates: classify the archetype (§6) and pick the mechanism.** Prefer making illegal states unrepresentable (types) over runtime guards over tests.
6. **For Enforce candidates: decide the tier.** Use the §2 criterion. Default to Tier 1 unless one of the four registry-worthy conditions is met.
7. **Encode the invariant.**
   - Add `// INVARIANT: <name>` at the enforcement site (§4).
   - Add or strengthen the violation test; the test name or comment references the invariant.
   - For boundary input rules, prefer a `protovalidate` annotation over a hand-rolled handler check.
   - For cross-artifact rules, add a codegen / boot-time validator following the `validateTransport` pattern.
8. **For Soften decisions: change the code so the rule is no longer required.** Record the decision in `PROBLEMS.md` under "Softened-out rules" with rationale.
9. **For Accept decisions: record in `INVARIANTS.md` Gaps section** with risk, rationale, and trigger condition.
10. **Reconcile the registry.** Tier-2 entries each have name, prose statement, `path:file:line` anchor, declared mechanism, and pointer to enforcing test.

Do not introduce new noisy or user-hostile failures while encoding invariants. Protect correctness without degrading UX (see `error-semantics-recovery-path-design` for soft-failure shapes).

---

### 10. Canonical File Shape for `INVARIANTS.md`

The doc is a **registry**, not a deep prose document. Keep it short, structured, and code-anchored. At L3+ it has four sections. For scenarios with more than ~15 Critical entries, group within each section by domain.

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

| Invariant | Resolution | Why unenforced | Risk if violated | Trigger to revisit |
|---|---|---|---|---|
| uniqueWorkflowIdPerScenario | Accept | No DB constraint yet; only service-level check. | Duplicate IDs would silently overwrite. | When concurrent writers ship. |
```

Rules:
- Code is the source of truth. If the table claims something the code does not, the table is the bug.
- Anchors use `path:` references (machine-readable).
- Only Tier-2 (registry-worthy) invariants appear here. Tier-1 invariants live entirely in code via the `INVARIANT:` tag.
- Cross-cutting observations that change the mental model belong in `ARCHITECTURE.md`; unresolved drift and soften-resolutions belong in `PROBLEMS.md`. Keep `INVARIANTS.md` focused on the registry.

---

### 11. Migrating Scenarios with Existing `ASSUMPTIONS.md`

Some scenarios have an `ASSUMPTIONS.md` from the retired `assumption-mapping-and-hardening` skill. When you next audit such a scenario:

1. Read each entry in `ASSUMPTIONS.md`.
2. Triage per §3: Enforce, Soften, or Accept.
3. For Enforce: encode per §7 (tag at enforcement site + test); Tier-2 entries move into `INVARIANTS.md` Critical/Important.
4. For Soften: change the code, record in `PROBLEMS.md`.
5. For Accept: move into `INVARIANTS.md` Gaps with trigger.
6. Delete `ASSUMPTIONS.md` once empty.

Do not preserve `ASSUMPTIONS.md` as a parallel doc — the single registry is the durable surface, and the local `INVARIANT:` tag is the durable code marker. If a full migration in one pass is too large, leave `ASSUMPTIONS.md` in place with a header note pointing at this skill and record the migration as an entry in `PROBLEMS.md`.

---

### 12. Documentation

Use `knowledge-observatory-tools` to read and update stable docs.

- `docs/internal/INVARIANTS.md` — primary surface. Four-section schema above. Tier-2 only.
- `docs/concepts/ARCHITECTURE.md` — when the invariant changes the mental model ("the API is the source of truth for X"), record it there too with a back-link.
- `docs/internal/PROBLEMS.md` — unresolved enforcement gaps, softened-out rules with rationale, and the narrative behind risky deferred work.

Do not create one-off `INVARIANT_AUDIT.md` reports, parallel `ASSUMPTIONS.md` files, or per-domain registry files. The single registry is the durable surface.

---

### **13. Output Expectations**

By the end of this loop, the scenario should:
- have a verified `INVARIANTS.md` where every Tier-2 entry is code-anchored and test-enforced (≥ L3)
- have at least one invariant that moved up the maturity ladder
- prefer mechanism strength in this order: type-system > declarative annotation (`protovalidate` / DB constraint) > runtime guard > test
- have `// INVARIANT: <name>` tags in code at the enforcement site for every Critical and Tier-2 invariant
- record softened-out rules in `PROBLEMS.md` with rationale (not in `INVARIANTS.md`)
- record accepted gaps in `INVARIANTS.md` with trigger conditions
- avoid promoting incidental quirks or current bugs into the registry

Avoid superficial edits. The goal is not a longer document; it is a codebase where the rules that must hold are mechanically protected and a future agent cannot violate one by accident.
