# Levels: decisions travel up, friction travels down

Research work runs at three levels, each a home in the sense of [R4](RULES.md):

- **Topic.** A research folder: its owners and its commissions.
- **Research manager.** The research home, across topics.
- **Portfolio steward.** The single top manager, across the whole project. If a live feed fails, it decides from retained evidence and states that evidence's age and gaps.

Each level keeps its open decisions, planned work and inbox **in its own folder**, so a restarted agent finds everything where it was started. Direction reaches an agent as files it re-reads at start; a chat message steers nothing durable (R4). Messages go into the receiver's inbox file, which it reads next time it starts.

## The files at each level

| | Topic: `~/.vrooli/research/topics/<t>/` | Research manager: `~/.vrooli/research/` | Steward: `~/.vrooli/portfolio/` |
|---|---|---|---|
| Open decisions | `DECISIONS.md`, `## Needs you` | `ATTENTION.md`, `## Actionable now` | `DECISIONS.md`, generated; raise with `steward/bin/steward.py ask --id <ID> … --packet steward/asks/<ID>/BRIEF.md` |
| Inbox | `INBOX.md` | `INBOX.md` | `INBOX.md` |
| Planned work and decision packets | `work/<ID>/BRIEF.md` | `work/<ID>/BRIEF.md` | `steward/asks/<ID>/BRIEF.md` (decisions); `steward/delegations/<id>/` (launched work) |
| History | `history/` | `ATTENTION-HISTORY.md`, `history/` | `history/INBOX.md`, `steward/state/`, `LEDGER-ARCHIVE.md` |

**What each level decides.** Topic: questions inside its mission. Research manager: big restructures of research folders and scope changes. Steward: project planning, where to focus next, new projects, income, capacity, forms that ask the operator for missing information, and fleet-scale simplifications. The steward raises every spend decision to the operator and never grants itself money (bright line 1).

A level decides alone what its mission and the operator's words already settle (R9). It asks the operator only when a choice changes what the operator gets (an end state, a scope, a merge that drops one), spends what only the operator grants (money, credentials, permissions, deleting their data), or cannot be undone.

**Route any item with one question: does it ask the operator to choose?**
- **Yes:** a decision at the level whose scope it is in.
- **No:** an inbox entry at the level that owns the fix or the work; a sibling topic counts. The writer signs as itself; operator words keep their `Source:`.
- **A worker's question** goes as a `clarification` to the level that gave it its brief, which answers by amending the packet.
- **A planned risk that fires** becomes an owner task, `work/<ID>/BRIEF.md`, at its level. It is not an operator decision.
- **A defect in a scenario's code** belongs to no level: file it with the `report-bug` skill.
- **Operator direction given in chat** goes, with its `Source:`, to the inbox of the level whose scope it limits. Capacity belongs to the steward; spend belongs to the operator, raised by the steward.
- **Progress the operator need not act on** is never pushed as a notification. Only decisions go to the Hub (R9).

**Names.** A decision `<ID>` is UPPER-KEBAB and names the decision, never a workstream (`LM-40GB-MODEL`). A feedback `<slug>` is lower-kebab and names the concern (`terminal-freeze`).

**Projection.** The steward's `DECISIONS.md` lists its own asks and every research decision from the levels' files (`needs_you.py open`), under the line each moves. A decision is never copied up as a second item: the projection is computed, so an item answered in its topic's inbox leaves it at the next tick.

## Decisions travel up

1. The agent that raises a decision understands it best, so it writes all of it before asking:
   - **The packet.** `work/<ID>/BRIEF.md` is the brief a worker would get, so its executor re-derives nothing: a dated `Checked:` line (the premise verified live, and how), the destination, the identities and paths involved, each command with its directory and what it prints (marked operator-only where so), what each answer settles, `other` included, with its proof, and what is decided. `needs_you.py write` refuses a decision whose folder lacks it or its page, and the courier holds such a decision back until it exists.
   - **The page.** `work/<ID>/page.html` is what the operator sees: what is chosen, the recommendation, what each answer does, the default and the date it takes effect with no answer (R9). It follows the `notification-hub` skill.
   - **The item.** One needs-you item in the level's decisions file, written with `scenarios/research-desk/skills/research-coordination/scripts/needs_you.py write --file <draft> --section <the level's section> --target <decisions file>`, which checks the item before it is written. Grammar: `scenarios/research-desk/docs/reference/FILE-CONTRACT.md` §5.
     ```markdown
     ### LM-40GB-MODEL — Download the 40 GB model now, or wait for a smaller one?
     Two to six plain sentences for a reader with no context.
     Recommended: wait — Wait for the 14 GB release due next week.
     Reason: One sentence on why.
     Options: now — Download it tonight; wait — Wait for the smaller one.
     If silent: wait on 2026-11-27
     Packet: work/LM-40GB-MODEL/BRIEF.md
     Explainer: work/LM-40GB-MODEL/page.html
     ```
     Anyone who relays an item copies the raiser's item byte for byte.
     `Default: <option key> on <date>` is what the level does if the operator has not answered by then (R9). The raiser writes it; `needs_you.py` does not check it yet.
2. The courier sends every open item to Notification Hub as a page (R9). The Hub delivers the ask; the answer is written back to the owning home (step 3), so the home stays the truth (R4).
3. The courier appends the answer to **the owning level's `INBOX.md`** as a `from research-desk · answer` entry with the answer key, the operator's note, the `Packet:` path and `Source: hub ask <id>`, even when the agent that asked has stopped.
4. The next agent at that level drains the answer: it adds the answer and note to the packet under `## Operator answer` (feedback amends the packet; it never starts a new one), starts a worker with the standard executor message, and moves the item to the level's history.
5. An answer never waits for nobody. `needs_you.py open` lists every answer still in an inbox, with its wait and its executor line. When no live commission reads that inbox, the research manager starts the executor.

## Friction and feedback travel down

- An agent that finds friction, or receives operator feedback that belongs elsewhere, writes an entry into the inbox **of the level that owns the fix**. The steward writes into a topic's `INBOX.md` with `steward.py feedback route <F> --topic <t>`, or `feedback send-down` for an item already routed.
- An operator concern that must outlive the conversation becomes `feedback/<slug>.md` at the owning level, in the operator's words. The sender greps that `feedback/` first. A repeat is added as a dated recurrence and raises priority; it never becomes a new record.

## Inbox entries

```markdown
### 2026-11-15T09:12Z · from ceo · friction
The terminal freezes during long agent sessions (second report; see feedback/terminal-freeze.md).
Source: operator chat (session 6b8ba06c, 2026-11-15T09:05Z)
```

| Kind | Use it for |
|---|---|
| `feedback` | The operator's view of the direction or quality of the work |
| `friction` | Something got in the way, whoever reports it |
| `answer` | A decision was answered |
| `clarification` | What something is for, or what it means |
| `priority` | Do this before that |
| `decision` | Please raise or carry a decision |

**Senders:** `operator`, `ceo` (the steward), `supervisor` (the research manager), `topic:<name>`, `commission:<id>`, or an agent name. The tools still spell the first two ids `ceo` and `supervisor`; the roles are the steward and the research manager (R5). `Source:` says where operator words came from (a Hub ask id, an `OPERATOR.md` line, or a chat with its session and time). `About:` says whom quoted words addressed.

**Writing and draining** use `scenarios/research-desk/skills/research-coordination/scripts/inbox.py`, which stamps the real time:
- Write: `inbox.py append <level folder> --from <sender> --kind <kind> [--source …] [--about …] --body '…'`. Single quotes keep `$` amounts intact.
- Drain: `inbox.py drain <level folder> --entry <stamp> --did '…'`. The next agent at that level does this at start; the entry moves verbatim to `history/INBOX.md` with an `Acted:` line.

## Standard spawn messages

One line, then the operator's stance verbatim (R3). An executor's is printed, never typed: `needs_you.py spawn <ID>` adds a re-check line when the brief's `Checked:` is over a day old or its page changed after it, and prints nothing for a brief that cannot run cold.
- **Owner of planned work:** "You own `<topic>/work/<ID>`. Read `<abs>/BRIEF.md` and the topic's `MISSION.md`, then build toward the brief's showcase."
- **Executor of an answered decision:** "Execute the approved decision `<ID>`. Read `<abs>/BRIEF.md`; the answer is `<key>`."

If the executor would need anything the packet does not say, the packet was incomplete. Fix the packet-writing step, not the executor.
