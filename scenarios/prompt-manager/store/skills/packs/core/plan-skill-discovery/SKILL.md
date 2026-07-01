## Practice focus: Plan Skill Discovery

Methodology for choosing the right **Prompt Manager skills** for a Plan Manager implementation plan. Use it to preserve Prompt Manager's curated skill-pack and budget behavior while Plan Manager owns the authoring flow and Search Hub owns broad federated recall.

Required reading:
- `docs/agent-system/SKILL_AUTHORING.md`
- `scenarios/plan-manager/docs/concepts/PLAN-MODEL.md`
- `scenarios/plan-manager/docs/reference/cli-commands.md`

---

### **1. When to Use This Methodology**

Use Plan Skill Discovery when:
- Creating an implementation plan and the user has **not** provided specific skills
- You need a curated, budget-aware `prompt-manager skill read ...` set for Plan Manager setup context
- Plan Manager `author context-discover` proposes a `prompt-manager-skill-discovery` candidate and you need to decide whether to accept it
- An automated workshop agent needs to discover domain-relevant **skills** for a backlog item

**Discovery layers are intentionally separate:**

| Layer | Use it for | What it does not replace |
|---|---|---|
| `search-hub query ...` | Broad recall across records, docs, skills, backlog, commands, and other registered corpora | The curated/budgeted Prompt Manager skill-read contract |
| `prompt-manager discover ... --type skill` | Plan-authoring skill selection: strong direct skill hits plus curated topic packs and complexity budgets | Broad cross-corpus recall |
| `plan-manager author context-discover` | Reviewable setup candidates inside the authoring wizard | Author judgment about which candidates matter |

This workflow uses **Prompt Manager skill mode** (`--type skill`, the default) on purpose. Skill mode is *curated*: `prompt-manager discover` treats topics as packs. When a query fires a relevant topic, that topic's skill pack is included and strong direct matches surface above the packs. `search-hub query --type skill` can find skill hits, but it does not currently expose the same multi-query, topic-pack, budget-status, and recommended-read-command artifact.

**Do NOT use** for:
- Routine task execution where Plan Manager has already provided accepted setup context
- Quick one-step fixes with no planning phase
- Executing a plan that already contains accepted Plan Manager `relevant_context`
- Cases where the user already provided the exact skills to use
- General prior-work recall across records/docs/backlog; use `search-hub query` for that

---

### **2. The Process**

```
RECALL BROADLY -> CLASSIFY SKILL NEEDS -> RUN CURATED SKILL DISCOVERY -> ACCEPT/REJECT IN PLAN MANAGER
```

---

### **Phase 1: Recall Broadly**

**Entry criteria:** You are about to create or update an implementation plan.

**Actions:**
1. Use Search Hub first for broad prior-work and source-of-truth recall:
   ```bash
   search-hub query "<one-sentence plan intent>" --type record,backlog,initiative,skill,doc
   ```
2. Use those results to identify prior work, existing plans/backlog, canonical docs, and candidate skill concepts. Do not treat this as the final skill-read bundle.

**Exit criteria:**
- [ ] Broad recall was run or explicitly unavailable
- [ ] Prior work/docs/backlog findings were folded into the plan context
- [ ] Candidate skill concepts are ready for curated discovery

---

### **Phase 2: Classify Skill Needs**

**Entry criteria:** Broad recall is complete.

**Actions:**
1. Decompose the work into distinct skill-discovery concepts: domain area, technology/stack, problem type, and scenario surface. Each concept becomes its own Prompt Manager discovery query.

```
What kind of work is this?
├─ Bug/debugging        -> concepts: "debugging", "<domain>", "<error type>"
├─ New feature/scenario -> concepts: "<scenario-name>", "<domain>", "<technology>"
├─ Refactor/cleanup     -> concepts: "refactor", "<area>", "<pattern>"
├─ Deployment/infra     -> concepts: "deploy", "<target>", "<infrastructure>"
└─ Unsure               -> concepts: "<goal description>", "<technology>", "<problem type>"
```

> **Ecosystem-fit check.** For **New feature/scenario** and **Refactor/cleanup** classifications, also load `ecosystem-fit` (`prompt-manager skill read ecosystem-fit`). It places the work in Vrooli's interfaces, functional role, and compound-value design so the plan reflects how the change fits the whole system.

2. Formulate 2-5 focused search queries, one per concept.

**Exit criteria:**
- [ ] Work type identified
- [ ] 2-5 focused skill-discovery queries formulated

---

### **Phase 3: Run Curated Skill Discovery**

**Entry criteria:** 2-5 focused search queries are ready.

**Actions:**
1. Run Prompt Manager discovery in skill mode with all queries and a complexity level:
   ```bash
   prompt-manager discover "concept1" "concept2" "concept3" --type skill --complexity moderate
   ```

   Phrase queries as activity + surface concepts so the right topic packs fire, for example: "refactor and clean up duplication", "design a CLI command surface", "graceful degradation under failure". A bare noun often falls back to pure skill search.

2. Choose the complexity that matches the work:

   | Complexity | When to Use |
   |---|---|
   | `minor` | Bug fix, small tweak |
   | `moderate` | New feature, refactor |
   | `major` | Multi-file feature, new endpoint |
   | `architectural` | Cross-scenario, new system design |

3. If over budget, use the recommended trimmed read command from the output. The trim keeps strong direct matches and curated packs before weak embedding-tail results.
4. If you get few results, broaden the query phrasing toward the kind of work and the affected surface.

**Exit criteria:**
- [ ] Prompt Manager discovery command executed with appropriate complexity
- [ ] Budget status checked
- [ ] Skill candidates identified

---

### **Phase 4: Confirm Skill Relevance**

**Entry criteria:** Candidate skills found.

**Actions:**
1. Read candidates:
   ```bash
   prompt-manager skill read <id-1> <id-2> -output combined
   ```
2. Assess each candidate autonomously:
   - Does it apply to this specific task?
   - Would it materially improve the plan's quality?
   - Is it a Practice, Steer, Search, or Tools skill the executing agent needs?
   - Can you articulate the specific way it improves the plan?
3. Include a skill only if you can articulate that concrete improvement. Vague relevance is insufficient.

| Skill Relevance | Action |
|---|---|
| Directly applicable to the task | Accept as required Plan Manager context |
| Tangentially related | Discard |
| No relevant skills found | Proceed with explicit no-context rationale |

**Exit criteria:**
- [ ] Final skill list determined
- [ ] Discarded candidates have a clear reason

---

### **Phase 5: Accept or Submit in Plan Manager**

**Entry criteria:** Relevant skills confirmed.

**Actions:**
1. Prefer Plan Manager's context candidate workflow when authoring is already in a session:
   ```bash
   plan-manager author context-discover <session> --concepts "concept1,concept2" --complexity moderate
   plan-manager author context-accept <session> <candidate-id>
   ```
   Plan Manager currently proposes multiple candidate setup commands per concept, including Prompt Manager curated skill discovery, Prompt Manager operational/action discovery, Search Hub recall, and CLI Health search. Accept only the candidates that materially improve execution.
2. If the curated `prompt-manager discover` output gives a concrete skill-read command, submit that exact setup item directly:
   ```bash
   plan-manager author context-submit <session> \
     --kind skill \
     --label "Curated Prompt Manager skills" \
     --reason "<why these skills are needed>" \
     --instruction "Run the command and read the returned skills before implementation." \
     --command "prompt-manager skill read <skill-1> <skill-2>" \
     --required
   ```
3. If no relevant skills exist, encode that explicitly in the authoring session instead of inventing filler context.

**Exit criteria:**
- [ ] Plan Manager session contains accepted `relevant_context` items, or explicit no-context rationale
- [ ] Each accepted item has a reason and executable setup instruction
- [ ] Executing agent can reproduce the same context through Plan Manager setup output

---

### **3. Plan Manager Conventions**

These conventions apply to implementation plans created through Plan Manager.

#### **3a. Work Posture Is Plan Manager-Owned**

Plan Manager derives work posture from scope, compatibility constraints, and user intent. Do not hand-author a generic "greenfield by default" markdown block. Instead, make the compatibility requirement explicit in the authoring loop:
- If there are no published consumers or compatibility obligations, state that compatibility shims and legacy wrappers are prohibited.
- If compatibility matters, identify the consumers and migration expectation.
- If unknown, add a plan risk or investigation step rather than assuming compatibility.

#### **3b. Scenario Plans Must End with Cleanup + Health Verification**

Any plan that modifies a scenario must encode final validation work in its phases and acceptance criteria:
1. Fix lint, type, and unit test issues in modified files, including issues that appear pre-existing but are exposed by the change.
2. Restart the scenario with `vrooli scenario restart <name>` or the scenario-local `make` lifecycle where appropriate.
3. Verify the scenario is healthy through its API/UI checks.

Do not leave this as a prose reminder outside the structured plan. It belongs in Plan Manager phase steps, validation commands, and acceptance criteria.

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| Searching during execution | Setup context should already be accepted in Plan Manager | Search during planning only |
| Including every result | Context bloat, irrelevant noise | Accept only materially useful context |
| Skipping search entirely | Misses organizational knowledge | Search when planning unless the user supplied exact context |
| Single broad query | Poor recall across diverse concepts | Decompose into 2-5 focused queries |
| Asking user to confirm in automated contexts | Blocks autonomous processing | Assess relevance autonomously |
| Hand-authoring Required Reading markdown | Duplicates Plan Manager's setup model | Use `relevant_context` candidates/submissions |
| Treating Search Hub skill hits as the curated bundle | Loses Prompt Manager topic-pack and budget behavior | Use `prompt-manager discover --type skill` for the final skill-read set |
| Accepting every Plan Manager context candidate | Bloats setup with duplicate search commands | Accept only candidates with a concrete execution reason |

---

### **5. Boundaries**

This methodology covers **curated Prompt Manager skill discovery for planning purposes** and **how to attach the resulting skill setup to Plan Manager plans**.

**Does NOT cover:**
- **Full plan structure/template** - Plan Manager owns the structured plan and phase model; use `implementation-plan-authoring` for the authoring loop
- **Broad cross-corpus recall** - Use Search Hub directly or the Search Hub candidate from Plan Manager `context-discover`
- **CLI command governance** - Use CLI Health directly or the CLI Health candidate from Plan Manager `context-discover`
- **Action/tool discovery** - Use `prompt-manager discover --type all` or the Prompt Manager actions candidate from Plan Manager `context-discover`
- **Skill authoring** - Use `skill-authoring-practice` for creating new skills
- **Situational skill loading** - The AGENTS/CLAUDE dispatch table handles conversation-mode detection separately

---

### **6. Output Expectations**

When applying Plan Skill Discovery, you **must** produce:

1. **2-5 focused skill-discovery queries** based on work classification
2. **A broad Search Hub recall pass** or an explicit unavailable note
3. **A `prompt-manager discover --type skill` command run** with appropriate complexity
4. **A relevance assessment** of found skills
5. **Accepted Plan Manager `relevant_context` items** with setup commands, or an explicit no-context rationale
6. **Scenario cleanup and health verification criteria** when the plan touches a scenario
