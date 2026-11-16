#!/usr/bin/env bash
set -euo pipefail

scenario_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$scenario_root/../.." && pwd)"
: "${BACKDROP_STUDIO_GPU_SHOWCASE_OUT:?set BACKDROP_STUDIO_GPU_SHOWCASE_OUT to a retained evidence directory}"
: "${BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES:?set this to a measured comparable GPU job peak in bytes}"
: "${LM17_FOOTPRINT_BASELINE:?set this to the cohort BASELINE.json}"
: "${LM17_FOOTPRINT_KEEP_FILE:?set this to the cohort FOOTPRINT-KEEP.txt}"

out="$BACKDROP_STUDIO_GPU_SHOWCASE_OUT/preflight"
mkdir -p "$out"
printf '%s\n' "$BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES" >"$out/measured-disk-peak-bytes.txt"
command -v jq >/dev/null
command -v python3 >/dev/null
command -v curl >/dev/null
command -v timeout >/dev/null

python3 -I "$repo_root/scenarios/research-desk/skills/research-coordination/scripts/footprint.py" diff \
	--baseline "$LM17_FOOTPRINT_BASELINE" --keep-file "$LM17_FOOTPRINT_KEEP_FILE" --disk --json \
	>"$out/footprint.json" || true
python3 - "$out/footprint.json" "$BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES" <<'PY'
import json, sys
from pathlib import Path

v = json.loads(Path(sys.argv[1]).read_text())
if v.get("not_checked"):
    raise SystemExit("GPU showcase refused: incomplete host inventory: " + str(v["not_checked"]))
for section in ("gpu", "resources", "claims", "containers", "timers", "sandboxes", "disk"):
    if not v.get(section, {}).get("available", False):
        raise SystemExit("GPU showcase refused: " + section + " inventory is unavailable")
if int(v.get("new_count", -1)) != 0:
    claims = [c for c in v["claims"].get("new", []) if not c.get("kept")]
    raise SystemExit("GPU showcase refused: fresh footprint has unkept new owners/resources: " + str(claims))
disk = v["disk"]
free = int(disk.get("free_bytes", 0))
growth = max(0, -int(disk.get("change_bytes", 0)))
peak = int(sys.argv[2])
if peak <= 0:
    raise SystemExit("GPU showcase refused: measured comparable disk peak must be positive")
if free - peak < 120 * 1024**3:
    raise SystemExit("GPU showcase refused: projected disk free falls below 120 GiB")
if growth + peak >= 25 * 1024**3:
    raise SystemExit("GPU showcase refused: projected cohort growth reaches 25 GiB")
PY

vrooli capacity list --active --json >"$out/capacity-list.json"
vrooli capacity fit --json >"$out/capacity-fit.json" || true
image-tools jobs list --limit 1000 --json >"$out/jobs.json"
jq -e '[.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0' "$out/jobs.json" >/dev/null
api_port="$(vrooli scenario port image-tools API_PORT --json | jq -er '.port')"
curl -fsS "http://127.0.0.1:${api_port}/api/v1/jobs/telemetry/heavy" >"$out/waiters.json"
jq -e '.complete == true and ([.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0) and (([.jobs[].waiter_count] | add) // 0) == 0' "$out/waiters.json" >/dev/null

selection="$(image-tools models select text_to_image --json)"
printf '%s\n' "$selection" >"$out/selection.json"
model="$(printf '%s' "$selection" | jq -er '.model.id')"
jq -e '.gpu_viable == true' "$out/selection.json" >/dev/null
image-tools models explain text_to_image --model "$model" --json >"$out/resolution.json"
jq -e '.resolution.gpu_viable == true and .resolution.tier == "local-gpu"' "$out/resolution.json" >/dev/null
doctor_status=0
image-tools backends doctor --json >"$out/backends.json" 2>"$out/backend-doctor.stderr" || doctor_status=$?
# The broad doctor also reports absent optional providers. Admission depends on
# the backend for this operation; the targeted predicates below decide that.
printf '%s\n' "$doctor_status" >"$out/backend-doctor-exit-status.txt"
jq -e '.backends[] | select(.name == "stable-diffusion.cpp") | select((.operations // []) | index("text_to_image")) | .available == true and .gpu_capable == true' "$out/backends.json" >/dev/null
free -b >"$out/memory.txt"
cat /proc/pressure/memory >"$out/memory-pressure.txt"
cat /proc/vmstat >"$out/vmstat.txt"
nvidia-smi --query-gpu=uuid,memory.used,memory.free,utilization.gpu --format=csv,noheader >"$out/gpu.txt"
printf 'Backdrop GPU preflight passed. Text-to-image model=%s; exact GPU claim is required by the render request. Evidence: %s\n' "$model" "$out"
