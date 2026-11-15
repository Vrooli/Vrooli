# Security — Notification Hub

Threat model, secret handling, and the authorization boundary.

## Purpose Of This Document

This scenario can be told anything by any scenario in the fleet, and it
sends what it is told to a human's locked screen. That makes two questions
security questions rather than product questions: what may appear on that
screen, and who may cause something to appear there.

## Data Sensitivity

Notification bodies carry whatever a caller puts in them. The scenario
does not inspect them and does not guess.

`sensitivity_label` is required at ingress and is never inferred. Only the
caller knows whether a body is safe on a locked screen or in a third
party's logs. A channel is treated as unapproved for a label unless it
declares approval; an unapproved channel receives a content-free pointer
to the console instead of the body, and the notification is never dropped
silently (OT-P0-010).

Web Push payloads are encrypted end to end under RFC 8291 with keys held
only by this scenario and the recipient's browser. No push service can
read a body. This property is why the self-hosted relay alternative was
rejected: that design kept bodies on an owner-run server but required the
phone to reach it at delivery time, degrading to a contentless placeholder
when it could not.

Delivery error reasons appear in the timeline and the UI. An adapter must
never place body content into an error reason.

## Auth And Authorization

**Identity is not owned here.** There is no password, no profile table,
and no scenario-issued API key (OT-P0-007). Owner tokens are verified
locally against `scenario-authenticator`'s published JWKS through
`api-core/owneridentity`, with no per-request callback. A recipient row is
a projection of a verified external identity.

Owner-only mutations fail closed. Notification and decision-request creation
is intentionally identityless and always stores to the configured recipient;
the per-source push bucket throttles device side effects only. Already-accepted
notifications continue to drain, because their recipient was resolved at
acceptance time. Ordinary notification, delivery, routing,
recipient/device metadata, and preference reads do not require an interactive
login; ask reads (`GetAsk`, `ListAsks`, `Wait`) do require a verified identity and
are scoped to the reader (see "Who reads an ask" below). Recipient data uses the Hub's configured recipient, and catalog reads
return only source/category metadata. Recipient lookup does not create a row;
routing status retains its current status-snapshot refresh, with cache keys limited to configured target IDs or `local`. Notification detail receipts omit push provider IDs for every caller on this public configured-recipient projection. Reading an ask does not authorize
answering it. Ask answers, subscription/credential operations, and the
recipient/device/preferences writes covered here retain their existing checks.
Other endpoints are unchanged by this read rule.

**Attachment bytes are held to the standard of answering.** `GET
/api/v1/asks/{ask_id}/attachments/{name}` requires a verified owner identity
through the same path the Answer handler uses (`identity.Subject`): a verified
Cloudflare Access login (the `Cf-Access-Jwt-Assertion` header Cloudflare adds to
every request, including `<img>`, `<video>` and fetch), the loopback owner in
`personal_local` mode, or the owner's scenario-authenticator bearer or cookie.
A request with none is `401`; an assertion that fails verification is `401` and
does not fall back to a weaker credential; an ask that is not the owner's, an
unknown ask, an unknown attachment and a bad `?path=` are the same `404`. A
self-declared subject header is honored only where no verifier is wired (tests).
This is stricter than `GetAsk`: a run or service that sent or raised an ask can
read it but is never served its attachment bytes.

**Decision writes and attachment uploads need no identity or approval.**
`PutAttachmentBlob` accepts anonymous callers under the existing hash, 64 MiB
per-file, 1 GiB unreferenced-upload quota, and two-upload concurrency limits.
Anonymous callers cannot use `StatAttachmentBlobs`, which reveals shared hash
inventory; the CLI falls back to direct content-addressed uploads. Ask
preparation strips anonymous `source_path` and `source_host`, so the Hub never
probes caller-selected host paths. Uploaded bytes are retained for asks subject
to the same packet limits. `Withdraw`, reads and owner answers still require
their existing identity checks.

**The drift check is owner-only.** For an ask whose sender ran on the Hub's host,
`GetAsk` hashes the sender's `source_path` and reports whether it still matches the
stored copy, but only when the reader is the owner. A run or service reading an ask
gets `source_status: unknown` and the Hub reads no host path for it, so a verified
run that can reach the Hub but not a file can no longer learn whether `/some/path`
exists or whether its bytes equal a hash it chose. For the owner the bounds remain:
only verified senders' retained source paths are checked, the read is limited to regular files
of at most 64 MiB, a read hashes at most 64 MiB, and a result without a hash
comparison is `unknown`, never `fresh`.

**Who reads an ask.** `GetAsk`, `ListAsks` and `Wait` refuse a caller with no
verified identity (`Unauthenticated`) and show an ask only to:

- the **owner** (Cloudflare Access login, the loopback owner in `personal_local`
  mode, or the owner's scenario-authenticator bearer or cookie): every ask of the
  configured recipient;
- a **verified agent run**, for the asks of its **conversation lineage**: the asks
  recorded with the run's own id (`asks.created_by = run:<id>`) and the asks sent
  from the same verified conversation (`asks.created_conversation`). Agent Manager
  signs a `conversation_id` claim into every run identity. A recovered run and a
  child run inherit their source's conversation, and an attached run's conversation
  is derived from its harness session, so re-attaching the same session continues
  it; a token with no such claim (minted before the claim existed) falls back to
  its run id alone. `ListAsks` shows only this lineage. A request that presents an
  agent identity is that run and never the owner, even where a loopback caller
  would be;
- the same run, for **any ask whose exact `subject_key` it names**, when it is a
  run of the **same owner account** as the run that sent the ask (`asks.created_subject`
  equals the reader's verified owner, both non-empty). The subject key is a
  capability, the same one that already lets a run revise an open ask: an agent
  that lost its run can re-derive the key and `get`, `wait` on or `withdraw` the
  ask, and it never enumerates other asks by it. An ask sent with no verified
  owner, and every event-sourced ask, has no such capability;
- the **service that raised it as an event**: a scenario-authenticator service
  token with the exclusive audience `scenario-authenticator:notification-hub-ask-reader`
  and scope `notification-hub:ask-read`, from a principal named `<source>-ask-reader`,
  sees only the event-sourced asks whose source is `<source>`.

Every other ask is reported as not found, exactly like an ask that does not exist,
so ids cannot be probed. `ListAsks` (with its `query` and cursors) returns only the
caller's lineage. Each ask records its creating identity when it is sent
(`created_by`: `run:<id>`, `owner:<subject>`, or empty; `created_conversation`;
`created_subject`); an ask sent with no identity, every event-sourced ask, and every
ask stored before these columns existed have none and are owner-only for anyone not
in their lineage. `WithdrawDirect` follows the same rule: by `ask_id`, only the
lineage (or the owner) may close an ask; by `subject_key`, naming it is enough for a
run of the same owner. A run of another owner that reuses an ask's subject key
revises it, as before, but reads nothing back, and the packet check says so
(`ask_read_restricted`). Autoheal reads its remediation decisions as the
`vrooli-autoheal-ask-reader` service. Every reading scenario uses the one client in
api-core, `askreader.Tokens`, which holds the principal naming, audience and scope
the Hub verifies (`askreader` is the single source for both sides) and refuses any
token that is not exactly that scenario's ask-reader identity. The credentials are
issued by the one operator command that issues every service principal,
`scenario-authenticator service-principals sync --apply`, from the
`.vrooli/service-principals.json` each reading scenario ships (preview with
`scenario-authenticator service-principals sync`): issuing needs a verified human
issuer, and an agent run is refused the local owner exchange and `--apply`, so no
scenario can issue its own. A declaration is only a request; the operator reads the
scope diff the dry run prints and applies it, which is the grant. Without a
credential autoheal approves nothing.

**Resolutions are confirmed with the Hub, not trusted from the bus.** The signed
`notification_hub.ask.resolved.v1` event carries the operator's answer to the scenario
that raised the ask, and each consumer's effect applies it (agent-manager
authorizations and run approvals, storage-manager deletion grants, deployment-manager
releases, swarm-manager proposals, data-backup-manager recovery, autoheal
remediation). The signature proves vrooli-events delivered the body; the bus accepts
an event from any caller with any `source`, so a body can be forged. The shared
`operatordecision.Webhook` therefore reads the ask back (`GetAsk`, as the scenario's
`<source>-ask-reader`) and applies only what the Hub says, including `answered_by` and
the `correlation` that locates the consumer's own record, which `GetAsk` returns for
this purpose. A body that disagrees with the Hub, a pending ask, or an ask the Hub
does not hold for that source is refused (409) and left for redelivery; an unreadable
Hub is 503. A consumer with no ask-reader credential yet applies the body and counts
and logs each such resolution until the operator runs
`scenario-authenticator service-principals sync --apply`. Rollout order: restart the
Hub (it must return `correlation`, or confirmed resolutions are refused as mismatches),
then vrooli-events, the consumers and the publishers, once; then apply. Nothing
restarts after the apply: a consumer reads its credential when it reads an ask, so its
next resolution is verified, and a publisher looks again for a missing credential every
minute.

A refusal reaches the operator: a consumer that refuses a resolution the Hub's record
contradicts sends one high-urgency notice through its own `<source>-notification-hub`
sender principal ("Something tried to apply an answer you didn't give for ..."), once per
ask and reason however often the bus redelivers, with one summary instead of a flood after
five in ten minutes. The notice quotes the Hub's own question for the ask, never the event
body. Outages and unheld asks do not notify. See
`packages/api-core/operatordecision/README.md`.

**A context link a sender supplies is validated.** `Send` (`context_url`) and `Relay`
(`context_url` in the payload) take the link the operator taps. `hub.ValidateContextURL`
accepts only a Hub-relative path (`/asks/<id>`) or an http(s) URL with a host and no
embedded credentials, and refuses everything else with the problem named
(`InvalidArgument`): `javascript:`, `data:`, `file:` and every other scheme, `//host/...`,
a scheme-less host, `https://trusted@evil`, backslashes, whitespace, control characters
and links over 2048 characters. The same check runs on the event path (`*.notification.v1`,
decision events, the heartbeat notices), but never rejects: an event whose link fails is
stored with the link dropped, the drop is counted per producing scenario
(`integrations.DroppedContextLinks`) and logged with the reason (not the link, which can
hold a token), so a bad link cannot lose a notice. Direct `Ask` (and `CheckAsk`, its dry run) refuses
only the schemes that run code (`javascript:`, `data:`, `vbscript:`; whitespace inside the scheme
is ignored, as browsers do) with `InvalidArgument` naming the scheme. It keeps accepting a local
`file://` path and a relative path, which the operator's agents legitimately link and the packet
check only advises on (`local_context_url`).

**Request creation does not require a sender identity.** Direct writes (`Send`,
`Relay`, `Ask`) accept identityless callers; a verified scenario sender is optional
and supplies source attribution and read access to its own asks. The signed event
webhook uses a separate trust path: one shared HMAC proves vrooli-events delivered
the event, not who published it, and `source_scenario` comes from the envelope. See
PROBLEMS.md for event-publisher provenance. Idempotency keys of direct producers
share one namespace per recipient, so producers should prefix keys when retries
could collide.

**Files follow the ask, never the event.** Attachments may be uploaded by an
identityless caller or a verified producer. `StatAttachmentBlobs` remains
verified, while `PutAttachmentBlob` is identity optional. `Withdraw` admits the
verified owner or a verified sender for its own asks (an identityless caller is
`401`, a verified service token that is not a sender is `403` naming the mismatch), and a direct sender withdraws only asks
stored under its own source with `source_event_type` empty, so it cannot close an
event ask or another scenario's. An event, whoever it names as its source, never
brings an attachment or a question into its ask. That is how Research Desk's
explainer pages reach the operator: the desk sends as itself, not as an event.

**Host programs can create notifications and asks without sender setup.** A timer
or service-manager process may call `notifications send` or `conversations ask`
without an agent identity, sender principal, or permission request. The Hub stores
the request when its source's device-push bucket is empty and suppresses device
pushes until tokens refill. This creation path does not grant access to read,
wait on, or withdraw asks; those operations retain their separate access checks.

**Request creation is separate from owner identity.** Send, relay and decision
asks do not require a verified identity, scope, grant or consent. A verified run
or owner selects its own bucket; callers with no identity share the anonymous
bucket. A scenario may instead send **as itself**: a scenario-authenticator service
token with the exclusive audience `scenario-authenticator:notification-hub` and scope
`notification-hub:send`, from a principal named `<source>-notification-hub`, is a
verified sender. The Hub reads `<source>` off the principal name, records it as the
source of every notice and ask the token sends, and gives it its own bucket; no
request field names a source, so a sender cannot send as another scenario. A sender
reads back the asks it sent (`created_by = service:<source>`) and no others. A
presented bearer that verifies as a service but is not a sender (another audience,
no send scope, a name that is not `<source>-notification-hub`) is refused with the
mismatch named, never downgraded to the anonymous bucket. Invalid credentials still fail verification. The leaky bucket suppresses
device pushes after its capacity is spent; every request is still stored and
other channels continue. This does not grant owner identity: answers, dismissals,
preferences and subscriptions require the owner.

**Trust posture selects the recipient model.** `trust_posture` in
`~/.vrooli/state/vrooli/vrooli-onboarding/operator-state.json` is `personal`, `shared`, or `hosted`.
Multi-user routing is a data consequence of keying recipients by
authenticator identity, not a separate feature: in `personal` there is one
recipient and the UI shows no picker; in `shared` or `hosted` the same
tables hold several.

**Cross-node authority belongs to `vrooli-bridge`.** This scenario never
decides whether it may reach a machine. It asks; the bridge checks the
verb against the derived scope catalog and the node's granted scopes, and
refuses or carries the call.

## Secrets

| Secret | Where it lives | Rules |
|---|---|---|
| VAPID private key | Scenario-owned storage, never in the repository | Rotating it invalidates nothing, but the public half is baked into every existing subscription, so rotation requires re-subscribing every browser. Treat rotation as a migration. |
| Push subscription material | `push_subscriptions` table | Credential-equivalent. Anyone holding an endpoint plus its `p256dh` and `auth` keys can push to that browser. Never logged, never exported, never in a support bundle. |
| SMTP credentials (OT-P1-004) | Credential authority, referenced by descriptor | Not stored by this scenario. |
| Twilio credentials (OT-P2-002) | The `twilio` resource's credential descriptors | Not stored by this scenario. |

The scenario holds no owner password and mints no token.

## Threat Model

| Threat | Impact | Control |
|---|---|---|
| A caller sends a sensitive body to a channel that shows it on a locked screen | Content disclosure to anyone holding the device | Required `sensitivity_label` with default-deny per channel and a content-free fallback |
| A stolen `push_subscriptions` row | Attacker pushes arbitrary notifications to that browser | Treat the table as credential storage; never export; delete on `410 Gone` |
| An expired subscription that still looks healthy | Every send silently succeeds into nothing, and the owner believes they are covered | OT-P0-014: delete on gone, renew on change, and report the channel unavailable rather than healthy |
| The public origin lapses | Every subscription on every device dies at once | OT-P0-015: core-tier exposure, never auto-expired |
| A compromised scenario floods the spine | Notification fatigue, the owner mutes everything, and every downstream promise is void | Rate limiting per caller, duplicate suppression, and quiet hours |
| A notification body reaches a remote machine that should not see it | Content disclosure across the fleet | Sensitivity policy is evaluated before dispatch, not on the remote side |
| Widening the bridge dispatch vocabulary widens every node at once | A node can be asked to run commands the operator never intended | The vocabulary refactor must ship with narrowed per-node grants; a bare effect scope must stop meaning "every verb of that effect" |

## Security Gaps

Recorded honestly rather than deferred silently.

- **Ask reads are scoped, but three paths still reach ask content.** The
  rewrite above closes the identityless `GetAsk`, `ListAsks` and `Wait` read. What
  remains, stated as it is:
  - *Notification reads are still open.* An ask is also a notification whose title
    and body carry the question, the recommendation and the default (never the big
    picture, technical details, answers, notes or attachments). Notification `Get`
    and `List`, and the delivery timeline, still need no identity under the
    documented read boundary, so any process that can reach the API can read every
    ask's question text there, and every event-sourced ask's too. Closing it
    needs the same rule applied to notifications that are asks.
  - *In `personal_local` mode, loopback is the owner.* Any local process without an
    agent identity is then the owner and sees every ask. That is the explicit,
    operator-chosen meaning of `VROOLI_AUTH_MODE=personal_local`, selected only
    through the Hub's declared `authentication.default_mode` and never inherited
    from the shell that restarted the Hub (and the same fact
    that makes `Answer` reachable), not something this change can undo. In any
    other mode the loopback is not the owner and the refusal holds.
  - *Event-sourced asks have no verified sender.* Anyone who can publish to Vrooli
    Events can raise an ask under any source name, so the source label proves
    nothing about who raised it. What the rule guarantees is the reverse: only a
    holder of that source's `<source>-ask-reader` principal can read back asks
    labelled with it. A forged ask can still reach the operator's screen; that is
    the creation boundary above, unchanged.
  - *Same-owner runs that know a subject key can read that ask.* The subject key is a
    capability for any verified run of the asking owner, and a subject key is not a
    secret: it is a name the producer chose to be re-derivable. So an unrelated run
    of the same owner that guesses or is told the key can get, wait on, revise and
    withdraw that ask, though it cannot list it. The owner account is the trust
    boundary here, not the run or the conversation. A run with no verified owner
    gets no capability, and a different owner gets none.
  - *Conversation lineage trusts Agent Manager's claim.* A run reads every ask of its
    signed `conversation_id`. That is right for recovered, child and re-attached
    runs, and wrong only if Agent Manager ever put unrelated work in one
    conversation.
  - *Pre-existing asks reach their sender only by key.* Asks stored before
    `created_by` existed have no creator or conversation, so a run that sent one
    before the upgrade can no longer read it by id (it can by subject key only if
    the ask has a verified owner, which these do not). They age out with the 90-day
    retention.
  - *Research Desk reads its asks back only once it can read its credential.* It uses
    the shared `askreader.Tokens` client, but its Go module cannot yet link the
    credential authority binding (the governed install is refused by
    scenario-dependency-analyzer), so its read says why and the desk falls back to
    "answer in the desk" until that install lands.
- **No rate limiting at ingress yet.** Notification fatigue is named in the
  PRD as the real failure mode, and duplicate suppression only covers
  repeats of the same key. A caller emitting distinct notifications in a
  loop is currently unbounded.
- **No per-caller authorization.** Any authenticated caller may notify any
  recipient on this install. Acceptable at `personal` trust posture;
  it is a gap the moment the posture changes.
- **Escalation chains can leak across sensitivity tiers.** An unanswered
  critical ask escalates to the next channel in the chain, and that
  channel may have a lower approval level than the first. Escalation must
  re-evaluate the sensitivity policy at each step rather than inherit the
  first decision.
- **The bridge grant model is currently coarse.** A bare
  `vrooli-bridge:write` grant satisfies any write verb in the dispatch
  vocabulary. This is bounded today only because that vocabulary is small.

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — retention and privacy notes.
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency trust boundaries.
- [`PROBLEMS.md`](PROBLEMS.md) — open defects.
- [`../../PRD.md`](../../PRD.md) — OT-P0-007, OT-P0-010, OT-P0-014, OT-P0-015.
