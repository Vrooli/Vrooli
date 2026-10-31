# Testing — Device Studio

## Shared guidance

Use repository `docs/TESTING.md`. Choose focused owner tests and relevant Test Genie phases for the change. Run scenario suites through `vrooli scenario test device-studio --phases <selected-phases>`. A server-owned run is awaited once with `test-genie runs wait --json device-studio <run-id>`; cancellation of the caller is not abort. Preserve source/evidence identity. A generated-type failure first requires the owning freshness verifier; do not patch generated output.

This foundation is validated through business-health requirements/PRD, experience-manager design contracts, documentation/link/schema checks and scoped scaffold tests. Comprehensive product runtime certification cannot pass before implementation and is not the foundation acceptance claim.

## Scenario-specific testing

[Requirements verification protocols](../../requirements/verification-protocols.md) define planned requirement cases. [ACCEPTANCE.md](ACCEPTANCE.md) defines craft and release boundaries. No manual protocol or aspirational experience claim is executed evidence until a real observation is retained.

Test domain compatibility/rejections, immutable revision conflicts, clocks/color/masks, rights/dependency closure, deterministic compilation, idempotent job admission, stale completion, cancellation/recovery, account switching/revocation, paid timeout/replay, external-media consent and isolated imports. Use recorded replay/fakes before bounded real provider tests. No live paid test without actual configured authority and cap.

Run continuous final-renderer phone/lid footage, not only first/last frames. Use bright backgrounds around the physical pill and independently configurable browser/desktop chrome. Test landscape/portrait 24/30 fps, VFR timestamps, explicit audio policy, native/portable reimport, disk-full and child cleanup. For each claimed desktop OS, test a fresh private install without checkout/fleet, offline local work, native GPU/CPU, secure store, callback transport, signed update and project retention.

Tag real tests with stable [REQ:ID] markers only when they exercise that requirement. Keep product requirements not_implemented until evidence supports promotion. Template notes tests validate the example only. Do not lower targets or convert a future feature into an informational warning to satisfy certification.

[Job and recovery state contracts](../concepts/FLOWS.md).

Cancellation regressions must cover mixed local and remote descendants: a stopped local process cannot hide an unknown remote execution, and a reconciled remote call cannot hide a still-running local child. Financial reconciliation remains independent of execution quiescence.
