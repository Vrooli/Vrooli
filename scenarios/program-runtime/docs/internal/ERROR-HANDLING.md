# Error Handling

## Shared contract

Read the [Error Handling shared guide](../../../template-manager/docs/internal/ERROR-HANDLING.md)
for the common error shape. Template Manager owns that contract and its
implementation references.

## Program Runtime recovery order

Use the [program contract guide](../guides/program-contracts.md) for the
canonical envelope, error classes, helpers, retry boundary, and library-call
semantics. Use the [construction guide](../guides/program-construction.md) for
the caller decision tree and reusable composition examples. Keep domain
recovery rules with the scenario that owns the operation.

When a program returns an unexpected result:

1. Read the Program Runtime execution record. `PROGRAM_STATUS_SUCCEEDED` means
   the program completed; it does not mean its domain task succeeded.
2. If an envelope exists, branch on its `status`, declared error class,
   `signals`, and evidence. `partial` names missing information; `unavailable`
   is unknown; `refused` means stop at the governance boundary.
3. For a nested `lib.<scenario>.<program>(...)` call, inspect and propagate the
   child envelope, error details, and artifact metadata before using its rows.
   Parent runner success does not promote a child failure or partial result to
   success.
4. Before retrying an operation that may have effects, reconcile the owner
   state. Retry policy belongs to the caller's decision tree; never hide a
   failed attempt with an in-program retry loop.
5. Use the kernel's `program.*` recovery helpers and the owner's domain error
   classes. Do not copy a transport classifier or collapse distinct refusal,
   unavailable, partial, and domain-failure cases into one workaround.

If the runtime fails before it prints an envelope, use the runner failure
detail. If output is truncated or malformed, the domain outcome may be unknown;
do not infer success from the runner status alone.

Add scenario-specific error classes, transport exceptions, or recovery rules
here only when they differ from the shared pattern and are not already defined
by the program contract guide.
