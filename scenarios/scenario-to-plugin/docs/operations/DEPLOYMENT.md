# Deployment — Scenario to Plugin

This document records how this scenario is deployed and what must be true
before a release. It covers deploying *the ramp*; the artifacts the ramp
publishes are covered in [`../concepts/FLOWS.md`](../concepts/FLOWS.md).

## Purpose Of This Document

Use this document to answer:

- Where does this scenario run, and where does it deliberately not?
- What does the host need before it can do its job?
- What is checked before a release, and in what order?
- How is a bad release backed out?

## Supported Tiers

| Tier | Supported | Reason |
|---|---|---|
| Tier 1 — local Vrooli control plane | **Yes.** The only supported tier today. | The ramp needs a local scenario tree to read declarations from, a local `cli-manifest` source, and a sandbox host. |
| Tier 2 — generated desktop application | No. | An operator publishing signed artifacts to a public registry from a bundled desktop app would move signing credentials onto an end-user machine. |
| Tier 3 — hosted | Deferred, and plausible. | CI is the natural long-run home for a publishing pipeline. Blocked on signing-identity delegation and on rehearsal isolation in a hosted sandbox. |
| Tier 4 — appliance | No. | No audience. |

The Tier 3 deferral is worth revisiting once one package has shipped by
hand: a publishing ramp that runs only on a developer workstation is a
bus-factor problem, not a design intent.

## Runtime Requirements

| Requirement | Why | Failure Behavior |
|---|---|---|
| Local scenario tree | Declarations and `cli-manifest` sources are read from it. | Composition refuses; readiness reports the scenario as unresolvable. |
| Writable capture-store directory | Artifact trees, attestations, and rehearsal logs are digest-addressed files. | Composition fails closed; no partial artifact is recorded as complete. |
| `workspace-sandbox` reachable | Rehearsal isolation. | Rehearsal is refused. The ramp never falls back to a host-local install. |
| `deployment-manager` reachable | Release decisions and verdict ingest. | Publication and revocation are refused; everything upstream still runs. |
| `cli-health` reachable | Pinned command surface for the drift gate. | Drift gate fails closed. |
| Managed release-signing authority | Cosign keyless signing. | Attestation fails; no unsigned artifact ever advances. |
| Outbound HTTPS | Scanners, Sigstore, and registry pushes. | Attestation and publication fail; composition and conformance are unaffected because they never touch the network. |
| Container runtime | Scanner invocation. | Attestation fails closed rather than recording an unscanned pass. |

Note what is *not* required: no Postgres, no Redis, no model runtime, and
no elevated privilege. The ramp runs entirely as the invoking user, and
`vrooli setup` remains the only place privilege is ever requested.

## Packaging

Standard scenario lifecycle packaging. The API, CLI, and UI are built and
started through the lifecycle; nothing here is run directly.

```bash
make setup     # one-time
make start     # start API + UI
make status    # lifecycle metadata
make test      # scenario suite
make stop
```

The CLI is installed on `PATH` by the lifecycle. Do not invoke built
binaries directly — that bypasses process naming, port allocation, and
health checks.

## Release Checklist

Run in order. Each step gates the next; a failure stops the release rather
than downgrading it to a warning.

1. `make test` passes for the scenario suite.
2. `vrooli scenario requirements validate scenario-to-plugin --json`
   reports `PASSED` with zero findings.
3. `experience-manager spec validate scenario-to-plugin --json` passes.
4. `make orient` reports every initialization gate closed.
5. `docs/manifest.json` maturity values match the real state of each
   document. A doc marked `active` that is still a stub is a release
   blocker, because downstream tooling reads maturity as truth.
6. `.vrooli/service.json` dependency declarations agree with
   [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md).
7. **Self-test the gates.** Run the ramp against a deliberately broken
   fixture package and confirm each gate closes: a skill documenting a
   removed command must fail conformance; an unpinned install must fail;
   a package with a credential literal must fail before any network call.
   A publishing pipeline whose refusals are untested is worse than none.
8. Confirm no credential literal appears in any emitted artifact, SBOM,
   attestation, or log for the fixture run.
9. Record the release in [`../internal/PROGRESS.md`](../internal/PROGRESS.md)
   with validation evidence.

Step 7 is the one that distinguishes a real release of this scenario from
a green build. Every other scenario proves what it does; this one must
also prove what it refuses.

## Rollback

- **The ramp itself** rolls back like any scenario: stop, restore the
  previous build, restart through the lifecycle. State is SQLite plus the
  capture store; both are additive and a downgrade reads older rows fine.
- **A published artifact does not roll back.** It is on machines this
  scenario does not control. The remedy is revocation
  (`PLG-DIST-REVOKE`), which withdraws or flags the artifact in every
  channel that received it and records per-channel outcomes.
- **A partial revocation is a real outcome.** Some registries cannot
  hard-delete a version. `revoked_partial` names the channels still
  carrying the artifact so an operator can escalate to that registry's
  process. Do not treat it as a transient state to retry away.
- **Never delete a `publications` row as part of a rollback.** The
  revocation fan-out is derived from it; deleting it erases the ability to
  withdraw the artifact at all.

## Cross-References

- [`RUNBOOK.md`](RUNBOOK.md) — start/stop, incidents, escalation
- [`OBSERVABILITY.md`](OBSERVABILITY.md) — signals and alerts
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — dependency contracts and failure modes
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — credential handling and threat model
- [`../reference/configuration.md`](../reference/configuration.md) — environment variables
