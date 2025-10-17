#!/bin/bash

# Wedding Planner Integration Example
# Shows how the wedding-planner scenario can use contact-book for guest management

echo "=== Wedding Planner - Contact Book Integration Example ==="
echo ""
echo "This example demonstrates how wedding-planner can leverage contact-book"
echo "for sophisticated guest management and relationship-aware seating."
echo ""

# 1. Get all contacts with relationship data for guest list
echo "1. Fetching potential wedding guests with relationships:"
echo "   Command: contact-book list --json"
GUESTS=$(contact-book list --json | jq -r '.persons[] | select(.tags | contains(["friend"]) or contains(["family"]))')
echo "   Found guests with friend/family tags"
echo ""

# 2. Analyze relationships for seating arrangements
echo "2. Getting relationship graph for seating optimization:"
echo "   Command: curl http://localhost:19313/api/v1/relationships"
curl -s http://localhost:19313/api/v1/relationships | jq '.relationships[] | {from: .from_person_id, to: .to_person_id, strength: .strength}' | head -5
echo ""

# 3. Get dietary restrictions and preferences
echo "3. Extracting dietary preferences from metadata:"
PERSON_ID=$(curl -s http://localhost:19313/api/v1/contacts?limit=1 | jq -r '.persons[0].id')
echo "   Command: contact-book get $PERSON_ID --json | jq '.metadata.dietary'"
# Simulating dietary data retrieval
echo '   Example output: {"vegetarian": true, "allergies": ["nuts", "shellfish"]}'
echo ""

# 4. Get communication preferences for invitations
echo "4. Learning communication preferences for invitation delivery:"
echo "   Command: curl http://localhost:19313/api/v1/contacts/$PERSON_ID/preferences"
curl -s "http://localhost:19313/api/v1/contacts/$PERSON_ID/preferences" 2>/dev/null || echo '   {
     "preferred_channels": ["email", "text"],
     "best_time_to_contact": {
       "start_hour": 10,
       "end_hour": 18,
       "timezone": "EST"
     },
     "communication_frequency": "weekly"
   }'
echo ""

# 5. Use analytics for VIP identification
echo "5. Identifying VIP guests using social analytics:"
echo "   Command: curl http://localhost:19313/api/v1/analytics"
curl -s http://localhost:19313/api/v1/analytics | jq '.analytics[] | select(.social_centrality_score > 0.7) | {person_id: .person_id, vip_score: .social_centrality_score}' | head -3
echo ""

# 6. Example: Creating a wedding guest group
echo "6. Creating relationships between wedding party members:"
BRIDE_ID="550e8400-e29b-41d4-a716-446655440001"
GROOM_ID="550e8400-e29b-41d4-a716-446655440002"
echo "   Creating strong relationship between bride and groom..."
cat << EOF
curl -X POST http://localhost:19313/api/v1/relationships \\
  -H "Content-Type: application/json" \\
  -d '{
    "from_person_id": "$BRIDE_ID",
    "to_person_id": "$GROOM_ID",
    "relationship_type": "spouse",
    "strength": 1.0,
    "shared_interests": ["wedding", "travel", "cooking"]
  }'
EOF
echo ""

echo "=== Integration Benefits for Wedding Planner ==="
echo "✓ Automatic guest list generation from existing contacts"
echo "✓ Relationship-aware seating algorithms using graph data"
echo "✓ Dietary and accessibility preferences for catering"
echo "✓ Optimal invitation delivery based on communication preferences"
echo "✓ VIP identification for special treatment"
echo "✓ Family tree visualization for seating arrangements"
echo ""

echo "=== Sample Code for Wedding Planner ==="
cat << 'EOF'
# In wedding-planner scenario:

# Function to get compatible table groups
get_table_groups() {
  # Get all relationships with high affinity
  relationships=$(contact-book relationships --min-strength 0.7 --json)

  # Group by shared interests and relationship strength
  groups=$(echo "$relationships" | jq 'group_by(.shared_interests)')

  # Return table assignments
  echo "$groups"
}

# Function to check dietary conflicts for table
check_dietary_conflicts() {
  table_guests=$1
  for guest_id in $table_guests; do
    dietary=$(contact-book get "$guest_id" --json | jq '.metadata.dietary')
    # Check for conflicts (e.g., vegans at meat-heavy tables)
  done
}

# Function to send personalized invitations
send_invitations() {
  guests=$(contact-book list --tag "wedding-guest" --json)
  for guest in $guests; do
    prefs=$(curl -s "http://localhost:19313/api/v1/contacts/$guest/preferences")
    channel=$(echo "$prefs" | jq -r '.preferred_channels[0]')
    # Send via preferred channel
    case "$channel" in
      "email") send_email_invitation "$guest" ;;
      "text") send_sms_invitation "$guest" ;;
      "mail") generate_physical_invitation "$guest" ;;
    esac
  done
}
EOF

echo ""
echo "=== Try It Yourself ==="
echo "1. List all contacts: contact-book list"
echo "2. Search for family: contact-book search 'family' --json"
echo "3. Get relationships: curl http://localhost:19313/api/v1/relationships"
echo "4. View analytics: curl http://localhost:19313/api/v1/analytics"