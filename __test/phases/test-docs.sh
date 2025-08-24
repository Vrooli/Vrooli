#!/usr/bin/env bash
# Documentation Validation Phase - Vrooli Testing Infrastructure
# 
# Validates all markdown documentation for:
# - Markdown syntax and formatting (markdownlint)
# - File references and internal links (remark-validate-links)
# 
# Uses npm-based tools with intelligent caching to avoid re-testing unchanged files.

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/__test"
PROJECT_ROOT="${PROJECT_ROOT:-$APP_ROOT}"

# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/logging.bash"
# shellcheck disable=SC1091  
source "$SCRIPT_DIR/shared/test-helpers.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/cache.bash"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/shared/scoping.bash"

show_usage() {
    cat << 'EOF'
Documentation Validation Phase - Test markdown syntax and file references

Usage: ./test-docs.sh [OPTIONS]

OPTIONS:
  --verbose        Show detailed output for each file tested
  --no-cache       Skip caching optimizations
  --dry-run        Show what would be tested without running
  --clear-cache    Clear documentation validation cache before running
  --resource=NAME  Test only specific resource docs (e.g., --resource=ollama)
  --scenario=NAME  Test only specific scenario docs (e.g., --scenario=app-generator)
  --path=PATH      Test only specific directory/file path
  --help           Show this help

ENVIRONMENT:
  DOCS_MAX_CORES Set max parallel workers (default: 4 or half available cores)

WHAT THIS PHASE TESTS:
  1. Markdown syntax and formatting using markdownlint
  2. File path references exist and are valid
  3. Internal heading anchors are correct
  4. Image and code example files exist
  5. Relative paths point to real files

SCOPING EXAMPLES:
  ./test-docs.sh --resource=ollama          # Only test ollama documentation
  ./test-docs.sh --scenario=app-generator   # Only test app-generator docs
  ./test-docs.sh --path=docs/architecture   # Only test docs in specific path

EXAMPLES:
  ./test-docs.sh                      # Validate all documentation
  ./test-docs.sh --verbose            # Show detailed validation output
  ./test-docs.sh --no-cache           # Force re-validation of all files
  ./test-docs.sh --clear-cache        # Clear cache and re-validate
EOF
}

# Parse command line arguments
CLEAR_CACHE=""
SCOPE_RESOURCE=""
SCOPE_SCENARIO=""
SCOPE_PATH=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --clear-cache)
            CLEAR_CACHE="true"
            shift
            ;;
        --resource=*)
            SCOPE_RESOURCE="${1#*=}"
            shift
            ;;
        --scenario=*)
            SCOPE_SCENARIO="${1#*=}"
            shift
            ;;
        --path=*)
            SCOPE_PATH="${1#*=}"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        --verbose|--parallel|--no-cache|--dry-run)
            # These are handled by the main orchestrator
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Export scope variables for the scoping functions
export SCOPE_RESOURCE
export SCOPE_SCENARIO
export SCOPE_PATH

# Set number of parallel workers (default to 4 or half of available cores, whichever is smaller)
MAX_CORES="${DOCS_MAX_CORES:-}"
if [[ -z "$MAX_CORES" ]]; then
    AVAILABLE_CORES=$(nproc 2>/dev/null || echo 8)
    HALF_CORES=$((AVAILABLE_CORES / 2))
    MAX_CORES=$((HALF_CORES > 4 ? 4 : HALF_CORES))
    MAX_CORES=$((MAX_CORES < 1 ? 1 : MAX_CORES))
fi

# Clear cache if requested
if [[ -n "$CLEAR_CACHE" ]]; then
    clear_cache "docs"
fi

# Check and install npm dependencies if needed
check_npm_dependencies() {
    log_info "🔧 Checking npm dependencies..."
    
    # Check if node and npm are available
    if ! command -v node >/dev/null 2>&1; then
        log_error "Node.js is not installed. Please install Node.js to run documentation tests."
        log_info "Visit https://nodejs.org/ or use your package manager"
        return 1
    fi
    
    if ! command -v npm >/dev/null 2>&1; then
        log_error "npm is not installed. Please install npm to run documentation tests."
        return 1
    fi
    
    # Check if package.json exists, create if not
    if [[ ! -f "$SCRIPT_DIR/package.json" ]]; then
        log_info "Creating package.json for documentation test dependencies..."
        cat > "$SCRIPT_DIR/package.json" << 'EOFJSON'
{
  "name": "vrooli-test-infrastructure",
  "version": "1.0.0",
  "private": true,
  "description": "Testing infrastructure for Vrooli documentation",
  "devDependencies": {
    "markdownlint-cli2": "^0.11.0",
    "remark-cli": "^12.0.0",
    "remark-validate-links": "^13.0.0",
    "remark-frontmatter": "^5.0.0"
  }
}
EOFJSON
    fi
    
    # Check if node_modules exists, install if not
    if [[ ! -d "$SCRIPT_DIR/node_modules" ]]; then
        log_info "Installing npm dependencies (first time setup)..."
        if is_dry_run; then
            log_info "[DRY RUN] Would run: npm install --prefix $SCRIPT_DIR"
        else
            (cd "$SCRIPT_DIR" && npm install --silent)
            log_success "Dependencies installed successfully"
        fi
    fi
    
    return 0
}

# Create configuration files if they don't exist
setup_config_files() {
    local config_dir="$SCRIPT_DIR/config"
    
    # Create config directory if it doesn't exist
    if [[ ! -d "$config_dir" ]]; then
        mkdir -p "$config_dir"
    fi
    
    # Create markdownlint configuration
    if [[ ! -f "$config_dir/.markdownlint.yaml" ]]; then
        log_info "Creating markdownlint configuration..."
        cat > "$config_dir/.markdownlint.yaml" << 'EOFYAML'
# Markdownlint configuration for Vrooli documentation
# Sensible defaults that allow flexibility for documentation

default: true

# MD013: Line length - documentation can have long lines
MD013: false

# MD033: Allow inline HTML (often needed in docs)
MD033: false

# MD041: First line doesn't need to be h1 (we use frontmatter)
MD041: false

# MD024: Multiple headers with same content
MD024:
  siblings_only: true  # Allow duplicate headings in different sections

# MD026: Trailing punctuation in headers (we allow it)
MD026: false

# MD034: Bare URLs (sometimes needed in docs)
MD034: false

# MD040: Fenced code blocks should have a language specified
MD040: true

# MD046: Code block style (consistent)
MD046:
  style: fenced
EOFYAML
    fi
    
    # Create remark configuration
    if [[ ! -f "$config_dir/.remarkrc.yaml" ]]; then
        log_info "Creating remark configuration..."
        cat > "$config_dir/.remarkrc.yaml" << 'EOFYAML'
# Remark configuration for link validation
plugins:
  - - remark-validate-links
    - 
      # Work with local files only (no repository URL needed)
      repository: false
      
      # File extensions to check when referenced
      fileExtensions: 
        - md
        - markdown
        - sh
        - bash
        - ts
        - tsx
        - js
        - jsx
        - json
        - yaml
        - yml
        - txt
        - sql
        
      # Ignore external URLs (we only validate local references)
      urlWhitelist:
        - "^https?://"
        - "^mailto:"
        - "^ftp://"
        - "^#"  # Allow pure anchor links
        
  # Support YAML frontmatter in markdown files
  - remark-frontmatter
EOFYAML
    fi
}

# Find all markdown files to test
find_markdown_files() {
    local scope_desc
    scope_desc=$(get_scope_description)
    log_info "Finding markdown files for $scope_desc..."
    
    # Use scoped file discovery if scope is set
    if [[ -n "${SCOPE_RESOURCE:-}" ]] || [[ -n "${SCOPE_SCENARIO:-}" ]] || [[ -n "${SCOPE_PATH:-}" ]]; then
        # Get scoped markdown files (.md and .markdown) with null termination for consumer compatibility
        {
            get_scoped_files "*.md" "$PROJECT_ROOT"
            get_scoped_files "*.markdown" "$PROJECT_ROOT"
        } | tr '\n' '\0'
    else
        # No scope - find all markdown files, excluding build directories
        find "$PROJECT_ROOT" \
            -type f \( -name "*.md" -o -name "*.markdown" \) \
            -not -path "*/node_modules/*" \
            -not -path "*/.git/*" \
            -not -path "*/dist/*" \
            -not -path "*/build/*" \
            -not -path "*/.next/*" \
            -not -path "*/coverage/*" \
            -not -path "*/.cache/*" \
            -print0 2>/dev/null | sort -z
    fi
}

# Get file hash for caching
get_file_hash() {
    local file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | cut -d' ' -f1
    elif command -v md5sum >/dev/null 2>&1; then
        md5sum "$file" | cut -d' ' -f1
    else
        # Fallback to file modification time if no hash command available
        stat -c "%Y" "$file" 2>/dev/null || stat -f "%m" "$file" 2>/dev/null || echo "nohash"
    fi
}

# Run markdownlint on a file
run_markdownlint() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "markdownlint" "$relative_path"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would validate syntax: $relative_path"
        return 0
    fi
    
    # Run markdownlint
    local output
    if output=$(npx --prefix "$SCRIPT_DIR" markdownlint-cli2 \
        --config "$SCRIPT_DIR/config/.markdownlint.yaml" \
        "$file" 2>&1); then
        log_test_pass "markdownlint" "$relative_path"
        update_cache "$file" "markdownlint" "passed" 0 "" 0
        return 0
    else
        log_test_fail "markdownlint" "$relative_path"
        # Show specific errors from markdownlint output
        # Parse markdownlint output for specific errors (format: file:line:col MDxxx/rule Description)
        echo "$output" | grep -E "^[^:]+:[0-9]+" | head -5 | while IFS= read -r line; do
            # Remove the file path prefix to make output cleaner
            local error_detail
            error_detail=$(echo "$line" | sed "s|^$file:||")
            log_warning "    └─ $error_detail"
        done
        
        # Count total errors and show if there are more
        local error_count
        error_count=$(echo "$output" | grep -c "^[^:]+:[0-9]+" 2>/dev/null || echo "0")
        # Ensure error_count is a valid number
        error_count=${error_count//[^0-9]/}
        error_count=${error_count:-0}
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more errors"
        fi
        
        update_cache "$file" "markdownlint" "failed" 0 "$output" 1
        return 1
    fi
}

# Run remark-validate-links on a file
run_remark_validate_links() {
    local file="$1"
    local relative_path
    relative_path=$(relative_path "$file")
    
    log_test_start "link-validation" "$relative_path"
    
    if is_dry_run; then
        log_info "[DRY RUN] Would validate links: $relative_path"
        return 0
    fi
    
    # Run remark with validate-links plugin
    local output
    if output=$(npx --prefix "$SCRIPT_DIR" remark \
        --rc-path "$SCRIPT_DIR/config/.remarkrc.yaml" \
        --quiet \
        --frail \
        "$file" 2>&1); then
        log_test_pass "link-validation" "$relative_path"
        update_cache "$file" "remark-links" "passed" 0 "" 0
        return 0
    else
        log_test_fail "link-validation" "$relative_path"
        # Show specific link validation errors
        # Look for common remark-validate-links error patterns
        echo "$output" | grep -E "(Cannot find file|Cannot find heading|warning|missing)" | head -5 | while IFS= read -r line; do
            # Extract just the error message part
            local clean_line
            # Remove file path prefix and clean up spacing
            clean_line=$(echo "$line" | sed -E 's/^[[:space:]]+//' | sed -E "s|^[^:]+:[0-9]+:[0-9]+-[0-9]+:[0-9]+[[:space:]]+||")
            log_warning "    └─ $clean_line"
        done
        
        # Count total issues and show if there are more
        local error_count
        error_count=$(echo "$output" | grep -cE "(Cannot find|warning)" 2>/dev/null || echo "0")
        # Ensure error_count is a valid number
        error_count=${error_count//[^0-9]/}
        error_count=${error_count:-0}
        if [[ "$error_count" -gt 5 ]]; then
            log_warning "    └─ ... and $((error_count - 5)) more issues"
        fi
        
        update_cache "$file" "remark-links" "failed" 0 "$output" 1
        return 1
    fi
}

# Main test execution
main() {
    local start_time
    start_time=$(date +%s)
    
    log_section "Documentation Validation Phase"
    
    # Load cache for this phase
    load_cache "docs"
    
    # Check and setup dependencies
    if ! check_npm_dependencies; then
        log_error "Failed to setup npm dependencies"
        return 1
    fi
    
    # Setup configuration files
    setup_config_files
    
    # Find all markdown files
    log_info "🔍 Discovering markdown files..."
    local markdown_files=()
    while IFS= read -r -d '' file; do
        markdown_files+=("$file")
    done < <(find_markdown_files)
    
    local total_files=${#markdown_files[@]}
    
    if [[ $total_files -eq 0 ]]; then
        log_warning "No markdown files found to validate"
        return 0
    fi
    
    log_info "📋 Found $total_files markdown files to validate"
    
    # Optional: limit files for testing/debugging
    if [[ -n "${MAX_DOCS_FILES:-}" ]] && [[ "${MAX_DOCS_FILES}" -lt "$total_files" ]]; then
        log_warning "⚠️  Limiting to first $MAX_DOCS_FILES files (unset MAX_DOCS_FILES to test all)"
        markdown_files=("${markdown_files[@]:0:$MAX_DOCS_FILES}")
        total_files=$MAX_DOCS_FILES
    fi
    
    if is_dry_run; then
        log_info "[DRY RUN] Would validate $total_files markdown files with $MAX_CORES parallel workers"
        return 0
    fi
    
    # Filter out cached files first
    log_info "🔍 Checking cache for previously validated files..."
    local files_to_test=()
    local syntax_cached=0
    local links_cached=0
    
    for file in "${markdown_files[@]}"; do
        local needs_test=false
        
        # Check markdownlint cache
        if ! check_cache "$file" "markdownlint"; then
            needs_test=true
        else
            ((syntax_cached++)) || true
        fi
        
        # Check remark-links cache
        if ! check_cache "$file" "remark-links"; then
            needs_test=true
        else
            ((links_cached++)) || true
        fi
        
        # Add to test list if either test needs to run
        if [[ "$needs_test" == "true" ]]; then
            files_to_test+=("$file")
        fi
    done
    
    local num_to_test=${#files_to_test[@]}
    log_info "📊 Cache results: $syntax_cached syntax + $links_cached links already validated"
    log_info "📦 Need to test: $num_to_test files"
    
    # Test counters (start with cached counts)
    local syntax_passed=$syntax_cached
    local syntax_failed=0
    local links_passed=$links_cached
    local links_failed=0
    
    if [[ $num_to_test -eq 0 ]]; then
        log_success "All files validated from cache!"
    else
        log_info "📦 Processing $num_to_test files in parallel (using $MAX_CORES workers)..."
    
    # Create temp file for results
    local results_file
    results_file=$(mktemp)
    
    # Export functions and variables for subshells
    export -f run_markdownlint run_remark_validate_links
    export -f log_test_start log_test_pass log_test_fail log_warning
    export -f relative_path get_file_hash check_cache update_cache load_cache save_cache
    export -f is_verbose is_dry_run timestamp
    export SCRIPT_DIR PROJECT_ROOT
    export CACHE_PHASE CACHE_MODIFIED
    export -A CACHE_DATA
    
    # Process files in parallel using xargs
    printf '%s\0' "${files_to_test[@]}" | \
    xargs -0 -n 1 -P "$MAX_CORES" -I {} bash -c '
        file="$1"
        result="PASS:PASS"
        
        # Run markdownlint
        if ! run_markdownlint "$file"; then
            result="FAIL:${result#*:}"
        fi
        
        # Run remark-validate-links  
        if ! run_remark_validate_links "$file"; then
            result="${result%:*}:FAIL"
        fi
        
        echo "$result"
    ' -- {} >> "$results_file"
    
    # Count results
    while IFS=: read -r syntax_result links_result; do
        if [[ "$syntax_result" == "PASS" ]]; then
            ((syntax_passed++)) || true
        else
            ((syntax_failed++)) || true
        fi
        if [[ "$links_result" == "PASS" ]]; then
            ((links_passed++)) || true
        else
            ((links_failed++)) || true
        fi
    done < "$results_file"
    
    rm -f "$results_file"
    fi  # End of if num_to_test > 0
    
    # Calculate total time
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Summary
    log_section "Documentation Validation Summary"
    log_info "📊 Results:"
    log_info "  Syntax Check: $syntax_passed/$total_files passed"
    if [[ $syntax_failed -gt 0 ]]; then
        log_warning "  Syntax Issues: $syntax_failed files"
    fi
    log_info "  Link Check: $links_passed/$total_files passed"
    if [[ $links_failed -gt 0 ]]; then
        log_warning "  Link Issues: $links_failed files"
    fi
    log_info "  Total Time: ${duration}s"
    
    # Save cache before returning
    save_cache
    
    # Return failure if any tests failed
    if [[ $syntax_failed -gt 0 ]] || [[ $links_failed -gt 0 ]]; then
        return 1
    fi
    
    log_success "All documentation validation passed!"
    return 0
}

# Run main function
main "$@"