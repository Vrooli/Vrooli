#!/bin/bash
# INVESTIGATION_SCRIPT
# NAME: Service Configuration Validator
# DESCRIPTION: Detects systemd services with configuration errors or invalid directives
# CATEGORY: resource-management
# TRIGGERS: service_config_errors, failed_services, systemd_warnings
# OUTPUTS: json
# AUTHOR: claude-agent
# CREATED: 2025-09-18
# LAST_MODIFIED: 2025-09-18
# VERSION: 1.0

set -euo pipefail

# Configuration
SCRIPT_NAME="service-config-validator"
OUTPUT_DIR="../results/$(date +%Y%m%d_%H%M%S)_${SCRIPT_NAME}"
TIMEOUT_CMD="timeout 10"

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Initialize results file
RESULTS_FILE="${OUTPUT_DIR}/results.json"

echo "🔍 Starting Service Configuration Validation..."

# Initialize JSON structure
cat > "${RESULTS_FILE}" << 'EOF'
{
  "investigation": "service-config-validator",
  "timestamp": "",
  "services_with_errors": [],
  "invalid_directives": [],
  "failed_services": [],
  "findings": [],
  "recommendations": [],
  "auto_fixed": []
}
EOF

# Update timestamp
TIMESTAMP=$(date -Iseconds)
jq ".timestamp = \"$TIMESTAMP\"" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"

# Check for services with configuration errors
echo "🔧 Checking for service configuration errors..."
SERVICES_WITH_ERRORS=()
INVALID_DIRECTIVES=()

# Get all service files
for service_file in /etc/systemd/system/*.service /usr/lib/systemd/system/*.service; do
    if [[ -f "$service_file" ]]; then
        SERVICE_NAME=$(basename "$service_file")
        
        # Check if systemctl can parse it without errors
        if ! systemctl show "$SERVICE_NAME" 2>&1 | grep -q "Failed to parse"; then
            continue
        fi
        
        # Check for common invalid directives
        if grep -q "Restart=unless-stopped" "$service_file" 2>/dev/null; then
            SERVICES_WITH_ERRORS+=("{\"service\": \"$SERVICE_NAME\", \"error\": \"Invalid Restart directive: unless-stopped (Docker syntax used in systemd)\"}")
            INVALID_DIRECTIVES+=("\"$SERVICE_NAME: Restart=unless-stopped should be Restart=always or Restart=on-failure\"")
        fi
        
        if grep -q "RestartPolicy=" "$service_file" 2>/dev/null; then
            SERVICES_WITH_ERRORS+=("{\"service\": \"$SERVICE_NAME\", \"error\": \"Invalid directive: RestartPolicy (Docker syntax)\"}")
            INVALID_DIRECTIVES+=("\"$SERVICE_NAME: RestartPolicy should be Restart\"")
        fi
    fi
done

# Check all failed services
echo "📋 Analyzing failed services..."
FAILED_SERVICES=()
while IFS= read -r line; do
    if [[ -n "$line" ]]; then
        SERVICE=$(echo "$line" | awk '{print $2}')
        STATUS=$(systemctl status "$SERVICE" 2>&1 | head -5 | grep -E "(Loaded:|Active:)" | tr '\n' ' ')
        
        # Check if it has config errors
        if systemctl status "$SERVICE" 2>&1 | grep -q "Failed to parse.*specifier"; then
            SERVICES_WITH_ERRORS+=("{\"service\": \"$SERVICE\", \"error\": \"Configuration parsing error detected\"}")
        fi
        
        FAILED_SERVICES+=("{\"name\": \"$SERVICE\", \"status\": \"$STATUS\"}")
    fi
done < <(systemctl list-units --failed --no-pager --no-legend 2>/dev/null || true)

# Update JSON with findings
if [ ${#SERVICES_WITH_ERRORS[@]} -gt 0 ]; then
    ERRORS_JSON=$(printf '%s\n' "${SERVICES_WITH_ERRORS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".services_with_errors = $ERRORS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

if [ ${#INVALID_DIRECTIVES[@]} -gt 0 ]; then
    DIRECTIVES_JSON=$(printf '%s\n' "${INVALID_DIRECTIVES[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".invalid_directives = $DIRECTIVES_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

if [ ${#FAILED_SERVICES[@]} -gt 0 ]; then
    FAILED_JSON=$(printf '%s\n' "${FAILED_SERVICES[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".failed_services = $FAILED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Generate findings and recommendations
echo "💡 Generating findings and recommendations..."
FINDINGS=()
RECOMMENDATIONS=()
AUTO_FIXED=()

ERROR_COUNT=$(jq '.services_with_errors | length' "${RESULTS_FILE}")
if [ "$ERROR_COUNT" -gt 0 ]; then
    FINDINGS+=("\"Found $ERROR_COUNT services with configuration errors\"")
    RECOMMENDATIONS+=("\"Fix service files by replacing Docker-style directives with systemd equivalents\"")
    
    # Auto-fix attempt for common issues
    if [ "$ERROR_COUNT" -le 10 ]; then
        echo "🔧 Attempting to auto-fix service configuration errors..."
        for service_info in $(jq -c '.services_with_errors[]' "${RESULTS_FILE}"); do
            SERVICE_NAME=$(echo "$service_info" | jq -r '.service')
            SERVICE_PATH="/etc/systemd/system/$SERVICE_NAME"
            
            if [[ -f "$SERVICE_PATH" && -w "$SERVICE_PATH" ]]; then
                # Create backup
                cp "$SERVICE_PATH" "${OUTPUT_DIR}/${SERVICE_NAME}.backup"
                
                # Fix common Docker-to-systemd translation errors
                if grep -q "Restart=unless-stopped" "$SERVICE_PATH"; then
                    sed -i 's/Restart=unless-stopped/Restart=always/g' "$SERVICE_PATH"
                    AUTO_FIXED+=("\"Fixed Restart directive in $SERVICE_NAME\"")
                fi
                
                if grep -q "RestartPolicy=" "$SERVICE_PATH"; then
                    sed -i 's/RestartPolicy=/Restart=/g' "$SERVICE_PATH"
                    AUTO_FIXED+=("\"Fixed RestartPolicy directive in $SERVICE_NAME\"")
                fi
                
                # Reload systemd to pick up changes
                systemctl daemon-reload 2>/dev/null || true
            fi
        done
    fi
fi

FAILED_COUNT=$(jq '.failed_services | length' "${RESULTS_FILE}")
if [ "$FAILED_COUNT" -gt 0 ]; then
    FINDINGS+=("\"$FAILED_COUNT services are in failed state\"")
    RECOMMENDATIONS+=("\"Review and fix failed services, then run: systemctl reset-failed\"")
fi

# Update findings
if [ ${#FINDINGS[@]} -gt 0 ]; then
    FINDINGS_JSON=$(printf '%s\n' "${FINDINGS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".findings = $FINDINGS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update recommendations
if [ ${#RECOMMENDATIONS[@]} -gt 0 ]; then
    RECS_JSON=$(printf '%s\n' "${RECOMMENDATIONS[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".recommendations = $RECS_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Update auto-fixed items
if [ ${#AUTO_FIXED[@]} -gt 0 ]; then
    FIXED_JSON=$(printf '%s\n' "${AUTO_FIXED[@]}" | paste -sd, - | sed 's/^/[/;s/$/]/')
    jq ".auto_fixed = $FIXED_JSON" "${RESULTS_FILE}" > "${RESULTS_FILE}.tmp" && mv "${RESULTS_FILE}.tmp" "${RESULTS_FILE}"
fi

# Output final results
echo "✅ Service configuration validation complete!"
echo "📄 Results saved to: ${OUTPUT_DIR}/results.json"
cat "${RESULTS_FILE}" | jq '.'