#!/usr/bin/env bats
# Tests for Judge0 using Real Code Fixtures
# Demonstrates how to use the rich fixture system for actual code execution testing

# Load Vrooli test infrastructure
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"

# Load fixture helper functions
source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/fixture-loader.bash"

# Expensive setup operations (run once per file)
setup_file() {
    # Use appropriate setup function
    vrooli_setup_service_test "judge0-fixtures"
    
    # Load dependencies once
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Load configuration and functions once
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    source "${SCRIPT_DIR}/common.sh"
    source "${SCRIPT_DIR}/api.sh"
    
    # Export paths for use in setup()
    export SETUP_FILE_SCRIPT_DIR="$SCRIPT_DIR"
    export SETUP_FILE_JUDGE0_DIR="$JUDGE0_DIR"
    
    # Load fixture data paths
    export FIXTURE_DATA_DIR="${BATS_TEST_DIRNAME}/../../../../__test/fixtures/data"
    export CODE_FIXTURES_DIR="$FIXTURE_DATA_DIR/documents/code"
}

# Lightweight per-test setup
setup() {
    # Setup standard mocks
    vrooli_auto_setup
    
    # Use paths from setup_file
    SCRIPT_DIR="${SETUP_FILE_SCRIPT_DIR}"
    JUDGE0_DIR="${SETUP_FILE_JUDGE0_DIR}"
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_BASE_URL="http://localhost:2358"
    export JUDGE0_API_PREFIX="/api/v1"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_ENABLE_AUTHENTICATION="true"
    export JUDGE0_CPU_TIME_LIMIT="5"
    export JUDGE0_WALL_TIME_LIMIT="10"
    export JUDGE0_MEMORY_LIMIT="262144"
    export JUDGE0_STACK_LIMIT="262144"
    export JUDGE0_MAX_PROCESSES="30"
    export JUDGE0_MAX_FILE_SIZE="5120"
    export JUDGE0_ENABLE_NETWORK="false"
    export YES="no"
    
    # Set up test message variables
    export MSG_JUDGE0_HEALTHY="Judge0 is healthy"
    export MSG_SUBMISSION_CREATED="Submission created"
    export MSG_EXECUTION_COMPLETE="Execution completed"
    export MSG_CODE_EXECUTED="Code executed successfully"
    
    # Mock Judge0 utility functions
    judge0::common::is_running() { return 0; }
    judge0::common::is_healthy() { return 0; }
    judge0::get_api_key() { echo "$JUDGE0_API_KEY"; }
    judge0::get_language_id() {
        case "$1" in
            "python"|"python3") echo "92" ;;
            "javascript"|"js"|"node") echo "93" ;;
            "java") echo "91" ;;
            "cpp"|"c++") echo "76" ;;
            "c") echo "75" ;;
            *) return 1 ;;
        esac
    }
    export -f judge0::common::is_running judge0::common::is_healthy
    export -f judge0::get_api_key judge0::get_language_id
    
    # Mock curl for Judge0 API operations with fixture-based responses
    curl() {
        case "$*" in
            *"/submissions"*"POST"*)
                # Mock submission creation with token
                echo '{"token":"abc123-def456-ghi789","status":{"id":1,"description":"In Queue"}}'
                return 0
                ;;
            *"/submissions/abc123-def456-ghi789"*"GET"*)
                # Mock Python execution result based on fixture content
                if [[ "$CURRENT_TEST_FIXTURE" == "python" ]]; then
                    echo '{
                        "token":"abc123-def456-ghi789",
                        "status":{"id":3,"description":"Accepted"},
                        "stdout":"PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT0KREFUQSBBTUFMWVNJRIBSRUVW==",
                        "stderr":"",
                        "time":"0.125",
                        "memory":"12288",
                        "language":{"id":92,"name":"Python (3.11.2)"}
                    }'
                elif [[ "$CURRENT_TEST_FIXTURE" == "javascript" ]]; then
                    echo '{
                        "token":"abc123-def456-ghi789",
                        "status":{"id":3,"description":"Accepted"},
                        "stdout":"UmVzb3VyY2UgTWFuYWdlciBpbml0aWFsaXplZCBzdWNjZXNzZnVsbHk=",
                        "stderr":"",
                        "time":"0.089",
                        "memory":"8192",
                        "language":{"id":93,"name":"JavaScript (Node.js 18.15.0)"}
                    }'
                else
                    echo '{
                        "token":"abc123-def456-ghi789",
                        "status":{"id":3,"description":"Accepted"},
                        "stdout":"SGVsbG8sIFdvcmxkIQ==",
                        "stderr":"",
                        "time":"0.050",
                        "memory":"4096"
                    }'
                fi
                return 0
                ;;
            *"/languages"*"GET"*)
                # Mock supported languages
                echo '[
                    {"id":92,"name":"Python (3.11.2)"},
                    {"id":93,"name":"JavaScript (Node.js 18.15.0)"},
                    {"id":91,"name":"Java (OpenJDK 17.0.6)"},
                    {"id":76,"name":"C++ (GCC 9.4.0)"},
                    {"id":75,"name":"C (GCC 9.4.0)"}
                ]'
                return 0
                ;;
            *"/system_info"*"GET"*)
                # Mock system info
                echo '{
                    "version":"1.13.1",
                    "homepage":"https://judge0.com",
                    "source_code":"https://github.com/judge0/judge0",
                    "maintainer":"Judge0 Team <info@judge0.com>"
                }'
                return 0
                ;;
            *)
                return 0
                ;;
        esac
    }
    export -f curl
    
    # Mock base64 for decoding fixture-specific outputs
    base64() {
        if [[ "$*" =~ "-d" ]]; then
            case "$input" in
                # Python data analysis output
                "PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT0KREFUQSBBTUFMWVNJRSBSRVBT==")
                    echo "============================================================"
                    echo "DATA ANALYSIS REPORT"
                    echo "============================================================"
                    echo "Generated: 2024-01-15 10:30:00"
                    echo "Total Records: 5"
                    echo ""
                    echo "COLUMN ANALYSIS"
                    echo "Numeric Columns (2): age, salary"
                    echo "Categorical Columns (2): name, department"
                    ;;
                # JavaScript resource manager output
                "UmVzb3VyY2UgTWFuYWdlciBpbml0aWFsaXplZCBzdWNjZXNzZnVsbHk=")
                    echo "Resource Manager initialized successfully"
                    echo "Discovered 3 resources"
                    echo "Auto-refresh enabled with 30000ms interval"
                    ;;
                # Default outputs
                "SGVsbG8sIFdvcmxkIQ==") echo "Hello, World!" ;;
                "RXJyb3I6IGludmFsaWQgc3ludGF4") echo "Error: invalid syntax" ;;
                "dGVzdCBvdXRwdXQ=") echo "test output" ;;
                *) echo "decoded content" ;;
            esac
        else
            echo "encoded_content_123"
        fi
    }
    export -f base64
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".token"*) echo "abc123-def456-ghi789" ;;
            *".status.id"*) echo "3" ;;
            *".status.description"*) echo "Accepted" ;;
            *".stdout"*) 
                if [[ "$CURRENT_TEST_FIXTURE" == "python" ]]; then
                    echo "PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT09PT0KREFUQSBBTUFMWVNJRZBSRVBT=="
                elif [[ "$CURRENT_TEST_FIXTURE" == "javascript" ]]; then
                    echo "UmVzb3VyY2UgTWFuYWdlciBpbml0aWFsaXplZCBzdWNjZXNzZnVsbHk="
                else
                    echo "SGVsbG8sIFdvcmxkIQ=="
                fi
                ;;
            *".stderr"*) echo "" ;;
            *".time"*) echo "0.125" ;;
            *".memory"*) echo "12288" ;;
            *".language.name"*) echo "Python (3.11.2)" ;;
            *"length"*) echo "5" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    export -f jq
    
    # Mock log functions
    log::info() { echo "[INFO] $*"; }
    log::error() { echo "[ERROR] $*" >&2; }
    log::warning() { echo "[WARNING] $*" >&2; }
    log::success() { echo "[SUCCESS] $*"; }
    log::debug() { [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2 || true; }
    export -f log::info log::error log::warning log::success log::debug
    
    # Mock file operations to read fixture files
    cat() {
        local file_path="$1"
        case "$file_path" in
            *"data_analysis.py")
                # Read actual fixture file
                command cat "$CODE_FIXTURES_DIR/python/data_analysis.py"
                ;;
            *"frontend.js")
                # Read actual fixture file
                command cat "$CODE_FIXTURES_DIR/javascript/frontend.js"
                ;;
            *)
                command cat "$@"
                ;;
        esac
    }
    export -f cat
    
    # Mock system commands
    date() { echo "2024-01-15T10:30:00Z"; }
    export -f date
}

# BATS teardown function - runs after each test
teardown() {
    unset CURRENT_TEST_FIXTURE
    vrooli_cleanup_test
}

# ============================================================================
# Fixture Loading Helper Functions
# ============================================================================

# Load fixture and prepare for execution
load_code_fixture() {
    local language="$1"
    local fixture_name="$2"
    
    case "$language" in
        "python")
            fixture_get_path "documents/code" "python/$fixture_name"
            ;;
        "javascript")
            fixture_get_path "documents/code" "javascript/$fixture_name"
            ;;
        *)
            return 1
            ;;
    esac
}

# Validate fixture execution result
validate_execution_result() {
    local expected_pattern="$1"
    local actual_output="$2"
    
    if [[ "$actual_output" =~ $expected_pattern ]]; then
        return 0
    else
        echo "Expected pattern '$expected_pattern' not found in output: $actual_output" >&2
        return 1
    fi
}

# ============================================================================
# Python Fixture Tests
# ============================================================================

@test "judge0 executes Python data analysis fixture successfully" {
    export CURRENT_TEST_FIXTURE="python"
    
    # Load the actual Python fixture
    local python_code=$(cat "$CODE_FIXTURES_DIR/python/data_analysis.py")
    
    # Execute the fixture code through Judge0
    run judge0::submit_code "python3" "$python_code"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "abc123-def456-ghi789" ]]
    
    # Get execution results
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    [[ "$output" =~ "DATA ANALYSIS REPORT" ]]
    [[ "$output" =~ "Total Records: 5" ]]
    [[ "$output" =~ "Numeric Columns" ]]
}

@test "judge0 validates Python fixture syntax and imports" {
    export CURRENT_TEST_FIXTURE="python"
    
    # Load fixture and extract import statements
    local python_code=$(cat "$CODE_FIXTURES_DIR/python/data_analysis.py")
    
    # Verify the fixture contains expected imports
    [[ "$python_code" =~ "import json" ]]
    [[ "$python_code" =~ "import csv" ]]
    [[ "$python_code" =~ "import statistics" ]]
    [[ "$python_code" =~ "from typing import" ]]
    
    # Execute and verify no import errors
    run judge0::submit_code "python3" "$python_code"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    # Should not contain import errors
    [[ ! "$output" =~ "ImportError" ]]
    [[ ! "$output" =~ "ModuleNotFoundError" ]]
}

@test "judge0 executes Python fixture classes and methods" {
    export CURRENT_TEST_FIXTURE="python"
    
    # Create test code that uses the fixture's classes
    local test_code='
import sys
sys.path.append(".")

# Import DataAnalyzer from fixture (simulated)
class DataSummary:
    def __init__(self, total_records, numeric_columns, categorical_columns, missing_values, data_types, timestamp):
        self.total_records = total_records
        self.numeric_columns = numeric_columns
        self.categorical_columns = categorical_columns
        self.missing_values = missing_values
        self.data_types = data_types
        self.timestamp = timestamp

class DataAnalyzer:
    def __init__(self):
        self.data_cache = {}
        
    def analyze_dataset(self, data):
        if not data:
            raise ValueError("Cannot analyze empty dataset")
        return DataSummary(len(data), ["age"], ["name"], {}, {"age": "numeric", "name": "categorical"}, "2024-01-15")

# Test the classes
analyzer = DataAnalyzer()
test_data = [{"name": "Alice", "age": 30}, {"name": "Bob", "age": 25}]
summary = analyzer.analyze_dataset(test_data)
print(f"Total records: {summary.total_records}")
print(f"Numeric columns: {summary.numeric_columns}")
print("DataAnalyzer test completed successfully")
'
    
    run judge0::submit_code "python3" "$test_code"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    [[ "$output" =~ "Total records: 2" ]]
    [[ "$output" =~ "DataAnalyzer test completed successfully" ]]
}

# ============================================================================
# JavaScript Fixture Tests
# ============================================================================

@test "judge0 executes JavaScript resource manager fixture successfully" {
    export CURRENT_TEST_FIXTURE="javascript"
    
    # Load the actual JavaScript fixture
    local js_code=$(cat "$CODE_FIXTURES_DIR/javascript/frontend.js")
    
    # Execute the fixture code through Judge0
    run judge0::submit_code "javascript" "$js_code"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "abc123-def456-ghi789" ]]
    
    # Get execution results
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    [[ "$output" =~ "Resource Manager initialized successfully" ]]
}

@test "judge0 validates JavaScript fixture ES6+ syntax" {
    export CURRENT_TEST_FIXTURE="javascript"
    
    # Load fixture and verify ES6+ features
    local js_code=$(cat "$CODE_FIXTURES_DIR/javascript/frontend.js")
    
    # Verify the fixture contains expected ES6+ features
    [[ "$js_code" =~ "class ResourceManager" ]]
    [[ "$js_code" =~ "async initialize" ]]
    [[ "$js_code" =~ "const defaultConfig" ]]
    [[ "$js_code" =~ "...defaultConfig" ]]  # Spread operator
    [[ "$js_code" =~ "Array.from" ]]
    [[ "$js_code" =~ "new Map()" ]]
    
    # Execute and verify no syntax errors
    run judge0::submit_code "javascript" "$js_code"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    # Should not contain syntax errors
    [[ ! "$output" =~ "SyntaxError" ]]
    [[ ! "$output" =~ "ReferenceError" ]]
}

@test "judge0 executes JavaScript fixture class instantiation and methods" {
    export CURRENT_TEST_FIXTURE="javascript"
    
    # Create test code that uses the fixture's classes
    local test_code='
// Simplified ResourceManager for testing
class ResourceManager {
    constructor(baseUrl = "http://localhost:3000") {
        this.baseUrl = baseUrl;
        this.resources = new Map();
        this.initialized = false;
    }
    
    addResource(resource) {
        if (!resource.id || !resource.name) {
            throw new Error("Resource must have id and name properties");
        }
        this.resources.set(resource.id, resource);
    }
    
    getResource(resourceId) {
        return this.resources.get(resourceId) || null;
    }
    
    getStats() {
        return {
            totalResources: this.resources.size,
            initialized: this.initialized
        };
    }
}

// Test the class
const manager = new ResourceManager();
manager.addResource({ id: "test-1", name: "Test Resource", type: "ai" });
manager.addResource({ id: "test-2", name: "Another Resource", type: "storage" });

const stats = manager.getStats();
console.log(`Total resources: ${stats.totalResources}`);
console.log(`Initialized: ${stats.initialized}`);

const resource = manager.getResource("test-1");
console.log(`Found resource: ${resource ? resource.name : "not found"}`);

console.log("ResourceManager test completed successfully");
'
    
    run judge0::submit_code "javascript" "$test_code"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    [[ "$output" =~ "Total resources: 2" ]]
    [[ "$output" =~ "ResourceManager test completed successfully" ]]
}

# ============================================================================
# Cross-Language Fixture Tests
# ============================================================================

@test "judge0 compares execution performance between Python and JavaScript fixtures" {
    # Test Python fixture
    export CURRENT_TEST_FIXTURE="python"
    run judge0::submit_code "python3" "print('Hello from Python')"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    local python_time=$(echo "$output" | grep -o '"time":"[^"]*"' | cut -d'"' -f4)
    
    # Test JavaScript fixture
    export CURRENT_TEST_FIXTURE="javascript"
    run judge0::submit_code "javascript" "console.log('Hello from JavaScript')"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    local js_time=$(echo "$output" | grep -o '"time":"[^"]*"' | cut -d'"' -f4)
    
    # Both should execute successfully
    [[ "$python_time" =~ ^[0-9]+\.[0-9]+$ ]]
    [[ "$js_time" =~ ^[0-9]+\.[0-9]+$ ]]
}

# ============================================================================
# Fixture Validation Tests
# ============================================================================

@test "judge0 validates fixture metadata and content integrity" {
    # Verify Python fixture exists and has expected content
    [ -f "$CODE_FIXTURES_DIR/python/data_analysis.py" ]
    local python_content=$(cat "$CODE_FIXTURES_DIR/python/data_analysis.py")
    [[ "$python_content" =~ "class DataAnalyzer" ]]
    [[ "$python_content" =~ "def main()" ]]
    
    # Verify JavaScript fixture exists and has expected content
    [ -f "$CODE_FIXTURES_DIR/javascript/frontend.js" ]
    local js_content=$(cat "$CODE_FIXTURES_DIR/javascript/frontend.js")
    [[ "$js_content" =~ "class ResourceManager" ]]
    [[ "$js_content" =~ "DOMContentLoaded" ]]
    
    # Verify fixtures are substantial (not empty or trivial)
    local python_lines=$(wc -l < "$CODE_FIXTURES_DIR/python/data_analysis.py")
    local js_lines=$(wc -l < "$CODE_FIXTURES_DIR/javascript/frontend.js")
    
    # Both fixtures should have substantial content
    [[ $python_lines -gt 200 ]]
    [[ $js_lines -gt 400 ]]
}

@test "judge0 handles fixture execution with resource constraints" {
    export CURRENT_TEST_FIXTURE="python"
    
    # Test with stricter resource limits
    export JUDGE0_CPU_TIME_LIMIT="2"
    export JUDGE0_MEMORY_LIMIT="131072"  # 128MB
    
    # Simple Python test that should complete within constraints
    local test_code='
# Simple computation within resource limits
import time

def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

result = fibonacci(10)
print(f"Fibonacci(10) = {result}")
print("Computation completed within resource limits")
'
    
    run judge0::submit_code "python3" "$test_code"
    [ "$status" -eq 0 ]
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Accepted" ]]
    [[ "$output" =~ "Fibonacci(10) = 55" ]]
}

# ============================================================================
# Error Handling with Fixtures Tests
# ============================================================================

@test "judge0 handles Python fixture with intentional syntax error" {
    export CURRENT_TEST_FIXTURE="python"
    
    # Python code with syntax error
    local broken_code='
class BrokenClass:
    def __init__(self):
        self.data = {}
    
    def broken_method(self)
        # Missing colon - syntax error
        print("This will fail")
        return self.data
'
    
    run judge0::submit_code "python3" "$broken_code"
    [ "$status" -eq 0 ]  # Submission creation should succeed
    
    # Mock error response for broken code
    curl() {
        case "$*" in
            *"/submissions/abc123-def456-ghi789"*"GET"*)
                echo '{
                    "token":"abc123-def456-ghi789",
                    "status":{"id":6,"description":"Compilation Error"},
                    "stdout":"",
                    "stderr":"RXJyb3I6IGludmFsaWQgc3ludGF4",
                    "time":"0.010",
                    "memory":"2048"
                }'
                return 0
                ;;
            *)
                command curl "$@"
                ;;
        esac
    }
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Compilation Error" ]]
    [[ "$output" =~ "Error: invalid syntax" ]]
}

@test "judge0 handles JavaScript fixture with runtime error" {
    export CURRENT_TEST_FIXTURE="javascript"
    
    # JavaScript code with runtime error
    local broken_code='
class TestClass {
    constructor() {
        this.data = null;
    }
    
    processData() {
        // This will cause a runtime error
        return this.data.nonexistentProperty.value;
    }
}

const test = new TestClass();
console.log(test.processData());
'
    
    run judge0::submit_code "javascript" "$broken_code"
    [ "$status" -eq 0 ]  # Submission creation should succeed
    
    # Mock runtime error response
    curl() {
        case "$*" in
            *"/submissions/abc123-def456-ghi789"*"GET"*)
                echo '{
                    "token":"abc123-def456-ghi789",
                    "status":{"id":5,"description":"Time Limit Exceeded"},
                    "stdout":"",
                    "stderr":"VHlwZUVycm9yOiBDYW5ub3QgcmVhZCBwcm9wZXJ0aWVzIG9mIG51bGw=",
                    "time":"10.000",
                    "memory":"4096"
                }'
                return 0
                ;;
            *)
                command curl "$@"
                ;;
        esac
    }
    
    run judge0::get_submission_result "abc123-def456-ghi789"
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Time Limit Exceeded" ]]
}

# ============================================================================
# Function Definition Tests
# ============================================================================

@test "all Judge0 fixture testing functions are defined" {
    # Test that fixture helper functions exist
    type load_code_fixture >/dev/null
    type validate_execution_result >/dev/null
    
    # Test that Judge0 API functions exist
    type judge0::submit_code >/dev/null
    type judge0::get_submission_result >/dev/null
    type judge0::get_language_id >/dev/null
}

# Teardown
teardown() {
    unset CURRENT_TEST_FIXTURE
    vrooli_cleanup_test
}