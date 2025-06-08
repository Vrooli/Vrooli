# Execution Test Fixtures

This directory contains test fixtures for the three-tier AI execution system, demonstrating emergent capabilities through agent-driven routine evolution.

## Files

### emergentAgentFixtures.ts
Contains agent configurations based on documentation examples:
- **Security Agents**: HIPAA compliance, medical safety, trading fraud detection, API security, data privacy
- **Resilience Agents**: Failure pattern learning, recovery optimization, circuit breaker management
- **Strategy Evolution Agents**: Routine performance analysis, cost optimization, strategy transformation
- **Quality Agents**: Output quality monitoring, bias detection
- **Monitoring Agents**: System health monitoring, anomaly detection
- **Swarm Configurations**: Multi-agent collaborations for complex tasks

## Routines Folder

The `routines/` subfolder contains organized routine fixtures split by domain:

### routines/securityRoutines.ts
Security-focused routines (4 total):
- HIPAA_COMPLIANCE_CHECK - Scans AI outputs for PHI
- API_SECURITY_SCAN - Analyzes API logs for security threats
- GDPR_DATA_AUDIT - Audits data processing for GDPR compliance
- TRADING_PATTERN_ANALYSIS - Detects suspicious trading patterns

### routines/medicalRoutines.ts
Medical and healthcare routines (1 total):
- MEDICAL_DIAGNOSIS_VALIDATION - Validates diagnoses against clinical guidelines

### routines/performanceRoutines.ts
Performance and optimization routines (3 total):
- PERFORMANCE_BOTTLENECK_DETECTION - Detects routine execution bottlenecks
- COST_ANALYSIS - Analyzes operational costs
- OUTPUT_QUALITY_ASSESSMENT - Evaluates output quality including bias

### routines/systemRoutines.ts
System monitoring and resilience routines (2 total):
- SYSTEM_FAILURE_ANALYSIS - Identifies failure patterns and correlations
- SYSTEM_HEALTH_CHECK - Comprehensive system health monitoring

### routines/bpmnWorkflows.ts
Complex multi-step BPMN workflows (3 total):
- COMPREHENSIVE_SECURITY_AUDIT - Parallel security checks combining API, privacy, and threat analysis
- MEDICAL_TREATMENT_VALIDATION - Sequential validation with compliance gates
- RESILIENCE_OPTIMIZATION_WORKFLOW - Combined failure and performance analysis

### routines/evolutionFixtures.ts
Demonstrates routine evolution through agent proposals:
- Evolution stages from conversational → reasoning → deterministic → adaptive
- Agent proposals that drive strategy improvements
- Performance metrics showing improvement at each stage
- Emergent capabilities that develop through evolution

### routines/index.ts
Main entry point providing:
- Aggregated exports of all routine categories
- Helper functions for filtering and finding routines
- Agent-to-routine mapping
- Statistics and categorization utilities

## Key Features

1. **Type Safety**: All fixtures are fully typed with TypeScript interfaces
2. **Agent-Routine Mapping**: Each agent type has corresponding routines to analyze/optimize
3. **Execution Strategies**: Routines use appropriate strategies (deterministic for patterns, reasoning for analysis)
4. **BPMN Support**: Complex workflows with parallel execution, conditional flows, and subroutine composition
5. **Input/Output Configurations**: Realistic FormSchema configurations for each routine type
6. **MCP Tool Integration**: Action routines use actual McpToolName enum values
7. **Evolution Examples**: Shows how agents propose routine improvements based on performance analysis

## Usage

```typescript
// Import from main index
import { 
    getAllRoutines, 
    getRoutineById, 
    getRoutinesByCategory,
    AGENT_ROUTINE_MAP,
    SEQUENTIAL_ROUTINES,
    BPMN_ROUTINES 
} from './routines/index.js';

// Import specific categories
import { SECURITY_ROUTINES } from './routines/securityRoutines.js';
import { MEDICAL_ROUTINES } from './routines/medicalRoutines.js';
import { PERFORMANCE_ROUTINES } from './routines/performanceRoutines.js';

// Import agents
import { getEmergentAgents, getAgentByGoal } from './emergentAgentFixtures.js';

// Get all routines
const allRoutines = getAllRoutines();

// Get routines by category
const securityRoutines = getRoutinesByCategory('security');
const bpmnWorkflows = getRoutinesByCategory('bpmn');

// Get specific routines
const hipaaCheck = SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK;
const medicalValidation = BPMN_ROUTINES.MEDICAL_TREATMENT_VALIDATION;

// Get routines for a specific agent
const hipaaRoutines = AGENT_ROUTINE_MAP['hipaa_compliance_monitor'];

// Get agent configuration
const costAgent = getAgentByGoal('cost_optimization');
```

## Type Safety

All routine fixtures are fully typed with TypeScript interfaces:

- `RoutineFixture`: Base interface for all routines
- `RoutineFixtureCollection<T>`: Typed collection of routines
- `ExecutionStrategy`: Union type for valid strategies
- `RoutineCategory`: Union type for routine categories
- `RoutineStats`: Interface for statistics object
- Type-safe helper functions with proper return types

Example:
```typescript
// TypeScript catches errors at compile time
const routine = SECURITY_ROUTINES.HIPAA_COMPLIANCE_CHECK;
routine.config.executionStrategy = "invalid"; // ❌ Type error

// Functions return properly typed values
const found = getRoutineById("test"); // Returns RoutineFixture | undefined
const stats = getRoutineStats(); // Returns RoutineStats interface
```

## Testing Strategy

These fixtures support testing:
- Type-safe routine configuration validation
- Agent initialization and capability development
- Routine execution with different strategies
- Event-driven learning and adaptation
- Multi-agent swarm coordination
- Routine evolution proposals
- Performance optimization