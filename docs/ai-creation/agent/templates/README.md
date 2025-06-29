# Agent Templates

This directory contains template patterns for common agent types in the Vrooli swarm ecosystem.

## Template Categories

### 🎯 Coordinator Templates
Agents that orchestrate workflows and manage other agents:
- **Workflow Coordinator**: Manages multi-step processes
- **Resource Manager**: Allocates computational resources
- **Task Distributor**: Distributes work across specialist agents

### 🔬 Specialist Templates  
Agents with deep domain expertise:
- **Data Analyst**: Specialized data processing and analysis
- **Content Creator**: Generates domain-specific content
- **Domain Expert**: Deep knowledge in specific fields

### 📊 Monitor Templates
Agents that watch systems and trigger alerts:
- **System Monitor**: Infrastructure and performance monitoring
- **Quality Assurance**: Output validation and quality control
- **Security Watcher**: Security threat detection and response

### 🌉 Bridge Templates
Agents that connect different systems or handle integration:
- **API Bridge**: Connects external services and systems
- **Format Translator**: Handles data transformation between systems
- **Human Interface**: Facilitates human-AI collaboration

## Using Templates

1. **Choose the Right Template**: Select based on the agent's primary role
2. **Customize for Domain**: Adapt subscriptions, behaviors, and resources
3. **Validate References**: Ensure all routine labels exist in your system
4. **Test Integration**: Verify the agent works within your swarm architecture

## Template Structure

Each template includes:
- **Complete AgentSpec**: Ready-to-use JSON structure
- **Common Behaviors**: Typical event-action patterns
- **Resource Suggestions**: Commonly needed resources
- **Customization Notes**: Guidance for adaptation

## Best Practices

- Start with a template and customize rather than building from scratch
- Maintain consistency in naming conventions across similar agent types
- Include appropriate QoS levels for reliability requirements
- Design behaviors to be composable and reusable