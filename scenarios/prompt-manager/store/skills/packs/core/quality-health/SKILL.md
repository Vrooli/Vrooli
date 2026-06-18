## Steer focus: Quality Health

Prioritize **making `scenarios/{{TARGET}}/` statically safe, strict, and resistant to quality-contract drift** — use Quality Health as the authority for lint, type-safety, config strictness, suppressions, and deterministic config repair, rather than loosening local tool settings to make failures disappear.

Your goal is to fix the source and tool wiring behind each finding, not to suppress or weaken the contract that surfaced it.

Required reading:
- `prompt-manager skill read scenario-maturity-ladder improvement-do-and-dont`

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`
- `scenarios/{{TARGET}}/docs/internal/PROGRESS.md`
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md`

Reference on demand — the audit output already embeds why-it-matters, remediation, and the autofix command per finding:
- `scenarios/quality-health/docs/reference/quality-contracts.md`, `finding-schema.md`, `cli-commands.md`

---

### 0. Provider-Owned Maturity — run this first

Take the Quality Health photograph before any manual judgment. The default human output is the single source of truth for current local maturity, next level, blockers, autofix candidates, and recommended skill IDs:

```bash
quality-health audit run {{TARGET}} --commands --autofix-preview
```

Use `--json` only for automation, a handoff, or parser debugging. When running the full quality loop, use the test-genie phase so findings normalize into `FINDING_SOURCE_STANDARDS`:

```bash
test-genie execute {{TARGET}} --phases quality --json
```

Use these results to prioritize fixes. The workflow below is for interpreting and resolving findings, not for duplicating Quality Health's deterministic checks or its `.vrooli/maturity.json` ladder.

---

### 1. Scope Boundaries

**In scope:**
- lint and type-check command presence, wiring, and behavior
- TypeScript, ESLint, Go lint, Makefile, and `.vrooli/testing.json` static-quality contracts
- protective comments that tell future agents not to weaken validation
- suppression/weakening patterns: `as any`, `@ts-ignore`, non-null assertions, bare `//nolint`, disabled strict rules
- deterministic config autofix previews and reviewed applications
- durable notes in `scenarios/{{TARGET}}/docs/internal/` when findings are intentionally deferred

**Out of scope:**
- unit-test framework setup, assertion strength, coverage, and flake; hand off to `test`
- file/function length, TODO density, duplication, and cleanup campaigns; hand off to `tidiness`
- broad architecture/file-placement cleanup; hand off to `screaming-architecture-audit`
- security vulnerabilities and secrets; hand off to `security`
- weakening a quality contract because a local failure is inconvenient

---

### 2. Provider Findings

Assess the scenario from provider findings, not by intent or a duplicated prose ladder. Walk the audit output in order: fix ERROR findings before WARNING findings, document accepted warnings, and rerun the audit after meaningful changes.

The provider owns current/next maturity and the recommended skill IDs per finding. If that output is inaccurate, update Quality Health or its maturity spec (`scenarios/quality-health/.vrooli/maturity.json`) instead of patching this skill.

---

### 3. Decision Table

Walk these rows in order. Two findings can be true at once: clear the mechanical (autofixable) bucket first, then the source-judgment bucket.

| Signal | Primary action | Handoff |
|---|---|---|
| Audit cannot discover surfaces | Check scenario metadata, Code Facts availability, and orientation docs; restore discovery before judging contracts. | `screaming-architecture-audit` if the physical layout is unclear |
| Finding reports `autofix_available: true` | Preview with `quality-health fix-config run {{TARGET}} --dry-run`, review the diff, then apply; do not hand-edit what the fixer owns. | None |
| Lint/type command missing or skipped without reason | Add or repair the command in the owning surface's package scripts or Makefile using existing scenario patterns. | `test` if the real gap is unit-test execution, not lint/type |
| Strict config or protective comment missing/weakened (`TS_CONFIG_STRICT`, `ESLINT_*`, `NODE_BUILD_TYPECHECK`, `GO_LINT_*`) | Restore the contract via autofix, then fix the underlying code that motivated the weakening. | `react-coherence` for UI typing/hook fixes; `test` for build/test-gate wiring |
| Dangerous-pattern cluster (`TS_DANGEROUS_PATTERNS`, `GO_DANGEROUS_PATTERNS`; `autofix_available: false`) | Replace unsafe assertions/suppressions with validated types, narrower guards, or a written `//nolint` reason; document any intentional residue. | `react-coherence` for UI type ambiguity |
| `QUALITY_COVERAGE_GAP` (info) | A surface's language has no contract pack; record it as known coverage debt. Never treat it as a pass. | `ecosystem-fit` |
| Audit and a local tool disagree | Trust the audit for contract status; inspect local command output to fix the implementation error. | None |

---

### 4. Workflow

1. Run the audit (§0) and read ERROR findings before WARNING findings.
2. **Mechanical pass** — for every `autofix_available` finding, follow its `autofix_command` (preview → review the diff → apply). One pass clears the deterministic config bucket; do not hand-edit config the fixer owns.
3. **Judgment pass** — re-run the audit; whatever survives is real source debt. Fix the code or wiring per §3, preferring fixes over suppressions.
4. Re-run the audit and the test-genie `quality` phase to confirm green:

```bash
quality-health audit run {{TARGET}} --commands --autofix-preview
test-genie execute {{TARGET}} --phases quality --json
```

5. Update durable docs only for meaningful residual debt:
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` for accepted warnings or deferred fixes.
- `scenarios/{{TARGET}}/docs/internal/PROGRESS.md` when quality hardening completes a tracked item.

Do not create standalone quality audit reports; the scenario docs and audit output are the durable surfaces.

---

### 5. Troubleshooting & Edge Cases

| Symptom | First check | Likely cause | Fix |
|---|---|---|---|
| `quality-health audit run {{TARGET}}` cannot connect | `cd scenarios/quality-health && make status` | Quality Health is stopped or unhealthy. | `cd scenarios/quality-health && make start`, then rerun the audit. |
| Audit reports no surfaces | Code Facts health and `scenarios/{{TARGET}}/.vrooli/service.json` | Discovery substrate or scenario metadata is missing/stale. | Restore orientation metadata and rerun. |
| `fix-config` preview is empty for a finding | The finding is `autofix_available: false`. | Only deterministic config fixes are supported. | Fix manually using the finding's remediation. |
| A strict rule breaks existing code | Local code relied on unsafe types or disabled checks. | The contract is exposing real debt. | Fix the code path; do not weaken config or remove protective comments. |
| test-genie `quality` differs from a direct audit | Compare scenario name, instance, and command flags. | Different execution context or stale provider. | Restart affected scenarios through lifecycle and rerun the direct audit first. |

Keep this section short. If the same failure recurs across scenarios, prefer improving Quality Health CLI output or creating a prompt-manager Action over expanding this prose.

---

### 6. Output Expectations

You may update scenario tool configuration, package scripts, Makefiles, strictness-preserving source code, quality-policy tests, and durable scenario docs.

Beyond the universal authoring and anti-gaming bars in `docs/agent-system/SKILL_AUTHORING.md`, this dimension's specific weakening shapes are forbidden as a way to get green:
- broad `eslint-disable`, `@ts-ignore`/`@ts-expect-error`, `as any`, or non-null `!` assertions
- bare `//nolint` without a written reason
- deleting suppressions without understanding the behavior they mask
- removing protective comments that a contract requires

Always rerun Quality Health and the test-genie `quality` phase after meaningful changes, and record accepted residual warnings in durable scenario docs.
