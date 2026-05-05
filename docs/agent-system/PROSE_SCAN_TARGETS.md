# Prose Scan Targets

**Status:** canon. The plan-of-record for Pillar 2 of topic validation. Anchor doc for the `prose_topic_leak` validator rule (P2.1) and the writer-skill `writes_to[]` registry (P2.2). Pairs with `TOPICS_SCHEMA.md` (declarations) and `RUNTIME_ATTRIBUTION.md` (receipts).

This document defines **what files the scanner walks, what patterns it detects, how owners are derived from file paths, and how each finding maps back to a declaration the writer should have made**. It is the input the P2.1 scanner consumes — its file-path globs, regex set, and severity guidance are normative; changes here are decision-gated.

---

## Why a prose pillar at all

P1 (`topics.json`) catches declaration mismatches. P3 (runtime attribution) catches observed writes that nobody declared. Neither pillar can see the **prose layer** — the markdown an agent reads on every heartbeat that tells it what to do. When prose says one thing and `topics.json` says another, the agent follows the prose and the declared graph silently rots.

Concrete failure mode this pillar exists to catch:

- A member's `RESPONSIBILITIES.md` instructs `prompt-manager team knowledge-add marketing-crew --topic="campaign-draft/<slug>"`, but the member's `topics.json::output[]` declares `campaign/*`. The agent will write to `campaign-draft/*`; the validator (P1) sees nothing wrong (the declaration is internally consistent); the runtime scanner (P3) eventually fires `actual_writer_undeclared` after the write happens. P2 catches it before the first wrong write — at the source of the confusion — by surfacing the prose/declaration mismatch.

P1 is the plan. P2 is the prose adjacent to the plan. P3 is the receipts. Each catches drift the others cannot.

---

## Scan targets

The scanner walks every file matching the rules below. Each target carries an **owning context** (a member, an agent, a skill, or the cross-cutting `docs/` namespace) that determines which `topics.json` declarations its references must align with.

### Per-member prose

Path: `scenarios/prompt-manager/store/teams/<team>/members/<member>/`

| File | Included | Owner | Notes |
|---|---|---|---|
| `RESPONSIBILITIES.md` | ✅ | `<team>/<member>` | Authoritative description of what the member does; references must align with the member's own `topics.json`. |
| `HEARTBEAT.md` | ✅ | `<team>/<member>` | Per-heartbeat instructions rendered into the prompt every tick; same alignment as RESPONSIBILITIES.md. |
| `last-handoff.md` | ❌ | n/a | Ephemeral run state; not authority. Excluded. |
| `topics.json` | n/a | the declaration itself | Not prose; consumed by P1 directly. |
| `heartbeat.json` | n/a | machine config | Not prose. |
| `logs/` | ❌ | n/a | Per-run output. Excluded. |

### Per-team prose

Path: `scenarios/prompt-manager/store/teams/<team>/`

| File | Included | Owner | Notes |
|---|---|---|---|
| `shared/TEAM.md` | ✅ | `<team>` (cross-member) | Team-level operating rules; references should align with **some** member's `topics.json` on the team. |
| `shared/<other>.md` (e.g. `AGING_SCAN.md`, `RUNTIME_LESSONS.md`, plan-of-record audits) | ✅ | `<team>` (cross-member) | Same alignment as TEAM.md — references must resolve to a topic declared by some member of the team. |
| `*.md` at the team root | ✅ if any exist | `<team>` | None today; defensive coverage. |
| `team.json`, `roles.json`, `org.json` | n/a | machine config | Not prose. |
| `shared/*.jsonl` | n/a | data files | Not prose; consumed by P3. |

### Per-agent prose (identity templates)

Path: `scenarios/prompt-manager/store/agents/<agent-id>/`

| File | Included | Owner | Notes |
|---|---|---|---|
| `SOUL.md` | ✅ | `agent:<agent-id>` | Identity prose; should describe the agent's purpose without hardcoding team-specific topic strings. Topic references here are usually drift (the same agent template binds to multiple teams). |
| `AGENTS.md` | ✅ | `agent:<agent-id>` | Workflow contract; same alignment rationale as SOUL.md. |
| `TOOLS.md` | ✅ | `agent:<agent-id>` | Tool/skill bindings. May reference topic prefixes when documenting `team knowledge-*` invocations the agent issues. The bindings are team-bound at runtime; references should be portable (`<team>` placeholder) or qualified to a known team. |
| `agent.json` | n/a | machine config | Not prose. |

**Owner-derivation note:** an agent identity template is **not** itself bound to a team. References in agent prose are checked against the union of `topics.json` declarations across every team that binds this agent (via `store/teams/<team>/members/<agent-id>/topics.json` existence). If the reference matches a declaration on **any** binding team, the reference is clean. If it matches none, the finding fires under the agent's owner key.

### Per-skill prose

Path: `scenarios/prompt-manager/store/skills/packs/<pack>/<skill-id>/`

Skill scanning is **conditional on skill kind** — the prose layer is held to a different bar depending on whether the skill is allowed to know about topics:

| Skill kind | Detection rule | Treatment |
|---|---|---|
| **Writer skill** — `skill.json::tags` contains `writer-skill` | `report-bug`, `report-friction`, `morning-vision-walk` (after P2.2 tagging); see § The writer-skill set | `SKILL.md` is **scanned**. Topic references must be a subset of the skill's `writes_to[]` declaration in `skill.json` (P2.2). Mismatch → `prose_topic_leak`. |
| **Classifier skill** — id matches `*-classifier` or `*-triage`, e.g. `marketing-signal-classifier`, `monetization-signal-classifier`, `market-validation-triage` | Hand-curated list lives at `validation.go::classifierForbiddenSubstrings` until P4.0 retires it | `SKILL.md` is **scanned with stricter rule** — *any* topic reference (declared or not) is forbidden, because classifier skills must be portable across teams. Existing rule is `non_portable_classifier` (P1); P2.1 subsumes it via P4.0's subsumption proof. |
| **Generic skill** (everything else; ~140 of 144 skills) | Default | `SKILL.md` is **scanned** with the classifier-skill rule (any topic reference is a finding) — generic skills should not embed team-specific topic strings. P4.0 ratifies this. |
| **Pack-level docs** (e.g. `local/`, `drafts/` packs) | Default | Same as generic skills. |

**The writer-skill set** as of P2.0 inventory:

| Skill id | `writer-skill` tag in `skill.json`? | Effective today? | P2.2 action |
|---|---|---|---|
| `report-bug` | ✅ tagged | ✅ writes via `team knowledge-add scenario-qa --topic=bug-inbox/...` | populate `writes_to[]` with `bug-inbox/*` |
| `report-friction` | ✅ tagged | ✅ writes via `team knowledge-add meta-optimization --topic=friction-inbox/...` | populate `writes_to[]` with `friction-inbox/*` |
| `morning-vision-walk` | ❌ not yet tagged | ✅ writes to `research-inbox/*`, `opportunity-inbox/*`, `validation-inbox/*`, `vision-walk/*` across multiple teams | P2.2 must add `writer-skill` tag and populate `writes_to[]` with all four prefixes |

P2.2 reconciles the gap: every skill that effectively writes must be tagged `writer-skill` and carry a `writes_to[]` declaration. Until then, `morning-vision-walk` is on the wishlist below — its `SKILL.md` will fire `prose_topic_leak` findings until P2.2 lands its tag and writes_to set, which is the intended pressure.

### Domain documentation

Path: `docs/<domain>/**/*.md` (e.g. `docs/agent-system/`, `docs/marketing/`, `docs/monetization/`)

| Inclusion rule | Owner | Notes |
|---|---|---|
| All `.md` files under `docs/` | `docs:<domain>` (first path segment after `docs/`) | Topic references must resolve to **some** declaration somewhere in the system. Cross-team references are permitted (these are operator-facing PoR docs). |
| `docs/agent-system/` canon docs (this file, RUNTIME_ATTRIBUTION.md, TOPICS.md, TOPICS_SCHEMA.md, INTAKE_PIPELINE.md, PRIMITIVES.md, README.md) | `docs:agent-system` | These docs **describe** the topic system, so they reference example prefixes pedagogically. The scanner respects fenced code blocks (see § Code-block exclusion); examples inside ` ```jsonc ` and similar fences are ignored. Backticked references in body prose still get checked but only at warning severity. |
| `docs/agent-system/drafts/` and `docs/agent-system/_outline.md` | excluded | Working notebooks; not authority. |

---

## Excluded paths

The scanner skips these explicitly. Each exclusion has a reason; relaxing one requires a decision.

| Path / pattern | Reason for exclusion |
|---|---|
| `**/last-handoff.md` | Ephemeral run state. |
| `**/logs/**` | Per-run output. |
| `**/coverage/**`, `**/dist/**`, `**/node_modules/**`, `**/__pycache__/**` | Build / dependency output. |
| `scenarios/prompt-manager/store/teams/*/shared/*.jsonl` | Data files, not prose. P3 handles these. |
| `scenarios/prompt-manager/store/teams/*/shared/*.jsonl.backup` | Migration artifacts. |
| `docs/agent-system/drafts/**` | Working drafts; not canon. |
| `docs/agent-system/_outline.md` | Working outline. |
| `**/*.json`, `**/*.json.backup`, `**/*.go`, `**/*.tsx`, `**/*.ts` | Not prose; type-system or P1 handles these. |
| Files larger than 1 MB | Defense against accidental binary inclusion. |

The scanner's globbing is **explicit-include-only** — anything outside the targets enumerated above is silently skipped. Adding a new scan target is a decision: PR to this file, then update the scanner.

---

## Canonical CLI patterns

Topic prefixes appear in prose almost exclusively as arguments to the prompt-manager CLI. The scanner's regex set is scoped tightly to the **`team knowledge-*` family** — the only commands whose `--topic` and `--topic-prefix` arguments are knowledge topic prefixes.

### Discriminator: `team knowledge-*` only

`prompt-manager team` exposes other `--topic` arguments that are **not** knowledge topic prefixes and must not match:

| Command | `--topic` semantic | Scanner behavior |
|---|---|---|
| `team knowledge-add` | knowledge topic prefix | **MATCH** |
| `team knowledge-list` | exact knowledge topic | **MATCH** (alongside `--topic-prefix`) |
| `team knowledge-update` | new knowledge topic | **MATCH** |
| `team knowledge-delete` | (no `--topic` flag) | n/a |
| `team decision-add` | decision title (free-form prose, e.g. `"What is being decided"`) | **SKIP** |
| `team decision-update` | decision title | **SKIP** |

The discriminator is the literal substring `team knowledge-` immediately preceding the verb on the same logical command line.

### Pattern set (P2.1 normative)

| Pattern id | Regex (Go RE2 syntax) | Matches | Severity |
|---|---|---|---|
| `cli-knowledge-add-topic` | ` `prompt-manager team knowledge-add\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?` ` | `prompt-manager team knowledge-add <team> --topic="audience-scan/2026-05-04/q2-creators"` | **error**-eligible (P4.1 promotes from warning) |
| `cli-knowledge-list-topic` | ` `prompt-manager team knowledge-list\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?` ` | `prompt-manager team knowledge-list marketing-crew --topic="campaign-draft/q2"` | **error**-eligible |
| `cli-knowledge-list-prefix` | ` `prompt-manager team knowledge-list\b[^\n]*?--topic-prefix[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)*/?)"?` ` | `prompt-manager team knowledge-list marketing-crew --topic-prefix=audience-scan/` | **error**-eligible |
| `cli-knowledge-update-topic` | ` `prompt-manager team knowledge-update\b[^\n]*?--topic[= ]"?([a-z][a-z0-9-]*(?:/[a-z0-9<>_*-]+)+)"?` ` | `prompt-manager team knowledge-update marketing-crew knw-abc --topic="audience-scan/keep"` | **error**-eligible |
| `backtick-topic-ref` | `` `([a-z][a-z0-9-]*/[a-z0-9<>_*/-]+)` `` (a backticked string with at least one `/`, lower-kebab segments, optional `<>` placeholders, `*` wildcard) | `` `audience-scan/<date>/<slug>` ``, `` `bug-inbox/regression/cli-flag-confusion` `` | **warning** (looser; high false-positive risk) |

Captured group `1` is the topic prefix. The scanner treats segments containing `<...>` placeholders or trailing `*` as wildcards when joining against declarations (e.g., `audience-scan/<date>/<slug>` joins against the declared `audience-scan/*` output prefix).

### What the scanner does **not** match

- Topic strings inside fenced code blocks (` ``` ... ``` ` or ` ```lang ... ``` `) in `docs/agent-system/` files. See § Code-block exclusion.
- Topic strings inside HTML comments (`<!-- ... -->`).
- URL paths that happen to contain slashes (`https://example.com/a/b/c` does not look like a topic prefix because it has scheme-like prefixes; the regex requires lower-kebab segment shape and the absence of `://`).
- Identifier strings without `/` (e.g. `audience-scan`) — bare prefixes are too generic to attribute.
- Decision contexts and skill ids — these have their own validators.

### Code-block exclusion

`docs/agent-system/*.md` files describe the system pedagogically and contain many example topic prefixes inside fenced code blocks. The scanner respects markdown's fenced-code-block syntax:

- A line whose stripped content equals ` ``` ` or matches ` ```<lang> ` opens a code block.
- The next line beginning with ` ``` ` closes it.
- Patterns inside the open/close pair are skipped entirely.

`docs/agent-system/` is scanned with code-block exclusion enabled. Other targets (member prose, agent prose, writer-skill SKILL.md, non-`docs/agent-system/` docs) are scanned **without** code-block exclusion — those files have no pedagogical-example use case. This is the only target-conditional scanner setting.

Backticked-string references (`backtick-topic-ref` pattern) remain at **warning** severity globally, even outside code blocks, because backticks are also used for non-topic identifiers (file paths, code symbols) and the regex over-matches.

---

## Cross-reference matrix

This is the matrix the P2.1 scanner consumes when joining a detected reference back to a declaration. For each (target × pattern) combination, the validation question and the consulted declaration set are:

| Target | Detected pattern | Validation question | Declaration consulted |
|---|---|---|---|
| `members/<id>/RESPONSIBILITIES.md` `members/<id>/HEARTBEAT.md` | any `cli-knowledge-*` pattern | "Does this member's own `topics.json` declare a write/read for this prefix?" | The member's `output[]` (for `knowledge-add` / `knowledge-update`); the union of `intake[] ∪ required_read[] ∪ evidence_consumed[]` (for `knowledge-list`). |
| `members/<id>/*.md` | `backtick-topic-ref` | "Is this prefix declared by **some** team member?" | Team-wide union of all members' `topics.json` declarations (warning severity). |
| `shared/TEAM.md` `shared/<other>.md` | any `cli-knowledge-*` pattern | "Is this prefix declared by some member of this team?" | Team-wide union of all members' `topics.json` declarations. |
| `shared/<other>.md` | `backtick-topic-ref` | same | same (warning severity). |
| `agents/<id>/SOUL.md` `agents/<id>/AGENTS.md` `agents/<id>/TOOLS.md` | any `cli-knowledge-*` pattern | "Is this prefix declared by **some** team that binds this agent?" | Union of `topics.json` declarations across every `store/teams/<team>/members/<id>/topics.json` matching this agent id. |
| `agents/<id>/*.md` | `backtick-topic-ref` | same | same (warning severity). |
| `skills/packs/<pack>/<id>/SKILL.md` (writer skill) | any `cli-knowledge-*` pattern | "Is this prefix in this skill's `writes_to[]`?" | The skill's `skill.json::writes_to[]` (P2.2). |
| `skills/packs/<pack>/<id>/SKILL.md` (writer skill) | `backtick-topic-ref` | same | same (warning severity). |
| `skills/packs/<pack>/<id>/SKILL.md` (classifier or generic skill) | any pattern (CLI or backtick) | "Are there ANY topic references at all? (There must not be.)" | None — every match is a finding. |
| `docs/<domain>/**/*.md` | any `cli-knowledge-*` pattern | "Is this prefix declared **anywhere** in the system?" | Global union of all members' `topics.json` declarations across all teams. |
| `docs/<domain>/**/*.md` | `backtick-topic-ref` | same | same (warning severity). |
| `docs/agent-system/*.md` | any pattern, **inside fenced code block** | n/a (excluded by code-block rule) | n/a |

**No-match outcome:** the scanner emits a `prose_topic_leak` finding with the file path, line number, the captured prefix, the owner key, and the consulted declaration set's hash (so the operator can reproduce). Severity is per the matrix above; warnings flow to `findings.json` (P3.7) but don't fail CI; errors do (after P4.1 promotes the CLI patterns).

**Special case — writer skill that has no `writes_to[]` yet:** if the SKILL.md is in the writer-skill set but `skill.json::writes_to[]` is missing or empty, every CLI-pattern match is a `prose_topic_leak` finding pointing the reader at P2.2. This is the pressure that gets `morning-vision-walk` tagged.

---

## Severity guidance

| Rule lifecycle stage | Severity for `cli-knowledge-*` | Severity for `backtick-topic-ref` |
|---|---|---|
| **Initial ship (P2.1)** | warning | warning |
| **Post-bake-in (P4.1)** | error | warning (kept at warning permanently — too lossy to enforce) |

`prose_topic_leak` follows the standard Pillar rule lifecycle: ship at warning, bake for one observation cycle, promote to error after the ecosystem is clean and the warning count has dropped to zero. The `backtick-topic-ref` pattern is **not promoted** — its false-positive rate (file paths, code symbols, slashed identifiers that happen to look like topic prefixes) makes it a perpetual hint rather than a CI gate.

---

## Owner-derivation rules

The scanner derives an owner key from each scanned file's path. The owner key controls which declaration set is consulted (per the cross-reference matrix) and is the unit by which findings are grouped in `findings.json`.

```
scenarios/prompt-manager/store/teams/<team>/members/<member>/<file>.md
  -> owner: "team:<team>/<member>"

scenarios/prompt-manager/store/teams/<team>/shared/<file>.md
  -> owner: "team:<team>"

scenarios/prompt-manager/store/teams/<team>/<file>.md (top-level)
  -> owner: "team:<team>"

scenarios/prompt-manager/store/agents/<agent-id>/<file>.md
  -> owner: "agent:<agent-id>"

scenarios/prompt-manager/store/skills/packs/<pack>/<skill-id>/SKILL.md
  -> owner: "skill:<skill-id>"

docs/<domain>/<...>.md
  -> owner: "docs:<domain>"
```

The owner string appears verbatim in the `Finding.OwnerKey` field. CI summary scripts group by owner.

---

## Inventory snapshot (P2.0)

Captured at P2.0 against the live store — this is the size the P2.1 scanner is built for, not a hard cap.

| Surface | Count |
|---|---|
| Teams | 6 (`director-swarm`, `infra-health`, `marketing-crew`, `meta-optimization`, `monetization`, `scenario-qa`) |
| Members | 29 |
| Per-member files scanned (RESPONSIBILITIES.md + HEARTBEAT.md) | 58 |
| Per-team shared files scanned | 18 (TEAM.md ×6 + meta-optimization audits ×9 + infra-health snapshots ×3) |
| Agent identity templates (SOUL.md + AGENTS.md + TOOLS.md per agent) | 28 agents × 3 files = 84 |
| Skills total | 144 |
| Writer-skill `SKILL.md` (scanned with `writes_to[]` consultation) | 2 today (`report-bug`, `report-friction`); 3 after P2.2 tags `morning-vision-walk` |
| Classifier / generic skill `SKILL.md` (scanned with strict no-topic rule) | 142 (post-P2.2: 141) |
| `docs/<domain>/**/*.md` files | varies; `docs/agent-system/` alone has ~17 canon files |

The scanner's expected runtime is well under one second on this corpus; budget concerns belong to a later workshop, not this one.

---

## Out-of-scope (intentional)

These patterns explicitly do **not** appear in the scanner. Each requires a decision to add.

- **Decision-context references** (e.g., `dec-1234` ids in prose). Decisions have their own validators and a different referential surface.
- **Skill-id references** (e.g., `marketing-signal-classifier` mentioned in member prose). Skill bindings are validated via member `topics.json::intake[].classifier_skill`; bare skill-id mentions in prose are descriptive and not gated.
- **Cross-scenario CLI commands** (e.g., `swarm-manager backlog add`). Other scenarios have their own knowledge surfaces; prompt-manager's prose scanner does not police them.
- **Prose claims about *who writes where*** beyond CLI invocations. Claims like "the researcher publishes audience scans" are caught indirectly when the natural rewrite is the CLI invocation; pure prose claims that never crystallize as a CLI command stay out of scope (no reliable detection regex).
- **Topic strings inside JSON files** (`*.json`). These are P1's surface and validated structurally.

---

## Migration plan reference

Pillar 2 ships across four phases:

- **P2.0** (this document) — Scan-target inventory + canonical CLI patterns. No code change; pure documentation.
- **P2.1** — `ruleProseTopicLeak` implementation in `scenarios/prompt-manager/api/memberflow/prose_scan.go`. Consumes this doc's pattern set, target list, and owner-derivation rules verbatim.
- **P2.2** — Writer-skill `writes_to[]` registry in `skill.json`. Adds the field, populates the three writer skills, and tags `morning-vision-walk` as `writer-skill`.
- **P2.3** — Golden-fixture tests covering the scanner's failure modes.
- **P4.0** — Subsumption proof for `non_portable_classifier` (the legacy P1 rule covering classifier-skill topic-purity); demonstrates strict-superset coverage by P2.1, then deletes the legacy rule.
- **P4.1** — Promotes `cli-knowledge-*` `prose_topic_leak` findings from warning to error. `backtick-topic-ref` stays at warning permanently.

See `/home/matthalloran8/.claude/plans/keen-growing-whisper.md` for the full plan and per-phase definition-of-done criteria. This file is the contract for what the scanner reads; that file is the rollout for when it ships.
