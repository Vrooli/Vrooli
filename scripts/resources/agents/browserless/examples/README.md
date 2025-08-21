# Browserless Examples

## Structure

### Active Workflow Definitions
- **`../adapters/n8n/workflows/`** - Production N8n adapter workflows
  - `execute-workflow.yaml` - Execute N8n workflows with auth handling
  - `list-workflows.yaml` - List available workflows  
  - `export-workflow.yaml` - Export workflow definitions

### Learning Examples
- **`core/`** - Basic browserless operations
- **`advanced/`** - Complex automation scenarios
- **`typescript/`** - TypeScript integration examples

### Sample N8n Workflows
- **`n8n-workflows/`** - Example N8n workflow JSON files for import

### Archives
- **`archive/prototypes/`** - Early prototypes and experimental workflows

## Usage

### For Production
Use the active adapter workflows:
```bash
resource-browserless for n8n execute <workflow-id>
```

### For Learning
Study the examples to understand browserless capabilities:
```bash
# Compile and run an example
resource-browserless workflow compile examples/core/01-navigation.yaml
```

### For N8n Setup
Import the sample workflows into your N8n instance:
```bash
# Use the JSON files in n8n-workflows/ directory
```