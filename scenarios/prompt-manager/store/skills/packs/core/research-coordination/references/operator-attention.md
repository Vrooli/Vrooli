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

For machine review, keep exactly one `## Actionable now` section in `ATTENTION.md`. Each ready item starts with `### STABLE-ID — concise title`; use the same ID for the corresponding owner decision where that route applies. When the section is empty, its only nonblank line is `No actionable items are recorded.` Put deferred, answered and historical items under other level-two headings. Preserve the detailed context beneath each actionable heading. At a review, supply the authorized research root and, when applicable, the exact enrolled effort reference to `program-runtime.research-supervision-read`. Its bounded index and owner-decision comparison identify items needing reconciliation; a partial board or unavailable owner means unobserved, not proven absent. The index reports only the recorded snapshot and cannot detect a need never written down. It does not send notifications, create authority, or wake a stopped supervisor.

Before dispatching a fresh commission and immediately before every delivery, read each relevant ask from Notification Hub and reconcile its latest owner state. Treat answered (including `other`), dismissed, withdrawn and expired as distinct terminal or review outcomes; none remains actionable unless the operator's answer created a new, explicit request. Move terminal items out of `Actionable now`, update the triage/checkpoint and recompute the inline digest before sending. Never copy an old actionable entry into a fresh commission without this owner-state check. If the owner cannot be read, mark the state unobserved and do not describe it as pending or resolved.

## Structure every operator delivery

Refresh the actionable section immediately before composing a delivery. Every Notification Hub ask or informational notification includes the full absolute path to the supervisor's current action file. For this research home, use `/home/matthalloran8/.vrooli/research/ATTENTION.md`. Also include a compact inline digest of the other currently actionable items (stable ID and short title, with the needed operator action); do not make a local file path the only way to find them, because it may not open on a phone. Include only ready actionable items, not agent prerequisites, future gates, answered requests or superseded items. If the file cannot be read, include the known path, state that the other-item digest is unverified, and do not claim the queue is empty.

Use a decision ask when the operator's answer can select among concrete, allowed next steps. Give at least two meaningful multiple-choice options and recommend one with a brief rationale whenever evidence supports a best choice. If the evidence does not favor an option, say that there is no recommendation and why; do not imply a false preference. Make the question self-contained: state the decision, the consequence of each option, the exact next action the answer unlocks, the genuine deadline or review point, and the action-file path plus other-item digest. Use an informational notification for status or findings that need no operator choice. Do not turn an update into a question or invent options to satisfy the ask API.

Use this content order so the operator can scan either format:

- **Decision ask:** Decision → why the operator is needed / what the answer unlocks → recommendation and rationale, or why there is no recommendation → native multiple-choice options with consequences → genuine deadline or review point → other currently actionable item IDs/titles → full action-file path.
- **Informational notification:** Update → current state and impact → whether any response is needed (normally “No response needed”) → other currently actionable item IDs/titles → full action-file path.

Keep the choices in the Notification Hub option fields as well as explaining their consequences in the question text. Keep the inline digest short enough for a phone screen; the action file holds full context.

## Use the existing notification owner

Before presenting a permission decision, trace the operation owner and the
implementation that would apply it. State both the requested operation and the
underlying access it requires; a SMART read may require raw block-device access
whose scope also exposes drive contents. A tool install hint is not evidence
that setup provisions that permission: confirm the named setup requirement or
owner command exists. If the approved operation has no executable owner path,
repair that path or ask about a concrete alternative; do not imply that consent
alone makes the host change possible. Prefer a one-shot, identity-bound
control-plane probe when it provides the required evidence without granting
persistent scenario access.

The notification field `requested_by` identifies the configured recipient
partition; it does not identify the human or agent actor. Do not infer
authorship from that field. For agent-authored delivery, retain the active
session/run identity, matching approved grant, successful authorized CLI/API
operation and delivery receipt; report any missing link.

### Resolve authority once at startup

Qualify the intended notification/decision producer when commissioning supervision, before promising alerts. If the commission expects operator notices or decisions, make this a startup action before worker dispatch: inspect the recipient, current session/run, existing grant requests and delivery route. Check current owner-issued authority through the canonical evaluator. Reuse valid coverage, including approved wildcard targets and a covering rule inside a multi-policy consent; do not ask again merely because another notification, worker check or reminder is due.

When authority is missing and the exact reviewed policy is supported, submit one bounded request through the existing `vrooli agent authorize` flow for the actual session/run, policy, target, purpose and duration. Do this at startup, not only after the first operator decision appears. Present the owner-generated consent URL in the current conversation immediately so the operator can approve it. Do not rely on the unavailable notification channel to bootstrap its own permission. Record the request identity, pending/approved/refused state, expiry and scope in the supervision record. Continue independent work while it is pending. Reconcile that same request instead of creating another. An in-chat question, `request_user_input`, or `ATTENTION.md` entry is not a Notification Hub ask or delivery receipt.

Request again only when authority actually expired or was revoked, the approved target/operation changed, or the receiving identity cannot legitimately inherit the existing coverage. A routine token refresh within valid coverage is not a new consent. Session-bound authority can end with its session; "ask once" does not mean permanent authority for every future agent. A refusal is not a reason for unchanged repeated requests.

Qualify decision-ask authority separately from informational-send authority. The reviewed `notification-decision-asking` policy grants only `notification.ask` for `notification-hub:recipient/<subject>`; `notification-sending` grants only `notification.send`. The direct ask handler accepts a verified owner or this exact approved agent operation. Answers, preferences and subscriptions remain owner-only. The Agent Manager alternative is `agent-manager effort decision raise` for a genuine decision on a real goal home associated with an enrolled effort. Verify that exact effort/home relationship before using it. If neither route is qualified, keep the item pending and report the owner-identity or enrollment blocker. Do not substitute an informational push, an in-chat question, or a newly invented/unrelated goal home for the decision feed.

When supervision needs both informational sends and decision asks, request both reviewed policies in one consent because a new approval replaces the session's previous grant. Otherwise request only the operations the commission needs. Read the configured recipient's `subject` with `notification-hub recipients get --json`, identify this session with `vrooli agent list --json`, and inspect its existing requests with `vrooli agent authorization <session-or-run>`. Resolve the authenticated Agent Manager browser origin from the configured `AGENT_MANAGER_AUTHORIZATION_URL` or its enabled `tunnel-manager routes list --json` entry. Then submit:

```bash
vrooli agent authorize <session-or-run> \
  --policy notification-sending \
  --target notification-hub:recipient/<subject> \
  --policy notification-decision-asking \
  --target notification-hub:recipient/<subject> \
  --expires-in 8h \
  --reason 'Send research summaries and reminders to the existing recipient during this supervision session.' \
  --technical-details 'Only notification.ask and notification.send for the named recipient. Answers, preferences and subscriptions remain with their existing owners.' \
  --approval-base https://<authenticated-agent-manager-host>
```

Fill the selectors from owner reads, and shorten the duration to the actual commission when appropriate. A new approval replaces this session's earlier grant: include any still-needed reviewed policies in the same request by repeating `--policy`, each followed by its own `--target`. Do not silently discard existing required coverage or request unrelated policies.

Successful submission prints a pending request ID, native approval URL and review expiry. Present that URL for independent human approval and reconcile with `vrooli agent authorization <same-session-or-run>`. The running session resolves approved authority on subsequent CLI calls; no agent restart or manual token handling is needed. A denied send is not a consent request. `unknown reviewed policy`, missing owner capability, or an unavailable authority is not a pending approval: retain the exact owner response and repair the supported path within scope. Never self-approve or substitute owner credentials. After approval, qualify one authorized send and its receipt before promising delivery.

The human approver must hold `agent-manager:write` and `notification-hub:write`. If the consent page names a missing capability, prepare the minimal change to that human's existing provider/subject binding and obtain explicit operator authorization before applying it. Keep that owner-policy setup separate from independent approval of the agent's grant; do not expand agent profiles or a managed run's admission ceiling to make a request pass.

### Route and reconcile requests

Use `notification-hub conversations list --open --limit <bounded-limit>` to inspect current asks and `notification-hub conversations get <ask-id>` for a selected one. Verify the configured recipient/channel through `notification-hub channels status`. An unavailable read stays unknown; do not infer that no ask or device exists. Do not alter recipients, subscriptions or quiet hours under a research commission.

Qualify sender authorization separately from channel readiness. Direct notification ingress requires a verified producer carrying the canonical `notification-hub:write` coverage or a legitimate owner credential. Interactive `notification-sending` grants require `notification.send`; the separate `notification-decision-asking` grant requires `notification.ask`, each bound to `notification-hub:recipient/<configured-subject>` through the canonical operation evaluator. Covering wildcard targets and rules inside multi-policy consent remain valid. Direct human asks require verified owner identity; agent asks require the approved decision-asking policy. Neither grant authorizes answers, preferences or subscriptions. For a genuine decision on an existing goal home, `agent-manager effort decision raise` is another supported producer route: the verified run raises a service-owned event that Hub translates into an ask. Qualify the actual repository-relative goal-home identity and immutable decision terms. Use `--big-picture` and `--technical-details` for context; do not disguise an informational update as a question. An unbound goal home can retain the decision/answer but has no effort signal or automatic wake; the active supervisor must reconcile it through `effort decision list`/`sync`. A real managed effort adds the existing signal/wake linkage. Do not fabricate an effort, invent a home reference for an unrelated path, or impersonate the owner. This source-qualified route is not a live delivery proof. If no eligible producer path exists, retain the concrete integration blocker and show requests in the current conversation.

When a qualified human or approved agent route is available and an item is ready, deliver it then; do not leave it only in the research index until handoff. For a real decision, use `notification-hub conversations ask` with the self-contained contents described above, at least two meaningful `--option key=label` choices, `--recommend`/`--recommend-reason` when evidence supports a recommendation, `--urgency normal`, the correct `--sensitivity-label`, a stable `--idempotency-key`, and an absolute `--deadline`. Agent submissions require the `notification-decision-asking` grant; a send grant alone must be refused. Put the full action-file path and the compact digest of other ready actions in the question text. Use `--context-url` only for a legitimately accessible context page; a private local file path is not a phone-readable web URL. For a fact rather than a choice, ask for the detail in an answer note with an explicit defer option only when deferring is a meaningful response. Do not manufacture choices to fit the API.

Every decision includes the Hub-provided `other` option. Its answer requires a non-empty note, which carries the operator's alternative back to the requester; do not add a competing `other` key to the ask. For source-bound decisions, preserve the exact owner-defined keys and treat `other` only as a non-approval response; free text must never be interpreted as a typed approval.

The recipient can use `notification-hub conversations dismiss <ask-id>` to close an open ask without recording an answer. This is a durable dismissed outcome, not deletion and not consent. The requester can use `notification-hub conversations withdraw <ask-id> --reason '<why it is obsolete>'` to withdraw a direct ask it created; source-owned event decisions retain their exact withdrawal route. Reconcile these outcomes separately from an operator answer.

The deadline is a real decision deadline or a clearly labeled review point, not implied consent. Persist it with the immutable request before submission. Reconcile a lost response; retry the exact same terms and key. Recomputing a relative deadline changes the terms. Use the existing ask for the same question; materially changed proposals need explicit linkage and revalidation.

Omit `--default` for access, funding, irreversible effects and human-only decisions. Silence, deadline expiry, push delivery or acknowledgement never supplies approval. A reversible preference default is only available when the actual commission permits it and the owner semantics fit. A reply does not replace a required canonical authorization grant or independent-human proof.

Read `notification-hub notifications get <notification-id>` to inspect acceptance and delivery receipts. Channel readiness, accepted, delivered and read/answered are different facts. Respect existing sensitivity, quiet-hour and escalation policy; do not relabel content or increase urgency to force delivery. Record held, failed, unavailable and unknown dispositions.

Bundle ordinary information with `notification-hub notifications send` using its stable idempotency/dedupe keys. Link or name the original ask in reminders. Remind for new decision-relevant context, a real approaching deadline, or a useful outstanding request at an established review. Unless the commission specifies otherwise, send no more than one routine reminder per request per day while supervision is active. Do not repeat unchanged reminders merely because a check ran. Use normal/low urgency; critical is reserved for a real critical condition.

## Answer, resume and close

Use `notification-hub conversations wait --ask-id <id> --deadline <bounded-wait>` when waiting through the owner can support the active work. Otherwise continue independent work and reconcile the canonical ask at the next material review or owner wake. A timed-out wait is not an answer. Do not create a private polling loop or scheduler.

Read the answer through its owner and validate whatever authority the intended operation actually requires. Do not answer your own request on behalf of the operator or impersonate an independent human. A reply received elsewhere must use a supported owner reconciliation path; do not forge a canonical answer. Ordinary operator direction still authorizes ordinary in-scope work when no extra grant is required.

Update the affected brief/checkpoint, resume the exact dependent step within the remaining allowance, and record its observed result. If another condition still blocks progress, explain it and continue other useful work. Record pickup pending when the supervisor is absent. Verify the actual wake/recovery route before claiming unattended follow-through.

If a request becomes stale, use a supported withdrawal operation when one exists. When none is exposed for that request kind, stop reminders and mark it superseded in the research view while retaining the unresolved owner state. Never simulate withdrawal by choosing an answer. Keep requests for later commissions distinct from the effects of the current one.
