#!/usr/bin/env bash
# Email Service Management Script
# Provides CLI interface for email processing operations

set -euo pipefail

# Email processing functions for the test application

#######################################
# Send notification email via configured provider
# Arguments:
#   $1 - Recipient email address
#   $2 - Subject line
#   $3 - Message body
#   $4 - Optional: attachment path
# Returns: 0 on success, 1 on failure
#######################################
email::send_notification() {
    local recipient="$1"
    local subject="$2"
    local message="$3"
    local attachment="${4:-}"
    
    if [[ -z "$recipient" ]] || [[ -z "$subject" ]]; then
        echo "Error: Recipient and subject are required" >&2
        return 1
    fi
    
    # Validate email format
    if ! email::validate_address "$recipient"; then
        echo "Error: Invalid email address format" >&2
        return 1
    fi
    
    # Prepare email data
    local email_data
    email_data=$(jq -n \
        --arg to "$recipient" \
        --arg subject "$subject" \
        --arg body "$message" \
        '{
            to: $to,
            subject: $subject,
            body: $body,
            timestamp: now
        }')
    
    # Add attachment if provided
    if [[ -n "$attachment" ]] && [[ -f "$attachment" ]]; then
        email_data=$(echo "$email_data" | jq --arg attachment "$attachment" '. + {attachment: $attachment}')
    fi
    
    # Send via API
    curl -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $EMAIL_API_TOKEN" \
        -d "$email_data" \
        "$EMAIL_API_ENDPOINT/send" || {
            echo "Error: Failed to send email" >&2
            return 1
        }
    
    echo "Email sent successfully to $recipient"
    return 0
}

#######################################
# Validate email address format
# Arguments:
#   $1 - Email address to validate
# Returns: 0 if valid, 1 if invalid
#######################################
email::validate_address() {
    local email="$1"
    
    # Basic email regex validation
    if [[ "$email" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Fetch emails from provider API
# Arguments:
#   $1 - User ID
#   $2 - Optional: limit (default: 50)
#   $3 - Optional: offset (default: 0)
# Returns: JSON array of email objects
#######################################
email::fetch_emails() {
    local user_id="$1"
    local limit="${2:-50}"
    local offset="${3:-0}"
    
    if [[ -z "$user_id" ]]; then
        echo "Error: User ID required" >&2
        return 1
    fi
    
    # Validate numeric parameters
    if ! [[ "$limit" =~ ^[0-9]+$ ]] || ! [[ "$offset" =~ ^[0-9]+$ ]]; then
        echo "Error: Limit and offset must be numeric" >&2
        return 1
    fi
    
    # Call email provider API
    local response
    response=$(curl -s \
        -H "Authorization: Bearer $EMAIL_API_TOKEN" \
        "$EMAIL_API_ENDPOINT/emails?user_id=$user_id&limit=$limit&offset=$offset")
    
    # Check if response is valid JSON
    if ! echo "$response" | jq . >/dev/null 2>&1; then
        echo "Error: Invalid response from email API" >&2
        return 1
    fi
    
    echo "$response"
}

#######################################
# Process email batch with AI categorization
# Arguments:
#   $1 - Path to JSON file containing email batch
# Returns: 0 on success
#######################################
email::process_batch() {
    local batch_file="$1"
    
    if [[ ! -f "$batch_file" ]]; then
        echo "Error: Batch file not found: $batch_file" >&2
        return 1
    fi
    
    # Validate JSON format
    if ! jq . "$batch_file" >/dev/null 2>&1; then
        echo "Error: Invalid JSON format in batch file" >&2
        return 1
    fi
    
    local batch_id
    batch_id="batch_$(date +%s)"
    
    echo "Processing email batch: $batch_id"
    
    # Process each email in the batch
    jq -c '.emails[]' "$batch_file" | while read -r email; do
        local email_id
        email_id=$(echo "$email" | jq -r '.id')
        
        # Categorize email using AI
        local category
        category=$(email::categorize_ai "$email")
        
        # Update email in database
        email::update_category "$email_id" "$category"
        
        echo "Processed email $email_id: category = $category"
    done
    
    echo "Batch processing complete: $batch_id"
}

#######################################
# Categorize email using AI service
# Arguments:
#   $1 - Email JSON object
# Returns: Category string
#######################################
email::categorize_ai() {
    local email_json="$1"
    
    local subject body
    subject=$(echo "$email_json" | jq -r '.subject')
    body=$(echo "$email_json" | jq -r '.body // ""')
    
    # Prepare AI request
    local ai_request
    ai_request=$(jq -n \
        --arg subject "$subject" \
        --arg body "$body" \
        '{
            model: "gpt-3.5-turbo",
            messages: [
                {
                    role: "system",
                    content: "Categorize this email as: important, normal, spam, or promotional"
                },
                {
                    role: "user", 
                    content: ("Subject: " + $subject + "\nBody: " + $body)
                }
            ],
            max_tokens: 10
        }')
    
    # Call AI service
    local ai_response
    ai_response=$(curl -s \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AI_API_TOKEN" \
        -d "$ai_request" \
        "$AI_API_ENDPOINT/chat/completions")
    
    # Extract category from response
    local category
    category=$(echo "$ai_response" | jq -r '.choices[0].message.content // "normal"')
    
    echo "$category"
}

# Main CLI interface
case "${1:-help}" in
    send)
        email::send_notification "${2:-}" "${3:-}" "${4:-}" "${5:-}"
        ;;
    fetch)
        email::fetch_emails "${2:-}" "${3:-}" "${4:-}"
        ;;
    process)
        email::process_batch "${2:-}"
        ;;
    validate)
        email::validate_address "${2:-}"
        ;;
    help|*)
        echo "Usage: $0 {send|fetch|process|validate} [args...]"
        echo "  send <email> <subject> <message> [attachment]"
        echo "  fetch <user_id> [limit] [offset]"  
        echo "  process <batch_file>"
        echo "  validate <email>"
        ;;
esac