#!/bin/bash
set -e
echo "=== Performance Tests ==="

API_PORT=${API_PORT:-19500}

# Check if API is running
if ! curl -sf http://localhost:${API_PORT}/health > /dev/null 2>&1; then
  echo "API not running, skipping performance tests"
  exit 0
fi

echo "Running performance benchmarks..."

# Test 1: Response time for FIRE calculation
echo "Testing FIRE calculation response time..."
start_time=$(date +%s%N)
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"current_age":30,"current_savings":100000,"annual_income":100000,"annual_expenses":50000,"savings_rate":50,"expected_return":7,"target_withdrawal_rate":4}' \
  http://localhost:${API_PORT}/api/v1/calculate/fire > /dev/null
end_time=$(date +%s%N)
fire_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [ $fire_time -lt 100 ]; then
  echo "✅ FIRE calculation: ${fire_time}ms (target: <100ms)"
else
  echo "⚠️  FIRE calculation: ${fire_time}ms (slower than target)"
fi

# Test 2: Response time for compound interest
echo "Testing compound interest response time..."
start_time=$(date +%s%N)
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"principal":10000,"annual_rate":7,"years":10,"monthly_contribution":500,"compound_frequency":"monthly"}' \
  http://localhost:${API_PORT}/api/v1/calculate/compound-interest > /dev/null
end_time=$(date +%s%N)
compound_time=$(( (end_time - start_time) / 1000000 ))

if [ $compound_time -lt 100 ]; then
  echo "✅ Compound interest: ${compound_time}ms (target: <100ms)"
else
  echo "⚠️  Compound interest: ${compound_time}ms (slower than target)"
fi

# Test 3: Response time for mortgage calculation
echo "Testing mortgage calculation response time..."
start_time=$(date +%s%N)
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{"loan_amount":300000,"annual_rate":4.5,"years":30}' \
  http://localhost:${API_PORT}/api/v1/calculate/mortgage > /dev/null
end_time=$(date +%s%N)
mortgage_time=$(( (end_time - start_time) / 1000000 ))

if [ $mortgage_time -lt 100 ]; then
  echo "✅ Mortgage calculation: ${mortgage_time}ms (target: <100ms)"
else
  echo "⚠️  Mortgage calculation: ${mortgage_time}ms (slower than target)"
fi

# Test 4: Throughput test (10 concurrent requests)
echo "Testing throughput (10 concurrent requests)..."
throughput_start=$(date +%s)

for i in {1..10}; do
  curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"current_age":30,"current_savings":100000,"annual_income":100000,"annual_expenses":50000,"savings_rate":50,"expected_return":7,"target_withdrawal_rate":4}' \
    http://localhost:${API_PORT}/api/v1/calculate/fire > /dev/null &
done

wait
throughput_end=$(date +%s)
throughput_time=$((throughput_end - throughput_start))

echo "✅ Processed 10 concurrent requests in ${throughput_time}s"

# Check memory usage if available
if command -v ps &> /dev/null; then
  api_pid=$(pgrep -f "financial-calculators-hub-api" | head -1)
  if [ -n "$api_pid" ]; then
    mem_usage=$(ps -p $api_pid -o rss= 2>/dev/null || echo "0")
    mem_mb=$((mem_usage / 1024))
    if [ $mem_mb -lt 512 ]; then
      echo "✅ Memory usage: ${mem_mb}MB (target: <512MB)"
    else
      echo "⚠️  Memory usage: ${mem_mb}MB (higher than target)"
    fi
  fi
fi

echo "✅ Performance tests completed"
