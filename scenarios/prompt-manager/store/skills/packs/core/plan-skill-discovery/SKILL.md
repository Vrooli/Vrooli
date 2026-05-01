## Practice focus: Plan Skill Discovery

Methodology for finding and loading relevant prompt-manager skills before creating implementation plans. Ensures plans are informed by existing organizational knowledge and that executing agents can load the same context.

Required reading:
- `prompt-manager skill read skill-principles`

---

### **1. When to Use This Methodology**

Use Plan Skill Discovery when:
- Creating an implementation plan and the user has **not** provided specific skills
- You need to find domain-relevant skills before writing a plan
- An executing agent needs to know which skills to load
- An automated workshop agent needs to discover domain-relevant skills for a backlog item

This workflow intentionally uses skill-only discovery. It finds guidelines, methodologies, and required reading for implementation plans. Do not add `--type all` here unless prompt-manager's budget and read-command generation are redesigned so executable Actions cannot crowd out relevant steer/practice skills.

**Do NOT use** for:
- Routine task execution (skills should already be embedded in the plan)
- Quick one-step fixes with no planning phase
- Executing a plan that already contains Required Reading
- The user already provided specific skills to use

---

### **2. The Process**

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                      PLAN SKILL DISCOVERY PROCESS                            │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐          │
│   │ CLASSIFY │ ──▶ │  SEARCH  │ ──▶ │ CONFIRM  │ ──▶ │  EMBED   │          │
│   │  WORK    │     │          │     │RELEVANCE │     │ IN PLAN  │          │
│   └──────────┘     └────┬─────┘     └──────────┘     └──────────┘          │
│                         │                                                    │
│                    Too many results?                                          │
│                    ──▶ Narrow with tags                                       │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

### **Phase 1: Classify the Work**

**Entry criteria:** You are about to create an implementation plan.

**Actions:**
1. Decompose the work into distinct concepts (domain area, technology/stack, problem type). Each concept becomes its own search query. Use this decision tree to guide decomposition:

```
What kind of work is this?
├─ Bug/debugging        → concepts: "debugging", "<domain>", "<error type>"
├─ New feature/scenario → concepts: "<scenario-name>", "<domain>", "<technology>"
├─ Refactor/cleanup     → concepts: "refactor", "<area>", "<pattern>"
├─ Deployment/infra     → concepts: "deploy", "<target>", "<infrastructure>"
└─ Unsure               → concepts: "<goal description>", "<technology>", "<problem type>"
```

2. Formulate 2-5 focused search queries, one per concept.

**Exit criteria:**
- [ ] Work type identified
- [ ] 2-5 focused search queries formulated (one per concept)

---

### **Phase 2: Search**

**Entry criteria:** 2-5 focused search queries are ready.

**Actions:**
1. **Run unified discovery with all queries and a complexity level:**
   ```bash
   prompt-manager discover "concept1" "concept2" "concept3" --complexity moderate
   ```
   The `discover` command performs both topic search and skill search for each query,
   deduplicates results, sorts them (general topics first, then specific skills),
   and reports content size against the complexity budget.

2. **Choose the complexity that matches the work:**

   | Complexity | Char Budget | When to Use |
   |---|---|---|
   | `minor` | ~4,000 | Bug fix, small tweak |
   | `moderate` | ~8,000 | New feature, refactor |
   | `major` | ~12,000 | Multi-file feature, new endpoint |
   | `architectural` | ~18,000 | Cross-scenario, new system design |

3. **If over budget:** use the recommended (trimmed) read command from the output.
4. **If under budget with few results**, consider broadening search terms or adding more concepts.
5. **If you need to narrow results for a specific concept, use tag filtering:**
   ```bash
   prompt-manager search "<concept>" -tag testing
   ```

**Exit criteria:**
- [ ] Discovery command executed with appropriate complexity
- [ ] Budget status checked (under/over)
- [ ] Skill candidates identified from discover output

---

### **Phase 3: Confirm Relevance**

**Entry criteria:** Candidate skills found.

**Actions:**
1. **Read the candidates:**
   ```bash
   prompt-manager skill read <id-1> <id-2> -output combined
   ```
2. **Assess each candidate autonomously using these criteria:**
   - Does it apply to this specific task?
   - Would it materially improve the plan's quality?
   - Is it a Practice, Steer, Search, or Tools skill that the executing agent needs?
   - Can you articulate a specific way it will improve the plan?
3. **Apply confidence threshold:** Include a skill only if you can articulate a specific way it will improve the plan. Vague relevance ("might be useful") is insufficient — discard.

**Decision table:**

| Skill Relevance | Action |
|---|---|
| Directly applicable to the task | Include as required reading |
| Tangentially related | Discard |
| No relevant skills found | Proceed without — not every plan needs skills |

**Exit criteria:**
- [ ] Final skill list determined

---

### **Phase 4: Embed in Plan**

**Entry criteria:** Relevant skills confirmed.

**Actions:**
1. **Copy the pre-built read command** from the `discover` output into the plan:
   ```markdown
   ## Required Reading
   prompt-manager skill read <skill-1> <skill-2> <skill-3>
   ```
   If you trimmed for budget, use the recommended (trimmed) command instead.
2. This ensures any agent executing the plan loads the same context.
3. In automated contexts (e.g., workshop rounds), embed discovered skills directly into plan.md's Required Reading section.

**Exit criteria:**
- [ ] Plan contains Required Reading section with explicit read commands
- [ ] Executing agent can reproduce the same skill context

---

### **3. Mandatory Plan Conventions**

These conventions apply to **every** implementation plan, regardless of which skills were discovered. They counteract common coding-agent anti-patterns that consistently produce worse outcomes.

#### **3a. Greenfield by Default**

Unless the user explicitly requires backwards compatibility, every plan must state:

> **This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.

**Why:** Coding agents default to preserving backwards compatibility — adding shims, re-exporting removed types, and keeping dead code "just in case." In practice this creates tech debt and messy code. Explicitly stating greenfield up front produces cleaner results.

**When to omit:** Only when the user says the change must be backwards-compatible with existing consumers (e.g., a published API, shared library, or migration path).

#### **3b. Scenario Plans Must End with Cleanup + Health Verification**

Any plan that modifies a scenario must include a final step:

1. Fix **all** lint, type, and unit test issues in modified files — **including pre-existing ones**
2. Restart the scenario with `vrooli scenario restart <name>`
3. Verify the scenario is healthy (API health check, UI loads)

**Why:** Coding agents commonly encounter a lint or type error, conclude it was "pre-existing," and skip it. ~90% of the time the agent is wrong — the error is from their own changes. Even when truly pre-existing, fixing it improves codebase health. The explicit "even if pre-existing" phrasing is intentional — it removes the agent's escape hatch for skipping issues.

**Example plan step:**
```markdown
### Final: Cleanup & Verification
- Run type checking (`npx tsc --noEmit` or `go build ./...`) and fix ALL errors, even pre-existing
- Run linter (`eslint` / `golangci-lint run`) and fix ALL warnings in modified files
- Run unit tests and fix any failures
- `vrooli scenario restart <scenario-name>`
- Verify health: `curl -s http://localhost:<port>/health`
```

---

### **4. Anti-Patterns**

| Anti-Pattern | Why It Fails | Better Approach |
|---|---|---|
| **Searching during execution** | Skills should already be in the plan | Search during planning only |
| **Including every result** | Context bloat, irrelevant noise | Only include materially useful skills |
| **Skipping search entirely** | Miss organizational knowledge | Always search when planning |
| **Single broad query** | Poor recall across diverse concepts | Decompose into 2-5 focused queries, one per concept |
| **Asking user to confirm in automated contexts** | Blocks autonomous processing | Assess relevance autonomously using Phase 3 criteria |

---

### **5. Boundaries**

This methodology covers **skill discovery for planning purposes** and **mandatory conventions** that every plan must follow.

**Does NOT cover:**
- **Full plan structure/template** — Use implementation-plan-authoring for the 13-section format
- **Skill authoring** — Use skill-authoring-practice for creating new skills
- **Situational skill loading** — The CLAUDE.md dispatch table handles conversation-mode detection separately

---

### **6. Output Expectations**

When applying Plan Skill Discovery, you **must** produce:

1. **2-5 focused search queries** based on work classification
2. **A `discover` command run** with appropriate complexity level
3. **A relevance assessment** of found skills (informed by budget status)
4. **A Required Reading section** in the plan (even if empty — state "no relevant skills found")
5. **A greenfield statement** unless the user explicitly requires backwards compatibility
6. **A cleanup & verification step** if the plan modifies a scenario

You **should** also:
- Use the pre-built read command from `discover` output in the Required Reading section
- Respect the budget recommendation — trim to the recommended command if over budget
- Prefer specific skill IDs over broad categories in the Required Reading section
