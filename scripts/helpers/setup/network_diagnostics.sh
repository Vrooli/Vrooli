#!/usr/bin/env bash
# Comprehensive network diagnostics script
# Tests various network layers and protocols to identify connectivity issues
set -eo pipefail

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"

# Track test results
declare -A TEST_RESULTS
CRITICAL_FAILURE=false

# Test wrapper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    local is_critical="${3:-false}"
    
    log::info "Testing: $test_name"
    # Use explicit if-else to avoid script exit on test failure
    set +e  # Temporarily disable exit on error
    eval "$test_command" >/dev/null 2>&1
    local test_result=$?
    set -e  # Re-enable exit on error
    
    if [[ $test_result -eq 0 ]]; then
        TEST_RESULTS["$test_name"]="PASS"
        log::success "  ✓ $test_name"
        return 0
    else
        TEST_RESULTS["$test_name"]="FAIL"
        log::error "  ✗ $test_name"
        if [[ "$is_critical" == "true" ]]; then
            CRITICAL_FAILURE=true
        fi
        return 0  # Don't exit the script, just continue testing
    fi
}

# Main diagnostics function
network_diagnostics::run() {
    log::header "Running Comprehensive Network Diagnostics"
    
    # 1. Basic connectivity tests
    log::subheader "Basic Connectivity"
    run_test "IPv4 ping to 8.8.8.8" "ping -4 -c 1 -W 2 8.8.8.8" true
    run_test "IPv4 ping to google.com" "ping -4 -c 1 -W 2 google.com"
    run_test "IPv6 ping to google.com" "ping -6 -c 1 -W 2 google.com 2>/dev/null"
    
    # 2. DNS resolution tests
    log::subheader "DNS Resolution"
    run_test "DNS lookup (getent)" "getent hosts google.com"
    run_test "DNS lookup (nslookup)" "command -v nslookup >/dev/null && nslookup google.com"
    run_test "DNS via 8.8.8.8" "command -v dig >/dev/null && dig @8.8.8.8 google.com +short"
    
    # 3. TCP connectivity tests
    log::subheader "TCP Connectivity"
    run_test "TCP port 80 (HTTP)" "nc -zv -w 2 google.com 80"
    run_test "TCP port 443 (HTTPS)" "nc -zv -w 2 google.com 443"
    run_test "TCP port 22 (SSH)" "nc -zv -w 2 github.com 22"
    
    # 4. HTTP protocol tests
    log::subheader "HTTP Protocol Tests"
    
    # HTTP/1.0
    if command -v curl >/dev/null 2>&1; then
        run_test "HTTP/1.0 to google.com" "timeout 5 curl -s --http1.0 --connect-timeout 3 --max-time 5 http://google.com"
        run_test "HTTP/1.1 to google.com" "timeout 5 curl -s --http1.1 --connect-timeout 3 --max-time 5 http://google.com"
        
        # Raw HTTP test with netcat
        run_test "Raw HTTP with netcat" "echo -e 'GET / HTTP/1.0\r\n\r\n' | timeout 3 nc google.com 80 | grep -q 'HTTP'"
    fi
    
    # 5. HTTPS/TLS tests
    log::subheader "HTTPS/TLS Tests"
    
    if command -v curl >/dev/null 2>&1; then
        # Test different domains and TLS configurations
        run_test "HTTPS to google.com" "timeout 8 curl -s --http1.1 --connect-timeout 5 https://www.google.com"
        run_test "HTTPS to github.com" "timeout 8 curl -s --http1.1 --connect-timeout 5 https://github.com"
        
        # Test specific TLS versions
        run_test "TLS 1.2 to google.com" "timeout 8 curl -s --tlsv1.2 --http1.1 --connect-timeout 5 https://www.google.com"
        run_test "TLS 1.3 to google.com" "timeout 8 curl -s --tlsv1.3 --http1.1 --connect-timeout 5 https://www.google.com"
        run_test "TLS 1.2 to github.com" "timeout 8 curl -s --tlsv1.2 --http1.1 --connect-timeout 5 --max-time 8 https://github.com"
        
        # Test with different cipher suites
        run_test "HTTPS with modern ciphers" "timeout 8 curl -s --ciphers ECDHE+AESGCM --http1.1 --connect-timeout 5 https://www.google.com"
    fi
    
    # 6. Local service tests
    log::subheader "Local Services"
    
    # Test common local ports
    run_test "Localhost port 80" "nc -zv -w 1 localhost 80 2>/dev/null"
    run_test "Localhost port 8080" "nc -zv -w 1 localhost 8080 2>/dev/null"
    run_test "Localhost port 9200 (SearXNG)" "nc -zv -w 1 localhost 9200 2>/dev/null"
    
    # If port 9200 is open, test HTTP response
    if nc -z localhost 9200 2>/dev/null; then
        run_test "HTTP to localhost:9200" "timeout 3 curl -s --connect-timeout 2 --max-time 3 http://localhost:9200/stats"
    fi
    
    # 7. Network configuration checks
    log::subheader "Network Configuration"
    
    # Check MTU
    local mtu=$(ip link show | grep -E "^[0-9]+: enp" | head -1 | grep -oP 'mtu \K[0-9]+')
    log::info "Primary interface MTU: ${mtu:-unknown}"
    
    # Check for proxy settings
    if [[ -n "${http_proxy:-}" ]] || [[ -n "${https_proxy:-}" ]]; then
        log::warning "Proxy detected: http_proxy=${http_proxy:-none}, https_proxy=${https_proxy:-none}"
    else
        log::info "No proxy configured"
    fi
    
    # Check TCP offload settings
    if command -v ethtool >/dev/null 2>&1; then
        local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1)
        if [[ -n "$primary_iface" ]]; then
            local tso_status=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}')
            log::info "TCP segmentation offload on $primary_iface: ${tso_status:-unknown}"
        fi
    fi
    
    # 8. Automatic Network Optimizations
    log::subheader "Automatic Network Optimizations"
    
    # Check if we have TLS issues that might be fixable
    local has_tls_issues=false
    local has_tcp_issues=false
    
    # Safely iterate over test results
    if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
        for test_name in "${!TEST_RESULTS[@]}"; do
            if [[ "$test_name" =~ TLS|HTTPS ]] && [[ "${TEST_RESULTS[$test_name]}" != "PASS" ]]; then
                has_tls_issues=true
            fi
            if [[ "$test_name" =~ TCP ]] && [[ "${TEST_RESULTS[$test_name]}" != "PASS" ]]; then
                has_tcp_issues=true
            fi
        done
    fi
    
    # Attempt automatic fixes if we have permission and issues exist
    if [[ "$has_tls_issues" == "true" ]] || [[ "$has_tcp_issues" == "true" ]]; then
        log::info "Network issues detected. Attempting automatic optimizations..."
        
        # 1. Flush DNS cache (safe)
        log::info "Flushing DNS cache..."
        if command -v systemctl >/dev/null 2>&1; then
            if systemctl is-active systemd-resolved >/dev/null 2>&1; then
                if sudo -n systemctl flush-dns 2>/dev/null || systemd-resolve --flush-caches 2>/dev/null; then
                    log::success "  ✓ DNS cache flushed"
                else
                    log::info "  → DNS cache flush requires sudo (skipped)"
                fi
            fi
        fi
        
        # 2. Test with TCP optimizations
        local primary_iface=$(ip route | grep default | awk '{print $5}' | head -1)
        if [[ -n "$primary_iface" ]] && command -v ethtool >/dev/null 2>&1; then
            log::info "Testing TCP segmentation offload settings on $primary_iface..."
            
            # Check current settings
            local current_tso=$(ethtool -k "$primary_iface" 2>/dev/null | grep "tcp-segmentation-offload" | awk '{print $2}')
            local current_gso=$(ethtool -k "$primary_iface" 2>/dev/null | grep "generic-segmentation-offload" | awk '{print $2}')
            
            if [[ "$current_tso" == "on" ]] || [[ "$current_gso" == "on" ]]; then
                log::warning "  → TCP/GSO offload is enabled (may cause TLS issues)"
                log::info "  → To test fix: sudo ethtool -K $primary_iface tso off gso off"
                log::info "  → To make permanent: add to /etc/network/interfaces or systemd-networkd"
                
                # If we have sudo without password, try temporarily
                if sudo -n true 2>/dev/null; then
                    log::info "Temporarily testing with offload disabled..."
                    if sudo ethtool -K "$primary_iface" tso off gso off 2>/dev/null; then
                        log::info "  → Testing HTTPS with offload disabled..."
                        if timeout 5 curl -s --http1.1 --connect-timeout 3 https://www.google.com >/dev/null 2>&1; then
                            log::success "  ✓ HTTPS works with TCP offload disabled!"
                            log::warning "  → Consider making this permanent"
                        else
                            log::info "  → Still having issues, reverting..."
                        fi
                        # Revert changes
                        sudo ethtool -K "$primary_iface" tso on gso on 2>/dev/null || true
                    fi
                fi
            else
                log::success "  ✓ TCP offload already optimized"
            fi
        fi
        
        # 3. Test different MTU sizes for fragmentation issues
        if [[ "$has_tcp_issues" == "true" ]]; then
            log::info "Testing MTU fragmentation..."
            local current_mtu=$(ip link show "$primary_iface" 2>/dev/null | grep -oP 'mtu \K[0-9]+')
            if [[ -n "$current_mtu" ]] && [[ "$current_mtu" -gt 1400 ]]; then
                # Test if smaller packets work better
                if ping -M do -s 1200 -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
                    log::success "  ✓ Small packets (1200 bytes) work fine"
                else
                    log::warning "  → Possible MTU/fragmentation issues detected"
                    log::info "  → Consider testing with smaller MTU: sudo ip link set dev $primary_iface mtu 1400"
                fi
            fi
        fi
        
        # 4. Test alternative DNS servers
        log::info "Testing alternative DNS servers..."
        if timeout 3 dig @1.1.1.1 google.com +short >/dev/null 2>&1; then
            log::success "  ✓ Cloudflare DNS (1.1.1.1) works"
        else
            log::warning "  → Cloudflare DNS issues detected"
        fi
        
        if timeout 3 dig @8.8.8.8 google.com +short >/dev/null 2>&1; then
            log::success "  ✓ Google DNS (8.8.8.8) works"
        else
            log::warning "  → Google DNS issues detected"
        fi
        
        # 5. Check for IPv6 interference
        if [[ "${TEST_RESULTS["IPv6 ping to google.com"]:-FAIL}" == "PASS" ]]; then
            log::info "Testing IPv6/IPv4 preference..."
            # Test if forcing IPv4 helps with TLS
            if timeout 5 curl -4 -s --http1.1 --connect-timeout 3 https://www.google.com >/dev/null 2>&1; then
                log::success "  ✓ IPv4-only HTTPS works"
                if ! timeout 5 curl -6 -s --http1.1 --connect-timeout 3 https://www.google.com >/dev/null 2>&1; then
                    log::warning "  → IPv6 HTTPS has issues, prefer IPv4"
                fi
            fi
        fi
        
        # 6. Test with different User-Agent (some sites block default curl)
        log::info "Testing with browser User-Agent..."
        if timeout 5 curl -s --user-agent "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36" --http1.1 --connect-timeout 3 https://github.com >/dev/null 2>&1; then
            log::success "  ✓ GitHub works with browser User-Agent"
            log::info "  → Some sites block default curl User-Agent"
        fi
        
        # 7. Quick retest of critical issues after optimizations
        log::info "Re-testing critical HTTPS connections after optimizations..."
        local retest_passed=0
        local retest_total=0
        
        
        # Use shorter timeouts and more robust error handling for retesting
        if [[ "${TEST_RESULTS["HTTPS to google.com"]:-FAIL}" == "FAIL" ]]; then
            retest_total=$((retest_total + 1))
            log::info "  → Re-testing Google HTTPS..."
            set +e
            timeout 3 curl -s --http1.1 --connect-timeout 2 --max-time 3 https://www.google.com >/dev/null 2>&1
            local google_retest=$?
            set -e
            if [[ $google_retest -eq 0 ]]; then
                log::success "  ✓ Google HTTPS now works!"
                retest_passed=$((retest_passed + 1))
                TEST_RESULTS["HTTPS to google.com (retest)"]="PASS"
            else
                log::info "  → Google HTTPS still failing"
                TEST_RESULTS["HTTPS to google.com (retest)"]="FAIL"
            fi
        else
            log::info "  → Google HTTPS already working, skipping retest"
        fi
        
        local github_https_status="${TEST_RESULTS["HTTPS to github.com"]:-FAIL}"
        if [[ "$github_https_status" == "FAIL" ]]; then
            retest_total=$((retest_total + 1))
            log::info "  → Re-testing GitHub HTTPS..."
            set +e
            timeout 3 curl -s --http1.1 --connect-timeout 2 --max-time 3 https://github.com >/dev/null 2>&1
            local github_retest=$?
            set -e
            if [[ $github_retest -eq 0 ]]; then
                log::success "  ✓ GitHub HTTPS now works!"
                retest_passed=$((retest_passed + 1))
                TEST_RESULTS["HTTPS to github.com (retest)"]="PASS"
            else
                log::info "  → GitHub HTTPS still failing"
                TEST_RESULTS["HTTPS to github.com (retest)"]="FAIL"
            fi
        else
            log::info "  → GitHub HTTPS already working, skipping retest"
        fi
        
        
        if [[ $retest_total -gt 0 ]]; then
            if [[ $retest_passed -eq $retest_total ]]; then
                log::success "🎉 All HTTPS issues resolved by automatic optimizations!"
            elif [[ $retest_passed -gt 0 ]]; then
                log::warning "⚠️ Some HTTPS issues resolved ($retest_passed/$retest_total)"
            else
                log::info "→ No improvement from automatic optimizations"
            fi
        else
            log::info "→ No failed HTTPS tests to recheck"
        fi
        
        
    else
        log::success "No network issues detected, skipping optimizations"
    fi
    
    # 9. Summary
    log::header "Network Diagnostics Summary"
    
    local total_tests=${#TEST_RESULTS[@]}
    local passed_tests=0
    local failed_tests=0
    
    for result in "${TEST_RESULTS[@]}"; do
        case "$result" in
            "PASS") passed_tests=$((passed_tests + 1)) ;;
            *) failed_tests=$((failed_tests + 1)) ;;
        esac
    done
    
    log::info "Total tests: $total_tests"
    log::success "Passed: $passed_tests"
    log::error "Failed: $failed_tests"
    
    # Analyze results and provide diagnosis
    log::subheader "Diagnosis"
    
    # Check for specific patterns
    local has_basic_connectivity="${TEST_RESULTS["IPv4 ping to 8.8.8.8"]:-FAIL}"
    local has_dns="${TEST_RESULTS["DNS lookup (getent)"]:-FAIL}"
    local has_tcp="${TEST_RESULTS["TCP port 80 (HTTP)"]:-FAIL}"
    local has_http="${TEST_RESULTS["HTTP/1.0 to google.com"]:-FAIL}"
    local has_https_google="${TEST_RESULTS["HTTPS to google.com"]:-FAIL}"
    local has_https_github="${TEST_RESULTS["HTTPS to github.com"]:-FAIL}"
    local has_tls12_google="${TEST_RESULTS["TLS 1.2 to google.com"]:-FAIL}"
    local has_tls13_google="${TEST_RESULTS["TLS 1.3 to google.com"]:-FAIL}"
    
    # Detailed analysis based on test patterns
    if [[ "$has_basic_connectivity" == "PASS" ]] && [[ "$has_dns" == "PASS" ]]; then
        if [[ "$has_tcp" == "PASS" ]] && [[ "$has_http" == "PASS" ]]; then
            
            # Analyze HTTPS patterns
            if [[ "$has_https_google" == "FAIL" ]] && [[ "$has_https_github" == "FAIL" ]]; then
                if [[ "$has_tls12_google" == "PASS" ]] || [[ "$has_tls13_google" == "PASS" ]]; then
                    log::warning "PARTIAL TLS ISSUE: Domain or TLS version negotiation problems"
                    log::info "- Basic networking works (ping, DNS, TCP, HTTP)"
                    log::info "- HTTPS fails with default settings"
                    log::info "- But specific TLS versions work"
                    log::info "- This suggests TLS version negotiation or cipher suite issues"
                    log::info ""
                    log::info "Likely causes:"
                    log::info "1. TLS version negotiation problems"
                    log::info "2. Cipher suite incompatibility"  
                    log::info "3. TCP segmentation offload (TSO) issues"
                    log::info "4. Some sites require specific User-Agent headers"
                else
                    log::error "CRITICAL: Complete TLS/HTTPS failure"
                    log::warning "- All HTTPS connections fail including forced TLS versions"
                    log::warning "- This suggests deep packet inspection or driver issues"
                fi
            elif [[ "$has_https_google" == "PASS" ]] && [[ "$has_https_github" == "FAIL" ]]; then
                log::warning "SELECTIVE HTTPS ISSUES: GitHub-specific problems"
                log::info "- Google HTTPS works but GitHub fails"
                log::info "- This suggests domain-specific blocking or different TLS requirements"
                log::info "- GitHub may have stricter TLS/cipher requirements"
            elif [[ "$has_https_google" == "FAIL" ]] && [[ "$has_https_github" == "PASS" ]]; then
                log::warning "SELECTIVE HTTPS ISSUES: Google-specific problems"
                log::info "- GitHub HTTPS works but Google fails"
                log::info "- This is unusual and may indicate DNS or routing issues"
            else
                log::success "HTTPS connectivity appears to be working"
            fi
            
        elif [[ "$has_tcp" == "FAIL" ]]; then
            log::error "TCP connectivity issues detected"
            log::warning "- Basic ping works but TCP connections fail"
            log::warning "- This suggests firewall or routing issues"
        fi
    elif [[ "$has_basic_connectivity" == "FAIL" ]]; then
        log::error "No basic network connectivity"
        log::warning "- Cannot reach external networks"  
        log::warning "- Check network cable/WiFi and gateway configuration"
    fi
    
    # Check for localhost issues
    if [[ "${TEST_RESULTS["Localhost port 9200 (SearXNG)"]:-FAIL}" == "PASS" ]] && 
       [[ "${TEST_RESULTS["HTTP to localhost:9200"]:-FAIL}" == "FAIL" ]]; then
        log::warning ""
        log::warning "Local service issue detected:"
        log::warning "- Port 9200 is open but HTTP requests hang"
        log::warning "- This matches the SearXNG uWSGI HTTP-socket issue"
        log::info "- Try the nginx reverse proxy workaround"
    fi
    
    # Provide specific recommendations based on patterns
    log::info ""
    log::info "Recommended next steps:"
    if [[ "$has_tls12_google" == "PASS" ]] && [[ "$has_https_google" == "FAIL" ]]; then
        log::info "1. Configure applications to prefer TLS 1.2/1.3 explicitly"
        log::info "2. Try: export CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt"
        log::info "3. Test TCP offload fix: sudo ethtool -K \$(ip route | grep default | awk '{print \$5}') tso off gso off"
    fi
    
    if [[ "$has_https_github" == "FAIL" ]]; then
        log::info "For git/GitHub issues:"
        log::info "1. Use SSH instead: git remote set-url origin git@github.com:user/repo.git"
        log::info "2. Or try: git config --global http.version HTTP/1.1"
        log::info "3. Or try: git config --global http.sslVersion tlsv1.2"
    fi
    
    # Provide helpful guidance based on results
    log::subheader "Setup Readiness Assessment"
    
    if [[ "$CRITICAL_FAILURE" == "true" ]]; then
        log::error "❌ CRITICAL: Basic network connectivity is broken"
        log::info ""
        log::info "🔍 What this means:"
        log::info "   • Your system cannot reach the internet for basic operations"
        log::info "   • Setup scripts cannot download dependencies or updates"
        log::info "   • Package managers (apt, npm, etc.) won't work"
        log::info ""
        log::info "⚠️  Why this blocks setup:"
        log::info "   • Setup requires downloading Docker images, packages, and dependencies"
        log::info "   • Without basic connectivity, the setup process will fail"
        log::info ""
        log::info "🛠️  How to proceed:"
        log::info "   1. Check your network connection (WiFi/Ethernet)"
        log::info "   2. Verify your router/gateway is working"
        log::info "   3. Try: ping 8.8.8.8"
        log::info "   4. Contact your network administrator if needed"
        log::info ""
        log::warning "Setup cannot continue until basic connectivity is restored."
        return "${ERROR_NO_INTERNET}"
        
    elif [[ $failed_tests -gt $((total_tests / 2)) ]]; then
        log::warning "⚠️  MAJOR: Significant network issues detected ($failed_tests/$total_tests tests failed)"
        log::info ""
        log::info "🔍 What this means:"
        log::info "   • Most advanced network features are not working properly"
        log::info "   • Setup may fail or encounter frequent errors"
        log::info "   • Some services may not function correctly"
        log::info ""
        log::info "⚠️  Impact on setup:"
        log::info "   • High likelihood of download failures"
        log::info "   • Docker image pulls may timeout"
        log::info "   • SSL/TLS connections may fail"
        log::info ""
        log::info "🛠️  Recommended actions:"
        log::info "   1. Review the specific failures above"
        log::info "   2. Try the automatic optimizations if available"
        log::info "   3. Consider fixing major issues before continuing"
        log::info ""
        log::warning "Continue with setup anyway? (high risk of failures) [y/N]"
        if read -r -t 10 response 2>/dev/null; then
            if [[ ! "$response" =~ ^[Yy]$ ]]; then
                log::info "Setup cancelled. Fix network issues and try again."
                return "${ERROR_NO_INTERNET}"
            fi
        else
            log::info "No response (timed out or non-interactive), defaulting to cancel setup."
            return "${ERROR_NO_INTERNET}"
        fi
        log::warning "⚠️  Continuing despite major network issues..."
        
    elif [[ $failed_tests -gt 0 ]]; then
        # Calculate risk level based on specific failures
        local risk_level="LOW"
        local github_fails=false
        local tls_issues=false
        
        # Check for specific high-impact failures
        if [[ "${TEST_RESULTS["HTTPS to github.com"]:-PASS}" == "FAIL" ]]; then
            github_fails=true
        fi
        
        if [[ ${#TEST_RESULTS[@]} -gt 0 ]]; then
            for test_name in "${!TEST_RESULTS[@]}"; do
                if [[ "$test_name" =~ TLS|HTTPS ]] && [[ "${TEST_RESULTS[$test_name]}" == "FAIL" ]]; then
                    if [[ "$test_name" =~ "google.com" ]]; then
                        tls_issues=true
                        risk_level="MEDIUM"
                    fi
                fi
            done
        fi
        
        if [[ "$github_fails" == "true" ]] && [[ "$tls_issues" == "false" ]]; then
            log::info "ℹ️  MINOR: GitHub-specific connectivity issues ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • GitHub HTTPS connections are failing"
            log::info "   • Git operations over HTTPS won't work"
            log::info "   • Other web services appear to be working fine"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Git clones over HTTPS will fail"
            log::info "   • npm packages from GitHub may fail to download"
            log::info "   • Most other operations should work normally"
            log::info ""
            log::info "🛠️  Workarounds available:"
            log::info "   • Use SSH for git operations (recommended)"
            log::info "   • Configure git to use HTTP/1.1: git config --global http.version HTTP/1.1"
            log::info "   • Use specific TLS version: git config --global http.sslVersion tlsv1.2"
            log::info ""
            log::success "✅ Setup can continue - GitHub issues are manageable"
            
        elif [[ "$risk_level" == "MEDIUM" ]]; then
            log::warning "⚠️  MODERATE: TLS/HTTPS connectivity issues ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • Some HTTPS connections are unreliable"
            log::info "   • SSL/TLS handshakes may timeout or fail"
            log::info "   • Package downloads might be slower or fail occasionally"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Moderate risk of download failures"
            log::info "   • Some Docker images may fail to pull"
            log::info "   • Setup may take longer due to retries"
            log::info ""
            log::info "🛠️  Recommended actions:"
            log::info "   • Review the TCP offload suggestions above"
            log::info "   • Consider using HTTP mirrors where available"
            log::info "   • Monitor setup progress closely"
            log::info ""
            log::info "Continue with setup? (moderate risk) [Y/n]"
            if read -r -t 10 response 2>/dev/null; then
                if [[ "$response" =~ ^[Nn]$ ]]; then
                    log::info "Setup cancelled. Address network issues and try again."
                    return "${ERROR_NO_INTERNET}"
                fi
            else
                log::info "No response (timed out or non-interactive), defaulting to continue."
            fi
            log::info "📋 Continuing with moderate network risk..."
            
        else
            log::info "ℹ️  MINOR: Some network tests failed ($failed_tests/$total_tests tests failed)"
            log::info ""
            log::info "🔍 What this means:"
            log::info "   • Core networking functionality works"
            log::info "   • Some advanced features may have issues"
            log::info "   • Setup should complete successfully"
            log::info ""
            log::info "⚠️  Impact on setup:"
            log::info "   • Low risk of failures"
            log::info "   • Specific services may need manual configuration"
            log::info ""
            log::success "✅ Setup can continue with minimal risk"
        fi
        
    else
        log::success "🎉 Excellent! All network tests passed!"
        log::info ""
        log::info "✅ Your system has:"
        log::info "   • Full internet connectivity"
        log::info "   • Working DNS resolution"  
        log::info "   • Functional HTTP/HTTPS protocols"
        log::info "   • Proper TLS/SSL support"
        log::info ""
        log::success "🚀 Ready to proceed with setup!"
    fi
    
    return 0
}

# If run directly, execute diagnostics
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    network_diagnostics::run "$@"
fi