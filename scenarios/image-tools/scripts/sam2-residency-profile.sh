#!/usr/bin/env bash
set -euo pipefail

scenario_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$scenario_root/../.." && pwd)"
cohort_root="${LM17_COHORT_ROOT:-$HOME/.vrooli/research/cohorts/models-monetization-music-20261115}"
baseline="${LM17_FOOTPRINT_BASELINE:-$cohort_root/BASELINE.json}"
keep_file="${LM17_FOOTPRINT_KEEP_FILE:-$cohort_root/FOOTPRINT-KEEP.txt}"
model_id="${SAM2_PROFILE_MODEL:-sam-2.1-base-plus}"
trials="${SAM2_PROFILE_TRIALS:-3}"
peak_bytes="${SAM2_PROFILE_DISK_PEAK_BYTES:-536870912}"
job_timeout="${SAM2_PROFILE_JOB_TIMEOUT:-10m}"
: "${SAM2_PROFILE_OUT:?set SAM2_PROFILE_OUT to a retained evidence directory}"
out="$SAM2_PROFILE_OUT"
input="$scenario_root/api/internal/ai/testdata/aquila-original-512.png"
mkdir -p "$out"
test -s "$input"
command -v jq >/dev/null
command -v python3 >/dev/null
command -v timeout >/dev/null
footprint() {
  python3 -I "$repo_root/scenarios/research-desk/skills/research-coordination/scripts/footprint.py" diff \
    --baseline "$baseline" --keep-file "$keep_file" --json
}

admit_host() {
  local path="$1"
  footprint >"$path" || true
  vrooli capacity list --active --json >"$path-capacity.json"
  python3 - "$path" "$peak_bytes" "$model_id" <<'PY'
import json, sys
from pathlib import Path
snapshot = json.loads(Path(sys.argv[1]).read_text())
disk = snapshot.get("disk", {})
if not disk.get("available") or not disk.get("free_bytes"):
    raise SystemExit("refused: fresh disk measurement unavailable")
free, peak = int(disk["free_bytes"]), int(sys.argv[2])
growth = max(0, -int(disk.get("change_bytes", 0)))
if free - peak < 120 * 1024**3:
    raise SystemExit("refused: projected free disk falls below 120 GiB")
if growth + peak >= 25 * 1024**3:
    raise SystemExit("refused: projected owner growth reaches 25 GiB")
claims = snapshot.get("claims", {})
if not claims.get("available"):
    raise SystemExit("refused: fresh capacity-claim observation is unavailable")
prefix = f"image-tools:resident:{sys.argv[3]}"
foreign = [claim for claim in claims.get("new", []) if not claim.get("kept") and not claim.get("owner", "").startswith(prefix)]
if foreign:
    raise SystemExit(f"refused: unowned capacity claims present: {foreign}")
PY
}


assert_residency_released() {
  local path="$1"
  footprint >"$path" || true
  python3 - "$path" "$model_id" <<'PY'
import json, sys
from pathlib import Path
snapshot = json.loads(Path(sys.argv[1]).read_text())
prefix = f"image-tools:resident:{sys.argv[2]}"
held = [claim for claim in snapshot.get("claims", {}).get("new", []) if claim.get("owner", "").startswith(prefix)]
if held:
    raise SystemExit(f"worker residency did not release after idle timeout: {held}")
PY
}

valid_trial() {
	local tier="$1" trial="$2" base job_id
	base="$out/$tier-$trial"
  [[ -s "$base-result.json" && -s "$base-job.json" && -s "$base-mask.png" ]] || return 1
  job_id="$(jq -er '.job_id' "$base-result.json")" || return 1
  jq -e --arg id "$job_id" '.job.id == $id and .job.state == "JOB_STATE_SUCCEEDED" and (.job.result_meta.disk_job_attempt_id | startswith($id + "/admission/"))' "$base-job.json" >/dev/null || return 1
  if [[ "$tier" == gpu ]]; then
    jq -e '.job.lane == "JOB_LANE_GPU" and .job.result_meta.gpu_capacity_claim_id != null and .job.result_meta.gpu_uuid != null and .job.result_meta.capacity_release == "residency-held" and .job.result_meta.inference_device == "cuda"' "$base-job.json" >/dev/null
  else
    jq -e '.job.result_meta.gpu_capacity_claim_id == null and .job.result_meta.backend == "sam2-transformers" and .job.result_meta.inference_device == "cpu"' "$base-job.json" >/dev/null
  fi
}

fence_jobs() {
  local base="$1" port
  image-tools jobs list --limit 1000 --json >"$base-jobs.json"
  jq -e '[.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0' "$base-jobs.json" >/dev/null
  port="$(vrooli scenario port image-tools API_PORT --json | jq -er '.port')"
  curl -fsS "http://127.0.0.1:${port}/api/v1/jobs/telemetry/heavy" >"$base-waiters.json"
  jq -e '.complete == true and ([.jobs[].waiter_count] | add // 0) == 0 and ([.jobs[] | select(.state == "JOB_STATE_QUEUED" or .state == "JOB_STATE_RUNNING")] | length == 0)' "$base-waiters.json" >/dev/null
}

originally_enabled="$(image-tools models get "$model_id" --json | jq -r '.model.enabled // false')"
enabled_by_profile=false
active_job_id=""
restore_model_visibility() {
  if [[ -n "$active_job_id" ]]; then
    set +e
    local state=""
    image-tools jobs get "$active_job_id" --json >"$out/interrupted-job-before-cancel.json" 2>/dev/null
    state="$(jq -r '.job.state // "unknown"' "$out/interrupted-job-before-cancel.json" 2>/dev/null)"
    if [[ "$state" == "JOB_STATE_QUEUED" || "$state" == "JOB_STATE_ADMITTING" || "$state" == "JOB_STATE_RUNNING" || "$state" == "JOB_STATE_CANCELING" ]]; then
      image-tools jobs cancel "$active_job_id" --json >"$out/interrupted-job-cancel.json" 2>/dev/null
      timeout --foreground --signal=TERM --kill-after=10s 60s image-tools jobs wait "$active_job_id" --json >"$out/interrupted-job-terminal.json" 2>/dev/null
      image-tools jobs get "$active_job_id" --json >"$out/interrupted-job-final.json" 2>/dev/null
      state="$(jq -r '.job.state // "unknown"' "$out/interrupted-job-final.json" 2>/dev/null)"
    fi
    if [[ "$state" != "JOB_STATE_SUCCEEDED" && "$state" != "JOB_STATE_FAILED" && "$state" != "JOB_STATE_CANCELED" ]]; then
      printf 'SAM2 profile cleanup could not confirm terminal state for Image Tools job %s (state=%s)\n' "$active_job_id" "$state" >&2
    fi
    active_job_id=""
    set -e
  fi
  if [[ "$enabled_by_profile" == true ]]; then
    image-tools models enable "$model_id" --disable --json >"$out/model-disabled-after-profile.json"
  fi
}
trap restore_model_visibility EXIT
if [[ "$originally_enabled" != true ]]; then
  enabled_by_profile=true
  image-tools models enable "$model_id" --json >"$out/model-enabled-for-profile.json"
fi
if [[ "$(image-tools models get "$model_id" --json | jq -r '.model.install.installed // false')" != true ]]; then
  fence_jobs "$out-before-install"
  footprint >"$out/footprint-before-install.json" || true
  vrooli capacity list --active --json >"$out/capacity-before-install.json"
  python3 - "$out/footprint-before-install.json" <<'PY'
import json, sys
v=json.load(open(sys.argv[1])); disk=v['disk']; free=int(disk['free_bytes']); growth=max(0,-int(disk['change_bytes'])); peak=1<<30
if free-peak < 120*(1<<30): raise SystemExit('install refused: projected free disk below 120 GiB')
if growth+peak >= 25*(1<<30): raise SystemExit('install refused: owner growth plus 1 GiB reaches 25 GiB')
if not v['gpu']['available'] or not v['claims']['available']: raise SystemExit('install refused: host capacity inventory incomplete')
foreign=[c for c in v['claims']['new'] if not c.get('kept')]
if foreign: raise SystemExit(f'install refused: unowned capacity claims present: {foreign}')
PY
  vrooli capacity fit --json >"$out/capacity-fit-before-install.json" || true
  image-tools models install "$model_id" --wait --json >"$out/install-result.json"
  image-tools models get "$model_id" --json >"$out/model-installed.json"
  jq -e '.model.install.installed == true' "$out/model-installed.json" >/dev/null
  footprint >"$out/footprint-after-install.json" || true
  python3 - "$out/footprint-after-install.json" <<'PY'
import json, sys
v=json.load(open(sys.argv[1])); disk=v['disk']; free=int(disk['free_bytes']); growth=max(0,-int(disk['change_bytes']))
if free < 120*(1<<30): raise SystemExit('install post-check: disk reserve below 120 GiB')
if growth >= 25*(1<<30): raise SystemExit('install post-check: cohort owner growth reached 25 GiB')
foreign=[c for c in v['claims']['new'] if not c.get('kept')]
if foreign: raise SystemExit(f'install post-check: unexpected capacity claims remain: {foreign}')
PY
  du -sb "$(jq -er '.model.install.path' "$out/model-installed.json")" >"$out/model-bytes.txt"
fi

image-tools models select segment --override "$model_id" --json >"$out/selection.json"
jq -e --arg model "$model_id" '.model.id == $model' "$out/selection.json" >/dev/null
vrooli capacity fit --json >"$out/capacity-fit-before.json" || true

# Try the GPU tier first while the preflight snapshot is still current. A CPU
# baseline may take several minutes and otherwise lets unrelated host work
# change between the baseline and the capability under investigation.
for tier in gpu cpu; do
  ran_tier=false
  for trial in $(seq 1 "$trials"); do
    base="$out/$tier-$trial"
    if valid_trial "$tier" "$trial"; then
      printf 'Reusing retained SAM2 %s trial %s job %s.\n' "$tier" "$trial" "$(jq -r '.job.id' "$base-job.json")"
      continue
    fi
    fence_jobs "$base-before"
    admit_host "$base-footprint-before.json"
    args=()
    if [[ "$tier" == cpu ]]; then args+=(--force-cpu); else args+=(--require-gpu); fi
    image-tools ai segment "$input" --model "$model_id" --selection '{"points":[{"x":0.56,"y":0.52}]}' \
      --fallback-policy local_only --priority batch --no-reclaim "${args[@]}" --json >"$base-result.json"
    job_id="$(jq -er '.job_id' "$base-result.json")"
    active_job_id="$job_id"
    if ! timeout --foreground --signal=TERM --kill-after=15s "$job_timeout" \
      image-tools jobs wait "$job_id" --json >"$base-wait.json"; then
      printf 'SAM2 %s trial %s exceeded the %s hard runtime bound; cancelling exact job %s\n' "$tier" "$trial" "$job_timeout" "$job_id" >&2
      exit 1
    fi
    image-tools jobs get "$job_id" --json >"$base-job.json"
    image-tools jobs download "$job_id" --out "$base-mask.png" >/dev/null
    jq -e --arg id "$job_id" '.job.result_meta.disk_job_attempt_id | startswith($id + "/admission/")' "$base-job.json" >/dev/null
    test -s "$base-mask.png"
    if [[ "$tier" == gpu ]]; then
      jq -e '.job.state == "JOB_STATE_SUCCEEDED" and .job.result_meta.gpu_capacity_claim_id != null and .job.result_meta.gpu_uuid != null and .job.result_meta.capacity_release == "residency-held" and .job.result_meta.inference_device == "cuda"' "$base-job.json" >/dev/null
    else
      jq -e '.job.state == "JOB_STATE_SUCCEEDED" and .job.result_meta.gpu_capacity_claim_id == null and .job.result_meta.backend == "sam2-transformers"' "$base-job.json" >/dev/null
    fi
    active_job_id=""
    ran_tier=true
  done
  if [[ "$ran_tier" == true || ! -s "$out-footprint-after-$tier.json" ]]; then
    # Workers hold only their own leases for 30 seconds after the final request.
    sleep 35
    fence_jobs "$out-$tier-idle"
    admit_host "$out-footprint-after-$tier.json"
    assert_residency_released "$out-footprint-residency-released-$tier.json"
  fi
  if [[ "$tier" == gpu ]]; then
    vrooli capacity list --active --json >"$out/capacity-after-gpu-idle.json"
    for trial in $(seq 1 "$trials"); do
      claim_id="$(jq -er '.job.result_meta.gpu_capacity_claim_id' "$out/gpu-$trial-job.json")"
      jq -e --arg id "$claim_id" '[.claims[] | select(.claim_id == $id)] | length == 0' "$out/capacity-after-gpu-idle.json" >/dev/null || {
        printf 'GPU worker claim %s remained after its 30-second idle release\n' "$claim_id" >&2
        exit 1
      }
    done
  fi
done

vrooli capacity fit --json >"$out/capacity-fit-after.json" || true
python3 - "$out" "$model_id" "$trials" <<'PY'
import json, statistics, sys
from datetime import datetime
from pathlib import Path
from PIL import Image
import numpy as np
out, model, count = Path(sys.argv[1]), sys.argv[2], int(sys.argv[3])
result = {"model": model, "operation": "segment", "trials_per_tier": count, "tiers": {}}
for tier in ("cpu", "gpu"):
    wall, load, forward, hits, disk_reservations, disk_peaks = [], [], [], [], [], []
    for n in range(1, count + 1):
        job = json.loads((out / f"{tier}-{n}-job.json").read_text())["job"]
        meta = job.get("result_meta", {})
        wall.append((datetime.fromisoformat(job["finished_at"].replace("Z", "+00:00")) - datetime.fromisoformat(job["started_at"].replace("Z", "+00:00"))).total_seconds())
        load.append(float(meta.get("inference_model_load_ms", "nan")))
        forward.append(float(meta.get("inference_forward_ms", "nan")))
        hits.append(meta.get("inference_model_cache_hit", "false").lower() == "true")
        disk_reservations.append(meta.get("disk_job_reservation_id", ""))
        disk_peaks.append(int(meta.get("disk_job_filesystem_0_peak_bytes", "0")))
    if any(value != value for value in load + forward):
        raise SystemExit(f"{tier} results missing warm-worker model timings")
    if any(not value for value in disk_reservations) or len(set(disk_reservations)) != count:
        raise SystemExit(f"{tier} jobs reused or omitted a per-job disk reservation id")
    if any(value <= 0 for value in disk_peaks):
        raise SystemExit(f"{tier} jobs omitted their measured disk reservation peak")
    result["tiers"][tier] = {"job_wall_seconds": wall, "job_wall_median_seconds": statistics.median(wall), "model_load_ms": load, "forward_ms": forward, "cache_hits": hits, "disk_job_reservation_ids": disk_reservations, "disk_job_peak_bytes": disk_peaks, "warm_forward_median_ms": statistics.median(forward[1:]) if len(forward) > 1 else None}
ious = []
for n in range(1, count + 1):
    cpu_mask = np.asarray(Image.open(out / f"cpu-{n}-mask.png").convert("L")) > 0
    gpu_mask = np.asarray(Image.open(out / f"gpu-{n}-mask.png").convert("L")) > 0
    if cpu_mask.shape != gpu_mask.shape:
        raise SystemExit(f"CPU/CUDA mask shape mismatch on trial {n}")
    union = int(np.logical_or(cpu_mask, gpu_mask).sum())
    ious.append(float(np.logical_and(cpu_mask, gpu_mask).sum() / union) if union else 1.0)
result["cpu_cuda_mask_iou"] = ious
cpu = result["tiers"]["cpu"]["job_wall_median_seconds"]
gpu = result["tiers"]["gpu"]["job_wall_median_seconds"]
result["speedup"] = cpu / gpu if gpu > 0 else 0
result["gpu_faster"] = gpu < cpu
(out / "profile.json").write_text(json.dumps(result, indent=2) + "\n")
print(f"SAM2 {model}: CPU median {cpu:.3f}s, CUDA median {gpu:.3f}s, {result['speedup']:.2f}x, mean mask IoU {statistics.mean(ious):.4f}")
if statistics.mean(ious) < 0.98:
    raise SystemExit("CPU/CUDA output-mask parity is below 0.98 IoU")
if not result["gpu_faster"]:
    raise SystemExit("CUDA service median did not beat CPU; profile retained for the next optimization iteration")
PY
