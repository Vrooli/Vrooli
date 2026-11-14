# Runtime CPU diagnostics

The control plane injects `VROOLI_RUNTIME_DIAGNOSTICS_DIR` and the existing
scenario, variant, runtime-instance and component-generation identity. Standard
`api-core/server.Run` servers then listen on a private Unix socket. An absent
directory setting disables instrumentation. A diagnostic setup failure is logged
and leaves the application available. Custom servers are not automatically mounted.

Collect from a rebuilt, normally restarted scenario:

```bash
vrooli scenario profile web-console --seconds 20 --out /tmp/web-console.cpu.pprof --json
go tool pprof -top /tmp/web-console.cpu.pprof
```

The default step is `start-api`; `--step` selects another exact component.
`name@variant` and `--instance` use the normal instance parser. Remote node
collection is explicitly unsupported. No raw PID selector is available.

The control-plane owner (`internal/runtimeprofile`) resolves the directory under
the canonical runtime-home state entry. It pins the recorded OS incarnation and
component generation, verifies the socket's launch identity, collects, and checks
the process again before publishing a profile. A collection is bounded to 1–30
seconds and 16 MiB, permits one CPU profiler per process, and cancels on client
disconnect or server shutdown. Output files have mode 0600 and publication refuses
overwrite, including a competing destination created during collection.

The directory must be owned by the current user with no group/other access.
Sockets have mode 0600; launch-derived names avoid stale sockets addressing a
reused PID. This boundary trusts processes running as the same OS user. There is
no TCP diagnostic listener, public application route, arbitrary file reader,
heap dump, goroutine dump, or privilege-broker action. Profiling needs no sudo or
kernel permission changes. CPU profiles contain symbol and source-path metadata;
handle the resulting artifact as operator data.

Linux collection is covered by a native child-process integration test. The
private-socket implementation also supports Unix permission checks on macOS and
BSD; other platforms report unsupported when enabled. Their end-to-end process
identity capabilities must be qualified separately. Old binaries require a
normal rebuild/restart before this optional diagnostic capability is available.
