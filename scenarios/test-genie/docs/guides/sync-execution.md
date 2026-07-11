# Server-Owned Execution for Agents

Test Genie executions are server-owned. Starting a run creates durable evidence
and returns a run ID; interrupting the caller does not cancel the run.

## Start an execution

```bash
test-genie execute my-scenario --preset comprehensive
```

The response prints the execution/run ID and a recommended single wait command.
Do not start a second execution merely because a terminal detached.

## Reattach once

```bash
test-genie runs wait --json --timeout=840 my-scenario <run-id>
```

This blocks until the durable run reaches a terminal state. A non-zero exit
means the phases reported a failed or aborted result; inspect the persisted run
instead of inventing a replacement suite:

```bash
test-genie runs show my-scenario <run-id> --json
test-genie runs follow my-scenario <run-id>
```

Use `test-genie runs abort my-scenario <run-id>` only to deliberately cancel a
run.

## Move from evidence to remediation

After a completed execution, select stable finding IDs and launch at most one
remediation job for the scenario:

```bash
test-genie remediate my-scenario \
  --execution <execution-id> \
  --findings <stable-finding-id,...> \
  --role code.default
```

Test Genie sends an immutable evidence packet to Agent Manager. Agent completion
is provisional. Verification is a new server-owned Test Genie execution with a
stable-ID finding delta; it is the only evidence that can mark selected findings
as resolved.

## HTTP clients

Use `POST /api/v1/executions` to create an execution and the remediation routes
under `/api/v1/scenarios/{scenario}/remediation/` to inspect evidence and create
or verify jobs. See [API Endpoints](../reference/api-endpoints.md) for payloads.
