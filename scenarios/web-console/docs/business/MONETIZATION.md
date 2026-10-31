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
- The coding-agent launcher's model choices come from the harness resource
  policy. The policy labels a configured billing source, but the chooser reports
  availability as unknown and does not verify credentials, account binding,
  entitlement, or execution. A listed native subscription, BYOK provider, or
  local model is not a Vrooli-funded coding-agent route. The API has an
  application-owned fixed `vrooli/coding` choice behind the qualification flag;
  the flag remains off, so production does not advertise that source.
- Account-bound coding execution is partially wired but not qualified. Web Console's browser
  sign-in now requests an S256 PKCE challenge and returns through a one-use
  loopback code. The login URL comes from the API's configured LPBS issuer, and
  Web Console exchanges the code through its same-origin API against that same
  issuer; the refresh credential stays in credential authority. The launcher
  now requires an explicit business-account selection and current LPBS usage
  facts before a qualified paid-source submission, then passes the account ID
  into session creation. The session adapter stores that binding and installs a
  session-scoped credential for the local OpenCode proxy. These paths have
  focused tests. An OpenCode 1.19.4 fixture completes a tool round trip through
  the Web Console proxy to an LPBS HTTP fixture; this is not yet one joined run
  through the real LPBS reservation service.
  The operator selected measured partial-stream usage as the failure policy.
  LPBS now has separate business-account usage, reservation, and event tables;
  each event retains the authenticated actor separately from its billed account.
  Measured provider token counts settle at the configured model rate, unmeasured
  failed streams release their hold, and failed settlement retains the hold for
  reconciliation. Focused fixtures cover account isolation, actor attribution,
  measured partial settlement, and idempotency. Account binding, fixed-model
  tool execution, cancellation, reservation, and settlement still need a joined
  cross-owner fixture before production qualification. Offer Desk lists the
  Business Bundle, Tier 1, and `ai_credits` as ideas, so there is no approved
  customer-facing offer or rate to show. Keep Vrooli-funded coding unavailable
  until both the joined accounting journey and operator-owned offer facts are
  established.
  No live account, paid coding-agent request, or charge was used for this work.
- `.vrooli/monetization.json` declares the routed inference meter. The shared meter
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
