# k6 Resource

Grafana k6 CLI resource for load and performance testing workflows.

## Intent

- Resource ID: `k6`
- Category: `development`
- Driver: `external-cli`
- Portability tier: `full`

## Use Cases

- Run load and smoke tests against scenario APIs before deployment.
- Benchmark latency and throughput changes across revisions.
- Add performance validation to local development and CI workflows.

## Architecture

This resource now follows the `external-cli` template structure.

- `resource.json` is the declarative authority for install, binary probing, version checks, health checks, freshness, and lifecycle metadata.
- `cli/` is the single binary entrypoint and command wiring surface.
- `cli/internal/` is the default home for k6-specific Go logic when the manifest and shared control plane are not enough.

The intended escalation path is:

1. express behavior in `resource.json`
2. rely on the shared `vrooli resource ...` control plane
3. add resource-specific Go code under `cli/internal/...` only where specialization is real
4. add custom CLI commands only when the resource truly needs operator actions beyond the standard lifecycle surface

Current internal package boundaries:

- `cli/internal/discovery`: host binary detection and probing helpers
- `cli/internal/install`: install/bootstrap helpers
- `cli/internal/version`: version parsing and compatibility helpers
- `cli/internal/env`: environment and test-path helpers
- `cli/internal/auth`: optional cloud/provider auth validation helpers

## Usage

```bash
# Install using the declarative platform contract
vrooli resource install k6

# Check that the binary is available and healthy
resource-k6 status
```

## Notes

- `k6` is an external CLI resource, not a local daemon owned by this resource.
- Keep binary/version/install behavior declarative in `resource.json` whenever possible.
- Keep `cli/main.go` thin; do not treat it as the implementation surface for k6-specific behavior.
- If the resource needs specialized detection, version parsing, or cloud auth validation, add that logic under `cli/internal/...` rather than shell wrappers.

## References

- [k6 Documentation](https://k6.io/docs/)
- [JavaScript API](https://k6.io/docs/javascript-api/)
- [Grafana Cloud](https://grafana.com/products/cloud/)
## Maturity

M4 (2026-08-05): lifecycle, health, platform gates, and Go CLI test evidence are covered by the fleet contract.
