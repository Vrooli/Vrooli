# Program Runtime child-envelope fault injection

## North Star

Every program that adopts `program.child_envelope` has a generated fixture for
the five shared child failure shapes.

## The rungs and their gates

Test Genie discovers every repository program source that calls
`program.child_envelope`, including programs owned by other scenarios. It
generates five rows per program and records that program's declared binding
requirements (or nested library dependency) plus the fallback observed through
the production helper. Each row pins the source and contract through a SHA-256
digest and invokes the production helper with a generated child handle. The
fixtures cover unavailable, degraded, healthy-empty, malformed, and partial
results. The result is explicitly fixture-only and cannot qualify native
provider behavior.

The phase also executes every adopting parent source through the Program
Runtime `SessionKernel`. It injects the same five child outcomes at each
declared library boundary and supplies inert handles for direct bindings, so
no scenario service or external effect runs. The receipt records invoked
child boundaries, normalized child envelopes, and the parent status/phase.
This exercises real parent control flow under faulted dependencies; it remains
injected fixture evidence, not native provider or live-binding qualification.

## What each finding means

`program_runtime_fault_injection.execution_failed` means a generated row did
not preserve its declared status or normalize an empty or malformed child
result as the parent contract declares.

## The canonical fix

Keep envelope handling in `program.child_envelope`; update the helper or the
contract that owns the expected failure classification. Do not add a local
retry or duplicate envelope parser.

## How to verify

Run `vrooli scenario test program-runtime --phases programs,fault-injection`.
The focused source-level flow tests are in
`scenarios/program-runtime/kernel/tests/test_child_composition.py`.

The helper-level matrix separately proves status preservation and normalization
for each adopting program contract. Its capability requirements are
descriptive and are not invoked. The parent-source matrix adds actual program
execution but replaces dependency responses with generated fixtures. Neither
matrix invokes each program's real service bindings, so neither proves native
provider behavior. Owner fixtures continue to cover specific composition
flows in `test_child_composition.py`.
