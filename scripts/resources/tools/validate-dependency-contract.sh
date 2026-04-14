#!/usr/bin/env bash

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCENARIOS_DIR="${APP_ROOT}/scenarios"
RESOURCES_DIR="${APP_ROOT}/resources"

failures=0

info() {
    printf 'INFO: %s\n' "$*"
}

error() {
    printf 'ERROR: %s\n' "$*"
    failures=$((failures + 1))
}

mapfile -t scenario_files < <(find "${SCENARIOS_DIR}" -path '*/.vrooli/service.json' | sort)
mapfile -t canonical_resources < <(find "${RESOURCES_DIR}" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)
mapfile -t canonical_scenarios < <(find "${SCENARIOS_DIR}" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | sort)

contains_exact() {
    local needle="$1"
    shift
    local item
    for item in "$@"; do
        [[ "${item}" == "${needle}" ]] && return 0
    done
    return 1
}

for file in "${scenario_files[@]}"; do
    while IFS= read -r key; do
        [[ -z "${key}" ]] && continue
        error "${file}: legacy resource dependency bucket '${key}' is not allowed; use a flat keyed map"
    done < <(jq -r '(.dependencies.resources // {} | keys[]?) | select(. == "required" or . == "optional")' "${file}")

    while IFS= read -r key; do
        [[ -z "${key}" ]] && continue
        error "${file}: resource dependency key '${key}' uses deprecated CLI alias syntax"
    done < <(jq -r '(.dependencies.resources // {} | keys[]?) | select(startswith("resource-"))' "${file}")

    while IFS= read -r key; do
        [[ -z "${key}" ]] && continue
        error "${file}: resource dependency key '${key}' must use canonical kebab-case instead of underscores"
    done < <(jq -r '(.dependencies.resources // {} | keys[]?) | select(contains("_"))' "${file}")

    while IFS= read -r key; do
        [[ -z "${key}" ]] && continue
        if contains_exact "${key}" "${canonical_scenarios[@]}"; then
            error "${file}: scenario dependency '${key}' is declared under dependencies.resources instead of dependencies.scenarios"
        fi
    done < <(jq -r '(.dependencies.resources // {} | keys[]?)' "${file}")
done

if (( failures > 0 )); then
    printf 'Dependency contract validation failed with %d error(s).\n' "${failures}"
    exit 1
fi

printf 'Dependency contract validation passed for %d scenario service files.\n' "${#scenario_files[@]}"
