# Integrations — Network Manager

## Purpose Of This Document

This document records Network Manager's external resources, scenario dependencies, third-party systems, failure modes, and cross-platform integration contracts.

## Dependency Inventory

| Dependency | Type | Phase | Required | Purpose |
|---|---|---|---|---|
| AdGuard Home | Resolver resource/adapter | P0 | Yes for managed filtering | First DNS filtering backend. |
| Home Automation | Scenario consumer | P0 | Integration target | Surfaces actions/events in home controls. |
| OS host adapters | Platform adapters | P0 | Yes | Read host facts and supported local actions. |
| Manual router instructions | Manual adapter | P0 | Yes | Safe fallback for unsupported routers. |
| Pi-hole | Resolver adapter | P1 | Optional | Focused DNS sinkhole backend. |
| First router adapter | Router adapter | P1 | Optional | Router reads/writes for one selected platform. |
| Technitium DNS | Resolver adapter | P2 | Optional | Advanced DNS platform backend. |

## Vrooli Resources

AdGuard Home is now modeled as the first managed resolver resource:

- Resource ID: `adguard-home`
- Resource CLI: `resource-adguard-home`
- Default admin/API export: `ADGUARD_HOME_BASE_URL`
- DNS bind export: `ADGUARD_HOME_DNS_BIND_IP`
- Credential reference export: `ADGUARD_HOME_CREDENTIAL_REF`
- Network Manager dependency: optional, ignored by default until an operator enables and bootstraps the resource.

The scenario must not hand-run package managers or raw Docker commands for
resolver installation. Resource setup follows Vrooli resource governance, and
Network Manager stores only the AdGuard credential reference. Live filtering is
claimed only after AdGuard confirms protection/filtering state; network-wide
enforcement additionally requires client/router DNS evidence.

Network Manager resolves the referenced AdGuard credential through
`resource-vault content get` when checking resolver health or previewing
upstreams. Persistent resolver/policy writes still require the approval and
rollback-backed policy adapter; the resolver client returns fail-closed instead
of applying direct upstream changes.

The resource binds DNS to the server LAN IP rather than `0.0.0.0:53`, preserving
the host's `systemd-resolved` loopback stub while exposing AdGuard to LAN
clients. Operators must reserve that server IP in the router before advertising
it as the DHCP/RDNSS DNS target.

## Scenario Dependencies

Home Automation consumes Network Manager rather than owning network state. Expected actions/events:

- `network.health.run`
- `network.adblock.pause_device`
- `network.policy.apply_profile`
- `network.outage.detected`
- `network.device.new_seen`
- `network.quality.degraded`

The older `network-tools` live scenario has been retired. Network Manager should not depend on it; use git history only for future archaeology if needed.

## Third-Party Services

No hosted third-party service is required for P0. Public speed-test or measurement endpoints may be used only after dependency and privacy review. ISP portals and paid account management are out of scope.

## Failure Modes

| Failure | Required Behavior |
|---|---|
| Resolver unreachable | Report degraded state; do not claim policy applied. |
| Adapter unsupported | Show unsupported reason and manual instructions when possible. |
| Router write unavailable | Keep P0 read-only/manual; do not fake automation. |
| DNS change breaks connectivity | Preserve rollback handle and surface recovery steps. |
| Query metadata too sensitive | Respect retention and visibility gates. |
| Home Automation unavailable | Keep Network Manager functional and queue or skip consumer events. |

## Cross-References

- [`../reference/configuration.md`](../reference/configuration.md)
- [`ARCHITECTURE.md`](ARCHITECTURE.md)
- [`DOMAINS.md`](DOMAINS.md)
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md)
