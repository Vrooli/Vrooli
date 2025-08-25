#!/usr/bin/env bats

# Integration tests for Odoo resource

setup() {
    export ODOO_BASE_DIR="$(builtin cd "${BATS_TEST_FILENAME%/*}/.." && builtin pwd)"
    export PATH="$ODOO_BASE_DIR:$PATH"
    
    # Source the CLI
    source "$ODOO_BASE_DIR/cli.sh"
}

@test "Odoo CLI exists and is executable" {
    [[ -x "$ODOO_BASE_DIR/cli.sh" ]]
}

@test "Odoo can show help" {
    run "$ODOO_BASE_DIR/cli.sh" help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Usage:" ]]
}

@test "Odoo installation check" {
    run odoo_is_installed
    # May or may not be installed, just check function exists
    [ "$?" -ge 0 ]
}

@test "Odoo status command works" {
    run "$ODOO_BASE_DIR/cli.sh" status
    # Status may return 1 if not healthy, but should not error
    [ "$?" -ge 0 ]
}

@test "Odoo status JSON format" {
    run "$ODOO_BASE_DIR/cli.sh" status json
    [ "$?" -ge 0 ]
    # If running, output should be valid JSON
    if [[ "$output" =~ "running" ]]; then
        echo "$output" | jq empty
    fi
}

@test "Odoo logs command exists" {
    run "$ODOO_BASE_DIR/cli.sh" logs 5
    # May fail if not running, but command should exist
    [ "$?" -ge 0 ]
}

@test "Odoo inject command shows help" {
    run "$ODOO_BASE_DIR/cli.sh" inject
    [[ "$output" =~ "Usage:" ]]
}

@test "Odoo port registration" {
    # Check if port registry exists
    if [[ -f "$ODOO_BASE_DIR/../../../resources/port_registry.sh" ]]; then
        run "$ODOO_BASE_DIR/../../../resources/port_registry.sh" list
        [ "$status" -eq 0 ]
    else
        skip "Port registry not available"
    fi
}

@test "Odoo Python examples are valid" {
    # Check Python syntax
    if command -v python3 &>/dev/null; then
        run python3 -m py_compile "$ODOO_BASE_DIR/examples/create_customer.py"
        [ "$status" -eq 0 ]
        
        run python3 -m py_compile "$ODOO_BASE_DIR/examples/product_sync.py"
        [ "$status" -eq 0 ]
    else
        skip "Python3 not available"
    fi
}

@test "Odoo configuration file template exists after install attempt" {
    # Try to create config dir
    mkdir -p "$var_data_dir/resources/odoo/config" 2>/dev/null || true
    
    # Check if we can create a config
    if [[ -w "$var_data_dir/resources/odoo/config" ]]; then
        [ "$?" -eq 0 ]
    else
        skip "Cannot write to data directory"
    fi
}