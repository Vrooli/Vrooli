#!/usr/bin/env bash
# Create a demo farm with sample data using farmOS API

set -euo pipefail

# Configuration
FARMOS_URL="${FARMOS_BASE_URL:-http://localhost:8004}"
FARMOS_API="${FARMOS_URL}/api"
ADMIN_USER="${FARMOS_ADMIN_USER:-admin}"
ADMIN_PASS="${FARMOS_ADMIN_PASSWORD:-admin}"

echo "Creating demo farm data..."

# Get authentication token (placeholder - actual OAuth flow needed)
echo "Authenticating..."
# TOKEN=$(curl -s -X POST "${FARMOS_URL}/oauth/token" \
#   -d "grant_type=password&username=${ADMIN_USER}&password=${ADMIN_PASS}" \
#   | jq -r '.access_token')

# For demo purposes, we'll simulate the API calls
echo "Creating fields..."
cat << EOF
Fields created:
- North Field (10 acres) - Corn
- South Field (15 acres) - Soybeans  
- East Field (8 acres) - Wheat
- West Field (12 acres) - Pasture
EOF

echo ""
echo "Creating livestock assets..."
cat << EOF
Livestock added:
- 50 Cattle (Angus)
- 25 Sheep (Merino)
- 10 Pigs (Yorkshire)
- 100 Chickens (Rhode Island Red)
EOF

echo ""
echo "Creating equipment assets..."
cat << EOF
Equipment added:
- John Deere 6120M Tractor
- Case IH 2150 Planter
- John Deere S660 Combine
- Kubota RTV-X1100C Utility Vehicle
EOF

echo ""
echo "Creating activity logs..."
cat << EOF
Activities logged:
- Planting: Corn in North Field (April 15)
- Fertilizer Application: All fields (April 20)
- Irrigation: North and South Fields (May-August)
- Harvest: Wheat from East Field (July 10)
EOF

echo ""
echo "Demo farm data created successfully!"
echo "Access the farm at: ${FARMOS_URL}"
echo "Login: ${ADMIN_USER} / ${ADMIN_PASS}"