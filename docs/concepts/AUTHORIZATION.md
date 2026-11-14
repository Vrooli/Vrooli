# Vrooli Authorization

This document defines the project-wide authorization boundary. It complements
[`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md), which
defines identity, providers, desktop modes, account linking, and entitlements.

The normative, test-pinned invariants that every change to the shared auth stack
must preserve — anonymous/forged/expired/over-limit refusal,
service-cannot-impersonate-human, run-identity-cannot-create-runs,
child-never-broadens-parent, one-decision-path-per-operation,
secret-custody-separate-from-identity, third-party-proof-never-trusted-as-human,
audience-mismatch rejection, and the scope-catalog wildcard matrix — are listed
with their enforcement path and pinning test in
[`AUTHORIZATION-INVARIANTS.md`](AUTHORIZATION-INVARIANTS.md), which also documents
the cross-scenario auth smoke gate and the Cf-Access binding startup self-check.

Three companion documents take the model further:

- [`AUTHORIZATION-APPROVAL-LIFECYCLE.md`](AUTHORIZATION-APPROVAL-LIFECYCLE.md) —
  the request state machine (pending → approved/denied/expired/superseded), the two
  origins (standing-grant ask vs interactive), the notify path, and retry-based
  resume.
- [`AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md`](AUTHORIZATION-CROSS-BOUNDARY-PROPAGATION.md)
  — what authority crosses a scenario boundary, what attenuates (child ⊆ parent),
  what must never happen, and the loopback-only continuity contract.
- [`../guides/working-on-auth-infra.md`](../guides/working-on-auth-infra.md) — how
  to change the shared auth stack without breaking the fleet.

## Four decisions, four owners

```text
Authentication      Who is this principal?
Capability          May this principal use this scenario/effect?
Domain authorization May this principal act on this exact object?
Entitlement         Has this customer paid for or activated this feature?
```

These decisions may inform one another, but they must not be collapsed into a
single role or token claim.

| Decision | Owner |
|---|---|
| Principal authentication | `scenario-authenticator` or explicitly configured provider |
| Scenario capability grants | Authenticator assignment plus `api-core/scopecatalog` |
| Resource/object authorization | Relying scenario |
| Agent-run attenuation | `agent-manager` |
| Commercial entitlement | LPBS or declared product commerce authority |

## Administrative boundaries

An administrator is an actor with authority in a particular control plane,
not a universal bypass:

| Control plane | Administrative decisions | Where enforcement lives |
|---|---|---|
| Identity | User lifecycle, credentials, MFA, sessions, roles, and coarse capabilities | `scenario-authenticator` |
| Product | Business accounts, billing, downloads, entitlements, and product data | LPBS or the relying scenario |
| Installation | Provider selection, deployment mode, secrets, backups, and service lifecycle | Vrooli control/deployment surfaces |

The same person may be an administrator in more than one control plane. A
relying party must preserve the actor context when it asks the identity
provider to perform an administrative operation. The target flow is:

```text
LPBS/product admin session
        -> verify actor and local product permission
        -> resolve scenario-authenticator by api-core/discovery
        -> authenticated API-to-API management call
        -> authenticator checks identity-management capability
        -> audit actor, target, operation, result
```

The browser does not call the authenticator management API directly. A
product administrator must not gain identity-admin authority merely because
the product has an administrator role. The product-to-authenticator mapping
must be explicit, scoped, revocable, and auditable.

Identity-management operations such as unlock, credential-reset initiation,
MFA reset, session revocation, and capability assignment belong in the
authenticator's management contract. If a management operation is not yet
implemented, the relying party must expose it as unavailable rather than
recreating the operation against authenticator storage.

## Capability vocabulary

The current coarse vocabulary is:

```text
<scenario>:<effect>
```

Supported effects are:

- `read` — observe or retrieve data;
- `write` — create or modify non-destructive state;
- `destructive` — delete, publish, execute, revoke, or otherwise high-impact state.

Grants are additive. There are no deny rules in the coarse scope grammar. An
absent scope never grants authority. An issued token must carry an explicit
scope list, including an empty list when the principal has no capabilities.

The catalog is derived from scenario CLI and service declarations. The
authenticator stores grant values without importing scenario code or interpreting
scenario commands. This keeps the provider independent from relying-party
release cycles.

## Enforcement sequence

Every protected mutation follows this order:

1. Verify the presented provider credential.
2. Verify issuer, algorithm, key id, audience, expiry, and revocation state.
3. Resolve the canonical principal from request context.
4. Check the required coarse capability.
5. Apply relying-party domain authorization.
6. Apply exact approval, human-control, or entitlement rules when required.
7. Audit the decision and the mutation result.

The UI and CLI may explain or hide unavailable actions. They never make the
authorization decision.

## Scope is not object authorization

`git-control-tower:write` may permit a principal to reach a Git mutation
surface. Git Control Tower must still decide whether:

- the repository is in scope;
- the current branch is protected;
- the operation requires a human;
- the actor is an allowed agent or service;
- the requested change has the required evidence or approval.

Similarly, `device-sync-hub:read` does not automatically grant access to every
device in every tenant.

## Agent delegation

An agent run receives the intersection of:

```text
owner grants ∩ agent profile grants ∩ requested grants
```

The child token or provenance record must be narrower than its parent and must
expire no later than its parent. Human-only capabilities are not materialized
into agent grants unless an explicit reviewed policy allows them. The
operator-approved Web Console session delegation below is that policy for an
attached run: it supplies only the exact scopes in its verified owner proof.

For an agent attached to a Web Console session, the verified, session-bound
owner delegation is that attached identity's scope ceiling: the attached agent
token carries exactly the verified delegation scopes and expires no later than
the delegation. A separately authenticated in-context human attach uses that
verified principal's exact scopes and expiry when the proof has enough lifetime
to produce a valid whole-second token expiration. A missing-expiry in-context
principal may identify the owner but contributes no scopes. If neither proof is
usable, the attached token has no owner and no scopes; invalid, expired, or
session-mismatched delegations cannot provide authority. Scope-free attached
identity keeps its existing run-token lifetime; scopeful identity expires with
its owner proof. This is the current operator-approved
behavior and supersedes the earlier attribution-only attached-token description
in the Phase 5 research trace. Attachment still does not resolve a managed agent
profile or requested-scope set; managed child runs continue to intersect the
retained owner ceiling with their profile and requested scopes. Explicit
operation grants remain required for operations governed by those grants.

**Attached ordinary identity continuity.**
An attached root's ordinary identity is separate from its delegated capabilities. When its inherited token expires, the broker may use that signed token only as a lookup anchor. Renewal requires the same active run, frozen anchor claims, live lease, Web Console session, kernel process incarnation and registered scope. Every renewed generation has zero capability scopes. It preserves the run and owner attribution, but does not renew the expired owner delegation or grant authority to protected operations. Explicit API verification remains expiry-strict.

The persisted renewal anchor identifies an initialized generation. An original owner-attributed token may already have an issuance timestamp; that alone does not make it a renewal generation. Once the anchor is recorded, its issuance state must be present and reconstruct the exact active token. Cancellation intent, revocation, terminal status, changed owner/anchor/binding or failed proof refuse renewal. The original process environment may retain its anchor while subsequent broker-backed requests resolve the current ordinary identity. Independent operation grants remain separately verified and bounded; identity continuity does not create or extend them.

Startup recovery retains externally attached sessions under their process owner rather than managed transcript recovery. An exact missing-parser interruption can be corrected only after the current session binding is proved and a lifecycle CAS fences cancellation, token replacement and process changes. This correction preserves credentials, original start time, owner and outcome evidence.

A224's reviewed consumer v8 and `run identity` CLI v1 were published together,
passed their focused gates, and were installed with the normal control-plane
install and Agent Manager restart. The bounded receiving check for original run
`57b2ba0c-f1d1-4d55-9516-cd1849eac842` reports it still RUNNING/EXECUTING/ATTACHED;
`agent-manager run identity --json` returns that exact run as valid with zero
scopes and no token fields. The joined owner proof matched its Web Console
session, process incarnation, and registered agent cgroup. These receipts qualify
identity continuity for that ownerless run; they do not establish a fleet-wide
recovery guarantee or an operation grant. See
[`A224 adoption and receiving evidence`](../internal/delivery-reliability-research/sources/A224logs/publication-v8-cli-v1/adoption-receiving-summary.md),
[`sanitized native identity result`](../internal/delivery-reliability-research/sources/A224logs/publication-v8-cli-v1/lifecycle/run-identity-final-sanitized.json),
and [`joined native owner proof`](../internal/delivery-reliability-research/sources/A224logs/owner-proof-candidate-v5/native-receiving-summary.json).

The original RCL effort's saved feedback is recorded as signal sequence 7.
Its first parent wake returned HTTP403 while the prior operation grant was
expired. After the operator approved cb8f0c37… for16h on October18 at09:41Z,
the same supported native wake succeeded at09:42:42Z for the original retained
run. Grant approval, successful wake, receiver acknowledgement and E3 acceptance
remain separate claims. A235 signal8 supplied the exact immediate-anchor source diff; the original owner acknowledged it and accepted E3 at10:03:08Z, preserving E1/E2 as unaccepted and no release publication. See the
[`signal receipt`](../internal/delivery-reliability-research/sources/A215logs/feedback-signal-native-receiving.receipt.json)
and [`wake receipt`](../internal/delivery-reliability-research/sources/A215logs/parent-wake-after-A224.receipt.json).

**Active independent grant and ordinary identity reporting (A234 adopted).** An active compatibility grant can be the default local broker credential for protected operations. That grant must not replace the original ordinary identity reported by `run identity`. After an inherited credential fails strict verification, the CLI requests `purpose=ordinary_identity` from the existing peer-checked local broker; that purpose returns only a verified same-run baseline. Default credential behavior, child isolation and strict API expiry remain unchanged. Unknown or repeated purposes refuse. The existing valid-inherited-token fast path remains unchanged. Native receiving now returns original root57b2ba0c… valid with zero scopes while the independent16h grant remains active;11focused tests, two compiling causal controls, guarded publication, normal restart and scoped API phase support this bounded claim. [A234 adoption and limitations](../internal/delivery-reliability-research/sources/A234logs/live-adoption/adoption-summary.md).

## The whole model in two questions

Capability vocabulary and the enforcement sequence above answer the common case.
For any request that crosses a scenario boundary — especially a **costly** one —
the model reduces to two questions:

1. **Who is calling?** — *identity*, one of three kinds (see
   [`IDENTITY-AND-AUTHENTICATION.md`](IDENTITY-AND-AUTHENTICATION.md) for how each
   is proven):
   - **Human** — a person, proven by a verified login token (scenario-authenticator
     JWT, Cloudflare Access, or the loopback desktop token).
   - **Agent** — an autonomous run, proven by a signed agent-identity token tied to a run.
   - **Service** — a scenario acting as itself (e.g. prompt-manager calling
     agent-manager), proven by an operator-issued service-principal token.

2. **May they do *this*?** — *authority*, decided by the operation's **tier**:

| Tier | Examples | What's required |
|---|---|---|
| **Read** | get/list runs, read a team, health | Any verified identity (human, agent, or service). |
| **Cheap write** | record a signal, mark a decision | Any verified identity. |
| **Costly write** | create/launch a run, global management, approvals | A verified identity plus the operation's required authority. Some operations require a human or a grant; Agent Manager agent-operation grant requirements are configurable as described below. |

This tier table describes additional operation-grant requirements. Explicit
endpoint capability and object-ownership rules still apply. An identity-only
read must not acquire an additional capability or grant requirement merely
because the caller also presents a grant for another operation.

**Notification Hub read exception.** Ordinary Notification Hub reads do not
require a human login or agent grant. Recipient-scoped reads (ask and
notification detail/list, delivery timeline/analytics, routing status, and
recipient/device/preferences metadata needed to inspect delivery) use the
installation’s configured recipient and return only that recipient’s data.
Recipient lookup is read-only (an absent configured recipient is not created);
routing status keeps its current bounded status-snapshot refresh and uses configured target IDs or `local` for cache keys. Notification detail receipts omit the push provider ID for every caller on this public configured-recipient projection. Catalog reads
are limited to the Hub's declared/observed source metadata. Ask
answers, notification creation, subscription/credential operations, and the
recipient/device/preferences writes covered by this contract retain their
existing identity and operation checks. Unrelated endpoints are not changed by
this exception.

Identity says *who*; the tier says *what it takes*. Reads and cheap writes need
only a known caller. Only costly writes — the ones that spend money or take
real-world action — need explicit authority, and there is exactly one decision
path that grants or refuses it. No gate silently overrides another.

### The one authority primitive: an OperationGrant

When an additional grant is required, its authority uses an `identity.OperationGrant`:
a bounded operation + target + limits + expiry, tied to an accountable owner,
evaluated by `OperationGrant.Allows`. A grant comes to exist in exactly three
ways, differing only in *timing*; all three converge on one representation and one
operator-notification path (notification-hub decision events). Anonymous,
over-limit, expired, and forged-owner requests are refused in every case.

1. **You're present (synchronous).** You act in a UI behind the Cloudflare secure
   tunnel (tunnel-manager); the request carries a verified `Cf-Access` human token,
   so you *are* the accountable human for your own clicks. When you **spawn an agent**
   from that session, the session mints a short-lived, session-scoped grant the agent
   inherits — proven human approval crosses the scenario boundary with it (see
   [`GATED-UI-AUTHENTICATION.md`](GATED-UI-AUTHENTICATION.md)).
2. **You approve on demand (ask, don't block).** An agent hits a costly op with no
   authority. Instead of a silent refusal, it raises a request: you get a
   notification and a chat link, approve from anywhere, a grant is issued, and the
   action proceeds.
3. **You pre-approved it (standing grant).** For recurring unattended work
   (team heartbeats, supervision ticks) you approve one bounded grant once; it is
   used without pinging you each time, within its limits, until it expires or you
   revoke it.

A **standing grant** is how a trusted service (or an unattended agent) does a
costly write without a human present each time. It is not "turn security off" — it
is the opposite: a small, operator-approved, durable record that makes the authority
explicit, bounded, attributable, and revocable. It reuses the `OperationGrant` shape
and its evaluator; the one durable piece is an operator-approved grant store keyed to
the caller's **verified identity** (subject, optionally session). The caller presents
only its identity; the authority check looks up the approved grant for that subject
and evaluates `Allows` plus limits. Approval writes the record; revocation deletes it;
expiry is enforced on every lookup.

```
grant: prompt-manager may run.create on agent-manager:*
  purpose:     team heartbeats
  owner:       <operator-subject>   # accountable for the runs
  limits:      ≤ N concurrent, budget ≤ X, expires <date|renewable>
  approved by: <operator-subject>, once
```

### One path per real situation

| Situation | How it's authorized |
|---|---|
| You, in the browser (Web Console) | Your human login token; you are the accountable human. |
| An agent you spawned through Web Console | Its verified session-bound owner delegation supplies baseline capabilities; an additional operation grant is required only for an explicitly governed operation. |
| A scheduled heartbeat / unattended tick | The team's standing grant. No human needed; still bounded and attributable. |
| One scenario reading another | Service identity alone. Just works. |
| One scenario launching work in another | Service identity **+** a standing grant for that operation. |

Ordinary operator-authorized host maintenance does not require an additional
grant unless that exact operation is explicitly governed. Additional grants apply
only to operations explicitly listed in a reviewed grant policy; they do not
follow merely from an agent identity.
A consent may carry several reviewed policies. Evaluate the requested operation
and target through the canonical rule evaluator; extra approved policies do not
invalidate a covered operation. Keep each rule's operations paired with that
rule's targets, and consume only the authority needed by the current operation.
When an owner-launched Web Console session performs an ordinarily authorized
operation, preserve its existing permissions through the control-plane call and
keep the actor identified as an agent—never impersonate a human. Capability,
object ownership, and physical-safety checks remain separate from grant checks.
Before treating a refusal as a missing-grant requirement, trace the actual
supported command and owner path (including whether it is standalone setup or a
normal restart); a safety fence does not itself create an authorization rule.

### Interactive grants for already running agents

A verified human may approve a bounded delegation to an existing live session. Agent
Manager owns that grant alongside its existing run identity and revocation state. The
control plane resolves and verifies a host session lease and process incarnation when
the session lacks a managed run. A Web Console attachment carries only the exact
scope ceiling from its verified, session-bound owner delegation, as described in
the agent-delegation section. Attachment does not infer scopes from a harness ID,
environment marker, PID, or Web Console `origin=ui`; none of those is human proof.
An agent cannot grant itself authority by invoking the command, setting a marker,
or exchanging its shared OS user credential for a human session. A deployment that
cannot separate same-user agents from owner credentials must use a separate human
approval path or explicitly describe its cooperative local trust model.

The operator surface is `vrooli agent authorize <session-or-run>` with an explicit
reviewed policy, object boundary and deadline, plus inspection and revocation.
Issuance requires authenticated owner consent at the configured approval origin:

```bash
vrooli agent authorize <session-or-run> \
  --policy swarm-bookkeeping --target 'swarm-manager:*' --expires-in 24h \
  --approval-base https://agent-manager.itsagitime.com
vrooli agent authorization <session-or-run>
vrooli agent revoke-authorization <session-or-run> \
  --approval-base https://agent-manager.itsagitime.com
```

Every command that refuses an agent for lack of authority names the request
command in its refusal. The shared owner is `authn.RequireOperation`: the error it
returns to an agent is an `authn.OperationRefusal` (still `errors.Is(err,
authn.ErrOperation)`) carrying the operation, the target, the narrowest reviewed
policy that grants that operation and the ready-to-run
`vrooli agent authorize … --policy <policy> --target <target>` command, so every
scenario that calls it inherits the text. Agent Manager's 403 bodies also carry it as
`request_authority`, and the CLIs print it. `--approval-base` is added when the
refusing process is configured with `AGENT_MANAGER_AUTHORIZATION_URL`; otherwise
`vrooli agent authorize` resolves Agent Manager's public HTTPS origin itself. An
operation no reviewed policy covers says so instead of offering a request. When a
live session cannot be proved (`live launcher session binding is unavailable`), the
refusal names the failing proof step and what to do next.

The commands that record an operator decision — run review (`run approve`,
`run reject`), an investigation's approved recommendations
(`run apply-investigation`) and workspace-sandbox's review result
(`run sandbox-sync`) — have no agent grant: the decision is the operator's. A
sandboxed run held for manual review (`approval_state` pending) and a waiting
investigation raise their decision in the operator's inbox themselves
(`agent_manager.run.decision_requested.v1`,
`agent_manager.investigation.decision_requested.v1`); a run that reached
`needs_review` another way (for example an in-place run whose completion could not
be verified, `approval_state` none) raises none, so there is nothing for the
operator's agent to hand over. sandbox-sync is the workspace-sandbox callback that
records a review result.

Agent Manager's additional grant requirements for verified agent operations are
operator-configurable in **Settings → Orchestration → Operation permissions**
and default to off. When off, verified identity, scope, ownership, and domain
rules still apply; when on, the corresponding reviewed grant below is required
unless an existing direct-lineage permission applies.
Service standing grants and explicitly human-only decisions remain separate.
See [Agent Manager configuration](../../scenarios/agent-manager/docs/reference/configuration.md#agent-operation-approvals).

Reviewed policies (`authn.InteractivePolicy`):

| Policy | Operations | Typical target |
|---|---|---|
| `swarm-bookkeeping` | `swarm.bookkeeping` | `swarm-manager:*` |
| `run-creation` | `run.create` (bounded unattended/service run creation) | `agent-manager:*` |
| `agent-investigation` | `agent.investigate` | `agent-manager:*` |
| `agent-management` | `agent.investigate`, `agent.manage` (stop, stop-by-tag, continue, wake, recover a named run, cancel or update a named task; `stop-all` only when the target covers every run) | `agent-manager:run/<id>`, `agent-manager:task/<id>`; `agent-manager:run/*` for `stop-all` |
| `run-deletion` | `run.delete` (permanently delete a named run; never run review) | `agent-manager:run/<id>` or `agent-manager:run/*` |
| `team-preparation` | `team.prepare` (disabled, undispatched draft revisions only) | `prompt-manager:team/<id>` |
| `team-management` | `team.manage` (team and membership owner mutations) | `prompt-manager:team/<id>` |
| `prompt-agent-creation` | `agent.create` (create one catalog entity only) | `prompt-manager:agent/<id>` |
| `effort-metadata` | `effort.reconcile` (owner-facing effort metadata only, such as the goal home) | `agent-manager:effort/<name>` |
| `delivery-management` | `effort.delivery` (start, pause, resume or stop the commission of an existing effort) | `agent-manager:effort/<name>` |
| `notification-sending` | `notification.send` (to an explicitly approved recipient only) | `notification-hub:recipient/<subject>` |
| `notification-decision-asking` | `notification.ask` (create a decision for an approved recipient only) | `notification-hub:recipient/<subject>` |
| `deployment-operations` | `deployment.manage` | `deployment-manager:*`, `scenario-to-cloud:*` |
| `nutrition-management` | `nutrition.manage` | `nutrition-planner:*` |
| `treasury-spending` | `treasury.spend` (the signed mandate, amount, recipient and kill switch still apply) | an explicit `treasury:` target naming the exact authorized spend |
| `treasury-wallet-reading` | `treasury.read` (the derived wallet) | an explicit `treasury:` target |

The catalogue behind this table is `authn.InteractivePolicy` (`packages/api-core/authn`).
A test fails when this table and the catalogue disagree. The same catalogue maps a
refused operation back to the policy that grants it, which is how every refusal
names its request command.

An exact `team.manage` grant also permits the target-scoped `GetTeam` read needed
to prepare management of that same team. It does not grant team listing or broaden
baseline Prompt Manager read access.

Delivery control is a delegated execution operation, separate from approval itself.
An explicitly approved `delivery-management` rule permits only commission changes on
the named existing, non-withdrawn effort. The agent keeps its identity; it does not
become a human operator. The owner still assigns clock timestamps, checks retained
owners, freezes the active budget/worker policy and fences revision/idempotency.
This rule cannot create an enrollment or change its subjects, destination, authority,
owner or withdrawal state. Existing metadata, team-management and run-creation grants
do not acquire this operation retrospectively. The operator approves in the browser;
the agent performs the existing native delivery command without a human shell step.

When run-creation approval is enabled, an unattended agent may create a top-level
run when the canonical evaluator verifies a grant for `run.create` on
`agent-manager:*`. This also works when the rule is one member of a reviewed
multi-policy grant. Agent Manager attributes that run to the grant approver. The new
run's scope ceiling starts with the separately verified grant-token scopes and
is narrowed by the normal requested/profile intersection. A paired baseline
identity remains separate and adds no scopes. The grant expiry, and any stricter
retained owner expiry, bound the new run identity. With approval disabled, a
verified agent with the write scope may create a run attributed to its verified
owner, bounded by its existing identity scopes and expiry. The existing
exact-parent child route and service-standing route keep their admission checks.

The task created immediately before an unattended run is an inert precursor, not
a separately governed `task.create` operation. Agent Manager admits that task
under the same configured `run.create` policy, including a covering rule inside
an approved multi-policy grant when approval is enabled. A live run whose current profile declares
`agent-manager:orchestrate` retains its existing intrinsic task-creation path;
that path is independent of any unrelated grant the run also carries.

`<session-or-run>` is any name of the live, kernel-verified session: the session ID
or owning run that `vrooli agent list` prints, the `vrooli-agent-…` scope name
(`$VROOLI_AGENT_SESSION` inside the session), or its PID.

A session uses only its newest approved grant, so one request should name every policy
the work needs. Repeat `--policy`, each followed by its own `--target` flags. The grant
pairs each policy's operations with that policy's targets only (`OperationGrant.Rules`),
so one policy's wildcard never extends another policy's operation:

```bash
vrooli agent authorize "$VROOLI_AGENT_SESSION" \
  --policy agent-management --target agent-manager:run/<run-id> \
  --policy effort-metadata --target agent-manager:effort/<name> \
  --approval-base https://agent-manager.itsagitime.com
```

A session has at most one pending request. A new, different request replaces it: the
old one becomes `superseded`, and approving it names its replacement. A pending request
past its review deadline reads `expired` everywhere. Neither state carries authority.
Preparing a request or printing a review URL grants no authority; the verified owner
confirms in the browser. New approval requests remain reviewable for seven days. The
permission lifetime starts when the owner confirms and lasts what the owner approved
(default 24h, at most 7 days), bounded by the retained run ceiling. A grant ends early
on revocation, a changed session process, or an ended run. When it lapses, the session
returns to the identity it held before the grant. Grant issuance preserves the actor as
an agent and records the human approver separately; audit records retain both
identities. A new grant does not rewrite historical run admission, relax a parent
ceiling, extend an expired grant's deadline, or reset consumed budgets.

Shared CLI identity resolution discovers the current run credential for each request
through a trusted local owner path bound to the calling process/session. It
distinguishes missing proof, invalid proof, expired/revoked grants, unknown provider
state, and denied operations; it does not discard offered invalid proof, select another
session's grant, or fall back from an agent to a human identity. This lets an existing
session adopt an approved grant without modifying its parent environment or restarting
Web Console. Cloudflare Access can provide the human proof for consent through the
shared verifier and canonical principal; capability assignment remains explicit and
does not arise from tunnel reachability, email headers, or copying a browser credential
into an agent environment.

### What this model deliberately does NOT have

- No second, always-on gate that silently overrides the enforcement decision. One place
  decides authority for a given operation, and everything honors it.
- No parallel finite-effort / kernel-broker / custody delegation system. A standing grant
  to a named caller does the same job in a form a human can read. The `effortauthority`
  broker package and its `VROOLI_FINITE_ENABLED` gate have been removed; the live
  effort-*delivery* surface (delivery teams, epoch delivery, goal homes, operator
  decisions, the `/api/v1/efforts` board) is a separate system and is unaffected.
- No duplicate `Admit/*` vs `Authorize/*` primitive families. One family.

### Approved operation policy changes

The operator selected the following scope after reviewing the human-gate
inventory. This selection supersedes recommendations to weaken other gates.

| Area | Approved agent authority | Retained human decisions and other conditions |
| --- | --- | --- |
| Nutrition Planner | Permit delegated read, write and destructive operations. Remove every Nutrition Planner-specific human-approval or human-only actor gate. | Retain authentication, owner/account boundaries, explicit grants, validation and audit. Removing human approval does not grant anonymous access. |
| Deployment Manager and Scenario to Cloud | Permit delegated preparation, configuration, provisioning, diagnostic work, execution and recovery within the approved deployment boundary. | Human decision gates apply to release approval/publication authority and evidence acceptance, attestations or waivers. Evidence collection, reading and automated evaluation remain delegatable. Existing credential-authority rules remain in force. |
| Agent Manager | Permit bounded agent investigations and delegated management of authorized runs. | Retain lineage, profile, target, budget and expiry limits. Preserve unrelated global control and approval rules. Delegated management does not permit self-approval or authority expansion. |
| Swarm Manager | Permit delegated backlog and goal bookkeeping, including updates, evidence-backed status reconciliation, consolidation and archival. | Preserve graph validation and audit. Strategic outcome acceptance, governed Development approval and other existing non-bookkeeping decisions keep their human rules. |
| Notification Hub | Permit recipient-bound direct send/relay through `notification-sending` (`notification.send`) and decision asks through the separate `notification-decision-asking` policy (`notification.ask`), both targeting `notification-hub:recipient/<subject>`. | Require independent human consent for each grant. Answers, preferences and subscriptions retain owner checks. Existing managed-profile sender scopes can send but cannot create decision asks. |
| All other areas | Preserve existing operation eligibility and human-control policies. | Git Control Tower preparation, commits, pushes and other writer gates are unchanged. Credential, identity, editorial, offer, device and other retained policies are not broadened. |

Release authority must bind the exact release and target. A delegated executor
may carry out an approved release without becoming its human approver. Evidence
acceptance must identify the accepted evidence, criterion and revision. A passing
automated test is evidence, not a fabricated human attestation.

Bookkeeping may reconcile completion against independent evidence or previously
recorded owner acceptance. It does not authorize a worker to accept its own
active delivery or assert that a strategic goal has achieved its intended
outcome. Archival retains history; hard deletion is not the default cleanup route.

### Shared authorization design

Reuse `api-core/authn`, `identity`, `cloudflareaccess`, `scopecatalog`, Agent
Manager run identity/revocation, and control-plane session leases. Introduce no
new identity realm or independently managed JWT signer for this workflow.

Separate these decisions in the shared operation boundary:

1. Authenticate the offered principal and reject invalid or conflicting proof.
2. Resolve the required capability and operation eligibility from owner metadata.
3. Validate the current delegation, operation, object and deadline.
4. Apply the relying scenario's ownership, evidence and human-decision rules.
5. Audit the actor, approver, operation, target and outcome.

Coarse capability eligibility and command `run_eligible` metadata must agree on
the selected policies. A coarse write grant alone cannot authorize every writer.
An operation that requires a human decision must check that decision explicitly.
Passive authentication status and a comment saying operator-only do not enforce
an operation policy. Missing agent provenance does not establish a human actor.

Web Console must establish the configured human principal at browser ingress and
record the verified owner separately from launch-origin metadata. The configured
Cloudflare Access assertion uses the shared verifier, including exact issuer,
application audience, signature, expiry and user-token checks. Resolve owner
capabilities through explicit provider/subject policy; do not infer grants from
email equality, tunnel reachability or an unsigned owner field.

Provider-neutral context replaces legacy bearer reparsing at caller adapters.
Apply each provider's declared audience instead of the frozen default
authenticator audience. Preserve the original agent actor across internal hops.
Forward only the proof appropriate to the intended trusted destination. Browser
credentials remain on the owner side rather than entering an agent environment.

### Governed program composition

Program Runtime verifies the agent proof at ingress through shared provenance.
Its runner passes only that verified agent proof over the private kernel pipe.
Execution-local bridge requests carry it only to loopback HTTP(S) owners;
credential-bearing requests disable ambient proxies and redirects. The kernel
resets this proof after execution. Browser/human proof is never supplied to the
program, and credentials are excluded from program source, environment,
receipts, caller metadata and durable session state.

The bridge verifies the offered proof again and forwards the original bounded
agent identity to the target owner. Every target retains its own operation,
object, expiry, revocation and independent-human decision checks. Composition
creates no broader grant or fresh permission lifetime. The local kernel is a
trusted agent execution surface, not containment for hostile code sharing the
operator's account; this transport preserves the agent's existing authority.

### Grant storage and resolution invariants

Grants live in Agent Manager's authoritative run domain. Each retains stable
run/session identity, verified owner binding, reviewed policy, operations, object
boundary, issue/expiry times, revision, revocation state, and approver attribution.
Secret custody uses the existing credential authority; diagnostics never print
plaintext run, owner, or grant credentials.

Grant issuance authenticates the human approver through a configured provider or an
explicitly qualified operator flow. It does not treat its own invocation, a TTY,
localhost, or a shared UID as human proof, and conversational authorization never
replaces the cryptographic caller proof required at issuance.

For an unmanaged session, the control plane resolves the live lease, owner
association, boot identity, and process incarnation before creating its attached
run, rejecting dead, reused, ambiguous, or mismatched bindings. A verified
Web Console delegation additionally binds the attached token to the exact owner
scope ceiling and expiry described above. An ordinary attached run with neither
that delegation nor a verified in-context human remains ownerless and zero-scope.
Attachment and grant approval are separate effects; neither launches a replacement
coding agent.

The resolver authenticates its requesting process/session through the trusted local
owner path and obtains the current approved credential for that binding before each
request. Offered invalid proof fails rather than falling back to another identity
source. Revocation, expiry, or terminal state prevents further protected requests even
when a previously issued token remains signed. A lease, process, or cgroup check
establishes attribution within the declared host trust model; it does not isolate
same-UID code from owner secrets, so the configured human consent boundary is required
rather than relying on file modes or an environment marker.

## Commercial entitlements

An LPBS entitlement may determine whether a product feature is available, but
it does not replace technical authorization. A desktop app may need both:

```text
valid local principal
    + required scenario capability
    + valid commercial entitlement
```

The entitlement must be scoped to a product, installation or audience, and
feature set. It must have an explicit expiry and offline behavior.

## Desktop modes

`personal_local` uses the OS user and private application boundary. It does
not require human sign-in by default. `local_multi_user` requires a local
identity provider and explicit account selection. `remote_vrooli` uses the
remote server's provider. `shared_provider` uses a broker-issued expiring
lease. See the canonical identity document for the full deployment contract.

## Relying-party implementation rule

Relying parties must use the shared provider-neutral identity and authentication
packages. They must not:

- parse JWTs independently;
- trust identity fields from request bodies;
- call authenticator validation on every request;
- treat email equality as account linking;
- treat UI visibility as authorization;
- issue broader child credentials than the verified parent;
- use a commercial entitlement as a substitute for domain policy.

The scenario-specific scope and authorization details remain in
[`../../scenarios/scenario-authenticator/docs/concepts/AUTHORIZATION.md`](../../scenarios/scenario-authenticator/docs/concepts/AUTHORIZATION.md).

### Owner-binding configuration

The interactive-grant commands shown under "Interactive grants for already running
agents" take their object boundary from `--target` and their approval origin from
`--approval-base`. `AGENT_MANAGER_AUTHORIZATION_URL` configures
the server's allowed browser origin and can supply the CLI's approval base.
`VROOLI_AUTH_OWNER_BINDINGS` is JSON deployment policy mapping exact verified
provider subjects to canonical subjects and capability lists. An empty mapping
does not infer capabilities from a Cloudflare login. For durable host policy,
`VROOLI_AUTH_OWNER_BINDINGS_FILE` selects a JSON file with the same binding array.
When neither inline policy nor an explicit path is supplied, shared configuration
reads `$HOME/.config/vrooli/owner-bindings.json` if present. Missing optional
policy grants no scopes; malformed, empty or unreadable existing policy fails
configuration. An explicit missing path also fails. Inline policy takes precedence.
The policy file contains subjects and reviewed capabilities, never credentials.
An optional `resources` list limits a binding to named relying scenarios
(`VROOLI_SCENARIO` or `SCENARIO_NAME`); unrelated scenarios retain their
existing provider assignment. Existing verified provider scopes are preserved.
Native human credentials
retain their issuer-assigned scopes. Web Console source changes remain staged
until the operator permits its restart.


Native human proof forwarded from Swarm Manager is accepted only on the immediate
Agent Manager `POST /api/v1/runs` adapter through a reviewed audience binding.
It does not authorize consent or global management with a Swarm resource token.
The current deployment shares one Cloudflare Access application audience across
these services; deployments with distinct application audiences must qualify a
new explicit binding rather than forward an assertion to an incompatible app.

Missing or invalid proof cannot preserve an earlier principal at a nested
boundary. Retained human decisions require independent human context and reject
agent attribution. Route admission must cover anonymous callers too: dropping
an agent header does not establish an operator. New agent-eligible coarse
capabilities are bounded by reviewed domain operations, not generic write access.

A managed run retains its original owner, requested, profile and frozen scope
ceiling. Missing historical ceiling evidence is a refusal, not a new admission.
An omitted run ID cannot convert a managed or finite process into an unmanaged
attachment. Bookkeeping preserves independent review evidence; ordinary milestone
fields cannot assert verified delivery, and active worker acceptance remains a
separate protected decision.

### Team creator ownership

Prompt Manager records the verified creation actor with the team in the same atomic snapshot. A live verified creator may manage that exact team's configuration, membership, heartbeat configuration and team scheduling controls without a separate operation grant. Other actors use the canonical operation/target evaluator. Creator attribution is server-owned; labels, authored membership and request JSON cannot establish it. Invalid presented agent credentials and verifier failures refuse team mutations before persistence. Legacy uncredentialed maintenance keeps its explicit existing contract; it cannot report verified creator ownership.

The team details response reports whether the verified current caller may manage that team. This is an owner-computed decision from current state, not an input field or a transferable credential. Agent Manager uses the existing authenticated Prompt Manager read to authorize team-backed effort enrollment against the exact declared effort and leader. The enrollment remains owned by Agent Manager and retains revision fencing, idempotency, withdrawal and commission protections. This team-specific path does not confer authority over unrelated efforts or existing run owners. Preparing enrollment does not enable scheduling or start a delivery clock. When one consent covers both team management and delivery management, preparation evaluates the team-owner path first; commission changes retain valid delivery authority on an existing enrollment. Additional approved policies do not remove valid team authority.

A team's creator rule does not grant catalog-wide agent mutation, global scheduler control, or unrestricted run creation. Non-creators use exact `team.manage` coverage under `team-management`; catalog agent creation uses its separate `prompt-agent-creation` policy. Existing paired operation/target rules remain evaluated together. No team route issues human consent. A legacy team without verified creator evidence uses the existing authenticated owner or exact `team.manage` grant path. Ordinary updates cannot invent creation provenance.
