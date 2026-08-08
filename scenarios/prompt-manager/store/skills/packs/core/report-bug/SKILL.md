## Tools focus: Report Bug

Universal writer skill any agent on any team may invoke when they observe a bug — broken code, broken scenario behavior, prompt confusion, data-shape mismatch, unexpected error, or anything that looks defective. The skill writes a structured entry to `team:scenario-qa`'s `topic:bug-inbox/<signal-type>/<slug>` topic; the `literal:scenario-qa/bug-investigator` member drains the inbox, applies a registered investigation technique, and closes the entry with a `topic:bug-investigation-report/<slug>` audit-log entry.

This skill is **destination-coupled by design** — writer skills always are. The portability rule (`prose_topic_leak`) applies to classifier skills, not to writers. Bug reporting is a one-way producer pattern.

Required reading:
- `docs/scenario-qa/taxonomies/bug-report/README.md` — signal types, schemas, evidence rules, honesty flags. Read this before invoking; the taxonomy is the source of truth for valid input shape.

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
- The observation is a **capability gap** (file a `capability-work` decision instead).
- The observation is a **half-formed idea or workaround** (write to the most specific typed team knowledge topic, or use `report-friction` when it is system-level friction).
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

Wording standard: write `repro` steps in Simplified Technical English — one
action per step, imperative mood, exact commands and paths, 20 words or fewer
per step ("Run `vrooli scenario start foo`", not "the scenario should be
started"). Write `expected` and `actual` as observable outcomes, not opinions.
A well-formed report reads as Given/When/Then — context, repro steps, expected
vs actual — which lets the investigator convert it directly into a regression
test.

---

### **3. Procedure**

1. **Validate inputs against the taxonomy.** Read `docs/scenario-qa/taxonomies/bug-report/README.md`. Confirm `signal_type` is one of the six values. Confirm severity is one of the three values. Confirm `repro`, `expected`, `actual` are populated (or that you've added the appropriate honesty flag).

2. **Capture once through the typed writer.** It constructs the topic,
   front-matter, writer attribution, and immutable inbox entry only when the
   taxonomy is complete:

   ```bash
   prompt-manager team bug-capture scenario-qa \
     --title '<short observation>' --signal-type <type> --severity <severity> \
     --repro '<step 1,step 2>' --expected '<expected>' --actual '<actual>' \
     --description '<useful details>'
   ```

   A complete submission publishes in this one command. An incomplete or
   invalid submission is instead saved as a private `draft`, with its missing
   fields, invalid values, and exact `bug-repair` command. Do not retype the
   report or make up a taxonomy value; repair the returned draft.

3. **Confirm the disposition.** Capture the published `knw-...` id, or retain
   the draft id until its repair publishes. Drafts are not bug reports and do
   not appear under `bug-inbox/*`.

---

### **4. Output expectations**

The skill produces one of two explicit outcomes: a published knowledge entry on
the scenario-qa team whose topic is `topic:bug-inbox/<signal-type>/<slug>`, or
a private repairable draft. The published entry's front-matter conforms to the
`bug-report` schema; its body provides enough context for the investigator to
start without your context.

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

- `docs/scenario-qa/taxonomies/bug-report/README.md` — taxonomy (required reading).
- `docs/scenario-qa/README.md` — scenario-qa team plan-of-record overview.
- `docs/scenario-qa/methods/investigation/scientific-debugging.md` — the default investigation technique the bug-investigator applies; useful context for understanding why repro and root-cause-friendly framing matter.
- `docs/agent-system/INTAKE_PIPELINE.md` — the inbox-router-drain pattern; bug-inbox uses deterministic-prefix routing (no separate classifier).
- `docs/agent-system/TOPICS_SCHEMA.md` § Universal-source intakes — the `source_team: "*"` semantics that make `bug-inbox/*` reachable from every team.
