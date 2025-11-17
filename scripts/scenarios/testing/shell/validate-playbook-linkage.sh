#!/usr/bin/env bash
# Validate playbook-requirement linkage
# Ensures every playbook is properly linked to an existing requirement
set -euo pipefail

SCENARIO_DIR="${1:-$(pwd)}"
SCENARIO_NAME="$(basename "$SCENARIO_DIR")"

PLAYBOOKS_DIR="$SCENARIO_DIR/test/playbooks"
REQUIREMENTS_DIR="$SCENARIO_DIR/requirements"

# Exit codes
EXIT_SUCCESS=0
EXIT_ORPHANED_PLAYBOOKS=1
EXIT_MISSING_PLAYBOOK_FILES=2
EXIT_INVALID_REQUIREMENT_REFS=3
EXIT_NO_REQUIREMENTS=4
EXIT_FIXTURE_ERRORS=5

# Track issues
declare -a ORPHANED_PLAYBOOKS=()
declare -a MISSING_REQUIREMENT_IDS=()
declare -a MISSING_PLAYBOOK_FILES=()
declare -a PLAYBOOKS_WITHOUT_METADATA=()

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_header() {
    echo -e "${BLUE}$1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if directories exist
if [ ! -d "$PLAYBOOKS_DIR" ]; then
    log_error "Playbooks directory not found: $PLAYBOOKS_DIR"
    exit $EXIT_SUCCESS  # Not an error if no playbooks exist yet
fi

if [ ! -d "$REQUIREMENTS_DIR" ]; then
    log_error "Requirements directory not found: $REQUIREMENTS_DIR"
    exit $EXIT_NO_REQUIREMENTS
fi

# Check for required tools
if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed"
    exit 1
fi

log_header "🔍 Validating Playbook-Requirement Linkage for ${SCENARIO_NAME}"
echo "=============================================================="
echo ""

# Step 1: Collect all requirement IDs and their playbook references
log_header "📋 Step 1: Scanning requirements registry..."
echo ""

# Use temp files instead of arrays (more reliable for complex parsing)
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

REQUIREMENT_IDS_FILE="$TEMP_DIR/requirement_ids.txt"
REQUIREMENT_REFS_FILE="$TEMP_DIR/requirement_refs.txt"

touch "$REQUIREMENT_IDS_FILE"
touch "$REQUIREMENT_REFS_FILE"

if [ -f "$REQUIREMENTS_DIR/index.json" ]; then
    # Get all requirement files from index.json imports
    requirement_files=""
    while IFS= read -r import_path; do
        [ -n "$import_path" ] && requirement_files="$requirement_files"$'\n'"$REQUIREMENTS_DIR/$import_path"
    done < <(jq -r '.imports[]?' "$REQUIREMENTS_DIR/index.json" 2>/dev/null)

    # Also include index.json itself if it has requirements
    if jq -e '.requirements[]?' "$REQUIREMENTS_DIR/index.json" >/dev/null 2>&1; then
        requirement_files="$REQUIREMENTS_DIR/index.json"$'\n'"$requirement_files"
    fi
else
    # Fallback: scan all JSON files
    requirement_files=$(find "$REQUIREMENTS_DIR" -name "*.json" -type f)
fi

# Parse all requirement files
while IFS= read -r req_file; do
    [ -z "$req_file" ] || [ ! -f "$req_file" ] && continue

    # Extract requirement IDs
    jq -r '.requirements[]?.id // empty' "$req_file" 2>/dev/null >> "$REQUIREMENT_IDS_FILE"

    # Extract automation playbook references (format: ref|req_id)
    jq -r '.requirements[] | select(.validation) | .id as $req_id | .validation[] | select(.type == "automation") | "\(.ref // "")|\($req_id)"' "$req_file" 2>/dev/null >> "$REQUIREMENT_REFS_FILE"
done <<< "$requirement_files"

# Deduplicate requirement IDs
sort -u "$REQUIREMENT_IDS_FILE" -o "$REQUIREMENT_IDS_FILE"

requirement_count=$(wc -l < "$REQUIREMENT_IDS_FILE" | tr -d ' ')
reference_count=$(wc -l < "$REQUIREMENT_REFS_FILE" | tr -d ' ')

echo "   Found $requirement_count unique requirement IDs"
echo "   Found $reference_count playbook references in requirements"
echo ""

# Step 2: Check for missing playbook files (Case 1: Requirement → Missing Playbook)
log_header "📂 Step 2: Checking for missing playbook files..."
echo ""

MISSING_FILES_LIST="$TEMP_DIR/missing_files.txt"
touch "$MISSING_FILES_LIST"

while IFS='|' read -r ref req_id; do
    [ -z "$ref" ] && continue

    # Try multiple path resolutions
    playbook_file=""
    if [ -f "$SCENARIO_DIR/$ref" ]; then
        playbook_file="$SCENARIO_DIR/$ref"
    elif [ -f "$ref" ]; then
        playbook_file="$ref"
    fi

    if [ -z "$playbook_file" ]; then
        echo "$req_id|$ref" >> "$MISSING_FILES_LIST"
    fi
done < "$REQUIREMENT_REFS_FILE"

missing_file_count=$(wc -l < "$MISSING_FILES_LIST" | tr -d ' ')

if [ "$missing_file_count" -gt 0 ]; then
    log_error "Found $missing_file_count requirement(s) referencing missing playbook files:"
    echo ""
    while IFS='|' read -r req_id ref; do
        echo "   • Requirement: $req_id"
        echo "     Referenced file: $ref"
        echo "     → File does not exist"
        echo ""
    done < "$MISSING_FILES_LIST"
else
    log_success "All referenced playbook files exist"
    echo ""
fi

# Step 3: Scan all playbook files
log_header "📝 Step 3: Scanning playbook files..."
echo ""

playbook_files=$(find "$PLAYBOOKS_DIR" -name "*.json" -type f | grep -v "/__" || true)
playbook_count=$(echo "$playbook_files" | grep -c . || echo 0)

echo "   Found $playbook_count playbook file(s)"
echo ""

# Fixture tracking containers
declare -A FIXTURE_FILES=()
declare -A FIXTURE_USAGE=()
declare -a FIXTURE_METADATA_ERRORS=()
declare -a FIXTURE_DUPLICATE_ERRORS=()
declare -a FIXTURE_UNKNOWN_REFERENCES=()
declare -a FIXTURE_UNUSED=()

# Load fixture workflow definitions
fixtures_dir="$PLAYBOOKS_DIR/__subflows"
if [ -d "$fixtures_dir" ]; then
    while IFS= read -r fixture_file; do
        [ -z "$fixture_file" ] && continue
        rel_path="${fixture_file#$SCENARIO_DIR/}"
        fixture_id=$(jq -r '.metadata.fixture_id // empty' "$fixture_file" 2>/dev/null)
        if [ -z "$fixture_id" ] || [ "$fixture_id" = "null" ]; then
            FIXTURE_METADATA_ERRORS+=("$rel_path")
            continue
        fi
        if [ -n "${FIXTURE_FILES[$fixture_id]:-}" ]; then
            FIXTURE_DUPLICATE_ERRORS+=("$fixture_id|$rel_path|${FIXTURE_FILES[$fixture_id]}")
            continue
        fi
        FIXTURE_FILES[$fixture_id]="$rel_path"
    done < <(find "$fixtures_dir" -type f -name '*.json' 2>/dev/null | sort)
fi

# Step 4: Check each playbook
log_header "🔗 Step 4: Validating playbook metadata..."
echo ""

ORPHANED_LIST="$TEMP_DIR/orphaned.txt"
MISSING_REQ_LIST="$TEMP_DIR/missing_req.txt"
NO_METADATA_LIST="$TEMP_DIR/no_metadata.txt"

touch "$ORPHANED_LIST"
touch "$MISSING_REQ_LIST"
touch "$NO_METADATA_LIST"

while IFS= read -r playbook_file; do
    [ -z "$playbook_file" ] && continue

    rel_path="${playbook_file#$SCENARIO_DIR/}"

    # Extract requirement ID from playbook metadata
    if ! jq -e '.metadata' "$playbook_file" >/dev/null 2>&1; then
        # Case 3a: Playbook has no metadata section at all
        echo "$rel_path|NO_METADATA_SECTION" >> "$NO_METADATA_LIST"
        continue
    fi

    declared_req=$(jq -r '.metadata.requirement // empty' "$playbook_file" 2>/dev/null)

    if [ -z "$declared_req" ] || [ "$declared_req" = "null" ]; then
        # Case 3b: Playbook has metadata but no requirement field
        echo "$rel_path|NO_REQUIREMENT_FIELD" >> "$NO_METADATA_LIST"
        continue
    fi

    # Check if declared requirement exists
    if ! grep -Fxq "$declared_req" "$REQUIREMENT_IDS_FILE"; then
        # Case 2: Playbook references non-existent requirement
        echo "$rel_path|$declared_req" >> "$MISSING_REQ_LIST"
        continue
    fi

    # Check if this playbook is referenced by the requirement it declares
    is_referenced=false
    while IFS='|' read -r ref req_id; do
        if [ "$ref" = "$rel_path" ] && [ "$req_id" = "$declared_req" ]; then
            is_referenced=true
            break
        fi
    done < "$REQUIREMENT_REFS_FILE"

    if [ "$is_referenced" = false ]; then
        # Playbook declares a valid requirement, but that requirement doesn't reference it back
        echo "$rel_path|$declared_req" >> "$ORPHANED_LIST"
    fi

    if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
        placeholders=$(rg -o '@fixture/[A-Za-z0-9_.:-]+' -N "$playbook_file" 2>/dev/null | sort -u || true)
        if [ -n "$placeholders" ]; then
            while IFS= read -r placeholder; do
                [ -z "$placeholder" ] && continue
                slug="${placeholder#@fixture/}"
                if [ -z "${FIXTURE_FILES[$slug]:-}" ]; then
                    FIXTURE_UNKNOWN_REFERENCES+=("$rel_path|$slug")
                    continue
                fi
                FIXTURE_USAGE["$slug"]="used"
            done <<< "$placeholders"
        fi
    fi
done <<< "$playbook_files"

if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
    for slug in "${!FIXTURE_FILES[@]}"; do
        if [ -z "${FIXTURE_USAGE[$slug]:-}" ]; then
            FIXTURE_UNUSED+=("$slug|${FIXTURE_FILES[$slug]}")
        fi
    done
fi

# Report Results
echo ""
echo "=============================================================="
log_header "📊 Validation Results"
echo "=============================================================="
echo ""

fixture_metadata_count=${#FIXTURE_METADATA_ERRORS[@]}
fixture_duplicate_count=${#FIXTURE_DUPLICATE_ERRORS[@]}
fixture_unknown_count=${#FIXTURE_UNKNOWN_REFERENCES[@]}
fixture_unused_count=${#FIXTURE_UNUSED[@]}
fixture_issue_total=$((fixture_metadata_count + fixture_duplicate_count + fixture_unknown_count + fixture_unused_count))

if [ "$fixture_metadata_count" -gt 0 ]; then
    log_error "Fixture workflows missing metadata.fixture_id"
    echo ""
    for entry in "${FIXTURE_METADATA_ERRORS[@]}"; do
        echo -e "   ❌ Fixture: ${BLUE}${entry}${NC}"
    done
    echo ""
fi

if [ "$fixture_duplicate_count" -gt 0 ]; then
    log_error "Duplicate fixture identifiers detected"
    echo ""
    for entry in "${FIXTURE_DUPLICATE_ERRORS[@]}"; do
        IFS='|' read -r slug new_path existing_path <<< "$entry"
        echo -e "   ❌ Fixture ID: ${BLUE}${slug}${NC}"
        echo -e "      Files: ${RED}${new_path}${NC} and ${YELLOW}${existing_path}${NC}"
        echo ""
    done
fi

if [ "$fixture_unknown_count" -gt 0 ]; then
    log_error "Playbooks reference unknown fixtures"
    echo ""
    for entry in "${FIXTURE_UNKNOWN_REFERENCES[@]}"; do
        IFS='|' read -r playbook slug <<< "$entry"
        echo -e "   ❌ Playbook: ${BLUE}${playbook}${NC}"
        echo -e "      Unknown fixture: ${RED}${slug}${NC}"
        echo ""
    done
fi

if [ "$fixture_unused_count" -gt 0 ]; then
    log_error "Fixture workflows are not referenced by any playbook"
    echo ""
    for entry in "${FIXTURE_UNUSED[@]}"; do
        IFS='|' read -r slug file_path <<< "$entry"
        echo -e "   ⚠️  Fixture: ${BLUE}${slug}${NC} (${file_path})"
    done
    echo ""
fi

missing_file_count=$(wc -l < "$MISSING_FILES_LIST" | tr -d ' ')
missing_req_count=$(wc -l < "$MISSING_REQ_LIST" | tr -d ' ')
no_metadata_count=$(wc -l < "$NO_METADATA_LIST" | tr -d ' ')
orphaned_count=$(wc -l < "$ORPHANED_LIST" | tr -d ' ')

total_issues=$((missing_file_count + missing_req_count + no_metadata_count + orphaned_count + fixture_issue_total))

# Report Case 1: Requirements pointing to missing files
if [ "$missing_file_count" -gt 0 ]; then
    log_error "CASE 1: Requirements Reference Missing Playbook Files"
    echo ""
    echo "These requirements reference playbook files that don't exist on disk:"
    echo ""
    while IFS='|' read -r req_id ref; do
        echo -e "   ❌ Requirement: ${BLUE}${req_id}${NC}"
        echo -e "      File: ${RED}${ref}${NC}"
        echo ""
    done < "$MISSING_FILES_LIST"
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Create the missing playbook file(s), OR"
    echo "   • Remove the automation validation entry from the requirement"
    echo ""
fi

# Report Case 2: Playbooks referencing non-existent requirements
if [ "$missing_req_count" -gt 0 ]; then
    log_error "CASE 2: Playbooks Reference Non-Existent Requirements"
    echo ""
    echo "These playbooks declare requirement IDs that don't exist in requirements/:"
    echo ""
    while IFS='|' read -r playbook req_id; do
        echo -e "   ❌ Playbook: ${BLUE}${playbook}${NC}"
        echo -e "      Declares: ${RED}${req_id}${NC} (not found in requirements/)"
        echo ""
    done < "$MISSING_REQ_LIST"
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Create requirement entry in appropriate requirements/*.json file, OR"
    echo "   • Update playbook metadata.requirement to reference an existing requirement, OR"
    echo "   • Delete the playbook if it's no longer needed"
    echo ""
fi

# Report Case 3: Playbooks without requirement metadata
if [ "$no_metadata_count" -gt 0 ]; then
    log_error "CASE 3: Playbooks Missing Requirement Metadata"
    echo ""
    echo "These playbooks don't declare which requirement they validate:"
    echo ""
    while IFS='|' read -r playbook reason; do
        echo -e "   ❌ Playbook: ${BLUE}${playbook}${NC}"
        if [ "$reason" = "NO_METADATA_SECTION" ]; then
            echo -e "      Issue: Missing ${RED}.metadata${NC} section entirely"
        else
            echo -e "      Issue: Missing ${RED}.metadata.requirement${NC} field"
        fi
        echo ""
    done < "$NO_METADATA_LIST"
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Add metadata section with requirement field:"
    echo "     {\"metadata\": {\"requirement\": \"BAS-REQ-ID\", \"description\": \"...\", \"version\": 1}}"
    echo ""
fi

# Report orphaned playbooks (valid requirement exists but doesn't reference back)
if [ "$orphaned_count" -gt 0 ]; then
    log_warning "Playbooks Not Referenced by Their Declared Requirements"
    echo ""
    echo "These playbooks declare valid requirements, but those requirements don't reference them:"
    echo ""
    while IFS='|' read -r playbook req_id; do
        echo -e "   ⚠️  Playbook: ${BLUE}${playbook}${NC}"
        echo -e "      Declares: ${YELLOW}${req_id}${NC} (requirement exists but doesn't link back)"
        echo ""
    done < "$ORPHANED_LIST"
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Add automation validation entry to the requirement:"
    echo "     {\"type\": \"automation\", \"ref\": \"<playbook_path>\", \"phase\": \"integration\", \"status\": \"implemented\"}"
    echo ""
fi

# Summary
echo "=============================================================="
echo ""

if [ "$total_issues" -eq 0 ]; then
    log_success "All playbooks are properly linked to requirements!"
    echo ""
    echo "   📊 Summary:"
    echo "      • Total playbooks: $playbook_count"
    echo "      • Total requirements: $requirement_count"
    echo "      • Playbook references in requirements: $reference_count"
    echo "      • All have valid requirement metadata"
    echo "      • All requirements reference existing files"
    if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
        echo "      • Fixture workflows validated: ${#FIXTURE_FILES[@]}"
    fi
    echo ""
    exit $EXIT_SUCCESS
else
    log_error "Validation Failed: $total_issues issue(s) found"
    echo ""
    echo "   📊 Summary:"
    echo "      • Total playbooks: $playbook_count"
    echo "      • Total requirements: $requirement_count"
    echo "      • Missing playbook files: $missing_file_count"
    echo "      • Invalid requirement refs: $missing_req_count"
    echo "      • Missing metadata: $no_metadata_count"
    echo "      • Orphaned (not referenced): $orphaned_count"
    if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
        echo "      • Fixture metadata issues: $fixture_metadata_count"
        echo "      • Fixture duplicates: $fixture_duplicate_count"
        echo "      • Unknown fixture references: $fixture_unknown_count"
        echo "      • Unused fixtures: $fixture_unused_count"
    fi
    echo ""

    # Exit with most severe error code
    if [ "$missing_file_count" -gt 0 ]; then
        exit $EXIT_MISSING_PLAYBOOK_FILES
    elif [ "$missing_req_count" -gt 0 ]; then
        exit $EXIT_INVALID_REQUIREMENT_REFS
    elif [ "$no_metadata_count" -gt 0 ] || [ "$orphaned_count" -gt 0 ]; then
        exit $EXIT_ORPHANED_PLAYBOOKS
    elif [ $((fixture_metadata_count + fixture_duplicate_count + fixture_unknown_count + fixture_unused_count)) -gt 0 ]; then
        exit $EXIT_FIXTURE_ERRORS
    fi
fi
