---
name: "skill-validation"
description: "Judge a skill on two questions: is its contract true and executable, and is it cheap and well-conditioned. Divergence probe, efficiency and conditioning defects, severity-classified findings and a Prose Retirement Map."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["meta"]
  tags: ["skill","quality","validation","improvement"]
  icon: "check-circle"
  status: "active"
  revision: 50
  createdAt: "2026-01-30T11:19:51-05:00"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "prompt-manager skill-set validate", "prompt-manager action", "prompt-manager discover"]
  origin:
    kind: "authored"
---
## Meta focus: Skill quality

Judge **{{SKILL}}** on two questions: **is the contract true and executable** (§2), and **is it cheap and well-conditioned** (§3)? This one skill owns both; it replaced the separate validation and improvement-suggestion skills. The universal quality bars, the conditioning defect table (C1-C5), the promotion ladder and the `Prose Retirement Map` shape are owned by `docs/agent-system/SKILL_AUTHORING.md`; anchor each finding to the specific bar and restate none of it (R7). Read {{SKILL}} fully first. Label anything you could not check "unverified". Judge against the category guide in `skill-authoring`. The skill-set role checks in §4 apply to scenario roles.

Out of scope: implementing fixes, scenario feature work, and architecture debates. This skill produces findings and targeted patches, never a rewrite of the whole skill.

### 1. The contract

A skill has inputs (prerequisites, tools, files), behavior (steps and decision rules), outputs (artifacts, observable success), failure modes and constraints. Extract it as a capability map: for each claim the skill makes, record the primary path, the verification, the failure path and hidden prerequisites. A claimed capability with no path and verification is a **Gap** even if the prose is polished. A skill that cannot say what done looks like is unusable.

### 2. Is it true and executable

Run in order; each stage yields findings.

1. **Structure.** Check against the universal bars and the category guide. Contract skills (`modes[0]: contract`, machine-invoked workflow prompts) use the five-item checklist in `SKILL_AUTHORING.md` §"Contract skills", not the generic template; also cross-check the workflow node whose `promptRef` names the skill: every `{{.var}}` has a binding, and the skill restates none of the node's `resultSpec` (the engine renders it).
2. **Destination clarity** (audit-shaped steer skills only). Check the four ingredients in `SKILL_AUTHORING.md` §"Destination over direction". A skill that names a direction ("improve X") and no verifiable end state is at least **Major**; if a provider owns the maturity ladder, route to its output instead of copying it.
3. **Divergence probe. Run it, do not just judge.** Take the one to three instructions that most decide executor behavior. For each, try to produce two materially different executions that both comply with the text (different files, commands or acceptance checks). Success is a confirmed C4 and at least **Major**, anchored to the sentence that permits both. The patch is a decision, not more prose. Report the outcome even when no divergence is found.
4. **Internal consistency.** Contradictions in terms, flags, placeholders, ordering, or must/may/never. Where several approaches exist, the skill says when to use which.
5. **External reality.** Referenced skills exist (`prompt-manager skill read <id>`), commands and flags match `--help`, paths resolve, examples are copy-safe. **Contract bypass:** default human-friendly CLI output is the contract; a format switch, `jq`/`grep`/`sed`/`awk` scraping or shell extraction in a primary workflow needs a stated reason, an empty-output guard and a durable fix. **Capability leakage:** a primary workflow that depends on direct `curl`, OS glue or resource CLIs is a capability signal unless labeled an allowed exception (black-box public endpoint, minimal input file, one-off diagnostic with a promotion note).
6. **Verification and failure.** Each major workflow names an artifact, a status command or a deterministic condition. Red flags: "run X" with no next step, happy path only, "check the logs" with no interpretation. Long-tail failures live under `Troubleshooting & Edge Cases`; scattered, that is at least **Major** for a CLI-operational skill. A simple skill that says it has no edge cases needs no section.
7. **Safety.** Any step that can delete data, mutate production or expose secrets needs a warning, a safe default and a verification step; hardcoded credentials are **Critical**.
8. **Cross-skill coherence.** No conflicting directives with adjacent skills; a tools skill that prescribes architecture or a steer skill that prescribes exact commands should point to its sibling. Name a single source of truth and propose consolidate-or-reference.
9. **Promotion and retirement.** Scan `Troubleshooting & Edge Cases` first: a clarification that repeats is a promotion candidate. Classify each major instruction `Keep`, `Collapse to Action/CLI contract` or `Delete`, naming the prerequisite contract, and record it in the `Prose Retirement Map`.

### 3. Is it cheap and well-conditioned

A skill is a conditioning signal, not documentation. A 5,000-token skill read 100 times costs 500,000 input tokens; every sentence must earn its place. Walk {{SKILL}} against both tables. A confirmed C4 belongs to §2; C1, C2, C3 and C5 belong here.

| # | Efficiency defect | Signal | Direction |
|---|---|---|---|
| E1 | Multi-step process | temp files, tracked ids, five or more steps for one outcome | general-purpose tool capability that collapses the steps (never a situational flag) |
| E2 | Unintuitive step | "now do X" with no why | add the why and verification, or promote it into tool output |
| E3 | Unexplained concept | fails the fresh-agent test: could an agent that has never seen this skill follow it first try? | add path and example; delete implementation detail |
| E4 | Manual research a tool could do | "read the whole file to check" repeated | CLI search or read with filters |
| E5 | Length without decisions | padding, restated context | cut to decision-bearing text |
| E6 | Duplicates tool output | restates what the CLI prints | trust the tool; delete |
| E7 | Verbose command reference | more than five commands with flags | `<group> help` in the CLI; skill keeps a pointer |
| E8 | Async where sync works | polling and sleep prose | tool `--wait`; the skill becomes one line |
| E9 | Long-tail accretion | repeated similar troubleshooting entries | promote to CLI next-actions; delete the prose |
| E10 | Step below its reachable rung | an `[S0]`/`[S1]` leaf with deterministic pass or fail, or `[S1]` leaves that always run together | cite the Action or propose a contract program; count leaves per rung as a figure, not a sensor |

C1-C5: `SKILL_AUTHORING.md` §"Conditioning defect patterns" is the only home of the table. Name the defect row and the lens it improves.

A tool suggestion passes the generality test when it applies in at least two of: multiple scenarios, multiple skill types, multiple use cases. Check that no existing tool already does it. Suggestions are specific and executable ("section 3 says 'use the pattern' with no pattern; add this example"), never "be clearer". Name-and-delete beats name-and-keep: replace eight phrasing rules with "write requirements in EARS form" and delete what EARS already encodes. Do not cite a named standard you have not verified.

### 4. Skill-set role checks (scenario roles)

When {{SKILL}} is a role in a scenario's `service.json` `skills` block, run `prompt-manager skill-set validate <scenario>` first; it checks source presence, metadata markers and waiver fields only. Then check:

| Check | Severity |
|---|---|
| Role matches body: `usage` is a decision tree with rung-labelled leaves; `improve` has the eight sections `scenario-skill-setup` names, in order | Major |
| Every `[S3]` leaf names a program that exists and branches only on its declared vocabulary; every `[S1]` leaf names one real command | Major |
| Learning path declared once (automatic capture or the Memory skill), not restated | Major or Minor |
| Improve: targets reference approved intent; absent targets stay undecided; unavailable readings keep the contract's actual reason; invented target, fabricated sensor or invalid band | Critical |
| Improve with a sensor program: skill rows and bands agree with the exercised output | Major or Gap |
| Improve: dependency edits need the caller's grant and the dependency owner; sensor evidence is never mutation authority | Critical |
| Development entry: successive in-scope repairs need no new approval; target amendments stay protected | Major or Critical |
| Required unavailable rows stay unmet even when the program reports success | Critical |

### 5. Severity and routing

| Severity | Meaning |
|---|---|
| Critical | incorrect or unsafe instructions, broken commands, wrong paths |
| Major | ambiguous or inconsistent, no verification, missing essential steps; forcing the agent to guess is at least Major |
| Gap | claims a capability with no reliable path or playbook; repeated long-tail clarifications that should be promoted |
| Minor | typos, mild redundancy, decision-hiding words |
| Nice-to-have | quality upgrades not needed for correctness |

Critical gets a fix, Major a clarifying decision, Gap an expansion patch (replace a false instruction before adding guidance; never append an exception beneath a contradiction), efficiency and conditioning findings a suggestion. Prefer the smallest delta: add when missing, delete when a tool already covers it.

### 6. Report

`Summary` (two to four sentences: safe to use as is?), `Findings` (capability map; divergence probe table with the permitting sentence and the decision needed; contract-bypass and capability-leakage registers for CLI skills; findings by severity, each with a quoted evidence line, impact and a copy-paste patch for Critical, Major and Gap; efficiency and conditioning suggestions; Prose Retirement Map for CLI-operational skills), `Recommendations` (numbered, each with choices A, B, C, one marked **(Recommended)**, each with what, why and verification), `Notes` (unverified items, residual risks).
