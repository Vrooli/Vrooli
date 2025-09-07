# Modify App for New Resource Prompt

You are Claude Code. Your task is to take an existing copied scenario and modify it to integrate a new resource.

## Your Assignment

**Base Scenario:** {{TARGET_SCENARIO}}
**New Resource:** {{NEW_RESOURCE}}
**Modification Goal:** {{MODIFICATION_GOAL}}

## Resource Integration Guidelines

### Storage Resources (postgres, redis, minio, qdrant, vault, questdb)

**Integration Points:**
- Add resource to `.vrooli/service.json`
- Create initialization scripts in `initialization/storage/[resource]/`
- Update workflows to use the storage
- Modify data handling logic

**Example Redis Integration:**
```json
// In service.json
"redis": {
  "type": "redis", 
  "enabled": true,
  "required": true,
  "purpose": "session storage and caching"
}
```

### AI Resources (ollama, comfyui, whisper, unstructured-io)

**Integration Points:**
- Configure AI service in `.vrooli/service.json`
- Create model/prompt configurations
- Add AI processing workflows
- Update UI for AI interactions

**Example Ollama Integration:**
```json
// In service.json
"ollama": {
  "type": "ollama",
  "enabled": true, 
  "required": true,
  "purpose": "local AI inference for content generation"
}
```

### Automation Resources (windmill, node-red, huginn)

**Integration Points:**
- Enable automation platform
- Create new workflows
- Connect to existing data sources
- Set up triggers and actions

### Agent Resources (claude-code, browserless, agent-s2)

**Integration Points:**
- Configure agent service
- Create agent interaction workflows
- Add agent capabilities to business logic

## Modification Process

### Step 1: Update Service Configuration
Modify `.vrooli/service.json` to include {{NEW_RESOURCE}}:
- Add to appropriate resource category
- Set enabled: true
- Define purpose and configuration
- Add any ports or environment variables needed

### Step 2: Create Initialization Files
Add necessary initialization files:
- **Storage**: schemas, seed data, configurations
- **AI**: model configs, prompt templates
- **Automation**: workflow definitions, trigger configs
- **Agents**: service configurations

### Step 3: Update Workflows
Modify existing automation workflows:
- Add new resource to data flows
- Create new processing steps
- Connect resource outputs to existing logic
- Add error handling for new resource

### Step 4: Enhance Business Logic
Integrate new capabilities:
- Add new features enabled by the resource
- Update APIs to expose new functionality
- Modify UI to access new capabilities
- Update documentation

### Step 5: Add Testing
Update test files:
- Add health checks for new resource
- Test new integration points
- Verify existing functionality still works

## Output Format

Present your modifications as:

### Integration Summary
- **Resource Added:** {{NEW_RESOURCE}}
- **Primary Purpose:** [what it adds to the scenario]
- **Key Benefits:** [2-3 bullet points]

### Modified Files

#### File: .vrooli/service.json
**Changes Made:**
- Added {{NEW_RESOURCE}} to resources section
- [other specific changes]

**Updated Content:**
```json
[complete modified file]
```

#### File: initialization/[category]/[file]
**Changes Made:**
- [specific changes made]

**Updated Content:**
```
[complete file content]
```

### New Files Created

#### File: path/to/new/file
**Purpose:** [what this file does]
**Content:**
```
[complete file content]
```

### Integration Benefits
1. **Enhanced Functionality:** [description]
2. **New Use Cases:** [description] 
3. **Improved Performance:** [description]

## Key Principles

1. **Seamless Integration:** New resource should feel native to the scenario
2. **Value Addition:** Clear business value from the integration
3. **Maintain Quality:** Don't break existing functionality
4. **Follow Patterns:** Use established Vrooli conventions
5. **Complete Implementation:** All necessary configs and workflows

Now, please modify the {{TARGET_SCENARIO}} scenario to integrate {{NEW_RESOURCE}} according to these guidelines.