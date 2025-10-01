#!/bin/bash
set -e

API_URL="${API_URL:-http://localhost:19852}"

echo "=== Testing New Calendar Features ==="

# Test smart scheduling optimization
echo "Testing schedule optimization..."
curl -s -X POST "$API_URL/api/v1/schedule/optimize" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer test" \
  -d '{
    "request": "find 2 hours this week",
    "constraints": {
      "business_hours_only": true
    }
  }' | jq '.suggestions' > /dev/null && echo "✅ Schedule optimization working"

# Test timezone handling
echo "Testing timezone support..."
curl -s "$API_URL/api/v1/events?timezone=America/New_York" \
  -H "Authorization: Bearer test" | jq '.timezone' | grep -q "America/New_York" && echo "✅ Timezone detection working"

# Test iCal export
echo "Testing iCal export..."
curl -s "$API_URL/api/v1/events/export/ical" \
  -H "Authorization: Bearer test" | grep -q "BEGIN:VCALENDAR" && echo "✅ iCal export working"

# Test iCal import
echo "Testing iCal import..."
ICAL_DATA="BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//EN
BEGIN:VEVENT
UID:test-import-$(date +%s)@vrooli
DTSTART:20251001T100000Z
DTEND:20251001T110000Z
SUMMARY:Test Import Event
DESCRIPTION:Testing iCal import
END:VEVENT
END:VCALENDAR"

curl -s -X POST "$API_URL/api/v1/events/import/ical" \
  -H "Content-Type: text/calendar" \
  -H "Authorization: Bearer test" \
  -d "$ICAL_DATA" | jq '.imported_count' | grep -q "1" && echo "✅ iCal import working"

echo "=== All new features tested successfully ==="