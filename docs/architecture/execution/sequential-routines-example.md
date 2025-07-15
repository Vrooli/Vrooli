# Sequential Routines Example

This document demonstrates how to use the new sequential (array-based) routine configuration for simple multi-step workflows.

## Overview

Sequential routines provide a simpler alternative to BPMN for straightforward linear processes. They execute steps in order (or optionally in parallel) without the complexity of gateways and flow control.

## Example: Data Processing Pipeline

Here's an example of a sequential routine that processes customer data through multiple steps:

```typescript
const dataProcessingRoutine: RoutineVersionConfigObject = {
    __version: "1.0",
    
    // Sequential graph configuration
    graph: {
        __version: "1.0",
        __type: "Sequential",
        schema: {
            // Array of steps to execute in order
            steps: [
                {
                    id: "validate-input",
                    name: "Validate Input Data",
                    description: "Ensures input data meets requirements",
                    subroutineId: "validator-routine-v1",
                    inputMap: {
                        "data": "validatorInput",
                        "schema": "validationSchema"
                    },
                    outputMap: {
                        "isValid": "validationResult",
                        "errors": "validationErrors"
                    }
                },
                {
                    id: "transform-data",
                    name: "Transform Data",
                    description: "Normalizes and enriches the data",
                    subroutineId: "transformer-routine-v1",
                    inputMap: {
                        "rawData": "transformerInput",
                        "config": "transformConfig"
                    },
                    outputMap: {
                        "transformedData": "processedData"
                    },
                    // Skip this step if validation failed
                    skipCondition: "!context.validationResult",
                    retryPolicy: {
                        maxAttempts: 3,
                        backoffMs: 2000
                    }
                },
                {
                    id: "store-results",
                    name: "Store Results",
                    description: "Saves processed data to database",
                    subroutineId: "storage-routine-v1",
                    inputMap: {
                        "data": "storageInput",
                        "metadata": "storageMetadata"
                    },
                    outputMap: {
                        "recordId": "savedRecordId",
                        "timestamp": "saveTimestamp"
                    }
                }
            ],
            
            // Maps routine inputs/outputs to first/last step
            rootContext: {
                inputMap: {
                    // Routine inputs mapped to first step
                    "customerData": "data",
                    "validationRules": "schema"
                },
                outputMap: {
                    // Last step outputs mapped to routine outputs
                    "recordId": "processedRecordId",
                    "timestamp": "completionTime"
                }
            },
            
            // Optional: execute all steps in parallel
            executionMode: "sequential" // or "parallel"
        }
    },
    
    // Execution strategy for AI-powered execution
    executionStrategy: "deterministic", // Since this is a data pipeline
    
    // Form configuration for inputs
    formInput: {
        __version: "1.0",
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "customerData",
                    id: "customer-data-input",
                    label: "Customer Data (JSON)",
                    type: InputType.JSON,
                    validation: { required: true }
                },
                {
                    fieldName: "validationRules",
                    id: "validation-rules-input",
                    label: "Validation Schema",
                    type: InputType.JSON,
                    validation: { required: true }
                }
            ]
        }
    },
    
    // Form configuration for outputs
    formOutput: {
        __version: "1.0",
        schema: {
            containers: [],
            elements: [
                {
                    fieldName: "processedRecordId",
                    id: "record-id-output",
                    label: "Processed Record ID",
                    type: InputType.Text
                },
                {
                    fieldName: "completionTime",
                    id: "completion-time-output",
                    label: "Completion Timestamp",
                    type: InputType.Text
                }
            ]
        }
    }
};
```

## Parallel Execution Example

Here's the same routine configured for parallel execution:

```typescript
const parallelProcessingRoutine: RoutineVersionConfigObject = {
    __version: "1.0",
    
    graph: {
        __version: "1.0",
        __type: "Sequential",
        schema: {
            steps: [
                {
                    id: "analyze-sentiment",
                    name: "Analyze Sentiment",
                    subroutineId: "sentiment-analyzer-v1",
                    inputMap: { "text": "sentimentInput" },
                    outputMap: { "sentiment": "sentimentScore" }
                },
                {
                    id: "extract-entities",
                    name: "Extract Entities",
                    subroutineId: "entity-extractor-v1",
                    inputMap: { "text": "entityInput" },
                    outputMap: { "entities": "extractedEntities" }
                },
                {
                    id: "classify-topic",
                    name: "Classify Topic",
                    subroutineId: "topic-classifier-v1",
                    inputMap: { "text": "classifierInput" },
                    outputMap: { "topics": "identifiedTopics" }
                }
            ],
            
            rootContext: {
                inputMap: {
                    "textContent": "text" // All steps use the same input
                },
                outputMap: {
                    "sentimentScore": "sentiment",
                    "extractedEntities": "entities",
                    "identifiedTopics": "topics"
                }
            },
            
            // Execute all steps in parallel since they're independent
            executionMode: "parallel"
        }
    },
    
    // Use reasoning strategy for NLP tasks
    executionStrategy: "reasoning"
};
```

## Navigation Flow

The SequentialNavigator handles these routines as follows:

### Sequential Mode:
1. Start at step index 0
2. Execute step
3. Check skip conditions for next step
4. Move to next step or skip to following step
5. Repeat until all steps complete

### Parallel Mode:
1. Start at virtual step index 0
2. Return all non-skipped steps as next locations
3. Execution engine runs all steps concurrently
4. Wait for all to complete

## Key Benefits

1. **Simplicity**: No need for BPMN XML or complex gateways
2. **Clarity**: Steps are clearly ordered and easy to understand
3. **Flexibility**: Support for skip conditions and retry policies
4. **Performance**: Parallel mode for independent operations
5. **Type Safety**: Full TypeScript support with clear interfaces

## When to Use Sequential vs BPMN

### Use Sequential When:
- Steps follow a linear path
- Limited branching logic (skip conditions only)
- All steps are subroutine calls
- Simplicity is preferred

### Use BPMN When:
- Complex branching and merging required
- Event-driven flows needed
- Multiple parallel paths with synchronization
- Industry-standard notation required

## Integration with Swarm Context

Sequential routines integrate seamlessly with the swarm execution context:

```typescript
// Swarm can pass context to routine
const swarmContext = {
    goal: "Process customer feedback",
    blackboard: {
        customerSegment: "premium",
        processingPriority: "high"
    }
};

// Skip conditions can reference swarm context
{
    skipCondition: "context.customerSegment !== 'premium'"
}
```

This allows routines to adapt their behavior based on the broader swarm goals and shared knowledge.