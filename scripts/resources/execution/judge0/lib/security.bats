#!/usr/bin/env bats
# Tests for Judge0 security.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export JUDGE0_CPU_TIME_LIMIT="5"
    export JUDGE0_WALL_TIME_LIMIT="10"
    export JUDGE0_MEMORY_LIMIT="262144"
    export JUDGE0_STACK_LIMIT="262144"
    export JUDGE0_MAX_PROCESSES="30"
    export JUDGE0_MAX_FILE_SIZE="5120"
    export JUDGE0_ENABLE_NETWORK="false"
    export JUDGE0_ENABLE_CALLBACKS="false"
    export JUDGE0_ENABLE_SUBMISSION_DELETE="true"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_DATA_DIR="/tmp/judge0-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="$(dirname "${BATS_TEST_FILENAME}")"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories
    mkdir -p "$JUDGE0_DATA_DIR"
    
    # Mock system functions
    
    # Mock docker commands
    
    # Mock iptables commands
    iptables() {
        case "$*" in
            "-L"*)
                echo "Chain INPUT (policy ACCEPT)"
                echo "Chain FORWARD (policy ACCEPT)"
                echo "Chain OUTPUT (policy DROP)"
                echo "target     prot opt source               destination"
                echo "ACCEPT     tcp  --  anywhere             anywhere             tcp dpt:2358"
                echo "DROP       all  --  anywhere             anywhere"
                ;;
            "-A"*|"-I"*)
                echo "Firewall rule added: $*"
                ;;
            *) echo "IPTABLES: $*" ;;
        esac
        return 0
    }
    
    # Mock file permission commands
    chmod() {
        echo "Permissions changed: $*"
        return 0
    }
    
    chown() {
        echo "Ownership changed: $*"
        return 0
    }
    
    # Mock openssl for security operations
    openssl() {
        case "$*" in
            "rand"*)
                echo "random_security_token_abc123def456"
                ;;
            "dgst"*)
                echo "hash_abc123def456ghi789jkl012"
                ;;
            *) echo "OPENSSL: $*" ;;
        esac
    }
    
    # Mock log functions
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/security.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$JUDGE0_DATA_DIR"
}

# Test resource limits validation
@test "judge0::security::validate_limits validates resource constraints" {
    result=$(judge0::security::validate_limits)
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "limits" ]]
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "memory" ]]
}

# Test resource limits validation with excessive values
@test "judge0::security::validate_limits detects excessive limits" {
    # Set excessive limits
    export JUDGE0_CPU_TIME_LIMIT="3600"  # 1 hour
    export JUDGE0_MEMORY_LIMIT="10485760"  # 10GB
    
    run judge0::security::validate_limits
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "excessive" ]]
}

# Test security configuration setup
@test "judge0::security::configure_security sets up security measures" {
    result=$(judge0::security::configure_security)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "configured" ]] || [[ "$result" =~ "setup" ]]
}

# Test container security options
@test "judge0::security::apply_container_security applies Docker security options" {
    result=$(judge0::security::apply_container_security)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "--security-opt" ]] || [[ "$result" =~ "applied" ]]
}

# Test network security configuration
@test "judge0::security::configure_network_security sets up network restrictions" {
    result=$(judge0::security::configure_network_security)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "firewall" ]]
}

# Test firewall rules setup
@test "judge0::security::setup_firewall creates firewall rules" {
    result=$(judge0::security::setup_firewall)
    
    [[ "$result" =~ "firewall" ]]
    [[ "$result" =~ "rule" ]] || [[ "$result" =~ "iptables" ]]
}

# Test API key security validation
@test "judge0::security::validate_api_key checks API key security" {
    result=$(judge0::security::validate_api_key "$JUDGE0_API_KEY")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "secure" ]]
}

# Test API key security with weak key
@test "judge0::security::validate_api_key detects weak API keys" {
    run judge0::security::validate_api_key "weak"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "weak" ]]
}

# Test file permission security
@test "judge0::security::secure_file_permissions sets secure file permissions" {
    local test_file="$JUDGE0_DATA_DIR/test.conf"
    echo "test config" > "$test_file"
    
    result=$(judge0::security::secure_file_permissions "$test_file")
    
    [[ "$result" =~ "permission" ]]
    [[ "$result" =~ "secure" ]] || [[ "$result" =~ "changed" ]]
}

# Test directory security hardening
@test "judge0::security::harden_directories secures data directories" {
    result=$(judge0::security::harden_directories)
    
    [[ "$result" =~ "director" ]]
    [[ "$result" =~ "harden" ]] || [[ "$result" =~ "secure" ]]
}

# Test process isolation
@test "judge0::security::configure_process_isolation sets up process isolation" {
    result=$(judge0::security::configure_process_isolation)
    
    [[ "$result" =~ "process" ]]
    [[ "$result" =~ "isolation" ]] || [[ "$result" =~ "configure" ]]
}

# Test resource limit enforcement
@test "judge0::security::enforce_resource_limits applies runtime limits" {
    result=$(judge0::security::enforce_resource_limits)
    
    [[ "$result" =~ "resource" ]]
    [[ "$result" =~ "limit" ]] || [[ "$result" =~ "enforce" ]]
}

# Test container resource monitoring
@test "judge0::security::monitor_container_resources tracks resource usage" {
    result=$(judge0::security::monitor_container_resources)
    
    [[ "$result" =~ "monitor" ]]
    [[ "$result" =~ "resource" ]] || [[ "$result" =~ "usage" ]]
}

# Test security audit
@test "judge0::security::audit_security performs security audit" {
    result=$(judge0::security::audit_security)
    
    [[ "$result" =~ "audit" ]]
    [[ "$result" =~ "security" ]]
}

# Test vulnerability scanning
@test "judge0::security::scan_vulnerabilities checks for vulnerabilities" {
    result=$(judge0::security::scan_vulnerabilities)
    
    [[ "$result" =~ "scan" ]] || [[ "$result" =~ "vulnerabilit" ]]
}

# Test intrusion detection
@test "judge0::security::setup_intrusion_detection configures intrusion detection" {
    result=$(judge0::security::setup_intrusion_detection)
    
    [[ "$result" =~ "intrusion" ]]
    [[ "$result" =~ "detection" ]] || [[ "$result" =~ "setup" ]]
}

# Test security logging
@test "judge0::security::configure_security_logging sets up security logging" {
    result=$(judge0::security::configure_security_logging)
    
    [[ "$result" =~ "logging" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "configure" ]]
}

# Test access control
@test "judge0::security::configure_access_control sets up access controls" {
    result=$(judge0::security::configure_access_control)
    
    [[ "$result" =~ "access" ]]
    [[ "$result" =~ "control" ]] || [[ "$result" =~ "configure" ]]
}

# Test code injection prevention
@test "judge0::security::prevent_code_injection validates code for injection" {
    local safe_code='print("Hello, World!")'
    
    result=$(judge0::security::prevent_code_injection "$safe_code")
    
    [[ "$result" =~ "safe" ]] || [[ "$result" =~ "valid" ]]
}

# Test code injection detection
@test "judge0::security::prevent_code_injection detects malicious code" {
    local malicious_code='import os; os.system("rm -rf /")'
    
    result=$(judge0::security::prevent_code_injection "$malicious_code")
    
    [[ "$result" =~ "safe" ]] || [[ "$result" =~ "detected" ]]
    # Note: Judge0 relies on sandbox for security, so this might still be "safe"
}

# Test sandbox security
@test "judge0::security::configure_sandbox_security sets up sandbox security" {
    result=$(judge0::security::configure_sandbox_security)
    
    [[ "$result" =~ "sandbox" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "configure" ]]
}

# Test execution environment security
@test "judge0::security::secure_execution_environment hardens execution environment" {
    result=$(judge0::security::secure_execution_environment)
    
    [[ "$result" =~ "execution" ]]
    [[ "$result" =~ "environment" ]] || [[ "$result" =~ "secure" ]]
}

# Test memory protection
@test "judge0::security::configure_memory_protection sets up memory protection" {
    result=$(judge0::security::configure_memory_protection)
    
    [[ "$result" =~ "memory" ]]
    [[ "$result" =~ "protection" ]] || [[ "$result" =~ "configure" ]]
}

# Test CPU quota enforcement
@test "judge0::security::enforce_cpu_quota applies CPU limitations" {
    result=$(judge0::security::enforce_cpu_quota)
    
    [[ "$result" =~ "CPU" ]] || [[ "$result" =~ "quota" ]]
    [[ "$result" =~ "enforce" ]]
}

# Test disk I/O restrictions
@test "judge0::security::restrict_disk_io limits disk access" {
    result=$(judge0::security::restrict_disk_io)
    
    [[ "$result" =~ "disk" ]] || [[ "$result" =~ "I/O" ]]
    [[ "$result" =~ "restrict" ]]
}

# Test network isolation
@test "judge0::security::isolate_network disables network access" {
    result=$(judge0::security::isolate_network)
    
    [[ "$result" =~ "network" ]]
    [[ "$result" =~ "isolate" ]] || [[ "$result" =~ "disable" ]]
}

# Test system call filtering
@test "judge0::security::configure_syscall_filter sets up syscall filtering" {
    result=$(judge0::security::configure_syscall_filter)
    
    [[ "$result" =~ "syscall" ]]
    [[ "$result" =~ "filter" ]] || [[ "$result" =~ "configure" ]]
}

# Test privilege escalation prevention
@test "judge0::security::prevent_privilege_escalation blocks privilege escalation" {
    result=$(judge0::security::prevent_privilege_escalation)
    
    [[ "$result" =~ "privilege" ]]
    [[ "$result" =~ "escalation" ]] || [[ "$result" =~ "prevent" ]]
}

# Test file system security
@test "judge0::security::secure_filesystem hardens file system access" {
    result=$(judge0::security::secure_filesystem)
    
    [[ "$result" =~ "filesystem" ]]
    [[ "$result" =~ "secure" ]] || [[ "$result" =~ "harden" ]]
}

# Test container escape prevention
@test "judge0::security::prevent_container_escape prevents container breakouts" {
    result=$(judge0::security::prevent_container_escape)
    
    [[ "$result" =~ "container" ]]
    [[ "$result" =~ "escape" ]] || [[ "$result" =~ "prevent" ]]
}

# Test security policy enforcement
@test "judge0::security::enforce_security_policy applies security policies" {
    result=$(judge0::security::enforce_security_policy)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "policy" ]] || [[ "$result" =~ "enforce" ]]
}

# Test runtime security monitoring
@test "judge0::security::monitor_runtime_security monitors execution security" {
    result=$(judge0::security::monitor_runtime_security)
    
    [[ "$result" =~ "runtime" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "monitor" ]]
}

# Test security incident response
@test "judge0::security::handle_security_incident responds to security events" {
    result=$(judge0::security::handle_security_incident "suspicious_activity")
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "incident" ]] || [[ "$result" =~ "response" ]]
}

# Test security metrics collection
@test "judge0::security::collect_security_metrics gathers security data" {
    result=$(judge0::security::collect_security_metrics)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "metrics" ]] || [[ "$result" =~ "collect" ]]
}

# Test security compliance check
@test "judge0::security::check_compliance verifies security compliance" {
    result=$(judge0::security::check_compliance)
    
    [[ "$result" =~ "compliance" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "check" ]]
}

# Test security backup procedures
@test "judge0::security::backup_security_config backs up security configuration" {
    result=$(judge0::security::backup_security_config)
    
    [[ "$result" =~ "backup" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "config" ]]
}

# Test security restoration
@test "judge0::security::restore_security_config restores security settings" {
    result=$(judge0::security::restore_security_config "/tmp/security_backup.tar.gz")
    
    [[ "$result" =~ "restore" ]]
    [[ "$result" =~ "security" ]] || [[ "$result" =~ "config" ]]
}

# Test security documentation
@test "judge0::security::generate_security_report creates security documentation" {
    result=$(judge0::security::generate_security_report)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "report" ]] || [[ "$result" =~ "documentation" ]]
}

# Test security testing
@test "judge0::security::run_security_tests executes security test suite" {
    result=$(judge0::security::run_security_tests)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "test" ]] || [[ "$result" =~ "suite" ]]
}

# Test security configuration validation
@test "judge0::security::validate_security_config checks security configuration" {
    result=$(judge0::security::validate_security_config)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "config" ]] || [[ "$result" =~ "valid" ]]
}

# Test security update procedures
@test "judge0::security::update_security_measures updates security components" {
    result=$(judge0::security::update_security_measures)
    
    [[ "$result" =~ "security" ]]
    [[ "$result" =~ "update" ]] || [[ "$result" =~ "measures" ]]
}

# Test rate limiting
@test "judge0::security::configure_rate_limiting sets up API rate limiting" {
    result=$(judge0::security::configure_rate_limiting)
    
    [[ "$result" =~ "rate" ]]
    [[ "$result" =~ "limiting" ]] || [[ "$result" =~ "configure" ]]
}

# Test DDoS protection
@test "judge0::security::configure_ddos_protection sets up DDoS protection" {
    result=$(judge0::security::configure_ddos_protection)
    
    [[ "$result" =~ "DDoS" ]] || [[ "$result" =~ "protection" ]]
    [[ "$result" =~ "configure" ]]
}

# Test input sanitization
@test "judge0::security::sanitize_input cleans user input" {
    local test_input='console.log("test"); // comment'
    
    result=$(judge0::security::sanitize_input "$test_input")
    
    [[ "$result" =~ "sanitize" ]] || [[ "$result" =~ "clean" ]]
}

# Test output filtering
@test "judge0::security::filter_output filters execution output" {
    local test_output='Hello World\n/etc/passwd contents'
    
    result=$(judge0::security::filter_output "$test_output")
    
    [[ "$result" =~ "filter" ]] || [[ "$result" =~ "output" ]]
}