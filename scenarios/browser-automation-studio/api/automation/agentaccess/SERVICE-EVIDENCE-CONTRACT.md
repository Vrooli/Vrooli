# BAS → Agent Manager browser evidence: unconfigured contract

Status: BAS callback/proof-reference path and the AM proof-verifier client/handler composition are implemented in source and focused-tested. The AM verifier principal is declared on the AM side; no credential was minted/applied, no Agent Manager operation grant was created, no service was restarted, and no live callback or trusted evidence ingestion has been exercised. The existing `POST /api/v1/browser/record-evidence` accepts run-token reports and labels them `reported_by=run-agent`; it remains untrusted and cannot qualify sessions, usage, or billing.

## BAS service principal request

BAS declares only the service principal it needs in [`.vrooli/service-principals.json`](../../../.vrooli/service-principals.json): subject `browser-automation-studio-agent-manager`, Agent Manager audience, and coarse `agent-manager:write` scope. This declaration does not grant authority. The exact operation grant below is a separate required control.

Safe preview command (default dry run):

```sh
scenario-authenticator service-principals sync \
  --only browser-automation-studio-agent-manager \
  --root /home/matthalloran8/Vrooli
```

The preview reports the principal, `agent-manager:write` scope and credential identity that a human could apply later. Do not add `--apply` here. BAS credential reading must use the existing credential authority and `api-core/servicetoken` pattern; a missing credential leaves the recorder disabled. BAS does not mint, print, cache or log credentials.

## Agent Manager operation gate

A future owner-authenticated evidence route must require all of the following before append:

1. Verified service principal subject `browser-automation-studio-agent-manager`, exact Agent Manager audience, and `agent-manager:write` capability.
2. Active signed Agent Manager run token and opaque browser grant receipt belonging to the same persisted run and browser capability.
3. Exact operation `browser-evidence.record` and exact target `agent-manager:run/<uuid-derived-from-the-verified-token>`. Do not accept run identity or target from the request body. Evaluate through `authn.RequireOperation` and Agent Manager's existing operation authority; do not add a grant store.
4. A separately human-approved one-run operation grant. Agent Manager now exposes `POST /api/v1/browser/evidence/authorize/{run_id}` as a no-body authorization request/check. It verifies the exact BAS service principal, `agent-manager:write`, the active signed run token and browser grant receipt, then evaluates `browser-evidence.record` for exact target `agent-manager:run/<run_id>`. With no grant it can raise the existing human decision; the request records no evidence and returns approval pending. The operation grant is one-time and has a maximum one-hour lifetime; it expires by time and can be revoked through Agent Manager's existing revoke route. Generic `/api/v1/standing-grants` remains fixed to `run.create`; it is not widened for BAS.
5. BAS must call this route only from the owner recorder after resolving the exact live Broker entry and immutable lease. The event's session ID, mode/profile, action outcome and closed-session time/action/byte measurements must be derived by BAS from that entry, not agent JSON or Agent Manager's run-reported transcript. URLs, page text, typed values, DOM, cookies, credentials and raw lease/node IDs stay out of the evidence payload.

The service principal proves which scenario requested authority; the authorization route does not record events and does not prove that a supplied session was live. The AM owner has now implemented `POST /api/v1/browser/bas-evidence/{run_id}` as a separate trusted ingest path: it enforces the exact operation grant at append time, then synchronously calls the BAS proof verifier before persisting a broker-owned closed-session summary. AM segregates transition state by `reported_by` so its legacy run-agent entries cannot block or inflate broker summaries.

BAS exposes the callback at `POST /api/v1/browser/internal/evidence-proof/verify` and `/consume`. Both require an exclusive scenario-authenticator service token for subject `agent-manager-browser-evidence-verifier`, audience `scenario-authenticator:browser-automation-studio`, scope `browser-automation-studio:browser-evidence.verify`; nil verifier is HTTP 503 and wrong service subject is denied. Verify accepts only `{run_id,grant_receipt,proof_reference}` and returns `{observation:{kind,session_id,state,session_minutes,actions,bytes}}`. Consume accepts `{run_id,proof_reference}` and returns `{consumed:true}`. Caller-supplied session IDs/counters are rejected. The callback resolves an opaque random 32-byte proof reference only from the existing Broker entry. On close, BAS binds it to the run principal, opaque browser grant, session, immutable lease, owner, profile, backend, closed timestamp and digest of the immutable summary; it expires after five minutes. The finalized entry is kept in the existing entries map for that TTL; normal operations reject closed entries. Verify re-resolves the finalized entry and lease binding; consume is idempotent for the same run/ref to tolerate an AM response loss. No proof key or separate proof store was added.

BAS derives elapsed minutes from its own open/close timestamps and action counts from its action-evidence boundary. Its `bytes` counter measures serialized input/output payload sizes on BAS Broker action, observe, accessibility-tree and forwarded-control-input operations; it does not measure browser network traffic and omits unrelated metadata and extension operations. These counters are not a full traffic/compute bill and must not be treated as such. BAS submits only the proof reference to the AM BAS-ingest route, alongside the trusted run token/grant and its existing operator-issued service token. That service token is lazily resolved through the existing credential authority and `api-core/servicetoken`; missing credentials or an unapproved/expired operation grant prevent ingest. Missing BAS service credentials do not fail a completed browser close; the broker proof remains short-lived and unappended. The AM→BAS verifier principal is declared, but its service-principal configuration and credentials have not been applied. No live session or trusted evidence claim is made until both service-auth directions and a human-approved exact-run grant are available and the full callback journey passes.

## AM-to-BAS proof verification

Agent Manager declares verifier principal `agent-manager-browser-evidence-verifier`, resource `browser-automation-studio`, audience `scenario-authenticator:browser-automation-studio`, and scope `browser-automation-studio:browser-evidence.verify`. Its lazy token client and callback composition are implemented. This is a declaration only; it does not mint credentials or authorize a verifier to claim a session is live.

BAS verifies the exact role through api-core's scenario-authenticator middleware. AM's callback client calls the exact mounted verify/consume paths above and sends no run-token or browser-grant header to BAS. Both service credentials remain operator-owned and unapplied; missing credentials keep the round trip fail-closed. A service principal does not replace the live-entry proof: BAS still resolves the opaque reference against a retained Broker entry and AM still enforces its exact operation grant at ingest. Legacy run-agent reports remain separate and untrusted.
