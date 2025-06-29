# Staged Agents Organization

This directory contains AI-generated agent definitions organized by role and ready for import into the Vrooli swarm system.

## Directory Structure

```
staged/
├── coordinator/         # Orchestration and workflow management agents
├── specialist/          # Domain-specific expert agents  
├── monitor/            # System monitoring and alerting agents
├── bridge/             # Integration and communication agents
└── main-agents/        # Primary coordinating agents for major workflows
```

## Agent Categories

### 🎯 Coordinator Agents (`coordinator/`)
Agents that orchestrate workflows and manage other agents:
- **Workflow Coordinators**: Manage complex multi-step processes
- **Resource Managers**: Allocate computational resources and agent lifecycles
- **Task Distributors**: Distribute work across specialist agents
- **Conflict Resolvers**: Handle agent disagreements and priority conflicts

### 🔬 Specialist Agents (`specialist/`)
Agents with deep domain expertise:
- **Domain Experts**: Finance, health, technical, legal, creative domains
- **Analysis Specialists**: Data processing, pattern recognition, insights
- **Tool Masters**: Platform-specific expertise (APIs, databases, services)
- **Content Creators**: Writing, design, media generation

### 📊 Monitor Agents (`monitor/`)
Agents that watch systems and trigger alerts:
- **System Monitors**: Infrastructure, performance, health metrics
- **Quality Assurance**: Output validation, standards enforcement
- **Security Watchers**: Threat detection, compliance monitoring
- **Event Processors**: Real-time event handling and routing

### 🌉 Bridge Agents (`bridge/`)
Agents that connect different systems or handle integration:
- **API Bridges**: External service integration and communication
- **Format Translators**: Data transformation between systems
- **Human Interfaces**: Facilitate human-AI collaboration
- **Legacy Adapters**: Interface with older systems and manual processes

### 🚀 Main Agents (`main-agents/`)
Primary orchestrating agents for major business workflows:
- **Business Process Coordinators**: End-to-end business workflow management
- **Customer Journey Managers**: Complete customer lifecycle orchestration
- **Project Managers**: Complex project coordination across multiple domains
- **Service Orchestrators**: High-level service delivery coordination

## Agent Specification Format

Each agent file follows the AgentSpec interface:

```json
{
  "identity": {
    "name": "unique-agent-name",
    "version": "1.0.0"
  },
  "goal": "Clear statement of agent's primary objective",
  "role": "coordinator|specialist|monitor|bridge",
  "subscriptions": ["topic1", "topic2"],
  "behaviours": [
    {
      "trigger": { "topic": "event-topic" },
      "action": { "type": "routine|invoke", "label": "routine-name" },
      "qos": 0
    }
  ],
  "norms": [
    { "modality": "obligation", "target": "behavioral-constraint" }
  ],
  "resources": ["resource1", "resource2"]
}
```

## Import Considerations

### Dependency Management
- **Routine Dependencies**: Ensure all referenced routine labels exist
- **Resource Availability**: Verify required resources are accessible
- **Topic Consistency**: Use consistent topic naming across agents

### Swarm Coordination
- **Role Balance**: Maintain appropriate ratios of different agent types
- **Communication Patterns**: Ensure agents can effectively communicate
- **Conflict Avoidance**: Prevent overlapping responsibilities and resource conflicts

### Performance Optimization
- **QoS Levels**: Use appropriate quality of service for each behavior
- **Resource Allocation**: Balance resource usage across agents
- **Scaling Considerations**: Design for horizontal and vertical scaling

## Validation Checklist

Before importing agents:

- ✅ **Structure**: Valid JSON following AgentSpec interface
- ✅ **Names**: Unique agent names within the swarm
- ✅ **Routines**: All referenced routine labels exist and are accessible
- ✅ **Topics**: Consistent topic naming and subscription patterns
- ✅ **Resources**: Required resources are available or can be provisioned
- ✅ **Behaviors**: Appropriate behavior types for intended actions
- ✅ **QoS**: Suitable quality of service levels for reliability needs
- ✅ **Norms**: Clear behavioral constraints and ethical guidelines

## Swarm Patterns

### Common Coordination Patterns

#### **Hierarchical Workflow**
```
Coordinator → Specialist → Monitor
     ↓           ↓          ↓
   Bridge ← Specialist ← Monitor
```

#### **Event-Driven Processing**
```
Event → Monitor → Coordinator → Specialist → Bridge
```

#### **Collaborative Analysis**
```
Specialist ↔ Specialist ↔ Coordinator
     ↓           ↓           ↓
   Monitor ← Bridge ← Monitor
```

### Topic Flow Examples

```
user.onboarding.started → customer-success-coordinator
                       → data-collection-specialist
                       → progress-monitor

system.error.detected → error-monitor
                     → escalation-coordinator  
                     → notification-bridge

workflow.complex.initiated → task-coordinator
                         → requirement-analyzer
                         → progress-tracker
```

## Best Practices

### Agent Design
- **Single Purpose**: Each agent should have one clear responsibility
- **Composable**: Design agents to work well in combination
- **Resilient**: Include error handling and graceful degradation
- **Observable**: Enable monitoring and debugging capabilities

### Swarm Organization
- **Balanced Roles**: Include appropriate mix of coordinator, specialist, monitor, and bridge agents
- **Clear Communication**: Use descriptive topic names and consistent patterns
- **Resource Efficiency**: Avoid resource conflicts and over-allocation
- **Scalable Architecture**: Design for growth and changing requirements

### Development Workflow
1. **Design**: Define agent goals, roles, and interactions
2. **Generate**: Create agent specifications using AI assistance
3. **Validate**: Check structure, dependencies, and patterns
4. **Stage**: Organize agents by role and prepare for import
5. **Test**: Verify agent behavior in isolated environments
6. **Deploy**: Import agents into production swarm system
7. **Monitor**: Track performance and optimize as needed