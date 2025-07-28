# Agent S2 Refactoring Plan

## 🚀 Overview
This document outlines the comprehensive refactoring plan for the Agent S2 resource to improve code organization, reduce duplication, and enhance maintainability.

## Current Issues Summary
1. **Duplication**: Test outputs, configuration, client code
2. **Monolithic Code**: 1000+ line API server needs splitting
3. **Inconsistent Organization**: Mix of well-organized (shell libs) and poorly organized (Python) code
4. **Configuration Scatter**: Constants and settings spread across many files
5. **Unclear Hierarchy**: Examples, tests, and documentation lack clear structure
6. **No Code Reuse**: Each example reimplements common functionality
7. **Mixed Concerns**: Docker directory contains too many different types of files

## Phase 1: Core Restructuring (Foundation)

### 1.1 Create Proper Python Package Structure
```
agent-s2/
├── agent_s2/                    # Main Python package
│   ├── __init__.py
│   ├── config.py               # Centralized configuration
│   ├── client/                 # API client library
│   │   ├── __init__.py
│   │   ├── base.py            # Base HTTP client
│   │   ├── screenshot.py      # Screenshot client
│   │   ├── automation.py      # Automation client
│   │   └── ai.py              # AI-driven client
│   ├── server/                 # API server modules
│   │   ├── __init__.py
│   │   ├── app.py             # FastAPI/Flask app initialization
│   │   ├── routes/            # API endpoints
│   │   │   ├── health.py
│   │   │   ├── screenshot.py
│   │   │   ├── mouse.py
│   │   │   ├── keyboard.py
│   │   │   └── ai.py
│   │   └── services/          # Business logic
│   │       ├── display.py     # X11/display management
│   │       ├── capture.py     # Screenshot capture logic
│   │       ├── automation.py  # PyAutoGUI wrapper
│   │       └── ai_handler.py  # AI integration logic
│   └── utils/                  # Shared utilities
│       ├── __init__.py
│       ├── image.py           # Image processing utilities
│       ├── validation.py      # Input validation
│       └── constants.py       # Shared constants
```

### 1.2 Reorganize Docker Structure
```
docker/
├── images/                     # Docker images
│   └── agent-s2/
│       ├── Dockerfile
│       └── requirements.txt
├── config/                     # Runtime configuration
│   ├── supervisor.conf
│   └── xvfb.conf
├── scripts/                    # Container scripts
│   ├── entrypoint.sh
│   ├── startup.sh
│   └── vnc-password.sh
└── compose/                    # Docker Compose files
    ├── docker-compose.yml     # Default development
    └── docker-compose.prod.yml
```

### 1.3 Consolidate Examples and Testing
```
examples/
├── 01-getting-started/         # Numbered progression
│   ├── README.md
│   ├── hello_screenshot.py    # Minimal example
│   └── requirements.txt
├── 02-basic-automation/
│   ├── mouse_control.py
│   ├── keyboard_input.py
│   └── combined_demo.py
├── 03-advanced-features/
│   ├── region_capture.py
│   ├── continuous_capture.py
│   └── performance_test.py
├── 04-ai-integration/
│   ├── ai_automation.py
│   ├── task_completion.py
│   └── ai_examples.md
├── setup-demo-environment.sh
└── run-all-examples.sh

testing/                        # Separate from examples
├── test-outputs/              # Single output location
│   └── .gitkeep
├── integration/
│   ├── test_api.py
│   ├── test_screenshots.py
│   └── test_automation.py
└── cleanup.sh
```

## Phase 2: Configuration Centralization

### 2.1 Create Central Configuration Module
```python
# agent_s2/config.py
from typing import Optional
import os

class Config:
    # API Configuration
    API_HOST = os.getenv("AGENT_S2_HOST", "0.0.0.0")
    API_PORT = int(os.getenv("AGENT_S2_PORT", "4113"))
    API_BASE_URL = f"http://localhost:{API_PORT}"
    
    # Display Configuration  
    DISPLAY = os.getenv("DISPLAY", ":99")
    SCREEN_WIDTH = int(os.getenv("SCREEN_WIDTH", "1920"))
    SCREEN_HEIGHT = int(os.getenv("SCREEN_HEIGHT", "1080"))
    
    # VNC Configuration
    VNC_PASSWORD = os.getenv("VNC_PASSWORD", "agents2vnc")
    VNC_PORT = int(os.getenv("VNC_PORT", "5900"))
    
    # AI Configuration
    AI_API_URL = os.getenv("AI_API_URL", "http://localhost:11434/api/chat")
    AI_MODEL = os.getenv("AI_MODEL", "llama3.2-vision:11b")
    
    # Output Configuration
    OUTPUT_DIR = os.getenv("AGENT_S2_OUTPUT_DIR", "/tmp/agent-s2-outputs")
```

### 2.2 Environment File Template
```bash
# .env.example
AGENT_S2_PORT=4113
AGENT_S2_HOST=0.0.0.0
VNC_PASSWORD=agents2vnc
AI_API_URL=http://localhost:11434/api/chat
AI_MODEL=llama3.2-vision:11b
```

## Phase 3: API Server Modularization

### 3.1 Split Monolithic API Server
Break the 1040-line `api-server.py` into:

1. **Main Application** (`app.py`) - 50 lines
2. **Route Handlers** (`routes/`) - 200 lines each max
3. **Service Layer** (`services/`) - Business logic
4. **Middleware** (`middleware/`) - CORS, logging, error handling
5. **Models** (`models/`) - Pydantic models for requests/responses

### 3.2 Create Shared Client Library
```python
# agent_s2/client/base.py
class AgentS2Client:
    def __init__(self, base_url: str = None):
        self.base_url = base_url or Config.API_BASE_URL
        self.session = requests.Session()
    
    def screenshot(self, format="png", quality=95, region=None):
        """Unified screenshot method"""
        ...
```

## Phase 4: Shell Script Improvements

### 4.1 Consolidate Shell Libraries
```
lib/
├── core/
│   ├── config.sh       # Configuration loading
│   ├── logging.sh      # Unified logging
│   └── validation.sh   # Input validation
├── docker/
│   ├── container.sh    # Container management
│   └── network.sh      # Network utilities
└── utils/
    ├── colors.sh       # Terminal colors
    └── helpers.sh      # General helpers
```

## Phase 5: Documentation Restructuring

### 5.1 Documentation Hierarchy
```
docs/
├── README.md           # Quick start (200 lines max)
├── installation/
│   ├── requirements.md
│   ├── docker.md
│   └── manual.md
├── usage/
│   ├── getting-started.md
│   ├── api-reference.md
│   ├── examples.md
│   └── troubleshooting.md
├── development/
│   ├── architecture.md
│   ├── contributing.md
│   └── testing.md
└── api/
    └── openapi.yaml    # OpenAPI specification
```

## Phase 6: Testing and Quality

### 6.1 Automated Testing Structure
```
tests/
├── unit/               # Unit tests
├── integration/        # Integration tests
├── e2e/               # End-to-end tests
├── fixtures/          # Test fixtures
└── conftest.py        # Pytest configuration
```

### 6.2 CI/CD Configuration
```yaml
# .github/workflows/agent-s2.yml
name: Agent S2 Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Agent S2 Tests
        run: |
          cd scripts/resources/agents/agent-s2
          ./manage.sh --action test
```

## Phase 7: Migration Strategy

### 7.1 Backward Compatibility
1. Keep existing `manage.sh` as main entry point
2. Create compatibility layer for old example scripts
3. Provide migration guide for existing users
4. Maintain old API endpoints with deprecation warnings

### 7.2 Gradual Migration Path
1. **Week 1**: Set up new directory structure
2. **Week 2**: Modularize Python code
3. **Week 3**: Migrate examples to new structure
4. **Week 4**: Update documentation
5. **Week 5**: Add tests and CI/CD
6. **Week 6**: Deprecate old structure

## Phase 8: New Features from Refactoring

### 8.1 Package Installation
```bash
# Make agent-s2 pip-installable
pip install -e ./scripts/resources/agents/agent-s2
```

### 8.2 CLI Enhancement
```bash
# New CLI commands
agent-s2 screenshot --region 0,0,800,600
agent-s2 automate --script automation.yaml
agent-s2 server --port 4113
```

### 8.3 Configuration Management
```bash
# New configuration commands
agent-s2 config set api.port 4113
agent-s2 config get vnc.password
agent-s2 config validate
```

## Implementation Priority

1. **High Priority** (Phase 1-2)
   - Python package structure
   - Configuration centralization
   - Fix duplicate test-outputs

2. **Medium Priority** (Phase 3-4)
   - API server modularization
   - Client library creation
   - Shell script improvements

3. **Low Priority** (Phase 5-8)
   - Documentation restructuring
   - Advanced testing setup
   - CLI enhancements

## Expected Benefits

1. **Reduced Code Duplication**: ~40% less code through consolidation
2. **Improved Maintainability**: Clear separation of concerns
3. **Better Testing**: Easier to test individual components
4. **Enhanced User Experience**: Clear example progression
5. **Easier Contributions**: Well-organized codebase
6. **Performance**: Potential for caching and optimization
7. **Scalability**: Easier to add new features

## Implementation Notes

- Maintain backward compatibility throughout
- Test each phase thoroughly before moving to next
- Keep manage.sh working at all times
- Document all breaking changes
- Create migration guides for users