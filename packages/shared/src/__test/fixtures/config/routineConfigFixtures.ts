import { type RoutineVersionConfigObject } from "../../../shape/configs/routine.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures } from "./baseConfigFixtures.js";

/**
 * Routine configuration fixtures for testing various routine types
 */
export const routineConfigFixtures: ConfigTestFixtures<RoutineVersionConfigObject> & {
    // Additional categorized fixtures for different routine types
    action: {
        simple: RoutineVersionConfigObject;
        withInputMapping: RoutineVersionConfigObject;
        withOutputMapping: RoutineVersionConfigObject;
        withMachine: RoutineVersionConfigObject;
    };
    generate: {
        basic: RoutineVersionConfigObject;
        withCustomModel: RoutineVersionConfigObject;
        withComplexPrompt: RoutineVersionConfigObject;
    };
    multiStep: {
        sequential: RoutineVersionConfigObject;
        withBranching: RoutineVersionConfigObject;
        complexWorkflow: RoutineVersionConfigObject;
    };
} = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        routineType: "action",
    },
    
    complete: {
        __version: LATEST_CONFIG_VERSION,
        routineType: "multiStep",
        steps: [
            {
                stepId: "step1",
                type: "action",
                name: "Data Collection",
                description: "Collect input data from user",
                actionConfig: {
                    actionType: "api_call",
                    endpoint: "/api/v1/collect",
                    method: "POST"
                }
            },
            {
                stepId: "step2",
                type: "generate",
                name: "Process Data",
                description: "Process collected data",
                generateConfig: {
                    model: "gpt-4",
                    prompt: "Process the following data: {{input}}",
                    temperature: 0.7
                }
            },
            {
                stepId: "step3",
                type: "action",
                name: "Save Results",
                description: "Save processed results",
                actionConfig: {
                    actionType: "api_call",
                    endpoint: "/api/v1/save",
                    method: "POST"
                }
            }
        ],
        resources: [{
            link: "https://example.com/routine-guide",
            usedFor: "Tutorial",
            translations: [{
                language: "en",
                name: "Routine Guide",
                description: "How to use this routine"
            }]
        }]
    },
    
    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        routineType: "action",
        resources: []
    },
    
    invalid: {
        missingVersion: {
            routineType: "action",
        },
        invalidVersion: {
            __version: "0.1",
            routineType: "action",
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "invalid_type", // Invalid routine type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "multiStep",
            steps: "not an array" // Should be array
        }
    },
    
    variants: {
        simpleApiCall: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "api_call",
                endpoint: "/api/v1/users",
                method: "GET"
            }
        },
        
        textGeneration: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "generate",
            generateConfig: {
                model: "gpt-4",
                prompt: "Generate a summary of: {{text}}",
                temperature: 0.5,
                maxTokens: 500
            }
        },
        
        dataTransformation: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "transform",
                transformationType: "json",
                inputSchema: {
                    type: "object",
                    properties: {
                        name: { type: "string" },
                        age: { type: "number" }
                    }
                },
                outputSchema: {
                    type: "object",
                    properties: {
                        fullName: { type: "string" },
                        isAdult: { type: "boolean" }
                    }
                }
            }
        }
    },
    
    // Action routine fixtures
    action: {
        simple: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "api_call",
                endpoint: "/api/v1/data",
                method: "GET"
            }
        },
        
        withInputMapping: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "api_call",
                endpoint: "/api/v1/process",
                method: "POST",
                inputMapping: {
                    "userInput": "$.data.input",
                    "userId": "$.context.userId",
                    "timestamp": "$.context.timestamp"
                }
            }
        },
        
        withOutputMapping: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "api_call",
                endpoint: "/api/v1/analyze",
                method: "POST",
                outputMapping: {
                    "result": "$.response.analysis",
                    "confidence": "$.response.metadata.confidence",
                    "processedAt": "$.response.timestamp"
                }
            }
        },
        
        withMachine: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "action",
            actionConfig: {
                actionType: "machine",
                machineId: "machine_123456789",
                operation: "process",
                parameters: {
                    mode: "advanced",
                    timeout: 30000
                }
            }
        }
    },
    
    // Generate routine fixtures
    generate: {
        basic: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "generate",
            generateConfig: {
                model: "gpt-3.5-turbo",
                prompt: "Answer the following question: {{question}}"
            }
        },
        
        withCustomModel: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "generate",
            generateConfig: {
                model: "claude-3-opus",
                prompt: "Analyze the following code and suggest improvements: {{code}}",
                temperature: 0.3,
                maxTokens: 2000,
                systemPrompt: "You are an expert code reviewer."
            }
        },
        
        withComplexPrompt: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "generate",
            generateConfig: {
                model: "gpt-4",
                prompt: `Based on the following context:
                User Profile: {{userProfile}}
                Previous Interactions: {{history}}
                Current Request: {{request}}
                
                Generate a personalized response that:
                1. Addresses the user's specific needs
                2. Takes into account their history
                3. Provides actionable next steps`,
                temperature: 0.7,
                maxTokens: 1000,
                stopSequences: ["END", "DONE"],
                frequencyPenalty: 0.5,
                presencePenalty: 0.5
            }
        }
    },
    
    // Multi-step routine fixtures
    multiStep: {
        sequential: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "multiStep",
            steps: [
                {
                    stepId: "fetch",
                    type: "action",
                    name: "Fetch Data",
                    actionConfig: {
                        actionType: "api_call",
                        endpoint: "/api/v1/data",
                        method: "GET"
                    }
                },
                {
                    stepId: "process",
                    type: "generate",
                    name: "Process Data",
                    generateConfig: {
                        model: "gpt-3.5-turbo",
                        prompt: "Summarize this data: {{fetch.output}}"
                    }
                },
                {
                    stepId: "save",
                    type: "action",
                    name: "Save Summary",
                    actionConfig: {
                        actionType: "api_call",
                        endpoint: "/api/v1/summaries",
                        method: "POST",
                        inputMapping: {
                            "summary": "{{process.output}}"
                        }
                    }
                }
            ]
        },
        
        withBranching: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "multiStep",
            steps: [
                {
                    stepId: "checkCondition",
                    type: "action",
                    name: "Check Condition",
                    actionConfig: {
                        actionType: "evaluate",
                        expression: "input.value > 100"
                    }
                },
                {
                    stepId: "highValuePath",
                    type: "action",
                    name: "Process High Value",
                    condition: "{{checkCondition.output}} === true",
                    actionConfig: {
                        actionType: "api_call",
                        endpoint: "/api/v1/high-value",
                        method: "POST"
                    }
                },
                {
                    stepId: "lowValuePath",
                    type: "action",
                    name: "Process Low Value",
                    condition: "{{checkCondition.output}} === false",
                    actionConfig: {
                        actionType: "api_call",
                        endpoint: "/api/v1/low-value",
                        method: "POST"
                    }
                }
            ]
        },
        
        complexWorkflow: {
            __version: LATEST_CONFIG_VERSION,
            routineType: "multiStep",
            steps: [
                {
                    stepId: "init",
                    type: "action",
                    name: "Initialize",
                    description: "Set up workflow context",
                    actionConfig: {
                        actionType: "initialize",
                        defaultValues: {
                            status: "pending",
                            attempts: 0,
                            results: []
                        }
                    }
                },
                {
                    stepId: "fetchSources",
                    type: "action",
                    name: "Fetch Data Sources",
                    parallel: true,
                    actionConfig: {
                        actionType: "api_call",
                        endpoint: "/api/v1/sources",
                        method: "GET"
                    }
                },
                {
                    stepId: "analyzeEach",
                    type: "generate",
                    name: "Analyze Each Source",
                    forEach: "{{fetchSources.output}}",
                    generateConfig: {
                        model: "gpt-4",
                        prompt: "Analyze this data source: {{item}}",
                        temperature: 0.5
                    }
                },
                {
                    stepId: "aggregate",
                    type: "action",
                    name: "Aggregate Results",
                    actionConfig: {
                        actionType: "aggregate",
                        aggregationType: "combine",
                        inputMapping: {
                            "analyses": "{{analyzeEach.outputs}}"
                        }
                    }
                },
                {
                    stepId: "generateReport",
                    type: "generate",
                    name: "Generate Final Report",
                    generateConfig: {
                        model: "gpt-4",
                        prompt: "Create a comprehensive report from these analyses: {{aggregate.output}}",
                        temperature: 0.3,
                        maxTokens: 3000
                    }
                },
                {
                    stepId: "notify",
                    type: "action",
                    name: "Send Notifications",
                    parallel: true,
                    actionConfig: {
                        actionType: "notify",
                        channels: ["email", "slack", "webhook"],
                        message: "Report completed: {{generateReport.output.summary}}"
                    }
                }
            ],
            errorHandling: {
                retryAttempts: 3,
                retryDelay: 1000,
                fallbackStep: "errorHandler"
            },
            timeout: 300000 // 5 minutes
        }
    }
};

/**
 * Create a simple action routine config
 */
export function createActionRoutineConfig(
    endpoint: string,
    method: string = "GET"
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        routineType: "action",
        actionConfig: {
            actionType: "api_call",
            endpoint,
            method
        }
    };
}

/**
 * Create a generate routine config
 */
export function createGenerateRoutineConfig(
    prompt: string,
    model: string = "gpt-3.5-turbo"
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        routineType: "generate",
        generateConfig: {
            model,
            prompt
        }
    };
}

/**
 * Create a multi-step routine config
 */
export function createMultiStepRoutineConfig(
    steps: RoutineVersionConfigObject["steps"]
): RoutineVersionConfigObject {
    return {
        __version: LATEST_CONFIG_VERSION,
        routineType: "multiStep",
        steps
    };
}