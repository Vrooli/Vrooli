#!/usr/bin/env bash
# BTCPay Point of Sale System

# POS configuration management
btcpay::pos::configure() {
    log::info "Configuring Point of Sale system..."
    
    local store_name="${1:-Default Store}"
    local currency="${2:-BTC}"
    local items_file="${3:-}"
    
    # Create POS configuration
    cat > "${BTCPAY_CONFIG_DIR}/pos-config.json" <<EOF
{
    "enabled": true,
    "storeName": "${store_name}",
    "defaultCurrency": "${currency}",
    "showPoweredBy": true,
    "requiresRefundEmail": false,
    "checkoutType": "V2",
    "customCSS": "",
    "embeddedCSS": "",
    "notificationUrl": "",
    "redirectUrl": "",
    "defaultPaymentMethod": "BTC",
    "requiresRefundEmail": false,
    "showRecommendedFee": true,
    "recommendedFeeBlockTarget": 1,
    "defaultLang": "en",
    "showPayInWalletButton": true,
    "showStoreHeader": true,
    "productInformation": "name_and_price"
}
EOF
    
    log::success "POS configuration saved"
    
    # If items file provided, import it
    if [[ -n "$items_file" && -f "$items_file" ]]; then
        btcpay::pos::import_items "$items_file"
    fi
    
    return 0
}

btcpay::pos::add_item() {
    local name="${1:-}"
    local price="${2:-}"
    local currency="${3:-BTC}"
    local description="${4:-}"
    local image="${5:-}"
    
    if [[ -z "$name" || -z "$price" ]]; then
        log::error "Usage: btcpay::pos::add_item <name> <price> [currency] [description] [image_url]"
        return 1
    fi
    
    # Load existing items or create new array
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    local items="[]"
    
    if [[ -f "$items_file" ]]; then
        items=$(cat "$items_file")
    fi
    
    # Generate item ID
    local item_id=$(date +%s%N | sha256sum | head -c 8)
    
    # Add new item
    local new_item=$(jq -n \
        --arg id "$item_id" \
        --arg name "$name" \
        --arg price "$price" \
        --arg currency "$currency" \
        --arg desc "$description" \
        --arg img "$image" \
        '{
            id: $id,
            name: $name,
            price: ($price | tonumber),
            currency: $currency,
            description: $desc,
            image: $img,
            enabled: true,
            inventory: null
        }')
    
    # Append to items array
    echo "$items" | jq ". += [$new_item]" > "$items_file"
    
    log::success "Item added: $name ($price $currency)"
    echo "Item ID: $item_id"
    
    return 0
}

btcpay::pos::list_items() {
    log::info "Point of Sale Items"
    log::info "==================="
    
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    
    if [[ ! -f "$items_file" ]]; then
        log::warning "No items configured"
        log::info "Add items with: resource-btcpay pos add-item <name> <price>"
        return 0
    fi
    
    # Display items in formatted table
    echo ""
    printf "%-10s %-30s %-10s %-10s %s\n" "ID" "Name" "Price" "Currency" "Status"
    printf "%-10s %-30s %-10s %-10s %s\n" "----------" "------------------------------" "----------" "----------" "----------"
    
    jq -r '.[] | "\(.id)\t\(.name)\t\(.price)\t\(.currency)\t\(if .enabled then "✓ Enabled" else "✗ Disabled" end)"' "$items_file" | \
    while IFS=$'\t' read -r id name price currency status; do
        printf "%-10s %-30s %-10s %-10s %s\n" "$id" "${name:0:30}" "$price" "$currency" "$status"
    done
    
    echo ""
    local total_items=$(jq '. | length' "$items_file")
    local enabled_items=$(jq '[.[] | select(.enabled == true)] | length' "$items_file")
    echo "Total items: $total_items (Enabled: $enabled_items)"
    
    return 0
}

btcpay::pos::remove_item() {
    local item_id="${1:-}"
    
    if [[ -z "$item_id" ]]; then
        log::error "Usage: btcpay::pos::remove_item <item_id>"
        return 1
    fi
    
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    
    if [[ ! -f "$items_file" ]]; then
        log::error "No items configured"
        return 1
    fi
    
    # Check if item exists
    if ! jq -e ".[] | select(.id == \"$item_id\")" "$items_file" &>/dev/null; then
        log::error "Item not found: $item_id"
        return 1
    fi
    
    # Remove item
    local tmp_file=$(mktemp)
    jq "map(select(.id != \"$item_id\"))" "$items_file" > "$tmp_file"
    mv "$tmp_file" "$items_file"
    
    log::success "Item removed: $item_id"
    return 0
}

btcpay::pos::update_item() {
    local item_id="${1:-}"
    local field="${2:-}"
    local value="${3:-}"
    
    if [[ -z "$item_id" || -z "$field" || -z "$value" ]]; then
        log::error "Usage: btcpay::pos::update_item <item_id> <field> <value>"
        log::info "Fields: name, price, currency, description, image, enabled"
        return 1
    fi
    
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    
    if [[ ! -f "$items_file" ]]; then
        log::error "No items configured"
        return 1
    fi
    
    # Check if item exists
    if ! jq -e ".[] | select(.id == \"$item_id\")" "$items_file" &>/dev/null; then
        log::error "Item not found: $item_id"
        return 1
    fi
    
    # Update item field
    local tmp_file=$(mktemp)
    
    case "$field" in
        price)
            jq "map(if .id == \"$item_id\" then .price = ($value | tonumber) else . end)" "$items_file" > "$tmp_file"
            ;;
        enabled)
            local bool_value="false"
            [[ "$value" == "true" || "$value" == "1" ]] && bool_value="true"
            jq "map(if .id == \"$item_id\" then .enabled = $bool_value else . end)" "$items_file" > "$tmp_file"
            ;;
        *)
            jq "map(if .id == \"$item_id\" then .$field = \"$value\" else . end)" "$items_file" > "$tmp_file"
            ;;
    esac
    
    mv "$tmp_file" "$items_file"
    
    log::success "Item updated: $item_id ($field = $value)"
    return 0
}

btcpay::pos::import_items() {
    local import_file="${1:-}"
    
    if [[ -z "$import_file" || ! -f "$import_file" ]]; then
        log::error "Usage: btcpay::pos::import_items <csv_file>"
        log::info "CSV format: name,price,currency,description"
        return 1
    fi
    
    log::info "Importing items from $import_file..."
    
    local items_imported=0
    local line_num=0
    
    # Skip header line if present
    while IFS=, read -r name price currency description; do
        ((line_num++))
        
        # Skip header
        if [[ $line_num -eq 1 && "$name" == "name" ]]; then
            continue
        fi
        
        # Skip empty lines
        [[ -z "$name" ]] && continue
        
        # Add item
        if btcpay::pos::add_item "$name" "$price" "${currency:-BTC}" "$description" &>/dev/null; then
            ((items_imported++))
        else
            log::warning "Failed to import item on line $line_num: $name"
        fi
    done < "$import_file"
    
    log::success "Imported $items_imported items"
    return 0
}

btcpay::pos::generate_terminal() {
    log::info "Generating POS terminal interface..."
    
    local terminal_file="${BTCPAY_CONFIG_DIR}/pos-terminal.html"
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    local pos_config="${BTCPAY_CONFIG_DIR}/pos-config.json"
    
    # Check prerequisites
    if [[ ! -f "$items_file" ]]; then
        log::error "No items configured. Add items first."
        return 1
    fi
    
    if [[ ! -f "$pos_config" ]]; then
        log::warning "POS not configured, using defaults..."
        btcpay::pos::configure
    fi
    
    # Get configuration
    local store_name=$(jq -r '.storeName // "Default Store"' "$pos_config")
    local default_currency=$(jq -r '.defaultCurrency // "BTC"' "$pos_config")
    
    # Generate HTML terminal
    cat > "$terminal_file" <<'EOF'
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>BTCPay POS Terminal</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f5f5; }
        .header { background: #2c3e50; color: white; padding: 1rem; text-align: center; }
        .container { max-width: 1200px; margin: 0 auto; padding: 1rem; }
        .items-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 1rem; margin: 2rem 0; }
        .item-card { background: white; border-radius: 8px; padding: 1rem; box-shadow: 0 2px 4px rgba(0,0,0,0.1); cursor: pointer; transition: transform 0.2s; }
        .item-card:hover { transform: translateY(-2px); box-shadow: 0 4px 8px rgba(0,0,0,0.15); }
        .item-name { font-weight: bold; margin-bottom: 0.5rem; }
        .item-price { color: #27ae60; font-size: 1.2rem; }
        .cart { position: fixed; right: 0; top: 0; width: 300px; height: 100vh; background: white; box-shadow: -2px 0 8px rgba(0,0,0,0.1); padding: 1rem; overflow-y: auto; }
        .cart-header { font-size: 1.2rem; font-weight: bold; margin-bottom: 1rem; padding-bottom: 1rem; border-bottom: 2px solid #ecf0f1; }
        .cart-item { display: flex; justify-content: space-between; padding: 0.5rem 0; border-bottom: 1px solid #ecf0f1; }
        .cart-total { font-size: 1.5rem; font-weight: bold; margin-top: 1rem; padding-top: 1rem; border-top: 2px solid #2c3e50; }
        .checkout-btn { width: 100%; padding: 1rem; background: #27ae60; color: white; border: none; border-radius: 4px; font-size: 1.1rem; cursor: pointer; margin-top: 1rem; }
        .checkout-btn:hover { background: #229954; }
        .checkout-btn:disabled { background: #95a5a6; cursor: not-allowed; }
        @media (max-width: 768px) {
            .cart { width: 100%; position: relative; height: auto; }
            .items-grid { grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); }
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>STORE_NAME_PLACEHOLDER</h1>
        <p>Point of Sale Terminal</p>
    </div>
    
    <div class="container" style="margin-right: 320px;">
        <div class="items-grid" id="itemsGrid"></div>
    </div>
    
    <div class="cart">
        <div class="cart-header">Shopping Cart</div>
        <div id="cartItems"></div>
        <div class="cart-total">Total: <span id="cartTotal">0.00</span> <span id="currency">CURRENCY_PLACEHOLDER</span></div>
        <button class="checkout-btn" id="checkoutBtn" onclick="checkout()" disabled>Checkout</button>
    </div>
    
    <script>
        const items = ITEMS_PLACEHOLDER;
        const currency = 'CURRENCY_PLACEHOLDER';
        let cart = [];
        
        function renderItems() {
            const grid = document.getElementById('itemsGrid');
            grid.innerHTML = items.filter(item => item.enabled).map(item => `
                <div class="item-card" onclick="addToCart('${item.id}')">
                    ${item.image ? `<img src="${item.image}" style="width:100%; height:150px; object-fit:cover; border-radius:4px; margin-bottom:0.5rem;">` : ''}
                    <div class="item-name">${item.name}</div>
                    <div class="item-price">${item.price} ${item.currency}</div>
                    ${item.description ? `<div style="color:#7f8c8d; font-size:0.9rem; margin-top:0.5rem;">${item.description}</div>` : ''}
                </div>
            `).join('');
        }
        
        function addToCart(itemId) {
            const item = items.find(i => i.id === itemId);
            const existingItem = cart.find(i => i.id === itemId);
            
            if (existingItem) {
                existingItem.quantity++;
            } else {
                cart.push({...item, quantity: 1});
            }
            
            updateCart();
        }
        
        function removeFromCart(itemId) {
            cart = cart.filter(item => item.id !== itemId);
            updateCart();
        }
        
        function updateCart() {
            const cartItems = document.getElementById('cartItems');
            const cartTotal = document.getElementById('cartTotal');
            const checkoutBtn = document.getElementById('checkoutBtn');
            
            if (cart.length === 0) {
                cartItems.innerHTML = '<p style="color:#95a5a6; text-align:center; padding:2rem;">Cart is empty</p>';
                cartTotal.textContent = '0.00';
                checkoutBtn.disabled = true;
            } else {
                cartItems.innerHTML = cart.map(item => `
                    <div class="cart-item">
                        <div>
                            <div>${item.name}</div>
                            <div style="color:#7f8c8d; font-size:0.9rem;">${item.quantity} × ${item.price} ${item.currency}</div>
                        </div>
                        <button onclick="removeFromCart('${item.id}')" style="background:#e74c3c; color:white; border:none; padding:0.25rem 0.5rem; border-radius:4px; cursor:pointer;">×</button>
                    </div>
                `).join('');
                
                const total = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
                cartTotal.textContent = total.toFixed(2);
                checkoutBtn.disabled = false;
            }
        }
        
        function checkout() {
            const total = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
            alert(`Checkout total: ${total.toFixed(2)} ${currency}\n\nIn a real implementation, this would create an invoice via BTCPay API.`);
            cart = [];
            updateCart();
        }
        
        // Initialize
        renderItems();
        updateCart();
    </script>
</body>
</html>
EOF
    
    # Replace placeholders using a temporary file
    local items_json=$(cat "$items_file" | jq -c '.')
    local tmp_file=$(mktemp)
    
    # Use awk for safer replacement
    awk -v store="$store_name" -v currency="$default_currency" -v items="$items_json" '
        { 
            gsub(/STORE_NAME_PLACEHOLDER/, store)
            gsub(/CURRENCY_PLACEHOLDER/, currency)
            gsub(/ITEMS_PLACEHOLDER/, items)
            print
        }
    ' "$terminal_file" > "$tmp_file"
    
    mv "$tmp_file" "$terminal_file"
    
    log::success "POS terminal generated: $terminal_file"
    log::info "Open in browser: file://${terminal_file}"
    
    return 0
}

btcpay::pos::status() {
    log::info "Point of Sale Status"
    log::info "===================="
    
    local pos_config="${BTCPAY_CONFIG_DIR}/pos-config.json"
    local items_file="${BTCPAY_CONFIG_DIR}/pos-items.json"
    
    # Check configuration
    if [[ -f "$pos_config" ]]; then
        local enabled=$(jq -r '.enabled // false' "$pos_config")
        local store_name=$(jq -r '.storeName // "Not configured"' "$pos_config")
        local currency=$(jq -r '.defaultCurrency // "BTC"' "$pos_config")
        
        echo "Configuration:"
        echo "  Status: $([ "$enabled" == "true" ] && echo "✓ Enabled" || echo "✗ Disabled")"
        echo "  Store Name: $store_name"
        echo "  Default Currency: $currency"
    else
        echo "Configuration: ✗ Not configured"
        echo "  Run: resource-btcpay pos configure"
    fi
    
    echo ""
    
    # Check items
    if [[ -f "$items_file" ]]; then
        local total_items=$(jq '. | length' "$items_file")
        local enabled_items=$(jq '[.[] | select(.enabled == true)] | length' "$items_file")
        
        echo "Inventory:"
        echo "  Total Items: $total_items"
        echo "  Enabled Items: $enabled_items"
    else
        echo "Inventory: ✗ No items configured"
        echo "  Run: resource-btcpay pos add-item"
    fi
    
    echo ""
    
    # Check terminal
    local terminal_file="${BTCPAY_CONFIG_DIR}/pos-terminal.html"
    if [[ -f "$terminal_file" ]]; then
        echo "Terminal: ✓ Generated"
        echo "  Location: $terminal_file"
    else
        echo "Terminal: ✗ Not generated"
        echo "  Run: resource-btcpay pos generate"
    fi
    
    return 0
}

# CLI command handler
btcpay::pos() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        configure)
            btcpay::pos::configure "$@"
            ;;
        add-item)
            btcpay::pos::add_item "$@"
            ;;
        list-items|list)
            btcpay::pos::list_items
            ;;
        remove-item|remove)
            btcpay::pos::remove_item "$@"
            ;;
        update-item|update)
            btcpay::pos::update_item "$@"
            ;;
        import)
            btcpay::pos::import_items "$@"
            ;;
        generate)
            btcpay::pos::generate_terminal
            ;;
        status)
            btcpay::pos::status
            ;;
        --help|-h|help)
            cat <<EOF
Point of Sale Management Commands:

  configure [name] [currency]        Configure POS system
  add-item <name> <price> [currency] Add item to inventory
  list-items                         List all POS items
  remove-item <item_id>              Remove item from inventory
  update-item <id> <field> <value>   Update item property
  import <csv_file>                  Import items from CSV
  generate                           Generate POS terminal interface
  status                             Show POS configuration status

Examples:
  resource-btcpay pos configure "My Store" BTC
  resource-btcpay pos add-item "Coffee" 5.99 USD "Fresh roasted coffee"
  resource-btcpay pos list-items
  resource-btcpay pos generate
EOF
            ;;
        *)
            log::error "Unknown POS subcommand: $subcommand"
            log::info "Use 'resource-btcpay pos --help' for usage"
            return 1
            ;;
    esac
}