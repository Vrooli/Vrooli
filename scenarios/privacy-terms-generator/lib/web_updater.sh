#!/bin/bash
# Web Template Updater
# Version: 1.0.0
# Description: Fetches current legal templates from authoritative sources

set -euo pipefail

# Source generator functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/generator.sh"

# Template sources by jurisdiction
declare -A TEMPLATE_SOURCES=(
    ["US_PRIVACY"]="https://www.ftc.gov/business-guidance/resources/complying-coppa-frequently-asked-questions"
    ["EU_GDPR"]="https://gdpr-info.eu/"
    ["UK_PRIVACY"]="https://ico.org.uk/for-organisations/guide-to-data-protection/"
    ["CA_PRIVACY"]="https://oag.ca.gov/privacy/ccpa"
    ["AU_PRIVACY"]="https://www.oaic.gov.au/privacy/australian-privacy-principles"
)

# Function to fetch and extract legal requirements
fetch_legal_requirements() {
    local jurisdiction="$1"
    local doc_type="$2"
    local source_url="${3:-}"
    
    echo "Fetching ${doc_type} requirements for ${jurisdiction}..."
    
    # Determine source URL based on jurisdiction and type
    if [ -z "$source_url" ]; then
        local key="${jurisdiction}_${doc_type^^}"
        source_url="${TEMPLATE_SOURCES[$key]:-}"
        
        if [ -z "$source_url" ]; then
            echo "No source URL defined for ${jurisdiction} ${doc_type}"
            return 1
        fi
    fi
    
    # Use WebFetch to get the content
    local prompt="Extract the key legal requirements for ${doc_type} documents in ${jurisdiction}. Focus on:
1. Required disclosures and notices
2. User rights that must be mentioned
3. Data collection and usage policies
4. Contact information requirements
5. Specific clauses mandated by law

Format as a structured list of requirements that can be used to generate a compliant legal document."
    
    local content=$(curl -s -X POST http://localhost:8092/api/webfetch \
        -H "Content-Type: application/json" \
        -d "{\"url\": \"${source_url}\", \"prompt\": \"${prompt}\"}" 2>/dev/null | jq -r '.content // empty')
    
    if [ -z "$content" ]; then
        echo "Failed to fetch content from ${source_url}"
        return 1
    fi
    
    echo "$content"
}

# Function to generate template from requirements
generate_template_from_requirements() {
    local jurisdiction="$1"
    local doc_type="$2"
    local requirements="$3"
    local model="${4:-llama3.2}"
    
    local prompt="You are a legal document template generator. Create a professional ${doc_type} template for ${jurisdiction} jurisdiction based on these requirements:

${requirements}

The template should:
1. Include all required legal clauses
2. Use placeholders like {{business_name}}, {{effective_date}}, {{contact_email}} for customizable fields
3. Be structured with clear sections and headings
4. Use appropriate legal language for ${jurisdiction}
5. Be compliant with current regulations

Generate only the template content with placeholders, no explanations."

    echo "${prompt}" | resource-ollama content add --model "${model}" --query "Generate template"
}

# Function to validate template against requirements
validate_template() {
    local template="$1"
    local requirements="$2"
    local model="${3:-llama3.2}"
    
    local prompt="Compare this legal template against the requirements and identify any missing elements:

TEMPLATE:
${template}

REQUIREMENTS:
${requirements}

List only missing requirements or return 'VALID' if all requirements are met."

    local validation=$(resource-ollama content prompt "${prompt}" --model "${model}")
    
    if [[ "$validation" == *"VALID"* ]]; then
        echo "Template validated successfully"
        return 0
    else
        echo "Template validation issues:"
        echo "$validation"
        return 1
    fi
}

# Function to update template in database
update_template_in_db() {
    local jurisdiction="$1"
    local doc_type="$2"
    local content="$3"
    local source_url="$4"
    
    local escaped_content="$(escape_json "$content")"
    local escaped_url="$(escape_json "$source_url")"
    
    # Check if template exists
    local existing=$(db_query "SELECT id, version FROM legal_templates 
                               WHERE template_type = '${doc_type}' 
                               AND jurisdiction = '${jurisdiction}' 
                               AND is_active = true
                               ORDER BY version DESC LIMIT 1")
    
    if [ -n "$existing" ]; then
        local template_id=$(echo "$existing" | awk '{print $1}')
        local current_version=$(echo "$existing" | awk '{print $3}')
        local new_version=$((current_version + 1))
        
        # Deactivate old version
        db_query "UPDATE legal_templates SET is_active = false WHERE id = '${template_id}'"
        
        # Insert new version
        db_query "INSERT INTO legal_templates 
                  (template_type, jurisdiction, content, source_url, version, is_active)
                  VALUES ('${doc_type}', '${jurisdiction}', '${escaped_content}', 
                          '${escaped_url}', ${new_version}, true)"
    else
        # Insert first version
        db_query "INSERT INTO legal_templates 
                  (template_type, jurisdiction, content, source_url, version, is_active)
                  VALUES ('${doc_type}', '${jurisdiction}', '${escaped_content}', 
                          '${escaped_url}', 1, true)"
    fi
    
    echo "Template updated in database"
}

# Function to perform full template update
perform_template_update() {
    local jurisdiction="$1"
    local doc_type="$2"
    local force="${3:-false}"
    
    # Check if update is needed
    if [ "$force" != "true" ]; then
        local freshness=$(db_query "SELECT EXTRACT(DAY FROM CURRENT_TIMESTAMP - fetched_at) as days_old 
                                    FROM legal_templates
                                    WHERE template_type = '${doc_type}' 
                                    AND jurisdiction = '${jurisdiction}'
                                    AND is_active = true
                                    LIMIT 1")
        
        if [ -n "$freshness" ] && [ "$freshness" -lt 14 ]; then
            echo "Template is fresh (${freshness} days old), skipping update"
            return 0
        fi
    fi
    
    echo "Updating ${doc_type} template for ${jurisdiction}..."
    
    # Fetch requirements
    local requirements=$(fetch_legal_requirements "$jurisdiction" "$doc_type")
    if [ $? -ne 0 ]; then
        echo "Failed to fetch requirements"
        return 1
    fi
    
    # Generate template
    local template=$(generate_template_from_requirements "$jurisdiction" "$doc_type" "$requirements")
    if [ -z "$template" ]; then
        echo "Failed to generate template"
        return 1
    fi
    
    # Validate template
    validate_template "$template" "$requirements"
    if [ $? -ne 0 ]; then
        echo "Warning: Template validation failed, but continuing..."
    fi
    
    # Update in database
    local source_key="${jurisdiction}_${doc_type^^}"
    local source_url="${TEMPLATE_SOURCES[$source_key]:-unknown}"
    update_template_in_db "$jurisdiction" "$doc_type" "$template" "$source_url"
    
    echo "Template update completed successfully"
}

# Function to update all templates
update_all_templates() {
    local force="${1:-false}"
    
    echo "Starting comprehensive template update..."
    
    local jurisdictions=("US" "EU" "UK" "CA" "AU")
    local doc_types=("privacy" "terms" "cookie" "eula")
    
    local updated=0
    local failed=0
    
    for jurisdiction in "${jurisdictions[@]}"; do
        for doc_type in "${doc_types[@]}"; do
            echo ""
            echo "Processing ${jurisdiction} ${doc_type}..."
            
            perform_template_update "$jurisdiction" "$doc_type" "$force"
            if [ $? -eq 0 ]; then
                ((updated++))
            else
                ((failed++))
                echo "Failed to update ${jurisdiction} ${doc_type}"
            fi
        done
    done
    
    echo ""
    echo "Update summary:"
    echo "  Updated: ${updated} templates"
    echo "  Failed: ${failed} templates"
    
    # Update metadata
    db_query "UPDATE legal_templates 
              SET metadata = jsonb_set(
                  COALESCE(metadata, '{}'::jsonb),
                  '{last_bulk_update}',
                  to_jsonb(CURRENT_TIMESTAMP)
              )
              WHERE is_active = true"
    
    return $([ $failed -eq 0 ] && echo 0 || echo 1)
}

# Export functions
export -f fetch_legal_requirements
export -f generate_template_from_requirements
export -f validate_template
export -f update_template_in_db
export -f perform_template_update
export -f update_all_templates

# Main execution
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    case "${1:-}" in
        update)
            shift
            if [ "${1:-}" = "all" ]; then
                update_all_templates "${2:-false}"
            else
                perform_template_update "$@"
            fi
            ;;
        fetch)
            shift
            fetch_legal_requirements "$@"
            ;;
        validate)
            shift
            echo "Please provide template and requirements via stdin"
            ;;
        *)
            echo "Usage: $0 {update|fetch|validate} [options]"
            echo ""
            echo "Commands:"
            echo "  update all [force]                    - Update all templates"
            echo "  update <jurisdiction> <type> [force]  - Update specific template"
            echo "  fetch <jurisdiction> <type> [url]     - Fetch requirements"
            echo "  validate                               - Validate template (via stdin)"
            exit 1
            ;;
    esac
fi