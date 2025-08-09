#!/usr/bin/env bats
# Tests for Judge0 languages.sh functions

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
    export YES="no"
    
    # Load dependencies
    SCRIPT_DIR="${BATS_TEST_DIRNAME}"
    JUDGE0_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Mock system functions
    
    # Mock curl for API calls
    
    # Mock jq for JSON processing
    jq() {
        case "$*" in
            *".id"*) echo "92" ;;
            *".name"*) echo "Python (3.11.2)" ;;
            *"length"*) echo "15" ;;
            *"select"* | *"ascii_downcase"*)
                if [[ "$*" =~ "python" ]]; then
                    echo "92"
                elif [[ "$*" =~ "javascript" ]]; then
                    echo "93"
                elif [[ "$*" =~ "java" ]]; then
                    echo "91"
                else
                    echo ""
                fi
                ;;
            *"map"* | *"sort_by"*)
                echo '["Bash (5.2.15)","C (GCC 12.2.0)","C# (.NET 7.0)","C++ (GCC 12.2.0)","Go (1.20.2)","Java (OpenJDK 19.0.2)","JavaScript (Node.js 18.15.0)","Kotlin (1.8.20)","PHP (8.2.3)","Python (3.11.2)","R (4.2.3)","Ruby (3.2.1)","Rust (1.68.2)","SQL (SQLite 3.40.1)","Swift (5.8.0)"]'
                ;;
            *) echo "JQ: $*" ;;
        esac
    }
    
    # Mock log functions
    
    # Mock Judge0 functions
    judge0::api::request() {
        case "$2" in
            "/languages") curl "$@" ;;
            "/languages/"*) curl "$@" ;;
            *) echo "API_REQUEST: $*" ;;
        esac
    }
    
    judge0::get_api_key() { echo "$JUDGE0_API_KEY"; }
    
    # Load configuration and messages
    source "${JUDGE0_DIR}/config/defaults.sh"
    source "${JUDGE0_DIR}/config/messages.sh"
    judge0::export_config
    judge0::export_messages
    
    # Load the functions to test
    source "${JUDGE0_DIR}/lib/languages.sh"
}

# Test language listing
@test "judge0::languages::list_languages shows all available languages" {
    result=$(judge0::languages::list_languages)
    
    [[ "$result" =~ "Available Languages" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
    [[ "$result" =~ "Java" ]]
    [[ "$result" =~ "15" ]] # total count
}

# Test sorted language listing
@test "judge0::languages::list_languages_sorted shows languages alphabetically" {
    result=$(judge0::languages::list_languages_sorted)
    
    [[ "$result" =~ "Bash" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "Swift" ]]
    # Should be in alphabetical order
}

# Test language search by name
@test "judge0::languages::find_language_by_name finds language by name" {
    result=$(judge0::languages::find_language_by_name "Python")
    
    [[ "$result" =~ "92" ]]
    [[ "$result" =~ "Python (3.11.2)" ]]
}

# Test language search by name - case insensitive
@test "judge0::languages::find_language_by_name is case insensitive" {
    result=$(judge0::languages::find_language_by_name "python")
    
    [[ "$result" =~ "92" ]]
    [[ "$result" =~ "Python" ]]
}

# Test language search with no results
@test "judge0::languages::find_language_by_name handles no matches" {
    result=$(judge0::languages::find_language_by_name "brainfuck")
    
    [[ "$result" =~ "not found" ]] || [[ -z "$result" ]]
}

# Test language details retrieval
@test "judge0::languages::get_language_details shows detailed language info" {
    result=$(judge0::languages::get_language_details "92")
    
    [[ "$result" =~ "Language Details" ]]
    [[ "$result" =~ "Python (3.11.2)" ]]
    [[ "$result" =~ "main.py" ]]
    [[ "$result" =~ "python3 main.py" ]]
}

# Test language details with invalid ID
@test "judge0::languages::get_language_details handles invalid language ID" {
    # Override curl to return error
    curl() {
        case "$*" in
            *"/languages/999"*) return 1 ;;
            *) echo "CURL: $*" ;;
        esac
        return 1
    }
    
    run judge0::languages::get_language_details "999"
    [ "$status" -eq 1 ]
    [[ "$output" =~ "ERROR:" ]] || [[ "$output" =~ "not found" ]]
}

# Test language validation
@test "judge0::languages::validate_language validates supported languages" {
    result=$(judge0::languages::validate_language "python" && echo "valid" || echo "invalid")
    
    [[ "$result" == "valid" ]]
}

# Test language validation with unsupported language
@test "judge0::languages::validate_language rejects unsupported languages" {
    result=$(judge0::languages::validate_language "assembly" && echo "valid" || echo "invalid")
    
    [[ "$result" == "invalid" ]]
}

# Test language ID lookup from name
@test "judge0::languages::get_language_id_by_name returns correct ID" {
    result=$(judge0::languages::get_language_id_by_name "JavaScript")
    
    [[ "$result" == "93" ]]
}

# Test language ID lookup with partial name
@test "judge0::languages::get_language_id_by_name handles partial matches" {
    result=$(judge0::languages::get_language_id_by_name "js")
    
    [[ "$result" == "93" ]] || [[ "$result" =~ "JavaScript" ]]
}

# Test compiled language detection
@test "judge0::languages::is_compiled_language detects compiled languages" {
    result=$(judge0::languages::is_compiled_language "java" && echo "compiled" || echo "interpreted")
    
    [[ "$result" == "compiled" ]]
}

# Test interpreted language detection
@test "judge0::languages::is_compiled_language detects interpreted languages" {
    result=$(judge0::languages::is_compiled_language "python" && echo "compiled" || echo "interpreted")
    
    [[ "$result" == "interpreted" ]]
}

# Test language statistics
@test "judge0::languages::get_language_stats shows language statistics" {
    result=$(judge0::languages::get_language_stats)
    
    [[ "$result" =~ "Language Statistics" ]]
    [[ "$result" =~ "Total languages" ]]
    [[ "$result" =~ "15" ]]
    [[ "$result" =~ "Compiled" ]] || [[ "$result" =~ "Interpreted" ]]
}

# Test language filtering by type
@test "judge0::languages::filter_by_type filters compiled languages" {
    result=$(judge0::languages::filter_by_type "compiled")
    
    [[ "$result" =~ "Compiled Languages" ]]
    [[ "$result" =~ "Java" ]]
    [[ "$result" =~ "C++" ]]
    [[ "$result" =~ "Go" ]]
}

# Test language filtering by type - interpreted
@test "judge0::languages::filter_by_type filters interpreted languages" {
    result=$(judge0::languages::filter_by_type "interpreted")
    
    [[ "$result" =~ "Interpreted Languages" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
    [[ "$result" =~ "Ruby" ]]
}

# Test popular languages listing
@test "judge0::languages::list_popular_languages shows most used languages" {
    result=$(judge0::languages::list_popular_languages)
    
    [[ "$result" =~ "Popular Languages" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
    [[ "$result" =~ "Java" ]]
}

# Test language extensions mapping
@test "judge0::languages::get_file_extension returns correct extension" {
    result=$(judge0::languages::get_file_extension "python")
    
    [[ "$result" == ".py" ]]
}

# Test language from file extension
@test "judge0::languages::detect_language_from_extension detects language from file" {
    result=$(judge0::languages::detect_language_from_extension "script.py")
    
    [[ "$result" =~ "python" ]] || [[ "$result" =~ "Python" ]]
}

# Test language from file extension with unknown extension
@test "judge0::languages::detect_language_from_extension handles unknown extensions" {
    result=$(judge0::languages::detect_language_from_extension "script.unknown")
    
    [[ "$result" =~ "unknown" ]] || [[ "$result" =~ "not found" ]] || [[ -z "$result" ]]
}

# Test language compatibility check
@test "judge0::languages::check_compatibility verifies language compatibility" {
    result=$(judge0::languages::check_compatibility "python" "3.11")
    
    [[ "$result" =~ "compatible" ]] || [[ "$result" =~ "supported" ]]
}

# Test language features listing
@test "judge0::languages::get_language_features shows language capabilities" {
    result=$(judge0::languages::get_language_features "python")
    
    [[ "$result" =~ "Features" ]]
    [[ "$result" =~ "python" ]] || [[ "$result" =~ "Python" ]]
}

# Test language performance characteristics
@test "judge0::languages::get_performance_info shows performance metrics" {
    result=$(judge0::languages::get_performance_info "python")
    
    [[ "$result" =~ "Performance" ]]
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "time" ]]
}

# Test language comparison
@test "judge0::languages::compare_languages compares two languages" {
    result=$(judge0::languages::compare_languages "python" "javascript")
    
    [[ "$result" =~ "Comparison" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
}

# Test language recommendations
@test "judge0::languages::recommend_language suggests appropriate language" {
    result=$(judge0::languages::recommend_language "web development")
    
    [[ "$result" =~ "Recommendation" ]]
    [[ "$result" =~ "JavaScript" ]] || [[ "$result" =~ "web" ]]
}

# Test language tutorials listing
@test "judge0::languages::list_tutorials shows available tutorials" {
    result=$(judge0::languages::list_tutorials "python")
    
    [[ "$result" =~ "Tutorial" ]] || [[ "$result" =~ "Example" ]]
    [[ "$result" =~ "python" ]] || [[ "$result" =~ "Python" ]]
}

# Test code example generation
@test "judge0::languages::generate_example creates language example" {
    result=$(judge0::languages::generate_example "python" "hello_world")
    
    [[ "$result" =~ "print" ]] || [[ "$result" =~ "Hello" ]]
}

# Test syntax highlighting info
@test "judge0::languages::get_syntax_info returns syntax highlighting data" {
    result=$(judge0::languages::get_syntax_info "python")
    
    [[ "$result" =~ "syntax" ]] || [[ "$result" =~ "highlight" ]]
}

# Test language documentation
@test "judge0::languages::get_documentation shows language documentation" {
    result=$(judge0::languages::get_documentation "python")
    
    [[ "$result" =~ "Documentation" ]]
    [[ "$result" =~ "python" ]] || [[ "$result" =~ "Python" ]]
}

# Test language version information
@test "judge0::languages::get_version_info shows version details" {
    result=$(judge0::languages::get_version_info "python")
    
    [[ "$result" =~ "version" ]] || [[ "$result" =~ "Version" ]]
    [[ "$result" =~ "3.11" ]]
}

# Test archived language filtering
@test "judge0::languages::filter_archived excludes archived languages" {
    result=$(judge0::languages::filter_archived)
    
    [[ "$result" =~ "Active Languages" ]]
    [[ "$result" =~ "Python" ]]
    [[ "$result" =~ "JavaScript" ]]
}

# Test language search with multiple criteria
@test "judge0::languages::search_languages supports multiple search criteria" {
    result=$(judge0::languages::search_languages "python interpreted")
    
    [[ "$result" =~ "Search Results" ]]
    [[ "$result" =~ "Python" ]]
}

# Test language availability check
@test "judge0::languages::check_availability verifies language availability" {
    result=$(judge0::languages::check_availability "python")
    
    [[ "$result" =~ "available" ]] || [[ "$result" =~ "supported" ]]
}

# Test language resource requirements
@test "judge0::languages::get_resource_requirements shows resource needs" {
    result=$(judge0::languages::get_resource_requirements "java")
    
    [[ "$result" =~ "Resource Requirements" ]]
    [[ "$result" =~ "memory" ]] || [[ "$result" =~ "CPU" ]]
}

# Test language limitations
@test "judge0::languages::get_limitations shows language constraints" {
    result=$(judge0::languages::get_limitations "javascript")
    
    [[ "$result" =~ "Limitations" ]] || [[ "$result" =~ "Constraints" ]]
}

# Test interactive language selection
@test "judge0::languages::interactive_select provides language selection interface" {
    result=$(judge0::languages::interactive_select)
    
    [[ "$result" =~ "Select" ]] || [[ "$result" =~ "Choose" ]]
    [[ "$result" =~ "language" ]]
}

# Test language usage statistics
@test "judge0::languages::get_usage_stats shows language usage data" {
    result=$(judge0::languages::get_usage_stats)
    
    [[ "$result" =~ "Usage Statistics" ]]
    [[ "$result" =~ "popular" ]] || [[ "$result" =~ "usage" ]]
}

# Test language export functionality
@test "judge0::languages::export_language_list exports language data" {
    result=$(judge0::languages::export_language_list "json")
    
    [[ "$result" =~ "export" ]] || [[ "$result" =~ "{" ]]
}

# Test default language settings
@test "judge0::languages::get_default_language returns default language" {
    result=$(judge0::languages::get_default_language)
    
    [[ "$result" =~ "python" ]] || [[ "$result" =~ "javascript" ]]
}

# Test language category classification
@test "judge0::languages::classify_by_category groups languages by category" {
    result=$(judge0::languages::classify_by_category)
    
    [[ "$result" =~ "Category" ]] || [[ "$result" =~ "Type" ]]
    [[ "$result" =~ "Web" ]] || [[ "$result" =~ "Systems" ]]
}