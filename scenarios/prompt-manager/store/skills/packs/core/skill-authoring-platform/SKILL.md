## Meta focus: Platform Skill Authoring

Guide for creating **Platform** skills (where `modes[0] = "platform"`). Platform skills steer safe evolution of shared code (for example `path:packages/*`, shared templates, shared contracts) that is consumed by many scenarios.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`

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

Platform skills should be enforcement-heavy, not prose-heavy:
1. **Intent statement** (1-2 sentences)
2. **Scope boundaries** (explicit blast-radius constraints)
3. **Compatibility envelope** (template; see below)
4. **Decision tables** (change classification + layering)
5. **Verification requirements** (tests + downstream “compat set”)
6. **Output expectations** (what may/must/must not change)
7. **Troubleshooting & Edge Cases** (only if operational complexity exists)

Avoid turning platform skills into “everything about the package.” Prefer linking to package docs.

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

When Platform skills touch CLI behavior (directly or via shared libraries):
- Default output must be human-first and action-guiding
- Blocking (`--wait`) flows must be progress-observable and return reliable exit codes
- Errors must contain next-step guidance (a command or a precise recovery hint)
- Machine output (`--json`) may exist, but must not be required for basic operation

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

### 8. Registration Notes

Follow `docs/agent-system/SKILL_AUTHORING.md` and ensure:
- `modes[0]` is `platform`
- The description names the shared package surface and the safety posture (compatibility-first, brownfield-safe)

