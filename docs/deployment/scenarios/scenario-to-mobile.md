# Scenario: scenario-to-mobile (Future)

## Goal

Produce native mobile packages (iOS + Android) for any scenario once dependencies are made mobile-friendly.

## Proposed Responsibilities

1. **Framework Bridge** — Decide on React Native / Capacitor / Flutter and provide scaffolding.
2. **UI Packaging** — Bundle production UI assets inside the chosen framework.
3. **API Strategy** — Either embed lightweight APIs locally or connect securely to cloud endpoints prepared via deployment-manager.
4. **Secret Prompts** — Integrate with secrets-manager metadata to prompt/store user secrets in Keychain/Keystore.
5. **Platform Services** — Handle push notifications, background sync, offline cache, etc.
6. **Store Compliance** — Generate assets/metadata for App Store / Play Store submission (icons, privacy labels, etc.).

## Dependencies

- deployment-manager (bundle manifests + swap decisions).
- scenario-dependency-analyzer (mobile requirements).
- secrets-manager (secret templates).
- scenario-to-cloud (if mobile clients connect to hosted APIs).

## Roadmap Sketch

1. Spike on packaging a simple scenario using React Native + WebView + REST API.
2. Define bundle manifest schema extensions for mobile-specific assets (app icons, splash screens, push configs).
3. Implement CLI `vrooli scenario to-mobile <name>` that reads deployment-manager bundles.
4. Automate store metadata generation.

Until this scenario exists, tiers 3 deployments should be considered conceptual only.
