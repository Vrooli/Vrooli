#!/usr/bin/env bats
# Tests for Python runtime setup script

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/python.sh"

@test "sourcing python.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f python::ensure_installed && declare -f python::check_and_install && declare -f python::get_python_command && declare -f python::get_version && declare -f python::verify_installation && declare -f python::install_system_packages && declare -f python::ensure_pip && declare -f python::ensure_venv_support"
    [ "$status" -eq 0 ]
    [[ "$output" =~ python::ensure_installed ]]
    [[ "$output" =~ python::check_and_install ]]
    [[ "$output" =~ python::get_python_command ]]
    [[ "$output" =~ python::get_version ]]
    [[ "$output" =~ python::verify_installation ]]
    [[ "$output" =~ python::install_system_packages ]]
    [[ "$output" =~ python::ensure_pip ]]
    [[ "$output" =~ python::ensure_venv_support ]]
}

@test "python::get_python_command returns python3 when available" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock python3 command
        python3() { echo 'Python 3.12.0'; }
        export -f python3
        command() { [[ \$1 == '-v' ]] && [[ \$2 == 'python3' ]] && return 0 || return 1; }
        export -f command
        python::get_python_command
    "
    [ "$status" -eq 0 ]
    [ "$output" = "python3" ]
}

@test "python::get_python_command returns empty when python not available" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock no python available
        command() { return 1; }
        export -f command
        python::get_python_command
    "
    [ "$status" -eq 0 ]
    [ "$output" = "" ]
}

@test "python::get_version extracts version from python output" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock python3 version output
        python3() { echo 'Python 3.12.5'; }
        export -f python3
        command() { [[ \$1 == '-v' ]] && [[ \$2 == 'python3' ]] && return 0 || return 1; }
        export -f command
        python::get_version python3
    "
    [ "$status" -eq 0 ]
    [ "$output" = "3.12.5" ]
}

@test "python::check_minimum_version compares versions correctly" {
    run bash -c "
        source '$SCRIPT_PATH'
        python::check_minimum_version '3.12.5' '3.8.0' && echo 'newer' || echo 'older'
    "
    [ "$status" -eq 0 ]
    [ "$output" = "newer" ]
}

@test "python::check_minimum_version rejects older versions" {
    run bash -c "
        source '$SCRIPT_PATH'
        python::check_minimum_version '3.7.0' '3.8.0' && echo 'newer' || echo 'older'
    "
    [ "$status" -eq 0 ]
    [ "$output" = "older" ]
}

@test "python::verify_installation detects working python" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock successful python installation
        python3() { 
            case \$1 in
                '--version') echo 'Python 3.12.5' ;;
                '-m') 
                    case \$2 in
                        'pip') echo 'pip 23.0.1' ;;
                        'venv') echo 'usage: python -m venv' ;;
                        *) return 1 ;;
                    esac
                    ;;
                *) return 1 ;;
            esac
        }
        export -f python3
        command() { [[ \$1 == '-v' ]] && [[ \$2 == 'python3' ]] && return 0 || return 1; }
        export -f command
        
        # Mock logging functions
        log::success() { echo \"SUCCESS: \$*\"; }
        log::info() { echo \"INFO: \$*\"; }
        log::error() { echo \"ERROR: \$*\"; }
        export -f log::success log::info log::error
        
        python::verify_installation
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "SUCCESS: Python installed: python3 (v3.12.5)" ]]
    [[ "$output" =~ "INFO: venv module: available" ]]
}

@test "python::create_venv creates virtual environment" {
    # Create temporary directory for testing
    local temp_dir
    temp_dir="$(mktemp -d)"
    
    run bash -c "
        source '$SCRIPT_PATH'
        cd '$temp_dir'
        
        # Mock python3 venv creation
        python3() { 
            if [[ \$1 == '-m' ]] && [[ \$2 == 'venv' ]]; then
                mkdir -p \"\$3/bin\"
                touch \"\$3/bin/pip\"
                return 0
            fi
            return 1
        }
        export -f python3
        command() { [[ \$1 == '-v' ]] && [[ \$2 == 'python3' ]] && return 0 || return 1; }
        export -f command
        
        # Mock logging functions
        log::success() { echo \"SUCCESS: \$*\"; }
        log::info() { echo \"INFO: \$*\"; }
        log::error() { echo \"ERROR: \$*\"; }
        export -f log::success log::info log::error
        
        python::create_venv test_venv
        [[ -d test_venv ]] && echo 'venv_created'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "INFO: Creating virtual environment at test_venv..." ]]
    [[ "$output" =~ "SUCCESS: Virtual environment created at test_venv" ]]
    [[ "$output" =~ "venv_created" ]]
    
    # Cleanup
    rm -rf "$temp_dir"
}

@test "python::install_requirements installs from requirements file" {
    # Create temporary directory for testing
    local temp_dir
    temp_dir="$(mktemp -d)"
    
    run bash -c "
        source '$SCRIPT_PATH'
        cd '$temp_dir'
        
        # Create test requirements file
        echo 'requests==2.28.0' > requirements.txt
        
        # Create mock venv structure
        mkdir -p .venv/bin
        
        # Mock venv pip
        cat > .venv/bin/pip << 'EOF'
#!/bin/bash
echo \"Installing from \$2\"
EOF
        chmod +x .venv/bin/pip
        
        # Mock python3 for venv creation
        python3() { 
            if [[ \$1 == '-m' ]] && [[ \$2 == 'venv' ]]; then
                # venv already exists
                return 0
            fi
            return 1
        }
        export -f python3
        
        # Mock logging functions
        log::success() { echo \"SUCCESS: \$*\"; }
        log::info() { echo \"INFO: \$*\"; }
        log::error() { echo \"ERROR: \$*\"; }
        export -f log::success log::info log::error
        
        python::install_requirements requirements.txt
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Installing from requirements.txt" ]]
    [[ "$output" =~ "SUCCESS: Requirements installed successfully" ]]
    
    # Cleanup
    rm -rf "$temp_dir"
}

# Integration tests (require actual environment)
@test "python::check_and_install with existing python skips installation" {
    # Skip if python3 not available in test environment
    command -v python3 >/dev/null 2>&1 || skip "python3 not available in test environment"
    
    run bash -c "
        source '$SCRIPT_PATH'
        
        # Mock logging functions to capture output
        log::header() { echo \"HEADER: \$*\"; }
        log::success() { echo \"SUCCESS: \$*\"; }
        log::info() { echo \"INFO: \$*\"; }
        log::warning() { echo \"WARNING: \$*\"; }
        log::error() { echo \"ERROR: \$*\"; }
        export -f log::header log::success log::info log::warning log::error
        
        # Mock flow and other functions that might not be available
        flow::can_run_sudo() { return 1; }
        export -f flow::can_run_sudo
        
        # Run the check (should skip installation since python exists)
        python::check_and_install
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "HEADER: Python Setup" ]]
    [[ "$output" =~ "Python version meets minimum requirements" ]]
}

@test "python runtime script is executable" {
    [ -x "$SCRIPT_PATH" ]
}

@test "python runtime script can be run directly" {
    # This tests the main execution guard at the bottom of the script
    run bash -c "$SCRIPT_PATH --help" 
    # Should not fail catastrophically - may return help or attempt installation
    [ "$status" -le 1 ]
}