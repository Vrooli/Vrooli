# Operator attention and research decisions

Read when a study appears to need the operator, when reviewing unanswered requests, or when handing supervision to another agent. The supervisor owns triage and follow-through across its commissioned topics. Notification Hub owns questions, answers and delivery receipts. The operation owner retains authorization and execution.

## Decide whether to ask

First discover supported routes and inspect the latest evidence, including superseding results. A missing tool, socket, environment, build, target fact or requirement does not by itself establish a human dependency. Reuse authorized sources, propose a contract, repair an understood blocker within scope, or choose independent useful work. Bind every gate to the actual subject and commission; a wallet implementation gate does not restrict unrelated validation research.

| Condition | Supervisor action |
|---|---|
| Agents can resolve the uncertainty or restore a permitted owner path | Assign that work. Do not ask the operator to diagnose it. |
| Only the operator can supply personal access, a preference, a private fact or an explicitly human-only decision | Prepare a concrete request with context and the exact next step it unlocks. |
| A human gate exists but the proposal or evidence is not reviewable | Finish the permitted preparation. Retain the gate as future work, not an actionable request. |
| Later evidence already removed the dependency | Mark the research item superseded. Do not send the stale request. |
| The original allowance ended or no supervisor is running | Preserve the prepared request and pickup state. Do not imply that an answer restarts the commission. |

## Keep one navigable view

Use the research home's `ATTENTION.md` as the consolidated index, or the existing cohort record when work is not retained there. Keep one writer across concurrent studies. Workers report candidate requests with evidence; the supervisor deduplicates and validates them. Link the view from the research home, topic and checkpoint rather than duplicating the request in each.

For each ready item, retain a stable identity, affected topic/commission, exact action, why the operator is necessary, alternatives checked, recommendation and consequences, evidence/artifact links, genuine deadline or review point, canonical ask ID, last owner-observed state, and resumption owner/step. Keep notification receipts separate from ask state. Record whether anything actually unblocked after the reply. The view is a dated projection, never an approval ledger or an alternate answer authority.

Keep actionable requests separate from agent prerequisites, future gates and superseded items. A supervisor reconciles them at launch, meaningful findings, new answers, approaching real deadlines and handoff/closure. An empty active queue does not require an invented question. State coverage; a skim of topic guides is not a complete audit of historical plans or accounts.

## Use the existing notification owner

### Resolve authority once at startup

Qualify the intended notification/decision producer when commissioning supervision, before promising alerts. Check current owner-issued authority through the canonical evaluator. Reuse valid coverage, including approved wildcard targets and a covering rule inside a multi-policy consent; do not ask again merely because another notification, worker check or reminder is due.

When authority is missing and the exact reviewed policy is supported, prepare one bounded request through the existing `vrooli agent authorize` flow for the actual session/run, policy, target, purpose and duration. Present the owner-generated consent route in the current conversation or another already-qualified owner surface. Do not rely on the unavailable notification channel to bootstrap its own permission. Record the request identity, pending/approved/refused state, expiry and scope in the supervision record. Continue independent work while it is pending. Reconcile that same request instead of creating another.

Request again only when authority actually expired or was revoked, the approved target/operation changed, or the receiving identity cannot legitimately inherit the existing coverage. A routine token refresh within valid coverage is not a new consent. Session-bound authority can end with its session; "ask once" does not mean permanent authority for every future agent. A refusal is not a reason for unchanged repeated requests.

For direct informational sends/reminders, use the reviewed `notification-sending` policy. Read the configured recipient's `subject` with `notification-hub recipients get --json`, identify this session with `vrooli agent list --json`, and inspect its existing requests with `vrooli agent authorization <session-or-run>`. Resolve the authenticated Agent Manager browser origin from the configured `AGENT_MANAGER_AUTHORIZATION_URL` or its enabled `tunnel-manager routes list --json` entry. Then submit:

```bash
vrooli agent authorize <session-or-run> \
  --policy notification-sending \
  --target notification-hub:recipient/<subject> \
  --expires-in 8h \
  --reason 'Send research summaries and reminders to the existing recipient during this supervision session.' \
  --technical-details 'Only notification.send for the named recipient. Answers, preferences and subscriptions remain with their existing owners.' \
  --approval-base https://<authenticated-agent-manager-host>
```

Fill the selectors from owner reads, and shorten the duration to the actual commission when appropriate. A new approval replaces this session's earlier grant: include any still-needed reviewed policies in the same request by repeating `--policy`, each followed by its own `--target`. Do not silently discard existing required coverage or request unrelated policies.

Successful submission prints a pending request ID, native approval URL and review expiry. Present that URL for independent human approval and reconcile with `vrooli agent authorization <same-session-or-run>`. The running session resolves approved authority on subsequent CLI calls; no agent restart or manual token handling is needed. A denied send is not a consent request. `unknown reviewed policy`, missing owner capability, or an unavailable authority is not a pending approval: retain the exact owner response and repair the supported path within scope. Never self-approve or substitute owner credentials. After approval, qualify one authorized send and its receipt before promising delivery.

The human approver must hold `agent-manager:write` and `notification-hub:write`. If the consent page names a missing capability, prepare the minimal change to that human's existing provider/subject binding and obtain explicit operator authorization before applying it. Keep that owner-policy setup separate from independent approval of the agent's grant; do not expand agent profiles or a managed run's admission ceiling to make a request pass.

### Route and reconcile requests

Use `notification-hub conversations list --open --limit <bounded-limit>` to inspect current asks and `notification-hub conversations get <ask-id>` for a selected one. Verify the configured recipient/channel through `notification-hub channels status`. An unavailable read stays unknown; do not infer that no ask or device exists. Do not alter recipients, subscriptions or quiet hours under a research commission.

Qualify sender authorization separately from channel readiness. Direct notification ingress requires a verified producer carrying the canonical `notification-hub:write` coverage or a legitimate owner credential. Interactive grants also require `notification.send` for `notification-hub:recipient/<configured-subject>` through the canonical operation evaluator. Covering wildcard targets and rules inside multi-policy consent remain valid. The direct conversation-ask handler requires verified owner identity; sender consent does not authorize asks or answers. For a genuine decision on an existing goal home, `agent-manager effort decision raise` is a separate supported producer route: the verified run raises a service-owned event that Hub translates into an ask. Qualify the actual repository-relative goal-home identity and immutable decision terms. Use `--big-picture` and `--technical-details` for context; do not disguise an informational update as a question. An unbound goal home can retain the decision/answer but has no effort signal or automatic wake; the active supervisor must reconcile it through `effort decision list`/`sync`. A real managed effort adds the existing signal/wake linkage. Do not fabricate an effort, invent a home reference for an unrelated path, or impersonate the owner. This source-qualified route is not a live delivery proof. If no eligible producer path exists, retain the concrete integration blocker and show requests in the current conversation.

When the qualified owner route is direct Notification Hub and an item is ready, use `notification-hub conversations ask` with a self-contained `--question`, at least two meaningful `--option key=label` choices, optional `--recommend`/`--recommend-reason`, `--urgency normal`, the correct `--sensitivity-label`, a stable `--idempotency-key`, and an absolute `--deadline`. Use `--context-url` only for a legitimately accessible context page; a private local file path is not a phone-readable web URL. Include the important context in the question when no such page exists. For a fact rather than a choice, an option can request the detail in an answer note plus an explicit defer option. Do not manufacture choices to fit the API.

The deadline is a real decision deadline or a clearly labeled review point, not implied consent. Persist it with the immutable request before submission. Reconcile a lost response; retry the exact same terms and key. Recomputing a relative deadline changes the terms. Use the existing ask for the same question; materially changed proposals need explicit linkage and revalidation.

Omit `--default` for access, funding, irreversible effects and human-only decisions. Silence, deadline expiry, push delivery or acknowledgement never supplies approval. A reversible preference default is only available when the actual commission permits it and the owner semantics fit. A reply does not replace a required canonical authorization grant or independent-human proof.

Read `notification-hub notifications get <notification-id>` to inspect acceptance and delivery receipts. Channel readiness, accepted, delivered and read/answered are different facts. Respect existing sensitivity, quiet-hour and escalation policy; do not relabel content or increase urgency to force delivery. Record held, failed, unavailable and unknown dispositions.

Bundle ordinary information with `notification-hub notifications send` using its stable idempotency/dedupe keys. Link or name the original ask in reminders. Remind for new decision-relevant context, a real approaching deadline, or a useful outstanding request at an established review. Unless the commission specifies otherwise, send no more than one routine reminder per request per day while supervision is active. Do not repeat unchanged reminders merely because a check ran. Use normal/low urgency; critical is reserved for a real critical condition.

## Answer, resume and close

Use `notification-hub conversations wait --ask-id <id> --deadline <bounded-wait>` when waiting through the owner can support the active work. Otherwise continue independent work and reconcile the canonical ask at the next material review or owner wake. A timed-out wait is not an answer. Do not create a private polling loop or scheduler.

Read the answer through its owner and validate whatever authority the intended operation actually requires. Do not answer your own request on behalf of the operator or impersonate an independent human. A reply received elsewhere must use a supported owner reconciliation path; do not forge a canonical answer. Ordinary operator direction still authorizes ordinary in-scope work when no extra grant is required.

Update the affected brief/checkpoint, resume the exact dependent step within the remaining allowance, and record its observed result. If another condition still blocks progress, explain it and continue other useful work. Record pickup pending when the supervisor is absent. Verify the actual wake/recovery route before claiming unattended follow-through.

If a request becomes stale, use a supported withdrawal operation when one exists. When none is exposed for that request kind, stop reminders and mark it superseded in the research view while retaining the unresolved owner state. Never simulate withdrawal by choosing an answer. Keep requests for later commissions distinct from the effects of the current one.
