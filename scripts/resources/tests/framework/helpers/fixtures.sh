#!/bin/bash
# ====================================================================
# Test Fixture Generation
# ====================================================================
#
# Generates test data and fixtures for integration tests. Provides
# consistent, reusable test data across different test scenarios.
#
# Functions:
#   - generate_test_audio()       - Generate test audio files
#   - generate_test_image()       - Generate test images
#   - generate_test_document()    - Generate test documents
#   - generate_test_json()        - Generate test JSON data
#   - generate_test_workflow()    - Generate test workflow files
#   - create_test_fixtures()      - Create all test fixtures
#
# ====================================================================

# Base fixture directory
FIXTURES_DIR="${SCRIPT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}/fixtures"

# Generate test audio file (simple beep)
generate_test_audio() {
    local filename="${1:-test-sample.wav}"
    local duration="${2:-5}"
    local output_path="$FIXTURES_DIR/audio/$filename"
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$output_path")"
    
    # Generate a simple test audio file using ffmpeg if available
    if command -v ffmpeg >/dev/null 2>&1; then
        ffmpeg -f lavfi -i "sine=frequency=440:duration=$duration" -y "$output_path" >/dev/null 2>&1
    else
        # Create a minimal WAV header for testing (silent audio)
        {
            # WAV header for a simple silent audio file
            printf "RIFF"
            printf "\x24\x08\x00\x00"  # File size - 8
            printf "WAVE"
            printf "fmt "
            printf "\x10\x00\x00\x00"  # fmt chunk size
            printf "\x01\x00"          # PCM format
            printf "\x01\x00"          # 1 channel
            printf "\x40\x1F\x00\x00"  # 8000 Hz sample rate
            printf "\x80\x3E\x00\x00"  # Byte rate
            printf "\x02\x00"          # Block align
            printf "\x10\x00"          # 16 bits per sample
            printf "data"
            printf "\x00\x08\x00\x00"  # Data chunk size
            # Generate some sample data (silence)
            dd if=/dev/zero bs=2048 count=1 2>/dev/null
        } > "$output_path"
    fi
    
    echo "$output_path"
}

# Generate test image (simple colored rectangle)
generate_test_image() {
    local filename="${1:-test-image.png}"
    local width="${2:-800}"
    local height="${3:-600}"
    local output_path="$FIXTURES_DIR/images/$filename"
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$output_path")"
    
    # Generate a simple test image using ImageMagick if available
    if command -v convert >/dev/null 2>&1; then
        convert -size "${width}x${height}" xc:lightblue -fill darkblue \
            -pointsize 48 -gravity center \
            -annotate +0+0 "Test Image\n${width}x${height}" \
            "$output_path" 2>/dev/null
    else
        # Create a minimal PNG for testing
        cat > "$output_path" << 'EOF'
iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI/hXlfZQAAAABJRU5ErkJggg==
EOF
        base64 -d "$output_path" > "${output_path}.tmp" && mv "${output_path}.tmp" "$output_path" 2>/dev/null || true
    fi
    
    echo "$output_path"
}

# Generate test document
generate_test_document() {
    local filename="${1:-test-document.txt}"
    local content_type="${2:-text}"
    local output_path="$FIXTURES_DIR/documents/$filename"
    
    # Create directory if it doesn't exist
    mkdir -p "$(dirname "$output_path")"
    
    case "$content_type" in
        "text")
            cat > "$output_path" << EOF
# Test Document

This is a test document for integration testing.

## Meeting Notes

- Discussed project timeline
- Reviewed resource requirements
- Planned next steps

## Action Items

1. Implement new features
2. Test integration points
3. Deploy to staging environment

Generated on: $(date)
Test ID: $(date +%s)
EOF
            ;;
        "json")
            cat > "$output_path" << EOF
{
  "test_document": {
    "id": "test_$(date +%s)",
    "title": "Integration Test Document",
    "created": "$(date -Iseconds)",
    "content": {
      "sections": [
        {
          "title": "Overview",
          "text": "This is a test document for integration testing."
        },
        {
          "title": "Data",
          "items": ["item1", "item2", "item3"]
        }
      ]
    },
    "metadata": {
      "source": "integration_test",
      "version": "1.0"
    }
  }
}
EOF
            ;;
        "markdown")
            cat > "$output_path" << EOF
# Integration Test Document

## Overview

This is a test document for **integration testing** of Vrooli resources.

## Features

- Resource discovery
- Health checking
- Integration workflows

## Code Example

\`\`\`bash
./run.sh --verbose --resource ollama
\`\`\`

## Checklist

- [x] Basic functionality
- [ ] Error handling
- [ ] Performance testing

---

*Generated on $(date)*
EOF
            ;;
    esac
    
    echo "$output_path"
}

# Generate test JSON data
generate_test_json() {
    local filename="${1:-test-data.json}"
    local data_type="${2:-generic}"
    local output_path="$FIXTURES_DIR/documents/$filename"
    
    mkdir -p "$(dirname "$output_path")"
    
    case "$data_type" in
        "generic")
            cat > "$output_path" << EOF
{
  "test_id": "$(date +%s)",
  "timestamp": "$(date -Iseconds)",
  "data": {
    "string_field": "test value",
    "number_field": 42,
    "boolean_field": true,
    "array_field": [1, 2, 3, "test"],
    "object_field": {
      "nested": "value",
      "count": 5
    }
  },
  "metadata": {
    "source": "integration_test_fixture",
    "version": "1.0"
  }
}
EOF
            ;;
        "llm_prompt")
            cat > "$output_path" << EOF
{
  "model": "llama3.1:8b",
  "prompt": "Explain the concept of integration testing in software development. Include benefits and challenges.",
  "temperature": 0.3,
  "max_tokens": 500,
  "stream": false
}
EOF
            ;;
        "workflow")
            cat > "$output_path" << EOF
{
  "workflow": {
    "name": "Test Integration Workflow",
    "description": "Test workflow for integration testing",
    "steps": [
      {
        "id": "step1",
        "type": "data_input",
        "config": {
          "source": "test_data"
        }
      },
      {
        "id": "step2",
        "type": "process",
        "config": {
          "operation": "transform"
        }
      },
      {
        "id": "step3",
        "type": "output",
        "config": {
          "destination": "test_output"
        }
      }
    ]
  }
}
EOF
            ;;
    esac
    
    echo "$output_path"
}

# Generate test workflow files
generate_test_workflow() {
    local workflow_type="$1"
    local output_path="$FIXTURES_DIR/workflows/${workflow_type}-workflow.json"
    
    mkdir -p "$(dirname "$output_path")"
    
    case "$workflow_type" in
        "n8n")
            cat > "$output_path" << EOF
{
  "name": "Integration Test Workflow",
  "nodes": [
    {
      "parameters": {},
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "typeVersion": 1,
      "position": [250, 300]
    },
    {
      "parameters": {
        "url": "http://localhost:11434/api/generate",
        "options": {},
        "bodyParametersUi": {
          "parameter": [
            {
              "name": "model",
              "value": "llama3.1:8b"
            },
            {
              "name": "prompt", 
              "value": "Hello from integration test"
            }
          ]
        }
      },
      "name": "HTTP Request",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 1,
      "position": [450, 300]
    }
  ],
  "connections": {
    "Start": {
      "main": [
        [
          {
            "node": "HTTP Request",
            "type": "main",
            "index": 0
          }
        ]
      ]
    }
  }
}
EOF
            ;;
        "node-red")
            cat > "$output_path" << EOF
[
  {
    "id": "test-inject",
    "type": "inject",
    "name": "Test Trigger",
    "props": [
      {
        "p": "payload",
        "v": "Hello from integration test",
        "vt": "str"
      }
    ],
    "repeat": "",
    "crontab": "",
    "once": false,
    "x": 130,
    "y": 80
  },
  {
    "id": "test-http",
    "type": "http request",
    "name": "Ollama Request",
    "method": "POST",
    "ret": "obj",
    "url": "http://localhost:11434/api/generate",
    "payload": "{\"model\":\"llama3.1:8b\",\"prompt\":\"{{payload}}\"}",
    "x": 350,
    "y": 80
  },
  {
    "id": "test-debug",
    "type": "debug",
    "name": "Output",
    "active": true,
    "console": false,
    "complete": "payload",
    "x": 550,
    "y": 80
  }
]
EOF
            ;;
    esac
    
    echo "$output_path"
}

# Create all test fixtures
create_test_fixtures() {
    echo "📦 Creating test fixtures..."
    
    # Create audio fixtures
    generate_test_audio "sample-meeting.wav" 10
    generate_test_audio "short-clip.wav" 3
    
    # Create image fixtures
    generate_test_image "test-screenshot.png" 1920 1080
    generate_test_image "small-image.png" 400 300
    
    # Create document fixtures
    generate_test_document "meeting-notes.txt" "text"
    generate_test_document "test-data.json" "json"
    generate_test_document "readme.md" "markdown"
    
    # Create JSON fixtures
    generate_test_json "llm-prompt.json" "llm_prompt"
    generate_test_json "workflow-data.json" "workflow"
    
    # Create workflow fixtures
    generate_test_workflow "n8n"
    generate_test_workflow "node-red"
    
    echo "✓ Test fixtures created successfully"
}

# Get fixture path
get_fixture_path() {
    local category="$1"
    local filename="$2"
    
    echo "$FIXTURES_DIR/$category/$filename"
}

# Verify fixture exists
verify_fixture() {
    local fixture_path="$1"
    
    if [[ -f "$fixture_path" ]]; then
        return 0
    else
        echo "Fixture not found: $fixture_path"
        return 1
    fi
}

# Get random test data
get_random_test_data() {
    local data_type="$1"
    
    case "$data_type" in
        "string")
            echo "test_$(date +%s)_$RANDOM"
            ;;
        "number")
            echo "$((RANDOM % 1000))"
            ;;
        "email")
            echo "test_$(date +%s)@example.com"
            ;;
        "url")
            echo "https://example.com/test/$(date +%s)"
            ;;
        *)
            echo "random_data_$(date +%s)"
            ;;
    esac
}

# Create temporary test file
create_temp_test_file() {
    local content="$1"
    local extension="${2:-txt}"
    
    local temp_file="/tmp/vrooli_test_$(date +%s)_$$.${extension}"
    echo "$content" > "$temp_file"
    
    # Register for cleanup
    add_cleanup_file "$temp_file"
    
    echo "$temp_file"
}