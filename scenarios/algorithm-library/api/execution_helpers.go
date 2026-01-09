package main

import (
	"fmt"
	"time"
)

// WrapCodeWithTestHarness wraps user code with a simple test harness for supported languages.
func WrapCodeWithTestHarness(language, code, algorithmName, inputJSON string) string {
	switch language {
	case "python":
		return fmt.Sprintf(`%s

# Test harness
import json
test_input = json.loads('%s')
result = %s(test_input['arr']) if 'arr' in test_input else %s(test_input.get('n', test_input.get('array', None)))
print(json.dumps(result))
`, code, inputJSON, algorithmName, algorithmName)
	case "javascript":
		return fmt.Sprintf(`%s

// Test harness
const testInput = %s;
const result = %s(testInput.arr !== undefined ? testInput.arr : (testInput.n !== undefined ? testInput.n : testInput.array));
console.log(JSON.stringify(result));
`, code, inputJSON, algorithmName)
	case "go":
		return fmt.Sprintf(`package main
import (
	"encoding/json"
	"fmt"
)

%s

func main() {
	input := %s
	var testInput map[string]interface{}
	json.Unmarshal([]byte(input), &testInput)
	var arr []int
	if testInput["arr"] != nil {
		for _, v := range testInput["arr"].([]interface{}) {
			arr = append(arr, int(v.(float64)))
		}
	}
	result := %s(arr)
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
`, code, inputJSON, algorithmName)
	default:
		// For unsupported languages, return code as-is
		return code
	}
}

// ExecuteWithFallback runs code locally via the language executor.
func ExecuteWithFallback(language, code, input string, timeout time.Duration) (*LocalExecutionResult, error) {
	executor := NewLocalExecutor(timeout)

	switch language {
	case "python":
		return executor.ExecutePython(code, input)
	case "javascript":
		return executor.ExecuteJavaScript(code, input)
	case "go":
		return executor.ExecuteGo(code, input)
	case "java":
		return executor.ExecuteJava(code, input)
	case "cpp", "c++":
		return executor.ExecuteCPP(code, input)
	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}
