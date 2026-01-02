/**
 * Workflow Creation Methods documentation content
 *
 * Covers the three ways to create workflows:
 * 1. Recording - Click through a browser yourself (easiest, recommended)
 * 2. AI-Assisted - Describe automation in natural language (best for e2e tests)
 * 3. Visual Builder - Drag-and-drop for full manual control (advanced)
 */

export const WORKFLOW_METHODS_CONTENT = `
# Creating Workflows

Vrooli Ascension offers three ways to create browser automation workflows, each suited to different needs and skill levels. Choose the method that best fits your use case.

---

## Recording (Recommended)

**Best for:** New users, simple automations, capturing exact user interactions

Recording is the easiest and most intuitive way to create workflows. You interact with a real browser while Vrooli captures every action - clicks, typing, navigation, and more.

### How It Works

1. **Start Recording** - Click "New Workflow" and select "Record"
2. **Browse Naturally** - Use the browser as you normally would. Click buttons, fill forms, navigate pages
3. **Watch Actions Appear** - Each action shows up in the timeline as you perform it
4. **Review & Edit** - Adjust selectors or remove unnecessary steps before saving
5. **Create Workflow** - Name your workflow and save it to a project

### Key Features

- **Real-time capture** - Actions appear instantly in the timeline as you perform them
- **Multi-tab support** - Record actions across multiple browser tabs
- **Selector confidence** - Visual indicators show selector reliability
- **Inline editing** - Edit selectors and parameters without leaving recording mode
- **Preview testing** - Test selected steps before saving the workflow

### Tips for Better Recordings

- **Use deliberate actions** - Avoid accidental clicks that you'll need to remove later
- **Wait for pages to load** - Give pages time to fully render before interacting
- **Click precisely** - Click directly on the element you want, not near it
- **Use test IDs when available** - Elements with \`data-testid\` attributes produce more reliable selectors

### When to Use Recording

- Creating your first workflow
- Capturing a specific user journey exactly as a user would experience it
- Automations where the exact sequence of clicks matters
- Quick prototyping before refining with the visual builder

---

## AI-Assisted

**Best for:** Generating e2e tests, complex multi-step workflows, users who know what they want

AI-Assisted mode lets you describe your automation in plain language. The AI navigates the browser for you, performing actions based on your instructions.

### How It Works

1. **Start AI Mode** - Click "New Workflow" and select "AI-Assisted", or use the Auto tab in the sidebar
2. **Describe Your Goal** - Type what you want to accomplish in natural language
3. **Watch AI Navigate** - The AI analyzes screenshots and performs actions automatically
4. **Guide When Needed** - Step in for human intervention if the AI requests help
5. **Review Generated Steps** - All AI actions appear in the timeline for review
6. **Save as Workflow** - Convert the AI-generated steps into a reusable workflow

### Example Prompts

- "Log in with username 'test@example.com' and password 'secret123'"
- "Search for 'laptop' and add the first result to the cart"
- "Fill out the contact form with my name, email, and a support inquiry"
- "Navigate to the pricing page and take a screenshot of the plan comparison"

### Key Features

- **Natural language input** - No need to know selectors or technical details
- **Visual reasoning** - AI uses screenshots to understand page context
- **Automatic assertions** - AI can add validation steps (ideal for e2e tests)
- **Human intervention** - AI pauses and asks for help when uncertain
- **Model selection** - Choose between different AI models based on capability and speed

### AI Settings

Access settings via the gear icon in the Auto tab:

- **Model** - Select the vision model (e.g., GPT-4V, Claude)
- **Max Steps** - Limit how many actions the AI performs per prompt
- **Timeout** - How long to wait for each step to complete

### Tips for Better AI Results

- **Be specific** - "Click the blue Submit button" works better than "submit the form"
- **Provide context** - Mention the page you're on or what should be visible
- **Break complex tasks** - Split large goals into smaller prompts
- **Use for assertions** - Say "verify the success message appears" to auto-generate test assertions

### When to Use AI-Assisted

- Generating end-to-end test workflows with automatic assertions
- Complex multi-page journeys where describing is easier than clicking
- Exploratory automation where you're unsure of exact steps
- Creating test cases from written requirements or user stories

---

## Visual Builder

**Best for:** Advanced users, conditional logic, branching workflows, full control

The Visual Builder provides a drag-and-drop canvas for building workflows node by node. It's the most powerful option but requires understanding workflow concepts.

### How It Works

1. **Start Building** - Click "New Workflow" and select "Visual Builder"
2. **Add Nodes** - Drag nodes from the palette onto the canvas
3. **Connect Nodes** - Draw connections between node handles to define flow
4. **Configure Each Node** - Click nodes to set selectors, values, conditions
5. **Add Logic** - Use conditional and loop nodes for branching behavior
6. **Test & Execute** - Run the workflow to verify it works correctly

### Node Categories

- **Navigation & Context** - Open URLs, scroll, switch tabs/frames
- **Pointer & Gestures** - Click, hover, drag, focus elements
- **Forms & Input** - Type text, select options, upload files
- **Data & Variables** - Extract data, store/use variables, run scripts
- **Assertions & Observability** - Validate conditions, wait for elements, screenshots
- **Workflow Logic** - Conditionals, loops, subflows
- **Storage & Network** - Manage cookies, localStorage, mock network requests

### Key Features

- **Full node library** - Access all 30+ node types
- **Visual connections** - See workflow structure at a glance
- **Conditional branching** - Create if/else logic with conditional nodes
- **Loops** - Repeat actions for each item in a list
- **Subflows** - Reference other workflows as reusable modules
- **Variables** - Store and pass data between nodes

### Tips for Visual Building

- **Start simple** - Add nodes one at a time and test frequently
- **Use the search** - Press \`Cmd/Ctrl + K\` to quickly find nodes
- **Read node docs** - Each node has detailed documentation in the Node Reference
- **Leverage subflows** - Extract common patterns into reusable workflows

### When to Use Visual Builder

- Workflows requiring conditional logic (if/else branching)
- Loops that iterate over data or element lists
- Complex data extraction and transformation
- Building reusable workflow libraries with subflows
- Maximum control over every aspect of execution

---

## Choosing the Right Method

| Need | Recommended Method |
| --- | --- |
| First-time user | **Recording** |
| Capture exact user journey | **Recording** |
| Generate e2e tests quickly | **AI-Assisted** |
| Don't know exact selectors | **AI-Assisted** |
| Conditional logic (if/else) | **Visual Builder** |
| Loop over multiple items | **Visual Builder** |
| Full manual control | **Visual Builder** |

### Combining Methods

You can mix methods for best results:

1. **Record then enhance** - Record a basic flow, then open in Visual Builder to add conditionals or loops
2. **AI then refine** - Let AI generate the structure, then manually edit selectors or add assertions
3. **Build then record** - Create a visual builder skeleton, then record specific interaction sequences

---

## Next Steps

- **Try Recording** - Start with a simple login or search workflow
- **Explore AI** - Describe a task and watch the AI perform it
- **Check Node Reference** - Learn about all available node types
- **View Schema Reference** - For programmatic workflow creation
`;
