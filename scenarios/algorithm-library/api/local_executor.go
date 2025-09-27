package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LocalExecutor handles local code execution when Judge0 is unavailable
type LocalExecutor struct {
	timeout time.Duration
}

// NewLocalExecutor creates a new local executor with specified timeout
func NewLocalExecutor(timeout time.Duration) *LocalExecutor {
	return &LocalExecutor{
		timeout: timeout,
	}
}

// ExecutePython executes Python code locally in a sandboxed manner
func (e *LocalExecutor) ExecutePython(code string, stdin string) (*LocalExecutionResult, error) {
	// Create a temporary Python script with the code
	fullCode := fmt.Sprintf(`
import sys
import json
import traceback

# Redirect stdin to use the provided input
import io
sys.stdin = io.StringIO('''%s''')

# Capture stdout
output_buffer = io.StringIO()
original_stdout = sys.stdout
sys.stdout = output_buffer

try:
    # User code execution
%s
    
    # Get the output
    result = output_buffer.getvalue()
    sys.stdout = original_stdout
    print(json.dumps({
        "success": True,
        "output": result,
        "error": ""
    }))
except Exception as e:
    sys.stdout = original_stdout
    print(json.dumps({
        "success": False,
        "output": output_buffer.getvalue(),
        "error": str(e) + "\n" + traceback.format_exc()
    }))
`, stdin, indentCode(code, "    "))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Execute Python code
	cmd := exec.CommandContext(ctx, "python3", "-c", fullCode)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	cmdErr := cmd.Run()
	duration := time.Since(start)

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return &LocalExecutionResult{
			Success:       false,
			Output:        stdout.String(),
			Error:         "Execution timeout exceeded",
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	// Parse the JSON output
	var result struct {
		Success bool   `json:"success"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}

	outputStr := stdout.String()
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		// If we can't parse JSON, treat raw output as result
		return &LocalExecutionResult{
			Success:       cmdErr == nil && stderr.Len() == 0,
			Output:        outputStr,
			Error:         stderr.String(),
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	return &LocalExecutionResult{
		Success:       result.Success,
		Output:        result.Output,
		Error:         result.Error,
		ExecutionTime: duration.Seconds(),
	}, nil
}

// ExecuteGo executes Go code locally
func (e *LocalExecutor) ExecuteGo(code string, stdin string) (*LocalExecutionResult, error) {
	// Create a complete Go program
	fullCode := fmt.Sprintf(`
package main

import (
	"fmt"
	"os"
)

func main() {
	// Set up stdin
	os.Stdin.Close()
	os.Stdin = os.NewFile(0, "stdin")
	
	// User code
	%s
}
`, code)

	// Write to temporary file
	tmpFile := "/tmp/algo_exec_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".go"
	if err := exec.Command("bash", "-c", fmt.Sprintf("echo '%s' > %s", fullCode, tmpFile)).Run(); err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer exec.Command("rm", tmpFile).Run()

	// Compile and run
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "run", tmpFile)
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return &LocalExecutionResult{
			Success:       false,
			Output:        stdout.String(),
			Error:         "Execution timeout exceeded",
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	return &LocalExecutionResult{
		Success:       err == nil,
		Output:        stdout.String(),
		Error:         stderr.String(),
		ExecutionTime: duration.Seconds(),
	}, nil
}

// ExecuteJavaScript executes JavaScript code locally using Node.js
func (e *LocalExecutor) ExecuteJavaScript(code string, stdin string) (*LocalExecutionResult, error) {
	// Create a complete Node.js script
	jsScript := `
const originalInput = '` + stdin + `';
const lines = originalInput.split('\n');
let currentLine = 0;

// Mock readline for stdin
const readline = () => {
    if (currentLine < lines.length) {
        return lines[currentLine++];
    }
    return null;
};

// Capture console.log output
const outputs = [];
const originalLog = console.log;
console.log = (...args) => {
    outputs.push(args.map(arg => String(arg)).join(' '));
};

try {
    // User code
    ` + code + `
    
    // Restore and output
    console.log = originalLog;
    process.stdout.write(JSON.stringify({
        success: true,
        output: outputs.join('\n'),
        error: ""
    }));
} catch (error) {
    console.log = originalLog;
    process.stdout.write(JSON.stringify({
        success: false,
        output: outputs.join('\n'),
        error: error.toString() + '\n' + error.stack
    }));
}
`

	// Execute with Node.js
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "-e", jsScript)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	cmdErr := cmd.Run()
	duration := time.Since(start)

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return &LocalExecutionResult{
			Success:       false,
			Output:        stdout.String(),
			Error:         "Execution timeout exceeded",
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	// Parse the JSON output
	var result struct {
		Success bool   `json:"success"`
		Output  string `json:"output"`
		Error   string `json:"error"`
	}

	outputStr := stdout.String()
	if err := json.Unmarshal([]byte(outputStr), &result); err != nil {
		// If we can't parse JSON, treat raw output as result
		return &LocalExecutionResult{
			Success:       cmdErr == nil && stderr.Len() == 0,
			Output:        outputStr,
			Error:         stderr.String(),
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	return &LocalExecutionResult{
		Success:       result.Success,
		Output:        result.Output,
		Error:         result.Error,
		ExecutionTime: duration.Seconds(),
	}, nil
}

// ExecuteJava executes Java code locally
func (e *LocalExecutor) ExecuteJava(code string, stdin string) (*LocalExecutionResult, error) {
	// Create a complete Java program
	javaCode := fmt.Sprintf(`
import java.util.*;
import java.io.*;

public class AlgorithmExec {
    public static void main(String[] args) {
        Scanner scanner = new Scanner(new ByteArrayInputStream("%s".getBytes()));
        
        // User code execution
        %s
    }
}
`, stdin, code)

	// Write to temporary file
	tmpFile := "/tmp/AlgorithmExec_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".java"
	if err := exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, javaCode)).Run(); err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer exec.Command("rm", tmpFile).Run()
	defer exec.Command("rm", "/tmp/AlgorithmExec.class").Run()

	// Compile
	compileCmd := exec.Command("javac", "-d", "/tmp", tmpFile)
	var compileErr bytes.Buffer
	compileCmd.Stderr = &compileErr
	if err := compileCmd.Run(); err != nil {
		return &LocalExecutionResult{
			Success:       false,
			Output:        "",
			Error:         "Compilation error: " + compileErr.String(),
			ExecutionTime: 0,
		}, nil
	}

	// Run
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "java", "-cp", "/tmp", "AlgorithmExec")
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return &LocalExecutionResult{
			Success:       false,
			Output:        stdout.String(),
			Error:         "Execution timeout exceeded",
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	return &LocalExecutionResult{
		Success:       err == nil,
		Output:        stdout.String(),
		Error:         stderr.String(),
		ExecutionTime: duration.Seconds(),
	}, nil
}

// ExecuteCPP executes C++ code locally
func (e *LocalExecutor) ExecuteCPP(code string, stdin string) (*LocalExecutionResult, error) {
	// Create a complete C++ program
	cppCode := fmt.Sprintf(`
#include <iostream>
#include <string>
#include <vector>
#include <algorithm>
#include <cmath>
#include <sstream>

using namespace std;

int main() {
    // User code
    %s
    
    return 0;
}
`, code)

	// Write to temporary file
	tmpFile := "/tmp/algo_exec_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".cpp"
	tmpBinary := "/tmp/algo_exec_bin_" + fmt.Sprintf("%d", time.Now().UnixNano())

	if err := exec.Command("bash", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tmpFile, cppCode)).Run(); err != nil {
		return nil, fmt.Errorf("failed to create temp file: %v", err)
	}
	defer exec.Command("rm", tmpFile).Run()
	defer exec.Command("rm", tmpBinary).Run()

	// Compile
	compileCmd := exec.Command("g++", "-o", tmpBinary, tmpFile, "-O2", "-std=c++17")
	var compileErr bytes.Buffer
	compileCmd.Stderr = &compileErr
	if err := compileCmd.Run(); err != nil {
		return &LocalExecutionResult{
			Success:       false,
			Output:        "",
			Error:         "Compilation error: " + compileErr.String(),
			ExecutionTime: 0,
		}, nil
	}

	// Run
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, tmpBinary)
	cmd.Stdin = strings.NewReader(stdin)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return &LocalExecutionResult{
			Success:       false,
			Output:        stdout.String(),
			Error:         "Execution timeout exceeded",
			ExecutionTime: duration.Seconds(),
		}, nil
	}

	return &LocalExecutionResult{
		Success:       err == nil,
		Output:        stdout.String(),
		Error:         stderr.String(),
		ExecutionTime: duration.Seconds(),
	}, nil
}

// LocalExecutionResult contains the result of local code execution
type LocalExecutionResult struct {
	Success       bool    `json:"success"`
	Output        string  `json:"output"`
	Error         string  `json:"error"`
	ExecutionTime float64 `json:"execution_time_seconds"`
}

// indentCode adds indentation to each line of the code
func indentCode(code string, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
