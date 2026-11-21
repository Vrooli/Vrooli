# Authorization — the KEEP-SYNC boundary

> **What this is.** The rule for which human-approval gates **stay synchronous-human-live** and must
> **never** be converted to the asynchronous operator-decision rail
> (`**.decision_requested.v1` → notification-hub → `ask.resolved.v1`), even though that rail is now the
> shared substrate for most approvals (see [`AUTHORIZATION-CURRENT-STATE.md`](./AUTHORIZATION-CURRENT-STATE.md)).
> It exists so a future "just raise a decision instead" generalization cannot quietly dissolve the base
> of trust. The api-core half of this boundary is **test-pinned** — see
> `packages/api-core/authn/keep_sync_boundary_test.go`.

---

## The rule

A gate may move onto the async operator-decision rail **only if** answering it late, out-of-band, and
non-interactively is safe. A gate must **stay synchronous-human-live** iff it does either:

- **(a) mints or revokes the base of trust** — a machine identity (service principal), a standing grant,
  or the live-human approval *itself*; or
- **(b) releases raw secret material, or opens an unbounded interactive session** (a shell, an SSH
  session, an interactive terminal) — authority that, once handed over, is no longer bounded by the
  gate.

Everything else — a release *decision*, a run review, an effort/goal question, a mutation-proposal
accept — is a **bounded, revocable, re-validatable** choice and legitimately rides the rail.

## The bootstrap argument (why (a) is circular on the rail)

The operator-decision rail is *attested*: a resolution (`ask.resolved.v1`) is trusted because
notification-hub verified a live human (Cloudflare Access) answered it, and the webhook is HMAC-signed.
That trust chain **depends on** the very primitives in clause (a):

- A **service principal** is what lets a scenario call notification-hub and verify the signed webhook.
- A **standing grant** / **authorization** is what the rail's own "ask-don't-block" approval *mints into*.
- The **live-human proof** (Cf-Access) is the root the entire resolution is attested against.

So routing the *issuance* of one of these through the rail that the issued credential would authorize is
**circular**: you would be asking a rail to authorize the creation of the thing that makes the rail
trustworthy. The gate that mints the base of trust therefore cannot itself be a rail consumer — it must
be a synchronous, live, cryptographically-verified human (or an operator-local trust root such as kernel
`SO_PEERCRED`). Clause (b) is the same principle for raw capability: once secret bytes are returned or an
interactive shell is open, no later "resolution" can un-release them, so the release must be gated live.

---

## The protected gates (MUST stay synchronous-human-live)

| Gate | Where | Clause | What it refuses |
|---|---|---|---|
| **RequireHuman** | `packages/api-core/authn/authn.go` (`RequireHuman`) | (a) approval root | any non-human (agent/service) principal |
| **RequireIndependentHuman** | `packages/api-core/authn/operation.go` (`RequireIndependentHuman`) | (a) approval root | a human request carrying agent provenance (an agent cannot borrow a co-presented human approval) |
| **Standing-grant authorize / mint / revoke** | decision: `packages/api-core/authn/standing_grant.go` (`AuthorizeStandingGrant`); mint/revoke: agent-manager `internal/orchestration` standing-grant authority + `RaiseAuthorizationApproval`'s human-only issuance | (a) mints a grant | an empty/uncovering grant authorizes nothing; a grant is *minted* only under a verified approver (the ask-don't-block approval attributes issuance to the human, never the requesting agent) |
| **Authorization grant mint (interactive)** | agent-manager `internal/orchestration/authorization_decision_events.go` (interactive type is **push-only**, no resolver) + `authorization.go` `ApproveAuthorization` | (a) mints authority | minting requires a live independently-verified human **and** a matching live process binding — properties a hub resolution cannot carry; the interactive decision event is a push, never a rail-minting path |
| **Service-principal issuance** | scenario-authenticator `api/internal/accounts/service_principal.go` (`ExchangeServiceCredential`) | (a) mints a machine identity | issues the credential that *lets* a caller use the rail at all; gated at the authenticator, not the rail |
| **Cloudflare Access assertion verification** | `packages/api-core/cloudflareaccess` verifier; notification-hub `internal/identity/access.go` | (a) the live-human proof | a service token / empty-subject assertion (human-login only); this is the proof the rail is attested against |
| **Raw-secret / release gate** | **not a standalone function** — it is the Mechanism-2 synchronous human check: deployment-manager / scenario-to-cloud `RequireIndependentHuman` on the release/secret operation | (b) raw capability | a non-independent-human caller for a release or secret op (an agent may only *execute* an already-approved one) |

> The api-core rows (`RequireHuman`, `RequireIndependentHuman`, `AuthorizeStandingGrant`, the Cf-Access
> verifier) are pinned by `packages/api-core/authn/keep_sync_boundary_test.go`, which asserts each human-only
> gate **refuses** an agent and a service caller and that the authn package **imports no operatordecision /
> event-rail publisher** (so none can route a decision onto the rail). The scenario-local rows
> (service-principal issuance, grant mint/revoke, the Mechanism-2 release check) are pinned by their own
> scenario suites; they cannot be imported into the api-core test without a dependency inversion, so the doc
> enumerates them and the api-core test pins the load-bearing shared half.

### Doctrine — protected by principle, no dedicated coded gate yet

These belong on the synchronous-human-live side of the boundary by the rule above, but there is **no
dedicated coded gate for them today**. They are stated here so a future implementer keeps them synchronous
rather than reaching for the decision rail:

| Boundary | Status today | Real enforcement / trust root |
|---|---|---|
| **SSH / interactive-terminal open** | no distinct "open-session" approval gate in code | the interactive-session open path (vrooli-bridge / web-console) is operator-driven; a future *agent-initiated* open MUST be gated live, never asked-and-resumed — opening a shell is unbounded authority once granted |
| **Break-glass / emergency access** | no distinct break-glass gate exists | emergency access is the operator-local trust root (CLI `--local-owner`, kernel `SO_PEERCRED`), already synchronous-human-live by construction; any future named break-glass path MUST stay synchronous-human and off the rail |

---

## GCT's git-write model is intentionally NOT on the rail

Git Control Tower keeps its **propose-then-apply** model, which is deliberately *not* the operator-decision
rail and *not* a synchronous human gate either:

- An agent may **create/edit** a commit proposal (`scenarios/git-control-tower/api/internal/proposals`),
  but **cannot apply** it — a direct git write fails closed.
- The **operator applies** the proposal at their own pace (`internal/proposals/apply.go`); commits are
  **never a gate** on any acceptance, admission, or resume.

This is a third shape on purpose. It is not converted to the decision rail because the artifact *is* the
proposal (a durable git object the operator reviews directly), and because a commit must never become a
blocking gate — forcing it onto an "approve/reject ask" would both misrepresent the review (the operator
reads a diff, not a yes/no) and risk coupling progress to a commit, which the operating model forbids.

## What intentionally DID move to the rail (for contrast)

These are bounded/revocable decisions and now share the `operatordecision` substrate + per-scenario
domain effects (see [`AUTHORIZATION-CURRENT-STATE.md`](./AUTHORIZATION-CURRENT-STATE.md)):

- agent-manager: run review (needs_review), effort/goal decisions, and the *ask-don't-block* run-create
  approval (which **mints a standing grant** only under a verified human — the mint stays human-only even
  though the *ask* rides the rail).
- deployment-manager: the release **decision** (artifact-bound, re-validated before any approval records).
- swarm-manager: mutation-list proposal **accept/revise**.

The distinction is exact: the **decision** can be asked and resumed; the **issuance of the base of trust
that a yes implies** is performed synchronously, under a verified human, at the moment of resolution —
never by the rail itself.

---

*Authored on branch `agi`, 2026-10-16, as part of the operator-decision consolidation capstone. The
api-core boundary is pinned by `packages/api-core/authn/keep_sync_boundary_test.go`; line references in
the table are pointers, not contracts.*
