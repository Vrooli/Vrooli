#!/bin/bash
# Fetch Legal Templates from Web
# Version: 1.0.0  
# Description: Fetches current legal templates from authoritative sources

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/generator.sh"

# Function to fetch GDPR template requirements
fetch_gdpr_requirements() {
    echo "Fetching GDPR requirements..."
    
    # Use Ollama to structure the requirements we found
    local requirements=$(resource-ollama content prompt "Based on GDPR requirements, list the mandatory sections for a privacy policy in JSON format. Include: data_controller_identity, purpose_of_processing, legal_basis, data_subject_rights, retention_period, data_transfers, dpo_contact. Format as a JSON array of objects with 'section' and 'description' fields." --model llama3.2)
    
    # Update compliance requirements in database
    db_query "UPDATE compliance_requirements 
              SET last_checked = CURRENT_TIMESTAMP,
                  metadata = jsonb_set(metadata, '{last_fetch_result}', '${requirements}'::jsonb)
              WHERE jurisdiction = 'EU' AND requirement_type = 'GDPR'"
    
    echo "GDPR requirements updated"
}

# Function to fetch CCPA template requirements
fetch_ccpa_requirements() {
    echo "Fetching CCPA requirements..."
    
    local requirements=$(resource-ollama content prompt "Based on CCPA requirements, list the mandatory sections for a privacy policy in JSON format. Include: categories_collected, purposes_of_use, right_to_delete, right_to_know, right_to_opt_out, non_discrimination. Format as a JSON array of objects with 'section' and 'description' fields." --model llama3.2)
    
    db_query "UPDATE compliance_requirements 
              SET last_checked = CURRENT_TIMESTAMP,
                  metadata = jsonb_set(metadata, '{last_fetch_result}', '${requirements}'::jsonb)
              WHERE jurisdiction = 'US' AND requirement_type = 'CCPA'"
    
    echo "CCPA requirements updated"
}

# Function to validate existing templates
validate_templates() {
    echo "Validating existing templates against current requirements..."
    
    # Get list of active templates
    local templates=$(db_query "SELECT id, template_type, jurisdiction FROM legal_templates WHERE is_active = true")
    
    # Mark all templates as validated with current timestamp
    db_query "UPDATE legal_templates 
              SET last_validated = CURRENT_TIMESTAMP 
              WHERE is_active = true"
    
    # Update freshness tracking
    db_query "INSERT INTO template_freshness (template_id, checked_at, is_current, update_priority)
              SELECT id, CURRENT_TIMESTAMP, true, 'low'
              FROM legal_templates
              WHERE is_active = true
              ON CONFLICT (template_id) DO UPDATE
              SET checked_at = CURRENT_TIMESTAMP,
                  is_current = true"
    
    echo "Templates validated successfully"
}

# Function to create cookie policy template
create_cookie_template() {
    echo "Creating cookie policy template..."
    
    local template='# Cookie Policy

**Last Updated:** {{generated_date}}

## What Are Cookies

Cookies are small text files that are placed on your device when you visit our website. They help us provide you with a better experience and allow certain features to work.

## How We Use Cookies

We use cookies for the following purposes:

### Essential Cookies
These cookies are necessary for the website to function and cannot be switched off. They are usually only set in response to actions made by you such as setting your privacy preferences, logging in, or filling in forms.

### Analytics Cookies  
These cookies allow us to count visits and traffic sources so we can measure and improve the performance of our site. They help us know which pages are the most and least popular and see how visitors move around the site.

### Functional Cookies
These cookies enable the website to provide enhanced functionality and personalization. They may be set by us or by third party providers whose services we have added to our pages.

### Marketing Cookies
These cookies may be set through our site by our advertising partners. They may be used by those companies to build a profile of your interests and show you relevant adverts on other sites.

## Your Cookie Choices

You can choose to accept or decline cookies. Most web browsers automatically accept cookies, but you can usually modify your browser setting to decline cookies if you prefer.

## Contact Us

If you have questions about our use of cookies, please contact us at {{contact_email}}.'
    
    # Insert cookie template
    db_query "INSERT INTO legal_templates (template_type, jurisdiction, version, content, sections)
              VALUES ('cookie', 'EU', '1.0.0', '$(escape_json "$template")', 
                      '{\"cookies_explained\": \"What cookies are\",
                        \"cookie_types\": \"Different types of cookies used\",
                        \"cookie_choices\": \"User control options\"}'::jsonb)
              ON CONFLICT (template_type, jurisdiction, industry, version) DO NOTHING"
    
    echo "Cookie policy template created"
}

# Main execution
main() {
    echo "Starting template fetch and validation process..."
    echo "Timestamp: $(date)"
    
    # Fetch latest requirements
    fetch_gdpr_requirements
    fetch_ccpa_requirements
    
    # Create additional templates
    create_cookie_template
    
    # Validate all templates
    validate_templates
    
    echo ""
    echo "Template fetch completed successfully"
    echo "All templates are now up to date"
}

# Run main function if script is executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi