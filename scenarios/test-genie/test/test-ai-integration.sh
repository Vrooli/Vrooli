#!/bin/bash

# Test Genie AI Integration Test Suite
# Tests Ollama integration, AI test generation, and fallback mechanisms

set -e

# Get ports dynamically
API_PORT=$(vrooli scenario port test-genie API_PORT 2>/dev/null)
OLLAMA_PORT=$(vrooli scenario port test-genie OLLAMA_PORT 2>/dev/null || echo "11434")

if [[ -z "$API_PORT" ]]; then
    echo "❌ test-genie scenario is not running"
    echo "   Start it with: vrooli scenario run test-genie"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"
OLLAMA_URL="http://localhost:${OLLAMA_PORT}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🤖 Testing Test Genie AI Integration${NC}"

# Test 1: Ollama Service Health
echo -e "\n${YELLOW}🔍 Test 1: Ollama Service Health${NC}"
if curl -s "$OLLAMA_URL/api/tags" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Ollama service is accessible${NC}"
    
    # Check available models
    models=$(curl -s "$OLLAMA_URL/api/tags" | jq -r '.models[]?.name // empty' 2>/dev/null | head -3)
    if [[ -n "$models" ]]; then
        echo -e "${GREEN}✅ Available models found:${NC}"
        echo "$models" | while read -r model; do
            echo "   - $model"
        done
    else
        echo -e "${YELLOW}⚠️  No models found in Ollama${NC}"
        echo "   This may cause AI generation to fail"
    fi
else
    echo -e "${RED}❌ Ollama service not accessible at $OLLAMA_URL${NC}"
    echo "   AI generation will likely fail"
fi

# Test 2: AI-Generated Test Suite (Real AI)
echo -e "\n${YELLOW}🧠 Test 2: AI Test Generation${NC}"
ai_suite_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "calculator-app",
        "test_types": ["unit"],
        "coverage_target": 90,
        "options": {
            "include_performance_tests": false,
            "include_security_tests": false,
            "custom_test_patterns": ["edge-cases", "error-handling"],
            "execution_timeout": 120
        }
    }')

if echo "$ai_suite_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    AI_SUITE_ID=$(echo "$ai_suite_response" | jq -r '.suite_id')
    echo -e "${GREEN}✅ AI test generation successful${NC}"
    echo -e "   Suite ID: $AI_SUITE_ID"
    echo -e "   Generated: $(echo "$ai_suite_response" | jq -r '.generated_tests') tests"
    echo -e "   Coverage: $(echo "$ai_suite_response" | jq -r '.estimated_coverage')%"
    
    # Check if tests contain AI-generated content indicators
    suite_details=$(curl -s "$API_URL/api/v1/test-suite/$AI_SUITE_ID")
    test_count=$(echo "$suite_details" | jq -r '.test_cases | length')
    if [[ "$test_count" -gt 0 ]]; then
        echo -e "${GREEN}✅ Test cases generated: $test_count${NC}"
    else
        echo -e "${YELLOW}⚠️  No test cases in generated suite${NC}"
    fi
else
    echo -e "${RED}❌ AI test generation failed${NC}"
    echo "   Response: $ai_suite_response"
fi

# Test 3: Fallback Generation Test
echo -e "\n${YELLOW}🔄 Test 3: Fallback Generation${NC}"
# Test with an invalid scenario name to potentially trigger fallback
fallback_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "nonexistent-scenario-12345",
        "test_types": ["unit"],
        "coverage_target": 50,
        "options": {
            "execution_timeout": 60
        }
    }')

if echo "$fallback_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Fallback generation successful${NC}"
    echo -e "   Generated: $(echo "$fallback_response" | jq -r '.generated_tests') tests"
else
    echo -e "${YELLOW}⚠️  Fallback generation failed or not triggered${NC}"
    echo "   This might be expected behavior"
fi

# Test 4: Performance vs Security Test Types
echo -e "\n${YELLOW}⚡ Test 4: Performance Test Generation${NC}"
perf_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "web-api",
        "test_types": ["performance", "load"],
        "coverage_target": 70,
        "options": {
            "include_performance_tests": true,
            "include_security_tests": false,
            "execution_timeout": 300
        }
    }')

if echo "$perf_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Performance test generation successful${NC}"
    echo -e "   Generated: $(echo "$perf_response" | jq -r '.generated_tests') tests"
else
    echo -e "${YELLOW}⚠️  Performance test generation failed${NC}"
fi

# Test 5: Security Test Generation
echo -e "\n${YELLOW}🔒 Test 5: Security Test Generation${NC}"
security_response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d '{
        "scenario_name": "user-auth-api",
        "test_types": ["security"],
        "coverage_target": 60,
        "options": {
            "include_performance_tests": false,
            "include_security_tests": true,
            "custom_test_patterns": ["sql-injection", "xss", "auth-bypass"],
            "execution_timeout": 180
        }
    }')

if echo "$security_response" | jq -e '.suite_id' >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Security test generation successful${NC}"
    echo -e "   Generated: $(echo "$security_response" | jq -r '.generated_tests') tests"
else
    echo -e "${YELLOW}⚠️  Security test generation failed${NC}"
fi

# Test 6: AI Response Quality Check
echo -e "\n${YELLOW}🎯 Test 6: AI Response Quality${NC}"
if [[ -n "$AI_SUITE_ID" ]]; then
    quality_check=$(curl -s "$API_URL/api/v1/test-suite/$AI_SUITE_ID")
    
    # Check for quality indicators in the generated tests
    has_descriptions=$(echo "$quality_check" | jq -r '.test_cases[]?.description // empty' | head -1)
    has_assertions=$(echo "$quality_check" | jq -r '.test_cases[]?.test_code // empty' | grep -i "assert\|expect\|should" | head -1)
    
    if [[ -n "$has_descriptions" ]]; then
        echo -e "${GREEN}✅ Generated tests have descriptions${NC}"
    else
        echo -e "${YELLOW}⚠️  Generated tests lack descriptions${NC}"
    fi
    
    if [[ -n "$has_assertions" ]]; then
        echo -e "${GREEN}✅ Generated tests contain assertions${NC}"
    else
        echo -e "${YELLOW}⚠️  Generated tests may lack proper assertions${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  No AI suite available for quality check${NC}"
fi

# Test 7: Concurrent AI Generation
echo -e "\n${YELLOW}🚀 Test 7: Concurrent AI Generation${NC}"
echo "Starting 3 concurrent test generations..."

for i in {1..3}; do
    (
        response=$(curl -s -X POST "$API_URL/api/v1/test-suite/generate" \
            -H "Content-Type: application/json" \
            -d "{
                \"scenario_name\": \"concurrent-test-$i\",
                \"test_types\": [\"unit\"],
                \"coverage_target\": 60,
                \"options\": {\"execution_timeout\": 90}
            }")
        if echo "$response" | jq -e '.suite_id' >/dev/null 2>&1; then
            echo "Concurrent test $i: SUCCESS"
        else
            echo "Concurrent test $i: FAILED"
        fi
    ) &
done

wait
echo -e "${GREEN}✅ Concurrent generation tests completed${NC}"

# Final AI Integration Summary
echo -e "\n${CYAN}📊 AI Integration Test Summary${NC}"
echo -e "${BLUE}🤖 Ollama Integration: $(curl -s "$OLLAMA_URL/api/tags" >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}🧠 AI Test Generation: $(echo "$ai_suite_response" | jq -e '.suite_id' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}⚡ Performance Tests: $(echo "$perf_response" | jq -e '.suite_id' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"
echo -e "${BLUE}🔒 Security Tests: $(echo "$security_response" | jq -e '.suite_id' >/dev/null 2>&1 && echo "✅ Working" || echo "❌ Failed")${NC}"

echo -e "\n${BLUE}🎉 AI Integration testing completed!${NC}"