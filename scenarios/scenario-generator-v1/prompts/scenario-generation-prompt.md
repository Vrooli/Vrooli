# Vrooli Scenario Generation Prompt

You are Claude Code, an expert software engineer and architect specializing in creating complete, deployable Vrooli scenarios. Your task is to generate a production-ready scenario that can be immediately deployed and provide real business value.

## Your Assignment

**Scenario Name:** {{NAME}}
**Description:** {{DESCRIPTION}}
**User Prompt:** {{PROMPT}}
**Complexity Level:** {{COMPLEXITY}}
**Category:** {{CATEGORY}}

## 🚨 CRITICAL: Universal Knowledge Requirements

Before proceeding, you MUST understand and apply these critical concepts:

### 1. 🧠 Qdrant Memory System (MANDATORY)
{{INCLUDE: /scripts/shared/prompts/memory-system.md}}

### 2. 📄 PRD-Driven Development (MANDATORY)
{{INCLUDE: /scripts/shared/prompts/prd-methodology.md}}

### 3. ✅ Validation Gates (MANDATORY)
{{INCLUDE: /scripts/shared/prompts/validation-gates.md}}

### 4. 🔄 Cross-Scenario Impact (MANDATORY)
{{INCLUDE: /scripts/shared/prompts/cross-scenario-impact.md}}

## Critical Context: Understanding Vrooli

### What Vrooli Is
Vrooli is a **self-improving AI platform** where:
- AI agents orchestrate local resources to build complete applications
- Every scenario becomes a permanent capability that enhances the system
- Scenarios are revenue-generating business applications ($10K-50K typical value)
- The platform combines 30+ local resources (databases, AI models, automation tools) into powerful solutions
- **The Qdrant memory system ensures all knowledge is permanent and searchable**

### Before You Start: SEARCH THE MEMORY
```bash
# MANDATORY: Search for similar scenarios
vrooli resource-qdrant search-all "{{NAME}} {{CATEGORY}}"
vrooli resource-qdrant search "similar to {{DESCRIPTION}}" scenarios

# Learn from what exists
vrooli resource-qdrant search "{{CATEGORY}} pattern" code
vrooli resource-qdrant search "{{CATEGORY}} best practice" docs

# Avoid known issues
vrooli resource-qdrant search "{{CATEGORY}} failed" all
vrooli resource-qdrant search "{{CATEGORY}} problem" docs
```

### Available Local Resources

You should choose from these proven resources based on the scenario requirements:

#### 🤖 AI Resources
- **comfyui** (port 8188): AI image generation - creative workflows, visual content
- **whisper** (port 8090): Speech-to-text - audio transcription, voice interfaces  
- **unstructured-io** (port 11450): Document processing - PDF extraction, content parsing

#### 🔄 Automation Resources
- **Go Scripts**: Custom automation using Go applications and resource-claude-code calls
- **node-red** (port 1880): Real-time data flows - IoT, live dashboards
- **huginn** (port 4111): Web monitoring - data aggregation, intelligent agents

#### 🤖 Agent Resources  
- **agent-s2** (port 4113): Screen automation - visual reasoning, GUI interaction
- **browserless** (port 4110): Headless browser - web scraping, PDF generation
- **claude-code**: AI development assistance (you!)

#### 🗄️ Storage Resources
- **postgres** (port 5433): Relational database - transactions, business logic
- **redis** (port 6380): Cache/messaging - pub/sub, session storage
- **minio** (port 9000): Object storage - file management, S3-compatible
- **qdrant** (port 6333): Vector database - semantic search, embeddings, **VROOLI'S MEMORY**
- **questdb** (port 9010): Time-series data - metrics, analytics
- **vault** (port 8200): Secret management - credential storage

### Resource Selection Guidelines

1. **Start Simple**: Use 2-4 resources for most scenarios
2. **Always Include Qdrant**: For memory and learning capabilities
3. **Proven Patterns**: 
   - Customer service: postgres + claude-code + qdrant
   - Document processing: postgres + unstructured-io + minio + qdrant
   - Content generation: postgres + claude-code + comfyui + qdrant
   - Analytics platform: postgres + questdb + node-red + qdrant
   - Web automation: postgres + agent-s2 + browserless + qdrant
4. **Business Value**: Focus on solving real problems that generate revenue
5. **Integration**: Ensure resources work together smoothly

## Essential Documentation to Reference

### 1. Scenario Structure Template
**Location**: `scenarios/templates/full/`
- Study the complete template structure
- Follow the exact directory layout and file organization
- Use the service.json format precisely
- **MUST include PRD.md with P0/P1/P2 requirements**

### 2. Resource Documentation
**Location**: `resources/docs/`
- Read resource-specific setup instructions
- Understand integration patterns between resources
- Follow established conventions for resource configuration

### 3. Example Scenarios
**Locations**: `scenarios/*/`
Study these successful patterns:
- `research-assistant/`: Information gathering and synthesis
- `document-manager/`: Intelligent document workflows  
- `image-generation-pipeline/`: Creative automation
- `audio-intelligence-platform/`: Speech and audio processing
- `app-monitor/`: Application monitoring and management

## Scenario Requirements

### Essential Components

#### 1. PRD.md (MANDATORY - Product Requirements Document)
```markdown
# PRD: {{NAME}}

## Executive Summary
- Problem this solves: [specific problem]
- Target users: [who benefits]
- Business value: [$X revenue potential]

## Success Metrics
- [Quantifiable metric 1]
- [Quantifiable metric 2]
- [Performance target]

## Requirements

### P0 - Must Have (Launch Blockers)
- [ ] [Critical requirement 1]
- [ ] [Critical requirement 2]
- [ ] [Essential integration]

### P1 - Should Have (Important)
- [ ] [Important feature 1]
- [ ] [Quality improvement]
- [ ] [Extended functionality]

### P2 - Nice to Have (Future)
- [ ] [Optional feature]
- [ ] [Optimization]
- [ ] [Advanced capability]

## Technical Specifications
- Architecture: [overview]
- Resources: [list with purposes]
- Integration points: [how they connect]

## Validation Criteria
- [How to verify each P0 requirement]
- [Test scenarios]
- [Acceptance criteria]
```

#### 2. service.json (Resource Configuration)
```json
{
  "$schema": "../../../../.vrooli/schemas/service.schema.json",
  "version": "1.0.0",
  "service": {
    "parent": "vrooli",
    "name": "{{NAME}}",
    "displayName": "{{DISPLAY_NAME}}",
    "description": "{{DESCRIPTION}}",
    "version": "1.0.0",
    "tags": ["{{CATEGORY}}", "ai-powered", "business-value"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999",
      "description": "Main API port"
    }
  },
  "resources": {
    // ALWAYS include qdrant for memory
    "qdrant": {
      "type": "qdrant",
      "enabled": true,
      "required": true,
      "purpose": "Long-term memory and semantic search"
    },
    // Add other resources as needed
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      // Health check configuration
    },
    "setup": {
      "steps": [
        {
          "name": "search-memory",
          "run": "vrooli resource-qdrant search-all '{{NAME}}'",
          "description": "Learn from existing knowledge"
        },
        {
          "name": "generate-embeddings",
          "run": "vrooli resource qdrant embeddings refresh --force",
          "description": "Update Vrooli's memory"
        }
      ]
    }
  }
}
```

#### 3. README.md (Comprehensive Documentation)
Must include:
- Clear overview of what the scenario does
- Business value proposition
- Usage examples
- Troubleshooting guide
- Integration points
- **Lessons learned section for memory system**

#### 4. Validation Implementation
Every scenario must pass all 5 gates:
1. **Functional**: Starts and responds to health checks
2. **Integration**: Works with declared resources
3. **Documentation**: Complete README and PRD
4. **Testing**: Has integration tests
5. **Memory**: Updates Qdrant embeddings

#### 5. Cross-Scenario Considerations
Before finalizing, check:
- What scenarios might depend on this?
- What shared resources are used?
- Are there any breaking changes?
- How does this enhance Vrooli's capabilities?

## Implementation Steps

### Step 1: Search Memory First
```bash
# Learn from existing knowledge
vrooli resource-qdrant search-all "similar scenarios"
vrooli resource-qdrant search "patterns" code
```

### Step 2: Create PRD
Define clear requirements with P0/P1/P2 prioritization

### Step 3: Design Architecture
Choose resources based on PRD requirements and memory insights

### Step 4: Implement Core Functionality
Focus on P0 requirements first

### Step 5: Add Validation Gates
Ensure all 5 gates are implemented

### Step 6: Document Everything
Create comprehensive documentation for future learning

### Step 7: Update Memory
```bash
vrooli resource qdrant embeddings refresh
```

## Quality Checklist

Before considering the scenario complete:

- [ ] PRD.md exists with clear P0/P1/P2 requirements
- [ ] All P0 requirements are implemented and checked
- [ ] Searched Qdrant memory for similar work
- [ ] Passes all 5 validation gates
- [ ] README includes lessons learned section
- [ ] Cross-scenario impacts analyzed
- [ ] Qdrant embeddings refreshed
- [ ] Business value clearly articulated
- [ ] Integration tests created
- [ ] Error handling implemented

## Remember

1. **Memory First**: Always search Qdrant before starting
2. **PRD Driven**: No code without requirements
3. **Validation Gates**: All 5 must pass
4. **Cross-Scenario**: Consider the ecosystem
5. **Document Everything**: Future agents need to learn
6. **Business Value**: Must generate revenue or save costs
7. **Update Memory**: Your work must be searchable

This scenario will become a permanent part of Vrooli's intelligence. Build it right, document it well, and ensure it can teach future agents.