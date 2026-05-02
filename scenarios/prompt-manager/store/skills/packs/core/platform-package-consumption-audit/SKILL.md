## Platform focus: Package Consumption Audit

Analyze how scenarios consume `packages/{{PACKAGE}}/`, identify repeated friction and contract drift, and produce a prioritized improvement backlog that makes the package more reliable and easier to use without breaking downstream consumers.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read platform-scope`

Optional reading:
- `prompt-manager skill read conversation-friction-analysis`
- `prompt-manager skill read cli-steer`

---

### 1. Scope Boundaries

**In scope:**
- Mapping consumers of `packages/{{PACKAGE}}/` across scenarios and packages
- Identifying inconsistencies in usage patterns, configuration, and output/behavior expectations
- Finding common failure modes and “paper cuts” (missing docs, unclear errors, non-observable progress, brittle output)
- Proposing package-level improvements (output contracts, small helper APIs, safer defaults, tests)

**Out of scope:**
- Large-scale rewrites or dependency/toolchain changes
- Mass-updating many scenarios “to match the new way” (prefer compatibility + gradual migration)
- Breaking changes without explicit request and deprecation plan

---

### 2. Inputs

- `{{PACKAGE}}`: package identifier (example: `cli-core`, `api-core`, `proto`)
- Optional `{{AREA}}`: `cli`, `api`, `ui`, `proto`, `runtime`

Derived:
- `{{PACKAGE_PATH}} = packages/{{PACKAGE}}/`

---

### 3. Audit Workflow

#### Step A: Find consumers (cross-language)

Run broad discovery first, then narrow.

```bash
# Monorepo path references
rg -n "packages/{{PACKAGE}}/" -S .

# Go imports (common patterns)
rg -n "github\\.com/vrooli/{{PACKAGE}}|vrooli/.*/packages/{{PACKAGE}}" -S scenarios packages

# TS/JS workspace imports (common patterns)
rg -n "from\\s+['\\\"](@vrooli/{{PACKAGE}}|{{PACKAGE}}|\\.\\./\\.\\./packages/{{PACKAGE}}/)" -S scenarios packages
```

Produce a consumer list grouped by:
- scenarios
- packages
- language/runtime (Go/TS/etc.)
- usage type (`cli`, `api`, `ui`, build scripts)

#### Step B: Identify usage clusters (convergence vs divergence)

For each cluster, answer:
- What is the intended contract?
- How do consumers currently do it?
- Where do they diverge?
- What breaks (or causes retries) when things change?

Use a table:

| Seam | Consumers | Intended contract | Observed patterns | Friction | Likely fix layer |
|---|---|---|---|---|---|
| config precedence | ... | ... | ... | ... | package / docs |
| output contract | ... | ... | ... | ... | package |
| error handling | ... | ... | ... | ... | package |

#### Step C: Extract friction as promotion candidates

When you find repeated workaround prose (or repeated “here’s how to interpret this output”), treat it as a package/tool contract gap.

Record:
- symptom
- current workaround
- missing contract/capability
- proposed durable fix
- verification method

#### Step D: Propose improvements (compatibility-first)

For each proposed improvement:
- classify change type (bugfix/additive/behavior change/breaking)
- list impacted consumers
- define verification (unit tests + minimal downstream compat set)
- define migration (if any)

---

### 4. Output Expectations

**Must produce:**
- A written audit artifact at `docs/internal/platform/{{PACKAGE}}/consumption-audit.md` using the template below
- A prioritized backlog of proposed package improvements with verification plans

**Must not:**
- Implement breaking changes as part of the audit
- Add new dependencies without explicit permission

---

### 5. Audit Artifact Template (Copy-Paste)

```markdown
# Package Consumption Audit: {{PACKAGE}}

## Compatibility Envelope
- Package: `packages/{{PACKAGE}}/`
- Consumers (known): ...
- Supported environments: ...
- Stability promise: no breaking changes by default

## Consumer Inventory
| Consumer | Type | Language | Notes |
|---|---|---|---|
| scenarios/... | scenario | go/ts | ... |

## Seam Map
| Seam | Intended Contract | Observed Patterns | Friction | Fix Layer |
|---|---|---|---|---|
| ... | ... | ... | ... | package/docs/scenario |

## Findings (Prioritized)
### P0
- ...
### P1
- ...

## Promotion Candidates (Durable Fixes)
| Workaround | Promote To | Expected Benefit | Verification |
|---|---|---|---|
| ... | package API/output contract | ... | ... |

## Proposed Compat Set
| Consumer | Command/Test | Pass criteria |
|---|---|---|
| ... | ... | ... |
```

