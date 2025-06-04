# SwarmStateMachine Architecture

The SwarmStateMachine is the central orchestrator for swarm coordination, managing the complete lifecycle from initialization through autonomous operation to graceful shutdown.

## 🔄 State Machine Overview

The following diagram shows how the swarm state machine is architected to support proactive subtask management and autonomous event generation:

```mermaid
stateDiagram-v2
    [*] --> Uninitialized
    
    %% Initialization States
    Uninitialized --> Initializing : start(conversationId, goal, user)
    Initializing --> LoadingSwarmConfig : Load conversation state
    LoadingSwarmConfig --> AttachingTeamConfig : Load team structure
    AttachingTeamConfig --> BuildingInitialPrompt : Generate leader system message
    BuildingInitialPrompt --> Starting : Configuration complete
    Starting --> Idle : Emit swarm_started event
    
    %% Core Operational States  
    Idle --> AnalyzingPendingWork : Check for incomplete subtasks
    AnalyzingPendingWork --> HasPendingSubtasks : Evaluate task queue
    AnalyzingPendingWork --> NoWorkAvailable : All tasks complete
    
    %% Autonomous Task Management
    HasPendingSubtasks --> SelectingNextTask : Choose highest priority task
    SelectingNextTask --> AssigningTaskLeader : Identify or assign responsible agent
    AssigningTaskLeader --> GeneratingTaskEvent : Create internal subtask_assigned event
    GeneratingTaskEvent --> Processing : Execute task delegation
    
    %% External Event Processing
    Idle --> Processing : External event received
    Processing --> ExecutingAgentResponse : Route to appropriate responder(s)
    ExecutingAgentResponse --> UpdatingSwarmState : Process agent outputs
    UpdatingSwarmState --> CheckingTaskCompletion : Evaluate task progress
    
    %% Task Completion Assessment
    CheckingTaskCompletion --> TaskCompleted : Subtask marked complete
    CheckingTaskCompletion --> TaskInProgress : Work continuing
    CheckingTaskCompletion --> TaskStalled : No progress detected
    
    TaskCompleted --> UpdatingRecords : Log completion
    TaskInProgress --> WaitingForProgress : Monitor for updates
    TaskStalled --> EscalatingStall : Notify leader/reassign
    
    %% Progress Monitoring
    WaitingForProgress --> CheckingTimeout : Monitor inactivity
    CheckingTimeout --> ProgressTimeout : Exceeded time limit
    CheckingTimeout --> WaitingForProgress : Within limits
    ProgressTimeout --> GeneratingReminderEvent : Create progress_check event
    GeneratingReminderEvent --> Processing : Prompt responsible agent
    
    %% Autonomous Event Generation
    NoWorkAvailable --> CheckingGoalCompletion : Evaluate overall progress
    CheckingGoalCompletion --> GoalCompleted : All objectives achieved
    CheckingGoalCompletion --> GeneratingPlanningEvent : Need more subtasks
    GeneratingPlanningEvent --> Processing : Request goal decomposition
    
    %% Tool Approval Workflows
    Processing --> ToolApprovalRequired : Tool requires user approval
    ToolApprovalRequired --> WaitingForApproval : Present approval request
    WaitingForApproval --> ToolApproved : User approves tool
    WaitingForApproval --> ToolRejected : User rejects tool
    WaitingForApproval --> ApprovalTimeout : Approval times out
    
    ToolApproved --> GeneratingApprovalEvent : Create approval event
    ToolRejected --> GeneratingRejectionEvent : Create rejection event
    ApprovalTimeout --> GeneratingTimeoutEvent : Create timeout event
    GeneratingApprovalEvent --> Processing : Execute approved tool
    GeneratingRejectionEvent --> Processing : Handle rejection
    GeneratingTimeoutEvent --> Processing : Handle timeout
    
    %% State Management
    UpdatingRecords --> Idle : Return to monitoring
    EscalatingStall --> Processing : Handle escalation
    
    %% Pause/Resume Control
    Idle --> Paused : pause() called
    Processing --> Paused : pause() called
    WaitingForProgress --> Paused : pause() called
    WaitingForApproval --> Paused : pause() called
    Paused --> Idle : resume() called
    
    %% Shutdown States
    Idle --> Stopping : stop() called
    Processing --> Stopping : stop() called
    Paused --> Stopping : stop() called
    Stopping --> Stopped : Graceful shutdown
    Stopped --> Terminated : Cleanup complete
    
    %% Error States
    Processing --> Failed : Unrecoverable error
    ExecutingAgentResponse --> Failed : Agent error cascade
    Failed --> Terminated : Error cleanup
    
    %% Final States
    GoalCompleted --> Stopped : Success
    Terminated --> [*]
    
    %% Autonomous Monitoring Loop
    state AnalyzingPendingWork {
        [*] --> CheckingSubtasks
        CheckingSubtasks --> EvaluatingPriorities : Subtasks found
        CheckingSubtasks --> CheckingGoalStatus : No active subtasks
        EvaluatingPriorities --> SelectingCandidate : Priority ranking complete
        CheckingGoalStatus --> [*] : Assessment complete
        SelectingCandidate --> [*] : Task selected
    }
    
    %% Agent Response Coordination
    state ExecutingAgentResponse {
        [*] --> RouteViaAgentGraph
        RouteViaAgentGraph --> DirectMention : respondingBots specified
        RouteViaAgentGraph --> EventSubscription : eventTopic matches
        RouteViaAgentGraph --> SwarmBaton : activeBotId set
        RouteViaAgentGraph --> FallbackSelection : No specific routing
        DirectMention --> AgentExecution
        EventSubscription --> AgentExecution
        SwarmBaton --> AgentExecution
        FallbackSelection --> AgentExecution
        AgentExecution --> [*]
    }
    
    %% Progress Tracking
    state CheckingTaskCompletion {
        [*] --> AnalyzingUpdates
        AnalyzingUpdates --> TaskStatusChange : Status updated
        AnalyzingUpdates --> OutputGenerated : New deliverables
        AnalyzingUpdates --> NoMeaningfulChange : Activity without progress
        TaskStatusChange --> DetermineCompletion
        OutputGenerated --> DetermineCompletion
        NoMeaningfulChange --> DetermineCompletion
        DetermineCompletion --> [*]
    }
```

## 🎯 Key Features for Autonomous Operation

The SwarmStateMachine supports the following critical features:

### 1. Autonomous Task Progression
```typescript
interface AutonomousSubtaskManager {
    // Continuously monitor for incomplete work
    checkPendingSubtasks(): Promise<SubtaskAnalysis>;
    
    // Select next highest priority task
    selectNextTask(availableSubtasks: SwarmSubTask[]): SwarmSubTask | null;
    
    // Generate internal events to drive progress
    generateTaskAssignmentEvent(task: SwarmSubTask, assignee: string): SwarmEvent;
    
    // Monitor task progress and escalate stalls
    monitorTaskProgress(taskId: string): Promise<ProgressAssessment>;
}
```

### 2. Proactive Event Generation
```typescript
interface SwarmEventGenerator {
    // Generate events when no external stimulus exists
    generatePeriodicCheckEvent(): SwarmEvent;
    
    // Create subtask assignment events
    generateSubtaskDelegationEvent(taskId: string, assigneeId: string): SwarmEvent;
    
    // Generate goal decomposition requests
    generatePlanningRequestEvent(currentGoal: string): SwarmEvent;
    
    // Create progress reminder events
    generateProgressReminderEvent(stalledTaskId: string): SwarmEvent;
}
```

### 3. Intelligent Task Assignment
```typescript
interface TaskAssignmentStrategy {
    // Analyze agent capabilities vs task requirements
    findBestAssignee(task: SwarmSubTask, availableAgents: BotParticipant[]): string;
    
    // Handle load balancing across agents
    redistributeWorkload(currentAssignments: Record<string, string[]>): void;
    
    // Escalate stalled tasks to leaders
    escalateStallToLeader(stalledTask: SwarmSubTask): SwarmEvent;
}
```

### 4. Progress Monitoring and Intervention
```typescript
interface ProgressMonitor {
    // Track time since last meaningful progress
    getTimeSinceLastProgress(taskId: string): number;
    
    // Detect when tasks are stalled
    detectStalledTasks(): string[];
    
    // Generate intervention events
    createInterventionEvent(taskId: string, intervention: InterventionType): SwarmEvent;
}
```

## 🔄 Autonomous Event Generation Workflow

The swarm state machine implements several autonomous behaviors:

```mermaid
sequenceDiagram
    participant SSM as SwarmStateMachine
    participant AM as AutonomousMonitor  
    participant EG as EventGenerator
    participant CS as CompletionService
    participant Agents as SwarmAgents

    Note over SSM,Agents: Autonomous Task Progression Cycle

    SSM->>AM: checkPendingSubtasks()
    AM->>AM: Analyze incomplete tasks
    AM-->>SSM: SubtaskAnalysis
    
    alt Has pending subtasks
        SSM->>AM: selectNextTask(subtasks)
        AM-->>SSM: Selected task
        SSM->>EG: generateTaskAssignmentEvent(task, assignee)
        EG-->>SSM: TaskAssignmentEvent
        SSM->>CS: handleInternalEvent(event)
        CS->>Agents: Route to assigned agent
        Agents-->>CS: Progress update
        CS-->>SSM: Task progress
    else No pending subtasks
        SSM->>EG: generatePlanningRequestEvent(goal)
        EG-->>SSM: PlanningRequestEvent
        SSM->>CS: handleInternalEvent(event)  
        CS->>Agents: Request goal decomposition
        Agents-->>CS: New subtasks created
        CS-->>SSM: Updated task list
    end
    
    Note over SSM,Agents: Progress Monitoring Loop
    
    loop Every monitoring interval
        SSM->>AM: detectStalledTasks()
        AM-->>SSM: Stalled task list
        
        alt Tasks are stalled
            SSM->>EG: createInterventionEvent(taskId)
            EG-->>SSM: InterventionEvent
            SSM->>CS: handleInternalEvent(event)
            CS->>Agents: Escalate/reassign task
        end
    end
```

## 📊 Event-Driven Architecture Integration

The SwarmStateMachine integrates seamlessly with the execution event system:

```typescript
// Event types for autonomous operation
type SwarmEvent = 
    | ExternalMessageEvent
    | ToolApprovalEvent  
    | SubtaskAssignmentEvent      // Autonomous task delegation
    | ProgressReminderEvent       // Stall prevention
    | PlanningRequestEvent        // Goal decomposition
    | TaskCompletionEvent         // Progress tracking
    | InterventionRequiredEvent;  // Escalation handling

// Autonomous event generation
class SwarmStateMachine {
    private autonomousMonitor: AutonomousMonitor;
    private eventGenerator: SwarmEventGenerator;
    private progressMonitor: ProgressMonitor;
    
    // Main monitoring loop that runs when IDLE
    private async autonomousMonitoringLoop(): Promise<void> {
        while (this.state === SwarmState.IDLE) {
            // Check for pending work
            const analysis = await this.autonomousMonitor.checkPendingSubtasks();
            
            if (analysis.hasPendingTasks) {
                const nextTask = await this.autonomousMonitor.selectNextTask(analysis.pendingTasks);
                if (nextTask) {
                    const assignmentEvent = this.eventGenerator.generateTaskAssignmentEvent(
                        nextTask, 
                        nextTask.assignedTo || await this.findBestAssignee(nextTask)
                    );
                    await this.handleEvent(assignmentEvent);
                }
            } else if (analysis.needsMorePlanning) {
                const planningEvent = this.eventGenerator.generatePlanningRequestEvent(this.currentGoal);
                await this.handleEvent(planningEvent);
            }
            
            // Check for stalled tasks
            const stalledTasks = await this.progressMonitor.detectStalledTasks();
            for (const taskId of stalledTasks) {
                const reminderEvent = this.eventGenerator.generateProgressReminderEvent(taskId);
                await this.handleEvent(reminderEvent);
            }
            
            // Wait before next monitoring cycle
            await this.waitForMonitoringInterval();
        }
    }
}
```

## ⚙️ Lifecycle Management

### Initialization Phase
1. **Load Configuration**: Retrieve conversation and team configuration
2. **Build Context**: Generate initial system prompts with MOISE+ structure
3. **Initialize Monitoring**: Set up autonomous monitoring loops
4. **Emit Start Event**: Signal that swarm is ready for operation

### Operational Phase
1. **Event Processing**: Handle external and internal events sequentially
2. **Agent Coordination**: Route events to appropriate responders
3. **Progress Tracking**: Monitor task completion and identify stalls
4. **Autonomous Actions**: Generate events to maintain forward progress

### Shutdown Phase
1. **Graceful Stop**: Complete current processing before stopping
2. **State Persistence**: Save current state for potential resumption
3. **Resource Cleanup**: Release resources and close connections

This architecture ensures that swarms remain active and productive even when no external events are occurring, solving the critical "idle swarm" problem.

---

**Next**: [Autonomous Operations](./autonomous-operations.md) - Detailed autonomous features and monitoring capabilities. 