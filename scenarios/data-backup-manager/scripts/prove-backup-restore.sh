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
# BUNDLE_ROOT is the operator-facing destination bundle root: it holds the
# README/RECOVERY/manifest files, with the vanilla kopia repository nested under
# repositories/<slug>.kopia.
BUNDLE_ROOT="${CANARY_ROOT}/bundle"
RESTORE_DIR="${CANARY_ROOT}/restore"
OWNER="data-backup-manager"
# Slug-safe suffix: lowercase, only [a-z0-9-]. mktemp -d XXXXXX can yield mixed
# case, which the slug-safe destination-name contract rejects.
SUFFIX="$(basename "${CANARY_ROOT}" | tr '[:upper:]' '[:lower:]' | tr -c 'a-z0-9-' '-' | sed 's/-\{2,\}/-/g; s/^-//; s/-$//')"
TARGET_NAME="e2e-source-${SUFFIX}"
DEST_NAME="e2e-local-${SUFFIX}"
DESTINATION_ID=""
REPOSITORY_LOCATION=""
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

mkdir -p "${SOURCE_DIR}/nested" "${BUNDLE_ROOT}"
printf 'alpha\n' > "${SOURCE_DIR}/root.txt"
printf 'beta\n' > "${SOURCE_DIR}/nested/child.txt"

DEST_JSON="$(data-backup-manager destinations create \
  --name "${DEST_NAME}" \
  --backend filesystem \
  --location "${BUNDLE_ROOT}" \
  --cap-bytes 0 \
  --json)"
DESTINATION_ID="$(jq -r '.destination.id // .id' <<<"${DEST_JSON}")"
REPOSITORY_LOCATION="$(jq -r '.destination.repositoryLocation // .destination.repository_location // empty' <<<"${DEST_JSON}")"

# --- Bundle layout assertions -------------------------------------------------
# The bundle root must carry the human-facing explanatory files...
for f in README.txt RECOVERY.txt vrooli-backup-destination.json; do
  if [[ ! -f "${BUNDLE_ROOT}/${f}" ]]; then
    echo "expected bundle file missing: ${BUNDLE_ROOT}/${f}" >&2
    exit 1
  fi
done
# ...and the actual kopia repository must live under repositories/<slug>.kopia.
EXPECTED_REPO="${BUNDLE_ROOT}/repositories/${DEST_NAME}.kopia"
if [[ "${REPOSITORY_LOCATION}" != "${EXPECTED_REPO}" ]]; then
  echo "repository_location was ${REPOSITORY_LOCATION}; expected ${EXPECTED_REPO}" >&2
  exit 1
fi
if [[ ! -d "${REPOSITORY_LOCATION}" ]]; then
  echo "kopia repository dir not found at ${REPOSITORY_LOCATION}" >&2
  exit 1
fi
# The manifest must carry a vault secret REFERENCE (a path), never a secret
# value, and must be valid JSON.
MANIFEST_SECRET_REF="$(jq -r '.secret_ref // empty' "${BUNDLE_ROOT}/vrooli-backup-destination.json")"
if jq -e '.repository_path' "${BUNDLE_ROOT}/vrooli-backup-destination.json" >/dev/null; then :; else
  echo "manifest missing repository_path" >&2
  exit 1
fi
# secret_ref must be a vault REFERENCE PATH (not blank, not a value): a detached
# bundle has to tell the operator where the passphrase lives for standalone
# recovery. resource-kopia keeps passphrases under secret/resources/kopia/...
if [[ "${MANIFEST_SECRET_REF}" != secret/resources/kopia/* ]]; then
  echo "manifest secret_ref is not a vault reference path: '${MANIFEST_SECRET_REF}'" >&2
  exit 1
fi
# Defense-in-depth: no obvious passphrase/credential value text in the bundle.
if grep -RiE 'passphrase[-_ ]?value|BEGIN [A-Z ]*PRIVATE KEY|secret[-_ ]?access[-_ ]?key' "${BUNDLE_ROOT}/README.txt" "${BUNDLE_ROOT}/RECOVERY.txt" "${BUNDLE_ROOT}/vrooli-backup-destination.json" >/dev/null; then
  echo "bundle metadata appears to contain a secret value" >&2
  exit 1
fi

TARGET_JSON="$(data-backup-manager targets register \
  --owner "${OWNER}" \
  --name "${TARGET_NAME}" \
  --kind filesystem \
  --locator "${SOURCE_DIR}" \
  --json)"
TARGET_ID="$(jq -r '.target.id // .id' <<<"${TARGET_JSON}")"

# This canary is a self-contained engine proof: it binds only its own disposable
# temp target to a disposable temp destination and tears both down on exit. The
# default-coverage guard reflects the *host's* global backup posture, which is
# irrelevant to an isolated engine check, so bypass it here explicitly.
PLAN_JSON="$(data-backup-manager plans create \
  --name "e2e-manual-${SUFFIX}" \
  --targets "${TARGET_ID}" \
  --destinations "${DESTINATION_ID}" \
  --schedule "" \
  --keep-latest 3 \
  --allow-incomplete-coverage \
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

# Browsing the snapshot should surface the disposable source files.
BROWSE_JSON="$(data-backup-manager runs browse \
  --destination "${DESTINATION_ID}" \
  --snapshot "${SNAPSHOT_ID}" \
  --json 2>/dev/null || true)"
if [[ -n "${BROWSE_JSON}" ]]; then
  if ! grep -q 'root.txt' <<<"${BROWSE_JSON}"; then
    echo "snapshot browse did not list expected file root.txt" >&2
    echo "${BROWSE_JSON}" >&2
    exit 1
  fi
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
  --arg bundleRoot "${BUNDLE_ROOT}" \
  --arg repositoryLocation "${REPOSITORY_LOCATION}" \
  --arg manifestSecretRef "${MANIFEST_SECRET_REF}" \
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
    bundle_root: $bundleRoot,
    repository_location: $repositoryLocation,
    manifest_secret_ref: $manifestSecretRef,
    destination_id: $destinationId,
    target_id: $targetId,
    plan_id: $planId,
    run_status: $runStatus,
    outcome_status: $outcomeStatus,
    snapshot_id: $snapshotId,
    verify_status: $verifyStatus,
    restore_status: $restoreStatus,
    checksum: $checksum,
    bundle_files: ["README.txt","RECOVERY.txt","vrooli-backup-destination.json"],
    diff: "clean"
  }'
