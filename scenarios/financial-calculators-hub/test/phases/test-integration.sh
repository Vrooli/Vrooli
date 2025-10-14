#!/bin/bash
set -e
echo "=== Integration Tests ==="

# API_PORT and UI_PORT are set by the lifecycle system
API_PORT=${API_PORT:-19500}
UI_PORT=${UI_PORT:-37601}

# Test API health
echo "Testing API health endpoint..."
if curl -sf http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "✅ API health check passed"
else
  echo "❌ API health check failed"
  exit 1
fi

# Test UI health
echo "Testing UI health endpoint..."
if curl -sf http://localhost:${UI_PORT}/health > /dev/null 2>&1; then
  echo "✅ UI health check passed"
else
  echo "❌ UI health check failed"
  exit 1
fi

# Test UI can reach API
echo "Testing UI to API connectivity..."
ui_health=$(curl -s http://localhost:${UI_PORT}/health)
if echo "$ui_health" | grep -q '"api_connectivity"'; then
  api_status=$(echo "$ui_health" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [ "$api_status" = "connected" ]; then
    echo "✅ UI successfully connects to API"
  else
    echo "⚠️  UI reports API status: $api_status"
  fi
else
  echo "⚠️  UI health endpoint doesn't report API connectivity"
fi

# Test calculation endpoints with realistic data
echo "Testing calculation endpoints..."

# Test FIRE calculation
fire_result=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"current_age":35,"current_savings":200000,"annual_income":120000,"annual_expenses":60000,"savings_rate":50,"expected_return":7,"target_withdrawal_rate":4}' \
  http://localhost:${API_PORT}/api/v1/calculate/fire)

if echo "$fire_result" | grep -q '"retirement_age"'; then
  retirement_age=$(echo "$fire_result" | grep -o '"retirement_age":[0-9.]*' | cut -d: -f2)
  echo "✅ FIRE calculation working (retirement at age $retirement_age)"
else
  echo "❌ FIRE calculation failed"
  exit 1
fi

# Test compound interest
compound_result=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"principal":50000,"annual_rate":8,"years":20,"monthly_contribution":500,"compound_frequency":"monthly"}' \
  http://localhost:${API_PORT}/api/v1/calculate/compound-interest)

if echo "$compound_result" | grep -q '"final_amount"'; then
  final_amount=$(echo "$compound_result" | grep -o '"final_amount":[0-9.]*' | cut -d: -f2)
  echo "✅ Compound interest working (final amount: \$$final_amount)"
else
  echo "❌ Compound interest calculation failed"
  exit 1
fi

# Test database integration if PostgreSQL is available
if [ -n "$POSTGRES_HOST" ]; then
  echo "Testing database integration..."

  # Try to fetch calculation history
  history_result=$(curl -s http://localhost:${API_PORT}/api/v1/calculations)

  if echo "$history_result" | grep -q 'calculations'; then
    echo "✅ Database integration working"
  else
    echo "⚠️  Database integration may not be working (this is optional)"
  fi
else
  echo "ℹ️  PostgreSQL not configured, skipping database integration tests"
fi

echo "✅ Integration tests completed"
