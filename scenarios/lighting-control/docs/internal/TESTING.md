# Testing — Lighting Control

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Record the behaviors this scenario must prove, deterministic fixtures, required
external services, and any exceptions to the shared harness. Link to real local
tests as they are added. Keep execution receipts with their owning plan.

## Binary startup

`api/main_e2e_test.go` uses `api-core/boottest` to build this API, verify its
health identity in isolated storage, and check shutdown. Keep the expected
service name in the scenario test. Shared process machinery and failure
regressions live in `packages/api-core/boottest`; see its package README section
for configuration and evidence limits. The existing E2E gate runs this test.

## Independent addressable-zone review A — 2026-11-11

The scoped W3 adversarial review found and rechecked three material defects: legacy name migration could overwrite a fixed peer's durable zone ID; boolean whole-controller Off authority omitted physical unmapped peers; and resize success followed by disconnect erased partial-write evidence. Production repairs preserve IDs transactionally, compare the exact complete physical fingerprint roster on the SDK connection before a global Direct transition, and retain resize write facts without claiming a completed render.

Independent full control-package race tests with private desired-behavior overlays passed (6.136s); public authentication-package race tests passed (9.218s); governed protobuf freshness verification passed. Evidence, source digests, command failures and applicability limits: `.vrooli/work/reviews/native-zone-a/REVIEW.md` in the isolated native-lighting delivery checkout. This finding-bearing pass plus affected repair recheck does not count as a fresh clean maturation pass. Actual native runtime, physical output, boot/logout/resume and hardware reconnect remain separate unverified acceptance gates.
