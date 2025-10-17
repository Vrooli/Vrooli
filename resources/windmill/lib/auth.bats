#!/usr/bin/env bats
# Tests for Windmill auth.sh functions

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/windmill/lib"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export WINDMILL_PORT="5681"
    export WINDMILL_BASE_URL="http://localhost:5681"
    export WINDMILL_ADMIN_EMAIL="admin@test.com"
    export WINDMILL_ADMIN_PASSWORD="admin123"
    export WINDMILL_API_TOKEN="wm_abc123def456"
    export WORKSPACE_NAME="demo"
    export USER_EMAIL="user@test.com"
    export USER_PASSWORD="userpass123"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    WINDMILL_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".token"*) echo "wm_admin_token_123" ;;
            *".user_id"*) echo "admin" ;;
            *".email"*) echo "admin@test.com" ;;
            *".role"*) echo "admin" ;;
            *".workspace"*) echo "demo" ;;
            *".message"*) echo "Success" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock openssl for password hashing
    openssl() {
        case "$*" in
            *"dgst"*)
                echo "hashed_password_abc123"
                ;;
            *"rand"*)
                echo "random_salt_def456"
                ;;
            *) echo "OPENSSL: $*" ;;
        esac
    }
    
    # Mock base64 for token encoding
    base64() {
        echo "encoded_token_123"
    }
    
    # Mock log functions
    
    # Mock Windmill utility functions
    windmill::is_running() { return 0; }
    windmill::is_healthy() { return 0; }
    
    # Load configuration and messages
    source "${WINDMILL_DIR}/config/defaults.sh"
    source "${WINDMILL_DIR}/config/messages.sh"
    windmill::export_config
    windmill::export_messages
    
    # Load the functions to test
    source "${WINDMILL_DIR}/lib/auth.sh"
}

# Test user authentication
@test "windmill::authenticate_user authenticates user credentials" {
    result=$(windmill::authenticate_user "$WINDMILL_ADMIN_EMAIL" "$WINDMILL_ADMIN_PASSWORD")
    
    [[ "$result" =~ "token" ]] || [[ "$result" =~ "authenticated" ]]
    [[ "$result" =~ "wm_admin_token_123" ]]
}

# Test authentication failure
@test "windmill::authenticate_user handles authentication failure" {
    # Override curl to return auth failure
    curl() {
        case "$*" in
            *"/api/auth/login"*)
                echo '{"error":"Invalid credentials"}'
                return 1
                ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    run windmill::authenticate_user "wrong@email.com" "wrongpass"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "Invalid credentials" ]] || [[ "$output" =~ "ERROR:" ]]
}

# Test user logout
@test "windmill::logout_user logs out user session" {
    result=$(windmill::logout_user "$WINDMILL_API_TOKEN")
    
    [[ "$result" =~ "logout" ]] || [[ "$result" =~ "Logged out" ]]
}

# Test user creation
@test "windmill::create_user creates new user account" {
    result=$(windmill::create_user "$USER_EMAIL" "$USER_PASSWORD" "user")
    
    [[ "$result" =~ "user" ]]
    [[ "$result" =~ "$USER_EMAIL" ]]
    [[ "$result" =~ "created" ]]
}

# Test user creation with invalid email
@test "windmill::create_user handles invalid email" {
    run windmill::create_user "invalid-email" "$USER_PASSWORD" "user"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "invalid" ]]
}

# Test user deletion
@test "windmill::delete_user removes user account" {
    export YES="yes"
    
    result=$(windmill::delete_user "user1")
    
    [[ "$result" =~ "delete" ]] || [[ "$result" =~ "removed" ]]
    [[ "$result" =~ "user1" ]]
}

# Test user deletion with confirmation
@test "windmill::delete_user handles user confirmation" {
    export YES="no"
    
    result=$(windmill::delete_user "user1")
    
    [[ "$result" =~ "cancelled" ]] || [[ "$result" =~ "aborted" ]]
}

# Test user list
@test "windmill::list_users shows workspace users" {
    result=$(windmill::list_users)
    
    [[ "$result" =~ "admin@test.com" ]]
    [[ "$result" =~ "user@test.com" ]]
    [[ "$result" =~ "admin" ]] || [[ "$result" =~ "user" ]]
}

# Test user role assignment
@test "windmill::assign_user_role assigns role to user" {
    result=$(windmill::assign_user_role "user1" "admin")
    
    [[ "$result" =~ "role" ]]
    [[ "$result" =~ "user1" ]]
    [[ "$result" =~ "admin" ]]
}

# Test user role removal
@test "windmill::remove_user_role removes role from user" {
    result=$(windmill::remove_user_role "user1" "admin")
    
    [[ "$result" =~ "role" ]]
    [[ "$result" =~ "removed" ]]
    [[ "$result" =~ "user1" ]]
}

# Test password change
@test "windmill::change_user_password changes user password" {
    result=$(windmill::change_user_password "admin" "newpassword123")
    
    [[ "$result" =~ "password" ]]
    [[ "$result" =~ "updated" ]] || [[ "$result" =~ "changed" ]]
}

# Test password reset
@test "windmill::reset_user_password resets user password" {
    result=$(windmill::reset_user_password "user1")
    
    [[ "$result" =~ "password" ]]
    [[ "$result" =~ "reset" ]]
    [[ "$result" =~ "user1" ]]
}

# Test API token creation
@test "windmill::create_api_token creates API token" {
    result=$(windmill::create_api_token "Test Token" "2025-01-15")
    
    [[ "$result" =~ "token" ]]
    [[ "$result" =~ "wm_new_token_789" ]]
    [[ "$result" =~ "Test Token" ]]
}

# Test API token list
@test "windmill::list_api_tokens shows user API tokens" {
    result=$(windmill::list_api_tokens)
    
    [[ "$result" =~ "token" ]]
    [[ "$result" =~ "API Token 1" ]]
}

# Test API token revocation
@test "windmill::revoke_api_token revokes API token" {
    result=$(windmill::revoke_api_token "wm_token_1")
    
    [[ "$result" =~ "revoke" ]] || [[ "$result" =~ "deleted" ]]
    [[ "$result" =~ "wm_token_1" ]]
}

# Test group creation
@test "windmill::create_group creates user group" {
    result=$(windmill::create_group "developers" "Development team")
    
    [[ "$result" =~ "group" ]]
    [[ "$result" =~ "developers" ]]
    [[ "$result" =~ "created" ]]
}

# Test group list
@test "windmill::list_groups shows workspace groups" {
    result=$(windmill::list_groups)
    
    [[ "$result" =~ "Administrators" ]]
    [[ "$result" =~ "Developers" ]]
}

# Test group deletion
@test "windmill::delete_group removes user group" {
    export YES="yes"
    
    result=$(windmill::delete_group "dev")
    
    [[ "$result" =~ "delete" ]] || [[ "$result" =~ "removed" ]]
    [[ "$result" =~ "dev" ]]
}

# Test user group assignment
@test "windmill::add_user_to_group adds user to group" {
    result=$(windmill::add_user_to_group "user1" "dev")
    
    [[ "$result" =~ "user1" ]]
    [[ "$result" =~ "dev" ]]
    [[ "$result" =~ "added" ]]
}

# Test user group removal
@test "windmill::remove_user_from_group removes user from group" {
    result=$(windmill::remove_user_from_group "user1" "dev")
    
    [[ "$result" =~ "user1" ]]
    [[ "$result" =~ "dev" ]]
    [[ "$result" =~ "removed" ]]
}

# Test permission management
@test "windmill::manage_permissions manages resource permissions" {
    result=$(windmill::manage_permissions "script" "user1" "read,write")
    
    [[ "$result" =~ "permission" ]]
    [[ "$result" =~ "script" ]]
    [[ "$result" =~ "user1" ]]
}

# Test permission list
@test "windmill::list_permissions shows resource permissions" {
    result=$(windmill::list_permissions)
    
    [[ "$result" =~ "permission" ]]
    [[ "$result" =~ "script" ]]
    [[ "$result" =~ "read" ]]
}

# Test current user info
@test "windmill::get_current_user returns current user information" {
    result=$(windmill::get_current_user)
    
    [[ "$result" =~ "admin@test.com" ]]
    [[ "$result" =~ "admin" ]]
    [[ "$result" =~ "demo" ]]
}

# Test user profile update
@test "windmill::update_user_profile updates user profile" {
    result=$(windmill::update_user_profile "admin" "New Name" "new@email.com")
    
    [[ "$result" =~ "profile" ]]
    [[ "$result" =~ "updated" ]]
    [[ "$result" =~ "admin" ]]
}

# Test session validation
@test "windmill::validate_session validates user session" {
    result=$(windmill::validate_session "$WINDMILL_API_TOKEN")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "session" ]]
}

# Test session validation with invalid token
@test "windmill::validate_session handles invalid session" {
    # Override curl to return invalid session
    curl() {
        case "$*" in
            *"/api/auth/me"*)
                echo '{"error":"Invalid token"}'
                return 1
                ;;
            *) echo "CURL: $*" ;;
        esac
    }
    
    run windmill::validate_session "invalid_token"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "invalid" ]] || [[ "$output" =~ "ERROR:" ]]
}

# Test password strength validation
@test "windmill::validate_password_strength validates password strength" {
    result=$(windmill::validate_password_strength "StrongPass123!")
    
    [[ "$result" =~ "strong" ]] || [[ "$result" =~ "valid" ]]
}

# Test weak password detection
@test "windmill::validate_password_strength detects weak passwords" {
    result=$(windmill::validate_password_strength "weak")
    
    [[ "$result" =~ "weak" ]] || [[ "$result" =~ "invalid" ]]
}

# Test email validation  
@test "windmill::validate_email validates email addresses" {
    result=$(windmill::validate_email "user@example.com")
    
    [[ "$result" =~ "valid" ]]
}

# Test invalid email detection
@test "windmill::validate_email detects invalid emails" {
    result=$(windmill::validate_email "invalid-email")
    
    [[ "$result" =~ "invalid" ]]
}

# Test user existence check
@test "windmill::user_exists checks if user exists" {
    result=$(windmill::user_exists "admin")
    
    [[ "$result" =~ "exists" ]] || [[ "$?" -eq 0 ]]
}

# Test non-existent user check
@test "windmill::user_exists handles non-existent user" {
    result=$(windmill::user_exists "nonexistent")
    
    [[ "$result" =~ "not found" ]] || [[ "$?" -eq 1 ]]
}

# Test admin user creation
@test "windmill::create_admin_user creates initial admin user" {
    result=$(windmill::create_admin_user "$WINDMILL_ADMIN_EMAIL" "$WINDMILL_ADMIN_PASSWORD")
    
    [[ "$result" =~ "admin" ]]
    [[ "$result" =~ "$WINDMILL_ADMIN_EMAIL" ]]
    [[ "$result" =~ "created" ]]
}

# Test authentication configuration
@test "windmill::configure_authentication sets up authentication" {
    result=$(windmill::configure_authentication)
    
    [[ "$result" =~ "authentication" ]] || [[ "$result" =~ "config" ]]
}

# Test SSO configuration
@test "windmill::configure_sso sets up single sign-on" {
    result=$(windmill::configure_sso "google")
    
    [[ "$result" =~ "SSO" ]] || [[ "$result" =~ "single sign-on" ]]
    [[ "$result" =~ "google" ]]
}

# Test LDAP configuration
@test "windmill::configure_ldap sets up LDAP authentication" {
    result=$(windmill::configure_ldap "ldap://server.com" "dc=example,dc=org")
    
    [[ "$result" =~ "LDAP" ]]
    [[ "$result" =~ "ldap://server.com" ]]
}

# Test multi-factor authentication
@test "windmill::setup_mfa configures multi-factor authentication" {
    result=$(windmill::setup_mfa "admin")
    
    [[ "$result" =~ "MFA" ]] || [[ "$result" =~ "multi-factor" ]]
    [[ "$result" =~ "admin" ]]
}

# Test audit log review
@test "windmill::review_audit_log reviews authentication audit log" {
    result=$(windmill::review_audit_log)
    
    [[ "$result" =~ "audit" ]] || [[ "$result" =~ "log" ]]
}

# Test security policy enforcement
@test "windmill::enforce_security_policy enforces security policies" {
    result=$(windmill::enforce_security_policy)
    
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "policy" ]]
}

# Test user lock/unlock
@test "windmill::lock_user locks user account" {
    result=$(windmill::lock_user "user1")
    
    [[ "$result" =~ "lock" ]] || [[ "$result" =~ "disabled" ]]
    [[ "$result" =~ "user1" ]]
}

@test "windmill::unlock_user unlocks user account" {
    result=$(windmill::unlock_user "user1")
    
    [[ "$result" =~ "unlock" ]] || [[ "$result" =~ "enabled" ]]
    [[ "$result" =~ "user1" ]]
}

# Test session management
@test "windmill::list_active_sessions shows active user sessions" {
    result=$(windmill::list_active_sessions)
    
    [[ "$result" =~ "session" ]] || [[ "$result" =~ "active" ]]
}

@test "windmill::terminate_session terminates user session" {
    result=$(windmill::terminate_session "session_123")
    
    [[ "$result" =~ "terminate" ]] || [[ "$result" =~ "ended" ]]
    [[ "$result" =~ "session_123" ]]
}

# Test bulk user operations
@test "windmill::bulk_create_users creates multiple users" {
    local users_file="/tmp/users.csv"
    echo "email,password,role" > "$users_file"
    echo "user1@test.com,pass1,user" >> "$users_file"
    echo "user2@test.com,pass2,admin" >> "$users_file"
    
    result=$(windmill::bulk_create_users "$users_file")
    
    [[ "$result" =~ "bulk" ]] || [[ "$result" =~ "created" ]]
    [[ "$result" =~ "2" ]] || [[ "$result" =~ "users" ]]
    
    trash::safe_remove "$users_file" --test-cleanup
}

# Test authentication metrics
@test "windmill::get_auth_metrics returns authentication metrics" {
    result=$(windmill::get_auth_metrics)
    
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "authentication" ]]
}

# Test backup/restore authentication data
@test "windmill::backup_auth_data backs up authentication data" {
    result=$(windmill::backup_auth_data "/tmp/auth_backup.json")
    
    [[ "$result" =~ "backup" ]]
    [[ "$result" =~ "/tmp/auth_backup.json" ]]
}

@test "windmill::restore_auth_data restores authentication data" {
    echo '{"users":[],"groups":[]}' > "/tmp/auth_backup.json"
    
    result=$(windmill::restore_auth_data "/tmp/auth_backup.json")
    
    [[ "$result" =~ "restore" ]]
    [[ "$result" =~ "/tmp/auth_backup.json" ]]
    
    trash::safe_remove "/tmp/auth_backup.json" --test-cleanup
}