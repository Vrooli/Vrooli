# Paid Features — Integration Contract

> **What this is:** the engineering contract for *how* any scenario turns a capability into a paid feature — the metered-vs-gated decision and how to wire each through `landing-page-business-suite` (LPBS). Agent-maintainable engineering reference, sibling to [`ECOSYSTEM.md`](./ECOSYSTEM.md).
>
> **What this is NOT:** the *whether*, the *price*, or the *bundle*. Those are operator-curated **monetization canon** under [`docs/monetization/`](../monetization/README.md) — which agents never edit directly (canon changes flow through operator-approved decisions). This doc *cites* that canon; it does not restate or override it.

## Why this exists

AI credits are documented cleanly (LPBS AI Gateway). Audio metering works but is scattered. Entitlement gating is documented per-scenario (LPBS, BAS). The result: every scenario rediscovers LPBS or invents its own approach. This doc is the single contract so a scenario builder asks the right question once — **metered or gated?** — and wires the known pattern instead of improvising.

It is the engineering half of the `ecosystem-fit` lens's "Monetization & bundle fit" cluster (`prompt-manager skill read ecosystem-fit`). The lens routes you here; this doc tells you how to build it.

## The one decision: free, metered, or gated?

```
Does this capability have a real marginal cost per use
(LLM tokens, STT/TTS seconds, compute, a paid third-party API)?
│
├─ YES → does it ALSO need to be restricted to certain plans?
│        ├─ YES → METERED + GATED (check entitlement, then charge credits)
│        └─ NO  → METERED (charge credits per use)
│
└─ NO → is it a plan/tier differentiator (premium feature, gated download, studio-only)?
         ├─ YES → GATED (entitlement flag / plan-tier check)
         └─ NO  → FREE (no integration — most features)
```

| Mode | Use when | Mechanism | Canonical reference |
|---|---|---|---|
| **Free** | No marginal cost, not a tier differentiator. The default. | Nothing to wire. | — |
| **Metered** | Each use costs real money (tokens, audio seconds, compute) | Charge credits via LPBS credit API (reserve → execute → finalize) | [`AI_GATEWAY.md`](../../scenarios/landing-page-business-suite/docs/reference/AI_GATEWAY.md) |
| **Gated** | Capability is a plan/tier differentiator with negligible marginal cost | Check entitlement (`status`, `plan_tier`/`plan_rank`, `features[]`) | LPBS `GET /api/v1/entitlements` |
| **Metered + gated** | Expensive *and* tier-restricted | Entitlement check first, then credit charge | both of the above |

> **Do not paywall core/free capability.** Per monetization `STRATEGY.md` principle 1, the subscription buys convenience and integrated API routing — not access to open-source code. If you find yourself gating something a self-hoster could already run with their own keys, the framing is wrong. BYOK (bring-your-own-key) must remain a valid path: a metered feature should fall back to the user's own provider key with no credit charge.

## Metered contract (credits)

Charge through LPBS so every metered feature shares one wallet, one auth, and one set of safety guarantees. The pattern (see `AI_GATEWAY.md` for the reference implementation):

1. **Authenticate** — the user's LPBS JWT (`Authorization: Bearer …`).
2. **Reserve** — atomically check balance and reserve estimated cost (row-level lock / `SELECT FOR UPDATE`) to avoid TOCTOU overspend. Estimates carry a safety margin (1.5× in the AI gateway).
3. **Execute** — run the operation. For streaming, hold the reservation (auto-expires ~10 min) and stream.
4. **Finalize** — settle to actual usage: refund the difference if under, charge extra if over.
5. **Handle `insufficient_credits` (HTTP 402)** — short-circuit gracefully with a clear upgrade/top-up path; never partially deliver and silently fail.
6. **Record** — write a usage-ledger row (provider, quantity, credits) so the user sees spend and we can reconcile.

Credits are stored as internal units (`credits_per_usd × USD`) and rendered with `display_credits_multiplier` / `display_credits_label` — do not hardcode display numbers.

**Reference implementations to copy from, not reinvent:**
- **AI / LLM tokens** — `AI_GATEWAY.md` (`POST /api/v1/ai/chat|stream`, `VrooliProvider`, `ErrInsufficientCredits`). The fully-canonical example.
- **Audio per-second / per-ms** — `scenarios/audio-tools/docs/domains/usage/` (usage ledger of provider/ms/credits) + the TTS/summarize chains' insufficient-credits short-circuit. Same credit model, metered by duration instead of tokens.

## Gated contract (entitlements)

For tier-differentiated features with no real marginal cost. Call LPBS and gate on the result:

- **`GET /api/v1/entitlements`** → `status` (`active`/`trialing`/…), `plan_tier` + `plan_rank` (solo → pro → studio → business), `features[]` flags (from plan metadata), `credits`.
- Gate capability on `status` being active/trialing **and** the required `plan_rank`/feature flag.
- **Cache** the payload (`SUBSCRIPTION_CACHE_TTL_SECONDS`, default 60s) and **degrade offline** to the last good payload with a "cached, may be stale" notice — never hard-fail a gate on a transient network error.
- For gated **downloads**, call `GET /api/v1/downloads?platform=…` (server-side entitlement check + analytics) rather than streaming artifacts directly.

Reference: `scenarios/landing-page-business-suite/docs/integrations/subscription-entitlements-system.md` (runtime API, offline guidance, and subsystem design).

## Bundle membership (which SKU does this scenario belong to?)

That is a **strategy** decision, not an engineering one. The registry is operator-curated:

- **`docs/monetization/catalogs/CATALOG.md`** — the sellable units (business bundle `active`; lifestyle `candidate`; add-ons held until a parent has paying users).
- **`docs/monetization/catalogs/scenario-sku-map.json`** — the authoritative many-to-many scenario→SKU map with role (`headliner` / `depth` / `amplifier` / `future-headliner` / `shared-utility`).

If you believe a scenario should join/change a bundle, **do not edit the map** — surface it: the `catalog-strategist` proposes via a `catalog-mapping-update` decision and the operator curates. At the portfolio level, `morning-vision-walk` is where bundle fit gets assessed.

## How to wire it

- `prompt-manager skill read bundle-integration-steer` — the steer skill for integrating a scenario with the bundle/entitlement stack.
- Copy the closest reference implementation above (AI gateway for metered-by-token, audio-tools for metered-by-duration, LPBS entitlements for gated).

## Worked examples

| Capability | Mode | How |
|---|---|---|
| LLM chat/analysis inside a scenario | Metered | LPBS AI Gateway (`VrooliProvider` → reserve/charge per token); BYOK bypasses credits |
| TTS / STT (audio-tools) | Metered | LPBS credits per audio-ms via the usage ledger; insufficient-credits short-circuit |
| Studio-tier-only UI feature | Gated | `entitlements.plan_rank ≥ studio` / a `features[]` flag |
| Desktop app download | Gated | `status` active + `GET /api/v1/downloads` server-side check |
| Premium *and* expensive export | Metered + gated | entitlement check, then credit charge |

## Governance & related

- **Strategy canon is operator-write-only.** `docs/monetization/` (STRATEGY, TIERS, PRICING, CATALOG, scenario-sku-map) changes via operator-approved decisions only. This doc is engineering reference and may be updated by agents as the LPBS contract evolves — but it must stay a *pointer* to that canon, never a second source of truth for pricing/bundles.
- [`ECOSYSTEM.md`](./ECOSYSTEM.md) — where a scenario fits the whole; this doc is the monetization facet's engineering detail.
- [`docs/monetization/README.md`](../monetization/README.md) — the monetization plan of record.
