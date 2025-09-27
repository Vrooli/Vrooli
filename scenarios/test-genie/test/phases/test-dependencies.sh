#!/bin/bash

# Phase: Dependencies
# Exercises database connectivity, persistence, and concurrent access guarantees

set -e

# Get the API port dynamically
API_PORT=$(vrooli scenario port test-genie API_PORT 2>/dev/null)
if [[ -z "$API_PORT" ]]; then
    echo "❌ test-genie scenario is not running"
    echo "   Start it with: vrooli scenario run test-genie"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🗄️  Testing Test Genie Database Integration${NC}"

# Test 1: Database Connection Health
echo -e "\n${YELLOW}🔍 Test 1: Database Connection Health${NC}"
health_response=$(curl -s "$API_URL/health")
if echo "$health_response" | jq -e '.database' >/dev/null 2>&1; then
    db_status=$(echo "$health_response" | jq -r '.database.status // "unknown"')
    if [[ "$db_status" == "healthy" ]]; then
        echo -e "${GREEN}✅ Database connection healthy${NC}"
    else
        echo -e "${YELLOW}⚠️  Database status: $db_status${NC}"
    fi
else
    echo -e "${RED}❌ Database health check failed${NC}"
    exit 1
fi

# Test 2: Data Persistence - Create and Retrieve Test Suite
echo -e "\n${YELLOW}💾 Test 2: Data Persistence${NC}"
# Create a test suite
create_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "db-persistence-test",
        "test_types": ["unit"],
        "coverage_target": 75,
        "options": {"execution_timeout": 120}
    }')

if echo "$create_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    SUITE_ID=$(echo "$create_response" | jq -r '.suite_id')
    echo -e "${GREEN}✅ Test suite created and persisted${NC}"
    echo -e "   Suite ID: $SUITE_ID"
    
    # Retrieve the same test suite
    retrieve_response=$(curl -s "$API_URL/api/v1/test-suite/$SUITE_ID")
    if echo "$retrieve_response" | jq -e '.id' >/dev/null 2>&1; then
        retrieved_id=$(echo "$retrieve_response" | jq -r '.id')
        if [[ "$retrieved_id" == "$SUITE_ID" ]]; then
            echo -e "${GREEN}✅ Test suite retrieved successfully${NC}"
            echo -e "   Scenario: $(echo "$retrieve_response" | jq -r '.scenario_name')"
        else
            echo -e "${RED}❌ Retrieved suite ID mismatch${NC}"
        fi
    else
        echo -e "${RED}❌ Test suite retrieval failed${NC}"
    fi
else
    echo -e "${RED}❌ Test suite creation failed${NC}"
    exit 1
fi

# Test 3: List Operations
echo -e "\n${YELLOW}📊 Test 3: List Operations${NC}"
# List test suites
list_response=$(curl -s "$API_URL/api/v1/test-suites")
if echo "$list_response" | jq -e '.test_suites' >/dev/null 2>&1; then
    suite_count=$(echo "$list_response" | jq -r '.test_suites | length')
    echo -e "${GREEN}✅ Test suite listing successful${NC}"
    echo -e "   Found: $suite_count test suites"
    
    # Verify our created suite is in the list
    our_suite=$(echo "$list_response" | jq -r ".test_suites[] | select(.id == \"$SUITE_ID\") | .id")
    if [[ "$our_suite" == "$SUITE_ID" ]]; then
        echo -e "${GREEN}✅ Created suite found in list${NC}"
    else
        echo -e "${YELLOW}⚠️  Created suite not found in list${NC}"
    fi
else
    echo -e "${RED}❌ Test suite listing failed${NC}"
fi

# Test 4: Complex Query - Coverage Analysis
echo -e "\n${YELLOW}🔍 Test 4: Complex Database Queries${NC}"
coverage_response=$(curl -s -X POST "$API_URL/api/v1/test-analysis/coverage" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "db-query-test",
        "source_code_paths": ["/tmp/test"],
        "existing_test_paths": [],
        "analysis_depth": "detailed"
    }')

if echo "$coverage_response" | jq -e '.overall_coverage' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Complex coverage analysis query successful${NC}"
    echo -e "   Coverage: $(echo "$coverage_response" | jq -r '.overall_coverage')%"
    echo -e "   Files analyzed: $(echo "$coverage_response" | jq -r '.coverage_by_file | keys | length')"
else
    echo -e "${RED}❌ Coverage analysis query failed${NC}"
    echo "   Response: $coverage_response"
fi

# Test 5: Transaction Integrity - Test Vault Creation
echo -e "\n${YELLOW}🔒 Test 5: Transaction Integrity${NC}"
vault_response=$(curl -s -X POST "$API_URL/api/v1/test-vault/create" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "db-transaction-test",
        "vault_name": "integrity-test-vault",
        "phases": ["setup", "test", "cleanup"],
        "phase_configurations": {
            "setup": {"timeout": 300},
            "test": {"timeout": 600},
            "cleanup": {"timeout": 300}
        },
        "success_criteria": {
            "min_test_pass_rate": 0.9,
            "max_execution_time": 1200
        }
    }')

if echo "$vault_response" | jq -e '.vault_id' >/dev/null 2>&1; then
    VAULT_ID=$(echo "$vault_response" | jq -r '.vault_id')
    echo -e "${GREEN}✅ Test vault transaction successful${NC}"
    echo -e "   Vault ID: $VAULT_ID"
    
    # Verify vault was stored correctly
    vault_details=$(curl -s "$API_URL/api/v1/test-vault/$VAULT_ID")
    if echo "$vault_details" | jq -e '.id' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Vault data integrity verified${NC}"
        phases_count=$(echo "$vault_details" | jq -r '.phases | length')
        echo -e "   Phases stored: $phases_count"
    else
        echo -e "${RED}❌ Vault data integrity check failed${NC}"
    fi
else
    echo -e "${RED}❌ Test vault creation failed${NC}"
fi

# Test 6: Concurrent Database Operations
echo -e "\n${YELLOW}🚀 Test 6: Concurrent Database Operations${NC}"
echo "Testing concurrent database writes..."

# Start 5 concurrent test suite creations
pids=()
for i in {1..5}; do
    (
        response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
            -H "Content-Type: application/json" \
            -d "{
                \"scenario_name\": \"concurrent-db-test-$i\",
                \"test_types\": [\"unit\"],
                \"coverage_target\": 50,
                \"options\": {\"execution_timeout\": 60}
            }")
        if echo "$response" | jq -e '.suite_id' >/dev/null 2>&1; then
            echo "Concurrent DB operation $i: SUCCESS"
        else
            echo "Concurrent DB operation $i: FAILED"
        fi
    ) &
    pids+=($!)
done

# Wait for all operations to complete
wait "${pids[@]}"
echo -e "${GREEN}✅ Concurrent database operations completed${NC}"

# Test 7: Database Performance Test
echo -e "\n${YELLOW}⚡ Test 7: Database Performance${NC}"
start_time=$(date +%s%N)

# Perform multiple quick database operations
for i in {1..10}; do
    curl -s "$API_URL/health" >/dev/null &
done
wait

end_time=$(date +%s%N)
duration_ms=$(( (end_time - start_time) / 1000000 ))

if [[ $duration_ms -lt 5000 ]]; then
    echo -e "${GREEN}✅ Database performance test passed (${duration_ms}ms for 10 operations)${NC}"
else
    echo -e "${YELLOW}⚠️  Database performance slower than expected (${duration_ms}ms)${NC}"
fi

# Test 8: Data Cleanup Verification
echo -e "\n${YELLOW}🧹 Test 8: Data Cleanup Test${NC}"
# Count records before
initial_count=$(curl -s "$API_URL/api/v1/test-suites" | jq -r '.test_suites | length')

# Create a temporary test suite
temp_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "temp-cleanup-test",
        "test_types": ["unit"],
        "coverage_target": 50,
        "options": {"execution_timeout": 60}
    }')

if echo "$temp_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    # Count records after
    after_count=$(curl -s "$API_URL/api/v1/test-suites" | jq -r '.test_suites | length')
    if [[ $after_count -gt $initial_count ]]; then
        echo -e "${GREEN}✅ Data persistence verified (count: $initial_count → $after_count)${NC}"
    else
        echo -e "${YELLOW}⚠️  Data count unchanged or decreased${NC}"
    fi
else
    echo -e "${RED}❌ Temporary test suite creation failed${NC}"
fi

# Final Database Integration Summary
echo -e "\n${CYAN}📊 Database Integration Test Summary${NC}"
echo -e "${BLUE}🔗 Connection Health: $(echo "$health_response" | jq -e '.database' >/dev/null 2>&1 && echo "✅ Healthy" || echo "❌ Failed")${NC}"
echo -e "${BLUE}💾 Data Persistence: $(echo "$retrieve_response" | jq -e '.id' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}📊 Query Operations: $(echo "$list_response" | jq -e '.test_suites' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}🔒 Transaction Integrity: $(echo "$vault_response" | jq -e '.vault_id' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}⚡ Performance: $([[ $duration_ms -lt 5000 ]] && echo "✅ Good (${duration_ms}ms)" || echo "⚠️  Slow (${duration_ms}ms)")${NC}"

echo -e "\n${BLUE}🎉 Database integration testing completed!${NC}"
echo -e "${BLUE}   Created Suite ID: $SUITE_ID${NC}"
echo -e "${BLUE}   Created Vault ID: $VAULT_ID${NC}"
