# Ecosystem Manager

**Unified system for generating and improving resources and scenarios**

The Ecosystem Manager is the unified platform for generating and improving both resources and scenarios across the Vrooli ecosystem, featuring a Trello-like interface for comprehensive visibility and management.

## 🎯 **Key Benefits**

**Ecosystem Manager provides**:
- **Unified visibility** - See all ecosystem work in one dashboard
- **Intelligent prioritization** - Cross-type impact analysis
- **Simplified operations** - One service instead of 4
- **Enhanced features** - Smart suggestions and dependency chains

## 🏗️ **Architecture Overview**

### **Unified Operations**
- **resource-generator**: Create new resources with v2.0 compliance
- **resource-improver**: Improve existing resources for reliability
- **scenario-generator**: Create new scenarios with comprehensive PRDs
- **scenario-improver**: Improve existing scenarios for PRD completion

### **Smart Prompt Selection**
```yaml
# Unified configuration with operation-specific sections
operations:
  resource-generator:
    sections: [core, methodologies, generator-specific, resource-specific]
  resource-improver:  
    sections: [core, methodologies, improver-specific, resource-specific]
  scenario-generator:
    sections: [core, methodologies, generator-specific, scenario-specific]
  scenario-improver:
    sections: [core, methodologies, improver-specific, scenario-specific]
```

### **Unified Queue Schema**
```yaml
id: "resource-generator-matrix-20250907"
title: "Generate Matrix Synapse resource"
type: "resource"          # resource | scenario
operation: "generator"    # generator | improver  
category: "communication" # Smart categorization
priority: "high"
effort_estimate: "4h"
requirements: {}          # Operation-specific config
status: "pending"         # pending | in-progress | completed | failed
```

## 🚀 **Getting Started**

### **Setup**
```bash
# Run setup (builds API, installs dependencies)
vrooli scenario ecosystem-manager setup

# Start the unified service
vrooli scenario ecosystem-manager develop

# Install CLI (optional)
cd /home/matthalloran8/Vrooli/scenarios/ecosystem-manager/cli && ./install.sh
```

### **Access Points**
- **Dashboard**: http://localhost:30500  
- **API**: http://localhost:30500/api
- **Health**: http://localhost:30500/health

## 📊 **Trello-like Interface**

### **Kanban Columns**
- **Pending** ⏳ - Tasks waiting to start
- **In Progress** 🔄 - Currently being worked on  
- **Review** 👁️ - Ready for validation
- **Completed** ✅ - Successfully finished
- **Failed** ❌ - Needs attention

### **Smart Filters**
- **Type**: Resource vs Scenario
- **Operation**: Generator vs Improver
- **Category**: AI/ML, Communication, Data, Security, etc.
- **Priority**: Critical, High, Medium, Low
- **Search**: Title, ID, category text search

### **Task Cards**
```
🔧➕ Matrix Synapse                    ← Type + Operation icons
Resource Generator                      ← Operation description  
#communication #federated-chat          ← Smart tags
────────────────────
Est: 4h | Priority: High              ← Effort + Priority
▓▓▓░░░░░░░ 30%                        ← Progress bar
Currently: Research                     ← Current phase
```

## 🔌 **API Endpoints**

### **Task Management**
```bash
# List tasks with filters
GET /api/tasks?status=pending&type=resource&category=ai-ml

# Create new task  
POST /api/tasks
{
  "title": "Generate Matrix Synapse resource",
  "type": "resource",
  "operation": "generator", 
  "category": "communication",
  "priority": "high"
}

# Get task details
GET /api/tasks/{id}

# Update task status
PUT /api/tasks/{id}/status
{
  "status": "in-progress",
  "current_phase": "implementation"
}

# Get task's prompt configuration
GET /api/tasks/{id}/prompt
```

### **Configuration**
```bash
# List available operation types
GET /api/operations
```

## 🖥️ **CLI Interface**

### **Natural Commands**
```bash
# Create tasks
ecosystem-manager add resource matrix-synapse --category communication
ecosystem-manager improve scenario system-monitor --priority high

# Monitor progress  
ecosystem-manager list --status pending --type resource
ecosystem-manager show <task-id>
ecosystem-manager queue

# Update tasks
ecosystem-manager status <task-id> --progress 75 --phase testing
```

### **Smart Shortcuts**
```bash
# Generators (create new)
ecosystem-manager add resource vault --category security
ecosystem-manager add scenario chat-bot --category ai-tools

# Improvers (enhance existing)  
ecosystem-manager improve resource ollama --priority high
ecosystem-manager improve scenario invoice-generator
```

## 🔄 **Testing & Validation**

### **Running Tests**
```bash
# Run comprehensive test suite
vrooli scenario ecosystem-manager test

# Stop the service when done
vrooli scenario ecosystem-manager stop

# Create test tasks of each type
ecosystem-manager add resource test-resource --category ai-ml  
ecosystem-manager improve resource ollama
ecosystem-manager add scenario test-scenario --category productivity
ecosystem-manager improve scenario system-monitor

# Monitor operations
ecosystem-manager queue
ecosystem-manager list --status in-progress
tail -f logs/api.log
```

## 📈 **Enhanced Features**

### **Cross-Type Intelligence**
- **Impact Analysis**: "Matrix resource would enable 3 communication scenarios"
- **Smart Suggestions**: "Upgrading Ollama affects 5 AI scenarios"
- **Dependency Chains**: "Create Vault → Secure scenarios → Generate security toolkit"

### **Better Resource Management**
- **Unified Port Allocation**: No conflicts between different operations
- **Shared Memory**: All operations benefit from collective Qdrant knowledge
- **Coordinated Selection**: Prevent multiple operations on same target

### **Advanced Monitoring**
- **Cross-scenario metrics**: Success rates by operation type
- **Resource utilization**: Which resources are most/least used
- **Completion trends**: Time to completion by category
- **Queue analytics**: Bottlenecks and optimization opportunities

## 🔧 **Development**

### **Adding New Operation Types**
```yaml
# In prompts/sections.yaml
operations:
  new-operation-type:
    name: "new-operation"
    type: "generator"  # or "improver"
    target: "resources"  # or "scenarios"
    additional_sections:
      - "specific/new-operation-sections"
    variables:
      focus: "specific focus area"
```

### **Extending Categories**
```javascript
// In ui/app.js
categoryOptions: {
  resource: [
    { value: 'new-category', label: 'New Category' }
  ]
}
```

### **Custom Validation**
```go
// In api/main.go - add custom validation logic
func validateTaskRequest(task TaskItem) error {
    // Custom validation logic
}
```

## 🎉 **Success Metrics**

### **Consolidation Benefits**
- ✅ **Reduced Complexity**: 1 codebase instead of 4
- ✅ **Better Visibility**: Single dashboard for all ecosystem work  
- ✅ **Enhanced Features**: Cross-type analysis and smart prioritization
- ✅ **Easier Maintenance**: Single deployment, monitoring, testing
- ✅ **Improved UX**: Natural workflow matching user mental models

### **Operational Improvements**
- **Time to value**: Faster task creation and monitoring
- **Resource efficiency**: No duplicate infrastructure
- **Knowledge sharing**: Unified memory and learning
- **Error reduction**: Consistent patterns and validation

---

**Ecosystem Manager transforms 4 overlapping tools into 1 powerful ecosystem management platform.** 🚀
