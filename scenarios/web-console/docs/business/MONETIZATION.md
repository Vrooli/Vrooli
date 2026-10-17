# Monetization

Web Console is the first customer-facing Vrooli scenario. Its commercial
boundary is resource ownership:

- local Ollama is free and works without an account;
- an OpenRouter key supplied by the user is BYOK and is stored by the
  credential authority, not by the browser;
- Vrooli-provided inference is routed through `ai-gateway`, which forwards the
  consumer token to LPBS and owns `ai_credits` reserve and settlement.

Web Console does not enforce plan rank locally and does not charge voice or
machine operations. The account tab is optional. Signed-out users retain local
terminal capability, while signed-in users can see the subscription state and
the source selected for AI generation.

The durable evidence contract is:

1. `service.json` declares the subscription refresh credential and OpenRouter
   BYOK credential.
2. Credential provisioning is declaration-checked and metadata-only on output.
3. `monetization.json` declares only the routed inference meter, with the
   trusted `ai-gateway` path as its enforcement boundary.
4. `meter-inventory.json` has one owner for `voice_minutes`: `audio-tools`.

See [`docs/internal/BUNDLE_INTEGRATION.md`](../internal/BUNDLE_INTEGRATION.md)
for runtime wiring and validation evidence.
