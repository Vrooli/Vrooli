# Proto Contract Exceptions

This register records the remaining Agent Manager V1 contract layout findings that cannot be corrected without changing public protobuf identities.

The machine-readable companion [`proto-health-exceptions.json`](../../.vrooli/proto-health-exceptions.json) permits only the three exact `proto.shared_type_misplaced` findings listed below. It does not waive any future shared-type debt.

| Finding | Scope | Decision | Exit criterion |
| --- | --- | --- | --- |
| `proto.shared_type_misplaced` | `RunEvent`, `ExecutionPolicySnapshot`, and `Run` | Preserve established V1 names and generated-client imports. Moving a message into `v1/shared` changes its full protobuf name. | V2 migration with consumer inventory, compatibility plan, regenerated clients, and cross-scenario validation. |

## Stability policy

`AgentManagerService` is marked `experimental` until the generated Connect handler is mounted and its RPC surface has an explicit compatibility commitment. The existing REST API remains supported independently; it is not evidence that the generated Connect transport is implemented.
