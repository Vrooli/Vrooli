#!/usr/bin/env bash
################################################################################
# BTCPay Crowdfunding Module - Campaign Management
#
# Handles crowdfunding campaign creation and management for BTCPay Server.
################################################################################

set -euo pipefail

# Set data directory if not already set
DATA_DIR="${DATA_DIR:-${APP_ROOT}/data/resources/btcpay}"

# ==============================================================================
# Crowdfunding Campaign Management
# ==============================================================================

btcpay::crowdfunding::configure() {
    local store_id="${1:-}"
    
    if [[ -z "$store_id" ]]; then
        log::error "Store ID required for crowdfunding configuration"
        echo "Usage: resource-btcpay crowdfunding configure <store-id>"
        return 1
    fi
    
    log::info "Configuring crowdfunding for store: $store_id"
    
    # Create crowdfunding config directory
    local config_dir="${DATA_DIR}/crowdfunding"
    mkdir -p "$config_dir"
    
    # Store configuration
    cat > "${config_dir}/config.json" <<EOF
{
    "store_id": "$store_id",
    "enabled": true,
    "default_currency": "BTC",
    "allow_anonymous": true,
    "show_contributors": true,
    "minimum_contribution": 0.0001
}
EOF
    
    log::success "Crowdfunding configured for store: $store_id"
}

btcpay::crowdfunding::create_campaign() {
    local title="${1:-}"
    local goal="${2:-}"
    local currency="${3:-BTC}"
    local description="${4:-}"
    local end_date="${5:-}"
    
    if [[ -z "$title" ]] || [[ -z "$goal" ]]; then
        log::error "Title and goal amount are required"
        echo "Usage: resource-btcpay crowdfunding create <title> <goal> [currency] [description] [end-date]"
        return 1
    fi
    
    log::info "Creating crowdfunding campaign: $title"
    
    # Generate campaign ID
    local campaign_id="cf_$(date +%s)_$(openssl rand -hex 4)"
    local campaigns_dir="${DATA_DIR}/crowdfunding/campaigns"
    mkdir -p "$campaigns_dir"
    
    # Create campaign file
    cat > "${campaigns_dir}/${campaign_id}.json" <<EOF
{
    "id": "$campaign_id",
    "title": "$title",
    "description": "$description",
    "goal_amount": $goal,
    "currency": "$currency",
    "raised_amount": 0,
    "contributors": 0,
    "created_at": "$(date -Iseconds)",
    "end_date": "$end_date",
    "status": "active",
    "contributions": []
}
EOF
    
    log::success "Campaign created: $campaign_id"
    echo "Campaign ID: $campaign_id"
    echo "Title: $title"
    echo "Goal: $goal $currency"
    echo "Status: Active"
}

btcpay::crowdfunding::list_campaigns() {
    local campaigns_dir="${DATA_DIR}/crowdfunding/campaigns"
    
    if [[ ! -d "$campaigns_dir" ]]; then
        log::info "No campaigns found"
        return 0
    fi
    
    log::info "Listing crowdfunding campaigns:"
    echo ""
    echo "ID | Title | Goal | Raised | Status"
    echo "---|-------|------|--------|--------"
    
    for campaign_file in "${campaigns_dir}"/*.json; do
        if [[ -f "$campaign_file" ]]; then
            local campaign_data=$(cat "$campaign_file")
            local id=$(echo "$campaign_data" | jq -r '.id')
            local title=$(echo "$campaign_data" | jq -r '.title')
            local goal=$(echo "$campaign_data" | jq -r '.goal_amount')
            local raised=$(echo "$campaign_data" | jq -r '.raised_amount')
            local currency=$(echo "$campaign_data" | jq -r '.currency')
            local status=$(echo "$campaign_data" | jq -r '.status')
            
            echo "$id | $title | $goal $currency | $raised $currency | $status"
        fi
    done
}

btcpay::crowdfunding::update_campaign() {
    local campaign_id="${1:-}"
    local property="${2:-}"
    local value="${3:-}"
    
    if [[ -z "$campaign_id" ]] || [[ -z "$property" ]] || [[ -z "$value" ]]; then
        log::error "Campaign ID, property, and value are required"
        echo "Usage: resource-btcpay crowdfunding update <campaign-id> <property> <value>"
        echo "Properties: title, description, goal_amount, end_date, status"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    log::info "Updating campaign $campaign_id: $property = $value"
    
    # Update the specific property
    case "$property" in
        goal_amount|raised_amount|contributors)
            jq ".$property = $value" "$campaign_file" > "${campaign_file}.tmp" && mv "${campaign_file}.tmp" "$campaign_file"
            ;;
        *)
            jq ".$property = \"$value\"" "$campaign_file" > "${campaign_file}.tmp" && mv "${campaign_file}.tmp" "$campaign_file"
            ;;
    esac
    
    log::success "Campaign updated successfully"
}

btcpay::crowdfunding::contribute() {
    local campaign_id="${1:-}"
    local amount="${2:-}"
    local contributor_name="${3:-Anonymous}"
    local message="${4:-}"
    
    if [[ -z "$campaign_id" ]] || [[ -z "$amount" ]]; then
        log::error "Campaign ID and amount are required"
        echo "Usage: resource-btcpay crowdfunding contribute <campaign-id> <amount> [name] [message]"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    log::info "Recording contribution to campaign: $campaign_id"
    
    # Read current campaign data
    local campaign_data=$(cat "$campaign_file")
    local current_raised=$(echo "$campaign_data" | jq -r '.raised_amount')
    local current_contributors=$(echo "$campaign_data" | jq -r '.contributors')
    
    # Calculate new totals
    local new_raised=$(echo "$current_raised + $amount" | bc)
    local new_contributors=$((current_contributors + 1))
    
    # Create contribution record
    local contribution_id="contrib_$(date +%s)_$(openssl rand -hex 4)"
    local contribution=$(jq -n \
        --arg id "$contribution_id" \
        --arg amount "$amount" \
        --arg name "$contributor_name" \
        --arg message "$message" \
        --arg date "$(date -Iseconds)" \
        '{id: $id, amount: ($amount | tonumber), contributor: $name, message: $message, date: $date}')
    
    # Update campaign with new contribution
    jq ".raised_amount = $new_raised | .contributors = $new_contributors | .contributions += [$contribution]" \
        "$campaign_file" > "${campaign_file}.tmp" && mv "${campaign_file}.tmp" "$campaign_file"
    
    log::success "Contribution recorded: $amount from $contributor_name"
    echo "New total raised: $new_raised"
    echo "Total contributors: $new_contributors"
}

btcpay::crowdfunding::status() {
    local campaign_id="${1:-}"
    
    if [[ -z "$campaign_id" ]]; then
        log::error "Campaign ID required"
        echo "Usage: resource-btcpay crowdfunding status <campaign-id>"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    log::info "Campaign Status: $campaign_id"
    echo ""
    
    local campaign_data=$(cat "$campaign_file")
    
    echo "Title: $(echo "$campaign_data" | jq -r '.title')"
    echo "Status: $(echo "$campaign_data" | jq -r '.status')"
    echo "Goal: $(echo "$campaign_data" | jq -r '.goal_amount') $(echo "$campaign_data" | jq -r '.currency')"
    echo "Raised: $(echo "$campaign_data" | jq -r '.raised_amount') $(echo "$campaign_data" | jq -r '.currency')"
    echo "Contributors: $(echo "$campaign_data" | jq -r '.contributors')"
    
    local goal=$(echo "$campaign_data" | jq -r '.goal_amount')
    local raised=$(echo "$campaign_data" | jq -r '.raised_amount')
    local percentage=$(echo "scale=2; ($raised / $goal) * 100" | bc 2>/dev/null || echo "0")
    
    echo "Progress: ${percentage}%"
    echo "Created: $(echo "$campaign_data" | jq -r '.created_at')"
    
    local end_date=$(echo "$campaign_data" | jq -r '.end_date')
    if [[ -n "$end_date" ]] && [[ "$end_date" != "null" ]]; then
        echo "End Date: $end_date"
    fi
}

btcpay::crowdfunding::close_campaign() {
    local campaign_id="${1:-}"
    
    if [[ -z "$campaign_id" ]]; then
        log::error "Campaign ID required"
        echo "Usage: resource-btcpay crowdfunding close <campaign-id>"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    log::info "Closing campaign: $campaign_id"
    
    # Update campaign status to closed
    jq '.status = "closed" | .closed_at = "'$(date -Iseconds)'"' \
        "$campaign_file" > "${campaign_file}.tmp" && mv "${campaign_file}.tmp" "$campaign_file"
    
    log::success "Campaign closed successfully"
    
    # Show final status
    btcpay::crowdfunding::status "$campaign_id"
}

btcpay::crowdfunding::export_campaign() {
    local campaign_id="${1:-}"
    local format="${2:-json}"
    
    if [[ -z "$campaign_id" ]]; then
        log::error "Campaign ID required"
        echo "Usage: resource-btcpay crowdfunding export <campaign-id> [format]"
        echo "Formats: json (default), csv"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    local export_dir="${DATA_DIR}/crowdfunding/exports"
    mkdir -p "$export_dir"
    
    local export_file="${export_dir}/${campaign_id}_$(date +%Y%m%d_%H%M%S).${format}"
    
    if [[ "$format" == "csv" ]]; then
        log::info "Exporting campaign to CSV: $export_file"
        
        # Create CSV header
        echo "Contribution ID,Amount,Contributor,Message,Date" > "$export_file"
        
        # Export contributions as CSV
        jq -r '.contributions[] | [.id, .amount, .contributor, .message, .date] | @csv' \
            "$campaign_file" >> "$export_file"
    else
        log::info "Exporting campaign to JSON: $export_file"
        cp "$campaign_file" "$export_file"
    fi
    
    log::success "Campaign exported to: $export_file"
    echo "Export file: $export_file"
}

btcpay::crowdfunding::generate_widget() {
    local campaign_id="${1:-}"
    
    if [[ -z "$campaign_id" ]]; then
        log::error "Campaign ID required"
        echo "Usage: resource-btcpay crowdfunding widget <campaign-id>"
        return 1
    fi
    
    local campaign_file="${DATA_DIR}/crowdfunding/campaigns/${campaign_id}.json"
    
    if [[ ! -f "$campaign_file" ]]; then
        log::error "Campaign not found: $campaign_id"
        return 1
    fi
    
    log::info "Generating embeddable widget for campaign: $campaign_id"
    
    local campaign_data=$(cat "$campaign_file")
    local title=$(echo "$campaign_data" | jq -r '.title')
    local goal=$(echo "$campaign_data" | jq -r '.goal_amount')
    local raised=$(echo "$campaign_data" | jq -r '.raised_amount')
    local currency=$(echo "$campaign_data" | jq -r '.currency')
    local percentage=$(echo "scale=2; ($raised / $goal) * 100" | bc 2>/dev/null || echo "0")
    
    local widget_dir="${DATA_DIR}/crowdfunding/widgets"
    mkdir -p "$widget_dir"
    
    local widget_file="${widget_dir}/${campaign_id}_widget.html"
    
    cat > "$widget_file" <<'EOF'
<!DOCTYPE html>
<html>
<head>
    <style>
        .crowdfund-widget {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 20px;
            max-width: 400px;
            margin: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .crowdfund-title {
            font-size: 24px;
            font-weight: bold;
            margin-bottom: 15px;
        }
        .crowdfund-progress {
            background: rgba(255, 255, 255, 0.3);
            border-radius: 10px;
            height: 30px;
            position: relative;
            overflow: hidden;
            margin: 15px 0;
        }
        .crowdfund-progress-bar {
            background: white;
            height: 100%;
            border-radius: 10px;
            transition: width 0.3s ease;
        }
        .crowdfund-stats {
            display: flex;
            justify-content: space-between;
            margin: 15px 0;
        }
        .crowdfund-stat {
            text-align: center;
        }
        .crowdfund-stat-value {
            font-size: 20px;
            font-weight: bold;
        }
        .crowdfund-stat-label {
            font-size: 12px;
            opacity: 0.9;
        }
        .crowdfund-button {
            background: white;
            color: #667eea;
            border: none;
            padding: 12px 24px;
            border-radius: 5px;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            width: 100%;
            margin-top: 15px;
        }
        .crowdfund-button:hover {
            opacity: 0.9;
        }
    </style>
</head>
<body>
    <div class="crowdfund-widget">
        <div class="crowdfund-title">CAMPAIGN_TITLE</div>
        <div class="crowdfund-progress">
            <div class="crowdfund-progress-bar" style="width: PROGRESS_PERCENTAGE%;"></div>
        </div>
        <div class="crowdfund-stats">
            <div class="crowdfund-stat">
                <div class="crowdfund-stat-value">RAISED_AMOUNT CURRENCY</div>
                <div class="crowdfund-stat-label">Raised</div>
            </div>
            <div class="crowdfund-stat">
                <div class="crowdfund-stat-value">PROGRESS_PERCENTAGE%</div>
                <div class="crowdfund-stat-label">Progress</div>
            </div>
            <div class="crowdfund-stat">
                <div class="crowdfund-stat-value">GOAL_AMOUNT CURRENCY</div>
                <div class="crowdfund-stat-label">Goal</div>
            </div>
        </div>
        <button class="crowdfund-button" onclick="window.open('http://localhost:23000/campaign/CAMPAIGN_ID', '_blank')">
            Contribute Now
        </button>
    </div>
</body>
</html>
EOF
    
    # Replace placeholders with actual values
    sed -i "s/CAMPAIGN_TITLE/$title/g" "$widget_file"
    sed -i "s/CAMPAIGN_ID/$campaign_id/g" "$widget_file"
    sed -i "s/GOAL_AMOUNT/$goal/g" "$widget_file"
    sed -i "s/RAISED_AMOUNT/$raised/g" "$widget_file"
    sed -i "s/CURRENCY/$currency/g" "$widget_file"
    sed -i "s/PROGRESS_PERCENTAGE/$percentage/g" "$widget_file"
    
    log::success "Widget generated: $widget_file"
    echo ""
    echo "Embed code:"
    echo "<iframe src=\"file://$widget_file\" width=\"450\" height=\"350\" frameborder=\"0\"></iframe>"
}