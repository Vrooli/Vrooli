# Scenario reference: scenario-to-desktop

`scenario-to-desktop` is the deployment Tier 2 ramp. It owns Electron
generation, desktop packaging, target execution, smoke journeys, and the
publication handoff. deployment-manager owns the profile, target decision,
release gate, and release record.

## Current route

The ramp supports two primary shapes:

- **Bundled private runtime** for eligible target plans. The desktop supervisor
  starts verified private artifacts and owns only their lifecycle.
- **External-server thin client** for a configured Tier 1 scenario API. Tier 1
  owns the server resources, credentials, data, and lifecycle.

Shared resource reuse is broker- and lease-controlled. Tier 2 peer
communication is unsupported; a resolver candidate is not a peer protocol.

## Integration contract

The ramp calls deployment-manager for target planning, bundle-manifest inputs,
approval, and release-gate decisions. It reports exact source, target,
artifact, and evidence identity back to the governance plane.

The ramp must not:

- bypass the target dependency plan;
- embed infrastructure credentials;
- publish before the release gate permits it;
- describe compile or package success as native runtime support;
- treat a remote bridge job as local desktop evidence.

Read the [scenario-to-desktop documentation](../../../scenario-to-desktop/docs/OVERVIEW.md)
for implementation details and the [desktop evidence contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md)
for release claims.
