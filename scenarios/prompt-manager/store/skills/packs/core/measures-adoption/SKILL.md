---
name: "measures-adoption"
description: "Steers a scenario toward provider-validated Measures adoption without degrading existing analytics products. It uses a decision model for direct aggregates, authoritative external reads, and event-sourced CQRS analytical read models; requires typed, parameterized, provenance-stamped Measures over shared analytical semantics; and distinguishes a transport-only aggregate contract from a Stats/dashboard/CLI product. Uses `measures-health validate scenario` as the maturity source of truth and cites packages/measures-go/README.md and docs/concepts/MEASURES.md."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer"]
  targetDimensions: ["measures"]
  tags: ["measures","measures-go","measures-health","analytical-query","time-window","cqrs","event-log","read-model","manifest","measure-block","governance","auto-execution-gate","search-hub","analytics-product","semantic-parity","behavioral-probe","adoption","steer"]
  icon: "bar-chart"
  status: "active"
  defaultScope: "architecture-scope"
  programmaticHome: "measures-health:measures"
  revision: 4
  createdAt: "2026-06-08T00:00:00Z"
  updatedAt: "2026-07-26T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager", "prompt-manager skill", "prompt-manager skill read", "vrooli scenario"]
  origin:
    kind: "authored"
---
## Steer focus: Measures Adoption

Prioritize making analytical questions in `scenarios/{{TARGET}}/` answerable as declared, typed, parameterized Measures without changing the scenario's analytical meaning or degrading an existing operator analytics product. Build each Measure as a thin, auditable query over the scenario's authoritative analytical substrate.

Your goal is for `measures-health validate scenario {{TARGET}}` to report the provider-owned maturity as unblocked. Do not invent business metrics merely to satisfy coverage. Expose the analytical questions that the scenario already owns and preserve every established consumer surface that has product value.

Required reading:

- `path:packages/measures-go/README.md` — contract API, registry, deterministic parameter resolution, provenance, and optional CQRS substrate.
- `path:docs/concepts/MEASURES.md` — measure vocabulary, domain coverage, waivers, and execution gate.
- `prompt-manager skill read cli-steer` — manifest bindings, Measure blocks, and governance.
- `prompt-manager skill read api-steer` — typed Connect contracts.
- `prompt-manager skill read storage-steer` — durable data ownership and event-log/read-model implementation.
- `prompt-manager skill read knowledge-observatory-tools` — durable scenario documentation.

Read first when present:

- `scenarios/{{TARGET}}/docs/internal/SEAMS.md`
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`

Universal skill-quality rules are in `path:docs/agent-system/SKILL_AUTHORING.md`.

---

### 1. Scope boundaries

**In scope**:

- Declare Measures through the manifest, governance, proto binding, and `measures-go` Registry.
- Classify each stateful domain as covered, waived, or not expected.
- Select an analytical substrate and ensure every consumer of a shared metric uses the same semantics.
- Preserve or improve existing Stats, dashboard, CLI, and API consumer surfaces.
- Record the Measures serve boundary and deferred work in `scenarios/{{TARGET}}/docs/internal/SEAMS.md` and `PROBLEMS.md`.

**Out of scope**:

- Inventing product metrics or changing the domain's business behavior.
- Replacing the scenario's CLI ergonomics, Connect contract design, or durable-storage architecture without the owning skill and task scope.
- Owning search-hub's central Measure index or the `measures-health` maturity implementation.

---

### 2. Vocabulary and architectural invariant

Use these terms consistently.

| Term | Meaning |
|---|---|
| **Measure** | A named, typed, parameterized analytical query. It is a contract and consumer surface, not the source of analytical truth. |
| **Analytical source of truth** | Durable facts from which an answer can be reproduced: domain state, an authoritative external system, or an append-only event log. |
| **Analytical read model** | Query-optimized derived state built from the analytical source of truth. |
| **Projection** | The deterministic fold that applies source facts to an analytical read model. |
| **Replay** | Rebuild a projection from retained historical facts. |
| **Watermark** | The last source position incorporated by an incremental projection. |
| **Consumer surface** | A purpose-built interface over analytical semantics: a Measure, typed RPC, CLI, Stats dashboard, or operator workflow. |
| **Semantic parity** | Two consumer surfaces return the same result for the same definitions, scope, time window, and source snapshot. |

The invariant is: define an analytical concept once, compute it from one authoritative substrate, and expose it through the consumer surfaces that need it. A Measure does not replace a Stats product merely because both answer related questions.

---

### 3. The Measure contract

A Measure is assembled from three sources joined on the manifest binding:

```
MeasureDeclaration = manifest measure block
                   + manifest governance
                   + proto-derived request schema
```

Put intent, example questions, result presentation, and defaults in the manifest. Put parameter types, bounds, enums, and required-ness in proto. Do not duplicate the proto schema in the manifest.

Apply these non-negotiable contract rules:

- Use the canonical `time_window` type for temporal questions.
- Abstain with `needs[]` when a required parameter cannot be resolved.
- Never auto-execute a `write` or `destructive` Measure.
- Return provenance for every answer.
- Use one Registry and one compute path for a Measure and any equivalent typed RPC.

Measures are not prompt-manager Actions and are not operational telemetry.

---

### 4. Choose the analytical substrate

Use the simplest substrate that preserves the required semantics. Do not introduce an event log because it sounds more mature. Do not repeatedly scan or reimplement the same historical calculations when the scenario already needs a shared analytical read model.

| Situation | Preferred substrate | Required property |
|---|---|---|
| A current value or one bounded aggregate answers the question | Direct query over durable domain state | The query has explicit scope and time semantics. |
| An external system is authoritative | Typed live read from that system | State freshness and availability limits in provenance or contract documentation. |
| Multiple historical questions fold over the same lifecycle facts | Event-sourced CQRS read model | Retain facts, replay the projection, and incrementally refresh to a watermark before reads. |
| A mature analytics product already derives insights from historical facts | Its existing analytical substrate | Make Measures and product views thin consumers of shared query semantics. |

For the event-sourced case, use the established CQRS terms: an append-only event log is the historical source; a projection builds an analytical read model; replay proves rebuildability; a watermark makes refresh incremental. This is a recommended architecture for shared historical analytics, not a fleet-wide requirement.

Do not use an event log as a second, competing source of truth. If durable domain state is authoritative and historical reconstruction is not required, query that state directly.

---

### 5. Separate contracts from products

Classify an existing Stats-like surface before changing it.

| Existing surface | Classification | Measures migration rule |
|---|---|---|
| A transport-only aggregate endpoint with no distinct operator workflow or product behavior | Legacy aggregate contract | Decompose it into Measures when semantic parity and consumer migration are proven. |
| Dashboard, operator CLI, recommendations, forecasts, drilldowns, or curated analytical workflow | Analytics product | Retain and improve it. Add Measures as a programmatic consumer surface over shared analytical semantics. |
| A mixed surface | Split the product behavior from unstructured machine aggregation | Migrate only the machine-contract portion after proof. Keep product behavior intact. |

Never rename an operator-facing Stats product to “Measures” merely because Measures are added. Never delete a Stats route, UI, CLI group, projection, or historical capability solely to clear a Measures maturity rung.

Before retiring a legacy aggregate contract, prove all of the following:

1. Every supported question has an equivalent declared Measure or an intentionally retained product surface.
2. Parameter, time-window, unit, zero-result, and provenance semantics match.
3. All identified machine consumers have migrated.
4. Product UX and operational workflows remain available and accepted.
5. Contract tests compare representative answers at the same source snapshot.

Record an intentionally retained product/contract boundary in `SEAMS.md`. Record any approved retirement work in `PROBLEMS.md` until complete.

---

### 6. Cover domains and implement thin consumers

Run the provider before manual judgment:

```bash
measures-health validate scenario {{TARGET}}
measures-health validate scenario {{TARGET}} --probe
```

Use the provider's expected/covered/waived classification and maturity output. Do not duplicate its ladder in this skill.

For each expected stateful domain:

1. Identify the existing analytical question families.
2. Declare one Measure per independently addressable question family.
3. Select the substrate with section 4.
4. Bind the Measure to its proto request and manifest declaration.
5. Register it against the same query function used by any equivalent RPC or product view.
6. Probe it and add semantic-parity tests where another consumer already exists.

Do not create a `category` or `kind` switch that recreates a monolithic transport contract. A table or series Measure is valid when it is one coherent analytical question, not an arbitrary response bag.

Waive only a stateful domain with no historical or countable analytical value. A waiver is not a substitute for an inconvenient implementation.

---

### 7. Verification and durable memory

For each changed Measure:

- Run `measures-health validate scenario {{TARGET}} --probe`.
- Verify the declared result shape, unit, provenance, and parameter behavior.
- For a shared historical concept, test semantic parity across its consumer surfaces using the same time window and source snapshot.
- Run the relevant scenario tests through `vrooli scenario test {{TARGET}}`.

Write durable findings:

- `SEAMS.md`: analytical source of truth, read model when present, projection ownership, watermark/freshness semantics, and consumer surfaces.
- `PROBLEMS.md`: deferred substrate migrations, unverified parity, or explicit retirement prerequisites.

Real progress means a scenario can answer a meaningful existing question correctly and reproducibly. Adding declarations, weakly typed parameters, or unjustified waivers does not count as adoption.

---

### 8. Output expectations

You may add or update Measure declarations, proto bindings, Registry compute functions, read-model queries, and parity tests. You may retire a transport-only aggregate contract after the section 5 proof.

You must preserve authoritative historical data and existing analytics-product behavior. You must keep the Measure contract typed, parameterized, provenance-stamped, and safe to auto-execute only when its governance permits it.

Do not add a compatibility shim by default. Keep a retained Stats/product surface when it remains an intentional consumer. Remove only obsolete machine contracts after their consumers and semantics have migrated.

---

### 9. Troubleshooting & Edge Cases

| Symptom | First check | Correct response |
|---|---|---|
| `--probe` is skipped | Target scenario reachability | Start or restart through scenario lifecycle tooling, then re-run the probe. Treat a skipped probe as unverified, not behavioral proof. |
| `/execute` returns 404 | Registry name and manifest binding | Make the Registry name equal `<domain>.<command>`. |
| A Measure and Stats disagree | Definition, time window, unit, and source snapshot | Add a parity test. Move both consumers onto one query function or document intentionally different questions. |
| Historical queries are slow or duplicate event scans | Number of question families and common source facts | Consider an event-sourced analytical read model. Do not add one for a single bounded aggregate. |
| A proposed migration removes a dashboard or workflow | Section 5 classification | Stop the deletion. Treat the surface as an analytics product until a scoped product decision says otherwise. |
| A domain is waived to pass validation | Whether historical/countable value exists | Remove the waiver and implement coverage, or document the concrete non-measurable reason. |
