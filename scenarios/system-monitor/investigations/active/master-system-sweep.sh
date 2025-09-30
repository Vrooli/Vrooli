#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Master System Sweep
# DESCRIPTION: Runs all core investigation scripts with safety timeouts and aggregates findings
# CATEGORY: meta
# TRIGGERS: full_diagnostic, manual_review
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-20
# LAST_MODIFIED: 2025-09-20
# VERSION: 1.0
# NOTE: Uses manifest tags to locate scripts tagged "core"

QUIET_MODE=${MASTER_SWEEP_QUIET:-1}
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MANIFEST="$(cd "${SCRIPT_DIR}/.." && pwd)/manifest.json"
RESULTS_ROOT="${SCRIPT_DIR}/../results"
SWEEP_ID="$(date +%Y%m%d_%H%M%S)_master-system-sweep"
export SWEEP_ID
OUTPUT_DIR="${RESULTS_ROOT}/${SWEEP_ID}"
SUMMARY_FILE="${OUTPUT_DIR}/summary.json"
mkdir -p "${OUTPUT_DIR}"

if [[ ! -f "${MANIFEST}" ]]; then
  echo "Manifest not found at ${MANIFEST}" >&2
  exit 1
fi

# Determine timeout budget per script from manifest settings with sane fallbacks
DEFAULT_TIMEOUT=$(jq -r '.settings.max_execution_time_seconds // 300' "${MANIFEST}")
if [[ -z "${DEFAULT_TIMEOUT}" || "${DEFAULT_TIMEOUT}" == "null" ]]; then
  DEFAULT_TIMEOUT=300
fi

# Collect scripts tagged as core and enabled
mapfile -t CORE_SCRIPTS < <(jq -r '.scripts | to_entries[]
  | select((.value.tags // []) | index("core"))
  | select((.value | has("enabled") | not) or (.value.enabled == true))
  | .key' "${MANIFEST}")

if [[ ${#CORE_SCRIPTS[@]} -eq 0 ]]; then
  echo "No core scripts defined in manifest" >&2
  exit 1
fi

jq -n '{
  sweep_id: env.SWEEP_ID,
  timestamp: (now | todateiso8601),
  scripts_run: [],
  failures: []
}' > "${SUMMARY_FILE}"

list_results_dirs() {
  find "${RESULTS_ROOT}" -maxdepth 1 -mindepth 1 -type d -printf '%f\n' 2>/dev/null | sort
}

run_script() {
  local script_id="$1"
  local script_meta_json
  script_meta_json=$(jq -c --arg id "$script_id" '.scripts[$id]' "${MANIFEST}")
  if [[ -z "${script_meta_json}" || "${script_meta_json}" == "null" ]]; then
    echo "Skipping unknown script ${script_id}" >&2
    return 1
  fi

  local script_path="${SCRIPT_DIR}/${script_id}.sh"
  if [[ ! -f "${script_path}" ]]; then
    echo "Script ${script_id} missing at ${script_path}" >&2
    jq --arg id "${script_id}" \
       '.failures += [{ id: $id, error: "missing_script" }]' "${SUMMARY_FILE}" > "${SUMMARY_FILE}.tmp" && mv "${SUMMARY_FILE}.tmp" "${SUMMARY_FILE}"
    return 1
  fi

  local timeout_override
  timeout_override=$(jq -r '.execution_time_estimate' <<<"${script_meta_json}")
  local timeout_seconds="${DEFAULT_TIMEOUT}"
  if [[ -n "${timeout_override}" && "${timeout_override}" != "null" ]]; then
    # Convert values like "45s" or "2m"
    if [[ ${timeout_override} =~ ^([0-9]+)s$ ]]; then
      timeout_seconds="${BASH_REMATCH[1]}"
    elif [[ ${timeout_override} =~ ^([0-9]+)m$ ]]; then
      timeout_seconds="$(( BASH_REMATCH[1] * 60 ))"
    elif [[ ${timeout_override} =~ ^[0-9]+$ ]]; then
      timeout_seconds="${timeout_override}"
    fi
  fi

  local log_dir="${OUTPUT_DIR}/${script_id}"
  mkdir -p "${log_dir}"
  local stdout_log="${log_dir}/stdout.log"
  local stderr_log="${log_dir}/stderr.log"

  echo "➤ Running ${script_id} (timeout ${timeout_seconds}s)"
  local start_ts start_epoch
  start_ts=$(date -Iseconds)
  start_epoch=$(date +%s)
  mapfile -t before_dirs < <(list_results_dirs)
  local run_exit=0
  if (
    cd "${SCRIPT_DIR}"
    timeout "${timeout_seconds}" bash "./${script_id}.sh"
  ) >"${stdout_log}" 2>"${stderr_log}"; then
    local end_ts end_epoch duration
    end_ts=$(date -Iseconds)
    end_epoch=$(date +%s)
    duration=$(( end_epoch - start_epoch ))
    jq --arg id "${script_id}" \
       --arg start "${start_ts}" \
       --arg end "${end_ts}" \
       --argjson duration "${duration:-0}" \
       --arg stdout "${stdout_log}" \
       --arg stderr "${stderr_log}" \
       --argjson meta "${script_meta_json}" \
       '.scripts_run += [{
          id: $id,
          started_at: $start,
          ended_at: $end,
          duration_seconds: $duration,
          status: "success",
          metadata: $meta,
          stdout_log: $stdout,
          stderr_log: $stderr
        }]' "${SUMMARY_FILE}" > "${SUMMARY_FILE}.tmp" && mv "${SUMMARY_FILE}.tmp" "${SUMMARY_FILE}"
    echo "✓ ${script_id} completed"
  else
    local exit_code=$?
    local end_ts end_epoch duration
    end_ts=$(date -Iseconds)
    end_epoch=$(date +%s)
    duration=$(( end_epoch - start_epoch ))
    echo "✗ ${script_id} failed (exit ${exit_code})" >&2
    jq --arg id "${script_id}" \
       --arg start "${start_ts}" \
       --arg end "${end_ts}" \
       --argjson duration "${duration:-0}" \
       --arg exit_code "${exit_code}" \
       --arg stderr "${stderr_log}" \
       '.failures += [{
          id: $id,
          started_at: $start,
          ended_at: $end,
          duration_seconds: $duration,
          exit_code: ($exit_code | tonumber?),
          stderr_log: $stderr
        }]' "${SUMMARY_FILE}" > "${SUMMARY_FILE}.tmp" && mv "${SUMMARY_FILE}.tmp" "${SUMMARY_FILE}"
    run_exit=1
  fi

  if [[ ${QUIET_MODE} -ne 0 ]]; then
    mapfile -t after_dirs < <(list_results_dirs)
    mapfile -t newly_created < <(comm -13 <(printf '%s\n' "${before_dirs[@]}" | sort) <(printf '%s\n' "${after_dirs[@]}" | sort))
    for new_dir in "${newly_created[@]}"; do
      [[ -z "${new_dir}" || "${new_dir}" == "${SWEEP_ID}" ]] && continue
      rm -rf "${RESULTS_ROOT}/${new_dir}" 2>/dev/null || true
    done
  fi

  return ${run_exit}
}

for script_id in "${CORE_SCRIPTS[@]}"; do
  run_script "${script_id}" || true
  # Push stdout snippet to console for quick awareness
  if [[ ${QUIET_MODE} -eq 0 ]]; then
    tail -n 10 "${OUTPUT_DIR}/${script_id}/stdout.log" 2>/dev/null || true
  fi
done

echo "📦 Sweep complete. Summary: ${SUMMARY_FILE}"
cat "${SUMMARY_FILE}"

if [[ ${QUIET_MODE} -ne 0 && -d "${OUTPUT_DIR}" ]]; then
  rm -rf "${OUTPUT_DIR}"
fi
