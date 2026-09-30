# Monetization — Audio Tools

## Purpose Of This Document

Define Audio Tools' commercial role and the evidence needed before its owned
subscription/credits route can be sold as working capability. This document does
not establish pricing, approve paid tests, or report hosted-service readiness.

## Role In Vrooli

Audio Tools is a shared capability whose hosted voice features can add value to
Vrooli subscriptions and credits in consuming apps. Local compute and BYOK remain
explicit alternatives. Their availability does not authorize billing those modes.

Keep entitlement, wallet, inference metering, and settlement rules with their
shared owners. Consumer UIs present the selected route and its consequences;
they must not implement private pricing or settlement ledgers.

## Customer / Buyer

The intended user needs dependable dictation across devices, with an explicit
choice of local processing, their own provider account, or an owned hosted service.
The hosted buyer may lack suitable local hardware or prefer managed capacity.
These are product hypotheses; willingness-to-pay evidence is not established here.

## Packaging

| Option | Decision state |
| --- | --- |
| Subscription allowance / purchased credits for hosted features | Intended commercial route; pricing, metering, and qualification remain prerequisites. |
| Shared capability included in consuming applications | Intended reuse model; each consumer needs product-path evidence. |
| Standalone application or separate add-on SKU | Candidate, not an approved catalog entry or launch commitment. |

Use the [project monetization canon](../../../../docs/monetization/README.md)
and its governed catalog for packaging decisions. This file is not a second catalog.

## Pricing Hypothesis

Charge for the owned hosted service through shared subscription/credits policy,
not a new per-consumer mechanism. Do not infer an automatic switch into a paid
route merely because local inference or BYOK fails.

The commercial owner must define billable units and settlement for each operation:
audio input duration, generated speech units, and summarization usage need not share
one meter. Cost accounting, retail credits, and provider invoices are different
quantities. Specify reservation, partial service, cancellation, retries, and
concurrent usage before asserting the expected charge.

The proposal below makes settlement decisions reviewable. The existing
[pilot decision sheet](../internal/TESTING.md#pilot-decision-sheet-and-safe-first-slice)
links to this policy instead of maintaining another version. No numeric price,
credit conversion, or paid-test allowance is granted here.

## Proposed voice billing policy v1

Status: proposed for product and shared-monetization-owner review, not deployed
behavior. This policy covers owned hosted STT first. TTS and summarization require
their own reviewed meters before sale; copying the STT duration meter is invalid.

| Decision | Proposed rule | Required proof before adoption / launch |
| --- | --- | --- |
| Route consent | Local and BYOK produce zero Vrooli voice-credit debit. BYOK can still incur the user's provider bill. Owned service requires an explicit selected route or previously approved user fallback policy. | Local/BYOK failures, revoked keys and insufficient balance cannot silently switch to owned service; UI states actual processing destination |
| Price identity | Resolve a versioned shared catalog meter and quote before admission. Preserve that version for the session. Reject an unavailable quote rather than assuming free service. | Catalog version, meter, account, operation and session attribution reconcile; no retail conversion is hardcoded in consumers |
| STT service unit | Meter unique audio duration acknowledged as processed under the hosted contract, using sample counts and sample rate. Silence is still processed input. Do not bill wall-clock connection time, duplicate/replayed intervals or local retained-but-unsent audio. | Known-duration, silence, resampling and duplicate/reconnect fixtures produce the same unique interval count |
| Admission | Shared owner atomically authorizes entitlement and reserves a bounded next service window before processing it. Subscription allowance is used before eligible purchased credits; overage requires the user's credit-spend policy. | Two concurrent sessions cannot reserve the same remaining balance; inactive entitlement and zero balance return explicit refusal |
| Exhaustion / outage | Stop admitting new billable windows when authorization is exhausted or unavailable. Finalize already authorized work and expose unsent audio as retained/recoverable according to the capture contract. | Mid-stream exhaustion, authorization outage and delayed settlement do not create free unbounded processing or silently discard captured input |
| Partial service / cancel | Settle only delivered, uniquely processed intervals up to acknowledged cancellation; release unused reservations. No minimum-session fee is proposed. Billable delivery acknowledgement and the cancellation boundary must be durable. | Cancel before delivery yields zero debit; cancel after partial delivery charges that subset only; retained output can be recovered without another debit |
| Retry / crash | Use stable operation, session and interval identities and an idempotent shared settlement key. Provider retries and browser replays do not create another customer charge. Reconciliation releases or settles stranded reservations from durable evidence. | Duplicate delivery, response loss after debit, process restart, replay and concurrent retry each preserve one ledger effect |
| Service failure | With no durable delivery acknowledgement, release the affected reservation. For partial acknowledged delivery, settle only the proven subset. Provider cost and customer charge remain separate accounts. | An upstream invoice or HTTP success alone cannot trigger customer settlement |
| Reconciliation and privacy | LPBS/shared metering owns ledger truth. Audio Tools owns interval/delivery evidence and passes metadata-only attribution. Expose usage/charge status without storing transcripts or credentials in billing logs. | Session evidence and ledger receipts reconcile; missing receipt is pending/unknown rather than assumed paid or free |

Before adoption, the shared owners must resolve delivery acknowledgement across
client disconnect/recovery, reservation-window size, retention/reconciliation
deadline, rounding/credit conversion, allowance renewal boundaries, and refund
operations. These affect money and cannot be inferred from available fixture
endpoints. If an existing shared policy differs, choose and record one policy;
do not install a competing Audio Tools implementation.

### Simulation contract

Use LPBS-owned routed fixture identity for account, subscription and wallet state.
Use an independent provider fake for successful partials, delayed finals and
failure. Propagate the same test identity and session/meter correlation through
the real shared monetization adapter. Fixture activation must be explicit and
isolated from production credentials and balances. A process named `test` is not
authorization to seed or use a production account.

For each policy row, assert both the user-visible outcome and the shared ledger
delta, reservation remainder, uniqueness of settlement, and delivery evidence.
Include active/expired subscription, allowance-only, credits-only, exhausted
allowance with permitted/forbidden overage, simultaneous admission, canceled
delivery, retry after debit, and unavailable authorization. Record the policy
revision and fake/live boundary in receipts. An entitlement-only test does not
qualify settlement; a settlement simulation does not qualify hosted inference.

Initial test spending is zero paid calls and zero purchases. A later live grant
must name providers, accounts, data scope, aggregate cost cap across retries and
child runs, and stop conditions. Seeded balances are not a live spending grant.

## Validation Plan

Reuse the [pilot evidence matrix](../internal/TESTING.md#development-pilot-evidence-contract)
instead of another commercial test ledger. LPBS-owned routed fixtures establish
subscription/wallet test state; a separate inference fake establishes provider
behavior. Neither establishes live hosted delivery or production settlement.

Before making a paid-capability claim, retain applicable evidence for the selected
route, entitlement, inference delivery, usage attribution, and reviewed settlement
policy. Demonstrate insufficient balance, cancellation, duplicate/retried delivery,
and concurrent usage. Qualify the consumer-facing voice path as well as the service
boundary. Live provider calls need an explicit spending grant.

## Current Status

Source inspected on 2026-09-08: the three provider chains have local/BYOK wiring,
but hosted delivery is not merely a completed feature behind a disabled flag.
`BuildChains` supplies no Vrooli provider; the LPBS STT, TTS, and summarization
clients return “gateway endpoint not implemented,” and shared availability is false.
A user without local capability or usable BYOK therefore cannot rely on an owned
hosted fallback today.

The [integration inventory](../concepts/INTEGRATIONS.md#scenario-dependencies)
owns those source references. The [problem register](../internal/PROBLEMS.md)
retains investigation evidence. Resolve shared AI Gateway/LPBS routing and metering
ownership during implementation; do not infer a finished integration from names,
a feature flag, or a test fixture.

The `portable-voice-v1` PRD includes this commercial scope in OT-P0-007, explicit
route consent in OT-P0-004, and isolated simulation in OT-P0-010. Requirements
ATD-P0-012, ATD-P0-013 and ATD-P0-017 make the missing delivery, settlement and
fixture obligations visible as planned work. Publishing those obligations does
not adopt the proposed meter, prices, cancellation boundary or spend allowance.
Hosted inference integration is not, by itself, authorization to build the
multi-tenant GPU platform excluded by the current PRD.

## Cross-References

- [PRD](../../PRD.md) — current operational targets
- [Architecture](../concepts/ARCHITECTURE.md#development-target-portable-streaming-voice) — intended portable voice boundaries
- [Integrations](../concepts/INTEGRATIONS.md) — current dependency and provider wiring
- [Testing](../internal/TESTING.md) — fixture and qualification contract
- [Go-to-market](GO-TO-MARKET.md) — channel hypotheses
- [Project monetization](../../../../docs/monetization/README.md) — commercial authority
