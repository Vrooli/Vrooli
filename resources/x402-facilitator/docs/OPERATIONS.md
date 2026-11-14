# Operations

`x402-facilitator` is a loopback-only managed service. The control plane owns
digest-pinned OCI acquisition, native process lifecycle, port publication, and
health checks. The OCI filesystem is extracted over HTTPS; the service does not
require a Docker daemon on Linux.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative lifecycle, acquisition digest, runtime, port, health, and export metadata.
- `cli/` owns the binary entrypoint, wiring, and delegated command registration.
- `cli/internal/` owns resource-specific Go logic that cannot be expressed through the manifest or shared control-plane packages.

Do not turn `cli/main.go` into the primary implementation surface. If the resource needs specialized install/runtime/status/health/env logic, grow the matching package under `cli/internal/` first.

## Safe bootstrap

The checked-in `config/config.json` intentionally contains empty `chains` and
`schemes` collections and no secret material. The upstream service therefore
reports healthy with an empty supported-network set and cannot verify or settle
value. Do not place signer material in `runtime.env`, a shell script, a compose
file, or source control. Automated-rail work must first add a Credential
Authority-backed injection path and prove it never appears in argv, logs,
status JSON, or repository files.

The pinned upstream `1.3.0` configuration format puts the signer reference in
the chain's `signers` list (for example, `$FACILITATOR_PRIVATE_KEY`) and
resolves that reference from the process environment. The resource now declares
`vrooli/x402-facilitator:facilitator-private-key`; the control plane resolves it
from Credential Authority and injects it only into the managed child process.
It is not placed in `runtime.env`, arguments, supervisor state/status metadata,
or source configuration. The value is still readable through
`/proc/<pid>/environ` to processes running as the same user, as with any
environment-injected third-party service secret. Upstream logs have not been
verified with a signer configured. Do not use this path for a funded or
value-bearing signer.

The checked-in config remains empty (`chains: {}`, `schemes: []`), so this
declaration alone does not enable payment networks. The shared facilitator is
already running in that empty-chain state; preserve it and do not restart or
edit its live configuration. A real signer exercise needs a separate
lifecycle-owned facilitator resource, a loopback Base fork, and a generated
test-only key. The CF4 in-process cryptographic harness is not evidence that
this upstream process settled a transaction.

Platform support is Linux only. macOS and Windows remain explicitly unsupported
until upstream publishes signed native facilitator artifacts; a running Linux
resource does not change those platform declarations.

## Operator checks

```text
vrooli resource validate x402-facilitator
vrooli resource install x402-facilitator
vrooli resource start x402-facilitator
vrooli resource status x402-facilitator --json
curl --fail --silent http://127.0.0.1:14020/health
curl --fail --silent http://127.0.0.1:14020/supported
```

Healthy means the process and protocol discovery surface are reachable. It
does not mean a payment network is configured. Production readiness additionally
requires an attended signer setup, explicit network allowlist, funded gas
account, rate limiting, alerting, testnet settlement proof, and reconciliation
of an intentionally induced unknown outcome.

## Upgrade and rollback

Pin OCI indexes by digest and the resulting materialized tree by SHA-256. Before changing the digest, review the upstream diff
from image revision `e75adda8a0e3cf23db446883ba91b88bbab2fe28`, verify provenance,
and exercise the protocol on testnet. Rollback is the inverse digest change plus
`vrooli resource restart x402-facilitator`; never reuse a floating tag.
