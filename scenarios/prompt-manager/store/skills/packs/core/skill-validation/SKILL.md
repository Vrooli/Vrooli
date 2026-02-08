## Meta focus: Skill Validation

Analyze **{{SKILL}}** by running a structured “validation suite” that surfaces **issues, inconsistencies, and capability gaps**. This skill teaches how to *test* a skill like you’d test an API: extract its contract, probe its edge cases, and produce an actionable report that can be used to **fix and extend** the skill (i.e. expand its reliable capability surface).

Required reading:

* `prompt-manager skill read {{SKILL}}`
* `prompt-manager skill read skill-principles`

Optional reading (recommended follow-up after validation):

* `prompt-manager skill read skill-improvement-suggestions`

---

### **1. Why This Matters**

A skill is **crystallized operational knowledge**: it’s not just “documentation,” it’s a **contract** between the agent and the future.

When a skill is wrong, ambiguous, or incomplete, the failure doesn’t just happen once:

* It repeats across sessions
* It creates inconsistent decisions across agents
* It burns tokens on confusion and retries
* It erodes trust in the skill system itself

**Validation is quality assurance for intelligence.**
It’s the equivalent of unit tests + integration tests + contract tests for a mental model.

And it compounds:

* Fixing a single broken command example can prevent dozens of future tool failures
* Adding one missing “failure mode” section can eliminate whole classes of debugging loops
* Clarifying one ambiguous decision rule can keep the codebase from drifting for months

**Skill Validation expands capability surface.**
It identifies what’s missing or unreliable and proposes concrete expansions (new sections, new examples, missing guardrails, missing verification steps).
Then **Skill Improvement Suggestions** can streamline and optimize the expanded skill so it becomes easy and cheap to apply.

Think of them as complementary forces:

* **Validation**: “Make it true and complete.”
* **Improvement Suggestions**: “Make it sharp and efficient.”

---

### **2. Category Scope**

**In scope:**

* Testing whether {{SKILL}} is **internally consistent**
* Testing whether {{SKILL}} is **externally executable** (commands, flags, paths, references)
* Identifying **capability gaps** (things the skill claims or implies, but doesn’t actually enable)
* Identifying **missing guardrails** (verification steps, stop conditions, safety constraints)
* Identifying **broken references** to other skills/tools/concepts
* Producing **expansion patches**: copy-pastable additions that close gaps and fix correctness issues

**Out of scope (handoff to other processes):**

* Full rewrites for token efficiency or tone (see **Skill Improvement Suggestions**)
* Implementing new CLI tools or changing tool behavior (that’s a tool PR / improvement suggestion)
* Scenario-specific feature work (belongs in scenario PRDs / issues)
* Deep architectural strategy debates (belongs in Steer skills, not a validation pass)

---

### **3. The Validation Mindset: Treat Skills Like APIs**

A reliable skill has an implicit contract:

* **Inputs**: prerequisites, context, assumptions, required tools, required files
* **Behavior**: steps, decision rules, “when to do what”
* **Outputs**: expected artifacts, observable success conditions, what to document
* **Failure modes**: what can go wrong and how to debug it safely
* **Constraints**: what you must not do, and why

**Validation means extracting this contract and checking it for:**

* **Correctness** (is it true?)
* **Completeness** (does it cover what it claims to cover?)
* **Consistency** (does it contradict itself?)
* **Executability** (can a real agent run this without guessing?)
* **Safety** (does it prevent foot-guns?)

If you can’t write down the contract clearly, the skill probably can’t be applied reliably.

---

### **4. The Skill Validation Suite**

Run these checks in order. Each stage produces findings you’ll later convert into a structured report.

#### **4.1 Structure Conformance Check (Skill Principles Compliance)**

Every skill should have the universal quality bars from **Skill Principles**:

* [ ] **Intent statement** (1–2 sentences at the top)
* [ ] **Scope boundaries** (In scope / Out of scope)
* [ ] **Convergence patterns** when choices must be consistent (decision tree / table / rules)
* [ ] **Output expectations** (what may/must/must not change or produce)

**Common failure:**

* Skill has “good content” but lacks explicit boundaries → agents over-apply it.
* Skill has steps but no convergence rules → different agents do different things.

**Expansion patch pattern (copy-paste style):**

```markdown
### Scope Boundaries

**In scope:**
- ...

**Out of scope:**
- ...
```

#### **4.2 Contract Extraction & Capability Map**

Make the contract explicit by building a capability map:

1. List the **top-level claims** the skill makes (“This skill lets you…”)
2. For each claim, record:

   * **Primary path**: the minimal steps to achieve the outcome
   * **Verification**: how you know it worked (observable pass/fail)
   * **Artifacts**: what files/logs/output should exist afterward
   * **Failure path**: what to do if it fails
   * **Hidden prerequisites**: any assumptions that aren’t stated

**Capability Map Template:**

```markdown
| Capability | Primary Path Exists? | Verification Exists? | Failure/Debug Path Exists? | Notes |
|-----------|-----------------------|----------------------|----------------------------|------|
| ...       | Yes/No                | Yes/No               | Yes/No                     | ...  |
```

**Gap detection rule:**
If a capability is claimed or implied but lacks a primary path + verification, that is a **capability gap** even if the skill is “well written.”

#### **4.3 Internal Consistency Audit (No Contradictions, No Drift)**

Look for contradictions in:

* Terminology (same concept, different names)
* Flags/commands (two different ways claimed as “the” way)
* Paths / placeholders (e.g. `{{TARGET}}` vs `{TARGET}` vs `<TARGET>`)
* Ordering (“Do X before Y” in one section, reversed elsewhere)
* “Must” vs “May” vs “Never” rules that conflict

**Concrete example of the kind of issue you should surface:**

* A skill may say “Parameters must be nested in `initial_params` via `--params`” in one place,
  but later show a command using a different flag like `--initial-params` without explaining the relationship.
  That’s either:

  * a real inconsistency (bug), or
  * a missing explanation (gap),
    but either way it fails validation because it forces guesswork.

**Consistency checklist:**

* [ ] One placeholder convention used consistently
* [ ] One canonical term per concept (with synonyms explicitly mapped if needed)
* [ ] If multiple approaches exist, the skill explains *when to use which* (decision table)

#### **4.4 External Reality Check (Tooling & References Actually Exist)**

A skill can be internally consistent and still be wrong in the real world.

Validate:

* **Referenced skills exist** (and the IDs match)

  * Use: `prompt-manager skill read <skill-id>` for each referenced skill ID
* **Commands and flags are plausible and consistent**

  * Where possible, validate against `--help`, `schema`, `lint`, or “list” commands
* **Paths resolve** (relative vs absolute rules are explained)
* **Examples are copy-paste safe**

  * Balanced quotes, correct escaping, no missing braces, placeholders are obvious
* **Parser dependency is justified**

  * Prefer default CLI output over parser pipelines (`--json`, `--raw`, `jq`)
  * Treat unnecessary parser pipelines as at least **Major** because they increase drift and break risk
  * If parser usage is required, require an explicit justification and a guard for empty/changed output

**Example executability test pattern:**

```bash
# If the skill shows a command with flags, sanity check it:
<tool> --help
<tool> schema --help
<tool> lint --help
```

If you can’t verify against tool help, label it explicitly as **“unverified”** in the report, not as a fact.

#### **4.5 Verification & Observability Check (Success Must Be Observable)**

A skill is not truly usable if it can’t tell you what “done” looks like.

For each major workflow, ensure it includes at least one of:

* A file to inspect (`README.md`, logs, output JSON)
* A command that returns a clear status
* A deterministic condition (“element exists,” “tests pass,” “build succeeds”)

**Red flag patterns:**

* “Run X” with no next step
* “Verify it works” with no method
* “Should be fine” language

**Expansion patch pattern:**

```markdown
### Verification

After completing the workflow:
- Check: ...
- Success looks like: ...
- If not: go to Debugging section ...
```

#### **4.6 Failure Modes & Debug Playbook Check**

Every mature skill needs a failure playbook proportional to its complexity.

Minimum expectation:

* A list of likely failure types
* A debugging order (what to check first vs later)
* A small number of “most common fixes”
* A clear distinction between symptoms vs root causes

**Red flags:**

* Only happy paths exist
* Debugging section is just “check logs” without interpretation
* No prioritization (agents flail)

**Expansion patch pattern:**

```markdown
### Common Failure Modes

| Symptom | Likely Cause | First Check | Fix |
|--------|--------------|------------|-----|
| ...    | ...          | ...        | ... |
```

#### **4.7 Safety & Guardrail Check**

Even non-security skills can create dangerous behavior if they normalize unsafe patterns.

Check for:

* Steps that could delete data / modify production / expose secrets
* Instructions that encourage hardcoding credentials, tokens, or personal data
* Missing “do not” constraints for dangerous tools
* Missing warnings around irreversible operations

**Guardrail rule:**
If a step is risky, the skill should include:

* a warning,
* a safe default,
* and a verification step.

#### **4.8 Cross-Skill Coherence Check**

Skills are a system. Validation includes checking coherence with:

* **Skill Principles** (structure, boundaries)
* Other referenced or adjacent skills (no conflicting directives)
* Category intent (Steer vs Tools vs Search vs Meta)

**Typical coherence failures:**

* A Tools skill starts prescribing architecture (should reference Steer skill instead)
* A Steer skill prescribes exact commands (should reference Tools skill instead)
* Two skills define the same concept differently (drift)

When you find cross-skill conflicts:

* Identify the conflicting statements
* Recommend a “single source of truth” location
* Propose a patch: either consolidate or reference

---

### **5. Severity Model (Triage Like a Production System)**

Classify every finding:

| Severity         | Definition                                                                                    | Why it matters                         |
| ---------------- | --------------------------------------------------------------------------------------------- | -------------------------------------- |
| **Critical**     | Skill gives incorrect instructions, broken commands, wrong paths, or unsafe guidance          | Causes direct failure or harm          |
| **Major**        | Skill is ambiguous, internally inconsistent, missing verification, or missing essential steps | Causes frequent mis-execution          |
| **Gap**          | Skill claims/implies capability but lacks a reliable path or playbook                         | Limits applicability; forces guesswork |
| **Minor**        | Typos, mild redundancy, small clarity issues                                                  | Annoying but not blocking              |
| **Nice-to-have** | Quality upgrades not required for correctness                                                 | Handoff to Improvement Suggestions     |

**Rule:** Treat “forces the agent to guess” as at least **Major**.

---

### **6. Convergence Pattern: What To Do With Each Finding**

Use this decision tree so validation results are consistent:

```
Finding found
  │
  ├─ Is it factually wrong, unsafe, or breaks execution?
  │      → Critical → Patch required (fix)
  │
  ├─ Is it contradictory or ambiguous (forces guessing)?
  │      → Major → Patch required (clarify)
  │
  ├─ Is a capability claimed/implied but not actually enabled?
  │      → Gap → Expansion patch required (extend)
  │
  └─ Is it mostly about efficiency/wording/streamlining?
         → Nice-to-have → Hand off to Skill Improvement Suggestions
```

This prevents “validation reports” from turning into opinionated rewrites.

---

### **7. Expansion Patches: How to Fix and Extend Without Rewriting**

Validation should produce **targeted, copy-paste expansions** that close gaps with minimal disruption.

High-leverage expansion patch types:

1. **Missing prerequisites section**
2. **Missing verification section**
3. **Missing failure mode table**
4. **Missing decision table (“when to use which approach”)**
5. **Missing output contract (what artifacts or results exist)**
6. **Missing “when NOT to use this” boundaries**

**Patch style rules:**

* Small, local edits that can be applied surgically
* Prefer adding a section over rewriting large paragraphs
* When clarifying inconsistency, define a single canonical way and explicitly list alternatives if they truly exist

---

### **8. Category-Specific Validation Checklists**

Different skill categories fail in different ways. Validate accordingly.

#### **Steer skills**

* [ ] Decision rules are explicit and consistent (tables/trees)
* [ ] Constraints are clear (what must not change)
* [ ] Examples are aligned with the claimed architecture
* [ ] No tool-flag micromanagement (belongs in Tools skills)
* [ ] Clear “migration patterns” when refactors are expected

#### **Tools skills**

* [ ] Every major workflow has a runnable example
* [ ] Verification steps exist (what to check after running)
* [ ] Debug playbook exists for common errors
* [ ] Risky operations have guardrails
* [ ] Placeholder conventions are consistent (`{{TARGET}}`, paths, env vars)
* [ ] Parser pipelines are avoided by default; exceptions are explicit and justified

#### **Search skills**

* [ ] Output contract exists (format, evidence level, stop conditions)
* [ ] Clear “done criteria” (when to stop searching)
* [ ] Bias toward primary sources where relevant
* [ ] Includes negative paths (“if you can’t find X, do Y”)
* [ ] Avoids “read everything” token sinks without filters

#### **Meta skills**

* [ ] Governance surface is explicit (what it controls)
* [ ] Avoids duplicating rules from other Meta skills
* [ ] Has clear decision rules for ambiguous cases
* [ ] Explicit boundaries (what it does *not* govern)
* [ ] References **Skill Principles** where appropriate

---

### **9. Output Expectations**

When validating **{{SKILL}}**, you must:

* Read {{SKILL}} fully before reporting
* Extract and present a capability map
* Classify findings by severity
* Include evidence (quotes/snippets) for each issue
* Provide **expansion patches** for Critical/Major/Gap findings (copy-pastable)
* Separate validation findings from optimization suggestions

You may:

* Recommend follow-up usage of **Skill Improvement Suggestions**
* Suggest where a tool verification step *should* exist (even if you can’t run it)
* Mark items as “unverified” if you cannot confirm externally

You must NOT:

* Rewrite the entire skill as part of validation
* Hide uncertainty (label unverified things explicitly)
* Turn validation into a stylistic critique
* Propose scenario-specific feature work as “skill validation”

---

### **10. Report Format**

When analyzing {{SKILL}}, produce this structured report:

````markdown
# Skill Validation Report for {{SKILL}}

## Summary
[2–4 sentences: overall reliability, main risk areas, whether it’s safe to use as-is]

## Capability Map
| Capability | Primary Path Exists? | Verification Exists? | Failure/Debug Path Exists? | Notes |
|-----------|-----------------------|----------------------|----------------------------|------|
| ...       | Yes/No                | Yes/No               | Yes/No                     | ...  |

## Findings

### Critical
- **[Title]**
  - Evidence: “...quote...”
  - Why it fails validation: ...
  - Impact: ...
  - **Patch (copy-paste):**
    ```markdown
    ...minimal patch...
    ```

### Major
[Same structure]

### Gaps
[Same structure, but focus on missing capability paths/playbooks]

### Minor / Nice-to-have
[Brief list]

## Cross-Skill Coherence Notes
- References validated: [list what you checked]
- Conflicts found: [if any]

## Recommended Next Step
- If primarily correctness/gaps: apply patches above.
- If primarily efficiency/streamlining: run **Skill Improvement Suggestions** next.
````

---

### **11. Remember**

A validated skill is not “perfect prose.”
A validated skill is **reliable operational truth**: consistent, executable, observable, and safe.

Your job is to make it hard for future agents to misunderstand, guess, or fail silently.

Validation is how the skill system earns trust.
