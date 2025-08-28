#!/usr/bin/env bash
# Unified Code Extractor Library
# Centralizes language detection and code extraction for all scenario extractors
#
# This library:
# 1. Sources all language parsers once
# 2. Provides unified language detection across directories
# 3. Dispatches to appropriate language extractors
# 4. Handles multi-language scenarios gracefully

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LIB_UTILS_DIR}/log.sh"

# Source ALL language parsers once
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/shell.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/javascript.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/go.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/python.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/rust.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/java.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/csharp.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/ruby.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/php.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/swift.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/kotlin.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/cpp.sh"
source "${APP_ROOT}/resources/qdrant/embeddings/parsers/sql.sh"

#######################################
# Detect primary language(s) in a directory
# 
# Uses configuration files first, then file counts as fallback
# Returns one or more languages separated by colons
#
# Arguments:
#   $1 - Path to directory
# Returns: Language name(s) separated by colons (e.g., "javascript:python")
#######################################
qdrant::lib::detect_languages() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        echo "unknown"
        return
    fi
    
    local languages=()
    
    # Check for configuration files (strongest signals)
    local has_package_json="false"
    local has_go_mod="false"
    local has_cargo_toml="false"
    local has_requirements_txt="false"
    local has_gemfile="false"
    local has_pom_xml="false"
    local has_csproj="false"
    local has_composer_json="false"
    local has_gradle="false"
    local has_package_swift="false"
    local has_cmake="false"
    
    [[ -f "$dir/package.json" ]] && has_package_json="true"
    [[ -f "$dir/go.mod" ]] && has_go_mod="true"
    [[ -f "$dir/Cargo.toml" ]] && has_cargo_toml="true"
    [[ -f "$dir/requirements.txt" ]] || [[ -f "$dir/setup.py" ]] && has_requirements_txt="true"
    [[ -f "$dir/Gemfile" ]] && has_gemfile="true"
    [[ -f "$dir/pom.xml" ]] && has_pom_xml="true"
    [[ -f "$dir"/*.csproj ]] && has_csproj="true"
    [[ -f "$dir/composer.json" ]] && has_composer_json="true"
    [[ -f "$dir/build.gradle" ]] || [[ -f "$dir/build.gradle.kts" ]] && has_gradle="true"
    [[ -f "$dir/Package.swift" ]] && has_package_swift="true"
    [[ -f "$dir/CMakeLists.txt" ]] && has_cmake="true"
    
    # Add languages based on config files
    [[ "$has_package_json" == "true" ]] && languages+=("javascript")
    [[ "$has_go_mod" == "true" ]] && languages+=("go")
    [[ "$has_cargo_toml" == "true" ]] && languages+=("rust")
    [[ "$has_requirements_txt" == "true" ]] && languages+=("python")
    [[ "$has_gemfile" == "true" ]] && languages+=("ruby")
    [[ "$has_pom_xml" == "true" ]] && languages+=("java")
    [[ "$has_csproj" == "true" ]] && languages+=("csharp")
    [[ "$has_composer_json" == "true" ]] && languages+=("php")
    [[ "$has_gradle" == "true" ]] && languages+=("kotlin")
    [[ "$has_package_swift" == "true" ]] && languages+=("swift")
    [[ "$has_cmake" == "true" ]] && languages+=("cpp")
    
    # If no config files found, fall back to file counts
    if [[ ${#languages[@]} -eq 0 ]]; then
        # Count files by extension
        local js_count=$(find "$dir" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \) 2>/dev/null | wc -l)
        local go_count=$(find "$dir" -type f -name "*.go" 2>/dev/null | wc -l)
        local py_count=$(find "$dir" -type f -name "*.py" 2>/dev/null | wc -l)
        local rust_count=$(find "$dir" -type f -name "*.rs" 2>/dev/null | wc -l)
        local java_count=$(find "$dir" -type f -name "*.java" 2>/dev/null | wc -l)
        local cs_count=$(find "$dir" -type f -name "*.cs" 2>/dev/null | wc -l)
        local rb_count=$(find "$dir" -type f -name "*.rb" 2>/dev/null | wc -l)
        local php_count=$(find "$dir" -type f -name "*.php" 2>/dev/null | wc -l)
        local swift_count=$(find "$dir" -type f -name "*.swift" 2>/dev/null | wc -l)
        local kotlin_count=$(find "$dir" -type f \( -name "*.kt" -o -name "*.kts" \) 2>/dev/null | wc -l)
        local cpp_count=$(find "$dir" -type f \( -name "*.cpp" -o -name "*.cc" -o -name "*.cxx" -o -name "*.h" -o -name "*.hpp" \) 2>/dev/null | wc -l)
        local shell_count=$(find "$dir" -type f \( -name "*.sh" -o -name "*.bash" \) 2>/dev/null | wc -l)
        local sql_count=$(find "$dir" -type f \( -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \) 2>/dev/null | wc -l)
        
        # Add languages with significant file counts (>0)
        [[ $js_count -gt 0 ]] && languages+=("javascript")
        [[ $go_count -gt 0 ]] && languages+=("go")
        [[ $py_count -gt 0 ]] && languages+=("python")
        [[ $rust_count -gt 0 ]] && languages+=("rust")
        [[ $java_count -gt 0 ]] && languages+=("java")
        [[ $cs_count -gt 0 ]] && languages+=("csharp")
        [[ $rb_count -gt 0 ]] && languages+=("ruby")
        [[ $php_count -gt 0 ]] && languages+=("php")
        [[ $swift_count -gt 0 ]] && languages+=("swift")
        [[ $kotlin_count -gt 0 ]] && languages+=("kotlin")
        [[ $cpp_count -gt 0 ]] && languages+=("cpp")
        [[ $shell_count -gt 0 ]] && languages+=("shell")
        [[ $sql_count -gt 0 ]] && languages+=("sql")
    fi
    
    # Return result
    if [[ ${#languages[@]} -eq 0 ]]; then
        echo "unknown"
    else
        # Join languages with colons
        local IFS=":"
        echo "${languages[*]}"
    fi
}

#######################################
# Detect primary language for a directory
# 
# Returns the most prominent language based on config and file counts
#
# Arguments:
#   $1 - Path to directory
# Returns: Single language name
#######################################
qdrant::lib::detect_primary_language() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        echo "unknown"
        return
    fi
    
    # Check for configuration files first (strongest signal)
    [[ -f "$dir/Cargo.toml" ]] && echo "rust" && return
    [[ -f "$dir/go.mod" ]] && echo "go" && return
    [[ -f "$dir/package.json" ]] && echo "javascript" && return
    [[ -f "$dir/requirements.txt" ]] || [[ -f "$dir/setup.py" ]] && echo "python" && return
    [[ -f "$dir/pom.xml" ]] && echo "java" && return
    [[ -f "$dir/build.gradle" ]] || [[ -f "$dir/build.gradle.kts" ]] && echo "kotlin" && return
    [[ -f "$dir"/*.csproj ]] && echo "csharp" && return
    [[ -f "$dir/Gemfile" ]] && echo "ruby" && return
    [[ -f "$dir/composer.json" ]] && echo "php" && return
    [[ -f "$dir/Package.swift" ]] && echo "swift" && return
    [[ -f "$dir/CMakeLists.txt" ]] && echo "cpp" && return
    
    # Fall back to file counts
    local max_count=0
    local primary_lang="unknown"
    
    # Count files for each language
    local js_count=$(find "$dir" -type f \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \) 2>/dev/null | wc -l)
    [[ $js_count -gt $max_count ]] && max_count=$js_count && primary_lang="javascript"
    
    local go_count=$(find "$dir" -type f -name "*.go" 2>/dev/null | wc -l)
    [[ $go_count -gt $max_count ]] && max_count=$go_count && primary_lang="go"
    
    local py_count=$(find "$dir" -type f -name "*.py" 2>/dev/null | wc -l)
    [[ $py_count -gt $max_count ]] && max_count=$py_count && primary_lang="python"
    
    local rust_count=$(find "$dir" -type f -name "*.rs" 2>/dev/null | wc -l)
    [[ $rust_count -gt $max_count ]] && max_count=$rust_count && primary_lang="rust"
    
    local java_count=$(find "$dir" -type f -name "*.java" 2>/dev/null | wc -l)
    [[ $java_count -gt $max_count ]] && max_count=$java_count && primary_lang="java"
    
    local kotlin_count=$(find "$dir" -type f \( -name "*.kt" -o -name "*.kts" \) 2>/dev/null | wc -l)
    [[ $kotlin_count -gt $max_count ]] && max_count=$kotlin_count && primary_lang="kotlin"
    
    local cs_count=$(find "$dir" -type f -name "*.cs" 2>/dev/null | wc -l)
    [[ $cs_count -gt $max_count ]] && max_count=$cs_count && primary_lang="csharp"
    
    local rb_count=$(find "$dir" -type f -name "*.rb" 2>/dev/null | wc -l)
    [[ $rb_count -gt $max_count ]] && max_count=$rb_count && primary_lang="ruby"
    
    local php_count=$(find "$dir" -type f -name "*.php" 2>/dev/null | wc -l)
    [[ $php_count -gt $max_count ]] && max_count=$php_count && primary_lang="php"
    
    local swift_count=$(find "$dir" -type f -name "*.swift" 2>/dev/null | wc -l)
    [[ $swift_count -gt $max_count ]] && max_count=$swift_count && primary_lang="swift"
    
    local cpp_count=$(find "$dir" -type f \( -name "*.cpp" -o -name "*.cc" -o -name "*.cxx" -o -name "*.h" -o -name "*.hpp" \) 2>/dev/null | wc -l)
    [[ $cpp_count -gt $max_count ]] && max_count=$cpp_count && primary_lang="cpp"
    
    local shell_count=$(find "$dir" -type f \( -name "*.sh" -o -name "*.bash" \) 2>/dev/null | wc -l)
    [[ $shell_count -gt $max_count ]] && max_count=$shell_count && primary_lang="shell"
    
    local sql_count=$(find "$dir" -type f \( -name "*.sql" -o -name "*.ddl" -o -name "*.dml" \) 2>/dev/null | wc -l)
    [[ $sql_count -gt $max_count ]] && max_count=$sql_count && primary_lang="sql"
    
    echo "$primary_lang"
}

#######################################
# Extract code from directory using appropriate language handler
# 
# Automatically detects language and calls the right extractor
#
# Arguments:
#   $1 - Path to directory
#   $2 - Component type (api, cli, ui, tests)
#   $3 - Scenario/resource name
#   $4 - Strategy: "primary" (single language), "all" (all languages), "auto" (detect)
# Returns: JSON lines of extracted code
#######################################
qdrant::lib::extract_code() {
    local dir="$1"
    local component_type="${2:-code}"
    local scenario_name="${3:-unknown}"
    local strategy="${4:-auto}"
    
    if [[ ! -d "$dir" ]]; then
        log::debug "Directory not found for code extraction: $dir" >&2
        return 1
    fi
    
    local extracted_any="false"
    
    if [[ "$strategy" == "primary" ]]; then
        # Extract using primary language only
        local primary_lang=$(qdrant::lib::detect_primary_language "$dir")
        
        if [[ "$primary_lang" != "unknown" ]]; then
            log::debug "Extracting $component_type code using primary language: $primary_lang" >&2
            qdrant::lib::extract_with_language "$dir" "$component_type" "$scenario_name" "$primary_lang"
            extracted_any="true"
        fi
    elif [[ "$strategy" == "all" ]]; then
        # Extract using all detected languages
        local languages=$(qdrant::lib::detect_languages "$dir")
        
        if [[ "$languages" != "unknown" ]]; then
            # Split languages and process each
            IFS=':' read -ra lang_array <<< "$languages"
            for lang in "${lang_array[@]}"; do
                log::debug "Extracting $component_type code for language: $lang" >&2
                qdrant::lib::extract_with_language "$dir" "$component_type" "$scenario_name" "$lang"
                extracted_any="true"
            done
        fi
    else
        # Auto mode: use primary for single-language, all for multi-language
        local languages=$(qdrant::lib::detect_languages "$dir")
        local lang_count=$(echo "$languages" | tr ':' '\n' | wc -l)
        
        if [[ "$languages" == "unknown" ]]; then
            # Try all extractors as fallback
            log::debug "Unknown language, trying all extractors for $component_type" >&2
            qdrant::lib::extract_all_languages "$dir" "$component_type" "$scenario_name"
        elif [[ $lang_count -eq 1 ]]; then
            # Single language detected
            log::debug "Extracting $component_type code using single language: $languages" >&2
            qdrant::lib::extract_with_language "$dir" "$component_type" "$scenario_name" "$languages"
            extracted_any="true"
        else
            # Multiple languages detected
            log::debug "Multiple languages detected, extracting all: $languages" >&2
            IFS=':' read -ra lang_array <<< "$languages"
            for lang in "${lang_array[@]}"; do
                qdrant::lib::extract_with_language "$dir" "$component_type" "$scenario_name" "$lang"
                extracted_any="true"
            done
        fi
    fi
    
    if [[ "$extracted_any" == "false" ]]; then
        log::debug "No code extracted from $dir" >&2
        return 1
    fi
}

#######################################
# Extract code using a specific language handler
# 
# Internal function that calls the appropriate language extractor
#
# Arguments:
#   $1 - Path to directory
#   $2 - Component type
#   $3 - Scenario/resource name
#   $4 - Language name
# Returns: JSON lines of extracted code
#######################################
qdrant::lib::extract_with_language() {
    local dir="$1"
    local component_type="$2"
    local scenario_name="$3"
    local language="$4"
    
    case "$language" in
        javascript)
            extractor::lib::javascript::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        go)
            extractor::lib::go::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        python)
            extractor::lib::python::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        rust)
            extractor::lib::rust::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        java)
            extractor::lib::java::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        kotlin)
            extractor::lib::kotlin::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        csharp)
            extractor::lib::csharp::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        ruby)
            extractor::lib::ruby::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        php)
            extractor::lib::php::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        swift)
            extractor::lib::swift::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        cpp)
            extractor::lib::cpp::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        shell)
            extractor::lib::shell::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        sql)
            extractor::lib::sql::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
            ;;
        *)
            log::debug "Unknown language for extraction: $language" >&2
            return 1
            ;;
    esac
}

#######################################
# Try extracting with all language handlers
# 
# Fallback function when language cannot be detected
#
# Arguments:
#   $1 - Path to directory
#   $2 - Component type
#   $3 - Scenario/resource name
# Returns: JSON lines from all successful extractors
#######################################
qdrant::lib::extract_all_languages() {
    local dir="$1"
    local component_type="$2"
    local scenario_name="$3"
    
    # Try all extractors (some will produce output, some won't)
    extractor::lib::javascript::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::go::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::python::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::rust::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::java::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::kotlin::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::csharp::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::ruby::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::php::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::swift::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::cpp::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::shell::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
    extractor::lib::sql::extract_all "$dir" "$component_type" "$scenario_name" 2>/dev/null || true
}

# Export all functions
export -f qdrant::lib::detect_languages
export -f qdrant::lib::detect_primary_language
export -f qdrant::lib::extract_code
export -f qdrant::lib::extract_with_language
export -f qdrant::lib::extract_all_languages