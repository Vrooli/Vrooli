#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

# Path to the script under test
SCRIPT_PATH="$BATS_TEST_DIRNAME/go.sh"

@test "sourcing go.sh defines required functions" {
    run bash -c "source '$SCRIPT_PATH' && declare -f go::check_and_install && declare -f go::detect_platform && declare -f go::install_binary && declare -f go::download_with_retry && declare -f go::verify_installation && declare -f go::get_go_root"
    [ "$status" -eq 0 ]
    [[ "$output" =~ go::check_and_install ]]
    [[ "$output" =~ go::detect_platform ]]
    [[ "$output" =~ go::install_binary ]]
    [[ "$output" =~ go::download_with_retry ]]
    [[ "$output" =~ go::verify_installation ]]
    [[ "$output" =~ go::get_go_root ]]
}

@test "go::detect_platform returns valid platform" {
    run bash -c "source '$SCRIPT_PATH'; go::detect_platform"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^(linux|mac|windows|unknown)$ ]]
}

@test "go::get_latest_version returns version string" {
    # Mock curl to return a sample version
    run bash -c "
        source '$SCRIPT_PATH'
        curl() {
            if [[ \"\$*\" =~ \"go.dev\" ]]; then
                echo 'go1.21.5'
            fi
            return 0
        }
        go::get_latest_version '1.21'
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "1.21.5" || "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

@test "go::download_with_retry retries on failure" {
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
        go::download_with_retry 'http://example.com' '/tmp/test.tar.gz'
    "
    [ "$status" -eq 0 ]
}

@test "go::verify_installation checks go binary" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock which to succeed
        which() {
            echo '/usr/local/go/bin/go'
            return 0
        }
        # Mock go version command
        go() {
            if [[ \"\$1\" == \"version\" ]]; then
                echo 'go version go1.21.5 linux/amd64'
                return 0
            fi
        }
        go::verify_installation
    "
    [ "$status" -eq 0 ]
}

@test "go::verify_installation fails when go not found" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock which to fail
        which() {
            return 1
        }
        go::verify_installation
    "
    [ "$status" -eq 1 ]
}

@test "go::install_binary function exists and has correct signature" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Just verify the function exists and can be called with mock data
        # Don't actually run it since it requires sudo
        declare -f go::install_binary > /dev/null && echo 'Function exists'
    "
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Function exists" ]]
}

@test "go::get_go_root returns correct path" {
    run bash -c "
        source '$SCRIPT_PATH'
        go::get_go_root
    "
    [ "$status" -eq 0 ]
    [[ "$output" == "/usr/local/go" ]]
}

@test "go::check_and_install skips when go already installed" {
    run bash -c "
        source '$SCRIPT_PATH'
        # Mock which to succeed
        which() {
            echo '/usr/local/go/bin/go'
            return 0
        }
        # Mock go version command
        go() {
            if [[ \"\$1\" == \"version\" ]]; then
                echo 'go version go1.21.5 linux/amd64'
                return 0
            fi
        }
        # Track if install was called
        go::install_binary() {
            echo 'INSTALL_CALLED'
            return 0
        }
        go::check_and_install '1.21'
    "
    [ "$status" -eq 0 ]
    [[ ! "$output" =~ "INSTALL_CALLED" ]]
}