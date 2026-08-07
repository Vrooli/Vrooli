# Scenario-to-desktop seams

This is the current implementation map for variation points in
scenario-to-desktop. It is intentionally smaller than a change log: behavior
contracts belong in the public documents linked below, while this page tells
contributors where to place code and tests.

## Ownership map

| Area | Production seam | Responsibility |
|---|---|---|
| Generation | `api/generation/` | Analyze a scenario and generate an Electron wrapper. |
| Build | `api/build/` | Compile, package, and persist desktop build results. |
| Bundling | `api/bundle/` | Resolve, stage, verify, and package the private runtime and service artifacts. |
| Pipeline | `api/pipeline/` | Orchestrate bundle, decision, preflight, generation, build, and smoke stages. |
| Deployment handoff | `api/deploy/` | Call deployment-manager or LPBS with a typed target request. |
| Preflight | `api/preflight/` | Start a staged runtime and verify readiness before packaging. |
| Smoke evidence | `api/smoketest/` and `api/captures/` | Run the desktop journey, persist assertions, and retain capture references. |
| Records | `api/records/` | Persist generated-app and pipeline records. |
| Signing | `api/signing/` | Validate signing configuration and generate signing inputs. It does not own release authority. |
| Telemetry | `api/telemetry/` and `runtime/telemetry/` | Record local lifecycle events and expose redacted operational metadata. |
| Runtime | `runtime/` | Own private service lifecycle, ports, readiness, secrets injection, and control API. |
| Templates | `templates/` | Generate the Electron shell and its browser/runtime adapters. |

The API root contains only cross-cutting adapters, handlers, and wiring. New
domain behavior should live in the owning package above rather than becoming a
new root-level implementation.

## Contract seams

### Deployment mode

`deployment_mode` selects the ownership model. Bundled mode owns a verified
private runtime; `external-server` is a thin client over a configured Tier 1
API; `cloud-api` is a reserved integration mode. The mode contract is in
[deployment modes](../concepts/deployment-modes.md).

### Target decision

The deployment stage consumes deployment-manager's target plan. It must carry
the selected target, resource verdicts, privilege level, bundling disposition,
and provenance into the generated artifact. A fitness score is advisory; the
release gate consumes explicit verdicts and evidence.

### Provider observation

The runtime may report provider tier, safe route class, service identity,
readiness, artifact digest, fallback decision, and lease expiry. It must never
report an endpoint, bearer token, generated operator configuration, or secret
value. The full boundary is in the [desktop evidence and tier contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md).

### Evidence

The smoke journey is a producer-owned timeline. A window launch or recording
alone is not communication proof. Each supported claim needs a machine
assertion, ordered journey data, and reviewer-visible capture references.
Unsupported and unavailable routes remain terminal verdicts.

### Runtime control

The runtime supervisor is the only component that starts or stops private
bundled services. The Electron shell talks to it through the loopback control
API. Shared resources are authorized by their broker; the desktop runtime does
not take over their lifecycle.

## Test seams

Tests should replace external effects at these boundaries:

| Effect | Test boundary |
|---|---|
| Clock and bounded waits | Pipeline and smoke-test clock/waiter interfaces |
| Process execution | Build, runtime, and smoke-test executor interfaces |
| Filesystem and staging | Bundle file-operation and staging interfaces |
| Deployment-manager / LPBS | Typed clients in `api/deploy/` |
| Runtime readiness | Preflight runtime client and service fakes |
| Evidence persistence | Capture store and smoke-test store interfaces |
| Browser/OS APIs | Template browser adapters and UI browser seam |

Tests should assert desired behavior at the seam, including failure and
redaction behavior. Do not test a private implementation detail when the typed
contract can express the expectation.

## Related contracts

- [Overview and current support](../OVERVIEW.md)
- [Architecture](../concepts/ARCHITECTURE.md)
- [Smoke-test pipeline](../reference/smoke-test-pipeline.md)
- [Telemetry](../reference/telemetry.md)
- [Security posture](SECURITY-POSTURE.md)
- [Invariants](INVARIANTS.md)

