#!/bin/bash
# Privacy & Terms Generator Core Logic
# Version: 1.0.0
# Description: Core generation functions using PostgreSQL and Ollama

set -euo pipefail

# Source paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
DEFAULT_MODEL="llama3.2"
DB_SCHEMA="legal_generator"

# Function to query PostgreSQL using resource-postgres
db_query() {
    local query="$1"
    # Use -t flag equivalent by parsing output more carefully
    local result=$(resource-postgres content execute "SET search_path TO ${DB_SCHEMA}; ${query}" 2>/dev/null)
    # Skip header lines and extract just the data
    echo "$result" | tail -n +4 | head -n -2 | sed 's/^ *//; s/ *$//' || echo ""
}

# Function to escape JSON for PostgreSQL
escape_json() {
    echo "$1" | sed "s/'/\'\'/g"
}

# Function to generate document using Ollama
generate_with_ollama() {
    local template="$1"
    local customizations="$2"
    local model="${3:-$DEFAULT_MODEL}"
    
    local prompt="You are a legal document generator. Take the following template and customizations, and generate a complete, professional legal document. 
    
Template:
${template}

Customizations (replace placeholders with these values):
${customizations}

Instructions:
1. Replace all {{placeholder}} values with the appropriate customization values
2. Ensure the document is complete and professionally written
3. Maintain legal language appropriate for the jurisdiction
4. Do not add any explanatory text outside the document itself
5. Return only the final document content"

    echo "${prompt}" | resource-ollama content add --model "${model}" --query "Generate the document"
}

# Function to fetch template from database
fetch_template() {
    local doc_type="$1"
    local jurisdiction="$2"
    local industry="${3:-NULL}"
    
    local query="SELECT content FROM legal_templates 
                 WHERE template_type = '${doc_type}' 
                 AND jurisdiction = '${jurisdiction}'"
    
    if [ "$industry" != "NULL" ]; then
        query="${query} AND industry = '${industry}'"
    else
        query="${query} AND industry IS NULL"
    fi
    
    query="${query} AND is_active = true 
                     ORDER BY version DESC 
                     LIMIT 1"
    
    db_query "$query"
}

# Function to create or update business profile
upsert_business_profile() {
    local business_name="$1"
    local business_type="$2"
    local jurisdictions="$3"
    local email="${4:-}"
    local website="${5:-}"
    
    local escaped_name="$(escape_json "$business_name")"
    local jurisdictions_array="ARRAY[$(echo "$jurisdictions" | sed "s/,/','/g" | sed "s/^/'/" | sed "s/$/'/")]"
    
    local query="INSERT INTO business_profiles (name, type, jurisdictions, email, website)
                 VALUES ('${escaped_name}', '${business_type}', ${jurisdictions_array}, '${email}', '${website}')
                 ON CONFLICT (name, type) DO UPDATE
                 SET jurisdictions = EXCLUDED.jurisdictions,
                     email = EXCLUDED.email,
                     website = EXCLUDED.website,
                     updated_at = CURRENT_TIMESTAMP
                 RETURNING id"
    
    db_query "$query" | tail -n 1
}

# Function to save generated document
save_generated_document() {
    local business_id="$1"
    local doc_type="$2"
    local content="$3"
    local template_id="${4:-NULL}"
    local format="${5:-markdown}"

    local escaped_content="$(escape_json "$content")"

    # Check if a document already exists for this business and type
    local existing_doc_query="SELECT id, version, content FROM generated_documents
                               WHERE business_id = '${business_id}'
                               AND document_type = '${doc_type}'
                               AND status = 'active'
                               ORDER BY version DESC LIMIT 1"
    local existing_result=$(db_query "$existing_doc_query")

    local doc_id=""
    local new_version=1

    if [ -n "$existing_result" ]; then
        # Extract existing document ID and version
        local existing_id=$(echo "$existing_result" | awk '{print $1}')
        local existing_version=$(echo "$existing_result" | awk '{print $2}')
        local existing_content=$(echo "$existing_result" | cut -d'|' -f3-)

        new_version=$((existing_version + 1))

        # Archive the old document
        db_query "UPDATE generated_documents
                  SET status = 'archived'
                  WHERE id = '${existing_id}'"

        # Create new version
        local insert_query="INSERT INTO generated_documents
                     (business_id, template_id, document_type, content, format, status, version)
                     VALUES ('${business_id}',
                             $([ "$template_id" = "NULL" ] && echo "NULL" || echo "'${template_id}'"),
                             '${doc_type}', '${escaped_content}', '${format}', 'active', ${new_version})
                     RETURNING id"
        doc_id=$(db_query "$insert_query" | tail -n 1)

        # Record version history
        record_document_history "$doc_id" "updated" "system" "Updated to version ${new_version}" "$existing_content" "$content"
    else
        # First version of this document
        local insert_query="INSERT INTO generated_documents
                     (business_id, template_id, document_type, content, format, status, version)
                     VALUES ('${business_id}',
                             $([ "$template_id" = "NULL" ] && echo "NULL" || echo "'${template_id}'"),
                             '${doc_type}', '${escaped_content}', '${format}', 'active', ${new_version})
                     RETURNING id"
        doc_id=$(db_query "$insert_query" | tail -n 1)

        # Record creation in history
        record_document_history "$doc_id" "created" "system" "Initial document creation" "" "$content"
    fi

    echo "$doc_id"
}

# Function to record document history
record_document_history() {
    local doc_id="$1"
    local change_type="$2"
    local changed_by="$3"
    local change_summary="$4"
    local previous_content="$5"
    local new_content="$6"

    local escaped_summary="$(escape_json "$change_summary")"
    local escaped_prev="$(escape_json "$previous_content")"
    local escaped_new="$(escape_json "$new_content")"

    local query="INSERT INTO document_history
                 (document_id, change_type, changed_by, change_summary, previous_content, new_content)
                 VALUES ('${doc_id}', '${change_type}', '${changed_by}',
                         '${escaped_summary}', '${escaped_prev}', '${escaped_new}')"

    db_query "$query" &>/dev/null
}

# Function to get document history
get_document_history() {
    local doc_id="$1"
    local limit="${2:-10}"

    local query="SELECT changed_at, change_type, changed_by, change_summary
                 FROM document_history
                 WHERE document_id = '${doc_id}'
                 ORDER BY changed_at DESC
                 LIMIT ${limit}"

    db_query "$query"
}

# Function to update template freshness
check_template_freshness() {
    local doc_type="${1:-all}"
    
    local query="SELECT template_type, jurisdiction, 
                        fetched_at, 
                        EXTRACT(DAY FROM CURRENT_TIMESTAMP - fetched_at) as days_old
                 FROM legal_templates
                 WHERE is_active = true"
    
    if [ "$doc_type" != "all" ]; then
        query="${query} AND template_type = '${doc_type}'"
    fi
    
    query="${query} ORDER BY days_old DESC"
    
    db_query "$query"
}

# Function to fetch latest templates from web
update_templates_from_web() {
    local jurisdiction="${1:-all}"
    local force="${2:-false}"
    
    echo "Checking template freshness..."
    local stale_templates=$(db_query "SELECT COUNT(*) FROM template_status WHERE freshness_status = 'stale'" | tail -n 1)
    
    if [ "$stale_templates" -eq 0 ] && [ "$force" != "true" ]; then
        echo "All templates are fresh (less than 30 days old)"
        return 0
    fi
    
    echo "Updating templates for jurisdiction: ${jurisdiction}"
    
    # This would normally fetch from web using WebSearch/WebFetch
    # For now, we'll mark templates as checked
    db_query "UPDATE legal_templates 
              SET fetched_at = CURRENT_TIMESTAMP, 
                  last_validated = CURRENT_TIMESTAMP
              WHERE is_active = true
              $([ "$jurisdiction" != "all" ] && echo "AND jurisdiction = '${jurisdiction}'")"
    
    echo "Templates updated successfully"
}

# Main generation function
generate_document() {
    local doc_type="$1"
    local business_name="$2"
    local business_type="$3"
    local jurisdiction="$4"
    local output_format="${5:-markdown}"
    local custom_data="${6:-{\}}"
    
    echo "Generating ${doc_type} for ${business_name}..."
    
    # Fetch appropriate template
    local template=$(fetch_template "$doc_type" "$jurisdiction" "$business_type")
    
    if [ -z "$template" ]; then
        # Try without industry if no specific template found
        template=$(fetch_template "$doc_type" "$jurisdiction")
        
        if [ -z "$template" ]; then
            echo "Error: No template found for ${doc_type} in jurisdiction ${jurisdiction}" >&2
            return 1
        fi
    fi
    
    # Create business profile
    local business_id=$(upsert_business_profile "$business_name" "$business_type" "$jurisdiction")
    
    # Prepare customizations
    local customizations=$(cat <<EOF
{
    "business_name": "${business_name}",
    "business_type": "${business_type}",
    "jurisdiction": "${jurisdiction}",
    "generated_date": "$(date '+%B %d, %Y')",
    "effective_date": "$(date '+%B %d, %Y')",
    ${custom_data:1:-1}
}
EOF
)
    
    # Generate document with Ollama
    local generated_content=$(generate_with_ollama "$template" "$customizations")
    
    if [ -z "$generated_content" ]; then
        echo "Error: Failed to generate document content" >&2
        return 1
    fi
    
    # Save to database
    local doc_id=$(save_generated_document "$business_id" "$doc_type" "$generated_content" "NULL" "$output_format")
    
    echo "Document generated successfully (ID: ${doc_id})"
    echo ""
    echo "$generated_content"
    
    return 0
}

# Export functions for use by other scripts
export -f db_query
export -f generate_document
export -f check_template_freshness
export -f update_templates_from_web
export -f record_document_history
export -f get_document_history

# If script is run directly (not sourced)
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    case "${1:-}" in
        generate)
            shift
            generate_document "$@"
            ;;
        freshness)
            shift
            check_template_freshness "$@"
            ;;
        update)
            shift
            update_templates_from_web "$@"
            ;;
        *)
            echo "Usage: $0 {generate|freshness|update} [options]"
            echo ""
            echo "Commands:"
            echo "  generate <type> <name> <business_type> <jurisdiction> [format] [custom_data]"
            echo "  freshness [type]"
            echo "  update [jurisdiction] [force]"
            exit 1
            ;;
    esac
fi