#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="${REPO_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"

if [[ ! -d "${REPO_ROOT}/scenarios" ]]; then
  echo "No scenarios directory found at ${REPO_ROOT}/scenarios"
  exit 1
fi

declare -a offenders=()
declare -a allowlist=(
  "scenarios/deployment-manager/cli/deployment-manager"
  "scenarios/elo-swipe/cli/elo-swipe"
  "scenarios/knowledge-observatory/cli/knowledge-observatory"
  "scenarios/landing-page-business-suite/cli/landing-page-business-suite"
  "scenarios/local-info-scout/cli/local-info-scout"
  "scenarios/local-info-scout/cli/local-info-scout-cli"
  "scenarios/prd-control-tower/cli/cli"
  "scenarios/prd-control-tower/cli/prd-control-tower"
  "scenarios/prompt-manager/cli/pm"
  "scenarios/scenario-to-cloud/cli/scenario-to-cloud"
  "scenarios/scenario-to-desktop/cli/scenario-to-desktop"
  "scenarios/test-genie/cli/test-genie-cli"
  "scenarios/visited-tracker/cli/cli"
  "scenarios/vrooli-autoheal/cli/loop/vrooli-autoheal-loop"
)

is_allowlisted() {
  local relative="$1"
  for allowed in "${allowlist[@]}"; do
    if [[ "${relative}" == "${allowed}" ]]; then
      return 0
    fi
  done
  return 1
}

record_offender() {
  local file_path="$1"
  local reason="$2"
  local relative="${file_path#${REPO_ROOT}/}"
  if is_allowlisted "${relative}"; then
    return
  fi
  offenders+=("${relative} (${reason})")
}

scan_cli_dir() {
  local cli_dir="$1"
  if command -v file >/dev/null; then
    while IFS= read -r -d '' file_path; do
      mime_type="$(file -Lb --mime-type "${file_path}")"
      case "${mime_type}" in
        application/x-executable|application/x-pie-executable|application/x-sharedlib|application/vnd.microsoft.portable-executable|application/x-mach-binary)
          record_offender "${file_path}" "${mime_type}"
          ;;
      esac
    done < <(find "${cli_dir}" -type f -print0)
    return
  fi

  while IFS= read -r -d '' file_path; do
    case "${file_path}" in
      *.sh|*.ps1|*.go|*.mod|*.sum|*.md|*.txt|*.json|*.yaml|*.yml)
        continue
        ;;
    esac
    if [[ -x "${file_path}" ]]; then
      record_offender "${file_path}" "executable-bit"
    fi
  done < <(find "${cli_dir}" -type f -print0)
}

for cli_dir in "${REPO_ROOT}"/scenarios/*/cli; do
  [[ -d "${cli_dir}" ]] || continue
  scan_cli_dir "${cli_dir}"
done

if [[ ${#offenders[@]} -gt 0 ]]; then
  echo "Found scenario-local executable binaries under scenarios/*/cli/."
  echo "These cause CLI drift and should not live in scenario CLI folders."
  printf '  - %s\n' "${offenders[@]}"
  echo
  echo "Remediation:"
  echo "  1. Remove local binaries from scenarios/*/cli/."
  echo "  2. Reinstall canonical CLI binaries with each scenario's install script (for example: cd scenarios/swarm-manager/cli && ./install.sh)."
  exit 1
fi

echo "CLI binary location check passed."
