#!/usr/bin/env bash
# Agent S2 Test Cleanup Script
# Removes test outputs and manages disk space

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_OUTPUT_DIR="${SCRIPT_DIR}/test-outputs"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MAX_AGE_DAYS="${AGENT_S2_TEST_MAX_AGE_DAYS:-7}"
MAX_SIZE_MB="${AGENT_S2_TEST_MAX_SIZE_MB:-100}"

echo -e "${GREEN}Agent S2 Test Cleanup${NC}"
echo "================================"

# Function to get directory size in MB
get_dir_size_mb() {
    local dir="$1"
    if [[ -d "$dir" ]]; then
        du -sm "$dir" 2>/dev/null | cut -f1
    else
        echo "0"
    fi
}

# Function to clean old files
clean_old_files() {
    local dir="$1"
    local age_days="$2"
    
    if [[ ! -d "$dir" ]]; then
        return
    fi
    
    echo -e "${YELLOW}Cleaning files older than ${age_days} days in ${dir}...${NC}"
    find "$dir" -type f -name "*" -mtime +${age_days} -not -name ".gitkeep" -delete 2>/dev/null || true
}

# Function to clean by size
clean_by_size() {
    local dir="$1"
    local max_size_mb="$2"
    
    local current_size=$(get_dir_size_mb "$dir")
    
    if [[ $current_size -gt $max_size_mb ]]; then
        echo -e "${YELLOW}Directory size (${current_size}MB) exceeds limit (${max_size_mb}MB)${NC}"
        echo "Removing oldest files..."
        
        # Remove oldest files until under size limit
        while [[ $(get_dir_size_mb "$dir") -gt $max_size_mb ]]; do
            oldest_file=$(find "$dir" -type f -not -name ".gitkeep" -printf '%T+ %p\n' 2>/dev/null | sort | head -n1 | cut -d' ' -f2-)
            if [[ -n "$oldest_file" ]]; then
                rm -f "$oldest_file"
                echo "  Removed: $(basename "$oldest_file")"
            else
                break
            fi
        done
    fi
}

# Parse command line arguments
FORCE_CLEAN=false
CLEAN_ALL=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --force|-f)
            FORCE_CLEAN=true
            shift
            ;;
        --all|-a)
            CLEAN_ALL=true
            shift
            ;;
        --max-age)
            MAX_AGE_DAYS="$2"
            shift 2
            ;;
        --max-size)
            MAX_SIZE_MB="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --force, -f      Force cleanup without confirmation"
            echo "  --all, -a        Remove all test outputs (except .gitkeep)"
            echo "  --max-age DAYS   Set maximum age for files (default: 7)"
            echo "  --max-size MB    Set maximum directory size in MB (default: 100)"
            echo "  --help, -h       Show this help message"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Show current status
echo "Current test output status:"
echo "  Total size: $(get_dir_size_mb "$TEST_OUTPUT_DIR")MB"
echo "  Screenshots: $(find "$TEST_OUTPUT_DIR/screenshots" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"
echo "  Logs: $(find "$TEST_OUTPUT_DIR/logs" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"
echo "  Reports: $(find "$TEST_OUTPUT_DIR/reports" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"
echo ""

# Confirm cleanup
if [[ "$FORCE_CLEAN" != "true" ]] && [[ "$CLEAN_ALL" != "true" ]]; then
    read -p "Clean test outputs older than ${MAX_AGE_DAYS} days? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cleanup cancelled."
        exit 0
    fi
fi

# Perform cleanup
if [[ "$CLEAN_ALL" == "true" ]]; then
    echo -e "${YELLOW}Removing all test outputs...${NC}"
    find "$TEST_OUTPUT_DIR" -type f -not -name ".gitkeep" -delete 2>/dev/null || true
else
    # Clean old files
    clean_old_files "$TEST_OUTPUT_DIR/screenshots" "$MAX_AGE_DAYS"
    clean_old_files "$TEST_OUTPUT_DIR/logs" "$MAX_AGE_DAYS"
    clean_old_files "$TEST_OUTPUT_DIR/reports" "$MAX_AGE_DAYS"
    
    # Clean by size
    clean_by_size "$TEST_OUTPUT_DIR" "$MAX_SIZE_MB"
fi

# Show final status
echo ""
echo -e "${GREEN}Cleanup complete!${NC}"
echo "Final test output status:"
echo "  Total size: $(get_dir_size_mb "$TEST_OUTPUT_DIR")MB"
echo "  Screenshots: $(find "$TEST_OUTPUT_DIR/screenshots" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"
echo "  Logs: $(find "$TEST_OUTPUT_DIR/logs" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"
echo "  Reports: $(find "$TEST_OUTPUT_DIR/reports" -type f -not -name ".gitkeep" 2>/dev/null | wc -l) files"