#!/usr/bin/env bash
set -euo pipefail

scenario_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$scenario_root/../.." && pwd)"
input="$scenario_root/api/internal/ai/testdata/aquila-original-512.png"
model_id="${ONNX_BG_REMOVAL_MODEL:-u2netp}"
disk_peak_bytes="${ONNX_BG_REMOVAL_DISK_PEAK_BYTES:-2147483648}"
: "${ONNX_BG_REMOVAL_E2E_OUT:?set this to a retained evidence directory}"
: "${LM17_FOOTPRINT_BASELINE:?set this to the cohort BASELINE.json}"
: "${LM17_FOOTPRINT_KEEP_FILE:?set this to the cohort FOOTPRINT-KEEP.txt}"

out="$ONNX_BG_REMOVAL_E2E_OUT"
mkdir -p "$out"
test -s "$input" || { printf "missing fixed fixture: %s\n" "$input" >&2; exit 1; }
command -v jq >/dev/null || { printf 'jq is required\n' >&2; exit 1; }
command -v python3 >/dev/null || { printf 'python3 is required\n' >&2; exit 1; }
command -v curl >/dev/null || { printf 'curl is required\n' >&2; exit 1; }

footprint_diff() {
	python3 -I "$repo_root/scenarios/research-desk/skills/research-coordination/scripts/footprint.py" diff \
		--baseline "$LM17_FOOTPRINT_BASELINE" --keep-file "$LM17_FOOTPRINT_KEEP_FILE" --json
}

check_footprint() {
	python3 - "$1" "$disk_peak_bytes" <<'PY'
import json
import sys
from pathlib import Path

snapshot = json.loads(Path(sys.argv[1]).read_text())
disk = snapshot.get('disk', {})
if not disk.get('available') or not disk.get('free_bytes'):
    raise SystemExit('footprint disk measurement is unavailable')
peak = int(sys.argv[2])
free = int(disk['free_bytes'])
growth = max(0, -int(disk.get('change_bytes', 0)))
floor = 120 * 1024**3
cohort_limit = 25 * 1024**3
if free - peak < floor:
    raise SystemExit(f'GPU run refused: free disk {free} minus peak {peak} falls below 120 GiB')
if growth + peak >= cohort_limit:
    raise SystemExit(f'GPU run refused: projected host/cohort growth {growth + peak} reaches the 25 GiB ceiling')
PY
}

capture_gpu_footprint() {
	local path="$1" status=0
	footprint_diff >"$path" || status=$?
	if [ "$status" -eq 0 ]; then return 0; fi
	# The CPU baseline's warm worker retains an explicitly admitted RAM claim
	# for its 30-second idle window. Let that owner release its own claim, then
	# take a new diff. Any other change remains a hard refusal.
	python3 - "$path" "$model_id" <<'PY'
import json
import sys
from pathlib import Path

snapshot = json.loads(Path(sys.argv[1]).read_text())
claims = [claim for claim in snapshot.get('claims', {}).get('new', []) if not claim.get('kept')]
expected = f"image-tools:resident:{sys.argv[2]}"
if len(claims) != 1 or claims[0].get('owner') != expected or claims[0].get('kind') != 'ram':
    raise SystemExit('GPU run refused: footprint diff contains unexpected new claims')
PY
	sleep 35
	status=0
	footprint_diff >"$path" || status=$?
	if [ "$status" -ne 0 ]; then
		printf 'GPU run refused: CPU worker residency did not release within its bounded idle interval\n' >&2
		return 1
	fi
}

after_diff() {
	local status=$?
	if footprint_diff >"$out/footprint-after.json" 2>"$out/footprint-after.err"; then
		if [ "$status" -eq 0 ]; then check_footprint "$out/footprint-after.json" || status=$?; fi
	else
		if [ "$status" -eq 0 ]; then status=1; fi
	fi
	return "$status"
}
trap after_diff EXIT

footprint_diff >"$out/footprint-before-cpu.json"
check_footprint "$out/footprint-before-cpu.json"
make -C "$scenario_root" onnx-gpu-backend-check
image-tools models select background_removal --override "$model_id" --json >"$out/selection.json"
jq -e --arg model "$model_id" '.model.id == $model' "$out/selection.json" >/dev/null || {
	printf 'model selection did not resolve the requested rights-qualified model %s\n' "$model_id" >&2
	exit 1
}

image-tools ai bg-removal "$input" --model "$model_id" --fallback-policy local_only --priority batch --no-reclaim \
	--force-cpu --wait --out "$out/cpu.png" --json >"$out/cpu-result.json"
cpu_job="$(jq -er '.job_id' "$out/cpu-result.json")"
image-tools jobs get "$cpu_job" --json >"$out/cpu-job.json"
test -s "$out/cpu.png" || { printf 'CPU output is empty\n' >&2; exit 1; }
jq -e '.job.state == "JOB_STATE_SUCCEEDED" and .job.result_meta.backend == "onnxruntime" and (.job.result_meta.gpu_capacity_claim_id == null)' "$out/cpu-job.json" >/dev/null || {
	printf 'CPU baseline did not use the intended local ONNX CPU path\n' >&2
	exit 1
}

# Recheck both durable jobs and server-side waiter telemetry after the CPU
# baseline, immediately before the GPU footprint and capacity admission.
image-tools jobs list --limit 1000 --json >"$out/jobs-before-gpu.json"
jq -e '[.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0' "$out/jobs-before-gpu.json" >/dev/null || {
	printf 'GPU comparison refused: Image Tools has queued or running durable jobs\n' >&2
	exit 1
}
api_port="$(vrooli scenario port image-tools API_PORT --json | jq -er '.port')"
curl -fsS "http://127.0.0.1:${api_port}/api/v1/jobs/telemetry/heavy" >"$out/waiters-before-gpu.json"
jq -e '.complete == true and ([.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0) and (([.jobs[].waiter_count] | add) // 0) == 0' "$out/waiters-before-gpu.json" >/dev/null || {
	printf 'GPU comparison refused: heavy-job inventory is incomplete or has queued/running work/waiters\n' >&2
	exit 1
}

# A fresh cohort diff immediately precedes the admitted GPU job. The job itself
# also uses --require-gpu, so the capacity broker rejects any CPU downgrade.
capture_gpu_footprint "$out/footprint-before-gpu.json"
check_footprint "$out/footprint-before-gpu.json"
image-tools ai bg-removal "$input" --model "$model_id" --fallback-policy local_only --priority batch --no-reclaim \
	--require-gpu --wait --out "$out/gpu.png" --json >"$out/gpu-result.json"
gpu_job="$(jq -er '.job_id' "$out/gpu-result.json")"
image-tools jobs get "$gpu_job" --json >"$out/gpu-job.json"
test -s "$out/gpu.png" || { printf 'GPU output is empty\n' >&2; exit 1; }
jq -e '.job.state == "JOB_STATE_SUCCEEDED" and .job.result_meta.backend == "onnxruntime" and .job.result_meta.gpu_capacity_claim_id != null and .job.result_meta.gpu_uuid != null and .job.result_meta.capacity_release == "released"' "$out/gpu-job.json" >/dev/null || {
	printf 'GPU run has no successful, released ONNX GPU capacity receipt\n' >&2
	exit 1
}

python3 - "$out/cpu-job.json" "$out/gpu-job.json" "$out/measurements.json" "$model_id" <<'PY'
import json
import sys
from datetime import datetime
from pathlib import Path

def duration(path):
    job = json.loads(Path(path).read_text())['job']
    start = datetime.fromisoformat(job['started_at'].replace('Z', '+00:00'))
    finish = datetime.fromisoformat(job['finished_at'].replace('Z', '+00:00'))
    return (finish - start).total_seconds()

cpu_seconds = duration(sys.argv[1])
gpu_seconds = duration(sys.argv[2])
if cpu_seconds <= 0 or gpu_seconds <= 0:
    raise SystemExit('job timestamps do not produce positive durations')
speedup = cpu_seconds / gpu_seconds
record = {
    'operation': 'background_removal',
    'model': sys.argv[4],
    'cpu_seconds': cpu_seconds,
    'gpu_seconds': gpu_seconds,
    'speedup': speedup,
    'gpu_faster': gpu_seconds < cpu_seconds,
}
Path(sys.argv[3]).write_text(json.dumps(record, indent=2) + '\n')
print(f"ONNX background removal: CPU {cpu_seconds:.3f}s, CUDA {gpu_seconds:.3f}s, {speedup:.2f}x")
if not record['gpu_faster']:
    raise SystemExit('CUDA execution did not beat the CPU baseline on this fixture')
PY

printf 'ONNX background-removal artifacts and capacity receipts retained at %s\n' "$out"
