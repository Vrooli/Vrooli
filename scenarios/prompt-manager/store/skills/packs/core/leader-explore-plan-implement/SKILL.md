## Practice focus: Leader Explore-Plan-Implement Pipeline

Orchestrate the full lifecycle of delegated technical work: explore the problem space, author an implementation plan, then coordinate implementation across team members. This is a three-phase gated pipeline over two leaf skills.

Required reading:
- `prompt-manager skill read team-coordination-leader-led` — the shared gated-pipeline contract (phase shape, gate/rework semantics, delegation template, convergence checklists, shared anti-patterns). This skill adds only what is specific to explore-plan-implement.
- `prompt-manager skill read systematic-exploration` — the Explore-phase leaf.
- `prompt-manager skill read implementation-plan-authoring` — the Plan-phase leaf.

---

### 1. Phase-to-leaf mapping

| Phase | Leaf skill / activity | Artifact |
|---|---|---|
| Explore | `systematic-exploration` | Exploration findings report |
| Plan | `implementation-plan-authoring` | Finalized Plan Manager plan id/slug + rendered review |
| Implement | Coordinate and delegate against the plan | Implemented code, tests, docs per plan |

Run the phases per the shared contract's phase shape. Enter at any phase whose inputs already exist (see the entry table).

### 2. Pipeline entry decision table

| You have... | Area understood? | Plan exists? | Entry point |
|---|---|---|---|
| New work request, unfamiliar area | No | No | Phase 1: Explore |
| New work request, familiar area | Yes | No | Phase 2: Plan |
| Existing plan from prior session | Yes | Yes | Phase 3: Implement |
| Bug with known cause | Yes | N/A | Not this pipeline — use `scientific-debugging` |
| Creative exploration request | N/A | N/A | Not this pipeline — use the `explore` steer skill |
| Single-agent work, no delegation | N/A | N/A | Not this pipeline — use the leaf skills directly |

### 3. Gate criteria

**Gate 1 (Explore → Plan):** findings report exists and answers the investigation question; key components, patterns, and constraints are documented; another agent could plan from the findings alone; no critical unknowns remain (or each is labeled with its risk).

**Gate 2 (Plan → Implement):** Plan Manager validation passes; the plan respects every constraint from exploration; acceptance criteria are objective (pass/fail, not narrative); phases are ordered with explicit dependencies; another agent could implement from the plan alone.

**Plan readiness** (from `implementation-plan-authoring`): a finalized Plan Manager plan with a stable id/slug; explicit scope, non-goals, constraints, compatibility posture; relevant skills/docs accepted as `relevant_context` (or a recorded no-context rationale); code references accepted (or an explicit `NO_CODE_REFS` rationale); phases with ordered steps, validation, acceptance criteria, and implementation context.

### 4. Rework triggers

| Signal | During phase | Action |
|---|---|---|
| "I don't understand how X works" | Plan | Return to Explore, scope X specifically |
| "The plan assumes X but code does Y" | Implement | Return to Plan, update assumptions |
| "I found a constraint not in the plan" | Implement | Return to Explore+Plan if systemic; update the plan if localized |
| "Requirements conflict with code reality" | Plan or Implement | Escalate to the work requester |
| "Tests reveal unexpected behavior" | Implement | Return to Explore if the behavior is not understood; fix the plan if it is |

### 5. Boundaries and output

Covers leader-coordinated technical work flowing through exploration, planning, and implementation. Does not cover: single-agent work (use leaf skills directly), debugging (`scientific-debugging`), creative exploration (`explore`), operations/deployment, or strategic prioritization (decided upstream).

You must produce the three phase artifacts in the mapping table (or verify a pre-existing one at partial entry). Record which phases ran and any rework loops — repeated Implement→Explore loops mean the Explore phase was insufficient, which is the signal to front-load more understanding next time.
