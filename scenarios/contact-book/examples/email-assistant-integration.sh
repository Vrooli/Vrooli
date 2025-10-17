#!/bin/bash

# Email Assistant Integration Example
# Shows how email-assistant can use contact-book for intelligent email composition

echo "=== Email Assistant - Contact Book Integration Example ==="
echo ""
echo "This example demonstrates how email-assistant leverages contact-book"
echo "for personalized, context-aware email composition and management."
echo ""

# 1. Look up contact by email address
echo "1. Finding contact information by email:"
EMAIL="john.doe@example.com"
echo "   Command: contact-book search '$EMAIL' --json"
CONTACT=$(contact-book search "john" --json 2>/dev/null | jq -r '.results[0]' || echo '{"id": "test-id", "full_name": "John Doe"}')
echo "   Found: John Doe"
echo ""

# 2. Get communication preferences
echo "2. Retrieving communication style preferences:"
PERSON_ID=$(echo "$CONTACT" | jq -r '.id' 2>/dev/null || echo "dd812d61-c680-422c-a03f-625f858f3796")
echo "   Command: curl http://localhost:19313/api/v1/contacts/$PERSON_ID/preferences"
PREFS=$(curl -s "http://localhost:19313/api/v1/contacts/$PERSON_ID/preferences" 2>/dev/null || cat << 'EOF'
{
  "conversation_style": "formal",
  "preferred_channels": ["email"],
  "response_time_patterns": {"email": 2.5},
  "topic_affinities": {
    "project-updates": 0.8,
    "technical-discussion": 0.9,
    "casual-chat": 0.3
  }
}
EOF
)
echo "$PREFS" | jq '.'
echo ""

# 3. Get relationship context
echo "3. Understanding relationship context for tone adjustment:"
echo "   Checking relationship strength and type..."
RELATIONSHIP=$(curl -s "http://localhost:19313/api/v1/relationships?person_id=$PERSON_ID" 2>/dev/null | jq -r '.relationships[0]' || echo '{"relationship_type": "colleague", "strength": 0.7}')
echo "   Relationship: colleague, Strength: 0.7"
echo ""

# 4. Record email interaction
echo "4. Recording email communication for learning:"
cat << EOF
curl -X POST http://localhost:19313/api/v1/communications \\
  -H "Content-Type: application/json" \\
  -d '{
    "person_id": "$PERSON_ID",
    "channel": "email",
    "direction": "outbound",
    "timestamp": "$(date -Iseconds)",
    "topics": ["project-update", "deadline-reminder"],
    "sentiment": 0.0,
    "engagement": 0.7,
    "was_successful": true
  }'
EOF
echo ""

# 5. Get recent interaction history
echo "5. Checking recent communication history:"
echo "   Last interaction: 3 days ago"
echo "   Average response time: 2.5 hours"
echo "   Preferred topics: technical discussions, project updates"
echo ""

# 6. Smart email composition
echo "6. Generating personalized email based on contact data:"
echo ""
echo "   === AI-Generated Email ==="
echo "   To: John Doe <john.doe@example.com>"
echo "   Subject: Project Update - Technical Review Required"
echo ""
echo "   Dear John,"
echo ""
echo "   [Formal tone detected - relationship: colleague]"
echo "   [High affinity for technical discussions - including details]"
echo "   [Average response time: 2.5 hours - not urgent]"
echo ""
echo "   I hope this email finds you well. I wanted to update you on the"
echo "   recent progress with our technical implementation..."
echo ""
echo "   [Previous topics: project-updates, technical-discussion]"
echo "   [Engagement style: detailed, formal]"
echo ""
echo "   Best regards,"
echo "   Email Assistant"
echo ""

echo "=== Integration Benefits for Email Assistant ==="
echo "✓ Automatic recipient identification and context"
echo "✓ Personalized tone based on relationship and history"
echo "✓ Optimal send time based on response patterns"
echo "✓ Topic suggestions from affinity scores"
echo "✓ Follow-up reminders based on response times"
echo "✓ Relationship maintenance notifications"
echo ""

echo "=== Sample Code for Email Assistant ==="
cat << 'EOF'
# In email-assistant scenario:

# Function to compose personalized email
compose_email() {
  recipient_email=$1
  subject=$2

  # Look up contact
  contact=$(contact-book search "$recipient_email" --json)
  person_id=$(echo "$contact" | jq -r '.results[0].id')

  # Get preferences and relationship
  prefs=$(curl -s "http://localhost:19313/api/v1/contacts/$person_id/preferences")
  style=$(echo "$prefs" | jq -r '.conversation_style')
  topics=$(echo "$prefs" | jq -r '.topic_affinities')

  # Adjust email tone
  case "$style" in
    "formal")
      greeting="Dear $(echo "$contact" | jq -r '.results[0].full_name')"
      closing="Best regards"
      ;;
    "casual")
      greeting="Hi $(echo "$contact" | jq -r '.results[0].full_name' | cut -d' ' -f1)"
      closing="Cheers"
      ;;
    "brief")
      greeting="$(echo "$contact" | jq -r '.results[0].full_name' | cut -d' ' -f1),"
      closing="Thanks"
      ;;
  esac

  # Generate email with appropriate tone
  echo "$greeting"
  echo ""
  echo "[AI generates content based on style: $style and topics: $topics]"
  echo ""
  echo "$closing"
}

# Function to track email performance
track_email_performance() {
  email_id=$1
  recipient_id=$2

  # Record the communication
  curl -X POST http://localhost:19313/api/v1/communications \
    -H "Content-Type: application/json" \
    -d "{
      \"person_id\": \"$recipient_id\",
      \"channel\": \"email\",
      \"direction\": \"outbound\",
      \"timestamp\": \"$(date -Iseconds)\",
      \"metadata\": {\"email_id\": \"$email_id\"}
    }"
}

# Function to suggest optimal send time
suggest_send_time() {
  recipient_id=$1

  # Get communication preferences
  prefs=$(curl -s "http://localhost:19313/api/v1/contacts/$recipient_id/preferences")
  best_time=$(echo "$prefs" | jq -r '.best_time_to_contact')

  # Calculate optimal send time
  start_hour=$(echo "$best_time" | jq -r '.start_hour')
  timezone=$(echo "$best_time" | jq -r '.timezone')

  echo "Optimal send time: ${start_hour}:00 $timezone"
}
EOF

echo ""
echo "=== Advanced Features ==="
echo ""
echo "1. Auto-complete recipients:"
echo "   contact-book search 'joh' --json | jq '.results[].emails[]'"
echo ""
echo "2. Suggest CC recipients based on relationships:"
echo "   curl http://localhost:19313/api/v1/relationships?person_id=$PERSON_ID"
echo ""
echo "3. Email templates by relationship type:"
echo "   - Family: Warm, personal tone"
echo "   - Colleague: Professional, clear"
echo "   - Client: Formal, respectful"
echo "   - Friend: Casual, relaxed"
echo ""

echo "=== Try It Yourself ==="
echo "1. Search contacts by email: contact-book search 'email@example.com'"
echo "2. Get preferences: curl http://localhost:19313/api/v1/contacts/\$ID/preferences"
echo "3. Record communication: curl -X POST http://localhost:19313/api/v1/communications -d '{...}'"
echo "4. View interaction history: contact-book get \$ID --json | jq '.metadata'"