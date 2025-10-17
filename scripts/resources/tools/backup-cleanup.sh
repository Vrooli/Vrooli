#!/bin/bash
# ====================================================================
# Backup File Cleanup Automation
# ====================================================================
#
# Automatically manages backup file proliferation in the Vrooli 
# resource testing ecosystem. Provides safe cleanup of old backup
# files while preserving recent ones.
#
# Usage:
#   ./backup-cleanup.sh --help
#   ./backup-cleanup.sh --dry-run
#   ./backup-cleanup.sh --age 7 --confirm yes
#
# ====================================================================

set -euo pipefail

# Configuration
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/scripts/resources/tools"
RESOURCES_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_AGE_DAYS=7
BACKUP_LOG="$SCRIPT_DIR/backup-cleanup.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "$(date '+%Y-%m-%d %H:%M:%S') - $1" | tee -a "$BACKUP_LOG"
}

# Usage information
usage() {
    cat << EOF
Backup File Cleanup Automation

USAGE:
    $(basename "$0") [OPTIONS]

OPTIONS:
    --age DAYS          Remove backup files older than DAYS (default: $DEFAULT_AGE_DAYS)
    --pattern PATTERN   Only clean files matching pattern (default: all backup files)
    --dry-run           Show what would be cleaned without removing files
    --confirm yes       Skip interactive confirmation
    --stats             Show current backup file statistics
    --help              Show this help message

EXAMPLES:
    # Safe dry-run to see what would be cleaned
    $(basename "$0") --dry-run

    # Clean backup files older than 3 days
    $(basename "$0") --age 3 --confirm yes

    # Clean only .backup.auto files older than 7 days
    $(basename "$0") --pattern "*.backup.auto" --age 7

    # Show current backup file statistics
    $(basename "$0") --stats

BACKUP FILE PATTERNS:
    *.backup.auto      - Automatically generated backups
    *.backup_timestamp - Timestamped backups  
    *.backup.manual    - Manual backups
    *.backup           - Simple backups

SAFETY FEATURES:
    - Interactive confirmation by default
    - Dry-run mode for safe testing
    - Detailed logging of all operations
    - Preservation of recent files
    - Statistics and reporting

LOG FILE: $BACKUP_LOG
EOF
}

# Show backup file statistics
show_stats() {
    cd "$RESOURCES_DIR"
    
    echo -e "${BLUE}=== Backup File Statistics ===${NC}"
    
    local total_files
    total_files=$(find . -name "*.backup*" -type f 2>/dev/null | wc -l)
    echo "Total backup files: $total_files"
    
    if [[ $total_files -eq 0 ]]; then
        echo "No backup files found."
        return 0
    fi
    
    echo -e "\n${YELLOW}By Pattern:${NC}"
    find . -name "*.backup*" -type f 2>/dev/null | sed 's/.*\(\.backup[^/]*\)$/\1/' | sort | uniq -c | sort -nr | while read -r count pattern; do
        echo "  $count files: $pattern"
    done
    
    echo -e "\n${YELLOW}By Age:${NC}"
    local recent=$(find . -name "*.backup*" -type f -mtime -1 2>/dev/null | wc -l)
    local week_old=$(find . -name "*.backup*" -type f -mtime +1 -mtime -7 2>/dev/null | wc -l) 
    local old=$(find . -name "*.backup*" -type f -mtime +7 2>/dev/null | wc -l)
    
    echo "  $recent files: < 1 day old"
    echo "  $week_old files: 1-7 days old"
    echo "  $old files: > 7 days old"
    
    echo -e "\n${YELLOW}By Directory (top 10):${NC}"
    find . -name "*.backup*" -type f 2>/dev/null | sed 's|/[^/]*$||' | sort | uniq -c | sort -nr | head -10 | while read -r count dir; do
        echo "  $count files: $dir"
    done
    
    echo -e "\n${YELLOW}Disk Usage:${NC}"
    local total_size
    total_size=$(find . -name "*.backup*" -type f -exec du -ch {} + 2>/dev/null | tail -1 | cut -f1)
    echo "  Total size: $total_size"
}

# Find backup files to clean
find_backup_files() {
    local age_days=$1
    local pattern=${2:-"*.backup*"}
    
    cd "$RESOURCES_DIR"
    
    # Find files matching pattern and age criteria
    find . -name "$pattern" -type f -mtime +$age_days 2>/dev/null | sort
}

# Perform cleanup
cleanup_files() {
    local age_days=$1
    local pattern=${2:-"*.backup*"} 
    local dry_run=${3:-false}
    local confirm=${4:-false}
    
    local files_to_clean
    files_to_clean=$(find_backup_files "$age_days" "$pattern")
    
    if [[ -z "$files_to_clean" ]]; then
        log "${GREEN}No backup files found older than $age_days days matching pattern: $pattern${NC}"
        return 0
    fi
    
    local file_count
    file_count=$(echo "$files_to_clean" | wc -l)
    
    if [[ "$dry_run" == "true" ]]; then
        echo -e "${YELLOW}=== DRY RUN: Files that would be cleaned ===${NC}"
        echo "$files_to_clean"
        echo -e "\n${YELLOW}Total files: $file_count${NC}"
        log "DRY RUN: Would clean $file_count files older than $age_days days"
        return 0
    fi
    
    # Confirmation
    if [[ "$confirm" != "true" ]]; then
        echo -e "${YELLOW}Found $file_count backup files to clean (older than $age_days days):${NC}"
        echo "$files_to_clean" | head -10
        if [[ $file_count -gt 10 ]]; then
            echo "... and $((file_count - 10)) more files"
        fi
        
        echo
        read -p "Continue with cleanup? [y/N]: " -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            log "Cleanup cancelled by user"
            return 0
        fi
    fi
    
    # Perform cleanup
    echo -e "${BLUE}Cleaning up backup files...${NC}"
    local cleaned_count=0
    local failed_count=0
    
    while IFS= read -r file; do
        if [[ -f "$file" ]]; then
            if rm "$file" 2>/dev/null; then
                log "Removed: $file"
                ((cleaned_count++))
            else
                log "Failed to remove: $file"
                ((failed_count++))
            fi
        fi
    done <<< "$files_to_clean"
    
    echo -e "${GREEN}Cleanup completed:${NC}"
    echo "  Files removed: $cleaned_count"
    if [[ $failed_count -gt 0 ]]; then
        echo -e "  ${RED}Failed to remove: $failed_count${NC}"
    fi
    
    log "Cleanup completed: $cleaned_count files removed, $failed_count failed"
}

# Create automated cleanup cron job
install_automation() {
    local cron_age=${1:-7}
    local cron_schedule="0 2 * * 0"  # Weekly at 2 AM on Sunday
    
    local cron_command="$SCRIPT_DIR/$(basename "$0") --age $cron_age --confirm yes"
    
    # Check if already installed
    if crontab -l 2>/dev/null | grep -q "backup-cleanup.sh"; then
        echo -e "${YELLOW}Automated cleanup already installed in crontab${NC}"
        return 0
    fi
    
    echo -e "${BLUE}Installing automated weekly cleanup...${NC}"
    
    # Add to crontab
    (crontab -l 2>/dev/null; echo "$cron_schedule $cron_command") | crontab -
    
    echo -e "${GREEN}Automated cleanup installed:${NC}"
    echo "  Schedule: Weekly on Sunday at 2 AM"
    echo "  Age threshold: $cron_age days"
    echo "  Command: $cron_command"
    
    log "Automated cleanup installed: $cron_schedule $cron_command"
}

# Main function
main() {
    local age_days=$DEFAULT_AGE_DAYS
    local pattern="*.backup*"
    local dry_run=false
    local confirm=false
    local show_stats_only=false
    local install_cron=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --age)
                age_days="$2"
                shift 2
                ;;
            --pattern)
                pattern="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --confirm)
                if [[ "$2" == "yes" ]]; then
                    confirm=true
                fi
                shift 2
                ;;
            --stats)
                show_stats_only=true
                shift
                ;;
            --install-automation)
                install_cron=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Validate age parameter
    if ! [[ "$age_days" =~ ^[0-9]+$ ]] || [[ $age_days -lt 0 ]]; then
        echo "Error: Age must be a positive integer"
        exit 1
    fi
    
    # Create log file if it doesn't exist
    touch "$BACKUP_LOG"
    
    # Show stats only
    if [[ "$show_stats_only" == "true" ]]; then
        show_stats
        exit 0
    fi
    
    # Install automation
    if [[ "$install_cron" == "true" ]]; then
        install_automation "$age_days"
        exit 0
    fi
    
    # Main cleanup operation
    log "Starting backup cleanup: age=$age_days days, pattern=$pattern, dry_run=$dry_run"
    
    echo -e "${BLUE}=== Vrooli Backup File Cleanup ===${NC}"
    echo "Working directory: $RESOURCES_DIR"
    echo "Age threshold: $age_days days"
    echo "Pattern: $pattern"
    
    if [[ "$dry_run" == "true" ]]; then
        echo -e "${YELLOW}DRY RUN MODE - No files will be removed${NC}"
    fi
    
    echo
    cleanup_files "$age_days" "$pattern" "$dry_run" "$confirm"
    
    log "Backup cleanup completed"
}

# Run main function
main "$@"