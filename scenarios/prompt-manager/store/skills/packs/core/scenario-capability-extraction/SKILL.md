## Practice focus: Scenario Capability Extraction

Extract mature product capabilities from one or more existing scenarios into a dedicated reusable Vrooli scenario. The outcome is never a headless service: every extracted capability becomes a full scenario with API, CLI, UI, docs, requirements, tests, adoption contracts, and a migration path for source and consumer scenarios.

Required reading:
- `prompt-manager skill read scenario-generation`
- `prompt-manager skill read screaming-architecture-audit boundary-of-responsibility-enforcement seam-discovery-and-enforcement decision-boundary-extraction`
- `templates/scenarios/react-vite/docs/START-HERE.md`
- `templates/scenarios/react-vite/docs/concepts/ARCHITECTURE.md`
- `templates/scenarios/react-vite/docs/concepts/DOMAINS.md`
- `docs/agent-system/SKILL_AUTHORING.md`

Optional reading:
- `prompt-manager skill read interoperability-steer temporal-flow-audit cross-platform-readiness`
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

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    SCENARIO CAPABILITY EXTRACTION                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ Intake ─▶ Source Audit ─▶ Boundary Map ─▶ UI Archetypes ─▶ Generate Scenario │
│   │            │              │              │                 │             │
│   ▼            ▼              ▼              ▼                 ▼             │
│ readiness   evidence     shared vs      required UI       extraction         │
│ decision    dossier      app glue       surfaces          orientation        │
│                                                                  │           │
│                                                                  ▼           │
│                 PRD + Requirements + Domains + Seams + Migration Plan        │
│                                                                  │           │
│                                                                  ▼           │
│                         Implement, Adopt, Retire Source Copies               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### **Phase 1: Extraction Intake**

**Entry criteria:** A capability candidate and at least one source scenario are known.

**Actions:**
1. Name the target capability in product language, not implementation language.
2. List source scenarios and known consumer scenarios.
3. State why extraction is warranted now:
   - second/third scenario needs it,
   - shared monetization/provider routing is needed,
   - source scenario code is stable enough to generalize,
   - duplicated implementations are diverging,
   - adoption friction is blocking higher-level scenarios.
4. Decide the target scenario ID, display name, description, initial design kit, and likely template. Use the normal scenario generator unless a more specific first-class template exists.
5. Record non-goals: what must remain source-scenario-specific.

**Exit criteria:** The extraction has a named target scenario, source list, consumer list, and extraction rationale.

**Artifacts:** Intake section in `docs/internal/EXTRACTION-SOURCES.md` after generation, or a temporary planning note before generation.

### **Phase 2: Source Evidence Audit**

**Entry criteria:** Source scenario(s) are available to inspect.

**Actions:**
1. Read each source scenario's PRD, domain map, architecture, seams, integrations, problems, decisions, and relevant requirements.
2. Inspect source code for the capability across API, CLI, UI, proto/contracts, storage, config, tests, and docs.
3. Build an evidence inventory:
   - source paths,
   - behavior currently implemented,
   - tests that prove behavior,
   - config/env/provider assumptions,
   - operational risks and known bugs,
   - source-specific product vocabulary,
   - missing tests or stale docs.
4. For multi-source extraction, compare behavior and mark conflicts rather than silently choosing one source.

**Exit criteria:** The agent can explain what exists today, where it lives, and what evidence proves it works.

**Artifacts:** `docs/internal/EXTRACTION-SOURCES.md` source inventory.

### **Phase 3: Boundary Classification**

**Entry criteria:** Source evidence inventory exists.

**Actions:**
Classify every meaningful piece of behavior:

| Classification | Meaning | Destination |
|---|---|---|
| Shared capability core | Product behavior the new scenario owns | Target scenario domains/API/CLI/UI |
| Source integration glue | Behavior only meaningful inside a source scenario | Stays in source; later calls target scenario |
| Consumer adapter | Code/config future adopters need | Target docs, examples, SDK/component surfaces |
| Provider/resource boundary | Local/BYOK/subscription/third-party routing choice | Target integrations, config, seams |
| Migration-only compatibility | Temporary bridge to preserve adoption | Target migration plan with removal trigger |
| Obsolete/debt | Behavior not worth carrying forward | Record in the problem log; do not port |

**Exit criteria:** The extraction boundary is explicit enough that an implementation agent will not blindly move source code wholesale.

**Artifacts:** Boundary map in `docs/internal/EXTRACTION-SOURCES.md`; durable decisions in `docs/internal/SWARM_MANAGER_WORK.md` when tradeoffs matter.

### **Phase 4: UI Archetype Selection**

**Entry criteria:** Target capability and boundary map are known.

**Actions:**
Pick one or more UI archetypes. The selected archetypes become product scope and must be reflected in PRD operational targets and requirements.

| UI Archetype | Use When | Typical Surfaces |
|---|---|---|
| Configuration Console | Operators must choose defaults, providers, credentials, feature flags, thresholds, or local vs subscription behavior | Settings, provider routing, credentials status, resource health |
| Usage And History Dashboard | The capability has cost, latency, volume, credits, fallbacks, generated artifacts, or audit history | Recent requests, cost/credit usage, provider used, failure reasons |
| Adoption Management | Other scenarios integrate with the capability and adoption status matters | Consumer list, version compatibility, integration health, setup checklist |
| Diagnostics Workbench | Operators need to manually test or debug the capability | Try-it panel, sample inputs, output inspector, provider trace |
| Docs / Integration Viewer | Adoption requires API/CLI/UI component guidance | Rendered docs, examples, schema links, copyable commands |
| Review / Moderation Queue | The capability produces items requiring human approval | Pending items, decisions, acceptance/reject reasons |
| Catalog / Asset Browser | The capability produces reusable assets or templates | Search, preview, metadata, export/use actions |

**Default for extracted infrastructure-like capabilities:** Configuration Console + Usage Dashboard + Diagnostics Workbench + Docs / Integration Viewer. Add Adoption Management when consumer status is important to day-to-day operation.

**Exit criteria:** The target scenario has a clear first UI shape and deferred UI surfaces are explicitly marked P1/P2 or not-applicable.

**Artifacts:** UI section in `docs/internal/EXTRACTION-SOURCES.md`; PRD `UX & Branding`; UI requirements; domain map UI surfaces.

### **Phase 5: Generate And Wire The Target Scenario**

**Entry criteria:** Target identity, template, design kit, extraction boundary, and UI archetypes are selected.

**Actions:**
1. Generate the scenario through the project CLI:

   ```bash
   template-manager generate react-vite --id "<scenario-id>" --display-name "<Display Name>" --description "<one-line purpose>" --design "<kit-id>"
   ```

2. Follow the generated `docs/START-HERE.md` orientation protocol.
3. Create `docs/internal/EXTRACTION-SOURCES.md` in the target scenario.
4. Add the extraction document to the scenario's `docs/manifest.json` under `internal` with:
   - `docType`: `extraction-sources`
   - `canonicalFor`: `["extraction-provenance", "source-capability-evidence", "migration-context"]`
   - `maturity`: `active`
   - `requiredBy`: `["oriented", "implemented"]`
5. Update the generated target scenario `docs/START-HERE.md` with an early extraction context gate before the PRD/requirements gates:
   - read `docs/internal/EXTRACTION-SOURCES.md`,
   - confirm source scenarios and boundary classification,
   - use it as required context for PRD, requirements, domains, seams, integrations, and migration planning.

**Exit criteria:** The generated target scenario cannot be oriented without confronting source evidence.

**Artifacts:** Target scenario scaffold, extraction sources doc, manifest entry, START-HERE extraction gate.

### **Phase 6: Synthesize PRD, Requirements, Domains, And Docs**

**Entry criteria:** Target scenario exists and extraction context is wired into orientation.

**Actions:**
1. Use the preserve-first PRD + requirements workflow from `scenario-generation` (validate/fix a preserved baseline, or author with the business-health wizard).
2. Generate or update PRD context from the extraction dossier:
   - permanent capability,
   - primary users/operators/consumer scenarios,
   - P0/P1/P2 operational targets,
   - API/CLI/UI deployment surfaces,
   - provider/resource strategy,
   - monetization/subscription/BYOK/local behavior,
   - migration and launch sequencing.
3. Generate requirements from the PRD and ensure UI archetypes have measurable requirements.
4. Update:
   - `docs/concepts/DOMAINS.md` for bounded contexts and UI/API/CLI surfaces,
   - `docs/concepts/INTEGRATIONS.md` for source/consumer scenarios, providers, resources, and failure behavior,
   - `docs/concepts/FLOWS.md` for stateful workflows,
   - `docs/internal/SEAMS.md` for provider, transport, storage, and test substitution boundaries,
   - `docs/internal/SWARM_MANAGER_WORK.md` for meaningful extraction tradeoffs,
   - `docs/internal/PROBLEMS.md` for source drift, missing tests, and deferred migrations.

**Exit criteria:** Future implementation agents can build the scenario from the target scenario's own PRD, requirements, and docs without needing the original planning conversation.

**Artifacts:** PRD, requirements registry, domain map, integrations, seams, flows, decisions, problems.

### **Phase 7: Migration And Adoption Plan**

**Entry criteria:** Target scenario docs and requirements define the intended capability.

**Actions:**
1. Define phased migration:
   - build target scenario core,
   - prove parity against source behavior,
   - migrate first source scenario,
   - migrate additional consumers,
   - remove or retire source implementation,
   - document compatibility removal triggers.
2. Decide whether adoption work belongs in item-level mode, `holistic-loop`, or `phased-plan-drain`:
   - Use item-level when integrations are independent and reviewable in isolation.
   - Use holistic-loop when source and target changes are coupled or ground truth is stale.
   - Use phased-plan-drain when there is a stable sequential plan with handoffs.
3. Create or update Swarm Manager items only after the target scenario has enough docs to anchor execution.

**Exit criteria:** Adoption is not left as an implicit "later"; it has owners, phases, validation, and cleanup triggers.

**Artifacts:** Migration section in target docs, backlog items or initiative plan, progress/problem entries.

### **Phase 8: Verification And Handoff**

**Entry criteria:** Setup or implementation phase is complete enough to hand off.

**Actions:**
1. Run relevant checks:
   - `template-manager orient <scenario-id>` while orientation is active,
   - `make test` from the scenario once implementation begins,
   - `vrooli scenario requirements validate <scenario-id> --json` (one command covers PRD linkage + requirements registry).
2. Check the extraction contract:
   - dossier exists and is registered,
   - START-HERE references it,
   - PRD and requirements include UI archetypes,
   - domain map names real target domains,
   - integrations document source/consumer/provider contracts,
   - migration plan names source cleanup.
3. Append progress and known gaps to target scenario docs.

**Exit criteria:** The next agent can continue without rediscovering source evidence or asking why the scenario exists.

**Artifacts:** Validation evidence, `docs/internal/PROGRESS.md`, final handoff.

---

### **3. Extraction Dossier Template**

Create this as `docs/internal/EXTRACTION-SOURCES.md` in the target scenario.

```markdown
# Extraction Sources

## Purpose
[Capability being extracted and why this scenario owns it.]

## Source Scenarios
| Scenario | Role | Relevant Paths | Evidence | Notes |
|---|---|---|---|---|
| ... | primary / secondary / reference | ... | tests/docs/behavior | ... |

## Target Consumers
| Scenario | Need | Required Contract | Adoption Phase |
|---|---|---|---|
| ... | ... | API/CLI/UI component/config | P0/P1/P2 |

## Boundary Classification
| Source Behavior | Classification | Target Destination | Source Fate |
|---|---|---|---|
| ... | shared core / glue / adapter / provider / migration-only / obsolete | ... | keep / migrate / remove / defer |

## UI Archetypes
| Archetype | Priority | Why It Fits | Primary Users | Requirements |
|---|---|---|---|---|
| Configuration Console | P0 | ... | operators | ... |

## Parity Requirements
| Existing Behavior | Source Evidence | Target Requirement | Validation |
|---|---|---|---|
| ... | ... | REQ-... | test/manual check |

## Migration Plan
1. ...

## Open Questions
| Question | Impact | Owner / Resolution Path |
|---|---|---|
| ... | ... | ... |
```

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

| Work Shape | Recommended Swarm Manager Mode |
|---|---|
| Independent adoption items, each reviewable alone | `item-level` |
| Target/source changes are coupled and likely to shift | `holistic-loop` |
| Stable multi-phase implementation should be drained sequentially with handoffs | `phased-plan-drain` |
| Full greenfield scenario development beyond extraction setup | Use the `scenario-generation` skill and a Swarm Manager phased-plan-drain workflow |

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
- Add a new operating mode unless the work shape requires one beyond existing `item-level`, `holistic-loop`, or `phased-plan-drain`.
