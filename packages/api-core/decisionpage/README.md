# decisionpage: the shared decision-page writer

A sender has the facts of a Notification Hub decision but cannot design its page. Research Desk's courier is one; scenario code and the portfolio steward are others. The sender hands those facts to this writer. Each job:

1. writes a brief (`BRIEF.md`) into the job folder;
2. starts one Codex `gpt-6-luna` run through Agent Manager, under Notification Hub's declared profile `notification-hub/decision-page-writer` (role `code.cheap`, medium effort). The run follows the `notification-hub` skill, writes `page.html`, checks it with `notification-hub pages check` and photographs it with `notification-hub pages shoot`;
3. judges the page with the Hub's own `CheckAsk`, the check behind `conversations ask --dry-run`. New Hub checks therefore apply without changes here;
4. on a failed check, starts one more run with the findings;
5. returns the page or a typed failure.

Jobs live in `DefaultRoot()` (`~/.vrooli/state/vrooli/decision-pages/<sender>/<subject>/<digest>/`). They are idempotent by subject key and facts: the same request returns the same job, and changed facts start a new one.

The writer presents what the sender and the evidence say and invents nothing. A thin brief stays thin; what the page lacks is listed in the job's `RETURN.md`.

## Interface

- `Writer.Status(Request)` reads a job and changes nothing.
- `Writer.Advance(ctx, Request)` starts or moves a job on without blocking.
- `Writer.Write(ctx, Request, wait, poll)` advances until the job is terminal or `wait` runs out.
- `Result.State` is `queued`, `working`, `ready` (`Page`, `HTML`, `SHA256`) or `failed` (`Failure.Class`).
- `Result.Attempts[]` gives each run's id and the model Agent Manager admitted.
- `Result.Advice[]` holds Hub findings about the sender's cover (for example `missing_recommendation_reason` or `recurring_subject`). They are the sender's to fix, not the page's.

| Failure class | Meaning | What the writer does |
|---|---|---|
| `invalid_request` | The request cannot be written (fewer than two options, unknown recommendation, relative evidence path, bad subject key). | Terminal. |
| `run_failed` | Both runs ended without a page. | Terminal. |
| `lint_failed` | The second page still fails the Hub's check. Page findings are `explainer_*` (except `explainer_missing`), `command_hidden`, `page_unreadable`, and anything the Hub blocks on that names no cover field. | Terminal. |
| `launch_refused` | The writer has no authority to start a run: no service credential, a credential that is not exactly the sender's, an Agent Manager 401/403, a pending operator approval (202), or that approval already outstanding (429). | Retried after `RetryAfter` (30 minutes). Never a page miss. |
| `unavailable` | Agent Manager or the Hub could not be reached, or answered 5xx. | Retried after `RetryAfter`. |

A sender treats only the three terminal classes as a miss. Research Desk keeps such a decision `packet_not_ready` and records it in `page-misses.json`. No sender falls back to a page that restates the question.

## Authority: who may start the runs

Starting an Agent Manager run is authority. `AgentManager.Authorization` must supply the bearer of the sender's own declared service principal. Without it, `Launch` refuses before making any call (`ErrNoCredential`, class `launch_refused`). This holds even though Agent Manager today admits run creation from a caller that presents no identity: the create-run caller switch AUTH-01 is off by operator decision P-18 (`docs/concepts/AUTHORIZATION-CURRENT-STATE.md:231-232`, §4). The writer never relies on that admission. If AUTH-01 is turned on, nothing changes for the writer.

Research Desk's principal is `research-desk-agent-manager` (resource `agent-manager`, scope `agent-manager:write`), declared in `scenarios/research-desk/.vrooli/service-principals.json` and loaded in `scenarios/research-desk/api/page_writer_credential.go`.

The credential only proves "I am Research Desk". What lets it create runs is the operator's standing `run.create` grant keyed to that subject. The operator provisions both with one command:

```
bash ~/Vrooli/docs/internal/auth-consolidation/provision-decision-page-writer.sh
```

That runs `scenario-authenticator service-principals sync --apply`, then `agent-manager standing-grants create --subject research-desk-agent-manager --max-concurrent-runs 2 --ttl 30d --local-owner`.

Until then, every launch is `launch_refused`. Decisions whose page was left out stay `packet_not_ready`, and the plan row says why.

When the grant expires, Agent Manager answers the next launch with 202 approval_pending and asks the operator in the Hub to renew it.

A standing grant admits run creation only. When a run outlives its time limit (`DefaultAttemptTimeout`), the writer still asks Agent Manager to stop it, but under a service principal that stop can be refused. The attempt then records that the run could not be stopped and may still be running; it counts against the grant's concurrency until it ends.

A new sender declares its own `<sender>-agent-manager` principal and gets its own grant, so each run is attributed to the sender that asked.

## Checking and photographing

`Inspect` judges a page the way the writer does. `Preview` and `Shoot` frame a page with the Hub's kit at phone width in light and dark (headless Chrome).

They read the page with the Hub's own scheme rules (`packages/api-core/pagescheme`, which the Hub's packet check also uses). A page whose literal colours work in only one scheme is shown locked to that scheme (`data-hub-scheme`), as the Hub shows it; any other page follows the theme.

The Hub's CLI exposes these as `notification-hub pages check <page> --cover-file <request.json>` and `notification-hub pages shoot <page>`.
