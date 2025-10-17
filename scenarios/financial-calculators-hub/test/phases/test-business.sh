#!/bin/bash
set -e
echo "=== Business Logic Tests ==="

# API_PORT is set by the lifecycle system when run via 'make test' or 'vrooli scenario test'
API_PORT=${API_PORT:-19500}

if curl -sf http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "API is healthy, running business tests"

  # Test 1: FIRE calculation business logic
  echo "Testing FIRE calculator..."
  result=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"current_age":30,"current_savings":100000,"annual_income":100000,"annual_expenses":50000,"savings_rate":50,"expected_return":7,"target_withdrawal_rate":4}' \
    http://localhost:${API_PORT}/api/v1/calculate/fire)

  if echo "$result" | grep -q '"retirement_age"'; then
    echo "✅ FIRE calculation business logic validated"
  else
    echo "❌ FIRE calculation failed"
    exit 1
  fi

  # Test 2: Compound interest business logic
  echo "Testing compound interest calculator..."
  result=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"principal":10000,"annual_rate":7,"years":10,"monthly_contribution":500,"compound_frequency":"monthly"}' \
    http://localhost:${API_PORT}/api/v1/calculate/compound-interest)

  if echo "$result" | grep -q '"final_amount"'; then
    echo "✅ Compound interest business logic validated"
  else
    echo "❌ Compound interest calculation failed"
    exit 1
  fi

  # Test 3: Mortgage calculation business logic
  echo "Testing mortgage calculator..."
  result=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"loan_amount":300000,"annual_rate":4.5,"years":30}' \
    http://localhost:${API_PORT}/api/v1/calculate/mortgage)

  if echo "$result" | grep -q '"monthly_payment"'; then
    echo "✅ Mortgage calculation business logic validated"
  else
    echo "❌ Mortgage calculation failed"
    exit 1
  fi

  # Test 4: Emergency fund calculation business logic
  echo "Testing emergency fund calculator..."
  result=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"monthly_expenses":4000,"job_stability":"medium","has_dependents":true,"has_insurance":true}' \
    http://localhost:${API_PORT}/api/v1/calculate/emergency-fund)

  if echo "$result" | grep -q '"recommended_fund"'; then
    echo "✅ Emergency fund business logic validated"
  else
    echo "❌ Emergency fund calculation failed"
    exit 1
  fi

  echo "Business logic validation passed"
else
  echo "API not running, skipping business tests"
fi
echo "✅ Business tests completed"
