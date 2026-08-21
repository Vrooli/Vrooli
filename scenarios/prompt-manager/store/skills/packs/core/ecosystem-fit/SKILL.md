## Practice focus: Ecosystem Fit

Before treating a scenario as a standalone app, decide how it becomes a good citizen of Vrooli's self-improving ecosystem — which **interfaces** it serves, what **functional role** it plays, and how it is built for **compound value**. The goal is to make the vision bite at decision time, so two agents planning the same scenario reach the same structural decisions about where it fits.

Required reading:
- `path:docs/concepts/ECOSYSTEM.md` — the canonical model (the two axes, the interface/enabler tables, worked examples). This skill is the on-demand walkthrough of that doc; do not restate its taxonomy, apply it.
- `path:docs/director-swarm/strategy/OBJECTIVES.md` — the objective set cluster 4 names an id from. Read live state with `prompt-manager graph objectives`.
- `path:docs/agent-system/PROMOTION_LADDER.md` — the retirement and retention criteria cluster 5 claims against.

Optional reading:
- `path:VISION.md` — the *why* behind the loop.
- `path:docs/concepts/PAID_FEATURES.md` — the free / metered / gated contract, for cluster 6 (monetization).
- `prompt-manager skill read morning-vision-walk` — the same lens applied at the portfolio level.

**Where this hands off.** After placement is decided: build a scenario that does not exist yet with `prompt-manager skill read scenario-generation`; change a scenario that already exists with `prompt-manager skill read scenario-work-ladder`, which locates the layer to change before any change is made.

---

### 1. When to Use This Skill

Apply the lens with **depth scaled to the work**. Do not run a full-cluster review on a one-line bugfix.

```
What kind of work is this?
├─ New scenario                     → apply ALL SIX clusters; record the answers in the plan
├─ Significant refactor / repurpose → apply ALL SIX; re-check role + interfaces may have shifted
├─ New feature on a scenario        → clusters 1 (interfaces), 3 (compound value), 5 (retirement claim); + 6 if it is a paid/expensive feature
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
- Skill discovery for the plan — that is the plan-manager authoring wizard's server-side context discovery.
- Editing the canonical taxonomy itself — that is a change to `path:docs/concepts/ECOSYSTEM.md` (operator-reviewed).
- Editing the objective set — that is a change to `path:docs/director-swarm/strategy/OBJECTIVES.md`, actuated by `director-swarm`.
- Portfolio prioritization — that is `morning-vision-walk`.
- Monetization *strategy* (whether to monetize, pricing, bundle membership) — operator-curated `path:docs/monetization/` canon; this skill only routes there, it never decides or edits it.
- Paid-feature *wiring* detail — `path:docs/concepts/PAID_FEATURES.md` + `prompt-manager skill read bundle-integration-steer`.

---

### 3. The Clusters

Walk these in order. Each row names the question and what a *good* answer looks like — not a step to mechanically perform.

| Cluster | Ask | A good answer names… |
|---|---|---|
| **1. Interfaces** | Which channel(s) does this serve or enable, and what does that make "done" mean? | Every interface the scenario touches, and its "done" obligation from the interface map below. |
| **2. Role & multiplier** | Which functional role does this play, and is there a cheap way to raise its multiplier? | The role (meta / interface-enabler / integration / product) **and** at least a quick check for: an LLM step that could become deterministic code or a `prompt-manager action`; a capability worth exposing instead of burying. |
| **3. Compound value** | Is it built to be extended and composed later? What seams make that cheap? | The concrete seam(s) (data surface, declared widgets/tools, stable CLI) that let a *future* scenario reuse this one instead of re-implementing it. `tech-tree-designer` answers this programmatically when it is running; treat its output as advisory — the scenario is built but unvalidated. |
| **4. Objective served** | Which objective does this advance, and which team owns it? | An objective id (`T1`–`T3`, `I1`–`I3`) plus the team that declares it, read from `prompt-manager graph objectives` — or "none — pure product." Name the id even when the sensor reports it unserved. No id fits → file the gap to `director-swarm`; never edit `OBJECTIVES.md`. |
| **5. Retirement claim** | What does this let the system delete? | A file path and section that becomes retireable, plus the trigger that makes it eligible, per `PROMOTION_LADDER.md` §Retirement criteria — or "nothing retires." Check §Retention criteria first. Do not claim a retirement you have not opened. |
| **6. Monetization & bundle fit** | How does this earn its keep, and which bundle does it serve? | The bundle (business / lifestyle) + headliner-or-depth, **and** whether each capability is free / metered / gated. Strategy (whether to monetize, pricing) is deferred to canon — routed, not decided here. |

**Why clusters 4 and 5 exist, and when they retire.** The intent chain in `path:docs/agent-system/README.md` is validated downward from objective to member surface. Nothing validates upward from a new capability, so a scenario can ship, work, and stay invisible to the strategy layer. Cluster 5 is that same README's ratchet — *every capability added to the system is supposed to make the system smaller* — claimed at design time rather than measured after the fact by `prompt-manager graph orientation-cost`. Both clusters retire from this skill once a scenario declares its objective edge and its retirement claim as data the sensors read directly, which is `PROMOTION_LADDER.md` step 2.

#### The interface "done" map (cluster 1 detail)

| If the scenario touches… | "Done" additionally requires |
|---|---|
| Direct UI | Polished, production-ready UI that renders loading, error, and empty states |
| Conversational / agentic | Widgets + tools declared per the contract **and** discoverable (`cli-health` / `ui-health`) |
| Voice | Voice features wired into the actual consuming scenarios, not merely available |
| Programmatic | A clean, reusable CLI / Connect surface — assume other scenarios will call it |
| Embodied / embedded | A connector seam; do not hand-roll the outbound integration inside the scenario |

#### Monetization routing (cluster 6 detail)

This skill **routes**, it does not decide strategy. Read the canon, pick the integration pattern:

| Need | Where (read-only — never edit canon) |
|---|---|
| Which bundle / SKU role | `path:docs/monetization/catalogs/CATALOG.md` for judgment, and `offer-desk offers catalog-list` / `catalog-edges` for the live scenario→SKU map (`belongs_to` edges) |
| Monetization posture / "should we even charge" | `path:docs/monetization/strategy/STRATEGY.md`; portfolio call → `morning-vision-walk` |
| Free vs **metered** vs **gated**, and how to wire each | `path:docs/concepts/PAID_FEATURES.md` |
| Wiring the bundle/entitlement integration | `prompt-manager skill read bundle-integration-steer` |

Two hard rules carried from `PAID_FEATURES.md`: never gate a capability a self-hoster could run with their own keys (keep BYOK valid); route metered/gated features through LPBS instead of reinventing credits/entitlements.

---

### **4. Output Expectations**

When this skill is applied during planning, you **must** produce:
- A one-line **role** classification and the **interface(s)** the scenario serves/enables.
- The resulting **"done" obligations** from cluster 1's interface map (these belong in the plan's Target End State / Definition of Done).
- At least a **compound-value seam** note (cluster 3) — or an explicit "no extension foreseen; minimal seam is X."
- The **objective id** this advances and its owning team (cluster 4) — or an explicit "none — pure product."
- The **retirement claim** with its trigger (cluster 5) — or an explicit "nothing retires," stated rather than omitted.

You **should** also:
- Note any cheap multiplier raise spotted in cluster 2 (LLM→action/code, capability to expose).
- When cluster 6 applies, name the **bundle** and the **free / metered / gated** mode per chargeable capability — routing to `path:docs/concepts/PAID_FEATURES.md`, never deciding pricing.
- Keep the output to a few lines — this is a lens, not a deliverable. The depth tree in §1 governs how much to write.

You **must not:**
- Run all six clusters on work the §1 tree says to skip.
- Invent interface or voice obligations for a scenario that genuinely has none — an honest "pure product, direct UI only" is a complete answer.
- Restate the taxonomy from `path:docs/concepts/ECOSYSTEM.md` in the plan; cite it and apply it.
- Edit `path:docs/director-swarm/strategy/OBJECTIVES.md` or invent an objective id. `director-swarm` owns the objective set; this skill names and routes.
- Claim a retirement you have not opened. Name the file and section, or say nothing retires — an unverifiable claim satisfies the ratchet on paper while the prose stays.

---

### 5. Troubleshooting & Edge Cases

- **Scenario spans multiple roles.** Expected and fine — classify by the *dominant* role for the multiplier question, but answer cluster 1 for *every* interface it touches.
- **`tech-tree-designer` errors or returns nothing.** Reason about compound value from `path:docs/concepts/ECOSYSTEM.md` instead; do not treat its absence as a blocker or its output as authoritative.
- **No objective id fits the work.** This is the cluster 4 rejection branch, not a stall. Record "no id fits", file the gap to `director-swarm`, and continue the remaining clusters.
