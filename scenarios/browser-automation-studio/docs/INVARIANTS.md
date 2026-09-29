# Browser Automation Studio Invariants

This registry indexes cross-cutting rules that keep workflow execution evidence safe to consume across the API, CLI, and replay/export surfaces. Code remains the source of truth; each entry is anchored to its enforcement site and violation test.

## Critical

| Invariant | Statement | Enforcement | Code anchor | Test evidence |
|---|---|---|---|---|
| `harDerivativeIsRedacted` | A HAR derivative that leaves protected storage is redacted according to the evidence policy. | Runtime sanitizer with a safe default policy. | [CODE: api/services/evidence/har.go#SanitizeHAR] | [CODE: api/services/evidence/policy_test.go#TestSanitizeHARRedactsHeadersQueriesAndBodies] |
| `protectedEvidenceHasNoPublicLocation` | Protected evidence metadata never includes a browser-accessible artifact URL. | Runtime access-policy guard before URL construction. | [CODE: api/services/workflow/execution_results.go#artifactURLForPolicy] | [CODE: api/services/workflow/execution_results_test.go#TestListExecutionArtifactsDoesNotExposeCapturePath] |

## Important

| Invariant | Statement | Enforcement | Code anchor | Test evidence |
|---|---|---|---|---|
| `replayArtifactHasIntegrityDigest` | Every replay-package artifact has a SHA-256 digest and portable metadata instead of a storage path. | Runtime package builder rejects incomplete artifact input. | [CODE: api/services/evidence/replay_package.go#BuildReplayPackage] | [CODE: api/services/evidence/replay_package_test.go#TestBuildReplayPackageIsStorageIndependentAndProtectsHar] |
| `replayPackageUsesStrictSchema` | Persisted replay packages are decoded with unknown fields rejected so producer and consumers cannot silently drift. | Strict `protojson` decode at API and export consumer boundaries. | [CODE: api/services/workflow/execution_results.go#GetExecutionReplayPackage] | [CODE: api/internal/protoconv/evidence_contract_test.go#TestReplayPackageContractRejectsUnknownFields] |

## AI navigation lifecycle

These invariants cover the callback, observation and recovery boundary. They
are source-level contracts until a fresh managed candidate is qualified.

| Invariant | Statement | Enforcement | Code anchor | Test evidence |
|---|---|---|---|---|
| `navigationTerminalStateIsImmutable` | Terminal navigation sessions ignore later callbacks and duplicate step numbers. | Session admission fence before mutation or fan-out. | [CODE: api/services/vision/playwright_navigator.go] | [CODE: api/services/vision/playwright_navigator_test.go] |
| `navigationCallbackUsesOneRedactedProjection` | An accepted callback is redacted once before history, recording and WebSocket consumers receive it. | Navigator callback normalizer and redaction helpers. | [CODE: api/services/vision/playwright_navigator.go] | [CODE: api/services/vision/redaction_test.go] |
| `navigationObservationLossIsRecoverable` | Status transport loss is non-authoritative; the live navigation identity remains available for stop or recovery. | UI observation state and generation fence. | [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.ts] | [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.test.ts] |
| `navigationSuccessorWaitsForSettlement` | A successor navigation cannot be admitted while the prior server operation remains unsettled. | Start admission checks the retained identity and handoff state. | [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.ts] | [CODE: ui/src/domains/recording/ai-navigation/useAINavigation.test.ts] |
| `managedShutdownAlwaysRunsOwnerCleanup` | HTTP drain expiry does not skip browser, sidecar or database cleanup. | BAS configures the shared server with separate drain and cleanup budgets; the shared server tests own the generic implementation. | [CODE: api/main.go] | `packages/api-core/server/server_test.go` (owner tests) |

## Drift Gate

The focused evidence, workflow, API-handler, and export tests are the current drift gate. The standard scenario suite runs them together with the generated proto contract. Any new replay consumer must use `ReplayPackage`, not filesystem paths or object-store locations; the package boundary is documented in [DOC: docs/SEAMS.md#evidence-and-replay-package-seam].

## Accepted Gaps

There are no accepted gaps for these evidence invariants. Legacy executions may not have an `evidence.proto.json` package; export keeps a read-only fallback to the existing timeline solely for those historical records. New execution writes produce the versioned package.
# Storage retention invariants

Recordings and captures are regenerable owner data with a seven-day retention
budget. The retention sweep selects complete execution directories and uses
the shared retention engine for containment, re-stat, deletion, and receipts.
It may resume after a timeout; a partial sweep does not claim that the budget
is true.

Performance traces use one compressed trace per execution. The trace is
opt-in for diagnostics and is off by default, so a normal run does not emit a
stat/open line for every repository file.
