# BAS monetization integration

This guide records the current Browser Automation Studio (BAS) boundaries for
sign-in, subscription features, AI metering, and workflow quota admission.

## Identity and entitlement

The [entitlement middleware](../api/middleware/entitlement.go) adapts the
verified LPBS lease to BAS route gates; caller email and UI flags do not prove
subscription access.

The shared refresh credential is `vrooli/lpbs-account` / `refresh-token`. The
browser keeps only the short-lived access token in session storage; native
credential authority is the durable refresh-token source. BAS APIs resolve the
account identity from the authenticated LPBS session.

## AI source policy

BAS selects one of three AI sources:

- **Local inference:** available without a BAS AI subscription gate and without
  Vrooli-credit metering.
- **Saved provider key:** requires the authenticated session used to resolve the
  saved credential; it is not charged to Vrooli credits and does not require the
  BAS `features.ai` gate.
- **Vrooli credits:** requires the signed `features.ai` entitlement. BAS sends the
  request to AI Gateway, which owns authorization, accounting, and provider
  dispatch. A saved key or caller header does not bypass the selected source
  policy. Confidential remote requests also require the separately verified,
  source-scoped consent receipt.

BAS does not calculate a local fixed cost or report AI usage directly to LPBS.

## Workflow execution quota

Workflow admission is a Class B count-based meter owned by LPBS. BAS reserves
`workflow_executions` before accepting a manual, API, adhoc, or scheduled run.
It persists the execution row before committing the reservation. A failed row
write releases the hold; an uncertain commit fails closed and marks the run
failed. Offline execution is denied because the current-period quota cannot be
reserved.

`monetization-go.QuotaReservationClient` uses the shared HTTPS/loopback-only
LPBS transport and rejects redirects. BAS does not send a duplicate post-run
usage report after reservation commit.

## Remaining local read surfaces

BAS still exposes its legacy current-usage, history, and operation-log read APIs.
The `credit_usage` and `operation_log` tables and their schemas remain in place;
no existing rows are deleted. The local fixed-cost writer has been removed, so
these local reads are not the authoritative current Gateway or LPBS accounting
record. The subscription UI still consumes parts of this legacy read surface;
replacing it with an LPBS-backed view remains open work.

The BAS durable `monetization_usage_outbox` remains for the governed desktop
monetization journey probe. It is not the production AI usage path. Generic
outbox support remains in `monetization-go` for actual consumers.

## Verification

Focused local proof for the remaining outbox adapter:

```sh
cd scenarios/browser-automation-studio/api
go test ./services/credits -run TestUsageOutboxSurvivesServiceRestart -count=1
```

The test queues an immutable event, constructs a fresh BAS service instance,
and verifies that the event is delivered once through the shared LPBS transport.
It uses a local fake authority and does not claim a customer billing event or
live LPBS proof. Gateway and LPBS owner tests cover their respective metering
contracts.

## Troubleshooting

- **Workflow admission fails:** check that LPBS is configured and that the
  authenticated session can reserve `workflow_executions`. BAS does not fall back
  to local admission when LPBS is unavailable.
- **Vrooli-credit AI is denied:** verify the signed BAS `features.ai` lease, the
  authenticated consumer session, Gateway availability, and any required
  source-scoped consent receipt.
- **Usage shown in the BAS legacy panel disagrees with LPBS:** the local panel is
  not the authoritative current usage record. Use the hosted LPBS account surface
  for account management while the BAS read surface is migrated.
