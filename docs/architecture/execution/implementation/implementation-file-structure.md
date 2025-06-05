# 📁 Implementation File Structure: Organizing the Three-Tier Architecture

> **TL;DR**: This document defines the ideal file structure for implementing Vrooli's three-tier execution architecture, organized to support clear separation of concerns and easy extension.

> 📋 **Architecture Context**: For foundational concepts and tier responsibilities, see **[Architecture Overview](_ARCHITECTURE_OVERVIEW.md)**. This document focuses on **code organization** and file structure best practices.

---

## 🎯 Organizational Principles

### **1. Clear Tier Separation**
- Each tier has its own directory with focused responsibilities
- Cross-tier communication happens through well-defined interfaces
- Shared concerns are isolated in the `cross-cutting` directory

### **2. Domain-Driven Structure**  
- Components are grouped by domain (coordination, orchestration, execution)
- Related functionality is co-located for easier maintenance
- Clear separation of concerns within each domain

### **3. Pluggable Architecture**
- Navigators, strategies, and agents are pluggable components
- Clear interfaces enable easy extension and customization
- Provider adapters allow switching between different implementations

### **4. Comprehensive Testing**
- Tests are organized by scope (unit, integration, performance)
- Test fixtures and utilities support reliable testing
- Performance and scalability testing are first-class concerns

### **5. Documentation-Code Alignment**
- Code structure mirrors documentation organization
- Documentation files are co-located with relevant code
- Clear navigation between docs and implementation

---

## 📚 Current Documentation Structure

This reflects the actual organization of `/docs/architecture/execution/`:

```
docs/architecture/execution/
├── README.md                                          # Main execution architecture overview
├── _ARCHITECTURE_OVERVIEW.md                         # Three-tier architecture reference
├── _PERFORMANCE_REFERENCE.md                         # Performance targets and metrics
├── _DOCUMENTATION_MAINTENANCE.md                     # Documentation guidelines
├── quick-start-guide.md                              # 15-minute hands-on introduction
├── core-technologies.md                              # Foundational concepts and terminology
├── swarm-execution-integration.md                    # ChatConfigObject integration details
│
├── tiers/                                            # 🎯 Tier-Specific Documentation
│   ├── tier1-coordination-intelligence/              # Tier 1: Swarm coordination
│   │   ├── README.md                                 # Coordination intelligence overview
│   │   ├── swarm-state-machine.md                   # Complete state machine lifecycle
│   │   ├── implementation-architecture.md            # Technical implementation
│   │   ├── autonomous-operations.md                  # Autonomous swarm behavior
│   │   ├── metacognitive-framework.md               # AI reasoning framework
│   │   ├── moise-comprehensive-guide.md             # MOISE+ organizational modeling
│   │   ├── mcp-tools-reference.md                   # MCP tool implementations
│   │   └── why-prompt-based-coordination.md         # Design rationale
│   ├── tier2-process-intelligence/                   # Tier 2: Process orchestration
│   │   ├── README.md                                 # Process intelligence overview
│   │   ├── run-state-machine-diagram.md             # Complete routine lifecycle
│   │   ├── navigators.md                            # Universal navigator interface
│   │   ├── routine-types.md                         # Routine type definitions
│   │   ├── responsibilities.md                      # Tier 2 responsibilities
│   │   └── architecture.md                          # Process architecture
│   └── tier3-execution-intelligence/                 # Tier 3: Strategy execution
│       ├── README.md                                 # Execution intelligence overview
│       └── ... (other tier 3 docs)
│
├── implementation/                                   # 🛠️ Implementation Guidance
│   ├── implementation-guide.md                      # Step-by-step implementation
│   ├── implementation-roadmap.md                    # Phased implementation approach
│   ├── implementation-file-structure.md             # This document
│   └── concrete-examples.md                         # Practical implementation examples
│
├── event-driven/                                    # 📡 Event System Documentation
│   ├── README.md                                    # Event-driven architecture overview
│   ├── event-catalog.md                            # Complete event specifications
│   └── event-bus-protocol.md                       # Technical communication specs
│
├── emergent-capabilities/                           # 🌱 Self-Improving Systems
│   ├── README.md                                    # Emergent capabilities overview
│   ├── api-bootstrapping.md                        # Emergent API integration
│   ├── data-bootstrapping.md                       # Emergent documentation creation
│   ├── agent-examples/                             # Example intelligent agents
│   │   └── strategy-evolution-agents.md            # Strategy evolution implementation
│   └── routine-examples/                           # Example optimization routines
│       └── optimization-agents.md                  # Optimization agent examples
│
├── systems/                                        # 🛡️ Specialized Systems
│   ├── tool-approval-system.md                     # User control over tool execution
│   └── url-redirection-system.md                   # URL redirection for testing
│
├── operations/                                     # 🔧 Operations & Debugging
│   └── debugging-guide.md                          # Systematic troubleshooting
│
├── security/                                       # 🔒 Security & Safety
│   ├── README.md                                   # Security architecture
│   ├── security-boundaries.md                     # Security boundary definitions
│   └── security-implementation-patterns.md        # Security implementation
│
├── monitoring/                                     # 📊 Observability
│   └── README.md                                   # Monitoring architecture
│
├── communication/                                  # 🌐 Inter-Tier Communication
│   └── README.md                                   # Communication patterns
│
├── context-memory/                                 # 🧠 Context Management
│   └── README.md                                   # Context & memory architecture
│
├── resource-management/                            # 💰 Resource Management
│   └── README.md                                   # Resource allocation & optimization
│
├── resilience/                                     # 🛡️ Fault Tolerance
│   └── README.md                                   # Resilience architecture
│
├── ai-services/                                    # 🤖 AI Model Management
│   └── README.md                                   # Multi-provider AI services
│
├── types/                                          # 📝 Type Definitions
│   └── README.md                                   # Shared type system
│
└── planning/                                       # 📋 Planning & Metrics
    ├── success-metrics.md                          # Success measurement
    └── future-expansion-roadmap.md                 # Long-term vision
```

---

## 🏗️ Implementation Code Structure

### **🎯 Tier 1: Coordination Intelligence**

```
packages/server/src/services/execution/tier1/
├── coordination/                                    # Core coordination services
│   ├── swarmStateMachine.ts                       # Main swarm orchestrator
│   ├── swarmStateMachine.test.ts                  # Swarm state machine tests
│   ├── completionService.ts                       # AI response coordination
│   ├── completionService.test.ts                  # Completion service tests
│   ├── promptEngine.ts                            # Dynamic prompt generation
│   ├── promptEngine.test.ts                       # Prompt engine tests
│   ├── teamManager.ts                             # Team formation & management
│   ├── teamManager.test.ts                        # Team manager tests
│   ├── goalDecomposer.ts                          # Strategic goal breakdown
│   └── goalDecomposer.test.ts                     # Goal decomposer tests
├── intelligence/                                   # Metacognitive reasoning
│   ├── reasoningEngine.ts                         # Core AI reasoning loop
│   ├── reasoningEngine.test.ts                    # Reasoning engine tests
│   ├── strategySelector.ts                        # Strategy selection logic
│   ├── strategySelector.test.ts                   # Strategy selector tests
│   ├── patternRecognizer.ts                       # Pattern learning system
│   ├── patternRecognizer.test.ts                  # Pattern recognizer tests
│   ├── improvementTracker.ts                      # Continuous improvement
│   └── improvementTracker.test.ts                 # Improvement tracker tests
├── communication/                                  # Multi-agent coordination
│   ├── agentGraph.ts                              # Agent relationship mapping
│   ├── agentGraph.test.ts                         # Agent graph tests
│   ├── messageRouter.ts                           # Inter-agent messaging
│   ├── messageRouter.test.ts                      # Message router tests
│   ├── consensusBuilder.ts                        # Group decision making
│   ├── consensusBuilder.test.ts                   # Consensus builder tests
│   ├── conflictResolver.ts                        # Conflict resolution
│   └── conflictResolver.test.ts                   # Conflict resolver tests
├── organization/                                   # MOISE+ organizational modeling
│   ├── moiseSerializer.ts                         # MOISE+ spec handling
│   ├── moiseSerializer.test.ts                    # MOISE serializer tests
│   ├── roleManager.ts                             # Role definitions & assignment
│   ├── roleManager.test.ts                        # Role manager tests
│   ├── hierarchyBuilder.ts                        # Team hierarchy construction
│   ├── hierarchyBuilder.test.ts                   # Hierarchy builder tests
│   ├── normEnforcer.ts                            # Organizational norm enforcement
│   └── normEnforcer.test.ts                       # Norm enforcer tests
└── tools/                                         # MCP tool implementations
    ├── mcpToolRunner.ts                           # MCP tool execution
    ├── mcpToolRunner.test.ts                      # MCP tool runner tests
    ├── swarmStateTools.ts                         # Swarm state manipulation tools
    ├── swarmStateTools.test.ts                    # Swarm state tools tests
    ├── resourceTools.ts                           # Resource management tools
    ├── resourceTools.test.ts                      # Resource tools tests
    ├── eventTools.ts                              # Event subscription tools
    └── eventTools.test.ts                         # Event tools tests
```

### **⚙️ Tier 2: Process Intelligence**

```
packages/server/src/services/execution/tier2/
├── orchestration/                                  # Core process orchestration
│   ├── runStateMachine.ts                         # Main routine orchestrator
│   ├── runStateMachine.test.ts                    # Run state machine tests
│   ├── stepCoordinator.ts                         # Step execution coordination
│   ├── stepCoordinator.test.ts                    # Step coordinator tests
│   ├── branchManager.ts                           # Parallel branch management
│   ├── branchManager.test.ts                      # Branch manager tests
│   ├── dependencyResolver.ts                      # Step dependency resolution
│   ├── dependencyResolver.test.ts                 # Dependency resolver tests
│   ├── progressTracker.ts                         # Execution progress monitoring
│   └── progressTracker.test.ts                    # Progress tracker tests
├── navigation/                                     # Navigator registry & management
│   ├── navigatorRegistry.ts                       # Plugin navigator registry
│   ├── navigatorRegistry.test.ts                  # Navigator registry tests
│   ├── bpmnNavigator.ts                           # BPMN workflow navigator
│   ├── bpmnNavigator.test.ts                      # BPMN navigator tests
│   ├── langchainNavigator.ts                      # Langchain navigator
│   ├── langchainNavigator.test.ts                 # Langchain navigator tests
│   ├── customNavigator.ts                         # Custom workflow navigator
│   ├── customNavigator.test.ts                    # Custom navigator tests
│   ├── navigatorAdapter.ts                        # Navigator interface adapter
│   └── navigatorAdapter.test.ts                   # Navigator adapter tests
├── intelligence/                                   # Process optimization & learning
│   ├── pathOptimizer.ts                           # Execution path optimization
│   ├── pathOptimizer.test.ts                      # Path optimizer tests
│   ├── performanceAnalyzer.ts                     # Process performance analysis
│   ├── performanceAnalyzer.test.ts                # Performance analyzer tests
│   ├── bottleneckDetector.ts                      # Process bottleneck detection
│   ├── bottleneckDetector.test.ts                 # Bottleneck detector tests
│   ├── evolutionTracker.ts                        # Strategy evolution tracking
│   └── evolutionTracker.test.ts                   # Evolution tracker tests
├── context/                                        # Context lifecycle management
│   ├── contextManager.ts                          # Run context lifecycle
│   ├── contextManager.test.ts                     # Context manager tests
│   ├── blackboardManager.ts                       # Shared memory management
│   ├── blackboardManager.test.ts                  # Blackboard manager tests
│   ├── variableResolver.ts                        # Variable resolution
│   ├── variableResolver.test.ts                   # Variable resolver tests
│   ├── scopeManager.ts                            # Context scope management
│   └── scopeManager.test.ts                       # Scope manager tests
├── persistence/                                    # State persistence & recovery
│   ├── statePersistor.ts                          # State persistence service
│   ├── statePersistor.test.ts                     # State persistor tests
│   ├── checkpointManager.ts                       # Execution checkpointing
│   ├── checkpointManager.test.ts                  # Checkpoint manager tests
│   ├── recoveryManager.ts                         # Failure recovery
│   ├── recoveryManager.test.ts                    # Recovery manager tests
│   ├── migrationHandler.ts                        # State migration handling
│   └── migrationHandler.test.ts                   # Migration handler tests
└── validation/                                     # Input/output validation
    ├── stepValidator.ts                           # Step input validation
    ├── stepValidator.test.ts                      # Step validator tests
    ├── flowValidator.ts                           # Workflow validation
    ├── flowValidator.test.ts                      # Flow validator tests
    ├── schemaValidator.ts                         # Schema validation
    ├── schemaValidator.test.ts                    # Schema validator tests
    ├── securityValidator.ts                       # Security validation
    └── securityValidator.test.ts                  # Security validator tests
```

### **🛠️ Tier 3: Execution Intelligence**

```
packages/server/src/services/execution/tier3/
├── engine/                                         # Core execution engine
│   ├── unifiedExecutor.ts                        # Main execution coordinator
│   ├── unifiedExecutor.test.ts                   # Unified executor tests
│   ├── stepExecutor.ts                           # Individual step execution
│   ├── stepExecutor.test.ts                      # Step executor tests
│   ├── toolIntegrator.ts                         # Tool integration layer
│   ├── toolIntegrator.test.ts                    # Tool integrator tests
│   ├── resultProcessor.ts                        # Execution result processing
│   ├── resultProcessor.test.ts                   # Result processor tests
│   ├── errorHandler.ts                           # Execution error handling
│   └── errorHandler.test.ts                      # Error handler tests
├── strategies/                                     # Execution strategies
│   ├── strategyFactory.ts                        # Strategy selection factory
│   ├── strategyFactory.test.ts                   # Strategy factory tests
│   ├── conversationalStrategy.ts                 # Conversational execution
│   ├── conversationalStrategy.test.ts            # Conversational strategy tests
│   ├── reasoningStrategy.ts                      # Reasoning-based execution
│   ├── reasoningStrategy.test.ts                 # Reasoning strategy tests
│   ├── deterministicStrategy.ts                  # Deterministic execution
│   ├── deterministicStrategy.test.ts             # Deterministic strategy tests
│   ├── strategyEvolution.ts                      # Strategy learning & evolution
│   └── strategyEvolution.test.ts                 # Strategy evolution tests
├── intelligence/                                   # Execution learning & adaptation
│   ├── outcomeAnalyzer.ts                        # Execution outcome analysis
│   ├── outcomeAnalyzer.test.ts                   # Outcome analyzer tests
│   ├── adaptationEngine.ts                       # Strategy adaptation
│   ├── adaptationEngine.test.ts                  # Adaptation engine tests
│   ├── feedbackProcessor.ts                      # Feedback processing
│   ├── feedbackProcessor.test.ts                 # Feedback processor tests
│   ├── learningTracker.ts                        # Learning progress tracking
│   └── learningTracker.test.ts                   # Learning tracker tests
├── tools/                                         # Tool execution & management
│   ├── toolRunner.ts                             # Tool execution service
│   ├── toolRunner.test.ts                        # Tool runner tests
│   ├── toolRegistry.ts                           # Available tools registry
│   ├── toolRegistry.test.ts                      # Tool registry tests
│   ├── sandboxManager.ts                         # Sandboxed execution
│   ├── sandboxManager.test.ts                    # Sandbox manager tests
│   ├── apiIntegrator.ts                          # API integration tools
│   ├── apiIntegrator.test.ts                     # API integrator tests
│   ├── codeExecutor.ts                           # Code execution tools
│   └── codeExecutor.test.ts                      # Code executor tests
└── context/                                       # Execution context management
    ├── executionContext.ts                       # Step execution context
    ├── executionContext.test.ts                  # Execution context tests
    ├── resourceTracker.ts                        # Resource usage tracking
    ├── resourceTracker.test.ts                   # Resource tracker tests
    ├── creditsManager.ts                         # Credits & billing
    ├── creditsManager.test.ts                    # Credits manager tests
    ├── environmentManager.ts                     # Execution environment
    └── environmentManager.test.ts                # Environment manager tests
```

### **🌐 Cross-Cutting Concerns**

```
packages/server/src/services/execution/cross-cutting/
├── events/                                        # Event-driven intelligence
│   ├── eventBus.ts                              # Event bus implementation
│   ├── eventBus.test.ts                         # Event bus tests
│   ├── eventRouter.ts                           # Event routing service
│   ├── eventRouter.test.ts                      # Event router tests
│   ├── eventStorage.ts                          # Event persistence
│   ├── eventStorage.test.ts                     # Event storage tests
│   ├── eventAnalytics.ts                        # Event stream analytics
│   └── eventAnalytics.test.ts                   # Event analytics tests
├── security/                                      # Security & safety framework
│   ├── guardRails.ts                            # Synchronous guard rails
│   ├── guardRails.test.ts                       # Guard rails tests
│   ├── barrierSync.ts                           # Barrier synchronization
│   ├── barrierSync.test.ts                      # Barrier sync tests
│   ├── threatDetector.ts                        # Threat detection
│   ├── threatDetector.test.ts                   # Threat detector tests
│   ├── complianceChecker.ts                     # Compliance validation
│   ├── complianceChecker.test.ts               # Compliance tests
│   ├── emergencyStop.ts                         # Emergency stop system
│   └── emergencyStop.test.ts                   # Emergency stop tests
├── resources/                                     # Resource management
│   ├── resourceManager.ts                       # Resource allocation
│   ├── resourceManager.test.ts                  # Resource manager tests
│   ├── creditTracker.ts                         # Credit tracking
│   ├── creditTracker.test.ts                    # Credit tracker tests
│   ├── limitEnforcer.ts                         # Limit enforcement
│   ├── limitEnforcer.test.ts                    # Limit enforcer tests
│   ├── costOptimizer.ts                         # Cost optimization
│   ├── costOptimizer.test.ts                    # Cost optimizer tests
│   ├── usageAnalyzer.ts                         # Usage analysis
│   └── usageAnalyzer.test.ts                    # Usage analyzer tests
├── monitoring/                                    # Observability & analytics
│   ├── metricsCollector.ts                      # Metrics collection
│   ├── metricsCollector.test.ts                 # Metrics collector tests
│   ├── healthMonitor.ts                         # System health monitoring
│   ├── healthMonitor.test.ts                    # Health monitor tests
│   ├── performanceTracker.ts                    # Performance tracking
│   ├── performanceTracker.test.ts               # Performance tracker tests
│   ├── alertManager.ts                          # Alert management
│   ├── alertManager.test.ts                     # Alert manager tests
│   ├── dashboardService.ts                      # Monitoring dashboard
│   └── dashboardService.test.ts                 # Dashboard service tests
├── communication/                                 # Inter-tier communication
│   ├── messageQueue.ts                          # Message queue system
│   ├── messageQueue.test.ts                     # Message queue tests
│   ├── protocolHandler.ts                       # Communication protocols
│   ├── protocolHandler.test.ts                  # Protocol handler tests
│   ├── serializer.ts                            # Message serialization
│   ├── serializer.test.ts                       # Serializer tests
│   ├── interfaceAdapter.ts                      # Tier interface adaptation
│   └── interfaceAdapter.test.ts                 # Interface adapter tests
├── ai-services/                                   # AI model management
│   ├── modelManager.ts                          # Multi-provider model mgmt
│   ├── modelManager.test.ts                     # Model manager tests
│   ├── fallbackChains.ts                        # Model fallback handling
│   ├── fallbackChains.test.ts                   # Fallback chains tests
│   ├── costOptimizer.ts                         # Model cost optimization
│   ├── costOptimizer.test.ts                    # Cost optimizer tests
│   ├── qualityTracker.ts                        # Model quality tracking
│   ├── qualityTracker.test.ts                   # Quality tracker tests
│   └── providerAdapters/                        # Provider-specific adapters
│       ├── openaiAdapter.ts                     # OpenAI integration
│       ├── openaiAdapter.test.ts                # OpenAI adapter tests
│       ├── anthropicAdapter.ts                  # Anthropic integration
│       ├── anthropicAdapter.test.ts             # Anthropic adapter tests
│       ├── localAdapter.ts                      # Local model integration
│       └── localAdapter.test.ts                 # Local adapter tests
├── knowledge/                                     # Knowledge management
│   ├── knowledgeBase.ts                         # Unified knowledge system
│   ├── knowledgeBase.test.ts                    # Knowledge base tests
│   ├── vectorStore.ts                           # Vector storage & retrieval
│   ├── vectorStore.test.ts                      # Vector store tests
│   ├── semanticSearch.ts                        # Semantic search
│   ├── semanticSearch.test.ts                   # Semantic search tests
│   ├── knowledgeGraph.ts                        # Knowledge graph
│   ├── knowledgeGraph.test.ts                   # Knowledge graph tests
│   ├── learningAggregator.ts                    # Cross-system learning
│   └── learningAggregator.test.ts               # Learning aggregator tests
└── resilience/                                    # Fault tolerance & recovery
    ├── circuitBreaker.ts                        # Circuit breaker pattern
│       ├── circuitBreaker.test.ts                   # Circuit breaker tests
│   ├── retryManager.ts                          # Retry logic
│   ├── retryManager.test.ts                     # Retry manager tests
│   ├── errorClassifier.ts                       # Error classification
│   ├── errorClassifier.test.ts                  # Error classifier tests
│   ├── recoveryStrategies.ts                    # Recovery strategies
│   ├── recoveryStrategies.test.ts               # Recovery strategies tests
│   └── gracefulDegradation.ts                   # Graceful degradation
│       └── gracefulDegradation.test.ts              # Graceful degradation tests
```

### **🔌 External Integrations**

```
packages/server/src/services/execution/integration/
├── api/                                          # Core API Handlers
│   ├── rest/                                    # REST API handlers
│   │   ├── executionController.ts               # Execution REST endpoints
│   │   ├── executionController.test.ts          # REST controller tests
│   │   ├── swarmController.ts                   # Swarm REST endpoints
│   │   └── swarmController.test.ts              # Swarm controller tests
│   ├── graphql/                                 # GraphQL resolvers
│   │   ├── executionResolvers.ts                # Execution GraphQL resolvers
│   │   ├── executionResolvers.test.ts           # GraphQL resolver tests
│   │   ├── swarmResolvers.ts                    # Swarm GraphQL resolvers
│   │   └── swarmResolvers.test.ts               # Swarm resolver tests
│   ├── websocket/                               # WebSocket handlers
│   │   ├── executionSocket.ts                   # Real-time execution updates
│   │   ├── executionSocket.test.ts              # WebSocket handler tests
│   │   ├── swarmSocket.ts                       # Real-time swarm updates
│   │   └── swarmSocket.test.ts                  # Swarm socket tests
│   └── webhooks/                                # Webhook handlers
│       ├── executionWebhooks.ts                 # Execution webhook endpoints
│       ├── executionWebhooks.test.ts            # Webhook handler tests
│       ├── externalTriggers.ts                  # External system triggers
│       └── externalTriggers.test.ts             # External trigger tests
├── mcp/                                         # Model Context Protocol
│   ├── mcpServer.ts                            # MCP server implementation
│   ├── mcpServer.test.ts                       # MCP server tests
│   ├── toolProviders/                          # MCP tool providers
│   │   ├── executionTools.ts                   # Execution-specific MCP tools
│   │   ├── executionTools.test.ts              # Execution tools tests
│   │   ├── swarmTools.ts                       # Swarm-specific MCP tools
│   │   └── swarmTools.test.ts                  # Swarm tools tests
│   └── clientAdapters/                         # MCP client adapters
│       ├── mcpClientAdapter.ts                 # Generic MCP client adapter
│       ├── mcpClientAdapter.test.ts            # Client adapter tests
│       ├── customMcpClient.ts                  # Custom MCP client
│       └── customMcpClient.test.ts             # Custom client tests
├── externalServiceManager.ts                   # API keys & OAuth management
└── externalServiceManager.test.ts              # External service manager tests
```

### **🧪 Shared Test Data & Fixtures**

```
packages/server/src/services/execution/__test/
├── fixtures/                                   # Test data & fixtures
│   ├── routines/                               # Sample routine configurations
│   │   ├── simple-workflow.json               # Basic workflow examples
│   │   ├── complex-bpmn.json                  # BPMN workflow examples
│   │   ├── multi-tier-routine.json            # Cross-tier routine examples
│   │   └── strategy-evolution-examples.json   # Strategy evolution test data
│   ├── swarms/                                # Sample swarm configurations
│   │   ├── basic-swarm-config.json            # Simple swarm setups
│   │   ├── multi-agent-config.json            # Complex multi-agent swarms
│   │   ├── moise-organizational-config.json    # MOISE+ organization examples
│   │   └── autonomous-swarm-config.json       # Autonomous operation examples
│   ├── contexts/                              # Sample execution contexts
│   │   ├── tier1-context-examples.json        # Coordination context examples
│   │   ├── tier2-context-examples.json        # Process context examples
│   │   ├── tier3-context-examples.json        # Execution context examples
│   │   └── cross-tier-context.json            # Shared context examples
│   ├── events/                                # Sample event data
│   │   ├── coordination-events.json           # Tier 1 event examples
│   │   ├── process-events.json                # Tier 2 event examples
│   │   ├── execution-events.json              # Tier 3 event examples
│   │   └── emergency-events.json              # Safety/emergency event examples
│   └── agents/                                # Sample agent configurations
│       ├── optimization-agents.json           # Optimization agent configs
│       ├── security-agents.json               # Security agent configs
│       ├── quality-agents.json                # Quality assurance agent configs
│       └── monitoring-agents.json             # Monitoring agent configs
│       # Note: These are data configurations that create emergent agents,
│       # not hard-coded event processing components. The agents arise from
│       # swarm/bot configurations and routine definitions stored as data.
├── integration/                                # Integration test setup
│   ├── tier-integration-setup.ts              # Cross-tier integration setup
│   ├── event-flow-setup.ts                    # Event system test setup
│   ├── end-to-end-setup.ts                    # Full system test setup
│   └── performance-setup.ts                   # Performance test setup
└── utils/                                      # Test utilities
    ├── mocks/                                  # Mock implementations
    │   ├── mockEventBus.ts                     # Event bus mocks
    │   ├── mockAiService.ts                    # AI service mocks
    │   ├── mockResourceManager.ts              # Resource manager mocks
    │   └── mockNavigators.ts                   # Navigator mocks
    ├── builders/                               # Test data builders
    │   ├── swarmConfigBuilder.ts               # Swarm config builders
    │   ├── routineBuilder.ts                   # Routine builders
    │   ├── contextBuilder.ts                   # Context builders
    │   └── eventBuilder.ts                     # Event builders
    └── helpers/                                # Test helper functions
        ├── assertionHelpers.ts                 # Custom assertions
        ├── setupHelpers.ts                     # Test setup utilities
        ├── teardownHelpers.ts                  # Test cleanup utilities
        └── performanceHelpers.ts               # Performance test utilities
```

### **🎨 Frontend Integration**

```
packages/ui/src/execution/
├── components/                                  # React components
│   ├── SwarmDashboard/                         # Swarm monitoring dashboard
│   │   ├── SwarmDashboard.tsx                  # Main dashboard component
│   │   ├── SwarmDashboard.test.tsx             # Dashboard component tests
│   │   ├── SwarmMetrics.tsx                    # Metrics display component
│   │   ├── SwarmMetrics.test.tsx               # Metrics tests
│   │   ├── AgentList.tsx                       # Agent list component
│   │   └── AgentList.test.tsx                  # Agent list tests
│   ├── RoutineBuilder/                         # Visual routine builder
│   │   ├── RoutineBuilder.tsx                  # Main builder component
│   │   ├── RoutineBuilder.test.tsx             # Builder tests
│   │   ├── StepEditor.tsx                      # Step editing component
│   │   ├── StepEditor.test.tsx                 # Step editor tests
│   │   ├── NavigatorSelector.tsx               # Navigator selection
│   │   └── NavigatorSelector.test.tsx          # Navigator selector tests
│   ├── ExecutionMonitor/                       # Real-time execution monitoring
│   │   ├── ExecutionMonitor.tsx                # Main monitor component
│   │   ├── ExecutionMonitor.test.tsx           # Monitor tests
│   │   ├── ProgressVisualizer.tsx              # Progress visualization
│   │   ├── ProgressVisualizer.test.tsx         # Progress tests
│   │   ├── ResourceMetrics.tsx                 # Resource usage display
│   │   └── ResourceMetrics.test.tsx            # Resource metrics tests
│   └── EventViewer/                            # Event stream visualization
│       ├── EventViewer.tsx                     # Main event viewer
│       ├── EventViewer.test.tsx                # Event viewer tests
│       ├── EventFilter.tsx                     # Event filtering
│       ├── EventFilter.test.tsx                # Event filter tests
│       ├── EventTimeline.tsx                   # Timeline visualization
│       └── EventTimeline.test.tsx              # Timeline tests
├── hooks/                                      # React hooks for execution
│   ├── useSwarmState.ts                        # Swarm state management
│   ├── useSwarmState.test.ts                   # Swarm state hook tests
│   ├── useExecution.ts                         # Execution monitoring
│   ├── useExecution.test.ts                    # Execution hook tests
│   ├── useEvents.ts                            # Event stream handling
│   └── useEvents.test.ts                       # Event hook tests
├── stores/                                     # State management
│   ├── swarmStore.ts                          # Swarm state store
│   ├── swarmStore.test.ts                     # Swarm store tests
│   ├── executionStore.ts                      # Execution state store
│   ├── executionStore.test.ts                 # Execution store tests
│   ├── eventStore.ts                          # Event state store
│   └── eventStore.test.ts                     # Event store tests
└── types/                                      # Frontend-specific types
    ├── ui.ts                                   # UI component types
    ├── ui.test.ts                              # UI type tests
    ├── store.ts                                # Store types
    └── store.test.ts                           # Store type tests
```

### **📦 Shared Types & Utilities**

```
packages/shared/src/execution/
├── types/                                      # Core type definitions
│   ├── index.ts                               # Re-exports all types
│   ├── index.test.ts                          # Type re-export tests
│   ├── swarm.ts                               # Swarm, team, agent types
│   ├── swarm.test.ts                          # Swarm type tests
│   ├── routine.ts                             # Routine, run, step types
│   ├── routine.test.ts                        # Routine type tests
│   ├── context.ts                             # Context and memory types
│   ├── context.test.ts                        # Context type tests
│   ├── events.ts                              # Event type definitions
│   ├── events.test.ts                         # Event type tests
│   ├── strategies.ts                          # Strategy type definitions
│   ├── strategies.test.ts                     # Strategy type tests
│   ├── security.ts                            # Security and safety types
│   ├── security.test.ts                       # Security type tests
│   ├── resources.ts                           # Resource management types
│   └── resources.test.ts                      # Resource type tests
├── utils/                                      # Shared utilities
│   ├── validation.ts                          # Cross-tier validation
│   ├── validation.test.ts                     # Validation utility tests
│   ├── serialization.ts                       # Data serialization helpers
│   ├── serialization.test.ts                  # Serialization tests
│   ├── errors.ts                              # Common error definitions
│   ├── errors.test.ts                         # Error utility tests
│   ├── constants.ts                           # Shared constants
│   └── constants.test.ts                      # Constants tests
├── events/                                     # Event system foundations
│   ├── eventBus.ts                           # Core event bus interface
│   ├── eventBus.test.ts                      # Event bus interface tests
│   ├── eventTypes.ts                         # Event type registry
│   ├── eventTypes.test.ts                    # Event type registry tests
│   ├── eventValidation.ts                    # Event schema validation
│   └── eventValidation.test.ts               # Event validation tests
└── security/                                  # Shared security components
    ├── guardRails.ts                          # Guard-rail interfaces
    ├── guardRails.test.ts                     # Guard-rail tests
    ├── barriers.ts                            # Barrier synchronization
    ├── barriers.test.ts                       # Barrier synchronization tests
    ├── limits.ts                              # Resource limit definitions
    └── limits.test.ts                         # Resource limit tests
```

---

## 🚀 Benefits of This Structure

This structure supports our vision of **recursive self-improvement** by making it easy to:

- ✅ **Navigate between docs and code** - Clear alignment between documentation and implementation
- ✅ **Add new strategies and agents** - Clear extension points in each tier
- ✅ **Extend cross-cutting capabilities** - Centralized shared services  
- ✅ **Monitor and optimize performance** - Dedicated monitoring infrastructure
- ✅ **Integrate with external platforms** - Pluggable navigator architecture
- ✅ **Test and validate improvements** - Comprehensive testing structure
- ✅ **Scale development teams** - Clear domain boundaries and ownership
- ✅ **Maintain code quality** - Separation of concerns throughout

---

## 🎯 Implementation Guidelines

### **Directory Naming Conventions**
- Use `kebab-case` for directory names
- Group related functionality together
- Keep directory depth reasonable (max 4-5 levels)

### **File Naming Conventions**  
- Use `camelCase` for TypeScript files
- Use descriptive names that indicate purpose
- Group related files with common prefixes

### **Module Organization**
- Each directory should have an `index.ts` for clean imports
- Keep files focused on single responsibilities
- Use barrel exports for cleaner import statements

### **Testing Structure**
- Mirror the source structure in test directories
- Use descriptive test file names with `.test.ts` suffix
- Group related tests in subdirectories

### **Documentation Alignment**
- Keep documentation structure aligned with code structure
- Co-locate relevant documentation with code when appropriate
- Use clear cross-references between docs and implementation

This structure ensures that as our execution architecture grows and evolves, both the documentation and codebase remain organized, maintainable, and easy to extend. 