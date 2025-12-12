#!/usr/bin/env bash
# Test Infrastructure Validator
# Validates scenario test infrastructure completeness

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

scenario::test::absolute_path() {
    local relative_path="$1"
    local base_dir="${APP_ROOT:-$(pwd -P)}"
    base_dir="${base_dir%/}"
    printf '%s/%s' "$base_dir" "$relative_path"
}

scenario::test::has_ui_component() {
    local scenario_path="$1"
    if [[ -d "$scenario_path/ui" ]] && [[ -f "$scenario_path/ui/package.json" ]]; then
        echo "true"
    else
        echo "false"
    fi
}

# Validate test infrastructure for a scenario
# Usage: scenario::test::validate_infrastructure <scenario_name> <scenario_path>
scenario::test::validate_infrastructure() {
    local scenario_name="$1"
    local scenario_path="$2"
    
    local validation_result="{}"
    local has_ui
    has_ui=$(scenario::test::has_ui_component "$scenario_path")
    
    # Check test lifecycle event
    local test_lifecycle_status
    test_lifecycle_status=$(scenario::test::check_test_lifecycle "$scenario_path")
    
    # Check required configs
    local config_status
    config_status=$(scenario::test::check_required_configs "$scenario_path" "$has_ui")
    
    # Check phased testing structure  
    local phased_structure_status
    phased_structure_status=$(scenario::test::check_phased_structure "$scenario_path")
    
    # Check unit tests
    local unit_tests_status
    unit_tests_status=$(scenario::test::check_unit_tests "$scenario_path")
    
    # Check CLI tests
    local cli_tests_status
    cli_tests_status=$(scenario::test::check_cli_tests "$scenario_path")
    
    # Check UI tests
    local ui_tests_status
    ui_tests_status=$(scenario::test::check_ui_tests "$scenario_path" "$has_ui")
    
    # Coverage artifacts
    local coverage_status
    coverage_status=$(scenario::test::check_coverage_status "$scenario_name" "$scenario_path")
    
    # Calculate overall completeness
    local overall_status
    overall_status=$(scenario::test::calculate_overall_status "$test_lifecycle_status" "$config_status" "$phased_structure_status" "$unit_tests_status" "$cli_tests_status" "$ui_tests_status")
    
    # Build comprehensive validation result JSON
    validation_result=$(jq -n \
        --argjson test_lifecycle "$test_lifecycle_status" \
        --argjson configs "$config_status" \
        --argjson phased_structure "$phased_structure_status" \
        --argjson unit_tests "$unit_tests_status" \
        --argjson cli_tests "$cli_tests_status" \
        --argjson ui_tests "$ui_tests_status" \
        --argjson coverage "$coverage_status" \
        --argjson overall "$overall_status" \
        '{
            test_lifecycle: $test_lifecycle,
            configs: $configs,
            phased_structure: $phased_structure,
            unit_tests: $unit_tests,
            cli_tests: $cli_tests,
            ui_tests: $ui_tests,
            coverage: $coverage,
            overall: $overall
        }')
    
    echo "$validation_result"
}

# Check if test lifecycle event is defined
scenario::test::check_test_lifecycle() {
    local scenario_path="$1"
    local service_json="$scenario_path/.vrooli/service.json"
    
    # First check for legacy test format
    if [[ -f "$scenario_path/scenario-test.yaml" ]]; then
        # This is a legacy format scenario
        echo '{"status": "legacy", "message": "⚠️  Legacy test format (scenario-test.yaml)", "recommendation": "Migrate to test-genie (Go-native phased suite)", "format": "legacy"}'
        return
    fi
    
    if [[ ! -f "$service_json" ]]; then
        echo '{"status": "missing", "message": "service.json not found"}'
        return
    fi
    
    # Check if test lifecycle event exists
    local test_event
    test_event=$(jq -r '.lifecycle.test // null' "$service_json" 2>/dev/null || echo "null")
    
    if [[ "$test_event" == "null" ]] || [[ -z "$test_event" ]]; then
        echo '{"status": "missing", "message": "No test lifecycle event defined", "recommendation": "Add test lifecycle event to .vrooli/service.json"}'
    else
        # Check if it's a valid command (could be a simple command string or steps array)
        local test_script=""
        
        # First check if it's a simple string (legacy format)
        if [[ "$(echo "$test_event" | jq -r 'type')" == "string" ]]; then
            test_script="$test_event"
        else
            # Check for steps array (new v2 format)
            local has_steps=$(echo "$test_event" | jq 'has("steps")' 2>/dev/null || echo "false")
            if [[ "$has_steps" == "true" ]]; then
                # Get the first step's run command as representative
                test_script=$(echo "$test_event" | jq -r '.steps[0].run // null' 2>/dev/null || echo "null")
            else
                # Legacy format with script/command field
                test_script=$(echo "$test_event" | jq -r '.script // .command // null' 2>/dev/null || echo "null")
            fi
        fi
        
        if [[ "$test_script" != "null" && -n "$test_script" ]]; then
            # Scenarios should invoke test-genie directly (scenario-local test/run-tests.sh is deprecated).
            if [[ "$test_script" != *"test-genie execute"* ]]; then
                echo '{"status": "invalid", "message": "Test lifecycle does not invoke test-genie", "script": "'"$test_script"'", "recommendation": "Point lifecycle.test to `test-genie execute <scenario> --preset comprehensive` (or an appropriate preset)"}'
            else
                echo '{"status": "present", "message": "Test lifecycle event defined", "script": "'"$test_script"'"}'
            fi
        else
            echo '{"status": "invalid", "message": "Test lifecycle event malformed", "recommendation": "Fix test lifecycle event structure in .vrooli/service.json"}'
        fi
    fi
}

scenario::test::check_required_configs() {
    local scenario_path="$1"
    local has_ui="$2"

    local required_total=2
    local present_count=0
    local missing_items=()
    local details=()

    local testing_path="$scenario_path/.vrooli/testing.json"
    if [[ -f "$testing_path" ]]; then
        present_count=$((present_count + 1))
        details+=(".vrooli/testing.json:present")
    else
        missing_items+=(".vrooli/testing.json")
        details+=(".vrooli/testing.json:missing")
    fi

    local endpoints_path="$scenario_path/.vrooli/endpoints.json"
    if [[ -f "$endpoints_path" ]]; then
        present_count=$((present_count + 1))
        details+=(".vrooli/endpoints.json:present")
    else
        missing_items+=(".vrooli/endpoints.json")
        details+=(".vrooli/endpoints.json:missing")
    fi

    if [[ "$has_ui" == "true" ]]; then
        required_total=$((required_total + 1))
        local lighthouse_path="$scenario_path/.vrooli/lighthouse.json"
        if [[ -f "$lighthouse_path" ]]; then
            present_count=$((present_count + 1))
            details+=(".vrooli/lighthouse.json:present")
        else
            missing_items+=(".vrooli/lighthouse.json")
            details+=(".vrooli/lighthouse.json:missing")
        fi
    fi

    local status="missing"
    local message="No required testing configs found"
    if [[ $present_count -eq 0 ]]; then
        status="missing"
        message="Required testing configs missing"
    elif [[ $present_count -lt $required_total ]]; then
        status="partial"
        message="Missing $((${required_total} - present_count)) config(s): ${missing_items[*]}"
    else
        status="complete"
        message="All required testing configs present"
    fi

    local missing_json
    if [[ ${#missing_items[@]} -gt 0 ]]; then
        missing_json=$(printf '%s\n' "${missing_items[@]}" | jq -R . | jq -s .)
    else
        missing_json='[]'
    fi

    local details_json=$(printf '%s\n' "${details[@]}" | jq -R . | jq -s .)

    echo "{\"status\": \"$status\", \"message\": \"$message\", \"missing_items\": $missing_json, \"details\": $details_json, \"present\": $present_count, \"required\": $required_total}"
}

# Check test directory structure (scenario playbooks + unit test runners)
scenario::test::check_phased_structure() {
    local scenario_path="$1"
    local test_dir="$scenario_path/test"

    local status="missing"
    local message=""
    local recommendations=()
    local found_components=()

    # Check for test directory
    if [[ ! -d "$test_dir" ]]; then
        echo '{"status": "missing", "message": "No test/ directory found", "recommendation": "Create test/ directory for scenario test assets", "components": []}'
        return
    fi

    found_components+=("test_directory")

    # Scenario-local phased runners (test/run-tests.sh + test/phases/*) are deprecated in favor
    # of test-genie's Go-native orchestrator. BAS automation assets live under bas/ (when used).
    local bas_dir="$scenario_path/bas"
    if [[ -d "$bas_dir" ]]; then
        found_components+=("bas_directory")
        local registry_path="$bas_dir/registry.json"
        if [[ -f "$registry_path" ]]; then
            found_components+=("bas_registry")
            status="complete"
            message="BAS registry present"
        else
            status="partial"
            message="bas/ directory present, missing registry.json"
            recommendations+=("Run 'test-genie registry build' from the scenario directory to generate bas/registry.json")
        fi
    else
        status="complete"
        message="Test directory present (no bas)"
    fi

    # Check for unit test runners
    if [[ -d "$test_dir/unit" ]]; then
        found_components+=("unit_runners")
    else
        recommendations+=("Create test/unit/ directory with language-specific runners")
    fi

    # If no components found except test directory
    if [[ ${#found_components[@]} -eq 1 ]]; then
        status="empty"
        message="Test directory exists but has no playbooks/unit runners"
    fi

    local recommendations_json=$(printf '%s\n' "${recommendations[@]}" | jq -R . | jq -s .)
    local components_json=$(printf '%s\n' "${found_components[@]}" | jq -R . | jq -s .)

    echo "{\"status\": \"$status\", \"message\": \"$message\", \"components\": $components_json, \"recommendations\": $recommendations_json}"
}

# Check unit tests
scenario::test::check_unit_tests() {
    local scenario_path="$1"
    local found_types=()
    local status="none"
    local message="No unit tests found"
    
    # Check for Go tests
    if [[ -d "$scenario_path/api" ]] && find "$scenario_path/api" -name "*_test.go" -type f | head -1 | grep -q .; then
        found_types+=("go")
    fi
    
    # Check for Node.js tests
    if [[ -f "$scenario_path/ui/package.json" ]]; then
        if grep -q '"test":' "$scenario_path/ui/package.json" 2>/dev/null; then
            local test_script
            test_script=$(jq -r '.scripts.test // ""' "$scenario_path/ui/package.json" 2>/dev/null)
            if [[ -n "$test_script" && "$test_script" != "null" ]]; then
                local ui_test_dir="$scenario_path/ui"
                if [[ -d "$scenario_path/ui/src" ]]; then
                    ui_test_dir="$scenario_path/ui/src"
                fi
                if find "$ui_test_dir" \( -path '*/node_modules/*' -o -path '*/dist/*' \) -prune -o \( -name '*.test.tsx' -o -name '*.test.ts' -o -name '*.test.jsx' -o -name '*.test.js' -o -name '*.test.mjs' \) -type f -print -quit | grep -q .; then
                    found_types+=("node")
                fi
            fi
        fi
    fi
    
    # Check for Python tests
    if [[ -f "$scenario_path/requirements.txt" ]] || [[ -f "$scenario_path/pyproject.toml" ]]; then
        if find "$scenario_path" -path "*/test*" -name "test_*.py" -o -name "*_test.py" | head -1 | grep -q .; then
            found_types+=("python")
        fi
    fi
    
    # Determine status and message
    if [[ ${#found_types[@]} -gt 0 ]]; then
        if [[ ${#found_types[@]} -eq 1 ]]; then
            status="partial"
            message="Unit tests found: ${found_types[*]}"
        else
            status="comprehensive"
            message="Multiple unit test types: ${found_types[*]}"
        fi
    fi
    
    local types_json=$(printf '%s\n' "${found_types[@]}" | jq -R . | jq -s .)
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"types\": $types_json}"
}

# Check CLI tests
scenario::test::check_cli_tests() {
    local scenario_path="$1"
    local cli_dir="$scenario_path/cli"
    local found_tests=()
    local status="none"
    local message="No CLI tests found"
    
    if [[ -d "$cli_dir" ]]; then
        # Look for BATS files
        while IFS= read -r -d '' bats_file; do
            local filename=$(basename "$bats_file")
            found_tests+=("$filename")
        done < <(find "$cli_dir" -name "*.bats" -type f -print0 2>/dev/null)
        
        if [[ ${#found_tests[@]} -gt 0 ]]; then
            status="present"
            message="BATS tests found: ${#found_tests[@]} file(s)"
        fi
    fi
    
    local tests_json=$(printf '%s\n' "${found_tests[@]}" | jq -R . | jq -s .)
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"tests\": $tests_json}"
}

# Check UI tests
scenario::test::check_ui_tests() {
    local scenario_path="$1"
    local has_ui_override="${2:-}"
    local ui_test_dir="$scenario_path/test/ui"
    local workflow_dir="$scenario_path/bas/cases"
    local found_workflows=()
    local status="none"
    local message="No UI automation tests found"
    
    # Check if scenario has UI component first
    local has_ui=false
    if [[ -n "$has_ui_override" ]]; then
        has_ui="$has_ui_override"
    elif [[ -d "$scenario_path/ui" ]] && [[ -f "$scenario_path/ui/package.json" ]]; then
        has_ui=true
    fi
    
    if [[ ! "$has_ui" == "true" ]]; then
        echo '{"status": "not_applicable", "message": "No UI component detected", "workflows": []}'
        return
    fi
    
    local workflow_count=0
    if [[ -d "$workflow_dir" ]]; then
        while IFS= read -r -d '' workflow_file; do
            local filename=$(basename "$workflow_file")
            found_workflows+=("$filename")
        done < <(find "$workflow_dir" -name "*.json" -type f -print0 2>/dev/null)
        workflow_count=${#found_workflows[@]}
    fi

    if [[ $workflow_count -gt 0 ]]; then
        status="present"
        message="BAS workflows available: $workflow_count"
    else
        status="missing"
        message="Export workflows to bas/ for UI automation"
    fi
    
    local workflows_json=$(printf '%s\n' "${found_workflows[@]}" | jq -R . | jq -s .)
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"workflows\": $workflows_json, \"workflow_count\": $workflow_count}"
}

scenario::test::check_coverage_status() {
    local scenario_name="$1"
    local scenario_path="$2"

    local aggregate_path="$scenario_path/coverage/${scenario_name}/aggregate.json"

    if [[ ! -f "$aggregate_path" ]]; then
        echo '{"status": "missing", "message": "Coverage artifacts not generated"}'
        return
    fi

    local generated_at
    generated_at=$(jq -r '.generated_at // ""' "$aggregate_path" 2>/dev/null)
    if [[ -z "$generated_at" || "$generated_at" == "null" ]]; then
        generated_at="$(date -u -r "$aggregate_path" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u +"%Y-%m-%dT%H:%M:%SZ")"
    fi

    local languages_json
    languages_json=$(jq -c '.languages // []' "$aggregate_path" 2>/dev/null || echo '[]')

    local summaries=()
    if [[ "$languages_json" != "[]" ]]; then
        while IFS= read -r entry; do
            [[ -z "$entry" ]] && continue
            local lang
            lang=$(echo "$entry" | jq -r '.language // "unknown"')
            local percent
            percent=$(echo "$entry" | jq -r '.metrics.statements // .metrics.lines // .metrics.coverage // empty')
            if [[ -n "$percent" && "$percent" != "null" ]]; then
                if echo "$percent" | grep -Eq '^[0-9]+(\.[0-9]+)?$'; then
                    percent=$(printf '%.1f%%' "$percent" 2>/dev/null || echo "$percent%")
                fi
            else
                percent="n/a"
            fi
            summaries+=("$lang $percent")
        done < <(echo "$languages_json" | jq -c '.[]')
    fi

    local summary_message
    if [[ ${#summaries[@]} -gt 0 ]]; then
        summary_message=$(printf '%s' "${summaries[0]}")
        for ((i=1; i<${#summaries[@]}; i++)); do
            summary_message+=" | ${summaries[$i]}"
        done
    else
        summary_message="Coverage generated"
    fi

    local rel_path="${aggregate_path#$scenario_path/}"
    echo "{\"status\": \"present\", \"message\": \"$summary_message\", \"generated_at\": \"$generated_at\", \"path\": \"$rel_path\", \"languages\": $languages_json}"
}

# Calculate overall test infrastructure status
scenario::test::calculate_overall_status() {
    local test_lifecycle="$1"
    local configs="$2"
    local phased_structure="$3"
    local unit_tests="$4"
    local cli_tests="$5" 
    local ui_tests="$6"

    local phased_docs_path
    phased_docs_path=$(scenario::test::absolute_path "scenarios/test-genie/docs/guides/phased-testing.md")
    
    local score=0
    local max_score=6
    local status="incomplete"
    local message=""
    local recommendations=()

    # Score each component
    local lifecycle_status=$(echo "$test_lifecycle" | jq -r '.status')
    if [[ "$lifecycle_status" == "present" ]]; then
        ((score++))
    elif [[ "$lifecycle_status" == "legacy" ]]; then
        # Give partial credit for legacy format but recommend migration
        score=$((score + 0))  # No credit for legacy format
        recommendations+=("⚠️  Migrate from legacy scenario-test.yaml to test-genie phased suite")
        recommendations+=("See ${phased_docs_path} for migration guide")
    else
        recommendations+=("Define test lifecycle event in .vrooli/service.json")
    fi
    
    local config_status=$(echo "$configs" | jq -r '.status')
    case "$config_status" in
        complete)
            ((score++))
            ;;
        partial)
            score=$((score + 1))
            local missing_configs
            missing_configs=$(echo "$configs" | jq -r '.missing_items[]?' 2>/dev/null)
            if [[ -n "$missing_configs" ]]; then
                while IFS= read -r item; do
                    [[ -z "$item" ]] && continue
                    recommendations+=("Add $item to align with test-genie configuration")
                done <<< "$missing_configs"
            fi
            ;;
        *)
            local missing_configs
            missing_configs=$(echo "$configs" | jq -r '.missing_items[]?' 2>/dev/null)
            if [[ -n "$missing_configs" ]]; then
                while IFS= read -r item; do
                    [[ -z "$item" ]] && continue
                    recommendations+=("Add $item to align with test-genie configuration")
                done <<< "$missing_configs"
            else
                recommendations+=("Define required configs in .vrooli/ (testing.json, endpoints.json)")
            fi
            ;;
    esac

    local structure_status=$(echo "$phased_structure" | jq -r '.status')
    case "$structure_status" in
        complete) ((score++)) ;;
        partial) score=$((score + 1)) ;; # Partial credit
        *) recommendations+=("Add playbooks and/or unit runners under test/") ;;
    esac
    
    local unit_status=$(echo "$unit_tests" | jq -r '.status')
    if [[ "$unit_status" == "comprehensive" ]]; then
        ((score++))
    elif [[ "$unit_status" == "partial" ]]; then
        score=$((score + 1)) # Partial credit
    else
        recommendations+=("Add unit tests for your codebase (Go, Node.js, Python)")
    fi
    
    local cli_status=$(echo "$cli_tests" | jq -r '.status')
    if [[ "$cli_status" == "present" ]]; then
        ((score++))
    elif [[ -d "$(echo "$phased_structure" | jq -r '.scenario_path')/cli" ]] 2>/dev/null; then
        recommendations+=("Add BATS tests for CLI functionality")
    fi
    
    local ui_status=$(echo "$ui_tests" | jq -r '.status')
    if [[ "$ui_status" == "present" ]]; then
        ((score++))
    elif [[ "$ui_status" == "missing" ]]; then
        recommendations+=("Add UI automation tests using browser-automation-studio")
    fi
    
    # Determine overall status and message
    local percentage=$((score * 100 / max_score))
    if [[ $percentage -ge 80 ]]; then
        status="complete"
        message="Comprehensive test infrastructure ($score/$max_score components)"
    elif [[ $percentage -ge 50 ]]; then
        status="good"
        message="Good test coverage ($score/$max_score components)" 
    elif [[ $percentage -ge 25 ]]; then
        status="partial"
        message="Basic test infrastructure ($score/$max_score components)"
    else
        status="minimal"
        message="Minimal test infrastructure ($score/$max_score components)"
    fi
    
    local recommendations_json
    if [[ ${#recommendations[@]} -gt 0 ]]; then
        recommendations_json=$(printf '%s\n' "${recommendations[@]}" | jq -R . | jq -s .)
    else
        recommendations_json='[]'
    fi
    
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"score\": $score, \"max_score\": $max_score, \"percentage\": $percentage, \"recommendations\": $recommendations_json}"
}

# Display test infrastructure validation in text format
scenario::test::display_validation() {
    local scenario_name="$1"
    local validation_data="$2"

    local phased_docs_path
    phased_docs_path=$(scenario::test::absolute_path "scenarios/test-genie/docs/guides/phased-testing.md")

    echo ""
    echo "🧪 Test Infrastructure:"
    
    # Overall status
    local overall_status=$(echo "$validation_data" | jq -r '.overall.status')
    local overall_message=$(echo "$validation_data" | jq -r '.overall.message')
    local overall_percentage=$(echo "$validation_data" | jq -r '.overall.percentage')
    
    case "$overall_status" in
        complete)
            echo "├── Overall Status: ✅ $overall_message"
            ;;
        good)
            echo "├── Overall Status: 🟢 $overall_message"
            ;;
        partial)
            echo "├── Overall Status: 🟡 $overall_message"
            ;;
        minimal)
            echo "├── Overall Status: 🟠 $overall_message"
            ;;
        *)
            echo "├── Overall Status: ❌ $overall_message"
            ;;
    esac
    
    # Configs
    local config_status=$(echo "$validation_data" | jq -r '.configs.status')
    local config_message=$(echo "$validation_data" | jq -r '.configs.message')
    case "$config_status" in
        complete)
            echo "├── Configs: ✅ $config_message"
            ;;
        partial)
            echo "├── Configs: 🟡 $config_message"
            ;;
        *)
            echo "├── Configs: ⚠️  $config_message"
            ;;
    esac

    # Test lifecycle
    local lifecycle_status=$(echo "$validation_data" | jq -r '.test_lifecycle.status')
    local lifecycle_message=$(echo "$validation_data" | jq -r '.test_lifecycle.message')
    
    case "$lifecycle_status" in
        present)
            echo "├── Test Lifecycle: ✅ $lifecycle_message"
            ;;
        legacy)
            echo "├── Test Lifecycle: ⚠️  $lifecycle_message"
            ;;
        *)
            echo "├── Test Lifecycle: ❌ $lifecycle_message"
            ;;
    esac
    
    # Phased structure
    local structure_status=$(echo "$validation_data" | jq -r '.phased_structure.status')
    local structure_message=$(echo "$validation_data" | jq -r '.phased_structure.message')
    
    case "$structure_status" in
        complete)
            echo "├── Phased Testing: ✅ $structure_message"
            ;;
        partial)
            echo "├── Phased Testing: 🟡 $structure_message"
            ;;
        empty)
            echo "├── Phased Testing: 🟠 $structure_message"
            ;;
        *)
            echo "├── Phased Testing: ❌ $structure_message"
            ;;
    esac
    
    # Unit tests
    local unit_status=$(echo "$validation_data" | jq -r '.unit_tests.status')
    local unit_message=$(echo "$validation_data" | jq -r '.unit_tests.message')
    
    case "$unit_status" in
        comprehensive)
            echo "├── Unit Tests: ✅ $unit_message"
            ;;
        partial)
            echo "├── Unit Tests: 🟡 $unit_message"
            ;;
        *)
            echo "├── Unit Tests: ⚠️  $unit_message"
            ;;
    esac
    
    # CLI tests
    local cli_status=$(echo "$validation_data" | jq -r '.cli_tests.status')
    local cli_message=$(echo "$validation_data" | jq -r '.cli_tests.message')
    
    case "$cli_status" in
        present)
            echo "├── CLI Tests: ✅ $cli_message"
            ;;
        *)
            echo "├── CLI Tests: ⚠️  $cli_message"
            ;;
    esac
    
    # UI tests
    local ui_status=$(echo "$validation_data" | jq -r '.ui_tests.status')
    local ui_message=$(echo "$validation_data" | jq -r '.ui_tests.message')
    
    case "$ui_status" in
        present)
            echo "├── UI Tests: ✅ $ui_message"
            ;;
        not_applicable)
            echo "├── UI Tests: ℹ️  $ui_message"
            ;;
        missing)
            echo "├── UI Tests: ⚠️  $ui_message"
            ;;
        *)
            echo "├── UI Tests: ⚠️  $ui_message"
            ;;
    esac

    # Coverage
    local coverage_status=$(echo "$validation_data" | jq -r '.coverage.status')
    local coverage_message=$(echo "$validation_data" | jq -r '.coverage.message')
    local coverage_generated_at=$(echo "$validation_data" | jq -r '.coverage.generated_at // empty')
    if [[ -z "$coverage_message" || "$coverage_message" == "null" ]]; then
        coverage_message="Coverage artifacts not generated"
    fi

    local coverage_line="Coverage: ⚠️  $coverage_message"
    case "$coverage_status" in
        present)
            if [[ -n "$coverage_generated_at" && "$coverage_generated_at" != "null" ]]; then
                coverage_line="Coverage: ✅ $coverage_message (generated $coverage_generated_at)"
            else
                coverage_line="Coverage: ✅ $coverage_message"
            fi
            ;;
        *)
            coverage_line="Coverage: ⚠️  $coverage_message"
            ;;
    esac
    echo "└── $coverage_line"
    
    # Show recommendations if any
    local recommendations_count
    recommendations_count=$(echo "$validation_data" | jq -r '.overall.recommendations | length')

    local extra_recommendations="${SCENARIO_STATUS_EXTRA_RECOMMENDATIONS:-}"
    local extra_count=0
    if [[ -n "$extra_recommendations" ]]; then
        extra_count=$(printf '%s' "$extra_recommendations" | sed '/^$/d' | wc -l | tr -d ' ')
    fi

    if [[ "$recommendations_count" -gt 0 || "$extra_count" -gt 0 ]]; then
        echo ""
        echo "💡 Recommendations:"
        if [[ "$recommendations_count" -gt 0 ]]; then
            echo "$validation_data" | jq -r '.overall.recommendations[] | "   • " + .'
        fi
        if [[ "$extra_count" -gt 0 ]]; then
            printf '%s' "$extra_recommendations" | while IFS= read -r line; do
                [[ -n "$line" ]] && echo "   • $line"
            done
        fi
        echo ""

        # Build documentation links
        local doc_links=("$phased_docs_path")
        if [[ "${SCENARIO_STATUS_NEEDS_PRODUCTION_BUNDLE:-false}" == "true" ]]; then
            local prod_bundle_doc="${VROOLI_ROOT}/docs/scenarios/PRODUCTION_BUNDLES.md"
            doc_links+=("$prod_bundle_doc")
        fi

        if [[ -n "${SCENARIO_STATUS_EXTRA_DOC_LINKS:-}" ]]; then
            while IFS= read -r extra_doc; do
                [[ -z "$extra_doc" ]] && continue
                local already_present="false"
                for existing in "${doc_links[@]}"; do
                    if [[ "$existing" == "$extra_doc" ]]; then
                        already_present="true"
                        break
                    fi
                done
                if [[ "$already_present" == "false" ]]; then
                    doc_links+=("$extra_doc")
                fi
            done < <(printf '%s\n' "${SCENARIO_STATUS_EXTRA_DOC_LINKS}" | sed '/^$/d')
        fi

        if [[ ${#doc_links[@]} -gt 0 ]]; then
            echo "📚 Further reading:"
            for doc in "${doc_links[@]}"; do
                [[ -n "$doc" ]] && echo "   • $doc"
            done
        fi
    fi

    SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
    SCENARIO_STATUS_EXTRA_DOC_LINKS=""
}
