#!/usr/bin/env bash
# Scenario Tests Extractor
# Extracts test configurations and test code from scenarios
#
# Handles:
# - scenario-test.yaml files
# - test/ directories with various test frameworks
# - Integration and unit tests

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source unified code extractor (handles all language detection and dispatch)
source "${APP_ROOT}/resources/qdrant/embeddings/lib/code-extractor.sh"

#######################################
# Extract scenario test configuration
# 
# Parses scenario-test.yaml files for test definitions
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with test configuration
#######################################
qdrant::extract::scenario_test_configuration() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    
    # Look for test configuration files
    local test_config=""
    [[ -f "$dir/scenario-test.yaml" ]] && test_config="$dir/scenario-test.yaml"
    [[ -f "$dir/scenario-test.yml" ]] && test_config="${test_config:-$dir/scenario-test.yml}"
    
    if [[ -z "$test_config" ]]; then
        return 1
    fi
    
    log::debug "Extracting test configuration for $scenario_name" >&2
    
    local test_data="{}"
    if command -v yq &>/dev/null; then
        test_data=$(yq eval -o=json '.' "$test_config" 2>/dev/null || echo "{}")
    else
        # Basic YAML parsing
        local test_name=$(grep "^name:" "$test_config" | cut -d: -f2- | tr -d ' "' || echo "")
        local description=$(grep "^description:" "$test_config" | cut -d: -f2- | tr -d '"' | sed 's/^ *//' || echo "")
        
        # Count test steps
        local step_count=$(grep -c "^  - " "$test_config" 2>/dev/null || echo "0")
        
        test_data=$(jq -n \
            --arg name "$test_name" \
            --arg description "$description" \
            --arg step_count "$step_count" \
            '{name: $name, description: $description, step_count: ($step_count | tonumber)}')
    fi
    
    # Extract key information
    local test_name=$(echo "$test_data" | jq -r '.name // empty')
    local description=$(echo "$test_data" | jq -r '.description // empty')
    local test_count=$(echo "$test_data" | jq '.tests | length' 2>/dev/null || echo "0")
    local step_count=$(echo "$test_data" | jq '.step_count // 0')
    
    # Extract test types and environments
    local test_types_json=$(echo "$test_data" | jq -c '.types // []' 2>/dev/null || echo "[]")
    local environments_json=$(echo "$test_data" | jq -c '.environments // []' 2>/dev/null || echo "[]")
    local dependencies_json=$(echo "$test_data" | jq -c '.dependencies // []' 2>/dev/null || echo "[]")
    
    # Count items
    local type_count=$(echo "$test_types_json" | jq 'length')
    local env_count=$(echo "$environments_json" | jq 'length')
    local dep_count=$(echo "$dependencies_json" | jq 'length')
    
    # Build content
    local content="Test Config: $test_name | Scenario: $scenario_name"
    [[ -n "$description" ]] && content="$content | Description: $description"
    content="$content | Tests: $test_count | Steps: $step_count"
    
    if [[ $type_count -gt 0 ]]; then
        content="$content | Types: $(echo "$test_types_json" | jq -r 'join(", ")')"
    fi
    
    if [[ $env_count -gt 0 ]]; then
        content="$content | Environments: $(echo "$environments_json" | jq -r 'join(", ")')"
    fi
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_file "$test_config" \
        --arg test_name "$test_name" \
        --arg description "$description" \
        --arg test_count "$test_count" \
        --arg step_count "$step_count" \
        --argjson test_types "$test_types_json" \
        --argjson environments "$environments_json" \
        --argjson dependencies "$dependencies_json" \
        --argjson full_config "$test_data" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_file: $source_file,
                component_type: "tests",
                test_config_type: "scenario_test",
                test_name: $test_name,
                description: $description,
                test_count: ($test_count | tonumber),
                step_count: ($step_count | tonumber),
                test_types: $test_types,
                environments: $environments,
                dependencies: $dependencies,
                full_configuration: $full_config,
                content_type: "scenario_tests",
                extraction_method: "scenario_test_config_parser"
            }
        }' | jq -c
}

#######################################
# Detect test framework and language
# 
# Analyzes test/ directory to determine test framework
#
# Arguments:
#   $1 - Path to test directory
# Returns: Framework name and language
#######################################
qdrant::extract::detect_test_framework() {
    local test_dir="$1"
    
    if [[ ! -d "$test_dir" ]]; then
        echo "none:unknown"
        return
    fi
    
    local framework="unknown"
    local language="unknown"
    
    # Check for framework-specific files
    if [[ -f "$test_dir/package.json" ]]; then
        language="javascript"
        
        # Check package.json for test frameworks
        if grep -q "jest" "$test_dir/package.json" 2>/dev/null; then
            framework="jest"
        elif grep -q "mocha" "$test_dir/package.json" 2>/dev/null; then
            framework="mocha"
        elif grep -q "cypress" "$test_dir/package.json" 2>/dev/null; then
            framework="cypress"
        elif grep -q "playwright" "$test_dir/package.json" 2>/dev/null; then
            framework="playwright"
        else
            framework="javascript"
        fi
    elif [[ -f "$test_dir/go.mod" ]] || find "$test_dir" -name "*_test.go" 2>/dev/null | grep -q .; then
        language="go"
        framework="go_test"
    elif [[ -f "$test_dir/pytest.ini" ]] || [[ -f "$test_dir/requirements.txt" ]] || find "$test_dir" -name "test_*.py" 2>/dev/null | grep -q .; then
        language="python"
        if [[ -f "$test_dir/pytest.ini" ]]; then
            framework="pytest"
        else
            framework="python_unittest"
        fi
    elif [[ -f "$test_dir/Cargo.toml" ]] || find "$test_dir" -name "*test*.rs" 2>/dev/null | grep -q .; then
        language="rust"
        framework="rust_test"
    elif [[ -f "$test_dir/pom.xml" ]] || find "$test_dir" -name "*Test.java" 2>/dev/null | grep -q .; then
        language="java"
        if grep -q "junit" "$test_dir/pom.xml" 2>/dev/null; then
            framework="junit"
        else
            framework="java_test"
        fi
    elif [[ -f "$test_dir/build.gradle" ]] || [[ -f "$test_dir/build.gradle.kts" ]] || find "$test_dir" -name "*Test.kt" 2>/dev/null | grep -q .; then
        language="kotlin"
        framework="kotlin_test"
    elif [[ -f "$test_dir"/*.csproj ]] || find "$test_dir" -name "*Test.cs" -o -name "*Tests.cs" 2>/dev/null | grep -q .; then
        language="csharp"
        if grep -q "xunit\|nunit\|mstest" "$test_dir"/*.csproj 2>/dev/null; then
            framework="dotnet_test"
        else
            framework="csharp_test"
        fi
    elif [[ -f "$test_dir/composer.json" ]] || find "$test_dir" -name "*Test.php" 2>/dev/null | grep -q .; then
        language="php"
        if grep -q "phpunit" "$test_dir/composer.json" 2>/dev/null; then
            framework="phpunit"
        else
            framework="php_test"
        fi
    elif [[ -f "$test_dir/Package.swift" ]] || find "$test_dir" -name "*Test.swift" -o -name "*Tests.swift" 2>/dev/null | grep -q .; then
        language="swift"
        framework="swift_test"
    elif [[ -f "$test_dir/CMakeLists.txt" ]] || find "$test_dir" -name "*test*.cpp" -o -name "*test*.cc" 2>/dev/null | grep -q .; then
        language="cpp"
        if grep -q "gtest\|catch" "$test_dir/CMakeLists.txt" 2>/dev/null; then
            framework="gtest"
        else
            framework="cpp_test"
        fi
    elif [[ -f "$test_dir/Gemfile" ]] || find "$test_dir" -name "*_spec.rb" -o -name "*test*.rb" 2>/dev/null | grep -q .; then
        language="ruby"
        if grep -q "rspec" "$test_dir/Gemfile" 2>/dev/null; then
            framework="rspec"
        else
            framework="ruby_test"
        fi
    elif find "$test_dir" -name "*.bats" 2>/dev/null | grep -q . || find "$test_dir" -name "test_*.sh" 2>/dev/null | grep -q .; then
        language="shell"
        if find "$test_dir" -name "*.bats" 2>/dev/null | grep -q .; then
            framework="bats"
        else
            framework="shell"
        fi
    fi
    
    echo "$framework:$language"
}

#######################################
# Extract test directory overview
# 
# Provides high-level information about test implementation
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON line with test overview
#######################################
qdrant::extract::scenario_test_overview() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local test_dir="$dir/test"
    
    if [[ ! -d "$test_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting test overview for $scenario_name" >&2
    
    # Detect framework and language
    local framework_info=$(qdrant::extract::detect_test_framework "$test_dir")
    local framework="${framework_info%:*}"
    local language="${framework_info#*:}"
    
    # Count different file types
    local total_files=$(find "$test_dir" -type f 2>/dev/null | wc -l)
    local test_files=0
    local config_files=0
    local spec_files=0
    
    # Count by category
    while IFS= read -r file; do
        case "$file" in
            *_test.*|test_*.*|*.test.*|*.spec.*|*.bats)
                ((test_files++))
                ;;
            *.json|*.yaml|*.yml|*.toml|*.ini|*.conf)
                ((config_files++))
                ;;
            *spec*|*Spec*)
                ((spec_files++))
                ;;
        esac
    done < <(find "$test_dir" -type f 2>/dev/null)
    
    # Check for common test patterns
    local has_unit_tests="false"
    local has_integration_tests="false"
    local has_e2e_tests="false"
    local has_fixtures="false"
    
    # Look for test type indicators
    [[ -d "$test_dir/unit" ]] || find "$test_dir" -name "*unit*" 2>/dev/null | grep -q . && has_unit_tests="true"
    [[ -d "$test_dir/integration" ]] || find "$test_dir" -name "*integration*" 2>/dev/null | grep -q . && has_integration_tests="true"
    [[ -d "$test_dir/e2e" ]] || find "$test_dir" -name "*e2e*" 2>/dev/null | grep -q . && has_e2e_tests="true"
    [[ -d "$test_dir/fixtures" ]] || [[ -d "$test_dir/data" ]] && has_fixtures="true"
    
    # Build content
    local content="Tests: $scenario_name | Framework: $framework | Language: $language"
    content="$content | Files: $total_files | TestFiles: $test_files"
    [[ "$has_unit_tests" == "true" ]] && content="$content | Unit: yes"
    [[ "$has_integration_tests" == "true" ]] && content="$content | Integration: yes"
    [[ "$has_e2e_tests" == "true" ]] && content="$content | E2E: yes"
    [[ "$has_fixtures" == "true" ]] && content="$content | Fixtures: yes"
    
    # Output as JSON line
    jq -n \
        --arg content "$content" \
        --arg scenario "$scenario_name" \
        --arg source_dir "$test_dir" \
        --arg framework "$framework" \
        --arg language "$language" \
        --arg total_files "$total_files" \
        --arg test_files "$test_files" \
        --arg config_files "$config_files" \
        --arg has_unit_tests "$has_unit_tests" \
        --arg has_integration_tests "$has_integration_tests" \
        --arg has_e2e_tests "$has_e2e_tests" \
        --arg has_fixtures "$has_fixtures" \
        '{
            content: $content,
            metadata: {
                scenario: $scenario,
                source_directory: $source_dir,
                component_type: "tests",
                test_framework: $framework,
                test_language: $language,
                total_files: ($total_files | tonumber),
                test_files: ($test_files | tonumber),
                config_files: ($config_files | tonumber),
                has_unit_tests: ($has_unit_tests == "true"),
                has_integration_tests: ($has_integration_tests == "true"),
                has_e2e_tests: ($has_e2e_tests == "true"),
                has_fixtures: ($has_fixtures == "true"),
                content_type: "scenario_tests",
                extraction_method: "test_overview_analyzer"
            }
        }' | jq -c
}

#######################################
# Extract test implementation details
# 
# Uses appropriate language handler to extract test code
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with test implementation details
#######################################
qdrant::extract::scenario_test_implementation() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local test_dir="$dir/test"
    
    if [[ ! -d "$test_dir" ]]; then
        return 1
    fi
    
    log::debug "Extracting test implementation for $scenario_name" >&2
    
    # Use unified code extractor with auto strategy (handles single and multi-language)
    qdrant::lib::extract_code "$test_dir" "tests" "$scenario_name" "auto"
        javascript)
            extractor::lib::javascript::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        go)
            extractor::lib::go::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        python)
            extractor::lib::python::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        rust)
            extractor::lib::rust::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        java)
            extractor::lib::java::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        kotlin)
            extractor::lib::kotlin::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        csharp)
            extractor::lib::csharp::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        php)
            extractor::lib::php::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        swift)
            extractor::lib::swift::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        cpp)
            extractor::lib::cpp::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        ruby)
            extractor::lib::ruby::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        shell)
            extractor::lib::shell::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            ;;
        *)
            log::debug "Unknown test language, attempting multi-language extraction" >&2
            # Try all extractors for mixed test implementations
            extractor::lib::shell::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::javascript::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::go::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::python::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::rust::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::java::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::kotlin::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::csharp::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::php::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::swift::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
            extractor::lib::cpp::extract_all "$test_dir" "tests" "$scenario_name" 2>/dev/null || true
}

#######################################
# Extract test fixtures and data
# 
# Processes test data files and fixtures
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines with test fixtures information
#######################################
qdrant::extract::scenario_test_fixtures() {
    local dir="$1"
    local scenario_name=$(basename "$dir")
    local test_dir="$dir/test"
    
    if [[ ! -d "$test_dir" ]]; then
        return 1
    fi
    
    # Look for fixture directories
    local fixture_dirs=()
    [[ -d "$test_dir/fixtures" ]] && fixture_dirs+=("$test_dir/fixtures")
    [[ -d "$test_dir/data" ]] && fixture_dirs+=("$test_dir/data")
    [[ -d "$test_dir/mocks" ]] && fixture_dirs+=("$test_dir/mocks")
    
    for fixture_dir in "${fixture_dirs[@]}"; do
        local fixture_type=$(basename "$fixture_dir")
        local file_count=$(find "$fixture_dir" -type f 2>/dev/null | wc -l)
        
        if [[ $file_count -gt 0 ]]; then
            # Count by file type
            local json_count=$(find "$fixture_dir" -name "*.json" 2>/dev/null | wc -l)
            local yaml_count=$(find "$fixture_dir" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
            local txt_count=$(find "$fixture_dir" -name "*.txt" 2>/dev/null | wc -l)
            
            local content="Fixtures: $fixture_type | Tests: $scenario_name | Files: $file_count"
            [[ $json_count -gt 0 ]] && content="$content | JSON: $json_count"
            [[ $yaml_count -gt 0 ]] && content="$content | YAML: $yaml_count"
            [[ $txt_count -gt 0 ]] && content="$content | Text: $txt_count"
            
            jq -n \
                --arg content "$content" \
                --arg scenario "$scenario_name" \
                --arg source_dir "$fixture_dir" \
                --arg fixture_type "$fixture_type" \
                --arg file_count "$file_count" \
                --arg json_count "$json_count" \
                --arg yaml_count "$yaml_count" \
                --arg txt_count "$txt_count" \
                '{
                    content: $content,
                    metadata: {
                        scenario: $scenario,
                        source_directory: $source_dir,
                        component_type: "tests",
                        fixture_type: $fixture_type,
                        file_count: ($file_count | tonumber),
                        json_files: ($json_count | tonumber),
                        yaml_files: ($yaml_count | tonumber),
                        text_files: ($txt_count | tonumber),
                        content_type: "scenario_tests",
                        extraction_method: "test_fixtures_analyzer"
                    }
                }' | jq -c
        fi
    done
}

#######################################
# Extract all test information
# 
# Main function that calls all test extractors
#
# Arguments:
#   $1 - Path to scenario directory
# Returns: JSON lines for all test components
#######################################
qdrant::extract::scenario_tests_all() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        return 1
    fi
    
    # Extract scenario test configuration (scenario-test.yaml)
    qdrant::extract::scenario_test_configuration "$dir" 2>/dev/null || true
    
    # Extract test directory information (if test/ exists)
    if [[ -d "$dir/test" ]]; then
        # Extract test overview
        qdrant::extract::scenario_test_overview "$dir" 2>/dev/null || true
        
        # Extract test implementation
        qdrant::extract::scenario_test_implementation "$dir" 2>/dev/null || true
        
        # Extract test fixtures
        qdrant::extract::scenario_test_fixtures "$dir" 2>/dev/null || true
    fi
}

# Export functions for use by main.sh
export -f qdrant::extract::scenario_test_configuration
export -f qdrant::extract::detect_test_framework
export -f qdrant::extract::scenario_test_overview
export -f qdrant::extract::scenario_test_implementation
export -f qdrant::extract::scenario_test_fixtures
export -f qdrant::extract::scenario_tests_all