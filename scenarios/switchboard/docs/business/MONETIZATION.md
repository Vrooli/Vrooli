# Monetization — Switchboard

## Purpose Of This Document

How this scenario earns its keep: which capabilities are free, which meter, which
gate, and how each is wired. This document **routes to canon and picks the
integration pattern**; it does not decide strategy.

Whether to monetize, pricing, and bundle membership are operator-curated canon
under `path:docs/monetization/`, which agents never edit. The engineering contract is
`path:docs/concepts/PAID_FEATURES.md`.

## Role In Vrooli

**Role (Axis 1):** interface enabler with a product face. It does not add one
capability — it makes every existing scenario reachable from a phone.

**Interfaces (Axis 2):** conversational in and out, voice in and out, direct UI,
programmatic, and — the one that matters — **embodied/embedded inbound**. That
cell is an em-dash in `ECOSYSTEM.md`'s own interface table, with "future
connector scenarios" named as its enabler. This is that connector scenario.

**Objective (cluster 4):** `T2 — Personal agency`, owned by `director-swarm`.
Live state at the time of writing: `unserved (pending-capability)`, `evidence:
none — cannot be scored`, with an `objective_unmeasurable` warning against it. It
is the only unserved terminal objective in the set, and this scenario would give
it its first evidence source.

## Customer / Buyer

The buyer and the user are the same person: an operator who wants a personal
agent reachable where they already read messages. Three recognisable shapes:

- **The private-by-default operator.** Already refuses hosted agent products
  because their conversations would transit somebody else's servers. This is the
  segment no competitor can serve, and the reason the hosted-relay decision
  matters commercially rather than just architecturally.
- **The person who wants one agent that actually does things.** Has tried
  chat-with-tools products and found the tool list thin. What they are buying is
  the 124-scenario ecosystem behind the message, not the message.
- **The small operator exposing an agent to others** — family, a small team,
  eventually customers. This is the highest-value and the most gated segment,
  because it is blocked on `SWBD-PROB-001`.

## Packaging

Per the `PAID_FEATURES.md` decision tree. **The hard rule this scenario must not
break: never gate a capability a self-hoster could run with their own keys.** The
subscription buys convenience and integrated access, not access to the code, so
BYOK stays valid everywhere below.

| Capability | Mode | Meter class | Reasoning |
|---|---|---|---|
| Creating and managing agents | **free** | — | Descriptor authoring against files on the owner's disk. Gating it would gate the code |
| In-app conversation and the console | **free** | — | The zero-setup trial path. Paywalling the only way to try the product is self-defeating |
| Telegram, and the owner's own Mac for iMessage | **free** | — | A bot token and hardware the owner already has. Vrooli pays nothing and supplies nothing they could not do alone |
| Local speech (`whisper`, `kyutai-stt`, `kokoro`) | **free** | — | The owner's machine does the work. Metering it would be charging for their own CPU |
| Agent inference routed by Vrooli | **metered** | **A — cost-bearing**, `ai_credits` | Real tokens on a Vrooli key. Reserve → execute → finalise through LPBS, server-side, unbypassable. Falls through to an operator-supplied key with no charge |
| Hosted speech fallback, and voice-call minutes | **metered** | **A**, `voice_minutes` | Same reasoning; only when local engines are unavailable, and for telephony minutes at OT-P2-002 |
| Provisioned number with 10DLC handled | **gated** | — | The clean gate: pure convenience over a wall the owner could climb themselves — a real Twilio account, real carrier verification, a week of waiting |
| Hosted iMessage relay for the Mac-less | **gated + metered** | **A** | Vrooli pays a vendor per message. The custody trade-off must be stated at the point of purchase, never in a footnote |
| Agent count, channel count | **gated** | **B — local-capacity**, *no limit key exists yet* | Bypassable by design and that is accepted. A nudge shaping plan choice, never a lock. Record overage as a pricing signal; never retro-bill. **Open gap** — see the manifest section below |

**Never metered:** notifications. `notification-hub` recorded "Never meter
notifications" on 2026-08-17, and keeping that line intact means a delivery
failure can never become a billing event.

## Pricing Hypothesis

Pricing is canon and is not decided here. The structural observations that should
inform whoever does decide it:

- **The renewable object is presence, not usage.** A handle in somebody's message
  list is retained by doing nothing, which argues for a low recurring floor plus
  metered inference rather than usage tiers.
- **Retention in this category is unusually high.** A competitor publicly reports
  roughly 93% of premium users still on their iMessage agent a year later, on a
  free tier they describe as a mistake. Treat as directional, not verified.
- **The gate should sit on setup effort, not capability.** Number provisioning
  and carrier registration are the two things an owner genuinely cannot do
  quickly, which makes them the honest paid convenience.
- **A blocked option the owner can buy past is the clearest upgrade prompt in the
  product**, which is why gated channels stay visible in the catalogue rather
  than being hidden.

## The Manifest To Be Written

`.vrooli/monetization.json` does not exist yet, and should not until the first
metered code path does — `enforcement_paths` naming files that have not been
written would be a false claim, and the `monetization-conformance` phase reads
this file as an assertion about real code.

Recording the intended content here means the wiring step is a transcription
rather than a fresh decision. Schema version 2, per
`path:.vrooli/schemas/monetization.schema.json`, requires `version`,
`bundle_key`, `app_key`, `features`, and `meters`.

```json
{
  "$schema": "../../../.vrooli/schemas/monetization.schema.json",
  "version": 2,
  "bundle_key": "<canon — see below>",
  "app_key": "switchboard",
  "requires_entitlement": true,
  "features": [],
  "meters": [
    { "limit_key": "ai_credits",   "class": "A", "byok": true,
      "enforcement_paths": ["api/internal/turns/inference_vrooli.go",
                            "api/integrations/lpbs/remote_reporter.go"] },
    { "limit_key": "voice_minutes", "class": "A", "byok": true,
      "enforcement_paths": ["api/internal/channels/voice/meter.go"] }
  ]
}
```

`requires_entitlement` is `true` because provisioned handles and the hosted
relay are gated. It must not gate the free tier: entitlement is checked at the
gated capability, never at start-up, or the zero-setup trial path dies with it.

### Two things this manifest cannot yet carry

**`bundle_key` is canon and is not this scenario's to choose.** Every scenario
that has wired monetization uses `business_suite`, but `offer-desk` holds no
`belongs_to` edge for `switchboard` and `path:docs/monetization/` names it
nowhere. The value must come from the catalog strategist through
`offer-desk offers catalog-list` / `catalog-edges`, not from precedent. This is
the single field blocking the manifest once code exists.

**The Class B meter has no limit key.** `packages/monetization-go/meter-inventory.json`
generates exactly three: `ai_credits`, `voice_minutes`, and
`workflow_executions`. None of them counts agents or attached channels, so the
Class B row in the packaging table above is currently a stated intent with no
mechanism behind it. Either a new limit key is added to the inventory, or that
row is dropped and plan differentiation rests on the gated capabilities alone.
Deciding this before implementation is cheaper than discovering it during
wiring, and dropping it is a legitimate outcome — Class B enforcement is a nudge
by design, and revenue integrity comes from Class A regardless.

## Validation Plan

| Question | Experiment | Signal |
|---|---|---|
| Does the zero-setup path convert? | Instrument the `first-agent` journey end to end | Proportion reaching a first real reply without any external account |
| Does presence retain? | Cohort by first external channel attached | 30/90-day active threads per cohort |
| Is metered inference the right meter? | Observe `metered_units_total` distribution per owner | Whether spend correlates with perceived value or with one runaway thread |
| Is provisioning worth gating? | Count owners who stall at the 10DLC step | Stall rate at the one step that takes a week |
| Which channel actually sells? | Attach rate per channel, ordered against declared friction | Whether iMessage justifies the Mac requirement |

Note that none of these can run until a vertical slice exists; this is a plan,
not a result.

## Current Status

**Pre-implementation.** Nothing is wired, nothing is charged, no bundle
membership is claimed.

- No `.vrooli/monetization.json` manifest exists yet, and deliberately so — see
  *The Manifest To Be Written* above, which records its exact intended content,
  the one field canon must supply (`bundle_key`), and the one declared meter
  that has no mechanism behind it yet.
- Bundle membership (business versus lifestyle, headliner versus depth) is a
  portfolio call for `morning-vision-walk` and the catalog strategist, read from
  `offer-desk offers catalog-list` / `catalog-edges`. My read is that this is a
  **headliner wherever it lands**, because it is the first scenario a new user
  would encounter before knowing what Vrooli is — but that is explicitly not a
  decision this document may make.
- The most monetisable segment — exposing an agent to people who are not the
  owner — is blocked on `SWBD-PROB-001` (runtime injection defence has no owner).
  That is a commercial fact, not only a security one.

## Cross-References

- `path:docs/concepts/PAID_FEATURES.md` — the free/metered/gated contract and meter classes
- `path:docs/monetization/` — canon: strategy, catalog, pricing (read-only)
- `path:docs/concepts/ECOSYSTEM.md` — the role and interface taxonomy applied above
- `docs/business/GO-TO-MARKET.md` — positioning and launch motion
- `docs/internal/PROBLEMS.md` — `SWBD-PROB-001`, the commercial blocker
