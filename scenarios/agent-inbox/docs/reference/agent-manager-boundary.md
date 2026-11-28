# Agent Manager boundary

Agent Inbox uses the generated `agent_manager.v1.AgentManagerService` Connect client for the proto-owned operations below. `agent_manager_connect_test.go` exercises these methods through the generated service handler.

| Agent Inbox operation | Agent Manager procedure |
|---|---|
| Start chat task | `CreateTask` |
| Start chat run | `CreateRun` |
| Continue chat | `ContinueRun` |
| Reconcile or read run status | `GetRun` |
| Read run events | `GetRunEvents` (unary, paginated by sequence) |
| List runs | `ListRuns` |
| Stop run | `StopRun` |

Run creation leaves `run_mode` unset so Agent Manager applies the referenced profile's protected sandbox policy. Callers must not force an in-place override around a protected profile.

`POST /api/v1/attachments/upload` remains a direct multipart request because Agent Manager has no corresponding proto procedure and the route accepts a multipart file body. It is the only Agent Inbox Agent Manager REST boundary. The Agent Manager WebSocket is not used by current callers; the inbox UI polls its own API for events.

Connect requests use the same bounded retry and circuit-breaker policy as the previous client. A retry re-resolves Agent Manager discovery so a service restart that changes its local port is observed.
