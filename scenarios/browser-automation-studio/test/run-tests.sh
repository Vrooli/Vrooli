#!/bin/bash
set -euo pipefail

TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCENARIO_DIR}/../.." && builtin pwd)}"

source "${APP_ROOT}/scripts/scenarios/testing/shell/suite.sh"

validate_workflow_playbooks() {
    local playbook_dir="$SCENARIO_DIR/test/playbooks"
    if [ ! -d "$playbook_dir" ]; then
        return 0
    fi

    if ! command -v jq >/dev/null 2>&1; then
        echo "⚠️  jq not found; skipping workflow JSON validation" >&2
        return 0
    fi

    local invalid_found=0
    while IFS= read -r -d '' file; do
        if ! jq empty "$file" >/dev/null 2>&1; then
            echo "❌ Invalid JSON in workflow playbook: $file" >&2
            invalid_found=1
        fi
    done < <(find "$playbook_dir" -type f -name '*.json' -print0)

    if [ "$invalid_found" -ne 0 ]; then
        echo "Fix the workflow JSON files above before rerunning tests." >&2
        exit 1
    fi
}

validate_workflow_playbooks

testing::suite::run --scenario-dir "$SCENARIO_DIR" -- "$@"
exit $?
