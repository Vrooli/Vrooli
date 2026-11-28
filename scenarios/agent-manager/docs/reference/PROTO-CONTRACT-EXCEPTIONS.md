# Proto Contract Exceptions

This register records the remaining Agent Manager V1 contract layout findings that cannot be corrected without changing public protobuf identities.

The machine-readable companion [`proto-health-exceptions.json`](../../.vrooli/proto-health-exceptions.json) permits only the three exact `proto.shared_type_misplaced` findings listed below. It does not waive any future shared-type debt.

| Finding | Scope | Decision | Exit criterion |
| --- | --- | --- | --- |
| `proto.shared_type_misplaced` | `RunEvent`, `ExecutionPolicySnapshot`, and `Run` | Preserve established V1 names and generated-client imports. Moving a message into `v1/shared` changes its full protobuf name. | V2 migration with consumer inventory, compatibility plan, regenerated clients, and cross-scenario validation. |

## Stability policy

`AgentManagerService` remains an experimental compatibility Connect service. A method is available over Connect only when the currently registered `AgentManagerConnectHandler` explicitly implements it; generated service descriptors and native CLI commands do not imply Connect support. The existing REST and native CLI owner paths remain supported independently and are not evidence that an unimplemented Connect RPC is available.
