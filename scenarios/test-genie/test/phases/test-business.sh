#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/runtime.sh"

tg::require_command jq "jq is required to inspect requirement modules"
registry_dir="${TEST_GENIE_SCENARIO_DIR}/requirements"
tg::require_dir "$registry_dir"
tg::require_file "${registry_dir}/index.json"

tg::log_info "Checking requirement modules for completeness"
shopt -s nullglob
modules=("${registry_dir}"/*/module.json)
if [[ ${#modules[@]} -eq 0 ]]; then
  tg::fail "No requirement modules found under requirements/"
fi

for module_file in "${modules[@]}"; do
  module_name="$(basename "$(dirname "$module_file")")"
  tg::log_info "Inspecting module ${module_name}"
  requirement_count=$(jq '.requirements | length' "$module_file")
  if [[ "$requirement_count" -eq 0 ]]; then
    tg::fail "Module ${module_name} does not declare any requirements"
  fi

  missing_fields=$(jq '[.requirements[] | select((.id // "") == "" or (.title // "") == "" or (.criticality // "") == "" or (.status // "") == "")] | length' "$module_file")
  if [[ "$missing_fields" -gt 0 ]]; then
    tg::fail "Module ${module_name} contains requirements missing metadata"
  fi

  missing_validation=$(jq '[.requirements[] | select((.validation | length) == 0)] | length' "$module_file")
  if [[ "$missing_validation" -gt 0 ]]; then
    tg::fail "Module ${module_name} contains requirements without validation entries"
  fi
done
shopt -u nullglob

tg::log_success "Business validation complete"
