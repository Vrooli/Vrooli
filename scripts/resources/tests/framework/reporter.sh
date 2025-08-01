#!/bin/bash
# ====================================================================
# Test Results Reporter
# ====================================================================
#
# Generates comprehensive test reports in both human-readable and
# machine-readable formats. Provides insights and recommendations
# based on test results.
#
# Functions:
#   - generate_final_report()     - Generate complete test report
#   - generate_json_report()      - Generate JSON report for CI/CD
#   - print_resource_status()     - Show resource status table
#   - print_test_results()        - Show test results table
#   - generate_recommendations()  - Provide actionable insights
#
# ====================================================================

# Generate final comprehensive report
generate_final_report() {
    local duration="$1"
    
    case "$OUTPUT_FORMAT" in
        "json")
            generate_json_report "$duration"
            ;;
        "text"|*)
            generate_text_report "$duration"
            ;;
    esac
}

# Generate human-readable text report
generate_text_report() {
    local duration="$1"
    
    echo
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                         Vrooli Resource Integration Tests                     ║"
    echo "╠══════════════════════════════════════════════════════════════════════════════╣"
    echo "║ Test Run: $(date)                                        ║"
    echo "║ Duration: ${duration}s                                                        ║"
    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo
    
    # Summary Section
    echo "📊 SUMMARY"
    echo "─────────"
    printf "%-25s %d\n" "Total Resources Found:" "${#ALL_RESOURCES[@]}"
    printf "%-25s %d\n" "Enabled Resources:" "${#ENABLED_RESOURCES[@]}"
    printf "%-25s %d\n" "Healthy Resources:" "${#HEALTHY_RESOURCES[@]}"
    printf "%-25s %d\n" "Tests Executed:" "$TOTAL_TESTS"
    echo
    printf "%-25s %s%d%s\n" "✅ Passed Tests:" "$GREEN" "$PASSED_TESTS" "$NC"
    printf "%-25s %s%d%s\n" "❌ Failed Tests:" "$RED" "$FAILED_TESTS" "$NC"
    printf "%-25s %s%d%s\n" "⚠️  Skipped Tests:" "$YELLOW" "$SKIPPED_TESTS" "$NC"
    echo
    
    # Resource Status Section
    echo "📋 RESOURCE STATUS"
    echo "─────────────────"
    print_resource_status_table
    echo
    
    # Test Results Section
    echo "🧪 TEST RESULTS"
    echo "──────────────"
    print_test_results_summary
    echo
    
    # Failed Tests Detail
    if [[ ${#FAILED_TEST_NAMES[@]} -gt 0 ]]; then
        echo "❌ FAILED TESTS DETAIL"
        echo "─────────────────────"
        for failed_test in "${FAILED_TEST_NAMES[@]}"; do
            echo "  • $failed_test"
        done
        echo
    fi
    
    # Skipped Tests Detail
    if [[ ${#SKIPPED_TEST_NAMES[@]} -gt 0 ]]; then
        echo "⚠️  SKIPPED TESTS DETAIL"
        echo "──────────────────────"
        for skipped_test in "${SKIPPED_TEST_NAMES[@]}"; do
            echo "  • $skipped_test"
        done
        echo
    fi
    
    # Missing Test Coverage
    if [[ ${#MISSING_TEST_RESOURCES[@]} -gt 0 ]]; then
        echo "📝 RESOURCES WITHOUT TEST COVERAGE"
        echo "─────────────────────────────────"
        echo "The following ${#MISSING_TEST_RESOURCES[@]} healthy resources lack test files:"
        for resource in "${MISSING_TEST_RESOURCES[@]}"; do
            echo "  • $resource"
        done
        echo
        echo "💡 To add test coverage, create files at:"
        echo "   \$SCRIPT_DIR/single/<category>/<resource>.test.sh"
        echo "   Example categories: ai/, storage/, automation/, agents/, search/"
        echo
    fi
    
    # Recommendations
    echo "📝 RECOMMENDATIONS"
    echo "─────────────────"
    generate_recommendations
    echo
    
    # Debug Information (if enabled)
    if [[ "${TEST_DEBUG:-false}" == "true" ]] || [[ "${DEBUG:-false}" == "true" ]]; then
        echo "🔍 DEBUG INFORMATION"
        echo "──────────────────"
        
        # Show HTTP log summary if available
        if [[ "$(type -t show_http_summary)" == "function" ]]; then
            show_http_summary
            echo
        fi
        
        # Show test log locations
        echo "📁 Test Logs:"
        ls -la /tmp/vrooli_test_*.log 2>/dev/null | tail -5 || echo "   No test logs found"
        
        # Show HTTP log locations
        echo -e "\n📁 HTTP Logs:"
        ls -la /tmp/vrooli_http_*.log 2>/dev/null | tail -5 || echo "   No HTTP logs found"
        echo
    fi
    
    # Final Status
    echo "🎯 FINAL STATUS"
    echo "──────────────"
    if [[ $FAILED_TESTS -eq 0 ]]; then
        echo -e "${GREEN}✅ All tests passed successfully!${NC}"
        if [[ $SKIPPED_TESTS -gt 0 ]]; then
            echo -e "${YELLOW}⚠️  Note: $SKIPPED_TESTS tests were skipped due to missing dependencies${NC}"
        fi
    else
        echo -e "${RED}❌ $FAILED_TESTS test(s) failed${NC}"
        echo "Review the failed tests and resource status above for troubleshooting guidance."
    fi
    echo
}

# Generate machine-readable JSON report
generate_json_report() {
    local duration="$1"
    local timestamp=$(date -Iseconds)
    
    cat << EOF
{
  "test_run": {
    "timestamp": "$timestamp",
    "duration_seconds": $duration,
    "format_version": "1.0"
  },
  "configuration": {
    "timeout": $TEST_TIMEOUT,
    "verbose": $VERBOSE,
    "fail_fast": $FAIL_FAST,
    "cleanup": $CLEANUP,
    "specific_resource": "${SPECIFIC_RESOURCE:-null}"
  },
  "resources": {
    "discovered": [$(printf '"%s",' "${ALL_RESOURCES[@]}" | sed 's/,$//')]],
    "enabled": [$(printf '"%s",' "${ENABLED_RESOURCES[@]}" | sed 's/,$//')]],
    "healthy": [$(printf '"%s",' "${HEALTHY_RESOURCES[@]}" | sed 's/,$//')]],
    "disabled": [$(printf '"%s",' "${DISABLED_RESOURCES[@]}" | sed 's/,$//')]],
    "unhealthy": [$(printf '"%s",' "${UNHEALTHY_RESOURCES[@]}" | sed 's/,$//')]
  },
  "test_summary": {
    "total": $TOTAL_TESTS,
    "passed": $PASSED_TESTS,
    "failed": $FAILED_TESTS,
    "skipped": $SKIPPED_TESTS,
    "success_rate": $(awk "BEGIN {printf \"%.2f\", $PASSED_TESTS / ($TOTAL_TESTS == 0 ? 1 : $TOTAL_TESTS) * 100}")
  },
  "failed_tests": [$(printf '"%s",' "${FAILED_TEST_NAMES[@]}" | sed 's/,$//')]],
  "skipped_tests": [$(printf '"%s",' "${SKIPPED_TEST_NAMES[@]}" | sed 's/,$//')]],
  "recommendations": $(generate_recommendations_json)
}
EOF
}

# Print resource status table
print_resource_status_table() {
    printf "%-15s %-12s %-20s\n" "Resource" "Status" "Notes"
    printf "%-15s %-12s %-20s\n" "--------" "------" "-----"
    
    # Healthy resources
    for resource in "${HEALTHY_RESOURCES[@]}"; do
        printf "%-15s %s%-12s%s %-20s\n" "$resource" "$GREEN" "✅ Healthy" "$NC" "Ready for testing"
    done
    
    # Unhealthy resources
    for resource in "${UNHEALTHY_RESOURCES[@]}"; do
        printf "%-15s %s%-12s%s %-20s\n" "$resource" "$RED" "❌ Unhealthy" "$NC" "Check service status"
    done
    
    # Disabled resources
    for resource in "${DISABLED_RESOURCES[@]}"; do
        printf "%-15s %s%-12s%s %-20s\n" "$resource" "$YELLOW" "⚠️  Disabled" "$NC" "Not enabled in config"
    done
}

# Print test results summary
print_test_results_summary() {
    local pass_rate=0
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        pass_rate=$(awk "BEGIN {printf \"%.1f\", $PASSED_TESTS / $TOTAL_TESTS * 100}")
    fi
    
    echo "Overall Success Rate: $pass_rate%"
    echo
    
    if [[ $TOTAL_TESTS -eq 0 ]]; then
        echo "No tests were executed. Check resource availability and configuration."
        return
    fi
    
    # Test categories breakdown
    if [[ $PASSED_TESTS -gt 0 ]]; then
        echo "✅ Passed Tests ($PASSED_TESTS):"
        echo "  All passed tests completed successfully within timeout limits."
        echo
    fi
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        echo "❌ Failed Tests ($FAILED_TESTS):"
        echo "  These tests encountered errors or timeouts. Check individual test output."
        echo
    fi
    
    if [[ $SKIPPED_TESTS -gt 0 ]]; then
        echo "⚠️  Skipped Tests ($SKIPPED_TESTS):"
        echo "  These tests were skipped due to missing dependencies or configuration."
        echo
    fi
}

# Generate actionable recommendations
generate_recommendations() {
    local recommendations=()
    
    # Check for unhealthy resources
    if [[ ${#UNHEALTHY_RESOURCES[@]} -gt 0 ]]; then
        recommendations+=("🔧 Fix unhealthy resources: ${UNHEALTHY_RESOURCES[*]}")
        recommendations+=("   Run: ./scripts/resources/index.sh --action status --resources \"${UNHEALTHY_RESOURCES[*]}\"")
    fi
    
    # Check for disabled resources
    if [[ ${#DISABLED_RESOURCES[@]} -gt 0 ]]; then
        recommendations+=("⚙️  Enable additional resources in ~/.vrooli/resources.local.json")
        recommendations+=("   Consider enabling: ${DISABLED_RESOURCES[*]}")
    fi
    
    # Check for failed tests
    if [[ $FAILED_TESTS -gt 0 ]]; then
        recommendations+=("🐛 Investigate failed tests:")
        recommendations+=("   • Run with --verbose for detailed output")
        recommendations+=("   • Check individual test: bash single/<category>/<resource>.test.sh")
        recommendations+=("   • Verify resource connectivity: curl http://localhost:<port>/health")
        recommendations+=("   • Check resource logs: ./scripts/resources/<category>/<resource>/manage.sh --action logs")
        recommendations+=("   • Review test requirements in test file headers")
        recommendations+=("   • Validate test file compliance: ./framework/helpers/test-validator.sh --path <test>")
    fi
    
    # Check for skipped tests
    if [[ $SKIPPED_TESTS -gt 0 ]]; then
        recommendations+=("📦 Reduce skipped tests by addressing missing resources:")
        recommendations+=("   • Check resource health: ./scripts/resources/index.sh --action discover")
        recommendations+=("   • Install missing resources: ./scripts/resources/index.sh --action install --resources <name>")
        recommendations+=("   • Enable resources in config: ~/.vrooli/resources.local.json")
        recommendations+=("   • Review skipped test requirements in test file headers")
    fi
    
    # Performance recommendations
    if [[ $TOTAL_TESTS -gt 0 ]]; then
        local avg_duration=$((duration / TOTAL_TESTS))
        if [[ $avg_duration -gt 30 ]]; then
            recommendations+=("⚡ Consider optimizing test performance (avg: ${avg_duration}s per test)")
            recommendations+=("   Use --timeout to adjust timeouts for slow resources")
        fi
    fi
    
    # Success recommendations
    if [[ $FAILED_TESTS -eq 0 && $SKIPPED_TESTS -eq 0 ]]; then
        recommendations+=("🎉 Excellent! All tests passed. Your resource ecosystem is healthy.")
        recommendations+=("💡 Consider adding more integration tests for edge cases")
    fi
    
    # Output recommendations
    if [[ ${#recommendations[@]} -eq 0 ]]; then
        echo "No specific recommendations at this time."
    else
        for rec in "${recommendations[@]}"; do
            echo "$rec"
        done
    fi
}

# Generate recommendations in JSON format
generate_recommendations_json() {
    local recommendations=()
    
    # Similar logic as above but format for JSON
    if [[ ${#UNHEALTHY_RESOURCES[@]} -gt 0 ]]; then
        recommendations+=("\"Fix unhealthy resources: ${UNHEALTHY_RESOURCES[*]}\"")
    fi
    
    if [[ ${#DISABLED_RESOURCES[@]} -gt 0 ]]; then
        recommendations+=("\"Enable additional resources in configuration\"")
    fi
    
    if [[ $FAILED_TESTS -gt 0 ]]; then
        recommendations+=("\"Investigate failed tests with verbose logging\"")
    fi
    
    if [[ $SKIPPED_TESTS -gt 0 ]]; then
        recommendations+=("\"Install missing dependencies to reduce skipped tests\"")
    fi
    
    if [[ ${#recommendations[@]} -eq 0 ]]; then
        echo "[]"
    else
        echo "[$(IFS=','; echo "${recommendations[*]}")]"
    fi
}

# Save report to file
save_report_to_file() {
    local filename="$1"
    local format="${2:-text}"
    
    case "$format" in
        "json")
            generate_json_report > "$filename"
            ;;
        "text"|*)
            generate_text_report > "$filename"
            ;;
    esac
    
    log_info "Report saved to: $filename"
}

# Print resource discovery summary
print_discovery_summary() {
    echo "Resource Discovery Summary:"
    echo "  • Total discovered: ${#ALL_RESOURCES[@]}"
    echo "  • Enabled in config: ${#ENABLED_RESOURCES[@]}"
    echo "  • Healthy and ready: ${#HEALTHY_RESOURCES[@]}"
    echo "  • Will be tested: $(get_testable_resource_count)"
    echo
}

# Get count of resources that will actually be tested
get_testable_resource_count() {
    local count=0
    for resource in "${HEALTHY_RESOURCES[@]}"; do
        if [[ -n "$SPECIFIC_RESOURCE" && "$resource" != "$SPECIFIC_RESOURCE" ]]; then
            continue
        fi
        
        local test_file
        test_file=$(find_resource_test_file "$resource" 2>/dev/null)
        if [[ -n "$test_file" ]]; then
            count=$((count + 1))
        fi
    done
    echo "$count"
}

# Print progress indicator
print_test_progress() {
    local current="$1"
    local total="$2"
    local test_name="$3"
    
    local percentage=$((current * 100 / total))
    local bar_length=20
    local filled_length=$((percentage * bar_length / 100))
    
    printf "\r[%s%s] %d%% (%d/%d) %s" \
        "$(printf "%*s" "$filled_length" "" | tr ' ' '=')" \
        "$(printf "%*s" $((bar_length - filled_length)) "")" \
        "$percentage" "$current" "$total" "$test_name"
    
    if [[ $current -eq $total ]]; then
        echo  # New line when complete
    fi
}