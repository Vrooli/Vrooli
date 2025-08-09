#!/usr/bin/env bash
set -euo pipefail

# Pre-commit Hook: Port Abstraction Check
# Prevents committing hardcoded port references that should use service references
# Add to .git/hooks/pre-commit or use in CI/CD pipeline

SCENARIO_TOOLS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SCENARIO_TOOLS_DIR}/../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"

#######################################
# Configuration
#######################################

# Ports that should be abstracted (critical ones only for pre-commit)
CRITICAL_PORTS=(
    "11434"  # ollama - most common
    "5678"   # n8n - most common
    "5681"   # windmill - most common
    "9200"   # searxng - most common
    "6333"   # qdrant - most common
    "9000"   # minio - most common
)

# Files to check (focus on scenario files)
FILE_PATTERNS=(
    "scripts/scenarios/**/*.json"
    "scripts/scenarios/**/*.yaml" 
    "scripts/scenarios/**/*.yml"
    "scripts/scenarios/**/*.ts"
)

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

#######################################
# Helper functions
#######################################

# Print colored output
pre_commit_port_check::print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

pre_commit_port_check::print_warning() {
    echo -e "${YELLOW}WARNING: $1${NC}" >&2
}

pre_commit_port_check::print_info() {
    echo -e "${BLUE}INFO: $1${NC}"
}

pre_commit_port_check::print_success() {
    echo -e "${GREEN}SUCCESS: $1${NC}"
}

# Get service suggestion for port
pre_commit_port_check::get_service_for_port() {
    local port="$1"
    case "$port" in
        "11434") echo "ollama" ;;
        "5678")  echo "n8n" ;;
        "5681")  echo "windmill" ;;
        "9200")  echo "searxng" ;;
        "6333")  echo "qdrant" ;;
        "9000")  echo "minio" ;;
        *)       echo "unknown" ;;
    esac
}

# Check staged files for hardcoded ports
pre_commit_port_check::check_staged_files() {
    local violations_found=false
    local total_violations=0
    
    pre_commit_port_check::print_info "🔍 Checking staged files for hardcoded port references..."
    
    # Get list of staged files
    local staged_files
    if ! staged_files=$(git diff --cached --name-only --diff-filter=ACM); then
        pre_commit_port_check::print_error "Failed to get staged files"
        return 1
    fi
    
    # Check each staged file
    while IFS= read -r file; do
        [[ -z "$file" ]] && continue
        [[ ! -f "$file" ]] && continue
        
        # Check if file matches patterns we care about
        local should_check=false
        for pattern in "${FILE_PATTERNS[@]}"; do
            if [[ "$file" == $pattern ]]; then
                should_check=true
                break
            fi
        done
        
        [[ "$should_check" == "false" ]] && continue
        
        # Check for hardcoded ports in this file
        local file_violations=0
        for port in "${CRITICAL_PORTS[@]}"; do
            local matches
            mapfile -t matches < <(git diff --cached "$file" | grep "^+" | grep -o "localhost:$port" 2>/dev/null || true)
            
            if [[ "${#matches[@]}" -gt 0 ]] && [[ -n "${matches[0]}" ]]; then
                if [[ "$file_violations" -eq 0 ]]; then
                    pre_commit_port_check::print_error "❌ $file contains hardcoded ports:"
                    violations_found=true
                fi
                
                local service
                service=$(pre_commit_port_check::get_service_for_port "$port")
                pre_commit_port_check::print_error "   localhost:$port should be \${service.$service.url}"
                
                ((file_violations++))
                ((total_violations++))
            fi
        done
        
    done <<< "$staged_files"
    
    if [[ "$violations_found" == "true" ]]; then
        echo ""
        pre_commit_port_check::print_error "🚫 COMMIT BLOCKED: Found $total_violations hardcoded port references"
        echo ""
        pre_commit_port_check::print_info "ℹ️  To fix these issues:"
        pre_commit_port_check::print_info "   1. Replace hardcoded localhost:PORT with \${service.NAME.url}"
        pre_commit_port_check::print_info "   2. Use the migration tool: scripts/scenarios/tools/migrate-ports-simple.sh"
        pre_commit_port_check::print_info "   3. Stage your changes and try committing again"
        echo ""
        pre_commit_port_check::print_warning "⚠️  If you need to commit anyway (not recommended):"
        pre_commit_port_check::print_warning "   Use: git commit --no-verify -m \"your message\""
        return 1
    else
        pre_commit_port_check::print_success "✅ No hardcoded ports found in staged files"
        return 0
    fi
}

# Check if we're in a git repository
pre_commit_port_check::check_git_repo() {
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        pre_commit_port_check::print_error "Not in a git repository"
        return 1
    fi
    return 0
}

# Show help information
pre_commit_port_check::show_help() {
    cat << 'EOF'
Pre-commit Port Abstraction Check

Prevents committing hardcoded port references that should use service references.

USAGE:
    # As pre-commit hook (automatic)
    git commit -m "your message"
    
    # Manual validation
    pre-commit-port-check.sh [options]

OPTIONS:
    --help      Show this help message
    --install   Install as git pre-commit hook
    --check     Run check on staged files (default)

INSTALLATION:
    # Copy to git hooks directory
    cp scripts/scenarios/tools/pre-commit-port-check.sh .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit

WHAT IT CHECKS:
    - Scenario JSON files (*.json in scripts/scenarios/)
    - YAML configuration files (*.yaml, *.yml)
    - TypeScript automation scripts (*.ts)
    
PORTS THAT TRIGGER WARNINGS:
    localhost:11434 (ollama) → ${service.ollama.url}
    localhost:5678  (n8n)    → ${service.n8n.url}
    localhost:5681  (windmill) → ${service.windmill.url}
    localhost:9200  (searxng) → ${service.searxng.url}
    localhost:6333  (qdrant) → ${service.qdrant.url}
    localhost:9000  (minio)  → ${service.minio.url}

EOF
}

# Install as git pre-commit hook
pre_commit_port_check::install_hook() {
    local hook_path=".git/hooks/pre-commit"
    
    if [[ ! -d ".git" ]]; then
        pre_commit_port_check::print_error "Not in a git repository root"
        return 1
    fi
    
    if [[ -f "$hook_path" ]]; then
        pre_commit_port_check::print_warning "Pre-commit hook already exists at $hook_path"
        pre_commit_port_check::print_info "Backup will be created as ${hook_path}.backup"
        cp "$hook_path" "${hook_path}.backup"
    fi
    
    # Create hook that calls this script
    cat > "$hook_path" << 'EOF'
#!/usr/bin/env bash
# Auto-generated pre-commit hook for port abstraction checking

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECKER="$PROJECT_ROOT/scripts/scenarios/tools/pre-commit-port-check.sh"

if [[ -x "$CHECKER" ]]; then
    exec "$CHECKER" --check
else
    echo "WARNING: Port abstraction checker not found at $CHECKER" >&2
    exit 0
fi
EOF
    
    chmod +x "$hook_path"
    pre_commit_port_check::print_success "✅ Pre-commit hook installed at $hook_path"
    pre_commit_port_check::print_info "ℹ️  The hook will run automatically on each commit"
}

#######################################
# Main script
#######################################

main() {
    local action="check"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --help|-h)
                pre_commit_port_check::show_help
                exit 0
                ;;
            --install)
                action="install"
                shift
                ;;
            --check)
                action="check"
                shift
                ;;
            *)
                pre_commit_port_check::print_error "Unknown option: $1"
                pre_commit_port_check::show_help
                exit 1
                ;;
        esac
    done
    
    # Check if we're in a git repo
    if ! pre_commit_port_check::check_git_repo; then
        exit 1
    fi
    
    # Execute requested action
    case "$action" in
        "install")
            pre_commit_port_check::install_hook
            ;;
        "check")
            pre_commit_port_check::check_staged_files
            ;;
        *)
            pre_commit_port_check::print_error "Unknown action: $action"
            exit 1
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi