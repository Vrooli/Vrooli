# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario audio-tools`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

Purpose: Audio Tools provides fast, reliable, portable voice capabilities that any Vrooli application can adopt through shared contracts. Speech-to-text and voice input are the first end-to-end qualification slice; text-to-speech, transcript summarization, and audio processing retain their existing scope and share the same provider and privacy boundaries.

Users include people dictating in consuming apps, developers embedding voice, and operators managing local capacity or hosted service. Local compute and bring-your-own-key (BYOK) remain explicit alternatives. The owned hosted route adds subscription or purchased-credit value wherever a consumer adopts voice; it is not an implicit charge on local or BYOK use.

Deployment surfaces include the Audio Tools UI, CLI, Connect and WebSocket APIs, shared browser capture, and consumer-owned host adapters. Portability is a stable integration contract plus evidence for declared host/device profiles, not a claim that every local model runs on every device.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Long-form dictation trust | When audio is captured, Audio Tools shall account for every interval as processed, retained for replay, or explicitly recoverable failure across interruption, reconnect, slow consumer, restart, and contention.
- [ ] OT-P0-002 | Provider-parity stable engines | When an engine is offered as stable, Audio Tools shall require the same no-loss, durability, recovery, policy, quality, and product-path trust floor; Whisper and Kyutai remain initial independent qualification candidates.
- [ ] OT-P0-003 | Explicit speaker-policy safety | When extraction or verification is requested, Audio Tools shall expose applied or degraded policy outcomes and shall fail closed when required speaker policy cannot be applied.
- [ ] OT-P0-004 | Portable explicit route contract | When any consuming app requests voice input, Audio Tools shall expose one provider-neutral local, BYOK, or owned-service contract with actual route, streaming capability, and actionable failure, without an unapproved remote or paid fallback.
- [ ] OT-P0-005 | Responsive genuine streaming | When a qualified interactive route captures speech, the consumer shall display revisable partial text and commit the complete final tail within the approved cold/warm cohort SLOs; batch-only routes shall be identified as such.
- [ ] OT-P0-006 | Measured recognition quality | When a speech route is qualified, Audio Tools shall meet the approved language and noise-cohort quality floors on a versioned, consented or licensed held-out corpus and shall report missing cohorts and interval loss separately.
- [ ] OT-P0-007 | Subscription and credit correctness | When owned voice is requested, shared service owners shall authorize entitlement and usage, deliver inference, and reconcile reservations and idempotent settlement under one versioned policy; local and BYOK shall not incur a Vrooli voice-service debit.
- [ ] OT-P0-008 | Portable capability and device qualification | When host, browser, model, or acceleration capabilities differ, Audio Tools shall select a compatible declared path or return an explicit unavailable state; every claimed supported route and device shall have matching evidence.
- [ ] OT-P0-009 | Private bounded voice lifecycle | While voice data or credentials are processed or retained, Audio Tools shall enforce authorized access, bounded recoverable retention and deletion, and metadata-only diagnostics, with explicit consent before changing the processing destination.
- [ ] OT-P0-010 | Trustworthy full-path qualification | When release or development completion is evaluated, Audio Tools shall require current owner-backed evidence for every required target and cohort, distinguish simulated from native and live-provider results, and reject missing, stale, truncated, or failed evidence.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Provider-neutral evidence | When operators compare experiments, Dictation Studio should expose persisted provider-neutral metrics and promotion verdicts by provider, strategy, policy, fault, and replay cell.
- [ ] OT-P1-002 | Mobile recovery and diagnostics | When a mobile turn is degraded or interrupted, users should be able to inspect coverage, recover retained audio, and export metadata-only diagnostics without developer tools.
- [ ] OT-P1-003 | Replaceable maintainable adapters | When providers or device technology change, maintainers should extend versioned capability and host adapters with shared conformance tests without duplicating capture, recovery, routing, or billing rules in consumers.
- [ ] OT-P1-004 | Consistent shared voice capabilities | When TTS or transcript summarization is enabled in a consuming app, it should use the same explicit route, capability, privacy, and usage boundaries as STT, with operation-specific delivery, quality, and billing acceptance.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Expanded device qualification | Where additional native devices or browser variants are proposed, Audio Tools may expand recorded qualification across iOS Safari, installed PWA, Android Chrome, and desktop microphones without implying untested local-engine support.

## 🧱 Tech Direction Snapshot

Preferred: Go owns session, routing, interval accounting, and policy logic; protobuf defines transport contracts; shared browser capture owns capture and recovery; consuming apps own thin transport and presentation adapters. Engine capability, host runtime, and streaming strategy are separate replaceable boundaries. Whisper and Kyutai are initial adapters, not permanent architectural dependencies.

Canonical PCM spans, ordered commits, bounded journals, cancellation, and recovery retain provider-independent identities. Shared monetization and hosted-delivery owners retain entitlement, catalog, wallet, metering, and settlement authority. Skills carry improvement judgment; governed programs compose bounded owner measurements; scenarios retain state and acceptance evidence.

Non-goals: training new speech models, unrelated consuming-app behavior, a private billing ledger, inventing a new multi-tenant GPU platform, simultaneous replacement of all voice implementations, or claiming unsupported device/language coverage.

## 🤝 Dependencies & Launch Plan

Resources: capability-selected local STT/TTS resources, audio-format tooling where required, and optional speaker or summarization providers; resources are not universally required for BYOK or owned service. Shared dependencies include the browser capture package, consumer adapters, Landing Page Business Suite and shared monetization/hosted-delivery services, corpus and experiment services, Test Genie, and native-device validation owners.

Sequence: align this contract and its obligations; qualify measurement instruments and lifecycle; repair the local shared-consumer streaming slice; validate local, simulated BYOK, and simulated subscription/credit routes; wire and separately qualify authorized live delivery; then qualify the approved device and quality matrix and harden adapter architecture against regression. Audio Tools, Web Console, and Swarm Manager each retain their own consumer-path evidence. A local slice is incremental progress, not completion of the full voice target.

Development uses one approved Swarm mandate pointing to audio-tools-improve, with successive in-scope repairs under the same authority. Harness dispatch, budgets, acceptance resolvers, and continuation must be qualified by their owners before autonomous launch. Tech Tree Designer is a deferred planning enhancement, not a prerequisite for the explicitly requested canonical documentation update.

Risks and remaining review decisions: numeric SLO adoption, realized corpus and native device inventory, credential/privacy boundaries, model capacity, hosted gateway availability, billing delivery semantics, and capped live-provider spending. Supporting documentation makes these decisions explicit; this PRD does not grant execution, spending, deployment, or publication authority.

## 🎨 UX & Branding

Accessibility: Provide keyboard-accessible voice controls, non-color-only state, readable recovery guidance, and calm, privacy-conscious language. Distinguish preparation, listening, revisable partial text, committed text, final drain, recovery, and failure. Show actual processing destination, capability, and relevant cost or authorization state without requiring developer tools. A recording animation, healthy process, or successful wrapper must never imply complete transcription or settled billing.

## 📎 Appendix

Contract revision: portable-voice-v1, regenerated 2026-09-09 at the user's request. Existing OT and ATD identifiers retain their original meanings; added obligations start unearned. Requirements synchronization alone earns completion.

The PRD owns product scope. docs/internal/TESTING.md owns cohort methodology and pending acceptance decisions; docs/internal/PERFORMANCE.md owns latency interpretation; docs/business/MONETIZATION.md owns commercial proposals and shared-owner boundaries; docs/concepts/ARCHITECTURE.md owns target seams. The historical local-dictation-proposal.json is a non-launchable subset, not the full mandate. The setpoint-read program enumerates target scope but remains a diagnostic instrument until owner-backed acceptance receipts are joined; an unknown outcome is never completion.
