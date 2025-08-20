# Vrooli Scenario Templates

> **Clean, organized template structure for all scenario creation needs**

## 📁 **Template Directory Structure**

```
templates/
├── full/          # Comprehensive template for deployable applications  
└── basic/         # Minimal template for simple integration testing
```

## 🚀 **Which Template Should I Use?**

### **Use `templates/full/` when:**
- ✅ Creating scenarios that will become customer applications
- ✅ Need full deployment orchestration (service.json, deployment/, initialization/)
- ✅ Building business applications with revenue models
- ✅ AI agents generating complete scenarios
- ✅ Want scenario-to-app.sh deployment capability

### **Use `templates/basic/` when:**
- ✅ Simple integration testing (just test resource connectivity)
- ✅ Internal development/validation scenarios
- ✅ No deployment or business requirements
- ✅ Quick resource integration validation

## 📋 **Quick Usage**

### **Create Full Application Scenario:**
```bash
# Copy comprehensive template
cp -r templates/full/ core/my-business-app/
cd core/my-business-app/

# Edit for your use case
# - service.json: Business model, resource requirements, and deployment orchestration
# - initialization/: Database, workflows, UI, configuration
# - deployment/: Startup, validation, monitoring scripts

# Run as live application
../../tools/scenario-to-app.sh my-business-app
```

### **Create Simple Integration Test:**
```bash
# Copy minimal template
cp -r templates/basic/ core/my-integration-test/
cd core/my-integration-test/

# Edit for your use case  
# - service.json: Resource requirements and test criteria
# - test.sh: Integration test logic

# Run integration test
./test.sh
```

## 🔄 **Template Consolidation Summary**

**What Changed:**
- ❌ **Before**: Scattered templates (`SCENARIO_TEMPLATE/` at root + `templates/ai-generation/`)
- ✅ **After**: Organized in single `templates/` directory
- ✅ **Cleaned**: Removed deprecated and empty template directories
- ✅ **Updated**: All documentation and scripts point to new locations

**Benefits:**
- 🎯 **Clear organization**: All templates in one logical location
- 🚀 **Purpose-driven**: Each template has a clear, distinct use case
- 🤖 **AI-friendly**: Full template includes AI generation patterns
- 🔧 **Deployment ready**: Full template works with scenario-to-app.sh
- 📚 **Easy discovery**: Simple structure for developers to navigate

## 📋 **PRD Integration (Product Requirements Document)**

The PRD serves as the **central source of truth** for each scenario, preventing drift and ensuring consistency:

### **Hub-and-Spokes Documentation Model:**
- **Hub (PRD.md)**: Defines requirements, success metrics, and capability evolution
- **Spokes**: Technical docs branch from PRD requirements
  - `README.md` - User-facing overview derived from PRD
  - `IMPLEMENTATION_PLAN.md` - Technical details implementing PRD specs
  - `api/`, `cli/`, `docs/` - All align with PRD contracts

### **Why PRDs Prevent Drift:**
- 🎯 **Single Source of Truth**: All decisions trace back to PRD
- 🔄 **Recursive Improvement**: PRDs document how capabilities compound
- 📊 **Measurable Success**: Clear metrics prevent scope creep
- 🤖 **AI Consistency**: Ensures agents maintain alignment across iterations

## 🏗️ **Template Details**

### **`templates/full/` Contents:**
- `PRD.md` - Product Requirements Document for preventing drift and maintaining consistency
- `service.json` - Complete configuration with AI patterns, business models, and deployment orchestration
- `deployment/` - startup.sh, monitor.sh
- `initialization/` - database/, workflows/, ui/, configuration/
- `test.sh` - Integration testing (optional)

### **`templates/basic/` Contents:**
- `service.json` - Simple resource requirements and test configuration  
- `test.sh` - Integration testing (optional)

**📝 Documentation Policy**: README.md files are now **optional** and only needed for scenarios with complex setup requirements. All essential information (business model, resources, deployment) is stored in service.json.

---

**Choose the template that matches your scenario's purpose and complexity!**