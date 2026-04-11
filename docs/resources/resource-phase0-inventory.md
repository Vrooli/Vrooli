# Resource Phase 0 Inventory

This is the first full Phase 0 triage inventory for project-level resources.

Classification rules used in this pass:

- `keep`
  - actively referenced by scenario manifests, or
  - still enabled at the project level after step 2 decisions
- `blueprint`
  - implemented or known concept, but not currently justified as an actively maintained integration
- `deprecate`
  - stale config/registry metadata with no implementation and no current usage signal

Additional bias applied in this pass:

- Explicit blueprint decisions override implementation/runtime signals for:
  - `kafka`
  - `node-red`
  - `pihole`
  - `pybullet`
  - `restic`
  - `ros2`
  - `segment-anything`
  - `su2`
  - `terraform`
  - `traccar`
  - `ultralytics-yolo`
  - `virustotal`
  - `wireguard`
  - `zigbee2mqtt`

## Inventory Table

| Resource | Presence | Registry | Root Enabled | Scenario Refs | Proposed State | Why |
|---|---|---|---:|---:|---|---|
| `autogen-studio` | registry-only | yes | no | 0 | `deprecate` | Registry-only stale metadata with no current usage signal. |
| `blender` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `browserless` | implemented | yes | yes | 11 | `keep` | Used by 11 scenarios. |
| `btcpay` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `claude-code` | implemented | yes | yes | 16 | `keep` | Used by 16 scenarios. |
| `cloudflare-ai-gateway` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `cncjs` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `codex` | implemented | yes | yes | 2 | `keep` | Used by 2 scenarios. |
| `comfyui` | implemented | yes | no | 1 | `keep` | Used by 1 scenario. |
| `earthly` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `eclipse-ditto` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `elmer-fem` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `erpnext` | registry-only | yes | no | 0 | `deprecate` | Registry-only stale metadata with no current usage signal. |
| `esphome` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `ffmpeg` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `freecad` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `gazebo` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `gemini` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `geonode` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `geth` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `ggwave` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `godot` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `gridlabd` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `haystack` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `home-assistant` | implemented | yes | yes | 1 | `keep` | Used by 1 scenario. |
| `judge0` | implemented | yes | yes | 1 | `keep` | Used by 1 scenario. |
| `k6` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `kafka` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `keycloak` | implemented | yes | no | 0 | `blueprint` | Explicit blueprint decision. |
| `kicad` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `kokoro` | implemented | no | yes | 1 | `keep` | Used by 1 scenario. |
| `langchain` | config+registry-only | yes | no | 0 | `deprecate` | Disabled root config plus stale registry metadata, with no implementation or active scenario usage. |
| `litellm` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `llamaindex` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `lnbits` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `mail-in-a-box` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `mathlib` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `matrix-synapse` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `mcrcon` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `meep` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `mifos` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `minio` | implemented | yes | yes | 17 | `keep` | Used by 17 scenarios. |
| `musicgen` | registry-only | yes | no | 0 | `deprecate` | Registry-only stale metadata with no current usage signal. |
| `neo4j` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `node-red` | registry-only | yes | no | 0 | `blueprint` | Explicit blueprint decision. Live scenario manifest references were removed during Phase 0 validation. |
| `nsfw-detector` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `obs-studio` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `octoprint` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `ollama` | implemented | yes | yes | 47 | `keep` | Used by 47 scenarios. |
| `open-data-cube` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `opencode` | implemented | yes | yes | 2 | `keep` | Used by 2 scenarios. |
| `openfoam` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `openrouter` | implemented | yes | yes | 6 | `keep` | Used by 6 scenarios. |
| `openscad` | implemented | yes | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `papermc` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `parlant` | config-only | no | no | 0 | `blueprint` | Disabled root config concept with no implementation yet; preserve as blueprint knowledge instead of treating it as supported. |
| `pihole` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `postgis` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `postgres` | implemented | yes | yes | 74 | `keep` | Used by 74 scenarios. |
| `pushover` | implemented | yes | no | 0 | `blueprint` | Explicit blueprint decision. |
| `pybullet` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `qdrant` | implemented | yes | yes | 36 | `keep` | Used by 36 scenarios. |
| `questdb` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `redis` | implemented | yes | yes | 46 | `keep` | Used by 46 scenarios. |
| `restic` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `ros2` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `sagemath` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `searxng` | implemented | yes | yes | 2 | `keep` | Used by 2 scenarios. |
| `segment-anything` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `simpy` | implemented | yes | no | 0 | `blueprint` | Explicit blueprint decision. |
| `speaker-verification` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `sqlite` | implemented | no | yes | 6 | `keep` | Used by 6 scenarios. |
| `step-ca` | implemented | yes | no | 0 | `blueprint` | Explicit blueprint decision. |
| `su2` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `terraform` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `traccar` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `twilio` | implemented | yes | yes | 0 | `keep` | Project-level enabled. |
| `ultralytics-yolo` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `unstructured-io` | implemented | yes | yes | 5 | `keep` | Used by 5 scenarios. |
| `vault` | implemented | yes | yes | 2 | `keep` | Used by 2 scenarios. |
| `virustotal` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `vnc` | implemented | no | no | 0 | `blueprint` | Implemented but no current project/scenario usage signal. |
| `whisper` | implemented | yes | yes | 1 | `keep` | Used by 1 scenario. |
| `wikijs` | implemented | yes | no | 0 | `blueprint` | Explicit blueprint decision. |
| `wireguard` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |
| `zigbee2mqtt` | implemented | no | no | 0 | `blueprint` | Explicit blueprint decision. |

## Phase 0 Notes

- This table should be treated as the working review artifact for Phase 0 acceptance.
- Registry presence is included as context only; it is no longer authoritative.
- `parlant` is intentionally tracked even though it has no implementation or registry entry, because root `.vrooli/service.json` is part of the Phase 0 source-of-truth set.
- `node-red` remains a blueprint. The stale scenario manifest references were removed during Phase 0 validation; remaining Node-RED files are historical prototypes, not active dependencies.
- `sqlite` and `kokoro` currently classify as `keep` even without registry entries because they have active scenario usage and/or project-level enablement.
