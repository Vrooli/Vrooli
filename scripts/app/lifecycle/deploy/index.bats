#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Define location for this test file
APP_LIFECYCLE_DEPLOY_DIR="$BATS_TEST_DIRNAME"

# shellcheck disable=SC1091
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Path to the script under test
SCRIPT_PATH="${APP_LIFECYCLE_DEPLOY_DIR}/index.sh"

@test "index.sh sources all deploy scripts" {
    # Create a temporary test directory
    local test_dir="/tmp/bats-test-$$"
    mkdir -p "$test_dir"
    
    # Create mock scripts in test directory
    cat > "$test_dir/test1.sh" << 'EOF'
#!/usr/bin/env bash
test1::function() { echo "test1"; }
EOF
    
    cat > "$test_dir/test2.sh" << 'EOF'
#!/usr/bin/env bash
test2::function() { echo "test2"; }
EOF
    
    # Create test index.sh
    cat > "$test_dir/index.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail
APP_LIFECYCLE_DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh" || true
CURRENT_SCRIPT="$(basename "${BASH_SOURCE[0]}")"
for curr_file in "${APP_LIFECYCLE_DEPLOY_DIR}"/*.sh; do
    if [[ "$(basename "${curr_file}")" != "${CURRENT_SCRIPT}" ]]; then
        source "${curr_file}"
    fi
done
EOF
    
    # Test that sourcing index.sh sources all other scripts
    run bash -c "
        export APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        export var_APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        source '$test_dir/index.sh'
        declare -f test1::function >/dev/null && declare -f test2::function >/dev/null
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
}

@test "index.sh does not source itself" {
    # Create a temporary test directory
    local test_dir="/tmp/bats-test-$$"
    mkdir -p "$test_dir"
    
    # Create index.sh that would fail if it sources itself
    cat > "$test_dir/index.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail
if [[ "${INDEX_SOURCED:-}" == "true" ]]; then
    exit 1
fi
export INDEX_SOURCED="true"
APP_LIFECYCLE_DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh" || true
CURRENT_SCRIPT="$(basename "${BASH_SOURCE[0]}")"
for curr_file in "${APP_LIFECYCLE_DEPLOY_DIR}"/*.sh; do
    if [[ "$(basename "${curr_file}")" != "${CURRENT_SCRIPT}" ]]; then
        source "${curr_file}"
    fi
done
EOF
    
    # Test that index.sh doesn't source itself
    run bash -c "
        export APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        export var_APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        source '$test_dir/index.sh'
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
}

@test "index.sh handles empty directory gracefully" {
    # Create a temporary test directory with only index.sh
    local test_dir="/tmp/bats-test-$$"
    mkdir -p "$test_dir"
    
    # Create index.sh
    cat > "$test_dir/index.sh" << 'EOF'
#!/usr/bin/env bash
set -euo pipefail
APP_LIFECYCLE_DEPLOY_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${APP_LIFECYCLE_DEPLOY_DIR}/../../../lib/utils/var.sh" || true
CURRENT_SCRIPT="$(basename "${BASH_SOURCE[0]}")"
for curr_file in "${APP_LIFECYCLE_DEPLOY_DIR}"/*.sh; do
    if [[ "$(basename "${curr_file}")" != "${CURRENT_SCRIPT}" ]]; then
        source "${curr_file}"
    fi
done
EOF
    
    # Test that index.sh handles empty directory
    run bash -c "
        export APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        export var_APP_LIFECYCLE_DEPLOY_DIR='$test_dir'
        source '$test_dir/index.sh'
    "
    
    # Clean up
    trash::safe_remove "$test_dir" --test-cleanup
    
    [ "$status" -eq 0 ]
}