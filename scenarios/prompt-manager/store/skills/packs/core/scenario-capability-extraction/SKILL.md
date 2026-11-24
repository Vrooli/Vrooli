---
name: "scenario-capability-extraction"
description: "Methodology for extracting a mature product capability from one or more existing scenarios into a dedicated reusable Vrooli scenario with API, CLI, UI, docs, requirements, migration plan, and adoption contracts."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["practice","scenario","extraction","architecture","reuse"]
  topics: ["feature extraction","capability extraction","shared scenario","scenario extraction","scenario migration"]
  icon: "git-branch-plus"
  status: "active"
  revision: 1
  createdAt: "2026-05-14T00:00:00Z"
  updatedAt: "2026-05-14T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "vrooli scenario"]
  origin:
    kind: "authored"
---
## Practice focus: Scenario Capability Extraction

Extract mature product capabilities from one or more existing scenarios into a dedicated reusable Vrooli scenario. The outcome is never a headless service: every extracted capability becomes a full scenario with API, CLI, UI, docs, requirements, tests, adoption contracts, and a migration path for source and consumer scenarios.

Required reading:
- `prompt-manager skill read scenario-generation`
- `prompt-manager skill read architecture-lenses`
- `templates/scenarios/react-vite/docs/START-HERE.md`
- `templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md`
- `templates/scenarios/react-vite/docs/concepts/DOMAINS.md`
- `docs/agent-system/SKILL_AUTHORING.md`

Optional reading:
- `prompt-manager skill read interoperability-steer resilience-lenses cross-platform-readiness`
- Source scenario docs: `PRD.md`, `docs/concepts/*`, `docs/internal/SEAMS.md`, `docs/internal/PROBLEMS.md`, and `docs/internal/SWARM_MANAGER_WORK.md`

---

### **1. When to Use This Methodology**

Use this when a feature or subsystem embedded in one or more scenarios is ready to become its own reusable scenario.

| Signal | Use this? | Notes |
|---|---|---|
| One scenario has a mature feature now needed by other scenarios | Yes | Single-source extraction. |
| Two or more scenarios independently implemented similar behavior | Yes | Multi-source extraction; reconcile differences explicitly. |
| A capability needs provider routing, subscription/BYOK/local modes, or shared monetization | Yes | Dedicated scenario ownership is usually appropriate. |
| The work is only moving files into subpackages inside one scenario | No | Use architecture/refactor skills instead. |
| The target is a reusable agent methodology | No | Use `capability-extraction`; that skill extracts agent instructions into skills. |
| The target is a shared package with no scenario surface | Usually no | Prefer a scenario when the capability is user/operator-facing, monetizable, or composed by other scenarios. |

**Hard rule:** all Vrooli scenarios have a UI. If the extracted capability began as API-only behavior, the extraction still designs a UI for configuration, usage, diagnostics, adoption, docs, or operations.

---

### **2. The Process**

Each phase has an exit gate; the artifacts accumulate in the target scenario's `docs/internal/EXTRACTION-SOURCES.md` (a temporary planning note before generation).

**1. Intake.** Name the capability in product language; list source scenarios and known consumers; state why extraction is warranted now (a second or third scenario needs it, shared monetization or provider routing, stable source code, diverging duplicates, adoption friction); decide target id, display name, description, design kit and template (the normal generator unless a first-class template exists); record non-goals that stay source-specific. Exit: named target, source and consumer lists, rationale.

**2. Source evidence audit.** Read each source's PRD, domains, architecture, seams, integrations, problems, decisions and requirements, and inspect the capability across API, CLI, UI, proto, storage, config, tests and docs. Inventory source paths, implemented behavior, proving tests, config and provider assumptions, risks and known bugs, source-specific vocabulary, and missing tests or stale docs. In multi-source extraction, mark conflicts rather than silently choosing one. Exit: you can say what exists, where, and what proves it works.

**3. Boundary classification.** Classify each piece of behavior: shared capability core (target domains, API, CLI, UI), source integration glue (stays in the source and later calls the target), consumer adapter (target docs, examples, SDK or component surfaces), provider or resource boundary (target integrations, config, seams), migration-only compatibility (migration plan with a removal trigger), obsolete or debt (problem log, not ported). Exit: explicit enough that no implementer moves source code wholesale; durable tradeoffs go in `docs/internal/SWARM_MANAGER_WORK.md`.

**4. UI archetypes.** Every scenario has a UI, so an API-only capability still designs one. Pick archetypes that become PRD operational targets and requirements: Configuration Console (defaults, providers, credentials, flags, thresholds), Usage and History Dashboard (cost, latency, volume, fallbacks, audit), Adoption Management (consumer list, compatibility, setup checklist), Diagnostics Workbench (try-it, sample inputs, provider trace), Docs and Integration Viewer, Review or Moderation Queue, Catalog or Asset Browser. Default for infrastructure-like capabilities: Configuration Console, Usage Dashboard, Diagnostics Workbench and Docs Viewer, plus Adoption Management when consumer status matters day to day. Mark deferred surfaces P1, P2 or not-applicable.

**5. Generate and wire.** `template-manager generate react-vite --id "<scenario-id>" --display-name "<Display Name>" --description "<one-line purpose>" --design "<kit-id>"`, follow the generated `docs/START-HERE.md`, create `docs/internal/EXTRACTION-SOURCES.md`, register it in `docs/manifest.json` under `internal` (`docType: extraction-sources`, `canonicalFor: ["extraction-provenance", "source-capability-evidence", "migration-context"]`, `maturity: active`, `requiredBy: ["oriented", "implemented"]`), and add an early extraction gate to `START-HERE.md` before the PRD gates (read the dossier, confirm sources and boundaries, use it as context for PRD, requirements, domains, seams, integrations and migration). Exit: the generated scenario cannot be oriented without confronting source evidence.

**6. Synthesize.** Use the preserve-first PRD and requirements workflow from `scenario-generation`, seeding the PRD from the dossier (permanent capability, users and consumers, P0 to P2 targets, API, CLI and UI surfaces, provider and resource strategy, monetization and BYOK or local behavior, launch sequencing); give UI archetypes measurable requirements; update `docs/concepts/DOMAINS.md`, `INTEGRATIONS.md`, `FLOWS.md`, `docs/internal/SEAMS.md`, `SWARM_MANAGER_WORK.md` and `PROBLEMS.md`. Exit: an implementer can build from the target's own docs without the planning conversation.

**7. Migration and adoption.** Phase it: build the core, prove parity against the source, migrate the first source, then other consumers, retire the source implementation, and record compatibility removal triggers. Pick the work shape per `docs/agent-system/SCENARIO_DEVELOPMENT.md#work-shapes`: bounded tasks for independent reviewable integrations, an adaptive mandate when source and target changes are coupled or ground truth is stale, a phased plan for a stable sequential route. Create Swarm Manager items only after the target docs can anchor execution. Exit: adoption has owners, phases, validation and cleanup triggers.

**8. Verify and hand off.** Run `template-manager orient <scenario-id>` while orientation is active, `make test` once implementation begins, and `vrooli scenario requirements validate <scenario-id> --json`. Check that the dossier exists and is registered, START-HERE references it, PRD and requirements include UI archetypes, the domain map names real target domains, integrations document source, consumer and provider contracts, and the migration plan names source cleanup. Append progress and known gaps to `docs/internal/PROGRESS.md`.

### **3. Extraction dossier**

`docs/internal/EXTRACTION-SOURCES.md` holds: Purpose; Source Scenarios (scenario, primary, secondary or reference role, paths, evidence); Target Consumers (need, required contract, adoption phase); Boundary Classification (source behavior, classification, target destination, source fate: keep, migrate, remove or defer); UI Archetypes (priority, fit, users, requirements); Parity Requirements (existing behavior, evidence, target requirement, validation); Migration Plan; Open Questions (impact, owner or resolution path).

---

### **4. Convergence Patterns**

#### **Extract Or Leave Local**

| Question | If YES | If NO |
|---|---|---|
| Is the capability needed by multiple scenarios now or soon? | Continue extraction assessment | Leave local unless monetization/platform need is strong |
| Does it need operator configuration, usage visibility, diagnostics, or docs? | Scenario extraction favored | Still design UI if extracting |
| Does it have provider/resource routing or monetization implications? | Scenario extraction favored | Shared package may be enough only if no scenario/product surface exists |
| Is the source behavior mature enough to define contracts? | Extract with parity requirements | Mature in place first |
| Can source-specific glue be separated from reusable behavior? | Proceed | Refactor source boundaries before extracting |

#### **Where UI Intent Lives**

| UI Decision | Durable Home |
|---|---|
| Which archetypes are P0/P1/P2 | `docs/internal/EXTRACTION-SOURCES.md`, `PRD.md` |
| Measurable UI behavior | `requirements/` |
| Which domain owns each UI surface | `docs/concepts/DOMAINS.md` |
| Stateful UI workflows | `docs/concepts/FLOWS.md` |
| Visual language, density, tokens, accessibility feel | `DESIGN.md` |
| Meaningful UI tradeoffs | `docs/internal/SWARM_MANAGER_WORK.md` |
| Known UI gaps/deferred surfaces | `docs/internal/PROBLEMS.md` |

#### **Operating Mode Choice After Setup**

| Work Shape | Recommended Swarm Manager shape |
|---|---|
| Independent adoption items, each reviewable alone | Bounded tasks |
| Target/source changes are coupled and likely to shift | Adaptive mandate (`adaptive-mandate-authoring`) |
| Stable multi-phase implementation should be drained sequentially with handoffs | Phased plan (`implementation-plan-authoring`), sliced or goal mode |
| Full greenfield scenario development beyond extraction setup | Use the `scenario-generation` skill and a plan-backed Swarm Manager item |

---

### **5. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| Headless extraction | Violates Vrooli scenario requirements and hides operator configuration/diagnostics | Always design API, CLI, and UI surfaces |
| Blind code move | Carries source-specific assumptions and debt into the shared scenario | Classify shared core vs glue vs obsolete first |
| PRD from imagination | Loses hard-won source behavior and tests | Generate PRD context from extraction evidence |
| UI as afterthought | Agents leave placeholder shell/settings in place | Select UI archetypes before PRD/requirements |
| Unregistered dossier | Future agents miss extraction context | Add dossier to docs manifest and START-HERE |
| Permanent compatibility bridge | Migration code becomes product code | Add removal trigger and owner |
| Multi-source averaging | Conflicts vanish into a vague combined design | Record conflicts and decide explicitly |

---

### **6. Troubleshooting & Edge Cases**

| Symptom | Likely Cause | First Check | Fix |
|---|---|---|---|
| Agent cannot tell what belongs in target scenario | Boundary map is missing or too broad | `EXTRACTION-SOURCES.md` classifications | Re-run Phase 3 before implementation |
| Generated scenario still has placeholder UI plan | UI archetypes were not selected or not copied into PRD/requirements | PRD `UX & Branding`, requirements modules | Add P0 UI targets and requirements before coding |
| Source scenario docs contradict source code | Stale docs after source evolution | Tests and current source paths | Record drift in dossier; trust current tested behavior over stale docs |
| Multiple source scenarios disagree | Different local product needs were conflated | Compare behavior/test evidence side by side | Choose target contract explicitly; record source-specific adapters |
| Extraction seems like a shared package only | No operator/admin surface was considered | UI archetype table | Reassess configuration, usage, diagnostics, docs, adoption, monetization |
| START-HERE does not mention extraction context | Skill setup was incomplete | Target `docs/START-HERE.md` | Add extraction gate before PRD/requirements |

---

### **7. Output Expectations**

When applying this skill, you must produce:

- A target scenario identity and extraction rationale.
- A source evidence inventory.
- `docs/internal/EXTRACTION-SOURCES.md` in the target scenario.
- A docs manifest entry for the extraction dossier.
- A `docs/START-HERE.md` extraction context gate in the target scenario.
- Selected UI archetypes with P0/P1/P2 priority.
- PRD and requirements context grounded in source evidence.
- Domain, integration, seam, flow, decision, problem, and migration updates sufficient for future agents.
- Verification evidence or explicit notes explaining what could not yet be run.

You may create or update Swarm Manager items after the target scenario has enough durable context.

You must not:

- Treat extracted scenarios as headless.
- Skip source evidence because the desired target seems obvious.
- Copy source code wholesale without classifying boundaries.
- Leave adoption and source cleanup implicit.
- Add a new work shape or execution mode; the four shapes in `SCENARIO_DEVELOPMENT.md` and the two modes in `SCENARIO_DEVELOPMENT.md` cover extraction work.
