# Known Issues & Follow-Up Tasks

> This file records what is not yet proven. It is intentionally narrower than
> the target contract: a passing contract check does not make an untested
> deployment claim true.

## Work ladder

- Rung: W3
- Evidence: onboarding had a scenario-specific Cloudflare bootstrap route, client, and UI predicate; those were removed. Credential provisioning now uses the generic descriptor-driven path, while tunnel-manager owns completion of its derived credentials during its lifecycle. Onboarding API and UI suites pass.
- Blocker: the broader remote VPS evidence limitation below remains; it does not block the generic onboarding implementation.
- Measured: 2026-08-18

## Current evidence

- Final authoritative run `20260811-112723-3656bc67`: **21/21 phases passed**.
- Final post-receipt authoritative run `20260812-235209-0fc3991d`: **21/21
  phases passed**, with Test Genie terminal verdict `PASS` and zero failed
  phases after adding the dedicated bundle-tier evidence receipt.
- All six declared journeys now have fresh, selector-resolved adhoc executions completed on 2026-08-12 with `--wait --requires-video`. Each returned `EXECUTION_STATUS_COMPLETED` and produced one non-empty WebM artifact:
  - `consent-to-a-host-safeguard`: run `b739118c-24f6-4a33-bae3-93b2a7ffb721`, artifact `execution-b739118c-24f6-4a33-bae3-93b2a7ffb721-page-1.webm`, SHA-256 `07825fe5443d0cfbae19287a1d2ad57066e2c2ae386e077f2178158d56530a1c`
  - `continue-with-a-degraded-install`: run `db58d95d-d7f5-4b06-841d-f8af826ad768`, artifact `execution-db58d95d-d7f5-4b06-841d-f8af826ad768-page-1.webm`, SHA-256 `278a9c0a6df94a7399ca5d5685094d39960425cfd168e238f431998d0ac1dc99`
  - `first-run-to-applied-install`: run `1a364e33-d0ec-4c7d-ad6d-401f3cf86e06`, artifact `execution-1a364e33-d0ec-4c7d-ad6d-401f3cf86e06-page-1.webm`, SHA-256 `59d9a0c5ca53f499b7d6a07842261e7bf68622ca7d7db0093b2b4d433404d87b`
  - `headless-parity-with-the-ui`: run `2ad278a7-6fd6-474a-b004-8781593602e0`, artifact `execution-2ad278a7-6fd6-474a-b004-8781593602e0-page-1.webm`, SHA-256 `408c46461d064f3535e3659ab7bda43022482d78d1bcf9d82a4594debc2bcb1d`
  - `provision-credential-on-headless-host`: run `7810f88d-cd70-4922-a3bf-aa906793c517`, artifact `execution-7810f88d-cd70-4922-a3bf-aa906793c517-page-1.webm`, SHA-256 `7b14595b0b5315e94a830ce8f882c16cc6f47e9127dadd99cf88144c0eea6754`
  - `re-entry-revise-and-reapply`: run `e1ba1464-f9fe-4031-a43d-44afd61c3ca5`, artifact `execution-e1ba1464-f9fe-4031-a43d-44afd61c3ca5-page-1.webm`, SHA-256 `33b10f56791cab8d3200548dde1176341a0941d5908cd3490e2c7015c6698998`
- `experience-manager spec validate vrooli-onboarding`: **L3, zero findings** after the picker binding and journey updates.
- The bundle-tier browser proof was also executed with only `BUNDLE_ROOT` set:
  BAS run `7076a1e2-747f-49b8-b91c-180d6f34fc99` completed the all-steps case,
  returned one non-empty WebM (1,400,307 bytes, SHA-256
  `127effab0d00fc8e9d65b1f8a3966f8a1c15090d31c6364b14fd114aafeb833c`), and
  the complete V2 endpoint sweep returned HTTP 200 before the normal runtime
  was restored.
- `vrooli scenario requirements validate vrooli-onboarding`: **zero findings**; 68/70 requirements are complete, with remote VPS proof and profile intake intentionally remaining.
- Bridge transport proof: typed dispatch to the online Linux node
  `swarminator` completed remote Test Genie run
  `20260812-232719-f5532e2a` with **21/21 phases passed**. A subsequent
  cross-OS gate was attempted against both registered nodes; it failed before
  validation because the Darwin node lacks the generated proto packages and
  the Linux delivery could not materialize the requested revision. This is
  recorded as Bridge/deployment evidence, not as onboarding success.
- 2026-08-12 real Bridge onboarding attempt: handoff was enabled at
  `http://127.0.0.1:19798/api/v2/handoff`, which returned the capability-shaped
  selection for the durable Linux Machine. Bridge operation
  `1de33768-a79d-4bf5-b95b-8e0741ecb47d` then stopped at `ssh-setup` with
  `ssh_setup_failed`: the machine's Bridge-managed key is not authorized and
  no first-touch password was supplied. The operation never reached remote
  apply/readiness, so no remote success is claimed.

- `vrooli scenario test vrooli-onboarding` final run `20260811-105557-49d03aed`:
  **21/21 phases passed**.
- `test-genie runs findings 20260811-105557-49d03aed`: **PASS, 21 phases,
  zero failed phases**.
- `experience-manager spec validate vrooli-onboarding`: **L3, zero findings**;
  all 11 pages and all 6 journeys are active.
- `vrooli scenario requirements validate vrooli-onboarding`: **zero findings**;
  68/70 requirements are complete, with remote VPS proof and profile intake
  intentionally remaining planned.
- API coverage is 75.1%, CLI coverage is 76.5%, and UI coverage is 96.32%
  statements / 85.21% branches (217 tests).
- Six BAS journey cases plus seven generated experience observers are registered
  and executed by the green workflow phase. The latest run is the durable
  evidence handle above.

- 2026-08-12 validation note: BAS now has compiler coverage for scenario-owned
  selector manifests, adhoc `--wait` completion, and a fail-closed
  `--requires-video` artifact contract. The server-owned BAS run
  `20260812-041517-894d9b4e` nevertheless failed outside those focused tests:
  its UI coverage command failed the merged floor, Lighthouse reported
  performance `0.46 < 0.75`, and MinIO was not running. The six requested
  video-backed journey artifacts are now earned by the six fresh executions
  recorded above. The comprehensive BAS suite still has unrelated quality,
  docs, performance, unit, business, and workflow failures; bug
  `knw-1786509853983617417` remains open for that broader external validation
  path and is not silently treated as green.

- 2026-08-12 release-authority readiness: `/api/v2/readiness` now reports the
  control-plane status as `ready`, `missing`, `degraded`, or `unsupported`,
  including whether the managed key and repository trust anchor match. The
  current host reports `configured=false` and `trust_anchor_match=false`, so
  the exact remediation remains `vrooli release-authority init`.

- 2026-08-12 final validation: onboarding server-owned run
  `20260812-222531-a0503548` passed **21/21 phases** in 204 seconds;
  `experience-manager spec validate vrooli-onboarding` and requirements
  validation both report zero findings. The all-steps accessibility receipt
  is BAS run `837143e8-e343-44d2-af88-1c1c45f27c29`, with a non-empty video
  artifact whose SHA-256 is
  `fdcc73dc9ba4b89db1462eb3ebecd132ac947abe95a68aee420cda958f3985bd`.
  The immutable collection comparison
  `vrooli-onboarding-final-20260812` completed with classification **clean**:
  onboarding was clean and BAS/Test Genie were preexisting for the captured
  scope. No baseline recapture was made.

## Remaining blockers and limitations

### Bundle acceptance is green; remote VPS acceptance remains unavailable

The onboarding boundary and bridge unit tests pass, including the transport-
neutral remote selection/apply/readiness client. The final full bridge run
`20260811-104540-c8815d7f` passed **20/20 phases**, including unit, security,
proto, and UI-health. `proto-health validate scenario vrooli-bridge` also
passes; the remaining findings are warnings for pre-existing domain/template
layout and unreachable legacy messages.

The bundle proof is now live: a staged catalog containing 117 scenario
manifests, 26 resource manifests, 48 tool manifests, and 23 safeguard
manifests was started with only `BUNDLE_ROOT` set. Every V2 catalog, host,
credential, union, readiness, surface, and session endpoint returned HTTP 200.
Bundle-local app-data fallback was exercised after both repository and storage
root variables were removed.

There is no real VPS in the current Bridge inventory. The available nodes are
the local Linux `swarminator` and local Darwin `minimouse`; the previous SSH
first-touch attempt to `minimouse` stopped because the Bridge machine key was
not authorized and no password credential was supplied. Consequently the
remote VPS selection/apply/idempotency/union proof remains externally blocked
and is not claimed complete.

### Trusted-experiment receipt signing is explicitly deferred

2026-08-12 deferral: provider receipts currently carry producer, run, artifact,
size, and SHA-256 evidence, but are not signed by a separate trusted-experiment
authority. Revisit when the release-authority trust anchor is initialized and
the evidence provider exposes a signed receipt contract; until then, receipt
hashes remain integrity evidence rather than release authorization.

### Plan baseline comparison is clean

The captured Git Control Tower baseline exists and is synchronized, but the
fresh three-member comparison `vrooli-onboarding-final-20260812` is
**clean**. The baseline remains the original pre-edit collection at SHA
`8019a9ecb760b1c15b53e50eef3e7ba74140bc7b`; the comparison classified the
BAS and Test Genie changes as preexisting and onboarding as clean. This is a
valid final verdict without recapturing the baseline.

### Alternate-state experience evidence remains fixture-governed

The experience contract reconciles cleanly at L3, all pages/journeys are active,
and the default-route journeys execute. Alternate/error-state assertions that
need deterministic backend fixtures or computed-style evidence remain
aspirational; they are not represented as machine-proven claims.

### Advisory UI debt remains

- The scenario has local Button/SearchInput/StatusBadge components but has not
  adopted the external `react-component-library` catalog.
- UI health still reports advisory focus-zoom and screen-reader clipping
  heuristics, plus a raw empty-state primitive in the glossary.
- Template provenance is not declared for this pre-template scenario.

### Explicitly deferred scope

Profile preselection, integrations, mobile tiers, and credential lifecycle
repair/recovery remain outside this plan's shipped scope. The profile
requirement is deliberately planned rather than presented as complete.

## Resolved in this implementation

- Operator-state writes now use one schema-validated, locked merge-patch
  authority that preserves unknown fields and records apply completion.
- Resource enablement, host requirements, closure/union projection, readiness,
  credentials, surface metadata, session state, and apply reporting use the
  new authority and typed degraded responses.
- Desktop catalog requirements and missing-catalog packaging/API tests cover
  bundle mode; the runtime package tests pass.
- The UI, interactive CLI, declarative wizard, and bridge selection boundary
  share the same selection-to-patch translation; endpoint and no-dead-command
  contract tests pass.
- BAS selector manifests are generated as part of UI build/test, all six
  onboarding journeys are executable, and the comprehensive onboarding suite
  is green.
