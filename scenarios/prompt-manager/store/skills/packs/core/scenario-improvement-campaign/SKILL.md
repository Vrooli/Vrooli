## Tools focus: Scenario Improvement Campaign

Drive one scenario toward a chosen goal by turning a test-genie audit into a **tracked, profile-ranked campaign** in architecture-cartographer, then working the ranked worklist to a target. This is the "point an agent at a scenario and say *make it better*" loop. The audit is the camera (stateless detection); the campaign is the project plan (stateful tracking); this skill is the driver.

Use this when the finding load is **too large to fix in one pass and track by hand** — the failure mode that sank the swarm-manager refactor. For a quick one-shot "is this clean?" check, just run the audit; you only need a campaign when you'll work findings over multiple passes.

Required reading:
- `prompt-manager skill read scenario-readiness-review` — the **measure** primitive. Compose it; do not re-implement readiness logic here.
- `prompt-manager skill read screaming-architecture-audit` — the architecture lens whose findings (and others) feed the campaign.

---

### 1. Two orthogonal knobs: profile and target

The single most important idea: **profile and target are independent.**

| Knob | Question it answers | Owned by |
|------|--------------------|----------|
| **profile** | *In what order* do I work the findings? | the tracker (`campaign next --profile`) |
| **target** | *When do I stop?* | you, the driver (this skill) — NOT the tracker |

Never fuse them. You can run a `fast` profile toward a `zero-blockers` target, or a `long-term` profile toward a `zero-findings` target — any combination. The tracker only orders; you decide done.

**Profiles** (the `--profile` flag on `campaign next`):

| Profile | Intent | Ordering |
|---------|--------|----------|
| `fast` | "Make it work now." | Gating sources first (advisory architecture/tidiness findings sink), then severity, then cheapest effort, then fewest locations — the shortest path to a **green suite**. |
| `balanced` *(default)* | The dependable middle. | Regressions first, then import cycles (they block dependent moves), then severity. Preserves the historical ordering. |
| `long-term` | "Best solution." | Regressions, then **structural root-causes first** (cycles, mislocation, structural-cohesion findings — fix the cause before the symptom), then severity. Higher-effort work is *not* deprioritized. |

**Targets** you choose and check after each re-audit (examples — pick what the goal demands):
- `zero-blockers` — no BLOCKER/ERROR open (cheapest "ship it").
- `suite-green` — `vrooli scenario test <s>` passes.
- `zero-findings` — every tracked item validated (the long-term ideal).
- `budget` — stop after N items or a time box, hand off the rest (the campaign persists; resume later).

---

### 2. The loop

```bash
# 1. AUDIT — photograph the scenario. Comprehensive catches every surface;
#    architecture-audit is the structural-only battery.
test-genie execute <scenario> --preset comprehensive --json > audit.json

# 2. OPEN the campaign (ingests every finding; all start `detected`).
architecture-cartographer campaign create "<scenario>" --name "<goal>" --from-audit audit.json
#   → prints the campaign id; capture it.

# 3. NEXT — pull the ranked worklist for your chosen profile.
architecture-cartographer campaign next "<campaign-id>" --profile fast    # or balanced / long-term

# 4. WORK the top item(s) by hand. Then record each:
architecture-cartographer campaign resolve "<campaign-id>" --finding "<afid>" --note "what you did"

# 5. RE-AUDIT and reconcile: gone → validated, persists → still open,
#    (re)appeared or brand-new → REGRESSION (handle these first).
test-genie execute <scenario> --preset comprehensive --json > audit-2.json
architecture-cartographer campaign reaudit "<campaign-id>" --from-audit audit-2.json

# 6. CHECK your target. Not met? → back to step 3 with the next audit.
#    Met? → close.
architecture-cartographer campaign status "<campaign-id>"
architecture-cartographer campaign close "<campaign-id>"
```

Between iterations, use **`scenario-readiness-review`** to judge whether the changes you made are coherent and commit-ready — that is your measure step; this skill does not duplicate its logic.

---

### 3. Reading the worklist

Each item line shows `afid  source/code  [severity]  locations  {effort}  → status`. Three signals drive your choice:
- **REGRESSED** items lead every profile — a fix that didn't hold, or new breakage you introduced. Fix these before new work.
- **effort** (`{trivial|small|medium|large}`) is advisory and starts as a coarse per-source heuristic — trust it for relative ordering under `fast`, not as an estimate to quote.
- **source** tells you where the fix lives: `architecture`/`tidiness` are advisory (never fail the suite); `standards`/`structure`/`cli`/`ui`/`docs` gate it. `fast` exploits exactly this.

---

### 4. One skill, profile is an argument

There is deliberately **one** campaign skill, not one per profile. The profile is a runtime argument to `campaign next`; re-rank freely (`--profile fast` then `--profile long-term`) without recreating the campaign. If you find yourself wanting a "fast-campaign skill" and a "long-term-campaign skill," that is the skill-compression anti-pattern — the difference is a flag, not a workflow.

---

### 5. When NOT to use

| Use a campaign when | Skip it when |
|---------------------|--------------|
| Findings outnumber a single responsible pass | A clean or near-clean audit (just fix the few inline) |
| You'll work over multiple re-audit passes | A one-shot "is this clean?" gate (run the audit, read it) |
| You need regression detection across passes | You're committing (human responsibility — see readiness-review) |
| You want goal-aware ordering (fast vs long-term) | Debugging one specific bug (use scientific-debugging) |

Use CLI human-default output everywhere; never call the cartographer or test-genie HTTP APIs directly.
