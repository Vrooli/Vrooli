# Execution Service Scenario Tests

## Overview

Scenario tests are comprehensive integration tests for Vrooli's **three-tier AI execution architecture**. They validate the complex interactions between multiple AI agents, event-driven coordination patterns, and emergent intelligence behaviors that arise from simple event-based rules.

These tests go beyond unit testing individual components - they verify that the entire execution framework can orchestrate sophisticated multi-agent workflows with real-world complexity, including failure handling, retry logic, safety mechanisms, and progressive intervention patterns.

## Architecture Context

### Three-Tier Execution Model

Vrooli's execution service implements a hierarchical AI architecture:

```
Tier 1: Swarm Intelligence (Coordination)
   ↓
Tier 2: Routine Intelligence (Process Orchestration)  
   ↓
Tier 3: Step Intelligence (Task Execution)
```

- **Tier 1 (SwarmStateMachine)**: Strategic coordination of multiple agents
- **Tier 2 (RoutineStateMachine)**: Process-level orchestration and navigation
- **Tier 3 (StepExecutor)**: Direct task execution and tool integration

### Key Components

- **Event Bus**: Redis-based pub/sub for agent communication
- **Blackboard**: Shared state management with accumulation operations
- **Agent Behaviors**: Event-driven rules that create emergent intelligence
- **OutputOperations**: Sophisticated state accumulation (append, increment, merge, set)

## How Scenario Tests Work

### 1. Scenario Definition Structure

Each scenario is a folder containing:
- `scenario.json` - Complete scenario configuration including agents, routines, and expected flow
- `README.md` - Detailed documentation with mermaid diagrams explaining the scenario

### 2. Agent Configuration

Agents are defined using production-compatible `BotConfigObject` structures:

```json
{
  "name": "agent-name",
  "role": "coordinator|specialist|monitor",
  "botConfig": {
    "__version": "1.0",
    "model": "claude-3-5-sonnet-20241022",
    "agentSpec": {
      "goal": "High-level agent objective",
      "subscriptions": ["event/topics/*"],
      "behaviors": [/* Event-driven rules */],
      "prompt": {/* Additional context */}
    }
  }
}
```

### 3. Event-Driven Behaviors

Agents coordinate through event subscriptions and emissions:

```json
{
  "trigger": {
    "topic": "custom/event/name",
    "when": "event.data.value > threshold"
  },
  "action": {
    "type": "routine|emit|invoke",
    "outputOperations": {
      "append": [/* Accumulate to arrays */],
      "increment": [/* Update counters */],
      "set": [/* Store current state */]
    }
  }
}
```

### 4. Blackboard Accumulation

The blackboard provides shared state with sophisticated accumulation patterns:
- **append**: Add to arrays (history, logs)
- **increment**: Update counters (attempts, warnings)
- **merge/deepMerge**: Combine complex objects
- **set**: Store current values

### 5. Success Criteria

Each scenario defines clear success conditions:
- Required events that must occur
- Expected blackboard state
- Minimum routine calls
- Agent coordination validation

## Current Scenario Tests

### 1. Debug Oversight Scenario (`debug-oversight-scenario/`)

**Purpose**: Tests sophisticated safety mechanisms and progressive intervention in AI debugging workflows.

**What it Tests**:
- **Scope Creep Detection**: Monitors when debugging expands beyond the original issue
- **Progressive Intervention**: Monitor → Warn → Halt escalation pattern
- **Cross-Agent Communication**: Debug Leader acknowledges Oversight Monitor warnings
- **Safety Mechanisms**: Automatic halting of ineffective debugging cycles

**Key Agents**:
- **Debug Leader**: Analyzes failures, implements fixes, escalates approaches
- **Oversight Monitor**: Watches for scope creep, issues warnings, can halt the swarm

**Why It's Important**:
This scenario validates that AI agents can self-regulate and prevent runaway processes. It demonstrates how simple event-driven rules can create sophisticated safety behaviors, essential for production AI systems that need guardrails against expanding beyond their intended scope.

**Real-World Application**: Prevents AI debugging sessions from suggesting architectural overhauls for simple bugs, maintaining focus and preventing resource waste.

### 2. Redis Fix Loop Scenario (`redis-fix-loop/`)

**Purpose**: Tests progressive retry coordination for infrastructure problem resolution with automatic escalation.

**What it Tests**:
- **Progressive Fix Strategy**: Each retry applies more sophisticated solutions
- **Retry Coordination**: Automatic retry loops with attempt limiting (max 5)
- **Multi-Agent Orchestration**: Three agents coordinate through custom events
- **Gradual Improvement Detection**: Validates partial progress between attempts

**Key Agents**:
- **Fixer Agent**: Applies escalating fixes (connection pool → retry logic → keepalive)
- **Validator Agent**: Comprehensive testing with detailed diagnostics
- **Coordinator Agent**: Manages retry loops and success/failure detection

**Why It's Important**:
This scenario proves the execution framework can handle real-world infrastructure issues that require multiple attempts with increasing sophistication. It validates that agents can learn from previous attempts and apply progressively complex solutions.

**Real-World Application**: Models how production systems should handle transient infrastructure failures - starting with simple fixes and escalating only when necessary, preventing over-engineering of solutions.

## Planned Scenario Tests (TODO)

### 3. React Component Refactor Scenario (`react-refactor-scenario/`)

**Purpose**: Tests leader delegation during a complex React component modernization from class components to hooks.

**What it Will Test**:
- **Leader Transition**: Main coordinator delegates to specialists for different aspects of the refactor
- **Context Isolation**: Each specialist works independently without affecting others' work
- **Handback Protocol**: Specialists return control after completing their specific refactoring tasks
- **State Preservation**: Overall refactoring progress is maintained across delegations

**Key Agents**:
- **Refactor Coordinator**: Analyzes components and orchestrates the modernization effort
- **State Management Specialist**: Converts Redux patterns to Context API and custom hooks
- **Testing Specialist**: Updates test suites from enzyme to React Testing Library
- **Performance Specialist**: Implements memo, useMemo, and useCallback optimizations
- **Styling Specialist**: Migrates from CSS modules to styled-components or CSS-in-JS

**Why It's Important**:
Large React codebases often need systematic modernization. This scenario validates that complex refactoring can be decomposed into specialized tasks, with each expert working in isolation while maintaining overall consistency.

**Real-World Application**: A legacy React application with 100+ class components needs modernization. The coordinator identifies components, delegates refactoring to specialists based on complexity (state management, performance-critical, heavily tested), and ensures the application remains functional throughout.

**TODO**: Implement this scenario with focus on:
- Testing parallel specialist work without conflicts
- Validating component functionality preservation
- Recording refactoring patterns for future reuse
- Testing rollback if any specialist encounters blockers

### 4. Mobile App Development Scenario (`mobile-app-dev-scenario/`)

**Purpose**: Tests director swarm coordinating parallel native mobile app development across iOS and Android platforms.

**What it Will Test**:
- **Platform-Specific Swarms**: Director spawns iOS and Android swarms with platform expertise
- **Shared Backend Swarm**: Common GraphQL API development for both platforms
- **Design System Swarm**: Ensures consistent UI components across platforms
- **Cross-Platform Coordination**: Synchronizing feature parity and shared logic

**Key Agents**:
- **Mobile Director**: Analyzes requirements and orchestrates platform-specific development
- **iOS Swarm**: Swift/SwiftUI specialists building native iOS features
- **Android Swarm**: Kotlin/Jetpack Compose specialists for Android implementation
- **Backend API Swarm**: GraphQL schema design and resolver implementation
- **Design System Swarm**: Creates platform-aware component library

**Why It's Important**:
Mobile development requires coordinated effort across platforms while maintaining native performance and feel. This scenario tests the framework's ability to manage parallel development streams that must stay synchronized.

**Real-World Application**: Building a social media app requiring native performance on both platforms. The director ensures feature parity, coordinates shared backend development, and manages platform-specific optimizations (iOS Core Data vs Android Room, push notifications, etc.).

**TODO**: Implement this scenario with focus on:
- Testing parallel platform development
- Validating API contract synchronization
- Managing shared vs platform-specific code
- Coordinating release readiness across platforms

### 5. Large Codebase Refactoring Scenario (`large-refactor-scenario/`)

**Purpose**: Tests resource management during extensive refactoring of a 50,000+ line legacy codebase.

**What it Will Test**:
- **Credit Monitoring**: Tracks token consumption across thousands of file analyses
- **Progressive Saves**: Completes and commits refactoring in chunks as resources deplete
- **Degradation Strategy**: Shifts from full refactoring to critical-path-only as credits run low
- **Work Documentation**: Creates detailed TODO list of remaining refactoring tasks

**Key Agents**:
- **Refactoring Orchestrator**: Plans and executes systematic codebase modernization
- **Code Analyzer**: Identifies refactoring opportunities and dependencies
- **Resource Monitor**: Tracks credit burn rate and triggers conservation modes
- **Progress Recorder**: Ensures all completed work is saved and documented

**Why It's Important**:
Large refactoring projects often exceed initial resource estimates. This scenario ensures that partial progress is valuable and the system can resume work later without losing effort.

**Real-World Application**: Modernizing a legacy Node.js application from callbacks to async/await. When credits run low at 60% completion, the system ensures all refactored modules are committed, tests still pass, and generates a detailed plan for completing the remaining 40%.

**TODO**: Implement this scenario with focus on:
- Testing checkpoint save mechanisms
- Validating partial refactoring stability
- Testing graceful feature degradation
- Documenting remaining work comprehensively

### 6. CRUD API Pattern Learning Scenario (`crud-learning-scenario/`)

**Purpose**: Tests learning from repeated REST API endpoint creation to generate reusable CRUD patterns.

**What it Will Test**:
- **Pattern Detection**: Recognizes common CRUD implementation sequences across multiple entities
- **Abstraction Creation**: Extracts generic patterns from specific implementations
- **Template Generation**: Creates parameterized routine for future CRUD endpoints
- **Validation Testing**: Ensures generated pattern produces correct implementations

**Key Agents**:
- **Pattern Analyst**: Reviews completed CRUD implementations for commonalities
- **Abstraction Specialist**: Identifies reusable components (validation, error handling, queries)
- **Template Generator**: Creates parameterized routine with configuration options
- **Quality Validator**: Tests generated endpoints match hand-written quality

**Why It's Important**:
CRUD operations are fundamental to most applications. Learning to generate them automatically from patterns reduces boilerplate and ensures consistency across the codebase.

**Real-World Application**: After implementing User, Product, and Order CRUD endpoints, the system recognizes patterns: route setup → validation middleware → database queries → error handling → response formatting. It generates a "create-crud-endpoints" routine that accepts an entity schema and produces all standard endpoints.

**TODO**: Implement this scenario with focus on:
- Capturing implementation patterns across entities
- Testing abstraction quality and reusability
- Validating generated code correctness
- Measuring efficiency improvements

### 7. API Integration Blocker Scenario (`api-integration-blocker-scenario/`)

**Purpose**: Tests intelligent response when payment integration subswarm encounters deprecated third-party API.

**What it Will Test**:
- **Specific Termination**: Payment subswarm stops with "third-party-api-deprecated" reason
- **Alternative Strategy**: Parent swarm pivots to different payment provider
- **State Preservation**: Maintains completed work while switching strategies
- **Dependency Updates**: Adjusts related components for new integration

**Key Agents**:
- **E-commerce Coordinator**: Orchestrates payment integration into existing system
- **Payment Integration Subswarm**: Attempts Stripe integration, discovers v2 API deprecated
- **Alternative Provider Scout**: Researches PayPal, Square, Braintree alternatives
- **Migration Specialist**: Adapts existing code to new payment provider

**Why It's Important**:
External dependencies can fail unexpectedly. This scenario tests the system's ability to intelligently recover from third-party blockers without abandoning all progress.

**Real-World Application**: During checkout implementation, the payment subswarm discovers the planned Stripe API version is deprecated. Instead of failing, it returns a specific reason, prompting the parent to evaluate alternatives and seamlessly switch to PayPal integration while preserving the cart, order, and UI work already completed.

**TODO**: Implement this scenario with focus on:
- Testing specific termination reason handling
- Validating alternative path selection
- Preserving completed work during pivots
- Recording decision rationale for future reference

### 8. Compliance-Restricted Development Scenario (`compliance-dev-scenario/`)

**Purpose**: Tests development under strict regulatory compliance requirements (HIPAA/SOC2).

**What it Will Test**:
- **Tool Whitelist**: Only approved, certified tools and libraries allowed
- **Audit Logging**: Every operation logged for compliance tracking
- **Library Restrictions**: Can only use pre-vetted, security-reviewed packages
- **Access Controls**: No direct database access, only through approved DAL

**Key Agents**:
- **Compliance Developer**: Works within strict regulatory boundaries
- **Security Auditor**: Monitors all operations for compliance violations
- **Approved Tool User**: Limited to certified static analysis and testing tools
- **Compliance Reporter**: Generates required audit documentation

**Why It's Important**:
Regulated industries require development within strict constraints. This scenario ensures the system can remain productive while adhering to compliance requirements.

**Real-World Application**: Building healthcare features requiring HIPAA compliance. Developers can't use arbitrary npm packages, must use approved encryption libraries, can't log PII, and all code changes must be traceable. The system adapts by using only certified tools while still delivering functional features.

**TODO**: Implement this scenario with focus on:
- Testing strict tool/library whitelisting
- Validating audit trail completeness
- Testing compliance-safe error handling
- Ensuring no sensitive data exposure

### 9. Accessibility Compliance Discovery Scenario (`a11y-discovery-scenario/`)

**Purpose**: Tests capability discovery when facing new WCAG 2.1 AA accessibility requirements.

**What it Will Test**:
- **Knowledge Gap Detection**: Recognizes lack of accessibility expertise
- **Tool Discovery**: Finds axe-core, WAVE, screen reader testing tools
- **Pattern Learning**: Discovers ARIA patterns, keyboard navigation requirements
- **Integration Testing**: Validates discovered solutions improve accessibility

**Key Agents**:
- **Frontend Developer**: Encounters accessibility requirement gaps
- **Capability Explorer**: Searches for a11y testing tools and patterns
- **Pattern Evaluator**: Tests discovered solutions against WCAG standards
- **Implementation Specialist**: Adapts patterns to existing components
- **Compliance Validator**: Ensures solutions meet AA standards

**Why It's Important**:
Accessibility requirements are often discovered mid-project. This scenario tests the system's ability to acquire new domain expertise and tools when facing unfamiliar requirements.

**Real-World Application**: An e-commerce site needs WCAG compliance for government contracts. The system discovers automated testing tools (axe-core), learns keyboard navigation patterns, finds screen reader testing methods, and adapts existing React components to be fully accessible, creating reusable a11y patterns.

**TODO**: Implement this scenario with focus on:
- Testing domain knowledge acquisition
- Validating tool discovery and integration
- Testing compliance verification methods
- Building reusable accessibility patterns

### 10. Staging Environment Development Scenario (`staging-env-scenario/`)

**Purpose**: Tests tool access controls in staging environment with production safeguards.

**What it Will Test**:
- **Permitted Tools**: Full code changes, local testing, staging deployments
- **Blocked Operations**: Production deployment, database deletion, credential access
- **Rate Limiting**: API call limits to prevent accidental DDoS of dependencies
- **Audit Requirements**: All staging changes logged for troubleshooting

**Key Agents**:
- **Staging Developer**: Full development capabilities within staging bounds
- **Deployment Agent**: Can deploy to staging but blocked from production
- **Database Agent**: Read/write staging data but cannot drop tables
- **Security Monitor**: Prevents access to production secrets
- **Rollback Specialist**: Can revert staging to last known good state

**Why It's Important**:
Staging environments need development flexibility while preventing accidental production impact. This scenario tests realistic boundary enforcement.

**Real-World Application**: Developing a new feature in staging that integrates with payment providers. Developers can modify code, test with sandbox APIs, and deploy to staging, but are prevented from accessing production payment credentials, deleting staging data, or accidentally deploying to production.

**TODO**: Implement this scenario with focus on:
- Testing production safeguard enforcement
- Validating staging-only tool access
- Testing rollback capabilities
- Ensuring audit trail for debugging

### 11. Autonomous Batch Processing Scenario (`batch-processing-scenario/`)

**Purpose**: Tests automated replication of human-in-the-loop batch workflows where an agent discovers candidates, selects items, and processes them iteratively.

**What it Will Test**:
- **Discovery-Selection-Action Loop**: Agent finds candidates, selects based on criteria, processes batches
- **Read-Only Tool Restrictions**: Discovery routine limited to reads/greps/analysis only
- **Intelligent Selection**: Agent evaluates and filters candidates using domain knowledge
- **Iterative Processing**: Continues until completion criteria met or maximum iterations reached

**Key Agents**:
- **File Discovery Agent**: Uses read-only Claude Code integration to find cleanup candidates
- **Selection Evaluator**: Analyzes discovered files and picks appropriate batch for processing  
- **Cleanup Executor**: Safely removes selected temporary/obsolete files
- **Progress Monitor**: Tracks completion and determines when to continue or stop

**Why It's Important**:
Many software engineering tasks involve finding candidates, making selections, and processing batches. This pattern is fundamental for cleanup tasks, code migrations, refactoring, and maintenance operations.

**Real-World Application**: Cleaning up a legacy codebase with accumulated temporary files. The discovery agent finds all files matching patterns (`.tmp`, `.backup`, old generated files), the evaluator selects safe-to-delete candidates avoiding recent or referenced files, and the executor removes them in batches until the codebase is clean.

**Workflow Example**:
1. **Discovery**: "Find all `.backup` files older than 30 days" → Returns 47 files
2. **Selection**: Agent evaluates list, selects 10 files that are clearly safe to delete
3. **Action**: Removes selected 10 files, updates blackboard with progress
4. **Iteration**: Repeats until all safe candidates processed or max iterations reached

**Technical Implementation**:
- Discovery routine calls Claude Code with read-only restrictions (no file modifications)
- Selection logic uses domain knowledge to evaluate risk/safety of candidates
- Action routine processes selected items with appropriate safeguards
- Blackboard tracks: total_found, total_processed, current_batch, iterations_completed

**TODO**: Implement this scenario with focus on:
- Testing read-only Claude Code integration for discovery
- Validating intelligent selection criteria application
- Testing safe batch processing with rollback capability
- Recording decision rationale for each selection

## Implementation Notes for Planned Scenarios

**Critical Infrastructure Needs**:
1. **Enhanced Records System**: Execution records must capture more than tool calls - need state transitions, resource events, coordination decisions
2. **Hierarchical Swarm Support**: Current flat model needs extension for director/subswarm patterns
3. **Resource Tracking**: Real-time consumption monitoring with event emissions
4. **Pattern Analysis**: Semantic understanding of execution sequences for routine generation

**Testing Considerations**:
- These scenarios require software engineering domain knowledge in test setup
- Mock responses must reflect realistic code generation and analysis
- Success criteria should validate both functional correctness and architectural quality
- Performance metrics needed for resource and optimization scenarios

## Writing New Scenarios

### Best Practices

1. **Single Configuration File**: Consolidate all agents, routines, and mocks in one `scenario.json`
2. **Comprehensive Documentation**: Include mermaid diagrams showing event flows and state transitions
3. **Production-Compatible Schemas**: Use actual `BotConfigObject` structures that match production
4. **Clear Success Criteria**: Define specific events and blackboard states that indicate success
5. **Realistic Mock Responses**: Create mocks that demonstrate actual system behavior patterns

### Scenario Template Structure

```json
{
  "identity": {
    "name": "scenario-name",
    "description": "What this scenario demonstrates",
    "category": "multi-agent|safety|coordination",
    "complexity": "basic|standard|advanced"
  },
  "agents": [/* Agent configurations */],
  "routines": [/* Routine definitions */],
  "expectedFlow": [/* Step-by-step expectations */],
  "resources": {/* Resource constraints */},
  "success": {/* Success criteria */},
  "initialConditions": {/* Starting state */},
  "mocks": {/* Mock responses */}
}
```

## Learning More

### Execution Service Documentation
- `/docs/architecture/execution/` - Architectural design docs
- `/packages/server/src/services/execution/` - Implementation code
  - `tier1/swarmStateMachine.ts` - Swarm coordination
  - `tier2/routineStateMachine.ts` - Routine orchestration
  - `tier3/stepExecutor.ts` - Step execution

### AI Creation Guides
- `/docs/ai-creation/agent/README.md` - Agent creation patterns
- `/docs/ai-creation/agent/OUTPUT_OPERATIONS_GUIDE.md` - Blackboard operations
- `/docs/ai-creation/routine/README.md` - Routine design principles
- `/docs/ai-creation/swarm/` - Swarm coordination patterns

### Event System
- `/packages/server/src/services/events/` - Event bus implementation
- `/docs/architecture/core-services/event-bus-system.md` - Event architecture

### Testing Framework
- `/packages/server/src/__test/execution/` - Test implementation
  - `ScenarioRunner.ts` - Main test orchestrator
  - `SwarmContextManager.ts` - Context management
  - `ScenarioFactory.ts` - Test setup/teardown

## Running Scenario Tests

```bash
# Run all execution tests
cd packages/server && pnpm test src/__test/execution/

# Run specific scenario test
cd packages/server && pnpm test src/__test/execution/scenarios.test.ts

# Run with detailed output
cd packages/server && pnpm test src/__test/execution/scenarios.test.ts -- --reporter=verbose
```

## Why Scenario Tests Matter

1. **Emergent Behavior Validation**: Verifies that simple rules create sophisticated behaviors
2. **Safety Mechanism Testing**: Ensures AI systems have proper guardrails and oversight
3. **Integration Complexity**: Tests the full stack from events to blackboard to agent coordination
4. **Production Readiness**: Uses actual production schemas and patterns
5. **Regression Prevention**: Catches breaks in complex multi-agent interactions

These tests are critical for ensuring that Vrooli's AI execution service can handle real-world complexity with proper safety mechanisms, retry logic, and multi-agent coordination - all essential for a production AI platform.