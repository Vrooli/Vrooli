# Inactive source policy candidate

7 November 2026. This is an implemented, offline-tested policy core, not deployed source authorization. No source gate is registered in a server or installed in an issuer. Program Runtime's bridge remains disabled. No live credentials, grant records, source requests, subscription, service enablement, security-setting changes or deployment were performed.

## What now exists

`api/internal/localworkbridge/source_policy.go` admits a finite request only after both canonical credential verification and a fresh trusted owner-binding lookup. Default-disabled construction and admission perform no verification, body reads, authority reads or process work.

- Events: only GET `/api/v1/events` with one selected `event_id` and `limit=2`. The configured selection is at most 64 distinct exact IDs. Extra filters, duplicate query values, ranges, other endpoints, request bodies and mutations are rejected. The owner must validate the canonical returned ID, source `agent-manager`, and one of the three selected event types before releasing bytes.
- Collector: only `RunDeclaredProgram`, `GetDeclaredExecution` and `WaitForProgram`. Admission pins `program-runtime.local-work-snapshot`, its digest and exact canonical inputs, provenance `PROVENANCE_AGENT`, asynchronous operation, a bounded local-work request key, and a deadline no more than ten minutes ahead or beyond credential/binding expiry. Observation pins the declaration/key. Wait is limited to 90 seconds and requires an owner-retained record for that principal's program before releasing its result.
- Unknown fields, caller attribution, grants, arbitrary program source, proto field aliases, nested duplicate JSON keys, excessive depth/size, alternate auth carriers, proxy/method overrides and test-routing headers are rejected. A validated call is copied; the original request must not be forwarded to an owner as authority.
- Subject, owner, source instance `swarminator`, commissioned epoch, binding identity/version and immutable policy digest are pinned. The binding must be current, unrevoked and at most seven days long. Unavailable authority fails closed. Token scopes must contain only the dedicated purpose, with no broad scenario scope, roles or token-borne grant.
- Operation/target checks use canonical `identity.OperationGrant.Allows`. Approved wildcard coverage remains valid, and operations cannot cross-match targets from another policy rule. The core does not invent an approval or a new grant store.
- `SourcePermit.Recheck` rereads live policy. Owners must call it within the actual admission serialization/transaction immediately before committing an effect, and before releasing read/wait results. HTTP admission alone is not final authorization.

The live binding lookup is an interface, not an implemented storage/issuer adapter. The durable program ownership, canonical finite adapters and native selection lifecycle are now implemented as inactive components; see OWNERSHIP.md. Live authority and confined integration still fail the production gate.

## Canonical verifier review repair

The existing canonical JWT verifier intentionally accepts configured migration audiences and reports its configured primary audience as `Principal.Audience`. Therefore checking that field cannot prove a credential has an exclusive audience.

`packages/api-core/authn/jwt.go` now has an opt-in `ExclusiveAudience` mode and an inspectable `UsesExclusiveAudience` construction contract. It accepts only the exact configured audience as a scalar or one-element audience array, rejects migration configurations, and rejects duplicate signed claims and case aliases of security claims, including Unicode case-folding aliases. Migration audience slices are defensively copied. Existing verifier defaults and runtime configuration are unchanged; there is no environment activation switch.

`NewSourceGate` requires the concrete canonical verifier configured for that exclusive source-purpose audience. Caller-supplied principal metadata or an arbitrary verifier interface cannot substitute. Tests use signed in-memory fixtures through an injected HTTP transport without a listener. Public fixture claims and test signing keys are not installed or retained as authority.

## Why source capabilities are still not production-ready

1. **Anonymous or bypass routes remain reachable.** Events' `api/main.go` mounts generic event, policy and subscription routes without shared authentication configuration; its handlers do not apply this policy. Program Runtime's shared authentication is passive, and the generic library/program handlers do not consume this permit. Rejecting a bridge bearer on unrelated paths would not suffice if the process can omit Authorization and call a generic anonymous path. A dedicated source token is not least privilege while those paths remain reachable to that process.
2. **Finite service ownership is implemented but not commissioned.** The existing durable program row now retains service ownership atomically, and canonical finite adapters enforce owned replay/read/wait. The live shared authority and lifecycle boundaries remain uninstalled. Caller-provided attribution and generic anonymous routes do not inherit this authority.
3. **No live binding adapter is installed.** The canonical authority must supply current subject-bound grant, owner, version, expiry and revocation under the approved action. Policy labels, memfd metadata and a broad user's token are not substitutes.
4. **Native selection has an inactive durable owner.** Authenticated native receipts, exact gate bindings, stable request keys/deadlines and collector-terminal tombstones now survive restart and redelivery. Actual commissioned key provenance, joined authority and inaccessible storage remain prerequisites. Later selections require their own approved gate scope; updating selection cannot silently expand parent approval.
5. **Execution and final-effect wiring remain absent.** The policy core is not an HTTP handler and is not registered anywhere. It does not enforce transaction-bound rechecks by itself, limit an owner's raw network reachability, create a dedicated namespace, or prepare the immutable worker. HTTP body deadlines and cancellation-cooperative owner implementations are still required.

Choose and implement one owned enforcement route before minting any source capability:

- Apply mandatory source authorization at every relevant owner ingress, including anonymous/missing-credential handling, with preserved approved user/CLI behavior and durable service admission ownership. Qualify the actual owner handlers and storage transactions.
- Or use an existing trusted broker with a confined worker unable to reach generic owner APIs. Expose only the finite event/collector operations over the reviewed private channel, and retain authority/revocation and admission ownership at the broker/owner boundary. Broker network confinement and credential placement require their normal action-time approval before provisioning.

Do not implement a catch-all proxy, broad bearer wrapper, anonymous localhost fallback, fabricated agent provenance, or generic namespace exceptions to make this candidate appear ready.

## Selected confinement continuation

The least-disruptive continuation selects a separate trusted finite broker and a network-denied worker. An unregistered default-disabled broker core now reconstructs four source operations, keeps source credentials outside worker requests, pins one trusted selection/request key/deadline, validates bounded results and carries the session fence into owner final rechecks. It does not close anonymous-route reachability by itself. The original Python network/credential path remains unconnected and unqualified. See [the broker decision, precise implemented scope and remaining owners](BROKER-CONFINEMENT.md).

## Verification and limits

Evidence: `.vrooli/evidence/local-work-bridge-20261106/source-policy-20261107/`.

Focused source tests cover method/path/parameter denial, exact declaration/digest/inputs, unknown fields and JSON duplicates, principal kind/issuer/audience/scope, broad scopes/roles/token grants, expiry during live lookup, revocation after admission and during result release, owner and retained-key mismatch, operation-target rule pairing, immutable copies, and disabled no-I/O.

Signed canonical JWT tests cover scalar/singleton acceptance; alternate, multiple and repeated audience rejection; invalid migration configuration; duplicate claim keys; ASCII and literal/escaped Unicode security-claim aliases; and the real JWT verifier through source admission.

Bridge and canonical-authentication package race tests pass against the final candidate (47 and 74 top-level tests respectively), and the three API-main race tests pass. Bridge vet passes. Canonical-authentication vet fails on an unchanged existing value-receiver test helper: `authn/operation_pair_test.go:24:7`, `Verify passes lock by value` because `pairedVerifier` embeds `atomic.Int32`. This warning is retained as an unrelated pre-existing validation limit; the task does not claim a full vet pass. Tests have no production source/cloud calls or installed credentials. Existing canonical auth tests may create temporary loopback fixture servers; these are not service activation. No scenario-wide or live-owner qualification is claimed.

The read-only protobuf freshness verifier still reports 23 unrelated generated-artifact differences, including the descriptor image and template-validation-python-uv-react-vite outputs. These were preserved; no generated files were edited or regenerated. Focused package checks do not turn that failed repository prerequisite into a pass.

## Remaining commissioning gate

First complete live owner/broker composition, authority and recovery adapters, immutable launcher/image/storage and joined runtime qualification. The implemented durable service ownership and native selection components do not replace those gates. Then obtain action-time approval for real source capability creation/placement, owner access, selected source/types, fixed collector digest/inputs, seven-day-or-shorter expiry, dedicated security boundary, cloud destination/access, subscription, service enablement and named transmissions, as specified in `PROVIDER-BINDING.md`.

After authorized commissioning, prove source → durable native queue → cloud receipt → journal → handler pickup → source-backed many-to-many reconciliation → readback. Require two later separated authorized changes without manual pickup, retaining pause/mute, permission and checkpoint holds, and legitimate local-only/cloud-only work. Passing this policy suite does not prove that chain.
