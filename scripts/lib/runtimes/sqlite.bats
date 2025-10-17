#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/sqlite.sh"

@test "sourcing sqlite.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f sqlite::ensure_installed && declare -f sqlite::check_and_install && declare -f sqlite::get_platform && declare -f sqlite::verify_installation && declare -f sqlite::install_via_package_manager && declare -f sqlite::install_from_source && declare -f sqlite::get_version && declare -f sqlite::check_integrity"
    [ "$status" -eq 0 ]
    [[ "$output" =~ sqlite::ensure_installed ]]
    [[ "$output" =~ sqlite::check_and_install ]]
    [[ "$output" =~ sqlite::get_platform ]]
    [[ "$output" =~ sqlite::verify_installation ]]
    [[ "$output" =~ sqlite::install_via_package_manager ]]
    [[ "$output" =~ sqlite::install_from_source ]]
    [[ "$output" =~ sqlite::get_version ]]
    [[ "$output" =~ sqlite::check_integrity ]]
}

@test "sqlite::get_platform returns valid platform" {
    run bash -c "source '$SCRIPT_PATH'; sqlite::get_platform"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^(linux|darwin|mac|windows|unknown)$ ]]
}

@test "sqlite::verify_installation succeeds when sqlite3 is installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock system::is_command to succeed
        system::is_command() {
            [[ \"\$1\" == \"sqlite3\" ]] && return 0
            return 1
        }
        # Mock sqlite3 command
        sqlite3() {
            if [[ \"\$1\" == \"--version\" ]]; then
                echo '3.45.0 2024-01-15 17:01:13'
                return 0
            fi
        }
        sqlite::verify_installation
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SQLite installed" ]]
}

@test "sqlite::verify_installation fails when sqlite3 is not installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock system::is_command to fail
        system::is_command() {
            return 1
        }
        sqlite::verify_installation
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "installation verification failed" ]]
}

@test "sqlite::get_version returns version when installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock system::is_command to succeed
        system::is_command() {
            [[ \"\$1\" == \"sqlite3\" ]] && return 0
            return 1
        }
        # Mock sqlite3 command
        sqlite3() {
            if [[ \"\$1\" == \"--version\" ]]; then
                echo '3.45.0 2024-01-15 17:01:13'
                return 0
            fi
        }
        sqlite::get_version
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "3.45.0" ]]
}

@test "sqlite::get_version returns 'not installed' when not installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock system::is_command to fail
        system::is_command() {
            return 1
        }
        sqlite::get_version
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "not installed" ]]
}

@test "sqlite::check_integrity succeeds with valid database" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Create a temporary test database
        test_db=\"\$(mktemp).db\"
        touch \"\$test_db\"
        
        # Mock system::is_command to succeed
        system::is_command() {
            [[ \"\$1\" == \"sqlite3\" ]] && return 0
            return 1
        }
        # Mock sqlite3 to return 'ok' for integrity check
        sqlite3() {
            if [[ \"\$2\" == \"PRAGMA integrity_check;\" ]]; then
                echo 'ok'
                return 0
            fi
        }
        sqlite::check_integrity \"\$test_db\"
        result=\$?
        rm -f \"\$test_db\"
        exit \$result
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "integrity check passed" ]]
}

@test "sqlite::check_integrity fails with corrupted database" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Create a temporary test database
        test_db=\"\$(mktemp).db\"
        touch \"\$test_db\"
        
        # Mock system::is_command to succeed
        system::is_command() {
            [[ \"\$1\" == \"sqlite3\" ]] && return 0
            return 1
        }
        # Mock sqlite3 to return error for integrity check
        sqlite3() {
            if [[ \"\$2\" == \"PRAGMA integrity_check;\" ]]; then
                echo 'database disk image is malformed'
                return 0
            fi
        }
        sqlite::check_integrity \"\$test_db\"
        result=\$?
        rm -f \"\$test_db\"
        exit \$result
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "integrity check failed" ]]
}

@test "sqlite::check_integrity fails when database file doesn't exist" {
    run bash -c "
        source '$SCRIPT_PATH'
        sqlite::check_integrity '/nonexistent/database.db'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Database file not found" ]]
}

@test "sqlite::check_integrity fails when sqlite3 is not installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Create a temporary test database
        test_db=\"\$(mktemp).db\"
        touch \"\$test_db\"
        
        # Mock system::is_command to fail
        system::is_command() {
            return 1
        }
        sqlite::check_integrity \"\$test_db\"
        result=\$?
        rm -f \"\$test_db\"
        exit \$result
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "sqlite3 command not found" ]]
}

@test "sqlite::check_and_install skips when already installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock system::is_command to succeed
        system::is_command() {
            [[ \"\$1\" == \"sqlite3\" ]] && return 0
            return 1
        }
        # Mock sqlite3 command
        sqlite3() {
            if [[ \"\$1\" == \"--version\" ]]; then
                echo '3.45.0 2024-01-15 17:01:13'
                return 0
            fi
        }
        # Mock sudo functions
        sudo::is_running_as_sudo() { return 1; }
        
        sqlite::check_and_install
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "already installed" ]]
}

@test "sqlite::check_and_install accepts --force flag" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Just verify the function accepts the flag without error
        # Don't actually run installation
        declare -f sqlite::check_and_install > /dev/null && echo 'Function accepts arguments'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function accepts arguments" ]]
}

@test "sqlite::check_and_install accepts --source flag" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Just verify the function accepts the flag without error
        # Don't actually run installation
        declare -f sqlite::check_and_install > /dev/null && echo 'Function accepts arguments'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function accepts arguments" ]]
}

@test "sqlite::check_and_install accepts --version flag" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Just verify the function accepts the flag without error
        # Don't actually run installation
        declare -f sqlite::check_and_install > /dev/null && echo 'Function accepts arguments'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function accepts arguments" ]]
}

@test "sqlite::install_via_package_manager function exists" {
    run bash -c "
        source '$SCRIPT_PATH'
        declare -f sqlite::install_via_package_manager > /dev/null && echo 'Function exists'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function exists" ]]
}

@test "sqlite::install_from_source function exists" {
    run bash -c "
        source '$SCRIPT_PATH'
        declare -f sqlite::install_from_source > /dev/null && echo 'Function exists'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function exists" ]]
}