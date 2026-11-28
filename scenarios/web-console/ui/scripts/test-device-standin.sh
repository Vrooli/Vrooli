#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "$script_dir/../../../.." && pwd)"
evidence_dir="${MB4_STANDIN_EVIDENCE_DIR:-}"

run_step() {
  local label="$1"
  local working_dir="$2"
  local receipt_name="$3"
  shift 3

  if [[ -n "$evidence_dir" ]]; then
    mkdir -p "$evidence_dir"
    {
      printf '=== SOFTWARE SIMULATION ONLY · %s ===\n' "$label"
      printf 'Command:'
      printf ' %q' "$@"
      printf '\n'
      (cd "$working_dir" && "$@")
    } 2>&1 | tee "$evidence_dir/$receipt_name"
  else
    printf '=== SOFTWARE SIMULATION ONLY · %s ===\n' "$label"
    printf 'Command:'
    printf ' %q' "$@"
    printf '\n'
    (cd "$working_dir" && "$@")
  fi
}

run_step \
  "Web Console production caller" \
  "$repo_root/scenarios/web-console/ui" \
  "mb4-device-standin-web-console-20261126.txt" \
  pnpm exec vitest run \
  src/features/desktop/DesktopMachineTab.test.tsx \
  src/features/desktop/DesktopPane.test.tsx \
  src/features/desktop/desktopTransport.test.ts \
  --reporter=dot --silent

run_step \
  "Device Control plus cross-process Bridge HTTP owner composition" \
  "$repo_root/scenarios/device-control/api" \
  "mb4-device-standin-device-control-20261126.txt" \
  env MB4_BRIDGE_API_DIR="$repo_root/scenarios/vrooli-bridge/api" \
  go test ./handlers/desktop ./internal/desktopwebrtc \
  -run '^(TestSoftwareSimulationCrossProcessBridgeComposition|TestSoftwareSimulationDesktopSessionHTTPContract|TestSoftwareSimulationDesktopSessionServiceLifecycle|TestDesktopSessionServiceMapsReadinessAndInvalidCalls|TestManagerRefusesNotReadyAndRequiresExplicitDisplay|TestManagerEnforcesControllerEpochAndExplicitClipboard|TestManagerReturnsClipboardTextOnlyInAuthorizedReceipt|TestManagerHandlesBrowserDataInputWithReceipt|TestManagerRevokesSessionWhenPeerTransportFails)$' \
  -count=1 -v

run_step \
  "Bridge HTTP/admission owner" \
  "$repo_root/scenarios/vrooli-bridge/api" \
  "mb4-device-standin-bridge-20261126.txt" \
  go test ./handlers/interactive ./internal/interactive \
  -run '^(TestSoftwareSimulationBridgeHTTPAdmissionLifecycle|TestSoftwareSimulationBridgeAdmissionAndRevocation|TestAdmissionRejectsControllerConflictAndOversizedSignal|TestAdmissionRejectsForgedGrantIdentityAndUnsupportedLane|TestAdmissionBoundsPendingSignalsAndReleasesCapacity)$' \
  -count=1 -v
