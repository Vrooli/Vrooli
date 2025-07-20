# Autonomous Batch Processing Scenario

## Overview

This scenario demonstrates an **autonomous batch processing workflow** that replicates the common human-in-the-loop pattern of discovering candidates, selecting appropriate items, and processing them iteratively. It showcases how AI agents can automate the typically manual process of reviewing lists and making intelligent selections for batch operations.

### Key Features

- **Read-Only Discovery**: Uses Claude Code with strict read-only restrictions for finding candidates
- **Intelligent Selection**: Agent evaluates and filters candidates using domain knowledge
- **Iterative Processing**: Continues processing batches until completion or limit reached
- **Safety-First Approach**: Conservative selection criteria to prevent accidental data loss
- **Comprehensive Tracking**: Full audit trail of discoveries, selections, and actions

## Agent Architecture

```mermaid
graph TB
    subgraph BatchSwarm[Batch Processing Swarm]
        BC[Batch Coordinator]
        DS[Discovery Specialist]
        SE[Selection Evaluator]
        CE[Cleanup Executor]
        PM[Progress Monitor]
        BB[(Blackboard)]
        
        BC -->|triggers discovery| DS
        DS -->|finds candidates| BB
        SE -->|selects batch| BB
        CE -->|processes files| BB
        PM -->|tracks progress| BB
        BC -->|monitors completion| BB
    end
    
    subgraph AgentRoles[Agent Roles]
        BC_Role[Batch Coordinator<br/>- Orchestrates workflow<br/>- Manages iterations<br/>- Determines completion]
        DS_Role[Discovery Specialist<br/>- Read-only Claude Code<br/>- Finds cleanup candidates<br/>- No file modifications]
        SE_Role[Selection Evaluator<br/>- Analyzes candidates<br/>- Applies safety rules<br/>- Picks safe batches]
        CE_Role[Cleanup Executor<br/>- Processes selections<br/>- Creates backups<br/>- Handles errors gracefully]
        PM_Role[Progress Monitor<br/>- Tracks metrics<br/>- Reports efficiency<br/>- Estimates remaining work]
    end
    
    BC_Role -.->|implements| BC
    DS_Role -.->|implements| DS
    SE_Role -.->|implements| SE
    CE_Role -.->|implements| CE
    PM_Role -.->|implements| PM
```

## Iterative Workflow Pattern

```mermaid
graph LR
    subgraph IterationCycle[Batch Processing Iteration]
        Start[Start Iteration] --> Discover[Discover<br/>Candidates]
        Discover --> Evaluate[Evaluate &<br/>Select Batch]
        Evaluate --> Process[Process<br/>Selected Items]
        Process --> Check{More to<br/>Process?}
        Check -->|Yes| NextIter[Next Iteration]
        Check -->|No| Complete[Workflow Complete]
        NextIter --> Discover
    end
    
    style Start fill:#e8f5e8
    style Complete fill:#e8f5e8
    style Check fill:#fff3e0
```

## Complete Event Flow

```mermaid
sequenceDiagram
    participant START as Swarm Start
    participant BC as Batch Coordinator
    participant DS as Discovery Specialist
    participant SE as Selection Evaluator
    participant CE as Cleanup Executor
    participant BB as Blackboard
    participant ES as Event System
    
    Note over START,ES: Iteration 1 Begins
    START->>BC: swarm/started
    BC->>ES: Emit batch/discovery_requested
    
    Note over DS,BB: Discovery Phase (Read-Only)
    ES->>DS: batch/discovery_requested
    DS->>DS: Execute file-discovery-routine<br/>(Claude Code read-only mode)
    DS->>BB: Store candidates_found (10 files)
    DS->>ES: Emit batch/candidates_found
    
    Note over SE,BB: Selection Phase
    ES->>SE: batch/candidates_found
    SE->>SE: Execute batch-selection-routine
    SE->>SE: Apply safety rules & confidence scoring
    SE->>BB: Store current_batch (4 safe files)
    SE->>ES: Emit batch/selection_complete
    
    Note over CE,BB: Processing Phase
    ES->>CE: batch/selection_complete
    CE->>CE: Execute cleanup-execution-routine
    CE->>CE: Backup files & remove safely
    CE->>BB: Update total_processed=4
    CE->>ES: Emit batch/processing_complete<br/>(continue=true)
    
    Note over BC,ES: Iteration Control
    ES->>BC: batch/processing_complete
    BC->>BC: Check: more candidates exist
    BC->>BB: Increment iterations_completed
    BC->>ES: Emit batch/discovery_requested
    
    Note over DS,ES: Iteration 2 Begins
    ES->>DS: batch/discovery_requested
    DS->>DS: Rediscover (excluding processed)
    DS->>BB: Store candidates_found (6 remaining)
    
    Note over BC,ES: Continue Until Complete
    loop Until all processed or max iterations
        DS->>SE: Discovery → Selection
        SE->>CE: Selection → Processing
        CE->>BC: Processing → Check continuation
    end
    
    Note over BC,ES: Workflow Completion
    BC->>ES: Emit batch/workflow_complete
    BC->>BB: Set processing_complete=true
```

## Read-Only Discovery Pattern

The Discovery Specialist uses a special Claude Code integration with strict read-only restrictions:

```mermaid
graph TD
    subgraph ReadOnlyOps[Allowed Operations]
        Find[find command]
        Grep[grep patterns]
        LS[ls listings]
        Stat[stat metadata]
        Read[read contents]
    end
    
    subgraph BlockedOps[Blocked Operations]
        Write[❌ write]
        Delete[❌ delete]
        Create[❌ create]
        Modify[❌ modify]
        Execute[❌ execute]
    end
    
    subgraph DiscoveryFlow[Discovery Process]
        Criteria[Search Criteria] --> Search[Execute Searches]
        Search --> Filter[Apply Patterns]
        Filter --> Metadata[Gather Metadata]
        Metadata --> Results[Return Candidates]
    end
    
    ReadOnlyOps --> DiscoveryFlow
    BlockedOps -.->|prevented| DiscoveryFlow
    
    style Write fill:#ffebee
    style Delete fill:#ffebee
    style Create fill:#ffebee
    style Modify fill:#ffebee
    style Execute fill:#ffebee
```

## Selection Intelligence

```mermaid
graph TD
    subgraph SelectionLogic[Selection Evaluation Process]
        Candidates[Candidate List] --> Safety{Safety Check}
        
        Safety -->|Pass| Confidence{Confidence Score}
        Safety -->|Fail| Skip[Skip File]
        
        Confidence -->|High > 0.8| Select[Add to Batch]
        Confidence -->|Medium 0.5-0.8| Review[Additional Checks]
        Confidence -->|Low < 0.5| Skip
        
        Review --> Age{File Age}
        Age -->|Old > 30 days| Select
        Age -->|Recent| Skip
        
        Select --> Batch{Batch Full?}
        Batch -->|No| Continue[Next Candidate]
        Batch -->|Yes| Complete[Return Batch]
    end
    
    subgraph SafetyRules[Safety Criteria]
        R1[Not modified in 7 days]
        R2[Not referenced in code]
        R3[Matches cleanup patterns]
        R4[Not in protected paths]
        R5[Has backup if needed]
    end
    
    SafetyRules --> Safety
```

## Blackboard State Evolution

```mermaid
graph LR
    subgraph StateProgression[Blackboard State Through Iterations]
        Init[Initial State<br/>- criteria defined<br/>- max_iterations=5<br/>- batch_size=10]
        
        Iter1[After Iteration 1<br/>+ candidates_found[10]<br/>+ current_batch[4]<br/>+ total_processed=4<br/>+ iterations=1]
        
        Iter2[After Iteration 2<br/>+ candidates_found[6]<br/>+ current_batch[3]<br/>+ total_processed=7<br/>+ iterations=2]
        
        IterN[After Iteration N<br/>+ candidates_found[2]<br/>+ current_batch[2]<br/>+ total_processed=9<br/>+ iterations=3]
        
        Final[Final State<br/>+ processing_complete=true<br/>+ total_processed=10<br/>+ cleanup_history[...]<br/>+ selection_history[...]]
    end
    
    Init --> Iter1
    Iter1 --> Iter2
    Iter2 --> IterN
    IterN --> Final
    
    style Init fill:#e1f5fe
    style Final fill:#e8f5e8
```

### Key Blackboard Fields

| Field | Type | Purpose | Updated By |
|-------|------|---------|------------|
| `discovery_criteria` | object | Search patterns and rules | Initial config |
| `candidates_found` | array | Current list of discovered files | Discovery Specialist |
| `current_batch` | array | Selected files for processing | Selection Evaluator |
| `total_processed` | number | Running count of processed files | Cleanup Executor |
| `iterations_completed` | number | Number of cycles completed | Batch Coordinator |
| `selection_history[]` | array | All selection decisions with rationale | Selection Evaluator |
| `cleanup_history[]` | array | Complete audit trail of operations | Cleanup Executor |
| `processing_complete` | boolean | Workflow completion flag | Batch Coordinator |

## Safety Mechanisms

```mermaid
graph TD
    subgraph SafetyLayers[Multi-Layer Safety Approach]
        L1[Layer 1: Read-Only Discovery<br/>Cannot modify during search]
        L2[Layer 2: Conservative Selection<br/>High confidence required]
        L3[Layer 3: Backup Creation<br/>Before any deletion]
        L4[Layer 4: Error Handling<br/>Graceful failure recovery]
        L5[Layer 5: Audit Trail<br/>Complete operation history]
    end
    
    L1 --> L2
    L2 --> L3
    L3 --> L4
    L4 --> L5
    
    subgraph Protections[Protection Examples]
        P1[Skip files < 7 days old]
        P2[Avoid referenced files]
        P3[Require 0.8+ confidence]
        P4[Limit batch sizes]
        P5[Max iteration limit]
    end
    
    Protections --> SafetyLayers
```

## Example Execution Trace

### Iteration 1
1. **Discovery**: Finds 10 cleanup candidates (.tmp, .backup, .old files)
2. **Selection**: Evaluates all 10, selects 4 safest files (highest confidence, oldest)
3. **Processing**: Backs up and removes 4 files successfully
4. **Decision**: 6 files remain, continue to iteration 2

### Iteration 2
1. **Discovery**: Re-scans, finds 6 remaining candidates
2. **Selection**: Evaluates 6, selects 3 files (medium confidence, verified safe)
3. **Processing**: Backs up and removes 3 files
4. **Decision**: 3 files remain, continue to iteration 3

### Iteration 3
1. **Discovery**: Finds 3 remaining candidates
2. **Selection**: Only 2 meet safety criteria (1 too recent)
3. **Processing**: Processes final 2 files
4. **Decision**: 1 file remains but doesn't meet criteria, workflow complete

## Success Criteria

```json
{
  "requiredEvents": [
    "batch/discovery_requested",
    "batch/candidates_found",
    "batch/selection_complete",
    "batch/processing_complete",
    "batch/workflow_complete"
  ],
  "blackboardState": {
    "total_processed": ">0",
    "iterations_completed": ">=1",
    "processing_complete": "true"
  },
  "expectations": {
    "safetyMaintained": "No critical files deleted",
    "backupsCreated": "All deletions backed up",
    "intelligentSelection": "Risky files skipped"
  }
}
```

## Running the Scenario

### Prerequisites
- Execution test framework operational
- Claude Code integration with read-only mode support
- Mock file system with test cleanup candidates

### Execution Steps

1. **Initialize Scenario**
   ```typescript
   const scenario = new ScenarioFactory("batch-processing-scenario");
   await scenario.setupScenario();
   ```

2. **Configure Discovery Criteria**
   ```typescript
   blackboard.set("discovery_criteria", {
     patterns: [".tmp", ".backup", ".old", "~"],
     olderThan: "30 days",
     excludePaths: ["/node_modules", "/.git"]
   });
   ```

3. **Start Workflow**
   ```typescript
   await scenario.emitEvent("swarm/started", {
     taskId: "cleanup-legacy-files"
   });
   ```

4. **Monitor Progress**
   - Track `iterations_completed` for cycle count
   - Monitor `total_processed` for progress
   - Check `selection_history` for decision rationale
   - Review `cleanup_history` for audit trail

### Debug Information

Key blackboard fields to monitor:
- `candidates_found` - Current discovery results
- `current_batch` - Active selection for processing
- `last_selection_reasoning` - Why files were selected/skipped
- `processing_metrics` - Efficiency and progress data

## Technical Implementation Details

### Claude Code Read-Only Integration
```json
{
  "type": "tool",
  "name": "claude-code-readonly",
  "config": {
    "restrictions": "read-only",
    "allowedOperations": ["read", "grep", "find", "ls", "stat"],
    "blockedOperations": ["write", "delete", "create", "modify", "execute"]
  }
}
```

### Resource Configuration
- **Max Credits**: 1B micro-dollars (supports large discoveries)
- **Max Duration**: 10 minutes (allows multiple iterations)
- **CPU/Memory**: Minimal requirements (read-heavy operations)

### Iteration Control
- **Max Iterations**: 5 (prevents infinite loops)
- **Batch Size**: 10 files (manageable chunks)
- **Safety Threshold**: 0.8 confidence (conservative approach)

This scenario demonstrates how the framework can automate complex human-in-the-loop workflows while maintaining safety, providing audit trails, and making intelligent decisions. It's a perfect example of augmenting human judgment with AI efficiency.