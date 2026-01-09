#!/bin/bash
# manage-sections.sh - Section management helper for agents
#
# AGENT INSTRUCTIONS:
# This script helps you add, remove, list, and validate landing page sections.
# Run with --help to see all available commands.
#
# QUICK REFERENCE:
#   ./scripts/manage-sections.sh list              # List all sections
#   ./scripts/manage-sections.sh validate          # Check consistency across all files
#   ./scripts/manage-sections.sh add <id>          # Add new section (guided)
#   ./scripts/manage-sections.sh remove <id>       # Remove section (guided)
#   ./scripts/manage-sections.sh info <id>         # Show section details
#
# ARCHITECTURE NOTE:
# The registry.tsx file is now the source of truth for component registration.
# Adding a section only requires 3 files:
#   1. Component TSX
#   2. registry.tsx
#   3. schema.sql (CHECK constraint)

set -euo pipefail

SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/.." && pwd)"

# File paths (relative to scenario root)
INDEX_FILE="${SCENARIO_ROOT}/api/templates/sections/_index.json"
SCHEMA_DIR="${SCENARIO_ROOT}/api/templates/sections"
COMPONENT_DIR="${SCENARIO_ROOT}/generated/test/ui/src/components/sections"
REGISTRY_FILE="${SCENARIO_ROOT}/generated/test/ui/src/components/sections/registry.tsx"
DB_SCHEMA_FILE="${SCENARIO_ROOT}/initialization/postgres/schema.sql"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Get all section IDs from _index.json
get_registered_sections() {
    jq -r '.sections[].id' "$INDEX_FILE" 2>/dev/null | sort
}

# Get implemented sections (status = "implemented")
get_implemented_sections() {
    jq -r '.sections[] | select(.status == "implemented") | .id' "$INDEX_FILE" 2>/dev/null | sort
}

# Get section types from registry.tsx (SECTION_REGISTRY keys)
get_registry_section_types() {
    # Extract section types from SECTION_REGISTRY object keys
    # Matches both 'key': and key: formats
    grep -oP "^\s*['\"]?([a-z][a-z0-9-]*)['\"]?\s*:\s*\{" "$REGISTRY_FILE" 2>/dev/null | \
        grep -oP "[a-z][a-z0-9-]*" | sort || true
}

# Get section types from DB schema
get_db_section_types() {
    grep -oP "section_type IN \([^)]+\)" "$DB_SCHEMA_FILE" 2>/dev/null | \
        grep -oP "'[a-z-]+'" | tr -d "'" | sort || true
}

# Check if component exists (reads actual path from registry)
component_exists() {
    local id="$1"
    local component_path
    component_path=$(jq -r --arg id "$id" '.sections[] | select(.id == $id) | .component' "$INDEX_FILE" 2>/dev/null)
    if [[ -n "$component_path" && "$component_path" != "null" ]]; then
        [[ -f "${SCENARIO_ROOT}/${component_path}" ]]
    else
        # Fallback to derived name
        local pascal_name
        pascal_name=$(echo "$id" | sed -r 's/(^|-)([a-z])/\U\2/g')
        [[ -f "${COMPONENT_DIR}/${pascal_name}Section.tsx" ]]
    fi
}

# Check if schema exists
schema_exists() {
    local id="$1"
    [[ -f "${SCHEMA_DIR}/${id}.json" ]]
}

# Convert kebab-case to PascalCase
to_pascal_case() {
    echo "$1" | sed -r 's/(^|-)([a-z])/\U\2/g'
}

# List all sections
cmd_list() {
    echo ""
    echo "=== Landing Page Sections ==="
    echo ""
    printf "%-20s %-15s %-20s %-10s %-10s\n" "ID" "STATUS" "COMPONENT" "SCHEMA" "CATEGORY"
    printf "%-20s %-15s %-20s %-10s %-10s\n" "----" "------" "---------" "------" "--------"

    jq -r '.sections[] | [.id, .status, .category] | @tsv' "$INDEX_FILE" | while IFS=$'\t' read -r id status category; do
        local pascal_name
        pascal_name=$(to_pascal_case "$id")
        local comp_exists="no"
        local schema_exists_flag="no"

        component_exists "$id" && comp_exists="yes"
        schema_exists "$id" && schema_exists_flag="yes"

        printf "%-20s %-15s %-20s %-10s %-10s\n" "$id" "$status" "${pascal_name}Section" "$schema_exists_flag" "$category"
    done

    echo ""
    echo "Total sections: $(get_registered_sections | wc -l)"
    echo "Implemented: $(get_implemented_sections | wc -l)"
    echo ""
}

# Validate consistency across all files
cmd_validate() {
    local errors=0
    local warnings=0

    echo ""
    echo "=== Section Consistency Validation ==="
    echo ""

    # Get sections from each source
    local registered
    registered=$(get_registered_sections)
    local implemented
    implemented=$(get_implemented_sections)
    local registry_types
    registry_types=$(get_registry_section_types)
    local db_types
    db_types=$(get_db_section_types)

    log_info "Checking registry.tsx (source of truth for UI)..."
    if [[ -z "$registry_types" ]]; then
        log_error "No sections found in registry.tsx"
        ((errors++))
    else
        log_success "Found $(echo "$registry_types" | wc -l) sections in registry.tsx"
    fi

    log_info "Checking _index.json metadata..."
    if [[ -z "$registered" ]]; then
        log_warn "No sections found in _index.json (optional, but recommended for documentation)"
    else
        log_success "Found $(echo "$registered" | wc -l) sections in _index.json"
    fi

    # Check each registered section in registry.tsx
    log_info "Checking sections in registry.tsx..."
    for id in $registry_types; do
        # Check component exists
        if ! component_exists "$id"; then
            local pascal_name
            pascal_name=$(to_pascal_case "$id")
            log_error "Missing component: ${COMPONENT_DIR}/${pascal_name}Section.tsx"
            ((errors++))
        fi

        # Check in DB schema
        if ! echo "$db_types" | grep -q "^${id}$"; then
            log_error "Missing from DB CHECK constraint in schema.sql: $id"
            ((errors++))
        fi
    done

    # Check for schema-only sections (in _index.json but not in registry.tsx)
    log_info "Checking schema-only sections..."
    for id in $implemented; do
        if ! echo "$registry_types" | grep -q "^${id}$"; then
            log_warn "Section '$id' is marked implemented in _index.json but not in registry.tsx"
            ((warnings++))
        fi
    done

    # Check for orphaned DB entries
    log_info "Checking for orphaned entries..."
    for type in $db_types; do
        if ! echo "$registry_types" | grep -q "^${type}$"; then
            log_warn "Type '$type' in schema.sql but not in registry.tsx"
            ((warnings++))
        fi
    done

    echo ""
    echo "=== Validation Summary ==="
    if [[ $errors -eq 0 && $warnings -eq 0 ]]; then
        log_success "All checks passed! Sections are consistent across all files."
        return 0
    else
        [[ $errors -gt 0 ]] && log_error "$errors error(s) found"
        [[ $warnings -gt 0 ]] && log_warn "$warnings warning(s) found"
        return 1
    fi
}

# Show info about a specific section
cmd_info() {
    local id="$1"

    if ! echo "$(get_registered_sections)" | grep -q "^${id}$"; then
        log_error "Section '$id' not found in registry"
        return 1
    fi

    echo ""
    echo "=== Section: $id ==="
    echo ""

    # Get section data from _index.json
    local section_data
    section_data=$(jq -r --arg id "$id" '.sections[] | select(.id == $id)' "$INDEX_FILE")

    echo "Name: $(echo "$section_data" | jq -r '.name')"
    echo "Description: $(echo "$section_data" | jq -r '.description')"
    echo "Category: $(echo "$section_data" | jq -r '.category')"
    echo "Status: $(echo "$section_data" | jq -r '.status')"
    echo "Default Order: $(echo "$section_data" | jq -r '.default_order')"
    echo ""

    echo "Content Fields:"
    echo "$section_data" | jq -r '.content_fields[]' | sed 's/^/  - /'
    echo ""

    local pascal_name
    pascal_name=$(to_pascal_case "$id")

    echo "Files:"
    echo "  Schema:    ${SCHEMA_DIR}/${id}.json"
    echo "  Component: ${COMPONENT_DIR}/${pascal_name}Section.tsx"
    echo ""

    echo "File Status:"
    schema_exists "$id" && echo "  Schema:    EXISTS" || echo "  Schema:    MISSING"
    component_exists "$id" && echo "  Component: EXISTS" || echo "  Component: MISSING"
    echo ""
}

# Guide for adding a new section
cmd_add() {
    local id="${1:-}"

    if [[ -z "$id" ]]; then
        echo "Usage: $0 add <section-id>"
        echo ""
        echo "Example: $0 add comparison-table"
        echo ""
        echo "The section ID should be kebab-case (lowercase with hyphens)."
        return 1
    fi

    # Validate ID format
    if ! [[ "$id" =~ ^[a-z][a-z0-9-]*$ ]]; then
        log_error "Invalid section ID. Must be kebab-case (e.g., 'comparison-table')"
        return 1
    fi

    # Check if already exists
    if echo "$(get_registered_sections)" | grep -q "^${id}$"; then
        log_error "Section '$id' already exists"
        return 1
    fi

    local pascal_name
    pascal_name=$(to_pascal_case "$id")

    echo ""
    echo "=== Adding Section: $id ==="
    echo ""
    echo "Only 3 files to update! (Thanks to the registry pattern)"
    echo ""
    echo "REQUIRED:"
    echo "  1. CREATE: ${COMPONENT_DIR}/${pascal_name}Section.tsx"
    echo "     - Implement the React component"
    echo ""
    echo "  2. MODIFY: ${REGISTRY_FILE}"
    echo "     - Import your component and add to SECTION_REGISTRY"
    echo ""
    echo "  3. MODIFY: ${DB_SCHEMA_FILE}"
    echo "     - Add '$id' to CHECK constraint (line ~88)"
    echo ""
    echo "OPTIONAL (for documentation):"
    echo "  - CREATE: ${SCHEMA_DIR}/${id}.json (field definitions)"
    echo "  - MODIFY: ${INDEX_FILE} (section metadata)"
    echo ""

    # Generate component template
    echo "=== Step 1: Component Template (${pascal_name}Section.tsx) ==="
    cat << EOF
interface ${pascal_name}SectionProps {
  content: {
    title?: string;
    // Add other fields as needed
  };
}

export function ${pascal_name}Section({ content }: ${pascal_name}SectionProps) {
  return (
    <section className="py-24 bg-slate-950">
      <div className="container mx-auto px-6">
        <h2 className="text-4xl font-bold text-white text-center mb-12">
          {content.title || 'Default Title'}
        </h2>
        {/* Add your content here */}
      </div>
    </section>
  );
}
EOF
    echo ""

    # Show what to add to registry.tsx
    echo "=== Step 2: Add to registry.tsx ==="
    echo ""
    echo "Add import at top:"
    echo "  import { ${pascal_name}Section } from './${pascal_name}Section';"
    echo ""
    echo "Add to SECTION_REGISTRY object:"
    cat << EOF
  '${id}': {
    component: ${pascal_name}Section,
    name: '${pascal_name} Section',
    category: 'value-proposition',
    defaultOrder: 50,
  },
EOF
    echo ""

    # Show what to add to schema.sql
    echo "=== Step 3: Add to schema.sql CHECK constraint ==="
    echo "Add '${id}' to the CHECK constraint list (line ~88)"
    echo ""

    echo "=== OPTIONAL: Schema Template (${id}.json) ==="
    cat << EOF
{
  "\$schema": "./_schema.json",
  "id": "${id}",
  "name": "${pascal_name} Section",
  "description": "TODO: Add description",
  "category": "value-proposition",
  "fields": {
    "title": {
      "type": "string",
      "label": "Section Title",
      "placeholder": "Your title here",
      "max_length": 80,
      "required": true
    }
  },
  "ui_component": "${pascal_name}Section",
  "default_order": 50
}
EOF
    echo ""

    log_info "After making changes, run: $0 validate"
}

# Guide for removing a section
cmd_remove() {
    local id="${1:-}"

    if [[ -z "$id" ]]; then
        echo "Usage: $0 remove <section-id>"
        return 1
    fi

    # Check both registry.tsx and _index.json
    local registry_types
    registry_types=$(get_registry_section_types)
    if ! echo "$registry_types" | grep -q "^${id}$"; then
        if ! echo "$(get_registered_sections)" | grep -q "^${id}$"; then
            log_error "Section '$id' not found in registry.tsx or _index.json"
            return 1
        fi
    fi

    local pascal_name
    pascal_name=$(to_pascal_case "$id")

    echo ""
    echo "=== Removing Section: $id ==="
    echo ""
    echo "Only 3 files to update! (Thanks to the registry pattern)"
    echo ""
    echo "REQUIRED:"
    echo "  1. MODIFY: ${REGISTRY_FILE}"
    echo "     - Remove import and entry from SECTION_REGISTRY"
    echo ""
    echo "  2. DELETE: ${COMPONENT_DIR}/${pascal_name}Section.tsx"
    echo ""
    echo "  3. MODIFY: ${DB_SCHEMA_FILE}"
    echo "     - Remove '${id}' from CHECK constraint"
    echo ""
    echo "OPTIONAL (if they exist):"
    echo "  - DELETE: ${SCHEMA_DIR}/${id}.json"
    echo "  - MODIFY: ${INDEX_FILE} - Remove entry from 'sections' array"
    echo ""

    log_warn "Make sure no existing content uses this section type before removing!"
    log_info "After making changes, run: $0 validate"
}

# Show help
cmd_help() {
    echo ""
    echo "Landing Page Section Manager"
    echo "============================"
    echo ""
    echo "This script helps agents manage landing page sections consistently."
    echo "Only 3 files needed to add a section! (Thanks to the registry pattern)"
    echo ""
    echo "Usage: $0 <command> [args]"
    echo ""
    echo "Commands:"
    echo "  list              List all registered sections"
    echo "  validate          Check consistency across all files"
    echo "  info <id>         Show detailed info about a section"
    echo "  add <id>          Guide for adding a new section (3 files only!)"
    echo "  remove <id>       Guide for removing a section"
    echo "  help              Show this help message"
    echo ""
    echo "Files to update when adding a section:"
    echo "  REQUIRED:"
    echo "    1. Component:   generated/test/ui/src/components/sections/{Name}Section.tsx"
    echo "    2. Registry:    generated/test/ui/src/components/sections/registry.tsx"
    echo "    3. DB Schema:   initialization/postgres/schema.sql (CHECK constraint)"
    echo ""
    echo "  OPTIONAL (for documentation):"
    echo "    - Schema:       api/templates/sections/{id}.json"
    echo "    - Metadata:     api/templates/sections/_index.json"
    echo ""
    echo "Auto-derived files (no manual updates needed):"
    echo "  - SectionType:    Re-exported from registry.tsx"
    echo "  - PublicHome:     Uses getSectionComponent() from registry"
    echo ""
}

# Main
main() {
    local cmd="${1:-help}"
    shift || true

    case "$cmd" in
        list)     cmd_list "$@" ;;
        validate) cmd_validate "$@" ;;
        info)     cmd_info "$@" ;;
        add)      cmd_add "$@" ;;
        remove)   cmd_remove "$@" ;;
        help|--help|-h) cmd_help ;;
        *)
            log_error "Unknown command: $cmd"
            cmd_help
            return 1
            ;;
    esac
}

main "$@"
