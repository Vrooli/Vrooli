## Meta focus: Platform Skill Authoring

Guide for creating **platform** skills (the authored skill declares `modes[0] = "platform"`). Platform skills steer safe evolution of shared code (for example `path:packages/*`, shared templates, shared contracts) that is consumed by many scenarios.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `docs/agent-system/PROMOTION_LADDER.md`

Optional reading:
- `prompt-manager skill read cli-steer`
- `prompt-manager skill read conversation-friction-analysis`

---

### 1. Category Scope

**In scope:**
- Shared package evolution (`path:packages/*`) and other shared platform code used by multiple scenarios
- Cross-scenario standardization (CLI output contracts, config precedence, error taxonomy, codegen contracts)
- Brownfield-safe improvement paths (additive changes, deprecation cycles, compatibility envelopes)
- Verification discipline proportional to blast radius (tests + downstream smoke/compat checks)

**Out of scope:**
- Scenario-specific feature design or refactors (use Steer skills; Steer must target `scenarios/{{TARGET}}/`)
- One-off operational runbooks (use Tools skills)
- Skill system governance (use Meta skills)

---

### 2. Required Placeholders and Targeting

Platform skills must not use `{{TARGET}}` (reserved for scenario-focused Steer skills).

Use:
- `{{PACKAGE}}` for a shared package identifier (example: `cli-core`, `api-core`, `proto`)
- `{{PACKAGE_PATH}}` when a path needs to be explicit (example: `packages/{{PACKAGE}}/`)

Optional:
- `{{AREA}}` for subdomains (example: `cli`, `api`, `ui`, `proto`, `runtime`)

---

### 3. Recommended Structure (Keep It Small)

Structure follows `docs/agent-system/SKILL_AUTHORING.md` §"Skill structure" — do not restate it. The platform-specific additions are a **compatibility envelope** (template in §7) and **decision tables** for change classification and layering (§4). Platform skills are enforcement-heavy, not prose-heavy — avoid turning them into “everything about the package”; prefer linking to package docs.

---

### 4. Compatibility-First Convergence Patterns

#### 4.1 Change classification table

| Proposed change | Default decision | Requires explicit opt-in? | Notes |
|---|---|---|---|
| Bugfix (no contract change) | Ship | No | Add regression test |
| Additive API/CLI capability | Ship | No | Prefer new flags/fields/commands |
| Behavior change (same inputs, different outputs) | Avoid | Yes | Requires migration plan + downstream tests |
| Breaking change (removes/renames) | Avoid | Yes | Deprecate first; document timeline |
| New dependency/toolchain | Avoid | Yes | Must justify; avoid forcing scenarios |

#### 4.2 Layering decision tree (where to fix)

```
Is this friction repeated across multiple scenarios?
  -> YES: Prefer package/tool output contract improvement
  -> NO: Prefer scenario-local fix or documentation

Does the skill propose a prose workaround for a missing tool capability?
  -> YES: Add minimal interim guardrail, but file a promotion candidate to package/tooling
```

---

### 5. Verification & Testing Bars (Blast Radius Aware)

Platform skills must require verification proportional to impact:
- Unit tests inside `{{PACKAGE_PATH}}` for logic changes
- Contract tests for user-facing output/behavior (CLI output, JSON schema, config precedence)
- A downstream “compat set”: 1-3 representative scenarios or integration tests that exercise the changed seam

Rule:
- If you can’t name how to verify a change, you don’t have a safe platform change plan yet.

---

### 6. Output Contract Standards (Human-First)

When Platform skills touch CLI behavior (directly or via shared libraries), apply the human-first CLI bar in `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" and the output-contract standards in `cli-steer`; do not restate them. The platform-specific delta: blocking (`--wait`) flows must be progress-observable and return reliable exit codes, because shared libraries set the contract for every consumer CLI.

---

### 7. Compatibility Envelope Template (Copy-Paste)

```markdown
### Compatibility Envelope

- Package: `packages/{{PACKAGE}}/`
- Consumers (known): [list scenarios/packages]
- Supported environments: [Go/Node versions, OS constraints]
- Stability promise: [no breaking changes unless explicitly requested]
- Compat set (must pass): [commands/tests]
```

---

### 8. Output Expectations

You may update:
- Platform skills to improve clarity, compatibility guidance, or verification coverage
- `skill.json` entries for Platform skills

You must:
- Use `{{PACKAGE}}`/`{{PACKAGE_PATH}}` (and optional `{{AREA}}`) — never `{{TARGET}}`, which is reserved for scenario-focused Steer skills
- State the compatibility envelope (template in §7) and the compat-set verification commands that prove a change is safe for downstream consumers
- Prefer human-first CLI output patterns and avoid parser-dependent workflows by default

Registration follows `docs/agent-system/SKILL_AUTHORING.md` §"Registration and metadata"; the authored skill declares `modes[0] = "platform"` and a description that names the shared package surface and the safety posture (compatibility-first, brownfield-safe).

