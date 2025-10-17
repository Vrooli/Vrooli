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

## Workflow Metadata

Every YAML workflow begins with a `workflow:` block that describes the automation:

```yaml
workflow:
  name: "navigation-fundamentals"
  description: "Essential navigation patterns"
  version: "1.0.0"
  tags: [navigation, screenshots]
  steps:
    - name: "open_homepage"
      action: "navigate"
      url: "https://example.com"
```

- `name` is required and becomes the slug used in debug directories and metadata.
- `description`, `version`, and `tags` are optional but strongly encouraged for discoverability.
- When the interpreter runs, it writes a `metadata.json` file next to the debug artifacts so other tools can display the friendly name/description without parsing the YAML again.

Quick discovery commands:

```bash
# Show metadata for a single example
resource-browserless workflow describe examples/core/01-navigation.yaml

# Enumerate all examples with friendly names
resource-browserless workflow catalog examples --json
```
