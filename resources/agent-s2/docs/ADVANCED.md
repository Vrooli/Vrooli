# Agent S2 Advanced Usage Guide

This guide covers advanced concepts, architectural details, integration patterns, and sophisticated usage scenarios for Agent S2.

## Architecture Overview

### Two-Layer Architecture

Agent S2 operates with a unique two-layer architecture that separates intelligence from execution:

```
User: "Take a screenshot and click the Chrome icon"
                    ↓
┌─────────────────────────────────────────────────┐
│              AI Layer (Intelligence)             │
├─────────────────────────────────────────────────┤
│ 1. Understand natural language command           │
│ 2. Plan sequence of actions:                    │
│    - Take screenshot to see screen              │
│    - Analyze screenshot to find Chrome icon     │
│    - Calculate Chrome icon coordinates          │
│    - Click at those coordinates                 │
└────────────────┬────────────────────────────────┘
                 ↓ Calls Core Functions
┌─────────────────────────────────────────────────┐
│         Core Automation Layer (Execution)        │
├─────────────────────────────────────────────────┤
│ • execute_screenshot() → PyAutoGUI screenshot    │
│ • analyze_image() → Process screenshot           │
│ • execute_click(x,y) → PyAutoGUI click          │
└─────────────────────────────────────────────────┘
                 ↓
            Screen Action Performed
```

**Key Principle**: AI doesn't replace core automation - it orchestrates it intelligently!

### AI Command Translation

When you send an AI command, here's the internal process:

```python
# What you send:
{
    "task": "take a screenshot and move mouse to center",
    "context": {"purpose": "demonstration"}
}

# What the AI layer does:
1. Parse task → identifies "screenshot" and "move mouse"
2. Plans execution sequence
3. Calls core API endpoints:
   - POST /screenshot?format=png
   - GET /mouse/position (to get screen dimensions)
   - POST /mouse/move {"x": width//2, "y": height//2}

# What gets returned:
{
    "task": "take a screenshot and move mouse to center",
    "actions_taken": [
        {"action": "screenshot", "endpoint": "/screenshot", "status": "completed"},
        {"action": "mouse_move", "endpoint": "/mouse/move", "parameters": {"x": 960, "y": 540}, "status": "completed"}
    ],
    "reasoning": "Captured screen state, calculated center coordinates, moved mouse to center position",
    "success": true
}
```

## Dual-Mode Operation

### Mode Comparison Matrix

| Feature | Sandbox Mode | Host Mode |
|---------|--------------|-----------|
| **Security Level** | High | Medium |
| **Isolation** | Full container isolation | Controlled host access |
| **File Access** | Container only (`/home/agents2`, `/tmp`, `/opt/agent-s2`) | Host filesystem (with security constraints) |
| **Applications** | Container apps only | Host applications + container apps |
| **Network Access** | External HTTPS only | Localhost + private networks |
| **Display Access** | Virtual display (X11) | Host display (X11 forwarding) |
| **Risk Level** | Minimal | Controlled |
| **Use Cases** | General automation, web browsing, document editing | System administration, development workflows, native app automation |

### Advanced Mode Management

#### Runtime Mode Switching

```bash
# Check current mode and capabilities
curl http://localhost:4113/modes/current | jq '.'

# Switch modes with validation
curl -X POST http://localhost:4113/modes/switch \
  -H "Content-Type: application/json" \
  -d '{
    "new_mode": "host",
    "validate": true,
    "backup_current": true
  }'

# Get mode-specific security constraints
curl http://localhost:4113/modes/security | jq '.constraints'

# List available applications per mode
curl http://localhost:4113/modes/applications | jq '.available'
```

#### Mode Configuration Profiles

Create custom mode profiles for different use cases:

```json
{
  "profiles": {
    "development": {
      "mode": "host",
      "allowed_apps": ["code", "firefox", "terminal"],
      "filesystem_access": ["/home/user/projects", "/tmp"],
      "network_access": ["localhost", "192.168.1.0/24"],
      "security_level": "medium"
    },
    "testing": {
      "mode": "sandbox",
      "applications": ["firefox", "chromium"],
      "filesystem_access": ["/home/agents2/test-data"],
      "network_access": ["https://"],
      "security_level": "high"
    }
  }
}
```

## Architectural Comparisons

### Agent S2 vs Other Automation Services

```
Agent S2:                          n8n/Node-RED:
┌─────────────────┐               ┌─────────────────┐
│   X11 Display   │               │   Web Server    │
├─────────────────┤               ├─────────────────┤
│  GUI Automation │               │ Workflow Engine │
├─────────────────┤               ├─────────────────┤
│   REST API      │               │   REST API      │
├─────────────────┤               ├─────────────────┤
│ Screenshot/OCR  │               │ Node Executor   │
└─────────────────┘               └─────────────────┘
```

### Capability Comparison

| Feature | Agent S2 | n8n | Node-RED | Browserless | Huginn |
|---------|----------|-----|----------|-------------|---------|
| **Primary Focus** | GUI automation & screenshots | Workflow automation | IoT/Flow programming | Web browser automation | Web scraping & monitoring |
| **Interface Type** | Desktop GUI (any app) | Web services/APIs | Hardware/APIs | Web pages only | Web pages/APIs |
| **Programming Model** | Task-based API | Visual workflows | Flow-based nodes | Browser scripting | Event-based agents |
| **AI Integration** | Built-in LLM planning | Via HTTP nodes | Custom nodes | None built-in | None built-in |
| **Execution Model** | Direct GUI control | Webhook/scheduled | Event-driven | Page automation | Periodic/triggered |

### Key Differentiators

1. **Desktop Application Control**
   - **Agent S2**: Can control ANY desktop application (Excel, Photoshop, games, etc.)
   - **Others**: Limited to web/API interactions

2. **Visual Computer Use**
   - **Agent S2**: Takes screenshots, analyzes screen content, makes decisions
   - **Others**: Work with structured data/APIs

3. **Human-like Interaction**
   - **Agent S2**: Simulates mouse movements, typing, clicking like a human
   - **Others**: Direct API calls or DOM manipulation

## Advanced Integration Patterns

### Multi-Resource Workflows

#### 1. Agent S2 + n8n Integration

**Scenario**: Automated testing workflow that captures desktop app screenshots and processes results

```json
{
  "workflow": "desktop_app_testing",
  "steps": [
    {
      "service": "agent-s2",
      "action": "ai/action",
      "parameters": {
        "task": "Open calculator app and perform basic operations",
        "context": {"test_suite": "basic_functionality"}
      }
    },
    {
      "service": "n8n",
      "action": "process_results",
      "parameters": {
        "input": "{{previous.screenshot}}",
        "validation": "check_calculation_result"
      }
    }
  ]
}
```

**n8n HTTP Request Node Configuration**:
```json
{
  "method": "POST",
  "url": "http://agent-s2:4113/ai/action",
  "authentication": "none",
  "jsonParameters": true,
  "body": {
    "task": "capture screen and click on the first visible button",
    "context": {"workflow": "automated_testing"}
  }
}
```

#### 2. Agent S2 + Browserless Combination

**Scenario**: Complete automation covering both desktop and web interactions

```python
# Sequential automation workflow
def complete_automation_workflow():
    # Step 1: Desktop automation with Agent S2
    desktop_result = agent_s2_client.ai_action(
        task="Open local data file and extract information",
        context={"source": "desktop_app"}
    )
    
    # Step 2: Web automation with Browserless
    web_result = browserless_client.execute_script(
        script="navigate to web form and submit extracted data",
        data=desktop_result['extracted_data']
    )
    
    # Step 3: Back to desktop for verification
    verification = agent_s2_client.ai_action(
        task="Check desktop notifications for confirmation",
        context={"verification": True}
    )
    
    return {
        "desktop_phase": desktop_result,
        "web_phase": web_result,
        "verification": verification
    }
```

#### 3. Agent S2 + ComfyUI Creative Pipeline

**Scenario**: Game screenshot → AI art generation workflow

```python
def gaming_art_pipeline():
    # Capture game screenshots
    screenshot = agent_s2_client.screenshot(format="png")
    
    # Process with ComfyUI for artistic enhancement
    artistic_result = comfyui_client.process_workflow({
        "input_image": screenshot,
        "style": "cyberpunk_art",
        "enhancement": "high_detail"
    })
    
    # Use Agent S2 to set as desktop background
    agent_s2_client.ai_action(
        task="Set the generated image as desktop wallpaper",
        context={"image_path": artistic_result['output_path']}
    )
```

### Advanced Automation Patterns

#### State-Aware Automation

```python
class StateAwareAutomation:
    def __init__(self):
        self.client = AgentS2Client()
        self.state_history = []
    
    def execute_with_state_tracking(self, task):
        # Capture initial state
        initial_state = self.capture_state()
        
        # Execute task
        result = self.client.ai_action(
            task=task,
            context={"previous_state": initial_state}
        )
        
        # Verify state change
        final_state = self.capture_state()
        state_diff = self.compare_states(initial_state, final_state)
        
        # Store in history
        self.state_history.append({
            "task": task,
            "initial_state": initial_state,
            "final_state": final_state,
            "changes": state_diff,
            "timestamp": datetime.now()
        })
        
        return result
    
    def capture_state(self):
        screenshot = self.client.screenshot()
        return {
            "screen_content": screenshot,
            "mouse_position": self.client.get_mouse_position(),
            "active_windows": self.client.ai_analyze(
                question="What applications are currently visible?"
            )
        }
```

#### Error Recovery Automation

```python
class ResilientAutomation:
    def __init__(self):
        self.client = AgentS2Client()
        self.max_retries = 3
    
    def execute_with_recovery(self, task, recovery_strategies=None):
        for attempt in range(self.max_retries):
            try:
                # Attempt the task
                result = self.client.ai_action(task=task)
                
                # Validate success
                if self.validate_result(result):
                    return result
                else:
                    raise AutomationError("Task validation failed")
                    
            except Exception as e:
                if attempt < self.max_retries - 1:
                    # Apply recovery strategy
                    recovery_action = self.select_recovery_strategy(e, recovery_strategies)
                    self.execute_recovery(recovery_action)
                else:
                    raise e
    
    def select_recovery_strategy(self, error, strategies):
        # Intelligent recovery strategy selection
        if "screenshot_failed" in str(error):
            return "reset_display"
        elif "element_not_found" in str(error):
            return "refresh_screen"
        elif "timeout" in str(error):
            return "wait_and_retry"
        else:
            return strategies.get("default", "capture_state_and_restart")
```

## Performance Optimization

### Resource Optimization Strategies

#### Memory Management

```bash
# Monitor memory usage patterns
docker stats agent-s2 --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"

# Optimize for different workloads
# High AI processing
export AGENT_S2_MEMORY_LIMIT=8g
export AGENT_S2_AI_CACHE_SIZE=2g

# Basic automation only
export AGENT_S2_MEMORY_LIMIT=2g
export AGENT_S2_SCREENSHOT_CACHE_SIZE=256m
```

#### CPU Optimization

```bash
# AI-heavy workloads
docker update agent-s2 --cpus 4.0 --cpu-shares 1024

# I/O bound operations
docker update agent-s2 --cpus 1.0 --cpu-shares 512
```

#### Display Performance

```bash
# High-resolution displays
export AGENT_S2_DISPLAY_WIDTH=3840
export AGENT_S2_DISPLAY_HEIGHT=2160
export AGENT_S2_SCREENSHOT_QUALITY=95

# Performance-optimized displays
export AGENT_S2_DISPLAY_WIDTH=1280
export AGENT_S2_DISPLAY_HEIGHT=720
export AGENT_S2_SCREENSHOT_QUALITY=75
```

### Scaling Patterns

#### Horizontal Scaling

```yaml
# Docker Compose scaling
version: '3.8'
services:
  agent-s2-1:
    image: agent-s2:latest
    ports:
      - "4113:4113"
      - "5900:5900"
    environment:
      - AGENT_S2_INSTANCE_ID=1
  
  agent-s2-2:
    image: agent-s2:latest
    ports:
      - "4114:4113"
      - "5901:5900"
    environment:
      - AGENT_S2_INSTANCE_ID=2
  
  load-balancer:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

#### Load Distribution

```python
class AgentS2LoadBalancer:
    def __init__(self, instances):
        self.instances = instances
        self.current_load = {instance: 0 for instance in instances}
    
    def select_instance(self):
        # Simple round-robin with load awareness
        available_instances = [
            instance for instance, load in self.current_load.items()
            if load < 5  # Max concurrent tasks per instance
        ]
        
        if not available_instances:
            raise Exception("All instances at capacity")
        
        return min(available_instances, key=lambda x: self.current_load[x])
    
    def execute_task(self, task):
        instance = self.select_instance()
        self.current_load[instance] += 1
        
        try:
            client = AgentS2Client(base_url=f"http://{instance}:4113")
            result = client.ai_action(task=task)
            return result
        finally:
            self.current_load[instance] -= 1
```

## Advanced Security Patterns

### Security Monitoring Integration

```python
class SecurityAwareAutomation:
    def __init__(self):
        self.client = AgentS2Client()
        self.security_monitor = SecurityMonitor()
    
    def secure_execute(self, task, security_context=None):
        # Pre-execution security check
        security_clearance = self.security_monitor.check_task_permissions(
            task=task,
            context=security_context
        )
        
        if not security_clearance.approved:
            raise SecurityError(f"Task blocked: {security_clearance.reason}")
        
        # Execute with monitoring
        with self.security_monitor.monitor_execution(task):
            result = self.client.ai_action(task=task)
        
        # Post-execution audit
        self.security_monitor.log_execution(
            task=task,
            result=result,
            security_events=self.security_monitor.get_events()
        )
        
        return result
```

### Compliance Patterns

```python
class ComplianceAutomation:
    def __init__(self):
        self.client = AgentS2Client()
        self.compliance_rules = self.load_compliance_rules()
    
    def compliant_execute(self, task, compliance_framework="SOC2"):
        # Validate against compliance rules
        violations = self.check_compliance_violations(task, compliance_framework)
        
        if violations:
            return self.handle_compliance_violations(violations)
        
        # Execute with audit trail
        audit_id = self.create_audit_record(task)
        
        try:
            result = self.client.ai_action(task=task)
            self.update_audit_record(audit_id, "SUCCESS", result)
            return result
        except Exception as e:
            self.update_audit_record(audit_id, "FAILED", str(e))
            raise
```

## Development and Testing Patterns

### Test Automation Framework

```python
class AgentS2TestFramework:
    def __init__(self):
        self.client = AgentS2Client()
        self.test_results = []
    
    def setup_test_case(self, test_name, setup_tasks=None):
        """Prepare environment for test case"""
        if setup_tasks:
            for task in setup_tasks:
                self.client.ai_action(task=task)
        
        # Capture baseline state
        baseline = self.client.screenshot()
        return {"test_name": test_name, "baseline": baseline}
    
    def run_test_case(self, test_case, steps):
        """Execute test steps and validate results"""
        results = []
        
        for i, step in enumerate(steps):
            try:
                result = self.client.ai_action(task=step['task'])
                
                # Validate step result
                if 'validation' in step:
                    validation_result = self.validate_step(result, step['validation'])
                    results.append({
                        "step": i + 1,
                        "task": step['task'],
                        "result": result,
                        "validation": validation_result,
                        "status": "PASS" if validation_result else "FAIL"
                    })
                else:
                    results.append({
                        "step": i + 1,
                        "task": step['task'],
                        "result": result,
                        "status": "COMPLETED"
                    })
                    
            except Exception as e:
                results.append({
                    "step": i + 1,
                    "task": step['task'],
                    "error": str(e),
                    "status": "ERROR"
                })
        
        return results
```

### Continuous Integration Patterns

```yaml
# GitHub Actions workflow for Agent S2 automation testing
name: Agent S2 Automation Tests
on: [push, pull_request]

jobs:
  automation-tests:
    runs-on: ubuntu-latest
    services:
      agent-s2:
        image: agent-s2:latest
        ports:
          - 4113:4113
          - 5900:5900
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
    
    steps:
      - uses: actions/checkout@v2
      
      - name: Wait for Agent S2 to be ready
        run: |
          timeout 60 bash -c 'until curl -f http://localhost:4113/health; do sleep 2; done'
      
      - name: Run automation tests
        run: |
          python -m pytest tests/automation/ --agent-s2-url=http://localhost:4113
      
      - name: Capture test artifacts
        if: failure()
        run: |
          mkdir -p test-artifacts
          curl http://localhost:4113/screenshot > test-artifacts/final-state.png
          docker logs agent-s2 > test-artifacts/service.log
      
      - name: Upload artifacts
        if: failure()
        uses: actions/upload-artifact@v2
        with:
          name: test-artifacts
          path: test-artifacts/
```

This advanced guide provides the foundation for sophisticated Agent S2 implementations, integrations, and automation patterns. For specific implementation details, refer to the examples directory and API documentation.