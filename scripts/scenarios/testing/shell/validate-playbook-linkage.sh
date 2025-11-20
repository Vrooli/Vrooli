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
declare -a REGISTRY_ORPHANED_PLAYBOOKS=()
declare -a REGISTRY_MISSING_FILES=()
declare -a REGISTRY_UNKNOWN_REQUIREMENTS=()
declare -a REGISTRY_MISMATCH_REQUIREMENTS=()
declare -a REQUIREMENT_REFERENCES_MISSING_REGISTRY=()
declare -A REQUIREMENT_FILE_MAP=()

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

_append_unique_line() {
    local current="$1"
    local value="$2"
    if [ -z "$value" ]; then
        printf '%s' "$current"
        return
    fi
    if [ -z "$current" ]; then
        printf '%s' "$value"
        return
    fi
    if grep -Fxq "$value" <<< "$current"; then
        printf '%s' "$current"
    else
        printf '%s\n%s' "$current" "$value"
    fi
}

_normalize_list() {
    local data="$1"
    if [ -z "$data" ]; then
        echo ""
        return
    fi
    printf '%s\n' "$data" | sed '/^$/d' | sort -u
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

    REQUIREMENT_FILE_MAP["$ref"]=$(_append_unique_line "${REQUIREMENT_FILE_MAP["$ref"]:-}" "$req_id")

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
declare -a FIXTURE_UNKNOWN_REQUIREMENTS=()
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

        fixture_requirements=""
        fixture_requirements=$(jq -r '.metadata.requirements[]? // empty' "$fixture_file" 2>/dev/null || true)
        if [ -n "$fixture_requirements" ]; then
            while IFS= read -r fixture_req; do
                [ -z "$fixture_req" ] && continue
                if ! grep -Fxq "$fixture_req" "$REQUIREMENT_IDS_FILE"; then
                    FIXTURE_UNKNOWN_REQUIREMENTS+=("$fixture_id|$fixture_req|$rel_path")
                fi
            done <<< "$fixture_requirements"
        fi
    done < <(find "$fixtures_dir" -type f -name '*.json' 2>/dev/null | sort)
fi

# Step 4: Cross-check registry ordering and requirement coverage
log_header "🔗 Step 4: Cross-checking registry order and requirement coverage..."
echo ""

REGISTRY_PATH="$PLAYBOOKS_DIR/registry.json"
if [ ! -f "$REGISTRY_PATH" ]; then
    log_error "Playbook registry not found at $REGISTRY_PATH"
    exit $EXIT_MISSING_PLAYBOOK_FILES
fi

declare -A REGISTRY_REQUIREMENTS_MAP=()
while IFS= read -r entry; do
    [ -z "$entry" ] && continue
    file=$(printf '%s' "$entry" | jq -r '.file // empty')
    [ -z "$file" ] && continue
    requirements=$(printf '%s' "$entry" | jq -r '.requirements[]?' 2>/dev/null || true)
    REGISTRY_REQUIREMENTS_MAP["$file"]="$requirements"
    if [ ! -f "$SCENARIO_DIR/$file" ]; then
        REGISTRY_MISSING_FILES+=("$file")
    fi
done < <(jq -c '.playbooks[]?' "$REGISTRY_PATH")

if [ ${#REGISTRY_REQUIREMENTS_MAP[@]} -eq 0 ]; then
    log_warning "Registry contains no playbooks; execution order cannot be determined"
fi

for ref in "${!REQUIREMENT_FILE_MAP[@]}"; do
    if [[ -z "${REGISTRY_REQUIREMENTS_MAP[$ref]+_}" ]]; then
        while IFS= read -r req_id; do
            [ -z "$req_id" ] && continue
            REQUIREMENT_REFERENCES_MISSING_REGISTRY+=("$req_id|$ref")
        done <<< "${REQUIREMENT_FILE_MAP[$ref]}"
        continue
    fi
    registry_list=$(_normalize_list "${REGISTRY_REQUIREMENTS_MAP[$ref]}")
    actual_list=$(_normalize_list "${REQUIREMENT_FILE_MAP[$ref]}")
    if [ -z "$actual_list" ]; then
        actual_list=""
    fi
    if [ -z "$registry_list" ] && [ -z "$actual_list" ]; then
        continue
    fi
    if [ -z "$registry_list" ] || [ -z "$actual_list" ] || [ "$registry_list" != "$actual_list" ]; then
        REGISTRY_MISMATCH_REQUIREMENTS+=("$ref|$registry_list|$actual_list")
    fi
done

for file in "${!REGISTRY_REQUIREMENTS_MAP[@]}"; do
    registry_list=$(_normalize_list "${REGISTRY_REQUIREMENTS_MAP[$file]}")
    if [ -z "$registry_list" ] && [ -z "${REQUIREMENT_FILE_MAP[$file]:-}" ]; then
        REGISTRY_ORPHANED_PLAYBOOKS+=("$file")
    fi
    while IFS= read -r req_id; do
        [ -z "$req_id" ] && continue
        if ! grep -Fxq "$req_id" "$REQUIREMENT_IDS_FILE"; then
            REGISTRY_UNKNOWN_REQUIREMENTS+=("$file|$req_id")
        fi
    done <<< "$registry_list"
done

if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
    while IFS= read -r playbook_file; do
        [ -z "$playbook_file" ] && continue
        rel_path="${playbook_file#$SCENARIO_DIR/}"
        if [ "$rel_path" = "test/playbooks/registry.json" ]; then
            continue
        fi
        placeholders=$(grep -oP '@fixture/[A-Za-z0-9_-]+' "$playbook_file" 2>/dev/null | sed 's|@fixture/||' | sort -u || true)
        if [ -n "$placeholders" ]; then
            while IFS= read -r placeholder; do
                [ -z "$placeholder" ] && continue
                slug="$placeholder"
                if [ -z "${FIXTURE_FILES[$slug]:-}" ]; then
                    FIXTURE_UNKNOWN_REFERENCES+=("$rel_path|$slug")
                    continue
                fi
                FIXTURE_USAGE["$slug"]="used"
            done <<< "$placeholders"
        fi
    done <<< "$playbook_files"
fi

if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
    for fixture_slug in "${!FIXTURE_FILES[@]}"; do
        fixture_path="$SCENARIO_DIR/${FIXTURE_FILES[$fixture_slug]}"
        placeholders=$(grep -oP '@fixture/[A-Za-z0-9_-]+' "$fixture_path" 2>/dev/null | sed 's|@fixture/||' | sort -u || true)
        if [ -n "$placeholders" ]; then
            while IFS= read -r placeholder; do
                [ -z "$placeholder" ] && continue
                slug="$placeholder"
                if [ -z "${FIXTURE_FILES[$slug]:-}" ]; then
                    FIXTURE_UNKNOWN_REFERENCES+=("${FIXTURE_FILES[$fixture_slug]}|$slug")
                    continue
                fi
                FIXTURE_USAGE["$slug"]="used"
            done <<< "$placeholders"
        fi
    done

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
fixture_unknown_req_count=${#FIXTURE_UNKNOWN_REQUIREMENTS[@]}
fixture_unused_count=${#FIXTURE_UNUSED[@]}
fixture_issue_total=$((fixture_metadata_count + fixture_duplicate_count + fixture_unknown_count + fixture_unknown_req_count + fixture_unused_count))

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

if [ "$fixture_unknown_req_count" -gt 0 ]; then
    log_error "Fixture metadata references unknown requirement IDs"
    echo ""
    for entry in "${FIXTURE_UNKNOWN_REQUIREMENTS[@]}"; do
        IFS='|' read -r slug req_id file_path <<< "$entry"
        echo -e "   ❌ Fixture: ${BLUE}${slug}${NC} (${file_path})"
        echo -e "      Requirement: ${RED}${req_id}${NC} (not found in requirements/)"
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
registry_missing_file_count=${#REGISTRY_MISSING_FILES[@]}
missing_registry_count=${#REQUIREMENT_REFERENCES_MISSING_REGISTRY[@]}
registry_unknown_req_count=${#REGISTRY_UNKNOWN_REQUIREMENTS[@]}
registry_mismatch_count=${#REGISTRY_MISMATCH_REQUIREMENTS[@]}
orphaned_count=${#REGISTRY_ORPHANED_PLAYBOOKS[@]}

fatal_issue_total=$((missing_file_count + registry_missing_file_count + missing_registry_count + registry_unknown_req_count + registry_mismatch_count + fixture_issue_total))

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

if [ "$registry_missing_file_count" -gt 0 ]; then
    log_error "CASE 1B: Registry References Files Missing On Disk"
    echo ""
    for file in "${REGISTRY_MISSING_FILES[@]}"; do
        echo -e "   ❌ Registry entry: ${BLUE}${file}${NC}"
        echo -e "      Issue: ${RED}File does not exist${NC}"
        echo ""
    done
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Regenerate the playbook via the UI or remove it from registry.json"
    echo ""
fi

if [ "$registry_unknown_req_count" -gt 0 ]; then
    log_error "CASE 2: Registry Lists Unknown Requirement IDs"
    echo ""
    for entry in "${REGISTRY_UNKNOWN_REQUIREMENTS[@]}"; do
        IFS='|' read -r file req_id <<< "$entry"
        echo -e "   ❌ Playbook: ${BLUE}${file}${NC}"
        echo -e "      Requirement: ${RED}${req_id}${NC} (not found in requirements/)"
        echo ""
    done
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Update requirements/*.json to include the validation, OR"
    echo "   • Regenerate registry.json after removing the requirement reference"
    echo ""
fi

if [ "$missing_registry_count" -gt 0 ]; then
    log_error "CASE 3: Requirements Reference Playbooks Missing From Registry"
    echo ""
    for entry in "${REQUIREMENT_REFERENCES_MISSING_REGISTRY[@]}"; do
        IFS='|' read -r req_id file <<< "$entry"
        echo -e "   ❌ Requirement: ${BLUE}${req_id}${NC}"
        echo -e "      File: ${RED}${file}${NC} (not present in registry.json)"
        echo ""
    done
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Run build-registry.mjs so the playbook order/coverage is updated"
    echo ""
fi

if [ "$registry_mismatch_count" -gt 0 ]; then
    log_error "CASE 4: Registry Coverage Differs From Requirement Files"
    echo ""
    for entry in "${REGISTRY_MISMATCH_REQUIREMENTS[@]}"; do
        IFS='|' read -r file reg_list actual_list <<< "$entry"
        formatted_registry=$(printf '%s' "$reg_list" | tr '\n' ',' | sed 's/,$//')
        formatted_actual=$(printf '%s' "$actual_list" | tr '\n' ',' | sed 's/,$//')
        [ -z "$formatted_registry" ] && formatted_registry="<none>"
        [ -z "$formatted_actual" ] && formatted_actual="<none>"
        echo -e "   ❌ Playbook: ${BLUE}${file}${NC}"
        echo -e "      Registry requirements: ${YELLOW}${formatted_registry}${NC}"
        echo -e "      Requirement files:     ${YELLOW}${formatted_actual}${NC}"
        echo ""
    done
    echo -e "   ${YELLOW}Action Required:${NC}"
    echo "   • Regenerate registry.json to sync with requirements/"
    echo ""
fi

if [ "$orphaned_count" -gt 0 ]; then
    log_warning "Playbooks Not Referenced By Any Requirements"
    echo ""
    for file in "${REGISTRY_ORPHANED_PLAYBOOKS[@]}"; do
        echo -e "   ⚠️  Playbook: ${BLUE}${file}${NC}"
        echo -e "      Note: Not referenced by any requirement validation"
        echo ""
    done
    echo -e "   ${YELLOW}Action Recommended:${NC}"
    echo "   • Add an automation validation entry in requirements/*.json to capture coverage"
    echo ""
fi

# Summary
echo "=============================================================="
echo ""

if [ "$fatal_issue_total" -eq 0 ]; then
    log_success "All playbooks are properly linked to requirements!"
    echo ""
    echo "   📊 Summary:"
    echo "      • Total playbooks: $playbook_count"
    echo "      • Total requirements: $requirement_count"
    echo "      • Playbook references in requirements: $reference_count"
    echo "      • Registry and requirements are in sync"
    if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
        echo "      • Fixture workflows validated: ${#FIXTURE_FILES[@]}"
    fi
    echo ""
    exit $EXIT_SUCCESS
else
    log_error "Validation Failed: $fatal_issue_total blocking issue(s) found"
    echo ""
    echo "   📊 Summary:"
    echo "      • Total playbooks: $playbook_count"
    echo "      • Total requirements: $requirement_count"
    echo "      • Missing playbook files: $missing_file_count"
    echo "      • Registry missing files: $registry_missing_file_count"
    echo "      • Requirements missing from registry: $missing_registry_count"
    echo "      • Unknown requirement IDs: $registry_unknown_req_count"
    echo "      • Coverage mismatches: $registry_mismatch_count"
    echo "      • Fixture warnings: $fixture_issue_total"
    if [ ${#FIXTURE_FILES[@]} -gt 0 ]; then
        echo "      • Fixture metadata issues: $fixture_metadata_count"
        echo "      • Fixture duplicates: $fixture_duplicate_count"
        echo "      • Unknown fixture references: $fixture_unknown_count"
        echo "      • Unused fixtures: $fixture_unused_count"
    fi
    echo ""

    # Exit with most severe error code
    if [ "$missing_file_count" -gt 0 ] || [ "$registry_missing_file_count" -gt 0 ]; then
        exit $EXIT_MISSING_PLAYBOOK_FILES
    elif [ "$registry_unknown_req_count" -gt 0 ] || [ "$missing_registry_count" -gt 0 ] || [ "$registry_mismatch_count" -gt 0 ]; then
        exit $EXIT_INVALID_REQUIREMENT_REFS
    elif [ $((fixture_metadata_count + fixture_duplicate_count + fixture_unknown_count + fixture_unused_count)) -gt 0 ]; then
        exit $EXIT_FIXTURE_ERRORS
    fi
fi
