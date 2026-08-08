## Meta focus: Skill Validation

Analyze **{{SKILL}}** by running a structured "validation suite" that surfaces **issues, inconsistencies, and capability gaps**. Validation asks one question: **is the contract true and executable?** The sibling skill `skill-improvement-suggestions` asks the complementary question — is the contract cheap and well-conditioned. The routing rule between them is canon (`docs/agent-system/SKILL_AUTHORING.md` §"Conditioning defect patterns"): validation keeps exactly one conditioning check — the divergence probe (§3.3) — because a demonstrated divergence is an executability defect; C1, C2, C3, and C5 findings hand off to `skill-improvement-suggestions`.

Required reading:

* `prompt-manager skill read {{SKILL}}`
* `docs/agent-system/SKILL_AUTHORING.md`
* `docs/agent-system/PROMOTION_LADDER.md`
* `docs/agent-system/LAYERS.md`

Optional reading (recommended follow-up after validation):

* `prompt-manager skill read skill-improvement-suggestions`

---

### 1. Category Scope

**In scope:**

* Testing whether {{SKILL}} is **internally consistent**
* Testing whether {{SKILL}} is **externally executable** (commands, flags, paths, references)
* Running the **divergence probe** (does the text condition one execution or several?)
* Detecting **CLI output contract bypass** (examples that bypass default human-friendly CLI output via format flags, parsing pipelines, or shell extraction)
* Detecting **capability leakage** (core workflows that rely on non-Vrooli tools: direct API calls, OS-specific glue, scripts, direct resource CLIs)
* Identifying **capability gaps** (things the skill claims or implies, but doesn't actually enable)
* Identifying **missing guardrails** (verification steps, stop conditions, safety constraints)
* Identifying **broken references** to other skills/tools/concepts
* Producing **expansion patches**: copy-pastable additions that close gaps and fix correctness issues

**Out of scope (handoff to other processes):**

* Efficiency and conditioning-cost findings — C1/C2/C3/C5 per canon, token cost, rewrites for tone (hand off to **skill-improvement-suggestions**)
* Implementing new CLI tools or changing tool behavior (that's a tool PR / improvement suggestion)
* Scenario-specific feature work (belongs in scenario PRDs / issues)
* Deep architectural strategy debates (belongs in Steer skills, not a validation pass)

---

### 2. The Contract

A reliable skill has an implicit contract: **inputs** (prerequisites, assumptions, required tools and files), **behavior** (steps and decision rules), **outputs** (artifacts, observable success conditions), **failure modes**, and **constraints**. Validation extracts this contract and checks it for correctness, completeness, consistency, executability, convergence, and safety. If you cannot write the contract down clearly, the skill cannot be applied reliably.

---

### 3. The Skill Validation Suite

Run these checks in order. Each stage produces findings you'll later convert into a structured report.

#### 3.1 Structure Conformance Check

Check {{SKILL}} against the universal quality bars in `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars" and against the matching per-category authoring guide (`skill-authoring` for Steer; `skill-authoring-tools`, `-search`, `-practice`, `-platform`, `-meta` otherwise). The authoring guide **is** the category checklist — anchor each finding to the specific bar or guide rule it violates; do not restate either.

**Contract skills** (`modes[0]: contract` — machine-invoked workflow prompts) use a different checklist: validate against the five required-content items and the exemption list in `SKILL_AUTHORING.md` §"Contract skills: machine-invoked workflow prompts" instead of the generic structure template. §3.2, §3.8, and §3.11 do not apply. The divergence probe (§3.3) is the primary gate and targets the outcome work table; probe specifically for end states where an affirmative row is arguably-but-not-provably satisfied — the skill must resolve these to the conservative outcome per SKILL_AUTHORING §"The conservative-branch default", and a probe that finds an optimistic-vs-conservative split unresolved by that default is a confirmed C4. Additionally cross-check the referencing workflow declaration (the node whose `promptRef` names {{SKILL}}): every `{{.var}}` in the skill has a matching node binding, every node binding is referenced by the skill or deliberately omitted, and the skill restates no part of the node's `resultSpec` schema — the engine renders that schema into the prompt; restated shape is drift.

Two placement checks specific to validation:

* [ ] Core workflow remains readable without rigid over-structuring
* [ ] Long-tail failures are centralized in `Troubleshooting & Edge Cases` (not scattered through the primary workflow)

**Common failure:**

* Skill has "good content" but lacks explicit boundaries → agents over-apply it.
* Skill has steps but no convergence rules → different agents do different things.

#### 3.2 Destination-Clarity Check (Audit-Shaped Skills)

Applies to **audit-shaped skills** as defined in `docs/agent-system/SKILL_AUTHORING.md` §"Destination over direction: maturity sources for audit-shaped skills". Check {{SKILL}} against the four mandatory ingredients in that section — anchor each finding to the specific ingredient it violates; do not restate the ingredients here.

**Failure mode — "fuzzy destination":** the skill describes a *direction* ("improve X", "consider Y", "make it more idiomatic") but never names a verifiable end state two agents would converge on. Classify as at least **Major** for audit-shaped skills; missing destination clarity blocks promotion (see `docs/agent-system/PROMOTION_LADDER.md`; `development-toolchain-validator` OT-P1-002 "Skill Maturity Score" is the *planned* programmatic reading of this check). If a provider owns the maturity ladder, duplicating that ladder in the skill is drift risk; route to provider output instead.

Reference exemplars to cite when patching: `docs/agent-system/SKILL_AUTHORING.md` §"Canonical exemplars".

Not applicable to Tools, Search, Practice, or greenfield-directive steer skills.

#### 3.3 Divergence Probe

**Run it, do not just judge:** pick the 1–3 instructions in {{SKILL}} that most determine executor behavior. For each, attempt to produce **two materially different executions that both comply with the text** (different files touched, different commands run, different acceptance checks). If you succeed, that is a confirmed C4 (canon §"Conditioning defect patterns") and at least a **Major** finding, anchored to the exact sentence that permits both readings. The patch is a *decision*, not more prose — an ambiguous sentence usually marks a decision the author never made. Models demonstrate ambiguity more reliably than they rate it; always attempt the probe before declaring the skill unambiguous.

If the probe surfaces C1/C2/C3/C5 patterns along the way, record a one-line pointer in the report's handoff section — do not patch them here.

#### 3.4 Contract Extraction & Capability Map

Make the contract explicit by building a capability map. List the **top-level claims** the skill makes ("This skill lets you…") and for each record: **primary path** (minimal steps), **verification** (observable pass/fail), **artifacts** (files/logs/output that should exist afterward), **failure path**, and **hidden prerequisites**.

**Capability Map Template:**

```markdown
| Capability | Primary Path Exists? | Verification Exists? | Failure/Debug Path Exists? | Notes |
|-----------|-----------------------|----------------------|----------------------------|------|
| ...       | Yes/No                | Yes/No               | Yes/No                     | ...  |
```

**Gap detection rule:**
If a capability is claimed or implied but lacks a primary path + verification, that is a **capability gap** even if the skill is "well written."

#### 3.5 Internal Consistency Audit

Look for contradictions in:

* Terminology (same concept, different names)
* Flags/commands (two different ways claimed as "the" way)
* Paths / placeholders (e.g. `{{TARGET}}` vs `{TARGET}` vs `<TARGET>`)
* Ordering ("Do X before Y" in one section, reversed elsewhere)
* "Must" vs "May" vs "Never" rules that conflict

Example: a skill says "Parameters must be nested in `initial_params` via `--params`" in one place, but later shows a command using `--initial-params` without explaining the relationship. That is either a real inconsistency (bug) or a missing explanation (gap) — either way it fails validation because it forces guesswork.

**Consistency checklist:**

* [ ] One placeholder convention used consistently
* [ ] One canonical term per concept (with synonyms explicitly mapped if needed)
* [ ] If multiple approaches exist, the skill explains *when to use which* (work table)

#### 3.6 External Reality Check (Tooling & References Actually Exist)

A skill can be internally consistent and still be wrong in the real world. Validate:

* **Referenced skills exist** (and the IDs match): `prompt-manager skill read <skill-id>` for each referenced ID
* **Commands and flags are real**: validate against `--help`, `schema`, `lint`, or "list" commands where possible
* **Paths resolve** (relative vs absolute rules are explained)
* **Examples are copy-paste safe**: balanced quotes, correct escaping, obvious placeholders

If you can't verify against tool help, label it explicitly as **"unverified"** in the report, not as a fact.

**Contract bypass check (human-first output is canonical):** default human-friendly CLI output is the contract for primary workflows. Treat format switches (`--json`, `--raw`, `--format json`, `--quiet` that hides next steps), parsing/scraping (`jq`, `grep`, `sed`, `awk`, `cut`, `head`/`tail`, regex), and shell extraction (`$(...)`, backticks) in a primary workflow as suspicious unless the skill justifies why default output is insufficient, includes guardrails for empty/changed output, and recommends a durable fix (improve the default output contract or add a small CLI capability such as `--print <field>`).

**Capability leakage check:** prefer Vrooli CLIs for core operations so work benefits from scenario testing, lifecycle safety, and output contract control. Treat non-Vrooli commands in a primary workflow (direct `curl` to APIs, OS-specific glue or bespoke scripts, direct resource CLIs like `psql`/`redis-cli`) as a capability signal unless explicitly labeled as an allowed exception: black-box verification against public endpoints, minimal file creation for CLI input (`cat > /tmp/payload.json`), or one-off diagnostics accompanied by a tool promotion recommendation.

#### 3.7 Verification & Observability Check

A skill is not usable if it can't tell you what "done" looks like. Each major workflow must include at least one of: a file or artifact to inspect, a command that returns a clear status, or a deterministic condition ("element exists," "tests pass," "build succeeds").

**Red flag patterns:** "Run X" with no next step; "Verify it works" with no method; "Should be fine" language.

**Expansion patch pattern:**

```markdown
### Verification

After completing the workflow:
- Check: ...
- Success looks like: ...
- If not: go to Debugging section ...
```

#### 3.8 Failure Modes & Debug Playbook Check

Every mature skill needs a failure playbook proportional to its complexity. Minimum expectation: a list of likely failure types, a debugging order (what to check first vs later), the most common fixes, and a clear distinction between symptoms and root causes.

For skills with operational CLI complexity, validate placement: failure tables, rare gotchas, diagnostics, and manual recovery live under `Troubleshooting & Edge Cases`. If these details are spread through the main workflow and force context switching, treat as at least **Major**. If the skill is simple and explicitly states no meaningful edge cases, do not force section expansion.

**Red flags:** only happy paths exist; debugging is just "check logs" without interpretation; no prioritization (agents flail).

**Expansion patch pattern:**

```markdown
### Common Failure Modes

| Symptom | Likely Cause | First Check | Fix |
|--------|--------------|------------|-----|
| ...    | ...          | ...        | ... |
```

#### 3.9 Safety & Guardrail Check

Check for: steps that could delete data / modify production / expose secrets; instructions that encourage hardcoding credentials, tokens, or personal data; missing "do not" constraints for dangerous tools; missing warnings around irreversible operations.

**Guardrail rule:** if a step is risky, the skill must include a warning, a safe default, and a verification step.

#### 3.10 Cross-Skill Coherence Check

Skills are a system. Check coherence with `docs/agent-system/SKILL_AUTHORING.md`, with referenced or adjacent skills (no conflicting directives), and with category intent per `PRIMITIVES.md` (Steer, Platform, Search, Tools, Practice, Meta).

**Typical coherence failures:** a Tools skill prescribing architecture (should reference a Steer skill); a Steer skill prescribing exact commands (should reference a Tools skill); two skills defining the same concept differently (drift).

When you find cross-skill conflicts: identify the conflicting statements, recommend a single-source-of-truth location, and propose a patch — consolidate or reference.

#### 3.11 Promotion & Retirement Pass (Required for CLI-Operational Skills)

`docs/agent-system/PROMOTION_LADDER.md` owns the lifecycle, the retirement and retention criteria, and — in §"Output requirement for meta analyses" — the mandate to classify each major workflow instruction as `Keep` / `Collapse to Action/CLI contract` / `Delete`, recorded in the canonical `Prose Retirement Map` table defined there. Do not restate the lifecycle; apply it.

Procedure:

1. Scan `Troubleshooting & Edge Cases` first. A clarification that repeats is a promotion candidate (usually at least **Gap**) — promote to a CLI output contract or tool capability before adding prose.
2. Classify each major gate/workflow instruction per the canon output requirement. For `Collapse`/`Delete`, name the prerequisite CLI/tool contract (or existing contract evidence) and any residual risk.
3. Record results in the report's `Prose Retirement Map`. Durable CLI/tool fixes rank above interim prose patches.

---

### 4. Severity Model

Classify every finding:

| Severity         | Definition                                                                                    | Why it matters                         |
| ---------------- | --------------------------------------------------------------------------------------------- | -------------------------------------- |
| **Critical**     | Skill gives incorrect instructions, broken commands, wrong paths, or unsafe guidance          | Causes direct failure or harm          |
| **Major**        | Skill is ambiguous, internally inconsistent, missing verification, or missing essential steps | Causes frequent mis-execution          |
| **Gap**          | Skill claims/implies capability but lacks a reliable path or playbook                         | Limits applicability; forces guesswork |
| **Minor**        | Typos, mild redundancy, small clarity issues                                                  | Annoying but not blocking              |
| **Nice-to-have** | Quality upgrades not required for correctness                                                 | Handoff to Improvement Suggestions     |

**Rule:** treat "forces the agent to guess" as at least **Major**. A confirmed divergence-probe finding (§3.3) is a proven instance of forced guessing.

Contract integrity severities:

- **Major**: primary workflow bypasses the default human-friendly CLI output contract without explicit justification and guardrails (§3.6).
- **Gap**: the skill's core promise requires non-Vrooli tools for routine execution (capability leakage, §3.6) — prefer a Vrooli CLI/tool promotion recommendation over prose workarounds. Repeated long-tail clarifications that should be promoted to CLI/tooling are also **Gap** — hand off with at least one durable conversion recommendation.
- **Critical**: out-of-band steps normalize unsafe behavior (secrets exposure, destructive ops, production mutation) without warnings and verification.
- **Major**: missing `Troubleshooting & Edge Cases` for a CLI-operational skill, or long-tail clarifications scattered outside it.
- **Minor**: decision-hiding words in rule text (canonical list: `docs/agent-system/SKILL_AUTHORING.md` §"Universal quality bars").

---

### 5. Convergence Pattern: What To Do With Each Finding

```
Finding found
  │
  ├─ Is it factually wrong, unsafe, or breaks execution?
  │      → Critical → Patch required (fix)
  │
  ├─ Is it contradictory or ambiguous (forces guessing — incl. confirmed C4)?
  │      → Major → Patch required (clarify — the patch is a decision, not more prose)
  │
  ├─ Is a capability claimed/implied but not actually enabled?
  │      → Gap → Expansion patch required (extend)
  │
  └─ Is it about efficiency or conditioning cost (C1/C2/C3/C5 per canon)?
         → Hand off to Skill Improvement Suggestions
```

This prevents validation reports from turning into opinionated rewrites.

---

### 6. Expansion Patches: Fix and Extend Without Rewriting

Validation produces **targeted, copy-paste expansions** that close gaps with minimal disruption. High-leverage patch types: missing prerequisites section; missing verification section; missing failure mode table; missing work table ("when to use which approach"); missing output contract; missing "when NOT to use this" boundaries.

**Patch style rules:**

* Small, local edits that can be applied surgically
* Prefer the smallest reliable delta: add when missing, but collapse/delete prose when CLI/tool contracts already cover it
* When clarifying inconsistency, define a single canonical way and explicitly list alternatives if they truly exist

---

### **7. Output Expectations**

When validating **{{SKILL}}**, you must:

* Read {{SKILL}} fully before reporting
* Produce the report in the format defined below: `Summary` -> `Findings` -> `Recommendations` -> `Notes`
* Extract and present a capability map (inside `Findings`)
* Run the divergence probe (§3.3) and report its outcome — including "no divergence found on probed instructions"
* For any skill that includes CLI commands, identify contract bypass and capability leakage (if any) and classify them (inside `Findings`)
* Classify findings by severity
* Include evidence (quotes/snippets) for each issue
* Provide **expansion patches** for Critical/Major/Gap findings (copy-pastable)
* Separate validation findings from the handoff list (conditioning-cost and efficiency observations for `skill-improvement-suggestions`)
* For CLI-operational skills, include the `Prose Retirement Map` per §3.11
* In `Recommendations`, provide numbered recommendations with choices (`A`, `B`, `C`...) and mark one choice as **(Recommended)**
* Each recommendation choice must include enough detail to execute: what to do, why, verification, and any required copy-paste patch

You may:

* Recommend follow-up usage of **Skill Improvement Suggestions**
* Suggest where a tool verification step *should* exist (even if you can't run it)
* Mark items as "unverified" if you cannot confirm externally

You must NOT:

* Rewrite the entire skill as part of validation
* Hide uncertainty (label unverified things explicitly)
* Turn validation into a stylistic critique
* Propose scenario-specific feature work as "skill validation"

---

### 8. Report Format

When analyzing {{SKILL}}, produce this structured report:

````markdown
# Skill Validation Report for {{SKILL}}

## Summary
[2–4 sentences: overall reliability, main risk areas, whether it's safe to use as-is]

## Findings

### Capability Map
| Capability | Primary Path Exists? | Verification Exists? | Failure/Debug Path Exists? | Notes |
|-----------|-----------------------|----------------------|----------------------------|------|
| ...       | Yes/No                | Yes/No               | Yes/No                     | ...  |

### Divergence Probe
| Probed Instruction | Divergent Readings Found? | Permitting Sentence | Decision Needed |
|---|---|---|---|
| ... | Yes/No | "...quote..." | ... |

### Contract Integrity (Required When {{SKILL}} Includes CLI Commands)

#### Contract Bypass Register
| Command / Step | Bypass Type | Why It's A Bypass | Allowed? | Notes / Fix |
|---|---|---|---|---|
| ... | `--json` / `--raw` / parsing / shell extraction | ... | Yes/No | ... |

#### Non-Vrooli Dependency Register (Capability Leakage)
| Non-Vrooli Command / Step | Purpose | Why It's Needed | Vrooli Replacement | Notes |
|---|---|---|---|---|
| ... | ... | ... | Existing or proposed CLI/tool contract | ... |

### Validation Findings

#### Critical
[Findings, if any]

#### Major
[Findings, if any]

#### Gap
[Findings, if any]

#### Minor
[Brief list]

Finding template (use for each `Critical` / `Major` / `Gap` item):

### [Severity]: [Title]
- Evidence: "...quote..."
- Why it fails validation: ...
- Impact: ...
- **Patch (copy-paste):**
  ```markdown
  ...minimal patch...
  ```

### Handoff to Skill Improvement Suggestions
- [One-line pointers: conditioning-cost (C1/C2/C3/C5) and efficiency observations, each anchored to a section]

### Cross-Skill Coherence Notes
- References validated: [list what you checked]
- Conflicts found: [if any]
- Promotion candidates from `Troubleshooting & Edge Cases`: [items to convert into CLI/tool improvements]

### Prose Retirement Map (CLI-Operational Skills)
[Canonical table shape: `docs/agent-system/PROMOTION_LADDER.md` §"Output requirement for meta analyses"]

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
- ...

Repeat for each recommendation (`R2`, `R3`...).

## Notes
- Unverified items: ...
- Residual risks: ...
- Suggested follow-ups (optional): run **Skill Improvement Suggestions**, open tool backlog item(s), etc.
````
