---
name: "report-friction"
description: "Universal writer skill for filing a friction observation — system-level capture-leak in tooling, runs, storage, or recurring workarounds. Writes a friction-inbox/<scope>/<slug> entry on the meta-optimization team for the friction-curator to drain and route to the appropriate scoped friction topic."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill","observability","friction-report","writer-skill"]
  writes_to: ["friction-inbox/*"]
  icon: "alert-triangle"
  status: "active"
  revision: 4
  createdAt: "2026-05-03T00:00:00Z"
  updatedAt: "2026-10-03T00:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager", "prompt-manager team"]
  origin:
    kind: "authored"
---
## Tools focus: Report Friction

Universal writer skill any agent on any team may invoke when they observe friction — something that was missing, broken, confusing, slow, undocumented, or harder than it should have been. The skill writes a structured entry to `team:meta-optimization`'s `topic:friction-inbox/<scope>/<slug>` topic; the `literal:meta-optimization/friction-curator` member drains the inbox, classifies the scope, and routes to the appropriate scoped friction topic owned by an existing meta-optimization sub-member.

This skill is **destination-coupled by design** — writer skills always are. The portability rule (`prose_topic_leak`) applies to classifier skills, not to writers. Friction reporting is a one-way producer pattern.

This is the sister to `report-bug`. The two together form the universal observation flow: bugs go to scenario-qa, friction to meta-optimization. Use the right one — see § "When NOT to use" below.

Required reading:
- `docs/meta-optimization/taxonomies/friction-report/README.md` — scopes, severities, schemas, evidence rules, honesty flags. Read this before invoking; the taxonomy is the source of truth for valid input shape.

---

### **1. When to use this skill**

Use `report-friction` when you observe **structural friction** — a gap between what the system promised and what it delivered when you tried to use it:

- **Tool/CLI gap or confusion.** A command flag was confusing, output was non-actionable, a capability was missing, or a tool returned a misleading shape. Scope: `toolchain`.
- **Run-loop or coordination friction.** A heartbeat stalled, a run looped on the same step, a coordination handoff went sideways, an extra heartbeat was needed where one should have sufficed. Scope: `run-execution`.
- **Storage-map or role-boundary confusion.** You weren't sure where to write a piece of information, or you wrote it and the wrong owner picked it up, or a role boundary between teams/members was ambiguous. Scope: `prompt-team-agent-storage`.
- **A workaround you keep applying.** You've used the same workaround across multiple heartbeats or runs because the underlying gap hasn't been fixed. Scope: `recurring-workaround`. Severity: `recurring`.
- **You're not sure where it fits but something is structurally wrong.** Use the `unknown` scope — the curator reclassifies during triage.

---

### **When NOT to use this skill**

Friction is *system-level capture-leak*. These adjacent signals look similar but route differently:

- **Broken code or scenario behavior** — code defects, regressions, prompt confusion, data-shape mismatches, unexpected errors. Use [`report-bug`](../report-bug/SKILL.md) — writes to `bug-inbox/*` on scenario-qa. Bugs are defects against documented behavior; friction is gaps in promised capability.
- **Disagreement with a decision, plan, or contract.** Raise a decision in the appropriate context (e.g., `decision-rejection-proposed` for stale decisions, `framework-update` for contract-level disputes). Disagreement is structural input, not observation.
- **A capability the system should have but doesn't.** The owning member raises a `capability-work` decision (toolchain-validator, run-introspector, or team-agent-optimizer). Capability gaps are commitments to build, not observations.
- **A fix you can apply right now in five minutes.** Just apply it. Filing friction for things you can fix yourself is overhead. The whole point is signal that the *system* should change.
- **Post-hoc deep analysis of a long conversation.** Use [`conversation-friction-analysis`](../conversation-friction-analysis/SKILL.md). That skill is for analytic decomposition with timeline, attribution, and scoring; `report-friction` is for in-flight observation.
- **One-off friction you ran into once.** Mention it in your handoff next time, not the inbox. The curator drops `one-off`-severity entries with a triage note.

If unsure, prefer to file: an over-eager friction report becomes a `drop` with a triage note, which costs less than a missed system-level signal. But respect the per-heartbeat cap (§ 4 below).

---

### **2. Required inputs**

Gather before invoking the writer:

| Input | Required | Format |
|---|---|---|
| `scope` | yes | One of: `toolchain`, `run-execution`, `prompt-team-agent-storage`, `recurring-workaround`, `unknown`. Pick the closest fit; the curator reclassifies if needed. |
| `severity` | yes | `blocking` (you are stopped, no workaround), `recurring` (observed multiple times — provide evidence of recurrence), `one-off` (observed once with workaround applied — note: curator drops these, prefer handoff). |
| `expected` | yes | One-line description of the promised or expected behavior. |
| `actual` | yes | One-line description of the observed behavior. |
| `description` | yes | Free-form notes — what you observed, why this is friction (not a bug, not a fix-it-yourself), hypotheses about cause if you have any. |
| `context` | recommended | Object with the most specific anchors you can give: `scenario`, `skill`, `member`, `command`, `doc`, `task`. Any may be null; more context shortens curator triage. |
| `honesty_flags` | when applicable | List from `speculative-cause`, `repeats-existing-friction-topic`, `minimal-context`, `auto-generated`. Be honest about what your report doesn't have. |

**Severity rule.** Severity is the reporter's claim. The curator may overrule based on observed scope or recurrence. `recurring` requires evidence of recurrence (count, prior-entry pointer); `blocking` requires you are currently stopped (not just slowed); `one-off` will be dropped — file it only if the symptom is genuinely interesting in isolation.

**Wording standard.** Write `expected` and `actual` as observable outcomes in
plain declarative sentences ("`skill sync` exits 0 and prints the changed
ids", not "sync is unreliable"). In `description`, state any steps you took in
Simplified Technical English — one action per sentence, imperative mood, exact
commands — so the curator can replay your path without your context.

---

### **3. Procedure**

1. **Validate inputs against the taxonomy.** Read `docs/meta-optimization/taxonomies/friction-report/README.md`. Confirm `scope` is one of the five values. Confirm `severity` is one of the three values. Confirm `expected`, `actual`, `description` are populated (or that you've added the appropriate honesty flag).

2. **Generate a kebab-case slug** that summarizes the friction in 3–6 words. Examples: `cli-rejects-valid-uuid-input`, `heartbeat-loops-on-empty-handoff`, `decision-vs-knowledge-routing-unclear`, `same-yaml-front-matter-fix-applied-fourth-time`.

3. **Prepare one UTF-8 JSON report file.** The command constructs the topic and
   YAML front matter. Supply declared reporter identity, the truthful observation
   date, available context anchors, and three explanatory paragraphs. Unknown
   context values may be null. Do not include secrets in commands or output.
   `reporter` and `reporter_team` are observational metadata, not authenticated
   identity or permission to write. Runtime writer attribution remains separate.

   Example shape (replace the illustrative values with your actual observation):

   ```json
   {
     "scope": "toolchain",
     "severity": "recurring",
     "slug": "capture-guidance-requires-repeated-workaround",
     "reporter": "your-agent-id",
     "reporter_team": "your-team-id",
     "observed_at": "2026-10-03",
     "context": {
       "scenario": "prompt-manager", "skill": "report-friction",
       "member": null, "command": "exact non-sensitive command",
       "doc": "path to expected contract", "task": null
     },
     "expected": "The documented command preserves the observation.",
     "actual": "The command needs the same workaround again.",
     "description": "Run the exact command. Observe the output. State uncertainty.",
     "honesty_flags": ["minimal-context"],
     "recurrence_count": 2,
     "attempt": "Explain what you were trying to do.",
     "observation": "Explain what happened, including output and recurrence evidence.",
     "explanation": "Explain why this is friction rather than a bug or immediate fix; cite the expected contract."
   }
   ```

   `recurring` requires `recurrence_count >= 2` or a nonempty `prior_entry` pointer.
   `blocking` requires `currently_blocked: true`. These declarations support
   mechanical validation; their semantic truth remains your authoring and review
   responsibility. `unknown` is valid for capture and requires curator
   reclassification before downstream routing. The three body fields are producer
   paragraphs; Routings, Drops and Blocked belong only to the curator's snapshot.
   JSON escapes preserve multiline text and special characters. Unknown fields,
   invalid taxonomy values, unsafe slugs and missing required fields fail before
   any storage operation.

4. **Invoke the qualified CLI.** Check the build identity if using a recently
   changed checkout; an installed wrapper can still run an older build.

   ```bash
   prompt-manager team friction-capture meta-optimization --report-file=/absolute/path/report.json --json
   ```

   Do not pass `--topic`, `--caller-note` or `--content`; they are unsupported.
   `--report-file` cannot be mixed with typed report flags. Existing typed callers
   using `--scope --severity --expected --actual --description --slug` and optional
   `--honesty-flags` remain supported, but produce narrower operator reports:
   reporter/team are `operator`, context is reduced, date is capture time, and the
   three explanatory body paragraphs are absent. Historical reports remain valid
   evidence of that narrower route, not of complete declared identity capture.

5. **Confirm and retain the receipt.** Include the returned `knw-...` ID and
   `friction-inbox/<scope>/<slug>` in your handoff. Receiver intake is:

   ```bash
   prompt-manager team knowledge-list meta-optimization --topic-prefix=friction-inbox/ --last=100 --json
   ```

   Prefix discovery is distinct from curator execution; a disabled curator has
   not processed the report. Keep the same file, scope and slug for an exact
   repeat: the structured route reads the existing topic first and returns the
   identical case instead of appending another. Different content at the same
   topic requires reconciliation and fails without a write. After an uncertain
   write, read the same topic before repeating; if that read is unavailable,
   retain an unresolved outcome and stop. Never mint another slug to bypass it.
   This is a bounded serial retry guarantee, not concurrent exactly-once storage.
   The current storage reader bounds its Source Ledger scan to 500 entries;
   older cases outside that window require separate historical reconciliation.

   Synthetic verification belongs only in the documented temporary fixture in
   `scenarios/prompt-manager/api/TESTING_GUIDE.md`, with no live ledger attached.
   Do not place fictional actionable observations in the live inbox.

---

### **4. Output expectations and caps**

The skill produces exactly one knowledge entry on the meta-optimization team. The entry's topic is `topic:friction-inbox/<scope>/<slug>`; its front-matter conforms to the `friction-report` schema; its body provides enough context for the curator to classify and route without your context.

**Per-heartbeat cap (honor-system):** at most **3** friction-inbox entries per heartbeat per agent. If you observe more than 3 distinct friction signals, group related signals into a single `recurring-workaround`-scope entry that lists all the symptoms, rather than filing each separately. This keeps the inbox actionable and respects the curator's `dailyInboxDrainCap`.

You **must not**:

- Modify the friction entry after writing — let the curator handle reclassification, severity changes, and follow-ups via `route-to-another-topic` mirroring or by writing to the destination scoped topic on your behalf.
- Write multiple friction-inbox entries for one root cause — if three symptoms trace to one structural gap, file one entry.
- Skip the honesty flags — `repeats-existing-friction-topic` and `speculative-cause` are not embarrassing; they're load-bearing for the curator's triage and merge logic.
- Use this skill as a fix-it backlog — friction-curator routes; the destination scoped-topic owner (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) decides whether the friction becomes a backlog item or a `capability-work`.

---

### **5. Boundaries**

This skill writes; it does not read, classify, route, or resolve. The friction-curator drains and routes. The destination scoped-topic owner synthesizes patterns and proposes fixes via their existing work types. Each role has its own lane; this skill exists so producers don't need to know any of that — they just file what they observed.

Friction-curator is a **router, not an analyst**. Synthesis stays with debt-curator (who reads scoped friction topics for recurring patterns). Deep root-cause analysis stays with `conversation-friction-analysis` (post-hoc, not in-flight). The curator owns no work types; capability-works and other decisions are still raised by the destination scoped-topic owners after they drain the routed entries.

If the `report-friction` skill is itself buggy or ambiguous, file a `bug-inbox/prompt-confusion/<slug>` entry via `report-bug` and let the bug-investigator pick it up. Recursive correctness is intentional.

---

### **6. Cross-references**

- `docs/meta-optimization/taxonomies/friction-report/README.md` — taxonomy (required reading).
- `docs/meta-optimization/README.md` — meta-optimization team's friction canon overview and cross-team flow diagram.
- `docs/scenario-qa/taxonomies/bug-report/README.md` — sister taxonomy for the bug-inbox flow; useful context for understanding why these are separate writer skills with separate destinations.
- `scenarios/prompt-manager/store/skills/packs/core/report-bug/SKILL.md` — sister writer skill for code/scenario defects.
- `scenarios/prompt-manager/store/skills/packs/core/conversation-friction-analysis/SKILL.md` — deeper post-hoc analysis skill; complementary, not a replacement for in-flight `report-friction`.
- `docs/agent-system/INTAKE_PIPELINE.md` — the inbox-router-drain pattern; friction-inbox uses deterministic-prefix routing (no separate classifier).
- `docs/agent-system/TOPICS_SCHEMA.md` § Universal-source intakes — the `source_team: "*"` semantics that make `friction-inbox/*` reachable from every team.
