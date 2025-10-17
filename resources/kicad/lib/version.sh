#!/bin/bash
# KiCad Version Control Management Functions

# Get script directory and APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
KICAD_VERSION_LIB_DIR="${APP_ROOT}/resources/kicad/lib"

# Source common functions if not already sourced
if ! declare -f kicad::init_dirs &>/dev/null; then
    source "${KICAD_VERSION_LIB_DIR}/common.sh"
fi

# Initialize git repository for a KiCad project
kicad::git::init() {
    local project="${1:-}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad version init <project-name>"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path" ]]; then
        echo "Error: Project not found: $project"
        return 1
    fi
    
    cd "$project_path" || return 1
    
    # Initialize git if not already initialized
    if [[ ! -d .git ]]; then
        git init
        echo "Git repository initialized for project: $project"
    else
        echo "Git repository already exists for project: $project"
    fi
    
    # Create or update .gitignore for KiCad
    cat > .gitignore <<'EOF'
# KiCad temporary files
*.000
*.bak
*.bck
*.kicad_pcb-bak
*.kicad_sch-bak
*-backups
*.kicad_prl
*.sch-bak
*~
_autosave-*
*.tmp
*-save.pro
*-save.kicad_pcb
fp-info-cache

# KiCad build outputs
*.dsn
*.ses

# Exported files
*.zip
*.tar.gz

# OS files
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo
EOF
    
    echo "Created KiCad-specific .gitignore"
    
    # Initial commit if no commits exist
    if ! git log -1 &>/dev/null; then
        git add .
        git commit -m "Initial KiCad project commit"
        echo "Created initial commit"
    fi
    
    return 0
}

# Show git status for KiCad project
kicad::git::status() {
    local project="${1:-}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad version status <project-name>"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path/.git" ]]; then
        echo "Error: Project is not under version control: $project"
        echo "Run 'resource-kicad version init $project' to initialize"
        return 1
    fi
    
    cd "$project_path" || return 1
    echo "Git status for project: $project"
    echo "================================"
    git status
    
    return 0
}

# Commit changes with automatic message
kicad::git::commit() {
    local project="${1:-}"
    local message="${2:-Auto-commit KiCad changes}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad version commit <project-name> [message]"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path/.git" ]]; then
        echo "Error: Project is not under version control: $project"
        return 1
    fi
    
    cd "$project_path" || return 1
    
    # Check for changes
    if git diff-index --quiet HEAD -- 2>/dev/null; then
        echo "No changes to commit in project: $project"
        return 0
    fi
    
    # Stage all changes
    git add -A
    
    # Commit with message
    git commit -m "$message"
    echo "Changes committed for project: $project"
    
    return 0
}

# Show commit history
kicad::git::log() {
    local project="${1:-}"
    local limit="${2:-10}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad version log <project-name> [limit]"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path/.git" ]]; then
        echo "Error: Project is not under version control: $project"
        return 1
    fi
    
    cd "$project_path" || return 1
    echo "Git history for project: $project (last $limit commits)"
    echo "================================"
    git log --oneline -n "$limit"
    
    return 0
}

# Create a backup branch
kicad::git::backup() {
    local project="${1:-}"
    local branch_name="${2:-backup-$(date +%Y%m%d-%H%M%S)}"
    
    if [[ -z "$project" ]]; then
        echo "Usage: resource-kicad version backup <project-name> [branch-name]"
        return 1
    fi
    
    kicad::init_dirs
    local project_path="${KICAD_PROJECTS_DIR}/${project}"
    
    if [[ ! -d "$project_path/.git" ]]; then
        echo "Error: Project is not under version control: $project"
        return 1
    fi
    
    cd "$project_path" || return 1
    
    # Create backup branch
    git checkout -b "$branch_name"
    echo "Created backup branch: $branch_name"
    
    # Return to main/master branch
    git checkout - 2>/dev/null || git checkout main 2>/dev/null || git checkout master
    echo "Returned to main branch"
    
    return 0
}

# Export functions for CLI framework
export -f kicad::git::init
export -f kicad::git::status  
export -f kicad::git::commit
export -f kicad::git::log
export -f kicad::git::backup