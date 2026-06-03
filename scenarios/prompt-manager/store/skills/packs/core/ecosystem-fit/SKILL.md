## Practice focus: Ecosystem Fit

Before treating a scenario as a standalone app, decide how it becomes a good citizen of Vrooli's self-improving ecosystem — which **interfaces** it serves, what **functional role** it plays, and how it is built for **compound value**. The goal is to make the vision bite at decision time, so two agents planning the same scenario reach the same structural decisions about where it fits.

Required reading:
- `path:docs/concepts/ECOSYSTEM.md` — the canonical model (the two axes, the interface/enabler tables, worked examples). This skill is the on-demand walkthrough of that doc; do not restate its taxonomy, apply it.

Optional reading:
- `path:VISION.md` — the *why* behind the loop.
- `prompt-manager skill read morning-vision-walk` — the same lens applied at the portfolio level.

---

### 1. When to Use This Skill

Apply the lens with **depth scaled to the work**. Do not run a four-cluster review on a one-line bugfix.

```
What kind of work is this?
├─ New scenario                     → apply ALL FOUR clusters; record the answers in the plan
├─ Significant refactor / repurpose → apply ALL FOUR; re-check role + interfaces may have shifted
├─ New feature on a scenario        → clusters 1 (interfaces) + 3 (compound value)
├─ Small feature / polish           → cluster 1 only (does "done" mean a new interface obligation?)
└─ Bugfix / chore                   → skip; note fit only if the fix reveals a missing interface/seam
```

| Situation | Use this skill? | Why |
|---|---|---|
| Authoring a plan that creates or repurposes a scenario | Yes | Fit decisions are cheapest before code exists |
| Deciding what "production-ready" means for a scenario | Yes | "Done" depends on which interfaces it must serve |
| A pure bugfix or dependency bump | No | No fit decision is in play |
| Portfolio-level "what should we build next" | No | Use `morning-vision-walk` |

---

### 2. Scope Boundaries

**In scope:**
- Classifying a scenario by functional role and interfaces (per `path:docs/concepts/ECOSYSTEM.md`).
- Translating that classification into concrete "what does done mean" obligations.
- Identifying cheap multiplier raises and compound-value seams.

**Out of scope:**
- The actual implementation plan structure — that is `implementation-plan-authoring`.
- Skill discovery for the plan — that is `plan-skill-discovery`.
- Editing the canonical taxonomy itself — that is a change to `path:docs/concepts/ECOSYSTEM.md` (operator-reviewed).
- Portfolio prioritization — that is `morning-vision-walk`.

---

### 3. The Four Clusters

Walk these in order. Each row names the question and what a *good* answer looks like — not a step to mechanically perform.

| Cluster | Ask | A good answer names… |
|---|---|---|
| **1. Interfaces** | Which channel(s) does this serve or enable, and what does that make "done" mean? | Direct UI → polished/production-ready. Conversational → widgets + tools declared **and discoverable** via `cli-health`/`ui-health`. Voice → actually wired into consumers, not just present. Programmatic → a clean CLI/Connect surface other scenarios can call. |
| **2. Role & multiplier** | Which functional role does this play, and is there a cheap way to raise its multiplier? | The role (meta / interface-enabler / integration / product) **and** at least a quick check for: an LLM step that could become deterministic code or a `prompt-manager action`; a capability worth exposing instead of burying. |
| **3. Compound value** | Is it built to be extended and composed later? What seams make that cheap? | The concrete seam(s) (data surface, declared widgets/tools, stable CLI) that let a *future* scenario reuse this one instead of re-implementing it. |
| **4. Self-improvement** | Could this cheaply advance a Vrooli meta-capability (engineering, testing, deployment, monetization, upkeep, or operator/user interaction)? | Either a concrete meta-capability it advances, or an explicit "no — pure product," which is a valid answer. |

#### The interface "done" map (cluster 1 detail)

| If the scenario touches… | "Done" additionally requires |
|---|---|
| Direct UI | Polished, production-ready UI (handles loading/error/empty states) |
| Conversational / agentic | Widgets + tools declared per the contract **and** discoverable (`cli-health` / `ui-health`) |
| Voice | Voice features wired into the actual consuming scenarios, not merely available |
| Programmatic | A clean, reusable CLI / Connect surface — assume other scenarios will call it |
| Embodied / embedded | A connector seam; do not hand-roll the outbound integration inside the scenario |

---

### 4. The `tech-tree-designer` hook (optional, graceful degradation)

Cluster 3 ("where does this sit / what does it unlock") can be answered programmatically **if** `tech-tree-designer` is available — it models the map of all possible software (domain × maturity) and what each node unlocks.

```
Is tech-tree-designer discoverable and running?
(prompt-manager discover "tech tree" --type all  /  scenario status)
├─ YES → query it for the closest node and downstream unlocks
│        (e.g. tech-tree-designer graph dependencies / catalog list)
│        Treat results as advisory — it may be unvalidated.
└─ NO  → reason about compound value manually from
         path:docs/concepts/ECOSYSTEM.md (the default path)
```

Never block fit analysis on `tech-tree-designer`. It is a power-up, not a prerequisite — the lens is fully usable from this skill alone. (As of this writing the scenario is built but unvalidated, with no functional-role dimension or semantic search; the manual path is the reliable one.)

---

### 5. Output Expectations

When this skill is applied during planning, you **must** produce:
- A one-line **role** classification and the **interface(s)** the scenario serves/enables.
- The resulting **"done" obligations** from cluster 1's interface map (these belong in the plan's Target End State / Definition of Done).
- At least a **compound-value seam** note (cluster 3) — or an explicit "no extension foreseen; minimal seam is X."

You **should** also:
- Note any cheap multiplier raise spotted in cluster 2 (LLM→action/code, capability to expose).
- Keep the output to a few lines — this is a lens, not a deliverable. The depth tree in §1 governs how much to write.

You **must not:**
- Run all four clusters on work the §1 tree says to skip.
- Invent interface or voice obligations for a scenario that genuinely has none — an honest "pure product, direct UI only" is a complete answer.
- Restate the taxonomy from `path:docs/concepts/ECOSYSTEM.md` in the plan; cite it and apply it.

---

### 6. Troubleshooting & Edge Cases

- **Scenario spans multiple roles.** Expected and fine — classify by the *dominant* role for the multiplier question, but answer cluster 1 for *every* interface it touches.
- **`tech-tree-designer` errors or returns nothing.** Fall back to the manual path (§4); do not treat its absence as a blocker or its output as authoritative.
- **The lens feels like overhead on small work.** That is the §1 depth tree doing its job — skip clusters the tree says to skip. Forcing a full review on a bugfix is the anti-pattern, not diligence.
