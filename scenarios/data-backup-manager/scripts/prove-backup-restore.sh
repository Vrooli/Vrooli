#!/usr/bin/env bash
set -euo pipefail

if [[ "${DBM_E2E_BACKUP:-}" != "1" ]]; then
  echo "DBM_E2E_BACKUP=1 is required to run the backup/verify/restore proof." >&2
  exit 2
fi

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 127
  fi
}

require_command data-backup-manager
require_command jq
require_command diff
require_command vrooli

SCENARIO_NAME="data-backup-manager"
CANARY_ROOT="${DBM_E2E_ROOT:-/tmp/dbm-e2e-$(date +%Y%m%d-%H%M%S)-$$}"
SOURCE_DIR="${CANARY_ROOT}/source"
REPO_DIR="${CANARY_ROOT}/repo"
RESTORE_DIR="${CANARY_ROOT}/restore"
OWNER="data-backup-manager"
SUFFIX="$(basename "${CANARY_ROOT}")"
TARGET_NAME="e2e-source-${SUFFIX}"
DESTINATION_ID=""
TARGET_ID=""
PLAN_ID=""

cleanup() {
  local status=$?
  set +e
  if [[ -n "${PLAN_ID}" ]]; then
    data-backup-manager plans delete --id "${PLAN_ID}" --json >/dev/null 2>&1
  fi
  if [[ -n "${TARGET_NAME}" ]]; then
    data-backup-manager targets deregister --owner "${OWNER}" --name "${TARGET_NAME}" --json >/dev/null 2>&1
  fi
  if [[ -n "${DESTINATION_ID}" ]]; then
    data-backup-manager destinations delete --id "${DESTINATION_ID}" --delete-repository true --json >/dev/null 2>&1
  fi
  if [[ "${DBM_E2E_KEEP:-}" != "1" ]]; then
    rm -rf "${CANARY_ROOT}"
  fi
  exit "${status}"
}
trap cleanup EXIT

vrooli scenario start "${SCENARIO_NAME}" >/dev/null

mkdir -p "${SOURCE_DIR}/nested" "${REPO_DIR}"
printf 'alpha\n' > "${SOURCE_DIR}/root.txt"
printf 'beta\n' > "${SOURCE_DIR}/nested/child.txt"

DEST_JSON="$(data-backup-manager destinations create \
  --name "e2e-local-${SUFFIX}" \
  --backend filesystem \
  --location "${REPO_DIR}" \
  --cap-bytes 0 \
  --json)"
DESTINATION_ID="$(jq -r '.destination.id // .id' <<<"${DEST_JSON}")"

TARGET_JSON="$(data-backup-manager targets register \
  --owner "${OWNER}" \
  --name "${TARGET_NAME}" \
  --kind filesystem \
  --locator "${SOURCE_DIR}" \
  --json)"
TARGET_ID="$(jq -r '.target.id // .id' <<<"${TARGET_JSON}")"

PLAN_JSON="$(data-backup-manager plans create \
  --name "e2e-manual-${SUFFIX}" \
  --targets "${TARGET_ID}" \
  --destinations "${DESTINATION_ID}" \
  --schedule "" \
  --keep-latest 3 \
  --json)"
PLAN_ID="$(jq -r '.plan.id // .id' <<<"${PLAN_JSON}")"

RUN_JSON="$(data-backup-manager runs trigger --plan "${PLAN_ID}" --json)"
RUN_STATUS="$(jq -r '.run.status // .status // empty' <<<"${RUN_JSON}")"
SNAPSHOT_ID="$(jq -r '.outcomes[0].snapshotId // .outcomes[0].snapshot_id // .run.outcomes[0].snapshotId // .run.outcomes[0].snapshot_id // empty' <<<"${RUN_JSON}")"
OUTCOME_STATUS="$(jq -r '.outcomes[0].status // .run.outcomes[0].status // empty' <<<"${RUN_JSON}")"

if [[ "${RUN_STATUS}" != "RUN_STATUS_COMPLETED" ]]; then
  echo "run status was ${RUN_STATUS}; expected RUN_STATUS_COMPLETED" >&2
  echo "${RUN_JSON}" >&2
  exit 1
fi
if [[ -z "${SNAPSHOT_ID}" || "${SNAPSHOT_ID}" == "null" ]]; then
  echo "run did not return a snapshot id" >&2
  echo "${RUN_JSON}" >&2
  exit 1
fi

VERIFY_JSON="$(data-backup-manager restores verify \
  --target "${TARGET_ID}" \
  --destination "${DESTINATION_ID}" \
  --snapshot "${SNAPSHOT_ID}" \
  --json)"
VERIFY_STATUS="$(jq -r '.restore.status // .status // empty' <<<"${VERIFY_JSON}")"
VERIFY_CHECKSUM="$(jq -r '.restore.checksum // .checksum // empty' <<<"${VERIFY_JSON}")"
if [[ "${VERIFY_STATUS}" != "RESTORE_STATUS_VERIFIED" || -z "${VERIFY_CHECKSUM}" || "${VERIFY_CHECKSUM}" == "null" ]]; then
  echo "verify did not produce verified status and checksum" >&2
  echo "${VERIFY_JSON}" >&2
  exit 1
fi

rm -rf "${RESTORE_DIR}"
RESTORE_JSON="$(data-backup-manager restores restore \
  --target "${TARGET_ID}" \
  --destination "${DESTINATION_ID}" \
  --snapshot "${SNAPSHOT_ID}" \
  --location "${RESTORE_DIR}" \
  --json)"
RESTORE_STATUS="$(jq -r '.restore.status // .status // empty' <<<"${RESTORE_JSON}")"
if [[ "${RESTORE_STATUS}" != "RESTORE_STATUS_RESTORED" ]]; then
  echo "restore status was ${RESTORE_STATUS}; expected RESTORE_STATUS_RESTORED" >&2
  echo "${RESTORE_JSON}" >&2
  exit 1
fi

diff -ru "${SOURCE_DIR}" "${RESTORE_DIR}" >/dev/null

jq -n \
  --arg canaryRoot "${CANARY_ROOT}" \
  --arg destinationId "${DESTINATION_ID}" \
  --arg targetId "${TARGET_ID}" \
  --arg planId "${PLAN_ID}" \
  --arg runStatus "${RUN_STATUS}" \
  --arg outcomeStatus "${OUTCOME_STATUS}" \
  --arg snapshotId "${SNAPSHOT_ID}" \
  --arg verifyStatus "${VERIFY_STATUS}" \
  --arg restoreStatus "${RESTORE_STATUS}" \
  --arg checksum "${VERIFY_CHECKSUM}" \
  '{
    canary_root: $canaryRoot,
    destination_id: $destinationId,
    target_id: $targetId,
    plan_id: $planId,
    run_status: $runStatus,
    outcome_status: $outcomeStatus,
    snapshot_id: $snapshotId,
    verify_status: $verifyStatus,
    restore_status: $restoreStatus,
    checksum: $checksum,
    diff: "clean"
  }'
