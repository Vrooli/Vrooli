# Security — Program Runtime

Program Runtime is a trusted-local-agent capability. Its callers are local
agents operating inside the Vrooli installation; the kernel is not a hostile
multi-tenant sandbox. The API owns authorization and the kernel supervisor
owns process lifetime, but host isolation remains deliberately bounded.

## Enforced controls

- Kernel processes receive only `PATH`, `HOME`, `LANG`, `PYTHONPATH`, and the
  four `PROGRAM_RUNTIME_*` protocol variables. API credentials and unrelated
  host environment variables are not inherited.
- Each session receives a mode-0700 working directory beneath the scenario
  data root. The directory is removed when its kernel is killed or reclaimed.
- Linux applies address-space, CPU-time, file-size, and open-file limits.
  Other platforms retain the portable wall-clock and lifecycle controls until
  native resource-limit adapters are added.
- Linux kernels run in their own process group. Deadline and reclamation
  cleanup kills the group, waits for the process, removes the protocol file,
  and forgets the live handle.
- Every submission has a 120-second supervisor deadline. A timeout is stored
  as `deadline_exceeded` and explicitly states that live session variables were
  lost when the kernel restarted.
- Destructive bindings require session grants and confirmation. Refusals are
  retained as durable evidence.

## Residual risks and revisit triggers

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
| Non-Linux hosts do not yet have native RLIMIT equivalents. | The wall-clock supervisor, environment allowlist, pinned directory, and process lifecycle still apply cross-platform. | A supported deployment requires stronger memory/CPU/file enforcement on macOS or Windows. |
| The API database is local SQLite and contains authored source and failure evidence. | Storage is scenario-owned, local, and declared non-regenerable with bounded retention. | External users or regulated data enter the corpus; add authentication, encryption, and a privacy-specific retention review. |

## Cross-references

- [`DATA.md`](../concepts/DATA.md) — durable data and retention
- [`INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external capability contracts
- [`Environment management`](../../../../docs/reference/environment-management.md) — runtime profiles
