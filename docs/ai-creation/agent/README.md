# AI Agent Creation System

This directory contains the AI-powered agent creation pipeline for Vrooli. It enables systematic generation, staging, and validation of agent definitions for the swarm execution system.

## Overview

The AI agent creation system enables **emergent swarm behaviors** through data-driven agent configurations. Key principles:

- **Emergent Capabilities**: Complex behaviors emerge from simple event-driven rules
- **Minimal Event Set**: Agents subscribe only to predefined SWARM, RUN, and SAFETY events
- **Data-Driven**: All behaviors defined through config objects, no new code required
- **Self-Improving**: Agents analyze execution patterns to continuously optimize

The system provides:
- **Ideation**: Collecting agent concepts that leverage the minimal event system
- **Generation**: Using AI to create agent configs that respond to real platform events
- **Staging**: Organizing generated agents for review and testing
- **Validation**: Ensuring agents use only valid events and routine labels

## Directory Structure

```
docs/ai-creation/agent/
├── README.md                    # This file
├── prompts/                     # Agent generation prompts
│   └── agent-generation-prompt.md # Main generation prompt template
├── backlog.md                   # Queue of agent ideas to process
├── agent-reference.json         # Complete reference for all agents (JSON format)
├── staged/                      # Generated agent definitions ready for import
│   ├── [swarm-type folders]     # Categorized agent JSON files by swarm type
│   └── main-agents/             # Primary coordinating agents
├── cache/                       # Resolution system cache
│   ├── routine-index.json       # Available routines for agent behaviors
│   ├── resource-hints.json      # Common resource patterns
│   └── swarm-patterns.json      # Agent interaction patterns
└── templates/                   # Common agent patterns
    ├── README.md                # Template usage guide
    ├── coordinator/             # Swarm coordination agents
    ├── specialist/              # Domain-specific agents
    ├── monitor/                 # Monitoring and alerting agents
    └── bridge/                  # Integration and communication agents
```

## Quick Start

### 1. Add Agent Ideas
Edit `backlog.md` to add new agent concepts:

```markdown
### Your Agent Name
- **Goal**: Primary objective and purpose
- **Role**: Specific function within the swarm
- **Subscriptions**: Topics/events the agent monitors
- **Swarm Type**: coordinator|specialist|monitor|bridge
- **Priority**: High|Medium|Low
- **Resources**: Any specific resources needed
- **Notes**: Additional context or requirements
```

### 2. Generate Agents

#### Option A: Direct Generation with Claude (Recommended)
```bash
# Generate prompt for manual use with Claude
./scripts/ai-creation/agent-generate.sh --direct

# This generates a prompt that you can copy and paste directly to Claude
```

#### Option B: Enhanced Generation with Smart Resolution
```bash
./scripts/ai-creation/agent-generate-enhanced.sh
```

This enhanced system provides:
- **Routine Discovery**: Automatically finds available routines for behaviors
- **Resource Resolution**: Suggests relevant resources based on agent role
- **Pattern Matching**: Uses templates for common agent types
- **Dependency Validation**: Ensures referenced routines exist

### 3. Import and Test
Import generated agents into your local Vrooli instance:

```bash
# Ensure local development environment is running
./scripts/main/develop.sh --target docker --detached yes

# Import agents using CLI
vrooli agent import-dir ./docs/ai-creation/agent/staged/

# Or import by swarm type
vrooli agent import-dir ./docs/ai-creation/agent/staged/coordinator/
vrooli agent import-dir ./docs/ai-creation/agent/staged/specialist/
```

### 4. Validate Agent Definitions

```bash
# Validate agent structure and behavior references
./scripts/ai-creation/validate-agent.sh staged/your-agent.json

# Check behavior-routine mappings
./scripts/ai-creation/validate-agent-behaviors.sh

# Validate swarm coordination patterns
./scripts/ai-creation/validate-swarm-patterns.sh
```

## Agent Categories

### Swarm Organization Framework

#### 🎯 **Coordinator Agents**
- **Swarm Orchestrators**: Manage multi-agent workflows and task distribution
- **Resource Managers**: Allocate computational resources and manage agent lifecycles
- **Conflict Resolvers**: Handle agent disagreements and priority conflicts
- **Performance Monitors**: Track swarm efficiency and optimize agent collaboration

#### 🔬 **Specialist Agents**
- **Domain Experts**: Deep expertise in specific areas (finance, health, technical)
- **Tool Specialists**: Masters of specific platforms, APIs, or data sources
- **Analysis Agents**: Focused on data processing, pattern recognition, research
- **Creative Agents**: Content generation, ideation, design assistance

#### 📊 **Monitor Agents**
- **System Watchers**: Monitor infrastructure, performance, and health metrics
- **Event Processors**: Handle real-time events and trigger appropriate responses
- **Quality Assurance**: Validate outputs, ensure standards, prevent errors
- **Alerting Agents**: Notify humans or other agents of important conditions

#### 🌉 **Bridge Agents**
- **Human-AI Interfaces**: Facilitate communication between humans and AI swarms
- **Platform Integrators**: Connect different systems, APIs, and data sources
- **Protocol Translators**: Convert between different data formats and communication styles
- **Legacy Adapters**: Interface with older systems and manual processes

### Domain-Specific Categories

#### 💼 **Business Intelligence Swarm**
- **Market Analysts**: Track trends, competitor analysis, opportunity identification
- **Financial Controllers**: Budget monitoring, expense analysis, revenue optimization
- **Customer Success Managers**: Relationship monitoring, satisfaction tracking, retention
- **Operations Coordinators**: Process optimization, workflow management, efficiency

#### 🏗️ **Technical Operations Swarm**
- **DevOps Engineers**: Infrastructure monitoring, deployment automation, scaling
- **Security Guardians**: Threat detection, compliance monitoring, access control
- **Data Engineers**: Pipeline management, quality assurance, transformation
- **Integration Specialists**: API management, system connectivity, data flow

#### 🎯 **Personal Productivity Swarm**
- **Task Coordinators**: Priority management, scheduling, workflow optimization
- **Learning Facilitators**: Skill development, knowledge acquisition, progress tracking
- **Health & Wellness Coaches**: Habit formation, wellness monitoring, lifestyle optimization
- **Communication Managers**: Email processing, meeting coordination, relationship management

#### 🚀 **Innovation & Research Swarm**
- **Research Coordinators**: Information gathering, synthesis, knowledge discovery
- **Ideation Facilitators**: Brainstorming, creative problem-solving, innovation workshops
- **Prototype Builders**: Rapid experimentation, concept validation, testing
- **Knowledge Curators**: Information organization, insight extraction, documentation

## Valid Platform Events

Agents can ONLY subscribe to these predefined events (no custom events allowed):

### SWARM Events
- `swarm/started` - New swarm execution begins
- `swarm/state/changed` - Swarm state transitions (RUNNING, PAUSED, FAILED, etc.)
- `swarm/resource/updated` - Resource allocation/consumption changes
- `swarm/config/updated` - Swarm configuration modified
- `swarm/team/updated` - Team composition changes
- `swarm/goal/created` - New goal added to swarm
- `swarm/goal/updated` - Goal progress or details updated
- `swarm/goal/completed` - Goal successfully achieved
- `swarm/goal/failed` - Goal could not be completed

### RUN Events  
- `run/started` - Routine execution begins
- `run/completed` - Routine finishes successfully
- `run/failed` - Routine execution fails
- `run/task/ready` - Task ready for execution
- `run/decision/requested` - User input needed
- `step/started` - Individual step begins
- `step/completed` - Step finishes (includes performance metrics)
- `step/failed` - Step execution fails

### SAFETY Events
- `safety/pre_action` - BEFORE any action (validate/sanitize inputs)
- `safety/post_action` - AFTER action completion (validate outputs)

These minimal events enable emergent behaviors through creative agent configurations.

## Agent Specification Schema

### Core Structure
```json
{
  "identity": {
    "name": "unique-agent-name",
    "id": "optional-snowflake-id",
    "version": "1.0.0"
  },
  "goal": "Clear statement of agent's primary objective",
  "role": "Specific function within the swarm ecosystem",
  "subscriptions": ["topic1", "topic2", "topic3"],
  "behaviors": [
    {
      "trigger": {
        "topic": "subscription-topic",
        "when": "optional-condition"
      },
      "action": {
        "type": "routine|invoke",
        "label": "routine-label",
        "inputMap": {"key": "value"},
        "purpose": "description-for-invoke"
      },
      "qos": 0
    }
  ],
  "prompt": {
    "mode": "supplement",
    "source": "direct",
    "content": "Natural language guidance for agent behavior"
  },
  "resources": ["resource1", "resource2"]
}
```

### Behavior Types

#### Routine Behaviors
- **Type**: `"routine"`
- **Use When**: Deterministic, predictable responses required
- **Examples**: Data processing, calculations, standard workflows
- **Configuration**: Requires `label` (routine name) and optional `inputMap`

#### Invoke Behaviors  
- **Type**: `"invoke"`
- **Use When**: Thinking, analysis, or adaptive responses needed
- **Examples**: Decision making, creative tasks, complex problem solving
- **Configuration**: Requires `purpose` description

### Quality of Service (QoS) Levels

- **QoS 0**: Fire-and-forget (best effort)
- **QoS 1**: At-least-once delivery (acknowledgment required)
- **QoS 2**: Exactly-once delivery (guaranteed, no duplicates)

### Agent Prompts

The `prompt` field provides agent-specific behavioral guidance:

- **Mode**: Always use `"supplement"` (never `"replace"`)
  - Supplement mode adds your guidance on top of core swarm functionality
  - Replace mode would remove essential system prompting about:
    - How to use swarm tools and subscribe to events
    - How to interact with the blackboard and other agents
    - Core safety and coordination protocols
- **Source**: Always use `"direct"` for simplicity
  - Direct prompts embed the guidance in the agent configuration
  - Resource prompts would require managing separate prompt objects
- **Content**: Natural language instructions for the agent
  - Be specific about behavioral constraints and expectations
  - Convert formal rules (obligations/permissions/prohibitions) to clear instructions
  - Keep prompts concise but comprehensive

Example:
```json
"prompt": {
  "mode": "supplement",
  "source": "direct",
  "content": "Monitor all database queries for performance issues. Alert when query time exceeds 5 seconds. Never expose sensitive data in logs or alerts."
}
```

## Generation Workflow

### 1. **Requirements Analysis**
- Extract agent goals and roles from backlog
- Identify required subscriptions and event patterns
- Determine appropriate behavior types (routine vs invoke)

### 2. **Routine Discovery**
- Search available routines matching agent capabilities
- Generate routine table for prompt template
- Resolve routine labels to ensure availability

### 3. **Template Selection**
- Choose appropriate agent template based on role
- Apply swarm-specific patterns and conventions
- Customize for domain requirements

### 4. **Generation Process**
- Fill prompt template with resolved data
- Generate agent specification using AI
- Validate against AgentSpec interface

### 5. **Validation & Staging**
- Structural validation of JSON schema
- Behavior-routine mapping verification
- Swarm pattern compliance checking
- Stage for import and testing

## Emergent Behavior Examples

### How Simple Events Create Complex Behaviors

#### Example 1: Self-Optimizing Execution
```
Agent subscribes to: step/completed
Behavior: Analyzes metrics in payload, calls optimization routine
Emergence: System learns optimal execution patterns without explicit programming
```

#### Example 2: Adaptive Resource Management  
```
Agent subscribes to: swarm/resource/updated, run/failed
Behavior: When resources low AND runs failing, calls resource-reallocation routine
Emergence: Dynamic resource balancing based on actual usage patterns
```

#### Example 3: Failure Pattern Recognition
```
Agent subscribes to: step/failed, safety/post_action
Behavior: Correlates failures with action types, builds prevention rules
Emergence: System becomes more reliable over time by learning from mistakes
```

#### Example 4: Goal Decomposition
```
Agent subscribes to: swarm/goal/created
Behavior: Calls task-decomposer routine to break goal into runs
Emergence: Complex goals automatically decomposed into executable steps
```

The key insight: We don't program specific behaviors - we configure agents to respond to events with routines, and complex behaviors emerge from their interactions.

## Best Practices

### Agent Design Principles

#### **Single Responsibility**
- Each agent should have one clear, focused purpose
- Avoid creating "god agents" that try to do everything
- Design for composability and collaboration

#### **Event-Driven Architecture**
- Agents react to events via subscriptions
- Use meaningful topic names that describe events clearly
- Design behaviors to be stateless when possible

#### **Graceful Degradation**
- Handle missing routines or resources gracefully
- Implement fallback behaviors for critical functions
- Use appropriate QoS levels for reliability requirements

#### **Swarm Coordination**
- Design agents to work well with others in the swarm
- Avoid resource conflicts and race conditions
- Use prompts to define behavioral guidance and constraints

### Naming Conventions

#### **Agent Names**
- Use descriptive, hyphenated names: `market-research-coordinator`
- Include role indication: `data-quality-monitor`, `content-creation-specialist`
- Keep names unique within the swarm ecosystem

#### **Topics and Subscriptions**
- Use hierarchical naming: `user.onboarding.started`, `system.performance.alert`
- Be specific about event types and contexts
- Follow consistent naming patterns across the swarm

#### **Behavior Triggers**
- Match subscription topics exactly
- Use `when` conditions for fine-grained control
- Design for clear event-action relationships

## Templates and Patterns

### Coordinator Agent Template
```json
{
  "identity": { "name": "workflow-coordinator" },
  "goal": "Orchestrate multi-step workflows across specialist agents",
  "role": "coordinator",
  "subscriptions": ["workflow.started", "task.completed", "agent.error"],
  "behaviors": [
    {
      "trigger": { "topic": "workflow.started" },
      "action": { "type": "routine", "label": "workflow-decomposer" }
    },
    {
      "trigger": { "topic": "task.completed" },
      "action": { "type": "invoke", "purpose": "Determine next workflow step" }
    }
  ]
}
```

### Specialist Agent Template
```json
{
  "identity": { "name": "data-analysis-specialist" },
  "goal": "Perform deep analysis on specific data types",
  "role": "specialist",
  "subscriptions": ["data.analysis.requested"],
  "behaviors": [
    {
      "trigger": { "topic": "data.analysis.requested" },
      "action": { "type": "routine", "label": "comprehensive-data-analyzer" }
    }
  ],
  "prompt": {
    "mode": "supplement",
    "source": "direct",
    "content": "Always validate data integrity and format before performing analysis. Reject malformed or suspicious data."
  }
}
```

### Monitor Agent Template
```json
{
  "identity": { "name": "system-health-monitor" },
  "goal": "Monitor system health and alert on issues",
  "role": "monitor",
  "subscriptions": ["system.metrics", "system.error", "system.performance"],
  "behaviors": [
    {
      "trigger": { "topic": "system.error" },
      "action": { "type": "routine", "label": "error-alert-generator" },
      "qos": 2
    },
    {
      "trigger": { "topic": "system.performance", "when": "degraded" },
      "action": { "type": "invoke", "purpose": "Analyze performance issues and recommend solutions" }
    }
  ]
}
```

## Troubleshooting

### Generation Issues
- **"No routines found"**: Ensure routine-reference.json is up to date
- **"Invalid agent specification"**: Check JSON structure against AgentSpec interface
- **"Duplicate agent name"**: Use unique names within the swarm

### Validation Issues
- **"Unknown routine label"**: Verify routine exists in staged or imported routines
- **"Invalid behavior type"**: Use only "routine" or "invoke" for action types
- **"Missing required field"**: Ensure identity.name, goal, role, and subscriptions are present

### Import Issues
- **"Agent conflict"**: Check for naming conflicts with existing agents
- **"Behavior validation failed"**: Verify all referenced routines are available
- **"Invalid subscription format"**: Use proper topic naming conventions

## Scripts and Utilities

### Validation and Testing

#### Agent Behavior Validation
```bash
# Validate all agent templates
./validate-agent-behaviors.sh templates/**/*.json

# Check specific agent template
./validate-agent-behaviors.sh templates/coordinator/workflow-coordinator.json

# Quick validation (structure only)
./validate-agent-behaviors.sh --quick templates/*.json
```

#### Routine Dependency Checking
```bash
# Update routine index from latest staged routines
./create-routine-index.sh

# Check which routines agents need but don't exist
./validate-agent-behaviors.sh templates/**/*.json | grep "NOT FOUND"

# List all routine names available for agents
jq -r '.availableNames[]' cache/routine-index.json
```

### Agent Generation

#### Generate Agent Prompts
```bash
# Generate prompt for creating new agents
./generate-agent-prompt.sh

# Generate prompt with specific agent role
./generate-agent-prompt.sh --role coordinator

# Generate prompt for specific behavioral pattern
./generate-agent-prompt.sh --pattern emergent-learning
```

### Routine Reference Management

The agent system automatically stays in sync with the routine system:

```bash
# Routine reference is automatically updated when routines are staged
cd ../routine && ./generate-routine-reference.sh

# This automatically:
# 1. Scans all staged routines (432 found)
# 2. Updates routine-reference.json
# 3. Copies to agent directory
# 4. Rebuilds agent routine index
```

#### Query Routine Dependencies
```bash
# Find routines needed by agents
grep -h '"label"' templates/**/*.json | sort -u

# Check if specific routine exists
grep "Routine Name" cache/routine-index.json

# List routines by category
jq '.byCategory' ../routine/routine-reference.json
```

### Cache Management

The `cache/` directory contains auto-generated indices:

```bash
# View routine index status
cat cache/routine-index.json | jq '.metadata'

# Clear and rebuild all caches
rm -rf cache/*.json && ./create-routine-index.sh

# View available routine names
jq -r '.availableNames[]' cache/routine-index.json | head -20
```

## Related Documentation

- [`../routine/README.md`](../routine/README.md) - Routine creation system for agent behaviors  
- [`/docs/architecture/execution/README.md`](../../architecture/execution/README.md) - Swarm execution architecture
- [`/docs/plans/agent-orchestration.md`](../../plans/agent-orchestration.md) - Agent coordination strategies
- [`IMPLEMENTATION_SUMMARY.md`](./IMPLEMENTATION_SUMMARY.md) - Current implementation status and next steps