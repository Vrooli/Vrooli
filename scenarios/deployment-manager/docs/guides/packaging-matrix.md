# Packaging matrix

This matrix describes the relationship between governance and target ramps. It
does not grant support to a scenario. Support is determined by the target plan
and evidence for that scenario.

| Ramp | Target | Current status | Owns |
| --- | --- | --- | --- |
| `scenario-to-desktop` | Deployment Tier 2 desktop | Implemented; claims are evidence-gated | Electron generation, packaging, desktop runtime, native journeys, publication handoff |
| `scenario-to-mobile` | Deployment Tier 3 mobile | Not implemented | Reserved target |
| `scenario-to-cloud` | Deployment Tier 4 hosted cloud | Planning/reference | Reserved target |
| `scenario-to-plugin` | Agent runtimes (Agent Plugins, skill registries) | Documented; not implemented | Plugin composition, skill/MCP conformance, supply-chain attestation, clean-room install rehearsal, channel distribution and revocation |
| Enterprise/appliance ramp | Deployment Tier 5 enterprise | Strategic framing | No current packager |

`scenario-to-plugin` is the one ramp whose target is not a deployment tier: it
delivers a capability to an agent runtime rather than to a machine. It emits
`protocol`-profile evidence rather than `visual`, so a recording is not part of
its release claim. See
[scenario-to-plugin](../scenarios/scenario-to-plugin.md).

`scenario-to-extension` is a registered scenario that presents itself as a
browser-extension ramp but is absent from this matrix and from the shared
delivery spine. Its ramp status is unsettled — it is listed here as a known gap
rather than as a supported target.

## Desktop output shapes

| Shape | Contents | Runtime dependency |
| --- | --- | --- |
| External-server thin client | Electron shell and UI assets | Reachable Tier 1 scenario API |
| Bundled private runtime | UI, scenario services, runtime supervisor, and selected verified resource artifacts | No server for local capabilities; remote dependencies remain explicit |
| Shared provider | Desktop application plus broker-issued provider binding | Running Tier 1 or authenticated provider |
| Tier 2 peer | Not currently supported | A future authenticated peer protocol |

The desktop ramp must not promise a bundled API, offline mode, or secret
bootstrap unless the target plan selects those capabilities and the native
journey proves them.

## Common ramp contract

Every target ramp should eventually:

1. consume an immutable target plan;
2. apply the declared dependency and secret strategies;
3. produce target-specific artifacts and metadata;
4. run target-native validation;
5. emit redacted evidence tied to the source and artifacts;
6. ask deployment-manager for the release decision before publication.

For desktop details, use the [scenario-to-desktop overview](../../../scenario-to-desktop/docs/OVERVIEW.md),
[desktop workflow](../workflows/desktop-deployment.md), and [evidence contract](evidence-contract.md).
