# Tier 3 · Mobile (iOS / Android)

Mobile apps inherit every challenge from desktop but add mobile-specific constraints (sandboxing, store policies, limited compute, intermittent connectivity).

## Current State

- No scenario currently targets native mobile packaging.
- `scenario-to-mobile` does not exist yet (placeholder for future automation).
- We have zero deployment metadata for mobile readiness.

## Target Experience

1. `deployment-manager` lets us pick "mobile" as the target platform.
2. It grades each dependency for mobile fitness (RAM, CPU, offline capability, binary size limits, background execution policies).
3. It suggests mobile-friendly swaps (e.g., `sqlite` for relational data, `chromadb` for vectors, OpenRouter/hosted inference instead of local LLMs).
4. `scenario-to-mobile` generates a React Native/Capacitor/Flutter shell that embeds the UI bundle and either:
   - Talks to bundled lightweight APIs, or
   - Securely proxies to cloud APIs prepared via Tier 4.
5. The first-run experience prompts for user secrets (API keys) and stores them via platform keystores.

## Blocking Questions

- What runtime will we standardize on? (React Native via Expo? Capacitor bridging desktop UIs?)
- How do we chunk large models/resources so mobile installs stay <200 MB?
- Which resources are categorically unsupported (GPU-only models, multi-GB vector stores) and how do we auto-swap them?

## Immediate Actions

- Capture mobile-readiness metadata on resources as we build them (`deployment.platforms.mobile`).
- Reuse dependency fitness scoring logic from Tier 2 with additional constraints (battery, offline vs online, push notifications).
- Spec the `scenario-to-mobile` scenario (see [Scenario Docs](../scenarios/scenario-to-mobile.md)).
