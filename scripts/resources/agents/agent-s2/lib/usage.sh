#!/usr/bin/env bash
# Agent S2 Usage Examples
# Demonstrations and usage examples

#######################################
# Run usage example based on type
# Arguments:
#   $1 - usage type
# Returns: 0 if successful, 1 if failed
#######################################
agents2::run_usage_example() {
    local usage_type="${1:-help}"
    
    # Ensure service is running
    if ! agents2::is_running; then
        log::error "Agent S2 is not running. Start it with: $0 --action start"
        return 1
    fi
    
    # Ensure service is healthy
    if ! agents2::is_healthy; then
        log::error "Agent S2 is not healthy. Check logs with: $0 --action logs"
        return 1
    fi
    
    case "$usage_type" in
        "screenshot")
            agents2::usage_screenshot
            ;;
        "automation")
            agents2::usage_automation
            ;;
        "planning")
            agents2::usage_planning
            ;;
        "capabilities")
            agents2::usage_capabilities
            ;;
        "all")
            agents2::usage_all
            ;;
        "help"|*)
            agents2::usage_help
            ;;
    esac
}

#######################################
# Show usage help
#######################################
agents2::usage_help() {
    log::header "Agent S2 Usage Examples"
    echo
    echo "Available usage examples:"
    echo "  screenshot    - Demonstrate screenshot capture"
    echo "  automation    - Demonstrate GUI automation"
    echo "  planning      - Demonstrate multi-step planning"
    echo "  capabilities  - Show detailed capabilities"
    echo "  all          - Run all examples"
    echo "  help         - Show this help"
    echo
    echo "Run with: $0 --action usage --usage-type <type>"
    echo
    echo "Interactive mode: $0 --action usage --usage-type all"
}

#######################################
# Screenshot usage example
#######################################
agents2::usage_screenshot() {
    log::header "📸 Screenshot Usage Example"
    
    log::info "Taking a full screen screenshot..."
    if agents2::test_screenshot "agent-s2-screenshot.png"; then
        echo
        log::info "Screenshot saved to: agent-s2-screenshot.png"
        
        # Show how to use in code
        echo
        log::info "Example API usage:"
        cat << 'EOF'
# Take screenshot via API (JSON response with base64)
curl -X POST http://localhost:4113/screenshot?format=png

# Take screenshot as raw binary file (NEW)
curl -X POST "http://localhost:4113/screenshot?format=png&response_format=binary" \
  -o screenshot.png

# Extract PNG from JSON response  
curl -X POST http://localhost:4113/screenshot?format=png | \
  jq -r '.data' | sed 's/^data:image\/[^;]*;base64,//' | base64 -d > screenshot.png

# Take screenshot of specific region
curl -X POST http://localhost:4113/screenshot?format=png \
  -H "Content-Type: application/json" \
  -d '[100, 100, 800, 600]'

# Python example (JSON format)
import requests
import base64

response = requests.post(
    "http://localhost:4113/screenshot",
    params={"format": "png"}
)
if response.ok:
    data = response.json()
    image_data = data['data'].split(',')[1]
    with open('screenshot.png', 'wb') as f:
        f.write(base64.b64decode(image_data))

# Python example (binary format)
response = requests.post(
    "http://localhost:4113/screenshot",
    params={"format": "png", "response_format": "binary"}
)
if response.ok:
    with open('screenshot.png', 'wb') as f:
        f.write(response.content)
EOF
    fi
}

#######################################
# Automation usage example
#######################################
agents2::usage_automation() {
    log::header "🤖 Automation Usage Example"
    
    log::info "This example will demonstrate mouse and keyboard automation"
    log::warn "The automation will run in the virtual display (view via VNC)"
    echo
    
    # Show VNC connection info
    log::info "To watch the automation:"
    log::info "1. Connect to VNC: $AGENTS2_VNC_URL"
    log::info "2. Password: $AGENTS2_VNC_PASSWORD"
    echo
    
    read -p "Press Enter to start automation demo..."
    
    # Run automation sequence
    log::info "Running automation sequence..."
    agents2::test_automation_sequence
    
    echo
    log::info "Example API usage:"
    cat << 'EOF'
# Click at specific position
curl -X POST http://localhost:4113/mouse/click \
  -H "Content-Type: application/json" \
  -d '{
    "x": 500,
    "y": 300,
    "button": "left"
  }'

# Type text
curl -X POST http://localhost:4113/keyboard/type \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello, World!",
    "interval": 0.1
  }'

# Press key combination
curl -X POST http://localhost:4113/keyboard/press \
  -H "Content-Type: application/json" \
  -d '{
    "key": "Return"
  }'

# Press key with modifiers
curl -X POST http://localhost:4113/keyboard/press \
  -H "Content-Type: application/json" \
  -d '{
    "key": "c",
    "modifiers": ["ctrl"]
  }'

# Press hotkey combination
curl -X POST http://localhost:4113/keyboard/hotkey \
  -H "Content-Type: application/json" \
  -d '["ctrl", "alt", "t"]'
EOF
}

#######################################
# Planning usage example
#######################################
agents2::usage_planning() {
    log::header "🧠 Planning Usage Example"
    
    log::info "Agent S2 can plan and execute multi-step tasks"
    echo
    
    # Demonstrate async task with planning
    log::info "Submitting a complex task for planning..."
    
    local planning_request
    planning_request=$(cat <<'EOF'
{
    "task_type": "automation_sequence",
    "async_execution": true,
    "parameters": {
        "description": "Open a text editor and write a note",
        "steps": [
            {"type": "key_press", "parameters": {"keys": ["alt", "F2"]}},
            {"type": "wait", "parameters": {"seconds": 1}},
            {"type": "type_text", "parameters": {"text": "gedit"}},
            {"type": "key_press", "parameters": {"keys": ["Return"]}},
            {"type": "wait", "parameters": {"seconds": 2}},
            {"type": "type_text", "parameters": {"text": "This is an automated note created by Agent S2\n\nTimestamp: "}},
            {"type": "type_text", "parameters": {"text": "$(date)"}},
            {"type": "key_press", "parameters": {"keys": ["ctrl", "s"]}},
            {"type": "wait", "parameters": {"seconds": 1}},
            {"type": "type_text", "parameters": {"text": "agent-s2-note.txt"}},
            {"type": "key_press", "parameters": {"keys": ["Return"]}}
        ]
    }
}
EOF
)
    
    # Submit the task
    local response
    if response=$(echo "$planning_request" | agents2::api_request "POST" "/tasks/execute" -); then
        local task_id
        task_id=$(echo "$response" | jq -r '.task_id // empty' 2>/dev/null)
        
        if [[ -n "$task_id" ]]; then
            log::info "Task submitted with ID: $task_id"
            log::info "Monitoring task progress..."
            
            # Monitor progress
            local completed=false
            for i in {1..20}; do
                sleep 2
                local status_response
                if status_response=$(agents2::api_request "GET" "/tasks/$task_id"); then
                    local status
                    status=$(echo "$status_response" | jq -r '.status // "unknown"' 2>/dev/null)
                    log::info "Task status: $status"
                    
                    if [[ "$status" == "completed" || "$status" == "failed" ]]; then
                        completed=true
                        echo
                        echo "$status_response" | jq . 2>/dev/null || echo "$status_response"
                        break
                    fi
                fi
            done
            
            if ! $completed; then
                log::warn "Task is still running. Check status with:"
                log::info "curl http://localhost:4113/tasks/$task_id"
            fi
        fi
    fi
    
    echo
    log::info "Planning capabilities:"
    cat << 'EOF'
Agent S2 can:
- Break down complex tasks into steps
- Execute multi-step workflows
- Handle conditional logic
- Adapt to GUI changes
- Learn from past executions

Integration with LLMs enables:
- Natural language task descriptions
- Intelligent error handling
- Context-aware automation
- Cross-application workflows
EOF
}

#######################################
# Show capabilities
#######################################
agents2::usage_capabilities() {
    log::header "🚀 Agent S2 Capabilities"
    
    log::info "Fetching capabilities from API..."
    
    if agents2::api_request "GET" "/capabilities" | jq . 2>/dev/null; then
        echo
        log::info "Agent S2 provides a comprehensive set of computer interaction capabilities"
        log::info "All operations run in a secure, sandboxed environment"
    else
        log::error "Failed to fetch capabilities"
    fi
    
    echo
    log::info "Key Features:"
    cat << 'EOF'
1. Visual Automation
   - Screenshot capture (full screen or regions)
   - Visual element detection
   - Screen recording capabilities

2. Input Control
   - Mouse movement and clicks
   - Keyboard input and shortcuts
   - Drag and drop operations
   - Scrolling and gestures

3. Task Execution
   - Synchronous and asynchronous execution
   - Multi-step automation sequences
   - Conditional logic support
   - Error handling and recovery

4. Integration
   - RESTful API interface
   - LLM provider support (OpenAI, Anthropic, etc.)
   - VNC access for monitoring
   - Docker containerization

5. Security
   - Isolated virtual display
   - Sandboxed execution environment
   - Application access restrictions
   - Non-root user execution
EOF
}

#######################################
# Run all examples
#######################################
agents2::usage_all() {
    log::header "🎯 Running All Agent S2 Examples"
    
    local examples=("capabilities" "screenshot" "automation" "planning")
    
    for example in "${examples[@]}"; do
        echo
        log::info "Running example: $example"
        echo "Press Enter to continue or Ctrl+C to skip..."
        read -r
        
        agents2::usage_"$example"
        
        echo
        log::info "Example completed. Press Enter to continue..."
        read -r
    done
    
    echo
    log::success "All examples completed!"
    log::info "For more information, visit: https://www.simular.ai/agent-s"
}

#######################################
# Show example code snippets
#######################################
agents2::show_code_examples() {
    log::header "📝 Code Examples"
    
    cat << 'EOF'
=== Python SDK Example ===
```python
from gui_agents import Agent
import asyncio

async def main():
    # Initialize agent
    agent = Agent(
        api_url="http://localhost:4113",
        llm_provider="openai",
        llm_model="gpt-4"
    )
    
    # Take screenshot
    screenshot = await agent.screenshot()
    screenshot.save("desktop.png")
    
    # Perform automation
    await agent.click(x=100, y=200)
    await agent.type_text("Hello from Agent S2!")
    
    # Execute complex task
    result = await agent.execute_task(
        "Open calculator and compute 42 * 17"
    )
    print(f"Result: {result}")

asyncio.run(main())
```

=== Node.js Example ===
```javascript
const axios = require('axios');

class AgentS2Client {
    constructor(baseUrl = 'http://localhost:4113') {
        this.baseUrl = baseUrl;
    }
    
    async screenshot(options = {}) {
        const response = await axios.post(
            `${this.baseUrl}/screenshot`,
            options
        );
        return response.data;
    }
    
    async click(x, y, button = 'left') {
        const response = await axios.post(
            `${this.baseUrl}/mouse/click`,
            { x, y, button }
        );
        return response.data;
    }
    
    async typeText(text, interval = 0.0) {
        const response = await axios.post(
            `${this.baseUrl}/keyboard/type`,
            { text, interval }
        );
        return response.data;
    }
    
    async pressKey(key, modifiers = null) {
        const response = await axios.post(
            `${this.baseUrl}/keyboard/press`,
            { key, modifiers }
        );
        return response.data;
    }
}

// Usage
const agent = new AgentS2Client();

// Take screenshot
const screenshot = await agent.screenshot({ format: 'png' });

// Click at position
await agent.click(500, 300);

// Type text
await agent.typeText('Automated with Agent S2!');
```

=== Bash/cURL Example ===
```bash
#!/bin/bash

AGENT_S2_URL="http://localhost:4113"

# Take screenshot (JSON format)
curl -X POST "${AGENT_S2_URL}/screenshot" \
    -H "Content-Type: application/json" \
    -d '{"format": "png"}' \
    -o screenshot.json

# Take screenshot (binary format)
curl -X POST "${AGENT_S2_URL}/screenshot?format=png&response_format=binary" \
    -o screenshot.png

# Mouse click
curl -X POST "${AGENT_S2_URL}/mouse/click" \
    -H "Content-Type: application/json" \
    -d '{"x": 100, "y": 100, "button": "left"}'

# Type text
curl -X POST "${AGENT_S2_URL}/keyboard/type" \
    -H "Content-Type: application/json" \
    -d '{"text": "Hello, World!", "interval": 0.1}'

# Press key
curl -X POST "${AGENT_S2_URL}/keyboard/press" \
    -H "Content-Type: application/json" \
    -d '{"key": "Return"}'

# Press key with modifiers
curl -X POST "${AGENT_S2_URL}/keyboard/press" \
    -H "Content-Type: application/json" \
    -d '{"key": "c", "modifiers": ["ctrl"]}'

# Complex workflow example
# Take screenshot
curl -X POST "${AGENT_S2_URL}/screenshot?format=png&response_format=binary" -o before.png

# Move mouse and click
curl -X POST "${AGENT_S2_URL}/mouse/move" \
    -H "Content-Type: application/json" \
    -d '{"x": 200, "y": 200}'

curl -X POST "${AGENT_S2_URL}/mouse/click" \
    -H "Content-Type: application/json" \
    -d '{"x": 200, "y": 200}'

# Type text
curl -X POST "${AGENT_S2_URL}/keyboard/type" \
    -H "Content-Type: application/json" \
    -d '{"text": "Automated workflow complete"}'
```
EOF
}

# Export functions for subshell availability
export -f agents2::run_usage_example
export -f agents2::usage_help
export -f agents2::usage_screenshot
export -f agents2::usage_automation
export -f agents2::usage_planning
export -f agents2::usage_capabilities
export -f agents2::usage_all
export -f agents2::show_code_examples