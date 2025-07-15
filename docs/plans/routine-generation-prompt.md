# Routine Generation Prompt

## Overview

You are an expert AI system designer tasked with creating **routines** for Vrooli's three-tier execution architecture. Routines are reusable, composable workflow definitions that orchestrate multiple steps to accomplish complex tasks. They serve as the core building blocks for AI-driven automation, reasoning, and conversation systems.

## Execution Architecture Context

### Three-Tier Architecture

Vrooli uses a sophisticated three-tier execution system:

- **Tier 1 (Swarm Coordination)**: Manages strategic planning, resource allocation, and multi-agent coordination. Handles goal decomposition and high-level decision making.
- **Tier 2 (Process Intelligence)**: Executes routines through workflow navigation, step orchestration, and parallel coordination. This is where your generated routines will run.
- **Tier 3 (Tool Execution)**: Performs direct task execution including LLM calls, API interactions, code execution, and tool integrations.

### How Routines Fit Into the Architecture

Routines are **Tier 2 constructs** that:
1. **Bridge strategic goals with tactical execution** - They translate high-level objectives from Tier 1 into concrete step sequences
2. **Orchestrate multiple subroutines** - Each step in a routine calls a subroutine (smaller, focused task) or performs a specific action
3. **Enable complex workflow patterns** - Support sequential execution, parallel processing, conditional branching, and error handling
4. **Provide reusability and composability** - Routines can call other routines as subroutines, creating hierarchical workflows
5. **Support multiple execution strategies** - Can operate in conversational, reasoning, or deterministic modes

## Routine Structure and Types

### Sequential Routines (Array-Based)

Use sequential routines for:
- **Linear workflows** with clear step-by-step progression
- **Simple parallel tasks** that can be executed concurrently
- **Most common use cases** where steps have clear dependencies

**Structure:**
```typescript
{
  __type: "Sequential",
  schema: {
    steps: [
      {
        id: "step1",
        name: "Step Name",
        description: "What this step does",
        subroutineId: "subroutine-id",
        inputMap: { "stepInput": "subroutineInput" },
        outputMap: { "subroutineOutput": "stepOutput" },
        skipCondition?: "condition expression",
        retryPolicy?: { maxAttempts: 3, backoffMs: 1000 }
      }
    ],
    rootContext: {
      inputMap: { "routineInput": "firstStepInput" },
      outputMap: { "lastStepOutput": "routineOutput" }
    },
    executionMode: "sequential" | "parallel"
  }
}
```

### BPMN Routines (Business Process Model)

Use BPMN routines for:
- **Complex workflow patterns** with multiple decision points
- **Enterprise-grade processes** requiring formal process modeling
- **Workflows with advanced patterns** like event handling, boundary events, or complex gateways
- **Integration with existing BPMN systems**

**Structure:**
```typescript
{
  __type: "BPMN-2.0",
  schema: {
    __format: "xml",
    data: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>...", // BPMN XML
    activityMap: {
      "callActivityId": {
        subroutineId: "subroutine-id",
        inputMap: { "activityInput": "subroutineInput" },
        outputMap: { "subroutineOutput": "activityOutput" }
      }
    },
    rootContext: {
      inputMap: { "routineInput": "processInput" },
      outputMap: { "processOutput": "routineOutput" }
    }
  }
}
```

#### BPMN Implementation Details

**Required Namespaces:**
```xml
xmlns:bpmn="http://bpmn.io/schema/bpmn"
xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
xmlns:vrooli="[APP_URL]/bpmn"
```

**Call Activity Structure:**
Call activities are the primary mechanism for executing subroutines in BPMN routines. Each call activity must:

1. **Reference a subroutine** via `calledElement` attribute
2. **Define I/O mapping** using Vrooli extensions
3. **Be mapped in activityMap** for subroutine binding

```xml
<bpmn:callActivity id="analyzeDataActivity" name="Analyze Data" calledElement="analyzeData">
  <bpmn:extensionElements>
    <vrooli:ioMapping>
      <!-- Input mappings -->
      <vrooli:input name="dataSet" fromContext="root.inputData" />
      <vrooli:input name="analysisType" fromContext="configureAnalysis.selectedType" />
      
      <!-- Output mappings -->
      <vrooli:output name="results" />
      <vrooli:output name="summary" toRootContext="analysisResult" />
    </vrooli:ioMapping>
  </bpmn:extensionElements>
  <bpmn:incoming>Flow_1</bpmn:incoming>
  <bpmn:outgoing>Flow_2</bpmn:outgoing>
</bpmn:callActivity>
```

**Activity Map Configuration:**
The `activityMap` connects BPMN call activities to actual subroutines:

```typescript
{
  "analyzeDataActivity": {
    "subroutineId": "data-analysis-routine-v2",
    "inputMap": {
      "dataSet": "inputData",      // vrooli:input name -> subroutine input
      "analysisType": "type"       // vrooli:input name -> subroutine input
    },
    "outputMap": {
      "analysisResults": "results", // subroutine output -> vrooli:output name
      "executiveSummary": "summary" // subroutine output -> vrooli:output name
    }
  }
}
```

**Root Context Configuration:**
Maps routine-level inputs/outputs to the BPMN process:

```typescript
{
  "inputMap": {
    "inputData": "routineDataInput",    // BPMN process variable -> routine input
    "configuration": "routineConfig"     // BPMN process variable -> routine input
  },
  "outputMap": {
    "analysisResult": "routineFinalOutput", // BPMN process variable -> routine output
    "metadata": "routineMetadata"           // BPMN process variable -> routine output
  }
}
```

**I/O Mapping Rules:**

1. **fromContext Patterns:**
   - `root.inputName` - References routine-level input
   - `activityId.outputName` - References output from another activity
   - No fromContext - Direct mapping from activity input

2. **toRootContext Pattern:**
   - `toRootContext="outputName"` - Exports to routine-level output
   - No toRootContext - Keeps output in activity scope

**Common BPMN Patterns:**

**Sequential Flow:**
```xml
<bpmn:startEvent id="StartEvent_1"/>
<bpmn:callActivity id="step1" calledElement="first"/>
<bpmn:callActivity id="step2" calledElement="second"/>
<bpmn:endEvent id="EndEvent_1"/>

<bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="step1"/>
<bpmn:sequenceFlow id="Flow_2" sourceRef="step1" targetRef="step2"/>
<bpmn:sequenceFlow id="Flow_3" sourceRef="step2" targetRef="EndEvent_1"/>
```

**Parallel Gateway (Fork/Join):**
```xml
<bpmn:parallelGateway id="fork" gatewayDirection="Diverging"/>
<bpmn:callActivity id="parallelTask1" calledElement="task1"/>
<bpmn:callActivity id="parallelTask2" calledElement="task2"/>
<bpmn:parallelGateway id="join" gatewayDirection="Converging"/>

<!-- Fork flows -->
<bpmn:sequenceFlow sourceRef="fork" targetRef="parallelTask1"/>
<bpmn:sequenceFlow sourceRef="fork" targetRef="parallelTask2"/>

<!-- Join flows -->
<bpmn:sequenceFlow sourceRef="parallelTask1" targetRef="join"/>
<bpmn:sequenceFlow sourceRef="parallelTask2" targetRef="join"/>
```

**Exclusive Gateway (Decision):**
```xml
<bpmn:exclusiveGateway id="decision" gatewayDirection="Diverging"/>
<bpmn:callActivity id="optionA" calledElement="pathA"/>
<bpmn:callActivity id="optionB" calledElement="pathB"/>

<bpmn:sequenceFlow sourceRef="decision" targetRef="optionA">
  <bpmn:conditionExpression>condition == "A"</bpmn:conditionExpression>
</bpmn:sequenceFlow>
<bpmn:sequenceFlow sourceRef="decision" targetRef="optionB">
  <bpmn:conditionExpression>condition == "B"</bpmn:conditionExpression>
</bpmn:sequenceFlow>
```

**Complete BPMN Example:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://bpmn.io/schema/bpmn" 
                  xmlns:vrooli="[APP_URL]/bpmn"
                  id="Definitions_Research"
                  targetNamespace="http://vrooli.com/research">
  
  <bpmn:process id="researchProcess" isExecutable="true">
    
    <bpmn:startEvent id="startResearch"/>
    
    <bpmn:callActivity id="gatherSources" name="Gather Sources" calledElement="webSearch">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="query" fromContext="root.researchTopic"/>
          <vrooli:input name="maxResults" fromContext="root.sourceLimit"/>
          <vrooli:output name="sources"/>
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_start</bpmn:incoming>
      <bpmn:outgoing>Flow_analyze</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:callActivity id="analyzeSources" name="Analyze Sources" calledElement="contentAnalysis">
      <bpmn:extensionElements>
        <vrooli:ioMapping>
          <vrooli:input name="sources" fromContext="gatherSources.sources"/>
          <vrooli:output name="insights"/>
          <vrooli:output name="summary" toRootContext="researchSummary"/>
        </vrooli:ioMapping>
      </bpmn:extensionElements>
      <bpmn:incoming>Flow_analyze</bpmn:incoming>
      <bpmn:outgoing>Flow_end</bpmn:outgoing>
    </bpmn:callActivity>
    
    <bpmn:endEvent id="endResearch"/>
    
    <bpmn:sequenceFlow id="Flow_start" sourceRef="startResearch" targetRef="gatherSources"/>
    <bpmn:sequenceFlow id="Flow_analyze" sourceRef="gatherSources" targetRef="analyzeSources"/>
    <bpmn:sequenceFlow id="Flow_end" sourceRef="analyzeSources" targetRef="endResearch"/>
    
  </bpmn:process>
</bpmn:definitions>
```

**BPMN Generation Guidelines:**

1. **Use meaningful IDs**: Call activity IDs should be descriptive and match activityMap keys
2. **Plan data flow**: Map inputs/outputs carefully using fromContext and toRootContext
3. **Include proper sequence flows**: Every element needs incoming/outgoing flows
4. **Use appropriate gateways**: Parallel for concurrent tasks, Exclusive for decisions
5. **Keep XML valid**: Ensure proper BPMN 2.0 structure and required attributes

## Available Subroutine Types

When designing routine steps, you can call these types of subroutines:

### 1. **Action Subroutines** (MCP Tools)
- **Purpose**: Execute specific Vrooli platform actions
- **Examples**: Create resources, manage data, send notifications, update user settings
- **Use for**: Platform integrations, data management, user interactions

### 2. **API Subroutines**
- **Purpose**: Call external REST APIs
- **Examples**: Fetch external data, integrate with third-party services, webhook calls
- **Use for**: External service integrations, data retrieval, system interconnectivity

### 3. **Code Subroutines**
- **Purpose**: Execute sandboxed code for data transformation
- **Examples**: Data processing, calculations, format conversions, algorithmic operations
- **Use for**: Complex computations, data transformations, custom logic

### 4. **Generate Subroutines** (LLM Calls)
- **Purpose**: Generate content using language models
- **Examples**: Text generation, analysis, reasoning, conversation, creative writing
- **Use for**: AI-powered content creation, analysis, reasoning tasks

### 5. **Smart Contract Subroutines**
- **Purpose**: Interact with blockchain smart contracts
- **Examples**: Token transfers, contract calls, blockchain data retrieval
- **Use for**: DeFi operations, blockchain integrations, decentralized applications

### 6. **Web Search Subroutines**
- **Purpose**: Perform web searches and extract information
- **Examples**: Research queries, real-time information gathering, fact checking
- **Use for**: Information retrieval, research tasks, current event analysis

### 7. **Multi-Step Subroutines** (Other Routines)
- **Purpose**: Call other routines as reusable components
- **Examples**: Common workflow patterns, shared business logic, modular processes
- **Use for**: Hierarchical workflows, code reuse, complex process composition

## Execution Strategies

### Conversational Strategy
- **Purpose**: Natural language interaction and dialogue management
- **Characteristics**: Flexible, adaptive, context-aware responses
- **Use cases**: Customer service, educational tutoring, interactive assistants
- **Subroutine types**: Primarily Generate subroutines with dynamic prompt generation

### Reasoning Strategy  
- **Purpose**: Structured analysis and logical problem-solving
- **Characteristics**: Step-by-step reasoning, evidence gathering, systematic evaluation
- **Use cases**: Research analysis, decision support, complex problem solving
- **Subroutine types**: Mix of Generate, Web Search, and Code subroutines for analysis

### Deterministic Strategy
- **Purpose**: Reliable, predictable automation
- **Characteristics**: Consistent outputs, minimal variation, high reliability
- **Use cases**: Data processing, system maintenance, routine automation
- **Subroutine types**: Primarily Action, API, and Code subroutines with minimal LLM usage

## Routine Categories

### 1. **Metareasoning Routines**
- **Purpose**: Reasoning about reasoning, strategy selection, process optimization
- **Examples**: 
  - Strategy selection routines that choose optimal execution approaches
  - Performance analysis routines that evaluate and improve workflow efficiency
  - Self-reflection routines that assess and adapt AI behavior
- **Typical structure**: Sequential with Generate subroutines for analysis

### 2. **Productivity & Task Management**
- **Purpose**: Personal and professional productivity enhancement
- **Examples**:
  - Project planning and breakdown routines
  - Calendar management and scheduling optimization
  - Task prioritization and time management systems
- **Typical structure**: Sequential or BPMN depending on complexity

### 3. **Summarization & Knowledge Management**
- **Purpose**: Information processing, organization, and retrieval
- **Examples**:
  - Document summarization and key insight extraction
  - Knowledge base creation and maintenance
  - Information synthesis from multiple sources
- **Typical structure**: Sequential with Generate and Code subroutines

### 4. **Research & Information Gathering**
- **Purpose**: Systematic information collection and analysis
- **Examples**:
  - Market research and competitive analysis
  - Academic research and literature reviews
  - Fact-checking and verification processes
- **Typical structure**: Sequential with Web Search and Generate subroutines

### 5. **Content Creation & Communication**
- **Purpose**: Generate and optimize various forms of content
- **Examples**:
  - Technical documentation generation
  - Marketing content creation workflows
  - Communication optimization routines
- **Typical structure**: Sequential with Generate subroutines

### 6. **Data Processing & Analysis**
- **Purpose**: Transform, analyze, and extract insights from data
- **Examples**:
  - ETL (Extract, Transform, Load) workflows
  - Statistical analysis and reporting
  - Data quality assessment and cleaning
- **Typical structure**: Sequential with Code and Generate subroutines

### 7. **Integration & Automation**
- **Purpose**: Connect systems and automate business processes
- **Examples**:
  - Multi-system data synchronization
  - Automated workflow triggers and responses
  - Cross-platform integration routines
- **Typical structure**: BPMN for complex integrations, Sequential for simple ones

### 8. **Learning & Adaptation**
- **Purpose**: Continuous improvement and knowledge acquisition
- **Examples**:
  - Skill development tracking and optimization
  - Feedback collection and analysis
  - Adaptive system configuration
- **Typical structure**: Sequential with Generate and Action subroutines

## Database Structure Requirements

To create a valid routine that can be stored and executed, you must generate a structure that matches Vrooli's database schema. **IMPORTANT**: The actual database stores data in normalized tables, not nested JSON. The structure below shows the logical relationships:

### Complete Routine Structure (Logical Format for Import)

```json
{
  "id": "[generate-snowflake-id]",
  "publicId": "[generate-10-12-char-alphanumeric]",
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,  // Only true for system routines
  "tags": [
    {
      "id": "[generate-snowflake-id]",
      "tag": "YourTagHere"
    }
  ],
  "versions": [
    {
      "id": "[generate-snowflake-id]",
      "publicId": "[generate-10-12-char-alphanumeric]",
      "versionLabel": "1.0.0",
      "versionNotes": "Initial version",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,  // true if can be run autonomously
      "resourceSubType": "[RoutineMultiStep|RoutineGenerate|RoutineApi|etc]",
      "config": {
        // RoutineVersionConfig object - see below
      },
      "translations": [
        {
          "id": "[generate-snowflake-id]",
          "language": "en",
          "name": "Your Routine Name",
          "description": "What this routine does (2-4 sentences)",
          "instructions": "How to use this routine",
          "details": "Extended documentation (optional)"
        }
      ]
    }
  ]
}
```

### Config Structure by Routine Type

#### For Multi-Step Routines (Sequential)

```json
{
  "__version": "1.0",
  "graph": {
    "__version": "1.0",
    "__type": "Sequential",
    "schema": {
      "steps": [
        {
          "id": "step1",
          "name": "Step Name",
          "description": "What this step does",
          "subroutineId": "[existing-routine-publicId-or-id]",
          "inputMap": {
            "stepInputName": "subroutineFormInputFieldName"
          },
          "outputMap": {
            "subroutineFormOutputFieldName": "stepOutputName"
          },
          "skipCondition": "optional-condition-expression",
          "retryPolicy": {
            "maxAttempts": 3,
            "backoffMs": 1000
          }
        }
      ],
      "rootContext": {
        "inputMap": {
          "routineFormInputFieldName": "firstStepInputName"
        },
        "outputMap": {
          "lastStepOutputName": "routineFormOutputFieldName"
        }
      },
      "executionMode": "sequential"  // or "parallel" for entire routine
    }
  },
  "executionStrategy": "reasoning",  // "conversational", "deterministic", or "auto"
  "allowStrategyOverride": true,
  "subroutineStrategies": {
    "step1": "deterministic"  // Optional per-step overrides (use step.id as key)
  },
  "formInput": {
    "__version": "1.0",
    "schema": {
      "layout": {
        "title": "Input Form Title",
        "description": "Form description"
      },
      "containers": [],
      "elements": [
        {
          "id": "input1",
          "fieldName": "routineFormInputFieldName",
          "label": "Input Label",
          "type": "Text",
          "isRequired": true,
          "props": {}
        }
      ]
    }
  },
  "formOutput": {
    "__version": "1.0",
    "schema": {
      "containers": [],
      "elements": [
        {
          "id": "output1",
          "fieldName": "routineFormOutputFieldName",
          "label": "Output Label",
          "type": "Text",
          "props": {}
        }
      ]
    }
  }
}
```

#### For Multi-Step Routines (BPMN)

```json
{
  "__version": "1.0",
  "graph": {
    "__version": "1.0",
    "__type": "BPMN-2.0",
    "schema": {
      "__format": "xml",
      "data": "[BPMN XML String - see BPMN section for format]",
      "activityMap": {
        "callActivityId": {
          "subroutineId": "[existing-routine-id]",
          "inputMap": {
            "activityInput": "subroutineInput"
          },
          "outputMap": {
            "subroutineOutput": "activityOutput"
          }
        }
      },
      "rootContext": {
        "inputMap": {
          "routineInput": "processInput"
        },
        "outputMap": {
          "processOutput": "routineOutput"
        }
      }
    }
  }
}
```

### Critical Validation Rules (MUST FOLLOW)

1. **ID Generation**:
   - Use snowflake IDs (19-digit strings) for all `id` fields
   - Use 10-12 character alphanumeric strings for `publicId` fields
   - All IDs must be unique within the routine
   - Format: IDs should be numeric strings like "1234567890123456789"
   - Format: publicIds should be lowercase alphanumeric like "yes-man-av-v1"

2. **Required Fields**:
   - Resource: `id`, `publicId`, `resourceType` (must be "Routine"), `versions` (at least one)
   - ResourceVersion: `id`, `publicId`, `versionLabel`, `isComplete`, `resourceSubType`, `config`, `translations` (at least one)
   - Translation: `id`, `language`, `name` (description and instructions highly recommended)

3. **Subroutine References (CRITICAL)**:
   - All `subroutineId` values MUST reference routines that exist in the database
   - Use publicIds or database IDs of actual routines
   - For testing/examples, use these placeholder IDs that represent common routine types:
     - "analyze-input-01" (Text analysis routine)
     - "generate-text-02" (LLM generation routine)
     - "web-search-03" (Web search routine)
     - "data-transform-04" (Data transformation routine)
     - "api-call-05" (External API routine)
   - In production, these must be replaced with actual routine IDs from the database

4. **Resource Subtypes**:
   - `RoutineMultiStep`: For workflows calling other routines (requires `graph` config)
   - `RoutineGenerate`: For LLM-based generation (requires `callDataGenerate` config)
   - `RoutineApi`: For external API calls (requires `callDataApi` config)
   - `RoutineCode`: For code execution (requires `callDataCode` config)
   - `RoutineInternalAction`: For Vrooli platform actions (requires `callDataAction` config)
   - `RoutineWeb`: For web searches (requires `callDataWeb` config)
   - `RoutineSmartContract`: For blockchain interactions (requires `callDataSmartContract` config)
   - `RoutineInformational`: For documentation only (no execution config needed)

5. **Config Structure Rules**:
   - Config MUST have `__version: "1.0"` at root level
   - RoutineMultiStep MUST have `graph` property with Sequential or BPMN-2.0 type
   - Multi-step routines MUST have both `formInput` and `formOutput` configs
   - Single-step routines need their specific call data config (e.g., `callDataGenerate`)
   - `executionStrategy`, `allowStrategyOverride`, and `subroutineStrategies` go at config root, NOT inside graph

6. **I/O Mapping Rules for Multi-Step (CRITICAL FOR RUNTIME SUCCESS)**:
   - Step `inputMap`: Maps step's input names to subroutine's formInput field names
   - Step `outputMap`: Maps subroutine's formOutput field names to step's output names
   - Root `inputMap`: Maps routine's formInput field names to first step's input names
   - Root `outputMap`: Maps last step's output names to routine's formOutput field names
   - Field names must match exactly between mappings and form schemas
   - **CRITICAL**: Every step input MUST reference either a routine input (via rootContext) or an output from a previous step
   - **CRITICAL**: Use unique variable names to avoid collisions (e.g., "originalUserInput" not "userInput")
   - **CRITICAL**: Every variable referenced in inputMap MUST be available from either rootContext.inputMap or previous step outputMaps

7. **Execution Mode**:
   - `executionMode` is set at the schema level for the entire routine, not per-step
   - Use "sequential" for step-by-step execution
   - Use "parallel" when all steps can run independently

8. **Strategy Configuration Rules (NEW)**:
   - For Sequential routines: `subroutineStrategies` keys MUST be step indices ("0", "1", "2") NOT step.id values
   - For BPMN routines: `subroutineStrategies` keys should be call activity IDs from the activityMap
   - Valid strategies: "reasoning", "deterministic", "conversational"

## Output Requirements

Generate a complete routine structure that:

1. **Is immediately executable**: All referenced subroutines must exist in the database
2. **Has complete metadata**: All required fields filled with valid values
3. **Follows naming conventions**: Use descriptive, clear names for all elements
4. **Includes proper documentation**: Name, description, and instructions in translations
5. **Maps data correctly**: Input/output mappings must align between steps

## Data Flow Planning and Variable Management

### Variable Naming Strategy
To prevent runtime errors, follow these variable naming conventions:

1. **Use Descriptive Prefixes**:
   - Original inputs: `original*` (e.g., `originalUserInput`, `originalContext`)
   - Processed data: `processed*` (e.g., `processedAnalysis`, `processedResults`)
   - Generated content: `generated*` (e.g., `generatedResponse`, `generatedSummary`)
   - External data: `external*` (e.g., `externalSources`, `externalData`)

2. **Avoid Generic Names**:
   - ❌ Don't use: `input`, `output`, `data`, `result`
   - ✅ Use instead: `userMessage`, `analysisResult`, `searchFindings`, `finalResponse`

3. **Step-Specific Naming**:
   - Include step context: `step1Analysis`, `step2SearchResults`, `step3Evaluation`
   - Or use functional names: `pressureDetection`, `evidenceGathering`, `responseGeneration`

### Data Flow Validation Checklist

Before finalizing a routine, trace the complete data flow:

1. **Trace Each Variable**:
   ```
   Routine Input → rootContext.inputMap → Step Input → Subroutine
   Subroutine → Step Output → Next Step Input → Next Subroutine
   Final Step Output → rootContext.outputMap → Routine Output
   ```

2. **Verify Variable Availability**:
   - Every step's `inputMap` value must exist in either:
     - `rootContext.inputMap` (for routine inputs)
     - Previous step's `outputMap` (for step outputs)
   
3. **Check Output Capture**:
   - All important step outputs should flow to final routine outputs
   - Don't lose valuable intermediate results

### Example Data Flow Planning

```typescript
// Step 1: Define all variables that will flow through the routine
const variableFlow = {
  // Routine inputs (from formInput)
  routineInputs: ["userMessage", "conversationContext", "topic"],
  
  // Step variables (intermediate values)
  stepVariables: {
    step0: {
      inputs: ["originalUserMessage", "originalContext"], // from routine
      outputs: ["pressureAnalysis", "riskLevel"]
    },
    step1: {
      inputs: ["searchTopic", "pressureAnalysis"], // topic from routine, analysis from step0
      outputs: ["searchResults", "counterEvidence"]
    },
    step2: {
      inputs: ["originalUserMessage", "counterEvidence", "pressureAnalysis"], // mixed sources
      outputs: ["evidenceEvaluation", "logicalFallacies"]
    }
  },
  
  // Routine outputs (to formOutput)
  routineOutputs: ["finalResponse", "analysisReasoning", "qualityMetrics"]
};

// Step 2: Build rootContext mappings
const rootContext = {
  inputMap: {
    // Map formInput fields to step variables
    "userMessage": "originalUserMessage",      // Unique name to avoid collisions
    "conversationContext": "originalContext",
    "topic": "searchTopic"
  },
  outputMap: {
    // Map final step variables to formOutput fields
    "finalResponse": "routineFinalResponse",
    "analysisReasoning": "routineReasoning",
    "qualityMetrics": "routineQuality"
  }
};

// Step 3: Build each step's mappings ensuring variable availability
const steps = [
  {
    id: "analyzePressure",
    inputMap: {
      // These variables MUST exist in rootContext.inputMap
      "messageText": "originalUserMessage",  // ✅ Available from routine
      "context": "originalContext"          // ✅ Available from routine
    },
    outputMap: {
      // These create new variables for subsequent steps
      "analysis": "pressureAnalysis",      // ✅ Creates pressureAnalysis
      "riskLevel": "riskLevel"            // ✅ Creates riskLevel
    }
  },
  {
    id: "searchCounterEvidence", 
    inputMap: {
      // These variables MUST exist from previous steps or routine
      "query": "searchTopic",              // ✅ Available from routine
      "context": "pressureAnalysis"       // ✅ Available from step 0
    },
    outputMap: {
      "results": "searchResults",         // ✅ Creates searchResults
      "evidence": "counterEvidence"       // ✅ Creates counterEvidence
    }
  }
  // Continue for all steps...
];
```

## Common Pitfalls to Avoid

1. **Wrong Config Nesting**: Don't put `executionStrategy` inside `graph`. It goes at the config root.
2. **Missing formOutput**: Multi-step routines MUST define their outputs in `formOutput`.
3. **Invalid Subroutine IDs**: Every `subroutineId` must exist in the database.
4. **Mismatched Field Names**: Input/output mappings must use exact field names from form schemas.
5. **Wrong ID Formats**: Use 19-digit strings for IDs, not random numbers.
6. **Missing __version**: Every config object needs `__version: "1.0"`.
7. **Broken Variable References**: Every inputMap value must reference an available variable.
8. **Variable Name Collisions**: Using generic names like "input" that get overwritten.
9. **Wrong Strategy Keys**: Using step.id instead of step index for Sequential routines.
10. **Incomplete Output Capture**: Not mapping all valuable step outputs to routine outputs.

## Complete Example

Here's a valid routine that could be imported and run immediately:

```json
{
  "id": "7234567890123456789",
  "publicId": "yes-man-avoid",
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,
  "tags": [
    {
      "id": "7234567890123456790",
      "tag": "Critical Thinking"
    }
  ],
  "versions": [
    {
      "id": "7234567890123456791",
      "publicId": "yma-v1",
      "versionLabel": "1.0.0",
      "versionNotes": "Initial release of yes-man avoidance routine",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,
      "resourceSubType": "RoutineMultiStep",
      "config": {
        "__version": "1.0",
        "graph": {
          "__version": "1.0",
          "__type": "Sequential",
          "schema": {
            "steps": [
              {
                "id": "detectPressure",
                "name": "Detect Agreement Pressure",
                "description": "Analyze if the user is pressuring for agreement",
                "subroutineId": "analyze-input-01",
                "inputMap": {
                  "messageInput": "originalUserMessage",  // Use unique variable name
                  "contextInput": "originalContext"       // Use unique variable name
                },
                "outputMap": {
                  "analysis": "pressureAnalysisResult"    // Use descriptive variable name
                }
              },
              {
                "id": "generateResponse",
                "name": "Generate Balanced Response",
                "description": "Create a thoughtful, balanced response",
                "subroutineId": "generate-text-02",
                "inputMap": {
                  "analysisInput": "pressureAnalysisResult",  // Reference previous step's output
                  "originalInput": "originalUserMessage"      // Reference routine input
                },
                "outputMap": {
                  "response": "generatedBalancedResponse"     // Use descriptive variable name
                }
              }
            ],
            "rootContext": {
              "inputMap": {
                "userMessage": "originalUserMessage",     // Map to unique variable names
                "conversationContext": "originalContext"  // Map to unique variable names
              },
              "outputMap": {
                "generatedBalancedResponse": "finalResponse"  // Map from final step output
              }
            },
            "executionMode": "sequential"
          }
        },
        "executionStrategy": "reasoning",
        "allowStrategyOverride": true,
        "subroutineStrategies": {
          "0": "reasoning",     // Step index for detectPressure step
          "1": "reasoning"      // Step index for generateResponse step  
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "layout": {
              "title": "Yes-Man Avoidance Input",
              "description": "Provide the message to analyze"
            },
            "containers": [],
            "elements": [
              {
                "id": "msg",
                "fieldName": "userMessage",
                "label": "User Message",
                "type": "Text",
                "isRequired": true,
                "props": {
                  "multiline": true,
                  "minRows": 3
                }
              },
              {
                "id": "ctx",
                "fieldName": "conversationContext",
                "label": "Conversation Context",
                "type": "Text",
                "isRequired": false,
                "props": {
                  "multiline": true,
                  "minRows": 5
                }
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0",
          "schema": {
            "containers": [],
            "elements": [
              {
                "id": "response",
                "fieldName": "finalResponse",
                "label": "Balanced Response",
                "type": "Text",
                "props": {
                  "multiline": true
                }
              }
            ]
          }
        }
      },
      "translations": [
        {
          "id": "7234567890123456792",
          "language": "en",
          "name": "Yes-Man Avoidance Critical Thinking Routine",
          "description": "Prevents AI systems from blindly agreeing with users by enforcing critical evaluation, fact-checking, and balanced perspective presentation. Essential for maintaining trustworthy AI interactions.",
          "instructions": "Provide the user's message and optionally the conversation context. The routine will analyze for agreement pressure and generate a balanced, thoughtful response."
        }
      ]
    }
  ]
}
```

### Key Points About The Example:

1. **Valid IDs**: Uses 19-digit snowflake IDs and lowercase alphanumeric publicIds
2. **Placeholder Subroutines**: Uses placeholder IDs that represent common routine types
3. **Complete Config**: All required fields at correct nesting levels
4. **Form I/O**: Both `formInput` and `formOutput` defined with matching field names
5. **Proper Mappings**: Field names flow correctly through all mappings:
   - `userMessage` (formInput) → `originalUserMessage` (step variable) → `messageInput` (subroutine input)
   - `response` (subroutine output) → `generatedBalancedResponse` (step variable) → `finalResponse` (formOutput)
6. **Unique Variable Names**: Uses descriptive, collision-resistant variable names
7. **Valid Strategy Keys**: Uses step indices ("0", "1") not step IDs for subroutineStrategies
8. **Complete Data Flow**: Every step input references an available variable
9. **Translations**: Complete with name, description, and instructions

## Generation Guidelines

1. **Choose the right structure**: Use Sequential for most routines. Only use BPMN when you need complex branching, parallel gateways, or event handling.

2. **Select appropriate strategy**: 
   - Conversational for user-facing, adaptive tasks
   - Reasoning for analysis and problem-solving
   - Deterministic for automation and data processing

3. **Design meaningful steps**: Each step should have a clear, single responsibility. Use descriptive names and include helpful descriptions.

4. **Consider input/output flow**: Ensure data flows logically between steps. Map inputs and outputs carefully to maintain data continuity.

5. **Plan for error handling**: Include retry policies where appropriate, especially for external API calls or network operations.

6. **Think hierarchically**: Consider whether your routine could benefit from calling other routines as subroutines for modularity.

7. **Optimize for reusability**: Design routines that can be parameterized and reused across different contexts.

## Snowflake ID Generation

When generating IDs, follow these patterns:
- **Snowflake IDs**: 19-digit strings like "7234567890123456789"
- **Public IDs**: 10-12 character alphanumeric strings like "yes-man-avoid" or "yma-v1"
- Ensure all IDs are unique within the routine

## Final Checklist

Before outputting a routine, verify:

### Basic Structure:
- [ ] All `id` fields have valid snowflake IDs (19-digit numeric strings)
- [ ] All `publicId` fields have 10-12 character lowercase alphanumeric values
- [ ] `resourceType` is set to "Routine" (not "Resource")
- [ ] `resourceSubType` is specified and matches the config structure
- [ ] All `subroutineId` values use placeholder IDs or actual routine IDs
- [ ] Config has `__version: "1.0"` at root level
- [ ] Multi-step routines have both `formInput` AND `formOutput`
- [ ] `executionStrategy` is at config root, NOT inside `graph`

### Critical Data Flow Validation:
- [ ] **Variable Availability**: Every step's `inputMap` value exists in either `rootContext.inputMap` OR previous step's `outputMap`
- [ ] **Unique Variable Names**: No generic names like "input", "output", "data" that could collide
- [ ] **Complete Flow**: Trace data from formInput → rootContext → steps → rootContext → formOutput
- [ ] **Strategy Keys**: For Sequential routines, use step indices ("0", "1", "2") in `subroutineStrategies`
- [ ] **Output Capture**: All valuable step outputs are captured in final routine outputs

### Input/Output Flow Verification:
- [ ] formInput fields → step variables (via rootContext.inputMap)
- [ ] step variables → subroutine inputs (via step.inputMap)  
- [ ] subroutine outputs → step variables (via step.outputMap)
- [ ] step variables → formOutput fields (via rootContext.outputMap)

### Documentation:
- [ ] At least one translation with `name`, `description`, and `instructions`
- [ ] No nested "shape" objects or "__typename" fields (those are for GraphQL, not database)

### Pre-Generation Planning:
- [ ] Create a variable flow diagram before writing the routine
- [ ] Plan unique, descriptive variable names for each step
- [ ] Verify each step can access its required inputs
- [ ] Ensure final outputs capture all important results

Generate routines that demonstrate the full power and flexibility of Vrooli's execution architecture while solving real-world problems effectively.