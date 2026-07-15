# UI Health Phase

The `ui-health` phase is the single authority for **all** UI validation. It runs in **execution mode** (`IncludeExecution: true`) and produces one consolidated report covering: `ui/manifest.json` bindings + slot directories + overlay rules; static UI-interop rules; net-new UI standards (i18n parity, design-token/no-raw-hex, a gating axe harness, PWA/viewport, tsconfig-strict, eslint stability); UI-bundle freshness (the canonical content-hash engine — a stale bundle gates); and a **BAS-driven runtime render + iframe-bridge handshake** group. The runtime group absorbed the retired native `smoke` phase.

The phase declares `NeedsUI: true` and `RequiredResources: [browser-automation-studio]`. Runtime evidence has three explicit states: evaluated, skipped/unavailable, and evidence-incomplete. An unavailable BAS/UI is an infrastructure condition rather than a source defect, but it returns `DEGRADED` (never `PASSED`); static-only runs likewise report `runtime_not_evaluated_static_only`. A runtime maturity rung can advance only when every configured viewport has downloadable screenshot, DOM, layout, viewport provenance, and handshake/interaction evidence.

This phase declares a [Phase Capability Contract](../../concepts/phase-capability-contract.md); the sections below follow the required remediation-doc skeleton.

## North Star

Every UI dimension is fully clean across the ladder: the manifest contract is authoritative and ready for downstream providers; the UI is fully proxy, tunnel, localhost, and iframe safe; freshness is fully clean and ready for downstream runtime validation; browser-driven render, handshake, console, network, and visual validation is fully clean; project standards are fully clean and self-healing for the mechanical subset; and installability, launch, offline, and declared platform capabilities are fully clean. At maximum maturity the UI is a **native-feeling, embed-safe web app** whose every surface is verified, not merely free of findings — the top rung of the deepest capability (`pwa_native_readiness` L5) is the aspiration the whole phase steers toward.

## The rungs and their gates

ui-health reports a ladder per capability. Each rung is monotone (it implies the one below); the top rung of each ladder is that capability's North Star and its next unlock is the single highest-value move toward it.

| Capability | Rungs | Top rung (North Star) | Example next unlock |
|---|---|---|---|
| `manifest_contract` | L0 No UI surface → L5 Contract authority | UI manifest contract fully clean, ready for downstream providers | L1→L2: declare a known template and compatible contract shape |
| `interop` | L0 Not embeddable → L5 Interop complete | Fully proxy, tunnel, localhost, and iframe safe | L2→L3: harden shortcut, focus, viewport, and server behavior |
| `freshness` | L0 Unknown bundle → L5 Freshness authority | Fully clean and ready for downstream runtime validation | L2→L3: wire freshness + code-facts authority into the provider |
| `runtime_render` | L0 Not runnable → L5 Runtime complete | Browser-driven UI validation fully clean | L2→L3: clean console, network, asset, and page-error checks |
| `project_standards` | L0 Standards absent → L5 Standards complete | Fully clean and self-healing for the mechanical subset | L2→L3: clean i18n parity, token use, a11y harness, strict config |
| `pwa_native_readiness` | L0 No install surface → L5 Native ready | Installability, launch, offline, and platform capabilities fully clean | L1→L2: register a service worker with offline app-shell fallback |

## What each finding means

Each finding caps the capability it names at the rung below its impact; only ERROR/BLOCKER severities fail the phase, so WARNING/INFO findings are honest, non-failing debt. Representative codes (full inventory in the descriptor `maturity.findings` map):

| Code | Capability | Caps at | Severity | Fails phase? |
|---|---|---|---|---|
| `service_json_missing` / `service_json_invalid` / `template_unknown` | manifest_contract | L0 | ERROR | Yes |
| `contract_kind_mismatch` / `slots_empty` / `overlay_unknown_slot` | manifest_contract | L1 | ERROR | Yes |
| `interop_relative_base` / `interop_router_basename` / `interop_helmet_frame_ancestors` | interop | L1 | ERROR | Yes |
| `interop_h_screen` / `interop_shortcut_relay` | interop | L1 | WARNING | No |
| `freshness_ui_bundle_stale` | freshness | L2 | ERROR | Yes |
| `runtime_load_failed` / `runtime_handshake_failed` / `visual_pixel_blank` | runtime_render | L3 | ERROR | Yes |
| `runtime_console_errors` / `visual_text_clipped` | runtime_render | L3 | WARNING | No |
| `runtime_not_evaluated_static_only` / `runtime_skipped_*` / `runtime_evidence_incomplete` | runtime_render | L0 | INFO | Provider is DEGRADED, never PASSED |
| `standard_tsconfig_strict` | project_standards | L4 | ERROR | Yes |
| `standard_i18n_locale_parity` | project_standards | L4 | WARNING | No |
| `standard_a11y_harness` | project_standards | L4 | ERROR | Yes — applicable UI scenarios require the axe dependency, canonical helper, and baseline test |
| `pwa_manifest_install_fields` / `pwa_service_worker_offline` | pwa_native_readiness | L1–L3 | WARNING | No |

## The canonical fix

- **Manifest-contract findings** → repair `ui/manifest.json` and `service.json`: declare a known `template`, fix contract kind/schema, populate slots and overlays, and align slot directories on the filesystem. Some (`slot_dir_missing`, `slot_parent_dir_missing`) have implemented safe fixers.
- **Interop findings** → adopt the relative Vite base, router basename, proxy-preserved API base, iframe bridge, and frame-ancestor helmet; remove hardcoded localhost. `interop_h_screen` and `interop_protective_comments` have safe fixers.
- **Freshness findings** → rebuild the UI bundle so its content hash matches source (`freshness_ui_bundle_stale` gates); a stale bundle means the running UI does not reflect committed code.
- **Runtime-render findings** → fix the live render: repair the iframe handshake, clear console/network/page errors, resolve visual defects (blank pixels, broken assets, viewport overflow, unsafe edge tap zones), and restore any missing artifact channel before claiming a visual pass.
- **Project-standards findings** → restore i18n locale parity, replace raw hex with design tokens, add the required axe dependency, canonical helper, and baseline harness test, enable strict tsconfig, and satisfy React-stability lint. Several (i18n parity, tsconfig-strict) have implemented mechanical fixers. The harness is a static gate for UI scenarios; live browser unavailability remains `DEGRADED`, never a static pass.
- **PWA findings** → complete install metadata, launch scope, service-worker offline fallback, and optional platform fields to climb toward native readiness.

## How to verify

```bash
# See the current rung, gaps, and next move for every capability:
ui-health validate scenario <scenario>
ui-health validate scenario <scenario> --static-only   # static groups only, no BAS

# Or drive it through Test Genie and read the per-phase scorecard:
test-genie execute <scenario> --phases ui-health
test-genie runs findings --scenario <scenario>
```

## How It Runs

Test Genie calls `ui-health` through `scenario-validation/v1.ScenarioValidationService.ValidateScenario` with `include_execution: true`. ui-health runs its static groups always; when execution is requested and the scenario has a UI, it resolves the running UI via discovery and drives BAS for the render + handshake check, then returns shared `status` plus `assessment.findings`.

Equivalent human operator flow:

```bash
ui-health validate scenario <scenario>            # full report (static + runtime)
ui-health validate scenario <scenario> --static-only   # static groups only, no BAS
```

## Opt-Out

Skip for a single run:

```bash
test-genie execute <scenario> --skip ui-health
```

Disable per scenario via `.vrooli/testing.json`:

```json
{
  "phases": {
    "ui-health": { "enabled": false }
  }
}
```

## Configuration

Per-scenario timeout override via `.vrooli/testing.json` (default: 60s):

```json
{
  "phases": {
    "ui-health": { "timeout": "120s" }
  }
}
```
