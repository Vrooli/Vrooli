## Steer focus: Proto Contract Audit

Prioritize **making `scenarios/{{TARGET}}/` use Protocol Buffer contracts that are domain-organized, generated, adopted, and transport-clear**. Audit proto structure first, then fix schema placement, generated artifacts, and API/CLI/UI consumption without reaching into fleet-wide dependency graph work.

Required reading:
- `packages/proto/STYLE_GUIDE.md` — canonical domain organization, version directory, shared bucket, and annotation registry.
- `prompt-manager skill read interoperability-steer` — deep schema, serialization, discovery, and runtime boundary rules.
- `prompt-manager skill read api-steer` — API transport decisions and REST exception discipline.
- `prompt-manager skill read screaming-architecture-audit` — domain vocabulary alignment for scenario code shape.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md`
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md`
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`

---

### 0. Provider-Owned Maturity — run this first

Take the proto-health photograph before doing manual judgment. The default human output is the single source of truth for current local maturity, next level, blockers, global impact grouping, and recommended skill IDs:

```bash
proto-health validate scenario {{TARGET}}
```

Use `--json` only for automation or parser debugging. When running through the full quality loop, use the test-genie phase so findings are normalized into `FINDING_SOURCE_PROTO`:

```bash
test-genie execute {{TARGET}} proto --json
```

Use these results to prioritize fixes. The manual workflow below is for interpreting and resolving findings, not duplicating proto-health's deterministic checks or its `.vrooli/maturity.json` ladder.

---

### 1. Scope Boundaries

**In scope:**
- scenario-owned proto files under `packages/proto/schemas/{{TARGET}}/`
- package/folder/version naming, `v<n>/shared/`, and domain folder organization
- generated artifact sync under `packages/proto/gen/`
- scenario adoption of generated Go/TypeScript/Python artifacts
- Connect-vs-hand-rolled transport signal inside `scenarios/{{TARGET}}/`
- durable notes in `ARCHITECTURE.md`, `SEAMS.md`, and `PROBLEMS.md`

**Out of scope:**
- fleet import graph, dependency drift, dead-proto reachability across scenarios, and cross-scenario encapsulation rules; hand off to scenario-dependency-analyzer work
- API resource modeling and REST exception design; hand off to `api-steer`
- broad interop, runtime recovery, service discovery, and envelope semantics; hand off to `interoperability-steer`
- code/domain relocation beyond what is required to make proto ownership clear; hand off to `screaming-architecture-audit`

---

### 2. Provider Findings

Assess the scenario from provider findings, not by intent or a duplicated prose ladder. Walk the `proto-health validate scenario {{TARGET}}` output in order: fix ERROR findings first, document accepted warnings, and rerun the provider CLI after meaningful changes.

The provider owns current/next maturity. If that maturity output is inaccurate, update proto-health or its maturity spec instead of patching this skill.

---

### 3. Decision Table

Walk these rows in order.

| Signal | Primary action | Handoff |
|---|---|---|
| `proto.gen_out_of_sync` | Run `cd packages/proto && make generate`, then re-run `make verify-committed-gen` when preparing the commit/CI gate. | None |
| `proto.package_mismatch` or `proto.version_naming` | Move/rename schema files to match `packages/proto/STYLE_GUIDE.md`; regenerate. | None |
| `proto.cross_domain_import` | Move shared message(s) into `v1/shared/` or reconsider the domain boundary. | `screaming-architecture-audit` if code domains are also blurred |
| `proto.unsupported_annotation` | Remove deprecated `@layer`, `@domain`, `@imports`, or unsupported tags; keep only registry-backed annotations. | None |
| `proto.hand_rolled_transport` | Migrate proto-owned calls to generated Connect handlers unless a REST exception applies. | `api-steer` for exception review |
| `proto.stability_dishonest` | Align `@stability` with implementation and serving reality. | None |
| `proto.template_source` | Keep the marker while the file is scaffold reference code; remove it only when the contract is intentionally adopted or replaced. | Template maintenance if new generated scenarios inherit the wrong marker |
| `proto.possibly_unused` | Treat as advisory; confirm locally and document if intentionally reserved. | scenario-dependency-analyzer for fleet-aware dead-proto decisions |

Generated-contract adoption is still part of proto-health's provider-owned maturity.
Automated proof belongs to the future `surface-code-facts` layer, not to a
declaration-only proto-health finding.

---

### 4. Workflow

1. Run the programmatic validation and read all ERROR findings first.
2. Inspect `packages/proto/schemas/{{TARGET}}/` and classify each file by version and domain.
3. Confirm each served API method has a generated Connect handler or an explicit REST exception.
4. Confirm API, CLI, and UI use generated contracts where the wire shape is proto-owned.
5. Regenerate artifacts after schema edits:

```bash
cd packages/proto
make generate
make verify-committed-gen
```

6. Update durable docs:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` for proto/domain ownership and served surfaces.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` for descriptor, generated-client, or transport seams.
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` for deferred warnings.

---

### 5. Troubleshooting & Edge Cases

| Symptom | First check | Likely cause | Fix |
|---|---|---|---|
| `proto-health validate scenario {{TARGET}}` cannot find or reach proto-health | `cd scenarios/proto-health && make status` | The producer scenario is stopped or unhealthy. | Start it through lifecycle with `cd scenarios/proto-health && make start`, then rerun validation. |
| `proto.gen_out_of_sync` appears after schema edits | `cd packages/proto && make generate` | Generated artifacts under `packages/proto/gen/` do not match `packages/proto/schemas/`. | Regenerate, inspect the scoped `gen/` diff, and rerun `proto-health validate scenario {{TARGET}}`. |
| `cd packages/proto && make verify-committed-gen` fails while schema/generated changes are intentionally uncommitted | `git diff --stat -- packages/proto/gen packages/proto/schemas` | The check compares generated artifacts against the git index; it is a commit/CI gate, not a substitute for reviewing uncommitted generated diffs. | Confirm `make generate` is idempotent, include the generated artifacts in the same commit as schema changes, then rerun `make verify-committed-gen` from a clean or staged state. |
| Only `proto.possibly_unused` or other WARNING/INFO findings remain | Inspect `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` | The scenario may be intentionally carrying advisory migration debt. | Document the reason and owner in durable docs; do not turn warning-tier maturity findings into blockers without a plan update. |

Keep this section short. If the same failure recurs across scenarios, prefer improving `proto-health` CLI output or creating a prompt-manager Action over expanding this prose.

---

### 6. Output Expectations

You may update schema files, generated artifacts, scenario API/CLI/UI consumers, and scenario docs.

You must:
- keep proto-health's scope scenario-local
- preserve generated-code provenance by editing schemas, not `gen/` directly
- run `proto-health validate scenario {{TARGET}}` after meaningful changes
- write unresolved proto warnings to durable scenario docs rather than a standalone audit report

Avoid:
- reintroducing `@layer` or layer-number reasoning
- creating compatibility shims for greenfield proto contracts
- turning warning-tier maturity findings into hard blockers without a plan update
