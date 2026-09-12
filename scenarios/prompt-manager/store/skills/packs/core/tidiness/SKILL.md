---
name: "tidiness"
description: "Maintainability cleanup, long-file and complexity debt, issue queues, and cleanup campaigns"
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","tidiness","audits"]
  tags: ["skill","audit-technique","maintainability"]
  icon: "list-checks"
  status: "active"
  targetDimensions: ["tidiness"]
  targetToolId: "run-agent"
  programmaticHome: "test-genie:tidiness"
  revision: 1
  createdAt: "2026-06-15T00:00:00Z"
  updatedAt: "2026-06-15T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "test-genie"]
    commands: ["prompt-manager", "prompt-manager skill", "prompt-manager skill read", "test-genie execute"]
  origin:
    kind: "authored"
---
## Steer focus: Tidiness

Prioritize **making `scenarios/{{TARGET}}/` easier to maintain by reducing file/function bloat, duplication, technical-debt markers, stale cleanup queues, and campaign drift**. Use Tidiness Manager for maintainability findings; leave lint, type-safety, and strict config policy to Quality Health.

Required reading:
- `prompt-manager skill read scenario-maturity-ladder improvement-do-and-dont`
- `scenarios/tidiness-manager/README.md` - active Tidiness Manager scope and CLI entrypoints.
- `scenarios/tidiness-manager/PRD.md` - maintainability and campaign goals.

Read first when present:
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`
- `scenarios/{{TARGET}}/docs/internal/PROGRESS.md`
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md`
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md`

---

### 0. Programmatic Validation - run this first

Start with the maintainability scan:

```bash
tidiness-manager scan {{TARGET}} --type tidiness
```

Use JSON when you need stable evidence for a handoff or tool consumer:

```bash
tidiness-manager scan {{TARGET}} --type tidiness --json
tidiness-manager score {{TARGET}} --json
tidiness-manager issues list --scenario {{TARGET}} --json
test-genie execute {{TARGET}} --phases tidiness --json
```

Use recommendations to choose focused cleanup candidates:

```bash
tidiness-manager recommend-refactors {{TARGET}} --limit 10
```

The scan output is the source of truth for this skill. Manual judgment decides which cleanup is worth doing now and which should be queued or documented.

---

### 1. Scope Boundaries

**In scope:**
- long files and oversized functions
- complexity hotspots, coupling, duplication, and repeated logic
- TODO/FIXME/HACK debt that blocks maintainability
- issue queue hygiene: list, resolve, ignore with notes, reopen
- cleanup campaigns and visited-tracker-informed refactor sequencing
- durable scenario docs for deferred cleanup debt

**Out of scope:**
- lint/type/static-quality contracts, strict config comments, and suppressions; hand off to `quality-health`
- architecture-level domain relocation and import-cycle design; hand off to `screaming-architecture-audit`
- test coverage or assertion quality; hand off to `test`
- visual polish or copy refinement unless it removes actual maintenance risk; hand off to `polish`
- large feature work that only happens to touch messy files; hand off to the relevant feature or scenario-maturity skill

---

### 2. Tidiness Maturity Model

Assess maturity by observable scan/campaign artifacts.

| Level | Name | What exists | When to stop here |
|---|---|---|---|
| 0 | No tidiness scan | `tidiness-manager scan {{TARGET}} --type tidiness` cannot run or has no meaningful target. | Stop only to restore scan availability. |
| 1 | Basic inventory | The scan reports file-size and technical-debt marker findings for the scenario. | Stop when obvious file/TODO inventory exists. |
| 2 | Metrics surfaced | Complexity, duplication, coupling, and score breakdowns are available through scan/score output. | Stop when hotspots are visible but not yet organized into work. |
| 3 | Issue queue usable | `tidiness-manager issues list --scenario {{TARGET}}` shows normalized issues with severity, category, file, and remediation. | Stop when agents can pick and close focused cleanup items. |
| 4 | Campaign workflow active | Campaigns or recommendation workflows can sequence cleanup and track resolved/ignored/reopened items. | Stop when larger cleanup can be managed without losing state. |
| 5 | Drift-gated | Test Genie's `tidiness` phase is green or has understood residual findings documented in scenario docs. | Stop when maintainability no longer blocks the tidiness dimension. |

---

### 3. Decision Table

Walk findings in this order.

| Signal | Primary action | Handoff |
|---|---|---|
| Long file with mixed responsibilities | Split by existing domain/feature boundaries, preserving behavior and tests. | `screaming-architecture-audit` if ownership is unclear. |
| Complex function with clear sub-steps | Extract named helpers inside the owning domain; add or preserve tests for behavior. | `test` if no protective coverage exists. |
| Duplication across one domain | Consolidate into a domain-local helper or type. | None |
| Duplication across unrelated domains | Prefer leaving local unless a real shared abstraction already exists or the pattern is stable. | `screaming-architecture-audit` for shared-boundary decisions. |
| TODO/FIXME/HACK cluster | Resolve if small and current; otherwise convert to durable docs or issue notes with owner/context. | Relevant feature skill if the marker represents missing behavior. |
| Open issue is stale after cleanup | Re-run scan, then resolve/ignore/reopen with notes. | None |
| Finding is actually lint/type policy | Do not fix here; reroute to `quality-health`. | `quality-health` |

---

### 4. Workflow

1. Run the tidiness scan and read high-severity findings first.
2. Check score and issue queue when deciding whether this is a quick cleanup or a campaign:

```bash
tidiness-manager score {{TARGET}}
tidiness-manager issues list --scenario {{TARGET}}
```

3. Choose the smallest cleanup that materially reduces maintenance risk.
4. Keep edits within the owning scenario and domain unless the architecture docs justify a wider move.
5. For multi-file cleanup, use recommendations or campaigns to avoid losing state:

```bash
tidiness-manager recommend-refactors {{TARGET}} --limit 10
tidiness-manager campaigns start {{TARGET}} --max-sessions 3 --files-per-session 10
```

6. After cleanup, rerun:

```bash
tidiness-manager scan {{TARGET}} --type tidiness
test-genie execute {{TARGET}} --phases tidiness --json
```

7. Update durable docs:
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` for accepted residual cleanup debt.
- `scenarios/{{TARGET}}/docs/internal/PROGRESS.md` when a cleanup campaign or notable hotspot is resolved.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` only when cleanup changes a real seam.

Do not create standalone tidiness audit reports by default; use the issue queue, campaigns, and scenario docs.

---

### 5. Troubleshooting & Edge Cases

| Symptom | First check | Likely cause | Fix |
|---|---|---|---|
| `tidiness-manager scan {{TARGET}} --type tidiness` cannot connect | `cd scenarios/tidiness-manager && make status` | Tidiness Manager is stopped or unhealthy. | Start it through lifecycle with `cd scenarios/tidiness-manager && make start`, then rerun. |
| Scan reports static-quality/lint/type findings | Confirm the command used `--type tidiness`. | Old light/type-safety path or stale docs were used. | Re-run the tidiness scan; route static-quality work to Quality Health. |
| Issue queue seems stale | `tidiness-manager issues list --scenario {{TARGET}}` retrieval hints | Stored issues predate current files. | Re-run scan, then resolve/ignore/reopen stale issues with notes. |
| Recommendation command returns no candidates | Check whether scan data exists and whether filters are too strict. | Metrics have not been seeded or the scenario is already below thresholds. | Run `tidiness-manager scan {{TARGET}} --type tidiness` and retry with a broader limit. |
| Cleanup requires broad domain relocation | Read `ARCHITECTURE.md` and `SEAMS.md`. | This is architecture drift, not simple tidiness. | Load `screaming-architecture-audit` before moving ownership boundaries. |

Repeated troubleshooting should become Tidiness Manager CLI output or a prompt-manager Action before this skill grows more operational prose.

---

### **6. Output Expectations**

You may update scenario implementation files, focused tests, issue statuses, campaign state, and durable scenario docs.

You must:
- keep behavior and tests intact while reducing maintenance risk
- avoid using tidiness cleanup as a cover for unrelated feature work
- rerun Tidiness Manager and Test Genie tidiness validation after meaningful cleanup
- resolve, ignore, or document stale issue-queue entries with clear notes

Avoid:
- creating premature shared abstractions from coincidental duplication
- moving files across domain boundaries without architecture justification
- treating Quality Health findings as tidiness work
- deleting TODO/FIXME/HACK markers without resolving or preserving the underlying context
