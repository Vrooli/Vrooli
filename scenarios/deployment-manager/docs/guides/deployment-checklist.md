# Deployment admission checklist

Use this checklist before describing a scenario as ready for a target. An
unchecked item is a named gap, not permission to soften the claim.

## Scenario declaration

- [ ] `.vrooli/service.json` lists every scenario and resource dependency.
- [ ] The target profile identifies supported OS and architecture combinations.
- [ ] Host requirements, privilege, bundling policy, and network needs are explicit.
- [ ] Target-specific limitations and functional differences are documented.
- [ ] Required user journeys exist in `bas/` or the target evidence plan.

## Dependency and secret plan

- [ ] The dependency analyzer report covers the complete graph.
- [ ] Every required dependency has an eligible target route or an explicit blocker.
- [ ] Swaps record migrations, limitations, and ownership.
- [ ] Infrastructure secrets are excluded from bundles.
- [ ] Generated, user-provided, and remote-fetched credentials have valid strategies.
- [ ] The credential authority and recovery behavior are documented.

## Artifact and runtime plan

- [ ] The target plan pins artifacts by platform, architecture, version, and checksum.
- [ ] Ports, health checks, readiness, data directories, and migrations are defined.
- [ ] The runtime ownership boundary is explicit for private, remote, and shared providers.
- [ ] Install, update, restart, rollback, and uninstall behavior are defined.

## Evidence and release

- [ ] Build output is bound to an exact source revision.
- [ ] Native launch, semantic interaction, dependency operation, and clean shutdown are proven.
- [ ] Communication evidence identifies the route and provider without exposing secrets.
- [ ] Unsupported and unavailable states remain terminal verdicts.
- [ ] Artifact trust, release-manifest signing, and OS signing status are clear.
- [ ] deployment-manager has recorded the approval or rejection decision.

## References

- [Deployment Hub](../../../../docs/deployment/README.md)
- [Desktop workflow](../workflows/desktop-deployment.md)
- [Desktop evidence contract](../../../../docs/reference/scenario-to-desktop-evidence-and-tier-contract.md)
- [Credential configuration](../../../../docs/configuration/secrets.md)
