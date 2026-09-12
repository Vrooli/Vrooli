# Operations

`whisper` is organized as a checksum-verified `managed-service` resource.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, native acquisition, port, export, and health metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns Whisper-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the implementation surface. The native
compatibility and capacity edge belongs in `cli/internal/activityproxy`; model
and server delivery belongs in the managed-service acquisition contract.

## Operator Checklist

- Keep native server arguments, ports, and health checks declared in `resource.json`.
- Keep mutable uploads, models, and cache state in canonical resource storage paths.
- Prefer shared `vrooli resource ...` lifecycle behavior before adding resource-local commands.

## Native macOS Candidate Build

Upstream whisper.cpp v1.9.2 does not provide the native `whisper-server`
executable needed by this resource on macOS. A candidate can be built on a
matching Apple arm64 host from a source tree pinned to the v1.9.2 tag:

```bash
make -C resources/whisper build-native-macos-arm64 \
  WHISPER_CPP_ROOT=/path/to/whisper.cpp-v1.9.2
```

The recipe checks for Darwin/arm64 before invoking CMake, builds the upstream
`whisper-server` target in Release mode, and writes
`resources/whisper/dist/whisper-server_macos_arm64`. It deliberately does not
cross-compile, fetch dependencies, sign the executable, or update
`resource.json`. The candidate is not a supported acquisition artifact until
it has a reproducible release checksum, native-secure-store signing, and a
macOS arm64 managed-service smoke run.
