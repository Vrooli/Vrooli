# Feature and inference route contract

**Status:** coding sessions and declared bounded features use the same funded
LPBS door and private Gateway executor. J4 covers Audio Tools summarization,
contract refusals, caps, prompt ownership and the explicit caller inventory.

## Route ownership

| Caller purpose | Public owner and route | Gateway role | Contract owner |
|---|---|---|---|
| A scenario asks for its declared, bounded AI feature | LPBS door: `POST /api/v1/features/<scenario>.<key>` | Private plan/execute/receipt path; checks the published instruction digest and bounded request shape | LPBS feature declaration and AI Gateway feature contract |
| A coding-agent harness requests model completion with credits | LPBS door: `POST /api/v1/ai/chat/completions` or `/api/v1/ai/responses` | Private funded request path; selects an eligible supplier route | LPBS public compatibility API and AI Gateway provider/rule contract |

The public hostname belongs to the landing-page door. The AI Gateway is a
required dependency behind that door; clients do not receive a second public
Gateway hostname. The private Gateway contract is not the public authentication
boundary. Authentication, reservation, billing, and the usage receipt are
owned by LPBS. The Gateway owns supplier selection and provider execution.

## Feature request boundary

A feature request identifies a declared feature and supplies one instruction,
one turn, and a bounded schema. The scenario API supplies the instruction; the hosted registry holds its
expected digest and output schema without prompt text. The request cannot grant
itself route eligibility, change the instruction, add tools, or request extra
turns. The Gateway validates the received digest and shape; LPBS reserves and
settles usage through its money owner. See
[Paid features](../../../../docs/concepts/PAID_FEATURES.md) and the
[AI provider contract](../../../../docs/resources/ai-provider-contract.md)
for the authoritative policy and billing details.

## Current implementation status

The coding route uses LPBS-issued device/session credentials, account funding
reservations and Gateway's private plan/stream/receipt endpoints. The selected
resource-owned route must carry a `stream_quote` with an input-byte ceiling,
total billable output-token ceiling, upstream cost ceiling in micros and the
supplier's enforceable token-limit field. LPBS pins its account conversion with
the plan; the quote must fit both the request budget and held credits.

Execution enforces that fixed quote before dispatch: one completion, bounded
text/function-tool input, an injected output-token limit and no caller provider
or reasoning overrides. Data collection and downstream host pins come from the
canonical supplier rule. Unknown supplier outcomes retain their receipt/hold;
the same durable operation never dispatches again. A supplier without a valid
quote or verified rule is unavailable. This does not approve any live supplier,
price, rule or trust decision. Bounded features use the same durable execution claim and receipt. Their
output is collected within the declared byte cap and validated against the
published output schema before it is returned. Invalid output is withheld;
known supplier usage still settles and unknown usage remains held.

Harness billing conditions come from the runner's own
`resources/<harness>/model-policy.json` `billing.sources` record, with an exact
verified human decision. Supplier routes cannot supply that permission.
Model `licence.commercial_use` restrictions remain part of both the canonical
approval digest and runtime eligibility. A private Gateway dial/DNS failure
before connection releases the unused wallet/key holds; ambiguous dispatches
remain held for reconciliation.

## Audio Tools feature

`audio-tools.summarize` is declared in Audio Tools' `.vrooli/monetization.json`.
Its API owns the fixed instruction and sends text and level as one user-data
turn. The LPBS request uses an issued Vrooli device key and an optional stable
`Idempotency-Key`; coding session keys cannot invoke application features.
Gateway chooses an eligible model from the declared role/profile, validates the
instruction digest and exact schema, and refuses tools, extra turns, unknown
fields and input overruns. LPBS applies the key and account funding caps and
records `audio-tools.summarize` on usage and Gateway receipts. The response is
JSON with `content`, model, measured tokens and receipt fields. Replay never
redispatches the operation. A failure after the durable dispatch claim returns
an uncertain outcome, retaining wallet and key holds even if saving the final
receipt fails. Receipt reads derive feature identity from the immutable accepted
plan, including coding receipts written before that field was stored.

`config/chat-callers.json` classifies the current production API/CLI callsites
and infrastructure matched by the maintained caller scan. The J4 drift check
fails on new or disappeared matches. Classification describes the source path;
it grants no provider permission and does not convert existing direct/local or
own-account calls into paid features.
