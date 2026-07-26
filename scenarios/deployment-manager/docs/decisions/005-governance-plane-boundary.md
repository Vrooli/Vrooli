# ADR-005: Deployment Manager Is The Governance Plane

## Status

Accepted

## Context

Deployment work is spread across several scenarios, and the boundary between them was never written down. The result is visible in the code:

- `api/codesigning` implements signing that `scenario-to-desktop` also implements, and carries a deprecation notice pointing at it
- `api/validation` implements screen-recorded validation that `vrooli-emulator` and `scenario-to-desktop` also implement
- `POST /api/v1/deploy/{profile_id}` implies deployment-manager orchestrates packaging, which it does not do

Each of these is the same mistake: deployment-manager acquiring target-specific capability that belongs to a ramp or to the evidence substrate. Without a stated boundary, the next feature repeats it.

Meanwhile the integration that does work — `scenario-to-desktop` calling deployment-manager for approvals, the release gate, and bundle manifests — was never documented as the intended shape, so it reads as one integration rather than as the contract every ramp should implement.

## Decision

Deployment work is divided into four planes. A component belongs to exactly one.

| Plane | Component | Job |
|-------|-----------|-----|
| Governance | `deployment-manager` | Decide whether a build may ship; record what shipped |
| Ramp | `scenario-to-*` | Build, package, sign, publish, and exercise for one target family |
| Evidence | `browser-automation-studio`, `workflow-health`, `vrooli-emulator`, `test-genie` | Produce the evidence a gate decides on |
| Reach | `vrooli-bridge` | Run flows on machines and operating systems that are not the local host |

Three rules follow.

**deployment-manager is tier-agnostic.** It holds profiles, fitness scores, approval gates, release records, channels, and evidence references. It does not learn artifact formats, platform toolchains, or signing mechanics. A change that would teach it one belongs in a ramp.

**Ramps call deployment-manager.** A ramp asks for a gate decision and publishes only when the gate allows it. deployment-manager does not drive a ramp's pipeline. The three endpoints `scenario-to-desktop` already calls are the contract every ramp implements.

**deployment-manager stores evidence references, never artifact bytes.** Recordings and screenshots stay with the producer. deployment-manager holds the reference and the verdict.

A generic deploy verb in deployment-manager is compatible with this decision only as a dispatcher: it resolves the profile's tier to a ramp and delegates. It must never implement packaging.

## Consequences

- `api/codesigning` becomes a proxy to `scenario-to-desktop` and its local implementation is removed
- `api/validation` is retired in favor of `vrooli-emulator`; the bridge from review decision to approval record stays in deployment-manager
- The generic deploy endpoints refuse until the dispatcher exists, rather than returning synthesized success
- Adding a ramp requires no deployment-manager change beyond a tier-to-ramp mapping entry
- Cross-tier evidence comparison becomes possible once every ramp emits one evidence shape, which is a separate decision

## References

- [Deployment Hub](../../../../docs/deployment/README.md)
- [SEAMS.md](../SEAMS.md)
