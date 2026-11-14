# Error handling — Device Studio

## Shared contract

Use api-core typed transport/validation and template error handling. Errors retain correlation/job/attempt IDs without secrets. UI and CLI render the same domain status and repair advice.

## Scenario details

Distinguish incomplete draft, incompatible capability, stale revision, missing/corrupt artifact, unsupported renderer, exhausted local capacity, unauthorized object, privacy refusal, exhausted budget, insufficient credits, expired account, provider unavailable, pending reconciliation and visual rejection. These are proposed semantic categories; generated enum names are not established here.

A failed render preserves input and valid checkpoints. Retry is offered only when safe; an uncertain paid dispatch first reconciles the existing operation. A release-pending asset can remain locally usable. A cancelled remote job can have pending charges. Unknown measurements or costs are shown as unknown, never zero. User-facing messages explain the consequence and next action without stack traces; safe diagnostics remain in logs. Never replace failure with a fabricated preview/output.
