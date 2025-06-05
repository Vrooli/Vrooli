# Inter-Tier Communication Architecture

This directory contains the core documentation for communication patterns and interfaces between Vrooli's three-tier execution architecture. This README serves as the navigation hub for understanding and implementing the communication system.

**Quick Start**: New to the architecture? Start with the [Visual Architecture Overview](#visual-architecture-overview) below, then follow the [Implementation Reading Order](#implementation-reading-order).

## Visual Architecture Overview

> 📖 **Architecture Context**: For complete three-tier architecture overview, see the **[Architecture Overview](../_ARCHITECTURE_OVERVIEW.md)**.

The communication architecture focuses on how the three tiers coordinate through four primary communication patterns:

```mermaid
graph TB
    subgraph "Communication Patterns"
        ToolRouting["🎯 Tool Routing Communication<br/>• T1 agents → T2 routines<br/>• CompositeToolRunner dispatch<br/>• MCP & OpenAI tools"]
        
        DirectInterface["📞 Direct Interface Communication<br/>• T2 orchestrator → T3 executor<br/>• TypeScript interfaces<br/>• High-performance calls"]
        
        EventBus["📡 Event-Driven Messaging<br/>• Cross-tier coordination<br/>• Async intelligence agents<br/>• Real-time monitoring"]
        
        SharedState["💾 Shared State Communication<br/>• ChatConfigObject persistence<br/>• Context inheritance<br/>• Distributed consistency"]
    end
    
    subgraph "Performance Targets"
        Perf["> 📊 See Performance Reference<br/>• Tool Routing: ~1-2s P95<br/>• Direct Interface: ~100-200ms P95<br/>• Event Bus: ~200-500ms P95<br/>• Shared State: ~50-100ms P95"]
    end
    
    ToolRouting -.-> Perf
    DirectInterface -.-> Perf
    EventBus -.-> Perf
    SharedState -.-> Perf
    
    classDef pattern fill:#e8f5e8,stroke:#2e7d32,stroke-width:2px
    classDef perf fill:#f3e5f5,stroke:#7b1fa2,stroke-width:1px,stroke-dasharray:3 3
    
    class ToolRouting,DirectInterface,EventBus,SharedState pattern
    class Perf perf
```

> 📊 **Performance Details**: For complete performance characteristics, monitoring thresholds, and optimization strategies, see **[Performance Reference](../_PERFORMANCE_REFERENCE.md)**.

## Implementation Reading Order

**Prerequisites**: Read [Main Execution Architecture](../README.md) for complete architectural context.

### **Phase 1: Foundation (Must Read First)**
1. **[Centralized Type System](../types/core-types.ts)** - Single source of truth for all interfaces
2. **[Communication Patterns](communication-patterns.md)** - Detailed definitions, selection framework, and coordination examples for the four primary communication patterns: MCP Tool Communication, Direct Service Interface, Event-Driven Messaging, and State Synchronization.
3. **[Tier Interface Contracts](tier-interfaces.md)** - Interface contracts between tiers

### **Phase 2: Implementation Details**
4. **[MCP Tool Integration](implementation/mcp-integration.md)** - T1→T2 MCP tool bridge and `CompositeToolRunner` details.
5. **[Integration Map and Validation](integration-map.md)** - Complete end-to-end flows, validation framework, and troubleshooting.

### **Phase 3: Cross-Cutting Architecture**
For comprehensive understanding of the full execution architecture, continue with:
- **[Event-Driven Architecture](../event-driven/README.md)** - Event bus protocol and coordination
- **[Context & Memory Management](../context-memory/README.md)** - Context architecture and layer management
- **[Resource Management](../resource-management/README.md)** - Resource allocation, coordination, and state management
- **[Security Boundaries](../security/README.md)** - Security model and permissions
- **[Error Handling & Resilience](../resilience/error-scenarios-guide.md)** - Error propagation and recovery
- **[Performance & Monitoring](../_PERFORMANCE_REFERENCE.md)** - Performance optimization and observability

## Communication Architecture Principles

The communication architecture is built upon four primary patterns, detailed in [Communication Patterns](communication-patterns.md), and guided by key design principles:

1. **🎯 Automatic Tool Routing**: T1 uses tool routing that seamlessly handles OpenAI and MCP tools.
2. **⚖️ Hierarchical Resource Management**: Clear resource ownership and budget inheritance, detailed in [Resource Management](../resource-management/README.md).
3. **🔄 Event-Driven Intelligence**: Capabilities emerge from specialized agents, not hard-coded logic, as described in [Event-Driven Architecture](../event-driven/README.md).
4. **🛡️ Security by Design**: Trust boundaries enforced at every tier transition, outlined in [Security Boundaries](../security/README.md).
5. **📊 Graceful Degradation**: System continues operation through intelligent failure recovery, managed by the [Error Handling Framework](../resilience/error-scenarios-guide.md).
6. **🔍 Observable by Default**: Comprehensive monitoring and audit trails, specified in [Performance & Monitoring](../_PERFORMANCE_REFERENCE.md).

## Core Communication Components

### **Communication Patterns**
- **[Communication Patterns](communication-patterns.md)** - Master definition of the four communication patterns
- **[Tier Interfaces](tier-interfaces.md)** - Interface contracts between tiers
- **[Integration Map](integration-map.md)** - End-to-end validation and troubleshooting

### **Implementation Details**
- **[MCP Tool Integration](implementation/mcp-integration.md)** - Model Context Protocol implementation
- **[Centralized Type System](../types/core-types.ts)** - All interface definitions

## Usage Guidelines

### **For New Implementation**
1. **Start with Prerequisites**: Read [Main Execution Architecture](../README.md) for context.
2. **Follow Reading Order**: Use the systematic reading order above.
3. **Apply Communication Patterns**: Understand the [Communication Patterns](communication-patterns.md).
4. **Validate Implementation**: Use [Integration Map](integration-map.md) for comprehensive validation.

### **For Understanding Architecture**
1. **Core Patterns**: Understand the [Communication Patterns](communication-patterns.md).
2. **Interface Contracts**: Review [Tier Interfaces](tier-interfaces.md).
3. **Cross-Cutting Concerns**: Study the related architecture domains listed in Phase 3 above.

### **For Troubleshooting**
1. **Use Integration Map**: The [Integration Map](integration-map.md) provides the primary troubleshooting framework.
2. **Apply Decision Trees**: Use systematic algorithms in [Error Handling](../resilience/error-scenarios-guide.md).
3. **Review Failure Scenarios**: Consult [Failure Scenarios](../resilience/failure-scenarios/README.md) for specific issues.

## Related Documentation

- **[Main Execution Architecture](../README.md)** - Complete three-tier execution architecture overview
- **[Individual Tier Documents](../tiers/)** - Tier-specific implementation details
- **[Event-Driven Architecture](../event-driven/README.md)** - Event bus and coordination
- **[Context & Memory Management](../context-memory/README.md)** - Context architecture and layer management
- **[Resource Management](../resource-management/README.md)** - Resource allocation, coordination, and state management
- **[Security Boundaries](../security/README.md)** - Security model and permissions
- **[Resilience & Error Handling](../resilience/error-scenarios-guide.md)** - Error propagation and recovery
- **[Performance & Monitoring](../_PERFORMANCE_REFERENCE.md)** - Performance optimization

## 📚 Deep Dive Documentation

- **[Communication Patterns](communication-patterns.md)** - Detailed patterns for each tier interaction
- **[Tier Communication Protocols](tier-communication-protocols.md)** - **🆕 Specific protocols, interfaces, and data flows between the three tiers**
- **[Integration Map](integration-map.md)** - Complete integration testing approach
- **[Implementation: MCP Integration](implementation/mcp-integration.md)** - Tool routing implementation details

This communication architecture ensures optimal operation across all tiers while providing comprehensive integration capabilities and robust coordination through systematic patterns and interfaces.