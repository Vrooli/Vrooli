/**
 * Getting Started guide content for Browser Automation Studio
 */

export const GETTING_STARTED_CONTENT = `
# Getting Started with Browser Automation Studio

Browser Automation Studio (BAS) helps you create visual workflows to automate browser tasks, test UIs, and extract data from websites—all without writing low-level browser automation code.

## Core Concepts

### Workflows
A **workflow** is a sequence of connected nodes that define an automation. Each node represents an action (click, type, navigate) or control flow (conditional, loop). Workflows execute from top to bottom, following the connections between nodes.

### Nodes
**Nodes** are the building blocks of workflows. BAS provides 30+ node types organized into categories:

- **Navigation & Context** - Open URLs, scroll, switch tabs/frames
- **Pointer & Gestures** - Click, hover, drag, focus elements
- **Forms & Input** - Type text, select options, upload files
- **Data & Variables** - Extract data, store/use variables, run scripts
- **Assertions & Observability** - Validate conditions, wait for elements, capture screenshots
- **Workflow Logic** - Conditionals, loops, subflows
- **Storage & Network** - Manage cookies, storage, mock network requests

### Projects
**Projects** help you organize related workflows. Each project has a folder on disk where workflows are saved as JSON files.

## Creating Your First Workflow

### 1. Create a Project
Click **"New Project"** from the dashboard. Give it a name and optionally a description. Projects are saved to a folder on your machine.

### 2. Create a Workflow
Inside your project, click **"New Workflow"**. You have two options:
- **AI Assistant** - Describe what you want to automate in plain English
- **Manual Builder** - Drag and drop nodes from the palette

### 3. Add Nodes
Drag nodes from the **Node Palette** on the left onto the canvas. Connect them by dragging from one node's output handle to another's input handle.

### 4. Configure Nodes
Click on a node to open its configuration panel. Each node has different options:
- **Navigate** - Enter a URL or select a scenario
- **Click** - Provide a CSS selector or use the visual picker
- **Type** - Enter text and optionally specify a selector

### 5. Execute
Click the **"Execute"** button in the header. BAS will run your workflow in a real browser and stream screenshots back in real-time.

### 6. Review Results
After execution, view the results in the **Executions** tab. You can:
- Replay the execution step-by-step
- Export screenshots and reports
- Debug failed steps

## Working with Selectors

Selectors identify which element to interact with. BAS supports several approaches:

### CSS Selectors
Standard CSS selectors like:
- \`#submit-button\` - By ID
- \`.btn-primary\` - By class
- \`button[type="submit"]\` - By attribute
- \`[data-testid="login-form"]\` - By test ID (recommended)

### Visual Picker
Many nodes include a **screenshot picker** that lets you point-and-click on an element to automatically generate a selector.

### AI Suggestions
Describe the element you want ("the blue submit button") and BAS will suggest selectors.

## Best Practices

### Use Test IDs
For reliable automation, prefer \`data-testid\` attributes over brittle selectors:
\`\`\`html
<!-- Good -->
<button data-testid="submit-form">Submit</button>

<!-- Brittle -->
<button class="btn btn-primary mt-4">Submit</button>
\`\`\`

### Add Waits After Navigation
After navigating to a new page, add a **Wait** node to ensure the page is fully loaded before interacting with elements.

### Use Variables
Store extracted values in **variables** to reuse them later in the workflow. This is useful for:
- Capturing dynamic IDs
- Passing data between nodes
- Building assertions

### Test Incrementally
Build workflows step-by-step, executing after each addition to verify it works. This makes debugging much easier.

## Keyboard Shortcuts

BAS supports extensive keyboard shortcuts for power users:

| Shortcut | Action |
| --- | --- |
| \`Cmd/Ctrl + K\` | Global search |
| \`Cmd/Ctrl + S\` | Save workflow |
| \`Cmd/Ctrl + Enter\` | Execute workflow |
| \`Cmd/Ctrl + Z\` | Undo |
| \`Cmd/Ctrl + Shift + Z\` | Redo |
| \`Delete/Backspace\` | Delete selected nodes |
| \`Cmd/Ctrl + A\` | Select all nodes |
| \`Cmd/Ctrl + D\` | Duplicate selected |

Press \`Cmd/Ctrl + ?\` to see all available shortcuts.

## Next Steps

- **Explore the Node Reference** to learn about all available node types
- **Review the Schema Reference** for programmatic workflow creation
- **Try the Demo Workflow** to see BAS in action
`;

export const WORKFLOW_SCHEMA_INTRO = `
# Workflow Schema Reference

Browser Automation Studio workflows are stored as JSON files following a specific schema. Understanding this schema is essential for:

- Creating workflows programmatically
- Building e2e test suites that generate BAS workflows
- Integrating BAS with CI/CD pipelines
- Debugging workflow JSON files

## Schema Overview

Every workflow JSON file has this structure:

\`\`\`json
{
  "metadata": {
    "description": "Optional workflow description",
    "requirement": "Optional requirement ID",
    "version": 1
  },
  "settings": {
    "executionViewport": {
      "width": 1280,
      "height": 720,
      "preset": "desktop"
    },
    "entrySelector": "optional-entry-selector",
    "entrySelectorTimeoutMs": 30000
  },
  "nodes": [...],
  "edges": [...]
}
\`\`\`

## Nodes

Each node in the \`nodes\` array represents an action or control flow step:

\`\`\`json
{
  "id": "unique-node-id",
  "type": "click",
  "position": { "x": 100, "y": 200 },
  "data": {
    "selector": "[data-testid=submit]",
    "button": "left",
    "timeoutMs": 30000
  }
}
\`\`\`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| \`id\` | string | Yes | Unique identifier for this node |
| \`type\` | string | Yes | Node type (navigate, click, type, etc.) |
| \`position\` | object | No | Canvas position {x, y}. Omit for auto-layout |
| \`data\` | object | Yes | Node-specific configuration |

## Edges

Edges define connections between nodes:

\`\`\`json
{
  "id": "edge-1",
  "source": "node-1",
  "target": "node-2",
  "sourceHandle": "output",
  "targetHandle": "input"
}
\`\`\`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| \`id\` | string | Yes | Unique edge identifier |
| \`source\` | string | Yes | ID of the source node |
| \`target\` | string | Yes | ID of the target node |
| \`sourceHandle\` | string | No | Output handle name |
| \`targetHandle\` | string | No | Input handle name |

## Node Data Reference

Each node type has specific fields in its \`data\` object. See the **Node Reference** for complete documentation of each node type's configuration options.

## Example: Login Flow

\`\`\`json
{
  "metadata": {
    "description": "Automated login test"
  },
  "settings": {
    "executionViewport": { "width": 1280, "height": 720, "preset": "desktop" }
  },
  "nodes": [
    {
      "id": "nav-1",
      "type": "navigate",
      "data": {
        "destinationType": "url",
        "url": "https://app.example.com/login",
        "waitUntil": "networkidle"
      }
    },
    {
      "id": "type-email",
      "type": "type",
      "data": {
        "selector": "[data-testid=email-input]",
        "text": "test@example.com"
      }
    },
    {
      "id": "type-password",
      "type": "type",
      "data": {
        "selector": "[data-testid=password-input]",
        "text": "secret123"
      }
    },
    {
      "id": "click-submit",
      "type": "click",
      "data": {
        "selector": "[data-testid=login-button]",
        "waitForSelector": "[data-testid=dashboard]"
      }
    },
    {
      "id": "assert-success",
      "type": "assert",
      "data": {
        "assertionType": "element-visible",
        "selector": "[data-testid=welcome-message]"
      }
    }
  ],
  "edges": [
    { "id": "e1", "source": "nav-1", "target": "type-email" },
    { "id": "e2", "source": "type-email", "target": "type-password" },
    { "id": "e3", "source": "type-password", "target": "click-submit" },
    { "id": "e4", "source": "click-submit", "target": "assert-success" }
  ]
}
\`\`\`

## Validation

BAS validates workflows against a JSON Schema on save and before execution. Common validation errors include:

- Missing required fields (\`nodes\`, \`edges\`)
- Invalid node types
- Missing node \`id\` or \`type\`
- Invalid edge references (source/target IDs that don't exist)

## Programmatic Creation

You can create workflows programmatically and save them to a project folder:

\`\`\`javascript
const workflow = {
  metadata: { description: "Generated test" },
  settings: { executionViewport: { width: 1280, height: 720, preset: "desktop" } },
  nodes: generateTestNodes(testCases),
  edges: generateEdges(nodes),
};

// Save to project folder
fs.writeFileSync(
  path.join(projectPath, "workflows", "generated-test.json"),
  JSON.stringify(workflow, null, 2)
);
\`\`\`

Then use the BAS API or UI to execute the workflow.
`;
