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

# Validate test infrastructure for a scenario
# Usage: scenario::test::validate_infrastructure <scenario_name> <scenario_path>
scenario::test::validate_infrastructure() {
    local scenario_name="$1"
    local scenario_path="$2"
    
    local validation_result="{}"
    
    # Check test lifecycle event
    local test_lifecycle_status
    test_lifecycle_status=$(scenario::test::check_test_lifecycle "$scenario_path")
    
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
    ui_tests_status=$(scenario::test::check_ui_tests "$scenario_path")
    
    # Calculate overall completeness
    local overall_status
    overall_status=$(scenario::test::calculate_overall_status "$test_lifecycle_status" "$phased_structure_status" "$unit_tests_status" "$cli_tests_status" "$ui_tests_status")
    
    # Build comprehensive validation result JSON
    validation_result=$(jq -n \
        --argjson test_lifecycle "$test_lifecycle_status" \
        --argjson phased_structure "$phased_structure_status" \
        --argjson unit_tests "$unit_tests_status" \
        --argjson cli_tests "$cli_tests_status" \
        --argjson ui_tests "$ui_tests_status" \
        --argjson overall "$overall_status" \
        '{
            test_lifecycle: $test_lifecycle,
            phased_structure: $phased_structure,
            unit_tests: $unit_tests,
            cli_tests: $cli_tests,
            ui_tests: $ui_tests,
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
        echo '{"status": "legacy", "message": "⚠️  Legacy test format (scenario-test.yaml)", "recommendation": "Migrate to new phased testing architecture", "format": "legacy"}'
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
            echo '{"status": "present", "message": "Test lifecycle event defined", "script": "'"$test_script"'"}'
        else
            echo '{"status": "invalid", "message": "Test lifecycle event malformed", "recommendation": "Fix test lifecycle event structure in .vrooli/service.json"}'
        fi
    fi
}

# Check phased testing structure
scenario::test::check_phased_structure() {
    local scenario_path="$1"
    local test_dir="$scenario_path/test"
    
    local status="missing"
    local message=""
    local recommendations=()
    local found_components=()
    
    # Check for test directory
    if [[ ! -d "$test_dir" ]]; then
        echo '{"status": "missing", "message": "No test/ directory found", "recommendation": "Create test/ directory with phased testing structure", "components": []}'
        return
    fi
    
    found_components+=("test_directory")
    
    # Check for main test runner
    if [[ -x "$test_dir/run-tests.sh" ]]; then
        found_components+=("test_runner")
        status="partial"
        message="Basic test structure present"
    else
        recommendations+=("Create executable test/run-tests.sh")
    fi
    
    # Check for phases directory
    if [[ -d "$test_dir/phases" ]]; then
        found_components+=("phases_directory")
        
        # Check for specific phase scripts
        local phases=("structure" "dependencies" "unit" "integration" "business" "performance")
        local found_phases=()
        
        for phase in "${phases[@]}"; do
            if [[ -f "$test_dir/phases/test-${phase}.sh" ]]; then
                found_phases+=("$phase")
            fi
        done
        
        if [[ ${#found_phases[@]} -gt 0 ]]; then
            found_components+=("phase_scripts")
            if [[ ${#found_phases[@]} -ge 4 ]]; then
                status="complete"
                message="Comprehensive phased testing structure"
            else
                status="partial"
                message="Some phase scripts present (${#found_phases[@]}/6)"
            fi
        else
            recommendations+=("Add phase scripts to test/phases/")
        fi
    else
        recommendations+=("Create test/phases/ directory with phase scripts")
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
        message="Test directory exists but is empty"
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
    if find "$scenario_path" -name "*_test.go" -type f | head -1 | grep -q .; then
        found_types+=("go")
    fi
    
    # Check for Node.js tests
    if [[ -f "$scenario_path/ui/package.json" ]]; then
        if grep -q '"test":' "$scenario_path/ui/package.json" 2>/dev/null; then
            local test_script
            test_script=$(jq -r '.scripts.test // ""' "$scenario_path/ui/package.json" 2>/dev/null)
            if [[ -n "$test_script" && "$test_script" != "null" ]]; then
                found_types+=("node")
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
    local ui_test_dir="$scenario_path/test/ui"
    local workflow_dir="$ui_test_dir/workflows"
    local found_workflows=()
    local status="none"
    local message="No UI automation tests found"
    
    # Check if scenario has UI component first
    local has_ui=false
    if [[ -d "$scenario_path/ui" ]] && [[ -f "$scenario_path/ui/package.json" ]]; then
        has_ui=true
    fi
    
    if [[ ! "$has_ui" == "true" ]]; then
        echo '{"status": "not_applicable", "message": "No UI component detected", "workflows": []}'
        return
    fi
    
    # Check for UI test workflows
    if [[ -d "$workflow_dir" ]]; then
        while IFS= read -r -d '' workflow_file; do
            local filename=$(basename "$workflow_file")
            found_workflows+=("$filename")
        done < <(find "$workflow_dir" -name "*.json" -type f -print0 2>/dev/null)
        
        if [[ ${#found_workflows[@]} -gt 0 ]]; then
            status="present"
            message="UI workflow tests found: ${#found_workflows[@]} workflow(s)"
        fi
    fi
    
    # If has UI but no tests, provide recommendation
    if [[ "$status" == "none" ]]; then
        message="UI component present but no automation tests found"
        status="missing"
    fi
    
    local workflows_json=$(printf '%s\n' "${found_workflows[@]}" | jq -R . | jq -s .)
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"workflows\": $workflows_json}"
}

# Calculate overall test infrastructure status
scenario::test::calculate_overall_status() {
    local test_lifecycle="$1"
    local phased_structure="$2"
    local unit_tests="$3"
    local cli_tests="$4" 
    local ui_tests="$5"

    local phased_docs_path
    phased_docs_path=$(scenario::test::absolute_path "docs/scenarios/PHASED_TESTING_ARCHITECTURE.md")
    
    local score=0
    local max_score=5
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
        recommendations+=("⚠️  Migrate from legacy scenario-test.yaml to new phased testing architecture")
        recommendations+=("See ${phased_docs_path} for migration guide")
    else
        recommendations+=("Define test lifecycle event in .vrooli/service.json")
    fi
    
    local structure_status=$(echo "$phased_structure" | jq -r '.status')
    case "$structure_status" in
        complete) ((score++)) ;;
        partial) score=$((score + 1)) ;; # Partial credit
        *) recommendations+=("Implement phased testing structure in test/ directory") ;;
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
    
    local recommendations_json=$(printf '%s\n' "${recommendations[@]}" | jq -R . | jq -s .)
    
    echo "{\"status\": \"$status\", \"message\": \"$message\", \"score\": $score, \"max_score\": $max_score, \"percentage\": $percentage, \"recommendations\": $recommendations_json}"
}

# Display test infrastructure validation in text format
scenario::test::display_validation() {
    local scenario_name="$1"
    local validation_data="$2"

    local phased_docs_path
    phased_docs_path=$(scenario::test::absolute_path "docs/scenarios/PHASED_TESTING_ARCHITECTURE.md")

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
            echo "└── UI Tests: ✅ $ui_message"
            ;;
        not_applicable)
            echo "└── UI Tests: ℹ️  $ui_message"
            ;;
        missing)
            echo "└── UI Tests: ⚠️  $ui_message"
            ;;
        *)
            echo "└── UI Tests: ⚠️  $ui_message"
            ;;
    esac
    
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
        echo "📚 See: $phased_docs_path"
    fi

    SCENARIO_STATUS_EXTRA_RECOMMENDATIONS=""
}
