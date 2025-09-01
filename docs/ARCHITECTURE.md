# Architecture Documentation

## Overview

Vrooli is a self-improving intelligence system that combines AI agents with local resources to build autonomous capabilities. The architecture enables recursive improvement where agents build tools, tools make agents smarter, and the cycle continues indefinitely.

## Core Architecture Principles

### 1. Three-Tier Intelligence System

<!-- EMBED:ARCHITECTURE:START -->
The system operates on three distinct intelligence tiers:

**Tier 1: Coordination Intelligence**
- Orchestrates agent swarms across multiple execution environments
- Allocates computational and memory resources
- Manages strategic planning and goal decomposition
- Handles high-level decision making

**Tier 2: Process Intelligence**  
- Manages workflow execution and sequencing
- Converts successful solutions into reusable patterns
- Handles routine navigation and task distribution
- Maintains system state and context

**Tier 3: Execution Intelligence**
- Interfaces directly with local resources (databases, AI models, automation tools)
- Builds applications from scenarios
- Performs tool integration and API calls
- Handles concrete implementation details
<!-- EMBED:ARCHITECTURE:END -->

### 2. Event-Driven Communication

<!-- EMBED:EVENT_BUS:START -->
All components communicate through a Redis-based event bus that enables:
- Asynchronous message passing between agents
- Collective learning through shared experiences
- Real-time coordination without tight coupling
- Fault tolerance through message persistence
<!-- EMBED:EVENT_BUS:END -->

### 3. Resource Layer

<!-- EMBED:RESOURCES:START -->
The resource layer provides 30+ local services including:
- **AI/Inference**: Ollama, LiteLLM, OpenAI, Claude
- **Storage**: PostgreSQL, Redis, Qdrant, Neo4j, MinIO
- **Automation**: N8n, Node-RED, Windmill
- **Agents**: Browserless, AutoGPT, CrewAI
- **Execution**: Judge0, FFmpeg, various specialized tools

Resources are shared across all applications, enabling:
- Resource pooling and efficiency
- Cross-application learning
- Unified state management
- Reduced redundancy
<!-- EMBED:RESOURCES:END -->

## Design Decisions

### 1. Scenario-Driven Development

<!-- EMBED:SCENARIOS:START -->
**Decision**: Use scenarios as the primary unit of capability development.

**Rationale**: 
- Scenarios serve triple duty: validation, revenue generation, and capability expansion
- Each scenario becomes a permanent tool in the system
- Enables business-oriented development while building technical capabilities

**Trade-offs**:
- (+) Clear business value for each development effort
- (+) Natural decomposition of complex systems
- (-) Requires upfront scenario design
- (-) May not cover all technical requirements
<!-- EMBED:SCENARIOS:END -->

### 2. Local Resource Architecture

<!-- EMBED:LOCAL_RESOURCES:START -->
**Decision**: Run all resources locally rather than using cloud services.

**Rationale**:
- Complete control over data and processing
- No API rate limits or usage costs
- Enables unrestricted experimentation
- Resources can be modified and extended

**Trade-offs**:
- (+) No ongoing service costs
- (+) Complete data privacy
- (+) Unlimited usage
- (-) Higher initial setup complexity
- (-) Requires more powerful hardware
- (-) User responsible for maintenance
<!-- EMBED:LOCAL_RESOURCES:END -->

### 3. Bash-to-Python Migration

<!-- EMBED:ORCHESTRATION:START -->
**Decision**: Migrate from bash-based orchestration to Python for critical components.

**Rationale**:
- Bash subprocess management proved unreliable at scale
- Python provides better async handling and error recovery
- Structured logging and debugging capabilities
- Better cross-platform compatibility

**Implementation**: Completed August 2025 for app orchestration
- Location: `scripts/scenarios/tools/orchestrator/app_orchestrator.py`
- Trigger: `vrooli develop` command
- Benefits: Eliminated fork bomb issues, improved startup time by 40%
<!-- EMBED:ORCHESTRATION:END -->

### 4. Semantic Knowledge System

<!-- EMBED:SEMANTIC_KNOWLEDGE:START -->
**Decision**: Implement Qdrant-based semantic search across all content.

**Rationale**:
- Enables AI agents to discover relevant code/workflows/documentation
- Supports cross-project learning and pattern discovery
- Provides context-aware assistance

**Architecture**:
- Unified embedding service eliminates code duplication
- Parallel processing with 16 workers for performance
- Separate collections for workflows, scenarios, code, docs, resources
- App-based isolation with cross-app search capabilities
<!-- EMBED:SEMANTIC_KNOWLEDGE:END -->

## System Components

### 1. CLI System (`/cli`)

<!-- EMBED:CLI:START -->
The Vrooli CLI provides the primary interface for system interaction:
- **Command Router**: Dispatches commands to appropriate handlers
- **Resource Management**: Start/stop/configure local resources
- **Scenario Execution**: Build and run scenario-based applications
- **Development Tools**: Testing, debugging, and monitoring utilities
<!-- EMBED:CLI:END -->

### 2. Scenario System (`/scripts/scenarios`)

<!-- EMBED:SCENARIO_SYSTEM:START -->
Scenarios define complete applications that agents can build:
- **PRD Files**: Product requirement documents defining the application
- **Initialization**: Workflows and configurations for the app
- **Validation**: Tests to ensure the scenario works correctly
- **Conversion**: Tools to transform scenarios into running applications
<!-- EMBED:SCENARIO_SYSTEM:END -->

### 3. Resource System (`/resources`)

<!-- EMBED:RESOURCE_SYSTEM:START -->
Resources provide capabilities to the system:
- **Standardized Interface**: Each resource has a CLI for uniform access
- **Adapters**: Enable resources to work together
- **Configuration**: Declarative configuration for each resource
- **Lifecycle Management**: Automated install, start, stop, upgrade
<!-- EMBED:RESOURCE_SYSTEM:END -->

### 4. Automation System

<!-- EMBED:AUTOMATION:START -->
Multiple automation layers working together:
- **N8n**: Visual workflow automation with 400+ integrations
- **Node-RED**: Flow-based programming for event handling
- **Custom Orchestration**: Python-based app orchestration
- **Task Loops**: Autonomous improvement cycles
<!-- EMBED:AUTOMATION:END -->

## Patterns

### 1. Recursive Improvement Pattern

<!-- EMBED:RECURSIVE_PATTERN:START -->
```
1. Agent identifies a problem or opportunity
2. Agent builds a solution using available resources
3. Solution is validated through testing
4. Successful solution becomes a new tool/workflow
5. New tool enables solving more complex problems
6. Return to step 1 with enhanced capabilities
```

This pattern ensures the system continuously improves without external intervention.
<!-- EMBED:RECURSIVE_PATTERN:END -->

### 2. Resource Composition Pattern

<!-- EMBED:COMPOSITION_PATTERN:START -->
Resources are designed to be composable:
- Each resource provides specific capabilities
- Resources can call other resources through adapters
- Workflows combine multiple resources for complex tasks
- No single resource tries to do everything

Example: Email processing scenario uses Ollama (AI), PostgreSQL (storage), N8n (workflow), and Redis (caching) together.
<!-- EMBED:COMPOSITION_PATTERN:END -->

### 3. Fail-Safe Pattern

<!-- EMBED:FAILSAFE_PATTERN:START -->
Multiple levels of failure handling:
- Process monitoring with automatic restart
- Resource health checks with fallback options  
- Workflow error branches for graceful degradation
- System-wide circuit breakers for cascading failures
<!-- EMBED:FAILSAFE_PATTERN:END -->

## Trade-offs

### 1. Complexity vs Capability

<!-- EMBED:COMPLEXITY_TRADEOFF:START -->
**Trade-off**: System complexity increases with each added resource and scenario.

**Mitigation**:
- Standardized interfaces reduce integration complexity
- Semantic search helps navigate large codebases
- Automated testing catches integration issues
- Documentation generation keeps knowledge current
<!-- EMBED:COMPLEXITY_TRADEOFF:END -->

### 2. Performance vs Flexibility

<!-- EMBED:PERFORMANCE_TRADEOFF:START -->
**Trade-off**: Generic interfaces and abstraction layers add overhead.

**Mitigation**:
- Critical paths optimized with direct implementations
- Caching at multiple levels (Redis, embeddings, file system)
- Parallel processing where possible
- Lazy loading of resources
<!-- EMBED:PERFORMANCE_TRADEOFF:END -->

### 3. Autonomy vs Control

<!-- EMBED:AUTONOMY_TRADEOFF:START -->
**Trade-off**: More autonomous behavior means less predictability.

**Mitigation**:
- All autonomous actions logged and traceable
- Human approval required for critical operations
- Rollback capabilities for all changes
- Configurable automation levels
<!-- EMBED:AUTONOMY_TRADEOFF:END -->

## Future Architecture Evolution

### Phase 1: Current State (Completed)
- Local resource integration
- Scenario-based development
- Basic autonomous improvement
- Semantic knowledge system

### Phase 2: Enhanced Intelligence (In Progress)
- Multi-agent coordination protocols
- Advanced pattern recognition
- Cross-scenario learning
- Improved resource optimization

### Phase 3: Distributed Intelligence (Planned)
- Multi-node deployment capabilities
- Federated learning across instances
- Specialized compute nodes (GPU, CPU, Storage)
- Global knowledge synchronization

### Phase 4: Full Autonomy (Future)
- Self-modifying code generation
- Automatic scenario creation from requirements
- Predictive resource provisioning
- Goal-directed autonomous operation

## Performance Characteristics

<!-- EMBED:PERFORMANCE:START -->
### Current Metrics
- **Startup Time**: ~30 seconds for full system
- **Embedding Generation**: 50-100 items/second with parallel processing
- **Workflow Execution**: 10-50ms overhead per node
- **Resource Response**: <100ms for local resources
- **Search Latency**: 50-200ms for semantic search

### Bottlenecks
- Embedding generation for large codebases (being optimized)
- Cold start of AI models (mitigated with preloading)
- Database connection pooling (resolved with connection management)
<!-- EMBED:PERFORMANCE:END -->

## Security Considerations

<!-- EMBED:SECURITY:START -->
### Security Layers
1. **Resource Isolation**: Each resource runs in its own process
2. **Permission System**: Fine-grained access control for operations
3. **Audit Logging**: All actions tracked and attributed
4. **Secret Management**: Vault integration for sensitive data
5. **Network Segmentation**: Resources communicate through defined interfaces

### Known Vulnerabilities
- Local resource access requires system permissions
- Shared Redis bus could be attack vector
- Generated code needs sandboxing

See SECURITY.md for detailed security documentation.
<!-- EMBED:SECURITY:END -->

## Conclusion

The Vrooli architecture represents a fundamental shift from traditional software development to self-improving intelligent systems. By combining local resources, scenario-driven development, and recursive improvement patterns, the system can evolve and expand its capabilities autonomously while maintaining business value and operational stability.

The architecture prioritizes:
1. **Composability** over monolithic solutions
2. **Local control** over cloud dependencies  
3. **Business value** over technical elegance
4. **Continuous improvement** over static functionality
5. **Collective intelligence** over isolated components

This design enables Vrooli to serve as both a practical automation platform and a research vehicle for artificial general intelligence development.