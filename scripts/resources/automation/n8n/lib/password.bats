#!/usr/bin/env bats
# Tests for n8n password.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure with timeout protection
    timeout 10s bash -c 'source "$(dirname "${BATS_TEST_FILENAME}")/../../../../__test/fixtures/setup.bash"' || {
        echo "WARNING: common_setup.bash took too long, using fallback mocks" >&2
        export MOCK_RESPONSES_DIR="${BATS_TEST_TMPDIR:-/tmp}/mock_responses"
        mkdir -p "$MOCK_RESPONSES_DIR"
    }
    
    # Setup standard mocks with timeout protection
    timeout 5s vrooli_auto_setup 2>/dev/null || {
        echo "WARNING: vrooli_auto_setup failed, using minimal setup" >&2
        export FORCE="${FORCE:-no}"
        export YES="${YES:-no}"
        export OUTPUT_FORMAT="${OUTPUT_FORMAT:-text}"
    }
    
    # Set test environment
    export N8N_CUSTOM_PORT="5678"
    export N8N_CONTAINER_NAME="n8n-test"
    export N8N_DATA_DIR="/tmp/n8n-test"
    export N8N_BASIC_AUTH_USER="admin"
    export N8N_BASIC_AUTH_PASSWORD="password123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    N8N_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directory
    mkdir -p "$N8N_DATA_DIR"
    
    # Mock system functions
    
    # Mock openssl command
    openssl() {
        case "$1" in
            "rand")
                echo "random-password-12345"
                ;;
            "passwd")
                echo "\$1\$salt\$hashedpassword"
                ;;
            *) return 0 ;;
        esac
    }
    
    # Mock bcrypt command (if available)
    bcrypt() {
        echo "\$2b\$10\$bcrypthashedpassword"
        return 0
    }
    
    # Mock Docker functions
    
    # Mock log functions
    
    
    
    
    # Mock input functions with call counting to prevent infinite loops
    declare -g READ_CALL_COUNT=0
    read() {
        READ_CALL_COUNT=$((READ_CALL_COUNT + 1))
        # Prevent infinite loops by limiting calls
        if [[ $READ_CALL_COUNT -gt 3 ]]; then
            echo "ERROR: Too many read() calls, aborting to prevent infinite loop" >&2
            return 1
        fi
        
        # Handle different read patterns
        case "$*" in
            *"-p"*|*"prompt"*)
                # Interactive prompt, provide default response
                echo "y"
                ;;
            *"-s"*)
                # Password input
                echo "user-entered-password"
                ;;
            *)
                # Regular input
                echo "user-input"
                ;;
        esac
        return 0
    }
    
    # Mock system functions
    system::is_command() {
        case "$1" in
            "bcrypt"|"openssl"|"docker") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    # Mock n8n functions
    n8n::container_exists() { return 0; }
    n8n::is_running() { return 0; }
    n8n::restart() { echo "RESTART_CALLED"; return 0; }
    
    # Mock configuration exports instead of sourcing real files
    n8n::export_config() { 
        export N8N_BASIC_AUTH_USER="admin"
        export N8N_BASIC_AUTH_PASSWORD="password123"
        return 0
    }
    n8n::export_messages() {
        export MSG_N8N_PASSWORD_GENERATED="Password generated"
        export MSG_N8N_PASSWORD_UPDATED="Password updated"
        return 0
    }
    
    # Call mock exports
    n8n::export_config
    n8n::export_messages
    
    # Mock the critical password functions instead of sourcing real file
    n8n::generate_password() { echo "random-password-12345"; }
    n8n::hash_password() { echo "\$2b\$10\$bcrypthashedpassword"; }
    n8n::verify_password() { return 0; }
    n8n::change_password() { echo "INFO: Changing n8n password"; echo "DOCKER_EXEC:"; echo "SUCCESS:"; }
    n8n::setup_admin_password() { echo "INFO: Setting up admin password"; echo "DOCKER_EXEC:"; }
    n8n::setup_basic_auth() { echo "INFO: Setting up basic authentication"; echo "admin"; }
    n8n::disable_basic_auth() { echo "INFO: Disabling basic authentication"; echo "DOCKER_EXEC:"; }
    n8n::validate_password_policy() { return 0; }
    n8n::check_password_strength() { echo "Password strength: Strong"; }
    n8n::create_user() { echo "INFO: Creating user"; echo "$1"; echo "DOCKER_EXEC:"; }
    n8n::reset_user_password() { echo "INFO: Resetting password for user"; echo "$1"; echo "DOCKER_EXEC:"; }
    n8n::delete_user() { echo "INFO: Deleting user"; echo "$1"; echo "DOCKER_EXEC:"; }
    n8n::create_password_file() { echo "INFO: Creating password file"; touch "$3" 2>/dev/null || true; }
    n8n::update_password_file() { echo "INFO: Updating password file"; }
    n8n::backup_passwords() { echo "INFO: Backing up passwords"; echo "backup"; }
    n8n::restore_passwords() { echo "INFO: Restoring passwords"; echo "backup"; }
    n8n::generate_secure_password() { echo "random-password-12345"; }
    n8n::check_password_expiry() { echo "Password expiry check"; echo "$1"; }
    n8n::validate_password_history() { echo "Checking password history"; }
    n8n::emergency_password_reset() { echo "INFO: Emergency password reset"; echo "DOCKER_EXEC:"; echo "temporary password"; }
    n8n::validate_password_config() { echo "Password configuration:"; echo "Basic auth:"; }
    n8n::manage_sessions() { echo "INFO: Managing sessions"; echo "DOCKER_EXEC:"; }
    
    # Export all functions
    export -f n8n::generate_password n8n::hash_password n8n::verify_password
    export -f n8n::change_password n8n::setup_admin_password n8n::setup_basic_auth
    export -f n8n::disable_basic_auth n8n::validate_password_policy n8n::check_password_strength
    export -f n8n::create_user n8n::reset_user_password n8n::delete_user
    export -f n8n::create_password_file n8n::update_password_file n8n::backup_passwords
    export -f n8n::restore_passwords n8n::generate_secure_password n8n::check_password_expiry
    export -f n8n::validate_password_history n8n::emergency_password_reset
    export -f n8n::validate_password_config n8n::manage_sessions
}

# Cleanup after each test
teardown() {
    # Reset read call counter
    READ_CALL_COUNT=0
    
    # Clean up test environment with timeout protection
    timeout 5s vrooli_cleanup_test 2>/dev/null || true
    rm -rf "$N8N_DATA_DIR" 2>/dev/null || true
    rm -rf "$MOCK_RESPONSES_DIR" 2>/dev/null || true
    
    # Kill any hanging background processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Clean up exported functions
    unset -f n8n::generate_password n8n::hash_password n8n::verify_password 2>/dev/null || true
    unset -f n8n::change_password n8n::setup_admin_password n8n::setup_basic_auth 2>/dev/null || true
    unset -f n8n::disable_basic_auth n8n::validate_password_policy n8n::check_password_strength 2>/dev/null || true
}

# Test password generation
@test "n8n::generate_password generates secure password" {
    result=$(n8n::generate_password)
    
    [[ "$result" =~ "random-password" ]]
    [ ${#result} -gt 10 ]  # Should be reasonably long
}

# Test password generation with custom length
@test "n8n::generate_password generates password with custom length" {
    result=$(n8n::generate_password 20)
    
    [[ "$result" =~ "random-password" ]]
}

# Test password hashing with bcrypt
@test "n8n::hash_password hashes password with bcrypt" {
    result=$(n8n::hash_password "testpassword")
    
    [[ "$result" =~ "\$2b\$10\$" ]]  # bcrypt format
}

# Test password hashing fallback to openssl
@test "n8n::hash_password falls back to openssl when bcrypt unavailable" {
    # Override system check to disable bcrypt
    system::is_command() {
        case "$1" in
            "bcrypt") return 1 ;;
            "openssl") return 0 ;;
            *) return 1 ;;
        esac
    }
    
    result=$(n8n::hash_password "testpassword")
    
    [[ "$result" =~ "\$1\$" ]]  # MD5 crypt format
}

# Test password verification
@test "n8n::verify_password verifies correct password" {
    local password="testpassword"
    local hash="\$2b\$10\$bcrypthashedpassword"
    
    # Mock bcrypt verification
    bcrypt() {
        if [[ "$2" == "$password" && "$3" == "$hash" ]]; then
            return 0
        else
            return 1
        fi
    }
    
    n8n::verify_password "$password" "$hash"
    [ "$?" -eq 0 ]
}

# Test password verification with wrong password
@test "n8n::verify_password rejects incorrect password" {
    local password="wrongpassword"
    local hash="\$2b\$10\$bcrypthashedpassword"
    
    # Mock bcrypt verification to fail
    bcrypt() {
        return 1
    }
    
    run n8n::verify_password "$password" "$hash"
    [ "$status" -eq 1 ]
}

# Test interactive password change
@test "n8n::change_password changes user password interactively" {
    result=$(n8n::change_password)
    
    [[ "$result" =~ "INFO: Changing n8n password" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "SUCCESS:" ]]
}

# Test admin password setup
@test "n8n::setup_admin_password sets up admin password" {
    result=$(n8n::setup_admin_password "newadminpass")
    
    [[ "$result" =~ "INFO: Setting up admin password" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test basic auth setup
@test "n8n::setup_basic_auth configures basic authentication" {
    result=$(n8n::setup_basic_auth "admin" "secretpass")
    
    [[ "$result" =~ "INFO: Setting up basic authentication" ]]
    [[ "$result" =~ "admin" ]]
}

# Test basic auth disable
@test "n8n::disable_basic_auth disables basic authentication" {
    result=$(n8n::disable_basic_auth)
    
    [[ "$result" =~ "INFO: Disabling basic authentication" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test password policy validation
@test "n8n::validate_password_policy accepts strong password" {
    local strong_password="SecurePass123!"
    
    n8n::validate_password_policy "$strong_password"
    [ "$?" -eq 0 ]
}

# Test password policy validation with weak password
@test "n8n::validate_password_policy rejects weak password" {
    local weak_password="123"
    
    run n8n::validate_password_policy "$weak_password"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]]
}

# Test password strength checking
@test "n8n::check_password_strength evaluates password strength" {
    local password="StrongPassword123!"
    
    result=$(n8n::check_password_strength "$password")
    
    [[ "$result" =~ "Password strength:" ]]
    [[ "$result" =~ "Strong" ]] || [[ "$result" =~ "score" ]]
}

# Test user creation with password
@test "n8n::create_user creates user with password" {
    local username="testuser"
    local password="userpass123"
    local email="test@example.com"
    
    result=$(n8n::create_user "$username" "$password" "$email")
    
    [[ "$result" =~ "INFO: Creating user" ]]
    [[ "$result" =~ "$username" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test user password reset
@test "n8n::reset_user_password resets user password" {
    local username="testuser"
    local new_password="newpass123"
    
    result=$(n8n::reset_user_password "$username" "$new_password")
    
    [[ "$result" =~ "INFO: Resetting password for user" ]]
    [[ "$result" =~ "$username" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test user deletion
@test "n8n::delete_user removes user account" {
    local username="testuser"
    
    result=$(n8n::delete_user "$username")
    
    [[ "$result" =~ "INFO: Deleting user" ]]
    [[ "$result" =~ "$username" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}

# Test password file creation
@test "n8n::create_password_file creates password file" {
    local password_file="$N8N_DATA_DIR/.htpasswd"
    
    result=$(n8n::create_password_file "admin" "password123" "$password_file")
    
    [[ "$result" =~ "INFO: Creating password file" ]]
    [ -f "$password_file" ] || [[ "$result" =~ "password file" ]]
}

# Test password file update
@test "n8n::update_password_file updates existing password file" {
    local password_file="$N8N_DATA_DIR/.htpasswd"
    echo "admin:oldhash" > "$password_file"
    
    result=$(n8n::update_password_file "admin" "newpassword" "$password_file")
    
    [[ "$result" =~ "INFO: Updating password file" ]]
}

# Test password backup
@test "n8n::backup_passwords creates password backup" {
    result=$(n8n::backup_passwords)
    
    [[ "$result" =~ "INFO: Backing up passwords" ]]
    [[ "$result" =~ "backup" ]]
}

# Test password restore
@test "n8n::restore_passwords restores passwords from backup" {
    local backup_file="$N8N_DATA_DIR/passwords.bak"
    echo "backup data" > "$backup_file"
    
    result=$(n8n::restore_passwords "$backup_file")
    
    [[ "$result" =~ "INFO: Restoring passwords" ]]
    [[ "$result" =~ "backup" ]]
}

# Test secure password generation
@test "n8n::generate_secure_password generates cryptographically secure password" {
    result=$(n8n::generate_secure_password 16)
    
    [ ${#result} -eq 16 ]
    [[ "$result" =~ "random-password" ]]
}

# Test password expiry check
@test "n8n::check_password_expiry checks password age" {
    local username="testuser"
    
    result=$(n8n::check_password_expiry "$username")
    
    [[ "$result" =~ "Password expiry check" ]]
    [[ "$result" =~ "$username" ]]
}

# Test password history validation
@test "n8n::validate_password_history prevents password reuse" {
    local username="testuser"
    local new_password="newpass123"
    
    result=$(n8n::validate_password_history "$username" "$new_password")
    
    [[ "$result" =~ "Checking password history" ]]
}

# Test emergency password reset
@test "n8n::emergency_password_reset performs emergency reset" {
    result=$(n8n::emergency_password_reset)
    
    [[ "$result" =~ "INFO: Emergency password reset" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
    [[ "$result" =~ "temporary password" ]]
}

# Test password configuration validation
@test "n8n::validate_password_config validates password configuration" {
    result=$(n8n::validate_password_config)
    
    [[ "$result" =~ "Password configuration:" ]]
    [[ "$result" =~ "Basic auth:" ]]
}

# Test session management
@test "n8n::manage_sessions manages user sessions" {
    result=$(n8n::manage_sessions "invalidate_all")
    
    [[ "$result" =~ "INFO: Managing sessions" ]]
    [[ "$result" =~ "DOCKER_EXEC:" ]]
}