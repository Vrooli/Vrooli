# Swarm Manager

**The autonomous consciousness of Vrooli** - A continuously running orchestrator that ensures the system never stops improving itself by intelligently coordinating work across all scenarios.

## 🎯 Purpose & Vision

Swarm Manager replaces the `auto/` folder's loops with intelligent, event-driven orchestration. It decides what work needs to be done, prioritizes it, and dispatches it to the appropriate scenarios - all while learning from outcomes to improve future decisions.

This is not just task management - it's the **decision-making brain** that enables Vrooli to be truly self-improving without human intervention.

## 🏗️ Architecture

### File-Based Task System
```
tasks/
├── active/           # Currently being worked on (max 5)
├── backlog/          # Waiting to be worked on
│   ├── manual/       # Human-created (priority)
│   └── generated/    # AI-suggested (requires approval)
├── staged/           # Analyzed and ready
├── completed/        # Successfully finished
└── failed/           # Failed attempts with logs
```

### Core Components
- **Task Scanner**: Monitors task folders and system state
- **Priority Engine**: Calculates task priority using configurable weights
- **Work Dispatcher**: Calls scenario CLIs to execute work
- **Learning System**: Adjusts weights based on outcomes
- **UI Dashboard**: Trello-like interface for task management

## 🎨 UI Style

**Modern Dark Professional** - A clean, Trello-inspired interface with a dark theme:
- Dark blue-black background (#1a1a2e)
- Kanban board with drag-and-drop task management
- Real-time agent monitoring panel
- Settings panel with priority weight sliders
- Task cards show priority score, estimates, and status

## 🔧 Key Features

### Intelligent Priority Calculation
```
Priority = (Impact × Urgency × Success_Probability) / (Resource_Cost × Cooldown_Factor)
```

Claude-code provides estimates, system calculates priority programmatically.

### Task Categories
1. **🔥 Critical Fixes** (Priority 1000+) - System failures, security issues
2. **🚨 User Requests** (Priority 500-999) - Explicit requests, customer issues
3. **🔧 System Health** (Priority 200-499) - Optimization, maintenance
4. **🌱 Capability Growth** (Priority 100-199) - New scenarios, experiments
5. **📚 Knowledge Building** (Priority 1-99) - Documentation, testing

### Learning & Adaptation
- Tracks success/failure rates per task type
- Adjusts priority weights based on actual vs predicted impact
- Stores patterns in Qdrant for future reference
- Identifies and avoids failure patterns

### CLI-Based Integration
Instead of APIs, swarm-manager calls scenario CLIs directly:
```bash
scenario-generator-v1 create --name "new-app" --requirements "..."
resource-experimenter test --scenario "app" --add-resource "ollama"
app-debugger analyze --app "broken-app" --suggest-fix
```

## 📦 Dependencies

### Required Resources
- **PostgreSQL**: Task metadata and execution history
- **Redis**: Task locks and real-time updates
- **Qdrant**: Learning patterns and task embeddings
- **n8n**: Orchestration workflows
- **Claude-code**: Task analysis and execution

### Integrated Scenarios
- scenario-generator-v1
- resource-experimenter
- app-debugger
- app-issue-tracker
- system-monitor
- task-planner

## 🚀 Usage

### CLI Commands
```bash
# Start the swarm manager
swarm-manager start

# Add a task manually
swarm-manager add-task "Fix n8n webhook reliability"

# View task status
swarm-manager status

# List active agents
swarm-manager agents

# Force task execution
swarm-manager execute <task-id>

# Enable YOLO mode (auto-approve AI tasks)
swarm-manager config set yolo_mode true
```

### Task File Format
```yaml
id: task-2024-001
title: "Improve n8n webhook reliability"
type: resource-improvement
target: n8n
priority_estimates:
  impact: 8          # 1-10
  urgency: "high"    # critical|high|medium|low
  success_prob: 0.75 # 0-1
  resource_cost: "moderate" # minimal|moderate|heavy
priority_score: null # Calculated programmatically
created_by: human    # human|ai
```

### Manual Task Creation
Simply create a YAML file in `tasks/backlog/manual/`:
```bash
echo "title: Fix broken test" > tasks/backlog/manual/fix-test.yaml
```

## 🔄 Task Lifecycle

1. **Creation**: Task added to backlog (manual or generated)
2. **Analysis**: Claude-code estimates priority variables
3. **Staging**: Task moved to staged/ when ready
4. **Execution**: Dispatcher calls appropriate scenario CLI
5. **Completion**: Task moved to completed/ or failed/
6. **Learning**: System adjusts weights based on outcome

## 📊 Monitoring

### Dashboard Views
- **Kanban Board**: Drag tasks between columns
- **Agent Monitor**: See what each agent is working on
- **Priority Queue**: Ranked list of all tasks
- **Success Metrics**: Charts showing completion rates
- **Resource Usage**: CPU, memory, API call graphs

### Health Indicators
- Work throughput (target: >50 improvements/day)
- Success rate (target: >80%)
- Backlog size (target: 10-20 tasks)
- Agent utilization (target: 60-70%)
- Learning rate (improving success over time)

## 🧠 Intelligence Features

### Cooldown Management
Prevents thrashing on repeatedly failing tasks with exponential backoff.

### Dependency Resolution
Ensures tasks are executed in the correct order based on dependencies.

### Resource Allocation
- Max 5 concurrent tasks
- Reserve 20% capacity for urgent work
- Batch similar work types for efficiency

### Pattern Recognition
- Stores successful task sequences
- Identifies common failure modes
- Suggests similar solutions to new problems

## 🔍 Problem Discovery and Resolution

The swarm-manager automatically discovers and resolves problems across your Vrooli system using structured PROBLEMS.md files. This provides semantic understanding of issues that enables autonomous problem resolution.

### How Problem Discovery Works

1. **Automatic Scanning**: Every 5 minutes, swarm-manager scans for PROBLEMS.md files in:
   - `${VROOLI_ROOT}/PROBLEMS.md` (system-wide issues)
   - `${VROOLI_ROOT}/resources/*/PROBLEMS.md` (resource-specific issues)
   - `${VROOLI_ROOT}/scenarios/*/PROBLEMS.md` (scenario-specific issues)

2. **Semantic Extraction**: Problems are parsed using embedded markers that provide structured data:
   ```markdown
   <!-- EMBED:ACTIVEPROBLEM:START -->
   ### Problem Title
   **Status:** [active|investigating|resolved]
   **Severity:** [critical|high|medium|low]
   **Impact:** [system_down|degraded_performance|user_impact|cosmetic]
   <!-- EMBED:ACTIVEPROBLEM:END -->
   ```

3. **Automatic Task Creation**: In YOLO mode, critical and high-severity problems automatically generate resolution tasks with appropriate priority scores.

4. **Intelligent Routing**: Problems are routed to the best scenario for resolution:
   - `app-debugger` for error analysis and fixes
   - `system-monitor` for performance issues
   - `resource-experimenter` for integration problems
   - `claude-code` for general development fixes

### PROBLEMS.md Format

Each resource/scenario should maintain a PROBLEMS.md file following the standard template (see `${VROOLI_ROOT}/PROBLEMS_TEMPLATE.md`). Key elements include:

- **Problem Metadata**: Status, severity, frequency, impact
- **Priority Estimates**: Impact score, urgency level, success probability, resource cost
- **Investigation Details**: Root cause analysis, workarounds, attempted solutions
- **Evidence**: Error messages, metrics, logs

### Problem Management Commands

```bash
# Scan for problems manually
swarm-manager scan-problems

# View discovered problems
swarm-manager problems list           # All problems
swarm-manager problems list critical  # Filter by severity
swarm-manager problems list active    # Filter by status

# View specific problem details
swarm-manager problems show prob-123

# Mark problem as resolved
swarm-manager problems resolve prob-123 "Fixed in v2.1"
```

### Problem Scanning Configuration

Configure in `config/settings.yaml`:

```yaml
problem_scanning:
  enabled: true                    # Enable automatic scanning
  scan_interval: 300               # Scan every 5 minutes
  auto_create_tasks: true          # Auto-create tasks for problems
  scan_paths:                      # Paths to scan
    - ${VROOLI_ROOT}/PROBLEMS.md
    - ${VROOLI_ROOT}/resources/*/PROBLEMS.md
    - ${VROOLI_ROOT}/scenarios/*/PROBLEMS.md
  severity_thresholds:
    auto_task_creation: high       # Min severity for auto tasks
    alert_notification: critical   # Min severity for alerts
```

### Problem-to-Task Flow

1. **Discovery**: Problem found in PROBLEMS.md with severity ≥ high
2. **Analysis**: Swarm-manager extracts priority estimates
3. **Task Creation**: Creates task with calculated priority score
4. **Assignment**: Routes to appropriate scenario based on problem type
5. **Execution**: Scenario attempts resolution
6. **Verification**: System monitors if problem recurs
7. **Learning**: Adjusts future priority calculations based on outcome

### Example Problem Entry

```markdown
<!-- EMBED:ACTIVEPROBLEM:START -->
### N8N Webhook Timeout Issues
**Status:** [active]
**Severity:** [high]
**Frequency:** [frequent]
**Impact:** [degraded_performance]
**Discovered:** 2025-01-28
**Discovered By:** [system-monitor]

#### Description
Webhooks timing out after 30 seconds, causing workflow failures.

#### Priority Estimates
```yaml
impact: 8
urgency: "high"
success_prob: 0.8
resource_cost: "moderate"
```

#### Investigation Status
- **Root Cause:** Docker container resource limits
- **Workarounds:** Manual workflow restart
<!-- EMBED:ACTIVEPROBLEM:END -->
```

This problem would automatically generate a high-priority task to fix the webhook timeouts, assigned to the app-debugger scenario.

## 🔮 Migration from auto/

### Phase 1: Parallel Operation
Deploy alongside auto/ to compare decisions and outcomes.

### Phase 2: Primary Control
Swarm-manager becomes primary, auto/ as fallback.

### Phase 3: Full Migration
Disable auto/ loops, achieve full autonomy.

## 🎯 Success Metrics

- **Autonomy**: Zero human intervention for 7+ days
- **Throughput**: >50 meaningful improvements per day
- **Success Rate**: >80% task completion
- **Learning**: Increasing success rate over time
- **Efficiency**: <70% average resource utilization

## 🔧 Configuration

### Priority Weights (`config/priority-weights.yaml`)
Adjust the importance of each factor in priority calculation.

### Settings (`config/settings.yaml`)
- `yolo_mode`: Auto-approve AI-generated tasks
- `max_concurrent`: Maximum simultaneous tasks
- `min_backlog_size`: When to generate new tasks
- `resource_limits`: CPU/memory constraints

### Prompts (`prompts/`)
All Claude-code prompts are editable markdown files:
- `task-analyzer.md`: Analyze and estimate task variables
- `task-selector.md`: Choose next task to work on
- `task-executor.md`: Execute the actual work
- `backlog-generator.md`: Generate new task suggestions

## 🌟 Why Swarm Manager?

Unlike auto/'s simple loops, swarm-manager provides:
- **Event-driven** instead of interval-based execution
- **Parallel coordination** of multiple scenarios
- **Learning and adaptation** from outcomes
- **File-based tasks** for easy manual intervention
- **Professional UI** for monitoring and control
- **True autonomy** through intelligent decision-making

This is the missing piece that transforms Vrooli from a collection of tools into a **self-improving intelligence system**.