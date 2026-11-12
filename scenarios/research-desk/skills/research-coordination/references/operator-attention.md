# Operator attention and research decisions

Read when a study appears to need the operator, when reviewing unanswered requests, or when handing supervision to another agent. The supervisor owns triage and follow-through across its commissioned topics. Notification Hub owns questions, answers and delivery receipts. The operation owner retains authorization and execution. Deliver as the charter's Decisions section says. The delivery sections ("Structure every operator delivery" to "Route and reconcile requests") serve a scheduler that delivers: always under a charter without that section, and under a charter built from `templates/` only for its fallback.

## Decide whether to ask

First discover supported routes and inspect the latest evidence, including superseding results. A missing tool, socket, environment, build, target fact or requirement does not by itself establish a human dependency. Reuse authorized sources, propose a contract, repair an understood blocker within scope, or choose independent useful work. Bind every gate to the actual subject and commission; a wallet implementation gate does not restrict unrelated validation research.

| Condition | Supervisor action |
|---|---|
| Agents can resolve the uncertainty or restore a permitted owner path | Assign that work. Do not ask the operator to diagnose it. |
| Only the operator can supply personal access, a preference, a private fact or an explicitly human-only decision | Prepare a concrete request with context and the exact next step it unlocks. |
| A human gate exists but the proposal or evidence is not reviewable | Finish the permitted preparation. Retain the gate as future work, not an actionable request. |
| Later evidence already removed the dependency | Mark the research item superseded. Do not send the stale request. |
| The original allowance ended or no supervisor is running | Preserve the prepared request and pickup state. Do not imply that an answer restarts the commission. |

## Needs you first

`ATTENTION.md` and cohort `STATE.md` are navigable views of what needs the operator. A Hub ask is answered in Hub; never repeat it as unanswered after compaction.

- Open every reply to the operator, and `STATE.md`, with the `Needs you` items (grammar under
  "Keep one navigable view").
- Each item opens with one or two plain sentences: the decision, your recommendation, and what it
  unlocks, followed by the link or command that answers it. Put no hash, run identifier or
  evidence path in the item; link the packet that holds them.
- Order: decisions that gate a showcase first, then proposals, then older actionable items.
- An item stays in the list, in every reply, until the operator answers it or it is withdrawn.
  A clause inside a status paragraph is not a delivery.
- Review packets that gate a showcase are one item: what to open, how many there are and how
  long the review takes. When a Notification Hub ask owns the choice, link its answer control.
- For a decision delivered as a Notification Hub ask, direct the operator to the Hub ask's
  answer control; do not ask for a chat reply. Read the canonical ask state before every
  delivery and at the next supervision unit. Record the answer verbatim in `ATTENTION.md`
  history and the topic's `MISSION.md`, update the current projection, and act on it in the
  next unit. If an operator independently supplies an answer in chat, reconcile it immediately
  with the existing canonical ask/owner when supported, then use that owner state as canonical;
  never keep soliciting or re-presenting a decision already answered. A canonical authorization
  grant still needs its own approval page.

## Keep one navigable view

Use the research home's `ATTENTION.md` as the consolidated index, or the existing cohort record when work is not retained there. Keep one writer across concurrent studies. Workers report candidate requests with evidence; the supervisor deduplicates and validates them. Link the view from the research home, topic and checkpoint rather than duplicating the request in each.

For each ready item, retain a stable identity, affected topic/commission, exact action, why the operator is necessary, alternatives checked, recommendation and consequences, evidence/artifact links, genuine deadline or review point, canonical ask ID, last owner-observed state, and resumption owner/step. Keep notification receipts separate from ask state. Record whether anything actually unblocked after the reply. The view is a dated projection, never an approval ledger or an alternate answer authority.

Keep actionable requests separate from agent prerequisites, future gates and superseded items. A supervisor reconciles them at launch, meaningful findings, new answers, approaching real deadlines and handoff/closure. An empty active queue does not require an invented question. State coverage; a skim of topic guides is not a complete audit of historical plans or accounts.

Keep exactly one `## Actionable now` section in `ATTENTION.md` and one `## Needs you` section in `STATE.md`, in one grammar (research-desk reads it: `docs/reference/FILE-CONTRACT.md` §5):

```markdown
### YTDLP-TARGET — yt-dlp target choice
One to three plain sentences: what is needed and why.
Recommended: keep — Keep yt-dlp uninstalled; use openly licensed footage.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: /abs/path/to/details.md
Secret: where the operator puts it
```

An item runs from `### <ID> — <title>` (ID `[A-Z0-9][A-Z0-9_-]{0,127}`, naming the decision, never a workstream) to the next `###` or `##`. Every decision has `Recommended:`; an item without it, or with `Secret:`, goes out as a plain notice; a secret's value never goes in a file. Option keys match `[a-z0-9-]{1,32}`. No prose before the first item; an empty section's only line is `No actionable items are recorded.` Identity is commission plus ID: a commission's item copied to `ATTENTION.md` gains `Commission: <id>`; one without it is global. After delivering an item yourself, add `Delivered: <route> <UTC>` to it; the desk never sends such an item. The `## Proposals awaiting approval` section above follows the same rules (empty: `No proposals are recorded.`); a missing or malformed section reads as unknown, not empty. Keep agent prerequisites and future gates in a concise section below. Move dated updates and answered or superseded requests verbatim to `ATTENTION-HISTORY.md` beside it, newest first. Preserve the detailed context beneath each heading. After acting on its answer, remove the item from `STATE.md` and `ATTENTION.md`.

Before dispatching a fresh commission and immediately before every delivery, read each relevant ask from Notification Hub and reconcile its latest owner state. Treat answered (including `other`), dismissed, withdrawn and expired as distinct terminal or review outcomes; none remains actionable unless the operator's answer created a new, explicit request. Move terminal items out of `Actionable now`, update the triage/checkpoint and recompute the inline digest before sending. Never copy an old actionable entry into a fresh commission without this owner-state check. If the owner cannot be read, mark the state unobserved and do not describe it as pending or resolved.

## Proposal packets

A `propose-then-build` topic ([missions](mission.md#modes)) sends each new maintained
capability, scenario or significant extension to the operator as a proposal, not as a question.
A proposal is how exploration becomes building, so make it easy to say yes: it opens with what gets built and the recommendation in plain words, and its hashes, run identifiers and challenge memos stay in the linked packet, never in `ATTENTION.md`. Dated updates go to the packet or `ATTENTION-HISTORY.md`, not under the proposal. Keep proposals in
`ATTENTION.md` under exactly one `## Proposals awaiting approval` section, above `## Actionable
now`, each headed `### PROPOSAL-<TOPIC>-<slug> — <title>`, with:

| Field | Content |
|---|---|
| In plain words | Two or three sentences a reader outside the programme understands: what the operator will be able to do afterwards that they cannot do today. |
| What Vrooli lacks today | Each missing piece, named. For each one say whether an existing resource or scenario really owns it (name the command that proves it) or nothing does. Software the prototype installed for itself is missing from Vrooli until a resource or scenario owns it. |
| What the prototype really did | What ran, on what inputs, and where its code and installed software live. Say what was real and what was hand-made or simulated: a stub that stood in for a client's services, workflows the kit wrote itself, synthetic data. |
| What gets built | Every resource, scenario and extension by name, what each one owns, and the first showcase. |
| Magnitude | Agent-hours per piece, and the showcase that would prove it landed. |
| Recommendation | Build, build smaller, or do not build, with the reason. |
| Options | Lettered choices the operator answers with: the recommended build, a smaller first step (say what it leaves out), defer and decline. Explain any product or term the operator may not know in its first sentence. |
| Cost of waiting | What the topic does meanwhile; it never idles. |

Size the build to what the work needs, not to what already exists. A new resource is the right
home for vendor software Vrooli will keep running (the first Freelance runtime proposal kept n8n
as a 2.3 GB private install in a research folder and called that "no new scenario"). A new
scenario is the right home for state and operations that no owner has (client engagements, their
rehearsal environments, simulated third-party services). "No new scenario" is a fact about a
design, never a reason to prefer it: the operator rejected three proposals on 2026-11-10 because
they shrank the build to wiring between existing owners and left out the parts that were
missing. Run `ecosystem-fit` on the proposed shape before filing, and state its verdict.

Deliver it as a Notification Hub decision ask with the options Approve, Approve with changes
(the `other` answer), Decline and Defer. When the operator approves, move the packet to the
topic's `MISSION.md` as a `build` workstream with status `approved` and the operator's answer
quoted under Operator intent. Staff it in the next unit with an owner who works on it for hours;
an approved proposal is implemented at the scale it was proposed. Record a decline under the
topic's Not this or workstream table so it is not proposed again unchanged. A topic with nothing
left to propose has an exploration lane that is not producing evidence, and needs an architect
pass.

## Review requests

A request for the operator to judge finished work is one self-contained HTML file, named by its
absolute path. The operator reads on a phone through the Web Console file drawer: no local
server, no links to sibling files, nothing fetched. For each item the page shows:

- what was asked for, in the requester's own words where they exist;
- the deliverable itself, and its full source code in collapsible blocks, so the operator can
  read the code without leaving the page;
- what it was tested on: who made the test target, how big it is, and which real targets it ran
  against;
- the hours spent and the lines of the deliverable;
- the weak points, and verdict buttons that open the linked Notification Hub decision ask.

Before asking, apply the trade's own bar: would a professional in that field send this to a
paying client? If not, it is not ready for review; finish it first ([proving
grounds](mission.md#proving-grounds)). The first Freelance review page showed no code, and the
operator deferred seven of its nine deliveries as too thin.

## Structure every operator delivery

Refresh the actionable section immediately before composing a delivery. Every Notification Hub ask or informational notification includes the full absolute path to the supervisor's current action file. For this research home, use `~/.vrooli/research/ATTENTION.md` expanded to its absolute path. Also include a compact inline digest of the other currently actionable items (stable ID and short title, with the needed operator action); do not make a local file path the only way to find them, because it may not open on a phone. Include only ready actionable items, not agent prerequisites, future gates, answered requests or superseded items. If the file cannot be read, include the known path, state that the other-item digest is unverified, and do not claim the queue is empty.

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
authorship from that field.

### Send and ask

A verified agent session or run sends and asks with no grant or consent: `notification-hub notifications send` for information, `notification-hub conversations ask` for a decision. Each run has a cooldown (default 6 sends and 3 asks per 10 minutes). Over the limit, the Hub refuses with `resource_exhausted` and the retry time, and stores nothing: retry the same terms and key after that time. Bundle updates into one notification; the cooldown is a flood guard, not a budget to spend. Answers, dismissals, preferences and subscriptions stay with the operator. For a decision on an enrolled effort's goal home, `agent-manager effort decision raise` is the alternative route.

### Route and reconcile requests

Use `notification-hub conversations list --open --limit <bounded-limit>` to inspect current asks and `notification-hub conversations get <ask-id>` for a selected one. Verify the configured recipient/channel through `notification-hub channels status`. An unavailable read stays unknown; do not infer that no ask or device exists. Do not alter recipients, subscriptions or quiet hours under a research commission.

Channel readiness does not prove delivery. For a genuine decision on an existing goal home, `agent-manager effort decision raise` is another supported producer route: the verified run raises a service-owned event that Hub translates into an ask. Qualify the actual repository-relative goal-home identity and immutable decision terms. Use `--big-picture` and `--technical-details` for context; do not disguise an informational update as a question. An unbound goal home can retain the decision/answer but has no effort signal or automatic wake; the active supervisor must reconcile it through `effort decision list`/`sync`. A real managed effort adds the existing signal/wake linkage. Do not fabricate an effort, invent a home reference for an unrelated path, or impersonate the owner. This source-qualified route is not a live delivery proof. If no eligible producer path exists, retain the concrete integration blocker and show requests in the current conversation.

When an item is ready, deliver it then; do not leave it only in the research index until handoff. For a real decision, use `notification-hub conversations ask` with the self-contained contents described above, at least two meaningful `--option key=label` choices, `--recommend`/`--recommend-reason` when evidence supports a recommendation, `--urgency normal`, the correct `--sensitivity-label`, a stable `--idempotency-key`, and an absolute `--deadline`. Put the full action-file path and the compact digest of other ready actions in the question text. Use `--context-url` only for a legitimately accessible context page; a private local file path is not a phone-readable web URL. For a fact rather than a choice, ask for the detail in an answer note with an explicit defer option only when deferring is a meaningful response. Do not manufacture choices to fit the API.

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
