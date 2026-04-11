# Dependency Contract Audit

This document records the normalized dependency contract for scenario `.vrooli/service.json` files and the current audit state after the repo-wide cleanup pass.

## Canonical Contract

- `dependencies.resources` is a flat object map keyed by canonical resource name.
- `dependencies.scenarios` is a flat object map keyed by canonical scenario name.
- `required` describes functional necessity.
- `startup_policy` describes orchestration behavior.
- `startup_policy` values:
  - `must_start`
  - `try_start`
  - `ignore`
- Do not use:
  - `required` / `optional` arrays
  - CLI alias keys such as `resource-claude-code`
  - underscore resource keys such as `unstructured_io`
  - scenario names under `dependencies.resources`

## Normalization Completed

The following scenarios were normalized to the canonical contract during this pass:

- `reference-react-vite`
- `swarm-manager`
- `prompt-manager`
- `web-console`
- `test-data-generator`
- `prd-control-tower`
- `landing-manager`
- `landing-manager/generated/fffff`
- `scenario-to-extension`
- `scenario-to-android`
- `code-smell`
- `device-sync-hub`
- `document-manager`
- `react-component-library`
- `home-automation`
- `calendar`

Repo-wide validation then confirmed:

- `92` scenario service files parse as valid JSON
- no scenario service file still uses `required` / `optional` resource arrays
- no scenario service file still uses `resource-*` dependency keys
- no scenario service file still places known scenario dependencies under `dependencies.resources`

## Validation Command

Run:

```bash
scripts/resources/tools/validate-dependency-contract.sh
```

This script enforces the scenario dependency rules above and also reports resource/registry drift as warnings.

## Remaining Reconciliation Work

Dependency shape is now normalized, but resource inventory reconciliation is still needed for Phase 0.

Current project-level audit snapshot:

- `92` scenario service files
- `80` resource directories under `resources/`
- `46` registry entries under `.vrooli/resource-registry/`

Resources currently present in `resources/` but missing registry entries include:

- `cncjs`
- `earthly`
- `eclipse-ditto`
- `elmer-fem`
- `esphome`
- `freecad`
- `gazebo`
- `geonode`
- `ggwave`
- `godot`
- `gridlabd`
- `kafka`
- `kokoro`
- `lnbits`
- `mathlib`
- `matrix-synapse`
- `mcrcon`
- `meep`
- `mifos`
- `nsfw-detector`
- `octoprint`
- `open-data-cube`
- `openfoam`
- `papermc`
- `pihole`
- `pybullet`
- `restic`
- `ros2`
- `segment-anything`
- `speaker-verification`
- `sqlite`
- `su2`
- `terraform`
- `traccar`
- `ultralytics-yolo`
- `virustotal`
- `vnc`
- `wireguard`
- `zigbee2mqtt`

Registry entries currently present without matching resource directories:

- `autogen-studio`
- `erpnext`
- `langchain`
- `musicgen`
- `node-red`

Those mismatches should be resolved before final `keep` / `blueprint` / `deprecate` classification.
