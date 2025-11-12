# Workflow Call Node

The Workflow Call node lets a workflow reuse another saved workflow inline. The referenced workflow runs inside the same Browserless session so state (cookies, DOM, variables) carries forward, and the calling workflow can pass scoped parameters or capture outputs.

## Configuration

- **Target workflow** – select from the workflow list or paste a workflow UUID.
- **Wait for completion** – currently always enabled. Asynchronous fire-and-forget calls are not supported yet.
- **Parameters** – optional JSON object of key/value pairs injected into the execution context before the callee starts. Existing variable values are restored after the run completes.
- **Output mapping** – optional JSON object mapping `calleeVariable` → `callerVariable`. Values are copied from the callee’s context after completion and written to the caller’s variables.

## Behavior

1. The runtime loads and compiles the referenced workflow before the Browserless session starts executing the node.
2. Parameters are interpolated (variables such as `{{user}}` are expanded) and temporarily applied to the shared execution context.
3. The referenced workflow is executed inline using the active CDP session, honoring the same branching, loops, and variable semantics as any other workflow.
4. After the nested workflow finishes, output mappings are applied and the temporary parameters are restored.
5. The Workflow Call node emits telemetry/logs like any other step and can optionally store its result via the standard `storeResult` field.

## Limitations

- Asynchronous calls (`waitForCompletion = false`) are not supported yet.
- The callee runs inside the caller’s context; variable names may collide if not scoped via parameters/output mappings.
- Workflow selection currently relies on the local workflow list; cross-environment references are not yet supported.

## Example

```json
{
  "type": "workflowCall",
  "data": {
    "workflowId": "123e4567-e89b-12d3-a456-426614174000",
    "parameters": {
      "username": "{{creds.user}}",
      "password": "{{creds.pass}}"
    },
    "outputMapping": {
      "sessionToken": "auth.token"
    }
  }
}
```

This node runs the shared login workflow, passes the caller’s credentials, and captures the `sessionToken` variable into `auth.token` for downstream steps.
