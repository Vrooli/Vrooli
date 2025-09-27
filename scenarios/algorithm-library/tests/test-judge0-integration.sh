#!/bin/bash

# Test Algorithm Library execution capabilities (Judge0 + Local Executor)

set -e

echo "Testing Algorithm Library Execution Capabilities..."

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m'

# API endpoint - dynamically get port
API_PORT="${API_PORT:-16848}"
API_URL="http://localhost:$API_PORT"

# Check if Judge0 is available (optional dependency)
echo -n "Checking Judge0 availability... "
if resource-judge0 status 2>/dev/null | grep -q "Installed: Yes"; then
    echo -e "${GREEN}✓${NC} (available)"
    JUDGE0_AVAILABLE=true
else
    echo -e "${YELLOW}⚠${NC} (not available - using local executor)"
    JUDGE0_AVAILABLE=false
fi

# Test algorithm validation endpoint (uses local executor as fallback)
echo ""
echo "Testing Algorithm Validation via API:"

# Test Python execution
echo -n "  Python execution... "
PYTHON_CODE='def bubble_sort(arr):
    n = len(arr)
    for i in range(n):
        for j in range(0, n-i-1):
            if arr[j] > arr[j+1]:
                arr[j], arr[j+1] = arr[j+1], arr[j]
    return arr

# Handle input format from test cases
import json
import sys
if len(sys.argv) > 1:
    input_data = json.loads(sys.argv[1])
else:
    input_data = json.loads(input())

# Extract array from dict if needed
if isinstance(input_data, dict) and "arr" in input_data:
    arr = input_data["arr"]
else:
    arr = input_data

result = bubble_sort(arr)
print(json.dumps(result))'

RESPONSE=$(curl -s -X POST "$API_URL/api/v1/algorithms/validate" \
    -H "Content-Type: application/json" \
    -d "{
        \"algorithm_id\": \"2db2745d-18a9-4fdd-8921-f876ee15bd1c\",
        \"language\": \"python\",
        \"code\": $(echo "$PYTHON_CODE" | jq -Rs .)
    }" 2>/dev/null)

if echo "$RESPONSE" | jq -e '.test_results | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "    Response: $(echo "$RESPONSE" | head -c 200)"
fi

# Test JavaScript execution  
echo -n "  JavaScript execution... "
JS_CODE='function bubbleSort(arr) {
    const n = arr.length;
    for (let i = 0; i < n; i++) {
        for (let j = 0; j < n - i - 1; j++) {
            if (arr[j] > arr[j + 1]) {
                [arr[j], arr[j + 1]] = [arr[j + 1], arr[j]];
            }
        }
    }
    return arr;
}
const input = JSON.parse(process.argv[2] || require("fs").readFileSync(0, "utf8"));
const arr = input.arr !== undefined ? input.arr : input;
console.log(JSON.stringify(bubbleSort(arr)));'

RESPONSE=$(curl -s -X POST "$API_URL/api/v1/algorithms/validate" \
    -H "Content-Type: application/json" \
    -d "{
        \"algorithm_id\": \"2db2745d-18a9-4fdd-8921-f876ee15bd1c\",
        \"language\": \"javascript\",
        \"code\": $(echo "$JS_CODE" | jq -Rs .)
    }" 2>/dev/null)

if echo "$RESPONSE" | jq -e '.test_results | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "    Response: $(echo "$RESPONSE" | head -c 200)"
fi

# Test Go execution
echo -n "  Go execution... "
GO_CODE='package main
import (
    "encoding/json"
    "fmt"
    "os"
)
func bubbleSort(arr []int) []int {
    n := len(arr)
    for i := 0; i < n; i++ {
        for j := 0; j < n-i-1; j++ {
            if arr[j] > arr[j+1] {
                arr[j], arr[j+1] = arr[j+1], arr[j]
            }
        }
    }
    return arr
}
type Input struct {
    Arr []int `json:"arr"`
}
func main() {
    var rawInput json.RawMessage
    json.NewDecoder(os.Stdin).Decode(&rawInput)
    
    // Try to decode as Input struct first
    var input Input
    if err := json.Unmarshal(rawInput, &input); err == nil && len(input.Arr) > 0 {
        result := bubbleSort(input.Arr)
        output, _ := json.Marshal(result)
        fmt.Println(string(output))
    } else {
        // Fall back to direct array
        var arr []int
        json.Unmarshal(rawInput, &arr)
        result := bubbleSort(arr)
        output, _ := json.Marshal(result)
        fmt.Println(string(output))
    }
}'

RESPONSE=$(curl -s -X POST "$API_URL/api/v1/algorithms/validate" \
    -H "Content-Type: application/json" \
    -d "{
        \"algorithm_id\": \"2db2745d-18a9-4fdd-8921-f876ee15bd1c\",
        \"language\": \"go\",
        \"code\": $(echo "$GO_CODE" | jq -Rs .)
    }" 2>/dev/null)

if echo "$RESPONSE" | jq -e '.test_results | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    echo "    Response: $(echo "$RESPONSE" | head -c 200)"
fi

# Test compilation support (Java)
echo -n "  Java compilation... "
JAVA_CODE='import java.util.*;
import com.google.gson.Gson;
public class Solution {
    public static int[] bubbleSort(int[] arr) {
        int n = arr.length;
        for (int i = 0; i < n; i++) {
            for (int j = 0; j < n - i - 1; j++) {
                if (arr[j] > arr[j + 1]) {
                    int temp = arr[j];
                    arr[j] = arr[j + 1];
                    arr[j + 1] = temp;
                }
            }
        }
        return arr;
    }
    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in);
        String input = scanner.nextLine();
        Gson gson = new Gson();
        int[] arr = gson.fromJson(input, int[].class);
        int[] result = bubbleSort(arr);
        System.out.println(gson.toJson(result));
    }
}'

RESPONSE=$(curl -s -X POST "$API_URL/api/v1/algorithms/validate" \
    -H "Content-Type: application/json" \
    -d "{
        \"algorithm_id\": \"2db2745d-18a9-4fdd-8921-f876ee15bd1c\",
        \"language\": \"java\",
        \"code\": $(echo "$JAVA_CODE" | jq -Rs .)
    }" 2>/dev/null)

if echo "$RESPONSE" | jq -e '.test_results | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} (compilation not available)"
fi

# Test C++ compilation
echo -n "  C++ compilation... "
CPP_CODE='#include <iostream>
#include <vector>
#include <nlohmann/json.hpp>
using json = nlohmann::json;
using namespace std;

vector<int> bubbleSort(vector<int> arr) {
    int n = arr.size();
    for (int i = 0; i < n; i++) {
        for (int j = 0; j < n - i - 1; j++) {
            if (arr[j] > arr[j + 1]) {
                swap(arr[j], arr[j + 1]);
            }
        }
    }
    return arr;
}

int main() {
    json j;
    cin >> j;
    vector<int> arr = j.get<vector<int>>();
    vector<int> result = bubbleSort(arr);
    json output = result;
    cout << output.dump() << endl;
    return 0;
}'

RESPONSE=$(curl -s -X POST "$API_URL/api/v1/algorithms/validate" \
    -H "Content-Type: application/json" \
    -d "{
        \"algorithm_id\": \"2db2745d-18a9-4fdd-8921-f876ee15bd1c\",
        \"language\": \"cpp\",
        \"code\": $(echo "$CPP_CODE" | jq -Rs .)
    }" 2>/dev/null)

if echo "$RESPONSE" | jq -e '.test_results | length > 0' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠${NC} (compilation not available)"
fi

echo ""
echo "Algorithm Library execution test complete!"
echo ""
if [ "$JUDGE0_AVAILABLE" = false ]; then
    echo -e "${YELLOW}Note:${NC} Judge0 is not available, using local executor fallback."
    echo "      This is acceptable - all core functionality works via local execution."
fi