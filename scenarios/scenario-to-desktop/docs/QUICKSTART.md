# Desktop quickstart

Use the bundled path for a self-contained application. Use the thin-client path
when a Tier 1 server remains the source of truth.

## Bundled application

1. Start the deployment services through the lifecycle manager:

   ```bash
   vrooli scenario start deployment-manager
   vrooli scenario start scenario-to-desktop
   ```

2. Create a deployment-manager profile:

   ```bash
   deployment-manager profile create my-profile my-scenario --tier 2
   ```

3. Review dependency fitness and target limitations:

   ```bash
   deployment-manager analyze my-scenario
   deployment-manager fitness my-scenario --tier 2
   ```

4. Run the desktop pipeline:

   ```bash
   deployment-manager deploy-desktop \
     --profile my-profile \
     --platforms linux \
     --timeout 20m
   ```

   Add the other target platforms only when their artifacts and native
   validation environment are available.

5. Inspect the generated artifact, runtime plan, and evidence before treating
   the result as a release. Package creation is not proof of native runtime
   behavior.

The bundled path requires a target-compatible dependency plan. A required
resource that is unsupported for the target must remain visible as a named
limitation; it must not be silently omitted.

## Thin client

1. Start the target scenario through the Tier 1 lifecycle.
2. Start `scenario-to-desktop`:

   ```bash
   vrooli scenario start scenario-to-desktop
   ```

3. In the scenario-to-desktop UI, select `external-server` and configure the
   target scenario API URL. Use a LAN or explicitly managed app-monitor route.
4. Generate the wrapper and platform package.
5. Validate the route and real scenario interaction before distribution.

The thin client contains the desktop shell and UI assets. It does not contain
the Tier 1 API or resources and cannot operate offline.

## Focused commands

```bash
# Start the scenario-owned service
make start

# Run the server-owned scenario suite
vrooli scenario test scenario-to-desktop

# Inspect the generated pipeline
scenario-to-desktop pipeline active --help
scenario-to-desktop pipeline list
```

Use the [build guide](guides/build-and-packaging.md) for platform prerequisites,
the [smoke-test reference](reference/smoke-test-pipeline.md) for evidence, and
the [Deployment Hub](../../../docs/deployment/README.md) for ownership and
target maturity.
