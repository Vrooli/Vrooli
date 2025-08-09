#!/usr/bin/env bash
# validate-subroutines.sh - Validates subroutine references in routine JSON files
# This script checks that all subroutineId references are valid IDs from available routines

set -e

# Colors for output (only if terminal supports it)
if [[ -t 1 ]] && [[ "${TERM:-}" != "dumb" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Default paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${SCRIPT_DIR}/../../../lib/utils/var.sh"

PROJECT_ROOT="${var_ROOT_DIR}"
ROUTINE_REFERENCE="$PROJECT_ROOT/docs/ai-creation/routine/routine-reference.json"
STAGED_DIR="$PROJECT_ROOT/docs/ai-creation/routine/staged"

# Usage function
usage() {
    echo "Usage: $0 [OPTIONS] [routine.json ...]"
    echo ""
    echo "Validates subroutine references in routine JSON files against available routines."
    echo ""
    echo "OPTIONS:"
    echo "  -r, --reference FILE    Path to routine reference JSON (default: docs/ai-creation/routine/routine-reference.json)"
    echo "  -d, --directory DIR     Validate all JSON files in directory (default: docs/ai-creation/routine/staged/)"
    echo "  -v, --verbose          Show detailed output including valid subroutines"
    echo "  -q, --quiet            Only show errors and summary"
    echo "  --todo-only            Only check for TODO placeholders (faster)"
    echo "  --list-available       List all available subroutines and exit"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "EXAMPLES:"
    echo "  $0                                    # Validate all files in staged/"
    echo "  $0 docs/ai-creation/routine/staged/my-routine.json  # Validate specific file"
    echo "  $0 -d ./custom-dir/ --verbose        # Validate custom directory with details"
    echo "  $0 --todo-only                       # Quick check for TODO placeholders"
    echo "  $0 --list-available                  # Show all available subroutines"
    echo ""
}

# Helper function to safely check if command succeeds
safe_check() {
    local cmd="$1"
    local file="$2"
    if eval "$cmd" "$file" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Function to extract routine IDs from reference file
get_available_routine_ids() {
    local reference_file="$1"
    
    if [[ ! -f "$reference_file" ]]; then
        echo "Error: Routine reference file not found: $reference_file" >&2
        echo "Run './scripts/ai-creation/routine-reference-generator.sh' to create it." >&2
        return 1
    fi
    
    if ! jq -e '.routines' "$reference_file" >/dev/null 2>&1; then
        echo "Error: Invalid routine reference file format" >&2
        return 1
    fi
    
    jq -r '.routines[].id' "$reference_file" 2>/dev/null || return 1
}

# Function to list available subroutines with details
list_available_subroutines() {
    local reference_file="$1"
    
    echo -e "${BLUE}Available Subroutines:${NC}"
    echo "======================================================"
    
    # Get total count
    local total_count
    total_count=$(jq '.metadata.totalRoutines // 0' "$reference_file" 2>/dev/null || echo "0")
    echo -e "${YELLOW}Total Available Routines: $total_count${NC}"
    echo ""
    
    # Group by type
    echo -e "${BLUE}By Type:${NC}"
    jq -r '.byType | to_entries[] | "  \(.key): \(.value | length) routines"' "$reference_file" 2>/dev/null || echo "  Error reading types"
    echo ""
    
    # Show first 10 routines as examples
    echo -e "${BLUE}Example Routines (first 10):${NC}"
    jq -r '.routines[0:10][] | "  \(.id) - \(.name) (\(.type))"' "$reference_file" 2>/dev/null || echo "  Error reading routines"
    
    # Show search commands
    echo ""
    echo -e "${BLUE}Search Commands:${NC}"
    echo "  # Find by ID:"
    echo "  jq '.routines[] | select(.id == \"YOUR_ID\")' $reference_file"
    echo ""
    echo "  # Find by name pattern:"
    echo "  jq '.routines[] | select(.name | test(\"PATTERN\"; \"i\"))' $reference_file"
    echo ""
    echo "  # List all IDs:"
    echo "  jq -r '.routines[].id' $reference_file"
}

# Function to extract subroutine IDs from a routine JSON file
extract_subroutine_ids() {
    local json_file="$1"
    local verbose="$2"
    
    # Check if file exists and is valid JSON
    if [[ ! -f "$json_file" ]]; then
        echo "Error: File not found: $json_file" >&2
        return 1
    fi
    
    if ! safe_check "jq empty" "$json_file"; then
        echo "Error: Invalid JSON in $json_file" >&2
        return 1
    fi
    
    # Extract subroutineId values from graph steps
    local subroutine_ids=()
    
    # For multi-step routines with graph.schema.steps
    if safe_check "jq -e '.versions[0].config.graph.schema.steps'" "$json_file"; then
        mapfile -t step_ids < <(jq -r '.versions[0].config.graph.schema.steps[]?.subroutineId // empty' "$json_file" 2>/dev/null || true)
        subroutine_ids+=("${step_ids[@]}")
    fi
    
    # For other graph structures (future-proofing)
    if safe_check "jq -e '.versions[0].config.graph.nodes'" "$json_file"; then
        mapfile -t node_ids < <(jq -r '.versions[0].config.graph.nodes[]?.subroutineId // empty' "$json_file" 2>/dev/null || true)
        subroutine_ids+=("${node_ids[@]}")
    fi
    
    # Remove empty entries and duplicates
    local unique_ids=()
    for id in "${subroutine_ids[@]}"; do
        if [[ -n "$id" ]] && [[ ! " ${unique_ids[*]} " =~ " ${id} " ]]; then
            unique_ids+=("$id")
        fi
    done
    
    if [[ "$verbose" == "true" ]] && [[ ${#unique_ids[@]} -gt 0 ]]; then
        echo "  Found subroutine references: ${unique_ids[*]}"
    fi
    
    printf "%s\n" "${unique_ids[@]}"
}

# Function to validate subroutine IDs
validate_subroutine_ids() {
    local json_file="$1"
    local available_ids="$2"
    local verbose="$3"
    local todo_only="$4"
    
    local errors=0
    local warnings=0
    
    if [[ "$verbose" == "true" ]]; then
        echo -e "${BLUE}🔍 Validating subroutines in: $(basename "$json_file")${NC}"
    fi
    
    # Extract subroutine IDs from the file
    local subroutine_ids
    mapfile -t subroutine_ids < <(extract_subroutine_ids "$json_file" "$verbose")
    
    if [[ ${#subroutine_ids[@]} -eq 0 ]]; then
        if [[ "$verbose" == "true" ]]; then
            echo -e "  ${YELLOW}⚠️  No subroutine references found${NC}"
        fi
        return 0
    fi
    
    # Convert available IDs to array for faster lookup
    local -A available_id_map
    while IFS= read -r id; do
        if [[ -n "$id" ]]; then
            available_id_map["$id"]=1
        fi
    done <<< "$available_ids"
    
    # Validate each subroutine ID
    for subroutine_id in "${subroutine_ids[@]}"; do
        if [[ -z "$subroutine_id" ]]; then
            continue
        fi
        
        # Check for TODO placeholders
        if [[ "$subroutine_id" =~ ^TODO: ]]; then
            echo -e "  ${YELLOW}❌ TODO placeholder found: $subroutine_id${NC}"
            ((warnings++))
            continue
        fi
        
        # If only checking TODOs, skip further validation
        if [[ "$todo_only" == "true" ]]; then
            continue
        fi
        
        # Check ID format (19-digit snowflake)
        if [[ ! "$subroutine_id" =~ ^[0-9]{19}$ ]]; then
            echo -e "  ${RED}❌ Invalid ID format: $subroutine_id (should be 19-digit numeric)${NC}"
            ((errors++))
            continue
        fi
        
        # Check if ID exists in available routines
        if [[ -z "${available_id_map[$subroutine_id]:-}" ]]; then
            echo -e "  ${RED}❌ Subroutine ID not found: $subroutine_id${NC}"
            ((errors++))
        else
            if [[ "$verbose" == "true" ]]; then
                # Get routine name for verbose output
                local routine_name
                routine_name=$(echo "$available_ids" | while read -r line; do echo "$line"; done | head -1)
                # This is a simplified approach - in practice, we'd need the full reference data
                echo -e "  ${GREEN}✅ Valid subroutine ID: $subroutine_id${NC}"
            fi
        fi
    done
    
    return $((errors + warnings))
}

# Function to validate routine file
validate_routine_file() {
    local json_file="$1"
    local available_ids="$2"
    local verbose="$3"
    local quiet="$4"
    local todo_only="$5"
    
    if [[ "$quiet" != "true" ]]; then
        echo ""
        if [[ "$todo_only" == "true" ]]; then
            echo -e "${BLUE}🔍 Checking TODOs in: $(basename "$json_file")${NC}"
        else
            echo -e "${BLUE}🔍 Validating subroutines in: $(basename "$json_file")${NC}"
        fi
    fi
    
    # Validate subroutine IDs
    local validation_errors=0
    if ! validate_subroutine_ids "$json_file" "$available_ids" "$verbose" "$todo_only"; then
        validation_errors=$?
    fi
    
    if [[ $validation_errors -eq 0 ]] && [[ "$quiet" != "true" ]]; then
        if [[ "$todo_only" == "true" ]]; then
            echo -e "  ${GREEN}✅ No TODO placeholders found${NC}"
        else
            echo -e "  ${GREEN}✅ All subroutine references are valid${NC}"
        fi
    fi
    
    return $validation_errors
}

# Parse command line arguments
VERBOSE=false
QUIET=false
TODO_ONLY=false
LIST_AVAILABLE=false
VALIDATE_FILES=()
VALIDATE_DIRECTORY=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--reference)
            ROUTINE_REFERENCE="$2"
            shift 2
            ;;
        -d|--directory)
            VALIDATE_DIRECTORY="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -q|--quiet)
            QUIET=true
            shift
            ;;
        --todo-only)
            TODO_ONLY=true
            shift
            ;;
        --list-available)
            LIST_AVAILABLE=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        -*)
            echo "Error: Unknown option $1" >&2
            usage >&2
            exit 1
            ;;
        *)
            VALIDATE_FILES+=("$1")
            shift
            ;;
    esac
done

# Validate dependencies
if ! command -v jq >/dev/null 2>&1; then
    echo "Error: jq is required but not installed." >&2
    echo "Please install jq: https://stedolan.github.io/jq/download/" >&2
    exit 1
fi

# Handle --list-available option
if [[ "$LIST_AVAILABLE" == "true" ]]; then
    if [[ ! -f "$ROUTINE_REFERENCE" ]]; then
        echo "Error: Routine reference file not found: $ROUTINE_REFERENCE" >&2
        echo "Run './scripts/ai-creation/routine-reference-generator.sh' to create it." >&2
        exit 1
    fi
    list_available_subroutines "$ROUTINE_REFERENCE"
    exit 0
fi

# Get available routine IDs (unless only checking TODOs)
AVAILABLE_IDS=""
if [[ "$TODO_ONLY" != "true" ]]; then
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${BLUE}📚 Loading available routine IDs from: $ROUTINE_REFERENCE${NC}"
    fi
    
    AVAILABLE_IDS=$(get_available_routine_ids "$ROUTINE_REFERENCE")
    if [[ $? -ne 0 ]] || [[ -z "$AVAILABLE_IDS" ]]; then
        echo "Error: Could not load routine IDs from reference file" >&2
        exit 1
    fi
    
    local available_count
    available_count=$(echo "$AVAILABLE_IDS" | wc -l)
    if [[ "$QUIET" != "true" ]]; then
        echo -e "${GREEN}✅ Loaded $available_count available routine IDs${NC}"
    fi
fi

# Determine files to validate
FILES_TO_VALIDATE=()

if [[ ${#VALIDATE_FILES[@]} -gt 0 ]]; then
    # Validate specific files provided as arguments
    FILES_TO_VALIDATE=("${VALIDATE_FILES[@]}")
elif [[ -n "$VALIDATE_DIRECTORY" ]]; then
    # Validate all JSON files in specified directory
    if [[ ! -d "$VALIDATE_DIRECTORY" ]]; then
        echo "Error: Directory not found: $VALIDATE_DIRECTORY" >&2
        exit 1
    fi
    mapfile -t FILES_TO_VALIDATE < <(find "$VALIDATE_DIRECTORY" -name "*.json" -type f | sort)
else
    # Default: validate all JSON files in staged directory
    if [[ ! -d "$STAGED_DIR" ]]; then
        echo "Error: Default staged directory not found: $STAGED_DIR" >&2
        echo "Use -d option to specify a different directory or provide specific files." >&2
        exit 1
    fi
    mapfile -t FILES_TO_VALIDATE < <(find "$STAGED_DIR" -name "*.json" -type f | sort)
fi

if [[ ${#FILES_TO_VALIDATE[@]} -eq 0 ]]; then
    echo "No JSON files found to validate." >&2
    exit 1
fi

# Main validation loop
TOTAL_FILES=0
FAILED_FILES=0
TOTAL_ERRORS=0

if [[ "$QUIET" != "true" ]]; then
    echo ""
    echo -e "${BLUE}🚀 Starting validation of ${#FILES_TO_VALIDATE[@]} files...${NC}"
fi

for json_file in "${FILES_TO_VALIDATE[@]}"; do
    ((TOTAL_FILES++))
    
    # Validate the file
    if ! validate_routine_file "$json_file" "$AVAILABLE_IDS" "$VERBOSE" "$QUIET" "$TODO_ONLY"; then
        local file_errors=$?
        ((FAILED_FILES++))
        ((TOTAL_ERRORS += file_errors))
    fi
done

# Final summary
echo ""
echo -e "${BLUE}📊 Validation Summary:${NC}"
echo "======================================"
echo -e "  Total files checked: ${BLUE}$TOTAL_FILES${NC}"
echo -e "  Files passed: ${GREEN}$((TOTAL_FILES - FAILED_FILES))${NC}"
echo -e "  Files with issues: ${RED}$FAILED_FILES${NC}"
echo -e "  Total issues found: ${RED}$TOTAL_ERRORS${NC}"

if [[ $FAILED_FILES -gt 0 ]]; then
    echo ""
    echo -e "${YELLOW}💡 Tips to fix issues:${NC}"
    echo "  • Replace TODO placeholders with actual routine IDs"
    echo "  • Use 'jq -r \".routines[].id\" $ROUTINE_REFERENCE' to list available IDs"
    echo "  • Use '$0 --list-available' to browse available subroutines"
    echo "  • Search by functionality: jq '.routines[] | select(.name | test(\"PATTERN\"; \"i\"))' $ROUTINE_REFERENCE"
    exit 1
else
    echo ""
    echo -e "${GREEN}✅ All subroutine references are valid!${NC}"
    exit 0
fi