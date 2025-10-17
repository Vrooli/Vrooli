#!/usr/bin/env bash
# Quick test script for judge0 functionality

set -euo pipefail

JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Judge0 Quick Test${NC}\n"

# Test 1: API is accessible
echo -n "Testing API health... "
if python3 -c "import urllib.request; print(urllib.request.urlopen('${API_URL}/system_info').status)" 2>/dev/null | grep -q "200"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    exit 1
fi

# Test 2: Get languages
echo -n "Testing language endpoint... "
LANG_COUNT=$(python3 -c "
import urllib.request, json
response = urllib.request.urlopen('${API_URL}/languages')
langs = json.loads(response.read())
print(len(langs))
" 2>/dev/null || echo "0")

if [[ $LANG_COUNT -gt 20 ]]; then
    echo -e "${GREEN}✓${NC} ($LANG_COUNT languages available)"
else
    echo -e "${RED}✗${NC} (Only $LANG_COUNT languages)"
    exit 1
fi

# Test 3: Submit Python code
echo -n "Testing Python execution... "
RESULT=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'print(42)',
    'language_id': 71,
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('stdout', '').strip())
" 2>/dev/null || echo "FAILED")

if [[ "$RESULT" == "42" ]]; then
    echo -e "${GREEN}✓${NC} (Output: $RESULT)"
else
    echo -e "${RED}✗${NC} (Expected: 42, Got: $RESULT)"
    exit 1
fi

# Test 4: Submit JavaScript code
echo -n "Testing JavaScript execution... "
RESULT=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'console.log(42)',
    'language_id': 63,
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('stdout', '').strip())
" 2>/dev/null || echo "FAILED")

if [[ "$RESULT" == "42" ]]; then
    echo -e "${GREEN}✓${NC} (Output: $RESULT)"
else
    echo -e "${RED}✗${NC} (Expected: 42, Got: $RESULT)"
    exit 1
fi

# Test 5: Submit Go code
echo -n "Testing Go execution... "
RESULT=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'package main\nimport \"fmt\"\nfunc main() { fmt.Println(42) }',
    'language_id': 60,
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('stdout', '').strip())
" 2>/dev/null || echo "FAILED")

if [[ "$RESULT" == "42" ]]; then
    echo -e "${GREEN}✓${NC} (Output: $RESULT)"
else
    echo -e "${RED}✗${NC} (Expected: 42, Got: $RESULT)"
    exit 1
fi

# Test 6: Test with input/output
echo -n "Testing stdin/stdout handling... "
RESULT=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'name = input(); print(f\"Hello, {name}!\")',
    'language_id': 71,
    'stdin': 'World',
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('stdout', '').strip())
" 2>/dev/null || echo "FAILED")

if [[ "$RESULT" == "Hello, World!" ]]; then
    echo -e "${GREEN}✓${NC} (Output: $RESULT)"
else
    echo -e "${RED}✗${NC} (Expected: 'Hello, World!', Got: '$RESULT')"
    exit 1
fi

# Test 7: Test error handling
echo -n "Testing error handling... "
STATUS_ID=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'print(undefined_variable)',
    'language_id': 71,
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('status', {}).get('id', 0))
" 2>/dev/null || echo "0")

# Status IDs 6-11 are various error states
if [[ $STATUS_ID -ge 6 && $STATUS_ID -le 11 ]]; then
    echo -e "${GREEN}✓${NC} (Error correctly detected, status: $STATUS_ID)"
else
    echo -e "${RED}✗${NC} (Expected error status 6-11, got: $STATUS_ID)"
    exit 1
fi

# Test 8: Test resource limits (CPU time)
echo -n "Testing CPU time limit... "
STATUS_ID=$(python3 -c "
import urllib.request, json
data = json.dumps({
    'source_code': 'while True: pass',
    'language_id': 71,
    'cpu_time_limit': 1,
    'wait': True
}).encode('utf-8')
req = urllib.request.Request('${API_URL}/submissions?wait=true', data=data)
req.add_header('Content-Type', 'application/json')
response = urllib.request.urlopen(req)
result = json.loads(response.read())
print(result.get('status', {}).get('id', 0))
" 2>/dev/null || echo "0")

# Status ID 5 is Time Limit Exceeded
if [[ $STATUS_ID -eq 5 ]]; then
    echo -e "${GREEN}✓${NC} (Time limit correctly enforced)"
else
    echo -e "${RED}✗${NC} (Expected TLE status 5, got: $STATUS_ID)"
    exit 1
fi

echo -e "\n${GREEN}All tests passed!${NC}"
exit 0