---
name: "platform-package-hardening"
description: "Improve a shared package’s reliability, tests, docs, and output contracts without breaking downstream consumers."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["platform","hardening"]
  tags: ["platform","packages","testing","docs","compatibility"]
  status: "active"
  defaultScope: "platform-scope"
  revision: 1
  createdAt: "2026-02-10T00:00:00Z"
  updatedAt: "2026-02-10T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Platform focus: Package Hardening

Harden `packages/{{PACKAGE}}/` for reliability and usability: stabilize contracts, improve default human output (when relevant), add tests and docs, and prove changes via a downstream compat set. Default posture is brownfield-safe and non-breaking.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `prompt-manager skill read platform-scope`

Optional reading:
- `prompt-manager skill read platform-package-consumption-audit`
- `prompt-manager skill read cli-steer`

---

### 1. Scope Boundaries

**In scope:**
- Reliability fixes, clearer errors, safer defaults
- Contract hardening (documented behavior, output contracts, config precedence rules)
- Test improvements (unit tests + contract tests + minimal downstream compat checks)
- Documentation improvements scoped to the package (`README`, `docs/`, examples)

**Out of scope:**
- Breaking changes without explicit request and deprecation plan
- New dependencies/toolchains without explicit permission
- Refactors that don’t pay for themselves in stability or testability

---

### 2. Inputs

- `{{PACKAGE}}`
- Optional `{{AREA}}` (example: `cli`, `api`, `ui`, `proto`, `runtime`)

Derived:
- `{{PACKAGE_PATH}} = packages/{{PACKAGE}}/`

---

### 3. Hardening Workflow (Compatibility-First)

#### Step A: Establish the compatibility envelope

Write down:
- known consumers
- supported environments
- stability promise
- the downstream compat set you will run

Use the template from `platform-package-consumption-audit` (or create a minimal one).

#### Step B: Identify the contract seam(s) you are hardening

Examples:
- output contract (human-first; progress-observable blocking flows)
- config precedence (env vs config file vs defaults)
- error taxonomy (actionable hints vs opaque errors)
- versioning/deprecation behavior

Rule:
- If the seam is user-facing, you must add a test (golden/snapshot/contract test) that locks it down.

#### Step C: Implement minimal, additive improvements

Preferred tactics:
- additive fields/options with safe defaults
- improved default output without requiring `--json`
- better error wrapping + next-step hints
- “blocking mode” progress visibility and reliable exit codes

Avoid:
- renaming exported APIs/flags/fields
- changing defaults in a way that surprises existing consumers

#### Step D: Verification (required)

Minimum:
- run package unit tests
- run package lint/format checks if the package already has them
- run a downstream compat set (1-3 representative consumers)

If the package influences CLI behavior:
- verify default output remains human-first and action-guiding
- verify `--json` remains available (if present) but optional

#### Step E: Document the contract

Update package docs to include:
- the stable contract (inputs/outputs)
- common failure modes and first checks
- examples that do not require parsing pipelines

---

### **4. Output Expectations**

**Must produce:**
- Improved test coverage for the hardened seam(s)
- Documentation updates in `{{PACKAGE_PATH}}` (or adjacent canonical docs if they already exist)
- A recorded downstream compat set (commands + pass criteria)

**Must not:**
- Introduce breaking changes without a deprecation plan and explicit approval
- Add dependencies without explicit permission

