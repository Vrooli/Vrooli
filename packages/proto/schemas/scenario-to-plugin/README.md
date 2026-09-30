# Scenario-to-plugin proto sources

This is the canonical wire contract for the Agent Plugin delivery ramp. The
six domain packages are declaration, composition, conformance, attestation,
rehearsal, and distribution; every RPC comment traces to a requirement id in
`scenarios/scenario-to-plugin/requirements/`.

Shared health and error envelopes are conventional runtime contracts. The
domain packages are scenario-owned contracts, not template example domains.

After changing these sources, regenerate bindings with:

```bash
cd packages/proto
make generate SCENARIO=scenario-to-plugin
```

Generated Go, TypeScript, and Python bindings under `packages/proto/gen/` are
compiled and validated by the scenario API and CLI tests.
