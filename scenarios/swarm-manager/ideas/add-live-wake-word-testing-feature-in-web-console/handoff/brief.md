# Idea Execution Handoff

This package captures the finalized swarm-manager idea context for downstream ecosystem-manager execution. It is regenerated from the latest finalized backlog state when idea execution begins so downstream work starts from a stable contract rather than scattered workshop artifacts.

## Execution Contract

- Backlog item: `idea/add-live-wake-word-testing-feature-in-web-console`
- Title: Add live wake word testing feature in web-console
- Target scenario: `add-live-wake-word-testing-feature-in-web-console`
- Recommended ecosystem operation: `generator`
- Recommended steer profile: `rapid-mvp`
- Item folder: `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console`
- Plan: `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/plan.md`
- Manifest: `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/handoff/manifest.json`
- Source index: `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/handoff/source-index.json`

## Downstream Requirements

- Read `plan.md` and `manifest.json` before creating the ecosystem-manager task.
- Use this `brief.md` file as the ecosystem-manager task notes.
- Preserve the origin metadata so later ecosystem-manager loops can trace back to the swarm-manager source artifacts.

## Product Intent

There needs to be a way to test whether the wake word detection actually works in practice — a live testing mode where the user can speak the wake word and see if it triggers correctly.

## Locked Decisions

- Round 001 `d1`: Audio capture approach for live testing -> Lightweight MediaRecorder loop in component
- Round 001 `d3`: Score visualization approach -> Score bar with threshold line
- Round 001 `d2`: Test interaction model -> Push-to-test button (hold to record, release to compare)
- Round 001 `d4`: Test result history -> Scrollable list of last 10 attempts
- Round 001 `d5`: Testing and verification strategy -> Unit tests for the test-loop hook/logic
- Round 002 `d2`: Hook file location and component extraction -> Hook in hooks/voice/wakeword/useWakeWordTest.ts, inline UI in VoiceInputSection
- Round 002 `d1`: Mutual exclusion mechanism between test and enrollment recording -> Shared isRecording flag lifted to VoiceInputSection
- Round 002 `d3`: Recording duration limits for push-to-test -> Min 0.5s, max 3s, auto-stop at max
- Round 002 `d4`: Score bar visual design details -> Plain div-based bar with CSS, inline threshold marker

## Remaining Open Decisions

- None.

## Execution Boundaries

- acceptance_allow: none recorded
- acceptance_deny: none recorded

## Validation Starting Point

- `vrooli scenario status add-live-wake-word-testing-feature-in-web-console`
- `scenario-completeness-scoring score add-live-wake-word-testing-feature-in-web-console`
- `scenario-auditor audit add-live-wake-word-testing-feature-in-web-console --timeout 240`

## Supporting Sources

- Spec: `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/spec.json`
- Workshop rounds:
  - `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/workshop/round-001.json`
  - `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/workshop/round-002.json`
  - `/home/matthalloran8/Vrooli/scenarios/swarm-manager/ideas/add-live-wake-word-testing-feature-in-web-console/workshop/round-003.json`

