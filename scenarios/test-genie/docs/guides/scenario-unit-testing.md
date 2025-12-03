# Scenario Unit Testing Guide

This guide covers unit testing for **scenario application code** (Go APIs, Node.js UIs, Python services).

## Multi-Framework Integration

```mermaid
graph TB
    subgraph "Test Frameworks"
        Go[Go Tests<br/>*_test.go]
        Vitest[Vitest Tests<br/>*.test.ts]
        Python[Python Tests<br/>test_*.py]
    end

    subgraph "Tag Extraction"
        GoParser[Go Parser<br/>grep [REQ:ID]]
        VitestReporter[vitest-requirement-reporter<br/>Custom reporter]
        PythonParser[Python Parser<br/>pytest markers]
    end

    subgraph "Test Genie Orchestration"
        Orchestrator[Go Orchestrator<br/>Unit Phase Runner]
        PhaseResults[Phase Results<br/>Structured JSON]
        AutoSync[Requirements Sync<br/>After comprehensive run]
    end

    Go --> GoParser
    Vitest --> VitestReporter
    Python --> PythonParser

    GoParser --> Orchestrator
    VitestReporter --> Orchestrator
    PythonParser --> Orchestrator

    Orchestrator --> PhaseResults
    PhaseResults --> AutoSync

    style Go fill:#00ADD8
    style Vitest fill:#729B1B
    style Python fill:#3776AB
    style PhaseResults fill:#c8e6c9
```

**Key insight:** Multiple test frameworks → One unified requirement tracking system.

## 🎯 Unit Testing Purpose

Unit tests validate individual functions and components in your scenario code:

- ✅ Complete in <60 seconds total
- ✅ Test business logic without external dependencies
- ✅ Achieve >80% code coverage (70% minimum)
- ✅ Run in isolation (no database, API calls, or file I/O)
- ✅ Fast feedback during development

## 🗂️ Testing by Language

### Go Unit Testing

#### Complete Working Example: HTTP Handler Testing

**File: `main.go` (implementation being tested)**
```go
package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
)

// VisitRequest represents a visit tracking request
type VisitRequest struct {
    Files []string `json:"files"`
}

// visitHandler handles visit tracking requests
func visitHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPOST {
        w.WriteHeader(http.StatusMethodNotAllowed)
        fmt.Fprint(w, "Method not allowed")
        return
    }

    var req VisitRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprint(w, "Invalid request")
        return
    }

    if len(req.Files) == 0 {
        w.WriteHeader(http.StatusBadRequest)
        fmt.Fprint(w, "No files provided")
        return
    }

    // Simulate processing
    w.WriteHeader(http.StatusOK)
    fmt.Fprint(w, "recorded")
}
```

**File: `main_test.go` (complete test suite)**
```go
package main

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestVisitHandler(t *testing.T) {
    tests := []struct {
        name           string
        method         string
        body           string
        expectedStatus int
        expectedBody   string
    }{
        {
            name:           "Valid POST request",
            method:         "POST",
            body:           `{"files":["test.go"]}`,
            expectedStatus: http.StatusOK,
            expectedBody:   "recorded",
        },
        {
            name:           "Invalid JSON",
            method:         "POST", 
            body:           `{invalid}`,
            expectedStatus: http.StatusBadRequest,
            expectedBody:   "Invalid request",
        },
        {
            name:           "GET not allowed",
            method:         "GET",
            body:           "",
            expectedStatus: http.StatusMethodNotAllowed,
            expectedBody:   "Method not allowed",
        },
        {
            name:           "Empty files array",
            method:         "POST",
            body:           `{"files":[]}`,
            expectedStatus: http.StatusBadRequest,
            expectedBody:   "No files provided",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest(tt.method, "/visit", bytes.NewBufferString(tt.body))
            req.Header.Set("Content-Type", "application/json")
            
            rr := httptest.NewRecorder()
            handler := http.HandlerFunc(visitHandler)
            handler.ServeHTTP(rr, req)
            
            if rr.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
            }
            
            if tt.expectedBody != "" && !strings.Contains(rr.Body.String(), tt.expectedBody) {
                t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, rr.Body.String())
            }
        })
    }
}

// Run tests with: go test -v
// Run with coverage: go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

#### Gorilla Mux Router Testing
```go
func TestMuxRoutesWithVariables(t *testing.T) {
    router := mux.NewRouter()
    router.HandleFunc("/files/{filename}", fileHandler).Methods("GET")
    
    req := httptest.NewRequest("GET", "/files/test.go", nil)
    rr := httptest.NewRecorder()
    
    router.ServeHTTP(rr, req)
    
    if rr.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", rr.Code)
    }
    
    // Test that mux variables are properly extracted
    vars := mux.Vars(req)
    if vars["filename"] != "test.go" {
        t.Errorf("Expected filename 'test.go', got %q", vars["filename"])
    }
}
```

#### Business Logic Testing
```go
func TestProcessFiles(t *testing.T) {
    tests := []struct {
        name     string
        input    []string
        expected ProcessResult
        wantErr  bool
    }{
        {
            name:     "Empty list",
            input:    []string{},
            expected: ProcessResult{Count: 0, Files: []string{}},
            wantErr:  false,
        },
        {
            name:     "Valid files",
            input:    []string{"main.go", "handler.go"},
            expected: ProcessResult{Count: 2, Files: []string{"main.go", "handler.go"}},
            wantErr:  false,
        },
        {
            name:    "Invalid file extension",
            input:   []string{"invalid.txt"},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ProcessFiles(tt.input)
            
            if tt.wantErr && err == nil {
                t.Error("Expected error but got none")
            }
            if !tt.wantErr && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("Expected %+v, got %+v", tt.expected, result)
            }
        })
    }
}
```

#### Coverage Commands
```bash
# Run tests with coverage
cd api/
go test -v -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out -o coverage.html

# Check coverage percentage
go tool cover -func=coverage.out | grep total

# Set coverage thresholds in CI
go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
    echo "❌ Coverage $COVERAGE% below 80% threshold"
    exit 1
fi
```

### Node.js Unit Testing

#### Jest Configuration
```json
// package.json
{
  "scripts": {
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  },
  "jest": {
    "testEnvironment": "node",
    "collectCoverageFrom": [
      "src/**/*.{js,ts}",
      "!src/**/*.d.ts",
      "!src/**/*.test.{js,ts}"
    ],
    "coverageThreshold": {
      "global": {
        "branches": 80,
        "functions": 80,
        "lines": 80,
        "statements": 80
      }
    }
  }
}
```

#### Complete Working Example: React Component Testing

**File: `src/components/FileList.js` (component being tested)**
```javascript
import React from 'react';
import PropTypes from 'prop-types';

const FileList = ({ files, onFileSelect }) => {
  if (!files || files.length === 0) {
    return <div className="empty-message">No files found</div>;
  }

  return (
    <div className="file-list">
      {files.map((file) => (
        <div
          key={file.id}
          className="file-item"
          onClick={() => onFileSelect && onFileSelect(file)}
          role="button"
          tabIndex={0}
          onKeyPress={(e) => e.key === 'Enter' && onFileSelect && onFileSelect(file)}
        >
          <span className="file-name">{file.name}</span>
          <span className="file-size">{(file.size / 1024).toFixed(1)} KB</span>
        </div>
      ))}
    </div>
  );
};

FileList.propTypes = {
  files: PropTypes.arrayOf(
    PropTypes.shape({
      id: PropTypes.number.isRequired,
      name: PropTypes.string.isRequired,
      size: PropTypes.number.isRequired,
    })
  ),
  onFileSelect: PropTypes.func,
};

FileList.defaultProps = {
  files: [],
  onFileSelect: null,
};

export default FileList;
```

**File: `src/components/FileList.test.js` (complete test suite)**
```javascript
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import FileList from './FileList';

describe('FileList Component', () => {
  const mockFiles = [
    { id: 1, name: 'test.js', size: 1024 },
    { id: 2, name: 'app.js', size: 2048 }
  ];

  test('renders file list correctly', () => {
    render(<FileList files={mockFiles} />);
    
    expect(screen.getByText('test.js')).toBeInTheDocument();
    expect(screen.getByText('app.js')).toBeInTheDocument();
    expect(screen.getByText('1.0 KB')).toBeInTheDocument();
    expect(screen.getByText('2.0 KB')).toBeInTheDocument();
  });

  test('calls onFileSelect when file clicked', () => {
    const mockOnSelect = jest.fn();
    render(<FileList files={mockFiles} onFileSelect={mockOnSelect} />);
    
    fireEvent.click(screen.getByText('test.js'));
    
    expect(mockOnSelect).toHaveBeenCalledTimes(1);
    expect(mockOnSelect).toHaveBeenCalledWith(mockFiles[0]);
  });

  test('handles keyboard navigation', () => {
    const mockOnSelect = jest.fn();
    render(<FileList files={mockFiles} onFileSelect={mockOnSelect} />);
    
    const firstFile = screen.getByText('test.js').closest('.file-item');
    fireEvent.keyPress(firstFile, { key: 'Enter', code: 'Enter' });
    
    expect(mockOnSelect).toHaveBeenCalledWith(mockFiles[0]);
  });

  test('displays empty message when no files', () => {
    render(<FileList files={[]} />);
    
    expect(screen.getByText('No files found')).toBeInTheDocument();
  });

  test('handles null files prop', () => {
    render(<FileList files={null} />);
    
    expect(screen.getByText('No files found')).toBeInTheDocument();
  });

  test('does not crash when onFileSelect is not provided', () => {
    render(<FileList files={mockFiles} />);
    
    // Should not throw error when clicked
    expect(() => {
      fireEvent.click(screen.getByText('test.js'));
    }).not.toThrow();
  });
});

// Run tests with: npm test
// Run with coverage: npm test -- --coverage
// Run specific test: npm test -- --testNamePattern="FileList"
```

**File: `src/setupTests.js` (required for @testing-library/jest-dom)**
```javascript
import '@testing-library/jest-dom';
```

#### API Client Testing
```javascript
// src/services/api.test.js
import { uploadFile, getFiles } from './api';

// Mock fetch globally
global.fetch = jest.fn();

describe('API Service', () => {
  beforeEach(() => {
    fetch.mockClear();
  });

  test('uploadFile sends correct request', async () => {
    const mockFile = new File(['content'], 'test.txt', { type: 'text/plain' });
    const mockResponse = { success: true, id: 123 };
    
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve(mockResponse)
    });

    const result = await uploadFile(mockFile);

    expect(fetch).toHaveBeenCalledWith('/api/upload', {
      method: 'POST',
      body: expect.any(FormData)
    });
    expect(result).toEqual(mockResponse);
  });

  test('getFiles handles API errors', async () => {
    fetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error'
    });

    await expect(getFiles()).rejects.toThrow('HTTP error! status: 500');
  });
});
```

### Requirement Tracking in Vitest

Tag tests with `[REQ:ID]` to automatically track requirement coverage:

```typescript
import RequirementReporter from '@vrooli/vitest-requirement-reporter';

// vite.config.ts
export default defineConfig({
  test: {
    reporters: [
      'default',
      new RequirementReporter({
        outputFile: 'coverage/vitest-requirements.json',
        emitStdout: true,  // Required for phase integration
        verbose: true,
      }),
    ],
  },
});

// projectStore.test.ts
describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  it('fetches projects', () => { ... });      // Inherits REQ tag
  it('creates project', () => { ... });       // Inherits REQ tag
});
```

**See Also:**
- [@vrooli/vitest-requirement-reporter](../../../packages/vitest-requirement-reporter/README.md) - Package documentation
- [Requirement Tracking Guide](requirement-tracking.md) - Complete system overview

#### Vitest Alternative
```javascript
// vitest.config.js
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    coverage: {
      provider: 'c8',
      reporter: ['text', 'html'],
      thresholds: {
        branches: 80,
        functions: 80,
        lines: 80,
        statements: 80
      }
    }
  }
});
```

### Python Unit Testing

#### pytest Configuration
```ini
# pytest.ini
[tool:pytest]
testpaths = tests
python_files = test_*.py *_test.py
python_classes = Test*
python_functions = test_*
addopts = 
    --cov=src
    --cov-report=html
    --cov-report=term
    --cov-fail-under=80
    --strict-markers
markers =
    slow: marks tests as slow (deselect with '-m "not slow"')
    integration: marks tests as integration tests
```

#### Complete Working Example: File Processing Service

**File: `src/file_processor.py` (implementation being tested)**
```python
import os
import hashlib
from typing import List, Dict, Optional
from pathlib import Path

class FileProcessingError(Exception):
    """Custom exception for file processing errors."""
    pass

class FileProcessor:
    """Handles file processing and validation for the visited tracker."""
    
    def __init__(self, base_dir: str = "/tmp/visited"):
        self.base_dir = Path(base_dir)
        self.base_dir.mkdir(exist_ok=True)
        self.processed_files = {}
    
    def process_files(self, files: List[str]) -> Dict[str, any]:
        """Process a list of files and return processing results."""
        if not files:
            return {"count": 0, "files": [], "status": "empty"}
        
        valid_files = []
        errors = []
        
        for file in files:
            if self.is_valid_file(file):
                valid_files.append(file)
                self.processed_files[file] = self._generate_hash(file)
            else:
                errors.append(f"Invalid file: {file}")
        
        if errors and not valid_files:
            raise FileProcessingError(f"No valid files found: {', '.join(errors)}")
        
        return {
            "count": len(valid_files),
            "files": valid_files,
            "errors": errors,
            "status": "success" if valid_files else "failed"
        }
    
    def is_valid_file(self, filename: str) -> bool:
        """Check if file has valid extension."""
        if not filename or not isinstance(filename, str):
            return False
        
        valid_extensions = {'.py', '.js', '.go', '.md', '.json'}
        file_path = Path(filename)
        return file_path.suffix.lower() in valid_extensions
    
    def read_file_content(self, filename: str) -> str:
        """Read content from a file safely."""
        if not self.is_valid_file(filename):
            raise FileProcessingError(f"Invalid file type: {filename}")
        
        file_path = self.base_dir / filename
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                return f.read()
        except FileNotFoundError:
            raise FileProcessingError(f"File not found: {filename}")
        except IOError as e:
            raise FileProcessingError(f"Error reading file {filename}: {e}")
    
    def save_file(self, filename: str, content: str) -> bool:
        """Save content to a file."""
        if not self.is_valid_file(filename):
            raise FileProcessingError(f"Invalid file type: {filename}")
        
        file_path = self.base_dir / filename
        try:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            return True
        except IOError as e:
            raise FileProcessingError(f"Error saving file {filename}: {e}")
    
    def _generate_hash(self, filename: str) -> str:
        """Generate hash for file tracking."""
        return hashlib.md5(filename.encode()).hexdigest()
    
    def get_stats(self) -> Dict[str, any]:
        """Get processing statistics."""
        return {
            "processed_count": len(self.processed_files),
            "base_dir": str(self.base_dir),
            "files": list(self.processed_files.keys())
        }

# Standalone function for backward compatibility
def process_files(files: List[str]) -> Dict[str, any]:
    """Legacy function wrapper for file processing."""
    processor = FileProcessor()
    return processor.process_files(files)

def is_python_file(filename: str) -> bool:
    """Check if file is a Python file."""
    if not filename:
        return False
    return filename.lower().endswith('.py')
```

**File: `tests/test_file_processor.py` (complete test suite)**
```python
import pytest
import tempfile
import shutil
from pathlib import Path
from unittest.mock import patch, mock_open, MagicMock
from src.file_processor import FileProcessor, FileProcessingError, process_files, is_python_file

class TestFileProcessor:
    """Test suite for FileProcessor class."""
    
    @pytest.fixture
    def temp_dir(self):
        """Create temporary directory for testing."""
        temp_path = tempfile.mkdtemp()
        yield temp_path
        shutil.rmtree(temp_path)
    
    @pytest.fixture
    def processor(self, temp_dir):
        """Create FileProcessor instance with temp directory."""
        return FileProcessor(temp_dir)
    
    def test_init_creates_directory(self, temp_dir):
        """Test that FileProcessor creates base directory."""
        new_dir = Path(temp_dir) / "new_subdir"
        processor = FileProcessor(str(new_dir))
        
        assert new_dir.exists()
        assert processor.base_dir == new_dir
    
    def test_process_empty_list(self, processor):
        """Test processing empty file list."""
        result = processor.process_files([])
        
        assert result == {
            "count": 0,
            "files": [],
            "status": "empty"
        }
    
    def test_process_valid_files(self, processor):
        """Test processing valid files."""
        files = ["main.py", "utils.py", "config.json"]
        result = processor.process_files(files)
        
        assert result["count"] == 3
        assert result["files"] == files
        assert result["status"] == "success"
        assert len(result["errors"]) == 0
    
    def test_process_mixed_files(self, processor):
        """Test processing mix of valid and invalid files."""
        files = ["main.py", "invalid.txt", "config.json"]
        result = processor.process_files(files)
        
        assert result["count"] == 2
        assert "main.py" in result["files"]
        assert "config.json" in result["files"]
        assert "invalid.txt" not in result["files"]
        assert len(result["errors"]) == 1
        assert "Invalid file: invalid.txt" in result["errors"]
    
    def test_process_all_invalid_files_raises_error(self, processor):
        """Test that all invalid files raise exception."""
        files = ["invalid.txt", "bad.exe"]
        
        with pytest.raises(FileProcessingError) as exc_info:
            processor.process_files(files)
        
        assert "No valid files found" in str(exc_info.value)
    
    @pytest.mark.parametrize("filename,expected", [
        ("test.py", True),
        ("app.js", True),
        ("main.go", True),
        ("README.md", True),
        ("config.json", True),
        ("app.txt", False),
        ("", False),
        ("file.PY", True),  # Case insensitive
        ("script.JS", True),
        ("data.xml", False),
    ])
    def test_is_valid_file(self, processor, filename, expected):
        """Test file validation with various extensions."""
        assert processor.is_valid_file(filename) == expected
    
    def test_is_valid_file_with_invalid_input(self, processor):
        """Test file validation with invalid inputs."""
        assert processor.is_valid_file(None) is False
        assert processor.is_valid_file(123) is False
        assert processor.is_valid_file([]) is False
    
    def test_read_file_content_success(self, processor, temp_dir):
        """Test successful file reading."""
        test_file = Path(temp_dir) / "test.py"
        test_content = "print('Hello, World!')\nprint('Testing')"
        test_file.write_text(test_content)
        
        content = processor.read_file_content("test.py")
        assert content == test_content
    
    def test_read_file_content_file_not_found(self, processor):
        """Test reading non-existent file."""
        with pytest.raises(FileProcessingError) as exc_info:
            processor.read_file_content("nonexistent.py")
        
        assert "File not found: nonexistent.py" in str(exc_info.value)
    
    def test_read_file_content_invalid_type(self, processor):
        """Test reading invalid file type."""
        with pytest.raises(FileProcessingError) as exc_info:
            processor.read_file_content("invalid.txt")
        
        assert "Invalid file type: invalid.txt" in str(exc_info.value)
    
    @patch("builtins.open", side_effect=IOError("Permission denied"))
    def test_read_file_content_io_error(self, mock_open, processor):
        """Test handling I/O errors during file reading."""
        with pytest.raises(FileProcessingError) as exc_info:
            processor.read_file_content("test.py")
        
        assert "Error reading file test.py" in str(exc_info.value)
    
    def test_save_file_success(self, processor, temp_dir):
        """Test successful file saving."""
        filename = "output.py"
        content = "# Generated code\nprint('success')"
        
        result = processor.save_file(filename, content)
        
        assert result is True
        saved_content = (Path(temp_dir) / filename).read_text()
        assert saved_content == content
    
    def test_save_file_invalid_type(self, processor):
        """Test saving invalid file type."""
        with pytest.raises(FileProcessingError) as exc_info:
            processor.save_file("output.txt", "content")
        
        assert "Invalid file type: output.txt" in str(exc_info.value)
    
    @patch("builtins.open", side_effect=IOError("Disk full"))
    def test_save_file_io_error(self, mock_open, processor):
        """Test handling I/O errors during file saving."""
        with pytest.raises(FileProcessingError) as exc_info:
            processor.save_file("test.py", "content")
        
        assert "Error saving file test.py" in str(exc_info.value)
    
    def test_generate_hash(self, processor):
        """Test hash generation for file tracking."""
        hash1 = processor._generate_hash("test.py")
        hash2 = processor._generate_hash("test.py")
        hash3 = processor._generate_hash("different.py")
        
        assert hash1 == hash2  # Same input, same hash
        assert hash1 != hash3  # Different input, different hash
        assert len(hash1) == 32  # MD5 hash length
    
    def test_get_stats_empty(self, processor):
        """Test statistics for empty processor."""
        stats = processor.get_stats()
        
        assert stats["processed_count"] == 0
        assert stats["files"] == []
        assert "base_dir" in stats
    
    def test_get_stats_with_files(self, processor):
        """Test statistics after processing files."""
        files = ["test1.py", "test2.js"]
        processor.process_files(files)
        
        stats = processor.get_stats()
        
        assert stats["processed_count"] == 2
        assert set(stats["files"]) == set(files)
    
    def test_processed_files_tracking(self, processor):
        """Test that processed files are tracked correctly."""
        files = ["main.py", "utils.js"]
        result = processor.process_files(files)
        
        assert len(processor.processed_files) == 2
        assert "main.py" in processor.processed_files
        assert "utils.js" in processor.processed_files
        
        # Hashes should be generated
        for file in files:
            assert isinstance(processor.processed_files[file], str)
            assert len(processor.processed_files[file]) == 32


class TestStandaloneFunctions:
    """Test standalone function compatibility."""
    
    def test_process_files_function(self):
        """Test standalone process_files function."""
        files = ["test.py", "app.js"]
        result = process_files(files)
        
        assert result["count"] == 2
        assert result["files"] == files
        assert result["status"] == "success"
    
    @pytest.mark.parametrize("filename,expected", [
        ("test.py", True),
        ("app.js", False),
        ("script.PY", True),
        ("", False),
        ("file.txt", False),
    ])
    def test_is_python_file_function(self, filename, expected):
        """Test standalone is_python_file function."""
        assert is_python_file(filename) == expected


# Integration tests that could be in a separate file
class TestFileProcessorIntegration:
    """Integration tests for FileProcessor."""
    
    @pytest.fixture
    def temp_dir(self):
        """Create temporary directory for integration testing."""
        temp_path = tempfile.mkdtemp()
        yield temp_path
        shutil.rmtree(temp_path)
    
    def test_full_workflow(self, temp_dir):
        """Test complete file processing workflow."""
        processor = FileProcessor(temp_dir)
        
        # Save some files
        files_to_create = {
            "main.py": "def main():\n    print('Hello')",
            "utils.js": "function helper() { return true; }",
            "config.json": '{"setting": "value"}'
        }
        
        for filename, content in files_to_create.items():
            processor.save_file(filename, content)
        
        # Process the files
        result = processor.process_files(list(files_to_create.keys()))
        
        # Verify results
        assert result["count"] == 3
        assert result["status"] == "success"
        
        # Read back and verify
        for filename, expected_content in files_to_create.items():
            actual_content = processor.read_file_content(filename)
            assert actual_content == expected_content
        
        # Check statistics
        stats = processor.get_stats()
        assert stats["processed_count"] == 3
        assert set(stats["files"]) == set(files_to_create.keys())


# Performance tests (marked as slow)
class TestFileProcessorPerformance:
    """Performance tests for FileProcessor."""
    
    @pytest.mark.slow
    def test_large_file_list_performance(self):
        """Test processing performance with large file lists."""
        import time
        
        # Generate large list of files
        large_file_list = [f"file_{i}.py" for i in range(1000)]
        
        processor = FileProcessor()
        start_time = time.time()
        
        result = processor.process_files(large_file_list)
        
        end_time = time.time()
        processing_time = end_time - start_time
        
        # Should process 1000 files in under 1 second
        assert processing_time < 1.0
        assert result["count"] == 1000
        assert result["status"] == "success"

# Run tests with: python -m pytest tests/test_file_processor.py -v
# Run with coverage: python -m pytest tests/test_file_processor.py --cov=src --cov-report=html
# Run excluding slow tests: python -m pytest tests/test_file_processor.py -m "not slow"
```

**File: `tests/conftest.py` (shared test configuration)**
```python
import pytest
import tempfile
import shutil
from pathlib import Path

@pytest.fixture(scope="session")
def test_data_dir():
    """Create test data directory for the session."""
    test_dir = tempfile.mkdtemp(prefix="fileprocessor_test_")
    yield Path(test_dir)
    shutil.rmtree(test_dir)

@pytest.fixture
def sample_files(test_data_dir):
    """Create sample files for testing."""
    files = {
        "sample.py": "# Python sample\nprint('test')",
        "sample.js": "// JavaScript sample\nconsole.log('test');",
        "sample.json": '{"test": true}',
        "README.md": "# Test README\nThis is a test."
    }
    
    created_files = {}
    for filename, content in files.items():
        file_path = test_data_dir / filename
        file_path.write_text(content)
        created_files[filename] = file_path
    
    return created_files

# Custom markers for test categorization
def pytest_configure(config):
    """Configure custom pytest markers."""
    config.addinivalue_line(
        "markers", "slow: marks tests as slow (deselect with '-m \"not slow\"')"
    )
    config.addinivalue_line(
        "markers", "integration: marks tests as integration tests"
    )
```

#### Class Testing
```python
# tests/test_file_manager.py
import pytest
from unittest.mock import Mock, patch
from src.file_manager import FileManager, FileNotFoundError

class TestFileManager:
    @pytest.fixture
    def manager(self):
        return FileManager(base_path="/tmp/test")

    def test_init_sets_base_path(self, manager):
        assert manager.base_path == "/tmp/test"

    def test_add_file_success(self, manager):
        file_data = {"name": "test.py", "content": "print('test')"}
        
        result = manager.add_file(file_data)
        
        assert result["status"] == "success"
        assert "test.py" in manager.files

    def test_get_nonexistent_file_raises_error(self, manager):
        with pytest.raises(FileNotFoundError):
            manager.get_file("nonexistent.py")

    @patch('os.path.exists')
    def test_file_exists_check(self, mock_exists, manager):
        mock_exists.return_value = True
        
        assert manager.file_exists("test.py") is True
        mock_exists.assert_called_with("/tmp/test/test.py")
```

#### Coverage Commands
```bash
# Run tests with coverage
pytest --cov=src --cov-report=html --cov-report=term

# Run specific test markers
pytest -m "not slow"  # Skip slow tests
pytest -m integration  # Only integration tests

# Check coverage and fail if below threshold
pytest --cov=src --cov-fail-under=80

# Generate coverage report
coverage html
coverage report --show-missing
```

## 🔧 Common Testing Patterns

### Mocking External Dependencies
```go
// Go: Interface-based mocking
type FileStorage interface {
    Save(filename string, data []byte) error
    Load(filename string) ([]byte, error)
}

type MockStorage struct {
    files map[string][]byte
}

func (m *MockStorage) Save(filename string, data []byte) error {
    m.files[filename] = data
    return nil
}

func TestWithMockStorage(t *testing.T) {
    storage := &MockStorage{files: make(map[string][]byte)}
    processor := NewFileProcessor(storage)
    
    err := processor.ProcessFile("test.txt", []byte("content"))
    assert.NoError(t, err)
}
```

```javascript
// JavaScript: Jest mocking
jest.mock('../services/api', () => ({
  uploadFile: jest.fn(),
  getFiles: jest.fn()
}));

import { uploadFile } from '../services/api';

test('component calls API correctly', async () => {
  uploadFile.mockResolvedValue({ success: true });
  
  // Test code that uses uploadFile
  
  expect(uploadFile).toHaveBeenCalledWith(expectedData);
});
```

```python
# Python: unittest.mock
from unittest.mock import patch, Mock

@patch('src.file_manager.requests.post')
def test_upload_file(self, mock_post):
    mock_post.return_value.json.return_value = {"success": True}
    
    manager = FileManager()
    result = manager.upload("test.txt")
    
    assert result["success"] is True
    mock_post.assert_called_once()
```

### Test Data Factories
```go
// Go: Test helpers
func createTestFile(t *testing.T, name string, size int) File {
    return File{
        Name: name,
        Size: size,
        CreatedAt: time.Now(),
    }
}

func TestMultipleFiles(t *testing.T) {
    files := []File{
        createTestFile(t, "test1.go", 100),
        createTestFile(t, "test2.go", 200),
    }
    // Test with files...
}
```

```javascript
// JavaScript: Factory functions
const createMockFile = (overrides = {}) => ({
  id: 1,
  name: 'test.js',
  size: 1024,
  type: 'application/javascript',
  ...overrides
});

test('processes different file types', () => {
  const jsFile = createMockFile();
  const pyFile = createMockFile({ name: 'test.py', type: 'text/x-python' });
  
  // Test with different files...
});
```

```python
# Python: pytest fixtures
@pytest.fixture
def sample_file():
    return {
        "name": "test.py",
        "content": "print('hello')",
        "size": 13
    }

@pytest.fixture
def file_manager():
    return FileManager(base_path="/tmp/test")

def test_add_file(file_manager, sample_file):
    result = file_manager.add_file(sample_file)
    assert result["status"] == "success"
```

## 📊 Coverage Best Practices

### Achieving High Coverage
1. **Start with happy paths** - Test main functionality first
2. **Add error cases** - Test invalid inputs, network failures, etc.
3. **Edge cases** - Empty inputs, boundary conditions, null values
4. **Integration points** - Where your code calls external dependencies

### Coverage Targets by Component Type
| Component | Minimum | Target | Notes |
|-----------|---------|--------|-------|
| Business Logic | 90% | 95% | Core algorithms, data processing |
| API Handlers | 80% | 90% | HTTP endpoints, request validation |
| UI Components | 70% | 85% | React/Vue components, user interactions |
| Utilities | 85% | 95% | Helper functions, data transformations |
| Models/DTOs | 60% | 80% | Data structures, simple getters/setters |

### What NOT to Test in Unit Tests
- External API calls (use integration tests)
- Database operations (use integration tests)
- File system operations (mock them)
- Network requests (mock them)
- Complex UI interactions (use E2E tests)

## 🚨 Common Pitfalls

### 1. Testing Implementation Details
```javascript
// ❌ BAD - Testing internal state
test('component uses useState', () => {
  const wrapper = shallow(<MyComponent />);
  expect(wrapper.state('isLoading')).toBe(false);
});

// ✅ GOOD - Testing behavior
test('component shows loading indicator', () => {
  render(<MyComponent loading={true} />);
  expect(screen.getByText('Loading...')).toBeInTheDocument();
});
```

### 2. Flaky Tests with Timing
```go
// ❌ BAD - Race conditions
func TestAsync(t *testing.T) {
    go processAsync()
    time.Sleep(100 * time.Millisecond) // Flaky!
    assert.True(t, isProcessed)
}

// ✅ GOOD - Synchronous or proper waiting
func TestAsync(t *testing.T) {
    done := make(chan bool)
    go func() {
        processAsync()
        done <- true
    }()
    
    select {
    case <-done:
        assert.True(t, isProcessed)
    case <-time.After(5 * time.Second):
        t.Fatal("Test timed out")
    }
}
```

### 3. Overly Complex Test Setup
```python
# ❌ BAD - Too much setup
def test_process_file():
    # 50 lines of setup...
    manager = FileManager()
    # More setup...
    result = manager.process("test.py")
    assert result.success

# ✅ GOOD - Simple, focused tests
@pytest.fixture
def configured_manager():
    return FileManager(config=test_config)

def test_process_file(configured_manager):
    result = configured_manager.process("test.py")
    assert result.success
```

## See Also

### Related Guides
- [Phased Testing](phased-testing.md) - How unit phase fits into 7-phase architecture
- [Requirements Sync](requirements-sync.md) - Link tests to requirements with `[REQ:ID]`
- [CLI Testing](cli-testing.md) - BATS testing for command-line tools
- [UI Testability](ui-testability.md) - Writing testable UIs

### Reference
- [Test Runners](../reference/test-runners.md) - Language-specific test runners
- [Phase Catalog](../reference/phase-catalog.md) - All phase definitions
- [Presets](../reference/presets.md) - Quick/Smoke/Comprehensive presets

### Concepts
- [Architecture](../concepts/architecture.md) - Go orchestrator design