# Aquila monetization: make existing tools more useful

Aquila's commercial hypothesis is that one coherent workspace can reduce the
attention and coordination cost of using several coding agents. Some buyers may
also use existing subscriptions and local resources more effectively. Neither
benefit establishes measured dollar savings. The product must earn a place in a
buyer's workflow before a subscription can be justified.

## What the buyer is paying for

Position the actual Vrooli offer around a maintained, integrated experience and
the eligible suite/services it provides. Validate packaging, updates, support and
entitlements against the selected offer before promising them. Do not present
free/local workspace functions as technically requiring payment when they do not.
Do not imply that terminal access or an external coding-agent subscription must
be purchased again through Aquila.

The operator's existing terminal is a credible alternative. Demonstrate the
repeatable advantage: consistent agent/session handling, attention state, rich
messages, linked-file review, context export and eligible voice controls. Test
whether that combination is worth the actual price to the intended buyer.

## Keep four funding boundaries separate

| Path | What it means | Commercial boundary |
|---|---|---|
| Existing coding agent/account | A configured provider's CLI/session continues using its own authentication and available native features | Provider plan, limits and eligibility remain provider-owned; Aquila does not pool entitlements or guarantee native voice is included |
| Local compute | A supported configured local service handles an eligible feature | No provider API-key charge is implied; hardware, electricity, setup and operating costs remain real |
| BYOK | A user-authorized API key funds a supported provider call | API billing is separate from a consumer coding-agent subscription; keys remain with the credential authority |
| Vrooli-provided assistance | Eligible signed-in requests use Vrooli's metered owner route | Subscription/credits, reserve/settlement and current entitlement rules apply; this is not unlimited third-party inference |

These paths belong to specific features. They do not establish one global route
that always minimizes cost. Explicit source choices, availability and configured
priority determine behavior; preserve the user's control and do not silently
change billing expectations.

## Current implementation evidence

AI command assistance constructs an Ollama/OpenRouter/Vrooli provider chain in
`api/main.go`. `api/internal/ai/` owns configuration, enabled providers and actual
selection/failure behavior. The mere ordering of that chain does not establish
current availability, provider-model quality, general coding-agent routing or a
universal cheapest-path guarantee.

- Local Ollama assistance can work without a Vrooli account when configured.
- OpenRouter BYOK uses credential resolution; do not store or display secrets in
  the browser or film.
- Vrooli inference uses AI Gateway and its LPBS/account/`ai_credits` enforcement.
- Native browser agent voice is a separate Codex/version/live-session capability.
  Dictation uses Audio Tools. Text-to-speech has its own service/browser paths.
  Voice selection is not proof that all optional dependencies are running.
- `monetization.json` declares the routed inference meter. The shared meter
  inventory assigns `voice_minutes` to Audio Tools; Aquila does not duplicate it.
- Aquila does not enforce plan rank locally. Signed-out local terminal capability
  remains available; account and metered-assistance behavior follow their owners.

See [Bundle integration](../internal/BUNDLE_INTEGRATION.md),
[Go to market](GO-TO-MARKET.md), `api/agent_voice.go`, and
`ui/src/hooks/useRoutedVoiceInput.ts` for current boundaries.

## Efficiency claims and validation

Use "get more from the agents you already have" or "keep your agents and their
work in one workspace" as positioning. Qualify actual provider installation and
setup. A potential reduction in duplicate effort or idle paid capacity is a
hypothesis until observed against a buyer's baseline. Do not claim a percentage,
a guaranteed subscription reduction, automatic quota balancing or invoice savings
from token counts alone.

Agent Manager can help investigate supported run cohorts and recurring friction;
its [monetization brief](../../../agent-manager/docs/business/MONETIZATION.md)
separates operational evidence from actual billing. The proposed unified limits/
reset dashboard remains future discovery, not a paid feature in this offer.
