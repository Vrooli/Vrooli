#!/usr/bin/env bash
################################################################################
# BTCPay Payment Button Module - Button Generation and Management
#
# Handles payment button creation for easy website integration.
################################################################################

set -euo pipefail

# Set data directory if not already set
DATA_DIR="${DATA_DIR:-${APP_ROOT}/data/resources/btcpay}"

# ==============================================================================
# Payment Button Management
# ==============================================================================

btcpay::paybutton::create() {
    local amount="${1:-}"
    local currency="${2:-BTC}"
    local description="${3:-Payment}"
    local button_text="${4:-Pay Now}"
    local store_id="${5:-default}"
    
    if [[ -z "$amount" ]]; then
        log::error "Amount is required"
        echo "Usage: resource-btcpay paybutton create <amount> [currency] [description] [button-text] [store-id]"
        return 1
    fi
    
    log::info "Creating payment button: $amount $currency"
    
    # Generate button ID
    local button_id="btn_$(date +%s)_$(openssl rand -hex 4)"
    local buttons_dir="${DATA_DIR}/paybuttons"
    mkdir -p "$buttons_dir"
    
    # Create button configuration
    cat > "${buttons_dir}/${button_id}.json" <<EOF
{
    "id": "$button_id",
    "amount": $amount,
    "currency": "$currency",
    "description": "$description",
    "button_text": "$button_text",
    "store_id": "$store_id",
    "created_at": "$(date -Iseconds)",
    "click_count": 0,
    "payment_count": 0,
    "total_received": 0
}
EOF
    
    # Generate button HTML
    btcpay::paybutton::_generate_html "$button_id"
    
    log::success "Payment button created: $button_id"
    echo "Button ID: $button_id"
    echo "Amount: $amount $currency"
    echo "Description: $description"
}

btcpay::paybutton::_generate_html() {
    local button_id="${1}"
    local buttons_dir="${DATA_DIR}/paybuttons"
    local button_file="${buttons_dir}/${button_id}.json"
    
    if [[ ! -f "$button_file" ]]; then
        return 1
    fi
    
    local button_data=$(cat "$button_file")
    local amount=$(echo "$button_data" | jq -r '.amount')
    local currency=$(echo "$button_data" | jq -r '.currency')
    local description=$(echo "$button_data" | jq -r '.description')
    local button_text=$(echo "$button_data" | jq -r '.button_text')
    
    local html_file="${buttons_dir}/${button_id}.html"
    
    cat > "$html_file" <<'EOF'
<!DOCTYPE html>
<html>
<head>
    <style>
        .btcpay-button {
            display: inline-block;
            padding: 12px 24px;
            background: linear-gradient(135deg, #f5af19 0%, #f12711 100%);
            color: white;
            text-decoration: none;
            border-radius: 5px;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            font-size: 16px;
            font-weight: bold;
            border: none;
            cursor: pointer;
            transition: all 0.3s ease;
            box-shadow: 0 4px 15px rgba(241, 39, 17, 0.3);
        }
        .btcpay-button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(241, 39, 17, 0.4);
        }
        .btcpay-button:active {
            transform: translateY(0);
        }
        .btcpay-amount {
            font-size: 20px;
            margin-right: 8px;
        }
    </style>
    <script>
        function initiateBTCPayment(buttonId, amount, currency, description) {
            // Track button click
            fetch('http://localhost:23000/api/button/click/' + buttonId, { method: 'POST' });
            
            // Create invoice and redirect
            const invoiceData = {
                amount: amount,
                currency: currency,
                description: description
            };
            
            fetch('http://localhost:23000/api/v1/stores/default/invoices', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(invoiceData)
            })
            .then(response => response.json())
            .then(data => {
                if (data.checkoutLink) {
                    window.open(data.checkoutLink, '_blank');
                } else {
                    alert('Payment initiation failed. Please try again.');
                }
            })
            .catch(error => {
                console.error('Error:', error);
                alert('Payment system unavailable. Please try again later.');
            });
        }
    </script>
</head>
<body>
    <button class="btcpay-button" onclick="initiateBTCPayment('BUTTON_ID', AMOUNT, 'CURRENCY', 'DESCRIPTION')">
        <span class="btcpay-amount">AMOUNT CURRENCY</span>
        BUTTON_TEXT
    </button>
</body>
</html>
EOF
    
    # Replace placeholders
    sed -i "s/BUTTON_ID/$button_id/g" "$html_file"
    sed -i "s/AMOUNT/$amount/g" "$html_file"
    sed -i "s/CURRENCY/$currency/g" "$html_file"
    sed -i "s/DESCRIPTION/$description/g" "$html_file"
    sed -i "s/BUTTON_TEXT/$button_text/g" "$html_file"
}

btcpay::paybutton::list() {
    local buttons_dir="${DATA_DIR}/paybuttons"
    
    if [[ ! -d "$buttons_dir" ]]; then
        log::info "No payment buttons found"
        return 0
    fi
    
    log::info "Listing payment buttons:"
    echo ""
    echo "ID | Amount | Currency | Description | Clicks | Payments"
    echo "---|--------|----------|-------------|--------|----------"
    
    for button_file in "${buttons_dir}"/*.json; do
        if [[ -f "$button_file" ]]; then
            local button_data=$(cat "$button_file")
            local id=$(echo "$button_data" | jq -r '.id')
            local amount=$(echo "$button_data" | jq -r '.amount')
            local currency=$(echo "$button_data" | jq -r '.currency')
            local description=$(echo "$button_data" | jq -r '.description')
            local clicks=$(echo "$button_data" | jq -r '.click_count')
            local payments=$(echo "$button_data" | jq -r '.payment_count')
            
            echo "$id | $amount | $currency | $description | $clicks | $payments"
        fi
    done
}

btcpay::paybutton::get_code() {
    local button_id="${1:-}"
    local format="${2:-html}"
    
    if [[ -z "$button_id" ]]; then
        log::error "Button ID required"
        echo "Usage: resource-btcpay paybutton get-code <button-id> [format]"
        echo "Formats: html (default), iframe, link, markdown"
        return 1
    fi
    
    local button_file="${DATA_DIR}/paybuttons/${button_id}.json"
    local html_file="${DATA_DIR}/paybuttons/${button_id}.html"
    
    if [[ ! -f "$button_file" ]]; then
        log::error "Button not found: $button_id"
        return 1
    fi
    
    if [[ ! -f "$html_file" ]]; then
        btcpay::paybutton::_generate_html "$button_id"
    fi
    
    local button_data=$(cat "$button_file")
    local amount=$(echo "$button_data" | jq -r '.amount')
    local currency=$(echo "$button_data" | jq -r '.currency')
    local description=$(echo "$button_data" | jq -r '.description')
    local button_text=$(echo "$button_data" | jq -r '.button_text')
    
    log::info "Embed code for button: $button_id"
    echo ""
    
    case "$format" in
        iframe)
            echo "<iframe src=\"file://$html_file\" width=\"200\" height=\"60\" frameborder=\"0\"></iframe>"
            ;;
        link)
            echo "<a href=\"http://localhost:23000/pay/$button_id\" target=\"_blank\">$button_text</a>"
            ;;
        markdown)
            echo "[$button_text](http://localhost:23000/pay/$button_id)"
            ;;
        html|*)
            cat "$html_file" | grep -A 20 '<button' | head -n 5
            ;;
    esac
}

btcpay::paybutton::update() {
    local button_id="${1:-}"
    local property="${2:-}"
    local value="${3:-}"
    
    if [[ -z "$button_id" ]] || [[ -z "$property" ]] || [[ -z "$value" ]]; then
        log::error "Button ID, property, and value are required"
        echo "Usage: resource-btcpay paybutton update <button-id> <property> <value>"
        echo "Properties: amount, currency, description, button_text"
        return 1
    fi
    
    local button_file="${DATA_DIR}/paybuttons/${button_id}.json"
    
    if [[ ! -f "$button_file" ]]; then
        log::error "Button not found: $button_id"
        return 1
    fi
    
    log::info "Updating button $button_id: $property = $value"
    
    # Update the specific property
    case "$property" in
        amount|click_count|payment_count|total_received)
            jq ".$property = $value" "$button_file" > "${button_file}.tmp" && mv "${button_file}.tmp" "$button_file"
            ;;
        *)
            jq ".$property = \"$value\"" "$button_file" > "${button_file}.tmp" && mv "${button_file}.tmp" "$button_file"
            ;;
    esac
    
    # Regenerate HTML with updated values
    btcpay::paybutton::_generate_html "$button_id"
    
    log::success "Button updated successfully"
}

btcpay::paybutton::delete() {
    local button_id="${1:-}"
    
    if [[ -z "$button_id" ]]; then
        log::error "Button ID required"
        echo "Usage: resource-btcpay paybutton delete <button-id>"
        return 1
    fi
    
    local button_file="${DATA_DIR}/paybuttons/${button_id}.json"
    local html_file="${DATA_DIR}/paybuttons/${button_id}.html"
    
    if [[ ! -f "$button_file" ]]; then
        log::error "Button not found: $button_id"
        return 1
    fi
    
    log::info "Deleting button: $button_id"
    
    rm -f "$button_file" "$html_file"
    
    log::success "Button deleted successfully"
}

btcpay::paybutton::stats() {
    local button_id="${1:-}"
    
    if [[ -z "$button_id" ]]; then
        log::error "Button ID required"
        echo "Usage: resource-btcpay paybutton stats <button-id>"
        return 1
    fi
    
    local button_file="${DATA_DIR}/paybuttons/${button_id}.json"
    
    if [[ ! -f "$button_file" ]]; then
        log::error "Button not found: $button_id"
        return 1
    fi
    
    log::info "Payment Button Statistics: $button_id"
    echo ""
    
    local button_data=$(cat "$button_file")
    
    echo "Button ID: $(echo "$button_data" | jq -r '.id')"
    echo "Amount: $(echo "$button_data" | jq -r '.amount') $(echo "$button_data" | jq -r '.currency')"
    echo "Description: $(echo "$button_data" | jq -r '.description')"
    echo "Button Text: $(echo "$button_data" | jq -r '.button_text')"
    echo "Created: $(echo "$button_data" | jq -r '.created_at')"
    echo ""
    echo "Performance:"
    echo "  Click Count: $(echo "$button_data" | jq -r '.click_count')"
    echo "  Payment Count: $(echo "$button_data" | jq -r '.payment_count')"
    echo "  Total Received: $(echo "$button_data" | jq -r '.total_received') $(echo "$button_data" | jq -r '.currency')"
    
    local clicks=$(echo "$button_data" | jq -r '.click_count')
    local payments=$(echo "$button_data" | jq -r '.payment_count')
    if [[ "$clicks" -gt 0 ]]; then
        local conversion=$(echo "scale=2; ($payments / $clicks) * 100" | bc 2>/dev/null || echo "0")
        echo "  Conversion Rate: ${conversion}%"
    fi
}

btcpay::paybutton::generate_styles() {
    local style="${1:-default}"
    
    log::info "Generating button style: $style"
    
    local styles_dir="${DATA_DIR}/paybuttons/styles"
    mkdir -p "$styles_dir"
    
    local style_file="${styles_dir}/${style}.css"
    
    case "$style" in
        minimal)
            cat > "$style_file" <<'EOF'
.btcpay-button {
    padding: 10px 20px;
    background: #0066cc;
    color: white;
    border: none;
    border-radius: 3px;
    font-size: 14px;
    cursor: pointer;
}
.btcpay-button:hover {
    background: #0052a3;
}
EOF
            ;;
        rounded)
            cat > "$style_file" <<'EOF'
.btcpay-button {
    padding: 14px 28px;
    background: linear-gradient(45deg, #FE6B8B 30%, #FF8E53 90%);
    color: white;
    border: none;
    border-radius: 50px;
    font-size: 16px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 3px 5px 2px rgba(255, 105, 135, .3);
}
.btcpay-button:hover {
    transform: scale(1.05);
}
EOF
            ;;
        dark)
            cat > "$style_file" <<'EOF'
.btcpay-button {
    padding: 12px 24px;
    background: #1a1a1a;
    color: #f5f5f5;
    border: 2px solid #333;
    border-radius: 4px;
    font-size: 16px;
    font-weight: bold;
    cursor: pointer;
    transition: all 0.3s ease;
}
.btcpay-button:hover {
    background: #333;
    border-color: #555;
}
EOF
            ;;
        default|*)
            cat > "$style_file" <<'EOF'
.btcpay-button {
    padding: 12px 24px;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    border: none;
    border-radius: 5px;
    font-size: 16px;
    font-weight: bold;
    cursor: pointer;
    box-shadow: 0 4px 15px rgba(102, 126, 234, 0.4);
}
.btcpay-button:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
}
EOF
            ;;
    esac
    
    log::success "Style generated: $style_file"
    echo "Style file: $style_file"
    echo ""
    echo "Include in your HTML:"
    echo "<link rel=\"stylesheet\" href=\"file://$style_file\">"
}

btcpay::paybutton::bulk_create() {
    local csv_file="${1:-}"
    
    if [[ -z "$csv_file" ]] || [[ ! -f "$csv_file" ]]; then
        log::error "CSV file required and must exist"
        echo "Usage: resource-btcpay paybutton bulk-create <csv-file>"
        echo "CSV Format: amount,currency,description,button_text"
        return 1
    fi
    
    log::info "Creating buttons from CSV: $csv_file"
    
    local count=0
    while IFS=, read -r amount currency description button_text; do
        # Skip header line
        if [[ "$amount" == "amount" ]]; then
            continue
        fi
        
        btcpay::paybutton::create "$amount" "$currency" "$description" "$button_text"
        ((count++))
    done < "$csv_file"
    
    log::success "Created $count payment buttons"
}