package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLocalExecutor(t *testing.T) {
	executor := NewLocalExecutor(5 * time.Second)
	assert.NotNil(t, executor)
	assert.Equal(t, 5*time.Second, executor.timeout)
}

func TestExecutePython(t *testing.T) {
	executor := NewLocalExecutor(5 * time.Second)
	
	tests := []struct {
		name       string
		code       string
		stdin      string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "Hello World",
			code:       `print("Hello, World!")`,
			stdin:      "",
			wantOutput: "Hello, World!",
			wantErr:    false,
		},
		{
			name: "Read input",
			code: `
import sys
data = sys.stdin.read().strip()
print(f"Input: {data}")`,
			stdin:      "test data",
			wantOutput: "Input: test data",
			wantErr:    false,
		},
		{
			name:       "Syntax error",
			code:       `print("unclosed`,
			stdin:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecutePython(tt.code, tt.stdin)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, strings.TrimSpace(result.Output), strings.TrimSpace(tt.wantOutput))
				assert.Greater(t, result.ExecutionTime, 0.0)
			}
		})
	}
}

func TestExecuteJavaScript(t *testing.T) {
	executor := NewLocalExecutor(5 * time.Second)
	
	tests := []struct {
		name       string
		code       string
		stdin      string
		wantOutput string
		wantErr    bool
	}{
		{
			name:       "Console log",
			code:       `console.log("Hello from JS");`,
			stdin:      "",
			wantOutput: "Hello from JS",
			wantErr:    false,
		},
		{
			name:       "Syntax error",
			code:       `console.log(unclosed`,
			stdin:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteJavaScript(tt.code, tt.stdin)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, strings.TrimSpace(result.Output), strings.TrimSpace(tt.wantOutput))
			}
		})
	}
}

func TestExecuteGo(t *testing.T) {
	executor := NewLocalExecutor(10 * time.Second) // Go compilation takes longer
	
	tests := []struct {
		name       string
		code       string
		stdin      string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "Hello World",
			code: `
package main
import "fmt"
func main() {
    fmt.Println("Hello from Go")
}`,
			stdin:      "",
			wantOutput: "Hello from Go",
			wantErr:    false,
		},
		{
			name: "Compile error",
			code: `
package main
func main() {
    fmt.Println("missing import")
}`,
			stdin:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteGo(tt.code, tt.stdin)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, strings.TrimSpace(result.Output), strings.TrimSpace(tt.wantOutput))
			}
		})
	}
}

func TestExecuteJava(t *testing.T) {
	executor := NewLocalExecutor(10 * time.Second)
	
	tests := []struct {
		name       string
		code       string
		stdin      string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "Hello World",
			code: `
public class Main {
    public static void main(String[] args) {
        System.out.println("Hello from Java");
    }
}`,
			stdin:      "",
			wantOutput: "Hello from Java",
			wantErr:    false,
		},
		{
			name: "Compile error",
			code: `
public class Main {
    public static void main(String[] args) {
        System.out.println(missingVariable);
    }
}`,
			stdin:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteJava(tt.code, tt.stdin)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, strings.TrimSpace(result.Output), strings.TrimSpace(tt.wantOutput))
			}
		})
	}
}

func TestExecuteCPP(t *testing.T) {
	executor := NewLocalExecutor(10 * time.Second)
	
	tests := []struct {
		name       string
		code       string
		stdin      string
		wantOutput string
		wantErr    bool
	}{
		{
			name: "Hello World",
			code: `
#include <iostream>
int main() {
    std::cout << "Hello from C++" << std::endl;
    return 0;
}`,
			stdin:      "",
			wantOutput: "Hello from C++",
			wantErr:    false,
		},
		{
			name: "Compile error",
			code: `
#include <iostream>
int main() {
    cout << "Missing std::" << endl;
    return 0;
}`,
			stdin:      "",
			wantOutput: "",
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteCPP(tt.code, tt.stdin)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Contains(t, strings.TrimSpace(result.Output), strings.TrimSpace(tt.wantOutput))
			}
		})
	}
}

func TestExecutorTimeout(t *testing.T) {
	executor := NewLocalExecutor(1 * time.Second)
	
	// Code that will timeout
	code := `
import time
time.sleep(5)
print("Should not appear")
`
	
	result, err := executor.ExecutePython(code, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Nil(t, result)
}

func TestIndentCode(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		indent string
		want   string
	}{
		{
			name:   "Single line",
			code:   "print('hello')",
			indent: "    ",
			want:   "    print('hello')",
		},
		{
			name:   "Multiple lines",
			code:   "line1\nline2\nline3",
			indent: "  ",
			want:   "  line1\n  line2\n  line3",
		},
		{
			name:   "Empty lines preserved",
			code:   "line1\n\nline2",
			indent: "    ",
			want:   "    line1\n    \n    line2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indentCode(tt.code, tt.indent)
			assert.Equal(t, tt.want, got)
		})
	}
}