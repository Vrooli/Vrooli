#!/usr/bin/env bats
# Tests for Judge0 usage.sh functions

# Setup for each test
setup() {
    # Load shared test infrastructure
    source "${BATS_TEST_DIRNAME}/../../../../__test/fixtures/setup.bash"
    
    # Setup standard mocks
    vrooli_auto_setup
    
    # Set test environment
    export JUDGE0_PORT="2358"
    export JUDGE0_BASE_URL="http://localhost:2358"
    export JUDGE0_API_KEY="test_api_key_12345"
    export JUDGE0_CONTAINER_NAME="judge0-test"
    export JUDGE0_DATA_DIR="/tmp/judge0-test"
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Create test directories and files
    mkdir -p "$JUDGE0_DATA_DIR"
    mkdir -p "${JUDGE0_DIR}/examples/basic"
    mkdir -p "${JUDGE0_DIR}/examples/ai-validation"
    mkdir -p "${JUDGE0_DIR}/examples/multi-language"
    mkdir -p "${JUDGE0_DIR}/examples/workflows"
    
    # Create example files
    echo 'console.log("Hello, World!");' > "${JUDGE0_DIR}/examples/basic/hello-world.js"
    echo 'name = input("Enter your name: "); print(f"Hello, {name}!")' > "${JUDGE0_DIR}/examples/basic/input-output.py"
    echo '# Fibonacci Example' > "${JUDGE0_DIR}/examples/multi-language/fibonacci.md"
    echo '#!/bin/bash' > "${JUDGE0_DIR}/examples/test-judge0.sh"
    chmod +x "${JUDGE0_DIR}/examples/test-judge0.sh"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".version"*) echo "1.13.1" ;;
            *".name"*) echo "Python (3.11.2)" ;;
            *"length"*) echo "3" ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock file reading commands
    cat() {
        case "$1" in
            */hello-world.js) echo 'console.log("Hello, World!");' ;;
            */input-output.py) echo 'name = input("Enter your name: "); print(f"Hello, {name}!")' ;;
            */fibonacci.md) echo '# Fibonacci Example\n\nThis demonstrates recursive algorithms.' ;;
            *) echo "FILE_CONTENT: $1" ;;
        esac
    }
    
    less() {
        echo "LESS_VIEWER: $*"
    }
    
    more() {
        echo "MORE_VIEWER: $*"
    }
    
    # Mock log functions
    
    # Mock Judge0 functions
    judge0::status::is_healthy() { return 0; }
    judge0::api::submit() {
        echo "Executing code: $1"
        echo "Language: $2"
        echo "✅ Execution successful"
        echo "📤 Output: Hello, World!"
    }
    judge0::languages::list_languages() {
        echo "Available Languages:"
        echo "  92 - Python (3.11.2)"
        echo "  93 - JavaScript (Node.js 18.15.0)"
        echo "  91 - Java (OpenJDK 19.0.2)"
    }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/usage.sh"
}

# Cleanup after each test
teardown() {
    rm -rf "$JUDGE0_DATA_DIR"
    rm -rf "${JUDGE0_DIR}/examples"
}

# Test usage information display
@test "judge0::usage::show_usage displays comprehensive usage information" {
    result=$(judge0::usage::show_usage)
    
    [[ "$result" =~ "Judge0 Usage" ]]
    [[ "$result" =~ "Examples" ]]
    [[ "$result" =~ "Getting Started" ]]
}

# Test quick start guide
@test "judge0::usage::show_quickstart shows quick start instructions" {
    result=$(judge0::usage::show_quickstart)
    
    [[ "$result" =~ "Quick Start" ]]
    [[ "$result" =~ "step" ]] || [[ "$result" =~ "1." ]]
    [[ "$result" =~ "Judge0" ]]
}

# Test example listing
@test "judge0::usage::list_examples shows available examples" {
    result=$(judge0::usage::list_examples)
    
    [[ "$result" =~ "Examples" ]]
    [[ "$result" =~ "hello-world.js" ]]
    [[ "$result" =~ "input-output.py" ]]
    [[ "$result" =~ "fibonacci" ]]
}

# Test example execution
@test "judge0::usage::run_example executes example code" {
    result=$(judge0::usage::run_example "hello-world.js")
    
    [[ "$result" =~ "Executing" ]]
    [[ "$result" =~ "hello-world.js" ]]
    [[ "$result" =~ "successful" ]]
    [[ "$result" =~ "Hello, World!" ]]
}

# Test example execution with missing file
@test "judge0::usage::run_example handles missing example file" {
    run judge0::usage::run_example "nonexistent.py"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "not found" ]]
}

# Test example display
@test "judge0::usage::show_example displays example code" {
    result=$(judge0::usage::show_example "hello-world.js")
    
    [[ "$result" =~ "Example: hello-world.js" ]]
    [[ "$result" =~ 'console.log("Hello, World!");' ]]
}

# Test interactive tutorial
@test "judge0::usage::interactive_tutorial provides guided tutorial" {
    result=$(judge0::usage::interactive_tutorial)
    
    [[ "$result" =~ "Tutorial" ]]
    [[ "$result" =~ "Judge0" ]]
    [[ "$result" =~ "step" ]] || [[ "$result" =~ "lesson" ]]
}

# Test basic tutorial steps
@test "judge0::usage::tutorial_basic_submission teaches basic code submission" {
    result=$(judge0::usage::tutorial_basic_submission)
    
    [[ "$result" =~ "Basic Submission" ]]
    [[ "$result" =~ "submit" ]] || [[ "$result" =~ "code" ]]
}

# Test language selection tutorial
@test "judge0::usage::tutorial_language_selection teaches language selection" {
    result=$(judge0::usage::tutorial_language_selection)
    
    [[ "$result" =~ "Language Selection" ]]
    [[ "$result" =~ "language" ]]
    [[ "$result" =~ "Python" ]] || [[ "$result" =~ "JavaScript" ]]
}

# Test common use cases
@test "judge0::usage::show_use_cases displays common usage scenarios" {
    result=$(judge0::usage::show_use_cases)
    
    [[ "$result" =~ "Use Cases" ]]
    [[ "$result" =~ "education" ]] || [[ "$result" =~ "testing" ]] || [[ "$result" =~ "validation" ]]
}

# Test best practices guide
@test "judge0::usage::show_best_practices shows usage best practices" {
    result=$(judge0::usage::show_best_practices)
    
    [[ "$result" =~ "Best Practices" ]]
    [[ "$result" =~ "performance" ]] || [[ "$result" =~ "security" ]] || [[ "$result" =~ "optimization" ]]
}

# Test troubleshooting guide
@test "judge0::usage::show_troubleshooting provides troubleshooting help" {
    result=$(judge0::usage::show_troubleshooting)
    
    [[ "$result" =~ "Troubleshooting" ]]
    [[ "$result" =~ "problem" ]] || [[ "$result" =~ "issue" ]] || [[ "$result" =~ "error" ]]
}

# Test FAQ display
@test "judge0::usage::show_faq displays frequently asked questions" {
    result=$(judge0::usage::show_faq)
    
    [[ "$result" =~ "FAQ" ]] || [[ "$result" =~ "Frequently Asked" ]]
    [[ "$result" =~ "Q:" ]] || [[ "$result" =~ "A:" ]]
}

# Test API documentation
@test "judge0::usage::show_api_docs displays API documentation" {
    result=$(judge0::usage::show_api_docs)
    
    [[ "$result" =~ "API Documentation" ]]
    [[ "$result" =~ "endpoint" ]] || [[ "$result" =~ "parameter" ]]
}

# Test code templates
@test "judge0::usage::show_templates displays code templates" {
    result=$(judge0::usage::show_templates)
    
    [[ "$result" =~ "Template" ]]
    [[ "$result" =~ "language" ]] || [[ "$result" =~ "example" ]]
}

# Test template creation
@test "judge0::usage::create_template creates code template" {
    result=$(judge0::usage::create_template "python" "hello_world")
    
    [[ "$result" =~ "Template created" ]]
    [[ "$result" =~ "python" ]]
    [[ "$result" =~ "hello_world" ]]
}

# Test configuration help
@test "judge0::usage::show_configuration_help displays configuration guidance" {
    result=$(judge0::usage::show_configuration_help)
    
    [[ "$result" =~ "Configuration" ]]
    [[ "$result" =~ "setting" ]] || [[ "$result" =~ "option" ]]
}

# Test performance tips
@test "judge0::usage::show_performance_tips provides performance optimization tips" {
    result=$(judge0::usage::show_performance_tips)
    
    [[ "$result" =~ "Performance" ]]
    [[ "$result" =~ "tip" ]] || [[ "$result" =~ "optimization" ]]
}

# Test security guidelines
@test "judge0::usage::show_security_guidelines displays security best practices" {
    result=$(judge0::usage::show_security_guidelines)
    
    [[ "$result" =~ "Security" ]]
    [[ "$result" =~ "guideline" ]] || [[ "$result" =~ "practice" ]]
}

# Test integration examples
@test "judge0::usage::show_integration_examples shows integration patterns" {
    result=$(judge0::usage::show_integration_examples)
    
    [[ "$result" =~ "Integration" ]]
    [[ "$result" =~ "example" ]] || [[ "$result" =~ "pattern" ]]
}

# Test workflow examples
@test "judge0::usage::show_workflow_examples displays workflow examples" {
    result=$(judge0::usage::show_workflow_examples)
    
    [[ "$result" =~ "Workflow" ]]
    [[ "$result" =~ "example" ]] || [[ "$result" =~ "pipeline" ]]
}

# Test batch processing examples
@test "judge0::usage::show_batch_examples demonstrates batch processing" {
    result=$(judge0::usage::show_batch_examples)
    
    [[ "$result" =~ "Batch" ]]
    [[ "$result" =~ "processing" ]] || [[ "$result" =~ "multiple" ]]
}

# Test CLI usage examples
@test "judge0::usage::show_cli_examples shows command line usage" {
    result=$(judge0::usage::show_cli_examples)
    
    [[ "$result" =~ "CLI" ]] || [[ "$result" =~ "Command" ]]
    [[ "$result" =~ "example" ]]
}

# Test educational content
@test "judge0::usage::show_educational_content provides learning resources" {
    result=$(judge0::usage::show_educational_content)
    
    [[ "$result" =~ "Educational" ]] || [[ "$result" =~ "Learning" ]]
    [[ "$result" =~ "resource" ]] || [[ "$result" =~ "content" ]]
}

# Test code review examples
@test "judge0::usage::show_code_review_examples demonstrates code review workflows" {
    result=$(judge0::usage::show_code_review_examples)
    
    [[ "$result" =~ "Code Review" ]]
    [[ "$result" =~ "example" ]] || [[ "$result" =~ "workflow" ]]
}

# Test testing examples
@test "judge0::usage::show_testing_examples shows automated testing patterns" {
    result=$(judge0::usage::show_testing_examples)
    
    [[ "$result" =~ "Testing" ]]
    [[ "$result" =~ "automated" ]] || [[ "$result" =~ "pattern" ]]
}

# Test validation examples
@test "judge0::usage::show_validation_examples demonstrates code validation" {
    result=$(judge0::usage::show_validation_examples)
    
    [[ "$result" =~ "Validation" ]]
    [[ "$result" =~ "code" ]] || [[ "$result" =~ "example" ]]
}

# Test monitoring examples
@test "judge0::usage::show_monitoring_examples shows monitoring integration" {
    result=$(judge0::usage::show_monitoring_examples)
    
    [[ "$result" =~ "Monitoring" ]]
    [[ "$result" =~ "integration" ]] || [[ "$result" =~ "example" ]]
}

# Test deployment examples
@test "judge0::usage::show_deployment_examples demonstrates deployment scenarios" {
    result=$(judge0::usage::show_deployment_examples)
    
    [[ "$result" =~ "Deployment" ]]
    [[ "$result" =~ "scenario" ]] || [[ "$result" =~ "example" ]]
}

# Test scaling examples
@test "judge0::usage::show_scaling_examples shows scaling strategies" {
    result=$(judge0::usage::show_scaling_examples)
    
    [[ "$result" =~ "Scaling" ]]
    [[ "$result" =~ "strategy" ]] || [[ "$result" =~ "example" ]]
}

# Test backup examples
@test "judge0::usage::show_backup_examples demonstrates backup procedures" {
    result=$(judge0::usage::show_backup_examples)
    
    [[ "$result" =~ "Backup" ]]
    [[ "$result" =~ "procedure" ]] || [[ "$result" =~ "example" ]]
}

# Test migration examples
@test "judge0::usage::show_migration_examples shows migration guides" {
    result=$(judge0::usage::show_migration_examples)
    
    [[ "$result" =~ "Migration" ]]
    [[ "$result" =~ "guide" ]] || [[ "$result" =~ "example" ]]
}

# Test custom examples
@test "judge0::usage::create_custom_example creates custom usage example" {
    result=$(judge0::usage::create_custom_example "fibonacci" "python")
    
    [[ "$result" =~ "Custom Example" ]]
    [[ "$result" =~ "fibonacci" ]]
    [[ "$result" =~ "python" ]]
}

# Test example validation
@test "judge0::usage::validate_example validates example code" {
    result=$(judge0::usage::validate_example "hello-world.js")
    
    [[ "$result" =~ "valid" ]] || [[ "$result" =~ "validation" ]]
    [[ "$result" =~ "hello-world.js" ]]
}

# Test usage statistics
@test "judge0::usage::show_usage_stats displays usage statistics" {
    result=$(judge0::usage::show_usage_stats)
    
    [[ "$result" =~ "Usage Statistics" ]]
    [[ "$result" =~ "statistic" ]] || [[ "$result" =~ "metric" ]]
}

# Test help system
@test "judge0::usage::show_help displays comprehensive help" {
    result=$(judge0::usage::show_help)
    
    [[ "$result" =~ "Help" ]]
    [[ "$result" =~ "Judge0" ]]
    [[ "$result" =~ "command" ]] || [[ "$result" =~ "option" ]]
}

# Test context-sensitive help
@test "judge0::usage::show_contextual_help provides context-specific help" {
    result=$(judge0::usage::show_contextual_help "api")
    
    [[ "$result" =~ "Help" ]]
    [[ "$result" =~ "api" ]] || [[ "$result" =~ "API" ]]
}

# Test interactive help
@test "judge0::usage::interactive_help provides interactive help system" {
    result=$(judge0::usage::interactive_help)
    
    [[ "$result" =~ "Interactive Help" ]]
    [[ "$result" =~ "help" ]] || [[ "$result" =~ "assistance" ]]
}

# Test documentation generation
@test "judge0::usage::generate_documentation creates usage documentation" {
    result=$(judge0::usage::generate_documentation)
    
    [[ "$result" =~ "Documentation" ]]
    [[ "$result" =~ "generated" ]] || [[ "$result" =~ "created" ]]
}

# Test usage export
@test "judge0::usage::export_usage_data exports usage information" {
    result=$(judge0::usage::export_usage_data "markdown")
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "usage" ]]
    [[ "$result" =~ "markdown" ]]
}

# Test service status check for usage
@test "judge0::usage::check_service_for_usage verifies service availability for examples" {
    result=$(judge0::usage::check_service_for_usage)
    
    [[ "$result" =~ "service" ]]
    [[ "$result" =~ "available" ]] || [[ "$result" =~ "ready" ]]
}

# Test example dependency check
@test "judge0::usage::check_example_dependencies verifies example requirements" {
    result=$(judge0::usage::check_example_dependencies)
    
    [[ "$result" =~ "dependencies" ]]
    [[ "$result" =~ "requirement" ]] || [[ "$result" =~ "check" ]]
}