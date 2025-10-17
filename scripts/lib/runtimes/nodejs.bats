#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/nodejs.sh"

@test "sourcing nodejs.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f nodejs::check_and_install && declare -f nodejs::detect_platform && declare -f nodejs::source_environment && declare -f nodejs::download_with_retry && declare -f nodejs::verify_installation"
    [ "$status" -eq 0 ]
    [[ "$output" =~ nodejs::check_and_install ]]
    [[ "$output" =~ nodejs::detect_platform ]]
    [[ "$output" =~ nodejs::source_environment ]]
    [[ "$output" =~ nodejs::download_with_retry ]]
    [[ "$output" =~ nodejs::verify_installation ]]
}

@test "nodejs::get_nvm_dir returns default directory when NVM_DIR not set" {
    run bash -c "source '$SCRIPT_PATH'; unset NVM_DIR; nodejs::get_nvm_dir"
    [ "$status" -eq 0 ]
    [[ "$output" == "$HOME/.nvm" ]]
}

@test "nodejs::get_nvm_dir returns NVM_DIR when set" {
    run bash -c "source '$SCRIPT_PATH'; export NVM_DIR=/custom/nvm; nodejs::get_nvm_dir"
    [ "$status" -eq 0 ]
    [[ "$output" == "/custom/nvm" ]]
}

@test "nodejs::detect_platform returns valid platform" {
    run bash -c "source '$SCRIPT_PATH'; nodejs::detect_platform"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^(linux|mac|windows|unknown)$ ]]
}

@test "nodejs::download_with_retry retries on failure" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock sleep to speed up test
        sleep() { :; }
        # Mock curl to fail twice then succeed
        curl_attempt=0
        curl() {
            ((curl_attempt++))
            if [[ \$curl_attempt -lt 3 ]]; then
                return 1
            fi
            echo 'success'
            return 0
        }
        nodejs::download_with_retry 'http://example.com'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Download attempt 1" ]]
    [[ "$output" =~ "Download attempt 2" ]]
    [[ "$output" =~ "Download attempt 3" ]]
    [[ "$output" =~ "success" ]]
}

@test "nodejs::download_with_retry fails after max attempts" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock sleep to speed up test
        sleep() { :; }
        # Mock curl to always fail
        curl() { return 1; }
        nodejs::download_with_retry 'http://example.com'
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Download failed after 3 attempts" ]]
}

@test "nodejs::verify_installation detects missing node" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock command to simulate node not found
        command() { return 1; }
        nodejs::verify_installation
    "
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Node.js installation verification failed" ]]
}

@test "nodejs::verify_installation succeeds with node installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock command and node
        command() { 
            [[ \"\$1\" == '-v' ]] && [[ \"\$2\" == 'node' ]] && return 0
            return 1
        }
        node() {
            [[ \"\$1\" == '--version' ]] && echo 'v20.0.0'
        }
        npm() {
            [[ \"\$1\" == '--version' ]] && echo '10.0.0'
        }
        nodejs::verify_installation '20'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Node.js installed successfully: v20.0.0" ]]
    [[ "$output" =~ "npm version: 10.0.0" ]]
}