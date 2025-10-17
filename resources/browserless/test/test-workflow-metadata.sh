#!/usr/bin/env bash
# Validates that the workflow interpreter extracts metadata and emits it for downstream tools.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_DIR="${APP_ROOT}/resources/browserless"

# Source logging utilities for consistent output
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

YAML_FILE="$TMP_DIR/metadata-test.yaml"
cat <<'YAML' > "$YAML_FILE"
workflow:
  name: "Metadata Validation Example"
  description: "Ensures the interpreter surfaces workflow metadata."
  version: "1.2.3"
  tags:
    - metadata
    - unit-test
  steps: []
YAML

TEST_HOME="$TMP_DIR/home"
mkdir -p "$TEST_HOME"

log::info "Running workflow interpreter metadata check..."

if ! APP_ROOT="$APP_ROOT" HOME="$TEST_HOME" "${BROWSERLESS_DIR}/lib/workflow/interpreter.sh" "$YAML_FILE" --session metadata_test >/dev/null 2>&1; then
    log::error "Workflow interpreter failed to execute test metadata workflow"
    exit 1
fi

metadata_file=$(find "$TEST_HOME/.vrooli/browserless/debug" -name metadata.json -type f -print -quit 2>/dev/null || true)

if [[ -z "$metadata_file" ]]; then
    log::error "Metadata file was not generated"
    exit 1
fi

expected_name="Metadata Validation Example"
actual_name=$(jq -r '.name' "$metadata_file")
if [[ "$actual_name" != "$expected_name" ]]; then
    log::error "Expected workflow name '$expected_name' but found '$actual_name'"
    exit 1
fi

expected_description="Ensures the interpreter surfaces workflow metadata."
actual_description=$(jq -r '.description' "$metadata_file")
if [[ "$actual_description" != "$expected_description" ]]; then
    log::error "Metadata description mismatch"
    exit 1
fi

expected_version="1.2.3"
actual_version=$(jq -r '.version' "$metadata_file")
if [[ "$actual_version" != "$expected_version" ]]; then
    log::error "Metadata version mismatch"
    exit 1
fi

tag_count=$(jq '.tags | length' "$metadata_file")
if [[ "$tag_count" -lt 2 ]]; then
    log::error "Expected metadata tags to include at least two entries"
    exit 1
fi

source_file=$(jq -r '.source_file' "$metadata_file")
if [[ "$source_file" != "$(realpath "$YAML_FILE")" ]]; then
    log::error "Metadata source_file does not match original workflow path"
    exit 1
fi

step_count=$(jq '.step_count' "$metadata_file")
if [[ "$step_count" -ne 0 ]]; then
    log::error "Expected step_count to be 0 for empty workflow"
    exit 1
fi

log::success "Workflow metadata verified: $metadata_file"
