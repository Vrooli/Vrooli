# Phase A5 — Windows development target — 2026-08-25

## Result

The repository now has named Windows peers for the two capabilities that the
baseline deliberately treated as real work:

| Capability | Implementer | Mechanism | Qualification |
| --- | --- | --- | --- |
| `test-execution` | `powershell-test-runner` | `pwsh Start-Job` | build-verified |
| `terminal-multiplexing` | `windows-terminal` | `wt.exe pane` | build-verified |

The peers are declared as `internal/tools/*/tool.json` manifests. They do not
claim host qualification; a future `windows-latest` run must supply structured
native evidence before either declaration can move above `build-verified`.

## Control decisions

The nine baseline controls are represented in the vocabulary's
`control_policies` and `control_policy_reasons` blocks for macOS and Windows.
The policy is `no_work_required`: the platform-native service, session, driver,
or launcher owns the boundary, so Vrooli must not pretend that a Linux-specific
safeguard handler was ported. The portability grid applies those policies to
the affected cells; an absent unpolicyed control remains `controls_incomplete`.

The policy-covered controls are `docker_host_firewall`,
`login_keyring_unlock`, `tpm_credential_access`, `nvidia_driver`,
`ollama_resource_controls`, `autoheal_recovery_privileges`,
`onboarding_apply_privileges`, `path_hygiene`, and `vrooli_launcher`.

## CI contract

`.github/workflows/test.yml` contains the required `Cross-platform lifecycle`
matrix over `macos-latest` and `windows-latest`. Each matrix leg installs the
pinned Go toolchain, runs the lifecycle/process/scenario CLI contract suites,
starts `hello-plugin` through `go run ./cmd/vrooli scenario start`, checks
`scenario status --json`, and stops it through the control plane. The workflow
does not use a soft-fail or skip path for this lifecycle.

## Evidence and limits

Local evidence:

- `go test ./internal/portability -run 'TestWindowsDevelopmentTargetResolvesPeersAndControlPolicies|TestGridClassifiesEveryCapabilitySituation' -count=1` passed.
- `go test ./internal/runtime -run 'TestEmbeddedToolManifestsValidateAgainstSchema|TestManifestPlatformTokensHaveOneVocabulary|TestRegistryContainsUniqueToolAndSafeguardHandlers' -count=1` passed.
- `go run ./internal/deployability/cmd/capability-vocabulary --root . --check` passed.
- `git diff --check` passed.

The first owner refresh entered a shared generated-package rebuild and expired
at the 120-second lifecycle ceiling. A subsequent governed start completed in
21 seconds with infrastructure-manager healthy but degraded because
`data-backup-manager` and `vrooli-autoheal` were unavailable. The resulting
`vrooli capability ledger --json` readout on this Linux control-plane host
reported:

- Windows/amd64 `unwired`: 0; the six baseline unwired cells are now either
  implemented by the two peers or explicitly policy-ineligible.
- Windows/amd64 `controls_incomplete`: 2, down from the baseline five. The two
  remaining capabilities are `process-containment` and `remote-desktop`, whose
  unrelated controls remain unported and are intentionally not covered by the
  nine-control policy set in this phase.
- `test-execution` and `terminal-multiplexing`: implemented with the named
  peers above.

The live readout still reports zero qualified Windows cells: the new peers are
build-verified and the policy-ineligible cells are not qualification claims.
