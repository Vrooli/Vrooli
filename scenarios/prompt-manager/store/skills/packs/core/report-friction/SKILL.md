## Tools focus: Report Friction

Universal writer skill any agent on any team may invoke when they observe friction — something that was missing, broken, confusing, slow, undocumented, or harder than it should have been. The skill writes a structured entry to `team:meta-optimization`'s `topic:friction-inbox/<scope>/<slug>` topic; the `literal:meta-optimization/friction-curator` member drains the inbox, classifies the scope, and routes to the appropriate scoped friction topic owned by an existing meta-optimization sub-member.

This skill is **destination-coupled by design** — writer skills always are. The portability rule (`non_portable_classifier`) applies to classifier skills, not to writers. Friction reporting is a one-way producer pattern.

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
- **A capability the system should have but doesn't.** The owning member raises a `capability-gap` decision (toolchain-validator, run-introspector, or team-agent-optimizer). Capability gaps are commitments to build, not observations.
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

---

### **3. Procedure**

1. **Validate inputs against the taxonomy.** Read `docs/meta-optimization/taxonomies/friction-report/README.md`. Confirm `scope` is one of the five values. Confirm `severity` is one of the three values. Confirm `expected`, `actual`, `description` are populated (or that you've added the appropriate honesty flag).

2. **Generate a kebab-case slug** that summarizes the friction in 3–6 words. Examples: `cli-rejects-valid-uuid-input`, `heartbeat-loops-on-empty-handoff`, `decision-vs-knowledge-routing-unclear`, `same-yaml-front-matter-fix-applied-fourth-time`.

3. **Construct the topic.** `topic:friction-inbox/<scope>/<slug>`.

4. **Format the front-matter.** Match the `friction-report` schema in the taxonomy exactly:

   ```yaml
   severity: <blocking|recurring|one-off>
   scope: <toolchain|run-execution|prompt-team-agent-storage|recurring-workaround|unknown>
   reporter: <your-agent-id>
   reporter_team: <your-team-id>
   observed_at: <today's date in YYYY-MM-DD>
   context:
     scenario: <scenario-id-or-null>
     skill: <skill-id-or-null>
     member: <member-id-or-null>
     command: <command-or-null>
     doc: <doc-path-or-null>
     task: <task-id-or-null>
   expected: <one-line>
   actual: <one-line>
   description: |
     <free-form notes>
   honesty_flags: [<flags>]
   ```

5. **Format the body.** Free-form, but include:
   - **What you were trying to do** (one paragraph).
   - **What happened** (one paragraph; specifics like command output, observed shape, recurrence count if `severity: recurring`).
   - **Why this is friction (not a bug, not fix-it-yourself)** (one paragraph; cite the promised behavior or the system contract you expected).

6. **Invoke the knowledge writer.** From the command line (or whatever invocation surface your runtime exposes):

   ```bash
   prompt-manager team knowledge-add meta-optimization \
     --by=<your-agent-id> \
     --topic="friction-inbox/<scope>/<slug>" \
     --content="$(cat <<'EOF'
   ---
   <front-matter from step 4>
   ---

   <body from step 5>
   EOF
   )"
   ```

7. **Confirm the write.** Capture the `knw-...` id returned by the CLI. Include it in your heartbeat output ("Filed friction-inbox/<scope>/<slug> as <id>") so the operator and the curator can trace. You can later track where the curator routed it via `prompt-manager team knowledge-list meta-optimization --topic-prefix=friction-report/`.

---

### **4. Output expectations and caps**

The skill produces exactly one knowledge entry on the meta-optimization team. The entry's topic is `topic:friction-inbox/<scope>/<slug>`; its front-matter conforms to the `friction-report` schema; its body provides enough context for the curator to classify and route without your context.

**Per-heartbeat cap (honor-system):** at most **3** friction-inbox entries per heartbeat per agent. If you observe more than 3 distinct friction signals, group related signals into a single `recurring-workaround`-scope entry that lists all the symptoms, rather than filing each separately. This keeps the inbox actionable and respects the curator's `dailyInboxDrainCap`.

You **must not**:

- Modify the friction entry after writing — let the curator handle reclassification, severity changes, and follow-ups via `route-to-another-topic` mirroring or by writing to the destination scoped topic on your behalf.
- Write multiple friction-inbox entries for one root cause — if three symptoms trace to one structural gap, file one entry.
- Skip the honesty flags — `repeats-existing-friction-topic` and `speculative-cause` are not embarrassing; they're load-bearing for the curator's triage and merge logic.
- Use this skill as a fix-it backlog — friction-curator routes; the destination scoped-topic owner (toolchain-validator, run-introspector, team-agent-optimizer, debt-curator) decides whether the friction becomes a backlog item or a `capability-gap`.

---

### **5. Boundaries**

This skill writes; it does not read, classify, route, or resolve. The friction-curator drains and routes. The destination scoped-topic owner synthesizes patterns and proposes fixes via their existing decision contexts. Each role has its own lane; this skill exists so producers don't need to know any of that — they just file what they observed.

Friction-curator is a **router, not an analyst**. Synthesis stays with debt-curator (who reads scoped friction topics for recurring patterns). Deep root-cause analysis stays with `conversation-friction-analysis` (post-hoc, not in-flight). The curator owns no decision contexts; capability-gaps and other decisions are still raised by the destination scoped-topic owners after they drain the routed entries.

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
