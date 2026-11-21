# Case: policy-batch

Taken from Hub ask 6052a317 (15 Nov 2026). You are Research Desk's agent for the
standing-teams programme (`--from "Research Desk"`).

## Facts you know

Before the standing-team programme runs, eight policies set how far it can go.
Each has a recommendation and a safe default that applies until the operator
answers, so nothing waits on them. Today the only answer offered is "take all
eight" or "keep the defaults", and the eight are only listed in a file on disk.

| # | Policy | Recommendation | Default until answered | Can it be undone? |
|---|---|---|---|---|
| 1 | Weekly usage ceiling for the programme | At most 15 weekly quota points per 24 hours; stop and report at 50% of the week. Keep the cheaper-model mode on. | Cheaper-model mode stays on, no ceiling | Yes |
| 2 | Delete about 8,000 lines of unused work-board and plan code (an unused 4,800-line control stack, a 3,200-line duplicate plan editor, three dormant plan families) | Delete them | Keep them; the board's code then shrinks by 10% instead of 25% | Code stays in history |
| 3 | Let the operator grant a team access from a notification, lasting until revoked | Yes, "until revoked" stated on the grant, and grants in force listed in every daily digest | Grants only by link or command line, 30 days at most | Grant can be revoked; no end date |
| 4 | Answer reviews and goal close-outs in the Hub, as run reviews already are | Yes | They stay in the command line and web console | Yes |
| 5 | Agents may permanently delete any old run of the same owner without asking | Turn this on | Unchanged (they must ask) | No — a deleted run is gone |
| 6 | Agents the operator launched may notify and ask without a separate grant each time (8 of the last 11 permission requests were for this) | Yes | A grant each time | Yes |
| 7 | Mute "turn finished" notifications (2,448 of 3,366 notifications in 30 days) | Mute them; one daily digest at normal urgency | Unchanged | Yes |
| 8 | Give the supervision team an objective covering research and standing-team owners, and enable it last | Yes | Unchanged, enabled last | Yes |

- The sender's own reviewer adds a second opinion: #3 and #5 cannot easily be taken
  back and deserve their own answer; the other six are easy to reverse.
- On an answer, the policies are recorded and handed to the teams that apply them.
- Why it matters: #2 decides whether the board's code shrinks by a quarter or a
  tenth; #3 and #6 remove most repeat permission asks once teams run.
- The Hub can offer a form question (`--form`) with a multi-select, so the operator
  can pick which of the eight to take without typing.

## What the sender would recommend

Take all eight (the research recommendation). A reasonable alternative is taking
the six reversible ones now and deciding #3 and #5 separately.
