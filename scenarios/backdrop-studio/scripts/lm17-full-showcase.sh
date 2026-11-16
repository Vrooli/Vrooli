#!/usr/bin/env bash
set -euo pipefail

scenario_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "$scenario_root/../.." && pwd)"
: "${BACKDROP_STUDIO_FULL_SHOWCASE_OUT:?set this to a retained LM17 evidence directory}"
: "${BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES:?set this to a measured comparable GPU job peak in bytes}"
: "${LM17_FOOTPRINT_BASELINE:?set this to the cohort BASELINE.json}"
: "${LM17_FOOTPRINT_KEEP_FILE:?set this to the cohort FOOTPRINT-KEEP.txt}"
job_timeout="${LM17_SHOWCASE_JOB_TIMEOUT:-10m}"

# The combined showcase includes SAM2 Base+, whose latest admitted profile
# reserved a measured 2,122,375,168 bytes. Do not let a smaller SD-only peak
# admit the first render and leave the combined run under-reserved.
minimum_measured_full_showcase_peak_bytes=2122375168
if (( BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES < minimum_measured_full_showcase_peak_bytes )); then
  printf 'LM17 full showcase requires a measured disk peak of at least %s bytes (latest SAM2 Base+ job peak); got %s\n' \
    "$minimum_measured_full_showcase_peak_bytes" "$BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES" >&2
  exit 2
fi

out="$BACKDROP_STUDIO_FULL_SHOWCASE_OUT"
mkdir -p "$out"
ollama_pid=""
sampler_pid=""
active_job_id=""
cleanup() {
  if [[ -n "$active_job_id" ]]; then
    set +e
    local job_state=""
    image-tools jobs get "$active_job_id" --json >"$out/interrupted-job-before-cancel.json" 2>/dev/null
    job_state="$(jq -r '.job.state // "unknown"' "$out/interrupted-job-before-cancel.json" 2>/dev/null)"
    if [[ "$job_state" == "JOB_STATE_QUEUED" || "$job_state" == "JOB_STATE_ADMITTING" || "$job_state" == "JOB_STATE_RUNNING" || "$job_state" == "JOB_STATE_CANCELING" ]]; then
      image-tools jobs cancel "$active_job_id" --json >"$out/interrupted-job-cancel.json" 2>/dev/null
      timeout --foreground --signal=TERM --kill-after=10s 60s image-tools jobs wait "$active_job_id" --json >"$out/interrupted-job-terminal.json" 2>/dev/null
      image-tools jobs get "$active_job_id" --json >"$out/interrupted-job-final.json" 2>/dev/null
      job_state="$(jq -r '.job.state // "unknown"' "$out/interrupted-job-final.json" 2>/dev/null)"
    fi
    if [[ "$job_state" != "JOB_STATE_SUCCEEDED" && "$job_state" != "JOB_STATE_FAILED" && "$job_state" != "JOB_STATE_CANCELED" ]]; then
      printf 'LM17 cleanup could not confirm terminal state for Image Tools job %s (state=%s)\n' "$active_job_id" "$job_state" >&2
    fi
    active_job_id=""
    set -e
  fi
  if [[ -n "$ollama_pid" ]] && kill -0 "$ollama_pid" 2>/dev/null; then
    kill "$ollama_pid" 2>/dev/null || true
    wait "$ollama_pid" 2>/dev/null || true
  fi
  if [[ -n "$sampler_pid" ]] && kill -0 "$sampler_pid" 2>/dev/null; then
    kill "$sampler_pid" 2>/dev/null || true
    wait "$sampler_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

preflight() {
  BACKDROP_STUDIO_GPU_SHOWCASE_OUT="$1" \
  BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES="$BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES" \
  bash "$scenario_root/scripts/gpu-showcase-preflight.sh"
}

start_ollama() {
  local prefix="$1"
  (
    date -u +'%Y-%m-%dT%H:%M:%S.%NZ' >"$prefix.started"
    set +e
    resource-ollama gateway chat --role chat.default \
      --system "Write a detailed, useful answer and continue until the output limit." \
      --prompt "Explain practical ways to reduce avoidable GPU memory use in a local AI workstation. Give concrete steps and tradeoffs." \
      --max-tokens 2048 --temperature 0.2 --json >"$prefix.response.json"
    status=$?
    date -u +'%Y-%m-%dT%H:%M:%S.%NZ' >"$prefix.finished"
    exit "$status"
  ) &
  ollama_pid="$!"
}

run_ollama_window() {
  local prefix="$1"
  wait "$ollama_pid"
  ollama_pid=""
  test -s "$prefix.response.json"
  test -s "$prefix.started"
  test -s "$prefix.finished"
  jq -e '(.response | type == "string" and length > 0) and ((.eval_count // 0) > 0)' "$prefix.response.json" >/dev/null
}

valid_ollama_response() {
  local prefix="$1"
  test -s "$prefix.response.json" && test -s "$prefix.started" && test -s "$prefix.finished" &&
    jq -e '(.response | type == "string" and length > 0) and ((.eval_count // 0) > 0)' "$prefix.response.json" >/dev/null
}

reuse_backdrop() {
  local receipt="$out/backdrop/receipt.json" job_id
  [[ -s "$receipt" && -s "$out/backdrop/backdrop.png" ]] || return 1
  valid_ollama_response "$out/ollama-backdrop" || return 1
  jq -e '.generation_tier == "local-gpu" and (.elapsed_seconds > 0 and .elapsed_seconds < 60) and (.image_tools_job_id | length > 0) and (.gpu_claim_id | length > 0) and (.gpu_uuid | length > 0) and (.disk_job_peak_bytes > 0)' "$receipt" >/dev/null || return 1
  job_id="$(jq -er '.image_tools_job_id' "$receipt")"
  image-tools jobs get "$job_id" --json >"$out/backdrop/reuse-job.json" || return 1
  jq -e --arg claim "$(jq -er '.gpu_claim_id' "$receipt")" --arg uuid "$(jq -er '.gpu_uuid' "$receipt")" \
    '.job.state == "JOB_STATE_SUCCEEDED" and .job.lane == "JOB_LANE_GPU" and .job.result_meta.gpu_capacity_claim_id == $claim and .job.result_meta.gpu_uuid == $uuid' \
    "$out/backdrop/reuse-job.json" >/dev/null || return 1
  printf 'Reusing retained Backdrop GPU receipt %s from this showcase attempt.\n' "$job_id"
}

reuse_text_to_image() {
  local result="$out/text-to-image-result.json" job_id
  [[ -s "$result" && -s "$out/text-to-image-job.json" && -s "$out/text-to-image.png" ]] || return 1
  valid_ollama_response "$out/ollama-text-to-image" || return 1
  job_id="$(jq -er '.job_id' "$result")" || return 1
  jq -e --arg id "$job_id" '.job.id == $id and .job.state == "JOB_STATE_SUCCEEDED" and .job.lane == "JOB_LANE_GPU" and .job.result_meta.gpu_capacity_claim_id != null and .job.result_meta.gpu_uuid != null and (.job.result_meta.disk_job_filesystem_0_peak_bytes | tonumber > 0)' "$out/text-to-image-job.json" >/dev/null || return 1
  printf 'Reusing retained Image Tools text_to_image GPU receipt %s from this showcase attempt.\n' "$job_id"
}

mkdir -p "$out/backdrop" "$out/text-to-image"
if ! reuse_backdrop; then
  preflight "$out/backdrop"
  nvidia-smi --query-gpu=timestamp,memory.used,memory.free,utilization.gpu --format=csv -l 1 >>"$out/gpu-overlap.csv" &
  sampler_pid="$!"
  start_ollama "$out/ollama-backdrop"
  cd "$scenario_root/api"
  BACKDROP_STUDIO_REQUIRE_GPU=1 BACKDROP_STUDIO_GPU_SHOWCASE_OUT="$out/backdrop" \
    GOWORK=off go test -tags integration -count=1 -timeout 2m \
    -run '^TestGPUModelBackedShowcase$' ./integration
  cd "$repo_root"
  run_ollama_window "$out/ollama-backdrop"
else
  nvidia-smi --query-gpu=timestamp,memory.used,memory.free,utilization.gpu --format=csv -l 1 >>"$out/gpu-overlap.csv" &
  sampler_pid="$!"
fi

if ! reuse_text_to_image; then
  preflight "$out/text-to-image"
  model="$(jq -er '.model.id' "$out/text-to-image/preflight/selection.json")"
  start_ollama "$out/ollama-text-to-image"
  image-tools ai generate --model "$model" --require-gpu --no-reclaim --priority batch \
    --fallback-policy local_only --prompt "a small red cube on a neutral tabletop" \
    --negative "text, watermark, logo" --width 256 --height 256 --steps 8 \
    --cfg-scale 5 --seed 42 --json >"$out/text-to-image-result.json"
  text_job="$(jq -er '.job_id' "$out/text-to-image-result.json")"
  active_job_id="$text_job"
  if ! timeout --foreground --signal=TERM --kill-after=15s "$job_timeout" \
    image-tools jobs wait "$text_job" --json >"$out/text-to-image-wait.json"; then
    printf 'Image Tools text_to_image exceeded the %s hard runtime bound; cancelling exact job %s\n' "$job_timeout" "$text_job" >&2
    exit 1
  fi
  image-tools jobs get "$text_job" --json >"$out/text-to-image-job.json"
  jq -e '.job.state == "JOB_STATE_SUCCEEDED" and .job.lane == "JOB_LANE_GPU" and .job.result_meta.gpu_capacity_claim_id != null and .job.result_meta.gpu_uuid != null' "$out/text-to-image-job.json" >/dev/null
  image-tools jobs download "$text_job" --out "$out/text-to-image.png" >/dev/null
  test -s "$out/text-to-image.png"
  active_job_id=""
  run_ollama_window "$out/ollama-text-to-image"
fi

SAM2_PROFILE_OUT="$out/sam2-residency-profile" \
SAM2_PROFILE_MODEL=sam-2.1-base-plus \
SAM2_PROFILE_TRIALS=3 \
SAM2_PROFILE_DISK_PEAK_BYTES="$BACKDROP_STUDIO_GPU_DISK_PEAK_BYTES" \
LM17_FOOTPRINT_BASELINE="$LM17_FOOTPRINT_BASELINE" \
LM17_FOOTPRINT_KEEP_FILE="$LM17_FOOTPRINT_KEEP_FILE" \
make -C "$repo_root/scenarios/image-tools" sam2-residency-profile

preflight "$out/final"

python3 - "$out" <<'PY'
import json, sys
from datetime import datetime
from pathlib import Path

out = Path(sys.argv[1])
def instant(raw):
    return datetime.fromisoformat(raw.replace("Z", "+00:00"))
def overlaps(a, b, c, d):
    return max(a, c) < min(b, d)
def oom_kill(path):
    values = {}
    for line in path.read_text().splitlines():
        key, value = line.split()
        values[key] = int(value)
    if "oom_kill" not in values:
        raise SystemExit(f"host OOM counter is unavailable in {path}")
    return values["oom_kill"]

backdrop = json.loads((out / "backdrop/receipt.json").read_text())
text_job = json.loads((out / "text-to-image-job.json").read_text())["job"]
measured_peak = int((out / "text-to-image/preflight/measured-disk-peak-bytes.txt").read_text())
backdrop_peak = int(backdrop["disk_job_peak_bytes"])
text_peak = int(text_job["result_meta"]["disk_job_filesystem_0_peak_bytes"])
sam2_profile = json.loads((out / "sam2-residency-profile/profile.json").read_text())
sam2_speedup = float(sam2_profile.get("speedup", 0))
if sam2_speedup <= 1 or sam2_profile.get("gpu_faster") is not True:
    raise SystemExit("SAM2 CUDA did not demonstrate a measured speedup over the fresh CPU baseline")
sam2_peak = max(max(tier["disk_job_peak_bytes"]) for tier in sam2_profile["tiers"].values())
if measured_peak < max(backdrop_peak, text_peak, sam2_peak):
    raise SystemExit("preflight measured disk peak was lower than an admitted job's reservation")
checks = []
for name, job_start, job_finish, prefix in (
    ("backdrop", instant(backdrop["image_tools_started_at"]), instant(backdrop["image_tools_finished_at"]), "ollama-backdrop"),
    ("text_to_image", instant(text_job["started_at"]), instant(text_job["finished_at"]), "ollama-text-to-image"),
):
    request_start = instant((out / f"{prefix}.started").read_text().strip())
    request_finish = instant((out / f"{prefix}.finished").read_text().strip())
    request = json.loads((out / f"{prefix}.response.json").read_text())
    response = request.get("response", "")
    token_count = int(request.get("eval_count", 0))
    if not response.strip() or token_count <= 0:
        raise SystemExit(f"{name} Ollama request did not return generated tokens")
    if not overlaps(job_start, job_finish, request_start, request_finish):
        raise SystemExit(f"{name} GPU job did not overlap its successful Ollama request")
    checks.append({"operation": name, "overlap": True, "ollama_eval_count": token_count, "ollama_response_chars": len(response), "ollama_duration_ns": int(request.get("total_duration_ns", 0)), "gpu_started_at": job_start.isoformat(), "gpu_finished_at": job_finish.isoformat(), "ollama_started_at": request_start.isoformat(), "ollama_finished_at": request_finish.isoformat()})
if text_job["state"] != "JOB_STATE_SUCCEEDED":
    raise SystemExit("text_to_image job did not succeed")
oom_start = oom_kill(out / "backdrop/preflight/vmstat.txt")
oom_finish = oom_kill(out / "final/preflight/vmstat.txt")
if oom_finish != oom_start:
    raise SystemExit(f"host OOM-kill counter changed during showcase: {oom_start} -> {oom_finish}")
record = {
    "cpu_baseline_minutes": [18, 26],
    "backdrop_seconds": backdrop["elapsed_seconds"],
    "text_to_image_seconds": (instant(text_job["finished_at"]) - instant(text_job["started_at"])).total_seconds(),
    "sam2_cpu_median_seconds": sam2_profile["tiers"]["cpu"]["job_wall_median_seconds"],
    "sam2_cuda_median_seconds": sam2_profile["tiers"]["gpu"]["job_wall_median_seconds"],
    "sam2_speedup": sam2_speedup,
    "measured_preflight_disk_peak_bytes": measured_peak,
    "ollama_overlap": checks,
    "ollama_requests": [{"operation": item["operation"], "eval_count": item["ollama_eval_count"], "response_chars": item["ollama_response_chars"], "duration_ns": item["ollama_duration_ns"]} for item in checks],
    "sam2_profile": sam2_profile,
    "job_disk_reservations_bytes": {"backdrop": backdrop_peak, "text_to_image": text_peak, "sam2_max": sam2_peak},
    "oom_evidence": {"source": "/proc/vmstat:oom_kill", "before": oom_start, "after": oom_finish, "delta": oom_finish - oom_start},
    "oom_free": oom_finish == oom_start,
}
(out / "showcase.json").write_text(json.dumps(record, indent=2) + "\n")
print(json.dumps(record, indent=2))
PY

printf 'LM17 full showcase evidence retained at %s\n' "$out"
