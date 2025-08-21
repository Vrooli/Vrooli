# Auto/ System Technical Architecture

## 🏗️ System Overview

The auto/ system implements a modular, event-driven architecture for orchestrating Claude Code in continuous improvement loops. It consists of a generic loop core, task-specific modules, and supporting infrastructure for process management, logging, and metrics.

## 📐 Component Architecture

```
auto/
├── Entry Points (Shell Scripts)
│   ├── task-manager.sh          # Central dispatcher
│   ├── manage-resource-loop.sh  # Resource task convenience wrapper
│   └── manage-scenario-loop.sh  # Scenario task convenience wrapper
│
├── Core Libraries (lib/)
│   ├── loop.sh                  # Main orchestration loop
│   ├── core.sh                  # Core utilities and constants
│   ├── process.sh               # Process management
│   ├── workers.sh               # Worker lifecycle management
│   ├── events.sh                # Event logging and tracking
│   ├── dispatch.sh              # Command routing
│   └── error-handler.sh         # Error handling and recovery
│
├── Task Modules (tasks/)
│   └── [task-name]/
│       ├── task.sh              # Task-specific hooks
│       └── prompts/             # Task prompts
│           └── *.md             # Prompt templates
│
├── Tools (tools/)
│   └── selection/               # Intelligent selection algorithms
│       ├── resource-candidates.sh
│       └── scenario-recommend.sh
│
└── Runtime Data (data/)
    └── [task-name]/
        ├── loop.pid             # Process ID
        ├── loop.log             # Main log file
        ├── events.ndjson        # Event stream
        ├── summary.json         # Metrics summary
        └── iterations/          # Per-iteration logs
```

## 🔄 Data Flow Architecture

### 1. Loop Initialization
```mermaid
task-manager.sh
    ├─> Parse arguments (--task, --prompt)
    ├─> Source task module (tasks/[task]/task.sh)
    ├─> Source loop core (lib/loop.sh)
    └─> Dispatch command via loop_dispatch()
```

### 2. Iteration Execution
```mermaid
loop_dispatch("run-loop")
    ├─> Initialize environment
    ├─> Load previous summary
    ├─> Build prompt
    │   ├─> Load base prompt
    │   ├─> Inject summary context
    │   └─> Add helper context
    ├─> Execute worker
    │   ├─> Spawn Claude Code process
    │   ├─> Monitor execution
    │   └─> Capture output
    ├─> Process results
    │   ├─> Extract summary
    │   ├─> Log events
    │   └─> Update metrics
    └─> Check continuation
        └─> Repeat or terminate
```

### 3. Worker Execution Model
```bash
# Worker process hierarchy
loop.sh (manager)
  └─> worker_wrapper.sh (isolation layer)
      └─> claude (actual work)
          └─> vrooli/docker/etc (system commands)
```

## 🎛️ Core Components

### Loop Core (`lib/loop.sh`)
**Purpose**: Generic iteration engine that can run any task type

**Key Functions**:
- `loop_dispatch()` - Main command router
- `run_loop()` - Iteration orchestrator
- `run_once()` - Single iteration executor

**Design Principles**:
- Task-agnostic execution
- Modular hook system
- Graceful failure handling
- Process isolation

### Task Modules (`tasks/*/task.sh`)
**Purpose**: Define task-specific behavior through hooks

**Required Hooks**:
```bash
# Define task name
LOOP_TASK="task-name"

# Return prompt file candidates
task_prompt_candidates()

# Build helper context for prompt
task_build_helper_context()

# Compose final prompt
task_build_prompt()

# Check worker prerequisites
task_check_worker_available()

# Prepare worker environment
task_prepare_worker_env()

# Execute worker with prompt
task_run_worker()
```

### Process Management (`lib/process.sh`)
**Purpose**: Robust process lifecycle management

**Features**:
- PID tracking and cleanup
- Signal handling (TERM, INT, HUP)
- Lock file management
- Zombie process prevention
- Graceful shutdown sequencing

### Worker Management (`lib/workers.sh`)
**Purpose**: Isolate and control Claude Code execution

**Key Features**:
- Timeout enforcement
- Output redaction (secrets, keys)
- Resource limits
- TCP connection gating
- Concurrent worker limiting

### Event System (`lib/events.sh`)
**Purpose**: Track all loop activities for analysis

**Event Schema**:
```json
{
  "timestamp": "ISO-8601",
  "event": "iteration_start|iteration_end|error|warning",
  "iteration": 1,
  "task": "resource-improvement",
  "duration": 300,
  "exit_code": 0,
  "details": {}
}
```

## 🔐 Security Architecture

### Process Isolation
```bash
# Each worker runs in isolation
Worker Process
  ├─> Separate process group
  ├─> Timeout enforcement
  ├─> Resource limits
  └─> Output sanitization
```

### Secret Management
```bash
# Redaction pipeline
Worker Output
  └─> Redaction Filter
      ├─> API keys (sk-*, api_*)
      ├─> Passwords
      ├─> Tokens
      └─> Clean output
```


## 📊 Metrics Architecture

### Collection Points
```
Worker Start → Event Log → Duration Calculation → Summary Generation
     ↓             ↓                ↓                    ↓
  Timestamp    NDJSON Log      Execution Time      JSON/TXT Summary
```

### Metric Types
```yaml
efficiency_metrics:
  - iterations_per_hour
  - success_rate
  - average_duration
  - resource_utilization

progress_metrics:
  - prd_completion_percentage
  - issues_fixed
  - resources_improved
  - scenarios_validated

quality_metrics:
  - validation_pass_rate
  - rollback_frequency
  - error_rate
  - drift_coefficient
```

## 🔄 State Management

### Persistent State
```bash
data/[task]/
├── loop.pid          # Current process
├── loop.lock         # Mutex lock
├── workers.pids      # Active workers
└── summary.json      # Cumulative metrics
```

### Iteration Context
```bash
# Context flow between iterations
Iteration N-1 Summary
    ↓
Iteration N Prompt
    ↓
Iteration N Execution
    ↓
Iteration N Summary
    ↓
Iteration N+1 Prompt
```

### Lock Management
```bash
# Prevents concurrent execution
start_loop()
  ├─> Acquire lock (loop.lock)
  ├─> Write PID (loop.pid)
  ├─> Execute iterations
  └─> Release lock on exit
```

## 🚦 Control Flow

### Normal Execution
```
START → Initialize → Loop[
  Select Target →
  Build Prompt →
  Execute Worker →
  Process Results →
  Check Continuation
] → Cleanup → END
```

### Error Recovery
```
ERROR → Capture State → Log Event → 
  ├─[Recoverable] → Continue Loop
  └─[Fatal] → Cleanup → EXIT
```

### Signal Handling
```
SIGTERM/SIGINT → 
  ├─> Stop accepting new iterations
  ├─> Finish current iteration
  ├─> Kill workers gracefully
  ├─> Cleanup resources
  └─> Exit cleanly
```

## 🔧 Configuration System

### Environment Variables
```bash
# Core configuration
LOOP_TASK              # Task identifier
INTERVAL_SECONDS       # Delay between iterations
MAX_ITERATIONS         # Loop termination
TIMEOUT               # Worker timeout

# Advanced configuration
MAX_CONCURRENT_WORKERS # Parallelism limit
MAX_TCP_CONNECTIONS   # Network throttling
LOOP_TCP_FILTER       # Process filter
OLLAMA_SUMMARY_MODEL  # NL summary generation

# Task-specific
PROMPT_PATH               # Custom prompt override
```

### Configuration Precedence
```
1. Command-line arguments (highest)
2. Environment variables
3. Task defaults
4. System defaults (lowest)
```

## 🔌 Extension Points

### Adding New Tasks
```bash
# 1. Create task structure
tasks/my-task/
├── task.sh           # Implement hooks
└── prompts/
    └── my-task.md    # Define prompt

# 2. Implement required hooks in task.sh
LOOP_TASK="my-task"
task_prompt_candidates() { ... }
task_build_helper_context() { ... }
# ... other hooks

# 3. Use via task-manager
./task-manager.sh --task my-task start
```

### Custom Selection Tools
```bash
tools/selection/my-selector.sh
# Output: Newline-separated candidates
# Input: JSON via stdin or file
# Logic: Priority scoring algorithm
```

### Worker Customization
```bash
# Override worker behavior in task.sh
task_run_worker() {
    local prompt="$1"
    local iteration="$2"
    
    # Custom worker logic
    my_custom_worker "$prompt"
}
```

## 🎨 Design Patterns

### 1. Hook Pattern
**Purpose**: Extensibility without modification
```bash
# Core calls hook if defined
if declare -f task_hook >/dev/null; then
    task_hook "$args"
fi
```

### 2. Pipeline Pattern
**Purpose**: Composable data processing
```bash
get_data | transform | filter | output
```

### 3. Guard Pattern
**Purpose**: Defensive programming
```bash
[[ -n "${VAR:-}" ]] || die "VAR required"
```

### 4. Singleton Pattern
**Purpose**: Prevent multiple instances
```bash
acquire_lock || die "Already running"
```

## 🔬 Performance Characteristics

### Resource Usage
```yaml
cpu:
  idle: <1%
  active: 10-30% (worker dependent)
  
memory:
  base: ~50MB
  per_worker: ~200-500MB
  
disk:
  logs: ~10MB/day
  events: ~1MB/day
  
network:
  api_calls: 50-200/iteration
  bandwidth: ~1-10MB/iteration
```

### Scalability Limits
```yaml
concurrent_workers: 5 (configurable)
iterations_per_day: ~288 (5-min intervals)
max_continuous_runtime: weeks
log_rotation: automatic
```

## 🚀 Optimization Opportunities

### Current Bottlenecks
1. **Sequential iteration execution** - Could parallelize independent targets
2. **Summary generation latency** - Could pre-compute while worker runs
3. **Log parsing overhead** - Could use structured logging
4. **Selection algorithm** - Could cache scores

### Future Enhancements
1. **Distributed execution** - Run loops across multiple machines
2. **Smart scheduling** - ML-based iteration timing
3. **Incremental summaries** - Stream processing instead of batch
4. **Shared intelligence** - Cross-loop learning database

## 🎭 Failure Modes

### Graceful Failures
- Worker timeout → Log and continue
- Selection failure → Skip iteration
- Validation failure → Document and move on
- Network issues → Retry with backoff

### Fatal Failures
- Disk full → Emergency shutdown
- Memory exhaustion → Process killed
- Corrupted state → Manual intervention
- Permission denied → Configuration error

## 🔍 Debugging Architecture

### Log Hierarchy
```
loop.log          # High-level orchestration
↓
iterations/*.log  # Detailed worker output
↓
events.ndjson    # Structured event stream
```

### Debug Commands
```bash
# Real-time monitoring
tail -f data/[task]/loop.log

# Event analysis
jq '.event == "error"' data/[task]/events.ndjson

# Performance profiling
jq '.duration' data/[task]/events.ndjson | stats

# State inspection
cat data/[task]/summary.json | jq
```

## 📚 Architecture Principles

1. **Modularity**: Components are loosely coupled and independently testable
2. **Resilience**: Failures in one iteration don't affect others
3. **Observability**: Every action is logged and measurable
4. **Extensibility**: New tasks can be added without core changes
5. **Simplicity**: Bash-based for transparency and debuggability
6. **Isolation**: Worker processes can't affect the loop manager
7. **Idempotency**: Iterations can be safely retried
8. **Convergence**: System progresses toward goal despite failures

## 🎬 Conclusion

The auto/ architecture implements a robust, extensible framework for orchestrating AI-driven development loops. Its modular design, comprehensive error handling, and metrics-driven approach enable reliable autonomous operation over extended periods, bootstrapping Vrooli's resources and scenarios toward true autonomy.