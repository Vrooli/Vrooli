# Security — Program Runtime

Program Runtime is a trusted-local-agent capability. Its callers are local
agents operating inside the Vrooli installation; the kernel is not a hostile
multi-tenant sandbox. The API owns authorization and the kernel supervisor
owns process lifetime, but host isolation remains deliberately bounded.

## Enforced controls

- Kernel processes receive an allowlist of environment and Runtime protocol
  variables. API credentials and unrelated host variables are not inherited.
- Each session receives a mode-0700 working directory beneath the scenario
  data root. The directory is removed when its kernel is killed or reclaimed.
- Linux applies address-space, CPU-time, file-size, and open-file limits.
  Other platforms retain the portable wall-clock and lifecycle controls until
  native resource-limit adapters are added.
- Linux kernels run in their own process group. Deadline and reclamation
  cleanup kills the group, waits for the process, removes the protocol file,
  and forgets the live handle.
- On non-Linux platforms, deadline and reclamation cleanup kills and waits for
  the direct kernel child. Descendant processes are not guaranteed to stop and
  may outlive the kernel; this is not process-tree isolation.
- Every submission has a 120-second supervisor deadline. A timeout is stored
  as `deadline_exceeded` and explicitly states that live session variables were
  lost when the kernel restarted.
- Destructive bindings require session grants and confirmation. Refusals are
  retained as durable evidence.
- Verified agent identity and operation bearer proofs are not passed into the
  Python kernel. Submitted code can import kernel modules, so a bearer stored
  there can be printed or otherwise disclosed. For a verified declared program,
  Linux supports four checkpoint actions, agent start and collect, and two Web Console native reads over a sealed one-shot owner relay;
  the recipient reverifies the identity on every call. Other proof-dependent
  kernel operations fail closed. Direct supported owner operations retain their
  own policy.

## Residual risks and revisit triggers

### Authenticated calls from persistent sessions

Ordinary sessions intentionally keep mutable Python state across submissions. Code
from one submission can leave a thread in the same process for a later
submission. A local relay that identifies only the kernel process cannot
distinguish those threads or safely bind a later caller's authority to just its
own code. Retained namespace objects can also run their earlier-defined
finalizers on a later submission's current thread, so identifying that thread
alone is insufficient. The actual-subprocess regression is
`kernel/tests/test_agent_forwarding.py::test_retained_finalizer_executes_in_later_submission_thread`.
A random bearer supplied to Python would recreate the disclosure
problem. Declared programs instead use a sealed session that admits one
execution, with a PID-checked Unix relay that is closed at execution end; the
Go owner retains the proof and the recipient verifies it again. This relay is
limited to task resolve, begin, start and complete, agent start and collect,
and the Web Console native thread-state and usage read bindings; it requires Linux peer PID and pidfd
support. The binding bridge reverifies identity and applies normal binding and
Web Console authorization. Agent calls also carry a verified supplemental operation
proof when available; the receiving owner enforces its own grants. An unconfirmed
effect requires owner-state reconciliation before retry. On unsupported or degraded hosts, checkpoints and delegation remain
unavailable and native reads may fail authentication; unrelated reads may proceed. A persistent session cannot use
the relay: the original expected-success checkpoint test tracks that regression.
The Python process remains trusted-local execution, not a hostile-code sandbox.

### Workspace composition gap

The session binding is enforced through a discovery-backed REST resolver for
workspace-sandbox. The dependency does not yet publish a shared typed API for
resolving an identifier to a root path, so the adapter validates the returned
absolute directory before pinning it as the kernel cwd. If workspace-sandbox is
unavailable, a session with no declared workspace continues in its private
scratch directory; a declared identifier is rejected. An explicit absolute
path is accepted only after local `EvalSymlinks` and directory validation, and
does not claim copy-on-write isolation. This reduced guarantee is tracked in
the workspace-sandbox QA intake. Revisit when workspace-sandbox publishes a
typed resolver or when execution becomes untrusted or multi-tenant.

| Risk | Why accepted now | Revisit trigger |
|---|---|---|
| Python can import `subprocess`, open sockets, and reach host resources from the local process. | The posture is trusted-local-agent, and typed binding governance still controls the advertised Vrooli surface. A full container/VM sandbox would add a separate deployment contract. | Any untrusted-user, hosted, or cross-tenant caller enters scope; then require workspace-sandbox plus OS/container isolation before enabling execution. |
| Non-Linux hosts do not yet have native RLIMIT equivalents or process-group cleanup. | The wall-clock supervisor kills and waits for the direct kernel child; descendants may survive. The environment allowlist and pinned directory still apply. | A supported deployment requires stronger memory/CPU/file enforcement and native process-tree cleanup on macOS or Windows. |
| The API database is local SQLite and contains authored source and failure evidence. | Storage is scenario-owned, local, and declared non-regenerable with bounded retention. | External users or regulated data enter the corpus; add authentication, encryption, and a privacy-specific retention review. |

## Cross-references

- [`DATA.md`](../concepts/DATA.md) — durable data and retention
- [`INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external capability contracts
- [`Environment management`](../../../../docs/reference/environment-management.md) — runtime profiles
