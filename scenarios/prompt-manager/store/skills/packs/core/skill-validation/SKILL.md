## Meta focus: Skill Validation

Analyze **{{SKILL}}** by running a structured “validation suite” that surfaces **issues, inconsistencies, and capability gaps**. This skill teaches how to *test* a skill like you’d test an API: extract its contract, probe its edge cases, and produce an actionable report that can be used to **fix and extend** the skill (i.e. expand its reliable capability surface).

Required reading:

* `prompt-manager skill read {{SKILL}}`
* `docs/agent-system/SKILL_AUTHORING.md`
* `docs/agent-system/PROMOTION_LADDER.md`
* `docs/agent-system/LAYERS.md`

Optional reading (recommended follow-up after validation):

* `prompt-manager skill read skill-improvement-suggestions`

---

### **1. Why This Matters**

Skill validation keeps operational guidance reliable.

* It catches correctness and contract drift before they spread.
* It reduces guesswork by enforcing explicit verification and failure paths.
* It distinguishes immediate correctness fixes from optimization follow-up.
* For CLI-operational skills, it enforces the promotion-retirement lifecycle from `docs/agent-system/PROMOTION_LADDER.md`.

---

### **2. Category Scope**

**In scope:**

* Testing whether {{SKILL}} is **internally consistent**
* Testing whether {{SKILL}} is **externally executable** (commands, flags, paths, references)
* Detecting **CLI output contract bypass** (examples that bypass default human-friendly CLI output contracts via format flags, parsing pipelines, or shell extraction)
* Detecting **capability leakage** (core workflows that rely on non-Vrooli tools like direct API calls, OS-specific glue, scripts, or direct resource CLIs)
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

Additional structure check for skills with operational CLI workflows:
* [ ] Core workflow remains readable without rigid over-structuring
* [ ] If operational complexity exists, a dedicated `Troubleshooting & Edge Cases` section exists
* [ ] Long-tail failures are centralized in that section (not scattered through primary workflow)

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

#### **4.1.1 Destination-Clarity Check (Audit-Shaped Skills)**

Applies to **audit-shaped skills** — skills that examine an existing scenario and produce findings/recommendations against a defined target state (the steer audit cohort: architecture, temporal flow, API, CLI, interop, storage, testing-seams, etc.). See `docs/agent-system/SKILL_AUTHORING.md` §"Destination over direction: maturity ladders for audit-shaped skills" for the canon definition.

Check:

* [ ] Skill declares a **named maturity ladder** (L0–L5 or similar) with a **verifiable artifact** per level (`ls`/`grep`/CLI-checkable) — not adjectives like "improved" or "more idiomatic".
* [ ] Each level has "What exists" and "When to stop here" columns (or equivalent).
* [ ] Findings are routed to **durable docs** (`ARCHITECTURE.md`, `SEAMS.md`, `PROBLEMS.md`) via `knowledge-observatory-tools`, **not** to standalone `*_AUDIT.md` files.

**Failure mode — "fuzzy destination":** the skill describes a *direction* ("improve X", "consider Y", "make it more idiomatic") but never names a verifiable end state two agents would converge on. Classify as at least **Major** for audit-shaped skills; missing destination clarity blocks promotion (see `docs/agent-system/PROMOTION_LADDER.md` and `development-toolchain-validator` P1-005 "Skill Maturity Score" — the programmatic reading of this check).

Reference exemplars to cite when patching:
* `scenarios/prompt-manager/store/skills/packs/core/temporal-flow-audit/SKILL.md` §2 "Temporal Maturity Model"
* `scenarios/prompt-manager/store/skills/packs/core/screaming-architecture-audit/SKILL.md` §2 "Architecture Maturity Model"

Not applicable to Tools, Search, Practice, or greenfield-directive steer skills.

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
* **Contract bypass and parsing are justified (human-first output is canonical)**

  * Prefer default human-friendly CLI output for primary workflows.
  * Treat contract bypass in primary workflows as suspicious unless explicitly justified:
    * Format switches: `--json`, `--raw`, `--format json`, `--quiet` (when it hides actionable next steps)
    * Parsing/scraping: `jq`, `grep`, `sed`, `awk`, `cut`, `tr`, regex scraping, `head`/`tail`
    * Shell extraction: `$(...)`, backticks, and similar output-to-input coupling
  * If bypass is required, the skill must:
    * justify why default output is insufficient (too long, ambiguous, missing required data),
    * include guardrails for empty/changed output,
    * recommend a durable fix: improve default CLI output contract or add a small CLI capability (for example: `--print <field>`).
* **Non-Vrooli tooling in primary workflows is justified (capability leakage audit)**

  * Prefer Vrooli CLIs for core operations so work benefits from scenario testing, lifecycle safety, and output contract control.
  * Treat non-Vrooli commands in a primary workflow as a capability signal (usually at least **Gap**) unless explicitly justified:
    * Direct API calls for core operations (for example: `curl https://.../api/v1/...`)
    * OS-specific glue or bespoke scripts (bash/python/systemctl)
    * Direct resource CLIs for routine workflows (`psql`, `redis-cli`, etc.)
  * Allowed exceptions must be labeled explicitly:
    * Black-box verification against public endpoints (for example: `curl` to confirm a deployed update URL responds)
    * Minimal file creation required to provide CLI input (`cat > /tmp/payload.json`)
    * One-off diagnostics when a Vrooli CLI capability is missing (must include a tool promotion recommendation)

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

* A file or artifact to inspect (`README.md`, logs, generated outputs)
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

For skills with operational CLI complexity, validate placement:
- Failure tables, rare gotchas, diagnostics, and manual recovery should live under `Troubleshooting & Edge Cases`.
- If these details are spread through the main workflow and force context switching, treat as at least **Major** (maintainability + execution risk).
- If the skill is simple and explicitly states no meaningful edge cases, do not force section expansion.

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

#### **4.6.1 Troubleshooting-First Promotion Pass (CLI-Operational Skills)**

For skills with operational CLI workflows, run a short promotion pass before proposing prose-heavy edits:

1. Scan `Troubleshooting & Edge Cases` entries first.
2. For each repeated or high-friction entry, decide:
   - promote to CLI output contract improvement,
   - promote to a tool capability improvement,
   - or keep as a rare/manual playbook item.
3. Use this pass to prioritize durable fixes before adding more skill text.

Guidance:
- This is a convergence aid, not a rigid format requirement.
- If the same troubleshooting clarification appears multiple times, treat it as a product-signal candidate (usually at least **Gap**).

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

#### **4.9 Complexity Retirement Pass (Required for CLI-Operational Skills)**

Apply the canonical lifecycle from `docs/agent-system/PROMOTION_LADDER.md`.

For each major gate/workflow instruction in {{SKILL}}, classify:
- `Keep` (policy/safety/ownership boundary; should remain in skill)
- `Collapse to CLI contract` (replace detailed prose with command + output contract expectation)
- `Delete` (fully superseded by durable CLI/tool contract)

For `Collapse`/`Delete`, include:
- prerequisite CLI/tool contract improvement (or existing contract evidence),
- the exact skill area to compress/remove,
- a residual risk note (if any).

Classification rule:
- If the instruction describes volatile operational mechanics already emitted by CLI output, default to `Collapse` or `Delete`, not expansion.
- If the instruction encodes durable policy or safety constraints, default to `Keep`.

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

Contract integrity rules:
- **Major**: Primary workflow bypasses default human-friendly CLI output contracts (format switches, parsing/scraping, shell extraction) without explicit justification and guardrails.
- **Gap**: Skill’s core promise requires non-Vrooli tools/commands for routine execution (capability leakage). Prefer a Vrooli CLI/tool promotion recommendation over expanding prose workarounds.
- **Critical**: Out-of-band steps normalize unsafe behavior (secrets exposure, destructive ops, production mutation) without warnings and verification.

Structure rule for CLI-operational skills:
- For operationally complex skills that rely on CLI workflows, missing `Troubleshooting & Edge Cases` or scattering long-tail clarifications outside it should be classified as **Major**.
- Repeated long-tail clarifications that should be promoted to CLI/tooling should be classified as **Gap** and handed off to Skill Improvement Suggestions.
- If repeated troubleshooting clarifications are found, include at least one durable CLI/tool conversion recommendation in addition to any interim skill patch.

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
* Prefer the smallest reliable delta: add when missing, but collapse/delete prose when CLI/tool contracts already cover it
* When clarifying inconsistency, define a single canonical way and explicitly list alternatives if they truly exist

---

### **8. Category-Specific Validation Checklists**

Use short, category-specific delta checks here. Baseline authoring structure belongs to:
- `docs/agent-system/SKILL_AUTHORING.md`
- `skill-authoring`
- `skill-authoring-tools`
- `skill-authoring-search`
- `skill-authoring-meta`

#### **Steer skills**

* [ ] Decision rules are explicit and consistent (tables/trees)
* [ ] No tool-command micromanagement that belongs in Tools skills
* [ ] Architecture guidance is coherent with referenced Steer skills

#### **Tools skills**

* [ ] Decision rules are explicit and consistent for cross-tool orchestration
* [ ] Troubleshooting is centralized and promotion-ready (`Troubleshooting & Edge Cases`)
* [ ] CLI-operational guidance follows promotion-retirement lifecycle (not one-way prose growth)

#### **Search skills**

* [ ] Evidence contract and stop conditions are explicit and enforceable
* [ ] Failure paths avoid token-sink “read everything” behavior

#### **Meta skills**

* [ ] Has clear decision rules for ambiguous cases
* [ ] Governance rules do not duplicate canonical policy from `docs/agent-system/SKILL_AUTHORING.md` or other PoR files under `path:docs/agent-system/`
* [ ] Meta guidance defines boundaries and ownership clearly

---

### **9. Output Expectations**

When validating **{{SKILL}}**, you must:

* Read {{SKILL}} fully before reporting
* Produce the report in the format defined below: `Summary` -> `Findings` -> `Recommendations` -> `Notes`
* Extract and present a capability map (inside `Findings`)
* For any skill that includes CLI commands, identify contract bypass and capability leakage (if any) and classify them (inside `Findings`)
* Classify findings by severity
* Include evidence (quotes/snippets) for each issue
* Provide **expansion patches** for Critical/Major/Gap findings (copy-pastable)
* Separate validation findings from optimization suggestions
* For CLI-operational skills, include a concise troubleshooting-promotion analysis (what should move to CLI/tooling vs remain manual)
* For CLI-operational skills, include a required `Complexity Retirement` section with `Keep/Collapse/Delete` decisions
* In `Recommendations`, provide numbered recommendations with choices (`A`, `B`, `C`...) and mark one choice as **(Recommended)**
* Each recommendation choice must include enough detail to execute: what to do, why, verification, and any required copy-paste patch

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

## Findings

### Capability Map
| Capability | Primary Path Exists? | Verification Exists? | Failure/Debug Path Exists? | Notes |
|-----------|-----------------------|----------------------|----------------------------|------|
| ...       | Yes/No                | Yes/No               | Yes/No                     | ...  |

### Contract Integrity (Required When {{SKILL}} Includes CLI Commands)

#### Contract Bypass Register
| Command / Step | Bypass Type | Why It’s A Bypass | Allowed? | Notes / Fix |
|---|---|---|---|---|
| ... | `--json` / `--raw` / parsing / shell extraction | ... | Yes/No | ... |

#### Non-Vrooli Dependency Register (Capability Leakage)
| Non-Vrooli Command / Step | Purpose | Why It’s Needed | Vrooli Replacement | Notes |
|---|---|---|---|---|
| ... | ... | ... | Existing or proposed CLI/tool contract | ... |

### Validation Findings

#### Critical
[Findings, if any]

#### Major
[Findings, if any]

#### Gap
[Findings, if any]

#### Minor / Nice-to-have
[Brief list]

Finding template (use for each `Critical` / `Major` / `Gap` item):

### [Severity]: [Title]
- Evidence: “...quote...”
- Why it fails validation: ...
- Impact: ...
- **Patch (copy-paste):**
  ```markdown
  ...minimal patch...
  ```

### Cross-Skill Coherence Notes
- References validated: [list what you checked]
- Conflicts found: [if any]
- Promotion candidates from `Troubleshooting & Edge Cases`: [items to convert into CLI/tool improvements]

### Durable vs Interim Fixes
- Durable CLI/tool conversion candidates: [high-leverage fixes that reduce repeated troubleshooting prose]
- Interim skill-level guardrails: [minimal wording/structure patches needed now]

### Complexity Retirement (CLI-Operational Skills)
| Skill Instruction / Gate | Decision (Keep/Collapse/Delete) | Rationale | Prerequisite Contract | Risk |
|---|---|---|---|---|
| ... | ... | ... | ... | ... |

### Complexity Signals
- Gate/step count: [before -> after if patches applied]
- Troubleshooting item count: [before -> after if patches applied]
- Net effect: [reduced / neutral / increased] with rationale

## Recommendations

Selection hint:
- You should be able to choose a path like `R1A, R2B, R3A`.

### R1: [Recommendation title]

**A (Recommended):** [Choice summary]
- What to do: ...
- Why: ...
- Verification: ...
- Patch (copy-paste) / Tooling change: ...

**B:** [Choice summary]
- What to do: ...
- Why: ...
- Verification: ...
- Patch (copy-paste) / Tooling change: ...

**C:** [Choice summary]
- What to do: ...
- Why: ...
- Verification: ...
- Patch (copy-paste) / Tooling change: ...

Repeat for each recommendation (`R2`, `R3`...).

## Notes
- Unverified items: ...
- Residual risks: ...
- Suggested follow-ups (optional): run **Skill Improvement Suggestions**, open tool backlog item(s), etc.
````

---

### **11. Remember**

A validated skill is reliable operational truth: consistent, executable, observable, and safe.
