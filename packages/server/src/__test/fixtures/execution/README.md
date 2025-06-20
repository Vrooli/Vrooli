# 🧪 Execution Architecture Test Fixtures

This directory contains comprehensive test fixtures for Vrooli's three-tier AI execution system, demonstrating emergent capabilities, agent collaboration, and self-improving intelligence.

## 📚 Architecture Documentation References

Before exploring these fixtures, familiarize yourself with the core architecture:

- **[Main Execution Architecture](../../../../../../docs/architecture/execution/README.md)** - Vision and overview
- **[Architecture Overview](../../../../../../docs/architecture/execution/_ARCHITECTURE_OVERVIEW.md)** - Three-tier quick reference
- **[Core Technologies](../../../../../../docs/architecture/execution/core-technologies.md)** - Foundational concepts
- **[Quick Start Guide](../../../../../../docs/architecture/execution/quick-start-guide.md)** - 15-minute hands-on introduction
- **[Testing Overview](../../../../../../docs/testing/fixtures-overview.md)** - Unified fixture architecture

## 🎯 Purpose & Role in Testing Ecosystem

Execution fixtures serve as the **AI system testing layer** in our unified testing pipeline, providing:

1. **Realistic AI Scenarios** - Test the three-tier intelligence system with production-like conditions
2. **Emergent Capability Validation** - Verify that capabilities emerge from agent collaboration, not hard-coded features
3. **Cross-Tier Integration** - Ensure seamless communication between coordination, process, and execution layers
4. **Performance Benchmarking** - Track routine evolution from conversational to deterministic strategies
5. **Safety Architecture Testing** - Validate multi-layered safety mechanisms at each tier

## 🏗️ Current Directory Structure

```
execution/
├── tier1-coordination/              # Tier 1: Coordination Intelligence
│   ├── swarms/                     # Dynamic swarm configurations
│   │   ├── customer-support/       # Domain-specific swarm examples
│   │   ├── security-response/
│   │   ├── healthcare-compliance/
│   │   └── financial-trading/
│   ├── moise-organizations/        # MOISE+ organizational structures
│   └── coordination-tools/         # MCP tools for natural coordination
│
├── tier2-process/                  # Tier 2: Process Intelligence
│   ├── routines/                   # Versioned workflow definitions
│   │   ├── by-evolution-stage/    # Show progression through strategies
│   │   └── by-domain/             # Current organization (secondary view)
│   ├── navigators/                # Format-specific translators
│   └── run-states/                # RunStateMachine examples
│
├── tier3-execution/               # Tier 3: Execution Intelligence
│   ├── strategies/                # Execution strategy examples
│   ├── unified-executor/          # UnifiedExecutor configurations
│   └── context-management/        # Execution context fixtures
│
├── emergent-capabilities/         # Cross-tier emergent behaviors
│   ├── agent-types/              # Specialized agent configurations
│   ├── evolution-examples/       # Routine evolution scenarios
│   └── self-improvement/         # Recursive capability growth
│
└── integration-scenarios/        # Complete system examples
    ├── healthcare-compliance/    # Full three-tier integration
    ├── financial-trading/
    └── customer-service/
```

## 🏭 Ideal Execution Fixture Architecture

### Type-Safe Factory Pattern

```typescript
// Core execution fixture factory interface
interface ExecutionFixtureFactory<TExecution, TContext> {
  // Three-tier system scenarios
  tiers: {
    tier1: CoordinationScenarios
    tier2: ProcessScenarios  
    tier3: ExecutionScenarios
  }
  
  // Cross-tier scenario collections
  scenarios: {
    [key: string]: TExecution
  }
  
  // Factory methods for creating test scenarios
  createSwarm: (config: SwarmConfig) => SwarmExecution
  createRoutine: (type: RoutineType, overrides?: Partial<Routine>) => RoutineExecution
  createAgent: (role: AgentRole, capabilities?: string[]) => Agent
  
  // AI testing helpers
  simulateExecution: (scenario: ExecutionScenario) => Promise<TestResult>
  validateEmergentBehavior: (execution: TExecution) => EmergentResult
  measureEvolution: (v1: TExecution, v2: TExecution) => EvolutionMetrics
  
  // Integration testing
  testCrossTierFlow: (input: any) => Promise<TierFlowResult>
  simulateAgentCollaboration: (agents: Agent[]) => CollaborationResult
  validateSafetyMechanisms: (execution: TExecution) => SafetyResult
  
  // Performance testing
  benchmarkStrategy: (strategy: ExecutionStrategy) => PerformanceMetrics
  compareStrategies: (strategies: ExecutionStrategy[]) => ComparisonResult
}
```

### Execution Categories

#### 1. **Tier 1 (Coordination) Fixtures**
```typescript
interface CoordinationFixtures {
  // Swarm configurations
  swarms: {
    minimal: SwarmConfig         // Basic swarm with 2-3 agents
    standard: SwarmConfig        // Typical 5-10 agent swarm
    large: SwarmConfig          // Stress-test with 20+ agents
    specialized: {
      customerSupport: SwarmConfig
      securityResponse: SwarmConfig
      healthcareCompliance: SwarmConfig
      financialTrading: SwarmConfig
    }
  }
  
  // MOISE+ organizations
  organizations: {
    flat: MOISEOrganization      // Simple peer-to-peer
    hierarchical: MOISEOrganization  // Multi-level management
    matrix: MOISEOrganization    // Cross-functional teams
    dynamic: MOISEOrganization   // Self-organizing structure
  }
  
  // Coordination patterns
  patterns: {
    consensus: CoordinationPattern    // Democratic decisions
    delegation: CoordinationPattern   // Task assignment
    negotiation: CoordinationPattern  // Resource allocation
    emergence: CoordinationPattern    // Self-organization
  }
}
```

#### 2. **Tier 2 (Process) Fixtures**
```typescript
interface ProcessFixtures {
  // Routine evolution stages
  routinesByStage: {
    conversational: Routine[]    // Novel problem-solving
    reasoning: Routine[]         // Pattern-based logic
    deterministic: Routine[]     // Optimized automation
    routing: Routine[]          // Multi-routine coordination
  }
  
  // Domain-specific routines
  routinesByDomain: {
    security: SecurityRoutines
    medical: MedicalRoutines
    performance: PerformanceRoutines
    system: SystemRoutines
    bpmn: BPMNWorkflows
  }
  
  // Navigator patterns
  navigators: {
    native: VrooliNavigator
    bpmn: BPMNNavigator
    custom: CustomNavigator[]
  }
  
  // Run state examples
  runStates: {
    sequential: RunStateMachine
    parallel: RunStateMachine
    conditional: RunStateMachine
    loop: RunStateMachine
  }
}
```

#### 3. **Tier 3 (Execution) Fixtures**
```typescript
interface ExecutionFixtures {
  // Strategy configurations
  strategies: {
    conversational: StrategyConfig   // Human-like reasoning
    reasoning: StrategyConfig        // Structured frameworks
    deterministic: StrategyConfig    // Direct automation
    routing: StrategyConfig         // Multi-path coordination
  }
  
  // Context management
  contexts: {
    minimal: ExecutionContext
    withSharedState: ExecutionContext
    withResourceLimits: ExecutionContext
    withSafetyConstraints: ExecutionContext
  }
  
  // Tool orchestration
  toolConfigs: {
    basic: ToolConfig              // Essential tools only
    extended: ToolConfig           // Domain-specific tools
    restricted: ToolConfig         // Safety-limited tools
    adaptive: ToolConfig           // Dynamic tool selection
  }
}
```

#### 4. **Emergent Capability Fixtures**
```typescript
interface EmergentFixtures {
  // Specialized agents
  agents: {
    security: SecurityAgent[]        // Threat detection, compliance
    quality: QualityAgent[]         // Validation, bias detection
    optimization: OptimizationAgent[] // Performance, cost reduction
    monitoring: MonitoringAgent[]    // Observability, analytics
  }
  
  // Evolution scenarios
  evolution: {
    customerSupport: EvolutionPath   // v1 → v2 → v3 progression
    securityScanning: EvolutionPath
    dataProcessing: EvolutionPath
    decisionMaking: EvolutionPath
  }
  
  // Self-improvement patterns
  learning: {
    patternRecognition: LearningScenario
    strategyProposal: LearningScenario
    collaborativeReview: LearningScenario
    knowledgeTransfer: LearningScenario
  }
  
  // Emergence validation
  validation: {
    measureEmergence: (before: State, after: State) => EmergenceMetrics
    trackCapabilities: (timeline: Event[]) => CapabilityGrowth
    validateCompounding: (iterations: Result[]) => CompoundEffect
  }
}
```

#### 5. **Integration Scenario Fixtures**
```typescript
interface IntegrationFixtures {
  // Complete workflows
  scenarios: {
    healthcare: {
      compliance: FullScenario      // HIPAA/PHI detection
      diagnosis: FullScenario       // Medical validation
      patientCare: FullScenario     // Treatment workflows
    }
    financial: {
      trading: FullScenario         // Risk assessment
      compliance: FullScenario      // Regulatory checks
      fraud: FullScenario          // Pattern detection
    }
    customer: {
      support: FullScenario         // Inquiry resolution
      onboarding: FullScenario      // User setup
      retention: FullScenario       // Engagement
    }
  }
  
  // Cross-tier flows
  flows: {
    simple: TierFlow               // Linear progression
    complex: TierFlow              // Multiple branches
    adaptive: TierFlow             // Dynamic routing
    emergent: TierFlow             // Self-organizing
  }
  
  // Performance benchmarks
  benchmarks: {
    latency: BenchmarkSuite
    throughput: BenchmarkSuite
    cost: BenchmarkSuite
    quality: BenchmarkSuite
  }
}
```

## 📊 Current Coverage Analysis

### ✅ Well-Covered Areas
- **Swarm Configurations**: Good variety of domain-specific swarms
- **Routine Evolution**: Clear progression through execution strategies
- **Agent Types**: Comprehensive specialized agent examples
- **MOISE+ Organizations**: Multiple organizational structures
- **Integration Scenarios**: Healthcare, financial, and customer service

### ⚠️ Areas Needing Improvement
- **Failure Scenarios**: Limited error handling and recovery testing
- **Performance Testing**: No systematic benchmarking fixtures
- **Multi-Agent Coordination**: Basic collaboration patterns only
- **Cross-Tier Communication**: Limited event bus testing
- **Resource Management**: Minimal credit/limit testing
- **Safety Validation**: Basic safety mechanism coverage

### ❌ Missing Coverage
- **Stress Testing**: No fixtures for system limits
- **Chaos Engineering**: No failure injection scenarios
- **Learning Validation**: Limited emergent capability measurement
- **A/B Testing**: No strategy comparison fixtures
- **Migration Testing**: No evolution path validation
- **Rollback Scenarios**: No regression testing fixtures

## 🚀 Usage Examples

### Testing Three-Tier Integration
```typescript
import { ExecutionFixtures } from '../fixtures/execution';

describe('Healthcare Compliance Flow', () => {
  it('should detect PHI in patient records', async () => {
    // Tier 1: Form specialized swarm
    const swarm = ExecutionFixtures.createSwarm({
      type: 'healthcare-compliance',
      agents: ['phi-detector', 'hipaa-validator', 'audit-logger']
    });
    
    // Tier 2: Select appropriate routine
    const routine = ExecutionFixtures.createRoutine(
      'PHI_DETECTION_V3',
      { executionStrategy: 'deterministic' }
    );
    
    // Tier 3: Execute with safety constraints
    const result = await ExecutionFixtures.simulateExecution({
      swarm,
      routine,
      input: { document: 'patient-record.pdf' },
      constraints: { maxTokens: 1000, timeout: 30000 }
    });
    
    // Validate results
    expect(result.phiDetected).toBe(true);
    expect(result.complianceScore).toBeGreaterThan(0.95);
    expect(result.auditTrail).toBeDefined();
  });
});
```

### Testing Routine Evolution
```typescript
describe('Customer Support Evolution', () => {
  it('should show performance improvement through stages', async () => {
    const evolution = ExecutionFixtures.routinesByStage;
    
    // Compare v1 (conversational) to v3 (deterministic)
    const v1Result = await ExecutionFixtures.benchmarkStrategy(
      evolution.conversational.find(r => r.name === 'CUSTOMER_INQUIRY_V1')
    );
    
    const v3Result = await ExecutionFixtures.benchmarkStrategy(
      evolution.deterministic.find(r => r.name === 'CUSTOMER_INQUIRY_V3')
    );
    
    // Verify improvements
    expect(v3Result.avgDuration).toBeLessThan(v1Result.avgDuration * 0.1);
    expect(v3Result.avgCost).toBeLessThan(v1Result.avgCost * 0.2);
    expect(v3Result.accuracy).toBeGreaterThan(v1Result.accuracy);
  });
});
```

### Testing Emergent Capabilities
```typescript
describe('Security Agent Learning', () => {
  it('should evolve threat detection patterns', async () => {
    const securityAgent = ExecutionFixtures.agents.security[0];
    
    // Simulate event stream
    const events = [
      { type: 'api_call', pattern: 'suspicious', timestamp: Date.now() },
      { type: 'auth_failure', pattern: 'brute_force', timestamp: Date.now() + 1000 },
      // ... more events
    ];
    
    // Let agent process and learn
    const learning = await ExecutionFixtures.validateEmergentBehavior({
      agent: securityAgent,
      events,
      duration: 60000 // 1 minute simulation
    });
    
    // Verify pattern recognition improved
    expect(learning.patternsIdentified).toBeGreaterThan(5);
    expect(learning.falsePositiveRate).toBeLessThan(0.1);
    expect(learning.newCapabilities).toContain('predictive_threat_detection');
  });
});
```

### Testing Cross-Tier Communication
```typescript
describe('Event-Driven Coordination', () => {
  it('should handle tier communication via event bus', async () => {
    const flow = await ExecutionFixtures.testCrossTierFlow({
      trigger: 'user_complaint_received',
      expectedPath: [
        'tier1.swarm.activate',
        'tier2.routine.select',
        'tier3.execute.start',
        'tier3.execute.complete',
        'tier2.routine.update',
        'tier1.swarm.report'
      ]
    });
    
    expect(flow.eventsEmitted).toHaveLength(6);
    expect(flow.latency).toBeLessThan(5000);
    expect(flow.success).toBe(true);
  });
});
```

## 🏗️ Best Practices

### 1. **Fixture Design Principles**
- **Realistic**: Match production scenarios as closely as possible
- **Isolated**: Each fixture should be independent and repeatable
- **Documented**: Include clear descriptions and usage examples
- **Versioned**: Track evolution of fixtures with routines
- **Typed**: Full TypeScript support with proper interfaces

### 2. **Testing Strategies**
- **Unit**: Test individual tier components in isolation
- **Integration**: Test cross-tier communication and coordination
- **End-to-End**: Test complete user scenarios through all tiers
- **Performance**: Benchmark and compare execution strategies
- **Chaos**: Test failure scenarios and recovery mechanisms

### 3. **Evolution Testing**
```typescript
// Track routine progression through stages
const evolutionPath = {
  v1: { strategy: 'conversational', metrics: {...} },
  v2: { strategy: 'reasoning', metrics: {...} },
  v3: { strategy: 'deterministic', metrics: {...} }
};

// Validate each transition improves key metrics
expect(evolutionPath.v2.metrics.speed).toBeLessThan(evolutionPath.v1.metrics.speed);
expect(evolutionPath.v3.metrics.cost).toBeLessThan(evolutionPath.v2.metrics.cost);
```

### 4. **Safety Testing**
```typescript
// Multi-layered safety validation
const safetyLayers = {
  synchronous: {    // <10ms checks
    inputValidation: true,
    resourceLimits: true,
    emergencyStop: true
  },
  asynchronous: {   // Event-driven monitoring
    complianceAgents: true,
    qualityAgents: true,
    auditTrail: true
  },
  teamSpecific: {   // Domain expertise
    medicalSafety: true,
    financialCompliance: true,
    dataPrivacy: true
  }
};
```

## 🔧 Contributing New Fixtures

### Fixture Checklist
- [ ] **Correct tier placement** - Does it belong in Tier 1, 2, or 3?
- [ ] **Execution strategy specified** - For routines, which strategy?
- [ ] **Evolution path documented** - How does it improve over time?
- [ ] **Integration points identified** - How does it work with other tiers?
- [ ] **Performance metrics included** - Speed, cost, quality measurements
- [ ] **Safety considerations addressed** - What constraints apply?
- [ ] **TypeScript types defined** - Full type safety
- [ ] **Examples provided** - Clear usage demonstrations
- [ ] **Tests written** - Fixture validation tests

### Adding a New Fixture

1. **Identify the tier and category**
   ```typescript
   // Example: Adding a new security routine
   // Location: tier2-process/routines/by-domain/securityRoutines.ts
   ```

2. **Create the fixture with proper typing**
   ```typescript
   export const newSecurityRoutine: RoutineFixture = {
     id: generatePK(),
     name: "ADVANCED_THREAT_DETECTION",
     description: "ML-based threat pattern recognition",
     config: {
       executionStrategy: "reasoning",
       version: "1.0.0",
       // ... full configuration
     }
   };
   ```

3. **Add evolution examples if applicable**
   ```typescript
   export const threatDetectionEvolution = {
     v1_conversational: { /* initial implementation */ },
     v2_reasoning: { /* pattern-based improvement */ },
     v3_deterministic: { /* fully optimized */ }
   };
   ```

4. **Include integration examples**
   ```typescript
   export const threatDetectionFlow = {
     tier1: "Security swarm coordinates scanning",
     tier2: "Threat detection routine executes",
     tier3: "Reasoning strategy analyzes patterns",
     emergent: "New threat signatures learned"
   };
   ```

5. **Write validation tests**
   ```typescript
   describe('Threat Detection Fixture', () => {
     it('should have valid configuration', () => {
       expect(newSecurityRoutine.config.executionStrategy).toBeDefined();
       expect(routineValidation.validate(newSecurityRoutine)).toBe(true);
     });
   });
   ```

## 📈 Metrics & Monitoring

### Key Metrics to Track
1. **Evolution Metrics**
   - Speed improvements (execution time reduction)
   - Cost reduction (token/credit usage)
   - Quality gains (accuracy, completeness)
   - Learning rate (new patterns identified)

2. **Integration Metrics**
   - Cross-tier latency
   - Event bus throughput
   - Coordination efficiency
   - Resource utilization

3. **Emergent Metrics**
   - New capabilities discovered
   - Agent collaboration effectiveness
   - Knowledge transfer rate
   - Compound growth effects

## 🚦 Roadmap for Improvement

### Phase 1: Fill Coverage Gaps (Immediate)
- [ ] Add comprehensive failure scenarios
- [ ] Create performance benchmarking suite
- [ ] Expand multi-agent coordination patterns
- [ ] Add stress testing fixtures

### Phase 2: Enhanced Testing (Short-term)
- [ ] Implement chaos engineering fixtures
- [ ] Add A/B testing framework
- [ ] Create migration path validators
- [ ] Build regression test suite

### Phase 3: Advanced Capabilities (Long-term)
- [ ] Machine learning validation fixtures
- [ ] Distributed system testing
- [ ] Real-time monitoring integration
- [ ] Automated fixture generation

## 🔗 Integration with Testing Ecosystem

Execution fixtures integrate with the broader testing architecture:

1. **API Fixtures** (`@vrooli/shared`) - Provide data structures
2. **Database Fixtures** (`server/fixtures/db`) - Handle persistence
3. **Event Fixtures** (`@vrooli/shared`) - Simulate real-time events
4. **Config Fixtures** (`@vrooli/shared`) - Supply configurations

### Round-Trip Testing Example
```typescript
// Complete data flow through all fixture types
const roundTripTest = async () => {
  // 1. API fixture provides input
  const apiInput = apiFixtures.routineFixtures.complete.create;
  
  // 2. Execution fixture processes
  const execution = await ExecutionFixtures.simulateExecution({
    routine: apiInput,
    strategy: 'reasoning'
  });
  
  // 3. Database fixture persists
  const saved = await RoutineDbFactory.create(execution.result);
  
  // 4. Event fixture emits updates
  await eventFixtures.emit('routine.completed', saved);
  
  // Validate complete flow
  expect(saved.id).toBeDefined();
  expect(execution.success).toBe(true);
};
```

---

*Remember: The three-tier architecture enables compound intelligence through emergent agent collaboration, not through built-in features. Every fixture should demonstrate this principle.*