## Tools focus: Report Bug

Universal writer skill any agent on any team may invoke when they observe a bug — broken code, broken scenario behavior, prompt confusion, data-shape mismatch, unexpected error, or anything that looks defective. The skill writes a structured entry to `scenario-qa`'s `bug-inbox/<signal-type>/<slug>` topic; the `scenario-qa/bug-investigator` member drains the inbox, applies a registered investigation technique, and closes the entry with a `bug-investigation/<slug>` audit-log entry.

This skill is **destination-coupled by design** — writer skills always are. The portability rule (`non_portable_classifier`) applies to classifier skills, not to writers. Bug reporting is a one-way producer pattern.

Required reading:
- `docs/scenario-qa/BUG_REPORT_TAXONOMY.md` — signal types, schemas, evidence rules, honesty flags. Read this before invoking; the taxonomy is the source of truth for valid input shape.

---

### **1. When to use this skill**

Use `report-bug` when you observe any of:

- **Code that doesn't work as documented or intended.** A scenario, skill, CLI, or component behaves differently from how its prose or schema describes it.
- **Behavior that worked before and doesn't now.** A regression — output changed, an action stopped working, performance degraded measurably.
- **Prompt content misled you.** A skill, RESPONSIBILITIES.md, heartbeat injection, or decision-context description was ambiguous or contradictory enough that you took the wrong action.
- **A payload didn't match its schema.** A CLI output, HTTP response, or file's contents diverged from the documented or declared shape.
- **An error you couldn't classify.** An exception, error code, or failure mode that no skill or doc anticipated, that surprised you.
- **You're not sure what the bug is** but something is clearly wrong. Use the `unknown` signal type — the investigator will classify during triage.

Do **not** use `report-bug` when:

- The observation is a **friction report** with a known owner (route to that owner directly via decision or knowledge entry).
- The observation is a **capability gap** (file a `capability-gap` decision instead).
- The observation is a **half-formed idea or workaround** (write to your team's notebook).
- The observation is a **bug you are about to fix in this same heartbeat**. Just fix it; reporting it for someone else when you're already there is overhead. (If the fix has cross-cutting implications, file `bug-resolution-proposal` after the fix lands.)

If unsure, prefer to file: an over-eager bug report becomes a `drop` after a short investigation, which costs less than a missed defect.

---

### **2. Required inputs**

Gather before invoking the writer:

| Input | Required | Format |
|---|---|---|
| `signal_type` | yes | One of: `code-defect`, `regression`, `prompt-confusion`, `data-shape-mismatch`, `unexpected-error`, `unknown`. Pick the closest fit; the investigator validates. |
| `severity` | yes | `blocker` (work stopped, no workaround), `major` (significant degradation), `minor` (annoyance with workaround). |
| `repro` | yes | Ordered list of steps that reproduce the bug. If you couldn't reproduce, set `honesty_flags` to include `repro-not-attempted` and put what you did observe under `repro` anyway. |
| `expected` | yes | One-line description of what you expected to happen. |
| `actual` | yes | One-line description of what actually happened. |
| `context` | recommended | Object with the most specific anchors you can give: `scenario`, `skill`, `member`, `command`. Any may be null; more context shortens the investigator's triage. |
| `description` | recommended | Free-form notes — what you observed, hypotheses you tried, why this looks like a bug rather than expected behavior. |
| `honesty_flags` | when applicable | List from `repro-not-attempted`, `speculative-cause`, `minimal-context`, `ai-generated-summary`. Be honest about what your report doesn't have. |

Severity rule: severity is **the reporter's claim**. The investigator may overrule it during triage based on actual scope of impact (e.g., a "minor" with a corrupted invariant escalates).

---

### **3. Procedure**

1. **Validate inputs against the taxonomy.** Read `docs/scenario-qa/BUG_REPORT_TAXONOMY.md`. Confirm `signal_type` is one of the six values. Confirm severity is one of the three values. Confirm `repro`, `expected`, `actual` are populated (or that you've added the appropriate honesty flag).

2. **Generate a kebab-case slug** that summarizes the bug in 3–6 words. Examples: `landing-page-builds-fail-on-empty-config`, `seam-discovery-misses-test-files`, `swarm-manager-cli-rejects-valid-uuid`.

3. **Construct the topic.** `bug-inbox/<signal-type>/<slug>`.

4. **Format the front-matter.** Match the `bug-report` schema in the taxonomy exactly:

   ```yaml
   severity: <blocker|major|minor>
   reporter: <your-agent-id>
   reporter_team: <your-team-id>
   observed_at: <today's date in YYYY-MM-DD>
   context:
     scenario: <scenario-id-or-null>
     skill: <skill-id-or-null>
     member: <member-id-or-null>
     command: <command-or-null>
   repro:
     - <step 1>
     - <step 2>
   expected: <one-line>
   actual: <one-line>
   description: |
     <free-form notes>
   honesty_flags: [<flags>]
   ```

5. **Format the body.** Free-form, but include:
   - **What you were trying to do** (one paragraph).
   - **What happened** (one paragraph; specifics like error texts, stack traces, command output go here).
   - **Why this looks like a bug** (one paragraph; cite the contradicting doc/spec/schema if applicable).

6. **Invoke the knowledge writer.** From the command line (or whatever invocation surface your runtime exposes):

   ```bash
   prompt-manager team knowledge-add scenario-qa \
     --by=<your-agent-id> \
     --topic="bug-inbox/<signal-type>/<slug>" \
     --content="$(cat <<'EOF'
   ---
   <front-matter from step 4>
   ---

   <body from step 5>
   EOF
   )"
   ```

7. **Confirm the write.** Capture the `knw-...` id returned by the CLI. Include it in your heartbeat output ("Filed bug-inbox/<topic> as <id>") so the operator and the investigator can trace.

---

### **4. Output expectations**

The skill produces exactly one knowledge entry on the scenario-qa team. The entry's topic is `bug-inbox/<signal-type>/<slug>`; its front-matter conforms to the `bug-report` schema; its body provides enough context for the investigator to start without your context.

You **must not**:

- Modify the bug after writing — let the investigator handle reclassification, severity changes, and follow-ups via `route-to-another-topic` or `bug-resolution-proposal`.
- Write multiple bug-inbox entries for one root cause — if you observed three symptoms of one bug, file one entry that lists all three under `description` or `repro`.
- Skip the honesty flags — `repro-not-attempted` and `speculative-cause` are not embarrassing; they're load-bearing for the investigator's triage.
- Use this skill as a backlog filer — bug-investigator decides whether the bug becomes a backlog item via `file-backlog`.

---

### **5. Boundaries**

This skill writes; it does not read, classify, or resolve. The bug-investigator drains. The qa-contrarian challenges. Each member has their own role; this skill exists so producers don't need to know any of that — they just file what they observed.

If the `report-bug` skill is itself buggy or ambiguous, file a `bug-inbox/prompt-confusion/<slug>` entry against this skill and let the investigator pick it up. Recursive correctness is intentional.

---

### **6. Cross-references**

- `docs/scenario-qa/BUG_REPORT_TAXONOMY.md` — taxonomy (required reading).
- `docs/scenario-qa/README.md` — scenario-qa team plan-of-record overview.
- `docs/scenario-qa/investigation-techniques/scientific-debugging.md` — the default investigation technique the bug-investigator applies; useful context for understanding why repro and root-cause-friendly framing matter.
- `docs/agent-system/INTAKE_PIPELINE.md` — the inbox-router-drain pattern; bug-inbox uses deterministic-prefix routing (no separate classifier).
- `docs/agent-system/TOPICS_SCHEMA.md` § Universal-source intakes — the `source_team: "*"` semantics that make `bug-inbox/*` reachable from every team.
