#!/usr/bin/env bash

#######################################
# N8N Credentials Management via UI Automation
#
# Provides UI-based credential management when the API
# is unavailable or when dealing with complex auth flows.
#######################################

#######################################
# Add credentials via UI automation
# Arguments:
#   $1 - Credential type (e.g., httpHeaderAuth, oauth2, etc.)
#   $2 - Credential name
#   $3 - Credential data (JSON)
# Returns:
#   0 on success, 1 on failure
#######################################
n8n::add_credentials_ui() {
    local cred_type="${1:?Credential type required}"
    local cred_name="${2:-$cred_type-$(date +%s)}"
    local cred_data="${3:-{}}"
    
    log::header "🔐 Adding N8N Credentials via UI: $cred_name"
    
    local function_code='
    export default async ({ page }) => {
        const credType = "'$cred_type'";
        const credName = "'$cred_name'";
        const credData = '$cred_data';
        
        try {
            // Navigate to credentials page
            await page.goto("'${N8N_URL}'/credentials", {
                waitUntil: "networkidle2",
                timeout: 30000
            });
            
            // Click add credential button
            await page.click("[data-test-id=\"resources-list-add\"], .add-button");
            await page.waitForTimeout(1000);
            
            // Search for credential type
            await page.type(".search-input, [data-test-id=\"credential-search\"]", credType);
            await page.waitForTimeout(500);
            
            // Click on the credential type
            await page.click(`[data-test-id*="${credType}"], [title*="${credType}"]`);
            await page.waitForTimeout(1000);
            
            // Fill in credential name
            const nameInput = await page.$("[data-test-id=\"credential-name\"], input[placeholder*=\"name\"]");
            if (nameInput) {
                await nameInput.click({ clickCount: 3 });
                await nameInput.type(credName);
            }
            
            // Fill credential fields based on type
            if (credType === "httpHeaderAuth" && credData.headerName && credData.headerValue) {
                await page.type("[data-test-id=\"parameter-input-name\"]", credData.headerName);
                await page.type("[data-test-id=\"parameter-input-value\"]", credData.headerValue);
            }
            
            // Save credential
            await page.click("[data-test-id=\"credential-save-button\"], .save-button");
            await page.waitForTimeout(2000);
            
            // Check for success
            const success = await page.evaluate(() => {
                const notifications = document.querySelectorAll(".el-notification");
                for (const notif of notifications) {
                    if (notif.textContent.includes("saved") || notif.textContent.includes("created")) {
                        return true;
                    }
                }
                return false;
            });
            
            return {
                success: success,
                credentialName: credName,
                credentialType: credType,
                timestamp: new Date().toISOString()
            };
            
        } catch (error) {
            return {
                success: false,
                error: error.message,
                timestamp: new Date().toISOString()
            };
        }
    }'
    
    # Execute via adapter framework
    adapter::execute_browser_function "$function_code" "30000" "true" "n8n_add_credential"
}

#######################################
# List credentials via UI scraping
# Returns:
#   JSON array of credentials
#######################################
n8n::list_credentials_ui() {
    log::header "📋 Listing N8N Credentials via UI"
    
    local function_code='
    export default async ({ page }) => {
        try {
            // Navigate to credentials page
            await page.goto("'${N8N_URL}'/credentials", {
                waitUntil: "networkidle2",
                timeout: 30000
            });
            
            // Wait for credentials list
            await page.waitForSelector("[data-test-id*=\"credential\"], .credential-card", {
                timeout: 10000
            });
            
            // Extract credential information
            const credentials = await page.evaluate(() => {
                const items = [];
                const cards = document.querySelectorAll("[data-test-id*=\"credential\"], .credential-card");
                
                cards.forEach(card => {
                    const name = card.querySelector(".credential-name, .name")?.textContent?.trim();
                    const type = card.querySelector(".credential-type, .type")?.textContent?.trim();
                    const id = card.getAttribute("data-credential-id");
                    
                    if (name || type) {
                        items.push({
                            id: id || "unknown",
                            name: name || "Unnamed",
                            type: type || "Unknown Type"
                        });
                    }
                });
                
                return items;
            });
            
            return {
                success: true,
                credentials: credentials,
                count: credentials.length,
                timestamp: new Date().toISOString()
            };
            
        } catch (error) {
            return {
                success: false,
                error: error.message,
                credentials: [],
                timestamp: new Date().toISOString()
            };
        }
    }'
    
    # Execute via adapter framework
    local result=$(adapter::execute_browser_function "$function_code" "30000" "true" "n8n_list_credentials")
    
    if [[ $? -eq 0 ]]; then
        echo "$result" | jq '.credentials'
    else
        log::error "Failed to list credentials"
        return 1
    fi
}

# Export functions
export -f n8n::add_credentials_ui
export -f n8n::list_credentials_ui