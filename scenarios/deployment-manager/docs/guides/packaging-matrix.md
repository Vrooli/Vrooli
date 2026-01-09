# Packaging Matrix (`scenario-to-*` Capabilities)

This matrix captures what the existing scenario-to-* scenarios can produce today and what upgrades they need.

| Packager | Current Output | Works For | Missing Pieces |
|----------|----------------|-----------|----------------|
| `scenario-to-desktop` | Electron thin client that bundles UI assets and lightweight menus | Tier 2 demos when a Tier 1 server is reachable | Bundled API/resource processes, secrets bootstrap, offline mode, installer/updater |
| `scenario-to-mobile` | ❌ Not implemented | — | Entire scenario (spec, UI framework, packager) |
| `scenario-to-cloud` (future) | ❌ Not implemented | — | Needs integration with deployment-manager + provider modules |

## Template for Future Packagers

Each packager should:

1. Consume the dependency bundle manifest produced by deployment-manager.
2. Install or embed each dependency according to the manifest (binary copy, container, managed service config).
3. Apply secret strategies (generate, prompt, reference placeholders).
4. Produce artifacts + documentation (`dist/<scenario>/<tier>/`).
5. Emit telemetry back to deployment-manager (success/failure, size, resource usage).

## Action Items

- Update `scenario-to-desktop` README with current limitations (see [Scenario Docs](../scenarios/scenario-to-desktop.md)).
- Spec `scenario-to-mobile` and `scenario-to-cloud` scenarios.
- Link each packager to automated Tier 2/3/4 tests once they exist.
